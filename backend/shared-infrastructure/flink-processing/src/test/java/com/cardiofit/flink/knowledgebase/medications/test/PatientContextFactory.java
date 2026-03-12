package com.cardiofit.flink.knowledgebase.medications.test;

import com.cardiofit.flink.models.PatientContext;
import java.util.List;

/**
 * Factory class for creating test PatientContext instances with various clinical scenarios.
 *
 * Provides standardized patient test data for:
 * - Standard adult patients
 * - Renal impairment scenarios
 * - Hepatic impairment scenarios
 * - Pediatric patients
 * - Neonatal patients
 * - Geriatric patients
 * - Obese patients
 * - Dialysis patients
 */
public class PatientContextFactory {

    /**
     * Creates a standard adult patient with normal organ function.
     * Age: 45, Weight: 70kg, Height: 1.75m, SCr: 1.0, CrCl: ~90
     */
    public static PatientContext createStandardAdult() {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        return pc;
    }

    /**
     * Creates a patient with specific creatinine clearance (CrCl).
     * Uses Cockcroft-Gault formula to calculate required serum creatinine.
     *
     * @param targetCrCl Desired creatinine clearance (mL/min)
     * @return PatientContext with calculated creatinine to achieve target CrCl
     */
    public static PatientContext createPatientWithCrCl(double targetCrCl) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        // Cockcroft-Gault: CrCl = (140-age) * weight / (72 * SCr)
        // Solving for SCr: SCr = (140-age) * weight / (72 * CrCl)
        double creatinine = ((140 - pc.getAge()) * pc.getWeight()) / (72 * targetCrCl);
        pc.setCreatinine(creatinine);
        return pc;
    }

    /**
     * Creates a pediatric patient.
     *
     * @param weight Weight in kg
     * @param ageYears Age in years (1-17)
     * @return PatientContext for pediatric patient
     */
    public static PatientContext createPediatricPatient(double weight, int ageYears) {
        PatientContext pc = new PatientContext();
        pc.setWeight(weight);
        pc.setAge(ageYears);
        pc.setAgeCategory("PEDIATRIC");
        pc.setCreatinine(0.3 + (ageYears * 0.05)); // Age-appropriate SCr
        return pc;
    }

    /**
     * Creates a neonatal patient (<1 month old).
     *
     * @param weight Weight in kg
     * @param ageMonths Age in months (0-1)
     * @return PatientContext for neonate
     */
    public static PatientContext createNeonatePatient(double weight, double ageMonths) {
        PatientContext pc = new PatientContext();
        pc.setWeight(weight);
        pc.setAge(0);
        pc.setAgeMonths(ageMonths);
        pc.setAgeCategory("NEONATE");
        pc.setCreatinine(0.2); // Neonatal SCr

        if (weight < 1.0) {
            pc.setAgeCategory("PREMATURE");
        }
        return pc;
    }

    /**
     * Creates a geriatric patient (age > 65).
     *
     * @param weight Weight in kg
     * @param creatinine Serum creatinine
     * @param age Age in years
     * @param sex "M" or "F"
     * @return PatientContext for geriatric patient
     */
    public static PatientContext createGeriatricPatient(double weight, double creatinine, int age, String sex) {
        PatientContext pc = new PatientContext();
        pc.setAge(age);
        pc.setWeight(weight);
        pc.setHeight(1.75);
        pc.setCreatinine(creatinine);
        pc.setSex(sex);
        pc.setAgeCategory("GERIATRIC");
        return pc;
    }

    /**
     * Creates an obese patient with specified weight and height.
     * Automatically calculates BMI.
     *
     * @param weight Weight in kg
     * @param height Height in meters
     * @return PatientContext for obese patient
     */
    public static PatientContext createObesePatient(double weight, double height) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(weight);
        pc.setHeight(height);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        double bmi = weight / (height * height);
        pc.setBMI(bmi);

        if (bmi > 50) {
            pc.setObesityCategory("MORBID");
        } else if (bmi > 40) {
            pc.setObesityCategory("SEVERE");
        } else if (bmi > 30) {
            pc.setObesityCategory("OBESE");
        }
        return pc;
    }

    /**
     * Creates a patient with hepatic impairment.
     *
     * @param childPughGrade Child-Pugh classification: "A", "B", or "C"
     * @return PatientContext with hepatic impairment
     */
    public static PatientContext createPatientWithChildPugh(String childPughGrade) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setChildPughScore(childPughGrade);
        pc.setHepaticImpairment(true);

        switch (childPughGrade) {
            case "C":
                pc.setHepaticImpairmentSeverity("SEVERE");
                break;
            case "B":
                pc.setHepaticImpairmentSeverity("MODERATE");
                break;
            case "A":
                pc.setHepaticImpairmentSeverity("MILD");
                break;
        }
        return pc;
    }

    /**
     * Creates a patient on hemodialysis.
     *
     * @return PatientContext for hemodialysis patient
     */
    public static PatientContext createHemodialysisPatient() {
        PatientContext pc = createPatientWithCrCl(5.0); // ESRD
        pc.setOnDialysis(true);
        pc.setDialysisType("HEMODIALYSIS");
        pc.setDialysisSchedule("Monday-Wednesday-Friday");
        return pc;
    }

    /**
     * Creates a patient with specific allergies.
     *
     * @param allergies List of allergy strings
     * @return PatientContext with allergies
     */
    public static PatientContext createPatientWithAllergies(List<String> allergies) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setAllergies(allergies);
        return pc;
    }

    /**
     * Creates a patient with specific diagnoses.
     *
     * @param diagnoses List of diagnosis strings
     * @return PatientContext with diagnoses
     */
    public static PatientContext createPatientWithDiagnoses(List<String> diagnoses) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setDiagnoses(diagnoses);
        return pc;
    }

    /**
     * Creates a patient with active medications.
     *
     * @param medications List of active medication names
     * @return PatientContext with active medications
     */
    public static PatientContext createPatientWithMedications(List<String> medications) {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setActiveMedicationsFromList(medications);
        return pc;
    }

    /**
     * Creates a complex patient with multiple comorbidities.
     * Simulates a typical ICU or complex care patient.
     */
    public static PatientContext createComplexPatient() {
        PatientContext pc = createGeriatricPatient(60.0, 1.5, 75, "F");
        pc.setDiagnoses(List.of("Heart Failure NYHA Class III", "CKD Stage 4", "Type 2 Diabetes"));
        pc.setActiveMedicationsFromList(List.of("Furosemide", "Lisinopril", "Metoprolol", "Insulin", "Aspirin"));
        pc.setAllergies(List.of("penicillin"));
        return pc;
    }

    /**
     * Creates a STEMI patient for cardiovascular testing.
     */
    public static PatientContext createSTEMIPatient() {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(70.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.0);
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setDiagnosis("STEMI");
        pc.setActiveMedicationsFromList(List.of());
        return pc;
    }

    /**
     * Creates a septic patient for antibiotic testing.
     */
    public static PatientContext createSepsisPatient() {
        PatientContext pc = new PatientContext();
        pc.setAge(45);
        pc.setWeight(85.0);
        pc.setHeight(1.75);
        pc.setCreatinine(1.8); // Mild AKI
        pc.setSex("M");
        pc.setAgeCategory("ADULT");
        pc.setDiagnosis("Septic Shock");
        return pc;
    }
}
