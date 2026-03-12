package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.safety.AllergyChecker.AllergyCheckResult;
import com.cardiofit.flink.knowledgebase.medications.safety.AllergyChecker.AllergyType;
import com.cardiofit.flink.knowledgebase.medications.safety.AllergyChecker.RiskLevel;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import com.cardiofit.flink.knowledgebase.medications.test.PatientContextFactory;
import com.cardiofit.flink.models.PatientContext;
import org.junit.jupiter.api.*;

import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.within;

/**
 * Test suite for AllergyChecker.
 *
 * Tests direct allergies and cross-reactivity patterns.
 * Coverage Target: >85% line, >82% branch
 * Test Count: 7 unit tests
 */
@DisplayName("AllergyChecker Test Suite")
class AllergyCheckerTest {

    private AllergyChecker checker;

    @BeforeEach
    void setUp() {
        checker = new AllergyChecker();
    }

    @Test
    @DisplayName("Should detect direct allergy: Penicillin allergy → Amoxicillin")
    void testDirectAllergy() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication amoxicillin = MedicationTestData.createBasicMedication("Amoxicillin");
        amoxicillin.setDrugClass("Beta-lactam");

        // Act
        AllergyCheckResult result = checker.check(amoxicillin, patient);

        // Assert
        assertThat(result.hasAllergy()).isTrue();
        assertThat(result.getAllergyType()).isEqualTo(AllergyType.DIRECT);
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.HIGH);
        assertThat(result.getShouldReject()).isTrue();
    }

    @Test
    @DisplayName("Should detect 10% cross-reactivity: Penicillin ↔ Cephalosporin")
    void testCrossReactivityPenicillinCephalosporin() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication ceftriaxone = MedicationTestData.createCeftriaxone();

        // Act
        AllergyCheckResult result = checker.check(ceftriaxone, patient);

        // Assert
        assertThat(result.hasCrossReactivity()).isTrue();
        assertThat(result.getCrossReactivityPercentage()).isCloseTo(10.0, within(1.0));
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.MODERATE);
        assertThat(result.getWarnings())
            .containsAnyOf("10% cross-reactivity", "penicillin", "cephalosporin");
        assertThat(result.requiresClinicalReview()).isTrue();
    }

    @Test
    @DisplayName("Should assess low-risk cross-reactivity: Sulfa antibiotic ↔ Sulfonylurea")
    void testCrossReactivitySulfa() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("sulfamethoxazole")
        );
        Medication glyburide = MedicationTestData.createBasicMedication("Glyburide");
        glyburide.setDrugClass("Sulfonylurea");

        // Act
        AllergyCheckResult result = checker.check(glyburide, patient);

        // Assert
        assertThat(result.hasCrossReactivity()).isTrue();
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.LOW);
        assertThat(result.getShouldReject()).isFalse();
        assertThat(result.getRecommendation())
            .containsAnyOf("Monitor for allergic reaction", "Low cross-reactivity risk");
    }

    @Test
    @DisplayName("Should detect high-risk NSAID cross-reactivity: Aspirin ↔ Ibuprofen")
    void testCrossReactivityNSAIDs() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("aspirin")
        );
        Medication ibuprofen = MedicationTestData.createBasicMedication("Ibuprofen");
        ibuprofen.setDrugClass("NSAID");

        // Act
        AllergyCheckResult result = checker.check(ibuprofen, patient);

        // Assert
        assertThat(result.hasCrossReactivity()).isTrue();
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.HIGH);
        assertThat(result.getCrossReactivityPercentage()).isGreaterThan(80.0);
    }

    @Test
    @DisplayName("Should detect low cross-reactivity: Penicillin → Carbapenem (1-2%)")
    void testCrossReactivityPenicillinCarbapenem() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication meropenem = MedicationTestData.createBasicMedication("Meropenem");
        meropenem.setDrugClass("Carbapenem");

        // Act
        AllergyCheckResult result = checker.check(meropenem, patient);

        // Assert
        assertThat(result.hasCrossReactivity()).isTrue();
        assertThat(result.getCrossReactivityPercentage()).isCloseTo(1.5, within(1.0));
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.LOW);
        assertThat(result.getShouldReject()).isFalse();
    }

    @Test
    @DisplayName("Should allow medications with no cross-reactivity")
    void testNoAllergy() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication cipro = MedicationTestData.createCiprofloxacin();

        // Act
        AllergyCheckResult result = checker.check(cipro, patient);

        // Assert
        assertThat(result.hasAllergy()).isFalse();
        assertThat(result.hasCrossReactivity()).isFalse();
        assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.NONE);
        assertThat(result.getShouldReject()).isFalse();
    }

    @Test
    @DisplayName("Should handle multiple allergies")
    void testMultipleAllergies() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin", "sulfa", "codeine")
        );
        Medication ceftriaxone = MedicationTestData.createCeftriaxone();

        // Act
        AllergyCheckResult result = checker.check(ceftriaxone, patient);

        // Assert
        assertThat(result.hasCrossReactivity()).isTrue(); // Penicillin cross-reactivity
        assertThat(result.getMatchedAllergies()).contains("penicillin");
    }

    @Nested
    @DisplayName("Complex Allergy Scenarios")
    class ComplexScenarios {

        @Test
        @DisplayName("Should prioritize direct allergy over cross-reactivity")
        void testDirectAllergyPriority() {
            // Arrange
            PatientContext patient = PatientContextFactory.createPatientWithAllergies(
                Arrays.asList("amoxicillin", "penicillin")
            );
            Medication amoxicillin = MedicationTestData.createBasicMedication("Amoxicillin");
            amoxicillin.setDrugClass("Beta-lactam");

            // Act
            AllergyCheckResult result = checker.check(amoxicillin, patient);

            // Assert
            assertThat(result.getAllergyType()).isEqualTo(AllergyType.DIRECT);
            assertThat(result.getRiskLevel()).isEqualTo(RiskLevel.HIGH);
        }

        @Test
        @DisplayName("Should provide appropriate warnings for moderate risk")
        void testModerateRiskWarnings() {
            // Arrange
            PatientContext patient = PatientContextFactory.createPatientWithAllergies(
                Arrays.asList("penicillin")
            );
            Medication cefazolin = MedicationTestData.createBasicMedication("Cefazolin");
            cefazolin.setDrugClass("Cephalosporin");

            // Act
            AllergyCheckResult result = checker.check(cefazolin, patient);

            // Assert
            assertThat(result.requiresClinicalReview()).isTrue();
            assertThat(result.getRecommendation())
                .containsAnyOf("Clinical review required", "Consider alternative");
        }
    }
}
