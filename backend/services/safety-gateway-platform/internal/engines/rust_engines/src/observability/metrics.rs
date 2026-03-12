//! Clinical Metrics Collection and Monitoring
//!
//! This module provides comprehensive metrics collection for clinical systems
//! with Prometheus integration, clinical-specific metrics, and performance monitoring.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc};
use prometheus::{
    Registry, Counter, Gauge, Histogram, HistogramVec, CounterVec, GaugeVec,
    Opts, HistogramOpts, exponential_buckets, register_counter, register_gauge,
    register_histogram, register_counter_vec, register_gauge_vec, register_histogram_vec,
};

use crate::protocol::error::ProtocolResult;

/// Metrics collector configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsConfig {
    /// Enable metrics collection
    pub enabled: bool,
    /// Metrics collection interval in seconds
    pub collection_interval_seconds: u64,
    /// Prometheus configuration
    pub prometheus: PrometheusConfig,
    /// Clinical metrics configuration
    pub clinical_metrics: ClinicalMetricsConfig,
    /// Performance metrics configuration
    pub performance_metrics: PerformanceMetricsConfig,
    /// Business metrics configuration
    pub business_metrics: BusinessMetricsConfig,
    /// Retention configuration
    pub retention: MetricsRetentionConfig,
}

/// Prometheus configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PrometheusConfig {
    /// Enable Prometheus metrics
    pub enabled: bool,
    /// Metrics endpoint path
    pub endpoint_path: String,
    /// Metrics port
    pub port: u16,
    /// Registry name
    pub registry_name: String,
    /// Default labels
    pub default_labels: HashMap<String, String>,
    /// Push gateway configuration
    pub push_gateway: Option<PushGatewayConfig>,
}

/// Push gateway configuration for Prometheus
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PushGatewayConfig {
    /// Gateway URL
    pub url: String,
    /// Job name
    pub job_name: String,
    /// Push interval in seconds
    pub push_interval_seconds: u64,
    /// Authentication
    pub auth: Option<PushGatewayAuth>,
}

/// Push gateway authentication
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PushGatewayAuth {
    /// Username
    pub username: String,
    /// Password
    pub password: String,
}

/// Clinical metrics configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalMetricsConfig {
    /// Enable clinical metrics
    pub enabled: bool,
    /// Track protocol execution metrics
    pub track_protocol_execution: bool,
    /// Track patient safety metrics
    pub track_patient_safety: bool,
    /// Track clinical workflow metrics
    pub track_clinical_workflows: bool,
    /// Track medication metrics
    pub track_medications: bool,
    /// Track alert metrics
    pub track_alerts: bool,
    /// Anonymize patient data in metrics
    pub anonymize_patient_data: bool,
}

/// Performance metrics configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetricsConfig {
    /// Enable performance metrics
    pub enabled: bool,
    /// Track response times
    pub track_response_times: bool,
    /// Track throughput
    pub track_throughput: bool,
    /// Track error rates
    pub track_error_rates: bool,
    /// Track resource utilization
    pub track_resource_utilization: bool,
    /// Response time histogram buckets
    pub response_time_buckets: Vec<f64>,
}

/// Business metrics configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BusinessMetricsConfig {
    /// Enable business metrics
    pub enabled: bool,
    /// Track service level objectives
    pub track_slos: bool,
    /// Track service level indicators
    pub track_slis: bool,
    /// Track compliance metrics
    pub track_compliance: bool,
    /// Track cost metrics
    pub track_costs: bool,
}

/// Metrics retention configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsRetentionConfig {
    /// High-resolution retention period (hours)
    pub high_res_retention_hours: u32,
    /// Medium-resolution retention period (days)
    pub medium_res_retention_days: u32,
    /// Low-resolution retention period (months)
    pub low_res_retention_months: u32,
    /// Enable automatic downsampling
    pub enable_downsampling: bool,
}

/// Clinical metrics data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalMetrics {
    /// Protocol execution metrics
    pub protocol_execution: ProtocolExecutionMetrics,
    /// Patient safety metrics
    pub patient_safety: PatientSafetyMetrics,
    /// Clinical workflow metrics
    pub clinical_workflows: ClinicalWorkflowMetrics,
    /// Medication metrics
    pub medications: MedicationMetrics,
    /// Alert metrics
    pub alerts: AlertMetrics,
}

