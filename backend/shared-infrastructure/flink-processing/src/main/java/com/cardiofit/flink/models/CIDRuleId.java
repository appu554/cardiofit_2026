package com.cardiofit.flink.models;

/**
 * Canonical CID rule identifiers.
 * Follows DD#7 authoritative numbering.
 *
 * HALT rules (CID-01 to CID-05): Life-threatening interactions.
 * PAUSE rules (CID-06 to CID-10): Requires physician review.
 * SOFT_FLAG rules (CID-11 to CID-17): Informational warnings.
 */
public enum CIDRuleId {
    // HALT
    CID_01("Triple Whammy AKI", CIDSeverity.HALT),
    CID_02("Hyperkalemia Cascade", CIDSeverity.HALT),
    CID_03("Hypoglycemia Masking", CIDSeverity.HALT),
    CID_04("Euglycemic DKA", CIDSeverity.HALT),
    CID_05("Severe Hypotension Risk", CIDSeverity.HALT),

    // PAUSE
    CID_06("Thiazide Glucose Worsening", CIDSeverity.PAUSE),
    CID_07("ACEi Sustained eGFR Decline", CIDSeverity.PAUSE),
    CID_08("Statin Myopathy", CIDSeverity.PAUSE),
    CID_09("GLP1RA GI Dehydration", CIDSeverity.PAUSE),
    CID_10("Concurrent Glucose BP Deterioration", CIDSeverity.PAUSE),

    // SOFT_FLAG
    CID_11("Genital Infection SGLT2i", CIDSeverity.SOFT_FLAG),
    CID_12("Polypharmacy Burden", CIDSeverity.SOFT_FLAG),
    CID_13("Elderly Intensive BP Target", CIDSeverity.SOFT_FLAG),
    CID_14("Metformin Near eGFR Threshold", CIDSeverity.SOFT_FLAG),
    CID_15("SGLT2i NSAID Use", CIDSeverity.SOFT_FLAG),
    CID_16("Salt Sensitive Sodium Retaining Drug", CIDSeverity.SOFT_FLAG),
    CID_17("SGLT2i Fasting Period", CIDSeverity.SOFT_FLAG);

    private final String displayName;
    private final CIDSeverity severity;

    CIDRuleId(String displayName, CIDSeverity severity) {
        this.displayName = displayName;
        this.severity = severity;
    }

    public String getDisplayName() { return displayName; }
    public CIDSeverity getSeverity() { return severity; }
}
