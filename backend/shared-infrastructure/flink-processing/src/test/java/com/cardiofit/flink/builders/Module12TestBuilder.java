package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;

public class Module12TestBuilder {

    public static final long HOUR_MS = 3_600_000L;
    public static final long MIN_MS = 60_000L;
    public static final long DAY_MS = 86_400_000L;
    public static final long BASE_TIME = 1743552000000L; // 2025-04-02 00:00:00 UTC

    public static long minutesAfter(long base, int minutes) { return base + minutes * MIN_MS; }
    public static long hoursAfter(long base, int hours) { return base + hours * HOUR_MS; }
    public static long daysAfter(long base, int days) { return base + days * DAY_MS; }

    public static CanonicalEvent interventionApproved(String patientId, long timestamp,
                                                       String interventionId,
                                                       InterventionType interventionType,
                                                       Map<String, Object> detail) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_APPROVED");
        payload.put("intervention_id", interventionId);
        payload.put("intervention_type", interventionType.name());
        payload.put("intervention_detail", detail != null ? detail : new HashMap<>());
        payload.put("originating_card_id", "card-" + UUID.randomUUID());
        payload.put("physician_action", "APPROVED");
        payload.put("physician_id", "dr-" + UUID.randomUUID());
        payload.put("observation_window_days", interventionType.getDefaultWindowDays());
        payload.put("data_tier", "TIER_2_HYBRID");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent interventionApproved(String patientId, long timestamp,
                                                       String interventionId,
                                                       InterventionType interventionType) {
        return interventionApproved(patientId, timestamp, interventionId, interventionType, null);
    }

    public static CanonicalEvent interventionCancelled(String patientId, long timestamp,
                                                        String interventionId) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_CANCELLED");
        payload.put("intervention_id", interventionId);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent interventionModified(String patientId, long timestamp,
                                                       String interventionId,
                                                       int newWindowDays) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "INTERVENTION_MODIFIED");
        payload.put("intervention_id", interventionId);
        payload.put("observation_window_days", newWindowDays);
        payload.put("modification_detail", Collections.singletonMap("window_days_changed", true));
        event.setPayload(payload);
        return event;
    }

    public static Map<String, Object> medicationDetail(String drugClass, String drugName, String dose) {
        Map<String, Object> detail = new HashMap<>();
        detail.put("drug_class", drugClass);
        detail.put("drug_name", drugName);
        detail.put("dose", dose);
        return detail;
    }

    public static Map<String, Object> lifestyleDetail(String targetDomain, String description) {
        Map<String, Object> detail = new HashMap<>();
        detail.put("target_domain", targetDomain);
        detail.put("description", description);
        return detail;
    }

    public static CanonicalEvent fbgReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "FBG");
        payload.put("value", value);
        payload.put("unit", "mg/dL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent sbpReading(String patientId, long timestamp,
                                             double sbp, double dbp) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("systolic_bp", sbp);
        payload.put("diastolic_bp", dbp);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent egfrReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.LAB_RESULT, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("lab_type", "EGFR");
        payload.put("value", value);
        payload.put("unit", "mL/min/1.73m2");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent weightReading(String patientId, long timestamp, double value) {
        CanonicalEvent event = baseEvent(patientId, EventType.VITAL_SIGN, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("weight_kg", value);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent externalMedicationEvent(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.MEDICATION_ORDERED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("source_system", "EXTERNAL_HOSPITAL");
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent hospitalisationEvent(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("report_type", "ADMISSION");
        payload.put("admission_flag", true);
        event.setPayload(payload);
        return event;
    }

    public static CanonicalEvent patientReportedIllness(String patientId, long timestamp) {
        CanonicalEvent event = baseEvent(patientId, EventType.PATIENT_REPORTED, timestamp);
        Map<String, Object> payload = new HashMap<>();
        payload.put("event_type", "ILLNESS");
        event.setPayload(payload);
        return event;
    }

    public static InterventionWindowState emptyState(String patientId) {
        return new InterventionWindowState(patientId);
    }

    public static InterventionWindowState stateWithBaselines(String patientId,
                                                              double fbg, double sbp, double weight) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        state.setLastKnownFBG(fbg);
        state.setLastKnownSBP(sbp);
        state.setLastKnownWeight(weight);
        return state;
    }

    public static InterventionWindowState stateWithFBGReadings(String patientId,
                                                                double[] values,
                                                                long endTimestamp) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        for (int i = 0; i < values.length; i++) {
            long ts = endTimestamp - (long)(values.length - 1 - i) * 2 * DAY_MS;
            state.addReading("FBG", values[i], ts);
        }
        return state;
    }

    public static InterventionWindowState stateWithSBPReadings(String patientId,
                                                                double[] values,
                                                                long endTimestamp) {
        InterventionWindowState state = new InterventionWindowState(patientId);
        for (int i = 0; i < values.length; i++) {
            long ts = endTimestamp - (long)(values.length - 1 - i) * 2 * DAY_MS;
            state.addReading("SBP", values[i], ts);
        }
        return state;
    }

    public static List<InterventionWindowState.TrajectoryDataPoint> fbgReadings(
            double[] values, long startTime, long intervalMs) {
        List<InterventionWindowState.TrajectoryDataPoint> points = new ArrayList<>();
        for (int i = 0; i < values.length; i++) {
            points.add(new InterventionWindowState.TrajectoryDataPoint(
                    "FBG", values[i], startTime + i * intervalMs));
        }
        return points;
    }

    public static List<InterventionWindowState.TrajectoryDataPoint> sbpReadings(
            double[] values, long startTime, long intervalMs) {
        List<InterventionWindowState.TrajectoryDataPoint> points = new ArrayList<>();
        for (int i = 0; i < values.length; i++) {
            points.add(new InterventionWindowState.TrajectoryDataPoint(
                    "SBP", values[i], startTime + i * intervalMs));
        }
        return points;
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
