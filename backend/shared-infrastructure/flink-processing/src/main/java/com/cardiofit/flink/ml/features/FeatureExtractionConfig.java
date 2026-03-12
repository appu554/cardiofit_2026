package com.cardiofit.flink.ml.features;

import java.io.Serializable;

/**
 * Feature Extraction Configuration
 *
 * Controls behavior of clinical feature extraction pipeline including:
 * - Feature selection and filtering
 * - Missing value handling
 * - Quality thresholds
 * - Extraction preferences
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class FeatureExtractionConfig implements Serializable {
    private static final long serialVersionUID = 1L;

    // Feature selection
    private boolean includeDemographics;
    private boolean includeVitals;
    private boolean includeLabs;
    private boolean includeClinicalScores;
    private boolean includeTemporal;
    private boolean includeMedications;
    private boolean includeComorbidities;
    private boolean includeCEPPatterns;

    // Missing value handling
    private boolean useDefaultsForMissingValues;
    private double missingValueDefault;

    // Quality control
    private double minimumFeatureCompleteness;  // 0.0 to 1.0
    private boolean failOnLowQuality;

    // Extraction preferences
    private boolean preferRecentData;
    private long maxDataAgeHours;

    /**
     * Private constructor - use Builder or factory methods
     */
    private FeatureExtractionConfig(Builder builder) {
        this.includeDemographics = builder.includeDemographics;
        this.includeVitals = builder.includeVitals;
        this.includeLabs = builder.includeLabs;
        this.includeClinicalScores = builder.includeClinicalScores;
        this.includeTemporal = builder.includeTemporal;
        this.includeMedications = builder.includeMedications;
        this.includeComorbidities = builder.includeComorbidities;
        this.includeCEPPatterns = builder.includeCEPPatterns;
        this.useDefaultsForMissingValues = builder.useDefaultsForMissingValues;
        this.missingValueDefault = builder.missingValueDefault;
        this.minimumFeatureCompleteness = builder.minimumFeatureCompleteness;
        this.failOnLowQuality = builder.failOnLowQuality;
        this.preferRecentData = builder.preferRecentData;
        this.maxDataAgeHours = builder.maxDataAgeHours;
    }

    // Getters
    public boolean isIncludeDemographics() { return includeDemographics; }
    public boolean isIncludeVitals() { return includeVitals; }
    public boolean isIncludeLabs() { return includeLabs; }
    public boolean isIncludeClinicalScores() { return includeClinicalScores; }
    public boolean isIncludeTemporal() { return includeTemporal; }
    public boolean isIncludeMedications() { return includeMedications; }
    public boolean isIncludeComorbidities() { return includeComorbidities; }
    public boolean isIncludeCEPPatterns() { return includeCEPPatterns; }
    public boolean isUseDefaultsForMissingValues() { return useDefaultsForMissingValues; }
    public double getMissingValueDefault() { return missingValueDefault; }
    public double getMinimumFeatureCompleteness() { return minimumFeatureCompleteness; }
    public boolean isFailOnLowQuality() { return failOnLowQuality; }
    public boolean isPreferRecentData() { return preferRecentData; }
    public long getMaxDataAgeHours() { return maxDataAgeHours; }

    /**
     * Create default configuration (all features enabled)
     */
    public static FeatureExtractionConfig createDefault() {
        return builder()
            .includeDemographics(true)
            .includeVitals(true)
            .includeLabs(true)
            .includeClinicalScores(true)
            .includeTemporal(true)
            .includeMedications(true)
            .includeComorbidities(true)
            .includeCEPPatterns(true)
            .useDefaultsForMissingValues(true)
            .missingValueDefault(0.0)
            .minimumFeatureCompleteness(0.7)
            .failOnLowQuality(false)
            .preferRecentData(true)
            .maxDataAgeHours(24)
            .build();
    }

    /**
     * Create configuration for ICU patients (strict quality requirements)
     */
    public static FeatureExtractionConfig createICU() {
        return builder()
            .includeDemographics(true)
            .includeVitals(true)
            .includeLabs(true)
            .includeClinicalScores(true)
            .includeTemporal(true)
            .includeMedications(true)
            .includeComorbidities(true)
            .includeCEPPatterns(true)
            .useDefaultsForMissingValues(true)
            .missingValueDefault(0.0)
            .minimumFeatureCompleteness(0.9)  // Higher quality required
            .failOnLowQuality(true)
            .preferRecentData(true)
            .maxDataAgeHours(6)  // More recent data required
            .build();
    }

    /**
     * Create configuration for sepsis prediction (specific features)
     */
    public static FeatureExtractionConfig createSepsisPredictor() {
        return builder()
            .includeDemographics(true)
            .includeVitals(true)  // Critical for sepsis
            .includeLabs(true)  // Lactate is key
            .includeClinicalScores(true)  // qSOFA, SOFA
            .includeTemporal(true)
            .includeMedications(true)  // Antibiotics, vasopressors
            .includeComorbidities(true)
            .includeCEPPatterns(true)  // CEP sepsis pattern
            .useDefaultsForMissingValues(true)
            .missingValueDefault(0.0)
            .minimumFeatureCompleteness(0.8)
            .failOnLowQuality(false)
            .preferRecentData(true)
            .maxDataAgeHours(12)
            .build();
    }

    @Override
    public String toString() {
        return "FeatureExtractionConfig{" +
            "allFeaturesEnabled=" + areAllFeaturesEnabled() +
            ", minCompleteness=" + minimumFeatureCompleteness +
            ", maxDataAge=" + maxDataAgeHours + "h" +
            '}';
    }

    private boolean areAllFeaturesEnabled() {
        return includeDemographics && includeVitals && includeLabs &&
               includeClinicalScores && includeTemporal && includeMedications &&
               includeComorbidities && includeCEPPatterns;
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private boolean includeDemographics = true;
        private boolean includeVitals = true;
        private boolean includeLabs = true;
        private boolean includeClinicalScores = true;
        private boolean includeTemporal = true;
        private boolean includeMedications = true;
        private boolean includeComorbidities = true;
        private boolean includeCEPPatterns = true;
        private boolean useDefaultsForMissingValues = true;
        private double missingValueDefault = 0.0;
        private double minimumFeatureCompleteness = 0.7;
        private boolean failOnLowQuality = false;
        private boolean preferRecentData = true;
        private long maxDataAgeHours = 24;

        public Builder includeDemographics(boolean includeDemographics) {
            this.includeDemographics = includeDemographics;
            return this;
        }

        public Builder includeVitals(boolean includeVitals) {
            this.includeVitals = includeVitals;
            return this;
        }

        public Builder includeLabs(boolean includeLabs) {
            this.includeLabs = includeLabs;
            return this;
        }

        public Builder includeClinicalScores(boolean includeClinicalScores) {
            this.includeClinicalScores = includeClinicalScores;
            return this;
        }

        public Builder includeTemporal(boolean includeTemporal) {
            this.includeTemporal = includeTemporal;
            return this;
        }

        public Builder includeMedications(boolean includeMedications) {
            this.includeMedications = includeMedications;
            return this;
        }

        public Builder includeComorbidities(boolean includeComorbidities) {
            this.includeComorbidities = includeComorbidities;
            return this;
        }

        public Builder includeCEPPatterns(boolean includeCEPPatterns) {
            this.includeCEPPatterns = includeCEPPatterns;
            return this;
        }

        public Builder useDefaultsForMissingValues(boolean useDefaultsForMissingValues) {
            this.useDefaultsForMissingValues = useDefaultsForMissingValues;
            return this;
        }

        public Builder missingValueDefault(double missingValueDefault) {
            this.missingValueDefault = missingValueDefault;
            return this;
        }

        public Builder minimumFeatureCompleteness(double minimumFeatureCompleteness) {
            this.minimumFeatureCompleteness = minimumFeatureCompleteness;
            return this;
        }

        public Builder failOnLowQuality(boolean failOnLowQuality) {
            this.failOnLowQuality = failOnLowQuality;
            return this;
        }

        public Builder preferRecentData(boolean preferRecentData) {
            this.preferRecentData = preferRecentData;
            return this;
        }

        public Builder maxDataAgeHours(long maxDataAgeHours) {
            this.maxDataAgeHours = maxDataAgeHours;
            return this;
        }

        public FeatureExtractionConfig build() {
            return new FeatureExtractionConfig(this);
        }
    }
}
