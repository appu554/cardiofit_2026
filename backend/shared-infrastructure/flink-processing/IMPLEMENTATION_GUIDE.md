# Flink EHR Intelligence Engine - Implementation Guide

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Prerequisites](#prerequisites)
4. [Module Implementation](#module-implementation)
5. [Integration with CardioFit](#integration-with-cardiofit)
6. [Deployment](#deployment)
7. [Testing](#testing)
8. [Monitoring](#monitoring)

## Overview

The Flink EHR Intelligence Engine is a real-time stream processing system that provides clinical intelligence, pattern detection, and decision support for the CardioFit platform. It processes healthcare events from multiple sources, maintains patient state, detects clinical patterns, and generates actionable alerts.

### Key Capabilities
- Real-time processing of clinical events (vitals, labs, medications)
- Complex Event Processing (CEP) for clinical pattern detection
- Patient state management with persistent storage
- ML inference for risk scoring
- Multi-sink routing for alerts and analytics

## Architecture

```
Input Sources → Kafka → Flink Processing → Multi-Sink Outputs
                         ├── Module 1: Ingestion
                         ├── Module 2: Context Assembly
                         ├── Module 3: Semantic Mesh
                         ├── Module 4: Pattern Detection
                         ├── Module 5: ML Inference
                         └── Module 6: Egress Routing
```

### Technology Stack
- **Apache Flink**: 1.17.x (Stream processing engine)
- **Apache Kafka**: 3.x (Event streaming)
- **RocksDB**: State backend for persistent storage
- **ONNX Runtime**: ML model inference
- **Docker/Kubernetes**: Container orchestration
- **Prometheus/Grafana**: Monitoring and alerting

## Prerequisites

### System Requirements
```yaml
# Development Environment
CPU: 8+ cores
RAM: 16GB minimum
Storage: 50GB SSD
Java: JDK 11 or 17
Maven: 3.8+
Docker: 20.10+
```

### Dependencies Setup
```xml
<!-- pom.xml -->
<dependencies>
    <!-- Flink Core -->
    <dependency>
        <groupId>org.apache.flink</groupId>
        <artifactId>flink-streaming-java</artifactId>
        <version>1.17.1</version>
    </dependency>

    <!-- Flink Kafka Connector -->
    <dependency>
        <groupId>org.apache.flink</groupId>
        <artifactId>flink-connector-kafka</artifactId>
        <version>1.17.1</version>
    </dependency>

    <!-- RocksDB State Backend -->
    <dependency>
        <groupId>org.apache.flink</groupId>
        <artifactId>flink-statebackend-rocksdb</artifactId>
        <version>1.17.1</version>
    </dependency>

    <!-- FlinkCEP for Pattern Detection -->
    <dependency>
        <groupId>org.apache.flink</groupId>
        <artifactId>flink-cep</artifactId>
        <version>1.17.1</version>
    </dependency>

    <!-- ONNX Runtime for ML -->
    <dependency>
        <groupId>com.microsoft.onnxruntime</groupId>
        <artifactId>onnxruntime</artifactId>
        <version>1.15.0</version>
    </dependency>

    <!-- Avro for Schema Management -->
    <dependency>
        <groupId>org.apache.avro</groupId>
        <artifactId>avro</artifactId>
        <version>1.11.1</version>
    </dependency>
</dependencies>
```

### Docker Compose Setup
```yaml
# docker-compose.yml
version: '3.8'
services:
  jobmanager:
    image: flink:1.17.1-java11
    ports:
      - "8081:8081"
    command: jobmanager
    environment:
      - JOB_MANAGER_RPC_ADDRESS=jobmanager
    volumes:
      - ./checkpoints:/checkpoints
      - ./savepoints:/savepoints
    networks:
      - cardiofit-network
    depends_on:
      - kafka1
      - kafka2
      - kafka3

  taskmanager:
    image: flink:1.17.1-java11
    depends_on:
      - jobmanager
    command: taskmanager
    scale: 3
    environment:
      - JOB_MANAGER_RPC_ADDRESS=jobmanager
      - TASK_MANAGER_NUMBER_OF_TASK_SLOTS=4
    volumes:
      - ./checkpoints:/checkpoints
    networks:
      - cardiofit-network

# External network reference to shared Kafka infrastructure
networks:
  cardiofit-network:
    external: true
    name: cardiofit-kafka_cardiofit-network
```

**Note:** This assumes the Kafka cluster from `/backend/shared-infrastructure/kafka/` is already running. Start it first with:
```bash
cd /backend/shared-infrastructure/kafka/
./start-kafka.sh
```

## Module Implementation

### Module 1: Ingestion & Gateway

This module handles raw event ingestion from multiple sources, performs validation, and routes events to appropriate processing streams.

```java
// Module1_Ingestion.java
package com.cardiofit.flink;

import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.connectors.kafka.FlinkKafkaConsumer;
import org.apache.flink.api.common.serialization.DeserializationSchema;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.functions.ProcessFunction;

public class Module1_Ingestion {

    public static void main(String[] args) throws Exception {
        // Set up the execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure checkpointing
        env.enableCheckpointing(30000); // 30 seconds
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(5000);

        // Kafka consumer configuration using local infrastructure
        Properties kafkaProps = new Properties();
        kafkaProps.setProperty("bootstrap.servers", "kafka1:29092,kafka2:29093,kafka3:29094");
        kafkaProps.setProperty("group.id", "flink-ehr-ingestion");
        kafkaProps.setProperty("enable.auto.commit", "false");
        kafkaProps.setProperty("auto.offset.reset", "latest");

        // Performance optimizations for local cluster
        kafkaProps.setProperty("fetch.min.bytes", "1048576");
        kafkaProps.setProperty("fetch.max.wait.ms", "500");
        kafkaProps.setProperty("max.poll.records", "5000");
        kafkaProps.setProperty("session.timeout.ms", "30000");
        kafkaProps.setProperty("heartbeat.interval.ms", "10000");

        // Create Kafka consumer for existing patient events
        FlinkKafkaConsumer<RawEvent> patientEventConsumer = new FlinkKafkaConsumer<>(
            ExistingTopics.PATIENT_EVENTS.getTopicName(),
            new AvroEventDeserializer(),
            kafkaProps
        );

        // Configure watermark strategy for late events
        rawEventConsumer.assignTimestampsAndWatermarks(
            WatermarkStrategy
                .<RawEvent>forBoundedOutOfOrderness(Duration.ofMinutes(5))
                .withTimestampAssigner((event, timestamp) -> event.getEventTime())
        );

        // Create data stream
        DataStream<RawEvent> rawEvents = env.addSource(rawEventConsumer);

        // Process and validate events
        DataStream<CanonicalEvent> validatedEvents = rawEvents
            .process(new ValidationFunction())
            .name("Event Validation");

        // Route to clean topic
        validatedEvents.addSink(
            new FlinkKafkaProducer<>(
                "ehr-events-clean",
                new CanonicalEventSerializer(),
                kafkaProps,
                FlinkKafkaProducer.Semantic.EXACTLY_ONCE
            )
        ).name("Clean Events Sink");

        // Execute the job
        env.execute("Module 1: EHR Event Ingestion");
    }

    // Validation and canonicalization function
    public static class ValidationFunction extends ProcessFunction<RawEvent, CanonicalEvent> {

        @Override
        public void processElement(RawEvent value, Context ctx, Collector<CanonicalEvent> out) {
            try {
                // Validate event schema
                if (!validateSchema(value)) {
                    // Route to DLQ
                    ctx.output(DLQ_OUTPUT_TAG, value);
                    return;
                }

                // Transform to canonical format
                CanonicalEvent canonical = canonicalize(value);

                // Enrich with metadata
                canonical.setProcessingTime(System.currentTimeMillis());
                canonical.setIngestionNode(getRuntimeContext().getIndexOfThisSubtask());

                out.collect(canonical);

            } catch (Exception e) {
                // Log error and route to DLQ
                LOG.error("Failed to process event: " + value, e);
                ctx.output(DLQ_OUTPUT_TAG, value);
            }
        }

        private boolean validateSchema(RawEvent event) {
            // Implement schema validation logic
            return event.getPatientId() != null &&
                   event.getEventType() != null &&
                   event.getEventTime() > 0;
        }

        private CanonicalEvent canonicalize(RawEvent raw) {
            // Transform to canonical format
            return CanonicalEvent.builder()
                .patientId(raw.getPatientId())
                .eventType(mapEventType(raw.getEventType()))
                .eventTime(raw.getEventTime())
                .payload(normalizePayload(raw.getPayload()))
                .build();
        }
    }
}
```

### Module 2: Context Assembly & Enrichment

This module maintains per-patient state and enriches events with contextual information.

```java
// Module2_ContextAssembly.java
package com.cardiofit.flink;

import org.apache.flink.api.common.state.*;
import org.apache.flink.streaming.api.functions.KeyedProcessFunction;
import org.apache.flink.configuration.Configuration;

public class Module2_ContextAssembly {

    public static class PatientContextAssembler
            extends KeyedProcessFunction<String, CanonicalEvent, EnrichedEvent> {

        // State handles
        private transient ValueState<PatientSnapshot> patientState;
        private transient MapState<String, MedicationEntry> medicationState;
        private transient ListState<VitalReading> vitalHistory;

        @Override
        public void open(Configuration parameters) {
            // Configure state with TTL
            StateTtlConfig ttlConfig = StateTtlConfig
                .newBuilder(Time.days(30))
                .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                .setStateVisibility(StateTtlConfig.StateVisibility.NeverReturnExpired)
                .cleanupInRocksdbCompactFilter(1000)
                .build();

            // Initialize patient snapshot state
            ValueStateDescriptor<PatientSnapshot> patientDescriptor =
                new ValueStateDescriptor<>("patient-snapshot", PatientSnapshot.class);
            patientDescriptor.enableTimeToLive(ttlConfig);
            patientState = getRuntimeContext().getState(patientDescriptor);

            // Initialize medication state with 30-day TTL
            MapStateDescriptor<String, MedicationEntry> medDescriptor =
                new MapStateDescriptor<>("medications", String.class, MedicationEntry.class);
            medDescriptor.enableTimeToLive(ttlConfig);
            medicationState = getRuntimeContext().getMapState(medDescriptor);

            // Initialize vital history with 24-hour TTL
            StateTtlConfig vitalTtl = StateTtlConfig
                .newBuilder(Time.hours(24))
                .setUpdateType(StateTtlConfig.UpdateType.OnCreateAndWrite)
                .build();

            ListStateDescriptor<VitalReading> vitalDescriptor =
                new ListStateDescriptor<>("vital-history", VitalReading.class);
            vitalDescriptor.enableTimeToLive(vitalTtl);
            vitalHistory = getRuntimeContext().getListState(vitalDescriptor);
        }

        @Override
        public void processElement(CanonicalEvent event, Context ctx, Collector<EnrichedEvent> out)
                throws Exception {

            // Get or create patient snapshot
            PatientSnapshot snapshot = patientState.value();
            if (snapshot == null) {
                snapshot = initializePatientSnapshot(event.getPatientId());
            }

            // Update snapshot based on event type
            switch (event.getEventType()) {
                case MEDICATION_ADMINISTERED:
                    updateMedications(event, snapshot);
                    break;

                case VITAL_SIGN:
                    updateVitals(event, snapshot);
                    break;

                case LAB_RESULT:
                    updateLabs(event, snapshot);
                    break;

                case DIAGNOSIS:
                    updateConditions(event, snapshot);
                    break;
            }

            // Calculate clinical scores
            calculateClinicalScores(snapshot);

            // Create enriched event
            EnrichedEvent enriched = EnrichedEvent.builder()
                .originalEvent(event)
                .patientContext(snapshot)
                .activeMedications(getActiveMedications())
                .recentVitals(getRecentVitals())
                .clinicalScores(snapshot.getClinicalScores())
                .currentLocation(snapshot.getCurrentLocation())
                .build();

            // Update state
            patientState.update(snapshot);

            // Emit enriched event
            out.collect(enriched);

            // Set cleanup timer if patient inactive
            if (isPatientInactive(snapshot)) {
                ctx.timerService().registerEventTimeTimer(
                    ctx.timestamp() + Duration.ofDays(7).toMillis()
                );
            }
        }

        @Override
        public void onTimer(long timestamp, OnTimerContext ctx, Collector<EnrichedEvent> out) {
            // Clean up inactive patient state
            patientState.clear();
            medicationState.clear();
            vitalHistory.clear();
        }

        private void calculateClinicalScores(PatientSnapshot snapshot) {
            // Calculate MEWS (Modified Early Warning Score)
            double mews = calculateMEWS(snapshot.getLatestVitals());
            snapshot.getClinicalScores().put("MEWS", mews);

            // Calculate Charlson Comorbidity Index
            double charlson = calculateCharlson(snapshot.getConditions());
            snapshot.getClinicalScores().put("Charlson", charlson);

            // Calculate readmission risk
            double readmissionRisk = calculateReadmissionRisk(snapshot);
            snapshot.getClinicalScores().put("ReadmissionRisk", readmissionRisk);
        }
    }
}
```

### Module 3: Semantic Mesh Integration

This module maintains broadcast state for clinical knowledge distribution.

```java
// Module3_SemanticMesh.java
package com.cardiofit.flink;

import org.apache.flink.api.common.state.BroadcastState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.streaming.api.functions.co.KeyedBroadcastProcessFunction;

public class Module3_SemanticMesh {

    // Broadcast state descriptors
    private static final MapStateDescriptor<String, DrugInteraction> DRUG_INTERACTIONS_DESC =
        new MapStateDescriptor<>("drug-interactions", String.class, DrugInteraction.class);

    private static final MapStateDescriptor<String, ClinicalProtocol> PROTOCOLS_DESC =
        new MapStateDescriptor<>("clinical-protocols", String.class, ClinicalProtocol.class);

    private static final MapStateDescriptor<String, ThresholdRule> THRESHOLDS_DESC =
        new MapStateDescriptor<>("alert-thresholds", String.class, ThresholdRule.class);

    public static class SemanticMeshProcessor
            extends KeyedBroadcastProcessFunction<String, EnrichedEvent, SemanticUpdate, EnrichedEvent> {

        @Override
        public void processElement(EnrichedEvent event, ReadOnlyContext ctx, Collector<EnrichedEvent> out)
                throws Exception {

            // Get broadcast state
            ReadOnlyBroadcastState<String, DrugInteraction> drugInteractions =
                ctx.getBroadcastState(DRUG_INTERACTIONS_DESC);
            ReadOnlyBroadcastState<String, ClinicalProtocol> protocols =
                ctx.getBroadcastState(PROTOCOLS_DESC);
            ReadOnlyBroadcastState<String, ThresholdRule> thresholds =
                ctx.getBroadcastState(THRESHOLDS_DESC);

            // Check for drug interactions
            List<DrugInteractionAlert> interactions = checkDrugInteractions(
                event.getActiveMedications(),
                drugInteractions
            );

            // Check protocol compliance
            List<ProtocolViolation> violations = checkProtocolCompliance(
                event,
                protocols
            );

            // Check threshold violations
            List<ThresholdAlert> thresholdAlerts = checkThresholds(
                event.getRecentVitals(),
                thresholds
            );

            // Enrich event with semantic information
            event.setDrugInteractions(interactions);
            event.setProtocolViolations(violations);
            event.setThresholdAlerts(thresholdAlerts);

            // Add clinical recommendations
            addClinicalRecommendations(event, protocols);

            out.collect(event);
        }

        @Override
        public void processBroadcastElement(SemanticUpdate update, Context ctx, Collector<EnrichedEvent> out)
                throws Exception {

            // Update broadcast state based on update type
            switch (update.getUpdateType()) {
                case DRUG_INTERACTION_UPDATE:
                    BroadcastState<String, DrugInteraction> drugState =
                        ctx.getBroadcastState(DRUG_INTERACTIONS_DESC);
                    updateDrugInteractions(update, drugState);
                    break;

                case PROTOCOL_UPDATE:
                    BroadcastState<String, ClinicalProtocol> protocolState =
                        ctx.getBroadcastState(PROTOCOLS_DESC);
                    updateProtocols(update, protocolState);
                    break;

                case THRESHOLD_UPDATE:
                    BroadcastState<String, ThresholdRule> thresholdState =
                        ctx.getBroadcastState(THRESHOLDS_DESC);
                    updateThresholds(update, thresholdState);
                    break;

                case FULL_REFRESH:
                    refreshAllState(update, ctx);
                    break;
            }
        }

        private List<DrugInteractionAlert> checkDrugInteractions(
                List<MedicationEntry> medications,
                ReadOnlyBroadcastState<String, DrugInteraction> interactions) throws Exception {

            List<DrugInteractionAlert> alerts = new ArrayList<>();

            for (int i = 0; i < medications.size(); i++) {
                for (int j = i + 1; j < medications.size(); j++) {
                    String drug1 = medications.get(i).getDrugCode();
                    String drug2 = medications.get(j).getDrugCode();

                    String interactionKey = generateInteractionKey(drug1, drug2);
                    DrugInteraction interaction = interactions.get(interactionKey);

                    if (interaction != null && interaction.getSeverity() >= MODERATE) {
                        alerts.add(DrugInteractionAlert.builder()
                            .drug1(drug1)
                            .drug2(drug2)
                            .severity(interaction.getSeverity())
                            .description(interaction.getDescription())
                            .recommendation(interaction.getRecommendation())
                            .build());
                    }
                }
            }

            return alerts;
        }
    }
}
```

### Module 4a: CEP Pattern Detection

This module implements Complex Event Processing for clinical pattern detection.

```java
// Module4a_CEPPatterns.java
package com.cardiofit.flink;

import org.apache.flink.cep.CEP;
import org.apache.flink.cep.PatternStream;
import org.apache.flink.cep.pattern.Pattern;
import org.apache.flink.cep.pattern.conditions.SimpleCondition;
import org.apache.flink.streaming.api.windowing.time.Time;

public class Module4a_CEPPatterns {

    // Sepsis detection pattern
    public static Pattern<EnrichedEvent, ?> createSepsisPattern() {
        return Pattern.<EnrichedEvent>begin("fever")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN &&
                           event.getVital("temperature") > 38.0;
                }
            })
            .followedBy("tachycardia")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN &&
                           event.getVital("heart_rate") > 100;
                }
            })
            .followedBy("hypotension")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.VITAL_SIGN &&
                           event.getVital("systolic_bp") < 90;
                }
            })
            .followedBy("elevated_lactate")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.LAB_RESULT &&
                           event.getLabValue("lactate") > 2.0;
                }
            })
            .within(Time.hours(2));
    }

    // Medication adherence pattern
    public static Pattern<EnrichedEvent, ?> createMedicationAdherencePattern() {
        return Pattern.<EnrichedEvent>begin("medication_due")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.MEDICATION_SCHEDULED;
                }
            })
            .notFollowedBy("administration")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.MEDICATION_ADMINISTERED;
                }
            })
            .within(Time.hours(6));
    }

    // Drug-lab monitoring pattern
    public static Pattern<EnrichedEvent, ?> createDrugLabMonitoringPattern() {
        return Pattern.<EnrichedEvent>begin("ace_inhibitor_started")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.MEDICATION_STARTED &&
                           isACEInhibitor(event.getMedicationCode());
                }
            })
            .notFollowedBy("creatinine_check")
            .where(new SimpleCondition<EnrichedEvent>() {
                @Override
                public boolean filter(EnrichedEvent event) {
                    return event.getEventType() == EventType.LAB_RESULT &&
                           event.getLabType().equals("creatinine");
                }
            })
            .within(Time.hours(48));
    }

    public static class ClinicalPatternProcessor {

        public static void setupPatternDetection(DataStream<EnrichedEvent> enrichedStream) {

            // Sepsis detection
            PatternStream<EnrichedEvent> sepsisPatterns = CEP.pattern(
                enrichedStream.keyBy(EnrichedEvent::getPatientId),
                createSepsisPattern()
            );

            sepsisPatterns.select(new PatternSelectFunction<EnrichedEvent, ClinicalAlert>() {
                @Override
                public ClinicalAlert select(Map<String, List<EnrichedEvent>> pattern) {
                    EnrichedEvent fever = pattern.get("fever").get(0);
                    EnrichedEvent tachycardia = pattern.get("tachycardia").get(0);
                    EnrichedEvent hypotension = pattern.get("hypotension").get(0);
                    EnrichedEvent lactate = pattern.get("elevated_lactate").get(0);

                    return ClinicalAlert.builder()
                        .alertType("SEPSIS_ALERT")
                        .severity(AlertSeverity.CRITICAL)
                        .patientId(fever.getPatientId())
                        .message("Sepsis criteria met: fever + tachycardia + hypotension + elevated lactate")
                        .evidence(Arrays.asList(fever, tachycardia, hypotension, lactate))
                        .recommendation("Initiate sepsis protocol immediately")
                        .timestamp(System.currentTimeMillis())
                        .build();
                }
            }).addSink(new AlertSink("critical-alerts"));

            // Medication adherence detection
            PatternStream<EnrichedEvent> adherencePatterns = CEP.pattern(
                enrichedStream.keyBy(EnrichedEvent::getPatientId),
                createMedicationAdherencePattern()
            );

            adherencePatterns.select(new PatternSelectFunction<EnrichedEvent, ClinicalAlert>() {
                @Override
                public ClinicalAlert select(Map<String, List<EnrichedEvent>> pattern) {
                    EnrichedEvent medicationDue = pattern.get("medication_due").get(0);

                    return ClinicalAlert.builder()
                        .alertType("MEDICATION_MISSED")
                        .severity(AlertSeverity.HIGH)
                        .patientId(medicationDue.getPatientId())
                        .message("Medication not administered within 6-hour window")
                        .medication(medicationDue.getMedicationName())
                        .dueTime(medicationDue.getScheduledTime())
                        .build();
                }
            }).addSink(new AlertSink("adherence-alerts"));
        }
    }
}
```

### Module 4b: Windowed Analytics

This module performs windowed aggregations for trend analysis.

```java
// Module4b_WindowedAnalytics.java
package com.cardiofit.flink;

import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.assigners.TumblingEventTimeWindows;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;

public class Module4b_WindowedAnalytics {

    public static class VitalTrendAnalyzer
            implements WindowFunction<EnrichedEvent, TrendAlert, String, TimeWindow> {

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<EnrichedEvent> events, Collector<TrendAlert> out) {

            List<Double> heartRates = new ArrayList<>();
            List<Double> bloodPressures = new ArrayList<>();
            List<Double> oxygenSaturations = new ArrayList<>();
            List<Double> temperatures = new ArrayList<>();

            // Collect vital signs from window
            for (EnrichedEvent event : events) {
                if (event.getEventType() == EventType.VITAL_SIGN) {
                    heartRates.add(event.getVital("heart_rate"));
                    bloodPressures.add(event.getVital("systolic_bp"));
                    oxygenSaturations.add(event.getVital("spo2"));
                    temperatures.add(event.getVital("temperature"));
                }
            }

            // Calculate trends using linear regression
            TrendAnalysis hrTrend = calculateTrend(heartRates, "heart_rate");
            TrendAnalysis bpTrend = calculateTrend(bloodPressures, "blood_pressure");
            TrendAnalysis o2Trend = calculateTrend(oxygenSaturations, "oxygen");

            // Detect deterioration patterns
            if (isDeteriorating(hrTrend, bpTrend, o2Trend)) {
                TrendAlert alert = TrendAlert.builder()
                    .patientId(patientId)
                    .alertType("VITAL_DETERIORATION")
                    .severity(AlertSeverity.HIGH)
                    .heartRateTrend(hrTrend)
                    .bloodPressureTrend(bpTrend)
                    .oxygenTrend(o2Trend)
                    .windowStart(window.getStart())
                    .windowEnd(window.getEnd())
                    .recommendation("Patient showing signs of clinical deterioration")
                    .build();

                out.collect(alert);
            }

            // Detect improvement patterns
            if (isImproving(hrTrend, bpTrend, o2Trend)) {
                TrendAlert alert = TrendAlert.builder()
                    .patientId(patientId)
                    .alertType("VITAL_IMPROVEMENT")
                    .severity(AlertSeverity.INFO)
                    .heartRateTrend(hrTrend)
                    .bloodPressureTrend(bpTrend)
                    .oxygenTrend(o2Trend)
                    .windowStart(window.getStart())
                    .windowEnd(window.getEnd())
                    .recommendation("Patient vitals showing improvement")
                    .build();

                out.collect(alert);
            }
        }

        private TrendAnalysis calculateTrend(List<Double> values, String metric) {
            if (values.size() < 2) {
                return TrendAnalysis.insufficient(metric);
            }

            // Simple linear regression
            double[] x = new double[values.size()];
            double[] y = new double[values.size()];

            for (int i = 0; i < values.size(); i++) {
                x[i] = i;
                y[i] = values.get(i);
            }

            double xMean = Arrays.stream(x).average().orElse(0);
            double yMean = Arrays.stream(y).average().orElse(0);

            double numerator = 0;
            double denominator = 0;

            for (int i = 0; i < values.size(); i++) {
                numerator += (x[i] - xMean) * (y[i] - yMean);
                denominator += Math.pow(x[i] - xMean, 2);
            }

            double slope = denominator == 0 ? 0 : numerator / denominator;
            double intercept = yMean - slope * xMean;

            // Calculate R-squared for confidence
            double ssTotal = 0;
            double ssResidual = 0;

            for (int i = 0; i < values.size(); i++) {
                double predicted = slope * x[i] + intercept;
                ssTotal += Math.pow(y[i] - yMean, 2);
                ssResidual += Math.pow(y[i] - predicted, 2);
            }

            double rSquared = ssTotal == 0 ? 0 : 1 - (ssResidual / ssTotal);

            return TrendAnalysis.builder()
                .metric(metric)
                .slope(slope)
                .intercept(intercept)
                .rSquared(rSquared)
                .direction(slope > 0.1 ? "INCREASING" : slope < -0.1 ? "DECREASING" : "STABLE")
                .confidence(rSquared > 0.7 ? "HIGH" : rSquared > 0.4 ? "MEDIUM" : "LOW")
                .dataPoints(values.size())
                .lastValue(values.get(values.size() - 1))
                .build();
        }
    }

    public static void setupWindowedAnalytics(DataStream<EnrichedEvent> enrichedStream) {

        // Sliding window for vital trends (1-hour window, 5-minute slide)
        enrichedStream
            .keyBy(EnrichedEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Time.hours(1), Time.minutes(5)))
            .apply(new VitalTrendAnalyzer())
            .addSink(new TrendAlertSink("vital-trends"));

        // Tumbling window for daily aggregates
        enrichedStream
            .keyBy(EnrichedEvent::getPatientId)
            .window(TumblingEventTimeWindows.of(Time.days(1)))
            .aggregate(new DailyAggregateFunction())
            .addSink(new AggregateSink("daily-aggregates"));

        // Session window for care episodes (4-hour inactivity gap)
        enrichedStream
            .keyBy(EnrichedEvent::getPatientId)
            .window(EventTimeSessionWindows.withGap(Time.hours(4)))
            .apply(new CareEpisodeAnalyzer())
            .addSink(new EpisodeSink("care-episodes"));
    }
}
```

### Module 5: ML Inference

This module handles machine learning model inference for risk scoring.

```java
// Module5_MLInference.java
package com.cardiofit.flink;

import ai.onnxruntime.*;
import org.apache.flink.streaming.api.functions.async.ResultFuture;
import org.apache.flink.streaming.api.functions.async.RichAsyncFunction;

public class Module5_MLInference {

    // Embedded ONNX model inference
    public static class SepsisRiskScorer extends RichMapFunction<EnrichedEvent, ScoredEvent> {

        private transient OrtSession sepsisModel;
        private transient OrtEnvironment env;

        @Override
        public void open(Configuration parameters) throws Exception {
            // Initialize ONNX Runtime
            env = OrtEnvironment.getEnvironment();

            // Load model from resources
            String modelPath = getRuntimeContext()
                .getDistributedCache()
                .getFile("sepsis-model.onnx")
                .getAbsolutePath();

            OrtSession.SessionOptions options = new OrtSession.SessionOptions();
            options.setOptimizationLevel(OrtSession.SessionOptions.OptLevel.ALL_OPT);

            sepsisModel = env.createSession(modelPath, options);
        }

        @Override
        public ScoredEvent map(EnrichedEvent event) throws Exception {
            // Extract features for model
            float[] features = extractFeatures(event);

            // Create ONNX tensor
            OnnxTensor inputTensor = OnnxTensor.createTensor(env, new float[][] {features});

            // Run inference
            Map<String, OnnxTensor> inputs = Collections.singletonMap("input", inputTensor);
            OrtSession.Result result = sepsisModel.run(inputs);

            // Extract prediction
            float[][] output = (float[][]) result.get(0).getValue();
            float sepsisRisk = output[0][0];

            // Get feature importance
            float[] importance = null;
            if (result.size() > 1) {
                importance = (float[]) result.get(1).getValue();
            }

            // Create scored event
            return ScoredEvent.builder()
                .originalEvent(event)
                .scoreType("SEPSIS_RISK")
                .score(sepsisRisk)
                .confidence(calculateConfidence(output[0]))
                .featureImportance(importance)
                .modelVersion("1.2.0")
                .timestamp(System.currentTimeMillis())
                .build();
        }

        private float[] extractFeatures(EnrichedEvent event) {
            // Extract features for sepsis model
            PatientSnapshot context = event.getPatientContext();

            return new float[] {
                // Vital signs
                (float) context.getLatestVitals().getHeartRate(),
                (float) context.getLatestVitals().getTemperature(),
                (float) context.getLatestVitals().getRespiratoryRate(),
                (float) context.getLatestVitals().getSystolicBP(),
                (float) context.getLatestVitals().getDiastolicBP(),
                (float) context.getLatestVitals().getSpO2(),

                // Lab values
                (float) context.getLatestLabs().getOrDefault("wbc", 0.0),
                (float) context.getLatestLabs().getOrDefault("creatinine", 0.0),
                (float) context.getLatestLabs().getOrDefault("lactate", 0.0),
                (float) context.getLatestLabs().getOrDefault("platelet", 0.0),

                // Demographics
                (float) context.getAge(),
                context.getGender().equals("M") ? 1.0f : 0.0f,

                // Clinical scores
                (float) context.getClinicalScores().getOrDefault("MEWS", 0.0),
                (float) context.getClinicalScores().getOrDefault("Charlson", 0.0),

                // Time features
                (float) context.getHoursSinceAdmission(),
                (float) context.getDayOfWeek()
            };
        }

        @Override
        public void close() throws Exception {
            if (sepsisModel != null) {
                sepsisModel.close();
            }
            if (env != null) {
                env.close();
            }
        }
    }

    // Async external model inference
    public static class AsyncModelInference extends RichAsyncFunction<EnrichedEvent, ScoredEvent> {

        private transient HttpClient httpClient;
        private String modelEndpoint;

        @Override
        public void open(Configuration parameters) {
            this.modelEndpoint = parameters.getString("ml.endpoint", "http://ml-service:8080/predict");
            this.httpClient = HttpClient.newBuilder()
                .connectTimeout(Duration.ofMillis(500))
                .executor(Executors.newFixedThreadPool(10))
                .build();
        }

        @Override
        public void asyncInvoke(EnrichedEvent input, ResultFuture<ScoredEvent> resultFuture) {
            // Create inference request
            InferenceRequest request = InferenceRequest.builder()
                .patientId(input.getPatientId())
                .features(extractFeatures(input))
                .modelType("readmission_risk")
                .build();

            // Make async HTTP call
            HttpRequest httpRequest = HttpRequest.newBuilder()
                .uri(URI.create(modelEndpoint))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(toJson(request)))
                .timeout(Duration.ofMillis(1000))
                .build();

            CompletableFuture<HttpResponse<String>> future =
                httpClient.sendAsync(httpRequest, HttpResponse.BodyHandlers.ofString());

            // Handle response
            future.whenComplete((response, throwable) -> {
                if (throwable != null) {
                    // Use default score on error
                    ScoredEvent defaultScore = createDefaultScore(input);
                    resultFuture.complete(Collections.singleton(defaultScore));
                } else {
                    try {
                        ModelPrediction prediction = parseResponse(response.body());
                        ScoredEvent scored = ScoredEvent.builder()
                            .originalEvent(input)
                            .scoreType("READMISSION_RISK")
                            .score(prediction.getScore())
                            .confidence(prediction.getConfidence())
                            .explanation(prediction.getExplanation())
                            .modelVersion(prediction.getModelVersion())
                            .build();

                        resultFuture.complete(Collections.singleton(scored));
                    } catch (Exception e) {
                        resultFuture.completeExceptionally(e);
                    }
                }
            });
        }

        @Override
        public void timeout(EnrichedEvent input, ResultFuture<ScoredEvent> resultFuture) {
            // Handle timeout with default score
            ScoredEvent defaultScore = createDefaultScore(input);
            resultFuture.complete(Collections.singleton(defaultScore));
        }
    }
}
```

### Module 6: Egress & Multi-Sink Routing

This module handles alert routing and multi-sink distribution.

```java
// Module6_EgressRouting.java
package com.cardiofit.flink;

import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.util.OutputTag;

public class Module6_EgressRouting {

    // Output tags for side outputs
    private static final OutputTag<ClinicalAlert> CRITICAL_TAG =
        new OutputTag<ClinicalAlert>("critical-alerts"){};
    private static final OutputTag<ClinicalAlert> URGENT_TAG =
        new OutputTag<ClinicalAlert>("urgent-alerts"){};
    private static final OutputTag<ClinicalAlert> ROUTINE_TAG =
        new OutputTag<ClinicalAlert>("routine-alerts"){};
    private static final OutputTag<ClinicalAlert> INFO_TAG =
        new OutputTag<ClinicalAlert>("info-alerts"){};

    public static class AlertRouter extends ProcessFunction<ScoredEvent, ClinicalAlert> {

        @Override
        public void processElement(ScoredEvent value, Context ctx, Collector<ClinicalAlert> out) {

            // Generate alerts based on scores and patterns
            List<ClinicalAlert> alerts = generateAlerts(value);

            // Route alerts by priority
            for (ClinicalAlert alert : alerts) {
                switch (alert.getSeverity()) {
                    case CRITICAL:
                        // Route to critical stream (pager, <1 min SLA)
                        ctx.output(CRITICAL_TAG, alert);
                        break;

                    case HIGH:
                        // Route to urgent stream (nurse station, <15 min SLA)
                        ctx.output(URGENT_TAG, alert);
                        break;

                    case MEDIUM:
                        // Route to routine stream (EHR inbox, <1 hour SLA)
                        ctx.output(ROUTINE_TAG, alert);
                        break;

                    case LOW:
                    case INFO:
                        // Route to info stream (analytics)
                        ctx.output(INFO_TAG, alert);
                        break;
                }

                // Also emit to main output for audit
                out.collect(alert);
            }
        }

        private List<ClinicalAlert> generateAlerts(ScoredEvent event) {
            List<ClinicalAlert> alerts = new ArrayList<>();

            // Check sepsis risk score
            if (event.getScoreType().equals("SEPSIS_RISK") && event.getScore() > 0.7) {
                alerts.add(ClinicalAlert.builder()
                    .alertType("HIGH_SEPSIS_RISK")
                    .severity(AlertSeverity.CRITICAL)
                    .patientId(event.getPatientId())
                    .score(event.getScore())
                    .message("High sepsis risk detected: " + (event.getScore() * 100) + "%")
                    .recommendation("Initiate sepsis screening protocol")
                    .build());
            }

            // Check readmission risk
            if (event.getScoreType().equals("READMISSION_RISK") && event.getScore() > 0.6) {
                alerts.add(ClinicalAlert.builder()
                    .alertType("HIGH_READMISSION_RISK")
                    .severity(AlertSeverity.MEDIUM)
                    .patientId(event.getPatientId())
                    .score(event.getScore())
                    .message("High readmission risk: " + (event.getScore() * 100) + "%")
                    .recommendation("Consider discharge planning intervention")
                    .build());
            }

            // Check for drug interactions
            if (event.getDrugInteractions() != null && !event.getDrugInteractions().isEmpty()) {
                for (DrugInteractionAlert interaction : event.getDrugInteractions()) {
                    alerts.add(ClinicalAlert.builder()
                        .alertType("DRUG_INTERACTION")
                        .severity(mapInteractionSeverity(interaction.getSeverity()))
                        .patientId(event.getPatientId())
                        .message(interaction.getDescription())
                        .recommendation(interaction.getRecommendation())
                        .build());
                }
            }

            return alerts;
        }
    }

    public static void setupEgressRouting(DataStream<ScoredEvent> scoredStream) {

        // Process and route alerts
        SingleOutputStreamOperator<ClinicalAlert> alertStream = scoredStream
            .process(new AlertRouter())
            .name("Alert Router");

        // Critical alerts → Pager system
        alertStream.getSideOutput(CRITICAL_TAG)
            .addSink(new PagerSink())
            .name("Critical Alert Pager");

        // Urgent alerts → Nurse station displays
        alertStream.getSideOutput(URGENT_TAG)
            .addSink(new NurseStationSink())
            .name("Urgent Alert Display");

        // Routine alerts → EHR inbox
        alertStream.getSideOutput(ROUTINE_TAG)
            .addSink(new EHRInboxSink())
            .name("Routine Alert Inbox");

        // Info alerts → Analytics warehouse
        alertStream.getSideOutput(INFO_TAG)
            .addSink(new ClickHouseSink())
            .name("Analytics Sink");

        // All alerts → Audit trail
        alertStream
            .addSink(new AuditTrailSink())
            .name("Audit Trail");

        // FHIR store sink for clinical data
        scoredStream
            .map(new FHIRTransformer())
            .addSink(new FHIRStoreSink())
            .name("FHIR Store");

        // Neo4j sink for graph relationships
        scoredStream
            .map(new GraphTransformer())
            .addSink(new Neo4jSink())
            .name("Graph Database");
    }
}
```

## Integration with CardioFit

### Using Local Kafka Infrastructure

The Flink EHR Intelligence Engine integrates with CardioFit's local Kafka infrastructure located in `/backend/shared-infrastructure/kafka/`. This provides a 3-broker cluster with Zookeeper, Schema Registry, and monitoring tools.

```java
// KafkaConfigLoader.java - Load configuration for local cluster
public class KafkaConfigLoader {

    // Local Kafka cluster endpoints (from docker-compose.yml)
    private static final String BOOTSTRAP_SERVERS = "kafka1:29092,kafka2:29093,kafka3:29094";
    private static final String SCHEMA_REGISTRY_URL = "http://schema-registry:8081";

    public static Properties getConsumerConfig(String groupId) {
        Properties props = new Properties();
        props.setProperty("bootstrap.servers", BOOTSTRAP_SERVERS);
        props.setProperty("group.id", groupId);
        props.setProperty("enable.auto.commit", "false");
        props.setProperty("auto.offset.reset", "latest");

        // Performance optimizations for local cluster
        props.setProperty("fetch.min.bytes", "1048576");
        props.setProperty("fetch.max.wait.ms", "500");
        props.setProperty("max.poll.records", "5000");
        props.setProperty("session.timeout.ms", "30000");
        props.setProperty("heartbeat.interval.ms", "10000");

        // Avro deserializer with local Schema Registry
        props.setProperty("key.deserializer",
            "org.apache.kafka.common.serialization.StringDeserializer");
        props.setProperty("value.deserializer",
            "io.confluent.kafka.serializers.KafkaAvroDeserializer");
        props.setProperty("schema.registry.url", SCHEMA_REGISTRY_URL);
        props.setProperty("specific.avro.reader", "true");

        return props;
    }

    public static Properties getProducerConfig() {
        Properties props = new Properties();
        props.setProperty("bootstrap.servers", BOOTSTRAP_SERVERS);

        // Producer optimizations for high throughput
        props.setProperty("compression.type", "snappy");  // Matches cluster default
        props.setProperty("batch.size", "32768");
        props.setProperty("linger.ms", "100");
        props.setProperty("acks", "all");
        props.setProperty("enable.idempotence", "true");
        props.setProperty("retries", "2147483647");
        props.setProperty("max.in.flight.requests.per.connection", "5");

        // Avro serializer
        props.setProperty("key.serializer",
            "org.apache.kafka.common.serialization.StringSerializer");
        props.setProperty("value.serializer",
            "io.confluent.kafka.serializers.KafkaAvroSerializer");
        props.setProperty("schema.registry.url", SCHEMA_REGISTRY_URL);

        return props;
    }

    // External access for development (when running outside Docker)
    public static Properties getExternalConsumerConfig(String groupId) {
        Properties props = getConsumerConfig(groupId);
        // Override with localhost addresses for external access
        props.setProperty("bootstrap.servers", "localhost:9092,localhost:9093,localhost:9094");
        props.setProperty("schema.registry.url", "http://localhost:8081");
        return props;
    }
}
```

### Existing Kafka Topics

The local Kafka infrastructure provides 68 topics across 11 categories. Key topics for Flink integration:

```java
// Existing topics from shared-infrastructure/kafka/KAFKA_TOPICS_REFERENCE.md
public enum ExistingTopics {
    // Clinical Events (input sources)
    PATIENT_EVENTS("patient-events.v1"),
    MEDICATION_EVENTS("medication-events.v1"),
    OBSERVATION_EVENTS("observation-events.v1"),
    SAFETY_EVENTS("safety-events.v1"),
    VITAL_SIGNS_EVENTS("vital-signs-events.v1"),
    LAB_RESULT_EVENTS("lab-result-events.v1"),
    ENCOUNTER_EVENTS("encounter-events.v1"),
    DIAGNOSTIC_EVENTS("diagnostic-events.v1"),
    PROCEDURE_EVENTS("procedure-events.v1"),

    // Device Data (real-time inputs)
    RAW_DEVICE_DATA("raw-device-data.v1"),
    VALIDATED_DEVICE_DATA("validated-device-data.v1"),
    WAVEFORM_DATA("waveform-data.v1"),
    DEVICE_TELEMETRY("device-telemetry.v1"),

    // Runtime Layer (Flink outputs)
    ENRICHED_PATIENT_EVENTS("enriched-patient-events.v1"),
    CLINICAL_PATTERNS("clinical-patterns.v1"),
    PATHWAY_ADHERENCE_EVENTS("pathway-adherence-events.v1"),
    SEMANTIC_MESH_UPDATES("semantic-mesh-updates.v1"),
    PATIENT_CONTEXT_SNAPSHOTS("patient-context-snapshots.v1"),

    // Knowledge Base CDC (broadcast state inputs)
    KB3_CLINICAL_PROTOCOLS("kb3.clinical_protocols.changes"),
    KB4_DRUG_CALCULATIONS("kb4.drug_calculations.changes"),
    KB5_DRUG_INTERACTIONS("kb5.drug_interactions.changes"),
    KB6_VALIDATION_RULES("kb6.validation_rules.changes"),
    KB7_TERMINOLOGY("kb7.terminology.changes"),

    // Evidence Management (outputs)
    AUDIT_EVENTS("audit-events.v1"),
    CLINICAL_REASONING_EVENTS("clinical-reasoning-events.v1"),
    INFERENCE_RESULTS("inference-results.v1");

    private final String topicName;

    ExistingTopics(String topicName) {
        this.topicName = topicName;
    }

    public String getTopicName() {
        return topicName;
    }
}
```

### Topic Configuration Reference

All topics use the following default configuration (from .env file):
- **Replication Factor**: 3 (high availability)
- **Min In-Sync Replicas**: 2 (consistency)
- **Compression**: snappy (balanced performance)
- **Auto Create Topics**: false (controlled creation)
- **Retention**: 3-365 days depending on category

### Avro Schema Integration

Use the existing Avro schemas defined in `/backend/services/shared/kafka/avro_schemas.py`:

```java
// AvroEventDeserializer.java
public class AvroEventDeserializer implements DeserializationSchema<EnrichedEvent> {

    private transient KafkaAvroDeserializer deserializer;

    @Override
    public void open(InitializationContext context) {
        Map<String, Object> config = new HashMap<>();
        config.put("schema.registry.url", getSchemaRegistryUrl());
        config.put("specific.avro.reader", true);

        deserializer = new KafkaAvroDeserializer();
        deserializer.configure(config, false);
    }

    @Override
    public EnrichedEvent deserialize(byte[] message) throws IOException {
        // Deserialize using EventEnvelope schema
        GenericRecord record = (GenericRecord) deserializer.deserialize(
            "fhir-resource-events", message);

        return EnrichedEvent.builder()
            .id(record.get("id").toString())
            .source(record.get("source").toString())
            .type(record.get("type").toString())
            .subject(record.get("subject").toString())
            .time(parseTimestamp(record.get("time").toString()))
            .data(parseJsonData(record.get("data").toString()))
            .correlationId(record.get("correlation_id"))
            .build();
    }
}
```

### Connecting to Existing Services

```java
// CardioFitIntegration.java
public class CardioFitIntegration {

    // Connect to existing Python microservices
    public static class PatientServiceConnector {
        private static final String PATIENT_SERVICE_URL = "http://localhost:8003";

        public PatientData fetchPatientData(String patientId) {
            // REST call to patient-service
            HttpClient client = HttpClient.newHttpClient();
            HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(PATIENT_SERVICE_URL + "/patients/" + patientId))
                .header("Authorization", "Bearer " + getToken())
                .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
            return parsePatientData(response.body());
        }
    }

    // Connect to Knowledge Base services
    public static class KnowledgeBaseConnector {
        private static final String KB_DRUG_RULES_URL = "http://localhost:8081";
        private static final String KB_GUIDELINES_URL = "http://localhost:8084";

        public DrugRules fetchDrugRules() {
            // Fetch TOML-based drug rules
            return fetchFromKnowledgeBase(KB_DRUG_RULES_URL + "/rules/drugs");
        }

        public ClinicalGuidelines fetchGuidelines() {
            // Fetch clinical guidelines
            return fetchFromKnowledgeBase(KB_GUIDELINES_URL + "/guidelines");
        }
    }

    // Connect to Safety Gateway via gRPC
    public static class SafetyGatewayConnector {
        private final ManagedChannel channel;
        private final SafetyServiceGrpc.SafetyServiceBlockingStub stub;

        public SafetyGatewayConnector() {
            channel = ManagedChannelBuilder.forAddress("localhost", 8090)
                .usePlaintext()
                .build();
            stub = SafetyServiceGrpc.newBlockingStub(channel);
        }

        public ValidationResult validateAlert(ClinicalAlert alert) {
            ValidationRequest request = ValidationRequest.newBuilder()
                .setPatientId(alert.getPatientId())
                .setAlertType(alert.getAlertType())
                .setSeverity(alert.getSeverity().toString())
                .build();

            ValidationResponse response = stub.validateAlert(request);
            return ValidationResult.fromProto(response);
        }
    }

    // Connect to Clinical Reasoning Service
    public static class ClinicalReasoningConnector {
        private static final String REASONING_SERVICE_URL = "http://localhost:8015";

        public ReasoningResult analyzePattern(List<EnrichedEvent> events) {
            // Send events to Neo4j-based reasoning service
            HttpClient client = HttpClient.newHttpClient();
            HttpRequest request = HttpRequest.newBuilder()
                .uri(URI.create(REASONING_SERVICE_URL + "/analyze"))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(toJson(events)))
                .build();

            HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
            return parseReasoningResult(response.body());
        }
    }
}
```

### Apollo Federation Integration

```javascript
// flink-subgraph.js
const { ApolloServer, gql } = require('apollo-server');
const { buildSubgraphSchema } = require('@apollo/federation');

const typeDefs = gql`
  type Query {
    patientSnapshot(patientId: ID!): PatientSnapshot
    clinicalAlerts(patientId: ID, severity: AlertSeverity): [ClinicalAlert!]!
    vitalTrends(patientId: ID!, window: TimeWindow!): VitalTrends
  }

  type Subscription {
    alertStream(severity: AlertSeverity): ClinicalAlert!
    patientEvents(patientId: ID!): EnrichedEvent!
  }

  type PatientSnapshot {
    patientId: ID!
    activeMedications: [Medication!]!
    recentVitals: VitalSigns!
    clinicalScores: ClinicalScores!
    currentLocation: Location!
    lastUpdated: DateTime!
  }

  type ClinicalAlert {
    id: ID!
    alertType: String!
    severity: AlertSeverity!
    patientId: ID!
    message: String!
    recommendation: String!
    timestamp: DateTime!
  }

  enum AlertSeverity {
    CRITICAL
    HIGH
    MEDIUM
    LOW
    INFO
  }
`;

const resolvers = {
  Query: {
    patientSnapshot: async (_, { patientId }) => {
      // Query Flink state via REST API
      return await flinkClient.getPatientSnapshot(patientId);
    },
    clinicalAlerts: async (_, { patientId, severity }) => {
      // Query alert history from ClickHouse
      return await clickHouseClient.queryAlerts(patientId, severity);
    },
    vitalTrends: async (_, { patientId, window }) => {
      // Query windowed analytics results
      return await flinkClient.getVitalTrends(patientId, window);
    }
  },
  Subscription: {
    alertStream: {
      subscribe: () => kafkaAlertSubscription()
    },
    patientEvents: {
      subscribe: (_, { patientId }) => kafkaEventSubscription(patientId)
    }
  }
};

const server = new ApolloServer({
  schema: buildSubgraphSchema([{ typeDefs, resolvers }])
});

server.listen(4001).then(({ url }) => {
  console.log(`Flink subgraph ready at ${url}`);
});
```

## Deployment

### Environment Variables Configuration

```yaml
# configmap.yaml - Kafka configuration for Flink jobs
apiVersion: v1
kind: ConfigMap
metadata:
  name: flink-kafka-config
  namespace: flink-ehr
data:
  KAFKA_BOOTSTRAP_SERVERS: "kafka1:29092,kafka2:29093,kafka3:29094"
  KAFKA_EXTERNAL_BOOTSTRAP_SERVERS: "localhost:9092,localhost:9093,localhost:9094"
  SCHEMA_REGISTRY_URL: "http://schema-registry:8081"
  KAFKA_COMPRESSION_TYPE: "snappy"
  KAFKA_REPLICATION_FACTOR: "3"
  KAFKA_MIN_INSYNC_REPLICAS: "2"
```

### Kubernetes Deployment

```yaml
# flink-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flink-jobmanager
  namespace: flink-ehr
spec:
  replicas: 1
  selector:
    matchLabels:
      app: flink
      component: jobmanager
  template:
    metadata:
      labels:
        app: flink
        component: jobmanager
    spec:
      containers:
      - name: jobmanager
        image: cardiofit/flink-ehr:latest
        command: ["jobmanager.sh"]
        ports:
        - containerPort: 6123
          name: rpc
        - containerPort: 8081
          name: webui
        env:
        - name: JOB_MANAGER_RPC_ADDRESS
          value: flink-jobmanager
        volumeMounts:
        - name: flink-config
          mountPath: /opt/flink/conf
        - name: checkpoints
          mountPath: /checkpoints
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
      volumes:
      - name: flink-config
        configMap:
          name: flink-config
      - name: checkpoints
        persistentVolumeClaim:
          claimName: flink-checkpoints-pvc
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flink-taskmanager
  namespace: flink-ehr
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flink
      component: taskmanager
  template:
    metadata:
      labels:
        app: flink
        component: taskmanager
    spec:
      containers:
      - name: taskmanager
        image: cardiofit/flink-ehr:latest
        command: ["taskmanager.sh"]
        env:
        - name: JOB_MANAGER_RPC_ADDRESS
          value: flink-jobmanager
        - name: TASK_MANAGER_NUMBER_OF_TASK_SLOTS
          value: "4"
        volumeMounts:
        - name: flink-config
          mountPath: /opt/flink/conf
        - name: rocksdb
          mountPath: /tmp/rocksdb
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
      volumes:
      - name: flink-config
        configMap:
          name: flink-config
      - name: rocksdb
        emptyDir:
          sizeLimit: 10Gi
```

### Configuration Files

```yaml
# flink-conf.yaml
jobmanager.rpc.address: flink-jobmanager
jobmanager.rpc.port: 6123
jobmanager.memory.process.size: 4g

taskmanager.memory.process.size: 8g
taskmanager.numberOfTaskSlots: 4

parallelism.default: 32

# Checkpointing
execution.checkpointing.interval: 60000
execution.checkpointing.min-pause: 5000
execution.checkpointing.timeout: 600000
execution.checkpointing.max-concurrent-checkpoints: 1
execution.checkpointing.externalized-checkpoint-retention: RETAIN_ON_CANCELLATION

# State Backend
state.backend: rocksdb
state.checkpoints.dir: s3://cardiofit-flink/checkpoints
state.savepoints.dir: s3://cardiofit-flink/savepoints
state.backend.rocksdb.memory.managed: true
state.backend.rocksdb.memory.high-prio-pool-ratio: 0.1
state.backend.rocksdb.block.cache-size: 256m

# Restart Strategy
restart-strategy: failure-rate
restart-strategy.failure-rate.max-failures-per-interval: 3
restart-strategy.failure-rate.failure-rate-interval: 5 min
restart-strategy.failure-rate.delay: 10 s

# Metrics
metrics.reporters: prometheus
metrics.reporter.prometheus.class: org.apache.flink.metrics.prometheus.PrometheusReporter
metrics.reporter.prometheus.port: 9249
```

## Testing

### Unit Testing

```java
// Module1_IngestionTest.java
public class Module1_IngestionTest {

    @Test
    public void testEventValidation() throws Exception {
        // Create test harness
        ValidationFunction function = new ValidationFunction();
        OneInputStreamOperatorTestHarness<RawEvent, CanonicalEvent> harness =
            new OneInputStreamOperatorTestHarness<>(
                new ProcessOperator<>(function)
            );

        harness.open();

        // Test valid event
        RawEvent validEvent = RawEvent.builder()
            .patientId("P123")
            .eventType("VITAL_SIGN")
            .eventTime(System.currentTimeMillis())
            .payload(Map.of("heart_rate", 75))
            .build();

        harness.processElement(new StreamRecord<>(validEvent));

        // Assert output
        assertEquals(1, harness.extractOutputValues().size());
        CanonicalEvent output = harness.extractOutputValues().get(0);
        assertEquals("P123", output.getPatientId());

        // Test invalid event
        RawEvent invalidEvent = RawEvent.builder()
            .eventType("VITAL_SIGN")  // Missing patientId
            .build();

        harness.processElement(new StreamRecord<>(invalidEvent));

        // Assert routed to DLQ
        assertEquals(1, harness.getSideOutput(DLQ_OUTPUT_TAG).size());

        harness.close();
    }
}
```

### Integration Testing

```java
// EndToEndIntegrationTest.java
public class EndToEndIntegrationTest {

    @ClassRule
    public static MiniClusterResource flinkCluster = new MiniClusterResource(
        new MiniClusterResourceConfiguration.Builder()
            .setNumberSlotsPerTaskManager(4)
            .setNumberTaskManagers(2)
            .build()
    );

    @Test
    public void testSepsisDetectionPattern() throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Create test data stream
        DataStream<EnrichedEvent> testStream = env.fromElements(
            createVitalEvent("P1", "temperature", 38.5, 1000),
            createVitalEvent("P1", "heart_rate", 105, 2000),
            createVitalEvent("P1", "systolic_bp", 85, 3000),
            createLabEvent("P1", "lactate", 2.5, 4000)
        );

        // Apply CEP pattern
        Pattern<EnrichedEvent, ?> sepsisPattern = Module4a_CEPPatterns.createSepsisPattern();
        PatternStream<EnrichedEvent> patternStream = CEP.pattern(
            testStream.keyBy(EnrichedEvent::getPatientId),
            sepsisPattern
        );

        // Collect results
        List<ClinicalAlert> alerts = new ArrayList<>();
        patternStream.select(pattern -> {
            // Pattern matched - create alert
            return ClinicalAlert.builder()
                .alertType("SEPSIS_ALERT")
                .severity(AlertSeverity.CRITICAL)
                .build();
        }).addSink(new CollectSink<>(alerts));

        env.execute();

        // Assert sepsis alert generated
        assertEquals(1, alerts.size());
        assertEquals("SEPSIS_ALERT", alerts.get(0).getAlertType());
    }
}
```

## Monitoring

### Prometheus Metrics

```yaml
# prometheus-config.yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'flink'
    static_configs:
      - targets:
        - 'flink-jobmanager:9249'
        - 'flink-taskmanager-0:9249'
        - 'flink-taskmanager-1:9249'
        - 'flink-taskmanager-2:9249'

  - job_name: 'kafka'
    static_configs:
      - targets: ['kafka:9308']
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Flink EHR Intelligence Engine",
    "panels": [
      {
        "title": "Event Processing Rate",
        "targets": [{
          "expr": "rate(flink_taskmanager_job_task_numRecordsInPerSecond[5m])"
        }]
      },
      {
        "title": "Checkpoint Duration",
        "targets": [{
          "expr": "flink_jobmanager_job_lastCheckpointDuration"
        }]
      },
      {
        "title": "Alert Generation Rate",
        "targets": [{
          "expr": "rate(clinical_alerts_generated_total[5m])"
        }]
      },
      {
        "title": "Patient State Size",
        "targets": [{
          "expr": "flink_taskmanager_job_task_Shuffle_Netty_Input_Buffers_outPoolUsage"
        }]
      }
    ]
  }
}
```

## Troubleshooting

### Common Issues

1. **Out of Memory Errors**
   - Increase TaskManager memory
   - Tune RocksDB block cache size
   - Enable managed memory

2. **Checkpoint Failures**
   - Increase checkpoint timeout
   - Use unaligned checkpoints
   - Optimize state backend configuration

3. **High Latency**
   - Increase parallelism
   - Optimize watermark strategy
   - Reduce checkpoint interval

4. **State Growth**
   - Configure state TTL
   - Implement compaction filters
   - Use incremental checkpoints

## Performance Tuning

### RocksDB Optimization

```java
// Custom RocksDB options
public class OptimizedRocksDBStateBackend {

    public static RocksDBStateBackend createOptimizedBackend() {
        RocksDBStateBackend backend = new RocksDBStateBackend("s3://checkpoints");

        // Configure options
        backend.setPredefinedOptions(PredefinedOptions.SPINNING_DISK_OPTIMIZED_HIGH_MEM);
        backend.setNumberOfTransferingThreads(4);
        backend.setDbStoragePath("/ssd/rocksdb");

        // Memory management
        Configuration config = new Configuration();
        config.setString("state.backend.rocksdb.memory.managed", "true");
        config.setString("state.backend.rocksdb.memory.fixed-per-slot", "512mb");
        config.setString("state.backend.rocksdb.memory.high-prio-pool-ratio", "0.1");

        backend.configure(config, Thread.currentThread().getContextClassLoader());

        return backend;
    }
}
```

### Parallelism Tuning

```java
// Dynamic parallelism based on load
public class DynamicParallelismConfig {

    public static int calculateOptimalParallelism(int numPatients, int eventsPerSecond) {
        // Base parallelism on patient count
        int baseParallelism = numPatients / 1000;

        // Adjust for event rate
        int eventParallelism = eventsPerSecond / 1000;

        // Take maximum and apply bounds
        int parallelism = Math.max(baseParallelism, eventParallelism);
        parallelism = Math.max(8, Math.min(parallelism, 128));

        return parallelism;
    }
}
```

## Next Steps

1. **Set up development environment**: Install Java, Maven, Docker
2. **Start local Kafka infrastructure**:
   ```bash
   cd /backend/shared-infrastructure/kafka/
   ./start-kafka.sh
   ```
3. **Verify Kafka cluster health**:
   ```bash
   ./health-check.sh
   ```
4. **Create additional Flink-specific topics**:
   ```bash
   ./manage-topics.sh --create --topic ehr-alerts-critical --partitions 16 --replication-factor 3
   ./manage-topics.sh --create --topic ehr-alerts-urgent --partitions 16 --replication-factor 3
   ./manage-topics.sh --create --topic ehr-ml-scores --partitions 12 --replication-factor 3
   ```
3. **Implement Module 1**: Start with ingestion and validation
4. **Test with sample data**: Use historical patient data for validation
5. **Iterate on patterns**: Refine CEP patterns with clinical team
6. **Performance testing**: Validate 10K events/sec throughput
7. **Production deployment**: Gradual rollout with monitoring

This implementation guide provides a comprehensive foundation for building the Flink EHR Intelligence Engine. Each module is designed to integrate seamlessly with CardioFit's existing infrastructure while adding advanced stream processing capabilities for real-time clinical intelligence.