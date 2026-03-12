# Layer 3 ML Implementation Guide - Hybrid Architecture

**Document Date**: November 1, 2025
**Status**: 📋 Design Document - Implementation Ready
**Approach**: Hybrid (Module 5 ML Service + Module 4 Pattern Integration)

---

## 🎯 Executive Summary

**Recommendation**: Implement Layer 3 ML as a **hybrid system** split between Module 5 (ML inference) and Module 4 (pattern integration).

### Why Hybrid?

```
┌─────────────────────────────────────────────────────────────────┐
│                    LAYER 3 HYBRID ARCHITECTURE                   │
└─────────────────────────────────────────────────────────────────┘

MODULE 5 (ML Inference Service)              MODULE 4 (Pattern Integration)
├─ 🧠 ML Model Training                     ├─ 📥 Consume ML Predictions
├─ 🔬 Feature Engineering                   ├─ 🔄 Convert to PatternEvent
├─ ⚡ Model Serving (ONNX)                  ├─ 🔀 Merge with Layer 1 & 2
├─ 🎯 Batch/Real-time Inference             ├─ 🛡️ Deduplication
├─ 📊 Model Monitoring                      ├─ ⚖️ Priority Assignment
└─ 🌐 Prediction API                        └─ 📮 Unified Alert Routing

         │                                              │
         └──────────── Kafka Topic ─────────────────────┘
                  ehr-ml-predictions
```

### Benefits Matrix

| Concern | Module 5 Only | Module 4 Only | Hybrid ✅ |
|---------|--------------|--------------|----------|
| **Separation of Concerns** | ✅ ML isolated | ❌ ML in pattern detection | ✅ Clean separation |
| **Reusability** | ✅ ML serves other modules | ❌ Locked to Module 4 | ✅ ML reusable |
| **Testing** | ✅ ML testable independently | ❌ Hard to test ML separately | ✅ Both testable |
| **Scalability** | ✅ Scale ML independently | ❌ Coupled scaling | ✅ Independent scaling |
| **Maintenance** | ⚠️ Need schema sync | ✅ All in one place | ⚠️ Need coordination |
| **Latency** | ⚠️ Network hop | ✅ In-process | ⚠️ Acceptable (<50ms) |
| **Deduplication** | ❌ Can't merge with Layer 1/2 | ✅ Automatic merging | ✅ Automatic merging |
| **Multi-Source Confirmation** | ❌ Isolated predictions | ✅ Confirmation possible | ✅ Layer 1+2+3 merged |

---

## 🏗️ Architecture Overview

### Data Flow Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│  SEMANTIC EVENTS (Module 3)                                      │
│  comprehensive-cds-events.v1                                     │
└────────┬─────────────────────────────────────────────────────────┘
         │
         ├─────────────────────────────────────────────────────────┐
         │                                                          │
         ▼                                                          ▼
┌─────────────────────┐                               ┌──────────────────────┐
│  MODULE 4           │                               │  MODULE 5            │
│  Pattern Detection  │                               │  ML Prediction       │
│                     │                               │                      │
│  Layer 1: Instant   │                               │  Feature Extraction  │
│  Layer 2: CEP       │                               │  ↓                   │
└────────┬────────────┘                               │  ML Inference        │
         │                                            │  ↓                   │
         │                                            │  Prediction Output   │
         │                                            └──────────┬───────────┘
         │                                                       │
         │                                  ehr-ml-predictions   │
         │                                  (Kafka Topic)        │
         │                                                       │
         │                              ┌────────────────────────┘
         │                              │
         ▼                              ▼
┌────────────────────────────────────────────────────┐
│  MODULE 4 - Layer 3 ML Consumer                    │
│  ─────────────────────────────                     │
│  1. Read ML predictions from Kafka                 │
│  2. Convert MLPrediction → PatternEvent            │
│  3. Union with Layer 1 & Layer 2                   │
│  4. Deduplication & Multi-Source Confirmation      │
└──────────────────┬─────────────────────────────────┘
                   │
                   ▼
         ┌─────────────────────┐
         │  UNIFIED ALERTS     │
         │  ehr-alerts-module4 │
         │                     │
         │  ALL 3 LAYERS       │
         │  MERGED & DEDUPED   │
         └─────────────────────┘
```

---

## 📦 Part 1: Module 5 ML Prediction Service (New Flink Job)

### Overview

Module 5 runs as a **separate Flink job** focused exclusively on ML inference. It consumes semantic events, extracts features, runs models, and publishes predictions to Kafka.

### Implementation

#### File: `Module5_MLPredictionService.java`

```java
package com.cardiofit.flink.ml;

import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.MLPrediction;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;

/**
 * MODULE 5: ML PREDICTION SERVICE
 *
 * Separate Flink job for machine learning inference on clinical data.
 * Consumes semantic events from Module 3, runs ML models, produces predictions.
 *
 * Models:
 * - Sepsis Onset Prediction (6-12 hour horizon)
 * - Mortality Risk Assessment (30-day horizon)
 * - Respiratory Failure Prediction (2-4 hour horizon)
 * - AKI Progression Risk (24-48 hour horizon)
 *
 * Output: High-confidence predictions to Kafka for Module 4 consumption
 *
 * @author CardioFit ML Team
 * @version 1.0.0
 */
public class Module5_MLPredictionService {

