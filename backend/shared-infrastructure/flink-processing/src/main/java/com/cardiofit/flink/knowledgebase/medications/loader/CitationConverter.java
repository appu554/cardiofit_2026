package com.cardiofit.flink.knowledgebase.medications.loader;

import com.cardiofit.flink.knowledgebase.Citation;
import com.cardiofit.flink.knowledgebase.Recommendation;
import com.cardiofit.flink.knowledgebase.GuidelineIntegrationService;
import com.cardiofit.flink.models.EvidenceChain;
import java.util.ArrayList;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Citation Type Converter
 *
 * Converts between two Citation implementations:
 * - com.cardiofit.flink.knowledgebase.Citation (full scientific model)
 * - com.cardiofit.flink.models.EvidenceChain.Citation (lightweight nested class)
 *
 * This utility enables seamless type conversion without architectural refactoring.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
public class CitationConverter {

    /**
     * Convert standalone Citation to EvidenceChain.Citation
     *
     * @param standalone Full citation model
     * @return Lightweight nested citation
     */
    public static EvidenceChain.Citation toEvidenceChainCitation(Citation standalone) {
        if (standalone == null) {
            return null;
        }

        EvidenceChain.Citation nested = new EvidenceChain.Citation();
        nested.setPmid(standalone.getPmid());
        nested.setTitle(standalone.getTitle());
        nested.setJournal(standalone.getJournal());
        nested.setYear(standalone.getPublicationYear());

        // Convert authors list to comma-separated string
        if (standalone.getAuthors() != null && !standalone.getAuthors().isEmpty()) {
            nested.setAuthors(String.join(", ", standalone.getAuthors()));
        } else if (standalone.getFirstAuthor() != null) {
            nested.setAuthors(standalone.getFirstAuthor() + " et al");
        }

        // Generate citation summary from study metadata
        nested.setCitationSummary(generateCitationSummary(standalone));

        return nested;
    }

    /**
     * Convert list of standalone Citations to EvidenceChain.Citation list
     *
     * @param standaloneList Full citation models
     * @return Lightweight nested citations
     */
    public static List<EvidenceChain.Citation> toEvidenceChainCitations(List<Citation> standaloneList) {
        if (standaloneList == null) {
            return new ArrayList<>();
        }

        return standaloneList.stream()
            .map(CitationConverter::toEvidenceChainCitation)
            .collect(Collectors.toList());
    }

    /**
     * Convert EvidenceChain.Citation to standalone Citation
     *
     * @param nested Lightweight nested citation
     * @return Full citation model
     */
    public static Citation toStandaloneCitation(EvidenceChain.Citation nested) {
        if (nested == null) {
            return null;
        }

        Citation standalone = new Citation();
        standalone.setPmid(nested.getPmid());
        standalone.setTitle(nested.getTitle());
        standalone.setJournal(nested.getJournal());
        standalone.setPublicationYear(nested.getYear());

        // Parse authors string back to list
        if (nested.getAuthors() != null) {
            String authorsStr = nested.getAuthors();
            if (authorsStr.contains(",")) {
                List<String> authorList = new ArrayList<>();
                for (String author : authorsStr.split(",\\s*")) {
                    if (!author.equals("et al")) {
                        authorList.add(author.trim());
                    }
                }
                standalone.setAuthors(authorList);
            } else {
                standalone.setFirstAuthor(authorsStr.replace(" et al", "").trim());
            }
        }

        return standalone;
    }

    /**
     * Convert list of EvidenceChain.Citations to standalone Citations
     *
     * @param nestedList Lightweight nested citations
     * @return Full citation models
     */
    public static List<Citation> toStandaloneCitations(List<EvidenceChain.Citation> nestedList) {
        if (nestedList == null) {
            return new ArrayList<>();
        }

        return nestedList.stream()
            .map(CitationConverter::toStandaloneCitation)
            .collect(Collectors.toList());
    }

    /**
     * Generate citation summary from standalone Citation metadata
     *
     * @param citation Full citation model
     * @return Summary string
     */
    private static String generateCitationSummary(Citation citation) {
        StringBuilder summary = new StringBuilder();

        if (citation.getStudyType() != null) {
            summary.append(citation.getStudyType());
        }

        if (citation.getEvidenceQuality() != null) {
            if (summary.length() > 0) summary.append(", ");
            summary.append(citation.getEvidenceQuality()).append(" quality");
        }

        if (citation.getSampleSize() != null) {
            if (summary.length() > 0) summary.append(", ");
            summary.append("n=").append(citation.getSampleSize());
        }

        if (citation.getPopulation() != null) {
            if (summary.length() > 0) summary.append(", ");
            summary.append(citation.getPopulation());
        }

        return summary.length() > 0 ? summary.toString() : null;
    }

