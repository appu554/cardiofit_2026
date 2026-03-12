//! Titration Engine - Advanced titration scheduling for chronic medications
//! 
//! This module implements sophisticated titration algorithms with multiple strategies:
//! - Linear titration (fixed increments)
//! - Exponential titration (accelerating increases)
//! - Symptom-driven titration (based on patient response)
//! - Biomarker-guided titration (lab-value driven)

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use chrono::{DateTime, Utc, Duration};
use tracing::{info, warn, error, debug};

/// Titration engine for generating dose escalation/de-escalation schedules
#[derive(Debug)]
pub struct TitrationEngine {
    strategies: HashMap<String, Box<dyn TitrationStrategy>>,
    protocols: HashMap<String, TitrationProtocol>,
    config: TitrationConfig,
}

/// Titration engine configuration
#[derive(Debug, Clone)]
pub struct TitrationConfig {
    pub max_titration_steps: u32,
    pub default_strategy: String,
    pub safety_check_enabled: bool,
    pub max_dose_increase_percent: f64,
    pub min_interval_days: u32,
    pub max_interval_days: u32,
}

impl Default for TitrationConfig {
    fn default() -> Self {
        Self {
            max_titration_steps: 10,
            default_strategy: "linear".to_string(),
            safety_check_enabled: true,
            max_dose_increase_percent: 100.0,
            min_interval_days: 3,
            max_interval_days: 90,
        }
    }
}

/// Titration request containing patient context and goals
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TitrationRequest {
    pub request_id: String,
    pub patient_id: String,
    pub drug_id: String,
    pub current_dose: f64,
    pub target_dose: Option<f64>,
    pub target_biomarker: Option<TargetBiomarker>,
    pub clinical_goals: Vec<ClinicalGoal>,
    pub patient_factors: PatientFactors,
    pub titration_strategy: String,
    pub max_duration_weeks: Option<u32>,
    pub safety_constraints: Vec<SafetyConstraint>,
}

/// Target biomarker for titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TargetBiomarker {
    pub biomarker_name: String,
    pub target_value: f64,
    pub target_range: Option<(f64, f64)>,
    pub unit: String,
    pub monitoring_frequency: String,
}

/// Clinical goal for titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalGoal {
    pub goal_type: ClinicalGoalType,
    pub description: String,
    pub target_value: Option<f64>,
    pub priority: u32,
}

/// Types of clinical goals
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ClinicalGoalType {
    SymptomControl,
    BiomarkerTarget,
    FunctionalImprovement,
    QualityOfLife,
    DiseasePrevention,
}

/// Patient factors affecting titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientFactors {
    pub age_years: f64,
    pub weight_kg: f64,
    pub renal_function: RenalFunction,
    pub hepatic_function: HepaticFunction,
    pub comorbidities: Vec<String>,
    pub concurrent_medications: Vec<String>,
    pub adherence_history: AdherenceHistory,
    pub tolerance_history: ToleranceHistory,
}

/// Renal function assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalFunction {
    pub egfr: f64,
    pub creatinine: f64,
    pub classification: String,
}

/// Hepatic function assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticFunction {
    pub child_pugh_class: Option<String>,
    pub alt: Option<f64>,
    pub ast: Option<f64>,
    pub bilirubin: Option<f64>,
}

/// Adherence history
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdherenceHistory {
    pub average_adherence_percent: f64,
    pub missed_doses_per_week: f64,
    pub adherence_pattern: String,
}

/// Tolerance history
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ToleranceHistory {
    pub previous_adverse_events: Vec<String>,
    pub dose_limiting_toxicities: Vec<String>,
    pub tolerance_level: ToleranceLevel,
}

/// Patient tolerance level
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ToleranceLevel {
    High,
    Normal,
    Low,
    VeryLow,
}

/// Safety constraint for titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyConstraint {
    pub constraint_type: SafetyConstraintType,
    pub parameter: String,
    pub limit_value: f64,
    pub action: String,
}

/// Types of safety constraints
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum SafetyConstraintType {
    MaxDose,
    MaxIncrease,
    MinInterval,
    BiomarkerLimit,
    SymptomThreshold,
}

