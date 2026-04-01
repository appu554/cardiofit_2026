package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import org.apache.flink.streaming.api.operators.KeyedProcessOperator;
import org.apache.flink.streaming.util.KeyedOneInputStreamOperatorTestHarness;
import org.apache.flink.api.common.typeinfo.Types;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * R9: Integration test that exercises the real KeyedProcessFunction.
 * Uses Flink's test harness to simulate processElement() calls
 * with proper keyed state, watermarks, and side-output collection.
 */
class Module8ProcessElementTest {

    private KeyedOneInputStreamOperatorTestHarness<String, CanonicalEvent, CIDAlert> harness;

    @BeforeEach
    void setUp() throws Exception {
        Module8_ComorbidityEngine engine = new Module8_ComorbidityEngine();
        harness = new KeyedOneInputStreamOperatorTestHarness<>(
            new KeyedProcessOperator<>(engine),
            CanonicalEvent::getPatientId,
            Types.STRING
        );
        harness.open();
    }

    @AfterEach
    void tearDown() throws Exception {
        if (harness != null) harness.close();
    }

    @Test
    @DisplayName("HALT alert emitted as both main output and side-output for triple whammy")
    void tripleWhammy_emitsHALTViaSideOutput() throws Exception {
        String patientId = "P-HARNESS-TW";
        long now = System.currentTimeMillis();

        // 1) Medication events to build state
        harness.processElement(buildMedEvent(patientId, "lisinopril", "ACEI", now - 5000), now - 5000);
        harness.processElement(buildMedEvent(patientId, "dapagliflozin", "SGLT2I", now - 4000), now - 4000);
        harness.processElement(buildMedEvent(patientId, "hydrochlorothiazide", "THIAZIDE", now - 3000), now - 3000);

        // 2) Weight drop >2kg in 7 days (precipitant for CID-01)
        harness.processElement(buildVitalEvent(patientId, 130.0, null, 80.0, now - 7L * 86_400_000L), now - 7L * 86_400_000L);
        harness.processElement(buildVitalEvent(patientId, 128.0, null, 77.0, now), now);

        // 3) Check main output contains CID-01
        List<CIDAlert> mainOutput = harness.extractOutputValues();
        assertTrue(mainOutput.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "CID-01 Triple Whammy should appear in main output");

        // 4) Check side-output contains same HALT alert
        var sideOutput = harness.getSideOutput(Module8_ComorbidityEngine.HALT_SAFETY_TAG);
        assertFalse(sideOutput.isEmpty(),
            "HALT side-output should have at least one record");
        assertTrue(sideOutput.stream().anyMatch(r -> "CID_01".equals(r.getValue().getRuleId())),
            "CID-01 should be routed to HALT safety-critical side-output");
    }

    @Test
    @DisplayName("Safe patient produces no alerts from processElement")
    void safePatient_noOutput() throws Exception {
        String patientId = "P-HARNESS-SAFE";
        long now = System.currentTimeMillis();

        // Single benign medication
        harness.processElement(buildMedEvent(patientId, "amlodipine", "CCB", now), now);

        // Normal vitals
        harness.processElement(buildVitalEvent(patientId, 125.0, null, null, now), now);

        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.isEmpty(), "Safe patient should produce zero alerts");
    }

    @Test
    @DisplayName("State accumulates across multiple events for same patient")
    void stateAccumulates_acrossEvents() throws Exception {
        String patientId = "P-HARNESS-ACCUM";
        long now = System.currentTimeMillis();

        // First event: SGLT2I alone — no CID-15 alert (needs NSAID too)
        harness.processElement(buildMedEvent(patientId, "empagliflozin", "SGLT2I", now - 2000), now - 2000);

        // Second event: NSAID — now SGLT2I + NSAID triggers CID-15 SOFT_FLAG
        harness.processElement(buildMedEvent(patientId, "ibuprofen", "NSAID", now), now);
        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "CID-15 should fire after NSAID added to existing SGLT2I");
    }

    // --- Test event builders using CORRECT payload field names ---

    private CanonicalEvent buildMedEvent(String patientId, String drugName, String drugClass, long time) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_name", drugName);
        payload.put("drug_class", drugClass);
        payload.put("dose_mg", 10.0);
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.MEDICATION_ORDERED)
            .eventTime(time)
            .payload(payload)
            .build();
    }

    private CanonicalEvent buildLabEvent(String patientId, String labType, double value, long time) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", labType);
        payload.put("value", value);
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.LAB_RESULT)
            .eventTime(time)
            .payload(payload)
            .build();
    }

    private CanonicalEvent buildVitalEvent(String patientId, Double sbp, Double dbp, Double weight, long time) {
        Map<String, Object> payload = new HashMap<>();
        if (sbp != null) payload.put("systolic_bp", sbp);
        if (dbp != null) payload.put("diastolic_bp", dbp);
        if (weight != null) payload.put("weight", weight);
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.VITAL_SIGN)
            .eventTime(time)
            .payload(payload)
            .build();
    }
}