/// Protocol execution metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolExecutionMetrics {
    /// Total protocol evaluations
    pub total_evaluations: u64,
    /// Successful evaluations
    pub successful_evaluations: u64,
    /// Failed evaluations
    pub failed_evaluations: u64,
    /// Average evaluation time (milliseconds)
    pub avg_evaluation_time_ms: f64,
    /// Protocol completion rate
    pub completion_rate: f64,
    /// Evaluation by protocol type
    pub evaluations_by_protocol: HashMap<String, u64>,
    /// Evaluation by department
    pub evaluations_by_department: HashMap<String, u64>,
}

/// Patient safety metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientSafetyMetrics {
    /// Safety alerts triggered
    pub safety_alerts_triggered: u64,
    /// Critical alerts
    pub critical_alerts: u64,
    /// Safety protocol violations
    pub protocol_violations: u64,
    /// Near miss events
    pub near_miss_events: u64,
    /// Patient safety score
    pub safety_score: f64,
    /// Time to safety alert response (milliseconds)
    pub avg_alert_response_time_ms: f64,
}

/// Clinical workflow metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalWorkflowMetrics {
    /// Active workflows
    pub active_workflows: u64,
    /// Completed workflows
    pub completed_workflows: u64,
    /// Workflow completion time
    pub avg_workflow_completion_time_ms: f64,
    /// Workflow step completion rate
    pub step_completion_rate: f64,
    /// Workflow efficiency score
    pub efficiency_score: f64,
    /// Workflow by type
    pub workflows_by_type: HashMap<String, u64>,
}

/// Medication metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationMetrics {
    /// Medication orders processed
    pub orders_processed: u64,
    /// Drug interaction checks
    pub interaction_checks: u64,
    /// Interactions found
    pub interactions_found: u64,
    /// Dosing errors detected
    pub dosing_errors_detected: u64,
    /// Medication adherence rate
    pub adherence_rate: f64,
    /// Time to medication administration
    pub avg_admin_time_ms: f64,
}

/// Alert metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertMetrics {
    /// Total alerts generated
    pub total_alerts: u64,
    /// Alerts by severity
    pub alerts_by_severity: HashMap<String, u64>,
    /// Alert acknowledgment time
    pub avg_acknowledgment_time_ms: f64,
    /// Alert resolution time
    pub avg_resolution_time_ms: f64,
    /// False positive rate
    pub false_positive_rate: f64,
    /// Alert fatigue score
    pub fatigue_score: f64,
}

/// Service metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceMetrics {
    /// Request count
    pub request_count: u64,
    /// Error count
    pub error_count: u64,
    /// Average response time
    pub avg_response_time_ms: f64,
    /// 95th percentile response time
    pub p95_response_time_ms: f64,
    /// 99th percentile response time
    pub p99_response_time_ms: f64,
    /// Throughput (requests per second)
    pub throughput_rps: f64,
    /// Error rate
    pub error_rate: f64,
    /// Availability
    pub availability: f64,
}

/// Protocol metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolMetrics {
    /// Protocol evaluations
    pub evaluations: HashMap<String, u64>,
    /// Protocol success rates
    pub success_rates: HashMap<String, f64>,
    /// Protocol execution times
    pub execution_times: HashMap<String, f64>,
    /// Protocol compliance scores
    pub compliance_scores: HashMap<String, f64>,
}

/// Metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MetricsSnapshot {
    /// Snapshot timestamp
    pub timestamp: DateTime<Utc>,
    /// Clinical metrics
    pub clinical: ClinicalMetrics,
    /// Service metrics
    pub service: ServiceMetrics,
    /// Protocol metrics
    pub protocol: ProtocolMetrics,
    /// System metrics
    pub system: SystemMetrics,
}

/// System metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SystemMetrics {
    /// CPU usage percentage
    pub cpu_usage_percent: f64,
    /// Memory usage in MB
    pub memory_usage_mb: u64,
    /// Disk usage percentage
    pub disk_usage_percent: f64,
    /// Network bytes in
    pub network_bytes_in: u64,
    /// Network bytes out
    pub network_bytes_out: u64,
    /// Open file descriptors
    pub open_file_descriptors: u32,
    /// Active connections
    pub active_connections: u32,
}

