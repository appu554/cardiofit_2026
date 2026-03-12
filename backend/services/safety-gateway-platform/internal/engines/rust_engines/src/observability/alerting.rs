//! Clinical Alerting and Notification System
//!
//! This module provides comprehensive alerting capabilities for clinical systems
//! with patient safety focus, escalation workflows, and multi-channel notifications.

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc};
use uuid::Uuid;

use crate::protocol::error::ProtocolResult;
use crate::observability::logging::{ClinicalContext, ClinicalPriority};

/// Alert manager configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertConfig {
    /// Enable alerting
    pub enabled: bool,
    /// Default alert timeout in minutes
    pub default_timeout_minutes: u32,
    /// Alert routing configuration
    pub routing: AlertRoutingConfig,
    /// Notification channels configuration
    pub channels: Vec<AlertChannelConfig>,
    /// Escalation policies
    pub escalation_policies: Vec<EscalationPolicy>,
    /// Alert suppression rules
    pub suppression_rules: Vec<SuppressionRule>,
    /// Clinical alert settings
    pub clinical_settings: ClinicalAlertSettings,
}

/// Alert routing configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertRoutingConfig {
    /// Default routing rules
    pub default_routes: Vec<RouteRule>,
    /// Emergency override routes
    pub emergency_routes: Vec<RouteRule>,
    /// Department-specific routes
    pub department_routes: HashMap<String, Vec<RouteRule>>,
    /// On-call scheduling integration
    pub oncall_integration: Option<OnCallIntegration>,
}

/// Route rule for alert routing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RouteRule {
    /// Rule name
    pub name: String,
    /// Matching conditions
    pub conditions: Vec<AlertCondition>,
    /// Target channels
    pub target_channels: Vec<String>,
    /// Route priority
    pub priority: u32,
    /// Enable route
    pub enabled: bool,
}

/// Alert condition for routing
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertCondition {
    /// Field to match
    pub field: String,
    /// Matching operator
    pub operator: ConditionOperator,
    /// Value to match
    pub value: serde_json::Value,
    /// Case sensitive matching
    pub case_sensitive: bool,
}

/// Condition operators
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConditionOperator {
    /// Equals
    Equals,
    /// Not equals
    NotEquals,
    /// Contains
    Contains,
    /// Matches regex
    Regex,
    /// Greater than
    GreaterThan,
    /// Less than
    LessThan,
    /// In list
    In,
    /// Not in list
    NotIn,
}

/// On-call integration configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnCallIntegration {
    /// Integration type (pagerduty, opsgenie, etc.)
    pub integration_type: String,
    /// Integration endpoint
    pub endpoint: String,
    /// API credentials
    pub credentials: OnCallCredentials,
    /// Escalation timeout in minutes
    pub escalation_timeout_minutes: u32,
}

/// On-call service credentials
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnCallCredentials {
    /// API key
    pub api_key: String,
    /// Service key
    pub service_key: Option<String>,
    /// Integration key
    pub integration_key: Option<String>,
}

/// Alert channel configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertChannelConfig {
    /// Channel name
    pub name: String,
    /// Channel type
    pub channel_type: AlertChannelType,
    /// Channel configuration
    pub config: ChannelConfig,
    /// Enable channel
    pub enabled: bool,
    /// Retry configuration
    pub retry_config: ChannelRetryConfig,
}

/// Alert channel types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertChannelType {
    /// Email notifications
    Email,
    /// SMS notifications
    SMS,
    /// Slack notifications
    Slack,
    /// Microsoft Teams
    Teams,
    /// Webhook notifications
    Webhook,
    /// PagerDuty integration
    PagerDuty,
    /// Mobile push notifications
    Push,
    /// Phone call notifications
    Phone,
    /// Hospital paging system
    HospitalPager,
    /// EMR integration
    EMR,
}

/// Channel-specific configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelConfig {
    /// SMTP configuration for email
    pub smtp: Option<SmtpConfig>,
    /// SMS configuration
    pub sms: Option<SmsConfig>,
    /// Slack configuration
    pub slack: Option<SlackConfig>,
    /// Teams configuration
    pub teams: Option<TeamsConfig>,
    /// Webhook configuration
    pub webhook: Option<WebhookConfig>,
    /// PagerDuty configuration
    pub pagerduty: Option<PagerDutyConfig>,
    /// Hospital pager configuration
    pub hospital_pager: Option<HospitalPagerConfig>,
}

