package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * VitalMetric represents aggregated vital sign statistics over a time window
 * Used by Module 6 Time-Series Aggregator for multi-resolution rollups
 *
 * Created from 1-minute window aggregations of vital signs with min/max/avg statistics
 * Output to time-series database (InfluxDB/TimescaleDB) for dashboard visualization
 *
 * Example: heart_rate aggregation over 1 minute
 * - avg: 88.5 bpm
 * - min: 82.0 bpm
 * - max: 95.0 bpm
 * - count: 12 readings
 */
public class VitalMetric implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("patient_id")
    private String patientId;

    @JsonProperty("vital_type")
    private String vitalType;

    @JsonProperty("avg")
    private double avg;

    @JsonProperty("min")
    private double min;

    @JsonProperty("max")
    private double max;

    @JsonProperty("count")
    private int count;

    @JsonProperty("timestamp")
    private long timestamp;

    @JsonProperty("window_start")
    private long windowStart;

    @JsonProperty("window_end")
    private long windowEnd;

    // Default constructor
    public VitalMetric() {}

    // Full constructor
    public VitalMetric(String patientId, String vitalType, double avg, double min, double max,
                      int count, long timestamp, long windowStart, long windowEnd) {
        this.patientId = patientId;
        this.vitalType = vitalType;
        this.avg = avg;
        this.min = min;
        this.max = max;
        this.count = count;
        this.timestamp = timestamp;
        this.windowStart = windowStart;
        this.windowEnd = windowEnd;
    }

    // Builder pattern
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private VitalMetric metric = new VitalMetric();

        public Builder patientId(String patientId) {
            metric.patientId = patientId;
            return this;
        }

        public Builder vitalType(String vitalType) {
            metric.vitalType = vitalType;
            return this;
        }

        public Builder avg(double avg) {
            metric.avg = avg;
            return this;
        }

        public Builder min(double min) {
            metric.min = min;
            return this;
        }

        public Builder max(double max) {
            metric.max = max;
            return this;
        }

        public Builder count(int count) {
            metric.count = count;
            return this;
        }

        public Builder timestamp(long timestamp) {
            metric.timestamp = timestamp;
            return this;
        }

        public Builder windowStart(long windowStart) {
            metric.windowStart = windowStart;
            return this;
        }

        public Builder windowEnd(long windowEnd) {
            metric.windowEnd = windowEnd;
            return this;
        }

        public VitalMetric build() {
            return metric;
        }
    }

    // Getters and Setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getVitalType() { return vitalType; }
    public void setVitalType(String vitalType) { this.vitalType = vitalType; }

    public double getAvg() { return avg; }
    public void setAvg(double avg) { this.avg = avg; }

    public double getMin() { return min; }
    public void setMin(double min) { this.min = min; }

    public double getMax() { return max; }
    public void setMax(double max) { this.max = max; }

    public int getCount() { return count; }
    public void setCount(int count) { this.count = count; }

    public long getTimestamp() { return timestamp; }
    public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

    public long getWindowStart() { return windowStart; }
    public void setWindowStart(long windowStart) { this.windowStart = windowStart; }

    public long getWindowEnd() { return windowEnd; }
    public void setWindowEnd(long windowEnd) { this.windowEnd = windowEnd; }

    // Utility methods
    /**
     * Get the variability (max - min) for this vital
     */
    public double getVariability() {
        return max - min;
    }

    /**
     * Check if this vital shows high variability (>20% of average)
     */
    public boolean isHighlyVariable() {
        if (avg == 0) return false;
        return (getVariability() / avg) > 0.20;
    }

    /**
     * Get window duration in milliseconds
     */
    public long getWindowDuration() {
        return windowEnd - windowStart;
    }

    @Override
    public String toString() {
        return "VitalMetric{" +
                "patientId='" + patientId + '\'' +
                ", vitalType='" + vitalType + '\'' +
                ", avg=" + String.format("%.2f", avg) +
                ", min=" + String.format("%.2f", min) +
                ", max=" + String.format("%.2f", max) +
                ", count=" + count +
                ", timestamp=" + timestamp +
                '}';
    }
}
