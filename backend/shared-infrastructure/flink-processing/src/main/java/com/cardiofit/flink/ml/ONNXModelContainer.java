package com.cardiofit.flink.ml;

import ai.onnxruntime.*;
import com.cardiofit.flink.models.MLPrediction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.io.Serializable;
import java.nio.FloatBuffer;
import java.util.*;

/**
 * ONNX Model Container for Flink Streaming
 *
 * Handles ONNX model loading, inference, and lifecycle management in Flink streaming context.
 * Designed for real-time clinical ML predictions with sub-50ms latency.
 *
 * Features:
 * - ONNX Runtime integration with optimized session configuration
 * - Single and batch inference support
 * - Performance metrics tracking (latency, throughput)
 * - Automatic resource cleanup
 * - Thread-safe inference execution
 *
 * Usage:
 * <pre>
 * ONNXModelContainer model = ONNXModelContainer.builder()
 *     .modelId("sepsis_prediction_v1")
 *     .modelName("Sepsis Onset Predictor")
 *     .modelType(ModelType.SEPSIS_ONSET)
 *     .modelVersion("1.0.0")
 *     .inputFeatureNames(featureNames)
 *     .outputNames(Arrays.asList("sepsis_probability"))
 *     .config(modelConfig)
 *     .build();
 *
 * model.initialize();
 * MLPrediction prediction = model.predict(features);
 * model.close();
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ONNXModelContainer implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ONNXModelContainer.class);

    // Model identification
    private final String modelId;
    private final String modelName;
    private final ModelType modelType;
    private final String modelVersion;

    // ONNX Runtime components (transient - not serialized)
    private transient OrtEnvironment environment;
    private transient OrtSession session;
    private transient Map<String, NodeInfo> inputInfo;
    private transient Map<String, NodeInfo> outputInfo;

    // Model configuration
    private final List<String> inputFeatureNames;
    private final List<String> outputNames;
    private final ModelConfig config;

    // Performance tracking
    private transient long inferenceCount = 0;
    private transient long totalInferenceTime = 0;
    private transient long lastInferenceTimestamp = 0;

    // Model types supported
    public enum ModelType {
        MORTALITY_PREDICTION("30-day mortality risk"),
        SEPSIS_ONSET("Sepsis onset prediction (6-hour horizon)"),
        READMISSION_RISK("30-day readmission risk"),
        AKI_PROGRESSION("Acute kidney injury progression"),
        CLINICAL_DETERIORATION("Clinical deterioration risk"),
        FALL_RISK("Patient fall risk"),
        LENGTH_OF_STAY("Length of stay prediction");

        private final String description;

        ModelType(String description) {
            this.description = description;
        }

        public String getDescription() {
            return description;
        }
    }

    /**
     * Private constructor - use Builder pattern
     */
    private ONNXModelContainer(Builder builder) {
        this.modelId = builder.modelId;
        this.modelName = builder.modelName;
        this.modelType = builder.modelType;
        this.modelVersion = builder.modelVersion;
        this.inputFeatureNames = builder.inputFeatureNames;
        this.outputNames = builder.outputNames;
        this.config = builder.config;
    }

    /**
     * Initialize ONNX Runtime session
     *
     * Called once per TaskManager (not per record). Sets up:
     * - ONNX Runtime environment
     * - Model session with optimization
     * - Input/output metadata
     *
     * @throws OrtException if model initialization fails
     */
    public void initialize() throws OrtException {
        try {
            LOG.info("Initializing ONNX model: {} v{}", modelName, modelVersion);

            // Create ONNX Runtime environment (shared across models)
            this.environment = OrtEnvironment.getEnvironment();

            // Load model bytes
            byte[] modelBytes = loadModelBytes();
            LOG.info("Loaded model bytes: {} bytes", modelBytes.length);

            // Create session options with optimization
            OrtSession.SessionOptions options = new OrtSession.SessionOptions();

            // Optimization level (ALL_OPT for maximum performance)
            options.setOptimizationLevel(OrtSession.SessionOptions.OptLevel.ALL_OPT);

            // Thread configuration for parallel inference
            options.setIntraOpNumThreads(config.getIntraOpThreads());
            options.setInterOpNumThreads(config.getInterOpThreads());

            // Enable memory pattern optimization
            options.setMemoryPatternOptimization(true);

            // Create ONNX session
            this.session = environment.createSession(modelBytes, options);

            // Extract input/output metadata
            this.inputInfo = new HashMap<>();
            this.outputInfo = new HashMap<>();

            for (Map.Entry<String, NodeInfo> entry : session.getInputInfo().entrySet()) {
                inputInfo.put(entry.getKey(), entry.getValue());
            }

            for (Map.Entry<String, NodeInfo> entry : session.getOutputInfo().entrySet()) {
                outputInfo.put(entry.getKey(), entry.getValue());
            }

            LOG.info("ONNX model initialized successfully: {} (inputs: {}, outputs: {})",
                modelName, inputInfo.size(), outputInfo.size());

            // Log input shapes for validation
            for (Map.Entry<String, NodeInfo> entry : inputInfo.entrySet()) {
                LOG.debug("Input '{}': {}", entry.getKey(), entry.getValue().getInfo());
            }

        } catch (IOException e) {
            throw new OrtException(OrtException.OrtErrorCode.ORT_FAIL,
                "Failed to load model bytes for: " + modelId + " - " + e.getMessage());
        }
    }

    /**
     * Perform inference on a single patient
     *
     * @param features Feature vector (must match model input schema)
     * @return ML prediction with scores, confidence, and metadata
     * @throws OrtException if inference fails
     */
    public MLPrediction predict(float[] features) throws OrtException {
        long startTime = System.nanoTime();

        try {
            // Validate input dimensions
            if (features.length != inputFeatureNames.size()) {
                throw new IllegalArgumentException(String.format(
                    "Feature dimension mismatch: expected %d, got %d",
                    inputFeatureNames.size(), features.length
                ));
            }

            // Determine batch size (single prediction)
            long[] shape = {1, features.length};

            // Create input tensor
            FloatBuffer buffer = FloatBuffer.wrap(features);
            OnnxTensor inputTensor = OnnxTensor.createTensor(environment, buffer, shape);

            // Prepare inputs map
            Map<String, OnnxTensor> inputs = new HashMap<>();
            String inputName = session.getInputNames().iterator().next();
            inputs.put(inputName, inputTensor);

            // Run inference
            OrtSession.Result result = session.run(inputs);

            // Extract output
            float[] outputArray = extractSingleOutput(result);

            // Calculate inference time
            long inferenceTimeNs = System.nanoTime() - startTime;
            long inferenceTimeMs = inferenceTimeNs / 1_000_000;

            // Build prediction object
            MLPrediction prediction = buildPrediction(features, outputArray, inferenceTimeMs);

            // Update metrics
            updateMetrics(inferenceTimeMs);

            // Cleanup
            inputTensor.close();
            result.close();

            return prediction;

        } catch (Exception e) {
            LOG.error("Inference failed for model: " + modelId, e);
            throw new OrtException(OrtException.OrtErrorCode.ORT_FAIL,
                "Inference failed for model: " + modelId + " - " + e.getMessage());
        }
    }

    /**
     * Batch inference for multiple patients
     *
     * More efficient than individual calls due to tensor batching.
     *
     * @param featureBatch List of feature vectors
     * @return List of ML predictions
     * @throws OrtException if batch inference fails
     */
    public List<MLPrediction> predictBatch(List<float[]> featureBatch) throws OrtException {
        long startTime = System.nanoTime();

        try {
            int batchSize = featureBatch.size();
            int featureCount = inputFeatureNames.size();

            // Validate all features have correct dimensions
            for (int i = 0; i < batchSize; i++) {
                if (featureBatch.get(i).length != featureCount) {
                    throw new IllegalArgumentException(String.format(
                        "Feature dimension mismatch at index %d: expected %d, got %d",
                        i, featureCount, featureBatch.get(i).length
                    ));
                }
            }

            // Flatten features into single array (row-major order)
            float[] flattenedFeatures = new float[batchSize * featureCount];
            for (int i = 0; i < batchSize; i++) {
                System.arraycopy(featureBatch.get(i), 0,
                    flattenedFeatures, i * featureCount, featureCount);
            }

            // Create batched input tensor
            long[] shape = {batchSize, featureCount};
            FloatBuffer buffer = FloatBuffer.wrap(flattenedFeatures);
            OnnxTensor inputTensor = OnnxTensor.createTensor(environment, buffer, shape);

            // Prepare inputs map
            Map<String, OnnxTensor> inputs = new HashMap<>();
            String inputName = session.getInputNames().iterator().next();
            inputs.put(inputName, inputTensor);

            // Run batch inference
            OrtSession.Result result = session.run(inputs);

            // Extract batch outputs
            float[][] outputs = extractBatchOutput(result, batchSize);

            // Calculate inference time
            long inferenceTimeNs = System.nanoTime() - startTime;
            long batchInferenceTimeMs = inferenceTimeNs / 1_000_000;
            long avgInferenceTimeMs = batchInferenceTimeMs / batchSize;

            // Build prediction objects
            List<MLPrediction> predictions = new ArrayList<>(batchSize);
            for (int i = 0; i < batchSize; i++) {
                MLPrediction prediction = buildPrediction(
                    featureBatch.get(i), outputs[i], avgInferenceTimeMs
                );
                predictions.add(prediction);
            }

            // Update metrics (count as batchSize inferences)
            updateMetrics(batchInferenceTimeMs, batchSize);

            // Cleanup
            inputTensor.close();
            result.close();

            LOG.debug("Batch inference completed: {} predictions in {}ms (avg: {}ms)",
                batchSize, batchInferenceTimeMs, avgInferenceTimeMs);

            return predictions;

        } catch (Exception e) {
            LOG.error("Batch inference failed for model: " + modelId, e);
            throw new OrtException(OrtException.OrtErrorCode.ORT_FAIL,
                "Batch inference failed for model: " + modelId + " - " + e.getMessage());
        }
    }

    /**
     * Extract single prediction output from ONNX result
     * Note: XGBoost ONNX models output TWO tensors:
     *   - Output[0]: Class labels (INT64) - binary predictions
     *   - Output[1]: Probabilities (FLOAT) - confidence scores
     * We access output[1] for clinical risk scoring.
     */
    private float[] extractSingleOutput(OrtSession.Result result) throws OrtException {
        OnnxValue outputValue = result.get(1);  // Get probabilities (FLOAT), not class labels (INT64)

        if (outputValue instanceof OnnxTensor) {
            OnnxTensor outputTensor = (OnnxTensor) outputValue;
            float[][] output2D = (float[][]) outputTensor.getValue();
            return output2D[0];  // First (and only) batch element
        }

        throw new OrtException("Unexpected output type: " + outputValue.getClass().getName());
    }

    /**
     * Extract batch predictions from ONNX result
     * Note: Accesses output[1] for probabilities (FLOAT), not output[0] (class labels INT64)
     */
    private float[][] extractBatchOutput(OrtSession.Result result, int expectedBatchSize) throws OrtException {
        OnnxValue outputValue = result.get(1);  // Get probabilities (FLOAT), not class labels (INT64)

        if (outputValue instanceof OnnxTensor) {
            OnnxTensor outputTensor = (OnnxTensor) outputValue;
            float[][] outputs = (float[][]) outputTensor.getValue();

            if (outputs.length != expectedBatchSize) {
                throw new OrtException(String.format(
                    "Batch size mismatch: expected %d, got %d",
                    expectedBatchSize, outputs.length
                ));
            }

            return outputs;
        }

        throw new OrtException("Unexpected output type: " + outputValue.getClass().getName());
    }

    /**
     * Build ML prediction object from model output
     */
    private MLPrediction buildPrediction(float[] features, float[] output, long inferenceTimeMs) {
        MLPrediction prediction = new MLPrediction();

        // Set model metadata
        prediction.setId(UUID.randomUUID().toString());
        prediction.setModelName(modelName);
        prediction.setModelType(modelType.name());
        prediction.setPredictionTime(System.currentTimeMillis());
        prediction.setInputFeatureCount(features.length);

        // Set prediction scores
        Map<String, Double> scores = new HashMap<>();
        scores.put("primary_score", (double) output[0]);

        if (output.length > 1) {
            scores.put("confidence_score", (double) output[1]);
        }

        prediction.setPredictionScores(scores);

        // Determine risk level
        String riskLevel = determineRiskLevel(output[0]);
        prediction.setRiskLevel(riskLevel);

        // Set confidence
        if (output.length > 1) {
            prediction.setConfidence((double) output[1]);
        } else {
            prediction.setConfidence(calculateDefaultConfidence(output[0]));
        }

        // Set model metadata
        Map<String, Object> metadata = new HashMap<>();
        metadata.put("model_id", modelId);
        metadata.put("model_version", modelVersion);
        metadata.put("model_type", modelType.name());
        metadata.put("inference_time_ms", inferenceTimeMs);
        metadata.put("feature_count", features.length);
        metadata.put("onnx_runtime_version", "1.17.0");

        prediction.setModelMetadata(metadata);

        return prediction;
    }

    /**
     * Determine risk level based on prediction score
     */
    private String determineRiskLevel(float score) {
        double threshold = config.getPredictionThreshold();

        if (score >= threshold) {
            return "HIGH";
        } else if (score >= threshold * 0.7) {
            return "MODERATE";
        } else if (score >= threshold * 0.4) {
            return "LOW";
        } else {
            return "VERY_LOW";
        }
    }

    /**
     * Calculate default confidence from prediction score
     * For binary classification: max(p, 1-p)
     */
    private double calculateDefaultConfidence(float score) {
        return Math.max(score, 1.0 - score);
    }

    /**
     * Update performance metrics (single inference)
     */
    private void updateMetrics(long inferenceTimeMs) {
        updateMetrics(inferenceTimeMs, 1);
    }

    /**
     * Update performance metrics (batch inference)
     */
    private synchronized void updateMetrics(long totalTimeMs, int count) {
        inferenceCount += count;
        totalInferenceTime += totalTimeMs;
        lastInferenceTimestamp = System.currentTimeMillis();
    }

    /**
     * Get model performance metrics
     */
    public ModelMetrics getMetrics() {
        return ModelMetrics.builder()
            .modelId(modelId)
            .modelName(modelName)
            .modelType(modelType.name())
            .inferenceCount(inferenceCount)
            .totalInferenceTimeMs(totalInferenceTime)
            .averageInferenceTimeMs(
                inferenceCount > 0 ? (double) totalInferenceTime / inferenceCount : 0.0
            )
            .throughputPerSecond(
                totalInferenceTime > 0 ? (inferenceCount * 1000.0) / totalInferenceTime : 0.0
            )
            .lastInferenceTimestamp(lastInferenceTimestamp)
            .build();
    }

    /**
     * Load model bytes from storage
     *
     * Supports multiple loading strategies:
     * 1. Classpath resources (embedded in JAR)
     * 2. External file system
     * 3. Cloud storage (S3, GCS) - placeholder for future implementation
     */
    private byte[] loadModelBytes() throws IOException {
        try {
            // Strategy 1: Load from classpath (embedded in JAR)
            String resourcePath = "/models/" + modelId + ".onnx";
            InputStream modelStream = getClass().getResourceAsStream(resourcePath);

            if (modelStream != null) {
                LOG.info("Loading model from classpath: {}", resourcePath);
                return modelStream.readAllBytes();
            }

            // Strategy 2: Load from external file system (if classpath fails)
            String externalPath = config.getModelPath();
            if (externalPath != null && !externalPath.isEmpty()) {
                LOG.info("Loading model from external path: {}", externalPath);
                return java.nio.file.Files.readAllBytes(java.nio.file.Paths.get(externalPath));
            }

            // Strategy 3: Cloud storage (future implementation)
            // if (config.isCloudStorageEnabled()) {
            //     return loadFromCloudStorage(config.getCloudStorageUrl());
            // }

            throw new IOException("Model not found: " + modelId +
                " (checked classpath and external path)");

        } catch (IOException e) {
            LOG.error("Failed to load model: {}", modelId, e);
            throw e;
        }
    }

    /**
     * Cleanup ONNX resources
     *
     * Called when TaskManager shuts down or during graceful termination.
     * Releases ONNX Runtime resources to prevent memory leaks.
     */
    public void close() {
        try {
            if (session != null) {
                session.close();
                session = null;
            }

            // Note: OrtEnvironment is shared and should not be closed here
            // It will be cleaned up by ONNX Runtime automatically

            LOG.info("Closed ONNX model: {} (processed {} inferences, avg latency: {:.2f}ms)",
                modelName,
                inferenceCount,
                inferenceCount > 0 ? (double) totalInferenceTime / inferenceCount : 0.0
            );

        } catch (Exception e) {
            LOG.error("Error closing ONNX model: " + modelId, e);
        }
    }

    // Getters
    public String getModelId() { return modelId; }
    public String getModelName() { return modelName; }
    public ModelType getModelType() { return modelType; }
    public String getModelVersion() { return modelVersion; }
    public List<String> getInputFeatureNames() { return inputFeatureNames; }
    public List<String> getOutputNames() { return outputNames; }
    public ModelConfig getConfig() { return config; }
    public long getInferenceCount() { return inferenceCount; }
    public long getTotalInferenceTime() { return totalInferenceTime; }

    /**
     * Check if model is initialized and ready for inference
     */
    public boolean isInitialized() {
        return session != null && environment != null;
    }

    /**
     * Get input shape information
     */
    public long[] getInputShape() {
        if (inputInfo == null || inputInfo.isEmpty()) {
            return new long[]{1, inputFeatureNames.size()};
        }

        NodeInfo firstInput = inputInfo.values().iterator().next();
        TensorInfo tensorInfo = (TensorInfo) firstInput.getInfo();
        return tensorInfo.getShape();
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String modelId;
        private String modelName;
        private ModelType modelType;
        private String modelVersion;
        private List<String> inputFeatureNames;
        private List<String> outputNames;
        private ModelConfig config;

        public Builder modelId(String modelId) {
            this.modelId = modelId;
            return this;
        }

        public Builder modelName(String modelName) {
            this.modelName = modelName;
            return this;
        }

        public Builder modelType(ModelType modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder modelVersion(String modelVersion) {
            this.modelVersion = modelVersion;
            return this;
        }

        public Builder inputFeatureNames(List<String> inputFeatureNames) {
            this.inputFeatureNames = inputFeatureNames;
            return this;
        }

        public Builder outputNames(List<String> outputNames) {
            this.outputNames = outputNames;
            return this;
        }

        public Builder config(ModelConfig config) {
            this.config = config;
            return this;
        }

        public ONNXModelContainer build() {
            // Validation
            if (modelId == null || modelId.isEmpty()) {
                throw new IllegalArgumentException("modelId is required");
            }
            if (modelName == null || modelName.isEmpty()) {
                throw new IllegalArgumentException("modelName is required");
            }
            if (modelType == null) {
                throw new IllegalArgumentException("modelType is required");
            }
            if (inputFeatureNames == null || inputFeatureNames.isEmpty()) {
                throw new IllegalArgumentException("inputFeatureNames is required");
            }
            if (config == null) {
                throw new IllegalArgumentException("config is required");
            }

            // Default values
            if (modelVersion == null) {
                modelVersion = "1.0.0";
            }
            if (outputNames == null) {
                outputNames = Arrays.asList("output");
            }

            return new ONNXModelContainer(this);
        }
    }
}
