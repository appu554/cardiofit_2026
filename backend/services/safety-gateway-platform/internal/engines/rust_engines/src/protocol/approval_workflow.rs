//! Approval Workflow System
//!
//! This module implements a comprehensive approval workflow system for clinical
//! protocols, enabling role-based approval processes, delegation chains, and
//! audit trails for critical clinical decisions.

use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock, mpsc};
use uuid::Uuid;

use crate::protocol::{
    types::*,
    error::*,
    event_publisher::{EventPublisher, ProtocolEvent, ProtocolEventType},
};

/// Approval workflow configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalWorkflowConfig {
    /// Default approval timeout in minutes
    pub default_approval_timeout_minutes: u32,
    /// Maximum approval chain depth
    pub max_approval_chain_depth: u32,
    /// Enable automatic escalation
    pub enable_auto_escalation: bool,
    /// Auto-escalation timeout in minutes
    pub auto_escalation_timeout_minutes: u32,
    /// Emergency bypass configuration
    pub emergency_bypass_config: EmergencyBypassConfig,
    /// Notification settings
    pub notification_config: ApprovalNotificationConfig,
}

/// Emergency bypass configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmergencyBypassConfig {
    /// Enable emergency bypass
    pub enabled: bool,
    /// Roles allowed to invoke emergency bypass
    pub authorized_roles: Vec<ClinicalRole>,
    /// Require dual authorization for emergency bypass
    pub require_dual_auth: bool,
    /// Maximum bypass duration in minutes
    pub max_bypass_duration_minutes: u32,
    /// Automatic audit notification for bypasses
    pub audit_notification_enabled: bool,
}

/// Approval notification configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalNotificationConfig {
    /// Enable email notifications
    pub email_enabled: bool,
    /// Enable SMS notifications for urgent approvals
    pub sms_enabled: bool,
    /// Enable in-app notifications
    pub in_app_enabled: bool,
    /// Notification retry attempts
    pub retry_attempts: u32,
    /// Notification escalation delay in minutes
    pub escalation_delay_minutes: u32,
}

/// Approval request for protocol modifications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequest {
    /// Unique approval request identifier
    pub request_id: String,
    /// Type of approval required
    pub approval_type: ApprovalType,
    /// Patient identifier
    pub patient_id: String,
    /// Protocol identifier
    pub protocol_id: String,
    /// Tenant/organization identifier
    pub tenant_id: String,
    /// Requesting user/service
    pub requester: ApprovalRequester,
    /// Details of what requires approval
    pub approval_details: ApprovalDetails,
    /// Required approver roles/individuals
    pub required_approvers: Vec<ApprovalRequirement>,
    /// Current approval status
    pub status: ApprovalStatus,
    /// Request priority level
    pub priority: ApprovalPriority,
    /// Request timestamp
    pub created_at: DateTime<Utc>,
    /// Approval deadline
    pub deadline: DateTime<Utc>,
    /// Current approval chain
    pub approval_chain: Vec<ApprovalAction>,
    /// Additional context and justification
    pub context: ApprovalContext,
}

/// Types of approval requests
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalType {
    /// Protocol modification requires approval
    ProtocolModification,
    /// State transition override
    StateTransitionOverride,
    /// Temporal constraint modification
    TemporalConstraintModification,
    /// Rule override or exception
    RuleOverride,
    /// Medication administration outside protocol
    MedicationVariance,
    /// Diagnostic procedure modification
    DiagnosticModification,
    /// Clinical pathway deviation
    PathwayDeviation,
    /// Risk acceptance for known issues
    RiskAcceptance,
    /// Emergency protocol activation
    EmergencyProtocolActivation,
    /// Protocol termination
    ProtocolTermination,
}

/// Approval requester information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequester {
    /// User identifier
    pub user_id: String,
    /// User name
    pub name: String,
    /// Clinical role
    pub role: ClinicalRole,
    /// Department/service
    pub department: String,
    /// License/credential information
    pub credentials: Vec<String>,
    /// Contact information
    pub contact: ContactInfo,
}

/// Clinical roles in the system
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
pub enum ClinicalRole {
    /// Attending physician
    AttendingPhysician,
    /// Resident physician
    ResidentPhysician,
    /// Nurse practitioner
    NursePractitioner,
    /// Registered nurse
    RegisteredNurse,
    /// Charge nurse
    ChargeNurse,
    /// Pharmacist
    Pharmacist,
    /// Clinical pharmacist
    ClinicalPharmacist,
    /// Department manager
    DepartmentManager,
    /// Medical director
    MedicalDirector,
    /// Chief medical officer
    ChiefMedicalOfficer,
    /// Clinical informaticist
    ClinicalInformaticist,
    /// Quality assurance
    QualityAssurance,
    /// Risk management
    RiskManagement,
    /// System administrator
    SystemAdministrator,
}

