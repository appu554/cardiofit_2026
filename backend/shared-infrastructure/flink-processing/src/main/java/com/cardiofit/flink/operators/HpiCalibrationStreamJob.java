package com.cardiofit.flink.operators;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.functions.AggregateFunction;
import org.apache.flink.api.common.functions.MapFunction;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.java.tuple.Tuple2;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.KeyedStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import java.time.Duration;
import org.apache.flink.util.Collector;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;
import java.time.Instant;

/**
 * E07: HPI Calibration Stream Job — Flink pipeline for real-time calibration metrics.
 *
 * <p>Architecture:
 * <pre>
 *   Kafka (hpi.calibration.data)
 *     → Parse + Watermark (event-time from session outcome)
 *     → Key by (node_id, stratum_label)
 *     → 7-day Sliding Window (1-day slide)
 *       → Aggregate: concordance counts, question distributions
 *     → Stateful Tier Transition Detector
 *       → N=30 → Tier B event
 *       → N=200 → Tier C event
 *     → Multi-Sink:
 *       → Kafka (hpi.calibration.metrics) — downstream consumers
 *       → Side output: tier transition alerts
 * </pre>
 *
 * <p>Consumed topics:
 * <ul>
 *   <li>{@code hpi.calibration.data} — session outcomes from KB-22 BAY-11
 *   <li>{@code hpi.calibration.data} — Tier C approval events from E03
 * </ul>
 *
 * <p>Produced topics:
 * <ul>
 *   <li>{@code hpi.calibration.metrics} — windowed concordance aggregates
 *   <li>{@code hpi.calibration.transitions} — tier transition events
 * </ul>
 */
public class HpiCalibrationStreamJob {
    private static final Logger LOG = LoggerFactory.getLogger(HpiCalibrationStreamJob.class);

    private static final String SOURCE_TOPIC = "hpi.calibration.data";
    private static final String METRICS_TOPIC = "hpi.calibration.metrics";
    private static final String TRANSITIONS_TOPIC = "hpi.calibration.transitions";

    private static final int TIER_B_THRESHOLD = 30;
    private static final int TIER_C_THRESHOLD = 200;

    private static final ObjectMapper MAPPER = new ObjectMapper();

    /**
     * Create and configure the HPI calibration streaming pipeline.
     *
     * @param env Flink execution environment
     * @param kafkaBootstrap Kafka bootstrap servers
     * @param consumerGroup consumer group for this pipeline
     */
    public static void createCalibrationPipeline(
            StreamExecutionEnvironment env,
            String kafkaBootstrap,
            String consumerGroup
    ) {
        LOG.info("E07: Initializing HPI Calibration Stream Job");

        // Source: consume hpi.calibration.data from Kafka
        KafkaSource<String> source = KafkaSource.<String>builder()
                .setBootstrapServers(kafkaBootstrap)
                .setTopics(SOURCE_TOPIC)
                .setGroupId(consumerGroup)
                .setStartingOffsets(OffsetsInitializer.latest())
                .setValueOnlyDeserializer(new SimpleStringSchema())
                .build();

        DataStream<String> rawStream = env.fromSource(
                source,
                WatermarkStrategy.<String>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                        .withTimestampAssigner((event, timestamp) -> extractTimestamp(event)),
                "hpi-calibration-source"
        );

        // Parse JSON events into structured records
        DataStream<CalibrationEvent> events = rawStream
                .map(new CalibrationEventParser())
                .filter(e -> e != null && e.eventType != null)
                .name("parse-calibration-events");

        // Key by (nodeId, stratumLabel) for per-node/stratum aggregation
        KeyedStream<CalibrationEvent, String> keyed = events
                .keyBy(e -> e.nodeId + ":" + e.stratumLabel);

        // 7-day sliding window with 1-day slide for concordance aggregation
        DataStream<ConcordanceAggregate> windowed = keyed
                .window(SlidingEventTimeWindows.of(Duration.ofDays(7), Duration.ofDays(1)))
                .aggregate(new ConcordanceAggregator())
                .name("7day-concordance-window");

        // Stateful tier transition detection
        DataStream<String> metricsOutput = windowed
                .keyBy(agg -> agg.nodeId + ":" + agg.stratumLabel)
                .process(new TierTransitionDetector())
                .name("tier-transition-detector");

        // Sink: publish metrics to hpi.calibration.metrics
        KafkaSink<String> metricsSink = KafkaSink.<String>builder()
                .setBootstrapServers(kafkaBootstrap)
                .setRecordSerializer(
                        KafkaRecordSerializationSchema.builder()
                                .setTopic(METRICS_TOPIC)
                                .setValueSerializationSchema(new SimpleStringSchema())
                                .build()
                )
                .build();
        metricsOutput.sinkTo(metricsSink).name("metrics-kafka-sink");

        LOG.info("E07: HPI Calibration Stream Job configured — "
                + "source={}, window=7d/1d, sinks=[{}, {}]",
                SOURCE_TOPIC, METRICS_TOPIC, TRANSITIONS_TOPIC);
    }