/// SMTP configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmtpConfig {
    /// SMTP server
    pub server: String,
    /// SMTP port
    pub port: u16,
    /// Username
    pub username: String,
    /// Password
    pub password: String,
    /// Use TLS
    pub use_tls: bool,
    /// From address
    pub from_address: String,
}

/// SMS configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SmsConfig {
    /// SMS provider (twilio, aws-sns, etc.)
    pub provider: String,
    /// Provider credentials
    pub credentials: HashMap<String, String>,
    /// Default sender ID
    pub sender_id: Option<String>,
}

/// Slack configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SlackConfig {
    /// Slack webhook URL
    pub webhook_url: String,
    /// Default channel
    pub default_channel: String,
    /// Bot token
    pub bot_token: Option<String>,
}

/// Microsoft Teams configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TeamsConfig {
    /// Teams webhook URL
    pub webhook_url: String,
    /// Team ID
    pub team_id: Option<String>,
}

/// Webhook configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookConfig {
    /// Webhook URL
    pub url: String,
    /// HTTP method
    pub method: String,
    /// Headers
    pub headers: HashMap<String, String>,
    /// Authentication
    pub auth: Option<WebhookAuth>,
}

/// Webhook authentication
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WebhookAuth {
    /// Auth type (bearer, basic, api-key)
    pub auth_type: String,
    /// Token or key
    pub token: String,
    /// Username (for basic auth)
    pub username: Option<String>,
}

/// PagerDuty configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PagerDutyConfig {
    /// Integration key
    pub integration_key: String,
    /// Routing key
    pub routing_key: Option<String>,
    /// Service key
    pub service_key: Option<String>,
}

/// Hospital pager configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HospitalPagerConfig {
    /// Paging system endpoint
    pub endpoint: String,
    /// System credentials
    pub credentials: HashMap<String, String>,
    /// Default pager group
    pub default_group: String,
}

/// Channel retry configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChannelRetryConfig {
    /// Maximum retry attempts
    pub max_retries: u32,
    /// Retry delay in seconds
    pub retry_delay_seconds: u32,
    /// Exponential backoff
    pub exponential_backoff: bool,
    /// Maximum delay in seconds
    pub max_delay_seconds: u32,
}

/// Escalation policy
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalationPolicy {
    /// Policy name
    pub name: String,
    /// Escalation levels
    pub levels: Vec<EscalationLevel>,
    /// Policy conditions
    pub conditions: Vec<AlertCondition>,
    /// Enable policy
    pub enabled: bool,
}

/// Escalation level
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalationLevel {
    /// Level number (1, 2, 3, etc.)
    pub level: u32,
    /// Escalation delay in minutes
    pub delay_minutes: u32,
    /// Target channels for this level
    pub channels: Vec<String>,
    /// Target recipients
    pub recipients: Vec<String>,
    /// Require acknowledgment
    pub require_acknowledgment: bool,
}

/// Alert suppression rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuppressionRule {
    /// Rule name
    pub name: String,
    /// Suppression conditions
    pub conditions: Vec<AlertCondition>,
    /// Suppression duration in minutes
    pub duration_minutes: u32,
    /// Max suppressed alerts
    pub max_suppressed: Option<u32>,
    /// Rule enabled
    pub enabled: bool,
}

/// Clinical alert settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalAlertSettings {
    /// Enable clinical alerts
    pub enabled: bool,
    /// Patient safety alert threshold
    pub safety_alert_threshold: f64,
    /// Medication interaction alert level
    pub medication_alert_level: MedicationAlertLevel,
    /// Clinical workflow alert settings
    pub workflow_alerts: WorkflowAlertSettings,
    /// Alert fatigue prevention
    pub fatigue_prevention: FatiguePreventionSettings,
}

/// Medication alert levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MedicationAlertLevel {
    /// All interactions
    All,
    /// Major interactions only
    Major,
    /// Critical interactions only
    Critical,
    /// Custom threshold
    Custom(f64),
}

