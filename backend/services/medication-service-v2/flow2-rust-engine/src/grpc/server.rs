//! gRPC server implementation for Flow2 Rust Engine

use std::sync::Arc;
use std::pin::Pin;
use std::collections::HashMap;
use tonic::{Request, Response, Status, Streaming};
use tokio::sync::mpsc;
use tokio_stream::wrappers::ReceiverStream;
use tokio_stream::Stream;
use tracing::{info, error, warn};
use chrono::Utc;

use crate::unified_clinical_engine::UnifiedClinicalEngine;
use crate::grpc::flow2::{
    flow2_engine_server::Flow2Engine,
    *,
};

/// Flow2 gRPC server implementation
pub struct Flow2GrpcServer {
    unified_engine: Arc<UnifiedClinicalEngine>,
}

impl Flow2GrpcServer {
    /// Create a new Flow2 gRPC server
    pub fn new(unified_engine: Arc<UnifiedClinicalEngine>) -> Self {
        Self { unified_engine }
    }
}

#[tonic::async_trait]
impl Flow2Engine for Flow2GrpcServer {
    /// Execute a medication recipe with clinical context
    async fn execute_recipe(
        &self,
        request: Request<RecipeExecutionRequest>,
    ) -> Result<Response<RecipeExecutionResponse>, Status> {
        let start_time = std::time::Instant::now();
        let req = request.into_inner();

        info!("🦀 [gRPC] Executing recipe: {} for patient: {}", req.recipe_id, req.patient_id);

        // Convert gRPC request to internal clinical request
        let clinical_request = match convert_recipe_request_to_clinical_request(&req) {
            Ok(req) => req,
            Err(e) => {
                error!("🦀 [gRPC] Failed to convert recipe request: {}", e);
                return Err(Status::invalid_argument(format!("Invalid request format: {}", e)));
            }
        };

        // Process request through unified clinical engine
        let clinical_response = match self.unified_engine.process_clinical_request(clinical_request).await {
            Ok(response) => response,
            Err(e) => {
                error!("🦀 [gRPC] Unified engine processing failed for {}: {}", req.request_id, e);
                return Err(Status::internal(format!("Engine processing failed: {}", e)));
            }
        };

        // Convert internal response to gRPC response
        let response = convert_clinical_response_to_recipe_response(&clinical_response, start_time.elapsed());

        info!("🦀 [gRPC] Recipe execution completed: {} ({}ms)",
              req.request_id, response.execution_time_ms);

        Ok(Response::new(response))
    }

    /// Optimize medication dosing based on patient parameters
    async fn optimize_dose(
        &self,
        request: Request<DoseOptimizationRequest>,
    ) -> Result<Response<DoseOptimizationResponse>, Status> {
        let start_time = std::time::Instant::now();
        let req = request.into_inner();

        info!("🦀 [gRPC] Optimizing dose for medication: {} patient: {}",
              req.medication_code, req.patient_id);

        // Convert gRPC request to internal clinical request
        let clinical_request = match convert_dose_request_to_clinical_request(&req) {
            Ok(req) => req,
            Err(e) => {
                error!("🦀 [gRPC] Failed to convert dose optimization request: {}", e);
                return Err(Status::invalid_argument(format!("Invalid request format: {}", e)));
            }
        };

        // Process request through unified clinical engine
        let clinical_response = match self.unified_engine.process_clinical_request(clinical_request).await {
            Ok(response) => response,
            Err(e) => {
                error!("🦀 [gRPC] Dose optimization failed for {}: {}", req.request_id, e);
                return Err(Status::internal(format!("Engine processing failed: {}", e)));
            }
        };

        // Convert internal response to gRPC response
        let response = convert_clinical_response_to_dose_response(&clinical_response, start_time.elapsed());

        info!("🦀 [gRPC] Dose optimization completed: {} ({}mg)",
              req.request_id, response.recommendation.as_ref().unwrap().dose_value);

        Ok(Response::new(response))
    }

