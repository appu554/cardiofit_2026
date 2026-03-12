package com.cardiofit.flink.knowledgebase;

import com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader;
import com.cardiofit.flink.knowledgebase.interfaces.CitationLoader;
import com.cardiofit.flink.models.EvidenceChain;
import com.cardiofit.flink.models.ProtocolAction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.LocalDate;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Guideline Integration Service
 *
 * Links protocol actions to evidence-based guidelines, resolves evidence chains,
 * validates guideline currency, and identifies evidence gaps.
 *
 * This service provides the integration layer between clinical protocols
 * and the knowledge base of guidelines, recommendations, and citations.
 *
 * @author CardioFit Platform - Module 3 Phase 5 Day 4
 * @version 1.0
 * @since 2025-10-24
 */
public class GuidelineIntegrationService {
    private static final Logger logger = LoggerFactory.getLogger(GuidelineIntegrationService.class);

    // Injected dependencies (would be provided by framework in production)
    private final GuidelineLoader guidelineLoader;
    private final CitationLoader citationLoader;
    private final EvidenceChainResolver evidenceChainResolver;

    // Cache for performance
    private final Map<String, Guideline> guidelineCache;
    private final Map<String, List<Recommendation>> recommendationCache;

    /**
     * Constructor with dependency injection
     */
    public GuidelineIntegrationService(
        GuidelineLoader guidelineLoader,
        CitationLoader citationLoader,
        EvidenceChainResolver evidenceChainResolver
    ) {
        this.guidelineLoader = guidelineLoader;
        this.citationLoader = citationLoader;
        this.evidenceChainResolver = evidenceChainResolver;
        this.guidelineCache = new HashMap<>();
        this.recommendationCache = new HashMap<>();
    }

    /**
     * Get complete evidence chain for a protocol action
     *
     * @param actionId The protocol action ID
     * @return Complete evidence chain with guideline, recommendation, and citations
     */
    public EvidenceChain getEvidenceChain(String actionId) {
        logger.info("Resolving evidence chain for action: {}", actionId);

        try {
            // Use EvidenceChainResolver to build complete chain
            EvidenceChain chain = evidenceChainResolver.resolveChain(actionId);

            if (chain == null) {
                logger.warn("No evidence chain found for action: {}", actionId);
                return createEmptyChain(actionId);
            }

            // Calculate completeness score
            chain.calculateCompletenessScore();

            // Check for evidence gaps
            if (chain.getChainCompletenessScore() < 0.8) {
                chain.setEvidenceGapIdentified(true);
                chain.setEvidenceGapDescription(
                    "Evidence chain incomplete. Score: " + chain.getChainCompletenessScore()
                );
            }

            logger.info("Evidence chain resolved for action {}: Quality={}, Strength={}, Citations={}",
                actionId,
                chain.getEvidenceQuality(),
                chain.getRecommendationStrength(),
                chain.getCitations() != null ? chain.getCitations().size() : 0
            );

            return chain;

        } catch (Exception e) {
            logger.error("Error resolving evidence chain for action {}: {}", actionId, e.getMessage(), e);
            return createEmptyChain(actionId);
        }
    }

    /**
     * Get all guidelines supporting a specific action
     *
     * @param actionId The protocol action ID
     * @return List of guidelines that support this action
     */
    public List<Guideline> getGuidelinesForAction(String actionId) {
        logger.info("Retrieving guidelines for action: {}", actionId);

        List<Guideline> supportingGuidelines = new ArrayList<>();

        try {
            // Get evidence chain to find guideline reference
            EvidenceChain chain = getEvidenceChain(actionId);

            if (chain != null && chain.getGuidelineId() != null) {
                Guideline guideline = getGuideline(chain.getGuidelineId());
                if (guideline != null) {
                    supportingGuidelines.add(guideline);
                }
            }

            // Also search for other guidelines that might reference this action
            List<GuidelineIntegrationService.Guideline> allGuidelines = guidelineLoader.loadAllGuidelines();
            for (GuidelineIntegrationService.Guideline guideline : allGuidelines) {
                if (guidelineReferencesAction(guideline, actionId)) {
                    if (!supportingGuidelines.contains(guideline)) {
                        supportingGuidelines.add(guideline);
                    }
                }
            }

            logger.info("Found {} supporting guidelines for action: {}", supportingGuidelines.size(), actionId);

        } catch (Exception e) {
            logger.error("Error retrieving guidelines for action {}: {}", actionId, e.getMessage(), e);
        }

        return supportingGuidelines;
    }

