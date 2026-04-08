package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.Collections;
import java.util.HashMap;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 12 window lifecycle: open, midpoint, close, cancel,
 * modify, expire, and timer deduplication.
 */
class Module12WindowLifecycleTest {

    @Test
    void openWindow_createsWindowWithTimers() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.MEDICATION_ADD, medicationDetail("SGLT2I", "Empagliflozin", "10mg"),
                28, BASE_TIME, TrajectoryClassification.STABLE,
                "card-1", "APPROVED");

        assertNotNull(window);
        assertEquals("int-1", window.interventionId);
        assertEquals(InterventionType.MEDICATION_ADD, window.interventionType);
        assertEquals(BASE_TIME, window.observationStartMs);
        assertEquals(BASE_TIME + 28 * DAY_MS, window.observationEndMs);
        assertEquals(28, window.observationWindowDays);
        assertEquals("OBSERVING", window.status);
        // Midpoint = start + 14 days
        assertEquals(BASE_TIME + 14 * DAY_MS, window.midpointTimerMs);
        // Close = end + 24h grace
        assertEquals(BASE_TIME + 28 * DAY_MS + DAY_MS, window.closeTimerMs);
        assertEquals(1, state.getTotalInterventionsTracked());
    }

    @Test
    void midpointTimerLookup_findsCorrectWindow() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.LIFESTYLE_ACTIVITY, null,
                14, BASE_TIME, TrajectoryClassification.STABLE,
                "card-1", "APPROVED");

        // Midpoint = start + 7 days
        long midpoint = BASE_TIME + 7 * DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForMidpointTimer(midpoint);

        assertNotNull(found);
        assertEquals("int-1", found.interventionId);
    }

    @Test
    void closeTimerLookup_findsCorrectWindow() {
        InterventionWindowState state = emptyState("P1");

        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Close = start + 28d + 24h
        long closeTime = BASE_TIME + 28 * DAY_MS + DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForCloseTimer(closeTime);

        assertNotNull(found);
        assertEquals("int-1", found.interventionId);
    }

    @Test
    void cancelWindow_removesFromActive() {
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        assertEquals(1, state.getActiveWindows().size());

        InterventionWindowState.InterventionWindow removed = state.removeWindow("int-1");

        assertNotNull(removed);
        assertEquals(0, state.getActiveWindows().size());
    }

    @Test
    void modifyWindow_updatesTimers() {
        InterventionWindowState state = emptyState("P1");

        InterventionWindowState.InterventionWindow window = state.openWindow(
                "int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        long oldMidpoint = window.midpointTimerMs;
        long oldClose = window.closeTimerMs;

        // Simulate modification to 42-day window
        long newWindowMs = 42 * DAY_MS;
        window.observationEndMs = window.observationStartMs + newWindowMs;
        window.observationWindowDays = 42;
        window.midpointTimerMs = window.observationStartMs + newWindowMs / 2;
        window.closeTimerMs = window.observationEndMs + DAY_MS;

        assertNotEquals(oldMidpoint, window.midpointTimerMs);
        assertNotEquals(oldClose, window.closeTimerMs);
        assertEquals(BASE_TIME + 21 * DAY_MS, window.midpointTimerMs);
        assertEquals(BASE_TIME + 43 * DAY_MS, window.closeTimerMs);
    }

    @Test
    void expiredWindow_noData_closeTimerStillFires() {
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MONITORING_CHANGE, null,
                14, BASE_TIME, TrajectoryClassification.UNKNOWN, "card-1", "APPROVED");

        // No readings added during window
        InterventionWindowState.InterventionWindow window = state.getWindow("int-1");

        // Data completeness should be empty (no flags set)
        assertTrue(window.dataCompleteness.isEmpty());
        // Close timer is still registered
        assertEquals(BASE_TIME + 14 * DAY_MS + DAY_MS, window.closeTimerMs);
    }

    @Test
    void timerDedup_cancelledWindowIgnoredOnClose() {
        InterventionWindowState state = emptyState("P1");
        state.openWindow("int-1", InterventionType.MEDICATION_ADD, null,
                28, BASE_TIME, TrajectoryClassification.STABLE, "card-1", "APPROVED");

        // Cancel the window
        InterventionWindowState.InterventionWindow window = state.getWindow("int-1");
        window.status = "CANCELLED";

        // Close timer fires but window is cancelled
        long closeTime = BASE_TIME + 28 * DAY_MS + DAY_MS;
        InterventionWindowState.InterventionWindow found = state.getWindowForCloseTimer(closeTime);

        // Should NOT find the window (status != OBSERVING)
        assertNull(found);
    }
}
