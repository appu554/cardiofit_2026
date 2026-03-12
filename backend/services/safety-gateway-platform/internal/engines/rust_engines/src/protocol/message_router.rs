//! Message Routing and Cross-Service Communication
//!
//! This module implements message routing patterns for cross-service
//! communication, enabling the Protocol Engine to coordinate with
//! various clinical services through different communication patterns.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc, oneshot};
use tokio::time::{timeout, sleep};
use uuid::Uuid;
use reqwest;

use crate::protocol::{
    types::*,
    error::*,
    event_publisher::{EventPublisher, ProtocolEvent, ProtocolEventType},
    cae_integration::{CaeEvaluationRequest, CaeEvaluationResponse},
};

/// Message router configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MessageRouterConfig {
    /// Service discovery configuration
    pub service_discovery: ServiceDiscoveryConfig,
    /// Circuit breaker configuration
    pub circuit_breaker: CircuitBreakerConfig,
    /// Retry policy configuration
    pub retry_policy: RetryPolicyConfig,
    /// Load balancing configuration
    pub load_balancer: LoadBalancerConfig,
    /// Message timeouts
    pub timeouts: TimeoutConfig,
    /// Enable message tracing
    pub enable_tracing: bool,
}

/// Service discovery configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceDiscoveryConfig {
    /// Service discovery method
    pub method: ServiceDiscoveryMethod,
    /// Service registry endpoint
    pub registry_endpoint: Option<String>,
    /// Static service endpoints
    pub static_endpoints: HashMap<String, ServiceEndpoint>,
    /// Service health check interval
    pub health_check_interval_seconds: u64,
    /// Service timeout for health checks
    pub health_check_timeout_seconds: u64,
}

/// Service discovery methods
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceDiscoveryMethod {
    /// Static configuration
    Static,
    /// Consul service discovery
    Consul,
    /// Kubernetes service discovery
    Kubernetes,
    /// Eureka service discovery
    Eureka,
    /// Custom service discovery
    Custom(String),
}

/// Service endpoint information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceEndpoint {
    /// Service name/identifier
    pub service_name: String,
    /// Base URL for the service
    pub base_url: String,
    /// Service version
    pub version: String,
    /// Health check endpoint
    pub health_endpoint: String,
    /// Service capabilities
    pub capabilities: Vec<ServiceCapability>,
    /// Service metadata
    pub metadata: HashMap<String, String>,
    /// Authentication configuration
    pub auth_config: Option<ServiceAuthConfig>,
}

/// Service capabilities
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceCapability {
    /// Clinical assessment and evaluation
    ClinicalAssessment,
    /// Medication management
    MedicationManagement,
    /// Patient data management
    PatientDataManagement,
    /// FHIR resource management
    FhirResourceManagement,
    /// Workflow orchestration
    WorkflowOrchestration,
    /// Notification services
    NotificationServices,
    /// Audit and compliance
    AuditCompliance,
    /// Reporting and analytics
    ReportingAnalytics,
}

/// Service authentication configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceAuthConfig {
    /// Authentication method
    pub method: ServiceAuthMethod,
    /// Service credentials
    pub credentials: ServiceCredentials,
}

/// Service authentication methods
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceAuthMethod {
    /// No authentication
    None,
    /// Bearer token
    Bearer,
    /// API key
    ApiKey,
    /// JWT token
    JWT,
    /// Mutual TLS
    MutualTLS,
    /// OAuth 2.0
    OAuth2,
}

/// Service credentials
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceCredentials {
    /// Token or key
    pub token: Option<String>,
    /// Client ID
    pub client_id: Option<String>,
    /// Client secret
    pub client_secret: Option<String>,
    /// Certificate path
    pub certificate_path: Option<String>,
}

/// Circuit breaker configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CircuitBreakerConfig {
    /// Enable circuit breaker
    pub enabled: bool,
    /// Failure threshold to open circuit
    pub failure_threshold: u32,
    /// Success threshold to close circuit
    pub success_threshold: u32,
    /// Timeout in seconds before attempting to close circuit
    pub timeout_seconds: u64,
    /// Window size for failure rate calculation
    pub window_size: u32,
}

