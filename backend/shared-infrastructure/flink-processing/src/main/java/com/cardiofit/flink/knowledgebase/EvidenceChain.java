package com.cardiofit.flink.knowledgebase;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Evidence Chain Model
 *
 * Represents complete evidence trail from protocol action → guideline → recommendation → citations
 *
 * @author CardioFit Platform - Module 3 Phase 5
 * @version 1.0
 * @since 2025-10-24
 */
public class EvidenceChain implements Serializable {
    private static final long serialVersionUID = 1L;

    private String actionId;
    private Guideline sourceGuideline;
    private Recommendation guidelineRecommendation;
    private List<Citation> supportingEvidence = new ArrayList<>();
    private String overallQuality; // High, Moderate, Low, Very Low
    private boolean current; // Is guideline current?
    private String qualityBadge; // Visual indicator

    // Default constructor
    public EvidenceChain() {}

    /**
     * Get formatted evidence trail for display
     */
    public String getEvidenceTrail() {
        StringBuilder trail = new StringBuilder();

        // Action
        trail.append("ACTION: ").append(actionId).append("\n");

        // Guideline
        if (sourceGuideline != null) {
            trail.append("GUIDELINE: ").append(sourceGuideline.getShortName())
                .append(" (").append(sourceGuideline.getStatus()).append(")\n");
        }

        // Recommendation
        if (guidelineRecommendation != null) {
            trail.append("RECOMMENDATION: ").append(guidelineRecommendation.getTitle())
                .append(" [").append(guidelineRecommendation.getStrength())
                .append(", ").append(guidelineRecommendation.getEvidenceQuality()).append("]\n");
        }

        // Supporting evidence
        if (!supportingEvidence.isEmpty()) {
            trail.append("EVIDENCE (").append(supportingEvidence.size()).append(" studies):\n");
            for (Citation citation : supportingEvidence) {
                trail.append("  - PMID ").append(citation.getPmid())
                    .append(": ").append(citation.getTitle())
                    .append(" (").append(citation.getStudyType()).append(")\n");
            }
        }

        // Quality assessment
        trail.append("QUALITY: ").append(qualityBadge).append(" - ").append(overallQuality).append("\n");

        return trail.toString();
    }

    /**
     * Get summary for UI display
     */
    public String getSummary() {
        return String.format("%s → %s → %s [%s]",
            actionId,
            sourceGuideline != null ? sourceGuideline.getShortName() : "N/A",
            guidelineRecommendation != null ? guidelineRecommendation.getStrength() : "N/A",
            qualityBadge
        );
    }

    // Getters and Setters
    public String getActionId() { return actionId; }
    public void setActionId(String actionId) { this.actionId = actionId; }

    public Guideline getSourceGuideline() { return sourceGuideline; }
    public void setSourceGuideline(Guideline sourceGuideline) { this.sourceGuideline = sourceGuideline; }

    public Recommendation getGuidelineRecommendation() { return guidelineRecommendation; }
    public void setGuidelineRecommendation(Recommendation guidelineRecommendation) {
        this.guidelineRecommendation = guidelineRecommendation;
    }

    public List<Citation> getSupportingEvidence() { return supportingEvidence; }
    public void setSupportingEvidence(List<Citation> supportingEvidence) {
        this.supportingEvidence = supportingEvidence;
    }

    public String getOverallQuality() { return overallQuality; }
    public void setOverallQuality(String overallQuality) { this.overallQuality = overallQuality; }

    public boolean isCurrent() { return current; }
    public void setCurrent(boolean current) { this.current = current; }

    public String getQualityBadge() { return qualityBadge; }
    public void setQualityBadge(String qualityBadge) { this.qualityBadge = qualityBadge; }

    @Override
    public String toString() {
        return "EvidenceChain{" +
            "actionId='" + actionId + '\'' +
            ", guideline=" + (sourceGuideline != null ? sourceGuideline.getShortName() : "null") +
            ", quality='" + overallQuality + '\'' +
            ", current=" + current +
            '}';
    }
}
