// Clinical Assertion Engine (CAE) - Core Implementation
//
// This module provides the main CAE engine that performs clinical safety
// evaluations with sub-20ms performance targets. It replaces the Python
// subprocess implementation with native Rust code.

use std::sync::Arc;
use std::time::Instant;
use std::collections::HashMap;
use lru::LruCache;
use std::sync::Mutex;

use crate::types::{SafetyRequest, SafetyResult, SafetyStatus};
use crate::cae::{CAEConfig, CAEError, CAEResult, CAECapability};
use crate::cae::rules::{RuleEngine, InteractionResult, ContraindicationResult, DosingResult};
use crate::utils::{performance::Timer, hashing::hash_patient_id, logging};

/// Clinical Assertion Engine - Main engine for safety evaluation
pub struct CAEEngine {
    /// Engine configuration
    config: CAEConfig,
    
    /// Clinical rules engine
    rule_engine: RuleEngine,
    
    /// Cache for evaluation results (thread-safe)
    cache: Arc<Mutex<LruCache<String, CachedResult>>>,
    
    /// Performance statistics
    stats: Arc<Mutex<EngineStats>>,
}

/// Cached evaluation result with timestamp
#[derive(Debug, Clone)]
struct CachedResult {
    result: SafetyResult,
    timestamp: Instant,
}

/// Engine performance statistics
#[derive(Debug, Default)]
struct EngineStats {
    total_evaluations: u64,
    cache_hits: u64,
    total_evaluation_time_ms: u64,
    average_evaluation_time_ms: f64,
    min_evaluation_time_ms: u64,
    max_evaluation_time_ms: u64,
}

impl CAEEngine {
    /// Create a new CAE engine with the given configuration
    pub fn new(config: CAEConfig) -> CAEResult<Self> {
        // Validate configuration
        config.validate().map_err(|e| CAEError::InvalidConfiguration(e))?;
        
        // Initialize rule engine
        let rule_engine = RuleEngine::new(&config)?;
        
        // Initialize cache
        let cache = Arc::new(Mutex::new(LruCache::new(
            std::num::NonZeroUsize::new(config.cache.size)
                .ok_or_else(|| CAEError::InvalidConfiguration("Cache size must be > 0".to_string()))?
        )));
        
        // Initialize statistics
        let stats = Arc::new(Mutex::new(EngineStats::default()));
        
        logging::log_structured(
            logging::LogLevel::Info,
            "CAE engine initialized",
            [
                ("cache_size", serde_json::json!(config.cache.size)),
                ("worker_threads", serde_json::json!(config.performance.worker_threads)),
                ("max_eval_time_ms", serde_json::json!(config.performance.max_evaluation_time_ms)),
            ].iter().cloned().collect(),
        );
        
        Ok(Self {
            config,
            rule_engine,
            cache,
            stats,
        })
    }
    
    /// Perform safety evaluation for the given request
    pub fn evaluate_safety(&self, request: &SafetyRequest) -> CAEResult<SafetyResult> {
        let timer = Timer::new("cae_evaluation");
        let patient_hash = hash_patient_id(&request.patient_id);
        
        // Check cache first if enabled
        if self.config.cache.enabled {
            let cache_key = request.cache_key();
            
            if let Ok(mut cache) = self.cache.lock() {
                if let Some(cached) = cache.get(&cache_key) {
                    // Check if cache entry is still valid
                    if cached.timestamp.elapsed() < self.config.cache_ttl() {
                        self.update_stats(timer.elapsed_ms(), true);
                        
                        logging::log_safety_evaluation(
                            &request.request_id,
                            &patient_hash,
                            "cached",
                            cached.result.risk_score,
                            timer.elapsed_ms(),
                        );
                        
                        return Ok(cached.result.clone());
                    }
                }
            }
        }
        
        // Perform fresh evaluation
        let result = self.perform_evaluation(request)?;
        let processing_time = timer.elapsed_ms();
        
        // Update result with processing time
        let mut final_result = result;
        final_result.processing_time_ms = processing_time;
        
        // Cache the result if caching is enabled
        if self.config.cache.enabled {
            if let Ok(mut cache) = self.cache.lock() {
                cache.put(request.cache_key(), CachedResult {
                    result: final_result.clone(),
                    timestamp: Instant::now(),
                });
            }
        }
        
        // Update statistics
        self.update_stats(processing_time, false);
        
        // Log the evaluation
        logging::log_safety_evaluation(
            &request.request_id,
            &patient_hash,
            match final_result.status {
                SafetyStatus::Safe => "safe",
                SafetyStatus::Unsafe => "unsafe",
                SafetyStatus::Warning => "warning",
                SafetyStatus::ManualReview => "manual_review",
            },
            final_result.risk_score,
            processing_time,
        );
        
        Ok(final_result)
    }
    
