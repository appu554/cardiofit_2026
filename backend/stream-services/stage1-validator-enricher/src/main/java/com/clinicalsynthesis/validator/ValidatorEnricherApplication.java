package com.clinicalsynthesis.validator;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.kafka.annotation.EnableKafkaStreams;

/**
 * Stage 1: Validator & Enricher Service
 * 
 * This is a dedicated, lightweight Kafka Streams application that:
 * 1. Consumes raw device data from Global Outbox Service
 * 2. Validates data using medical-grade validation rules
 * 3. Enriches data with patient context from Redis cache
 * 4. Publishes clean, validated events to downstream topics
 * 
 * This service replaces the validation and enrichment portion of the
 * monolithic Spark reactor, providing better performance and maintainability.
 */
@SpringBootApplication
@EnableKafkaStreams
public class ValidatorEnricherApplication {

    public static void main(String[] args) {
        SpringApplication.run(ValidatorEnricherApplication.class, args);
    }
}
