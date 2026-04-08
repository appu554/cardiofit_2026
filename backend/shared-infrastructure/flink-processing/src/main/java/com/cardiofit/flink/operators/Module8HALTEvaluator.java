package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.thresholds.CIDThresholdResolver;
import com.cardiofit.flink.thresholds.CIDThresholdResolver.ResolvedThreshold;
import com.cardiofit.flink.thresholds.CIDThresholdSet;
import com.cardiofit.flink.thresholds.ThresholdProvenance;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * HALT-severity CID rule evaluator (CID-01 through CID-05).
 *
 * Life-threatening comorbidity interactions. Each HALT fires
 * immediately to ingestion.safety-critical with <5 min physician SLA.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8HALTEvaluator {
    private Module8HALTEvaluator() {}

    // Thresholds — kept as compile-time constants for backward compatibility.
    // New code should use the CIDThresholdSet overload instead.
    private static final double WEIGHT_DROP_THRESHOLD_KG = 2.0;     // CID-01
    private static final double EGFR_DROP_THRESHOLD_PCT = 15.0;     // CID-01
    private static final double POTASSIUM_THRESHOLD = 5.3;          // CID-02
    private static final double GLUCOSE_HYPO_THRESHOLD = 60.0;      // CID-03
    private static final double SBP_HYPOTENSION_THRESHOLD = 95.0;   // CID-05
    private static final double SBP_DROP_THRESHOLD = 30.0;          // CID-05
    private static final int MIN_ANTIHYPERTENSIVES = 3;             // CID-05

    /**
     * Evaluate all 5 HALT rules with externalized thresholds and per-patient personalization.
     * @param state current ComorbidityState snapshot
     * @param eventTime event timestamp (for rolling buffer computations)
     * @param thresholds externalized threshold set (Layer 2/3)
     * @return list of CIDAlerts with ThresholdProvenance attached
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime,
                                           CIDThresholdSet thresholds) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID01(state, eventTime, thresholds, alerts);
        evaluateCID02(state, thresholds, alerts);
        evaluateCID03(state, thresholds, alerts);
        evaluateCID04(state, alerts);
        evaluateCID05(state, eventTime, thresholds, alerts);

        return alerts;
    }

    /**
     * Backward-compatible overload — delegates with hardcoded defaults.
     * Existing tests call this signature and require zero changes.
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime) {
        return evaluate(state, eventTime, CIDThresholdSet.hardcodedDefaults());
    }

    /**
     * CID-01: Triple Whammy AKI.
     * ACEi/ARB + SGLT2i + diuretic + precipitant.
     * DD#7: precipitant = weight drop >2kg/7d OR eGFR drop >15%/14d OR illness.
     */
    private static void evaluateCID01(ComorbidityState state, long eventTime,
                                      CIDThresholdSet thresholds, List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        boolean hasDiuretic = state.hasAnyDrugClass("THIAZIDE", "LOOP_DIURETIC");

        if (!hasRASi || !hasSGLT2i || !hasDiuretic) return;

        List<ThresholdProvenance> provenance = new ArrayList<>();

        // Resolve weight drop threshold through three-layer hierarchy
        ResolvedThreshold weightThreshold = CIDThresholdResolver.resolve(
                "WEIGHT_DROP_THRESHOLD_KG", CIDSeverity.HALT,
                state.getPersonalizedEGFRDropThresholdPct() != null ? null : null, // no per-patient weight threshold yet
                thresholds.getWeightDropThresholdKg(),
                WEIGHT_DROP_THRESHOLD_KG);

        // Resolve eGFR drop threshold — lower % = fires on smaller declines = more sensitive
        ResolvedThreshold egfrThreshold = CIDThresholdResolver.resolve(
                "EGFR_DROP_THRESHOLD_PCT", CIDSeverity.HALT,
                state.getPersonalizedEGFRDropThresholdPct(),
                thresholds.getEgfrDropThresholdPct(),
                EGFR_DROP_THRESHOLD_PCT);

        // Check precipitant — per DD#7 spec
        boolean weightDrop = false;
        Double weightDelta = state.getWeightDelta7d(eventTime);
        if (weightDelta != null && weightDelta < -weightThreshold.getValue()) {
            weightDrop = true;
        }
        provenance.add(weightThreshold.getProvenance());

        // Use 14-day acute eGFR decline (not baseline) per DD#7
        boolean eGFRDrop = false;
        Double eGFRAcuteDecline = state.getEGFRAcuteDeclinePercent14d();
        if (eGFRAcuteDecline != null && eGFRAcuteDecline >= egfrThreshold.getValue() - 1e-9) {
            eGFRDrop = true;
        }
        provenance.add(egfrThreshold.getProvenance());

        boolean illness = state.isSymptomReportedNauseaVomiting();

        if (!weightDrop && !eGFRDrop && !illness) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("THIAZIDE"));
        meds.addAll(state.getMedicationsByClass("LOOP_DIURETIC"));

        String precipitant = weightDrop ? "weight drop" : eGFRDrop ? "eGFR decline" : "illness/vomiting";

        // Resolve eGFR critical renal threshold
        ResolvedThreshold renalThreshold = CIDThresholdResolver.resolve(
                "EGFR_CRITICAL_RENAL_THRESHOLD", CIDSeverity.HALT,
                null, // no per-patient override for this threshold
                thresholds.getEgfrCriticalRenalThreshold(),
                45.0);
        provenance.add(renalThreshold.getProvenance());

        boolean criticalRenal = eGFRDrop
                && state.getEGFRCurrent() != null
                && state.getEGFRCurrent() < renalThreshold.getValue();

        String summary = String.format(
            "HALT: Triple whammy AKI risk. Patient on RASi + SGLT2i + diuretic with %s.%s",
            precipitant,
            criticalRenal ? " CRITICAL: eGFR " + String.format("%.0f", state.getEGFRCurrent())
                    + " with acute decline — immediate AKI risk." : "");
        String action = criticalRenal
            ? "IMMEDIATE: Hold SGLT2i and diuretic NOW. Stat eGFR + creatinine + potassium within 4 hours. "
                + "If confirmed AKI: hold all three agents until renal function recovers."
            : "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours. "
                + "If confirmed AKI: hold all three agents until renal function recovers.";

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_01, state.getPatientId(),
            summary, meds, action, null);
        alert.setThresholdProvenance(provenance);
        alerts.add(alert);
    }

    /**
     * CID-02: Hyperkalemia Cascade.
     * ACEi/ARB + finerenone + K+ > 5.3 AND rising (trajectory, not just threshold).
     */
    private static void evaluateCID02(ComorbidityState state, CIDThresholdSet thresholds,
                                      List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasFinerenone = state.hasDrugClass("FINERENONE");

        if (!hasRASi || !hasFinerenone) return;

        // Resolve potassium threshold through three-layer hierarchy
        ResolvedThreshold kThreshold = CIDThresholdResolver.resolve(
                "POTASSIUM_THRESHOLD", CIDSeverity.HALT,
                state.getPersonalizedPotassiumThreshold(),
                thresholds.getPotassiumThreshold(),
                POTASSIUM_THRESHOLD);

        Double potassium = state.getLabValue("potassium");
        if (potassium == null || potassium < kThreshold.getValue() - 1e-9) return;

        // Must be RISING — stable elevated K+ is a different clinical scenario
        Double previousK = state.getPreviousPotassium();
        if (previousK == null || potassium <= previousK) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));
        meds.addAll(state.getMedicationsByClass("FINERENONE"));

        String summary = String.format(
            "HALT: Hyperkalemia cascade. K+ %.1f on RASi + finerenone.", potassium);
        String action = "Hold finerenone immediately. Recheck K+ in 48-72 hours. " +
            "If K+ >5.5: hold ACEi/ARB dose. If K+ >6.0: emergency protocol.";

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_02, state.getPatientId(),
            summary, meds, action, null);
        alert.setThresholdProvenance(List.of(kThreshold.getProvenance()));
        alerts.add(alert);
    }

    /**
     * CID-03: Hypoglycemia Masking.
     * Insulin/SU + beta-blocker + glucose <60 + no symptom report.
     */
    private static void evaluateCID03(ComorbidityState state, CIDThresholdSet thresholds,
                                      List<CIDAlert> alerts) {
        boolean hasHypoAgent = state.hasAnyDrugClass("INSULIN", "SU");
        boolean hasBetaBlocker = state.hasDrugClass("BETA_BLOCKER");

        if (!hasHypoAgent || !hasBetaBlocker) return;

        // Resolve glucose hypo threshold — higher threshold = fires for more patients = more sensitive
        ResolvedThreshold glucoseThreshold = CIDThresholdResolver.resolve(
                "GLUCOSE_HYPO_THRESHOLD", CIDSeverity.HALT,
                state.getPersonalizedGlucoseHypoThreshold(),
                thresholds.getGlucoseHypoThreshold(),
                GLUCOSE_HYPO_THRESHOLD,
                false);

        Double glucose = state.getLatestGlucose();
        if (glucose == null || glucose >= glucoseThreshold.getValue()) return;

        // Masking = no symptoms despite low glucose
        if (state.isSymptomReportedHypoglycemia()) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("INSULIN"));
        meds.addAll(state.getMedicationsByClass("SU"));
        meds.addAll(state.getMedicationsByClass("BETA_BLOCKER"));

        String summary = String.format(
            "HALT: Masked hypoglycemia. Glucose %.0f mg/dL with no symptoms. " +
            "Beta-blocker masking adrenergic warning signs.", glucose);
        String action = "Deprescribe sulfonylurea if present. If insulin: reduce basal by 20%. " +
            "Consider switching beta-blocker to carvedilol. Educate on neuroglycopenic symptoms.";

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_03, state.getPatientId(),
            summary, meds, action, null);
        alert.setThresholdProvenance(List.of(glucoseThreshold.getProvenance()));
        alerts.add(alert);
    }

    /**
     * CID-04: Euglycemic DKA.
     * SGLT2i + (nausea/vomiting OR keto diet OR insulin reduction in LADA).
     */
    private static void evaluateCID04(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        if (!hasSGLT2i) return;

        boolean hasTrigger = state.isSymptomReportedNauseaVomiting()
            || state.isSymptomReportedKetoDiet();
        // Note: insulin dose reduction detection requires comparing current vs previous dose.
        // Deferred to Phase 2 — requires medication change tracking in state.

        if (!hasTrigger) return;

        List<String> meds = state.getMedicationsByClass("SGLT2I");
        String trigger = state.isSymptomReportedNauseaVomiting() ? "nausea/vomiting" : "keto/low-carb diet";

        String summary = String.format(
            "HALT: Euglycemic DKA risk. Patient on SGLT2i with %s. " +
            "Glucose may be NORMAL despite ketoacidosis.", trigger);
        String action = "Hold SGLT2i immediately. Check blood ketones urgently. " +
            "If ketones elevated: emergency department. If no symptoms: hold 48h, resume after trigger resolves.";

        // CID-04 has no externalized thresholds — no provenance to attach
        alerts.add(CIDAlert.create(CIDRuleId.CID_04, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-05: Severe Hypotension Risk.
     * >= 3 antihypertensives + SGLT2i + SBP < 95 (or SBP drop > 30 from 7d avg).
     */
    private static void evaluateCID05(ComorbidityState state, long eventTime,
                                      CIDThresholdSet thresholds, List<CIDAlert> alerts) {
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        int ahCount = state.countAntihypertensives();

        if (!hasSGLT2i || ahCount < thresholds.getMinAntihypertensives()) return;

        Double sbp = state.getLatestSBP();
        if (sbp == null) return;

        List<ThresholdProvenance> provenance = new ArrayList<>();

        // Resolve SBP hypotension threshold — higher threshold = fires for more patients = more sensitive
        ResolvedThreshold sbpThreshold = CIDThresholdResolver.resolve(
                "SBP_HYPOTENSION_THRESHOLD", CIDSeverity.HALT,
                state.getPersonalizedSBPHypotensionThreshold(),
                thresholds.getSbpHypotensionThreshold(),
                SBP_HYPOTENSION_THRESHOLD,
                false);
        provenance.add(sbpThreshold.getProvenance());

        boolean sbpLow = sbp < sbpThreshold.getValue();

        // Resolve SBP drop threshold
        ResolvedThreshold dropThreshold = CIDThresholdResolver.resolve(
                "SBP_DROP_THRESHOLD", CIDSeverity.HALT,
                null, // no per-patient override for drop threshold
                thresholds.getSbpDropThreshold(),
                SBP_DROP_THRESHOLD);
        provenance.add(dropThreshold.getProvenance());

        boolean sbpDrop = false;
        Double sbpAvg = state.getSbpSevenDayAvg(eventTime);
        if (sbpAvg != null && (sbpAvg - sbp) > dropThreshold.getValue()) {
            sbpDrop = true;
        }

        if (!sbpLow && !sbpDrop) return;

        List<String> meds = new ArrayList<>(state.getActiveMedications().keySet());

        String summary = String.format(
            "HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.", sbp, ahCount);
        String action = "Review all antihypertensive doses. Reduce or hold most recently added. " +
            "Check orthostatic BP. Assess hydration. If SBP <85: urgent review.";

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_05, state.getPatientId(),
            summary, meds, action, null);
        alert.setThresholdProvenance(provenance);
        alerts.add(alert);
    }
}
