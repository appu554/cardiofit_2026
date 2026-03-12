
package com.cardiofit.flink.monitoring;

import org.apache.flink.api.common.functions.RichMapFunction;
import org.apache.flink.api.common.functions.OpenContext;
import org.apache.flink.dropwizard.metrics.DropwizardMeterWrapper;
import org.apache.flink.dropwizard.metrics.DropwizardHistogramWrapper;
import org.apache.flink.metrics.*;
import org.apache.flink.streaming.api.functions.ProcessFunction;
// DISABLED FOR FLINK 2.X: import org.apache.flink.streaming.api.functions.sink.SinkFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;

import com.codahale.metrics.SlidingWindowReservoir;
import com.cardiofit.flink.models.*;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * Comprehensive healthcare-specific metrics collection and monitoring
 * Provides clinical, operational, and performance metrics for the EHR Intelligence Engine
 */
public class HealthcareMetrics {

    // ========== Business/Clinical Metrics ==========

    /**
     * Clinical outcome metrics for healthcare quality monitoring
     */
    public static class ClinicalMetricsCollector extends RichMapFunction<PatternEvent, PatternEvent> {

        // Clinical quality metrics
        private transient Counter alertsGenerated;
        private transient Counter criticalAlertsGenerated;
        private transient Counter falsePositiveAlerts;
        private transient Counter truePositiveAlerts;

        // Patient safety metrics
        private transient Histogram patientRiskScoreDistribution;
        private transient Meter sepsisDetectionRate;
        private transient Meter cardiacEventDetectionRate;
        private transient Meter medicationErrorPrevention;

        // Clinical pathway metrics
        private transient Counter pathwayAdherenceCount;
        private transient Histogram pathwayCompletionTime;
        private transient Gauge<Double> overallPathwayAdherenceRate;

        // Alert quality metrics
        private transient Histogram alertResponseTime;
        private transient Counter alertEscalations;
        private transient Meter alertFatigueRate;

        @Override
        public void open(OpenContext openContext) throws Exception {
            super.open(openContext);

            MetricGroup clinicalMetrics = getRuntimeContext()
                .getMetricGroup()
                .addGroup("clinical");

            // Initialize clinical quality metrics
            alertsGenerated = clinicalMetrics.counter("alerts_generated_total");
            criticalAlertsGenerated = clinicalMetrics.counter("critical_alerts_total");
            falsePositiveAlerts = clinicalMetrics.counter("false_positive_alerts_total");
            truePositiveAlerts = clinicalMetrics.counter("true_positive_alerts_total");

            // Initialize patient safety metrics
            patientRiskScoreDistribution = clinicalMetrics.histogram("patient_risk_score_distribution",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            sepsisDetectionRate = clinicalMetrics.meter("sepsis_detection_rate",
                new DropwizardMeterWrapper(new com.codahale.metrics.Meter()));

            cardiacEventDetectionRate = clinicalMetrics.meter("cardiac_event_detection_rate",
                new DropwizardMeterWrapper(new com.codahale.metrics.Meter()));

            medicationErrorPrevention = clinicalMetrics.meter("medication_error_prevention_rate",
                new DropwizardMeterWrapper(new com.codahale.metrics.Meter()));

            // Initialize clinical pathway metrics
            pathwayAdherenceCount = clinicalMetrics.counter("pathway_adherence_count");
            pathwayCompletionTime = clinicalMetrics.histogram("pathway_completion_time_hours",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(500))));

            // Initialize alert quality metrics
            alertResponseTime = clinicalMetrics.histogram("alert_response_time_minutes",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            alertEscalations = clinicalMetrics.counter("alert_escalations_total");
            alertFatigueRate = clinicalMetrics.meter("alert_fatigue_rate",
                new DropwizardMeterWrapper(new com.codahale.metrics.Meter()));

            // Register dynamic gauge for pathway adherence rate
            overallPathwayAdherenceRate = clinicalMetrics.gauge("pathway_adherence_rate_percent",
                new Gauge<Double>() {
                    @Override
                    public Double getValue() {
                        return calculatePathwayAdherenceRate();
                    }
                });
        }