/// Contact information for users
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContactInfo {
    pub email: Option<String>,
    pub phone: Option<String>,
    pub pager: Option<String>,
    pub mobile: Option<String>,
}

/// Approval details describing what needs approval
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalDetails {
    /// Summary of the change
    pub summary: String,
    /// Detailed description
    pub description: String,
    /// What is being changed
    pub changes: Vec<ApprovalChange>,
    /// Clinical justification
    pub clinical_justification: String,
    /// Risk assessment
    pub risk_assessment: ApprovalRiskAssessment,
    /// Expected benefits
    pub expected_benefits: Vec<String>,
    /// Alternative options considered
    pub alternatives_considered: Vec<String>,
}

/// Individual change requiring approval
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalChange {
    /// Type of change
    pub change_type: ChangeType,
    /// Element being changed
    pub target_element: String,
    /// Current value
    pub current_value: serde_json::Value,
    /// Proposed new value
    pub proposed_value: serde_json::Value,
    /// Reason for change
    pub rationale: String,
}

/// Types of changes
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ChangeType {
    /// Modify existing parameter
    Modification,
    /// Add new element
    Addition,
    /// Remove existing element
    Removal,
    /// Override existing constraint
    Override,
    /// Temporary bypass
    TemporaryBypass,
}

/// Risk assessment for approval request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRiskAssessment {
    /// Overall risk level
    pub risk_level: RiskLevel,
    /// Identified risks
    pub identified_risks: Vec<IdentifiedRisk>,
    /// Mitigation strategies
    pub mitigation_strategies: Vec<String>,
    /// Residual risk after mitigation
    pub residual_risk: RiskLevel,
    /// Risk acceptance criteria
    pub acceptance_criteria: String,
}

/// Risk levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskLevel {
    /// Very low risk
    VeryLow,
    /// Low risk
    Low,
    /// Medium risk
    Medium,
    /// High risk
    High,
    /// Very high risk
    VeryHigh,
    /// Critical risk
    Critical,
}

/// Individual identified risk
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IdentifiedRisk {
    /// Risk description
    pub description: String,
    /// Probability of occurrence
    pub probability: f64, // 0.0 to 1.0
    /// Impact severity
    pub impact: RiskLevel,
    /// Affected stakeholders
    pub affected_parties: Vec<String>,
    /// Risk category
    pub category: RiskCategory,
}

/// Risk categories
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskCategory {
    /// Patient safety risk
    PatientSafety,
    /// Clinical outcome risk
    ClinicalOutcome,
    /// Operational risk
    Operational,
    /// Regulatory compliance risk
    Compliance,
    /// Financial risk
    Financial,
    /// Reputation risk
    Reputation,
}

/// Approval requirement specification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequirement {
    /// Requirement identifier
    pub requirement_id: String,
    /// Required approver specification
    pub approver_spec: ApproverSpecification,
    /// Is this requirement mandatory
    pub mandatory: bool,
    /// Approval order/sequence
    pub sequence_order: Option<u32>,
    /// Timeout for this specific approval
    pub timeout_minutes: Option<u32>,
}

/// Specification of who can approve
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApproverSpecification {
    /// Specific user by ID
    SpecificUser(String),
    /// Any user with specified role
    Role(ClinicalRole),
    /// Any user with one of specified roles
    AnyRole(Vec<ClinicalRole>),
    /// User with specific credentials
    Credential(String),
    /// User from specific department
    Department(String),
    /// Combination requirements (AND logic)
    Combination(Vec<ApproverSpecification>),
    /// Alternative requirements (OR logic)
    Alternative(Vec<ApproverSpecification>),
}

/// Approval status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalStatus {
    /// Waiting for approval
    Pending,
    /// Partially approved (some requirements met)
    PartiallyApproved,
    /// Fully approved
    Approved,
    /// Rejected by approver
    Rejected,
    /// Expired due to timeout
    Expired,
    /// Cancelled by requester
    Cancelled,
    /// Emergency bypass applied
    EmergencyBypassed,
    /// Under review
    UnderReview,
    /// Escalated to higher authority
    Escalated,
}

/// Approval priority levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalPriority {
    /// Life-threatening emergency
    Emergency,
    /// Urgent - within 1 hour
    Urgent,
    /// High - within 4 hours
    High,
    /// Normal - within 24 hours
    Normal,
    /// Low - within 72 hours
    Low,
}