    /// Perform comprehensive medication intelligence analysis
    async fn analyze_medication(
        &self,
        request: Request<MedicationIntelligenceRequest>,
    ) -> Result<Response<MedicationIntelligenceResponse>, Status> {
        let start_time = std::time::Instant::now();
        let req = request.into_inner();

        info!("🦀 [gRPC] Analyzing medication intelligence for {} medications, patient: {}",
              req.medication_codes.len(), req.patient_id);

        // Convert gRPC request to internal clinical request
        let clinical_request = match convert_intelligence_request_to_clinical_request(&req) {
            Ok(req) => req,
            Err(e) => {
                error!("🦀 [gRPC] Failed to convert medication intelligence request: {}", e);
                return Err(Status::invalid_argument(format!("Invalid request format: {}", e)));
            }
        };

        // Process request through unified clinical engine
        let clinical_response = match self.unified_engine.process_clinical_request(clinical_request).await {
            Ok(response) => response,
            Err(e) => {
                error!("🦀 [gRPC] Medication intelligence failed for {}: {}", req.request_id, e);
                return Err(Status::internal(format!("Engine processing failed: {}", e)));
            }
        };

        // Convert internal response to gRPC response
        let response = convert_clinical_response_to_intelligence_response(&clinical_response, start_time.elapsed());

        info!("🦀 [gRPC] Medication intelligence completed: {}", req.request_id);

        Ok(Response::new(response))
    }

    /// Execute Flow2 workflow for complex clinical decisions
    async fn execute_flow2(
        &self,
        request: Request<Flow2Request>,
    ) -> Result<Response<Flow2Response>, Status> {
        let start_time = std::time::Instant::now();
        let req = request.into_inner();

        info!("🦀 [gRPC] Executing Flow2 workflow: {} for patient: {}",
              req.action_type, req.patient_id);

        // Convert gRPC request to internal clinical request
        let clinical_request = match convert_flow2_request_to_clinical_request(&req) {
            Ok(req) => req,
            Err(e) => {
                error!("🦀 [gRPC] Failed to convert Flow2 request: {}", e);
                return Err(Status::invalid_argument(format!("Invalid request format: {}", e)));
            }
        };

        // Process request through unified clinical engine
        let clinical_response = match self.unified_engine.process_clinical_request(clinical_request).await {
            Ok(response) => response,
            Err(e) => {
                error!("🦀 [gRPC] Flow2 execution failed for {}: {}", req.request_id, e);
                return Err(Status::internal(format!("Engine processing failed: {}", e)));
            }
        };

        // Convert internal response to gRPC response
        let response = convert_clinical_response_to_flow2_response(&clinical_response, start_time.elapsed());

        info!("🦀 [gRPC] Flow2 execution completed: {}", req.request_id);

        Ok(Response::new(response))
    }

    /// Health check for service availability
    async fn health_check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> Result<Response<HealthCheckResponse>, Status> {
        info!("🦀 [gRPC] Health check requested");

        let response = HealthCheckResponse {
            status: ServiceStatus::Healthy as i32,
            version: crate::VERSION.to_string(),
            capabilities: {
                let mut caps = HashMap::new();
                caps.insert("dose_calculation".to_string(), "available".to_string());
                caps.insert("safety_verification".to_string(), "available".to_string());
                caps.insert("clinical_intelligence".to_string(), "available".to_string());
                caps.insert("flow2_integration".to_string(), "available".to_string());
                caps
            },
            timestamp: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
        };

        Ok(Response::new(response))
    }

    /// Stream type for patient updates
    type StreamPatientUpdatesStream = Pin<Box<dyn Stream<Item = Result<ClinicalAlert, Status>> + Send>>;

    /// Stream patient updates for real-time monitoring
    async fn stream_patient_updates(
        &self,
        request: Request<Streaming<PatientUpdateRequest>>,
    ) -> Result<Response<Self::StreamPatientUpdatesStream>, Status> {
        let mut stream = request.into_inner();
        let (tx, rx) = mpsc::channel(100);

        info!("🦀 [gRPC] Starting patient updates stream");

        // Spawn task to handle incoming patient updates
        tokio::spawn(async move {
            while let Some(update_result) = stream.message().await.transpose() {
                match update_result {
                    Ok(update) => {
                        info!("🦀 [gRPC] Received patient update for: {}", update.patient_id);

                        // Process update and generate clinical alert if needed
                        let alert = ClinicalAlert {
                            alert_id: uuid::Uuid::new_v4().to_string(),
                            patient_id: update.patient_id.clone(),
                            timestamp: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
                            severity: AlertSeverity::Info as i32,
                            message: format!("Patient update processed for {}", update.patient_id),
                            actions: vec!["Monitor patient status".to_string()],
                        };

                        if let Err(_) = tx.send(Ok(alert)).await {
                            warn!("🦀 [gRPC] Client disconnected from patient updates stream");
                            break;
                        }
                    }
                    Err(e) => {
                        error!("🦀 [gRPC] Error in patient update stream: {}", e);
                        if let Err(_) = tx.send(Err(Status::internal("Stream error"))).await {
                            break;
                        }
                    }
                }
            }
            info!("🦀 [gRPC] Patient updates stream ended");
        });

        let output_stream = ReceiverStream::new(rx);
        Ok(Response::new(Box::pin(output_stream) as Self::StreamPatientUpdatesStream))
    }
}

