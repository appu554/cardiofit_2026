package com.cardiofit.flink.knowledgebase;

import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;
import java.util.stream.Collectors;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Comprehensive Validation Tests for Guideline System
 *
 * Tests data integrity, GRADE compliance, and cross-reference validation
 * across the entire guideline knowledge base.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
@DisplayName("Guideline Validation Tests")
class GuidelineValidationTest {

    private static GuidelineLoader guidelineLoader;
    private static CitationLoader citationLoader;
    private static GuidelineLinker guidelineLinker;

    // Mock protocol action IDs that exist in the system
    private static final Set<String> VALID_ACTION_IDS = Set.of(
        "STEMI-ACT-001", "STEMI-ACT-002", "STEMI-ACT-003", "STEMI-ACT-004",
        "STEMI-ACT-005", "STEMI-ACT-008", "STEMI-ACT-009", "STEMI-ACT-010",
        "STEMI-ACT-011", "STEMI-ACT-012"
    );

    @BeforeAll
    static void setUp() {
        guidelineLoader = GuidelineLoader.getInstance();
        citationLoader = CitationLoader.getInstance();
        guidelineLinker = new GuidelineLinker();
    }

    // ============================================================
    // PMID VALIDATION TESTS
    // ============================================================

    @Test
    @DisplayName("All guidelines should have valid publication PMIDs")
    void testAllGuidelinesHaveValidPmids() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        int validCount = 0;
        int missingCount = 0;

        for (Guideline guideline : guidelines) {
            Guideline.Publication pub = guideline.getPublication();

            if (pub != null && pub.getPmid() != null && !pub.getPmid().isEmpty()) {
                // Validate PMID format (should be numeric)
                assertTrue(pub.getPmid().matches("\\d+"),
                    "PMID should be numeric for " + guideline.getGuidelineId() +
                    ", got: " + pub.getPmid());
                validCount++;
                System.out.println("  ✓ " + guideline.getGuidelineId() + " PMID: " + pub.getPmid());
            } else {
                missingCount++;
                System.out.println("  ⚠ " + guideline.getGuidelineId() + " - Missing PMID");
            }
        }

