//! Clinical Assessment Engine (CAE) Integration
//!
//! This module provides integration with the Clinical Assessment Engine for
//! coordinated safety evaluation and clinical decision-making. It bridges
//! the Protocol Engine with external clinical reasoning services through
//! async messaging and event-driven communication.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc};
use uuid::Uuid;

use crate::protocol::{
    types::*,
    error::*,
    state_machine::{ProtocolStateMachine, StateTransition},
    temporal::{TemporalConstraint, TimeWindow},
};

/// Clinical Assessment Engine integration configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeIntegrationConfig {
    /// CAE service endpoint URL
    pub cae_endpoint: String,
    /// Request timeout in milliseconds
    pub request_timeout_ms: u64,
    /// Maximum concurrent CAE requests
    pub max_concurrent_requests: usize,
    /// Enable CAE integration
    pub enabled: bool,
    /// CAE authentication configuration
    pub auth_config: CaeAuthConfig,
    /// Event publishing configuration
    pub event_config: CaeEventConfig,
}

/// CAE authentication configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeAuthConfig {
    /// Service-to-service authentication token
    pub service_token: String,
    /// CAE API key
    pub api_key: String,
    /// Authentication method
    pub auth_method: CaeAuthMethod,
}

/// CAE authentication methods
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CaeAuthMethod {
    /// Bearer token authentication
    Bearer,
    /// API key authentication
    ApiKey,
    /// Mutual TLS authentication
    MutualTLS,
    /// Service mesh authentication
    ServiceMesh,
}

/// CAE event publishing configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeEventConfig {
    /// Enable event publishing to CAE
    pub publish_events: bool,
    /// Event buffer size
    pub event_buffer_size: usize,
    /// Event batch size for publishing
    pub batch_size: usize,
    /// Event publishing interval in milliseconds
    pub publish_interval_ms: u64,
}

/// Clinical Assessment Engine request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeEvaluationRequest {
    /// Unique request ID
    pub request_id: String,
    /// Patient identifier
    pub patient_id: String,
    /// Protocol being evaluated
    pub protocol_id: String,
    /// Clinical context and data
    pub clinical_context: CaeClinicalContext,
    /// Current protocol state
    pub protocol_state: Option<String>,
    /// Temporal constraints to evaluate
    pub temporal_constraints: Vec<TemporalConstraint>,
    /// Request priority level
    pub priority: CaeRequestPriority,
    /// Request timestamp
    pub timestamp: DateTime<Utc>,
}

/// CAE clinical context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeClinicalContext {
    /// Patient demographics
    pub demographics: PatientDemographics,
    /// Current vital signs
    pub vital_signs: VitalSigns,
    /// Laboratory results
    pub lab_results: Vec<LabResult>,
    /// Current medications
    pub medications: Vec<Medication>,
    /// Medical history
    pub medical_history: Vec<MedicalCondition>,
    /// Current location/department
    pub location: ClinicalLocation,
}

/// Patient demographics for CAE evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientDemographics {
    pub age: u32,
    pub gender: String,
    pub weight_kg: Option<f64>,
    pub height_cm: Option<f64>,
    pub allergies: Vec<String>,
    pub emergency_contact: Option<String>,
}

/// Vital signs data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VitalSigns {
    pub temperature_celsius: Option<f64>,
    pub heart_rate_bpm: Option<u32>,
    pub blood_pressure_systolic: Option<u32>,
    pub blood_pressure_diastolic: Option<u32>,
    pub respiratory_rate: Option<u32>,
    pub oxygen_saturation: Option<f64>,
    pub timestamp: DateTime<Utc>,
}

/// Laboratory result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabResult {
    pub test_name: String,
    pub value: String,
    pub unit: Option<String>,
    pub reference_range: Option<String>,
    pub abnormal_flag: Option<String>,
    pub timestamp: DateTime<Utc>,
}

/// Medication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Medication {
    pub name: String,
    pub dosage: String,
    pub route: String,
    pub frequency: String,
    pub start_date: DateTime<Utc>,
    pub end_date: Option<DateTime<Utc>>,
    pub prescriber: String,
}

