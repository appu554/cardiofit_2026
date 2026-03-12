//! Observability Infrastructure
//!
//! This module provides comprehensive observability capabilities for the Protocol Engine,
//! including structured logging, metrics collection, distributed tracing, and alerting
//! specifically designed for clinical environments with HIPAA compliance and patient safety requirements.

pub mod logging;
pub mod metrics;
pub mod tracing;
pub mod alerting;
pub mod health_checks;
pub mod diagnostics;

// Re-export main types for convenience
pub use logging::{
    ClinicalLogger, LogConfig, ClinicalLogLevel, ClinicalContext,
    AuditLogger, SecurityLogger, PerformanceLogger
};
pub use metrics::{
    MetricsCollector, MetricsConfig, ClinicalMetrics,
    ProtocolMetrics, ServiceMetrics, MetricsSnapshot
};
pub use tracing::{
    ClinicalTracer, TracingConfig, TraceContext,
    ClinicalSpan, TraceEvent, TracingProvider
};
pub use alerting::{
    AlertManager, AlertConfig, ClinicalAlert, AlertSeverity,
    AlertChannel, AlertRule, NotificationTarget
};
pub use health_checks::{
    HealthMonitor, HealthConfig, HealthStatus,
    ComponentHealth, SystemHealth, HealthMetrics
};
pub use diagnostics::{
    DiagnosticsEngine, DiagnosticsConfig, SystemDiagnostics,
    PerformanceDiagnostics, ComponentDiagnostics
};

use std::sync::Arc;
use tokio::sync::RwLock;
use chrono::{DateTime, Utc};
use uuid::Uuid;

use crate::protocol::error::ProtocolResult;

/// Observability engine that coordinates all observability components
pub struct ObservabilityEngine {
    /// Clinical logger
    pub logger: Arc<ClinicalLogger>,
    /// Metrics collector
    pub metrics: Arc<MetricsCollector>,
    /// Distributed tracer
    pub tracer: Arc<ClinicalTracer>,
    /// Alert manager
    pub alerts: Arc<AlertManager>,
    /// Health monitor
    pub health: Arc<HealthMonitor>,
    /// Diagnostics engine
    pub diagnostics: Arc<DiagnosticsEngine>,
    /// Engine configuration
    config: ObservabilityConfig,
    /// Engine state
    state: Arc<RwLock<ObservabilityState>>,
}

/// Observability configuration
#[derive(Debug, Clone)]
pub struct ObservabilityConfig {
    /// Logging configuration
    pub logging: LogConfig,
    /// Metrics configuration
    pub metrics: MetricsConfig,
    /// Tracing configuration
    pub tracing: TracingConfig,
    /// Alerting configuration
    pub alerting: AlertConfig,
    /// Health monitoring configuration
    pub health: HealthConfig,
    /// Diagnostics configuration
    pub diagnostics: DiagnosticsConfig,
    /// Global observability settings
    pub global: GlobalObservabilityConfig,
}

/// Global observability settings
#[derive(Debug, Clone)]
pub struct GlobalObservabilityConfig {
    /// Enable observability
    pub enabled: bool,
    /// Service name for identification
    pub service_name: String,
    /// Service version
    pub service_version: String,
    /// Environment (prod, staging, dev)
    pub environment: String,
    /// Data center/region
    pub region: String,
    /// HIPAA compliance mode
    pub hipaa_compliance: bool,
    /// Maximum retention period for logs/traces
    pub max_retention_days: u32,
    /// Sampling rate for high-volume events
    pub sampling_rate: f64,
}

/// Observability engine state
#[derive(Debug)]
struct ObservabilityState {
    /// Engine start time
    pub started_at: DateTime<Utc>,
    /// Component status
    pub component_status: std::collections::HashMap<String, ComponentStatus>,
    /// Active traces count
    pub active_traces: u64,
    /// Active alerts count
    pub active_alerts: u64,
    /// Health check results
    pub last_health_check: Option<SystemHealth>,
}

/// Component status
#[derive(Debug, Clone)]
pub struct ComponentStatus {
    /// Component name
    pub name: String,
    /// Is healthy
    pub healthy: bool,
    /// Last update time
    pub last_updated: DateTime<Utc>,
    /// Error message if unhealthy
    pub error: Option<String>,
    /// Component metrics
    pub metrics: std::collections::HashMap<String, f64>,
}

