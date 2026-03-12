package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Clinical Score Calculators
 *
 * Implements evidence-based risk scoring systems used in clinical practice:
 * 1. Framingham Risk Score - 10-year cardiovascular disease risk
 * 2. CHA2DS2-VASc Score - Stroke risk in atrial fibrillation
 * 3. qSOFA Score - Quick sepsis screening tool
 * 4. Metabolic Syndrome Risk Score - Cardiometabolic risk assessment
 *
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 4
 */
public class ClinicalScoreCalculators implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ClinicalScoreCalculators.class);

    /**
     * Calculate Framingham Risk Score for 10-year CVD risk
     *
     * Uses simplified ATP III model (Adult Treatment Panel III)
     * Factors: Age, Gender, Total Cholesterol, HDL, Systolic BP, Smoking, Diabetes
     *
     * @param snapshot Patient snapshot with conditions and vitals
     * @param labs Lab results including lipid panel
     * @return FraminghamScore object with risk percentage and category
     */
    public static FraminghamScore calculateFraminghamScore(
            PatientSnapshot snapshot,
            Map<String, Object> labs) {

        FraminghamScore score = new FraminghamScore();

        try {
            // Extract patient demographics
            Integer age = extractAge(snapshot);
            String gender = extractGender(snapshot);

            if (age == null || gender == null) {
                score.setRiskPercentage(-1.0);
                score.setRiskCategory("INSUFFICIENT_DATA");
                return score;
            }

            // Extract risk factors
            Integer totalCholesterol = extractInteger(labs, "totalCholesterol");
            Integer hdlCholesterol = extractInteger(labs, "hdlCholesterol");
            Integer systolicBP = extractInteger(labs, "systolicBP");
            boolean smoking = hasCondition(snapshot, "smoking");
            boolean diabetes = hasCondition(snapshot, "diabetes") || hasCondition(snapshot, "E11");
            boolean onBPMeds = hasCondition(snapshot, "hypertension_treatment");

            // Calculate points based on gender
            int points = 0;

            if ("M".equalsIgnoreCase(gender) || "MALE".equalsIgnoreCase(gender)) {
                // Male scoring
                points += getAgePointsMale(age);
                points += getCholesterolPointsMale(age, totalCholesterol);
                points += getHDLPoints(hdlCholesterol);
                points += getSystolicBPPointsMale(systolicBP, onBPMeds);
                points += smoking ? 2 : 0;
                points += diabetes ? 2 : 0;

                // Convert points to risk percentage (male)
                double riskPercent = convertMalePointsToRisk(points);
                score.setRiskPercentage(riskPercent);
                score.setRiskCategory(categorizeRisk(riskPercent));
                score.setTotalPoints(points);

            } else {
                // Female scoring
                points += getAgePointsFemale(age);
                points += getCholesterolPointsFemale(age, totalCholesterol);
                points += getHDLPoints(hdlCholesterol);
                points += getSystolicBPPointsFemale(systolicBP, onBPMeds);
                points += smoking ? 2 : 0;
                points += diabetes ? 4 : 0;

                // Convert points to risk percentage (female)
                double riskPercent = convertFemalePointsToRisk(points);
                score.setRiskPercentage(riskPercent);
                score.setRiskCategory(categorizeRisk(riskPercent));
                score.setTotalPoints(points);
            }

            // Add details
            score.addDetail("Age", age.toString());
            score.addDetail("Gender", gender);
            score.addDetail("Total Cholesterol", totalCholesterol != null ? totalCholesterol.toString() : "N/A");
            score.addDetail("HDL", hdlCholesterol != null ? hdlCholesterol.toString() : "N/A");
            score.addDetail("Systolic BP", systolicBP != null ? systolicBP.toString() : "N/A");
            score.addDetail("Smoking", String.valueOf(smoking));
            score.addDetail("Diabetes", String.valueOf(diabetes));

            LOG.debug("Framingham score calculated: {}% risk, category: {}",
                score.getRiskPercentage(), score.getRiskCategory());

        } catch (Exception e) {
            LOG.error("Error calculating Framingham score", e);
            score.setRiskPercentage(-1.0);
            score.setRiskCategory("ERROR");
        }

        return score;
    }

    /**
     * Calculate CHA2DS2-VASc Score for stroke risk in AF patients
     *
     * Scoring:
     * C - Congestive heart failure (1 point)
     * H - Hypertension (1 point)
     * A2 - Age ≥75 (2 points)
     * D - Diabetes (1 point)
     * S2 - Prior stroke/TIA (2 points)
     * V - Vascular disease (1 point)
     * A - Age 65-74 (1 point)
     * Sc - Sex category (female = 1 point)
     *
     * Score 0: Low risk (0.2% annual stroke risk)
     * Score 1: Low-moderate risk (0.6-2% annual stroke risk)
     * Score ≥2: Moderate-high risk (>2% annual stroke risk) - anticoagulation recommended
     *
     * @param snapshot Patient snapshot with conditions
     * @return CHADS2VAScScore object with score and recommendation
     */
    public static CHADS2VAScScore calculateCHADS2VAScScore(PatientSnapshot snapshot) {
        CHADS2VAScScore score = new CHADS2VAScScore();

        try {
            int points = 0;

            // Extract patient data
            Integer age = extractAge(snapshot);
            String gender = extractGender(snapshot);

            // Check conditions
            boolean hasHF = hasCondition(snapshot, "I50") || hasCondition(snapshot, "heart_failure");
            boolean hasHTN = hasCondition(snapshot, "I10") || hasCondition(snapshot, "hypertension");
            boolean hasDM = hasCondition(snapshot, "E11") || hasCondition(snapshot, "diabetes");
            boolean hasStroke = hasCondition(snapshot, "I63") || hasCondition(snapshot, "stroke");
            boolean hasVascular = hasCondition(snapshot, "I25") || hasCondition(snapshot, "CAD");

            // Calculate points
            if (hasHF) {
                points += 1;
                score.addFactor("Congestive heart failure", 1);
            }

            if (hasHTN) {
                points += 1;
                score.addFactor("Hypertension", 1);
            }

            if (age != null && age >= 75) {
                points += 2;
                score.addFactor("Age ≥75", 2);
            } else if (age != null && age >= 65) {
                points += 1;
                score.addFactor("Age 65-74", 1);
            }

            if (hasDM) {
                points += 1;
                score.addFactor("Diabetes", 1);
            }

            if (hasStroke) {
                points += 2;
                score.addFactor("Prior stroke/TIA", 2);
            }

            if (hasVascular) {
                points += 1;
                score.addFactor("Vascular disease", 1);
            }

            if ("F".equalsIgnoreCase(gender) || "FEMALE".equalsIgnoreCase(gender)) {
                points += 1;
                score.addFactor("Female gender", 1);
            }

            score.setTotalScore(points);

            // Determine risk and recommendation
            if (points == 0) {
                score.setRiskCategory("LOW");
                score.setAnnualStrokeRisk("0.2%");
                score.setRecommendation("Consider no anticoagulation or aspirin");
            } else if (points == 1) {
                score.setRiskCategory("LOW_MODERATE");
                score.setAnnualStrokeRisk("0.6-2.0%");
                score.setRecommendation("Consider anticoagulation or aspirin");
            } else {
                score.setRiskCategory("MODERATE_HIGH");
                score.setAnnualStrokeRisk(">2.0%");
                score.setRecommendation("Anticoagulation recommended (unless contraindicated)");
            }

            LOG.debug("CHA2DS2-VASc score calculated: {} points, category: {}",
                points, score.getRiskCategory());

        } catch (Exception e) {
            LOG.error("Error calculating CHA2DS2-VASc score", e);
            score.setTotalScore(-1);
            score.setRiskCategory("ERROR");
        }

        return score;
    }

    /**
     * Calculate qSOFA (quick Sequential Organ Failure Assessment) Score
     *
     * Rapid sepsis screening tool. Positive if ≥2 of:
     * - Respiratory rate ≥22/min
     * - Altered mentation (GCS <15)
     * - Systolic BP ≤100 mmHg
     *
     * qSOFA ≥2: High risk of poor outcomes, consider sepsis workup
     *
     * @param vitals Current vital signs
     * @return qSOFAScore object with score and interpretation
     */
    public static qSOFAScore calculateQSOFAScore(Map<String, Object> vitals) {
        qSOFAScore score = new qSOFAScore();

        try {
            int points = 0;

            // Respiratory rate ≥22
            Integer rr = extractInteger(vitals, "respiratoryRate");
            if (rr != null && rr >= 22) {
                points += 1;
                score.addCriterion("Respiratory rate ≥22", true);
            } else {
                score.addCriterion("Respiratory rate ≥22", false);
            }

            // Altered mentation (GCS <15 or altered consciousness)
            Integer gcs = extractInteger(vitals, "glasgowComaScale");
            String consciousness = extractString(vitals, "consciousness");
            boolean alteredMentation = false;

            if (gcs != null && gcs < 15) {
                alteredMentation = true;
            } else if (consciousness != null &&
                       !consciousness.toUpperCase().contains("ALERT")) {
                alteredMentation = true;
            }

            if (alteredMentation) {
                points += 1;
                score.addCriterion("Altered mentation", true);
            } else {
                score.addCriterion("Altered mentation", false);
            }

            // Systolic BP ≤100
            Integer sbp = extractInteger(vitals, "systolicBP");
            if (sbp != null && sbp <= 100) {
                points += 1;
                score.addCriterion("Systolic BP ≤100", true);
            } else {
                score.addCriterion("Systolic BP ≤100", false);
            }

            score.setTotalScore(points);

            // Interpretation
            if (points >= 2) {
                score.setInterpretation("POSITIVE");
                score.setRiskLevel("HIGH");
                score.setRecommendation("Consider sepsis - Evaluate for organ dysfunction and infection");
            } else {
                score.setInterpretation("NEGATIVE");
                score.setRiskLevel("LOW");
                score.setRecommendation("Low risk of sepsis-related complications");
            }

            LOG.debug("qSOFA score calculated: {} points, interpretation: {}",
                points, score.getInterpretation());

        } catch (Exception e) {
            LOG.error("Error calculating qSOFA score", e);
            score.setTotalScore(-1);
            score.setInterpretation("ERROR");
        }

        return score;
    }

    // Helper methods for Framingham calculations

    private static int getAgePointsMale(int age) {
        if (age < 35) return -1;
        if (age < 40) return 0;
        if (age < 45) return 1;
        if (age < 50) return 2;
        if (age < 55) return 3;
        if (age < 60) return 4;
        if (age < 65) return 5;
        if (age < 70) return 6;
        return 7;
    }

    private static int getAgePointsFemale(int age) {
        if (age < 35) return -9;
        if (age < 40) return -4;
        if (age < 45) return 0;
        if (age < 50) return 3;
        if (age < 55) return 6;
        if (age < 60) return 7;
        if (age < 65) return 8;
        if (age < 70) return 8;
        return 8;
    }

    private static int getCholesterolPointsMale(int age, Integer cholesterol) {
        if (cholesterol == null) return 0;
        if (age < 40) {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 1;
            if (cholesterol < 240) return 2;
            if (cholesterol < 280) return 3;
            return 4;
        } else if (age < 50) {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 1;
            if (cholesterol < 240) return 2;
            if (cholesterol < 280) return 3;
            return 4;
        } else {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 0;
            if (cholesterol < 240) return 1;
            if (cholesterol < 280) return 2;
            return 3;
        }
    }

    private static int getCholesterolPointsFemale(int age, Integer cholesterol) {
        if (cholesterol == null) return 0;
        if (age < 40) {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 1;
            if (cholesterol < 240) return 3;
            if (cholesterol < 280) return 4;
            return 5;
        } else if (age < 50) {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 1;
            if (cholesterol < 240) return 2;
            if (cholesterol < 280) return 3;
            return 4;
        } else {
            if (cholesterol < 160) return 0;
            if (cholesterol < 200) return 0;
            if (cholesterol < 240) return 1;
            if (cholesterol < 280) return 1;
            return 2;
        }
    }

    private static int getHDLPoints(Integer hdl) {
        if (hdl == null) return 0;
        if (hdl < 35) return 2;
        if (hdl < 45) return 1;
        if (hdl < 50) return 0;
        if (hdl < 60) return 0;
        return -1;
    }

    private static int getSystolicBPPointsMale(Integer sbp, boolean onMeds) {
        if (sbp == null) return 0;
        if (onMeds) {
            if (sbp < 120) return -2;
            if (sbp < 130) return 0;
            if (sbp < 140) return 1;
            if (sbp < 160) return 2;
            return 3;
        } else {
            if (sbp < 120) return 0;
            if (sbp < 130) return 0;
            if (sbp < 140) return 1;
            if (sbp < 160) return 2;
            return 3;
        }
    }

    private static int getSystolicBPPointsFemale(Integer sbp, boolean onMeds) {
        if (sbp == null) return 0;
        if (onMeds) {
            if (sbp < 120) return -3;
            if (sbp < 130) return 0;
            if (sbp < 140) return 2;
            if (sbp < 160) return 3;
            return 5;
        } else {
            if (sbp < 120) return -1;
            if (sbp < 130) return 0;
            if (sbp < 140) return 1;
            if (sbp < 160) return 3;
            return 4;
        }
    }

    private static double convertMalePointsToRisk(int points) {
        // Simplified conversion (actual Framingham uses more complex calculation)
        if (points < 0) return 1.0;
        if (points < 5) return 2.0;
        if (points < 10) return 5.0;
        if (points < 12) return 8.0;
        if (points < 14) return 12.0;
        if (points < 16) return 16.0;
        return 25.0;
    }

    private static double convertFemalePointsToRisk(int points) {
        // Simplified conversion (actual Framingham uses more complex calculation)
        if (points < 0) return 1.0;
        if (points < 5) return 1.5;
        if (points < 10) return 3.0;
        if (points < 15) return 6.0;
        if (points < 20) return 11.0;
        if (points < 23) return 17.0;
        return 27.0;
    }

    private static String categorizeRisk(double riskPercent) {
        if (riskPercent < 5) return "LOW";
        if (riskPercent < 10) return "MODERATE";
        if (riskPercent < 20) return "HIGH";
        return "VERY_HIGH";
    }

    // Helper methods for data extraction

    private static Integer extractAge(PatientSnapshot snapshot) {
        // Extract age from snapshot - implementation depends on PatientSnapshot structure
        return 50; // Placeholder
    }

    private static String extractGender(PatientSnapshot snapshot) {
        // Extract gender from snapshot
        return "M"; // Placeholder
    }

    private static boolean hasCondition(PatientSnapshot snapshot, String condition) {
        // Check if patient has specific condition
        return false; // Placeholder
    }

    private static Integer extractInteger(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Integer) return (Integer) value;
        if (value instanceof Number) return ((Number) value).intValue();
        try {
            return Integer.parseInt(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static String extractString(Map<String, Object> map, String key) {
        Object value = map.get(key);
        return value != null ? value.toString() : null;
    }

    // Result classes

    public static class FraminghamScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private double riskPercentage;
        private String riskCategory;
        private int totalPoints;
        private Map<String, String> details = new HashMap<>();

        public double getRiskPercentage() { return riskPercentage; }
        public void setRiskPercentage(double riskPercentage) {
            this.riskPercentage = riskPercentage;
        }

        public String getRiskCategory() { return riskCategory; }
        public void setRiskCategory(String riskCategory) {
            this.riskCategory = riskCategory;
        }

        public int getTotalPoints() { return totalPoints; }
        public void setTotalPoints(int totalPoints) {
            this.totalPoints = totalPoints;
        }

        public Map<String, String> getDetails() { return details; }
        public void addDetail(String key, String value) {
            this.details.put(key, value);
        }

        @Override
        public String toString() {
            return "FraminghamScore{" +
                    "risk=" + riskPercentage + "%" +
                    ", category='" + riskCategory + '\'' +
                    '}';
        }
    }

    public static class CHADS2VAScScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private int totalScore;
        private String riskCategory;
        private String annualStrokeRisk;
        private String recommendation;
        private Map<String, Integer> factors = new HashMap<>();

        public int getTotalScore() { return totalScore; }
        public void setTotalScore(int totalScore) {
            this.totalScore = totalScore;
        }

        public String getRiskCategory() { return riskCategory; }
        public void setRiskCategory(String riskCategory) {
            this.riskCategory = riskCategory;
        }

        public String getAnnualStrokeRisk() { return annualStrokeRisk; }
        public void setAnnualStrokeRisk(String annualStrokeRisk) {
            this.annualStrokeRisk = annualStrokeRisk;
        }

        public String getRecommendation() { return recommendation; }
        public void setRecommendation(String recommendation) {
            this.recommendation = recommendation;
        }

        public Map<String, Integer> getFactors() { return factors; }
        public void addFactor(String factor, int points) {
            this.factors.put(factor, points);
        }

        @Override
        public String toString() {
            return "CHADS2VAScScore{" +
                    "score=" + totalScore +
                    ", risk='" + riskCategory + '\'' +
                    ", strokeRisk='" + annualStrokeRisk + '\'' +
                    '}';
        }
    }

    public static class qSOFAScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private int totalScore;
        private String interpretation;
        private String riskLevel;
        private String recommendation;
        private Map<String, Boolean> criteria = new HashMap<>();

        public int getTotalScore() { return totalScore; }
        public void setTotalScore(int totalScore) {
            this.totalScore = totalScore;
        }

        public String getInterpretation() { return interpretation; }
        public void setInterpretation(String interpretation) {
            this.interpretation = interpretation;
        }

        public String getRiskLevel() { return riskLevel; }
        public void setRiskLevel(String riskLevel) {
            this.riskLevel = riskLevel;
        }

        public String getRecommendation() { return recommendation; }
        public void setRecommendation(String recommendation) {
            this.recommendation = recommendation;
        }

        public Map<String, Boolean> getCriteria() { return criteria; }
        public void addCriterion(String criterion, boolean met) {
            this.criteria.put(criterion, met);
        }

        @Override
        public String toString() {
            return "qSOFAScore{" +
                    "score=" + totalScore +
                    ", interpretation='" + interpretation + '\'' +
                    '}';
        }
    }

    /**
     * Calculate Metabolic Syndrome Risk Score
     *
     * Assesses cardiometabolic risk based on 5 core metabolic syndrome components:
     * 1. Central obesity (BMI ≥30 or waist circumference)
     * 2. Elevated blood pressure (≥130/85 mmHg)
     * 3. Elevated fasting glucose (≥100 mg/dL)
     * 4. Low HDL cholesterol (<40 mg/dL men, <50 mg/dL women)
     * 5. Elevated triglycerides (≥150 mg/dL)
     *
     * Score: Ratio of present components (0.0 to 1.0)
     * ≥0.6 (3+ components) indicates metabolic syndrome
     *
     * Reference: NCEP ATP III criteria
     * Spec reference: MODULE2_ADVANCED_ENHANCEMENTS.md lines 188-198
     *
     * @param snapshot Patient snapshot with demographics and conditions
     * @param vitals Current vital signs
     * @param labs Lab results including lipids and glucose
     * @return MetabolicSyndromeScore with risk ratio and interpretation
     */
    public static MetabolicSyndromeScore calculateMetabolicSyndromeScore(
            PatientSnapshot snapshot,
            Map<String, Object> vitals,
            Map<String, Object> labs) {

        MetabolicSyndromeScore score = new MetabolicSyndromeScore();
        int componentCount = 0;

        // Component 1: Central Obesity
        Double bmi = extractDouble(labs, "bmi");
        if (bmi == null) {
            bmi = extractDouble(vitals, "bmi");
        }
        // Alternative: calculate from height/weight
        if (bmi == null) {
            Double weight = extractDouble(vitals, "weight");
            Double height = extractDouble(vitals, "height");
            if (weight != null && height != null && height > 0) {
                bmi = weight / (height * height);
            }
        }
        if (bmi != null && bmi >= 30.0) {
            componentCount++;
            score.setObesityPresent(true);
        }

        // Component 2: Elevated Blood Pressure
        Integer systolic = extractInteger(vitals, "systolicBP");
        Integer diastolic = extractInteger(vitals, "diastolicBP");
        if ((systolic != null && systolic >= 130) || (diastolic != null && diastolic >= 85)) {
            componentCount++;
            score.setElevatedBPPresent(true);
        }

        // Component 3: Elevated Fasting Glucose
        Double glucose = extractDouble(labs, "glucose");
        if (glucose == null) {
            glucose = extractDouble(labs, "fastingGlucose");
        }
        if (glucose != null && glucose >= 100.0) {
            componentCount++;
            score.setElevatedGlucosePresent(true);
        }

        // Component 4: Low HDL Cholesterol (gender-specific)
        Double hdl = extractDouble(labs, "hdlCholesterol");
        if (hdl == null) {
            hdl = extractDouble(labs, "hdl");
        }
        if (hdl != null) {
            String gender = extractGender(snapshot);
            boolean lowHDL = false;
            if ("M".equalsIgnoreCase(gender) || "MALE".equalsIgnoreCase(gender)) {
                lowHDL = hdl < 40.0;
            } else if ("F".equalsIgnoreCase(gender) || "FEMALE".equalsIgnoreCase(gender)) {
                lowHDL = hdl < 50.0;
            }
            if (lowHDL) {
                componentCount++;
                score.setLowHDLPresent(true);
            }
        }

        // Component 5: Elevated Triglycerides
        Double triglycerides = extractDouble(labs, "triglycerides");
        if (triglycerides == null) {
            triglycerides = extractDouble(labs, "trig");
        }
        if (triglycerides != null && triglycerides >= 150.0) {
            componentCount++;
            score.setElevatedTriglyceridesPresent(true);
        }

        // Calculate risk score as ratio
        double riskScore = (double) componentCount / 5.0;
        score.setRiskScore(riskScore);
        score.setComponentCount(componentCount);

        // Categorize risk
        String category;
        String interpretation;
        if (componentCount >= 3) {
            category = "HIGH";
            interpretation = String.format("Metabolic syndrome present (%d/5 components). Clinical intervention indicated.", componentCount);
        } else if (componentCount == 2) {
            category = "MODERATE";
            interpretation = "2/5 metabolic syndrome components. Lifestyle modification recommended.";
        } else if (componentCount == 1) {
            category = "LOW_MODERATE";
            interpretation = "1/5 metabolic syndrome component. Monitor for progression.";
        } else {
            category = "LOW";
            interpretation = "No metabolic syndrome components detected.";
        }

        score.setRiskCategory(category);
        score.setInterpretation(interpretation);

        LOG.debug("Metabolic syndrome score calculated: {}/5 components present, risk={}",
            componentCount, riskScore);

        return score;
    }

    // Helper method for extracting Double values
    private static Double extractDouble(Map<String, Object> map, String key) {
        if (map == null) return null;
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Double) return (Double) value;
        if (value instanceof Number) return ((Number) value).doubleValue();
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    /**
     * Metabolic Syndrome Score Result Class
     */
    public static class MetabolicSyndromeScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private double riskScore; // 0.0 to 1.0 (componentCount / 5)
        private int componentCount; // 0 to 5
        private String riskCategory; // LOW, LOW_MODERATE, MODERATE, HIGH
        private String interpretation;

        // Individual component flags
        private boolean obesityPresent;
        private boolean elevatedBPPresent;
        private boolean elevatedGlucosePresent;
        private boolean lowHDLPresent;
        private boolean elevatedTriglyceridesPresent;

        // Getters and Setters

        public double getRiskScore() {
            return riskScore;
        }

        public void setRiskScore(double riskScore) {
            this.riskScore = riskScore;
        }

        public int getComponentCount() {
            return componentCount;
        }

        public void setComponentCount(int componentCount) {
            this.componentCount = componentCount;
        }

        public String getRiskCategory() {
            return riskCategory;
        }

        public void setRiskCategory(String riskCategory) {
            this.riskCategory = riskCategory;
        }

        public String getInterpretation() {
            return interpretation;
        }

        public void setInterpretation(String interpretation) {
            this.interpretation = interpretation;
        }

        public boolean isObesityPresent() {
            return obesityPresent;
        }

        public void setObesityPresent(boolean obesityPresent) {
            this.obesityPresent = obesityPresent;
        }

        public boolean isElevatedBPPresent() {
            return elevatedBPPresent;
        }

        public void setElevatedBPPresent(boolean elevatedBPPresent) {
            this.elevatedBPPresent = elevatedBPPresent;
        }

        public boolean isElevatedGlucosePresent() {
            return elevatedGlucosePresent;
        }

        public void setElevatedGlucosePresent(boolean elevatedGlucosePresent) {
            this.elevatedGlucosePresent = elevatedGlucosePresent;
        }

        public boolean isLowHDLPresent() {
            return lowHDLPresent;
        }

        public void setLowHDLPresent(boolean lowHDLPresent) {
            this.lowHDLPresent = lowHDLPresent;
        }

        public boolean isElevatedTriglyceridesPresent() {
            return elevatedTriglyceridesPresent;
        }

        public void setElevatedTriglyceridesPresent(boolean elevatedTriglyceridesPresent) {
            this.elevatedTriglyceridesPresent = elevatedTriglyceridesPresent;
        }

        @Override
        public String toString() {
            return "MetabolicSyndromeScore{" +
                    "riskScore=" + riskScore +
                    ", componentCount=" + componentCount +
                    ", category='" + riskCategory + '\'' +
                    '}';
        }
    }
}