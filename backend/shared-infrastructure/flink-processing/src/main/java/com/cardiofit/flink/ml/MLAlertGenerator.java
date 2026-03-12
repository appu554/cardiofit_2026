package com.cardiofit.flink.ml;

import com.cardiofit.flink.ml.explainability.SHAPExplanation;
import com.cardiofit.flink.models.EnhancedAlert;
import com.cardiofit.flink.models.MLPrediction;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * ML Alert Generator
 *
 * Generates clinical alerts based on ML prediction thresholds and clinical rules.
 * Supports multiple alert types with configurable thresholds, hysteresis, and
 * suppression to prevent alert fatigue.
 *
 * Alert Generation Logic:
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 1. Threshold Evaluation                                                  │
 * │    - Compare ML score against configured thresholds                      │
 * │    - Apply hysteresis to prevent flapping                                │
 * │    - Check minimum confidence requirements                               │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 2. Trend Analysis                                                        │
 * │    - Analyze recent prediction history                                   │
 * │    - Detect rising/falling trends                                        │
 * │    - Calculate rate of change                                            │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 3. Alert Suppression                                                     │
 * │    - Check for duplicate alerts in suppression window                    │
 * │    - Apply rate limiting (max alerts per patient per hour)               │
 * │    - Escalation: Re-alert if severity increases                          │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 4. Alert Generation                                                      │
 * │    - Create EnhancedAlert with evidence and recommendations              │
 * │    - Include SHAP explanation for clinical context                       │
 * │    - Add trend information for clinical decision-making                  │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * Configuration Example:
 * <pre>
 * MLAlertThresholdConfig config = MLAlertThresholdConfig.builder()
 *     .addThreshold("sepsis_risk", AlertThreshold.builder()
 *         .criticalThreshold(0.85)
 *         .highThreshold(0.70)
 *         .mediumThreshold(0.50)
 *         .hysteresis(0.05)  // Require 0.05 drop before clearing alert
 *         .minConfidence(0.80)
 *         .suppressionWindowMs(300_000)  // 5 minutes
 *         .build())
 *     .build();
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class MLAlertGenerator extends ProcessFunction<MLPrediction, EnhancedAlert> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(MLAlertGenerator.class);

    // Configuration
    private final MLAlertThresholdConfig config;

    // State: Recent predictions for trend analysis
    private transient ValueState<List<MLPrediction>> recentPredictionsState;

    // State: Last alert per patient for suppression
    private transient ValueState<Map<String, AlertHistory>> alertHistoryState;

    public MLAlertGenerator(MLAlertThresholdConfig config) {
        this.config = config;
    }

    @Override
    public void open(OpenContext context) throws Exception {
        super.open(context);

        recentPredictionsState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("recent-predictions",
                TypeInformation.of(new TypeHint<List<MLPrediction>>(){}))
        );

        alertHistoryState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("alert-history",
                TypeInformation.of(new TypeHint<Map<String, AlertHistory>>(){}))
        );

        LOG.info("MLAlertGenerator initialized with {} threshold configs", config.getThresholds().size());
    }

    @Override
    public void processElement(MLPrediction prediction,
                              Context ctx,
                              Collector<EnhancedAlert> out) throws Exception {
        long startTime = System.nanoTime();

        String patientId = prediction.getPatientId();
        String modelType = prediction.getModelType();
        double score = prediction.getPrimaryScore();
        double confidence = prediction.getModelConfidence();

        LOG.debug("Processing ML prediction for patient {}: model={}, score={}, confidence={}",
            patientId, modelType, score, confidence);

        // Get threshold configuration for this model type
        AlertThreshold threshold = config.getThreshold(modelType);
        if (threshold == null) {
            LOG.debug("No threshold configuration for model type: {}", modelType);
            return;
        }

        // Step 1: Threshold evaluation
        String severity = evaluateThreshold(score, threshold);
        if (severity == null) {
            LOG.debug("Prediction below minimum threshold for patient {}", patientId);
            return;
        }

        // Check minimum confidence
        if (confidence < threshold.getMinConfidence()) {
            LOG.debug("Prediction confidence ({}) below minimum ({}) for patient {}",
                confidence, threshold.getMinConfidence(), patientId);
            return;
        }

        // Step 2: Trend analysis
        TrendAnalysis trend = analyzeTrend(prediction);

        // Step 3: Alert suppression check
        if (shouldSuppressAlert(patientId, modelType, severity, threshold, trend)) {
            LOG.debug("Alert suppressed for patient {}: model={}, severity={}", patientId, modelType, severity);
            return;
        }

        // Step 4: Generate alert
        EnhancedAlert alert = generateAlert(prediction, severity, threshold, trend);

        // Update alert history
        updateAlertHistory(patientId, modelType, alert);

        // Emit alert
        out.collect(alert);

        double processingTimeMs = (System.nanoTime() - startTime) / 1_000_000.0;
        LOG.info("Generated ML alert for patient {}: model={}, severity={}, score={} ({}ms)",
            patientId, modelType, severity, score, String.format("%.2f", processingTimeMs));
    }

    /**
     * Evaluate threshold and determine severity level
     */
    private String evaluateThreshold(double score, AlertThreshold threshold) {
        if (score >= threshold.getCriticalThreshold()) {
            return "CRITICAL";
        } else if (score >= threshold.getHighThreshold()) {
            return "HIGH";
        } else if (score >= threshold.getMediumThreshold()) {
            return "MEDIUM";
        } else if (score >= threshold.getLowThreshold()) {
            return "LOW";
        }

        return null;  // Below minimum threshold
    }

    /**
     * Analyze prediction trend
     */
    private TrendAnalysis analyzeTrend(MLPrediction prediction) throws Exception {
        List<MLPrediction> recentPredictions = recentPredictionsState.value();
        if (recentPredictions == null) {
            recentPredictions = new ArrayList<>();
        }

        // Add current prediction
        recentPredictions.add(prediction);

        // Keep last 10 predictions
        if (recentPredictions.size() > 10) {
            recentPredictions.remove(0);
        }

        recentPredictionsState.update(recentPredictions);

        // Calculate trend
        if (recentPredictions.size() < 3) {
            return new TrendAnalysis("INSUFFICIENT_DATA", 0.0, 0.0);
        }

        // Get scores for same model type
        List<Double> scores = new ArrayList<>();
        for (MLPrediction pred : recentPredictions) {
            if (pred.getModelType().equals(prediction.getModelType())) {
                scores.add(pred.getPrimaryScore());
            }
        }

        if (scores.size() < 3) {
            return new TrendAnalysis("INSUFFICIENT_DATA", 0.0, 0.0);
        }

        // Calculate linear regression slope
        double slope = calculateSlope(scores);
        double latestScore = scores.get(scores.size() - 1);
        double earliestScore = scores.get(0);
        double change = latestScore - earliestScore;

        // Determine trend direction
        String direction;
        if (Math.abs(slope) < 0.01) {
            direction = "STABLE";
        } else if (slope > 0) {
            direction = "RISING";
        } else {
            direction = "FALLING";
        }

        return new TrendAnalysis(direction, slope, change);
    }

    /**
     * Calculate linear regression slope
     */
    private double calculateSlope(List<Double> scores) {
        int n = scores.size();
        double sumX = 0.0;
        double sumY = 0.0;
        double sumXY = 0.0;
        double sumX2 = 0.0;

        for (int i = 0; i < n; i++) {
            double x = i;
            double y = scores.get(i);
            sumX += x;
            sumY += y;
            sumXY += x * y;
            sumX2 += x * x;
        }

        double slope = (n * sumXY - sumX * sumY) / (n * sumX2 - sumX * sumX);
        return slope;
    }

    /**
     * Check if alert should be suppressed
     */
    private boolean shouldSuppressAlert(String patientId,
                                       String modelType,
                                       String severity,
                                       AlertThreshold threshold,
                                       TrendAnalysis trend) throws Exception {
        Map<String, AlertHistory> alertHistory = alertHistoryState.value();
        if (alertHistory == null) {
            alertHistory = new HashMap<>();
        }

        AlertHistory lastAlert = alertHistory.get(modelType);
        if (lastAlert == null) {
            return false;  // No previous alert
        }

        long currentTime = System.currentTimeMillis();
        long timeSinceLastAlert = currentTime - lastAlert.getTimestamp();

        // Check suppression window
        if (timeSinceLastAlert < threshold.getSuppressionWindowMs()) {
            // Within suppression window - check for escalation
            if (severityLevel(severity) > severityLevel(lastAlert.getSeverity())) {
                LOG.info("Alert escalated for patient {}: {} → {}", patientId, lastAlert.getSeverity(), severity);
                return false;  // Allow escalation
            }

            // Check for rapid deterioration (rising trend with high slope)
            if ("RISING".equals(trend.getDirection()) && Math.abs(trend.getSlope()) > 0.05) {
                LOG.info("Rapid deterioration detected for patient {}: slope={}", patientId, trend.getSlope());
                return false;  // Allow alert for rapid change
            }

            return true;  // Suppress duplicate
        }

        // Check hysteresis for clearing alerts
        if (timeSinceLastAlert < threshold.getSuppressionWindowMs() * 2) {
            // Apply hysteresis: require score to drop below (threshold - hysteresis)
            // before clearing alert and re-alerting
            double hysteresisAdjustedThreshold = getThresholdForSeverity(lastAlert.getSeverity(), threshold) -
                                                 threshold.getHysteresis();

            if (lastAlert.getScore() >= hysteresisAdjustedThreshold) {
                return true;  // Still within hysteresis band
            }
        }

        return false;  // Allow alert
    }

    private double getThresholdForSeverity(String severity, AlertThreshold threshold) {
        switch (severity.toUpperCase()) {
            case "CRITICAL": return threshold.getCriticalThreshold();
            case "HIGH": return threshold.getHighThreshold();
            case "MEDIUM": return threshold.getMediumThreshold();
            case "LOW": return threshold.getLowThreshold();
            default: return 0.0;
        }
    }

    private int severityLevel(String severity) {
        switch (severity.toUpperCase()) {
            case "CRITICAL": return 4;
            case "HIGH": return 3;
            case "MEDIUM": return 2;
            case "LOW": return 1;
            default: return 0;
        }
    }

    /**
     * Generate enhanced alert
     */
    private EnhancedAlert generateAlert(MLPrediction prediction,
                                       String severity,
                                       AlertThreshold threshold,
                                       TrendAnalysis trend) {
        List<String> evidenceSources = new ArrayList<>();
        evidenceSources.add("ML Model: " + prediction.getModelType() + " (score: " +
                           String.format("%.3f", prediction.getPrimaryScore()) + ")");
        evidenceSources.add("Model Confidence: " + String.format("%.1f%%", prediction.getModelConfidence() * 100));
        evidenceSources.add("Trend: " + trend.getDirection() + " (slope: " +
                           String.format("%+.4f", trend.getSlope()) + ")");

        // Get SHAP explanation text
        String shapExplanationText = null;
        if (prediction.getExplainabilityData() != null) {
            shapExplanationText = prediction.getExplainabilityData().getShapExplanation();
            if (shapExplanationText != null && !shapExplanationText.isEmpty()) {
                evidenceSources.add("ML Explanation: " + shapExplanationText);
            }
        }

        // Generate recommendations
        List<String> recommendations = generateRecommendations(prediction, severity, trend, threshold);

        // Generate clinical interpretation
        String clinicalInterpretation = generateClinicalInterpretation(prediction, severity, trend, shapExplanationText);

        return EnhancedAlert.builder()
            .patientId(prediction.getPatientId())
            .alertType(prediction.getModelType())
            .severity(severity)
            .timestamp(System.currentTimeMillis())
            .alertSource("ML_THRESHOLD")
            .confidence(prediction.getModelConfidence())
            .evidenceSources(evidenceSources)
            .mlPrediction(prediction)
            .shapExplanation(null)  // SHAP explanation not available as object, only as text
            .recommendations(recommendations)
            .clinicalInterpretation(clinicalInterpretation)
            .build();
    }

    private List<String> generateRecommendations(MLPrediction prediction,
                                                 String severity,
                                                 TrendAnalysis trend,
                                                 AlertThreshold threshold) {
        List<String> recommendations = new ArrayList<>();

        // Severity-based recommendations
        if ("CRITICAL".equals(severity)) {
            recommendations.add("Immediate clinical assessment required");
            recommendations.add("Consider ICU transfer or rapid response team activation");
            recommendations.add("Notify attending physician immediately");
        } else if ("HIGH".equals(severity)) {
            recommendations.add("Clinical assessment within 30 minutes");
            recommendations.add("Increase monitoring frequency");
            recommendations.add("Review recent vital signs and lab values");
        } else if ("MEDIUM".equals(severity)) {
            recommendations.add("Clinical review within 2 hours");
            recommendations.add("Continue routine monitoring");
        }

        // Trend-based recommendations
        if ("RISING".equals(trend.getDirection()) && Math.abs(trend.getSlope()) > 0.05) {
            recommendations.add("⚠️ Rapidly deteriorating - escalate care urgently");
        } else if ("RISING".equals(trend.getDirection())) {
            recommendations.add("Upward trend detected - monitor closely for deterioration");
        } else if ("FALLING".equals(trend.getDirection())) {
            recommendations.add("Improving trend - continue current management");
        }

        // Model-specific recommendations
        String modelType = prediction.getModelType();
        if (modelType.contains("sepsis")) {
            recommendations.add("Consider sepsis protocol: lactate, blood cultures, broad-spectrum antibiotics");
        } else if (modelType.contains("respiratory")) {
            recommendations.add("Assess respiratory status: oxygen saturation, respiratory rate, lung sounds");
        } else if (modelType.contains("cardiac")) {
            recommendations.add("Cardiac assessment: ECG, troponin, chest pain evaluation");
        }

        return recommendations;
    }

    private String generateClinicalInterpretation(MLPrediction prediction,
                                                  String severity,
                                                  TrendAnalysis trend,
                                                  String shapExplanationText) {
        StringBuilder interpretation = new StringBuilder();

        interpretation.append("ML model detected ");
        interpretation.append(severity.toLowerCase());
        interpretation.append(" risk of ");
        interpretation.append(prediction.getModelType().replace("_", " "));
        interpretation.append(" (score: ");
        interpretation.append(String.format("%.3f", prediction.getPrimaryScore()));
        interpretation.append("). ");

        if (shapExplanationText != null && !shapExplanationText.isEmpty()) {
            interpretation.append("ML Insights: ");
            interpretation.append(shapExplanationText.substring(0, Math.min(150, shapExplanationText.length())));
            interpretation.append(". ");
        }

        interpretation.append("Trend: ");
        interpretation.append(trend.getDirection().toLowerCase());
        if (Math.abs(trend.getChange()) > 0.1) {
            interpretation.append(" (change: ");
            interpretation.append(String.format("%+.3f", trend.getChange()));
            interpretation.append(")");
        }
        interpretation.append(".");

        return interpretation.toString();
    }

    /**
     * Update alert history for suppression tracking
     */
    private void updateAlertHistory(String patientId, String modelType, EnhancedAlert alert) throws Exception {
        Map<String, AlertHistory> alertHistory = alertHistoryState.value();
        if (alertHistory == null) {
            alertHistory = new HashMap<>();
        }

        MLPrediction prediction = alert.getMlPrediction();
        AlertHistory history = new AlertHistory(
            alert.getTimestamp(),
            alert.getSeverity(),
            prediction.getPrimaryScore(),
            prediction.getModelConfidence()
        );

        alertHistory.put(modelType, history);
        alertHistoryState.update(alertHistory);
    }

    // ===== Helper Classes =====

    /**
     * Trend analysis result
     */
    private static class TrendAnalysis implements Serializable {
        private final String direction;  // RISING, FALLING, STABLE
        private final double slope;
        private final double change;

        TrendAnalysis(String direction, double slope, double change) {
            this.direction = direction;
            this.slope = slope;
            this.change = change;
        }

        public String getDirection() { return direction; }
        public double getSlope() { return slope; }
        public double getChange() { return change; }
    }

    /**
     * Alert history for suppression
     */
    private static class AlertHistory implements Serializable {
        private final long timestamp;
        private final String severity;
        private final double score;
        private final double confidence;

        AlertHistory(long timestamp, String severity, double score, double confidence) {
            this.timestamp = timestamp;
            this.severity = severity;
            this.score = score;
            this.confidence = confidence;
        }

        public long getTimestamp() { return timestamp; }
        public String getSeverity() { return severity; }
        public double getScore() { return score; }
        public double getConfidence() { return confidence; }
    }
}
