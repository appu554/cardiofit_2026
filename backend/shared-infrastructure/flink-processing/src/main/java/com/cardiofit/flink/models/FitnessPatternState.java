package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

@JsonIgnoreProperties(ignoreUnknown = true)
public class FitnessPatternState implements Serializable {

    private static final long serialVersionUID = 1L;

    public static final int VO2MAX_BUFFER_MAX_DAYS = 90;
    public static final long WEEK_MS = 7L * 86_400_000L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("patientAge")
    private Integer patientAge;

    @JsonProperty("hrMax")
    private double hrMax;

    @JsonProperty("lastKnownRestingHR")
    private Double lastKnownRestingHR;

    @JsonProperty("vo2maxEstimates")
    private List<VO2maxEstimate> vo2maxEstimates;

    @JsonProperty("weeklyActivityRecords")
    private List<ActivityResponseRecord> weeklyActivityRecords;

    @JsonProperty("weeklyTimerRegistered")
    private boolean weeklyTimerRegistered;

    @JsonProperty("lastWeeklyEmitTimestamp")
    private long lastWeeklyEmitTimestamp;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    @JsonProperty("totalRecordsProcessed")
    private long totalRecordsProcessed;

    public FitnessPatternState() {
        this.vo2maxEstimates = new ArrayList<>();
        this.weeklyActivityRecords = new ArrayList<>();
        this.hrMax = ActivityIntensityZone.DEFAULT_HR_MAX;
    }

    public FitnessPatternState(String patientId) {
        this();
        this.patientId = patientId;
    }

    public void addActivityRecord(ActivityResponseRecord record) {
        weeklyActivityRecords.add(record);
        totalRecordsProcessed++;
    }

    public void addVO2maxEstimate(double vo2max, long timestamp) {
        vo2maxEstimates.add(new VO2maxEstimate(vo2max, timestamp));
        long cutoff = timestamp - (VO2MAX_BUFFER_MAX_DAYS * 86_400_000L);
        vo2maxEstimates.removeIf(e -> e.timestamp < cutoff);
    }

    public List<ActivityResponseRecord> drainWeeklyRecords() {
        List<ActivityResponseRecord> drained = new ArrayList<>(weeklyActivityRecords);
        weeklyActivityRecords.clear();
        return drained;
    }

    public Double computeVO2maxTrendPerWeek() {
        if (vo2maxEstimates.size() < 3) return null;
        int n = vo2maxEstimates.size();
        double sumX = 0, sumY = 0;
        for (VO2maxEstimate e : vo2maxEstimates) {
            sumX += e.timestamp;
            sumY += e.vo2max;
        }
        double meanX = sumX / n;
        double meanY = sumY / n;
        double ssXX = 0, ssXY = 0;
        for (VO2maxEstimate e : vo2maxEstimates) {
            double dx = e.timestamp - meanX;
            ssXX += dx * dx;
            ssXY += dx * (e.vo2max - meanY);
        }
        if (ssXX < 1e-12) return 0.0;
        double slopePerMs = ssXY / ssXX;
        return slopePerMs * WEEK_MS;
    }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public Integer getPatientAge() { return patientAge; }
    public void setPatientAge(Integer v) { this.patientAge = v; }
    public double getHrMax() { return hrMax; }
    public void setHrMax(double v) { this.hrMax = v; }
    public Double getLastKnownRestingHR() { return lastKnownRestingHR; }
    public void setLastKnownRestingHR(Double v) { this.lastKnownRestingHR = v; }
    public List<VO2maxEstimate> getVo2maxEstimates() { return vo2maxEstimates; }
    public List<ActivityResponseRecord> getWeeklyActivityRecords() { return weeklyActivityRecords; }
    public boolean isWeeklyTimerRegistered() { return weeklyTimerRegistered; }
    public void setWeeklyTimerRegistered(boolean v) { this.weeklyTimerRegistered = v; }
    public long getLastWeeklyEmitTimestamp() { return lastWeeklyEmitTimestamp; }
    public void setLastWeeklyEmitTimestamp(long v) { this.lastWeeklyEmitTimestamp = v; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }
    public long getTotalRecordsProcessed() { return totalRecordsProcessed; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class VO2maxEstimate implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("vo2max")
        public double vo2max;

        @JsonProperty("timestamp")
        public long timestamp;

        public VO2maxEstimate() {}

        public VO2maxEstimate(double vo2max, long timestamp) {
            this.vo2max = vo2max;
            this.timestamp = timestamp;
        }
    }
}
