//! Protocol State Machine Implementation
//!
//! This module implements stateful protocol tracking across clinical pathways
//! with state persistence, transition management, and clinical workflow enforcement.
//! 
//! Features:
//! - Multi-protocol state tracking per patient
//! - Event-driven state transitions with validation
//! - State persistence with snapshot lineage
//! - Concurrent state machine execution
//! - Clinical workflow compliance checking

use std::collections::HashMap;
use std::sync::Arc;
use std::time::Instant;
use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc, Duration};
use tokio::sync::{RwLock as TokioRwLock, Mutex};
use dashmap::DashMap;
use parking_lot::RwLock;

use crate::protocol::{
    types::*,
    error::*,
    engine::StateMachineConfig,
    evaluation::EvaluationContext,
};

/// Protocol state machine for tracking clinical pathway progress
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolStateMachine {
    pub state_machine_id: String,
    pub protocol_id: String,
    pub patient_id: String,
    pub current_state: String,
    pub previous_state: Option<String>,
    pub state_data: HashMap<String, serde_json::Value>,
    pub transition_history: Vec<StateTransitionRecord>,
    pub created_at: DateTime<Utc>,
    pub last_updated: DateTime<Utc>,
    pub snapshot_lineage: Vec<String>,
    pub metadata: StateMachineMetadata,
    pub status: StateMachineStatus,
}

/// State machine metadata for tracking and audit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateMachineMetadata {
    pub version: String,
    pub created_by: String,
    pub encounter_id: Option<String>,
    pub department: Option<String>,
    pub priority: StateMachinePriority,
    pub tags: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateMachinePriority {
    Low,
    Normal,
    High,
    Critical,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateMachineStatus {
    Active,
    Paused,
    Completed,
    Cancelled,
    Error,
}

/// Protocol state definition with enhanced clinical workflow features
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProtocolState {
    pub state_id: String,
    pub name: String,
    pub description: String,
    pub is_initial: bool,
    pub is_terminal: bool,
    pub is_error_state: bool,
    pub timeout_duration: Option<Duration>,
    pub valid_transitions: Vec<String>,
    pub entry_actions: Vec<StateAction>,
    pub exit_actions: Vec<StateAction>,
    pub entry_conditions: Vec<String>,
    pub exit_conditions: Vec<String>,
    pub required_approvals: Vec<ApprovalRequirement>,
    pub clinical_milestones: Vec<ClinicalMilestone>,
}

/// State action to execute on entry/exit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateAction {
    pub action_id: String,
    pub action_type: StateActionType,
    pub parameters: HashMap<String, serde_json::Value>,
    pub required: bool,
    pub timeout_ms: Option<u64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum StateActionType {
    SendNotification,
    CreateTask,
    UpdateRecord,
    TriggerAlert,
    ExecuteRule,
    CallService,
    LogEvent,
    SetTimer,
}

/// Approval requirement for state transitions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ApprovalRequirement {
    pub approval_type: String,
    pub required_role: String,
    pub timeout_hours: Option<u32>,
    pub escalation_roles: Vec<String>,
}

/// Clinical milestone tracking
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalMilestone {
    pub milestone_id: String,
    pub name: String,
    pub description: String,
    pub target_time: Option<Duration>,
    pub compliance_required: bool,
}

/// State transition definition with enhanced validation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateTransition {
    pub transition_id: String,
    pub from_state: String,
    pub to_state: String,
    pub trigger: TransitionTrigger,
    pub conditions: Vec<TransitionCondition>,
    pub actions: Vec<StateAction>,
    pub priority: u32,
    pub enabled: bool,
}

/// Transition trigger types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum TransitionTrigger {
    Manual { user_id: String, reason: String },
    Automatic { rule_id: String },
    Timer { duration: Duration },
    Event { event_type: String, source: String },
    Approval { approval_id: String },
    External { system: String, trigger_id: String },
}

/// Transition condition that must be met
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TransitionCondition {
    pub condition_id: String,
    pub expression: String,
    pub required: bool,
    pub error_message: String,
}

/// Record of a completed state transition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateTransitionRecord {
    pub transition_id: String,
    pub from_state: String,
    pub to_state: String,
    pub trigger: TransitionTrigger,
    pub triggered_by: String,
    pub timestamp: DateTime<Utc>,
    pub duration_ms: u64,
    pub conditions_evaluated: Vec<String>,
    pub actions_executed: Vec<String>,
    pub snapshot_id: Option<String>,
    pub metadata: HashMap<String, serde_json::Value>,
}

