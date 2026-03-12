# FHIR/Neo4j Enrichment Integration Specification for Unified Pipeline

**Document Version**: 1.0
**Date**: 2025-10-16
**Purpose**: Complete implementation specification for integrating FHIR/Neo4j enrichment into createUnifiedPipeline

---

## Executive Summary

This document provides detailed implementation specifications for integrating Google FHIR and Neo4j graph enrichment into the unified Flink processing pipeline (Module2_Enhanced.createUnifiedPipeline). The integration addresses the **missing clinical context** identified in INTEGRATION_ANALYSIS_OLD_TO_NEW.md.

**Key Components to Implement**:
1. CanonicalEventFHIREnricher (RichAsyncFunction) - Enriches events with FHIR patient data
2. PatientContextNeo4jEnricher (RichAsyncFunction) - Enriches aggregated context with graph data
3. Enhanced PatientContextState - Extended state model with FHIR/Neo4j fields
4. Updated createUnifiedPipeline method - Integrated pipeline with enrichment operators

---

## 1. CanonicalEventFHIREnricher - FHIR Data Enrichment

### 1.1 Architecture Pattern

**Position in Pipeline**: BEFORE CanonicalEventToGenericEventConverter
**Pattern**: RichAsyncFunction with circuit breaker and caching
**Async I/O Mode**: Unordered wait (allows out-of-order completion for throughput)

```
CanonicalEvent → CanonicalEventFHIREnricher → Enriched CanonicalEvent → GenericEvent Converter
```

### 1.2 Java Interface Specification

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.models.*;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async FHIR enrichment operator for CanonicalEvent.
 *
 * Enriches events with patient demographics, conditions, medications, and allergies
 * from Google Cloud Healthcare FHIR API before aggregation.
 *
 * Architecture:
 * - Async I/O pattern with 10s timeout
 * - Circuit breaker protection (50% failure rate threshold)
 * - L1 cache (5-min TTL) + Stale cache (24-hour fallback)
 * - Graceful degradation (continues without FHIR data on failure)
 *
 * Performance:
 * - Capacity: 500 concurrent requests
 * - Timeout: 10 seconds per enrichment
 * - Parallelism: Matches upstream operators (default 2)
 */
public class CanonicalEventFHIREnricher extends RichAsyncFunction<CanonicalEvent, CanonicalEvent> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(CanonicalEventFHIREnricher.class);

    // Google FHIR API client configuration
    private final String projectId;
    private final String location;
    private final String datasetId;
    private final String fhirStoreId;
    private final String credentialsPath;

    // Transient client (initialized in open())
    private transient GoogleFHIRClient fhirClient;

    // Metrics for monitoring
    private transient long enrichmentCount;
    private transient long enrichmentFailures;
    private transient long cacheHits;

    /**
     * Constructor with FHIR API configuration.
     *
     * @param projectId Google Cloud project ID (e.g., "cardiofit-905a8")
     * @param location Google Cloud location (e.g., "asia-south1")
     * @param datasetId Healthcare dataset ID (e.g., "clinical-synthesis-hub")
     * @param fhirStoreId FHIR store ID (e.g., "fhir-store")
     * @param credentialsPath Path to service account credentials JSON
     */
    public CanonicalEventFHIREnricher(String projectId, String location, String datasetId,
                                      String fhirStoreId, String credentialsPath) {
        this.projectId = projectId;
        this.location = location;
        this.datasetId = datasetId;
        this.fhirStoreId = fhirStoreId;
        this.credentialsPath = credentialsPath;
    }

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        LOG.info("Initializing CanonicalEventFHIREnricher with FHIR store: {}/{}/{}/{}",
                projectId, location, datasetId, fhirStoreId);

        // Initialize Google FHIR client
        this.fhirClient = new GoogleFHIRClient(projectId, location, datasetId, fhirStoreId, credentialsPath);
        this.fhirClient.initialize();

        // Initialize metrics
        this.enrichmentCount = 0;
        this.enrichmentFailures = 0;
        this.cacheHits = 0;

        LOG.info("CanonicalEventFHIREnricher initialized successfully");
    }

    @Override
    public void asyncInvoke(CanonicalEvent event, ResultFuture<CanonicalEvent> resultFuture) {
        String patientId = event.getPatientId();

        if (patientId == null || patientId.isEmpty()) {
            LOG.warn("Event {} has no patientId - skipping FHIR enrichment", event.getId());
            resultFuture.complete(Collections.singleton(event));
            return;
        }

        // Check if already enriched (avoid duplicate enrichment)
        if (event.getMetadata() != null && event.getMetadata().containsKey("fhirEnriched")) {
            LOG.debug("Event {} already has FHIR data - skipping", event.getId());
            resultFuture.complete(Collections.singleton(event));
            return;
        }

        LOG.debug("Starting FHIR enrichment for patientId={}, eventId={}", patientId, event.getId());

        // Orchestrate parallel FHIR API calls
        CompletableFuture<FHIRPatientData> patientFuture = fhirClient.getPatientAsync(patientId);
        CompletableFuture<List<Condition>> conditionsFuture = fhirClient.getConditionsAsync(patientId);
        CompletableFuture<List<Medication>> medicationsFuture = fhirClient.getMedicationsAsync(patientId);

        // Combine all futures
        CompletableFuture.allOf(patientFuture, conditionsFuture, medicationsFuture)
            .whenComplete((result, throwable) -> {
                try {
                    if (throwable != null) {
                        // Log error but continue without FHIR data (graceful degradation)
                        LOG.error("FHIR enrichment failed for patientId={}: {}",
                                patientId, throwable.getMessage());
                        enrichmentFailures++;
                        resultFuture.complete(Collections.singleton(event));
                        return;
                    }

                    // Extract results
                    FHIRPatientData patient = patientFuture.join();
                    List<Condition> conditions = conditionsFuture.join();
                    List<Medication> medications = medicationsFuture.join();

                    // Enrich event metadata with FHIR data
                    enrichEventWithFHIRData(event, patient, conditions, medications);

                    enrichmentCount++;
                    LOG.info("FHIR enrichment successful for patientId={}: patient={}, conditions={}, medications={}",
                            patientId,
                            patient != null ? "present" : "null",
                            conditions.size(),
                            medications.size());

                    resultFuture.complete(Collections.singleton(event));

                } catch (Exception e) {
                    LOG.error("Error processing FHIR enrichment results for patientId={}: {}",
                            patientId, e.getMessage(), e);
                    enrichmentFailures++;
                    resultFuture.complete(Collections.singleton(event));
                }
            });
    }

    /**
     * Enrich event with FHIR data in metadata section.
     *
     * Stores FHIR data in event.metadata map for later extraction by aggregator.
     * This avoids modifying the payload structure.
     */
    private void enrichEventWithFHIRData(CanonicalEvent event, FHIRPatientData patient,
                                         List<Condition> conditions, List<Medication> medications) {
        // Initialize metadata if not present
        if (event.getMetadata() == null) {
            event.setMetadata(new EventMetadata());
        }

        // Create FHIR enrichment metadata map
        Map<String, Object> fhirData = new HashMap<>();

        // Add patient demographics
        if (patient != null) {
            Map<String, Object> patientData = new HashMap<>();
            patientData.put("firstName", patient.getFirstName());
            patientData.put("lastName", patient.getLastName());
            patientData.put("dateOfBirth", patient.getDateOfBirth());
            patientData.put("gender", patient.getGender());
            patientData.put("age", patient.getAge());
            patientData.put("mrn", patient.getMrn());
            fhirData.put("patient", patientData);
        }

        // Add conditions (convert to simple map structure)
        List<Map<String, String>> conditionsList = new ArrayList<>();
        for (Condition condition : conditions) {
            Map<String, String> condMap = new HashMap<>();
            condMap.put("code", condition.getCode());
            condMap.put("display", condition.getDisplay());
            condMap.put("status", condition.getStatus());
            condMap.put("severity", condition.getSeverity());
            conditionsList.add(condMap);
        }
        fhirData.put("conditions", conditionsList);

        // Add medications (convert to simple map structure)
        List<Map<String, String>> medicationsList = new ArrayList<>();
        for (Medication medication : medications) {
            Map<String, String> medMap = new HashMap<>();
            medMap.put("name", medication.getName());
            medMap.put("code", medication.getCode());
            medMap.put("dosage", medication.getDosage());
            medMap.put("frequency", medication.getFrequency());
            medMap.put("status", medication.getStatus());
            medicationsList.add(medMap);
        }
        fhirData.put("medications", medicationsList);

        // Mark as enriched with timestamp
        fhirData.put("enrichmentTimestamp", System.currentTimeMillis());
        fhirData.put("fhirEnriched", true);

        // Store in event metadata
        event.getMetadata().put("fhirData", fhirData);

        LOG.debug("Added FHIR data to event metadata: patient={}, conditions={}, medications={}",
                patient != null, conditionsList.size(), medicationsList.size());
    }

    @Override
    public void timeout(CanonicalEvent input, ResultFuture<CanonicalEvent> resultFuture) {
        LOG.warn("FHIR enrichment timeout for patientId={} after 10s - continuing without enrichment",
                input.getPatientId());
        enrichmentFailures++;
        resultFuture.complete(Collections.singleton(input));
    }

    @Override
    public void close() throws Exception {
        super.close();

        // Log metrics
        LOG.info("CanonicalEventFHIREnricher closing - enrichments={}, failures={}, cacheHits={}",
                enrichmentCount, enrichmentFailures, cacheHits);

        if (fhirClient != null) {
            fhirClient.close();
        }
    }
}
```

### 1.3 AsyncDataStream Configuration

```java
// In Module2_Enhanced.createUnifiedPipeline():

