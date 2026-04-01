package com.cardiofit.flink.models;

import java.io.Serializable;
import java.time.Instant;
import java.time.LocalDate;
import java.time.ZoneOffset;
import java.util.*;

/**
 * Per-patient BP state maintained in Flink keyed state.
 * Contains a 30-day rolling window of DailyBPSummary entries.
 *
 * State TTL: 30 days (per architecture Section 7.2).
 * First event after state expiry produces contextDepth: INITIAL.
 */
public class PatientBPState implements Serializable {
    private static final long serialVersionUID = 1L;

    private String patientId;
    private Map<String, DailyBPSummary> dailySummaries = new LinkedHashMap<>(); // dateKey -> summary
    private long lastReadingTime;
    private int totalReadingsProcessed;

    // Cuffless ARV: rolling buffer of recent cuffless SBP values (reading-to-reading, not daily).
    // Capped at 50 readings (~7 days at ~7 readings/day for a wearable).
    private static final int MAX_CUFFLESS_BUFFER = 50;
    private List<Double> cufflessSBPBuffer = new ArrayList<>();

    public PatientBPState() {}
    public PatientBPState(String patientId) { this.patientId = patientId; }

    /**
     * Add a reading to the appropriate day's summary.
     * Creates a new DailyBPSummary if this is the first reading for the day.
     * Evicts entries older than 30 days.
     */
    public void addReading(BPReading reading) {
        String dateKey = toDateKey(reading.getTimestamp());
        DailyBPSummary summary = dailySummaries.computeIfAbsent(
            dateKey, DailyBPSummary::new);
        summary.addReading(reading);
        lastReadingTime = Math.max(lastReadingTime, reading.getTimestamp());
        totalReadingsProcessed++;
        evictOlderThan30Days(reading.getTimestamp());
    }

    /**
     * Get daily summaries for the last N days that HAVE readings.
     * Days without readings are skipped - they don't contribute to ARV.
     * @param windowDays 7 or 30
     * @param referenceTime the current event time
     * @return ordered list of DailyBPSummary (oldest first)
     */
    public List<DailyBPSummary> getSummariesInWindow(int windowDays, long referenceTime) {
        LocalDate refDate = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate();
        LocalDate cutoff = refDate.minusDays(windowDays);

        List<DailyBPSummary> result = new ArrayList<>();
        for (Map.Entry<String, DailyBPSummary> entry : dailySummaries.entrySet()) {
            LocalDate date = LocalDate.parse(entry.getKey());
            if (!date.isBefore(cutoff) && !date.isAfter(refDate)) {
                result.add(entry.getValue());
            }
        }
        result.sort(Comparator.comparing(DailyBPSummary::getDateKey));
        return result;
    }

    private void evictOlderThan30Days(long referenceTime) {
        LocalDate cutoff = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().minusDays(31);
        dailySummaries.entrySet().removeIf(entry -> {
            LocalDate date = LocalDate.parse(entry.getKey());
            return date.isBefore(cutoff);
        });
    }

    private String toDateKey(long timestamp) {
        return Instant.ofEpochMilli(timestamp)
            .atZone(ZoneOffset.UTC).toLocalDate().toString();
    }

    /**
     * Add a cuffless reading to the rolling buffer.
     * Called BEFORE the non-clinical-grade skip in the operator,
     * so cuffless readings accumulate for research ARV even though
     * they don't contribute to clinical metrics.
     */
    public void addCufflessReading(double sbp) {
        cufflessSBPBuffer.add(sbp);
        if (cufflessSBPBuffer.size() > MAX_CUFFLESS_BUFFER) {
            cufflessSBPBuffer.remove(0);
        }
    }

    /**
     * Compute reading-to-reading ARV from the cuffless buffer.
     * Returns null if fewer than 3 cuffless readings.
     */
    public Double getCufflessARV() {
        if (cufflessSBPBuffer.size() < 3) return null;
        double sumAbsDiff = 0;
        for (int i = 1; i < cufflessSBPBuffer.size(); i++) {
            sumAbsDiff += Math.abs(cufflessSBPBuffer.get(i) - cufflessSBPBuffer.get(i - 1));
        }
        return sumAbsDiff / (cufflessSBPBuffer.size() - 1);
    }

    public String getContextDepth() {
        if (totalReadingsProcessed <= 1) return "INITIAL";
        List<DailyBPSummary> recent = getSummariesInWindow(7, lastReadingTime);
        if (recent.size() < 3) return "BUILDING";
        return "ESTABLISHED";
    }

    // — Standard getters/setters —
    public String getPatientId() { return patientId; }
    public void setPatientId(String patientId) { this.patientId = patientId; }
    public Map<String, DailyBPSummary> getDailySummaries() { return dailySummaries; }
    public void setDailySummaries(Map<String, DailyBPSummary> dailySummaries) { this.dailySummaries = dailySummaries; }
    public long getLastReadingTime() { return lastReadingTime; }
    public void setLastReadingTime(long lastReadingTime) { this.lastReadingTime = lastReadingTime; }
    public int getTotalReadingsProcessed() { return totalReadingsProcessed; }
    public void setTotalReadingsProcessed(int v) { totalReadingsProcessed = v; }
    public List<Double> getCufflessSBPBuffer() { return cufflessSBPBuffer; }
    public void setCufflessSBPBuffer(List<Double> v) { this.cufflessSBPBuffer = v; }
}
