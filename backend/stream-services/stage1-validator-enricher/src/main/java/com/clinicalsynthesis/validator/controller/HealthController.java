package com.clinicalsynthesis.validator.controller;

import com.clinicalsynthesis.validator.service.MedicalValidationService;
import com.clinicalsynthesis.validator.service.PatientContextService;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

/**
 * Health Check Controller for Stage 1: Validator & Enricher Service
 */
@RestController
@RequestMapping("/health")
public class HealthController {
    
    @Autowired
    private MedicalValidationService validationService;
    
    @Autowired
    private PatientContextService patientContextService;
    
    /**
     * Overall health check
     */
    @GetMapping
    public ResponseEntity<Map<String, Object>> health() {
        Map<String, Object> health = new HashMap<>();
        
        try {
            // Check validation service
            boolean validationHealthy = validationService.isValidationSystemHealthy();
            
            // Check patient context service
            boolean patientContextHealthy = patientContextService.isHealthy();
            
            // Overall status
            boolean overallHealthy = validationHealthy && patientContextHealthy;
            
            health.put("status", overallHealthy ? "UP" : "DOWN");
            health.put("service", "stage1-validator-enricher");
            health.put("port", 8041);
            health.put("timestamp", System.currentTimeMillis());
            
            // Component health details
            Map<String, Object> components = new HashMap<>();
            components.put("validation", validationHealthy ? "UP" : "DOWN");
            components.put("patientContext", patientContextHealthy ? "UP" : "DOWN");
            health.put("components", components);
            
            return ResponseEntity.ok(health);
            
        } catch (Exception e) {
            health.put("status", "DOWN");
            health.put("error", e.getMessage());
            return ResponseEntity.status(503).body(health);
        }
    }
    
    /**
     * Validation service health check
     */
    @GetMapping("/validation")
    public ResponseEntity<Map<String, Object>> validationHealth() {
        Map<String, Object> health = new HashMap<>();
        
        try {
            boolean healthy = validationService.isValidationSystemHealthy();
            
            health.put("status", healthy ? "UP" : "DOWN");
            health.put("component", "medical-validation");
            health.put("timestamp", System.currentTimeMillis());
            
            return ResponseEntity.ok(health);
            
        } catch (Exception e) {
            health.put("status", "DOWN");
            health.put("error", e.getMessage());
            return ResponseEntity.status(503).body(health);
        }
    }
    
    /**
     * Patient context service health check
     */
    @GetMapping("/patient-context")
    public ResponseEntity<Map<String, Object>> patientContextHealth() {
        Map<String, Object> health = new HashMap<>();
        
        try {
            boolean healthy = patientContextService.isHealthy();
            String cacheStats = patientContextService.getCacheStats();
            
            health.put("status", healthy ? "UP" : "DOWN");
            health.put("component", "patient-context");
            health.put("cacheStats", cacheStats);
            health.put("timestamp", System.currentTimeMillis());
            
            return ResponseEntity.ok(health);
            
        } catch (Exception e) {
            health.put("status", "DOWN");
            health.put("error", e.getMessage());
            return ResponseEntity.status(503).body(health);
        }
    }
}