    private static final Logger LOG = LoggerFactory.getLogger(Module5_MLPredictionService.class);

    public static void main(String[] args) throws Exception {

        LOG.info("🤖 MODULE 5: ML PREDICTION SERVICE - Starting");

        // ═══════════════════════════════════════════════════════════
        // ENVIRONMENT CONFIGURATION
        // ═══════════════════════════════════════════════════════════

        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Parallelism: ML inference can be CPU-intensive
        env.setParallelism(16);

        // Checkpointing for fault tolerance
        env.enableCheckpointing(60000); // 1 minute

        LOG.info("⚙️ Environment configured - parallelism: 16, checkpointing: 60s");

        // ═══════════════════════════════════════════════════════════
        // INPUT: SEMANTIC EVENTS FROM MODULE 3
        // ═══════════════════════════════════════════════════════════

        DataStream<SemanticEvent> semanticEvents = env
            .fromSource(
                KafkaSourceBuilder.createKafkaSource("comprehensive-cds-events.v1", SemanticEvent.class),
                WatermarkStrategy
                    .<SemanticEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                    .withTimestampAssigner((event, ts) -> event.getEventTime()),
                "semantic-events-input"
            )
            .uid("semantic-events-source")
            .name("Semantic Events from Module 3");

        // Key by patient for stateful operations
        DataStream<SemanticEvent> keyedSemanticEvents = semanticEvents
            .keyBy(SemanticEvent::getPatientId);

        LOG.info("📥 Semantic events source configured");

        // ═══════════════════════════════════════════════════════════
        // FEATURE ENGINEERING
        // ═══════════════════════════════════════════════════════════

        DataStream<FeatureVector> featureVectors = keyedSemanticEvents
            .process(new FeatureEngineeringFunction())
            .uid("feature-engineering")
            .name("Feature Engineering Pipeline");

        LOG.info("🔬 Feature engineering pipeline configured (70 features)");

        // ═══════════════════════════════════════════════════════════
        // ML MODEL INFERENCE (4 MODELS IN PARALLEL)
        // ═══════════════════════════════════════════════════════════

        // Model 1: Sepsis Prediction (6-12 hour horizon)
        DataStream<MLPrediction> sepsisPredictions = featureVectors
            .map(new SepsisPredictionModel())
            .uid("sepsis-prediction")
            .name("Sepsis Onset Prediction Model");

        // Model 2: Mortality Risk (30-day horizon)
        DataStream<MLPrediction> mortalityPredictions = featureVectors
            .map(new MortalityRiskModel())
            .uid("mortality-prediction")
            .name("30-Day Mortality Risk Model");

        // Model 3: Respiratory Failure (2-4 hour horizon)
        DataStream<MLPrediction> respiratoryPredictions = featureVectors
            .map(new RespiratoryFailurePredictionModel())
            .uid("respiratory-prediction")
            .name("Respiratory Failure Prediction Model");

        // Model 4: AKI Progression (24-48 hour horizon)
        DataStream<MLPrediction> akiPredictions = featureVectors
            .map(new AKIProgressionModel())
            .uid("aki-prediction")
            .name("AKI Progression Risk Model");

        LOG.info("🧠 ML models configured (4 models running in parallel)");

        // ═══════════════════════════════════════════════════════════
        // MERGE ALL PREDICTIONS
        // ═══════════════════════════════════════════════════════════

        DataStream<MLPrediction> allPredictions = sepsisPredictions
            .union(mortalityPredictions)
            .union(respiratoryPredictions)
            .union(akiPredictions)
            .name("All ML Predictions");

        // ═══════════════════════════════════════════════════════════
        // FILTER: ONLY HIGH-CONFIDENCE PREDICTIONS
        // ═══════════════════════════════════════════════════════════

        DataStream<MLPrediction> highConfidencePredictions = allPredictions
            .filter(pred -> {
                double confidence = pred.getPrediction().getConfidence();
                boolean highConfidence = confidence >= 0.70;

                if (highConfidence) {
                    LOG.debug("✅ High-confidence prediction: patient={}, type={}, confidence={:.3f}",
                        pred.getPatientId(),
                        pred.getModelType(),
                        confidence);
                } else {
                    LOG.trace("❌ Filtered low-confidence prediction: confidence={:.3f}", confidence);
                }

                return highConfidence;
            })
            .uid("high-confidence-filter")
            .name("High-Confidence Filter (≥0.70)");

        LOG.info("⚖️ Confidence filter configured (threshold: 0.70)");

        // ═══════════════════════════════════════════════════════════
        // OUTPUT: KAFKA TOPIC FOR MODULE 4 CONSUMPTION
        // ═══════════════════════════════════════════════════════════

        highConfidencePredictions.sinkTo(
            KafkaSinkBuilder.createKafkaSink("ehr-ml-predictions", MLPrediction.class)
        ).uid("ml-predictions-output")
         .name("ML Predictions to Kafka");

        LOG.info("📮 Output sink configured - topic: ehr-ml-predictions");

        // ═══════════════════════════════════════════════════════════
        // MONITORING & METRICS
        // ═══════════════════════════════════════════════════════════

        // Add prediction count metrics
        highConfidencePredictions
            .map(pred -> 1L)
            .keyBy(x -> "predictions")
            .sum(0)
            .name("Prediction Count Metric");

        LOG.info("📊 Monitoring metrics configured");

        // ═══════════════════════════════════════════════════════════
        // EXECUTE JOB
        // ═══════════════════════════════════════════════════════════

        LOG.info("🚀 Starting Module 5 ML Prediction Service");

        env.execute("Module 5: ML Prediction Service");
    }
}
```

---

### MLPrediction Data Model

```java
package com.cardiofit.flink.models;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.io.Serializable;
import java.util.List;
import java.util.Map;

