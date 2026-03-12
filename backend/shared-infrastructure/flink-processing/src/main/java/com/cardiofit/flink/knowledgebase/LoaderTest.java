package com.cardiofit.flink.knowledgebase;

/**
 * Simple test class to verify GuidelineLoader and CitationLoader functionality
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class LoaderTest {

    public static void main(String[] args) {
        System.out.println("=".repeat(60));
        System.out.println("Testing GuidelineLoader and CitationLoader");
        System.out.println("=".repeat(60));

        // Test GuidelineLoader
        testGuidelineLoader();

        // Test CitationLoader
        testCitationLoader();

        System.out.println("\n" + "=".repeat(60));
        System.out.println("All tests completed!");
        System.out.println("=".repeat(60));
    }

    private static void testGuidelineLoader() {
        System.out.println("\n--- GuidelineLoader Test ---");

        GuidelineLoader loader = GuidelineLoader.getInstance();
        System.out.println("✓ GuidelineLoader singleton instance created");

        // Load all guidelines
        loader.loadAllGuidelines();
        System.out.println("✓ Attempted to load guidelines from knowledge base");

        // Get cache stats
        System.out.println("✓ Cache stats: " + loader.getCacheStats());

        // Test specific guideline retrieval (if any loaded)
        int count = loader.getGuidelineCount();
        System.out.println("✓ Total guidelines cached: " + count);

        if (count > 0) {
            System.out.println("✓ Sample guidelines:");
            loader.getCurrentGuidelines().stream()
                .limit(3)
                .forEach(g -> System.out.println("  - " + g.getShortName() + " (" + g.getGuidelineId() + ")"));
        }
    }

    private static void testCitationLoader() {
        System.out.println("\n--- CitationLoader Test ---");

        CitationLoader loader = CitationLoader.getInstance();
        System.out.println("✓ CitationLoader singleton instance created");

        // Load all citations
        loader.loadAllCitations();
        System.out.println("✓ Attempted to load citations from knowledge base");

        // Get cache stats
        System.out.println("✓ Cache stats: " + loader.getCacheStats());

        // Test specific citation retrieval (if any loaded)
        int count = loader.getCitationCount();
        System.out.println("✓ Total citations cached: " + count);

        if (count > 0) {
            System.out.println("✓ Sample citations:");
            loader.getHighQualityCitations().stream()
                .limit(3)
                .forEach(c -> System.out.println("  - PMID " + c.getPmid() + ": " + c.getShortCitation()));
        }
    }
}
