//! Event Publishing Infrastructure
//!
//! This module provides event publishing capabilities for the Protocol Engine,
//! enabling integration with Kafka and other messaging systems for real-time
//! clinical event distribution and system-wide coordination.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc, Mutex};
use tokio::time::{interval, timeout};
use uuid::Uuid;
use rdkafka::{
    config::ClientConfig,
    producer::{FutureProducer, FutureRecord},
    message::OwnedHeaders,
};

use crate::protocol::{
    types::*,
    error::*,
    cae_integration::{CaeEvent, CaeEventType},
    state_machine::{ProtocolStateMachine, StateTransition},
};

/// Event publisher configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventPublisherConfig {
    /// Kafka broker configuration
    pub kafka_config: KafkaConfig,
    /// Event buffering configuration
    pub buffer_config: BufferConfig,
    /// Schema validation configuration
    pub schema_config: SchemaConfig,
    /// Publishing reliability settings
    pub reliability_config: ReliabilityConfig,
    /// Enable event publishing
    pub enabled: bool,
}

/// Kafka configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KafkaConfig {
    /// Kafka bootstrap servers
    pub bootstrap_servers: String,
    /// Security protocol (PLAINTEXT, SASL_SSL, etc.)
    pub security_protocol: String,
    /// SASL mechanism
    pub sasl_mechanism: Option<String>,
    /// SASL username
    pub sasl_username: Option<String>,
    /// SASL password
    pub sasl_password: Option<String>,
    /// Topic prefix for all protocol events
    pub topic_prefix: String,
    /// Default partitions for topics
    pub default_partitions: i32,
    /// Replication factor
    pub replication_factor: i16,
}

/// Event buffering configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BufferConfig {
    /// Maximum buffer size per topic
    pub max_buffer_size: usize,
    /// Batch size for publishing
    pub batch_size: usize,
    /// Maximum batch wait time in milliseconds
    pub max_batch_wait_ms: u64,
    /// Buffer flush interval in milliseconds
    pub flush_interval_ms: u64,
    /// Enable compression
    pub compression_enabled: bool,
    /// Compression type (gzip, snappy, lz4)
    pub compression_type: String,
}

/// Schema validation configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaConfig {
    /// Enable schema validation
    pub validation_enabled: bool,
    /// Schema registry URL
    pub schema_registry_url: Option<String>,
    /// Schema registry authentication
    pub registry_auth: Option<SchemaRegistryAuth>,
    /// Schema version strategy
    pub version_strategy: SchemaVersionStrategy,
}

/// Schema registry authentication
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SchemaRegistryAuth {
    pub username: String,
    pub password: String,
}

/// Schema version strategy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SchemaVersionStrategy {
    /// Always use latest schema version
    Latest,
    /// Use specific schema version
    Specific(String),
    /// Use backward compatible versions
    BackwardCompatible,
}

/// Reliability configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReliabilityConfig {
    /// Number of retries for failed publishes
    pub max_retries: u32,
    /// Retry delay in milliseconds
    pub retry_delay_ms: u64,
    /// Enable dead letter queue
    pub dead_letter_enabled: bool,
    /// Dead letter topic suffix
    pub dead_letter_suffix: String,
    /// Acknowledgment timeout in milliseconds
    pub ack_timeout_ms: u64,
}

/// Protocol event for publishing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEvent {
    /// Event identifier
    pub event_id: String,
    /// Event type
    pub event_type: ProtocolEventType,
    /// Patient identifier
    pub patient_id: String,
    /// Protocol identifier
    pub protocol_id: String,
    /// Tenant/organization identifier
    pub tenant_id: String,
    /// Event source (which service generated it)
    pub source: String,
    /// Event version for schema evolution
    pub version: String,
    /// Event timestamp
    pub timestamp: DateTime<Utc>,
    /// Event correlation ID for tracing
    pub correlation_id: Option<String>,
    /// Event causation ID (what caused this event)
    pub causation_id: Option<String>,
    /// Event payload
    pub payload: ProtocolEventPayload,
    /// Event metadata
    pub metadata: EventMetadata,
}

