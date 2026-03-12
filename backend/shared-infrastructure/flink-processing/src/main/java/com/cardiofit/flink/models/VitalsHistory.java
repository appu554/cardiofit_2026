package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;

/**
 * Circular buffer for vital signs history.
 *
 * Maintains a fixed-size buffer of recent vital signs with automatic eviction
 * of oldest entries when capacity is reached. This is memory-efficient for
 * storing in Flink's state backend.
 */
public class VitalsHistory implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("maxSize")
    private final int maxSize;

    @JsonProperty("vitals")
    private final LinkedList<VitalSign> vitals;

    // ============================================================
    // CONSTRUCTORS
    // ============================================================

    public VitalsHistory(int maxSize) {
        this.maxSize = maxSize;
        this.vitals = new LinkedList<>();
    }

    // ============================================================
    // OPERATIONS
    // ============================================================

    /**
     * Add new vital sign to the buffer.
     * If buffer is full, removes the oldest entry.
     */
    public void add(VitalSign vital) {
        if (vital == null) {
            return;
        }

        vitals.addLast(vital);

        // Evict oldest if exceeded capacity
        if (vitals.size() > maxSize) {
            vitals.removeFirst();
        }
    }

    /**
     * Get the N most recent vitals.
     *
     * @param count Number of vitals to retrieve
     * @return List of recent vitals (newest last)
     */
    public List<VitalSign> getRecent(int count) {
        int size = vitals.size();
        if (count >= size) {
            return new ArrayList<>(vitals);
        }

        // Get last N elements
        return new ArrayList<>(vitals.subList(size - count, size));
    }

    /**
     * Get all vitals in the buffer.
     */
    public List<VitalSign> getAll() {
        return new ArrayList<>(vitals);
    }

    /**
     * Get the most recent (latest) vital sign.
     */
    public VitalSign getLatest() {
        return vitals.isEmpty() ? null : vitals.getLast();
    }

    /**
     * Get trend for a specific vital type.
     *
     * @param vitalType The vital sign type (e.g., "heart_rate")
     * @return "improving", "stable", "declining", or "unknown"
     */
    public String getTrend(String vitalType) {
        if (vitals.size() < 2) {
            return "unknown";
        }

        List<Double> values = new ArrayList<>();
        for (VitalSign vital : vitals) {
            Double value = extractVitalValue(vital, vitalType);
            if (value != null) {
                values.add(value);
            }
        }

        if (values.size() < 2) {
            return "unknown";
        }

        // Simple trend: compare first half average to second half average
        int mid = values.size() / 2;
        double firstHalf = values.subList(0, mid).stream()
                .mapToDouble(Double::doubleValue).average().orElse(0);
        double secondHalf = values.subList(mid, values.size()).stream()
                .mapToDouble(Double::doubleValue).average().orElse(0);

        double change = secondHalf - firstHalf;
        double threshold = firstHalf * 0.05; // 5% change threshold

        if (Math.abs(change) < threshold) {
            return "stable";
        } else if (change > 0) {
            // For vitals, increasing may be improving or declining depending on the vital
            // This is simplified - real implementation would use clinical knowledge
            return isImprovingWhenIncreasing(vitalType) ? "improving" : "declining";
        } else {
            return isImprovingWhenIncreasing(vitalType) ? "declining" : "improving";
        }
    }

    private Double extractVitalValue(VitalSign vital, String vitalType) {
        switch (vitalType.toLowerCase()) {
            case "heart_rate":
                return vital.getHeartRate();
            case "blood_pressure_systolic":
                return vital.getBloodPressureSystolic();
            case "blood_pressure_diastolic":
                return vital.getBloodPressureDiastolic();
            case "temperature":
                return vital.getTemperature();
            case "respiratory_rate":
                return vital.getRespiratoryRate();
            case "oxygen_saturation":
                return vital.getOxygenSaturation();
            default:
                return null;
        }
    }

    private boolean isImprovingWhenIncreasing(String vitalType) {
        // Simplified clinical logic
        switch (vitalType.toLowerCase()) {
            case "oxygen_saturation":
                return true; // Higher O2 sat is better
            case "heart_rate":
            case "respiratory_rate":
            case "temperature":
                return false; // Lower is generally better (in abnormal ranges)
            default:
                return false;
        }
    }

    /**
     * Check if buffer is empty.
     */
    public boolean isEmpty() {
        return vitals.isEmpty();
    }

    /**
     * Get current size of buffer.
     */
    public int size() {
        return vitals.size();
    }

    /**
     * Clear all vitals from buffer.
     */
    public void clear() {
        vitals.clear();
    }

    // ============================================================
    // SPECIFIC TREND METHODS (for Module 2 compatibility)
    // ============================================================

    /**
     * Get heart rate trend analysis.
     * @return Trend direction enum
     */
    public TrendDirection getHeartRateTrend() {
        return convertStringToTrendDirection(getTrend("heart_rate"));
    }

    /**
     * Get blood pressure trend analysis.
     * @return Trend direction based on systolic pressure
     */
    public TrendDirection getBloodPressureTrend() {
        return convertStringToTrendDirection(getTrend("blood_pressure_systolic"));
    }

    /**
     * Get oxygen saturation trend analysis.
     * @return Trend direction enum
     */
    public TrendDirection getOxygenSaturationTrend() {
        return convertStringToTrendDirection(getTrend("oxygen_saturation"));
    }

    /**
     * Get temperature trend analysis.
     * @return Trend direction enum
     */
    public TrendDirection getTemperatureTrend() {
        return convertStringToTrendDirection(getTrend("temperature"));
    }

    /**
     * Convert string trend to TrendDirection enum.
     */
    private TrendDirection convertStringToTrendDirection(String trend) {
        if (trend == null) {
            return TrendDirection.UNKNOWN;
        }
        switch (trend.toLowerCase()) {
            case "improving":
            case "increasing":
                return TrendDirection.INCREASING;
            case "declining":
            case "decreasing":
                return TrendDirection.DECREASING;
            case "stable":
                return TrendDirection.STABLE;
            default:
                return TrendDirection.UNKNOWN;
        }
    }

    @Override
    public String toString() {
        return "VitalsHistory{" +
                "size=" + vitals.size() +
                "/" + maxSize +
                ", latest=" + getLatest() +
                '}';
    }
}
