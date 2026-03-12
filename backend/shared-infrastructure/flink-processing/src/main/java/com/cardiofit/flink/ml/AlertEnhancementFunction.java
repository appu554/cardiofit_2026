package com.cardiofit.flink.ml;

import com.cardiofit.flink.ml.explainability.SHAPCalculator;
import com.cardiofit.flink.ml.explainability.SHAPExplanation;
import com.cardiofit.flink.ml.features.ClinicalFeatureExtractor;
import com.cardiofit.flink.ml.features.ClinicalFeatureVector;
import com.cardiofit.flink.ml.features.FeatureNormalizer;
import com.cardiofit.flink.ml.features.FeatureValidator;
import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.streaming.api.functions.co.RichCoFlatMapFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * Alert Enhancement Function
 *
 * Merges CEP pattern-based alerts with ML risk predictions to create enhanced,
 * multi-dimensional clinical alerts. Combines rule-based pattern detection with
 * ML-based risk scoring for comprehensive clinical decision support.
 *
 * Integration Architecture:
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ Input Stream 1: CEP Pattern Alerts (Module 4)                           │
 * │ - Sepsis pattern detected                                               │
 * │ - Deterioration pattern detected                                        │
 * │ - Medication interaction detected                                       │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 *                   ┌─────────────────┐
 *                   │ State: Patient  │
 *                   │ Context Snapshot│
 *                   └─────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ Input Stream 2: ML Predictions (Module 5)                               │
 * │ - Sepsis risk: 0.82 (HIGH)                                              │
 * │ - SHAP explanation: lactate, WBC, temperature                           │
 * │ - Confidence: 0.91                                                      │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 *              ┌──────────────────────────────┐
 *              │ Alert Enhancement Logic      │
 *              │ - Merge CEP + ML insights    │
 *              │ - Calculate combined severity│
 *              │ - Generate recommendations   │
 *              │ - Add explainability         │
 *              └──────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ Output: Enhanced Alert                                                   │
 * │ - Alert type: SEPSIS_RISK                                               │
 * │ - Combined severity: CRITICAL (CEP: HIGH, ML: 0.82)                     │
 * │ - Evidence: Pattern + ML prediction + SHAP explanation                  │
 * │ - Recommendations: Start sepsis protocol, order lactate, broad abx      │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * State Management:
 * - Patient context snapshot (last known state from Module 2)
 * - Recent ML predictions (last 10 predictions per patient)
 * - Alert history (last 50 alerts per patient for deduplication)
 *
 * Enhancement Strategies:
 * 1. **Correlation**: When CEP pattern matches ML prediction → increase confidence
 * 2. **Contradiction**: When CEP pattern conflicts with ML → flag for review
 * 3. **Augmentation**: ML prediction without CEP pattern → add ML evidence
 * 4. **Validation**: CEP pattern without ML prediction → validate with features
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class AlertEnhancementFunction extends RichCoFlatMapFunction<PatternEvent, MLPrediction, EnhancedAlert> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(AlertEnhancementFunction.class);

    // State: Patient context snapshot
    private transient ValueState<PatientContextSnapshot> patientContextState;

    // State: Recent ML predictions (for correlation analysis)
    private transient ValueState<List<MLPrediction>> recentPredictionsState;

    // State: Alert history (for deduplication)
    private transient ValueState<List<EnhancedAlert>> alertHistoryState;

    // Feature extraction and ML components
    private transient ClinicalFeatureExtractor featureExtractor;
    private transient FeatureValidator featureValidator;
    private transient FeatureNormalizer featureNormalizer;
    private transient SHAPCalculator shapCalculator;

    // Configuration
    private final int maxRecentPredictions;
    private final int maxAlertHistory;
    private final long alertDeduplicationWindowMs;
    private final double mlThreshold;

    public AlertEnhancementFunction() {
        this(10, 50, 300_000, 0.7);  // 5-minute deduplication window
    }

    public AlertEnhancementFunction(int maxRecentPredictions,
                                   int maxAlertHistory,
                                   long alertDeduplicationWindowMs,
                                   double mlThreshold) {
        this.maxRecentPredictions = maxRecentPredictions;
        this.maxAlertHistory = maxAlertHistory;
        this.alertDeduplicationWindowMs = alertDeduplicationWindowMs;
        this.mlThreshold = mlThreshold;
    }

    @Override
    public void open(OpenContext context) throws Exception {
        super.open(context);

        // Initialize state
        patientContextState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("patient-context", PatientContextSnapshot.class)
        );

        recentPredictionsState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("recent-predictions",
                TypeInformation.of(new TypeHint<List<MLPrediction>>(){}))
        );

        alertHistoryState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("alert-history",
                TypeInformation.of(new TypeHint<List<EnhancedAlert>>(){}))
        );

        // Initialize feature extraction components
        featureExtractor = new ClinicalFeatureExtractor();
        featureValidator = new FeatureValidator();
        featureNormalizer = new FeatureNormalizer();

        // Initialize SHAP calculator with empty feature names list (will use default features)
        shapCalculator = new SHAPCalculator(Arrays.asList(
            "vital_heart_rate", "vital_systolic_bp", "vital_respiratory_rate",
            "vital_temperature_c", "lab_lactate_mmol", "lab_wbc_k_ul"
        ));

        LOG.info("AlertEnhancementFunction initialized: maxRecentPredictions={}, " +
                "maxAlertHistory={}, deduplicationWindow={}ms, mlThreshold={}",
            maxRecentPredictions, maxAlertHistory, alertDeduplicationWindowMs, mlThreshold);
    }

    /**
     * Process CEP pattern alert (Input Stream 1)
     *
     * When a CEP pattern is detected, check for recent ML predictions
     * and enhance the alert with ML insights
     */
    @Override
    public void flatMap1(PatternEvent patternEvent, Collector<EnhancedAlert> out) throws Exception {
        long startTime = System.nanoTime();

        String patientId = patternEvent.getPatientId();
        LOG.debug("Processing CEP pattern alert for patient {}: {}", patientId, patternEvent.getPatternName());

        // Get recent ML predictions for this patient
        List<MLPrediction> recentPredictions = recentPredictionsState.value();
        if (recentPredictions == null) {
            recentPredictions = new ArrayList<>();
        }

        // Find most recent relevant ML prediction
        MLPrediction relevantPrediction = findRelevantMLPrediction(patternEvent, recentPredictions);

        // Get patient context
        PatientContextSnapshot patientContext = patientContextState.value();

        // Create enhanced alert
        EnhancedAlert enhancedAlert;
        if (relevantPrediction != null) {
            // Strategy 1: Correlation - CEP pattern + ML prediction
            enhancedAlert = createCorrelatedAlert(patternEvent, relevantPrediction, patientContext);
            LOG.info("Created correlated alert (CEP + ML) for patient {}: severity={}",
                patientId, enhancedAlert.getSeverity());
        } else {
            // Strategy 4: Validation - CEP pattern without ML prediction
            enhancedAlert = createValidatedAlert(patternEvent, patientContext);
            LOG.info("Created validated alert (CEP only) for patient {}: severity={}",
                patientId, enhancedAlert.getSeverity());
        }

        // Check for duplicate alerts
        if (!isDuplicateAlert(enhancedAlert)) {
            // Add to alert history
            addToAlertHistory(enhancedAlert);

            // Emit enhanced alert
            out.collect(enhancedAlert);

            double processingTimeMs = (System.nanoTime() - startTime) / 1_000_000.0;
            LOG.debug("Enhanced alert created for patient {} in {:.2f}ms", patientId, processingTimeMs);
        } else {
            LOG.debug("Skipping duplicate alert for patient {}", patientId);
        }
    }

    /**
     * Process ML prediction (Input Stream 2)
     *
     * When an ML prediction is received:
     * 1. Store it for future correlation with CEP patterns
     * 2. If prediction exceeds threshold, create ML-based alert
     */
    @Override
    public void flatMap2(MLPrediction mlPrediction, Collector<EnhancedAlert> out) throws Exception {
        long startTime = System.nanoTime();

        String patientId = mlPrediction.getPatientId();
        LOG.debug("Processing ML prediction for patient {}: score={}", patientId, mlPrediction.getPrimaryScore());

        // Store prediction for future correlation
        List<MLPrediction> recentPredictions = recentPredictionsState.value();
        if (recentPredictions == null) {
            recentPredictions = new ArrayList<>();
        }

        // Add new prediction and maintain window
        recentPredictions.add(mlPrediction);
        if (recentPredictions.size() > maxRecentPredictions) {
            recentPredictions.remove(0);  // Remove oldest
        }
        recentPredictionsState.update(recentPredictions);

        // Get patient context
        PatientContextSnapshot patientContext = patientContextState.value();

        // If ML prediction exceeds threshold, create alert
        if (mlPrediction.getPrimaryScore() >= mlThreshold) {
            // Strategy 3: Augmentation - ML prediction without CEP pattern
            EnhancedAlert enhancedAlert = createMLBasedAlert(mlPrediction, patientContext);

            // Check for duplicate alerts
            if (!isDuplicateAlert(enhancedAlert)) {
                // Add to alert history
                addToAlertHistory(enhancedAlert);

                // Emit enhanced alert
                out.collect(enhancedAlert);

                LOG.info("Created ML-based alert for patient {}: severity={}, score={}",
                    patientId, enhancedAlert.getSeverity(), mlPrediction.getPrimaryScore());
            } else {
                LOG.debug("Skipping duplicate ML alert for patient {}", patientId);
            }
        }

        double processingTimeMs = (System.nanoTime() - startTime) / 1_000_000.0;
        LOG.debug("ML prediction processed for patient {} in {:.2f}ms", patientId, processingTimeMs);
    }

    /**
     * Find most recent ML prediction relevant to CEP pattern
     */
    private MLPrediction findRelevantMLPrediction(PatternEvent patternEvent,
                                                  List<MLPrediction> recentPredictions) {
        String patternName = patternEvent.getPatternName();
        long patternTimestamp = patternEvent.getTimestamp().toEpochMilli();

        // Find predictions within 5 minutes of pattern detection
        long timeWindow = 300_000;  // 5 minutes

        MLPrediction bestMatch = null;
        double bestScore = 0.0;

        for (MLPrediction prediction : recentPredictions) {
            // Check time proximity
            long timeDiff = Math.abs(prediction.getTimestamp().toEpochMilli() - patternTimestamp);
            if (timeDiff > timeWindow) {
                continue;
            }

            // Check model type relevance
            String modelType = prediction.getModelType();
            if (isRelevantModelType(patternName, modelType)) {
                if (prediction.getPrimaryScore() > bestScore) {
                    bestScore = prediction.getPrimaryScore();
                    bestMatch = prediction;
                }
            }
        }

        return bestMatch;
    }

    /**
     * Check if ML model type is relevant to CEP pattern
     */
    private boolean isRelevantModelType(String patternName, String modelType) {
        // Map CEP patterns to ML model types
        if (patternName.contains("sepsis") && modelType.contains("sepsis")) {
            return true;
        }
        if (patternName.contains("deterioration") && modelType.contains("deterioration")) {
            return true;
        }
        if (patternName.contains("medication") && modelType.contains("medication")) {
            return true;
        }
        if (patternName.contains("respiratory") && modelType.contains("respiratory")) {
            return true;
        }
        if (patternName.contains("cardiac") && modelType.contains("cardiac")) {
            return true;
        }

        // Generic risk models match all patterns
        return modelType.contains("risk") || modelType.contains("general");
    }

    /**
     * Create correlated alert (CEP pattern + ML prediction)
     */
    private EnhancedAlert createCorrelatedAlert(PatternEvent patternEvent,
                                               MLPrediction mlPrediction,
                                               PatientContextSnapshot patientContext) {
        // Calculate combined severity
        String cepSeverity = patternEvent.getSeverity();
        double mlScore = mlPrediction.getPrimaryScore();
        String combinedSeverity = calculateCombinedSeverity(cepSeverity, mlScore);

        // Merge evidence sources
        List<String> evidenceSources = new ArrayList<>();
        evidenceSources.add("CEP Pattern: " + patternEvent.getPatternName());
        evidenceSources.add("ML Model: " + mlPrediction.getModelType() + " (score: " + mlScore + ")");

        // Get SHAP explanation if available (from explanationText field)
        String shapExplanationText = mlPrediction.getExplainabilityData() != null ?
            mlPrediction.getExplainabilityData().getShapExplanation() : null;

        if (shapExplanationText != null && !shapExplanationText.isEmpty()) {
            evidenceSources.add("ML Explanation: " + shapExplanationText);
        }

        // Generate recommendations
        List<String> recommendations = generateRecommendations(
            patternEvent, mlPrediction, combinedSeverity, patientContext
        );

        return EnhancedAlert.builder()
            .patientId(patternEvent.getPatientId())
            .alertType(patternEvent.getPatternName())
            .severity(combinedSeverity)
            .timestamp(System.currentTimeMillis())
            .evidenceSources(evidenceSources)
            .recommendations(recommendations)
            .cepPattern(patternEvent)
            .mlPrediction(mlPrediction)
            .shapExplanation(null)  // SHAP explanation not available as object, only as text
            .alertSource("CORRELATED")
            .confidence(calculateConfidence(patternEvent, mlPrediction))
            .build();
    }

    /**
     * Create validated alert (CEP pattern without ML prediction)
     */
    private EnhancedAlert createValidatedAlert(PatternEvent patternEvent,
                                              PatientContextSnapshot patientContext) {
        List<String> evidenceSources = new ArrayList<>();
        evidenceSources.add("CEP Pattern: " + patternEvent.getPatternName());
        evidenceSources.add("Pattern Confidence: " + patternEvent.getConfidence());

        List<String> recommendations = generateRecommendations(
            patternEvent, null, patternEvent.getSeverity(), patientContext
        );

        return EnhancedAlert.builder()
            .patientId(patternEvent.getPatientId())
            .alertType(patternEvent.getPatternName())
            .severity(patternEvent.getSeverity())
            .timestamp(System.currentTimeMillis())
            .evidenceSources(evidenceSources)
            .recommendations(recommendations)
            .cepPattern(patternEvent)
            .alertSource("CEP_ONLY")
            .confidence(patternEvent.getConfidence())
            .build();
    }

    /**
     * Create ML-based alert (ML prediction without CEP pattern)
     */
    private EnhancedAlert createMLBasedAlert(MLPrediction mlPrediction,
                                            PatientContextSnapshot patientContext) {
        double mlScore = mlPrediction.getPrimaryScore();
        String severity = calculateMLSeverity(mlScore);

        List<String> evidenceSources = new ArrayList<>();
        evidenceSources.add("ML Model: " + mlPrediction.getModelType() + " (score: " + mlScore + ")");

        // Get SHAP explanation if available (from explanationText field)
        String shapExplanationText = mlPrediction.getExplainabilityData() != null ?
            mlPrediction.getExplainabilityData().getShapExplanation() : null;

        if (shapExplanationText != null && !shapExplanationText.isEmpty()) {
            evidenceSources.add("ML Explanation: " + shapExplanationText);
        }

        List<String> recommendations = generateRecommendations(
            null, mlPrediction, severity, patientContext
        );

        return EnhancedAlert.builder()
            .patientId(mlPrediction.getPatientId())
            .alertType(mlPrediction.getModelType())
            .severity(severity)
            .timestamp(System.currentTimeMillis())
            .evidenceSources(evidenceSources)
            .recommendations(recommendations)
            .mlPrediction(mlPrediction)
            .shapExplanation(null)  // SHAP explanation not available as object, only as text
            .alertSource("ML_ONLY")
            .confidence(mlPrediction.getModelConfidence())
            .build();
    }

    /**
     * Calculate combined severity from CEP pattern and ML score
     */
    private String calculateCombinedSeverity(String cepSeverity, double mlScore) {
        int cepLevel = severityToLevel(cepSeverity);
        int mlLevel = scoreToLevel(mlScore);

        // Take maximum severity
        int combinedLevel = Math.max(cepLevel, mlLevel);

        return levelToSeverity(combinedLevel);
    }

    private int severityToLevel(String severity) {
        switch (severity.toUpperCase()) {
            case "CRITICAL": return 4;
            case "HIGH": return 3;
            case "MEDIUM": return 2;
            case "LOW": return 1;
            default: return 0;
        }
    }

    private int scoreToLevel(double score) {
        if (score >= 0.9) return 4;  // CRITICAL
        if (score >= 0.75) return 3;  // HIGH
        if (score >= 0.5) return 2;   // MEDIUM
        if (score >= 0.3) return 1;   // LOW
        return 0;
    }

    private String levelToSeverity(int level) {
        switch (level) {
            case 4: return "CRITICAL";
            case 3: return "HIGH";
            case 2: return "MEDIUM";
            case 1: return "LOW";
            default: return "INFO";
        }
    }

    private String calculateMLSeverity(double mlScore) {
        return levelToSeverity(scoreToLevel(mlScore));
    }

    /**
     * Calculate confidence score for correlated alert
     */
    private double calculateConfidence(PatternEvent patternEvent, MLPrediction mlPrediction) {
        double cepConfidence = patternEvent.getConfidence();
        double mlConfidence = mlPrediction.getModelConfidence();

        // Average confidence when both sources agree
        return (cepConfidence + mlConfidence) / 2.0;
    }

    /**
     * Generate clinical recommendations based on alert context
     */
    private List<String> generateRecommendations(PatternEvent patternEvent,
                                                 MLPrediction mlPrediction,
                                                 String severity,
                                                 PatientContextSnapshot patientContext) {
        List<String> recommendations = new ArrayList<>();

        // Add pattern-specific recommendations
        if (patternEvent != null) {
            String patternName = patternEvent.getPatternName();
            if (patternName.contains("sepsis")) {
                recommendations.add("Initiate sepsis protocol immediately");
                recommendations.add("Order stat lactate and blood cultures");
                recommendations.add("Consider broad-spectrum antibiotics");
            } else if (patternName.contains("deterioration")) {
                recommendations.add("Rapid response team activation");
                recommendations.add("Increase monitoring frequency");
                recommendations.add("Notify attending physician");
            } else if (patternName.contains("medication")) {
                recommendations.add("Review medication orders for interactions");
                recommendations.add("Consult pharmacy for dosing recommendations");
            }
        }

        // Add ML-specific recommendations from explanation text if available
        if (mlPrediction != null && mlPrediction.getExplainabilityData() != null) {
            String shapText = mlPrediction.getExplainabilityData().getShapExplanation();
            if (shapText != null && !shapText.isEmpty()) {
                // Extract key insights from SHAP explanation text
                recommendations.add("ML Insights: " + shapText.substring(0, Math.min(200, shapText.length())));
            }
        }

        // Add severity-specific recommendations
        if ("CRITICAL".equals(severity)) {
            recommendations.add("Consider ICU transfer");
            recommendations.add("Notify department supervisor");
        }

        return recommendations;
    }

    /**
     * Check if alert is duplicate of recent alert
     */
    private boolean isDuplicateAlert(EnhancedAlert alert) throws Exception {
        List<EnhancedAlert> alertHistory = alertHistoryState.value();
        if (alertHistory == null || alertHistory.isEmpty()) {
            return false;
        }

        long currentTime = System.currentTimeMillis();

        for (EnhancedAlert historicalAlert : alertHistory) {
            // Check time window
            long timeDiff = currentTime - historicalAlert.getTimestamp();
            if (timeDiff > alertDeduplicationWindowMs) {
                continue;  // Outside deduplication window
            }

            // Check alert type match
            if (alert.getAlertType().equals(historicalAlert.getAlertType()) &&
                alert.getSeverity().equals(historicalAlert.getSeverity())) {
                return true;  // Duplicate found
            }
        }

        return false;
    }

    /**
     * Add alert to history for deduplication
     */
    private void addToAlertHistory(EnhancedAlert alert) throws Exception {
        List<EnhancedAlert> alertHistory = alertHistoryState.value();
        if (alertHistory == null) {
            alertHistory = new ArrayList<>();
        }

        // Add new alert
        alertHistory.add(alert);

        // Maintain window size
        if (alertHistory.size() > maxAlertHistory) {
            alertHistory.remove(0);  // Remove oldest
        }

        alertHistoryState.update(alertHistory);
    }
}