/// Protocol event types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ProtocolEventType {
    /// Protocol evaluation events
    ProtocolEvaluationStarted,
    ProtocolEvaluationCompleted,
    ProtocolEvaluationFailed,
    
    /// State machine events
    StateTransitionInitiated,
    StateTransitionCompleted,
    StateTransitionFailed,
    StateTimeoutOccurred,
    
    /// Rule evaluation events
    RuleEvaluationStarted,
    RuleEvaluationCompleted,
    RuleViolationDetected,
    
    /// Temporal constraint events
    TemporalConstraintViolated,
    TemporalConstraintSatisfied,
    TemporalWindowOpened,
    TemporalWindowClosed,
    
    /// Clinical events
    ClinicalAlertTriggered,
    ClinicalRecommendationGenerated,
    RiskLevelChanged,
    
    /// Administrative events
    ProtocolActivated,
    ProtocolDeactivated,
    ProtocolModified,
    ProtocolApprovalRequired,
    
    /// Integration events
    CaeIntegrationRequested,
    CaeIntegrationCompleted,
    ExternalServiceIntegration,
    
    /// Audit events
    AuditLogGenerated,
    ComplianceCheckPerformed,
}

/// Protocol event payload
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEventPayload {
    /// Type-specific data
    pub data: serde_json::Value,
    /// Previous state (for transition events)
    pub previous_state: Option<serde_json::Value>,
    /// Current state
    pub current_state: Option<serde_json::Value>,
    /// Error details (for failure events)
    pub error: Option<ProtocolEventError>,
}

/// Error details in events
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEventError {
    pub error_code: String,
    pub error_message: String,
    pub error_details: Option<serde_json::Value>,
    pub stack_trace: Option<String>,
}

/// Event metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EventMetadata {
    /// Publishing service version
    pub service_version: String,
    /// Environment (prod, staging, dev)
    pub environment: String,
    /// Data center/region
    pub region: Option<String>,
    /// Request tracing information
    pub trace_id: Option<String>,
    /// Span ID for distributed tracing
    pub span_id: Option<String>,
    /// User/service that initiated the action
    pub initiated_by: Option<String>,
    /// Additional custom metadata
    pub custom: HashMap<String, String>,
}

/// Event publishing statistics
#[derive(Debug, Default)]
pub struct PublishingStats {
    /// Total events published
    pub total_published: std::sync::atomic::AtomicU64,
    /// Events failed to publish
    pub failed_publishes: std::sync::atomic::AtomicU64,
    /// Events currently buffered
    pub buffered_events: std::sync::atomic::AtomicU64,
    /// Average publish latency
    pub avg_publish_latency_ms: std::sync::atomic::AtomicU64,
    /// Events sent to dead letter queue
    pub dead_letter_events: std::sync::atomic::AtomicU64,
    /// Schema validation failures
    pub schema_validation_failures: std::sync::atomic::AtomicU64,
}

/// Event publisher engine
pub struct EventPublisher {
    /// Configuration
    config: EventPublisherConfig,
    /// Kafka producer
    producer: FutureProducer,
    /// Event buffers by topic
    buffers: Arc<RwLock<HashMap<String, Vec<ProtocolEvent>>>>,
    /// Publishing statistics
    stats: Arc<PublishingStats>,
    /// Topic management
    topics: Arc<RwLock<HashMap<String, TopicMetadata>>>,
    /// Schema validator
    schema_validator: Option<SchemaValidator>,
    /// Event batch processor
    batch_processor: Arc<Mutex<BatchProcessor>>,
}

/// Topic metadata
#[derive(Debug, Clone)]
pub struct TopicMetadata {
    pub name: String,
    pub partitions: i32,
    pub replication_factor: i16,
    pub created_at: DateTime<Utc>,
    pub last_published: Option<DateTime<Utc>>,
}

/// Schema validator for events
pub struct SchemaValidator {
    /// Schema cache
    schemas: HashMap<String, serde_json::Value>,
    /// Schema registry client
    registry_client: Option<reqwest::Client>,
}

/// Batch processor for efficient publishing
pub struct BatchProcessor {
    /// Pending batches by topic
    pending_batches: HashMap<String, Vec<ProtocolEvent>>,
    /// Last flush time by topic
    last_flush: HashMap<String, DateTime<Utc>>,
}

