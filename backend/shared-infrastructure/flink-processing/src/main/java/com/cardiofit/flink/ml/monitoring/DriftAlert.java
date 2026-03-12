package com.cardiofit.flink.ml.monitoring;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;

/**
 * Drift Alert Model
 *
 * Represents a model drift detection alert triggered when statistical drift
 * exceeds configured thresholds. Includes feature drift, prediction drift,
 * and accuracy degradation information.
 *
 * Drift Types:
 * - FEATURE_DRIFT: Input feature distributions have changed (KS test)
 * - PREDICTION_DRIFT: Prediction score distribution has shifted (PSI)
 * - ACCURACY_DRIFT: Model accuracy has degraded (requires ground truth)
 *
 * Severity Levels:
 * - CRITICAL: Severe prediction drift (PSI > 0.25) or accuracy drop > 10%
 * - HIGH: Many features drifting (>10) or moderate prediction drift
 * - MEDIUM: Few features drifting or minor accuracy degradation
 * - LOW: Edge case detection, monitoring recommended
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class DriftAlert implements Serializable {
    private static final long serialVersionUID = 1L;

    // Identification
    private String alertId;
    private String modelType;
    private long timestamp;

    // Drift classification
    private String severity;  // CRITICAL, HIGH, MEDIUM, LOW
    private boolean hasFeatureDrift;
    private boolean hasPredictionDrift;
    private boolean hasAccuracyDrift;

    // Feature drift details
    private List<String> driftedFeatures;
    private int totalFeaturesDrifted;

    // Prediction drift details
    private double predictionDriftPSI;  // Population Stability Index
    private String predictionDriftSeverity;  // NONE, MODERATE, SEVERE

    // Accuracy drift details (if ground truth available)
    private Double baselineAccuracy;
    private Double currentAccuracy;
    private Double accuracyDrop;

    // Actions and recommendations
    private List<String> recommendations;
    private boolean retrainingRequired;
    private String alertMessage;

    /**
     * Private constructor - use Builder
     */
    private DriftAlert(Builder builder) {
        this.alertId = builder.alertId;
        this.modelType = builder.modelType;
        this.timestamp = builder.timestamp;
        this.severity = builder.severity;
        this.hasFeatureDrift = builder.hasFeatureDrift;
        this.hasPredictionDrift = builder.hasPredictionDrift;
        this.hasAccuracyDrift = builder.hasAccuracyDrift;
        this.driftedFeatures = builder.driftedFeatures;
        this.totalFeaturesDrifted = builder.driftedFeatures.size();
        this.predictionDriftPSI = builder.predictionDriftPSI;
        this.predictionDriftSeverity = builder.predictionDriftSeverity;
        this.baselineAccuracy = builder.baselineAccuracy;
        this.currentAccuracy = builder.currentAccuracy;
        this.accuracyDrop = builder.accuracyDrop;
        this.recommendations = builder.recommendations;
        this.retrainingRequired = builder.retrainingRequired;
        this.alertMessage = builder.alertMessage;
    }

    // ===== Getters =====

    public String getAlertId() {
        return alertId;
    }

    public String getModelType() {
        return modelType;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public String getSeverity() {
        return severity;
    }

    public boolean hasFeatureDrift() {
        return hasFeatureDrift;
    }

    public boolean hasPredictionDrift() {
        return hasPredictionDrift;
    }

    public boolean hasAccuracyDrift() {
        return hasAccuracyDrift;
    }

    public List<String> getDriftedFeatures() {
        return driftedFeatures;
    }

    public int getTotalFeaturesDrifted() {
        return totalFeaturesDrifted;
    }

    public double getPredictionDriftPSI() {
        return predictionDriftPSI;
    }

    public String getPredictionDriftSeverity() {
        return predictionDriftSeverity;
    }

    public Double getBaselineAccuracy() {
        return baselineAccuracy;
    }

    public Double getCurrentAccuracy() {
        return currentAccuracy;
    }

    public Double getAccuracyDrop() {
        return accuracyDrop;
    }

    public List<String> getRecommendations() {
        return recommendations;
    }

    public boolean isRetrainingRequired() {
        return retrainingRequired;
    }

    public String getAlertMessage() {
        return alertMessage;
    }

    /**
     * Check if drift is critical (requires immediate action)
     */
    public boolean isCritical() {
        return "CRITICAL".equals(severity);
    }

    /**
     * Check if drift requires human review
     */
    public boolean requiresHumanReview() {
        return isCritical() || (hasAccuracyDrift && accuracyDrop != null && accuracyDrop > 0.05);
    }

    /**
     * Generate detailed drift report
     */
    public String toDetailedReport() {
        StringBuilder report = new StringBuilder();

        report.append("═══════════════════════════════════════════════════════════════\n");
        report.append("  MODEL DRIFT ALERT - ").append(modelType.toUpperCase()).append("\n");
        report.append("═══════════════════════════════════════════════════════════════\n\n");

        // Alert metadata
        report.append(String.format("Alert ID: %s\n", alertId));
        report.append(String.format("Timestamp: %d\n", timestamp));
        report.append(String.format("Severity: %s %s\n\n",
            severity, isCritical() ? "[ACTION REQUIRED]" : ""));

        // Drift summary
        report.append("Drift Summary:\n");
        report.append("───────────────────────────────────────────────────────────────\n");
        report.append(String.format("Feature Drift: %s (%d features affected)\n",
            hasFeatureDrift ? "YES" : "NO", totalFeaturesDrifted));
        report.append(String.format("Prediction Drift: %s (PSI=%.3f, %s)\n",
            hasPredictionDrift ? "YES" : "NO", predictionDriftPSI, predictionDriftSeverity));

        if (hasAccuracyDrift && accuracyDrop != null) {
            report.append(String.format("Accuracy Drift: YES (baseline=%.3f, current=%.3f, drop=%.1f%%)\n",
                baselineAccuracy, currentAccuracy, accuracyDrop * 100));
        } else {
            report.append("Accuracy Drift: NO\n");
        }
        report.append("\n");

        // Feature drift details
        if (hasFeatureDrift && !driftedFeatures.isEmpty()) {
            report.append("Drifted Features:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            int count = Math.min(10, driftedFeatures.size());
            for (int i = 0; i < count; i++) {
                report.append(String.format("%d. %s\n", i + 1, driftedFeatures.get(i)));
            }
            if (driftedFeatures.size() > 10) {
                report.append(String.format("... and %d more features\n", driftedFeatures.size() - 10));
            }
            report.append("\n");
        }

        // Recommendations
        if (!recommendations.isEmpty()) {
            report.append("Recommendations:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            for (int i = 0; i < recommendations.size(); i++) {
                report.append(String.format("%d. %s\n", i + 1, recommendations.get(i)));
            }
            report.append("\n");
        }

        // Action required
        if (retrainingRequired) {
            report.append("⚠️  ACTION REQUIRED: Model retraining is strongly recommended\n\n");
        }

        report.append("═══════════════════════════════════════════════════════════════\n");

        return report.toString();
    }

    /**
     * Export as JSON
     */
    public String toJson() {
        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append(String.format("\"alert_id\":\"%s\",", alertId));
        sb.append(String.format("\"model_type\":\"%s\",", modelType));
        sb.append(String.format("\"timestamp\":%d,", timestamp));
        sb.append(String.format("\"severity\":\"%s\",", severity));
        sb.append(String.format("\"has_feature_drift\":%b,", hasFeatureDrift));
        sb.append(String.format("\"has_prediction_drift\":%b,", hasPredictionDrift));
        sb.append(String.format("\"has_accuracy_drift\":%b,", hasAccuracyDrift));
        sb.append(String.format("\"total_features_drifted\":%d,", totalFeaturesDrifted));
        sb.append(String.format("\"prediction_drift_psi\":%.4f,", predictionDriftPSI));
        sb.append(String.format("\"retraining_required\":%b", retrainingRequired));

        if (accuracyDrop != null) {
            sb.append(String.format(",\"accuracy_drop\":%.4f", accuracyDrop));
        }

        sb.append("}");
        return sb.toString();
    }

    /**
     * Export to Prometheus format
     */
    public String toPrometheusFormat() {
        StringBuilder sb = new StringBuilder();

        sb.append(String.format("ml_drift_detected{model=\"%s\",type=\"feature\"} %d\n",
            modelType, hasFeatureDrift ? 1 : 0));
        sb.append(String.format("ml_drift_detected{model=\"%s\",type=\"prediction\"} %d\n",
            modelType, hasPredictionDrift ? 1 : 0));
        sb.append(String.format("ml_drift_features_count{model=\"%s\"} %d\n",
            modelType, totalFeaturesDrifted));
        sb.append(String.format("ml_drift_psi{model=\"%s\"} %.4f\n",
            modelType, predictionDriftPSI));

        if (accuracyDrop != null) {
            sb.append(String.format("ml_accuracy_drop{model=\"%s\"} %.4f\n",
                modelType, accuracyDrop));
        }

        sb.append(String.format("ml_retraining_required{model=\"%s\"} %d\n",
            modelType, retrainingRequired ? 1 : 0));

        return sb.toString();
    }

    @Override
    public String toString() {
        return String.format(
            "DriftAlert{model=%s, severity=%s, featureDrift=%b (%d features), predictionDrift=%b (PSI=%.3f), retrainingRequired=%b}",
            modelType, severity, hasFeatureDrift, totalFeaturesDrifted,
            hasPredictionDrift, predictionDriftPSI, retrainingRequired
        );
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String alertId = UUID.randomUUID().toString();
        private String modelType;
        private long timestamp = System.currentTimeMillis();
        private String severity = "MEDIUM";
        private boolean hasFeatureDrift = false;
        private boolean hasPredictionDrift = false;
        private boolean hasAccuracyDrift = false;
        private List<String> driftedFeatures = new ArrayList<>();
        private double predictionDriftPSI = 0.0;
        private String predictionDriftSeverity = "NONE";
        private Double baselineAccuracy;
        private Double currentAccuracy;
        private Double accuracyDrop;
        private List<String> recommendations = new ArrayList<>();
        private boolean retrainingRequired = false;
        private String alertMessage = "";

        public Builder alertId(String alertId) {
            this.alertId = alertId;
            return this;
        }

        public Builder modelType(String modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder timestamp(long timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public Builder severity(String severity) {
            this.severity = severity;
            return this;
        }

        public Builder hasFeatureDrift(boolean hasFeatureDrift) {
            this.hasFeatureDrift = hasFeatureDrift;
            return this;
        }

        public Builder hasPredictionDrift(boolean hasPredictionDrift) {
            this.hasPredictionDrift = hasPredictionDrift;
            return this;
        }

        public Builder hasAccuracyDrift(boolean hasAccuracyDrift) {
            this.hasAccuracyDrift = hasAccuracyDrift;
            return this;
        }

        public Builder driftedFeatures(List<String> driftedFeatures) {
            this.driftedFeatures = new ArrayList<>(driftedFeatures);
            return this;
        }

        public Builder addDriftedFeature(String feature) {
            this.driftedFeatures.add(feature);
            return this;
        }

        public Builder predictionDriftPSI(double predictionDriftPSI) {
            this.predictionDriftPSI = predictionDriftPSI;
            return this;
        }

        public Builder predictionDriftSeverity(String predictionDriftSeverity) {
            this.predictionDriftSeverity = predictionDriftSeverity;
            return this;
        }

        public Builder baselineAccuracy(Double baselineAccuracy) {
            this.baselineAccuracy = baselineAccuracy;
            return this;
        }

        public Builder currentAccuracy(Double currentAccuracy) {
            this.currentAccuracy = currentAccuracy;
            return this;
        }

        public Builder accuracyDrop(Double accuracyDrop) {
            this.accuracyDrop = accuracyDrop;
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

        public Builder retrainingRequired(boolean retrainingRequired) {
            this.retrainingRequired = retrainingRequired;
            return this;
        }

        public Builder alertMessage(String alertMessage) {
            this.alertMessage = alertMessage;
            return this;
        }

        public DriftAlert build() {
            if (modelType == null || modelType.isEmpty()) {
                throw new IllegalArgumentException("modelType is required");
            }

            // Auto-determine retraining requirement
            if ("CRITICAL".equals(severity) || "SEVERE".equals(predictionDriftSeverity) ||
                (accuracyDrop != null && accuracyDrop > 0.10)) {
                this.retrainingRequired = true;
            }

            // Generate alert message if not provided
            if (alertMessage == null || alertMessage.isEmpty()) {
                this.alertMessage = generateAlertMessage();
            }

            return new DriftAlert(this);
        }

        private String generateAlertMessage() {
            StringBuilder msg = new StringBuilder();
            msg.append("Model drift detected for ").append(modelType).append(": ");

            List<String> issues = new ArrayList<>();
            if (hasFeatureDrift) {
                issues.add(driftedFeatures.size() + " features drifted");
            }
            if (hasPredictionDrift) {
                issues.add("prediction drift (PSI=" + String.format("%.3f", predictionDriftPSI) + ")");
            }
            if (hasAccuracyDrift && accuracyDrop != null) {
                issues.add("accuracy drop (" + String.format("%.1f%%", accuracyDrop * 100) + ")");
            }

            msg.append(String.join(", ", issues));

            if (retrainingRequired) {
                msg.append(". Model retraining required.");
            }

            return msg.toString();
        }
    }
}
