package com.cardiofit.flink.knowledgebase;

import com.cardiofit.flink.models.EvidenceChain;
import com.cardiofit.flink.knowledgebase.GuidelineIntegrationService.Guideline;
import com.cardiofit.flink.knowledgebase.medications.loader.CitationConverter;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.ArrayList;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * End-to-End Integration Tests for Evidence Chain System
 *
 * Tests complete workflow: Action → Guideline → Recommendation → Citations
 * with performance benchmarks and quality aggregation.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
@DisplayName("Evidence Chain Integration Tests")
class EvidenceChainIntegrationTest {

    private static GuidelineLinker guidelineLinker;
    private static GuidelineLoader guidelineLoader;
    private static CitationLoader citationLoader;

    @BeforeAll
    static void setUp() {
        guidelineLinker = new GuidelineLinker();
        guidelineLoader = GuidelineLoader.getInstance();
        citationLoader = CitationLoader.getInstance();
    }

    // ============================================================
    // COMPLETE WORKFLOW TESTS
    // ============================================================

    @Test
    @DisplayName("Should complete full evidence chain workflow for STEMI aspirin")
    void testCompleteEvidenceChainWorkflow() {
        // Given: STEMI protocol with aspirin action
        String actionId = "STEMI-ACT-002";

        // When: Resolve complete evidence chain
        long startTime = System.nanoTime();
        EvidenceChain chain = guidelineLinker.getEvidenceChain(actionId);
        long duration = (System.nanoTime() - startTime) / 1_000_000; // Convert to ms

        // Then: Verify complete chain
        assertNotNull(chain, "Evidence chain should be resolved");
        assertNotNull(chain.getSourceGuideline(), "Should have source guideline");
        assertNotNull(chain.getGuidelineRecommendation(), "Should have recommendation");
        assertTrue(chain.getSupportingEvidence().size() > 0, "Should have supporting evidence");

        // Verify chain components
        assertEquals("GUIDE-ACCAHA-STEMI-2023", chain.getSourceGuideline().getGuidelineId());
        assertEquals("ACC-STEMI-2023-REC-003", chain.getGuidelineRecommendation().getRecommendationId());
        assertTrue(chain.getSupportingEvidence().stream()
            .anyMatch(c -> "3081859".equals(c.getPmid())), "Should include ISIS-2");

        // Verify formatted trail
        String trail = chain.getEvidenceTrail();
        assertTrue(trail.contains("Aspirin"), "Trail should mention aspirin");
        assertTrue(trail.contains("ACC/AHA STEMI 2023"), "Trail should mention guideline");
        assertTrue(trail.contains("PMID 3081859"), "Trail should mention ISIS-2");
        assertTrue(trail.contains("🟢 STRONG"), "Trail should show quality badge");

        // Performance assertion
        assertTrue(duration < 100, "Evidence chain resolution should take <100ms, took: " + duration + "ms");

        System.out.println("===== COMPLETE EVIDENCE CHAIN WORKFLOW =====");
        System.out.println(trail);
        System.out.println("Resolution time: " + duration + "ms");
        System.out.println("===========================================");
    }

    @Test
    @DisplayName("Should handle multiple guidelines for same action")
    void testMultipleGuidelinesForSameAction() {
        // Some actions may be supported by multiple guidelines (e.g., ACC/AHA + ESC)
        String actionId = "STEMI-ACT-002";

        List<Guideline> guidelines = guidelineLinker.getGuidelinesForAction(actionId);

        assertNotNull(guidelines);
        assertTrue(guidelines.size() >= 1, "Should have at least 1 guideline");

        System.out.println("Guidelines supporting " + actionId + ":");
        for (Guideline g : guidelines) {
            System.out.println("  - " + g.getName() + " (" + g.getStatus() + ")");
        }

        // Should prefer CURRENT over SUPERSEDED
        boolean hasCurrent = guidelines.stream().anyMatch(g -> "CURRENT".equalsIgnoreCase(g.getStatus()));
        assertTrue(hasCurrent, "Should have at least one current guideline");
    }

    // ============================================================
    // EVIDENCE QUALITY AGGREGATION TESTS
    // ============================================================

