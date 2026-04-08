package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;
import java.util.HashMap;
import java.util.Map;

/**
 * Static analyzer: accumulates confounder flags from events that arrive
 * during active observation windows. Called by the main KPF whenever a
 * non-intervention event arrives for a patient with active windows.
 *
 * Event routing rules per spec Section 5.3:
 * - MEDICATION_ORDERED (external) → EXTERNAL_MEDICATION_CHANGE
 * - PATIENT_REPORTED + admission_flag → HOSPITALISATION
 * - PATIENT_REPORTED + event_type=ILLNESS → INTERCURRENT_ILLNESS
 * - PATIENT_REPORTED + event_type=TRAVEL → TRAVEL_DISRUPTION
 * - LAB_RESULT → accumulated to lab_changes list (not a confounder flag)
 * - VITAL_SIGN / DEVICE_READING → trajectory tracker (not a confounder)
 */
public final class Module12ConfounderAccumulator {

    private Module12ConfounderAccumulator() {}

    /**
     * Evaluates an event against an active observation window and accumulates
     * any detected confounders or lab changes.
     */
    public static void accumulate(InterventionWindowState.InterventionWindow window,
                                   CanonicalEvent event) {
        if (window == null || !"OBSERVING".equals(window.status)) return;
        if (event == null || event.getPayload() == null) return;

        EventType eventType = event.getEventType();
        Map<String, Object> payload = event.getPayload();

        if (eventType == EventType.MEDICATION_ORDERED) {
            String source = getStr(payload, "source_system");
            if (source != null && !source.isEmpty()) {
                addConfounder(window, "EXTERNAL_MEDICATION_CHANGE");
            }
        } else if (eventType == EventType.PATIENT_REPORTED) {
            Boolean admissionFlag = getBool(payload, "admission_flag");
            if (Boolean.TRUE.equals(admissionFlag)) {
                addConfounder(window, "HOSPITALISATION");
                return;
            }

            String subEventType = getStr(payload, "event_type");
            if ("ILLNESS".equals(subEventType)) {
                addConfounder(window, "INTERCURRENT_ILLNESS");
            } else if ("TRAVEL".equals(subEventType)) {
                addConfounder(window, "TRAVEL_DISRUPTION");
            }
        } else if (eventType == EventType.LAB_RESULT) {
            Map<String, Object> labEntry = new HashMap<>();
            labEntry.put("lab_type", getStr(payload, "lab_type"));
            labEntry.put("value", payload.get("value"));
            labEntry.put("timestamp", event.getEventTime());
            window.labChanges.add(labEntry);
        }
        // VITAL_SIGN and DEVICE_READING are handled by trajectory tracker, not here
    }

    /**
     * Adds a festival/seasonal confounder flag (from KB-21 cultural calendar lookup).
     */
    public static void addFestivalConfounder(InterventionWindowState.InterventionWindow window,
                                              String festivalName) {
        if (window == null) return;
        addConfounder(window, "FESTIVAL_PERIOD:" + festivalName);
    }

    private static void addConfounder(InterventionWindowState.InterventionWindow window,
                                       String flag) {
        if (!window.confoundersDetected.contains(flag)) {
            window.confoundersDetected.add(flag);
        }
    }

    private static String getStr(Map<String, Object> m, String key) {
        Object v = m.get(key);
        return v != null ? v.toString() : null;
    }

    private static Boolean getBool(Map<String, Object> m, String key) {
        Object v = m.get(key);
        if (v instanceof Boolean) return (Boolean) v;
        return null;
    }
}
