package com.cardiofit.flink.thresholds;

import com.cardiofit.flink.models.CIDSeverity;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Three-layer clinical threshold resolver for Module 8 CID rules.
 *
 * Resolution order:
 *   Layer 1: Per-patient personalised value from KB-20 (in ComorbidityState)
 *   Layer 2: Guideline-level value from CIDThresholdSet (via BroadcastStream)
 *   Layer 3: Hardcoded default constant (zero-regression guarantee)
 *
 * <h3>HALT Safety Invariant</h3>
 * For HALT-severity rules, personalised/guideline values can only make the rule
 * MORE sensitive (more likely to fire), never LESS sensitive. This is enforced by
 * clamping: for thresholds where lower = more sensitive (K+, glucose, SBP, weight),
 * the resolver uses {@code Math.min(resolved, hardcoded)}. A misconfigured KB-20
 * returning K+=5.8 for a CKD patient would be rejected in favour of the hardcoded 5.3.
 *
 * For PAUSE/SOFT_FLAG rules, no clamping is applied — clinical context may legitimately
 * relax these thresholds (e.g., relaxing FBG delta for elderly patients).
 *
 * @see CIDThresholdSet
 * @see ThresholdProvenance
 */
public final class CIDThresholdResolver {
    private CIDThresholdResolver() {}

    private static final Logger LOG = LoggerFactory.getLogger(CIDThresholdResolver.class);

    /**
     * Resolve a threshold through the three-layer hierarchy.
     *
     * @param thresholdKey     identifier for audit trail (e.g., "POTASSIUM_THRESHOLD")
     * @param severity         CID rule severity — controls HALT safety clamping
     * @param personalizedValue Layer 1: per-patient value from ComorbidityState (nullable)
     * @param guidelineValue    Layer 2: population value from CIDThresholdSet/BroadcastStream
     * @param hardcodedDefault  Layer 3: compile-time constant from evaluator
     * @param lowerIsSensitive  true if a LOWER threshold makes the rule MORE sensitive
     *                          (K+, glucose, SBP, weight); false if HIGHER is more sensitive
     *                          (eGFR decline %). Only relevant for HALT clamping.
     * @return resolved threshold with provenance
     */
    public static ResolvedThreshold resolve(
            String thresholdKey,
            CIDSeverity severity,
            Double personalizedValue,
            double guidelineValue,
            double hardcodedDefault,
            boolean lowerIsSensitive) {

        // Layer 1: per-patient personalised value
        if (personalizedValue != null) {
            double clamped = applyHALTClamp(
                    personalizedValue, hardcodedDefault, severity, lowerIsSensitive, thresholdKey);
            ThresholdProvenance prov = ThresholdProvenance.create(
                    "PATIENT", "KB-20", thresholdKey,
                    clamped, hardcodedDefault, null);
            return new ResolvedThreshold(clamped, prov);
        }

        // Layer 2: guideline-level from BroadcastStream
        if (Math.abs(guidelineValue - hardcodedDefault) > 1e-9) {
            double clamped = applyHALTClamp(
                    guidelineValue, hardcodedDefault, severity, lowerIsSensitive, thresholdKey);
            ThresholdProvenance prov = ThresholdProvenance.create(
                    "GUIDELINE", "KB-16", thresholdKey,
                    clamped, hardcodedDefault, null);
            return new ResolvedThreshold(clamped, prov);
        }

        // Layer 3: hardcoded default
        ThresholdProvenance prov = ThresholdProvenance.hardcoded(thresholdKey, hardcodedDefault);
        return new ResolvedThreshold(hardcodedDefault, prov);
    }

    /**
     * Overload for the common case: lower-is-sensitive (most M8 thresholds).
     */
    public static ResolvedThreshold resolve(
            String thresholdKey,
            CIDSeverity severity,
            Double personalizedValue,
            double guidelineValue,
            double hardcodedDefault) {
        return resolve(thresholdKey, severity, personalizedValue,
                guidelineValue, hardcodedDefault, true);
    }

    /**
     * Apply HALT safety clamping.
     *
     * For HALT-severity rules:
     *   - If lowerIsSensitive=true:  use Math.min(value, hardcoded) — can only tighten downward
     *   - If lowerIsSensitive=false: use Math.max(value, hardcoded) — can only tighten upward
     *
     * For PAUSE/SOFT_FLAG: no clamping, return value as-is.
     */
    private static double applyHALTClamp(
            double value, double hardcodedDefault,
            CIDSeverity severity, boolean lowerIsSensitive,
            String thresholdKey) {

        if (severity != CIDSeverity.HALT) {
            return value;
        }

        double clamped;
        if (lowerIsSensitive) {
            // Lower threshold = more sensitive → clamp to min(value, hardcoded)
            clamped = Math.min(value, hardcodedDefault);
        } else {
            // Higher threshold = more sensitive → clamp to max(value, hardcoded)
            clamped = Math.max(value, hardcodedDefault);
        }

        if (Math.abs(clamped - value) > 1e-9) {
            LOG.warn("HALT safety clamp: {} requested {} but clamped to {} (hardcoded default). "
                    + "Personalised/guideline value would have relaxed a life-threatening rule.",
                    thresholdKey, value, clamped);
        }

        return clamped;
    }

    /**
     * Immutable result of threshold resolution.
     */
    public static final class ResolvedThreshold {
        private final double value;
        private final ThresholdProvenance provenance;

        public ResolvedThreshold(double value, ThresholdProvenance provenance) {
            this.value = value;
            this.provenance = provenance;
        }

        public double getValue() { return value; }
        public ThresholdProvenance getProvenance() { return provenance; }
    }
}
