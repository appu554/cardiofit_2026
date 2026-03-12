//! # Flow 2 Rust Engine
//!
//! High-performance clinical decision support engine built in Rust.
//! Provides sub-millisecond medication orchestration and rule evaluation
//! for production healthcare systems.

pub mod models;
pub mod knowledge;
pub mod engine;
pub mod api;
pub mod utils;
pub mod jit_safety;
pub mod unified_clinical_engine;
pub mod clients;
pub mod grpc;

// Phase 3 Clinical Intelligence Engine
#[cfg(feature = "phase3")]
pub mod phase3;

// Re-export commonly used types
pub use models::*;
pub use engine::*;
pub use api::*;
pub use jit_safety::*;

/// Engine version information
pub const VERSION: &str = env!("CARGO_PKG_VERSION");
pub const ENGINE_NAME: &str = "Flow2-Rust-Engine";

/// Result type alias for the engine
pub type EngineResult<T> = Result<T, EngineError>;

/// Main engine error type
#[derive(thiserror::Error, Debug)]
pub enum EngineError {
    #[error("Knowledge base error: {0}")]
    KnowledgeBase(#[from] knowledge::KnowledgeError),

    #[error("Rule evaluation error: {0}")]
    RuleEvaluation(String),

    #[error("Validation error: {0}")]
    ValidationError(String),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("YAML parsing error: {0}")]
    YamlParsing(#[from] serde_yaml::Error),

    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),

    #[error("JIT Safety error: {0}")]
    JitSafety(#[from] jit_safety::JitSafetyError),

    #[error("Generic error: {0}")]
    Generic(String),
}