/// Metrics collector
pub struct MetricsCollector {
    /// Configuration
    config: MetricsConfig,
    /// Prometheus registry
    registry: Registry,
    /// Clinical metrics
    clinical_metrics: Arc<RwLock<ClinicalMetrics>>,
    /// Service metrics
    service_metrics: Arc<RwLock<ServiceMetrics>>,
    /// Protocol metrics
    protocol_metrics: Arc<RwLock<ProtocolMetrics>>,
    /// System metrics
    system_metrics: Arc<RwLock<SystemMetrics>>,
    /// Prometheus counters
    counters: PrometheusCounters,
    /// Prometheus gauges
    gauges: PrometheusGauges,
    /// Prometheus histograms
    histograms: PrometheusHistograms,
    /// Metrics collection task handle
    collection_task: Option<tokio::task::JoinHandle<()>>,
}

/// Prometheus counters
struct PrometheusCounters {
    /// Protocol evaluations
    protocol_evaluations: CounterVec,
    /// Safety alerts
    safety_alerts: CounterVec,
    /// Service requests
    service_requests: CounterVec,
    /// Errors
    errors: CounterVec,
    /// Medications processed
    medications_processed: Counter,
    /// Workflow completions
    workflow_completions: CounterVec,
}

/// Prometheus gauges
struct PrometheusGauges {
    /// Active patients
    active_patients: Gauge,
    /// Active protocols
    active_protocols: Gauge,
    /// System CPU usage
    system_cpu_usage: Gauge,
    /// System memory usage
    system_memory_usage: Gauge,
    /// Active connections
    active_connections: Gauge,
    /// Safety score
    safety_score: GaugeVec,
}

/// Prometheus histograms
struct PrometheusHistograms {
    /// Response time
    response_time: HistogramVec,
    /// Protocol evaluation time
    protocol_evaluation_time: HistogramVec,
    /// Alert response time
    alert_response_time: HistogramVec,
    /// Workflow completion time
    workflow_completion_time: HistogramVec,
}

impl MetricsCollector {
    /// Create new metrics collector
    pub async fn new(config: MetricsConfig) -> ProtocolResult<Self> {
        let registry = Registry::new();

        // Initialize Prometheus metrics
        let counters = Self::create_prometheus_counters(&registry)?;
        let gauges = Self::create_prometheus_gauges(&registry)?;
        let histograms = Self::create_prometheus_histograms(&registry, &config)?;

        // Initialize metric storage
        let clinical_metrics = Arc::new(RwLock::new(ClinicalMetrics::default()));
        let service_metrics = Arc::new(RwLock::new(ServiceMetrics::default()));
        let protocol_metrics = Arc::new(RwLock::new(ProtocolMetrics::default()));
        let system_metrics = Arc::new(RwLock::new(SystemMetrics::default()));

        Ok(Self {
            config,
            registry,
            clinical_metrics,
            service_metrics,
            protocol_metrics,
            system_metrics,
            counters,
            gauges,
            histograms,
            collection_task: None,
        })
    }

    /// Start metrics collection
    pub async fn start(&mut self) -> ProtocolResult<()> {
        if !self.config.enabled {
            return Ok(());
        }

        // Start collection task
        let collector = self.clone();
        let task = tokio::spawn(async move {
            collector.collection_task().await;
        });

        self.collection_task = Some(task);
        Ok(())
    }

    /// Stop metrics collection
    pub async fn stop(&mut self) -> ProtocolResult<()> {
        if let Some(task) = &self.collection_task {
            task.abort();
        }
        self.collection_task = None;
        Ok(())
    }

