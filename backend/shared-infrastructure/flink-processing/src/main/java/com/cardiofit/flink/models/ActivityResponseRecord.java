package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class ActivityResponseRecord implements Serializable {

    private static final long serialVersionUID = 1L;

    // --- Identity (4 fields) ---
    @JsonProperty("recordId")
    private String recordId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("activityEventId")
    private String activityEventId;

    @JsonProperty("correlationId")
    private String correlationId;

    // --- Activity metadata (5 fields) ---
    @JsonProperty("activityStartTime")
    private long activityStartTime;

    @JsonProperty("activityDurationMin")
    private double activityDurationMin;

    @JsonProperty("exerciseType")
    private ExerciseType exerciseType;

    @JsonProperty("reportedMETs")
    private double reportedMETs;

    @JsonProperty("metMinutes")
    private double metMinutes;

    // --- HR features (8 fields) ---
    @JsonProperty("restingHR")
    private Double restingHR;

    @JsonProperty("peakHR")
    private Double peakHR;

    @JsonProperty("meanActiveHR")
    private Double meanActiveHR;

    @JsonProperty("hrr1")
    private Double hrr1;

    @JsonProperty("hrr2")
    private Double hrr2;

    @JsonProperty("hrRecoveryClass")
    private HRRecoveryClass hrRecoveryClass;

    @JsonProperty("dominantZone")
    private ActivityIntensityZone dominantZone;

    @JsonProperty("hrReadingCount")
    private int hrReadingCount;

    // --- Glucose features (5 fields) ---
    @JsonProperty("preExerciseGlucose")
    private Double preExerciseGlucose;

    @JsonProperty("exerciseGlucoseDelta")
    private Double exerciseGlucoseDelta;

    @JsonProperty("glucoseNadir")
    private Double glucoseNadir;

    @JsonProperty("hypoglycemiaFlag")
    private boolean hypoglycemiaFlag;

    @JsonProperty("reboundHyperglycemiaFlag")
    private boolean reboundHyperglycemiaFlag;

    // --- BP features (4 fields) ---
    @JsonProperty("preExerciseSBP")
    private Double preExerciseSBP;

    @JsonProperty("peakExerciseSBP")
    private Double peakExerciseSBP;

    @JsonProperty("postExerciseSBP")
    private Double postExerciseSBP;

    @JsonProperty("exerciseBPResponse")
    private ExerciseBPResponse exerciseBPResponse;

    // --- Derived cardiac metrics (1 field) ---
    @JsonProperty("peakRPP")
    private Double peakRPP;

    // --- Processing metadata (5 fields) ---
    @JsonProperty("windowDurationMs")
    private long windowDurationMs;

    @JsonProperty("concurrent")
    private boolean concurrent;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("version")
    private String version;

    public ActivityResponseRecord() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final ActivityResponseRecord r = new ActivityResponseRecord();

        public Builder recordId(String v) { r.recordId = v; return this; }
        public Builder patientId(String v) { r.patientId = v; return this; }
        public Builder activityEventId(String v) { r.activityEventId = v; return this; }
        public Builder correlationId(String v) { r.correlationId = v; return this; }
        public Builder activityStartTime(long v) { r.activityStartTime = v; return this; }
        public Builder activityDurationMin(double v) { r.activityDurationMin = v; return this; }
        public Builder exerciseType(ExerciseType v) { r.exerciseType = v; return this; }
        public Builder reportedMETs(double v) { r.reportedMETs = v; return this; }
        public Builder metMinutes(double v) { r.metMinutes = v; return this; }
        public Builder restingHR(Double v) { r.restingHR = v; return this; }
        public Builder peakHR(Double v) { r.peakHR = v; return this; }
        public Builder meanActiveHR(Double v) { r.meanActiveHR = v; return this; }
        public Builder hrr1(Double v) { r.hrr1 = v; return this; }
        public Builder hrr2(Double v) { r.hrr2 = v; return this; }
        public Builder hrRecoveryClass(HRRecoveryClass v) { r.hrRecoveryClass = v; return this; }
        public Builder dominantZone(ActivityIntensityZone v) { r.dominantZone = v; return this; }
        public Builder hrReadingCount(int v) { r.hrReadingCount = v; return this; }
        public Builder preExerciseGlucose(Double v) { r.preExerciseGlucose = v; return this; }
        public Builder exerciseGlucoseDelta(Double v) { r.exerciseGlucoseDelta = v; return this; }
        public Builder glucoseNadir(Double v) { r.glucoseNadir = v; return this; }
        public Builder hypoglycemiaFlag(boolean v) { r.hypoglycemiaFlag = v; return this; }
        public Builder reboundHyperglycemiaFlag(boolean v) { r.reboundHyperglycemiaFlag = v; return this; }
        public Builder preExerciseSBP(Double v) { r.preExerciseSBP = v; return this; }
        public Builder peakExerciseSBP(Double v) { r.peakExerciseSBP = v; return this; }
        public Builder postExerciseSBP(Double v) { r.postExerciseSBP = v; return this; }
        public Builder exerciseBPResponse(ExerciseBPResponse v) { r.exerciseBPResponse = v; return this; }
        public Builder peakRPP(Double v) { r.peakRPP = v; return this; }
        public Builder windowDurationMs(long v) { r.windowDurationMs = v; return this; }
        public Builder concurrent(boolean v) { r.concurrent = v; return this; }
        public Builder qualityScore(double v) { r.qualityScore = v; return this; }
        public ActivityResponseRecord build() { return r; }
    }

    // --- Getters ---
    public String getRecordId() { return recordId; }
    public String getPatientId() { return patientId; }
    public String getActivityEventId() { return activityEventId; }
    public String getCorrelationId() { return correlationId; }
    public long getActivityStartTime() { return activityStartTime; }
    public double getActivityDurationMin() { return activityDurationMin; }
    public ExerciseType getExerciseType() { return exerciseType; }
    public double getReportedMETs() { return reportedMETs; }
    public double getMetMinutes() { return metMinutes; }
    public Double getRestingHR() { return restingHR; }
    public Double getPeakHR() { return peakHR; }
    public Double getMeanActiveHR() { return meanActiveHR; }
    public Double getHrr1() { return hrr1; }
    public Double getHrr2() { return hrr2; }
    public HRRecoveryClass getHrRecoveryClass() { return hrRecoveryClass; }
    public ActivityIntensityZone getDominantZone() { return dominantZone; }
    public int getHrReadingCount() { return hrReadingCount; }
    public Double getPreExerciseGlucose() { return preExerciseGlucose; }
    public Double getExerciseGlucoseDelta() { return exerciseGlucoseDelta; }
    public Double getGlucoseNadir() { return glucoseNadir; }
    public boolean isHypoglycemiaFlag() { return hypoglycemiaFlag; }
    public boolean isReboundHyperglycemiaFlag() { return reboundHyperglycemiaFlag; }
    public Double getPreExerciseSBP() { return preExerciseSBP; }
    public Double getPeakExerciseSBP() { return peakExerciseSBP; }
    public Double getPostExerciseSBP() { return postExerciseSBP; }
    public ExerciseBPResponse getExerciseBPResponse() { return exerciseBPResponse; }
    public Double getPeakRPP() { return peakRPP; }
    public long getWindowDurationMs() { return windowDurationMs; }
    public boolean isConcurrent() { return concurrent; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public double getQualityScore() { return qualityScore; }
    public String getVersion() { return version; }
    public void setCorrelationId(String v) { this.correlationId = v; }
    public void setRecordId(String v) { this.recordId = v; }

    @Override
    public String toString() {
        return "ActivityResponseRecord{" +
                "patientId='" + patientId + '\'' +
                ", activityEventId='" + activityEventId + '\'' +
                ", type=" + exerciseType +
                ", peakHR=" + peakHR +
                ", hrr1=" + hrr1 +
                ", hrrClass=" + hrRecoveryClass +
                ", glucoseDelta=" + exerciseGlucoseDelta +
                ", bpResponse=" + exerciseBPResponse +
                '}';
    }
}
