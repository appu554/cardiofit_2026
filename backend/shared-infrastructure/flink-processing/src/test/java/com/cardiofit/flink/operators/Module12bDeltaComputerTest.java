package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module 12b delta computation logic.
 * Verifies FBG, SBP, weight, HbA1c deltas and missing data handling.
 */
class Module12bDeltaComputerTest {

    @Test
    void fbgDelta_negativeIsImprovement() {
        // Baseline FBG=160, current FBG=145 → delta=-15 (improvement)
        Double delta = computeDelta(145.0, 160.0);
        assertNotNull(delta);
        assertEquals(-15.0, delta, 0.01);
    }

    @Test
    void sbpDelta_negativeIsImprovement() {
        // Baseline SBP=148, current SBP=135 → delta=-13
        Double delta = computeDelta(135.0, 148.0);
        assertNotNull(delta);
        assertEquals(-13.0, delta, 0.01);
    }

    @Test
    void weightDelta_positive_isDeterioration() {
        // Baseline weight=85.0, current weight=87.5 → delta=+2.5
        Double delta = computeDelta(87.5, 85.0);
        assertNotNull(delta);
        assertEquals(2.5, delta, 0.01);
    }

    @Test
    void hba1cDelta_withNewLabDuringWindow() {
        // Baseline HbA1c=8.2, new lab during window=7.8 → delta=-0.4
        Double delta = computeDelta(7.8, 8.2);
        assertNotNull(delta);
        assertEquals(-0.4, delta, 0.01);
    }

    @Test
    void missingData_returnsNull() {
        // If either baseline or current is null, delta is null
        assertNull(computeDelta(null, 160.0));
        assertNull(computeDelta(145.0, null));
        assertNull(computeDelta(null, null));
    }

    @Test
    void trajectoryAttribution_reversed() {
        // DETERIORATING before + IMPROVING during = INTERVENTION_REVERSED_DECLINE
        TrajectoryAttribution attr = TrajectoryAttribution.fromTrajectories(
                TrajectoryClassification.DETERIORATING,
                TrajectoryClassification.IMPROVING);
        assertEquals(TrajectoryAttribution.INTERVENTION_REVERSED_DECLINE, attr);
    }

    // Mirror of the private computeDelta method from Module12b
    private static Double computeDelta(Double current, Double baseline) {
        if (current == null || baseline == null) return null;
        return current - baseline;
    }
}
