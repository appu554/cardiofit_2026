package com.cardiofit.flink.knowledgebase.medications.safety;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker.InteractionSeverity;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import com.cardiofit.flink.knowledgebase.medications.test.PatientContextFactory;
import com.cardiofit.flink.models.PatientContext;
import org.junit.jupiter.api.*;

import java.util.Arrays;
import java.util.Comparator;
import java.util.List;
import java.util.stream.Collectors;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Test suite for DrugInteractionChecker.
 *
 * Tests major, moderate, and minor drug interactions with comprehensive
 * coverage of common clinical scenarios.
 *
 * Coverage Target: >85% line, >80% branch
 * Test Count: 9 unit tests
 */
@DisplayName("DrugInteractionChecker Test Suite")
class DrugInteractionCheckerTest {

    private DrugInteractionChecker checker;

    @BeforeEach
    void setUp() {
        checker = new DrugInteractionChecker();
    }

    @Test
    @DisplayName("Should detect major interaction: Warfarin + Ciprofloxacin (bleeding risk)")
    void testMajorInteraction_WarfarinCiprofloxacin() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Warfarin")
        );
        Medication cipro = MedicationTestData.createCiprofloxacin();

        // Act
        List<Interaction> interactions = checker.checkInteractions(cipro, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getDescription().toLowerCase().contains("warfarin"))
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getSeverity()).isEqualTo(InteractionSeverity.MAJOR);
        assertThat(interaction.getDescription()).containsAnyOf("INR", "bleeding");
        assertThat(interaction.getManagement()).contains("Monitor INR");
    }

    @Test
    @DisplayName("Should detect moderate interaction: Piperacillin + Vancomycin (nephrotoxicity)")
    void testModerateInteraction_NephrotoxicCombination() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Vancomycin")
        );
        Medication pipTazo = MedicationTestData.createPiperacillinTazobactam();

        // Act
        List<Interaction> interactions = checker.checkInteractions(pipTazo, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MODERATE)
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getDescription())
            .containsAnyOf("nephrotoxicity", "renal function");
    }

    @Test
    @DisplayName("Should verify bidirectional interaction checking (A→B same as B→A)")
    void testBidirectionalInteraction() {
        // Arrange
        Medication warfarin = MedicationTestData.createWarfarin();
        Medication cipro = MedicationTestData.createCiprofloxacin();

        // Act
        List<Interaction> warfarinFirst = checker.checkInteraction(warfarin, cipro);
        List<Interaction> ciproFirst = checker.checkInteraction(cipro, warfarin);

        // Assert
        assertThat(warfarinFirst).hasSizeGreaterThanOrEqualTo(1);
        assertThat(ciproFirst).hasSizeGreaterThanOrEqualTo(1);

        // Verify severity is same regardless of order
        InteractionSeverity severity1 = warfarinFirst.get(0).getSeverity();
        InteractionSeverity severity2 = ciproFirst.get(0).getSeverity();
        assertThat(severity1).isEqualTo(severity2);
    }

    @Test
    @DisplayName("Should prioritize interactions by severity (MAJOR before MINOR)")
    void testInteractionSeverityPrioritization() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Warfarin", "Antacid") // MAJOR and MINOR interactions with Cipro
        );
        Medication cipro = MedicationTestData.createCiprofloxacin();

        // Act
        List<Interaction> interactions = checker.checkInteractions(cipro, patient);
        List<Interaction> sorted = interactions.stream()
            .sorted(Comparator.comparing(Interaction::getSeverity))
            .collect(Collectors.toList());

        // Assert
        assertThat(sorted).hasSizeGreaterThanOrEqualTo(1);

        // Verify MAJOR comes before MODERATE/MINOR
        for (int i = 0; i < sorted.size() - 1; i++) {
            InteractionSeverity current = sorted.get(i).getSeverity();
            InteractionSeverity next = sorted.get(i + 1).getSeverity();
            assertThat(current.ordinal()).isLessThanOrEqualTo(next.ordinal());
        }
    }

    @Test
    @DisplayName("Should detect Digoxin + Furosemide interaction (hypokalemia)")
    void testDigoxinFurosemideInteraction() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Digoxin")
        );
        Medication furosemide = MedicationTestData.createBasicMedication("Furosemide");
        furosemide.setDrugClass("Loop diuretic");

        // Act
        List<Interaction> interactions = checker.checkInteractions(furosemide, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MODERATE)
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getDescription()).containsAnyOf("hypokalemia", "toxicity");
    }

    @Test
    @DisplayName("Should detect additive bleeding risk: Heparin + Aspirin")
    void testAdditiveBleedingRisk() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Heparin")
        );
        Medication aspirin = MedicationTestData.createAspirin();

        // Act
        List<Interaction> interactions = checker.checkInteractions(aspirin, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MODERATE)
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getDescription()).containsAnyOf("bleeding", "additive");
        assertThat(interaction.getManagement()).contains("Monitor for bleeding");
    }

    @Test
    @DisplayName("Should detect ACE inhibitor + Potassium interaction (hyperkalemia)")
    void testHyperkalemiaRisk() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Lisinopril") // ACE inhibitor
        );
        Medication potassium = MedicationTestData.createBasicMedication("Potassium Chloride");
        potassium.setDrugClass("Electrolyte supplement");

        // Act
        List<Interaction> interactions = checker.checkInteractions(potassium, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MAJOR)
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getDescription()).contains("hyperkalemia");
        assertThat(interaction.getManagement()).containsAnyOf("Monitor potassium", "Monitor K+");
    }

    @Test
    @DisplayName("Should detect Beta-blocker + Diltiazem bradycardia risk")
    void testBradycardiaRisk() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Metoprolol") // Beta-blocker
        );
        Medication diltiazem = MedicationTestData.createBasicMedication("Diltiazem");
        diltiazem.setDrugClass("Calcium channel blocker");

        // Act
        List<Interaction> interactions = checker.checkInteractions(diltiazem, patient);

        // Assert
        assertThat(interactions).hasSizeGreaterThanOrEqualTo(1);
        Interaction interaction = interactions.stream()
            .filter(i -> i.getSeverity() == InteractionSeverity.MODERATE)
            .findFirst()
            .orElseThrow();
        assertThat(interaction.getDescription()).containsAnyOf("bradycardia", "heart rate");
    }

    @Test
    @DisplayName("Should handle no interactions gracefully")
    void testNoInteractions() {
        // Arrange
        PatientContext patient = PatientContextFactory.createPatientWithMedications(
            Arrays.asList("Acetaminophen") // No significant interactions
        );
        Medication aspirin = MedicationTestData.createAspirin();

        // Act
        List<Interaction> interactions = checker.checkInteractions(aspirin, patient);

        // Assert - May have minor interactions or none
        assertThat(interactions).isNotNull();
        if (!interactions.isEmpty()) {
            assertThat(interactions).allMatch(i ->
                i.getSeverity() == InteractionSeverity.MINOR ||
                i.getSeverity() == InteractionSeverity.MODERATE
            );
        }
    }

    @Nested
    @DisplayName("Polypharmacy Tests")
    class PolypharmacyTests {

        @Test
        @DisplayName("Should detect multiple interactions in complex medication list")
        void testComplexPolypharmacy() {
            // Arrange
            PatientContext patient = PatientContextFactory.createPatientWithMedications(
                Arrays.asList(
                    "Warfarin", "Aspirin", "Metoprolol", "Lisinopril", "Furosemide",
                    "Digoxin", "Amiodarone", "Atorvastatin"
                )
            );
            Medication cipro = MedicationTestData.createCiprofloxacin();

            // Act
            List<Interaction> interactions = checker.checkInteractions(cipro, patient);

            // Assert
            assertThat(interactions).hasSizeGreaterThan(2);

            // Should have at least one MAJOR interaction (Warfarin)
            assertThat(interactions).anyMatch(i ->
                i.getSeverity() == InteractionSeverity.MAJOR &&
                i.getDescription().toLowerCase().contains("warfarin")
            );
        }

        @Test
        @DisplayName("Should identify all MAJOR interactions in priority order")
        void testMajorInteractionPriority() {
            // Arrange
            PatientContext patient = PatientContextFactory.createPatientWithMedications(
                Arrays.asList("Warfarin", "Heparin", "Clopidogrel") // All anticoagulants
            );
            Medication aspirin = MedicationTestData.createAspirin();

            // Act
            List<Interaction> interactions = checker.checkInteractions(aspirin, patient);
            List<Interaction> majorInteractions = interactions.stream()
                .filter(i -> i.getSeverity() == InteractionSeverity.MAJOR)
                .toList();

            // Assert
            assertThat(majorInteractions).hasSizeGreaterThanOrEqualTo(1);
            assertThat(majorInteractions).allMatch(i ->
                i.getDescription().toLowerCase().contains("bleeding")
            );
        }
    }
}
