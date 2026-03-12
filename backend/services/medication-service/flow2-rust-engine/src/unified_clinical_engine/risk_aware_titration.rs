//! Risk-Aware Titration - Integration of titration scheduling with risk management
//! 
//! This module combines the titration engine with cumulative risk assessment
//! to create risk-adjusted titration schedules with enhanced safety monitoring.

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use chrono::{DateTime, Utc, Duration};
use tracing::{info, warn, error, debug};

use super::titration_engine::{TitrationEngine, TitrationRequest, TitrationSchedule, TitrationStep};
use super::cumulative_risk::{CumulativeRiskAssessment, CumulativeRiskProfile, RiskLevel};

/// Risk-aware titration engine that integrates risk assessment with titration scheduling
#[derive(Debug)]
pub struct RiskAwareTitrationEngine {
    titration_engine: TitrationEngine,
    risk_assessment: CumulativeRiskAssessment,
    risk_adjusters: HashMap<RiskLevel, RiskAdjuster>,
    config: RiskAwareTitrationConfig,
}

/// Configuration for risk-aware titration
#[derive(Debug, Clone)]
pub struct RiskAwareTitrationConfig {
    pub enable_risk_adjustment: bool,
    pub enable_dynamic_monitoring: bool,
    pub enable_safety_checkpoints: bool,
    pub risk_threshold_for_modification: f64,
    pub max_risk_adjusted_steps: u32,
    pub mandatory_review_interval_days: u32,
}

impl Default for RiskAwareTitrationConfig {
    fn default() -> Self {
        Self {
            enable_risk_adjustment: true,
            enable_dynamic_monitoring: true,
            enable_safety_checkpoints: true,
            risk_threshold_for_modification: 0.6,
            max_risk_adjusted_steps: 8,
            mandatory_review_interval_days: 14,
        }
    }
}

/// Risk adjuster for different risk levels
#[derive(Debug, Clone)]
pub struct RiskAdjuster {
    pub dose_increment_modifier: f64,
    pub interval_extension_days: u32,
    pub additional_monitoring: Vec<String>,
    pub safety_checkpoint_frequency: u32,
    pub escalation_criteria: Vec<String>,
}

/// Risk-adjusted titration schedule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAdjustedTitrationSchedule {
    pub base_schedule: TitrationSchedule,
    pub risk_profile: CumulativeRiskProfile,
    pub risk_adjustments: Vec<RiskAdjustment>,
    pub enhanced_monitoring: EnhancedMonitoringPlan,
    pub safety_checkpoints: Vec<RiskAwareSafetyCheckpoint>,
    pub escalation_protocols: Vec<EscalationProtocol>,
    pub decision_support: Vec<ClinicalDecisionSupport>,
}

/// Individual risk adjustment applied to titration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAdjustment {
    pub adjustment_id: String,
    pub step_number: u32,
    pub adjustment_type: RiskAdjustmentType,
    pub original_value: f64,
    pub adjusted_value: f64,
    pub rationale: String,
    pub risk_factors_addressed: Vec<String>,
}

/// Types of risk adjustments
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RiskAdjustmentType {
    DoseReduction,
    IntervalExtension,
    AdditionalMonitoring,
    SafetyCheckpoint,
    AlternativeStrategy,
}

/// Enhanced monitoring plan with risk-based intensification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnhancedMonitoringPlan {
    pub base_monitoring: Vec<MonitoringRequirement>,
    pub risk_based_monitoring: Vec<RiskBasedMonitoring>,
    pub dynamic_adjustments: Vec<DynamicMonitoringAdjustment>,
    pub alert_thresholds: HashMap<String, AlertThreshold>,
}

/// Monitoring requirement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRequirement {
    pub parameter: String,
    pub frequency: String,
    pub method: String,
    pub target_range: Option<(f64, f64)>,
    pub rationale: String,
}

/// Risk-based monitoring intensification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskBasedMonitoring {
    pub risk_factor_id: String,
    pub monitoring_parameter: String,
    pub intensified_frequency: String,
    pub special_instructions: Vec<String>,
    pub duration: String,
}

/// Dynamic monitoring adjustment based on ongoing assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DynamicMonitoringAdjustment {
    pub trigger_condition: String,
    pub adjustment_action: String,
    pub new_frequency: String,
    pub duration: String,
    pub review_criteria: Vec<String>,
}

