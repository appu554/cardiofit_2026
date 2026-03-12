package com.cardiofit.flink.knowledgebase.medications.substitution;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import com.cardiofit.flink.knowledgebase.medications.test.PatientContextFactory;
import com.cardiofit.flink.models.PatientContext;
import org.junit.jupiter.api.*;

import java.util.Arrays;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Test suite for TherapeuticSubstitutionEngine.
 *
 * Tests formulary substitution, cost optimization, same-class and different-class
 * substitutions, and efficacy-based ranking.
 *
 * Coverage Target: >80% line, >75% branch
 * Test Count: 7 unit tests
 */
@DisplayName("TherapeuticSubstitutionEngine Test Suite")
class TherapeuticSubstitutionEngineTest {

    private TherapeuticSubstitutionEngine engine;

    @BeforeEach
    void setUp() {
        engine = new TherapeuticSubstitutionEngine();
    }

    @Test
    @DisplayName("Should suggest formulary substitute: Non-formulary → Formulary equivalent")
    void testFormularySubstitution() {
        // Arrange
        Medication nonFormulary = MedicationTestData.createNonFormularyMedication("Cefepime");
        nonFormulary.setDrugClass("Cephalosporin");

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(nonFormulary);

        // Assert
        assertThat(substitutes).isNotEmpty();
        SubstitutionOption formularyOption = substitutes.stream()
            .filter(s -> s.getMedication().isFormulary())
            .findFirst()
            .orElseThrow();
        assertThat(formularyOption.getMedication().getName()).isEqualTo("Ceftriaxone");
        assertThat(formularyOption.getReason()).contains("formulary preferred");
    }

    @Test
    @DisplayName("Should suggest cost optimization: Brand → Generic substitution")
    void testCostOptimization() {
        // Arrange
        Medication brand = MedicationTestData.createBrandMedication("Zosyn", 450.00);
        brand.setGenericName("piperacillin-tazobactam");

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(brand);

        // Assert
        assertThat(substitutes).isNotEmpty();
        SubstitutionOption generic = substitutes.stream()
            .filter(s -> s.getSubstitutionType() == SubstitutionType.GENERIC_EQUIVALENT)
            .findFirst()
            .orElseThrow();
        assertThat(generic.getMedication().getName()).contains("Piperacillin-Tazobactam");
        assertThat(generic.getCostSavings()).isGreaterThan(400.00);
        assertThat(generic.getReason()).containsAnyOf("generic equivalent", "cost savings");
    }

    @Test
    @DisplayName("Should provide same-class substitute: Cefepime → Ceftriaxone")
    void testSameClassSubstitution() {
        // Arrange
        Medication cefepime = MedicationTestData.createBasicMedication("Cefepime");
        cefepime.setDrugClass("Cephalosporin");
        cefepime.setCategory("Antibiotic");

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(cefepime);

        // Assert
        assertThat(substitutes).isNotEmpty();
        SubstitutionOption sameClass = substitutes.stream()
            .filter(s -> s.getSubstitutionType() == SubstitutionType.SAME_CLASS)
            .findFirst()
            .orElseThrow();
        assertThat(sameClass.getMedication().getDrugClass()).isEqualTo("Cephalosporin");
    }

    @Test
    @DisplayName("Should provide different-class substitute for allergy: Penicillin → Fluoroquinolone")
    void testDifferentClassSubstitution() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication penicillin = MedicationTestData.createBasicMedication("Penicillin");
        penicillin.setDrugClass("Beta-lactam");

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(penicillin, patient);

        // Assert
        assertThat(substitutes).isNotEmpty();
        SubstitutionOption differentClass = substitutes.stream()
            .filter(s -> s.getSubstitutionType() == SubstitutionType.DIFFERENT_CLASS)
            .findFirst()
            .orElseThrow();
        assertThat(differentClass.getMedication().getDrugClass()).isEqualTo("Fluoroquinolone");
        assertThat(differentClass.getReason()).containsAnyOf("allergy", "different class");
    }

    @Test
    @DisplayName("Should rank alternatives by efficacy for specific indication")
    void testEfficacyComparison() {
        // Arrange
        Medication vancomycin = MedicationTestData.createVancomycin();
        String indication = "MRSA bacteremia";

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(vancomycin, indication);
        List<SubstitutionOption> sorted = engine.sortByEfficacy(substitutes, indication);

        // Assert
        assertThat(sorted).isNotEmpty();

        // Verify descending order by efficacy score
        for (int i = 0; i < sorted.size() - 1; i++) {
            assertThat(sorted.get(i).getEfficacyScore())
                .isGreaterThanOrEqualTo(sorted.get(i + 1).getEfficacyScore());
        }
    }

    @Test
    @DisplayName("Should provide IV to PO substitution when appropriate")
    void testIVtoPOSubstitution() {
        // Arrange
        Medication ivLevofloxacin = MedicationTestData.createLevofloxacin();
        ivLevofloxacin.setRoute("IV");
        PatientContext patient = PatientContextFactory.createStandardAdult();
        patient.setAbilityToTakePO(true);

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(ivLevofloxacin, patient);

        // Assert
        SubstitutionOption poOption = substitutes.stream()
            .filter(s -> s.getSubstitutionType() == SubstitutionType.ROUTE_CONVERSION)
            .filter(s -> s.getMedication().getRoute().equals("PO"))
            .findFirst()
            .orElseThrow();
        assertThat(poOption.getReason()).contains("oral bioavailability");
        assertThat(poOption.getCostSavings()).isGreaterThan(0);
    }

    @Test
    @DisplayName("Should not suggest substitutes when patient has allergy to alternative class")
    void testNoSubstitutionWithAllergy() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin", "fluoroquinolone")
        );
        Medication penicillin = MedicationTestData.createBasicMedication("Penicillin");
        penicillin.setDrugClass("Beta-lactam");

        // Act
        List<SubstitutionOption> substitutes = engine.findSubstitutes(penicillin, patient);

        // Assert
        // Should still provide options but exclude fluoroquinolones
        assertThat(substitutes).allMatch(s ->
            !s.getMedication().getDrugClass().equals("Fluoroquinolone")
        );
    }

    @Nested
    @DisplayName("Cost and Formulary Optimization")
    class CostOptimizationTests {

        @Test
        @DisplayName("Should calculate accurate cost savings for generic substitution")
        void testCostSavingsCalculation() {
            // Arrange
            Medication brand = MedicationTestData.createBrandMedication("Lipitor", 200.00);
            brand.setGenericName("atorvastatin");

            // Act
            List<SubstitutionOption> substitutes = engine.findSubstitutes(brand);
            SubstitutionOption generic = substitutes.stream()
                .filter(s -> s.getSubstitutionType() == SubstitutionType.GENERIC_EQUIVALENT)
                .findFirst()
                .orElseThrow();

            // Assert
            assertThat(generic.getCostSavings()).isGreaterThan(150.00);
            assertThat(generic.getCostSavingsPercentage()).isGreaterThan(75.0);
        }

        @Test
        @DisplayName("Should prioritize formulary alternatives over non-formulary")
        void testFormularyPriority() {
            // Arrange
            Medication nonFormulary = MedicationTestData.createNonFormularyMedication("Expensive Med");

            // Act
            List<SubstitutionOption> substitutes = engine.findSubstitutes(nonFormulary);
            List<SubstitutionOption> sorted = engine.sortByPreference(substitutes);

            // Assert
            // First option should be formulary
            assertThat(sorted.get(0).getMedication().isFormulary()).isTrue();
        }
    }
}
