package com.cardiofit.flink.ml.util;

import ai.onnxruntime.OrtEnvironment;
import ai.onnxruntime.OrtException;
import ai.onnxruntime.OrtSession;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.*;
import org.mockito.Mockito;

import java.util.*;

/**
 * Test Data Factory for Module 5 ML Inference Tests
 *
 * Provides convenient methods to generate mock data for testing ML inference components.
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class TestDataFactory {

    // ===== PatientContextSnapshot Generation =====

    /**
     * Create a complete patient context snapshot for testing
     */
    public static PatientContextSnapshot createPatientContext() {
        return createPatientContext("patient-001", true);
    }

    /**
     * Create patient context with specific ID and clinical state
     */
    public static PatientContextSnapshot createPatientContext(String patientId, boolean highRisk) {
        PatientContextSnapshot context = new PatientContextSnapshot();
        context.setPatientId(patientId);
        context.setEncounterId("encounter-001");

        // Demographics - use actual field setters
        context.setAge(highRisk ? 72 : 45);
        context.setGender("male");
        context.setBmi(highRisk ? 32.0 : 24.0);
        context.setCurrentLocation(highRisk ? "ICU" : "WARD");
        context.setAdmissionType(highRisk ? "emergency" : "elective");
        context.setHoursFromAdmission(highRisk ? 48L : 6L);

        // Vitals - set individual fields
        context.setHeartRate(highRisk ? 115.0 : 75.0);
        context.setSystolicBP(highRisk ? 85.0 : 120.0);
        context.setDiastolicBP(highRisk ? 55.0 : 80.0);
        context.setRespiratoryRate(highRisk ? 24.0 : 16.0);
        context.setTemperature(highRisk ? 38.5 : 37.0);
        context.setOxygenSaturation(highRisk ? 92.0 : 98.0);

        // Calculate derived metrics (MAP, shock index, pulse width)
        context.calculateDerivedMetrics();

        // Labs - set individual fields
        context.setLactate(highRisk ? 4.5 : 1.2);
        context.setCreatinine(highRisk ? 2.1 : 0.9);
        context.setBun(highRisk ? 35.0 : 12.0);
        context.setSodium(138.0);
        context.setPotassium(4.2);
        context.setChloride(102.0);
        context.setBicarbonate(highRisk ? 18.0 : 24.0);
        context.setWhiteBloodCells(highRisk ? 16.0 : 7.5);
        context.setHemoglobin(highRisk ? 9.5 : 13.5);
        context.setPlatelets(highRisk ? 90.0 : 220.0);
        context.setAst(highRisk ? 120.0 : 25.0);
        context.setAlt(highRisk ? 110.0 : 28.0);
        context.setBilirubin(highRisk ? 2.5 : 0.7);

        // Clinical Scores - use correct method names and Integer type
        context.setNews2Score(highRisk ? 9 : 2);
        context.setQsofaScore(highRisk ? 2 : 0);
        context.setSofaScore(highRisk ? 6 : 1);
        context.setApacheScore(highRisk ? 22 : 8);
        context.setCharlsonIndex(highRisk ? 5 : 1);

        // Medications - use correct method names
        context.setOnVasopressors(highRisk);
        context.setOnAntibiotics(highRisk);
        context.setOnAnticoagulants(false);
        context.setOnSedatives(highRisk);
        context.setOnInsulin(false);

        // Comorbidities - set individual flags
        if (highRisk) {
            context.setHasDiabetes(true);
            context.setHasHypertension(true);
            context.setHasChronicKidneyDisease(true);
            context.setHasHeartFailure(true);
            context.setHasCopd(true);
        } else {
            context.setHasHypertension(true);
        }

        // Vital trends - set 6h change fields directly
        if (highRisk) {
            context.setHeartRateChange6h(15.0);  // HR increasing
            context.setBpChange6h(-25.0);        // BP decreasing
            context.setLactateChange6h(2.5);     // Lactate increasing
        }

        return context;
    }

    // ===== MLPrediction Generation =====

    /**
     * Create ML prediction with SHAP explanation
     */
    public static MLPrediction createMLPrediction() {
        return createMLPrediction("patient-001", 0.82, true);
    }

    /**
     * Create ML prediction with specific score and SHAP data
     */
    public static MLPrediction createMLPrediction(String patientId, double score, boolean includeShap) {
        MLPrediction prediction = new MLPrediction();
        prediction.setId(UUID.randomUUID().toString());
        prediction.setPatientId(patientId);
        prediction.setEncounterId("encounter-001");
        prediction.setModelName("Sepsis Onset Predictor");
        prediction.setModelType("SEPSIS_ONSET");
        prediction.setPredictionTime(System.currentTimeMillis());
        prediction.setInputFeatureCount(70);

        // Prediction scores
        Map<String, Double> scores = new HashMap<>();
        scores.put("primary_score", score);
        scores.put("confidence_score", 0.91);
        prediction.setPredictionScores(scores);

        // Risk level
        if (score >= 0.8) {
            prediction.setRiskLevel("HIGH");
        } else if (score >= 0.5) {
            prediction.setRiskLevel("MODERATE");
        } else {
            prediction.setRiskLevel("LOW");
        }

        prediction.setConfidence(0.91);

        // Model metadata
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("model_id", "sepsis_v1");
        metadata.put("model_version", "1.0.0");
        metadata.put("inference_time_ms", 15L);
        metadata.put("feature_count", 70);
        prediction.setModelMetadata(metadata);

        // SHAP explanation
        if (includeShap) {
            MLPrediction.ExplainabilityData explainability = new MLPrediction.ExplainabilityData();

            Map<String, Double> shapValues = new LinkedHashMap<>();
            shapValues.put("lab_lactate_mmol", 0.15);
            shapValues.put("vital_heart_rate", 0.12);
            shapValues.put("vital_systolic_bp", -0.08);
            shapValues.put("lab_wbc_k_ul", 0.09);
            shapValues.put("score_sofa", 0.11);
            explainability.setShapValues(shapValues);

            List<String> topContributors = Arrays.asList(
                "lab_lactate_mmol (+0.15)",
                "vital_heart_rate (+0.12)",
                "score_sofa (+0.11)"
            );
            explainability.setTopContributors(topContributors);

            explainability.setExplanationText(
                "Model predicts HIGH risk (score: 0.820). Key contributing factors:\n" +
                "1. Elevated lactate (4.5 mmol/L) increased risk by 0.150\n" +
                "2. High heart rate (115 bpm) increased risk by 0.120"
            );
            explainability.setExplainabilityMethod("Kernel SHAP");

            prediction.setExplainabilityData(explainability);
        }

        return prediction;
    }

    // ===== PatternEvent Generation =====

    /**
     * Create CEP pattern event for testing
     */
    public static PatternEvent createPatternEvent() {
        return createPatternEvent("patient-001", "sepsis_pattern", "HIGH");
    }

    /**
     * Create pattern event with specific attributes
     */
    public static PatternEvent createPatternEvent(String patientId, String patternName, String severity) {
        PatternEvent event = new PatternEvent();
        event.setPatientId(patientId);
        event.setPatternType(patternName);  // Use setPatternType instead of setPatternName
        event.setSeverity(severity);
        event.setConfidence(0.88);
        event.setDetectionTime(System.currentTimeMillis());  // Use setDetectionTime instead of setTimestamp

        // Set pattern details
        Map<String, Object> patternDetails = new HashMap<>();
        patternDetails.put("trigger_event", "lactate_elevation");
        patternDetails.put("pattern_duration_ms", 1800000L);
        patternDetails.put("deterioration_pattern", patternName.contains("deterioration"));
        event.setPatternDetails(patternDetails);  // Use setPatternDetails instead of setPatternData

        return event;
    }

    // ===== SemanticEvent Generation =====

    /**
     * Create semantic event from Module 3
     */
    public static SemanticEvent createSemanticEvent() {
        return createSemanticEvent("patient-001", 75.0);
    }

    /**
     * Create semantic event with specific clinical significance
     */
    public static SemanticEvent createSemanticEvent(String patientId, double clinicalSignificance) {
        SemanticEvent event = new SemanticEvent();
        event.setPatientId(patientId);
        event.setEventType(EventType.LAB_RESULT);  // Use EventType enum instead of String
        event.setEventTime(System.currentTimeMillis());  // Use setEventTime instead of setTimestamp

        // Create PatientContext (not PatientContextSnapshot) for SemanticEvent
        PatientContext context = new PatientContext();
        context.setPatientId(patientId);

        // Add clinical significance to enrichment data map
        Map<String, Object> enrichmentData = new HashMap<>();
        enrichmentData.put("clinical_significance", clinicalSignificance);
        event.setEnrichmentData(enrichmentData);

        event.setPatientContext(context);

        return event;
    }

    // ===== ClinicalFeatureVector Generation =====

    /**
     * Create clinical feature vector with all 70 features
     */
    public static ClinicalFeatureVector createFeatureVector() {
        return createFeatureVector("patient-001", true);
    }

    /**
     * Create feature vector with specific characteristics
     */
    public static ClinicalFeatureVector createFeatureVector(String patientId, boolean highRisk) {
        Map<String, Double> features = new LinkedHashMap<>();

        // Demographics (5)
        features.put("demo_age_years", highRisk ? 72.0 : 45.0);
        features.put("demo_gender_male", 1.0);
        features.put("demo_bmi", highRisk ? 32.0 : 24.0);
        features.put("demo_icu_patient", highRisk ? 1.0 : 0.0);
        features.put("demo_admission_emergency", highRisk ? 1.0 : 0.0);

        // Vitals (12)
        features.put("vital_heart_rate", highRisk ? 115.0 : 75.0);
        features.put("vital_systolic_bp", highRisk ? 85.0 : 120.0);
        features.put("vital_diastolic_bp", highRisk ? 55.0 : 80.0);
        features.put("vital_respiratory_rate", highRisk ? 24.0 : 16.0);
        features.put("vital_temperature_c", highRisk ? 38.5 : 37.0);
        features.put("vital_oxygen_saturation", highRisk ? 92.0 : 98.0);
        features.put("vital_mean_arterial_pressure", highRisk ? 65.0 : 93.0);
        features.put("vital_pulse_pressure", highRisk ? 30.0 : 40.0);
        features.put("vital_shock_index", highRisk ? 1.35 : 0.63);
        features.put("vital_hr_abnormal", highRisk ? 1.0 : 0.0);
        features.put("vital_bp_hypotensive", highRisk ? 1.0 : 0.0);
        features.put("vital_fever", highRisk ? 1.0 : 0.0);

        // Labs (15)
        features.put("lab_lactate_mmol", highRisk ? 4.5 : 1.2);
        features.put("lab_creatinine_mg_dl", highRisk ? 2.1 : 0.9);
        features.put("lab_bun_mg_dl", highRisk ? 35.0 : 12.0);
        features.put("lab_sodium_meq", 138.0);
        features.put("lab_potassium_meq", 4.2);
        features.put("lab_chloride_meq", 102.0);
        features.put("lab_bicarbonate_meq", highRisk ? 18.0 : 24.0);
        features.put("lab_wbc_k_ul", highRisk ? 16.0 : 7.5);
        features.put("lab_hemoglobin_g_dl", highRisk ? 9.5 : 13.5);
        features.put("lab_platelets_k_ul", highRisk ? 90.0 : 220.0);
        features.put("lab_ast_u_l", highRisk ? 120.0 : 25.0);
        features.put("lab_alt_u_l", highRisk ? 110.0 : 28.0);
        features.put("lab_bilirubin_mg_dl", highRisk ? 2.5 : 0.7);
        features.put("lab_lactate_elevated", highRisk ? 1.0 : 0.0);
        features.put("lab_aki_present", highRisk ? 1.0 : 0.0);

        // Clinical Scores (5)
        features.put("score_news2", highRisk ? 9.0 : 2.0);
        features.put("score_qsofa", highRisk ? 2.0 : 0.0);
        features.put("score_sofa", highRisk ? 6.0 : 1.0);
        features.put("score_apache", highRisk ? 22.0 : 8.0);
        features.put("score_acuity_combined", highRisk ? 85.0 : 35.0);

        // Temporal (10)
        features.put("temporal_hours_since_admission", highRisk ? 48.0 : 6.0);
        features.put("temporal_hours_since_last_vitals", 0.083); // 5 minutes
        features.put("temporal_hours_since_last_labs", 0.5); // 30 minutes
        features.put("temporal_length_of_stay_hours", highRisk ? 48.0 : 6.0);
        features.put("temporal_hr_trend_increasing", highRisk ? 1.0 : 0.0);
        features.put("temporal_bp_trend_decreasing", highRisk ? 1.0 : 0.0);
        features.put("temporal_lactate_trend_increasing", highRisk ? 1.0 : 0.0);
        features.put("temporal_hour_of_day", 14.0); // 2 PM
        features.put("temporal_is_night_shift", 0.0);
        features.put("temporal_is_weekend", 0.0);

        // Medications (8)
        features.put("med_total_count", highRisk ? 12.0 : 3.0);
        features.put("med_high_risk_count", highRisk ? 4.0 : 0.0);
        features.put("med_vasopressor_active", highRisk ? 1.0 : 0.0);
        features.put("med_antibiotic_active", highRisk ? 1.0 : 0.0);
        features.put("med_anticoagulation_active", 0.0);
        features.put("med_sedation_active", highRisk ? 1.0 : 0.0);
        features.put("med_insulin_active", 0.0);
        features.put("med_polypharmacy", highRisk ? 1.0 : 0.0);

        // Comorbidities (10)
        features.put("comorbid_diabetes", highRisk ? 1.0 : 0.0);
        features.put("comorbid_hypertension", 1.0);
        features.put("comorbid_ckd", highRisk ? 1.0 : 0.0);
        features.put("comorbid_heart_failure", highRisk ? 1.0 : 0.0);
        features.put("comorbid_copd", highRisk ? 1.0 : 0.0);
        features.put("comorbid_cancer", 0.0);
        features.put("comorbid_immunosuppressed", 0.0);
        features.put("comorbid_liver_disease", 0.0);
        features.put("comorbid_stroke_history", 0.0);
        features.put("comorbid_charlson_index", highRisk ? 5.0 : 1.0);

        // CEP Patterns (5)
        features.put("pattern_sepsis_detected", highRisk ? 1.0 : 0.0);
        features.put("pattern_deterioration_detected", highRisk ? 1.0 : 0.0);
        features.put("pattern_aki_detected", highRisk ? 1.0 : 0.0);
        features.put("pattern_confidence_score", highRisk ? 0.88 : 0.0);
        features.put("pattern_clinical_significance", highRisk ? 75.0 : 35.0);

        return ClinicalFeatureVector.builder()
            .patientId(patientId)
            .features(features)
            .featureCount(70)
            .extractionTimestamp(System.currentTimeMillis())
            .extractionTimeMs(2.5)
            .build();
    }

    // ===== Mock ONNX Model =====

    /**
     * Create mock ONNX model container for testing (no actual inference)
     */
    public static ONNXModelContainer createMockONNXModel() throws OrtException {
        List<String> featureNames = createFeatureNames();

        ModelConfig config = ModelConfig.builder()
            .modelPath("/models/sepsis_v1.onnx")
            .predictionThreshold(0.7)
            .intraOpThreads(2)
            .interOpThreads(2)
            .build();

        return ONNXModelContainer.builder()
            .modelId("sepsis_v1")
            .modelName("Sepsis Onset Predictor")
            .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
            .modelVersion("1.0.0")
            .inputFeatureNames(featureNames)
            .outputNames(Arrays.asList("sepsis_probability"))
            .config(config)
            .build();
    }

    /**
     * Create list of all 70 feature names
     */
    public static List<String> createFeatureNames() {
        List<String> names = new ArrayList<>(70);

        // Demographics (5)
        names.add("demo_age_years");
        names.add("demo_gender_male");
        names.add("demo_bmi");
        names.add("demo_icu_patient");
        names.add("demo_admission_emergency");

        // Vitals (12)
        names.add("vital_heart_rate");
        names.add("vital_systolic_bp");
        names.add("vital_diastolic_bp");
        names.add("vital_respiratory_rate");
        names.add("vital_temperature_c");
        names.add("vital_oxygen_saturation");
        names.add("vital_mean_arterial_pressure");
        names.add("vital_pulse_pressure");
        names.add("vital_shock_index");
        names.add("vital_hr_abnormal");
        names.add("vital_bp_hypotensive");
        names.add("vital_fever");

        // Labs (15)
        names.add("lab_lactate_mmol");
        names.add("lab_creatinine_mg_dl");
        names.add("lab_bun_mg_dl");
        names.add("lab_sodium_meq");
        names.add("lab_potassium_meq");
        names.add("lab_chloride_meq");
        names.add("lab_bicarbonate_meq");
        names.add("lab_wbc_k_ul");
        names.add("lab_hemoglobin_g_dl");
        names.add("lab_platelets_k_ul");
        names.add("lab_ast_u_l");
        names.add("lab_alt_u_l");
        names.add("lab_bilirubin_mg_dl");
        names.add("lab_lactate_elevated");
        names.add("lab_aki_present");

        // Clinical Scores (5)
        names.add("score_news2");
        names.add("score_qsofa");
        names.add("score_sofa");
        names.add("score_apache");
        names.add("score_acuity_combined");

        // Temporal (10)
        names.add("temporal_hours_since_admission");
        names.add("temporal_hours_since_last_vitals");
        names.add("temporal_hours_since_last_labs");
        names.add("temporal_length_of_stay_hours");
        names.add("temporal_hr_trend_increasing");
        names.add("temporal_bp_trend_decreasing");
        names.add("temporal_lactate_trend_increasing");
        names.add("temporal_hour_of_day");
        names.add("temporal_is_night_shift");
        names.add("temporal_is_weekend");

        // Medications (8)
        names.add("med_total_count");
        names.add("med_high_risk_count");
        names.add("med_vasopressor_active");
        names.add("med_antibiotic_active");
        names.add("med_anticoagulation_active");
        names.add("med_sedation_active");
        names.add("med_insulin_active");
        names.add("med_polypharmacy");

        // Comorbidities (10)
        names.add("comorbid_diabetes");
        names.add("comorbid_hypertension");
        names.add("comorbid_ckd");
        names.add("comorbid_heart_failure");
        names.add("comorbid_copd");
        names.add("comorbid_cancer");
        names.add("comorbid_immunosuppressed");
        names.add("comorbid_liver_disease");
        names.add("comorbid_stroke_history");
        names.add("comorbid_charlson_index");

        // CEP Patterns (5)
        names.add("pattern_sepsis_detected");
        names.add("pattern_deterioration_detected");
        names.add("pattern_aki_detected");
        names.add("pattern_confidence_score");
        names.add("pattern_clinical_significance");

        return names;
    }

    // ===== Feature Array Generation =====

    /**
     * Create float array of 70 features for inference
     */
    public static float[] createFeatureArray(boolean highRisk) {
        ClinicalFeatureVector vector = createFeatureVector("patient-001", highRisk);
        return vector.toFloatArray();
    }

    /**
     * Create batch of feature arrays
     */
    public static List<float[]> createFeatureBatch(int batchSize, boolean highRisk) {
        List<float[]> batch = new ArrayList<>(batchSize);
        for (int i = 0; i < batchSize; i++) {
            batch.add(createFeatureArray(highRisk));
        }
        return batch;
    }
}
