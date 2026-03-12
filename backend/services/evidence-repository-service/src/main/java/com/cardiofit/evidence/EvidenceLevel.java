package com.cardiofit.evidence;

/**
 * GRADE Evidence Quality Levels
 *
 * Based on the GRADE (Grading of Recommendations Assessment,
 * Development and Evaluation) framework for evidence quality.
 *
 * Used to categorize the strength of medical evidence supporting
 * clinical recommendations.
 *
 * Reference: https://www.gradeworkinggroup.org/
 */
public enum EvidenceLevel {

    /**
     * HIGH - High confidence in the evidence
     *
     * Further research is very unlikely to change confidence in the estimate of effect.
     *
     * Typical sources:
     * - Well-designed RCTs with consistent results
     * - Overwhelming evidence from observational studies
     */
    HIGH(
        "High quality",
        "RCTs with consistent results or overwhelming observational evidence",
        4
    ),

    /**
     * MODERATE - Moderate confidence in the evidence
     *
     * Further research is likely to have an important impact on confidence
     * in the estimate of effect and may change the estimate.
     *
     * Typical sources:
     * - RCTs with important limitations (inconsistent results, methodological flaws)
     * - Strong evidence from observational studies
     */
    MODERATE(
        "Moderate quality",
        "RCTs with limitations or strong observational studies",
        3
    ),

    /**
     * LOW - Low confidence in the evidence
     *
     * Further research is very likely to have an important impact on confidence
     * in the estimate of effect and is likely to change the estimate.
     *
     * Typical sources:
     * - Observational studies with limitations
     * - Weak or inconsistent RCTs
     */
    LOW(
        "Low quality",
        "Observational studies with limitations or weak RCTs",
        2
    ),

    /**
     * VERY_LOW - Very low confidence in the evidence
     *
     * Any estimate of effect is very uncertain.
     *
     * Typical sources:
     * - Case reports
     * - Expert opinion
     * - Severely flawed studies
     */
    VERY_LOW(
        "Very low quality",
        "Case reports, expert opinion, or severely flawed studies",
        1
    );

    private final String displayName;
    private final String description;
    private final int qualityScore;

    EvidenceLevel(String displayName, String description, int qualityScore) {
        this.displayName = displayName;
        this.description = description;
        this.qualityScore = qualityScore;
    }

    public String getDisplayName() {
        return displayName;
    }

    public String getDescription() {
        return description;
    }

    /**
     * Numeric quality score (1-4)
     * Higher = better quality evidence
     * Used for sorting and filtering
     */
    public int getQualityScore() {
        return qualityScore;
    }

    /**
     * Get evidence level from quality score
     */
    public static EvidenceLevel fromQualityScore(int score) {
        for (EvidenceLevel level : values()) {
            if (level.qualityScore == score) {
                return level;
            }
        }
        return VERY_LOW; // Default to lowest quality
    }

    @Override
    public String toString() {
        return displayName;
    }
}
