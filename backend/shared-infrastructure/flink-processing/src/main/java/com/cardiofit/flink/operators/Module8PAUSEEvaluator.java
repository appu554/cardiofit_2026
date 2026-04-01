package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.List;

/**
 * PAUSE-severity CID rule evaluator (CID-06 through CID-10).
 *
 * Correction loop paused. Physician review within 48 hours.
 * Not immediately life-threatening but requires intervention.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8PAUSEEvaluator {
    private Module8PAUSEEvaluator() {}

    private static final double FBG_DELTA_THRESHOLD = 15.0;        // CID-06 mg/dL in 14d
    private static final double EGFR_DECLINE_THRESHOLD = 25.0;     // CID-07 percentage
    private static final double EGFR_DIP_WINDOW_WEEKS = 6.0;       // CID-07
    private static final double WEIGHT_DROP_GI_THRESHOLD = 1.5;    // CID-09 kg in 7d
    private static final double GLUCOSE_WORSENING_THRESHOLD = 10.0; // CID-10 mg/dL
    private static final double SBP_WORSENING_THRESHOLD = 10.0;    // CID-10 mmHg

    public static List<CIDAlert> evaluate(ComorbidityState state, long eventTime) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID06(state, eventTime, alerts);
        evaluateCID07(state, alerts);
        evaluateCID08(state, alerts);
        evaluateCID09(state, eventTime, alerts);
        evaluateCID10(state, eventTime, alerts);

        return alerts;
    }

    /** CID-06: Thiazide + FBG increase > 15 mg/dL in 14 days. */
    private static void evaluateCID06(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("THIAZIDE")) return;
        Double fbgDelta = state.getFBGDelta14d(eventTime);
        if (fbgDelta == null || fbgDelta < FBG_DELTA_THRESHOLD - 1e-9) return;

        List<String> meds = state.getMedicationsByClass("THIAZIDE");
        String summary = String.format(
            "PAUSE: Thiazide-associated glucose rise. FBG increased %.0f mg/dL in 14 days.", fbgDelta);
        String action = "Consider: CCB substitution, dose reduction, or SGLT2i addition.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_06, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-07: ACEi/ARB + eGFR drop > 25% + > 6 weeks since initiation. */
    private static void evaluateCID07(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasAnyDrugClass("ACEI", "ARB")) return;
        Double decline = state.getEGFRDeclinePercent();
        Double weeks = state.getWeeksSinceEGFRBaseline();
        if (decline == null || weeks == null) return;
        if (decline < EGFR_DECLINE_THRESHOLD - 1e-9) return;
        if (weeks < EGFR_DIP_WINDOW_WEEKS) return; // within expected dip window

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("ACEI"));
        meds.addAll(state.getMedicationsByClass("ARB"));

        String summary = String.format(
            "PAUSE: Sustained eGFR decline on RASi. eGFR %.0f vs baseline %.0f (%.0f%% decline). " +
            "Expected dip window (6 weeks) has passed.", state.getEGFRCurrent(), state.getEGFRBaseline(), decline);
        String action = "Renal ultrasound with Doppler. If stenosis: stop RASi, switch to CCB. " +
            "If no stenosis: reduce RASi dose by 50%, recheck eGFR in 4 weeks.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_07, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-08: Statin + new muscle symptoms. */
    private static void evaluateCID08(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("STATIN")) return;
        if (!state.isSymptomReportedMusclePain()) return;

        List<String> meds = state.getMedicationsByClass("STATIN");
        String summary = "PAUSE: Possible statin myopathy. Patient reports muscle pain/weakness.";
        String action = "Order CK. If CK <5x ULN: consider statin switch. If 5-10x: hold, recheck 2 weeks. " +
            "If >10x: discontinue, urgent rhabdomyolysis review.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_08, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-09: GLP-1RA + persistent GI (≥48h) + weight drop > 1.5 kg/7d. */
    private static void evaluateCID09(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("GLP1RA")) return;
        // R6 fix: DD#7 requires nausea/vomiting persisting ≥48 hours, not just boolean flag
        if (!state.isNauseaPersistent(eventTime, 48L * 3600_000L)) return;
        Double weightDelta = state.getWeightDelta7d(eventTime);
        if (weightDelta == null || weightDelta > -WEIGHT_DROP_GI_THRESHOLD) return;

        List<String> meds = state.getMedicationsByClass("GLP1RA");
        String summary = String.format(
            "PAUSE: GLP-1RA GI intolerance with possible dehydration. Weight change: %.1f kg in 7 days.",
            weightDelta);
        String action = "Hold GLP-1RA dose escalation. Assess hydration. " +
            "If concurrent SGLT2i/diuretic: assess renal function.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_09, state.getPatientId(), summary, meds, action, null));
    }

    /** CID-10: Concurrent glucose AND BP deterioration without medication change in past 14d. */
    private static void evaluateCID10(ComorbidityState state, long eventTime, List<CIDAlert> alerts) {
        // R7 fix: DD#7 says "without medication change" — skip if meds changed within 14 days
        if (state.hadMedicationChangeWithin(eventTime, 14L * 86_400_000L)) return;

        Double fbgDelta = state.getFBGDelta14d(eventTime);
        boolean glucoseWorsening = fbgDelta != null && fbgDelta >= GLUCOSE_WORSENING_THRESHOLD - 1e-9;

        Double sbpDelta = state.getSbpDelta14d(eventTime);
        boolean bpWorsening = sbpDelta != null && sbpDelta >= SBP_WORSENING_THRESHOLD - 1e-9;

        if (!glucoseWorsening || !bpWorsening) return;

        String summary = String.format(
            "PAUSE: Concurrent deterioration. FBG +%.0f mg/dL, SBP +%.0f mmHg over 14 days " +
            "without medication change.", fbgDelta, sbpDelta);
        String action = "Review adherence. Assess lifestyle factors (diet, sleep, stress). " +
            "Consider medication intensification across both domains.";
        alerts.add(CIDAlert.create(CIDRuleId.CID_10, state.getPatientId(), summary, List.of(), action, null));
    }
}
