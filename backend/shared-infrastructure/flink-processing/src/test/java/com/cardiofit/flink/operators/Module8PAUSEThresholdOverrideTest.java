package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.thresholds.CIDThresholdSet;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Phase 2e: Verify PAUSE evaluator respects externalized thresholds.
 *
 * PAUSE rules have no safety clamping — clinical context may legitimately
 * relax thresholds (e.g., relaxing FBG delta for elderly patients).
 */
class Module8PAUSEThresholdOverrideTest {

    private static final long NOW = System.currentTimeMillis();

    // ── CID-06: Thiazide + FBG delta override ──

    @Test
    void cid06_firesAtTightenedFBGDelta() {
        // FBG delta = 12 mg/dL in 14d — below hardcoded 15 but above tightened 10
        ComorbidityState state = new ComorbidityState("P-FBG-TIGHT");
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addToRollingBuffer("fbg", 100.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 112.0, NOW);

        // Should NOT fire at default (15)
        List<CIDAlert> defaultAlerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())),
            "FBG delta 12 should not fire CID-06 at default threshold 15");

        // Should fire at tightened (10)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setFbgDeltaThreshold(10.0);
        List<CIDAlert> tightenedAlerts = Module8PAUSEEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())),
            "FBG delta 12 should fire CID-06 at tightened threshold 10");
    }

    // ── CID-07: eGFR decline + dip window override ──

    @Test
    void cid07_firesAtTightenedEGFRDecline() {
        // eGFR decline = 22% — below hardcoded 25% but above tightened 20%
        ComorbidityState state = new ComorbidityState("P-EGFR-TIGHT");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.setEGFRBaseline(100.0);
        state.setEGFRBaselineTimestamp(NOW - 90L * 86_400_000L);
        state.setEGFRCurrent(78.0);  // 22% decline from 100
        state.setEGFRCurrentTimestamp(NOW);

        // Should NOT fire at default decline threshold (25%)
        List<CIDAlert> defaultAlerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "22% eGFR decline should not fire CID-07 at default 25%");

        // Should fire at tightened decline threshold (20%)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setEgfrDeclineThresholdPct(20.0);
        List<CIDAlert> tightenedAlerts = Module8PAUSEEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "22% eGFR decline should fire CID-07 at tightened 20%");
    }

    // ── CID-10: Concurrent deterioration thresholds ──

    @Test
    void cid10_firesAtTightenedGlucoseAndSBPWorsening() {
        // FBG +8 mg/dL, SBP +8 mmHg — below hardcoded 10 each, above tightened 5
        ComorbidityState state = new ComorbidityState("P-CONC-TIGHT");
        state.addToRollingBuffer("fbg", 100.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 108.0, NOW);
        state.addToRollingBuffer("sbp", 130.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("sbp", 138.0, NOW);
        // No medication change — required for CID-10
        state.setLastMedicationChangeTimestamp(NOW - 30L * 86_400_000L);

        // Should NOT fire at default (10 each)
        List<CIDAlert> defaultAlerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "+8 glucose/SBP should not fire CID-10 at default threshold 10");

        // Should fire at tightened (5 each)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setGlucoseWorseningThreshold(5.0);
        tightened.setSbpWorseningThreshold(5.0);
        List<CIDAlert> tightenedAlerts = Module8PAUSEEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "+8 glucose/SBP should fire CID-10 at tightened threshold 5");
    }
}
