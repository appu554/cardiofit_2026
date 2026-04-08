package com.cardiofit.flink.thresholds;

import java.io.Serializable;

/**
 * Externalizable thresholds for Module 8 CID rules (CID-01 through CID-17).
 *
 * Organized by severity tier: HALT → PAUSE → SOFT_FLAG.
 * Each field's default matches the value currently hardcoded in the corresponding
 * evaluator class, guaranteeing zero regression when no external source is available.
 *
 * Data sources (three-layer resolver):
 *   Layer 1: KB-20 per-patient personalised targets (in ComorbidityState)
 *   Layer 2: KB-16/KB-4 guideline thresholds via BroadcastStream (this class)
 *   Layer 3: {@link #hardcodedDefaults()} — Tier 3 fallback
 *
 * @see CIDThresholdResolver
 * @see ThresholdProvenance
 */
public class CIDThresholdSet implements Serializable {
    private static final long serialVersionUID = 1L;

    // ════════════════════════════════════════════════════════════════════════
    // HALT tier — life-threatening, <5 min physician SLA
    // Safety invariant: personalized values can only TIGHTEN these thresholds
    // ════════════════════════════════════════════════════════════════════════

    /** CID-02: Hyperkalemia cascade — K+ threshold (mEq/L). KDIGO: 5.0 for CKD G4. */
    private double potassiumThreshold = 5.3;

    /** CID-03: Hypoglycemia masking — glucose threshold (mg/dL). ADA: 70 for elderly. */
    private double glucoseHypoThreshold = 60.0;

    /** CID-05: Severe hypotension — SBP absolute threshold (mmHg). */
    private double sbpHypotensionThreshold = 95.0;

    /** CID-05: Severe hypotension — SBP drop from 7-day average (mmHg). */
    private double sbpDropThreshold = 30.0;

    /** CID-01: Triple whammy AKI — eGFR acute decline threshold (%). */
    private double egfrDropThresholdPct = 15.0;

    /** CID-01: Triple whammy AKI — weight drop in 7 days (kg). */
    private double weightDropThresholdKg = 2.0;

    /** CID-05: Minimum antihypertensives for polypharmacy hypotension rule. */
    private int minAntihypertensives = 3;

    /** CID-01: eGFR below which AKI risk is critical (mL/min). Inline constant at HALTEvaluator:88. */
    private double egfrCriticalRenalThreshold = 45.0;

    // ════════════════════════════════════════════════════════════════════════
    // PAUSE tier — 48h physician review SLA
    // ════════════════════════════════════════════════════════════════════════

    /** CID-06: Thiazide glucose worsening — FBG delta threshold (mg/dL). */
    private double fbgDeltaThreshold = 15.0;

    /** CID-07: ACEi sustained eGFR decline — decline percent threshold. */
    private double egfrDeclineThresholdPct = 25.0;

    /** CID-07: ACEi eGFR dip window — weeks after initiation. */
    private int egfrDipWindowWeeks = 6;

    /** CID-09: GLP-1RA GI dehydration — weight drop threshold (kg). */
    private double weightDropGIThresholdKg = 1.5;

    /** CID-10: Concurrent deterioration — glucose worsening threshold (mg/dL). */
    private double glucoseWorseningThreshold = 10.0;

    /** CID-10: Concurrent deterioration — SBP worsening threshold (mmHg). */
    private double sbpWorseningThreshold = 10.0;

    // ════════════════════════════════════════════════════════════════════════
    // SOFT_FLAG tier — informational warnings on Decision Cards
    // ════════════════════════════════════════════════════════════════════════

    /** CID-12: Polypharmacy burden — medication count threshold. */
    private int polypharmacyThreshold = 8;

    /** CID-13: Elderly intensive BP target — age threshold (years). */
    private int elderlyAgeThreshold = 75;

    /** CID-13: Intensive SBP target (mmHg). */
    private double sbpTargetIntensive = 130.0;

    /** CID-14: Metformin near eGFR threshold — lower bound (mL/min). */
    private double egfrMetforminLow = 30.0;

    /** CID-14: Metformin near eGFR threshold — upper bound (mL/min). */
    private double egfrMetforminHigh = 35.0;

    /** CID-17: SGLT2i fasting period — duration threshold (hours). */
    private int fastingDurationThreshold = 16;

    // ════════════════════════════════════════════════════════════════════════
    // Governance
    // ════════════════════════════════════════════════════════════════════════

    /** Version identifier for audit trail (e.g., "KDIGO-2024-v2.1-extracted-2026-04-15"). */
    private String version;

    /** Epoch ms when this threshold set was loaded/resolved. */
    private long loadedAtEpochMs;

    // ════════════════════════════════════════════════════════════════════════
    // Tier 3 fallback — MUST match values in Module8HALT/PAUSE/SOFTFLAGEvaluator
    // ════════════════════════════════════════════════════════════════════════

