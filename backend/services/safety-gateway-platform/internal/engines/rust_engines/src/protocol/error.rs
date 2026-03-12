//! Protocol Engine Error Types
//!
//! Comprehensive error handling for the Protocol Engine with detailed
//! error classification and context for debugging and monitoring.

use thiserror::Error;
use std::fmt;

/// Main error type for Protocol Engine operations
#[derive(Error, Debug, Clone)]
pub enum ProtocolEngineError {
    /// Configuration errors
    #[error("Configuration error: {message}")]
    ConfigurationError { message: String },

    /// Engine not found or not initialized
    #[error("Protocol engine not found for tenant: {tenant_id}")]
    EngineNotFound(String),

    /// Engine lock error (threading)
    #[error("Failed to acquire engine lock")]
    EngineLockError,

    /// Protocol definition errors
    #[error("Protocol definition error: {message}")]
    ProtocolDefinitionError { message: String },

    /// Protocol not found
    #[error("Protocol not found: {protocol_id}")]
    ProtocolNotFound { protocol_id: String },

    /// Rule evaluation errors
    #[error("Rule evaluation failed: {rule_id} - {message}")]
    RuleEvaluationError { rule_id: String, message: String },

    /// Expression parsing/evaluation errors
    #[error("Expression error in {context}: {message}")]
    ExpressionError { context: String, message: String },

    /// State machine errors
    #[error("State machine error: {state_machine_id} - {message}")]
    StateMachineError { state_machine_id: String, message: String },

    /// Temporal constraint errors
    #[error("Temporal constraint error: {constraint_id} - {message}")]
    TemporalConstraintError { constraint_id: String, message: String },

    /// Snapshot resolution errors
    #[error("Snapshot error: {snapshot_id} - {message}")]
    SnapshotError { snapshot_id: String, message: String },

    /// Data validation errors
    #[error("Data validation error: {field} - {message}")]
    DataValidationError { field: String, message: String },

    /// Serialization/deserialization errors
    #[error("Serialization error: {context} - {message}")]
    SerializationError { context: String, message: String },

    /// Database/persistence errors
    #[error("Persistence error: {operation} - {message}")]
    PersistenceError { operation: String, message: String },

    /// Network/communication errors
    #[error("Communication error: {service} - {message}")]
    CommunicationError { service: String, message: String },

    /// Timeout errors
    #[error("Operation timeout: {operation} exceeded {timeout_ms}ms")]
    TimeoutError { operation: String, timeout_ms: u64 },

    /// Resource exhaustion errors
    #[error("Resource exhaustion: {resource} - {message}")]
    ResourceExhaustionError { resource: String, message: String },

    /// Authentication/authorization errors
    #[error("Security error: {message}")]
    SecurityError { message: String },

    /// FFI boundary errors
    #[error("FFI error: {context} - {message}")]
    FfiError { context: String, message: String },

    /// Internal errors (should not happen in production)
    #[error("Internal error: {message}")]
    InternalError { message: String },

    /// Multiple errors aggregated
    #[error("Multiple errors occurred: {count} errors")]
    MultipleErrors { 
        count: usize, 
        errors: Vec<ProtocolEngineError> 
    },
}

/// Error type classification for monitoring and metrics
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub enum ProtocolErrorType {
    Configuration,
    Engine,
    Protocol,
    Rule,
    Expression,
    StateMachine,
    Temporal,
    Snapshot,
    DataValidation,
    Serialization,
    Persistence,
    Communication,
    Timeout,
    Resource,
    Security,
    Ffi,
    Internal,
    Multiple,
}

impl ProtocolEngineError {
    /// Get the error type for classification and metrics
    pub fn error_type(&self) -> ProtocolErrorType {
        match self {
            Self::ConfigurationError { .. } => ProtocolErrorType::Configuration,
            Self::EngineNotFound(_) | Self::EngineLockError => ProtocolErrorType::Engine,
            Self::ProtocolDefinitionError { .. } | Self::ProtocolNotFound { .. } => ProtocolErrorType::Protocol,
            Self::RuleEvaluationError { .. } => ProtocolErrorType::Rule,
            Self::ExpressionError { .. } => ProtocolErrorType::Expression,
            Self::StateMachineError { .. } => ProtocolErrorType::StateMachine,
            Self::TemporalConstraintError { .. } => ProtocolErrorType::Temporal,
            Self::SnapshotError { .. } => ProtocolErrorType::Snapshot,
            Self::DataValidationError { .. } => ProtocolErrorType::DataValidation,
            Self::SerializationError { .. } => ProtocolErrorType::Serialization,
            Self::PersistenceError { .. } => ProtocolErrorType::Persistence,
            Self::CommunicationError { .. } => ProtocolErrorType::Communication,
            Self::TimeoutError { .. } => ProtocolErrorType::Timeout,
            Self::ResourceExhaustionError { .. } => ProtocolErrorType::Resource,
            Self::SecurityError { .. } => ProtocolErrorType::Security,
            Self::FfiError { .. } => ProtocolErrorType::Ffi,
            Self::InternalError { .. } => ProtocolErrorType::Internal,
            Self::MultipleErrors { .. } => ProtocolErrorType::Multiple,
        }
    }