/// Retry policy configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryPolicyConfig {
    /// Maximum retry attempts
    pub max_retries: u32,
    /// Base retry delay in milliseconds
    pub base_delay_ms: u64,
    /// Maximum retry delay in milliseconds
    pub max_delay_ms: u64,
    /// Retry backoff strategy
    pub backoff_strategy: BackoffStrategy,
    /// Retryable error types
    pub retryable_errors: Vec<RetryableErrorType>,
}

/// Backoff strategies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BackoffStrategy {
    /// Fixed delay
    Fixed,
    /// Linear backoff
    Linear,
    /// Exponential backoff
    Exponential,
    /// Exponential backoff with jitter
    ExponentialJitter,
}

/// Retryable error types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RetryableErrorType {
    /// Network timeout errors
    NetworkTimeout,
    /// Connection errors
    ConnectionError,
    /// Temporary service unavailable
    ServiceUnavailable,
    /// Rate limiting errors
    RateLimited,
    /// Internal server errors (5xx)
    InternalServerError,
}

/// Load balancer configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LoadBalancerConfig {
    /// Load balancing strategy
    pub strategy: LoadBalancingStrategy,
    /// Health check configuration
    pub health_check: HealthCheckConfig,
    /// Enable sticky sessions
    pub sticky_sessions: bool,
}

/// Load balancing strategies
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LoadBalancingStrategy {
    /// Round robin
    RoundRobin,
    /// Least connections
    LeastConnections,
    /// Weighted round robin
    WeightedRoundRobin,
    /// Random selection
    Random,
    /// Consistent hashing
    ConsistentHash,
}

/// Health check configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HealthCheckConfig {
    /// Health check interval in seconds
    pub interval_seconds: u64,
    /// Health check timeout in seconds
    pub timeout_seconds: u64,
    /// Unhealthy threshold
    pub unhealthy_threshold: u32,
    /// Healthy threshold
    pub healthy_threshold: u32,
}

/// Timeout configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeoutConfig {
    /// Default request timeout in milliseconds
    pub default_timeout_ms: u64,
    /// Connection timeout in milliseconds
    pub connection_timeout_ms: u64,
    /// Read timeout in milliseconds
    pub read_timeout_ms: u64,
    /// Service-specific timeouts
    pub service_timeouts: HashMap<String, u64>,
}

/// Cross-service message
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceMessage {
    /// Message identifier
    pub message_id: String,
    /// Message type
    pub message_type: ServiceMessageType,
    /// Source service
    pub source_service: String,
    /// Target service
    pub target_service: String,
    /// Message payload
    pub payload: serde_json::Value,
    /// Message headers
    pub headers: HashMap<String, String>,
    /// Message timestamp
    pub timestamp: DateTime<Utc>,
    /// Correlation ID for tracing
    pub correlation_id: Option<String>,
    /// Reply-to address for responses
    pub reply_to: Option<String>,
    /// Message expiration time
    pub expires_at: Option<DateTime<Utc>>,
}

/// Service message types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceMessageType {
    /// Request for clinical assessment
    ClinicalAssessmentRequest,
    /// Response from clinical assessment
    ClinicalAssessmentResponse,
    /// Medication validation request
    MedicationValidationRequest,
    /// Medication validation response
    MedicationValidationResponse,
    /// Patient data query
    PatientDataQuery,
    /// Patient data response
    PatientDataResponse,
    /// FHIR resource operation
    FhirResourceOperation,
    /// Workflow coordination message
    WorkflowCoordination,
    /// Notification message
    NotificationMessage,
    /// Health check message
    HealthCheck,
    /// Service discovery message
    ServiceDiscovery,
    /// Error response
    ErrorResponse,
}

/// Message routing entry
#[derive(Debug, Clone)]
struct RoutingEntry {
    /// Service endpoint
    pub endpoint: ServiceEndpoint,
    /// Current health status
    pub health_status: ServiceHealthStatus,
    /// Last health check time
    pub last_health_check: DateTime<Utc>,
    /// Active connections count
    pub active_connections: u32,
    /// Circuit breaker state
    pub circuit_state: CircuitBreakerState,
    /// Request statistics
    pub stats: ServiceStats,
}

/// Service health status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ServiceHealthStatus {
    /// Service is healthy
    Healthy,
    /// Service is unhealthy
    Unhealthy,
    /// Service health unknown
    Unknown,
    /// Service is degraded
    Degraded,
}

