package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.Instant;
import java.util.Map;
import java.util.HashMap;

/**
 * Enriched patient context emitted by PatientContextAggregator.
 *
 * This model represents the complete patient clinical state after processing
 * a single event (vital/lab/med) through the unified aggregator.
 *
 * Contains:
 * - Full patient state (vitals, labs, meds, alerts, risk indicators)
 * - Event metadata (type, timestamp)
 * - Derived clinical scores (NEWS2, qSOFA, combined acuity)
 * - Protocol trigger time for time constraint tracking
 *
 * Flows to:
 * - ClinicalIntelligenceEvaluator for cross-domain reasoning
 * - ClinicalEventFinalizer for canonical output formatting
 * - Neo4j for knowledge graph storage
 * - Elasticsearch for search indexing
 */
public class EnrichedPatientContext implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Patient identifier
     */
    @JsonProperty("patientId")
    private String patientId;

    /**
     * Complete patient state from aggregator
     * Contains vitals, labs, meds, alerts, risk indicators
     */
    @JsonProperty("patientState")
    private PatientContextState patientState;

    /**
     * Triggering event metadata
     */
    @JsonProperty("eventType")
    private String eventType; // "VITAL_SIGN", "LAB_RESULT", "MEDICATION_UPDATE"

    @JsonProperty("eventTime")
    private long eventTime; // When the clinical event occurred (standardized naming)

    /**
     * Processing metadata
     */
    @JsonProperty("processingTime")
    private long processingTime; // When Flink processed the event (standardized naming)

    @JsonProperty("latencyMs")
    private Long latencyMs; // Event-to-processing latency

    /**
     * Protocol trigger time (for time constraint tracking)
     */
    @JsonProperty("triggerTime")
    private Instant triggerTime; // When protocol was triggered (for bundle tracking)

    /**
     * Encounter identifier (for escalation tracking)
     */
    @JsonProperty("encounterId")
    private String encounterId;

    /**
     * FHIR enrichment data from Google Healthcare API
     * Contains: fhir_demographics, fhir_medications, fhir_conditions, fhir_allergies
     */
    @JsonProperty("enrichmentData")
    private Map<String, Object> enrichmentData;

    @JsonProperty("enrichmentVersion")
    private String enrichmentVersion;

    /**
     * Terminology context from KB-7 CDC BroadcastStream
     * Contains: terminology_version, snomed_version, rxnorm_version, loinc_version, graphdb_endpoint
     * Updated via CDC hot-swap when new terminology releases become ACTIVE
     */
    @JsonProperty("terminologyContext")
    private Map<String, Object> terminologyContext;

    /**
     * V4 data tier classification from Module 1b ingestion pipeline.
     * Determines signal fidelity for downstream computations (e.g., MHRI in Module 3).
     * Values: TIER_1_CGM, TIER_2_FINGERSTICK, TIER_3_SMBG, etc.
     * Defaults to TIER_3_SMBG if not set (legacy EHR path doesn't emit data_tier).
     *
     * Propagation path: CanonicalEvent.payload["data_tier"] → VitalsPayload.additionalVitals
     * → PatientContextState.latestVitals → extracted here as first-class field.
     */
    @JsonProperty("dataTier")
    private String dataTier;

    public EnrichedPatientContext() {
        this.processingTime = System.currentTimeMillis();
        this.enrichmentData = new HashMap<>();
        this.enrichmentVersion = "2.0";
    }

    public EnrichedPatientContext(String patientId, PatientContextState patientState) {
        this();
        this.patientId = patientId;
        this.patientState = patientState;
    }

    // Getters and setters

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public PatientContextState getPatientState() {
        return patientState;
    }

    public void setPatientState(PatientContextState patientState) {
        this.patientState = patientState;
    }

    public String getEventType() {
        return eventType;
    }

    public void setEventType(String eventType) {
        this.eventType = eventType;
    }

    public long getEventTime() {
        return eventTime;
    }

    public void setEventTime(long eventTime) {
        this.eventTime = eventTime;
        // Calculate latency if processing time is set
        if (this.processingTime > 0) {
            this.latencyMs = this.processingTime - eventTime;
        }
    }

    public long getProcessingTime() {
        return processingTime;
    }

    public void setProcessingTime(long processingTime) {
        this.processingTime = processingTime;
    }

    public Long getLatencyMs() {
        return latencyMs;
    }

    public void setLatencyMs(Long latencyMs) {
        this.latencyMs = latencyMs;
    }

    public Instant getTriggerTime() {
        return triggerTime;
    }

    public void setTriggerTime(Instant triggerTime) {
        this.triggerTime = triggerTime;
    }

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    public Map<String, Object> getEnrichmentData() {
        return enrichmentData;
    }

    public void setEnrichmentData(Map<String, Object> enrichmentData) {
        this.enrichmentData = enrichmentData;
    }

    public String getEnrichmentVersion() {
        return enrichmentVersion;
    }

    public void setEnrichmentVersion(String enrichmentVersion) {
        this.enrichmentVersion = enrichmentVersion;
    }

    public Map<String, Object> getTerminologyContext() {
        return terminologyContext;
    }

    public void setTerminologyContext(Map<String, Object> terminologyContext) {
        this.terminologyContext = terminologyContext;
    }

    public String getDataTier() {
        return dataTier;
    }

    public void setDataTier(String dataTier) {
        this.dataTier = dataTier;
    }

    /**
     * Convenience method to get risk indicators (internal use only, not serialized)
     */
    @com.fasterxml.jackson.annotation.JsonIgnore
    public RiskIndicators getRiskIndicators() {
        return patientState != null ? patientState.getRiskIndicators() : null;
    }

    /**
     * Convenience method to check if patient has high acuity (internal use only, not serialized)
     */
    @com.fasterxml.jackson.annotation.JsonIgnore
    public boolean isHighAcuity() {
        if (patientState == null || patientState.getCombinedAcuityScore() == null) {
            return false;
        }
        return patientState.getCombinedAcuityScore() > 5.0; // Threshold for high acuity
    }

    /**
     * Convenience method to get alert count (internal use only, not serialized)
     */
    @com.fasterxml.jackson.annotation.JsonIgnore
    public int getAlertCount() {
        return patientState != null && patientState.getActiveAlerts() != null
                ? patientState.getActiveAlerts().size()
                : 0;
    }

    @Override
    public String toString() {
        return "EnrichedPatientContext{" +
                "patientId='" + patientId + '\'' +
                ", eventType='" + eventType + '\'' +
                ", eventCount=" + (patientState != null ? patientState.getEventCount() : 0) +
                ", alertCount=" + getAlertCount() +
                ", acuityScore=" + (patientState != null ? patientState.getCombinedAcuityScore() : null) +
                ", latencyMs=" + latencyMs +
                '}';
    }
}
