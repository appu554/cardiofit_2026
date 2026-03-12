
package com.cardiofit.flink.sinks;

import com.cardiofit.flink.models.RoutedEvent;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.stream.models.CanonicalEvent;
import com.cardiofit.flink.models.MLPrediction;
import com.clickhouse.jdbc.ClickHouseDataSource;
import org.apache.flink.api.connector.sink2.Sink;
import org.apache.flink.api.connector.sink2.WriterInitContext;
import org.apache.flink.api.connector.sink2.SinkWriter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.util.Properties;
import java.util.Map;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.TimeUnit;

/**
 * ClickHouse sink for time-series analytics and aggregations
 * Optimized for OLAP queries and real-time dashboards
 * Stores clinical metrics, aggregated patterns, and performance data
 *
 * Migrated to Flink 2.x Sink API (replaces RichSinkFunction)
 */
public class ClickHouseSink implements Sink<RoutedEvent> {

    private static final Logger LOG = LoggerFactory.getLogger(ClickHouseSink.class);

    // Configuration
    private final String jdbcUrl;
    private final String username;
    private final String password;
    private final String database;
    private final int batchSize;
    private final long flushIntervalMs;

    public ClickHouseSink(String host, int port, String database, String username, String password) {
        this.jdbcUrl = String.format("jdbc:clickhouse://%s:%d/%s", host, port, database);
        this.database = database;
        this.username = username;
        this.password = password;
        this.batchSize = 1000;
        this.flushIntervalMs = 5000;
    }

    public ClickHouseSink() {
        // Default configuration matching shared infrastructure setup
        String host = System.getenv().getOrDefault("CLICKHOUSE_HOST", "localhost");
        int port = Integer.parseInt(System.getenv().getOrDefault("CLICKHOUSE_PORT", "8123"));
        this.database = System.getenv().getOrDefault("CLICKHOUSE_DATABASE", "cardiofit_analytics");
        this.username = System.getenv().getOrDefault("CLICKHOUSE_USERNAME", "cardiofit_user");
        this.password = System.getenv().getOrDefault("CLICKHOUSE_PASSWORD", "ClickHouse2024!");
        this.jdbcUrl = String.format("jdbc:clickhouse://%s:%d/%s", host, port, database);
        this.batchSize = Integer.parseInt(System.getenv().getOrDefault("CLICKHOUSE_BATCH_SIZE", "1000"));
        this.flushIntervalMs = Long.parseLong(System.getenv().getOrDefault("CLICKHOUSE_FLUSH_INTERVAL_MS", "5000"));
    }

    @Override
    public SinkWriter<RoutedEvent> createWriter(WriterInitContext context) throws IOException {
        return new ClickHouseSinkWriter(jdbcUrl, username, password, database, batchSize, flushIntervalMs);
    }

    /**
     * SinkWriter implementation for ClickHouse batch operations
     */
    private static class ClickHouseSinkWriter implements SinkWriter<RoutedEvent> {

        private static final Logger LOG = LoggerFactory.getLogger(ClickHouseSinkWriter.class);

        private final String jdbcUrl;
        private final String username;
        private final String password;
        private final String database;
        private final int batchSize;
        private final long flushIntervalMs;

        private transient ClickHouseDataSource dataSource;
        private transient Connection connection;
        private transient PreparedStatement clinicalEventsStmt;
        private transient PreparedStatement patternMetricsStmt;
        private transient PreparedStatement mlPredictionsStmt;
        private transient BlockingQueue<RoutedEvent> buffer;
        private transient Thread flushThread;
        private volatile boolean running = true;

        public ClickHouseSinkWriter(String jdbcUrl, String username, String password, String database,
                                   int batchSize, long flushIntervalMs) {
            this.jdbcUrl = jdbcUrl;
            this.username = username;
            this.password = password;
            this.database = database;
            this.batchSize = batchSize;
            this.flushIntervalMs = flushIntervalMs;

            try {
                initializeClickHouse();
            } catch (Exception e) {
                LOG.error("Failed to initialize ClickHouse sink", e);
                throw new RuntimeException("ClickHouse initialization failed", e);
            }
        }

