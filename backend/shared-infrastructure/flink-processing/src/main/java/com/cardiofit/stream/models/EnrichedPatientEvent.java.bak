package com.cardiofit.stream.models;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Enriched Patient Event Model
 * Contains original patient event plus enrichment data from semantic mesh and clinical context
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class EnrichedPatientEvent {

    @JsonProperty("original_event")
    private PatientEvent originalEvent;

    @JsonProperty("patient_context")
    private PatientContext patientContext;

    @JsonProperty("semantic_enrichment")
    private Map<String, Object> semanticEnrichment;

    @JsonProperty("clinical_insights")
    private List<ClinicalInsight> clinicalInsights;

    @JsonProperty("risk_assessment")
    private RiskAssessment riskAssessment;

    @JsonProperty("detected_patterns")
    private List<DetectedPattern> detectedPatterns;

    @JsonProperty("enrichment_metadata")
    private EnrichmentMetadata enrichmentMetadata;

    @JsonProperty("urgency_level")
    private String urgencyLevel; // NORMAL, HIGH, CRITICAL

    @JsonProperty("recommended_actions")
    private List<String> recommendedActions;

    // Default constructor for Jackson
    public EnrichedPatientEvent() {
        this.clinicalInsights = new ArrayList<>();
        this.detectedPatterns = new ArrayList<>();
        this.recommendedActions = new ArrayList<>();
        this.semanticEnrichment = new HashMap<>();
    }

    // Constructor from original event
    public EnrichedPatientEvent(PatientEvent originalEvent) {
        this();
        this.originalEvent = originalEvent;
        this.urgencyLevel = "NORMAL";
        this.enrichmentMetadata = new EnrichmentMetadata();
    }

    // Getters and setters
    public PatientEvent getOriginalEvent() {
        return originalEvent;
    }

    public void setOriginalEvent(PatientEvent originalEvent) {
        this.originalEvent = originalEvent;
    }

    public PatientContext getPatientContext() {
        return patientContext;
    }

    public void setPatientContext(PatientContext patientContext) {
        this.patientContext = patientContext;
    }

    public Map<String, Object> getSemanticEnrichment() {
        return semanticEnrichment;
    }

    public void setSemanticEnrichment(Map<String, Object> semanticEnrichment) {
        this.semanticEnrichment = semanticEnrichment;
    }

    public List<ClinicalInsight> getClinicalInsights() {
        return clinicalInsights;
    }

    public void setClinicalInsights(List<ClinicalInsight> clinicalInsights) {
        this.clinicalInsights = clinicalInsights;
    }

    public RiskAssessment getRiskAssessment() {
        return riskAssessment;
    }

    public void setRiskAssessment(RiskAssessment riskAssessment) {
        this.riskAssessment = riskAssessment;
    }

    public List<DetectedPattern> getDetectedPatterns() {
        return detectedPatterns;
    }

    public void setDetectedPatterns(List<DetectedPattern> detectedPatterns) {
        this.detectedPatterns = detectedPatterns;
    }

    public EnrichmentMetadata getEnrichmentMetadata() {
        return enrichmentMetadata;
    }

    public void setEnrichmentMetadata(EnrichmentMetadata enrichmentMetadata) {
        this.enrichmentMetadata = enrichmentMetadata;
    }

    public String getUrgencyLevel() {
        return urgencyLevel;
    }

    public void setUrgencyLevel(String urgencyLevel) {
        this.urgencyLevel = urgencyLevel;
    }

    public List<String> getRecommendedActions() {
        return recommendedActions;
    }

    public void setRecommendedActions(List<String> recommendedActions) {
        this.recommendedActions = recommendedActions;
    }

    // Helper methods
    public void addClinicalInsight(ClinicalInsight insight) {
        this.clinicalInsights.add(insight);
    }

    public void addDetectedPattern(DetectedPattern pattern) {
        this.detectedPatterns.add(pattern);
        // Auto-escalate urgency based on patterns
        if ("CRITICAL".equals(pattern.getSeverity()) && !"CRITICAL".equals(this.urgencyLevel)) {
            this.urgencyLevel = "CRITICAL";
        } else if ("HIGH".equals(pattern.getSeverity()) && "NORMAL".equals(this.urgencyLevel)) {
            this.urgencyLevel = "HIGH";
        }
    }

    public void addRecommendedAction(String action) {
        this.recommendedActions.add(action);
    }

    public boolean requiresImmediateAttention() {
        return "CRITICAL".equals(urgencyLevel) ||
               detectedPatterns.stream().anyMatch(p -> "CRITICAL".equals(p.getSeverity())) ||
               originalEvent.isCritical();
    }

    public boolean requiresPushNotification() {
        return "CRITICAL".equals(urgencyLevel) ||
               "HIGH".equals(urgencyLevel);
    }

    /**
     * Patient Context nested class
     */
    public static class PatientContext {
        @JsonProperty("patient_id")
        private String patientId;

        @JsonProperty("demographics")
        private Map<String, Object> demographics;

        @JsonProperty("active_medications")
        private List<Map<String, Object>> activeMedications;

        @JsonProperty("medical_conditions")
        private List<Map<String, Object>> medicalConditions;

        @JsonProperty("allergies")
        private List<Map<String, Object>> allergies;

        @JsonProperty("recent_vitals")
        private Map<String, Object> recentVitals;

        @JsonProperty("last_updated")
        @JsonFormat(pattern = "yyyy-MM-dd'T'HH:mm:ss")
        private LocalDateTime lastUpdated;

        // Constructors
        public PatientContext() {
            this.demographics = new HashMap<>();
            this.activeMedications = new ArrayList<>();
            this.medicalConditions = new ArrayList<>();
            this.allergies = new ArrayList<>();
            this.recentVitals = new HashMap<>();
        }

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public Map<String, Object> getDemographics() { return demographics; }
        public void setDemographics(Map<String, Object> demographics) { this.demographics = demographics; }

        public List<Map<String, Object>> getActiveMedications() { return activeMedications; }
        public void setActiveMedications(List<Map<String, Object>> activeMedications) { this.activeMedications = activeMedications; }

        public List<Map<String, Object>> getMedicalConditions() { return medicalConditions; }
        public void setMedicalConditions(List<Map<String, Object>> medicalConditions) { this.medicalConditions = medicalConditions; }

        public List<Map<String, Object>> getAllergies() { return allergies; }
        public void setAllergies(List<Map<String, Object>> allergies) { this.allergies = allergies; }

        public Map<String, Object> getRecentVitals() { return recentVitals; }
        public void setRecentVitals(Map<String, Object> recentVitals) { this.recentVitals = recentVitals; }

        public LocalDateTime getLastUpdated() { return lastUpdated; }
        public void setLastUpdated(LocalDateTime lastUpdated) { this.lastUpdated = lastUpdated; }
    }

    /**
     * Clinical Insight nested class
     */
    public static class ClinicalInsight {
        @JsonProperty("insight_type")
        private String insightType;

        @JsonProperty("description")
        private String description;

        @JsonProperty("confidence")
        private Double confidence;

        @JsonProperty("source")
        private String source;

        @JsonProperty("evidence")
        private Map<String, Object> evidence;

        public ClinicalInsight() {
            this.evidence = new HashMap<>();
        }

        public ClinicalInsight(String insightType, String description, Double confidence, String source) {
            this();
            this.insightType = insightType;
            this.description = description;
            this.confidence = confidence;
            this.source = source;
        }

        // Getters and setters
        public String getInsightType() { return insightType; }
        public void setInsightType(String insightType) { this.insightType = insightType; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public Double getConfidence() { return confidence; }
        public void setConfidence(Double confidence) { this.confidence = confidence; }

        public String getSource() { return source; }
        public void setSource(String source) { this.source = source; }

        public Map<String, Object> getEvidence() { return evidence; }
        public void setEvidence(Map<String, Object> evidence) { this.evidence = evidence; }
    }

    /**
     * Risk Assessment nested class
     */
    public static class RiskAssessment {
        @JsonProperty("overall_risk_score")
        private Double overallRiskScore;

        @JsonProperty("risk_factors")
        private List<RiskFactor> riskFactors;

        @JsonProperty("risk_category")
        private String riskCategory; // LOW, MEDIUM, HIGH, CRITICAL

        public RiskAssessment() {
            this.riskFactors = new ArrayList<>();
            this.overallRiskScore = 0.0;
            this.riskCategory = "LOW";
        }

        // Getters and setters
        public Double getOverallRiskScore() { return overallRiskScore; }
        public void setOverallRiskScore(Double overallRiskScore) { this.overallRiskScore = overallRiskScore; }

        public List<RiskFactor> getRiskFactors() { return riskFactors; }
        public void setRiskFactors(List<RiskFactor> riskFactors) { this.riskFactors = riskFactors; }

        public String getRiskCategory() { return riskCategory; }
        public void setRiskCategory(String riskCategory) { this.riskCategory = riskCategory; }

        public static class RiskFactor {
            @JsonProperty("factor_type")
            private String factorType;

            @JsonProperty("description")
            private String description;

            @JsonProperty("risk_score")
            private Double riskScore;

            public RiskFactor() {}

            public RiskFactor(String factorType, String description, Double riskScore) {
                this.factorType = factorType;
                this.description = description;
                this.riskScore = riskScore;
            }

            // Getters and setters
            public String getFactorType() { return factorType; }
            public void setFactorType(String factorType) { this.factorType = factorType; }

            public String getDescription() { return description; }
            public void setDescription(String description) { this.description = description; }

            public Double getRiskScore() { return riskScore; }
            public void setRiskScore(Double riskScore) { this.riskScore = riskScore; }
        }
    }

    /**
     * Detected Pattern nested class
     */
    public static class DetectedPattern {
        @JsonProperty("pattern_type")
        private String patternType;

        @JsonProperty("pattern_name")
        private String patternName;

        @JsonProperty("description")
        private String description;

        @JsonProperty("severity")
        private String severity;

        @JsonProperty("confidence")
        private Double confidence;

        @JsonProperty("time_window")
        private String timeWindow;

        public DetectedPattern() {}

        public DetectedPattern(String patternType, String patternName, String description, String severity, Double confidence) {
            this.patternType = patternType;
            this.patternName = patternName;
            this.description = description;
            this.severity = severity;
            this.confidence = confidence;
        }

        // Getters and setters
        public String getPatternType() { return patternType; }
        public void setPatternType(String patternType) { this.patternType = patternType; }

        public String getPatternName() { return patternName; }
        public void setPatternName(String patternName) { this.patternName = patternName; }

        public String getDescription() { return description; }
        public void setDescription(String description) { this.description = description; }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public Double getConfidence() { return confidence; }
        public void setConfidence(Double confidence) { this.confidence = confidence; }

        public String getTimeWindow() { return timeWindow; }
        public void setTimeWindow(String timeWindow) { this.timeWindow = timeWindow; }
    }

    /**
     * Enrichment Metadata nested class
     */
    public static class EnrichmentMetadata {
        @JsonProperty("enrichment_timestamp")
        @JsonFormat(pattern = "yyyy-MM-dd'T'HH:mm:ss")
        private LocalDateTime enrichmentTimestamp;

        @JsonProperty("processing_duration_ms")
        private Long processingDurationMs;

        @JsonProperty("enrichment_sources")
        private List<String> enrichmentSources;

        @JsonProperty("semantic_queries_executed")
        private Integer semanticQueriesExecuted;

        @JsonProperty("cache_hits")
        private Integer cacheHits;

        @JsonProperty("cache_misses")
        private Integer cacheMisses;

        public EnrichmentMetadata() {
            this.enrichmentTimestamp = LocalDateTime.now();
            this.enrichmentSources = new ArrayList<>();
            this.semanticQueriesExecuted = 0;
            this.cacheHits = 0;
            this.cacheMisses = 0;
        }

        // Getters and setters
        public LocalDateTime getEnrichmentTimestamp() { return enrichmentTimestamp; }
        public void setEnrichmentTimestamp(LocalDateTime enrichmentTimestamp) { this.enrichmentTimestamp = enrichmentTimestamp; }

        public Long getProcessingDurationMs() { return processingDurationMs; }
        public void setProcessingDurationMs(Long processingDurationMs) { this.processingDurationMs = processingDurationMs; }

        public List<String> getEnrichmentSources() { return enrichmentSources; }
        public void setEnrichmentSources(List<String> enrichmentSources) { this.enrichmentSources = enrichmentSources; }

        public Integer getSemanticQueriesExecuted() { return semanticQueriesExecuted; }
        public void setSemanticQueriesExecuted(Integer semanticQueriesExecuted) { this.semanticQueriesExecuted = semanticQueriesExecuted; }

        public Integer getCacheHits() { return cacheHits; }
        public void setCacheHits(Integer cacheHits) { this.cacheHits = cacheHits; }

        public Integer getCacheMisses() { return cacheMisses; }
        public void setCacheMisses(Integer cacheMisses) { this.cacheMisses = cacheMisses; }
    }

    @Override
    public String toString() {
        return "EnrichedPatientEvent{" +
               "originalEvent=" + originalEvent +
               ", urgencyLevel='" + urgencyLevel + '\'' +
               ", clinicalInsightsCount=" + clinicalInsights.size() +
               ", detectedPatternsCount=" + detectedPatterns.size() +
               ", recommendedActionsCount=" + recommendedActions.size() +
               '}';
    }
}