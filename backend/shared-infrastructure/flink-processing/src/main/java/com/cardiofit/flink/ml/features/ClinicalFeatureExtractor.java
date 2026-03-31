package com.cardiofit.flink.ml.features;

import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.PatternEvent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Duration;
import java.util.*;

/**
 * Clinical Feature Extraction Pipeline
 *
 * Extracts 70 clinical features from patient state, semantic events, and pattern events
 * for ML model inference. Features span demographics, vitals, labs, clinical scores,
 * temporal patterns, medications, comorbidities, and CEP-detected patterns.
 *
 * Feature Categories:
 * - Demographics (5): age, gender, BMI, ICU status, admission source
 * - Vitals (12): HR, BP, RR, temp, O2, derived metrics (MAP, shock index)
 * - Labs (15): lactate, creatinine, BUN, electrolytes, CBC, LFTs
 * - Clinical Scores (5): NEWS2, qSOFA, SOFA, APACHE, combined acuity
 * - Temporal (10): time since admission, vitals/labs recency, trends
 * - Medications (8): vasopressors, antibiotics, anticoagulation, counts
 * - Comorbidities (10): diabetes, CKD, heart failure, COPD, cancer
 * - CEP Patterns (5): sepsis pattern, deterioration, AKI, confidence
 *
 * TOTAL: 70 features
 *
 * Usage:
 * <pre>
 * ClinicalFeatureExtractor extractor = new ClinicalFeatureExtractor();
 * ClinicalFeatureVector features = extractor.extract(
 *     patientContext,
 *     semanticEvent,
 *     patternEvent
 * );
 * float[] featureArray = features.toFloatArray();
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ClinicalFeatureExtractor implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ClinicalFeatureExtractor.class);

    // Feature extraction configuration
    private final FeatureExtractionConfig config;

    public ClinicalFeatureExtractor() {
        this(FeatureExtractionConfig.createDefault());
    }

    public ClinicalFeatureExtractor(FeatureExtractionConfig config) {
        this.config = config;
    }

    /**
     * Extract all 70 clinical features from patient state
     *
     * @param patientContext Patient context snapshot from Module 2
     * @param semanticEvent Latest semantic event from Module 3
     * @param patternEvent Latest pattern event from Module 4
     * @return Clinical feature vector with 70 features
     */
    public ClinicalFeatureVector extract(
            PatientContextSnapshot patientContext,
            SemanticEvent semanticEvent,
            PatternEvent patternEvent) {

        long startTime = System.nanoTime();

        try {
            Map<String, Double> features = new LinkedHashMap<>(70);

            // Category 1: Demographics (5 features)
            extractDemographics(patientContext, features);

            // Category 2: Vitals (12 features)
            extractVitals(patientContext, features);

            // Category 3: Labs (15 features)
            extractLabs(patientContext, features);

            // Category 4: Clinical Scores (5 features)
            extractClinicalScores(patientContext, semanticEvent, features);

            // Category 5: Temporal (10 features)
            extractTemporal(patientContext, semanticEvent, features);

            // Category 6: Medications (8 features)
            extractMedications(patientContext, features);

            // Category 7: Comorbidities (10 features)
            extractComorbidities(patientContext, features);

            // Category 8: CEP Patterns (5 features)
            extractCEPPatterns(patternEvent, semanticEvent, features);

            long extractionTimeNs = System.nanoTime() - startTime;
            double extractionTimeMs = extractionTimeNs / 1_000_000.0;

            LOG.debug("Extracted {} features in {:.2f}ms for patient: {}",
                features.size(), extractionTimeMs,
                patientContext != null ? patientContext.getPatientId() : "unknown");

            return ClinicalFeatureVector.builder()
                .patientId(patientContext != null ? patientContext.getPatientId() : "unknown")
                .features(features)
                .featureCount(features.size())
                .extractionTimestamp(System.currentTimeMillis())
                .extractionTimeMs(extractionTimeMs)
                .build();

        } catch (Exception e) {
            LOG.error("Feature extraction failed", e);
            return createEmptyFeatureVector(patientContext);
        }
    }

    // ===== CATEGORY 1: DEMOGRAPHICS (5 features) =====

    private void extractDemographics(PatientContextSnapshot context, Map<String, Double> features) {
        if (context == null) {
            features.put("demo_age_years", 0.0);
            features.put("demo_gender_male", 0.0);
            features.put("demo_bmi", 0.0);
            features.put("demo_icu_patient", 0.0);
            features.put("demo_admission_emergency", 0.0);
            return;
        }

        // Age (continuous)
        features.put("demo_age_years", context.getAgeYears() != null ? context.getAgeYears() : 0.0);

        // Gender (binary: 1=male, 0=female)
        features.put("demo_gender_male",
            "male".equalsIgnoreCase(context.getGender()) ? 1.0 : 0.0);

        // BMI (continuous)
        features.put("demo_bmi", context.getBMI() != null ? context.getBMI() : 25.0);

        // ICU patient status (binary)
        features.put("demo_icu_patient", context.isICUPatient() ? 1.0 : 0.0);

        // Emergency admission (binary)
        features.put("demo_admission_emergency",
            "emergency".equalsIgnoreCase(context.getAdmissionSource()) ? 1.0 : 0.0);
    }

    // ===== CATEGORY 2: VITALS (12 features) =====

    private void extractVitals(PatientContextSnapshot context, Map<String, Double> features) {
        if (context == null || context.getLatestVitals() == null) {
            // Set defaults for all vital features
            features.put("vital_heart_rate", 80.0);
            features.put("vital_systolic_bp", 120.0);
            features.put("vital_diastolic_bp", 80.0);
            features.put("vital_respiratory_rate", 16.0);
            features.put("vital_temperature_c", 37.0);
            features.put("vital_oxygen_saturation", 98.0);
            features.put("vital_mean_arterial_pressure", 93.0);
            features.put("vital_pulse_pressure", 40.0);
            features.put("vital_shock_index", 0.67);
            features.put("vital_hr_abnormal", 0.0);
            features.put("vital_bp_hypotensive", 0.0);
            features.put("vital_fever", 0.0);
            return;
        }

        // Convert Map<String, Double> to Map<String, Object> for helper method
        Map<String, Object> vitals = new HashMap<>(context.getLatestVitals());

        // Normalize vital keys to handle both production (heartrate) and SemanticEvent (heart_rate) formats
        Map<String, Object> normalized = new HashMap<>(vitals);
        Map<String, String> aliases = Map.of(
            "heartrate", "heart_rate", "systolicbloodpressure", "systolic_bp",
            "diastolicbloodpressure", "diastolic_bp", "respiratoryrate", "respiratory_rate",
            "oxygensaturation", "oxygen_saturation"
        );
        for (Map.Entry<String, Object> e : vitals.entrySet()) {
            String alias = aliases.get(e.getKey());
            if (alias != null) normalized.put(alias, e.getValue());
        }
        vitals = normalized;

        // Primary vitals
        double hr = getDoubleValue(vitals, "heart_rate", 80.0);
        double sysBP = getDoubleValue(vitals, "systolic_bp", 120.0);
        double diasBP = getDoubleValue(vitals, "diastolic_bp", 80.0);
        double rr = getDoubleValue(vitals, "respiratory_rate", 16.0);
        double temp = getDoubleValue(vitals, "temperature", 37.0);
        double o2Sat = getDoubleValue(vitals, "oxygen_saturation", 98.0);

        features.put("vital_heart_rate", hr);
        features.put("vital_systolic_bp", sysBP);
        features.put("vital_diastolic_bp", diasBP);
        features.put("vital_respiratory_rate", rr);
        features.put("vital_temperature_c", temp);
        features.put("vital_oxygen_saturation", o2Sat);

        // Derived metrics
        double map = (sysBP + 2 * diasBP) / 3.0;  // Mean Arterial Pressure
        double pulsePressure = sysBP - diasBP;
        double shockIndex = hr / sysBP;  // Shock index (HR/SBP)

        features.put("vital_mean_arterial_pressure", map);
        features.put("vital_pulse_pressure", pulsePressure);
        features.put("vital_shock_index", shockIndex);

        // Clinical flags
        features.put("vital_hr_abnormal", (hr < 60 || hr > 100) ? 1.0 : 0.0);
        features.put("vital_bp_hypotensive", sysBP < 90 ? 1.0 : 0.0);
        features.put("vital_fever", temp >= 38.0 ? 1.0 : 0.0);
    }

    // ===== CATEGORY 3: LABS (15 features) =====

    private void extractLabs(PatientContextSnapshot context, Map<String, Double> features) {
        if (context == null || context.getLatestLabs() == null) {
            // Set defaults for all lab features
            features.put("lab_lactate_mmol", 1.0);
            features.put("lab_creatinine_mg_dl", 1.0);
            features.put("lab_bun_mg_dl", 15.0);
            features.put("lab_sodium_meq", 140.0);
            features.put("lab_potassium_meq", 4.0);
            features.put("lab_chloride_meq", 100.0);
            features.put("lab_bicarbonate_meq", 24.0);
            features.put("lab_wbc_k_ul", 8.0);
            features.put("lab_hemoglobin_g_dl", 13.0);
            features.put("lab_platelets_k_ul", 200.0);
            features.put("lab_ast_u_l", 30.0);
            features.put("lab_alt_u_l", 30.0);
            features.put("lab_bilirubin_mg_dl", 0.8);
            features.put("lab_lactate_elevated", 0.0);
            features.put("lab_aki_present", 0.0);
            return;
        }

        // Convert Map<String, Double> to Map<String, Object> for helper method
        Map<String, Object> labs = new HashMap<>(context.getLatestLabs());

        // Lactate (sepsis marker)
        double lactate = getDoubleValue(labs, "lactate", 1.0);
        features.put("lab_lactate_mmol", lactate);
        features.put("lab_lactate_elevated", lactate > 2.0 ? 1.0 : 0.0);

        // Kidney function
        double creatinine = getDoubleValue(labs, "creatinine", 1.0);
        double bun = getDoubleValue(labs, "bun", 15.0);
        features.put("lab_creatinine_mg_dl", creatinine);
        features.put("lab_bun_mg_dl", bun);
        features.put("lab_aki_present", creatinine > 1.5 ? 1.0 : 0.0);

        // Electrolytes
        features.put("lab_sodium_meq", getDoubleValue(labs, "sodium", 140.0));
        features.put("lab_potassium_meq", getDoubleValue(labs, "potassium", 4.0));
        features.put("lab_chloride_meq", getDoubleValue(labs, "chloride", 100.0));
        features.put("lab_bicarbonate_meq", getDoubleValue(labs, "bicarbonate", 24.0));

        // CBC
        features.put("lab_wbc_k_ul", getDoubleValue(labs, "wbc", 8.0));
        features.put("lab_hemoglobin_g_dl", getDoubleValue(labs, "hemoglobin", 13.0));
        features.put("lab_platelets_k_ul", getDoubleValue(labs, "platelets", 200.0));

        // Liver function
        features.put("lab_ast_u_l", getDoubleValue(labs, "ast", 30.0));
        features.put("lab_alt_u_l", getDoubleValue(labs, "alt", 30.0));
        features.put("lab_bilirubin_mg_dl", getDoubleValue(labs, "bilirubin", 0.8));
    }

    // ===== CATEGORY 4: CLINICAL SCORES (5 features) =====

    private void extractClinicalScores(PatientContextSnapshot context,
                                       SemanticEvent semanticEvent,
                                       Map<String, Double> features) {
        if (context == null) {
            features.put("score_news2", 0.0);
            features.put("score_qsofa", 0.0);
            features.put("score_sofa", 0.0);
            features.put("score_apache", 0.0);
            features.put("score_acuity_combined", 0.0);
            return;
        }

        // NEWS2 (National Early Warning Score 2)
        double news2 = context.getNEWS2Score() != null ? context.getNEWS2Score() : 0.0;
        features.put("score_news2", news2);

        // qSOFA (quick Sequential Organ Failure Assessment)
        double qsofa = context.getQSOFAScore() != null ? context.getQSOFAScore() : 0.0;
        features.put("score_qsofa", qsofa);

        // SOFA (Sequential Organ Failure Assessment)
        double sofa = context.getSOFAScore() != null ? context.getSOFAScore() : 0.0;
        features.put("score_sofa", sofa);

        // APACHE (Acute Physiology and Chronic Health Evaluation)
        double apache = context.getAPACHEScore() != null ? context.getAPACHEScore() : 0.0;
        features.put("score_apache", apache);

        // Combined acuity score from semantic event
        double acuity = 0.0;
        if (semanticEvent != null && semanticEvent.getPatientContext() != null) {
            acuity = semanticEvent.getPatientContext().getAcuityScore();
        } else if (context.getAcuityScore() != null) {
            acuity = context.getAcuityScore();
        }
        features.put("score_acuity_combined", acuity);
    }

    // ===== CATEGORY 5: TEMPORAL (10 features) =====

    private void extractTemporal(PatientContextSnapshot context,
                                 SemanticEvent semanticEvent,
                                 Map<String, Double> features) {
        long currentTime = System.currentTimeMillis();

        if (context == null) {
            // Set defaults
            features.put("temporal_hours_since_admission", 0.0);
            features.put("temporal_hours_since_last_vitals", 0.0);
            features.put("temporal_hours_since_last_labs", 0.0);
            features.put("temporal_length_of_stay_hours", 0.0);
            features.put("temporal_hr_trend_increasing", 0.0);
            features.put("temporal_bp_trend_decreasing", 0.0);
            features.put("temporal_lactate_trend_increasing", 0.0);
            features.put("temporal_hour_of_day", (double) ((currentTime / (1000 * 3600)) % 24));
            features.put("temporal_is_night_shift", 0.0);
            features.put("temporal_is_weekend", 0.0);
            return;
        }

        // Time since admission
        double hoursSinceAdmission = 0.0;
        if (context.getAdmissionTime() != null) {
            hoursSinceAdmission = (currentTime - context.getAdmissionTime().toEpochMilli()) / (1000.0 * 3600.0);
        }
        features.put("temporal_hours_since_admission", hoursSinceAdmission);

        // Time since last measurements
        double hoursSinceVitals = 0.0;
        if (context.getLastVitalsTimestamp() != null) {
            hoursSinceVitals = (currentTime - context.getLastVitalsTimestamp().toEpochMilli()) / (1000.0 * 3600.0);
        }
        features.put("temporal_hours_since_last_vitals", hoursSinceVitals);

        double hoursSinceLabs = 0.0;
        if (context.getLastLabsTimestamp() != null) {
            hoursSinceLabs = (currentTime - context.getLastLabsTimestamp().toEpochMilli()) / (1000.0 * 3600.0);
        }
        features.put("temporal_hours_since_last_labs", hoursSinceLabs);

        // Length of stay - convert Long to Double
        Double lengthOfStay = context.getLengthOfStayHours() != null ?
            context.getLengthOfStayHours().doubleValue() : hoursSinceAdmission;
        features.put("temporal_length_of_stay_hours", lengthOfStay);

        // Trends (from semantic event or context)
        features.put("temporal_hr_trend_increasing",
            context.isHRTrendIncreasing() ? 1.0 : 0.0);
        features.put("temporal_bp_trend_decreasing",
            context.isBPTrendDecreasing() ? 1.0 : 0.0);
        features.put("temporal_lactate_trend_increasing",
            context.isLactateTrendIncreasing() ? 1.0 : 0.0);

        // Time of day features
        Calendar cal = Calendar.getInstance();
        cal.setTimeInMillis(currentTime);
        int hourOfDay = cal.get(Calendar.HOUR_OF_DAY);
        boolean isNightShift = (hourOfDay >= 19 || hourOfDay < 7);
        boolean isWeekend = (cal.get(Calendar.DAY_OF_WEEK) == Calendar.SATURDAY ||
                           cal.get(Calendar.DAY_OF_WEEK) == Calendar.SUNDAY);

        features.put("temporal_hour_of_day", (double) hourOfDay);
        features.put("temporal_is_night_shift", isNightShift ? 1.0 : 0.0);
        features.put("temporal_is_weekend", isWeekend ? 1.0 : 0.0);
    }

    // ===== CATEGORY 6: MEDICATIONS (8 features) =====

    private void extractMedications(PatientContextSnapshot context, Map<String, Double> features) {
        if (context == null) {
            features.put("med_total_count", 0.0);
            features.put("med_high_risk_count", 0.0);
            features.put("med_vasopressor_active", 0.0);
            features.put("med_antibiotic_active", 0.0);
            features.put("med_anticoagulation_active", 0.0);
            features.put("med_sedation_active", 0.0);
            features.put("med_insulin_active", 0.0);
            features.put("med_polypharmacy", 0.0);
            return;
        }

        // Total medication count
        int totalMeds = context.getActiveMedicationCount() != null ?
            context.getActiveMedicationCount() : 0;
        features.put("med_total_count", (double) totalMeds);

        // High-risk medications
        int highRiskCount = context.getHighRiskMedicationCount() != null ?
            context.getHighRiskMedicationCount() : 0;
        features.put("med_high_risk_count", (double) highRiskCount);

        // Specific medication classes (binary)
        features.put("med_vasopressor_active",
            context.isOnVasopressors() ? 1.0 : 0.0);
        features.put("med_antibiotic_active",
            context.isOnAntibiotics() ? 1.0 : 0.0);
        features.put("med_anticoagulation_active",
            context.isOnAnticoagulation() ? 1.0 : 0.0);
        features.put("med_sedation_active",
            context.isOnSedation() ? 1.0 : 0.0);
        features.put("med_insulin_active",
            context.isOnInsulin() ? 1.0 : 0.0);

        // Polypharmacy flag (≥5 medications)
        features.put("med_polypharmacy", totalMeds >= 5 ? 1.0 : 0.0);
    }

    // ===== CATEGORY 7: COMORBIDITIES (10 features) =====

    private void extractComorbidities(PatientContextSnapshot context, Map<String, Double> features) {
        if (context == null || context.getComorbidities() == null) {
            features.put("comorbid_diabetes", 0.0);
            features.put("comorbid_hypertension", 0.0);
            features.put("comorbid_ckd", 0.0);
            features.put("comorbid_heart_failure", 0.0);
            features.put("comorbid_copd", 0.0);
            features.put("comorbid_cancer", 0.0);
            features.put("comorbid_immunosuppressed", 0.0);
            features.put("comorbid_liver_disease", 0.0);
            features.put("comorbid_stroke_history", 0.0);
            features.put("comorbid_charlson_index", 0.0);
            return;
        }

        // Convert Map<String, Boolean> to List<String> (extract keys where value is true)
        Map<String, Boolean> comorbiditiesMap = context.getComorbidities();
        List<String> comorbidities = new ArrayList<>();
        for (Map.Entry<String, Boolean> entry : comorbiditiesMap.entrySet()) {
            if (Boolean.TRUE.equals(entry.getValue())) {
                comorbidities.add(entry.getKey());
            }
        }

        // Individual comorbidities (binary)
        features.put("comorbid_diabetes",
            comorbidities.contains("diabetes") ? 1.0 : 0.0);
        features.put("comorbid_hypertension",
            comorbidities.contains("hypertension") ? 1.0 : 0.0);
        features.put("comorbid_ckd",
            comorbidities.contains("chronic_kidney_disease") || comorbidities.contains("chronicKidneyDisease") ? 1.0 : 0.0);
        features.put("comorbid_heart_failure",
            comorbidities.contains("heart_failure") || comorbidities.contains("heartFailure") ? 1.0 : 0.0);
        features.put("comorbid_copd",
            comorbidities.contains("copd") || comorbidities.contains("COPD") ? 1.0 : 0.0);
        features.put("comorbid_cancer",
            comorbidities.contains("cancer") ? 1.0 : 0.0);
        features.put("comorbid_immunosuppressed",
            comorbidities.contains("immunosuppressed") ? 1.0 : 0.0);
        features.put("comorbid_liver_disease",
            comorbidities.contains("liver_disease") ? 1.0 : 0.0);
        features.put("comorbid_stroke_history",
            comorbidities.contains("stroke") ? 1.0 : 0.0);

        // Charlson Comorbidity Index (calculated)
        double charlsonIndex = context.getCharlsonIndex() != null ?
            context.getCharlsonIndex() : calculateCharlsonIndex(comorbidities);
        features.put("comorbid_charlson_index", charlsonIndex);
    }

    // ===== CATEGORY 8: CEP PATTERNS (5 features) =====

    private void extractCEPPatterns(PatternEvent patternEvent,
                                   SemanticEvent semanticEvent,
                                   Map<String, Double> features) {
        if (patternEvent == null && semanticEvent == null) {
            features.put("pattern_sepsis_detected", 0.0);
            features.put("pattern_deterioration_detected", 0.0);
            features.put("pattern_aki_detected", 0.0);
            features.put("pattern_confidence_score", 0.0);
            features.put("pattern_clinical_significance", 0.0);
            return;
        }

        // Pattern detection flags from Module 4
        if (patternEvent != null) {
            features.put("pattern_sepsis_detected",
                patternEvent.getPatternType() != null &&
                patternEvent.getPatternType().contains("sepsis") ? 1.0 : 0.0);

            features.put("pattern_deterioration_detected",
                patternEvent.isDeteriorationPattern() ? 1.0 : 0.0);

            features.put("pattern_aki_detected",
                patternEvent.getPatternType() != null &&
                patternEvent.getPatternType().contains("aki") ? 1.0 : 0.0);

            features.put("pattern_confidence_score", patternEvent.getConfidence());
        } else {
            features.put("pattern_sepsis_detected", 0.0);
            features.put("pattern_deterioration_detected", 0.0);
            features.put("pattern_aki_detected", 0.0);
            features.put("pattern_confidence_score", 0.0);
        }

        // Clinical significance from Module 3
        if (semanticEvent != null) {
            features.put("pattern_clinical_significance",
                semanticEvent.getClinicalSignificance());
        } else {
            features.put("pattern_clinical_significance", 0.0);
        }
    }

    // ===== HELPER METHODS =====

    private double getDoubleValue(Map<String, Object> map, String key, double defaultValue) {
        if (map == null || !map.containsKey(key)) {
            return defaultValue;
        }

        Object value = map.get(key);
        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return defaultValue;
        }
    }

    private double calculateCharlsonIndex(List<String> comorbidities) {
        double index = 0.0;

        if (comorbidities.contains("mi")) index += 1;
        if (comorbidities.contains("heart_failure")) index += 1;
        if (comorbidities.contains("pvd")) index += 1;
        if (comorbidities.contains("cerebrovascular")) index += 1;
        if (comorbidities.contains("dementia")) index += 1;
        if (comorbidities.contains("copd")) index += 1;
        if (comorbidities.contains("connective_tissue")) index += 1;
        if (comorbidities.contains("peptic_ulcer")) index += 1;
        if (comorbidities.contains("liver_disease_mild")) index += 1;
        if (comorbidities.contains("diabetes_uncomplicated")) index += 1;
        if (comorbidities.contains("diabetes_complicated")) index += 2;
        if (comorbidities.contains("hemiplegia")) index += 2;
        if (comorbidities.contains("ckd_moderate_severe")) index += 2;
        if (comorbidities.contains("cancer_localized")) index += 2;
        if (comorbidities.contains("leukemia")) index += 2;
        if (comorbidities.contains("lymphoma")) index += 2;
        if (comorbidities.contains("liver_disease_moderate_severe")) index += 3;
        if (comorbidities.contains("cancer_metastatic")) index += 6;
        if (comorbidities.contains("aids")) index += 6;

        return index;
    }

    private ClinicalFeatureVector createEmptyFeatureVector(PatientContextSnapshot context) {
        Map<String, Double> emptyFeatures = new LinkedHashMap<>();

        // Fill with zeros for all 70 features
        for (int i = 0; i < 70; i++) {
            emptyFeatures.put("feature_" + i, 0.0);
        }

        return ClinicalFeatureVector.builder()
            .patientId(context != null ? context.getPatientId() : "unknown")
            .features(emptyFeatures)
            .featureCount(70)
            .extractionTimestamp(System.currentTimeMillis())
            .extractionTimeMs(0.0)
            .build();
    }

    /**
     * Get feature names in extraction order
     */
    public List<String> getFeatureNames() {
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

    /**
     * Get feature count (always 70)
     */
    public int getFeatureCount() {
        return 70;
    }
}
