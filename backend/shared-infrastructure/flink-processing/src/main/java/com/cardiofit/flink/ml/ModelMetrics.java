package com.cardiofit.flink.ml;

import java.io.Serializable;
import java.time.Instant;

/**
 * Model performance metrics container
 *
 * Tracks runtime performance metrics for ML models including:
 * - Inference count and latency
 * - Throughput and resource utilization
 * - Temporal tracking
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelMetrics implements Serializable {
    private static final long serialVersionUID = 1L;

    private String modelId;
    private String modelName;
    private String modelType;

    // Inference statistics
    private long inferenceCount;
    private long totalInferenceTimeMs;
    private double averageInferenceTimeMs;
    private double throughputPerSecond;

    // Temporal tracking
    private long lastInferenceTimestamp;
    private long metricsCollectionTimestamp;

    // Performance percentiles (if tracked)
    private Double p50LatencyMs;
    private Double p95LatencyMs;
    private Double p99LatencyMs;

    // Quality metrics
    private Double recentAccuracy;
    private Double recentPrecision;
    private Double recentRecall;

    /**
     * Private constructor - use Builder pattern
     */
    private ModelMetrics(Builder builder) {
        this.modelId = builder.modelId;
        this.modelName = builder.modelName;
        this.modelType = builder.modelType;
        this.inferenceCount = builder.inferenceCount;
        this.totalInferenceTimeMs = builder.totalInferenceTimeMs;
        this.averageInferenceTimeMs = builder.averageInferenceTimeMs;
        this.throughputPerSecond = builder.throughputPerSecond;
        this.lastInferenceTimestamp = builder.lastInferenceTimestamp;
        this.metricsCollectionTimestamp = builder.metricsCollectionTimestamp;
        this.p50LatencyMs = builder.p50LatencyMs;
        this.p95LatencyMs = builder.p95LatencyMs;
        this.p99LatencyMs = builder.p99LatencyMs;
        this.recentAccuracy = builder.recentAccuracy;
        this.recentPrecision = builder.recentPrecision;
        this.recentRecall = builder.recentRecall;
    }

    // Getters
    public String getModelId() { return modelId; }
    public String getModelName() { return modelName; }
    public String getModelType() { return modelType; }
    public long getInferenceCount() { return inferenceCount; }
    public long getTotalInferenceTimeMs() { return totalInferenceTimeMs; }
    public double getAverageInferenceTimeMs() { return averageInferenceTimeMs; }
    public double getThroughputPerSecond() { return throughputPerSecond; }
    public long getLastInferenceTimestamp() { return lastInferenceTimestamp; }
    public long getMetricsCollectionTimestamp() { return metricsCollectionTimestamp; }
    public Double getP50LatencyMs() { return p50LatencyMs; }
    public Double getP95LatencyMs() { return p95LatencyMs; }
    public Double getP99LatencyMs() { return p99LatencyMs; }
    public Double getRecentAccuracy() { return recentAccuracy; }
    public Double getRecentPrecision() { return recentPrecision; }
    public Double getRecentRecall() { return recentRecall; }

    /**
     * Check if model is meeting performance SLA
     *
     * @param maxLatencyMs Maximum acceptable latency (p99)
     * @param minThroughput Minimum required throughput
     * @return true if SLA is met
     */
    public boolean isMeetingSLA(double maxLatencyMs, double minThroughput) {
        boolean latencyOk = p99LatencyMs == null || p99LatencyMs <= maxLatencyMs;
        boolean throughputOk = throughputPerSecond >= minThroughput;
        return latencyOk && throughputOk;
    }

    /**
     * Get human-readable metrics summary
     */
    public String getSummary() {
        return String.format(
            "Model: %s | Inferences: %d | Avg Latency: %.2fms | Throughput: %.1f/sec | Last Run: %s",
            modelName,
            inferenceCount,
            averageInferenceTimeMs,
            throughputPerSecond,
            lastInferenceTimestamp > 0 ?
                Instant.ofEpochMilli(lastInferenceTimestamp).toString() : "Never"
        );
    }

    @Override
    public String toString() {
        return "ModelMetrics{" +
            "modelId='" + modelId + '\'' +
            ", modelName='" + modelName + '\'' +
            ", inferenceCount=" + inferenceCount +
            ", averageInferenceTimeMs=" + String.format("%.2f", averageInferenceTimeMs) +
            ", throughputPerSecond=" + String.format("%.1f", throughputPerSecond) +
            (p99LatencyMs != null ? ", p99LatencyMs=" + String.format("%.2f", p99LatencyMs) : "") +
            (recentAccuracy != null ? ", recentAccuracy=" + String.format("%.4f", recentAccuracy) : "") +
            '}';
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String modelId;
        private String modelName;
        private String modelType;
        private long inferenceCount;
        private long totalInferenceTimeMs;
        private double averageInferenceTimeMs;
        private double throughputPerSecond;
        private long lastInferenceTimestamp;
        private long metricsCollectionTimestamp = System.currentTimeMillis();
        private Double p50LatencyMs;
        private Double p95LatencyMs;
        private Double p99LatencyMs;
        private Double recentAccuracy;
        private Double recentPrecision;
        private Double recentRecall;

        public Builder modelId(String modelId) {
            this.modelId = modelId;
            return this;
        }

        public Builder modelName(String modelName) {
            this.modelName = modelName;
            return this;
        }

        public Builder modelType(String modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder inferenceCount(long inferenceCount) {
            this.inferenceCount = inferenceCount;
            return this;
        }

        public Builder totalInferenceTimeMs(long totalInferenceTimeMs) {
            this.totalInferenceTimeMs = totalInferenceTimeMs;
            return this;
        }

        public Builder averageInferenceTimeMs(double averageInferenceTimeMs) {
            this.averageInferenceTimeMs = averageInferenceTimeMs;
            return this;
        }

        public Builder throughputPerSecond(double throughputPerSecond) {
            this.throughputPerSecond = throughputPerSecond;
            return this;
        }

        public Builder lastInferenceTimestamp(long lastInferenceTimestamp) {
            this.lastInferenceTimestamp = lastInferenceTimestamp;
            return this;
        }

        public Builder metricsCollectionTimestamp(long metricsCollectionTimestamp) {
            this.metricsCollectionTimestamp = metricsCollectionTimestamp;
            return this;
        }

        public Builder p50LatencyMs(Double p50LatencyMs) {
            this.p50LatencyMs = p50LatencyMs;
            return this;
        }

        public Builder p95LatencyMs(Double p95LatencyMs) {
            this.p95LatencyMs = p95LatencyMs;
            return this;
        }

        public Builder p99LatencyMs(Double p99LatencyMs) {
            this.p99LatencyMs = p99LatencyMs;
            return this;
        }

        public Builder recentAccuracy(Double recentAccuracy) {
            this.recentAccuracy = recentAccuracy;
            return this;
        }

        public Builder recentPrecision(Double recentPrecision) {
            this.recentPrecision = recentPrecision;
            return this;
        }

        public Builder recentRecall(Double recentRecall) {
            this.recentRecall = recentRecall;
            return this;
        }

        public ModelMetrics build() {
            return new ModelMetrics(this);
        }
    }
}
