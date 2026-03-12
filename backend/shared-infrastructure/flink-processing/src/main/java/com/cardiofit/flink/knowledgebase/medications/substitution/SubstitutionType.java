package com.cardiofit.flink.knowledgebase.medications.substitution;

/**
 * Types of therapeutic substitutions.
 *
 * @author CardioFit Module 3 Team
 * @version 1.0
 * @since 2025-10-24
 */
public enum SubstitutionType {
    /**
     * Generic equivalent of a brand-name medication.
     * Same active ingredient, different manufacturer.
     */
    GENERIC_EQUIVALENT,

    /**
     * Same pharmacologic class substitution.
     * Example: Cefepime → Ceftriaxone (both cephalosporins)
     */
    SAME_CLASS,

    /**
     * Different pharmacologic class substitution.
     * Used when patient has allergy to original class.
     * Example: Penicillin → Fluoroquinolone
     */
    DIFFERENT_CLASS,

    /**
     * Route of administration conversion.
     * Example: IV → PO (intravenous to oral)
     */
    ROUTE_CONVERSION,

    /**
     * Formulary-preferred alternative.
     * Non-formulary → Formulary equivalent
     */
    FORMULARY_SUBSTITUTION,

    /**
     * Dose form conversion.
     * Example: Tablet → Suspension
     */
    DOSE_FORM_CONVERSION,

    /**
     * Cost optimization substitution.
     * Higher cost → Lower cost alternative
     */
    COST_OPTIMIZATION
}
