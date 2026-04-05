package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Module 13: CKM Risk Velocity Computer
 *
 * Computes per-domain risk velocity from temporal deltas between two
 * MetricSnapshots (previousSnapshot → currentSnapshot).  Velocity is
 * normalised to [-1, +1] where negative = improving, positive = deteriorating.
 *
 * When previousSnapshot is null (first 7 days before first rotation),
 * returns UNKNOWN classification.
 *
 * Domains (per AHA 2023 CKM Advisory):
 *   METABOLIC  – FBG, HbA1c, meal-iAUC, weight
 *   RENAL      – eGFR (inverted), UACR, BP-kidney impact, adherence
 *   CARDIOVASCULAR – ARV, LDL, morning surge, engagement, intervention effectiveness
 */
public final class Module13CKMRiskComputer {

    // --- Normalisation ranges (clinically meaningful delta per 7 days) ---
    private static final double FBG_DELTA_RANGE = 40.0;
    private static final double HBA1C_DELTA_RANGE = 1.0;
    private static final double IAUC_DELTA_RANGE = 30.0;
    private static final double WEIGHT_DELTA_RANGE = 3.0;
    private static final double EGFR_DELTA_RANGE = 8.0;
    private static final double UACR_DELTA_RANGE = 80.0;
    private static final double ARV_DELTA_RANGE = 8.0;
    private static final double LDL_DELTA_RANGE = 30.0;
    private static final double SURGE_DELTA_RANGE = 20.0;
    private static final double ENGAGEMENT_DELTA_RANGE = 0.4;

    // --- Cross-domain amplification ---
    private static final double AMPLIFICATION_THRESHOLD = 0.2;
    private static final double AMPLIFICATION_FACTOR = 1.5;
    private static final int MIN_DOMAINS_FOR_AMPLIFICATION = 2;
    private static final int MIN_DOMAINS_FOR_VALID_SCORE = 2;

    // --- Composite classification thresholds ---
    private static final double COMPOSITE_DETERIORATING_THRESHOLD = 0.40;
    private static final double COMPOSITE_IMPROVING_THRESHOLD = -0.30;

    private Module13CKMRiskComputer() {}

    /**
     * Compute CKM risk velocity from temporal deltas between two snapshots.
     *
     * @param state per-patient clinical state containing current and previous snapshots
     * @return velocity result with per-domain scores and composite classification
     */
    public static CKMRiskVelocity compute(ClinicalStateSummary state) {
        if (!state.hasVelocityData()) {
            return CKMRiskVelocity.builder()
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0)
                    .dataCompleteness(0.0)
                    .computationTimestamp(state.getLastUpdated())
                    .build();
        }

        ClinicalStateSummary.MetricSnapshot prev = state.previous();
        ClinicalStateSummary.MetricSnapshot curr = state.current();

        double metabolic = computeMetabolicVelocity(prev, curr, state);
        double renal = computeRenalVelocity(prev, curr, state);
        double cardiovascular = computeCardiovascularVelocity(prev, curr, state);

        // Data completeness: fraction of domains with enough signals
        int validDomains = 0;
        if (!Double.isNaN(metabolic)) validDomains++;
        if (!Double.isNaN(renal)) validDomains++;
        if (!Double.isNaN(cardiovascular)) validDomains++;
        double dataCompleteness = validDomains / 3.0;