/// Alert threshold for monitoring parameters
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlertThreshold {
    pub parameter: String,
    pub warning_threshold: f64,
    pub critical_threshold: f64,
    pub action_required: String,
    pub escalation_timeframe: String,
}

/// Risk-aware safety checkpoint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskAwareSafetyCheckpoint {
    pub checkpoint_id: String,
    pub scheduled_date: DateTime<Utc>,
    pub step_number: u32,
    pub risk_level_at_checkpoint: RiskLevel,
    pub mandatory_assessments: Vec<MandatoryAssessment>,
    pub decision_criteria: Vec<DecisionCriterion>,
    pub possible_outcomes: Vec<CheckpointOutcome>,
}

/// Mandatory assessment at safety checkpoint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MandatoryAssessment {
    pub assessment_type: String,
    pub parameters: Vec<String>,
    pub completion_required: bool,
    pub timeout_hours: u32,
}

/// Decision criterion for checkpoint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DecisionCriterion {
    pub criterion_id: String,
    pub condition: String,
    pub threshold: Option<f64>,
    pub action_if_met: String,
    pub action_if_not_met: String,
}

/// Possible outcomes from safety checkpoint
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CheckpointOutcome {
    pub outcome_type: CheckpointOutcomeType,
    pub description: String,
    pub next_actions: Vec<String>,
    pub schedule_modifications: Vec<String>,
}

/// Types of checkpoint outcomes
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CheckpointOutcomeType {
    Continue,
    ModifySchedule,
    HoldTitration,
    ReduceDose,
    SwitchStrategy,
    Discontinue,
}

/// Escalation protocol for concerning findings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalationProtocol {
    pub protocol_id: String,
    pub trigger_conditions: Vec<String>,
    pub escalation_levels: Vec<EscalationLevel>,
    pub contact_information: Vec<ContactInfo>,
    pub documentation_requirements: Vec<String>,
}

/// Escalation level
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalationLevel {
    pub level: u32,
    pub timeframe: String,
    pub required_actions: Vec<String>,
    pub responsible_party: String,
    pub escalation_criteria: Vec<String>,
}

/// Contact information for escalation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContactInfo {
    pub role: String,
    pub contact_method: String,
    pub availability: String,
    pub escalation_level: u32,
}

/// Clinical decision support
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalDecisionSupport {
    pub decision_point: String,
    pub clinical_question: String,
    pub evidence_summary: String,
    pub recommendations: Vec<ClinicalRecommendation>,
    pub risk_benefit_analysis: RiskBenefitAnalysis,
}

/// Clinical recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRecommendation {
    pub recommendation_text: String,
    pub strength: RecommendationStrength,
    pub evidence_level: String,
    pub applicability: Vec<String>,
    pub contraindications: Vec<String>,
}

/// Strength of clinical recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationStrength {
    Strong,
    Moderate,
    Weak,
    Conditional,
}

/// Risk-benefit analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskBenefitAnalysis {
    pub expected_benefits: Vec<ExpectedBenefit>,
    pub potential_risks: Vec<PotentialRisk>,
    pub net_clinical_benefit: f64,
    pub uncertainty_factors: Vec<String>,
}

/// Expected benefit
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExpectedBenefit {
    pub benefit_type: String,
    pub probability: f64,
    pub magnitude: f64,
    pub time_to_benefit: String,
    pub evidence_quality: String,
}

/// Potential risk
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PotentialRisk {
    pub risk_type: String,
    pub probability: f64,
    pub severity: f64,
    pub time_to_risk: String,
    pub mitigation_strategies: Vec<String>,
}

impl RiskAwareTitrationEngine {
    /// Create a new risk-aware titration engine
    pub fn new() -> Result<Self> {
        let mut engine = Self {
            titration_engine: TitrationEngine::new()?,
            risk_assessment: CumulativeRiskAssessment::new(),
            risk_adjusters: HashMap::new(),
            config: RiskAwareTitrationConfig::default(),
        };
        
        // Initialize risk adjusters
        engine.initialize_risk_adjusters();
        
        info!("🎯 Risk-Aware Titration Engine initialized");
        
        Ok(engine)
    }
    
