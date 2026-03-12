package com.clinicalsynthesis.validator.stream;

import com.clinicalsynthesis.validator.model.DeviceReading;
import com.clinicalsynthesis.validator.model.EnrichedDeviceReading;
import com.clinicalsynthesis.validator.model.ValidationResult;
import com.clinicalsynthesis.validator.service.MedicalValidationService;
import com.clinicalsynthesis.validator.service.PatientContextService;
import com.clinicalsynthesis.validator.service.DeadLetterQueueService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.kafka.common.serialization.Serdes;
import org.apache.kafka.streams.StreamsBuilder;
import org.apache.kafka.streams.kstream.Consumed;
import org.apache.kafka.streams.kstream.KStream;
import org.apache.kafka.streams.kstream.Produced;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

/**
 * Validator & Enricher Kafka Streams Topology
 * 
 * This is the core stream processing topology for Stage 1.
 * It implements the exact same validation and enrichment logic as the
 * monolithic PySpark reactor, but in a lightweight, dedicated service.
 * 
 * Flow:
 * 1. Consume raw device data from Global Outbox
 * 2. Validate using medical validation rules
 * 3. Enrich with patient context from Redis cache
 * 4. Route valid data to validated topic
 * 5. Route invalid data to dead letter queue
 */
@Component
public class ValidatorEnricherTopology {
    
    private static final Logger logger = LoggerFactory.getLogger(ValidatorEnricherTopology.class);
    
    // Topic names
    private static final String INPUT_TOPIC = "raw-device-data.v1";
    private static final String OUTPUT_TOPIC = "validated-device-data.v1";
    private static final String DLQ_TOPIC = "failed-validation.v1";
    
    @Autowired
    private MedicalValidationService validationService;

    @Autowired
    private PatientContextService patientContextService;

    @Autowired
    private DeadLetterQueueService dlqService;

    @Autowired
    private ObjectMapper objectMapper;
    
    /**
     * Build the Kafka Streams topology
     */
    @Autowired
    public void buildTopology(StreamsBuilder streamsBuilder) {
        logger.info("Building Validator & Enricher Kafka Streams topology");
        
        // Create the main stream from raw device data
        KStream<String, String> rawDeviceStream = streamsBuilder
            .stream(INPUT_TOPIC, Consumed.with(Serdes.String(), Serdes.String()));
        
        // Branch the stream: valid vs invalid data
        KStream<String, String>[] branches = rawDeviceStream
            .peek((key, value) -> logger.debug("Processing raw device data: key={}, value={}", key, value))
            .branch(
                (key, value) -> isValidDeviceReading(value),  // Valid data
                (key, value) -> true                          // Invalid data (catch-all)
            );
        
        // Process valid data: validate and enrich
        KStream<String, String> validStream = branches[0];
        KStream<String, String> invalidStream = branches[1];
        
        // Transform valid data through validation and enrichment pipeline
        KStream<String, String> enrichedStream = validStream
            .mapValues(this::parseAndValidateDeviceReading)
            .filter((key, value) -> value != null)  // Filter out parsing failures
            .mapValues(this::enrichWithPatientContext)
            .filter((key, value) -> value != null)  // Filter out enrichment failures
            .peek((key, value) -> logger.debug("Enriched device data: key={}, enriched={}", key, value != null));
        
        // Send enriched data to validated topic
        enrichedStream
            .peek((key, value) -> logger.info("📤 Sending enriched data to {}: key={}, data_length={}", OUTPUT_TOPIC, key, value != null ? value.length() : 0))
            .to(OUTPUT_TOPIC, Produced.with(Serdes.String(), Serdes.String()));
        
        // Send invalid data to dead letter queue with comprehensive error handling
        invalidStream
            .peek((key, value) -> {
                logger.warn("Processing invalid data for DLQ: key={}", key);
                // Send to DLQ service for proper categorization and routing
                try {
                    dlqService.sendParsingFailure(value, key, new RuntimeException("Failed initial validation"));
                } catch (Exception e) {
                    logger.error("Failed to send parsing failure to DLQ service", e);
                }
            });
        
        logger.info("Validator & Enricher topology built successfully");
        logger.info("  Input topic: {}", INPUT_TOPIC);
        logger.info("  Output topic: {}", OUTPUT_TOPIC);
        logger.info("  DLQ topic: {}", DLQ_TOPIC);
    }
    
    /**
     * Check if raw device data is valid for processing
     */
    private boolean isValidDeviceReading(String rawData) {
        try {
            if (rawData == null || rawData.trim().isEmpty()) {
                return false;
            }
            
            // Try to parse as JSON
            DeviceReading reading = objectMapper.readValue(rawData, DeviceReading.class);
            
            // Basic validation (same as PySpark isValidReading)
            return reading.isValidReading();
            
        } catch (Exception e) {
            logger.debug("Invalid device reading JSON: {}", e.getMessage());
            return false;
        }
    }
    