        @Override
        public PatternEvent map(PatternEvent pattern) throws Exception {
            // Update clinical metrics based on pattern type and outcomes
            updateClinicalMetrics(pattern);
            return pattern;
        }

        private void updateClinicalMetrics(PatternEvent pattern) {
            alertsGenerated.inc();

            // Track by severity (getSeverity() returns String)
            if ("CRITICAL".equals(pattern.getSeverity())) {
                criticalAlertsGenerated.inc();
            }

            // Track by pattern type for specific clinical outcomes
            switch (pattern.getPatternType()) {
                case "SEPSIS_DETERIORATION":
                    sepsisDetectionRate.markEvent();
                    patientRiskScoreDistribution.update((long) (pattern.getConfidence() * 100));
                    break;
                case "CARDIAC_DETERIORATION":
                    cardiacEventDetectionRate.markEvent();
                    break;
                case "MEDICATION_NONADHERENCE":
                    medicationErrorPrevention.markEvent();
                    break;
                case "PATHWAY_ADHERENCE":
                    pathwayAdherenceCount.inc();
                    if (pattern.getPatternDetails().containsKey("completion_time_hours")) {
                        double completionTime = (Double) pattern.getPatternDetails().get("completion_time_hours");
                        pathwayCompletionTime.update((long) completionTime);
                    }
                    break;
            }

            // Track alert quality metrics
            if (pattern.getPatternDetails().containsKey("response_time_minutes")) {
                double responseTime = (Double) pattern.getPatternDetails().get("response_time_minutes");
                alertResponseTime.update((long) responseTime);
            }

            if (pattern.getPatternDetails().containsKey("is_escalated")) {
                boolean isEscalated = (Boolean) pattern.getPatternDetails().get("is_escalated");
                if (isEscalated) {
                    alertEscalations.inc();
                }
            }

            if (pattern.getPatternDetails().containsKey("alert_fatigue_indicator")) {
                boolean isFatigueRelated = (Boolean) pattern.getPatternDetails().get("alert_fatigue_indicator");
                if (isFatigueRelated) {
                    alertFatigueRate.markEvent();
                }
            }
        }

        private double calculatePathwayAdherenceRate() {
            // This would be calculated based on historical data
            // For now, return a placeholder calculation
            return 85.5; // 85.5% adherence rate
        }
    }

    // ========== Technical/Operational Metrics ==========

    /**
     * Event processing performance metrics
     */
    public static class EventProcessingMetrics extends ProcessFunction<CanonicalEvent, CanonicalEvent> {

        // Processing performance metrics
        private transient Histogram eventProcessingLatency;
        private transient Counter eventsProcessedTotal;
        private transient Counter processingErrors;
        private transient Meter eventThroughputRate;

        // State management metrics
        private transient Histogram stateSizePerPatient;
        private transient Counter stateOperations;
        private transient Histogram stateAccessLatency;

        // Backpressure and resource metrics
        private transient Gauge<Double> backpressureRatio;
        private transient Gauge<Long> totalMemoryUsage;
        private transient Gauge<Double> cpuUtilization;

        // Side output for detailed metrics
        private static final OutputTag<MetricEvent> METRICS_OUTPUT =
            new OutputTag<MetricEvent>("metrics-output") {};

        @Override
        public void open(OpenContext openContext) throws Exception {
            super.open(openContext);

            MetricGroup operationalMetrics = getRuntimeContext()
                .getMetricGroup()
                .addGroup("operational");

            // Initialize processing metrics
            eventProcessingLatency = operationalMetrics.histogram("event_processing_latency_ms",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            eventsProcessedTotal = operationalMetrics.counter("events_processed_total");
            processingErrors = operationalMetrics.counter("processing_errors_total");

            eventThroughputRate = operationalMetrics.meter("event_throughput_per_second",
                new DropwizardMeterWrapper(new com.codahale.metrics.Meter()));

            // Initialize state management metrics
            stateSizePerPatient = operationalMetrics.histogram("state_size_per_patient_kb",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(500))));