    /**
     * Check if two citations represent the same publication
     *
     * @param c1 First citation (either type)
     * @param c2 Second citation (either type)
     * @return True if they have matching PMID or DOI
     */
    public static boolean isSameCitation(Object c1, Object c2) {
        if (c1 == null || c2 == null) {
            return false;
        }

        String pmid1 = null;
        String pmid2 = null;

        if (c1 instanceof Citation) {
            pmid1 = ((Citation) c1).getPmid();
        } else if (c1 instanceof EvidenceChain.Citation) {
            pmid1 = ((EvidenceChain.Citation) c1).getPmid();
        }

        if (c2 instanceof Citation) {
            pmid2 = ((Citation) c2).getPmid();
        } else if (c2 instanceof EvidenceChain.Citation) {
            pmid2 = ((EvidenceChain.Citation) c2).getPmid();
        }

        return pmid1 != null && pmid1.equals(pmid2);
    }

    /**
     * Validate citation has minimum required fields
     *
     * @param citation Citation to validate (either type)
     * @return True if valid
     */
    public static boolean isValidCitation(Object citation) {
        if (citation == null) {
            return false;
        }

        if (citation instanceof Citation) {
            Citation c = (Citation) citation;
            return c.getPmid() != null || (c.getTitle() != null && c.getPublicationYear() != null);
        } else if (citation instanceof EvidenceChain.Citation) {
            EvidenceChain.Citation c = (EvidenceChain.Citation) citation;
            return c.getPmid() != null || (c.getTitle() != null && c.getYear() != null);
        }

        return false;
    }

    // ========================================================================
    // RECOMMENDATION CONVERSION METHODS
    // ========================================================================

    /**
     * Convert standalone Recommendation to GuidelineIntegrationService.Recommendation
     *
     * @param standalone Full recommendation model
     * @return Nested recommendation
     */
    public static GuidelineIntegrationService.Recommendation toNestedRecommendation(Recommendation standalone) {
        if (standalone == null) {
            return null;
        }

        GuidelineIntegrationService.Recommendation nested = new GuidelineIntegrationService.Recommendation();
        nested.setRecommendationId(standalone.getRecommendationId());
        nested.setStatement(standalone.getStatement());
        nested.setStrength(standalone.getStrength());
        nested.setEvidenceQuality(standalone.getEvidenceQuality());
        nested.setClassOfRecommendation(standalone.getClassOfRecommendation());
        nested.setLevelOfEvidence(standalone.getLevelOfEvidence());
        nested.setLinkedProtocolActions(standalone.getLinkedProtocolActions());
        nested.setCitationPmids(standalone.getKeyEvidence());

        return nested;
    }

    /**
     * Convert list of standalone Recommendations to nested Recommendations
     *
     * @param standaloneList Full recommendation models
     * @return Nested recommendations
     */
    public static List<GuidelineIntegrationService.Recommendation> toNestedRecommendations(List<Recommendation> standaloneList) {
        if (standaloneList == null) {
            return new ArrayList<>();
        }

        return standaloneList.stream()
            .map(CitationConverter::toNestedRecommendation)
            .collect(Collectors.toList());
    }

    /**
     * Convert GuidelineIntegrationService.Recommendation to standalone Recommendation
     *
     * @param nested Nested recommendation
     * @return Full recommendation model
     */
    public static Recommendation toStandaloneRecommendation(GuidelineIntegrationService.Recommendation nested) {
        if (nested == null) {
            return null;
        }

        Recommendation standalone = new Recommendation();
        standalone.setRecommendationId(nested.getRecommendationId());
        standalone.setStatement(nested.getStatement());
        standalone.setStrength(nested.getStrength());
        standalone.setEvidenceQuality(nested.getEvidenceQuality());
        standalone.setClassOfRecommendation(nested.getClassOfRecommendation());
        standalone.setLevelOfEvidence(nested.getLevelOfEvidence());
        standalone.setLinkedProtocolActions(nested.getLinkedProtocolActions());
        standalone.setKeyEvidence(nested.getCitationPmids());

        return standalone;
    }

    /**
     * Convert list of nested Recommendations to standalone Recommendations
     *
     * @param nestedList Nested recommendations
     * @return Full recommendation models
     */
    public static List<Recommendation> toStandaloneRecommendations(List<GuidelineIntegrationService.Recommendation> nestedList) {
        if (nestedList == null) {
            return new ArrayList<>();
        }

        return nestedList.stream()
            .map(CitationConverter::toStandaloneRecommendation)
            .collect(Collectors.toList());
    }
}
