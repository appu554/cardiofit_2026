package com.cardiofit.flink.models;

import com.cardiofit.flink.alerts.SmartAlertGenerator;
import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.neo4j.CohortInsights;
import com.cardiofit.flink.neo4j.SimilarPatient;
import com.cardiofit.flink.protocols.ProtocolMatcher;
import com.cardiofit.flink.recommendations.Recommendations;
import com.cardiofit.flink.scoring.ClinicalScoreCalculators;
import com.cardiofit.flink.scoring.CombinedAcuityCalculator;
import com.cardiofit.flink.scoring.ConfidenceScoreCalculator;
import com.cardiofit.flink.scoring.MetabolicAcuityCalculator;
import com.cardiofit.flink.scoring.NEWS2Calculator;

import java.io.Serializable;
import java.util.List;

/**
 * ClinicalIntelligence - Comprehensive Clinical Intelligence Bundle (Phase 1 + Phase 2)
 *
 * This class bundles all Phase 1 (P0 - Critical) and Phase 2 (P1 - Advanced) outputs:
 *
 * PHASE 1 COMPONENTS:
 * - Risk Assessment (cardiac, BP, vitals freshness, trends)
 * - NEWS2 Scoring (standardized acuity assessment)
 * - Metabolic Acuity Scoring (metabolic syndrome assessment)
 * - Combined Acuity Score (NEWS2 + metabolic weighted combination)
 * - Smart Alerts (time-based suppression, priority routing)
 * - Clinical Scores (Framingham, CHADS-VASc, qSOFA, Metabolic Syndrome)
 * - Confidence Scoring (explainable assessment reliability)
 *
 * PHASE 2 COMPONENTS:
 * - Clinical Protocol Matching (evidence-based care pathways)
 * - Similar Patient Analysis (outcome prediction from historical data)
 * - Cohort Analytics (population-level insights)
 * - Intelligent Recommendations (actionable clinical guidance)
 *
 * Used in Module2_Enhanced ComprehensiveEnrichmentFunction to carry
 * all clinical intelligence through the async pipeline.
 */
public class ClinicalIntelligence implements Serializable {
    private static final long serialVersionUID = 1L;

    // Phase 1 Component Outputs
    private EnhancedRiskIndicators.RiskAssessment riskAssessment;
    private NEWS2Calculator.NEWS2Score news2Score;
    private MetabolicAcuityCalculator.MetabolicAcuityScore metabolicAcuityScore;
    private CombinedAcuityCalculator.CombinedAcuityScore combinedAcuityScore;
    private List<SmartAlertGenerator.ClinicalAlert> alerts;
    private ClinicalScoreCalculators.FraminghamScore framinghamScore;
    private ClinicalScoreCalculators.CHADS2VAScScore chadsVascScore;
    private ClinicalScoreCalculators.qSOFAScore qsofaScore;
    private ClinicalScoreCalculators.MetabolicSyndromeScore metabolicSyndromeScore;
    private ConfidenceScoreCalculator.ConfidenceScore confidenceScore;

    // Phase 2 Component Outputs
    private List<ProtocolMatcher.Protocol> applicableProtocols;
    private List<SimilarPatient> similarPatients;
    private CohortInsights cohortInsights;
    private Recommendations recommendations;

    // Metadata
    private long calculationTimestamp;
    private String patientId;

    /**
     * Default constructor
     */
    public ClinicalIntelligence() {
        this.calculationTimestamp = System.currentTimeMillis();
    }

    /**
     * Full constructor with all Phase 1 components
     */
    public ClinicalIntelligence(
            EnhancedRiskIndicators.RiskAssessment riskAssessment,
            NEWS2Calculator.NEWS2Score news2Score,
            List<SmartAlertGenerator.ClinicalAlert> alerts,
            ClinicalScoreCalculators.FraminghamScore framinghamScore,
            ClinicalScoreCalculators.CHADS2VAScScore chadsVascScore,
            ClinicalScoreCalculators.qSOFAScore qsofaScore,
            ConfidenceScoreCalculator.ConfidenceScore confidenceScore) {
        this.riskAssessment = riskAssessment;
        this.news2Score = news2Score;
        this.alerts = alerts;
        this.framinghamScore = framinghamScore;
        this.chadsVascScore = chadsVascScore;
        this.qsofaScore = qsofaScore;
        this.confidenceScore = confidenceScore;
        this.calculationTimestamp = System.currentTimeMillis();
    }

