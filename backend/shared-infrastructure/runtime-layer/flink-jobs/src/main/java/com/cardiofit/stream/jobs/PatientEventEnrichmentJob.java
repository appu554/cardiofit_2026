package com.cardiofit.stream.jobs;

import com.cardiofit.stream.functions.EventEnrichmentFunction;
import com.cardiofit.stream.functions.ClinicalPatternDetectionFunction;
import com.cardiofit.stream.models.PatientEvent;
import com.cardiofit.stream.models.EnrichedPatientEvent;
import com.cardiofit.stream.sinks.FHIRStoreSink;
import com.cardiofit.stream.sinks.ElasticsearchSink;
import com.cardiofit.stream.sinks.NotificationSink;
import com.cardiofit.stream.sinks.Neo4jSummarySink;
import com.cardiofit.stream.utils.PatientEventSchema;
import com.cardiofit.stream.utils.FlinkConfigurationUtils;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.restartstrategy.RestartStrategies;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.api.common.time.Time;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.CheckpointingMode;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.Collector;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.util.Properties;

/**
 * Patient Event Enrichment Job
 *
 * Main Flink job that processes real-time patient events with <500ms end-to-end latency.
 * Performs semantic enrichment, clinical pattern detection, and multi-sink distribution.
 *
 * Processing Pipeline:
 * 1. Consume patient events from Kafka
 * 2. Apply watermarks for event time processing
 * 3. Key by patient ID for stateful processing
 * 4. Enrich events with semantic mesh data
 * 5. Detect clinical patterns using CEP
 * 6. Distribute to multiple sinks (FHIR, Elasticsearch, Notifications, Neo4j)
 *
 * Performance Targets:
 * - End-to-end latency: <500ms for critical events, <2s for standard events
 * - Throughput: 10,000 events/second
 * - Availability: 99.99%
 */
public class PatientEventEnrichmentJob {

    private static final Logger logger = LoggerFactory.getLogger(PatientEventEnrichmentJob.class);

    // Kafka topic configuration
    private static final String[] PATIENT_EVENT_TOPICS = {
        "patient-events",
        "medication-events",
        "safety-events",
        "vital-signs-events",
        "lab-result-events"
    };

    private static final String KAFKA_CONSUMER_GROUP = "patient-event-enrichment";

    public static void main(String[] args) throws Exception {
        logger.info("🚀 Starting Patient Event Enrichment Job");

        // Create streaming execution environment
        final StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure the execution environment
        configureExecutionEnvironment(env);

        // Create and execute the processing pipeline
        createProcessingPipeline(env);

        // Execute the job
        env.execute("Patient Event Enrichment Job v1.0.0");

        logger.info("✅ Patient Event Enrichment Job completed");
    }

    /**
     * Configure the Flink execution environment for clinical data processing
     */
    private static void configureExecutionEnvironment(StreamExecutionEnvironment env) {
        logger.info("⚙️ Configuring execution environment for clinical processing");

        // Set parallelism based on available slots
        env.setParallelism(4);

        // Enable checkpointing for fault tolerance
        env.enableCheckpointing(30_000, CheckpointingMode.EXACTLY_ONCE); // 30 seconds
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(10_000); // 10 seconds minimum
        env.getCheckpointConfig().setCheckpointTimeout(60_000); // 1 minute timeout
        env.getCheckpointConfig().setMaxConcurrentCheckpoints(1);

        // Configure restart strategy for production resilience
        env.setRestartStrategy(RestartStrategies.exponentialDelayRestart(
            Time.milliseconds(1000), // initial delay
            Time.milliseconds(60_000), // max delay
            1.2, // backoff multiplier
            Time.minutes(10), // reset time
            0.1 // jitter
        ));

        // Configure latency tracking for performance monitoring
        env.getConfig().setLatencyTrackingInterval(100); // 100ms

        // Enable object reuse for performance
        env.getConfig().enableObjectReuse();

        // Set time characteristic to event time
        env.getConfig().setAutoWatermarkInterval(1000); // 1 second watermark interval

        logger.info("✅ Environment configured - Parallelism: {}, Checkpointing: 30s",
                   env.getParallelism());
    }

    /**
     * Create the main processing pipeline
     */
    private static void createProcessingPipeline(StreamExecutionEnvironment env) throws Exception {
        logger.info("🏗️ Creating patient event processing pipeline");

        // 1. Create Kafka source for patient events
        KafkaSource<PatientEvent> kafkaSource = createKafkaSource();

        DataStream<PatientEvent> patientEvents = env
            .fromSource(
                kafkaSource,
                createWatermarkStrategy(),
                "patient-events-source"
            )
            .name("Patient Events Source")
            .uid("patient-events-source");

        logger.info("📥 Created Kafka source for topics: {}", String.join(", ", PATIENT_EVENT_TOPICS));

        // 2. Filter and route events based on priority
        SingleOutputStreamOperator<PatientEvent> filteredEvents = patientEvents
            .process(new EventFilterFunction())
            .name("Event Filter")
            .uid("event-filter");

        // 3. Key by patient ID for stateful processing
        DataStream<PatientEvent> keyedEvents = filteredEvents
            .keyBy(PatientEvent::getPatientId);

        // 4. Enrich events with semantic mesh data
        SingleOutputStreamOperator<EnrichedPatientEvent> enrichedEvents = keyedEvents
            .process(new EventEnrichmentFunction())
            .name("Event Enrichment")
            .uid("event-enrichment");

        logger.info("🔍 Added semantic enrichment processing");

        // 5. Detect clinical patterns using complex event processing
        SingleOutputStreamOperator<EnrichedPatientEvent> patternDetectedEvents = enrichedEvents
            .keyBy(event -> event.getOriginalEvent().getPatientId())
            .process(new ClinicalPatternDetectionFunction())
            .name("Pattern Detection")
            .uid("pattern-detection");

        logger.info("🎯 Added clinical pattern detection");

        // 6. Setup multi-sink distribution
        setupMultiSinkDistribution(patternDetectedEvents);

        logger.info("✅ Processing pipeline created successfully");
    }