/// Circuit breaker state
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CircuitBreakerState {
    /// Circuit is closed (normal operation)
    Closed,
    /// Circuit is open (failing fast)
    Open,
    /// Circuit is half-open (testing recovery)
    HalfOpen,
}

/// Service request statistics
#[derive(Debug, Clone, Default)]
pub struct ServiceStats {
    /// Total requests sent
    pub total_requests: u64,
    /// Successful requests
    pub successful_requests: u64,
    /// Failed requests
    pub failed_requests: u64,
    /// Average response time in milliseconds
    pub avg_response_time_ms: u64,
    /// Last request time
    pub last_request_time: Option<DateTime<Utc>>,
}

/// Message router for cross-service communication
pub struct MessageRouter {
    /// Configuration
    config: MessageRouterConfig,
    /// Service registry
    services: Arc<RwLock<HashMap<String, RoutingEntry>>>,
    /// HTTP client for requests
    client: reqwest::Client,
    /// Event publisher for tracing
    event_publisher: Option<Arc<EventPublisher>>,
    /// Pending requests tracking
    pending_requests: Arc<RwLock<HashMap<String, oneshot::Sender<ServiceMessage>>>>,
    /// Message queue for async processing
    message_queue: Arc<RwLock<Vec<ServiceMessage>>>,
    /// Router metrics
    metrics: Arc<RouterMetrics>,
}

/// Router metrics
#[derive(Debug, Default)]
pub struct RouterMetrics {
    /// Total messages routed
    pub total_messages: std::sync::atomic::AtomicU64,
    /// Successful deliveries
    pub successful_deliveries: std::sync::atomic::AtomicU64,
    /// Failed deliveries
    pub failed_deliveries: std::sync::atomic::AtomicU64,
    /// Messages retried
    pub retried_messages: std::sync::atomic::AtomicU64,
    /// Circuit breaker activations
    pub circuit_breaker_trips: std::sync::atomic::AtomicU64,
    /// Average routing time
    pub avg_routing_time_ms: std::sync::atomic::AtomicU64,
}

impl MessageRouter {
    /// Create new message router
    pub fn new(config: MessageRouterConfig) -> ProtocolResult<Self> {
        let client = reqwest::Client::builder()
            .timeout(std::time::Duration::from_millis(config.timeouts.default_timeout_ms))
            .connect_timeout(std::time::Duration::from_millis(config.timeouts.connection_timeout_ms))
            .build()
            .map_err(|e| ProtocolEngineError::IntegrationError(e.to_string()))?;

        let mut services = HashMap::new();

        // Initialize static service endpoints
        for (service_name, endpoint) in &config.service_discovery.static_endpoints {
            let routing_entry = RoutingEntry {
                endpoint: endpoint.clone(),
                health_status: ServiceHealthStatus::Unknown,
                last_health_check: Utc::now(),
                active_connections: 0,
                circuit_state: CircuitBreakerState::Closed,
                stats: ServiceStats::default(),
            };
            services.insert(service_name.clone(), routing_entry);
        }

        Ok(Self {
            config,
            services: Arc::new(RwLock::new(services)),
            client,
            event_publisher: None,
            pending_requests: Arc::new(RwLock::new(HashMap::new())),
            message_queue: Arc::new(RwLock::new(Vec::new())),
            metrics: Arc::new(RouterMetrics::default()),
        })
    }

    /// Set event publisher for message tracing
    pub fn set_event_publisher(&mut self, publisher: Arc<EventPublisher>) {
        self.event_publisher = Some(publisher);
    }

    /// Start message router background tasks
    pub async fn start(&self) -> ProtocolResult<()> {
        // Start service health monitoring
        let router = self.clone();
        tokio::spawn(async move {
            router.health_monitoring_task().await;
        });

        // Start message processing
        let router = self.clone();
        tokio::spawn(async move {
            router.message_processing_task().await;
        });

        // Start service discovery
        let router = self.clone();
        tokio::spawn(async move {
            router.service_discovery_task().await;
        });

        Ok(())
    }

