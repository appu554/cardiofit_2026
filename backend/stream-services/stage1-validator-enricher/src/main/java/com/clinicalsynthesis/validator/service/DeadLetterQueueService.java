package com.clinicalsynthesis.validator.service;

import com.clinicalsynthesis.validator.model.DeviceReading;
import com.clinicalsynthesis.validator.model.ValidationResult;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;

import java.time.Instant;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Dead Letter Queue Service for Stage 1: Validator & Enricher
 * 
 * Handles invalid data that fails validation with comprehensive error logging,
 * categorization, and routing to appropriate DLQ topics for manual review.
 */
@Service
public class DeadLetterQueueService {
    
    private static final Logger logger = LoggerFactory.getLogger(DeadLetterQueueService.class);
    
    // DLQ Topics
    private static final String FAILED_VALIDATION_TOPIC = "failed-validation.v1";
    private static final String CRITICAL_DATA_DLQ_TOPIC = "critical-data-dlq.v1";
    private static final String POISON_MESSAGE_TOPIC = "poison-messages.v1";
    
    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;
    
    @Autowired
    private ObjectMapper objectMapper;
    
    // Metrics
    private final AtomicLong totalDlqMessages = new AtomicLong(0);
    private final AtomicLong validationFailures = new AtomicLong(0);
    private final AtomicLong criticalDataFailures = new AtomicLong(0);
    private final AtomicLong poisonMessages = new AtomicLong(0);
    
    /**
     * Send validation failure to DLQ with detailed error information
     */
    public void sendValidationFailure(String originalData, ValidationResult validationResult, 
                                    String deviceId, Exception error) {
        try {
            DLQRecord dlqRecord = createValidationFailureRecord(
                originalData, validationResult, deviceId, error
            );
            
            String dlqJson = objectMapper.writeValueAsString(dlqRecord);
            
            // Route to appropriate DLQ topic based on criticality
            String topic = dlqRecord.isCriticalData() ? CRITICAL_DATA_DLQ_TOPIC : FAILED_VALIDATION_TOPIC;
            
            kafkaTemplate.send(topic, deviceId, dlqJson);
            
            // Update metrics
            totalDlqMessages.incrementAndGet();
            validationFailures.incrementAndGet();
            if (dlqRecord.isCriticalData()) {
                criticalDataFailures.incrementAndGet();
            }
            
            logger.warn("Validation failure sent to DLQ", 
                       Map.of("deviceId", deviceId,
                             "topic", topic,
                             "errorType", dlqRecord.getErrorType(),
                             "isCritical", dlqRecord.isCriticalData()));
            
        } catch (Exception e) {
            logger.error("Failed to send validation failure to DLQ", e);
        }
    }
    
    /**
     * Send parsing failure to DLQ (for malformed JSON, etc.)
     */
    public void sendParsingFailure(String originalData, String key, Exception error) {
        try {
            DLQRecord dlqRecord = createParsingFailureRecord(originalData, key, error);
            
            String dlqJson = objectMapper.writeValueAsString(dlqRecord);
            
            kafkaTemplate.send(FAILED_VALIDATION_TOPIC, key, dlqJson);
            
            // Update metrics
            totalDlqMessages.incrementAndGet();
            validationFailures.incrementAndGet();
            
            logger.warn("Parsing failure sent to DLQ", 
                       Map.of("key", key,
                             "errorType", dlqRecord.getErrorType(),
                             "errorMessage", dlqRecord.getErrorMessage()));
            
        } catch (Exception e) {
            logger.error("Failed to send parsing failure to DLQ", e);
        }
    }
    
    /**
     * Send poison message to special DLQ (for messages that repeatedly fail)
     */
    public void sendPoisonMessage(String originalData, String key, String reason, int retryCount) {
        try {
            DLQRecord dlqRecord = createPoisonMessageRecord(originalData, key, reason, retryCount);
            
            String dlqJson = objectMapper.writeValueAsString(dlqRecord);
            
            kafkaTemplate.send(POISON_MESSAGE_TOPIC, key, dlqJson);
            
            // Update metrics
            totalDlqMessages.incrementAndGet();
            poisonMessages.incrementAndGet();
            
            logger.error("Poison message sent to DLQ", 
                        Map.of("key", key,
                              "retryCount", retryCount,
                              "reason", reason));
            
        } catch (Exception e) {
            logger.error("Failed to send poison message to DLQ", e);
        }
    }
    