        if (validDomains < MIN_DOMAINS_FOR_VALID_SCORE) {
            return CKMRiskVelocity.builder()
                    .domainVelocity(CKMRiskDomain.METABOLIC, safe(metabolic))
                    .domainVelocity(CKMRiskDomain.RENAL, safe(renal))
                    .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, safe(cardiovascular))
                    .compositeClassification(CKMRiskVelocity.CompositeClassification.UNKNOWN)
                    .compositeScore(0.0)
                    .dataCompleteness(dataCompleteness)
                    .computationTimestamp(state.getLastUpdated())
                    .build();
        }

        // Safe values for composite calculation (NaN → 0.0)
        double sM = safe(metabolic), sR = safe(renal), sCV = safe(cardiovascular);

        // Count domains above deteriorating threshold
        int deteriorating = 0;
        if (sM > CKMRiskDomain.METABOLIC.getDeterioratingThreshold()) deteriorating++;
        if (sR > CKMRiskDomain.RENAL.getDeterioratingThreshold()) deteriorating++;
        if (sCV > CKMRiskDomain.CARDIOVASCULAR.getDeterioratingThreshold()) deteriorating++;

        // Cross-domain amplification: 2+ domains above amplification threshold
        int aboveAmp = 0;
        if (sM > AMPLIFICATION_THRESHOLD) aboveAmp++;
        if (sR > AMPLIFICATION_THRESHOLD) aboveAmp++;
        if (sCV > AMPLIFICATION_THRESHOLD) aboveAmp++;
        boolean amplify = aboveAmp >= MIN_DOMAINS_FOR_AMPLIFICATION;
        double factor = amplify ? AMPLIFICATION_FACTOR : 1.0;

        // Composite classification: worst-domain-wins
        double worst = Math.max(sM, Math.max(sR, sCV));
        double best = Math.min(sM, Math.min(sR, sCV));

        CKMRiskVelocity.CompositeClassification classification;
        double compositeScore;

        if (worst > COMPOSITE_DETERIORATING_THRESHOLD) {
            classification = CKMRiskVelocity.CompositeClassification.DETERIORATING;
            compositeScore = Math.min(1.0, worst * factor);
        } else if (best < COMPOSITE_IMPROVING_THRESHOLD
                && sM <= 0 && sR <= 0 && sCV <= 0) {
            classification = CKMRiskVelocity.CompositeClassification.IMPROVING;
            compositeScore = best;
        } else {
            classification = CKMRiskVelocity.CompositeClassification.STABLE;
            compositeScore = sM * CKMRiskDomain.METABOLIC.getCompositeWeight()
                    + sR * CKMRiskDomain.RENAL.getCompositeWeight()
                    + sCV * CKMRiskDomain.CARDIOVASCULAR.getCompositeWeight();
        }

        return CKMRiskVelocity.builder()
                .domainVelocity(CKMRiskDomain.METABOLIC, sM)
                .domainVelocity(CKMRiskDomain.RENAL, sR)
                .domainVelocity(CKMRiskDomain.CARDIOVASCULAR, sCV)
                .compositeScore(compositeScore)
                .compositeClassification(classification)
                .crossDomainAmplification(amplify)
                .amplificationFactor(factor)
                .domainsDeteriorating(deteriorating)
                .dataCompleteness(dataCompleteness)
                .computationTimestamp(System.currentTimeMillis())
                .build();
    }

    /**
     * Metabolic velocity: 0.35*deltaFBG + 0.30*deltaHbA1c + 0.20*deltaMealIAUC + 0.15*deltaWeight
     * Positive delta (current > previous) = deteriorating. Negative = improving.
     */
    static double computeMetabolicVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        double totalWeight = 0.0;
        double weighted = 0.0;

        if (prev.fbg != null && curr.fbg != null) {
            weighted += 0.35 * clamp((curr.fbg - prev.fbg) / FBG_DELTA_RANGE);
            totalWeight += 0.35;
        }
        if (prev.hba1c != null && curr.hba1c != null) {
            weighted += 0.30 * clamp((curr.hba1c - prev.hba1c) / HBA1C_DELTA_RANGE);
            totalWeight += 0.30;
        }
        if (prev.meanIAUC != null && curr.meanIAUC != null) {
            weighted += 0.20 * clamp((curr.meanIAUC - prev.meanIAUC) / IAUC_DELTA_RANGE);
            totalWeight += 0.20;
        }
        if (prev.weight != null && curr.weight != null) {
            weighted += 0.15 * clamp((curr.weight - prev.weight) / WEIGHT_DELTA_RANGE);
            totalWeight += 0.15;
        }
        return totalWeight == 0 ? Double.NaN : clamp(weighted / totalWeight);
    }

    /**
     * Renal velocity: 0.50*deltaeGFR(inverted) + 0.25*deltaUACR + 0.15*BP_kidney_impact + 0.10*adherence
     * eGFR inverted: drop in eGFR = positive velocity (deteriorating).
     */
    static double computeRenalVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        double totalWeight = 0.0;
        double weighted = 0.0;

        if (prev.egfr != null && curr.egfr != null) {
            // Inverted: prev - curr so that eGFR decline → positive velocity
            weighted += 0.50 * clamp((prev.egfr - curr.egfr) / EGFR_DELTA_RANGE);
            totalWeight += 0.50;
        }
        if (prev.uacr != null && curr.uacr != null) {
            weighted += 0.25 * clamp((curr.uacr - prev.uacr) / UACR_DELTA_RANGE);
            totalWeight += 0.25;
        }
        if (curr.arv != null && curr.meanSBP != null) {
            double arvNorm = curr.arv > 16.0 ? 1.0 : (curr.arv > 12.0 ? 0.5 : 0.0);
            double sbpFactor = curr.meanSBP > 140.0 ? 1.2 : 1.0;
            weighted += 0.15 * clamp(arvNorm * sbpFactor);
            totalWeight += 0.15;
        }
        if (!state.getRecentInterventionDeltas().isEmpty()) {
            double adherence = state.getRecentInterventionDeltas().stream()
                    .filter(d -> d.getAdherenceScore() != null)
                    .mapToDouble(ClinicalStateSummary.InterventionDeltaSummary::getAdherenceScore)
                    .average().orElse(0.5);
            // Low adherence → positive velocity (deteriorating)
            weighted += 0.10 * clamp(0.5 - adherence);
            totalWeight += 0.10;
        }
        return totalWeight == 0 ? Double.NaN : clamp(weighted / totalWeight);
    }

    /**
     * Cardiovascular velocity: 0.30*deltaARV + 0.25*deltaLDL + 0.20*deltaSurge + 0.15*deltaEngagement + 0.10*intervention_effectiveness
     */
    static double computeCardiovascularVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        double totalWeight = 0.0;
        double weighted = 0.0;

        if (prev.arv != null && curr.arv != null) {
            weighted += 0.30 * clamp((curr.arv - prev.arv) / ARV_DELTA_RANGE);
            totalWeight += 0.30;
        }
        if (prev.ldl != null && curr.ldl != null) {
            weighted += 0.25 * clamp((curr.ldl - prev.ldl) / LDL_DELTA_RANGE);
            totalWeight += 0.25;
        }
        if (prev.morningSurgeMagnitude != null && curr.morningSurgeMagnitude != null) {
            weighted += 0.20 * clamp((curr.morningSurgeMagnitude - prev.morningSurgeMagnitude) / SURGE_DELTA_RANGE);
            totalWeight += 0.20;
        }
        if (prev.engagementScore != null && curr.engagementScore != null) {
            // Inverted: engagement drop = positive velocity (deteriorating)
            weighted += 0.15 * clamp((prev.engagementScore - curr.engagementScore) / ENGAGEMENT_DELTA_RANGE);
            totalWeight += 0.15;
        }
        if (!state.getRecentInterventionDeltas().isEmpty()) {
            long insufficient = state.getRecentInterventionDeltas().stream()
                    .filter(d -> d.getAttribution() == TrajectoryAttribution.INTERVENTION_INSUFFICIENT)
                    .count();
            double ratio = (double) insufficient / state.getRecentInterventionDeltas().size();
            weighted += 0.10 * clamp(ratio * 2 - 1);
            totalWeight += 0.10;
        }
        return totalWeight == 0 ? Double.NaN : clamp(weighted / totalWeight);
    }

    /** Clamp value to [-1.0, +1.0] */
    private static double clamp(double v) { return Math.max(-1.0, Math.min(1.0, v)); }

    /** Convert NaN to 0.0 for safe composite arithmetic */
    private static double safe(double v) { return Double.isNaN(v) ? 0.0 : v; }

}
