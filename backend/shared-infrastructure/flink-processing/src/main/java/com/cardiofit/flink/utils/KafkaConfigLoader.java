package com.cardiofit.flink.utils;

import java.util.Properties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Kafka Configuration Loader for local Docker-based Kafka cluster
 * Provides configuration for both internal (container) and external (host) access
 */
public class KafkaConfigLoader {
    private static final Logger LOG = LoggerFactory.getLogger(KafkaConfigLoader.class);

    // Local Kafka cluster endpoints (internal Docker network)
    private static final String INTERNAL_BOOTSTRAP_SERVERS = "kafka:29092";
    // External access from host machine
    private static final String EXTERNAL_BOOTSTRAP_SERVERS = "localhost:9092";

    // Schema Registry URL
    private static final String INTERNAL_SCHEMA_REGISTRY = "http://schema-registry:8081";
    private static final String EXTERNAL_SCHEMA_REGISTRY = "http://localhost:8081";

    // Google Cloud Healthcare API Configuration
    private static final String GOOGLE_CLOUD_PROJECT_ID = "cardiofit-905a8";
    private static final String GOOGLE_CLOUD_LOCATION = "asia-south1";
    private static final String GOOGLE_CLOUD_DATASET_ID = "clinical-synthesis-hub";
    private static final String GOOGLE_CLOUD_FHIR_STORE_ID = "fhir-store";
    private static final String GOOGLE_CLOUD_CREDENTIALS_PATH =
        "/app/credentials/google-credentials.json"; // Docker path

    // Neo4j Configuration (from docker-compose.hybrid-kafka.yml)
    private static final String NEO4J_URI = "bolt://neo4j:7687"; // Internal Docker
    private static final String NEO4J_EXTERNAL_URI = "bolt://localhost:7687"; // External (mapped port)
    private static final String NEO4J_USERNAME = "neo4j";
    private static final String NEO4J_PASSWORD = "CardioFit2024!"; // From docker-compose

    private KafkaConfigLoader() {
        // Utility class, prevent instantiation
    }

    /**
     * Get consumer configuration for Flink running in Docker (JSON-based)
     */
    public static Properties getConsumerConfig(String groupId) {
        Properties props = new Properties();
        props.setProperty("bootstrap.servers", getBootstrapServers());
        props.setProperty("group.id", groupId);
        props.setProperty("enable.auto.commit", "false");
        props.setProperty("auto.offset.reset", "latest");

        // Performance optimizations for local cluster
        props.setProperty("fetch.min.bytes", "1048576"); // 1MB
        props.setProperty("fetch.max.wait.ms", "500");
        props.setProperty("max.poll.records", "5000");
        props.setProperty("session.timeout.ms", "30000");
        props.setProperty("heartbeat.interval.ms", "10000");
        props.setProperty("max.partition.fetch.bytes", "10485760"); // 10MB

        // NOTE: Flink 2.x uses DeserializationSchema API, not Kafka deserializers
        // Do NOT set key.deserializer or value.deserializer properties here
        // Use .setValueOnlyDeserializer() in KafkaSource builder instead

        LOG.info("Created Kafka consumer config for group: {}", groupId);
        return props;
    }

    /**
     * Get producer configuration for Flink running in Docker (JSON-based)
     */
    public static Properties getProducerConfig() {
        Properties props = new Properties();
        props.setProperty("bootstrap.servers", getBootstrapServers());

        // Producer optimizations for high throughput
        props.setProperty("compression.type", "snappy");
        props.setProperty("batch.size", "32768"); // 32KB
        props.setProperty("linger.ms", "100");
        props.setProperty("buffer.memory", "33554432"); // 32MB
        props.setProperty("acks", "all");
        props.setProperty("enable.idempotence", "true");
        props.setProperty("retries", "2147483647");
        props.setProperty("max.in.flight.requests.per.connection", "5");
        props.setProperty("delivery.timeout.ms", "120000");

        // JSON serializer (Schema Registry not required)
        props.setProperty("key.serializer",
            "org.apache.kafka.common.serialization.StringSerializer");
        props.setProperty("value.serializer",
            "org.apache.kafka.common.serialization.StringSerializer");

        LOG.info("Created Kafka producer config");
        return props;
    }

    /**
     * Get consumer configuration for external development (running from IDE)
     */
    public static Properties getExternalConsumerConfig(String groupId) {
        Properties props = getConsumerConfig(groupId);
        props.setProperty("bootstrap.servers", EXTERNAL_BOOTSTRAP_SERVERS);
        LOG.info("Created external Kafka consumer config for group: {}", groupId);
        return props;
    }

    /**
     * Get producer configuration for external development (running from IDE)
     */
    public static Properties getExternalProducerConfig() {
        Properties props = getProducerConfig();
        props.setProperty("bootstrap.servers", EXTERNAL_BOOTSTRAP_SERVERS);
        LOG.info("Created external Kafka producer config");
        return props;
    }

    /**
     * Get consumer configuration with custom properties
     */
    public static Properties getConsumerConfig(String groupId, Properties customProps) {
        Properties props = getConsumerConfig(groupId);
        props.putAll(customProps);
        return props;
    }

    /**
     * Get bootstrap servers string. Checks KAFKA_BOOTSTRAP_SERVERS env var first,
     * then falls back to Docker/local defaults.
     */
    public static String getBootstrapServers() {
        String envServers = System.getenv("KAFKA_BOOTSTRAP_SERVERS");
        if (envServers != null && !envServers.isEmpty()) {
            return envServers;
        }
        return isRunningInDocker() ? INTERNAL_BOOTSTRAP_SERVERS : EXTERNAL_BOOTSTRAP_SERVERS;
    }

