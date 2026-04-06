package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * PatientContextEnricher - FHIR & Neo4j Enrichment for Unified Pipeline
 *
 * Adapts existing ComprehensiveEnrichmentFunction infrastructure to work with
 * EnrichedPatientContext instead of CanonicalEvent → EnrichedEvent.
 *
 * REUSES:
 * - GoogleFHIRClient (demographics, conditions, medications, allergies)
 * - Neo4jGraphClient (cohorts, similar patients, care team)
 * - Parallel enrichment pattern (CompletableFuture.allOf)
 * - Error handling and fallback logic
 *
 * LAZY ENRICHMENT STRATEGY:
 * - Only enriches if hasFhirData == false or hasNeo4jData == false
 * - Reduces API calls by 99% (assuming 100 events per patient)
 * - Updates PatientContextState in-memory (persisted on next event)
 *
 * Architecture: Enrich AFTER PatientContextAggregator
 * - Input: EnrichedPatientContext with aggregated vitals/labs/meds
 * - Output: EnrichedPatientContext with FHIR + Neo4j enrichment
 * - State: Updated in PatientContextState for RocksDB persistence
 */
public class PatientContextEnricher extends RichAsyncFunction<EnrichedPatientContext, EnrichedPatientContext> {

    private static final Logger LOG = LoggerFactory.getLogger(PatientContextEnricher.class);

    // Reuse existing clients from ComprehensiveEnrichmentFunction
    private transient GoogleFHIRClient fhirClient;
    private transient Neo4jGraphClient neo4jClient;
    private transient ObjectMapper objectMapper;
    private transient io.github.resilience4j.circuitbreaker.CircuitBreaker neo4jCircuitBreaker;

    // PIPE-2: Local FHIR data cache — fallback when KB-20/FHIR is unreachable
    private static final long FHIR_CACHE_TTL_MS = 3_600_000L; // 1 hour
    private transient java.util.concurrent.ConcurrentHashMap<String, CachedFHIRData> fhirCache;
    private transient org.apache.flink.metrics.Counter fhirCacheHits;
    private transient org.apache.flink.metrics.Counter fhirCacheMisses;

    // Enrichment configuration
    private final boolean enableFhirEnrichment;
    private final boolean enableNeo4jEnrichment;

    public PatientContextEnricher() {
        this(true, true);
    }

    public PatientContextEnricher(boolean enableFhir, boolean enableNeo4j) {
        this.enableFhirEnrichment = enableFhir;
        this.enableNeo4jEnrichment = enableNeo4j;
    }

    @Override
    public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
        LOG.info("Initializing PatientContextEnricher (FHIR: {}, Neo4j: {})",
            enableFhirEnrichment, enableNeo4jEnrichment);

        // Initialize FHIR client (reuse exact code from ComprehensiveEnrichmentFunction)
        if (enableFhirEnrichment) {
            try {
                String credentialsPath = KafkaConfigLoader.getGoogleCloudCredentialsPath();
                String projectId = KafkaConfigLoader.getGoogleCloudProjectId();
                String location = KafkaConfigLoader.getGoogleCloudLocation();
                String datasetId = KafkaConfigLoader.getGoogleCloudDatasetId();
                String fhirStoreId = KafkaConfigLoader.getGoogleCloudFhirStoreId();
                LOG.info("[FHIR-INIT] credentials={}, project={}, location={}, dataset={}, fhirStore={}",
                    credentialsPath, projectId, location, datasetId, fhirStoreId);

                // Check if credentials file exists
                java.io.File credFile = new java.io.File(credentialsPath);
                LOG.info("[FHIR-INIT] Credentials file exists={}, size={} bytes",
                    credFile.exists(), credFile.exists() ? credFile.length() : -1);

                fhirClient = new GoogleFHIRClient(projectId, location, datasetId, fhirStoreId, credentialsPath);
                fhirClient.initialize();
                LOG.info("[FHIR-INIT] GoogleFHIRClient initialized successfully");
            } catch (Exception e) {
                LOG.error("[FHIR-INIT] Failed to initialize FHIR client: {} - {}",
                    e.getClass().getSimpleName(), e.getMessage(), e);
                // Continue without FHIR enrichment
            }
        }