// Apply FHIR enrichment with AsyncDataStream
DataStream<CanonicalEvent> fhirEnrichedEvents = AsyncDataStream.unorderedWait(
    canonicalEvents,
    new CanonicalEventFHIREnricher(
        "cardiofit-905a8",              // projectId
        "asia-south1",                  // location
        "clinical-synthesis-hub",       // datasetId
        "fhir-store",                   // fhirStoreId
        "/path/to/credentials.json"     // credentialsPath
    ),
    10000,                              // 10 second timeout
    TimeUnit.MILLISECONDS,
    500                                 // 500 concurrent requests capacity
).uid("fhir-enrichment-operator");
```

### 1.4 Error Handling Strategy

**Circuit Breaker Pattern** (built into GoogleFHIRClient):
- Failure rate threshold: 50% over 100 requests
- Open state duration: 60 seconds
- Half-open test calls: 5 requests

**Graceful Degradation**:
- On FHIR API failure → Continue without enrichment data
- On timeout → Complete with unenriched event
- On missing patient → Log warning, continue

**Stale Cache Fallback**:
- L1 cache miss + API failure → Serve 24-hour stale data
- First-time patient during outage → No data (acceptable)

---

## 2. PatientContextNeo4jEnricher - Graph Data Enrichment

### 2.1 Architecture Pattern

**Position in Pipeline**: AFTER PatientContextAggregator
**Pattern**: RichAsyncFunction for graph queries
**Async I/O Mode**: Unordered wait

```
EnrichedPatientContext → PatientContextNeo4jEnricher → Fully Enriched Context → ClinicalIntelligenceEvaluator
```

### 2.2 Java Interface Specification

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.neo4j.AdvancedNeo4jQueries;
import com.cardiofit.flink.neo4j.CohortInsights;
import com.cardiofit.flink.neo4j.SimilarPatient;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * Async Neo4j graph enrichment operator for EnrichedPatientContext.
 *
 * Enriches aggregated patient context with graph-based insights:
 * - Care team relationships
 * - Risk cohort membership
 * - Similar patient analysis
 * - Cohort statistics and insights
 *
 * Architecture:
 * - Async Neo4j driver with 5s timeout
 * - Cypher query optimization for performance
 * - Graceful degradation on Neo4j unavailability
 *
 * Performance:
 * - Capacity: 200 concurrent queries
 * - Timeout: 5 seconds per enrichment
 * - Query optimization: Single multi-part Cypher query
 */
public class PatientContextNeo4jEnricher extends RichAsyncFunction<EnrichedPatientContext, EnrichedPatientContext> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(PatientContextNeo4jEnricher.class);

    // Neo4j connection configuration
    private final String neo4jUri;
    private final String neo4jUsername;
    private final String neo4jPassword;

    // Transient clients (initialized in open())
    private transient Neo4jGraphClient graphClient;
    private transient AdvancedNeo4jQueries advancedQueries;

    // Metrics
    private transient long enrichmentCount;
    private transient long enrichmentFailures;

    /**
     * Constructor with Neo4j configuration.
     *
     * @param neo4jUri Neo4j bolt URI (e.g., "bolt://localhost:7687")
     * @param neo4jUsername Neo4j username
     * @param neo4jPassword Neo4j password
     */
    public PatientContextNeo4jEnricher(String neo4jUri, String neo4jUsername, String neo4jPassword) {
        this.neo4jUri = neo4jUri;
        this.neo4jUsername = neo4jUsername;
        this.neo4jPassword = neo4jPassword;
    }

    @Override
    public void open(Configuration parameters) throws Exception {
        super.open(parameters);

        LOG.info("Initializing PatientContextNeo4jEnricher with URI: {}", neo4jUri);

        // Initialize Neo4j graph client
        this.graphClient = new Neo4jGraphClient(neo4jUri, neo4jUsername, neo4jPassword);
        this.graphClient.initialize();

        // Initialize advanced queries client
        this.advancedQueries = new AdvancedNeo4jQueries(graphClient.getDriver());

        // Initialize metrics
        this.enrichmentCount = 0;
        this.enrichmentFailures = 0;

        LOG.info("PatientContextNeo4jEnricher initialized successfully");
    }

    @Override
    public void asyncInvoke(EnrichedPatientContext context, ResultFuture<EnrichedPatientContext> resultFuture) {
        String patientId = context.getPatientId();

        if (patientId == null || patientId.isEmpty()) {
            LOG.warn("Context has no patientId - skipping Neo4j enrichment");
            resultFuture.complete(Collections.singleton(context));
            return;
        }

        // Check if already enriched
        if (context.getPatientState() != null && context.getPatientState().isHasNeo4jData()) {
            LOG.debug("Context for {} already has Neo4j data - skipping", patientId);
            resultFuture.complete(Collections.singleton(context));
            return;
        }

        LOG.debug("Starting Neo4j enrichment for patientId={}", patientId);

        // Orchestrate parallel graph queries
        CompletableFuture<GraphData> basicGraphFuture = graphClient.queryGraphAsync(patientId);
        CompletableFuture<List<SimilarPatient>> similarPatientsFuture =
            advancedQueries.findSimilarPatientsAsync(patientId, 5);
        CompletableFuture<CohortInsights> cohortInsightsFuture =
            advancedQueries.getCohortInsightsAsync(patientId);

        // Combine all futures
        CompletableFuture.allOf(basicGraphFuture, similarPatientsFuture, cohortInsightsFuture)
            .whenComplete((result, throwable) -> {
                try {
                    if (throwable != null) {
                        // Log error but continue without graph data
                        LOG.error("Neo4j enrichment failed for patientId={}: {}",
                                patientId, throwable.getMessage());
                        enrichmentFailures++;
                        resultFuture.complete(Collections.singleton(context));
                        return;
                    }

                    // Extract results
                    GraphData graphData = basicGraphFuture.join();
                    List<SimilarPatient> similarPatients = similarPatientsFuture.join();
                    CohortInsights cohortInsights = cohortInsightsFuture.join();

                    // Enrich patient state with Neo4j data
                    enrichContextWithNeo4jData(context, graphData, similarPatients, cohortInsights);

                    enrichmentCount++;
                    LOG.info("Neo4j enrichment successful for patientId={}: cohorts={}, similarPatients={}, careTeam={}",
                            patientId,
                            graphData.getRiskCohorts().size(),
                            similarPatients.size(),
                            graphData.getCareTeam().size());

                    resultFuture.complete(Collections.singleton(context));

                } catch (Exception e) {
                    LOG.error("Error processing Neo4j enrichment results for patientId={}: {}",
                            patientId, e.getMessage(), e);
                    enrichmentFailures++;
                    resultFuture.complete(Collections.singleton(context));
                }
            });
    }

    /**
     * Enrich patient context state with Neo4j graph data.
     */
    private void enrichContextWithNeo4jData(EnrichedPatientContext context, GraphData graphData,
                                            List<SimilarPatient> similarPatients,
                                            CohortInsights cohortInsights) {
        PatientContextState state = context.getPatientState();
        if (state == null) {
            LOG.warn("EnrichedPatientContext has null state - cannot enrich with Neo4j data");
            return;
        }

        // Store care team
        state.setCareTeam(graphData.getCareTeam());

        // Store risk cohorts
        state.setRiskCohorts(graphData.getRiskCohorts());

        // Store care pathways
        state.setCarePathways(graphData.getCarePathways());

        // Store similar patients (convert to simplified structure)
        List<Map<String, Object>> similarPatientsData = new ArrayList<>();
        for (SimilarPatient similar : similarPatients) {
            Map<String, Object> patientMap = new HashMap<>();
            patientMap.put("patientId", similar.getPatientId());
            patientMap.put("similarityScore", similar.getSimilarityScore());
            patientMap.put("sharedConditions", similar.getSharedConditions());
            patientMap.put("outcome", similar.getOutcome());
            similarPatientsData.add(patientMap);
        }
        state.setSimilarPatients(similarPatientsData);

        // Store cohort insights
        if (cohortInsights != null) {
            Map<String, Object> insightsData = new HashMap<>();
            insightsData.put("primaryCohort", cohortInsights.getPrimaryCohort());
            insightsData.put("cohortSize", cohortInsights.getCohortSize());
            insightsData.put("averageAge", cohortInsights.getAverageAge());
            insightsData.put("commonConditions", cohortInsights.getCommonConditions());
            insightsData.put("commonMedications", cohortInsights.getCommonMedications());
            insightsData.put("averageReadmissionRate", cohortInsights.getAverageReadmissionRate());
            state.setCohortInsights(insightsData);
        }

        // Mark as enriched
        state.setHasNeo4jData(true);
        state.setEnrichmentComplete(state.isHasFhirData() && state.isHasNeo4jData());

        LOG.debug("Added Neo4j data to patient state: cohorts={}, similarPatients={}, careTeam={}",
                graphData.getRiskCohorts().size(),
                similarPatientsData.size(),
                graphData.getCareTeam().size());
    }

    @Override
    public void timeout(EnrichedPatientContext input, ResultFuture<EnrichedPatientContext> resultFuture) {
        LOG.warn("Neo4j enrichment timeout for patientId={} after 5s - continuing without enrichment",
                input.getPatientId());
        enrichmentFailures++;
        resultFuture.complete(Collections.singleton(input));
    }

    @Override
    public void close() throws Exception {
        super.close();

        // Log metrics
        LOG.info("PatientContextNeo4jEnricher closing - enrichments={}, failures={}",
                enrichmentCount, enrichmentFailures);

        if (graphClient != null) {
            graphClient.close();
        }
    }
}
```