    /**
     * Returns the exact threshold values currently hardcoded in M8 evaluators.
     * This is the Tier 3 (last-resort) fallback guaranteeing zero clinical regression.
     */
    public static CIDThresholdSet hardcodedDefaults() {
        CIDThresholdSet set = new CIDThresholdSet();
        set.setVersion("hardcoded-cid-v1.0.0");
        set.setLoadedAtEpochMs(System.currentTimeMillis());
        // All field defaults in the class declaration match evaluator constants,
        // so a freshly constructed instance IS the hardcoded defaults.
        return set;
    }

    // ════════════════════════════════════════════════════════════════════════
    // Getters / Setters — HALT
    // ════════════════════════════════════════════════════════════════════════

    public double getPotassiumThreshold() { return potassiumThreshold; }
    public void setPotassiumThreshold(double v) { this.potassiumThreshold = v; }

    public double getGlucoseHypoThreshold() { return glucoseHypoThreshold; }
    public void setGlucoseHypoThreshold(double v) { this.glucoseHypoThreshold = v; }

    public double getSbpHypotensionThreshold() { return sbpHypotensionThreshold; }
    public void setSbpHypotensionThreshold(double v) { this.sbpHypotensionThreshold = v; }

    public double getSbpDropThreshold() { return sbpDropThreshold; }
    public void setSbpDropThreshold(double v) { this.sbpDropThreshold = v; }

    public double getEgfrDropThresholdPct() { return egfrDropThresholdPct; }
    public void setEgfrDropThresholdPct(double v) { this.egfrDropThresholdPct = v; }

    public double getWeightDropThresholdKg() { return weightDropThresholdKg; }
    public void setWeightDropThresholdKg(double v) { this.weightDropThresholdKg = v; }

    public int getMinAntihypertensives() { return minAntihypertensives; }
    public void setMinAntihypertensives(int v) { this.minAntihypertensives = v; }

    public double getEgfrCriticalRenalThreshold() { return egfrCriticalRenalThreshold; }
    public void setEgfrCriticalRenalThreshold(double v) { this.egfrCriticalRenalThreshold = v; }

    // ════════════════════════════════════════════════════════════════════════
    // Getters / Setters — PAUSE
    // ════════════════════════════════════════════════════════════════════════

    public double getFbgDeltaThreshold() { return fbgDeltaThreshold; }
    public void setFbgDeltaThreshold(double v) { this.fbgDeltaThreshold = v; }

    public double getEgfrDeclineThresholdPct() { return egfrDeclineThresholdPct; }
    public void setEgfrDeclineThresholdPct(double v) { this.egfrDeclineThresholdPct = v; }

    public int getEgfrDipWindowWeeks() { return egfrDipWindowWeeks; }
    public void setEgfrDipWindowWeeks(int v) { this.egfrDipWindowWeeks = v; }

    public double getWeightDropGIThresholdKg() { return weightDropGIThresholdKg; }
    public void setWeightDropGIThresholdKg(double v) { this.weightDropGIThresholdKg = v; }

    public double getGlucoseWorseningThreshold() { return glucoseWorseningThreshold; }
    public void setGlucoseWorseningThreshold(double v) { this.glucoseWorseningThreshold = v; }

    public double getSbpWorseningThreshold() { return sbpWorseningThreshold; }
    public void setSbpWorseningThreshold(double v) { this.sbpWorseningThreshold = v; }

    // ════════════════════════════════════════════════════════════════════════
    // Getters / Setters — SOFT_FLAG
    // ════════════════════════════════════════════════════════════════════════

    public int getPolypharmacyThreshold() { return polypharmacyThreshold; }
    public void setPolypharmacyThreshold(int v) { this.polypharmacyThreshold = v; }

    public int getElderlyAgeThreshold() { return elderlyAgeThreshold; }
    public void setElderlyAgeThreshold(int v) { this.elderlyAgeThreshold = v; }

    public double getSbpTargetIntensive() { return sbpTargetIntensive; }
    public void setSbpTargetIntensive(double v) { this.sbpTargetIntensive = v; }

    public double getEgfrMetforminLow() { return egfrMetforminLow; }
    public void setEgfrMetforminLow(double v) { this.egfrMetforminLow = v; }

    public double getEgfrMetforminHigh() { return egfrMetforminHigh; }
    public void setEgfrMetforminHigh(double v) { this.egfrMetforminHigh = v; }

    public int getFastingDurationThreshold() { return fastingDurationThreshold; }
    public void setFastingDurationThreshold(int v) { this.fastingDurationThreshold = v; }

    // ════════════════════════════════════════════════════════════════════════
    // Getters / Setters — Governance
    // ════════════════════════════════════════════════════════════════════════

    public String getVersion() { return version; }
    public void setVersion(String v) { this.version = v; }

    public long getLoadedAtEpochMs() { return loadedAtEpochMs; }
    public void setLoadedAtEpochMs(long v) { this.loadedAtEpochMs = v; }
}
