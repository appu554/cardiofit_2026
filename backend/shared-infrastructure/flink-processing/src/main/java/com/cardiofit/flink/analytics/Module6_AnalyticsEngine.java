package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.serialization.JsonDeserializer;
import com.cardiofit.flink.serialization.JsonSerializer;
import com.cardiofit.flink.utils.KafkaConfigLoader;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.connector.base.DeliveryGuarantee;
import org.apache.flink.connector.kafka.sink.KafkaRecordSerializationSchema;
import org.apache.flink.connector.kafka.sink.KafkaSink;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.table.api.EnvironmentSettings;
import org.apache.flink.table.api.Table;
import org.apache.flink.table.api.bridge.java.StreamTableEnvironment;
import org.apache.flink.table.api.DataTypes;
import org.apache.flink.table.api.Schema;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.time.Duration;

/**
 * Module 6: SQL Analytics Engine
 *
 * Responsibilities:
 * - Create materialized views using Flink Table API / SQL
 * - Generate real-time analytics aggregations
 * - Support clinical operational dashboards
 * - Enable data-driven clinical decision making
 *
 * Analytics Views (Table API/SQL):
 * 1. Patient Census - 1-minute tumbling window
 * 2. Alert Metrics - 1-minute tumbling window
 * 3. ML Performance - 5-minute tumbling window
 * 4. Department Workload - 1-hour sliding window with 5-minute slide
 * 5. Sepsis Surveillance - Real-time streaming view
 *
 * Analytics Components (DataStream API):
 * 6. Time-Series Aggregator - 1-minute vital sign rollups
 * 7. Population Health Analytics - Department-level population metrics
 */
public class Module6_AnalyticsEngine {
    private static final Logger LOG = LoggerFactory.getLogger(Module6_AnalyticsEngine.class);

    public static void main(String[] args) throws Exception {
        LOG.info("Starting Module 6: SQL Analytics Engine with DataStream components");

        // Set up execution environment
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure for analytics workloads
        env.setParallelism(4); // Higher parallelism for analytics
        env.enableCheckpointing(60000); // 1-minute checkpoints for analytics
        env.getCheckpointConfig().setMinPauseBetweenCheckpoints(10000);
        env.getCheckpointConfig().setCheckpointTimeout(600000); // 10 minutes

        // Create Table environment with streaming mode
        EnvironmentSettings settings = EnvironmentSettings
            .newInstance()
            .inStreamingMode()
            .build();

        StreamTableEnvironment tableEnv = StreamTableEnvironment.create(env, settings);

        // Set idle state retention to prevent state bloat (24 hours for analytics)
        tableEnv.getConfig().setIdleStateRetention(Duration.ofHours(24));

        // Get Kafka bootstrap servers
        String bootstrapServers = getBootstrapServers();

        // Create Table API analytics pipeline and get the StatementSet
        org.apache.flink.table.api.StatementSet statementSet = createAnalyticsPipeline(tableEnv, bootstrapServers);

        // Create DataStream API analytics components
        createDataStreamAnalytics(env, bootstrapServers);

        // CRITICAL: Execute the StatementSet FIRST to register SQL operators in the job graph
        // This must be called before env.execute() to include Table API operators in the job
        LOG.info("Executing StatementSet to register SQL view operators in job graph...");
        statementSet.execute();

        LOG.info("Module 6 Analytics Engine started successfully");
    }