    /// Record protocol evaluation
    pub async fn record_protocol_evaluation(
        &self,
        protocol_id: &str,
        department: &str,
        duration_ms: u64,
        success: bool,
    ) {
        // Update clinical metrics
        {
            let mut clinical = self.clinical_metrics.write().await;
            clinical.protocol_execution.total_evaluations += 1;
            if success {
                clinical.protocol_execution.successful_evaluations += 1;
            } else {
                clinical.protocol_execution.failed_evaluations += 1;
            }
            
            // Update averages
            let total = clinical.protocol_execution.total_evaluations as f64;
            let current_avg = clinical.protocol_execution.avg_evaluation_time_ms;
            clinical.protocol_execution.avg_evaluation_time_ms = 
                (current_avg * (total - 1.0) + duration_ms as f64) / total;

            // Update by protocol type
            *clinical.protocol_execution.evaluations_by_protocol
                .entry(protocol_id.to_string())
                .or_insert(0) += 1;

            // Update by department
            *clinical.protocol_execution.evaluations_by_department
                .entry(department.to_string())
                .or_insert(0) += 1;
        }

        // Update Prometheus metrics
        self.counters.protocol_evaluations
            .with_label_values(&[protocol_id, department, if success { "success" } else { "failure" }])
            .inc();

        self.histograms.protocol_evaluation_time
            .with_label_values(&[protocol_id])
            .observe(duration_ms as f64 / 1000.0);
    }

    /// Record safety alert
    pub async fn record_safety_alert(&self, severity: &str, alert_type: &str, response_time_ms: u64) {
        // Update clinical metrics
        {
            let mut clinical = self.clinical_metrics.write().await;
            clinical.patient_safety.safety_alerts_triggered += 1;
            
            if severity == "critical" {
                clinical.patient_safety.critical_alerts += 1;
            }

            // Update average response time
            let total_alerts = clinical.patient_safety.safety_alerts_triggered as f64;
            let current_avg = clinical.patient_safety.avg_alert_response_time_ms;
            clinical.patient_safety.avg_alert_response_time_ms = 
                (current_avg * (total_alerts - 1.0) + response_time_ms as f64) / total_alerts;
        }

        // Update Prometheus metrics
        self.counters.safety_alerts
            .with_label_values(&[severity, alert_type])
            .inc();

        self.histograms.alert_response_time
            .with_label_values(&[alert_type])
            .observe(response_time_ms as f64 / 1000.0);
    }

    /// Record service request
    pub async fn record_service_request(
        &self,
        service: &str,
        method: &str,
        status_code: u16,
        duration_ms: u64,
    ) {
        let success = status_code < 400;

        // Update service metrics
        {
            let mut service_metrics = self.service_metrics.write().await;
            service_metrics.request_count += 1;
            
            if !success {
                service_metrics.error_count += 1;
            }

            // Update averages
            let total_requests = service_metrics.request_count as f64;
            let current_avg = service_metrics.avg_response_time_ms;
            service_metrics.avg_response_time_ms = 
                (current_avg * (total_requests - 1.0) + duration_ms as f64) / total_requests;

            // Update error rate
            service_metrics.error_rate = service_metrics.error_count as f64 / total_requests;
        }

        // Update Prometheus metrics
        self.counters.service_requests
            .with_label_values(&[service, method, &status_code.to_string()])
            .inc();

        if !success {
            self.counters.errors
                .with_label_values(&[service, method, "http_error"])
                .inc();
        }

        self.histograms.response_time
            .with_label_values(&[service, method])
            .observe(duration_ms as f64 / 1000.0);
    }

    /// Record medication processing
    pub async fn record_medication_processing(&self, interactions_found: u32, dosing_errors: u32) {
        // Update clinical metrics
        {
            let mut clinical = self.clinical_metrics.write().await;
            clinical.medications.orders_processed += 1;
            clinical.medications.interaction_checks += 1;
            clinical.medications.interactions_found += interactions_found as u64;
            clinical.medications.dosing_errors_detected += dosing_errors as u64;
        }

        // Update Prometheus metrics
        self.counters.medications_processed.inc();
        
        if interactions_found > 0 {
            self.counters.errors
                .with_label_values(&["medication", "processing", "interaction"])
                .inc_by(interactions_found as f64);
        }

        if dosing_errors > 0 {
            self.counters.errors
                .with_label_values(&["medication", "processing", "dosing"])
                .inc_by(dosing_errors as f64);
        }
    }