/// Complete titration schedule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TitrationSchedule {
    pub schedule_id: String,
    pub patient_id: String,
    pub drug_id: String,
    pub strategy_used: String,
    pub created_at: DateTime<Utc>,
    pub total_duration_weeks: u32,
    pub steps: Vec<TitrationStep>,
    pub monitoring_plan: MonitoringPlan,
    pub safety_checkpoints: Vec<SafetyCheckpoint>,
    pub success_criteria: Vec<SuccessCriterion>,
    pub alternative_plans: Vec<AlternativePlan>,
}

/// Individual titration step
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TitrationStep {
    pub step_number: u32,
    pub scheduled_date: DateTime<Utc>,
    pub dose: f64,
    pub dose_change: f64,
    pub dose_change_percent: f64,
    pub rationale: String,
    pub monitoring_requirements: Vec<String>,
    pub patient_instructions: Vec<String>,
    pub safety_warnings: Vec<String>,
    pub next_review_date: DateTime<Utc>,
}

/// Monitoring plan for titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringPlan {
    pub laboratory_monitoring: Vec<LabMonitoring>,
    pub clinical_monitoring: Vec<ClinicalMonitoring>,
    pub patient_reported_outcomes: Vec<String>,
    pub emergency_contacts: Vec<String>,
}

/// Laboratory monitoring requirement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabMonitoring {
    pub test_name: String,
    pub frequency: String,
    pub target_range: Option<(f64, f64)>,
    pub alert_thresholds: Option<(f64, f64)>,
    pub action_on_abnormal: String,
}

/// Clinical monitoring requirement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalMonitoring {
    pub assessment_type: String,
    pub frequency: String,
    pub parameters: Vec<String>,
    pub escalation_criteria: Vec<String>,
}

/// Safety checkpoint in titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyCheckpoint {
    pub checkpoint_date: DateTime<Utc>,
    pub step_number: u32,
    pub required_assessments: Vec<String>,
    pub decision_criteria: Vec<String>,
    pub possible_actions: Vec<String>,
}

/// Success criterion for titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuccessCriterion {
    pub criterion_type: String,
    pub description: String,
    pub target_value: Option<f64>,
    pub measurement_method: String,
    pub evaluation_timepoint: String,
}

/// Alternative plan if primary titration fails
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlternativePlan {
    pub plan_name: String,
    pub trigger_conditions: Vec<String>,
    pub alternative_strategy: String,
    pub dose_modifications: Vec<String>,
    pub additional_monitoring: Vec<String>,
}

/// Titration protocol for specific conditions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TitrationProtocol {
    pub protocol_id: String,
    pub name: String,
    pub indication: String,
    pub drug_class: String,
    pub evidence_level: String,
    pub default_strategy: String,
    pub starting_dose: f64,
    pub target_dose_range: (f64, f64),
    pub typical_duration_weeks: u32,
    pub monitoring_requirements: Vec<String>,
    pub contraindications: Vec<String>,
}

/// Titration strategy trait
pub trait TitrationStrategy: Send + Sync + std::fmt::Debug {
    fn generate_schedule(&self, request: &TitrationRequest) -> Result<TitrationSchedule>;
    fn strategy_name(&self) -> &str;
    fn suitable_for(&self, request: &TitrationRequest) -> bool;
}

impl TitrationEngine {
    /// Create a new titration engine
    pub fn new() -> Result<Self> {
        let mut engine = Self {
            strategies: HashMap::new(),
            protocols: HashMap::new(),
            config: TitrationConfig::default(),
        };
        
        // Register default strategies
        engine.register_default_strategies();
        
        // Load default protocols
        engine.load_default_protocols();
        
        info!("💊 Titration Engine initialized");
        info!("📋 Strategies: {}", engine.strategies.len());
        info!("🏥 Protocols: {}", engine.protocols.len());
        
        Ok(engine)
    }
    
    /// Register default titration strategies
    fn register_default_strategies(&mut self) {
        self.strategies.insert("linear".to_string(), Box::new(LinearTitrationStrategy));
        self.strategies.insert("exponential".to_string(), Box::new(ExponentialTitrationStrategy));
        self.strategies.insert("symptom_driven".to_string(), Box::new(SymptomDrivenTitrationStrategy));
        self.strategies.insert("biomarker_guided".to_string(), Box::new(BiomarkerGuidedTitrationStrategy));
    }
    