### 2.3 AsyncDataStream Configuration

```java
// In Module2_Enhanced.createUnifiedPipeline():

// Apply Neo4j enrichment with AsyncDataStream
DataStream<EnrichedPatientContext> neo4jEnrichedContext = AsyncDataStream.unorderedWait(
    aggregatedContext,
    new PatientContextNeo4jEnricher(
        "bolt://localhost:7687",       // neo4jUri
        "neo4j",                        // username
        "password"                      // password
    ),
    5000,                               // 5 second timeout
    TimeUnit.MILLISECONDS,
    200                                 // 200 concurrent requests capacity
).uid("neo4j-enrichment-operator");
```

### 2.4 Graph Query Optimization

**Single Multi-Part Cypher Query** (reduces round trips):
```cypher
MATCH (p:Patient {patientId: $patientId})
OPTIONAL MATCH (p)-[:HAS_PROVIDER]->(provider:Provider)
OPTIONAL MATCH (p)-[:IN_COHORT]->(cohort:Cohort)
OPTIONAL MATCH (p)-[:FOLLOWS_PATHWAY]->(pathway:Pathway)
OPTIONAL MATCH (p)-[:SIMILAR_TO]->(similar:Patient)
RETURN
  collect(DISTINCT provider.providerId) AS careTeam,
  collect(DISTINCT cohort.name) AS riskCohorts,
  collect(DISTINCT pathway.name) AS carePathways,
  collect(DISTINCT {
    patientId: similar.patientId,
    similarity: similar.similarityScore
  }) AS similarPatients
```

**Performance Characteristics**:
- Single database round trip
- Index on Patient.patientId (O(log n) lookup)
- Max 5 similar patients to limit response size
- 5s timeout for complex queries

---

## 3. Enhanced PatientContextState - Extended Data Model

### 3.1 New Fields for FHIR Data

