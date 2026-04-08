package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;

public class Module9TestBuilder {

    private static final long DAY_MS = 86_400_000L;
    private static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long daysAgo(int days) {
        return BASE_TIME - (days * DAY_MS);
    }

    public static long hoursAgo(int hours) {
        return BASE_TIME - (hours * 3600_000L);
    }

    // --- Event Builders ---

    public static CanonicalEvent medicationAdministered(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_ADMINISTERED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_name", "metformin");
        payload.put("drug_class", "BIGUANIDE");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent medicationMissed(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_MISSED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("drug_name", "metformin");
        payload.put("reason", "FORGOT");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent bpReading(String patientId, long timestamp, double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent mealLog(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "MEAL_LOG");
        payload.put("meal_type", "lunch");
        payload.put("carb_grams", 45.0);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent appSession(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "APP_SESSION");
        payload.put("session_duration_sec", 120);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent goalCompleted(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "GOAL_COMPLETED");
        payload.put("goal_type", "steps_10000");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent appointmentAttended(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.ENCOUNTER_START, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("encounter_type", "FOLLOW_UP");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent glucoseLabResult(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "glucose");
        payload.put("value", 110.0);
        payload.put("unit", "mg/dL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent cgmDeviceReading(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("data_tier", "TIER_1_CGM");
        payload.put("glucose_value", 105.0);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent weightReading(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("weight", 75.5);
        payload.put("unit", "kg");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent nonGlucoseLabResult(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "creatinine");
        payload.put("value", 1.2);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent symptomReport(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("symptom_type", "HYPOGLYCEMIA");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent withDataTier(CanonicalEvent event, String dataTier) {
        Map<String, Object> payload = event.getPayload();
        if (payload == null) payload = new HashMap<>();
        payload.put("data_tier", dataTier);
        event.setPayload(payload);
        return event;
    }

    // --- State Builders ---

    public static EngagementState stateWithChannel(String patientId, String channel) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        state.setChannel(channel);
        return state;
    }

    public static EngagementState fullyEngagedState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            java.util.Arrays.fill(bitmap, true);
            state.setSignalBitmap(signal, bitmap);
        }
        state.setLastUpdated(BASE_TIME);
        return state;
    }

    public static EngagementState decliningState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        for (SignalType signal : SignalType.values()) {
            boolean[] bitmap = new boolean[14];
            for (int i = 7; i < 14; i++) bitmap[i] = true;
            bitmap[1] = true;
            bitmap[4] = true;
            state.setSignalBitmap(signal, bitmap);
        }
        state.setLastUpdated(BASE_TIME);
        return state;
    }

    public static EngagementState disengagedState(String patientId) {
        EngagementState state = new EngagementState();
        state.setPatientId(patientId);
        for (SignalType signal : SignalType.values()) {
            state.setSignalBitmap(signal, new boolean[14]);
        }
        state.setLastUpdated(BASE_TIME);
        return state;
    }

    private static CanonicalEvent baseEvent(String patientId, EventType type, long timestamp) {
        CanonicalEvent event = new CanonicalEvent();
        event.setPatientId(patientId);
        event.setEventType(type);
        event.setEventTime(timestamp);
        event.setProcessingTime(System.currentTimeMillis());
        event.setCorrelationId("test-" + java.util.UUID.randomUUID());
        return event;
    }
}
