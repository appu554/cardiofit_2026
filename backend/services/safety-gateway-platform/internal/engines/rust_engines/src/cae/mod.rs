// Clinical Assertion Engine (CAE) Module
//
// This module contains the core clinical safety evaluation logic that replaces
// the Python subprocess implementation. It provides high-performance, memory-safe
// clinical decision support with comprehensive safety checks.

pub mod engine;
pub mod rules;
pub mod database;
pub mod config;

// Re-export main types for convenience
pub use engine::CAEEngine;
pub use rules::{RuleEngine, DrugInteractionDB, ContraindicationDB, DosingRuleDB};
pub use database::{ClinicalDatabase, DatabaseError};
pub use config::CAEConfig;

use thiserror::Error;

/// Errors that can occur during CAE operations
#[derive(Error, Debug)]
pub enum CAEError {
    #[error("CAE engine not initialized")]
    EngineNotInitialized,
    
    #[error("Invalid configuration: {0}")]
    InvalidConfiguration(String),
    
    #[error("Database error: {0}")]
    DatabaseError(#[from] DatabaseError),
    
    #[error("Rule evaluation error: {0}")]
    RuleEvaluationError(String),
    
    #[error("Invalid input data: {0}")]
    InvalidInput(String),
    
    #[error("Timeout during evaluation")]
    Timeout,
    
    #[error("Internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

/// Result type for CAE operations
pub type CAEResult<T> = Result<T, CAEError>;

/// CAE engine capabilities
#[derive(Debug, Clone)]
pub enum CAECapability {
    DrugInteraction,
    Contraindication,
    DosingValidation,
    AllergyCheck,
    DuplicateTherapy,
    ClinicalProtocol,
}

impl CAECapability {
    pub fn as_str(&self) -> &'static str {
        match self {
            CAECapability::DrugInteraction => "drug_interaction",
            CAECapability::Contraindication => "contraindication",
            CAECapability::DosingValidation => "dosing_validation",
            CAECapability::AllergyCheck => "allergy_check",
            CAECapability::DuplicateTherapy => "duplicate_therapy",
            CAECapability::ClinicalProtocol => "clinical_protocol",
        }
    }
    
    pub fn all() -> Vec<CAECapability> {
        vec![
            CAECapability::DrugInteraction,
            CAECapability::Contraindication,
            CAECapability::DosingValidation,
            CAECapability::AllergyCheck,
            CAECapability::DuplicateTherapy,
            CAECapability::ClinicalProtocol,
        ]
    }
}