    /// Record workflow completion
    pub async fn record_workflow_completion(
        &self,
        workflow_type: &str,
        duration_ms: u64,
        steps_completed: u32,
        total_steps: u32,
    ) {
        // Update clinical metrics
        {
            let mut clinical = self.clinical_metrics.write().await;
            clinical.clinical_workflows.completed_workflows += 1;
            
            // Update averages
            let total_workflows = clinical.clinical_workflows.completed_workflows as f64;
            let current_avg = clinical.clinical_workflows.avg_workflow_completion_time_ms;
            clinical.clinical_workflows.avg_workflow_completion_time_ms = 
                (current_avg * (total_workflows - 1.0) + duration_ms as f64) / total_workflows;

            // Update completion rate
            let completion_rate = steps_completed as f64 / total_steps as f64;
            let current_rate = clinical.clinical_workflows.step_completion_rate;
            clinical.clinical_workflows.step_completion_rate = 
                (current_rate * (total_workflows - 1.0) + completion_rate) / total_workflows;

            // Update by type
            *clinical.clinical_workflows.workflows_by_type
                .entry(workflow_type.to_string())
                .or_insert(0) += 1;
        }

        // Update Prometheus metrics
        self.counters.workflow_completions
            .with_label_values(&[workflow_type])
            .inc();

        self.histograms.workflow_completion_time
            .with_label_values(&[workflow_type])
            .observe(duration_ms as f64 / 1000.0);
    }

    /// Update system metrics
    pub async fn update_system_metrics(
        &self,
        cpu_usage: f64,
        memory_usage_mb: u64,
        active_connections: u32,
    ) {
        // Update system metrics
        {
            let mut system = self.system_metrics.write().await;
            system.cpu_usage_percent = cpu_usage;
            system.memory_usage_mb = memory_usage_mb;
            system.active_connections = active_connections;
        }

        // Update Prometheus gauges
        self.gauges.system_cpu_usage.set(cpu_usage);
        self.gauges.system_memory_usage.set(memory_usage_mb as f64);
        self.gauges.active_connections.set(active_connections as f64);
    }

    /// Get metrics snapshot
    pub async fn get_snapshot(&self) -> MetricsSnapshot {
        MetricsSnapshot {
            timestamp: Utc::now(),
            clinical: self.clinical_metrics.read().await.clone(),
            service: self.service_metrics.read().await.clone(),
            protocol: self.protocol_metrics.read().await.clone(),
            system: self.system_metrics.read().await.clone(),
        }
    }

    /// Metrics collection background task
    async fn collection_task(&self) {
        let mut interval = tokio::time::interval(
            std::time::Duration::from_secs(self.config.collection_interval_seconds)
        );

        loop {
            interval.tick().await;
            
            // Collect system metrics
            self.collect_system_metrics().await;
            
            // Update derived metrics
            self.update_derived_metrics().await;
        }
    }

    /// Collect system metrics
    async fn collect_system_metrics(&self) {
        // TODO: Implement actual system metrics collection
        // This would use system libraries to get actual CPU, memory, etc.
        let cpu_usage = 15.5; // Placeholder
        let memory_usage_mb = 256; // Placeholder
        let active_connections = 10; // Placeholder

        self.update_system_metrics(cpu_usage, memory_usage_mb, active_connections).await;
    }

    /// Update derived metrics
    async fn update_derived_metrics(&self) {
        // Update safety score
        let safety_score = self.calculate_safety_score().await;
        self.gauges.safety_score
            .with_label_values(&["overall"])
            .set(safety_score);

        // Update availability
        let availability = self.calculate_availability().await;
        {
            let mut service = self.service_metrics.write().await;
            service.availability = availability;
        }
    }

    /// Calculate safety score
    async fn calculate_safety_score(&self) -> f64 {
        let clinical = self.clinical_metrics.read().await;
        
        // Simple safety score calculation
        let total_alerts = clinical.patient_safety.safety_alerts_triggered as f64;
        let critical_alerts = clinical.patient_safety.critical_alerts as f64;
        
        if total_alerts == 0.0 {
            100.0
        } else {
            let critical_ratio = critical_alerts / total_alerts;
            (1.0 - critical_ratio) * 100.0
        }
    }

    /// Calculate service availability
    async fn calculate_availability(&self) -> f64 {
        let service = self.service_metrics.read().await;
        
        if service.request_count == 0 {
            100.0
        } else {
            let success_count = service.request_count - service.error_count;
            (success_count as f64 / service.request_count as f64) * 100.0
        }
    }

