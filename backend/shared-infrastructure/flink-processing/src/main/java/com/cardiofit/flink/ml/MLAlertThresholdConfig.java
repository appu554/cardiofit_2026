package com.cardiofit.flink.ml;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * ML Alert Threshold Configuration
 *
 * Configures thresholds for generating clinical alerts from ML predictions.
 * Each model type (sepsis_risk, deterioration_risk, etc.) has its own
 * threshold configuration with severity levels, hysteresis, and suppression.
 *
 * Example Usage:
 * <pre>
 * MLAlertThresholdConfig config = MLAlertThresholdConfig.builder()
 *     // Sepsis risk thresholds
 *     .addThreshold("sepsis_risk", AlertThreshold.builder()
 *         .criticalThreshold(0.85)
 *         .highThreshold(0.70)
 *         .mediumThreshold(0.50)
 *         .lowThreshold(0.30)
 *         .hysteresis(0.05)
 *         .minConfidence(0.80)
 *         .suppressionWindowMs(300_000)  // 5 minutes
 *         .build())
 *
 *     // Respiratory failure thresholds
 *     .addThreshold("respiratory_failure", AlertThreshold.builder()
 *         .criticalThreshold(0.80)
 *         .highThreshold(0.65)
 *         .mediumThreshold(0.45)
 *         .lowThreshold(0.25)
 *         .build())
 *
 *     .build();
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class MLAlertThresholdConfig implements Serializable {
    private static final long serialVersionUID = 1L;

    private final Map<String, AlertThreshold> thresholds;

    private MLAlertThresholdConfig(Builder builder) {
        this.thresholds = new HashMap<>(builder.thresholds);
    }

    /**
     * Get threshold configuration for model type
     */
    public AlertThreshold getThreshold(String modelType) {
        return thresholds.get(modelType);
    }

    /**
     * Get all configured thresholds
     */
    public Map<String, AlertThreshold> getThresholds() {
        return new HashMap<>(thresholds);
    }

    /**
     * Check if model type has threshold configuration
     */
    public boolean hasThreshold(String modelType) {
        return thresholds.containsKey(modelType);
    }

    /**
     * Create default configuration for common clinical models
     */
    public static MLAlertThresholdConfig createDefault() {
        return builder()
            // Sepsis Risk
            .addThreshold("sepsis_risk", AlertThreshold.builder()
                .criticalThreshold(0.85)
                .highThreshold(0.70)
                .mediumThreshold(0.50)
                .lowThreshold(0.30)
                .hysteresis(0.05)
                .minConfidence(0.75)
                .suppressionWindowMs(300_000)  // 5 minutes
                .build())

            // Clinical Deterioration
            .addThreshold("deterioration_risk", AlertThreshold.builder()
                .criticalThreshold(0.80)
                .highThreshold(0.65)
                .mediumThreshold(0.45)
                .lowThreshold(0.25)
                .hysteresis(0.05)
                .minConfidence(0.70)
                .suppressionWindowMs(300_000)
                .build())

            // Respiratory Failure
            .addThreshold("respiratory_failure", AlertThreshold.builder()
                .criticalThreshold(0.80)
                .highThreshold(0.65)
                .mediumThreshold(0.45)
                .lowThreshold(0.25)
                .hysteresis(0.05)
                .minConfidence(0.75)
                .suppressionWindowMs(180_000)  // 3 minutes (more urgent)
                .build())

            // Cardiac Event
            .addThreshold("cardiac_event", AlertThreshold.builder()
                .criticalThreshold(0.85)
                .highThreshold(0.70)
                .mediumThreshold(0.50)
                .lowThreshold(0.30)
                .hysteresis(0.05)
                .minConfidence(0.80)
                .suppressionWindowMs(180_000)  // 3 minutes
                .build())

            // Medication Adverse Event
            .addThreshold("medication_adverse_event", AlertThreshold.builder()
                .criticalThreshold(0.80)
                .highThreshold(0.65)
                .mediumThreshold(0.45)
                .lowThreshold(0.25)
                .hysteresis(0.05)
                .minConfidence(0.70)
                .suppressionWindowMs(600_000)  // 10 minutes (less urgent)
                .build())

            .build();
    }

    /**
     * Create ICU configuration (stricter thresholds)
     */
    public static MLAlertThresholdConfig createICU() {
        return builder()
            .addThreshold("sepsis_risk", AlertThreshold.builder()
                .criticalThreshold(0.75)  // Lower threshold for ICU
                .highThreshold(0.60)
                .mediumThreshold(0.40)
                .lowThreshold(0.20)
                .hysteresis(0.05)
                .minConfidence(0.70)
                .suppressionWindowMs(180_000)  // 3 minutes (more frequent)
                .build())

            .addThreshold("deterioration_risk", AlertThreshold.builder()
                .criticalThreshold(0.70)
                .highThreshold(0.55)
                .mediumThreshold(0.35)
                .lowThreshold(0.20)
                .hysteresis(0.05)
                .minConfidence(0.65)
                .suppressionWindowMs(180_000)
                .build())

            .addThreshold("respiratory_failure", AlertThreshold.builder()
                .criticalThreshold(0.70)
                .highThreshold(0.55)
                .mediumThreshold(0.35)
                .lowThreshold(0.20)
                .hysteresis(0.05)
                .minConfidence(0.70)
                .suppressionWindowMs(120_000)  // 2 minutes (very urgent)
                .build())

            .addThreshold("cardiac_event", AlertThreshold.builder()
                .criticalThreshold(0.75)
                .highThreshold(0.60)
                .mediumThreshold(0.40)
                .lowThreshold(0.20)
                .hysteresis(0.05)
                .minConfidence(0.75)
                .suppressionWindowMs(120_000)
                .build())

            .build();
    }

    @Override
    public String toString() {
        return String.format("MLAlertThresholdConfig{models=%d}", thresholds.size());
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private Map<String, AlertThreshold> thresholds = new HashMap<>();

        public Builder addThreshold(String modelType, AlertThreshold threshold) {
            this.thresholds.put(modelType, threshold);
            return this;
        }

        public Builder addThresholds(Map<String, AlertThreshold> thresholds) {
            this.thresholds.putAll(thresholds);
            return this;
        }

        public MLAlertThresholdConfig build() {
            if (thresholds.isEmpty()) {
                throw new IllegalArgumentException("At least one threshold must be configured");
            }
            return new MLAlertThresholdConfig(this);
        }
    }
}

