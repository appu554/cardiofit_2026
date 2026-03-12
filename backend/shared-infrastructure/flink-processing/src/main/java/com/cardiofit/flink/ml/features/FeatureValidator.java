package com.cardiofit.flink.ml.features;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Feature Validation Pipeline
 *
 * Validates and cleans clinical features before ML model inference:
 * - Missing value imputation (median, mean, mode strategies)
 * - Outlier detection and handling (Winsorization, clipping)
 * - Range validation (ensure features within clinical bounds)
 * - Quality scoring
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class FeatureValidator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(FeatureValidator.class);

    // Feature value ranges (min, max) for clinical validity
    private static final Map<String, double[]> FEATURE_RANGES = createFeatureRanges();

    // Imputation strategies
    private final ImputationStrategy imputationStrategy;

    // Outlier handling
    private final boolean enableOutlierDetection;
    private final double outlierPercentileLower;  // e.g., 1st percentile
    private final double outlierPercentileUpper;  // e.g., 99th percentile

    public enum ImputationStrategy {
        MEDIAN,     // Use median value for missing continuous features
        MEAN,       // Use mean value for missing continuous features
        MODE,       // Use mode for categorical features
        ZERO,       // Replace with 0.0
        FORWARD_FILL  // Use last known value (requires history)
    }

    public FeatureValidator() {
        this(ImputationStrategy.MEDIAN, true, 1.0, 99.0);
    }

    public FeatureValidator(ImputationStrategy strategy,
                           boolean enableOutlierDetection,
                           double outlierPercentileLower,
                           double outlierPercentileUpper) {
        this.imputationStrategy = strategy;
        this.enableOutlierDetection = enableOutlierDetection;
        this.outlierPercentileLower = outlierPercentileLower;
        this.outlierPercentileUpper = outlierPercentileUpper;
    }

    /**
     * Validate and clean feature vector
     *
     * @param featureVector Input feature vector
     * @return Validated and cleaned feature vector
     */
    public ClinicalFeatureVector validate(ClinicalFeatureVector featureVector) {
        if (featureVector == null || featureVector.getFeatures() == null) {
            LOG.warn("Null feature vector provided for validation");
            return featureVector;
        }

        long startTime = System.nanoTime();
        Map<String, Double> features = new LinkedHashMap<>(featureVector.getFeatures());

        int imputedCount = 0;
        int clippedCount = 0;
        int invalidCount = 0;

        // Process each feature
        for (Map.Entry<String, Double> entry : features.entrySet()) {
            String featureName = entry.getKey();
            Double value = entry.getValue();

            // Step 1: Handle missing values
            if (value == null || Double.isNaN(value)) {
                value = imputeMissingValue(featureName);
                entry.setValue(value);
                imputedCount++;
            }

            // Step 2: Validate range
            if (FEATURE_RANGES.containsKey(featureName)) {
                double[] range = FEATURE_RANGES.get(featureName);
                double minValue = range[0];
                double maxValue = range[1];

                if (value < minValue || value > maxValue) {
                    // Clip to valid range
                    double originalValue = value;
                    value = Math.max(minValue, Math.min(maxValue, value));
                    entry.setValue(value);

                    if (originalValue != value) {
                        clippedCount++;
                        LOG.debug("Clipped feature '{}' from {} to {}",
                            featureName, originalValue, value);
                    }
                }
            }

            // Step 3: Outlier detection (Winsorization)
            if (enableOutlierDetection && isContinuousFeature(featureName)) {
                value = handleOutlier(featureName, value);
                entry.setValue(value);
            }

            // Step 4: Check for infinite values
            if (Double.isInfinite(value)) {
                value = imputeMissingValue(featureName);
                entry.setValue(value);
                invalidCount++;
            }
        }

        long validationTimeNs = System.nanoTime() - startTime;
        double validationTimeMs = validationTimeNs / 1_000_000.0;

        LOG.debug("Feature validation: {} imputed, {} clipped, {} invalid ({}ms)",
            imputedCount, clippedCount, invalidCount, String.format("%.2f", validationTimeMs));

        // Return new feature vector with validated features
        return ClinicalFeatureVector.builder()
            .patientId(featureVector.getPatientId())
            .features(features)
            .featureCount(features.size())
            .extractionTimestamp(featureVector.getExtractionTimestamp())
            .extractionTimeMs(featureVector.getExtractionTimeMs() + validationTimeMs)
            .build();
    }

    /**
     * Impute missing value based on configured strategy
     */
    private double imputeMissingValue(String featureName) {
        switch (imputationStrategy) {
            case MEDIAN:
                return getMedianValue(featureName);

            case MEAN:
                return getMeanValue(featureName);

            case MODE:
                return getModeValue(featureName);

            case ZERO:
                return 0.0;

            case FORWARD_FILL:
                // Would require historical data - fall back to median
                return getMedianValue(featureName);

            default:
                return 0.0;
        }
    }

    /**
     * Handle outliers using Winsorization
     */
    private double handleOutlier(String featureName, double value) {
        if (!FEATURE_RANGES.containsKey(featureName)) {
            return value;
        }

        double[] range = FEATURE_RANGES.get(featureName);
        double minValue = range[0];
        double maxValue = range[1];

        // Calculate percentile bounds
        double rangeWidth = maxValue - minValue;
        double lowerBound = minValue + (rangeWidth * outlierPercentileLower / 100.0);
        double upperBound = minValue + (rangeWidth * outlierPercentileUpper / 100.0);

        // Winsorize: cap at percentile bounds
        if (value < lowerBound) {
            return lowerBound;
        } else if (value > upperBound) {
            return upperBound;
        }

        return value;
    }

    /**
     * Check if feature is continuous (vs binary/categorical)
     */
    private boolean isContinuousFeature(String featureName) {
        // Binary features (0 or 1)
        if (featureName.contains("_abnormal") ||
            featureName.contains("_detected") ||
            featureName.contains("_active") ||
            featureName.contains("_present") ||
            featureName.contains("_elevated") ||
            featureName.contains("comorbid_") ||
            featureName.contains("_male") ||
            featureName.contains("_emergency") ||
            featureName.contains("_icu") ||
            featureName.contains("_increasing") ||
            featureName.contains("_decreasing") ||
            featureName.contains("_night") ||
            featureName.contains("_weekend") ||
            featureName.contains("_fever")) {
            return false;
        }

        return true;
    }

    /**
     * Get median value for feature (clinical population median)
     */
    private double getMedianValue(String featureName) {
        // Clinical population medians
        if (featureName.equals("demo_age_years")) return 65.0;
        if (featureName.equals("demo_bmi")) return 27.0;

        if (featureName.equals("vital_heart_rate")) return 80.0;
        if (featureName.equals("vital_systolic_bp")) return 120.0;
        if (featureName.equals("vital_diastolic_bp")) return 75.0;
        if (featureName.equals("vital_respiratory_rate")) return 16.0;
        if (featureName.equals("vital_temperature_c")) return 37.0;
        if (featureName.equals("vital_oxygen_saturation")) return 97.0;
        if (featureName.equals("vital_mean_arterial_pressure")) return 90.0;
        if (featureName.equals("vital_pulse_pressure")) return 45.0;
        if (featureName.equals("vital_shock_index")) return 0.67;

        if (featureName.equals("lab_lactate_mmol")) return 1.5;
        if (featureName.equals("lab_creatinine_mg_dl")) return 1.0;
        if (featureName.equals("lab_bun_mg_dl")) return 18.0;
        if (featureName.equals("lab_sodium_meq")) return 140.0;
        if (featureName.equals("lab_potassium_meq")) return 4.0;
        if (featureName.equals("lab_chloride_meq")) return 102.0;
        if (featureName.equals("lab_bicarbonate_meq")) return 24.0;
        if (featureName.equals("lab_wbc_k_ul")) return 8.0;
        if (featureName.equals("lab_hemoglobin_g_dl")) return 13.0;
        if (featureName.equals("lab_platelets_k_ul")) return 220.0;

        if (featureName.equals("score_news2")) return 3.0;
        if (featureName.equals("score_qsofa")) return 0.0;
        if (featureName.equals("score_sofa")) return 2.0;
        if (featureName.equals("score_apache")) return 15.0;

        if (featureName.equals("temporal_length_of_stay_hours")) return 48.0;
        if (featureName.equals("med_total_count")) return 5.0;

        // Default to 0.0 for unknown features
        return 0.0;
    }

    /**
     * Get mean value for feature (clinical population mean)
     */
    private double getMeanValue(String featureName) {
        // For most features, mean ≈ median in clinical populations
        return getMedianValue(featureName);
    }

    /**
     * Get mode value for categorical/binary features
     */
    private double getModeValue(String featureName) {
        // Most binary features default to 0 (not present)
        return 0.0;
    }

    /**
     * Define clinical valid ranges for features
     */
    private static Map<String, double[]> createFeatureRanges() {
        Map<String, double[]> ranges = new HashMap<>();

        // Demographics
        ranges.put("demo_age_years", new double[]{0, 120});
        ranges.put("demo_bmi", new double[]{10, 60});

        // Vitals
        ranges.put("vital_heart_rate", new double[]{20, 220});
        ranges.put("vital_systolic_bp", new double[]{40, 250});
        ranges.put("vital_diastolic_bp", new double[]{20, 180});
        ranges.put("vital_respiratory_rate", new double[]{4, 60});
        ranges.put("vital_temperature_c", new double[]{32, 42});
        ranges.put("vital_oxygen_saturation", new double[]{50, 100});
        ranges.put("vital_mean_arterial_pressure", new double[]{30, 200});
        ranges.put("vital_pulse_pressure", new double[]{10, 150});
        ranges.put("vital_shock_index", new double[]{0.2, 3.0});

        // Labs
        ranges.put("lab_lactate_mmol", new double[]{0.1, 20});
        ranges.put("lab_creatinine_mg_dl", new double[]{0.3, 15});
        ranges.put("lab_bun_mg_dl", new double[]{2, 150});
        ranges.put("lab_sodium_meq", new double[]{110, 170});
        ranges.put("lab_potassium_meq", new double[]{1.5, 8.0});
        ranges.put("lab_chloride_meq", new double[]{70, 130});
        ranges.put("lab_bicarbonate_meq", new double[]{5, 45});
        ranges.put("lab_wbc_k_ul", new double[]{0.5, 50});
        ranges.put("lab_hemoglobin_g_dl", new double[]{3, 20});
        ranges.put("lab_platelets_k_ul", new double[]{5, 1000});
        ranges.put("lab_ast_u_l", new double[]{5, 5000});
        ranges.put("lab_alt_u_l", new double[]{5, 5000});
        ranges.put("lab_bilirubin_mg_dl", new double[]{0.1, 30});

        // Clinical Scores
        ranges.put("score_news2", new double[]{0, 20});
        ranges.put("score_qsofa", new double[]{0, 3});
        ranges.put("score_sofa", new double[]{0, 24});
        ranges.put("score_apache", new double[]{0, 71});
        ranges.put("score_acuity_combined", new double[]{0, 10});

        // Temporal
        ranges.put("temporal_hours_since_admission", new double[]{0, 8760});  // 1 year
        ranges.put("temporal_hours_since_last_vitals", new double[]{0, 168});  // 1 week
        ranges.put("temporal_hours_since_last_labs", new double[]{0, 168});
        ranges.put("temporal_length_of_stay_hours", new double[]{0, 8760});
        ranges.put("temporal_hour_of_day", new double[]{0, 23});

        // Medications
        ranges.put("med_total_count", new double[]{0, 50});
        ranges.put("med_high_risk_count", new double[]{0, 20});

        // Comorbidities
        ranges.put("comorbid_charlson_index", new double[]{0, 30});

        // CEP Patterns
        ranges.put("pattern_confidence_score", new double[]{0, 1});
        ranges.put("pattern_clinical_significance", new double[]{0, 1});

        return ranges;
    }

    /**
     * Get validation statistics
     */
    public ValidationStatistics getValidationStatistics(ClinicalFeatureVector original,
                                                       ClinicalFeatureVector validated) {
        int totalFeatures = validated.getFeatureCount();
        int modifiedFeatures = 0;

        Map<String, Double> origFeatures = original.getFeatures();
        Map<String, Double> validFeatures = validated.getFeatures();

        for (String key : validFeatures.keySet()) {
            Double origValue = origFeatures.get(key);
            Double validValue = validFeatures.get(key);

            if (origValue == null && validValue != null) {
                modifiedFeatures++;  // Imputed
            } else if (origValue != null && validValue != null &&
                      !origValue.equals(validValue)) {
                modifiedFeatures++;  // Clipped or Winsorized
            }
        }

        return new ValidationStatistics(totalFeatures, modifiedFeatures);
    }

    /**
     * Validation statistics container
     */
    public static class ValidationStatistics implements Serializable {
        private final int totalFeatures;
        private final int modifiedFeatures;

        public ValidationStatistics(int totalFeatures, int modifiedFeatures) {
            this.totalFeatures = totalFeatures;
            this.modifiedFeatures = modifiedFeatures;
        }

        public int getTotalFeatures() { return totalFeatures; }
        public int getModifiedFeatures() { return modifiedFeatures; }
        public double getModificationRate() {
            return totalFeatures > 0 ? (double) modifiedFeatures / totalFeatures : 0.0;
        }

        @Override
        public String toString() {
            return String.format("ValidationStats{total=%d, modified=%d, rate=%.1f%%}",
                totalFeatures, modifiedFeatures, getModificationRate() * 100);
        }
    }
}