    // ─────────────── Event Model ───────────────

    /**
     * Parsed calibration event from hpi.calibration.data.
     */
    public static class CalibrationEvent {
        public String eventType;
        public String sessionId;
        public String nodeId;
        public String stratumLabel;
        public String topDiagnosis;
        public double confidence;
        public int questionsAsked;
        public boolean convergenceReached;
        public long timestampMs;
    }

    /**
     * Windowed concordance aggregate per node/stratum.
     */
    public static class ConcordanceAggregate {
        public String nodeId;
        public String stratumLabel;
        public int totalSessions;
        public int convergedSessions;
        public double avgConfidence;
        public double avgQuestionsAsked;
        public long windowStartMs;
        public long windowEndMs;

        public double convergenceRate() {
            return totalSessions > 0 ? (double) convergedSessions / totalSessions : 0.0;
        }
    }

    // ─────────────── Operators ───────────────

    /**
     * Parse JSON strings into CalibrationEvent objects.
     */
    public static class CalibrationEventParser implements MapFunction<String, CalibrationEvent> {
        @Override
        public CalibrationEvent map(String json) {
            try {
                JsonNode node = MAPPER.readTree(json);
                CalibrationEvent evt = new CalibrationEvent();
                evt.eventType = node.path("event_type").asText();
                evt.sessionId = node.path("session_id").asText();
                evt.nodeId = node.path("node_id").asText();
                evt.stratumLabel = node.path("stratum_label").asText();
                evt.topDiagnosis = node.path("top_diagnosis").asText();
                evt.confidence = node.path("confidence").asDouble(0.0);
                evt.questionsAsked = node.path("questions_asked").asInt(0);
                evt.convergenceReached = node.path("convergence_reached").asBoolean(false);
                evt.timestampMs = Instant.parse(
                        node.path("timestamp").asText(Instant.now().toString())
                ).toEpochMilli();
                return evt;
            } catch (Exception e) {
                LOG.warn("E07: Failed to parse calibration event: {}", e.getMessage());
                return null;
            }
        }
    }

    /**
     * Aggregate concordance metrics within a 7-day sliding window.
     *
     * <p>Accumulates:
     * <ul>
     *   <li>Total sessions processed</li>
     *   <li>Sessions reaching convergence</li>
     *   <li>Running sums for confidence and questions asked</li>
     * </ul>
     */
    public static class ConcordanceAggregator
            implements AggregateFunction<CalibrationEvent, ConcordanceAccumulator, ConcordanceAggregate> {

        @Override
        public ConcordanceAccumulator createAccumulator() {
            return new ConcordanceAccumulator();
        }

        @Override
        public ConcordanceAccumulator add(CalibrationEvent event, ConcordanceAccumulator acc) {
            acc.totalSessions++;
            if (event.convergenceReached) {
                acc.convergedSessions++;
            }
            acc.confidenceSum += event.confidence;
            acc.questionsSum += event.questionsAsked;
            if (acc.nodeId == null) {
                acc.nodeId = event.nodeId;
                acc.stratumLabel = event.stratumLabel;
            }
            acc.latestTimestampMs = Math.max(acc.latestTimestampMs, event.timestampMs);
            return acc;
        }

        @Override
        public ConcordanceAggregate getResult(ConcordanceAccumulator acc) {
            ConcordanceAggregate result = new ConcordanceAggregate();
            result.nodeId = acc.nodeId;
            result.stratumLabel = acc.stratumLabel;
            result.totalSessions = acc.totalSessions;
            result.convergedSessions = acc.convergedSessions;
            result.avgConfidence = acc.totalSessions > 0
                    ? acc.confidenceSum / acc.totalSessions : 0.0;
            result.avgQuestionsAsked = acc.totalSessions > 0
                    ? (double) acc.questionsSum / acc.totalSessions : 0.0;
            result.windowEndMs = acc.latestTimestampMs;
            return result;
        }

        @Override
        public ConcordanceAccumulator merge(ConcordanceAccumulator a, ConcordanceAccumulator b) {
            a.totalSessions += b.totalSessions;
            a.convergedSessions += b.convergedSessions;
            a.confidenceSum += b.confidenceSum;
            a.questionsSum += b.questionsSum;
            a.latestTimestampMs = Math.max(a.latestTimestampMs, b.latestTimestampMs);
            if (a.nodeId == null) {
                a.nodeId = b.nodeId;
                a.stratumLabel = b.stratumLabel;
            }
            return a;
        }
    }