/// Workflow alert settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowAlertSettings {
    /// Alert on workflow timeout
    pub timeout_alerts: bool,
    /// Timeout threshold in minutes
    pub timeout_threshold_minutes: u32,
    /// Alert on step failures
    pub step_failure_alerts: bool,
    /// Alert on approval delays
    pub approval_delay_alerts: bool,
    /// Approval delay threshold in minutes
    pub approval_delay_threshold_minutes: u32,
}

/// Fatigue prevention settings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FatiguePreventionSettings {
    /// Enable fatigue prevention
    pub enabled: bool,
    /// Maximum alerts per hour
    pub max_alerts_per_hour: u32,
    /// Similar alert suppression window in minutes
    pub similar_alert_window_minutes: u32,
    /// Intelligent alert grouping
    pub intelligent_grouping: bool,
    /// Priority-based alert limiting
    pub priority_limiting: bool,
}

/// Clinical alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalAlert {
    /// Alert ID
    pub id: String,
    /// Alert type
    pub alert_type: AlertType,
    /// Alert severity
    pub severity: AlertSeverity,
    /// Alert title
    pub title: String,
    /// Alert description
    pub description: String,
    /// Clinical context
    pub clinical_context: Option<ClinicalContext>,
    /// Alert source
    pub source: AlertSource,
    /// Alert timestamp
    pub timestamp: DateTime<Utc>,
    /// Alert expiration
    pub expires_at: Option<DateTime<Utc>>,
    /// Alert tags
    pub tags: HashMap<String, String>,
    /// Alert metadata
    pub metadata: AlertMetadata,
    /// Current status
    pub status: AlertStatus,
    /// Acknowledgments
    pub acknowledgments: Vec<AlertAcknowledgment>,
}

/// Alert types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertType {
    /// Patient safety alert
    PatientSafety,
    /// Medication interaction
    MedicationInteraction,
    /// Clinical workflow timeout
    WorkflowTimeout,
    /// Protocol violation
    ProtocolViolation,
    /// System performance
    SystemPerformance,
    /// Service availability
    ServiceAvailability,
    /// Data quality issue
    DataQuality,
    /// Security incident
    SecurityIncident,
    /// Compliance violation
    ComplianceViolation,
    /// Custom alert
    Custom(String),
}

/// Alert severity levels
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, PartialOrd, Ord)]
pub enum AlertSeverity {
    /// Critical - immediate action required
    Critical,
    /// High - urgent action needed
    High,
    /// Medium - action needed soon
    Medium,
    /// Low - informational
    Low,
    /// Info - for reference
    Info,
}

/// Alert source information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertSource {
    /// Source service
    pub service: String,
    /// Source component
    pub component: String,
    /// Source version
    pub version: String,
    /// Source instance
    pub instance: Option<String>,
}

/// Alert metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertMetadata {
    /// Alert correlation ID
    pub correlation_id: Option<String>,
    /// Parent alert ID
    pub parent_alert_id: Option<String>,
    /// Related alerts
    pub related_alerts: Vec<String>,
    /// Alert fingerprint for deduplication
    pub fingerprint: String,
    /// Clinical decision context
    pub clinical_decision_context: Option<ClinicalDecisionContext>,
    /// Performance impact
    pub performance_impact: Option<PerformanceImpact>,
}

/// Clinical decision context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalDecisionContext {
    /// Decision point
    pub decision_point: String,
    /// Available options
    pub options: Vec<String>,
    /// Recommended action
    pub recommended_action: String,
    /// Clinical evidence
    pub evidence: Vec<ClinicalEvidence>,
    /// Risk assessment
    pub risk_assessment: RiskAssessment,
}

/// Clinical evidence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalEvidence {
    /// Evidence type
    pub evidence_type: String,
    /// Evidence source
    pub source: String,
    /// Evidence level
    pub level: EvidenceLevel,
    /// Evidence summary
    pub summary: String,
}

/// Evidence levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceLevel {
    /// Level 1 - Systematic review
    Level1,
    /// Level 2 - Randomized controlled trial
    Level2,
    /// Level 3 - Controlled trial without randomization
    Level3,
    /// Level 4 - Case-control or cohort study
    Level4,
    /// Level 5 - Case series or expert opinion
    Level5,
}

/// Risk assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAssessment {
    /// Overall risk score (0.0 to 1.0)
    pub risk_score: f64,
    /// Risk factors
    pub risk_factors: Vec<RiskFactor>,
    /// Mitigation strategies
    pub mitigation_strategies: Vec<String>,
}

