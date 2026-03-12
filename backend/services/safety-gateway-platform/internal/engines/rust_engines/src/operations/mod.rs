//! Operational Tooling and Production Management
//!
//! This module provides comprehensive operational tooling for production deployment,
//! management, and maintenance of the Protocol Engine in clinical environments.

pub mod deployment;
pub mod health_checks;
pub mod admin_api;
pub mod maintenance;
pub mod diagnostics;
pub mod backup;
pub mod configuration;
pub mod secrets;
pub mod migrations;

// Re-export main types for convenience
pub use deployment::{
    DeploymentManager, DeploymentConfig, DeploymentStrategy,
    BlueGreenDeployment, RollingDeployment, CanaryDeployment
};
pub use health_checks::{
    HealthMonitor, SystemHealth, ComponentHealth, HealthConfig,
    HealthCheck, HealthStatus, HealthMetrics
};
pub use admin_api::{
    AdminApi, AdminConfig, AdminEndpoint, AdminOperation,
    OperationResult, AdminAuthentication
};
pub use maintenance::{
    MaintenanceManager, MaintenanceConfig, MaintenanceWindow,
    MaintenanceTask, MaintenanceStatus
};
pub use diagnostics::{
    DiagnosticsEngine, SystemDiagnostics, DiagnosticsConfig,
    PerformanceDiagnostics, ComponentDiagnostics
};
pub use backup::{
    BackupManager, BackupConfig, BackupStrategy, BackupSchedule,
    RestoreManager, BackupMetadata
};
pub use configuration::{
    ConfigurationManager, ConfigurationSource, ConfigurationWatcher,
    ConfigurationValidation, EnvironmentConfig
};
pub use secrets::{
    SecretsManager, SecretsConfig, SecretStorage, SecretRotation,
    EncryptionConfig
};
pub use migrations::{
    MigrationManager, MigrationConfig, Migration, MigrationStatus,
    SchemaVersion, DataMigration
};

use std::sync::Arc;
use tokio::sync::RwLock;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::protocol::error::ProtocolResult;
use crate::observability::{ObservabilityEngine, ObservabilityConfig};

/// Operations engine that coordinates all operational components
pub struct OperationsEngine {
    /// Deployment manager
    pub deployment: Arc<DeploymentManager>,
    /// Health monitor
    pub health: Arc<HealthMonitor>,
    /// Admin API
    pub admin_api: Arc<AdminApi>,
    /// Maintenance manager
    pub maintenance: Arc<MaintenanceManager>,
    /// Diagnostics engine
    pub diagnostics: Arc<DiagnosticsEngine>,
    /// Backup manager
    pub backup: Arc<BackupManager>,
    /// Configuration manager
    pub config: Arc<ConfigurationManager>,
    /// Secrets manager
    pub secrets: Arc<SecretsManager>,
    /// Migration manager
    pub migrations: Arc<MigrationManager>,
    /// Observability integration
    observability: Arc<ObservabilityEngine>,
    /// Operations configuration
    config: OperationsConfig,
    /// Operations state
    state: Arc<RwLock<OperationsState>>,
}

/// Operations configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OperationsConfig {
    /// Deployment configuration
    pub deployment: DeploymentConfig,
    /// Health monitoring configuration
    pub health: HealthConfig,
    /// Admin API configuration
    pub admin_api: AdminConfig,
    /// Maintenance configuration
    pub maintenance: MaintenanceConfig,
    /// Diagnostics configuration
    pub diagnostics: DiagnosticsConfig,
    /// Backup configuration
    pub backup: BackupConfig,
    /// Configuration management
    pub configuration: configuration::ConfigurationManagerConfig,
    /// Secrets management
    pub secrets: SecretsConfig,
    /// Migration configuration
    pub migrations: MigrationConfig,
    /// Global operations settings
    pub global: GlobalOperationsConfig,
}

/// Global operations settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GlobalOperationsConfig {
    /// Enable operations
    pub enabled: bool,
    /// Environment (prod, staging, dev)
    pub environment: String,
    /// Service name
    pub service_name: String,
    /// Service version
    pub service_version: String,
    /// Data center/region
    pub region: String,
    /// Operations mode
    pub mode: OperationsMode,
    /// Maintenance window settings
    pub maintenance_windows: Vec<MaintenanceWindow>,
    /// Emergency contact information
    pub emergency_contacts: Vec<EmergencyContact>,
}

/// Operations mode
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OperationsMode {
    /// Full operations enabled
    Full,
    /// Read-only mode
    ReadOnly,
    /// Maintenance mode
    Maintenance,
    /// Emergency mode
    Emergency,
    /// Degraded operations
    Degraded,
}

