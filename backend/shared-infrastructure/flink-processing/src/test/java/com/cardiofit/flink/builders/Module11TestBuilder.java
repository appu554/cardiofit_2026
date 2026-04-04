package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

/**
 * Test builder for Module 11/11b tests.
 */
public class Module11TestBuilder {

    private static final long HOUR_MS = 3_600_000L;
    private static final long MIN_MS = 60_000L;

    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long minutesAfter(long base, int minutes) { return base + minutes * MIN_MS; }
    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }

    // --- Activity Events ---
    public static CanonicalEvent activityEvent(String patientId, long timestamp,
                                                String exerciseType, int durationMinutes) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "ACTIVITY_LOG");
        payload.put("exercise_type", exerciseType);
        payload.put("duration_minutes", durationMinutes);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent activityWithMETs(String patientId, long timestamp,
                                                   String exerciseType, int durationMinutes, double mets) {
        CanonicalEvent event = activityEvent(patientId, timestamp, exerciseType, durationMinutes);
        event.getPayload().put("mets", mets);
        return event;
    }

    // --- HR Events ---
    public static CanonicalEvent hrReading(String patientId, long timestamp, double heartRate) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("heart_rate", heartRate);
        payload.put("source", "WEARABLE");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent restingHRReading(String patientId, long timestamp, double heartRate) {
        CanonicalEvent event = hrReading(patientId, timestamp, heartRate);
        event.getPayload().put("activity_state", "RESTING");
        return event;
    }

    // --- Glucose Events ---
    public static CanonicalEvent cgmReading(String patientId, long timestamp, double glucose) {
        CanonicalEvent event = baseEvent(patientId, EventType.DEVICE_READING, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("glucose_value", glucose);
        payload.put("source", "CGM");
        event.setPayload(payload);
        return event;
    }

    // --- BP Events ---
    public static CanonicalEvent bpReading(String patientId, long timestamp,
                                            double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    // --- State Builders ---
    public static ActivityCorrelationState emptyState(String patientId) {
        return new ActivityCorrelationState(patientId);
    }

    public static ActivityCorrelationState stateWithAge(String patientId, int age) {
        ActivityCorrelationState state = new ActivityCorrelationState(patientId);
        state.setPatientAge(age);
        return state;
    }

    // --- ActivityResponseRecord Builder ---
    public static ActivityResponseRecord activityRecord(String patientId, long timestamp,
                                                         ExerciseType type, double mets, double durationMin,
                                                         Double peakHR, Double hrr1, Double glucoseDelta) {
        return ActivityResponseRecord.builder()
                .recordId("test-" + UUID.randomUUID())
                .patientId(patientId)
                .activityEventId("act-" + UUID.randomUUID())
                .activityStartTime(timestamp)
                .exerciseType(type)
                .reportedMETs(mets)
                .activityDurationMin(durationMin)
                .metMinutes(mets * durationMin)
                .peakHR(peakHR)
                .hrr1(hrr1)
                .hrRecoveryClass(hrr1 != null ? HRRecoveryClass.fromHRR1(hrr1) : HRRecoveryClass.INSUFFICIENT_DATA)
                .exerciseGlucoseDelta(glucoseDelta)
                .qualityScore(0.7)
                .build();
    }

    private static CanonicalEvent baseEvent(String patientId, EventType type, long timestamp) {
        CanonicalEvent event = new CanonicalEvent();
        event.setId(UUID.randomUUID().toString());
        event.setPatientId(patientId);
        event.setEventType(type);
        event.setEventTime(timestamp);
        event.setProcessingTime(System.currentTimeMillis());
        event.setCorrelationId("test-" + UUID.randomUUID());
        return event;
    }
}
