package com.cardiofit.flink.models;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.io.Serializable;
import java.time.LocalDate;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * DailyRiskScore - Aggregate patient risk assessment over 24-hour window
 *
 * Combines three clinical risk domains into unified daily risk score:
 * - Vital Stability (40% weight): Physiological instability via vital sign abnormalities
 * - Lab Abnormalities (35% weight): Organ dysfunction via critical lab values
 * - Medication Complexity (25% weight): Polypharmacy + high-risk meds + adherence issues
 *
 * Clinical Use Cases:
 * - Population health: Identify highest-risk patients for proactive intervention
 * - Resource allocation: Prioritize nursing/physician time to unstable patients
 * - Quality metrics: Track overall unit acuity and deterioration trends
 * - Discharge planning: Quantify readmission risk for post-acute care planning
 *
 * Evidence Base:
 * - Rothman Index validation (Rothman et al., Critical Care Medicine 2013)
 * - Epic Deterioration Index (EDI) methodology (Escobar et al., JAMA 2020)
 * - NEWS2 composite scoring (Royal College of Physicians 2017)
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class DailyRiskScore implements Serializable {
    private static final long serialVersionUID = 1L;

    /**
     * Patient identifier
     */
    private String patientId;

    /**
     * Date for this risk score (start of 24-hour window)
     */
    private LocalDate date;

    /**
     * Window timestamps for debugging
     */
    private Long windowStart;
    private Long windowEnd;

    /**
     * Aggregate risk score (0-100)
     * 0-25: LOW risk (routine monitoring)
     * 25-50: MODERATE risk (enhanced monitoring)
     * 50-75: HIGH risk (frequent assessment, rapid response consideration)
     * 75-100: CRITICAL risk (ICU-level monitoring, immediate physician review)
     */
    private int aggregateRiskScore;

    /**
     * Categorical risk level based on aggregate score
     */
    private RiskLevel riskLevel;

    /**
     * Component scores (0-100 each, weighted for aggregate)
     */
    private int vitalStabilityScore;      // 40% weight in aggregate
    private int labAbnormalityScore;      // 35% weight in aggregate
    private int medicationComplexityScore; // 25% weight in aggregate

    /**
     * Event counts contributing to this score
     */
    private int vitalSignCount;
    private int labResultCount;
    private int medicationEventCount;

    /**
     * Clinical context for score interpretation
     */
    @Builder.Default
    private Map<String, Object> contributingFactors = new HashMap<>();

    /**
     * Actionable recommendations based on risk level
     */
    @Builder.Default
    private List<String> recommendations = new ArrayList<>();

    /**
     * Risk level categories aligned with clinical workflows
     */
    public enum RiskLevel {
        /**
         * LOW (0-24): Stable patient, routine monitoring sufficient
         * Typical: Post-op day 3+ with normal vitals, no complications
         */
        LOW,

        /**
         * MODERATE (25-49): Mild instability, enhanced monitoring recommended
         * Typical: New onset fever, single abnormal vital, minor lab abnormality
         */
        MODERATE,

        /**
         * HIGH (50-74): Significant instability, frequent reassessment required
         * Typical: Multiple vital abnormalities, organ dysfunction labs, rapid response criteria
         */
        HIGH,

        /**
         * CRITICAL (75-100): Severe instability, ICU-level care consideration
         * Typical: Hemodynamic instability, respiratory failure, multi-organ dysfunction
         */
        CRITICAL
    }

    /**
     * Calculate risk level from aggregate score
     *
     * @param score Aggregate risk score (0-100)
     * @return Corresponding risk level
     */
    public static RiskLevel calculateRiskLevel(int score) {
        if (score >= 75) return RiskLevel.CRITICAL;
        else if (score >= 50) return RiskLevel.HIGH;
        else if (score >= 25) return RiskLevel.MODERATE;
        else return RiskLevel.LOW;
    }

    /**
     * Get human-readable risk description
     *
     * @return Clinical interpretation of current risk level
     */
    public String getRiskDescription() {
        switch (riskLevel) {
            case CRITICAL:
                return String.format("CRITICAL RISK (Score: %d/100) - Immediate physician review required. " +
                        "Consider ICU-level monitoring and interdisciplinary team assessment.",
                        aggregateRiskScore);
            case HIGH:
                return String.format("HIGH RISK (Score: %d/100) - Enhanced monitoring protocol indicated. " +
                        "Increase vital sign frequency and consider rapid response team consultation.",
                        aggregateRiskScore);
            case MODERATE:
                return String.format("MODERATE RISK (Score: %d/100) - Standard monitoring with close trend observation. " +
                        "Continue current care plan with heightened vigilance.",
                        aggregateRiskScore);
            case LOW:
                return String.format("LOW RISK (Score: %d/100) - Stable condition. " +
                        "Routine monitoring sufficient, focus on discharge planning if appropriate.",
                        aggregateRiskScore);
            default:
                return String.format("UNKNOWN RISK (Score: %d/100)", aggregateRiskScore);
        }
    }

    /**
     * Get emoji indicator for quick visual assessment
     *
     * @return Emoji representing risk level
     */
    public String getRiskEmoji() {
        switch (riskLevel) {
            case CRITICAL: return "🔴";
            case HIGH: return "⚠️";
            case MODERATE: return "🟡";
            case LOW: return "🟢";
            default: return "⚪";
        }
    }

    /**
     * Check if this risk score warrants immediate clinical action
     *
     * @return true if HIGH or CRITICAL risk
     */
    public boolean requiresImmediateAction() {
        return riskLevel == RiskLevel.HIGH || riskLevel == RiskLevel.CRITICAL;
    }

    /**
     * Get component score breakdown as formatted string
     *
     * @return Human-readable breakdown of component contributions
     */
    public String getComponentBreakdown() {
        return String.format(
            "Vital Stability: %d/100 (40%% weight = %.1f pts)\n" +
            "Lab Abnormalities: %d/100 (35%% weight = %.1f pts)\n" +
            "Medication Complexity: %d/100 (25%% weight = %.1f pts)\n" +
            "--------------------\n" +
            "Aggregate Score: %d/100",
            vitalStabilityScore, vitalStabilityScore * 0.4,
            labAbnormalityScore, labAbnormalityScore * 0.35,
            medicationComplexityScore, medicationComplexityScore * 0.25,
            aggregateRiskScore
        );
    }

    /**
     * Add a contributing factor to the risk score context
     *
     * @param key Factor name (e.g., "abnormal_vitals", "critical_labs")
     * @param value Factor value (count, list, or description)
     */
    public void addContributingFactor(String key, Object value) {
        if (contributingFactors == null) {
            contributingFactors = new HashMap<>();
        }
        contributingFactors.put(key, value);
    }

    /**
     * Add a clinical recommendation
     *
     * @param recommendation Actionable clinical recommendation
     */
    public void addRecommendation(String recommendation) {
        if (recommendations == null) {
            recommendations = new ArrayList<>();
        }
        recommendations.add(recommendation);
    }
}
