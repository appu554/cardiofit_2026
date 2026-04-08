package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Timestamped heart rate readings buffer for a single exercise session window.
 *
 * Three-phase structure:
 * 1. Pre-exercise: readings within 10 min before activity start (baseline HR)
 * 2. Active exercise: readings during the activity (peak HR, zone distribution)
 * 3. Recovery: readings within 2h after activity end (HRR₁, HRR₂)
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class HRWindow implements Serializable {

    private static final long serialVersionUID = 1L;

    @JsonProperty("readings")
    private List<HRReading> readings;

    @JsonProperty("restingHR")
    private Double restingHR;

    @JsonProperty("hrMax")
    private double hrMax;

    @JsonProperty("activityStartTime")
    private long activityStartTime;

    @JsonProperty("activityEndTime")
    private Long activityEndTime;

    @JsonProperty("windowCloseTime")
    private long windowCloseTime;

    public HRWindow() {
        this.readings = new ArrayList<>();
        this.hrMax = ActivityIntensityZone.DEFAULT_HR_MAX;
    }

    public void addReading(long timestamp, double heartRate, String source) {
        readings.add(new HRReading(timestamp, heartRate, source));
    }

    public int size() {
        return readings.size();
    }

    public boolean isEmpty() {
        return readings.isEmpty();
    }

    public void sortByTime() {
        readings.sort((a, b) -> Long.compare(a.timestamp, b.timestamp));
    }

    public List<HRReading> getActivePhaseReadings() {
        List<HRReading> active = new ArrayList<>();
        long end = activityEndTime != null ? activityEndTime : windowCloseTime;
        for (HRReading r : readings) {
            if (r.timestamp >= activityStartTime && r.timestamp <= end) {
                active.add(r);
            }
        }
        return active;
    }

    public List<HRReading> getRecoveryPhaseReadings() {
        if (activityEndTime == null) return new ArrayList<>();
        List<HRReading> recovery = new ArrayList<>();
        for (HRReading r : readings) {
            if (r.timestamp > activityEndTime) {
                recovery.add(r);
            }
        }
        return recovery;
    }

    public Map<ActivityIntensityZone, Long> computeZoneDistribution() {
        Map<ActivityIntensityZone, Long> zones = new HashMap<>();
        for (ActivityIntensityZone z : ActivityIntensityZone.values()) {
            zones.put(z, 0L);
        }
        List<HRReading> active = getActivePhaseReadings();
        if (active.size() < 2) return zones;
        for (int i = 0; i < active.size() - 1; i++) {
            double avgHR = (active.get(i).heartRate + active.get(i + 1).heartRate) / 2.0;
            long dt = active.get(i + 1).timestamp - active.get(i).timestamp;
            ActivityIntensityZone zone = ActivityIntensityZone.fromHeartRate(avgHR, hrMax);
            zones.merge(zone, dt, Long::sum);
        }
        return zones;
    }

    // --- Getters/Setters ---
    public List<HRReading> getReadings() { return readings; }
    public Double getRestingHR() { return restingHR; }
    public void setRestingHR(Double v) { this.restingHR = v; }
    public double getHrMax() { return hrMax; }
    public void setHrMax(double v) { this.hrMax = v; }
    public long getActivityStartTime() { return activityStartTime; }
    public void setActivityStartTime(long v) { this.activityStartTime = v; }
    public Long getActivityEndTime() { return activityEndTime; }
    public void setActivityEndTime(Long v) { this.activityEndTime = v; }
    public long getWindowCloseTime() { return windowCloseTime; }
    public void setWindowCloseTime(long v) { this.windowCloseTime = v; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class HRReading implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("timestamp")
        public long timestamp;

        @JsonProperty("heartRate")
        public double heartRate;

        @JsonProperty("source")
        public String source;

        public HRReading() {}

        public HRReading(long timestamp, double heartRate, String source) {
            this.timestamp = timestamp;
            this.heartRate = heartRate;
            this.source = source;
        }
    }
}
