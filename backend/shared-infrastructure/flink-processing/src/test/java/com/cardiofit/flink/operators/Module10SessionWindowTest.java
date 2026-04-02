package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module10TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.Map;
import java.util.HashMap;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 10 session window mechanics:
 * - Meal opens window, glucose fills, timer closes
 * - Pre-meal BP retroactive attachment
 * - Window duration and grace period
 */
class Module10SessionWindowTest {

    private static final long BASE = Module10TestBuilder.BASE_TIME;
    private static final long MIN_5 = 5 * 60_000L;
    private static final long HOUR = 3_600_000L;

    @Test
    void openSession_setsTimerAt3h05m() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        long mealTime = BASE;
        Map<String, Object> payload = new HashMap<>();
        payload.put("carb_grams", 50.0);

        long timerFireTime = state.openSession("meal-1", mealTime, payload);

        long expected = mealTime + 3 * HOUR + 5 * 60_000L;
        assertEquals(expected, timerFireTime);
        assertEquals(1, state.getActiveSessions().size());
    }

    @Test
    void glucoseReadings_addedToActiveSession() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // Add CGM readings within 3h window
        state.addGlucoseReading(BASE + 10 * MIN_5, 120.0, "CGM");
        state.addGlucoseReading(BASE + 20 * MIN_5, 150.0, "CGM");
        state.addGlucoseReading(BASE + 30 * MIN_5, 130.0, "CGM");

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertEquals(3, session.glucoseWindow.size());
    }

    @Test
    void glucoseReadings_ignoredOutsideWindow() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // Reading after 3h window → should NOT be added
        state.addGlucoseReading(BASE + 4 * HOUR, 120.0, "CGM");

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertEquals(0, session.glucoseWindow.size());
    }

    @Test
    void preMealBP_retroactiveAttachment() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");

        // BP reading 30 min before meal
        state.addBPReading(BASE - 30 * 60_000L, 120.0, 80.0);

        // Meal arrives — should attach BP as pre-meal
        state.openSession("meal-1", BASE, null);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertTrue(session.bpWindow.hasPreMeal());
        assertEquals(120.0, session.bpWindow.getPreMealSBP());
    }

    @Test
    void preMealBP_notAttachedIfTooOld() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");

        // BP reading 90 min before meal (> 60 min lookback)
        state.addBPReading(BASE - 90 * 60_000L, 120.0, 80.0);

        state.openSession("meal-1", BASE, null);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertFalse(session.bpWindow.hasPreMeal());
    }

    @Test
    void postMealBP_capturedWithin4h() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);

        // BP reading 2h after meal → captured as post-meal
        state.addBPReading(BASE + 2 * HOUR, 135.0, 85.0);

        MealCorrelationState.MealSession session = state.getActiveSessions().get("meal-1");
        assertTrue(session.bpWindow.hasPostMeal());
        assertEquals(135.0, session.bpWindow.getPostMealSBP());
    }

    @Test
    void closeSession_removesFromActive() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        assertEquals(1, state.getActiveSessions().size());

        MealCorrelationState.MealSession closed = state.closeSession("meal-1");
        assertNotNull(closed);
        assertEquals(0, state.getActiveSessions().size());
    }
}