/// Medical condition/diagnosis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicalCondition {
    pub condition_name: String,
    pub icd_10_code: Option<String>,
    pub severity: Option<String>,
    pub onset_date: Option<DateTime<Utc>>,
    pub status: ConditionStatus,
}

/// Medical condition status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConditionStatus {
    Active,
    Inactive,
    Resolved,
    Chronic,
    Acute,
}

/// Clinical location information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalLocation {
    pub department: String,
    pub unit: Option<String>,
    pub room: Option<String>,
    pub bed: Option<String>,
    pub facility_id: String,
}

/// CAE request priority levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CaeRequestPriority {
    /// Emergency - immediate response required
    Emergency,
    /// Urgent - response within minutes
    Urgent,
    /// High - response within 15 minutes
    High,
    /// Normal - response within 30 minutes
    Normal,
    /// Low - response within 1 hour
    Low,
}

/// Clinical Assessment Engine response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeEvaluationResponse {
    /// Request ID this response corresponds to
    pub request_id: String,
    /// Overall assessment result
    pub assessment_result: CaeAssessmentResult,
    /// Clinical recommendations
    pub recommendations: Vec<ClinicalRecommendation>,
    /// Risk assessment
    pub risk_assessment: RiskAssessment,
    /// Protocol modifications suggested
    pub protocol_modifications: Vec<ProtocolModification>,
    /// Next evaluation time
    pub next_evaluation: Option<DateTime<Utc>>,
    /// Response timestamp
    pub timestamp: DateTime<Utc>,
}

/// CAE assessment result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CaeAssessmentResult {
    /// Continue with current protocol
    Continue,
    /// Modify current protocol
    Modify,
    /// Escalate to higher level of care
    Escalate,
    /// Complete current protocol
    Complete,
    /// Hold/pause current protocol
    Hold,
    /// Cancel current protocol
    Cancel,
}

/// Clinical recommendation from CAE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRecommendation {
    /// Recommendation identifier
    pub recommendation_id: String,
    /// Recommendation text
    pub text: String,
    /// Priority level
    pub priority: RecommendationPriority,
    /// Category of recommendation
    pub category: RecommendationCategory,
    /// Evidence level
    pub evidence_level: EvidenceLevel,
    /// Expiration time for recommendation
    pub expires_at: Option<DateTime<Utc>>,
}

/// Recommendation priority levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationPriority {
    Critical,
    High,
    Medium,
    Low,
    Informational,
}

/// Recommendation categories
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationCategory {
    Medication,
    Monitoring,
    Diagnostic,
    Intervention,
    Consultation,
    Discharge,
    Safety,
}

/// Evidence levels for recommendations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceLevel {
    /// Systematic review/meta-analysis
    Level1,
    /// Randomized controlled trial
    Level2,
    /// Cohort study
    Level3,
    /// Case-control study
    Level4,
    /// Case series/expert opinion
    Level5,
}

/// Risk assessment from CAE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    /// Overall risk score (0.0 to 1.0)
    pub overall_risk_score: f64,
    /// Individual risk factors
    pub risk_factors: Vec<RiskFactor>,
    /// Risk mitigation strategies
    pub mitigation_strategies: Vec<String>,
    /// Risk trend over time
    pub risk_trend: RiskTrend,
}

/// Individual risk factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    pub factor_name: String,
    pub severity: f64, // 0.0 to 1.0
    pub description: String,
    pub modifiable: bool,
}

/// Risk trend indicators
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskTrend {
    Improving,
    Stable,
    Worsening,
    Unknown,
}

/// Protocol modification suggested by CAE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolModification {
    /// Type of modification
    pub modification_type: ModificationType,
    /// Target element to modify
    pub target: String,
    /// New value or configuration
    pub new_value: serde_json::Value,
    /// Justification for modification
    pub justification: String,
    /// Approval required
    pub requires_approval: bool,
}

