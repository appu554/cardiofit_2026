package com.cardiofit.flink.ml;

import java.io.Serializable;
import java.util.Arrays;
import java.util.Objects;

/**
 * ClinicalFeatureVector - 70-dimensional feature array for ML model input
 *
 * This class represents a normalized 70-feature vector extracted from PatientContextSnapshot
 * for input into ONNX ML models (sepsis, deterioration, mortality, readmission risk).
 *
 * Feature organization (70 total):
 * - Demographics: 7 features (indices 0-6)
 * - Vital signs: 12 features (indices 7-18)
 * - Vital trends (6h): 6 features (indices 19-24)
 * - Labs - CBC: 4 features (indices 25-28)
 * - Labs - Chemistry: 8 features (indices 29-36)
 * - Labs - Liver panel: 4 features (indices 37-40)
 * - Labs - Blood gases: 3 features (indices 41-43)
 * - Labs - Cardiac: 2 features (indices 44-45)
 * - Labs - Coagulation: 2 features (indices 46-47)
 * - Labs - Other: 2 features (indices 48-49)
 * - Medications: 8 features (indices 50-57)
 * - Clinical scores: 6 features (indices 58-63)
 * - Comorbidities: 6 features (indices 64-69)
 *
 * All features are normalized to [0, 1] or standardized to mean=0, std=1
 * Missing values are imputed with median values or 0 for binary features.
 *
 * @see ClinicalFeatureExtractor for extraction logic from PatientContextSnapshot
 * @see ONNXModelContainer for model inference usage
 */
public class ClinicalFeatureVector implements Serializable {

    private static final long serialVersionUID = 1L;

    /**
     * Expected feature count for all models
     */
    public static final int FEATURE_COUNT = 70;

    /**
     * 70-dimensional feature array
     */
    private float[] features;

    /**
     * Patient identifier for tracking
     */
    private String patientId;

    /**
     * Encounter identifier for tracking
     */
    private String encounterId;

    /**
     * Feature extraction timestamp (milliseconds since epoch)
     */
    private long timestamp;

    /**
     * Indicates if any features are missing/imputed
     */
    private boolean hasMissingData;

    /**
     * Count of missing features that were imputed
     */
    private int missingFeatureCount;

    /**
     * Default constructor - initializes with zeros
     */
    public ClinicalFeatureVector() {
        this.features = new float[FEATURE_COUNT];
        this.timestamp = System.currentTimeMillis();
        this.hasMissingData = false;
        this.missingFeatureCount = 0;
    }

    /**
     * Constructor with patient identifiers
     *
     * @param patientId Patient identifier
     * @param encounterId Encounter identifier
     */
    public ClinicalFeatureVector(String patientId, String encounterId) {
        this();
        this.patientId = patientId;
        this.encounterId = encounterId;
    }

    /**
     * Constructor with pre-populated feature array
     *
     * @param patientId Patient identifier
     * @param encounterId Encounter identifier
     * @param features 70-element feature array
     * @throws IllegalArgumentException if features array is not length 70
     */
    public ClinicalFeatureVector(String patientId, String encounterId, float[] features) {
        if (features == null || features.length != FEATURE_COUNT) {
            throw new IllegalArgumentException(
                String.format("Features array must have exactly %d elements, got %d",
                    FEATURE_COUNT, features == null ? 0 : features.length)
            );
        }
        this.patientId = patientId;
        this.encounterId = encounterId;
        this.features = Arrays.copyOf(features, FEATURE_COUNT);
        this.timestamp = System.currentTimeMillis();
        this.hasMissingData = false;
        this.missingFeatureCount = 0;
    }

    /**
     * Set a specific feature value by index
     *
     * @param index Feature index (0-69)
     * @param value Feature value
     * @throws IndexOutOfBoundsException if index is not in range [0, 69]
     */
    public void setFeature(int index, float value) {
        if (index < 0 || index >= FEATURE_COUNT) {
            throw new IndexOutOfBoundsException(
                String.format("Feature index must be in range [0, %d], got %d",
                    FEATURE_COUNT - 1, index)
            );
        }
        features[index] = value;
    }

    /**
     * Get a specific feature value by index
     *
     * @param index Feature index (0-69)
     * @return Feature value
     * @throws IndexOutOfBoundsException if index is not in range [0, 69]
     */
    public float getFeature(int index) {
        if (index < 0 || index >= FEATURE_COUNT) {
            throw new IndexOutOfBoundsException(
                String.format("Feature index must be in range [0, %d], got %d",
                    FEATURE_COUNT - 1, index)
            );
        }
        return features[index];
    }

