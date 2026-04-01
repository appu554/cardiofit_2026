package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.List;

/**
 * SOFT_FLAG-severity CID rule evaluator (CID-11 through CID-17).
 *
 * Warnings attached to Decision Cards. No correction loop pause.
 * Informational alerts that influence KB-23 card generation.
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module8SOFTFLAGEvaluator {
    private Module8SOFTFLAGEvaluator() {}

    private static final int POLYPHARMACY_THRESHOLD = 8;    // CID-12
    private static final int ELDERLY_AGE_THRESHOLD = 75;     // CID-13
    private static final double SBP_TARGET_INTENSIVE = 130.0; // CID-13
    private static final double EGFR_METFORMIN_LOW = 30.0;   // CID-14
    private static final double EGFR_METFORMIN_HIGH = 35.0;  // CID-14
    private static final int FASTING_DURATION_THRESHOLD = 16; // CID-17 hours

    /**
     * Evaluate all 7 SOFT_FLAG rules.
     * @param state current patient comorbidity state
     * @param sbpTargetMmHg patient's SBP target (from KB-20), nullable
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, Double sbpTargetMmHg) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID11(state, alerts);
        evaluateCID12(state, alerts);
        evaluateCID13(state, sbpTargetMmHg, alerts);
        evaluateCID14(state, alerts);
        evaluateCID15(state, alerts);
        evaluateCID16(state, alerts);
        evaluateCID17(state, alerts);

        return alerts;
    }

    /** CID-11: Genital infection history + SGLT2i. */
    private static void evaluateCID11(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isGenitalInfectionHistory()) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_11, state.getPatientId(),
            "WARNING: Patient has genital infection history. SGLT2i may increase recurrence.",
            state.getMedicationsByClass("SGLT2I"),
            "Monitor closely. Consider prophylactic antifungal if recurrence occurs.", null));
    }

    /** CID-12: Polypharmacy burden >= 8 medications. */
    private static void evaluateCID12(ComorbidityState state, List<CIDAlert> alerts) {
        int count = state.getActiveMedicationCount();
        if (count < POLYPHARMACY_THRESHOLD) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_12, state.getPatientId(),
            String.format("WARNING: Polypharmacy burden high (%d daily medications). " +
                "Consider deprescribing review before adding.", count),
            new ArrayList<>(state.getActiveMedications().keySet()),
            "Review medication list. Assess adherence risk. Consider deprescribing.", null));
    }

    /** CID-13: Elderly + intensive BP target. */
    private static void evaluateCID13(ComorbidityState state, Double sbpTarget, List<CIDAlert> alerts) {
        if (state.getAge() == null || state.getAge() < ELDERLY_AGE_THRESHOLD) return;
        if (sbpTarget == null || sbpTarget >= SBP_TARGET_INTENSIVE) return;

        boolean hasRisk = false;
        if (state.getEGFRCurrent() != null && state.getEGFRCurrent() < 45.0) hasRisk = true;
        if (state.isFallsHistory()) hasRisk = true;
        if (state.isOrthostaticHypotension()) hasRisk = true;

        if (!hasRisk) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_13, state.getPatientId(),
            String.format("WARNING: Intensive SBP target (<%.0f) may increase adverse events " +
                "in this elderly patient (age %d). Consider relaxing to <140 mmHg.", sbpTarget, state.getAge()),
            List.of(),
            "Review BP target per ADA 2026 frailty guidance. Consider <140 mmHg.", null));
    }

    /** CID-14: Metformin + eGFR 30-35 + declining trajectory. */
    private static void evaluateCID14(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("METFORMIN")) return;
        Double eGFR = state.getEGFRCurrent();
        if (eGFR == null) return;
        if (eGFR < EGFR_METFORMIN_LOW || eGFR > EGFR_METFORMIN_HIGH) return;

        // Check declining trajectory
        Double decline = state.getEGFRDeclinePercent();
        if (decline == null || decline <= 0) return; // not declining

        alerts.add(CIDAlert.create(CIDRuleId.CID_14, state.getPatientId(),
            String.format("WARNING: eGFR approaching metformin threshold (30). Current: %.0f. " +
                "Plan for dose reduction at 30-45, discontinuation at <30.", eGFR),
            state.getMedicationsByClass("METFORMIN"),
            "Consider SGLT2i as glucose-lowering replacement if eGFR permits.", null));
    }

    /** CID-15: SGLT2i + NSAID use. */
    private static void evaluateCID15(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I") || !state.hasDrugClass("NSAID")) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("NSAID"));

        alerts.add(CIDAlert.create(CIDRuleId.CID_15, state.getPatientId(),
            "WARNING: NSAIDs + SGLT2i increase AKI risk and reduce SGLT2i renal benefit.",
            meds,
            "Advise NSAID avoidance. If analgesia needed: paracetamol preferred.", null));
    }

    /** CID-16: Salt-sensitive patient + sodium-retaining drug. */
    private static void evaluateCID16(ComorbidityState state, List<CIDAlert> alerts) {
        if (!"HIGH".equalsIgnoreCase(state.getSaltSensitivityPhenotype())) return;
        if (!state.hasAnyDrugClass("CORTICOSTEROID", "FLUDROCORTISONE")) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("CORTICOSTEROID"));
        meds.addAll(state.getMedicationsByClass("FLUDROCORTISONE"));

        alerts.add(CIDAlert.create(CIDRuleId.CID_16, state.getPatientId(),
            "WARNING: Patient is salt-sensitive. Sodium-retaining drug may elevate BP.",
            meds,
            "Monitor BP closely after initiation.", null));
    }

    /** CID-17: SGLT2i + fasting period > 16 hours. */
    private static void evaluateCID17(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isActiveFastingPeriod()) return;
        if (state.getActiveFastingDurationHours() < FASTING_DURATION_THRESHOLD) return;

        alerts.add(CIDAlert.create(CIDRuleId.CID_17, state.getPatientId(),
            String.format("WARNING: Extended fasting (%dh) on SGLT2i increases DKA and dehydration risk.",
                state.getActiveFastingDurationHours()),
            state.getMedicationsByClass("SGLT2I"),
            "Advise adequate hydration during non-fasting hours. Consider holding SGLT2i during fasts >20h.", null));
    }
}
