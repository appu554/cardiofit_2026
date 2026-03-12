package com.cardiofit.flink.ml.features;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Clinical Feature Vector
 *
 * Container for extracted clinical features ready for ML model inference.
 * Maintains feature ordering and provides conversion to float arrays for ONNX Runtime.
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ClinicalFeatureVector implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private Map<String, Double> features;
    private int featureCount;
    private long extractionTimestamp;
    private double extractionTimeMs;

    // Quality metrics
    private int missingFeatureCount;
    private double featureCompleteness;  // 0.0 to 1.0

    /**
     * Private constructor - use Builder pattern
     */
    private ClinicalFeatureVector(Builder builder) {
        this.patientId = builder.patientId;
        this.features = builder.features;
        this.featureCount = builder.featureCount;
        this.extractionTimestamp = builder.extractionTimestamp;
        this.extractionTimeMs = builder.extractionTimeMs;
        this.missingFeatureCount = builder.missingFeatureCount;
        this.featureCompleteness = builder.featureCompleteness;
    }

    /**
     * Convert feature map to float array for ONNX Runtime
     *
     * Features are ordered by insertion order (LinkedHashMap preservation).
     * This ensures consistent feature ordering for model inference.
     *
     * @return Float array of feature values
     */
    public float[] toFloatArray() {
        if (features == null || features.isEmpty()) {
            return new float[0];
        }

        float[] array = new float[features.size()];
        int index = 0;

        for (Double value : features.values()) {
            array[index++] = value != null ? value.floatValue() : 0.0f;
        }

        return array;
    }

    /**
     * Convert to double array (for non-ONNX ML libraries)
     */
    public double[] toDoubleArray() {
        if (features == null || features.isEmpty()) {
            return new double[0];
        }

        double[] array = new double[features.size()];
        int index = 0;

        for (Double value : features.values()) {
            array[index++] = value != null ? value : 0.0;
        }

        return array;
    }

    /**
     * Get feature value by name
     */
    public Double getFeature(String featureName) {
        return features != null ? features.get(featureName) : null;
    }

    /**
     * Get feature names in order
     */
    public List<String> getFeatureNames() {
        return features != null ? new ArrayList<>(features.keySet()) : new ArrayList<>();
    }

    /**
     * Check if feature vector is complete (all 70 features present)
     */
    public boolean isComplete() {
        return featureCount == 70 && missingFeatureCount == 0;
    }

    /**
     * Check if feature vector meets quality threshold
     */
    public boolean meetsQualityThreshold(double minCompleteness) {
        return featureCompleteness >= minCompleteness;
    }

    // Getters
    public String getPatientId() { return patientId; }
    public Map<String, Double> getFeatures() { return features; }
    public int getFeatureCount() { return featureCount; }
    public long getExtractionTimestamp() { return extractionTimestamp; }
    public double getExtractionTimeMs() { return extractionTimeMs; }
    public int getMissingFeatureCount() { return missingFeatureCount; }
    public double getFeatureCompleteness() { return featureCompleteness; }

    @Override
    public String toString() {
        return "ClinicalFeatureVector{" +
            "patientId='" + patientId + '\'' +
            ", featureCount=" + featureCount +
            ", extractionTimeMs=" + String.format("%.2f", extractionTimeMs) +
            ", completeness=" + String.format("%.1f%%", featureCompleteness * 100) +
            '}';
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String patientId;
        private Map<String, Double> features = new LinkedHashMap<>();
        private int featureCount;
        private long extractionTimestamp = System.currentTimeMillis();
        private double extractionTimeMs;
        private int missingFeatureCount;
        private double featureCompleteness = 1.0;

        public Builder patientId(String patientId) {
            this.patientId = patientId;
            return this;
        }

        public Builder features(Map<String, Double> features) {
            this.features = features;
            calculateQualityMetrics();
            return this;
        }

        public Builder featureCount(int featureCount) {
            this.featureCount = featureCount;
            return this;
        }

        public Builder extractionTimestamp(long extractionTimestamp) {
            this.extractionTimestamp = extractionTimestamp;
            return this;
        }

        public Builder extractionTimeMs(double extractionTimeMs) {
            this.extractionTimeMs = extractionTimeMs;
            return this;
        }

        private void calculateQualityMetrics() {
            if (features == null || features.isEmpty()) {
                this.missingFeatureCount = 70;
                this.featureCompleteness = 0.0;
                return;
            }

            // Count missing features (null or 0.0 values that should be present)
            int missing = 0;
            for (Map.Entry<String, Double> entry : features.entrySet()) {
                if (entry.getValue() == null) {
                    missing++;
                }
            }

            this.missingFeatureCount = missing;
            this.featureCompleteness = features.isEmpty() ? 0.0 :
                (features.size() - missing) / (double) features.size();
        }

        public ClinicalFeatureVector build() {
            if (featureCount == 0 && features != null) {
                featureCount = features.size();
            }
            return new ClinicalFeatureVector(this);
        }
    }
}
