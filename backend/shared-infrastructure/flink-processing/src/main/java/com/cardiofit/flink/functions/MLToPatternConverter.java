package com.cardiofit.flink.functions;

import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.models.PatternEvent;
import org.apache.flink.api.common.functions.MapFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Converts ML predictions from Module 5 to PatternEvent format.
 *
 * This converter allows ML predictions to be treated identically to
 * Layer 1 (instant state) and Layer 2 (CEP) patterns in the unified
 * alert routing and deduplication pipeline.
 *
 * Features:
 * - Maps ML risk categories to pattern severities
 * - Calculates priority and urgency from prediction horizon
 * - Builds human-readable clinical messages
 * - Generates condition-specific recommended actions
 * - Preserves explainability information
 *
 * @author CardioFit ML Integration Team
 * @version 1.0.0
 */
public class MLToPatternConverter implements MapFunction<MLPrediction, PatternEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(MLToPatternConverter.class);

    @Override
    public PatternEvent map(MLPrediction prediction) throws Exception {

        long startTime = System.nanoTime();

        PatternEvent pattern = new PatternEvent();

        // ═══════════════════════════════════════════════════════════
        // PATTERN IDENTIFICATION
        // ═══════════════════════════════════════════════════════════

        pattern.setId(UUID.randomUUID().toString());

        // Pattern type includes "PREDICTIVE_" prefix to distinguish from Layer 1/2
        String modelType = prediction.getModelType();
        String patternType = "PREDICTIVE_" + (modelType != null ? modelType : "RISK");
        pattern.setPatternType(patternType);

        pattern.setPatientId(prediction.getPatientId());
        pattern.setEncounterId(prediction.getEncounterId());
        pattern.setCorrelationId("ML-" + prediction.getId());

        // ═══════════════════════════════════════════════════════════
        // TEMPORAL CONTEXT
        // ═══════════════════════════════════════════════════════════

        long timestamp = prediction.getPredictionTime();
        pattern.setDetectionTime(timestamp);
        pattern.setPatternStartTime(timestamp);

        // Future prediction time (when condition is expected to manifest)
        // For now, use prediction time as end time (will be enhanced with time horizon)
        pattern.setPatternEndTime(timestamp);

        // ═══════════════════════════════════════════════════════════
        // SEVERITY & CONFIDENCE
        // ═══════════════════════════════════════════════════════════

        String riskLevel = prediction.getRiskLevel();
        String severity = mapRiskLevelToSeverity(riskLevel);
        pattern.setSeverity(severity);

        // ML confidence becomes pattern confidence
        Double confidence = prediction.getConfidence();
        pattern.setConfidence(confidence != null ? confidence : 0.70);

        // ═══════════════════════════════════════════════════════════
        // PRIORITY & URGENCY
        // ═══════════════════════════════════════════════════════════

        // Priority auto-calculated from severity (1-4 scale)
        pattern.setPriority(calculatePriorityFromSeverity(severity));

        // Urgency based on risk level and confidence (stored in pattern details)
        String urgency = determineUrgency(riskLevel, confidence);

        // ═══════════════════════════════════════════════════════════
        // INVOLVED EVENTS
        // ═══════════════════════════════════════════════════════════

        // ML predictions don't have a single triggering event
        // Use model identifier instead
        String modelVersion = prediction.getModelName();
        pattern.addInvolvedEvent("ML_MODEL_" + (modelVersion != null ? modelVersion : "UNKNOWN"));

        // ═══════════════════════════════════════════════════════════
        // PATTERN DETAILS (EXTENDED INFORMATION)
        // ═══════════════════════════════════════════════════════════

        Map<String, Object> details = new HashMap<>();

        // ML-specific metadata
        details.put("modelType", modelType);
        details.put("modelName", modelVersion);
        details.put("urgency", urgency);

        // Prediction scores
        if (prediction.getPredictionScores() != null) {
            details.put("predictionScores", prediction.getPredictionScores());
        }

        // Risk level and confidence
        details.put("riskLevel", riskLevel);
        details.put("confidence", confidence);
        details.put("confidenceInterval", prediction.getConfidenceInterval());

        // Feature importance (explainability)
        if (prediction.getFeatureImportance() != null) {
            details.put("featureImportance", prediction.getFeatureImportance());
        }

        // Input feature count
        details.put("inputFeatureCount", prediction.getInputFeatureCount());

        // Temporal context markers
        details.put("temporalContext", "PREDICTIVE");
        details.put("isAcute", false);
        details.put("isPredictive", true);
        details.put("predictionSource", "MODULE_5_ML");

        // Human-readable clinical message
        String clinicalMessage = buildMLClinicalMessage(prediction);
        details.put("clinicalMessage", clinicalMessage);

        pattern.setPatternDetails(details);

        // ═══════════════════════════════════════════════════════════
        // RECOMMENDED ACTIONS
        // ═══════════════════════════════════════════════════════════

        List<String> actions = buildMLRecommendedActions(prediction);
        pattern.setRecommendedActions(actions);

        // ═══════════════════════════════════════════════════════════
        // TAGS
        // ═══════════════════════════════════════════════════════════

        Set<String> tags = new HashSet<>();
        tags.add("ML_BASED");
        tags.add("PREDICTIVE");
        tags.add("LAYER_3");
        if (modelVersion != null) {
            tags.add("MODEL_" + modelVersion.replaceAll("[^A-Za-z0-9_]", "_"));
        }

        if (confidence != null && confidence >= 0.85) {
            tags.add("HIGH_CONFIDENCE");
        }

        if ("IMMEDIATE".equals(urgency)) {
            tags.add("URGENT");
        }

        pattern.setTags(tags);

        // ═══════════════════════════════════════════════════════════
        // PATTERN METADATA
        // ═══════════════════════════════════════════════════════════

        PatternEvent.PatternMetadata metadata = new PatternEvent.PatternMetadata();
        metadata.setAlgorithm("ML_PREDICTIVE_ANALYSIS");
        metadata.setVersion(modelVersion != null ? modelVersion : "1.0.0");

        Map<String, Object> algorithmParams = new HashMap<>();
        algorithmParams.put("modelType", modelType);
        algorithmParams.put("confidenceThreshold", 0.70);
        metadata.setAlgorithmParameters(algorithmParams);

        long endTime = System.nanoTime();
        double processingTimeMs = (endTime - startTime) / 1_000_000.0;
        metadata.setProcessingTime(processingTimeMs);

        // Quality score based on ML confidence
        String qualityScore = determineQualityScore(confidence);
        metadata.setQualityScore(qualityScore);

        pattern.setPatternMetadata(metadata);

        LOG.debug("✅ ML PATTERN for patient {}: type={}, confidence={:.3f}, risk={}, urgency={}, processingTime={:.2f}ms",
            prediction.getPatientId(),
            patternType,
            confidence,
            riskLevel,
            urgency,
            processingTimeMs);

        return pattern;
    }

    // ═══════════════════════════════════════════════════════════
    // HELPER METHODS
    // ═══════════════════════════════════════════════════════════

    private String mapRiskLevelToSeverity(String riskLevel) {
        if (riskLevel == null) return "MODERATE";

        switch (riskLevel.toUpperCase()) {
            case "CRITICAL":
                return "CRITICAL";
            case "HIGH":
                return "HIGH";
            case "MEDIUM":
            case "MODERATE":
                return "MODERATE";
            case "LOW":
                return "LOW";
            default:
                return "MODERATE";
        }
    }

    private int calculatePriorityFromSeverity(String severity) {
        switch (severity.toUpperCase()) {
            case "CRITICAL":
                return 1; // Highest priority
            case "HIGH":
                return 2;
            case "MODERATE":
                return 3;
            case "LOW":
            default:
                return 4; // Lowest priority
        }
    }

    private String determineUrgency(String riskLevel, Double confidence) {
        if (riskLevel == null) return "MODERATE";

        // Immediate: Critical risk with high confidence
        if ("CRITICAL".equalsIgnoreCase(riskLevel) &&
            confidence != null && confidence >= 0.85) {
            return "IMMEDIATE";
        }

        // Urgent: High risk OR critical with lower confidence
        if ("HIGH".equalsIgnoreCase(riskLevel) ||
            "CRITICAL".equalsIgnoreCase(riskLevel)) {
            return "URGENT";
        }

        // Moderate: Everything else
        return "MODERATE";
    }

    private String buildMLClinicalMessage(MLPrediction prediction) {
        StringBuilder message = new StringBuilder();
        message.append("ML PREDICTION: ");

        String modelType = prediction.getModelType();
        if (modelType != null) {
            message.append(modelType.replace("_", " "));
        } else {
            message.append("Risk assessment");
        }

        String riskLevel = prediction.getRiskLevel();
        if (riskLevel != null) {
            message.append(" - ").append(riskLevel).append(" risk detected");
        }

        Double confidence = prediction.getConfidence();
        if (confidence != null) {
            message.append(String.format(". Model confidence: %.0f%%", confidence * 100));
        }

        // Add feature importance if available
        Map<String, Double> featureImportance = prediction.getFeatureImportance();
        if (featureImportance != null && !featureImportance.isEmpty()) {
            message.append(". Key indicators: ");

            // Get top 3 features
            List<String> topFeatures = new ArrayList<>();
            featureImportance.entrySet().stream()
                .sorted(Map.Entry.<String, Double>comparingByValue().reversed())
                .limit(3)
                .forEach(entry -> topFeatures.add(
                    entry.getKey() + "=" + String.format("%.2f", entry.getValue())
                ));

            message.append(String.join(", ", topFeatures));
        }

        return message.toString();
    }

    private List<String> buildMLRecommendedActions(MLPrediction prediction) {
        List<String> actions = new ArrayList<>();

        String riskLevel = prediction.getRiskLevel();
        String modelType = prediction.getModelType();

        // Generic predictive actions
        actions.add("ENHANCED_MONITORING");
        actions.add("REASSESS_IN_4_HOURS");

        // Risk level specific actions
        if ("CRITICAL".equalsIgnoreCase(riskLevel)) {
            actions.add("IMMEDIATE_CLINICAL_ASSESSMENT");
            actions.add("ESCALATE_TO_RAPID_RESPONSE");
            actions.add("NOTIFY_SENIOR_CLINICIAN");
        } else if ("HIGH".equalsIgnoreCase(riskLevel)) {
            actions.add("CLINICAL_ASSESSMENT_REQUIRED");
            actions.add("INCREASE_MONITORING_FREQUENCY");
            actions.add("NOTIFY_CARE_TEAM");
        } else {
            actions.add("ROUTINE_MONITORING");
        }

        // Model type specific actions
        if (modelType != null) {
            if (modelType.contains("SEPSIS")) {
                actions.add("MONITOR_FOR_SIRS_CRITERIA");
                actions.add("CHECK_LACTATE_LEVEL");
                actions.add("PREPARE_FOR_SEPSIS_BUNDLE");
            } else if (modelType.contains("RESPIRATORY")) {
                actions.add("MONITOR_OXYGEN_SATURATION_CLOSELY");
                actions.add("ASSESS_RESPIRATORY_RATE_Q15MIN");
                actions.add("PREPARE_RESPIRATORY_SUPPORT");
            } else if (modelType.contains("CARDIAC")) {
                actions.add("CONTINUOUS_CARDIAC_MONITORING");
                actions.add("CHECK_TROPONIN_LEVELS");
                actions.add("ECG_IF_NOT_RECENT");
            } else if (modelType.contains("AKI") || modelType.contains("RENAL")) {
                actions.add("MONITOR_URINE_OUTPUT");
                actions.add("CHECK_CREATININE_LEVELS");
                actions.add("REVIEW_NEPHROTOXIC_MEDICATIONS");
            }
        }

        // Confidence-based actions
        Double confidence = prediction.getConfidence();
        if (confidence != null && confidence >= 0.85) {
            actions.add("HIGH_CONFIDENCE_PREDICTION_NOTIFY_SENIOR_CLINICIAN");
        }

        return actions;
    }

    private String determineQualityScore(Double confidence) {
        if (confidence == null) return "MODERATE";
        if (confidence >= 0.85) return "HIGH";
        if (confidence >= 0.70) return "MODERATE";
        return "LOW";
    }
}
