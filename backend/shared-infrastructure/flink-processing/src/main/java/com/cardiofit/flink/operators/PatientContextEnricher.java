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
                LOG.info("Loading Google Cloud credentials from: {}", credentialsPath);

                fhirClient = new GoogleFHIRClient(
                    KafkaConfigLoader.getGoogleCloudProjectId(),
                    KafkaConfigLoader.getGoogleCloudLocation(),
                    KafkaConfigLoader.getGoogleCloudDatasetId(),
                    KafkaConfigLoader.getGoogleCloudFhirStoreId(),
                    credentialsPath
                );
                fhirClient.initialize();
                LOG.info("GoogleFHIRClient initialized successfully");
            } catch (Exception e) {
                LOG.error("Failed to initialize FHIR client", e);
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
            } catch (Exception e) {
                LOG.error("Failed to initialize Neo4j client", e);
                // Continue without Neo4j enrichment
            }
        }

        objectMapper = new ObjectMapper();
        objectMapper.registerModule(new JavaTimeModule());
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

                // Note: GoogleFHIRClient doesn't have direct getAllergiesAsync or getCareTeamAsync
                // These would need to be parsed from AllergyIntolerance and CareTeam FHIR resources
                // For Phase 1, we'll leave these for Phase 2 enhancement
                // Allergies and care team can be extracted from patient/condition resources later

                LOG.debug("FHIR enrichment complete for patient {}", patientId);

            } catch (Exception e) {
                LOG.error("FHIR data fetch failed for patient {}", patientId, e);
                // Partial enrichment is acceptable
            }
        });
    }

    /**
     * Fetch Neo4j graph data and apply to PatientContextState
     * Reuses fetchGraphData logic from ComprehensiveEnrichmentFunction
     */
    private CompletableFuture<Void> fetchAndApplyNeo4jData(PatientContextState state) {
        if (neo4jClient == null) {
            return CompletableFuture.completedFuture(null);
        }

        return neo4jClient.queryGraphAsync(state.getPatientId())
            .thenAccept(graphData -> {
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

                    // Placeholder for similar patients (would require new Neo4j query)
                    // state.setSimilarPatients(...);

                    // Placeholder for cohort insights (would require new Neo4j query)
                    // state.setCohortInsights(...);

                    LOG.debug("Neo4j enrichment complete for patient {}", state.getPatientId());
                }
            })
            .exceptionally(throwable -> {
                LOG.warn("Graph data fetch failed for patient {}: {}",
                    state.getPatientId(), throwable.getMessage());
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
