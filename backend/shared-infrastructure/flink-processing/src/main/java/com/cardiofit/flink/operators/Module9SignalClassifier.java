package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.CanonicalEvent;
import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.SignalType;
import java.util.Map;

/**
 * Classifies a CanonicalEvent into one of the 8 engagement signal channels.
 * Returns null if the event is irrelevant to engagement tracking.
 *
 * This is the HIGHEST-RISK piece of Module 9 — field names depend on the
 * actual CanonicalEvent payload schema from Module 1b.
 *
 * R1 Fix: Glucose monitoring (S1) is restored as the highest-weighted signal
 * (tied with BP at 0.20). Sourced from LAB_RESULT with glucose/fbg/cgm lab types
 * and from DEVICE_READING for CGM devices.
 */
public final class Module9SignalClassifier {

    private Module9SignalClassifier() {}

    /**
     * Classify a CanonicalEvent into a SignalType.
     * @return SignalType if the event maps to an engagement signal, null otherwise
     */
    public static SignalType classify(CanonicalEvent event) {
        if (event == null || event.getEventType() == null) return null;

        EventType type = event.getEventType();
        Map<String, Object> payload = event.getPayload();

        switch (type) {
            case MEDICATION_ADMINISTERED:
                return SignalType.MEDICATION_ADHERENCE;

            case LAB_RESULT:
                return classifyLabResult(payload);

            case VITAL_SIGN:
            case VITAL_SIGNS:
                return classifyVitalSign(payload);

            case PATIENT_REPORTED:
                return classifyPatientReported(payload);

            case ENCOUNTER_START:
            case ENCOUNTER_UPDATE:
                return SignalType.APPOINTMENT_ATTENDANCE;

            case TASK_COMPLETED:
                if (payload != null) {
                    String taskType = getStringField(payload, "task_type");
                    if ("APPOINTMENT".equalsIgnoreCase(taskType)
                        || "CHECKIN".equalsIgnoreCase(taskType)) {
                        return SignalType.APPOINTMENT_ATTENDANCE;
                    }
                }
                return null;

            case DEVICE_READING:
                return classifyDeviceReading(payload);

            default:
                return null;
        }
    }

    /**
     * Classify LAB_RESULT events.
     * Glucose/FBG/CGM lab types -> GLUCOSE_MONITORING (S1).
     * Other lab types (creatinine, potassium, etc.) are NOT patient-driven
     * engagement — they're ordered by physicians.
     */
    private static SignalType classifyLabResult(Map<String, Object> payload) {
        if (payload == null) return null;
        String labType = getStringField(payload, "lab_type");
        if (labType == null) labType = getStringField(payload, "labName");
        if (labType == null) return null;

        String upper = labType.toUpperCase();
        if (upper.contains("GLUCOSE") || upper.contains("FBG") || upper.contains("PPBG")
            || upper.contains("CGM") || upper.contains("HBA1C")
            || upper.contains("BLOOD_SUGAR") || upper.contains("SMBG")) {
            return SignalType.GLUCOSE_MONITORING;
        }
        return null;
    }

    /**
     * Classify VITAL_SIGN events into BP (S2) or Weight (S6).
     * BP takes priority if both are present.
     */
    private static SignalType classifyVitalSign(Map<String, Object> payload) {
        if (payload == null) return null;
        if (payload.containsKey("systolic_bp")) {
            return SignalType.BP_MEASUREMENT;
        }
        if (payload.containsKey("weight")) {
            return SignalType.WEIGHT_MEASUREMENT;
        }
        return null;
    }

    /**
     * Classify DEVICE_READING events.
     * CGM data -> GLUCOSE_MONITORING (S1).
     * Home BP monitor -> BP_MEASUREMENT (S2).
     */
    private static SignalType classifyDeviceReading(Map<String, Object> payload) {
        if (payload == null) return null;
        String dataTier = getStringField(payload, "data_tier");
        if ("TIER_1_CGM".equals(dataTier)) {
            return SignalType.GLUCOSE_MONITORING;
        }
        Boolean cgmActive = payload.get("cgm_active") instanceof Boolean
            ? (Boolean) payload.get("cgm_active") : null;
        if (Boolean.TRUE.equals(cgmActive)) {
            return SignalType.GLUCOSE_MONITORING;
        }
        if (payload.containsKey("systolic_bp")) {
            return SignalType.BP_MEASUREMENT;
        }
        return null;
    }

    /**
     * Sub-classify PATIENT_REPORTED events by report_type field.
     */
    private static SignalType classifyPatientReported(Map<String, Object> payload) {
        if (payload == null) return null;

        String reportType = getStringField(payload, "report_type");
        if (reportType == null) {
            String symptomType = getStringField(payload, "symptom_type");
            if (symptomType != null) return null;
            return null;
        }

        switch (reportType.toUpperCase()) {
            case "MEAL_LOG":
            case "FOOD_LOG":
            case "DIETARY_LOG":
                return SignalType.MEAL_LOGGING;

            case "APP_SESSION":
            case "APP_INTERACTION":
                return SignalType.APP_SESSION;

            case "GOAL_COMPLETED":
            case "GOAL_ACHIEVED":
            case "TARGET_MET":
                return SignalType.GOAL_COMPLETION;

            default:
                return null;
        }
    }

    private static String getStringField(Map<String, Object> payload, String key) {
        Object val = payload.get(key);
        return (val instanceof String) ? (String) val : null;
    }
}
