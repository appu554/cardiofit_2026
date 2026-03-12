//! API request handlers

use crate::models::*;
use axum::{http::StatusCode, response::Json};
use serde_json;
use tracing::{info, warn, error};

/// Standard API response wrapper
#[derive(Debug, serde::Serialize)]
pub struct ApiResponse<T> {
    pub success: bool,
    pub data: Option<T>,
    pub error: Option<String>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub request_id: Option<String>,
}

impl<T> ApiResponse<T> {
    /// Create a successful response
    pub fn success(data: T) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
            timestamp: chrono::Utc::now(),
            request_id: None,
        }
    }

    /// Create a successful response with request ID
    pub fn success_with_id(data: T, request_id: String) -> Self {
        Self {
            success: true,
            data: Some(data),
            error: None,
            timestamp: chrono::Utc::now(),
            request_id: Some(request_id),
        }
    }

    /// Create an error response
    pub fn error(message: String) -> ApiResponse<()> {
        ApiResponse {
            success: false,
            data: None,
            error: Some(message),
            timestamp: chrono::Utc::now(),
            request_id: None,
        }
    }

    /// Create an error response with request ID
    pub fn error_with_id(message: String, request_id: String) -> ApiResponse<()> {
        ApiResponse {
            success: false,
            data: None,
            error: Some(message),
            timestamp: chrono::Utc::now(),
            request_id: Some(request_id),
        }
    }
}

/// Validate recipe execution request
pub fn validate_recipe_request(request: &RecipeExecutionRequest) -> Result<(), String> {
    if request.request_id.is_empty() {
        return Err("request_id is required".to_string());
    }
    if request.recipe_id.is_empty() {
        return Err("recipe_id is required".to_string());
    }
    if request.patient_id.is_empty() {
        return Err("patient_id is required".to_string());
    }
    if request.medication_code.is_empty() {
        return Err("medication_code is required".to_string());
    }
    
    // For snapshot-based requests, snapshot_id is required instead of clinical_context
    if let Some(ref snapshot_id) = request.snapshot_id {
        if snapshot_id.is_empty() {
            return Err("snapshot_id cannot be empty".to_string());
        }
        // For snapshot-based requests, clinical_context is optional
    } else {
        // For traditional requests, clinical_context is required
        if request.clinical_context.is_empty() {
            return Err("clinical_context is required when snapshot_id is not provided".to_string());
        }
        
        // Validate clinical context is valid JSON
        if let Err(_) = serde_json::from_str::<serde_json::Value>(&request.clinical_context) {
            return Err("clinical_context must be valid JSON".to_string());
        }
    }

    Ok(())
}

/// Validate snapshot-based request
pub fn validate_snapshot_based_request(request: &SnapshotBasedRequest) -> Result<(), String> {
    if request.request_id.is_empty() {
        return Err("request_id is required".to_string());
    }
    if request.recipe_id.is_empty() {
        return Err("recipe_id is required".to_string());
    }
    if request.patient_id.is_empty() {
        return Err("patient_id is required".to_string());
    }
    if request.medication_code.is_empty() {
        return Err("medication_code is required".to_string());
    }
    if request.snapshot_id.is_empty() {
        return Err("snapshot_id is required".to_string());
    }

    // Validate snapshot ID format (basic validation)
    if !is_valid_snapshot_id(&request.snapshot_id) {
        return Err("invalid snapshot_id format".to_string());
    }

    Ok(())
}

/// Validate snapshot ID format
pub fn is_valid_snapshot_id(snapshot_id: &str) -> bool {
    // Basic validation: should be non-empty, reasonable length, alphanumeric with dashes/underscores
    !snapshot_id.is_empty() && 
    snapshot_id.len() >= 8 && 
    snapshot_id.len() <= 128 &&
    snapshot_id.chars().all(|c| c.is_alphanumeric() || c == '-' || c == '_')
}

/// Validate Flow2 request
pub fn validate_flow2_request(request: &Flow2Request) -> Result<(), String> {
    if request.request_id.is_empty() {
        return Err("request_id is required".to_string());
    }
    if request.patient_id.is_empty() {
        return Err("patient_id is required".to_string());
    }
    if request.action_type.is_empty() {
        return Err("action_type is required".to_string());
    }

    // Validate action type
    match request.action_type.as_str() {
        "MEDICATION_ANALYSIS" | "DOSE_OPTIMIZATION" | "SAFETY_CHECK" => {}
        _ => return Err(format!("Invalid action_type: {}", request.action_type)),
    }

    Ok(())
}

/// Validate medication intelligence request
pub fn validate_medication_intelligence_request(request: &MedicationIntelligenceRequest) -> Result<(), String> {
    if request.request_id.is_empty() {
        return Err("request_id is required".to_string());
    }
    if request.patient_id.is_empty() {
        return Err("patient_id is required".to_string());
    }
    if request.medications.is_empty() {
        return Err("medications list cannot be empty".to_string());
    }

    // Validate intelligence type
    match request.intelligence_type.as_str() {
        "basic" | "comprehensive" | "advanced" => {}
        _ => return Err(format!("Invalid intelligence_type: {}", request.intelligence_type)),
    }

    // Validate analysis depth
    match request.analysis_depth.as_str() {
        "shallow" | "deep" | "comprehensive" => {}
        _ => return Err(format!("Invalid analysis_depth: {}", request.analysis_depth)),
    }

    Ok(())
}

