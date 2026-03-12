package com.cardiofit.flink.knowledgebase;

import com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader;
import com.cardiofit.flink.knowledgebase.interfaces.CitationLoader;
import com.cardiofit.flink.models.EvidenceChain;
import com.cardiofit.flink.models.ProtocolAction;

/**
 * Guideline Integration Example
 *
 * Demonstrates complete evidence chain integration from protocol actions
 * through guidelines to supporting citations.
 *
 * This example shows:
 * 1. How to resolve evidence chains for protocol actions
 * 2. How to display complete evidence trails
 * 3. How to assess evidence quality and currency
 * 4. How to identify evidence gaps
 *
 * @author CardioFit Platform - Module 3 Phase 5 Day 4
 * @version 1.0
 * @since 2025-10-24
 */
public class GuidelineIntegrationExample {

    public static void main(String[] args) {
        System.out.println("=============================================================");
        System.out.println("GUIDELINE INTEGRATION EXAMPLE");
        System.out.println("Phase 5 Day 4: Protocol-Guideline-Citation Integration");
        System.out.println("=============================================================\n");

        // Initialize services (in production, these would be injected)
        com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader guidelineLoader = new MockGuidelineLoader();
        com.cardiofit.flink.knowledgebase.interfaces.CitationLoader citationLoader = new MockCitationLoader();
        EvidenceChainResolver resolver = new EvidenceChainResolver(guidelineLoader, citationLoader);
        GuidelineIntegrationService integrationService = new GuidelineIntegrationService(
            guidelineLoader,
            citationLoader,
            resolver
        );

        // Example 1: STEMI Aspirin Action
        System.out.println("EXAMPLE 1: STEMI Aspirin (STEMI-ACT-002)");
        System.out.println("=========================================\n");
        demonstrateEvidenceChain(integrationService, resolver, "STEMI-ACT-002");

        System.out.println("\n\n");

        // Example 2: Sepsis Antibiotic Action
        System.out.println("EXAMPLE 2: Sepsis Antibiotics (SEPSIS-ACT-004)");
        System.out.println("===============================================\n");
        demonstrateEvidenceChain(integrationService, resolver, "SEPSIS-ACT-004");

        System.out.println("\n\n");

        // Example 3: Guideline Currency Check
        System.out.println("EXAMPLE 3: Guideline Currency Assessment");
        System.out.println("=========================================\n");
        demonstrateGuidelineCurrency(integrationService);

        System.out.println("\n\n");

        // Example 4: Evidence Gap Analysis
        System.out.println("EXAMPLE 4: Evidence Gap Identification");
        System.out.println("========================================\n");
        demonstrateEvidenceGapAnalysis(integrationService);

        System.out.println("\n\n");

        // Example 5: Protocol Action Enrichment
        System.out.println("EXAMPLE 5: Protocol Action Enrichment with Evidence");
        System.out.println("====================================================\n");
        demonstrateActionEnrichment(integrationService);

        System.out.println("\n=============================================================");
        System.out.println("INTEGRATION EXAMPLES COMPLETE");
        System.out.println("=============================================================");
    }

    /**
     * Demonstrate complete evidence chain resolution
     */
    private static void demonstrateEvidenceChain(
        GuidelineIntegrationService service,
        EvidenceChainResolver resolver,
        String actionId
    ) {
        // Get evidence chain
        EvidenceChain chain = service.getEvidenceChain(actionId);

        if (chain != null) {
            // Display formatted evidence trail
            System.out.println("📋 FORMATTED EVIDENCE TRAIL:");
            System.out.println(chain.getFormattedEvidenceTrail());

            System.out.println("\n📊 EVIDENCE QUALITY METRICS:");
            System.out.println("  • Completeness Score: " +
                String.format("%.1f%%", chain.getChainCompletenessScore() * 100));
            System.out.println("  • Evidence Quality: " + chain.getEvidenceQuality());
            System.out.println("  • Recommendation Strength: " + chain.getRecommendationStrength());
            System.out.println("  • Class of Recommendation: " + chain.getClassOfRecommendation());
            System.out.println("  • Level of Evidence: " + chain.getLevelOfEvidence());
            System.out.println("  • Citation Count: " +
                (chain.getCitations() != null ? chain.getCitations().size() : 0));
            System.out.println("  • Guideline Status: " +
                (chain.isGuidelineCurrent() ? "✅ CURRENT" : "⚠️ OUTDATED"));
            System.out.println("  • Quality Badge: " + chain.getQualityBadge());

            if (chain.getEvidenceGapIdentified() != null && chain.getEvidenceGapIdentified()) {
                System.out.println("\n⚠️  EVIDENCE GAP IDENTIFIED:");
                System.out.println("  " + chain.getEvidenceGapDescription());
            }

            // Display compact summary
            System.out.println("\n📝 COMPACT SUMMARY:");
            System.out.println("  " + resolver.getEvidenceSummary(actionId));

        } else {
            System.out.println("❌ No evidence chain found for action: " + actionId);
        }
    }

