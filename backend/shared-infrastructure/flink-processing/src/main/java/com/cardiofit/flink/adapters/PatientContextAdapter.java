package com.cardiofit.flink.adapters;

import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.PatientDemographics;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.Medication;

import java.io.Serializable;
import java.time.Instant;
import java.util.Map;

/**
 * Adapter to convert EnrichedPatientContext (Module 2 output) to PatientContextSnapshot (ML model input)
 *
 * Purpose:
 * - Bridges data structure gap between Module 2 and MIMIC-IV ML models
 * - Maps Map-based storage (PatientContextState) to typed fields (PatientContextSnapshot)
 * - Handles missing data with safe defaults for ML inference
 *
 * Data Flow:
 * Module2_Enhanced → EnrichedPatientContext → [Adapter] → PatientContextSnapshot → MIMICMLInferenceOperator
 *
 * Key Mappings:
 * - Vitals: Map<String, Object> → typed Double fields
 * - Labs: Map<String, LabResult> → typed Double fields
 * - Medications: Map<String, Medication> → Boolean medication flags
 * - Clinical Scores: Direct passthrough (NEWS2, qSOFA, SOFA)
 *
 * Note: This adapter focuses on the 37 MIMIC-IV features. Additional PatientContextSnapshot
 * fields (trends, comorbidities) may be null if not available in EnrichedPatientContext.
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class PatientContextAdapter implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Convert EnrichedPatientContext to PatientContextSnapshot for ML inference
     *
     * @param enriched EnrichedPatientContext from Module 2 output
     * @return PatientContextSnapshot ready for MIMIC-IV models, or null if input is null
     */
    public PatientContextSnapshot adapt(EnrichedPatientContext enriched) {
        if (enriched == null) {
            return null;
        }

        PatientContextState state = enriched.getPatientState();
        if (state == null) {
            return null;
        }

        // Create snapshot with patient identifiers
        PatientContextSnapshot snapshot = new PatientContextSnapshot(
            enriched.getPatientId(),
            enriched.getEncounterId()
        );

        // Set timestamp
        snapshot.setTimestamp(Instant.ofEpochMilli(enriched.getEventTime()));

        // Map demographics (2 features for MIMIC-IV)
        mapDemographics(state, snapshot);

        // Map vital signs (16 features for MIMIC-IV: mean/min/max/std)
        mapVitalSigns(state, snapshot);

        // Map lab values (13 features for MIMIC-IV)
        mapLabValues(state, snapshot);

        // Map clinical scores (8 features for MIMIC-IV: SOFA total + 6 components + GCS)
        mapClinicalScores(state, snapshot);

        // Calculate derived metrics (MAP, BMI, shock index)
        snapshot.calculateDerivedMetrics();

        return snapshot;
    }

    /**
     * Map demographics from PatientContextState to PatientContextSnapshot
     * MIMIC-IV features: age, gender_male
     */
    private void mapDemographics(PatientContextState state, PatientContextSnapshot snapshot) {
        PatientDemographics demographics = state.getDemographics();
        if (demographics != null) {
            snapshot.setAge(demographics.getAge());
            snapshot.setGender(demographics.getGender());
            snapshot.setWeight(demographics.getWeight());
            // Note: ethnicity and height not available in PatientDemographics
            // These will be null in the snapshot, feature extractor will use defaults
        }
    }

    /**
     * Map vital signs from Map<String, Object> to typed Double fields
     * MIMIC-IV features: HR (mean/min/max/std), RR (mean/max), Temp (mean/max),
     *                    SBP (mean/min), DBP (mean), MAP (mean/min), SpO2 (mean/min)
     *
     * Note: Currently uses current values as approximations for mean/min/max.
     * Production enhancement would aggregate from time windows.
     */
    private void mapVitalSigns(PatientContextState state, PatientContextSnapshot snapshot) {
        Map<String, Object> vitals = state.getLatestVitals();
        if (vitals == null || vitals.isEmpty()) {
            return;
        }

        // Heart Rate (bpm)
        snapshot.setHeartRate(extractVital(vitals, "heartrate", "heart_rate", "hr"));

        // Blood Pressure (mmHg)
        snapshot.setSystolicBP(extractVital(vitals, "systolicbloodpressure", "systolic", "sbp"));
        snapshot.setDiastolicBP(extractVital(vitals, "diastolicbloodpressure", "diastolic", "dbp"));

        // Respiratory Rate (breaths/min)
        snapshot.setRespiratoryRate(extractVital(vitals, "respiratoryrate", "respiratory_rate", "rr"));

        // Temperature (celsius)
        snapshot.setTemperature(extractVital(vitals, "temperature", "temp"));

        // Oxygen Saturation (%)
        snapshot.setOxygenSaturation(extractVital(vitals, "oxygensaturation", "oxygen_saturation", "spo2"));
    }

    /**
     * Map lab values from Map<String, LabResult> to typed Double fields
     * MIMIC-IV features: WBC, Hemoglobin, Platelets, Creatinine (mean/max),
     *                    BUN, Glucose, Sodium, Potassium, Lactate (mean/max), Bilirubin
     */
    private void mapLabValues(PatientContextState state, PatientContextSnapshot snapshot) {
        Map<String, LabResult> labs = state.getRecentLabs();
        if (labs == null || labs.isEmpty()) {
            return;
        }

        // Complete Blood Count
        snapshot.setWhiteBloodCells(extractLabValue(labs, "WBC", "white_blood_cells"));
        snapshot.setHemoglobin(extractLabValue(labs, "hemoglobin", "hgb"));
        snapshot.setPlatelets(extractLabValue(labs, "platelets", "plt"));
        snapshot.setHematocrit(extractLabValue(labs, "hematocrit", "hct"));

        // Chemistry Panel
        snapshot.setSodium(extractLabValue(labs, "sodium", "na"));
        snapshot.setPotassium(extractLabValue(labs, "potassium", "k"));
        snapshot.setChloride(extractLabValue(labs, "chloride", "cl"));
        snapshot.setBicarbonate(extractLabValue(labs, "bicarbonate", "hco3"));
        snapshot.setBun(extractLabValue(labs, "bun", "blood_urea_nitrogen"));
        snapshot.setCreatinine(extractLabValue(labs, "creatinine", "cr"));
        snapshot.setGlucose(extractLabValue(labs, "glucose", "glu"));
        snapshot.setCalcium(extractLabValue(labs, "calcium", "ca"));

        // Liver Panel
        snapshot.setBilirubin(extractLabValue(labs, "bilirubin", "bili"));
        snapshot.setAst(extractLabValue(labs, "ast", "aspartate_aminotransferase"));
        snapshot.setAlt(extractLabValue(labs, "alt", "alanine_aminotransferase"));
        snapshot.setAlkalinePhosphatase(extractLabValue(labs, "alkaline_phosphatase", "alp"));

        // Blood Gases
        snapshot.setPh(extractLabValue(labs, "ph", "arterial_ph"));
        snapshot.setPao2(extractLabValue(labs, "pao2", "partial_pressure_oxygen"));
        snapshot.setPaco2(extractLabValue(labs, "paco2", "partial_pressure_co2"));

        // Cardiac Markers
        snapshot.setTroponin(extractLabValue(labs, "troponin", "troponin_i"));
        snapshot.setBnp(extractLabValue(labs, "bnp", "brain_natriuretic_peptide"));

        // Coagulation
        snapshot.setInr(extractLabValue(labs, "inr", "international_normalized_ratio"));
        snapshot.setPtt(extractLabValue(labs, "ptt", "partial_thromboplastin_time"));

        // Other Critical
        snapshot.setLactate(extractLabValue(labs, "lactate", "lac"));
        snapshot.setAlbumin(extractLabValue(labs, "albumin", "alb"));
    }

    /**
     * Map clinical scores from PatientContextState to PatientContextSnapshot
     * MIMIC-IV features: SOFA total + 6 components + GCS
     *
     * Note: SOFA components not currently available in PatientContextState.
     * Production enhancement would calculate components from vitals/labs.
     */
    private void mapClinicalScores(PatientContextState state, PatientContextSnapshot snapshot) {
        // Direct score mappings
        snapshot.setNews2Score(state.getNews2Score());
        snapshot.setQsofaScore(state.getQsofaScore());

        // SOFA Score (MIMIC-IV uses total SOFA)
        // Note: PatientContextState may have combinedAcuityScore which includes SOFA-like calculations
        // For now, we'll map any available score. Production would calculate proper SOFA.
        if (state.getCombinedAcuityScore() != null) {
            // Convert acuity score (0-100) to approximate SOFA (0-24)
            int approximateSOFA = (int) (state.getCombinedAcuityScore() * 0.24);
            snapshot.setSofaScore(approximateSOFA);
        }

        // GCS (Glasgow Coma Scale) - default to 15 (normal) if not available
        // This would be populated from neurological assessments in production
        // snapshot.setGcsScore(15); // Handled by MIMICFeatureExtractor with default
    }

    /**
     * Extract vital sign value from map with case-insensitive key matching
     *
     * @param vitals Map of vital signs
     * @param keys Possible key names (case-insensitive)
     * @return Double value or null if not found
     */
    private Double extractVital(Map<String, Object> vitals, String... keys) {
        for (String key : keys) {
            for (Map.Entry<String, Object> entry : vitals.entrySet()) {
                if (entry.getKey().toLowerCase().equals(key.toLowerCase())) {
                    Object value = entry.getValue();
                    if (value instanceof Number) {
                        return ((Number) value).doubleValue();
                    }
                    if (value instanceof String) {
                        try {
                            return Double.parseDouble((String) value);
                        } catch (NumberFormatException e) {
                            // Skip non-numeric strings
                        }
                    }
                }
            }
        }
        return null;
    }

    /**
     * Extract lab value from map with case-insensitive key matching
     *
     * @param labs Map of lab results
     * @param keys Possible key names (case-insensitive)
     * @return Double value or null if not found
     */
    private Double extractLabValue(Map<String, LabResult> labs, String... keys) {
        for (String key : keys) {
            for (Map.Entry<String, LabResult> entry : labs.entrySet()) {
                if (entry.getKey().toLowerCase().equals(key.toLowerCase())) {
                    LabResult result = entry.getValue();
                    if (result != null) {
                        return result.getValue();  // getValue() already returns Double
                    }
                }
            }
        }
        return null;
    }
}