    /**
     * Parse and validate device reading
     */
    private String parseAndValidateDeviceReading(String rawData) {
        try {
            // Parse the device reading
            DeviceReading reading = objectMapper.readValue(rawData, DeviceReading.class);
            
            // Perform medical validation (same logic as PySpark)
            ValidationResult validationResult = validationService.validateDeviceReading(reading);
            
            // Only proceed if validation passed
            if (validationResult.getValid()) {
                // Create enriched reading with validation metadata
                EnrichedDeviceReading enriched = new EnrichedDeviceReading(reading);
                enriched.setIsCriticalData(validationResult.getCriticalData());

                return objectMapper.writeValueAsString(enriched);
            } else {
                logger.warn("Device reading failed validation: {} - {}",
                    reading.getDeviceId(), validationResult.getValidationMessage());

                // Send validation failure to DLQ service
                try {
                    dlqService.sendValidationFailure(rawData, validationResult, reading.getDeviceId(), null);
                } catch (Exception dlqError) {
                    logger.error("Failed to send validation failure to DLQ service", dlqError);
                }

                return null;
            }
            
        } catch (Exception e) {
            logger.error("Error parsing/validating device reading: {}", e.getMessage());
            return null;
        }
    }
    
    /**
     * Enrich device reading with patient context
     */
    private String enrichWithPatientContext(String enrichedData) {
        try {
            // Parse the enriched reading
            EnrichedDeviceReading reading = objectMapper.readValue(enrichedData, EnrichedDeviceReading.class);
            
            // Get patient context from cache/service
            if (reading.getPatientId() != null) {
                EnrichedDeviceReading.PatientContext context = 
                    patientContextService.getPatientContext(reading.getPatientId());
                
                if (context != null) {
                    reading.setPatientContext(context);
                    logger.debug("Patient context enriched for patient: {}", reading.getPatientId());
                } else {
                    logger.debug("No patient context found for patient: {}", reading.getPatientId());
                    // Note: We don't send enrichment failures to DLQ for missing patient context
                    // as this is expected for some device readings
                }
            }

            return objectMapper.writeValueAsString(reading);

        } catch (Exception e) {
            logger.error("Error enriching with patient context: {}", e.getMessage());

            // Send enrichment failure to DLQ service for serious errors
            try {
                EnrichedDeviceReading reading = objectMapper.readValue(enrichedData, EnrichedDeviceReading.class);
                dlqService.sendEnrichmentFailure(enrichedData, reading.getDeviceId(), reading.getPatientId(), e);
            } catch (Exception dlqError) {
                logger.error("Failed to send enrichment failure to DLQ service", dlqError);
            }

            return null;
        }
    }
    
    /**
     * Create failure record for dead letter queue
     */
    private String createFailureRecord(String rawData) {
        try {
            // Create a failure record with metadata
            FailureRecord failure = new FailureRecord();
            failure.setOriginalData(rawData);
            failure.setFailureReason("Validation failed");
            failure.setFailureTimestamp(System.currentTimeMillis());
            failure.setProcessingStage("stage1-validator-enricher");
            
            return objectMapper.writeValueAsString(failure);
            
        } catch (Exception e) {
            logger.error("Error creating failure record: {}", e.getMessage());
            // Return a simple failure message if JSON serialization fails
            return "{\"error\":\"Failed to create failure record\",\"originalData\":\"" + 
                   (rawData != null ? rawData.replace("\"", "\\\"") : "null") + "\"}";
        }
    }
    
    /**
     * Failure Record for Dead Letter Queue
     */
    public static class FailureRecord {
        private String originalData;
        private String failureReason;
        private Long failureTimestamp;
        private String processingStage;
        
        // Getters and Setters
        public String getOriginalData() {
            return originalData;
        }
        
        public void setOriginalData(String originalData) {
            this.originalData = originalData;
        }
        
        public String getFailureReason() {
            return failureReason;
        }
        
        public void setFailureReason(String failureReason) {
            this.failureReason = failureReason;
        }
        
        public Long getFailureTimestamp() {
            return failureTimestamp;
        }
        
        public void setFailureTimestamp(Long failureTimestamp) {
            this.failureTimestamp = failureTimestamp;
        }
        
        public String getProcessingStage() {
            return processingStage;
        }
        
        public void setProcessingStage(String processingStage) {
            this.processingStage = processingStage;
        }
    }
}