            stateOperations = operationalMetrics.counter("state_operations_total");
            stateAccessLatency = operationalMetrics.histogram("state_access_latency_ms",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            // Initialize resource metrics
            backpressureRatio = operationalMetrics.gauge("backpressure_ratio",
                new Gauge<Double>() {
                    @Override
                    public Double getValue() {
                        return getBackpressureRatio();
                    }
                });

            totalMemoryUsage = operationalMetrics.gauge("total_memory_usage_mb",
                new Gauge<Long>() {
                    @Override
                    public Long getValue() {
                        Runtime runtime = Runtime.getRuntime();
                        return (runtime.totalMemory() - runtime.freeMemory()) / (1024 * 1024);
                    }
                });

            cpuUtilization = operationalMetrics.gauge("cpu_utilization_percent",
                new Gauge<Double>() {
                    @Override
                    public Double getValue() {
                        return getCpuUtilization();
                    }
                });
        }

        @Override
        public void processElement(CanonicalEvent event, Context ctx, Collector<CanonicalEvent> out) throws Exception {
            long startTime = System.currentTimeMillis();

            try {
                // Process the event (placeholder for actual processing)
                processEvent(event);

                // Update success metrics
                eventsProcessedTotal.inc();
                eventThroughputRate.markEvent();

                // Calculate and record processing latency
                long processingTime = System.currentTimeMillis() - startTime;
                eventProcessingLatency.update(processingTime);

                // Emit detailed metrics event
                emitMetricsEvent(ctx, event, processingTime, true);

                out.collect(event);

            } catch (Exception e) {
                processingErrors.inc();

                // Emit error metrics event
                emitMetricsEvent(ctx, event, System.currentTimeMillis() - startTime, false);

                throw e; // Re-throw to maintain error handling
            }
        }

        private void processEvent(CanonicalEvent event) {
            // Simulate state operations
            long stateStartTime = System.currentTimeMillis();

            // Placeholder for state access
            stateOperations.inc();

            long stateLatency = System.currentTimeMillis() - stateStartTime;
            stateAccessLatency.update(stateLatency);

            // Simulate state size calculation (would be actual in real implementation)
            stateSizePerPatient.update(250); // 250KB average per patient
        }

        private void emitMetricsEvent(Context ctx, CanonicalEvent event, long processingTime, boolean success) {
            MetricEvent metricEvent = MetricEvent.builder()
                .timestamp(System.currentTimeMillis())
                .patientId(((CanonicalEvent)event).getPatientId())
                .eventType(event.getEventType().toString())
                .processingTimeMs(processingTime)
                .success(success)
                .build();

            ctx.output(METRICS_OUTPUT, metricEvent);
        }

        private double getBackpressureRatio() {
            // Placeholder implementation - would integrate with Flink's backpressure monitoring
            return 0.1; // 10% backpressure
        }

        private double getCpuUtilization() {
            // Placeholder implementation - would integrate with system monitoring
            return 65.5; // 65.5% CPU utilization
        }
    }

    // ========== Kafka Consumer Lag Metrics ==========

    /**
     * Kafka consumer lag monitoring for stream processing health
     */
    public static class KafkaLagMetrics extends RichMapFunction<RawEvent, RawEvent> {

        private transient Histogram kafkaLagTime;
        private transient Counter messagesConsumed;
        private transient Gauge<Long> maxLagAcrossTopics;
        private transient Map<String, Long> topicLags;