    /// Create Prometheus counters
    fn create_prometheus_counters(registry: &Registry) -> ProtocolResult<PrometheusCounters> {
        let protocol_evaluations = CounterVec::new(
            Opts::new("protocol_evaluations_total", "Total protocol evaluations"),
            &["protocol_id", "department", "result"]
        )?;
        registry.register(Box::new(protocol_evaluations.clone()))?;

        let safety_alerts = CounterVec::new(
            Opts::new("safety_alerts_total", "Total safety alerts"),
            &["severity", "type"]
        )?;
        registry.register(Box::new(safety_alerts.clone()))?;

        let service_requests = CounterVec::new(
            Opts::new("service_requests_total", "Total service requests"),
            &["service", "method", "status_code"]
        )?;
        registry.register(Box::new(service_requests.clone()))?;

        let errors = CounterVec::new(
            Opts::new("errors_total", "Total errors"),
            &["service", "method", "error_type"]
        )?;
        registry.register(Box::new(errors.clone()))?;

        let medications_processed = Counter::new(
            "medications_processed_total", "Total medications processed"
        )?;
        registry.register(Box::new(medications_processed.clone()))?;

        let workflow_completions = CounterVec::new(
            Opts::new("workflow_completions_total", "Total workflow completions"),
            &["workflow_type"]
        )?;
        registry.register(Box::new(workflow_completions.clone()))?;

        Ok(PrometheusCounters {
            protocol_evaluations,
            safety_alerts,
            service_requests,
            errors,
            medications_processed,
            workflow_completions,
        })
    }

    /// Create Prometheus gauges
    fn create_prometheus_gauges(registry: &Registry) -> ProtocolResult<PrometheusGauges> {
        let active_patients = Gauge::new("active_patients", "Number of active patients")?;
        registry.register(Box::new(active_patients.clone()))?;

        let active_protocols = Gauge::new("active_protocols", "Number of active protocols")?;
        registry.register(Box::new(active_protocols.clone()))?;

        let system_cpu_usage = Gauge::new("system_cpu_usage_percent", "System CPU usage percentage")?;
        registry.register(Box::new(system_cpu_usage.clone()))?;

        let system_memory_usage = Gauge::new("system_memory_usage_mb", "System memory usage in MB")?;
        registry.register(Box::new(system_memory_usage.clone()))?;

        let active_connections = Gauge::new("active_connections", "Number of active connections")?;
        registry.register(Box::new(active_connections.clone()))?;

        let safety_score = GaugeVec::new(
            Opts::new("safety_score", "Clinical safety score"),
            &["category"]
        )?;
        registry.register(Box::new(safety_score.clone()))?;

        Ok(PrometheusGauges {
            active_patients,
            active_protocols,
            system_cpu_usage,
            system_memory_usage,
            active_connections,
            safety_score,
        })
    }

    /// Create Prometheus histograms
    fn create_prometheus_histograms(
        registry: &Registry,
        config: &MetricsConfig,
    ) -> ProtocolResult<PrometheusHistograms> {
        let buckets = if !config.performance_metrics.response_time_buckets.is_empty() {
            config.performance_metrics.response_time_buckets.clone()
        } else {
            exponential_buckets(0.001, 2.0, 15)?
        };

        let response_time = HistogramVec::new(
            HistogramOpts::new("response_time_seconds", "Response time in seconds")
                .buckets(buckets.clone()),
            &["service", "method"]
        )?;
        registry.register(Box::new(response_time.clone()))?;

        let protocol_evaluation_time = HistogramVec::new(
            HistogramOpts::new("protocol_evaluation_time_seconds", "Protocol evaluation time in seconds")
                .buckets(buckets.clone()),
            &["protocol_id"]
        )?;
        registry.register(Box::new(protocol_evaluation_time.clone()))?;

        let alert_response_time = HistogramVec::new(
            HistogramOpts::new("alert_response_time_seconds", "Alert response time in seconds")
                .buckets(buckets.clone()),
            &["alert_type"]
        )?;
        registry.register(Box::new(alert_response_time.clone()))?;

        let workflow_completion_time = HistogramVec::new(
            HistogramOpts::new("workflow_completion_time_seconds", "Workflow completion time in seconds")
                .buckets(buckets),
            &["workflow_type"]
        )?;
        registry.register(Box::new(workflow_completion_time.clone()))?;

        Ok(PrometheusHistograms {
            response_time,
            protocol_evaluation_time,
            alert_response_time,
            workflow_completion_time,
        })
    }
}

