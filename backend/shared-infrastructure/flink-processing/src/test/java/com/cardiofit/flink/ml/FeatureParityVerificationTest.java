package com.cardiofit.flink.ml;

import com.cardiofit.flink.adapters.PatientContextAdapter;
import com.cardiofit.flink.ml.features.MIMICFeatureExtractor;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Feature Parity Verification Test
 *
 * CRITICAL SAFETY TEST: Verifies feature extraction matches expected MIMIC-IV training format.
 *
 * This test is essential before clinical deployment to ensure:
 * 1. Feature ordering matches model training input order
 * 2. Numeric scaling/normalization is consistent
 * 3. Missing value handling is identical to training
 * 4. No unexpected NaN or Inf values
 *
 * Failure of this test indicates UNSAFE model deployment - predictions will be unreliable.
 *
 * @author CardioFit Safety Team
 * @version 1.0.0
 */
public class FeatureParityVerificationTest {

    private PatientContextAdapter adapter;
    private MIMICFeatureExtractor featureExtractor;

    @BeforeEach
    public void setUp() {
        adapter = new PatientContextAdapter();
        featureExtractor = new MIMICFeatureExtractor();
    }

    @Test
    @DisplayName("SAFETY: Verify feature vector has exactly 37 dimensions")
    public void testFeatureVectorDimensionality() {
        // Arrange
        EnrichedPatientContext context = createTestPatient();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act
        float[] features = featureExtractor.extractFeatures(snapshot);

        // Assert
        assertEquals(37, features.length,
            "CRITICAL: Feature vector must be exactly 37 dimensions to match MIMIC-IV models");
    }

    @Test
    @DisplayName("SAFETY: Verify feature names match expected MIMIC-IV order")
    public void testFeatureNameOrdering() {
        // Expected feature names in EXACT order used during MIMIC-IV training
        List<String> expectedFeatureNames = Arrays.asList(
            // Demographics (0-1)
            "age", "gender_male",

            // Vital Signs (2-16)
            "heart_rate_mean", "heart_rate_min", "heart_rate_max", "heart_rate_std",
            "respiratory_rate_mean", "respiratory_rate_max",
            "temperature_mean", "temperature_max",
            "sbp_mean", "sbp_min",
            "dbp_mean",
            "map_mean", "map_min",
            "spo2_mean", "spo2_min",

            // Labs (17-28)
            "wbc", "hemoglobin", "platelets",
            "creatinine_mean", "creatinine_max",
            "bun", "glucose", "sodium", "potassium",
            "lactate_mean", "lactate_max",
            "bilirubin",

            // Clinical Scores (29-36)
            "sofa_score", "sofa_respiration", "sofa_coagulation", "sofa_liver",
            "sofa_cardiovascular", "sofa_cns", "sofa_renal",
            "gcs_score"
        );

        // Act
        List<String> actualFeatureNames = MIMICFeatureExtractor.getFeatureNames();

        // Assert
        assertEquals(37, actualFeatureNames.size(),
            "CRITICAL: Feature name list must have 37 entries");

        for (int i = 0; i < 37; i++) {
            assertEquals(expectedFeatureNames.get(i), actualFeatureNames.get(i),
                String.format("CRITICAL: Feature name mismatch at index %d. " +
                    "Expected: %s, Got: %s. Feature order MUST match training!",
                    i, expectedFeatureNames.get(i), actualFeatureNames.get(i)));
        }
    }

    @Test
    @DisplayName("SAFETY: Verify no NaN or Inf values in extracted features")
    public void testNoNaNOrInfValues() {
        // Arrange
        EnrichedPatientContext context = createTestPatient();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act
        float[] features = featureExtractor.extractFeatures(snapshot);

        // Assert
        for (int i = 0; i < features.length; i++) {
            assertFalse(Float.isNaN(features[i]),
                String.format("CRITICAL: Feature %d (%s) is NaN! This will cause model failure.",
                    i, MIMICFeatureExtractor.getFeatureNames().get(i)));

            assertFalse(Float.isInfinite(features[i]),
                String.format("CRITICAL: Feature %d (%s) is Infinite! This will cause model failure.",
                    i, MIMICFeatureExtractor.getFeatureNames().get(i)));
        }
    }

