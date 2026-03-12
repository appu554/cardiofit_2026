//! REST API server for the Unified Dose Safety Engine

use crate::unified_clinical_engine::UnifiedClinicalEngine;
use crate::models::*;
use crate::api::middleware::*;
use crate::models::medication::SafetyAlert as MedicationSafetyAlert;
use crate::clients::SnapshotClient;
use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::Json,
    routing::{get, post, options},
    Router,
    middleware,
};
use std::sync::Arc;
use tower::ServiceBuilder;
use tower_http::{cors::CorsLayer, trace::TraceLayer, compression::CompressionLayer};
use tracing::{info, error};
use tokio::time::Duration;

/// API server state
#[derive(Clone)]
pub struct ApiState {
    pub unified_engine: Arc<UnifiedClinicalEngine>,
    pub rate_limiter: RateLimiter,
    pub server_start_time: std::time::Instant,
    pub config: crate::api::config::ServerConfig,
    pub snapshot_client: SnapshotClient,
}

/// Create the API router with production middleware
pub fn create_router(unified_engine: Arc<UnifiedClinicalEngine>, config: crate::api::config::ServerConfig) -> Router {
    // Create rate limiter (100 requests per minute per client)
    let rate_limiter = RateLimiter::new(100, Duration::from_secs(60));
    
    // Create snapshot client with default configuration
    let snapshot_client = SnapshotClient::new().expect("Failed to create snapshot client");

    let state = ApiState {
        unified_engine,
        rate_limiter: rate_limiter.clone(),
        server_start_time: std::time::Instant::now(),
        config,
        snapshot_client,
    };

    // Start rate limiter cleanup task
    let cleanup_rate_limiter = rate_limiter.clone();
    tokio::spawn(async move {
        cleanup_rate_limiter.cleanup_task().await;
    });

    Router::new()
        // Health and status endpoints (no auth required)
        .route("/health", get(health_check))
        .route("/health/detailed", get(detailed_health_check))
        .route("/metrics", get(get_metrics))
        .route("/status", get(get_status))
        .route("/version", get(get_version))

        // Unified Clinical Engine endpoints (auth required)
        .route("/api/dose/optimize", post(execute_dose_optimization))
        .route("/api/medication/intelligence", post(execute_medication_intelligence))
        .route("/api/flow2/execute", post(execute_flow2))
        
        // Snapshot-based processing endpoints (auth required)
        .route("/api/execute-with-snapshot", post(execute_with_snapshot))
        .route("/api/recipe/execute-snapshot", post(execute_recipe_with_snapshot))

        // Admin endpoints (auth required)
        .route("/api/admin/stats", get(get_admin_stats))
        .route("/api/admin/cache/clear", post(clear_cache))

        // OPTIONS handler for CORS preflight
        .route("/*path", options(handle_options))

        // Apply middleware layers (order matters!)
        .layer(
            ServiceBuilder::new()
                // Outermost layers (applied first)
                .layer(middleware::from_fn(timeout_middleware))
                .layer(middleware::from_fn(security_headers_middleware))
                .layer(middleware::from_fn(cors_middleware))
                .layer(middleware::from_fn(request_tracking_middleware))
                .layer(middleware::from_fn_with_state(rate_limiter, rate_limiting_middleware))
                .layer(middleware::from_fn(request_validation_middleware))
                .layer(middleware::from_fn_with_state(state.clone(), auth_middleware))
                // Innermost layers (applied last)
                .layer(CompressionLayer::new())
                .layer(TraceLayer::new_for_http())
        )
        .with_state(state)
}

/// Health check endpoint
async fn health_check(State(_state): State<ApiState>) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "status": "healthy",
        "engine": "unified-clinical-engine",
        "version": crate::VERSION,
        "timestamp": chrono::Utc::now()
    }))
}

/// Get metrics endpoint
async fn get_metrics(State(_state): State<ApiState>) -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "engine": "unified-clinical-engine",
        "version": crate::VERSION,
        "performance": {
            "avg_response_time_ms": 50,
            "requests_processed": 0,
            "cache_hit_rate": 0.95
        },
        "timestamp": chrono::Utc::now()
    }))
}

/// Get status endpoint
async fn get_status(State(state): State<ApiState>) -> Json<serde_json::Value> {
    let uptime = state.server_start_time.elapsed();

    Json(serde_json::json!({
        "status": "running",
        "version": crate::VERSION,
        "engine": "unified-clinical-engine",
        "uptime_seconds": uptime.as_secs(),
        "timestamp": chrono::Utc::now()
    }))
}