// ============================================================================
// CONVERSION FUNCTIONS
// ============================================================================

/// Convert gRPC RecipeExecutionRequest to internal ClinicalRequest
fn convert_recipe_request_to_clinical_request(
    request: &RecipeExecutionRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    let patient_context = extract_patient_context_from_struct(&request.clinical_context)?;

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: request.medication_code.clone(),
        indication: request.recipe_id.clone(),
        patient_context,
        timestamp: Utc::now(),
    })
}

/// Convert gRPC DoseOptimizationRequest to internal ClinicalRequest
fn convert_dose_request_to_clinical_request(
    request: &DoseOptimizationRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    let patient_context = if let Some(ctx) = &request.patient_context {
        convert_protobuf_patient_context(ctx)?
    } else {
        return Err("Patient context is required".to_string());
    };

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: request.medication_code.clone(),
        indication: format!("dose_optimization_{:?}", request.purpose),
        patient_context,
        timestamp: Utc::now(),
    })
}

/// Convert gRPC MedicationIntelligenceRequest to internal ClinicalRequest
fn convert_intelligence_request_to_clinical_request(
    request: &MedicationIntelligenceRequest,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    let medication_code = request.medication_codes.first()
        .ok_or("At least one medication code is required")?
        .clone();

    let patient_context = if let Some(ctx) = &request.patient_context {
        convert_protobuf_patient_context(ctx)?
    } else {
        return Err("Patient context is required".to_string());
    };

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: medication_code,
        indication: format!("intelligence_{:?}", request.intelligence_type),
        patient_context,
        timestamp: Utc::now(),
    })
}

/// Convert gRPC Flow2Request to internal ClinicalRequest
fn convert_flow2_request_to_clinical_request(
    request: &Flow2Request,
) -> Result<crate::unified_clinical_engine::ClinicalRequest, String> {
    let patient_context = extract_patient_context_from_struct(&request.clinical_data)?;

    Ok(crate::unified_clinical_engine::ClinicalRequest {
        request_id: request.request_id.clone(),
        drug_id: "unknown".to_string(), // Will be extracted from parameters
        indication: request.action_type.clone(),
        patient_context,
        timestamp: Utc::now(),
    })
}

/// Extract PatientContext from protobuf Struct (simplified)
fn extract_patient_context_from_struct(
    _clinical_context: &Option<prost_types::Struct>,
) -> Result<crate::unified_clinical_engine::PatientContext, String> {
    use crate::unified_clinical_engine::{PatientContext, BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction};

    // Return default patient context for now
    Ok(PatientContext {
        age_years: 45.0,
        weight_kg: 70.0,
        height_cm: 170.0,
        sex: BiologicalSex::Male,
        pregnancy_status: PregnancyStatus::NotPregnant,
        renal_function: RenalFunction {
            egfr_ml_min: None,
            egfr_ml_min_1_73m2: None,
            creatinine_clearance: None,
            creatinine_mg_dl: None,
            bun_mg_dl: None,
            stage: None,
        },
        hepatic_function: HepaticFunction {
            child_pugh_class: None,
            alt_u_l: None,
            ast_u_l: None,
            bilirubin_mg_dl: None,
            albumin_g_dl: None,
        },
        active_medications: vec![],
        allergies: vec![],
        conditions: vec![],
        lab_values: HashMap::new(),
    })
}

