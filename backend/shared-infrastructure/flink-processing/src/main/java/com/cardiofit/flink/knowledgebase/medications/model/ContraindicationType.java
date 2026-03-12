package com.cardiofit.flink.knowledgebase.medications.model;

/**
 * Contraindication Type Classification
 *
 * Categorizes contraindications by severity and clinical action required.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-25
 */
public enum ContraindicationType {
    /**
     * ABSOLUTE: Medication must never be given under any circumstances.
     * Examples: Penicillin with documented anaphylaxis, Warfarin in active bleeding
     *
     * Clinical Action: Do not prescribe - reject order immediately
     */
    ABSOLUTE,

    /**
     * RELATIVE: Medication should generally be avoided but may be used with extreme caution.
     * Examples: Metformin in moderate renal impairment, Beta-blockers in asthma
     *
     * Clinical Action: Warn clinician, require override justification, enhanced monitoring
     */
    RELATIVE,

    /**
     * PRECAUTION: Special monitoring or dose adjustment required, but not contraindicated.
     * Examples: NSAIDs in heart failure, Statins in mild liver disease
     *
     * Clinical Action: Alert clinician, recommend monitoring, suggest dose modifications
     */
    PRECAUTION,

    /**
     * BLACK_BOX_WARNING: FDA black box warning applies to this patient population.
     * Examples: Antipsychotics in elderly dementia patients, Codeine in children post-tonsillectomy
     *
     * Clinical Action: Prominent warning, require acknowledgment, enhanced monitoring
     */
    BLACK_BOX_WARNING,

    /**
     * DISEASE_STATE: Contraindicated due to specific disease state or condition.
     * Examples: QT-prolonging drugs in long QT syndrome, Anticholinergics in narrow-angle glaucoma
     *
     * Clinical Action: Check for disease state, warn if present, suggest alternatives
     */
    DISEASE_STATE,

    /**
     * AGE_RELATED: Contraindicated for specific age groups (pediatric, geriatric).
     * Examples: Aspirin in children (Reye's syndrome), Benzodiazepines in elderly (falls risk)
     *
     * Clinical Action: Age-appropriate alert, suggest safer alternatives
     */
    AGE_RELATED,

    /**
     * PREGNANCY: Contraindicated during pregnancy or lactation.
     * Examples: ACE inhibitors in pregnancy (teratogenic), Isotretinoin in pregnancy
     *
     * Clinical Action: Pregnancy screening, contraception counseling, alternative therapy
     */
    PREGNANCY,

    /**
     * NONE: No contraindications identified.
     */
    NONE;

    /**
     * Check if this contraindication type requires order rejection.
     *
     * @return True if medication should be rejected
     */
    public boolean requiresRejection() {
        return this == ABSOLUTE;
    }

    /**
     * Check if this contraindication type requires clinician override.
     *
     * @return True if override documentation needed
     */
    public boolean requiresOverride() {
        return this == RELATIVE || this == BLACK_BOX_WARNING || this == DISEASE_STATE;
    }

    /**
     * Get alert severity level (1 = highest, 5 = lowest).
     *
     * @return Severity score for alerting priority
     */
    public int getAlertSeverity() {
        switch (this) {
            case ABSOLUTE: return 1;
            case BLACK_BOX_WARNING: return 1;
            case RELATIVE: return 2;
            case DISEASE_STATE: return 2;
            case AGE_RELATED: return 3;
            case PREGNANCY: return 2;
            case PRECAUTION: return 4;
            case NONE: return 5;
            default: return 5;
        }
    }

    /**
     * Get display emoji for visual alerts.
     *
     * @return Emoji character
     */
    public String getEmoji() {
        switch (this) {
            case ABSOLUTE: return "🛑";
            case BLACK_BOX_WARNING: return "⚫";
            case RELATIVE: return "⚠️";
            case DISEASE_STATE: return "🔴";
            case AGE_RELATED: return "👶👴";
            case PREGNANCY: return "🤰";
            case PRECAUTION: return "⚡";
            case NONE: return "✅";
            default: return "⚪";
        }
    }
}