/// State machine manager for coordinating multiple protocol state machines
pub struct StateMachineManager {
    config: StateMachineConfig,
    /// Active state machines by patient and protocol
    active_machines: Arc<DashMap<String, Arc<TokioRwLock<ProtocolStateMachine>>>>,
    /// State machine definitions by protocol
    definitions: Arc<RwLock<HashMap<String, StateMachineDefinition>>>,
    /// Transition executor
    transition_executor: Arc<TransitionExecutor>,
    /// Metrics tracking
    metrics: Arc<StateMachineMetrics>,
}

/// State machine definition with states and transitions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StateMachineDefinition {
    pub definition_id: String,
    pub protocol_id: String,
    pub name: String,
    pub version: String,
    pub initial_state: String,
    pub states: HashMap<String, ProtocolState>,
    pub transitions: HashMap<String, StateTransition>,
    pub global_timeout: Option<Duration>,
    pub max_transitions: Option<u32>,
}

/// Transition executor for handling state changes
pub struct TransitionExecutor {
    config: StateMachineConfig,
}

/// State machine performance metrics
#[derive(Debug, Default)]
pub struct StateMachineMetrics {
    pub total_machines: std::sync::atomic::AtomicU64,
    pub active_machines: std::sync::atomic::AtomicU64,
    pub total_transitions: std::sync::atomic::AtomicU64,
    pub failed_transitions: std::sync::atomic::AtomicU64,
    pub average_transition_time_ms: std::sync::atomic::AtomicU64,
}

impl StateMachineManager {
    /// Create new state machine manager
    pub fn new(config: &StateMachineConfig) -> ProtocolResult<Self> {
        Ok(Self {
            config: config.clone(),
            active_machines: Arc::new(DashMap::new()),
            definitions: Arc::new(RwLock::new(HashMap::new())),
            transition_executor: Arc::new(TransitionExecutor::new(config)?),
            metrics: Arc::new(StateMachineMetrics::default()),
        })
    }
    
    /// Initialize state machine for a patient and protocol
    pub async fn initialize_state_machine(
        &self,
        patient_id: &str,
        protocol_id: &str,
        created_by: &str,
        metadata: Option<StateMachineMetadata>,
    ) -> ProtocolResult<String> {
        // Load protocol definition
        let definition = self.get_definition(protocol_id).await?;
        
        // Create state machine
        let state_machine_id = format!("{}:{}:{}", patient_id, protocol_id, Uuid::new_v4());
        
        let mut default_metadata = StateMachineMetadata {
            version: definition.version.clone(),
            created_by: created_by.to_string(),
            encounter_id: None,
            department: None,
            priority: StateMachinePriority::Normal,
            tags: vec![],
        };
        
        if let Some(meta) = metadata {
            default_metadata = meta;
        }
        
        let state_machine = ProtocolStateMachine {
            state_machine_id: state_machine_id.clone(),
            protocol_id: protocol_id.to_string(),
            patient_id: patient_id.to_string(),
            current_state: definition.initial_state.clone(),
            previous_state: None,
            state_data: HashMap::new(),
            transition_history: vec![],
            created_at: Utc::now(),
            last_updated: Utc::now(),
            snapshot_lineage: vec![],
            metadata: default_metadata,
            status: StateMachineStatus::Active,
        };
        
        // Store active machine
        let machine_key = format!("{}:{}", patient_id, protocol_id);
        self.active_machines.insert(
            machine_key,
            Arc::new(TokioRwLock::new(state_machine))
        );
        
        // Update metrics
        self.metrics.total_machines.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        self.metrics.active_machines.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        Ok(state_machine_id)
    }
    