    /// Initialize risk adjusters for different risk levels
    fn initialize_risk_adjusters(&mut self) {
        // Low risk - minimal adjustments
        self.risk_adjusters.insert(RiskLevel::Low, RiskAdjuster {
            dose_increment_modifier: 1.0,
            interval_extension_days: 0,
            additional_monitoring: vec![],
            safety_checkpoint_frequency: 4, // Every 4 steps
            escalation_criteria: vec!["Unexpected adverse events".to_string()],
        });
        
        // Moderate risk - slight caution
        self.risk_adjusters.insert(RiskLevel::Moderate, RiskAdjuster {
            dose_increment_modifier: 0.8,
            interval_extension_days: 3,
            additional_monitoring: vec!["Enhanced symptom monitoring".to_string()],
            safety_checkpoint_frequency: 3, // Every 3 steps
            escalation_criteria: vec![
                "New symptoms".to_string(),
                "Lab value changes".to_string(),
            ],
        });
        
        // High risk - significant modifications
        self.risk_adjusters.insert(RiskLevel::High, RiskAdjuster {
            dose_increment_modifier: 0.6,
            interval_extension_days: 7,
            additional_monitoring: vec![
                "Daily symptom assessment".to_string(),
                "Weekly lab monitoring".to_string(),
                "Vital signs monitoring".to_string(),
            ],
            safety_checkpoint_frequency: 2, // Every 2 steps
            escalation_criteria: vec![
                "Any new symptoms".to_string(),
                "Any lab abnormalities".to_string(),
                "Patient concerns".to_string(),
            ],
        });
        
        // Very high risk - conservative approach
        self.risk_adjusters.insert(RiskLevel::VeryHigh, RiskAdjuster {
            dose_increment_modifier: 0.4,
            interval_extension_days: 14,
            additional_monitoring: vec![
                "Continuous monitoring".to_string(),
                "Daily lab checks".to_string(),
                "Specialist consultation".to_string(),
            ],
            safety_checkpoint_frequency: 1, // Every step
            escalation_criteria: vec![
                "Any change from baseline".to_string(),
                "Patient discomfort".to_string(),
                "Caregiver concerns".to_string(),
            ],
        });
    }
    
    /// Generate risk-adjusted titration schedule
    pub async fn generate_risk_adjusted_schedule(
        &self,
        titration_request: TitrationRequest,
    ) -> Result<RiskAdjustedTitrationSchedule> {
        info!("🎯 Generating risk-adjusted titration schedule for patient: {}", titration_request.patient_id);
        
        // First, generate base titration schedule
        let base_schedule = self.titration_engine.generate_titration_schedule(titration_request.clone())?;
        
        // Perform risk assessment
        let risk_context = self.create_risk_assessment_context(&titration_request)?;
        let risk_profile = self.risk_assessment.assess_cumulative_risk(risk_context).await?;
        
        info!("📊 Risk assessment completed: {:?} (Score: {:.2})", 
              risk_profile.risk_level, risk_profile.overall_risk_score);
        
        // Apply risk adjustments if needed
        let (adjusted_schedule, risk_adjustments) = if self.config.enable_risk_adjustment {
            self.apply_risk_adjustments(base_schedule, &risk_profile)?
        } else {
            (base_schedule.clone(), vec![])
        };
        
        // Create enhanced monitoring plan
        let enhanced_monitoring = self.create_enhanced_monitoring_plan(&adjusted_schedule, &risk_profile)?;
        
        // Generate safety checkpoints
        let safety_checkpoints = if self.config.enable_safety_checkpoints {
            self.generate_safety_checkpoints(&adjusted_schedule, &risk_profile)?
        } else {
            vec![]
        };
        
        // Create escalation protocols
        let escalation_protocols = self.create_escalation_protocols(&risk_profile)?;
        
        // Generate clinical decision support
        let decision_support = self.generate_clinical_decision_support(&adjusted_schedule, &risk_profile)?;
        
        let risk_adjusted_schedule = RiskAdjustedTitrationSchedule {
            base_schedule: adjusted_schedule,
            risk_profile,
            risk_adjustments,
            enhanced_monitoring,
            safety_checkpoints,
            escalation_protocols,
            decision_support,
        };
        
        info!("✅ Risk-adjusted titration schedule generated with {} adjustments", 
              risk_adjusted_schedule.risk_adjustments.len());
        
        Ok(risk_adjusted_schedule)
    }
    