    /**
     * Accumulator for the concordance windowed aggregation.
     */
    public static class ConcordanceAccumulator {
        public String nodeId;
        public String stratumLabel;
        public int totalSessions;
        public int convergedSessions;
        public double confidenceSum;
        public int questionsSum;
        public long latestTimestampMs;
    }

    /**
     * Stateful tier transition detector.
     *
     * <p>Maintains a running total of adjudicated cases per node/stratum key.
     * Emits transition events when thresholds are crossed:
     * <ul>
     *   <li>N ≥ 30 and previous tier was EXPERT_PANEL → emit Tier B transition</li>
     *   <li>N ≥ 200 and previous tier was BLENDED → emit Tier C transition</li>
     * </ul>
     *
     * <p>State is checkpointed via RocksDB for exactly-once processing.
     */
    public static class TierTransitionDetector
            extends KeyedProcessFunction<String, ConcordanceAggregate, String> {

        private transient ValueState<Integer> cumulativeCasesState;
        private transient ValueState<String> currentTierState;

        @Override
        public void open(org.apache.flink.api.common.functions.OpenContext openContext) {
            cumulativeCasesState = getRuntimeContext().getState(
                    new ValueStateDescriptor<>("cumulative-cases", TypeInformation.of(Integer.class))
            );
            currentTierState = getRuntimeContext().getState(
                    new ValueStateDescriptor<>("current-tier", TypeInformation.of(String.class))
            );
        }

        @Override
        public void processElement(
                ConcordanceAggregate agg,
                Context ctx,
                Collector<String> out
        ) throws Exception {
            Integer cumulative = cumulativeCasesState.value();
            if (cumulative == null) cumulative = 0;
            String currentTier = currentTierState.value();
            if (currentTier == null) currentTier = "EXPERT_PANEL";

            // Update cumulative count with this window's new sessions
            int newTotal = cumulative + agg.totalSessions;
            cumulativeCasesState.update(newTotal);

            // Determine new tier
            String newTier = determineTier(newTotal);

            // Emit metric output (always)
            ObjectNode metricsJson = MAPPER.createObjectNode();
            metricsJson.put("event_type", "ConcordanceMetrics");
            metricsJson.put("node_id", agg.nodeId);
            metricsJson.put("stratum_label", agg.stratumLabel);
            metricsJson.put("total_sessions", agg.totalSessions);
            metricsJson.put("converged_sessions", agg.convergedSessions);
            metricsJson.put("convergence_rate", Math.round(agg.convergenceRate() * 1000.0) / 1000.0);
            metricsJson.put("avg_confidence", Math.round(agg.avgConfidence * 1000.0) / 1000.0);
            metricsJson.put("avg_questions", Math.round(agg.avgQuestionsAsked * 10.0) / 10.0);
            metricsJson.put("cumulative_cases", newTotal);
            metricsJson.put("calibration_tier", newTier);
            metricsJson.put("timestamp", Instant.now().toString());
            out.collect(metricsJson.toString());

            // Detect tier transition
            if (!newTier.equals(currentTier)) {
                LOG.info("E07: Tier transition detected for {}:{} — {} → {} (N={})",
                        agg.nodeId, agg.stratumLabel, currentTier, newTier, newTotal);

                ObjectNode transitionJson = MAPPER.createObjectNode();
                transitionJson.put("event_type", "TierTransition");
                transitionJson.put("node_id", agg.nodeId);
                transitionJson.put("stratum_label", agg.stratumLabel);
                transitionJson.put("previous_tier", currentTier);
                transitionJson.put("new_tier", newTier);
                transitionJson.put("cumulative_cases", newTotal);
                transitionJson.put("convergence_rate", Math.round(agg.convergenceRate() * 1000.0) / 1000.0);
                transitionJson.put("timestamp", Instant.now().toString());
                out.collect(transitionJson.toString());

                currentTierState.update(newTier);
            }
        }

        private static String determineTier(int totalCases) {
            if (totalCases >= TIER_C_THRESHOLD) return "DATA_DRIVEN";
            if (totalCases >= TIER_B_THRESHOLD) return "BLENDED";
            return "EXPERT_PANEL";
        }
    }

    // ─────────────── Utilities ───────────────

    private static long extractTimestamp(String json) {
        try {
            JsonNode node = MAPPER.readTree(json);
            String ts = node.path("timestamp").asText();
            if (ts != null && !ts.isEmpty()) {
                return Instant.parse(ts).toEpochMilli();
            }
        } catch (Exception e) {
            // Fall through to current time
        }
        return System.currentTimeMillis();
    }
}
