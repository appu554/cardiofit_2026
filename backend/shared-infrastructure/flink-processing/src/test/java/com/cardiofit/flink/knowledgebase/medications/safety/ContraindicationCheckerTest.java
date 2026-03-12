package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.safety.EnhancedContraindicationChecker.ContraindicationResult;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import com.cardiofit.flink.knowledgebase.medications.test.PatientContextFactory;
import com.cardiofit.flink.models.PatientContext;
import org.junit.jupiter.api.*;

import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Test suite for EnhancedContraindicationChecker.
 *
 * Tests absolute and relative contraindications, disease state contraindications,
 * and black box warnings.
 *
 * Coverage Target: >85% line, >78% branch
 * Test Count: 7 unit tests
 */
@DisplayName("ContraindicationChecker Test Suite")
class ContraindicationCheckerTest {

    private EnhancedContraindicationChecker checker;

    @BeforeEach
    void setUp() {
        checker = new EnhancedContraindicationChecker();
    }

    @Test
    @DisplayName("Should reject penicillin for patient with penicillin allergy (absolute contraindication)")
    void testAbsoluteContraindication_PenicillinAllergy() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithAllergies(
            Arrays.asList("penicillin")
        );
        Medication pipTazo = MedicationTestData.createPiperacillinTazobactam();

        // Act
        ContraindicationResult result = checker.check(pipTazo, patient);

        // Assert
        assertThat(result.isContraindicated()).isTrue();
        assertThat(result.getContraindicationType()).isEqualTo(ContraindicationType.ABSOLUTE);
        assertThat(result.getReason()).containsAnyOf("allergy", "penicillin");
        assertThat(result.getShouldReject()).isTrue();
    }

    @Test
    @DisplayName("Should warn for metformin in renal impairment (relative contraindication)")
    void testRelativeContraindication_MetforminRenalImpairment() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithCrCl(25.0); // Severe renal impairment
        Medication metformin = MedicationTestData.createMetformin();

        // Act
        ContraindicationResult result = checker.check(metformin, patient);

        // Assert
        assertThat(result.isContraindicated()).isFalse(); // Relative, not absolute
        assertThat(result.getContraindicationType()).isEqualTo(ContraindicationType.RELATIVE);
        assertThat(result.getWarnings()).containsAnyOf("renal impairment", "lactic acidosis risk");
        assertThat(result.requiresClinicalReview()).isTrue();
    }

    @Test
    @DisplayName("Should flag NSAIDs for heart failure patients (disease state contraindication)")
    void testDiseaseStateContraindication_NSAIDsHeartFailure() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithDiagnoses(
            Arrays.asList("Heart Failure NYHA Class III")
        );
        Medication ibuprofen = MedicationTestData.createBasicMedication("Ibuprofen");
        ibuprofen.setDrugClass("NSAID");

        // Act
        ContraindicationResult result = checker.check(ibuprofen, patient);

        // Assert
        assertThat(result.getWarnings())
            .containsAnyOf("heart failure", "fluid retention", "worsening HF");
        assertThat(result.getRecommendation())
            .contains("Consider alternative analgesic");
    }

    @Test
    @DisplayName("Should display black box warning for warfarin")
    void testBlackBoxWarningCheck() {
        // Arrange
        PatientContext patient = PatientContextFactory.createStandardAdult();
        Medication warfarin = MedicationTestData.createWarfarin();

        // Act
        ContraindicationResult result = checker.check(warfarin, patient);

        // Assert
        assertThat(result.hasBlackBoxWarning()).isTrue();
        assertThat(result.getBlackBoxWarning())
            .containsAnyOf("Bleeding risk", "INR monitoring required");
    }

    @Test
    @DisplayName("Should detect ciprofloxacin contraindication in severe hepatic impairment")
    void testCiproHepaticFailure() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithChildPugh("C");
        Medication cipro = MedicationTestData.createCiprofloxacin();

        // Act
        ContraindicationResult result = checker.check(cipro, patient);

        // Assert
        assertThat(result.getContraindicationType()).isIn(
            ContraindicationType.RELATIVE,
            ContraindicationType.ABSOLUTE
        );
        assertThat(result.getWarnings()).containsAnyOf(
            "Severe hepatic impairment",
            "Liver dysfunction"
        );
    }

    @Test
    @DisplayName("Should flag pregnancy category X medications")
    void testPregnancyCategoryX() {
        // Arrange
        PatientContext patient = PatientContextFactory.createStandardAdult();
        patient.setSex("F");
        patient.setAge(28);
        patient.setPregnant(true);

        Medication isotretinoin = MedicationTestData.createBasicMedication("Isotretinoin");
        isotretinoin.setPregnancyCategory("X");

        // Act
        ContraindicationResult result = checker.check(isotretinoin, patient);

        // Assert
        assertThat(result.isContraindicated()).isTrue();
        assertThat(result.getContraindicationType()).isEqualTo(ContraindicationType.ABSOLUTE);
        assertThat(result.getReason()).contains("Pregnancy Category X");
        assertThat(result.getShouldReject()).isTrue();
    }

    @Test
    @DisplayName("Should warn for tetracycline in pediatric patients (age <18)")
    void testPediatricContraindication() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPediatricPatient(30.0, 12);
        Medication tetracycline = MedicationTestData.createBasicMedication("Tetracycline");
        tetracycline.setDrugClass("Tetracycline antibiotic");

        // Act
        ContraindicationResult result = checker.check(tetracycline, patient);

        // Assert
        assertThat(result.isContraindicated()).isTrue();
        assertThat(result.getContraindicationType()).isEqualTo(ContraindicationType.ABSOLUTE);
        assertThat(result.getReason()).containsAnyOf(
            "Age <18",
            "tooth discoloration",
            "bone development"
        );
    }

    @Nested
    @DisplayName("Complex Contraindication Scenarios")
    class ComplexScenarios {

        @Test
        @DisplayName("Should detect multiple contraindications in complex patient")
        void testMultipleContraindications() {
            // Arrange
            PatientContext patient = PatientContextFactory.createComplexPatient();
            // Patient has: Heart failure, CKD stage 4, penicillin allergy

            Medication ibuprofen = MedicationTestData.createBasicMedication("Ibuprofen");
            ibuprofen.setDrugClass("NSAID");

            // Act
            ContraindicationResult result = checker.check(ibuprofen, patient);

            // Assert
            assertThat(result.getWarnings()).hasSizeGreaterThanOrEqualTo(2);
            assertThat(result.getWarnings()).anyMatch(w ->
                w.toLowerCase().contains("heart failure")
            );
            assertThat(result.getWarnings()).anyMatch(w ->
                w.toLowerCase().contains("renal")
            );
        }

        @Test
        @DisplayName("Should prioritize absolute over relative contraindications")
        void testContraindicationPriority() {
            // Arrange
            PatientContext patient = PatientContextFactory.createPatientWithAllergies(
                Arrays.asList("sulfa")
            );
            patient.setCreatinine(2.5); // Also has renal impairment

            Medication bactrim = MedicationTestData.createBasicMedication("Trimethoprim-Sulfamethoxazole");
            bactrim.setDrugClass("Sulfonamide antibiotic");

            // Act
            ContraindicationResult result = checker.check(bactrim, patient);

            // Assert
            assertThat(result.isContraindicated()).isTrue();
            assertThat(result.getContraindicationType()).isEqualTo(ContraindicationType.ABSOLUTE);
            assertThat(result.getReason()).contains("allergy");
        }
    }
}
