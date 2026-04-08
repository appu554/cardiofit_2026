package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.thresholds.CIDThresholdSet;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Phase 2e: Verify HALT evaluator respects externalized thresholds.
 *
 * Each test constructs a patient state that would NOT fire at hardcoded defaults
 * but DOES fire at a tightened threshold — proving the CIDThresholdSet parameter
 * is actually driving evaluator logic.
 */
class Module8HALTThresholdOverrideTest {

    private static final long NOW = System.currentTimeMillis();

    // ── CID-02: Hyperkalemia — K+ threshold override ──

    @Test
    void cid02_firesAtTightenedPotassiumThreshold() {
        // K+ = 5.1, rising from 4.9 — below hardcoded 5.3 but above tightened 5.0
        ComorbidityState state = new ComorbidityState("P-K-TIGHT");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.setPreviousPotassium(4.9);
        state.updateLab("potassium", 5.1);

        // Should NOT fire at hardcoded defaults (5.3)
        List<CIDAlert> defaultAlerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "K+ 5.1 should not fire CID-02 at default threshold 5.3");

        // Should fire with tightened threshold (5.0) — e.g., CKD G4 patient
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setPotassiumThreshold(5.0);
        List<CIDAlert> tightenedAlerts = Module8HALTEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "K+ 5.1 should fire CID-02 at tightened threshold 5.0");
    }

    @Test
    void cid02_haltSafetyClampPreventsRelaxation() {
        // K+ = 5.5, rising from 5.3 — fires at default 5.3
        ComorbidityState state = new ComorbidityState("P-K-RELAX");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.setPreviousPotassium(5.3);
        state.updateLab("potassium", 5.5);

        // Should fire at hardcoded defaults (5.3)
        List<CIDAlert> defaultAlerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(defaultAlerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "K+ 5.5 should fire CID-02 at default threshold 5.3");

        // Even with a relaxed CIDThresholdSet (5.8), the evaluator uses CIDThresholdResolver
        // which clamps HALT thresholds: min(5.8, 5.3) = 5.3. K+ 5.5 > 5.3 → still fires.
        // This proves the safety invariant: HALT rules can only be tightened, never relaxed.
        CIDThresholdSet relaxed = CIDThresholdSet.hardcodedDefaults();
        relaxed.setPotassiumThreshold(5.8);
        List<CIDAlert> relaxedAlerts = Module8HALTEvaluator.evaluate(state, NOW, relaxed);
        assertTrue(relaxedAlerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "K+ 5.5 should STILL fire CID-02 — HALT safety clamp prevents relaxation to 5.8");
    }

    // ── CID-03: Hypo Masking — glucose threshold override ──

    @Test
    void cid03_firesAtTightenedGlucoseThreshold() {
        // Glucose = 65 — above hardcoded 60 but below tightened 70 (elderly patient)
        ComorbidityState state = new ComorbidityState("P-HYPO-TIGHT");
        state.addMedication("insulin glargine", "INSULIN", 30.0);
        state.addMedication("metoprolol", "BETA_BLOCKER", 100.0);
        state.setLatestGlucose(65.0);
        state.setSymptomReportedHypoglycemia(false);

        // Should NOT fire at default (60)
        List<CIDAlert> defaultAlerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Glucose 65 should not fire CID-03 at default threshold 60");

        // Should fire with tightened threshold (70) — ADA guidance for elderly
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setGlucoseHypoThreshold(70.0);
        List<CIDAlert> tightenedAlerts = Module8HALTEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Glucose 65 should fire CID-03 at tightened threshold 70");
    }

    // ── CID-05: Severe Hypotension — SBP + antihypertensive count override ──

    @Test
    void cid05_firesAtTightenedSBPThreshold() {
        // SBP = 97 — above hardcoded 95 but below tightened 100 (elderly)
        ComorbidityState state = new ComorbidityState("P-HYPO-SBP");
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("losartan", "ARB", 100.0);
        state.addMedication("hydrochlorothiazide", "THIAZIDE", 25.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(97.0);

        // Should NOT fire at default SBP threshold (95)
        List<CIDAlert> defaultAlerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "SBP 97 should not fire CID-05 at default threshold 95");

        // Should fire with tightened threshold (100) — e.g., age ≥80
        CIDThresholdSet tightened = CIDThresholdSet.hardcodedDefaults();
        tightened.setSbpHypotensionThreshold(100.0);
        List<CIDAlert> tightenedAlerts = Module8HALTEvaluator.evaluate(state, NOW, tightened);
        assertTrue(tightenedAlerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "SBP 97 should fire CID-05 at tightened threshold 100");
    }

    @Test
    void cid05_firesAtLowerMinAntihypertensives() {
        // Only 2 antihypertensives — below hardcoded minimum 3 but at relaxed 2
        ComorbidityState state = new ComorbidityState("P-AH-MIN");
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("losartan", "ARB", 100.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(90.0); // clearly low

        // Should NOT fire at default (min 3 antihypertensives) — only has 2
        List<CIDAlert> defaultAlerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(defaultAlerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "2 antihypertensives should not fire CID-05 at default minimum 3");

        // Should fire with lowered minimum (2)
        CIDThresholdSet lowered = CIDThresholdSet.hardcodedDefaults();
        lowered.setMinAntihypertensives(2);
        List<CIDAlert> loweredAlerts = Module8HALTEvaluator.evaluate(state, NOW, lowered);
        assertTrue(loweredAlerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "2 antihypertensives should fire CID-05 at lowered minimum 2");
    }
}