    /**
     * Demonstrate guideline currency checking
     */
    private static void demonstrateGuidelineCurrency(GuidelineIntegrationService service) {
        String[] guidelinesToCheck = {
            "GUIDE-ACCAHA-STEMI-2023",
            "GUIDE-SSC-2021",
            "GUIDE-ACCAHA-STEMI-2013"  // Superseded guideline
        };

        for (String guidelineId : guidelinesToCheck) {
            boolean isCurrent = service.isGuidelineCurrent(guidelineId);
            String status = isCurrent ? "✅ CURRENT" : "⚠️ OUTDATED";
            System.out.println("  " + guidelineId + ": " + status);
        }
    }

    /**
     * Demonstrate evidence gap analysis
     */
    private static void demonstrateEvidenceGapAnalysis(GuidelineIntegrationService service) {
        // Get actions without complete evidence
        java.util.List<String> actionsWithoutEvidence = service.getActionsWithoutEvidence();

        System.out.println("  Actions lacking complete evidence: " + actionsWithoutEvidence.size());

        if (!actionsWithoutEvidence.isEmpty()) {
            System.out.println("\n  ⚠️  Actions requiring evidence update:");
            for (String actionId : actionsWithoutEvidence) {
                String badge = service.getQualityBadge(actionId);
                System.out.println("    • " + actionId + " - " + badge);
            }
        } else {
            System.out.println("  ✅ All actions have complete evidence support!");
        }

        // Generate detailed gap report
        java.util.Map<String, String> gapReport = service.generateEvidenceGapReport();
        if (!gapReport.isEmpty()) {
            System.out.println("\n  📋 DETAILED GAP REPORT:");
            for (java.util.Map.Entry<String, String> entry : gapReport.entrySet()) {
                System.out.println("    " + entry.getKey() + ":");
                System.out.println("      → " + entry.getValue());
            }
        }
    }

    /**
     * Demonstrate protocol action enrichment with evidence
     */
    private static void demonstrateActionEnrichment(GuidelineIntegrationService service) {
        // Create sample protocol action
        ProtocolAction action = new ProtocolAction();
        action.setActionId("STEMI-ACT-002");
        action.setActionType("MEDICATION");
        action.setDescription("Aspirin 324 mg PO (chewable)");
        action.setPriority("CRITICAL");

        System.out.println("BEFORE ENRICHMENT:");
        System.out.println("  Action ID: " + action.getActionId());
        System.out.println("  Guideline Reference: " +
            (action.getGuidelineReference() != null ? action.getGuidelineReference() : "NONE"));
        System.out.println("  Evidence Quality: " +
            (action.getEvidenceQuality() != null ? action.getEvidenceQuality() : "NONE"));

        // Enrich with evidence
        action = service.enrichActionWithEvidence(action);

        System.out.println("\nAFTER ENRICHMENT:");
        System.out.println("  Action ID: " + action.getActionId());
        System.out.println("  Guideline Reference: " + action.getGuidelineReference());
        System.out.println("  Recommendation ID: " + action.getRecommendationId());
        System.out.println("  Evidence Quality: " + action.getEvidenceQuality());
        System.out.println("  Recommendation Strength: " + action.getRecommendationStrength());
        System.out.println("  Class of Recommendation: " + action.getClassOfRecommendation());
        System.out.println("  Level of Evidence: " + action.getLevelOfEvidence());
        System.out.println("  Quality Badge: " + action.getQualityBadge());
        System.out.println("  Citation Count: " +
            (action.getCitationPmids() != null ? action.getCitationPmids().size() : 0));

        System.out.println("\n  📝 EVIDENCE SUMMARY:");
        System.out.println(action.getEvidenceSummary());

        System.out.println("\n  ✅ Evidence Chain Completeness: " +
            (action.getEvidenceChain() != null ?
                String.format("%.1f%%", action.getEvidenceChain().getChainCompletenessScore() * 100) :
                "N/A"));
    }

