package com.cardiofit.stream.models;

import java.io.Serializable;
import java.time.LocalDateTime;
import java.util.Map;

/**
 * VitalSigns represents vital signs data from patient monitoring
 */
public class VitalSigns implements Serializable {
    private static final long serialVersionUID = 1L;

    private String vitalSignsId;
    private String patientId;
    private LocalDateTime timestamp;
    private Double heartRate;
    private Double systolicBP;
    private Double diastolicBP;
    private Double temperature;
    private Double respiratoryRate;
    private Double oxygenSaturation;
    private Map<String, Object> additionalVitals;
    private String deviceId;
    private boolean isAbnormal;

    public VitalSigns() {}

    public VitalSigns(String vitalSignsId, String patientId, LocalDateTime timestamp) {
        this.vitalSignsId = vitalSignsId;
        this.patientId = patientId;
        this.timestamp = timestamp;
    }

    // Getters and Setters
    public String getVitalSignsId() { return vitalSignsId; }
    public void setVitalSignsId(String vitalSignsId) { this.vitalSignsId = vitalSignsId; }

    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public LocalDateTime getTimestamp() { return timestamp; }
    public void setTimestamp(LocalDateTime timestamp) { this.timestamp = timestamp; }

    public Double getHeartRate() { return heartRate; }
    public void setHeartRate(Double heartRate) { this.heartRate = heartRate; }

    public Double getSystolicBP() { return systolicBP; }
    public void setSystolicBP(Double systolicBP) { this.systolicBP = systolicBP; }

    public Double getDiastolicBP() { return diastolicBP; }
    public void setDiastolicBP(Double diastolicBP) { this.diastolicBP = diastolicBP; }

    public Double getTemperature() { return temperature; }
    public void setTemperature(Double temperature) { this.temperature = temperature; }

    public Double getRespiratoryRate() { return respiratoryRate; }
    public void setRespiratoryRate(Double respiratoryRate) { this.respiratoryRate = respiratoryRate; }

    public Double getOxygenSaturation() { return oxygenSaturation; }
    public void setOxygenSaturation(Double oxygenSaturation) { this.oxygenSaturation = oxygenSaturation; }

    public Map<String, Object> getAdditionalVitals() { return additionalVitals; }
    public void setAdditionalVitals(Map<String, Object> additionalVitals) {
        this.additionalVitals = additionalVitals;
    }

    public String getDeviceId() { return deviceId; }
    public void setDeviceId(String deviceId) { this.deviceId = deviceId; }

    public boolean isAbnormal() { return isAbnormal; }
    public void setAbnormal(boolean abnormal) { isAbnormal = abnormal; }
}