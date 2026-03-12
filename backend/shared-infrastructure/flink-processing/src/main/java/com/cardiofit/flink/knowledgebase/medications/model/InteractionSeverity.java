package com.cardiofit.flink.knowledgebase.medications.model;

/**
 * Drug Interaction Severity Levels
 *
 * Categorizes drug-drug interactions by clinical significance and required action.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
public enum InteractionSeverity {
    /**
     * MAJOR: Life-threatening or requires immediate medical intervention.
     * Examples: Warfarin + NSAIDs (major bleeding risk), MAOIs + SSRIs (serotonin syndrome)
     *
     * Action Required: Consider alternative therapy, closely monitor, or adjust doses
     */
    MAJOR,

    /**
     * MODERATE: Clinically significant interaction requiring monitoring or dose adjustment.
     * Examples: ACE inhibitors + K+ supplements (hyperkalemia), Statins + Fibrates (myopathy risk)
     *
     * Action Required: Monitor patient closely, consider dose modifications
     */
    MODERATE,

    /**
     * MINOR: Limited clinical significance, usually managed with monitoring or patient education.
     * Examples: Caffeine + Ciprofloxacin (delayed caffeine clearance), Antacids + Tetracyclines (absorption)
     *
     * Action Required: Document interaction, educate patient, consider timing adjustments
     */
    MINOR,

    /**
     * UNKNOWN: Interaction severity not yet classified or insufficient evidence.
     * Used for newly discovered interactions or theoretical concerns.
     *
     * Action Required: Use clinical judgment, monitor for unexpected effects
     */
    UNKNOWN;

    /**
     * Get alert priority score (1 = highest, 5 = lowest)
     *
     * @return Priority score for clinical decision support alerts
     */
    public int getAlertPriority() {
        switch (this) {
            case MAJOR: return 1;
            case MODERATE: return 3;
            case MINOR: return 4;
            case UNKNOWN: return 5;
            default: return 5;
        }
    }

    /**
     * Check if this severity level requires immediate clinical intervention
     *
     * @return True if MAJOR severity
     */
    public boolean requiresImmediateIntervention() {
        return this == MAJOR;
    }

    /**
     * Check if this severity level requires monitoring
     *
     * @return True if MAJOR or MODERATE severity
     */
    public boolean requiresMonitoring() {
        return this == MAJOR || this == MODERATE;
    }

    /**
     * Get display color for UI alerts
     *
     * @return Color code (red, orange, yellow, gray)
     */
    public String getAlertColor() {
        switch (this) {
            case MAJOR: return "red";
            case MODERATE: return "orange";
            case MINOR: return "yellow";
            case UNKNOWN: return "gray";
            default: return "gray";
        }
    }

    /**
     * Get display emoji for visual alerts
     *
     * @return Emoji character
     */
    public String getEmoji() {
        switch (this) {
            case MAJOR: return "🔴";
            case MODERATE: return "🟠";
            case MINOR: return "🟡";
            case UNKNOWN: return "⚪";
            default: return "⚪";
        }
    }
}