/// Convert protobuf PatientContext to internal PatientContext (simplified)
fn convert_protobuf_patient_context(
    proto_context: &PatientContext,
) -> Result<crate::unified_clinical_engine::PatientContext, String> {
    use crate::unified_clinical_engine::{
        PatientContext as InternalPatientContext,
        BiologicalSex,
        PregnancyStatus,
        RenalFunction,
        HepaticFunction
    };

    let sex = match proto_context.gender.to_lowercase().as_str() {
        "female" => BiologicalSex::Female,
        "other" => BiologicalSex::Other,
        _ => BiologicalSex::Male,
    };

    let pregnancy_status = match proto_context.pregnancy_status.to_lowercase().as_str() {
        "pregnant" => PregnancyStatus::Pregnant { trimester: 1 },
        "unknown" => PregnancyStatus::Unknown,
        _ => PregnancyStatus::NotPregnant,
    };

    Ok(InternalPatientContext {
        age_years: proto_context.age_years,
        weight_kg: proto_context.weight_kg,
        height_cm: proto_context.height_cm,
        sex,
        pregnancy_status,
        renal_function: RenalFunction {
            egfr_ml_min: Some(proto_context.egfr),
            egfr_ml_min_1_73m2: Some(proto_context.egfr),
            creatinine_clearance: Some(proto_context.creatinine_clearance),
            creatinine_mg_dl: None,
            bun_mg_dl: None,
            stage: Some(proto_context.hepatic_function.clone()),
        },
        hepatic_function: HepaticFunction {
            child_pugh_class: Some(proto_context.hepatic_function.clone()),
            alt_u_l: None,
            ast_u_l: None,
            bilirubin_mg_dl: None,
            albumin_g_dl: None,
        },
        active_medications: vec![], // Simplified - no conversion for now
        allergies: vec![], // Simplified - no conversion for now
        conditions: proto_context.active_conditions.clone(),
        lab_values: HashMap::new(), // Simplified - no conversion for now
    })
}

/// Convert internal ClinicalResponse to gRPC RecipeExecutionResponse
fn convert_clinical_response_to_recipe_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
    execution_time: std::time::Duration,
) -> RecipeExecutionResponse {
    let success = matches!(
        clinical_response.final_recommendation.action,
        crate::unified_clinical_engine::RecommendationAction::Prescribe |
        crate::unified_clinical_engine::RecommendationAction::PrescribeWithMonitoring
    );

    RecipeExecutionResponse {
        request_id: clinical_response.request_id.clone(),
        success,
        result: Some(RecipeResult {
            recipe_id: "unified_clinical_engine".to_string(),
            recipe_name: "Unified Clinical Analysis".to_string(),
            safety_status: if success { SafetyStatus::Safe } else { SafetyStatus::Caution } as i32,
            alerts: clinical_response.safety_result.findings.iter().map(|finding| SafetyAlert {
                alert_id: uuid::Uuid::new_v4().to_string(),
                severity: match finding.severity.as_str() {
                    "critical" => AlertSeverity::Critical,
                    "high" => AlertSeverity::High,
                    "medium" => AlertSeverity::Medium,
                    "low" => AlertSeverity::Low,
                    _ => AlertSeverity::Info,
                } as i32,
                category: finding.category.clone(),
                message: finding.message.clone(),
                clinical_significance: finding.message.clone(),
                recommended_actions: vec!["Monitor patient".to_string()],
                evidence_references: vec![],
            }).collect(),
            recommendations: vec![ClinicalRecommendation {
                recommendation_id: uuid::Uuid::new_v4().to_string(),
                r#type: "dose_recommendation".to_string(),
                description: format!("Recommended dose: {}mg", clinical_response.final_recommendation.final_dose_mg),
                priority: Priority::Medium as i32,
                rationale: "Clinical analysis recommendation".to_string(), // Fixed field access
                action_items: clinical_response.final_recommendation.monitoring_required.clone(),
            }],
            clinical_evidence: None,
        }),
        errors: if success { vec![] } else { vec!["Clinical analysis identified safety concerns".to_string()] },
        timestamp: Some(prost_types::Timestamp::from(std::time::SystemTime::now())),
        execution_time_ms: execution_time.as_millis() as i64,
    }
}