/// Types of protocol modifications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ModificationType {
    /// Modify temporal constraint
    TemporalConstraint,
    /// Add new constraint
    AddConstraint,
    /// Remove constraint
    RemoveConstraint,
    /// Modify state transition
    StateTransition,
    /// Change monitoring frequency
    MonitoringFrequency,
    /// Update target parameters
    TargetParameters,
}

/// CAE Integration Engine
pub struct CaeIntegrationEngine {
    /// Configuration
    config: CaeIntegrationConfig,
    /// HTTP client for CAE requests
    client: reqwest::Client,
    /// Active requests tracking
    active_requests: Arc<RwLock<HashMap<String, DateTime<Utc>>>>,
    /// Event publishing channel
    event_sender: Option<mpsc::UnboundedSender<CaeEvent>>,
    /// Request metrics
    metrics: CaeMetrics,
}

/// CAE event for publishing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeEvent {
    /// Event identifier
    pub event_id: String,
    /// Event type
    pub event_type: CaeEventType,
    /// Patient identifier
    pub patient_id: String,
    /// Protocol identifier
    pub protocol_id: String,
    /// Event data
    pub data: serde_json::Value,
    /// Event timestamp
    pub timestamp: DateTime<Utc>,
}

/// CAE event types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CaeEventType {
    /// Protocol evaluation initiated
    ProtocolEvaluationStarted,
    /// Protocol evaluation completed
    ProtocolEvaluationCompleted,
    /// CAE assessment completed
    AssessmentCompleted,
    /// Risk level changed
    RiskLevelChanged,
    /// Recommendation generated
    RecommendationGenerated,
    /// Protocol modified
    ProtocolModified,
    /// Clinical alert triggered
    ClinicalAlert,
}

/// CAE integration metrics
#[derive(Debug, Default)]
pub struct CaeMetrics {
    /// Total CAE requests made
    pub total_requests: std::sync::atomic::AtomicU64,
    /// Successful responses
    pub successful_responses: std::sync::atomic::AtomicU64,
    /// Failed requests
    pub failed_requests: std::sync::atomic::AtomicU64,
    /// Average response time
    pub average_response_time_ms: std::sync::atomic::AtomicU64,
    /// Active request count
    pub active_requests: std::sync::atomic::AtomicU64,
}

impl CaeIntegrationEngine {
    /// Create new CAE integration engine
    pub fn new(config: CaeIntegrationConfig) -> ProtocolResult<Self> {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_millis(config.request_timeout_ms))
            .build()
            .map_err(|e| ProtocolEngineError::IntegrationError(e.to_string()))?;