/// Risk factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    /// Factor name
    pub name: String,
    /// Factor weight (0.0 to 1.0)
    pub weight: f64,
    /// Factor description
    pub description: String,
    /// Is modifiable
    pub modifiable: bool,
}

/// Performance impact
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceImpact {
    /// Impact score (0.0 to 1.0)
    pub impact_score: f64,
    /// Affected services
    pub affected_services: Vec<String>,
    /// Performance degradation
    pub degradation_percent: Option<f64>,
    /// Estimated recovery time
    pub estimated_recovery_minutes: Option<u32>,
}

/// Alert status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AlertStatus {
    /// Active alert
    Active,
    /// Acknowledged
    Acknowledged,
    /// In progress
    InProgress,
    /// Resolved
    Resolved,
    /// Suppressed
    Suppressed,
    /// Expired
    Expired,
    /// Escalated
    Escalated,
}

/// Alert acknowledgment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertAcknowledgment {
    /// Acknowledgment ID
    pub id: String,
    /// User who acknowledged
    pub user_id: String,
    /// User name
    pub user_name: String,
    /// User role
    pub user_role: String,
    /// Acknowledgment timestamp
    pub timestamp: DateTime<Utc>,
    /// Acknowledgment message
    pub message: Option<String>,
    /// Acknowledgment type
    pub ack_type: AcknowledgmentType,
}

/// Acknowledgment types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AcknowledgmentType {
    /// Manual acknowledgment
    Manual,
    /// Automatic acknowledgment
    Automatic,
    /// System acknowledgment
    System,
    /// Escalation acknowledgment
    Escalation,
}

/// Notification target
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationTarget {
    /// Target type
    pub target_type: TargetType,
    /// Target identifier
    pub identifier: String,
    /// Display name
    pub display_name: String,
    /// Contact preferences
    pub preferences: ContactPreferences,
    /// On-call status
    pub oncall_status: OnCallStatus,
}

/// Notification target types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TargetType {
    /// Individual user
    User,
    /// Team or group
    Team,
    /// Role-based target
    Role,
    /// Department
    Department,
    /// On-call schedule
    OnCallSchedule,
}

/// Contact preferences
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContactPreferences {
    /// Preferred channels by severity
    pub channels_by_severity: HashMap<AlertSeverity, Vec<String>>,
    /// Quiet hours
    pub quiet_hours: Option<QuietHours>,
    /// Emergency contact override
    pub emergency_override: bool,
    /// Maximum alerts per hour
    pub max_alerts_per_hour: Option<u32>,
}

/// Quiet hours configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QuietHours {
    /// Start time (24-hour format)
    pub start_hour: u8,
    /// End time (24-hour format)
    pub end_hour: u8,
    /// Days of week (0 = Sunday)
    pub days_of_week: Vec<u8>,
    /// Emergency override during quiet hours
    pub emergency_override: bool,
}

/// On-call status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OnCallStatus {
    /// Is currently on-call
    pub on_call: bool,
    /// On-call shift start
    pub shift_start: Option<DateTime<Utc>>,
    /// On-call shift end
    pub shift_end: Option<DateTime<Utc>>,
    /// On-call role
    pub role: Option<String>,
    /// Backup contacts
    pub backup_contacts: Vec<String>,
}

/// Alert manager
pub struct AlertManager {
    /// Configuration
    config: AlertConfig,
    /// Active alerts
    active_alerts: Arc<RwLock<HashMap<String, ClinicalAlert>>>,
    /// Alert channels
    channels: Arc<RwLock<HashMap<String, Box<dyn AlertChannel>>>>,
    /// Notification targets
    targets: Arc<RwLock<HashMap<String, NotificationTarget>>>,
    /// Escalation engine
    escalation_engine: Arc<EscalationEngine>,
    /// Suppression engine
    suppression_engine: Arc<SuppressionEngine>,
    /// Alert router
    router: Arc<AlertRouter>,
    /// Fatigue prevention engine
    fatigue_prevention: Arc<FatiguePreventionEngine>,
}

/// Alert channel trait
pub trait AlertChannel: Send + Sync {
    /// Send alert notification
    async fn send_alert(&self, alert: &ClinicalAlert, target: &NotificationTarget) -> ProtocolResult<()>;
    