// Clone implementation for background tasks
impl Clone for MetricsCollector {
    fn clone(&self) -> Self {
        Self {
            config: self.config.clone(),
            registry: self.registry.clone(),
            clinical_metrics: Arc::clone(&self.clinical_metrics),
            service_metrics: Arc::clone(&self.service_metrics),
            protocol_metrics: Arc::clone(&self.protocol_metrics),
            system_metrics: Arc::clone(&self.system_metrics),
            counters: PrometheusCounters {
                protocol_evaluations: self.counters.protocol_evaluations.clone(),
                safety_alerts: self.counters.safety_alerts.clone(),
                service_requests: self.counters.service_requests.clone(),
                errors: self.counters.errors.clone(),
                medications_processed: self.counters.medications_processed.clone(),
                workflow_completions: self.counters.workflow_completions.clone(),
            },
            gauges: PrometheusGauges {
                active_patients: self.gauges.active_patients.clone(),
                active_protocols: self.gauges.active_protocols.clone(),
                system_cpu_usage: self.gauges.system_cpu_usage.clone(),
                system_memory_usage: self.gauges.system_memory_usage.clone(),
                active_connections: self.gauges.active_connections.clone(),
                safety_score: self.gauges.safety_score.clone(),
            },
            histograms: PrometheusHistograms {
                response_time: self.histograms.response_time.clone(),
                protocol_evaluation_time: self.histograms.protocol_evaluation_time.clone(),
                alert_response_time: self.histograms.alert_response_time.clone(),
                workflow_completion_time: self.histograms.workflow_completion_time.clone(),
            },
            collection_task: None,
        }
    }
}

// Default implementations
impl Default for ClinicalMetrics {
    fn default() -> Self {
        Self {
            protocol_execution: ProtocolExecutionMetrics::default(),
            patient_safety: PatientSafetyMetrics::default(),
            clinical_workflows: ClinicalWorkflowMetrics::default(),
            medications: MedicationMetrics::default(),
            alerts: AlertMetrics::default(),
        }
    }
}

impl Default for ProtocolExecutionMetrics {
    fn default() -> Self {
        Self {
            total_evaluations: 0,
            successful_evaluations: 0,
            failed_evaluations: 0,
            avg_evaluation_time_ms: 0.0,
            completion_rate: 0.0,
            evaluations_by_protocol: HashMap::new(),
            evaluations_by_department: HashMap::new(),
        }
    }
}

impl Default for PatientSafetyMetrics {
    fn default() -> Self {
        Self {
            safety_alerts_triggered: 0,
            critical_alerts: 0,
            protocol_violations: 0,
            near_miss_events: 0,
            safety_score: 100.0,
            avg_alert_response_time_ms: 0.0,
        }
    }
}

impl Default for ClinicalWorkflowMetrics {
    fn default() -> Self {
        Self {
            active_workflows: 0,
            completed_workflows: 0,
            avg_workflow_completion_time_ms: 0.0,
            step_completion_rate: 0.0,
            efficiency_score: 0.0,
            workflows_by_type: HashMap::new(),
        }
    }
}

impl Default for MedicationMetrics {
    fn default() -> Self {
        Self {
            orders_processed: 0,
            interaction_checks: 0,
            interactions_found: 0,
            dosing_errors_detected: 0,
            adherence_rate: 0.0,
            avg_admin_time_ms: 0.0,
        }
    }
}

impl Default for AlertMetrics {
    fn default() -> Self {
        Self {
            total_alerts: 0,
            alerts_by_severity: HashMap::new(),
            avg_acknowledgment_time_ms: 0.0,
            avg_resolution_time_ms: 0.0,
            false_positive_rate: 0.0,
            fatigue_score: 0.0,
        }
    }
}