    /// Send message to service
    pub async fn send_message(&self, message: ServiceMessage) -> ProtocolResult<ServiceMessage> {
        let start_time = std::time::Instant::now();
        
        self.metrics.total_messages.fetch_add(1, std::sync::atomic::Ordering::Relaxed);

        // Trace message if enabled
        if self.config.enable_tracing {
            self.trace_message(&message, "routing_started").await;
        }

        // Find target service
        let routing_entry = {
            let services = self.services.read().await;
            services.get(&message.target_service)
                .cloned()
                .ok_or_else(|| ProtocolEngineError::IntegrationError(
                    format!("Service not found: {}", message.target_service)
                ))?
        };

        // Check circuit breaker
        if matches!(routing_entry.circuit_state, CircuitBreakerState::Open) {
            return Err(ProtocolEngineError::IntegrationError(
                format!("Circuit breaker open for service: {}", message.target_service)
            ));
        }

        // Route message with retry policy
        let result = self.route_with_retry(&message, &routing_entry).await;

        // Update metrics
        let routing_time = start_time.elapsed().as_millis() as u64;
        self.update_routing_time_metric(routing_time);

        match &result {
            Ok(_) => {
                self.metrics.successful_deliveries.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                if self.config.enable_tracing {
                    self.trace_message(&message, "routing_completed").await;
                }
            },
            Err(_) => {
                self.metrics.failed_deliveries.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                if self.config.enable_tracing {
                    self.trace_message(&message, "routing_failed").await;
                }
            }
        }

        result
    }

    /// Send clinical assessment request
    pub async fn send_clinical_assessment_request(
        &self,
        request: CaeEvaluationRequest,
    ) -> ProtocolResult<CaeEvaluationResponse> {
        let message = ServiceMessage {
            message_id: Uuid::new_v4().to_string(),
            message_type: ServiceMessageType::ClinicalAssessmentRequest,
            source_service: "protocol-engine".to_string(),
            target_service: "clinical-assessment-engine".to_string(),
            payload: serde_json::to_value(request)?,
            headers: HashMap::new(),
            timestamp: Utc::now(),
            correlation_id: Some(Uuid::new_v4().to_string()),
            reply_to: None,
            expires_at: Some(Utc::now() + Duration::minutes(5)),
        };

        let response = self.send_message(message).await?;
        
        if matches!(response.message_type, ServiceMessageType::ClinicalAssessmentResponse) {
            let cae_response: CaeEvaluationResponse = serde_json::from_value(response.payload)?;
            Ok(cae_response)
        } else {
            Err(ProtocolEngineError::IntegrationError(
                "Invalid response type from clinical assessment service".to_string()
            ))
        }
    }

    /// Send medication validation request
    pub async fn send_medication_validation_request(
        &self,
        patient_id: &str,
        medication_request: &serde_json::Value,
    ) -> ProtocolResult<serde_json::Value> {
        let payload = serde_json::json!({
            "patient_id": patient_id,
            "medication_request": medication_request
        });

        let message = ServiceMessage {
            message_id: Uuid::new_v4().to_string(),
            message_type: ServiceMessageType::MedicationValidationRequest,
            source_service: "protocol-engine".to_string(),
            target_service: "medication-service".to_string(),
            payload,
            headers: HashMap::new(),
            timestamp: Utc::now(),
            correlation_id: Some(Uuid::new_v4().to_string()),
            reply_to: None,
            expires_at: Some(Utc::now() + Duration::minutes(2)),
        };

        let response = self.send_message(message).await?;
        Ok(response.payload)
    }

    /// Query patient data
    pub async fn query_patient_data(
        &self,
        patient_id: &str,
        data_types: &[String],
    ) -> ProtocolResult<serde_json::Value> {
        let payload = serde_json::json!({
            "patient_id": patient_id,
            "data_types": data_types
        });

        let message = ServiceMessage {
            message_id: Uuid::new_v4().to_string(),
            message_type: ServiceMessageType::PatientDataQuery,
            source_service: "protocol-engine".to_string(),
            target_service: "patient-service".to_string(),
            payload,
            headers: HashMap::new(),
            timestamp: Utc::now(),
            correlation_id: Some(Uuid::new_v4().to_string()),
            reply_to: None,
            expires_at: Some(Utc::now() + Duration::minutes(1)),
        };

        let response = self.send_message(message).await?;
        Ok(response.payload)
    }