    /// Get channel name
    fn name(&self) -> &str;
    
    /// Check if channel is healthy
    async fn health_check(&self) -> bool;
}

/// Escalation engine
pub struct EscalationEngine {
    policies: Vec<EscalationPolicy>,
    active_escalations: Arc<RwLock<HashMap<String, ActiveEscalation>>>,
}

/// Active escalation
#[derive(Debug)]
struct ActiveEscalation {
    alert_id: String,
    policy: EscalationPolicy,
    current_level: u32,
    started_at: DateTime<Utc>,
    next_escalation: DateTime<Utc>,
}

/// Suppression engine
pub struct SuppressionEngine {
    rules: Vec<SuppressionRule>,
    suppressed_alerts: Arc<RwLock<HashMap<String, DateTime<Utc>>>>,
}

/// Alert router
pub struct AlertRouter {
    routing_config: AlertRoutingConfig,
}

/// Fatigue prevention engine
pub struct FatiguePreventionEngine {
    config: FatiguePreventionSettings,
    alert_counts: Arc<RwLock<HashMap<String, Vec<DateTime<Utc>>>>>,
    grouped_alerts: Arc<RwLock<HashMap<String, Vec<String>>>>,
}

impl AlertManager {
    /// Create new alert manager
    pub async fn new(config: AlertConfig) -> ProtocolResult<Self> {
        let escalation_engine = Arc::new(EscalationEngine::new(config.escalation_policies.clone()));
        let suppression_engine = Arc::new(SuppressionEngine::new(config.suppression_rules.clone()));
        let router = Arc::new(AlertRouter::new(config.routing.clone()));
        let fatigue_prevention = Arc::new(FatiguePreventionEngine::new(config.clinical_settings.fatigue_prevention.clone()));

        // Initialize channels
        let mut channels: HashMap<String, Box<dyn AlertChannel>> = HashMap::new();
        for channel_config in &config.channels {
            if channel_config.enabled {
                let channel = Self::create_channel(channel_config)?;
                channels.insert(channel_config.name.clone(), channel);
            }
        }

        Ok(Self {
            config,
            active_alerts: Arc::new(RwLock::new(HashMap::new())),
            channels: Arc::new(RwLock::new(channels)),
            targets: Arc::new(RwLock::new(HashMap::new())),
            escalation_engine,
            suppression_engine,
            router,
            fatigue_prevention,
        })
    }

    /// Start alert manager
    pub async fn start(&self) -> ProtocolResult<()> {
        if !self.config.enabled {
            return Ok(());
        }

        // Start escalation monitoring
        let escalation_engine = Arc::clone(&self.escalation_engine);
        let alert_manager = self.clone_for_task();
        tokio::spawn(async move {
            escalation_engine.monitor_escalations(alert_manager).await;
        });

        Ok(())
    }

    /// Stop alert manager
    pub async fn stop(&self) -> ProtocolResult<()> {
        // Stop background tasks (implementation depends on task management)
        Ok(())
    }

    /// Send clinical alert
    pub async fn send_alert(&self, mut alert: ClinicalAlert) -> ProtocolResult<String> {
        if !self.config.enabled {
            return Ok(alert.id.clone());
        }

        // Check suppression rules
        if self.suppression_engine.should_suppress(&alert).await {
            alert.status = AlertStatus::Suppressed;
            return Ok(alert.id.clone());
        }

        // Check fatigue prevention
        if !self.fatigue_prevention.should_send_alert(&alert).await {
            return Ok(alert.id.clone());
        }

        // Route alert to appropriate channels
        let routes = self.router.route_alert(&alert).await?;
        
        // Send notifications
        for route in routes {
            self.send_notifications(&alert, &route).await?;
        }

        // Store active alert
        {
            let mut active_alerts = self.active_alerts.write().await;
            active_alerts.insert(alert.id.clone(), alert.clone());
        }

        // Start escalation if configured
        if let Some(policy) = self.escalation_engine.find_policy(&alert).await {
            self.escalation_engine.start_escalation(&alert, policy).await?;
        }

        // Update fatigue prevention tracking
        self.fatigue_prevention.record_alert(&alert).await;

        Ok(alert.id)
    }

