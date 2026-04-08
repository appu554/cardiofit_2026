package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.thresholds.CIDThresholdResolver;
import com.cardiofit.flink.thresholds.CIDThresholdResolver.ResolvedThreshold;
import com.cardiofit.flink.thresholds.CIDThresholdSet;
import com.cardiofit.flink.thresholds.ThresholdProvenance;
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
     * Evaluate all 7 SOFT_FLAG rules with externalized thresholds.
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, Double sbpTargetMmHg,
                                           CIDThresholdSet thresholds) {
        List<CIDAlert> alerts = new ArrayList<>();
        if (state == null) return alerts;

        evaluateCID11(state, alerts);
        evaluateCID12(state, thresholds, alerts);
        evaluateCID13(state, sbpTargetMmHg, thresholds, alerts);
        evaluateCID14(state, thresholds, alerts);
        evaluateCID15(state, alerts);
        evaluateCID16(state, alerts);
        evaluateCID17(state, thresholds, alerts);

        return alerts;
    }

    /**
     * Backward-compatible overload — delegates with hardcoded defaults.
     */
    public static List<CIDAlert> evaluate(ComorbidityState state, Double sbpTargetMmHg) {
        return evaluate(state, sbpTargetMmHg, CIDThresholdSet.hardcodedDefaults());
    }

    /** CID-11: Genital infection history + SGLT2i. */
    private static void evaluateCID11(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isGenitalInfectionHistory()) return;

        // Purely boolean — no externalized thresholds, no provenance
        alerts.add(CIDAlert.create(CIDRuleId.CID_11, state.getPatientId(),
            "WARNING: Patient has genital infection history. SGLT2i may increase recurrence.",
            state.getMedicationsByClass("SGLT2I"),
            "Monitor closely. Consider prophylactic antifungal if recurrence occurs.", null));
    }

    /** CID-12: Polypharmacy burden >= threshold medications. */
    private static void evaluateCID12(ComorbidityState state, CIDThresholdSet thresholds,
                                      List<CIDAlert> alerts) {
        int count = state.getActiveMedicationCount();

        ResolvedThreshold polyThreshold = CIDThresholdResolver.resolve(
                "POLYPHARMACY_THRESHOLD", CIDSeverity.SOFT_FLAG,
                null, // no per-patient polypharmacy threshold
                (double) thresholds.getPolypharmacyThreshold(),
                (double) POLYPHARMACY_THRESHOLD,
                false); // higher count threshold = more sensitive (fewer alerts)

        if (count < polyThreshold.getValue()) return;

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_12, state.getPatientId(),
            String.format("WARNING: Polypharmacy burden high (%d daily medications). " +
                "Consider deprescribing review before adding.", count),
            new ArrayList<>(state.getActiveMedications().keySet()),
            "Review medication list. Assess adherence risk. Consider deprescribing.", null);
        alert.setThresholdProvenance(List.of(polyThreshold.getProvenance()));
        alerts.add(alert);
    }

    /** CID-13: Elderly + intensive BP target. */
    private static void evaluateCID13(ComorbidityState state, Double sbpTarget,
                                      CIDThresholdSet thresholds, List<CIDAlert> alerts) {
        if (state.getAge() == null || state.getAge() < thresholds.getElderlyAgeThreshold()) return;
        if (sbpTarget == null || sbpTarget >= thresholds.getSbpTargetIntensive()) return;

        boolean hasRisk = false;
        if (state.getEGFRCurrent() != null && state.getEGFRCurrent() < 45.0) hasRisk = true;
        if (state.isFallsHistory()) hasRisk = true;
        if (state.isOrthostaticHypotension()) hasRisk = true;

        if (!hasRisk) return;

        // Provenance for the age and SBP target thresholds consulted
        List<ThresholdProvenance> provenance = new ArrayList<>();
        ResolvedThreshold ageThreshold = CIDThresholdResolver.resolve(
                "ELDERLY_AGE_THRESHOLD", CIDSeverity.SOFT_FLAG,
                null,
                (double) thresholds.getElderlyAgeThreshold(),
                (double) ELDERLY_AGE_THRESHOLD,
                false);
        provenance.add(ageThreshold.getProvenance());

        ResolvedThreshold sbpIntensive = CIDThresholdResolver.resolve(
                "SBP_TARGET_INTENSIVE", CIDSeverity.SOFT_FLAG,
                null,
                thresholds.getSbpTargetIntensive(),
                SBP_TARGET_INTENSIVE);
        provenance.add(sbpIntensive.getProvenance());

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_13, state.getPatientId(),
            String.format("WARNING: Intensive SBP target (<%.0f) may increase adverse events " +
                "in this elderly patient (age %d). Consider relaxing to <140 mmHg.", sbpTarget, state.getAge()),
            List.of(),
            "Review BP target per ADA 2026 frailty guidance. Consider <140 mmHg.", null);
        alert.setThresholdProvenance(provenance);
        alerts.add(alert);
    }

    /** CID-14: Metformin + eGFR 30-35 + declining trajectory. */
    private static void evaluateCID14(ComorbidityState state, CIDThresholdSet thresholds,
                                      List<CIDAlert> alerts) {
        if (!state.hasDrugClass("METFORMIN")) return;
        Double eGFR = state.getEGFRCurrent();
        if (eGFR == null) return;

        ResolvedThreshold lowThreshold = CIDThresholdResolver.resolve(
                "EGFR_METFORMIN_LOW", CIDSeverity.SOFT_FLAG,
                state.getPersonalizedEGFRMetforminLow(),
                thresholds.getEgfrMetforminLow(),
                EGFR_METFORMIN_LOW);

        ResolvedThreshold highThreshold = CIDThresholdResolver.resolve(
                "EGFR_METFORMIN_HIGH", CIDSeverity.SOFT_FLAG,
                null, // no per-patient high threshold
                thresholds.getEgfrMetforminHigh(),
                EGFR_METFORMIN_HIGH);

        if (eGFR < lowThreshold.getValue() || eGFR > highThreshold.getValue()) return;

        // Check declining trajectory
        Double decline = state.getEGFRDeclinePercent();
        if (decline == null || decline <= 0) return; // not declining

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_14, state.getPatientId(),
            String.format("WARNING: eGFR approaching metformin threshold (30). Current: %.0f. " +
                "Plan for dose reduction at 30-45, discontinuation at <30.", eGFR),
            state.getMedicationsByClass("METFORMIN"),
            "Consider SGLT2i as glucose-lowering replacement if eGFR permits.", null);
        alert.setThresholdProvenance(List.of(lowThreshold.getProvenance(), highThreshold.getProvenance()));
        alerts.add(alert);
    }

    /** CID-15: SGLT2i + NSAID use. */
    private static void evaluateCID15(ComorbidityState state, List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I") || !state.hasDrugClass("NSAID")) return;

        List<String> meds = new ArrayList<>();
        meds.addAll(state.getMedicationsByClass("SGLT2I"));
        meds.addAll(state.getMedicationsByClass("NSAID"));

        // Purely boolean — no provenance
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

        // Purely boolean — no provenance
        alerts.add(CIDAlert.create(CIDRuleId.CID_16, state.getPatientId(),
            "WARNING: Patient is salt-sensitive. Sodium-retaining drug may elevate BP.",
            meds,
            "Monitor BP closely after initiation.", null));
    }

    /** CID-17: SGLT2i + fasting period > threshold hours. */
    private static void evaluateCID17(ComorbidityState state, CIDThresholdSet thresholds,
                                      List<CIDAlert> alerts) {
        if (!state.hasDrugClass("SGLT2I")) return;
        if (!state.isActiveFastingPeriod()) return;

        ResolvedThreshold fastingThreshold = CIDThresholdResolver.resolve(
                "FASTING_DURATION_THRESHOLD", CIDSeverity.SOFT_FLAG,
                null,
                (double) thresholds.getFastingDurationThreshold(),
                (double) FASTING_DURATION_THRESHOLD,
                false); // higher threshold = more sensitive (fewer alerts)

        if (state.getActiveFastingDurationHours() < fastingThreshold.getValue()) return;

        CIDAlert alert = CIDAlert.create(CIDRuleId.CID_17, state.getPatientId(),
            String.format("WARNING: Extended fasting (%dh) on SGLT2i increases DKA and dehydration risk.",
                state.getActiveFastingDurationHours()),
            state.getMedicationsByClass("SGLT2I"),
            "Advise adequate hydration during non-fasting hours. Consider holding SGLT2i during fasts >20h.", null);
        alert.setThresholdProvenance(List.of(fastingThreshold.getProvenance()));
        alerts.add(alert);
    }
}
