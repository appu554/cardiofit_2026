//! Protocol Engine Core Types
//!
//! This module defines the fundamental data structures used throughout
//! the Protocol Engine for clinical pathway enforcement and evaluation.

use chrono::{DateTime, Utc, Duration};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;
use indexmap::IndexMap;

/// Protocol evaluation request containing all necessary context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEvaluationRequest {
    /// Unique identifier for the protocol to evaluate
    pub protocol_id: String,
    /// Patient identifier
    pub patient_id: String,
    /// Clinical context and data for evaluation
    pub clinical_context: ProtocolContext,
    /// Optional snapshot ID for deterministic evaluation
    pub snapshot_id: Option<String>,
    /// Timestamp for evaluation (for temporal constraints)
    pub evaluation_timestamp: DateTime<Utc>,
    /// Optional request metadata
    pub metadata: Option<ProtocolRequestMetadata>,
}

/// Clinical context containing patient data and environmental factors
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolContext {
    /// Patient demographic information
    pub patient_demographics: PatientDemographics,
    /// Current medications
    pub medications: Vec<MedicationInfo>,
    /// Active medical conditions
    pub conditions: Vec<ConditionInfo>,
    /// Known allergies and intolerances
    pub allergies: Vec<AllergyInfo>,
    /// Recent laboratory results
    pub lab_results: Vec<LabResult>,
    /// Current vital signs
    pub vital_signs: Vec<VitalSign>,
    /// Clinical encounter context
    pub encounter_context: EncounterContext,
    /// Institution-specific context
    pub institutional_context: InstitutionalContext,
}

impl Default for ProtocolContext {
    fn default() -> Self {
        Self {
            patient_demographics: PatientDemographics::default(),
            medications: Vec::new(),
            conditions: Vec::new(),
            allergies: Vec::new(),
            lab_results: Vec::new(),
            vital_signs: Vec::new(),
            encounter_context: EncounterContext::default(),
            institutional_context: InstitutionalContext::default(),
        }
    }
}

/// Patient demographic information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientDemographics {
    pub age: Option<u32>,
    pub weight_kg: Option<f64>,
    pub height_cm: Option<f64>,
    pub gender: Option<String>,
    pub pregnancy_status: Option<PregnancyStatus>,
}