    /**
     * Create Kafka source with proper configuration
     */
    private static KafkaSource<PatientEvent> createKafkaSource() {
        Properties kafkaProperties = FlinkConfigurationUtils.getKafkaProperties();

        return KafkaSource.<PatientEvent>builder()
            .setBootstrapServers(kafkaProperties.getProperty("bootstrap.servers", "kafka:9092"))
            .setTopics(PATIENT_EVENT_TOPICS)
            .setGroupId(KAFKA_CONSUMER_GROUP)
            .setStartingOffsets(OffsetsInitializer.latest())
            .setDeserializer(new PatientEventSchema())
            .setProperty("auto.offset.reset", "latest")
            .setProperty("enable.auto.commit", "false")
            .setProperty("max.poll.records", "1000")
            .setProperty("max.poll.interval.ms", "300000") // 5 minutes
            .setProperty("session.timeout.ms", "30000") // 30 seconds
            .setProperty("heartbeat.interval.ms", "10000") // 10 seconds
            .build();
    }

    /**
     * Create watermark strategy for event time processing
     */
    private static WatermarkStrategy<PatientEvent> createWatermarkStrategy() {
        return WatermarkStrategy
            .<PatientEvent>forBoundedOutOfOrderness(Duration.ofSeconds(5))
            .withTimestampAssigner((event, timestamp) ->
                event.getTimestamp().atZone(java.time.ZoneId.systemDefault()).toEpochSecond() * 1000)
            .withIdleness(Duration.ofMinutes(1)); // Handle idle partitions
    }

    /**
     * Setup multi-sink distribution for enriched events
     */
    private static void setupMultiSinkDistribution(SingleOutputStreamOperator<EnrichedPatientEvent> enrichedEvents) {
        logger.info("📤 Setting up multi-sink distribution");

        // Sink 1: FHIR Store (system of record)
        enrichedEvents
            .addSink(new FHIRStoreSink())
            .name("FHIR Store Sink")
            .uid("fhir-store-sink");

        // Sink 2: Elasticsearch (searchability and analytics)
        enrichedEvents
            .addSink(new ElasticsearchSink())
            .name("Elasticsearch Sink")
            .uid("elasticsearch-sink");

        // Sink 3: Push notifications (urgent alerts only)
        enrichedEvents
            .filter(EnrichedPatientEvent::requiresPushNotification)
            .addSink(new NotificationSink())
            .name("Notification Sink")
            .uid("notification-sink");

        // Sink 4: Neo4j summary (graph relationships)
        enrichedEvents
            .addSink(new Neo4jSummarySink())
            .name("Neo4j Summary Sink")
            .uid("neo4j-summary-sink");

        logger.info("✅ Multi-sink distribution configured: FHIR, Elasticsearch, Notifications, Neo4j");
    }

    /**
     * Event Filter Function - filters and routes events based on clinical relevance
     */
    public static class EventFilterFunction extends ProcessFunction<PatientEvent, PatientEvent> {

        private static final Logger logger = LoggerFactory.getLogger(EventFilterFunction.class);

        // Counter for monitoring
        private transient org.apache.flink.metrics.Counter processedEventsCounter;
        private transient org.apache.flink.metrics.Counter filteredEventsCounter;
        private transient org.apache.flink.metrics.Counter criticalEventsCounter;

        @Override
        public void open(Configuration parameters) throws Exception {
            super.open(parameters);

            // Initialize metrics counters
            processedEventsCounter = getRuntimeContext()
                .getMetricGroup()
                .counter("processed_events");

            filteredEventsCounter = getRuntimeContext()
                .getMetricGroup()
                .counter("filtered_events");

            criticalEventsCounter = getRuntimeContext()
                .getMetricGroup()
                .counter("critical_events");
        }

        @Override
        public void processElement(PatientEvent event, Context ctx, Collector<PatientEvent> out)
                throws Exception {

            processedEventsCounter.inc();

            // Filter out non-clinical events
            if (!isClinicallyRelevant(event)) {
                filteredEventsCounter.inc();
                logger.debug("Filtered non-clinical event: {}", event.getEventType());
                return;
            }

            // Track critical events
            if (event.isCritical()) {
                criticalEventsCounter.inc();
                logger.info("Processing critical event: {} for patient {}",
                           event.getEventType(), event.getPatientId());
            }

            // Forward clinically relevant events
            out.collect(event);
        }

        /**
         * Determine if an event is clinically relevant and should be processed
         */
        private boolean isClinicallyRelevant(PatientEvent event) {
            // Always process critical events
            if (event.isCritical()) {
                return true;
            }

            // Process clinical events
            if (event.isClinicalEvent()) {
                return true;
            }

            // Process specific event types that impact clinical decisions
            String eventType = event.getEventType();
            return eventType != null && (
                eventType.contains("medication") ||
                eventType.contains("vital") ||
                eventType.contains("lab") ||
                eventType.contains("allergy") ||
                eventType.contains("diagnosis") ||
                eventType.contains("procedure") ||
                eventType.contains("safety")
            );
        }
    }
}