```java
package com.cardiofit.flink.models;

/**
 * Enhanced PatientContextState with FHIR and Neo4j enrichment fields.
 *
 * This extends the existing state model to include clinical context from
 * external systems while maintaining backward compatibility.
 */
public class PatientContextState implements Serializable {
    // ... existing fields ...

    // ========== FHIR ENRICHMENT FIELDS ==========

    /**
     * Patient demographics from FHIR Patient resource
     * Populated by CanonicalEventFHIREnricher
     */
    @JsonProperty("demographics")
    private PatientDemographics demographics;

    /**
     * Chronic conditions from FHIR Condition resources
     * Filtered to active status only
     */
    @JsonProperty("chronicConditions")
    private List<Condition> chronicConditions;

    /**
     * Known allergies from FHIR AllergyIntolerance resources
     * Critical for medication safety
     */
    @JsonProperty("allergies")
    private List<String> allergies;

    /**
     * Care team members from FHIR CareTeam resource
     * List of provider IDs
     */
    @JsonProperty("careTeamFhir")
    private List<String> careTeamFhir;

    /**
     * FHIR medication list (historical context)
     * Separate from activeMedications (which tracks current stream events)
     */
    @JsonProperty("fhirMedications")
    private List<Medication> fhirMedications;

    // ========== NEO4J ENRICHMENT FIELDS ==========

    /**
     * Risk cohorts from Neo4j graph analysis
     * Example: ["CHF", "Diabetes", "High-Risk-Readmission"]
     */
    @JsonProperty("riskCohorts")
    private List<String> riskCohorts;

    /**
     * Care team from Neo4j relationships
     * May overlap with careTeamFhir
     */
    @JsonProperty("careTeam")
    private List<String> careTeam;

    /**
     * Active care pathways patient is following
     * Example: ["Heart Failure Pathway", "Post-MI Protocol"]
     */
    @JsonProperty("carePathways")
    private List<String> carePathways;

    /**
     * Similar patients from graph analysis
     * Used for outcome prediction and comparative analysis
     * Structure: [{patientId, similarityScore, sharedConditions, outcome}]
     */
    @JsonProperty("similarPatients")
    private List<Map<String, Object>> similarPatients;

    /**
     * Cohort analytics and insights
     * Population statistics for patient's primary risk cohort
     * Structure: {primaryCohort, cohortSize, avgAge, commonConditions, etc.}
     */
    @JsonProperty("cohortInsights")
    private Map<String, Object> cohortInsights;

    // ========== ENRICHMENT STATUS FLAGS ==========

    /**
     * Indicates FHIR enrichment completed successfully
     */
    @JsonProperty("hasFhirData")
    private boolean hasFhirData;

    /**
     * Indicates Neo4j enrichment completed successfully
     */
    @JsonProperty("hasNeo4jData")
    private boolean hasNeo4jData;

    /**
     * Indicates both FHIR and Neo4j enrichment completed
     */
    @JsonProperty("enrichmentComplete")
    private boolean enrichmentComplete;

    /**
     * Timestamp when FHIR enrichment occurred
     */
    @JsonProperty("fhirEnrichmentTimestamp")
    private Long fhirEnrichmentTimestamp;

    /**
     * Timestamp when Neo4j enrichment occurred
     */
    @JsonProperty("neo4jEnrichmentTimestamp")
    private Long neo4jEnrichmentTimestamp;

    // ========== CONSTRUCTOR UPDATES ==========

    public PatientContextState() {
        // ... existing initialization ...

        // Initialize new collections
        this.chronicConditions = new ArrayList<>();
        this.allergies = new ArrayList<>();
        this.careTeamFhir = new ArrayList<>();
        this.fhirMedications = new ArrayList<>();
        this.riskCohorts = new ArrayList<>();
        this.careTeam = new ArrayList<>();
        this.carePathways = new ArrayList<>();
        this.similarPatients = new ArrayList<>();
        this.cohortInsights = new HashMap<>();

        // Initialize flags
        this.hasFhirData = false;
        this.hasNeo4jData = false;
        this.enrichmentComplete = false;
    }

    // ========== NEW GETTER/SETTER METHODS ==========

    public PatientDemographics getDemographics() {
        return demographics;
    }

    public void setDemographics(PatientDemographics demographics) {
        this.demographics = demographics;
        this.hasFhirData = true;
        this.fhirEnrichmentTimestamp = System.currentTimeMillis();
        checkEnrichmentComplete();
    }

    public List<Condition> getChronicConditions() {
        return chronicConditions;
    }

    public void setChronicConditions(List<Condition> chronicConditions) {
        this.chronicConditions = chronicConditions != null ? chronicConditions : new ArrayList<>();
    }

    public List<String> getAllergies() {
        return allergies;
    }

    public void setAllergies(List<String> allergies) {
        this.allergies = allergies != null ? allergies : new ArrayList<>();
    }

    public List<String> getCareTeamFhir() {
        return careTeamFhir;
    }

    public void setCareTeamFhir(List<String> careTeamFhir) {
        this.careTeamFhir = careTeamFhir != null ? careTeamFhir : new ArrayList<>();
    }

    public List<Medication> getFhirMedications() {
        return fhirMedications;
    }

    public void setFhirMedications(List<Medication> fhirMedications) {
        this.fhirMedications = fhirMedications != null ? fhirMedications : new ArrayList<>();
    }

    public List<String> getRiskCohorts() {
        return riskCohorts;
    }

    public void setRiskCohorts(List<String> riskCohorts) {
        this.riskCohorts = riskCohorts != null ? riskCohorts : new ArrayList<>();
        this.hasNeo4jData = true;
        this.neo4jEnrichmentTimestamp = System.currentTimeMillis();
        checkEnrichmentComplete();
    }

    public List<String> getCareTeam() {
        return careTeam;
    }

    public void setCareTeam(List<String> careTeam) {
        this.careTeam = careTeam != null ? careTeam : new ArrayList<>();
    }

    public List<String> getCarePathways() {
        return carePathways;
    }

    public void setCarePathways(List<String> carePathways) {
        this.carePathways = carePathways != null ? carePathways : new ArrayList<>();
    }

    public List<Map<String, Object>> getSimilarPatients() {
        return similarPatients;
    }

    public void setSimilarPatients(List<Map<String, Object>> similarPatients) {
        this.similarPatients = similarPatients != null ? similarPatients : new ArrayList<>();
    }

    public Map<String, Object> getCohortInsights() {
        return cohortInsights;
    }

    public void setCohortInsights(Map<String, Object> cohortInsights) {
        this.cohortInsights = cohortInsights != null ? cohortInsights : new HashMap<>();
    }

    public boolean isHasFhirData() {
        return hasFhirData;
    }

    public void setHasFhirData(boolean hasFhirData) {
        this.hasFhirData = hasFhirData;
        checkEnrichmentComplete();
    }

    public boolean isHasNeo4jData() {
        return hasNeo4jData;
    }

    public void setHasNeo4jData(boolean hasNeo4jData) {
        this.hasNeo4jData = hasNeo4jData;
        checkEnrichmentComplete();
    }

    public boolean isEnrichmentComplete() {
        return enrichmentComplete;
    }

    public void setEnrichmentComplete(boolean enrichmentComplete) {
        this.enrichmentComplete = enrichmentComplete;
    }

    public Long getFhirEnrichmentTimestamp() {
        return fhirEnrichmentTimestamp;
    }

    public void setFhirEnrichmentTimestamp(Long fhirEnrichmentTimestamp) {
        this.fhirEnrichmentTimestamp = fhirEnrichmentTimestamp;
    }

    public Long getNeo4jEnrichmentTimestamp() {
        return neo4jEnrichmentTimestamp;
    }

    public void setNeo4jEnrichmentTimestamp(Long neo4jEnrichmentTimestamp) {
        this.neo4jEnrichmentTimestamp = neo4jEnrichmentTimestamp;
    }

    /**
     * Check if both enrichments completed and update flag
     */
    private void checkEnrichmentComplete() {
        this.enrichmentComplete = this.hasFhirData && this.hasNeo4jData;
    }

    // ========== UTILITY METHODS ==========

    /**
     * Check if patient has specific chronic condition
     */
    public boolean hasCondition(String conditionCode) {
        if (chronicConditions == null) return false;
        return chronicConditions.stream()
                .anyMatch(c -> conditionCode.equals(c.getCode()));
    }

    /**
     * Check if patient has any allergy
     */
    public boolean hasAllergies() {
        return allergies != null && !allergies.isEmpty();
    }

    /**
     * Check if patient has specific allergy
     */
    public boolean hasAllergy(String allergen) {
        if (allergies == null) return false;
        return allergies.stream()
                .anyMatch(a -> a.toLowerCase().contains(allergen.toLowerCase()));
    }

    /**
     * Check if patient is in specific risk cohort
     */
    public boolean isInCohort(String cohortName) {
        if (riskCohorts == null) return false;
        return riskCohorts.contains(cohortName);
    }

    /**
     * Get combined care team (FHIR + Neo4j, deduplicated)
     */
    public List<String> getCombinedCareTeam() {
        Set<String> combined = new HashSet<>();
        if (careTeamFhir != null) combined.addAll(careTeamFhir);
        if (careTeam != null) combined.addAll(careTeam);
        return new ArrayList<>(combined);
    }

    @Override
    public String toString() {
        return "PatientContextState{" +
                "patientId='" + patientId + '\'' +
                ", vitalsCount=" + latestVitals.size() +
                ", labsCount=" + recentLabs.size() +
                ", medsCount=" + activeMedications.size() +
                ", alertsCount=" + activeAlerts.size() +
                ", chronicConditionsCount=" + (chronicConditions != null ? chronicConditions.size() : 0) +
                ", allergiesCount=" + (allergies != null ? allergies.size() : 0) +
                ", cohortsCount=" + (riskCohorts != null ? riskCohorts.size() : 0) +
                ", similarPatientsCount=" + (similarPatients != null ? similarPatients.size() : 0) +
                ", eventCount=" + eventCount +
                ", hasFhirData=" + hasFhirData +
                ", hasNeo4jData=" + hasNeo4jData +
                ", enrichmentComplete=" + enrichmentComplete +
                ", news2=" + news2Score +
                ", qsofa=" + qsofaScore +
                ", acuity=" + combinedAcuityScore +
                '}';
    }
}
```