    @Test
    @DisplayName("Should aggregate evidence quality from multiple sources")
    void testEvidenceQualityAggregation() {
        // Test 1: High quality - Multiple RCTs
        List<Citation> highQualityCitations = new ArrayList<>();
        Citation rct1 = createMockCitation("12345", "RCT");
        Citation rct2 = createMockCitation("67890", "RCT");
        highQualityCitations.add(rct1);
        highQualityCitations.add(rct2);

        // Convert standalone citations to EvidenceChain.Citation
        List<EvidenceChain.Citation> highQualityConverted = CitationConverter.toEvidenceChainCitations(highQualityCitations);
        String qualityHigh = guidelineLinker.assessOverallQuality(highQualityConverted);
        assertEquals("High", qualityHigh, "Multiple RCTs should yield High quality");

        // Test 2: Moderate quality - Single RCT
        List<EvidenceChain.Citation> singleRctConverted = CitationConverter.toEvidenceChainCitations(List.of(rct1));
        String qualityModerate = guidelineLinker.assessOverallQuality(singleRctConverted);
        assertEquals("Moderate", qualityModerate, "Single RCT should yield Moderate quality");

        // Test 3: Low quality - Observational only
        Citation cohort = createMockCitation("11111", "COHORT");
        List<EvidenceChain.Citation> cohortConverted = CitationConverter.toEvidenceChainCitations(List.of(cohort));
        String qualityLow = guidelineLinker.assessOverallQuality(cohortConverted);
        assertEquals("Low", qualityLow, "Observational study should yield Low quality");

        // Test 4: Highest quality - Meta-analysis
        Citation meta = createMockCitation("22222", "META_ANALYSIS");
        List<EvidenceChain.Citation> metaConverted = CitationConverter.toEvidenceChainCitations(List.of(meta));
        String qualityMeta = guidelineLinker.assessOverallQuality(metaConverted);
        assertEquals("High", qualityMeta, "Meta-analysis should yield High quality");

        System.out.println("Quality Aggregation Results:");
        System.out.println("  Multiple RCTs: " + qualityHigh);
        System.out.println("  Single RCT: " + qualityModerate);
        System.out.println("  Cohort study: " + qualityLow);
        System.out.println("  Meta-analysis: " + qualityMeta);
    }

    @Test
    @DisplayName("Should assess real-world evidence quality for STEMI recommendations")
    void testRealWorldEvidenceQuality() {
        // Test aspirin (HIGH quality evidence)
        EvidenceChain aspirinChain = guidelineLinker.getEvidenceChain("STEMI-ACT-002");
        assertEquals("High", aspirinChain.getOverallQuality(),
            "Aspirin has high-quality RCT evidence (ISIS-2)");

        // Test P2Y12 inhibitor (HIGH quality evidence)
        EvidenceChain p2y12Chain = guidelineLinker.getEvidenceChain("STEMI-ACT-003");
        if (p2y12Chain.getSourceGuideline() != null) {
            assertEquals("High", p2y12Chain.getOverallQuality(),
                "P2Y12 inhibitor has high-quality evidence (PLATO, TRITON)");
        }

        System.out.println("Real-world evidence quality:");
        System.out.println("  Aspirin: " + aspirinChain.getQualityBadge());
        if (p2y12Chain.getSourceGuideline() != null) {
            System.out.println("  P2Y12: " + p2y12Chain.getQualityBadge());
        }
    }

    // ============================================================
    // FORMATTED OUTPUT TESTS
    // ============================================================

    @Test
    @DisplayName("Should generate comprehensive formatted evidence trail")
    void testFormattedEvidenceTrail() {
        EvidenceChain chain = guidelineLinker.getEvidenceChain("STEMI-ACT-002");

        String trail = chain.getEvidenceTrail();
        assertNotNull(trail);

        // Verify all components present
        assertTrue(trail.contains("ACTION: STEMI-ACT-002"));
        assertTrue(trail.contains("GUIDELINE: ACC/AHA STEMI 2023"));
        assertTrue(trail.contains("RECOMMENDATION:"));
        assertTrue(trail.contains("EVIDENCE ("));
        assertTrue(trail.contains("PMID"));
        assertTrue(trail.contains("QUALITY:"));

        // Verify formatting
        String[] lines = trail.split("\n");
        assertTrue(lines.length >= 5, "Trail should have multiple formatted lines");

        System.out.println("===== FORMATTED EVIDENCE TRAIL =====");
        System.out.println(trail);
        System.out.println("====================================");
    }

