package com.cardiofit.flink.knowledgebase;

import com.cardiofit.flink.knowledgebase.interfaces.GuidelineLoader;
import com.cardiofit.flink.knowledgebase.interfaces.CitationLoader;
import com.cardiofit.flink.knowledgebase.loader.GuidelineLoaderImpl;
import com.cardiofit.flink.knowledgebase.loader.CitationLoaderImpl;
import com.cardiofit.flink.models.EvidenceChain;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;
import java.util.stream.Collectors;

/**
 * Evidence Chain Resolver
 *
 * Resolves complete evidence chains from protocol actions through guidelines
 * to supporting citations. Aggregates evidence across multiple guidelines,
 * assesses overall evidence quality using GRADE methodology, and generates
 * formatted evidence trails for UI display.
 *
 * Core responsibilities:
 * - Build complete evidence chains: Action → Guideline → Recommendation → Citations
 * - Aggregate evidence from multiple sources
 * - Apply GRADE evidence quality assessment
 * - Generate formatted evidence trails for UI
 *
 * @author CardioFit Platform - Module 3 Phase 5 Day 4
 * @version 1.0
 * @since 2025-10-24
 */
public class EvidenceChainResolver {
    private static final Logger logger = LoggerFactory.getLogger(EvidenceChainResolver.class);

    // Injected dependencies
    private final GuidelineLoader guidelineLoader;
    private final CitationLoader citationLoader;
    private final Map<String, ActionGuidelineMapping> actionMappings;

    /**
     * Constructor with dependency injection
     */
    public EvidenceChainResolver(
        GuidelineLoader guidelineLoader,
        CitationLoader citationLoader
    ) {
        this.guidelineLoader = guidelineLoader;
        this.citationLoader = citationLoader;
        this.actionMappings = new HashMap<>();
        initializeActionMappings();
    }

    /**
     * Resolve complete evidence chain for a protocol action
     *
     * @param actionId The protocol action ID
     * @return Complete evidence chain with all linkages
     */
    public EvidenceChain resolveChain(String actionId) {
        logger.info("Resolving complete evidence chain for action: {}", actionId);

        try {
            // Get action-guideline mapping
            ActionGuidelineMapping mapping = actionMappings.get(actionId);

            if (mapping == null) {
                logger.warn("No guideline mapping found for action: {}", actionId);
                return null;
            }

            // Build evidence chain
            EvidenceChain chain = new EvidenceChain(
                actionId,
                mapping.getGuidelineId(),
                mapping.getRecommendationId()
            );

            // Populate guideline information
            populateGuidelineInfo(chain, mapping.getGuidelineId());

            // Populate recommendation information
            populateRecommendationInfo(chain, mapping.getGuidelineId(), mapping.getRecommendationId());

            // Populate citations
            populateCitations(chain, mapping.getCitationPmids());

            // Aggregate evidence quality
            aggregateEvidenceQuality(chain);

            // Calculate completeness
            chain.calculateCompletenessScore();

            logger.info("Evidence chain resolved: Action={}, Guideline={}, Rec={}, Citations={}",
                actionId,
                chain.getGuidelineId(),
                chain.getRecommendationId(),
                chain.getCitations() != null ? chain.getCitations().size() : 0
            );

            return chain;

        } catch (Exception e) {
            logger.error("Error resolving evidence chain for action {}: {}", actionId, e.getMessage(), e);
            return null;
        }
    }

    /**
     * Resolve evidence chains for multiple actions
     *
     * @param actionIds List of protocol action IDs
     * @return Map of action IDs to evidence chains
     */
    public Map<String, EvidenceChain> resolveChains(List<String> actionIds) {
        logger.info("Resolving evidence chains for {} actions", actionIds.size());

        Map<String, EvidenceChain> chains = new HashMap<>();

        for (String actionId : actionIds) {
            EvidenceChain chain = resolveChain(actionId);
            if (chain != null) {
                chains.put(actionId, chain);
            }
        }

        logger.info("Resolved {} evidence chains", chains.size());
        return chains;
    }

