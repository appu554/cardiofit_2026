package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.time.Instant;

/**
 * Complete output model for Module 7 BP Variability Engine.
 *
 * Emitted on every processElement() invocation. Downstream consumers
 * (Module 8 comorbidity interaction, decision cards, V-MCU) read typed
 * enum fields rather than raw strings.
 *
 * Field groups:
 *   1. Identity: patientId, correlationId, computedAt
 *   2. Trigger: the reading that triggered this computation
 *   3. 7-day metrics: arvSbp7d, sdSbp7d, cvSbp7d, variabilityClassification7d
 *   4. 30-day metrics: arvSbp30d, sdSbp30d, cvSbp30d, variabilityClassification30d
 *   5. Morning surge: morningSurgeToday, morningSurge7dAvg, surgeClassification
 *   6. Dipping: dipRatio, dipClassification
 *   7. BP control: sbp7dAvg, dbp7dAvg, bpControlStatus
 *   8. White-coat / masked HTN: whiteCoatSuspected, maskedHTNSuspected,
 *      clinicHomeGapSBP
 *   9. Crisis: crisisFlag
 *  10. Data quality: contextDepth, totalReadingsInState, daysWithDataIn7d, daysWithDataIn30d
 */
public class BPVariabilityMetrics implements Serializable {
    private static final long serialVersionUID = 1L;

    // — 1. Identity —
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("correlation_id") private String correlationId;
    @JsonProperty("computed_at") private long computedAt; // epoch millis

    // — 2. Trigger context —
    @JsonProperty("trigger_sbp") private double triggerSBP;
    @JsonProperty("trigger_dbp") private double triggerDBP;
    @JsonProperty("trigger_source") private String triggerSource;       // BPSource name
    @JsonProperty("trigger_time_context") private String triggerTimeContext; // TimeContext name
    @JsonProperty("trigger_timestamp") private long triggerTimestamp;

    // — 3. 7-day variability metrics —
    @JsonProperty("arv_sbp_7d") private Double arvSbp7d;
    @JsonProperty("sd_sbp_7d") private Double sdSbp7d;
    @JsonProperty("cv_sbp_7d") private Double cvSbp7d;
    @JsonProperty("variability_classification_7d") private String variabilityClassification7d;

    // — 4. 30-day variability metrics —
    @JsonProperty("arv_sbp_30d") private Double arvSbp30d;
    @JsonProperty("sd_sbp_30d") private Double sdSbp30d;
    @JsonProperty("cv_sbp_30d") private Double cvSbp30d;
    @JsonProperty("variability_classification_30d") private String variabilityClassification30d;

    // — 5. Morning surge —
    @JsonProperty("morning_surge_today") private Double morningSurgeToday;
    @JsonProperty("morning_surge_7d_avg") private Double morningSurge7dAvg;
    @JsonProperty("surge_classification") private String surgeClassification;

    // — 6. Dipping pattern —
    @JsonProperty("dip_ratio") private Double dipRatio;
    @JsonProperty("dip_classification") private String dipClassification;

    // — 7. BP control —
    @JsonProperty("sbp_7d_avg") private Double sbp7dAvg;
    @JsonProperty("dbp_7d_avg") private Double dbp7dAvg;
    @JsonProperty("bp_control_status") private String bpControlStatus;

    // — 8. White-coat / masked HTN —
    @JsonProperty("white_coat_suspected") private boolean whiteCoatSuspected;
    @JsonProperty("masked_htn_suspected") private boolean maskedHTNSuspected;
    @JsonProperty("clinic_home_gap_sbp") private Double clinicHomeGapSBP;

    // — 9. Crisis —
    @JsonProperty("crisis_flag") private boolean crisisFlag;
    @JsonProperty("acute_surge_flag") private boolean acuteSurgeFlag;

    // — 9b. Cuffless research metrics —
    @JsonProperty("arv_cuffless") private Double arvCuffless; // reading-to-reading ARV from cuffless only

    // — 9c. Within-day variability —
    @JsonProperty("within_day_sd_sbp") private Double withinDaySdSbp; // SD of today's readings (>2 needed)

    // — 10. Data quality —
    @JsonProperty("context_depth") private String contextDepth; // INITIAL, BUILDING, ESTABLISHED
    @JsonProperty("total_readings_in_state") private int totalReadingsInState;
    @JsonProperty("days_with_data_in_7d") private int daysWithDataIn7d;
    @JsonProperty("days_with_data_in_30d") private int daysWithDataIn30d;

    public BPVariabilityMetrics() {}

    // — Getters and setters —

