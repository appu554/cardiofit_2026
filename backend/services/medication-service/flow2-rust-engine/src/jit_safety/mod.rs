//! # JIT Safety Engine Module
//!
//! Just-in-Time Safety Check engine for dose-aware medication safety validation.
//! Provides precise safety checks on specific drug + dose + frequency combinations
//! against complete patient context (renal/hepatic/age/pregnancy/DDI).
//!
//! ## Architecture
//!
//! The JIT Safety Engine follows a deterministic evaluation order:
//! 1. Context normalization (dose units, CrCl calculation)
//! 2. Hard contraindications (allergies, pregnancy, history flags)
//! 3. DDI contraindicated checks
//! 4. Renal banding and dose adjustments
//! 5. Hepatic constraints and QT considerations
//! 6. Absolute dose boundaries
//! 7. Duplicate class/therapeutic duplication
//! 8. Final decision synthesis
//!
//! ## Key Features
//!
//! - **Config-driven**: All rules stored in versioned TOML files
//! - **Deterministic**: Same input always produces same output
//! - **Auditable**: Complete evaluation trace for regulatory compliance
//! - **Performance**: Sub-50ms evaluation per drug
//! - **SaMD-ready**: Built for medical device software requirements

pub mod domain;
pub mod engine;
pub mod rules;
pub mod normalization;
pub mod ddi_adapter;
pub mod error;

// Re-export main types for convenience
pub use domain::*;
pub use engine::JitEngine;
pub use error::JitSafetyError;

/// JIT Safety Engine version
pub const JIT_ENGINE_VERSION: &str = "1.0.0";

/// Result type alias for JIT Safety operations
pub type JitResult<T> = Result<T, JitSafetyError>;

#[cfg(test)]
mod tests;