/// Individual approval action
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalAction {
    /// Action identifier
    pub action_id: String,
    /// Type of action
    pub action_type: ApprovalActionType,
    /// User who performed the action
    pub approver: ApprovalRequester,
    /// Action timestamp
    pub timestamp: DateTime<Utc>,
    /// Comments from approver
    pub comments: Option<String>,
    /// Digital signature/authentication proof
    pub signature: Option<DigitalSignature>,
    /// Conditions or stipulations
    pub conditions: Vec<ApprovalCondition>,
}

/// Types of approval actions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalActionType {
    /// Approved the request
    Approved,
    /// Rejected the request
    Rejected,
    /// Requested more information
    InformationRequested,
    /// Delegated to another approver
    Delegated,
    /// Applied conditions to approval
    ConditionalApproval,
    /// Escalated to higher authority
    Escalated,
    /// Acknowledged the request
    Acknowledged,
    /// Emergency bypass applied
    EmergencyBypass,
}

/// Digital signature for approval
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DigitalSignature {
    /// Signature algorithm
    pub algorithm: String,
    /// Signature value
    pub signature: String,
    /// Certificate information
    pub certificate: Option<String>,
    /// Timestamp server proof
    pub timestamp_proof: Option<String>,
}

/// Approval condition or stipulation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalCondition {
    /// Condition identifier
    pub condition_id: String,
    /// Condition description
    pub description: String,
    /// Is condition mandatory
    pub mandatory: bool,
    /// Condition deadline
    pub deadline: Option<DateTime<Utc>>,
    /// Verification requirements
    pub verification_required: bool,
}

/// Approval context and supporting information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalContext {
    /// Clinical context
    pub clinical_situation: String,
    /// Patient acuity level
    pub patient_acuity: String,
    /// Time sensitivity
    pub time_sensitivity: TimeSensitivity,
    /// Related protocols or guidelines
    pub related_protocols: Vec<String>,
    /// Supporting evidence
    pub supporting_evidence: Vec<SupportingEvidence>,
    /// Previous similar cases
    pub precedent_cases: Vec<String>,
}

/// Time sensitivity indicators
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TimeSensitivity {
    /// Immediate action required (minutes)
    Immediate,
    /// Urgent action required (hours)
    Urgent,
    /// Timely action required (days)
    Timely,
    /// Routine scheduling acceptable
    Routine,
}

/// Supporting evidence for approval request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SupportingEvidence {
    /// Evidence type
    pub evidence_type: EvidenceType,
    /// Evidence description
    pub description: String,
    /// Reference or citation
    pub reference: Option<String>,
    /// Quality of evidence
    pub quality_level: EvidenceQuality,
    /// Relevance to current case
    pub relevance: f64, // 0.0 to 1.0
}

/// Types of supporting evidence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceType {
    /// Published clinical guideline
    ClinicalGuideline,
    /// Peer-reviewed research
    Research,
    /// Institutional policy
    InstitutionalPolicy,
    /// Expert consultation
    ExpertConsultation,
    /// Previous case outcome
    CaseOutcome,
    /// Laboratory data
    LaboratoryData,
    /// Imaging results
    ImagingResults,
    /// Clinical observation
    ClinicalObservation,
}

/// Evidence quality levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceQuality {
    /// Systematic review/meta-analysis
    High,
    /// Randomized controlled trial
    Moderate,
    /// Cohort/case-control study
    Low,
    /// Case series/expert opinion
    VeryLow,
}

/// Approval workflow engine
pub struct ApprovalWorkflowEngine {
    /// Configuration
    config: ApprovalWorkflowConfig,
    /// Active approval requests
    active_requests: Arc<RwLock<HashMap<String, ApprovalRequest>>>,
    /// User role mappings
    user_roles: Arc<RwLock<HashMap<String, UserRoleMapping>>>,
    /// Event publisher for notifications
    event_publisher: Option<Arc<EventPublisher>>,
    /// Notification channels
    notification_sender: Option<mpsc::UnboundedSender<ApprovalNotification>>,
    /// Workflow metrics
    metrics: Arc<ApprovalMetrics>,
}

/// User role mapping for authorization
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserRoleMapping {
    pub user_id: String,
    pub roles: HashSet<ClinicalRole>,
    pub departments: HashSet<String>,
    pub credentials: HashSet<String>,
    pub delegation_permissions: DelegationPermissions,
    pub emergency_bypass_authorized: bool,
}

/// Delegation permissions for users
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DelegationPermissions {
    /// Can delegate approvals to others
    pub can_delegate: bool,
    /// Roles that can be delegated to
    pub delegatable_roles: HashSet<ClinicalRole>,
    /// Maximum delegation chain length
    pub max_delegation_depth: u32,
}