    // Identity
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String v) { this.correlationId = v; }
    public long getComputedAt() { return computedAt; }
    public void setComputedAt(long v) { this.computedAt = v; }

    // Trigger
    public double getTriggerSBP() { return triggerSBP; }
    public void setTriggerSBP(double v) { this.triggerSBP = v; }
    public double getTriggerDBP() { return triggerDBP; }
    public void setTriggerDBP(double v) { this.triggerDBP = v; }
    public String getTriggerSource() { return triggerSource; }
    public void setTriggerSource(String v) { this.triggerSource = v; }
    public String getTriggerTimeContext() { return triggerTimeContext; }
    public void setTriggerTimeContext(String v) { this.triggerTimeContext = v; }
    public long getTriggerTimestamp() { return triggerTimestamp; }
    public void setTriggerTimestamp(long v) { this.triggerTimestamp = v; }

    // 7-day
    public Double getArvSbp7d() { return arvSbp7d; }
    public void setArvSbp7d(Double v) { this.arvSbp7d = v; }
    public Double getSdSbp7d() { return sdSbp7d; }
    public void setSdSbp7d(Double v) { this.sdSbp7d = v; }
    public Double getCvSbp7d() { return cvSbp7d; }
    public void setCvSbp7d(Double v) { this.cvSbp7d = v; }
    public String getVariabilityClassification7d() { return variabilityClassification7d; }
    public void setVariabilityClassification7d(String v) { this.variabilityClassification7d = v; }

    // 30-day
    public Double getArvSbp30d() { return arvSbp30d; }
    public void setArvSbp30d(Double v) { this.arvSbp30d = v; }
    public Double getSdSbp30d() { return sdSbp30d; }
    public void setSdSbp30d(Double v) { this.sdSbp30d = v; }
    public Double getCvSbp30d() { return cvSbp30d; }
    public void setCvSbp30d(Double v) { this.cvSbp30d = v; }
    public String getVariabilityClassification30d() { return variabilityClassification30d; }
    public void setVariabilityClassification30d(String v) { this.variabilityClassification30d = v; }

    // Morning surge
    public Double getMorningSurgeToday() { return morningSurgeToday; }
    public void setMorningSurgeToday(Double v) { this.morningSurgeToday = v; }
    public Double getMorningSurge7dAvg() { return morningSurge7dAvg; }
    public void setMorningSurge7dAvg(Double v) { this.morningSurge7dAvg = v; }
    public String getSurgeClassification() { return surgeClassification; }
    public void setSurgeClassification(String v) { this.surgeClassification = v; }

    // Dipping
    public Double getDipRatio() { return dipRatio; }
    public void setDipRatio(Double v) { this.dipRatio = v; }
    public String getDipClassification() { return dipClassification; }
    public void setDipClassification(String v) { this.dipClassification = v; }

    // BP control
    public Double getSbp7dAvg() { return sbp7dAvg; }
    public void setSbp7dAvg(Double v) { this.sbp7dAvg = v; }
    public Double getDbp7dAvg() { return dbp7dAvg; }
    public void setDbp7dAvg(Double v) { this.dbp7dAvg = v; }
    public String getBpControlStatus() { return bpControlStatus; }
    public void setBpControlStatus(String v) { this.bpControlStatus = v; }

    // White-coat / masked
    public boolean isWhiteCoatSuspected() { return whiteCoatSuspected; }
    public void setWhiteCoatSuspected(boolean v) { this.whiteCoatSuspected = v; }
    public boolean isMaskedHTNSuspected() { return maskedHTNSuspected; }
    public void setMaskedHTNSuspected(boolean v) { this.maskedHTNSuspected = v; }
    public Double getClinicHomeGapSBP() { return clinicHomeGapSBP; }
    public void setClinicHomeGapSBP(Double v) { this.clinicHomeGapSBP = v; }

    // Crisis
    public boolean isCrisisFlag() { return crisisFlag; }
    public void setCrisisFlag(boolean v) { this.crisisFlag = v; }
    public boolean isAcuteSurgeFlag() { return acuteSurgeFlag; }
    public void setAcuteSurgeFlag(boolean v) { this.acuteSurgeFlag = v; }

    // Cuffless research
    public Double getArvCuffless() { return arvCuffless; }
    public void setArvCuffless(Double v) { this.arvCuffless = v; }

    // Within-day variability
    public Double getWithinDaySdSbp() { return withinDaySdSbp; }
    public void setWithinDaySdSbp(Double v) { this.withinDaySdSbp = v; }

    // Data quality
    public String getContextDepth() { return contextDepth; }
    public void setContextDepth(String v) { this.contextDepth = v; }
    public int getTotalReadingsInState() { return totalReadingsInState; }
    public void setTotalReadingsInState(int v) { this.totalReadingsInState = v; }
    public int getDaysWithDataIn7d() { return daysWithDataIn7d; }
    public void setDaysWithDataIn7d(int v) { this.daysWithDataIn7d = v; }
    public int getDaysWithDataIn30d() { return daysWithDataIn30d; }
    public void setDaysWithDataIn30d(int v) { this.daysWithDataIn30d = v; }
}