    /**
     * Check if a guideline is current (not outdated)
     *
     * @param guidelineId The guideline ID
     * @return true if guideline is current, false if outdated or superseded
     */
    public boolean isGuidelineCurrent(String guidelineId) {
        try {
            Guideline guideline = getGuideline(guidelineId);

            if (guideline == null) {
                logger.warn("Guideline not found: {}", guidelineId);
                return false;
            }

            // Check status
            if ("OUTDATED".equalsIgnoreCase(guideline.getStatus()) ||
                "SUPERSEDED".equalsIgnoreCase(guideline.getStatus())) {
                logger.info("Guideline {} is {}", guidelineId, guideline.getStatus());
                return false;
            }

            // Check next review date
            if (guideline.getNextReviewDate() != null) {
                try {
                    LocalDate nextReview = LocalDate.parse(guideline.getNextReviewDate());
                    LocalDate now = LocalDate.now();

                    if (now.isAfter(nextReview)) {
                        logger.warn("Guideline {} is past review date: {}", guidelineId, nextReview);
                        return false;
                    }
                } catch (Exception e) {
                    logger.warn("Unable to parse next review date for guideline {}: {}",
                        guidelineId, guideline.getNextReviewDate());
                }
            }

            return true;

        } catch (Exception e) {
            logger.error("Error checking guideline currency for {}: {}", guidelineId, e.getMessage(), e);
            return false;
        }
    }

    /**
     * Get all protocol actions that lack guideline support
     *
     * @return List of action IDs without guideline evidence
     */
    public List<String> getActionsWithoutEvidence() {
        logger.info("Identifying actions without guideline evidence");

        List<String> actionsWithoutEvidence = new ArrayList<>();

        try {
            // This would typically scan all protocol actions
            // For now, return based on evidence chain completeness
            List<String> allActionIds = getAllActionIds();

            for (String actionId : allActionIds) {
                EvidenceChain chain = getEvidenceChain(actionId);

                if (chain == null ||
                    chain.getGuidelineId() == null ||
                    chain.getRecommendationId() == null ||
                    chain.getCitations() == null ||
                    chain.getCitations().isEmpty()) {

                    actionsWithoutEvidence.add(actionId);
                    logger.warn("Action {} lacks complete evidence support", actionId);
                }
            }

            logger.info("Found {} actions without complete evidence", actionsWithoutEvidence.size());

        } catch (Exception e) {
            logger.error("Error identifying actions without evidence: {}", e.getMessage(), e);
        }

        return actionsWithoutEvidence;
    }

    /**
     * Get quality badge for an action based on evidence strength
     *
     * @param actionId The protocol action ID
     * @return Quality badge string (🟢 STRONG, 🟡 MODERATE, 🟠 WEAK, ⚠️ OUTDATED)
     */
    public String getQualityBadge(String actionId) {
        try {
            EvidenceChain chain = getEvidenceChain(actionId);

            if (chain == null) {
                return "⚪ UNGRADED";
            }

            // Check if guideline is outdated
            if (chain.getGuidelineId() != null && !isGuidelineCurrent(chain.getGuidelineId())) {
                return "⚠️ OUTDATED";
            }

            return chain.getQualityBadge();

        } catch (Exception e) {
            logger.error("Error getting quality badge for action {}: {}", actionId, e.getMessage(), e);
            return "⚪ ERROR";
        }
    }

    /**
     * Enrich a protocol action with guideline integration data
     *
     * @param action The protocol action to enrich
     * @return Enriched protocol action with evidence chain
     */
    public ProtocolAction enrichActionWithEvidence(ProtocolAction action) {
        if (action == null || action.getActionId() == null) {
            logger.warn("Cannot enrich null action or action without ID");
            return action;
        }

        logger.info("Enriching action {} with evidence data", action.getActionId());

        try {
            // Get evidence chain
            EvidenceChain chain = getEvidenceChain(action.getActionId());

            if (chain != null) {
                // Set evidence chain
                action.setEvidenceChain(chain);

                // Set guideline reference fields
                action.setGuidelineReference(chain.getGuidelineId());
                action.setRecommendationId(chain.getRecommendationId());
                action.setEvidenceQuality(chain.getEvidenceQuality());
                action.setRecommendationStrength(chain.getRecommendationStrength());
                action.setClassOfRecommendation(chain.getClassOfRecommendation());
                action.setLevelOfEvidence(chain.getLevelOfEvidence());
                action.setClinicalRationale(chain.getClinicalRationale());

                // Set citation PMIDs
                if (chain.getKeyEvidencePmids() != null) {
                    action.setCitationPmids(chain.getKeyEvidencePmids());
                }

                logger.info("Action {} enriched with evidence: Quality={}, Strength={}",
                    action.getActionId(),
                    action.getEvidenceQuality(),
                    action.getRecommendationStrength()
                );
            } else {
                logger.warn("No evidence chain available for action: {}", action.getActionId());
            }

        } catch (Exception e) {
            logger.error("Error enriching action {} with evidence: {}",
                action.getActionId(), e.getMessage(), e);
        }

        return action;
    }

