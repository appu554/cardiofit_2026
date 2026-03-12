package com.cardiofit.flink.ml.features;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Feature Normalization Pipeline
 *
 * Normalizes clinical features for ML model inference using various strategies:
 * - Standard scaling: (x - mean) / std
 * - Min-max scaling: (x - min) / (max - min)
 * - Log transformation: log(x + 1) for skewed distributions
 * - Z-score normalization with clipping
 *
 * Normalization improves ML model convergence and prediction accuracy by:
 * - Putting all features on similar scales
 * - Reducing impact of outliers
 * - Handling skewed distributions
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class FeatureNormalizer implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(FeatureNormalizer.class);

    // Normalization strategy
    private final NormalizationStrategy strategy;

    // Z-score clipping bounds (prevent extreme values)
    private final double zScoreMin;
    private final double zScoreMax;

    // Feature statistics (population-based from training data)
    private static final Map<String, FeatureStatistics> FEATURE_STATS = createFeatureStatistics();

    public enum NormalizationStrategy {
        STANDARD_SCALING,   // (x - mean) / std
        MIN_MAX_SCALING,    // (x - min) / (max - min)
        LOG_TRANSFORM,      // log(x + 1)
        Z_SCORE_CLIPPED,    // Z-score with [-3, +3] clipping
        NONE                // No normalization
    }

    public FeatureNormalizer() {
        this(NormalizationStrategy.STANDARD_SCALING, -3.0, 3.0);
    }

    public FeatureNormalizer(NormalizationStrategy strategy) {
        this(strategy, -3.0, 3.0);
    }

    public FeatureNormalizer(NormalizationStrategy strategy,
                            double zScoreMin,
                            double zScoreMax) {
        this.strategy = strategy;
        this.zScoreMin = zScoreMin;
        this.zScoreMax = zScoreMax;
    }

    /**
     * Normalize feature vector
     *
     * @param featureVector Input feature vector (validated)
     * @return Normalized feature vector ready for ML model
     */
    public ClinicalFeatureVector normalize(ClinicalFeatureVector featureVector) {
        if (featureVector == null || featureVector.getFeatures() == null) {
            LOG.warn("Null feature vector provided for normalization");
            return featureVector;
        }

        if (strategy == NormalizationStrategy.NONE) {
            return featureVector;
        }

        long startTime = System.nanoTime();
        Map<String, Double> features = new LinkedHashMap<>(featureVector.getFeatures());

        int normalizedCount = 0;

        // Normalize each feature
        for (Map.Entry<String, Double> entry : features.entrySet()) {
            String featureName = entry.getKey();
            Double value = entry.getValue();

            if (value == null || Double.isNaN(value)) {
                continue;  // Skip null/NaN (should be handled by validator)
            }

            // Skip binary features (already 0 or 1)
            if (isBinaryFeature(featureName)) {
                continue;
            }

            // Apply normalization
            double normalizedValue = normalizeValue(featureName, value);

            if (normalizedValue != value) {
                entry.setValue(normalizedValue);
                normalizedCount++;
            }
        }

        long normalizationTimeNs = System.nanoTime() - startTime;
        double normalizationTimeMs = normalizationTimeNs / 1_000_000.0;

        LOG.debug("Feature normalization: {} features normalized using {} ({}ms)",
            normalizedCount, strategy, String.format("%.2f", normalizationTimeMs));

        // Return new feature vector with normalized features
        return ClinicalFeatureVector.builder()
            .patientId(featureVector.getPatientId())
            .features(features)
            .featureCount(features.size())
            .extractionTimestamp(featureVector.getExtractionTimestamp())
            .extractionTimeMs(featureVector.getExtractionTimeMs() + normalizationTimeMs)
            .build();
    }

    /**
     * Normalize single feature value based on strategy
     */
    private double normalizeValue(String featureName, double value) {
        FeatureStatistics stats = FEATURE_STATS.get(featureName);

        if (stats == null) {
            // Unknown feature - return as-is
            return value;
        }

        switch (strategy) {
            case STANDARD_SCALING:
                return standardScale(value, stats);

            case MIN_MAX_SCALING:
                return minMaxScale(value, stats);

            case LOG_TRANSFORM:
                return logTransform(value, stats);

            case Z_SCORE_CLIPPED:
                return zScoreClipped(value, stats);

            default:
                return value;
        }
    }

    /**
     * Standard scaling: (x - mean) / std
     */
    private double standardScale(double value, FeatureStatistics stats) {
        if (stats.std == 0.0) {
            return 0.0;  // Constant feature
        }

        return (value - stats.mean) / stats.std;
    }

    /**
     * Min-max scaling: (x - min) / (max - min)
     * Scales to [0, 1] range
     */
    private double minMaxScale(double value, FeatureStatistics stats) {
        double range = stats.max - stats.min;

        if (range == 0.0) {
            return 0.0;  // Constant feature
        }

        return (value - stats.min) / range;
    }

    /**
     * Log transformation: log(x + 1)
     * Used for skewed distributions (e.g., length of stay, lab values)
     */
    private double logTransform(double value, FeatureStatistics stats) {
        // Ensure non-negative
        double shiftedValue = value - stats.min + 1.0;
        return Math.log(shiftedValue);
    }

    /**
     * Z-score with clipping to prevent extreme values
     */
    private double zScoreClipped(double value, FeatureStatistics stats) {
        if (stats.std == 0.0) {
            return 0.0;
        }

        double zScore = (value - stats.mean) / stats.std;

        // Clip to [-3, +3] standard deviations (configurable)
        return Math.max(zScoreMin, Math.min(zScoreMax, zScore));
    }

    /**
     * Check if feature is binary (0/1)
     */
    private boolean isBinaryFeature(String featureName) {
        return featureName.contains("_abnormal") ||
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
               featureName.contains("_fever") ||
               featureName.contains("_polypharmacy");
    }

    /**
     * Create feature statistics from population data
     * (In production, these would be computed from training data)
     */
    private static Map<String, FeatureStatistics> createFeatureStatistics() {
        Map<String, FeatureStatistics> stats = new HashMap<>();

        // Demographics
        stats.put("demo_age_years", new FeatureStatistics(65.0, 15.0, 0, 120));
        stats.put("demo_bmi", new FeatureStatistics(27.0, 6.0, 10, 60));

        // Vitals
        stats.put("vital_heart_rate", new FeatureStatistics(80.0, 18.0, 20, 220));
        stats.put("vital_systolic_bp", new FeatureStatistics(120.0, 20.0, 40, 250));
        stats.put("vital_diastolic_bp", new FeatureStatistics(75.0, 12.0, 20, 180));
        stats.put("vital_respiratory_rate", new FeatureStatistics(16.0, 4.0, 4, 60));
        stats.put("vital_temperature_c", new FeatureStatistics(37.0, 0.8, 32, 42));
        stats.put("vital_oxygen_saturation", new FeatureStatistics(97.0, 3.0, 50, 100));
        stats.put("vital_mean_arterial_pressure", new FeatureStatistics(90.0, 15.0, 30, 200));
        stats.put("vital_pulse_pressure", new FeatureStatistics(45.0, 15.0, 10, 150));
        stats.put("vital_shock_index", new FeatureStatistics(0.67, 0.2, 0.2, 3.0));

        // Labs
        stats.put("lab_lactate_mmol", new FeatureStatistics(1.5, 1.2, 0.1, 20));
        stats.put("lab_creatinine_mg_dl", new FeatureStatistics(1.0, 0.8, 0.3, 15));
        stats.put("lab_bun_mg_dl", new FeatureStatistics(18.0, 12.0, 2, 150));
        stats.put("lab_sodium_meq", new FeatureStatistics(140.0, 5.0, 110, 170));
        stats.put("lab_potassium_meq", new FeatureStatistics(4.0, 0.6, 1.5, 8.0));
        stats.put("lab_chloride_meq", new FeatureStatistics(102.0, 5.0, 70, 130));
        stats.put("lab_bicarbonate_meq", new FeatureStatistics(24.0, 4.0, 5, 45));
        stats.put("lab_wbc_k_ul", new FeatureStatistics(8.0, 4.0, 0.5, 50));
        stats.put("lab_hemoglobin_g_dl", new FeatureStatistics(13.0, 2.0, 3, 20));
        stats.put("lab_platelets_k_ul", new FeatureStatistics(220.0, 80.0, 5, 1000));
        stats.put("lab_ast_u_l", new FeatureStatistics(30.0, 40.0, 5, 5000));
        stats.put("lab_alt_u_l", new FeatureStatistics(30.0, 40.0, 5, 5000));
        stats.put("lab_bilirubin_mg_dl", new FeatureStatistics(0.8, 0.6, 0.1, 30));

        // Clinical Scores
        stats.put("score_news2", new FeatureStatistics(3.0, 3.0, 0, 20));
        stats.put("score_qsofa", new FeatureStatistics(0.5, 0.7, 0, 3));
        stats.put("score_sofa", new FeatureStatistics(2.0, 3.0, 0, 24));
        stats.put("score_apache", new FeatureStatistics(15.0, 10.0, 0, 71));
        stats.put("score_acuity_combined", new FeatureStatistics(5.0, 2.5, 0, 10));

        // Temporal
        stats.put("temporal_hours_since_admission", new FeatureStatistics(48.0, 72.0, 0, 8760));
        stats.put("temporal_hours_since_last_vitals", new FeatureStatistics(2.0, 3.0, 0, 168));
        stats.put("temporal_hours_since_last_labs", new FeatureStatistics(6.0, 8.0, 0, 168));
        stats.put("temporal_length_of_stay_hours", new FeatureStatistics(48.0, 72.0, 0, 8760));
        stats.put("temporal_hour_of_day", new FeatureStatistics(12.0, 7.0, 0, 23));

        // Medications
        stats.put("med_total_count", new FeatureStatistics(5.0, 4.0, 0, 50));
        stats.put("med_high_risk_count", new FeatureStatistics(1.0, 1.5, 0, 20));

        // Comorbidities
        stats.put("comorbid_charlson_index", new FeatureStatistics(3.0, 3.0, 0, 30));

        // CEP Patterns
        stats.put("pattern_confidence_score", new FeatureStatistics(0.5, 0.3, 0, 1));
        stats.put("pattern_clinical_significance", new FeatureStatistics(0.5, 0.3, 0, 1));

        return stats;
    }

    /**
     * Feature statistics container
     */
    private static class FeatureStatistics implements Serializable {
        final double mean;
        final double std;
        final double min;
        final double max;

        FeatureStatistics(double mean, double std, double min, double max) {
            this.mean = mean;
            this.std = std;
            this.min = min;
            this.max = max;
        }
    }

    /**
     * Get normalization statistics
     */
    public NormalizationStatistics getNormalizationStatistics(ClinicalFeatureVector original,
                                                              ClinicalFeatureVector normalized) {
        int totalFeatures = normalized.getFeatureCount();
        int normalizedFeatures = 0;

        Map<String, Double> origFeatures = original.getFeatures();
        Map<String, Double> normFeatures = normalized.getFeatures();

        double sumAbsChange = 0.0;

        for (String key : normFeatures.keySet()) {
            Double origValue = origFeatures.get(key);
            Double normValue = normFeatures.get(key);

            if (origValue != null && normValue != null && !origValue.equals(normValue)) {
                normalizedFeatures++;
                sumAbsChange += Math.abs(normValue - origValue);
            }
        }

        double avgAbsChange = normalizedFeatures > 0 ? sumAbsChange / normalizedFeatures : 0.0;

        return new NormalizationStatistics(
            totalFeatures,
            normalizedFeatures,
            avgAbsChange,
            strategy
        );
    }

    /**
     * Normalization statistics container
     */
    public static class NormalizationStatistics implements Serializable {
        private final int totalFeatures;
        private final int normalizedFeatures;
        private final double averageAbsoluteChange;
        private final NormalizationStrategy strategy;

        public NormalizationStatistics(int totalFeatures,
                                      int normalizedFeatures,
                                      double averageAbsoluteChange,
                                      NormalizationStrategy strategy) {
            this.totalFeatures = totalFeatures;
            this.normalizedFeatures = normalizedFeatures;
            this.averageAbsoluteChange = averageAbsoluteChange;
            this.strategy = strategy;
        }

        public int getTotalFeatures() { return totalFeatures; }
        public int getNormalizedFeatures() { return normalizedFeatures; }
        public double getAverageAbsoluteChange() { return averageAbsoluteChange; }
        public NormalizationStrategy getStrategy() { return strategy; }

        public double getNormalizationRate() {
            return totalFeatures > 0 ? (double) normalizedFeatures / totalFeatures : 0.0;
        }

        @Override
        public String toString() {
            return String.format("NormalizationStats{strategy=%s, total=%d, normalized=%d, avgChange=%.3f}",
                strategy, totalFeatures, normalizedFeatures, averageAbsoluteChange);
        }
    }
}