    /**
     * Get the entire feature array (copy for safety)
     *
     * @return 70-element feature array
     */
    public float[] getFeatures() {
        return Arrays.copyOf(features, FEATURE_COUNT);
    }

    /**
     * Get the feature array reference (unsafe - for performance-critical paths)
     *
     * @return Direct reference to 70-element feature array
     */
    public float[] getFeaturesUnsafe() {
        return features;
    }

    /**
     * Set the entire feature array
     *
     * @param features 70-element feature array
     * @throws IllegalArgumentException if array is not length 70
     */
    public void setFeatures(float[] features) {
        if (features == null || features.length != FEATURE_COUNT) {
            throw new IllegalArgumentException(
                String.format("Features array must have exactly %d elements, got %d",
                    FEATURE_COUNT, features == null ? 0 : features.length)
            );
        }
        this.features = Arrays.copyOf(features, FEATURE_COUNT);
    }

    /**
     * Validate that all features are within reasonable bounds
     *
     * @return true if all features are valid (not NaN, not infinite)
     */
    public boolean isValid() {
        for (float feature : features) {
            if (Float.isNaN(feature) || Float.isInfinite(feature)) {
                return false;
            }
        }
        return true;
    }

    /**
     * Replace NaN and infinite values with 0
     */
    public void sanitize() {
        for (int i = 0; i < features.length; i++) {
            if (Float.isNaN(features[i]) || Float.isInfinite(features[i])) {
                features[i] = 0.0f;
            }
        }
    }

    /**
     * Mark a feature as missing (increments missing count)
     *
     * @param index Feature index
     * @param imputedValue Value to use for imputation
     */
    public void markMissing(int index, float imputedValue) {
        setFeature(index, imputedValue);
        this.hasMissingData = true;
        this.missingFeatureCount++;
    }

    /**
     * Calculate data completeness ratio
     *
     * @return Ratio of non-missing features (0.0 to 1.0)
     */
    public double getCompletenessRatio() {
        return (FEATURE_COUNT - missingFeatureCount) / (double) FEATURE_COUNT;
    }

    /**
     * Get summary statistics of feature values
     *
     * @return FeatureStats object with min, max, mean, std
     */
    public FeatureStats getStats() {
        float min = Float.MAX_VALUE;
        float max = Float.MIN_VALUE;
        double sum = 0.0;

        for (float feature : features) {
            if (feature < min) min = feature;
            if (feature > max) max = feature;
            sum += feature;
        }

        double mean = sum / FEATURE_COUNT;

        double sumSquaredDiff = 0.0;
        for (float feature : features) {
            double diff = feature - mean;
            sumSquaredDiff += diff * diff;
        }
        double std = Math.sqrt(sumSquaredDiff / FEATURE_COUNT);

        return new FeatureStats(min, max, mean, std);
    }

    // Getters and Setters

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public String getEncounterId() {
        return encounterId;
    }

