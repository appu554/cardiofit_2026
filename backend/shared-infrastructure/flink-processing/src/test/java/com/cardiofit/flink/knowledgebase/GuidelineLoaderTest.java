package com.cardiofit.flink.knowledgebase;

import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.time.LocalDate;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive Tests for GuidelineLoader
 *
 * Tests guideline YAML loading, parsing, caching, and filtering.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
@DisplayName("Guideline Loader Tests")
class GuidelineLoaderTest {

    private static GuidelineLoader guidelineLoader;

    @BeforeAll
    static void setUp() {
        guidelineLoader = GuidelineLoader.getInstance();
    }

    // ============================================================
    // BASIC LOADING TESTS
    // ============================================================

    @Test
    @DisplayName("Should load all guideline YAMLs successfully")
    void testLoadAllGuidelines() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();

        assertNotNull(guidelines, "Guidelines list should not be null");
        assertTrue(guidelines.size() >= 3, "Should load at least 3 guidelines");

        System.out.println("Loaded " + guidelines.size() + " guidelines:");
        for (Guideline g : guidelines) {
            System.out.println("  - " + g.getGuidelineId() + ": " + g.getShortName());
        }
    }

    @Test
    @DisplayName("Should load ACC/AHA STEMI 2023 guideline correctly")
    void testLoadAccAhaStemi2023() {
        Guideline guideline = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");

        assertNotNull(guideline, "ACC/AHA STEMI 2023 guideline should be loaded");
        assertEquals("GUIDE-ACCAHA-STEMI-2023", guideline.getGuidelineId());
        assertEquals("ACC/AHA STEMI 2023", guideline.getShortName());
        assertEquals("2023 ACC/AHA/SCAI Guideline for the Management of Patients With Acute Myocardial Infarction",
            guideline.getName());
        assertEquals("American College of Cardiology / American Heart Association / Society for Cardiovascular Angiography and Interventions",
            guideline.getOrganization());
        assertEquals("CURRENT", guideline.getStatus());
    }

    @Test
    @DisplayName("Should parse guideline metadata correctly")
    void testGuidelineMetadata() {
        Guideline guideline = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");
        assertNotNull(guideline);

        // Version info
        assertEquals("2023.1", guideline.getVersion());
        assertEquals(LocalDate.of(2023, 4, 20), guideline.getPublicationDate());
        assertEquals(LocalDate.of(2023, 4, 20), guideline.getLastReviewDate());
        assertEquals(LocalDate.of(2028, 4, 20), guideline.getNextReviewDate());

        // Publication details
        Guideline.Publication pub = guideline.getPublication();
        assertNotNull(pub, "Publication details should be present");
        assertEquals("Journal of the American College of Cardiology", pub.getJournal());
        assertEquals(2023, pub.getYear());
        assertEquals(81, pub.getVolume());
        assertEquals(14, pub.getIssue());
        assertEquals("37079885", pub.getPmid());
        assertEquals("10.1016/j.jacc.2023.04.001", pub.getDoi());
    }

    // ============================================================
    // RECOMMENDATION PARSING TESTS
    // ============================================================

    @Test
    @DisplayName("Should parse recommendations with all fields")
    void testRecommendationParsing() {
        Guideline guideline = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");
        assertNotNull(guideline);

        List<Recommendation> recommendations = guideline.getRecommendations();
        assertNotNull(recommendations, "Recommendations should not be null");
        assertEquals(8, recommendations.size(), "Should have 8 recommendations");

        // Test first recommendation (12-lead ECG)
        Recommendation rec1 = recommendations.stream()
            .filter(r -> "ACC-STEMI-2023-REC-001".equals(r.getRecommendationId()))
            .findFirst()
            .orElse(null);

        assertNotNull(rec1, "Recommendation 001 should exist");
        assertEquals("1.1", rec1.getNumber());
        assertEquals("Initial Evaluation and Risk Stratification", rec1.getSection());
        assertEquals("12-Lead ECG Within 10 Minutes of First Medical Contact", rec1.getTitle());
        assertEquals("STRONG", rec1.getStrength());
        assertEquals("Class I", rec1.getClassOfRecommendation());
        assertEquals("HIGH", rec1.getEvidenceQuality());
        assertEquals("B-NR", rec1.getLevelOfEvidence());

        assertTrue(rec1.getKeyEvidence().contains("37079885"), "Should reference guideline PMID");
        assertTrue(rec1.getLinkedProtocolActions().contains("STEMI-ACT-001"), "Should link to protocol action");
    }

    @Test
    @DisplayName("Should parse aspirin recommendation correctly")
    void testAspirinRecommendation() {
        Guideline guideline = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");
        assertNotNull(guideline);

        Recommendation aspirinRec = guideline.getRecommendations().stream()
            .filter(r -> "ACC-STEMI-2023-REC-003".equals(r.getRecommendationId()))
            .findFirst()
            .orElse(null);

        assertNotNull(aspirinRec, "Aspirin recommendation should exist");
        assertEquals("Aspirin 162-325 mg Loading Dose", aspirinRec.getTitle());
        assertEquals("STRONG", aspirinRec.getStrength());
        assertEquals("HIGH", aspirinRec.getEvidenceQuality());

        // Should reference ISIS-2 trial
        assertTrue(aspirinRec.getKeyEvidence().contains("3081859"),
            "Should reference ISIS-2 trial (PMID 3081859)");

        // Should link to aspirin action
        assertTrue(aspirinRec.getLinkedProtocolActions().contains("STEMI-ACT-002"),
            "Should link to STEMI-ACT-002 (aspirin action)");
    }

    // ============================================================
    // STATUS FILTERING TESTS
    // ============================================================

    @Test
    @DisplayName("Should filter guidelines by CURRENT status")
    void testCurrentGuidelines() {
        List<Guideline> current = guidelineLoader.getCurrentGuidelines();

        assertNotNull(current);
        assertTrue(current.size() > 0, "Should have at least one current guideline");

        // All should have CURRENT status
        for (Guideline g : current) {
            assertTrue(g.isCurrent(), "Guideline " + g.getGuidelineId() + " should be CURRENT");
        }

        System.out.println("Current guidelines: " + current.size());
    }

    @Test
    @DisplayName("Should identify superseded guidelines")
    void testSupersededGuideline() {
        Guideline guideline2013 = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2013");

        if (guideline2013 != null) {
            assertEquals("SUPERSEDED", guideline2013.getStatus(),
                "2013 guideline should be marked as SUPERSEDED");

            // Note: supersededBy would need to be set in the YAML or via relationship processing
            System.out.println("2013 guideline status: " + guideline2013.getStatus());
        } else {
            System.out.println("2013 guideline not loaded (optional for this test)");
        }
    }

    // ============================================================
    // CACHING TESTS
    // ============================================================

    @Test
    @DisplayName("Should cache guidelines properly (singleton behavior)")
    void testGuidelineCaching() {
        GuidelineLoader loader1 = GuidelineLoader.getInstance();
        GuidelineLoader loader2 = GuidelineLoader.getInstance();

        assertSame(loader1, loader2, "Should return same singleton instance");

        Guideline g1 = loader1.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");
        Guideline g2 = loader2.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");

        assertSame(g1, g2, "Should return same cached guideline object");
    }

    @Test
    @DisplayName("Should return null for missing guideline")
    void testMissingGuideline() {
        Guideline missing = guidelineLoader.getGuidelineById("GUIDE-NONEXISTENT-9999");

        assertNull(missing, "Should return null for non-existent guideline");
    }

    // ============================================================
    // VALIDATION TESTS
    // ============================================================

    @Test
    @DisplayName("All recommendations should have required GRADE fields")
    void testGradeFieldsPresent() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                assertNotNull(rec.getRecommendationId(),
                    "Recommendation should have ID in " + guideline.getGuidelineId());
                assertNotNull(rec.getStrength(),
                    "Recommendation " + rec.getRecommendationId() + " should have strength");
                assertNotNull(rec.getEvidenceQuality(),
                    "Recommendation " + rec.getRecommendationId() + " should have evidence quality");

                System.out.println("  ✓ " + rec.getRecommendationId() + ": " +
                    rec.getStrength() + " / " + rec.getEvidenceQuality());
            }
        }
    }

    @Test
    @DisplayName("All guidelines should have publication PMID")
    void testPublicationPmidPresent() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();

        for (Guideline guideline : guidelines) {
            Guideline.Publication pub = guideline.getPublication();
            if (pub != null) {
                assertNotNull(pub.getPmid(),
                    "Guideline " + guideline.getGuidelineId() + " should have publication PMID");
                System.out.println("  ✓ " + guideline.getGuidelineId() + " PMID: " + pub.getPmid());
            }
        }
    }

    // ============================================================
    // PERFORMANCE TESTS
    // ============================================================

    @Test
    @DisplayName("Guideline retrieval should be fast (<5ms)")
    void testRetrievalPerformance() {
        long startTime = System.nanoTime();

        Guideline guideline = guidelineLoader.getGuidelineById("GUIDE-ACCAHA-STEMI-2023");

        long duration = (System.nanoTime() - startTime) / 1_000_000; // Convert to ms

        assertNotNull(guideline);
        assertTrue(duration < 5, "Retrieval should take <5ms, took: " + duration + "ms");

        System.out.println("Guideline retrieval time: " + duration + "ms");
    }
}