/// Approval notification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalNotification {
    pub notification_id: String,
    pub notification_type: ApprovalNotificationType,
    pub request_id: String,
    pub recipient: String,
    pub channel: NotificationChannel,
    pub message: String,
    pub priority: ApprovalPriority,
    pub timestamp: DateTime<Utc>,
}

/// Approval notification types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalNotificationType {
    /// New approval request
    NewRequest,
    /// Approval request reminder
    Reminder,
    /// Request escalation
    Escalation,
    /// Request approved
    Approved,
    /// Request rejected
    Rejected,
    /// Request expired
    Expired,
    /// Emergency bypass used
    EmergencyBypass,
}

/// Notification channels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum NotificationChannel {
    Email,
    SMS,
    InApp,
    Pager,
    Push,
}

/// Approval workflow metrics
#[derive(Debug, Default)]
pub struct ApprovalMetrics {
    /// Total approval requests
    pub total_requests: std::sync::atomic::AtomicU64,
    /// Approved requests
    pub approved_requests: std::sync::atomic::AtomicU64,
    /// Rejected requests
    pub rejected_requests: std::sync::atomic::AtomicU64,
    /// Expired requests
    pub expired_requests: std::sync::atomic::AtomicU64,
    /// Emergency bypasses
    pub emergency_bypasses: std::sync::atomic::AtomicU64,
    /// Average approval time
    pub avg_approval_time_minutes: std::sync::atomic::AtomicU64,
}

impl ApprovalWorkflowEngine {
    /// Create new approval workflow engine
    pub fn new(config: ApprovalWorkflowConfig) -> Self {
        Self {
            config,
            active_requests: Arc::new(RwLock::new(HashMap::new())),
            user_roles: Arc::new(RwLock::new(HashMap::new())),
            event_publisher: None,
            notification_sender: None,
            metrics: Arc::new(ApprovalMetrics::default()),
        }
    }

    /// Set event publisher for workflow events
    pub fn set_event_publisher(&mut self, publisher: Arc<EventPublisher>) {
        self.event_publisher = Some(publisher);
    }

    /// Initialize notification system
    pub fn initialize_notifications(&mut self) -> mpsc::UnboundedReceiver<ApprovalNotification> {
        let (sender, receiver) = mpsc::unbounded_channel();
        self.notification_sender = Some(sender);
        receiver
    }

    /// Submit new approval request
    pub async fn submit_approval_request(
        &self,
        mut request: ApprovalRequest,
    ) -> ProtocolResult<String> {
        // Validate request
        self.validate_approval_request(&request)?;

        // Set request metadata
        request.request_id = Uuid::new_v4().to_string();
        request.created_at = Utc::now();
        request.status = ApprovalStatus::Pending;
        
        // Calculate deadline based on priority
        request.deadline = self.calculate_approval_deadline(&request.priority);

        // Store request
        {
            let mut requests = self.active_requests.write().await;
            requests.insert(request.request_id.clone(), request.clone());
        }

        // Update metrics
        self.metrics.total_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);

        // Send notifications to approvers
        self.notify_approvers(&request).await?;

        // Publish event
        if let Some(publisher) = &self.event_publisher {
            publisher.publish_event(self.create_approval_event(
                &request,
                ProtocolEventType::ProtocolApprovalRequired,
            )).await?;
        }

        // Start auto-escalation timer if enabled
        if self.config.enable_auto_escalation {
            self.schedule_auto_escalation(&request.request_id).await;
        }

