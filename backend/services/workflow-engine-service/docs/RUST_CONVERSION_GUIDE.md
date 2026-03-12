# Workflow Engine Service - Rust Conversion Guide

## Overview

This guide provides comprehensive documentation for converting the Python-based workflow engine service to Rust. The conversion leverages Rust's memory safety, zero-cost abstractions, and fearless concurrency to deliver enterprise-grade performance for clinical workflow orchestration systems.

## Architecture Analysis

### Current Python Structure
```
app/
├── core/              # Configuration and settings
├── db/                # Database models and connections
├── models/            # SQLAlchemy models  
├── orchestration/     # Strategic orchestration layer
├── clinical/          # Clinical workflow implementations
├── gql_schema/        # GraphQL Federation schemas
├── security/          # Authentication middleware
├── monitoring/        # Observability components
└── main.py           # FastAPI application
```

### Proposed Rust Structure
```
src/
├── main.rs           # Application entry point
├── lib.rs            # Library crate root
├── config/           # Configuration management
│   ├── mod.rs
│   └── settings.rs
├── domain/           # Business domain models
│   ├── mod.rs
│   ├── snapshot.rs
│   ├── workflow.rs
│   └── orchestration.rs
├── infrastructure/   # External integrations
│   ├── mod.rs
│   ├── database/
│   │   ├── mod.rs
│   │   ├── models.rs
│   │   └── connection.rs
│   ├── clients/
│   │   ├── mod.rs
│   │   ├── flow2.rs
│   │   ├── safety_gateway.rs
│   │   └── medication_service.rs
│   └── monitoring/
│       ├── mod.rs
│       ├── metrics.rs
│       └── tracing.rs
├── application/      # Application services
│   ├── mod.rs
│   ├── orchestration/
│   │   ├── mod.rs
│   │   └── strategic_orchestrator.rs
│   ├── snapshot/
│   │   ├── mod.rs
│   │   └── service.rs
│   └── workflow/
│       ├── mod.rs
│       └── engine.rs
├── presentation/     # API layer
│   ├── mod.rs
│   ├── rest/
│   │   ├── mod.rs
│   │   ├── handlers.rs
│   │   └── middleware.rs
│   ├── graphql/
│   │   ├── mod.rs
│   │   ├── schema.rs
│   │   └── resolvers.rs
│   └── grpc/
│       ├── mod.rs
│       └── server.rs
└── utils/            # Utilities and helpers
    ├── mod.rs
    ├── error.rs
    └── validation.rs

Cargo.toml            # Dependencies and metadata
migrations/           # Database migrations (via sqlx)
proto/               # Protocol buffer definitions
configs/             # Configuration files
```

## Core Components Conversion

### 1. Configuration Management

**Python Implementation:**
```python
# app/core/config.py
class Settings(BaseSettings):
    SERVICE_NAME: str = "workflow-engine-service"
    SERVICE_PORT: int = 8017
    DATABASE_URL: str
```

**Rust Implementation:**
```rust
// src/config/settings.rs
use config::{Config, ConfigError, Environment, File};
use serde::{Deserialize, Serialize};
use std::time::Duration;

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct DatabaseConfig {
    pub url: String,
    pub max_connections: u32,
    pub min_connections: u32,
    pub acquire_timeout: Duration,
    pub idle_timeout: Duration,
    pub max_lifetime: Duration,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct GoogleCloudConfig {
    pub project_id: String,
    pub location: String,
    pub dataset: String,
    pub fhir_store: String,
    pub credentials_path: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ExternalServicesConfig {
    pub auth_service_url: String,
    pub medication_service_url: String,
    pub safety_gateway_url: String,
    pub flow2_go_url: String,
    pub flow2_rust_url: String,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct WorkflowConfig {
    pub execution_timeout: Duration,
    pub task_assignment_timeout: Duration,
    pub event_polling_interval: Duration,
    pub task_polling_interval: Duration,
    pub mock_mode: bool,
    pub enable_webhooks: bool,
    pub enable_fhir_monitoring: bool,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct PerformanceConfig {
    pub calculate_target_ms: u64,
    pub validate_target_ms: u64,
    pub commit_target_ms: u64,
    pub total_target_ms: u64,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct MonitoringConfig {
    pub prometheus_enabled: bool,
    pub jaeger_endpoint: String,
    pub metrics_port: u16,
    pub health_check_interval: Duration,
}

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct Settings {
    pub service_name: String,
    pub service_port: u16,
    pub debug: bool,
    pub log_level: String,
    
    pub database: DatabaseConfig,
    pub google_cloud: GoogleCloudConfig,
    pub external_services: ExternalServicesConfig,
    pub workflow: WorkflowConfig,
    pub performance: PerformanceConfig,
    pub monitoring: MonitoringConfig,
}

impl Settings {
    pub fn new() -> Result<Self, ConfigError> {
        let mut config = Config::builder()
            // Start with default values
            .set_default("service_name", "workflow-engine-service")?
            .set_default("service_port", 8017)?
            .set_default("debug", true)?
            .set_default("log_level", "info")?
            
            // Database defaults
            .set_default("database.max_connections", 10)?
            .set_default("database.min_connections", 5)?
            .set_default("database.acquire_timeout", "30s")?
            .set_default("database.idle_timeout", "600s")?
            .set_default("database.max_lifetime", "1800s")?
            
            // Workflow defaults
            .set_default("workflow.execution_timeout", "3600s")?
            .set_default("workflow.task_assignment_timeout", "86400s")?
            .set_default("workflow.event_polling_interval", "30s")?
            .set_default("workflow.task_polling_interval", "10s")?
            .set_default("workflow.mock_mode", false)?
            .set_default("workflow.enable_webhooks", true)?
            .set_default("workflow.enable_fhir_monitoring", true)?
            
            // Performance targets from documentation
            .set_default("performance.calculate_target_ms", 175)?
            .set_default("performance.validate_target_ms", 100)?
            .set_default("performance.commit_target_ms", 50)?
            .set_default("performance.total_target_ms", 325)?
            
            // Monitoring defaults
            .set_default("monitoring.prometheus_enabled", true)?
            .set_default("monitoring.jaeger_endpoint", "http://localhost:14268/api/traces")?
            .set_default("monitoring.metrics_port", 9090)?
            .set_default("monitoring.health_check_interval", "30s")?;

        // Add configuration file if it exists
        config = config.add_source(
            File::with_name("config/workflow-engine")
                .required(false)
        );

        // Add environment variables
        config = config.add_source(
            Environment::with_prefix("WORKFLOW_ENGINE")
                .separator("_")
        );

        let settings: Settings = config.build()?.try_deserialize()?;
        
        Ok(settings)
    }
    
    pub fn validate(&self) -> Result<(), ConfigError> {
        if self.database.url.is_empty() {
            return Err(ConfigError::Message("database.url is required".into()));
        }
        
        if self.google_cloud.project_id.is_empty() {
            return Err(ConfigError::Message("google_cloud.project_id is required".into()));
        }
        
        if self.external_services.auth_service_url.is_empty() {
            return Err(ConfigError::Message("external_services.auth_service_url is required".into()));
        }
        
        Ok(())
    }
}

impl Default for Settings {
    fn default() -> Self {
        Self::new().expect("Failed to load default settings")
    }
}
```