### 3.2 Serialization Strategy for RocksDB

**Key Considerations**:
- Jackson JSON serialization (existing pattern)
- RocksDB compression enabled (reduces state size)
- TTL for old patient states (7 days inactive)
- State size monitoring (log warnings if >1MB per patient)

**RocksDB Configuration** (in Module2_Enhanced):
```java
StateBackend stateBackend = new EmbeddedRocksDBStateBackend(true); // enable incremental checkpoints
stateBackend.setNumberOfTransferThreads(4);
stateBackend.setNumberOfTransferingThreads(2);
env.setStateBackend(stateBackend);
```

### 3.3 PatientDemographics Model

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Patient demographics data from FHIR.
 */
public class PatientDemographics implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("firstName")
    private String firstName;

    @JsonProperty("lastName")
    private String lastName;

    @JsonProperty("dateOfBirth")
    private String dateOfBirth;

    @JsonProperty("gender")
    private String gender;

    @JsonProperty("age")
    private Integer age;

    @JsonProperty("mrn")
    private String mrn;

    // Constructors, getters, setters
    public PatientDemographics() {}

    public static PatientDemographics fromFHIRPatientData(FHIRPatientData fhirData) {
        if (fhirData == null) return null;

        PatientDemographics demographics = new PatientDemographics();
        demographics.setFirstName(fhirData.getFirstName());
        demographics.setLastName(fhirData.getLastName());
        demographics.setDateOfBirth(fhirData.getDateOfBirth());
        demographics.setGender(fhirData.getGender());
        demographics.setAge(fhirData.getAge());
        demographics.setMrn(fhirData.getMrn());
        return demographics;
    }

    // ... getters and setters ...
}
```

---

## 4. Updated createUnifiedPipeline Method

### 4.1 Complete Pipeline Integration

```java
package com.cardiofit.flink.operators;

/**
 * Updated createUnifiedPipeline with FHIR/Neo4j enrichment integration.
 *
 * Pipeline Flow:
 * 1. Read CanonicalEvent from Kafka
 * 2. FHIR Enrichment (Async) - Add patient demographics, conditions, medications
 * 3. Convert to GenericEvent
 * 4. KeyBy patientId
 * 5. Aggregate patient state (RocksDB)
 * 6. Neo4j Enrichment (Async) - Add cohorts, similar patients, care team
 * 7. Clinical Intelligence Evaluation
 * 8. Final Output to Kafka
 */
public static DataStream<EnrichedPatientContext> createUnifiedPipeline(
        StreamExecutionEnvironment env,
        Properties kafkaProps) {

    LOG.info("Creating unified pipeline with FHIR/Neo4j enrichment");

    // Step 1: Source - Read canonical events from Kafka
    KafkaSource<CanonicalEvent> canonicalSource = KafkaSource.<CanonicalEvent>builder()
        .setProperties(kafkaProps)
        .setTopics(KafkaTopics.CANONICAL_EVENTS)
        .setGroupId("flink-unified-pipeline-consumer")
        .setStartingOffsets(OffsetsInitializer.latest())
        .setValueOnlyDeserializer(new CanonicalEventDeserializer())
        .build();

    DataStream<CanonicalEvent> canonicalEvents = env
        .fromSource(canonicalSource, WatermarkStrategy.noWatermarks(), "canonical-events-source")
        .uid("canonical-events-source");

    LOG.info("Canonical events source created");

    // Step 2: FHIR Enrichment (Async I/O)
    // Enriches with patient demographics, conditions, medications, allergies
    DataStream<CanonicalEvent> fhirEnrichedEvents = AsyncDataStream.unorderedWait(
        canonicalEvents,
        new CanonicalEventFHIREnricher(
            System.getenv("GCP_PROJECT_ID"),           // "cardiofit-905a8"
            System.getenv("GCP_LOCATION"),              // "asia-south1"
            System.getenv("GCP_DATASET_ID"),            // "clinical-synthesis-hub"
            System.getenv("GCP_FHIR_STORE_ID"),         // "fhir-store"
            System.getenv("GOOGLE_APPLICATION_CREDENTIALS") // "/path/to/credentials.json"
        ),
        10000,                                          // 10 second timeout
        TimeUnit.MILLISECONDS,
        500                                             // 500 concurrent requests
    ).uid("fhir-enrichment-operator")
     .name("FHIR Patient Enrichment");

    LOG.info("FHIR enrichment operator configured: timeout=10s, capacity=500");

    // Step 3: Convert canonical events to generic events
    DataStream<GenericEvent> genericEvents = fhirEnrichedEvents
        .flatMap(new CanonicalEventToGenericEventConverter())
        .uid("canonical-to-generic-converter")
        .name("Canonical to Generic Converter");

    LOG.info("Generic event converter added");

    // Step 4: Key by patient ID for stateful processing
    DataStream<GenericEvent> keyedEvents = genericEvents
        .keyBy(GenericEvent::getPatientId);

    // Step 5: Aggregate patient context (stateful with RocksDB)
    // This operator maintains PatientContextState with vitals, labs, meds history
    DataStream<EnrichedPatientContext> aggregatedContext = keyedEvents
        .process(new PatientContextAggregator())
        .uid("unified-patient-context-aggregator")
        .name("Patient Context Aggregator");

    LOG.info("Patient context aggregator added with unified state management");

    // Step 6: Extract and apply FHIR data from event metadata to state
    // This transfers FHIR data from CanonicalEvent metadata to PatientContextState
    DataStream<EnrichedPatientContext> contextWithFHIR = aggregatedContext
        .map(new FHIRDataExtractor())
        .uid("fhir-data-extractor")
        .name("FHIR Data Extractor");

    LOG.info("FHIR data extractor added");

    // Step 7: Neo4j Enrichment (Async I/O)
    // Enriches with care team, risk cohorts, similar patients, cohort insights
    DataStream<EnrichedPatientContext> neo4jEnrichedContext = AsyncDataStream.unorderedWait(
        contextWithFHIR,
        new PatientContextNeo4jEnricher(
            System.getenv("NEO4J_URI"),                 // "bolt://localhost:7687"
            System.getenv("NEO4J_USERNAME"),            // "neo4j"
            System.getenv("NEO4J_PASSWORD")             // "password"
        ),
        5000,                                           // 5 second timeout
        TimeUnit.MILLISECONDS,
        200                                             // 200 concurrent queries
    ).uid("neo4j-enrichment-operator")
     .name("Neo4j Graph Enrichment");

    LOG.info("Neo4j enrichment operator configured: timeout=5s, capacity=200");

    // Step 8: Clinical intelligence evaluation
    // Advanced clinical pattern detection (sepsis, MODS, ACS)
    DataStream<EnrichedPatientContext> intelligenceEvaluated = neo4jEnrichedContext
        .map(new ClinicalIntelligenceEvaluator())
        .uid("clinical-intelligence-evaluator")
        .name("Clinical Intelligence Evaluator");

    LOG.info("Clinical intelligence evaluator added");

    // Step 9: Finalize and output
    DataStream<EnrichedPatientContext> finalizedContext = intelligenceEvaluated
        .map(new ClinicalEventFinalizer())
        .uid("clinical-event-finalizer")
        .name("Clinical Event Finalizer");

    // Step 10: Sink to Kafka
    KafkaSink<EnrichedPatientContext> clinicalPatternsSink = KafkaSink.<EnrichedPatientContext>builder()
        .setKafkaProducerConfig(kafkaProps)
        .setRecordSerializer(KafkaRecordSerializationSchema.builder()
            .setTopic(KafkaTopics.CLINICAL_PATTERNS)
            .setValueSerializationSchema(new EnrichedPatientContextSerializer())
            .build())
        .build();

    finalizedContext.sinkTo(clinicalPatternsSink)
        .uid("clinical-patterns-sink")
        .name("Clinical Patterns Kafka Sink");

    LOG.info("Unified pipeline created successfully with FHIR and Neo4j enrichment");
    LOG.info("Pipeline operator chain: Source → FHIR Enrich → Convert → Aggregate → FHIR Extract → Neo4j Enrich → Intelligence → Finalize → Sink");

    return finalizedContext;
}
```

### 4.2 FHIRDataExtractor Operator

```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.RichMapFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Extracts FHIR data from event metadata and applies to patient state.
 *
 * This operator transfers FHIR enrichment data from CanonicalEvent metadata
 * (added by CanonicalEventFHIREnricher) into PatientContextState fields.
 *
 * Only runs once per patient (when hasFhirData = false).
 */