    public void setEncounterId(String encounterId) {
        this.encounterId = encounterId;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(long timestamp) {
        this.timestamp = timestamp;
    }

    public boolean isHasMissingData() {
        return hasMissingData;
    }

    public void setHasMissingData(boolean hasMissingData) {
        this.hasMissingData = hasMissingData;
    }

    public int getMissingFeatureCount() {
        return missingFeatureCount;
    }

    public void setMissingFeatureCount(int missingFeatureCount) {
        this.missingFeatureCount = missingFeatureCount;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        ClinicalFeatureVector that = (ClinicalFeatureVector) o;
        return timestamp == that.timestamp &&
               Arrays.equals(features, that.features) &&
               Objects.equals(patientId, that.patientId) &&
               Objects.equals(encounterId, that.encounterId);
    }

    @Override
    public int hashCode() {
        int result = Objects.hash(patientId, encounterId, timestamp);
        result = 31 * result + Arrays.hashCode(features);
        return result;
    }

    @Override
    public String toString() {
        return String.format(
            "ClinicalFeatureVector{patientId='%s', encounterId='%s', timestamp=%d, features=%d, missing=%d (%.1f%% complete)}",
            patientId, encounterId, timestamp, FEATURE_COUNT, missingFeatureCount, getCompletenessRatio() * 100
        );
    }

    /**
     * Inner class for feature statistics
     */
    public static class FeatureStats {
        private final float min;
        private final float max;
        private final double mean;
        private final double std;

        public FeatureStats(float min, float max, double mean, double std) {
            this.min = min;
            this.max = max;
            this.mean = mean;
            this.std = std;
        }

        public float getMin() { return min; }
        public float getMax() { return max; }
        public double getMean() { return mean; }
        public double getStd() { return std; }

        @Override
        public String toString() {
            return String.format("FeatureStats{min=%.3f, max=%.3f, mean=%.3f, std=%.3f}",
                min, max, mean, std);
        }
    }

    /**
     * Feature index constants for self-documentation
     */
    public static class FeatureIndex {
        // Demographics (0-6)
        public static final int AGE = 0;
        public static final int GENDER = 1;
        public static final int ETHNICITY = 2;
        public static final int WEIGHT = 3;
        public static final int HEIGHT = 4;
        public static final int BMI = 5;
        public static final int ADMISSION_TYPE = 6;

        // Vital signs (7-18)
        public static final int HEART_RATE = 7;
        public static final int SYSTOLIC_BP = 8;
        public static final int DIASTOLIC_BP = 9;
        public static final int MAP = 10;
        public static final int RESPIRATORY_RATE = 11;
        public static final int TEMPERATURE = 12;
        public static final int SPO2 = 13;
        public static final int SHOCK_INDEX = 14;
        public static final int PULSE_WIDTH = 15;

        // Vital trends 6h (19-24)
        public static final int HR_CHANGE_6H = 19;
        public static final int BP_CHANGE_6H = 20;
        public static final int RR_CHANGE_6H = 21;
        public static final int TEMP_CHANGE_6H = 22;
        public static final int LACTATE_CHANGE_6H = 23;
        public static final int CREAT_CHANGE_6H = 24;

        // Labs - CBC (25-28)
        public static final int WBC = 25;
        public static final int HEMOGLOBIN = 26;
        public static final int PLATELETS = 27;
        public static final int HEMATOCRIT = 28;

        // Labs - Chemistry (29-36)
        public static final int SODIUM = 29;
        public static final int POTASSIUM = 30;
        public static final int CHLORIDE = 31;
        public static final int BICARBONATE = 32;
        public static final int BUN = 33;
        public static final int CREATININE = 34;
        public static final int GLUCOSE = 35;
        public static final int CALCIUM = 36;

        // Labs - Liver panel (37-40)
        public static final int BILIRUBIN = 37;
        public static final int AST = 38;
        public static final int ALT = 39;
        public static final int ALKALINE_PHOSPHATASE = 40;

        // Labs - Blood gases (41-43)
        public static final int PH = 41;
        public static final int PAO2 = 42;
        public static final int PACO2 = 43;

        // Labs - Cardiac markers (44-45)
        public static final int TROPONIN = 44;
        public static final int BNP = 45;

        // Labs - Coagulation (46-47)
        public static final int INR = 46;
        public static final int PTT = 47;

        // Labs - Other critical (48-49)
        public static final int LACTATE = 48;
        public static final int ALBUMIN = 49;

        // Medications (50-57)
        public static final int ON_VASOPRESSORS = 50;
        public static final int ON_SEDATIVES = 51;
        public static final int ON_ANTIBIOTICS = 52;
        public static final int ON_ANTICOAGULANTS = 53;
        public static final int ON_INSULIN = 54;
        public static final int ON_DIALYSIS = 55;
        public static final int ON_MECHANICAL_VENT = 56;
        public static final int ON_SUPPLEMENTAL_O2 = 57;

        // Clinical scores (58-63)
        public static final int SOFA_SCORE = 58;
        public static final int APACHE_SCORE = 59;
        public static final int NEWS2_SCORE = 60;
        public static final int QSOFA_SCORE = 61;
        public static final int CHARLSON_INDEX = 62;
        public static final int ELIXHAUSER_SCORE = 63;

        // Comorbidities (64-69)
        public static final int HAS_DIABETES = 64;
        public static final int HAS_HYPERTENSION = 65;
        public static final int HAS_HEART_FAILURE = 66;
        public static final int HAS_COPD = 67;
        public static final int HAS_CKD = 68;
        public static final int HAS_CANCER = 69;
    }
}