    /// Create risk assessment context from titration request
    fn create_risk_assessment_context(
        &self,
        request: &TitrationRequest,
    ) -> Result<super::cumulative_risk::RiskAssessmentContext> {
        use super::cumulative_risk::{RiskAssessmentContext, MedicationContext, PatientRiskFactors};
        
        // Convert titration request to risk assessment context
        let medication_context = MedicationContext {
            drug_id: request.drug_id.clone(),
            drug_name: request.drug_id.clone(), // Simplified
            dose: request.current_dose,
            frequency: "daily".to_string(), // Simplified
            route: "oral".to_string(), // Simplified
            start_date: Utc::now(),
            duration: None,
            indication: "titration".to_string(),
        };
        
        let patient_risk_factors = PatientRiskFactors {
            age_years: request.patient_factors.age_years,
            weight_kg: request.patient_factors.weight_kg,
            renal_function: request.patient_factors.renal_function.egfr,
            hepatic_function: request.patient_factors.hepatic_function.child_pugh_class
                .clone()
                .unwrap_or_else(|| "A".to_string()),
            genetic_variants: vec![],
            smoking_status: "unknown".to_string(),
            alcohol_use: "unknown".to_string(),
            adherence_score: request.patient_factors.adherence_history.average_adherence_percent / 100.0,
        };
        
        Ok(RiskAssessmentContext {
            patient_id: request.patient_id.clone(),
            medications: vec![medication_context],
            patient_factors: patient_risk_factors,
            laboratory_values: HashMap::new(),
            comorbidities: request.patient_factors.comorbidities.clone(),
            assessment_timepoint: Utc::now(),
        })
    }
    
    /// Apply risk adjustments to base schedule
    fn apply_risk_adjustments(
        &self,
        mut base_schedule: TitrationSchedule,
        risk_profile: &CumulativeRiskProfile,
    ) -> Result<(TitrationSchedule, Vec<RiskAdjustment>)> {
        let mut adjustments = Vec::new();
        
        // Get risk adjuster for this risk level
        if let Some(adjuster) = self.risk_adjusters.get(&risk_profile.risk_level) {
            // Apply adjustments to each step
            for step in &mut base_schedule.steps {
                // Adjust dose increment
                if adjuster.dose_increment_modifier < 1.0 {
                    let original_dose = step.dose;
                    let original_change = step.dose_change;
                    
                    step.dose_change *= adjuster.dose_increment_modifier;
                    step.dose = step.dose - original_change + step.dose_change;
                    step.dose_change_percent = (step.dose_change / (step.dose - step.dose_change)) * 100.0;
                    
                    adjustments.push(RiskAdjustment {
                        adjustment_id: format!("dose_adj_{}", step.step_number),
                        step_number: step.step_number,
                        adjustment_type: RiskAdjustmentType::DoseReduction,
                        original_value: original_dose,
                        adjusted_value: step.dose,
                        rationale: format!("Dose reduced due to {} risk level", 
                                         format!("{:?}", risk_profile.risk_level).to_lowercase()),
                        risk_factors_addressed: risk_profile.risk_factors.iter()
                            .take(3)
                            .map(|rf| rf.factor_id.clone())
                            .collect(),
                    });
                }
                
                // Extend intervals
                if adjuster.interval_extension_days > 0 {
                    let original_date = step.scheduled_date;
                    step.scheduled_date = step.scheduled_date + Duration::days(adjuster.interval_extension_days as i64);
                    step.next_review_date = step.next_review_date + Duration::days(adjuster.interval_extension_days as i64);
                    
                    adjustments.push(RiskAdjustment {
                        adjustment_id: format!("interval_adj_{}", step.step_number),
                        step_number: step.step_number,
                        adjustment_type: RiskAdjustmentType::IntervalExtension,
                        original_value: original_date.timestamp() as f64,
                        adjusted_value: step.scheduled_date.timestamp() as f64,
                        rationale: format!("Interval extended by {} days due to risk level", 
                                         adjuster.interval_extension_days),
                        risk_factors_addressed: vec![],
                    });
                }
                
                // Add additional monitoring requirements
                for monitoring in &adjuster.additional_monitoring {
                    step.monitoring_requirements.push(monitoring.clone());
                }
                
                // Add safety warnings
                if matches!(risk_profile.risk_level, RiskLevel::High | RiskLevel::VeryHigh) {
                    step.safety_warnings.push(format!(
                        "Patient has {} risk level - enhanced monitoring required", 
                        format!("{:?}", risk_profile.risk_level).to_lowercase()
                    ));
                }
            }
            
            // Update total duration
            if let Some(last_step) = base_schedule.steps.last() {
                let duration_days = (last_step.scheduled_date - base_schedule.created_at).num_days();
                base_schedule.total_duration_weeks = (duration_days / 7) as u32;
            }
        }
        
        Ok((base_schedule, adjustments))
    }
    