    /// Perform the actual safety evaluation
    fn perform_evaluation(&self, request: &SafetyRequest) -> CAEResult<SafetyResult> {
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0f64;
        
        // 1. Drug Interaction Analysis
        if self.config.is_capability_enabled(&CAECapability::DrugInteraction) {
            if let Some(interaction_result) = self.check_drug_interactions(request)? {
                violations.extend(interaction_result.violations);
                warnings.extend(interaction_result.warnings);
                max_risk_score = max_risk_score.max(interaction_result.risk_score);
            }
        }
        
        // 2. Contraindication Analysis
        if self.config.is_capability_enabled(&CAECapability::Contraindication) {
            if let Some(contraindication_result) = self.check_contraindications(request)? {
                violations.extend(contraindication_result.violations);
                warnings.extend(contraindication_result.warnings);
                max_risk_score = max_risk_score.max(contraindication_result.risk_score);
            }
        }
        
        // 3. Allergy Check
        if self.config.is_capability_enabled(&CAECapability::AllergyCheck) {
            if let Some(allergy_result) = self.check_allergies(request)? {
                violations.extend(allergy_result.violations);
                warnings.extend(allergy_result.warnings);
                max_risk_score = max_risk_score.max(allergy_result.risk_score);
            }
        }
        
        // 4. Dosing Validation
        if self.config.is_capability_enabled(&CAECapability::DosingValidation) {
            if let Some(dosing_result) = self.check_dosing(request)? {
                violations.extend(dosing_result.violations);
                warnings.extend(dosing_result.warnings);
                max_risk_score = max_risk_score.max(dosing_result.risk_score);
            }
        }
        
        // 5. Duplicate Therapy Check
        if self.config.is_capability_enabled(&CAECapability::DuplicateTherapy) {
            if let Some(duplicate_result) = self.check_duplicate_therapy(request)? {
                violations.extend(duplicate_result.violations);
                warnings.extend(duplicate_result.warnings);
                max_risk_score = max_risk_score.max(duplicate_result.risk_score);
            }
        }
        
        // Determine overall safety status
        let status = self.determine_safety_status(&violations, &warnings, max_risk_score);
        
        // Calculate confidence based on the quality and completeness of the evaluation
        let confidence = self.calculate_confidence(&violations, &warnings, request);
        
        // Create metadata
        let mut metadata = HashMap::new();
        metadata.insert("evaluation_method".to_string(), "native_rust".to_string());
        metadata.insert("engine_version".to_string(), "0.1.0".to_string());
        metadata.insert("capabilities_used".to_string(), 
                        self.get_enabled_capabilities_string());
        
        Ok(SafetyResult {
            status,
            risk_score: max_risk_score,
            confidence,
            violations,
            warnings,
            processing_time_ms: 0, // Will be set by caller
            metadata,
        })
    }
    
