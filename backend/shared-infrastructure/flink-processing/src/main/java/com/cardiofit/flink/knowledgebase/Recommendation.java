package com.cardiofit.flink.knowledgebase;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Clinical Recommendation Model
 *
 * Represents a single recommendation within a clinical guideline.
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class Recommendation implements Serializable {
    private static final long serialVersionUID = 1L;

    private String recommendationId;
    private String number; // e.g., "1.1", "2.3"
    private String section;
    private String title;
    private String statement;

    // GRADE system fields
    private String strength; // STRONG, WEAK, CONDITIONAL
    private String classOfRecommendation; // Class I, IIa, IIb, III
    private String evidenceQuality; // HIGH, MODERATE, LOW, VERY_LOW
    private String gradeLevel; // High, Moderate, Low, Very Low
    private String levelOfEvidence; // A, B-R, B-NR, C-LD, C-EO

    private String rationale;
    private List<String> keyEvidence = new ArrayList<>(); // PMIDs
    private List<String> linkedProtocolActions = new ArrayList<>();
    private String clinicalConsiderations;

    // Default constructor
    public Recommendation() {}

    // Utility methods
    public boolean isStrongRecommendation() {
        return "STRONG".equals(strength);
    }

    public boolean isHighQuality() {
        return "HIGH".equals(evidenceQuality);
    }

    public boolean isClassI() {
        return "Class I".equals(classOfRecommendation);
    }

    // Getters and Setters
    public String getRecommendationId() { return recommendationId; }
    public void setRecommendationId(String recommendationId) { this.recommendationId = recommendationId; }

    public String getNumber() { return number; }
    public void setNumber(String number) { this.number = number; }

    public String getSection() { return section; }
    public void setSection(String section) { this.section = section; }

    public String getTitle() { return title; }
    public void setTitle(String title) { this.title = title; }

    public String getStatement() { return statement; }
    public void setStatement(String statement) { this.statement = statement; }

    public String getStrength() { return strength; }
    public void setStrength(String strength) { this.strength = strength; }

    public String getClassOfRecommendation() { return classOfRecommendation; }
    public void setClassOfRecommendation(String classOfRecommendation) { this.classOfRecommendation = classOfRecommendation; }

    public String getEvidenceQuality() { return evidenceQuality; }
    public void setEvidenceQuality(String evidenceQuality) { this.evidenceQuality = evidenceQuality; }

    public String getGradeLevel() { return gradeLevel; }
    public void setGradeLevel(String gradeLevel) { this.gradeLevel = gradeLevel; }

    public String getLevelOfEvidence() { return levelOfEvidence; }
    public void setLevelOfEvidence(String levelOfEvidence) { this.levelOfEvidence = levelOfEvidence; }

    public String getRationale() { return rationale; }
    public void setRationale(String rationale) { this.rationale = rationale; }

    public List<String> getKeyEvidence() { return keyEvidence; }
    public void setKeyEvidence(List<String> keyEvidence) { this.keyEvidence = keyEvidence; }

    public List<String> getLinkedProtocolActions() { return linkedProtocolActions; }
    public void setLinkedProtocolActions(List<String> linkedProtocolActions) { this.linkedProtocolActions = linkedProtocolActions; }

    public String getClinicalConsiderations() { return clinicalConsiderations; }
    public void setClinicalConsiderations(String clinicalConsiderations) { this.clinicalConsiderations = clinicalConsiderations; }

    @Override
    public String toString() {
        return "Recommendation{" +
            "recommendationId='" + recommendationId + '\'' +
            ", title='" + title + '\'' +
            ", strength='" + strength + '\'' +
            ", evidenceQuality='" + evidenceQuality + '\'' +
            '}';
    }
}
