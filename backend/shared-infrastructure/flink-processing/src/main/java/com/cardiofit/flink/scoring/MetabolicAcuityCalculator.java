package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;

import java.io.Serializable;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Metabolic Acuity Score Calculator
 *
 * Calculates metabolic risk based on 5 core metabolic syndrome components:
 * 1. Central Obesity (BMI ≥ 30 or waist circumference)
 * 2. Elevated Blood Pressure (≥130/85 mmHg)
 * 3. Elevated Glucose (fasting ≥100 mg/dL)
 * 4. Low HDL Cholesterol (<40 mg/dL men, <50 mg/dL women)
 * 5. Elevated Triglycerides (≥150 mg/dL)
 *
 * Score range: 0.0 to 5.0 (count of present components)
 *
 * Clinical Interpretation:
 * - 0-1: Low metabolic risk
 * - 2: Moderate metabolic risk
 * - 3+: High metabolic risk (meets metabolic syndrome criteria)
 *
 * Reference: National Cholesterol Education Program Adult Treatment Panel III (NCEP ATP III)
 *
 * @see MODULE2_ADVANCED_ENHANCEMENTS.md lines 96-112
 */
public class MetabolicAcuityCalculator {

    // Metabolic syndrome thresholds (NCEP ATP III criteria)
    private static final double BMI_OBESITY_THRESHOLD = 30.0;
    private static final int BP_SYSTOLIC_THRESHOLD = 130;
    private static final int BP_DIASTOLIC_THRESHOLD = 85;
    private static final double GLUCOSE_THRESHOLD = 100.0; // mg/dL (fasting)
    private static final double HDL_MALE_THRESHOLD = 40.0; // mg/dL
    private static final double HDL_FEMALE_THRESHOLD = 50.0; // mg/dL
    private static final double TRIGLYCERIDES_THRESHOLD = 150.0; // mg/dL

    /**
     * Calculate metabolic acuity score
     *
     * @param snapshot Patient clinical snapshot
     * @param vitals Current vital signs
     * @param labs Current lab results
     * @return Metabolic acuity score result
     */
    public static MetabolicAcuityScore calculate(
            PatientSnapshot snapshot,
            Map<String, Object> vitals,
            Map<String, Object> labs) {

        MetabolicAcuityScore score = new MetabolicAcuityScore();
        score.setCalculationTimestamp(System.currentTimeMillis());

        List<String> presentComponents = new ArrayList<>();
        int componentCount = 0;

        // Component 1: Central Obesity (BMI-based assessment)
        if (hasObesity(snapshot, vitals, labs)) {
            componentCount++;
            presentComponents.add("Central Obesity");
            score.setObesityPresent(true);
        }

        // Component 2: Elevated Blood Pressure
        if (hasElevatedBloodPressure(vitals)) {
            componentCount++;
            presentComponents.add("Elevated Blood Pressure");
            score.setElevatedBPPresent(true);
        }

        // Component 3: Elevated Glucose
        if (hasElevatedGlucose(labs)) {
            componentCount++;
            presentComponents.add("Elevated Glucose");
            score.setElevatedGlucosePresent(true);
        }

        // Component 4: Low HDL Cholesterol
        if (hasLowHDL(snapshot, labs)) {
            componentCount++;
            presentComponents.add("Low HDL Cholesterol");
            score.setLowHDLPresent(true);
        }

        // Component 5: Elevated Triglycerides
        if (hasElevatedTriglycerides(labs)) {
            componentCount++;
            presentComponents.add("Elevated Triglycerides");
            score.setElevatedTriglyceridesPresent(true);
        }

        // Set final score
        score.setScore((double) componentCount);
        score.setComponentCount(componentCount);
        score.setPresentComponents(presentComponents);

        // Determine risk level
        String riskLevel;
        if (componentCount >= 3) {
            riskLevel = "HIGH"; // Meets metabolic syndrome criteria
        } else if (componentCount == 2) {
            riskLevel = "MODERATE";
        } else {
            riskLevel = "LOW";
        }
        score.setRiskLevel(riskLevel);

        // Clinical interpretation
        score.setInterpretation(generateInterpretation(componentCount, presentComponents));

        return score;
    }

    /**
     * Check for central obesity (BMI-based)
     */
    private static boolean hasObesity(PatientSnapshot snapshot, Map<String, Object> vitals, Map<String, Object> labs) {
        // Try to get BMI from labs or vitals
        Double bmi = extractDouble(labs, "bmi");
        if (bmi == null) {
            bmi = extractDouble(vitals, "bmi");
        }

        // If BMI available, use it directly
        if (bmi != null && bmi >= BMI_OBESITY_THRESHOLD) {
            return true;
        }

        // Alternative: Calculate from height and weight if available
        Double weight = extractDouble(vitals, "weight"); // kg
        Double height = extractDouble(vitals, "height"); // meters

        if (weight != null && height != null && height > 0) {
            double calculatedBMI = weight / (height * height);
            return calculatedBMI >= BMI_OBESITY_THRESHOLD;
        }

        // Alternative: Waist circumference (>102cm men, >88cm women)
        Double waistCircumference = extractDouble(vitals, "waistCircumference"); // cm
        if (waistCircumference != null) {
            String gender = snapshot.getGender();
            if ("male".equalsIgnoreCase(gender) && waistCircumference > 102) {
                return true;
            } else if ("female".equalsIgnoreCase(gender) && waistCircumference > 88) {
                return true;
            }
        }

        return false;
    }