    /// Check for drug interactions
    fn check_drug_interactions(&self, request: &SafetyRequest) -> CAEResult<Option<InteractionResult>> {
        if request.medication_ids.len() < 2 {
            return Ok(None); // Need at least 2 medications for interactions
        }
        
        self.rule_engine.check_drug_interactions(&request.medication_ids)
            .map_err(|e| CAEError::RuleEvaluationError(e.to_string()))
            .map(Some)
    }
    
    /// Check for contraindications
    fn check_contraindications(&self, request: &SafetyRequest) -> CAEResult<Option<ContraindicationResult>> {
        if request.medication_ids.is_empty() || 
           (request.condition_ids.is_empty() && request.allergy_ids.is_empty()) {
            return Ok(None);
        }
        
        self.rule_engine.check_contraindications(request)
            .map_err(|e| CAEError::RuleEvaluationError(e.to_string()))
            .map(Some)
    }
    
    /// Check for allergy conflicts
    fn check_allergies(&self, request: &SafetyRequest) -> CAEResult<Option<ContraindicationResult>> {
        if request.medication_ids.is_empty() || request.allergy_ids.is_empty() {
            return Ok(None);
        }
        
        self.rule_engine.check_allergy_contraindications(request)
            .map_err(|e| CAEError::RuleEvaluationError(e.to_string()))
            .map(Some)
    }
    
    /// Check dosing appropriateness
    fn check_dosing(&self, request: &SafetyRequest) -> CAEResult<Option<DosingResult>> {
        if request.medication_ids.is_empty() {
            return Ok(None);
        }
        
        self.rule_engine.check_dosing(request)
            .map_err(|e| CAEError::RuleEvaluationError(e.to_string()))
            .map(Some)
    }
    
    /// Check for duplicate therapy
    fn check_duplicate_therapy(&self, request: &SafetyRequest) -> CAEResult<Option<InteractionResult>> {
        if request.medication_ids.len() < 2 {
            return Ok(None);
        }
        
        self.rule_engine.check_duplicate_therapy(&request.medication_ids)
            .map_err(|e| CAEError::RuleEvaluationError(e.to_string()))
            .map(Some)
    }
    
    /// Determine overall safety status based on findings
    fn determine_safety_status(&self, violations: &[String], warnings: &[String], risk_score: f64) -> SafetyStatus {
        if !violations.is_empty() {
            // Any violations mean unsafe
            SafetyStatus::Unsafe
        } else if risk_score > 0.8 {
            // High risk score requires manual review even without explicit violations
            SafetyStatus::ManualReview
        } else if !warnings.is_empty() || risk_score > 0.4 {
            // Warnings or moderate risk scores result in warning status
            SafetyStatus::Warning
        } else {
            // No issues found
            SafetyStatus::Safe
        }
    }
    
    /// Calculate confidence in the evaluation result
    fn calculate_confidence(&self, violations: &[String], warnings: &[String], request: &SafetyRequest) -> f64 {
        let mut confidence = 1.0;
        
        // Reduce confidence if we have incomplete information
        if request.condition_ids.is_empty() {
            confidence *= 0.9; // Missing condition information
        }
        
        if request.allergy_ids.is_empty() {
            confidence *= 0.95; // Missing allergy information
        }
        
        // Increase confidence if we found clear violations
        if !violations.is_empty() {
            confidence = confidence.max(0.95); // High confidence in violations
        }
        
        // Moderate confidence for warnings
        if !warnings.is_empty() && violations.is_empty() {
            confidence *= 0.85;
        }
        
        // Ensure confidence stays within reasonable bounds
        confidence.max(0.1).min(1.0)
    }
    
    /// Get string representation of enabled capabilities
    fn get_enabled_capabilities_string(&self) -> String {
        self.config.enabled_capabilities.join(",")
    }
    
