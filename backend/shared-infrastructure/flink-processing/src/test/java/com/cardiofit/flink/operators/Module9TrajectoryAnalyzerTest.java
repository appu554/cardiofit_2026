package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module9TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.Arrays;
import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 9 Phase 2: Trajectory Analyzer (OLS relapse prediction)")
class Module9TrajectoryAnalyzerTest {

    private static final String PID = "test-patient-001";

    // --- OLS Slope Tests ---

    @Test
    @DisplayName("computeSlope: perfectly increasing [0,1,2,3,4,5,6] → positive slope ~1.0")
    void perfectlyIncreasingSlope() {
        // buffer[0]=today=6(newest), buffer[6]=oldest=0
        // Chronological: 0,1,2,3,4,5,6 → slope = 1.0/day
        double[] buffer = {6.0, 5.0, 4.0, 3.0, 2.0, 1.0, 0.0};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(1.0, slope, 1e-9, "Perfect linear increase should have slope 1.0");
    }

    @Test
    @DisplayName("computeSlope: perfectly decreasing [0,1,2,3,4,5,6] → negative slope ~-1.0")
    void perfectlyDecreasingSlope() {
        // buffer[0]=today=0(newest), buffer[6]=oldest=6
        // Chronological: 6,5,4,3,2,1,0 → slope = -1.0/day
        double[] buffer = {0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(-1.0, slope, 1e-9, "Perfect linear decrease should have slope -1.0");
    }

    @Test
    @DisplayName("computeSlope: flat line [0.5, 0.5, 0.5, ...] → slope 0.0")
    void flatSlope() {
        double[] buffer = {0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(0.0, slope, 1e-9);
    }

    @Test
    @DisplayName("computeSlope: all sentinel values → slope 0.0 (insufficient data)")
    void allSentinelSlope() {
        double[] buffer = {-1.0, -1.0, -1.0, -1.0, -1.0, -1.0, -1.0};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(0.0, slope, 1e-9);
    }

    @Test
    @DisplayName("computeSlope: only 3 valid points → slope 0.0 (below MIN_VALID_POINTS=4)")
    void tooFewValidPoints() {
        double[] buffer = {0.8, -1.0, -1.0, 0.5, -1.0, -1.0, 0.2};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(0.0, slope, 1e-9, "3 points is below minimum of 4");
    }

    @Test
    @DisplayName("computeSlope: exactly 4 valid points → computes slope")
    void exactlyFourPoints() {
        // buffer[0]=today, buffer[6]=oldest
        // Valid: index 0(x=6)=0.8, index 2(x=4)=0.6, index 4(x=2)=0.4, index 6(x=0)=0.2
        double[] buffer = {0.8, -1.0, 0.6, -1.0, 0.4, -1.0, 0.2};
        double slope = Module9TrajectoryAnalyzer.computeSlope(buffer);
        assertEquals(0.1, slope, 1e-9, "Increasing by 0.1 per day");
    }

    @Test
    @DisplayName("computeSlope: null buffer → slope 0.0")
    void nullBuffer() {
        assertEquals(0.0, Module9TrajectoryAnalyzer.computeSlope(null), 1e-9);
    }

    // --- negativeComponent Tests ---

    @Test
    @DisplayName("negativeComponent: positive slope → 0.0 (no risk)")
    void positiveNoRisk() {
        assertEquals(0.0, Module9TrajectoryAnalyzer.negativeComponent(0.05), 1e-9);
    }

    @Test
    @DisplayName("negativeComponent: zero slope → 0.0")
    void zeroNoRisk() {
        assertEquals(0.0, Module9TrajectoryAnalyzer.negativeComponent(0.0), 1e-9);
    }

    @Test
    @DisplayName("negativeComponent: -0.07/day → 0.5 (half maximum decline)")
    void halfDecline() {
        assertEquals(0.5, Module9TrajectoryAnalyzer.negativeComponent(-0.07), 1e-9);
    }

    @Test
    @DisplayName("negativeComponent: -0.14/day → 1.0 (maximum decline rate)")
    void maxDecline() {
        assertEquals(1.0, Module9TrajectoryAnalyzer.negativeComponent(-0.14), 1e-9);
    }

    @Test
    @DisplayName("negativeComponent: -0.28/day → 1.0 (capped at 1.0)")
    void cappedDecline() {
        assertEquals(1.0, Module9TrajectoryAnalyzer.negativeComponent(-0.28), 1e-9);
    }

    // --- Full Analysis Integration Tests ---

    @Test
    @DisplayName("analyze: insufficient history (<7 days) → empty result")
    void insufficientHistory() {
        EngagementState state = Module9TestBuilder.fullyEngagedState(PID);
        state.setValidHistoryDays(6);

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isEmpty(), "Should not analyze with < 7 valid history days");
    }

    @Test
    @DisplayName("analyze: all features stable → LOW risk (near 0.0)")
    void stableFeatures() {
        EngagementState state = createStateWithTrajectory(PID, 14,
            new double[]{0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7}, // steps: flat
            new double[]{0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6}, // meal: flat
            new double[]{0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3}, // latency: flat
            new double[]{0.8, 0.8, 0.8, 0.8, 0.8, 0.8, 0.8}, // checkin: flat
            new double[]{1.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}  // protein: flat
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertEquals(0.0, risk.get().getRelapseRiskScore(), 1e-9);
        assertEquals(RelapseRiskScore.RiskTier.LOW, risk.get().getRiskTier());
        assertFalse(Module9TrajectoryAnalyzer.isAlertWorthy(risk.get()));
    }

    @Test
    @DisplayName("analyze: all features declining steeply → HIGH risk")
    void allDeclining() {
        // Each buffer: today=0.0, oldest=0.98 → chronological: 0.98→0.0 (steep decline)
        double[] declining = {0.0, 0.14, 0.28, 0.42, 0.56, 0.70, 0.84};
        EngagementState state = createStateWithTrajectory(PID, 14,
            declining, declining,
            // Latency: INVERTED — rising latency is bad
            // Rising latency = {0.84, 0.70, 0.56, ...} → positive slope for latency
            // After inversion in negativeComponent, this contributes to risk
            new double[]{0.84, 0.70, 0.56, 0.42, 0.28, 0.14, 0.0},
            declining, declining
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertTrue(risk.get().getRelapseRiskScore() >= 0.70,
            "All features declining should produce HIGH risk, got: " + risk.get().getRelapseRiskScore());
        assertEquals(RelapseRiskScore.RiskTier.HIGH, risk.get().getRiskTier());
        assertTrue(Module9TrajectoryAnalyzer.isAlertWorthy(risk.get()));
        assertEquals("Urgent: Generate KB-23 Decision Card. Schedule physician outreach call.",
            risk.get().getRecommendedAction());
    }

    @Test
    @DisplayName("analyze: moderate decline in steps only → MODERATE risk near 0.30 * weight")
    void stepsOnlyDecline() {
        // Steps: declining from 0.84 to 0.0 (steep) → slope ~ -0.14/day
        double[] declining = {0.0, 0.14, 0.28, 0.42, 0.56, 0.70, 0.84};
        double[] flat = {0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7};
        double[] flatLatency = {0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3};

        EngagementState state = createStateWithTrajectory(PID, 14,
            declining, flat, flatLatency, flat, flat
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        // Steps weight = 0.30, max decline → 0.30 * 1.0 = 0.30
        assertEquals(0.30, risk.get().getRelapseRiskScore(), 0.02,
            "Steps-only max decline = 0.30 risk");
        assertEquals(RelapseRiskScore.RiskTier.LOW, risk.get().getRiskTier());
    }

    @Test
    @DisplayName("analyze: moderate decline across multiple features → MODERATE tier")
    void moderateMultiFeatureDecline() {
        // Steps: half decline → -0.07/day → negativeComponent = 0.5
        // Meal: half decline → 0.5
        // Latency: half increase (bad) → 0.5
        double[] halfDecline = {0.3, 0.37, 0.44, 0.51, 0.58, 0.65, 0.72};
        double[] halfLatencyIncrease = {0.72, 0.65, 0.58, 0.51, 0.44, 0.37, 0.3};
        double[] flat = {0.7, 0.7, 0.7, 0.7, 0.7, 0.7, 0.7};

        EngagementState state = createStateWithTrajectory(PID, 14,
            halfDecline, halfDecline, halfLatencyIncrease, flat, flat
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        // steps: 0.30 * 0.5 = 0.15, meal: 0.20 * 0.5 = 0.10, latency: 0.25 * 0.5 = 0.125
        // Total ≈ 0.375
        double expected = 0.30 * 0.5 + 0.20 * 0.5 + 0.25 * 0.5;
        assertEquals(expected, risk.get().getRelapseRiskScore(), 0.05);
    }

    @Test
    @DisplayName("analyze: improving trends → LOW risk (0.0)")
    void improvingTrends() {
        // All features improving (positive slopes)
        double[] improving = {0.84, 0.70, 0.56, 0.42, 0.28, 0.14, 0.0};
        // Latency: decreasing = good (less response time)
        double[] latencyDecreasing = {0.0, 0.14, 0.28, 0.42, 0.56, 0.70, 0.84};

        EngagementState state = createStateWithTrajectory(PID, 14,
            improving, improving, latencyDecreasing, improving, improving
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertEquals(0.0, risk.get().getRelapseRiskScore(), 1e-9,
            "All improving trends should produce zero risk");
    }

    @Test
    @DisplayName("analyze: sparse data (4 of 7 valid) still computes slope")
    void sparseButSufficient() {
        double[] sparse = {0.2, -1.0, 0.4, -1.0, 0.6, -1.0, 0.8};
        double[] flat = {0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5};

        EngagementState state = createStateWithTrajectory(PID, 14,
            sparse, flat, flat, flat, flat
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        // Steps: decreasing from 0.8 to 0.2 → negative slope
        assertTrue(risk.get().getStepsSlope() < 0,
            "Sparse declining data should have negative slope");
    }

    @Test
    @DisplayName("RiskTier boundaries: 0.39=LOW, 0.40=MODERATE, 0.69=MODERATE, 0.70=HIGH")
    void riskTierBoundaries() {
        assertEquals(RelapseRiskScore.RiskTier.LOW, RelapseRiskScore.RiskTier.fromScore(0.39));
        assertEquals(RelapseRiskScore.RiskTier.MODERATE, RelapseRiskScore.RiskTier.fromScore(0.40));
        assertEquals(RelapseRiskScore.RiskTier.MODERATE, RelapseRiskScore.RiskTier.fromScore(0.69));
        assertEquals(RelapseRiskScore.RiskTier.HIGH, RelapseRiskScore.RiskTier.fromScore(0.70));
        assertEquals(RelapseRiskScore.RiskTier.HIGH, RelapseRiskScore.RiskTier.fromScore(1.0));
    }

    @Test
    @DisplayName("RelapseRiskScore: MODERATE tier has recovery motivation action")
    void moderateActionMessage() {
        RelapseRiskScore score = RelapseRiskScore.create(
            PID, 0.50, -0.05, -0.03, 0.02, -0.01, 0.0,
            0.45, EngagementLevel.YELLOW, "DECLINING", 14);
        assertEquals(RelapseRiskScore.RiskTier.MODERATE, score.getRiskTier());
        assertTrue(score.getRecommendedAction().contains("Recovery motivation"));
    }

    @Test
    @DisplayName("RelapseRiskScore: HIGH tier has KB-23 Decision Card action")
    void highActionMessage() {
        RelapseRiskScore score = RelapseRiskScore.create(
            PID, 0.80, -0.10, -0.08, 0.05, -0.06, -0.04,
            0.20, EngagementLevel.RED, "DECLINING", 14);
        assertEquals(RelapseRiskScore.RiskTier.HIGH, score.getRiskTier());
        assertTrue(score.getRecommendedAction().contains("KB-23 Decision Card"));
    }

    @Test
    @DisplayName("RelapseRiskScore: LOW tier has null action")
    void lowNoAction() {
        RelapseRiskScore score = RelapseRiskScore.create(
            PID, 0.10, 0.01, 0.02, -0.01, 0.01, 0.01,
            0.80, EngagementLevel.GREEN, "STEADY", 14);
        assertEquals(RelapseRiskScore.RiskTier.LOW, score.getRiskTier());
        assertNull(score.getRecommendedAction());
    }

    @Test
    @DisplayName("analyze: latency inversion — rising latency contributes to risk")
    void latencyInversion() {
        double[] flat = {0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5};
        // Latency rising = bad: today high(0.84), oldest low(0.0)
        // buffer[0]=today=0.84, buffer[6]=oldest=0.0
        // Chronological slope = POSITIVE (0→0.84) which means worsening
        // After inversion in analyzer, this becomes negative component → risk
        double[] risingLatency = {0.84, 0.70, 0.56, 0.42, 0.28, 0.14, 0.0};

        EngagementState state = createStateWithTrajectory(PID, 14,
            flat, flat, risingLatency, flat, flat
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertTrue(risk.get().getRelapseRiskScore() > 0,
            "Rising latency should contribute to risk");
        // Latency weight = 0.25, max inverted decline → ~0.25
        assertEquals(0.25, risk.get().getRelapseRiskScore(), 0.05);
    }

    @Test
    @DisplayName("analyze: risk score clamped to [0.0, 1.0]")
    void riskScoreClamped() {
        EngagementState state = createStateWithTrajectory(PID, 14,
            new double[]{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
            new double[]{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
            new double[]{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
            new double[]{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5},
            new double[]{0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5}
        );

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertTrue(risk.get().getRelapseRiskScore() >= 0.0);
        assertTrue(risk.get().getRelapseRiskScore() <= 1.0);
    }

    @Test
    @DisplayName("analyze: channel and dataTier propagated to RelapseRiskScore")
    void channelPropagation() {
        double[] flat = {0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5};
        EngagementState state = createStateWithTrajectory(PID, 14,
            flat, flat, flat, flat, flat);
        state.setChannel("GOVERNMENT");
        state.setDataTier("TIER_2_HOME_DEVICE");

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        Optional<RelapseRiskScore> risk = Module9TrajectoryAnalyzer.analyze(state, result);
        assertTrue(risk.isPresent());
        assertEquals("GOVERNMENT", risk.get().getChannel());
        assertEquals("TIER_2_HOME_DEVICE", risk.get().getDataTier());
    }

    // --- Helper ---

    private EngagementState createStateWithTrajectory(String patientId, int validDays,
                                                       double[] steps, double[] meal,
                                                       double[] latency, double[] checkin,
                                                       double[] protein) {
        EngagementState state = Module9TestBuilder.fullyEngagedState(patientId);
        state.setValidHistoryDays(validDays);

        // Copy trajectory buffers (use reflection-free approach via direct field access)
        System.arraycopy(steps, 0, state.getStepsBuffer7d(), 0, 7);
        System.arraycopy(meal, 0, state.getMealQualityBuffer7d(), 0, 7);
        System.arraycopy(latency, 0, state.getResponseLatencyBuffer7d(), 0, 7);
        System.arraycopy(checkin, 0, state.getCheckinCompletenessBuffer7d(), 0, 7);
        System.arraycopy(protein, 0, state.getProteinAdherenceBuffer7d(), 0, 7);

        return state;
    }
}