    // Getters and Setters

    public EnhancedRiskIndicators.RiskAssessment getRiskAssessment() {
        return riskAssessment;
    }

    public void setRiskAssessment(EnhancedRiskIndicators.RiskAssessment riskAssessment) {
        this.riskAssessment = riskAssessment;
    }

    public NEWS2Calculator.NEWS2Score getNews2Score() {
        return news2Score;
    }

    public void setNews2Score(NEWS2Calculator.NEWS2Score news2Score) {
        this.news2Score = news2Score;
    }

    public List<SmartAlertGenerator.ClinicalAlert> getAlerts() {
        return alerts;
    }

    public void setAlerts(List<SmartAlertGenerator.ClinicalAlert> alerts) {
        this.alerts = alerts;
    }

    public ClinicalScoreCalculators.FraminghamScore getFraminghamScore() {
        return framinghamScore;
    }

    public void setFraminghamScore(ClinicalScoreCalculators.FraminghamScore framinghamScore) {
        this.framinghamScore = framinghamScore;
    }

    public ClinicalScoreCalculators.CHADS2VAScScore getChadsVascScore() {
        return chadsVascScore;
    }

    public void setChadsVascScore(ClinicalScoreCalculators.CHADS2VAScScore chadsVascScore) {
        this.chadsVascScore = chadsVascScore;
    }

    public ClinicalScoreCalculators.qSOFAScore getQsofaScore() {
        return qsofaScore;
    }

    public void setQsofaScore(ClinicalScoreCalculators.qSOFAScore qsofaScore) {
        this.qsofaScore = qsofaScore;
    }

    public MetabolicAcuityCalculator.MetabolicAcuityScore getMetabolicAcuityScore() {
        return metabolicAcuityScore;
    }

    public void setMetabolicAcuityScore(MetabolicAcuityCalculator.MetabolicAcuityScore metabolicAcuityScore) {
        this.metabolicAcuityScore = metabolicAcuityScore;
    }

    public CombinedAcuityCalculator.CombinedAcuityScore getCombinedAcuityScore() {
        return combinedAcuityScore;
    }

    public void setCombinedAcuityScore(CombinedAcuityCalculator.CombinedAcuityScore combinedAcuityScore) {
        this.combinedAcuityScore = combinedAcuityScore;
    }

    public ClinicalScoreCalculators.MetabolicSyndromeScore getMetabolicSyndromeScore() {
        return metabolicSyndromeScore;
    }

    public void setMetabolicSyndromeScore(ClinicalScoreCalculators.MetabolicSyndromeScore metabolicSyndromeScore) {
        this.metabolicSyndromeScore = metabolicSyndromeScore;
    }

    public ConfidenceScoreCalculator.ConfidenceScore getConfidenceScore() {
        return confidenceScore;
    }

    public void setConfidenceScore(ConfidenceScoreCalculator.ConfidenceScore confidenceScore) {
        this.confidenceScore = confidenceScore;
    }

    public long getCalculationTimestamp() {
        return calculationTimestamp;
    }

