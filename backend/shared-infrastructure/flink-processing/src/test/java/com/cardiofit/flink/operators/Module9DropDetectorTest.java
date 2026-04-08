package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.Optional;

import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Module 9: Drop Detector (3-day persistence, R3 Fix)")
class Module9DropDetectorTest {

    private static final String PID = "test-patient-001";
    private static final long NOW = System.currentTimeMillis();

    private EngagementState freshState() {
        EngagementState state = new EngagementState();
        state.setPatientId(PID);
        return state;
    }

    @Test
    @DisplayName("R3: YELLOW -> ORANGE, day 1 -> NO alert (persistence not met)")
    void levelTransitionDay1NoAlert() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.30, EngagementLevel.ORANGE,
            0.50, EngagementLevel.YELLOW,
            0, 1, // consecutiveDaysAtLevel = 1 (day 1)
            freshState(), NOW
        );
        assertTrue(alert.isEmpty(), "Should not fire on day 1 — 3-day persistence required");
    }

    @Test
    @DisplayName("R3: YELLOW -> ORANGE, day 3 -> alert emitted (persistence met)")
    void levelTransitionDay3Alert() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.30, EngagementLevel.ORANGE,
            0.50, EngagementLevel.YELLOW,
            0, 3, // consecutiveDaysAtLevel = 3
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        assertEquals(EngagementDropAlert.DropType.LEVEL_TRANSITION, alert.get().getDropType());
        assertEquals("WARNING", alert.get().getSeverity());
    }

    @Test
    @DisplayName("R3: ORANGE -> RED transition fires IMMEDIATELY (patient safety)")
    void redTransitionFiresImmediately() {
        // Use a delta < 0.30 to avoid triggering cliff drop first
        // ORANGE at 0.25 -> RED at 0.10 = delta 0.15
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.10, EngagementLevel.RED,
            0.25, EngagementLevel.ORANGE,
            0, 1, // day 1 — RED should still fire immediately
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        assertEquals(EngagementDropAlert.DropType.LEVEL_TRANSITION, alert.get().getDropType());
        assertEquals("CRITICAL", alert.get().getSeverity());
    }

    @Test
    @DisplayName("R3: ORANGE -> RED transition fires IMMEDIATELY")
    void orangeToRedFiresImmediately() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.10, EngagementLevel.RED,
            0.25, EngagementLevel.ORANGE,
            3, 1,
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        assertEquals("CRITICAL", alert.get().getSeverity());
    }

    @Test
    @DisplayName("GREEN -> GREEN (no change) -> no alert")
    void noTransitionNoAlert() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.80, EngagementLevel.GREEN,
            0.85, EngagementLevel.GREEN,
            0, 5,
            freshState(), NOW
        );
        assertTrue(alert.isEmpty());
    }

    @Test
    @DisplayName("ORANGE for 4 consecutive days -> no sustained alert")
    void sustainedLowNotYet() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.25, EngagementLevel.ORANGE,
            0.25, EngagementLevel.ORANGE, // same level, no transition
            4, 4, // 4 consecutive low days
            freshState(), NOW
        );
        assertTrue(alert.isEmpty(), "Sustained alert requires 5 days, not 4");
    }

    @Test
    @DisplayName("ORANGE for 5 consecutive days -> sustained alert")
    void sustainedLowFires() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.25, EngagementLevel.ORANGE,
            0.25, EngagementLevel.ORANGE,
            5, 5,
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        assertEquals(EngagementDropAlert.DropType.SUSTAINED_LOW, alert.get().getDropType());
    }

    @Test
    @DisplayName("Score drops 0.31 in one day -> cliff drop alert (CRITICAL)")
    void cliffDrop() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.39, EngagementLevel.ORANGE,
            0.70, EngagementLevel.GREEN,
            0, 1,
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        assertEquals(EngagementDropAlert.DropType.CLIFF_DROP, alert.get().getDropType());
        assertEquals("CRITICAL", alert.get().getSeverity());
    }

    @Test
    @DisplayName("Score drops 0.29 -> no cliff drop alert")
    void notQuiteCliffDrop() {
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.51, EngagementLevel.YELLOW,
            0.80, EngagementLevel.GREEN,
            0, 1,
            freshState(), NOW
        );
        // delta = 0.29, below 0.30 threshold
        assertTrue(alert.isEmpty());
    }

    @Test
    @DisplayName("Alert suppression: emit once, suppress for 7 days")
    void alertSuppression() {
        EngagementState state = freshState();

        // First alert should fire
        Optional<EngagementDropAlert> first = Module9DropDetector.detect(
            PID, 0.25, EngagementLevel.ORANGE,
            0.25, EngagementLevel.ORANGE,
            5, 5, state, NOW
        );
        assertTrue(first.isPresent());
        state.recordAlertEmission(first.get().getSuppressionKey(), NOW);

        // Same alert type next day should be suppressed
        Optional<EngagementDropAlert> suppressed = Module9DropDetector.detect(
            PID, 0.25, EngagementLevel.ORANGE,
            0.25, EngagementLevel.ORANGE,
            6, 6, state, NOW + 86_400_000L // 1 day later
        );
        assertTrue(suppressed.isEmpty(), "Should be suppressed within 7-day window");
    }

    @Test
    @DisplayName("Alert suppression: re-emit after 7 days expire")
    void alertSuppressionExpires() {
        EngagementState state = freshState();

        // Record suppression
        String key = "SUSTAINED_LOW:" + PID;
        state.recordAlertEmission(key, NOW);

        // 8 days later — should no longer be suppressed
        long eightDaysLater = NOW + (8L * 86_400_000L);
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.25, EngagementLevel.ORANGE,
            0.25, EngagementLevel.ORANGE,
            5, 5, state, eightDaysLater
        );
        assertTrue(alert.isPresent(), "Should fire after 7-day suppression expires");
    }

    @Test
    @DisplayName("Cliff drop has priority over level transition")
    void cliffDropPriority() {
        // Score drops from 0.75 to 0.30 = delta 0.45 (cliff drop)
        // AND it's a GREEN->ORANGE level transition
        Optional<EngagementDropAlert> alert = Module9DropDetector.detect(
            PID, 0.30, EngagementLevel.ORANGE,
            0.75, EngagementLevel.GREEN,
            0, 3,
            freshState(), NOW
        );
        assertTrue(alert.isPresent());
        // Cliff drop is checked first, so it wins
        assertEquals(EngagementDropAlert.DropType.CLIFF_DROP, alert.get().getDropType());
    }
}