### 2. Domain Models with Type Safety

**Python Implementation:**
```python
@dataclass
class SnapshotReference:
    snapshot_id: str
    checksum: str
    created_at: datetime
    status: SnapshotStatus
```

**Rust Implementation:**
```rust
// src/domain/snapshot.rs
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;
use sha2::{Sha256, Digest};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum SnapshotStatus {
    Created,
    Active,
    Expired,
    Archived,
    Corrupted,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "lowercase")]
pub enum WorkflowPhase {
    Calculate,
    Validate,
    Commit,
    Override,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotReference {
    pub snapshot_id: String,
    pub checksum: String,
    pub created_at: DateTime<Utc>,
    pub expires_at: DateTime<Utc>,
    pub status: SnapshotStatus,
    pub phase_created: WorkflowPhase,
    pub patient_id: String,
    pub context_version: String,
    pub metadata: HashMap<String, serde_json::Value>,
}

impl SnapshotReference {
    pub fn new(
        patient_id: String,
        phase_created: WorkflowPhase,
        context_version: String,
        expires_at: DateTime<Utc>,
        data: &HashMap<String, serde_json::Value>,
    ) -> Result<Self, Box<dyn std::error::Error>> {
        let snapshot_id = format!("snap_{}", Uuid::new_v4().simple());
        let checksum = Self::calculate_checksum(data)?;
        
        Ok(Self {
            snapshot_id,
            checksum,
            created_at: Utc::now(),
            expires_at,
            status: SnapshotStatus::Created,
            phase_created,
            patient_id,
            context_version,
            metadata: HashMap::new(),
        })
    }
    
    pub fn is_valid(&self) -> bool {
        let now = Utc::now();
        self.status == SnapshotStatus::Active && self.expires_at > now
    }
    
    pub fn validate_integrity(&self, data: &HashMap<String, serde_json::Value>) -> Result<bool, Box<dyn std::error::Error>> {
        let calculated_checksum = Self::calculate_checksum(data)?;
        Ok(calculated_checksum == self.checksum)
    }
    
    fn calculate_checksum(data: &HashMap<String, serde_json::Value>) -> Result<String, Box<dyn std::error::Error>> {
        let serialized = serde_json::to_string(data)?;
        let mut hasher = Sha256::new();
        hasher.update(serialized.as_bytes());
        let result = hasher.finalize();
        Ok(hex::encode(result))
    }
    
    pub fn activate(mut self) -> Self {
        self.status = SnapshotStatus::Active;
        self
    }
    
    pub fn expire(mut self) -> Self {
        self.status = SnapshotStatus::Expired;
        self
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotChainTracker {
    pub workflow_id: String,
    pub calculate_snapshot: Option<SnapshotReference>,
    pub validate_snapshot: Option<SnapshotReference>,
    pub commit_snapshot: Option<SnapshotReference>,
    pub override_snapshot: Option<SnapshotReference>,
    pub chain_created_at: DateTime<Utc>,
}

impl SnapshotChainTracker {
    pub fn new(workflow_id: String) -> Self {
        Self {
            workflow_id,
            calculate_snapshot: None,
            validate_snapshot: None,
            commit_snapshot: None,
            override_snapshot: None,
            chain_created_at: Utc::now(),
        }
    }
    
    pub fn add_phase_snapshot(&mut self, phase: WorkflowPhase, snapshot: SnapshotReference) {
        match phase {
            WorkflowPhase::Calculate => self.calculate_snapshot = Some(snapshot),
            WorkflowPhase::Validate => self.validate_snapshot = Some(snapshot),
            WorkflowPhase::Commit => self.commit_snapshot = Some(snapshot),
            WorkflowPhase::Override => self.override_snapshot = Some(snapshot),
        }
    }
    
    pub fn validate_chain_consistency(&self) -> bool {
        let snapshots: Vec<&SnapshotReference> = [
            self.calculate_snapshot.as_ref(),
            self.validate_snapshot.as_ref(),
            self.commit_snapshot.as_ref(),
        ]
        .into_iter()
        .flatten()
        .collect();
        
        if snapshots.len() < 2 {
            return true; // Single or no snapshots are consistent by definition
        }
        
        let base_snapshot = snapshots[0];
        snapshots.iter().skip(1).all(|snapshot| {
            snapshot.patient_id == base_snapshot.patient_id
                && snapshot.context_version == base_snapshot.context_version
        })
    }
    
    pub fn get_primary_snapshot(&self) -> Option<&SnapshotReference> {
        self.calculate_snapshot
            .as_ref()
            .or(self.validate_snapshot.as_ref())
            .or(self.commit_snapshot.as_ref())
            .or(self.override_snapshot.as_ref())
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecipeReference {
    pub recipe_id: String,
    pub version: String,
    pub resolved_at: DateTime<Utc>,
    pub resolution_source: String, // "cache", "service", "fallback"
    pub metadata: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvidenceEnvelope {
    pub evidence_id: String,
    pub snapshot_id: String,
    pub phase: WorkflowPhase,
    pub evidence_type: String, // "clinical_reasoning", "safety_assessment", "decision_support"
    pub content: HashMap<String, serde_json::Value>,
    pub confidence_score: f64,
    pub generated_at: DateTime<Utc>,
    pub source: String, // "flow2_engine", "safety_gateway", "clinical_rules"
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalOverride {
    pub override_id: String,
    pub workflow_id: String,
    pub snapshot_id: String,
    pub override_type: String, // "warning_override", "safety_override", "protocol_override"
    pub original_verdict: String,
    pub overridden_to: String,
    pub clinician_id: String,
    pub justification: String,
    pub override_tokens: Vec<String>,
    pub override_timestamp: DateTime<Utc>,
    pub patient_context: HashMap<String, serde_json::Value>,
}
```

