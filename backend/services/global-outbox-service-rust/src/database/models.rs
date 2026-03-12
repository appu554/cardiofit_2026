use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sqlx::types::JsonValue;
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "event_status", rename_all = "snake_case")]
pub enum EventStatus {
    Pending,
    Published,
    Failed,
    DeadLetter,
}

impl std::fmt::Display for EventStatus {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            EventStatus::Pending => write!(f, "pending"),
            EventStatus::Published => write!(f, "published"),
            EventStatus::Failed => write!(f, "failed"),
            EventStatus::DeadLetter => write!(f, "dead_letter"),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, sqlx::Type)]
#[sqlx(type_name = "medical_context", rename_all = "snake_case")]
pub enum MedicalContext {
    Critical,
    Urgent,
    Routine,
    Background,
}

impl std::fmt::Display for MedicalContext {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            MedicalContext::Critical => write!(f, "critical"),
            MedicalContext::Urgent => write!(f, "urgent"),
            MedicalContext::Routine => write!(f, "routine"),
            MedicalContext::Background => write!(f, "background"),
        }
    }
}

impl Default for MedicalContext {
    fn default() -> Self {
        MedicalContext::Routine
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, sqlx::FromRow)]
pub struct OutboxEvent {
    pub id: Uuid,
    pub service_name: String,
    pub event_type: String,
    pub event_data: String,
    pub topic: String,
    pub correlation_id: Option<String>,
    pub priority: i32,
    pub metadata: Option<JsonValue>,
    pub medical_context: MedicalContext,
    pub created_at: DateTime<Utc>,
    pub published_at: Option<DateTime<Utc>>,
    pub retry_count: i32,
    pub status: EventStatus,
    pub error_message: Option<String>,
    pub next_retry_at: Option<DateTime<Utc>>,
}

impl OutboxEvent {
    pub fn new(
        service_name: String,
        event_type: String,
        event_data: String,
        topic: String,
        priority: i32,
        medical_context: MedicalContext,
    ) -> Self {
        Self {
            id: Uuid::new_v4(),
            service_name,
            event_type,
            event_data,
            topic,
            correlation_id: None,
            priority,
            metadata: None,
            medical_context,
            created_at: Utc::now(),
            published_at: None,
            retry_count: 0,
            status: EventStatus::Pending,
            error_message: None,
            next_retry_at: None,
        }
    }

    pub fn is_critical(&self) -> bool {
        self.medical_context == MedicalContext::Critical
    }

    pub fn is_urgent(&self) -> bool {
        self.medical_context == MedicalContext::Urgent
    }

    pub fn can_retry(&self, max_retries: u32) -> bool {
        self.status == EventStatus::Failed && (self.retry_count as u32) < max_retries
    }

    pub fn increment_retry_count(&mut self, next_retry_at: DateTime<Utc>) {
        self.retry_count += 1;
        self.next_retry_at = Some(next_retry_at);
    }

    pub fn mark_published(&mut self) {
        self.status = EventStatus::Published;
        self.published_at = Some(Utc::now());
        self.error_message = None;
    }

    pub fn mark_failed(&mut self, error_message: String) {
        self.status = EventStatus::Failed;
        self.error_message = Some(error_message);
    }

    pub fn mark_dead_letter(&mut self, error_message: String) {
        self.status = EventStatus::DeadLetter;
        self.error_message = Some(error_message);
    }

    pub fn table_name(&self) -> String {
        format!("outbox_events_{}", self.service_name.replace("-", "_"))
    }

    pub fn priority_order(&self) -> i32 {
        match self.medical_context {
            MedicalContext::Critical => 1,
            MedicalContext::Urgent => 2,
            MedicalContext::Routine => 3,
            MedicalContext::Background => 4,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OutboxStats {
    pub service_name: Option<String>,
    pub queue_depths: HashMap<String, i64>,
    pub total_processed_24h: i64,
    pub dead_letter_count: i64,
    pub success_rates: HashMap<String, f64>,
    pub critical_events_processed: i64,
    pub non_critical_events_dropped: i64,
}

impl Default for OutboxStats {
    fn default() -> Self {
        Self {
            service_name: None,
            queue_depths: HashMap::new(),
            total_processed_24h: 0,
            dead_letter_count: 0,
            success_rates: HashMap::new(),
            critical_events_processed: 0,
            non_critical_events_dropped: 0,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum CircuitBreakerState {
    Closed,
    Open,
    HalfOpen,
}

impl std::fmt::Display for CircuitBreakerState {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            CircuitBreakerState::Closed => write!(f, "CLOSED"),
            CircuitBreakerState::Open => write!(f, "OPEN"),
            CircuitBreakerState::HalfOpen => write!(f, "HALF_OPEN"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CircuitBreakerStatus {
    pub enabled: bool,
    pub state: CircuitBreakerState,
    pub current_load: f64,
    pub total_requests: i64,
    pub failed_requests: i64,
    pub critical_events_processed: i64,
    pub non_critical_events_dropped: i64,
    pub next_retry_at: Option<DateTime<Utc>>,
}

impl Default for CircuitBreakerStatus {
    fn default() -> Self {
        Self {
            enabled: false,
            state: CircuitBreakerState::Closed,
            current_load: 0.0,
            total_requests: 0,
            failed_requests: 0,
            critical_events_processed: 0,
            non_critical_events_dropped: 0,
            next_retry_at: None,
        }
    }
}

// Database row structures for statistics queries
#[derive(Debug, sqlx::FromRow)]
pub struct QueueDepthRow {
    pub service_name: String,
    pub queue_depth: i64,
}

#[derive(Debug, sqlx::FromRow)]
pub struct SuccessRateRow {
    pub service_name: String,
    pub total: i64,
    pub successful: i64,
}

#[derive(Debug, sqlx::FromRow)]
pub struct StatsRow {
    pub total_processed_24h: Option<i64>,
    pub dead_letter_count: Option<i64>,
}