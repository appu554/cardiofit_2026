package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * Output record from Module 10b: weekly meal pattern aggregation.
 * 21 fields. Emitted to flink.meal-patterns topic.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealPatternSummary implements Serializable {
    private static final long serialVersionUID = 1L;

    // --- Identity (3 fields) ---
    @JsonProperty("summaryId")
    private String summaryId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("correlationId")
    private String correlationId;

    // --- Time range (2 fields) ---
    @JsonProperty("periodStartMs")
    private long periodStartMs;

    @JsonProperty("periodEndMs")
    private long periodEndMs;

    // --- Aggregated glucose metrics (4 fields) ---
    @JsonProperty("meanIAUC")
    private Double meanIAUC;

    @JsonProperty("medianExcursion")
    private Double medianExcursion;

    @JsonProperty("meanTimeToPeakMin")
    private Double meanTimeToPeakMin;

    @JsonProperty("dominantCurveShape")
    private CurveShape dominantCurveShape;

    // --- Per-meal-time breakdown (1 field — map) ---
    @JsonProperty("mealTimeBreakdown")
    private Map<MealTimeCategory, MealTimeStats> mealTimeBreakdown;

    // --- Salt sensitivity (4 fields) ---
    @JsonProperty("saltSensitivityClass")
    private SaltSensitivityClass saltSensitivityClass;

    @JsonProperty("saltBeta")
    private Double saltBeta;

    @JsonProperty("saltRSquared")
    private Double saltRSquared;

    @JsonProperty("saltPairCount")
    private int saltPairCount;

    // --- Food impact ranking (1 field) ---
    @JsonProperty("topFoodsByExcursion")
    private List<FoodImpact> topFoodsByExcursion;

    // --- Processing metadata (6 fields) ---
    @JsonProperty("totalMealsInPeriod")
    private int totalMealsInPeriod;

    @JsonProperty("dataTier")
    private DataTier dataTier;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("mealsWithGlucose")
    private int mealsWithGlucose;

    @JsonProperty("version")
    private String version;

    public MealPatternSummary() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    // --- Getters/Setters ---
    public String getSummaryId() { return summaryId; }
    public void setSummaryId(String v) { this.summaryId = v; }
    public String getPatientId() { return patientId; }
    public void setPatientId(String v) { this.patientId = v; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String v) { this.correlationId = v; }
    public long getPeriodStartMs() { return periodStartMs; }
    public void setPeriodStartMs(long v) { this.periodStartMs = v; }
    public long getPeriodEndMs() { return periodEndMs; }
    public void setPeriodEndMs(long v) { this.periodEndMs = v; }
    public Double getMeanIAUC() { return meanIAUC; }
    public void setMeanIAUC(Double v) { this.meanIAUC = v; }
    public Double getMedianExcursion() { return medianExcursion; }
    public void setMedianExcursion(Double v) { this.medianExcursion = v; }
    public Double getMeanTimeToPeakMin() { return meanTimeToPeakMin; }
    public void setMeanTimeToPeakMin(Double v) { this.meanTimeToPeakMin = v; }
    public CurveShape getDominantCurveShape() { return dominantCurveShape; }
    public void setDominantCurveShape(CurveShape v) { this.dominantCurveShape = v; }
    public Map<MealTimeCategory, MealTimeStats> getMealTimeBreakdown() { return mealTimeBreakdown; }
    public void setMealTimeBreakdown(Map<MealTimeCategory, MealTimeStats> v) { this.mealTimeBreakdown = v; }
    public SaltSensitivityClass getSaltSensitivityClass() { return saltSensitivityClass; }
    public void setSaltSensitivityClass(SaltSensitivityClass v) { this.saltSensitivityClass = v; }
    public Double getSaltBeta() { return saltBeta; }
    public void setSaltBeta(Double v) { this.saltBeta = v; }
    public Double getSaltRSquared() { return saltRSquared; }
    public void setSaltRSquared(Double v) { this.saltRSquared = v; }
    public int getSaltPairCount() { return saltPairCount; }
    public void setSaltPairCount(int v) { this.saltPairCount = v; }
    public List<FoodImpact> getTopFoodsByExcursion() { return topFoodsByExcursion; }
    public void setTopFoodsByExcursion(List<FoodImpact> v) { this.topFoodsByExcursion = v; }
    public int getTotalMealsInPeriod() { return totalMealsInPeriod; }
    public void setTotalMealsInPeriod(int v) { this.totalMealsInPeriod = v; }
    public DataTier getDataTier() { return dataTier; }
    public void setDataTier(DataTier v) { this.dataTier = v; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public int getMealsWithGlucose() { return mealsWithGlucose; }
    public void setMealsWithGlucose(int v) { this.mealsWithGlucose = v; }
    public double getQualityScore() { return qualityScore; }
    public void setQualityScore(double v) { this.qualityScore = v; }
    public String getVersion() { return version; }

    /**
     * Per-meal-time aggregated stats.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class MealTimeStats implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("mealCount")
        public int mealCount;

        @JsonProperty("meanExcursion")
        public double meanExcursion;

        @JsonProperty("meanIAUC")
        public double meanIAUC;

        @JsonProperty("dominantCurve")
        public CurveShape dominantCurve;

        public MealTimeStats() {}
    }

    /**
     * Food impact entry for ranking.
     */
    @JsonIgnoreProperties(ignoreUnknown = true)
    public static class FoodImpact implements Serializable {
        private static final long serialVersionUID = 1L;

        @JsonProperty("foodDescription")
        public String foodDescription;

        @JsonProperty("mealCount")
        public int mealCount;

        @JsonProperty("meanExcursion")
        public double meanExcursion;

        @JsonProperty("meanIAUC")
        public double meanIAUC;

        public FoodImpact() {}

        public FoodImpact(String desc, int count, double excursion, double iauc) {
            this.foodDescription = desc;
            this.mealCount = count;
            this.meanExcursion = excursion;
            this.meanIAUC = iauc;
        }
    }
}
