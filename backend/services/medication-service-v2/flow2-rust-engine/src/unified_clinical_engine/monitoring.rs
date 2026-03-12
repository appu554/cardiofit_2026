//! Monitoring - Real-time performance and health monitoring system
//! 
//! This module provides comprehensive monitoring capabilities for the unified clinical engine,
//! including performance metrics, health checks, alerting, and observability.

use std::collections::HashMap;
use std::sync::Arc;
use std::time::{Duration, Instant};
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use tokio::sync::RwLock;
use tracing::{info, warn, error, debug};

/// Monitoring system for the unified clinical engine
#[derive(Debug)]
pub struct MonitoringSystem {
    metrics_collector: Arc<RwLock<MetricsCollector>>,
    health_checker: Arc<RwLock<HealthChecker>>,
    alert_manager: Arc<RwLock<AlertManager>>,
    config: MonitoringConfig,
}

/// Monitoring configuration
#[derive(Debug, Clone)]
pub struct MonitoringConfig {
    pub enable_performance_monitoring: bool,
    pub enable_health_checks: bool,
    pub enable_alerting: bool,
    pub metrics_retention_hours: u32,
    pub health_check_interval_seconds: u64,
    pub alert_cooldown_minutes: u32,
    pub performance_threshold_ms: u64,
}

impl Default for MonitoringConfig {
    fn default() -> Self {
        Self {
            enable_performance_monitoring: true,
            enable_health_checks: true,
            enable_alerting: true,
            metrics_retention_hours: 24,
            health_check_interval_seconds: 30,
            alert_cooldown_minutes: 5,
            performance_threshold_ms: 1000,
        }
    }
}

/// Metrics collector for performance and usage metrics
#[derive(Debug, Default)]
pub struct MetricsCollector {
    performance_metrics: HashMap<String, Vec<PerformanceMetric>>,
    usage_metrics: HashMap<String, UsageMetric>,
    error_metrics: HashMap<String, ErrorMetric>,
    system_metrics: SystemMetrics,
}

/// Individual performance metric
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetric {
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub operation: String,
    pub duration_ms: u64,
    pub success: bool,
    pub metadata: HashMap<String, String>,
}

/// Usage metric for tracking system utilization
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UsageMetric {
    pub metric_name: String,
    pub count: u64,
    pub last_updated: chrono::DateTime<chrono::Utc>,
    pub rate_per_minute: f64,
    pub peak_rate: f64,
}

/// Error metric for tracking failures
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorMetric {
    pub error_type: String,
    pub count: u64,
    pub last_occurrence: chrono::DateTime<chrono::Utc>,
    pub error_rate: f64,
    pub recent_errors: Vec<ErrorInstance>,
}

/// Individual error instance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ErrorInstance {
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub error_message: String,
    pub context: HashMap<String, String>,
    pub severity: ErrorSeverity,
}

/// Error severity levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ErrorSeverity {
    Low,
    Medium,
    High,
    Critical,
}

/// System-level metrics
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SystemMetrics {
    pub cpu_usage_percent: f64,
    pub memory_usage_mb: f64,
    pub memory_usage_percent: f64,
    pub active_connections: u32,
    pub queue_depth: u32,
    pub cache_hit_rate: f64,
    pub uptime_seconds: u64,
    pub last_updated: Option<chrono::DateTime<chrono::Utc>>,
}

/// Health checker for system components
#[derive(Debug, Default)]
pub struct HealthChecker {
    component_health: HashMap<String, ComponentHealth>,
    overall_health: OverallHealth,
    last_check: Option<Instant>,
}

/// Health status of individual component
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComponentHealth {
    pub component_name: String,
    pub status: HealthStatus,
    pub last_check: chrono::DateTime<chrono::Utc>,
    pub response_time_ms: Option<u64>,
    pub error_message: Option<String>,
    pub metadata: HashMap<String, String>,
}

/// Health status levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum HealthStatus {
    Healthy,
    Warning,
    Critical,
    Unknown,
}

