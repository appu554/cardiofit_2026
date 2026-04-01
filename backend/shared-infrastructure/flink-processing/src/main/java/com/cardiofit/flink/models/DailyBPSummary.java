package com.cardiofit.flink.models;

import java.io.Serializable;

/**
 * Aggregated BP data for a single calendar day.
 * Stored in the 30-day rolling window within PatientBPState.
 *
 * A day's summary accumulates as readings arrive. On each new reading,
 * the running averages are updated incrementally.
 */
public class DailyBPSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    private String dateKey;           // "YYYY-MM-DD"
    private double sumSBP;
    private double sumDBP;
    private int readingCount;

    // Time-context-specific averages for surge/dip computation
    private double sumMorningSBP;
    private int morningCount;
    private double sumEveningSBP;
    private int eveningCount;
    private double sumNocturnalSBP;
    private int nocturnalCount;
    private double sumDaytimeSBP;
    private int daytimeCount;

    // Clinic vs home for white-coat/masked detection
    private double sumClinicSBP;
    private int clinicCount;
    private double sumHomeSBP;
    private int homeCount;

    // Within-day variance (online computation via sum of squares)
    private double sumSBPSquared;

    // Extremes for crisis detection
    private double maxSBP;
    private double maxDBP;
    private double minSBP;

    public DailyBPSummary() {}

    public DailyBPSummary(String dateKey) {
        this.dateKey = dateKey;
        this.maxSBP = Double.MIN_VALUE;
        this.maxDBP = Double.MIN_VALUE;
        this.minSBP = Double.MAX_VALUE;
    }

    /**
     * Add a reading to this day's summary. Incremental update.
     */
    public void addReading(BPReading reading) {
        double sbp = reading.getSystolic();
        double dbp = reading.getDiastolic();

        sumSBP += sbp;
        sumSBPSquared += sbp * sbp;
        sumDBP += dbp;
        readingCount++;

        if (sbp > maxSBP) maxSBP = sbp;
        if (dbp > maxDBP) maxDBP = dbp;
        if (sbp < minSBP) minSBP = sbp;

        // Time-context-specific accumulation
        TimeContext tc = reading.resolveTimeContext();
        switch (tc) {
            case MORNING  -> { sumMorningSBP += sbp; morningCount++; }
            case EVENING  -> { sumEveningSBP += sbp; eveningCount++; }
            case NIGHT    -> { sumNocturnalSBP += sbp; nocturnalCount++; }
            default -> {}
        }
        if (tc.isDaytime()) { sumDaytimeSBP += sbp; daytimeCount++; }

        // Source-specific accumulation
        BPSource src = reading.resolveSource();
        if (src == BPSource.CLINIC) { sumClinicSBP += sbp; clinicCount++; }
        else if (src == BPSource.HOME_CUFF) { sumHomeSBP += sbp; homeCount++; }
    }

    // — Computed averages —
    public double getAvgSBP() { return readingCount > 0 ? sumSBP / readingCount : 0; }
    public double getAvgDBP() { return readingCount > 0 ? sumDBP / readingCount : 0; }
    public Double getMorningAvgSBP() { return morningCount > 0 ? sumMorningSBP / morningCount : null; }
    public Double getEveningAvgSBP() { return eveningCount > 0 ? sumEveningSBP / eveningCount : null; }
    public Double getNocturnalAvgSBP() { return nocturnalCount > 0 ? sumNocturnalSBP / nocturnalCount : null; }
    public Double getDaytimeAvgSBP() { return daytimeCount > 0 ? sumDaytimeSBP / daytimeCount : null; }
    public Double getClinicAvgSBP() { return clinicCount > 0 ? sumClinicSBP / clinicCount : null; }
    public Double getHomeAvgSBP() { return homeCount > 0 ? sumHomeSBP / homeCount : null; }

    /**
     * Within-day SD of systolic readings, available when readingCount > 2.
     * Uses online variance formula: Var = (sumSQ/n - mean²) * n/(n-1).
     * Returns null if fewer than 3 readings (SD meaningless with 1-2 data points).
     */
    public Double getWithinDaySdSBP() {
        if (readingCount < 3) return null;
        double mean = sumSBP / readingCount;
        double variance = (sumSBPSquared / readingCount - mean * mean) * readingCount / (readingCount - 1);
        return variance > 0 ? Math.sqrt(variance) : 0.0;
    }

    // — Standard getters —
    public String getDateKey() { return dateKey; }
    public void setDateKey(String dateKey) { this.dateKey = dateKey; }
    public int getReadingCount() { return readingCount; }
    public double getMaxSBP() { return maxSBP; }
    public double getMaxDBP() { return maxDBP; }
    public double getMinSBP() { return minSBP; }
    public int getMorningCount() { return morningCount; }
    public int getEveningCount() { return eveningCount; }
    public int getNocturnalCount() { return nocturnalCount; }
    public int getDaytimeCount() { return daytimeCount; }
    public int getClinicCount() { return clinicCount; }
    public int getHomeCount() { return homeCount; }

    // Setters for all fields needed for serialization
    public void setSumSBP(double v) { sumSBP = v; }
    public double getSumSBP() { return sumSBP; }
    public void setSumDBP(double v) { sumDBP = v; }
    public double getSumDBP() { return sumDBP; }
    public void setReadingCount(int v) { readingCount = v; }
    public void setSumMorningSBP(double v) { sumMorningSBP = v; }
    public double getSumMorningSBP() { return sumMorningSBP; }
    public void setMorningCount(int v) { morningCount = v; }
    public void setSumEveningSBP(double v) { sumEveningSBP = v; }
    public double getSumEveningSBP() { return sumEveningSBP; }
    public void setEveningCount(int v) { eveningCount = v; }
    public void setSumNocturnalSBP(double v) { sumNocturnalSBP = v; }
    public double getSumNocturnalSBP() { return sumNocturnalSBP; }
    public void setNocturnalCount(int v) { nocturnalCount = v; }
    public void setSumDaytimeSBP(double v) { sumDaytimeSBP = v; }
    public double getSumDaytimeSBP() { return sumDaytimeSBP; }
    public void setDaytimeCount(int v) { daytimeCount = v; }
    public void setSumClinicSBP(double v) { sumClinicSBP = v; }
    public double getSumClinicSBP() { return sumClinicSBP; }
    public void setClinicCount(int v) { clinicCount = v; }
    public void setSumHomeSBP(double v) { sumHomeSBP = v; }
    public double getSumHomeSBP() { return sumHomeSBP; }
    public void setHomeCount(int v) { homeCount = v; }
    public void setMaxSBP(double v) { maxSBP = v; }
    public void setMaxDBP(double v) { maxDBP = v; }
    public void setMinSBP(double v) { minSBP = v; }
    public void setSumSBPSquared(double v) { sumSBPSquared = v; }
    public double getSumSBPSquared() { return sumSBPSquared; }
}
