//! API response models and utilities

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

/// Standard error response
#[derive(Debug, Serialize, Deserialize)]
pub struct ErrorResponse {
    pub error: String,
    pub message: String,
    pub request_id: Option<String>,
    pub timestamp: DateTime<Utc>,
    pub error_code: Option<String>,
}

impl ErrorResponse {
    /// Create a new error response
    pub fn new(error: String, message: String) -> Self {
        Self {
            error,
            message,
            request_id: None,
            timestamp: Utc::now(),
            error_code: None,
        }
    }

    /// Create error response with request ID
    pub fn with_request_id(mut self, request_id: String) -> Self {
        self.request_id = Some(request_id);
        self
    }

    /// Create error response with error code
    pub fn with_error_code(mut self, error_code: String) -> Self {
        self.error_code = Some(error_code);
        self
    }
}

/// Health check response
#[derive(Debug, Serialize, Deserialize)]
pub struct HealthResponse {
    pub status: String,
    pub version: String,
    pub engine: String,
    pub timestamp: DateTime<Utc>,
    pub uptime_seconds: u64,
    pub knowledge_base: HealthKnowledgeBase,
    pub performance: HealthPerformance,
}

/// Knowledge base health information
#[derive(Debug, Serialize, Deserialize)]
pub struct HealthKnowledgeBase {
    pub loaded: bool,
    pub total_items: usize,
    pub rules_count: usize,
    pub recipes_count: usize,
    pub medications_count: usize,
    pub last_updated: String,
}

/// Performance health information
#[derive(Debug, Serialize, Deserialize)]
pub struct HealthPerformance {
    pub average_response_time_ms: f64,
    pub total_requests: u64,
    pub successful_requests: u64,
    pub failed_requests: u64,
    pub cache_hit_rate: f64,
}

/// Metrics response
#[derive(Debug, Serialize, Deserialize)]
pub struct MetricsResponse {
    pub timestamp: DateTime<Utc>,
    pub version: String,
    pub engine: String,
    pub performance: PerformanceMetrics,
    pub knowledge_base: KnowledgeBaseMetrics,
    pub system: SystemMetrics,
}

/// Performance metrics
#[derive(Debug, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    pub total_requests: u64,
    pub successful_requests: u64,
    pub failed_requests: u64,
    pub average_response_time_ms: f64,
    pub p95_response_time_ms: f64,
    pub p99_response_time_ms: f64,
    pub requests_per_second: f64,
    pub cache_hit_rate: f64,
}

/// Knowledge base metrics
#[derive(Debug, Serialize, Deserialize)]
pub struct KnowledgeBaseMetrics {
    pub total_items: usize,
    pub rules_count: usize,
    pub recipes_count: usize,
    pub medications_count: usize,
    pub evidence_count: usize,
    pub context_recipes_count: usize,
    pub formulary_count: usize,
    pub monitoring_profiles_count: usize,
    pub last_loaded: DateTime<Utc>,
    pub load_time_ms: u64,
}

/// System metrics
#[derive(Debug, Serialize, Deserialize)]
pub struct SystemMetrics {
    pub memory_usage_mb: f64,
    pub cpu_usage_percent: f64,
    pub uptime_seconds: u64,
    pub thread_count: usize,
    pub gc_collections: u64,
}

/// Status response
#[derive(Debug, Serialize, Deserialize)]
pub struct StatusResponse {
    pub status: String,
    pub version: String,
    pub engine: String,
    pub timestamp: DateTime<Utc>,
    pub components: ComponentStatus,
}

/// Component status
#[derive(Debug, Serialize, Deserialize)]
pub struct ComponentStatus {
    pub knowledge_base: String,
    pub rule_evaluator: String,
    pub recipe_executor: String,
    pub api_server: String,
}

/// Knowledge summary response
#[derive(Debug, Serialize, Deserialize)]
pub struct KnowledgeSummaryResponse {
    pub timestamp: DateTime<Utc>,
    pub version: String,
    pub summary: crate::models::KnowledgeSummary,
    pub details: KnowledgeDetails,
}

/// Knowledge details
#[derive(Debug, Serialize, Deserialize)]
pub struct KnowledgeDetails {
    pub medication_knowledge_core: KnowledgeBaseDetail,
    pub evidence_repository: KnowledgeBaseDetail,
    pub orb_rules: KnowledgeBaseDetail,
    pub context_recipes: KnowledgeBaseDetail,
    pub clinical_recipes: KnowledgeBaseDetail,
    pub formulary_database: KnowledgeBaseDetail,
    pub monitoring_database: KnowledgeBaseDetail,
}

/// Knowledge base detail
#[derive(Debug, Serialize, Deserialize)]
pub struct KnowledgeBaseDetail {
    pub name: String,
    pub version: String,
    pub item_count: usize,
    pub last_updated: String,
    pub source: String,
    pub description: String,
}

/// Validation response
#[derive(Debug, Serialize, Deserialize)]
pub struct ValidationResponse {
    pub status: String,
    pub timestamp: DateTime<Utc>,
    pub validation_results: ValidationResults,
}

/// Validation results
#[derive(Debug, Serialize, Deserialize)]
pub struct ValidationResults {
    pub total_rules: usize,
    pub valid_rules: usize,
    pub invalid_rules: usize,
    pub warnings: Vec<ValidationWarning>,
    pub errors: Vec<ValidationError>,
}

/// Validation warning
#[derive(Debug, Serialize, Deserialize)]
pub struct ValidationWarning {
    pub rule_id: String,
    pub message: String,
    pub severity: String,
}

/// Validation error
#[derive(Debug, Serialize, Deserialize)]
pub struct ValidationError {
    pub rule_id: String,
    pub message: String,
    pub error_type: String,
}

/// Create a standardized success response
pub fn create_success_response<T: Serialize>(data: T, request_id: Option<String>) -> serde_json::Value {
    let mut response = serde_json::json!({
        "success": true,
        "data": data,
        "timestamp": Utc::now()
    });

    if let Some(id) = request_id {
        response["request_id"] = serde_json::Value::String(id);
    }

    response
}

/// Create a standardized error response
pub fn create_error_response(error: String, message: String, request_id: Option<String>) -> serde_json::Value {
    let mut response = serde_json::json!({
        "success": false,
        "error": error,
        "message": message,
        "timestamp": Utc::now()
    });

    if let Some(id) = request_id {
        response["request_id"] = serde_json::Value::String(id);
    }

    response
}

/// Create a paginated response
pub fn create_paginated_response<T: Serialize>(
    data: Vec<T>,
    page: usize,
    page_size: usize,
    total_items: usize,
) -> serde_json::Value {
    let total_pages = (total_items + page_size - 1) / page_size;
    
    serde_json::json!({
        "success": true,
        "data": data,
        "pagination": {
            "page": page,
            "page_size": page_size,
            "total_items": total_items,
            "total_pages": total_pages,
            "has_next": page < total_pages,
            "has_previous": page > 1
        },
        "timestamp": Utc::now()
    })
}