        Ok(Self {
            config,
            client,
            active_requests: Arc::new(RwLock::new(HashMap::new())),
            event_sender: None,
            metrics: CaeMetrics::default(),
        })
    }

    /// Initialize event publishing channel
    pub fn initialize_event_publishing(&mut self) -> mpsc::UnboundedReceiver<CaeEvent> {
        let (sender, receiver) = mpsc::unbounded_channel();
        self.event_sender = Some(sender);
        receiver
    }

    /// Evaluate protocol with CAE integration
    pub async fn evaluate_with_cae(
        &self,
        protocol_request: &ProtocolEvaluationRequest,
        clinical_context: CaeClinicalContext,
    ) -> ProtocolResult<CaeEvaluationResponse> {
        if !self.config.enabled {
            return Err(ProtocolEngineError::IntegrationError(
                "CAE integration is disabled".to_string()
            ));
        }

        let request_id = Uuid::new_v4().to_string();
        
        // Track request start
        {
            let mut active = self.active_requests.write().await;
            active.insert(request_id.clone(), Utc::now());
        }

        self.metrics.total_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        self.metrics.active_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);

        let cae_request = CaeEvaluationRequest {
            request_id: request_id.clone(),
            patient_id: protocol_request.patient_id.clone(),
            protocol_id: protocol_request.protocol_id.clone(),
            clinical_context,
            protocol_state: None, // TODO: Extract from protocol request
            temporal_constraints: vec![], // TODO: Extract from protocol request
            priority: CaeRequestPriority::Normal,
            timestamp: Utc::now(),
        };

        // Publish event
        if let Some(sender) = &self.event_sender {
            let event = CaeEvent {
                event_id: Uuid::new_v4().to_string(),
                event_type: CaeEventType::ProtocolEvaluationStarted,
                patient_id: protocol_request.patient_id.clone(),
                protocol_id: protocol_request.protocol_id.clone(),
                data: serde_json::to_value(&cae_request).unwrap_or_default(),
                timestamp: Utc::now(),
            };
            let _ = sender.send(event);
        }

        let response = self.send_cae_request(&cae_request).await;

        // Clean up tracking
        {
            let mut active = self.active_requests.write().await;
            active.remove(&request_id);
        }
        self.metrics.active_requests.fetch_sub(1, std::sync::atomic::Ordering::Relaxed);

        match response {
            Ok(cae_response) => {
                self.metrics.successful_responses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                
                // Publish completion event
                if let Some(sender) = &self.event_sender {
                    let event = CaeEvent {
                        event_id: Uuid::new_v4().to_string(),
                        event_type: CaeEventType::AssessmentCompleted,
                        patient_id: protocol_request.patient_id.clone(),
                        protocol_id: protocol_request.protocol_id.clone(),
                        data: serde_json::to_value(&cae_response).unwrap_or_default(),
                        timestamp: Utc::now(),
                    };
                    let _ = sender.send(event);
                }

                Ok(cae_response)
            },
            Err(e) => {
                self.metrics.failed_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Err(e)
            }
        }
    }

    /// Send request to CAE service
    async fn send_cae_request(
        &self,
        request: &CaeEvaluationRequest,
    ) -> ProtocolResult<CaeEvaluationResponse> {
        let start_time = std::time::Instant::now();

        let mut req_builder = self.client
            .post(&format!("{}/evaluate", self.config.cae_endpoint))
            .json(request);

        // Add authentication
        match &self.config.auth_config.auth_method {
            CaeAuthMethod::Bearer => {
                req_builder = req_builder.bearer_auth(&self.config.auth_config.service_token);
            },
            CaeAuthMethod::ApiKey => {
                req_builder = req_builder.header("X-API-Key", &self.config.auth_config.api_key);
            },
            CaeAuthMethod::MutualTLS => {
                // TODO: Implement mutual TLS configuration
            },
            CaeAuthMethod::ServiceMesh => {
                // Authentication handled by service mesh
            },
        }

        let response = req_builder
            .send()
            .await
            .map_err(|e| ProtocolEngineError::IntegrationError(format!("CAE request failed: {}", e)))?;

        let response_time = start_time.elapsed().as_millis() as u64;
        self.update_response_time(response_time);

        if !response.status().is_success() {
            return Err(ProtocolEngineError::IntegrationError(
                format!("CAE responded with status: {}", response.status())
            ));
        }

        let cae_response: CaeEvaluationResponse = response
            .json()
            .await
            .map_err(|e| ProtocolEngineError::IntegrationError(
                format!("Failed to parse CAE response: {}", e)
            ))?;

        Ok(cae_response)
    }

    /// Update average response time metric
    fn update_response_time(&self, response_time_ms: u64) {
        let current_avg = self.metrics.average_response_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_requests = self.metrics.total_requests.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_requests > 1 {
            let new_avg = (current_avg * (total_requests - 1) + response_time_ms) / total_requests;
            self.metrics.average_response_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.metrics.average_response_time_ms.store(response_time_ms, std::sync::atomic::Ordering::Relaxed);
        }
    }

    /// Get current metrics snapshot
    pub fn get_metrics(&self) -> CaeMetricsSnapshot {
        CaeMetricsSnapshot {
            total_requests: self.metrics.total_requests.load(std::sync::atomic::Ordering::Relaxed),
            successful_responses: self.metrics.successful_responses.load(std::sync::atomic::Ordering::Relaxed),
            failed_requests: self.metrics.failed_requests.load(std::sync::atomic::Ordering::Relaxed),
            average_response_time_ms: self.metrics.average_response_time_ms.load(std::sync::atomic::Ordering::Relaxed),
            active_requests: self.metrics.active_requests.load(std::sync::atomic::Ordering::Relaxed),
        }
    }

    /// Check health of CAE integration
    pub async fn health_check(&self) -> ProtocolResult<CaeHealthStatus> {
        if !self.config.enabled {
            return Ok(CaeHealthStatus::Disabled);
        }

        let health_endpoint = format!("{}/health", self.config.cae_endpoint);
        
        match self.client.get(&health_endpoint).send().await {
            Ok(response) => {
                if response.status().is_success() {
                    Ok(CaeHealthStatus::Healthy)
                } else {
                    Ok(CaeHealthStatus::Unhealthy(format!("Status: {}", response.status())))
                }
            },
            Err(e) => Ok(CaeHealthStatus::Unhealthy(e.to_string())),
        }
    }
}