    /**
     * Generate evidence gap report for all actions
     *
     * @return Map of action IDs to evidence gap descriptions
     */
    public Map<String, String> generateEvidenceGapReport() {
        logger.info("Generating evidence gap report");

        Map<String, String> gapReport = new HashMap<>();

        try {
            List<String> allActionIds = getAllActionIds();

            for (String actionId : allActionIds) {
                EvidenceChain chain = getEvidenceChain(actionId);

                if (chain != null && chain.getEvidenceGapIdentified() != null &&
                    chain.getEvidenceGapIdentified()) {

                    gapReport.put(actionId, chain.getEvidenceGapDescription());
                }
            }

            logger.info("Evidence gap report generated: {} gaps identified", gapReport.size());

        } catch (Exception e) {
            logger.error("Error generating evidence gap report: {}", e.getMessage(), e);
        }

        return gapReport;
    }

    // Private helper methods

    /**
     * Get guideline by ID (with caching)
     */
    private Guideline getGuideline(String guidelineId) {
        if (guidelineCache.containsKey(guidelineId)) {
            return guidelineCache.get(guidelineId);
        }

        GuidelineIntegrationService.Guideline guideline = guidelineLoader.loadGuideline(guidelineId);
        if (guideline != null) {
            guidelineCache.put(guidelineId, guideline);
        }

        return guideline;
    }

    /**
     * Get all guidelines from the guideline loader.
     * This is a convenience method for testing and batch operations.
     *
     * @return List of all guidelines loaded from the guideline database
     */
    public List<Guideline> getAllGuidelines() {
        logger.info("Retrieving all guidelines");
        try {
            return guidelineLoader.loadAllGuidelines();
        } catch (Exception e) {
            logger.error("Error loading all guidelines", e);
            return new ArrayList<>();
        }
    }

    /**
     * Check if guideline references a specific action
     */
    private boolean guidelineReferencesAction(Guideline guideline, String actionId) {
        if (guideline.getRecommendations() == null) {
            return false;
        }

        for (Recommendation rec : guideline.getRecommendations()) {
            if (rec.getLinkedProtocolActions() != null &&
                rec.getLinkedProtocolActions().contains(actionId)) {
                return true;
            }
        }

        return false;
    }

    /**
     * Get all action IDs from protocols
     * This would typically come from a protocol registry
     */
    private List<String> getAllActionIds() {
        // Placeholder - would be implemented with actual protocol registry
        // For now, return empty list
        return new ArrayList<>();
    }

    /**
     * Create empty evidence chain for actions without evidence
     */
    private EvidenceChain createEmptyChain(String actionId) {
        EvidenceChain chain = new EvidenceChain();
        chain.setActionId(actionId);
        chain.setEvidenceGapIdentified(true);
        chain.setEvidenceGapDescription("No guideline evidence found for this action");
        chain.setChainCompletenessScore(0.0);
        return chain;
    }