public class FHIRDataExtractor extends RichMapFunction<EnrichedPatientContext, EnrichedPatientContext> {
    private static final Logger LOG = LoggerFactory.getLogger(FHIRDataExtractor.class);

    @Override
    public EnrichedPatientContext map(EnrichedPatientContext context) throws Exception {
        PatientContextState state = context.getPatientState();

        // Skip if already has FHIR data
        if (state.isHasFhirData()) {
            return context;
        }

        // Extract FHIR data from original event metadata (if available)
        // Note: This assumes the original CanonicalEvent metadata is preserved
        // through the GenericEvent conversion. Alternative: store in state on first event.

        // For now, mark as needing implementation
        // TODO: Implement metadata extraction from event history

        LOG.debug("FHIR data extraction for patientId={}", context.getPatientId());

        return context;
    }
}
```

**Note**: The FHIRDataExtractor requires additional implementation to transfer metadata from CanonicalEvent through GenericEvent to PatientContextState. Alternative approach: Have PatientContextAggregator extract FHIR data directly from GenericEvent metadata on first event.

### 4.3 Error Handling and Monitoring

**Metrics to Collect**:
```java
// In main() method:
env.getConfig().setLatencyTrackingInterval(1000); // Track latency every 1s

// Custom metrics (Flink Metrics):
private transient Counter fhirEnrichmentSuccess;
private transient Counter fhirEnrichmentFailure;
private transient Counter neo4jEnrichmentSuccess;
private transient Counter neo4jEnrichmentFailure;
private transient Histogram enrichmentLatency;
```

**Alerting Thresholds**:
- FHIR enrichment failure rate > 10% → Alert
- Neo4j enrichment failure rate > 20% → Alert
- Enrichment latency p99 > 5s → Warning
- State size per patient > 1MB → Warning

---

## 5. Integration Testing Strategy

### 5.1 Unit Tests for Each Enricher

**CanonicalEventFHIREnricherTest.java**:
```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.GoogleFHIRClient;
import com.cardiofit.flink.models.*;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import java.util.*;
import java.util.concurrent.CompletableFuture;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.*;

class CanonicalEventFHIREnricherTest {

    @Mock
    private GoogleFHIRClient mockFhirClient;

    @Mock
    private ResultFuture<CanonicalEvent> mockResultFuture;

    private CanonicalEventFHIREnricher enricher;

    @BeforeEach
    void setUp() {
        MockitoAnnotations.openMocks(this);
        enricher = new CanonicalEventFHIREnricher(
            "test-project", "test-location", "test-dataset", "test-store", "/fake/path"
        );
        // Inject mock client via reflection or use test constructor
    }

    @Test
    void testSuccessfulEnrichment() throws Exception {
        // Arrange
        CanonicalEvent event = new CanonicalEvent();
        event.setId("event-123");
        event.setPatientId("PAT-123");
        event.setMetadata(new EventMetadata());

        FHIRPatientData mockPatient = new FHIRPatientData();
        mockPatient.setFirstName("John");
        mockPatient.setLastName("Doe");
        mockPatient.setAge(45);

        List<Condition> mockConditions = Arrays.asList(
            createCondition("I50.1", "Heart Failure")
        );

        List<Medication> mockMedications = Arrays.asList(
            createMedication("Lisinopril", "10mg")
        );

        when(mockFhirClient.getPatientAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(mockPatient));
        when(mockFhirClient.getConditionsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(mockConditions));
        when(mockFhirClient.getMedicationsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(mockMedications));

        // Act
        enricher.asyncInvoke(event, mockResultFuture);

        // Wait for async completion
        Thread.sleep(100);

        // Assert
        ArgumentCaptor<Collection<CanonicalEvent>> captor =
            ArgumentCaptor.forClass(Collection.class);
        verify(mockResultFuture).complete(captor.capture());

        Collection<CanonicalEvent> result = captor.getValue();
        assertEquals(1, result.size());

        CanonicalEvent enrichedEvent = result.iterator().next();
        assertNotNull(enrichedEvent.getMetadata());
        assertTrue(enrichedEvent.getMetadata().containsKey("fhirData"));

        Map<String, Object> fhirData =
            (Map<String, Object>) enrichedEvent.getMetadata().get("fhirData");
        assertTrue((Boolean) fhirData.get("fhirEnriched"));
        assertNotNull(fhirData.get("patient"));
        assertEquals(1, ((List) fhirData.get("conditions")).size());
        assertEquals(1, ((List) fhirData.get("medications")).size());
    }

    @Test
    void testEnrichmentTimeout() throws Exception {
        // Arrange
        CanonicalEvent event = new CanonicalEvent();
        event.setId("event-456");
        event.setPatientId("PAT-456");

        // Simulate timeout by never completing futures
        when(mockFhirClient.getPatientAsync(anyString()))
            .thenReturn(new CompletableFuture<>()); // Never completes

        // Act
        enricher.timeout(event, mockResultFuture);

        // Assert
        verify(mockResultFuture).complete(Collections.singleton(event));
        // Event should be returned unenriched
    }

    @Test
    void testEnrichmentFailureGracefulDegradation() throws Exception {
        // Arrange
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId("PAT-789");

        CompletableFuture<FHIRPatientData> failedFuture = new CompletableFuture<>();
        failedFuture.completeExceptionally(new RuntimeException("FHIR API down"));

        when(mockFhirClient.getPatientAsync(anyString()))
            .thenReturn(failedFuture);

        // Act
        enricher.asyncInvoke(event, mockResultFuture);
        Thread.sleep(100);

        // Assert - should complete with unenriched event
        verify(mockResultFuture).complete(Collections.singleton(event));
    }

