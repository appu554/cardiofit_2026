package com.cardiofit.flink.operators;

import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.models.*;
import org.apache.flink.api.common.state.StateTtlConfig;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.streaming.api.functions.co.KeyedCoProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.time.Duration;
import java.util.*;

/**
 * Module 5: ML Inference Engine — KeyedCoProcessFunction implementation.
 *
 * Dual-input streams:
 *   Input 1 (CDS events): Triggers inference after cooldown check
 *   Input 2 (Pattern events): Buffers into patient state, triggers on CRITICAL escalation
 *
 * Replaces the union-based chain to prevent 4-5x inference amplification.
 */
public class Module5_MLInferenceEngine
        extends KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction> {

    private static final Logger LOG = LoggerFactory.getLogger(Module5_MLInferenceEngine.class);

    // Output tags for side outputs
    public static final OutputTag<MLPrediction> HIGH_RISK_TAG =
        new OutputTag<>("high-risk-predictions", TypeInformation.of(MLPrediction.class)){};
    public static final OutputTag<MLPrediction> AUDIT_TAG =
        new OutputTag<>("prediction-audit", TypeInformation.of(MLPrediction.class)){};

    // State
    private transient ValueState<PatientMLState> patientState;
    private transient ValueState<Long> lastInferenceTime;

    // ONNX models (transient — initialized in open())
    private transient Map<String, ONNXModelContainer> modelSessions;

    private static final String[] CATEGORIES = {
        "readmission", "sepsis", "deterioration", "fall", "mortality"
    };

    @Override
    public void open(OpenContext openContext) throws Exception {
        super.open(openContext);

        // State with 7-day TTL
        StateTtlConfig ttlConfig = StateTtlConfig.newBuilder(Duration.ofDays(7))
            .setUpdateType(StateTtlConfig.UpdateType.OnReadAndWrite)
            .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
            .build();

        ValueStateDescriptor<PatientMLState> stateDesc =
            new ValueStateDescriptor<>("patient-ml-state", PatientMLState.class);
        stateDesc.enableTimeToLive(ttlConfig);
        patientState = getRuntimeContext().getState(stateDesc);

        ValueStateDescriptor<Long> timeDesc =
            new ValueStateDescriptor<>("last-inference-time", Long.class);
        timeDesc.enableTimeToLive(ttlConfig);
        lastInferenceTime = getRuntimeContext().getState(timeDesc);

        // Load ONNX models (graceful degradation — Section 10)
        modelSessions = new HashMap<>();
        String modelBasePath = System.getenv("ML_MODEL_PATH");
        if (modelBasePath == null) modelBasePath = "/opt/flink/models";

        for (String category : CATEGORIES) {
            String modelPath = modelBasePath + "/" + category + "/model.onnx";
            if (new File(modelPath).exists()) {
                try {
                    List<String> featureNames = new ArrayList<>();
                    for (int i = 0; i < Module5FeatureExtractor.FEATURE_COUNT; i++) {
                        featureNames.add("f" + i);
                    }
                    ONNXModelContainer model = ONNXModelContainer.builder()
                        .modelId(category + "_v1")
                        .modelName(category + " predictor")
                        .modelType(mapCategoryToModelType(category))
                        .modelVersion("1.0.0")
                        .inputFeatureNames(featureNames)
                        .config(ModelConfig.builder()
                            .predictionThreshold(0.5)
                            .intraOpThreads(1)
                            .interOpThreads(1)
                            .modelPath(modelPath)
                            .build())
                        .build();
                    model.initialize();
                    modelSessions.put(category, model);
                    LOG.info("Loaded ONNX model for {} from {}", category, modelPath);
                } catch (Exception e) {
                    LOG.warn("Failed to load model for {}: {}", category, e.getMessage());
                }
            } else {
                LOG.info("Model not found for {}: {} — predictions disabled", category, modelPath);
            }
        }

        // Register metrics
        getRuntimeContext().getMetricGroup().gauge("models_loaded",
            () -> (long) modelSessions.size());
    }

    // ═══════════════════════════════════════════
    // PRIMARY PATH: CDS events trigger inference
    // ═══════════════════════════════════════════

    @Override
    public void processElement1(
            EnrichedPatientContext cdsEvent,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) throws Exception {

        if (cdsEvent == null || cdsEvent.getPatientId() == null) {
            LOG.warn("Null CDS event or patientId — skipping (Lesson 1: silent deser failure)");
            return;
        }

        // Update patient state
        PatientMLState state = getOrCreateState(cdsEvent.getPatientId());
        updateStateFromCDS(state, cdsEvent);
        patientState.update(state);

        // No models → pass through
        if (modelSessions.isEmpty()) {
            LOG.debug("No ML models loaded — skipping inference for {}", cdsEvent.getPatientId());
            return;
        }

        // Check cooldown (Gap 3: lab-aware)
        Long lastRun = lastInferenceTime.value();
        long currentTime = cdsEvent.getProcessingTime();
        if (!Module5ClinicalScoring.shouldRunInference(
                state.getNews2Score(), state.getQsofaScore(),
                state.getRiskIndicators(),
                lastRun != null ? lastRun : 0, currentTime)) {
            return;
        }

        // Run inference
        runInferenceAndEmit(state, "CDS_EVENT", ctx, out);
        lastInferenceTime.update(currentTime);

        // Clear pattern buffer after inference
        state.clearPatternBuffer();
        patientState.update(state);
    }

    // ═══════════════════════════════════════════
    // SECONDARY PATH: Pattern events update state
    // ═══════════════════════════════════════════

    @Override
    public void processElement2(
            PatternEvent patternEvent,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) throws Exception {

        if (patternEvent == null || patternEvent.getPatientId() == null) return;

        PatientMLState state = getOrCreateState(patternEvent.getPatientId());

        // Buffer pattern
        state.addPattern(new PatientMLState.PatternSummary(
            patternEvent.getPatternType(),
            patternEvent.getSeverity(),
            patternEvent.getConfidence(),
            patternEvent.getDetectionTime(),
            patternEvent.getTags()
        ));
        patientState.update(state);

        // CRITICAL escalation → bypass cooldown and trigger immediate inference
        if ("CRITICAL".equals(patternEvent.getSeverity())
                && patternEvent.getTags() != null
                && patternEvent.getTags().contains("SEVERITY_ESCALATION")
                && !modelSessions.isEmpty()) {

            LOG.info("CRITICAL escalation for {} — triggering immediate inference",
                patternEvent.getPatientId());
            runInferenceAndEmit(state, "SEVERITY_ESCALATION", ctx, out);
            lastInferenceTime.update(System.currentTimeMillis());
            state.clearPatternBuffer();
            patientState.update(state);
        }
    }

    // ═══════════════════════════════════════════
    // Inference execution
    // ═══════════════════════════════════════════

    private void runInferenceAndEmit(
            PatientMLState state, String triggerSource,
            KeyedCoProcessFunction<String, EnrichedPatientContext, PatternEvent, MLPrediction>.Context ctx,
            Collector<MLPrediction> out) {

        float[] features = Module5FeatureExtractor.extractFeatures(state);

        for (Map.Entry<String, ONNXModelContainer> entry : modelSessions.entrySet()) {
            String category = entry.getKey();
            ONNXModelContainer model = entry.getValue();

            try {
                long start = System.nanoTime();
                MLPrediction rawPrediction = model.predict(features);
                long elapsedMs = (System.nanoTime() - start) / 1_000_000;

                // Enrich with Module 5 metadata
                rawPrediction.setPatientId(state.getPatientId());
                rawPrediction.setPredictionCategory(category);
                rawPrediction.setTriggerSource(triggerSource);
                rawPrediction.setContextDepth(
                    state.getTotalEventCount() > 3 ? "ESTABLISHED" : "INITIAL");

                // Calibrate (Gap 3 related)
                double rawScore = rawPrediction.getPrimaryScore();
                double calibrated = Module5ClinicalScoring.calibrate(rawScore, category);
                rawPrediction.setCalibratedScore(calibrated);
                rawPrediction.setRiskLevel(
                    Module5ClinicalScoring.classifyRiskLevel(calibrated, category));

                // Store input features for audit
                rawPrediction.setInputFeatures(features);

                // Main output
                out.collect(rawPrediction);

                // Side output: high risk
                if ("HIGH".equals(rawPrediction.getRiskLevel())
                        || "CRITICAL".equals(rawPrediction.getRiskLevel())) {
                    ctx.output(HIGH_RISK_TAG, rawPrediction);
                }

                // Side output: audit
                ctx.output(AUDIT_TAG, rawPrediction);

                if (elapsedMs > 100) {
                    LOG.warn("ONNX inference slow for {} on {}: {}ms",
                        category, state.getPatientId(), elapsedMs);
                }

            } catch (Exception e) {
                LOG.error("Inference failed for {} on {}: {}",
                    category, state.getPatientId(), e.getMessage());
            }
        }
    }

    // ═══════════════════════════════════════════
    // State management helpers
    // ═══════════════════════════════════════════

    private PatientMLState getOrCreateState(String patientId) throws Exception {
        PatientMLState state = patientState.value();
        if (state == null) {
            state = new PatientMLState();
            state.setPatientId(patientId);
            state.setFirstEventTime(System.currentTimeMillis());
        }
        return state;
    }

    @SuppressWarnings("unchecked")
    private void updateStateFromCDS(PatientMLState state, EnrichedPatientContext cds) {
        PatientContextState ps = cds.getPatientState();
        if (ps == null) return;

        // Vitals — normalize keys from production format
        if (ps.getLatestVitals() != null) {
            Map<String, Double> vitals = new HashMap<>();
            for (Map.Entry<String, Object> e : ps.getLatestVitals().entrySet()) {
                if (e.getValue() instanceof Number) {
                    vitals.put(e.getKey(), ((Number) e.getValue()).doubleValue());
                }
            }
            state.setLatestVitals(vitals);
        }

        // Labs — extract numeric values with null safety (Gap 1)
        if (ps.getRecentLabs() != null) {
            Map<String, Double> labs = new HashMap<>();
            for (Map.Entry<String, LabResult> e : ps.getRecentLabs().entrySet()) {
                LabResult labResult = e.getValue();
                if (labResult != null && labResult.getValue() != null) {
                    String key = labResult.getLabType() != null
                        ? labResult.getLabType().toLowerCase() : e.getKey();
                    labs.put(key, labResult.getValue());
                }
            }
            state.setLatestLabs(labs);
        }

        // Clinical scores
        state.setNews2Score(ps.getNews2Score() != null ? ps.getNews2Score() : 0);
        state.setQsofaScore(ps.getQsofaScore() != null ? ps.getQsofaScore() : 0);
        state.setAcuityScore(ps.getCombinedAcuityScore() != null ? ps.getCombinedAcuityScore() : 0.0);
        state.pushNews2(state.getNews2Score());
        state.pushAcuity(state.getAcuityScore());

        // Risk indicators — manual POJO to Map conversion (no toMap() method exists)
        if (ps.getRiskIndicators() != null) {
            RiskIndicators ri = ps.getRiskIndicators();
            Map<String, Object> riskMap = new HashMap<>();
            riskMap.put("tachycardia", ri.isTachycardia());
            riskMap.put("hypotension", ri.isHypotension());
            riskMap.put("fever", ri.isFever());
            riskMap.put("hypoxia", ri.isHypoxia());
            riskMap.put("elevatedLactate", ri.isElevatedLactate());
            riskMap.put("severelyElevatedLactate", ri.isSeverelyElevatedLactate());
            riskMap.put("elevatedCreatinine", ri.isElevatedCreatinine());
            riskMap.put("hyperkalemia", ri.isHyperkalemia());
            riskMap.put("thrombocytopenia", ri.isThrombocytopenia());
            riskMap.put("onAnticoagulation", ri.isOnAnticoagulation());
            riskMap.put("onVasopressors", ri.isOnVasopressors());
            riskMap.put("sepsisRisk", ri.getSepsisRisk());
            riskMap.put("leukocytosis", ri.isLeukocytosis());
            state.setRiskIndicators(riskMap);
        }

        // Active alerts (Gap 4) — SimpleAlert has enum getters
        if (ps.getActiveAlerts() != null) {
            Map<String, Object> alertMap = new HashMap<>();
            for (SimpleAlert alert : ps.getActiveAlerts()) {
                alertMap.put(
                    alert.getAlertType() != null ? alert.getAlertType().name() : "UNKNOWN",
                    Map.of(
                        "severity", alert.getSeverity() != null ? alert.getSeverity().name() : "UNKNOWN",
                        "message", alert.getMessage() != null ? alert.getMessage() : ""
                    ));
            }
            state.setActiveAlerts(alertMap);
        }

        state.setTotalEventCount(state.getTotalEventCount() + 1);
    }

    private static ONNXModelContainer.ModelType mapCategoryToModelType(String category) {
        return switch (category) {
            case "readmission" -> ONNXModelContainer.ModelType.READMISSION_RISK;
            case "sepsis" -> ONNXModelContainer.ModelType.SEPSIS_ONSET;
            case "deterioration" -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
            case "fall" -> ONNXModelContainer.ModelType.FALL_RISK;
            case "mortality" -> ONNXModelContainer.ModelType.MORTALITY_PREDICTION;
            default -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
        };
    }

    @Override
    public void close() throws Exception {
        if (modelSessions != null) {
            for (ONNXModelContainer model : modelSessions.values()) {
                model.close();
            }
        }
        super.close();
    }
}