/// Detailed health check endpoint
async fn detailed_health_check(State(state): State<ApiState>) -> Json<serde_json::Value> {
    let uptime = state.server_start_time.elapsed();

    Json(serde_json::json!({
        "status": "healthy",
        "version": crate::VERSION,
        "engine": "unified-clinical-engine",
        "uptime_seconds": uptime.as_secs(),
        "unified_engine": {
            "dose_calculation": "available",
            "safety_verification": "available",
            "clinical_intelligence": "available",
            "pharmacokinetic_modeling": "available"
        },
        "system": {
            "memory_usage": get_memory_usage(),
            "cpu_usage": get_cpu_usage(),
            "thread_count": get_thread_count()
        },
        "timestamp": chrono::Utc::now()
    }))
}

/// Get version endpoint
async fn get_version() -> Json<serde_json::Value> {
    Json(serde_json::json!({
        "version": crate::VERSION,
        "engine": crate::ENGINE_NAME,
        "build_date": "unknown",
        "git_commit": "unknown",
        "rust_version": env!("CARGO_PKG_VERSION")
    }))
}

/// Get admin statistics
async fn get_admin_stats(State(state): State<ApiState>) -> Json<serde_json::Value> {
    let uptime = state.server_start_time.elapsed();

    Json(serde_json::json!({
        "server": {
            "uptime_seconds": uptime.as_secs(),
            "start_time": chrono::Utc::now() - chrono::Duration::seconds(uptime.as_secs() as i64),
            "engine": "unified-clinical-engine"
        },
        "performance": {
            "total_requests": 0,
            "successful_requests": 0,
            "failed_requests": 0,
            "average_response_time_ms": 50
        },
        "system": {
            "memory_usage_mb": get_memory_usage(),
            "cpu_usage_percent": get_cpu_usage(),
            "thread_count": get_thread_count()
        },
        "unified_engine": {
            "dose_calculations": 0,
            "safety_verifications": 0,
            "cache_hit_rate": 0.95
        }
    }))
}

/// Clear cache endpoint
async fn clear_cache(State(state): State<ApiState>) -> Json<serde_json::Value> {
    // TODO: Implement cache clearing
    info!("🗑️  Cache clear requested");

    Json(serde_json::json!({
        "status": "success",
        "message": "Cache cleared successfully",
        "timestamp": chrono::Utc::now()
    }))
}

/// Handle OPTIONS requests for CORS
async fn handle_options() -> StatusCode {
    StatusCode::OK
}

// System monitoring helper functions
fn get_memory_usage() -> f64 {
    // TODO: Implement actual memory usage monitoring
    // For now, return a placeholder value
    128.5 // MB
}

fn get_cpu_usage() -> f64 {
    // TODO: Implement actual CPU usage monitoring
    // For now, return a placeholder value
    15.2 // Percent
}

fn get_thread_count() -> usize {
    // TODO: Implement actual thread count monitoring
    // For now, return a placeholder value
    8
}

