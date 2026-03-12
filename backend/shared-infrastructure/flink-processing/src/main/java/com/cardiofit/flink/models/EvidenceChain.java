package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.List;

/**
 * Evidence Chain - Complete Traceability from Action to Citations
 *
 * Represents the complete evidence chain for a protocol action:
 * Action → Guideline → Recommendation → Citations
 *
 * This class enables full evidence traceability, quality assessment,
 * and identification of outdated or weak evidence.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class EvidenceChain implements Serializable {
    private static final long serialVersionUID = 1L;

    // Action being supported
    @JsonProperty("action_id")
    private String actionId;

    @JsonProperty("action_description")
    private String actionDescription;

    // Guideline Source
    @JsonProperty("guideline_id")
    private String guidelineId;

    @JsonProperty("guideline_name")
    private String guidelineName;

    @JsonProperty("guideline_organization")
    private String guidelineOrganization;

    @JsonProperty("guideline_publication_date")
    private String guidelinePublicationDate;

    @JsonProperty("guideline_next_review_date")
    private String guidelineNextReviewDate;

    @JsonProperty("guideline_status")
    private String guidelineStatus;  // CURRENT, OUTDATED, SUPERSEDED

    // Recommendation Details
    @JsonProperty("recommendation_id")
    private String recommendationId;

    @JsonProperty("recommendation_statement")
    private String recommendationStatement;

    @JsonProperty("recommendation_number")
    private String recommendationNumber;

    @JsonProperty("recommendation_strength")
    private String recommendationStrength;  // STRONG, WEAK, CONDITIONAL

    @JsonProperty("class_of_recommendation")
    private String classOfRecommendation;  // CLASS_I, CLASS_IIA, CLASS_IIB, CLASS_III

    @JsonProperty("level_of_evidence")
    private String levelOfEvidence;  // A, B-R, B-NR, C-LD, C-EO

    // Evidence Quality Assessment
    @JsonProperty("evidence_quality")
    private String evidenceQuality;  // HIGH, MODERATE, LOW, VERY_LOW (GRADE system)

    @JsonProperty("grade_level")
    private String gradeLevel;  // High, Moderate, Low, Very Low

    // Supporting Citations
    @JsonProperty("citations")
    private List<Citation> citations;

    @JsonProperty("key_evidence_pmids")
    private List<String> keyEvidencePmids;

    // Clinical Considerations
    @JsonProperty("clinical_rationale")
    private String clinicalRationale;

    @JsonProperty("clinical_considerations")
    private String clinicalConsiderations;

    // Evidence Chain Metadata
    @JsonProperty("chain_completeness_score")
    private Double chainCompletenessScore;  // 0.0 to 1.0

    @JsonProperty("evidence_gap_identified")
    private Boolean evidenceGapIdentified;

    @JsonProperty("evidence_gap_description")
    private String evidenceGapDescription;

    @JsonProperty("last_evidence_review_date")
    private String lastEvidenceReviewDate;

    // Default constructor
    public EvidenceChain() {
        this.citations = new ArrayList<>();
        this.keyEvidencePmids = new ArrayList<>();
        this.chainCompletenessScore = 0.0;
        this.evidenceGapIdentified = false;
    }

    // Constructor with essential fields
    public EvidenceChain(String actionId, String guidelineId, String recommendationId) {
        this();
        this.actionId = actionId;
        this.guidelineId = guidelineId;
        this.recommendationId = recommendationId;
    }

    // Getters and Setters

    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }

    public String getActionDescription() { return actionDescription; }
    public void setActionDescription(String actionDescription) { this.actionDescription = actionDescription; }

    public String getGuidelineId() { return guidelineId; }
    public void setGuidelineId(String guidelineId) { this.guidelineId = guidelineId; }

    public String getGuidelineName() { return guidelineName; }
    public void setGuidelineName(String guidelineName) { this.guidelineName = guidelineName; }

    public String getGuidelineOrganization() { return guidelineOrganization; }
    public void setGuidelineOrganization(String guidelineOrganization) {
        this.guidelineOrganization = guidelineOrganization;
    }

    public String getGuidelinePublicationDate() { return guidelinePublicationDate; }
    public void setGuidelinePublicationDate(String guidelinePublicationDate) {
        this.guidelinePublicationDate = guidelinePublicationDate;
    }

    public String getGuidelineNextReviewDate() { return guidelineNextReviewDate; }
    public void setGuidelineNextReviewDate(String guidelineNextReviewDate) {
        this.guidelineNextReviewDate = guidelineNextReviewDate;
    }

    public String getGuidelineStatus() { return guidelineStatus; }
    public void setGuidelineStatus(String guidelineStatus) { this.guidelineStatus = guidelineStatus; }

    public String getRecommendationId() { return recommendationId; }
    public void setRecommendationId(String recommendationId) { this.recommendationId = recommendationId; }

    public String getRecommendationStatement() { return recommendationStatement; }
    public void setRecommendationStatement(String recommendationStatement) {
        this.recommendationStatement = recommendationStatement;
    }

    public String getRecommendationNumber() { return recommendationNumber; }
    public void setRecommendationNumber(String recommendationNumber) {
        this.recommendationNumber = recommendationNumber;
    }

    public String getRecommendationStrength() { return recommendationStrength; }
    public void setRecommendationStrength(String recommendationStrength) {
        this.recommendationStrength = recommendationStrength;
    }

    public String getClassOfRecommendation() { return classOfRecommendation; }
    public void setClassOfRecommendation(String classOfRecommendation) {
        this.classOfRecommendation = classOfRecommendation;
    }

    public String getLevelOfEvidence() { return levelOfEvidence; }
    public void setLevelOfEvidence(String levelOfEvidence) { this.levelOfEvidence = levelOfEvidence; }

    public String getEvidenceQuality() { return evidenceQuality; }
    public void setEvidenceQuality(String evidenceQuality) { this.evidenceQuality = evidenceQuality; }

    public String getGradeLevel() { return gradeLevel; }
    public void setGradeLevel(String gradeLevel) { this.gradeLevel = gradeLevel; }

    public List<Citation> getCitations() { return citations; }
    public void setCitations(List<Citation> citations) { this.citations = citations; }

    public List<String> getKeyEvidencePmids() { return keyEvidencePmids; }
    public void setKeyEvidencePmids(List<String> keyEvidencePmids) { this.keyEvidencePmids = keyEvidencePmids; }

    public String getClinicalRationale() { return clinicalRationale; }
    public void setClinicalRationale(String clinicalRationale) { this.clinicalRationale = clinicalRationale; }

    public String getClinicalConsiderations() { return clinicalConsiderations; }
    public void setClinicalConsiderations(String clinicalConsiderations) {
        this.clinicalConsiderations = clinicalConsiderations;
    }

    public Double getChainCompletenessScore() { return chainCompletenessScore; }
    public void setChainCompletenessScore(Double chainCompletenessScore) {
        this.chainCompletenessScore = chainCompletenessScore;
    }

    public Boolean getEvidenceGapIdentified() { return evidenceGapIdentified; }
    public void setEvidenceGapIdentified(Boolean evidenceGapIdentified) {
        this.evidenceGapIdentified = evidenceGapIdentified;
    }

    public String getEvidenceGapDescription() { return evidenceGapDescription; }
    public void setEvidenceGapDescription(String evidenceGapDescription) {
        this.evidenceGapDescription = evidenceGapDescription;
    }

    public String getLastEvidenceReviewDate() { return lastEvidenceReviewDate; }
    public void setLastEvidenceReviewDate(String lastEvidenceReviewDate) {
        this.lastEvidenceReviewDate = lastEvidenceReviewDate;
    }

    // Convenience Methods for GuidelineLinker Integration

    /**
     * Set supporting evidence citations (alias for setCitations)
     */
    public void setSupportingEvidence(List<Citation> citations) {
        setCitations(citations);
    }

    /**
     * Get supporting evidence citations (alias for getCitations)
     */
    public List<Citation> getSupportingEvidence() {
        return getCitations();
    }

    /**
     * Set source guideline from GuidelineIntegrationService.Guideline object
     */
    public void setSourceGuideline(com.cardiofit.flink.knowledgebase.GuidelineIntegrationService.Guideline guideline) {
        if (guideline != null) {
            this.guidelineId = guideline.getGuidelineId();
            this.guidelineName = guideline.getName();
            this.guidelineOrganization = guideline.getOrganization();
            this.guidelinePublicationDate = guideline.getPublicationDate();
            this.guidelineNextReviewDate = guideline.getNextReviewDate();
            this.guidelineStatus = guideline.getStatus();
        }
    }

    /**
     * Get source guideline as a lightweight GuidelineReference object
     */
    public GuidelineReference getSourceGuideline() {
        GuidelineReference ref = new GuidelineReference();
        ref.guidelineId = this.guidelineId;
        ref.guidelineName = this.guidelineName;
        ref.guidelineOrganization = this.guidelineOrganization;
        ref.guidelinePublicationDate = this.guidelinePublicationDate;
        ref.guidelineStatus = this.guidelineStatus;
        return ref;
    }

    /**
     * Set guideline recommendation from GuidelineIntegrationService.Recommendation object
     */
    public void setGuidelineRecommendation(com.cardiofit.flink.knowledgebase.GuidelineIntegrationService.Recommendation recommendation) {
        if (recommendation != null) {
            this.recommendationId = recommendation.getRecommendationId();
            this.recommendationStatement = recommendation.getStatement();
            this.recommendationStrength = recommendation.getStrength();
            this.classOfRecommendation = recommendation.getClassOfRecommendation();
            this.levelOfEvidence = recommendation.getLevelOfEvidence();
            this.evidenceQuality = recommendation.getEvidenceQuality();
        }
    }

    /**
     * Get guideline recommendation as a lightweight RecommendationReference object
     */
    public RecommendationReference getGuidelineRecommendation() {
        RecommendationReference ref = new RecommendationReference();
        ref.recommendationId = this.recommendationId;
        ref.recommendationStatement = this.recommendationStatement;
        ref.recommendationStrength = this.recommendationStrength;
        ref.classOfRecommendation = this.classOfRecommendation;
        ref.levelOfEvidence = this.levelOfEvidence;
        return ref;
    }

    /**
     * Set whether guideline is current (alias for setGuidelineStatus)
     */
    public void setCurrent(boolean current) {
        this.guidelineStatus = current ? "CURRENT" : "OUTDATED";
    }

    /**
     * Set overall evidence quality (alias for setEvidenceQuality)
     */
    public void setOverallQuality(String quality) {
        setEvidenceQuality(quality);
        setGradeLevel(quality);
    }

    /**
     * Set quality badge directly (overrides calculated value)
     */
    private String qualityBadgeOverride;

    public void setQualityBadge(String badge) {
        this.qualityBadgeOverride = badge;
    }

    // Utility Methods

    /**
     * Check if guideline is current (not outdated)
     */
    public boolean isGuidelineCurrent() {
        if (guidelineStatus != null && "OUTDATED".equalsIgnoreCase(guidelineStatus)) {
            return false;
        }

        if (guidelineNextReviewDate != null) {
            try {
                LocalDate nextReview = LocalDate.parse(guidelineNextReviewDate);
                return LocalDate.now().isBefore(nextReview);
            } catch (Exception e) {
                // If parsing fails, assume current
                return true;
            }
        }

        return true;
    }

    /**
     * Check if evidence chain indicates outdated guideline
     */
    public boolean isOutdated() {
        return !isGuidelineCurrent();
    }

    /**
     * Check if evidence is high quality
     */
    public boolean hasHighQualityEvidence() {
        return "HIGH".equalsIgnoreCase(evidenceQuality) || "High".equalsIgnoreCase(gradeLevel);
    }

    /**
     * Check if recommendation is strong
     */
    public boolean hasStrongRecommendation() {
        return "STRONG".equalsIgnoreCase(recommendationStrength) ||
               "CLASS_I".equalsIgnoreCase(classOfRecommendation);
    }

    /**
     * Check if chain is complete (has all required elements)
     */
    public boolean isChainComplete() {
        return chainCompletenessScore != null && chainCompletenessScore >= 0.8;
    }

    /**
     * Calculate chain completeness score based on available data
     */
    public void calculateCompletenessScore() {
        double score = 0.0;
        int totalFields = 10;

        if (guidelineId != null && !guidelineId.isEmpty()) score += 0.1;
        if (guidelineName != null && !guidelineName.isEmpty()) score += 0.1;
        if (recommendationId != null && !recommendationId.isEmpty()) score += 0.2;
        if (recommendationStatement != null && !recommendationStatement.isEmpty()) score += 0.1;
        if (evidenceQuality != null && !evidenceQuality.isEmpty()) score += 0.1;
        if (recommendationStrength != null && !recommendationStrength.isEmpty()) score += 0.1;
        if (citations != null && !citations.isEmpty()) score += 0.2;
        if (clinicalRationale != null && !clinicalRationale.isEmpty()) score += 0.05;
        if (classOfRecommendation != null && !classOfRecommendation.isEmpty()) score += 0.025;
        if (levelOfEvidence != null && !levelOfEvidence.isEmpty()) score += 0.025;

        this.chainCompletenessScore = Math.min(1.0, score);
    }

    /**
     * Get quality badge for UI display
     */
    public String getQualityBadge() {
        // Return override if set
        if (qualityBadgeOverride != null) {
            return qualityBadgeOverride;
        }

        if (isOutdated()) {
            return "⚠️ OUTDATED";
        }

        if (hasHighQualityEvidence() && hasStrongRecommendation()) {
            return "🟢 STRONG";
        } else if ("MODERATE".equalsIgnoreCase(evidenceQuality)) {
            return "🟡 MODERATE";
        } else if ("LOW".equalsIgnoreCase(evidenceQuality) || "VERY_LOW".equalsIgnoreCase(evidenceQuality)) {
            return "🟠 WEAK";
        }

        return "⚪ UNGRADED";
    }

    /**
     * Get formatted evidence trail for display
     */
    public String getFormattedEvidenceTrail() {
        StringBuilder trail = new StringBuilder();

        trail.append("Action: ").append(actionDescription != null ? actionDescription : actionId).append("\n");
        trail.append("  ↓\n");

        trail.append("Guideline: ").append(guidelineName != null ? guidelineName : guidelineId);
        if (guidelinePublicationDate != null) {
            trail.append(" (").append(guidelinePublicationDate).append(")");
        }
        trail.append("\n  ↓\n");

        trail.append("Recommendation: ").append(recommendationNumber != null ? recommendationNumber : recommendationId);
        if (recommendationStatement != null && recommendationStatement.length() <= 80) {
            trail.append(" - ").append(recommendationStatement);
        }
        trail.append("\n");

        if (recommendationStrength != null || classOfRecommendation != null) {
            trail.append("  Strength: ");
            if (recommendationStrength != null) {
                trail.append(recommendationStrength);
            }
            if (classOfRecommendation != null) {
                trail.append(" (").append(classOfRecommendation).append(")");
            }
            trail.append("\n");
        }

        if (evidenceQuality != null || levelOfEvidence != null) {
            trail.append("  Evidence: ");
            if (evidenceQuality != null) {
                trail.append(evidenceQuality);
            }
            if (levelOfEvidence != null) {
                trail.append(" (Level ").append(levelOfEvidence).append(")");
            }
            trail.append("\n");
        }

        trail.append("  ↓\n");

        if (citations != null && !citations.isEmpty()) {
            trail.append("Citations:\n");
            for (Citation citation : citations) {
                trail.append("  • ").append(citation.getFormattedCitation()).append("\n");
            }
        }

        trail.append("  ↓\n");
        trail.append("Quality Badge: ").append(getQualityBadge());

        return trail.toString();
    }

    /**
     * Alias for getFormattedEvidenceTrail() for test compatibility.
     * @return Formatted evidence trail string
     */
    public String getEvidenceTrail() {
        return getFormattedEvidenceTrail();
    }

    /**
     * Get overall quality assessment of the evidence chain.
     * Returns the evidence quality (GRADE system) if available, otherwise derives from level of evidence.
     * @return Quality rating (HIGH, MODERATE, LOW, VERY_LOW)
     */
    public String getOverallQuality() {
        if (evidenceQuality != null) {
            return evidenceQuality;
        }

        // Derive from level of evidence if evidenceQuality not set
        if (levelOfEvidence != null) {
            switch (levelOfEvidence.toUpperCase()) {
                case "A": return "HIGH";
                case "B-R": return "MODERATE";
                case "B-NR": return "MODERATE";
                case "C-LD": return "LOW";
                case "C-EO": return "VERY_LOW";
                default: return "LOW";
            }
        }

        return "LOW"; // Default to LOW if no quality indicators present
    }

    /**
     * Get short name for the guideline.
     * Extracts a shortened version from guidelineName or returns guidelineId.
     * @return Short guideline name
     */
    public String getShortName() {
        if (guidelineName != null) {
            // Extract acronym if guideline name contains organization name
            if (guidelineName.contains("/")) {
                return guidelineName.split("/")[0].trim();
            }
            // Return first 50 characters if too long
            if (guidelineName.length() > 50) {
                return guidelineName.substring(0, 47) + "...";
            }
            return guidelineName;
        }
        return guidelineId;
    }

    /**
     * Set short name (currently maps to guidelineName for compatibility).
     * @param shortName The short name to set
     */
    public void setShortName(String shortName) {
        this.guidelineName = shortName;
    }

    /**
     * Get concise summary of evidence chain for UI display
     * @return Formatted summary string
     */
    public String getSummary() {
        return String.format("%s → %s → %s [%s]",
            actionId != null ? actionId : "N/A",
            guidelineName != null ? guidelineName : guidelineId != null ? guidelineId : "N/A",
            recommendationStrength != null ? recommendationStrength : "N/A",
            getQualityBadge()
        );
    }

    @Override
    public String toString() {
        return "EvidenceChain{" +
            "actionId='" + actionId + '\'' +
            ", guidelineId='" + guidelineId + '\'' +
            ", recommendationId='" + recommendationId + '\'' +
            ", evidenceQuality='" + evidenceQuality + '\'' +
            ", recommendationStrength='" + recommendationStrength + '\'' +
            ", citationCount=" + (citations != null ? citations.size() : 0) +
            ", completenessScore=" + chainCompletenessScore +
            ", isOutdated=" + isOutdated() +
            ", qualityBadge='" + getQualityBadge() + '\'' +
            '}';
    }

    /**
     * Lightweight guideline reference for getSourceGuideline()
     */
    public static class GuidelineReference implements Serializable {
        private static final long serialVersionUID = 1L;

        public String guidelineId;
        public String guidelineName;
        public String guidelineOrganization;
        public String guidelinePublicationDate;
        public String guidelineStatus;

        public String getGuidelineId() { return guidelineId; }
        public String getGuidelineName() { return guidelineName; }
        public String getGuidelineOrganization() { return guidelineOrganization; }
        public String getGuidelinePublicationDate() { return guidelinePublicationDate; }
        public String getGuidelineStatus() { return guidelineStatus; }
    }

    /**
     * Lightweight recommendation reference for getGuidelineRecommendation()
     */
    public static class RecommendationReference implements Serializable {
        private static final long serialVersionUID = 1L;

        public String recommendationId;
        public String recommendationStatement;
        public String recommendationStrength;
        public String classOfRecommendation;
        public String levelOfEvidence;

        public String getRecommendationId() { return recommendationId; }
        public String getRecommendationStatement() { return recommendationStatement; }
        public String getRecommendationStrength() { return recommendationStrength; }
        public String getClassOfRecommendation() { return classOfRecommendation; }
        public String getLevelOfEvidence() { return levelOfEvidence; }
    }

    /**
     * Citation nested class for evidence chain
     */
    public static class Citation implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("pmid")
        private String pmid;

        @JsonProperty("authors")
        private String authors;

        @JsonProperty("title")
        private String title;

        @JsonProperty("journal")
        private String journal;

        @JsonProperty("year")
        private Integer year;

        @JsonProperty("citation_summary")
        private String citationSummary;

        // Getters and setters
        public String getPmid() { return pmid; }
        public void setPmid(String pmid) { this.pmid = pmid; }

        public String getAuthors() { return authors; }
        public void setAuthors(String authors) { this.authors = authors; }

        public String getTitle() { return title; }
        public void setTitle(String title) { this.title = title; }

        public String getJournal() { return journal; }
        public void setJournal(String journal) { this.journal = journal; }

        public Integer getYear() { return year; }
        public void setYear(Integer year) { this.year = year; }

        public String getCitationSummary() { return citationSummary; }
        public void setCitationSummary(String citationSummary) { this.citationSummary = citationSummary; }

        /**
         * Get formatted citation for display
         */
        public String getFormattedCitation() {
            StringBuilder formatted = new StringBuilder();

            if (pmid != null) {
                formatted.append("PMID ").append(pmid).append(": ");
            }

            if (title != null && title.length() <= 60) {
                formatted.append(title);
            } else if (authors != null) {
                formatted.append(authors);
            }

            if (citationSummary != null) {
                formatted.append(" - ").append(citationSummary);
            }

            return formatted.toString();
        }

        @Override
        public String toString() {
            return "Citation{pmid='" + pmid + "', title='" + title + "'}";
        }
    }
}