        private void initializeClickHouse() throws SQLException {
            // Create ClickHouse data source
            Properties properties = new Properties();
            properties.setProperty("user", username);
            properties.setProperty("password", password);
            properties.setProperty("socket_timeout", "300000");
            properties.setProperty("dataTransferTimeout", "300000");
            properties.setProperty("keepAliveTimeout", "300000");

            dataSource = new ClickHouseDataSource(jdbcUrl, properties);
            connection = dataSource.getConnection();

            // Create tables if not exist
            createTablesIfNotExist();

            // Prepare batch insert statements
            prepareBatchStatements();

            // Initialize buffer and flush thread
            buffer = new ArrayBlockingQueue<>(batchSize * 2);
            startFlushThread();

            LOG.info("Initialized ClickHouse sink connecting to {}", jdbcUrl);
        }

        private void createTablesIfNotExist() throws SQLException {
            // Clinical events time-series table
            String createClinicalEventsTable =
                "CREATE TABLE IF NOT EXISTS clinical_events (" +
                    "event_id String, " +
                    "patient_id String, " +
                    "event_type String, " +
                    "priority Enum8('LOW' = 1, 'NORMAL' = 2, 'HIGH' = 3, 'CRITICAL' = 4), " +
                    "event_time DateTime64(3), " +
                    "processing_time DateTime64(3), " +
                    "latency_ms UInt32, " +
                    "clinical_significance Float32, " +
                    "confidence_score Float32, " +
                    "destinations Array(String), " +
                    "metadata String, " +
                    "date Date MATERIALIZED toDate(event_time) " +
                ") ENGINE = MergeTree() " +
                "PARTITION BY toYYYYMM(date) " +
                "ORDER BY (patient_id, event_time) " +
                "TTL date + INTERVAL 90 DAY " +
                "SETTINGS index_granularity = 8192";

            // Pattern detection metrics table
            String createPatternMetricsTable =
                "CREATE TABLE IF NOT EXISTS pattern_metrics (" +
                    "pattern_id String, " +
                    "patient_id String, " +
                    "pattern_type String, " +
                    "detection_time DateTime64(3), " +
                    "confidence Float32, " +
                    "severity Float32, " +
                    "contributing_events Array(String), " +
                    "pattern_features String, " +
                    "date Date MATERIALIZED toDate(detection_time) " +
                ") ENGINE = MergeTree() " +
                "PARTITION BY toYYYYMM(date) " +
                "ORDER BY (pattern_type, patient_id, detection_time) " +
                "TTL date + INTERVAL 180 DAY " +
                "SETTINGS index_granularity = 8192";

            // ML predictions analytics table
            String createMLPredictionsTable =
                "CREATE TABLE IF NOT EXISTS ml_predictions (" +
                    "prediction_id String, " +
                    "patient_id String, " +
                    "prediction_type String, " +
                    "risk_score Float32, " +
                    "confidence Float32, " +
                    "prediction_time DateTime64(3), " +
                    "model_version String, " +
                    "feature_importance String, " +
                    "date Date MATERIALIZED toDate(prediction_time) " +
                ") ENGINE = MergeTree() " +
                "PARTITION BY toYYYYMM(date) " +
                "ORDER BY (prediction_type, patient_id, prediction_time) " +
                "TTL date + INTERVAL 365 DAY " +
                "SETTINGS index_granularity = 8192";

            // Aggregated metrics materialized view
            String createAggregatedMetricsView =
                "CREATE MATERIALIZED VIEW IF NOT EXISTS hourly_metrics " +
                "ENGINE = SummingMergeTree() " +
                "PARTITION BY toYYYYMM(hour_bucket) " +
                "ORDER BY (hour_bucket, event_type, priority) " +
                "AS SELECT " +
                    "toStartOfHour(event_time) as hour_bucket, " +
                    "event_type, " +
                    "priority, " +
                    "count() as event_count, " +
                    "avg(latency_ms) as avg_latency, " +
                    "max(latency_ms) as max_latency, " +
                    "min(latency_ms) as min_latency, " +
                    "avg(confidence_score) as avg_confidence " +
                "FROM clinical_events " +
                "GROUP BY hour_bucket, event_type, priority";

            try (var stmt = connection.createStatement()) {
                stmt.execute(createClinicalEventsTable);
                stmt.execute(createPatternMetricsTable);
                stmt.execute(createMLPredictionsTable);
                stmt.execute(createAggregatedMetricsView);
                LOG.info("ClickHouse tables and views created successfully");
            }
        }