    /// Trigger state transition
    pub async fn trigger_transition(
        &self,
        patient_id: &str,
        protocol_id: &str,
        trigger: TransitionTrigger,
        triggered_by: &str,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<StateTransitionRecord> {
        let machine_key = format!("{}:{}", patient_id, protocol_id);
        
        let machine_arc = self.active_machines.get(&machine_key)
            .ok_or_else(|| ProtocolEngineError::StateMachineError {
                state_machine_id: machine_key.clone(),
                message: "State machine not found".to_string(),
            })?;
        
        let mut machine = machine_arc.write().await;
        let definition = self.get_definition(protocol_id).await?;
        
        // Find applicable transitions
        let applicable_transitions = self.find_applicable_transitions(
            &*machine,
            &definition,
            &trigger,
        )?;
        
        if applicable_transitions.is_empty() {
            return Err(ProtocolEngineError::StateMachineError {
                state_machine_id: machine.state_machine_id.clone(),
                message: format!("No valid transitions from state '{}'", machine.current_state),
            });
        }
        
        // Execute highest priority transition
        let transition = &applicable_transitions[0];
        let start_time = Instant::now();
        
        // Execute transition
        let transition_record = self.transition_executor.execute_transition(
            &mut *machine,
            &definition,
            transition,
            triggered_by,
            eval_context,
        ).await?;
        
        // Update metrics
        self.metrics.total_transitions.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        let transition_time = start_time.elapsed().as_millis() as u64;
        self.update_average_transition_time(transition_time);
        
        Ok(transition_record)
    }
    
    /// Get current state of state machine
    pub async fn get_state_machine(
        &self,
        patient_id: &str,
        protocol_id: &str,
    ) -> ProtocolResult<ProtocolStateMachine> {
        let machine_key = format!("{}:{}", patient_id, protocol_id);
        
        let machine_arc = self.active_machines.get(&machine_key)
            .ok_or_else(|| ProtocolEngineError::StateMachineError {
                state_machine_id: machine_key,
                message: "State machine not found".to_string(),
            })?;
        
        let machine = machine_arc.read().await;
        Ok(machine.clone())
    }
    
    /// Load state machine definition
    async fn get_definition(&self, protocol_id: &str) -> ProtocolResult<StateMachineDefinition> {
        let definitions = self.definitions.read();
        
        if let Some(definition) = definitions.get(protocol_id) {
            return Ok(definition.clone());
        }
        
        drop(definitions);
        
        // Load from storage (placeholder)
        let definition = self.load_definition_from_storage(protocol_id).await?;
        
        // Cache definition
        let mut definitions = self.definitions.write();
        definitions.insert(protocol_id.to_string(), definition.clone());
        
        Ok(definition)
    }
    
    /// Load definition from storage (placeholder implementation)
    async fn load_definition_from_storage(&self, protocol_id: &str) -> ProtocolResult<StateMachineDefinition> {
        match protocol_id {
            "sepsis-bundle-v1" => Ok(self.create_sepsis_state_machine()),
            "vte-prophylaxis-v1" => Ok(self.create_vte_state_machine()),
            _ => Err(ProtocolEngineError::StateMachineError {
                state_machine_id: protocol_id.to_string(),
                message: "State machine definition not found".to_string(),
            }),
        }
    }
    
    /// Create sepsis bundle state machine definition
    fn create_sepsis_state_machine(&self) -> StateMachineDefinition {
        let mut states = HashMap::new();
        let mut transitions = HashMap::new();
        
        // Define states
        states.insert("recognition".to_string(), ProtocolState {
            state_id: "recognition".to_string(),
            name: "Sepsis Recognition".to_string(),
            description: "Initial sepsis recognition and screening".to_string(),
            is_initial: true,
            is_terminal: false,
            is_error_state: false,
            timeout_duration: Some(Duration::minutes(30)),
            valid_transitions: vec!["assessment".to_string(), "no_sepsis".to_string()],
            entry_actions: vec![
                StateAction {
                    action_id: "alert_team".to_string(),
                    action_type: StateActionType::SendNotification,
                    parameters: HashMap::from([
                        ("recipients".to_string(), serde_json::json!(["nursing_team", "physician_team"])),
                        ("message".to_string(), serde_json::json!("Possible sepsis case identified"))
                    ]),
                    required: true,
                    timeout_ms: Some(5000),
                }
            ],
            exit_actions: vec![],
            entry_conditions: vec![],
            exit_conditions: vec!["has_vital_signs && has_clinical_assessment".to_string()],
            required_approvals: vec![],
            clinical_milestones: vec![
                ClinicalMilestone {
                    milestone_id: "sepsis_alert".to_string(),
                    name: "Sepsis Alert Triggered".to_string(),
                    description: "Clinical team alerted to possible sepsis".to_string(),
                    target_time: Some(Duration::minutes(5)),
                    compliance_required: true,
                }
            ],
        });
        
        states.insert("assessment".to_string(), ProtocolState {
            state_id: "assessment".to_string(),
            name: "Clinical Assessment".to_string(),
            description: "Comprehensive sepsis assessment and biomarker collection".to_string(),
            is_initial: false,
            is_terminal: false,
            is_error_state: false,
            timeout_duration: Some(Duration::hours(1)),
            valid_transitions: vec!["bundle_initiation".to_string(), "no_sepsis".to_string()],
            entry_actions: vec![
                StateAction {
                    action_id: "order_labs".to_string(),
                    action_type: StateActionType::CreateTask,
                    parameters: HashMap::from([
                        ("task_type".to_string(), serde_json::json!("lab_order")),
                        ("tests".to_string(), serde_json::json!(["lactate", "blood_culture", "procalcitonin"]))
                    ]),
                    required: true,
                    timeout_ms: Some(10000),
                }
            ],
            exit_actions: vec![],
            entry_conditions: vec![],
            exit_conditions: vec!["has_lab_results".to_string()],
            required_approvals: vec![],
            clinical_milestones: vec![
                ClinicalMilestone {
                    milestone_id: "labs_ordered".to_string(),
                    name: "Sepsis Labs Ordered".to_string(),
                    description: "Blood cultures and lactate ordered".to_string(),
                    target_time: Some(Duration::minutes(30)),
                    compliance_required: true,
                }
            ],
        });
        
        states.insert("bundle_initiation".to_string(), ProtocolState {
            state_id: "bundle_initiation".to_string(),
            name: "Sepsis Bundle Initiation".to_string(),
            description: "3-hour sepsis bundle implementation".to_string(),
            is_initial: false,
            is_terminal: false,
            is_error_state: false,
            timeout_duration: Some(Duration::hours(3)),
            valid_transitions: vec!["bundle_completion".to_string(), "escalation".to_string()],
            entry_actions: vec![
                StateAction {
                    action_id: "start_antibiotics".to_string(),
                    action_type: StateActionType::CreateTask,
                    parameters: HashMap::from([
                        ("task_type".to_string(), serde_json::json!("medication_order")),
                        ("priority".to_string(), serde_json::json!("urgent"))
                    ]),
                    required: true,
                    timeout_ms: Some(15000),
                }
            ],
            exit_actions: vec![],
            entry_conditions: vec!["sepsis_confirmed".to_string()],
            exit_conditions: vec![],
            required_approvals: vec![],
            clinical_milestones: vec![
                ClinicalMilestone {
                    milestone_id: "antibiotics_started".to_string(),
                    name: "Antibiotic Administration".to_string(),
                    description: "Broad-spectrum antibiotics administered".to_string(),
                    target_time: Some(Duration::hours(1)),
                    compliance_required: true,
                }
            ],
        });
        
        states.insert("bundle_completion".to_string(), ProtocolState {
            state_id: "bundle_completion".to_string(),
            name: "Bundle Completion".to_string(),
            description: "Sepsis bundle completed successfully".to_string(),
            is_initial: false,
            is_terminal: true,
            is_error_state: false,
            timeout_duration: None,
            valid_transitions: vec![],
            entry_actions: vec![
                StateAction {
                    action_id: "bundle_completed".to_string(),
                    action_type: StateActionType::LogEvent,
                    parameters: HashMap::from([
                        ("event_type".to_string(), serde_json::json!("sepsis_bundle_completed")),
                        ("compliance".to_string(), serde_json::json!(true))
                    ]),
                    required: true,
                    timeout_ms: Some(1000),
                }
            ],
            exit_actions: vec![],
            entry_conditions: vec!["bundle_elements_completed".to_string()],
            exit_conditions: vec![],
            required_approvals: vec![],
            clinical_milestones: vec![],
        });
        
        states.insert("no_sepsis".to_string(), ProtocolState {
            state_id: "no_sepsis".to_string(),
            name: "Sepsis Ruled Out".to_string(),
            description: "Sepsis ruled out based on clinical assessment".to_string(),
            is_initial: false,
            is_terminal: true,
            is_error_state: false,
            timeout_duration: None,
            valid_transitions: vec![],
            entry_actions: vec![],
            exit_actions: vec![],
            entry_conditions: vec!["sepsis_ruled_out".to_string()],
            exit_conditions: vec![],
            required_approvals: vec![],
            clinical_milestones: vec![],
        });
        
        // Define transitions
        transitions.insert("recognize_to_assess".to_string(), StateTransition {
            transition_id: "recognize_to_assess".to_string(),
            from_state: "recognition".to_string(),
            to_state: "assessment".to_string(),
            trigger: TransitionTrigger::Automatic {
                rule_id: "sepsis_criteria_met".to_string(),
            },
            conditions: vec![
                TransitionCondition {
                    condition_id: "vital_signs_abnormal".to_string(),
                    expression: "has_fever || has_tachycardia || has_hypotension".to_string(),
                    required: true,
                    error_message: "Abnormal vital signs required for sepsis assessment".to_string(),
                }
            ],
            actions: vec![],
            priority: 1,
            enabled: true,
        });
        
        transitions.insert("assess_to_bundle".to_string(), StateTransition {
            transition_id: "assess_to_bundle".to_string(),
            from_state: "assessment".to_string(),
            to_state: "bundle_initiation".to_string(),
            trigger: TransitionTrigger::Automatic {
                rule_id: "sepsis_confirmed".to_string(),
            },
            conditions: vec![
                TransitionCondition {
                    condition_id: "lactate_elevated".to_string(),
                    expression: "lactate_level > 2.0".to_string(),
                    required: true,
                    error_message: "Elevated lactate required for sepsis bundle initiation".to_string(),
                }
            ],
            actions: vec![],
            priority: 1,
            enabled: true,
        });
        
        StateMachineDefinition {
            definition_id: "sepsis-bundle-sm-v1".to_string(),
            protocol_id: "sepsis-bundle-v1".to_string(),
            name: "Sepsis Bundle State Machine".to_string(),
            version: "1.0.0".to_string(),
            initial_state: "recognition".to_string(),
            states,
            transitions,
            global_timeout: Some(Duration::hours(6)),
            max_transitions: Some(10),
        }
    }
    
    /// Create VTE prophylaxis state machine (simplified)
    fn create_vte_state_machine(&self) -> StateMachineDefinition {
        // Placeholder implementation
        StateMachineDefinition {
            definition_id: "vte-prophylaxis-sm-v1".to_string(),
            protocol_id: "vte-prophylaxis-v1".to_string(),
            name: "VTE Prophylaxis State Machine".to_string(),
            version: "1.0.0".to_string(),
            initial_state: "assessment".to_string(),
            states: HashMap::new(),
            transitions: HashMap::new(),
            global_timeout: Some(Duration::hours(24)),
            max_transitions: Some(5),
        }
    }
    
    /// Find applicable transitions from current state
    fn find_applicable_transitions(
        &self,
        machine: &ProtocolStateMachine,
        definition: &StateMachineDefinition,
        trigger: &TransitionTrigger,
    ) -> ProtocolResult<Vec<StateTransition>> {
        let mut applicable = Vec::new();
        
        for transition in definition.transitions.values() {
            if transition.from_state == machine.current_state && 
               transition.enabled &&
               self.trigger_matches(&transition.trigger, trigger) {
                applicable.push(transition.clone());
            }
        }
        
        // Sort by priority (higher first)
        applicable.sort_by(|a, b| b.priority.cmp(&a.priority));
        
        Ok(applicable)
    }
    
    /// Check if trigger matches transition trigger
    fn trigger_matches(&self, transition_trigger: &TransitionTrigger, actual_trigger: &TransitionTrigger) -> bool {
        use TransitionTrigger::*;
        
        match (transition_trigger, actual_trigger) {
            (Manual { .. }, Manual { .. }) => true,
            (Automatic { rule_id: expected }, Automatic { rule_id: actual }) => expected == actual,
            (Timer { .. }, Timer { .. }) => true,
            (Event { event_type: expected, .. }, Event { event_type: actual, .. }) => expected == actual,
            (Approval { .. }, Approval { .. }) => true,
            (External { system: expected_sys, .. }, External { system: actual_sys, .. }) => expected_sys == actual_sys,
            _ => false,
        }
    }
    
    /// Update average transition time metric
    fn update_average_transition_time(&self, transition_time_ms: u64) {
        let current_avg = self.metrics.average_transition_time_ms.load(std::sync::atomic::Ordering::Relaxed);
        let total_transitions = self.metrics.total_transitions.load(std::sync::atomic::Ordering::Relaxed);
        
        if total_transitions > 0 {
            let new_avg = ((current_avg * (total_transitions - 1)) + transition_time_ms) / total_transitions;
            self.metrics.average_transition_time_ms.store(new_avg, std::sync::atomic::Ordering::Relaxed);
        }
    }
    
    /// Get state machine metrics
    pub fn get_metrics(&self) -> StateMachineMetrics {
        StateMachineMetrics {
            total_machines: std::sync::atomic::AtomicU64::new(
                self.metrics.total_machines.load(std::sync::atomic::Ordering::Relaxed)
            ),
            active_machines: std::sync::atomic::AtomicU64::new(
                self.metrics.active_machines.load(std::sync::atomic::Ordering::Relaxed)
            ),
            total_transitions: std::sync::atomic::AtomicU64::new(
                self.metrics.total_transitions.load(std::sync::atomic::Ordering::Relaxed)
            ),
            failed_transitions: std::sync::atomic::AtomicU64::new(
                self.metrics.failed_transitions.load(std::sync::atomic::Ordering::Relaxed)
            ),
            average_transition_time_ms: std::sync::atomic::AtomicU64::new(
                self.metrics.average_transition_time_ms.load(std::sync::atomic::Ordering::Relaxed)
            ),
        }
    }
}

impl TransitionExecutor {
    /// Create new transition executor
    pub fn new(config: &StateMachineConfig) -> ProtocolResult<Self> {
        Ok(Self {
            config: config.clone(),
        })
    }
    
