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

    @Test
    @DisplayName("CID-06 fires when thiazide + FBG delta >15 mg/dL over 14 days via processElement")
    void thiazideFBGDelta_firesCID06() throws Exception {
        String patientId = "P-HARNESS-FBG";
        long now = System.currentTimeMillis();

        // 1) Thiazide medication
        harness.processElement(buildMedEvent(patientId, "hydrochlorothiazide", "THIAZIDE", now - 14L * 86_400_000L), now - 14L * 86_400_000L);

        // 2) FBG reading 14 days ago: 110 mg/dL
        harness.processElement(buildLabEvent(patientId, "fbg", 110.0, now - 14L * 86_400_000L), now - 14L * 86_400_000L);

        // 3) FBG reading now: 130 mg/dL (delta = +20, exceeds 15 threshold)
        harness.processElement(buildLabEvent(patientId, "fbg", 130.0, now), now);

        // 4) Verify CID-06 fires through the full operator pipeline
        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())),
            "CID-06 should fire when FBG increases >15 mg/dL over 14 days on thiazide");

        // 5) Verify it's PAUSE severity (not routed to HALT side-output)
        // getSideOutput() returns null (not empty list) when no HALT alerts exist
        var sideOutput = harness.getSideOutput(Module8_ComorbidityEngine.HALT_SAFETY_TAG);
        assertTrue(sideOutput == null || sideOutput.stream().noneMatch(r -> "CID_06".equals(r.getValue().getRuleId())),
            "CID-06 is PAUSE severity — should NOT appear in HALT side-output");
    }

    @Test
    @DisplayName("Keto diet flag persists past 72h — dietary choices don't auto-resolve")
    void ketoDietPersists_CID04StillFires() throws Exception {
        String patientId = "P-HARNESS-KETO";
        long now = System.currentTimeMillis();

        // 1) SGLT2i medication
        harness.processElement(buildMedEvent(patientId, "dapagliflozin", "SGLT2I", now - 80L * 3600_000L), now - 80L * 3600_000L);

        // 2) Keto diet reported 80 hours ago (>72h ago, but keto has no TTL)
        harness.processElement(buildSymptomEvent(patientId, "KETO_DIET", null, now - 80L * 3600_000L), now - 80L * 3600_000L);

        // 3) Benign vital event NOW — keto should NOT expire (dietary choice is sticky)
        harness.processElement(buildVitalEvent(patientId, 120.0, null, null, now), now);

        // 4) CID-04 should fire — keto diet persists until explicit RESOLVED event
        List<CIDAlert> output = harness.extractOutputValues();
        assertTrue(output.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "CID-04 should fire — keto diet flag has no TTL (dietary choices don't auto-resolve)");
    }

    @Test
    @DisplayName("Keto diet resolves only via explicit RESOLVED event")
    void ketoDietResolved_CID04Stops() throws Exception {
        String patientId = "P-HARNESS-KETO-RES";
        long now = System.currentTimeMillis();

        // 1) SGLT2i + keto diet
        harness.processElement(buildMedEvent(patientId, "dapagliflozin", "SGLT2I", now - 5000), now - 5000);
        harness.processElement(buildSymptomEvent(patientId, "KETO_DIET", null, now - 4000), now - 4000);

        // 2) Verify CID-04 fires
        List<CIDAlert> output1 = harness.extractOutputValues();
        assertTrue(output1.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "CID-04 should fire when SGLT2i + keto diet active");

        // 3) Patient reports stopping keto diet
        harness.processElement(buildSymptomEvent(patientId, "KETO_DIET", "RESOLVED", now), now);

        // 4) CID-04 should NOT appear in new output
        // extractOutputValues() is non-destructive — use size delta to check new entries
        List<CIDAlert> output2 = harness.extractOutputValues();
        List<CIDAlert> newAlerts = output2.subList(output1.size(), output2.size());
        assertFalse(newAlerts.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "CID-04 should stop firing after keto diet explicitly resolved");
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

    private CanonicalEvent buildSymptomEvent(String patientId, String symptomType, String status, long time) {
        Map<String, Object> payload = new HashMap<>();
        payload.put("symptom_type", symptomType);
        if (status != null) payload.put("status", status);
        return CanonicalEvent.builder()
            .id("evt-" + System.nanoTime())
            .patientId(patientId)
            .eventType(EventType.PATIENT_REPORTED)
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