        private void prepareBatchStatements() throws SQLException {
            // Prepare batch insert for clinical events
            String insertClinicalEvents =
                "INSERT INTO clinical_events (" +
                    "event_id, patient_id, event_type, priority, " +
                    "event_time, processing_time, latency_ms, " +
                    "clinical_significance, confidence_score, " +
                    "destinations, metadata " +
                ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";
            clinicalEventsStmt = connection.prepareStatement(insertClinicalEvents);

            // Prepare batch insert for pattern metrics
            String insertPatternMetrics =
                "INSERT INTO pattern_metrics (" +
                    "pattern_id, patient_id, pattern_type, " +
                    "detection_time, confidence, severity, " +
                    "contributing_events, pattern_features " +
                ") VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
            patternMetricsStmt = connection.prepareStatement(insertPatternMetrics);

            // Prepare batch insert for ML predictions
            String insertMLPredictions =
                "INSERT INTO ml_predictions (" +
                    "prediction_id, patient_id, prediction_type, " +
                    "risk_score, confidence, prediction_time, " +
                    "model_version, feature_importance " +
                ") VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
            mlPredictionsStmt = connection.prepareStatement(insertMLPredictions);
        }

        @Override
        public void write(RoutedEvent event, Context context) throws IOException, InterruptedException {
            // Only process events routed to analytics
            if (!event.hasDestination("analytics") && !event.hasDestination("clickhouse")) {
                return;
            }

            // Add to buffer for batch processing
            if (!buffer.offer(event, 100, TimeUnit.MILLISECONDS)) {
                LOG.warn("ClickHouse buffer full, dropping event {}", event.getId());
            }
        }

        private void startFlushThread() {
            flushThread = new Thread(() -> {
                while (running) {
                    try {
                        Thread.sleep(flushIntervalMs);
                        flushBuffer();
                    } catch (InterruptedException e) {
                        Thread.currentThread().interrupt();
                        break;
                    } catch (Exception e) {
                        LOG.error("Error flushing ClickHouse buffer", e);
                    }
                }
            }, "ClickHouse-Flush-Thread");
            flushThread.setDaemon(true);
            flushThread.start();
        }

        private void flushBuffer() throws SQLException {
            if (buffer.isEmpty()) {
                return;
            }

            int count = 0;
            RoutedEvent event;

            while ((event = buffer.poll()) != null && count < batchSize) {
                try {
                    switch (event.getSourceEventType()) {
                        case "PATTERN_EVENT":
                            addPatternMetric(event);
                            break;
                        case "ML_PREDICTION":
                            addMLPrediction(event);
                            break;
                        default:
                            addClinicalEvent(event);
                    }
                    count++;
                } catch (Exception e) {
                    LOG.error("Error processing event {} for ClickHouse", event.getId(), e);
                }
            }

            // Execute batch inserts
            if (count > 0) {
                clinicalEventsStmt.executeBatch();
                patternMetricsStmt.executeBatch();
                mlPredictionsStmt.executeBatch();
                LOG.debug("Flushed {} events to ClickHouse", count);
            }
        }