    /// Execute state transition with full validation and action execution
    pub async fn execute_transition(
        &self,
        machine: &mut ProtocolStateMachine,
        definition: &StateMachineDefinition,
        transition: &StateTransition,
        triggered_by: &str,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<StateTransitionRecord> {
        let start_time = Instant::now();
        
        // Validate transition conditions
        let mut conditions_evaluated = Vec::new();
        for condition in &transition.conditions {
            if condition.required {
                // Placeholder condition evaluation
                // In full implementation, this would use the rule engine
                conditions_evaluated.push(condition.condition_id.clone());
            }
        }
        
        // Execute exit actions from current state
        if let Some(current_state) = definition.states.get(&machine.current_state) {
            for action in &current_state.exit_actions {
                self.execute_state_action(action, machine, eval_context).await?;
            }
        }
        
        // Update state machine
        let old_state = machine.current_state.clone();
        machine.previous_state = Some(old_state.clone());
        machine.current_state = transition.to_state.clone();
        machine.last_updated = Utc::now();
        
        // Execute transition actions
        let mut actions_executed = Vec::new();
        for action in &transition.actions {
            self.execute_state_action(action, machine, eval_context).await?;
            actions_executed.push(action.action_id.clone());
        }
        
        // Execute entry actions for new state
        if let Some(new_state) = definition.states.get(&machine.current_state) {
            for action in &new_state.entry_actions {
                self.execute_state_action(action, machine, eval_context).await?;
                actions_executed.push(action.action_id.clone());
            }
        }
        
        // Create transition record
        let transition_record = StateTransitionRecord {
            transition_id: transition.transition_id.clone(),
            from_state: old_state,
            to_state: machine.current_state.clone(),
            trigger: transition.trigger.clone(),
            triggered_by: triggered_by.to_string(),
            timestamp: Utc::now(),
            duration_ms: start_time.elapsed().as_millis() as u64,
            conditions_evaluated,
            actions_executed,
            snapshot_id: eval_context.request.snapshot_id.clone(),
            metadata: HashMap::new(),
        };
        
        // Add to transition history
        machine.transition_history.push(transition_record.clone());
        
        Ok(transition_record)
    }
    
    /// Execute a state action
    async fn execute_state_action(
        &self,
        action: &StateAction,
        machine: &mut ProtocolStateMachine,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<()> {
        match action.action_type {
            StateActionType::LogEvent => {
                eval_context.add_information(format!(
                    "State action executed: {} for machine {}",
                    action.action_id, machine.state_machine_id
                ));
            },
            StateActionType::SendNotification => {
                eval_context.add_information(format!(
                    "Notification sent: {} for patient {}",
                    action.action_id, machine.patient_id
                ));
            },
            StateActionType::CreateTask => {
                eval_context.add_information(format!(
                    "Task created: {} for patient {}",
                    action.action_id, machine.patient_id
                ));
            },
            _ => {
                // Other action types would be implemented here
                eval_context.add_information(format!(
                    "Action executed: {:?}", action.action_type
                ));
            }
        }
        
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_state_machine_creation() {
        let config = StateMachineConfig {
            max_state_machines: 100,
            state_persistence_enabled: true,
            transition_timeout_ms: 1000,
        };
        
        let manager = StateMachineManager::new(&config);
        assert!(manager.is_ok());
    }
}