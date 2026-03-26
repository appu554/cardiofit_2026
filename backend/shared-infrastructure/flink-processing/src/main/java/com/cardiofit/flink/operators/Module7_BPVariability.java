package com.cardiofit.flink.operators;

import com.cardiofit.flink.analytics.*;
import com.cardiofit.flink.models.BPVariabilityMetrics;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import com.cardiofit.flink.utils.KafkaTopics;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;

import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.api.common.serialization.SerializationSchema;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.state.ValueState;
import org.apache.flink.api.common.state.ValueStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.TypeHint;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.util.Collector;
import org.apache.flink.util.OutputTag;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.*;
import java.util.*;
import java.util.stream.Collectors;

/**
 * Module 7: BP Variability Engine (DD#1)
 *
 * Consumes BP readings from ingestion.vitals, maintains per-patient keyed state
 * (30-day rolling window of daily BP summaries), and computes:
 * 1. ARV (7d/30d) — Average Real Variability
 * 2. Morning surge detection (sleep-trough method)
 * 3. Dipping pattern classification (nocturnal dip ratio)
 * 4. Hypertensive crisis bypass (SBP>180/DBP>120 → side output to safety-critical, <1s latency)
 */
public class Module7_BPVariability {

    private static final Logger LOG = LoggerFactory.getLogger(Module7_BPVariability.class);
    private static final String CONSUMER_GROUP = "flink-module7-bp-variability";
    private static final int PARALLELISM = 4;