### 3. Strategic Orchestrator with Async/Await

**Python Implementation:**
```python
class StrategicOrchestrator:
    async def orchestrate_medication_request(
        self, request: CalculateRequest
    ) -> Dict[str, Any]:
```

**Rust Implementation:**
```rust
// src/application/orchestration/strategic_orchestrator.rs
use crate::config::Settings;
use crate::domain::{snapshot::*, orchestration::*};
use crate::infrastructure::clients::{
    flow2::Flow2Client,
    safety_gateway::SafetyGatewayClient,
    medication_service::MedicationServiceClient,
};
use crate::application::snapshot::SnapshotService;
use crate::utils::error::{OrchestrationError, Result};

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::time::timeout;
use tracing::{info, error, warn, instrument};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(rename_all = "SCREAMING_SNAKE_CASE")]
pub enum OrchestrationResult {
    Success,
    Warning,
    Failure,
    Blocked,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalculateRequest {
    pub patient_id: String,
    pub medication_request: HashMap<String, serde_json::Value>,
    pub clinical_intent: HashMap<String, serde_json::Value>,
    pub provider_context: HashMap<String, serde_json::Value>,
    pub correlation_id: String,
    pub urgency: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalculateResponse {
    pub proposal_set_id: String,
    pub snapshot_id: String,
    pub ranked_proposals: Vec<HashMap<String, serde_json::Value>>,
    pub clinical_evidence: HashMap<String, serde_json::Value>,
    pub monitoring_plan: HashMap<String, serde_json::Value>,
    pub kb_versions: HashMap<String, String>,
    pub execution_time_ms: f64,
    pub result: OrchestrationResult,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidateRequest {
    pub proposal_set_id: String,
    pub snapshot_id: String,
    pub selected_proposals: Vec<HashMap<String, serde_json::Value>>,
    pub validation_requirements: HashMap<String, serde_json::Value>,
    pub correlation_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidateResponse {
    pub validation_id: String,
    pub verdict: String, // SAFE, WARNING, UNSAFE
    pub findings: Vec<HashMap<String, serde_json::Value>>,
    pub override_tokens: Option<Vec<String>>,
    pub approval_requirements: Option<HashMap<String, serde_json::Value>>,
    pub result: OrchestrationResult,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommitRequest {
    pub proposal_set_id: String,
    pub validation_id: String,
    pub selected_proposal: HashMap<String, serde_json::Value>,
    pub provider_decision: HashMap<String, serde_json::Value>,
    pub correlation_id: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CommitResponse {
    pub medication_order_id: String,
    pub persistence_status: String,
    pub event_publication_status: String,
    pub audit_trail_id: String,
    pub result: OrchestrationResult,
}

pub struct PerformanceTargets {
    pub calculate_ms: u64,
    pub validate_ms: u64,
    pub commit_ms: u64,
    pub total_ms: u64,
}

#[async_trait]
pub trait OrchestrationService: Send + Sync {
    async fn orchestrate_medication_request(
        &self,
        request: CalculateRequest,
    ) -> Result<HashMap<String, serde_json::Value>>;
    
    async fn health_check(&self) -> Result<HashMap<String, serde_json::Value>>;
}

pub struct StrategicOrchestrator {
    flow2_client: Arc<dyn Flow2Client>,
    safety_client: Arc<dyn SafetyGatewayClient>,
    medication_client: Arc<dyn MedicationServiceClient>,
    snapshot_service: Arc<dyn SnapshotService>,
    performance_targets: PerformanceTargets,
    settings: Settings,
}

impl StrategicOrchestrator {
    pub fn new(
        flow2_client: Arc<dyn Flow2Client>,
        safety_client: Arc<dyn SafetyGatewayClient>,
        medication_client: Arc<dyn MedicationServiceClient>,
        snapshot_service: Arc<dyn SnapshotService>,
        settings: Settings,
    ) -> Self {
        let performance_targets = PerformanceTargets {
            calculate_ms: settings.performance.calculate_target_ms,
            validate_ms: settings.performance.validate_target_ms,
            commit_ms: settings.performance.commit_target_ms,
            total_ms: settings.performance.total_target_ms,
        };
        
        Self {
            flow2_client,
            safety_client,
            medication_client,
            snapshot_service,
            performance_targets,
            settings,
        }
    }
}

#[async_trait]
impl OrchestrationService for StrategicOrchestrator {
    #[instrument(skip(self), fields(correlation_id = %request.correlation_id, patient_id = %request.patient_id))]
    async fn orchestrate_medication_request(
        &self,
        request: CalculateRequest,
    ) -> Result<HashMap<String, serde_json::Value>> {
        let orchestration_start = Instant::now();
        let correlation_id = &request.correlation_id;
        
        info!(
            correlation_id = %correlation_id,
            patient_id = %request.patient_id,
            "Starting medication orchestration"
        );
        
        // Initialize snapshot chain tracker
        let mut snapshot_chain = SnapshotChainTracker::new(correlation_id.clone());
        
        // Add timeout for the entire orchestration
        let orchestration_future = async {
            // STEP 1: CALCULATE - Generate medication proposals
            info!(correlation_id = %correlation_id, "Starting CALCULATE step");
            let calculate_response = self.execute_calculate_step(&request).await?;
            
            if calculate_response.result != OrchestrationResult::Success {
                return Ok(self.create_error_response(
                    "CALCULATE_FAILED",
                    &format!("Calculate step failed: {:?}", calculate_response.result),
                    correlation_id,
                ));
            }
            
            // Update snapshot chain
            let calculate_snapshot = self
                .snapshot_service
                .get_snapshot(&calculate_response.snapshot_id)
                .await?;
            snapshot_chain.add_phase_snapshot(WorkflowPhase::Calculate, calculate_snapshot);
            
            // STEP 2: VALIDATE - Comprehensive safety validation
            info!(correlation_id = %correlation_id, "Starting VALIDATE step");
            let validate_request = ValidateRequest {
                proposal_set_id: calculate_response.proposal_set_id.clone(),
                snapshot_id: calculate_response.snapshot_id.clone(),
                selected_proposals: calculate_response
                    .ranked_proposals
                    .iter()
                    .take(3)
                    .cloned()
                    .collect(),
                validation_requirements: [
                    ("cae_engine".to_string(), serde_json::Value::Bool(true)),
                    ("protocol_engine".to_string(), serde_json::Value::Bool(true)),
                    ("comprehensive_validation".to_string(), serde_json::Value::Bool(true)),
                ]
                .iter()
                .cloned()
                .collect(),
                correlation_id: correlation_id.clone(),
            };
            
            let validate_response = self.execute_validate_step(&validate_request).await?;
            
            // Update snapshot chain
            let validate_snapshot = self
                .snapshot_service
                .get_snapshot(&calculate_response.snapshot_id)
                .await?;
            snapshot_chain.add_phase_snapshot(WorkflowPhase::Validate, validate_snapshot);
            
            // Validate snapshot chain consistency
            if !snapshot_chain.validate_chain_consistency() {
                return Ok(self.create_error_response(
                    "SNAPSHOT_CONSISTENCY_ERROR",
                    "Snapshot consistency validation failed",
                    correlation_id,
                ));
            }
            
            // STEP 3: COMMIT - Conditional based on validation result
            match validate_response.verdict.as_str() {
                "SAFE" => {
                    info!(correlation_id = %correlation_id, "Starting COMMIT step");
                    let commit_request = CommitRequest {
                        proposal_set_id: calculate_response.proposal_set_id.clone(),
                        validation_id: validate_response.validation_id.clone(),
                        selected_proposal: calculate_response.ranked_proposals[0].clone(),
                        provider_decision: [("auto_selected".to_string(), serde_json::Value::Bool(true))]
                            .iter()
                            .cloned()
                            .collect(),
                        correlation_id: correlation_id.clone(),
                    };
                    
                    let commit_response = self.execute_commit_step(&commit_request).await?;
                    
                    // Success path
                    let total_time = orchestration_start.elapsed().as_millis() as u64;
                    
                    Ok([
                        ("status".to_string(), serde_json::Value::String("SUCCESS".to_string())),
                        ("correlation_id".to_string(), serde_json::Value::String(correlation_id.clone())),
                        ("medication_order_id".to_string(), serde_json::Value::String(commit_response.medication_order_id)),
                        ("calculation".to_string(), serde_json::json!({
                            "proposal_set_id": calculate_response.proposal_set_id,
                            "snapshot_id": calculate_response.snapshot_id,
                            "execution_time_ms": calculate_response.execution_time_ms,
                        })),
                        ("validation".to_string(), serde_json::json!({
                            "validation_id": validate_response.validation_id,
                            "verdict": validate_response.verdict,
                        })),
                        ("commitment".to_string(), serde_json::json!({
                            "order_id": commit_response.medication_order_id,
                            "audit_trail_id": commit_response.audit_trail_id,
                        })),
                        ("performance".to_string(), serde_json::json!({
                            "total_time_ms": total_time,
                            "meets_target": total_time <= self.performance_targets.total_ms,
                        })),
                    ].iter().cloned().collect())
                }
                "WARNING" => {
                    // Return to provider with warning and override options
                    Ok([
                        ("status".to_string(), serde_json::Value::String("REQUIRES_PROVIDER_DECISION".to_string())),
                        ("correlation_id".to_string(), serde_json::Value::String(correlation_id.clone())),
                        ("validation_findings".to_string(), serde_json::Value::Array(
                            validate_response.findings.into_iter().map(|f| serde_json::Value::Object(
                                f.into_iter().collect()
                            )).collect()
                        )),
                        ("override_tokens".to_string(), serde_json::Value::Array(
                            validate_response.override_tokens.unwrap_or_default()
                                .into_iter()
                                .map(serde_json::Value::String)
                                .collect()
                        )),
                        ("proposals".to_string(), serde_json::Value::Array(
                            calculate_response.ranked_proposals.into_iter().map(|p| serde_json::Value::Object(
                                p.into_iter().collect()
                            )).collect()
                        )),
                        ("snapshot_id".to_string(), serde_json::Value::String(calculate_response.snapshot_id)),
                    ].iter().cloned().collect())
                }
                _ => {
                    // UNSAFE - Block and suggest alternatives
                    let alternatives = self
                        .generate_alternatives(&calculate_response.snapshot_id, &validate_response.findings)
                        .await
                        .unwrap_or_default();
                        
                    Ok([
                        ("status".to_string(), serde_json::Value::String("BLOCKED_UNSAFE".to_string())),
                        ("correlation_id".to_string(), serde_json::Value::String(correlation_id.clone())),
                        ("blocking_findings".to_string(), serde_json::Value::Array(
                            validate_response.findings.into_iter().map(|f| serde_json::Value::Object(
                                f.into_iter().collect()
                            )).collect()
                        )),
                        ("alternative_approaches".to_string(), serde_json::Value::Array(alternatives)),
                    ].iter().cloned().collect())
                }
            }
        };
        
        // Apply timeout to orchestration
        let timeout_duration = Duration::from_secs(30); // 30-second timeout
        match timeout(timeout_duration, orchestration_future).await {
            Ok(result) => result,
            Err(_) => Ok(self.create_error_response(
                "ORCHESTRATION_TIMEOUT",
                "Orchestration exceeded timeout limit",
                correlation_id,
            )),
        }
    }
    
    #[instrument(skip(self))]
    async fn health_check(&self) -> Result<HashMap<String, serde_json::Value>> {
        let mut services_status = HashMap::new();
        
        // Check Flow2 Go Engine
        match self.flow2_client.health_check().await {
            Ok(_) => services_status.insert("flow2_go".to_string(), "healthy".to_string()),
            Err(_) => services_status.insert("flow2_go".to_string(), "unavailable".to_string()),
        };
        
        // Check Safety Gateway
        match self.safety_client.health_check().await {
            Ok(_) => services_status.insert("safety_gateway".to_string(), "healthy".to_string()),
            Err(_) => services_status.insert("safety_gateway".to_string(), "unavailable".to_string()),
        };
        
        // Check Medication Service
        match self.medication_client.health_check().await {
            Ok(_) => services_status.insert("medication_service".to_string(), "healthy".to_string()),
            Err(_) => services_status.insert("medication_service".to_string(), "unavailable".to_string()),
        };
        
        let overall_healthy = services_status.values().all(|status| status == "healthy");
        let status = if overall_healthy { "healthy" } else { "degraded" };
        
        Ok([
            ("status".to_string(), serde_json::Value::String(status.to_string())),
            ("services".to_string(), serde_json::Value::Object(
                services_status.into_iter().map(|(k, v)| (k, serde_json::Value::String(v))).collect()
            )),
            ("orchestration_pattern".to_string(), serde_json::Value::String("Calculate > Validate > Commit".to_string())),
            ("performance_targets".to_string(), serde_json::json!({
                "calculate_ms": self.performance_targets.calculate_ms,
                "validate_ms": self.performance_targets.validate_ms,
                "commit_ms": self.performance_targets.commit_ms,
                "total_ms": self.performance_targets.total_ms,
            })),
        ].iter().cloned().collect())
    }
}

impl StrategicOrchestrator {
    #[instrument(skip(self), fields(correlation_id = %request.correlation_id))]
    async fn execute_calculate_step(&self, request: &CalculateRequest) -> Result<CalculateResponse> {
        let calculate_start = Instant::now();
        
        let flow2_request = crate::infrastructure::clients::flow2::ExecuteAdvancedRequest {
            patient_id: request.patient_id.clone(),
            medication: request.medication_request.clone(),
            clinical_intent: request.clinical_intent.clone(),
            provider_context: request.provider_context.clone(),
            execution_mode: "snapshot_optimized".to_string(),
            correlation_id: request.correlation_id.clone(),
        };
        
        match self.flow2_client.execute_advanced(&flow2_request).await {
            Ok(flow2_result) => {
                let execution_time = calculate_start.elapsed().as_secs_f64() * 1000.0; // Convert to milliseconds
                
                Ok(CalculateResponse {
                    proposal_set_id: flow2_result.proposal_set_id,
                    snapshot_id: flow2_result.snapshot_id,
                    ranked_proposals: flow2_result.ranked_proposals,
                    clinical_evidence: flow2_result.clinical_evidence,
                    monitoring_plan: flow2_result.monitoring_plan,
                    kb_versions: flow2_result.kb_versions,
                    execution_time_ms: execution_time,
                    result: OrchestrationResult::Success,
                })
            }
            Err(e) => {
                error!(error = %e, "Calculate step failed");
                Ok(CalculateResponse {
                    proposal_set_id: String::new(),
                    snapshot_id: String::new(),
                    ranked_proposals: Vec::new(),
                    clinical_evidence: HashMap::new(),
                    monitoring_plan: HashMap::new(),
                    kb_versions: HashMap::new(),
                    execution_time_ms: 0.0,
                    result: OrchestrationResult::Failure,
                })
            }
        }
    }
    
    #[instrument(skip(self), fields(correlation_id = %request.correlation_id))]
    async fn execute_validate_step(&self, request: &ValidateRequest) -> Result<ValidateResponse> {
        info!(
            correlation_id = %request.correlation_id,
            "Executing VALIDATE step via Safety Gateway"
        );
        
        let safety_request = crate::infrastructure::clients::safety_gateway::ValidationRequest {
            proposal_set_id: request.proposal_set_id.clone(),
            snapshot_id: request.snapshot_id.clone(),
            proposals: request.selected_proposals.clone(),
            patient_context: HashMap::new(), // Will be populated from snapshot
            validation_requirements: request.validation_requirements.clone(),
            correlation_id: request.correlation_id.clone(),
        };
        
        match self.safety_client.comprehensive_validation(&safety_request).await {
            Ok(safety_response) => {
                // Convert Safety Gateway response to orchestrator format
                if safety_response.verdict == "ERROR" {
                    return Ok(ValidateResponse {
                        validation_id: safety_response.validation_id,
                        verdict: "UNSAFE".to_string(),
                        findings: vec![[("error".to_string(), serde_json::Value::String("Safety Gateway validation error".to_string()))]
                            .iter()
                            .cloned()
                            .collect()],
                        override_tokens: None,
                        approval_requirements: None,
                        result: OrchestrationResult::Failure,
                    });
                }
                
                Ok(ValidateResponse {
                    validation_id: safety_response.validation_id,
                    verdict: safety_response.verdict,
                    findings: safety_response.findings,
                    override_tokens: safety_response.override_tokens,
                    approval_requirements: safety_response.override_requirements,
                    result: OrchestrationResult::Success,
                })
            }
            Err(e) => {
                error!(
                    correlation_id = %request.correlation_id,
                    error = %e,
                    "Validate step failed"
                );
                Ok(ValidateResponse {
                    validation_id: String::new(),
                    verdict: "UNSAFE".to_string(),
                    findings: vec![[
                        ("error".to_string(), serde_json::Value::String(format!("Safety Gateway validation failed: {}", e))),
                        ("severity".to_string(), serde_json::Value::String("CRITICAL".to_string())),
                        ("category".to_string(), serde_json::Value::String("SYSTEM_ERROR".to_string())),
                    ]
                    .iter()
                    .cloned()
                    .collect()],
                    override_tokens: None,
                    approval_requirements: None,
                    result: OrchestrationResult::Failure,
                })
            }
        }
    }
    
    #[instrument(skip(self), fields(correlation_id = %request.correlation_id))]
    async fn execute_commit_step(&self, request: &CommitRequest) -> Result<CommitResponse> {
        let medication_request = crate::infrastructure::clients::medication_service::CommitRequest {
            proposal_set_id: request.proposal_set_id.clone(),
            validation_id: request.validation_id.clone(),
            selected_proposal: request.selected_proposal.clone(),
            provider_decision: request.provider_decision.clone(),
            correlation_id: request.correlation_id.clone(),
        };
        
        match self.medication_client.commit(&medication_request).await {
            Ok(medication_result) => {
                Ok(CommitResponse {
                    medication_order_id: medication_result.medication_order_id,
                    persistence_status: medication_result.persistence_status,
                    event_publication_status: medication_result.event_publication_status,
                    audit_trail_id: medication_result.audit_trail_id,
                    result: OrchestrationResult::Success,
                })
            }
            Err(e) => {
                error!(error = %e, "Commit step failed");
                Ok(CommitResponse {
                    medication_order_id: String::new(),
                    persistence_status: "FAILED".to_string(),
                    event_publication_status: "FAILED".to_string(),
                    audit_trail_id: String::new(),
                    result: OrchestrationResult::Failure,
                })
            }
        }
    }
    
    #[instrument(skip(self, findings))]
    async fn generate_alternatives(
        &self,
        snapshot_id: &str,
        findings: &[HashMap<String, serde_json::Value>],
    ) -> Result<Vec<serde_json::Value>> {
        let request = crate::infrastructure::clients::flow2::GenerateAlternativesRequest {
            snapshot_id: snapshot_id.to_string(),
            blocking_findings: findings.to_vec(),
        };
        
        match self.flow2_client.generate_alternatives(&request).await {
            Ok(response) => Ok(response.alternatives),
            Err(e) => {
                warn!(error = %e, "Alternative generation failed");
                Ok(Vec::new())
            }
        }
    }
    
    fn create_error_response(
        &self,
        error_code: &str,
        error_message: &str,
        correlation_id: &str,
    ) -> HashMap<String, serde_json::Value> {
        [
            ("status".to_string(), serde_json::Value::String("ERROR".to_string())),
            ("error_code".to_string(), serde_json::Value::String(error_code.to_string())),
            ("error_message".to_string(), serde_json::Value::String(error_message.to_string())),
            ("correlation_id".to_string(), serde_json::Value::String(correlation_id.to_string())),
            ("timestamp".to_string(), serde_json::Value::String(Utc::now().to_rfc3339())),
        ]
        .iter()
        .cloned()
        .collect()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use mockall::mock;
    use tokio;
    
    mock! {
        Flow2ClientImpl {}
        
        #[async_trait]
        impl Flow2Client for Flow2ClientImpl {
            async fn execute_advanced(
                &self,
                request: &crate::infrastructure::clients::flow2::ExecuteAdvancedRequest,
            ) -> Result<crate::infrastructure::clients::flow2::ExecuteAdvancedResponse>;
            
            async fn generate_alternatives(
                &self,
                request: &crate::infrastructure::clients::flow2::GenerateAlternativesRequest,
            ) -> Result<crate::infrastructure::clients::flow2::GenerateAlternativesResponse>;
            
            async fn health_check(&self) -> Result<()>;
        }
    }
    
    #[tokio::test]
    async fn test_orchestrate_medication_request_success() {
        // Test implementation with mocks
        let mut mock_flow2 = MockFlow2ClientImpl::new();
        mock_flow2
            .expect_execute_advanced()
            .returning(|_| {
                Ok(crate::infrastructure::clients::flow2::ExecuteAdvancedResponse {
                    proposal_set_id: "prop-123".to_string(),
                    snapshot_id: "snap-456".to_string(),
                    ranked_proposals: vec![
                        [("medication".to_string(), serde_json::Value::String("aspirin".to_string()))]
                            .iter()
                            .cloned()
                            .collect()
                    ],
                    clinical_evidence: HashMap::new(),
                    monitoring_plan: HashMap::new(),
                    kb_versions: HashMap::new(),
                })
            });
        
        // Additional test setup and assertions would go here
    }
    
    #[tokio::test]
    async fn test_snapshot_chain_consistency() {
        let mut chain = SnapshotChainTracker::new("workflow-123".to_string());
        
        let snapshot1 = SnapshotReference {
            snapshot_id: "snap-1".to_string(),
            checksum: "checksum1".to_string(),
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::hours(1),
            status: SnapshotStatus::Active,
            phase_created: WorkflowPhase::Calculate,
            patient_id: "patient-123".to_string(),
            context_version: "v1.0".to_string(),
            metadata: HashMap::new(),
        };
        
        let snapshot2 = SnapshotReference {
            snapshot_id: "snap-2".to_string(),
            checksum: "checksum2".to_string(),
            created_at: Utc::now(),
            expires_at: Utc::now() + chrono::Duration::hours(1),
            status: SnapshotStatus::Active,
            phase_created: WorkflowPhase::Validate,
            patient_id: "patient-123".to_string(), // Same patient
            context_version: "v1.0".to_string(),   // Same context
            metadata: HashMap::new(),
        };
        
        chain.add_phase_snapshot(WorkflowPhase::Calculate, snapshot1);
        chain.add_phase_snapshot(WorkflowPhase::Validate, snapshot2);
        
        assert!(chain.validate_chain_consistency());
    }
}
```