    /**
     * Check for elevated blood pressure
     */
    private static boolean hasElevatedBloodPressure(Map<String, Object> vitals) {
        Integer systolic = extractInteger(vitals, "systolicBP");
        Integer diastolic = extractInteger(vitals, "diastolicBP");

        if (systolic != null && systolic >= BP_SYSTOLIC_THRESHOLD) {
            return true;
        }

        if (diastolic != null && diastolic >= BP_DIASTOLIC_THRESHOLD) {
            return true;
        }

        return false;
    }

    /**
     * Check for elevated fasting glucose
     */
    private static boolean hasElevatedGlucose(Map<String, Object> labs) {
        Double glucose = extractDouble(labs, "glucose");
        if (glucose == null) {
            glucose = extractDouble(labs, "fastingGlucose");
        }
        if (glucose == null) {
            glucose = extractDouble(labs, "bloodGlucose");
        }

        // Check for HbA1c as alternative (≥5.7% indicates prediabetes)
        if (glucose == null) {
            Double hba1c = extractDouble(labs, "hba1c");
            if (hba1c != null && hba1c >= 5.7) {
                return true;
            }
        }

        return glucose != null && glucose >= GLUCOSE_THRESHOLD;
    }

    /**
     * Check for low HDL cholesterol (gender-specific thresholds)
     */
    private static boolean hasLowHDL(PatientSnapshot snapshot, Map<String, Object> labs) {
        Double hdl = extractDouble(labs, "hdlCholesterol");
        if (hdl == null) {
            hdl = extractDouble(labs, "hdl");
        }

        if (hdl == null) {
            return false;
        }

        String gender = snapshot.getGender();
        if ("male".equalsIgnoreCase(gender)) {
            return hdl < HDL_MALE_THRESHOLD;
        } else if ("female".equalsIgnoreCase(gender)) {
            return hdl < HDL_FEMALE_THRESHOLD;
        }

        // Unknown gender - use male threshold as default
        return hdl < HDL_MALE_THRESHOLD;
    }

    /**
     * Check for elevated triglycerides
     */
    private static boolean hasElevatedTriglycerides(Map<String, Object> labs) {
        Double triglycerides = extractDouble(labs, "triglycerides");
        if (triglycerides == null) {
            triglycerides = extractDouble(labs, "trig");
        }

        return triglycerides != null && triglycerides >= TRIGLYCERIDES_THRESHOLD;
    }

    /**
     * Generate clinical interpretation
     */
    private static String generateInterpretation(int componentCount, List<String> components) {
        if (componentCount == 0) {
            return "No metabolic syndrome components detected. Low metabolic risk.";
        } else if (componentCount == 1) {
            return String.format("1 metabolic syndrome component present: %s. Monitor for progression.",
                components.get(0));
        } else if (componentCount == 2) {
            return String.format("2 metabolic syndrome components present: %s. Moderate metabolic risk. Lifestyle modification recommended.",
                String.join(", ", components));
        } else {
            return String.format("%d metabolic syndrome components present: %s. Meets criteria for metabolic syndrome. Clinical intervention indicated.",
                componentCount, String.join(", ", components));
        }
    }

    // Helper methods for type-safe extraction

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

    private static Integer extractInteger(Map<String, Object> map, String key) {
        if (map == null) return null;
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

    /**
     * Metabolic Acuity Score Result Class
     */
    public static class MetabolicAcuityScore implements Serializable {
        private static final long serialVersionUID = 1L;

        private double score; // 0.0 to 5.0
        private int componentCount;
        private String riskLevel; // LOW, MODERATE, HIGH
        private String interpretation;
        private long calculationTimestamp;

        // Individual component flags
        private boolean obesityPresent;
        private boolean elevatedBPPresent;
        private boolean elevatedGlucosePresent;
        private boolean lowHDLPresent;
        private boolean elevatedTriglyceridesPresent;

        private List<String> presentComponents = new ArrayList<>();

        // Getters and Setters

        public double getScore() {
            return score;
        }

        public void setScore(double score) {
            this.score = score;
        }

        public int getComponentCount() {
            return componentCount;
        }

        public void setComponentCount(int componentCount) {
            this.componentCount = componentCount;
        }

        public String getRiskLevel() {
            return riskLevel;
        }

        public void setRiskLevel(String riskLevel) {
            this.riskLevel = riskLevel;
        }

        public String getInterpretation() {
            return interpretation;
        }

        public void setInterpretation(String interpretation) {
            this.interpretation = interpretation;
        }

        public long getCalculationTimestamp() {
            return calculationTimestamp;
        }

        public void setCalculationTimestamp(long calculationTimestamp) {
            this.calculationTimestamp = calculationTimestamp;
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

        public List<String> getPresentComponents() {
            return presentComponents;
        }

        public void setPresentComponents(List<String> presentComponents) {
            this.presentComponents = presentComponents;
        }

        @Override
        public String toString() {
            return "MetabolicAcuityScore{" +
                    "score=" + score +
                    ", componentCount=" + componentCount +
                    ", riskLevel='" + riskLevel + '\'' +
                    ", components=" + presentComponents +
                    '}';
        }
    }
}