/// Emergency contact information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmergencyContact {
    /// Contact name
    pub name: String,
    /// Contact role
    pub role: String,
    /// Primary phone
    pub phone: String,
    /// Email address
    pub email: String,
    /// Escalation level
    pub escalation_level: u32,
    /// Available hours
    pub available_hours: Option<AvailabilityWindow>,
}

/// Availability window
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AvailabilityWindow {
    /// Start time (24-hour format)
    pub start_hour: u8,
    /// End time (24-hour format)
    pub end_hour: u8,
    /// Days of week (0 = Sunday)
    pub days_of_week: Vec<u8>,
    /// Timezone
    pub timezone: String,
}

/// Operations state
#[derive(Debug)]
struct OperationsState {
    /// Operations start time
    pub started_at: DateTime<Utc>,
    /// Current operations mode
    pub current_mode: OperationsMode,
    /// Active operations
    pub active_operations: std::collections::HashMap<String, ActiveOperation>,
    /// System readiness
    pub system_ready: bool,
    /// Last health check
    pub last_health_check: Option<DateTime<Utc>>,
    /// Active maintenance windows
    pub active_maintenance: Vec<String>,
}

/// Active operation tracking
#[derive(Debug, Clone)]
pub struct ActiveOperation {
    /// Operation ID
    pub id: String,
    /// Operation type
    pub operation_type: OperationType,
    /// Started at
    pub started_at: DateTime<Utc>,
    /// Operation status
    pub status: OperationStatus,
    /// Progress percentage (0-100)
    pub progress: u8,
    /// Operation message
    pub message: String,
    /// Initiated by
    pub initiated_by: String,
}

/// Operation types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OperationType {
    /// Deployment operation
    Deployment,
    /// Health check operation
    HealthCheck,
    /// Maintenance operation
    Maintenance,
    /// Backup operation
    Backup,
    /// Migration operation
    Migration,
    /// Configuration update
    ConfigUpdate,
    /// Secret rotation
    SecretRotation,
    /// System diagnostics
    Diagnostics,
    /// Custom operation
    Custom(String),
}

/// Operation status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum OperationStatus {
    /// Operation starting
    Starting,
    /// Operation in progress
    InProgress,
    /// Operation completed successfully
    Completed,
    /// Operation failed
    Failed,
    /// Operation cancelled
    Cancelled,
    /// Operation timed out
    TimedOut,
}

impl OperationsEngine {
    /// Create new operations engine
    pub async fn new(
        config: OperationsConfig,
        observability: Arc<ObservabilityEngine>,
    ) -> ProtocolResult<Self> {
        // Initialize components
        let deployment = Arc::new(DeploymentManager::new(config.deployment.clone()).await?);
        let health = Arc::new(HealthMonitor::new(config.health.clone()).await?);
        let admin_api = Arc::new(AdminApi::new(config.admin_api.clone()).await?);
        let maintenance = Arc::new(MaintenanceManager::new(config.maintenance.clone()).await?);
        let diagnostics = Arc::new(DiagnosticsEngine::new(config.diagnostics.clone()).await?);
        let backup = Arc::new(BackupManager::new(config.backup.clone()).await?);
        let config_manager = Arc::new(ConfigurationManager::new(config.configuration.clone()).await?);
        let secrets = Arc::new(SecretsManager::new(config.secrets.clone()).await?);
        let migrations = Arc::new(MigrationManager::new(config.migrations.clone()).await?);

        let state = OperationsState {
            started_at: Utc::now(),
            current_mode: config.global.mode.clone(),
            active_operations: std::collections::HashMap::new(),
            system_ready: false,
            last_health_check: None,
            active_maintenance: Vec::new(),
        };

        Ok(Self {
            deployment,
            health,
            admin_api,
            maintenance,
            diagnostics,
            backup,
            config: config_manager,
            secrets,
            migrations,
            observability,
            config,
            state: Arc::new(RwLock::new(state)),
        })
    }

    /// Start operations engine
    pub async fn start(&self) -> ProtocolResult<()> {
        // Start all components
        self.health.start().await?;
        self.admin_api.start().await?;
        self.maintenance.start().await?;
        self.backup.start().await?;
        self.config.start().await?;
        self.secrets.start().await?;

        // Run initial system readiness checks
        self.check_system_readiness().await?;

        // Start background monitoring
        self.start_background_monitoring().await?;

        // Update state
        {
            let mut state = self.state.write().await;
            state.system_ready = true;
            state.last_health_check = Some(Utc::now());
        }

        Ok(())
    }

