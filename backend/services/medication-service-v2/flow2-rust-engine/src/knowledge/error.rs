//! Knowledge base error types

use thiserror::Error;

/// Knowledge base specific errors
#[derive(Error, Debug)]
pub enum KnowledgeError {
    #[error("Failed to load knowledge base from path: {path}")]
    LoadFailed { path: String },
    
    #[error("Invalid YAML format in file: {file}")]
    InvalidYaml { file: String },
    
    #[error("Missing required field: {field} in file: {file}")]
    MissingField { field: String, file: String },
    
    #[error("Knowledge base validation failed: {reason}")]
    ValidationFailed { reason: String },
    
    #[error("File not found: {path}")]
    FileNotFound { path: String },
    
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    
    #[error("YAML parsing error: {0}")]
    Yaml(#[from] serde_yaml::Error),
}
