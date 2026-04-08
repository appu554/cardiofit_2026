package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module5TestBuilder;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.models.MLPrediction;
import com.cardiofit.flink.models.PatientMLState;
import org.junit.jupiter.api.*;

import static org.assertj.core.api.Assertions.*;

import java.io.File;
import java.util.*;

/**
 * Module 5 v3.0.0 ONNX Integration Tests.
 *
 * Validates the ACTUAL production pipeline:
 *   Module5FeatureExtractor (55 features) → ONNXModelContainer → MLPrediction
 *
 * Uses v3.0.0 mock models in models/{category}/model.onnx.
 * These are XGBoost ONNX models with 55-feature input, trained on
 * synthetic clinically-weighted data.
 *
 * Tests exercise:
 * - Model loading for all 5 categories
 * - Feature extraction → ONNX inference path
 * - Output format (labels + probabilities)
 * - Calibration + risk classification
 * - Batch inference
 * - Inference latency
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
@DisplayName("Module 5 v3.0.0: ONNX Integration (55-feature production pipeline)")
class Module5OnnxIntegrationTest {

    private static final String MODELS_DIR = "models";
    private static final String[] CATEGORIES = {
        "sepsis", "deterioration", "mortality", "readmission", "fall"
    };

    private static Map<String, ONNXModelContainer> models;

    @BeforeAll
    static void loadModels() {
        models = new HashMap<>();

        for (String category : CATEGORIES) {
            String modelPath = MODELS_DIR + "/" + category + "/model.onnx";
            File modelFile = new File(modelPath);

            if (!modelFile.exists()) {
                System.out.println("WARNING: Model not found: " + modelPath
                    + " — run: python scripts/create_v3_mock_models.py");
                continue;
            }

            try {
                List<String> featureNames = new ArrayList<>();
                for (int i = 0; i < Module5FeatureExtractor.FEATURE_COUNT; i++) {
                    featureNames.add("f" + i);
                }

                ONNXModelContainer model = ONNXModelContainer.builder()
                    .modelId(category + "_v3")
                    .modelName(category + " predictor v3.0.0")
                    .modelType(mapCategoryToModelType(category))
                    .modelVersion("3.0.0")
                    .inputFeatureNames(featureNames)
                    .config(ModelConfig.builder()
                        .predictionThreshold(0.5)
                        .intraOpThreads(1)
                        .interOpThreads(1)
                        .modelPath(modelPath)
                        .build())
                    .build();
                model.initialize();
                models.put(category, model);
            } catch (Exception e) {
                System.out.println("FAILED to load " + category + ": " + e.getMessage());
            }
        }

        System.out.println("Loaded " + models.size() + "/" + CATEGORIES.length + " models");
    }

    @AfterAll
    static void closeModels() {
        for (ONNXModelContainer model : models.values()) {
            try { model.close(); } catch (Exception ignored) {}
        }
    }

    // ════════════════════════════════════════════
    // Model Loading
    // ════════════════════════════════════════════

    @Test
    @Order(1)
    @DisplayName("All 5 models load with 55-feature input")
    void allModelsLoadedWith55Features() {
        assertThat(models).hasSize(5);
        for (String category : CATEGORIES) {
            assertThat(models).containsKey(category);
        }
    }

    // ════════════════════════════════════════════
    // Feature Extraction → Inference
    // ════════════════════════════════════════════

    @Test
    @Order(2)
    @DisplayName("Stable patient: extractFeatures → predict produces valid output")
    void stablePatientInference() throws Exception {
        PatientMLState state = Module5TestBuilder.stablePatientState("test-stable");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        assertThat(features).hasSize(55);

        for (ONNXModelContainer model : models.values()) {
            MLPrediction prediction = model.predict(features);
            assertThat(prediction).isNotNull();
            assertThat(prediction.getPrimaryScore()).isBetween(0.0, 1.0);
        }
    }

    @Test
    @Order(3)
    @DisplayName("Sepsis patient: extractFeatures → predict produces valid output")
    void sepsisPatientInference() throws Exception {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("test-sepsis");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        assertThat(features).hasSize(55);

        MLPrediction prediction = models.get("sepsis").predict(features);
        assertThat(prediction).isNotNull();
        assertThat(prediction.getPrimaryScore()).isBetween(0.0, 1.0);
    }

    @Test
    @Order(4)
    @DisplayName("AKI patient: extractFeatures → predict produces valid output")
    void akiPatientInference() throws Exception {
        PatientMLState state = Module5TestBuilder.akiPatientState("test-aki");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        assertThat(features).hasSize(55);

        for (ONNXModelContainer model : models.values()) {
            MLPrediction prediction = model.predict(features);
            assertThat(prediction).isNotNull();
            assertThat(prediction.getPrimaryScore()).isBetween(0.0, 1.0);
        }
    }

    @Test
    @Order(5)
    @DisplayName("Sparse patient (minimal data): no NaN/Inf in predictions")
    void sparsePatientInference() throws Exception {
        PatientMLState state = Module5TestBuilder.sparsePatientState("test-sparse");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        assertThat(features).hasSize(55);

        // Verify no NaN/Inf in features
        for (int i = 0; i < features.length; i++) {
            assertThat(Float.isNaN(features[i])).as("Feature %d is NaN", i).isFalse();
            assertThat(Float.isInfinite(features[i])).as("Feature %d is Inf", i).isFalse();
        }

        for (ONNXModelContainer model : models.values()) {
            MLPrediction prediction = model.predict(features);
            assertThat(prediction.getPrimaryScore()).isFinite();
        }
    }

    // ════════════════════════════════════════════
    // Calibration + Risk Classification
    // ════════════════════════════════════════════

    @Test
    @Order(6)
    @DisplayName("Calibration + risk classification produces valid levels")
    void calibrationAndRiskClassification() throws Exception {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("test-calibration");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        MLPrediction rawPrediction = models.get("sepsis").predict(features);
        double rawScore = rawPrediction.getPrimaryScore();

        // Calibrate
        double calibrated = Module5ClinicalScoring.calibrate(rawScore, "sepsis");
        assertThat(calibrated).isBetween(0.0, 1.0);

        // Classify
        String riskLevel = Module5ClinicalScoring.classifyRiskLevel(calibrated, "sepsis");
        assertThat(riskLevel).isIn("LOW", "MODERATE", "HIGH", "CRITICAL");
    }

    @Test
    @Order(7)
    @DisplayName("All 5 categories produce valid calibrated risk levels")
    void allCategoriesCalibrate() throws Exception {
        PatientMLState state = Module5TestBuilder.drugLabPatientState("test-drug-lab");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        for (String category : CATEGORIES) {
            MLPrediction prediction = models.get(category).predict(features);
            double calibrated = Module5ClinicalScoring.calibrate(
                prediction.getPrimaryScore(), category);
            String riskLevel = Module5ClinicalScoring.classifyRiskLevel(calibrated, category);

            assertThat(calibrated).as(category + " calibrated score").isBetween(0.0, 1.0);
            assertThat(riskLevel).as(category + " risk level")
                .isIn("LOW", "MODERATE", "HIGH", "CRITICAL");
        }
    }

    // ════════════════════════════════════════════
    // Batch Inference
    // ════════════════════════════════════════════

    @Test
    @Order(8)
    @DisplayName("Batch inference (32 patients) produces correct count")
    void batchInference() throws Exception {
        int batchSize = 32;
        List<float[]> batch = new ArrayList<>();

        PatientMLState[] patients = {
            Module5TestBuilder.stablePatientState("batch-stable"),
            Module5TestBuilder.sepsisPatientState("batch-sepsis"),
            Module5TestBuilder.akiPatientState("batch-aki"),
            Module5TestBuilder.drugLabPatientState("batch-drug-lab"),
            Module5TestBuilder.sparsePatientState("batch-sparse"),
        };

        for (int i = 0; i < batchSize; i++) {
            PatientMLState state = patients[i % patients.length];
            batch.add(Module5FeatureExtractor.extractFeatures(state));
        }

        List<MLPrediction> predictions = models.get("sepsis").predictBatch(batch);
        assertThat(predictions).hasSize(batchSize);
        for (MLPrediction p : predictions) {
            assertThat(p.getPrimaryScore()).isBetween(0.0, 1.0);
        }
    }

    // ════════════════════════════════════════════
    // Latency
    // ════════════════════════════════════════════

    @Test
    @Order(9)
    @DisplayName("Single inference latency < 50ms")
    void singleInferenceLatency() throws Exception {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("latency-test");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Warmup
        for (int i = 0; i < 100; i++) {
            models.get("sepsis").predict(features);
        }

        // Measure
        long start = System.nanoTime();
        MLPrediction prediction = models.get("sepsis").predict(features);
        long latencyMs = (System.nanoTime() - start) / 1_000_000;

        assertThat(prediction).isNotNull();
        assertThat(latencyMs).as("Single inference latency").isLessThan(50);
    }

    @Test
    @Order(10)
    @DisplayName("All 5 models inference < 100ms total")
    void allModelsInference() throws Exception {
        PatientMLState state = Module5TestBuilder.sepsisPatientState("all-models-test");
        float[] features = Module5FeatureExtractor.extractFeatures(state);

        // Warmup
        for (int i = 0; i < 50; i++) {
            for (ONNXModelContainer model : models.values()) {
                model.predict(features);
            }
        }

        // Measure all 5 sequentially
        long start = System.nanoTime();
        for (ONNXModelContainer model : models.values()) {
            model.predict(features);
        }
        long totalMs = (System.nanoTime() - start) / 1_000_000;

        assertThat(totalMs).as("5-model total inference").isLessThan(100);
    }

    // ════════════════════════════════════════════
    // Helpers
    // ════════════════════════════════════════════

    private static ONNXModelContainer.ModelType mapCategoryToModelType(String category) {
        return switch (category) {
            case "sepsis" -> ONNXModelContainer.ModelType.SEPSIS_ONSET;
            case "deterioration" -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
            case "mortality" -> ONNXModelContainer.ModelType.MORTALITY_PREDICTION;
            case "readmission" -> ONNXModelContainer.ModelType.READMISSION_RISK;
            case "fall" -> ONNXModelContainer.ModelType.FALL_RISK;
            default -> ONNXModelContainer.ModelType.CLINICAL_DETERIORATION;
        };
    }
}
