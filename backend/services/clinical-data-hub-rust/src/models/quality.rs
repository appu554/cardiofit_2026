use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DataQuality {
    pub completeness_score: f64,
    pub accuracy_score: f64,
    pub consistency_score: f64,
    pub timeliness_score: f64,
    pub overall_score: f64,
    pub issues: Vec<QualityIssue>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QualityIssue {
    pub issue_type: QualityIssueType,
    pub severity: Severity,
    pub description: String,
    pub field: Option<String>,
    pub count: u32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum QualityIssueType {
    MissingData,
    InvalidFormat,
    OutOfRange,
    Duplicate,
    Inconsistent,
    Stale,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum Severity {
    Low,
    Medium,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub is_valid: bool,
    pub errors: Vec<ValidationError>,
    pub warnings: Vec<ValidationWarning>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationError {
    pub field: String,
    pub message: String,
    pub error_code: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationWarning {
    pub field: String,
    pub message: String,
    pub warning_code: String,
}

impl Default for DataQuality {
    fn default() -> Self {
        Self {
            completeness_score: 1.0,
            accuracy_score: 1.0,
            consistency_score: 1.0,
            timeliness_score: 1.0,
            overall_score: 1.0,
            issues: Vec::new(),
        }
    }
}