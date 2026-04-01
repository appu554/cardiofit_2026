package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
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

    // Thresholds
    private static final double WEIGHT_DROP_THRESHOLD_KG = 2.0;     // CID-01
    private static final double EGFR_DROP_THRESHOLD_PCT = 15.0;     // CID-01
    private static final double POTASSIUM_THRESHOLD = 5.3;          // CID-02
    private static final double GLUCOSE_HYPO_THRESHOLD = 60.0;      // CID-03
    private static final double SBP_HYPOTENSION_THRESHOLD = 95.0;   // CID-05
    private static final double SBP_DROP_THRESHOLD = 30.0;          // CID-05
    private static final int MIN_ANTIHYPERTENSIVES = 3;             // CID-05

    /**
     * Evaluate all 5 HALT rules against current patient state.
     * @param state current ComorbidityState snapshot
     * @param eventTime event timestamp (for rolling buffer computations)
     * @return list of CIDAlerts (may be empty, may contain multiple)
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID01(state, eventTime, alerts);
        evaluateCID02(state, alerts);
        evaluateCID03(state, alerts);
        evaluateCID04(state, alerts);
        evaluateCID05(state, eventTime, alerts);

        return alerts;
    }

    /**
     * CID-01: Triple Whammy AKI.
     * ACEi/ARB + SGLT2i + diuretic + precipitant.
     * DD#7: precipitant = weight drop >2kg/7d OR eGFR drop >15%/14d OR illness.
     */
    private static void evaluateCID01(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        boolean hasDiuretic = state.hasAnyDrugClass("THIAZIDE", "LOOP_DIURETIC");

        if (!hasRASi || !hasSGLT2i || !hasDiuretic) return;

        // Check precipitant — per DD#7 spec
        boolean weightDrop = false;
        Double weightDelta = state.getWeightDelta7d(eventTime);
        if (weightDelta != null && weightDelta < -WEIGHT_DROP_THRESHOLD_KG) {
            weightDrop = true;
        }

        // Use 14-day acute eGFR decline (not baseline) per DD#7
        boolean eGFRDrop = false;
        Double eGFRAcuteDecline = state.getEGFRAcuteDeclinePercent14d();
        if (eGFRAcuteDecline != null && eGFRAcuteDecline >= EGFR_DROP_THRESHOLD_PCT - 1e-9) {
            eGFRDrop = true;
        }

        boolean illness = state.isSymptomReportedNauseaVomiting();

        if (!weightDrop && !eGFRDrop && !illness) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("THIAZIDE"));
        meds.addAll(state.getMedicationsByClass("LOOP_DIURETIC"));

        String precipitant = weightDrop ? "weight drop" : eGFRDrop ? "eGFR decline" : "illness/vomiting";
        String summary = String.format(
            "HALT: Triple whammy AKI risk. Patient on RASi + SGLT2i + diuretic with %s.",
            precipitant);
        String action = "Pause SGLT2i and diuretic. Urgent eGFR + creatinine within 48 hours. " +
            "If confirmed AKI: hold all three agents until renal function recovers.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_01, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-02: Hyperkalemia Cascade.
     * ACEi/ARB + finerenone + K+ > 5.3 AND rising (trajectory, not just threshold).
     */
    private static void evaluateCID02(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasRASi = state.hasAnyDrugClass("ACEI", "ARB");
        boolean hasFinerenone = state.hasDrugClass("FINERENONE");

        if (!hasRASi || !hasFinerenone) return;

        Double potassium = state.getLabValue("potassium");
        if (potassium == null || potassium < POTASSIUM_THRESHOLD - 1e-9) return;

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

        alerts.add(CIDAlert.create(CIDRuleId.CID_02, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-03: Hypoglycemia Masking.
     * Insulin/SU + beta-blocker + glucose <60 + no symptom report.
     */
    private static void evaluateCID03(ComorbidityState state, List<CIDAlert> alerts) {
        boolean hasHypoAgent = state.hasAnyDrugClass("INSULIN", "SU");
        boolean hasBetaBlocker = state.hasDrugClass("BETA_BLOCKER");

        if (!hasHypoAgent || !hasBetaBlocker) return;

        Double glucose = state.getLatestGlucose();
        if (glucose == null || glucose >= GLUCOSE_HYPO_THRESHOLD) return;

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

        alerts.add(CIDAlert.create(CIDRuleId.CID_03, state.getPatientId(),
            summary, meds, action, null));
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

        alerts.add(CIDAlert.create(CIDRuleId.CID_04, state.getPatientId(),
            summary, meds, action, null));
    }

    /**
     * CID-05: Severe Hypotension Risk.
     * >= 3 antihypertensives + SGLT2i + SBP < 95 (or SBP drop > 30 from 7d avg).
     */
    private static void evaluateCID05(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        boolean hasSGLT2i = state.hasDrugClass("SGLT2I");
        int ahCount = state.countAntihypertensives();

        if (!hasSGLT2i || ahCount < MIN_ANTIHYPERTENSIVES) return;

        Double sbp = state.getLatestSBP();
        if (sbp == null) return;

        boolean sbpLow = sbp < SBP_HYPOTENSION_THRESHOLD;

        boolean sbpDrop = false;
        Double sbpAvg = state.getSbpSevenDayAvg(eventTime);
        if (sbpAvg != null && (sbpAvg - sbp) > SBP_DROP_THRESHOLD) {
            sbpDrop = true;
        }

        if (!sbpLow && !sbpDrop) return;

        List<String> meds = new ArrayList<>(state.getActiveMedications().keySet());

        String summary = String.format(
            "HALT: Severe hypotension risk. SBP %.0f on %d antihypertensives + SGLT2i.", sbp, ahCount);
        String action = "Review all antihypertensive doses. Reduce or hold most recently added. " +
            "Check orthostatic BP. Assess hydration. If SBP <85: urgent review.";

        alerts.add(CIDAlert.create(CIDRuleId.CID_05, state.getPatientId(),
            summary, meds, action, null));
    }
}
