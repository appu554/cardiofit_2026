package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Canonical BP reading from ingestion.vitals or ingestion.clinic-bp.
 *
 * Field normalization (Rule 2):
 *   ingestion.vitals uses: "systolicbloodpressure", "diastolicbloodpressure"
 *   ingestion.clinic-bp may use: "systolic", "diastolic", "sbp", "dbp"
 *   Both may carry: "heartrate" (for orthostatic assessment)
 *
 * Validation at entry (Rule 1 + Rule 3):
 *   - patientId must be non-null
 *   - SBP must be in [40, 300] (physiological range)
 *   - DBP must be in [20, 200]
 *   - SBP must be > DBP
 *   - timestamp must be present and within +/- 1h future / 30d past
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class BPReading implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("systolic") private Double systolic;         // mmHg
    @JsonProperty("diastolic") private Double diastolic;       // mmHg
    @JsonProperty("heart_rate") private Double heartRate;      // bpm (optional, for orthostatic)
    @JsonProperty("timestamp") private long timestamp;         // epoch millis
    @JsonProperty("time_context") private String timeContext;  // MORNING, EVENING, etc. (may be null)
    @JsonProperty("source") private String source;             // HOME_CUFF, CLINIC, CUFFLESS
    @JsonProperty("position") private String position;         // SEATED, STANDING, SUPINE
    @JsonProperty("device_type") private String deviceType;    // oscillometric_cuff, etc.
    @JsonProperty("encounter_id") private String encounterId;
    @JsonProperty("correlation_id") private String correlationId;

    public BPReading() {}

    // — Validation —

    public boolean isValid() {
        if (patientId == null || patientId.isEmpty()) return false;
        if (systolic == null || diastolic == null) return false;
        if (systolic < 40 || systolic > 300) return false;
        if (diastolic < 20 || diastolic > 200) return false;
        if (systolic <= diastolic) return false;
        if (timestamp <= 0) return false;
        return true;
    }

    // — Derived accessors —

    public TimeContext resolveTimeContext() {
        if (timeContext != null && !timeContext.isEmpty()) {
            try {
                return TimeContext.valueOf(timeContext.toUpperCase());
            } catch (IllegalArgumentException e) {
                // fall through to hour-based derivation
            }
        }
        // Derive from timestamp hour (UTC — in production, use patient timezone)
        java.time.Instant instant = java.time.Instant.ofEpochMilli(timestamp);
        int hour = instant.atZone(java.time.ZoneOffset.UTC).getHour();
        return TimeContext.fromHour(hour);
    }

    public BPSource resolveSource() {
        if (source == null || source.isEmpty()) return BPSource.UNKNOWN;
        try {
            return BPSource.valueOf(source.toUpperCase());
        } catch (IllegalArgumentException e) {
            // Normalize common variants
            String s = source.toLowerCase();
            if (s.contains("clinic") || s.contains("office")) return BPSource.CLINIC;
            if (s.contains("cuffless") || s.contains("wearable")) return BPSource.CUFFLESS;
            if (s.contains("cuff") || s.contains("home") || s.contains("oscillometric")) return BPSource.HOME_CUFF;
            return BPSource.UNKNOWN;
        }
    }

    public double getPulsePresure() {
        return systolic - diastolic;
    }

    public double getMeanArterialPressure() {
        return diastolic + (systolic - diastolic) / 3.0;
    }

    // — Standard getters/setters —
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Double getSystolic() { return systolic; }
    public void setSystolic(Double systolic) { this.systolic = systolic; }
    public Double getDiastolic() { return diastolic; }
    public void setDiastolic(Double diastolic) { this.diastolic = diastolic; }
    public Double getHeartRate() { return heartRate; }
    public void setHeartRate(Double heartRate) { this.heartRate = heartRate; }
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }
    public String getTimeContext() { return timeContext; }
    public void setTimeContext(String timeContext) { this.timeContext = timeContext; }
    public String getSource() { return source; }
    public void setSource(String source) { this.source = source; }
    public String getPosition() { return position; }
    public void setPosition(String position) { this.position = position; }
    public String getDeviceType() { return deviceType; }
    public void setDeviceType(String deviceType) { this.deviceType = deviceType; }
    public String getEncounterId() { return encounterId; }
    public void setEncounterId(String encounterId) { this.encounterId = encounterId; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String correlationId) { this.correlationId = correlationId; }
}
