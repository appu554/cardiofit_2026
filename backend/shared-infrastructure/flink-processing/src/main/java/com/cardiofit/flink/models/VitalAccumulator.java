package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * VitalAccumulator maintains running statistics during window aggregation
 * Used as intermediate state by Flink AggregateFunction
 *
 * Pattern: createAccumulator() → add() → add() → ... → getResult()
 *
 * Tracks:
 * - Running sum for average calculation
 * - Min/max values seen in window
 * - Count of data points
 * - Patient and vital type identifiers
 */
public class VitalAccumulator implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private String vitalType;
    private double sum = 0.0;
    private double min = Double.MAX_VALUE;
    private double max = Double.MIN_VALUE;
    private int count = 0;
    private long windowStart = 0L;
    private long windowEnd = 0L;

    // Default constructor
    public VitalAccumulator() {}

    // Constructor with basic fields
    public VitalAccumulator(String patientId, String vitalType) {
        this.patientId = patientId;
        this.vitalType = vitalType;
    }

    // Getters and Setters
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }

    public String getVitalType() { return vitalType; }
    public void setVitalType(String vitalType) { this.vitalType = vitalType; }

    public double getSum() { return sum; }
    public void setSum(double sum) { this.sum = sum; }

    public double getMin() { return min; }
    public void setMin(double min) { this.min = min; }

    public double getMax() { return max; }
    public void setMax(double max) { this.max = max; }

    public int getCount() { return count; }
    public void setCount(int count) { this.count = count; }

    public long getWindowStart() { return windowStart; }
    public void setWindowStart(long windowStart) { this.windowStart = windowStart; }

    public long getWindowEnd() { return windowEnd; }
    public void setWindowEnd(long windowEnd) { this.windowEnd = windowEnd; }

    /**
     * Calculate average from accumulated sum and count
     */
    public double getAverage() {
        return count > 0 ? sum / count : 0.0;
    }

    /**
     * Add a new value to the accumulator
     */
    public void add(double value) {
        this.sum += value;
        this.min = Math.min(this.min, value);
        this.max = Math.max(this.max, value);
        this.count++;
    }

    /**
     * Merge another accumulator into this one (for parallel processing)
     */
    public void merge(VitalAccumulator other) {
        this.sum += other.sum;
        this.min = Math.min(this.min, other.min);
        this.max = Math.max(this.max, other.max);
        this.count += other.count;

        // Keep earliest window start and latest window end
        if (other.windowStart > 0 && (this.windowStart == 0 || other.windowStart < this.windowStart)) {
            this.windowStart = other.windowStart;
        }
        if (other.windowEnd > this.windowEnd) {
            this.windowEnd = other.windowEnd;
        }
    }

    /**
     * Reset the accumulator to initial state
     */
    public void reset() {
        this.sum = 0.0;
        this.min = Double.MAX_VALUE;
        this.max = Double.MIN_VALUE;
        this.count = 0;
        this.windowStart = 0L;
        this.windowEnd = 0L;
    }

    @Override
    public String toString() {
        return "VitalAccumulator{" +
                "patientId='" + patientId + '\'' +
                ", vitalType='" + vitalType + '\'' +
                ", sum=" + sum +
                ", min=" + min +
                ", max=" + max +
                ", count=" + count +
                ", avg=" + String.format("%.2f", getAverage()) +
                '}';
    }
}