/// Validate dose optimization request
pub fn validate_dose_optimization_request(request: &DoseOptimizationRequest) -> Result<(), String> {
    if request.request_id.is_empty() {
        return Err("request_id is required".to_string());
    }
    if request.patient_id.is_empty() {
        return Err("patient_id is required".to_string());
    }
    if request.medication_code.is_empty() {
        return Err("medication_code is required".to_string());
    }

    // Validate optimization type
    match request.optimization_type.as_str() {
        "standard" | "ml_guided" | "pharmacokinetic" => {}
        _ => return Err(format!("Invalid optimization_type: {}", request.optimization_type)),
    }

    Ok(())
}

/// Create error response for validation failures
pub fn validation_error(message: String) -> (StatusCode, Json<ApiResponse<()>>) {
    warn!("Validation error: {}", message);
    (
        StatusCode::BAD_REQUEST,
        Json(ApiResponse::<()>::error(format!("Validation error: {}", message)))
    )
}

/// Create error response for internal server errors
pub fn internal_error(message: String, request_id: Option<String>) -> (StatusCode, Json<ApiResponse<()>>) {
    error!("Internal server error: {}", message);
    let response = match request_id {
        Some(id) => ApiResponse::<()>::error_with_id(format!("Internal server error: {}", message), id),
        None => ApiResponse::<()>::error(format!("Internal server error: {}", message)),
    };
    (StatusCode::INTERNAL_SERVER_ERROR, Json(response))
}

/// Create validation error response as JSON Value (for compatibility with server endpoints)
pub fn validation_error_json(message: String) -> (StatusCode, Json<serde_json::Value>) {
    warn!("Validation error: {}", message);
    let response = ApiResponse::<()>::error(format!("Validation error: {}", message));
    let json_value = serde_json::to_value(response).unwrap();
    (StatusCode::BAD_REQUEST, Json(json_value))
}

/// Create internal error response as JSON Value (for compatibility with server endpoints)
pub fn internal_error_json(message: String, request_id: Option<String>) -> (StatusCode, Json<serde_json::Value>) {
    error!("Internal server error: {}", message);
    let response = match request_id {
        Some(id) => ApiResponse::<()>::error_with_id(format!("Internal server error: {}", message), id),
        None => ApiResponse::<()>::error(format!("Internal server error: {}", message)),
    };
    let json_value = serde_json::to_value(response).unwrap();
    (StatusCode::INTERNAL_SERVER_ERROR, Json(json_value))
}

/// Create success response
pub fn success_response<T: serde::Serialize>(data: T, request_id: Option<String>) -> Json<ApiResponse<T>> {
    let response = match request_id {
        Some(id) => ApiResponse::success_with_id(data, id),
        None => ApiResponse::success(data),
    };
    Json(response)
}

/// Log request received
pub fn log_request_received(endpoint: &str, request_id: &str) {
    info!("📥 Request received: {} - {}", endpoint, request_id);
}

/// Log request completed
pub fn log_request_completed(endpoint: &str, request_id: &str, duration_ms: u128) {
    info!("✅ Request completed: {} - {} ({}ms)", endpoint, request_id, duration_ms);
}

/// Log request failed
pub fn log_request_failed(endpoint: &str, request_id: &str, error: &str) {
    error!("❌ Request failed: {} - {} - {}", endpoint, request_id, error);
}

/// Extract request ID from various request types
pub fn extract_request_id(request: &serde_json::Value) -> Option<String> {
    request.get("request_id")
        .and_then(|v| v.as_str())
        .map(|s| s.to_string())
}

/// Sanitize error message for API response
pub fn sanitize_error_message(error: &str) -> String {
    // Remove sensitive information and stack traces
    let sanitized = error
        .lines()
        .take(1) // Only take the first line
        .collect::<Vec<_>>()
        .join("");
    
    // Limit length
    if sanitized.len() > 200 {
        format!("{}...", &sanitized[..200])
    } else {
        sanitized
    }
}

/// Create a standardized health check response
pub fn create_health_response(status: &str, details: serde_json::Value) -> serde_json::Value {
    serde_json::json!({
        "status": status,
        "timestamp": chrono::Utc::now(),
        "version": crate::VERSION,
        "engine": crate::ENGINE_NAME,
        "details": details
    })
}

/// Create a standardized metrics response
pub fn create_metrics_response(metrics: serde_json::Value) -> serde_json::Value {
    serde_json::json!({
        "timestamp": chrono::Utc::now(),
        "version": crate::VERSION,
        "engine": crate::ENGINE_NAME,
        "metrics": metrics
    })
}