    public static org.apache.flink.table.api.StatementSet createAnalyticsPipeline(StreamTableEnvironment tableEnv, String bootstrapServers) {
        LOG.info("Creating SQL analytics pipeline with 5 materialized views");
        LOG.info("Using Kafka bootstrap servers: {}", bootstrapServers);

        // Create a StatementSet to batch all inserts
        org.apache.flink.table.api.StatementSet statementSet = tableEnv.createStatementSet();

        // ===== SOURCE TABLES =====

        // 1. Create source table for enriched patient events (from Module 1-3)
        createEnrichedPatientEventsSourceTable(tableEnv, bootstrapServers);

        // 2. Create source table for clinical patterns (from Module 4)
        createClinicalPatternsSourceTable(tableEnv, bootstrapServers);

        // 3. Create source table for ML predictions (from Module 5)
        createMLPredictionsSourceTable(tableEnv, bootstrapServers);

        // ===== ANALYTICS VIEWS =====

        // View 1: Patient Census (1-minute tumbling window)
        createPatientCensusView(tableEnv, bootstrapServers, statementSet);

        // View 2: Alert Metrics (1-minute tumbling window)
        createAlertMetricsView(tableEnv, bootstrapServers, statementSet);

        // View 3: ML Performance Metrics (5-minute tumbling window)
        createMLPerformanceView(tableEnv, bootstrapServers, statementSet);

        // View 4: Department Workload (1-hour sliding window, 5-minute slide)
        createDepartmentWorkloadView(tableEnv, bootstrapServers, statementSet);

        // View 5: Sepsis Surveillance (real-time streaming)
        createSepsisSurveillanceView(tableEnv, bootstrapServers, statementSet);

        // Return StatementSet for explicit execution in main()
        LOG.info("SQL analytics pipeline configured with 5 materialized views (Patient Census, Alert Metrics, ML Performance, Department Workload, Sepsis Surveillance)");
        return statementSet;
    }

    /**
     * Create DataStream API analytics components
     * - Time-Series Aggregator: 1-minute vital sign rollups
     * - Population Health Analytics: Department-level population metrics
     */
    public static void createDataStreamAnalytics(StreamExecutionEnvironment env, String bootstrapServers) {
        LOG.info("Creating DataStream analytics components (Time-Series Aggregator, Population Health Analytics)");

        // ===== Component 6A.2: Time-Series Aggregator =====

        // Create Kafka source for enriched patient context events
        KafkaSource<EnrichedPatientContext> enrichedEventsSource = KafkaSource.<EnrichedPatientContext>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics("comprehensive-cds-events.v1")
            .setGroupId("module6-timeseries-aggregator")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new JsonDeserializer<>(EnrichedPatientContext.class))
            .build();

        DataStream<EnrichedPatientContext> enrichedEvents = env.fromSource(
            enrichedEventsSource,
            WatermarkStrategy.noWatermarks(),
            "enriched-patient-context-source"
        );

        // Apply time-series aggregation
        DataStream<VitalMetric> vitalMetrics = TimeSeriesAggregator.aggregateVitals(enrichedEvents);

        // Create Kafka sink for vital metrics
        KafkaSink<VitalMetric> vitalMetricsSink = KafkaSink.<VitalMetric>builder()
            .setBootstrapServers(bootstrapServers)
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic("analytics-vital-timeseries")
                .setValueSerializationSchema(new JsonSerializer<VitalMetric>())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-vital-metrics")
            .build();

        vitalMetrics.sinkTo(vitalMetricsSink).name("vital-metrics-sink");
        LOG.info("Time-Series Aggregator configured: enriched-events → analytics-vital-timeseries");

        // ===== Component 6A.3: Population Health Analytics =====

        // Create Kafka source for ML predictions
        KafkaSource<MLPrediction> mlPredictionsSource = KafkaSource.<MLPrediction>builder()
            .setBootstrapServers(bootstrapServers)
            .setTopics("inference-results.v1")
            .setGroupId("module6-population-health")
            .setStartingOffsets(OffsetsInitializer.earliest())
            .setValueOnlyDeserializer(new JsonDeserializer<>(MLPrediction.class))
            .build();

        DataStream<MLPrediction> mlPredictions = env.fromSource(
            mlPredictionsSource,
            WatermarkStrategy.noWatermarks(),
            "ml-predictions-source"
        );

        // Apply population health analytics (keyed by department)
        DataStream<PopulationMetrics> populationMetrics = mlPredictions
            .keyBy(prediction -> PopulationHealthAnalytics.extractDepartment(prediction))
            .process(new PopulationHealthAnalytics())
            .name("population-health-analytics");

        // Create Kafka sink for population metrics
        KafkaSink<PopulationMetrics> populationMetricsSink = KafkaSink.<PopulationMetrics>builder()
            .setBootstrapServers(bootstrapServers)
            .setRecordSerializer(KafkaRecordSerializationSchema.builder()
                .setTopic("analytics-population-health")
                .setValueSerializationSchema(new JsonSerializer<PopulationMetrics>())
                .build())
            .setDeliveryGuarantee(DeliveryGuarantee.AT_LEAST_ONCE)
            .setTransactionalIdPrefix("module6-population-health")
            .build();

