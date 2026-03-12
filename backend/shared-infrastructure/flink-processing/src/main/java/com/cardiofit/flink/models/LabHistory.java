package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.LinkedList;
import java.util.List;
import java.util.stream.Collectors;

/**
 * Circular buffer for lab results history.
 *
 * Maintains a fixed-size buffer of recent lab results with automatic eviction
 * of oldest entries when capacity is reached.
 */
public class LabHistory implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("maxSize")
    private final int maxSize;

    @JsonProperty("labs")
    private final LinkedList<LabResult> labs;

    // ============================================================
    // CONSTRUCTORS
    // ============================================================

    public LabHistory(int maxSize) {
        this.maxSize = maxSize;
        this.labs = new LinkedList<>();
    }

    // ============================================================
    // OPERATIONS
    // ============================================================

    /**
     * Add new lab result to the buffer.
     * If buffer is full, removes the oldest entry.
     */
    public void add(LabResult lab) {
        if (lab == null) {
            return;
        }

        labs.addLast(lab);

        // Evict oldest if exceeded capacity
        if (labs.size() > maxSize) {
            labs.removeFirst();
        }
    }

    /**
     * Get the N most recent labs.
     *
     * @param count Number of labs to retrieve
     * @return List of recent labs (newest last)
     */
    public List<LabResult> getRecent(int count) {
        int size = labs.size();
        if (count >= size) {
            return new ArrayList<>(labs);
        }

        // Get last N elements
        return new ArrayList<>(labs.subList(size - count, size));
    }

    /**
     * Get all labs in the buffer.
     */
    public List<LabResult> getAll() {
        return new ArrayList<>(labs);
    }

    /**
     * Get the most recent (latest) lab result.
     */
    public LabResult getLatest() {
        return labs.isEmpty() ? null : labs.getLast();
    }

    /**
     * Get latest lab results as consolidated LabValues object.
     * This converts the lab history into a single LabValues instance
     * with the most recent value for each lab type.
     *
     * Used by Module 2 for clinical decision support.
     *
     * @return LabValues object with latest values, or null if no labs
     */
    public LabValues getLatestAsLabValues() {
        if (labs.isEmpty()) {
            return null;
        }
        return LabValues.fromLabHistory(this);
    }

    /**
     * Get all lab results of a specific type.
     *
     * @param labType The lab test type (e.g., "WBC", "Hemoglobin")
     * @return List of matching lab results
     */
    public List<LabResult> getByType(String labType) {
        return labs.stream()
                .filter(lab -> labType.equalsIgnoreCase(lab.getLabType()))
                .collect(Collectors.toList());
    }

    /**
     * Get the most recent result for a specific lab type.
     */
    public LabResult getLatestByType(String labType) {
        for (int i = labs.size() - 1; i >= 0; i--) {
            LabResult lab = labs.get(i);
            if (labType.equalsIgnoreCase(lab.getLabType())) {
                return lab;
            }
        }
        return null;
    }

    /**
     * Check if any recent labs have abnormal flags.
     *
     * @param recentCount Number of recent labs to check
     * @return True if any abnormal results found
     */
    public boolean hasAbnormalResults(int recentCount) {
        List<LabResult> recent = getRecent(recentCount);
        return recent.stream().anyMatch(LabResult::isAbnormal);
    }

    /**
     * Get trend for a specific lab type.
     *
     * @param labType The lab test type
     * @return "improving", "stable", "worsening", or "unknown"
     */
    public String getTrend(String labType) {
        List<LabResult> labResults = getByType(labType);
        if (labResults.size() < 2) {
            return "unknown";
        }

        // Compare first and last values
        LabResult oldest = labResults.get(0);
        LabResult newest = labResults.get(labResults.size() - 1);

        if (oldest.getValue() == null || newest.getValue() == null) {
            return "unknown";
        }

        double change = newest.getValue() - oldest.getValue();
        double threshold = Math.abs(oldest.getValue() * 0.05); // 5% change threshold

        if (Math.abs(change) < threshold) {
            return "stable";
        }

        // Check if moving toward normal range
        if (newest.isAbnormal() && !oldest.isAbnormal()) {
            return "worsening";
        } else if (!newest.isAbnormal() && oldest.isAbnormal()) {
            return "improving";
        } else if (newest.isAbnormal() && oldest.isAbnormal()) {
            // Both abnormal - check if getting closer to normal
            return isMovingTowardNormal(oldest, newest, labType) ? "improving" : "worsening";
        } else {
            return "stable";
        }
    }

    private boolean isMovingTowardNormal(LabResult older, LabResult newer, String labType) {
        // Simplified logic - would need reference ranges for each lab type
        // For now, just check if absolute deviation is decreasing
        Double normalValue = getNormalValue(labType);
        if (normalValue == null) {
            return false;
        }

        double olderDist = Math.abs(older.getValue() - normalValue);
        double newerDist = Math.abs(newer.getValue() - normalValue);

        return newerDist < olderDist;
    }

    private Double getNormalValue(String labType) {
        // Simplified normal values - would need comprehensive reference ranges
        switch (labType.toUpperCase()) {
            case "WBC":
                return 7.5; // Normal WBC ~5-10 K/uL, use midpoint
            case "HEMOGLOBIN":
                return 14.0; // Normal ~12-16 g/dL
            case "PLATELET":
                return 250.0; // Normal ~150-400 K/uL
            case "GLUCOSE":
                return 100.0; // Normal fasting ~70-130 mg/dL
            case "CREATININE":
                return 1.0; // Normal ~0.7-1.3 mg/dL
            default:
                return null;
        }
    }

    /**
     * Check if buffer is empty.
     */
    public boolean isEmpty() {
        return labs.isEmpty();
    }

    /**
     * Get current size of buffer.
     */
    public int size() {
        return labs.size();
    }

    /**
     * Clear all labs from buffer.
     */
    public void clear() {
        labs.clear();
    }

    @Override
    public String toString() {
        return "LabHistory{" +
                "size=" + labs.size() +
                "/" + maxSize +
                ", latest=" + getLatest() +
                '}';
    }
}