    /// Route message with retry logic
    async fn route_with_retry(
        &self,
        message: &ServiceMessage,
        routing_entry: &RoutingEntry,
    ) -> ProtocolResult<ServiceMessage> {
        let mut retries = 0;
        let mut last_error = None;

        loop {
            match self.route_message_once(message, routing_entry).await {
                Ok(response) => {
                    if retries > 0 {
                        self.metrics.retried_messages.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    }
                    return Ok(response);
                },
                Err(e) => {
                    last_error = Some(e.clone());
                    
                    if retries >= self.config.retry_policy.max_retries {
                        break;
                    }

                    if !self.is_retryable_error(&e) {
                        break;
                    }

                    // Calculate retry delay
                    let delay = self.calculate_retry_delay(retries);
                    sleep(std::time::Duration::from_millis(delay)).await;
                    
                    retries += 1;
                }
            }
        }

        // Update circuit breaker on persistent failures
        if retries >= self.config.retry_policy.max_retries {
            self.update_circuit_breaker(&message.target_service, false).await;
        }

        Err(last_error.unwrap_or_else(|| ProtocolEngineError::IntegrationError(
            "Unknown routing error".to_string()
        )))
    }

    /// Route message once (single attempt)
    async fn route_message_once(
        &self,
        message: &ServiceMessage,
        routing_entry: &RoutingEntry,
    ) -> ProtocolResult<ServiceMessage> {
        let endpoint_url = format!("{}/api/v1/messages", routing_entry.endpoint.base_url);
        
        let mut request_builder = self.client.post(&endpoint_url)
            .json(message);

        // Add authentication if configured
        if let Some(auth_config) = &routing_entry.endpoint.auth_config {
            request_builder = self.add_authentication(request_builder, auth_config)?;
        }

        // Add correlation ID for tracing
        if let Some(correlation_id) = &message.correlation_id {
            request_builder = request_builder.header("X-Correlation-ID", correlation_id);
        }

        // Calculate timeout
        let service_timeout = self.config.timeouts.service_timeouts
            .get(&message.target_service)
            .copied()
            .unwrap_or(self.config.timeouts.default_timeout_ms);

        let response = timeout(
            std::time::Duration::from_millis(service_timeout),
            request_builder.send()
        ).await
        .map_err(|_| ProtocolEngineError::IntegrationError("Request timeout".to_string()))?
        .map_err(|e| ProtocolEngineError::IntegrationError(e.to_string()))?;

        if !response.status().is_success() {
            return Err(ProtocolEngineError::IntegrationError(
                format!("Service returned status: {}", response.status())
            ));
        }

        let service_response: ServiceMessage = response
            .json()
            .await
            .map_err(|e| ProtocolEngineError::IntegrationError(e.to_string()))?;

        // Update service statistics
        self.update_service_stats(&message.target_service, true, 0).await;

        Ok(service_response)
    }

    /// Add authentication to request
    fn add_authentication(
        &self,
        mut request_builder: reqwest::RequestBuilder,
        auth_config: &ServiceAuthConfig,
    ) -> ProtocolResult<reqwest::RequestBuilder> {
        match &auth_config.method {
            ServiceAuthMethod::None => Ok(request_builder),
            ServiceAuthMethod::Bearer => {
                if let Some(token) = &auth_config.credentials.token {
                    Ok(request_builder.bearer_auth(token))
                } else {
                    Err(ProtocolEngineError::IntegrationError(
                        "Bearer token not configured".to_string()
                    ))
                }
            },
            ServiceAuthMethod::ApiKey => {
                if let Some(token) = &auth_config.credentials.token {
                    Ok(request_builder.header("X-API-Key", token))
                } else {
                    Err(ProtocolEngineError::IntegrationError(
                        "API key not configured".to_string()
                    ))
                }
            },
            ServiceAuthMethod::JWT => {
                if let Some(token) = &auth_config.credentials.token {
                    Ok(request_builder.header("Authorization", format!("JWT {}", token)))
                } else {
                    Err(ProtocolEngineError::IntegrationError(
                        "JWT token not configured".to_string()
                    ))
                }
            },
            _ => Err(ProtocolEngineError::IntegrationError(
                format!("Unsupported authentication method: {:?}", auth_config.method)
            )),
        }
    }

