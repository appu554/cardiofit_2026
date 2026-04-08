package com.cardiofit.flink.models;

/**
 * Source classification for BP readings.
 *
 * HOME_CUFF: oscillometric home device - clinical grade by default.
 * CLINIC: in-office measurement - reference standard but infrequent.
 * CUFFLESS: wearable-derived estimate - NOT clinical grade until validated.
 * UNKNOWN: source metadata missing - treated as home for processing,
 *          flagged in output for data quality tracking.
 *
 * White-coat detection requires both CLINIC and HOME_CUFF readings.
 * Masked hypertension detection requires the same.
 * Cuffless readings compute separate ARV (arv_cuffless) for research
 * but do NOT contribute to clinical Decision Cards.
 */
public enum BPSource {
    HOME_CUFF(true),
    CLINIC(true),
    CUFFLESS(false),  // NOT clinical grade until upgrade
    UNKNOWN(true);    // assume clinical grade, flag for review

    private final boolean clinicalGrade;

    BPSource(boolean clinicalGrade) {
        this.clinicalGrade = clinicalGrade;
    }

    public boolean isClinicalGrade() { return clinicalGrade; }
}