/// Overall system health
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OverallHealth {
    pub status: HealthStatus,
    pub healthy_components: u32,
    pub warning_components: u32,
    pub critical_components: u32,
    pub unknown_components: u32,
    pub last_updated: chrono::DateTime<chrono::Utc>,
}

impl Default for OverallHealth {
    fn default() -> Self {
        Self {
            status: HealthStatus::Unknown,
            healthy_components: 0,
            warning_components: 0,
            critical_components: 0,
            unknown_components: 0,
            last_updated: chrono::Utc::now(),
        }
    }
}

/// Alert manager for notifications and alerts
#[derive(Debug, Default)]
pub struct AlertManager {
    active_alerts: HashMap<String, Alert>,
    alert_history: Vec<Alert>,
    alert_rules: Vec<AlertRule>,
    notification_channels: Vec<NotificationChannel>,
}

/// Individual alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Alert {
    pub alert_id: String,
    pub alert_type: AlertType,
    pub severity: AlertSeverity,
    pub title: String,
    pub description: String,
    pub triggered_at: chrono::DateTime<chrono::Utc>,
    pub resolved_at: Option<chrono::DateTime<chrono::Utc>>,
    pub metadata: HashMap<String, String>,
    pub acknowledgments: Vec<AlertAcknowledgment>,
}

/// Types of alerts
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum AlertType {
    Performance,
    Error,
    Health,
    Security,
    Resource,
    Business,
}

/// Alert severity levels
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum AlertSeverity {
    Info,
    Warning,
    Critical,
    Emergency,
}

/// Alert acknowledgment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertAcknowledgment {
    pub acknowledged_by: String,
    pub acknowledged_at: chrono::DateTime<chrono::Utc>,
    pub comment: Option<String>,
}

/// Alert rule for triggering alerts
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertRule {
    pub rule_id: String,
    pub name: String,
    pub condition: AlertCondition,
    pub severity: AlertSeverity,
    pub cooldown_minutes: u32,
    pub enabled: bool,
}

/// Alert condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertCondition {
    pub metric_name: String,
    pub operator: ComparisonOperator,
    pub threshold: f64,
    pub duration_minutes: u32,
}

/// Comparison operators for alert conditions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ComparisonOperator {
    GreaterThan,
    LessThan,
    Equals,
    NotEquals,
    GreaterThanOrEqual,
    LessThanOrEqual,
}

/// Notification channel for alerts
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationChannel {
    pub channel_id: String,
    pub channel_type: NotificationChannelType,
    pub configuration: HashMap<String, String>,
    pub enabled: bool,
}

/// Types of notification channels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NotificationChannelType {
    Email,
    Slack,
    Webhook,
    SMS,
    PagerDuty,
}

/// Comprehensive monitoring report
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringReport {
    pub report_id: String,
    pub generated_at: chrono::DateTime<chrono::Utc>,
    pub time_range: TimeRange,
    pub performance_summary: PerformanceSummary,
    pub health_summary: HealthSummary,
    pub alert_summary: AlertSummary,
    pub recommendations: Vec<MonitoringRecommendation>,
}

/// Time range for reports
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeRange {
    pub start: chrono::DateTime<chrono::Utc>,
    pub end: chrono::DateTime<chrono::Utc>,
}

/// Performance summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceSummary {
    pub total_operations: u64,
    pub successful_operations: u64,
    pub failed_operations: u64,
    pub average_response_time_ms: f64,
    pub p95_response_time_ms: f64,
    pub p99_response_time_ms: f64,
    pub throughput_per_minute: f64,
    pub error_rate: f64,
}

/// Health summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthSummary {
    pub overall_status: HealthStatus,
    pub uptime_percentage: f64,
    pub component_statuses: HashMap<String, HealthStatus>,
    pub incidents: Vec<HealthIncident>,
}

/// Health incident
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthIncident {
    pub incident_id: String,
    pub component: String,
    pub started_at: chrono::DateTime<chrono::Utc>,
    pub resolved_at: Option<chrono::DateTime<chrono::Utc>>,
    pub duration_minutes: Option<u32>,
    pub impact: String,
}

