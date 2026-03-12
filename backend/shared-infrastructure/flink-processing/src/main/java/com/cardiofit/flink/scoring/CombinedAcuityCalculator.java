package com.cardiofit.flink.scoring;

import com.cardiofit.flink.models.PatientSnapshot;

import java.io.Serializable;
import java.util.Map;

/**
 * Combined Acuity Score Calculator
 *
 * Produces a multi-dimensional acuity assessment by combining:
 * - NEWS2 Score (70% weight): Physiological acuity based on vital signs
 * - Metabolic Acuity Score (30% weight): Cardiometabolic risk based on metabolic syndrome
 *
 * Formula: Combined Acuity = (0.7 × NEWS2) + (0.3 × Metabolic Acuity)
 *
 * This provides a more comprehensive clinical picture than NEWS2 alone:
 * - NEWS2 captures acute physiological deterioration
 * - Metabolic captures chronic disease burden and long-term risk
 *
 * Combined Acuity Levels:
 * - LOW: <2.0 - Routine monitoring
 * - MEDIUM: 2.0-4.9 - Increased monitoring frequency
 * - HIGH: 5.0-6.9 - Urgent clinical review
 * - CRITICAL: ≥7.0 - Emergency response
 *
 * Reference: MODULE2_ADVANCED_ENHANCEMENTS.md lines 96-112
 */
public class CombinedAcuityCalculator {

    // Weighting factors (per spec line 103)
    private static final double NEWS2_WEIGHT = 0.7;
    private static final double METABOLIC_WEIGHT = 0.3;

    // Acuity level thresholds
    private static final double CRITICAL_THRESHOLD = 7.0;
    private static final double HIGH_THRESHOLD = 5.0;
    private static final double MEDIUM_THRESHOLD = 2.0;

    /**
     * Calculate combined acuity score
     *
     * @param news2Score NEWS2 score (0-20 scale)
     * @param metabolicAcuityScore Metabolic acuity score (0-5 scale)
     * @return Combined acuity score result
     */
    public static CombinedAcuityScore calculate(
            NEWS2Calculator.NEWS2Score news2Score,
            MetabolicAcuityCalculator.MetabolicAcuityScore metabolicAcuityScore) {

        CombinedAcuityScore combined = new CombinedAcuityScore();
        combined.setCalculationTimestamp(System.currentTimeMillis());

        // Extract component scores
        int news2Value = news2Score != null ? news2Score.getTotalScore() : 0;
        double metabolicValue = metabolicAcuityScore != null ? metabolicAcuityScore.getScore() : 0.0;

        // Store component scores
        combined.setNews2Score(news2Value);
        combined.setNews2Interpretation(news2Score != null ? news2Score.getRiskLevel() : "UNKNOWN");
        combined.setMetabolicAcuityScore(metabolicValue);
        combined.setMetabolicInterpretation(metabolicAcuityScore != null ?
            metabolicAcuityScore.getRiskLevel() : "UNKNOWN");

        // Calculate weighted combination and round to 1 decimal place to avoid floating point artifacts
        double combinedScore = (NEWS2_WEIGHT * news2Value) + (METABOLIC_WEIGHT * metabolicValue);
        double roundedScore = Math.round(combinedScore * 10.0) / 10.0;
        combined.setCombinedAcuityScore(roundedScore);

        // Determine overall acuity level
        String acuityLevel;
        String monitoringRecommendation;

        if (combinedScore >= CRITICAL_THRESHOLD) {
            acuityLevel = "CRITICAL";
            monitoringRecommendation = "Emergency response required. Immediate clinical assessment. Continuous monitoring.";
        } else if (combinedScore >= HIGH_THRESHOLD) {
            acuityLevel = "HIGH";
            monitoringRecommendation = "Urgent clinical review within 30 minutes. Vital signs every 15-30 minutes.";
        } else if (combinedScore >= MEDIUM_THRESHOLD) {
            acuityLevel = "MEDIUM";
            monitoringRecommendation = "Increased monitoring frequency. Vital signs every 1-2 hours. Clinical review within 4 hours.";
        } else {
            acuityLevel = "LOW";
            monitoringRecommendation = "Routine monitoring. Vital signs every 4-6 hours as per ward protocol.";
        }

        combined.setAcuityLevel(acuityLevel);
        combined.setMonitoringRecommendation(monitoringRecommendation);

        // Generate interpretation
        combined.setInterpretation(generateInterpretation(
            news2Value, metabolicValue, combinedScore, acuityLevel));

        return combined;
    }