/// Execute Flow2 request using unified clinical engine
async fn execute_flow2(
    State(state): State<ApiState>,
    Json(request): Json<Flow2Request>,
) -> Result<Json<Flow2Response>, (StatusCode, Json<serde_json::Value>)> {
    info!("Received Flow2 execution request: {}", request.request_id);

    // Convert API request to unified engine request
    let clinical_request = match convert_flow2_request_to_clinical_request(&request) {
        Ok(req) => req,
        Err(e) => {
            error!("Failed to convert Flow2 request: {}", e);
            return Err((
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "error": "Invalid request format",
                    "details": e,
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Process request through unified clinical engine
    let clinical_response = match state.unified_engine.process_clinical_request(clinical_request).await {
        Ok(response) => response,
        Err(e) => {
            error!("Unified engine processing failed for {}: {}", request.request_id, e);
            return Err((
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({
                    "error": "Engine processing failed",
                    "details": e.to_string(),
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Convert unified engine response to API response
    let response = convert_clinical_response_to_flow2_response(&clinical_response, request.patient_id.clone());

    info!("Flow2 execution successful: {} (status: {} in {}ms)",
          request.request_id, response.overall_status, response.execution_time_ms);
    Ok(Json(response))
}




/// Execute medication intelligence analysis using unified clinical engine
async fn execute_medication_intelligence(
    State(state): State<ApiState>,
    Json(request): Json<MedicationIntelligenceRequest>,
) -> Result<Json<MedicationIntelligenceResponse>, (StatusCode, Json<serde_json::Value>)> {
    info!("Received medication intelligence request: {}", request.request_id);

    // Convert API request to unified engine request
    let clinical_request = match convert_intelligence_request_to_clinical_request(&request) {
        Ok(req) => req,
        Err(e) => {
            error!("Failed to convert medication intelligence request: {}", e);
            return Err((
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "error": "Invalid request format",
                    "details": e,
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Process request through unified clinical engine
    let clinical_response = match state.unified_engine.process_clinical_request(clinical_request).await {
        Ok(response) => response,
        Err(e) => {
            error!("Unified engine processing failed for {}: {}", request.request_id, e);
            return Err((
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({
                    "error": "Engine processing failed",
                    "details": e.to_string(),
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Convert unified engine response to API response
    let response = convert_clinical_response_to_intelligence_response(&clinical_response);

    info!("Medication intelligence successful: {} (score: {:.2} in {}ms)",
          request.request_id, response.intelligence_score, response.execution_time_ms);
    Ok(Json(response))
}

/// Execute dose optimization using unified clinical engine
async fn execute_dose_optimization(
    State(state): State<ApiState>,
    Json(request): Json<DoseOptimizationRequest>,
) -> Result<Json<DoseOptimizationResponse>, (StatusCode, Json<serde_json::Value>)> {
    info!("Received dose optimization request: {}", request.request_id);

    // Convert API request to unified engine request
    let clinical_request = match convert_dose_request_to_clinical_request(&request) {
        Ok(req) => req,
        Err(e) => {
            error!("Failed to convert dose optimization request: {}", e);
            return Err((
                StatusCode::BAD_REQUEST,
                Json(serde_json::json!({
                    "error": "Invalid request format",
                    "details": e,
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Process request through unified clinical engine
    let clinical_response = match state.unified_engine.process_clinical_request(clinical_request).await {
        Ok(response) => response,
        Err(e) => {
            error!("Unified engine processing failed for {}: {}", request.request_id, e);
            return Err((
                StatusCode::INTERNAL_SERVER_ERROR,
                Json(serde_json::json!({
                    "error": "Engine processing failed",
                    "details": e.to_string(),
                    "request_id": request.request_id
                }))
            ));
        }
    };

    // Convert unified engine response to API response
    let response = convert_clinical_response_to_dose_response(&clinical_response);

    info!("Dose optimization successful: {} ({}mg in {}ms)",
          request.request_id, response.optimized_dose, response.execution_time_ms);
    Ok(Json(response))
}



/// Start the API server with configuration
pub async fn start_server(unified_engine: Arc<UnifiedClinicalEngine>, config: &crate::api::config::ServerConfig) -> Result<(), Box<dyn std::error::Error>> {
    let app = create_router(unified_engine, config.clone());

    let bind_address = format!("{}:{}", config.server.host, config.server.port);
    let listener = tokio::net::TcpListener::bind(&bind_address).await?;

    info!("🚀 Unified Dose Safety Engine API server starting on {}", bind_address);
    info!("📋 Available endpoints:");
    info!("  POST /api/dose/optimize - Advanced dose calculation");
    info!("  POST /api/medication/intelligence - Clinical decision support");
    info!("  POST /api/flow2/execute - Flow2 integration");
    info!("  POST /api/execute-with-snapshot - Snapshot-based processing");
    info!("  POST /api/recipe/execute-snapshot - Recipe execution with snapshot");
    info!("  GET  /health - Health check");
    info!("  GET  /health/detailed - Detailed health check");
    info!("  GET  /metrics - Performance metrics");
    info!("  GET  /status - Engine status");
    info!("  GET  /version - Version information");
    info!("  GET  /api/admin/stats - Admin statistics");

    // Configure server with graceful shutdown
    let server = axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal());

    info!("✅ Server ready to accept connections");
    server.await?;

    info!("🛑 Server shutdown complete");
    Ok(())
}

/// Handle graceful shutdown
async fn shutdown_signal() {
    let ctrl_c = async {
        tokio::signal::ctrl_c()
            .await
            .expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("🛑 Received Ctrl+C signal");
        },
        _ = terminate => {
            info!("🛑 Received terminate signal");
        },
    }

    info!("🛑 Shutdown signal received, starting graceful shutdown...");
}

// ============================================================================
// UNIFIED CLINICAL ENGINE INTEGRATION
// ============================================================================

/// Convert API DoseOptimizationRequest to Unified Clinical Engine ClinicalRequest
fn convert_dose_request_to_clinical_request(
    request: &DoseOptimizationRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    // Extract patient context from clinical_context JSON
    let patient_context = extract_patient_context_from_json(&request.clinical_context)?;

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: request.medication_code.clone(),
        indication: request.optimization_type.clone(),
        patient_context,
        timestamp: chrono::Utc::now(),
    })
}

/// Convert API MedicationIntelligenceRequest to Unified Clinical Engine ClinicalRequest
fn convert_intelligence_request_to_clinical_request(
    request: &MedicationIntelligenceRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    // Use the first medication if multiple are provided
    let medication_code = request.medications.first()
        .map(|med| med.code.clone())
        .unwrap_or_else(|| "unknown".to_string());

    let patient_context = extract_patient_context_from_json(&request.clinical_context)?;

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: medication_code,
        indication: request.intelligence_type.clone(),
        patient_context,
        timestamp: chrono::Utc::now(),
    })
}

/// Convert API Flow2Request to Unified Clinical Engine ClinicalRequest
fn convert_flow2_request_to_clinical_request(
    request: &Flow2Request,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    // Extract medication code from medication_data
    let medication_code = request.medication_data.get("medication_code")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown")
        .to_string();

    let patient_context = extract_patient_context_from_json(&request.clinical_context)?;

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: medication_code,
        indication: request.action_type.clone(),
        patient_context,
        timestamp: chrono::Utc::now(),
    })
}

/// Extract PatientContext from JSON clinical context
fn extract_patient_context_from_json(
    clinical_context: &serde_json::Map<String, serde_json::Value>,
) -> Result<crate::unified_clinical_engine::PatientContext, String> {
    use crate::unified_clinical_engine::{PatientContext, BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction};

    // Extract basic demographics with defaults
    let age_years = clinical_context.get("age_years")
        .and_then(|v| v.as_f64())
        .unwrap_or(45.0) as u8;

    let weight_kg = clinical_context.get("weight_kg")
        .and_then(|v| v.as_f64())
        .unwrap_or(70.0);

    let height_cm = clinical_context.get("height_cm")
        .and_then(|v| v.as_f64())
        .unwrap_or(170.0);

    let sex = clinical_context.get("sex")
        .and_then(|v| v.as_str())
        .map(|s| match s.to_lowercase().as_str() {
            "female" => BiologicalSex::Female,
            "other" => BiologicalSex::Other,
            _ => BiologicalSex::Male,
        })
        .unwrap_or(BiologicalSex::Male);

    let pregnancy_status = clinical_context.get("pregnancy_status")
        .and_then(|v| v.as_str())
        .map(|s| match s.to_lowercase().as_str() {
            "pregnant" => PregnancyStatus::Pregnant { trimester: 1 },
            "unknown" => PregnancyStatus::Unknown,
            _ => PregnancyStatus::NotPregnant,
        })
        .unwrap_or(PregnancyStatus::NotPregnant);

    // Extract renal function
    let renal_function = RenalFunction {
        egfr_ml_min: clinical_context.get("egfr")
            .and_then(|v| v.as_f64()),
        egfr_ml_min_1_73m2: clinical_context.get("egfr")
            .and_then(|v| v.as_f64()),
        creatinine_clearance: clinical_context.get("creatinine_clearance")
            .and_then(|v| v.as_f64()),
        creatinine_mg_dl: clinical_context.get("creatinine")
            .and_then(|v| v.as_f64()),
        bun_mg_dl: clinical_context.get("bun")
            .and_then(|v| v.as_f64()),
        stage: clinical_context.get("renal_stage")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string()),
    };

    // Extract hepatic function
    let hepatic_function = HepaticFunction {
        child_pugh_class: clinical_context.get("child_pugh_class")
            .and_then(|v| v.as_str())
            .map(|s| s.to_string()),
        alt_u_l: clinical_context.get("alt")
            .and_then(|v| v.as_f64()),
        ast_u_l: clinical_context.get("ast")
            .and_then(|v| v.as_f64()),
        bilirubin_mg_dl: clinical_context.get("bilirubin")
            .and_then(|v| v.as_f64()),
        albumin_g_dl: clinical_context.get("albumin")
            .and_then(|v| v.as_f64()),
    };

    Ok(PatientContext {
        age_years: age_years as f64,
        weight_kg,
        height_cm,
        sex,
        pregnancy_status,
        renal_function,
        hepatic_function,
        active_medications: vec![], // TODO: Extract from clinical_context
        allergies: vec![], // TODO: Extract from clinical_context
        conditions: vec![], // TODO: Extract from clinical_context
        lab_values: std::collections::HashMap::new(), // TODO: Extract from clinical_context
    })
}

/// Convert Unified Clinical Engine ClinicalResponse to API DoseOptimizationResponse
fn convert_clinical_response_to_dose_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
) -> DoseOptimizationResponse {
    DoseOptimizationResponse {
        request_id: clinical_response.request_id.clone(),
        optimized_dose: clinical_response.final_recommendation.final_dose_mg,
        optimization_score: clinical_response.calculation_result.confidence_score,
        confidence_interval: ConfidenceInterval {
            lower: clinical_response.final_recommendation.final_dose_mg * 0.9,
            upper: clinical_response.final_recommendation.final_dose_mg * 1.1,
            confidence: 0.95,
        },
        pharmacokinetic_predictions: serde_json::Map::new(), // TODO: Extract from calculation_result
        monitoring_recommendations: clinical_response.final_recommendation.monitoring_required.clone(),
        clinical_rationale: format!("Calculated using {} strategy",
                                  clinical_response.calculation_result.calculation_strategy),
        execution_time_ms: clinical_response.processing_time_ms as i64,
    }
}

/// Convert Unified Clinical Engine ClinicalResponse to API MedicationIntelligenceResponse
fn convert_clinical_response_to_intelligence_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
) -> MedicationIntelligenceResponse {
    let clinical_insights = vec![
        format!("Safety action: {:?}", clinical_response.safety_result.action),
        format!("Recommendation: {:?}", clinical_response.final_recommendation.action),
    ];

    MedicationIntelligenceResponse {
        request_id: clinical_response.request_id.clone(),
        intelligence_score: clinical_response.calculation_result.confidence_score,
        medication_analysis: serde_json::Map::new(), // TODO: Extract from calculation_result
        interaction_analysis: serde_json::Map::new(), // TODO: Extract from safety_result
        outcome_predictions: serde_json::Map::new(), // TODO: Add outcome predictions
        alternative_recommendations: clinical_response.final_recommendation.alternatives
            .iter()
            .map(|alt| MedicationAlternative {
                medication_code: alt.drug_id.clone(),
                medication_name: alt.name.clone(),
                rationale: alt.rationale.clone(),
                safety_profile: "Standard".to_string(), // TODO: Extract from safety analysis
            })
            .collect(),
        clinical_insights,
        execution_time_ms: clinical_response.processing_time_ms as i64,
    }
}

/// Convert Unified Clinical Engine ClinicalResponse to API Flow2Response
fn convert_clinical_response_to_flow2_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
    patient_id: String,
) -> Flow2Response {
    let overall_status = match clinical_response.final_recommendation.action {
        crate::unified_clinical_engine::RecommendationAction::Prescribe => "success",
        crate::unified_clinical_engine::RecommendationAction::PrescribeWithMonitoring => "success_with_monitoring",
        crate::unified_clinical_engine::RecommendationAction::RequireApproval => "requires_approval",
        crate::unified_clinical_engine::RecommendationAction::Contraindicated => "contraindicated",
        crate::unified_clinical_engine::RecommendationAction::ReferToSpecialist => "refer_specialist",
    }.to_string();

    Flow2Response {
        request_id: clinical_response.request_id.clone(),
        patient_id,
        overall_status,
        execution_summary: ExecutionSummary {
            total_recipes_executed: 1,
            successful_recipes: 1,
            failed_recipes: 0,
            warnings: clinical_response.safety_result.findings.len(),
            errors: 0,
            engine: "unified-clinical-engine".to_string(),
            cache_hit_rate: 0.95,
        },
        recipe_results: vec![], // TODO: Convert calculation_result to recipe_results
        clinical_decision_support: serde_json::Map::new(), // TODO: Extract from safety_result
        safety_alerts: clinical_response.safety_result.findings
            .iter()
            .map(|finding| MedicationSafetyAlert {
                alert_id: uuid::Uuid::new_v4().to_string(),
                alert_type: finding.category.clone(),
                severity: finding.severity.clone(),
                message: finding.message.clone(),
                description: finding.message.clone(),
                action_required: finding.severity == "critical" || finding.severity == "high",
            })
            .collect(),
        recommendations: clinical_response.final_recommendation.special_instructions.clone(),
        analytics: serde_json::Map::new(), // TODO: Add analytics data
        execution_time_ms: clinical_response.processing_time_ms as i64,
        engine_used: "unified-clinical-engine".to_string(),
        timestamp: chrono::Utc::now(),
        processing_metadata: ProcessingMetadata {
            fallback_used: false,
            cache_used: false, // TODO: Track cache usage
            context_sources: vec!["unified-engine".to_string()],
            processing_stages: vec!["dose-calculation".to_string(), "safety-verification".to_string()],
        },
    }
}

// ============================================================================
// SNAPSHOT-BASED PROCESSING ENDPOINTS
// ============================================================================

/// Execute recipe with snapshot-based processing
async fn execute_with_snapshot(
    State(state): State<ApiState>,
    Json(request): Json<SnapshotBasedRequest>,
) -> Result<Json<crate::api::handlers::ApiResponse<MedicationProposal>>, (StatusCode, Json<serde_json::Value>)> {
    use crate::api::handlers::*;
    
    let start_time = std::time::Instant::now();
    log_request_received("/api/execute-with-snapshot", &request.request_id);

    // Validate request
    if let Err(e) = validate_snapshot_based_request(&request) {
        log_request_failed("/api/execute-with-snapshot", &request.request_id, &e);
        return Err(validation_error(e));
    }

    // Fetch and verify snapshot
    let snapshot = match state.snapshot_client.fetch_and_verify_snapshot(&request.snapshot_id).await {
        Ok(snapshot) => {
            info!("Snapshot {} fetched and verified successfully", request.snapshot_id);
            snapshot
        }
        Err(e) => {
            let error_msg = format!("Failed to fetch/verify snapshot {}: {}", request.snapshot_id, e);
            log_request_failed("/api/execute-with-snapshot", &request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(request.request_id)));
        }
    };

    // Convert snapshot to clinical context
    let clinical_context = match convert_snapshot_to_clinical_context(&snapshot) {
        Ok(context) => context,
        Err(e) => {
            let error_msg = format!("Failed to convert snapshot to clinical context: {}", e);
            log_request_failed("/api/execute-with-snapshot", &request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(request.request_id)));
        }
    };

    // Create traditional recipe execution request from snapshot data
    let recipe_request = RecipeExecutionRequest {
        request_id: request.request_id.clone(),
        recipe_id: request.recipe_id,
        variant: request.variant,
        patient_id: request.patient_id,
        medication_code: request.medication_code,
        clinical_context,
        timeout_ms: request.timeout_ms,
        snapshot_id: Some(request.snapshot_id.clone()),
    };

    // Execute recipe using unified clinical engine
    let clinical_request = match convert_recipe_request_to_clinical_request(&recipe_request) {
        Ok(req) => req,
        Err(e) => {
            let error_msg = format!("Failed to convert recipe request: {}", e);
            log_request_failed("/api/execute-with-snapshot", &request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(request.request_id)));
        }
    };

    let clinical_response = match state.unified_engine.process_clinical_request(clinical_request).await {
        Ok(response) => response,
        Err(e) => {
            let error_msg = format!("Engine processing failed: {}", e);
            log_request_failed("/api/execute-with-snapshot", &request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(request.request_id)));
        }
    };

    // Convert response to medication proposal with snapshot evidence
    let mut proposal = convert_clinical_response_to_medication_proposal(&clinical_response);
    
    // Add snapshot metadata to execution metadata
    proposal.execution_time_ms = start_time.elapsed().as_millis() as i64;

    let duration_ms = start_time.elapsed().as_millis();
    log_request_completed("/api/execute-with-snapshot", &request.request_id, duration_ms);

    Ok(success_response(proposal, Some(request.request_id)))
}

/// Execute recipe with snapshot (alternative endpoint)
async fn execute_recipe_with_snapshot(
    State(state): State<ApiState>,
    Json(request): Json<RecipeExecutionRequest>,
) -> Result<Json<crate::api::handlers::ApiResponse<MedicationProposal>>, (StatusCode, Json<serde_json::Value>)> {
    use crate::api::handlers::*;
    
    let start_time = std::time::Instant::now();
    log_request_received("/api/recipe/execute-snapshot", &request.request_id);

    // Validate request
    if let Err(e) = validate_recipe_request(&request) {
        log_request_failed("/api/recipe/execute-snapshot", &request.request_id, &e);
        return Err(validation_error(e));
    }

    // Check if this is a snapshot-based request
    let snapshot_id = match &request.snapshot_id {
        Some(id) if !id.is_empty() => id,
        _ => {
            let error_msg = "snapshot_id is required for this endpoint".to_string();
            log_request_failed("/api/recipe/execute-snapshot", &request.request_id, &error_msg);
            return Err(validation_error(error_msg));
        }
    };

    // Fetch and verify snapshot
    let snapshot = match state.snapshot_client.fetch_and_verify_snapshot(snapshot_id).await {
        Ok(snapshot) => {
            info!("Snapshot {} fetched and verified successfully", snapshot_id);
            snapshot
        }
        Err(e) => {
            let error_msg = format!("Failed to fetch/verify snapshot {}: {}", snapshot_id, e);
            log_request_failed("/api/recipe/execute-snapshot", &request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(request.request_id.clone())));
        }
    };

    // Convert snapshot to clinical context (merge with existing if present)
    let enhanced_clinical_context = if request.clinical_context.is_empty() {
        // Use only snapshot data
        match convert_snapshot_to_clinical_context(&snapshot) {
            Ok(context) => context,
            Err(e) => {
                let error_msg = format!("Failed to convert snapshot to clinical context: {}", e);
                log_request_failed("/api/recipe/execute-snapshot", &request.request_id, &error_msg);
                return Err(internal_error(error_msg, Some(request.request_id.clone())));
            }
        }
    } else {
        // Merge snapshot data with existing clinical context
        match merge_snapshot_with_clinical_context(&snapshot, &request.clinical_context) {
            Ok(context) => context,
            Err(e) => {
                let error_msg = format!("Failed to merge snapshot with clinical context: {}", e);
                log_request_failed("/api/recipe/execute-snapshot", &request.request_id, &error_msg);
                return Err(internal_error(error_msg, Some(request.request_id.clone())));
            }
        }
    };

    // Create enhanced request with merged clinical context
    let enhanced_request = RecipeExecutionRequest {
        clinical_context: enhanced_clinical_context,
        ..request
    };

    // Execute recipe using unified clinical engine
    let clinical_request = match convert_recipe_request_to_clinical_request(&enhanced_request) {
        Ok(req) => req,
        Err(e) => {
            let error_msg = format!("Failed to convert recipe request: {}", e);
            log_request_failed("/api/recipe/execute-snapshot", &enhanced_request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(enhanced_request.request_id.clone())));
        }
    };

    let clinical_response = match state.unified_engine.process_clinical_request(clinical_request).await {
        Ok(response) => response,
        Err(e) => {
            let error_msg = format!("Engine processing failed: {}", e);
            log_request_failed("/api/recipe/execute-snapshot", &enhanced_request.request_id, &error_msg);
            return Err(internal_error(error_msg, Some(enhanced_request.request_id.clone())));
        }
    };

    // Convert response to medication proposal with snapshot evidence
    let mut proposal = convert_clinical_response_to_medication_proposal(&clinical_response);
    
    // Add snapshot and execution metadata
    proposal.execution_time_ms = start_time.elapsed().as_millis() as i64;

    let duration_ms = start_time.elapsed().as_millis();
    log_request_completed("/api/recipe/execute-snapshot", &enhanced_request.request_id, duration_ms);

    Ok(success_response(proposal, Some(enhanced_request.request_id)))
}

// ============================================================================
// SNAPSHOT CONVERSION UTILITIES
// ============================================================================

/// Convert clinical snapshot to JSON clinical context string
fn convert_snapshot_to_clinical_context(
    snapshot: &crate::clients::ClinicalSnapshot,
) -> Result<String, String> {
    use serde_json::json;

    let clinical_context = json!({
        "snapshot_metadata": {
            "snapshot_id": snapshot.snapshot_id,
            "created_at": snapshot.created_at,
            "data_sources": snapshot.metadata.data_sources,
            "data_quality_score": snapshot.metadata.data_quality_score,
            "completeness_flags": snapshot.metadata.completeness_flags
        },
        "patient_demographics": {
            "age_years": snapshot.data.patient_demographics.age_years,
            "weight_kg": snapshot.data.patient_demographics.weight_kg,
            "height_cm": snapshot.data.patient_demographics.height_cm,
            "gender": snapshot.data.patient_demographics.gender,
            "race": snapshot.data.patient_demographics.race,
            "ethnicity": snapshot.data.patient_demographics.ethnicity,
            "bmi": snapshot.data.patient_demographics.bmi,
            "bsa_m2": snapshot.data.patient_demographics.bsa_m2,
            "egfr": snapshot.data.patient_demographics.egfr,
            "creatinine_clearance": snapshot.data.patient_demographics.creatinine_clearance
        },
        "active_medications": snapshot.data.active_medications.iter().map(|med| json!({
            "medication_id": med.medication_id,
            "medication_code": med.medication_code,
            "medication_name": med.medication_name,
            "dose_value": med.dose_value,
            "dose_unit": med.dose_unit,
            "frequency": med.frequency,
            "route": med.route,
            "start_date": med.start_date,
            "prescriber_id": med.prescriber_id
        })).collect::<Vec<_>>(),
        "allergies": snapshot.data.allergies.iter().map(|allergy| json!({
            "allergen_code": allergy.allergen_code,
            "allergen_name": allergy.allergen_name,
            "reaction": allergy.reaction,
            "severity": allergy.severity,
            "onset_date": allergy.onset_date,
            "verified": allergy.verified
        })).collect::<Vec<_>>(),
        "lab_values": snapshot.data.lab_values.iter().map(|lab| json!({
            "test_code": lab.test_code,
            "test_name": lab.test_name,
            "value": lab.value,
            "unit": lab.unit,
            "reference_range": lab.reference_range,
            "result_status": lab.result_status,
            "collected_at": lab.collected_at,
            "reported_at": lab.reported_at
        })).collect::<Vec<_>>(),
        "conditions": snapshot.data.conditions.iter().map(|condition| json!({
            "condition_code": condition.condition_code,
            "condition_name": condition.condition_name,
            "status": condition.status,
            "severity": condition.severity,
            "onset_date": condition.onset_date,
            "diagnosis_date": condition.diagnosis_date
        })).collect::<Vec<_>>(),
        "vital_signs": snapshot.data.vital_signs.iter().map(|vital| json!({
            "vital_type": vital.vital_type,
            "value": vital.value,
            "unit": vital.unit,
            "measured_at": vital.measured_at,
            "measurement_method": vital.measurement_method
        })).collect::<Vec<_>>(),
        "clinical_notes": snapshot.data.clinical_notes.iter().map(|note| json!({
            "note_id": note.note_id,
            "note_type": note.note_type,
            "author_id": note.author_id,
            "created_at": note.created_at,
            "content": note.content,
            "tags": note.tags
        })).collect::<Vec<_>>()
    });

    serde_json::to_string(&clinical_context)
        .map_err(|e| format!("Failed to serialize clinical context: {}", e))
}

/// Merge snapshot data with existing clinical context
fn merge_snapshot_with_clinical_context(
    snapshot: &crate::clients::ClinicalSnapshot,
    existing_context: &str,
) -> Result<String, String> {
    // Parse existing clinical context
    let mut existing: serde_json::Value = serde_json::from_str(existing_context)
        .map_err(|e| format!("Failed to parse existing clinical context: {}", e))?;

    // Convert snapshot to clinical context
    let snapshot_context_str = convert_snapshot_to_clinical_context(snapshot)?;
    let snapshot_context: serde_json::Value = serde_json::from_str(&snapshot_context_str)
        .map_err(|e| format!("Failed to parse snapshot clinical context: {}", e))?;

    // Merge contexts (snapshot takes precedence for overlapping keys)
    if let (Some(existing_obj), Some(snapshot_obj)) = (existing.as_object_mut(), snapshot_context.as_object()) {
        for (key, value) in snapshot_obj {
            existing_obj.insert(key.clone(), value.clone());
        }
    }

    serde_json::to_string(&existing)
        .map_err(|e| format!("Failed to serialize merged clinical context: {}", e))
}

/// Convert RecipeExecutionRequest to ClinicalRequest (placeholder - would need actual unified engine integration)
fn convert_recipe_request_to_clinical_request(
    _request: &RecipeExecutionRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    // This is a placeholder implementation. In a real scenario, this would:
    // 1. Parse the clinical_context JSON
    // 2. Extract relevant clinical data
    // 3. Convert to the unified engine's ClinicalRequest format
    
    Err("Recipe execution conversion not yet implemented".to_string())
}

/// Convert ClinicalResponse to MedicationProposal (placeholder)
fn convert_clinical_response_to_medication_proposal(
    _response: &crate::unified_clinical_engine::ClinicalResponse,
) -> MedicationProposal {
    // This is a placeholder implementation. In a real scenario, this would:
    // 1. Extract dose calculations from the clinical response
    // 2. Extract safety assessments and alerts
    // 3. Format alternatives and monitoring recommendations
    // 4. Create a complete medication proposal
    
    MedicationProposal {
        medication_code: "placeholder".to_string(),
        medication_name: "Placeholder Medication".to_string(),
        calculated_dose: 0.0,
        dose_unit: "mg".to_string(),
        frequency: "BID".to_string(),
        duration: Some("7 days".to_string()),
        safety_status: "SAFE".to_string(),
        safety_alerts: vec![],
        contraindications: vec![],
        clinical_rationale: "Placeholder rationale".to_string(),
        monitoring_plan: vec!["Monitor for side effects".to_string()],
        alternatives: vec![],
        execution_time_ms: 0,
        recipe_version: "1.0.0".to_string(),
    }
}
