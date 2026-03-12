package com.cardiofit.stream.state;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;
import java.util.List;

public class PathwayAnalysis implements Serializable {
    private static final long serialVersionUID = 1L;

    private String analysisId;
    private String patientId;
    private String protocolId;
    private LocalDateTime analysisTime;
    private String pathwayStatus;
    private Map<String, Object> metrics;
    private List<String> deviations;
    private double adherenceScore;
    private String recommendation;

    public PathwayAnalysis() {}

    public PathwayAnalysis(String analysisId, String patientId, String protocolId) {
        this.analysisId = analysisId;
        this.patientId = patientId;
        this.protocolId = protocolId;
        this.analysisTime = LocalDateTime.now();
    }

    public String getAnalysisId() { return analysisId; }
    public void setAnalysisId(String analysisId) { this.analysisId = analysisId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getProtocolId() { return protocolId; }
    public void setProtocolId(String protocolId) { this.protocolId = protocolId; }

    public LocalDateTime getAnalysisTime() { return analysisTime; }
    public void setAnalysisTime(LocalDateTime analysisTime) { this.analysisTime = analysisTime; }

    public String getPathwayStatus() { return pathwayStatus; }
    public void setPathwayStatus(String pathwayStatus) { this.pathwayStatus = pathwayStatus; }

    public Map<String, Object> getMetrics() { return metrics; }
    public void setMetrics(Map<String, Object> metrics) { this.metrics = metrics; }

    public List<String> getDeviations() { return deviations; }
    public void setDeviations(List<String> deviations) { this.deviations = deviations; }

    public double getAdherenceScore() { return adherenceScore; }
    public void setAdherenceScore(double adherenceScore) { this.adherenceScore = adherenceScore; }

    public String getRecommendation() { return recommendation; }
    public void setRecommendation(String recommendation) { this.recommendation = recommendation; }

    // Methods required by ClinicalPathwayAdherenceFunction
    public boolean hasDeviation() {
        return deviations != null && !deviations.isEmpty();
    }

    public Severity getSeverity() {
        // Determine severity based on adherence score and number of deviations
        if (adherenceScore < 0.3 || (deviations != null && deviations.size() > 3)) {
            return Severity.CRITICAL;
        } else if (adherenceScore < 0.6 || (deviations != null && deviations.size() > 1)) {
            return Severity.HIGH;
        } else if (adherenceScore < 0.8 || (deviations != null && deviations.size() > 0)) {
            return Severity.MODERATE;
        }
        return Severity.LOW;
    }

    // Additional methods required by ClinicalPathwayAdherenceFunction
    public List<String> getRecommendations() {
        // Return recommendation as a list
        if (recommendation != null) {
            return java.util.Arrays.asList(recommendation.split(","));
        }
        return new java.util.ArrayList<>();
    }

    public void setHasDeviation(boolean hasDeviation) {
        // Update deviations list based on hasDeviation flag
        if (hasDeviation && (deviations == null || deviations.isEmpty())) {
            this.deviations = java.util.Arrays.asList("pathway_deviation_detected");
        } else if (!hasDeviation) {
            this.deviations = new java.util.ArrayList<>();
        }
    }

    public void setSeverity(com.cardiofit.stream.models.Severity severity) {
        // Convert from models.Severity to PathwayAnalysis.Severity
        // This is a compatibility method for the different Severity enums
        if (severity != null) {
            // Store as adherence score based on severity
            switch (severity) {
                case CRITICAL:
                    this.adherenceScore = 0.2;
                    break;
                case HIGH:
                    this.adherenceScore = 0.4;
                    break;
                case MEDIUM:
                    this.adherenceScore = 0.6;
                    break;
                case LOW:
                    this.adherenceScore = 0.8;
                    break;
            }
        }
    }

    public void setRecommendations(List<String> recommendations) {
        // Convert list to comma-separated recommendation string
        if (recommendations != null && !recommendations.isEmpty()) {
            this.recommendation = String.join(",", recommendations);
        } else {
            this.recommendation = null;
        }
    }

    public void setConfidence(double confidence) {
        // Store confidence as adherence score (they represent similar concept)
        this.adherenceScore = confidence;
    }

    public double getConfidence() {
        // Return adherence score as confidence (they represent similar concept)
        return this.adherenceScore;
    }

    // Severity enum for pathway analysis
    public enum Severity {
        LOW, MODERATE, HIGH, CRITICAL
    }
}