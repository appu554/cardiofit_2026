package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;
import com.cardiofit.flink.models.LabValues;
import com.cardiofit.flink.models.Condition;
import com.cardiofit.flink.models.VitalSign;
import com.cardiofit.flink.models.Medication;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Clinical Score Calculator for Module 2 Advanced Enrichment.
 *
 * Implements evidence-based clinical scoring systems:
 * - NEWS2 (National Early Warning Score 2)
 * - Framingham Risk Score
 * - Metabolic Syndrome Risk Assessment
 * - CHADS-VASc Score for AFib patients
 * - qSOFA for sepsis screening
 */
public class ClinicalScoreCalculator implements Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ClinicalScoreCalculator.class);

    /**
     * Container for multi-dimensional acuity scores.
     */
    public static class AcuityScores implements Serializable {
        private int news2Score;
        private String news2Interpretation;
        private double metabolicAcuityScore;
        private double combinedAcuityScore;
        private String acuityLevel;
        private int qsofaScore;

        // Getters and setters
        public int getNews2Score() { return news2Score; }
        public void setNews2Score(int news2Score) { this.news2Score = news2Score; }

        public String getNews2Interpretation() { return news2Interpretation; }
        public void setNews2Interpretation(String news2Interpretation) {
            this.news2Interpretation = news2Interpretation;
        }

        public double getMetabolicAcuityScore() { return metabolicAcuityScore; }
        public void setMetabolicAcuityScore(double metabolicAcuityScore) {
            this.metabolicAcuityScore = metabolicAcuityScore;
        }

        public double getCombinedAcuityScore() { return combinedAcuityScore; }
        public void setCombinedAcuityScore(double combinedAcuityScore) {
            this.combinedAcuityScore = combinedAcuityScore;
        }

        public String getAcuityLevel() { return acuityLevel; }
        public void setAcuityLevel(String acuityLevel) { this.acuityLevel = acuityLevel; }

        public int getQsofaScore() { return qsofaScore; }
        public void setQsofaScore(int qsofaScore) { this.qsofaScore = qsofaScore; }
    }

    /**
     * Calculate all acuity scores for a patient.
     */
    public AcuityScores calculateAcuityScores(PatientSnapshot snapshot, Map<String, Object> payload) {
        AcuityScores scores = new AcuityScores();

        // Calculate NEWS2 score
        int news2 = calculateNEWS2(payload);
        scores.setNews2Score(news2);
        scores.setNews2Interpretation(interpretNEWS2(news2));

        // Calculate metabolic acuity
        double metabolicScore = calculateMetabolicAcuity(snapshot);
        scores.setMetabolicAcuityScore(Math.round(metabolicScore * 10) / 10.0);

        // Calculate combined score with weighting
        // NEWS2 gets 70% weight, metabolic gets 30% weight
        double combined = (0.7 * news2) + (0.3 * metabolicScore);
        scores.setCombinedAcuityScore(Math.round(combined * 10) / 10.0);

        // Set overall acuity level based on combined score
        if (combined >= 7) {
            scores.setAcuityLevel("CRITICAL");
        } else if (combined >= 5) {
            scores.setAcuityLevel("HIGH");
        } else if (combined >= 3) {
            scores.setAcuityLevel("MEDIUM");
        } else {
            scores.setAcuityLevel("LOW");
        }

        LOG.info("Calculated acuity scores for patient {}: NEWS2={}, Metabolic={}, Combined={}, Level={}",
            snapshot.getPatientId(), news2, metabolicScore, combined, scores.getAcuityLevel());

        return scores;
    }

    /**
     * Calculate NEWS2 (National Early Warning Score 2).
     * Based on Royal College of Physicians guidelines.
     */
    private int calculateNEWS2(Map<String, Object> payload) {
        int score = 0;

        // Heart rate scoring
        Integer hr = extractInteger(payload, "heart_rate");
        if (hr != null) {
            if (hr <= 40 || hr >= 131) {
                score += 3;
            } else if (hr <= 50) {
                score += 1;
            } else if (hr >= 111 && hr <= 130) {
                score += 2;
            } else if (hr >= 91 && hr <= 110) {
                score += 1;
            }
            // 51-90 = 0 points
        }

        // Blood pressure scoring (systolic)
        String bp = (String) payload.get("blood_pressure");
        if (bp != null) {
            int systolic = parseSystolic(bp);
            if (systolic > 0) {
                if (systolic <= 90 || systolic >= 220) {
                    score += 3;
                } else if (systolic <= 100) {
                    score += 2;
                } else if (systolic <= 110) {
                    score += 1;
                }
                // 111-219 = 0 points
            }
        }

        // Respiratory rate scoring
        Integer rr = extractInteger(payload, "respiratory_rate");
        if (rr != null) {
            if (rr <= 8 || rr >= 25) {
                score += 3;
            } else if (rr >= 21 && rr <= 24) {
                score += 2;
            } else if (rr >= 9 && rr <= 11) {
                score += 1;
            }
            // 12-20 = 0 points
        }

        // Temperature scoring
        Double temp = extractDouble(payload, "temperature");
        if (temp != null) {
            if (temp <= 35.0) {
                score += 3;
            } else if (temp >= 39.1) {
                score += 2;
            } else if (temp <= 36.0 || (temp >= 38.1 && temp <= 39.0)) {
                score += 1;
            }
            // 36.1-38.0 = 0 points
        }

        // SpO2 scoring (Scale 1 - on air, Scale 2 - on oxygen)
        Integer spo2 = extractInteger(payload, "oxygen_saturation");
        Boolean onOxygen = (Boolean) payload.get("on_oxygen");
        if (spo2 != null) {
            if (onOxygen != null && onOxygen) {
                // Scale 2 (on oxygen)
                if (spo2 <= 92 || spo2 >= 97) {
                    score += 3;
                } else if (spo2 == 93 || spo2 == 94) {
                    score += 2;
                } else if (spo2 == 95 || spo2 == 96) {
                    score += 1;
                }
            } else {
                // Scale 1 (on air)
                if (spo2 <= 91) {
                    score += 3;
                } else if (spo2 == 92 || spo2 == 93) {
                    score += 2;
                } else if (spo2 == 94 || spo2 == 95) {
                    score += 1;
                }
                // >=96 = 0 points
            }
        }

        // Consciousness level (AVPU scale)
        String consciousness = (String) payload.get("consciousness");
        if (consciousness != null && !consciousness.equalsIgnoreCase("alert")) {
            score += 3; // Any deviation from Alert = 3 points
        }

        return score;
    }

    /**
     * Interpret NEWS2 score into clinical categories.
     */
    private String interpretNEWS2(int score) {
        if (score == 0) {
            return "LOW_BASELINE";
        } else if (score >= 1 && score <= 4) {
            return "LOW";
        } else if (score == 5 || score == 6) {
            return "MEDIUM";
        } else if (score >= 7) {
            return "HIGH";
        }
        return "UNKNOWN";
    }

    /**
     * Calculate metabolic acuity score based on chronic conditions and risk factors.
     */
    private double calculateMetabolicAcuity(PatientSnapshot snapshot) {
        double score = 0;

        // Check for metabolic conditions (each adds to score)
        List<String> conditions = conditionsToStrings(snapshot.getActiveConditions());
        if (conditions != null && !conditions.isEmpty()) {
            if (hasCondition(conditions, "Diabetes") || hasCondition(conditions, "Prediabetes")) {
                score += 1.0;
            }
            if (hasCondition(conditions, "Hypertension") || hasCondition(conditions, "Hypertensive disorder")) {
                score += 1.0;
            }
            if (hasCondition(conditions, "Chronic kidney disease") || hasCondition(conditions, "CKD")) {
                score += 1.5;
            }
            if (hasCondition(conditions, "Heart failure") || hasCondition(conditions, "CHF")) {
                score += 2.0;
            }
            if (hasCondition(conditions, "Obesity")) {
                score += 0.5;
            }
        }

        // Check metabolic labs if available
        if (snapshot.getLabHistory() != null) {
            LabValues labs = snapshot.getLabHistory().getLatestAsLabValues();
            if (labs != null) {
                // Elevated creatinine indicates kidney dysfunction
                if (labs.getCreatinine() != null && labs.getCreatinine() > 1.5) {
                    score += 1.0;
                }
            }

            // Check HbA1c from lab history
            com.cardiofit.flink.models.LabResult hba1c = snapshot.getLabHistory().getLatestByType("HbA1c");
            if (hba1c != null && hba1c.getValue() != null && hba1c.getValue() >= 6.5) {
                score += 0.5;
            }

            // Check triglycerides from lab history
            com.cardiofit.flink.models.LabResult triglycerides = snapshot.getLabHistory().getLatestByType("Triglycerides");
            if (triglycerides != null && triglycerides.getValue() != null && triglycerides.getValue() > 150) {
                score += 0.5;
            }
        }

        // Risk cohort membership adds to score
        if (snapshot.getRiskCohorts() != null) {
            for (String cohort : snapshot.getRiskCohorts()) {
                if (cohort.toLowerCase().contains("metabolic")) {
                    score += 0.5;
                    break;
                }
            }
        }

        // Cap the score at 5.0
        return Math.min(score, 5.0);
    }

    /**
     * Calculate all clinical scores for comprehensive assessment.
     */
    public Map<String, Object> calculateAllClinicalScores(PatientSnapshot snapshot) {
        Map<String, Object> scores = new HashMap<>();

        // Calculate Framingham Risk Score if data available
        if (hasRequiredDataForFramingham(snapshot)) {
            double framingham = calculateFraminghamRisk(snapshot);
            scores.put("framingham_risk_10yr", framingham);
            scores.put("framingham_interpretation", interpretFramingham(framingham));
        }

        // Calculate Metabolic Syndrome Risk
        double metabolicRisk = calculateMetabolicSyndromeRisk(snapshot);
        scores.put("metabolic_syndrome_risk", metabolicRisk);
        scores.put("metabolic_interpretation",
            metabolicRisk > 0.8 ? "HIGH" : metabolicRisk > 0.5 ? "MODERATE" : "LOW");

        // Calculate CHADS-VASc for AFib patients
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Atrial fibrillation") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "AFib")) {
            int chadsVasc = calculateCHADSVASc(snapshot);
            scores.put("chads_vasc_score", chadsVasc);
            scores.put("stroke_risk_annual", getStrokeRiskFromCHADSVASc(chadsVasc));
        }

        // Calculate qSOFA if relevant vitals available
        if (hasRelevantVitalsForSepsis(snapshot)) {
            int qsofa = calculateqSOFA(snapshot);
            scores.put("qsofa_score", qsofa);
            scores.put("sepsis_risk", qsofa >= 2 ? "HIGH" : "LOW");
        }

        return scores;
    }

    /**
     * Calculate Framingham Risk Score for 10-year cardiovascular disease risk.
     * Simplified version for demonstration.
     */
    private double calculateFraminghamRisk(PatientSnapshot snapshot) {
        double points = 0;

        // Age scoring (simplified - actual has different tables for men/women)
        int age = snapshot.getAge() != null ? snapshot.getAge() : 40;
        boolean male = "male".equalsIgnoreCase(snapshot.getGender());

        if (male) {
            if (age >= 70) points += 13;
            else if (age >= 60) points += 10;
            else if (age >= 50) points += 6;
            else if (age >= 45) points += 3;
        } else {
            if (age >= 70) points += 16;
            else if (age >= 60) points += 12;
            else if (age >= 50) points += 8;
            else if (age >= 45) points += 4;
        }

        // Total cholesterol (using default if not available)
        double totalChol = 200; // default
        if (snapshot.getLabHistory() != null) {
            com.cardiofit.flink.models.LabResult cholLab = snapshot.getLabHistory().getLatestByType("Total Cholesterol");
            if (cholLab != null && cholLab.getValue() != null) {
                totalChol = cholLab.getValue();
            }
        }

        if (totalChol >= 280) points += 3;
        else if (totalChol >= 240) points += 2;
        else if (totalChol >= 200) points += 1;

        // HDL cholesterol
        double hdl = 50; // default
        if (snapshot.getLabHistory() != null) {
            com.cardiofit.flink.models.LabResult hdlLab = snapshot.getLabHistory().getLatestByType("HDL");
            if (hdlLab != null && hdlLab.getValue() != null) {
                hdl = hdlLab.getValue();
            }
        }

        if (hdl < 40) points += 2;
        else if (hdl < 50) points += 1;
        else if (hdl >= 60) points -= 1;

        // Systolic blood pressure (from current vitals if available)
        int sbp = 120; // default
        if (snapshot.getVitalsHistory() != null && snapshot.getVitalsHistory().getLatest() != null) {
            VitalSign latestVitals = snapshot.getVitalsHistory().getLatest();
            if (latestVitals.getBloodPressureSystolic() != null) {
                sbp = latestVitals.getBloodPressureSystolic().intValue();
            }
        }

        boolean onBPMeds = isOnMedication(snapshot, "antihypertensive") ||
                           isOnMedication(snapshot, "Telmisartan");

        if (onBPMeds) {
            if (sbp >= 160) points += 3;
            else if (sbp >= 140) points += 2;
            else if (sbp >= 130) points += 1;
        } else {
            if (sbp >= 160) points += 2;
            else if (sbp >= 140) points += 1;
        }

        // Smoking status (not available in our data, so skip)
        // Diabetes
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Diabetes")) {
            points += 2;
        }

        // Convert points to 10-year risk percentage
        double risk;
        if (points <= 0) risk = 0.01;
        else if (points <= 4) risk = 0.01;
        else if (points <= 8) risk = 0.03;
        else if (points <= 12) risk = 0.08;
        else if (points <= 16) risk = 0.16;
        else if (points <= 20) risk = 0.25;
        else risk = 0.30;

        return risk;
    }

    /**
     * Calculate Metabolic Syndrome Risk based on 5 components.
     */
    private double calculateMetabolicSyndromeRisk(PatientSnapshot snapshot) {
        int components = 0;
        int totalComponents = 5;

        // Component 1: Central obesity (using BMI as proxy since waist circumference not available)
        // Note: In real implementation, would need waist circumference
        Double bmi = getBMI(snapshot);
        if (bmi != null && bmi >= 30) {
            components++;
        }

        // Component 2: Elevated blood pressure
        boolean elevatedBP = false;
        if (snapshot.getVitalsHistory() != null && snapshot.getVitalsHistory().getLatest() != null) {
            VitalSign latestVitals = snapshot.getVitalsHistory().getLatest();
            if (latestVitals.getBloodPressureSystolic() != null && latestVitals.getBloodPressureDiastolic() != null) {
                int systolic = latestVitals.getBloodPressureSystolic().intValue();
                int diastolic = latestVitals.getBloodPressureDiastolic().intValue();
                if (systolic >= 130 || diastolic >= 85) {
                    elevatedBP = true;
                }
            }
        }
        if (elevatedBP || isOnMedication(snapshot, "antihypertensive") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Hypertension")) {
            components++;
        }

        // Component 3: Elevated fasting glucose
        boolean elevatedGlucose = false;
        if (snapshot.getLabHistory() != null) {
            LabValues labs = snapshot.getLabHistory().getLatestAsLabValues();
            if (labs != null) {
                if (labs.getGlucose() != null && labs.getGlucose() >= 100) {
                    elevatedGlucose = true;
                }
                if (labs.getHba1c() != null && labs.getHba1c() >= 5.7) {
                    elevatedGlucose = true;
                }
            }
        }
        if (elevatedGlucose || hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Prediabetes") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Diabetes")) {
            components++;
        }

        // Component 4: Low HDL cholesterol
        boolean male = "male".equalsIgnoreCase(snapshot.getGender());
        if (snapshot.getLabHistory() != null) {
            com.cardiofit.flink.models.LabResult hdlLab = snapshot.getLabHistory().getLatestByType("HDL");
            if (hdlLab != null && hdlLab.getValue() != null) {
                double hdl = hdlLab.getValue();
                if ((male && hdl < 40) || (!male && hdl < 50)) {
                    components++;
                }
            }
        }

        // Component 5: High triglycerides
        if (snapshot.getLabHistory() != null) {
            com.cardiofit.flink.models.LabResult trigLab = snapshot.getLabHistory().getLatestByType("Triglycerides");
            if (trigLab != null && trigLab.getValue() != null) {
                if (trigLab.getValue() >= 150) {
                    components++;
                }
            }
        }

        // Calculate risk as percentage of components present
        double risk = (double) components / totalComponents;

        // Adjust based on cohort membership
        if (snapshot.getRiskCohorts() != null &&
            snapshot.getRiskCohorts().contains("Urban Metabolic Syndrome Cohort")) {
            risk = Math.min(1.0, risk * 1.2); // 20% increase for cohort membership
        }

        return Math.round(risk * 100) / 100.0;
    }

    /**
     * Calculate CHADS-VASc score for stroke risk in AFib patients.
     */
    private int calculateCHADSVASc(PatientSnapshot snapshot) {
        int score = 0;

        // C - Congestive heart failure (1 point)
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Heart failure") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "CHF")) {
            score++;
        }

        // H - Hypertension (1 point)
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Hypertension")) {
            score++;
        }

        // A2 - Age ≥75 (2 points)
        if (snapshot.getAge() != null && snapshot.getAge() >= 75) {
            score += 2;
        }
        // A - Age 65-74 (1 point)
        else if (snapshot.getAge() != null && snapshot.getAge() >= 65) {
            score++;
        }

        // D - Diabetes (1 point)
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Diabetes")) {
            score++;
        }

        // S2 - Stroke/TIA/thromboembolism (2 points)
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Stroke") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "TIA") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "CVA")) {
            score += 2;
        }

        // V - Vascular disease (1 point)
        if (hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Coronary artery disease") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "CAD") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "Peripheral artery disease") ||
            hasCondition(conditionsToStrings(snapshot.getActiveConditions()), "PAD")) {
            score++;
        }

        // Sc - Sex category (female = 1 point)
        if ("female".equalsIgnoreCase(snapshot.getGender())) {
            score++;
        }

        return score;
    }

    /**
     * Calculate qSOFA (quick Sequential Organ Failure Assessment) for sepsis screening.
     */
    private int calculateqSOFA(PatientSnapshot snapshot) {
        int score = 0;

        // Get latest vitals
        if (snapshot.getVitalsHistory() != null && snapshot.getVitalsHistory().getLatest() != null) {
            VitalSign latestVitals = snapshot.getVitalsHistory().getLatest();

            // Respiratory rate ≥22
            if (latestVitals.getRespiratoryRate() != null && latestVitals.getRespiratoryRate() >= 22) {
                score++;
            }

            // Systolic BP ≤100
            if (latestVitals.getBloodPressureSystolic() != null) {
                int systolic = latestVitals.getBloodPressureSystolic().intValue();
                if (systolic > 0 && systolic <= 100) {
                    score++;
                }
            }

            // Altered mental status (GCS <15 or not "Alert")
            // Note: VitalSign doesn't have consciousness field, would need to check observations
            // For now, assume normal consciousness if not explicitly tracked
        }

        return score;
    }

    // Helper methods

    private boolean hasRequiredDataForFramingham(PatientSnapshot snapshot) {
        return snapshot.getAge() != null && snapshot.getGender() != null;
    }

    private boolean hasRelevantVitalsForSepsis(PatientSnapshot snapshot) {
        if (snapshot.getVitalsHistory() == null || snapshot.getVitalsHistory().getLatest() == null) {
            return false;
        }
        VitalSign latest = snapshot.getVitalsHistory().getLatest();
        return latest.getRespiratoryRate() != null ||
               (latest.getBloodPressureSystolic() != null && latest.getBloodPressureDiastolic() != null);
    }

    private boolean hasCondition(List<String> conditions, String condition) {
        if (conditions == null || condition == null) return false;
        return conditions.stream()
            .anyMatch(c -> c != null && c.toLowerCase().contains(condition.toLowerCase()));
    }

    /**
     * Convert List<Condition> to List<String> for condition checking
     */
    private List<String> conditionsToStrings(List<Condition> conditions) {
        if (conditions == null) return new ArrayList<>();
        return conditions.stream()
            .filter(c -> c != null)
            .map(c -> c.getDisplay() != null ? c.getDisplay() : c.getCode())
            .filter(s -> s != null)
            .collect(Collectors.toList());
    }

    /**
     * Check if patient is on a specific medication or medication class
     */
    private boolean isOnMedication(PatientSnapshot snapshot, String medicationName) {
        if (snapshot.getActiveMedications() == null || medicationName == null) {
            return false;
        }
        return snapshot.getActiveMedications().stream()
            .anyMatch(med -> med != null && med.getName() != null &&
                med.getName().toLowerCase().contains(medicationName.toLowerCase()));
    }

    /**
     * Calculate BMI from vitals (height and weight)
     * Returns null if height or weight not available
     *
     * Note: Current VitalSign model doesn't include height/weight fields.
     * In production, these would come from patient demographics or separate observations.
     */
    private Double getBMI(PatientSnapshot snapshot) {
        // TODO: Integrate with FHIR Observation resources for height/weight
        // For now, return null as BMI data not available in current model
        return null;
    }

    private String interpretFramingham(double risk) {
        if (risk < 0.05) return "LOW (<5%)";
        else if (risk < 0.10) return "BORDERLINE (5-10%)";
        else if (risk < 0.20) return "INTERMEDIATE (10-20%)";
        else return "HIGH (≥20%)";
    }

    private double getStrokeRiskFromCHADSVASc(int score) {
        // Annual stroke risk percentages based on CHA2DS2-VASc score
        switch (score) {
            case 0: return 0.0;
            case 1: return 1.3;
            case 2: return 2.2;
            case 3: return 3.2;
            case 4: return 4.0;
            case 5: return 6.7;
            case 6: return 9.8;
            case 7: return 9.6;
            case 8: return 12.5;
            case 9: return 15.2;
            default: return 15.2;
        }
    }

    private int parseSystolic(String bp) {
        try {
            if (bp != null && bp.contains("/")) {
                String[] parts = bp.split("/");
                return Integer.parseInt(parts[0].trim());
            }
        } catch (Exception e) {
            LOG.warn("Failed to parse systolic BP from: {}", bp);
        }
        return 0;
    }

    private int parseDiastolic(String bp) {
        try {
            if (bp != null && bp.contains("/")) {
                String[] parts = bp.split("/");
                return Integer.parseInt(parts[1].trim());
            }
        } catch (Exception e) {
            LOG.warn("Failed to parse diastolic BP from: {}", bp);
        }
        return 0;
    }

    private Integer extractInteger(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value instanceof Integer) {
            return (Integer) value;
        } else if (value instanceof Number) {
            return ((Number) value).intValue();
        } else if (value instanceof String) {
            try {
                return Integer.parseInt((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }

    private Double extractDouble(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value instanceof Double) {
            return (Double) value;
        } else if (value instanceof Number) {
            return ((Number) value).doubleValue();
        } else if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }
        return null;
    }
}