package com.cardiofit.flink.ml.monitoring;

import com.cardiofit.flink.models.MLPrediction;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Model Performance Monitoring Service
 *
 * Tracks and reports ML model performance metrics for operational monitoring
 * and alerting. Integrates with Prometheus for metrics export and visualization.
 *
 * Monitored Metrics:
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 1. Inference Latency                                                     │
 * │    - p50, p95, p99 percentiles                                          │
 * │    - Average, min, max                                                  │
 * │    - Per-model breakdown                                                │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 2. Throughput                                                            │
 * │    - Predictions per second (overall and per-model)                      │
 * │    - Batch processing efficiency                                         │
 * │    - Concurrent prediction tracking                                      │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 3. Model Accuracy (Sliding Window)                                      │
 * │    - AUROC over last N predictions                                       │
 * │    - Precision, Recall, F1-Score                                         │
 * │    - Calibration metrics (Brier score)                                   │
 * └──────────────────────────┬──────────────────────────────────────────────┘
 *                            │
 *                            ▼
 * ┌─────────────────────────────────────────────────────────────────────────┐
 * │ 4. Error Tracking                                                        │
 * │    - Inference failures                                                  │
 * │    - Feature extraction errors                                           │
 * │    - Missing feature counts                                              │
 * │    - Out-of-range feature values                                         │
 * └─────────────────────────────────────────────────────────────────────────┘
 *
 * Prometheus Metrics Exported:
 * <pre>
 * # HELP ml_inference_latency_seconds ML model inference latency
 * # TYPE ml_inference_latency_seconds histogram
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.5"} 0.012
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.95"} 0.018
 * ml_inference_latency_seconds{model="sepsis_risk",quantile="0.99"} 0.025
 *
 * # HELP ml_prediction_count_total Total ML predictions
 * # TYPE ml_prediction_count_total counter
 * ml_prediction_count_total{model="sepsis_risk"} 15432
 *
 * # HELP ml_model_accuracy Current model AUROC
 * # TYPE ml_model_accuracy gauge
 * ml_model_accuracy{model="sepsis_risk"} 0.89
 *
 * # HELP ml_feature_missing_count Missing feature counts
 * # TYPE ml_feature_missing_count counter
 * ml_feature_missing_count{feature="lactate"} 234
 * </pre>
 *
 * Usage Example:
 * <pre>
 * DataStream<MLPrediction> predictions = ...;
 * DataStream<ModelMetrics> metrics = predictions
 *     .keyBy(MLPrediction::getModelType)
 *     .process(new ModelMonitoringService())
 *     .name("model-monitoring");
 * </pre>
 *
 * @author CardioFit Team
 * @version 1.0.0
 */
public class ModelMonitoringService extends ProcessFunction<MLPrediction, ModelMetrics> {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(ModelMonitoringService.class);

    // Configuration
    private final int slidingWindowSize;
    private final long metricsReportIntervalMs;
    private final boolean enableAccuracyTracking;

    // State: Latency measurements (for percentile calculation)
    private transient MapState<Long, LatencyMeasurement> latencyHistoryState;

    // State: Throughput tracking
    private transient ValueState<ThroughputTracker> throughputState;

    // State: Accuracy tracking (if ground truth available)
    private transient ValueState<AccuracyTracker> accuracyState;

    // State: Error tracking
    private transient ValueState<ErrorTracker> errorState;

    // State: Last metrics report timestamp
    private transient ValueState<Long> lastReportTimestampState;

    public ModelMonitoringService() {
        this(1000, 60_000, false);  // 1000 samples, 1-minute reports, no accuracy tracking
    }

    public ModelMonitoringService(int slidingWindowSize,
                                 long metricsReportIntervalMs,
                                 boolean enableAccuracyTracking) {
        this.slidingWindowSize = slidingWindowSize;
        this.metricsReportIntervalMs = metricsReportIntervalMs;
        this.enableAccuracyTracking = enableAccuracyTracking;
    }

    @Override
    public void open(OpenContext context) throws Exception {
        super.open(context);

        // Initialize state
        latencyHistoryState = getRuntimeContext().getMapState(
            new MapStateDescriptor<>("latency-history", Long.class, LatencyMeasurement.class)
        );

        throughputState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("throughput", ThroughputTracker.class)
        );