impl EventPublisher {
    /// Create new event publisher
    pub async fn new(config: EventPublisherConfig) -> ProtocolResult<Self> {
        if !config.enabled {
            return Err(ProtocolEngineError::IntegrationError(
                "Event publishing is disabled".to_string()
            ));
        }

        // Configure Kafka producer
        let mut kafka_config = ClientConfig::new();
        kafka_config.set("bootstrap.servers", &config.kafka_config.bootstrap_servers);
        kafka_config.set("security.protocol", &config.kafka_config.security_protocol);
        
        if let Some(mechanism) = &config.kafka_config.sasl_mechanism {
            kafka_config.set("sasl.mechanism", mechanism);
        }
        
        if let (Some(username), Some(password)) = 
            (&config.kafka_config.sasl_username, &config.kafka_config.sasl_password) {
            kafka_config.set("sasl.username", username);
            kafka_config.set("sasl.password", password);
        }

        // Production optimization settings
        kafka_config.set("message.timeout.ms", "60000");
        kafka_config.set("retry.backoff.ms", "1000");
        kafka_config.set("batch.num.messages", &config.buffer_config.batch_size.to_string());
        
        if config.buffer_config.compression_enabled {
            kafka_config.set("compression.type", &config.buffer_config.compression_type);
        }

        let producer: FutureProducer = kafka_config
            .create()
            .map_err(|e| ProtocolEngineError::IntegrationError(
                format!("Failed to create Kafka producer: {}", e)
            ))?;

        let schema_validator = if config.schema_config.validation_enabled {
            Some(SchemaValidator::new(&config.schema_config).await?)
        } else {
            None
        };

        Ok(Self {
            config,
            producer,
            buffers: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(PublishingStats::default()),
            topics: Arc::new(RwLock::new(HashMap::new())),
            schema_validator,
            batch_processor: Arc::new(Mutex::new(BatchProcessor {
                pending_batches: HashMap::new(),
                last_flush: HashMap::new(),
            })),
        })
    }

    /// Start the event publisher background tasks
    pub async fn start(&self) -> ProtocolResult<()> {
        // Start batch flushing task
        let publisher = self.clone();
        tokio::spawn(async move {
            publisher.batch_flush_task().await;
        });

        // Start buffer monitoring task
        let publisher = self.clone();
        tokio::spawn(async move {
            publisher.buffer_monitoring_task().await;
        });

        Ok(())
    }

    /// Publish a protocol event
    pub async fn publish_event(&self, event: ProtocolEvent) -> ProtocolResult<()> {
        if !self.config.enabled {
            return Ok(());
        }

        // Validate schema if enabled
        if let Some(validator) = &self.schema_validator {
            validator.validate_event(&event).await?;
        }

        let topic_name = self.get_topic_name(&event.event_type);
        
        // Ensure topic exists
        self.ensure_topic_exists(&topic_name).await?;

        // Add to batch processor
        {
            let mut processor = self.batch_processor.lock().await;
            processor.add_event(topic_name.clone(), event);
        }

        // Check if immediate flush is needed
        let should_flush = {
            let processor = self.batch_processor.lock().await;
            processor.should_flush(&topic_name, &self.config.buffer_config)
        };

        if should_flush {
            self.flush_topic_batch(&topic_name).await?;
        }

        Ok(())
    }

    /// Publish protocol evaluation started event
    pub async fn publish_protocol_evaluation_started(
        &self,
        patient_id: &str,
        protocol_id: &str,
        tenant_id: &str,
        evaluation_context: &serde_json::Value,
    ) -> ProtocolResult<()> {
        let event = ProtocolEvent {
            event_id: Uuid::new_v4().to_string(),
            event_type: ProtocolEventType::ProtocolEvaluationStarted,
            patient_id: patient_id.to_string(),
            protocol_id: protocol_id.to_string(),
            tenant_id: tenant_id.to_string(),
            source: "protocol-engine".to_string(),
            version: "1.0".to_string(),
            timestamp: Utc::now(),
            correlation_id: None,
            causation_id: None,
            payload: ProtocolEventPayload {
                data: evaluation_context.clone(),
                previous_state: None,
                current_state: None,
                error: None,
            },
            metadata: self.create_event_metadata(),
        };

        self.publish_event(event).await
    }

