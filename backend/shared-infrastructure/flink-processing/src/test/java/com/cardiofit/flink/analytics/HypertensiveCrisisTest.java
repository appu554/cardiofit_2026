package com.cardiofit.flink.analytics;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class HypertensiveCrisisTest {

    @Test
    void testCrisis_SBPAbove180() {
        assertTrue(HypertensiveCrisisDetector.isCrisis(185.0, 90.0));
    }

    @Test
    void testCrisis_DBPAbove120() {
        assertTrue(HypertensiveCrisisDetector.isCrisis(140.0, 125.0));
    }

    @Test
    void testNoCrisis_NormalBP() {
        assertFalse(HypertensiveCrisisDetector.isCrisis(135.0, 85.0));
    }

    @Test
    void testCuffConfirmation_Required() {
        assertTrue(HypertensiveCrisisDetector.requiresCuffConfirmation(true));
        assertFalse(HypertensiveCrisisDetector.requiresCuffConfirmation(false));
    }
}