    /**
     * Overloaded method that calculates metabolic acuity if not provided
     *
     * @param news2Score NEWS2 score object
     * @param snapshot Patient snapshot
     * @param vitals Vital signs
     * @param labs Lab results
     * @return Combined acuity score result
     */
    public static CombinedAcuityScore calculate(
            NEWS2Calculator.NEWS2Score news2Score,
            PatientSnapshot snapshot,
            Map<String, Object> vitals,
            Map<String, Object> labs) {

        // Calculate metabolic acuity if not already done
        MetabolicAcuityCalculator.MetabolicAcuityScore metabolicScore =
            MetabolicAcuityCalculator.calculate(snapshot, vitals, labs);

        return calculate(news2Score, metabolicScore);
    }

    /**
     * Generate clinical interpretation of combined acuity
     */
    private static String generateInterpretation(
            int news2, double metabolic, double combined, String level) {

        StringBuilder interpretation = new StringBuilder();

        interpretation.append(String.format("Combined Acuity: %.1f (%s). ", combined, level));

        // Interpret component contributions
        if (news2 >= 7) {
            interpretation.append("High physiological acuity (NEWS2=").append(news2).append("). ");
        } else if (news2 >= 5) {
            interpretation.append("Moderate physiological acuity (NEWS2=").append(news2).append("). ");
        }

        if (metabolic >= 3.0) {
            interpretation.append("High metabolic risk (").append((int)metabolic).append("/5 components). ");
        } else if (metabolic >= 2.0) {
            interpretation.append("Moderate metabolic risk (").append((int)metabolic).append("/5 components). ");
        }

        // Identify dominant component
        double news2Contribution = NEWS2_WEIGHT * news2;
        double metabolicContribution = METABOLIC_WEIGHT * metabolic;

        if (news2Contribution > metabolicContribution * 1.5) {
            interpretation.append("Primarily driven by acute physiological changes.");
        } else if (metabolicContribution > news2Contribution * 1.5) {
            interpretation.append("Primarily driven by chronic metabolic burden.");
        } else {
            interpretation.append("Balanced acute and chronic risk factors.");
        }

        return interpretation.toString();
    }

    /**
     * Combined Acuity Score Result Class
     */
    public static class CombinedAcuityScore implements Serializable {
        private static final long serialVersionUID = 1L;

        // Component scores
        private int news2Score; // 0-20
        private String news2Interpretation;
        private double metabolicAcuityScore; // 0-5
        private String metabolicInterpretation;

        // Combined result
        private double combinedAcuityScore; // Weighted combination
        private String acuityLevel; // LOW, MEDIUM, HIGH, CRITICAL
        private String interpretation;
        private String monitoringRecommendation;
        private long calculationTimestamp;

        // Getters and Setters

        public int getNews2Score() {
            return news2Score;
        }

        public void setNews2Score(int news2Score) {
            this.news2Score = news2Score;
        }

        public String getNews2Interpretation() {
            return news2Interpretation;
        }

        public void setNews2Interpretation(String news2Interpretation) {
            this.news2Interpretation = news2Interpretation;
        }

        public double getMetabolicAcuityScore() {
            return metabolicAcuityScore;
        }

        public void setMetabolicAcuityScore(double metabolicAcuityScore) {
            this.metabolicAcuityScore = metabolicAcuityScore;
        }

        public String getMetabolicInterpretation() {
            return metabolicInterpretation;
        }

        public void setMetabolicInterpretation(String metabolicInterpretation) {
            this.metabolicInterpretation = metabolicInterpretation;
        }

        public double getCombinedAcuityScore() {
            return combinedAcuityScore;
        }

        public void setCombinedAcuityScore(double combinedAcuityScore) {
            this.combinedAcuityScore = combinedAcuityScore;
        }

        public String getAcuityLevel() {
            return acuityLevel;
        }

        public void setAcuityLevel(String acuityLevel) {
            this.acuityLevel = acuityLevel;
        }

        public String getInterpretation() {
            return interpretation;
        }

        public void setInterpretation(String interpretation) {
            this.interpretation = interpretation;
        }

        public String getMonitoringRecommendation() {
            return monitoringRecommendation;
        }

        public void setMonitoringRecommendation(String monitoringRecommendation) {
            this.monitoringRecommendation = monitoringRecommendation;
        }

        public long getCalculationTimestamp() {
            return calculationTimestamp;
        }

        public void setCalculationTimestamp(long calculationTimestamp) {
            this.calculationTimestamp = calculationTimestamp;
        }

        @Override
        public String toString() {
            return "CombinedAcuityScore{" +
                    "combinedScore=" + combinedAcuityScore +
                    ", level='" + acuityLevel + '\'' +
                    ", NEWS2=" + news2Score +
                    ", metabolic=" + metabolicAcuityScore +
                    '}';
        }
    }
}