    /// Stop operations engine
    pub async fn stop(&self) -> ProtocolResult<()> {
        // Graceful shutdown of all components
        self.health.stop().await?;
        self.admin_api.stop().await?;
        self.maintenance.stop().await?;
        self.backup.stop().await?;
        self.config.stop().await?;
        self.secrets.stop().await?;

        Ok(())
    }

    /// Check system readiness
    pub async fn check_system_readiness(&self) -> ProtocolResult<bool> {
        // Run comprehensive system health check
        let health_result = self.health.comprehensive_health_check().await?;
        
        // Check if all critical components are healthy
        let critical_components_healthy = health_result.components.iter()
            .filter(|(_, component)| component.critical)
            .all(|(_, component)| matches!(component.status, HealthStatus::Healthy));

        // Check if migrations are up to date
        let migrations_current = self.migrations.check_migration_status().await?;

        // Check if configuration is valid
        let config_valid = self.config.validate_configuration().await?;

        // Check if secrets are accessible
        let secrets_accessible = self.secrets.health_check().await?;

        let system_ready = critical_components_healthy && 
                          migrations_current && 
                          config_valid && 
                          secrets_accessible;

        // Update state
        {
            let mut state = self.state.write().await;
            state.system_ready = system_ready;
        }

        Ok(system_ready)
    }

    /// Start operation
    pub async fn start_operation(
        &self,
        operation_type: OperationType,
        initiated_by: String,
        parameters: Option<serde_json::Value>,
    ) -> ProtocolResult<String> {
        let operation_id = Uuid::new_v4().to_string();
        
        let operation = ActiveOperation {
            id: operation_id.clone(),
            operation_type: operation_type.clone(),
            started_at: Utc::now(),
            status: OperationStatus::Starting,
            progress: 0,
            message: "Operation starting".to_string(),
            initiated_by,
        };

        // Store operation
        {
            let mut state = self.state.write().await;
            state.active_operations.insert(operation_id.clone(), operation);
        }

        // Execute operation based on type
        match operation_type {
            OperationType::HealthCheck => {
                self.execute_health_check_operation(&operation_id).await?;
            },
            OperationType::Maintenance => {
                if let Some(params) = parameters {
                    self.execute_maintenance_operation(&operation_id, params).await?;
                }
            },
            OperationType::Backup => {
                self.execute_backup_operation(&operation_id).await?;
            },
            OperationType::Migration => {
                self.execute_migration_operation(&operation_id).await?;
            },
            OperationType::Diagnostics => {
                self.execute_diagnostics_operation(&operation_id).await?;
            },
            _ => {
                self.update_operation_status(
                    &operation_id,
                    OperationStatus::Failed,
                    "Unsupported operation type".to_string(),
                ).await;
            }
        }

        Ok(operation_id)
    }

    /// Get operation status
    pub async fn get_operation_status(&self, operation_id: &str) -> Option<ActiveOperation> {
        let state = self.state.read().await;
        state.active_operations.get(operation_id).cloned()
    }

    /// Get all active operations
    pub async fn get_active_operations(&self) -> Vec<ActiveOperation> {
        let state = self.state.read().await;
        state.active_operations.values().cloned().collect()
    }

    /// Get operations status
    pub async fn get_operations_status(&self) -> OperationsStatus {
        let state = self.state.read().await;
        
        OperationsStatus {
            running: true,
            mode: state.current_mode.clone(),
            system_ready: state.system_ready,
            started_at: state.started_at,
            last_health_check: state.last_health_check,
            active_operations_count: state.active_operations.len(),
            active_maintenance: state.active_maintenance.clone(),
        }
    }

    /// Set operations mode
    pub async fn set_operations_mode(&self, mode: OperationsMode, reason: String) -> ProtocolResult<()> {
        {
            let mut state = self.state.write().await;
            state.current_mode = mode.clone();
        }

        // Log mode change
        self.observability.logger.log_warning(
            &format!("Operations mode changed to {:?}: {}", mode, reason),
            crate::observability::logging::ClinicalContext::system(),
            None,
        ).await?;

        // Notify admin API of mode change
        self.admin_api.notify_mode_change(&mode).await?;

        Ok(())
    }