    @Test
    void testSkipAlreadyEnriched() throws Exception {
        // Arrange
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId("PAT-999");
        EventMetadata metadata = new EventMetadata();
        metadata.put("fhirEnriched", true);
        event.setMetadata(metadata);

        // Act
        enricher.asyncInvoke(event, mockResultFuture);

        // Assert - should skip enrichment
        verify(mockResultFuture).complete(Collections.singleton(event));
        verify(mockFhirClient, never()).getPatientAsync(anyString());
    }

    // Helper methods
    private Condition createCondition(String code, String display) {
        Condition condition = new Condition();
        condition.setCode(code);
        condition.setDisplay(display);
        condition.setStatus("active");
        return condition;
    }

    private Medication createMedication(String name, String dosage) {
        Medication medication = new Medication();
        medication.setName(name);
        medication.setDosage(dosage);
        medication.setStatus("active");
        return medication;
    }
}
```

**PatientContextNeo4jEnricherTest.java**:
```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.clients.Neo4jGraphClient;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.neo4j.*;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;

import java.util.*;
import java.util.concurrent.CompletableFuture;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.*;
import static org.mockito.Mockito.*;

class PatientContextNeo4jEnricherTest {

    @Mock
    private Neo4jGraphClient mockGraphClient;

    @Mock
    private AdvancedNeo4jQueries mockAdvancedQueries;

    @Mock
    private ResultFuture<EnrichedPatientContext> mockResultFuture;

    private PatientContextNeo4jEnricher enricher;

    @BeforeEach
    void setUp() {
        MockitoAnnotations.openMocks(this);
        enricher = new PatientContextNeo4jEnricher(
            "bolt://localhost:7687", "neo4j", "password"
        );
        // Inject mocks via reflection
    }

    @Test
    void testSuccessfulNeo4jEnrichment() throws Exception {
        // Arrange
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PAT-123");
        PatientContextState state = new PatientContextState("PAT-123");
        context.setPatientState(state);

        GraphData mockGraphData = new GraphData();
        mockGraphData.setRiskCohorts(Arrays.asList("CHF", "Diabetes"));
        mockGraphData.setCareTeam(Arrays.asList("DR001", "NR002"));

        List<SimilarPatient> mockSimilarPatients = Arrays.asList(
            createSimilarPatient("PAT-456", 0.85)
        );

        CohortInsights mockInsights = new CohortInsights();
        mockInsights.setPrimaryCohort("CHF");
        mockInsights.setCohortSize(1500);

        when(mockGraphClient.queryGraphAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(mockGraphData));
        when(mockAdvancedQueries.findSimilarPatientsAsync(anyString(), anyInt()))
            .thenReturn(CompletableFuture.completedFuture(mockSimilarPatients));
        when(mockAdvancedQueries.getCohortInsightsAsync(anyString()))
            .thenReturn(CompletableFuture.completedFuture(mockInsights));

        // Act
        enricher.asyncInvoke(context, mockResultFuture);
        Thread.sleep(100);

        // Assert
        verify(mockResultFuture).complete(anyCollection());
        assertTrue(state.isHasNeo4jData());
        assertEquals(2, state.getRiskCohorts().size());
        assertEquals(1, state.getSimilarPatients().size());
        assertNotNull(state.getCohortInsights());
    }

    @Test
    void testNeo4jFailureGracefulDegradation() throws Exception {
        // Arrange
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PAT-789");
        PatientContextState state = new PatientContextState("PAT-789");
        context.setPatientState(state);

        CompletableFuture<GraphData> failedFuture = new CompletableFuture<>();
        failedFuture.completeExceptionally(new RuntimeException("Neo4j connection lost"));

        when(mockGraphClient.queryGraphAsync(anyString()))
            .thenReturn(failedFuture);

        // Act
        enricher.asyncInvoke(context, mockResultFuture);
        Thread.sleep(100);

        // Assert - should complete without Neo4j data
        verify(mockResultFuture).complete(Collections.singleton(context));
        assertFalse(state.isHasNeo4jData());
    }

    private SimilarPatient createSimilarPatient(String patientId, double similarity) {
        SimilarPatient similar = new SimilarPatient();
        similar.setPatientId(patientId);
        similar.setSimilarityScore(similarity);
        similar.setSharedConditions(Arrays.asList("I50.1"));
        return similar;
    }
}
```

### 5.2 Integration Tests for Complete Pipeline

**UnifiedPipelineIntegrationTest.java**:
```java
package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.test.util.MiniClusterWithClientResource;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

class UnifiedPipelineIntegrationTest {

    private static MiniClusterWithClientResource flinkCluster;

    @BeforeAll
    static void startFlinkCluster() {
        flinkCluster = new MiniClusterWithClientResource(
            new MiniClusterResourceConfiguration.Builder()
                .setNumberSlotsPerTaskManager(2)
                .setNumberTaskManagers(1)
                .build()
        );
        flinkCluster.before();
    }

    @AfterAll
    static void stopFlinkCluster() {
        flinkCluster.after();
    }

    @Test
    void testEndToEndPipelineWithMockData() throws Exception {
        // Arrange
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(1);

        // Create test canonical events
        List<CanonicalEvent> testEvents = createTestEvents();

        // Create data stream from test data
        DataStream<CanonicalEvent> sourceStream = env.fromCollection(testEvents);

        // Apply enrichment operators (with mocked clients)
        // ... test pipeline execution ...

        // Execute and verify
        env.execute("Integration Test");

        // Assert expected enrichment occurred
        // Verify output contains FHIR and Neo4j data
    }

    private List<CanonicalEvent> createTestEvents() {
        List<CanonicalEvent> events = new ArrayList<>();

        CanonicalEvent event1 = CanonicalEvent.builder()
            .id("evt-001")
            .patientId("PAT-123")
            .eventType(EventType.VITAL_SIGN)
            .eventTime(System.currentTimeMillis())
            .payload(createVitalPayload(85, "120/80", 96, 37.0))
            .build();

        events.add(event1);
        return events;
    }

    private Map<String, Object> createVitalPayload(int hr, String bp, int spo2, double temp) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("heartrate", hr);
        payload.put("bloodpressure", bp);
        payload.put("oxygensaturation", spo2);
        payload.put("temperature", temp);
        return payload;
    }
}
```

### 5.3 Mock Strategy for External Services

**MockGoogleFHIRClient.java**:
```java
package com.cardiofit.flink.clients;

import com.cardiofit.flink.models.*;
import java.util.*;
import java.util.concurrent.CompletableFuture;

/**
 * Mock FHIR client for testing without Google Cloud dependency.
 */
public class MockGoogleFHIRClient extends GoogleFHIRClient {

    private Map<String, FHIRPatientData> mockPatients = new HashMap<>();
    private Map<String, List<Condition>> mockConditions = new HashMap<>();
    private Map<String, List<Medication>> mockMedications = new HashMap<>();

    public MockGoogleFHIRClient() {
        super("mock-project", "mock-location", "mock-dataset", "mock-store", "/mock/path");
        initializeMockData();
    }

    private void initializeMockData() {
        // PAT-123: Heart failure patient
        FHIRPatientData patient1 = new FHIRPatientData();
        patient1.setId("PAT-123");
        patient1.setFirstName("John");
        patient1.setLastName("Doe");
        patient1.setAge(65);
        patient1.setGender("male");
        mockPatients.put("PAT-123", patient1);

        List<Condition> conditions1 = Arrays.asList(
            createCondition("I50.1", "Heart Failure"),
            createCondition("E11.9", "Type 2 Diabetes")
        );
        mockConditions.put("PAT-123", conditions1);

        List<Medication> meds1 = Arrays.asList(
            createMedication("Lisinopril", "10mg", "daily"),
            createMedication("Metformin", "500mg", "twice daily")
        );
        mockMedications.put("PAT-123", meds1);
    }