    /**
     * Mock Guideline Loader for demonstration
     */
    private static class MockGuidelineLoader implements com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader {
        @Override
        public GuidelineIntegrationService.Guideline loadGuideline(String guidelineId) {
            GuidelineIntegrationService.Guideline guideline = new GuidelineIntegrationService.Guideline();
            guideline.setGuidelineId(guidelineId);

            if ("GUIDE-ACCAHA-STEMI-2023".equals(guidelineId)) {
                guideline.setName("2023 ACC/AHA/SCAI STEMI Guideline");
                guideline.setOrganization("ACC/AHA/SCAI");
                guideline.setPublicationDate("2023-04-20");
                guideline.setNextReviewDate("2028-04-20");
                guideline.setStatus("CURRENT");
            } else if ("GUIDE-SSC-2021".equals(guidelineId)) {
                guideline.setName("Surviving Sepsis Campaign 2021");
                guideline.setOrganization("Surviving Sepsis Campaign");
                guideline.setPublicationDate("2021-11-01");
                guideline.setNextReviewDate("2026-11-01");
                guideline.setStatus("CURRENT");
            }

            return guideline;
        }

        @Override
        public GuidelineIntegrationService.Recommendation loadRecommendation(
            String guidelineId,
            String recommendationId
        ) {
            GuidelineIntegrationService.Recommendation rec = new GuidelineIntegrationService.Recommendation();
            rec.setRecommendationId(recommendationId);

            if ("ACC-STEMI-2023-REC-003".equals(recommendationId)) {
                rec.setStatement("Aspirin 162-325 mg should be given immediately to all STEMI patients");
                rec.setStrength("STRONG");
                rec.setEvidenceQuality("HIGH");
                rec.setCitationPmids(java.util.Arrays.asList("37079885", "3081859", "18160631"));
            } else if ("SSC-2021-REC-004".equals(recommendationId)) {
                rec.setStatement("Broad-spectrum antibiotics within 1 hour of sepsis recognition");
                rec.setStrength("STRONG");
                rec.setEvidenceQuality("HIGH");
                rec.setCitationPmids(java.util.Arrays.asList("34599691", "16625125"));
            }

            return rec;
        }

        @Override
        public java.util.List<GuidelineIntegrationService.Guideline> loadAllGuidelines() {
            return java.util.Arrays.asList(
                loadGuideline("GUIDE-ACCAHA-STEMI-2023"),
                loadGuideline("GUIDE-SSC-2021")
            );
        }
    }

    /**
     * Mock Citation Loader for demonstration
     */
    private static class MockCitationLoader implements com.cardiofit.flink.knowledgebase.interfaces.CitationLoader {
        @Override
        public EvidenceChain.Citation loadCitation(String pmid) {
            EvidenceChain.Citation citation = new EvidenceChain.Citation();
            citation.setPmid(pmid);

            if ("3081859".equals(pmid)) {
                citation.setAuthors("ISIS-2 Collaborative Group");
                citation.setTitle("Randomised trial of aspirin in acute myocardial infarction");
                citation.setJournal("Lancet");
                citation.setYear(1988);
                citation.setCitationSummary("23% mortality reduction with aspirin (RCT, n=17,187)");
            } else if ("16625125".equals(pmid)) {
                citation.setAuthors("Kumar A, Roberts D, Wood KE, et al.");
                citation.setTitle("Duration of hypotension before antimicrobial therapy in septic shock");
                citation.setJournal("Critical Care Medicine");
                citation.setYear(2006);
                citation.setCitationSummary("7.6% mortality increase per hour antibiotic delay");
            } else if ("37079885".equals(pmid)) {
                citation.setAuthors("O'Gara PT, Kushner FG, et al.");
                citation.setTitle("2023 ACC/AHA STEMI Guideline Update");
                citation.setJournal("JACC");
                citation.setYear(2023);
                citation.setCitationSummary("Updated guideline recommendations for STEMI management");
            } else if ("34599691".equals(pmid)) {
                citation.setAuthors("Evans L, Rhodes A, Alhazzani W, et al.");
                citation.setTitle("Surviving Sepsis Campaign Guidelines 2021");
                citation.setJournal("Intensive Care Medicine");
                citation.setYear(2021);
                citation.setCitationSummary("International consensus guidelines for sepsis management");
            }

            return citation;
        }
    }
}
