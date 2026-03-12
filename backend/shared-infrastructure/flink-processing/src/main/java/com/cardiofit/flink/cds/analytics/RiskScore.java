package com.cardiofit.flink.cds.analytics;

import java.time.LocalDateTime;
import java.util.HashMap;
import java.util.Map;

/**
 * Phase 8 Module 3 - Predictive Risk Scoring
 *
 * Core risk score model representing calculated risk for various clinical outcomes.
 * Supports multiple risk types (mortality, readmission, sepsis, deterioration).
 *
 * Based on evidence-based risk models:
 * - APACHE III (Mortality risk)
 * - HOSPITAL Score (30-day readmission)
 * - qSOFA (Sepsis risk)
 * - MEWS (Modified Early Warning Score for deterioration)
 *
 * @author CardioFit Clinical Intelligence Team
 * @version 1.0.0
 * @since Phase 8
 */
public class RiskScore {

    // Core Identification
    private String scoreId;
    private String patientId;
    private RiskType riskType;

    // Score Values
    private double score;                    // Primary risk score (e.g., 0.0 to 1.0 for probability)
    private double confidenceLower;          // Lower bound of 95% confidence interval
    private double confidenceUpper;          // Upper bound of 95% confidence interval
    private RiskCategory riskCategory;       // Categorical risk level

    // Calculation Metadata
    private String calculationMethod;        // e.g., "APACHE_III_v1.0", "HOSPITAL_SCORE_v2.0"
    private LocalDateTime calculationTime;
    private String calculatedBy;             // Service or algorithm identifier

    // Contributing Factors
    private Map<String, Object> inputParameters;  // Raw inputs used in calculation
    private Map<String, Double> featureWeights;   // Contribution of each factor to final score

    // Clinical Context
    private String primaryDiagnosis;         // ICD-10 code or description
    private boolean requiresIntervention;    // True if score triggers alert/protocol
    private String recommendedAction;        // Suggested clinical action based on score

    // Versioning and Validation
    private String modelVersion;             // Algorithm version for reproducibility
    private boolean isValidated;             // True if score passed internal validation
    private String validationNotes;          // Any validation warnings or notes

    /**
     * Risk types supported by the predictive engine
     */
    public enum RiskType {
        MORTALITY,          // Risk of death (APACHE III)
        READMISSION,        // 30-day readmission risk (HOSPITAL Score)
        SEPSIS,             // Sepsis development (qSOFA)
        DETERIORATION,      // Clinical deterioration (MEWS)
        CARDIAC_EVENT,      // Cardiac event risk
        RESPIRATORY_FAILURE,// Respiratory failure risk
        RENAL_FAILURE,      // Acute kidney injury risk
        CUSTOM              // Custom risk model
    }

    /**
     * Categorical risk levels for clinical decision support
     */
    public enum RiskCategory {
        LOW(0, "Routine monitoring sufficient"),
        MODERATE(1, "Enhanced monitoring recommended"),
        HIGH(2, "Immediate clinical assessment required"),
        CRITICAL(3, "Urgent intervention required");

        private final int severity;
        private final String clinicalGuidance;

        RiskCategory(int severity, String clinicalGuidance) {
            this.severity = severity;
            this.clinicalGuidance = clinicalGuidance;
        }

        public int getSeverity() {
            return severity;
        }

        public String getClinicalGuidance() {
            return clinicalGuidance;
        }
    }

    // Constructors
    public RiskScore() {
        this.inputParameters = new HashMap<>();
        this.featureWeights = new HashMap<>();
        this.calculationTime = LocalDateTime.now();
        this.isValidated = false;
    }

    public RiskScore(String patientId, RiskType riskType, double score) {
        this();
        this.patientId = patientId;
        this.riskType = riskType;
        this.score = score;
        this.scoreId = generateScoreId(patientId, riskType);
    }

    /**
     * Generate unique score identifier
     */
    private String generateScoreId(String patientId, RiskType riskType) {
        return String.format("RISK_%s_%s_%d",
            patientId,
            riskType.name(),
            System.currentTimeMillis());
    }

    /**
     * Determine risk category based on score threshold
     */
    public RiskCategory categorizeRisk() {
        // Default thresholds (can be overridden per risk type)
        if (score < 0.2) return RiskCategory.LOW;
        if (score < 0.5) return RiskCategory.MODERATE;
        if (score < 0.8) return RiskCategory.HIGH;
        return RiskCategory.CRITICAL;
    }

    /**
     * Check if score requires immediate clinical action
     */
    public boolean requiresImmediateAction() {
        return riskCategory == RiskCategory.HIGH || riskCategory == RiskCategory.CRITICAL;
    }

    /**
     * Add input parameter used in calculation
     */
    public void addInputParameter(String name, Object value) {
        this.inputParameters.put(name, value);
    }