    /**
     * Determine if running inside Docker container
     */
    public static boolean isRunningInDocker() {
        // Check if running in container by looking for Docker-specific files
        return new java.io.File("/.dockerenv").exists() ||
               System.getenv("DOCKER_CONTAINER") != null;
    }

    /**
     * Get global parameters for Flink configuration
     */
    public static org.apache.flink.configuration.Configuration getGlobalParameters() {
        org.apache.flink.configuration.Configuration config = new org.apache.flink.configuration.Configuration();

        // Set healthcare-specific configuration
        config.setString("kafka.bootstrap.servers", getBootstrapServers());
        config.setString("kafka.schema.registry.url",
                         isRunningInDocker() ? INTERNAL_SCHEMA_REGISTRY : EXTERNAL_SCHEMA_REGISTRY);
        config.setString("environment.mode", "production");
        config.setString("healthcare.compliance.mode", "hipaa");

        return config;
    }

    /**
     * Get appropriate consumer config based on environment
     */
    public static Properties getAutoConsumerConfig(String groupId) {
        if (isRunningInDocker()) {
            return getConsumerConfig(groupId);
        } else {
            return getExternalConsumerConfig(groupId);
        }
    }

    /**
     * Get appropriate producer config based on environment
     */
    public static Properties getAutoProducerConfig() {
        if (isRunningInDocker()) {
            return getProducerConfig();
        } else {
            return getExternalProducerConfig();
        }
    }

    /**
     * Get producer config for KafkaSink with RecordSerializer
     * IMPORTANT: When using KafkaSink.setRecordSerializer(), do NOT include
     * key/value serializers in producer config - Flink uses ByteArraySerializer
     * internally and RecordSerializer handles the actual serialization.
     */
    public static Properties getProducerConfigForSink() {
        Properties props = getAutoProducerConfig();

        // Remove serializers that conflict with RecordSerializer
        props.remove("key.serializer");
        props.remove("value.serializer");

        LOG.info("Created Kafka producer config for KafkaSink (no serializers)");
        return props;
    }

    /**
     * Get transactional producer configuration for EXACTLY_ONCE semantics
     * Required for the Hybrid Kafka Topic Architecture
     */
    public static Properties getTransactionalProducerConfig(String transactionalId) {
        Properties props = getAutoProducerConfig();

        // Enable transactional producer
        props.setProperty("enable.idempotence", "true");
        props.setProperty("transactional.id", transactionalId);

        // Transaction timeout (5 minutes)
        props.setProperty("transaction.timeout.ms", "300000");

        // Stronger consistency guarantees
        props.setProperty("acks", "all");
        props.setProperty("retries", "2147483647");
        props.setProperty("max.in.flight.requests.per.connection", "5");

        // Optimizations for transactional workloads
        props.setProperty("batch.size", "65536");  // 64KB
        props.setProperty("linger.ms", "100");
        props.setProperty("compression.type", "snappy");

        LOG.info("🔐 Configured transactional producer: {}", transactionalId);
        return props;
    }

    // ========== Google Cloud Healthcare API Configuration Getters ==========

    /**
     * Get Google Cloud project ID for Healthcare API
     */
    public static String getGoogleCloudProjectId() {
        String projectId = System.getenv("GOOGLE_CLOUD_PROJECT_ID");
        return projectId != null ? projectId : GOOGLE_CLOUD_PROJECT_ID;
    }

    /**
     * Get Google Cloud location for Healthcare API
     */
    public static String getGoogleCloudLocation() {
        String location = System.getenv("GOOGLE_CLOUD_LOCATION");
        return location != null ? location : GOOGLE_CLOUD_LOCATION;
    }

    /**
     * Get Google Cloud dataset ID for Healthcare API
     */
    public static String getGoogleCloudDatasetId() {
        String datasetId = System.getenv("GOOGLE_CLOUD_DATASET_ID");
        return datasetId != null ? datasetId : GOOGLE_CLOUD_DATASET_ID;
    }

    /**
     * Get Google Cloud FHIR store ID for Healthcare API
     */
    public static String getGoogleCloudFhirStoreId() {
        String fhirStoreId = System.getenv("GOOGLE_CLOUD_FHIR_STORE_ID");
        return fhirStoreId != null ? fhirStoreId : GOOGLE_CLOUD_FHIR_STORE_ID;
    }

    /**
     * Get Google Cloud credentials path for Healthcare API
     * Returns Docker path if running in container, otherwise local path
     */
    public static String getGoogleCloudCredentialsPath() {
        String credentialsPath = System.getenv("GOOGLE_APPLICATION_CREDENTIALS");
        if (credentialsPath != null) {
            return credentialsPath;
        }

        // Return appropriate path based on environment
        if (isRunningInDocker()) {
            return GOOGLE_CLOUD_CREDENTIALS_PATH;
        } else {
            // For local development, use patient-service credentials path
            return "/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json";
        }
    }

    // ========== Neo4j Configuration Getters ==========

    /**
     * Get Neo4j URI based on environment (Docker vs local)
     */
    public static String getNeo4jUri() {
        String uri = System.getenv("NEO4J_URI");
        if (uri != null) {
            return uri;
        }
        return isRunningInDocker() ? NEO4J_URI : NEO4J_EXTERNAL_URI;
    }

    /**
     * Get Neo4j username
     */
    public static String getNeo4jUsername() {
        String username = System.getenv("NEO4J_USERNAME");
        return username != null ? username : NEO4J_USERNAME;
    }

    /**
     * Get Neo4j password
     */
    public static String getNeo4jPassword() {
        String password = System.getenv("NEO4J_PASSWORD");
        return password != null ? password : NEO4J_PASSWORD;
    }
}