impl Default for PatientDemographics {
    fn default() -> Self {
        Self {
            age: None,
            weight_kg: None,
            height_cm: None,
            gender: None,
            pregnancy_status: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PregnancyStatus {
    NotPregnant,
    Pregnant { gestational_weeks: Option<u32> },
    Lactating,
    Unknown,
}

/// Medication information for protocol evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationInfo {
    pub medication_id: String,
    pub name: String,
    pub dose: Option<String>,
    pub frequency: Option<String>,
    pub route: Option<String>,
    pub start_date: Option<DateTime<Utc>>,
    pub end_date: Option<DateTime<Utc>>,
    pub indication: Option<String>,
    pub status: MedicationStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum MedicationStatus {
    Active,
    Completed,
    OnHold,
    Stopped,
    Unknown,
}

/// Medical condition information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConditionInfo {
    pub condition_id: String,
    pub code: String,
    pub display: String,
    pub severity: Option<ConditionSeverity>,
    pub onset_date: Option<DateTime<Utc>>,
    pub status: ConditionStatus,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConditionSeverity {
    Mild,
    Moderate,
    Severe,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConditionStatus {
    Active,
    Inactive,
    Resolved,
    Unknown,
}

/// Allergy and intolerance information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AllergyInfo {
    pub allergy_id: String,
    pub substance: String,
    pub reaction_type: AllergyReactionType,
    pub severity: AllergySeverity,
    pub verified_date: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AllergyReactionType {
    Allergy,
    Intolerance,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AllergySeverity {
    Low,
    High,
    UnableToAssess,
}

/// Laboratory result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabResult {
    pub test_name: String,
    pub value: LabValue,
    pub reference_range: Option<String>,
    pub unit: Option<String>,
    pub test_date: DateTime<Utc>,
    pub interpretation: Option<LabInterpretation>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LabValue {
    Numeric(f64),
    Text(String),
    Boolean(bool),
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum LabInterpretation {
    Normal,
    High,
    Low,
    Critical,
    Abnormal,
}

/// Vital sign measurement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VitalSign {
    pub vital_type: VitalSignType,
    pub value: f64,
    pub unit: String,
    pub measurement_time: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VitalSignType {
    BloodPressureSystolic,
    BloodPressureDiastolic,
    HeartRate,
    RespiratoryRate,
    Temperature,
    OxygenSaturation,
    Pain,
}

/// Clinical encounter context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EncounterContext {
    pub encounter_type: EncounterType,
    pub department: Option<String>,
    pub attending_physician: Option<String>,
    pub admission_date: Option<DateTime<Utc>>,
    pub acuity_level: Option<AcuityLevel>,
}

impl Default for EncounterContext {
    fn default() -> Self {
        Self {
            encounter_type: EncounterType::Unknown,
            department: None,
            attending_physician: None,
            admission_date: None,
            acuity_level: None,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EncounterType {
    Inpatient,
    Outpatient,
    Emergency,
    Ambulatory,
    HomeHealth,
    Unknown,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AcuityLevel {
    Level1, // Most acute
    Level2,
    Level3,
    Level4,
    Level5, // Least acute
}

/// Institution-specific context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct InstitutionalContext {
    pub hospital_id: Option<String>,
    pub formulary_preferences: HashMap<String, String>,
    pub local_protocols: Vec<String>,
    pub quality_metrics: HashMap<String, f64>,
}

impl Default for InstitutionalContext {
    fn default() -> Self {
        Self {
            hospital_id: None,
            formulary_preferences: HashMap::new(),
            local_protocols: Vec::new(),
            quality_metrics: HashMap::new(),
        }
    }
}

/// Request metadata for tracking and audit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolRequestMetadata {
    pub request_id: Uuid,
    pub user_id: Option<String>,
    pub session_id: Option<String>,
    pub source_system: Option<String>,
    pub priority: RequestPriority,
    pub timeout_ms: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RequestPriority {
    Low,
    Normal,
    High,
    Critical,
}

/// Protocol evaluation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolEvaluationResult {
    /// Unique result identifier
    pub result_id: Uuid,
    /// Reference to the original request
    pub request_id: Option<Uuid>,
    /// Protocol that was evaluated
    pub protocol_id: String,
    /// Overall decision from protocol evaluation
    pub decision: ProtocolDecision,
    /// Detailed evaluation outcomes
    pub evaluation_details: EvaluationDetails,
    /// State changes resulting from evaluation
    pub state_changes: Vec<StateChange>,
    /// Temporal constraints that apply
    pub temporal_constraints: Vec<AppliedTemporalConstraint>,
    /// Recommendations and alerts
    pub recommendations: Vec<ProtocolRecommendation>,
    /// Evaluation performance metrics
    pub performance_metrics: EvaluationMetrics,
    /// Evaluation timestamp
    pub evaluated_at: DateTime<Utc>,
    /// Snapshot context used for evaluation
    pub snapshot_context: Option<String>,
}

/// Overall protocol decision
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolDecision {
    pub decision_type: ProtocolDecisionType,
    pub confidence: f64, // 0.0 to 1.0
    pub reasoning: String,
    pub requires_approval: bool,
    pub override_available: bool,
}

/// Types of protocol decisions
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub enum ProtocolDecisionType {
    /// Protocol allows the action to proceed
    Allow,
    /// Protocol recommends modifications before proceeding
    Modify,
    /// Protocol blocks the action with hard constraint
    Block,
    /// Protocol requires manual review/approval
    RequireApproval,
    /// Insufficient information to make determination
    Indeterminate,
}

/// Detailed evaluation information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationDetails {
    pub rules_evaluated: Vec<RuleEvaluationResult>,
    pub constraints_checked: Vec<ConstraintEvaluationResult>,
    pub conditions_met: Vec<String>,
    pub conditions_failed: Vec<String>,
    pub warnings: Vec<String>,
    pub information: Vec<String>,
}

/// Individual rule evaluation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleEvaluationResult {
    pub rule_id: String,
    pub rule_name: String,
    pub result: RuleResult,
    pub execution_time_ms: u64,
    pub details: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RuleResult {
    Pass,
    Fail,
    Warning,
    NotApplicable,
    Error(String),
}

/// Constraint evaluation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConstraintEvaluationResult {
    pub constraint_id: String,
    pub constraint_type: ConstraintType,
    pub satisfied: bool,
    pub details: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConstraintType {
    Hard,        // Must be satisfied
    Soft,        // Should be satisfied (warning if not)
    Temporal,    // Time-based constraint
    Conditional, // Depends on other conditions
}

/// State change resulting from protocol evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateChange {
    pub state_machine_id: String,
    pub from_state: String,
    pub to_state: String,
    pub trigger: String,
    pub timestamp: DateTime<Utc>,
}

/// Applied temporal constraint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppliedTemporalConstraint {
    pub constraint_id: String,
    pub constraint_type: String,
    pub time_window_start: DateTime<Utc>,
    pub time_window_end: DateTime<Utc>,
    pub satisfied: bool,
}

/// Protocol recommendation or alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolRecommendation {
    pub recommendation_id: Uuid,
    pub recommendation_type: RecommendationType,
    pub title: String,
    pub description: String,
    pub severity: RecommendationSeverity,
    pub actionable: bool,
    pub suggested_actions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationType {
    DoseAdjustment,
    AlternativeMedication,
    AdditionalMonitoring,
    LabOrderRecommendation,
    ConsultationRecommendation,
    ProtocolDeviation,
    SafetyAlert,
    QualityImprovement,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationSeverity {
    Information,
    Low,
    Medium,
    High,
    Critical,
}

/// Evaluation performance metrics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationMetrics {
    pub total_execution_time_ms: u64,
    pub rules_execution_time_ms: u64,
    pub constraints_execution_time_ms: u64,
    pub state_machine_time_ms: u64,
    pub snapshot_resolution_time_ms: u64,
    pub memory_usage_bytes: Option<u64>,
}

/// Protocol definition structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolDefinition {
    pub protocol_id: String,
    pub name: String,
    pub version: String,
    pub description: String,
    pub metadata: ProtocolMetadata,
    pub rules: Vec<ProtocolRule>,
    pub constraints: Vec<ProtocolConstraint>,
    pub state_machines: Vec<StateMachineDefinition>,
    pub temporal_constraints: Vec<TemporalConstraintDefinition>,
}

/// Protocol metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolMetadata {
    pub author: String,
    pub created_date: DateTime<Utc>,
    pub last_modified: DateTime<Utc>,
    pub approval_status: ApprovalStatus,
    pub clinical_domain: String,
    pub target_population: String,
    pub evidence_level: EvidenceLevel,
    pub tags: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ApprovalStatus {
    Draft,
    UnderReview,
    Approved,
    Deprecated,
    Withdrawn,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceLevel {
    A, // Strong evidence
    B, // Moderate evidence  
    C, // Limited evidence
    D, // Consensus opinion
}

/// Individual protocol rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolRule {
    pub rule_id: String,
    pub name: String,
    pub description: String,
    pub condition: RuleCondition,
    pub action: RuleAction,
    pub priority: u32,
    pub enabled: bool,
}

/// Rule condition expression
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RuleCondition {
    Expression(String),
    And(Vec<RuleCondition>),
    Or(Vec<RuleCondition>),
    Not(Box<RuleCondition>),
}

/// Rule action to take when condition is met
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleAction {
    pub action_type: RuleActionType,
    pub parameters: HashMap<String, serde_json::Value>,
    pub message: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RuleActionType {
    Allow,
    Block,
    Warn,
    Modify,
    RequireApproval,
    StateTransition,
    TriggerAlert,
}

/// Protocol constraint definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolConstraint {
    pub constraint_id: String,
    pub name: String,
    pub constraint_type: ConstraintType,
    pub condition: String,
    pub violation_message: String,
    pub severity: ConstraintSeverity,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ConstraintSeverity {
    Info,
    Warning,
    Error,
    Critical,
}

/// State machine definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateMachineDefinition {
    pub state_machine_id: String,
    pub name: String,
    pub initial_state: String,
    pub states: Vec<StateDefinition>,
    pub transitions: Vec<TransitionDefinition>,
}

/// State definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateDefinition {
    pub state_id: String,
    pub name: String,
    pub description: String,
    pub is_terminal: bool,
    pub entry_actions: Vec<String>,
    pub exit_actions: Vec<String>,
}

/// State transition definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransitionDefinition {
    pub from_state: String,
    pub to_state: String,
    pub trigger: String,
    pub condition: Option<String>,
    pub actions: Vec<String>,
}

/// Temporal constraint definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TemporalConstraintDefinition {
    pub constraint_id: String,
    pub name: String,
    pub constraint_type: TemporalConstraintType,
    pub time_window: TimeWindowDefinition,
    pub condition: String,
    pub violation_action: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TemporalConstraintType {
    WithinTimeWindow,
    BeforeDeadline,
    AfterMinimumTime,
    Periodic,
    SequentialTiming,
}

/// Time window definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeWindowDefinition {
    pub start_offset: Option<Duration>,
    pub end_offset: Option<Duration>,
    pub duration: Option<Duration>,
    pub reference_event: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_protocol_evaluation_request_serialization() {
        let request = ProtocolEvaluationRequest {
            protocol_id: "sepsis-bundle-v1".to_string(),
            patient_id: "patient-12345".to_string(),
            clinical_context: ProtocolContext::default(),
            snapshot_id: Some("snapshot-001".to_string()),
            evaluation_timestamp: Utc::now(),
            metadata: None,
        };
        
        let serialized = serde_json::to_string(&request).unwrap();
        let deserialized: ProtocolEvaluationRequest = serde_json::from_str(&serialized).unwrap();
        assert_eq!(request.protocol_id, deserialized.protocol_id);
    }

    #[test]
    fn test_protocol_decision_types() {
        assert_eq!(ProtocolDecisionType::Allow, ProtocolDecisionType::Allow);
        assert_ne!(ProtocolDecisionType::Allow, ProtocolDecisionType::Block);
    }
}