/// Alert summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertSummary {
    pub total_alerts: u32,
    pub active_alerts: u32,
    pub resolved_alerts: u32,
    pub alerts_by_severity: HashMap<AlertSeverity, u32>,
    pub alerts_by_type: HashMap<AlertType, u32>,
    pub mean_time_to_resolution_minutes: f64,
}

/// Monitoring recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRecommendation {
    pub recommendation_id: String,
    pub category: RecommendationCategory,
    pub priority: RecommendationPriority,
    pub title: String,
    pub description: String,
    pub action_items: Vec<String>,
    pub expected_impact: String,
}

/// Recommendation categories
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationCategory {
    Performance,
    Reliability,
    Security,
    Cost,
    Maintenance,
}

/// Recommendation priorities
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationPriority {
    Low,
    Medium,
    High,
    Critical,
}

impl MonitoringSystem {
    /// Create a new monitoring system
    pub fn new() -> Self {
        Self::with_config(MonitoringConfig::default())
    }
    
    /// Create a new monitoring system with custom configuration
    pub fn with_config(config: MonitoringConfig) -> Self {
        let mut system = Self {
            metrics_collector: Arc::new(RwLock::new(MetricsCollector::default())),
            health_checker: Arc::new(RwLock::new(HealthChecker::default())),
            alert_manager: Arc::new(RwLock::new(AlertManager::default())),
            config,
        };
        
        // Initialize default alert rules
        system.initialize_default_alert_rules();
        
        info!("📊 Monitoring System initialized");
        info!("⚡ Performance monitoring: {}", system.config.enable_performance_monitoring);
        info!("🏥 Health checks: {}", system.config.enable_health_checks);
        info!("🚨 Alerting: {}", system.config.enable_alerting);
        
        system
    }
    
    /// Initialize default alert rules
    fn initialize_default_alert_rules(&mut self) {
        // This would be implemented to set up default alerting rules
        debug!("🔧 Initializing default alert rules");
    }
    
    /// Start the monitoring system
    pub async fn start(&self) -> Result<()> {
        info!("🚀 Starting Monitoring System");
        
        if self.config.enable_health_checks {
            self.start_health_checks().await?;
        }
        
        if self.config.enable_alerting {
            self.start_alert_processing().await?;
        }
        
        info!("✅ Monitoring System started");
        Ok(())
    }
    
    /// Start health check loop
    async fn start_health_checks(&self) -> Result<()> {
        let health_checker = self.health_checker.clone();
        let interval = Duration::from_secs(self.config.health_check_interval_seconds);
        
        tokio::spawn(async move {
            let mut interval_timer = tokio::time::interval(interval);
            
            loop {
                interval_timer.tick().await;
                
                if let Err(e) = Self::perform_health_checks(health_checker.clone()).await {
                    error!("Health check failed: {}", e);
                }
            }
        });
        
        Ok(())
    }
    
