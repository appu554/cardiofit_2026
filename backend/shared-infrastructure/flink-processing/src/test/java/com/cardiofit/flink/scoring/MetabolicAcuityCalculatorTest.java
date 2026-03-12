package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for MetabolicAcuityCalculator
 *
 * Tests all 5 metabolic syndrome components and score calculations
 */
public class MetabolicAcuityCalculatorTest {

    @Test
    public void testNoMetabolicComponents() {
        // Normal patient with no metabolic syndrome components
        PatientSnapshot snapshot = createNormalPatient();
        Map<String, Object> vitals = createNormalVitals();
        Map<String, Object> labs = createNormalLabs();

        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        assertEquals(0.0, score.getScore());
        assertEquals(0, score.getComponentCount());
        assertEquals("LOW", score.getRiskLevel());
        assertFalse(score.isObesityPresent());
        assertFalse(score.isElevatedBPPresent());
        assertFalse(score.isElevatedGlucosePresent());
        assertFalse(score.isLowHDLPresent());
        assertFalse(score.isElevatedTriglyceridesPresent());
    }

    @Test
    public void testAllMetabolicComponents() {
        // Patient with all 5 metabolic syndrome components
        PatientSnapshot snapshot = createMetabolicSyndromePatient();
        Map<String, Object> vitals = createMetabolicSyndromeVitals();
        Map<String, Object> labs = createMetabolicSyndromeLabs();

        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        assertEquals(5.0, score.getScore());
        assertEquals(5, score.getComponentCount());
        assertEquals("HIGH", score.getRiskLevel());
        assertTrue(score.isObesityPresent());
        assertTrue(score.isElevatedBPPresent());
        assertTrue(score.isElevatedGlucosePresent());
        assertTrue(score.isLowHDLPresent());
        assertTrue(score.isElevatedTriglyceridesPresent());
        assertTrue(score.getInterpretation().contains("Metabolic syndrome present"));
    }

    @Test
    public void testThreeComponentsMetabolicSyndrome() {
        // Patient with exactly 3 components (metabolic syndrome threshold)
        PatientSnapshot snapshot = createPatient("male", 55);
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("systolicBP", 135); // Elevated BP
        vitals.put("diastolicBP", 88);
        vitals.put("bmi", 32.0); // Obesity

        Map<String, Object> labs = new HashMap<>();
        labs.put("hdlCholesterol", 35.0); // Low HDL for male
        labs.put("glucose", 95.0); // Normal
        labs.put("triglycerides", 140.0); // Normal

        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        assertEquals(3.0, score.getScore());
        assertEquals("HIGH", score.getRiskLevel());
        assertEquals(3, score.getPresentComponents().size());
    }

    @Test
    public void testGenderSpecificHDLThresholds() {
        PatientSnapshot snapshot = createPatient("male", 50);
        Map<String, Object> vitals = new HashMap<>();
        Map<String, Object> labs = new HashMap<>();
        labs.put("hdlCholesterol", 45.0); // Between male (40) and female (50) thresholds

        // For male: 45 > 40, so not low
        MetabolicAcuityCalculator.MetabolicAcuityScore scoreMale =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);
        assertFalse(scoreMale.isLowHDLPresent());

        // For female: 45 < 50, so is low
        snapshot.setGender("female");
        MetabolicAcuityCalculator.MetabolicAcuityScore scoreFemale =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);
        assertTrue(scoreFemale.isLowHDLPresent());
    }

    @Test
    public void testBMICalculationFromHeightWeight() {
        PatientSnapshot snapshot = createPatient("male", 50);
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("weight", 100.0); // kg
        vitals.put("height", 1.75); // meters
        // BMI = 100 / (1.75 * 1.75) = 32.65 > 30 → obesity

        Map<String, Object> labs = new HashMap<>();

        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        assertTrue(score.isObesityPresent());
    }

    @Test
    public void testModerateRiskTwoComponents() {
        PatientSnapshot snapshot = createPatient("male", 50);
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("systolicBP", 135);
        vitals.put("bmi", 31.0);

        Map<String, Object> labs = new HashMap<>();

        MetabolicAcuityCalculator.MetabolicAcuityScore score =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        assertEquals(2.0, score.getScore());
        assertEquals("MODERATE", score.getRiskLevel());
    }

    // Helper methods

    private PatientSnapshot createNormalPatient() {
        return createPatient("male", 45);
    }

    private PatientSnapshot createMetabolicSyndromePatient() {
        return createPatient("male", 58);
    }

    private PatientSnapshot createPatient(String gender, int age) {
        PatientSnapshot snapshot = new PatientSnapshot("TEST-PATIENT-001");
        snapshot.setGender(gender);
        snapshot.setAge(age);
        return snapshot;
    }

    private Map<String, Object> createNormalVitals() {
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("systolicBP", 118);
        vitals.put("diastolicBP", 76);
        vitals.put("bmi", 24.0);
        vitals.put("heartRate", 72);
        return vitals;
    }

    private Map<String, Object> createMetabolicSyndromeVitals() {
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("systolicBP", 145); // Elevated
        vitals.put("diastolicBP", 92); // Elevated
        vitals.put("bmi", 33.0); // Obese
        vitals.put("heartRate", 88);
        return vitals;
    }

    private Map<String, Object> createNormalLabs() {
        Map<String, Object> labs = new HashMap<>();
        labs.put("glucose", 92.0);
        labs.put("hdlCholesterol", 55.0);
        labs.put("triglycerides", 130.0);
        return labs;
    }

    private Map<String, Object> createMetabolicSyndromeLabs() {
        Map<String, Object> labs = new HashMap<>();
        labs.put("glucose", 115.0); // Elevated
        labs.put("hdlCholesterol", 35.0); // Low
        labs.put("triglycerides", 185.0); // Elevated
        return labs;
    }
}
