
package com.cardiofit.flink.operators;

import com.cardiofit.flink.adapters.PatientContextAdapter;
import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.stream.models.CanonicalEvent;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.FlatMapFunction;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.functions.RichMapFunction;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.TypeHint;

import java.io.Serializable;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.streaming.api.functions.co.CoProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.time.Duration;
import java.util.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Module 5: ML Inference Engine
 *
 * Responsibilities:
 * - Load and manage ML models (ONNX format)
 * - Perform real-time inference on clinical data
 * - Generate risk scores and predictions
 * - Ensemble model predictions for improved accuracy
 * - Model versioning and A/B testing
 * - Feature engineering and preprocessing
 * - Prediction quality monitoring
 */
public class Module5_MLInference {
    private static final Logger LOG = LoggerFactory.getLogger(Module5_MLInference.class);

    // Output tags for different prediction types
    private static final OutputTag<MLPrediction> READMISSION_RISK_TAG =
        new OutputTag<MLPrediction>("readmission-risk"){};

    private static final OutputTag<MLPrediction> SEPSIS_PREDICTION_TAG =
        new OutputTag<MLPrediction>("sepsis-prediction"){};

    private static final OutputTag<MLPrediction> DETERIORATION_RISK_TAG =
        new OutputTag<MLPrediction>("deterioration-risk"){};

    private static final OutputTag<MLPrediction> FALL_RISK_TAG =
        new OutputTag<MLPrediction>("fall-risk"){};

    private static final OutputTag<MLPrediction> MORTALITY_RISK_TAG =
        new OutputTag<MLPrediction>("mortality-risk"){};

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 5: ML Inference Engine");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for ML processing
        env.setParallelism(2);
        env.enableCheckpointing(30000);
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        // Create ML inference pipeline
        createMLInferencePipeline(env);

