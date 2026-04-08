package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module9TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 9: Score Computer (channel-aware, R7 phenotype)")
class Module9ScoreComputerTest {

    @Test
    @DisplayName("Fully engaged state -> score 1.0, GREEN, SPORADIC (no history yet)")
    void fullyEngaged() {
        EngagementState state = Module9TestBuilder.fullyEngagedState("patient-001");
        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);

        assertEquals(1.0, result.compositeScore, 1e-9);
        assertEquals(EngagementLevel.GREEN, result.level);
    }

    @Test
    @DisplayName("Fully disengaged state -> score 0.0, RED (CORPORATE)")
    void fullyDisengaged() {
        EngagementState state = Module9TestBuilder.disengagedState("patient-002");
        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);

        assertEquals(0.0, result.compositeScore, 1e-9);
        assertEquals(EngagementLevel.RED, result.level);
    }

    @Test
    @DisplayName("Only glucose monitoring (weight 0.20), all 14 days -> score 0.20")
    void singleSignalGlucose() {
        EngagementState state = new EngagementState();
        state.setPatientId("patient-003");
        boolean[] glucoseBitmap = new boolean[14];
        java.util.Arrays.fill(glucoseBitmap, true);
        state.setSignalBitmap(SignalType.GLUCOSE_MONITORING, glucoseBitmap);

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);

        assertEquals(0.20, result.compositeScore, 1e-9);
        // CORPORATE: 0.20 is at ORANGE threshold boundary
        assertEquals(EngagementLevel.ORANGE, result.level);
    }

    @Test
    @DisplayName("All signals active 7 of 14 days -> score ~0.50")
    void halfEngaged() {
        EngagementState state = new EngagementState();
        state.setPatientId("patient-004");
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            for (int i = 0; i < 7; i++) bitmap[i] = true;
            state.setSignalBitmap(signal, bitmap);
        }

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);

        assertEquals(0.50, result.compositeScore, 1e-9);
        // CORPORATE: 0.50 is between 0.40 (YELLOW) and 0.70 (GREEN)
        assertEquals(EngagementLevel.YELLOW, result.level);
    }

    @Test
    @DisplayName("R2: Score 0.35 — CORPORATE=ORANGE, GOVERNMENT=YELLOW, ACCHS=GREEN")
    void channelDivergence() {
        // Construct a state that gives exactly 0.35 composite:
        // All 8 signals at density 0.35 -> composite = 0.35 * 1.0 = 0.35
        EngagementState state = new EngagementState();
        state.setPatientId("patient-005");
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            // 5 of 14 = 0.357, close enough; use exact 5 days
            for (int i = 0; i < 5; i++) bitmap[i] = true;
            state.setSignalBitmap(signal, bitmap);
        }

        // CORPORATE channel
        state.setChannel("CORPORATE");
        Module9ScoreComputer.Result corpResult = Module9ScoreComputer.compute(state);
        double score = corpResult.compositeScore; // 5/14 = 0.357
        assertEquals(EngagementLevel.ORANGE, EngagementLevel.fromScore(score, EngagementChannel.CORPORATE));

        // GOVERNMENT channel
        assertEquals(EngagementLevel.YELLOW, EngagementLevel.fromScore(score, EngagementChannel.GOVERNMENT));

        // ACCHS channel
        assertEquals(EngagementLevel.GREEN, EngagementLevel.fromScore(score, EngagementChannel.ACCHS));
    }

    @Test
    @DisplayName("R7: validHistoryDays < 10 -> phenotype always SPORADIC")
    void insufficientHistoryDays() {
        EngagementState state = Module9TestBuilder.fullyEngagedState("patient-006");
        state.setValidHistoryDays(5);

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        assertEquals("SPORADIC", result.phenotype);
    }

    @Test
    @DisplayName("R7: Declining state with 14 valid days -> DECLINING phenotype")
    void decliningPhenotype() {
        EngagementState state = Module9TestBuilder.decliningState("patient-007");
        state.setValidHistoryDays(14);

        // Set composite history: older half high, recent half low
        double[] history = state.getCompositeHistory14d();
        for (int i = 0; i < 7; i++) history[i] = 0.20;  // recent: low
        for (int i = 7; i < 14; i++) history[i] = 0.80;  // older: high

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        assertEquals("DECLINING", result.phenotype);
    }

    @Test
    @DisplayName("Steady state with 14 valid days -> STEADY phenotype")
    void steadyPhenotype() {
        EngagementState state = Module9TestBuilder.fullyEngagedState("patient-008");
        state.setValidHistoryDays(14);

        // Set composite history: consistent 0.80
        double[] history = state.getCompositeHistory14d();
        java.util.Arrays.fill(history, 0.80);

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        assertEquals("STEADY", result.phenotype);
    }

    @Test
    @DisplayName("Edge: CORPORATE score at exactly 0.70 -> GREEN (epsilon boundary)")
    void greenBoundary() {
        assertEquals(EngagementLevel.GREEN,
            EngagementLevel.fromScore(0.70, EngagementChannel.CORPORATE));
    }

    @Test
    @DisplayName("R7: Patient with zero composite for 14 days -> phenotype uses zeros correctly")
    void zeroCompositeHistoryNotExcluded() {
        EngagementState state = Module9TestBuilder.disengagedState("patient-009");
        state.setValidHistoryDays(14);
        // All zeros in history — should not be excluded from phenotype calc
        // olderAvg = 0, recentAvg = 0 -> SPORADIC (both zero path)

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        assertEquals("SPORADIC", result.phenotype);
    }

    @Test
    @DisplayName("R7: Older period all zero, recent period improving -> STEADY")
    void improvingFromZero() {
        EngagementState state = new EngagementState();
        state.setPatientId("patient-010");
        state.setValidHistoryDays(14);

        double[] history = state.getCompositeHistory14d();
        for (int i = 0; i < 7; i++) history[i] = 0.50;   // recent: engaged
        for (int i = 7; i < 14; i++) history[i] = 0.0;    // older: zero

        Module9ScoreComputer.Result result = Module9ScoreComputer.compute(state);
        assertEquals("STEADY", result.phenotype);
    }
}
