package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Pre-meal and post-meal BP reading pair for a single meal session.
 *
 * Pre-meal BP: retroactive — most recent BP reading within 60 min BEFORE meal.
 * Post-meal BP: first BP reading within 4h AFTER meal.
 * BP excursion = post_sbp - pre_sbp (can be negative).
 *
 * Window duration: 4h (longer than glucose window).
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class BPWindow implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("preMealSBP")
    private Double preMealSBP;

    @JsonProperty("preMealDBP")
    private Double preMealDBP;

    @JsonProperty("preMealTimestamp")
    private Long preMealTimestamp;

    @JsonProperty("postMealSBP")
    private Double postMealSBP;

    @JsonProperty("postMealDBP")
    private Double postMealDBP;

    @JsonProperty("postMealTimestamp")
    private Long postMealTimestamp;

    public BPWindow() {}

    public boolean hasPreMeal() {
        return preMealSBP != null;
    }

    public boolean hasPostMeal() {
        return postMealSBP != null;
    }

    public boolean isComplete() {
        return hasPreMeal() && hasPostMeal();
    }

    /**
     * SBP excursion = post - pre. Null if either reading missing.
     */
    public Double getSBPExcursion() {
        if (!isComplete()) return null;
        return postMealSBP - preMealSBP;
    }

    // --- Getters/Setters ---
    public Double getPreMealSBP() { return preMealSBP; }
    public void setPreMealSBP(Double v) { this.preMealSBP = v; }
    public Double getPreMealDBP() { return preMealDBP; }
    public void setPreMealDBP(Double v) { this.preMealDBP = v; }
    public Long getPreMealTimestamp() { return preMealTimestamp; }
    public void setPreMealTimestamp(Long t) { this.preMealTimestamp = t; }
    public Double getPostMealSBP() { return postMealSBP; }
    public void setPostMealSBP(Double v) { this.postMealSBP = v; }
    public Double getPostMealDBP() { return postMealDBP; }
    public void setPostMealDBP(Double v) { this.postMealDBP = v; }
    public Long getPostMealTimestamp() { return postMealTimestamp; }
    public void setPostMealTimestamp(Long t) { this.postMealTimestamp = t; }
}