    /**
     * Add feature weight showing contribution to score
     */
    public void addFeatureWeight(String featureName, double weight) {
        this.featureWeights.put(featureName, weight);
    }

    /**
     * Get top contributing factors to risk score
     */
    public Map<String, Double> getTopContributors(int topN) {
        return featureWeights.entrySet().stream()
            .sorted(Map.Entry.<String, Double>comparingByValue().reversed())
            .limit(topN)
            .collect(HashMap::new, (m, e) -> m.put(e.getKey(), e.getValue()), HashMap::putAll);
    }

    /**
     * Validate score is within expected range and has required metadata
     */
    public boolean validate() {
        if (score < 0.0 || score > 1.0) {
            validationNotes = "Score out of range [0.0, 1.0]: " + score;
            return false;
        }

        if (confidenceLower > confidenceUpper) {
            validationNotes = "Invalid confidence interval: lower > upper";
            return false;
        }

        if (patientId == null || patientId.isEmpty()) {
            validationNotes = "Missing patient ID";
            return false;
        }

        if (riskType == null) {
            validationNotes = "Missing risk type";
            return false;
        }

        if (calculationMethod == null || calculationMethod.isEmpty()) {
            validationNotes = "Missing calculation method";
            return false;
        }

        this.isValidated = true;
        this.validationNotes = "Validation passed";
        return true;
    }

    // Getters and Setters
    public String getScoreId() {
        return scoreId;
    }

    public void setScoreId(String scoreId) {
        this.scoreId = scoreId;
    }

    public String getPatientId() {
        return patientId;
    }

    public void setPatientId(String patientId) {
        this.patientId = patientId;
    }

    public RiskType getRiskType() {
        return riskType;
    }

    public void setRiskType(RiskType riskType) {
        this.riskType = riskType;
    }

    public double getScore() {
        return score;
    }

    public void setScore(double score) {
        this.score = score;
    }

    public double getConfidenceLower() {
        return confidenceLower;
    }

    public void setConfidenceLower(double confidenceLower) {
        this.confidenceLower = confidenceLower;
    }

    public double getConfidenceUpper() {
        return confidenceUpper;
    }

    public void setConfidenceUpper(double confidenceUpper) {
        this.confidenceUpper = confidenceUpper;
    }

    public RiskCategory getRiskCategory() {
        return riskCategory;
    }

    public void setRiskCategory(RiskCategory riskCategory) {
        this.riskCategory = riskCategory;
    }

    public String getCalculationMethod() {
        return calculationMethod;
    }

    public void setCalculationMethod(String calculationMethod) {
        this.calculationMethod = calculationMethod;
    }

    public LocalDateTime getCalculationTime() {
        return calculationTime;
    }

    public void setCalculationTime(LocalDateTime calculationTime) {
        this.calculationTime = calculationTime;
    }

    public String getCalculatedBy() {
        return calculatedBy;
    }

    public void setCalculatedBy(String calculatedBy) {
        this.calculatedBy = calculatedBy;
    }

    public Map<String, Object> getInputParameters() {
        return inputParameters;
    }

    public void setInputParameters(Map<String, Object> inputParameters) {
        this.inputParameters = inputParameters;
    }

    public Map<String, Double> getFeatureWeights() {
        return featureWeights;
    }

    public void setFeatureWeights(Map<String, Double> featureWeights) {
        this.featureWeights = featureWeights;
    }

    public String getPrimaryDiagnosis() {
        return primaryDiagnosis;
    }

    public void setPrimaryDiagnosis(String primaryDiagnosis) {
        this.primaryDiagnosis = primaryDiagnosis;
    }

    public boolean isRequiresIntervention() {
        return requiresIntervention;
    }

    public void setRequiresIntervention(boolean requiresIntervention) {
        this.requiresIntervention = requiresIntervention;
    }

    public String getRecommendedAction() {
        return recommendedAction;
    }

    public void setRecommendedAction(String recommendedAction) {
        this.recommendedAction = recommendedAction;
    }

    public String getModelVersion() {
        return modelVersion;
    }

    public void setModelVersion(String modelVersion) {
        this.modelVersion = modelVersion;
    }

    public boolean isValidated() {
        return isValidated;
    }

    public void setValidated(boolean validated) {
        isValidated = validated;
    }

    public String getValidationNotes() {
        return validationNotes;
    }

    public void setValidationNotes(String validationNotes) {
        this.validationNotes = validationNotes;
    }

    @Override
    public String toString() {
        return String.format("RiskScore{id='%s', patient='%s', type=%s, score=%.3f, category=%s, method='%s', validated=%s}",
            scoreId, patientId, riskType, score, riskCategory, calculationMethod, isValidated);
    }
}