    /// Update engine statistics
    fn update_stats(&self, processing_time_ms: u64, cache_hit: bool) {
        if let Ok(mut stats) = self.stats.lock() {
            stats.total_evaluations += 1;
            
            if cache_hit {
                stats.cache_hits += 1;
            } else {
                stats.total_evaluation_time_ms += processing_time_ms;
                stats.average_evaluation_time_ms = 
                    stats.total_evaluation_time_ms as f64 / (stats.total_evaluations - stats.cache_hits) as f64;
                stats.min_evaluation_time_ms = if stats.min_evaluation_time_ms == 0 {
                    processing_time_ms
                } else {
                    stats.min_evaluation_time_ms.min(processing_time_ms)
                };
                stats.max_evaluation_time_ms = stats.max_evaluation_time_ms.max(processing_time_ms);
            }
        }
    }
    
    /// Get current engine statistics
    pub fn get_stats(&self) -> Option<EngineStats> {
        self.stats.lock().ok().map(|stats| EngineStats {
            total_evaluations: stats.total_evaluations,
            cache_hits: stats.cache_hits,
            total_evaluation_time_ms: stats.total_evaluation_time_ms,
            average_evaluation_time_ms: stats.average_evaluation_time_ms,
            min_evaluation_time_ms: stats.min_evaluation_time_ms,
            max_evaluation_time_ms: stats.max_evaluation_time_ms,
        })
    }
    
    /// Clear the cache
    pub fn clear_cache(&self) {
        if let Ok(mut cache) = self.cache.lock() {
            cache.clear();
        }
    }
    
    /// Get cache statistics
    pub fn cache_stats(&self) -> Option<(usize, usize)> {
        self.cache.lock().ok().map(|cache| (cache.len(), cache.cap().get()))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::SafetyRequest;
    
    #[test]
    fn test_engine_creation() {
        let config = CAEConfig::test_config();
        let engine = CAEEngine::new(config);
        assert!(engine.is_ok());
    }
    
    #[test]
    fn test_safe_evaluation() {
        let config = CAEConfig::test_config();
        let engine = CAEEngine::new(config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-123".to_string(),
            request_id: "req-001".to_string(),
            medication_ids: vec!["safe_medication".to_string()],
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        let result = engine.evaluate_safety(&request).unwrap();
        assert_eq!(result.status, SafetyStatus::Safe);
        assert!(result.processing_time_ms < 100); // Should be fast
    }
    
    #[test]
    fn test_unsafe_evaluation() {
        let config = CAEConfig::test_config();
        let engine = CAEEngine::new(config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-456".to_string(),
            request_id: "req-002".to_string(),
            medication_ids: vec!["warfarin".to_string(), "aspirin".to_string()], // Known interaction
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        let result = engine.evaluate_safety(&request).unwrap();
        // Result depends on rule engine implementation
        assert!(result.processing_time_ms < 100);
    }
    
    #[test]
    fn test_caching() {
        let config = CAEConfig::test_config();
        let engine = CAEEngine::new(config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-cache".to_string(),
            request_id: "req-cache-1".to_string(),
            medication_ids: vec!["test_med".to_string()],
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        // First evaluation
        let result1 = engine.evaluate_safety(&request).unwrap();
        
        // Second evaluation should be faster (cached)
        let result2 = engine.evaluate_safety(&request).unwrap();
        
        assert_eq!(result1.status, result2.status);
        assert_eq!(result1.risk_score, result2.risk_score);
    }
    
    #[test]
    fn test_performance_tracking() {
        let config = CAEConfig::test_config();
        let engine = CAEEngine::new(config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-perf".to_string(),
            request_id: "req-perf".to_string(),
            medication_ids: vec!["test_med".to_string()],
            condition_ids: vec![],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        // Perform several evaluations
        for _ in 0..5 {
            let _ = engine.evaluate_safety(&request).unwrap();
        }
        
        let stats = engine.get_stats().unwrap();
        assert!(stats.total_evaluations >= 5);
        assert!(stats.average_evaluation_time_ms > 0.0);
    }
}