package com.cardiofit.flink.knowledgebase.medications;

import com.cardiofit.flink.knowledgebase.medications.loader.MedicationDatabaseLoader;
import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import com.cardiofit.flink.models.DrugInteraction;
import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker;
import com.cardiofit.flink.knowledgebase.medications.safety.DrugInteractionChecker.InteractionResult;
import org.junit.jupiter.api.*;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Performance benchmark tests for Medication Database.
 *
 * Tests load time, lookup performance, and interaction checking speed.
 * Coverage Target: Performance benchmarking
 * Test Count: 3 performance tests
 */
@DisplayName("Medication Database Performance Test Suite")
@Tag("performance")
class MedicationDatabasePerformanceTest {

    @Test
    @DisplayName("Should load 100 medications in <5 seconds")
    @Timeout(5)
    void testLoadTime() throws Exception {
        // Arrange
        Path testDir = Files.createTempDirectory("med-perf-test");
        createTestMedicationDirectory(testDir, 100);

        // Act
        long startTime = System.currentTimeMillis();
        MedicationDatabaseLoader loader = MedicationDatabaseLoader.getInstance();
        loader.loadMedicationsFromDirectory(testDir.toString());
        long duration = System.currentTimeMillis() - startTime;

        // Assert
        assertThat(duration).isLessThan(5000);
        assertThat(loader.getAllMedications()).hasSize(100);

        System.out.printf("✓ Load performance: %d medications loaded in %d ms%n", 100, duration);
    }

    @Test
    @DisplayName("Should perform 10,000 medication lookups in <1 second")
    @Timeout(1)
    void testLookupPerformance() throws Exception {
        // Arrange
        MedicationDatabaseLoader loader = MedicationDatabaseLoader.getInstance();
        String[] medicationIds = generateMedicationIds(100);

        // Pre-load medications
        Path testDir = Files.createTempDirectory("med-lookup-test");
        createTestMedicationDirectory(testDir, 100);
        loader.loadMedicationsFromDirectory(testDir.toString());

        // Act
        long startTime = System.currentTimeMillis();
        for (int i = 0; i < 10000; i++) {
            String id = medicationIds[i % 100];
            Medication med = loader.getMedicationById(id);
            assertThat(med).isNotNull();
        }
        long duration = System.currentTimeMillis() - startTime;

        // Assert
        assertThat(duration).isLessThan(1000);
        double avgTime = (duration / 10000.0);
        System.out.printf("✓ Lookup performance: 10,000 lookups in %d ms (avg: %.3f ms/lookup)%n",
            duration, avgTime);
    }

    @Test
    @DisplayName("Should check 100 medication pairs for interactions in <2 seconds")
    @Timeout(2)
    void testInteractionCheckPerformance() {
        // Arrange
        DrugInteractionChecker checker = new DrugInteractionChecker();
        List<String> medications = Arrays.asList(
            "Warfarin", "Aspirin", "Heparin", "Clopidogrel", "Metoprolol",
            "Lisinopril", "Furosemide", "Digoxin", "Amiodarone", "Ciprofloxacin"
        );

        // Act
        long startTime = System.currentTimeMillis();
        int checks = 0;
        for (String med1 : medications) {
            for (String med2 : medications) {
                if (!med1.equals(med2)) {
                    InteractionResult result = checker.checkInteraction(med1, med2);
                    checks++;
                }
            }
        }
        long duration = System.currentTimeMillis() - startTime;

        // Assert
        assertThat(checks).isEqualTo(90); // 10 * 9 pairs
        assertThat(duration).isLessThan(2000);
        double avgTime = (duration / 90.0);
        System.out.printf("✓ Interaction check performance: %d pairs checked in %d ms (avg: %.2f ms/pair)%n",
            checks, duration, avgTime);
    }

    @Nested
    @DisplayName("Memory and Caching Tests")
    class MemoryTests {

        @Test
        @DisplayName("Should cache singleton instance efficiently")
        void testSingletonCaching() {
            // Arrange
            MedicationDatabaseLoader first = MedicationDatabaseLoader.getInstance();

            // Act
            long startTime = System.nanoTime();
            MedicationDatabaseLoader second = MedicationDatabaseLoader.getInstance();
            long duration = System.nanoTime() - startTime;

            // Assert
            assertThat(second).isSameAs(first);
            assertThat(duration).isLessThan(1_000_000); // <1ms
            System.out.printf("✓ Singleton cache: %.3f μs%n", duration / 1000.0);
        }

        @Test
        @DisplayName("Should handle repeated lookups efficiently")
        void testRepeatedLookupCaching() {
            // Arrange
            MedicationDatabaseLoader loader = MedicationDatabaseLoader.getInstance();

            // Act - First lookup (may involve initial load)
            long firstStart = System.nanoTime();
            Medication first = loader.getMedicationById("MED-001");
            long firstDuration = System.nanoTime() - firstStart;

            // Act - Second lookup (should be cached)
            long secondStart = System.nanoTime();
            Medication second = loader.getMedicationById("MED-001");
            long secondDuration = System.nanoTime() - secondStart;

            // Assert
            assertThat(first).isSameAs(second);
            assertThat(secondDuration).isLessThan(firstDuration);
            System.out.printf("✓ Lookup caching: first=%.3f μs, second=%.3f μs%n",
                firstDuration / 1000.0, secondDuration / 1000.0);
        }
    }

    // Helper methods
    private void createTestMedicationDirectory(Path dir, int count) throws Exception {
        for (int i = 1; i <= count; i++) {
            String id = String.format("MED-%03d", i);
            String yaml = String.format("""
                medication_id: %s
                name: Medication%d
                generic_name: medication%d
                category: test
                drug_class: test-class
                standard_dose: 100mg
                route: IV
                frequency: q24h
                formulary: true
                high_alert: false
                """, id, i, i);
            Files.writeString(dir.resolve(id + ".yaml"), yaml);
        }
    }

    private String[] generateMedicationIds(int count) {
        String[] ids = new String[count];
        for (int i = 0; i < count; i++) {
            ids[i] = String.format("MED-%03d", i + 1);
        }
        return ids;
    }
}