    /// Check if error is recoverable (can retry)
    pub fn is_recoverable(&self) -> bool {
        match self {
            Self::TimeoutError { .. } => true,
            Self::CommunicationError { .. } => true,
            Self::ResourceExhaustionError { .. } => true,
            Self::EngineLockError => true,
            Self::PersistenceError { .. } => true, // May be transient
            Self::ConfigurationError { .. } => false,
            Self::ProtocolDefinitionError { .. } => false,
            Self::ProtocolNotFound { .. } => false,
            Self::DataValidationError { .. } => false,
            Self::SerializationError { .. } => false,
            Self::SecurityError { .. } => false,
            Self::InternalError { .. } => false,
            Self::EngineNotFound(_) => false,
            Self::RuleEvaluationError { .. } => false, // Usually indicates logic issue
            Self::ExpressionError { .. } => false,
            Self::StateMachineError { .. } => false,
            Self::TemporalConstraintError { .. } => false,
            Self::SnapshotError { .. } => true, // May be transient
            Self::FfiError { .. } => false,
            Self::MultipleErrors { errors, .. } => {
                errors.iter().any(|e| e.is_recoverable())
            }
        }
    }

    /// Get suggested retry delay in milliseconds for recoverable errors
    pub fn retry_delay_ms(&self) -> Option<u64> {
        if !self.is_recoverable() {
            return None;
        }

        match self {
            Self::TimeoutError { .. } => Some(1000),
            Self::CommunicationError { .. } => Some(2000),
            Self::ResourceExhaustionError { .. } => Some(5000),
            Self::EngineLockError => Some(100),
            Self::PersistenceError { .. } => Some(1000),
            Self::SnapshotError { .. } => Some(500),
            _ => Some(1000),
        }
    }

    /// Convert to a structured error response for external APIs
    pub fn to_api_error(&self) -> ApiError {
        ApiError {
            error_type: self.error_type(),
            message: self.to_string(),
            recoverable: self.is_recoverable(),
            retry_after_ms: self.retry_delay_ms(),
            details: self.get_error_details(),
        }
    }

    /// Get detailed error information for debugging
    pub fn get_error_details(&self) -> Option<serde_json::Value> {
        match self {
            Self::RuleEvaluationError { rule_id, message } => {
                Some(serde_json::json!({
                    "rule_id": rule_id,
                    "message": message
                }))
            },
            Self::StateMachineError { state_machine_id, message } => {
                Some(serde_json::json!({
                    "state_machine_id": state_machine_id,
                    "message": message
                }))
            },
            Self::TimeoutError { operation, timeout_ms } => {
                Some(serde_json::json!({
                    "operation": operation,
                    "timeout_ms": timeout_ms
                }))
            },
            Self::MultipleErrors { count, errors } => {
                Some(serde_json::json!({
                    "error_count": count,
                    "error_types": errors.iter().map(|e| e.error_type()).collect::<Vec<_>>()
                }))
            },
            _ => None,
        }
    }
}

/// Structured error response for external APIs
#[derive(Debug, Clone, serde::Serialize)]
pub struct ApiError {
    pub error_type: ProtocolErrorType,
    pub message: String,
    pub recoverable: bool,
    pub retry_after_ms: Option<u64>,
    pub details: Option<serde_json::Value>,
}

/// Result type alias for Protocol Engine operations
pub type ProtocolResult<T> = Result<T, ProtocolEngineError>;

/// Helper macro for creating configuration errors
#[macro_export]
macro_rules! config_error {
    ($msg:expr) => {
        ProtocolEngineError::ConfigurationError {
            message: $msg.to_string()
        }
    };
    ($fmt:expr, $($arg:tt)*) => {
        ProtocolEngineError::ConfigurationError {
            message: format!($fmt, $($arg)*)
        }
    };
}

/// Helper macro for creating rule evaluation errors
#[macro_export]
macro_rules! rule_error {
    ($rule_id:expr, $msg:expr) => {
        ProtocolEngineError::RuleEvaluationError {
            rule_id: $rule_id.to_string(),
            message: $msg.to_string()
        }
    };
    ($rule_id:expr, $fmt:expr, $($arg:tt)*) => {
        ProtocolEngineError::RuleEvaluationError {
            rule_id: $rule_id.to_string(),
            message: format!($fmt, $($arg)*)
        }
    };
}