        // Execute the job
        env.execute("Module 5: ML Inference Engine");
    }

    public static void createMLInferencePipeline(StreamExecutionEnvironment env) {
        LOG.info("Creating ML inference pipeline (KeyedCoProcessFunction architecture)");

        // Input stream 1: CDS events from Module 3
        DataStream<EnrichedPatientContext> cdsEvents = createEnrichedPatientContextSource(env);

        // Input stream 2: Pattern events from Module 4
        DataStream<PatternEvent> patternEvents = createPatternEventSource(env);

        // Dual-input KeyedCoProcessFunction — CDS triggers inference, patterns buffer
        SingleOutputStreamOperator<MLPrediction> predictions = cdsEvents
            .keyBy(EnrichedPatientContext::getPatientId)
            .connect(patternEvents.keyBy(PatternEvent::getPatientId))
            .process(new Module5_MLInferenceEngine())
            .uid("Module5-ML-Inference-Engine");

        // Main output: all predictions
        predictions
            .sinkTo(createMLPredictionsSink())
            .uid("ML Predictions Sink");

        // Side output: high-risk predictions
        predictions.getSideOutput(Module5_MLInferenceEngine.HIGH_RISK_TAG)
            .sinkTo(createHighRiskAlertsSink())
            .uid("High Risk Predictions Sink");

        // Side output: audit trail
        predictions.getSideOutput(Module5_MLInferenceEngine.AUDIT_TAG)
            .sinkTo(createAuditSink())
            .uid("Prediction Audit Sink");

        // ========================================
        // MIMIC-IV Real ML Inference Pipeline
        // ========================================
        LOG.info("Adding MIMIC-IV real ML inference pipeline");

        // Add EnrichedPatientContext source from Module 2
        DataStream<EnrichedPatientContext> enrichedContext = createEnrichedPatientContextSource(env);

        // Adapt to PatientContextSnapshot for MIMIC-IV models
        PatientContextAdapter adapter = new PatientContextAdapter();
        DataStream<PatientContextSnapshot> patientSnapshots = enrichedContext
            .map(context -> adapter.adapt(context))
            .name("Patient Context Adapter")
            .uid("mimic-context-adapter");

        // Run MIMIC-IV ML inference (returns List<MLPrediction>)
        DataStream<List<MLPrediction>> mimicPredictionLists = patientSnapshots
            .map(new MIMICMLInferenceOperator())
            .name("MIMIC-IV ML Inference")
            .uid("mimic-ml-inference");

        // Flatten prediction lists to individual predictions
        DataStream<MLPrediction> mimicPredictions = mimicPredictionLists
            .flatMap((FlatMapFunction<List<MLPrediction>, MLPrediction>)
                (list, out) -> list.forEach(out::collect))
            .returns(TypeInformation.of(MLPrediction.class))
            .name("MIMIC Prediction Flattener")
            .uid("mimic-prediction-flattener");

        // Sink MIMIC-IV predictions to output topics
        mimicPredictions
            .sinkTo(createMIMICMLPredictionsSink())
            .uid("MIMIC Predictions Sink");

        // Route high-risk MIMIC-IV predictions to alert topic
        mimicPredictions
            .filter(pred -> "HIGH".equals(pred.getRiskLevel()))
            .sinkTo(createHighRiskAlertsSink())
            .uid("MIMIC High-Risk Alerts Sink");

        LOG.info("MIMIC-IV ML inference pipeline added successfully");
        LOG.info("ML inference pipeline created successfully");
    }

    /**
     * Create semantic event source
     */
    private static DataStream<SemanticEvent> createSemanticEventSource(StreamExecutionEnvironment env) {
        KafkaSource<SemanticEvent> source = KafkaSource.<SemanticEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics("semantic-mesh-updates.v1")  // Module 4 semantic output topic
            .setGroupId("ml-inference-semantic")
            .setStartingOffsets(OffsetsInitializer.timestamp(System.currentTimeMillis()))
            .setValueOnlyDeserializer(new SemanticEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("ml-inference-semantic"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<SemanticEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "ML Semantic Events Source");
    }

    /**
     * Create pattern event source
     */
    private static DataStream<PatternEvent> createPatternEventSource(StreamExecutionEnvironment env) {
        // Read PatternEvent from Module 4 output topic (pattern-events.v1)
        // Note: This topic will be empty until Module 4 is deployed
        KafkaSource<PatternEvent> source = KafkaSource.<PatternEvent>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics("pattern-events.v1")  // Module 4 output - separate from clinical-patterns.v1
            .setGroupId("ml-inference-patterns")
            .setStartingOffsets(OffsetsInitializer.timestamp(System.currentTimeMillis()))
            .setValueOnlyDeserializer(new PatternEventDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("ml-inference-patterns"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<PatternEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getDetectionTime()),
            "ML Pattern Events Source");
    }

    // ===== Feature Engineering =====

    /**
     * Extract features from semantic events
     */
    public static class SemanticFeatureExtractor implements MapFunction<SemanticEvent, FeatureVector> {
        @Override
        public FeatureVector map(SemanticEvent event) throws Exception {
            FeatureVector features = new FeatureVector();
            features.setPatientId(event.getPatientId());
            features.setFeatureType("SEMANTIC");
            features.setTimestamp(event.getEventTime());

            Map<String, Double> featureMap = new HashMap<>();

            // Clinical significance features
            featureMap.put("clinical_significance", event.getClinicalSignificance());
            featureMap.put("overall_confidence", event.getOverallConfidence());

            // Risk level encoding
            String riskLevel = event.getRiskLevel();
            featureMap.put("risk_level_low", riskLevel.equals("low") ? 1.0 : 0.0);
            featureMap.put("risk_level_moderate", riskLevel.equals("moderate") ? 1.0 : 0.0);
            featureMap.put("risk_level_high", riskLevel.equals("high") ? 1.0 : 0.0);

            // Event type features
            featureMap.put("is_critical_event", event.getEventType().isCritical() ? 1.0 : 0.0);
            featureMap.put("is_clinical_event", event.getEventType().isClinical() ? 1.0 : 0.0);
            featureMap.put("is_medication_event", event.getEventType().isMedicationRelated() ? 1.0 : 0.0);

            // Clinical context features
            if (event.getPatientContext() != null) {
                featureMap.put("acuity_score", event.getPatientContext().getAcuityScore());
                featureMap.put("active_medication_count",
                    (double) event.getPatientContext().getActiveMedicationCount());
                featureMap.put("is_high_acuity", event.getPatientContext().isHighAcuity() ? 1.0 : 0.0);
                featureMap.put("is_currently_admitted",
                    event.getPatientContext().isCurrentlyAdmitted() ? 1.0 : 0.0);

                // Length of stay feature
                if (event.getPatientContext().getLengthOfStayHours() != null) {
                    featureMap.put("length_of_stay_hours", event.getPatientContext().getLengthOfStayHours());
                }
            }

            // Clinical alerts and interactions
            featureMap.put("has_clinical_alerts", event.hasClinicalAlerts() ? 1.0 : 0.0);
            featureMap.put("has_drug_interactions", event.hasDrugInteractions() ? 1.0 : 0.0);
            featureMap.put("has_guideline_recommendations", event.hasGuidelineRecommendations() ? 1.0 : 0.0);

            // Temporal features
            featureMap.put("is_acute", event.isAcute() ? 1.0 : 0.0);

            features.setFeatures(featureMap);
            features.setFeatureCount(featureMap.size());

            return features;
        }
    }

    /**
     * Extract features from pattern events
     */
    public static class PatternFeatureExtractor implements MapFunction<PatternEvent, FeatureVector> {
        @Override
        public FeatureVector map(PatternEvent event) throws Exception {
            // Null safety checks
            if (event == null) {
                throw new IllegalArgumentException("PatternEvent cannot be null");
            }
            if (event.getPatientId() == null || event.getPatientId().isEmpty()) {
                throw new IllegalArgumentException("PatientId cannot be null or empty in PatternEvent");
            }

            FeatureVector features = new FeatureVector();
            features.setPatientId(event.getPatientId());
            features.setFeatureType("PATTERN");
            features.setTimestamp(event.getDetectionTime());

            Map<String, Double> featureMap = new HashMap<>();

            // Pattern type features
            featureMap.put("is_deterioration_pattern", event.isDeteriorationPattern() ? 1.0 : 0.0);
            featureMap.put("is_medication_adherence_pattern", event.isMedicationAdherencePattern() ? 1.0 : 0.0);
            featureMap.put("is_anomaly_pattern", event.isAnomalyPattern() ? 1.0 : 0.0);
            featureMap.put("is_trend_pattern", event.isTrendPattern() ? 1.0 : 0.0);
            featureMap.put("is_pathway_compliance_pattern", event.isPathwayCompliancePattern() ? 1.0 : 0.0);

            // Severity and confidence
            featureMap.put("pattern_confidence", event.getConfidence());
            featureMap.put("is_high_severity", event.isHighSeverity() ? 1.0 : 0.0);
            featureMap.put("requires_immediate_action", event.requiresImmediateAction() ? 1.0 : 0.0);

            // Pattern characteristics
            featureMap.put("involved_event_count", (double) event.getInvolvedEventCount());

            // Pattern duration
            if (event.getPatternDurationHours() != null) {
                featureMap.put("pattern_duration_hours", event.getPatternDurationHours());
            }

            // Acknowledgment status
            featureMap.put("is_acknowledged", event.isAcknowledged() ? 1.0 : 0.0);

            features.setFeatures(featureMap);
            features.setFeatureCount(featureMap.size());

            return features;
        }
    }

    /**
     * Combine features from different sources
     */
    public static class FeatureCombiner extends KeyedProcessFunction<String, FeatureVector, FeatureVector> {
        private transient ValueState<FeatureVector> semanticFeaturesState;
        private transient ValueState<FeatureVector> patternFeaturesState;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) throws Exception {
            semanticFeaturesState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("semantic-features", FeatureVector.class));
            patternFeaturesState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("pattern-features", FeatureVector.class));
        }

        @Override
        public void processElement(FeatureVector value, Context ctx, Collector<FeatureVector> out) throws Exception {
            // Null safety checks
            if (value == null) {
                return; // Skip null values
            }
            if (value.getFeatureType() == null) {
                return; // Skip if feature type is null
            }

            // Ensure state is initialized
            if (semanticFeaturesState == null || patternFeaturesState == null) {
                throw new IllegalStateException("State not initialized. open() method may not have been called.");
            }

            if ("SEMANTIC".equals(value.getFeatureType())) {
                semanticFeaturesState.update(value);
            } else if ("PATTERN".equals(value.getFeatureType())) {
                patternFeaturesState.update(value);
            }

            // Combine features when both are available
            FeatureVector semantic = semanticFeaturesState.value();
            FeatureVector pattern = patternFeaturesState.value();

            if (semantic != null && pattern != null) {
                FeatureVector combined = combineFeatures(semantic, pattern);
                out.collect(combined);

                // Clear state after combination
                semanticFeaturesState.clear();
                patternFeaturesState.clear();
            } else if (semantic != null && isFeatureVectorRecent(semantic)) {
                // Emit semantic features alone if recent enough
                out.collect(semantic);
                semanticFeaturesState.clear();
            }
        }

        private FeatureVector combineFeatures(FeatureVector semantic, FeatureVector pattern) {
            FeatureVector combined = new FeatureVector();
            combined.setPatientId(semantic.getPatientId());
            combined.setFeatureType("COMBINED");
            combined.setTimestamp(Math.max(semantic.getTimestamp(), pattern.getTimestamp()));

            Map<String, Double> combinedFeatures = new HashMap<>();

            // Add semantic features with prefix
            for (Map.Entry<String, Double> entry : semantic.getFeatures().entrySet()) {
                combinedFeatures.put("semantic_" + entry.getKey(), entry.getValue());
            }

            // Add pattern features with prefix
            for (Map.Entry<String, Double> entry : pattern.getFeatures().entrySet()) {
                combinedFeatures.put("pattern_" + entry.getKey(), entry.getValue());
            }

            // Add interaction features
            combinedFeatures.put("semantic_pattern_confidence_product",
                semantic.getFeatures().getOrDefault("overall_confidence", 0.0) *
                pattern.getFeatures().getOrDefault("pattern_confidence", 0.0));

            combined.setFeatures(combinedFeatures);
            combined.setFeatureCount(combinedFeatures.size());

            return combined;
        }

        private boolean isFeatureVectorRecent(FeatureVector vector) {
            long currentTime = System.currentTimeMillis();
            long featureAge = currentTime - vector.getTimestamp();
            return featureAge < Duration.ofMinutes(30).toMillis(); // 30 minutes threshold
        }
    }

    // ===== ML Inference Engine =====

    /**
     * Main ML inference processor
     */
    public static class MLInferenceProcessor extends KeyedProcessFunction<String, FeatureVector, MLPrediction> {
        private transient Map<String, MLModel> models;
        private transient ExecutorService executorService;

        // @Override - Removed for Flink 2.x
        public void open(org.apache.flink.configuration.Configuration parameters) throws Exception {
            // Initialize ML models
            models = new HashMap<>();
            loadMLModels();

            // Create thread pool for async inference
            executorService = Executors.newFixedThreadPool(4);

            LOG.info("ML Inference Processor initialized with {} models", models.size());
        }

        @Override
        public void close() throws Exception {
            if (executorService != null) {
                executorService.shutdown();
            }
        }

        @Override
        public void processElement(FeatureVector features, Context ctx, Collector<MLPrediction> out) throws Exception {
            try {
                String patientId = features.getPatientId();

                // Run inference for each model
                for (Map.Entry<String, MLModel> modelEntry : models.entrySet()) {
                    String modelName = modelEntry.getKey();
                    MLModel model = modelEntry.getValue();

                    if (model.isApplicable(features)) {
                        MLPrediction prediction = runInference(model, features, ctx);
                        if (prediction != null) {
                            out.collect(prediction);

                            // Route to specific sinks based on prediction type
                            routePredictionToSideOutput(prediction, ctx);
                        }
                    }
                }

            } catch (Exception e) {
                LOG.error("Failed to run ML inference for patient: " + features.getPatientId(), e);
            }
        }

        private void loadMLModels() {
            try {
                // Readmission Risk Model
                MLModel readmissionModel = new MLModel();
                readmissionModel.setModelName("readmission_risk_v1");
                readmissionModel.setModelType("READMISSION_RISK");
                readmissionModel.setModelPath("/models/readmission_risk_v1.onnx");
                readmissionModel.setThreshold(0.7);
                readmissionModel.setRequiredFeatures(Arrays.asList(
                    "semantic_acuity_score", "semantic_length_of_stay_hours",
                    "semantic_active_medication_count", "pattern_is_deterioration_pattern"
                ));
                models.put("readmission_risk", readmissionModel);

                // Sepsis Prediction Model
                MLModel sepsisModel = new MLModel();
                sepsisModel.setModelName("sepsis_prediction_v2");
                sepsisModel.setModelType("SEPSIS_PREDICTION");
                sepsisModel.setModelPath("/models/sepsis_prediction_v2.onnx");
                sepsisModel.setThreshold(0.8);
                sepsisModel.setRequiredFeatures(Arrays.asList(
                    "semantic_clinical_significance", "semantic_is_critical_event",
                    "pattern_is_deterioration_pattern", "pattern_pattern_confidence"
                ));
                models.put("sepsis_prediction", sepsisModel);

                // Clinical Deterioration Model
                MLModel deteriorationModel = new MLModel();
                deteriorationModel.setModelName("deterioration_risk_v1");
                deteriorationModel.setModelType("DETERIORATION_RISK");
                deteriorationModel.setModelPath("/models/deterioration_risk_v1.onnx");
                deteriorationModel.setThreshold(0.75);
                deteriorationModel.setRequiredFeatures(Arrays.asList(
                    "semantic_acuity_score", "semantic_is_high_acuity",
                    "pattern_is_deterioration_pattern", "pattern_requires_immediate_action"
                ));
                models.put("deterioration_risk", deteriorationModel);

                // Fall Risk Model
                MLModel fallRiskModel = new MLModel();
                fallRiskModel.setModelName("fall_risk_v1");
                fallRiskModel.setModelType("FALL_RISK");
                fallRiskModel.setModelPath("/models/fall_risk_v1.onnx");
                fallRiskModel.setThreshold(0.6);
                fallRiskModel.setRequiredFeatures(Arrays.asList(
                    "semantic_active_medication_count", "semantic_length_of_stay_hours",
                    "pattern_is_medication_adherence_pattern"
                ));
                models.put("fall_risk", fallRiskModel);

                // Mortality Risk Model
                MLModel mortalityModel = new MLModel();
                mortalityModel.setModelName("mortality_risk_v1");
                mortalityModel.setModelType("MORTALITY_RISK");
                mortalityModel.setModelPath("/models/mortality_risk_v1.onnx");
                mortalityModel.setThreshold(0.9);
                mortalityModel.setRequiredFeatures(Arrays.asList(
                    "semantic_acuity_score", "semantic_is_critical_event",
                    "pattern_is_deterioration_pattern", "pattern_is_high_severity"
                ));
                models.put("mortality_risk", mortalityModel);

                LOG.info("Loaded {} ML models successfully", models.size());

            } catch (Exception e) {
                LOG.error("Failed to load ML models", e);
            }
        }

        private MLPrediction runInference(MLModel model, FeatureVector features, Context ctx) {
            try {
                // Prepare input features
                double[] inputFeatures = prepareInputFeatures(model, features);

                // Run model inference (simplified - in practice would use ONNX Runtime)
                double[] predictions = simulateModelInference(model, inputFeatures);

                // Create prediction result
                MLPrediction prediction = new MLPrediction();
                prediction.setId(UUID.randomUUID().toString());
                prediction.setPatientId(features.getPatientId());
                prediction.setModelName(model.getModelName());
                prediction.setModelType(model.getModelType());
                prediction.setPredictionTime(System.currentTimeMillis());
                prediction.setInputFeatureCount(inputFeatures.length);

                // Set prediction scores
                Map<String, Double> scores = new HashMap<>();
                scores.put("primary_score", predictions[0]);
                if (predictions.length > 1) {
                    scores.put("confidence_score", predictions[1]);
                }
                prediction.setPredictionScores(scores);

                // Determine risk level
                String riskLevel = determineRiskLevel(predictions[0], model.getThreshold());
                prediction.setRiskLevel(riskLevel);

                // Set model metadata
                Map<String, Object> metadata = new HashMap<>();
                metadata.put("model_version", model.getModelName());
                metadata.put("feature_count", inputFeatures.length);
                metadata.put("inference_time", System.currentTimeMillis());
                metadata.put("threshold", model.getThreshold());
                prediction.setModelMetadata(metadata);

                // Generate recommendations based on prediction
                List<String> recommendations = generateRecommendations(prediction);
                prediction.setRecommendedActions(recommendations);

                LOG.debug("Generated {} prediction for patient {}: score={}, risk={}",
                    model.getModelType(), features.getPatientId(), predictions[0], riskLevel);

                return prediction;

            } catch (Exception e) {
                LOG.error("Failed to run inference for model: " + model.getModelName(), e);
                return null;
            }
        }

        private double[] prepareInputFeatures(MLModel model, FeatureVector features) {
            List<String> requiredFeatures = model.getRequiredFeatures();
            double[] inputArray = new double[requiredFeatures.size()];

            for (int i = 0; i < requiredFeatures.size(); i++) {
                String featureName = requiredFeatures.get(i);
                Double featureValue = features.getFeatures().get(featureName);
                inputArray[i] = featureValue != null ? featureValue : 0.0;
            }

            return inputArray;
        }

        private double[] simulateModelInference(MLModel model, double[] input) {
            // Simplified simulation of ML model inference
            // In practice, this would use ONNX Runtime to run actual models

            double score = 0.0;
            double confidence = 0.8;

            // Simple weighted sum simulation
            for (double feature : input) {
                score += feature * (0.1 + Math.random() * 0.1); // Random weights for simulation
            }

            // Normalize score to 0-1 range
            score = Math.min(1.0, Math.max(0.0, score / input.length));

            // Add some noise for realism
            score += (Math.random() - 0.5) * 0.1;
            score = Math.min(1.0, Math.max(0.0, score));

            return new double[]{score, confidence};
        }

        private String determineRiskLevel(double score, double threshold) {
            if (score >= threshold) {
                return "HIGH";
            } else if (score >= threshold * 0.7) {
                return "MODERATE";
            } else {
                return "LOW";
            }
        }

        private List<String> generateRecommendations(MLPrediction prediction) {
            List<String> recommendations = new ArrayList<>();
            String modelType = prediction.getModelType();
            String riskLevel = prediction.getRiskLevel();

            if ("HIGH".equals(riskLevel)) {
                switch (modelType) {
                    case "READMISSION_RISK":
                        recommendations.add("DISCHARGE_PLANNING_REVIEW");
                        recommendations.add("HOME_CARE_COORDINATION");
                        recommendations.add("MEDICATION_RECONCILIATION");
                        break;
                    case "SEPSIS_PREDICTION":
                        recommendations.add("IMMEDIATE_SEPSIS_WORKUP");
                        recommendations.add("BLOOD_CULTURES_STAT");
                        recommendations.add("CONSIDER_ANTIBIOTICS");
                        break;
                    case "DETERIORATION_RISK":
                        recommendations.add("INCREASE_MONITORING_FREQUENCY");
                        recommendations.add("NOTIFY_PHYSICIAN");
                        recommendations.add("CONSIDER_ICU_TRANSFER");
                        break;
                    case "FALL_RISK":
                        recommendations.add("IMPLEMENT_FALL_PRECAUTIONS");
                        recommendations.add("MEDICATION_REVIEW");
                        recommendations.add("ENVIRONMENTAL_ASSESSMENT");
                        break;
                    case "MORTALITY_RISK":
                        recommendations.add("PALLIATIVE_CARE_CONSULT");
                        recommendations.add("FAMILY_NOTIFICATION");
                        recommendations.add("GOALS_OF_CARE_DISCUSSION");
                        break;
                }
            } else if ("MODERATE".equals(riskLevel)) {
                recommendations.add("ENHANCED_MONITORING");
                recommendations.add("CLINICAL_REVIEW");
            }

            return recommendations;
        }

        private void routePredictionToSideOutput(MLPrediction prediction, Context ctx) {
            switch (prediction.getModelType()) {
                case "READMISSION_RISK":
                    ctx.output(READMISSION_RISK_TAG, prediction);
                    break;
                case "SEPSIS_PREDICTION":
                    ctx.output(SEPSIS_PREDICTION_TAG, prediction);
                    break;
                case "DETERIORATION_RISK":
                    ctx.output(DETERIORATION_RISK_TAG, prediction);
                    break;
                case "FALL_RISK":
                    ctx.output(FALL_RISK_TAG, prediction);
                    break;
                case "MORTALITY_RISK":
                    ctx.output(MORTALITY_RISK_TAG, prediction);
                    break;
            }
        }
    }

    // ===== Ensemble Processing =====

    /**
     * Ensemble processor for combining predictions
     */
    public static class EnsembleProcessor extends KeyedProcessFunction<String, MLPrediction, MLPrediction> {
        private transient ValueState<List<MLPrediction>> recentPredictionsState;

        // @Override - Removed for Flink 2.x
        public void open(org.apache.flink.configuration.Configuration parameters) {
            recentPredictionsState = getRuntimeContext().getState(
                new ValueStateDescriptor<>("recent-predictions",
                    TypeInformation.of(new TypeHint<List<MLPrediction>>(){})));
        }

        @Override
        public void processElement(MLPrediction prediction, Context ctx, Collector<MLPrediction> out) throws Exception {
            // Get recent predictions
            List<MLPrediction> recentPredictions = recentPredictionsState.value();
            if (recentPredictions == null) {
                recentPredictions = new ArrayList<>();
            }

            // Add new prediction
            recentPredictions.add(prediction);

            // Keep only recent predictions (last 10 minutes)
            long cutoffTime = System.currentTimeMillis() - Duration.ofMinutes(10).toMillis();
            recentPredictions.removeIf(p -> p.getPredictionTime() < cutoffTime);

            // Update state
            recentPredictionsState.update(recentPredictions);

            // Create ensemble prediction if we have multiple predictions for same model type
            List<MLPrediction> sameTypePredictions = recentPredictions.stream()
                .filter(p -> p.getModelType().equals(prediction.getModelType()))
                .collect(java.util.stream.Collectors.toList());

            if (sameTypePredictions.size() >= 2) {
                MLPrediction ensembled = createEnsemblePrediction(sameTypePredictions);
                out.collect(ensembled);
            } else {
                // Emit individual prediction
                out.collect(prediction);
            }
        }

        private MLPrediction createEnsemblePrediction(List<MLPrediction> predictions) {
            MLPrediction ensembled = new MLPrediction();
            ensembled.setId(UUID.randomUUID().toString());
            ensembled.setPatientId(predictions.get(0).getPatientId());
            ensembled.setModelName("ensemble_" + predictions.get(0).getModelType().toLowerCase());
            ensembled.setModelType(predictions.get(0).getModelType());
            ensembled.setPredictionTime(System.currentTimeMillis());

            // Calculate ensemble score (simple average)
            double averageScore = predictions.stream()
                .mapToDouble(p -> p.getPredictionScores().getOrDefault("primary_score", 0.0))
                .average()
                .orElse(0.0);

            Map<String, Double> ensembleScores = new HashMap<>();
            ensembleScores.put("ensemble_score", averageScore);
            ensembleScores.put("prediction_count", (double) predictions.size());
            ensembled.setPredictionScores(ensembleScores);

            // Determine ensemble risk level
            String riskLevel = determineEnsembleRiskLevel(predictions, averageScore);
            ensembled.setRiskLevel(riskLevel);

            // Combine recommendations
            Set<String> combinedRecommendations = new HashSet<>();
            for (MLPrediction pred : predictions) {
                if (pred.getRecommendedActions() != null) {
                    combinedRecommendations.addAll(pred.getRecommendedActions());
                }
            }
            ensembled.setRecommendedActions(new ArrayList<>(combinedRecommendations));

            return ensembled;
        }

        private String determineEnsembleRiskLevel(List<MLPrediction> predictions, double averageScore) {
            // Count risk levels
            long highCount = predictions.stream().filter(p -> "HIGH".equals(p.getRiskLevel())).count();
            long moderateCount = predictions.stream().filter(p -> "MODERATE".equals(p.getRiskLevel())).count();

            // Conservative approach: if any prediction is high risk, ensemble is high risk
            if (highCount > 0 || averageScore > 0.8) {
                return "HIGH";
            } else if (moderateCount > 0 || averageScore > 0.5) {
                return "MODERATE";
            } else {
                return "LOW";
            }
        }
    }

    // ===== Helper Classes =====

    public static class FeatureVector implements Serializable {
        private String patientId;
        private String featureType;
        private long timestamp;
        private Map<String, Double> features;
        private int featureCount;

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public String getFeatureType() { return featureType; }
        public void setFeatureType(String featureType) { this.featureType = featureType; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        public Map<String, Double> getFeatures() { return features; }
        public void setFeatures(Map<String, Double> features) { this.features = features; }

        public int getFeatureCount() { return featureCount; }
        public void setFeatureCount(int featureCount) { this.featureCount = featureCount; }
    }

    public static class MLModel implements Serializable {
        private String modelName;
        private String modelType;
        private String modelPath;
        private double threshold;
        private List<String> requiredFeatures;
        private String version;

        public boolean isApplicable(FeatureVector features) {
            // Check if all required features are present
            if (requiredFeatures == null) return true;

            for (String requiredFeature : requiredFeatures) {
                if (!features.getFeatures().containsKey(requiredFeature)) {
                    return false;
                }
            }
            return true;
        }

        // Getters and setters
        public String getModelName() { return modelName; }
        public void setModelName(String modelName) { this.modelName = modelName; }

        public String getModelType() { return modelType; }
        public void setModelType(String modelType) { this.modelType = modelType; }

        public String getModelPath() { return modelPath; }
        public void setModelPath(String modelPath) { this.modelPath = modelPath; }

        public double getThreshold() { return threshold; }
        public void setThreshold(double threshold) { this.threshold = threshold; }

        public List<String> getRequiredFeatures() { return requiredFeatures; }
        public void setRequiredFeatures(List<String> requiredFeatures) { this.requiredFeatures = requiredFeatures; }

        public String getVersion() { return version; }
        public void setVersion(String version) { this.version = version; }
    }

    // ===== Sink Creation Methods =====

    private static KafkaSink<MLPrediction> createMLPredictionsSink() {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("key.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");
        producerConfig.setProperty("value.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");

        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.INFERENCE_RESULTS.getTopicName())
                .setKeySerializationSchema((MLPrediction prediction) ->
                    prediction.getPatientId().getBytes(java.nio.charset.StandardCharsets.UTF_8))
                .setValueSerializationSchema(new MLPredictionSerializer())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setTransactionalIdPrefix("module5-ml-predictions-ensemble")
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<MLPrediction> createMIMICMLPredictionsSink() {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("key.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");
        producerConfig.setProperty("value.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");

        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(KafkaTopics.INFERENCE_RESULTS.getTopicName())
                .setKeySerializationSchema((MLPrediction prediction) ->
                    prediction.getPatientId().getBytes(java.nio.charset.StandardCharsets.UTF_8))
                .setValueSerializationSchema(new MLPredictionSerializer())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setTransactionalIdPrefix("module5-ml-predictions-mimic")
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<MLPrediction> createReadmissionRiskSink() {
        return createSpecializedMLSink(KafkaTopics.CLINICAL_REASONING_EVENTS, "module5-readmission-risk");
    }

    private static KafkaSink<MLPrediction> createSepsisPredictionSink() {
        return createSpecializedMLSink(KafkaTopics.ALERT_MANAGEMENT, "module5-sepsis-prediction");
    }

    private static KafkaSink<MLPrediction> createDeteriorationRiskSink() {
        return createSpecializedMLSink(KafkaTopics.ALERT_MANAGEMENT, "module5-deterioration-risk");
    }

    private static KafkaSink<MLPrediction> createFallRiskSink() {
        return createSpecializedMLSink(KafkaTopics.SAFETY_EVENTS, "module5-fall-risk");
    }

    private static KafkaSink<MLPrediction> createMortalityRiskSink() {
        return createSpecializedMLSink(KafkaTopics.CLINICAL_REASONING_EVENTS, "module5-mortality-risk");
    }

    private static KafkaSink<MLPrediction> createSpecializedMLSink(KafkaTopics topic, String transactionalIdPrefix) {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("key.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");
        producerConfig.setProperty("value.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");

        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic(topic.getTopicName())
                .setKeySerializationSchema((MLPrediction prediction) ->
                    prediction.getPatientId().getBytes(java.nio.charset.StandardCharsets.UTF_8))
                .setValueSerializationSchema(new MLPredictionSerializer())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setTransactionalIdPrefix(transactionalIdPrefix)
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    /**
     * Create EnrichedPatientContext source from Module 2 output
     * Reads from clinical-patterns.v1 topic for MIMIC-IV ML inference
     */
    private static DataStream<EnrichedPatientContext> createEnrichedPatientContextSource(StreamExecutionEnvironment env) {
        KafkaSource<EnrichedPatientContext> source = KafkaSource.<EnrichedPatientContext>builder()
            .setBootstrapServers(getBootstrapServers())
            .setTopics(KafkaTopics.CLINICAL_PATTERNS.getTopicName())
            .setGroupId("module5-mimic-inference")
            .setStartingOffsets(OffsetsInitializer.timestamp(System.currentTimeMillis()))
            .setValueOnlyDeserializer(new EnrichedPatientContextDeserializer())
            .setProperties(KafkaConfigLoader.getAutoConsumerConfig("module5-mimic-inference"))
            .build();

        return env.fromSource(source,
            WatermarkStrategy
                .<EnrichedPatientContext>forBoundedOutOfOrderness(Duration.ofSeconds(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime()),
            "MIMIC Enriched Patient Context Source");
    }

    /**
     * Create high-risk alerts sink for MIMIC-IV predictions
     * Routes high-risk ML predictions to dedicated ml-risk-alerts.v1 topic
     * FIXED: Changed from alert-management.v1 (which is for PatternEvent alerts from Module 4)
     */
    private static KafkaSink<MLPrediction> createHighRiskAlertsSink() {
        Properties producerConfig = new Properties();
        producerConfig.setProperty("bootstrap.servers", getBootstrapServers());
        producerConfig.setProperty("key.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");
        producerConfig.setProperty("value.serializer", "org.apache.kafka.common.serialization.ByteArraySerializer");

        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(getBootstrapServers())
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic("ml-risk-alerts.v1")  // FIXED: Dedicated topic for ML risk predictions
                .setKeySerializationSchema((MLPrediction prediction) ->
                    prediction.getPatientId().getBytes(java.nio.charset.StandardCharsets.UTF_8))
                .setValueSerializationSchema(new MLPredictionSerializer())
                .build())
            .setKafkaProducerConfig(producerConfig)
            .setTransactionalIdPrefix("module5-high-risk-alerts")
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static KafkaSink<MLPrediction> createAuditSink() {
        return KafkaSink.<MLPrediction>builder()
            .setBootstrapServers(KafkaConfigLoader.getBootstrapServers())
            .setRecordSerializer(
                KafkaRecordSerializationSchema.builder()
                    .setTopic("prediction-audit.v1")
                    .setValueSerializationSchema(new MLPredictionSerializer())
                    .build()
            )
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .build();
    }

    private static String getBootstrapServers() {
        // Use single Kafka instance (aligned with KafkaConfigLoader)
        return KafkaConfigLoader.isRunningInDocker()
            ? "kafka:29092"
            : "localhost:9092";
    }

    // ===== Serialization Classes =====

    private static class SemanticEventDeserializer implements DeserializationSchema<SemanticEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
            // Use snake_case for JSON property names to match SemanticEvent from Module 4
            objectMapper.setPropertyNamingStrategy(com.fasterxml.jackson.databind.PropertyNamingStrategies.SNAKE_CASE);
        }

        @Override
        public SemanticEvent deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, SemanticEvent.class);
        }

        @Override
        public boolean isEndOfStream(SemanticEvent nextElement) { return false; }

        @Override
        public TypeInformation<SemanticEvent> getProducedType() {
            return TypeInformation.of(SemanticEvent.class);
        }
    }

    private static class PatternEventDeserializer implements DeserializationSchema<PatternEvent> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public PatternEvent deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, PatternEvent.class);
        }

        @Override
        public boolean isEndOfStream(PatternEvent nextElement) { return false; }

        @Override
        public TypeInformation<PatternEvent> getProducedType() {
            return TypeInformation.of(PatternEvent.class);
        }
    }

    /**
     * Deserializer for EnrichedPatientContext from Module 2 output
     * Used for MIMIC-IV ML inference pipeline
     */
    private static class EnrichedPatientContextDeserializer implements DeserializationSchema<EnrichedPatientContext> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(DeserializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public EnrichedPatientContext deserialize(byte[] message) throws IOException {
            return objectMapper.readValue(message, EnrichedPatientContext.class);
        }

        @Override
        public boolean isEndOfStream(EnrichedPatientContext nextElement) { return false; }

        @Override
        public TypeInformation<EnrichedPatientContext> getProducedType() {
            return TypeInformation.of(EnrichedPatientContext.class);
        }
    }

    private static class MLPredictionSerializer implements SerializationSchema<MLPrediction> {
        private transient ObjectMapper objectMapper;

        @Override
        public void open(SerializationSchema.InitializationContext context) {
            objectMapper = new ObjectMapper();
            objectMapper.registerModule(new JavaTimeModule());
        }

        @Override
        public byte[] serialize(MLPrediction element) {
            try {
                return objectMapper.writeValueAsBytes(element);
            } catch (Exception e) {
                throw new RuntimeException("Failed to serialize MLPrediction", e);
            }
        }
    }
}
