package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for the three gap fixes:
 *   A. Cuffless ARV (reading-to-reading, research only)
 *   B. Separate ACUTE_SURGE_TAG OutputTag
 *   C. Within-day SD of systolic readings
 */
class Module7GapFixTest {

    // ─── Gap C: Within-day SD ───

    @Test
    void withinDaySd_withThreeReadings_computesCorrectly() {
        // 3 readings: 120, 130, 140 → mean=130, variance=100, SD=10
        DailyBPSummary summary = new DailyBPSummary("2026-04-01");
        summary.addReading(Module7TestBuilder.morningReading("P1", 120, 75, System.currentTimeMillis()));
        summary.addReading(Module7TestBuilder.reading("P1", 130, 80,
            System.currentTimeMillis() + 3600_000, "AFTERNOON", "HOME_CUFF"));
        summary.addReading(Module7TestBuilder.eveningReading("P1", 140, 85,
            System.currentTimeMillis() + 7200_000));

        Double sd = summary.getWithinDaySdSBP();
        assertNotNull(sd, "3 readings should produce within-day SD");
        assertEquals(10.0, sd, 0.01, "SD of {120,130,140} should be 10");
    }

    @Test
    void withinDaySd_withTwoReadings_returnsNull() {
        DailyBPSummary summary = new DailyBPSummary("2026-04-01");
        summary.addReading(Module7TestBuilder.morningReading("P1", 120, 75, System.currentTimeMillis()));
        summary.addReading(Module7TestBuilder.eveningReading("P1", 140, 85,
            System.currentTimeMillis() + 7200_000));

        assertNull(summary.getWithinDaySdSBP(), "< 3 readings → null");
    }

    @Test
    void withinDaySd_withIdenticalReadings_returnsZero() {
        DailyBPSummary summary = new DailyBPSummary("2026-04-01");
        long ts = System.currentTimeMillis();
        summary.addReading(Module7TestBuilder.morningReading("P1", 130, 80, ts));
        summary.addReading(Module7TestBuilder.reading("P1", 130, 80, ts + 3600_000, "AFTERNOON", "HOME_CUFF"));
        summary.addReading(Module7TestBuilder.eveningReading("P1", 130, 80, ts + 7200_000));

        assertEquals(0.0, summary.getWithinDaySdSBP(), 1e-9, "Identical readings → SD 0");
    }

    // ─── Gap A: Cuffless ARV ───

    @Test
    void cufflessArv_withThreeReadings_computesReadingToReading() {
        PatientBPState state = new PatientBPState("P-CUFF");
        // Cuffless SBP sequence: 120, 130, 125
        // |130-120| = 10, |125-130| = 5 → ARV = 7.5
        state.addCufflessReading(120.0);
        state.addCufflessReading(130.0);
        state.addCufflessReading(125.0);

        Double arv = state.getCufflessARV();
        assertNotNull(arv);
        assertEquals(7.5, arv, 1e-9, "ARV of |10|+|5| / 2 = 7.5");
    }

    @Test
    void cufflessArv_withTwoReadings_returnsNull() {
        PatientBPState state = new PatientBPState("P-CUFF");
        state.addCufflessReading(120.0);
        state.addCufflessReading(130.0);

        assertNull(state.getCufflessARV(), "< 3 cuffless readings → null");
    }

    @Test
    void cufflessArv_bufferCappedAt50() {
        PatientBPState state = new PatientBPState("P-CUFF");
        for (int i = 0; i < 60; i++) {
            state.addCufflessReading(120.0 + i);
        }
        assertEquals(50, state.getCufflessSBPBuffer().size(),
            "Buffer should be capped at 50");
        // First 10 evicted, buffer starts at 130
        assertEquals(130.0, state.getCufflessSBPBuffer().get(0), 1e-9);
    }

    // ─── Gap B: Separate ACUTE_SURGE_TAG ───

    @Test
    void acuteSurgeTag_isDifferentFromCrisisTag() {
        assertNotEquals(
            Module7_BPVariabilityEngine.CRISIS_TAG.getId(),
            Module7_BPVariabilityEngine.ACUTE_SURGE_TAG.getId(),
            "ACUTE_SURGE_TAG must have different ID from CRISIS_TAG");
    }

    @Test
    void acuteSurgeDetection_sbpJumpOver30_detected() {
        BPReading prev = Module7TestBuilder.morningReading("P1", 140, 85, System.currentTimeMillis());
        // 40 mmHg jump in 30 minutes
        BPReading current = Module7TestBuilder.morningReading("P1", 180, 95,
            System.currentTimeMillis() + 30 * 60 * 1000L);
        assertTrue(Module7CrisisDetector.isAcuteSurge(prev, current),
            "SBP delta 40 in 30min should be acute surge");
        // 180 is >= 180, so it IS a crisis per ACC/AHA
        assertTrue(Module7CrisisDetector.isCrisis(current),
            "SBP=180 IS >= 180, so it is a crisis per ACC/AHA");
    }

    @Test
    void acuteSurgeDetection_sbpJumpOver30ButOver1Hr_notDetected() {
        BPReading prev = Module7TestBuilder.morningReading("P1", 140, 85, System.currentTimeMillis());
        // Same delta but 2 hours apart
        BPReading current = Module7TestBuilder.morningReading("P1", 180, 95,
            System.currentTimeMillis() + 2 * 60 * 60 * 1000L);
        assertFalse(Module7CrisisDetector.isAcuteSurge(prev, current),
            "SBP delta 40 in 2hr should NOT be acute surge (>1hr window)");
    }
}