    /**
     * Assess overall evidence quality from a list of citations.
     *
     * Uses GRADE-like methodology:
     * - Multiple RCTs or Meta-analysis → High
     * - Single RCT or multiple cohort studies → Moderate
     * - Single cohort or case-control → Low
     * - Case series, expert opinion → Very Low
     *
     * @param citations List of supporting citations
     * @return Quality rating (High, Moderate, Low, Very Low)
     */
    public String assessOverallQuality(List<com.cardiofit.flink.knowledgebase.Citation> citations) {
        if (citations == null || citations.isEmpty()) {
            return "Very Low";
        }

        int metaAnalysisCount = 0;
        int rctCount = 0;
        int cohortCount = 0;
        int caseControlCount = 0;

        for (com.cardiofit.flink.knowledgebase.Citation citation : citations) {
            String studyType = citation.getStudyType();
            if (studyType == null) continue;

            switch (studyType.toUpperCase()) {
                case "META_ANALYSIS":
                    metaAnalysisCount++;
                    break;
                case "RCT":
                    rctCount++;
                    break;
                case "COHORT":
                    cohortCount++;
                    break;
                case "CASE_CONTROL":
                    caseControlCount++;
                    break;
            }
        }

        // Quality assessment logic
        if (metaAnalysisCount > 0) {
            return "High";  // Meta-analysis is highest quality
        } else if (rctCount >= 2) {
            return "High";  // Multiple RCTs
        } else if (rctCount == 1) {
            return "Moderate";  // Single RCT
        } else if (cohortCount >= 2) {
            return "Moderate";  // Multiple cohort studies
        } else if (cohortCount == 1 || caseControlCount >= 1) {
            return "Low";  // Single cohort or case-control
        } else {
            return "Very Low";  // Case series, expert opinion, or unknown
        }
    }

    // Nested classes for guideline data structures

    /**
     * Guideline data structure
     */
    public static class Guideline {
        private String guidelineId;
        private String name;
        private String organization;
        private String publicationDate;
        private String nextReviewDate;
        private String status;
        private List<Recommendation> recommendations;

        // Getters and setters
        public String getGuidelineId() { return guidelineId; }
        public void setGuidelineId(String guidelineId) { this.guidelineId = guidelineId; }

        public String getName() { return name; }
        public void setName(String name) { this.name = name; }

        public String getOrganization() { return organization; }
        public void setOrganization(String organization) { this.organization = organization; }

        public String getPublicationDate() { return publicationDate; }
        public void setPublicationDate(String publicationDate) { this.publicationDate = publicationDate; }

        public String getNextReviewDate() { return nextReviewDate; }
        public void setNextReviewDate(String nextReviewDate) { this.nextReviewDate = nextReviewDate; }

        public String getStatus() { return status; }
        public void setStatus(String status) { this.status = status; }

        public List<Recommendation> getRecommendations() { return recommendations; }
        public void setRecommendations(List<Recommendation> recommendations) {
            this.recommendations = recommendations;
        }
    }

    /**
     * Recommendation data structure
     */
    public static class Recommendation {
        private String recommendationId;
        private String statement;
        private String strength;
        private String evidenceQuality;
        private String classOfRecommendation;  // CLASS_I, CLASS_IIA, CLASS_IIB, CLASS_III
        private String levelOfEvidence;  // A, B-R, B-NR, C-LD, C-EO
        private List<String> linkedProtocolActions;
        private List<String> citationPmids;

        // Getters and setters
        public String getRecommendationId() { return recommendationId; }
        public void setRecommendationId(String recommendationId) { this.recommendationId = recommendationId; }

        public String getStatement() { return statement; }
        public void setStatement(String statement) { this.statement = statement; }

        public String getStrength() { return strength; }
        public void setStrength(String strength) { this.strength = strength; }

        public String getEvidenceQuality() { return evidenceQuality; }
        public void setEvidenceQuality(String evidenceQuality) { this.evidenceQuality = evidenceQuality; }

        public String getClassOfRecommendation() { return classOfRecommendation; }
        public void setClassOfRecommendation(String classOfRecommendation) {
            this.classOfRecommendation = classOfRecommendation;
        }

        public String getLevelOfEvidence() { return levelOfEvidence; }
        public void setLevelOfEvidence(String levelOfEvidence) { this.levelOfEvidence = levelOfEvidence; }

        public List<String> getLinkedProtocolActions() { return linkedProtocolActions; }
        public void setLinkedProtocolActions(List<String> linkedProtocolActions) {
            this.linkedProtocolActions = linkedProtocolActions;
        }

        public List<String> getCitationPmids() { return citationPmids; }
        public void setCitationPmids(List<String> citationPmids) { this.citationPmids = citationPmids; }

        /**
         * Alias for getCitationPmids() for compatibility with EvidenceChain
         */
        public List<String> getKeyEvidence() {
            return getCitationPmids();
        }

        /**
         * Alias for setCitationPmids() for compatibility with EvidenceChain
         */
        public void setKeyEvidence(List<String> keyEvidence) {
            setCitationPmids(keyEvidence);
        }
    }
}
