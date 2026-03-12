package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Clinical Snapshot - Point-in-time patient clinical state
 *
 * Captures vital signs, clinical scores, and active alerts at a specific
 * point in time for trajectory analysis and clinical decision support.
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class ClinicalSnapshot implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("vital_signs")
    private Map<String, Double> vitalSigns;

    @JsonProperty("lab_values")
    private Map<String, Double> labValues;

    @JsonProperty("news2_score")
    private Double news2Score;

    @JsonProperty("qsofa_score")
    private Integer qsofaScore;

    @JsonProperty("apache_score")
    private Double apacheScore;

    @JsonProperty("active_alerts")
    private List<String> activeAlerts;

    @JsonProperty("acuity_level")
    private String acuityLevel;

    @JsonProperty("clinical_status")
    private String clinicalStatus;  // STABLE, IMPROVING, DETERIORATING, CRITICAL

    // Default constructor
    public ClinicalSnapshot() {
        this.timestamp = System.currentTimeMillis();
    }

    // Constructor with timestamp
    public ClinicalSnapshot(long timestamp) {
        this.timestamp = timestamp;
    }

    // Getters and Setters
    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    public Map<String, Double> getVitalSigns() { return vitalSigns; }
    public void setVitalSigns(Map<String, Double> vitalSigns) { this.vitalSigns = vitalSigns; }

    public Map<String, Double> getLabValues() { return labValues; }
    public void setLabValues(Map<String, Double> labValues) { this.labValues = labValues; }

    public Double getNews2Score() { return news2Score; }
    public void setNews2Score(Double news2Score) { this.news2Score = news2Score; }

    public Integer getQsofaScore() { return qsofaScore; }
    public void setQsofaScore(Integer qsofaScore) { this.qsofaScore = qsofaScore; }

    public Double getApacheScore() { return apacheScore; }
    public void setApacheScore(Double apacheScore) { this.apacheScore = apacheScore; }

    public List<String> getActiveAlerts() { return activeAlerts; }
    public void setActiveAlerts(List<String> activeAlerts) { this.activeAlerts = activeAlerts; }

    public String getAcuityLevel() { return acuityLevel; }
    public void setAcuityLevel(String acuityLevel) { this.acuityLevel = acuityLevel; }

    public String getClinicalStatus() { return clinicalStatus; }
    public void setClinicalStatus(String clinicalStatus) { this.clinicalStatus = clinicalStatus; }

    // Utility methods

    /**
     * Check if patient is deteriorating based on clinical status
     */
    public boolean isDeteriorating() {
        return "DETERIORATING".equals(clinicalStatus) || "CRITICAL".equals(clinicalStatus);
    }

    /**
     * Get specific vital sign value
     */
    public Double getVitalSign(String vitalName) {
        return vitalSigns != null ? vitalSigns.get(vitalName) : null;
    }

    /**
     * Get specific lab value
     */
    public Double getLabValue(String labName) {
        return labValues != null ? labValues.get(labName) : null;
    }

    @Override
    public String toString() {
        return "ClinicalSnapshot{" +
            "timestamp=" + timestamp +
            ", news2Score=" + news2Score +
            ", qsofaScore=" + qsofaScore +
            ", acuityLevel='" + acuityLevel + '\'' +
            ", clinicalStatus='" + clinicalStatus + '\'' +
            '}';
    }
}
