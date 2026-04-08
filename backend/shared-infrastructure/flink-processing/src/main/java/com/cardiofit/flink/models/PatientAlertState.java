package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Per-patient alert state maintained in Flink keyed state.
 * Tracks active alerts, dedup history, and fatigue metrics.
 */
public class PatientAlertState implements Serializable {
    private static final long serialVersionUID = 2L;

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("active_alerts") private Map<String, ClinicalAlert> activeAlerts = new HashMap<>();
    @JsonProperty("alerts_in_last_24h") private int alertsInLast24Hours;
    @JsonProperty("alert_window_start") private long alertWindowStart;
    /** Maps processing-time timer timestamp → alertId for O(1) lookup in onTimer. */
    @JsonProperty("timer_mappings") private Map<Long, String> timerMappings = new HashMap<>();

    public PatientAlertState() { this.alertWindowStart = System.currentTimeMillis(); }
    public PatientAlertState(String patientId) { this(); this.patientId = patientId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, ClinicalAlert> getActiveAlerts() { return activeAlerts; }
    public void setActiveAlerts(Map<String, ClinicalAlert> activeAlerts) { this.activeAlerts = activeAlerts; }
    public int getAlertsInLast24Hours() { return alertsInLast24Hours; }
    public void setAlertsInLast24Hours(int alertsInLast24Hours) { this.alertsInLast24Hours = alertsInLast24Hours; }
    public long getAlertWindowStart() { return alertWindowStart; }
    public void setAlertWindowStart(long alertWindowStart) { this.alertWindowStart = alertWindowStart; }
    public Map<Long, String> getTimerMappings() { return timerMappings; }

    /** Register a timer→alertId mapping for SLA escalation lookup. */
    public void registerTimerMapping(long timerTimestamp, String alertId) {
        timerMappings.put(timerTimestamp, alertId);
    }

    /** Pop and return the alertId for a fired timer, or null if not found. */
    public String popTimerMapping(long timerTimestamp) {
        return timerMappings.remove(timerTimestamp);
    }
}
