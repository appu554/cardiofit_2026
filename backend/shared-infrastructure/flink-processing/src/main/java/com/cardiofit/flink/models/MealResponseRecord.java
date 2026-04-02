package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;

/**
 * Output record from Module 10: per-meal glucose and BP response.
 * 28 fields across all tiers. Tier 3 patients will have null CGM-specific fields.
 * Emitted to flink.meal-response topic.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class MealResponseRecord implements Serializable {
    private static final long serialVersionUID = 1L;

    // --- Identity (4 fields) ---
    @JsonProperty("recordId")
    private String recordId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("mealEventId")
    private String mealEventId;

    @JsonProperty("correlationId")
    private String correlationId;

    // --- Meal metadata (5 fields) ---
    @JsonProperty("mealTimestamp")
    private long mealTimestamp;

    @JsonProperty("mealTimeCategory")
    private MealTimeCategory mealTimeCategory;

    @JsonProperty("carbGrams")
    private Double carbGrams;

    @JsonProperty("proteinGrams")
    private Double proteinGrams;

    @JsonProperty("sodiumMg")
    private Double sodiumMg;

    // --- Glucose features (8 fields) — Tier 1/2 only ---
    @JsonProperty("glucoseBaseline")
    private Double glucoseBaseline;

    @JsonProperty("glucosePeak")
    private Double glucosePeak;

    @JsonProperty("glucoseExcursion")
    private Double glucoseExcursion;

    @JsonProperty("timeTopeakMin")
    private Double timeToPeakMin;

    @JsonProperty("iAUC")
    private Double iAUC;

    @JsonProperty("recoveryTimeMin")
    private Double recoveryTimeMin;

    @JsonProperty("curveShape")
    private CurveShape curveShape;

    @JsonProperty("glucoseReadingCount")
    private int glucoseReadingCount;

    // --- BP features (4 fields) ---
    @JsonProperty("preMealSBP")
    private Double preMealSBP;

    @JsonProperty("postMealSBP")
    private Double postMealSBP;

    @JsonProperty("sbpExcursion")
    private Double sbpExcursion;

    @JsonProperty("bpComplete")
    private boolean bpComplete;

    // --- Processing metadata (7 fields) ---
    @JsonProperty("dataTier")
    private DataTier dataTier;

    @JsonProperty("windowDurationMs")
    private long windowDurationMs;

    @JsonProperty("overlapping")
    private boolean overlapping;

    @JsonProperty("processingTimestamp")
    private long processingTimestamp;

    @JsonProperty("mealPayload")
    private Map<String, Object> mealPayload;

    @JsonProperty("qualityScore")
    private double qualityScore;

    @JsonProperty("version")
    private String version;

    public MealResponseRecord() {
        this.version = "1.0";
        this.processingTimestamp = System.currentTimeMillis();
    }

    public static Builder builder() { return new Builder(); }

    public static class Builder {
        private final MealResponseRecord r = new MealResponseRecord();

        public Builder recordId(String v) { r.recordId = v; return this; }
        public Builder patientId(String v) { r.patientId = v; return this; }
        public Builder mealEventId(String v) { r.mealEventId = v; return this; }
        public Builder correlationId(String v) { r.correlationId = v; return this; }
        public Builder mealTimestamp(long v) { r.mealTimestamp = v; return this; }
        public Builder mealTimeCategory(MealTimeCategory v) { r.mealTimeCategory = v; return this; }
        public Builder carbGrams(Double v) { r.carbGrams = v; return this; }
        public Builder proteinGrams(Double v) { r.proteinGrams = v; return this; }
        public Builder sodiumMg(Double v) { r.sodiumMg = v; return this; }
        public Builder glucoseBaseline(Double v) { r.glucoseBaseline = v; return this; }
        public Builder glucosePeak(Double v) { r.glucosePeak = v; return this; }
        public Builder glucoseExcursion(Double v) { r.glucoseExcursion = v; return this; }
        public Builder timeToPeakMin(Double v) { r.timeToPeakMin = v; return this; }
        public Builder iAUC(Double v) { r.iAUC = v; return this; }
        public Builder recoveryTimeMin(Double v) { r.recoveryTimeMin = v; return this; }
        public Builder curveShape(CurveShape v) { r.curveShape = v; return this; }
        public Builder glucoseReadingCount(int v) { r.glucoseReadingCount = v; return this; }
        public Builder preMealSBP(Double v) { r.preMealSBP = v; return this; }
        public Builder postMealSBP(Double v) { r.postMealSBP = v; return this; }
        public Builder sbpExcursion(Double v) { r.sbpExcursion = v; return this; }
        public Builder bpComplete(boolean v) { r.bpComplete = v; return this; }
        public Builder dataTier(DataTier v) { r.dataTier = v; return this; }
        public Builder windowDurationMs(long v) { r.windowDurationMs = v; return this; }
        public Builder overlapping(boolean v) { r.overlapping = v; return this; }
        public Builder mealPayload(Map<String, Object> v) { r.mealPayload = v; return this; }
        public Builder qualityScore(double v) { r.qualityScore = v; return this; }
        public MealResponseRecord build() { return r; }
    }

    // --- Getters ---
    public String getRecordId() { return recordId; }
    public String getPatientId() { return patientId; }
    public String getMealEventId() { return mealEventId; }
    public String getCorrelationId() { return correlationId; }
    public long getMealTimestamp() { return mealTimestamp; }
    public MealTimeCategory getMealTimeCategory() { return mealTimeCategory; }
    public Double getCarbGrams() { return carbGrams; }
    public Double getProteinGrams() { return proteinGrams; }
    public Double getSodiumMg() { return sodiumMg; }
    public Double getGlucoseBaseline() { return glucoseBaseline; }
    public Double getGlucosePeak() { return glucosePeak; }
    public Double getGlucoseExcursion() { return glucoseExcursion; }
    public Double getTimeToPeakMin() { return timeToPeakMin; }
    public Double getIAUC() { return iAUC; }
    public Double getRecoveryTimeMin() { return recoveryTimeMin; }
    public CurveShape getCurveShape() { return curveShape; }
    public int getGlucoseReadingCount() { return glucoseReadingCount; }
    public Double getPreMealSBP() { return preMealSBP; }
    public Double getPostMealSBP() { return postMealSBP; }
    public Double getSbpExcursion() { return sbpExcursion; }
    public boolean isBpComplete() { return bpComplete; }
    public DataTier getDataTier() { return dataTier; }
    public long getWindowDurationMs() { return windowDurationMs; }
    public boolean isOverlapping() { return overlapping; }
    public long getProcessingTimestamp() { return processingTimestamp; }
    public Map<String, Object> getMealPayload() { return mealPayload; }
    public double getQualityScore() { return qualityScore; }
    public String getVersion() { return version; }

    // --- Setters for correlation ---
    public void setCorrelationId(String v) { this.correlationId = v; }
    public void setRecordId(String v) { this.recordId = v; }

    @Override
    public String toString() {
        return "MealResponseRecord{" +
            "patientId='" + patientId + '\'' +
            ", mealEventId='" + mealEventId + '\'' +
            ", tier=" + dataTier +
            ", glucose=" + glucoseExcursion +
            ", iAUC=" + iAUC +
            ", curve=" + curveShape +
            ", sbpExcursion=" + sbpExcursion +
            '}';
    }
}