/**
 * ML Prediction Output Format
 * Produced by Module 5, consumed by Module 4 Layer 3
 */
public class MLPrediction implements Serializable {

    @JsonProperty("predictionId")
    private String predictionId;

    @JsonProperty("patientId")
    private String patientId;

    @JsonProperty("encounterId")
    private String encounterId;

    @JsonProperty("modelType")
    private String modelType; // SEPSIS_ONSET_PREDICTION, MORTALITY_RISK, etc.

    @JsonProperty("modelVersion")
    private String modelVersion;

    @JsonProperty("prediction")
    private PredictionDetails prediction;

    @JsonProperty("features")
    private FeatureSummary features;

    @JsonProperty("explainability")
    private ExplainabilityInfo explainability;

    @JsonProperty("metadata")
    private PredictionMetadata metadata;

    // ═══════════════════════════════════════════════════════════
    // NESTED CLASSES
    // ═══════════════════════════════════════════════════════════

    public static class PredictionDetails implements Serializable {
        private String riskType; // SEPSIS, MORTALITY, RESPIRATORY_FAILURE, AKI
        private Double probability; // 0.0-1.0
        private Double confidence; // Model confidence in prediction
        private String riskCategory; // CRITICAL, HIGH, MODERATE, LOW
        private String timeHorizon; // "8_HOURS", "30_DAYS", "4_HOURS"
        private Long expectedOnsetTime; // Unix timestamp

        // Getters/setters...
    }

    public static class FeatureSummary implements Serializable {
        private Integer news2Score;
        private Integer qsofaScore;
        private Double lactate;
        private Double temperature;
        private Integer heartRate;
        private String trend; // IMPROVING, STABLE, WORSENING

        // Getters/setters...
    }

    public static class ExplainabilityInfo implements Serializable {
        private List<FeatureContribution> topFeatures;

        public static class FeatureContribution implements Serializable {
            private String name;
            private Double contribution; // SHAP value or feature importance

            // Getters/setters...
        }

        // Getters/setters...
    }

    public static class PredictionMetadata implements Serializable {
        private Double inferenceTime; // Milliseconds
        private Long timestamp; // Unix timestamp
        private String source; // "MODULE_5_ML"

        // Getters/setters...
    }

    // Getters/setters for all fields...
}
```

---

### Example ML Prediction Output

```json
{
  "predictionId": "pred-550e8400-e29b-41d4-a716-446655440000",
  "patientId": "PAT-001",
  "encounterId": "ENC-001",
  "modelType": "SEPSIS_ONSET_PREDICTION",
  "modelVersion": "v1.2.0",

  "prediction": {
    "riskType": "SEPSIS",
    "probability": 0.82,
    "confidence": 0.85,
    "riskCategory": "HIGH",
    "timeHorizon": "8_HOURS",
    "expectedOnsetTime": 1735718400000
  },

  "features": {
    "news2Score": 6,
    "qsofaScore": 1,
    "lactate": 2.3,
    "temperature": 38.5,
    "heartRate": 108,
    "trend": "WORSENING"
  },

  "explainability": {
    "topFeatures": [
      {"name": "lactate", "contribution": 0.15},
      {"name": "temperature", "contribution": 0.12},
      {"name": "news2Score", "contribution": 0.10},
      {"name": "heartRate", "contribution": 0.08},
      {"name": "trend", "contribution": 0.07}
    ]
  },

  "metadata": {
    "inferenceTime": 15.3,
    "timestamp": 1735689600000,
    "source": "MODULE_5_ML"
  }
}
```

---

## 📦 Part 2: Module 4 Layer 3 Integration (Extend Existing)

### Overview

Module 4's orchestrator needs a **small addition** (~450 lines) to:
1. Consume ML predictions from Kafka
2. Convert `MLPrediction` → `PatternEvent`
3. Union with Layer 1 & Layer 2
4. Let existing deduplication handle multi-source confirmation

### Implementation

Add to `Module4PatternOrchestrator.java`:

```java
/**
 * Layer 3: ML Predictive Analysis
 *
 * Consumes ML predictions from Module 5 and converts them to PatternEvent format.
 * This allows ML predictions to participate in multi-source confirmation and
 * deduplication alongside Layer 1 (instant state) and Layer 2 (CEP patterns).
 *
 * @param env Flink execution environment
 * @return Stream of ML-based pattern events
 */