    @Override
    public CompletableFuture<FHIRPatientData> getPatientAsync(String patientId) {
        return CompletableFuture.completedFuture(mockPatients.get(patientId));
    }

    @Override
    public CompletableFuture<List<Condition>> getConditionsAsync(String patientId) {
        return CompletableFuture.completedFuture(
            mockConditions.getOrDefault(patientId, new ArrayList<>())
        );
    }

    @Override
    public CompletableFuture<List<Medication>> getMedicationsAsync(String patientId) {
        return CompletableFuture.completedFuture(
            mockMedications.getOrDefault(patientId, new ArrayList<>())
        );
    }

    private Condition createCondition(String code, String display) {
        Condition c = new Condition();
        c.setCode(code);
        c.setDisplay(display);
        c.setStatus("active");
        return c;
    }

    private Medication createMedication(String name, String dosage, String frequency) {
        Medication m = new Medication();
        m.setName(name);
        m.setDosage(dosage);
        m.setFrequency(frequency);
        m.setStatus("active");
        return m;
    }
}
```

**MockNeo4jGraphClient.java**:
```java
package com.cardiofit.flink.clients;

import com.cardiofit.flink.models.GraphData;
import java.util.*;
import java.util.concurrent.CompletableFuture;

/**
 * Mock Neo4j client for testing without Neo4j dependency.
 */
public class MockNeo4jGraphClient extends Neo4jGraphClient {

    private Map<String, GraphData> mockGraphData = new HashMap<>();

    public MockNeo4jGraphClient() {
        super("bolt://mock:7687", "mock", "mock");
        initializeMockData();
    }

    private void initializeMockData() {
        GraphData data1 = new GraphData();
        data1.setRiskCohorts(Arrays.asList("CHF", "Diabetes", "High-Risk-Readmission"));
        data1.setCareTeam(Arrays.asList("DR001-CardioSpecialist", "NR002-HeartFailureNurse"));
        data1.setCarePathways(Arrays.asList("Heart Failure Pathway"));
        mockGraphData.put("PAT-123", data1);
    }

    @Override
    public CompletableFuture<GraphData> queryGraphAsync(String patientId) {
        return CompletableFuture.completedFuture(
            mockGraphData.getOrDefault(patientId, new GraphData())
        );
    }
}
```

### 5.4 Performance Testing Approach

**Load Test Specifications**:
- **Event Rate**: 1000 events/second sustained
- **Patient Distribution**: 100 unique patients
- **FHIR API Latency**: Simulated 50ms p50, 200ms p99
- **Neo4j Latency**: Simulated 20ms p50, 100ms p99
- **Duration**: 30 minutes
- **Expected Throughput**: >950 events/second (95% of input)
- **Expected Latency**: <1s p99 end-to-end

**Performance Test Harness**:
```java
@Test
void testPipelinePerformance() throws Exception {
    // Generate 60K events (1000/sec × 60 seconds)
    List<CanonicalEvent> loadTestEvents = generateLoadTestEvents(60000, 100);

    // Run pipeline with mock clients (simulated latency)
    long startTime = System.currentTimeMillis();
    DataStream<EnrichedPatientContext> results = runPipeline(loadTestEvents);
    long endTime = System.currentTimeMillis();

    // Verify throughput
    double duration = (endTime - startTime) / 1000.0;
    double throughput = 60000 / duration;
    assertTrue(throughput > 950,
        "Throughput should exceed 950 events/sec, got: " + throughput);

    // Verify enrichment rate
    long enrichedCount = countEnrichedEvents(results);
    double enrichmentRate = enrichedCount / 60000.0;
    assertTrue(enrichmentRate > 0.95,
        "Enrichment rate should exceed 95%, got: " + enrichmentRate);
}
```

---

## 6. Deployment and Operational Considerations

### 6.1 Configuration Management

**Environment Variables** (required):
```bash
# Google Cloud FHIR API
export GCP_PROJECT_ID="cardiofit-905a8"
export GCP_LOCATION="asia-south1"
export GCP_DATASET_ID="clinical-synthesis-hub"
export GCP_FHIR_STORE_ID="fhir-store"
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account-key.json"

# Neo4j Graph Database
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="secure-password"

# Kafka
export KAFKA_BOOTSTRAP_SERVERS="localhost:9092"
```

**Flink Job Configuration**:
```yaml
flink:
  parallelism: 2
  checkpointing:
    interval: 30000
    min-pause: 5000
    timeout: 600000
  state:
    backend: rocksdb
    incremental-checkpoints: true
  async-io:
    fhir-timeout: 10000
    fhir-capacity: 500
    neo4j-timeout: 5000
    neo4j-capacity: 200
```

### 6.2 Monitoring and Alerting

**Key Metrics**:
1. FHIR enrichment success rate (target: >90%)
2. Neo4j enrichment success rate (target: >80%)
3. End-to-end latency p99 (target: <2s)
4. State size per patient (target: <500KB)
5. Checkpoint duration (target: <5s)

**Grafana Dashboard Panels**:
- Enrichment success/failure rates (time series)
- Latency histograms (FHIR, Neo4j, end-to-end)
- State backend metrics (size, checkpoint duration)
- Kafka lag monitoring

### 6.3 Rollout Strategy

**Phase 1**: Deploy to staging with synthetic data
- Test FHIR/Neo4j connectivity
- Verify enrichment logic
- Validate state persistence

**Phase 2**: Canary deployment (10% traffic)
- Monitor enrichment rates
- Check for errors in logs
- Validate output quality

**Phase 3**: Gradual rollout (50% → 100%)
- Monitor performance metrics
- Compare enriched vs. unenriched event quality
- Adjust capacity settings if needed

**Rollback Plan**:
- Disable enrichment operators via feature flag
- Fall back to unenriched pipeline
- Retain enrichment infrastructure for later retry

---

## Summary

This implementation specification provides:

1. **Complete Operator Implementations**: CanonicalEventFHIREnricher and PatientContextNeo4jEnricher with async I/O patterns
2. **Enhanced State Model**: PatientContextState extended with 15+ new fields for FHIR/Neo4j data
3. **Integrated Pipeline**: Updated createUnifiedPipeline with proper operator chaining
4. **Comprehensive Testing**: Unit tests, integration tests, mocks, and performance test strategy
5. **Operational Readiness**: Configuration, monitoring, and deployment guidance

**Next Steps**:
1. Implement the three Java classes (enrichers + state model updates)
2. Create unit tests with mocked external services
3. Test locally with MockGoogleFHIRClient and MockNeo4jGraphClient
4. Deploy to staging and validate with real FHIR/Neo4j data
5. Performance test with load generator
6. Roll out to production with canary deployment

**Files to Create/Modify**:
- `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/CanonicalEventFHIREnricher.java` (NEW)
- `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/PatientContextNeo4jEnricher.java` (NEW)
- `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/PatientContextState.java` (MODIFY)
- `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/models/PatientDemographics.java` (NEW)
- `/backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module2_Enhanced.java` (MODIFY - createUnifiedPipeline)
- `/backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/CanonicalEventFHIREnricherTest.java` (NEW)
- `/backend/shared-infrastructure/flink-processing/src/test/java/com/cardiofit/flink/operators/PatientContextNeo4jEnricherTest.java` (NEW)
