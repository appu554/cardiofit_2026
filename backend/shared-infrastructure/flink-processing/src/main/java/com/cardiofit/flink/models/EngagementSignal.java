package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.Map;
import java.util.UUID;

/**
 * Daily per-patient engagement signal emitted by Module 9.
 * Published to flink.engagement-signals.
 * Consumed by KB-20 (patient profile), KB-26 (MHRI), KB-21 (behavioral intelligence), SPAN.
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
public class EngagementSignal implements Serializable {
    private static final long serialVersionUID = 1L;

    @JsonProperty("signalId")
    private String signalId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("compositeScore")
    private double compositeScore;

    @JsonProperty("engagementLevel")
    private EngagementLevel engagementLevel;

    @JsonProperty("signalDensities")
    private Map<SignalType, Double> signalDensities;

    @JsonProperty("phenotype")
    private String phenotype;

    @JsonProperty("previousScore")
    private Double previousScore;

    @JsonProperty("scoreDelta")
    private Double scoreDelta;

    @JsonProperty("consecutiveLowDays")
    private int consecutiveLowDays;

    @JsonProperty("computedAt")
    private long computedAt;

    @JsonProperty("correlationId")
    private String correlationId;

    @JsonProperty("channel")
    private String channel;

    @JsonProperty("dataTier")
    private String dataTier;

    @JsonProperty("relapseRiskScore")
    private Double relapseRiskScore;

    public EngagementSignal() {}

    public static EngagementSignal create(String patientId, double score,
                                           EngagementLevel level,
                                           Map<SignalType, Double> densities,
                                           String phenotype,
                                           Double previousScore,
                                           int consecutiveLowDays) {
        EngagementSignal signal = new EngagementSignal();
        signal.signalId = UUID.randomUUID().toString();
        signal.patientId = patientId;
        signal.compositeScore = score;
        signal.engagementLevel = level;
        signal.signalDensities = densities;
        signal.phenotype = phenotype;
        signal.previousScore = previousScore;
        signal.scoreDelta = (previousScore != null) ? score - previousScore : null;
        signal.consecutiveLowDays = consecutiveLowDays;
        signal.computedAt = System.currentTimeMillis();
        return signal;
    }

    // Getters
    public String getSignalId() { return signalId; }
    public String getPatientId() { return patientId; }
    public double getCompositeScore() { return compositeScore; }
    public EngagementLevel getEngagementLevel() { return engagementLevel; }
    public Map<SignalType, Double> getSignalDensities() { return signalDensities; }
    public String getPhenotype() { return phenotype; }
    public Double getPreviousScore() { return previousScore; }
    public Double getScoreDelta() { return scoreDelta; }
    public int getConsecutiveLowDays() { return consecutiveLowDays; }
    public long getComputedAt() { return computedAt; }
    public String getCorrelationId() { return correlationId; }
    public void setCorrelationId(String id) { this.correlationId = id; }
    public String getChannel() { return channel; }
    public void setChannel(String channel) { this.channel = channel; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
    public Double getRelapseRiskScore() { return relapseRiskScore; }
    public void setRelapseRiskScore(Double score) { this.relapseRiskScore = score; }
}