    private static final OutputTag<BPVariabilityMetrics> CRISIS_TAG =
        new OutputTag<BPVariabilityMetrics>("hypertensive-crisis") {};

    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
        env.setParallelism(PARALLELISM);
        env.enableCheckpointing(30_000L);
        createBPVariabilityPipeline(env);
        env.execute("Module 7: BP Variability Engine");
    }

    public static void createBPVariabilityPipeline(StreamExecutionEnvironment env) {
        String bootstrap = KafkaConfigLoader.getBootstrapServers();

        KafkaSource<Map<String, Object>> source = KafkaSource
            .<Map<String, Object>>builder()
            .setBootstrapServers(bootstrap)
            .setTopics(KafkaTopics.INGESTION_VITALS.getTopicName())
            .setGroupId(CONSUMER_GROUP)
            .setValueOnlyDeserializer(new Module8_ComorbidityInteraction.JsonMapDeserializer())
            .build();

        DataStream<Map<String, Object>> events = env.fromSource(
            source,
            WatermarkStrategy.<Map<String, Object>>forBoundedOutOfOrderness(Duration.ofMinutes(2))
                .withTimestampAssigner((e, ts) -> {
                    Object t = e.get("timestamp");
                    return t instanceof Number ? ((Number) t).longValue() : System.currentTimeMillis();
                }),
            "Kafka Source: Vitals"
        );

        SingleOutputStreamOperator<BPVariabilityMetrics> metrics = events
            .filter(e -> "BP".equals(e.get("vitalType")) || "BLOOD_PRESSURE".equals(e.get("vitalType")))
            .keyBy(e -> String.valueOf(e.getOrDefault("patientId", "unknown")))
            .process(new BPVariabilityProcessor())
            .uid("BP Variability Processor")
            .name("BP Variability Processor");

        // Main output → flink.bp-variability-metrics
        metrics.sinkTo(
            KafkaSink.<BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.FLINK_BP_VARIABILITY_METRICS.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());

        // Side output: crisis → ingestion.safety-critical
        metrics.getSideOutput(CRISIS_TAG).sinkTo(
            KafkaSink.<BPVariabilityMetrics>builder()
                .setBootstrapServers(bootstrap)
                .setRecordSerializer(
                    KafkaRecordSerializationSchema.builder()
                        .setTopic(KafkaTopics.INGESTION_SAFETY_CRITICAL.getTopicName())
                        .setValueSerializationSchema(new BPMetricsSerializer())
                        .build())
                .build());
    }

    static class BPVariabilityProcessor
            extends KeyedProcessFunction<String, Map<String, Object>, BPVariabilityMetrics> {

        // date string "YYYY-MM-DD" → "avgSBP,avgDBP,count"
        private transient MapState<String, String> dailySummaries;
        private transient ValueState<Double> lastMorningSBP;
        private transient ValueState<Double> lastEveningSBP;

        @Override
        public void open(Configuration params) {
            dailySummaries = getRuntimeContext().getMapState(
                new MapStateDescriptor<>("daily_bp_30d", String.class, String.class));
            lastMorningSBP = getRuntimeContext().getState(
                new ValueStateDescriptor<>("morning_sbp", Double.class));
            lastEveningSBP = getRuntimeContext().getState(
                new ValueStateDescriptor<>("evening_sbp", Double.class));
        }

        @Override
        public void processElement(Map<String, Object> event, Context ctx,
                                   Collector<BPVariabilityMetrics> out) throws Exception {
            double sbp = ((Number) event.getOrDefault("sbp", 0)).doubleValue();
            double dbp = ((Number) event.getOrDefault("dbp", 0)).doubleValue();
            String patientId = ctx.getCurrentKey();
            boolean isCuffless = Boolean.TRUE.equals(event.get("cuffless"));

            // 1. Crisis bypass — SBP>180 or DBP>120
            if (HypertensiveCrisisDetector.isCrisis(sbp, dbp) &&
                !HypertensiveCrisisDetector.requiresCuffConfirmation(isCuffless)) {
                BPVariabilityMetrics crisis = new BPVariabilityMetrics();
                crisis.setPatientId(patientId);
                crisis.setBpControlStatus("CRISIS");
                crisis.setComputedAt(Instant.now());
                ctx.output(CRISIS_TAG, crisis);
            }

            // 2. Update daily summary
            String today = LocalDate.now().toString();
            String existing = dailySummaries.get(today);
            double dayAvgSBP = sbp, dayAvgDBP = dbp;
            int count = 1;
            if (existing != null) {
                String[] parts = existing.split(",");
                double prevAvg = Double.parseDouble(parts[0]);
                double prevDBP = Double.parseDouble(parts[1]);
                int prevCount = Integer.parseInt(parts[2]);
                dayAvgSBP = (prevAvg * prevCount + sbp) / (prevCount + 1);
                dayAvgDBP = (prevDBP * prevCount + dbp) / (prevCount + 1);
                count = prevCount + 1;
            }
            dailySummaries.put(today, String.format("%.1f,%.1f,%d", dayAvgSBP, dayAvgDBP, count));

            // 3. Track morning/evening for surge detection
            int hour = LocalTime.now().getHour();
            if (hour >= 6 && hour <= 9) { lastMorningSBP.update(sbp); }
            if (hour >= 20 && hour <= 23) { lastEveningSBP.update(sbp); }

            // 4. Compute variability metrics from 30-day state
            // Collect date→SBP pairs and sort by date to ensure temporal ordering
            // (MapState iteration order is not guaranteed)
            List<Map.Entry<String, String>> entries = new ArrayList<>();
            for (Map.Entry<String, String> entry : dailySummaries.entries()) {
                entries.add(new AbstractMap.SimpleEntry<>(entry.getKey(), entry.getValue()));
            }
            entries.sort(Map.Entry.comparingByKey()); // ISO date strings sort lexicographically
            double[] sbpArr = entries.stream()
                .mapToDouble(e -> Double.parseDouble(e.getValue().split(",")[0]))
                .toArray();

            BPVariabilityMetrics result = new BPVariabilityMetrics();
            result.setPatientId(patientId);

            // ARV (7d and 30d)
            if (sbpArr.length >= 7) {
                double[] last7 = Arrays.copyOfRange(sbpArr, Math.max(0, sbpArr.length - 7), sbpArr.length);
                result.setArvSbp7d(ARVCalculator.compute(last7));
            }
            result.setArvSbp30d(ARVCalculator.compute(sbpArr));

            // Morning surge
            Double morning = lastMorningSBP.value();
            Double evening = lastEveningSBP.value();
            if (morning != null && evening != null) {
                result.setMorningSurgeToday(MorningSurgeDetector.computeSurge(morning, evening));
            }

            // Dipping (uses last night vs last day average)
            if (sbpArr.length >= 2) {
                Double nightAvgSBP = lastEveningSBP.value();
                if (nightAvgSBP != null) {
                    result.setDipClassification(
                        DippingPatternClassifier.classify(dayAvgSBP, nightAvgSBP));
                    result.setDipConfidence(DippingPatternClassifier.confidence(isCuffless));
                }
            }

            result.setComputedAt(Instant.now());
            out.collect(result);
        }
    }

    static class BPMetricsSerializer implements SerializationSchema<BPVariabilityMetrics> {
        private transient ObjectMapper mapper;
        @Override public void open(SerializationSchema.InitializationContext ctx) {
            mapper = new ObjectMapper(); mapper.registerModule(new JavaTimeModule());
        }
        @Override public byte[] serialize(BPVariabilityMetrics m) {
            try { return mapper.writeValueAsBytes(m); }
            catch (Exception e) { throw new RuntimeException("Serialize BPMetrics failed", e); }
        }
    }
}