    /// Acknowledge alert
    pub async fn acknowledge_alert(
        &self,
        alert_id: &str,
        user_id: &str,
        user_name: &str,
        user_role: &str,
        message: Option<String>,
    ) -> ProtocolResult<()> {
        let mut active_alerts = self.active_alerts.write().await;
        
        if let Some(alert) = active_alerts.get_mut(alert_id) {
            let acknowledgment = AlertAcknowledgment {
                id: Uuid::new_v4().to_string(),
                user_id: user_id.to_string(),
                user_name: user_name.to_string(),
                user_role: user_role.to_string(),
                timestamp: Utc::now(),
                message,
                ack_type: AcknowledgmentType::Manual,
            };

            alert.acknowledgments.push(acknowledgment);
            alert.status = AlertStatus::Acknowledged;

            // Stop escalation if running
            self.escalation_engine.stop_escalation(alert_id).await;
        }

        Ok(())
    }

    /// Resolve alert
    pub async fn resolve_alert(&self, alert_id: &str, resolution_message: Option<String>) -> ProtocolResult<()> {
        let mut active_alerts = self.active_alerts.write().await;
        
        if let Some(alert) = active_alerts.get_mut(alert_id) {
            alert.status = AlertStatus::Resolved;
            
            // Add system acknowledgment for resolution
            let acknowledgment = AlertAcknowledgment {
                id: Uuid::new_v4().to_string(),
                user_id: "system".to_string(),
                user_name: "System".to_string(),
                user_role: "system".to_string(),
                timestamp: Utc::now(),
                message: resolution_message,
                ack_type: AcknowledgmentType::System,
            };
            alert.acknowledgments.push(acknowledgment);

            // Stop escalation
            self.escalation_engine.stop_escalation(alert_id).await;
        }

        Ok(())
    }

    /// Get active alerts
    pub async fn get_active_alerts(&self) -> Vec<ClinicalAlert> {
        let active_alerts = self.active_alerts.read().await;
        active_alerts.values().cloned().collect()
    }

    /// Get alert by ID
    pub async fn get_alert(&self, alert_id: &str) -> Option<ClinicalAlert> {
        let active_alerts = self.active_alerts.read().await;
        active_alerts.get(alert_id).cloned()
    }

    /// Send notifications for alert
    async fn send_notifications(&self, alert: &ClinicalAlert, route: &RouteResult) -> ProtocolResult<()> {
        let channels = self.channels.read().await;
        let targets = self.targets.read().await;

        for channel_name in &route.channels {
            if let Some(channel) = channels.get(channel_name) {
                for target_id in &route.targets {
                    if let Some(target) = targets.get(target_id) {
                        // Check contact preferences
                        if self.should_contact_target(alert, target) {
                            if let Err(e) = channel.send_alert(alert, target).await {
                                eprintln!("Failed to send alert via {}: {}", channel_name, e);
                            }
                        }
                    }
                }
            }
        }

        Ok(())
    }

    /// Check if target should be contacted
    fn should_contact_target(&self, alert: &ClinicalAlert, target: &NotificationTarget) -> bool {
        // Check quiet hours
        if let Some(quiet_hours) = &target.preferences.quiet_hours {
            if !quiet_hours.emergency_override || alert.severity > AlertSeverity::Critical {
                let now = Utc::now();
                let hour = now.time().hour() as u8;
                let weekday = now.weekday().num_days_from_sunday() as u8;

                if quiet_hours.days_of_week.contains(&weekday) &&
                   hour >= quiet_hours.start_hour && hour < quiet_hours.end_hour {
                    return false;
                }
            }
        }

        // Check preferred channels
        if let Some(preferred_channels) = target.preferences.channels_by_severity.get(&alert.severity) {
            // Implementation would check if current channel is preferred
        }

        true
    }

    /// Create alert channel from configuration
    fn create_channel(config: &AlertChannelConfig) -> ProtocolResult<Box<dyn AlertChannel>> {
        match config.channel_type {
            AlertChannelType::Email => {
                if let Some(smtp_config) = &config.config.smtp {
                    Ok(Box::new(EmailChannel::new(smtp_config.clone())))
                } else {
                    Err(crate::protocol::error::ProtocolEngineError::ConfigurationError(
                        "SMTP configuration required for email channel".to_string()
                    ))
                }
            },
            AlertChannelType::Slack => {
                if let Some(slack_config) = &config.config.slack {
                    Ok(Box::new(SlackChannel::new(slack_config.clone())))
                } else {
                    Err(crate::protocol::error::ProtocolEngineError::ConfigurationError(
                        "Slack configuration required for Slack channel".to_string()
                    ))
                }
            },
            _ => {
                // TODO: Implement other channel types
                Err(crate::protocol::error::ProtocolEngineError::ConfigurationError(
                    format!("Channel type {:?} not implemented", config.channel_type)
                ))
            }
        }
    }

