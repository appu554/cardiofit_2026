package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;
import java.util.List;
import java.util.Set;
import java.util.HashSet;
import com.cardiofit.stream.state.PatientContext;

/**
 * EnrichedClinicalEvent represents a clinical event that has been processed
 * through the semantic enrichment pipeline and is ready for hybrid routing.
 */
public class EnrichedClinicalEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    private String eventId;
    private String patientId;
    private String eventType;
    private LocalDateTime timestamp;
    private EventPriority priority;
    private double clinicalSignificance;
    private double confidenceScore;

    // Enrichment data
    private Map<String, Object> semanticEnrichment;
    private List<String> detectedPatterns;
    private Map<String, Double> riskScores;
    private List<String> triggeredAlerts;

    // FHIR compliance
    private String fhirResourceType;
    private Map<String, Object> fhirData;

    // Routing metadata
    private List<String> routingDestinations;
    private Map<String, Object> routingMetadata;

    // Additional fields for complete pipeline
    private String id;
    private String sourceEventType;
    private Map<String, Object> originalPayload;
    private Map<String, Object> enrichedData;
    private List<SemanticEvent.DrugInteraction> drugInteractions;
    private Set<String> clinicalConcepts;
    private List<MLPrediction> mlPredictions;
    private PatientContext patientContext;
    private Set<String> destinations;
    private boolean criticalEvent;
    private boolean highClinicalSignificance;

    public EnrichedClinicalEvent() {}

    public EnrichedClinicalEvent(String eventId, String patientId, String eventType,
                               LocalDateTime timestamp, EventPriority priority) {
        this.eventId = eventId;
        this.patientId = patientId;
        this.eventType = eventType;
        this.timestamp = timestamp;
        this.priority = priority;
    }

    // Getters and Setters
    public String getEventId() { return eventId; }
    public void setEventId(String eventId) { this.eventId = eventId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getEventType() { return eventType; }
    public void setEventType(String eventType) { this.eventType = eventType; }

    public LocalDateTime getTimestamp() { return timestamp; }
    public void setTimestamp(LocalDateTime timestamp) { this.timestamp = timestamp; }

    public EventPriority getPriority() { return priority; }
    public void setPriority(EventPriority priority) { this.priority = priority; }

    public double getClinicalSignificance() { return clinicalSignificance; }
    public void setClinicalSignificance(double clinicalSignificance) {
        this.clinicalSignificance = clinicalSignificance;
    }

    public double getConfidenceScore() { return confidenceScore; }
    public void setConfidenceScore(double confidenceScore) { this.confidenceScore = confidenceScore; }

    public Map<String, Object> getSemanticEnrichment() { return semanticEnrichment; }
    public void setSemanticEnrichment(Map<String, Object> semanticEnrichment) {
        this.semanticEnrichment = semanticEnrichment;
    }

    public List<String> getDetectedPatterns() { return detectedPatterns; }
    public void setDetectedPatterns(List<String> detectedPatterns) { this.detectedPatterns = detectedPatterns; }

    public Map<String, Double> getRiskScores() { return riskScores; }
    public void setRiskScores(Map<String, Double> riskScores) { this.riskScores = riskScores; }

    public List<String> getTriggeredAlerts() { return triggeredAlerts; }
    public void setTriggeredAlerts(List<String> triggeredAlerts) { this.triggeredAlerts = triggeredAlerts; }

    public String getFhirResourceType() { return fhirResourceType; }
    public void setFhirResourceType(String fhirResourceType) { this.fhirResourceType = fhirResourceType; }

    public Map<String, Object> getFhirData() { return fhirData; }
    public void setFhirData(Map<String, Object> fhirData) { this.fhirData = fhirData; }

    public List<String> getRoutingDestinations() { return routingDestinations; }
    public void setRoutingDestinations(List<String> routingDestinations) {
        this.routingDestinations = routingDestinations;
    }

    public Map<String, Object> getRoutingMetadata() { return routingMetadata; }
    public void setRoutingMetadata(Map<String, Object> routingMetadata) {
        this.routingMetadata = routingMetadata;
    }

    // Additional getters and setters
    public String getId() { return id; }
    public void setId(String id) { this.id = id; }

    public String getSourceEventType() { return sourceEventType; }
    public void setSourceEventType(String sourceEventType) { this.sourceEventType = sourceEventType; }

    public Map<String, Object> getOriginalPayload() { return originalPayload; }
    public void setOriginalPayload(Map<String, Object> originalPayload) { this.originalPayload = originalPayload; }

    public Map<String, Object> getEnrichedData() { return enrichedData; }
    public void setEnrichedData(Map<String, Object> enrichedData) { this.enrichedData = enrichedData; }

    public List<SemanticEvent.DrugInteraction> getDrugInteractions() { return drugInteractions; }
    public void setDrugInteractions(List<SemanticEvent.DrugInteraction> drugInteractions) {
        this.drugInteractions = drugInteractions;
    }

    public Set<String> getClinicalConcepts() { return clinicalConcepts; }
    public void setClinicalConcepts(Set<String> clinicalConcepts) {
        this.clinicalConcepts = clinicalConcepts;
    }

    public List<MLPrediction> getMlPredictions() { return mlPredictions; }
    public void setMlPredictions(List<MLPrediction> mlPredictions) {
        this.mlPredictions = mlPredictions;
    }

    public PatientContext getPatientContext() { return patientContext; }
    public void setPatientContext(PatientContext patientContext) {
        this.patientContext = patientContext;
    }

    // Methods required by TransactionalMultiSinkRouter
    public boolean hasPatientRelationshipChanges() {
        return patientContext != null && patientContext.getClinicalState() != null &&
               patientContext.getClinicalState().containsKey("relationship_changes");
    }

    public boolean hasClinicalConceptRelationships() {
        return clinicalConcepts != null && !clinicalConcepts.isEmpty();
    }

    public boolean hasDrugInteractions() {
        return drugInteractions != null && !drugInteractions.isEmpty();
    }

    public Set<String> getDestinations() { return destinations; }
    public void setDestinations(Set<String> destinations) {
        this.destinations = destinations;
    }

    public boolean isCriticalEvent() { return criticalEvent; }
    public void setCriticalEvent(boolean criticalEvent) { this.criticalEvent = criticalEvent; }

    public boolean isHighClinicalSignificance() { return highClinicalSignificance; }
    public void setHighClinicalSignificance(boolean highClinicalSignificance) {
        this.highClinicalSignificance = highClinicalSignificance;
    }

    // Convert LocalDateTime to long for timestamp
    public void setTimestamp(long timestamp) {
        this.timestamp = LocalDateTime.ofEpochSecond(timestamp / 1000, 0, java.time.ZoneOffset.UTC);
    }

    @Override
    public String toString() {
        return "EnrichedClinicalEvent{" +
                "eventId='" + eventId + '\'' +
                ", patientId='" + patientId + '\'' +
                ", eventType='" + eventType + '\'' +
                ", timestamp=" + timestamp +
                ", priority=" + priority +
                ", clinicalSignificance=" + clinicalSignificance +
                ", confidenceScore=" + confidenceScore +
                '}';
    }

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String eventId;
        private String patientId;
        private String eventType;
        private LocalDateTime timestamp;
        private EventPriority priority = EventPriority.MEDIUM;
        private double clinicalSignificance = 0.0;
        private double confidenceScore = 0.0;
        private Map<String, Object> originalPayload;
        private Map<String, Object> enrichedData;
        private String sourceEventType;
        private List<SemanticEvent.DrugInteraction> drugInteractions;
        private Set<String> clinicalConcepts;
        private List<MLPrediction> mlPredictions;
        private PatientContext patientContext;
        private Set<String> destinations;
        private boolean criticalEvent = false;
        private boolean highClinicalSignificance = false;

        public Builder eventId(String eventId) { this.eventId = eventId; return this; }
        public Builder patientId(String patientId) { this.patientId = patientId; return this; }
        public Builder eventType(String eventType) { this.eventType = eventType; return this; }
        public Builder timestamp(LocalDateTime timestamp) { this.timestamp = timestamp; return this; }
        public Builder priority(EventPriority priority) { this.priority = priority; return this; }
        public Builder clinicalSignificance(double clinicalSignificance) {
            this.clinicalSignificance = clinicalSignificance; return this;
        }
        public Builder confidenceScore(double confidenceScore) {
            this.confidenceScore = confidenceScore; return this;
        }
        public Builder originalPayload(Map<String, Object> originalPayload) {
            this.originalPayload = originalPayload; return this;
        }
        public Builder enrichedData(Map<String, Object> enrichedData) {
            this.enrichedData = enrichedData; return this;
        }
        public Builder sourceEventType(String sourceEventType) {
            this.sourceEventType = sourceEventType; return this;
        }
        public Builder drugInteractions(List<SemanticEvent.DrugInteraction> drugInteractions) {
            this.drugInteractions = drugInteractions; return this;
        }
        public Builder clinicalConcepts(Set<String> clinicalConcepts) {
            this.clinicalConcepts = clinicalConcepts; return this;
        }
        public Builder mlPredictions(List<MLPrediction> mlPredictions) {
            this.mlPredictions = mlPredictions; return this;
        }
        public Builder patientContext(PatientContext patientContext) {
            this.patientContext = patientContext; return this;
        }
        public Builder destinations(Set<String> destinations) {
            this.destinations = destinations; return this;
        }
        public Builder criticalEvent(boolean criticalEvent) {
            this.criticalEvent = criticalEvent; return this;
        }
        public Builder highClinicalSignificance(boolean highClinicalSignificance) {
            this.highClinicalSignificance = highClinicalSignificance; return this;
        }

        public EnrichedClinicalEvent build() {
            EnrichedClinicalEvent event = new EnrichedClinicalEvent();
            event.eventId = this.eventId;
            event.patientId = this.patientId;
            event.eventType = this.eventType;
            event.timestamp = this.timestamp;
            event.priority = this.priority;
            event.clinicalSignificance = this.clinicalSignificance;
            event.confidenceScore = this.confidenceScore;
            event.originalPayload = this.originalPayload;
            event.enrichedData = this.enrichedData;
            event.sourceEventType = this.sourceEventType;
            event.drugInteractions = this.drugInteractions;
            event.clinicalConcepts = this.clinicalConcepts;
            event.mlPredictions = this.mlPredictions;
            event.patientContext = this.patientContext;
            event.destinations = this.destinations;
            event.criticalEvent = this.criticalEvent;
            event.highClinicalSignificance = this.highClinicalSignificance;
            return event;
        }
    }
}