    /// Publish state transition event
    pub async fn publish_state_transition(
        &self,
        patient_id: &str,
        protocol_id: &str,
        tenant_id: &str,
        transition: &StateTransition,
        previous_state: &str,
        new_state: &str,
    ) -> ProtocolResult<()> {
        let event = ProtocolEvent {
            event_id: Uuid::new_v4().to_string(),
            event_type: ProtocolEventType::StateTransitionCompleted,
            patient_id: patient_id.to_string(),
            protocol_id: protocol_id.to_string(),
            tenant_id: tenant_id.to_string(),
            source: "protocol-engine".to_string(),
            version: "1.0".to_string(),
            timestamp: Utc::now(),
            correlation_id: None,
            causation_id: None,
            payload: ProtocolEventPayload {
                data: serde_json::to_value(transition).unwrap_or_default(),
                previous_state: Some(serde_json::Value::String(previous_state.to_string())),
                current_state: Some(serde_json::Value::String(new_state.to_string())),
                error: None,
            },
            metadata: self.create_event_metadata(),
        };

        self.publish_event(event).await
    }

    /// Publish clinical alert event
    pub async fn publish_clinical_alert(
        &self,
        patient_id: &str,
        protocol_id: &str,
        tenant_id: &str,
        alert_type: &str,
        severity: &str,
        message: &str,
        recommendations: &[String],
    ) -> ProtocolResult<()> {
        let alert_data = serde_json::json!({
            "alert_type": alert_type,
            "severity": severity,
            "message": message,
            "recommendations": recommendations,
            "alert_time": Utc::now().to_rfc3339(),
        });

        let event = ProtocolEvent {
            event_id: Uuid::new_v4().to_string(),
            event_type: ProtocolEventType::ClinicalAlertTriggered,
            patient_id: patient_id.to_string(),
            protocol_id: protocol_id.to_string(),
            tenant_id: tenant_id.to_string(),
            source: "protocol-engine".to_string(),
            version: "1.0".to_string(),
            timestamp: Utc::now(),
            correlation_id: None,
            causation_id: None,
            payload: ProtocolEventPayload {
                data: alert_data,
                previous_state: None,
                current_state: None,
                error: None,
            },
            metadata: self.create_event_metadata(),
        };

        self.publish_event(event).await
    }

    /// Get topic name for event type
    fn get_topic_name(&self, event_type: &ProtocolEventType) -> String {
        let suffix = match event_type {
            ProtocolEventType::ProtocolEvaluationStarted |
            ProtocolEventType::ProtocolEvaluationCompleted |
            ProtocolEventType::ProtocolEvaluationFailed => "protocol-evaluation",
            
            ProtocolEventType::StateTransitionInitiated |
            ProtocolEventType::StateTransitionCompleted |
            ProtocolEventType::StateTransitionFailed => "state-transition",
            
            ProtocolEventType::RuleEvaluationStarted |
            ProtocolEventType::RuleEvaluationCompleted |
            ProtocolEventType::RuleViolationDetected => "rule-evaluation",
            
            ProtocolEventType::TemporalConstraintViolated |
            ProtocolEventType::TemporalConstraintSatisfied => "temporal-constraint",
            
            ProtocolEventType::ClinicalAlertTriggered |
            ProtocolEventType::ClinicalRecommendationGenerated => "clinical-event",
            
            _ => "protocol-event",
        };

        format!("{}.{}", self.config.kafka_config.topic_prefix, suffix)
    }

    /// Ensure topic exists
    async fn ensure_topic_exists(&self, topic_name: &str) -> ProtocolResult<()> {
        let mut topics = self.topics.write().await;
        
        if !topics.contains_key(topic_name) {
            // In production, topics should be created by infrastructure
            // This is a placeholder for topic metadata tracking
            topics.insert(topic_name.to_string(), TopicMetadata {
                name: topic_name.to_string(),
                partitions: self.config.kafka_config.default_partitions,
                replication_factor: self.config.kafka_config.replication_factor,
                created_at: Utc::now(),
                last_published: None,
            });
        }

        Ok(())
    }