    /// Load default clinical protocols
    fn load_default_protocols(&mut self) {
        // Heart failure protocol
        self.protocols.insert("heart_failure_ace_inhibitor".to_string(), TitrationProtocol {
            protocol_id: "hf_acei_v1".to_string(),
            name: "Heart Failure ACE Inhibitor Titration".to_string(),
            indication: "Heart Failure with Reduced Ejection Fraction".to_string(),
            drug_class: "ACE Inhibitor".to_string(),
            evidence_level: "Class I, Level A".to_string(),
            default_strategy: "linear".to_string(),
            starting_dose: 2.5,
            target_dose_range: (10.0, 20.0),
            typical_duration_weeks: 8,
            monitoring_requirements: vec![
                "Blood pressure".to_string(),
                "Serum creatinine".to_string(),
                "Serum potassium".to_string(),
                "Symptoms assessment".to_string(),
            ],
            contraindications: vec![
                "Bilateral renal artery stenosis".to_string(),
                "Pregnancy".to_string(),
                "Angioedema history".to_string(),
            ],
        });
        
        // Hypertension protocol
        self.protocols.insert("hypertension_amlodipine".to_string(), TitrationProtocol {
            protocol_id: "htn_amlodipine_v1".to_string(),
            name: "Hypertension Amlodipine Titration".to_string(),
            indication: "Essential Hypertension".to_string(),
            drug_class: "Calcium Channel Blocker".to_string(),
            evidence_level: "Class I, Level A".to_string(),
            default_strategy: "linear".to_string(),
            starting_dose: 2.5,
            target_dose_range: (5.0, 10.0),
            typical_duration_weeks: 6,
            monitoring_requirements: vec![
                "Blood pressure".to_string(),
                "Heart rate".to_string(),
                "Ankle edema assessment".to_string(),
            ],
            contraindications: vec![
                "Severe aortic stenosis".to_string(),
                "Cardiogenic shock".to_string(),
            ],
        });
        
        // Diabetes protocol
        self.protocols.insert("diabetes_metformin".to_string(), TitrationProtocol {
            protocol_id: "dm_metformin_v1".to_string(),
            name: "Type 2 Diabetes Metformin Titration".to_string(),
            indication: "Type 2 Diabetes Mellitus".to_string(),
            drug_class: "Biguanide".to_string(),
            evidence_level: "Class I, Level A".to_string(),
            default_strategy: "symptom_driven".to_string(),
            starting_dose: 500.0,
            target_dose_range: (1000.0, 2000.0),
            typical_duration_weeks: 4,
            monitoring_requirements: vec![
                "HbA1c".to_string(),
                "Fasting glucose".to_string(),
                "Gastrointestinal tolerance".to_string(),
                "Renal function".to_string(),
            ],
            contraindications: vec![
                "eGFR < 30 mL/min/1.73m²".to_string(),
                "Metabolic acidosis".to_string(),
                "Severe hepatic impairment".to_string(),
            ],
        });
        
        // Anticoagulation protocol
        self.protocols.insert("anticoagulation_warfarin".to_string(), TitrationProtocol {
            protocol_id: "anticoag_warfarin_v1".to_string(),
            name: "Warfarin Anticoagulation Titration".to_string(),
            indication: "Atrial Fibrillation Anticoagulation".to_string(),
            drug_class: "Vitamin K Antagonist".to_string(),
            evidence_level: "Class I, Level A".to_string(),
            default_strategy: "biomarker_guided".to_string(),
            starting_dose: 5.0,
            target_dose_range: (2.0, 10.0),
            typical_duration_weeks: 12,
            monitoring_requirements: vec![
                "INR".to_string(),
                "PT".to_string(),
                "Bleeding assessment".to_string(),
                "Drug interactions review".to_string(),
            ],
            contraindications: vec![
                "Active bleeding".to_string(),
                "Severe hepatic impairment".to_string(),
                "Pregnancy".to_string(),
            ],
        });
    }
    
    /// Generate a titration schedule
    pub fn generate_titration_schedule(&self, request: TitrationRequest) -> Result<TitrationSchedule> {
        info!("📋 Generating titration schedule for patient: {}", request.patient_id);
        
        // Select appropriate strategy
        let strategy_name = if self.strategies.contains_key(&request.titration_strategy) {
            &request.titration_strategy
        } else {
            &self.config.default_strategy
        };
        
        let strategy = self.strategies.get(strategy_name)
            .ok_or_else(|| anyhow!("Titration strategy not found: {}", strategy_name))?;
        
        // Check if strategy is suitable for this request
        if !strategy.suitable_for(&request) {
            warn!("Strategy {} may not be suitable for this request", strategy_name);
        }
        
        // Generate the schedule
        let mut schedule = strategy.generate_schedule(&request)?;
        
        // Apply safety checks
        if self.config.safety_check_enabled {
            self.apply_safety_checks(&mut schedule, &request)?;
        }
        
        // Add monitoring plan
        self.enhance_monitoring_plan(&mut schedule, &request)?;
        
        info!("✅ Titration schedule generated: {} steps over {} weeks", 
              schedule.steps.len(), schedule.total_duration_weeks);
        
        Ok(schedule)
    }
    