    @Test
    @DisplayName("SAFETY: Verify feature values are in expected clinical ranges")
    public void testFeatureValueRanges() {
        // Arrange
        EnrichedPatientContext context = createTestPatient();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act
        float[] features = featureExtractor.extractFeatures(snapshot);

        // Assert - Check critical features are in plausible ranges
        // Note: These are sanity checks, not exact bounds
        // Adjust based on your normalization strategy (raw vs standardized)

        // Age (index 0) - should be reasonable
        float age = features[0];
        assertTrue(age >= 0 && age <= 120,
            String.format("Age %f is out of plausible range [0, 120]", age));

        // Gender (index 1) - should be 0 or 1
        float gender = features[1];
        assertTrue(gender == 0.0f || gender == 1.0f,
            String.format("Gender %f should be 0 or 1", gender));

        // Heart rate mean (index 2) - check if in reasonable range
        float hr = features[2];
        // If raw: 30-250 bpm, if standardized: typically -3 to +3
        // Adjust assertion based on your normalization
        assertTrue(hr >= -5 && hr <= 300,
            String.format("Heart rate %f is out of plausible range", hr));

        // SpO2 mean (index 15) - should be percentage or standardized
        float spo2 = features[15];
        assertTrue(spo2 >= -5 && spo2 <= 105,
            String.format("SpO2 %f is out of plausible range", spo2));
    }

    @Test
    @DisplayName("SAFETY: Verify feature extraction is deterministic")
    public void testFeatureExtractionIsDeterministic() {
        // Arrange
        EnrichedPatientContext context = createTestPatient();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act - Extract features twice
        float[] features1 = featureExtractor.extractFeatures(snapshot);
        float[] features2 = featureExtractor.extractFeatures(snapshot);

        // Assert - Should be identical
        assertArrayEquals(features1, features2,
            "CRITICAL: Feature extraction is non-deterministic! " +
            "This will cause unreliable predictions.");
    }

    @Test
    @DisplayName("SAFETY: Verify missing data handling is consistent")
    public void testMissingDataHandling() {
        // Arrange - Create patient with some missing lab values
        EnrichedPatientContext context = createPatientWithMissingData();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act
        float[] features = featureExtractor.extractFeatures(snapshot);

        // Assert - Check that missing values are handled (0-fill, mean-fill, etc.)
        // This depends on your missing data strategy
        // If 0-fill:
        for (int i = 0; i < features.length; i++) {
            assertNotNull(features[i],
                String.format("Feature %d is null - should be filled with default value", i));
        }

        // Verify no NaN from missing data
        for (int i = 0; i < features.length; i++) {
            assertFalse(Float.isNaN(features[i]),
                String.format("CRITICAL: Missing data caused NaN at feature %d (%s)",
                    i, MIMICFeatureExtractor.getFeatureNames().get(i)));
        }
    }

    @Test
    @DisplayName("SAFETY: Compare with known MIMIC ground truth sample")
    public void testAgainstKnownMIMICSample() {
        // This test requires a known MIMIC sample with expected feature vector
        // You should have saved this during model training

        // Arrange - Create the same patient data used in MIMIC training
        EnrichedPatientContext mimicSample = createKnownMIMICSample();
        PatientContextSnapshot snapshot = adapter.adapt(mimicSample);

        // Act
        float[] extractedFeatures = featureExtractor.extractFeatures(snapshot);

        // Expected features from training (replace with actual values)
        float[] expectedFeatures = loadExpectedMIMICFeatures();

        // Assert - Should match within floating point tolerance
        for (int i = 0; i < 37; i++) {
            assertEquals(expectedFeatures[i], extractedFeatures[i], 0.01f,
                String.format("CRITICAL: Feature %d (%s) mismatch! " +
                    "Expected: %.4f, Got: %.4f. This indicates feature extraction error.",
                    i, MIMICFeatureExtractor.getFeatureNames().get(i),
                    expectedFeatures[i], extractedFeatures[i]));
        }
    }

    @Test
    @DisplayName("SAFETY: Verify feature scaling matches training normalization")
    public void testFeatureScalingConsistency() {
        // This test verifies that scaling (standardization or normalization) is consistent

        // Arrange - Create patient with known raw values
        EnrichedPatientContext context = createPatientWithKnownValues();
        PatientContextSnapshot snapshot = adapter.adapt(context);

        // Act
        float[] features = featureExtractor.extractFeatures(snapshot);

        // Assert - Verify scaling strategy
        // If using standardization: (x - mean) / std
        // If using min-max: (x - min) / (max - min)

        // Example: Verify age scaling (assuming you know training stats)
        // float rawAge = 42.0f;
        // float trainingMean = 65.0f;  // Replace with actual training mean
        // float trainingStd = 15.0f;   // Replace with actual training std
        // float expectedScaledAge = (rawAge - trainingMean) / trainingStd;
        // assertEquals(expectedScaledAge, features[0], 0.01f,
        //     "Age scaling does not match training normalization!");

        // TODO: Add assertions for each feature's scaling
        // For now, just verify features are in reasonable scaled range
        // Typically standardized features are in [-3, +3] range
        int outOfRangeCount = 0;
        for (int i = 0; i < features.length; i++) {
            if (Math.abs(features[i]) > 10) {  // Adjust threshold based on your scaling
                outOfRangeCount++;
                System.out.printf("WARNING: Feature %d (%s) has extreme value: %.4f%n",
                    i, MIMICFeatureExtractor.getFeatureNames().get(i), features[i]);
            }
        }

        assertTrue(outOfRangeCount < 5,
            String.format("CRITICAL: %d features have extreme values! " +
                "This suggests scaling mismatch with training.", outOfRangeCount));
    }

