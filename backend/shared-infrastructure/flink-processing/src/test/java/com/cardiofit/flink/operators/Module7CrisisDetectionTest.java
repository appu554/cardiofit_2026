package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

class Module7CrisisDetectionTest {

    @Test
    void sbpAbove180_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 185, 95,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "SBP 185 must trigger crisis");
    }

    @Test
    void dbpAbove120_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 165, 125,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "DBP 125 must trigger crisis");
    }

    @Test
    void bothAboveThreshold_isCrisis() {
        BPReading reading = Module7TestBuilder.crisisReading("P001");
        assertTrue(Module7CrisisDetector.isCrisis(reading));
    }

    @Test
    void normalReading_notCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 140, 88,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isCrisis(reading),
            "SBP 140 / DBP 88 is not crisis");
    }

    @Test
    void exactlyAt180_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 180, 110,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "SBP exactly 180 is crisis per ACC/AHA (threshold is >= 180)");
    }

    @Test
    void exactlyAt120DBP_isCrisis() {
        BPReading reading = Module7TestBuilder.reading("P001", 170, 120,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isCrisis(reading),
            "DBP exactly 120 is crisis per ACC/AHA (threshold is >= 120)");
    }

    @Test
    void acuteSurge_sbpJump30InOneHour() {
        BPReading previous = Module7TestBuilder.reading("P001", 135, 82,
            System.currentTimeMillis() - 45 * 60 * 1000L, "MORNING", "HOME_CUFF");
        BPReading current = Module7TestBuilder.reading("P001", 170, 95,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertTrue(Module7CrisisDetector.isAcuteSurge(previous, current),
            "SBP jump from 135 to 170 (35 mmHg) in 45 min should be acute surge");
    }

    @Test
    void acuteSurge_sbpJump20_notSurge() {
        BPReading previous = Module7TestBuilder.reading("P001", 135, 82,
            System.currentTimeMillis() - 45 * 60 * 1000L, "MORNING", "HOME_CUFF");
        BPReading current = Module7TestBuilder.reading("P001", 152, 88,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
        assertFalse(Module7CrisisDetector.isAcuteSurge(previous, current),
            "SBP jump of 17 mmHg should not be acute surge");
    }
}