    /// Check if error is retryable
    fn is_retryable_error(&self, error: &ProtocolEngineError) -> bool {
        match error {
            ProtocolEngineError::IntegrationError(msg) => {
                msg.contains("timeout") || 
                msg.contains("connection") ||
                msg.contains("503") || 
                msg.contains("502") ||
                msg.contains("500")
            },
            _ => false,
        }
    }

    /// Calculate retry delay based on backoff strategy
    fn calculate_retry_delay(&self, retry_count: u32) -> u64 {
        let base_delay = self.config.retry_policy.base_delay_ms;
        let max_delay = self.config.retry_policy.max_delay_ms;

        let delay = match self.config.retry_policy.backoff_strategy {
            BackoffStrategy::Fixed => base_delay,
            BackoffStrategy::Linear => base_delay * (retry_count as u64 + 1),
            BackoffStrategy::Exponential => base_delay * 2_u64.pow(retry_count),
            BackoffStrategy::ExponentialJitter => {
                let exponential_delay = base_delay * 2_u64.pow(retry_count);
                let jitter = fastrand::u64(0..=exponential_delay / 4);
                exponential_delay + jitter
            },
        };

        delay.min(max_delay)
    }

    /// Update circuit breaker state
    async fn update_circuit_breaker(&self, service_name: &str, success: bool) {
        if !self.config.circuit_breaker.enabled {
            return;
        }

        let mut services = self.services.write().await;
        if let Some(entry) = services.get_mut(service_name) {
            match entry.circuit_state {
                CircuitBreakerState::Closed => {
                    if !success && entry.stats.failed_requests >= self.config.circuit_breaker.failure_threshold as u64 {
                        entry.circuit_state = CircuitBreakerState::Open;
                        self.metrics.circuit_breaker_trips.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    }
                },
                CircuitBreakerState::Open => {
                    // Circuit remains open until timeout
                },
                CircuitBreakerState::HalfOpen => {
                    if success {
                        if entry.stats.successful_requests >= self.config.circuit_breaker.success_threshold as u64 {
                            entry.circuit_state = CircuitBreakerState::Closed;
                        }
                    } else {
                        entry.circuit_state = CircuitBreakerState::Open;
                    }
                },
            }
        }
    }

    /// Update service statistics
    async fn update_service_stats(&self, service_name: &str, success: bool, response_time_ms: u64) {
        let mut services = self.services.write().await;
        if let Some(entry) = services.get_mut(service_name) {
            entry.stats.total_requests += 1;
            if success {
                entry.stats.successful_requests += 1;
            } else {
                entry.stats.failed_requests += 1;
            }
            entry.stats.last_request_time = Some(Utc::now());
            
            // Update average response time
            if response_time_ms > 0 {
                let total_requests = entry.stats.total_requests;
                let current_avg = entry.stats.avg_response_time_ms;
                entry.stats.avg_response_time_ms = 
                    (current_avg * (total_requests - 1) + response_time_ms) / total_requests;
            }
        }
    }

    /// Update routing time metric
    fn update_routing_time_metric(&self, routing_time_ms: u64) {
        let current_avg = self.metrics.avg_routing_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_messages = self.metrics.total_messages.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_messages > 1 {
            let new_avg = (current_avg * (total_messages - 1) + routing_time_ms) / total_messages;
            self.metrics.avg_routing_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.metrics.avg_routing_time_ms.store(routing_time_ms, std::sync::atomic::Ordering::Relaxed);
        }
    }

    /// Trace message for debugging
    async fn trace_message(&self, message: &ServiceMessage, event: &str) {
        if let Some(publisher) = &self.event_publisher {
            let trace_data = serde_json::json!({
                "message_id": message.message_id,
                "message_type": message.message_type,
                "source_service": message.source_service,
                "target_service": message.target_service,
                "correlation_id": message.correlation_id,
                "event": event
            });

            let trace_event = ProtocolEvent {
                event_id: Uuid::new_v4().to_string(),
                event_type: ProtocolEventType::MessageTrace,
                patient_id: "".to_string(),
                protocol_id: "".to_string(),
                tenant_id: "".to_string(),
                source: "message-router".to_string(),
                version: "1.0".to_string(),
                timestamp: Utc::now(),
                correlation_id: message.correlation_id.clone(),
                causation_id: None,
                payload: crate::protocol::event_publisher::ProtocolEventPayload {
                    data: trace_data,
                    previous_state: None,
                    current_state: None,
                    error: None,
                },
                metadata: crate::protocol::event_publisher::EventMetadata {
                    service_version: "1.0.0".to_string(),
                    environment: "production".to_string(),
                    region: Some("us-east-1".to_string()),
                    trace_id: message.correlation_id.clone(),
                    span_id: Some(message.message_id.clone()),
                    initiated_by: Some(message.source_service.clone()),
                    custom: std::collections::HashMap::new(),
                },
            };

            let _ = publisher.publish_event(trace_event).await;
        }
    }

