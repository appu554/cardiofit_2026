package com.cardiofit.flink.knowledgebase.medications;

import com.cardiofit.flink.knowledgebase.medications.calculator.DoseCalculator;
import com.cardiofit.flink.knowledgebase.medications.calculator.CalculatedDose;
import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker;
import com.cardiofit.flink.knowledgebase.medications.safety.Interaction;
import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker.InteractionSeverity;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import com.cardiofit.flink.knowledgebase.medications.test.PatientContextFactory;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContext;
import com.cardiofit.flink.models.PatientState;
import org.junit.jupiter.api.*;

import java.util.Arrays;
import java.util.List;
import java.util.HashMap;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Edge case tests for Medication Database.
 *
 * Tests extreme clinical scenarios and boundary conditions.
 * Coverage Target: Edge case coverage
 * Test Count: 5 edge case tests
 */
@DisplayName("Medication Database Edge Case Test Suite")
@Tag("edge-case")
class MedicationDatabaseEdgeCaseTest {

    private DoseCalculator calculator;
    private DrugInteractionChecker interactionChecker;

    @BeforeEach
    void setUp() {
        calculator = new DoseCalculator();
        interactionChecker = new DrugInteractionChecker();
    }

    private static EnrichedPatientContext toEnrichedContext(PatientContext pc) {
        PatientState patientState = new PatientState("TEST-PATIENT-001");
        patientState.setAge(pc.getAge());
        patientState.setSex(pc.getSex());
        patientState.setWeight(pc.getWeight());
        if (pc.getCreatinine() != null) {
            patientState.setCreatinine(pc.getCreatinine());
        }
        if (pc.getAllergies() != null) {
            patientState.setAllergies(pc.getAllergies());
        }
        // Note: PatientContext.getActiveMedications() returns Map<String, Object>
        // but PatientState.setActiveMedications() expects Map<String, Medication>
        // Skip copying activeMedications to avoid type mismatch - tests don't need it

        EnrichedPatientContext enriched = new EnrichedPatientContext("TEST-PATIENT-001", patientState);
        enriched.setEventType("TEST_EVENT");
        return enriched;
    }

    @Test
    @DisplayName("Extreme renal failure (CrCl <5): Many medications contraindicated")
    void testExtremeRenalFailure() {
        // Arrange
        EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createPatientWithCrCl(3.0));

        List<String> renalDependentMeds = Arrays.asList(
            "Metformin", "Gabapentin", "Digoxin", "Enoxaparin"
        );