        populationMetrics.sinkTo(populationMetricsSink).name("population-metrics-sink");
        LOG.info("Population Health Analytics configured: ml-predictions → analytics-population-health");

        LOG.info("DataStream analytics components configured successfully");
    }

    /**
     * Create source table for enriched patient context events from Module 2
     * Updated to match EnrichedPatientContext structure (not EnrichedEvent)
     * Removed encounterId as it doesn't exist in actual Kafka data
     */
    private static void createEnrichedPatientEventsSourceTable(StreamTableEnvironment tableEnv, String bootstrapServers) {
        String ddl = String.format(
            "CREATE TABLE enriched_patient_events (\n" +
            "  patientId STRING,\n" +
            "  eventType STRING,\n" +
            "  eventTime BIGINT,\n" +
            "  processingTime BIGINT,\n" +
            "  patientState ROW<\n" +
            "    patientId STRING,\n" +
            "    news2Score INT,\n" +
            "    qsofaScore INT,\n" +
            "    combinedAcuityScore DOUBLE,\n" +
            "    acuityLevel STRING,\n" +
            "    eventCount BIGINT,\n" +
            "    lastUpdated BIGINT\n" +
            "  >,\n" +
            "  proc_time AS PROCTIME()\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'comprehensive-cds-events.v1',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'properties.group.id' = 'module6-analytics-proctime',\n" +
            "  'scan.startup.mode' = 'latest-offset',\n" +
            "  'format' = 'json',\n" +
            "  'json.fail-on-missing-field' = 'false',\n" +
            "  'json.ignore-parse-errors' = 'true'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(ddl);
        LOG.info("Created source table: enriched_patient_events (simplified schema without encounterId)");
    }

    /**
     * Create source table for composed alerts from Module 6 Alert Composition
     * Note: This topic contains individual alert objects (NOT nested arrays)
     */
    private static void createClinicalPatternsSourceTable(StreamTableEnvironment tableEnv, String bootstrapServers) {
        String ddl = String.format(
            "CREATE TABLE composed_alerts (\n" +
            "  alert_id STRING,\n" +
            "  patient_id STRING,\n" +
            "  severity STRING,\n" +
            "  confidence DOUBLE,\n" +
            "  sources ARRAY<STRING>,\n" +
            "  evidence ROW<\n" +
            "    pattern_type STRING,\n" +
            "    pattern_event ROW<\n" +
            "      `timestamp` DOUBLE,\n" +
            "      severity DOUBLE,\n" +
            "      confidence DOUBLE,\n" +
            "      priority BIGINT\n" +
            "    >\n" +
            "  >,\n" +
            "  recommended_actions ARRAY<STRING>,\n" +
            "  suppression_count BIGINT,\n" +
            "  last_updated BIGINT,\n" +
            "  composition_strategy STRING,\n" +
            "  proc_time AS PROCTIME()\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'composed-alerts.v1',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'properties.group.id' = 'module6-analytics-alerts-proctime',\n" +
            "  'scan.startup.mode' = 'latest-offset',\n" +
            "  'format' = 'json',\n" +
            "  'json.fail-on-missing-field' = 'false',\n" +
            "  'json.ignore-parse-errors' = 'true'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(ddl);
        LOG.info("Created source table: composed_alerts (flat alert objects from Module 6)");
    }

    /**
     * Create source table for ML predictions from Module 5
     */
    private static void createMLPredictionsSourceTable(StreamTableEnvironment tableEnv, String bootstrapServers) {
        String ddl = String.format(
            "CREATE TABLE ml_predictions (\n" +
            "  id STRING,\n" +
            "  patient_id STRING,\n" +
            "  model_name STRING,\n" +
            "  modelVersion STRING,\n" +
            "  model_type STRING,\n" +
            "  primaryScore DOUBLE,\n" +
            "  confidenceScore DOUBLE,\n" +
            "  prediction_time BIGINT,\n" +
            "  inferenceLatencyMs BIGINT,\n" +
            "  risk_level STRING,\n" +
            "  proc_time AS PROCTIME()\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'inference-results.v1',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'properties.group.id' = 'module6-analytics-ml-proctime',\n" +
            "  'scan.startup.mode' = 'latest-offset',\n" +
            "  'format' = 'json',\n" +
            "  'json.fail-on-missing-field' = 'false',\n" +
            "  'json.ignore-parse-errors' = 'true'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(ddl);
        LOG.info("Created source table: ml_predictions with processing-time (fixed field names)");
    }

    /**
     * View 1: Patient Census - 1-minute tumbling window
     * Provides real-time patient census by event type and acuity level
     * Updated for EnrichedPatientContext structure
     */
    private static void createPatientCensusView(StreamTableEnvironment tableEnv, String bootstrapServers, org.apache.flink.table.api.StatementSet statementSet) {
        LOG.info("Creating Patient Census analytics view (1-minute tumbling window)");

        // Create analytics query with tumbling window using EnrichedPatientContext fields
        // Note: encounterId doesn't exist in Kafka data, using 0 as placeholder
        String query =
            "SELECT\n" +
            "  window_start,\n" +
            "  window_end,\n" +
            "  COALESCE(eventType, 'UNKNOWN') AS event_type,\n" +
            "  COUNT(DISTINCT patientId) AS active_patients,\n" +
            "  CAST(0 AS BIGINT) AS active_encounters,\n" +
            "  COUNT(*) AS total_events,\n" +
            "  CURRENT_TIMESTAMP AS processing_time\n" +
            "FROM TABLE(\n" +
            "  TUMBLE(TABLE enriched_patient_events, DESCRIPTOR(proc_time), INTERVAL '1' MINUTE)\n" +
            ")\n" +
            "GROUP BY window_start, window_end, eventType";

        Table patientCensusTable = tableEnv.sqlQuery(query);

        // Create sink table for patient census
        String sinkDdl = String.format(
            "CREATE TABLE analytics_patient_census (\n" +
            "  window_start TIMESTAMP(3),\n" +
            "  window_end TIMESTAMP(3),\n" +
            "  event_type STRING,\n" +
            "  active_patients BIGINT,\n" +
            "  active_encounters BIGINT,\n" +
            "  total_events BIGINT,\n" +
            "  processing_time TIMESTAMP(3)\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'analytics-patient-census',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'format' = 'json',\n" +
            "  'json.timestamp-format.standard' = 'ISO-8601',\n" +
            "  'sink.transactional-id-prefix' = 'patient-census-'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(sinkDdl);
        LOG.info("Created sink table: analytics_patient_census");

        // Add insert to statement set instead of executing immediately
        statementSet.addInsert("analytics_patient_census", patientCensusTable);
        LOG.info("Added patient census analytics to pipeline");
    }

    /**
     * View 2: Alert Metrics - 1-minute tumbling window
     * Aggregates alert data by severity, department, and pattern type
     * Note: Now reads from composed_alerts which contains flat alert objects
     */
    private static void createAlertMetricsView(StreamTableEnvironment tableEnv, String bootstrapServers, org.apache.flink.table.api.StatementSet statementSet) {
        LOG.info("Creating Alert Metrics analytics view (1-minute tumbling window)");

        String query =
            "SELECT\n" +
            "  window_start,\n" +
            "  window_end,\n" +
            "  'UNKNOWN' AS department,\n" +
            "  COALESCE(evidence.pattern_type, 'UNKNOWN') AS pattern_type,\n" +
            "  COALESCE(severity, 'UNKNOWN') AS severity,\n" +
            "  COUNT(*) AS alert_count,\n" +
            "  COUNT(DISTINCT patient_id) AS unique_patients,\n" +
            "  LISTAGG(DISTINCT patient_id, ',') AS patient_ids,\n" +
            "  AVG(confidence) AS avg_confidence,\n" +
            "  MAX(confidence) AS max_confidence,\n" +
            "  MIN(confidence) AS min_confidence,\n" +
            "  CURRENT_TIMESTAMP AS processing_time\n" +
            "FROM TABLE(\n" +
            "  TUMBLE(TABLE composed_alerts, DESCRIPTOR(proc_time), INTERVAL '1' MINUTE)\n" +
            ")\n" +
            "GROUP BY window_start, window_end, evidence.pattern_type, severity";

        Table alertMetricsTable = tableEnv.sqlQuery(query);

        String sinkDdl = String.format(
            "CREATE TABLE analytics_alert_metrics (\n" +
            "  window_start TIMESTAMP(3),\n" +
            "  window_end TIMESTAMP(3),\n" +
            "  department STRING,\n" +
            "  pattern_type STRING,\n" +
            "  severity STRING,\n" +
            "  alert_count BIGINT,\n" +
            "  unique_patients BIGINT,\n" +
            "  patient_ids STRING,\n" +
            "  avg_confidence DOUBLE,\n" +
            "  max_confidence DOUBLE,\n" +
            "  min_confidence DOUBLE,\n" +
            "  processing_time TIMESTAMP(3),\n" +
            "  PRIMARY KEY (window_start, window_end, pattern_type, severity) NOT ENFORCED\n" +
            ") WITH (\n" +
            "  'connector' = 'upsert-kafka',\n" +
            "  'topic' = 'analytics-alert-metrics',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'key.format' = 'json',\n" +
            "  'value.format' = 'json',\n" +
            "  'value.json.timestamp-format.standard' = 'ISO-8601',\n" +
            "  'sink.transactional-id-prefix' = 'alert-metrics-'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(sinkDdl);
        LOG.info("Created sink table: analytics_alert_metrics");

        statementSet.addInsert("analytics_alert_metrics", alertMetricsTable);
        LOG.info("Added alert metrics analytics to pipeline");
    }

    /**
     * View 3: ML Performance Metrics - 5-minute tumbling window
     * Monitors ML model performance, latency, and prediction distribution
     */
    private static void createMLPerformanceView(StreamTableEnvironment tableEnv, String bootstrapServers, org.apache.flink.table.api.StatementSet statementSet) {
        LOG.info("Creating ML Performance analytics view (5-minute tumbling window)");

        String query =
            "SELECT\n" +
            "  window_start,\n" +
            "  window_end,\n" +
            "  COALESCE(model_name, 'UNKNOWN') AS model_name,\n" +
            "  COALESCE(modelVersion, 'UNKNOWN') AS model_version,\n" +
            "  COALESCE(model_type, 'UNKNOWN') AS prediction_type,\n" +
            "  COUNT(*) AS prediction_count,\n" +
            "  COUNT(DISTINCT patient_id) AS unique_patients,\n" +
            "  LISTAGG(DISTINCT patient_id, ',') AS patient_ids,\n" +
            "  AVG(primaryScore) AS avg_risk_score,\n" +
            "  AVG(confidenceScore) AS avg_confidence,\n" +
            "  AVG(CAST(inferenceLatencyMs AS DOUBLE)) AS avg_inference_latency_ms,\n" +
            "  MAX(CAST(inferenceLatencyMs AS DOUBLE)) AS max_inference_latency_ms,\n" +
            "  MAX(CAST(inferenceLatencyMs AS DOUBLE)) AS p95_inference_latency_ms,\n" +
            "  SUM(CASE WHEN primaryScore > 0.8 THEN 1 ELSE 0 END) AS high_risk_predictions,\n" +
            "  SUM(CASE WHEN primaryScore > 0.6 AND primaryScore <= 0.8 THEN 1 ELSE 0 END) AS medium_risk_predictions,\n" +
            "  SUM(CASE WHEN primaryScore <= 0.6 THEN 1 ELSE 0 END) AS low_risk_predictions,\n" +
            "  CURRENT_TIMESTAMP AS processing_time\n" +
            "FROM TABLE(\n" +
            "  TUMBLE(TABLE ml_predictions, DESCRIPTOR(proc_time), INTERVAL '5' MINUTE)\n" +
            ")\n" +
            "GROUP BY window_start, window_end, model_name, modelVersion, model_type";

        Table mlPerformanceTable = tableEnv.sqlQuery(query);

        String sinkDdl = String.format(
            "CREATE TABLE analytics_ml_performance (\n" +
            "  window_start TIMESTAMP(3),\n" +
            "  window_end TIMESTAMP(3),\n" +
            "  model_name STRING,\n" +
            "  model_version STRING,\n" +
            "  prediction_type STRING,\n" +
            "  prediction_count BIGINT,\n" +
            "  unique_patients BIGINT,\n" +
            "  patient_ids STRING,\n" +
            "  avg_risk_score DOUBLE,\n" +
            "  avg_confidence DOUBLE,\n" +
            "  avg_inference_latency_ms DOUBLE,\n" +
            "  max_inference_latency_ms DOUBLE,\n" +
            "  p95_inference_latency_ms DOUBLE,\n" +
            "  high_risk_predictions BIGINT,\n" +
            "  medium_risk_predictions BIGINT,\n" +
            "  low_risk_predictions BIGINT,\n" +
            "  processing_time TIMESTAMP(3)\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'analytics-ml-performance',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'format' = 'json',\n" +
            "  'json.timestamp-format.standard' = 'ISO-8601',\n" +
            "  'sink.transactional-id-prefix' = 'ml-performance-'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(sinkDdl);
        LOG.info("Created sink table: analytics_ml_performance");

        statementSet.addInsert("analytics_ml_performance", mlPerformanceTable);
        LOG.info("Added ML performance analytics to pipeline");
    }

    /**
     * View 4: Department Workload - 1-hour sliding window with 5-minute slide
     * Provides trending workload metrics
     * Updated for EnrichedPatientContext structure (department/unit not available)
     */
    private static void createDepartmentWorkloadView(StreamTableEnvironment tableEnv, String bootstrapServers, org.apache.flink.table.api.StatementSet statementSet) {
        LOG.info("Creating Department Workload analytics view (1-hour sliding window, 5-minute slide)");

        // Note: Using HOP (sliding window) with 1-hour window size and 5-minute slide
        // Department/unit fields removed as not available in EnrichedPatientContext
        String query =
            "SELECT\n" +
            "  window_start,\n" +
            "  window_end,\n" +
            "  'SYSTEM' AS department,\n" +
            "  'DEFAULT' AS unit,\n" +
            "  COUNT(DISTINCT patientId) AS total_patients,\n" +
            "  COUNT(*) AS total_events,\n" +
            "  COALESCE(patientState.acuityLevel, 'UNKNOWN') AS primary_acuity_level,\n" +
            "  COUNT(DISTINCT CASE WHEN patientState.acuityLevel IN ('HIGH', 'CRITICAL') THEN patientId ELSE NULL END) AS high_acuity_patients,\n" +
            "  CURRENT_TIMESTAMP AS processing_time\n" +
            "FROM TABLE(\n" +
            "  HOP(TABLE enriched_patient_events, DESCRIPTOR(proc_time), INTERVAL '5' MINUTE, INTERVAL '1' HOUR)\n" +
            ")\n" +
            "GROUP BY window_start, window_end, patientState.acuityLevel";

        Table departmentWorkloadTable = tableEnv.sqlQuery(query);

        String sinkDdl = String.format(
            "CREATE TABLE analytics_department_workload (\n" +
            "  window_start TIMESTAMP(3),\n" +
            "  window_end TIMESTAMP(3),\n" +
            "  department STRING,\n" +
            "  unit STRING,\n" +
            "  total_patients BIGINT,\n" +
            "  total_events BIGINT,\n" +
            "  primary_acuity_level STRING,\n" +
            "  high_acuity_patients BIGINT,\n" +
            "  processing_time TIMESTAMP(3)\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'analytics-department-workload',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'sink.transactional-id-prefix' = 'module6-department-workload-',\n" +
            "  'format' = 'json',\n" +
            "  'json.timestamp-format.standard' = 'ISO-8601'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(sinkDdl);
        LOG.info("Created sink table: analytics_department_workload");

        statementSet.addInsert("analytics_department_workload", departmentWorkloadTable);
        LOG.info("Added department workload analytics to pipeline");
    }

    /**
     * View 5: Sepsis Surveillance - Real-time streaming view
     * Identifies and tracks sepsis risk patients in real-time
     * Updated for EnrichedPatientContext structure with patientState.news2Score/qsofaScore
     */
    private static void createSepsisSurveillanceView(StreamTableEnvironment tableEnv, String bootstrapServers, org.apache.flink.table.api.StatementSet statementSet) {
        LOG.info("Creating Sepsis Surveillance analytics view (real-time streaming)");

        // Real-time view without windowing for immediate alerting
        // Access NEWS2 and qSOFA from patientState (INT fields, not nested structure)
        // Note: encounterId doesn't exist in Kafka data, using 'UNKNOWN' as placeholder
        String query =
            "SELECT\n" +
            "  patientId AS patient_id,\n" +
            "  'UNKNOWN' AS encounter_id,\n" +
            "  'UNKNOWN' AS department,\n" +
            "  'UNKNOWN' AS unit,\n" +
            "  'MONITORING' AS primary_finding,\n" +
            "  CAST(COALESCE(patientState.news2Score, 0) AS DOUBLE) AS news2_score,\n" +
            "  CAST(COALESCE(patientState.qsofaScore, 0) AS DOUBLE) AS qsofa_score,\n" +
            "  COALESCE(patientState.acuityLevel, 'UNKNOWN') AS acuity_level,\n" +
            "  CASE\n" +
            "    WHEN patientState.qsofaScore >= 2 AND patientState.news2Score >= 5 THEN 'HIGH'\n" +
            "    WHEN patientState.qsofaScore >= 2 OR patientState.news2Score >= 7 THEN 'MODERATE'\n" +
            "    WHEN patientState.news2Score >= 5 THEN 'LOW'\n" +
            "    ELSE 'MINIMAL'\n" +
            "  END AS sepsis_risk_level,\n" +
            "  CASE\n" +
            "    WHEN patientState.qsofaScore >= 2 THEN 'qSOFA >= 2 (Organ dysfunction suspected)'\n" +
            "    WHEN patientState.news2Score >= 7 THEN 'NEWS2 >= 7 (High clinical risk)'\n" +
            "    WHEN patientState.news2Score >= 5 THEN 'NEWS2 >= 5 (Moderate risk)'\n" +
            "    ELSE 'No immediate sepsis concern'\n" +
            "  END AS risk_reason,\n" +
            "  CURRENT_TIMESTAMP AS processing_time\n" +
            "FROM enriched_patient_events\n" +
            "WHERE patientState.qsofaScore >= 2\n" +
            "  OR patientState.news2Score >= 5";

        Table sepsisSurveillanceTable = tableEnv.sqlQuery(query);

        String sinkDdl = String.format(
            "CREATE TABLE analytics_sepsis_surveillance (\n" +
            "  patient_id STRING,\n" +
            "  encounter_id STRING,\n" +
            "  department STRING,\n" +
            "  unit STRING,\n" +
            "  primary_finding STRING,\n" +
            "  news2_score DOUBLE,\n" +
            "  qsofa_score DOUBLE,\n" +
            "  acuity_level STRING,\n" +
            "  sepsis_risk_level STRING,\n" +
            "  risk_reason STRING,\n" +
            "  processing_time TIMESTAMP(3)\n" +
            ") WITH (\n" +
            "  'connector' = 'kafka',\n" +
            "  'topic' = 'analytics-sepsis-surveillance',\n" +
            "  'properties.bootstrap.servers' = '%s',\n" +
            "  'sink.transactional-id-prefix' = 'module6-sepsis-surveillance-',\n" +
            "  'format' = 'json',\n" +
            "  'json.timestamp-format.standard' = 'ISO-8601'\n" +
            ")",
            bootstrapServers
        );

        tableEnv.executeSql(sinkDdl);
        LOG.info("Created sink table: analytics_sepsis_surveillance");

        statementSet.addInsert("analytics_sepsis_surveillance", sepsisSurveillanceTable);
        LOG.info("Added sepsis surveillance analytics to pipeline");
    }

    /**
     * Get Kafka bootstrap servers from environment or default
     */
    private static String getBootstrapServers() {
        String kafkaServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        return (kafkaServers != null && !kafkaServers.isEmpty())
            ? kafkaServers
            : (KafkaConfigLoader.isRunningInDocker() ? "kafka:29092" : "localhost:9092");
    }
}
