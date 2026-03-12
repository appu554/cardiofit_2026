package com.cardiofit.flink.knowledgebase.medications.loader;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.knowledgebase.medications.test.MedicationTestData;
import org.junit.jupiter.api.*;
import org.junit.jupiter.api.io.TempDir;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.junit.jupiter.api.Assertions.assertThrows;

/**
 * Test suite for MedicationDatabaseLoader.
 *
 * Tests singleton pattern, YAML loading, caching, and error handling.
 * Coverage Target: >90% line, >85% branch
 *
 * Test Count: 8 unit tests
 */
@DisplayName("MedicationDatabaseLoader Test Suite")
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
class MedicationDatabaseLoaderTest {

    private MedicationDatabaseLoader loader;

    @TempDir
    Path tempMedicationDir;

    @BeforeEach
    void setUp() {
        // Reset singleton instance for testing isolation
        MedicationDatabaseLoader.reset();
        loader = MedicationDatabaseLoader.getInstance();
    }

    @Test
    @Order(1)
    @DisplayName("Should return same singleton instance on multiple calls")
    void testSingletonPattern() {
        // Arrange & Act
        MedicationDatabaseLoader instance1 = MedicationDatabaseLoader.getInstance();
        MedicationDatabaseLoader instance2 = MedicationDatabaseLoader.getInstance();

        // Assert
        assertThat(instance1).isSameAs(instance2);
        assertThat(instance1).isNotNull();
    }

    @Test
    @Order(2)
    @DisplayName("Should load 100 medications from YAML directory successfully")
    void testLoadAllMedications() throws Exception {
        // Arrange
        createTestMedicationYAMLs(tempMedicationDir, 100);

        // Act
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Assert
        assertThat(loader.getAllMedications()).hasSize(100);
        assertThat(loader.getMedicationCount()).isEqualTo(100);
    }

    @Test
    @Order(3)
    @DisplayName("Should retrieve medication by ID successfully")
    void testGetMedicationById() throws Exception {
        // Arrange
        createSampleMedication(tempMedicationDir, "MED-ASA-001", "Aspirin");
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Act
        Medication aspirin = loader.getMedicationById("MED-ASA-001");

        // Assert
        assertThat(aspirin).isNotNull();
        assertThat(aspirin.getMedicationId()).isEqualTo("MED-ASA-001");
        assertThat(aspirin.getName()).isEqualTo("Aspirin");
    }

    @Test
    @Order(4)
    @DisplayName("Should perform case-insensitive medication name search")
    void testGetMedicationByName() throws Exception {
        // Arrange
        createSampleMedication(tempMedicationDir, "MED-ASA-001", "Aspirin");
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Act
        Medication found1 = loader.getMedicationByName("aspirin");
        Medication found2 = loader.getMedicationByName("ASPIRIN");
        Medication found3 = loader.getMedicationByName("Aspirin");

        // Assert
        assertThat(found1).isNotNull();
        assertThat(found2).isNotNull();
        assertThat(found3).isNotNull();
        assertThat(found1.getMedicationId()).isEqualTo("MED-ASA-001");
        assertThat(found2.getMedicationId()).isEqualTo("MED-ASA-001");
        assertThat(found3.getMedicationId()).isEqualTo("MED-ASA-001");
    }

    @Test
    @Order(5)
    @DisplayName("Should filter medications by category")
    void testGetMedicationsByCategory() throws Exception {
        // Arrange
        createMedicationWithCategory(tempMedicationDir, "MED-PIP-001", "Piperacillin", "Antibiotic");
        createMedicationWithCategory(tempMedicationDir, "MED-CEF-001", "Ceftriaxone", "Antibiotic");
        createMedicationWithCategory(tempMedicationDir, "MED-MET-001", "Metoprolol", "Cardiovascular");
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Act
        List<Medication> antibiotics = loader.getMedicationsByCategory("Antibiotic");
        List<Medication> cardiovascular = loader.getMedicationsByCategory("Cardiovascular");

        // Assert
        assertThat(antibiotics).hasSize(2);
        assertThat(cardiovascular).hasSize(1);
        assertThat(antibiotics).extracting("name")
            .containsExactlyInAnyOrder("Piperacillin", "Ceftriaxone");
    }

    @Test
    @Order(6)
    @DisplayName("Should filter medications by formulary status")
    void testGetFormularyMedications() throws Exception {
        // Arrange
        createMedicationWithFormularyStatus(tempMedicationDir, "MED-001", "Preferred", true);
        createMedicationWithFormularyStatus(tempMedicationDir, "MED-002", "Alternative", false);
        createMedicationWithFormularyStatus(tempMedicationDir, "MED-003", "Preferred", true);
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Act
        List<Medication> formulary = loader.getFormularyMedications();

        // Assert
        assertThat(formulary).hasSize(2);
        assertThat(formulary).allMatch(m -> m.isFormulary());
    }