        private void addClinicalEvent(RoutedEvent event) throws SQLException {
            clinicalEventsStmt.setString(1, event.getId());
            clinicalEventsStmt.setString(2, ((CanonicalEvent)event).getPatientId());
            clinicalEventsStmt.setString(3, event.getSourceEventType());
            clinicalEventsStmt.setString(4, event.getPriority().name());
            clinicalEventsStmt.setTimestamp(5, new Timestamp(event.getRoutingTime()));
            clinicalEventsStmt.setTimestamp(6, new Timestamp(System.currentTimeMillis()));
            clinicalEventsStmt.setInt(7, (int)(System.currentTimeMillis() - event.getRoutingTime()));

            // Extract clinical significance and confidence from payload
            float significance = 0.5f;
            float confidence = 0.8f;
            if (event.getOriginalPayload() instanceof Map) {
                Map<String, Object> payload = (Map<String, Object>) event.getOriginalPayload();
                significance = ((Number) payload.getOrDefault("clinicalSignificance", 0.5f)).floatValue();
                confidence = ((Number) payload.getOrDefault("confidence", 0.8f)).floatValue();
            }

            clinicalEventsStmt.setFloat(8, significance);
            clinicalEventsStmt.setFloat(9, confidence);
            clinicalEventsStmt.setArray(10, connection.createArrayOf("String",
                event.getDestinations().toArray(new String[0])));
            clinicalEventsStmt.setString(11, convertMetadata(event.getTransformationMetadata()));

            clinicalEventsStmt.addBatch();
        }

        private void addPatternMetric(RoutedEvent event) throws SQLException {
            if (!(event.getOriginalPayload() instanceof PatternEvent)) {
                return;
            }

            PatternEvent pattern = (PatternEvent) event.getOriginalPayload();

            patternMetricsStmt.setString(1, pattern.getId());
            patternMetricsStmt.setString(2, pattern.getPatientId());
            patternMetricsStmt.setString(3, pattern.getPatternType());
            patternMetricsStmt.setTimestamp(4, new Timestamp(pattern.getDetectionTime()));
            patternMetricsStmt.setFloat(5, (float) pattern.getConfidence());
            patternMetricsStmt.setFloat(6, parseSeverityToFloat(pattern.getSeverity()));
            patternMetricsStmt.setArray(7, connection.createArrayOf("String",
                pattern.getInvolvedEvents().toArray(new String[0])));
            patternMetricsStmt.setString(8, convertMetadata(pattern.getPatternMetadata()));

            patternMetricsStmt.addBatch();
        }

        private void addMLPrediction(RoutedEvent event) throws SQLException {
            if (!(event.getOriginalPayload() instanceof MLPrediction)) {
                return;
            }

            MLPrediction prediction = (MLPrediction) event.getOriginalPayload();

            mlPredictionsStmt.setString(1, prediction.getId());
            mlPredictionsStmt.setString(2, prediction.getPatientId());
            mlPredictionsStmt.setString(3, prediction.getModelType());
            mlPredictionsStmt.setFloat(4, (float) prediction.getPrimaryScore());
            mlPredictionsStmt.setFloat(5, prediction.getConfidence() != null ? prediction.getConfidence().floatValue() : 0.0f);
            mlPredictionsStmt.setTimestamp(6, new Timestamp(prediction.getPredictionTime()));
            mlPredictionsStmt.setString(7, prediction.getModelVersion());
            mlPredictionsStmt.setString(8, convertMetadata(prediction.getFeatureImportance()));

            mlPredictionsStmt.addBatch();
        }

        private String convertMetadata(Object metadata) {
            if (metadata == null) return "{}";
            try {
                return metadata.toString();
            } catch (Exception e) {
                return "{}";
            }
        }

        private float parseSeverityToFloat(String severity) {
            if (severity == null) return 0.0f;
            switch (severity.toUpperCase()) {
                case "LOW": return 1.0f;
                case "MODERATE": return 2.0f;
                case "HIGH": return 3.0f;
                case "CRITICAL": return 4.0f;
                default: return 0.0f;
            }
        }

        @Override
        public void flush(boolean endOfInput) throws IOException, InterruptedException {
            try {
                flushBuffer();
            } catch (SQLException e) {
                throw new IOException("Failed to flush ClickHouse buffer", e);
            }
        }

        @Override
        public void close() throws Exception {
            running = false;

            if (flushThread != null) {
                flushThread.interrupt();
                flushThread.join(5000);
            }

            // Final flush
            flushBuffer();

            if (clinicalEventsStmt != null) clinicalEventsStmt.close();
            if (patternMetricsStmt != null) patternMetricsStmt.close();
            if (mlPredictionsStmt != null) mlPredictionsStmt.close();
            if (connection != null) connection.close();

            LOG.info("ClickHouse sink closed");
        }
    }
}
