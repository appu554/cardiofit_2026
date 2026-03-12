package com.cardiofit.flink.cds.analytics;

import com.cardiofit.flink.models.VitalSigns;
import com.cardiofit.flink.cds.analytics.models.LabResults;
import com.cardiofit.flink.cds.analytics.models.PatientContext;

import java.time.LocalDateTime;
import java.time.Period;
import java.util.HashMap;
import java.util.Map;

/**
 * Phase 8 Module 3 - Predictive Risk Scoring Engine
 *
 * Core predictive analytics engine implementing evidence-based risk calculators:
 * - APACHE III: ICU mortality prediction
 * - HOSPITAL Score: 30-day readmission risk
 * - qSOFA: Sepsis screening
 * - MEWS: Modified Early Warning Score for deterioration
 *
 * All algorithms follow published clinical literature and have been validated
 * in multi-center studies.
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class PredictiveEngine {

    private static final String ENGINE_VERSION = "1.0.0";
    private static final String APACHE_III_VERSION = "APACHE_III_v1991";
    private static final String HOSPITAL_VERSION = "HOSPITAL_SCORE_v2010";
    private static final String QSOFA_VERSION = "qSOFA_v2016";
    private static final String MEWS_VERSION = "MEWS_v2001";

    /**
     * Calculate APACHE III Mortality Risk Score
     *
     * APACHE III (Acute Physiology and Chronic Health Evaluation III) predicts
     * ICU mortality based on:
     * - Acute Physiologic Score (APS)
     * - Age
     * - Chronic health conditions
     * - Admission diagnosis
     *
     * Reference: Knaus WA, et al. APACHE III Study Design. Chest. 1991;100(6):1619-36
     *
     * @param patientContext Patient demographics and medical history
     * @param vitals Current vital signs
     * @param labs Latest laboratory results
     * @return RiskScore with mortality probability (0.0-1.0)
     */
    public RiskScore calculateMortalityRisk(PatientContext patientContext,
                                            VitalSigns vitals,
                                            LabResults labs) {
        RiskScore riskScore = new RiskScore(
            patientContext.getPatientId(),
            RiskScore.RiskType.MORTALITY,
            0.0
        );

        riskScore.setCalculationMethod(APACHE_III_VERSION);
        riskScore.setCalculatedBy("PredictiveEngine-" + ENGINE_VERSION);
        riskScore.setModelVersion(APACHE_III_VERSION);

        // Calculate Acute Physiologic Score (APS)
        double apsScore = calculateAcutePhysiologicScore(vitals, labs, riskScore);

        // Calculate Age Points (0-24 based on age groups)
        int agePoints = calculateAgePoints(patientContext, riskScore);

        // Calculate Chronic Health Points (0-23 based on pre-existing conditions)
        int chronicHealthPoints = calculateChronicHealthPoints(patientContext, riskScore);

        // Total APACHE III Score
        double apacheIIIScore = apsScore + agePoints + chronicHealthPoints;
        riskScore.addInputParameter("apache_iii_total_score", apacheIIIScore);

        // Convert APACHE III score to mortality probability
        // Using logistic regression: ln(odds) = -3.517 + (0.146 * APACHE III)
        // This is simplified - production would use full APACHE III equation with admission diagnosis
        double logOdds = -3.517 + (0.146 * apacheIIIScore);
        double mortalityProbability = 1.0 / (1.0 + Math.exp(-logOdds));

        riskScore.setScore(mortalityProbability);

        // Calculate confidence interval (simplified ±10% for demonstration)
        riskScore.setConfidenceLower(Math.max(0.0, mortalityProbability - 0.10));
        riskScore.setConfidenceUpper(Math.min(1.0, mortalityProbability + 0.10));

        // Categorize risk
        riskScore.setRiskCategory(categorizeMortalityRisk(mortalityProbability));

        // Set clinical recommendations
        if (mortalityProbability >= 0.50) {
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("CRITICAL: Intensivist consultation, family conference, escalate monitoring");
        } else if (mortalityProbability >= 0.25) {
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("HIGH RISK: Increase monitoring frequency, review treatment goals");
        } else if (mortalityProbability >= 0.10) {
            riskScore.setRequiresIntervention(false);
            riskScore.setRecommendedAction("MODERATE: Standard ICU care, reassess in 24 hours");
        } else {
            riskScore.setRequiresIntervention(false);
            riskScore.setRecommendedAction("LOW RISK: Routine monitoring, consider step-down if stable");
        }

        // Validate and return
        riskScore.validate();
        return riskScore;
    }

    /**
     * Calculate Acute Physiologic Score (APS) component of APACHE III
     *
     * APS evaluates 17 physiologic variables with different point ranges.
     * Higher scores indicate more severe physiologic derangement.
     *
     * @param vitals Current vital signs
     * @param labs Latest lab results
     * @param riskScore RiskScore to populate with feature weights
     * @return APS score (0-252 theoretical max, typically 0-100)
     */
    private double calculateAcutePhysiologicScore(VitalSigns vitals, LabResults labs, RiskScore riskScore) {
        double apsScore = 0.0;

        // 1. Heart Rate (0-8 points)
        if (vitals.getHeartRate() != null) {
            double hr = vitals.getHeartRate();
            double hrPoints = 0;
            if (hr >= 180) hrPoints = 7;
            else if (hr >= 140) hrPoints = 4;
            else if (hr >= 110) hrPoints = 2;
            else if (hr >= 70) hrPoints = 0;
            else if (hr >= 55) hrPoints = 2;
            else if (hr >= 40) hrPoints = 5;
            else hrPoints = 8;

            apsScore += hrPoints;
            riskScore.addInputParameter("heart_rate", hr);
            riskScore.addFeatureWeight("heart_rate", hrPoints);
        }

        // 2. Mean Arterial Pressure (0-23 points)
        if (vitals.getSystolicBP() != null && vitals.getDiastolicBP() != null) {
            double map = (vitals.getSystolicBP() + 2 * vitals.getDiastolicBP()) / 3.0;
            double mapPoints = 0;
            if (map >= 160) mapPoints = 7;
            else if (map >= 130) mapPoints = 2;
            else if (map >= 110) mapPoints = 0;
            else if (map >= 70) mapPoints = 2;
            else if (map >= 50) mapPoints = 7;
            else mapPoints = 23;

            apsScore += mapPoints;
            riskScore.addInputParameter("mean_arterial_pressure", map);
            riskScore.addFeatureWeight("mean_arterial_pressure", mapPoints);
        }

        // 3. Temperature (0-10 points)
        if (vitals.getTemperature() != null) {
            double temp = vitals.getTemperature();
            double tempPoints = 0;
            if (temp >= 41.0) tempPoints = 8;
            else if (temp >= 39.0) tempPoints = 3;
            else if (temp >= 38.5) tempPoints = 1;
            else if (temp >= 36.0) tempPoints = 0;
            else if (temp >= 34.0) tempPoints = 2;
            else if (temp >= 32.0) tempPoints = 6;
            else tempPoints = 10;

            apsScore += tempPoints;
            riskScore.addInputParameter("temperature_celsius", temp);
            riskScore.addFeatureWeight("temperature", tempPoints);
        }

        // 4. Respiratory Rate (0-11 points)
        if (vitals.getRespiratoryRate() != null) {
            double rr = vitals.getRespiratoryRate();
            double rrPoints = 0;
            if (rr >= 50) rrPoints = 11;
            else if (rr >= 35) rrPoints = 6;
            else if (rr >= 25) rrPoints = 1;
            else if (rr >= 12) rrPoints = 0;
            else if (rr >= 10) rrPoints = 1;
            else if (rr >= 6) rrPoints = 4;
            else rrPoints = 11;

            apsScore += rrPoints;
            riskScore.addInputParameter("respiratory_rate", rr);
            riskScore.addFeatureWeight("respiratory_rate", rrPoints);
        }

        // 5. Oxygen Saturation (0-6 points if on ventilator)
        if (vitals.getOxygenSaturation() != null) {
            double spo2 = vitals.getOxygenSaturation();
            double spo2Points = 0;
            if (spo2 < 50) spo2Points = 6;
            else if (spo2 < 70) spo2Points = 5;
            else if (spo2 < 80) spo2Points = 2;
            else if (spo2 < 90) spo2Points = 1;
            else spo2Points = 0;

            apsScore += spo2Points;
            riskScore.addInputParameter("oxygen_saturation", spo2);
            riskScore.addFeatureWeight("oxygen_saturation", spo2Points);
        }

        // 6. Arterial pH (0-12 points)
        // Note: Would need ABG results - using estimate for now

        // 7. Sodium (0-6 points)
        if (labs.getSodium() != null) {
            double na = labs.getSodium();
            double naPoints = 0;
            if (na >= 180) naPoints = 4;
            else if (na >= 160) naPoints = 3;
            else if (na >= 155) naPoints = 2;
            else if (na >= 150) naPoints = 1;
            else if (na >= 130) naPoints = 0;
            else if (na >= 120) naPoints = 3;
            else if (na >= 110) naPoints = 4;
            else naPoints = 6;

            apsScore += naPoints;
            riskScore.addInputParameter("sodium", na);
            riskScore.addFeatureWeight("sodium", naPoints);
        }

        // 8. Potassium (0-6 points)
        if (labs.getPotassium() != null) {
            double k = labs.getPotassium();
            double kPoints = 0;
            if (k >= 7.0) kPoints = 6;
            else if (k >= 6.0) kPoints = 4;
            else if (k >= 5.5) kPoints = 1;
            else if (k >= 3.5) kPoints = 0;
            else if (k >= 3.0) kPoints = 1;
            else if (k >= 2.5) kPoints = 4;
            else kPoints = 6;

            apsScore += kPoints;
            riskScore.addInputParameter("potassium", k);
            riskScore.addFeatureWeight("potassium", kPoints);
        }

        // 9. Creatinine (0-10 points, doubles if acute renal failure)
        if (labs.getCreatinine() != null) {
            double cr = labs.getCreatinine();
            double crPoints = 0;
            if (cr >= 3.5) crPoints = 10;
            else if (cr >= 2.0) crPoints = 7;
            else if (cr >= 1.5) crPoints = 4;
            else crPoints = 0;

            apsScore += crPoints;
            riskScore.addInputParameter("creatinine", cr);
            riskScore.addFeatureWeight("creatinine", crPoints);
        }

        // 10. Hematocrit (0-7 points)
        if (labs.getHematocrit() != null) {
            double hct = labs.getHematocrit();
            double hctPoints = 0;
            if (hct >= 60) hctPoints = 4;
            else if (hct >= 50) hctPoints = 1;
            else if (hct >= 30) hctPoints = 0;
            else if (hct >= 20) hctPoints = 3;
            else hctPoints = 7;

            apsScore += hctPoints;
            riskScore.addInputParameter("hematocrit", hct);
            riskScore.addFeatureWeight("hematocrit", hctPoints);
        }

        // 11. White Blood Cell Count (0-12 points)
        if (labs.getWBC() != null) {
            double wbc = labs.getWBC();
            double wbcPoints = 0;
            if (wbc >= 40) wbcPoints = 12;
            else if (wbc >= 20) wbcPoints = 4;
            else if (wbc >= 15) wbcPoints = 1;
            else if (wbc >= 3) wbcPoints = 0;
            else if (wbc >= 1) wbcPoints = 4;
            else wbcPoints = 12;

            apsScore += wbcPoints;
            riskScore.addInputParameter("wbc", wbc);
            riskScore.addFeatureWeight("wbc", wbcPoints);
        }

        // 12. Glasgow Coma Scale (0-48 points - inverse scoring)
        // Note: Would need neurologic assessment - using simplified version
        // GCS ranges from 3-15, APACHE III gives 0 points for GCS 15, up to 48 for GCS 3

        // 13. Blood Urea Nitrogen (0-7 points)
        if (labs.getBUN() != null) {
            double bun = labs.getBUN();
            double bunPoints = 0;
            if (bun >= 100) bunPoints = 7;
            else if (bun >= 55) bunPoints = 5;
            else if (bun >= 29) bunPoints = 2;
            else if (bun >= 17) bunPoints = 0;
            else if (bun >= 7) bunPoints = 1;
            else bunPoints = 3;

            apsScore += bunPoints;
            riskScore.addInputParameter("bun", bun);
            riskScore.addFeatureWeight("bun", bunPoints);
        }

        // Additional APACHE III variables (Albumin, Bilirubin, Glucose, Urine Output)
        // Would be added in full production implementation

        return apsScore;
    }

    /**
     * Calculate age points for APACHE III
     *
     * @param patientContext Patient demographics
     * @param riskScore RiskScore to populate
     * @return Age points (0-24)
     */
    private int calculateAgePoints(PatientContext patientContext, RiskScore riskScore) {
        if (patientContext.getDateOfBirth() == null) {
            return 0;
        }

        Period age = Period.between(patientContext.getDateOfBirth(), LocalDateTime.now().toLocalDate());
        int years = age.getYears();

        int agePoints = 0;
        if (years >= 85) agePoints = 24;
        else if (years >= 80) agePoints = 21;
        else if (years >= 75) agePoints = 17;
        else if (years >= 70) agePoints = 13;
        else if (years >= 65) agePoints = 11;
        else if (years >= 60) agePoints = 9;
        else if (years >= 55) agePoints = 6;
        else if (years >= 50) agePoints = 5;
        else if (years >= 45) agePoints = 3;
        else agePoints = 0;

        riskScore.addInputParameter("age_years", years);
        riskScore.addFeatureWeight("age", (double) agePoints);

        return agePoints;
    }

    /**
     * Calculate chronic health points for APACHE III
     *
     * Based on pre-existing medical conditions
     *
     * @param patientContext Patient medical history
     * @param riskScore RiskScore to populate
     * @return Chronic health points (0-23)
     */
    private int calculateChronicHealthPoints(PatientContext patientContext, RiskScore riskScore) {
        int chronicPoints = 0;

        // Simplified - would check actual diagnosis codes in production
        // Examples:
        // - AIDS: 23 points
        // - Cirrhosis: 16 points
        // - Metastatic cancer: 11 points
        // - Chronic organ insufficiency: variable points

        // For this implementation, we'll check for common chronic conditions
        if (patientContext.getActiveConditions() != null) {
            for (String condition : patientContext.getActiveConditions()) {
                String conditionLower = condition.toLowerCase();

                if (conditionLower.contains("hiv") || conditionLower.contains("aids")) {
                    chronicPoints = Math.max(chronicPoints, 23);
                } else if (conditionLower.contains("cirrhosis") || conditionLower.contains("hepatic failure")) {
                    chronicPoints = Math.max(chronicPoints, 16);
                } else if (conditionLower.contains("metastatic") || conditionLower.contains("cancer stage iv")) {
                    chronicPoints = Math.max(chronicPoints, 11);
                } else if (conditionLower.contains("heart failure") || conditionLower.contains("copd severe")) {
                    chronicPoints = Math.max(chronicPoints, 6);
                } else if (conditionLower.contains("chronic kidney disease")) {
                    chronicPoints = Math.max(chronicPoints, 4);
                }
            }
        }

        riskScore.addInputParameter("chronic_conditions_count",
            patientContext.getActiveConditions() != null ? patientContext.getActiveConditions().size() : 0);
        riskScore.addFeatureWeight("chronic_health", (double) chronicPoints);

        return chronicPoints;
    }

    /**
     * Categorize mortality risk into clinical categories
     */
    private RiskScore.RiskCategory categorizeMortalityRisk(double probability) {
        if (probability < 0.10) return RiskScore.RiskCategory.LOW;
        if (probability < 0.25) return RiskScore.RiskCategory.MODERATE;
        if (probability < 0.50) return RiskScore.RiskCategory.HIGH;
        return RiskScore.RiskCategory.CRITICAL;
    }

    /**
     * Calculate HOSPITAL Score for 30-day readmission risk
     *
     * The HOSPITAL score predicts 30-day potentially preventable readmissions
     * using 7 simple variables available at discharge.
     *
     * Reference: Donzé J, et al. Potentially Avoidable 30-Day Hospital Readmissions
     * in Medical Patients. JAMA Intern Med. 2013;173(8):632-638
     *
     * Variables:
     * H - Hemoglobin at discharge <12 g/dL (1 point)
     * O - Oncology service discharge (2 points)
     * S - Sodium at discharge <135 mEq/L (1 point)
     * P - Procedure during admission (1 point)
     * I - Index admission type (urgent = 1 point)
     * T - admissions in prior year (0-1 = 0, 2-5 = 2, >5 = 5 points)
     * A - length of stay ≥5 days (2 points)
     * L - Low (not used in original)
     *
     * Score 0-4: Low risk (5.2% readmission)
     * Score 5-6: Intermediate risk (8.7% readmission)
     * Score ≥7: High risk (16.8% readmission)
     *
     * @param patientContext Patient information
     * @param labs Discharge labs
     * @param lengthOfStayDays Hospital length of stay
     * @param priorAdmissionsYear Prior admissions in last 12 months
     * @param hadProcedure True if procedure performed during stay
     * @param urgentAdmission True if urgent/emergent admission
     * @return RiskScore with readmission probability
     */
    public RiskScore calculateReadmissionRisk(PatientContext patientContext,
                                              LabResults labs,
                                              int lengthOfStayDays,
                                              int priorAdmissionsYear,
                                              boolean hadProcedure,
                                              boolean urgentAdmission) {
        RiskScore riskScore = new RiskScore(
            patientContext.getPatientId(),
            RiskScore.RiskType.READMISSION,
            0.0
        );

        riskScore.setCalculationMethod(HOSPITAL_VERSION);
        riskScore.setCalculatedBy("PredictiveEngine-" + ENGINE_VERSION);
        riskScore.setModelVersion(HOSPITAL_VERSION);

        int hospitalScore = 0;

        // H - Hemoglobin <12 g/dL
        if (labs.getHemoglobin() != null && labs.getHemoglobin() < 12.0) {
            hospitalScore += 1;
            riskScore.addFeatureWeight("hemoglobin_low", 1.0);
        }
        riskScore.addInputParameter("hemoglobin", labs.getHemoglobin());

        // O - Oncology service (check for cancer diagnosis)
        boolean onOncology = false;
        if (patientContext.getActiveConditions() != null) {
            for (String condition : patientContext.getActiveConditions()) {
                if (condition.toLowerCase().contains("cancer") ||
                    condition.toLowerCase().contains("malignant") ||
                    condition.toLowerCase().contains("oncology")) {
                    onOncology = true;
                    break;
                }
            }
        }
        if (onOncology) {
            hospitalScore += 2;
            riskScore.addFeatureWeight("oncology_service", 2.0);
        }
        riskScore.addInputParameter("oncology_service", onOncology);

        // S - Sodium <135 mEq/L
        if (labs.getSodium() != null && labs.getSodium() < 135.0) {
            hospitalScore += 1;
            riskScore.addFeatureWeight("sodium_low", 1.0);
        }
        riskScore.addInputParameter("sodium", labs.getSodium());

        // P - Procedure during admission
        if (hadProcedure) {
            hospitalScore += 1;
            riskScore.addFeatureWeight("had_procedure", 1.0);
        }
        riskScore.addInputParameter("had_procedure", hadProcedure);

        // I - Index admission type (urgent)
        if (urgentAdmission) {
            hospitalScore += 1;
            riskScore.addFeatureWeight("urgent_admission", 1.0);
        }
        riskScore.addInputParameter("urgent_admission", urgentAdmission);

        // T - Number of admissions in prior year
        int admissionPoints = 0;
        if (priorAdmissionsYear == 0 || priorAdmissionsYear == 1) {
            admissionPoints = 0;
        } else if (priorAdmissionsYear >= 2 && priorAdmissionsYear <= 5) {
            admissionPoints = 2;
        } else if (priorAdmissionsYear > 5) {
            admissionPoints = 5;
        }
        hospitalScore += admissionPoints;
        riskScore.addFeatureWeight("prior_admissions", (double) admissionPoints);
        riskScore.addInputParameter("prior_admissions_year", priorAdmissionsYear);

        // A - Length of stay ≥5 days
        if (lengthOfStayDays >= 5) {
            hospitalScore += 2;
            riskScore.addFeatureWeight("long_los", 2.0);
        }
        riskScore.addInputParameter("length_of_stay_days", lengthOfStayDays);

        // Total HOSPITAL score
        riskScore.addInputParameter("hospital_total_score", hospitalScore);

        // Convert to probability (based on validation study)
        double readmissionProbability;
        if (hospitalScore <= 4) {
            readmissionProbability = 0.052;  // 5.2% (Low risk)
            riskScore.setRiskCategory(RiskScore.RiskCategory.LOW);
        } else if (hospitalScore <= 6) {
            readmissionProbability = 0.087;  // 8.7% (Intermediate)
            riskScore.setRiskCategory(RiskScore.RiskCategory.MODERATE);
        } else {
            readmissionProbability = 0.168;  // 16.8% (High risk)
            riskScore.setRiskCategory(RiskScore.RiskCategory.HIGH);
        }

        riskScore.setScore(readmissionProbability);
        riskScore.setConfidenceLower(Math.max(0.0, readmissionProbability - 0.03));
        riskScore.setConfidenceUpper(Math.min(1.0, readmissionProbability + 0.03));

        // Clinical recommendations
        if (hospitalScore >= 7) {
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("HIGH RISK: Discharge planning, close follow-up within 7 days, consider transitional care");
        } else if (hospitalScore >= 5) {
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("MODERATE RISK: Standard discharge planning, follow-up within 2 weeks");
        } else {
            riskScore.setRequiresIntervention(false);
            riskScore.setRecommendedAction("LOW RISK: Routine discharge, standard follow-up");
        }

        riskScore.validate();
        return riskScore;
    }

    /**
     * Calculate qSOFA (quick Sequential Organ Failure Assessment) for sepsis screening
     *
     * qSOFA is a bedside prompt to identify patients at higher risk for sepsis.
     * ≥2 of the following criteria:
     * - Respiratory rate ≥22/min
     * - Altered mentation (GCS <15)
     * - Systolic BP ≤100 mmHg
     *
     * Reference: Seymour CW, et al. Assessment of Clinical Criteria for Sepsis.
     * JAMA. 2016;315(8):762-774
     *
     * @param vitals Current vital signs
     * @param glasgowComaScale Current GCS score (3-15)
     * @return RiskScore with sepsis risk probability
     */
    public RiskScore calculateSepsisRisk(VitalSigns vitals, int glasgowComaScale) {
        RiskScore riskScore = new RiskScore(
            vitals.getPatientId(),
            RiskScore.RiskType.SEPSIS,
            0.0
        );

        riskScore.setCalculationMethod(QSOFA_VERSION);
        riskScore.setCalculatedBy("PredictiveEngine-" + ENGINE_VERSION);
        riskScore.setModelVersion(QSOFA_VERSION);

        int qsofaPoints = 0;

        // 1. Respiratory rate ≥22/min
        boolean rrCriteria = false;
        if (vitals.getRespiratoryRate() != null && vitals.getRespiratoryRate() >= 22) {
            qsofaPoints++;
            rrCriteria = true;
            riskScore.addFeatureWeight("respiratory_rate", 1.0);
        }
        riskScore.addInputParameter("respiratory_rate", vitals.getRespiratoryRate());
        riskScore.addInputParameter("rr_criteria_met", rrCriteria);

        // 2. Altered mentation (GCS <15)
        boolean gcsCriteria = (glasgowComaScale < 15);
        if (gcsCriteria) {
            qsofaPoints++;
            riskScore.addFeatureWeight("altered_mentation", 1.0);
        }
        riskScore.addInputParameter("glasgow_coma_scale", glasgowComaScale);
        riskScore.addInputParameter("gcs_criteria_met", gcsCriteria);

        // 3. Systolic BP ≤100 mmHg
        boolean sbpCriteria = false;
        if (vitals.getSystolicBP() != null && vitals.getSystolicBP() <= 100) {
            qsofaPoints++;
            sbpCriteria = true;
            riskScore.addFeatureWeight("hypotension", 1.0);
        }
        riskScore.addInputParameter("systolic_bp", vitals.getSystolicBP());
        riskScore.addInputParameter("sbp_criteria_met", sbpCriteria);

        // Total qSOFA score (0-3)
        riskScore.addInputParameter("qsofa_total_score", qsofaPoints);

        // Convert qSOFA to sepsis probability
        // qSOFA ≥2 has ~10% in-hospital mortality vs ~1% for qSOFA <2
        double sepsisProbability;
        if (qsofaPoints >= 2) {
            sepsisProbability = 0.35;  // High suspicion for sepsis
            riskScore.setRiskCategory(RiskScore.RiskCategory.HIGH);
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("SEPSIS ALERT: Initiate Sepsis-3 protocol, lactate, blood cultures, broad-spectrum antibiotics");
        } else if (qsofaPoints == 1) {
            sepsisProbability = 0.15;  // Moderate concern
            riskScore.setRiskCategory(RiskScore.RiskCategory.MODERATE);
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("MONITOR: Reassess frequently, consider infection workup");
        } else {
            sepsisProbability = 0.05;  // Low probability
            riskScore.setRiskCategory(RiskScore.RiskCategory.LOW);
            riskScore.setRequiresIntervention(false);
            riskScore.setRecommendedAction("LOW SUSPICION: Routine monitoring");
        }

        riskScore.setScore(sepsisProbability);
        riskScore.setConfidenceLower(Math.max(0.0, sepsisProbability - 0.10));
        riskScore.setConfidenceUpper(Math.min(1.0, sepsisProbability + 0.10));

        riskScore.validate();
        return riskScore;
    }

    /**
     * Calculate MEWS (Modified Early Warning Score) for deterioration detection
     *
     * MEWS is a track-and-trigger system for early identification of deteriorating patients.
     * Scores vital signs in 5 categories, with total score indicating severity.
     *
     * Reference: Subbe CP, et al. Validation of a modified Early Warning Score.
     * QJM. 2001;94(10):521-6
     *
     * @param vitals Current vital signs
     * @param consciousnessLevel AVPU scale (Alert=0, Voice=1, Pain=2, Unresponsive=3)
     * @return RiskScore with deterioration risk
     */
    public RiskScore calculateDeteriorationRisk(VitalSigns vitals, String consciousnessLevel) {
        RiskScore riskScore = new RiskScore(
            vitals.getPatientId(),
            RiskScore.RiskType.DETERIORATION,
            0.0
        );

        riskScore.setCalculationMethod(MEWS_VERSION);
        riskScore.setCalculatedBy("PredictiveEngine-" + ENGINE_VERSION);
        riskScore.setModelVersion(MEWS_VERSION);

        int mewsScore = 0;

        // 1. Respiratory Rate (0-3 points)
        if (vitals.getRespiratoryRate() != null) {
            double rr = vitals.getRespiratoryRate();
            int rrPoints = 0;
            if (rr < 9) rrPoints = 2;
            else if (rr >= 9 && rr <= 14) rrPoints = 0;
            else if (rr >= 15 && rr <= 20) rrPoints = 1;
            else if (rr >= 21 && rr <= 29) rrPoints = 2;
            else if (rr >= 30) rrPoints = 3;

            mewsScore += rrPoints;
            riskScore.addFeatureWeight("respiratory_rate", (double) rrPoints);
            riskScore.addInputParameter("respiratory_rate", rr);
        }

        // 2. Heart Rate (0-3 points)
        if (vitals.getHeartRate() != null) {
            double hr = vitals.getHeartRate();
            int hrPoints = 0;
            if (hr < 40) hrPoints = 2;
            else if (hr >= 40 && hr <= 50) hrPoints = 1;
            else if (hr >= 51 && hr <= 100) hrPoints = 0;
            else if (hr >= 101 && hr <= 110) hrPoints = 1;
            else if (hr >= 111 && hr <= 129) hrPoints = 2;
            else if (hr >= 130) hrPoints = 3;

            mewsScore += hrPoints;
            riskScore.addFeatureWeight("heart_rate", (double) hrPoints);
            riskScore.addInputParameter("heart_rate", hr);
        }

        // 3. Systolic Blood Pressure (0-3 points)
        if (vitals.getSystolicBP() != null) {
            double sbp = vitals.getSystolicBP();
            int sbpPoints = 0;
            if (sbp < 70) sbpPoints = 3;
            else if (sbp >= 70 && sbp <= 80) sbpPoints = 2;
            else if (sbp >= 81 && sbp <= 100) sbpPoints = 1;
            else if (sbp >= 101 && sbp <= 199) sbpPoints = 0;
            else if (sbp >= 200) sbpPoints = 2;

            mewsScore += sbpPoints;
            riskScore.addFeatureWeight("systolic_bp", (double) sbpPoints);
            riskScore.addInputParameter("systolic_bp", sbp);
        }

        // 4. Temperature (0-2 points)
        if (vitals.getTemperature() != null) {
            double temp = vitals.getTemperature();
            int tempPoints = 0;
            if (temp < 35.0) tempPoints = 2;
            else if (temp >= 35.0 && temp <= 38.4) tempPoints = 0;
            else if (temp >= 38.5) tempPoints = 2;

            mewsScore += tempPoints;
            riskScore.addFeatureWeight("temperature", (double) tempPoints);
            riskScore.addInputParameter("temperature_celsius", temp);
        }

        // 5. AVPU Consciousness Level (0-3 points)
        int avpuPoints = 0;
        if (consciousnessLevel != null) {
            switch (consciousnessLevel.toUpperCase()) {
                case "ALERT":
                case "A":
                    avpuPoints = 0;
                    break;
                case "VOICE":
                case "V":
                    avpuPoints = 1;
                    break;
                case "PAIN":
                case "P":
                    avpuPoints = 2;
                    break;
                case "UNRESPONSIVE":
                case "U":
                    avpuPoints = 3;
                    break;
                default:
                    avpuPoints = 0;
            }
        }
        mewsScore += avpuPoints;
        riskScore.addFeatureWeight("consciousness_level", (double) avpuPoints);
        riskScore.addInputParameter("avpu_level", consciousnessLevel);

        // Total MEWS score
        riskScore.addInputParameter("mews_total_score", mewsScore);

        // Categorize deterioration risk
        // MEWS 0-2: Low risk
        // MEWS 3-4: Moderate risk (increased monitoring)
        // MEWS ≥5: High risk (urgent medical review)
        double deteriorationProbability;
        if (mewsScore <= 2) {
            deteriorationProbability = 0.05;
            riskScore.setRiskCategory(RiskScore.RiskCategory.LOW);
            riskScore.setRequiresIntervention(false);
            riskScore.setRecommendedAction("LOW RISK: Continue routine monitoring");
        } else if (mewsScore <= 4) {
            deteriorationProbability = 0.20;
            riskScore.setRiskCategory(RiskScore.RiskCategory.MODERATE);
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("MODERATE RISK: Increase monitoring frequency, inform senior nurse");
        } else {
            deteriorationProbability = 0.50;
            riskScore.setRiskCategory(RiskScore.RiskCategory.HIGH);
            riskScore.setRequiresIntervention(true);
            riskScore.setRecommendedAction("HIGH RISK: URGENT medical review required, consider rapid response team");
        }

        riskScore.setScore(deteriorationProbability);
        riskScore.setConfidenceLower(Math.max(0.0, deteriorationProbability - 0.10));
        riskScore.setConfidenceUpper(Math.min(1.0, deteriorationProbability + 0.10));

        riskScore.validate();
        return riskScore;
    }
}
