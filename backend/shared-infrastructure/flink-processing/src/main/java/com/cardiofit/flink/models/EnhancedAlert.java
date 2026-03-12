package com.cardiofit.flink.models;

import com.cardiofit.flink.ml.explainability.SHAPExplanation;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

/**
 * Enhanced Alert Model
 *
 * Represents a multi-dimensional clinical alert that combines:
 * - CEP pattern-based detection (Module 4)
 * - ML risk prediction (Module 5)
 * - SHAP explainability
 * - Clinical recommendations
 *
 * Alert Sources:
 * - CORRELATED: Both CEP pattern and ML prediction
 * - CEP_ONLY: CEP pattern without ML prediction
 * - ML_ONLY: ML prediction without CEP pattern
 * - CONTRADICTED: CEP and ML disagree (requires review)
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class EnhancedAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    // Identification
    private String alertId;
    private String patientId;
    private long timestamp;

    // Alert classification
    private String alertType;
    private String severity;  // CRITICAL, HIGH, MEDIUM, LOW, INFO
    private String alertSource;  // CORRELATED, CEP_ONLY, ML_ONLY, CONTRADICTED
    private double confidence;

    // Evidence
    private List<String> evidenceSources;
    private PatternEvent cepPattern;
    private MLPrediction mlPrediction;
    private SHAPExplanation shapExplanation;

    // Clinical context
    private List<String> recommendations;
    private String clinicalInterpretation;

    /**
     * Private constructor - use Builder
     */
    private EnhancedAlert(Builder builder) {
        this.alertId = builder.alertId;
        this.patientId = builder.patientId;
        this.timestamp = builder.timestamp;
        this.alertType = builder.alertType;
        this.severity = builder.severity;
        this.alertSource = builder.alertSource;
        this.confidence = builder.confidence;
        this.evidenceSources = builder.evidenceSources;
        this.cepPattern = builder.cepPattern;
        this.mlPrediction = builder.mlPrediction;
        this.shapExplanation = builder.shapExplanation;
        this.recommendations = builder.recommendations;
        this.clinicalInterpretation = builder.clinicalInterpretation;
    }

    // ===== Getters =====

    public String getAlertId() {
        return alertId;
    }

    public String getPatientId() {
        return patientId;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public String getAlertType() {
        return alertType;
    }

    public String getSeverity() {
        return severity;
    }

    public String getAlertSource() {
        return alertSource;
    }

    public double getConfidence() {
        return confidence;
    }

    public List<String> getEvidenceSources() {
        return evidenceSources;
    }

    public PatternEvent getCepPattern() {
        return cepPattern;
    }

    public MLPrediction getMlPrediction() {
        return mlPrediction;
    }

    public SHAPExplanation getShapExplanation() {
        return shapExplanation;
    }

    public List<String> getRecommendations() {
        return recommendations;
    }

    public String getClinicalInterpretation() {
        return clinicalInterpretation;
    }

    /**
     * Check if alert is correlated (both CEP and ML)
     */
    public boolean isCorrelated() {
        return "CORRELATED".equals(alertSource);
    }

    /**
     * Check if alert requires immediate action
     */
    public boolean requiresImmediateAction() {
        return "CRITICAL".equals(severity) || "HIGH".equals(severity);
    }

    /**
     * Get priority score (0-100)
     */
    public int getPriorityScore() {
        int severityScore = getSeverityScore();
        int confidenceScore = (int) (confidence * 20);  // 0-20
        int sourceScore = getSourceScore();  // 0-20

        return Math.min(100, severityScore + confidenceScore + sourceScore);
    }

    private int getSeverityScore() {
        switch (severity.toUpperCase()) {
            case "CRITICAL": return 60;
            case "HIGH": return 45;
            case "MEDIUM": return 30;
            case "LOW": return 15;
            default: return 0;
        }
    }

    private int getSourceScore() {
        switch (alertSource.toUpperCase()) {
            case "CORRELATED": return 20;  // Both CEP and ML
            case "ML_ONLY": return 15;
            case "CEP_ONLY": return 10;
            case "CONTRADICTED": return 5;
            default: return 0;
        }
    }

    @Override
    public String toString() {
        return String.format(
            "EnhancedAlert{id=%s, patient=%s, type=%s, severity=%s, source=%s, confidence=%.2f, priority=%d}",
            alertId, patientId, alertType, severity, alertSource, confidence, getPriorityScore()
        );
    }

    /**
     * Generate detailed clinical report
     */
    public String toDetailedReport() {
        StringBuilder report = new StringBuilder();

        report.append("═══════════════════════════════════════════════════════════════\n");
        report.append("  CLINICAL ALERT - ").append(alertType.toUpperCase()).append("\n");
        report.append("═══════════════════════════════════════════════════════════════\n\n");

        // Alert metadata
        report.append("Alert ID: ").append(alertId).append("\n");
        report.append("Patient ID: ").append(patientId).append("\n");
        report.append("Timestamp: ").append(timestamp).append("\n");
        report.append("Severity: ").append(severity).append("\n");
        report.append("Source: ").append(alertSource).append("\n");
        report.append("Confidence: ").append(String.format("%.1f%%", confidence * 100)).append("\n");
        report.append("Priority Score: ").append(getPriorityScore()).append("/100\n\n");

        // Evidence sources
        report.append("Evidence Sources:\n");
        report.append("───────────────────────────────────────────────────────────────\n");
        for (int i = 0; i < evidenceSources.size(); i++) {
            report.append(String.format("%d. %s\n", i + 1, evidenceSources.get(i)));
        }
        report.append("\n");

        // CEP Pattern details
        if (cepPattern != null) {
            report.append("CEP Pattern Detection:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append("Pattern: ").append(cepPattern.getPatternName()).append("\n");
            report.append("Confidence: ").append(String.format("%.1f%%", cepPattern.getConfidence() * 100)).append("\n");
            report.append("Detected at: ").append(cepPattern.getTimestamp()).append("\n\n");
        }

        // ML Prediction details
        if (mlPrediction != null) {
            report.append("ML Risk Prediction:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append("Model: ").append(mlPrediction.getModelType()).append("\n");
            report.append("Risk Score: ").append(String.format("%.3f", mlPrediction.getPrimaryScore())).append("\n");
            report.append("Model Confidence: ").append(String.format("%.1f%%", mlPrediction.getModelConfidence() * 100)).append("\n\n");
        }

        // SHAP Explanation
        if (shapExplanation != null) {
            report.append("Model Explainability (Top Contributing Factors):\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            List<SHAPExplanation.FeatureContribution> topContributions = shapExplanation.getTopContributions();
            for (int i = 0; i < Math.min(5, topContributions.size()); i++) {
                SHAPExplanation.FeatureContribution fc = topContributions.get(i);
                report.append(String.format(
                    "%d. %s: %.2f %s (impact: %+.3f)\n",
                    i + 1, fc.getFeatureName(), fc.getFeatureValue(), fc.getUnit(), fc.getShapValue()
                ));
            }
            report.append("\n");
        }

        // Clinical recommendations
        if (recommendations != null && !recommendations.isEmpty()) {
            report.append("Clinical Recommendations:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            for (int i = 0; i < recommendations.size(); i++) {
                report.append(String.format("%d. %s\n", i + 1, recommendations.get(i)));
            }
            report.append("\n");
        }

        // Clinical interpretation
        if (clinicalInterpretation != null && !clinicalInterpretation.isEmpty()) {
            report.append("Clinical Interpretation:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append(clinicalInterpretation).append("\n\n");
        }

        report.append("═══════════════════════════════════════════════════════════════\n");

        return report.toString();
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String alertId = UUID.randomUUID().toString();
        private String patientId;
        private long timestamp = System.currentTimeMillis();
        private String alertType;
        private String severity;
        private String alertSource;
        private double confidence;
        private List<String> evidenceSources = new ArrayList<>();
        private PatternEvent cepPattern;
        private MLPrediction mlPrediction;
        private SHAPExplanation shapExplanation;
        private List<String> recommendations = new ArrayList<>();
        private String clinicalInterpretation = "";

        public Builder alertId(String alertId) {
            this.alertId = alertId;
            return this;
        }

        public Builder patientId(String patientId) {
            this.patientId = patientId;
            return this;
        }

        public Builder timestamp(long timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public Builder alertType(String alertType) {
            this.alertType = alertType;
            return this;
        }

        public Builder severity(String severity) {
            this.severity = severity;
            return this;
        }

        public Builder alertSource(String alertSource) {
            this.alertSource = alertSource;
            return this;
        }

        public Builder confidence(double confidence) {
            this.confidence = confidence;
            return this;
        }

        public Builder evidenceSources(List<String> evidenceSources) {
            this.evidenceSources = new ArrayList<>(evidenceSources);
            return this;
        }

        public Builder addEvidenceSource(String evidenceSource) {
            this.evidenceSources.add(evidenceSource);
            return this;
        }

        public Builder cepPattern(PatternEvent cepPattern) {
            this.cepPattern = cepPattern;
            return this;
        }

        public Builder mlPrediction(MLPrediction mlPrediction) {
            this.mlPrediction = mlPrediction;
            return this;
        }

        public Builder shapExplanation(SHAPExplanation shapExplanation) {
            this.shapExplanation = shapExplanation;
            return this;
        }

        public Builder recommendations(List<String> recommendations) {
            this.recommendations = new ArrayList<>(recommendations);
            return this;
        }

        public Builder addRecommendation(String recommendation) {
            this.recommendations.add(recommendation);
            return this;
        }

        public Builder clinicalInterpretation(String clinicalInterpretation) {
            this.clinicalInterpretation = clinicalInterpretation;
            return this;
        }

        public EnhancedAlert build() {
            // Validation
            if (patientId == null || patientId.isEmpty()) {
                throw new IllegalArgumentException("patientId is required");
            }
            if (alertType == null || alertType.isEmpty()) {
                throw new IllegalArgumentException("alertType is required");
            }
            if (severity == null || severity.isEmpty()) {
                throw new IllegalArgumentException("severity is required");
            }
            if (alertSource == null || alertSource.isEmpty()) {
                throw new IllegalArgumentException("alertSource is required");
            }

            return new EnhancedAlert(this);
        }
    }
}