private static DataStream<PatternEvent> mlPredictiveAnalysis(
        StreamExecutionEnvironment env) {

    LOG.info("🤖 Layer 3: ML Predictive Analysis - Activating predictive intelligence");

    // ═══════════════════════════════════════════════════════════
    // INPUT: ML PREDICTIONS FROM MODULE 5 (KAFKA)
    // ═══════════════════════════════════════════════════════════

    DataStream<MLPrediction> mlPredictions = env
        .fromSource(
            KafkaSourceBuilder.createKafkaSource("ehr-ml-predictions", MLPrediction.class),
            WatermarkStrategy
                .<MLPrediction>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((pred, ts) -> pred.getMetadata().getTimestamp()),
            "ml-predictions-input"
        )
        .uid("ml-predictions-source")
        .name("ML Predictions from Module 5");

    // Key by patient for stateful operations
    DataStream<MLPrediction> keyedMLPredictions = mlPredictions
        .keyBy(MLPrediction::getPatientId);

    LOG.info("📥 ML predictions source configured - consuming from ehr-ml-predictions");

    // ═══════════════════════════════════════════════════════════
    // CONVERT: MLPrediction → PatternEvent
    // ═══════════════════════════════════════════════════════════

    DataStream<PatternEvent> mlPatterns = keyedMLPredictions
        .map(new MLToPatternConverter())
        .uid("ml-to-pattern-conversion")
        .name("ML to Pattern Conversion");

    LOG.info("🔄 ML to PatternEvent converter configured");

    return mlPatterns;
}
```

---

### ML to Pattern Converter

```java
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
 */
