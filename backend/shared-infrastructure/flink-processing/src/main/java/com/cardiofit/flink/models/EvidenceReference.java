package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Evidence Reference - Citation for clinical evidence
 *
 * Represents a reference to clinical evidence supporting a recommendation,
 * including guideline source, section, and evidence grade.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class EvidenceReference implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("reference_id")
    private String referenceId;

    @JsonProperty("guideline_source")
    private String guidelineSource;  // e.g., "Surviving Sepsis Campaign 2021"

    @JsonProperty("section")
    private String section;  // e.g., "Section 3.2, pp. 15-18"

    @JsonProperty("recommendation_number")
    private String recommendationNumber;  // e.g., "Recommendation 1.1"

    @JsonProperty("evidence_grade")
    private String evidenceGrade;  // A, B, C, D (GRADE system)

    @JsonProperty("quality_of_evidence")
    private String qualityOfEvidence;  // HIGH, MODERATE, LOW, VERY_LOW

    @JsonProperty("strength_of_recommendation")
    private String strengthOfRecommendation;  // STRONG, WEAK

    @JsonProperty("publication_year")
    private Integer publicationYear;

    @JsonProperty("url")
    private String url;

    // Default constructor
    public EvidenceReference() {
        this.referenceId = java.util.UUID.randomUUID().toString();
    }

    // Constructor with essential fields
    public EvidenceReference(String guidelineSource, String section, String evidenceGrade) {
        this();
        this.guidelineSource = guidelineSource;
        this.section = section;
        this.evidenceGrade = evidenceGrade;
    }

    // Getters and Setters
    public String getReferenceId() { return referenceId; }
    public void setReferenceId(String referenceId) { this.referenceId = referenceId; }

    public String getGuidelineSource() { return guidelineSource; }
    public void setGuidelineSource(String guidelineSource) { this.guidelineSource = guidelineSource; }

    public String getSection() { return section; }
    public void setSection(String section) { this.section = section; }

    public String getRecommendationNumber() { return recommendationNumber; }
    public void setRecommendationNumber(String recommendationNumber) {
        this.recommendationNumber = recommendationNumber;
    }

    public String getEvidenceGrade() { return evidenceGrade; }
    public void setEvidenceGrade(String evidenceGrade) { this.evidenceGrade = evidenceGrade; }

    public String getQualityOfEvidence() { return qualityOfEvidence; }
    public void setQualityOfEvidence(String qualityOfEvidence) { this.qualityOfEvidence = qualityOfEvidence; }

    public String getStrengthOfRecommendation() { return strengthOfRecommendation; }
    public void setStrengthOfRecommendation(String strengthOfRecommendation) {
        this.strengthOfRecommendation = strengthOfRecommendation;
    }

    public Integer getPublicationYear() { return publicationYear; }
    public void setPublicationYear(Integer publicationYear) { this.publicationYear = publicationYear; }

    public String getUrl() { return url; }
    public void setUrl(String url) { this.url = url; }

    // Utility methods

    /**
     * Check if evidence is high quality
     */
    public boolean isHighQuality() {
        return "HIGH".equals(qualityOfEvidence);
    }

    /**
     * Check if recommendation is strong
     */
    public boolean isStrongRecommendation() {
        return "STRONG".equals(strengthOfRecommendation);
    }

    /**
     * Check if evidence grade is A (highest)
     */
    public boolean isGradeA() {
        return "A".equals(evidenceGrade);
    }

    /**
     * Get formatted citation
     */
    public String getFormattedCitation() {
        StringBuilder citation = new StringBuilder();
        if (guidelineSource != null) {
            citation.append(guidelineSource);
        }
        if (section != null) {
            citation.append(", ").append(section);
        }
        if (recommendationNumber != null) {
            citation.append(" (").append(recommendationNumber).append(")");
        }
        if (evidenceGrade != null) {
            citation.append(" [Grade ").append(evidenceGrade).append("]");
        }
        return citation.toString();
    }

    @Override
    public String toString() {
        return "EvidenceReference{" +
            "guidelineSource='" + guidelineSource + '\'' +
            ", section='" + section + '\'' +
            ", evidenceGrade='" + evidenceGrade + '\'' +
            ", qualityOfEvidence='" + qualityOfEvidence + '\'' +
            '}';
    }
}
