package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

import java.util.Optional;

/**
 * Module 9 Phase 2: Trajectory Analyzer for early relapse prediction.
 *
 * Analyzes 7-day sliding windows of 5 behavioral features using OLS
 * (Ordinary Least Squares) linear regression to compute trend slopes.
 * Negative slopes indicate declining engagement trajectory.
 *
 * Five trajectory features and weights (from plan P2.2):
 *   1. Steps slope       (0.30) — wearable step count trend
 *   2. Meal quality slope (0.20) — carb/protein ratio trend
 *   3. Response latency   (0.25) — app session response time trend
 *   4. Check-in complete  (0.15) — fields filled per check-in trend
 *   5. Protein adherence  (0.10) — protein flag trend
 *
 * Relapse risk = weighted sum of NEGATIVE normalized slopes → [0.0, 1.0]
 *   0.0 = stable or improving, 1.0 = maximum decline across all features.
 *
 * Requires validHistoryDays >= 7 for meaningful trajectory calculation.
 * Returns Optional.empty() if insufficient history.
 *
 * Static utility class (no Flink dependency — unit testable).
 */
public final class Module9TrajectoryAnalyzer {

    private Module9TrajectoryAnalyzer() {}

    // Expert weights from plan P2.2
    static final double W_STEPS = 0.30;
    static final double W_MEAL = 0.20;
    static final double W_LATENCY = 0.25;
    static final double W_CHECKIN = 0.15;
    static final double W_PROTEIN = 0.10;

    // Minimum valid data points in a 7-day buffer for reliable OLS
    private static final int MIN_VALID_POINTS = 4;

    // Minimum validHistoryDays to attempt trajectory analysis
    private static final int MIN_HISTORY_DAYS = 7;

    /**
     * Compute relapse risk from the 5 trajectory buffers in state.
     *
     * @param state   EngagementState with populated trajectory buffers
     * @param result  Current day's score computation result
     * @return RelapseRiskScore if sufficient data, empty otherwise
     */
    public static Optional<RelapseRiskScore> analyze(EngagementState state,
                                                      Module9ScoreComputer.Result result) {
        if (state.getValidHistoryDays() < MIN_HISTORY_DAYS) {
            return Optional.empty();
        }

        double stepsSlope = computeSlope(state.getStepsBuffer7d());
        double mealSlope = computeSlope(state.getMealQualityBuffer7d());
        double latencySlope = computeSlope(state.getResponseLatencyBuffer7d());
        double checkinSlope = computeSlope(state.getCheckinCompletenessBuffer7d());
        double proteinSlope = computeSlope(state.getProteinAdherenceBuffer7d());

        // Weighted sum of negative slopes only (positive slopes = improving, don't add risk)
        // For latency, POSITIVE slope = worsening (higher latency = bad), so invert
        double riskScore = 0.0;
        riskScore += W_STEPS * negativeComponent(stepsSlope);
        riskScore += W_MEAL * negativeComponent(mealSlope);
        riskScore += W_LATENCY * negativeComponent(-latencySlope); // inverted: rising latency is bad
        riskScore += W_CHECKIN * negativeComponent(checkinSlope);
        riskScore += W_PROTEIN * negativeComponent(proteinSlope);

        // Clamp to [0.0, 1.0]
        riskScore = Math.max(0.0, Math.min(1.0, riskScore));

        RelapseRiskScore score = RelapseRiskScore.create(
            state.getPatientId(),
            riskScore,
            stepsSlope,
            mealSlope,
            latencySlope,
            checkinSlope,
            proteinSlope,
            result.compositeScore,
            result.level,
            result.phenotype,
            state.getValidHistoryDays()
        );
        score.setChannel(state.getChannel());
        score.setDataTier(state.getDataTier());

        return Optional.of(score);
    }

    /**
     * Extract the negative component of a slope, normalized to [0, 1].
     * Slope is per-day change in a 0-1 normalized feature.
     * A slope of -0.14/day (losing 1.0 over 7 days) = maximum negative = 1.0.
     * A slope of 0 or positive = 0.0 (no risk contribution).
     */
    static double negativeComponent(double slope) {
        if (slope >= 0.0) return 0.0;
        // Normalize: -0.14/day maps to 1.0 risk contribution
        // |slope| / 0.14 capped at 1.0
        return Math.min(1.0, Math.abs(slope) / 0.14);
    }

    /**
     * Compute OLS linear regression slope over a 7-day buffer.
     * Buffer: index 0 = today (most recent), index 6 = 7 days ago (oldest).
     * Sentinel value -1.0 indicates no data for that day (excluded from regression).
     *
     * Returns 0.0 if fewer than MIN_VALID_POINTS are available.
     *
     * OLS formula:
     *   slope = (n * Σ(xi*yi) - Σxi * Σyi) / (n * Σ(xi²) - (Σxi)²)
     *
     * where x = time index (0=oldest, 6=newest for chronological order)
     *       y = feature value
     */
    static double computeSlope(double[] buffer) {
        if (buffer == null) return 0.0;

        // Reverse chronological index: buffer[0]=today → x=6, buffer[6]=oldest → x=0
        int n = 0;
        double sumX = 0, sumY = 0, sumXY = 0, sumX2 = 0;

        for (int i = 0; i < buffer.length; i++) {
            if (buffer[i] < 0.0) continue; // sentinel: skip missing data
            int x = buffer.length - 1 - i; // chronological: oldest=0, newest=6
            double y = buffer[i];
            n++;
            sumX += x;
            sumY += y;
            sumXY += x * y;
            sumX2 += x * x;
        }

        if (n < MIN_VALID_POINTS) return 0.0;

        double denominator = n * sumX2 - sumX * sumX;
        if (Math.abs(denominator) < 1e-12) return 0.0;

        return (n * sumXY - sumX * sumY) / denominator;
    }

    /**
     * Check if the risk score warrants an alert emission (MODERATE or HIGH tier).
     */
    public static boolean isAlertWorthy(RelapseRiskScore score) {
        return score.getRiskTier() != RelapseRiskScore.RiskTier.LOW;
    }
}