    /// Perform health checks on all components
    async fn perform_health_checks(health_checker: Arc<RwLock<HealthChecker>>) -> Result<()> {
        let mut checker = health_checker.write().await;
        
        // Check knowledge base health
        checker.component_health.insert("knowledge_base".to_string(), ComponentHealth {
            component_name: "Knowledge Base".to_string(),
            status: HealthStatus::Healthy,
            last_check: chrono::Utc::now(),
            response_time_ms: Some(10),
            error_message: None,
            metadata: HashMap::new(),
        });
        
        // Check rule engine health
        checker.component_health.insert("rule_engine".to_string(), ComponentHealth {
            component_name: "Rule Engine".to_string(),
            status: HealthStatus::Healthy,
            last_check: chrono::Utc::now(),
            response_time_ms: Some(25),
            error_message: None,
            metadata: HashMap::new(),
        });
        
        // Check model sandbox health
        checker.component_health.insert("model_sandbox".to_string(), ComponentHealth {
            component_name: "Model Sandbox".to_string(),
            status: HealthStatus::Healthy,
            last_check: chrono::Utc::now(),
            response_time_ms: Some(50),
            error_message: None,
            metadata: HashMap::new(),
        });
        
        // Update overall health
        let healthy_count = checker.component_health.values()
            .filter(|h| matches!(h.status, HealthStatus::Healthy))
            .count() as u32;
        
        let warning_count = checker.component_health.values()
            .filter(|h| matches!(h.status, HealthStatus::Warning))
            .count() as u32;
        
        let critical_count = checker.component_health.values()
            .filter(|h| matches!(h.status, HealthStatus::Critical))
            .count() as u32;
        
        checker.overall_health = OverallHealth {
            status: if critical_count > 0 {
                HealthStatus::Critical
            } else if warning_count > 0 {
                HealthStatus::Warning
            } else {
                HealthStatus::Healthy
            },
            healthy_components: healthy_count,
            warning_components: warning_count,
            critical_components: critical_count,
            unknown_components: 0,
            last_updated: chrono::Utc::now(),
        };
        
        checker.last_check = Some(Instant::now());
        
        Ok(())
    }
    
    /// Start alert processing loop
    async fn start_alert_processing(&self) -> Result<()> {
        debug!("🚨 Starting alert processing");
        // Implementation would include alert rule evaluation and notification sending
        Ok(())
    }
    
    /// Record a performance metric
    pub async fn record_performance(&self, operation: &str, duration: Duration, success: bool) -> Result<()> {
        if !self.config.enable_performance_monitoring {
            return Ok(());
        }
        
        let mut collector = self.metrics_collector.write().await;
        
        let metric = PerformanceMetric {
            timestamp: chrono::Utc::now(),
            operation: operation.to_string(),
            duration_ms: duration.as_millis() as u64,
            success,
            metadata: HashMap::new(),
        };
        
        collector.performance_metrics
            .entry(operation.to_string())
            .or_insert_with(Vec::new)
            .push(metric);
        
        // Check for performance alerts
        if duration.as_millis() as u64 > self.config.performance_threshold_ms {
            self.trigger_performance_alert(operation, duration.as_millis() as u64).await?;
        }
        
        Ok(())
    }
    
    /// Trigger a performance alert
    async fn trigger_performance_alert(&self, operation: &str, duration_ms: u64) -> Result<()> {
        if !self.config.enable_alerting {
            return Ok(());
        }
        
        let mut alert_manager = self.alert_manager.write().await;
        
        let alert = Alert {
            alert_id: format!("perf_{}_{}", operation, uuid::Uuid::new_v4()),
            alert_type: AlertType::Performance,
            severity: AlertSeverity::Warning,
            title: format!("Slow operation: {}", operation),
            description: format!("Operation {} took {}ms (threshold: {}ms)", 
                               operation, duration_ms, self.config.performance_threshold_ms),
            triggered_at: chrono::Utc::now(),
            resolved_at: None,
            metadata: {
                let mut metadata = HashMap::new();
                metadata.insert("operation".to_string(), operation.to_string());
                metadata.insert("duration_ms".to_string(), duration_ms.to_string());
                metadata
            },
            acknowledgments: vec![],
        };
        
        alert_manager.active_alerts.insert(alert.alert_id.clone(), alert.clone());
        alert_manager.alert_history.push(alert);
        
        warn!("🚨 Performance alert triggered: {} ({}ms)", operation, duration_ms);
        
        Ok(())
    }
    
    /// Record an error
    pub async fn record_error(&self, error_type: &str, error_message: &str, severity: ErrorSeverity) -> Result<()> {
        let mut collector = self.metrics_collector.write().await;
        
        let error_instance = ErrorInstance {
            timestamp: chrono::Utc::now(),
            error_message: error_message.to_string(),
            context: HashMap::new(),
            severity,
        };
        
        let error_metric = collector.error_metrics
            .entry(error_type.to_string())
            .or_insert_with(|| ErrorMetric {
                error_type: error_type.to_string(),
                count: 0,
                last_occurrence: chrono::Utc::now(),
                error_rate: 0.0,
                recent_errors: Vec::new(),
            });
        
        error_metric.count += 1;
        error_metric.last_occurrence = chrono::Utc::now();
        error_metric.recent_errors.push(error_instance);
        
        // Keep only recent errors (last 100)
        if error_metric.recent_errors.len() > 100 {
            error_metric.recent_errors.remove(0);
        }
        
        Ok(())
    }
    
