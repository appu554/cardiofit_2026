package com.cardiofit.flink.ml.registry;

import com.fasterxml.jackson.annotation.JsonIgnore;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.io.Serializable;
import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.Objects;

/**
 * Model Metadata for Model Registry
 *
 * Comprehensive metadata tracking for ML model versions including:
 * - Training metadata (date, dataset, hyperparameters)
 * - Performance metrics (AUROC, precision, recall, F1)
 * - Model properties (size, input/output schema, latency)
 * - Deployment tracking (status, rollout percentage)
 * - Version comparison utilities
 *
 * Design Pattern: Immutable value object with Builder pattern
 * State Management: Serializable for Flink state storage
 *
 * Usage Example:
 * <pre>
 * ModelMetadata metadata = ModelMetadata.builder()
 *     .modelType("sepsis_risk")
 *     .version("v2")
 *     .trainingDate(System.currentTimeMillis())
 *     .auroc(0.92)
 *     .precision(0.89)
 *     .recall(0.88)
 *     .modelPath("s3://models/sepsis_v2.onnx")
 *     .modelSizeBytes(15_728_640L)
 *     .avgInferenceLatencyMs(12.5)
 *     .hyperparameter("learning_rate", 0.001)
 *     .hyperparameter("hidden_layers", 3)
 *     .build();
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelMetadata implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final ObjectMapper JSON_MAPPER = new ObjectMapper();

    // Model identification
    @JsonProperty("model_type")
    private final String modelType;

    @JsonProperty("version")
    private final String version;

    @JsonProperty("model_path")
    private final String modelPath;

    // Training metadata
    @JsonProperty("training_date")
    private final long trainingDate;

    @JsonProperty("training_dataset")
    private final String trainingDataset;

    @JsonProperty("training_sample_count")
    private final long trainingSampleCount;

    @JsonProperty("hyperparameters")
    private final Map<String, Object> hyperparameters;

    // Performance metrics
    @JsonProperty("auroc")
    private final double auroc;

    @JsonProperty("precision")
    private final double precision;

    @JsonProperty("recall")
    private final double recall;

    @JsonProperty("f1_score")
    private final double f1Score;

    @JsonProperty("brier_score")
    private final double brierScore;

    @JsonProperty("calibration_slope")
    private final Double calibrationSlope;

    @JsonProperty("calibration_intercept")
    private final Double calibrationIntercept;

    // Model properties
    @JsonProperty("model_size_bytes")
    private final long modelSizeBytes;

    @JsonProperty("input_feature_count")
    private final int inputFeatureCount;

    @JsonProperty("output_dimension")
    private final int outputDimension;

    @JsonProperty("avg_inference_latency_ms")
    private final double avgInferenceLatencyMs;

    @JsonProperty("p99_inference_latency_ms")
    private final Double p99InferenceLatencyMs;

    @JsonProperty("model_framework")
    private final String modelFramework;

    @JsonProperty("onnx_opset_version")
    private final Integer onnxOpsetVersion;

    // Deployment tracking
    @JsonProperty("deployment_date")
    private final Long deploymentDate;

    @JsonProperty("approval_status")
    private final ApprovalStatus approvalStatus;

    @JsonProperty("rollout_percentage")
    private final double rolloutPercentage;

    @JsonProperty("deployment_environment")
    private final String deploymentEnvironment;

    @JsonProperty("approved_by")
    private final String approvedBy;

    @JsonProperty("approval_date")
    private final Long approvalDate;

    // Metadata
    @JsonProperty("creation_date")
    private final long creationDate;

    @JsonProperty("last_updated")
    private final long lastUpdated;

    @JsonProperty("created_by")
    private final String createdBy;

    @JsonProperty("description")
    private final String description;

    @JsonProperty("tags")
    private final Map<String, String> tags;

    /**
     * Private constructor - use Builder pattern
     */
    private ModelMetadata(Builder builder) {
        this.modelType = builder.modelType;
        this.version = builder.version;
        this.modelPath = builder.modelPath;
        this.trainingDate = builder.trainingDate;
        this.trainingDataset = builder.trainingDataset;
        this.trainingSampleCount = builder.trainingSampleCount;
        this.hyperparameters = new HashMap<>(builder.hyperparameters);
        this.auroc = builder.auroc;
        this.precision = builder.precision;
        this.recall = builder.recall;
        this.f1Score = builder.f1Score;
        this.brierScore = builder.brierScore;
        this.calibrationSlope = builder.calibrationSlope;
        this.calibrationIntercept = builder.calibrationIntercept;
        this.modelSizeBytes = builder.modelSizeBytes;
        this.inputFeatureCount = builder.inputFeatureCount;
        this.outputDimension = builder.outputDimension;
        this.avgInferenceLatencyMs = builder.avgInferenceLatencyMs;
        this.p99InferenceLatencyMs = builder.p99InferenceLatencyMs;
        this.modelFramework = builder.modelFramework;
        this.onnxOpsetVersion = builder.onnxOpsetVersion;
        this.deploymentDate = builder.deploymentDate;
        this.approvalStatus = builder.approvalStatus;
        this.rolloutPercentage = builder.rolloutPercentage;
        this.deploymentEnvironment = builder.deploymentEnvironment;
        this.approvedBy = builder.approvedBy;
        this.approvalDate = builder.approvalDate;
        this.creationDate = builder.creationDate;
        this.lastUpdated = builder.lastUpdated;
        this.createdBy = builder.createdBy;
        this.description = builder.description;
        this.tags = new HashMap<>(builder.tags);
    }

    // Getters
    public String getModelType() { return modelType; }
    public String getVersion() { return version; }
    public String getModelPath() { return modelPath; }
    public long getTrainingDate() { return trainingDate; }
    public String getTrainingDataset() { return trainingDataset; }
    public long getTrainingSampleCount() { return trainingSampleCount; }
    public Map<String, Object> getHyperparameters() { return new HashMap<>(hyperparameters); }
    public double getAuroc() { return auroc; }
    public double getPrecision() { return precision; }
    public double getRecall() { return recall; }
    public double getF1Score() { return f1Score; }
    public double getBrierScore() { return brierScore; }
    public Double getCalibrationSlope() { return calibrationSlope; }
    public Double getCalibrationIntercept() { return calibrationIntercept; }
    public long getModelSizeBytes() { return modelSizeBytes; }
    public int getInputFeatureCount() { return inputFeatureCount; }
    public int getOutputDimension() { return outputDimension; }
    public double getAvgInferenceLatencyMs() { return avgInferenceLatencyMs; }
    public Double getP99InferenceLatencyMs() { return p99InferenceLatencyMs; }
    public String getModelFramework() { return modelFramework; }
    public Integer getOnnxOpsetVersion() { return onnxOpsetVersion; }
    public Long getDeploymentDate() { return deploymentDate; }
    public ApprovalStatus getApprovalStatus() { return approvalStatus; }
    public double getRolloutPercentage() { return rolloutPercentage; }
    public String getDeploymentEnvironment() { return deploymentEnvironment; }
    public String getApprovedBy() { return approvedBy; }
    public Long getApprovalDate() { return approvalDate; }
    public long getCreationDate() { return creationDate; }
    public long getLastUpdated() { return lastUpdated; }
    public String getCreatedBy() { return createdBy; }
    public String getDescription() { return description; }
    public Map<String, String> getTags() { return new HashMap<>(tags); }

    /**
     * Check if model is approved for deployment
     */
    @JsonIgnore
    public boolean isApproved() {
        return approvalStatus == ApprovalStatus.APPROVED;
    }

    /**
     * Check if model is fully deployed (100% rollout)
     */
    @JsonIgnore
    public boolean isFullyDeployed() {
        return rolloutPercentage >= 100.0;
    }

    /**
     * Check if model is in canary deployment
     */
    @JsonIgnore
    public boolean isCanaryDeployment() {
        return rolloutPercentage > 0.0 && rolloutPercentage < 100.0;
    }

    /**
     * Compare performance with another model version
     *
     * @param other Other model metadata to compare
     * @return Performance comparison result
     */
    public PerformanceComparison comparePerformance(ModelMetadata other) {
        double aurocDelta = this.auroc - other.auroc;
        double precisionDelta = this.precision - other.precision;
        double recallDelta = this.recall - other.recall;
        double f1Delta = this.f1Score - other.f1Score;
        double latencyDelta = this.avgInferenceLatencyMs - other.avgInferenceLatencyMs;

        return new PerformanceComparison(
            this.version,
            other.version,
            aurocDelta,
            precisionDelta,
            recallDelta,
            f1Delta,
            latencyDelta,
            calculateOverallImprovement(aurocDelta, f1Delta, latencyDelta)
        );
    }

    /**
     * Calculate overall improvement score
     * Positive = improvement, Negative = regression
     */
    private double calculateOverallImprovement(double aurocDelta, double f1Delta, double latencyDelta) {
        // Weighted score: AUROC (50%), F1 (30%), Latency (20% - negative is good)
        double accuracyScore = (aurocDelta * 0.5 + f1Delta * 0.3);
        double latencyScore = -(latencyDelta / 100.0) * 0.2;  // Normalize and invert
        return accuracyScore + latencyScore;
    }

    /**
     * Export to JSON string
     */
    public String toJson() {
        try {
            return JSON_MAPPER.writeValueAsString(this);
        } catch (Exception e) {
            return "{\"error\": \"Failed to serialize metadata\"}";
        }
    }

    /**
     * Create copy with updated deployment info
     */
    public ModelMetadata withDeployment(long deploymentDate, double rolloutPercentage) {
        return ModelMetadata.builder()
            .from(this)
            .deploymentDate(deploymentDate)
            .rolloutPercentage(rolloutPercentage)
            .lastUpdated(System.currentTimeMillis())
            .build();
    }

    /**
     * Create copy with updated approval status
     */
    public ModelMetadata withApproval(ApprovalStatus status, String approvedBy) {
        return ModelMetadata.builder()
            .from(this)
            .approvalStatus(status)
            .approvedBy(approvedBy)
            .approvalDate(System.currentTimeMillis())
            .lastUpdated(System.currentTimeMillis())
            .build();
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        ModelMetadata that = (ModelMetadata) o;
        return Objects.equals(modelType, that.modelType) &&
               Objects.equals(version, that.version);
    }

    @Override
    public int hashCode() {
        return Objects.hash(modelType, version);
    }

    @Override
    public String toString() {
        return String.format("ModelMetadata{type=%s, version=%s, auroc=%.3f, rollout=%.0f%%, status=%s}",
            modelType, version, auroc, rolloutPercentage, approvalStatus);
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String modelType;
        private String version;
        private String modelPath;
        private long trainingDate = System.currentTimeMillis();
        private String trainingDataset = "unknown";
        private long trainingSampleCount = 0;
        private Map<String, Object> hyperparameters = new HashMap<>();
        private double auroc = 0.0;
        private double precision = 0.0;
        private double recall = 0.0;
        private double f1Score = 0.0;
        private double brierScore = 0.0;
        private Double calibrationSlope = null;
        private Double calibrationIntercept = null;
        private long modelSizeBytes = 0;
        private int inputFeatureCount = 0;
        private int outputDimension = 1;
        private double avgInferenceLatencyMs = 0.0;
        private Double p99InferenceLatencyMs = null;
        private String modelFramework = "ONNX";
        private Integer onnxOpsetVersion = null;
        private Long deploymentDate = null;
        private ApprovalStatus approvalStatus = ApprovalStatus.PENDING;
        private double rolloutPercentage = 0.0;
        private String deploymentEnvironment = "production";
        private String approvedBy = null;
        private Long approvalDate = null;
        private long creationDate = System.currentTimeMillis();
        private long lastUpdated = System.currentTimeMillis();
        private String createdBy = "system";
        private String description = "";
        private Map<String, String> tags = new HashMap<>();

        public Builder modelType(String modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder version(String version) {
            this.version = version;
            return this;
        }

        public Builder modelPath(String modelPath) {
            this.modelPath = modelPath;
            return this;
        }

        public Builder trainingDate(long trainingDate) {
            this.trainingDate = trainingDate;
            return this;
        }

        public Builder trainingDataset(String trainingDataset) {
            this.trainingDataset = trainingDataset;
            return this;
        }

        public Builder trainingSampleCount(long trainingSampleCount) {
            this.trainingSampleCount = trainingSampleCount;
            return this;
        }

        public Builder hyperparameters(Map<String, Object> hyperparameters) {
            this.hyperparameters = new HashMap<>(hyperparameters);
            return this;
        }

        public Builder hyperparameter(String key, Object value) {
            this.hyperparameters.put(key, value);
            return this;
        }

        public Builder auroc(double auroc) {
            this.auroc = auroc;
            return this;
        }

        public Builder precision(double precision) {
            this.precision = precision;
            return this;
        }

        public Builder recall(double recall) {
            this.recall = recall;
            return this;
        }

        public Builder f1Score(double f1Score) {
            this.f1Score = f1Score;
            return this;
        }

        public Builder brierScore(double brierScore) {
            this.brierScore = brierScore;
            return this;
        }

        public Builder calibrationSlope(Double calibrationSlope) {
            this.calibrationSlope = calibrationSlope;
            return this;
        }

        public Builder calibrationIntercept(Double calibrationIntercept) {
            this.calibrationIntercept = calibrationIntercept;
            return this;
        }

        public Builder modelSizeBytes(long modelSizeBytes) {
            this.modelSizeBytes = modelSizeBytes;
            return this;
        }

        public Builder inputFeatureCount(int inputFeatureCount) {
            this.inputFeatureCount = inputFeatureCount;
            return this;
        }

        public Builder outputDimension(int outputDimension) {
            this.outputDimension = outputDimension;
            return this;
        }

        public Builder avgInferenceLatencyMs(double avgInferenceLatencyMs) {
            this.avgInferenceLatencyMs = avgInferenceLatencyMs;
            return this;
        }

        public Builder p99InferenceLatencyMs(Double p99InferenceLatencyMs) {
            this.p99InferenceLatencyMs = p99InferenceLatencyMs;
            return this;
        }

        public Builder modelFramework(String modelFramework) {
            this.modelFramework = modelFramework;
            return this;
        }

        public Builder onnxOpsetVersion(Integer onnxOpsetVersion) {
            this.onnxOpsetVersion = onnxOpsetVersion;
            return this;
        }

        public Builder deploymentDate(Long deploymentDate) {
            this.deploymentDate = deploymentDate;
            return this;
        }

        public Builder approvalStatus(ApprovalStatus approvalStatus) {
            this.approvalStatus = approvalStatus;
            return this;
        }

        public Builder rolloutPercentage(double rolloutPercentage) {
            this.rolloutPercentage = Math.max(0.0, Math.min(100.0, rolloutPercentage));
            return this;
        }

        public Builder deploymentEnvironment(String deploymentEnvironment) {
            this.deploymentEnvironment = deploymentEnvironment;
            return this;
        }

        public Builder approvedBy(String approvedBy) {
            this.approvedBy = approvedBy;
            return this;
        }

        public Builder approvalDate(Long approvalDate) {
            this.approvalDate = approvalDate;
            return this;
        }

        public Builder creationDate(long creationDate) {
            this.creationDate = creationDate;
            return this;
        }

        public Builder lastUpdated(long lastUpdated) {
            this.lastUpdated = lastUpdated;
            return this;
        }

        public Builder createdBy(String createdBy) {
            this.createdBy = createdBy;
            return this;
        }

        public Builder description(String description) {
            this.description = description;
            return this;
        }

        public Builder tags(Map<String, String> tags) {
            this.tags = new HashMap<>(tags);
            return this;
        }

        public Builder tag(String key, String value) {
            this.tags.put(key, value);
            return this;
        }

        /**
         * Initialize from existing metadata (for creating modified copies)
         */
        public Builder from(ModelMetadata metadata) {
            this.modelType = metadata.modelType;
            this.version = metadata.version;
            this.modelPath = metadata.modelPath;
            this.trainingDate = metadata.trainingDate;
            this.trainingDataset = metadata.trainingDataset;
            this.trainingSampleCount = metadata.trainingSampleCount;
            this.hyperparameters = new HashMap<>(metadata.hyperparameters);
            this.auroc = metadata.auroc;
            this.precision = metadata.precision;
            this.recall = metadata.recall;
            this.f1Score = metadata.f1Score;
            this.brierScore = metadata.brierScore;
            this.calibrationSlope = metadata.calibrationSlope;
            this.calibrationIntercept = metadata.calibrationIntercept;
            this.modelSizeBytes = metadata.modelSizeBytes;
            this.inputFeatureCount = metadata.inputFeatureCount;
            this.outputDimension = metadata.outputDimension;
            this.avgInferenceLatencyMs = metadata.avgInferenceLatencyMs;
            this.p99InferenceLatencyMs = metadata.p99InferenceLatencyMs;
            this.modelFramework = metadata.modelFramework;
            this.onnxOpsetVersion = metadata.onnxOpsetVersion;
            this.deploymentDate = metadata.deploymentDate;
            this.approvalStatus = metadata.approvalStatus;
            this.rolloutPercentage = metadata.rolloutPercentage;
            this.deploymentEnvironment = metadata.deploymentEnvironment;
            this.approvedBy = metadata.approvedBy;
            this.approvalDate = metadata.approvalDate;
            this.creationDate = metadata.creationDate;
            this.lastUpdated = metadata.lastUpdated;
            this.createdBy = metadata.createdBy;
            this.description = metadata.description;
            this.tags = new HashMap<>(metadata.tags);
            return this;
        }

        public ModelMetadata build() {
            Objects.requireNonNull(modelType, "modelType is required");
            Objects.requireNonNull(version, "version is required");
            Objects.requireNonNull(modelPath, "modelPath is required");
            return new ModelMetadata(this);
        }
    }

    // ===== Inner Classes =====

    /**
     * Model approval status
     */
    public enum ApprovalStatus {
        PENDING,      // Awaiting approval
        APPROVED,     // Approved for deployment
        REJECTED,     // Rejected, not suitable for deployment
        DEPRECATED    // Previously approved but now deprecated
    }

    /**
     * Performance comparison result
     */
    public static class PerformanceComparison implements Serializable {
        private final String thisVersion;
        private final String otherVersion;
        private final double aurocDelta;
        private final double precisionDelta;
        private final double recallDelta;
        private final double f1Delta;
        private final double latencyDelta;
        private final double overallImprovement;

        public PerformanceComparison(String thisVersion, String otherVersion,
                                    double aurocDelta, double precisionDelta, double recallDelta,
                                    double f1Delta, double latencyDelta, double overallImprovement) {
            this.thisVersion = thisVersion;
            this.otherVersion = otherVersion;
            this.aurocDelta = aurocDelta;
            this.precisionDelta = precisionDelta;
            this.recallDelta = recallDelta;
            this.f1Delta = f1Delta;
            this.latencyDelta = latencyDelta;
            this.overallImprovement = overallImprovement;
        }

        public String getThisVersion() { return thisVersion; }
        public String getOtherVersion() { return otherVersion; }
        public double getAurocDelta() { return aurocDelta; }
        public double getPrecisionDelta() { return precisionDelta; }
        public double getRecallDelta() { return recallDelta; }
        public double getF1Delta() { return f1Delta; }
        public double getLatencyDelta() { return latencyDelta; }
        public double getOverallImprovement() { return overallImprovement; }

        public boolean isImprovement() {
            return overallImprovement > 0.01;  // 1% threshold
        }

        @Override
        public String toString() {
            return String.format("Comparison{%s vs %s: AUROC=%+.3f, F1=%+.3f, Latency=%+.2fms, Overall=%+.3f}",
                thisVersion, otherVersion, aurocDelta, f1Delta, latencyDelta, overallImprovement);
        }
    }
}