    /// Apply safety checks to the schedule
    fn apply_safety_checks(&self, schedule: &mut TitrationSchedule, request: &TitrationRequest) -> Result<()> {
        debug!("🛡️ Applying safety checks to titration schedule");
        
        for step in &mut schedule.steps {
            // Check maximum dose increase
            if step.dose_change_percent > self.config.max_dose_increase_percent {
                step.dose_change_percent = self.config.max_dose_increase_percent;
                step.dose = request.current_dose * (1.0 + self.config.max_dose_increase_percent / 100.0);
                step.safety_warnings.push(format!(
                    "Dose increase limited to {}% for safety", 
                    self.config.max_dose_increase_percent
                ));
            }
            
            // Apply safety constraints
            for constraint in &request.safety_constraints {
                match constraint.constraint_type {
                    SafetyConstraintType::MaxDose => {
                        if step.dose > constraint.limit_value {
                            step.dose = constraint.limit_value;
                            step.safety_warnings.push(format!(
                                "Dose capped at {} mg due to safety constraint", 
                                constraint.limit_value
                            ));
                        }
                    }
                    SafetyConstraintType::MaxIncrease => {
                        if step.dose_change > constraint.limit_value {
                            step.dose_change = constraint.limit_value;
                            step.dose = step.dose - step.dose_change + constraint.limit_value;
                            step.safety_warnings.push(format!(
                                "Dose increase limited to {} mg", 
                                constraint.limit_value
                            ));
                        }
                    }
                    _ => {} // Other constraints handled elsewhere
                }
            }
        }
        
        Ok(())
    }
    
    /// Enhance monitoring plan based on patient factors
    fn enhance_monitoring_plan(&self, schedule: &mut TitrationSchedule, request: &TitrationRequest) -> Result<()> {
        debug!("📊 Enhancing monitoring plan");
        
        // Add renal monitoring if impaired
        if request.patient_factors.renal_function.egfr < 60.0 {
            schedule.monitoring_plan.laboratory_monitoring.push(LabMonitoring {
                test_name: "Serum Creatinine".to_string(),
                frequency: "Weekly".to_string(),
                target_range: None,
                alert_thresholds: Some((1.5, 3.0)),
                action_on_abnormal: "Hold titration, consider dose reduction".to_string(),
            });
        }
        
        // Add hepatic monitoring if indicated
        if request.patient_factors.hepatic_function.child_pugh_class.is_some() {
            schedule.monitoring_plan.laboratory_monitoring.push(LabMonitoring {
                test_name: "Liver Function Tests".to_string(),
                frequency: "Bi-weekly".to_string(),
                target_range: None,
                alert_thresholds: Some((2.0, 5.0)), // Times upper limit of normal
                action_on_abnormal: "Hold titration, hepatology consultation".to_string(),
            });
        }
        
        // Enhanced monitoring for elderly patients
        if request.patient_factors.age_years >= 65.0 {
            schedule.monitoring_plan.clinical_monitoring.push(ClinicalMonitoring {
                assessment_type: "Geriatric Assessment".to_string(),
                frequency: "Each visit".to_string(),
                parameters: vec![
                    "Cognitive function".to_string(),
                    "Fall risk".to_string(),
                    "Polypharmacy review".to_string(),
                ],
                escalation_criteria: vec![
                    "New confusion".to_string(),
                    "Falls".to_string(),
                    "Functional decline".to_string(),
                ],
            });
        }
        
        Ok(())
    }
    
    /// Get available protocols
    pub fn get_protocols(&self) -> Vec<&TitrationProtocol> {
        self.protocols.values().collect()
    }
    
    /// Get protocol by ID
    pub fn get_protocol(&self, protocol_id: &str) -> Option<&TitrationProtocol> {
        self.protocols.get(protocol_id)
    }
    
    /// Get available strategies
    pub fn get_strategy_names(&self) -> Vec<String> {
        self.strategies.keys().cloned().collect()
    }
}

