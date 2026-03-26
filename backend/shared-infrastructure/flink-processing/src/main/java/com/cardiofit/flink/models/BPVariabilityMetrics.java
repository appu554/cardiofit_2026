package com.cardiofit.flink.models;

import java.time.Instant;

public class BPVariabilityMetrics {
    private String patientId;
    private Double arvSbp7d;
    private Double arvSbp30d;
    private Double arvCuffless;
    private Double morningSurgeToday;
    private Double morningSurge7dAvg;
    private Double surgePrewaking;
    private Double dipRatio;
    private String dipClassification;     // DIPPER/NON_DIPPER/EXTREME/REVERSE
    private String dipConfidence;         // HIGH/LOW
    private Double sbp7dAvg;
    private Double dbp7dAvg;
    private String bpControlStatus;       // CONTROLLED/ELEVATED/STAGE1/STAGE2
    private Instant computedAt;

    public BPVariabilityMetrics() {}
    public String getPatientId() { return patientId; }
    public void setPatientId(String p) { this.patientId = p; }
    public Double getArvSbp7d() { return arvSbp7d; }
    public void setArvSbp7d(Double v) { this.arvSbp7d = v; }
    public Double getArvSbp30d() { return arvSbp30d; }
    public void setArvSbp30d(Double v) { this.arvSbp30d = v; }
    public String getDipClassification() { return dipClassification; }
    public void setDipClassification(String v) { this.dipClassification = v; }
    public String getBpControlStatus() { return bpControlStatus; }
    public void setBpControlStatus(String v) { this.bpControlStatus = v; }
    public Instant getComputedAt() { return computedAt; }
    public void setComputedAt(Instant v) { this.computedAt = v; }
    public Double getArvCuffless() { return arvCuffless; }
    public void setArvCuffless(Double v) { this.arvCuffless = v; }
    public Double getMorningSurgeToday() { return morningSurgeToday; }
    public void setMorningSurgeToday(Double v) { this.morningSurgeToday = v; }
    public Double getMorningSurge7dAvg() { return morningSurge7dAvg; }
    public void setMorningSurge7dAvg(Double v) { this.morningSurge7dAvg = v; }
    public Double getSurgePrewaking() { return surgePrewaking; }
    public void setSurgePrewaking(Double v) { this.surgePrewaking = v; }
    public Double getDipRatio() { return dipRatio; }
    public void setDipRatio(Double v) { this.dipRatio = v; }
    public String getDipConfidence() { return dipConfidence; }
    public void setDipConfidence(String v) { this.dipConfidence = v; }
    public Double getSbp7dAvg() { return sbp7dAvg; }
    public void setSbp7dAvg(Double v) { this.sbp7dAvg = v; }
    public Double getDbp7dAvg() { return dbp7dAvg; }
    public void setDbp7dAvg(Double v) { this.dbp7dAvg = v; }
}