    /// Create enhanced monitoring plan
    fn create_enhanced_monitoring_plan(
        &self,
        schedule: &TitrationSchedule,
        risk_profile: &CumulativeRiskProfile,
    ) -> Result<EnhancedMonitoringPlan> {
        let mut base_monitoring = vec![
            MonitoringRequirement {
                parameter: "Clinical symptoms".to_string(),
                frequency: "Each visit".to_string(),
                method: "Clinical assessment".to_string(),
                target_range: None,
                rationale: "Monitor therapeutic response and adverse effects".to_string(),
            },
        ];
        
        let mut risk_based_monitoring = Vec::new();
        let mut alert_thresholds = HashMap::new();
        
        // Add risk-specific monitoring
        for risk_factor in &risk_profile.risk_factors {
            if risk_factor.risk_score > 0.5 {
                risk_based_monitoring.push(RiskBasedMonitoring {
                    risk_factor_id: risk_factor.factor_id.clone(),
                    monitoring_parameter: "Enhanced safety monitoring".to_string(),
                    intensified_frequency: "Weekly".to_string(),
                    special_instructions: vec![
                        format!("Monitor for {}", risk_factor.description),
                        "Document any changes from baseline".to_string(),
                    ],
                    duration: "Throughout titration".to_string(),
                });
            }
        }
        
        // Add alert thresholds based on risk level
        match risk_profile.risk_level {
            RiskLevel::High | RiskLevel::VeryHigh => {
                alert_thresholds.insert("symptom_severity".to_string(), AlertThreshold {
                    parameter: "Symptom severity score".to_string(),
                    warning_threshold: 3.0,
                    critical_threshold: 5.0,
                    action_required: "Hold titration and reassess".to_string(),
                    escalation_timeframe: "24 hours".to_string(),
                });
            }
            _ => {}
        }
        
        Ok(EnhancedMonitoringPlan {
            base_monitoring,
            risk_based_monitoring,
            dynamic_adjustments: vec![],
            alert_thresholds,
        })
    }
    
    /// Generate safety checkpoints
    fn generate_safety_checkpoints(
        &self,
        schedule: &TitrationSchedule,
        risk_profile: &CumulativeRiskProfile,
    ) -> Result<Vec<RiskAwareSafetyCheckpoint>> {
        let mut checkpoints = Vec::new();
        
        if let Some(adjuster) = self.risk_adjusters.get(&risk_profile.risk_level) {
            let checkpoint_frequency = adjuster.safety_checkpoint_frequency;
            
            for (i, step) in schedule.steps.iter().enumerate() {
                if (i + 1) % checkpoint_frequency as usize == 0 {
                    checkpoints.push(RiskAwareSafetyCheckpoint {
                        checkpoint_id: format!("checkpoint_{}_{}", schedule.schedule_id, step.step_number),
                        scheduled_date: step.scheduled_date,
                        step_number: step.step_number,
                        risk_level_at_checkpoint: risk_profile.risk_level.clone(),
                        mandatory_assessments: vec![
                            MandatoryAssessment {
                                assessment_type: "Clinical evaluation".to_string(),
                                parameters: vec!["Symptoms", "Vital signs", "Functional status"].iter()
                                    .map(|s| s.to_string()).collect(),
                                completion_required: true,
                                timeout_hours: 48,
                            },
                        ],
                        decision_criteria: vec![
                            DecisionCriterion {
                                criterion_id: "safety_assessment".to_string(),
                                condition: "No concerning findings".to_string(),
                                threshold: None,
                                action_if_met: "Continue titration".to_string(),
                                action_if_not_met: "Hold and reassess".to_string(),
                            },
                        ],
                        possible_outcomes: vec![
                            CheckpointOutcome {
                                outcome_type: CheckpointOutcomeType::Continue,
                                description: "Safe to continue titration".to_string(),
                                next_actions: vec!["Proceed to next step".to_string()],
                                schedule_modifications: vec![],
                            },
                            CheckpointOutcome {
                                outcome_type: CheckpointOutcomeType::HoldTitration,
                                description: "Safety concerns identified".to_string(),
                                next_actions: vec!["Hold titration", "Specialist consultation"].iter()
                                    .map(|s| s.to_string()).collect(),
                                schedule_modifications: vec!["Extend current dose period".to_string()],
                            },
                        ],
                    });
                }
            }
        }
        
        Ok(checkpoints)
    }
    
