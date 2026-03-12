package com.cardiofit.flink.ml.explainability;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * SHAP Explanation Container
 *
 * Holds SHAP (SHapley Additive exPlanations) values and clinical interpretations
 * for ML model predictions. Provides complete explainability for clinical risk scores.
 *
 * Key Components:
 * - SHAP values: Feature importance scores (positive = increases risk, negative = decreases)
 * - Top contributions: Most influential features ranked by absolute impact
 * - Clinical explanation: Human-readable interpretation for clinicians
 * - Calculation metrics: Performance tracking for explainability computation
 *
 * Usage:
 * <pre>
 * SHAPExplanation explanation = SHAPExplanation.builder()
 *     .patientId("P12345")
 *     .predictionId("pred_67890")
 *     .modelType("sepsis_risk")
 *     .predictionScore(0.82)
 *     .shapValues(shapValuesMap)
 *     .topContributions(topFeatures)
 *     .explanationText("High sepsis risk driven by elevated lactate...")
 *     .calculationTimeMs(45.3)
 *     .build();
 * </pre>
 *
 * Clinical Example:
 * For a sepsis risk prediction of 0.82:
 * - Top positive contributors: lactate (+0.25), WBC (+0.18), temperature (+0.12)
 * - Top negative contributors: blood pressure (-0.08), oxygen saturation (-0.05)
 * - Explanation: "Patient at HIGH risk for sepsis. Primary drivers:
 *   elevated lactate (4.2 mmol/L) indicating tissue hypoperfusion,
 *   leukocytosis (18,000 cells/μL) suggesting infection..."
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class SHAPExplanation implements Serializable {
    private static final long serialVersionUID = 1L;

    // Identification
    private final String patientId;
    private final String predictionId;
    private final String modelType;
    private final long timestamp;

    // Prediction context
    private final double predictionScore;
    private final double baselineScore;  // Population baseline for comparison

    // SHAP values
    private final Map<String, Double> shapValues;  // Feature name → SHAP value
    private final int totalFeatures;

    // Top contributors
    private final List<FeatureContribution> topContributions;
    private final int topK;

    // Clinical interpretation
    private final String explanationText;
    private final String riskLevel;  // "LOW", "MEDIUM", "HIGH", "CRITICAL"
    private final List<String> clinicalRecommendations;

    // Calculation metrics
    private final double calculationTimeMs;
    private final int numSamples;  // Number of SHAP samples used

    /**
     * Private constructor - use Builder
     */
    private SHAPExplanation(Builder builder) {
        this.patientId = builder.patientId;
        this.predictionId = builder.predictionId;
        this.modelType = builder.modelType;
        this.timestamp = builder.timestamp;
        this.predictionScore = builder.predictionScore;
        this.baselineScore = builder.baselineScore;
        this.shapValues = builder.shapValues;
        this.totalFeatures = builder.shapValues != null ? builder.shapValues.size() : 0;
        this.topContributions = builder.topContributions;
        this.topK = builder.topContributions != null ? builder.topContributions.size() : 0;
        this.explanationText = builder.explanationText;
        this.riskLevel = builder.riskLevel;
        this.clinicalRecommendations = builder.clinicalRecommendations;
        this.calculationTimeMs = builder.calculationTimeMs;
        this.numSamples = builder.numSamples;
    }

    // ===== Getters =====

    public String getPatientId() {
        return patientId;
    }

    public String getPredictionId() {
        return predictionId;
    }

    public String getModelType() {
        return modelType;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public double getPredictionScore() {
        return predictionScore;
    }

    public double getBaselineScore() {
        return baselineScore;
    }

    public Map<String, Double> getShapValues() {
        return shapValues;
    }

    public int getTotalFeatures() {
        return totalFeatures;
    }

    public List<FeatureContribution> getTopContributions() {
        return topContributions;
    }

    public int getTopK() {
        return topK;
    }

    public String getExplanationText() {
        return explanationText;
    }

    public String getRiskLevel() {
        return riskLevel;
    }

    public List<String> getClinicalRecommendations() {
        return clinicalRecommendations;
    }

    public double getCalculationTimeMs() {
        return calculationTimeMs;
    }

    public int getNumSamples() {
        return numSamples;
    }

    /**
     * Get positive contributors (features that increase risk)
     */
    public List<FeatureContribution> getPositiveContributions() {
        List<FeatureContribution> positive = new ArrayList<>();
        for (FeatureContribution fc : topContributions) {
            if (fc.getShapValue() > 0) {
                positive.add(fc);
            }
        }
        return positive;
    }

    /**
     * Get negative contributors (features that decrease risk)
     */
    public List<FeatureContribution> getNegativeContributions() {
        List<FeatureContribution> negative = new ArrayList<>();
        for (FeatureContribution fc : topContributions) {
            if (fc.getShapValue() < 0) {
                negative.add(fc);
            }
        }
        return negative;
    }

    /**
     * Get total positive contribution (sum of all positive SHAP values)
     */
    public double getTotalPositiveContribution() {
        return shapValues.values().stream()
            .filter(v -> v > 0)
            .mapToDouble(Double::doubleValue)
            .sum();
    }

    /**
     * Get total negative contribution (sum of all negative SHAP values)
     */
    public double getTotalNegativeContribution() {
        return shapValues.values().stream()
            .filter(v -> v < 0)
            .mapToDouble(Double::doubleValue)
            .sum();
    }

    /**
     * Get deviation from baseline
     */
    public double getDeviationFromBaseline() {
        return predictionScore - baselineScore;
    }

    /**
     * Get explanation quality score (0.0 to 1.0)
     * Higher score = better explanation coverage
     */
    public double getExplanationQuality() {
        if (topContributions == null || topContributions.isEmpty()) {
            return 0.0;
        }

        // Sum of top K absolute SHAP values
        double topKSum = topContributions.stream()
            .mapToDouble(fc -> Math.abs(fc.getShapValue()))
            .sum();

        // Total absolute SHAP values
        double totalSum = shapValues.values().stream()
            .mapToDouble(Math::abs)
            .sum();

        return totalSum > 0 ? topKSum / totalSum : 0.0;
    }

    @Override
    public String toString() {
        return String.format(
            "SHAPExplanation{patient=%s, model=%s, score=%.3f, baseline=%.3f, topContributions=%d, quality=%.1f%%}",
            patientId, modelType, predictionScore, baselineScore, topK, getExplanationQuality() * 100
        );
    }

    /**
     * Generate detailed clinical report
     */
    public String toDetailedReport() {
        StringBuilder report = new StringBuilder();

        // Header
        report.append("═══════════════════════════════════════════════════════════════\n");
        report.append(String.format("  ML Prediction Explainability Report - %s\n", modelType));
        report.append("═══════════════════════════════════════════════════════════════\n\n");

        // Patient and Prediction
        report.append(String.format("Patient ID: %s\n", patientId));
        report.append(String.format("Prediction ID: %s\n", predictionId));
        report.append(String.format("Timestamp: %d\n\n", timestamp));

        // Risk Score
        report.append(String.format("Risk Score: %.3f (%s)\n", predictionScore, riskLevel));
        report.append(String.format("Population Baseline: %.3f\n", baselineScore));
        report.append(String.format("Deviation from Baseline: %+.3f\n\n", getDeviationFromBaseline()));

        // Clinical Explanation
        report.append("Clinical Interpretation:\n");
        report.append(explanationText).append("\n\n");

        // Top Contributing Factors
        report.append(String.format("Top %d Contributing Factors:\n", topK));
        report.append("───────────────────────────────────────────────────────────────\n");

        int rank = 1;
        for (FeatureContribution fc : topContributions) {
            String direction = fc.getShapValue() > 0 ? "increases" : "decreases";
            report.append(String.format(
                "%d. %s (%.3f %s) - %s risk by %+.3f\n",
                rank++,
                fc.getFeatureName(),
                fc.getFeatureValue(),
                fc.getUnit(),
                direction,
                fc.getShapValue()
            ));
            report.append(String.format("   Clinical Context: %s\n\n", fc.getClinicalInterpretation()));
        }

        // Summary Statistics
        report.append("Summary Statistics:\n");
        report.append("───────────────────────────────────────────────────────────────\n");
        report.append(String.format("Total Features: %d\n", totalFeatures));
        report.append(String.format("Top %d Features Explained: %.1f%% of prediction\n",
            topK, getExplanationQuality() * 100));
        report.append(String.format("Total Positive Contribution: %+.3f\n", getTotalPositiveContribution()));
        report.append(String.format("Total Negative Contribution: %+.3f\n", getTotalNegativeContribution()));
        report.append(String.format("Calculation Time: %.2f ms\n", calculationTimeMs));
        report.append(String.format("SHAP Samples: %d\n\n", numSamples));

        // Clinical Recommendations
        if (clinicalRecommendations != null && !clinicalRecommendations.isEmpty()) {
            report.append("Clinical Recommendations:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            for (int i = 0; i < clinicalRecommendations.size(); i++) {
                report.append(String.format("%d. %s\n", i + 1, clinicalRecommendations.get(i)));
            }
        }

        report.append("═══════════════════════════════════════════════════════════════\n");

        return report.toString();
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String patientId;
        private String predictionId;
        private String modelType;
        private long timestamp = System.currentTimeMillis();
        private double predictionScore;
        private double baselineScore = 0.5;  // Default population baseline
        private Map<String, Double> shapValues = new LinkedHashMap<>();
        private List<FeatureContribution> topContributions = new ArrayList<>();
        private String explanationText = "";
        private String riskLevel = "UNKNOWN";
        private List<String> clinicalRecommendations = new ArrayList<>();
        private double calculationTimeMs;
        private int numSamples = 1000;  // Default SHAP samples

        public Builder patientId(String patientId) {
            this.patientId = patientId;
            return this;
        }

        public Builder predictionId(String predictionId) {
            this.predictionId = predictionId;
            return this;
        }

        public Builder modelType(String modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder timestamp(long timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public Builder predictionScore(double predictionScore) {
            this.predictionScore = predictionScore;
            return this;
        }

        public Builder baselineScore(double baselineScore) {
            this.baselineScore = baselineScore;
            return this;
        }

        public Builder shapValues(Map<String, Double> shapValues) {
            this.shapValues = new LinkedHashMap<>(shapValues);
            return this;
        }

        public Builder topContributions(List<FeatureContribution> topContributions) {
            this.topContributions = new ArrayList<>(topContributions);
            return this;
        }

        public Builder explanationText(String explanationText) {
            this.explanationText = explanationText;
            return this;
        }

        public Builder riskLevel(String riskLevel) {
            this.riskLevel = riskLevel;
            return this;
        }

        public Builder clinicalRecommendations(List<String> clinicalRecommendations) {
            this.clinicalRecommendations = new ArrayList<>(clinicalRecommendations);
            return this;
        }

        public Builder addClinicalRecommendation(String recommendation) {
            this.clinicalRecommendations.add(recommendation);
            return this;
        }

        public Builder calculationTimeMs(double calculationTimeMs) {
            this.calculationTimeMs = calculationTimeMs;
            return this;
        }

        public Builder numSamples(int numSamples) {
            this.numSamples = numSamples;
            return this;
        }

        public SHAPExplanation build() {
            // Validation
            if (patientId == null || patientId.isEmpty()) {
                throw new IllegalArgumentException("patientId is required");
            }
            if (predictionId == null || predictionId.isEmpty()) {
                throw new IllegalArgumentException("predictionId is required");
            }
            if (modelType == null || modelType.isEmpty()) {
                throw new IllegalArgumentException("modelType is required");
            }
            if (shapValues == null || shapValues.isEmpty()) {
                throw new IllegalArgumentException("shapValues cannot be empty");
            }
            if (topContributions == null || topContributions.isEmpty()) {
                throw new IllegalArgumentException("topContributions cannot be empty");
            }

            return new SHAPExplanation(this);
        }
    }

    /**
     * Feature Contribution Container
     *
     * Represents a single feature's contribution to the prediction with clinical context
     */
    public static class FeatureContribution implements Serializable {
        private static final long serialVersionUID = 1L;

        private final String featureName;
        private final double featureValue;
        private final String unit;
        private final double shapValue;
        private final String clinicalInterpretation;
        private final double normalRangeLower;
        private final double normalRangeUpper;

        public FeatureContribution(String featureName,
                                  double featureValue,
                                  String unit,
                                  double shapValue,
                                  String clinicalInterpretation,
                                  double normalRangeLower,
                                  double normalRangeUpper) {
            this.featureName = featureName;
            this.featureValue = featureValue;
            this.unit = unit;
            this.shapValue = shapValue;
            this.clinicalInterpretation = clinicalInterpretation;
            this.normalRangeLower = normalRangeLower;
            this.normalRangeUpper = normalRangeUpper;
        }

        public String getFeatureName() {
            return featureName;
        }

        public double getFeatureValue() {
            return featureValue;
        }

        public String getUnit() {
            return unit;
        }

        public double getShapValue() {
            return shapValue;
        }

        public String getClinicalInterpretation() {
            return clinicalInterpretation;
        }

        public double getNormalRangeLower() {
            return normalRangeLower;
        }

        public double getNormalRangeUpper() {
            return normalRangeUpper;
        }

        public boolean isAbnormal() {
            return featureValue < normalRangeLower || featureValue > normalRangeUpper;
        }

        public String getAbnormalityDirection() {
            if (featureValue < normalRangeLower) {
                return "LOW";
            } else if (featureValue > normalRangeUpper) {
                return "HIGH";
            } else {
                return "NORMAL";
            }
        }

        public double getDeviationFromNormal() {
            if (featureValue < normalRangeLower) {
                return featureValue - normalRangeLower;
            } else if (featureValue > normalRangeUpper) {
                return featureValue - normalRangeUpper;
            } else {
                return 0.0;
            }
        }

        @Override
        public String toString() {
            return String.format(
                "%s: %.3f %s (SHAP: %+.3f) - %s",
                featureName, featureValue, unit, shapValue,
                isAbnormal() ? getAbnormalityDirection() : "NORMAL"
            );
        }
    }
}