    @Test
    @Order(7)
    @DisplayName("Should filter high-alert medications (insulin, heparin, warfarin)")
    void testGetHighAlertMedications() throws Exception {
        // Arrange
        createHighAlertMedication(tempMedicationDir, "MED-INS-001", "Insulin", true);
        createHighAlertMedication(tempMedicationDir, "MED-HEP-001", "Heparin", true);
        createHighAlertMedication(tempMedicationDir, "MED-ASA-001", "Aspirin", false);
        loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

        // Act
        List<Medication> highAlert = loader.getHighAlertMedications();

        // Assert
        assertThat(highAlert).hasSize(2);
        assertThat(highAlert).extracting("name")
            .containsExactlyInAnyOrder("Insulin", "Heparin");
    }

    @Test
    @Order(8)
    @DisplayName("Should gracefully handle malformed YAML files")
    void testInvalidYAMLHandling() throws Exception {
        // Arrange
        Path malformedYAML = tempMedicationDir.resolve("malformed.yaml");
        Files.writeString(malformedYAML, "invalid: yaml: structure:::");

        // Act & Assert
        assertThrows(MedicationLoadException.class, () -> {
            loader.loadMedicationsFromDirectory(tempMedicationDir.toString());
        });
    }

    @Nested
    @DisplayName("Edge Case Tests")
    class EdgeCaseTests {

        @Test
        @DisplayName("Should throw exception when medicationId is missing")
        void testMissingMedicationId() throws Exception {
            // Arrange
            Path invalidMed = tempMedicationDir.resolve("no-id.yaml");
            Files.writeString(invalidMed, "name: NoIdMedication\ncategory: Test");

            // Act & Assert
            MedicationLoadException exception = assertThrows(
                MedicationLoadException.class,
                () -> loader.loadMedicationsFromDirectory(tempMedicationDir.toString())
            );
            assertThat(exception.getMessage()).contains("medicationId is required");
        }

        @Test
        @DisplayName("Should log warning and use first entry when duplicate medicationId found")
        void testDuplicateMedicationId() throws Exception {
            // Arrange
            createSampleMedication(tempMedicationDir, "MED-DUP-001", "First Med");
            Path duplicateFile = tempMedicationDir.resolve("MED-DUP-001-duplicate.yaml");
            Files.writeString(duplicateFile, MedicationTestData.createMedicationYAML(
                "MED-DUP-001", "Second Med", "Test"));

            // Act
            loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

            // Assert
            Medication loaded = loader.getMedicationById("MED-DUP-001");
            assertThat(loaded.getName()).isEqualTo("First Med"); // First wins
        }

        @Test
        @DisplayName("Should handle empty YAML directory gracefully")
        void testEmptyDirectory() throws Exception {
            // Act
            loader.loadMedicationsFromDirectory(tempMedicationDir.toString());

            // Assert
            assertThat(loader.getAllMedications()).isEmpty();
            assertThat(loader.getMedicationCount()).isZero();
        }
    }

    @Nested
    @DisplayName("Performance Tests")
    class PerformanceTests {

        @Test
        @DisplayName("Should load 100 medications in <5 seconds")
        @Timeout(5)
        void testLoadPerformance() throws Exception {
            // Arrange
            createTestMedicationYAMLs(tempMedicationDir, 100);

            // Act
            long startTime = System.currentTimeMillis();
            loader.loadMedicationsFromDirectory(tempMedicationDir.toString());
            long duration = System.currentTimeMillis() - startTime;

            // Assert
            assertThat(duration).isLessThan(5000);
            assertThat(loader.getAllMedications()).hasSize(100);
        }

        @Test
        @DisplayName("Should cache results - second getInstance() should be instant")
        void testCachingPerformance() {
            // Arrange
            MedicationDatabaseLoader first = MedicationDatabaseLoader.getInstance();

            // Act
            long startTime = System.nanoTime();
            MedicationDatabaseLoader second = MedicationDatabaseLoader.getInstance();
            long duration = System.nanoTime() - startTime;

            // Assert
            assertThat(second).isSameAs(first);
            assertThat(duration).isLessThan(1_000_000); // <1ms
        }
    }

    // Helper methods
    private void createTestMedicationYAMLs(Path dir, int count) throws Exception {
        for (int i = 1; i <= count; i++) {
            String id = String.format("MED-%03d", i);
            String name = "Medication" + i;
            createSampleMedication(dir, id, name);
        }
    }

    private void createSampleMedication(Path dir, String id, String name) throws Exception {
        String yaml = MedicationTestData.createMedicationYAML(id, name, "test-category");
        Files.writeString(dir.resolve(id + ".yaml"), yaml);
    }

    private void createMedicationWithCategory(Path dir, String id, String name, String category) throws Exception {
        String yaml = String.format("""
            medication_id: %s
            name: %s
            category: %s
            formulary: true
            """, id, name, category);
        Files.writeString(dir.resolve(id + ".yaml"), yaml);
    }

    private void createMedicationWithFormularyStatus(Path dir, String id, String status, boolean formulary) throws Exception {
        String yaml = String.format("""
            medication_id: %s
            formulary_status: %s
            formulary: %b
            """, id, status, formulary);
        Files.writeString(dir.resolve(id + ".yaml"), yaml);
    }

    private void createHighAlertMedication(Path dir, String id, String name, boolean highAlert) throws Exception {
        String yaml = String.format("""
            medication_id: %s
            name: %s
            high_alert: %b
            """, id, name, highAlert);
        Files.writeString(dir.resolve(id + ".yaml"), yaml);
    }
}