    /// Health monitoring background task
    async fn health_monitoring_task(&self) {
        let mut health_check_interval = tokio::time::interval(
            std::time::Duration::from_secs(self.config.service_discovery.health_check_interval_seconds)
        );

        loop {
            health_check_interval.tick().await;
            self.perform_health_checks().await;
        }
    }

    /// Perform health checks on all services
    async fn perform_health_checks(&self) {
        let service_names: Vec<String> = {
            let services = self.services.read().await;
            services.keys().cloned().collect()
        };

        for service_name in service_names {
            self.check_service_health(&service_name).await;
        }
    }

    /// Check health of specific service
    async fn check_service_health(&self, service_name: &str) {
        let (endpoint, health_endpoint) = {
            let services = self.services.read().await;
            if let Some(entry) = services.get(service_name) {
                (entry.endpoint.base_url.clone(), entry.endpoint.health_endpoint.clone())
            } else {
                return;
            }
        };

        let health_url = format!("{}{}", endpoint, health_endpoint);
        let health_check_timeout = std::time::Duration::from_secs(
            self.config.service_discovery.health_check_timeout_seconds
        );

        let health_status = match timeout(health_check_timeout, self.client.get(&health_url).send()).await {
            Ok(Ok(response)) if response.status().is_success() => ServiceHealthStatus::Healthy,
            Ok(Ok(_)) => ServiceHealthStatus::Degraded,
            Ok(Err(_)) => ServiceHealthStatus::Unhealthy,
            Err(_) => ServiceHealthStatus::Unhealthy,
        };

        // Update service health status
        {
            let mut services = self.services.write().await;
            if let Some(entry) = services.get_mut(service_name) {
                entry.health_status = health_status;
                entry.last_health_check = Utc::now();
                
                // Update circuit breaker for unhealthy services
                if matches!(entry.health_status, ServiceHealthStatus::Unhealthy) {
                    entry.circuit_state = CircuitBreakerState::Open;
                }
            }
        }
    }

    /// Message processing background task
    async fn message_processing_task(&self) {
        let mut processing_interval = tokio::time::interval(
            std::time::Duration::from_millis(100)
        );

        loop {
            processing_interval.tick().await;
            self.process_queued_messages().await;
        }
    }

    /// Process queued messages
    async fn process_queued_messages(&self) {
        let messages = {
            let mut queue = self.message_queue.write().await;
            let current_messages = queue.clone();
            queue.clear();
            current_messages
        };

        for message in messages {
            if let Err(e) = self.send_message(message).await {
                log::error!("Failed to process queued message: {}", e);
            }
        }
    }

    /// Service discovery background task
    async fn service_discovery_task(&self) {
        let mut discovery_interval = tokio::time::interval(
            std::time::Duration::from_secs(30)
        );

        loop {
            discovery_interval.tick().await;
            self.discover_services().await;
        }
    }

    /// Discover services based on configured method
    async fn discover_services(&self) {
        match self.config.service_discovery.method {
            ServiceDiscoveryMethod::Static => {
                // Static configuration - nothing to discover
            },
            ServiceDiscoveryMethod::Consul => {
                // TODO: Implement Consul service discovery
            },
            ServiceDiscoveryMethod::Kubernetes => {
                // TODO: Implement Kubernetes service discovery
            },
            _ => {
                // Other discovery methods not implemented
            }
        }
    }

