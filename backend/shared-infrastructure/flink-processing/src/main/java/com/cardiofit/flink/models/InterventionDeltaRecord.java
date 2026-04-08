package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

@JsonIgnoreProperties(ignoreUnknown = true)
public class InterventionDeltaRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("delta_id") private String deltaId;
    @JsonProperty("intervention_id") private String interventionId;
    @JsonProperty("patient_id") private String patientId;
    @JsonProperty("intervention_type") private InterventionType interventionType;
    @JsonProperty("fbg_delta") private Double fbgDelta;
    @JsonProperty("sbp_delta") private Double sbpDelta;
    @JsonProperty("dbp_delta") private Double dbpDelta;
    @JsonProperty("weight_delta_kg") private Double weightDeltaKg;
    @JsonProperty("hba1c_delta") private Double hba1cDelta;
    @JsonProperty("egfr_delta") private Double egfrDelta;
    @JsonProperty("tir_delta") private Double tirDelta;
    /** Placeholder for future MRI risk index integration. Currently always null. */
    @JsonProperty("mri_score_delta") private Double mriScoreDelta;
    @JsonProperty("trajectory_attribution") private TrajectoryAttribution trajectoryAttribution;
    @JsonProperty("adherence_score") private Double adherenceScore;
    @JsonProperty("concurrent_count") private int concurrentCount;
    @JsonProperty("data_completeness_score") private double dataCompletenessScore;
    @JsonProperty("processing_timestamp") private long processingTimestamp;
    @JsonProperty("version") private String version = "1.0";

    public InterventionDeltaRecord() {}

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final InterventionDeltaRecord r = new InterventionDeltaRecord();
        public Builder deltaId(String v) { r.deltaId = v; return this; }
        public Builder interventionId(String v) { r.interventionId = v; return this; }
        public Builder patientId(String v) { r.patientId = v; return this; }
        public Builder interventionType(InterventionType v) { r.interventionType = v; return this; }
        public Builder fbgDelta(Double v) { r.fbgDelta = v; return this; }
        public Builder sbpDelta(Double v) { r.sbpDelta = v; return this; }
        public Builder dbpDelta(Double v) { r.dbpDelta = v; return this; }
        public Builder weightDeltaKg(Double v) { r.weightDeltaKg = v; return this; }
        public Builder hba1cDelta(Double v) { r.hba1cDelta = v; return this; }
        public Builder egfrDelta(Double v) { r.egfrDelta = v; return this; }
        public Builder tirDelta(Double v) { r.tirDelta = v; return this; }
        public Builder mriScoreDelta(Double v) { r.mriScoreDelta = v; return this; }
        public Builder trajectoryAttribution(TrajectoryAttribution v) { r.trajectoryAttribution = v; return this; }
        public Builder adherenceScore(Double v) { r.adherenceScore = v; return this; }
        public Builder concurrentCount(int v) { r.concurrentCount = v; return this; }
        public Builder dataCompletenessScore(double v) { r.dataCompletenessScore = v; return this; }
        public Builder processingTimestamp(long v) { r.processingTimestamp = v; return this; }
        public Builder version(String v) { r.version = v; return this; }
        public InterventionDeltaRecord build() { return r; }
    }

    public String getDeltaId() { return deltaId; }
    public String getInterventionId() { return interventionId; }
    public String getPatientId() { return patientId; }
    public InterventionType getInterventionType() { return interventionType; }
    public Double getFbgDelta() { return fbgDelta; }
    public Double getSbpDelta() { return sbpDelta; }
    public Double getDbpDelta() { return dbpDelta; }
    public Double getWeightDeltaKg() { return weightDeltaKg; }
    public Double getHba1cDelta() { return hba1cDelta; }
    public Double getEgfrDelta() { return egfrDelta; }
    public Double getTirDelta() { return tirDelta; }
    public Double getMriScoreDelta() { return mriScoreDelta; }
    public TrajectoryAttribution getTrajectoryAttribution() { return trajectoryAttribution; }
    public Double getAdherenceScore() { return adherenceScore; }
    public int getConcurrentCount() { return concurrentCount; }
    public double getDataCompletenessScore() { return dataCompletenessScore; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public String getVersion() { return version; }
}