## Key Rust Advantages

**★ Insight ─────────────────────────────────────**
Rust delivers exceptional benefits for clinical workflow orchestration:
1. **Memory Safety**: Zero-cost abstractions prevent buffer overflows and memory corruption that could compromise patient safety
2. **Fearless Concurrency**: Ownership model eliminates data races, enabling safe parallel processing of multiple workflows
3. **Performance**: Native compilation and zero-cost abstractions deliver consistent sub-millisecond latencies
**─────────────────────────────────────────────────**

### Performance Benefits

1. **Zero-cost Abstractions**: High-level constructs compile to optimal machine code
2. **Memory Efficiency**: No garbage collection overhead, predictable memory usage
3. **Concurrent Processing**: Tokio async runtime handles thousands of concurrent workflows
4. **System Resource Usage**: Minimal system resource consumption compared to interpreted languages

### Safety Benefits

1. **Compile-time Guarantees**: Memory safety and thread safety verified at compile time
2. **Error Handling**: Result<T, E> type ensures all errors are handled explicitly
3. **Type System**: Strong typing prevents many categories of runtime errors
4. **Ownership Model**: Prevents data races and memory leaks automatically

### Maintainability Benefits

1. **Documentation**: Integrated documentation system with examples
2. **Testing**: Built-in test framework with property-based testing
3. **Dependency Management**: Cargo provides reliable dependency resolution
4. **Code Organization**: Module system enforces clean architecture