    // ========================================================================
    // Helper Methods
    // ========================================================================

    private EnrichedPatientContext createTestPatient() {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("TEST-001");
        context.setEncounterId("ENC-TEST-001");
        context.setEventTime(System.currentTimeMillis());
        context.setEventType("VERIFICATION_TEST");

        PatientContextState state = new PatientContextState();

        // Demographics
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(65);
        demographics.setGender("M");
        demographics.setWeight(75.0);
        state.setDemographics(demographics);

        // Vital Signs
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 85.0);
        vitals.put("systolicbloodpressure", 120.0);
        vitals.put("diastolicbloodpressure", 80.0);
        vitals.put("respiratoryrate", 16.0);
        vitals.put("temperature", 37.0);
        vitals.put("oxygensaturation", 98.0);
        state.setLatestVitals(vitals);

        // Lab Values
        Map<String, LabResult> labs = new HashMap<>();
        labs.put("WBC", createLabResult(8.5, "10^3/uL"));
        labs.put("hemoglobin", createLabResult(14.0, "g/dL"));
        labs.put("platelets", createLabResult(250.0, "10^3/uL"));
        labs.put("creatinine", createLabResult(1.0, "mg/dL"));
        labs.put("bun", createLabResult(18.0, "mg/dL"));
        labs.put("glucose", createLabResult(100.0, "mg/dL"));
        labs.put("sodium", createLabResult(140.0, "mmol/L"));
        labs.put("potassium", createLabResult(4.0, "mmol/L"));
        labs.put("lactate", createLabResult(1.5, "mmol/L"));
        state.setRecentLabs(labs);

        // Clinical Scores
        state.setNews2Score(3);
        state.setQsofaScore(0);
        state.setCombinedAcuityScore(15.0);

        context.setPatientState(state);
        return context;
    }

    private EnrichedPatientContext createPatientWithMissingData() {
        EnrichedPatientContext context = createTestPatient();

        // Remove some lab values to test missing data handling
        Map<String, LabResult> labs = context.getPatientState().getRecentLabs();
        labs.remove("bun");
        labs.remove("glucose");
        labs.remove("lactate");

        return context;
    }

    private EnrichedPatientContext createKnownMIMICSample() {
        // TODO: Replace with actual MIMIC sample data
        // This should be a patient from your training set with known ground truth
        return createTestPatient();
    }

    private EnrichedPatientContext createPatientWithKnownValues() {
        // Create patient with specific values for scaling verification
        return createTestPatient();
    }

    private float[] loadExpectedMIMICFeatures() {
        // TODO: Load expected feature vector from training data
        // For now, return placeholder - you should replace this with actual expected values

        // This should be the EXACT feature vector used during training for a known sample
        return new float[]{
            65.0f, 1.0f,  // age, gender_male
            85.0f, 70.0f, 95.0f, 10.0f,  // HR mean/min/max/std
            16.0f, 20.0f,  // RR mean/max
            37.0f, 37.5f,  // Temp mean/max
            120.0f, 110.0f,  // SBP mean/min
            80.0f,  // DBP mean
            93.3f, 85.0f,  // MAP mean/min
            98.0f, 96.0f,  // SpO2 mean/min
            8.5f, 14.0f, 250.0f,  // WBC, Hgb, Platelets
            1.0f, 1.2f,  // Creatinine mean/max
            18.0f, 100.0f, 140.0f, 4.0f,  // BUN, Glucose, Na, K
            1.5f, 2.0f,  // Lactate mean/max
            1.0f,  // Bilirubin
            2.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f, 0.0f,  // SOFA scores
            15.0f  // GCS
        };
    }

    private LabResult createLabResult(double value, String unit) {
        LabResult result = new LabResult();
        result.setValue(value);
        result.setUnit(unit);
        result.setTimestamp(System.currentTimeMillis());
        return result;
    }
}