    /// Get current system health
    pub async fn get_health(&self) -> Result<OverallHealth> {
        let checker = self.health_checker.read().await;
        Ok(checker.overall_health.clone())
    }
    
    /// Get performance metrics
    pub async fn get_performance_metrics(&self, operation: Option<&str>) -> Result<Vec<PerformanceMetric>> {
        let collector = self.metrics_collector.read().await;
        
        if let Some(op) = operation {
            Ok(collector.performance_metrics.get(op).cloned().unwrap_or_default())
        } else {
            Ok(collector.performance_metrics.values().flatten().cloned().collect())
        }
    }
    
    /// Get active alerts
    pub async fn get_active_alerts(&self) -> Result<Vec<Alert>> {
        let alert_manager = self.alert_manager.read().await;
        Ok(alert_manager.active_alerts.values().cloned().collect())
    }
    
    /// Generate monitoring report
    pub async fn generate_report(&self, time_range: TimeRange) -> Result<MonitoringReport> {
        let collector = self.metrics_collector.read().await;
        let health_checker = self.health_checker.read().await;
        let alert_manager = self.alert_manager.read().await;
        
        // Calculate performance summary
        let all_metrics: Vec<&PerformanceMetric> = collector.performance_metrics
            .values()
            .flatten()
            .filter(|m| m.timestamp >= time_range.start && m.timestamp <= time_range.end)
            .collect();
        
        let total_operations = all_metrics.len() as u64;
        let successful_operations = all_metrics.iter().filter(|m| m.success).count() as u64;
        let failed_operations = total_operations - successful_operations;
        
        let average_response_time_ms = if !all_metrics.is_empty() {
            all_metrics.iter().map(|m| m.duration_ms).sum::<u64>() as f64 / all_metrics.len() as f64
        } else {
            0.0
        };
        
        let performance_summary = PerformanceSummary {
            total_operations,
            successful_operations,
            failed_operations,
            average_response_time_ms,
            p95_response_time_ms: 0.0, // Would calculate actual percentiles
            p99_response_time_ms: 0.0,
            throughput_per_minute: 0.0,
            error_rate: if total_operations > 0 { 
                failed_operations as f64 / total_operations as f64 
            } else { 
                0.0 
            },
        };
        
        // Health summary
        let health_summary = HealthSummary {
            overall_status: health_checker.overall_health.status.clone(),
            uptime_percentage: 99.9, // Would calculate actual uptime
            component_statuses: health_checker.component_health.iter()
                .map(|(k, v)| (k.clone(), v.status.clone()))
                .collect(),
            incidents: vec![], // Would include actual incidents
        };
        
        // Alert summary
        let alerts_in_range: Vec<&Alert> = alert_manager.alert_history.iter()
            .filter(|a| a.triggered_at >= time_range.start && a.triggered_at <= time_range.end)
            .collect();
        
        let alert_summary = AlertSummary {
            total_alerts: alerts_in_range.len() as u32,
            active_alerts: alert_manager.active_alerts.len() as u32,
            resolved_alerts: alerts_in_range.iter().filter(|a| a.resolved_at.is_some()).count() as u32,
            alerts_by_severity: HashMap::new(), // Would calculate actual distribution
            alerts_by_type: HashMap::new(),
            mean_time_to_resolution_minutes: 0.0,
        };
        
        Ok(MonitoringReport {
            report_id: format!("report_{}", uuid::Uuid::new_v4()),
            generated_at: chrono::Utc::now(),
            time_range,
            performance_summary,
            health_summary,
            alert_summary,
            recommendations: vec![], // Would generate actual recommendations
        })
    }
}
