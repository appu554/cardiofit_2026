package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.UUID;

/**
 * Relapse risk assessment from Module 9 Phase 2 trajectory analysis.
 * Published to alerts.relapse-risk when risk exceeds MODERATE threshold (0.40).
 * Consumed by BCE (behavioral coaching engine) and KB-23 Decision Cards.
 *
 * Risk tiers (from plan P2.4):
 *   LOW:      0.00 - 0.39 → No alert emitted
 *   MODERATE: 0.40 - 0.69 → Recovery motivation phase, reduced targets
 *   HIGH:     0.70 - 1.00 → KB-23 Decision Card + physician outreach call
 *
 * Five trajectory features (7-day OLS regression slopes):
 *   1. stepsSlope       — wearable step count trend (weight 0.30)
 *   2. mealQualitySlope — carb/protein ratio trend (weight 0.20)
 *   3. responseLSlope   — app session response latency trend (weight 0.25)
 *   4. checkinSlope     — check-in field completeness trend (weight 0.15)
 *   5. proteinSlope     — protein adherence flag trend (weight 0.10)
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
public class RelapseRiskScore implements Serializable {
    private static final long serialVersionUID = 1L;

    public enum RiskTier {
        LOW,
        MODERATE,
        HIGH;

        public static RiskTier fromScore(double score) {
            if (score >= 0.70) return HIGH;
            if (score >= 0.40) return MODERATE;
            return LOW;
        }
    }

    @JsonProperty("alertId")
    private String alertId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("relapseRiskScore")
    private double relapseRiskScore;

    @JsonProperty("riskTier")
    private RiskTier riskTier;

    // --- Five trajectory slopes (negative = declining) ---

    @JsonProperty("stepsSlope")
    private double stepsSlope;

    @JsonProperty("mealQualitySlope")
    private double mealQualitySlope;

    @JsonProperty("responseLatencySlope")
    private double responseLatencySlope;

    @JsonProperty("checkinCompletenessSlope")
    private double checkinCompletenessSlope;

    @JsonProperty("proteinAdherenceSlope")
    private double proteinAdherenceSlope;

    // --- Context ---

    @JsonProperty("compositeEngagementScore")
    private double compositeEngagementScore;

    @JsonProperty("engagementLevel")
    private EngagementLevel engagementLevel;

    @JsonProperty("phenotype")
    private String phenotype;

    @JsonProperty("validHistoryDays")
    private int validHistoryDays;

    @JsonProperty("recommendedAction")
    private String recommendedAction;

    @JsonProperty("createdAt")
    private long createdAt;

    @JsonProperty("channel")
    private String channel;

    @JsonProperty("dataTier")
    private String dataTier;

    public RelapseRiskScore() {}

    public static RelapseRiskScore create(String patientId, double riskScore,
                                           double stepsSlope, double mealQualitySlope,
                                           double responseLatencySlope,
                                           double checkinCompletenessSlope,
                                           double proteinAdherenceSlope,
                                           double compositeEngagementScore,
                                           EngagementLevel level, String phenotype,
                                           int validHistoryDays) {
        RelapseRiskScore score = new RelapseRiskScore();
        score.alertId = UUID.randomUUID().toString();
        score.patientId = patientId;
        score.relapseRiskScore = riskScore;
        score.riskTier = RiskTier.fromScore(riskScore);
        score.stepsSlope = stepsSlope;
        score.mealQualitySlope = mealQualitySlope;
        score.responseLatencySlope = responseLatencySlope;
        score.checkinCompletenessSlope = checkinCompletenessSlope;
        score.proteinAdherenceSlope = proteinAdherenceSlope;
        score.compositeEngagementScore = compositeEngagementScore;
        score.engagementLevel = level;
        score.phenotype = phenotype;
        score.validHistoryDays = validHistoryDays;
        score.createdAt = System.currentTimeMillis();

        switch (score.riskTier) {
            case MODERATE:
                score.recommendedAction =
                    "Recovery motivation phase: reduce daily targets, empathetic messaging tone.";
                break;
            case HIGH:
                score.recommendedAction =
                    "Urgent: Generate KB-23 Decision Card. Schedule physician outreach call.";
                break;
            default:
                score.recommendedAction = null;
                break;
        }

        return score;
    }

    // --- Getters ---
    public String getAlertId() { return alertId; }
    public String getPatientId() { return patientId; }
    public double getRelapseRiskScore() { return relapseRiskScore; }
    public RiskTier getRiskTier() { return riskTier; }
    public double getStepsSlope() { return stepsSlope; }
    public double getMealQualitySlope() { return mealQualitySlope; }
    public double getResponseLatencySlope() { return responseLatencySlope; }
    public double getCheckinCompletenessSlope() { return checkinCompletenessSlope; }
    public double getProteinAdherenceSlope() { return proteinAdherenceSlope; }
    public double getCompositeEngagementScore() { return compositeEngagementScore; }
    public EngagementLevel getEngagementLevel() { return engagementLevel; }
    public String getPhenotype() { return phenotype; }
    public int getValidHistoryDays() { return validHistoryDays; }
    public String getRecommendedAction() { return recommendedAction; }
    public long getCreatedAt() { return createdAt; }
    public String getChannel() { return channel; }
    public void setChannel(String channel) { this.channel = channel; }
    public String getDataTier() { return dataTier; }
    public void setDataTier(String dataTier) { this.dataTier = dataTier; }
}
