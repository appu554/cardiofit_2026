package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;

/**
 * Ring buffer for CGM readings with a maximum capacity of 1500 readings.
 * Maintains sorted order by timestamp. Excludes sensor warmup readings,
 * physiologically impossible values, and duplicates.
 *
 * Used by Module3_CGMAnalytics for TIR/TBR/TAR/CV/GMI/GRI computation,
 * sustained hypo/hyper detection, and AGP percentile generation.
 */
public class CGMReadingBuffer implements Serializable {
    private static final long serialVersionUID = 1L;

    public static final int MAX_CAPACITY = 1500;
    public static final double MIN_PHYSIOLOGICAL = 20.0;   // mg/dL
    public static final double MAX_PHYSIOLOGICAL = 500.0;   // mg/dL
    public static final long SENSOR_WARMUP_MS = 60L * 60_000L; // 60 minutes

    private final List<TimestampedReading> readings;
    private long sensorStartMs = Long.MIN_VALUE;

    public CGMReadingBuffer() {
        this.readings = new ArrayList<>();
    }

    /**
     * Add a reading, rejecting sensor warmup, physiologically impossible values,
     * and duplicate timestamps. Maintains sorted order by timestamp.
     *
     * @return true if the reading was accepted
     */
    public boolean addReading(long timestampMs, double glucoseMgDl) {
        // Reject sensor warmup period
        if (sensorStartMs != Long.MIN_VALUE
                && timestampMs < sensorStartMs + SENSOR_WARMUP_MS) {
            return false;
        }

        // Reject physiologically impossible values
        if (glucoseMgDl < MIN_PHYSIOLOGICAL || glucoseMgDl > MAX_PHYSIOLOGICAL) {
            return false;
        }

        // Reject duplicate timestamps
        int idx = Collections.binarySearch(readings, new TimestampedReading(timestampMs, 0),
                (a, b) -> Long.compare(a.timestampMs, b.timestampMs));
        if (idx >= 0) {
            return false; // exact timestamp already present
        }

        // Insert in sorted position
        int insertionPoint = -(idx + 1);
        readings.add(insertionPoint, new TimestampedReading(timestampMs, glucoseMgDl));

        // Evict oldest if over capacity
        if (readings.size() > MAX_CAPACITY) {
            readings.remove(0);
        }

        return true;
    }

    public void setSensorStartTime(long sensorStartMs) {
        this.sensorStartMs = sensorStartMs;
    }

    public long getSensorStartTime() {
        return this.sensorStartMs;
    }

    /**
     * Return all readings within [startMs, endMs] inclusive.
     */
    public List<TimestampedReading> getReadingsInWindow(long startMs, long endMs) {
        List<TimestampedReading> result = new ArrayList<>();
        for (TimestampedReading r : readings) {
            if (r.timestampMs >= startMs && r.timestampMs <= endMs) {
                result.add(r);
            } else if (r.timestampMs > endMs) {
                break; // sorted, no more matches
            }
        }
        return result;
    }

    /**
     * Compute data coverage as a percentage.
     * coverage = actualReadings / (windowDays * expectedPerDay) * 100
     */
    public double computeCoverage(long startMs, long endMs, int expectedPerDay) {
        List<TimestampedReading> window = getReadingsInWindow(startMs, endMs);
        double windowDays = (endMs - startMs) / (24.0 * 3600_000.0);
        if (windowDays <= 0 || expectedPerDay <= 0) return 0.0;
        double expected = windowDays * expectedPerDay;
        return (window.size() / expected) * 100.0;
    }

    /**
     * Find consecutive runs of readings below a given threshold.
     */
    public List<ConsecutiveRun> findConsecutiveRunsBelowThreshold(
            double threshold, long startMs, long endMs) {
        return findConsecutiveRuns(threshold, startMs, endMs, true);
    }

    /**
     * Find consecutive runs of readings above a given threshold.
     */
    public List<ConsecutiveRun> findConsecutiveRunsAboveThreshold(
            double threshold, long startMs, long endMs) {
        return findConsecutiveRuns(threshold, startMs, endMs, false);
    }

    private List<ConsecutiveRun> findConsecutiveRuns(
            double threshold, long startMs, long endMs, boolean below) {
        List<ConsecutiveRun> runs = new ArrayList<>();
        List<TimestampedReading> window = getReadingsInWindow(startMs, endMs);

        long runStart = -1;
        int runCount = 0;

        for (TimestampedReading r : window) {
            boolean matches = below
                    ? r.glucoseMgDl < threshold
                    : r.glucoseMgDl > threshold;

            if (matches) {
                if (runStart == -1) {
                    runStart = r.timestampMs;
                    runCount = 1;
                } else {
                    runCount++;
                }
            } else {
                if (runStart != -1) {
                    // End of run — use the previous reading's timestamp as endMs
                    long runEnd = window.get(window.indexOf(r) - 1).timestampMs;
                    runs.add(new ConsecutiveRun(runStart, runEnd, runCount));
                    runStart = -1;
                    runCount = 0;
                }
            }
        }

        // Close any open run
        if (runStart != -1 && !window.isEmpty()) {
            long runEnd = window.get(window.size() - 1).timestampMs;
            runs.add(new ConsecutiveRun(runStart, runEnd, runCount));
        }

        return runs;
    }

    public int size() {
        return readings.size();
    }

    public List<TimestampedReading> getAllReadings() {
        return Collections.unmodifiableList(readings);
    }

    // ---- Inner classes ----

    public static class TimestampedReading implements Serializable {
        private static final long serialVersionUID = 1L;

        public long timestampMs;
        public double glucoseMgDl;

        public TimestampedReading() {}

        public TimestampedReading(long timestampMs, double glucoseMgDl) {
            this.timestampMs = timestampMs;
            this.glucoseMgDl = glucoseMgDl;
        }
    }

    public static class ConsecutiveRun implements Serializable {
        private static final long serialVersionUID = 1L;

        public long startMs;
        public long endMs;
        public int readingCount;

        public ConsecutiveRun() {}

        public ConsecutiveRun(long startMs, long endMs, int readingCount) {
            this.startMs = startMs;
            this.endMs = endMs;
            this.readingCount = readingCount;
        }

        /** Duration of the run in minutes. */
        public double durationMinutes() {
            return (endMs - startMs) / 60_000.0;
        }
    }
}