    /// Flush batch for specific topic
    async fn flush_topic_batch(&self, topic_name: &str) -> ProtocolResult<()> {
        let events_to_publish = {
            let mut processor = self.batch_processor.lock().await;
            processor.extract_batch(topic_name)
        };

        if events_to_publish.is_empty() {
            return Ok(());
        }

        for event in events_to_publish {
            self.publish_single_event(topic_name, event).await?;
        }

        Ok(())
    }

    /// Publish single event to Kafka
    async fn publish_single_event(&self, topic_name: &str, event: ProtocolEvent) -> ProtocolResult<()> {
        let start_time = std::time::Instant::now();
        
        let payload = serde_json::to_string(&event)
            .map_err(|e| ProtocolEngineError::SerializationError(e.to_string()))?;

        let mut headers = OwnedHeaders::new();
        headers = headers.insert("event-type", &event.event_type.to_string());
        headers = headers.insert("event-id", &event.event_id);
        headers = headers.insert("protocol-id", &event.protocol_id);
        headers = headers.insert("patient-id", &event.patient_id);
        headers = headers.insert("tenant-id", &event.tenant_id);

        let record = FutureRecord::to(topic_name)
            .key(&event.patient_id) // Use patient_id as partition key
            .payload(&payload)
            .headers(headers);

        let publish_result = timeout(
            std::time::Duration::from_millis(self.config.reliability_config.ack_timeout_ms),
            self.producer.send(record, std::time::Duration::from_millis(1000))
        ).await;

        match publish_result {
            Ok(Ok(_)) => {
                self.stats.total_published.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                
                // Update latency metric
                let latency = start_time.elapsed().as_millis() as u64;
                self.update_latency_metric(latency);
                
                // Update topic metadata
                {
                    let mut topics = self.topics.write().await;
                    if let Some(topic_meta) = topics.get_mut(topic_name) {
                        topic_meta.last_published = Some(Utc::now());
                    }
                }
                
                Ok(())
            },
            Ok(Err(e)) => {
                self.stats.failed_publishes.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                self.handle_publish_failure(topic_name, event, e).await
            },
            Err(_) => {
                self.stats.failed_publishes.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Err(ProtocolEngineError::IntegrationError(
                    "Publish timeout".to_string()
                ))
            },
        }
    }

    /// Handle publish failure with retries and dead letter queue
    async fn handle_publish_failure(
        &self,
        _topic_name: &str,
        event: ProtocolEvent,
        error: rdkafka::error::KafkaError,
    ) -> ProtocolResult<()> {
        // TODO: Implement retry logic and dead letter queue
        log::error!("Failed to publish event {}: {}", event.event_id, error);
        self.stats.dead_letter_events.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        Err(ProtocolEngineError::IntegrationError(
            format!("Failed to publish event: {}", error)
        ))
    }

    /// Update average latency metric
    fn update_latency_metric(&self, latency_ms: u64) {
        let current_avg = self.stats.avg_publish_latency_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_published = self.stats.total_published.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_published > 1 {
            let new_avg = (current_avg * (total_published - 1) + latency_ms) / total_published;
            self.stats.avg_publish_latency_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.stats.avg_publish_latency_ms.store(latency_ms, std::sync::atomic::Ordering::Relaxed);
        }
    }

    /// Create standard event metadata
    fn create_event_metadata(&self) -> EventMetadata {
        EventMetadata {
            service_version: "1.0.0".to_string(),
            environment: "production".to_string(),
            region: Some("us-east-1".to_string()),
            trace_id: None, // TODO: Extract from tracing context
            span_id: None,
            initiated_by: Some("protocol-engine".to_string()),
            custom: HashMap::new(),
        }
    }

    /// Batch flush background task
    async fn batch_flush_task(&self) {
        let mut flush_interval = interval(
            std::time::Duration::from_millis(self.config.buffer_config.flush_interval_ms)
        );

        loop {
            flush_interval.tick().await;
            
            let topic_names: Vec<String> = {
                let processor = self.batch_processor.lock().await;
                processor.pending_batches.keys().cloned().collect()
            };

            for topic_name in topic_names {
                if let Err(e) = self.flush_topic_batch(&topic_name).await {
                    log::error!("Failed to flush batch for topic {}: {}", topic_name, e);
                }
            }
        }
    }