/// CAE metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaeMetricsSnapshot {
    pub total_requests: u64,
    pub successful_responses: u64,
    pub failed_requests: u64,
    pub average_response_time_ms: u64,
    pub active_requests: u64,
}

/// CAE health status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CaeHealthStatus {
    Healthy,
    Unhealthy(String),
    Disabled,
}

impl Default for CaeIntegrationConfig {
    fn default() -> Self {
        Self {
            cae_endpoint: "http://localhost:8020".to_string(),
            request_timeout_ms: 30000,
            max_concurrent_requests: 100,
            enabled: true,
            auth_config: CaeAuthConfig {
                service_token: "".to_string(),
                api_key: "".to_string(),
                auth_method: CaeAuthMethod::Bearer,
            },
            event_config: CaeEventConfig {
                publish_events: true,
                event_buffer_size: 1000,
                batch_size: 10,
                publish_interval_ms: 1000,
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_cae_integration_engine_creation() {
        let config = CaeIntegrationConfig::default();
        let engine = CaeIntegrationEngine::new(config);
        assert!(engine.is_ok());
    }

    #[tokio::test]
    async fn test_event_publishing_initialization() {
        let config = CaeIntegrationConfig::default();
        let mut engine = CaeIntegrationEngine::new(config).unwrap();
        let receiver = engine.initialize_event_publishing();
        assert!(engine.event_sender.is_some());
    }

    #[test]
    fn test_cae_request_serialization() {
        let request = CaeEvaluationRequest {
            request_id: "test-123".to_string(),
            patient_id: "patient-456".to_string(),
            protocol_id: "sepsis-bundle-v1".to_string(),
            clinical_context: CaeClinicalContext {
                demographics: PatientDemographics {
                    age: 65,
                    gender: "M".to_string(),
                    weight_kg: Some(80.0),
                    height_cm: Some(175.0),
                    allergies: vec!["penicillin".to_string()],
                    emergency_contact: Some("spouse".to_string()),
                },
                vital_signs: VitalSigns {
                    temperature_celsius: Some(38.5),
                    heart_rate_bpm: Some(110),
                    blood_pressure_systolic: Some(90),
                    blood_pressure_diastolic: Some(60),
                    respiratory_rate: Some(22),
                    oxygen_saturation: Some(95.0),
                    timestamp: Utc::now(),
                },
                lab_results: vec![],
                medications: vec![],
                medical_history: vec![],
                location: ClinicalLocation {
                    department: "Emergency".to_string(),
                    unit: Some("ED".to_string()),
                    room: Some("ED-12".to_string()),
                    bed: Some("A".to_string()),
                    facility_id: "HOSP-001".to_string(),
                },
            },
            protocol_state: None,
            temporal_constraints: vec![],
            priority: CaeRequestPriority::High,
            timestamp: Utc::now(),
        };

        let serialized = serde_json::to_string(&request);
        assert!(serialized.is_ok());
        
        let deserialized: Result<CaeEvaluationRequest, _> = serde_json::from_str(&serialized.unwrap());
        assert!(deserialized.is_ok());
    }
}