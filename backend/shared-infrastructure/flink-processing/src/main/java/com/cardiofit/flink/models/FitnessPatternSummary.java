package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

@JsonIgnoreProperties(ignoreUnknown = true)
public class FitnessPatternSummary implements Serializable {

    private static final long serialVersionUID = 1L;

    @JsonProperty("summaryId")
    private String summaryId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("correlationId")
    private String correlationId;

    @JsonProperty("periodStartMs")
    private long periodStartMs;

    @JsonProperty("periodEndMs")
    private long periodEndMs;

    @JsonProperty("totalMetMinutes")
    private double totalMetMinutes;

    @JsonProperty("totalActiveDurationMin")
    private double totalActiveDurationMin;

    @JsonProperty("activityCount")
    private int activityCount;

    @JsonProperty("meanPeakHR")
    private Double meanPeakHR;

    @JsonProperty("meanHRR1")
    private Double meanHRR1;

    @JsonProperty("dominantHRRecoveryClass")
    private HRRecoveryClass dominantHRRecoveryClass;

    @JsonProperty("zoneDistributionPct")
    private Map<ActivityIntensityZone, Double> zoneDistributionPct;

    @JsonProperty("estimatedVO2max")
    private Double estimatedVO2max;

    @JsonProperty("fitnessLevel")
    private FitnessLevel fitnessLevel;

    @JsonProperty("vo2maxTrend")
    private Double vo2maxTrend;

    @JsonProperty("meanExerciseGlucoseDelta")
    private Double meanExerciseGlucoseDelta;

    @JsonProperty("hypoglycemiaEventCount")
    private int hypoglycemiaEventCount;

    @JsonProperty("meanGlucoseDropAerobic")
    private Double meanGlucoseDropAerobic;

    @JsonProperty("exerciseTypeBreakdown")
    private Map<ExerciseType, ExerciseTypeStats> exerciseTypeBreakdown;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("sessionsWithHR")
    private int sessionsWithHR;

    @JsonProperty("sessionsWithGlucose")
    private int sessionsWithGlucose;

    @JsonProperty("version")
    private String version;

    public FitnessPatternSummary() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    // --- Getters/Setters ---
    public String getSummaryId() { return summaryId; }
    public void setSummaryId(String v) { this.summaryId = v; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String v) { this.correlationId = v; }
    public long getPeriodStartMs() { return periodStartMs; }
    public void setPeriodStartMs(long v) { this.periodStartMs = v; }
    public long getPeriodEndMs() { return periodEndMs; }
    public void setPeriodEndMs(long v) { this.periodEndMs = v; }
    public double getTotalMetMinutes() { return totalMetMinutes; }
    public void setTotalMetMinutes(double v) { this.totalMetMinutes = v; }
    public double getTotalActiveDurationMin() { return totalActiveDurationMin; }
    public void setTotalActiveDurationMin(double v) { this.totalActiveDurationMin = v; }
    public int getActivityCount() { return activityCount; }
    public void setActivityCount(int v) { this.activityCount = v; }
    public Double getMeanPeakHR() { return meanPeakHR; }
    public void setMeanPeakHR(Double v) { this.meanPeakHR = v; }
    public Double getMeanHRR1() { return meanHRR1; }
    public void setMeanHRR1(Double v) { this.meanHRR1 = v; }
    public HRRecoveryClass getDominantHRRecoveryClass() { return dominantHRRecoveryClass; }
    public void setDominantHRRecoveryClass(HRRecoveryClass v) { this.dominantHRRecoveryClass = v; }
    public Map<ActivityIntensityZone, Double> getZoneDistributionPct() { return zoneDistributionPct; }
    public void setZoneDistributionPct(Map<ActivityIntensityZone, Double> v) { this.zoneDistributionPct = v; }
    public Double getEstimatedVO2max() { return estimatedVO2max; }
    public void setEstimatedVO2max(Double v) { this.estimatedVO2max = v; }
    public FitnessLevel getFitnessLevel() { return fitnessLevel; }
    public void setFitnessLevel(FitnessLevel v) { this.fitnessLevel = v; }
    public Double getVo2maxTrend() { return vo2maxTrend; }
    public void setVo2maxTrend(Double v) { this.vo2maxTrend = v; }
    public Double getMeanExerciseGlucoseDelta() { return meanExerciseGlucoseDelta; }
    public void setMeanExerciseGlucoseDelta(Double v) { this.meanExerciseGlucoseDelta = v; }
    public int getHypoglycemiaEventCount() { return hypoglycemiaEventCount; }
    public void setHypoglycemiaEventCount(int v) { this.hypoglycemiaEventCount = v; }
    public Double getMeanGlucoseDropAerobic() { return meanGlucoseDropAerobic; }
    public void setMeanGlucoseDropAerobic(Double v) { this.meanGlucoseDropAerobic = v; }
    public Map<ExerciseType, ExerciseTypeStats> getExerciseTypeBreakdown() { return exerciseTypeBreakdown; }
    public void setExerciseTypeBreakdown(Map<ExerciseType, ExerciseTypeStats> v) { this.exerciseTypeBreakdown = v; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public double getQualityScore() { return qualityScore; }
    public void setQualityScore(double v) { this.qualityScore = v; }
    public int getSessionsWithHR() { return sessionsWithHR; }
    public void setSessionsWithHR(int v) { this.sessionsWithHR = v; }
    public int getSessionsWithGlucose() { return sessionsWithGlucose; }
    public void setSessionsWithGlucose(int v) { this.sessionsWithGlucose = v; }
    public String getVersion() { return version; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class ExerciseTypeStats implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("sessionCount")
        public int sessionCount;

        @JsonProperty("totalMetMinutes")
        public double totalMetMinutes;

        @JsonProperty("meanPeakHR")
        public double meanPeakHR;

        @JsonProperty("meanGlucoseDelta")
        public Double meanGlucoseDelta;

        public ExerciseTypeStats() {}
    }
}
