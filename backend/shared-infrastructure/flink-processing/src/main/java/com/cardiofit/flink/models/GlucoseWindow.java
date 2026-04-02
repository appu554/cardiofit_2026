package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;

/**
 * Timestamped glucose readings buffer for a single meal session window.
 * Sorted by timestamp. Used by Module10GlucoseAnalyzer to compute iAUC,
 * peak, time-to-peak, recovery time, and curve shape.
 *
 * Window duration: 3h (CGM) or 2h (SMBG).
 * Grace period: +5 min (processing-time timer fires at 3h05m).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class GlucoseWindow implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("readings")
    private List<Reading> readings;

    @JsonProperty("baseline")
    private Double baseline;

    @JsonProperty("windowOpenTime")
    private long windowOpenTime;

    @JsonProperty("windowCloseTime")
    private long windowCloseTime;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    public GlucoseWindow() {
        this.readings = new ArrayList<>();
    }

    public void addReading(long timestamp, double glucoseValue, String source) {
        readings.add(new Reading(timestamp, glucoseValue, source));
    }

    public int size() {
        return readings.size();
    }

    public boolean isEmpty() {
        return readings.isEmpty();
    }

    /** Sort readings by timestamp (call before analysis). */
    public void sortByTime() {
        readings.sort((a, b) -> Long.compare(a.timestamp, b.timestamp));
    }

    // --- Getters/Setters ---
    public List<Reading> getReadings() { return readings; }
    public Double getBaseline() { return baseline; }
    public void setBaseline(Double baseline) { this.baseline = baseline; }
    public long getWindowOpenTime() { return windowOpenTime; }
    public void setWindowOpenTime(long t) { this.windowOpenTime = t; }
    public long getWindowCloseTime() { return windowCloseTime; }
    public void setWindowCloseTime(long t) { this.windowCloseTime = t; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier tier) { this.dataTier = tier; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class Reading implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("timestamp")
        public long timestamp;

        @JsonProperty("value")
        public double value;

        @JsonProperty("source")
        public String source; // "CGM", "SMBG"

        public Reading() {}

        public Reading(long timestamp, double value, String source) {
            this.timestamp = timestamp;
            this.value = value;
            this.source = source;
        }
    }
}