impl Default for ServiceMetrics {
    fn default() -> Self {
        Self {
            request_count: 0,
            error_count: 0,
            avg_response_time_ms: 0.0,
            p95_response_time_ms: 0.0,
            p99_response_time_ms: 0.0,
            throughput_rps: 0.0,
            error_rate: 0.0,
            availability: 100.0,
        }
    }
}

impl Default for ProtocolMetrics {
    fn default() -> Self {
        Self {
            evaluations: HashMap::new(),
            success_rates: HashMap::new(),
            execution_times: HashMap::new(),
            compliance_scores: HashMap::new(),
        }
    }
}

impl Default for SystemMetrics {
    fn default() -> Self {
        Self {
            cpu_usage_percent: 0.0,
            memory_usage_mb: 0,
            disk_usage_percent: 0.0,
            network_bytes_in: 0,
            network_bytes_out: 0,
            open_file_descriptors: 0,
            active_connections: 0,
        }
    }
}

impl Default for MetricsConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            collection_interval_seconds: 30,
            prometheus: PrometheusConfig {
                enabled: true,
                endpoint_path: "/metrics".to_string(),
                port: 9090,
                registry_name: "protocol_engine".to_string(),
                default_labels: HashMap::from([
                    ("service".to_string(), "protocol-engine".to_string()),
                    ("version".to_string(), "1.0.0".to_string()),
                ]),
                push_gateway: None,
            },
            clinical_metrics: ClinicalMetricsConfig {
                enabled: true,
                track_protocol_execution: true,
                track_patient_safety: true,
                track_clinical_workflows: true,
                track_medications: true,
                track_alerts: true,
                anonymize_patient_data: true,
            },
            performance_metrics: PerformanceMetricsConfig {
                enabled: true,
                track_response_times: true,
                track_throughput: true,
                track_error_rates: true,
                track_resource_utilization: true,
                response_time_buckets: vec![
                    0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0
                ],
            },
            business_metrics: BusinessMetricsConfig {
                enabled: true,
                track_slos: true,
                track_slis: true,
                track_compliance: true,
                track_costs: false,
            },
            retention: MetricsRetentionConfig {
                high_res_retention_hours: 48,
                medium_res_retention_days: 30,
                low_res_retention_months: 13,
                enable_downsampling: true,
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_metrics_collector_creation() {
        let config = MetricsConfig::default();
        let collector = MetricsCollector::new(config).await;
        assert!(collector.is_ok());
    }

    #[tokio::test]
    async fn test_record_protocol_evaluation() {
        let config = MetricsConfig::default();
        let collector = MetricsCollector::new(config).await.unwrap();
        
        collector.record_protocol_evaluation(
            "sepsis-bundle-v1",
            "emergency",
            1500,
            true,
        ).await;

        let snapshot = collector.get_snapshot().await;
        assert_eq!(snapshot.clinical.protocol_execution.total_evaluations, 1);
        assert_eq!(snapshot.clinical.protocol_execution.successful_evaluations, 1);
        assert_eq!(snapshot.clinical.protocol_execution.avg_evaluation_time_ms, 1500.0);
    }

    #[tokio::test]
    async fn test_record_safety_alert() {
        let config = MetricsConfig::default();
        let collector = MetricsCollector::new(config).await.unwrap();
        
        collector.record_safety_alert("critical", "medication_interaction", 2000).await;

        let snapshot = collector.get_snapshot().await;
        assert_eq!(snapshot.clinical.patient_safety.safety_alerts_triggered, 1);
        assert_eq!(snapshot.clinical.patient_safety.critical_alerts, 1);
        assert_eq!(snapshot.clinical.patient_safety.avg_alert_response_time_ms, 2000.0);
    }

    #[tokio::test]
    async fn test_safety_score_calculation() {
        let config = MetricsConfig::default();
        let collector = MetricsCollector::new(config).await.unwrap();
        
        // Record some alerts
        collector.record_safety_alert("warning", "low_priority", 1000).await;
        collector.record_safety_alert("critical", "high_priority", 500).await;

        let safety_score = collector.calculate_safety_score().await;
        assert!(safety_score >= 0.0 && safety_score <= 100.0);
        assert_eq!(safety_score, 50.0); // 1 critical out of 2 total = 50% safety score
    }
}