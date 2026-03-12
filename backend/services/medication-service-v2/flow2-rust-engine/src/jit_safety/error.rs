//! # JIT Safety Error Types
//!
//! Comprehensive error taxonomy for the JIT Safety Engine.
//! All errors are structured, non-panic, and include complete context
//! for debugging and audit purposes.

use thiserror::Error;

/// JIT Safety Engine error types
#[derive(Error, Debug)]
pub enum JitSafetyError {
    /// Input validation errors
    #[error("JIT-INPUT-VALIDATION: {message}")]
    InputValidation {
        message: String,
        request_id: Option<String>,
        drug_id: Option<String>,
    },

    /// Rule pack not found errors
    #[error("JIT-RULEPACK-NOT-FOUND: Missing rule pack for drug '{drug_id}'")]
    RulePackNotFound {
        drug_id: String,
        request_id: Option<String>,
    },

    /// Rule pack parsing errors
    #[error("JIT-RULEPACK-PARSE: Failed to parse rule pack for '{drug_id}': {message}")]
    RulePackParse {
        drug_id: String,
        message: String,
        request_id: Option<String>,
    },

    /// DDI adapter errors
    #[error("JIT-DDI-ERROR: DDI adapter failed: {message}")]
    DdiError {
        message: String,
        request_id: Option<String>,
        drug_id: Option<String>,
    },

    /// Context normalization errors
    #[error("JIT-NORMALIZATION: Context normalization failed: {message}")]
    Normalization {
        message: String,
        request_id: Option<String>,
        drug_id: Option<String>,
    },

    /// TOML parsing errors
    #[error("JIT-TOML-PARSE: TOML parsing error: {0}")]
    TomlParse(#[from] toml::de::Error),

    /// IO errors
    #[error("JIT-IO-ERROR: IO operation failed: {0}")]
    Io(#[from] std::io::Error),

    /// Generic errors
    #[error("JIT-GENERIC-ERROR: {message}")]
    Generic {
        message: String,
        request_id: Option<String>,
    },
}

impl JitSafetyError {
    /// Create an input validation error
    pub fn input_validation(message: impl Into<String>) -> Self {
        Self::InputValidation {
            message: message.into(),
            request_id: None,
            drug_id: None,
        }
    }

    /// Create an input validation error with context
    pub fn input_validation_with_context(
        message: impl Into<String>,
        request_id: Option<String>,
        drug_id: Option<String>,
    ) -> Self {
        Self::InputValidation {
            message: message.into(),
            request_id,
            drug_id,
        }
    }

    /// Create a rule pack not found error
    pub fn rule_pack_not_found(drug_id: impl Into<String>) -> Self {
        Self::RulePackNotFound {
            drug_id: drug_id.into(),
            request_id: None,
        }
    }

    /// Create a rule pack not found error with context
    pub fn rule_pack_not_found_with_context(
        drug_id: impl Into<String>,
        request_id: Option<String>,
    ) -> Self {
        Self::RulePackNotFound {
            drug_id: drug_id.into(),
            request_id,
        }
    }

    /// Create a rule pack parse error
    pub fn rule_pack_parse(drug_id: impl Into<String>, message: impl Into<String>) -> Self {
        Self::RulePackParse {
            drug_id: drug_id.into(),
            message: message.into(),
            request_id: None,
        }
    }

    /// Create a DDI error
    pub fn ddi_error(message: impl Into<String>) -> Self {
        Self::DdiError {
            message: message.into(),
            request_id: None,
            drug_id: None,
        }
    }

    /// Create a DDI error with context
    pub fn ddi_error_with_context(
        message: impl Into<String>,
        request_id: Option<String>,
        drug_id: Option<String>,
    ) -> Self {
        Self::DdiError {
            message: message.into(),
            request_id,
            drug_id,
        }
    }

    /// Create a normalization error
    pub fn normalization(message: impl Into<String>) -> Self {
        Self::Normalization {
            message: message.into(),
            request_id: None,
            drug_id: None,
        }
    }

    /// Create a normalization error with context
    pub fn normalization_with_context(
        message: impl Into<String>,
        request_id: Option<String>,
        drug_id: Option<String>,
    ) -> Self {
        Self::Normalization {
            message: message.into(),
            request_id,
            drug_id,
        }
    }

    /// Create a generic error
    pub fn generic(message: impl Into<String>) -> Self {
        Self::Generic {
            message: message.into(),
            request_id: None,
        }
    }

    /// Create a generic error with context
    pub fn generic_with_context(message: impl Into<String>, request_id: Option<String>) -> Self {
        Self::Generic {
            message: message.into(),
            request_id,
        }
    }

    /// Get the request ID if available
    pub fn request_id(&self) -> Option<&str> {
        match self {
            Self::InputValidation { request_id, .. }
            | Self::RulePackNotFound { request_id, .. }
            | Self::RulePackParse { request_id, .. }
            | Self::DdiError { request_id, .. }
            | Self::Normalization { request_id, .. }
            | Self::Generic { request_id, .. } => request_id.as_deref(),
            _ => None,
        }
    }

    /// Get the drug ID if available
    pub fn drug_id(&self) -> Option<&str> {
        match self {
            Self::InputValidation { drug_id, .. }
            | Self::DdiError { drug_id, .. }
            | Self::Normalization { drug_id, .. } => drug_id.as_deref(),
            Self::RulePackNotFound { drug_id, .. } | Self::RulePackParse { drug_id, .. } => {
                Some(drug_id)
            }
            _ => None,
        }
    }

    /// Check if this is a recoverable error
    pub fn is_recoverable(&self) -> bool {
        matches!(
            self,
            Self::DdiError { .. } | Self::Normalization { .. } | Self::Generic { .. }
        )
    }

    /// Get error category for metrics
    pub fn category(&self) -> &'static str {
        match self {
            Self::InputValidation { .. } => "input_validation",
            Self::RulePackNotFound { .. } => "rule_pack_not_found",
            Self::RulePackParse { .. } => "rule_pack_parse",
            Self::DdiError { .. } => "ddi_error",
            Self::Normalization { .. } => "normalization",
            Self::TomlParse(_) => "toml_parse",
            Self::Io(_) => "io_error",
            Self::Generic { .. } => "generic",
        }
    }
}