    /// Clone for background tasks
    fn clone_for_task(&self) -> AlertManagerHandle {
        AlertManagerHandle {
            active_alerts: Arc::clone(&self.active_alerts),
            escalation_engine: Arc::clone(&self.escalation_engine),
        }
    }
}

/// Handle for background tasks
#[derive(Clone)]
struct AlertManagerHandle {
    active_alerts: Arc<RwLock<HashMap<String, ClinicalAlert>>>,
    escalation_engine: Arc<EscalationEngine>,
}

/// Route result
struct RouteResult {
    channels: Vec<String>,
    targets: Vec<String>,
}

/// Email alert channel
struct EmailChannel {
    config: SmtpConfig,
}

impl EmailChannel {
    fn new(config: SmtpConfig) -> Self {
        Self { config }
    }
}

#[async_trait::async_trait]
impl AlertChannel for EmailChannel {
    async fn send_alert(&self, alert: &ClinicalAlert, target: &NotificationTarget) -> ProtocolResult<()> {
        // TODO: Implement email sending
        println!("Sending email alert '{}' to {}", alert.title, target.identifier);
        Ok(())
    }

    fn name(&self) -> &str {
        "email"
    }

    async fn health_check(&self) -> bool {
        // TODO: Implement SMTP health check
        true
    }
}

/// Slack alert channel
struct SlackChannel {
    config: SlackConfig,
}

impl SlackChannel {
    fn new(config: SlackConfig) -> Self {
        Self { config }
    }
}

#[async_trait::async_trait]
impl AlertChannel for SlackChannel {
    async fn send_alert(&self, alert: &ClinicalAlert, target: &NotificationTarget) -> ProtocolResult<()> {
        // TODO: Implement Slack message sending
        println!("Sending Slack alert '{}' to {}", alert.title, target.identifier);
        Ok(())
    }

    fn name(&self) -> &str {
        "slack"
    }

    async fn health_check(&self) -> bool {
        // TODO: Implement Slack health check
        true
    }
}