// Default strategy implementations
#[derive(Debug)]
struct LinearTitrationStrategy;

impl TitrationStrategy for LinearTitrationStrategy {
    fn generate_schedule(&self, request: &TitrationRequest) -> Result<TitrationSchedule> {
        let mut steps = Vec::new();
        let target_dose = request.target_dose.unwrap_or(request.current_dose * 2.0);
        let total_increase = target_dose - request.current_dose;
        let num_steps = 4; // Fixed number of steps for linear strategy
        let dose_increment = total_increase / num_steps as f64;
        
        let mut current_date = Utc::now();
        let mut current_dose = request.current_dose;
        
        for step in 1..=num_steps {
            current_dose += dose_increment;
            current_date = current_date + Duration::weeks(2); // 2-week intervals
            
            steps.push(TitrationStep {
                step_number: step,
                scheduled_date: current_date,
                dose: current_dose,
                dose_change: dose_increment,
                dose_change_percent: (dose_increment / (current_dose - dose_increment)) * 100.0,
                rationale: format!("Linear titration step {} of {}", step, num_steps),
                monitoring_requirements: vec!["Clinical assessment".to_string()],
                patient_instructions: vec![
                    format!("Take {} mg daily", current_dose),
                    "Monitor for side effects".to_string(),
                ],
                safety_warnings: vec![],
                next_review_date: current_date + Duration::weeks(2),
            });
        }
        
        Ok(TitrationSchedule {
            schedule_id: format!("linear-{}", uuid::Uuid::new_v4()),
            patient_id: request.patient_id.clone(),
            drug_id: request.drug_id.clone(),
            strategy_used: self.strategy_name().to_string(),
            created_at: Utc::now(),
            total_duration_weeks: (num_steps * 2) as u32,
            steps,
            monitoring_plan: MonitoringPlan {
                laboratory_monitoring: vec![],
                clinical_monitoring: vec![],
                patient_reported_outcomes: vec!["Symptom diary".to_string()],
                emergency_contacts: vec!["24/7 clinical helpline".to_string()],
            },
            safety_checkpoints: vec![],
            success_criteria: vec![],
            alternative_plans: vec![],
        })
    }
    
    fn strategy_name(&self) -> &str {
        "linear"
    }
    
    fn suitable_for(&self, _request: &TitrationRequest) -> bool {
        true // Linear strategy is suitable for most cases
    }
}

#[derive(Debug)]
struct ExponentialTitrationStrategy;

impl TitrationStrategy for ExponentialTitrationStrategy {
    fn generate_schedule(&self, request: &TitrationRequest) -> Result<TitrationSchedule> {
        // Placeholder implementation - would implement exponential dose increases
        LinearTitrationStrategy.generate_schedule(request)
    }
    
    fn strategy_name(&self) -> &str {
        "exponential"
    }
    
    fn suitable_for(&self, request: &TitrationRequest) -> bool {
        // Suitable for patients with good tolerance history
        matches!(request.patient_factors.tolerance_history.tolerance_level, ToleranceLevel::High | ToleranceLevel::Normal)
    }
}

#[derive(Debug)]
struct SymptomDrivenTitrationStrategy;

impl TitrationStrategy for SymptomDrivenTitrationStrategy {
    fn generate_schedule(&self, request: &TitrationRequest) -> Result<TitrationSchedule> {
        // Placeholder implementation - would implement symptom-based titration
        LinearTitrationStrategy.generate_schedule(request)
    }
    
    fn strategy_name(&self) -> &str {
        "symptom_driven"
    }
    
    fn suitable_for(&self, request: &TitrationRequest) -> bool {
        // Suitable when clinical goals include symptom control
        request.clinical_goals.iter().any(|goal| matches!(goal.goal_type, ClinicalGoalType::SymptomControl))
    }
}

#[derive(Debug)]
struct BiomarkerGuidedTitrationStrategy;

impl TitrationStrategy for BiomarkerGuidedTitrationStrategy {
    fn generate_schedule(&self, request: &TitrationRequest) -> Result<TitrationSchedule> {
        // Placeholder implementation - would implement biomarker-guided titration
        LinearTitrationStrategy.generate_schedule(request)
    }
    
    fn strategy_name(&self) -> &str {
        "biomarker_guided"
    }
    
    fn suitable_for(&self, request: &TitrationRequest) -> bool {
        // Suitable when target biomarker is specified
        request.target_biomarker.is_some()
    }
}
