package com.cardiofit.flink.ml.monitoring;

import com.cardiofit.flink.ml.monitoring.ModelMonitoringService.AccuracyMetrics;
import com.cardiofit.flink.ml.monitoring.ModelMonitoringService.ErrorMetrics;
import com.cardiofit.flink.ml.monitoring.ModelMonitoringService.LatencyMetrics;

import java.io.Serializable;
import java.util.HashMap;
import java.util.Map;

/**
 * Model Metrics Output
 *
 * Aggregated metrics for ML model performance monitoring. This class represents
 * a time-windowed snapshot of model performance across multiple dimensions.
 *
 * Metrics Categories:
 * - Latency: p50, p95, p99, average, min, max
 * - Throughput: predictions/second, total predictions
 * - Accuracy: AUROC, precision, recall, F1, Brier score (if ground truth available)
 * - Errors: total errors, breakdown by error type
 *
 * Prometheus Export Format:
 * <pre>
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.5"} 0.012
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.95"} 0.018
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.99"} 0.025
 * ml_prediction_count_total{model="sepsis_risk"} 15432
 * ml_throughput_per_second{model="sepsis_risk"} 45.2
 * ml_model_accuracy{model="sepsis_risk",metric="auroc"} 0.89
 * ml_errors_total{model="sepsis_risk"} 12
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelMetrics implements Serializable {
    private static final long serialVersionUID = 1L;

    // Identification
    private String modelType;
    private long timestamp;

    // Throughput metrics
    private long predictionCount;
    private double throughputPerSecond;

    // Latency metrics
    private LatencyMetrics latencyMetrics;

    // Accuracy metrics (optional - only if ground truth available)
    private AccuracyMetrics accuracyMetrics;

    // Error metrics
    private ErrorMetrics errorMetrics;

    /**
     * Private constructor - use Builder
     */
    private ModelMetrics(Builder builder) {
        this.modelType = builder.modelType;
        this.timestamp = builder.timestamp;
        this.predictionCount = builder.predictionCount;
        this.throughputPerSecond = builder.throughputPerSecond;
        this.latencyMetrics = builder.latencyMetrics;
        this.accuracyMetrics = builder.accuracyMetrics;
        this.errorMetrics = builder.errorMetrics;
    }

    // ===== Getters =====

    public String getModelType() {
        return modelType;
    }

    public long getTimestamp() {
        return timestamp;
    }

    public long getPredictionCount() {
        return predictionCount;
    }

    public double getThroughputPerSecond() {
        return throughputPerSecond;
    }

    public LatencyMetrics getLatencyMetrics() {
        return latencyMetrics;
    }

    public double getAverageLatencyMs() {
        return latencyMetrics != null ? latencyMetrics.getAverage() : 0.0;
    }

    public double getP95LatencyMs() {
        return latencyMetrics != null ? latencyMetrics.getP95() : 0.0;
    }

    public double getP99LatencyMs() {
        return latencyMetrics != null ? latencyMetrics.getP99() : 0.0;
    }

    public AccuracyMetrics getAccuracyMetrics() {
        return accuracyMetrics;
    }

    public boolean hasAccuracyMetrics() {
        return accuracyMetrics != null;
    }

    public ErrorMetrics getErrorMetrics() {
        return errorMetrics;
    }

    public long getTotalErrors() {
        return errorMetrics != null ? errorMetrics.getTotalErrors() : 0;
    }

    /**
     * Export metrics in Prometheus format
     */
    public String toPrometheusFormat() {
        StringBuilder sb = new StringBuilder();

        // Latency metrics
        if (latencyMetrics != null) {
            sb.append(String.format("ml_inference_latency_seconds{model=\"%s\",quantile=\"0.5\"} %.6f\n",
                modelType, latencyMetrics.getP50() / 1000.0));
            sb.append(String.format("ml_inference_latency_seconds{model=\"%s\",quantile=\"0.95\"} %.6f\n",
                modelType, latencyMetrics.getP95() / 1000.0));
            sb.append(String.format("ml_inference_latency_seconds{model=\"%s\",quantile=\"0.99\"} %.6f\n",
                modelType, latencyMetrics.getP99() / 1000.0));
            sb.append(String.format("ml_inference_latency_seconds_avg{model=\"%s\"} %.6f\n",
                modelType, latencyMetrics.getAverage() / 1000.0));
        }

        // Throughput metrics
        sb.append(String.format("ml_prediction_count_total{model=\"%s\"} %d\n",
            modelType, predictionCount));
        sb.append(String.format("ml_throughput_per_second{model=\"%s\"} %.2f\n",
            modelType, throughputPerSecond));

        // Accuracy metrics (if available)
        if (accuracyMetrics != null) {
            sb.append(String.format("ml_model_accuracy{model=\"%s\",metric=\"auroc\"} %.4f\n",
                modelType, accuracyMetrics.getAuroc()));
            sb.append(String.format("ml_model_accuracy{model=\"%s\",metric=\"precision\"} %.4f\n",
                modelType, accuracyMetrics.getPrecision()));
            sb.append(String.format("ml_model_accuracy{model=\"%s\",metric=\"recall\"} %.4f\n",
                modelType, accuracyMetrics.getRecall()));
            sb.append(String.format("ml_model_accuracy{model=\"%s\",metric=\"f1_score\"} %.4f\n",
                modelType, accuracyMetrics.getF1Score()));
            sb.append(String.format("ml_model_calibration{model=\"%s\",metric=\"brier_score\"} %.4f\n",
                modelType, accuracyMetrics.getBrierScore()));
        }

        // Error metrics
        if (errorMetrics != null) {
            sb.append(String.format("ml_errors_total{model=\"%s\"} %d\n",
                modelType, errorMetrics.getTotalErrors()));

            for (Map.Entry<String, Long> entry : errorMetrics.getErrorCountsByType().entrySet()) {
                sb.append(String.format("ml_errors_by_type{model=\"%s\",error_type=\"%s\"} %d\n",
                    modelType, entry.getKey(), entry.getValue()));
            }
        }

        return sb.toString();
    }

    /**
     * Export metrics as JSON
     */
    public String toJson() {
        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append(String.format("\"model_type\":\"%s\",", modelType));
        sb.append(String.format("\"timestamp\":%d,", timestamp));
        sb.append(String.format("\"prediction_count\":%d,", predictionCount));
        sb.append(String.format("\"throughput_per_second\":%.2f,", throughputPerSecond));

        // Latency metrics
        if (latencyMetrics != null) {
            sb.append("\"latency\":{");
            sb.append(String.format("\"p50\":%.2f,", latencyMetrics.getP50()));
            sb.append(String.format("\"p95\":%.2f,", latencyMetrics.getP95()));
            sb.append(String.format("\"p99\":%.2f,", latencyMetrics.getP99()));
            sb.append(String.format("\"average\":%.2f,", latencyMetrics.getAverage()));
            sb.append(String.format("\"min\":%.2f,", latencyMetrics.getMin()));
            sb.append(String.format("\"max\":%.2f", latencyMetrics.getMax()));
            sb.append("},");
        }

        // Accuracy metrics
        if (accuracyMetrics != null) {
            sb.append("\"accuracy\":{");
            sb.append(String.format("\"auroc\":%.4f,", accuracyMetrics.getAuroc()));
            sb.append(String.format("\"precision\":%.4f,", accuracyMetrics.getPrecision()));
            sb.append(String.format("\"recall\":%.4f,", accuracyMetrics.getRecall()));
            sb.append(String.format("\"f1_score\":%.4f,", accuracyMetrics.getF1Score()));
            sb.append(String.format("\"brier_score\":%.4f", accuracyMetrics.getBrierScore()));
            sb.append("},");
        }

        // Error metrics
        if (errorMetrics != null) {
            sb.append("\"errors\":{");
            sb.append(String.format("\"total\":%d,", errorMetrics.getTotalErrors()));
            sb.append("\"by_type\":{");
            int i = 0;
            for (Map.Entry<String, Long> entry : errorMetrics.getErrorCountsByType().entrySet()) {
                if (i > 0) sb.append(",");
                sb.append(String.format("\"%s\":%d", entry.getKey(), entry.getValue()));
                i++;
            }
            sb.append("}}");
        } else {
            // Remove trailing comma if no error metrics
            sb.setLength(sb.length() - 1);
        }

        sb.append("}");
        return sb.toString();
    }

    /**
     * Generate human-readable summary report
     */
    public String toSummaryReport() {
        StringBuilder report = new StringBuilder();

        report.append("═══════════════════════════════════════════════════════════════\n");
        report.append(String.format("  MODEL METRICS REPORT - %s\n", modelType.toUpperCase()));
        report.append("═══════════════════════════════════════════════════════════════\n\n");

        report.append(String.format("Report Timestamp: %d\n", timestamp));
        report.append(String.format("Total Predictions: %,d\n", predictionCount));
        report.append(String.format("Throughput: %.2f predictions/sec\n\n", throughputPerSecond));

        // Latency section
        if (latencyMetrics != null) {
            report.append("Inference Latency:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append(String.format("  p50 (median):  %.2f ms\n", latencyMetrics.getP50()));
            report.append(String.format("  p95:           %.2f ms\n", latencyMetrics.getP95()));
            report.append(String.format("  p99:           %.2f ms\n", latencyMetrics.getP99()));
            report.append(String.format("  Average:       %.2f ms\n", latencyMetrics.getAverage()));
            report.append(String.format("  Min:           %.2f ms\n", latencyMetrics.getMin()));
            report.append(String.format("  Max:           %.2f ms\n\n", latencyMetrics.getMax()));
        }

        // Accuracy section (if available)
        if (accuracyMetrics != null) {
            report.append("Model Accuracy (Sliding Window):\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append(String.format("  AUROC:         %.4f %s\n",
                accuracyMetrics.getAuroc(), getAUROCRating(accuracyMetrics.getAuroc())));
            report.append(String.format("  Precision:     %.4f\n", accuracyMetrics.getPrecision()));
            report.append(String.format("  Recall:        %.4f\n", accuracyMetrics.getRecall()));
            report.append(String.format("  F1-Score:      %.4f\n", accuracyMetrics.getF1Score()));
            report.append(String.format("  Brier Score:   %.4f %s\n\n",
                accuracyMetrics.getBrierScore(), getBrierRating(accuracyMetrics.getBrierScore())));
        }

        // Error section
        if (errorMetrics != null && errorMetrics.getTotalErrors() > 0) {
            report.append("Errors:\n");
            report.append("───────────────────────────────────────────────────────────────\n");
            report.append(String.format("  Total Errors:  %d\n", errorMetrics.getTotalErrors()));

            if (!errorMetrics.getErrorCountsByType().isEmpty()) {
                report.append("  By Type:\n");
                for (Map.Entry<String, Long> entry : errorMetrics.getErrorCountsByType().entrySet()) {
                    report.append(String.format("    - %s: %d\n", entry.getKey(), entry.getValue()));
                }
            }
            report.append("\n");
        }

        report.append("═══════════════════════════════════════════════════════════════\n");

        return report.toString();
    }

    private String getAUROCRating(double auroc) {
        if (auroc >= 0.90) return "[EXCELLENT]";
        if (auroc >= 0.80) return "[GOOD]";
        if (auroc >= 0.70) return "[ACCEPTABLE]";
        return "[NEEDS IMPROVEMENT]";
    }

    private String getBrierRating(double brier) {
        if (brier <= 0.10) return "[EXCELLENT]";
        if (brier <= 0.20) return "[GOOD]";
        if (brier <= 0.30) return "[ACCEPTABLE]";
        return "[NEEDS IMPROVEMENT]";
    }

    @Override
    public String toString() {
        return String.format(
            "ModelMetrics{model=%s, predictions=%d, throughput=%.2f/sec, avgLatency=%.2fms, p95=%.2fms, errors=%d}",
            modelType, predictionCount, throughputPerSecond,
            latencyMetrics != null ? latencyMetrics.getAverage() : 0.0,
            latencyMetrics != null ? latencyMetrics.getP95() : 0.0,
            errorMetrics != null ? errorMetrics.getTotalErrors() : 0
        );
    }

    // ===== Builder Pattern =====

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private String modelType;
        private long timestamp = System.currentTimeMillis();
        private long predictionCount;
        private double throughputPerSecond;
        private LatencyMetrics latencyMetrics;
        private AccuracyMetrics accuracyMetrics;
        private ErrorMetrics errorMetrics;

        public Builder modelType(String modelType) {
            this.modelType = modelType;
            return this;
        }

        public Builder timestamp(long timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public Builder predictionCount(long predictionCount) {
            this.predictionCount = predictionCount;
            return this;
        }

        public Builder throughputPerSecond(double throughputPerSecond) {
            this.throughputPerSecond = throughputPerSecond;
            return this;
        }

        public Builder latencyMetrics(LatencyMetrics latencyMetrics) {
            this.latencyMetrics = latencyMetrics;
            return this;
        }

        public Builder accuracyMetrics(AccuracyMetrics accuracyMetrics) {
            this.accuracyMetrics = accuracyMetrics;
            return this;
        }

        public Builder errorMetrics(ErrorMetrics errorMetrics) {
            this.errorMetrics = errorMetrics;
            return this;
        }

        public ModelMetrics build() {
            if (modelType == null || modelType.isEmpty()) {
                throw new IllegalArgumentException("modelType is required");
            }

            return new ModelMetrics(this);
        }
    }
}
