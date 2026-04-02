package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module10TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for overlapping meal detection (second meal within 90 min).
 */
class Module10OverlappingMealTest {

    private static final long BASE = Module10TestBuilder.BASE_TIME;
    private static final long MIN = 60_000L;

    @Test
    void secondMealWithin90min_bothFlagged() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 60 * MIN, null); // 60 min later

        assertTrue(state.getActiveSessions().get("meal-1").overlapping);
        assertTrue(state.getActiveSessions().get("meal-2").overlapping);
    }

    @Test
    void secondMealAfter90min_notFlagged() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 100 * MIN, null); // 100 min later

        assertFalse(state.getActiveSessions().get("meal-1").overlapping);
        assertFalse(state.getActiveSessions().get("meal-2").overlapping);
    }

    @Test
    void glucoseReadings_feedBothOverlappingSessions() {
        MealCorrelationState state = Module10TestBuilder.emptyState("P1");
        state.openSession("meal-1", BASE, null);
        state.openSession("meal-2", BASE + 60 * MIN, null);

        // Reading at 90 min — within both windows
        state.addGlucoseReading(BASE + 90 * MIN, 150.0, "CGM");

        assertEquals(1, state.getActiveSessions().get("meal-1").glucoseWindow.size());
        assertEquals(1, state.getActiveSessions().get("meal-2").glucoseWindow.size());
    }
}