    /**
     * Send enrichment failure to DLQ (when patient context enrichment fails)
     */
    public void sendEnrichmentFailure(String originalData, String deviceId, String patientId, Exception error) {
        try {
            DLQRecord dlqRecord = createEnrichmentFailureRecord(originalData, deviceId, patientId, error);
            
            String dlqJson = objectMapper.writeValueAsString(dlqRecord);
            
            kafkaTemplate.send(FAILED_VALIDATION_TOPIC, deviceId, dlqJson);
            
            // Update metrics
            totalDlqMessages.incrementAndGet();
            
            logger.warn("Enrichment failure sent to DLQ", 
                       Map.of("deviceId", deviceId,
                             "patientId", patientId,
                             "errorType", dlqRecord.getErrorType()));
            
        } catch (Exception e) {
            logger.error("Failed to send enrichment failure to DLQ", e);
        }
    }
    
    /**
     * Create DLQ record for validation failures
     */
    private DLQRecord createValidationFailureRecord(String originalData, ValidationResult validationResult, 
                                                   String deviceId, Exception error) {
        DLQRecord record = new DLQRecord();
        record.setOriginalData(originalData);
        record.setErrorType("VALIDATION_FAILURE");
        record.setErrorMessage(validationResult != null ? validationResult.getValidationMessage() : "Unknown validation error");
        record.setDeviceId(deviceId);
        record.setFailureTimestamp(Instant.now().getEpochSecond());
        record.setProcessingStage("stage1-validator-enricher");
        record.setRetryable(true);
        record.setMaxRetries(3);
        
        // Check if this is critical medical data
        if (validationResult != null) {
            record.setCriticalData(validationResult.getCriticalData() != null ? validationResult.getCriticalData() : false);
            record.setAlertLevel(validationResult.getAlertLevel());
        }
        
        // Add validation details
        Map<String, Object> details = new HashMap<>();
        if (validationResult != null) {
            details.put("alertLevel", validationResult.getAlertLevel());
            details.put("requiresAttention", validationResult.getRequiresAttention());
            details.put("validationRulesVersion", validationResult.getValidationRulesVersion());
        }
        if (error != null) {
            details.put("exceptionMessage", error.getMessage());
            details.put("exceptionType", error.getClass().getSimpleName());
        }
        record.setErrorDetails(details);
        
        return record;
    }
    
    /**
     * Create DLQ record for parsing failures
     */
    private DLQRecord createParsingFailureRecord(String originalData, String key, Exception error) {
        DLQRecord record = new DLQRecord();
        record.setOriginalData(originalData);
        record.setErrorType("PARSING_FAILURE");
        record.setErrorMessage("Failed to parse device reading JSON: " + (error != null ? error.getMessage() : "Unknown error"));
        record.setDeviceId(key);
        record.setFailureTimestamp(Instant.now().getEpochSecond());
        record.setProcessingStage("stage1-validator-enricher");
        record.setRetryable(false); // Parsing failures are usually not retryable
        record.setCriticalData(false);
        
        // Add parsing error details
        Map<String, Object> details = new HashMap<>();
        if (error != null) {
            details.put("exceptionMessage", error.getMessage());
            details.put("exceptionType", error.getClass().getSimpleName());
        }
        details.put("dataLength", originalData != null ? originalData.length() : 0);
        record.setErrorDetails(details);
        
        return record;
    }
    
    /**
     * Create DLQ record for poison messages
     */
    private DLQRecord createPoisonMessageRecord(String originalData, String key, String reason, int retryCount) {
        DLQRecord record = new DLQRecord();
        record.setOriginalData(originalData);
        record.setErrorType("POISON_MESSAGE");
        record.setErrorMessage("Message repeatedly failed processing: " + reason);
        record.setDeviceId(key);
        record.setFailureTimestamp(Instant.now().getEpochSecond());
        record.setProcessingStage("stage1-validator-enricher");
        record.setRetryable(false); // Poison messages should not be retried automatically
        record.setCriticalData(false);
        
        // Add poison message details
        Map<String, Object> details = new HashMap<>();
        details.put("retryCount", retryCount);
        details.put("reason", reason);
        details.put("requiresManualReview", true);
        record.setErrorDetails(details);
        
        return record;
    }
    