    /// Buffer monitoring background task
    async fn buffer_monitoring_task(&self) {
        let mut monitor_interval = interval(std::time::Duration::from_secs(30));

        loop {
            monitor_interval.tick().await;
            
            let buffer_count = {
                let processor = self.batch_processor.lock().await;
                processor.pending_batches.values().map(|v| v.len()).sum::<usize>()
            };

            self.stats.buffered_events.store(
                buffer_count as u64,
                std::sync::atomic::Ordering::Relaxed
            );

            // Log metrics
            let stats = self.get_stats();
            log::info!(
                "Event Publisher Stats - Published: {}, Failed: {}, Buffered: {}, Avg Latency: {}ms",
                stats.total_published,
                stats.failed_publishes,
                stats.buffered_events,
                stats.avg_publish_latency_ms
            );
        }
    }

    /// Get publishing statistics
    pub fn get_stats(&self) -> PublishingStatsSnapshot {
        PublishingStatsSnapshot {
            total_published: self.stats.total_published.load(std::sync::atomic::Ordering::Relaxed),
            failed_publishes: self.stats.failed_publishes.load(std::sync::atomic::Ordering::Relaxed),
            buffered_events: self.stats.buffered_events.load(std::sync::atomic::Ordering::Relaxed),
            avg_publish_latency_ms: self.stats.avg_publish_latency_ms.load(std::sync::atomic::Ordering::Relaxed),
            dead_letter_events: self.stats.dead_letter_events.load(std::sync::atomic::Ordering::Relaxed),
            schema_validation_failures: self.stats.schema_validation_failures.load(std::sync::atomic::Ordering::Relaxed),
        }
    }
}

/// Publishing statistics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PublishingStatsSnapshot {
    pub total_published: u64,
    pub failed_publishes: u64,
    pub buffered_events: u64,
    pub avg_publish_latency_ms: u64,
    pub dead_letter_events: u64,
    pub schema_validation_failures: u64,
}

impl SchemaValidator {
    /// Create new schema validator
    async fn new(config: &SchemaConfig) -> ProtocolResult<Self> {
        let registry_client = if let Some(url) = &config.schema_registry_url {
            let client = reqwest::Client::new();
            Some(client)
        } else {
            None
        };

        Ok(Self {
            schemas: HashMap::new(),
            registry_client,
        })
    }

    /// Validate event against schema
    async fn validate_event(&self, _event: &ProtocolEvent) -> ProtocolResult<()> {
        // TODO: Implement schema validation
        Ok(())
    }
}

impl BatchProcessor {
    /// Add event to batch
    fn add_event(&mut self, topic: String, event: ProtocolEvent) {
        self.pending_batches.entry(topic.clone()).or_insert_with(Vec::new).push(event);
        self.last_flush.insert(topic, Utc::now());
    }

    /// Check if batch should be flushed
    fn should_flush(&self, topic: &str, config: &BufferConfig) -> bool {
        if let Some(events) = self.pending_batches.get(topic) {
            if events.len() >= config.batch_size {
                return true;
            }

            if let Some(last_flush) = self.last_flush.get(topic) {
                let elapsed = Utc::now().signed_duration_since(*last_flush);
                return elapsed.num_milliseconds() >= config.max_batch_wait_ms as i64;
            }
        }

        false
    }

    /// Extract batch for publishing
    fn extract_batch(&mut self, topic: &str) -> Vec<ProtocolEvent> {
        self.pending_batches.remove(topic).unwrap_or_default()
    }
}

impl Clone for EventPublisher {
    fn clone(&self) -> Self {
        Self {
            config: self.config.clone(),
            producer: self.producer.clone(),
            buffers: Arc::clone(&self.buffers),
            stats: Arc::clone(&self.stats),
            topics: Arc::clone(&self.topics),
            schema_validator: None, // Schema validator cannot be cloned easily
            batch_processor: Arc::clone(&self.batch_processor),
        }
    }
}