/// Helper macro for creating state machine errors
#[macro_export]
macro_rules! state_machine_error {
    ($sm_id:expr, $msg:expr) => {
        ProtocolEngineError::StateMachineError {
            state_machine_id: $sm_id.to_string(),
            message: $msg.to_string()
        }
    };
    ($sm_id:expr, $fmt:expr, $($arg:tt)*) => {
        ProtocolEngineError::StateMachineError {
            state_machine_id: $sm_id.to_string(),
            message: format!($fmt, $($arg)*)
        }
    };
}

// Conversion implementations for common error types

impl From<serde_json::Error> for ProtocolEngineError {
    fn from(err: serde_json::Error) -> Self {
        Self::SerializationError {
            context: "JSON".to_string(),
            message: err.to_string(),
        }
    }
}

impl From<std::io::Error> for ProtocolEngineError {
    fn from(err: std::io::Error) -> Self {
        Self::PersistenceError {
            operation: "IO".to_string(),
            message: err.to_string(),
        }
    }
}

impl From<chrono::ParseError> for ProtocolEngineError {
    fn from(err: chrono::ParseError) -> Self {
        Self::DataValidationError {
            field: "datetime".to_string(),
            message: err.to_string(),
        }
    }
}

impl From<uuid::Error> for ProtocolEngineError {
    fn from(err: uuid::Error) -> Self {
        Self::DataValidationError {
            field: "uuid".to_string(),
            message: err.to_string(),
        }
    }
}

impl From<evalexpr::EvalexprError> for ProtocolEngineError {
    fn from(err: evalexpr::EvalexprError) -> Self {
        Self::ExpressionError {
            context: "evaluation".to_string(),
            message: err.to_string(),
        }
    }
}

impl From<tokio::time::error::Elapsed> for ProtocolEngineError {
    fn from(_: tokio::time::error::Elapsed) -> Self {
        Self::TimeoutError {
            operation: "async_operation".to_string(),
            timeout_ms: 0, // Will be set by caller if known
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_type_classification() {
        let error = ProtocolEngineError::RuleEvaluationError {
            rule_id: "test-rule".to_string(),
            message: "test error".to_string(),
        };
        
        assert_eq!(error.error_type(), ProtocolErrorType::Rule);
        assert!(!error.is_recoverable());
        assert!(error.retry_delay_ms().is_none());
    }

    #[test]
    fn test_timeout_error_recoverability() {
        let error = ProtocolEngineError::TimeoutError {
            operation: "evaluation".to_string(),
            timeout_ms: 5000,
        };
        
        assert_eq!(error.error_type(), ProtocolErrorType::Timeout);
        assert!(error.is_recoverable());
        assert_eq!(error.retry_delay_ms(), Some(1000));
    }

    #[test]
    fn test_api_error_conversion() {
        let error = ProtocolEngineError::ProtocolNotFound {
            protocol_id: "missing-protocol".to_string(),
        };
        
        let api_error = error.to_api_error();
        assert_eq!(api_error.error_type, ProtocolErrorType::Protocol);
        assert!(!api_error.recoverable);
    }

    #[test]
    fn test_error_macros() {
        let config_err = config_error!("Invalid configuration: {}", "timeout");
        match config_err {
            ProtocolEngineError::ConfigurationError { message } => {
                assert!(message.contains("Invalid configuration"));
            },
            _ => panic!("Wrong error type"),
        }

        let rule_err = rule_error!("rule-123", "Evaluation failed: {}", "division by zero");
        match rule_err {
            ProtocolEngineError::RuleEvaluationError { rule_id, message } => {
                assert_eq!(rule_id, "rule-123");
                assert!(message.contains("division by zero"));
            },
            _ => panic!("Wrong error type"),
        }
    }

    #[test]
    fn test_multiple_errors() {
        let errors = vec![
            ProtocolEngineError::RuleEvaluationError {
                rule_id: "rule1".to_string(),
                message: "error1".to_string(),
            },
            ProtocolEngineError::TimeoutError {
                operation: "op1".to_string(),
                timeout_ms: 1000,
            },
        ];

        let multi_error = ProtocolEngineError::MultipleErrors {
            count: errors.len(),
            errors: errors.clone(),
        };

        assert_eq!(multi_error.error_type(), ProtocolErrorType::Multiple);
        // Should be recoverable because one of the errors is recoverable
        assert!(multi_error.is_recoverable());
    }
}