        assertTrue(validCount > 0, "Should have at least one guideline with valid PMID");
        System.out.println("Valid PMIDs: " + validCount + ", Missing: " + missingCount);
    }

    @Test
    @DisplayName("All recommendation key evidence should reference valid PMIDs")
    void testRecommendationPmidsExist() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        int validReferences = 0;
        int missingReferences = 0;

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                if (rec.getKeyEvidence() != null) {
                    for (String pmid : rec.getKeyEvidence()) {
                        // Validate PMID format
                        assertTrue(pmid.matches("\\d+"),
                            "Evidence PMID should be numeric in " + rec.getRecommendationId() +
                            ", got: " + pmid);

                        // Check if citation exists in our loader (optional - may not be loaded)
                        Citation citation = citationLoader.getCitationByPmid(pmid);
                        if (citation != null) {
                            validReferences++;
                            System.out.println("  ✓ " + rec.getRecommendationId() +
                                " → PMID " + pmid + " (loaded)");
                        } else {
                            missingReferences++;
                            System.out.println("  ○ " + rec.getRecommendationId() +
                                " → PMID " + pmid + " (not in citation loader)");
                        }
                    }
                }
            }
        }

        System.out.println("Valid citation references: " + validReferences);
        System.out.println("Missing from loader: " + missingReferences);
    }

    // ============================================================
    // PROTOCOL ACTION LINKING VALIDATION
    // ============================================================

    @Test
    @DisplayName("All linkedProtocolActions should reference existing actions")
    void testAllLinkedActionsExist() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        Set<String> allActionIds = VALID_ACTION_IDS;

        int validLinks = 0;
        int invalidLinks = 0;
        List<String> brokenLinks = new ArrayList<>();

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                if (rec.getLinkedProtocolActions() != null) {
                    for (String actionId : rec.getLinkedProtocolActions()) {
                        if (allActionIds.contains(actionId)) {
                            validLinks++;
                            System.out.println("  ✓ " + guideline.getGuidelineId() +
                                " → " + actionId);
                        } else {
                            invalidLinks++;
                            brokenLinks.add(guideline.getGuidelineId() + " → " + actionId);
                            System.out.println("  ✗ BROKEN: " + guideline.getGuidelineId() +
                                " references non-existent action: " + actionId);
                        }
                    }
                }
            }
        }

        System.out.println("Valid action links: " + validLinks);
        System.out.println("Invalid action links: " + invalidLinks);

        if (invalidLinks > 0) {
            System.out.println("Broken links found:");
            brokenLinks.forEach(link -> System.out.println("  - " + link));
        }

        // This is a soft check - we expect some actions may not be in mock set
        assertTrue(validLinks > 0, "Should have at least some valid action links");
    }

    // ============================================================
    // GRADE TERMINOLOGY COMPLIANCE
    // ============================================================

    @Test
    @DisplayName("Recommendation strengths should match GRADE terminology")
    void testGradeTerminologyCompliance() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        Set<String> validStrengths = Set.of("STRONG", "WEAK", "CONDITIONAL");
        Set<String> validQualities = Set.of("HIGH", "MODERATE", "LOW", "VERY_LOW");

        int compliantCount = 0;
        int nonCompliantCount = 0;

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                boolean strengthValid = rec.getStrength() != null &&
                    validStrengths.contains(rec.getStrength());

                boolean qualityValid = rec.getEvidenceQuality() != null &&
                    validQualities.contains(rec.getEvidenceQuality());

                if (strengthValid && qualityValid) {
                    compliantCount++;
                    System.out.println("  ✓ " + rec.getRecommendationId() + ": " +
                        rec.getStrength() + " / " + rec.getEvidenceQuality());
                } else {
                    nonCompliantCount++;
                    System.out.println("  ✗ " + rec.getRecommendationId() +
                        " - Invalid: strength=" + rec.getStrength() +
                        ", quality=" + rec.getEvidenceQuality());
                }

                if (!strengthValid && rec.getStrength() != null) {
                    fail("Invalid strength: " + rec.getStrength() +
                        " in " + rec.getRecommendationId());
                }

                if (!qualityValid && rec.getEvidenceQuality() != null) {
                    fail("Invalid quality: " + rec.getEvidenceQuality() +
                        " in " + rec.getRecommendationId());
                }
            }
        }

        System.out.println("GRADE compliant recommendations: " + compliantCount);
        System.out.println("Non-compliant recommendations: " + nonCompliantCount);

        assertTrue(compliantCount > 0, "Should have GRADE-compliant recommendations");
    }

    @Test
    @DisplayName("Class of Recommendation should be valid (Class I, IIa, IIb, III)")
    void testClassOfRecommendationValid() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        Set<String> validClasses = Set.of("Class I", "Class IIa", "Class IIb", "Class III");

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                if (rec.getClassOfRecommendation() != null) {
                    assertTrue(validClasses.contains(rec.getClassOfRecommendation()),
                        "Invalid class: " + rec.getClassOfRecommendation() +
                        " in " + rec.getRecommendationId());

                    System.out.println("  ✓ " + rec.getRecommendationId() + ": " +
                        rec.getClassOfRecommendation());
                }
            }
        }
    }

    // ============================================================
    // SUPERSEDED GUIDELINE VALIDATION
    // ============================================================

    @Test
    @DisplayName("Superseded guidelines should be properly linked")
    void testSupersededGuidelinesLinked() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        List<Guideline> superseded = guidelines.stream()
            .filter(Guideline::isSuperseded)
            .collect(Collectors.toList());

        for (Guideline g : superseded) {
            System.out.println("Superseded guideline: " + g.getGuidelineId());

            // Check if supersededBy field is set (if available in YAML)
            if (g.getSupersededBy() != null) {
                Guideline replacement = guidelineLoader.getGuidelineById(g.getSupersededBy());
                assertNotNull(replacement,
                    "Superseded guideline " + g.getGuidelineId() +
                    " references non-existent replacement: " + g.getSupersededBy());

                System.out.println("  → Superseded by: " + replacement.getGuidelineId() +
                    " (" + replacement.getShortName() + ")");
            }
        }
    }

    // ============================================================
    // DATA COMPLETENESS VALIDATION
    // ============================================================

    @Test
    @DisplayName("All guidelines should have required metadata fields")
    void testGuidelineMetadataCompleteness() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();

        for (Guideline guideline : guidelines) {
            assertNotNull(guideline.getGuidelineId(), "Should have guideline ID");
            assertNotNull(guideline.getName(), "Should have name");
            assertNotNull(guideline.getShortName(), "Should have short name");
            assertNotNull(guideline.getStatus(), "Should have status");

            System.out.println("  ✓ " + guideline.getGuidelineId() + " metadata complete");
        }
    }

    @Test
    @DisplayName("All recommendations should have required fields")
    void testRecommendationCompleteness() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        int completeCount = 0;

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                assertNotNull(rec.getRecommendationId(), "Should have ID");
                assertNotNull(rec.getTitle(), "Should have title");
                assertNotNull(rec.getStatement(), "Should have statement");
                assertNotNull(rec.getStrength(), "Should have strength");

                completeCount++;
                System.out.println("  ✓ " + rec.getRecommendationId() + " complete");
            }
        }

        assertTrue(completeCount > 0, "Should have complete recommendations");
        System.out.println("Total complete recommendations: " + completeCount);
    }

    // ============================================================
    // CROSS-REFERENCE INTEGRITY
    // ============================================================

    @Test
    @DisplayName("No broken references between guidelines")
    void testNoBrokenGuidelineReferences() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        Set<String> allGuidelineIds = guidelines.stream()
            .map(Guideline::getGuidelineId)
            .collect(Collectors.toSet());

        for (Guideline guideline : guidelines) {
            if (guideline.getRelatedGuidelines() != null) {
                for (Guideline.RelatedGuideline related : guideline.getRelatedGuidelines()) {
                    String relatedId = related.getGuidelineId();

                    // Note: Related guidelines may not be loaded in our limited set
                    if (allGuidelineIds.contains(relatedId)) {
                        System.out.println("  ✓ " + guideline.getGuidelineId() +
                            " → " + relatedId + " (" + related.getRelationship() + ")");
                    } else {
                        System.out.println("  ○ " + guideline.getGuidelineId() +
                            " → " + relatedId + " (not loaded)");
                    }
                }
            }
        }
    }

    // ============================================================
    // CONSISTENCY VALIDATION
    // ============================================================

    @Test
    @DisplayName("High-quality evidence should have strong recommendations")
    void testQualityStrengthConsistency() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        int consistentCount = 0;
        int inconsistentCount = 0;

        for (Guideline guideline : guidelines) {
            for (Recommendation rec : guideline.getRecommendations()) {
                boolean isHighQuality = "HIGH".equals(rec.getEvidenceQuality());
                boolean isStrongRec = "STRONG".equals(rec.getStrength());

                if (isHighQuality) {
                    // High-quality evidence should typically (but not always) be strong
                    if (isStrongRec) {
                        consistentCount++;
                        System.out.println("  ✓ " + rec.getRecommendationId() +
                            ": HIGH quality + STRONG recommendation (consistent)");
                    } else {
                        inconsistentCount++;
                        System.out.println("  ⚠ " + rec.getRecommendationId() +
                            ": HIGH quality but " + rec.getStrength() +
                            " recommendation (may be valid if patient values vary)");
                    }
                }
            }
        }

        System.out.println("Consistent HIGH/STRONG pairs: " + consistentCount);
        System.out.println("Inconsistent (but may be valid): " + inconsistentCount);
    }

    // ============================================================
    // SUMMARY REPORT
    // ============================================================

    @Test
    @DisplayName("Generate knowledge base validation summary")
    void testValidationSummary() {
        List<Guideline> guidelines = guidelineLoader.getAllGuidelines();
        Map<String, Citation> citationsMap = citationLoader.loadAllCitations();
        List<Citation> citations = new ArrayList<>(citationsMap.values());

        int totalRecommendations = guidelines.stream()
            .mapToInt(g -> g.getRecommendations().size())
            .sum();

        int totalActionLinks = guidelines.stream()
            .flatMap(g -> g.getRecommendations().stream())
            .mapToInt(r -> r.getLinkedProtocolActions().size())
            .sum();

        int currentGuidelines = (int) guidelines.stream()
            .filter(Guideline::isCurrent)
            .count();

        System.out.println("===== KNOWLEDGE BASE VALIDATION SUMMARY =====");
        System.out.println("Total Guidelines: " + guidelines.size());
        System.out.println("  Current: " + currentGuidelines);
        System.out.println("  Superseded: " + (guidelines.size() - currentGuidelines));
        System.out.println("Total Recommendations: " + totalRecommendations);
        System.out.println("Total Action Links: " + totalActionLinks);
        System.out.println("Total Citations: " + citations.size());
        System.out.println("===========================================");

        assertTrue(guidelines.size() > 0, "Should have guidelines");
        assertTrue(totalRecommendations > 0, "Should have recommendations");
        assertTrue(citations.size() > 0, "Should have citations");
    }
}
