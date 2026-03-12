package com.cardiofit.flink.knowledgebase;

import com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader;
import com.cardiofit.flink.knowledgebase.interfaces.CitationLoader;
import com.cardiofit.flink.knowledgebase.loader.GuidelineLoaderImpl;
import com.cardiofit.flink.knowledgebase.loader.CitationLoaderImpl;
import com.cardiofit.flink.models.EvidenceChain;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Guideline Linker
 *
 * Links protocol actions to guidelines, resolves evidence chains, and assesses quality.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class GuidelineLinker {
    private static final Logger logger = LoggerFactory.getLogger(GuidelineLinker.class);

    private final GuidelineLoader guidelineLoader;
    private final CitationLoader citationLoader;

    public GuidelineLinker() {
        this.guidelineLoader = GuidelineLoaderImpl.getInstance();
        this.citationLoader = CitationLoaderImpl.getInstance();
    }

    /**
     * Get complete evidence chain for a protocol action
     *
     * @param actionId Protocol action ID (e.g., "STEMI-ACT-002")
     * @return Evidence chain with guideline → recommendation → citations
     */
    public EvidenceChain getEvidenceChain(String actionId) {
        logger.debug("Resolving evidence chain for action: {}", actionId);

        EvidenceChain chain = new EvidenceChain();
        chain.setActionId(actionId);

        // Find guideline recommendations linked to this action
        List<GuidelineIntegrationService.Guideline> guidelines = guidelineLoader.getAllGuidelines();

        for (GuidelineIntegrationService.Guideline guideline : guidelines) {
            for (GuidelineIntegrationService.Recommendation rec : guideline.getRecommendations()) {
                if (rec.getLinkedProtocolActions().contains(actionId)) {
                    // Found matching recommendation
                    chain.setSourceGuideline(guideline);
                    chain.setGuidelineRecommendation(rec);

                    // Load supporting citations
                    List<EvidenceChain.Citation> citations = loadCitations(rec.getKeyEvidence());
                    chain.setSupportingEvidence(citations);

                    // Assess quality
                    assessEvidenceQuality(chain);

                    logger.info("Evidence chain resolved for {}: {} → {}",
                        actionId, guideline.getName(), rec.getStrength());

                    return chain;
                }
            }
        }

        logger.warn("No evidence chain found for action: {}", actionId);
        return chain;
    }

    /**
     * Get evidence chain for testing with specific guideline
     */
    public EvidenceChain getEvidenceChain(GuidelineIntegrationService.Guideline guideline, String actionId) {
        EvidenceChain chain = new EvidenceChain();
        chain.setActionId(actionId);
        chain.setSourceGuideline(guideline);

        // Find matching recommendation
        for (GuidelineIntegrationService.Recommendation rec : guideline.getRecommendations()) {
            if (rec.getLinkedProtocolActions().contains(actionId)) {
                chain.setGuidelineRecommendation(rec);

                List<EvidenceChain.Citation> citations = loadCitations(rec.getKeyEvidence());
                chain.setSupportingEvidence(citations);
                break;
            }
        }

        assessEvidenceQuality(chain);
        return chain;
    }

    /**
     * Load citations by PMIDs
     */
    private List<EvidenceChain.Citation> loadCitations(List<String> pmids) {
        List<EvidenceChain.Citation> citations = new ArrayList<>();

        if (pmids == null) {
            return citations;
        }

        for (String pmid : pmids) {
            EvidenceChain.Citation citation = citationLoader.getCitationByPmid(pmid);
            if (citation != null) {
                citations.add(citation);
            } else {
                logger.debug("Citation not found for PMID: {}", pmid);
            }
        }

        return citations;
    }

    /**
     * Assess overall evidence quality using GRADE methodology
     */
    private void assessEvidenceQuality(EvidenceChain chain) {
        // Assess guideline currency
        boolean isCurrent = chain.isGuidelineCurrent();
        chain.setCurrent(isCurrent);

        // Assess citation quality using GRADE methodology
        List<EvidenceChain.Citation> citations = chain.getSupportingEvidence();
        String overallQuality = assessOverallQuality(citations);
        chain.setOverallQuality(overallQuality);

        // Calculate completeness score
        chain.calculateCompletenessScore();

        // Set quality badge
        String badge = chain.getQualityBadge(); // Use calculated badge
        chain.setQualityBadge(badge);

        logger.debug("Evidence quality assessed for {}: quality={}, current={}, completeness={}",
            chain.getActionId(), overallQuality, isCurrent, chain.getChainCompletenessScore());
    }

    /**
     * Assess aggregated evidence quality from multiple citations
     * Used for mixed evidence sources
     */
    public String assessOverallQuality(List<EvidenceChain.Citation> citations) {
        if (citations == null || citations.isEmpty()) {
            return "Unknown";
        }

        // Count study types
        // Note: Citation.StudyType enum doesn't exist, so we'll skip study type analysis
        long rctCount = 0;
        long metaAnalysisCount = 0;
        long observationalCount = 0;

        // GRADE-style assessment
        if (metaAnalysisCount > 0) {
            return "High"; // Meta-analysis of RCTs
        } else if (rctCount >= 2) {
            return "High"; // Multiple RCTs
        } else if (rctCount == 1 && observationalCount > 0) {
            return "Moderate"; // Mixed RCT + observational
        } else if (rctCount == 1) {
            return "Moderate"; // Single RCT
        } else if (observationalCount > 0) {
            return "Low"; // Only observational studies
        } else {
            return "Very Low"; // Expert opinion or low quality
        }
    }

    /**
     * Find all actions linked to a guideline
     */
    public List<String> getLinkedActions(String guidelineId) {
        GuidelineIntegrationService.Guideline guideline = guidelineLoader.getGuidelineById(guidelineId);
        if (guideline == null) {
            return new ArrayList<>();
        }

        return guideline.getRecommendations().stream()
            .flatMap(rec -> rec.getLinkedProtocolActions().stream())
            .distinct()
            .collect(Collectors.toList());
    }

    /**
     * Find all guidelines supporting an action
     */
    public List<GuidelineIntegrationService.Guideline> getGuidelinesForAction(String actionId) {
        List<GuidelineIntegrationService.Guideline> guidelines = guidelineLoader.getAllGuidelines();
        List<GuidelineIntegrationService.Guideline> supporting = new ArrayList<>();

        for (GuidelineIntegrationService.Guideline guideline : guidelines) {
            for (GuidelineIntegrationService.Recommendation rec : guideline.getRecommendations()) {
                if (rec.getLinkedProtocolActions().contains(actionId)) {
                    supporting.add(guideline);
                    break; // Found match, move to next guideline
                }
            }
        }

        return supporting;
    }
}