public static class MLToPatternConverter
    implements MapFunction<MLPrediction, PatternEvent> {

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
        String patternType = "PREDICTIVE_" + prediction.getPrediction().getRiskType();
        pattern.setPatternType(patternType);

        pattern.setPatientId(prediction.getPatientId());
        pattern.setEncounterId(prediction.getEncounterId());
        pattern.setCorrelationId("ML-" + prediction.getPredictionId());

        // ═══════════════════════════════════════════════════════════
        // TEMPORAL CONTEXT
        // ═══════════════════════════════════════════════════════════

        pattern.setDetectionTime(prediction.getMetadata().getTimestamp());
        pattern.setPatternStartTime(prediction.getMetadata().getTimestamp());

        // Future prediction time (when condition is expected to manifest)
        long predictionHorizonMs = parsePredictionHorizon(
            prediction.getPrediction().getTimeHorizon()
        );
        pattern.setPatternEndTime(
            prediction.getMetadata().getTimestamp() + predictionHorizonMs
        );

        // ═══════════════════════════════════════════════════════════
        // SEVERITY & CONFIDENCE
        // ═══════════════════════════════════════════════════════════

        String severity = mapRiskCategoryToSeverity(
            prediction.getPrediction().getRiskCategory()
        );
        pattern.setSeverity(severity);

        // ML confidence becomes pattern confidence
        pattern.setConfidence(prediction.getPrediction().getConfidence());

        // ═══════════════════════════════════════════════════════════
        // PRIORITY & URGENCY
        // ═══════════════════════════════════════════════════════════

        // Priority auto-calculated from severity (1-4 scale)
        pattern.setPriority(calculatePriorityFromSeverity(severity));

        // Urgency based on time horizon and severity
        String urgency = determineUrgency(
            prediction.getPrediction().getTimeHorizon(),
            severity
        );
        pattern.setUrgency(urgency);

        // ═══════════════════════════════════════════════════════════
        // INVOLVED EVENTS
        // ═══════════════════════════════════════════════════════════

        // ML predictions don't have a single triggering event
        // Use model identifier instead
        pattern.addInvolvedEvent("ML_MODEL_" + prediction.getModelVersion());

        // ═══════════════════════════════════════════════════════════
        // PATTERN DETAILS (EXTENDED INFORMATION)
        // ═══════════════════════════════════════════════════════════

        Map<String, Object> details = new HashMap<>();

        // ML-specific metadata
        details.put("modelType", prediction.getModelType());
        details.put("modelVersion", prediction.getModelVersion());
        details.put("probability", prediction.getPrediction().getProbability());
        details.put("timeHorizon", prediction.getPrediction().getTimeHorizon());
        details.put("expectedOnsetTime", prediction.getPrediction().getExpectedOnsetTime());

        // Feature information (for clinical context)
        if (prediction.getFeatures() != null) {
            FeatureSummary features = prediction.getFeatures();
            details.put("news2Score", features.getNews2Score());
            details.put("qsofaScore", features.getQsofaScore());
            details.put("trend", features.getTrend());
            details.put("lactate", features.getLactate());
            details.put("temperature", features.getTemperature());
            details.put("heartRate", features.getHeartRate());
        }

        // Explainability (SHAP values, feature importance)
        if (prediction.getExplainability() != null) {
            details.put("featureImportance",
                prediction.getExplainability().getTopFeatures());
        }

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

        List<String> tags = new ArrayList<>();
        tags.add("ML_BASED");
        tags.add("PREDICTIVE");
        tags.add("LAYER_3");
        tags.add("MODEL_" + prediction.getModelVersion());

        if (prediction.getPrediction().getConfidence() >= 0.85) {
            tags.add("HIGH_CONFIDENCE");
        }

        if (urgency.equals("IMMEDIATE")) {
            tags.add("URGENT");
        }

        pattern.setTags(tags);

        // ═══════════════════════════════════════════════════════════
        // PATTERN METADATA
        // ═══════════════════════════════════════════════════════════

        PatternEvent.PatternMetadata metadata = new PatternEvent.PatternMetadata();
        metadata.setAlgorithm("ML_PREDICTIVE_ANALYSIS");
        metadata.setVersion(prediction.getModelVersion());

        Map<String, Object> algorithmParams = new HashMap<>();
        algorithmParams.put("modelType", prediction.getModelType());
        algorithmParams.put("confidenceThreshold", 0.70);
        algorithmParams.put("predictionHorizon", prediction.getPrediction().getTimeHorizon());
        metadata.setAlgorithmParameters(algorithmParams);

        long endTime = System.nanoTime();
        double processingTimeMs = (endTime - startTime) / 1_000_000.0;
        metadata.setProcessingTime(processingTimeMs);

        // Quality score based on ML confidence
        String qualityScore = determineQualityScore(prediction.getPrediction().getConfidence());
        metadata.setQualityScore(qualityScore);

        pattern.setPatternMetadata(metadata);

        LOG.debug("✅ ML PATTERN for patient {}: type={}, probability={:.3f}, confidence={:.3f}, horizon={}, urgency={}, processingTime={:.2f}ms",
            prediction.getPatientId(),
            patternType,
            prediction.getPrediction().getProbability(),
            prediction.getPrediction().getConfidence(),
            prediction.getPrediction().getTimeHorizon(),
            urgency,
            processingTimeMs);

        return pattern;
    }

    // ═══════════════════════════════════════════════════════════
    // HELPER METHODS
    // ═══════════════════════════════════════════════════════════

    private String mapRiskCategoryToSeverity(String riskCategory) {
        if (riskCategory == null) return "MODERATE";

        switch (riskCategory.toUpperCase()) {
            case "CRITICAL":
                return "CRITICAL";
            case "HIGH":
                return "HIGH";
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

    private String determineUrgency(String timeHorizon, String severity) {
        // Extract hours from time horizon (e.g., "8_HOURS" → 8)
        int hours = Integer.parseInt(timeHorizon.split("_")[0]);

        // Immediate: Critical + short horizon
        if (severity.equals("CRITICAL") && hours <= 4) {
            return "IMMEDIATE";
        }

        // Urgent: High severity OR short horizon
        if (severity.equals("HIGH") || hours <= 8) {
            return "URGENT";
        }

        // Moderate: Everything else
        return "MODERATE";
    }

    private long parsePredictionHorizon(String timeHorizon) {
        // Parse "8_HOURS" → 8 * 3600 * 1000 ms
        try {
            String[] parts = timeHorizon.split("_");
            int value = Integer.parseInt(parts[0]);

            if (timeHorizon.contains("HOURS")) {
                return value * 3600L * 1000L;
            } else if (timeHorizon.contains("DAYS")) {
                return value * 24L * 3600L * 1000L;
            }

            return 8 * 3600L * 1000L; // Default: 8 hours
        } catch (Exception e) {
            LOG.warn("Failed to parse time horizon: {}, using default 8 hours", timeHorizon);
            return 8 * 3600L * 1000L;
        }
    }

    private String buildMLClinicalMessage(MLPrediction prediction) {
        String riskType = prediction.getPrediction().getRiskType();
        double probability = prediction.getPrediction().getProbability();
        double confidence = prediction.getPrediction().getConfidence();
        String timeHorizon = prediction.getPrediction().getTimeHorizon()
            .replace("_", " ").toLowerCase();

        StringBuilder message = new StringBuilder();
        message.append("ML PREDICTION: ");
        message.append(riskType.replace("_", " "));
        message.append(" risk in next ");
        message.append(timeHorizon);
        message.append(". ");

        message.append(String.format("Risk probability: %.0f%%, ", probability * 100));
        message.append(String.format("Model confidence: %.0f%%. ", confidence * 100));

        // Add key clinical indicators if available
        if (prediction.getFeatures() != null) {
            FeatureSummary features = prediction.getFeatures();
            message.append("Key indicators: ");

            List<String> indicators = new ArrayList<>();
            if (features.getNews2Score() != null) {
                indicators.add("NEWS2=" + features.getNews2Score());
            }
            if (features.getQsofaScore() != null) {
                indicators.add("qSOFA=" + features.getQsofaScore());
            }
            if (features.getTrend() != null) {
                indicators.add("Trend=" + features.getTrend());
            }
            if (features.getLactate() != null) {
                indicators.add(String.format("Lactate=%.1f", features.getLactate()));
            }

            message.append(String.join(", ", indicators));
        }

        return message.toString();
    }

    private List<String> buildMLRecommendedActions(MLPrediction prediction) {
        List<String> actions = new ArrayList<>();

        String riskType = prediction.getPrediction().getRiskType();
        String timeHorizon = prediction.getPrediction().getTimeHorizon();
        int hours = Integer.parseInt(timeHorizon.split("_")[0]);

        // Generic predictive actions
        actions.add("ENHANCED_MONITORING");
        actions.add("REASSESS_IN_" + Math.max(1, hours / 2) + "_HOURS");

        // Risk-specific actions
        switch (riskType.toUpperCase()) {
            case "SEPSIS":
                actions.add("MONITOR_FOR_SIRS_CRITERIA");
                actions.add("CHECK_LACTATE_LEVEL");
                actions.add("MONITOR_VITAL_SIGNS_Q15MIN");
                actions.add("PREPARE_FOR_SEPSIS_BUNDLE");
                actions.add("NOTIFY_INFECTIOUS_DISEASE_TEAM");
                break;

            case "RESPIRATORY_FAILURE":
                actions.add("MONITOR_OXYGEN_SATURATION_CLOSELY");
                actions.add("ASSESS_RESPIRATORY_RATE_Q15MIN");
                actions.add("PREPARE_RESPIRATORY_SUPPORT");
                actions.add("NOTIFY_RESPIRATORY_THERAPY");
                actions.add("ARTERIAL_BLOOD_GAS_IF_NOT_RECENT");
                break;

            case "CARDIAC_EVENT":
                actions.add("CONTINUOUS_CARDIAC_MONITORING");
                actions.add("CHECK_TROPONIN_LEVELS");
                actions.add("ECG_IF_NOT_RECENT");
                actions.add("NOTIFY_CARDIOLOGY");
                actions.add("ASSESS_CHEST_PAIN_SYMPTOMS");
                break;

            case "AKI":
                actions.add("MONITOR_URINE_OUTPUT");
                actions.add("CHECK_CREATININE_LEVELS");
                actions.add("REVIEW_NEPHROTOXIC_MEDICATIONS");
                actions.add("ASSESS_FLUID_BALANCE");
                actions.add("NOTIFY_NEPHROLOGY_IF_WORSENING");
                break;

            case "MORTALITY":
                actions.add("ASSESS_ADVANCE_DIRECTIVES");
                actions.add("FAMILY_MEETING_RECOMMENDED");
                actions.add("PALLIATIVE_CARE_CONSULT");
                actions.add("GOALS_OF_CARE_DISCUSSION");
                break;

            default:
                actions.add("CLINICAL_ASSESSMENT_REQUIRED");
                actions.add("NOTIFY_CARE_TEAM");
        }

        // Confidence-based actions
        double confidence = prediction.getPrediction().getConfidence();
        if (confidence >= 0.85) {
            actions.add("HIGH_CONFIDENCE_PREDICTION_NOTIFY_SENIOR_CLINICIAN");
        }

        // Urgency-based actions
        if (hours <= 4) {
            actions.add("URGENT_INTERVENTION_WINDOW");
            actions.add("ESCALATE_TO_RAPID_RESPONSE");
        }

        return actions;
    }

    private String determineQualityScore(double confidence) {
        if (confidence >= 0.85) return "HIGH";
        if (confidence >= 0.70) return "MODERATE";
        return "LOW";
    }
}
```

---

### Update Orchestrator Main Method

Modify `Module4PatternOrchestrator.orchestrate()` to include Layer 3:

```java
public static DataStream<PatternEvent> orchestrate(
        DataStream<SemanticEvent> semanticEvents,
        StreamExecutionEnvironment env) {

    LOG.info("🎯 MODULE 4 PATTERN ORCHESTRATOR - Starting multi-layer pattern detection");

    // Key semantic events by patient ID for stateful operations
    KeyedStream<SemanticEvent, String> keyedSemanticEvents = semanticEvents
        .keyBy(SemanticEvent::getPatientId);

    // ═══════════════════════════════════════════════════════════
    // LAYER 1: INSTANT STATE ASSESSMENT (Triage Nurse)
    // ═══════════════════════════════════════════════════════════

    DataStream<PatternEvent> instantPatterns = instantStateAssessment(semanticEvents);

    // ═══════════════════════════════════════════════════════════
    // LAYER 2: COMPLEX EVENT PROCESSING (ICU Monitor)
    // ═══════════════════════════════════════════════════════════

    DataStream<PatternEvent> cepPatterns = cepPatternDetection(keyedSemanticEvents);

    // ═══════════════════════════════════════════════════════════
    // LAYER 3: ML PREDICTIVE ANALYSIS (Crystal Ball) ✅ NEW!
    // ═══════════════════════════════════════════════════════════

    DataStream<PatternEvent> mlPatterns = mlPredictiveAnalysis(env);

    // ═══════════════════════════════════════════════════════════
    // PATTERN UNIFICATION
    // ═══════════════════════════════════════════════════════════

    // Combine all pattern detection layers
    DataStream<PatternEvent> allPatterns = instantPatterns
        .union(cepPatterns)
        .union(mlPatterns)  // ✅ ML predictions now included!
        .name("All Pattern Streams (Layer 1 + 2 + 3)");

    LOG.info("✅ All 3 layers unified - instant state + CEP + ML predictions");

    // ═══════════════════════════════════════════════════════════
    // DEDUPLICATION & MULTI-SOURCE CONFIRMATION
    // ═══════════════════════════════════════════════════════════

    // Apply 5-minute deduplication window to prevent alert storms
    // This automatically handles multi-source confirmation when
    // Layer 1, Layer 2, and Layer 3 fire together!
    DataStream<PatternEvent> dedupedPatterns = allPatterns
        .keyBy(PatternEvent::getPatientId)
        .process(new PatternDeduplicationFunction())
        .uid("Pattern Deduplication")
        .name("Deduplicated Multi-Source Patterns (3 Layers)");

    LOG.info("✅ MODULE 4 PATTERN ORCHESTRATOR - Multi-layer pattern detection configured with ML");

    return dedupedPatterns;
}
```

---

## 🔄 Multi-Source Confirmation Example

### Scenario: Sepsis Development Over Time

```
T+0 hours: Patient stable baseline
  Layer 1: ❌ No alert (NEWS2=3, qSOFA=0)
  Layer 2: ❌ No pattern detected
  Layer 3: ❌ No ML prediction

  OUTPUT: None

─────────────────────────────────────────────────────────────

T+2 hours: ML detects early warning signs
  Layer 1: ❌ No alert (NEWS2=5, not critical yet)
  Layer 2: ❌ No pattern yet (need event sequence)
  Layer 3: ✅ "PREDICTIVE_SEPSIS in 8 hours" (confidence: 0.78)

  OUTPUT:
  {
    "patternType": "PREDICTIVE_SEPSIS",
    "severity": "HIGH",
    "confidence": 0.78,
    "urgency": "URGENT",
    "timeHorizon": "8_HOURS",
    "source": "LAYER_3_ML",
    "tags": ["ML_BASED", "PREDICTIVE", "LAYER_3"],
    "clinicalMessage": "ML PREDICTION: SEPSIS risk in next 8 hours. Risk probability: 82%, Model confidence: 78%. Key indicators: NEWS2=5, qSOFA=0, Trend=WORSENING, Lactate=2.1"
  }

─────────────────────────────────────────────────────────────

T+6 hours: Patient continues to deteriorate
  Layer 1: ⚠️ "HIGH_RISK_STATE_DETECTED" (NEWS2=7)
  Layer 2: ❌ Pattern still building
  Layer 3: ✅ Still predicting (now 4 hours until expected onset)

  DEDUPLICATION MERGES Layer 1 + Layer 3:
  {
    "patternType": "HIGH_RISK_STATE_DETECTED",
    "severity": "HIGH",
    "confidence": 0.81,  // Weighted: Layer 1 (0.84) + Layer 3 (0.78)
    "tags": ["MULTI_SOURCE_CONFIRMED", "ML_PREDICTED", "STATE_BASED"],
    "predictionStatus": "TRENDING_TOWARD_ML_PREDICTION"
  }

─────────────────────────────────────────────────────────────

T+8 hours: CEP detects deterioration pattern
  Layer 1: ⚠️ "HIGH_RISK_STATE_DETECTED" (NEWS2=8)
  Layer 2: ✅ "SEPSIS_DETERIORATION_PATTERN" (confidence: 0.86)
  Layer 3: ✅ Still predicting (now 2 hours until expected onset)

  DEDUPLICATION MERGES ALL 3 LAYERS:
  {
    "patternType": "SEPSIS_DETERIORATION_PATTERN",
    "severity": "HIGH",
    "confidence": 0.83,  // Weighted: Layer 1 (0.84) + Layer 2 (0.86) + Layer 3 (0.78)
    "tags": ["MULTI_SOURCE_CONFIRMED", "ALL_3_LAYERS", "ML_ACCURATE"],
    "predictionAccuracy": "ML_PREDICTED_6_HOURS_EARLY",
    "clinicalMessage": "SEPSIS DETERIORATION PATTERN detected. Multiple sources confirm. ML predicted 6 hours ago."
  }

─────────────────────────────────────────────────────────────

T+10 hours: Critical state reached (ML prediction onset time)
  Layer 1: 🚨 "SEPSIS_CRITERIA_MET" (qSOFA=2, confidence: 0.95)
  Layer 2: Pattern complete
  Layer 3: Expected onset time reached

  DEDUPLICATION MERGES - ALL LAYERS AGREE:
  {
    "patternType": "SEPSIS_CRITERIA_MET",
    "severity": "CRITICAL",
    "confidence": 0.91,  // All 3 layers agree = HIGH confidence
    "priority": 1,
    "urgency": "IMMEDIATE",
    "tags": ["MULTI_SOURCE_CONFIRMED", "ALL_LAYERS", "ML_ACCURATE", "CRITICAL"],
    "timeline": {
      "mlPredictionTime": "T+2h (8 hours early warning)",
      "layer1DetectionTime": "T+6h (4 hours early warning)",
      "cepDetectionTime": "T+8h (2 hours confirmation)",
      "criticalStateTime": "T+10h (ML prediction accurate)",
      "totalLeadTime": "8 hours advance notice"
    },
    "clinicalMessage": "SEPSIS CRITERIA MET - qSOFA ≥ 2. Multiple independent detection systems confirm. ML prediction accurate (predicted 8 hours ago). Immediate sepsis bundle initiation required."
  }
```

---

## 📊 Implementation Effort Estimate

### Module 5 ML Service (New Flink Job)

| Component | Estimated LOC | Estimated Time |
|-----------|--------------|----------------|
| ML Models (4 models) | ~5,000 | 3-4 weeks |
| Feature Engineering | ~1,500 | 1 week |
| Model Serving (ONNX) | ~1,000 | 3-5 days |
| Kafka Integration | ~500 | 2-3 days |
| Testing & Validation | ~1,000 | 1 week |
| **Module 5 Total** | **~9,000** | **5-6 weeks** |

---

### Module 4 Layer 3 Integration (Extend Existing)

| Component | Estimated LOC | Estimated Time |
|-----------|--------------|----------------|
| Kafka Consumer Setup | ~100 | 2 hours |
| MLToPatternConverter | ~350 | 4 hours |
| Orchestrator Integration | ~50 | 1 hour |
| Testing | ~200 | 2-3 hours |
| **Module 4 Total** | **~700** | **1-2 days** |

---

## ✅ Recommended Implementation Order

### Phase 1: Module 5 ML Service (Priority 1)

**Timeline**: 5-6 weeks
**Team**: ML Engineers + Data Scientists

1. **Week 1-2**: ML model development
   - Sepsis prediction model (XGBoost/Random Forest)
   - Mortality risk model
   - Respiratory failure model
   - AKI progression model

2. **Week 3**: Feature engineering pipeline
   - Extract 70 features from semantic events
   - Temporal aggregations (rolling windows)
   - Clinical score integration

3. **Week 4**: Model serving infrastructure
   - ONNX Runtime integration
   - Model loading and caching
   - Batch inference optimization

4. **Week 5**: Kafka integration & testing
   - Consumer/producer setup
   - End-to-end testing
   - Performance benchmarking

5. **Week 6**: Monitoring & validation
   - Model performance tracking
   - Prediction accuracy metrics
   - Production deployment

**Deliverable**: ML predictions flowing to `ehr-ml-predictions` Kafka topic

---

### Phase 2: Module 4 Layer 3 Integration (Priority 2)

**Timeline**: 1-2 days
**Team**: Flink/Stream Processing Engineers

**Day 1 Morning** (4 hours):
- Implement `mlPredictiveAnalysis()` method
- Implement `MLToPatternConverter`
- Add Kafka consumer configuration

**Day 1 Afternoon** (4 hours):
- Update orchestrator to include Layer 3
- Build and test locally
- Verify compilation

**Day 2 Morning** (4 hours):
- Integration testing with Module 5 predictions
- Verify deduplication works across all 3 layers
- Test multi-source confirmation

**Day 2 Afternoon** (4 hours):
- Production deployment
- Monitoring setup
- Documentation updates

**Deliverable**: Module 4 consuming ML predictions and merging with Layer 1 & 2

---

## 🎯 Success Criteria

### Module 5 (ML Service)

- ✅ All 4 ML models deployed and producing predictions
- ✅ Feature engineering pipeline operational
- ✅ Inference latency <100ms per prediction
- ✅ High-confidence predictions (≥0.70) outputted to Kafka
- ✅ Model monitoring dashboards operational

### Module 4 (Layer 3 Integration)

- ✅ ML predictions converted to PatternEvents
- ✅ Layer 3 patterns merged with Layer 1 & 2
- ✅ Deduplication working across all 3 layers
- ✅ Multi-source confirmation when layers agree
- ✅ Build successful with zero errors
- ✅ End-to-end testing complete

---

## 🚀 Current Status & Next Steps

### Current Status ✅

**Module 4**: 100% complete for Layer 1 & Layer 2
- ✅ Instant state assessment operational
- ✅ CEP patterns operational
- ✅ Deduplication working
- ✅ 225MB JAR built successfully
- ✅ Layer 3 placeholder ready for integration

**Module 5**: Not yet started (separate project)

---

### Immediate Next Steps

#### Option 1: Start Module 5 Now (Recommended if ML is priority)
1. Create Module 5 project structure
2. Begin ML model development
3. Implement feature engineering
4. Build prediction service
5. **Then** integrate Layer 3 into Module 4 (1-2 day task)

#### Option 2: Leave Layer 3 for Later (Recommended for now)
1. ✅ **Module 4 is production-ready** for Layer 1 & Layer 2
2. ⏳ Layer 3 is a **future enhancement**
3. ⏳ Implement when Module 5 ML models are ready
4. ⏳ Integration is only 1-2 days when needed

---

## 🎓 Key Architectural Decisions

`★ Insight ─────────────────────────────────────────────`

**Why Hybrid (Module 5 + Module 4) Instead of Module 4 Only?**

1. **Separation of Concerns**: ML model training/serving is fundamentally different from pattern detection. Keeping ML in Module 5 allows independent development, testing, and scaling.

2. **Reusability**: Module 5 ML predictions can serve multiple consumers (Module 4, dashboards, external systems) not just pattern detection.

3. **Technology Stack**: ML inference benefits from different infrastructure (GPU acceleration, model caching) than stream processing.

4. **Team Organization**: ML Engineers can work on Module 5 independently while Stream Processing Engineers maintain Module 4.

5. **Failure Isolation**: If ML models fail, Layer 1 & Layer 2 continue operating. If Module 4 integrated ML directly, ML failures could affect all pattern detection.

6. **Scalability**: Module 5 can scale horizontally based on ML inference load independently from Module 4's stream processing needs.

**Why Not Module 5 Only?**

- ML predictions need to participate in **deduplication** with Layer 1 & Layer 2
- ML predictions need **priority assignment** and **alert routing** from Module 4
- ML predictions benefit from **multi-source confirmation** when Layer 1/2 agree
- Module 4's existing infrastructure (Kafka sinks, monitoring, alerting) shouldn't be duplicated

**Result**: Hybrid approach gives you the best of both worlds!

`─────────────────────────────────────────────────────────`

---

**Status**: 📋 **DESIGN DOCUMENT COMPLETE - READY FOR IMPLEMENTATION**

**Recommendation**: **Keep Module 4 as-is** (production-ready for Layer 1 & 2). Implement Module 5 ML service when ML models are ready. Layer 3 integration into Module 4 is a simple 1-2 day task when Module 5 is operational.

---

**Next Steps**: Focus on Module 5 ML model development OR proceed with Module 4 production deployment for Layer 1 & Layer 2 (Layer 3 can be added later).