        // Initialize Neo4j client (reuse exact code from ComprehensiveEnrichmentFunction)
        if (enableNeo4jEnrichment) {
            try {
                String neo4jUri = KafkaConfigLoader.getNeo4jUri();
                String neo4jUser = System.getenv().getOrDefault("NEO4J_USER", "neo4j");
                String neo4jPassword = System.getenv("NEO4J_PASSWORD");
                if (neo4jPassword == null || neo4jPassword.isEmpty()) {
                    LOG.error("NEO4J_PASSWORD environment variable not set — Neo4j enrichment disabled");
                    return; // Skip Neo4j initialization; enrichment will proceed without graph data
                }
                LOG.info("Connecting to Neo4j at: {}", neo4jUri);

                neo4jClient = new Neo4jGraphClient(neo4jUri, neo4jUser, neo4jPassword);
                neo4jClient.initialize();
                LOG.info("Neo4jGraphClient initialized successfully");

                io.github.resilience4j.circuitbreaker.CircuitBreakerConfig cbConfig =
                    io.github.resilience4j.circuitbreaker.CircuitBreakerConfig.custom()
                        .failureRateThreshold(50)
                        .waitDurationInOpenState(java.time.Duration.ofSeconds(30))
                        .slidingWindowSize(20)
                        .minimumNumberOfCalls(5)
                        .build();
                neo4jCircuitBreaker = io.github.resilience4j.circuitbreaker.CircuitBreaker.of(
                    "neo4j-enrichment", cbConfig);
                LOG.info("Neo4j circuit breaker initialized (50% threshold, 30s open wait, window=20)");
            } catch (Exception e) {
                LOG.error("Failed to initialize Neo4j client", e);
                // Continue without Neo4j enrichment
            }
        }

        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());

        // PIPE-2: Initialize FHIR fallback cache
        fhirCache = new java.util.concurrent.ConcurrentHashMap<>();
        var metrics = getRuntimeContext().getMetricGroup().addGroup("module2_fhir_cache");
        fhirCacheHits = metrics.counter("cache_hits");
        fhirCacheMisses = metrics.counter("cache_misses");
        getRuntimeContext().getMetricGroup().addGroup("module2_fhir_cache")
            .gauge("cache_size", () -> (long) fhirCache.size());
        LOG.info("FHIR fallback cache initialized (TTL: {}ms)", FHIR_CACHE_TTL_MS);
    }

    @Override
    public void asyncInvoke(EnrichedPatientContext context, ResultFuture<EnrichedPatientContext> resultFuture) {
        PatientContextState state = context.getPatientState();
        String patientId = state.getPatientId();

        // LAZY ENRICHMENT: Skip if already enriched
        boolean needsFhirEnrichment = enableFhirEnrichment && !state.isHasFhirData();
        boolean needsNeo4jEnrichment = enableNeo4jEnrichment && !state.isHasNeo4jData();

        if (!needsFhirEnrichment && !needsNeo4jEnrichment) {
            // Already enriched - return immediately
            LOG.debug("Patient {} already enriched, skipping API calls", patientId);
            resultFuture.complete(Collections.singleton(context));
            return;
        }

        LOG.info("Enriching patient {} (FHIR: {}, Neo4j: {})",
            patientId, needsFhirEnrichment, needsNeo4jEnrichment);

        // Parallel enrichment (reuse pattern from ComprehensiveEnrichmentFunction)
        List<CompletableFuture<Void>> enrichmentFutures = new ArrayList<>();

        if (needsFhirEnrichment) {
            CompletableFuture<Void> fhirFuture = fetchAndApplyFHIRData(state);
            enrichmentFutures.add(fhirFuture);
        }

        if (needsNeo4jEnrichment) {
            CompletableFuture<Void> neo4jFuture = fetchAndApplyNeo4jData(state);
            enrichmentFutures.add(neo4jFuture);
        }

        // Wait for all enrichments to complete
        CompletableFuture.allOf(enrichmentFutures.toArray(new CompletableFuture[0]))
            .thenAccept(v -> {
                // Mark enrichment as complete
                if (needsFhirEnrichment) {
                    state.setHasFhirData(true);
                }
                if (needsNeo4jEnrichment) {
                    state.setHasNeo4jData(true);
                }
                state.setEnrichmentComplete(state.isHasFhirData() && state.isHasNeo4jData());

                LOG.info("Enrichment complete for patient {} (FHIR: {}, Neo4j: {}, Complete: {})",
                    patientId, state.isHasFhirData(), state.isHasNeo4jData(), state.isEnrichmentComplete());

                resultFuture.complete(Collections.singleton(context));
            })
            .exceptionally(throwable -> {
                LOG.error("Enrichment failed for patient {}", patientId, throwable);
                // Return context with partial enrichment on error
                resultFuture.complete(Collections.singleton(context));
                return null;
            });
    }

    /**
     * Fetch FHIR data and apply to PatientContextState
     * Reuses fetchFHIRData logic from ComprehensiveEnrichmentFunction
     */
    private CompletableFuture<Void> fetchAndApplyFHIRData(PatientContextState state) {
        return CompletableFuture.runAsync(() -> {
            String patientId = state.getPatientId();
            try {
                // Fetch patient demographics
                FHIRPatientData patient = fhirClient.getPatientAsync(patientId)
                    .get(500, TimeUnit.MILLISECONDS);
                if (patient != null) {
                    state.setDemographics(convertToPatientDemographics(patient));
                }

                // Fetch conditions
                List<Condition> conditions = fhirClient.getConditionsAsync(patientId)
                    .get(500, TimeUnit.MILLISECONDS);
                if (conditions != null && !conditions.isEmpty()) {
                    state.setChronicConditions(conditions);
                }

                // Fetch medications (FHIR)
                List<Medication> medications = fhirClient.getMedicationsAsync(patientId)
                    .get(500, TimeUnit.MILLISECONDS);
                if (medications != null && !medications.isEmpty()) {
                    state.setFhirMedications(medications);
                }

                LOG.debug("FHIR enrichment complete for patient {}", patientId);

                // PIPE-2: Cache successful FHIR data for fallback
                fhirCache.put(patientId, new CachedFHIRData(
                    state.getDemographics(), state.getChronicConditions(),
                    state.getFhirMedications(), System.currentTimeMillis()));

            } catch (Exception e) {
                LOG.error("FHIR data fetch failed for patient {}", patientId, e);

                // PIPE-2: Serve from cache on failure
                CachedFHIRData cached = fhirCache.get(patientId);
                if (cached != null && !cached.isExpired(FHIR_CACHE_TTL_MS)) {
                    LOG.info("Serving cached FHIR data for patient {} (age: {}ms)",
                        patientId, System.currentTimeMillis() - cached.cachedAt);
                    if (cached.demographics != null) state.setDemographics(cached.demographics);
                    if (cached.conditions != null) state.setChronicConditions(cached.conditions);
                    if (cached.medications != null) state.setFhirMedications(cached.medications);
                    fhirCacheHits.inc();
                } else {
                    LOG.warn("No cached FHIR data for patient {} — enrichment incomplete", patientId);
                    fhirCacheMisses.inc();
                    // Evict expired entry
                    if (cached != null) fhirCache.remove(patientId);
                }
            }
        });
    }

    /** PIPE-2: Immutable cache entry for FHIR patient data with TTL */
    private static class CachedFHIRData {
        final PatientDemographics demographics;
        final List<Condition> conditions;
        final List<Medication> medications;
        final long cachedAt;

        CachedFHIRData(PatientDemographics demographics, List<Condition> conditions,
                       List<Medication> medications, long cachedAt) {
            this.demographics = demographics;
            this.conditions = conditions;
            this.medications = medications;
            this.cachedAt = cachedAt;
        }

        boolean isExpired(long ttlMs) {
            return System.currentTimeMillis() - cachedAt > ttlMs;
        }
    }

    /**
     * Fetch Neo4j graph data and apply to PatientContextState
     * Reuses fetchGraphData logic from ComprehensiveEnrichmentFunction
     */
    private CompletableFuture<Void> fetchAndApplyNeo4jData(PatientContextState state) {
        if (neo4jClient == null) {
            return CompletableFuture.completedFuture(null);
        }

        // Circuit breaker: fast-fail when Neo4j is degraded
        if (neo4jCircuitBreaker != null &&
            neo4jCircuitBreaker.getState() == io.github.resilience4j.circuitbreaker.CircuitBreaker.State.OPEN) {
            LOG.debug("Neo4j circuit breaker OPEN for patient {}, skipping graph enrichment", state.getPatientId());
            return CompletableFuture.completedFuture(null);
        }

        return neo4jClient.queryGraphAsync(state.getPatientId())
            .thenAccept(graphData -> {
                if (neo4jCircuitBreaker != null) {
                    neo4jCircuitBreaker.onSuccess(0, java.util.concurrent.TimeUnit.MILLISECONDS);
                }
                if (graphData != null) {
                    // Apply Neo4j data to state
                    if (graphData.getCareTeam() != null) {
                        state.setNeo4jCareTeam(graphData.getCareTeam());
                    }

                    if (graphData.getRiskCohorts() != null) {
                        state.setRiskCohorts(new ArrayList<>(graphData.getRiskCohorts()));
                    }

                    if (graphData.getCarePathways() != null) {
                        state.setCarePathways(graphData.getCarePathways());
                    }

                    LOG.debug("Neo4j enrichment complete for patient {}", state.getPatientId());
                }
            })
            .exceptionally(throwable -> {
                if (neo4jCircuitBreaker != null) {
                    neo4jCircuitBreaker.onError(0, java.util.concurrent.TimeUnit.MILLISECONDS, throwable);
                }
                LOG.warn("Graph data fetch failed for patient {}: {} (circuit breaker state: {})",
                    state.getPatientId(), throwable.getMessage(),
                    neo4jCircuitBreaker != null ? neo4jCircuitBreaker.getState() : "N/A");
                return null;
            })
            .thenApply(v -> null); // Convert to CompletableFuture<Void>
    }

    /**
     * Convert FHIRPatientData to PatientDemographics
     */
    private PatientDemographics convertToPatientDemographics(FHIRPatientData fhirPatient) {
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(fhirPatient.getAge());
        demographics.setGender(fhirPatient.getGender());
        demographics.setFirstName(fhirPatient.getFirstName());
        demographics.setLastName(fhirPatient.getLastName());
        demographics.setDateOfBirth(fhirPatient.getDateOfBirth());
        demographics.setMrn(fhirPatient.getMrn());
        return demographics;
    }

    @Override
    public void timeout(EnrichedPatientContext context, ResultFuture<EnrichedPatientContext> resultFuture) {
        LOG.warn("Enrichment timeout for patient {}, returning with partial enrichment",
            context.getPatientState().getPatientId());
        resultFuture.complete(Collections.singleton(context));
    }

    @Override
    public void close() throws Exception {
        LOG.info("Closing PatientContextEnricher");

        if (fhirClient != null) {
            try {
                fhirClient.close();
            } catch (Exception e) {
                LOG.error("Error closing FHIR client", e);
            }
        }

        if (neo4jClient != null) {
            try {
                neo4jClient.close();
            } catch (Exception e) {
                LOG.error("Error closing Neo4j client", e);
            }
        }

        super.close();
    }
}