    /**
     * Aggregate citations across multiple guidelines supporting the same action
     *
     * @param actionId The protocol action ID
     * @return Aggregated list of unique citations
     */
    public List<EvidenceChain.Citation> aggregateCitations(String actionId) {
        logger.info("Aggregating citations for action: {}", actionId);

        Set<String> uniquePmids = new HashSet<>();
        List<EvidenceChain.Citation> aggregatedCitations = new ArrayList<>();

        try {
            // Get all guidelines supporting this action
            List<ActionGuidelineMapping> mappings = getGuidelineMappingsForAction(actionId);

            for (ActionGuidelineMapping mapping : mappings) {
                if (mapping.getCitationPmids() != null) {
                    for (String pmid : mapping.getCitationPmids()) {
                        if (!uniquePmids.contains(pmid)) {
                            EvidenceChain.Citation citation = citationLoader.loadCitation(pmid);
                            if (citation != null) {
                                aggregatedCitations.add(citation);
                                uniquePmids.add(pmid);
                            }
                        }
                    }
                }
            }

            logger.info("Aggregated {} unique citations for action: {}", aggregatedCitations.size(), actionId);

        } catch (Exception e) {
            logger.error("Error aggregating citations for action {}: {}", actionId, e.getMessage(), e);
        }

        return aggregatedCitations;
    }

    /**
     * Assess overall evidence quality using GRADE methodology
     *
     * @param citations List of supporting citations
     * @return GRADE quality assessment (HIGH, MODERATE, LOW, VERY_LOW)
     */
    public String assessEvidenceQuality(List<EvidenceChain.Citation> citations) {
        if (citations == null || citations.isEmpty()) {
            return "VERY_LOW";
        }

        // GRADE methodology factors:
        // 1. Study design (RCTs start HIGH, observational start LOW)
        // 2. Risk of bias
        // 3. Inconsistency
        // 4. Indirectness
        // 5. Imprecision

        int highQualityCount = 0;
        int rctCount = 0;
        int totalCitations = citations.size();

        for (EvidenceChain.Citation citation : citations) {
            // Simplified assessment based on citation metadata
            // In production, this would use full GRADE assessment
            if (citation.getCitationSummary() != null) {
                String summary = citation.getCitationSummary().toLowerCase();
                if (summary.contains("randomized") || summary.contains("rct")) {
                    rctCount++;
                }
                if (summary.contains("high quality") || summary.contains("large")) {
                    highQualityCount++;
                }
            }
        }

        // Simple scoring algorithm
        if (rctCount >= 2 && highQualityCount >= 1) {
            return "HIGH";
        } else if (rctCount >= 1 || totalCitations >= 3) {
            return "MODERATE";
        } else if (totalCitations >= 2) {
            return "LOW";
        } else {
            return "VERY_LOW";
        }
    }

    /**
     * Generate formatted evidence trail for UI display
     *
     * @param actionId The protocol action ID
     * @return Formatted evidence trail string
     */
    public String generateFormattedEvidenceTrail(String actionId) {
        logger.info("Generating formatted evidence trail for action: {}", actionId);

        EvidenceChain chain = resolveChain(actionId);

        if (chain == null) {
            return "No evidence trail available for action: " + actionId;
        }

        return chain.getFormattedEvidenceTrail();
    }

    /**
     * Get evidence summary for UI display
     *
     * @param actionId The protocol action ID
     * @return Compact evidence summary
     */
    public String getEvidenceSummary(String actionId) {
        EvidenceChain chain = resolveChain(actionId);

        if (chain == null) {
            return "No evidence available";
        }

        StringBuilder summary = new StringBuilder();

        if (chain.getRecommendationStrength() != null && chain.getEvidenceQuality() != null) {
            summary.append(chain.getQualityBadge()).append(" | ");
            summary.append("Strength: ").append(chain.getRecommendationStrength()).append(", ");
            summary.append("Quality: ").append(chain.getEvidenceQuality());
        }

        if (chain.getCitations() != null && !chain.getCitations().isEmpty()) {
            summary.append(" | ").append(chain.getCitations().size()).append(" citations");
        }

        return summary.toString();
    }

    // Private helper methods