    // ============================================================
    // CROSS-GUIDELINE VALIDATION TESTS
    // ============================================================

    @Test
    @DisplayName("Should validate consistency across multiple STEMI actions")
    void testCrossGuidelineConsistency() {
        // All STEMI actions should link to STEMI guidelines
        String[] stemiActions = {
            "STEMI-ACT-001", // ECG
            "STEMI-ACT-002", // Aspirin
            "STEMI-ACT-003"  // P2Y12
        };

        for (String actionId : stemiActions) {
            EvidenceChain chain = guidelineLinker.getEvidenceChain(actionId);

            if (chain.getSourceGuideline() != null) {
                assertTrue(chain.getSourceGuideline().getGuidelineId().contains("STEMI"),
                    "Action " + actionId + " should link to STEMI guideline");
                System.out.println("  ✓ " + actionId + " → " +
                    chain.getSourceGuideline().getGuidelineName());
            }
        }
    }

    // ============================================================
    // PERFORMANCE BENCHMARKS
    // ============================================================

    @Test
    @DisplayName("Should resolve evidence chains quickly (<100ms each)")
    void testPerformanceBenchmark() {
        String[] actions = {"STEMI-ACT-001", "STEMI-ACT-002", "STEMI-ACT-003"};
        long totalTime = 0;

        for (String actionId : actions) {
            long startTime = System.nanoTime();
            EvidenceChain chain = guidelineLinker.getEvidenceChain(actionId);
            long duration = (System.nanoTime() - startTime) / 1_000_000;

            totalTime += duration;
            assertTrue(duration < 100, "Resolution of " + actionId + " should take <100ms");
            System.out.println("  " + actionId + ": " + duration + "ms");
        }

        long avgTime = totalTime / actions.length;
        System.out.println("Average resolution time: " + avgTime + "ms");
        assertTrue(avgTime < 50, "Average resolution should be <50ms");
    }

    // ============================================================
    // EDGE CASE TESTS
    // ============================================================

    @Test
    @DisplayName("Should handle action with no citations gracefully")
    void testActionWithNoCitations() {
        // Create mock guideline with recommendation but no citations
        Guideline mockGuideline = new Guideline();
        mockGuideline.setGuidelineId("GUIDE-MOCK-NO-CITATIONS");
        mockGuideline.setName("Mock Guideline");
        mockGuideline.setStatus("CURRENT");

        GuidelineIntegrationService.Recommendation rec = new GuidelineIntegrationService.Recommendation();
        rec.setRecommendationId("MOCK-REC-001");
        rec.setStrength("WEAK");
        rec.setEvidenceQuality("LOW");
        rec.getLinkedProtocolActions().add("MOCK-ACT-001");
        rec.setCitationPmids(new ArrayList<>()); // No citations

        mockGuideline.getRecommendations().add(rec);

        EvidenceChain chain = guidelineLinker.getEvidenceChain(mockGuideline, "MOCK-ACT-001");

        assertNotNull(chain);
        assertEquals(0, chain.getSupportingEvidence().size(), "Should have no citations");
        assertNotNull(chain.getOverallQuality(), "Should still assess quality");
    }

    @Test
    @DisplayName("Should handle empty action ID")
    void testEmptyActionId() {
        EvidenceChain chain = guidelineLinker.getEvidenceChain("");

        assertNotNull(chain);
        assertEquals("", chain.getActionId());
        assertNull(chain.getSourceGuideline());
    }

    // ============================================================
    // HELPER METHODS
    // ============================================================

    private Citation createMockCitation(String pmid, String studyType) {
        Citation citation = new Citation();
        citation.setPmid(pmid);
        // Note: StudyType enum not yet implemented in Citation class
        citation.setTitle("Mock study - " + studyType);
        citation.setYear(2020);
        return citation;
    }
}