    /**
     * Create DLQ record for enrichment failures
     */
    private DLQRecord createEnrichmentFailureRecord(String originalData, String deviceId, String patientId, Exception error) {
        DLQRecord record = new DLQRecord();
        record.setOriginalData(originalData);
        record.setErrorType("ENRICHMENT_FAILURE");
        record.setErrorMessage("Failed to enrich with patient context: " + (error != null ? error.getMessage() : "Unknown error"));
        record.setDeviceId(deviceId);
        record.setPatientId(patientId);
        record.setFailureTimestamp(Instant.now().getEpochSecond());
        record.setProcessingStage("stage1-validator-enricher");
        record.setRetryable(true); // Enrichment failures can be retried
        record.setMaxRetries(2);
        record.setCriticalData(false);
        
        // Add enrichment error details
        Map<String, Object> details = new HashMap<>();
        if (error != null) {
            details.put("exceptionMessage", error.getMessage());
            details.put("exceptionType", error.getClass().getSimpleName());
        }
        details.put("patientId", patientId);
        record.setErrorDetails(details);
        
        return record;
    }
    
    /**
     * Get DLQ metrics
     */
    public Map<String, Long> getDlqMetrics() {
        Map<String, Long> metrics = new HashMap<>();
        metrics.put("totalDlqMessages", totalDlqMessages.get());
        metrics.put("validationFailures", validationFailures.get());
        metrics.put("criticalDataFailures", criticalDataFailures.get());
        metrics.put("poisonMessages", poisonMessages.get());
        return metrics;
    }
    
    /**
     * Check if DLQ service is healthy
     */
    public boolean isHealthy() {
        // Simple health check - could be enhanced with Kafka connectivity check
        return true;
    }
    
    /**
     * DLQ Record Model
     */
    public static class DLQRecord {
        private String originalData;
        private String errorType;
        private String errorMessage;
        private String deviceId;
        private String patientId;
        private Long failureTimestamp;
        private String processingStage;
        private boolean retryable;
        private int maxRetries;
        private boolean criticalData;
        private String alertLevel;
        private Map<String, Object> errorDetails;
        
        // Getters and Setters
        public String getOriginalData() { return originalData; }
        public void setOriginalData(String originalData) { this.originalData = originalData; }
        
        public String getErrorType() { return errorType; }
        public void setErrorType(String errorType) { this.errorType = errorType; }
        
        public String getErrorMessage() { return errorMessage; }
        public void setErrorMessage(String errorMessage) { this.errorMessage = errorMessage; }
        
        public String getDeviceId() { return deviceId; }
        public void setDeviceId(String deviceId) { this.deviceId = deviceId; }
        
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }
        
        public Long getFailureTimestamp() { return failureTimestamp; }
        public void setFailureTimestamp(Long failureTimestamp) { this.failureTimestamp = failureTimestamp; }
        
        public String getProcessingStage() { return processingStage; }
        public void setProcessingStage(String processingStage) { this.processingStage = processingStage; }
        
        public boolean isRetryable() { return retryable; }
        public void setRetryable(boolean retryable) { this.retryable = retryable; }
        
        public int getMaxRetries() { return maxRetries; }
        public void setMaxRetries(int maxRetries) { this.maxRetries = maxRetries; }
        
        public boolean isCriticalData() { return criticalData; }
        public void setCriticalData(boolean criticalData) { this.criticalData = criticalData; }
        
        public String getAlertLevel() { return alertLevel; }
        public void setAlertLevel(String alertLevel) { this.alertLevel = alertLevel; }
        
        public Map<String, Object> getErrorDetails() { return errorDetails; }
        public void setErrorDetails(Map<String, Object> errorDetails) { this.errorDetails = errorDetails; }
    }
}
