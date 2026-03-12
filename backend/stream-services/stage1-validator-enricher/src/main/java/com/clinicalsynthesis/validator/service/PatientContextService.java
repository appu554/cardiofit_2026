package com.clinicalsynthesis.validator.service;

import com.clinicalsynthesis.validator.model.EnrichedDeviceReading.PatientContext;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.data.redis.core.RedisTemplate;
import org.springframework.stereotype.Service;
import org.springframework.web.client.RestTemplate;

import java.time.Instant;
import java.util.concurrent.TimeUnit;

/**
 * Patient Context Service
 * 
 * Handles patient context enrichment using Redis cache and fallback to Patient Service.
 * This service provides the same patient context enrichment as the PySpark pipeline
 * but optimized for real-time stream processing.
 */
@Service
public class PatientContextService {
    
    private static final Logger logger = LoggerFactory.getLogger(PatientContextService.class);
    
    private static final String PATIENT_CACHE_PREFIX = "patient:context:";
    private static final long CACHE_TTL_SECONDS = 300; // 5 minutes
    
    @Autowired
    private RedisTemplate<String, String> redisTemplate;
    
    @Autowired
    private RestTemplate restTemplate;
    
    @Autowired
    private ObjectMapper objectMapper;
    
    // Patient Service URL - should be configured via application.properties
    private String patientServiceUrl = "http://patient-service:8003/api/v1/patient";
    
    /**
     * Get patient context with Redis caching
     * 
     * @param patientId The patient ID to look up
     * @return PatientContext or null if not found
     */
    public PatientContext getPatientContext(String patientId) {
        if (patientId == null || patientId.trim().isEmpty()) {
            logger.debug("Patient ID is null or empty, skipping context enrichment");
            return null;
        }
        
        try {
            // Try Redis cache first
            PatientContext cachedContext = getFromCache(patientId);
            if (cachedContext != null) {
                logger.debug("Patient context cache HIT for patient: {}", patientId);
                return cachedContext;
            }
            
            logger.debug("Patient context cache MISS for patient: {}", patientId);
            
            // Fallback to Patient Service API
            PatientContext context = fetchFromPatientService(patientId);
            if (context != null) {
                // Cache the result
                cachePatientContext(patientId, context);
                logger.debug("Patient context fetched and cached for patient: {}", patientId);
                return context;
            }
            
            logger.warn("Patient context not found for patient: {}", patientId);
            return null;
            
        } catch (Exception e) {
            logger.error("Error getting patient context for patient {}: {}", patientId, e.getMessage(), e);
            return null;
        }
    }
    
    /**
     * Get patient context from Redis cache
     */
    private PatientContext getFromCache(String patientId) {
        try {
            String cacheKey = PATIENT_CACHE_PREFIX + patientId;
            String cachedJson = redisTemplate.opsForValue().get(cacheKey);
            
            if (cachedJson != null) {
                return objectMapper.readValue(cachedJson, PatientContext.class);
            }
            
            return null;
            
        } catch (Exception e) {
            logger.error("Error reading patient context from cache for patient {}: {}", patientId, e.getMessage());
            return null;
        }
    }
    
    /**
     * Cache patient context in Redis
     */
    private void cachePatientContext(String patientId, PatientContext context) {
        try {
            String cacheKey = PATIENT_CACHE_PREFIX + patientId;
            String contextJson = objectMapper.writeValueAsString(context);
            
            redisTemplate.opsForValue().set(cacheKey, contextJson, CACHE_TTL_SECONDS, TimeUnit.SECONDS);
            
            logger.debug("Patient context cached for patient: {} with TTL: {} seconds", patientId, CACHE_TTL_SECONDS);
            
        } catch (Exception e) {
            logger.error("Error caching patient context for patient {}: {}", patientId, e.getMessage());
        }
    }
    
