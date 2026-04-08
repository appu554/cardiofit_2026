package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.thresholds.CIDThresholdSet;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Phase 2e: Verify SOFT_FLAG evaluator respects externalized thresholds.
 */
class Module8SOFTFLAGThresholdOverrideTest {

    // ── CID-12: Polypharmacy threshold override ──

    @Test
    void cid12_firesAtLoweredPolypharmacyThreshold() {
        // 6 medications — below hardcoded 8 but at/above tightened 5
        ComorbidityState state = new ComorbidityState("P-POLY-TIGHT");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.addMedication("atorvastatin", "STATIN", 20.0);
        state.addMedication("aspirin", "ANTIPLATELET", 75.0);

        // Should NOT fire at default (8)
        List<CIDAlert> defaultAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "6 meds should not fire CID-12 at default threshold 8");

        // Should fire at tightened (5)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setPolypharmacyThreshold(5);
        List<CIDAlert> tightenedAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "6 meds should fire CID-12 at tightened threshold 5");
    }

    // ── CID-13: Elderly age threshold override ──

    @Test
    void cid13_firesAtLoweredElderlyAge() {
        // Age 70 — below hardcoded 75 but at/above tightened 65
        ComorbidityState state = new ComorbidityState("P-ELDER-TIGHT");
        state.setAge(70);
        state.setEGFRCurrent(40.0); // risk factor: eGFR < 45
        state.setEGFRBaseline(50.0);

        Double sbpTarget = 120.0; // intensive target

        // Should NOT fire at default elderly threshold (75)
        List<CIDAlert> defaultAlerts = Module8SOFTFLAGEvaluator.evaluate(state, sbpTarget);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 70 should not fire CID-13 at default elderly threshold 75");

        // Should fire at tightened (65)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setElderlyAgeThreshold(65);
        List<CIDAlert> tightenedAlerts = Module8SOFTFLAGEvaluator.evaluate(state, sbpTarget, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 70 should fire CID-13 at tightened elderly threshold 65");
    }

    // ── CID-14: Metformin eGFR range override ──

    @Test
    void cid14_firesAtWidenedEGFRRange() {
        // eGFR = 38 — above hardcoded high bound (35) but within widened (40)
        ComorbidityState state = new ComorbidityState("P-MET-WIDE");
        state.addMedication("metformin", "METFORMIN", 1000.0);
        state.setEGFRCurrent(38.0);
        state.setEGFRBaseline(45.0);
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 90L * 86_400_000L);
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());

        // Should NOT fire at default range (30-35)
        List<CIDAlert> defaultAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_14".equals(a.getRuleId())),
            "eGFR 38 should not fire CID-14 at default range 30-35");

        // Should fire at widened range (30-40)
        CIDThresholdSet widened = CIDThresholdSet.hardcodedDefaults();
        widened.setEgfrMetforminHigh(40.0);
        List<CIDAlert> widenedAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null, widened);
        assertTrue(widenedAlerts.stream().anyMatch(a -> "CID_14".equals(a.getRuleId())),
            "eGFR 38 should fire CID-14 at widened range 30-40");
    }

    // ── CID-17: Fasting duration threshold override ──

    @Test
    void cid17_firesAtLoweredFastingDuration() {
        // Fasting 14h — below hardcoded 16 but above tightened 12
        ComorbidityState state = new ComorbidityState("P-FAST-TIGHT");
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setActiveFastingPeriod(true);
        state.setActiveFastingDurationHours(14);

        // Should NOT fire at default (16)
        List<CIDAlert> defaultAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "14h fasting should not fire CID-17 at default threshold 16");

        // Should fire at tightened (12)
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setFastingDurationThreshold(12);
        List<CIDAlert> tightenedAlerts = Module8SOFTFLAGEvaluator.evaluate(state, null, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "14h fasting should fire CID-17 at tightened threshold 12");
    }
}