## Cargo.toml Configuration

```toml
[package]
name = "workflow-engine-service"
version = "1.0.0"
edition = "2021"
authors = ["Clinical Synthesis Hub Team"]
description = "High-performance workflow orchestration service for clinical decision support"
repository = "https://github.com/clinical-synthesis-hub/workflow-engine-rust"
license = "MIT"
keywords = ["healthcare", "workflow", "fhir", "clinical"]
categories = ["web-programming::http-server", "science"]

[dependencies]
# Async runtime
tokio = { version = "1.0", features = ["full"] }
tokio-util = "0.7"

# HTTP and API
axum = "0.7"
tower = "0.4"
tower-http = { version = "0.5", features = ["cors", "trace", "timeout"] }
hyper = { version = "1.0", features = ["full"] }

# Serialization
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
serde_yaml = "0.9"

# Configuration
config = "0.14"
figment = { version = "0.10", features = ["yaml", "env"] }

# Database
sqlx = { version = "0.7", features = ["runtime-tokio-rustls", "postgres", "uuid", "chrono", "json"] }
uuid = { version = "1.0", features = ["v4", "serde"] }

# Time and dates
chrono = { version = "0.4", features = ["serde"] }

# HTTP client
reqwest = { version = "0.11", features = ["json", "rustls-tls"] }

# Error handling
anyhow = "1.0"
thiserror = "1.0"

# Logging and tracing
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter", "json"] }
tracing-opentelemetry = "0.22"

# Monitoring
opentelemetry = { version = "0.21", features = ["metrics"] }
opentelemetry-otlp = "0.14"
opentelemetry-prometheus = "0.14"
prometheus = "0.13"

# GraphQL
async-graphql = { version = "7.0", features = ["apollo_tracing", "apollo_persisted_queries"] }
async-graphql-axum = "7.0"

# Authentication and security
jsonwebtoken = "9.0"
argon2 = "0.5"

# Cryptography
ring = "0.17"
hex = "0.4"
sha2 = "0.10"

# Utilities
once_cell = "1.19"
futures = "0.3"
async-trait = "0.1"

[dev-dependencies]
# Testing
tokio-test = "0.4"
mockall = "0.12"
proptest = "1.0"
criterion = { version = "0.5", features = ["html_reports"] }

# Test utilities
tempfile = "3.0"
wiremock = "0.5"

[profile.release]
# Optimize for performance
lto = true
codegen-units = 1
panic = "abort"
strip = true

[profile.dev]
# Optimize for development speed
incremental = true

[[bin]]
name = "workflow-engine"
path = "src/main.rs"

[[bench]]
name = "orchestration_benchmark"
harness = false
path = "benches/orchestration_benchmark.rs"
```

