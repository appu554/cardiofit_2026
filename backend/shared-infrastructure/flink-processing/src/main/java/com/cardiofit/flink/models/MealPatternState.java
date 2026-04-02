package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Per-patient state for Module 10b weekly aggregation.
 *
 * Stores:
 * - 60-day rolling buffer of (sodium_mg, SBP_excursion) pairs for OLS regression
 * - Per-meal-time accumulator maps for weekly summary
 * - Food description → excursion accumulator for food impact ranking
 *
 * State TTL: 60 days (OnReadAndWrite + NeverReturnExpired).
 * Weekly timer fires every Monday 00:00 UTC.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealPatternState implements Serializable {
    private static final long serialVersionUID = 1L;
    public static final int SALT_BUFFER_MAX_DAYS = 60;
    public static final long WEEK_MS = 7L * 86_400_000L;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    // --- Salt sensitivity OLS buffer (60-day rolling) ---
    @JsonProperty("sodiumSBPPairs")
    private List<SodiumSBPPair> sodiumSBPPairs;

    // --- Weekly accumulators (reset after each weekly emit) ---
    @JsonProperty("weeklyMealRecords")
    private List<MealResponseRecord> weeklyMealRecords;

    // --- Timer state ---
    @JsonProperty("weeklyTimerRegistered")
    private boolean weeklyTimerRegistered;

    @JsonProperty("lastWeeklyEmitTimestamp")
    private long lastWeeklyEmitTimestamp;

    @JsonProperty("lastUpdated")
    private long lastUpdated;

    @JsonProperty("totalRecordsProcessed")
    private long totalRecordsProcessed;

    public MealPatternState() {
        this.sodiumSBPPairs = new ArrayList<>();
        this.weeklyMealRecords = new ArrayList<>();
    }

    public MealPatternState(String patientId) {
        this();
        this.patientId = patientId;
    }

    /**
     * Add a MealResponseRecord to the weekly buffer.
     * Also add to salt sensitivity buffer if sodium + SBP excursion both present.
     */
    public void addMealRecord(MealResponseRecord record) {
        weeklyMealRecords.add(record);
        totalRecordsProcessed++;

        // Add to salt sensitivity buffer if both values present
        if (record.getSodiumMg() != null && record.getSbpExcursion() != null) {
            sodiumSBPPairs.add(new SodiumSBPPair(
                record.getSodiumMg(),
                record.getSbpExcursion(),
                record.getMealTimestamp()
            ));
        }

        // Evict pairs older than 60 days
        long cutoff = record.getMealTimestamp() - (SALT_BUFFER_MAX_DAYS * 86_400_000L);
        sodiumSBPPairs.removeIf(p -> p.timestamp < cutoff);
    }

    /**
     * Drain weekly records (returns list and clears buffer).
     */
    public List<MealResponseRecord> drainWeeklyRecords() {
        List<MealResponseRecord> drained = new ArrayList<>(weeklyMealRecords);
        weeklyMealRecords.clear();
        return drained;
    }

    // --- Getters/Setters ---
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier v) { this.dataTier = v; }
    public List<SodiumSBPPair> getSodiumSBPPairs() { return sodiumSBPPairs; }
    public List<MealResponseRecord> getWeeklyMealRecords() { return weeklyMealRecords; }
    public boolean isWeeklyTimerRegistered() { return weeklyTimerRegistered; }
    public void setWeeklyTimerRegistered(boolean v) { this.weeklyTimerRegistered = v; }
    public long getLastWeeklyEmitTimestamp() { return lastWeeklyEmitTimestamp; }
    public void setLastWeeklyEmitTimestamp(long v) { this.lastWeeklyEmitTimestamp = v; }
    public long getLastUpdated() { return lastUpdated; }
    public void setLastUpdated(long v) { this.lastUpdated = v; }
    public long getTotalRecordsProcessed() { return totalRecordsProcessed; }

    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class SodiumSBPPair implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("sodiumMg")
        public double sodiumMg;

        @JsonProperty("sbpExcursion")
        public double sbpExcursion;

        @JsonProperty("timestamp")
        public long timestamp;

        public SodiumSBPPair() {}

        public SodiumSBPPair(double sodiumMg, double sbpExcursion, long timestamp) {
            this.sodiumMg = sodiumMg;
            this.sbpExcursion = sbpExcursion;
            this.timestamp = timestamp;
        }
    }
}