    public void setCalculationTimestamp(long calculationTimestamp) {
        this.calculationTimestamp = calculationTimestamp;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    // Phase 2 Getters and Setters

    public List<ProtocolMatcher.Protocol> getApplicableProtocols() {
        return applicableProtocols;
    }

    public void setApplicableProtocols(List<ProtocolMatcher.Protocol> applicableProtocols) {
        this.applicableProtocols = applicableProtocols;
    }

    public List<SimilarPatient> getSimilarPatients() {
        return similarPatients;
    }

    public void setSimilarPatients(List<SimilarPatient> similarPatients) {
        this.similarPatients = similarPatients;
    }

    public CohortInsights getCohortInsights() {
        return cohortInsights;
    }

    public void setCohortInsights(CohortInsights cohortInsights) {
        this.cohortInsights = cohortInsights;
    }

    public Recommendations getRecommendations() {
        return recommendations;
    }

    public void setRecommendations(Recommendations recommendations) {
        this.recommendations = recommendations;
    }

    /**
     * Get overall clinical urgency level based on all assessments
     */
    public String getOverallUrgency() {
        // Prefer combined acuity score if available (most comprehensive)
        if (combinedAcuityScore != null) {
            return combinedAcuityScore.getAcuityLevel();
        }

        // Check for critical conditions
        if (news2Score != null && "HIGH".equals(news2Score.getRiskLevel())) {
            return "CRITICAL";
        }

        if (riskAssessment != null && riskAssessment.getOverallRiskLevel() != null
                && "SEVERE".equals(riskAssessment.getOverallRiskLevel().toString())) {
            return "CRITICAL";
        }

        if (alerts != null) {
            boolean hasCriticalAlert = alerts.stream()
                .anyMatch(a -> "CRITICAL".equals(a.getPriority()));
            if (hasCriticalAlert) {
                return "CRITICAL";
            }
        }

        if (qsofaScore != null && qsofaScore.getTotalScore() >= 2) {
            return "HIGH"; // Potential sepsis
        }

        // Check for high urgency
        if (riskAssessment != null && riskAssessment.getOverallRiskLevel() != null
                && "HIGH".equals(riskAssessment.getOverallRiskLevel().toString())) {
            return "HIGH";
        }

        if (news2Score != null && "MEDIUM".equals(news2Score.getRiskLevel())) {
            return "MEDIUM";
        }

        return "ROUTINE";
    }

    /**
     * Check if patient requires immediate clinical attention
     */
    public boolean requiresImmediateAttention() {
        return "CRITICAL".equals(getOverallUrgency());
    }

    /**
     * Get summary of key clinical findings
     */
    public String getSummaryFindings() {
        StringBuilder summary = new StringBuilder();

        if (riskAssessment != null && riskAssessment.getOverallRiskLevel() != null
                && !"LOW".equals(riskAssessment.getOverallRiskLevel().toString())) {
            summary.append("Risk: ").append(riskAssessment.getOverallRiskLevel()).append(". ");
        }

        if (news2Score != null && news2Score.getTotalScore() >= 5) {
            summary.append("NEWS2: ").append(news2Score.getTotalScore())
                   .append(" (").append(news2Score.getRiskLevel()).append("). ");
        }

        if (alerts != null && !alerts.isEmpty()) {
            long criticalCount = alerts.stream()
                .filter(a -> "CRITICAL".equals(a.getPriority()))
                .count();
            if (criticalCount > 0) {
                summary.append(criticalCount).append(" critical alert(s). ");
            }
        }

        if (framinghamScore != null && "HIGH".equals(framinghamScore.getRiskCategory())) {
            summary.append("High CVD risk (").append(framinghamScore.getRiskPercentage())
                   .append("%). ");
        }

        if (qsofaScore != null && qsofaScore.getTotalScore() >= 2) {
            summary.append("Sepsis concern (qSOFA=").append(qsofaScore.getTotalScore()).append("). ");
        }

        if (confidenceScore != null && confidenceScore.getOverallConfidence() < 60) {
            summary.append("⚠️ Low confidence assessment. ");
        }

        return summary.length() > 0 ? summary.toString().trim() : "No significant clinical findings.";
    }

    @Override
    public String toString() {
        return "ClinicalIntelligence{" +
                "urgency='" + getOverallUrgency() + '\'' +
                ", riskLevel='" + (riskAssessment != null ? riskAssessment.getOverallRiskLevel() : "N/A") + '\'' +
                ", news2Score=" + (news2Score != null ? news2Score.getTotalScore() : "N/A") +
                ", alertCount=" + (alerts != null ? alerts.size() : 0) +
                ", confidence=" + (confidenceScore != null ? confidenceScore.getConfidenceLevel() : "N/A") +
                '}';
    }
}