## Migration Strategy

### Phase 1: Foundation (Weeks 1-2)
- Project setup with Cargo workspace
- Configuration management with environment variables
- Database layer with SQLx migrations
- Basic domain models with strong typing

### Phase 2: Core Services (Weeks 3-4)
- Strategic orchestrator with async/await
- Snapshot management with integrity validation
- Client libraries with circuit breakers
- Error handling with structured errors

### Phase 3: Infrastructure (Weeks 5-6)
- HTTP server with Axum framework
- GraphQL API with async-graphql
- Authentication middleware with JWT
- Monitoring with OpenTelemetry

### Phase 4: Integration (Weeks 7-8)
- External service clients
- End-to-end testing with integration tests
- Performance benchmarking with Criterion
- Production deployment preparation

## Testing and Benchmarking

Rust provides comprehensive testing capabilities:

```rust
// Integration test example
#[cfg(test)]
mod integration_tests {
    use super::*;
    
    #[tokio::test]
    async fn test_complete_workflow_orchestration() {
        // Setup test environment
        let config = setup_test_config().await;
        let orchestrator = setup_test_orchestrator(config).await;
        
        // Test request
        let request = CalculateRequest {
            patient_id: "test-patient-123".to_string(),
            correlation_id: "test-correlation-456".to_string(),
            medication_request: test_medication_request(),
            clinical_intent: test_clinical_intent(),
            provider_context: test_provider_context(),
            urgency: Some("ROUTINE".to_string()),
        };
        
        // Execute orchestration
        let result = orchestrator
            .orchestrate_medication_request(request)
            .await
            .expect("Orchestration should succeed");
        
        // Verify results
        assert_eq!(result.get("status").unwrap(), "SUCCESS");
        assert!(result.contains_key("medication_order_id"));
        assert!(result.contains_key("performance"));
        
        // Verify performance targets were met
        let performance = result.get("performance").unwrap();
        let total_time = performance
            .get("total_time_ms")
            .unwrap()
            .as_u64()
            .unwrap();
        assert!(total_time <= 325); // Performance target
    }
}

// Benchmark example
use criterion::{criterion_group, criterion_main, Criterion};

fn benchmark_orchestration(c: &mut Criterion) {
    let rt = tokio::runtime::Runtime::new().unwrap();
    
    c.bench_function("orchestrate_medication_request", |b| {
        b.to_async(&rt).iter(|| async {
            // Benchmark the orchestration process
            let orchestrator = setup_benchmark_orchestrator().await;
            let request = create_benchmark_request();
            orchestrator.orchestrate_medication_request(request).await
        });
    });
}

criterion_group!(benches, benchmark_orchestration);
criterion_main!(benches);
```

This Rust conversion provides enterprise-grade performance, memory safety, and maintainability while preserving the clinical workflow orchestration architecture and ensuring FHIR compliance for healthcare applications.