    /**
     * Populate guideline information in evidence chain
     */
    private void populateGuidelineInfo(EvidenceChain chain, String guidelineId) {
        try {
            GuidelineIntegrationService.Guideline guideline = guidelineLoader.loadGuideline(guidelineId);

            if (guideline != null) {
                chain.setGuidelineName(guideline.getName());
                chain.setGuidelineOrganization(guideline.getOrganization());
                chain.setGuidelinePublicationDate(guideline.getPublicationDate());
                chain.setGuidelineNextReviewDate(guideline.getNextReviewDate());
                chain.setGuidelineStatus(guideline.getStatus());
            }

        } catch (Exception e) {
            logger.error("Error populating guideline info for {}: {}", guidelineId, e.getMessage());
        }
    }

    /**
     * Populate recommendation information in evidence chain
     */
    private void populateRecommendationInfo(
        EvidenceChain chain,
        String guidelineId,
        String recommendationId
    ) {
        try {
            GuidelineIntegrationService.Recommendation recommendation =
                guidelineLoader.loadRecommendation(guidelineId, recommendationId);

            if (recommendation != null) {
                chain.setRecommendationStatement(recommendation.getStatement());
                chain.setRecommendationStrength(recommendation.getStrength());
                chain.setEvidenceQuality(recommendation.getEvidenceQuality());

                // Extract class and level from recommendation
                extractClassAndLevel(chain, recommendation);
            }

        } catch (Exception e) {
            logger.error("Error populating recommendation info for {}: {}",
                recommendationId, e.getMessage());
        }
    }

    /**
     * Populate citations in evidence chain
     */
    private void populateCitations(EvidenceChain chain, List<String> pmids) {
        if (pmids == null || pmids.isEmpty()) {
            return;
        }

        List<EvidenceChain.Citation> citations = new ArrayList<>();
        List<String> validPmids = new ArrayList<>();

        for (String pmid : pmids) {
            try {
                EvidenceChain.Citation citation = citationLoader.loadCitation(pmid);
                if (citation != null) {
                    citations.add(citation);
                    validPmids.add(pmid);
                }
            } catch (Exception e) {
                logger.warn("Error loading citation for PMID {}: {}", pmid, e.getMessage());
            }
        }

        chain.setCitations(citations);
        chain.setKeyEvidencePmids(validPmids);
    }

    /**
     * Aggregate evidence quality from all sources
     */
    private void aggregateEvidenceQuality(EvidenceChain chain) {
        // If evidence quality not set, assess from citations
        if (chain.getEvidenceQuality() == null && chain.getCitations() != null) {
            String assessedQuality = assessEvidenceQuality(chain.getCitations());
            chain.setEvidenceQuality(assessedQuality);
            chain.setGradeLevel(assessedQuality);
        }
    }

    /**
     * Extract class of recommendation and level of evidence
     */
    private void extractClassAndLevel(
        EvidenceChain chain,
        GuidelineIntegrationService.Recommendation recommendation
    ) {
        // Parse from statement or use dedicated fields
        String statement = recommendation.getStatement();

        if (statement != null) {
            // Extract Class (I, IIA, IIB, III)
            if (statement.contains("Class I")) {
                chain.setClassOfRecommendation("CLASS_I");
            } else if (statement.contains("Class IIa") || statement.contains("Class IIA")) {
                chain.setClassOfRecommendation("CLASS_IIA");
            } else if (statement.contains("Class IIb") || statement.contains("Class IIB")) {
                chain.setClassOfRecommendation("CLASS_IIB");
            } else if (statement.contains("Class III")) {
                chain.setClassOfRecommendation("CLASS_III");
            }

            // Extract Level (A, B-R, B-NR, C-LD, C-EO)
            if (statement.contains("Level A")) {
                chain.setLevelOfEvidence("A");
            } else if (statement.contains("Level B-R")) {
                chain.setLevelOfEvidence("B-R");
            } else if (statement.contains("Level B-NR")) {
                chain.setLevelOfEvidence("B-NR");
            } else if (statement.contains("Level C-LD")) {
                chain.setLevelOfEvidence("C-LD");
            } else if (statement.contains("Level C-EO")) {
                chain.setLevelOfEvidence("C-EO");
            }
        }
    }