    /// Get router metrics
    pub fn get_metrics(&self) -> RouterMetricsSnapshot {
        RouterMetricsSnapshot {
            total_messages: self.metrics.total_messages.load(std::sync::atomic::Ordering::Relaxed),
            successful_deliveries: self.metrics.successful_deliveries.load(std::sync::atomic::Ordering::Relaxed),
            failed_deliveries: self.metrics.failed_deliveries.load(std::sync::atomic::Ordering::Relaxed),
            retried_messages: self.metrics.retried_messages.load(std::sync::atomic::Ordering::Relaxed),
            circuit_breaker_trips: self.metrics.circuit_breaker_trips.load(std::sync::atomic::Ordering::Relaxed),
            avg_routing_time_ms: self.metrics.avg_routing_time_ms.load(std::sync::atomic::Ordering::Relaxed),
        }
    }

    /// Get service health status
    pub async fn get_service_health(&self, service_name: &str) -> Option<ServiceHealthStatus> {
        let services = self.services.read().await;
        services.get(service_name).map(|entry| entry.health_status.clone())
    }
}

/// Router metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouterMetricsSnapshot {
    pub total_messages: u64,
    pub successful_deliveries: u64,
    pub failed_deliveries: u64,
    pub retried_messages: u64,
    pub circuit_breaker_trips: u64,
    pub avg_routing_time_ms: u64,
}

impl Clone for MessageRouter {
    fn clone(&self) -> Self {
        Self {
            config: self.config.clone(),
            services: Arc::clone(&self.services),
            client: self.client.clone(),
            event_publisher: self.event_publisher.clone(),
            pending_requests: Arc::clone(&self.pending_requests),
            message_queue: Arc::clone(&self.message_queue),
            metrics: Arc::clone(&self.metrics),
        }
    }
}

impl Default for MessageRouterConfig {
    fn default() -> Self {
        Self {
            service_discovery: ServiceDiscoveryConfig {
                method: ServiceDiscoveryMethod::Static,
                registry_endpoint: None,
                static_endpoints: HashMap::new(),
                health_check_interval_seconds: 30,
                health_check_timeout_seconds: 5,
            },
            circuit_breaker: CircuitBreakerConfig {
                enabled: true,
                failure_threshold: 5,
                success_threshold: 3,
                timeout_seconds: 60,
                window_size: 10,
            },
            retry_policy: RetryPolicyConfig {
                max_retries: 3,
                base_delay_ms: 1000,
                max_delay_ms: 30000,
                backoff_strategy: BackoffStrategy::ExponentialJitter,
                retryable_errors: vec![
                    RetryableErrorType::NetworkTimeout,
                    RetryableErrorType::ConnectionError,
                    RetryableErrorType::ServiceUnavailable,
                    RetryableErrorType::InternalServerError,
                ],
            },
            load_balancer: LoadBalancerConfig {
                strategy: LoadBalancingStrategy::RoundRobin,
                health_check: HealthCheckConfig {
                    interval_seconds: 30,
                    timeout_seconds: 5,
                    unhealthy_threshold: 3,
                    healthy_threshold: 2,
                },
                sticky_sessions: false,
            },
            timeouts: TimeoutConfig {
                default_timeout_ms: 30000,
                connection_timeout_ms: 5000,
                read_timeout_ms: 30000,
                service_timeouts: HashMap::new(),
            },
            enable_tracing: true,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_message_creation() {
        let message = ServiceMessage {
            message_id: Uuid::new_v4().to_string(),
            message_type: ServiceMessageType::ClinicalAssessmentRequest,
            source_service: "protocol-engine".to_string(),
            target_service: "cae-service".to_string(),
            payload: serde_json::json!({"test": "data"}),
            headers: HashMap::new(),
            timestamp: Utc::now(),
            correlation_id: Some(Uuid::new_v4().to_string()),
            reply_to: None,
            expires_at: None,
        };

        assert!(!message.message_id.is_empty());
        assert_eq!(message.source_service, "protocol-engine");
    }

    #[tokio::test]
    async fn test_message_router_creation() {
        let config = MessageRouterConfig::default();
        let router = MessageRouter::new(config);
        assert!(router.is_ok());
    }

    #[test]
    fn test_retry_delay_calculation() {
        let config = MessageRouterConfig::default();
        let router = MessageRouter::new(config).unwrap();
        
        let delay_0 = router.calculate_retry_delay(0);
        let delay_1 = router.calculate_retry_delay(1);
        let delay_2 = router.calculate_retry_delay(2);
        
        // Exponential jitter should increase delays
        assert!(delay_1 >= delay_0);
        assert!(delay_2 >= delay_1);
    }
}