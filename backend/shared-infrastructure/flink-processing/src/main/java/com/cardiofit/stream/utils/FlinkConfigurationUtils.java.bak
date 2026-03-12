package com.cardiofit.stream.utils;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Properties;

/**
 * Flink Configuration Utilities
 * Centralizes configuration management for Flink jobs
 */
public class FlinkConfigurationUtils {

    private static final Logger logger = LoggerFactory.getLogger(FlinkConfigurationUtils.class);

    /**
     * Get Kafka configuration properties
     */
    public static Properties getKafkaProperties() {
        Properties props = new Properties();

        props.setProperty("bootstrap.servers",
            System.getenv().getOrDefault("KAFKA_BOOTSTRAP_SERVERS", "kafka:9092"));
        props.setProperty("group.id", "patient-event-enrichment");
        props.setProperty("auto.offset.reset", "latest");
        props.setProperty("enable.auto.commit", "false");
        props.setProperty("max.poll.records", "1000");
        props.setProperty("fetch.min.bytes", "1");
        props.setProperty("fetch.max.wait.ms", "100");

        logger.info("Kafka configuration: {}", props);
        return props;
    }

    /**
     * Get Neo4j configuration properties
     */
    public static Properties getNeo4jProperties() {
        Properties props = new Properties();

        props.setProperty("neo4j.uri",
            System.getenv().getOrDefault("NEO4J_URI", "bolt://neo4j:7687"));
        props.setProperty("neo4j.username",
            System.getenv().getOrDefault("NEO4J_USERNAME", "neo4j"));
        props.setProperty("neo4j.password",
            System.getenv().getOrDefault("NEO4J_PASSWORD", "kb7password"));

        return props;
    }
}