    /**
     * Initialize action-guideline mappings
     * In production, this would be loaded from configuration/database
     */
    private void initializeActionMappings() {
        // STEMI Protocol Mappings
        actionMappings.put("STEMI-ACT-001", new ActionGuidelineMapping(
            "STEMI-ACT-001",
            "GUIDE-ACCAHA-STEMI-2023",
            "ACC-STEMI-2023-REC-001",
            Arrays.asList("37079885", "12517460", "27282490")
        ));

        actionMappings.put("STEMI-ACT-002", new ActionGuidelineMapping(
            "STEMI-ACT-002",
            "GUIDE-ACCAHA-STEMI-2023",
            "ACC-STEMI-2023-REC-003",
            Arrays.asList("37079885", "3081859", "18160631")
        ));

        actionMappings.put("STEMI-ACT-003", new ActionGuidelineMapping(
            "STEMI-ACT-003",
            "GUIDE-ACCAHA-STEMI-2023",
            "ACC-STEMI-2023-REC-004",
            Arrays.asList("37079885", "19717846", "23031330", "17982182")
        ));

        actionMappings.put("STEMI-ACT-004", new ActionGuidelineMapping(
            "STEMI-ACT-004",
            "GUIDE-ACCAHA-STEMI-2023",
            "ACC-STEMI-2023-REC-005",
            Arrays.asList("37079885", "18645041", "24315361", "23131066")
        ));

        actionMappings.put("STEMI-ACT-005", new ActionGuidelineMapping(
            "STEMI-ACT-005",
            "GUIDE-ACCAHA-STEMI-2023",
            "ACC-STEMI-2023-REC-002",
            Arrays.asList("37079885", "12517460", "26260736", "18160631")
        ));

        // Sepsis Protocol Mappings
        actionMappings.put("SEPSIS-ACT-001", new ActionGuidelineMapping(
            "SEPSIS-ACT-001",
            "GUIDE-SSC-2021",
            "SSC-2021-REC-001",
            Arrays.asList("34599691", "16625125")
        ));

        actionMappings.put("SEPSIS-ACT-002", new ActionGuidelineMapping(
            "SEPSIS-ACT-002",
            "GUIDE-SSC-2021",
            "SSC-2021-REC-002",
            Arrays.asList("34599691", "16625125")
        ));

        actionMappings.put("SEPSIS-ACT-004", new ActionGuidelineMapping(
            "SEPSIS-ACT-004",
            "GUIDE-SSC-2021",
            "SSC-2021-REC-004",
            Arrays.asList("34599691", "16625125")
        ));

        actionMappings.put("SEPSIS-ACT-005", new ActionGuidelineMapping(
            "SEPSIS-ACT-005",
            "GUIDE-SSC-2021",
            "SSC-2021-REC-005",
            Arrays.asList("34599691")
        ));

        logger.info("Initialized {} action-guideline mappings", actionMappings.size());
    }

    /**
     * Get all guideline mappings for a specific action
     */
    private List<ActionGuidelineMapping> getGuidelineMappingsForAction(String actionId) {
        List<ActionGuidelineMapping> mappings = new ArrayList<>();

        ActionGuidelineMapping mapping = actionMappings.get(actionId);
        if (mapping != null) {
            mappings.add(mapping);
        }

        return mappings;
    }

    /**
     * Action-Guideline Mapping data structure
     */
    private static class ActionGuidelineMapping {
        private final String actionId;
        private final String guidelineId;
        private final String recommendationId;
        private final List<String> citationPmids;

        public ActionGuidelineMapping(
            String actionId,
            String guidelineId,
            String recommendationId,
            List<String> citationPmids
        ) {
            this.actionId = actionId;
            this.guidelineId = guidelineId;
            this.recommendationId = recommendationId;
            this.citationPmids = citationPmids;
        }

        public String getActionId() { return actionId; }
        public String getGuidelineId() { return guidelineId; }
        public String getRecommendationId() { return recommendationId; }
        public List<String> getCitationPmids() { return citationPmids; }
    }
}