    /**
     * Fetch patient context from Patient Service API
     */
    private PatientContext fetchFromPatientService(String patientId) {
        try {
            String url = patientServiceUrl + "/" + patientId;
            logger.debug("Fetching patient context from: {}", url);
            
            // Make HTTP request to Patient Service
            String response = restTemplate.getForObject(url, String.class);
            
            if (response != null) {
                // Parse the response and create PatientContext
                JsonNode patientData = objectMapper.readTree(response);
                return parsePatientResponse(patientData);
            }
            
            return null;
            
        } catch (Exception e) {
            logger.error("Error fetching patient context from service for patient {}: {}", patientId, e.getMessage());
            return null;
        }
    }
    
    /**
     * Parse patient service response into PatientContext
     */
    private PatientContext parsePatientResponse(JsonNode patientData) {
        try {
            PatientContext context = new PatientContext();
            
            // Extract basic patient information
            context.setPatientId(getTextValue(patientData, "id"));
            context.setPatientName(getTextValue(patientData, "name"));
            
            // Extract age (calculate from birthDate if needed)
            Integer age = getIntValue(patientData, "age");
            if (age == null) {
                String birthDate = getTextValue(patientData, "birthDate");
                if (birthDate != null) {
                    age = calculateAgeFromBirthDate(birthDate);
                }
            }
            context.setAge(age);
            
            // Extract gender
            context.setGender(getTextValue(patientData, "gender"));
            
            // Extract medical conditions
            JsonNode conditions = patientData.get("medicalConditions");
            if (conditions != null) {
                context.setMedicalConditions(conditions);
            }
            
            // Set cache timestamp
            context.setCacheTimestamp(Instant.now().getEpochSecond());
            
            return context;
            
        } catch (Exception e) {
            logger.error("Error parsing patient response: {}", e.getMessage());
            return null;
        }
    }
    
    /**
     * Helper method to safely extract text values from JSON
     */
    private String getTextValue(JsonNode node, String fieldName) {
        JsonNode field = node.get(fieldName);
        return field != null && !field.isNull() ? field.asText() : null;
    }
    
    /**
     * Helper method to safely extract integer values from JSON
     */
    private Integer getIntValue(JsonNode node, String fieldName) {
        JsonNode field = node.get(fieldName);
        return field != null && !field.isNull() ? field.asInt() : null;
    }
    
    /**
     * Calculate age from birth date string (simplified implementation)
     */
    private Integer calculateAgeFromBirthDate(String birthDate) {
        try {
            // This is a simplified implementation
            // In production, you'd want proper date parsing and calculation
            // For now, return null to indicate age calculation is not available
            logger.debug("Age calculation from birth date not implemented: {}", birthDate);
            return null;
        } catch (Exception e) {
            logger.error("Error calculating age from birth date {}: {}", birthDate, e.getMessage());
            return null;
        }
    }
    
    /**
     * Invalidate patient context cache
     */
    public void invalidatePatientCache(String patientId) {
        try {
            String cacheKey = PATIENT_CACHE_PREFIX + patientId;
            redisTemplate.delete(cacheKey);
            logger.debug("Patient context cache invalidated for patient: {}", patientId);
        } catch (Exception e) {
            logger.error("Error invalidating patient cache for patient {}: {}", patientId, e.getMessage());
        }
    }
    
    /**
     * Check if patient context service is healthy
     */
    public boolean isHealthy() {
        try {
            // Test Redis connectivity
            redisTemplate.opsForValue().get("health:check");
            
            // Test Patient Service connectivity (optional)
            // You could add a health check endpoint call here
            
            return true;
        } catch (Exception e) {
            logger.error("Patient context service health check failed: {}", e.getMessage());
            return false;
        }
    }
    
    /**
     * Get cache statistics
     */
    public String getCacheStats() {
        try {
            // This is a simplified implementation
            // In production, you'd want proper cache statistics
            return "Patient context cache operational";
        } catch (Exception e) {
            logger.error("Error getting cache stats: {}", e.getMessage());
            return "Cache stats unavailable";
        }
    }
}