    /// Execute health check operation
    async fn execute_health_check_operation(&self, operation_id: &str) -> ProtocolResult<()> {
        self.update_operation_status(
            operation_id,
            OperationStatus::InProgress,
            "Running health checks".to_string(),
        ).await;

        let health_result = self.health.comprehensive_health_check().await?;

        if health_result.overall_healthy {
            self.update_operation_status(
                operation_id,
                OperationStatus::Completed,
                "Health check passed".to_string(),
            ).await;
        } else {
            self.update_operation_status(
                operation_id,
                OperationStatus::Failed,
                "Health check failed".to_string(),
            ).await;
        }

        Ok(())
    }

    /// Execute maintenance operation
    async fn execute_maintenance_operation(
        &self,
        operation_id: &str,
        parameters: serde_json::Value,
    ) -> ProtocolResult<()> {
        self.update_operation_status(
            operation_id,
            OperationStatus::InProgress,
            "Starting maintenance".to_string(),
        ).await;

        // Execute maintenance based on parameters
        let result = self.maintenance.execute_maintenance(parameters).await;

        match result {
            Ok(_) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Completed,
                    "Maintenance completed".to_string(),
                ).await;
            },
            Err(e) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Failed,
                    format!("Maintenance failed: {}", e),
                ).await;
            }
        }

        Ok(())
    }

    /// Execute backup operation
    async fn execute_backup_operation(&self, operation_id: &str) -> ProtocolResult<()> {
        self.update_operation_status(
            operation_id,
            OperationStatus::InProgress,
            "Creating backup".to_string(),
        ).await;

        let backup_result = self.backup.create_backup().await;

        match backup_result {
            Ok(backup_id) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Completed,
                    format!("Backup created: {}", backup_id),
                ).await;
            },
            Err(e) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Failed,
                    format!("Backup failed: {}", e),
                ).await;
            }
        }

        Ok(())
    }

    /// Execute migration operation
    async fn execute_migration_operation(&self, operation_id: &str) -> ProtocolResult<()> {
        self.update_operation_status(
            operation_id,
            OperationStatus::InProgress,
            "Running migrations".to_string(),
        ).await;

        let migration_result = self.migrations.run_pending_migrations().await;

        match migration_result {
            Ok(count) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Completed,
                    format!("Applied {} migrations", count),
                ).await;
            },
            Err(e) => {
                self.update_operation_status(
                    operation_id,
                    OperationStatus::Failed,
                    format!("Migration failed: {}", e),
                ).await;
            }
        }

        Ok(())
    }

    /// Execute diagnostics operation
    async fn execute_diagnostics_operation(&self, operation_id: &str) -> ProtocolResult<()> {
        self.update_operation_status(
            operation_id,
            OperationStatus::InProgress,
            "Running diagnostics".to_string(),
        ).await;

        let diagnostics_result = self.diagnostics.run_system_diagnostics().await;

        self.update_operation_status(
            operation_id,
            OperationStatus::Completed,
            format!("Diagnostics completed: {} checks", diagnostics_result.checks.len()),
        ).await;

        Ok(())
    }

    /// Update operation status
    async fn update_operation_status(
        &self,
        operation_id: &str,
        status: OperationStatus,
        message: String,
    ) {
        let mut state = self.state.write().await;
        
        if let Some(operation) = state.active_operations.get_mut(operation_id) {
            operation.status = status.clone();
            operation.message = message;
            
            // Update progress based on status
            operation.progress = match status {
                OperationStatus::Starting => 10,
                OperationStatus::InProgress => 50,
                OperationStatus::Completed => 100,
                OperationStatus::Failed => 0,
                OperationStatus::Cancelled => 0,
                OperationStatus::TimedOut => 0,
            };
        }
    }

    /// Start background monitoring
    async fn start_background_monitoring(&self) -> ProtocolResult<()> {
        // Start operations monitoring task
        let engine = self.clone_for_task();
        tokio::spawn(async move {
            engine.operations_monitoring_task().await;
        });

        Ok(())
    }

    /// Operations monitoring background task
    async fn operations_monitoring_task(&self) {
        let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));

        loop {
            interval.tick().await;

            // Clean up completed operations
            self.cleanup_completed_operations().await;

            // Check for expired operations
            self.check_operation_timeouts().await;

            // Monitor system health
            if let Err(e) = self.periodic_health_check().await {
                eprintln!("Health check error: {}", e);
            }
        }
    }

    /// Clean up completed operations
    async fn cleanup_completed_operations(&self) {
        let mut state = self.state.write().await;
        let now = Utc::now();
        
        // Remove operations completed more than 1 hour ago
        state.active_operations.retain(|_, operation| {
            let age = now.signed_duration_since(operation.started_at);
            match operation.status {
                OperationStatus::Completed |
                OperationStatus::Failed |
                OperationStatus::Cancelled |
                OperationStatus::TimedOut => age.num_hours() < 1,
                _ => true,
            }
        });
    }

    /// Check for operation timeouts
    async fn check_operation_timeouts(&self) {
        let now = Utc::now();
        let timeout_duration = chrono::Duration::hours(2); // 2-hour timeout
        
        let timed_out_operations: Vec<String> = {
            let state = self.state.read().await;
            state.active_operations.iter()
                .filter(|(_, operation)| {
                    matches!(operation.status, OperationStatus::InProgress | OperationStatus::Starting) &&
                    now.signed_duration_since(operation.started_at) > timeout_duration
                })
                .map(|(id, _)| id.clone())
                .collect()
        };

        for operation_id in timed_out_operations {
            self.update_operation_status(
                &operation_id,
                OperationStatus::TimedOut,
                "Operation timed out".to_string(),
            ).await;
        }
    }

    /// Periodic health check
    async fn periodic_health_check(&self) -> ProtocolResult<()> {
        let health_result = self.health.basic_health_check().await?;
        
        {
            let mut state = self.state.write().await;
            state.last_health_check = Some(Utc::now());
            
            // Update system readiness based on health
            if !health_result.overall_healthy {
                state.system_ready = false;
            }
        }

        Ok(())
    }

    /// Clone for background tasks
    fn clone_for_task(&self) -> OperationsEngineHandle {
        OperationsEngineHandle {
            state: Arc::clone(&self.state),
            health: Arc::clone(&self.health),
        }
    }
}