        Ok(request.request_id)
    }

    /// Process approval action
    pub async fn process_approval_action(
        &self,
        request_id: &str,
        action: ApprovalAction,
    ) -> ProtocolResult<ApprovalStatus> {
        let mut request = {
            let mut requests = self.active_requests.write().await;
            requests.get_mut(request_id)
                .ok_or_else(|| ProtocolEngineError::ApprovalError(
                    format!("Approval request not found: {}", request_id)
                ))?
                .clone()
        };

        // Validate approver authorization
        self.validate_approver_authorization(&request, &action.approver).await?;

        // Add action to approval chain
        request.approval_chain.push(action.clone());

        // Update request status based on action
        let new_status = match action.action_type {
            ApprovalActionType::Approved => {
                if self.are_all_requirements_satisfied(&request).await? {
                    self.metrics.approved_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    ApprovalStatus::Approved
                } else {
                    ApprovalStatus::PartiallyApproved
                }
            },
            ApprovalActionType::Rejected => {
                self.metrics.rejected_requests.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                ApprovalStatus::Rejected
            },
            ApprovalActionType::EmergencyBypass => {
                self.metrics.emergency_bypasses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                ApprovalStatus::EmergencyBypassed
            },
            ApprovalActionType::Escalated => {
                ApprovalStatus::Escalated
            },
            _ => request.status.clone(),
        };

        request.status = new_status.clone();

        // Update stored request
        {
            let mut requests = self.active_requests.write().await;
            requests.insert(request_id.to_string(), request.clone());
        }

        // Send notifications about status change
        self.notify_status_change(&request).await?;

        // Publish event
        if let Some(publisher) = &self.event_publisher {
            let event_type = match new_status {
                ApprovalStatus::Approved => ProtocolEventType::ProtocolApprovalCompleted,
                ApprovalStatus::Rejected => ProtocolEventType::ProtocolApprovalRejected,
                _ => ProtocolEventType::ProtocolApprovalStatusChanged,
            };
            
            publisher.publish_event(self.create_approval_event(&request, event_type)).await?;
        }

        // Remove from active requests if final status
        if matches!(new_status, ApprovalStatus::Approved | ApprovalStatus::Rejected | 
                   ApprovalStatus::Expired | ApprovalStatus::Cancelled | ApprovalStatus::EmergencyBypassed) {
            let mut requests = self.active_requests.write().await;
            requests.remove(request_id);
            
            // Update approval time metric
            let approval_time = Utc::now().signed_duration_since(request.created_at);
            self.update_approval_time_metric(approval_time.num_minutes() as u64);
        }

        Ok(new_status)
    }

    /// Apply emergency bypass
    pub async fn apply_emergency_bypass(
        &self,
        request_id: &str,
        bypasser: &ApprovalRequester,
        justification: &str,
    ) -> ProtocolResult<()> {
        // Validate emergency bypass authorization
        self.validate_emergency_bypass_authorization(bypasser).await?;

        let action = ApprovalAction {
            action_id: Uuid::new_v4().to_string(),
            action_type: ApprovalActionType::EmergencyBypass,
            approver: bypasser.clone(),
            timestamp: Utc::now(),
            comments: Some(justification.to_string()),
            signature: None, // TODO: Implement digital signatures
            conditions: vec![],
        };

        self.process_approval_action(request_id, action).await?;

        // Send audit notification for emergency bypass
        if self.config.emergency_bypass_config.audit_notification_enabled {
            self.send_audit_notification(request_id, bypasser, justification).await?;
        }

        Ok(())
    }

    /// Get approval request status
    pub async fn get_approval_status(&self, request_id: &str) -> ProtocolResult<ApprovalRequest> {
        let requests = self.active_requests.read().await;
        requests.get(request_id)
            .cloned()
            .ok_or_else(|| ProtocolEngineError::ApprovalError(
                format!("Approval request not found: {}", request_id)
            ))
    }

    /// List pending approvals for user
    pub async fn get_pending_approvals_for_user(
        &self,
        user_id: &str,
    ) -> ProtocolResult<Vec<ApprovalRequest>> {
        let requests = self.active_requests.read().await;
        let user_roles = self.get_user_roles(user_id).await?;
        
        let mut pending_approvals = Vec::new();
        
        for request in requests.values() {
            if matches!(request.status, ApprovalStatus::Pending | ApprovalStatus::PartiallyApproved) {
                if self.can_user_approve(request, &user_roles).await? {
                    pending_approvals.push(request.clone());
                }
            }
        }

        Ok(pending_approvals)
    }

    /// Register user roles
    pub async fn register_user_roles(&self, mapping: UserRoleMapping) {
        let mut roles = self.user_roles.write().await;
        roles.insert(mapping.user_id.clone(), mapping);
    }

    /// Validate approval request
    fn validate_approval_request(&self, request: &ApprovalRequest) -> ProtocolResult<()> {
        if request.patient_id.is_empty() {
            return Err(ProtocolEngineError::ApprovalError(
                "Patient ID is required".to_string()
            ));
        }

        if request.protocol_id.is_empty() {
            return Err(ProtocolEngineError::ApprovalError(
                "Protocol ID is required".to_string()
            ));
        }

        if request.required_approvers.is_empty() {
            return Err(ProtocolEngineError::ApprovalError(
                "At least one approver requirement is required".to_string()
            ));
        }

        Ok(())
    }

    /// Calculate approval deadline based on priority
    fn calculate_approval_deadline(&self, priority: &ApprovalPriority) -> DateTime<Utc> {
        let timeout_minutes = match priority {
            ApprovalPriority::Emergency => 15,
            ApprovalPriority::Urgent => 60,
            ApprovalPriority::High => 240,
            ApprovalPriority::Normal => 1440, // 24 hours
            ApprovalPriority::Low => 4320, // 72 hours
        };

        Utc::now() + Duration::minutes(timeout_minutes)
    }

    /// Validate approver authorization
    async fn validate_approver_authorization(
        &self,
        request: &ApprovalRequest,
        approver: &ApprovalRequester,
    ) -> ProtocolResult<()> {
        let user_roles = self.get_user_roles(&approver.user_id).await?;
        
        for requirement in &request.required_approvers {
            if self.matches_approver_spec(&requirement.approver_spec, &user_roles).await? {
                return Ok(());
            }
        }

        Err(ProtocolEngineError::ApprovalError(
            format!("User {} is not authorized to approve this request", approver.user_id)
        ))
    }

    /// Check if all approval requirements are satisfied
    async fn are_all_requirements_satisfied(&self, request: &ApprovalRequest) -> ProtocolResult<bool> {
        for requirement in &request.required_approvers {
            if requirement.mandatory {
                let satisfied = request.approval_chain.iter().any(|action| {
                    matches!(action.action_type, ApprovalActionType::Approved) &&
                    self.matches_approver_spec_sync(&requirement.approver_spec, &action.approver)
                });
                
                if !satisfied {
                    return Ok(false);
                }
            }
        }
        
        Ok(true)
    }

    /// Check if approver specification matches user roles (synchronous version)
    fn matches_approver_spec_sync(&self, spec: &ApproverSpecification, approver: &ApprovalRequester) -> bool {
        match spec {
            ApproverSpecification::SpecificUser(user_id) => approver.user_id == *user_id,
            ApproverSpecification::Role(role) => approver.role == *role,
            ApproverSpecification::AnyRole(roles) => roles.contains(&approver.role),
            ApproverSpecification::Department(dept) => approver.department == *dept,
            ApproverSpecification::Credential(cred) => approver.credentials.contains(cred),
            _ => false, // Complex specs need async evaluation
        }
    }

    /// Check if approver specification matches user roles
    async fn matches_approver_spec(
        &self,
        spec: &ApproverSpecification,
        user_roles: &UserRoleMapping,
    ) -> ProtocolResult<bool> {
        match spec {
            ApproverSpecification::SpecificUser(user_id) => Ok(user_roles.user_id == *user_id),
            ApproverSpecification::Role(role) => Ok(user_roles.roles.contains(role)),
            ApproverSpecification::AnyRole(roles) => Ok(roles.iter().any(|r| user_roles.roles.contains(r))),
            ApproverSpecification::Department(dept) => Ok(user_roles.departments.contains(dept)),
            ApproverSpecification::Credential(cred) => Ok(user_roles.credentials.contains(cred)),
            ApproverSpecification::Combination(specs) => {
                for spec in specs {
                    if !self.matches_approver_spec(spec, user_roles).await? {
                        return Ok(false);
                    }
                }
                Ok(true)
            },
            ApproverSpecification::Alternative(specs) => {
                for spec in specs {
                    if self.matches_approver_spec(spec, user_roles).await? {
                        return Ok(true);
                    }
                }
                Ok(false)
            },
        }
    }

    /// Get user roles
    async fn get_user_roles(&self, user_id: &str) -> ProtocolResult<UserRoleMapping> {
        let roles = self.user_roles.read().await;
        roles.get(user_id)
            .cloned()
            .ok_or_else(|| ProtocolEngineError::ApprovalError(
                format!("User roles not found: {}", user_id)
            ))
    }

    /// Check if user can approve request
    async fn can_user_approve(
        &self,
        request: &ApprovalRequest,
        user_roles: &UserRoleMapping,
    ) -> ProtocolResult<bool> {
        for requirement in &request.required_approvers {
            if self.matches_approver_spec(&requirement.approver_spec, user_roles).await? {
                return Ok(true);
            }
        }
        Ok(false)
    }

    /// Validate emergency bypass authorization
    async fn validate_emergency_bypass_authorization(
        &self,
        bypasser: &ApprovalRequester,
    ) -> ProtocolResult<()> {
        if !self.config.emergency_bypass_config.enabled {
            return Err(ProtocolEngineError::ApprovalError(
                "Emergency bypass is disabled".to_string()
            ));
        }

        let user_roles = self.get_user_roles(&bypasser.user_id).await?;
        
        if !user_roles.emergency_bypass_authorized {
            return Err(ProtocolEngineError::ApprovalError(
                "User not authorized for emergency bypass".to_string()
            ));
        }

        if !self.config.emergency_bypass_config.authorized_roles.contains(&bypasser.role) {
            return Err(ProtocolEngineError::ApprovalError(
                "Role not authorized for emergency bypass".to_string()
            ));
        }

        Ok(())
    }

    /// Send notifications to approvers
    async fn notify_approvers(&self, request: &ApprovalRequest) -> ProtocolResult<()> {
        if let Some(sender) = &self.notification_sender {
            // TODO: Identify specific users who can approve based on requirements
            let notification = ApprovalNotification {
                notification_id: Uuid::new_v4().to_string(),
                notification_type: ApprovalNotificationType::NewRequest,
                request_id: request.request_id.clone(),
                recipient: "approvers".to_string(), // TODO: Resolve to specific users
                channel: NotificationChannel::InApp,
                message: format!("New approval request: {}", request.approval_details.summary),
                priority: request.priority.clone(),
                timestamp: Utc::now(),
            };

            let _ = sender.send(notification);
        }

        Ok(())
    }

    /// Send status change notifications
    async fn notify_status_change(&self, request: &ApprovalRequest) -> ProtocolResult<()> {
        if let Some(sender) = &self.notification_sender {
            let notification_type = match request.status {
                ApprovalStatus::Approved => ApprovalNotificationType::Approved,
                ApprovalStatus::Rejected => ApprovalNotificationType::Rejected,
                ApprovalStatus::Expired => ApprovalNotificationType::Expired,
                _ => return Ok(()),
            };

            let notification = ApprovalNotification {
                notification_id: Uuid::new_v4().to_string(),
                notification_type,
                request_id: request.request_id.clone(),
                recipient: request.requester.user_id.clone(),
                channel: NotificationChannel::InApp,
                message: format!("Approval request status changed: {:?}", request.status),
                priority: request.priority.clone(),
                timestamp: Utc::now(),
            };

            let _ = sender.send(notification);
        }

        Ok(())
    }

    /// Send audit notification for emergency bypass
    async fn send_audit_notification(
        &self,
        request_id: &str,
        bypasser: &ApprovalRequester,
        justification: &str,
    ) -> ProtocolResult<()> {
        if let Some(sender) = &self.notification_sender {
            let notification = ApprovalNotification {
                notification_id: Uuid::new_v4().to_string(),
                notification_type: ApprovalNotificationType::EmergencyBypass,
                request_id: request_id.to_string(),
                recipient: "audit@hospital.com".to_string(), // TODO: Configure audit recipients
                channel: NotificationChannel::Email,
                message: format!(
                    "Emergency bypass applied by {} ({}): {}",
                    bypasser.name, bypasser.role, justification
                ),
                priority: ApprovalPriority::Emergency,
                timestamp: Utc::now(),
            };

            let _ = sender.send(notification);
        }

        Ok(())
    }

    /// Schedule auto-escalation for request
    async fn schedule_auto_escalation(&self, _request_id: &str) {
        // TODO: Implement auto-escalation scheduling
        // This would typically involve creating a background task
        // that escalates requests after the configured timeout
    }

    /// Create approval event for publishing
    fn create_approval_event(&self, request: &ApprovalRequest, event_type: ProtocolEventType) -> ProtocolEvent {
        ProtocolEvent {
            event_id: Uuid::new_v4().to_string(),
            event_type,
            patient_id: request.patient_id.clone(),
            protocol_id: request.protocol_id.clone(),
            tenant_id: request.tenant_id.clone(),
            source: "approval-workflow".to_string(),
            version: "1.0".to_string(),
            timestamp: Utc::now(),
            correlation_id: Some(request.request_id.clone()),
            causation_id: None,
            payload: crate::protocol::event_publisher::ProtocolEventPayload {
                data: serde_json::to_value(request).unwrap_or_default(),
                previous_state: None,
                current_state: Some(serde_json::to_value(&request.status).unwrap_or_default()),
                error: None,
            },
            metadata: crate::protocol::event_publisher::EventMetadata {
                service_version: "1.0.0".to_string(),
                environment: "production".to_string(),
                region: Some("us-east-1".to_string()),
                trace_id: None,
                span_id: None,
                initiated_by: Some(request.requester.user_id.clone()),
                custom: std::collections::HashMap::new(),
            },
        }
    }

    /// Update approval time metric
    fn update_approval_time_metric(&self, approval_time_minutes: u64) {
        let current_avg = self.metrics.avg_approval_time_minutes.load(std::sync::atomic::Ordering::Relaxed);
        let total_approved = self.metrics.approved_requests.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_approved > 1 {
            let new_avg = (current_avg * (total_approved - 1) + approval_time_minutes) / total_approved;
            self.metrics.avg_approval_time_minutes.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        } else {
            self.metrics.avg_approval_time_minutes.store(approval_time_minutes, std::sync::atomic::Ordering::Relaxed);
        }
    }

    /// Get approval metrics
    pub fn get_metrics(&self) -> ApprovalMetricsSnapshot {
        ApprovalMetricsSnapshot {
            total_requests: self.metrics.total_requests.load(std::sync::atomic::Ordering::Relaxed),
            approved_requests: self.metrics.approved_requests.load(std::sync::atomic::Ordering::Relaxed),
            rejected_requests: self.metrics.rejected_requests.load(std::sync::atomic::Ordering::Relaxed),
            expired_requests: self.metrics.expired_requests.load(std::sync::atomic::Ordering::Relaxed),
            emergency_bypasses: self.metrics.emergency_bypasses.load(std::sync::atomic::Ordering::Relaxed),
            avg_approval_time_minutes: self.metrics.avg_approval_time_minutes.load(std::sync::atomic::Ordering::Relaxed),
        }
    }
}

