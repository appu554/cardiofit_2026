package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module11TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

class Module11SessionWindowTest {

    private static final long BASE = Module11TestBuilder.BASE_TIME;
    private static final long MIN = 60_000L;
    private static final long HOUR = 3_600_000L;

    @Test
    void openSession_timerAt_durationPlus2hPlus5m() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        long timerFireTime = state.openSession("act-1", BASE, 45 * MIN, new HashMap<>());
        // 45 min activity + 2h recovery + 5 min grace = 2h50m
        long expected = BASE + 45 * MIN + 2 * HOUR + 5 * MIN;
        assertEquals(expected, timerFireTime);
    }

    @Test
    void hrReadings_addedDuringExerciseAndRecovery() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());

        // HR during exercise
        state.addHRReading(BASE + 10 * MIN, 140.0, "WEARABLE");
        state.addHRReading(BASE + 20 * MIN, 155.0, "WEARABLE");
        // HR during recovery (after 30 min activity end)
        state.addHRReading(BASE + 35 * MIN, 130.0, "WEARABLE");
        state.addHRReading(BASE + 60 * MIN, 100.0, "WEARABLE");

        assertEquals(4, state.getActiveSessions().get("act-1").hrWindow.size());
    }

    @Test
    void hrReadings_ignoredOutsideWindow() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());

        // HR 3 hours after exercise end — outside 2h recovery window
        state.addHRReading(BASE + 30 * MIN + 3 * HOUR, 75.0, "WEARABLE");

        assertEquals(0, state.getActiveSessions().get("act-1").hrWindow.size());
    }

    @Test
    void preExerciseBP_retroactiveAttachment() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        // BP 20 min before exercise (within 30 min lookback)
        state.addBPReading(BASE - 20 * MIN, 125.0, 82.0);
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());

        ActivityCorrelationState.ActivitySession session = state.getActiveSessions().get("act-1");
        assertTrue(session.bpWindow.hasPreMeal()); // reusing BPWindow: pre-meal = pre-exercise
        assertEquals(125.0, session.bpWindow.getPreMealSBP());
    }

    @Test
    void peakExerciseBP_trackedDuringActivity() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());

        state.addBPReading(BASE + 10 * MIN, 155.0, 88.0);
        state.addBPReading(BASE + 20 * MIN, 170.0, 92.0); // higher → peak
        state.addBPReading(BASE + 25 * MIN, 160.0, 90.0);

        ActivityCorrelationState.ActivitySession session = state.getActiveSessions().get("act-1");
        assertEquals(170.0, session.peakExerciseSBP);
    }

    @Test
    void durationCap_maxAt4hours() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        // Report 6h activity — should be capped at 4h
        long timerFireTime = state.openSession("act-1", BASE, 6 * HOUR, new HashMap<>());
        long expected = BASE + 4 * HOUR + 2 * HOUR + 5 * MIN; // capped at 4h + 2h + 5m
        assertEquals(expected, timerFireTime);
    }

    @Test
    void closeSession_removesFromActive() {
        ActivityCorrelationState state = Module11TestBuilder.emptyState("P1");
        state.openSession("act-1", BASE, 30 * MIN, new HashMap<>());
        assertEquals(1, state.getActiveSessions().size());

        ActivityCorrelationState.ActivitySession closed = state.closeSession("act-1");
        assertNotNull(closed);
        assertEquals(0, state.getActiveSessions().size());
    }
}