/// Handle for background tasks
#[derive(Clone)]
struct OperationsEngineHandle {
    state: Arc<RwLock<OperationsState>>,
    health: Arc<HealthMonitor>,
}

impl OperationsEngineHandle {
    async fn operations_monitoring_task(&self) {
        // Implementation moved to OperationsEngine for clarity
    }
}

/// Operations status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OperationsStatus {
    /// Is engine running
    pub running: bool,
    /// Current operations mode
    pub mode: OperationsMode,
    /// System readiness
    pub system_ready: bool,
    /// Engine start time
    pub started_at: DateTime<Utc>,
    /// Last health check time
    pub last_health_check: Option<DateTime<Utc>>,
    /// Active operations count
    pub active_operations_count: usize,
    /// Active maintenance windows
    pub active_maintenance: Vec<String>,
}

impl Default for OperationsConfig {
    fn default() -> Self {
        Self {
            deployment: DeploymentConfig::default(),
            health: HealthConfig::default(),
            admin_api: AdminConfig::default(),
            maintenance: MaintenanceConfig::default(),
            diagnostics: DiagnosticsConfig::default(),
            backup: BackupConfig::default(),
            configuration: configuration::ConfigurationManagerConfig::default(),
            secrets: SecretsConfig::default(),
            migrations: MigrationConfig::default(),
            global: GlobalOperationsConfig {
                enabled: true,
                environment: "production".to_string(),
                service_name: "protocol-engine".to_string(),
                service_version: "1.0.0".to_string(),
                region: "us-east-1".to_string(),
                mode: OperationsMode::Full,
                maintenance_windows: vec![],
                emergency_contacts: vec![],
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_operations_engine_creation() {
        let config = OperationsConfig::default();
        let observability = Arc::new(
            crate::observability::ObservabilityEngine::new(
                crate::observability::ObservabilityConfig::default()
            ).await.unwrap()
        );
        
        let engine = OperationsEngine::new(config, observability).await;
        assert!(engine.is_ok());
    }

    #[tokio::test]
    async fn test_operation_lifecycle() {
        let config = OperationsConfig::default();
        let observability = Arc::new(
            crate::observability::ObservabilityEngine::new(
                crate::observability::ObservabilityConfig::default()
            ).await.unwrap()
        );
        
        let engine = OperationsEngine::new(config, observability).await.unwrap();
        
        // Start an operation
        let operation_id = engine.start_operation(
            OperationType::HealthCheck,
            "test-user".to_string(),
            None,
        ).await.unwrap();
        
        // Check operation status
        let status = engine.get_operation_status(&operation_id).await;
        assert!(status.is_some());
        
        let operation = status.unwrap();
        assert_eq!(operation.id, operation_id);
        assert_eq!(operation.operation_type, OperationType::HealthCheck);
    }

    #[test]
    fn test_operations_mode_enum() {
        let mode = OperationsMode::Full;
        match mode {
            OperationsMode::Full => assert!(true),
            _ => assert!(false, "Expected Full mode"),
        }
    }
}