        // Act & Assert
        for (String medName : renalDependentMeds) {
            Medication med = MedicationTestData.createBasicMedication(medName);

            // Set renal contraindication - create adjustment for <5 mL/min
            Map<String, Medication.RenalDosing.DoseAdjustment> renalAdjustments = new HashMap<>();
            renalAdjustments.put("<5", Medication.RenalDosing.DoseAdjustment.builder()
                .crClRange("<5 mL/min")
                .adjustedDose("0mg")
                .adjustedFrequency("N/A")
                .rationale("severe renal impairment")
                .contraindicated(true)
                .build());

            Medication.AdultDosing adultDosing = med.getAdultDosing();
            if (adultDosing == null) {
                adultDosing = Medication.AdultDosing.builder().build();
            }
            adultDosing.setRenalAdjustment(Medication.RenalDosing.builder()
                .creatinineClearanceMethod("Cockcroft-Gault")
                .adjustments(renalAdjustments)
                .requiresDialysisAdjustment(false)
                .build());
            med.setAdultDosing(adultDosing);

            CalculatedDose dose = calculator.calculateDose(med, patient, "test-indication");

            assertThat(dose.isContraindicated()).isTrue();
            assertThat(dose.getContraindicationReason())
                .containsAnyOf("severe renal impairment", "CrCl", "renal");
        }
    }

    @Test
    @DisplayName("Premature neonate (<1kg, <28 weeks): Special dosing required")
    void testPrematureNeonate() {
        // Arrange
        EnrichedPatientContext neonate = toEnrichedContext(PatientContextFactory.createNeonatePatient(0.75, 0.08)); // 750g, 3 days old

        // Act
        Medication ampicillin = MedicationTestData.createBasicMedication("Ampicillin");
        ampicillin.setNeonatalDosing(new Medication.NeonatalDosing("50mg/kg", "q12h"));

        CalculatedDose dose = calculator.calculateDose(ampicillin, neonate, "test-indication");

        // Assert
        assertThat(dose.getWarnings())
            .anyMatch(w -> w.contains("Premature") || w.contains("neonate") ||
                          w.contains("NICU") || w.contains("pharmacist") ||
                          w.contains("renal") || w.contains("Immature"));
        assertThat(dose.getCalculatedFrequency()).containsAnyOf("q12h", "q24h"); // Extended interval
    }

    @Test
    @DisplayName("Morbid obesity (BMI >50, weight >200kg): Adjusted body weight required")
    void testMorbidObesity() {
        // Arrange
        EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createObesePatient(220.0, 1.60)); // BMI 86

        // Act
        Medication vancomycin = MedicationTestData.createVancomycin();
        CalculatedDose dose = calculator.calculateDose(vancomycin, patient, "test-indication");

        // Assert
        assertThat(dose.getWeightUsed()).isLessThan(220.0); // ABW used
        assertThat(dose.getWarnings())
            .anyMatch(w -> w.contains("obesity") || w.contains("Obesity") ||
                          w.contains("Adjusted") || w.contains("weight"));
        String notes = dose.getCalculationNotes();
        assertThat(notes).isNotNull();
        assertThat(notes.contains("ABW") || notes.contains("IBW") ||
                   notes.contains("adjusted") || notes.contains("weight")).isTrue();
    }

    @Test
    @DisplayName("Centenarian (age >100): Age-related adjustments")
    void testCentenarian() {
        // Arrange
        EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createGeriatricPatient(50.0, 1.8, 105, "F"));

        // Act
        Medication metoprolol = MedicationTestData.createMetoprolol();
        CalculatedDose dose = calculator.calculateDose(metoprolol, patient, "test-indication");

        // Assert
        assertThat(dose.getWarnings())
            .anyMatch(w -> w.contains("age") || w.contains("Age") ||
                          w.contains("geriatric") || w.contains("Geriatric") ||
                          w.contains("elderly") || w.contains("risk"));
        // Check dose was adjusted
        assertThat(dose.wasAdjusted() || dose.getAdjustmentReason() != null).isTrue();
    }

    @Test
    @DisplayName("Polypharmacy (20+ medications): Comprehensive interaction checking")
    void testPolypharmacy() {
        // Arrange
        EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createStandardAdult());
        PatientState state = (PatientState) patient.getPatientState();

        // Act
        Medication ciprofloxacin = MedicationTestData.createCiprofloxacin();
        // Note: DrugInteractionChecker needs PatientContext with List<String> active meds
        PatientContext pc = new PatientContext();
        pc.setAge(state.getAge());
        pc.setSex(state.getSex());
        pc.setActiveMedicationsFromList(Arrays.asList(
            "Warfarin", "Aspirin", "Metoprolol", "Lisinopril", "Furosemide",
            "Digoxin", "Amiodarone", "Atorvastatin", "Metformin", "Insulin",
            "Levothyroxine", "Pantoprazole", "Allopurinol", "Colchicine", "Prednisone",
            "Sertraline", "Gabapentin", "Tramadol", "Acetaminophen", "Docusate"
        ));
        List<Interaction> interactions = interactionChecker.checkInteractions(ciprofloxacin, pc);

        // Assert
        assertThat(interactions).isNotEmpty();
        assertThat(interactions).hasSizeGreaterThan(0);

        // Should detect at least one major interaction
        assertThat(interactions).anyMatch(i ->
            i.getSeverity() == InteractionSeverity.MAJOR ||
            (i.getDescription() != null && i.getDescription().toLowerCase().contains("warfarin"))
        );
    }

    @Nested
    @DisplayName("Boundary Condition Tests")
    class BoundaryConditions {

        @Test
        @DisplayName("Should handle zero CrCl gracefully")
        void testZeroCrCl() {
            // Arrange
            EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createPatientWithCrCl(0.1)); // Near zero

            // Act
            Medication med = MedicationTestData.createMetformin();
            CalculatedDose dose = calculator.calculateDose(med, patient, "test-indication");

            // Assert
            assertThat(dose.isContraindicated()).isTrue();
            assertThat(dose.getContraindicationReason()).isNotNull();
        }

        @Test
        @DisplayName("Should handle extremely high creatinine (>10)")
        void testExtremeCreatinine() {
            // Arrange
            EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createStandardAdult());
            PatientState state = (PatientState) patient.getPatientState();
            state.setCreatinine(12.0); // Extreme renal failure

            // Act
            PatientContext pc = new PatientContext();
            pc.setAge(state.getAge());
            pc.setSex(state.getSex());
            pc.setWeight(state.getWeight());
            pc.setCreatinine(state.getCreatinine());
            double crCl = calculator.calculateCrCl(pc);

            // Assert
            assertThat(crCl).isLessThan(10.0);
        }

        @Test
        @DisplayName("Should handle very low weight pediatric patient")
        void testVeryLowWeightPediatric() {
            // Arrange
            EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createPediatricPatient(5.0, 1));

            // Act
            Medication med = MedicationTestData.createBasicMedication("TestMed");
            med.setPediatricDosing(new Medication.PediatricDosing("10mg/kg", "q8h", 0, 5));
            CalculatedDose dose = calculator.calculateDose(med, patient, "test-indication");

            // Assert
            assertThat(dose.getCalculatedDose()).contains("50");
            assertThat(dose.getWarnings()).anyMatch(w -> w.contains("Pediatric") || w.contains("pediatric"));
        }
    }

    @Nested
    @DisplayName("Complex Multi-Factor Scenarios")
    class ComplexScenarios {

        @Test
        @DisplayName("Should handle patient with extreme age, renal failure, and polypharmacy")
        void testTripleThreat() {
            // Arrange
            EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createGeriatricPatient(45.0, 3.5, 95, "F"));
            PatientState state = (PatientState) patient.getPatientState();

            // Act
            Medication cipro = MedicationTestData.createCiprofloxacin();
            CalculatedDose dose = calculator.calculateDose(cipro, patient, "test-indication");
            PatientContext pc = new PatientContext();
            pc.setAge(state.getAge());
            pc.setSex(state.getSex());
            pc.setActiveMedicationsFromList(Arrays.asList(
                "Warfarin", "Digoxin", "Furosemide", "Metoprolol", "Lisinopril"
            ));
            List<Interaction> interactions = interactionChecker.checkInteractions(cipro, pc);

            // Assert
            assertThat(dose.getWarnings()).hasSizeGreaterThanOrEqualTo(1);
            assertThat(interactions).isNotEmpty();
            // Check that dose has warnings or was adjusted - indicates clinical review needed
            assertThat(dose.hasWarnings() || dose.wasAdjusted()).isTrue();
        }

        @Test
        @DisplayName("Should handle patient with multiple allergies and contraindications")
        void testMultipleAllergiesAndContraindications() {
            // Arrange
            EnrichedPatientContext patient = toEnrichedContext(PatientContextFactory.createStandardAdult());
            PatientState state = (PatientState) patient.getPatientState();
            state.setAllergies(Arrays.asList("penicillin", "sulfa", "codeine"));
            state.setCreatinine(2.8);

            // Act
            Medication bactrim = MedicationTestData.createBasicMedication("Trimethoprim-Sulfamethoxazole");
            bactrim.setDrugClass("Sulfonamide antibiotic");

            CalculatedDose dose = calculator.calculateDose(bactrim, patient, "test-indication");

            // Assert
            assertThat(dose.getWarnings()).hasSizeGreaterThanOrEqualTo(1);
            // Check that dose requires attention - has warnings or is adjusted (due to renal impairment and allergies)
            assertThat(dose.hasWarnings() || dose.wasAdjusted()).isTrue();
        }
    }
}
