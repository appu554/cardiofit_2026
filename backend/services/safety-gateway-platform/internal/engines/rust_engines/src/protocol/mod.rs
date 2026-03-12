//! Protocol Engine - Clinical Pathway Enforcement Engine
//!
//! This module implements a high-performance protocol engine for enforcing
//! clinical pathways, care protocols, and institutional policies within the
//! Safety Gateway Platform. It provides deterministic, snapshot-driven
//! protocol evaluation with advanced features including:
//!
//! - Stateful protocol tracking across clinical pathways
//! - Temporal constraint enforcement for time-sensitive protocols
//! - Rule-based evaluation with parallel processing
//! - FFI integration with Go orchestration layer
//! - Event-driven workflow integration

pub mod engine;
pub mod types;
pub mod rules;
pub mod state_machine;
pub mod temporal;
pub mod snapshot;
pub mod evaluation;
pub mod ffi_protocol;
pub mod error;
pub mod cae_integration;
pub mod event_publisher;
pub mod approval_workflow;
pub mod message_router;
pub mod schema_validation;
pub mod integration_tests;

// Re-export main types for convenience
pub use engine::{ProtocolEngine, ProtocolEngineConfig};
pub use types::{
    ProtocolEvaluationRequest, ProtocolEvaluationResult, 
    ProtocolDecisionType, ProtocolConstraint, ProtocolContext,
    ProtocolDefinition, ProtocolRule, ProtocolMetadata
};
pub use error::{ProtocolEngineError, ProtocolErrorType};
pub use state_machine::{ProtocolStateMachine, ProtocolState, StateTransition};
pub use temporal::{TemporalConstraint, TemporalConstraintEngine, TimeWindow};
pub use snapshot::{SnapshotContext, SnapshotMetadata, ProtocolSnapshot};
pub use evaluation::{
    EvaluationContext, EvaluationResult, RuleEvaluator,
    ConstraintEvaluator, DecisionAggregator
};
pub use cae_integration::{
    CaeIntegrationEngine, CaeEvaluationRequest, CaeEvaluationResponse,
    CaeIntegrationConfig, CaeHealthStatus
};
pub use event_publisher::{
    EventPublisher, ProtocolEvent, ProtocolEventType,
    EventPublisherConfig, PublishingStatsSnapshot
};
pub use approval_workflow::{
    ApprovalWorkflowEngine, ApprovalRequest, ApprovalStatus,
    ApprovalWorkflowConfig, ClinicalRole
};
pub use message_router::{
    MessageRouter, ServiceMessage, ServiceMessageType,
    MessageRouterConfig, ServiceHealthStatus
};
pub use schema_validation::{
    SchemaValidator, ValidationResult, SchemaValidationConfig,
    ValidationError, ValidationSeverity
};
pub use integration_tests::{
    IntegrationTestEngine, IntegrationTestSuite, IntegrationTestCase,
    IntegrationTestConfig, TestExecutionResult
};

use once_cell::sync::Lazy;
use std::sync::{Arc, Mutex};
use dashmap::DashMap;

/// Global protocol engine registry for multi-tenant support
static PROTOCOL_ENGINE_REGISTRY: Lazy<DashMap<String, Arc<Mutex<ProtocolEngine>>>> = 
    Lazy::new(|| DashMap::new());

/// Initialize a protocol engine instance for a specific tenant/environment
pub fn initialize_protocol_engine(
    tenant_id: String,
    config: ProtocolEngineConfig,
) -> Result<(), ProtocolEngineError> {
    let engine = ProtocolEngine::new(config)?;
    PROTOCOL_ENGINE_REGISTRY.insert(tenant_id, Arc::new(Mutex::new(engine)));
    Ok(())
}

/// Get a reference to a protocol engine for a specific tenant
pub fn with_protocol_engine<F, R>(
    tenant_id: &str,
    f: F,
) -> Result<R, ProtocolEngineError>
where
    F: FnOnce(&ProtocolEngine) -> Result<R, ProtocolEngineError>,
{
    match PROTOCOL_ENGINE_REGISTRY.get(tenant_id) {
        Some(engine_ref) => {
            let engine = engine_ref.value().lock()
                .map_err(|_| ProtocolEngineError::EngineLockError)?;
            f(&*engine)
        },
        None => Err(ProtocolEngineError::EngineNotFound(tenant_id.to_string())),
    }
}

/// Shutdown a specific protocol engine instance
pub fn shutdown_protocol_engine(tenant_id: &str) -> Result<(), ProtocolEngineError> {
    match PROTOCOL_ENGINE_REGISTRY.remove(tenant_id) {
        Some(_) => Ok(()),
        None => Err(ProtocolEngineError::EngineNotFound(tenant_id.to_string())),
    }
}

/// Shutdown all protocol engine instances
pub fn shutdown_all_protocol_engines() {
    PROTOCOL_ENGINE_REGISTRY.clear();
}

/// Get the number of active protocol engine instances
pub fn active_engine_count() -> usize {
    PROTOCOL_ENGINE_REGISTRY.len()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::protocol::types::*;

    #[tokio::test]
    async fn test_protocol_engine_registry() {
        let tenant_id = "test-tenant-001";
        let config = ProtocolEngineConfig::test_config();
        
        // Initialize engine
        assert!(initialize_protocol_engine(tenant_id.to_string(), config).is_ok());
        assert_eq!(active_engine_count(), 1);
        
        // Use engine
        let request = ProtocolEvaluationRequest {
            protocol_id: "sepsis-bundle-v1".to_string(),
            patient_id: "patient-12345".to_string(),
            clinical_context: ProtocolContext::default(),
            snapshot_id: Some("snapshot-001".to_string()),
            evaluation_timestamp: chrono::Utc::now(),
        };
        
        let result = with_protocol_engine(tenant_id, |engine| {
            engine.evaluate_protocol(&request)
        });
        assert!(result.is_ok());
        
        // Shutdown specific engine
        assert!(shutdown_protocol_engine(tenant_id).is_ok());
        assert_eq!(active_engine_count(), 0);
        
        // Engine should not be available after shutdown
        let result = with_protocol_engine(tenant_id, |engine| {
            engine.evaluate_protocol(&request)
        });
        assert!(result.is_err());
    }
}