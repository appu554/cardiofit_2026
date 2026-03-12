package com.cardiofit.flink.knowledgebase.medications;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import org.junit.jupiter.api.*;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Basic test suite for Medication model class.
 *
 * Tests basic model functionality, getters/setters, and validation.
 * Coverage Target: >90% line
 * Test Count: 3 unit tests
 */
@DisplayName("Medication Model Test Suite")
class MedicationTest {

    @Test
    @DisplayName("Should create medication with all required fields")
    void testMedicationCreation() {
        // Arrange & Act
        Medication med = MedicationTestData.createCeftriaxone();

        // Assert
        assertThat(med.getMedicationId()).isEqualTo(MedicationTestData.CEFTRIAXONE_ID);
        assertThat(med.getName()).isEqualTo("Ceftriaxone");
        assertThat(med.getGenericName()).isEqualTo("ceftriaxone");
        assertThat(med.getDrugClass()).isEqualTo("Cephalosporin");
        assertThat(med.getStandardDose()).isEqualTo("2g");
        assertThat(med.getRoute()).isEqualTo("IV");
        assertThat(med.getCalculatedFrequency()).isEqualTo("q24h");
    }

    @Test
    @DisplayName("Should correctly identify high-alert medications")
    void testHighAlertIdentification() {
        // Arrange
        Medication heparin = MedicationTestData.createHeparin();
        Medication aspirin = MedicationTestData.createAspirin();

        // Act & Assert
        assertThat(heparin.isHighAlert()).isTrue();
        assertThat(aspirin.isHighAlert()).isFalse();
    }

    @Test
    @DisplayName("Should correctly identify formulary status")
    void testFormularyStatus() {
        // Arrange
        Medication formulary = MedicationTestData.createCeftriaxone();
        Medication nonFormulary = MedicationTestData.createNonFormularyMedication("TestMed");

        // Act & Assert
        assertThat(formulary.isFormulary()).isTrue();
        assertThat(nonFormulary.isFormulary()).isFalse();
    }
}
