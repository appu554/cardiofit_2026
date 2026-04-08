package com.cardiofit.flink.thresholds;

import java.io.Serializable;

/**
 * Audit envelope tracking which layer resolved a clinical threshold for a CID rule.
 *
 * Every CIDAlert carries a list of ThresholdProvenance entries — one per threshold
 * consulted during rule evaluation. This enables clinical audit to answer:
 *   - Was a personalised threshold used?
 *   - Would the alert have fired at the population default?
 *   - Which guideline version was active at decision time?
 *
 * The {@code wouldFireAtDefault} flag is critical for patient safety review:
 * when a personalised threshold suppresses an alert that the population default
 * would have fired, the clinical team must be able to trace that decision.
 *
 * @see CIDThresholdResolver
 * @see CIDThresholdSet
 */
public class ThresholdProvenance implements Serializable {
    private static final long serialVersionUID = 1L;

    /** Which resolver layer provided the threshold value. */
    private String layer;           // "PATIENT" | "GUIDELINE" | "HARDCODED"

    /** Upstream source system. */
    private String source;          // "KB-20" | "KB-16" | "KB-4" | "HARDCODED"

    /** Threshold identifier (e.g., "POTASSIUM_THRESHOLD", "GLUCOSE_HYPO_THRESHOLD"). */
    private String thresholdKey;

    /** The actual threshold value used in this evaluation. */
    private double valueUsed;

    /** The hardcoded fallback value — always populated for comparison. */
    private double defaultValue;

    /**
     * True if the hardcoded default would have caused the alert to fire but the
     * personalised/guideline threshold suppressed it.
     *
     * For HALT rules this should NEVER be true (safety clamping prevents it).
     * For PAUSE/SOFT_FLAG rules this is legitimate and expected.
     */
    private boolean wouldFireAtDefault;

    /** Guideline version that produced this threshold (e.g., "KDIGO-2024-v2.1"). Null for hardcoded. */
    private String guidelineVersion;

    /** Epoch ms when the threshold was resolved. */
    private long resolvedAt;

    public ThresholdProvenance() {}

    /**
     * Factory for building a provenance record.
     */
    public static ThresholdProvenance create(
            String layer, String source, String thresholdKey,
            double valueUsed, double defaultValue,
            String guidelineVersion) {
        ThresholdProvenance p = new ThresholdProvenance();
        p.layer = layer;
        p.source = source;
        p.thresholdKey = thresholdKey;
        p.valueUsed = valueUsed;
        p.defaultValue = defaultValue;
        p.wouldFireAtDefault = false; // set by caller after rule evaluation
        p.guidelineVersion = guidelineVersion;
        p.resolvedAt = System.currentTimeMillis();
        return p;
    }

    /**
     * Convenience factory for hardcoded-layer provenance.
     */
    public static ThresholdProvenance hardcoded(String thresholdKey, double value) {
        return create("HARDCODED", "HARDCODED", thresholdKey, value, value, null);
    }

    // --- Getters / Setters ---

    public String getLayer() { return layer; }
    public void setLayer(String layer) { this.layer = layer; }

    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }

    public String getThresholdKey() { return thresholdKey; }
    public void setThresholdKey(String thresholdKey) { this.thresholdKey = thresholdKey; }

    public double getValueUsed() { return valueUsed; }
    public void setValueUsed(double valueUsed) { this.valueUsed = valueUsed; }

    public double getDefaultValue() { return defaultValue; }
    public void setDefaultValue(double defaultValue) { this.defaultValue = defaultValue; }

    public boolean isWouldFireAtDefault() { return wouldFireAtDefault; }
    public void setWouldFireAtDefault(boolean wouldFireAtDefault) { this.wouldFireAtDefault = wouldFireAtDefault; }

    public String getGuidelineVersion() { return guidelineVersion; }
    public void setGuidelineVersion(String guidelineVersion) { this.guidelineVersion = guidelineVersion; }

    public long getResolvedAt() { return resolvedAt; }
    public void setResolvedAt(long resolvedAt) { this.resolvedAt = resolvedAt; }
}
