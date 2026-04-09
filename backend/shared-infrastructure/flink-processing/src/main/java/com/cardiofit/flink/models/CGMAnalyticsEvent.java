package com.cardiofit.flink.models;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Output event for CGM analytics computation.
 * Contains TIR/TBR/TAR metrics, GMI, GRI, alerts, and AGP percentiles.
 *
 * Uses HashMap (not EnumMap) for Flink Kryo serialization compatibility.
 * Constructed via Builder pattern.
 */
public class CGMAnalyticsEvent implements Serializable {
    private static final long serialVersionUID = 1L;

    // --- Identity ---
    private String patientId;
    private String correlationId;
    private long computedAt;
    private String reportType; // INCREMENTAL, DAILY_SUMMARY, PERIOD_REPORT

    // --- Data Quality ---
    private double coveragePct;
    private int totalReadings;
    private int windowDays;
    private boolean sufficientData;
    private String confidenceLevel; // HIGH, MODERATE, LOW

    // --- Core Metrics ---
    private double meanGlucose;
    private double sdGlucose;
    private double cvPct;
    private boolean glucoseStable; // cv <= 36%

    // --- Time in Range ---
    private double tirPct;
    private double tbrL1Pct;
    private double tbrL2Pct;
    private double tarL1Pct;
    private double tarL2Pct;

    // --- Derived ---
    private double gmi;
    private double gmiHba1cDiscrepancy;
    private double gri;
    private String griZone; // A, B, C, D, E

    // --- Alerts ---
    private boolean sustainedHypoDetected;
    private boolean sustainedSevereHypoDetected;
    private boolean sustainedHyperDetected;
    private boolean nocturnalHypoDetected;
    private boolean rapidRiseDetected;
    private boolean rapidFallDetected;

    // --- AGP ---
    private HashMap<Integer, double[]> agpPercentiles; // percentile -> 48 half-hour buckets

    private CGMAnalyticsEvent() {
        this.agpPercentiles = new HashMap<>();
    }

    // --- Getters ---
    public String getPatientId() { return patientId; }
    public String getCorrelationId() { return correlationId; }
    public long getComputedAt() { return computedAt; }
    public String getReportType() { return reportType; }
    public double getCoveragePct() { return coveragePct; }
    public int getTotalReadings() { return totalReadings; }
    public int getWindowDays() { return windowDays; }
    public boolean isSufficientData() { return sufficientData; }
    public String getConfidenceLevel() { return confidenceLevel; }
    public double getMeanGlucose() { return meanGlucose; }
    public double getSdGlucose() { return sdGlucose; }
    public double getCvPct() { return cvPct; }
    public boolean isGlucoseStable() { return glucoseStable; }
    public double getTirPct() { return tirPct; }
    public double getTbrL1Pct() { return tbrL1Pct; }
    public double getTbrL2Pct() { return tbrL2Pct; }
    public double getTarL1Pct() { return tarL1Pct; }
    public double getTarL2Pct() { return tarL2Pct; }
    public double getGmi() { return gmi; }
    public double getGmiHba1cDiscrepancy() { return gmiHba1cDiscrepancy; }
    public double getGri() { return gri; }
    public String getGriZone() { return griZone; }
    public boolean isSustainedHypoDetected() { return sustainedHypoDetected; }
    public boolean isSustainedSevereHypoDetected() { return sustainedSevereHypoDetected; }
    public boolean isSustainedHyperDetected() { return sustainedHyperDetected; }
    public boolean isNocturnalHypoDetected() { return nocturnalHypoDetected; }
    public boolean isRapidRiseDetected() { return rapidRiseDetected; }
    public boolean isRapidFallDetected() { return rapidFallDetected; }
    public Map<Integer, double[]> getAgpPercentiles() { return agpPercentiles; }

    // --- Builder ---
    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private final CGMAnalyticsEvent event;

        private Builder() {
            this.event = new CGMAnalyticsEvent();
        }

        // Identity
        public Builder patientId(String v) { event.patientId = v; return this; }
        public Builder correlationId(String v) { event.correlationId = v; return this; }
        public Builder computedAt(long v) { event.computedAt = v; return this; }
        public Builder reportType(String v) { event.reportType = v; return this; }

        // Data Quality
        public Builder coveragePct(double v) { event.coveragePct = v; return this; }
        public Builder totalReadings(int v) { event.totalReadings = v; return this; }
        public Builder windowDays(int v) { event.windowDays = v; return this; }
        public Builder sufficientData(boolean v) { event.sufficientData = v; return this; }
        public Builder confidenceLevel(String v) { event.confidenceLevel = v; return this; }

        // Core Metrics
        public Builder meanGlucose(double v) { event.meanGlucose = v; return this; }
        public Builder sdGlucose(double v) { event.sdGlucose = v; return this; }
        public Builder cvPct(double v) { event.cvPct = v; return this; }
        public Builder glucoseStable(boolean v) { event.glucoseStable = v; return this; }

        // Time in Range
        public Builder tirPct(double v) { event.tirPct = v; return this; }
        public Builder tbrL1Pct(double v) { event.tbrL1Pct = v; return this; }
        public Builder tbrL2Pct(double v) { event.tbrL2Pct = v; return this; }
        public Builder tarL1Pct(double v) { event.tarL1Pct = v; return this; }
        public Builder tarL2Pct(double v) { event.tarL2Pct = v; return this; }

        // Derived
        public Builder gmi(double v) { event.gmi = v; return this; }
        public Builder gmiHba1cDiscrepancy(double v) { event.gmiHba1cDiscrepancy = v; return this; }
        public Builder gri(double v) { event.gri = v; return this; }
        public Builder griZone(String v) { event.griZone = v; return this; }

        // Alerts
        public Builder sustainedHypoDetected(boolean v) { event.sustainedHypoDetected = v; return this; }
        public Builder sustainedSevereHypoDetected(boolean v) { event.sustainedSevereHypoDetected = v; return this; }
        public Builder sustainedHyperDetected(boolean v) { event.sustainedHyperDetected = v; return this; }
        public Builder nocturnalHypoDetected(boolean v) { event.nocturnalHypoDetected = v; return this; }
        public Builder rapidRiseDetected(boolean v) { event.rapidRiseDetected = v; return this; }
        public Builder rapidFallDetected(boolean v) { event.rapidFallDetected = v; return this; }

        // AGP
        public Builder agpPercentiles(HashMap<Integer, double[]> v) { event.agpPercentiles = v; return this; }

        public CGMAnalyticsEvent build() {
            return event;
        }
    }
}