/**
 * Alert Threshold Configuration
 *
 * Defines thresholds and behavior for a single ML model type.
 */
class AlertThreshold implements Serializable {
    private static final long serialVersionUID = 1L;

    // Severity thresholds
    private final double criticalThreshold;
    private final double highThreshold;
    private final double mediumThreshold;
    private final double lowThreshold;

    // Hysteresis (prevent flapping)
    private final double hysteresis;

    // Confidence requirements
    private final double minConfidence;

    // Suppression (prevent alert fatigue)
    private final long suppressionWindowMs;

    private AlertThreshold(Builder builder) {
        this.criticalThreshold = builder.criticalThreshold;
        this.highThreshold = builder.highThreshold;
        this.mediumThreshold = builder.mediumThreshold;
        this.lowThreshold = builder.lowThreshold;
        this.hysteresis = builder.hysteresis;
        this.minConfidence = builder.minConfidence;
        this.suppressionWindowMs = builder.suppressionWindowMs;
    }

    // ===== Getters =====

    public double getCriticalThreshold() {
        return criticalThreshold;
    }

    public double getHighThreshold() {
        return highThreshold;
    }

    public double getMediumThreshold() {
        return mediumThreshold;
    }

    public double getLowThreshold() {
        return lowThreshold;
    }

    public double getHysteresis() {
        return hysteresis;
    }

    public double getMinConfidence() {
        return minConfidence;
    }

    public long getSuppressionWindowMs() {
        return suppressionWindowMs;
    }

    /**
     * Get threshold for severity level
     */
    public double getThresholdForSeverity(String severity) {
        switch (severity.toUpperCase()) {
            case "CRITICAL": return criticalThreshold;
            case "HIGH": return highThreshold;
            case "MEDIUM": return mediumThreshold;
            case "LOW": return lowThreshold;
            default: return 0.0;
        }
    }

    /**
     * Check if score meets severity threshold
     */
    public boolean meetsThreshold(double score, String severity) {
        return score >= getThresholdForSeverity(severity);
    }

    @Override
    public String toString() {
        return String.format(
            "AlertThreshold{critical=%.2f, high=%.2f, medium=%.2f, low=%.2f, hysteresis=%.2f, minConf=%.2f, suppress=%dms}",
            criticalThreshold, highThreshold, mediumThreshold, lowThreshold,
            hysteresis, minConfidence, suppressionWindowMs
        );
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private double criticalThreshold = 0.85;
        private double highThreshold = 0.70;
        private double mediumThreshold = 0.50;
        private double lowThreshold = 0.30;
        private double hysteresis = 0.05;
        private double minConfidence = 0.70;
        private long suppressionWindowMs = 300_000;  // 5 minutes

        public Builder criticalThreshold(double criticalThreshold) {
            this.criticalThreshold = criticalThreshold;
            return this;
        }

        public Builder highThreshold(double highThreshold) {
            this.highThreshold = highThreshold;
            return this;
        }

        public Builder mediumThreshold(double mediumThreshold) {
            this.mediumThreshold = mediumThreshold;
            return this;
        }

        public Builder lowThreshold(double lowThreshold) {
            this.lowThreshold = lowThreshold;
            return this;
        }

        public Builder hysteresis(double hysteresis) {
            this.hysteresis = hysteresis;
            return this;
        }

        public Builder minConfidence(double minConfidence) {
            this.minConfidence = minConfidence;
            return this;
        }

        public Builder suppressionWindowMs(long suppressionWindowMs) {
            this.suppressionWindowMs = suppressionWindowMs;
            return this;
        }

        public AlertThreshold build() {
            // Validation
            if (criticalThreshold <= highThreshold) {
                throw new IllegalArgumentException("Critical threshold must be higher than high threshold");
            }
            if (highThreshold <= mediumThreshold) {
                throw new IllegalArgumentException("High threshold must be higher than medium threshold");
            }
            if (mediumThreshold <= lowThreshold) {
                throw new IllegalArgumentException("Medium threshold must be higher than low threshold");
            }
            if (hysteresis < 0 || hysteresis > 0.2) {
                throw new IllegalArgumentException("Hysteresis must be between 0 and 0.2");
            }
            if (minConfidence < 0 || minConfidence > 1) {
                throw new IllegalArgumentException("Min confidence must be between 0 and 1");
            }
            if (suppressionWindowMs < 0) {
                throw new IllegalArgumentException("Suppression window must be non-negative");
            }

            return new AlertThreshold(this);
        }
    }
}