/// Approval metrics snapshot
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalMetricsSnapshot {
    pub total_requests: u64,
    pub approved_requests: u64,
    pub rejected_requests: u64,
    pub expired_requests: u64,
    pub emergency_bypasses: u64,
    pub avg_approval_time_minutes: u64,
}

impl Default for ApprovalWorkflowConfig {
    fn default() -> Self {
        Self {
            default_approval_timeout_minutes: 1440, // 24 hours
            max_approval_chain_depth: 5,
            enable_auto_escalation: true,
            auto_escalation_timeout_minutes: 240, // 4 hours
            emergency_bypass_config: EmergencyBypassConfig {
                enabled: true,
                authorized_roles: vec![
                    ClinicalRole::AttendingPhysician,
                    ClinicalRole::MedicalDirector,
                    ClinicalRole::ChiefMedicalOfficer,
                ],
                require_dual_auth: false,
                max_bypass_duration_minutes: 60,
                audit_notification_enabled: true,
            },
            notification_config: ApprovalNotificationConfig {
                email_enabled: true,
                sms_enabled: true,
                in_app_enabled: true,
                retry_attempts: 3,
                escalation_delay_minutes: 60,
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_approval_workflow_creation() {
        let config = ApprovalWorkflowConfig::default();
        let workflow = ApprovalWorkflowEngine::new(config);
        assert!(workflow.active_requests.read().await.is_empty());
    }

    #[test]
    fn test_approval_request_serialization() {
        let request = ApprovalRequest {
            request_id: "req-123".to_string(),
            approval_type: ApprovalType::ProtocolModification,
            patient_id: "patient-456".to_string(),
            protocol_id: "sepsis-bundle-v1".to_string(),
            tenant_id: "tenant-001".to_string(),
            requester: ApprovalRequester {
                user_id: "user-789".to_string(),
                name: "Dr. Smith".to_string(),
                role: ClinicalRole::AttendingPhysician,
                department: "Emergency Medicine".to_string(),
                credentials: vec!["MD".to_string(), "FACEP".to_string()],
                contact: ContactInfo {
                    email: Some("dr.smith@hospital.com".to_string()),
                    phone: Some("+1-555-1234".to_string()),
                    pager: None,
                    mobile: None,
                },
            },
            approval_details: ApprovalDetails {
                summary: "Modify sepsis bundle timing".to_string(),
                description: "Adjust antibiotic administration window".to_string(),
                changes: vec![],
                clinical_justification: "Patient has unusual allergies".to_string(),
                risk_assessment: ApprovalRiskAssessment {
                    risk_level: RiskLevel::Medium,
                    identified_risks: vec![],
                    mitigation_strategies: vec![],
                    residual_risk: RiskLevel::Low,
                    acceptance_criteria: "Acceptable with monitoring".to_string(),
                },
                expected_benefits: vec!["Improved patient safety".to_string()],
                alternatives_considered: vec!["Alternative antibiotic".to_string()],
            },
            required_approvers: vec![ApprovalRequirement {
                requirement_id: "req-1".to_string(),
                approver_spec: ApproverSpecification::Role(ClinicalRole::ClinicalPharmacist),
                mandatory: true,
                sequence_order: Some(1),
                timeout_minutes: Some(120),
            }],
            status: ApprovalStatus::Pending,
            priority: ApprovalPriority::High,
            created_at: Utc::now(),
            deadline: Utc::now() + Duration::hours(4),
            approval_chain: vec![],
            context: ApprovalContext {
                clinical_situation: "Sepsis with drug allergies".to_string(),
                patient_acuity: "High".to_string(),
                time_sensitivity: TimeSensitivity::Urgent,
                related_protocols: vec!["sepsis-bundle-v1".to_string()],
                supporting_evidence: vec![],
                precedent_cases: vec![],
            },
        };

        let serialized = serde_json::to_string(&request).unwrap();
        assert!(!serialized.is_empty());
        
        let deserialized: ApprovalRequest = serde_json::from_str(&serialized).unwrap();
        assert_eq!(deserialized.request_id, request.request_id);
    }
}