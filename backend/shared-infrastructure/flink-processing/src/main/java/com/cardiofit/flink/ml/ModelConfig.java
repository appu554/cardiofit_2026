package com.cardiofit.flink.ml;

import java.io.Serializable;

/**
 * Configuration for ML model execution
 *
 * Defines ONNX Runtime settings, performance tuning, and model behavior parameters.
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelConfig implements Serializable {
    private static final long serialVersionUID = 1L;

    // Model file path (optional, falls back to classpath loading)
    private String modelPath;

    // Input/output dimensions
    private int inputDimension;
    private int outputDimension;

    // Prediction threshold for risk categorization
    private double predictionThreshold;

    // Batch inference configuration
    private boolean enableBatchInference;
    private int batchSize;
    private long batchTimeoutMs;

    // ONNX Runtime threading configuration
    private int intraOpThreads;  // Threads for parallelizing operations within a single node
    private int interOpThreads;  // Threads for parallelizing across multiple nodes

    // Explainability
    private boolean enableExplainability;
    private String explainabilityMethod;

    // Model versioning
    private String modelVersion;
    private boolean enableModelVersioning;

    // Performance optimization
    private boolean enableMemoryPatternOptimization;
    private boolean enableCpuMemArena;

    // Cloud storage configuration (future use)
    private boolean cloudStorageEnabled;
    private String cloudStorageUrl;

    /**
     * Private constructor - use Builder pattern
     */
    private ModelConfig(Builder builder) {
        this.modelPath = builder.modelPath;
        this.inputDimension = builder.inputDimension;
        this.outputDimension = builder.outputDimension;
        this.predictionThreshold = builder.predictionThreshold;
        this.enableBatchInference = builder.enableBatchInference;
        this.batchSize = builder.batchSize;
        this.batchTimeoutMs = builder.batchTimeoutMs;
        this.intraOpThreads = builder.intraOpThreads;
        this.interOpThreads = builder.interOpThreads;
        this.enableExplainability = builder.enableExplainability;
        this.explainabilityMethod = builder.explainabilityMethod;
        this.modelVersion = builder.modelVersion;
        this.enableModelVersioning = builder.enableModelVersioning;
        this.enableMemoryPatternOptimization = builder.enableMemoryPatternOptimization;
        this.enableCpuMemArena = builder.enableCpuMemArena;
        this.cloudStorageEnabled = builder.cloudStorageEnabled;
        this.cloudStorageUrl = builder.cloudStorageUrl;
    }

    // Getters
    public String getModelPath() { return modelPath; }
    public int getInputDimension() { return inputDimension; }
    public int getOutputDimension() { return outputDimension; }
    public double getPredictionThreshold() { return predictionThreshold; }
    public boolean isEnableBatchInference() { return enableBatchInference; }
    public int getBatchSize() { return batchSize; }
    public long getBatchTimeoutMs() { return batchTimeoutMs; }
    public int getIntraOpThreads() { return intraOpThreads; }
    public int getInterOpThreads() { return interOpThreads; }
    public boolean isEnableExplainability() { return enableExplainability; }
    public String getExplainabilityMethod() { return explainabilityMethod; }
    public String getModelVersion() { return modelVersion; }
    public boolean isEnableModelVersioning() { return enableModelVersioning; }
    public boolean isEnableMemoryPatternOptimization() { return enableMemoryPatternOptimization; }
    public boolean isEnableCpuMemArena() { return enableCpuMemArena; }
    public boolean isCloudStorageEnabled() { return cloudStorageEnabled; }
    public String getCloudStorageUrl() { return cloudStorageUrl; }

    /**
     * Create default configuration for clinical models
     */
    public static ModelConfig createDefault() {
        return builder()
            .inputDimension(70)
            .outputDimension(2)  // [prediction, confidence]
            .predictionThreshold(0.5)
            .enableBatchInference(false)
            .batchSize(10)
            .batchTimeoutMs(1000)
            .intraOpThreads(4)
            .interOpThreads(2)
            .enableExplainability(false)
            .explainabilityMethod("SHAP")
            .enableModelVersioning(true)
            .enableMemoryPatternOptimization(true)
            .enableCpuMemArena(true)
            .build();
    }

    /**
     * Create configuration for high-throughput scenarios
     */
    public static ModelConfig createHighThroughput() {
        return builder()
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.5)
            .enableBatchInference(true)
            .batchSize(32)
            .batchTimeoutMs(500)
            .intraOpThreads(8)
            .interOpThreads(4)
            .enableExplainability(false)
            .enableMemoryPatternOptimization(true)
            .enableCpuMemArena(true)
            .build();
    }

    /**
     * Create configuration for low-latency scenarios
     */
    public static ModelConfig createLowLatency() {
        return builder()
            .inputDimension(70)
            .outputDimension(2)
            .predictionThreshold(0.5)
            .enableBatchInference(false)
            .intraOpThreads(2)
            .interOpThreads(1)
            .enableExplainability(false)
            .enableMemoryPatternOptimization(false)
            .enableCpuMemArena(false)
            .build();
    }

    @Override
    public String toString() {
        return "ModelConfig{" +
            "modelPath='" + modelPath + '\'' +
            ", inputDimension=" + inputDimension +
            ", outputDimension=" + outputDimension +
            ", predictionThreshold=" + predictionThreshold +
            ", enableBatchInference=" + enableBatchInference +
            ", batchSize=" + batchSize +
            ", intraOpThreads=" + intraOpThreads +
            ", interOpThreads=" + interOpThreads +
            ", enableExplainability=" + enableExplainability +
            '}';
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String modelPath;
        private int inputDimension = 70;
        private int outputDimension = 2;
        private double predictionThreshold = 0.5;
        private boolean enableBatchInference = false;
        private int batchSize = 10;
        private long batchTimeoutMs = 1000;
        private int intraOpThreads = 4;
        private int interOpThreads = 2;
        private boolean enableExplainability = false;
        private String explainabilityMethod = "SHAP";
        private String modelVersion = "1.0.0";
        private boolean enableModelVersioning = true;
        private boolean enableMemoryPatternOptimization = true;
        private boolean enableCpuMemArena = true;
        private boolean cloudStorageEnabled = false;
        private String cloudStorageUrl;

        public Builder modelPath(String modelPath) {
            this.modelPath = modelPath;
            return this;
        }

        public Builder inputDimension(int inputDimension) {
            this.inputDimension = inputDimension;
            return this;
        }

        public Builder outputDimension(int outputDimension) {
            this.outputDimension = outputDimension;
            return this;
        }

        public Builder predictionThreshold(double predictionThreshold) {
            this.predictionThreshold = predictionThreshold;
            return this;
        }

        public Builder enableBatchInference(boolean enableBatchInference) {
            this.enableBatchInference = enableBatchInference;
            return this;
        }

        public Builder batchSize(int batchSize) {
            this.batchSize = batchSize;
            return this;
        }

        public Builder batchTimeoutMs(long batchTimeoutMs) {
            this.batchTimeoutMs = batchTimeoutMs;
            return this;
        }

        public Builder intraOpThreads(int intraOpThreads) {
            this.intraOpThreads = intraOpThreads;
            return this;
        }

        public Builder interOpThreads(int interOpThreads) {
            this.interOpThreads = interOpThreads;
            return this;
        }

        public Builder enableExplainability(boolean enableExplainability) {
            this.enableExplainability = enableExplainability;
            return this;
        }

        public Builder explainabilityMethod(String explainabilityMethod) {
            this.explainabilityMethod = explainabilityMethod;
            return this;
        }

        public Builder modelVersion(String modelVersion) {
            this.modelVersion = modelVersion;
            return this;
        }

        public Builder enableModelVersioning(boolean enableModelVersioning) {
            this.enableModelVersioning = enableModelVersioning;
            return this;
        }

        public Builder enableMemoryPatternOptimization(boolean enableMemoryPatternOptimization) {
            this.enableMemoryPatternOptimization = enableMemoryPatternOptimization;
            return this;
        }

        public Builder enableCpuMemArena(boolean enableCpuMemArena) {
            this.enableCpuMemArena = enableCpuMemArena;
            return this;
        }

        public Builder cloudStorageEnabled(boolean cloudStorageEnabled) {
            this.cloudStorageEnabled = cloudStorageEnabled;
            return this;
        }

        public Builder cloudStorageUrl(String cloudStorageUrl) {
            this.cloudStorageUrl = cloudStorageUrl;
            return this;
        }

        public ModelConfig build() {
            return new ModelConfig(this);
        }
    }
}
