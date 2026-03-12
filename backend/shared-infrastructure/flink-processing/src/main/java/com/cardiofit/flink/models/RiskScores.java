package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;

/**
 * Clinical risk scores for enriched events.
 */
public class RiskScores implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("sepsisScore")
    private Double sepsisScore; // 0.0-1.0

    @JsonProperty("deteriorationScore")
    private Double deteriorationScore; // 0.0-1.0

    @JsonProperty("readmissionRisk")
    private Double readmissionRisk; // 0.0-1.0

    public RiskScores() {
    }

    public RiskScores(Double sepsisScore, Double deteriorationScore, Double readmissionRisk) {
        this.sepsisScore = sepsisScore;
        this.deteriorationScore = deteriorationScore;
        this.readmissionRisk = readmissionRisk;
    }

    // Getters and setters
    public Double getSepsisScore() { return sepsisScore; }
    public void setSepsisScore(Double sepsisScore) { this.sepsisScore = sepsisScore; }

    public Double getDeteriorationScore() { return deteriorationScore; }
    public void setDeteriorationScore(Double deteriorationScore) {
        this.deteriorationScore = deteriorationScore;
    }

    public Double getReadmissionRisk() { return readmissionRisk; }
    public void setReadmissionRisk(Double readmissionRisk) {
        this.readmissionRisk = readmissionRisk;
    }

    @Override
    public String toString() {
        return "RiskScores{" +
                "sepsis=" + (sepsisScore != null ? String.format("%.2f", sepsisScore) : "N/A") +
                ", deterioration=" + (deteriorationScore != null ? String.format("%.2f", deteriorationScore) : "N/A") +
                ", readmission=" + (readmissionRisk != null ? String.format("%.2f", readmissionRisk) : "N/A") +
                '}';
    }
}
