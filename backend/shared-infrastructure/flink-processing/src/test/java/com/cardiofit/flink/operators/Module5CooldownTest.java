package com.cardiofit.flink.operators;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.*;

import static org.junit.jupiter.api.Assertions.*;

class Module5CooldownTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    @DisplayName("First event for patient — always triggers inference")
    void firstEvent_alwaysTriggersInference() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), 0, NOW));
    }

    @Test
    @DisplayName("High NEWS2 (>=7) — no cooldown")
    void highNews2_noCooldown() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            7, 0, Map.of(), NOW - 1000, NOW));
    }

    @Test
    @DisplayName("High qSOFA (>=2) — no cooldown")
    void highQsofa_noCooldown() {
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            3, 2, Map.of(), NOW - 1000, NOW));
    }

    @Test
    @DisplayName("Gap 3: AKI patient (NEWS2=1, hyperkalemia=true) — no cooldown")
    void akiPatient_labCritical_noCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("hyperkalemia", true);
        risk.put("elevatedCreatinine", true);

        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, risk, NOW - 1000, NOW),
            "AKI patient with normal vitals should bypass cooldown");
    }

    @Test
    @DisplayName("Gap 3: Drug-lab patient (NEWS2=1, thrombocytopenia=true) — no cooldown")
    void drugLabPatient_thrombocytopenia_noCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("thrombocytopenia", true);

        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, risk, NOW - 1000, NOW),
            "Thrombocytopenic patient should bypass cooldown");
    }

    @Test
    @DisplayName("Moderate NEWS2 (5-6) — 10s cooldown")
    void moderateNews2_10sCooldown() {
        // Within 10s → blocked
        assertFalse(Module5ClinicalScoring.shouldRunInference(
            5, 0, Map.of(), NOW - 5000, NOW));
        // After 10s → allowed
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            5, 0, Map.of(), NOW - 11000, NOW));
    }

    @Test
    @DisplayName("Elevated lactate — 10s cooldown (moderate tier)")
    void elevatedLactate_10sCooldown() {
        Map<String, Object> risk = new HashMap<>();
        risk.put("elevatedLactate", true);

        assertFalse(Module5ClinicalScoring.shouldRunInference(
            2, 0, risk, NOW - 5000, NOW));
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            2, 0, risk, NOW - 11000, NOW));
    }

    @Test
    @DisplayName("Stable patient (NEWS2=1) — 30s cooldown")
    void stablePatient_30sCooldown() {
        assertFalse(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), NOW - 15000, NOW));
        assertTrue(Module5ClinicalScoring.shouldRunInference(
            1, 0, Map.of(), NOW - 31000, NOW));
    }
}
