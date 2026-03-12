package com.cardiofit.flink.knowledgebase.medications.substitution;

import com.cardiofit.flink.knowledgebase.medications.model.Medication;
import lombok.Builder;
import lombok.Data;

/**
 * Represents a therapeutic substitution option for a medication.
 *
 * Contains the alternative medication, substitution type, rationale,
 * cost savings, and efficacy comparison information.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
@Data
@Builder
public class SubstitutionOption {

    /**
     * The alternative medication being recommended.
     */
    private Medication medication;

    /**
     * Type of substitution (generic, same class, different class, etc.).
     */
    private SubstitutionType substitutionType;

    /**
     * Clinical rationale for the substitution.
     * Example: "formulary preferred alternative", "generic equivalent", "allergy avoidance"
     */
    private String reason;

    /**
     * Estimated cost savings in dollars (positive number indicates savings).
     * Null if cost comparison not available.
     */
    private Double costSavings;

    /**
     * Efficacy score for the specific indication (0.0 to 1.0).
     * Higher score indicates better evidence of efficacy.
     * 1.0 = equivalent efficacy to original
     * 0.8-0.99 = slightly less effective
     * 0.5-0.79 = moderately less effective
     * <0.5 = significantly less effective
     */
    private Double efficacyScore;

    /**
     * Detailed efficacy comparison text.
     * Example: "Therapeutically equivalent", "Non-inferior for community-acquired pneumonia"
     */
    private String efficacyComparison;

    /**
     * Safety considerations for this substitution.
     * Example: "Monitor QTc interval", "Avoid in severe renal impairment"
     */
    private String safetyNotes;

    /**
     * Whether this substitution requires additional clinical review.
     */
    private boolean requiresClinicalReview;

    /**
     * Priority ranking (1 = highest priority, higher numbers = lower priority).
     */
    private Integer priority;

    /**
     * Whether the alternative medication is on formulary.
     */
    private boolean onFormulary;

    /**
     * Get a human-readable summary of this substitution option.
     *
     * @return Summary string for display
     */
    public String getSummary() {
        StringBuilder summary = new StringBuilder();

        if (medication != null) {
            summary.append(medication.getName());
        }

        summary.append(" (").append(substitutionType).append(")");

        if (costSavings != null && costSavings > 0) {
            summary.append(" - Save $").append(String.format("%.2f", costSavings));
        }

        if (efficacyScore != null) {
            summary.append(" - Efficacy: ").append(String.format("%.1f%%", efficacyScore * 100));
        }

        return summary.toString();
    }

    /**
     * Calculate a composite quality score for ranking alternatives.
     * Considers efficacy, cost, formulary status, and safety.
     *
     * @return Quality score (0.0 to 100.0, higher is better)
     */
    public double getQualityScore() {
        double score = 0.0;

        // Efficacy contributes up to 50 points
        if (efficacyScore != null) {
            score += efficacyScore * 50.0;
        } else {
            score += 25.0; // Default if not specified
        }

        // Cost savings contribute up to 25 points
        if (costSavings != null && costSavings > 0) {
            // Cap at $500 for scoring purposes
            double cappedSavings = Math.min(costSavings, 500.0);
            score += (cappedSavings / 500.0) * 25.0;
        }

        // Formulary status contributes 15 points
        if (onFormulary) {
            score += 15.0;
        }

        // No clinical review needed contributes 10 points
        if (!requiresClinicalReview) {
            score += 10.0;
        }

        return score;
    }

    /**
     * Get cost savings as a percentage (convenience method for tests).
     * Assumes original medication cost for percentage calculation.
     *
     * @return Cost savings percentage (e.g., 25.0 for 25% savings)
     */
    public Double getCostSavingsPercentage() {
        if (costSavings == null || costSavings <= 0) {
            return 0.0;
        }

        // Estimate percentage based on typical medication costs
        // Assumes $200 original cost if not specified
        double estimatedOriginalCost = 200.0;
        return (costSavings / estimatedOriginalCost) * 100.0;
    }
}