impl EscalationEngine {
    fn new(policies: Vec<EscalationPolicy>) -> Self {
        Self {
            policies,
            active_escalations: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    async fn find_policy(&self, alert: &ClinicalAlert) -> Option<EscalationPolicy> {
        for policy in &self.policies {
            if policy.enabled && self.matches_conditions(alert, &policy.conditions) {
                return Some(policy.clone());
            }
        }
        None
    }

    fn matches_conditions(&self, _alert: &ClinicalAlert, _conditions: &[AlertCondition]) -> bool {
        // TODO: Implement condition matching
        true
    }

    async fn start_escalation(&self, alert: &ClinicalAlert, policy: EscalationPolicy) -> ProtocolResult<()> {
        let escalation = ActiveEscalation {
            alert_id: alert.id.clone(),
            policy: policy.clone(),
            current_level: 1,
            started_at: Utc::now(),
            next_escalation: Utc::now() + Duration::minutes(policy.levels[0].delay_minutes as i64),
        };

        let mut active = self.active_escalations.write().await;
        active.insert(alert.id.clone(), escalation);
        Ok(())
    }

    async fn stop_escalation(&self, alert_id: &str) {
        let mut active = self.active_escalations.write().await;
        active.remove(alert_id);
    }

    async fn monitor_escalations(&self, _alert_manager: AlertManagerHandle) {
        let mut interval = tokio::time::interval(std::time::Duration::from_secs(60));
        
        loop {
            interval.tick().await;
            
            // Check for escalations that need to be triggered
            let now = Utc::now();
            let escalations_to_trigger: Vec<String> = {
                let active = self.active_escalations.read().await;
                active.iter()
                    .filter(|(_, escalation)| escalation.next_escalation <= now)
                    .map(|(alert_id, _)| alert_id.clone())
                    .collect()
            };

            for alert_id in escalations_to_trigger {
                self.trigger_escalation(&alert_id).await;
            }
        }
    }

    async fn trigger_escalation(&self, _alert_id: &str) {
        // TODO: Implement escalation triggering
    }
}

impl SuppressionEngine {
    fn new(rules: Vec<SuppressionRule>) -> Self {
        Self {
            rules,
            suppressed_alerts: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    async fn should_suppress(&self, _alert: &ClinicalAlert) -> bool {
        // TODO: Implement suppression logic
        false
    }
}

impl AlertRouter {
    fn new(config: AlertRoutingConfig) -> Self {
        Self {
            routing_config: config,
        }
    }

    async fn route_alert(&self, _alert: &ClinicalAlert) -> ProtocolResult<Vec<RouteResult>> {
        // TODO: Implement alert routing
        Ok(vec![RouteResult {
            channels: vec!["email".to_string()],
            targets: vec!["default".to_string()],
        }])
    }
}

impl FatiguePreventionEngine {
    fn new(config: FatiguePreventionSettings) -> Self {
        Self {
            config,
            alert_counts: Arc::new(RwLock::new(HashMap::new())),
            grouped_alerts: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    async fn should_send_alert(&self, _alert: &ClinicalAlert) -> bool {
        if !self.config.enabled {
            return true;
        }

        // TODO: Implement fatigue prevention logic
        true
    }

    async fn record_alert(&self, _alert: &ClinicalAlert) {
        // TODO: Record alert for fatigue tracking
    }
}

impl Default for AlertConfig {
    fn default() -> Self {
        Self {
            enabled: true,
            default_timeout_minutes: 60,
            routing: AlertRoutingConfig {
                default_routes: vec![],
                emergency_routes: vec![],
                department_routes: HashMap::new(),
                oncall_integration: None,
            },
            channels: vec![],
            escalation_policies: vec![],
            suppression_rules: vec![],
            clinical_settings: ClinicalAlertSettings {
                enabled: true,
                safety_alert_threshold: 0.8,
                medication_alert_level: MedicationAlertLevel::Major,
                workflow_alerts: WorkflowAlertSettings {
                    timeout_alerts: true,
                    timeout_threshold_minutes: 30,
                    step_failure_alerts: true,
                    approval_delay_alerts: true,
                    approval_delay_threshold_minutes: 60,
                },
                fatigue_prevention: FatiguePreventionSettings {
                    enabled: true,
                    max_alerts_per_hour: 20,
                    similar_alert_window_minutes: 10,
                    intelligent_grouping: true,
                    priority_limiting: true,
                },
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_alert_manager_creation() {
        let config = AlertConfig::default();
        let manager = AlertManager::new(config).await;
        assert!(manager.is_ok());
    }

    #[tokio::test]
    async fn test_clinical_alert_creation() {
        let alert = ClinicalAlert {
            id: Uuid::new_v4().to_string(),
            alert_type: AlertType::PatientSafety,
            severity: AlertSeverity::Critical,
            title: "Patient Safety Alert".to_string(),
            description: "Critical patient safety issue detected".to_string(),
            clinical_context: Some(ClinicalContext::emergency("PAT12345".to_string())),
            source: AlertSource {
                service: "protocol-engine".to_string(),
                component: "safety-monitor".to_string(),
                version: "1.0.0".to_string(),
                instance: None,
            },
            timestamp: Utc::now(),
            expires_at: Some(Utc::now() + Duration::hours(24)),
            tags: HashMap::new(),
            metadata: AlertMetadata {
                correlation_id: None,
                parent_alert_id: None,
                related_alerts: vec![],
                fingerprint: "safety-alert-pat12345".to_string(),
                clinical_decision_context: None,
                performance_impact: None,
            },
            status: AlertStatus::Active,
            acknowledgments: vec![],
        };

        assert_eq!(alert.alert_type, AlertType::PatientSafety);
        assert_eq!(alert.severity, AlertSeverity::Critical);
        assert_eq!(alert.status, AlertStatus::Active);
    }

    #[test]
    fn test_alert_severity_ordering() {
        assert!(AlertSeverity::Critical < AlertSeverity::High);
        assert!(AlertSeverity::High < AlertSeverity::Medium);
        assert!(AlertSeverity::Medium < AlertSeverity::Low);
        assert!(AlertSeverity::Low < AlertSeverity::Info);
    }
}