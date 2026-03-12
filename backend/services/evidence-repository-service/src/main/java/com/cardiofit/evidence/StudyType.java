package com.cardiofit.evidence;

/**
 * Study Design Types
 *
 * Hierarchical classification of medical research study designs
 * ordered by typical evidence quality (strongest to weakest).
 *
 * Study design is a primary factor in GRADE evidence level assessment.
 */
public enum StudyType {

    /**
     * Systematic Review / Meta-Analysis
     *
     * Comprehensive analysis combining results from multiple studies.
     * Typically provides the highest quality evidence when well-conducted.
     *
     * Characteristics:
     * - Systematic literature search
     * - Statistical combination of results (meta-analysis)
     * - Quality assessment of included studies
     */
    SYSTEMATIC_REVIEW(
        "Systematic Review/Meta-Analysis",
        "Comprehensive analysis of multiple studies",
        7
    ),

    /**
     * Randomized Controlled Trial (RCT)
     *
     * Experimental study with randomized treatment assignment.
     * Gold standard for evaluating treatment efficacy.
     *
     * Characteristics:
     * - Random allocation to treatment/control
     * - Prospective data collection
     * - Controlled intervention
     */
    RANDOMIZED_CONTROLLED_TRIAL(
        "Randomized Controlled Trial",
        "Experimental study with randomization",
        6
    ),

    /**
     * Cohort Study
     *
     * Observational study following groups over time.
     * Good for studying outcomes and natural history.
     *
     * Characteristics:
     * - Prospective or retrospective
     * - Exposure-outcome relationship
     * - No intervention
     */
    COHORT_STUDY(
        "Cohort Study",
        "Observational follow-up of groups over time",
        5
    ),

    /**
     * Case-Control Study
     *
     * Observational study comparing cases (with disease) to controls (without).
     * Useful for studying rare diseases or outcomes.
     *
     * Characteristics:
     * - Retrospective comparison
     * - Efficient for rare conditions
     * - Risk factor identification
     */
    CASE_CONTROL(
        "Case-Control Study",
        "Comparison of cases with disease to controls without",
        4
    ),

    /**
     * Cross-Sectional Study
     *
     * Observational study measuring exposures and outcomes at single time point.
     * Useful for prevalence estimation.
     *
     * Characteristics:
     * - Snapshot in time
     * - Cannot establish causation
     * - Prevalence data
     */
    CROSS_SECTIONAL(
        "Cross-Sectional Study",
        "Snapshot measurement at single time point",
        3
    ),

    /**
     * Case Series / Case Report
     *
     * Descriptive report of clinical observations.
     * Lowest tier of evidence but valuable for rare conditions.
     *
     * Characteristics:
     * - No control group
     * - Small sample size
     * - Hypothesis-generating
     */
    CASE_SERIES(
        "Case Series/Case Report",
        "Descriptive clinical observations",
        2
    ),

    /**
     * Expert Opinion / Guidelines
     *
     * Expert consensus or clinical practice guidelines.
     * Lowest evidence level in GRADE framework.
     *
     * Characteristics:
     * - No systematic research
     * - Consensus-based
     * - May synthesize other evidence
     */
    EXPERT_OPINION(
        "Expert Opinion/Clinical Guidelines",
        "Consensus recommendations or expert judgment",
        1
    );

    private final String displayName;
    private final String description;
    private final int hierarchyLevel;

    StudyType(String displayName, String description, int hierarchyLevel) {
        this.displayName = displayName;
        this.description = description;
        this.hierarchyLevel = hierarchyLevel;
    }

    public String getDisplayName() {
        return displayName;
    }

    public String getDescription() {
        return description;
    }

    /**
     * Hierarchy level (1-7)
     * Higher = stronger study design
     * Used for evidence quality assessment
     */
    public int getHierarchyLevel() {
        return hierarchyLevel;
    }

    /**
     * Suggest initial GRADE evidence level based on study type
     *
     * Note: This is a starting point. GRADE methodology requires
     * additional assessment of study quality, consistency, and directness.
     */
    public EvidenceLevel suggestEvidenceLevel() {
        switch (this) {
            case SYSTEMATIC_REVIEW:
            case RANDOMIZED_CONTROLLED_TRIAL:
                return EvidenceLevel.HIGH; // Starting point, may be downgraded
            case COHORT_STUDY:
            case CASE_CONTROL:
                return EvidenceLevel.MODERATE; // Starting point for observational
            case CROSS_SECTIONAL:
            case CASE_SERIES:
                return EvidenceLevel.LOW;
            case EXPERT_OPINION:
                return EvidenceLevel.VERY_LOW;
            default:
                return EvidenceLevel.VERY_LOW;
        }
    }

    /**
     * Get study type from hierarchy level
     */
    public static StudyType fromHierarchyLevel(int level) {
        for (StudyType type : values()) {
            if (type.hierarchyLevel == level) {
                return type;
            }
        }
        return EXPERT_OPINION; // Default to lowest level
    }

    @Override
    public String toString() {
        return displayName;
    }
}