        @Override
        public void open(OpenContext openContext) throws Exception {
            super.open(openContext);

            MetricGroup kafkaMetrics = getRuntimeContext()
                .getMetricGroup()
                .addGroup("kafka");

            kafkaLagTime = kafkaMetrics.histogram("consumer_lag_ms",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            messagesConsumed = kafkaMetrics.counter("messages_consumed_total");

            topicLags = new HashMap<>();

            maxLagAcrossTopics = kafkaMetrics.gauge("max_lag_across_topics_ms",
                new Gauge<Long>() {
                    @Override
                    public Long getValue() {
                        return topicLags.values().stream()
                            .mapToLong(Long::longValue)
                            .max()
                            .orElse(0L);
                    }
                });
        }

        @Override
        public RawEvent map(RawEvent event) throws Exception {
            messagesConsumed.inc();

            // Calculate lag (current time - event timestamp)
            long currentTime = System.currentTimeMillis();
            long eventTime = event.getEventTime();
            long lag = currentTime - eventTime;

            kafkaLagTime.update(lag);

            // Update topic-specific lag
            topicLags.put(event.getSource(), lag);

            return event;
        }
    }

    // ========== ML Model Performance Metrics ==========

    /**
     * Machine learning model performance and accuracy metrics
     */
    public static class MLModelMetrics extends RichMapFunction<MLPrediction, MLPrediction> {

        private transient Histogram modelInferenceLatency;
        private transient Counter predictionsGenerated;
        private transient Histogram predictionConfidenceDistribution;
        private transient Map<String, Counter> modelAccuracyCounters;
        private transient Map<String, Histogram> modelFeatureImportance;

        @Override
        public void open(OpenContext openContext) throws Exception {
            super.open(openContext);

            MetricGroup mlMetrics = getRuntimeContext()
                .getMetricGroup()
                .addGroup("ml_models");

            modelInferenceLatency = mlMetrics.histogram("inference_latency_ms",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            predictionsGenerated = mlMetrics.counter("predictions_generated_total");

            predictionConfidenceDistribution = mlMetrics.histogram("prediction_confidence_distribution",
                new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(1000))));

            // Initialize model-specific metrics
            modelAccuracyCounters = new HashMap<>();
            modelFeatureImportance = new HashMap<>();

            String[] modelTypes = {"sepsis_risk", "readmission_risk", "fall_risk", "mortality_risk", "deterioration_risk"};
            for (String modelType : modelTypes) {
                modelAccuracyCounters.put(modelType + "_correct",
                    mlMetrics.counter("model_" + modelType + "_correct_predictions"));
                modelAccuracyCounters.put(modelType + "_total",
                    mlMetrics.counter("model_" + modelType + "_total_predictions"));

                modelFeatureImportance.put(modelType,
                    mlMetrics.histogram("model_" + modelType + "_feature_importance",
                        new DropwizardHistogramWrapper(new com.codahale.metrics.Histogram(new SlidingWindowReservoir(100)))));
            }
        }

        @Override
        public MLPrediction map(MLPrediction prediction) throws Exception {
            predictionsGenerated.inc();

            // Record inference latency (using prediction time as approximation)
            long inferenceTime = System.currentTimeMillis() - prediction.getPredictionTime();
            modelInferenceLatency.update(inferenceTime);

            // Record prediction confidence
            double confidence = prediction.getConfidence();
            predictionConfidenceDistribution.update((long) (confidence * 100));

            // Update model-specific metrics
            String predictionType = prediction.getModelType();
            if (modelAccuracyCounters.containsKey(predictionType + "_total")) {
                modelAccuracyCounters.get(predictionType + "_total").inc();

                // If we have ground truth (in model metadata), update accuracy
                if (prediction.getModelMetadata().containsKey("ground_truth")) {
                    boolean isCorrect = (Boolean) prediction.getModelMetadata().get("is_correct_prediction");
                    if (isCorrect) {
                        modelAccuracyCounters.get(predictionType + "_correct").inc();
                    }
                }
            }

            // Record feature importance if available
            if (prediction.getFeatureImportance() != null &&
                modelFeatureImportance.containsKey(predictionType)) {
                double avgImportance = prediction.getFeatureImportance().values().stream()
                    .mapToDouble(Double::doubleValue)
                    .average()
                    .orElse(0.0);
                modelFeatureImportance.get(predictionType).update((long) (avgImportance * 100));
            }

            return prediction;
        }
    }

