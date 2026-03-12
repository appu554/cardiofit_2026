//! Protocol Engine Core Implementation
//!
//! This module provides the main Protocol Engine implementation that orchestrates
//! clinical pathway enforcement, rule evaluation, state management, and temporal
//! constraint processing.

use std::collections::HashMap;
use std::sync::{Arc, RwLock};
use std::time::Instant;
use tokio::time::{timeout, Duration};
use uuid::Uuid;
use chrono::{DateTime, Utc};
use dashmap::DashMap;
use lru::LruCache;
use serde::{Deserialize, Serialize};

use crate::protocol::{
    types::*,
    error::*,
    rules::RuleEngine,
    state_machine::{ProtocolStateMachine, StateMachineManager},
    temporal::{TemporalConstraintEngine, TemporalConstraint},
    snapshot::{SnapshotResolver, SnapshotContext},
    evaluation::{EvaluationContext, EvaluationResult, DecisionAggregator},
};

/// Protocol Engine configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEngineConfig {
    /// Maximum number of concurrent evaluations
    pub max_concurrent_evaluations: usize,
    
    /// Default evaluation timeout in milliseconds
    pub default_timeout_ms: u64,
    
    /// Maximum evaluation timeout in milliseconds
    pub max_timeout_ms: u64,
    
    /// Rule engine configuration
    pub rule_engine_config: RuleEngineConfig,
    
    /// State machine configuration
    pub state_machine_config: StateMachineConfig,
    
    /// Temporal engine configuration
    pub temporal_engine_config: TemporalEngineConfig,
    
    /// Snapshot resolution configuration
    pub snapshot_config: SnapshotConfig,
    
    /// Cache configuration
    pub cache_config: CacheConfig,
    
    /// Performance monitoring configuration
    pub monitoring_config: MonitoringConfig,
    
    /// Security configuration
    pub security_config: SecurityConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleEngineConfig {
    pub max_rule_execution_time_ms: u64,
    pub max_parallel_rules: usize,
    pub enable_rule_caching: bool,
    pub cache_size: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateMachineConfig {
    pub max_state_machines: usize,
    pub state_persistence_enabled: bool,
    pub transition_timeout_ms: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemporalEngineConfig {
    pub precision_ms: u64,
    pub max_temporal_constraints: usize,
    pub enable_temporal_caching: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotConfig {
    pub snapshot_resolution_timeout_ms: u64,
    pub enable_snapshot_caching: bool,
    pub max_cached_snapshots: usize,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheConfig {
    pub protocol_definition_cache_size: usize,
    pub evaluation_result_cache_size: usize,
    pub cache_ttl_seconds: u64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringConfig {
    pub enable_metrics: bool,
    pub enable_tracing: bool,
    pub performance_sampling_rate: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SecurityConfig {
    pub enable_input_validation: bool,
    pub enable_output_sanitization: bool,
    pub max_request_size_bytes: usize,
}

impl Default for ProtocolEngineConfig {
    fn default() -> Self {
        Self {
            max_concurrent_evaluations: 100,
            default_timeout_ms: 5000,
            max_timeout_ms: 30000,
            rule_engine_config: RuleEngineConfig {
                max_rule_execution_time_ms: 1000,
                max_parallel_rules: 10,
                enable_rule_caching: true,
                cache_size: 1000,
            },
            state_machine_config: StateMachineConfig {
                max_state_machines: 1000,
                state_persistence_enabled: true,
                transition_timeout_ms: 1000,
            },
            temporal_engine_config: TemporalEngineConfig {
                precision_ms: 1000,
                max_temporal_constraints: 100,
                enable_temporal_caching: true,
            },
            snapshot_config: SnapshotConfig {
                snapshot_resolution_timeout_ms: 2000,
                enable_snapshot_caching: true,
                max_cached_snapshots: 100,
            },
            cache_config: CacheConfig {
                protocol_definition_cache_size: 500,
                evaluation_result_cache_size: 1000,
                cache_ttl_seconds: 3600,
            },
            monitoring_config: MonitoringConfig {
                enable_metrics: true,
                enable_tracing: false,
                performance_sampling_rate: 0.1,
            },
            security_config: SecurityConfig {
                enable_input_validation: true,
                enable_output_sanitization: true,
                max_request_size_bytes: 1024 * 1024, // 1MB
            },
        }
    }
}

impl ProtocolEngineConfig {
    /// Create a test configuration with minimal settings
    pub fn test_config() -> Self {
        Self {
            max_concurrent_evaluations: 10,
            default_timeout_ms: 1000,
            max_timeout_ms: 5000,
            ..Default::default()
        }
    }
}

/// Main Protocol Engine implementation
pub struct ProtocolEngine {
    /// Engine configuration
    config: ProtocolEngineConfig,
    
    /// Rule evaluation engine
    rule_engine: Arc<RuleEngine>,
    
    /// State machine manager
    state_machine_manager: Arc<StateMachineManager>,
    
    /// Temporal constraint engine
    temporal_engine: Arc<TemporalConstraintEngine>,
    
    /// Snapshot resolver
    snapshot_resolver: Arc<SnapshotResolver>,
    
    /// Decision aggregator
    decision_aggregator: Arc<DecisionAggregator>,
    
    /// Protocol definition cache
    protocol_cache: Arc<RwLock<LruCache<String, Arc<ProtocolDefinition>>>>,
    
    /// Evaluation result cache
    result_cache: Arc<RwLock<LruCache<String, Arc<ProtocolEvaluationResult>>>>,
    
    /// Active evaluations (for monitoring and cleanup)
    active_evaluations: Arc<DashMap<Uuid, EvaluationContext>>,
    
    /// Engine metrics
    metrics: Arc<EngineMetrics>,
    
    /// Engine creation timestamp
    created_at: DateTime<Utc>,
}

/// Engine performance and operational metrics
#[derive(Debug, Default)]
pub struct EngineMetrics {
    pub total_evaluations: std::sync::atomic::AtomicU64,
    pub successful_evaluations: std::sync::atomic::AtomicU64,
    pub failed_evaluations: std::sync::atomic::AtomicU64,
    pub timeout_evaluations: std::sync::atomic::AtomicU64,
    pub average_evaluation_time_ms: std::sync::atomic::AtomicU64,
    pub cache_hits: std::sync::atomic::AtomicU64,
    pub cache_misses: std::sync::atomic::AtomicU64,
}

impl ProtocolEngine {
    /// Create a new Protocol Engine instance
    pub fn new(config: ProtocolEngineConfig) -> ProtocolResult<Self> {
        // Initialize rule engine
        let rule_engine = Arc::new(RuleEngine::new(&config.rule_engine_config)?);
        
        // Initialize state machine manager
        let state_machine_manager = Arc::new(StateMachineManager::new(&config.state_machine_config)?);
        
        // Initialize temporal constraint engine
        let temporal_engine = Arc::new(TemporalConstraintEngine::new(&config.temporal_engine_config)?);
        
        // Initialize snapshot resolver
        let snapshot_resolver = Arc::new(SnapshotResolver::new(&config.snapshot_config)?);
        
        // Initialize decision aggregator
        let decision_aggregator = Arc::new(DecisionAggregator::new()?);
        
        // Initialize caches
        let protocol_cache = Arc::new(RwLock::new(
            LruCache::new(config.cache_config.protocol_definition_cache_size.try_into()
                .map_err(|_| ProtocolEngineError::ConfigurationError {
                    message: "Invalid protocol cache size".to_string()
                })?
            )
        ));
        
        let result_cache = Arc::new(RwLock::new(
            LruCache::new(config.cache_config.evaluation_result_cache_size.try_into()
                .map_err(|_| ProtocolEngineError::ConfigurationError {
                    message: "Invalid result cache size".to_string()
                })?
            )
        ));
        
        Ok(Self {
            config,
            rule_engine,
            state_machine_manager,
            temporal_engine,
            snapshot_resolver,
            decision_aggregator,
            protocol_cache,
            result_cache,
            active_evaluations: Arc::new(DashMap::new()),
            metrics: Arc::new(EngineMetrics::default()),
            created_at: Utc::now(),
        })
    }
    
    /// Evaluate a protocol for the given request
    pub async fn evaluate_protocol(
        &self,
        request: &ProtocolEvaluationRequest,
    ) -> ProtocolResult<ProtocolEvaluationResult> {
        let start_time = Instant::now();
        let evaluation_id = Uuid::new_v4();
        
        // Increment total evaluations counter
        self.metrics.total_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        // Validate request
        self.validate_request(request)?;
        
        // Create evaluation context
        let mut eval_context = EvaluationContext::new(
            evaluation_id,
            request.clone(),
            start_time,
        );
        
        // Register active evaluation
        self.active_evaluations.insert(evaluation_id, eval_context.clone());
        
        // Determine timeout
        let timeout_ms = request.metadata
            .as_ref()
            .and_then(|m| m.timeout_ms)
            .unwrap_or(self.config.default_timeout_ms)
            .min(self.config.max_timeout_ms);
        
        // Execute evaluation with timeout
        let result = timeout(
            Duration::from_millis(timeout_ms),
            self.execute_evaluation(&mut eval_context)
        ).await;
        
        // Clean up active evaluation
        self.active_evaluations.remove(&evaluation_id);
        
        // Process result
        match result {
            Ok(Ok(evaluation_result)) => {
                self.metrics.successful_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                
                // Update average execution time
                let execution_time = start_time.elapsed().as_millis() as u64;
                self.update_average_execution_time(execution_time);
                
                // Cache result if applicable
                if self.should_cache_result(&evaluation_result) {
                    self.cache_evaluation_result(&evaluation_result).await;
                }
                
                Ok(evaluation_result)
            },
            Ok(Err(error)) => {
                self.metrics.failed_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Err(error)
            },
            Err(_) => {
                self.metrics.timeout_evaluations.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Err(ProtocolEngineError::TimeoutError {
                    operation: "protocol_evaluation".to_string(),
                    timeout_ms,
                })
            },
        }
    }
    
    /// Execute the main evaluation logic
    async fn execute_evaluation(
        &self,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<ProtocolEvaluationResult> {
        let request = &eval_context.request;
        
        // Step 1: Resolve snapshot context if provided
        let snapshot_context = if let Some(snapshot_id) = &request.snapshot_id {
            Some(self.snapshot_resolver.resolve_snapshot(snapshot_id).await?)
        } else {
            None
        };
        
        // Step 2: Load protocol definition
        let protocol_def = self.load_protocol_definition(&request.protocol_id).await?;
        
        // Step 3: Initialize state machines for this protocol
        let state_machines = self.initialize_state_machines(
            &protocol_def, 
            &request.patient_id,
            eval_context
        ).await?;
        
        // Step 4: Evaluate rules in parallel
        let rule_results = self.rule_engine.evaluate_rules(
            &protocol_def.rules,
            &request.clinical_context,
            snapshot_context.as_ref(),
            eval_context
        ).await?;
        
        // Step 5: Evaluate constraints
        let constraint_results = self.evaluate_constraints(
            &protocol_def.constraints,
            &request.clinical_context,
            snapshot_context.as_ref(),
            eval_context
        ).await?;
        
        // Step 6: Evaluate temporal constraints
        let temporal_results = self.temporal_engine.evaluate_temporal_constraints(
            &protocol_def.temporal_constraints,
            &request.clinical_context,
            request.evaluation_timestamp,
            eval_context
        ).await?;
        
        // Step 7: Process state machine transitions
        let state_changes = self.process_state_transitions(
            &state_machines,
            &rule_results,
            &constraint_results,
            &temporal_results,
            eval_context
        ).await?;
        
        // Step 8: Aggregate decision
        let decision = self.decision_aggregator.aggregate_decision(
            &rule_results,
            &constraint_results,
            &temporal_results,
            &state_changes,
            eval_context
        ).await?;
        
        // Step 9: Generate recommendations
        let recommendations = self.generate_recommendations(
            &decision,
            &rule_results,
            &constraint_results,
            &temporal_results,
            eval_context
        ).await?;
        
        // Step 10: Create final result
        let result = ProtocolEvaluationResult {
            result_id: Uuid::new_v4(),
            request_id: request.metadata.as_ref().map(|m| m.request_id),
            protocol_id: request.protocol_id.clone(),
            decision,
            evaluation_details: EvaluationDetails {
                rules_evaluated: rule_results,
                constraints_checked: constraint_results,
                conditions_met: eval_context.conditions_met.clone(),
                conditions_failed: eval_context.conditions_failed.clone(),
                warnings: eval_context.warnings.clone(),
                information: eval_context.information.clone(),
            },
            state_changes,
            temporal_constraints: temporal_results,
            recommendations,
            performance_metrics: EvaluationMetrics {
                total_execution_time_ms: eval_context.start_time.elapsed().as_millis() as u64,
                rules_execution_time_ms: eval_context.rules_execution_time_ms,
                constraints_execution_time_ms: eval_context.constraints_execution_time_ms,
                state_machine_time_ms: eval_context.state_machine_time_ms,
                snapshot_resolution_time_ms: eval_context.snapshot_resolution_time_ms,
                memory_usage_bytes: None, // TODO: Implement memory tracking
            },
            evaluated_at: Utc::now(),
            snapshot_context: snapshot_context.map(|sc| sc.snapshot_id),
        };
        
        Ok(result)
    }
    
    /// Validate the evaluation request
    fn validate_request(&self, request: &ProtocolEvaluationRequest) -> ProtocolResult<()> {
        if !self.config.security_config.enable_input_validation {
            return Ok(());
        }
        
        // Validate protocol ID
        if request.protocol_id.is_empty() {
            return Err(ProtocolEngineError::DataValidationError {
                field: "protocol_id".to_string(),
                message: "Protocol ID cannot be empty".to_string(),
            });
        }
        
        // Validate patient ID
        if request.patient_id.is_empty() {
            return Err(ProtocolEngineError::DataValidationError {
                field: "patient_id".to_string(),
                message: "Patient ID cannot be empty".to_string(),
            });
        }
        
        // Validate timestamp
        let now = Utc::now();
        let time_diff = (now - request.evaluation_timestamp).num_seconds().abs();
        if time_diff > 3600 { // 1 hour tolerance
            return Err(ProtocolEngineError::DataValidationError {
                field: "evaluation_timestamp".to_string(),
                message: format!("Evaluation timestamp is too far from current time: {} seconds", time_diff),
            });
        }
        
        // Validate request size (approximate)
        if let Ok(serialized) = serde_json::to_string(request) {
            if serialized.len() > self.config.security_config.max_request_size_bytes {
                return Err(ProtocolEngineError::DataValidationError {
                    field: "request_size".to_string(),
                    message: format!("Request size {} exceeds maximum {}", 
                        serialized.len(), 
                        self.config.security_config.max_request_size_bytes
                    ),
                });
            }
        }
        
        Ok(())
    }
    
    /// Load protocol definition (with caching)
    async fn load_protocol_definition(&self, protocol_id: &str) -> ProtocolResult<Arc<ProtocolDefinition>> {
        // Check cache first
        {
            let mut cache = self.protocol_cache.write()
                .map_err(|_| ProtocolEngineError::InternalError {
                    message: "Failed to acquire protocol cache lock".to_string()
                })?;
            
            if let Some(protocol) = cache.get(protocol_id) {
                self.metrics.cache_hits.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                return Ok(protocol.clone());
            }
        }
        
        self.metrics.cache_misses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        // Load from storage (placeholder implementation)
        let protocol = self.load_protocol_from_storage(protocol_id).await?;
        let protocol_arc = Arc::new(protocol);
        
        // Cache the protocol
        {
            let mut cache = self.protocol_cache.write()
                .map_err(|_| ProtocolEngineError::InternalError {
                    message: "Failed to acquire protocol cache lock for update".to_string()
                })?;
            cache.put(protocol_id.to_string(), protocol_arc.clone());
        }
        
        Ok(protocol_arc)
    }
    
    /// Load protocol from storage (placeholder - should integrate with actual storage)
    async fn load_protocol_from_storage(&self, protocol_id: &str) -> ProtocolResult<ProtocolDefinition> {
        // This is a placeholder implementation
        // In a real system, this would load from a database, file system, or external service
        
        match protocol_id {
            "sepsis-bundle-v1" => Ok(self.create_test_sepsis_protocol()),
            "vte-prophylaxis-v1" => Ok(self.create_test_vte_protocol()),
            _ => Err(ProtocolEngineError::ProtocolNotFound {
                protocol_id: protocol_id.to_string(),
            }),
        }
    }
    
    /// Create test sepsis protocol (for demonstration)
    fn create_test_sepsis_protocol(&self) -> ProtocolDefinition {
        ProtocolDefinition {
            protocol_id: "sepsis-bundle-v1".to_string(),
            name: "Sepsis Bundle Protocol".to_string(),
            version: "1.0.0".to_string(),
            description: "3-hour sepsis bundle implementation".to_string(),
            metadata: ProtocolMetadata {
                author: "Clinical Team".to_string(),
                created_date: Utc::now(),
                last_modified: Utc::now(),
                approval_status: ApprovalStatus::Approved,
                clinical_domain: "Infectious Disease".to_string(),
                target_population: "Adult patients with suspected sepsis".to_string(),
                evidence_level: EvidenceLevel::A,
                tags: vec!["sepsis".to_string(), "bundle".to_string(), "critical-care".to_string()],
            },
            rules: vec![
                ProtocolRule {
                    rule_id: "sepsis-lactate-check".to_string(),
                    name: "Lactate Level Check".to_string(),
                    description: "Ensure lactate level is checked within 1 hour".to_string(),
                    condition: RuleCondition::Expression("has_lactate_order || lactate_within_1hr".to_string()),
                    action: RuleAction {
                        action_type: RuleActionType::RequireApproval,
                        parameters: HashMap::new(),
                        message: Some("Lactate level should be checked within 1 hour of sepsis recognition".to_string()),
                    },
                    priority: 1,
                    enabled: true,
                },
            ],
            constraints: vec![],
            state_machines: vec![],
            temporal_constraints: vec![],
        }
    }
    
    /// Create test VTE protocol (for demonstration)
    fn create_test_vte_protocol(&self) -> ProtocolDefinition {
        ProtocolDefinition {
            protocol_id: "vte-prophylaxis-v1".to_string(),
            name: "VTE Prophylaxis Protocol".to_string(),
            version: "1.0.0".to_string(),
            description: "Venous thromboembolism prophylaxis protocol".to_string(),
            metadata: ProtocolMetadata {
                author: "Clinical Team".to_string(),
                created_date: Utc::now(),
                last_modified: Utc::now(),
                approval_status: ApprovalStatus::Approved,
                clinical_domain: "Hematology".to_string(),
                target_population: "Hospitalized patients at risk for VTE".to_string(),
                evidence_level: EvidenceLevel::A,
                tags: vec!["vte".to_string(), "prophylaxis".to_string(), "anticoagulation".to_string()],
            },
            rules: vec![],
            constraints: vec![],
            state_machines: vec![],
            temporal_constraints: vec![],
        }
    }
    
    // Additional helper methods would be implemented here...
    // (The full implementation would include all the remaining methods)
    
    /// Update average execution time metric
    fn update_average_execution_time(&self, execution_time_ms: u64) {
        // Simple running average (in production, this would use a more sophisticated approach)
        let current_avg = self.metrics.average_evaluation_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_evals = self.metrics.total_evaluations.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_evals > 0 {
            let new_avg = ((current_avg * (total_evals - 1)) + execution_time_ms) / total_evals;
            self.metrics.average_evaluation_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        }
    }
    
    /// Check if result should be cached
    fn should_cache_result(&self, _result: &ProtocolEvaluationResult) -> bool {
        // For now, cache all successful results
        // In production, this might depend on result type, protocol, etc.
        true
    }
    
    /// Cache evaluation result
    async fn cache_evaluation_result(&self, result: &ProtocolEvaluationResult) {
        let cache_key = format!("{}:{}", result.protocol_id, result.result_id);
        let mut cache = match self.result_cache.write() {
            Ok(cache) => cache,
            Err(_) => return, // Don't fail evaluation if caching fails
        };
        
        cache.put(cache_key, Arc::new(result.clone()));
    }
    
    /// Get engine status and metrics
    pub fn get_engine_status(&self) -> EngineStatus {
        EngineStatus {
            created_at: self.created_at,
            total_evaluations: self.metrics.total_evaluations.load(std::sync::atomic::Ordering::Relaxed),
            successful_evaluations: self.metrics.successful_evaluations.load(std::sync::atomic::Ordering::Relaxed),
            failed_evaluations: self.metrics.failed_evaluations.load(std::sync::atomic::Ordering::Relaxed),
            timeout_evaluations: self.metrics.timeout_evaluations.load(std::sync::atomic::Ordering::Relaxed),
            average_evaluation_time_ms: self.metrics.average_evaluation_time_ms.load(std::sync::atomic::Ordering::Relaxed),
            active_evaluations: self.active_evaluations.len(),
            cache_hits: self.metrics.cache_hits.load(std::sync::atomic::Ordering::Relaxed),
            cache_misses: self.metrics.cache_misses.load(std::sync::atomic::Ordering::Relaxed),
        }
    }
    
    // Placeholder implementations for methods that would be fully implemented
    async fn initialize_state_machines(&self, _protocol_def: &ProtocolDefinition, _patient_id: &str, _eval_context: &mut EvaluationContext) -> ProtocolResult<Vec<String>> {
        Ok(vec![])
    }
    
    async fn evaluate_constraints(&self, _constraints: &[ProtocolConstraint], _context: &ProtocolContext, _snapshot: Option<&SnapshotContext>, _eval_context: &mut EvaluationContext) -> ProtocolResult<Vec<ConstraintEvaluationResult>> {
        Ok(vec![])
    }
    
    async fn process_state_transitions(&self, _state_machines: &[String], _rule_results: &[RuleEvaluationResult], _constraint_results: &[ConstraintEvaluationResult], _temporal_results: &[AppliedTemporalConstraint], _eval_context: &mut EvaluationContext) -> ProtocolResult<Vec<StateChange>> {
        Ok(vec![])
    }
    
    async fn generate_recommendations(&self, _decision: &ProtocolDecision, _rule_results: &[RuleEvaluationResult], _constraint_results: &[ConstraintEvaluationResult], _temporal_results: &[AppliedTemporalConstraint], _eval_context: &mut EvaluationContext) -> ProtocolResult<Vec<ProtocolRecommendation>> {
        Ok(vec![])
    }
}

/// Engine status information
#[derive(Debug, Serialize)]
pub struct EngineStatus {
    pub created_at: DateTime<Utc>,
    pub total_evaluations: u64,
    pub successful_evaluations: u64,
    pub failed_evaluations: u64,
    pub timeout_evaluations: u64,
    pub average_evaluation_time_ms: u64,
    pub active_evaluations: usize,
    pub cache_hits: u64,
    pub cache_misses: u64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use tokio_test;

    #[tokio::test]
    async fn test_protocol_engine_creation() {
        let config = ProtocolEngineConfig::test_config();
        let engine = ProtocolEngine::new(config);
        assert!(engine.is_ok());
    }

    #[tokio::test]
    async fn test_basic_evaluation() {
        let config = ProtocolEngineConfig::test_config();
        let engine = ProtocolEngine::new(config).unwrap();
        
        let request = ProtocolEvaluationRequest {
            protocol_id: "sepsis-bundle-v1".to_string(),
            patient_id: "patient-12345".to_string(),
            clinical_context: ProtocolContext::default(),
            snapshot_id: None,
            evaluation_timestamp: Utc::now(),
            metadata: None,
        };
        
        let result = engine.evaluate_protocol(&request).await;
        assert!(result.is_ok());
        
        let status = engine.get_engine_status();
        assert_eq!(status.total_evaluations, 1);
        assert_eq!(status.successful_evaluations, 1);
    }

    #[tokio::test]
    async fn test_invalid_protocol() {
        let config = ProtocolEngineConfig::test_config();
        let engine = ProtocolEngine::new(config).unwrap();
        
        let request = ProtocolEvaluationRequest {
            protocol_id: "nonexistent-protocol".to_string(),
            patient_id: "patient-12345".to_string(),
            clinical_context: ProtocolContext::default(),
            snapshot_id: None,
            evaluation_timestamp: Utc::now(),
            metadata: None,
        };
        
        let result = engine.evaluate_protocol(&request).await;
        assert!(result.is_err());
        
        if let Err(ProtocolEngineError::ProtocolNotFound { protocol_id }) = result {
            assert_eq!(protocol_id, "nonexistent-protocol");
        } else {
            panic!("Expected ProtocolNotFound error");
        }
    }
}