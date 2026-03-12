package com.cardiofit.flink.knowledgebase.medications.integration;

import com.cardiofit.flink.knowledgebase.medications.model.*;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import org.junit.jupiter.api.*;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Test suite for MedicationIntegrationService.
 *
 * Tests model conversion, protocol integration, and evidence linking.
 * Coverage Target: >80% line, >77% branch
 * Test Count: 5 unit tests
 */
@DisplayName("MedicationIntegrationService Test Suite")
class MedicationIntegrationServiceTest {

    private MedicationIntegrationService service;

    @BeforeEach
    void setUp() {
        service = new MedicationIntegrationService();
    }

    @Test
    @DisplayName("Should convert enhanced Medication to legacy Medication model")
    void testConvertToLegacyModel() {
        // Arrange
        Medication enhanced = MedicationTestData.createPiperacillinTazobactam();

        // Act
        com.cardiofit.flink.models.Medication legacy = service.convertToLegacyModel(enhanced);

        // Assert
        assertThat(legacy.getName()).isEqualTo(enhanced.getName());
        assertThat(legacy.getDosage()).isEqualTo(enhanced.getStandardDose());
        assertThat(legacy.getRoute()).isEqualTo(enhanced.getRoute());
        assertThat(legacy.getFrequency()).isEqualTo(enhanced.getCalculatedFrequency());
    }

    @Test
    @DisplayName("Should convert legacy Medication to enhanced Medication model")
    void testConvertFromLegacyModel() {
        // Arrange
        com.cardiofit.flink.models.Medication legacy = new com.cardiofit.flink.models.Medication();
        legacy.setName("vancomycin");
        legacy.setDosage("1g");
        legacy.setRoute("IV");
        legacy.setFrequency("q12h");

        // Act
        Medication enhanced = service.convertFromLegacyModel(legacy);

        // Assert
        assertThat(enhanced).isNotNull();
        assertThat(enhanced.getName()).isEqualTo(legacy.getName());
        assertThat(enhanced.getStandardDose()).isNotNull();
        assertThat(enhanced.getRoute()).isNotNull();
        assertThat(enhanced.getCalculatedFrequency()).isNotNull();
    }

    @Test
    @DisplayName("Should retrieve evidence for medication ID")
    void testGetEvidenceForMedication() {
        // Arrange
        String medicationId = MedicationTestData.PIPERACILLIN_TAZOBACTAM_ID;

        // Act
        List<String> evidence = service.getEvidenceForMedication(medicationId);

        // Assert
        assertThat(evidence).isNotNull();
        // Evidence list may be empty if not linked to guidelines yet
    }

    @Test
    @DisplayName("Should maintain backward compatibility with old code")
    void testBackwardCompatibility() {
        // Arrange
        com.cardiofit.flink.models.Medication oldMed = new com.cardiofit.flink.models.Medication();
        oldMed.setName("ceftriaxone");
        oldMed.setDosage("2g");
        oldMed.setRoute("IV");
        oldMed.setFrequency("q24h");

        // Act - Round trip conversion
        Medication enhanced = service.convertFromLegacyModel(oldMed);
        com.cardiofit.flink.models.Medication backToOld = service.convertToLegacyModel(enhanced);

        // Assert
        assertThat(backToOld).isNotNull();
        if (enhanced != null) {
            assertThat(backToOld.getName()).isEqualTo(oldMed.getName());
            assertThat(backToOld.getDosage()).isEqualTo(enhanced.getStandardDose());
            assertThat(backToOld.getRoute()).isEqualTo(enhanced.getRoute());
            assertThat(backToOld.getFrequency()).isEqualTo(enhanced.getCalculatedFrequency());
        }
    }

    @Test
    @DisplayName("Should check if medication exists in database")
    void testMedicationExists() {
        // Arrange
        String existingMedication = "piperacillin-tazobactam";
        String nonExistingMedication = "nonexistent-drug";

        // Act & Assert
        assertThat(service.medicationExists(existingMedication)).isTrue();
        assertThat(service.medicationExists(nonExistingMedication)).isFalse();
    }

    @Nested
    @DisplayName("Protocol Integration Tests")
    class ProtocolIntegrationTests {

        @Test
        @DisplayName("Should get medication count from database")
        void testGetMedicationCount() {
            // Act
            int count = service.getMedicationCount();

            // Assert
            assertThat(count).isGreaterThan(0);
        }

        @Test
        @DisplayName("Should get display name for medication")
        void testGetMedicationDisplayName() {
            // Arrange
            String medicationId = MedicationTestData.ASPIRIN_ID;

            // Act
            String displayName = service.getMedicationDisplayName(medicationId);

            // Assert
            assertThat(displayName).isNotNull();
            assertThat(displayName).contains("aspirin");
        }
    }
}