    // ========== Healthcare Metrics Sink ==========

    /**
     * Dedicated sink for healthcare metrics to external monitoring systems
     *
     * TODO: Migrate to Flink 2.x Sink API (disabled for now - SinkFunction removed in 2.x)
     */
    /* DISABLED FOR FLINK 2.X MIGRATION
    public static class HealthcareMetricsSink implements SinkFunction<MetricEvent> {

        public void invoke(MetricEvent metricEvent, Context context) throws Exception {
            // Send to Prometheus/Grafana
            sendToPrometheus(metricEvent);

            // Send to healthcare-specific monitoring dashboard
            sendToHealthcareDashboard(metricEvent);

            // Send to compliance audit system if required
            if (metricEvent.requiresCompliance()) {
                sendToComplianceSystem(metricEvent);
            }
        }

        private void sendToPrometheus(MetricEvent metricEvent) {
            // Implementation would push metrics to Prometheus pushgateway or expose via HTTP
            // Format metrics according to Prometheus standards
            String prometheusMetric = formatForPrometheus(metricEvent);
            // Send via HTTP POST to pushgateway
        }

        private void sendToHealthcareDashboard(MetricEvent metricEvent) {
            // Send to specialized healthcare monitoring dashboard
            // Could be Epic's monitoring system, Cerner PowerChart, or custom dashboard
        }

        private void sendToComplianceSystem(MetricEvent metricEvent) {
            // Send to compliance and audit systems for regulatory reporting
            // HIPAA, HITECH, Joint Commission, CMS reporting
        }

        private String formatForPrometheus(MetricEvent metricEvent) {
            return String.format(
                "healthcare_event_processing_time{patient_id=\"%s\",event_type=\"%s\"} %d %d",
                metricEvent.getPatientId(),
                metricEvent.getEventType(),
                metricEvent.getProcessingTimeMs(),
                metricEvent.getTimestamp()
            );
        }
    }
    */ // END DISABLED FOR FLINK 2.X MIGRATION

    // ========== Metric Event Model ==========

    /**
     * Internal metric event model for comprehensive monitoring
     */
    public static class MetricEvent {
        private long timestamp;
        private String patientId;
        private String eventType;
        private long processingTimeMs;
        private boolean success;
        private Map<String, Object> metadata;

        private MetricEvent(Builder builder) {
            this.timestamp = builder.timestamp;
            this.patientId = builder.patientId;
            this.eventType = builder.eventType;
            this.processingTimeMs = builder.processingTimeMs;
            this.success = builder.success;
            this.metadata = builder.metadata;
        }

        public static Builder builder() {
            return new Builder();
        }

        public static class Builder {
            private long timestamp;
            private String patientId;
            private String eventType;
            private long processingTimeMs;
            private boolean success;
            private Map<String, Object> metadata = new HashMap<>();

            public Builder timestamp(long timestamp) {
                this.timestamp = timestamp;
                return this;
            }

            public Builder patientId(String patientId) {
                this.patientId = patientId;
                return this;
            }

            public Builder eventType(String eventType) {
                this.eventType = eventType;
                return this;
            }

            public Builder processingTimeMs(long processingTimeMs) {
                this.processingTimeMs = processingTimeMs;
                return this;
            }

            public Builder success(boolean success) {
                this.success = success;
                return this;
            }

            public Builder addMetadata(String key, Object value) {
                this.metadata.put(key, value);
                return this;
            }

            public MetricEvent build() {
                return new MetricEvent(this);
            }
        }

        // Getters
        public long getTimestamp() { return timestamp; }
        public String getPatientId() { return patientId; }
        public String getEventType() { return eventType; }
        public long getProcessingTimeMs() { return processingTimeMs; }
        public boolean isSuccess() { return success; }
        public Map<String, Object> getMetadata() { return metadata; }

        public boolean requiresCompliance() {
            return metadata.containsKey("compliance_required") &&
                   (Boolean) metadata.get("compliance_required");
        }
    }
}