impl ToString for ProtocolEventType {
    fn to_string(&self) -> String {
        match self {
            ProtocolEventType::ProtocolEvaluationStarted => "protocol.evaluation.started".to_string(),
            ProtocolEventType::ProtocolEvaluationCompleted => "protocol.evaluation.completed".to_string(),
            ProtocolEventType::ProtocolEvaluationFailed => "protocol.evaluation.failed".to_string(),
            ProtocolEventType::StateTransitionInitiated => "state.transition.initiated".to_string(),
            ProtocolEventType::StateTransitionCompleted => "state.transition.completed".to_string(),
            ProtocolEventType::StateTransitionFailed => "state.transition.failed".to_string(),
            ProtocolEventType::StateTimeoutOccurred => "state.timeout.occurred".to_string(),
            ProtocolEventType::ClinicalAlertTriggered => "clinical.alert.triggered".to_string(),
            ProtocolEventType::ClinicalRecommendationGenerated => "clinical.recommendation.generated".to_string(),
            ProtocolEventType::RiskLevelChanged => "risk.level.changed".to_string(),
            _ => format!("{:?}", self).to_lowercase().replace('_', "."),
        }
    }
}

impl Default for EventPublisherConfig {
    fn default() -> Self {
        Self {
            kafka_config: KafkaConfig {
                bootstrap_servers: "localhost:9092".to_string(),
                security_protocol: "PLAINTEXT".to_string(),
                sasl_mechanism: None,
                sasl_username: None,
                sasl_password: None,
                topic_prefix: "cardiofit.protocol".to_string(),
                default_partitions: 3,
                replication_factor: 1,
            },
            buffer_config: BufferConfig {
                max_buffer_size: 10000,
                batch_size: 100,
                max_batch_wait_ms: 1000,
                flush_interval_ms: 5000,
                compression_enabled: true,
                compression_type: "gzip".to_string(),
            },
            schema_config: SchemaConfig {
                validation_enabled: false,
                schema_registry_url: None,
                registry_auth: None,
                version_strategy: SchemaVersionStrategy::Latest,
            },
            reliability_config: ReliabilityConfig {
                max_retries: 3,
                retry_delay_ms: 1000,
                dead_letter_enabled: true,
                dead_letter_suffix: ".dlq".to_string(),
                ack_timeout_ms: 30000,
            },
            enabled: true,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_event_creation() {
        let event = ProtocolEvent {
            event_id: Uuid::new_v4().to_string(),
            event_type: ProtocolEventType::ProtocolEvaluationStarted,
            patient_id: "patient-123".to_string(),
            protocol_id: "sepsis-bundle-v1".to_string(),
            tenant_id: "tenant-001".to_string(),
            source: "protocol-engine".to_string(),
            version: "1.0".to_string(),
            timestamp: Utc::now(),
            correlation_id: None,
            causation_id: None,
            payload: ProtocolEventPayload {
                data: serde_json::json!({"test": "data"}),
                previous_state: None,
                current_state: None,
                error: None,
            },
            metadata: EventMetadata {
                service_version: "1.0.0".to_string(),
                environment: "test".to_string(),
                region: None,
                trace_id: None,
                span_id: None,
                initiated_by: None,
                custom: HashMap::new(),
            },
        };

        let serialized = serde_json::to_string(&event).unwrap();
        assert!(!serialized.is_empty());
        
        let deserialized: ProtocolEvent = serde_json::from_str(&serialized).unwrap();
        assert_eq!(deserialized.event_id, event.event_id);
    }

    #[test]
    fn test_topic_name_generation() {
        let config = EventPublisherConfig::default();
        let publisher = EventPublisher {
            config: config.clone(),
            producer: panic!("Not used in test"),
            buffers: Arc::new(RwLock::new(HashMap::new())),
            stats: Arc::new(PublishingStats::default()),
            topics: Arc::new(RwLock::new(HashMap::new())),
            schema_validator: None,
            batch_processor: Arc::new(Mutex::new(BatchProcessor {
                pending_batches: HashMap::new(),
                last_flush: HashMap::new(),
            })),
        };

        let topic = publisher.get_topic_name(&ProtocolEventType::ProtocolEvaluationStarted);
        assert_eq!(topic, "cardiofit.protocol.protocol-evaluation");
        
        let topic = publisher.get_topic_name(&ProtocolEventType::StateTransitionCompleted);
        assert_eq!(topic, "cardiofit.protocol.state-transition");
    }
}