/// Convert internal ClinicalResponse to gRPC DoseOptimizationResponse
fn convert_clinical_response_to_dose_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
    _execution_time: std::time::Duration,
) -> DoseOptimizationResponse {
    let success = matches!(
        clinical_response.final_recommendation.action,
        crate::unified_clinical_engine::RecommendationAction::Prescribe |
        crate::unified_clinical_engine::RecommendationAction::PrescribeWithMonitoring
    );

    DoseOptimizationResponse {
        request_id: clinical_response.request_id.clone(),
        success,
        recommendation: Some(DoseRecommendation {
            dose_value: clinical_response.final_recommendation.final_dose_mg,
            dose_unit: "mg".to_string(),
            route: "oral".to_string(),
            frequency: "BID".to_string(),
            duration_days: 7,
            calculation_method: clinical_response.calculation_result.calculation_strategy.clone(),
            calculation_factors: HashMap::new(),
            adjustments: vec![],
        }),
        safety_alerts: clinical_response.safety_result.findings.iter().map(|finding| SafetyAlert {
            alert_id: uuid::Uuid::new_v4().to_string(),
            severity: AlertSeverity::Medium as i32,
            category: finding.category.clone(),
            message: finding.message.clone(),
            clinical_significance: finding.message.clone(),
            recommended_actions: vec!["Monitor patient".to_string()],
            evidence_references: vec![],
        }).collect(),
        errors: if success { vec![] } else { vec!["Dose optimization failed".to_string()] },
    }
}

/// Convert internal ClinicalResponse to gRPC MedicationIntelligenceResponse
fn convert_clinical_response_to_intelligence_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
    _execution_time: std::time::Duration,
) -> MedicationIntelligenceResponse {
    let success = matches!(
        clinical_response.final_recommendation.action,
        crate::unified_clinical_engine::RecommendationAction::Prescribe |
        crate::unified_clinical_engine::RecommendationAction::PrescribeWithMonitoring
    );

    MedicationIntelligenceResponse {
        request_id: clinical_response.request_id.clone(),
        success,
        analysis: Some(MedicationAnalysis {
            medication_code: "analyzed_medication".to_string(),
            medication_name: "Clinical Analysis Result".to_string(),
            safety_considerations: clinical_response.safety_result.findings.iter().map(|finding| SafetyConsideration {
                r#type: finding.category.clone(),
                description: finding.message.clone(),
                risk_level: RiskLevel::Moderate as i32,
                mitigation_strategies: vec!["Monitor patient".to_string()],
            }).collect(),
            pk_profile: Some(PharmacokineticProfile {
                half_life_hours: 8.0,
                clearance: 5.0,
                volume_distribution: 50.0,
                metabolism_pathway: "hepatic".to_string(),
                cyp_interactions: vec!["CYP3A4".to_string()],
                bioavailability: 0.8,
            }),
            guidelines: vec![],
            monitoring: vec![],
        }),
        interactions: vec![],
        alerts: clinical_response.safety_result.findings.iter().map(|finding| SafetyAlert {
            alert_id: uuid::Uuid::new_v4().to_string(),
            severity: AlertSeverity::Medium as i32,
            category: finding.category.clone(),
            message: finding.message.clone(),
            clinical_significance: finding.message.clone(),
            recommended_actions: vec!["Monitor patient".to_string()],
            evidence_references: vec![],
        }).collect(),
        errors: if success { vec![] } else { vec!["Intelligence analysis failed".to_string()] },
    }
}

/// Convert internal ClinicalResponse to gRPC Flow2Response
fn convert_clinical_response_to_flow2_response(
    clinical_response: &crate::unified_clinical_engine::ClinicalResponse,
    _execution_time: std::time::Duration,
) -> Flow2Response {
    let success = matches!(
        clinical_response.final_recommendation.action,
        crate::unified_clinical_engine::RecommendationAction::Prescribe |
        crate::unified_clinical_engine::RecommendationAction::PrescribeWithMonitoring
    );

    Flow2Response {
        request_id: clinical_response.request_id.clone(),
        success,
        data: None, // TODO: Add proper data serialization
        errors: if success { vec![] } else { vec!["Flow2 execution failed".to_string()] },
        metadata: {
            let mut meta = HashMap::new();
            meta.insert("engine".to_string(), "unified_clinical_engine".to_string());
            meta.insert("dose_mg".to_string(), clinical_response.final_recommendation.final_dose_mg.to_string());
            meta
        },
    }
}