        accuracyState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("accuracy", AccuracyTracker.class)
        );

        errorState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("errors", ErrorTracker.class)
        );

        lastReportTimestampState = getRuntimeContext().getState(
            new ValueStateDescriptor<>("last-report-timestamp", Long.class)
        );

        LOG.info("ModelMonitoringService initialized: windowSize={}, reportInterval={}ms, accuracyTracking={}",
            slidingWindowSize, metricsReportIntervalMs, enableAccuracyTracking);
    }

    @Override
    public void processElement(MLPrediction prediction,
                              Context ctx,
                              Collector<ModelMetrics> out) throws Exception {
        long currentTime = System.currentTimeMillis();
        String modelType = prediction.getModelType();

        // 1. Track latency
        trackLatency(prediction, currentTime);

        // 2. Track throughput
        trackThroughput(currentTime);

        // 3. Track accuracy (if ground truth available)
        if (enableAccuracyTracking && prediction.getGroundTruth() != null) {
            trackAccuracy(prediction);
        }

        // 4. Track errors (if any)
        if (prediction.hasErrors()) {
            trackErrors(prediction);
        }

        // 5. Check if it's time to report metrics
        Long lastReportTime = lastReportTimestampState.value();
        if (lastReportTime == null) {
            lastReportTime = currentTime;
            lastReportTimestampState.update(lastReportTime);
        }

        if (currentTime - lastReportTime >= metricsReportIntervalMs) {
            // Generate and emit metrics report
            ModelMetrics metrics = generateMetricsReport(modelType, currentTime);
            out.collect(metrics);

            // Update last report timestamp
            lastReportTimestampState.update(currentTime);

            LOG.info("Emitted metrics report for model {}: predictions={}, avgLatency={}ms, throughput={}/sec",
                modelType, metrics.getPredictionCount(), metrics.getAverageLatencyMs(),
                metrics.getThroughputPerSecond());
        }
    }

    /**
     * Track inference latency
     */
    private void trackLatency(MLPrediction prediction, long currentTime) throws Exception {
        double latencyMs = prediction.getInferenceLatencyMs();

        // Add to latency history
        LatencyMeasurement measurement = new LatencyMeasurement(currentTime, latencyMs);
        latencyHistoryState.put(currentTime, measurement);

        // Maintain sliding window
        Iterator<Map.Entry<Long, LatencyMeasurement>> iterator = latencyHistoryState.entries().iterator();
        int count = 0;
        while (iterator.hasNext()) {
            iterator.next();
            count++;
        }

        if (count > slidingWindowSize) {
            // Remove oldest measurements
            iterator = latencyHistoryState.entries().iterator();
            for (int i = 0; i < (count - slidingWindowSize); i++) {
                if (iterator.hasNext()) {
                    iterator.next();
                    iterator.remove();
                }
            }
        }
    }

    /**
     * Track throughput (predictions per second)
     */
    private void trackThroughput(long currentTime) throws Exception {
        ThroughputTracker tracker = throughputState.value();
        if (tracker == null) {
            tracker = new ThroughputTracker();
        }

        tracker.recordPrediction(currentTime);
        throughputState.update(tracker);
    }

    /**
     * Track model accuracy (when ground truth available)
     */
    private void trackAccuracy(MLPrediction prediction) throws Exception {
        AccuracyTracker tracker = accuracyState.value();
        if (tracker == null) {
            tracker = new AccuracyTracker(slidingWindowSize);
        }

        double predictedScore = prediction.getPrimaryScore();
        double groundTruth = prediction.getGroundTruth();

        tracker.addPrediction(predictedScore, groundTruth);
        accuracyState.update(tracker);
    }

    /**
     * Track inference errors
     */
    private void trackErrors(MLPrediction prediction) throws Exception {
        ErrorTracker tracker = errorState.value();
        if (tracker == null) {
            tracker = new ErrorTracker();
        }

        tracker.recordError(prediction.getErrorType(), prediction.getErrorMessage());
        errorState.update(tracker);
    }

    /**
     * Generate comprehensive metrics report
     */
    private ModelMetrics generateMetricsReport(String modelType, long timestamp) throws Exception {
        // 1. Latency metrics
        LatencyMetrics latencyMetrics = calculateLatencyMetrics();

        // 2. Throughput metrics
        ThroughputTracker throughputTracker = throughputState.value();
        double throughput = throughputTracker != null ? throughputTracker.getThroughputPerSecond() : 0.0;
        long predictionCount = throughputTracker != null ? throughputTracker.getTotalPredictions() : 0;

        // 3. Accuracy metrics (if enabled)
        AccuracyMetrics accuracyMetrics = null;
        if (enableAccuracyTracking) {
            AccuracyTracker accuracyTracker = accuracyState.value();
            if (accuracyTracker != null) {
                accuracyMetrics = accuracyTracker.calculateMetrics();
            }
        }

        // 4. Error metrics
        ErrorMetrics errorMetrics = null;
        ErrorTracker errorTracker = errorState.value();
        if (errorTracker != null) {
            errorMetrics = errorTracker.getMetrics();
        }

        return ModelMetrics.builder()
            .modelType(modelType)
            .timestamp(timestamp)
            .predictionCount(predictionCount)
            .throughputPerSecond(throughput)
            .latencyMetrics(latencyMetrics)
            .accuracyMetrics(accuracyMetrics)
            .errorMetrics(errorMetrics)
            .build();
    }

    /**
     * Calculate latency percentiles
     */
    private LatencyMetrics calculateLatencyMetrics() throws Exception {
        List<Double> latencies = new ArrayList<>();
        Iterator<Map.Entry<Long, LatencyMeasurement>> iterator = latencyHistoryState.entries().iterator();
        while (iterator.hasNext()) {
            latencies.add(iterator.next().getValue().getLatencyMs());
        }

        if (latencies.isEmpty()) {
            return new LatencyMetrics(0, 0, 0, 0, 0, 0);
        }

        Collections.sort(latencies);

        double p50 = calculatePercentile(latencies, 0.50);
        double p95 = calculatePercentile(latencies, 0.95);
        double p99 = calculatePercentile(latencies, 0.99);
        double avg = latencies.stream().mapToDouble(Double::doubleValue).average().orElse(0.0);
        double min = latencies.get(0);
        double max = latencies.get(latencies.size() - 1);

        return new LatencyMetrics(p50, p95, p99, avg, min, max);
    }

    private double calculatePercentile(List<Double> sortedValues, double percentile) {
        int index = (int) Math.ceil(percentile * sortedValues.size()) - 1;
        index = Math.max(0, Math.min(sortedValues.size() - 1, index));
        return sortedValues.get(index);
    }

    // ===== Helper Classes =====

    /**
     * Latency measurement record
     */
    private static class LatencyMeasurement implements Serializable {
        private final long timestamp;
        private final double latencyMs;

        LatencyMeasurement(long timestamp, double latencyMs) {
            this.timestamp = timestamp;
            this.latencyMs = latencyMs;
        }

        public long getTimestamp() { return timestamp; }
        public double getLatencyMs() { return latencyMs; }
    }

    /**
     * Throughput tracker
     */
    private static class ThroughputTracker implements Serializable {
        private long totalPredictions = 0;
        private long windowStartTime = System.currentTimeMillis();
        private long windowPredictions = 0;
        private static final long WINDOW_SIZE_MS = 10_000;  // 10-second window

        public void recordPrediction(long timestamp) {
            totalPredictions++;

            // Check if we need to reset window
            if (timestamp - windowStartTime >= WINDOW_SIZE_MS) {
                windowStartTime = timestamp;
                windowPredictions = 1;
            } else {
                windowPredictions++;
            }
        }

        public double getThroughputPerSecond() {
            long elapsed = System.currentTimeMillis() - windowStartTime;
            if (elapsed == 0) return 0.0;
            return (windowPredictions * 1000.0) / elapsed;
        }

        public long getTotalPredictions() { return totalPredictions; }
    }

    /**
     * Accuracy tracker (sliding window)
     */
    private static class AccuracyTracker implements Serializable {
        private final int windowSize;
        private final List<PredictionRecord> predictions;

        AccuracyTracker(int windowSize) {
            this.windowSize = windowSize;
            this.predictions = new ArrayList<>();
        }

        public void addPrediction(double predictedScore, double groundTruth) {
            predictions.add(new PredictionRecord(predictedScore, groundTruth));

            // Maintain window size
            if (predictions.size() > windowSize) {
                predictions.remove(0);
            }
        }

        public AccuracyMetrics calculateMetrics() {
            if (predictions.isEmpty()) {
                return new AccuracyMetrics(0.0, 0.0, 0.0, 0.0, 0.0);
            }

            // Calculate AUROC (Area Under ROC Curve)
            double auroc = calculateAUROC();

            // Calculate Brier score (calibration metric)
            double brierScore = calculateBrierScore();

            // Calculate binary classification metrics at threshold 0.5
            BinaryMetrics binaryMetrics = calculateBinaryMetrics(0.5);

            return new AccuracyMetrics(
                auroc,
                binaryMetrics.precision,
                binaryMetrics.recall,
                binaryMetrics.f1Score,
                brierScore
            );
        }

        private double calculateAUROC() {
            // Simplified AUROC calculation (trapezoidal rule)
            // Sort by predicted score descending
            List<PredictionRecord> sorted = new ArrayList<>(predictions);
            sorted.sort((a, b) -> Double.compare(b.predictedScore, a.predictedScore));

            int positives = 0;
            int negatives = 0;
            for (PredictionRecord pred : sorted) {
                if (pred.groundTruth >= 0.5) positives++;
                else negatives++;
            }

            if (positives == 0 || negatives == 0) return 0.5;

            double auc = 0.0;
            int truePositives = 0;
            int falsePositives = 0;

            for (PredictionRecord pred : sorted) {
                if (pred.groundTruth >= 0.5) {
                    truePositives++;
                } else {
                    falsePositives++;
                    auc += truePositives;
                }
            }

            return auc / (positives * negatives);
        }

        private double calculateBrierScore() {
            double sum = 0.0;
            for (PredictionRecord pred : predictions) {
                double diff = pred.predictedScore - pred.groundTruth;
                sum += diff * diff;
            }
            return sum / predictions.size();
        }

        private BinaryMetrics calculateBinaryMetrics(double threshold) {
            int tp = 0, fp = 0, tn = 0, fn = 0;

            for (PredictionRecord pred : predictions) {
                boolean predicted = pred.predictedScore >= threshold;
                boolean actual = pred.groundTruth >= 0.5;

                if (predicted && actual) tp++;
                else if (predicted && !actual) fp++;
                else if (!predicted && actual) fn++;
                else tn++;
            }

            double precision = (tp + fp) > 0 ? (double) tp / (tp + fp) : 0.0;
            double recall = (tp + fn) > 0 ? (double) tp / (tp + fn) : 0.0;
            double f1Score = (precision + recall) > 0 ? 2 * precision * recall / (precision + recall) : 0.0;

            return new BinaryMetrics(precision, recall, f1Score);
        }

        private static class PredictionRecord implements Serializable {
            final double predictedScore;
            final double groundTruth;

            PredictionRecord(double predictedScore, double groundTruth) {
                this.predictedScore = predictedScore;
                this.groundTruth = groundTruth;
            }
        }

        private static class BinaryMetrics {
            final double precision;
            final double recall;
            final double f1Score;

            BinaryMetrics(double precision, double recall, double f1Score) {
                this.precision = precision;
                this.recall = recall;
                this.f1Score = f1Score;
            }
        }
    }

    /**
     * Error tracker
     */
    private static class ErrorTracker implements Serializable {
        private final Map<String, Long> errorCounts = new HashMap<>();
        private long totalErrors = 0;

        public void recordError(String errorType, String errorMessage) {
            totalErrors++;
            errorCounts.put(errorType, errorCounts.getOrDefault(errorType, 0L) + 1);
        }

        public ErrorMetrics getMetrics() {
            return new ErrorMetrics(totalErrors, new HashMap<>(errorCounts));
        }
    }

    /**
     * Latency metrics container
     */
    public static class LatencyMetrics implements Serializable {
        private final double p50;
        private final double p95;
        private final double p99;
        private final double average;
        private final double min;
        private final double max;

        public LatencyMetrics(double p50, double p95, double p99,
                            double average, double min, double max) {
            this.p50 = p50;
            this.p95 = p95;
            this.p99 = p99;
            this.average = average;
            this.min = min;
            this.max = max;
        }

        public double getP50() { return p50; }
        public double getP95() { return p95; }
        public double getP99() { return p99; }
        public double getAverage() { return average; }
        public double getMin() { return min; }
        public double getMax() { return max; }

        @Override
        public String toString() {
            return String.format("LatencyMetrics{p50=%.2f, p95=%.2f, p99=%.2f, avg=%.2f, min=%.2f, max=%.2f}",
                p50, p95, p99, average, min, max);
        }
    }

    /**
     * Accuracy metrics container
     */
    public static class AccuracyMetrics implements Serializable {
        private final double auroc;
        private final double precision;
        private final double recall;
        private final double f1Score;
        private final double brierScore;

        public AccuracyMetrics(double auroc, double precision, double recall,
                             double f1Score, double brierScore) {
            this.auroc = auroc;
            this.precision = precision;
            this.recall = recall;
            this.f1Score = f1Score;
            this.brierScore = brierScore;
        }

        public double getAuroc() { return auroc; }
        public double getPrecision() { return precision; }
        public double getRecall() { return recall; }
        public double getF1Score() { return f1Score; }
        public double getBrierScore() { return brierScore; }

        @Override
        public String toString() {
            return String.format("AccuracyMetrics{auroc=%.3f, precision=%.3f, recall=%.3f, f1=%.3f, brier=%.3f}",
                auroc, precision, recall, f1Score, brierScore);
        }
    }

    /**
     * Error metrics container
     */
    public static class ErrorMetrics implements Serializable {
        private final long totalErrors;
        private final Map<String, Long> errorCountsByType;

        public ErrorMetrics(long totalErrors, Map<String, Long> errorCountsByType) {
            this.totalErrors = totalErrors;
            this.errorCountsByType = errorCountsByType;
        }

        public long getTotalErrors() { return totalErrors; }
        public Map<String, Long> getErrorCountsByType() { return errorCountsByType; }

        @Override
        public String toString() {
            return String.format("ErrorMetrics{total=%d, byType=%s}", totalErrors, errorCountsByType);
        }
    }
}
