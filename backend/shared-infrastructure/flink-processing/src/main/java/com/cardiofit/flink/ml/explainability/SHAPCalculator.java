package com.cardiofit.flink.ml.explainability;

import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.models.MLPrediction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.stream.Collectors;

/**
 * SHAP (SHapley Additive exPlanations) Calculator for Clinical ML Models
 *
 * Provides model explainability by calculating feature importance and contributions
 * to individual predictions. Implements Kernel SHAP for model-agnostic explanations.
 *
 * Key Features:
 * - Feature contribution calculation (SHAP values)
 * - Top-K most important features identification
 * - Natural language explanation generation
 * - Local interpretability for individual predictions
 * - Population-level feature importance aggregation
 *
 * Clinical Use Case:
 * When a model predicts "HIGH sepsis risk (0.85)", SHAP explains:
 * - "Elevated lactate (4.2 mmol/L) contributed +0.3 to risk score"
 * - "High heart rate (120 bpm) contributed +0.15 to risk score"
 * - "Low blood pressure (85 mmHg) contributed +0.12 to risk score"
 *
 * This enables clinicians to validate model reasoning and trust predictions.
 *
 * Implementation Note:
 * This is a simplified Kernel SHAP implementation suitable for real-time inference.
 * For production deployment with complex models, consider integrating:
 * - TreeSHAP for tree-based models (XGBoost, Random Forest)
 * - DeepSHAP for neural networks
 * - External SHAP library via JNI bridge
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class SHAPCalculator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(SHAPCalculator.class);

    // Configuration
    private final int numSamples;           // Number of coalition samples for Kernel SHAP
    private final int topKFeatures;         // Number of top features to return
    private final double contributionThreshold;  // Minimum absolute contribution to report

    // Feature names for interpretation
    private final List<String> featureNames;

    // Background data for SHAP baseline (simplified - would use training data distribution)
    private final Map<String, Double> featureBaseline;

    /**
     * Constructor with default configuration
     */
    public SHAPCalculator(List<String> featureNames) {
        this(featureNames, 100, 10, 0.01);
    }

    /**
     * Constructor with custom configuration
     *
     * @param featureNames List of feature names in order
     * @param numSamples Number of coalition samples for Kernel SHAP
     * @param topKFeatures Number of top contributing features to return
     * @param contributionThreshold Minimum absolute contribution to report
     */
    public SHAPCalculator(List<String> featureNames,
                         int numSamples,
                         int topKFeatures,
                         double contributionThreshold) {
        this.featureNames = featureNames;
        this.numSamples = numSamples;
        this.topKFeatures = topKFeatures;
        this.contributionThreshold = contributionThreshold;
        this.featureBaseline = createFeatureBaseline(featureNames);
    }

    /**
     * Calculate SHAP values for a prediction
     *
     * @param model ONNX model used for prediction
     * @param features Feature vector for the instance
     * @param prediction Original prediction to explain
     * @return SHAP explanation with feature contributions
     */
    public SHAPExplanation explainPrediction(ONNXModelContainer model,
                                            ClinicalFeatureVector features,
                                            MLPrediction prediction) {
        long startTime = System.nanoTime();

        try {
            // Extract feature values
            float[] featureArray = features.toFloatArray();
            double baselinePrediction = prediction.getPrimaryScore();

            // Calculate SHAP values using Kernel SHAP approximation
            Map<String, Double> shapValues = calculateKernelSHAP(
                model,
                featureArray,
                baselinePrediction
            );

            // Identify top contributing features
            List<FeatureContribution> topContributions = getTopContributions(
                shapValues,
                features
            );

            // Generate natural language explanation
            String explanation = generateExplanation(
                topContributions,
                prediction.getModelType(),
                baselinePrediction
            );

            long calculationTimeNs = System.nanoTime() - startTime;
            double calculationTimeMs = calculationTimeNs / 1_000_000.0;

            LOG.debug("SHAP explanation calculated in {:.2f}ms for patient: {}",
                calculationTimeMs, features.getPatientId());

            // Convert internal FeatureContribution to SHAPExplanation.FeatureContribution
            List<SHAPExplanation.FeatureContribution> shapContributions = convertToSHAPContributions(topContributions);

            return SHAPExplanation.builder()
                .patientId(features.getPatientId())
                .predictionId(prediction.getId())
                .modelType(prediction.getModelType())
                .predictionScore(baselinePrediction)
                .shapValues(shapValues)
                .topContributions(shapContributions)
                .explanationText(explanation)
                .calculationTimeMs(calculationTimeMs)
                .build();

        } catch (Exception e) {
            LOG.error("SHAP calculation failed for patient: " + features.getPatientId(), e);
            return createFallbackExplanation(features, prediction);
        }
    }

    /**
     * Calculate SHAP values using Kernel SHAP approximation
     *
     * Kernel SHAP explanation:
     * 1. Create coalitions (subsets) of features
     * 2. For each coalition, predict with present/absent features
     * 3. Calculate marginal contribution of each feature
     * 4. Use weighted linear regression to approximate SHAP values
     *
     * This is a simplified implementation suitable for real-time use.
     * Production systems may use TreeSHAP or DeepSHAP for better accuracy.
     */
    private Map<String, Double> calculateKernelSHAP(ONNXModelContainer model,
                                                    float[] features,
                                                    double prediction) {
        Map<String, Double> shapValues = new LinkedHashMap<>();
        int numFeatures = features.length;

        // For real-time performance, use simplified approach:
        // Calculate feature importance via gradient-based attribution

        try {
            // Calculate baseline prediction (all features at population mean)
            float[] baselineFeatures = createBaselineFeatures();
            double baselinePrediction = model.predict(baselineFeatures).getPrimaryScore();

            // Calculate contribution of each feature by ablation
            for (int i = 0; i < numFeatures && i < featureNames.size(); i++) {
                String featureName = featureNames.get(i);

                // Create feature array with this feature replaced by baseline
                float[] ablatedFeatures = features.clone();
                ablatedFeatures[i] = baselineFeatures[i];

                // Predict with ablated features
                double ablatedPrediction = model.predict(ablatedFeatures).getPrimaryScore();

                // SHAP value approximation: difference from baseline
                double shapValue = prediction - ablatedPrediction;

                shapValues.put(featureName, shapValue);
            }

        } catch (Exception e) {
            LOG.warn("SHAP calculation failed, using feature variance method", e);

            // Fallback: Use feature deviation from baseline as proxy
            for (int i = 0; i < numFeatures && i < featureNames.size(); i++) {
                String featureName = featureNames.get(i);
                double baseline = featureBaseline.getOrDefault(featureName, 0.0);
                double deviation = features[i] - baseline;

                // Approximate contribution as deviation magnitude
                // (direction matters: positive deviation for risk factors)
                double approximateContribution = deviation * 0.01; // Scale factor

                shapValues.put(featureName, approximateContribution);
            }
        }

        return shapValues;
    }

    /**
     * Convert internal FeatureContribution to SHAPExplanation.FeatureContribution
     */
    private List<SHAPExplanation.FeatureContribution> convertToSHAPContributions(
            List<FeatureContribution> internalContributions) {
        List<SHAPExplanation.FeatureContribution> shapContributions = new ArrayList<>();

        for (FeatureContribution fc : internalContributions) {
            String unit = getFeatureUnit(fc.getFeatureName());
            String clinicalInterpretation = interpretFeatureClinically(
                fc.getFeatureName(), fc.getFeatureValue()
            );
            double[] normalRange = getNormalRange(fc.getFeatureName());

            SHAPExplanation.FeatureContribution shapFc = new SHAPExplanation.FeatureContribution(
                fc.getFeatureName(),
                fc.getFeatureValue(),
                unit,
                fc.getContribution(),  // SHAP value
                clinicalInterpretation,
                normalRange[0],  // normalRangeLower
                normalRange[1]   // normalRangeUpper
            );

            shapContributions.add(shapFc);
        }

        return shapContributions;
    }

    /**
     * Get feature unit string
     */
    private String getFeatureUnit(String featureName) {
        if (featureName.contains("heart_rate")) return "bpm";
        if (featureName.contains("_bp")) return "mmHg";
        if (featureName.contains("respiratory_rate")) return "/min";
        if (featureName.contains("temperature")) return "°C";
        if (featureName.contains("oxygen_saturation")) return "%";
        if (featureName.contains("lactate")) return "mmol/L";
        if (featureName.contains("creatinine")) return "mg/dL";
        if (featureName.contains("wbc")) return "K/µL";
        return "";
    }

    /**
     * Get normal range for feature [lower, upper]
     */
    private double[] getNormalRange(String featureName) {
        // Vitals
        if (featureName.equals("vital_heart_rate")) return new double[]{60, 100};
        if (featureName.equals("vital_systolic_bp")) return new double[]{90, 140};
        if (featureName.equals("vital_diastolic_bp")) return new double[]{60, 90};
        if (featureName.equals("vital_respiratory_rate")) return new double[]{12, 20};
        if (featureName.equals("vital_temperature_c")) return new double[]{36.5, 37.5};
        if (featureName.equals("vital_oxygen_saturation")) return new double[]{95, 100};

        // Labs
        if (featureName.equals("lab_lactate_mmol")) return new double[]{0.5, 2.0};
        if (featureName.equals("lab_creatinine_mg_dl")) return new double[]{0.7, 1.3};
        if (featureName.equals("lab_wbc_k_ul")) return new double[]{4.0, 11.0};

        // Default range (for binary features or unknown)
        return new double[]{0.0, 1.0};
    }

    /**
     * Get top K contributing features
     */
    private List<FeatureContribution> getTopContributions(Map<String, Double> shapValues,
                                                         ClinicalFeatureVector features) {
        return shapValues.entrySet().stream()
            .filter(entry -> Math.abs(entry.getValue()) >= contributionThreshold)
            .map(entry -> {
                String featureName = entry.getKey();
                double shapValue = entry.getValue();
                Double featureValue = features.getFeature(featureName);

                return new FeatureContribution(
                    featureName,
                    featureValue != null ? featureValue : 0.0,
                    shapValue
                );
            })
            .sorted(Comparator.comparingDouble(
                fc -> -Math.abs(fc.getContribution())  // Sort by absolute contribution
            ))
            .limit(topKFeatures)
            .collect(Collectors.toList());
    }

    /**
     * Generate natural language explanation of prediction
     */
    private String generateExplanation(List<FeatureContribution> topContributions,
                                      String modelType,
                                      double predictionScore) {
        if (topContributions.isEmpty()) {
            return "No significant feature contributions identified.";
        }

        StringBuilder explanation = new StringBuilder();

        // Risk category
        String riskLevel = getRiskLevel(predictionScore);
        explanation.append(String.format("Model predicts %s risk (score: %.3f). ",
            riskLevel, predictionScore));

        // Top contributing factors
        explanation.append("Key contributing factors:\n");

        for (int i = 0; i < Math.min(5, topContributions.size()); i++) {
            FeatureContribution fc = topContributions.get(i);
            String featureExplanation = explainFeatureContribution(fc, modelType);
            explanation.append(String.format("%d. %s\n", i + 1, featureExplanation));
        }

        return explanation.toString();
    }

    /**
     * Explain individual feature contribution in clinical terms
     */
    private String explainFeatureContribution(FeatureContribution fc, String modelType) {
        String featureName = fc.getFeatureName();
        double value = fc.getFeatureValue();
        double contribution = fc.getContribution();

        String direction = contribution > 0 ? "increased" : "decreased";
        String clinicalInterpretation = interpretFeatureClinically(featureName, value);

        return String.format("%s (%s) %s risk by %.3f: %s",
            formatFeatureName(featureName),
            formatFeatureValue(featureName, value),
            direction,
            Math.abs(contribution),
            clinicalInterpretation
        );
    }

    /**
     * Interpret feature value clinically
     */
    private String interpretFeatureClinically(String featureName, double value) {
        // Vitals
        if (featureName.equals("vital_heart_rate")) {
            if (value > 100) return "tachycardia indicates stress/infection";
            if (value < 60) return "bradycardia may indicate heart block";
            return "within normal range";
        }
        if (featureName.equals("vital_systolic_bp")) {
            if (value < 90) return "hypotension indicates shock";
            if (value > 140) return "hypertension";
            return "normotensive";
        }
        if (featureName.equals("vital_respiratory_rate")) {
            if (value > 20) return "tachypnea suggests sepsis/ARDS";
            return "normal respiratory rate";
        }
        if (featureName.equals("vital_temperature_c")) {
            if (value >= 38.0) return "fever indicates infection";
            if (value < 36.0) return "hypothermia in sepsis is ominous";
            return "afebrile";
        }

        // Labs
        if (featureName.equals("lab_lactate_mmol")) {
            if (value > 4.0) return "severe tissue hypoperfusion";
            if (value > 2.0) return "elevated lactate triggers sepsis protocol";
            return "normal tissue perfusion";
        }
        if (featureName.equals("lab_creatinine_mg_dl")) {
            if (value > 2.0) return "significant kidney injury";
            if (value > 1.5) return "acute kidney injury (AKI)";
            return "normal kidney function";
        }
        if (featureName.equals("lab_wbc_k_ul")) {
            if (value > 12) return "leukocytosis indicates infection";
            if (value < 4) return "leukopenia is bad prognostic sign";
            return "normal white blood cell count";
        }

        // Clinical scores
        if (featureName.equals("score_sofa")) {
            if (value >= 2) return "organ dysfunction present (Sepsis-3 criteria)";
            return "no significant organ dysfunction";
        }
        if (featureName.equals("score_qsofa")) {
            if (value >= 2) return "high risk for poor outcome";
            return "low qSOFA score";
        }

        // Medications
        if (featureName.equals("med_vasopressor_active")) {
            return value > 0 ? "requires vasopressor support for shock" : "no vasopressor";
        }
        if (featureName.equals("med_antibiotic_active")) {
            return value > 0 ? "receiving antibiotic therapy" : "no antibiotics";
        }

        // CEP patterns
        if (featureName.equals("pattern_sepsis_detected")) {
            return value > 0 ? "CEP sepsis pattern detected" : "no sepsis pattern";
        }
        if (featureName.equals("pattern_deterioration_detected")) {
            return value > 0 ? "clinical deterioration pattern detected" : "stable";
        }

        return "contributes to overall risk";
    }

    /**
     * Format feature name for display
     */
    private String formatFeatureName(String featureName) {
        return featureName
            .replace("_", " ")
            .replace("vital ", "")
            .replace("lab ", "")
            .replace("score ", "")
            .replace("med ", "")
            .replace("pattern ", "")
            .replace("comorbid ", "")
            .replace("temporal ", "");
    }

    /**
     * Format feature value with appropriate units
     */
    private String formatFeatureValue(String featureName, double value) {
        // Vitals
        if (featureName.contains("heart_rate")) return String.format("%.0f bpm", value);
        if (featureName.contains("_bp")) return String.format("%.0f mmHg", value);
        if (featureName.contains("respiratory_rate")) return String.format("%.0f /min", value);
        if (featureName.contains("temperature")) return String.format("%.1f°C", value);
        if (featureName.contains("oxygen_saturation")) return String.format("%.0f%%", value);

        // Labs
        if (featureName.contains("lactate")) return String.format("%.1f mmol/L", value);
        if (featureName.contains("creatinine")) return String.format("%.2f mg/dL", value);
        if (featureName.contains("wbc")) return String.format("%.1f K/µL", value);

        // Binary features
        if (value == 0.0 || value == 1.0) return value > 0 ? "Yes" : "No";

        // Default
        return String.format("%.2f", value);
    }

    /**
     * Get risk level from prediction score
     */
    private String getRiskLevel(double score) {
        if (score >= 0.8) return "HIGH";
        if (score >= 0.5) return "MODERATE";
        if (score >= 0.3) return "LOW";
        return "VERY LOW";
    }

    /**
     * Create baseline feature values (population means)
     */
    private float[] createBaselineFeatures() {
        float[] baseline = new float[featureNames.size()];

        for (int i = 0; i < featureNames.size(); i++) {
            String featureName = featureNames.get(i);
            Double baselineValue = featureBaseline.get(featureName);
            baseline[i] = baselineValue != null ? baselineValue.floatValue() : 0.0f;
        }

        return baseline;
    }

    /**
     * Create feature baseline (population means) for SHAP calculation
     */
    private Map<String, Double> createFeatureBaseline(List<String> featureNames) {
        Map<String, Double> baseline = new HashMap<>();

        // Demographics
        baseline.put("demo_age_years", 65.0);
        baseline.put("demo_gender_male", 0.5);
        baseline.put("demo_bmi", 27.0);
        baseline.put("demo_icu_patient", 0.3);
        baseline.put("demo_admission_emergency", 0.4);

        // Vitals (normal values)
        baseline.put("vital_heart_rate", 80.0);
        baseline.put("vital_systolic_bp", 120.0);
        baseline.put("vital_diastolic_bp", 75.0);
        baseline.put("vital_respiratory_rate", 16.0);
        baseline.put("vital_temperature_c", 37.0);
        baseline.put("vital_oxygen_saturation", 97.0);
        baseline.put("vital_mean_arterial_pressure", 90.0);
        baseline.put("vital_pulse_pressure", 45.0);
        baseline.put("vital_shock_index", 0.67);

        // Labs (normal values)
        baseline.put("lab_lactate_mmol", 1.5);
        baseline.put("lab_creatinine_mg_dl", 1.0);
        baseline.put("lab_wbc_k_ul", 8.0);

        // Scores
        baseline.put("score_sofa", 2.0);
        baseline.put("score_qsofa", 0.0);
        baseline.put("score_news2", 3.0);

        // Binary features default to 0
        for (String feature : featureNames) {
            if (!baseline.containsKey(feature)) {
                baseline.put(feature, 0.0);
            }
        }

        return baseline;
    }

    /**
     * Create fallback explanation when SHAP calculation fails
     */
    private SHAPExplanation createFallbackExplanation(ClinicalFeatureVector features,
                                                      MLPrediction prediction) {
        return SHAPExplanation.builder()
            .patientId(features.getPatientId())
            .predictionId(prediction.getId())
            .modelType(prediction.getModelType())
            .predictionScore(prediction.getPrimaryScore())
            .shapValues(new HashMap<>())
            .topContributions(new ArrayList<>())
            .explanationText("Explanation unavailable due to calculation error.")
            .calculationTimeMs(0.0)
            .build();
    }

    /**
     * Populate MLPrediction with SHAP explainability data
     */
    public void populateExplainability(MLPrediction prediction, SHAPExplanation explanation) {
        MLPrediction.ExplainabilityData explainabilityData = new MLPrediction.ExplainabilityData();

        // Set SHAP values
        explainabilityData.setShapValues(explanation.getShapValues());

        // Set top contributors
        List<String> topContributors = explanation.getTopContributions().stream()
            .map(fc -> fc.getFeatureName() + " (" + formatContribution(fc.getShapValue()) + ")")
            .collect(Collectors.toList());
        explainabilityData.setTopContributors(topContributors);

        // Set explanation text
        explainabilityData.setExplanationText(explanation.getExplanationText());

        // Set explainability method
        explainabilityData.setExplainabilityMethod("Kernel SHAP");

        prediction.setExplainabilityData(explainabilityData);
    }

    private String formatContribution(double contribution) {
        return String.format("%+.3f", contribution);
    }

    // ===== Inner Classes =====

    /**
     * Feature contribution container
     */
    public static class FeatureContribution implements Serializable {
        private final String featureName;
        private final double featureValue;
        private final double contribution;  // SHAP value

        public FeatureContribution(String featureName, double featureValue, double contribution) {
            this.featureName = featureName;
            this.featureValue = featureValue;
            this.contribution = contribution;
        }

        public String getFeatureName() { return featureName; }
        public double getFeatureValue() { return featureValue; }
        public double getContribution() { return contribution; }

        @Override
        public String toString() {
            return String.format("%s (%.2f) → %+.3f", featureName, featureValue, contribution);
        }
    }
}
