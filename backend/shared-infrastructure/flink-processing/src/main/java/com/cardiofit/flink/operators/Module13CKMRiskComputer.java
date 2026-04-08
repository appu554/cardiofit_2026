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

    // --- Absolute severity thresholds for current-state contribution ---
    // Cardiovascular: ARV > 16 = HIGH, meanSBP > 160 = Stage 2 uncontrolled
    private static final double ARV_HIGH_THRESHOLD = 16.0;
    private static final double ARV_ELEVATED_THRESHOLD = 12.0;
    private static final double SBP_STAGE2_THRESHOLD = 160.0;
    private static final double SBP_STAGE1_THRESHOLD = 140.0;
    private static final double SURGE_ELEVATED_THRESHOLD = 20.0;
    // Metabolic: FBG > 140 = alarming, HbA1c > 8.0 = poor control
    private static final double FBG_ALARMING_THRESHOLD = 140.0;
    private static final double HBA1C_POOR_THRESHOLD = 8.0;

    // --- Population-level defaults (overridden by A1 personalised targets when available) ---
    static final double DEFAULT_SBP_KIDNEY_THRESHOLD = 140.0;

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
                .computationTimestamp(state.getLastUpdated())
                .build();
    }

    /**
     * Metabolic velocity: combines delta-based and absolute-severity signals.
     *
     * Delta signals:
     *   0.25*deltaFBG + 0.20*deltaHbA1c + 0.15*deltaMealIAUC + 0.10*deltaWeight
     * Absolute severity signals:
     *   0.20*currentGlycaemicSeverity + 0.10*currentWeightSeverity
     *
     * Prevents velocity=0.0 when FBG is persistently above 140 mg/dL or HbA1c > 8%.
     */
    static double computeMetabolicVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        double totalWeight = 0.0;
        double weighted = 0.0;

        // --- Delta-based signals ---

        if (prev.fbg != null && curr.fbg != null) {
            weighted += 0.25 * clamp((curr.fbg - prev.fbg) / FBG_DELTA_RANGE);
            totalWeight += 0.25;
        }
        if (prev.hba1c != null && curr.hba1c != null) {
            weighted += 0.20 * clamp((curr.hba1c - prev.hba1c) / HBA1C_DELTA_RANGE);
            totalWeight += 0.20;
        }
        if (prev.meanIAUC != null && curr.meanIAUC != null) {
            weighted += 0.15 * clamp((curr.meanIAUC - prev.meanIAUC) / IAUC_DELTA_RANGE);
            totalWeight += 0.15;
        }
        if (prev.weight != null && curr.weight != null) {
            weighted += 0.10 * clamp((curr.weight - prev.weight) / WEIGHT_DELTA_RANGE);
            totalWeight += 0.10;
        }

        // --- Absolute severity signals ---

        // Glycaemic severity: combines FBG + HbA1c current values
        if (curr.fbg != null || curr.hba1c != null) {
            double glycSeverity = 0.0;
            if (curr.fbg != null) {
                // Use personalised FBG target if available, else default 110
                double fbgTarget = state.getPersonalizedFBGTarget() != null
                        ? state.getPersonalizedFBGTarget() : 110.0;
                glycSeverity += curr.fbg > FBG_ALARMING_THRESHOLD ? 1.0
                        : (curr.fbg > fbgTarget ? 0.4 : 0.0);
            }
            if (curr.hba1c != null) {
                // Use personalised HbA1c target if available, else default 7.0
                double hba1cTarget = state.getPersonalizedHbA1cTarget() != null
                        ? state.getPersonalizedHbA1cTarget() : 7.0;
                glycSeverity += curr.hba1c > HBA1C_POOR_THRESHOLD ? 1.0
                        : (curr.hba1c > hba1cTarget ? 0.4 : 0.0);
            }
            // Normalise: max raw = 2.0
            weighted += 0.20 * clamp(glycSeverity / 2.0);
            totalWeight += 0.20;
        }

        // Weight/BMI severity (BMI > 30 = obese)
        if (curr.bmi != null) {
            double weightSeverity = curr.bmi > 35.0 ? 1.0
                    : (curr.bmi > 30.0 ? 0.5 : 0.0);
            weighted += 0.10 * weightSeverity;
            totalWeight += 0.10;
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
            // A1: Use personalised SBP-kidney threshold from KB-20 if available
            double sbpKidneyThreshold = state.getPersonalizedSBPKidneyThreshold() != null
                    ? state.getPersonalizedSBPKidneyThreshold() : DEFAULT_SBP_KIDNEY_THRESHOLD;
            double sbpFactor = curr.meanSBP > sbpKidneyThreshold ? 1.2 : 1.0;
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
     * Cardiovascular velocity: combines delta-based and absolute-severity signals.
     *
     * Delta signals (change between snapshots):
     *   0.20*deltaARV + 0.15*deltaLDL + 0.10*deltaSurge + 0.10*deltaEngagement + 0.05*intervention
     * Absolute severity signals (current snapshot state):
     *   0.20*currentBPSeverity + 0.10*currentVariabilityClass + 0.10*currentSurgeSeverity
     *
     * The absolute-severity component prevents velocity=0.0 when a patient is persistently
     * in a dangerous state (e.g., ARV=17 HIGH + SBP=167 Stage 2) but the delta is small
     * because both snapshots are similarly bad.
     */
    static double computeCardiovascularVelocity(ClinicalStateSummary.MetricSnapshot prev,
            ClinicalStateSummary.MetricSnapshot curr, ClinicalStateSummary state) {
        double totalWeight = 0.0;
        double weighted = 0.0;

        // --- Delta-based signals (trend between snapshot windows) ---

        if (prev.arv != null && curr.arv != null) {
            weighted += 0.20 * clamp((curr.arv - prev.arv) / ARV_DELTA_RANGE);
            totalWeight += 0.20;
        }
        if (prev.ldl != null && curr.ldl != null) {
            weighted += 0.15 * clamp((curr.ldl - prev.ldl) / LDL_DELTA_RANGE);
            totalWeight += 0.15;
        }
        if (prev.morningSurgeMagnitude != null && curr.morningSurgeMagnitude != null) {
            weighted += 0.10 * clamp((curr.morningSurgeMagnitude - prev.morningSurgeMagnitude) / SURGE_DELTA_RANGE);
            totalWeight += 0.10;
        }
        if (prev.engagementScore != null && curr.engagementScore != null) {
            // Inverted: engagement drop = positive velocity (deteriorating)
            weighted += 0.10 * clamp((prev.engagementScore - curr.engagementScore) / ENGAGEMENT_DELTA_RANGE);
            totalWeight += 0.10;
        }
        if (!state.getRecentInterventionDeltas().isEmpty()) {
            long insufficient = state.getRecentInterventionDeltas().stream()
                    .filter(d -> d.getAttribution() == TrajectoryAttribution.INTERVENTION_INSUFFICIENT)
                    .count();
            double ratio = (double) insufficient / state.getRecentInterventionDeltas().size();
            weighted += 0.05 * clamp(ratio * 2 - 1);
            totalWeight += 0.05;
        }

        // --- Absolute severity signals (current clinical state) ---

        // BP severity: combines ARV (variability) + meanSBP (control)
        if (curr.arv != null || curr.meanSBP != null) {
            double bpSeverity = 0.0;
            if (curr.arv != null) {
                bpSeverity += curr.arv > ARV_HIGH_THRESHOLD ? 1.0
                        : (curr.arv > ARV_ELEVATED_THRESHOLD ? 0.5 : 0.0);
            }
            if (curr.meanSBP != null) {
                bpSeverity += curr.meanSBP > SBP_STAGE2_THRESHOLD ? 1.0
                        : (curr.meanSBP > SBP_STAGE1_THRESHOLD ? 0.4 : 0.0);
            }
            // Normalise combined severity to [0, 1]: max raw = 2.0
            weighted += 0.20 * clamp(bpSeverity / 2.0);
            totalWeight += 0.20;
        }

        // Variability classification severity
        if (curr.variabilityClass != null) {
            double varSeverity;
            switch (curr.variabilityClass) {
                case HIGH: varSeverity = 1.0; break;
                case ELEVATED: varSeverity = 0.5; break;
                default: varSeverity = 0.0;
            }
            weighted += 0.10 * varSeverity;
            totalWeight += 0.10;
        }

        // Morning surge absolute severity
        if (curr.morningSurgeMagnitude != null) {
            double surgeSeverity = curr.morningSurgeMagnitude > SURGE_ELEVATED_THRESHOLD ? 1.0
                    : (curr.morningSurgeMagnitude > 15.0 ? 0.4 : 0.0);
            weighted += 0.10 * surgeSeverity;
            totalWeight += 0.10;
        }

        return totalWeight == 0 ? Double.NaN : clamp(weighted / totalWeight);
    }

    /** Clamp value to [-1.0, +1.0] */
    private static double clamp(double v) { return Math.max(-1.0, Math.min(1.0, v)); }

    /** Convert NaN to 0.0 for safe composite arithmetic */
    private static double safe(double v) { return Double.isNaN(v) ? 0.0 : v; }

}