impl ObservabilityEngine {
    /// Create new observability engine
    pub async fn new(config: ObservabilityConfig) -> ProtocolResult<Self> {
        // Initialize components
        let logger = Arc::new(ClinicalLogger::new(config.logging.clone()).await?);
        let metrics = Arc::new(MetricsCollector::new(config.metrics.clone()).await?);
        let tracer = Arc::new(ClinicalTracer::new(config.tracing.clone()).await?);
        let alerts = Arc::new(AlertManager::new(config.alerting.clone()).await?);
        let health = Arc::new(HealthMonitor::new(config.health.clone()).await?);
        let diagnostics = Arc::new(DiagnosticsEngine::new(config.diagnostics.clone()).await?);

        let state = ObservabilityState {
            started_at: Utc::now(),
            component_status: std::collections::HashMap::new(),
            active_traces: 0,
            active_alerts: 0,
            last_health_check: None,
        };

        Ok(Self {
            logger,
            metrics,
            tracer,
            alerts,
            health,
            diagnostics,
            config,
            state: Arc::new(RwLock::new(state)),
        })
    }

    /// Start observability engine
    pub async fn start(&self) -> ProtocolResult<()> {
        // Start all components
        self.logger.start().await?;
        self.metrics.start().await?;
        self.tracer.start().await?;
        self.alerts.start().await?;
        self.health.start().await?;
        self.diagnostics.start().await?;

        // Log engine startup
        self.logger.log_info(
            "Observability engine started",
            ClinicalContext::system(),
            None,
        ).await?;

        Ok(())
    }

    /// Stop observability engine
    pub async fn stop(&self) -> ProtocolResult<()> {
        // Stop all components
        self.logger.stop().await?;
        self.metrics.stop().await?;
        self.tracer.stop().await?;
        self.alerts.stop().await?;
        self.health.stop().await?;
        self.diagnostics.stop().await?;

        Ok(())
    }

    /// Get observability status
    pub async fn status(&self) -> ObservabilityStatus {
        let state = self.state.read().await;
        
        ObservabilityStatus {
            running: true,
            started_at: state.started_at,
            components: state.component_status.clone(),
            active_traces: state.active_traces,
            active_alerts: state.active_alerts,
            last_health_check: state.last_health_check.clone(),
        }
    }

    /// Update component status
    pub async fn update_component_status(&self, component: ComponentStatus) {
        let mut state = self.state.write().await;
        state.component_status.insert(component.name.clone(), component);
    }

    /// Get system metrics snapshot
    pub async fn get_metrics_snapshot(&self) -> MetricsSnapshot {
        self.metrics.get_snapshot().await
    }

    /// Perform system diagnostics
    pub async fn run_diagnostics(&self) -> SystemDiagnostics {
        self.diagnostics.run_system_diagnostics().await
    }
}

/// Observability status
#[derive(Debug, Clone)]
pub struct ObservabilityStatus {
    /// Is engine running
    pub running: bool,
    /// Engine start time
    pub started_at: DateTime<Utc>,
    /// Component status map
    pub components: std::collections::HashMap<String, ComponentStatus>,
    /// Active traces count
    pub active_traces: u64,
    /// Active alerts count
    pub active_alerts: u64,
    /// Last health check result
    pub last_health_check: Option<SystemHealth>,
}

impl Default for ObservabilityConfig {
    fn default() -> Self {
        Self {
            logging: LogConfig::default(),
            metrics: MetricsConfig::default(),
            tracing: TracingConfig::default(),
            alerting: AlertConfig::default(),
            health: HealthConfig::default(),
            diagnostics: DiagnosticsConfig::default(),
            global: GlobalObservabilityConfig {
                enabled: true,
                service_name: "protocol-engine".to_string(),
                service_version: "1.0.0".to_string(),
                environment: "production".to_string(),
                region: "us-east-1".to_string(),
                hipaa_compliance: true,
                max_retention_days: 2555, // 7 years for clinical data
                sampling_rate: 1.0, // Full sampling for clinical systems
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_observability_engine_creation() {
        let config = ObservabilityConfig::default();
        let engine = ObservabilityEngine::new(config).await;
        assert!(engine.is_ok());
    }

    #[tokio::test]
    async fn test_observability_engine_lifecycle() {
        let config = ObservabilityConfig::default();
        let engine = ObservabilityEngine::new(config).await.unwrap();
        
        // Test start
        let start_result = engine.start().await;
        assert!(start_result.is_ok());
        
        // Test status
        let status = engine.status().await;
        assert!(status.running);
        
        // Test stop
        let stop_result = engine.stop().await;
        assert!(stop_result.is_ok());
    }
}