    /// Create escalation protocols
    fn create_escalation_protocols(&self, risk_profile: &CumulativeRiskProfile) -> Result<Vec<EscalationProtocol>> {
        let mut protocols = Vec::new();
        
        // Create general escalation protocol
        protocols.push(EscalationProtocol {
            protocol_id: format!("escalation_{}", uuid::Uuid::new_v4()),
            trigger_conditions: vec![
                "Unexpected adverse events".to_string(),
                "Concerning lab values".to_string(),
                "Patient or caregiver concerns".to_string(),
            ],
            escalation_levels: vec![
                EscalationLevel {
                    level: 1,
                    timeframe: "4 hours".to_string(),
                    required_actions: vec!["Clinical assessment", "Documentation"].iter()
                        .map(|s| s.to_string()).collect(),
                    responsible_party: "Primary clinician".to_string(),
                    escalation_criteria: vec!["Unresolved concerns".to_string()],
                },
                EscalationLevel {
                    level: 2,
                    timeframe: "24 hours".to_string(),
                    required_actions: vec!["Specialist consultation", "Risk-benefit reassessment"].iter()
                        .map(|s| s.to_string()).collect(),
                    responsible_party: "Specialist".to_string(),
                    escalation_criteria: vec!["Serious safety concerns".to_string()],
                },
            ],
            contact_information: vec![
                ContactInfo {
                    role: "Primary clinician".to_string(),
                    contact_method: "Phone/Pager".to_string(),
                    availability: "24/7".to_string(),
                    escalation_level: 1,
                },
                ContactInfo {
                    role: "Specialist".to_string(),
                    contact_method: "Consultation system".to_string(),
                    availability: "Business hours".to_string(),
                    escalation_level: 2,
                },
            ],
            documentation_requirements: vec![
                "Incident description".to_string(),
                "Timeline of events".to_string(),
                "Actions taken".to_string(),
                "Outcome".to_string(),
            ],
        });
        
        Ok(protocols)
    }
    
    /// Generate clinical decision support
    fn generate_clinical_decision_support(
        &self,
        schedule: &TitrationSchedule,
        risk_profile: &CumulativeRiskProfile,
    ) -> Result<Vec<ClinicalDecisionSupport>> {
        let mut decision_support = Vec::new();
        
        // Add decision support for high-risk patients
        if matches!(risk_profile.risk_level, RiskLevel::High | RiskLevel::VeryHigh) {
            decision_support.push(ClinicalDecisionSupport {
                decision_point: "Titration initiation".to_string(),
                clinical_question: "Is titration appropriate for this high-risk patient?".to_string(),
                evidence_summary: "High-risk patients require careful consideration of risk-benefit ratio".to_string(),
                recommendations: vec![
                    ClinicalRecommendation {
                        recommendation_text: "Consider slower titration with enhanced monitoring".to_string(),
                        strength: RecommendationStrength::Strong,
                        evidence_level: "Expert consensus".to_string(),
                        applicability: vec!["High-risk patients".to_string()],
                        contraindications: vec![],
                    },
                ],
                risk_benefit_analysis: RiskBenefitAnalysis {
                    expected_benefits: vec![
                        ExpectedBenefit {
                            benefit_type: "Therapeutic improvement".to_string(),
                            probability: 0.7,
                            magnitude: 0.6,
                            time_to_benefit: "2-4 weeks".to_string(),
                            evidence_quality: "Moderate".to_string(),
                        },
                    ],
                    potential_risks: vec![
                        PotentialRisk {
                            risk_type: "Adverse events".to_string(),
                            probability: 0.3,
                            severity: 0.7,
                            time_to_risk: "1-2 weeks".to_string(),
                            mitigation_strategies: vec!["Enhanced monitoring".to_string()],
                        },
                    ],
                    net_clinical_benefit: 0.4,
                    uncertainty_factors: vec!["Individual patient variability".to_string()],
                },
            });
        }
        
        Ok(decision_support)
    }
}
