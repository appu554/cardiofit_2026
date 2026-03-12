//! Rule Engine - Executes TOML-based clinical rules for dose calculation and safety verification

use std::collections::HashMap;
use std::sync::Arc;
use anyhow::{Result, anyhow};
use serde::{Deserialize, Serialize};
use serde_json::Value;

use super::{
    ClinicalRequest, CalculationResult, SafetyResult, SafetyAction, SafetyFinding,
    CalculationStep, DrugKnowledge, PatientContext,
};
use super::knowledge_base::{
    KnowledgeBase, SafetyVerificationRules, RenalSafety, HepaticSafety,
    DrugInteractions, AbsoluteContraindications, MonitoringRequirement
};
use super::expression_parser::{ExpressionParser, MathExpression};
use super::expression_evaluator::{ExpressionEvaluator, EvaluationContext, EvaluationConfig};
use super::variable_substitution::VariableSubstitution;

/// Rule engine that executes TOML-based clinical logic with mathematical expression support
pub struct RuleEngine {
    knowledge_base: Arc<KnowledgeBase>,
    expression_evaluator: ExpressionEvaluator,
    variable_substitution: VariableSubstitution,
}

/// Rule execution context
#[derive(Debug, Clone)]
pub struct RuleContext {
    pub patient: PatientContext,
    pub drug_id: String,
    pub indication: String,
    pub variables: HashMap<String, f64>,
}

/// Dose calculation rule structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseCalculationRules {
    pub base_dose: BaseDoseRules,
    pub weight_adjustment: Option<WeightAdjustmentRules>,
    pub age_adjustment: Option<AgeAdjustmentRules>,
    pub indication_specific: Option<HashMap<String, IndicationRules>>,
    pub dose_limits: DoseLimitsRules,
}

/// Base dose rules with expression support
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BaseDoseRules {
    pub default_mg: DoseValue,
    pub calculation_method: String, // "fixed", "weight_based", "bsa_based", "expression"
    pub weight_factor_mg_per_kg: Option<DoseValue>,
    pub bsa_factor_mg_per_m2: Option<DoseValue>,
    pub expression: Option<String>, // Mathematical expression for complex calculations
}

/// A dose value that can be either a fixed number or a mathematical expression
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(untagged)]
pub enum DoseValue {
    Fixed(f64),
    Expression(String),
}

impl DoseValue {
    /// Evaluate the dose value in the context of a clinical request
    pub fn evaluate(&self, rule_engine: &RuleEngine, request: &ClinicalRequest) -> Result<f64> {
        match self {
            DoseValue::Fixed(value) => Ok(*value),
            DoseValue::Expression(expr) => rule_engine.evaluate_expression(expr, request),
        }
    }

    /// Check if this is a mathematical expression
    pub fn is_expression(&self) -> bool {
        matches!(self, DoseValue::Expression(_))
    }

    /// Get the raw value (for fixed values) or expression string
    pub fn raw_value(&self) -> String {
        match self {
            DoseValue::Fixed(value) => value.to_string(),
            DoseValue::Expression(expr) => expr.clone(),
        }
    }
}

/// Weight adjustment rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WeightAdjustmentRules {
    pub bands: Vec<WeightBand>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WeightBand {
    pub min_kg: f64,
    pub max_kg: f64,
    pub factor: f64,
    pub action: String, // "multiply", "add", "cap"
}

/// Age adjustment rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgeAdjustmentRules {
    pub bands: Vec<AgeBand>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgeBand {
    pub min_years: u8,
    pub max_years: u8,
    pub factor: f64,
    pub action: String, // "multiply", "add", "cap"
}

/// Indication-specific rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndicationRules {
    pub base_dose_mg: f64,
    pub weight_factor: Option<f64>,
    pub max_dose_mg: Option<f64>,
}

/// Dose limits rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseLimitsRules {
    pub absolute_min_mg: f64,
    pub absolute_max_mg: f64,
    pub max_mg_per_kg: Option<f64>,
    pub max_mg_per_day: Option<f64>,
}

/// Safety verification rules - UNIFIED STRUCTURE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyRules {
    pub renal: Option<RenalRules>,
    pub hepatic: Option<HepaticRules>,
    pub age_specific: Option<AgeSpecificRules>,
    pub pregnancy: Option<PregnancyRules>,
    pub interactions: Option<DrugInteractionRules>,  // UNIFIED: Use 'interactions' instead of 'drug_interactions'
    pub contraindications: Option<ContraindicationRules>,
    pub monitoring: Option<MonitoringRules>,
}

/// Renal function rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalRules {
    pub bands: Vec<RenalBand>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalBand {
    pub min_egfr: f64,
    pub max_egfr: f64,
    pub action: String, // "allow", "cap", "reduce", "block"
    pub dose_factor: Option<f64>,
    pub max_dose_mg: Option<f64>,
    pub message: String,
}

/// Hepatic function rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticRules {
    pub child_pugh_a: HepaticAction,
    pub child_pugh_b: HepaticAction,
    pub child_pugh_c: HepaticAction,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticAction {
    pub action: String, // "allow", "cap", "reduce", "block"
    pub dose_factor: Option<f64>,
    pub max_dose_mg: Option<f64>,
    pub message: String,
}

/// Age-specific rules (Beers Criteria, pediatric considerations)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgeSpecificRules {
    pub geriatric_threshold_years: u8,
    pub geriatric_action: AgeAction,
    pub pediatric_threshold_years: u8,
    pub pediatric_action: AgeAction,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgeAction {
    pub action: String,
    pub dose_factor: Option<f64>,
    pub max_dose_mg: Option<f64>,
    pub message: String,
    pub beers_criteria: Option<String>,
}

/// Pregnancy rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PregnancyRules {
    pub category: String, // "A", "B", "C", "D", "X"
    pub action: String,   // "allow", "caution", "avoid", "block"
    pub trimester_specific: Option<HashMap<u8, PregnancyAction>>,
    pub message: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PregnancyAction {
    pub action: String,
    pub message: String,
}

/// Drug interaction rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugInteractionRules {
    pub interactions: Vec<DrugInteraction>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugInteraction {
    pub interacting_drug: String,
    pub severity: String, // "minor", "moderate", "major", "contraindicated"
    pub action: String,   // "monitor", "adjust", "avoid"
    pub message: String,
    pub mechanism: String,
}

/// Contraindication rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContraindicationRules {
    pub absolute: Vec<Contraindication>,
    pub relative: Vec<Contraindication>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Contraindication {
    pub condition: String, // ICD-10 code or condition name
    pub severity: String,  // "absolute", "relative"
    pub message: String,
    pub evidence_level: String,
}

/// Monitoring rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRules {
    pub required_labs: Vec<LabMonitoring>,
    pub clinical_monitoring: Vec<ClinicalMonitoring>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabMonitoring {
    pub lab_name: String,
    pub frequency: String,
    pub threshold_action: Option<String>,
    pub critical_value: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalMonitoring {
    pub parameter: String,
    pub frequency: String,
    pub instructions: String,
}

impl RuleEngine {
    /// Create a new rule engine with mathematical expression support
    pub fn new(knowledge_base: Arc<KnowledgeBase>) -> Result<Self> {
        let expression_evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let variable_substitution = VariableSubstitution::new();

        Ok(Self {
            knowledge_base,
            expression_evaluator,
            variable_substitution,
        })
    }

    /// Calculate dose using rule-driven logic
    pub async fn calculate_dose(
        &self,
        request: &ClinicalRequest,
        drug_knowledge: &DrugKnowledge,
    ) -> Result<CalculationResult> {
        // Parse calculation rules from TOML
        let calc_rules: DoseCalculationRules = serde_json::from_value(
            drug_knowledge.calculation_rules.clone()
        )?;

        let mut context = RuleContext {
            patient: request.patient_context.clone(),
            drug_id: request.drug_id.clone(),
            indication: request.indication.clone(),
            variables: HashMap::new(),
        };

        let mut calculation_steps = Vec::new();
        let mut dose_mg = 0.0;

        // Step 1: Calculate base dose
        let base_dose_step = self.calculate_base_dose(&calc_rules.base_dose, &mut context)?;
        dose_mg = base_dose_step.result;
        calculation_steps.push(base_dose_step);

        // Step 2: Apply weight adjustments
        if let Some(weight_rules) = &calc_rules.weight_adjustment {
            let weight_step = self.apply_weight_adjustment(weight_rules, dose_mg, &mut context)?;
            dose_mg = weight_step.result;
            calculation_steps.push(weight_step);
        }

        // Step 3: Apply age adjustments
        if let Some(age_rules) = &calc_rules.age_adjustment {
            let age_step = self.apply_age_adjustment(age_rules, dose_mg, &mut context)?;
            dose_mg = age_step.result;
            calculation_steps.push(age_step);
        }

        // Step 4: Apply indication-specific adjustments
        if let Some(indication_rules) = &calc_rules.indication_specific {
            if let Some(specific_rules) = indication_rules.get(&request.indication) {
                let indication_step = self.apply_indication_adjustment(specific_rules, dose_mg, &mut context)?;
                dose_mg = indication_step.result;
                calculation_steps.push(indication_step);
            }
        }

        // Step 5: Apply dose limits
        let limits_step = self.apply_dose_limits(&calc_rules.dose_limits, dose_mg, &mut context)?;
        dose_mg = limits_step.result;
        calculation_steps.push(limits_step);

        Ok(CalculationResult {
            proposed_dose_mg: dose_mg,
            calculation_strategy: "standard_rules".to_string(),
            calculation_steps,
            confidence_score: 0.95, // High confidence for rule-based calculations
        })
    }

    /// Early safety screening for absolute contraindications (before dose calculation)
    pub async fn check_early_safety_contraindications(
        &self,
        request: &ClinicalRequest,
        drug_knowledge: &DrugKnowledge,
    ) -> Result<SafetyResult> {
        // Parse safety rules from TOML
        let safety_rules: SafetyVerificationRules = serde_json::from_value(
            drug_knowledge.safety_rules.clone()
        )?;

        let mut findings = Vec::new();
        let mut action = SafetyAction::Proceed;

        // Check ONLY absolute contraindications at this stage
        let (contra_action, contra_findings) =
            self.check_absolute_contraindications(&safety_rules.absolute_contraindications, &request.patient_context)?;

        findings.extend(contra_findings);
        action = self.merge_safety_actions(action, contra_action);

        // Return early safety result
        Ok(SafetyResult {
            action,
            findings,
            adjusted_dose_mg: None,
            monitoring_parameters: vec![],
        })
    }

    /// Verify safety using rule-driven logic (comprehensive check after dose calculation)
    pub async fn verify_safety(
        &self,
        request: &ClinicalRequest,
        calculation: &CalculationResult,
        drug_knowledge: &DrugKnowledge,
    ) -> Result<SafetyResult> {
        // Parse safety rules from TOML
        let safety_rules: SafetyVerificationRules = serde_json::from_value(
            drug_knowledge.safety_rules.clone()
        )?;

        let mut findings = Vec::new();
        let mut action = SafetyAction::Proceed;
        let mut adjusted_dose = None;
        let mut monitoring_params = Vec::new();

        // Check renal function
        if let Some(renal_rules) = &safety_rules.renal_safety {
            let (renal_action, renal_findings, renal_monitoring) =
                self.check_renal_safety_kb(renal_rules, calculation.proposed_dose_mg, &request.patient_context)?;

            findings.extend(renal_findings);
            monitoring_params.extend(renal_monitoring);

            // Update action if more restrictive
            action = self.merge_safety_actions(action, renal_action);
        }

        // Check hepatic function
        if let Some(hepatic_rules) = &safety_rules.hepatic_safety {
            let (hepatic_action, hepatic_findings, hepatic_monitoring) =
                self.check_hepatic_safety_kb(hepatic_rules, calculation.proposed_dose_mg, &request.patient_context)?;

            findings.extend(hepatic_findings);
            monitoring_params.extend(hepatic_monitoring);

            action = self.merge_safety_actions(action, hepatic_action);
        }

        // Check absolute contraindications
        let (contra_action, contra_findings) =
            self.check_absolute_contraindications(&safety_rules.absolute_contraindications, &request.patient_context)?;

        findings.extend(contra_findings);
        action = self.merge_safety_actions(action, contra_action);

        // Check drug interactions - UNIFIED STRUCTURE
        if let Some(interaction_rules) = &safety_rules.interactions {
            let (interaction_action, interaction_findings) =
                self.check_drug_interactions_kb(interaction_rules, &request.patient_context)?;

            findings.extend(interaction_findings);
            action = self.merge_safety_actions(action, interaction_action);
        }

        // Add monitoring requirements from knowledge base
        let monitoring_requirements = self.determine_monitoring_requirements(&safety_rules.monitoring_requirements)?;
        monitoring_params.extend(monitoring_requirements);

        // Extract adjusted dose if action requires it
        if let SafetyAction::AdjustDose { new_dose_mg, .. } = &action {
            adjusted_dose = Some(*new_dose_mg);
        }

        Ok(SafetyResult {
            action,
            findings,
            adjusted_dose_mg: adjusted_dose,
            monitoring_parameters: monitoring_params,
        })
    }

    // Helper methods for dose calculation
    fn calculate_base_dose(&self, rules: &BaseDoseRules, context: &mut RuleContext) -> Result<CalculationStep> {
        // Create a temporary request for expression evaluation
        let temp_request = ClinicalRequest {
            request_id: "temp".to_string(),
            patient_context: context.patient.clone(),
            drug_id: context.drug_id.clone(),
            indication: context.indication.clone(),
            timestamp: chrono::Utc::now(),
        };

        let dose = match rules.calculation_method.as_str() {
            "fixed" => rules.default_mg.evaluate(self, &temp_request)?,
            "weight_based" => {
                let weight_factor = if let Some(ref factor) = rules.weight_factor_mg_per_kg {
                    factor.evaluate(self, &temp_request)?
                } else {
                    1.0
                };
                context.patient.weight_kg * weight_factor
            },
            "bsa_based" => {
                let bsa = self.calculate_bsa(context.patient.weight_kg, context.patient.height_cm);
                let bsa_factor = if let Some(ref factor) = rules.bsa_factor_mg_per_m2 {
                    factor.evaluate(self, &temp_request)?
                } else {
                    1.0
                };
                bsa * bsa_factor
            },
            "expression" => {
                if let Some(ref expr) = rules.expression {
                    self.evaluate_expression(expr, &temp_request)?
                } else {
                    return Err(anyhow!("Expression calculation method specified but no expression provided"));
                }
            },
            _ => rules.default_mg.evaluate(self, &temp_request)?,
        };

        let mut input_values = HashMap::new();
        input_values.insert("weight_kg".to_string(), context.patient.weight_kg);
        input_values.insert("height_cm".to_string(), context.patient.height_cm);

        Ok(CalculationStep {
            step_name: "base_dose_calculation".to_string(),
            input_values,
            calculation: format!("method: {}, result: {:.2} mg", rules.calculation_method, dose),
            result: dose,
            rule_applied: Some("base_dose".to_string()),
        })
    }

    fn calculate_bsa(&self, weight_kg: f64, height_cm: f64) -> f64 {
        // Mosteller formula: BSA (m²) = √[(height(cm) × weight(kg)) / 3600]
        ((height_cm * weight_kg) / 3600.0).sqrt()
    }

    // Additional helper methods would be implemented here...
    // (apply_weight_adjustment, apply_age_adjustment, etc.)

    fn merge_safety_actions(&self, current: SafetyAction, new: SafetyAction) -> SafetyAction {
        // Implement logic to merge safety actions, choosing the most restrictive
        match (&current, &new) {
            (SafetyAction::Contraindicated { .. }, _) => current,
            (_, SafetyAction::Contraindicated { .. }) => new,
            (SafetyAction::RequireSpecialist { .. }, _) => current,
            (_, SafetyAction::RequireSpecialist { .. }) => new,
            (SafetyAction::Hold { .. }, _) => current,
            (_, SafetyAction::Hold { .. }) => new,
            (SafetyAction::AdjustDose { .. }, _) => current,
            (_, SafetyAction::AdjustDose { .. }) => new,
            _ => new,
        }
    }

    // Placeholder implementations for safety checks
    fn check_renal_safety(&self, _rules: &RenalRules, _dose: f64, _patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>, Vec<String>)> {
        Ok((SafetyAction::Proceed, vec![], vec![]))
    }

    fn check_hepatic_safety(&self, _rules: &HepaticRules, _dose: f64, _patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>, Vec<String>)> {
        Ok((SafetyAction::Proceed, vec![], vec![]))
    }

    fn check_pregnancy_safety(&self, _rules: &PregnancyRules, _patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>)> {
        Ok((SafetyAction::Proceed, vec![]))
    }

    fn check_drug_interactions(&self, _rules: &DrugInteractionRules, _patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>)> {
        Ok((SafetyAction::Proceed, vec![]))
    }

    fn check_contraindications(&self, _rules: &ContraindicationRules, _patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>)> {
        Ok((SafetyAction::Proceed, vec![]))
    }

    fn determine_monitoring_requirements(&self, _rules: &[MonitoringRequirement]) -> Result<Vec<String>> {
        Ok(vec![])
    }

    // Placeholder implementations for adjustment methods
    fn apply_weight_adjustment(&self, _rules: &WeightAdjustmentRules, dose: f64, _context: &mut RuleContext) -> Result<CalculationStep> {
        Ok(CalculationStep {
            step_name: "weight_adjustment".to_string(),
            input_values: HashMap::new(),
            calculation: "no adjustment applied".to_string(),
            result: dose,
            rule_applied: None,
        })
    }

    fn apply_age_adjustment(&self, _rules: &AgeAdjustmentRules, dose: f64, _context: &mut RuleContext) -> Result<CalculationStep> {
        Ok(CalculationStep {
            step_name: "age_adjustment".to_string(),
            input_values: HashMap::new(),
            calculation: "no adjustment applied".to_string(),
            result: dose,
            rule_applied: None,
        })
    }

    fn apply_indication_adjustment(&self, _rules: &IndicationRules, dose: f64, _context: &mut RuleContext) -> Result<CalculationStep> {
        Ok(CalculationStep {
            step_name: "indication_adjustment".to_string(),
            input_values: HashMap::new(),
            calculation: "no adjustment applied".to_string(),
            result: dose,
            rule_applied: None,
        })
    }

    fn apply_dose_limits(&self, rules: &DoseLimitsRules, dose: f64, _context: &mut RuleContext) -> Result<CalculationStep> {
        let final_dose = dose.max(rules.absolute_min_mg).min(rules.absolute_max_mg);
        
        Ok(CalculationStep {
            step_name: "dose_limits".to_string(),
            input_values: HashMap::new(),
            calculation: format!("applied limits: min={}, max={}, final={:.2}", 
                rules.absolute_min_mg, rules.absolute_max_mg, final_dose),
            result: final_dose,
            rule_applied: Some("dose_limits".to_string()),
        })
    }

    // New methods that work with knowledge base structures
    fn check_renal_safety_kb(&self, rules: &RenalSafety, dose: f64, patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>, Vec<String>)> {
        let mut findings = Vec::new();
        let mut monitoring = Vec::new();
        let mut action = SafetyAction::Proceed;

        // Get patient's eGFR
        let egfr = patient.renal_function.egfr_ml_min_1_73m2.unwrap_or(90.0);

        // Check each renal safety band
        for band in &rules.bands {
            if egfr >= band.min_egfr && egfr <= band.max_egfr {
                match band.action.as_str() {
                    "contraindicated" => {
                        findings.push(SafetyFinding {
                            finding_id: "RENAL_CONTRAINDICATION".to_string(),
                            category: "contraindication".to_string(),
                            severity: "Critical".to_string(),
                            message: format!("eGFR {:.1} mL/min/1.73m²: {}", egfr, band.reason),
                            evidence_level: "high".to_string(),
                            references: vec!["KDIGO-Guidelines".to_string()],
                        });
                        action = SafetyAction::Contraindicated {
                            reason: band.reason.clone(),
                            alternatives: vec!["Select renally-safe alternative medication".to_string()]
                        };
                    },
                    "dose_reduce" => {
                        if let Some(max_dose) = band.max_dose_mg_per_day {
                            if dose > max_dose {
                                findings.push(SafetyFinding {
                                    finding_id: "RENAL_DOSE_ADJUSTMENT".to_string(),
                                    category: "dose_adjustment".to_string(),
                                    severity: "Warning".to_string(),
                                    message: format!("eGFR {:.1} mL/min/1.73m²: Dose reduction required. Max dose: {} mg/day", egfr, max_dose),
                                    evidence_level: "moderate".to_string(),
                                    references: vec!["Renal-Dosing-Guidelines".to_string()],
                                });
                                action = SafetyAction::AdjustDose {
                                    new_dose_mg: max_dose,
                                    reason: format!("Renal dose adjustment for eGFR {:.1}", egfr)
                                };
                            }
                        }
                    },
                    "monitor_closely" => {
                        findings.push(SafetyFinding {
                            finding_id: "RENAL_MONITORING_REQUIRED".to_string(),
                            category: "monitoring".to_string(),
                            severity: "Warning".to_string(),
                            message: format!("eGFR {:.1} mL/min/1.73m²: Close monitoring required", egfr),
                            evidence_level: "moderate".to_string(),
                            references: vec!["Renal-Monitoring-Guidelines".to_string()],
                        });
                        if action == SafetyAction::Proceed {
                            action = SafetyAction::ProceedWithMonitoring {
                                parameters: vec!["renal_function".to_string(), "creatinine".to_string()]
                            };
                        }
                    },
                    _ => {}
                }

                // Add monitoring requirements
                if let Some(ref monitoring_required) = band.monitoring_required {
                    monitoring.extend(monitoring_required.clone());
                }
                break;
            }
        }

        // Special check for multi-organ failure (renal + hepatic)
        if egfr < 30.0 && patient.hepatic_function.child_pugh_class.as_ref().map_or(false, |c| c == "C" || c == "B") {
            findings.push(SafetyFinding {
                finding_id: "MULTI_ORGAN_FAILURE".to_string(),
                category: "contraindication".to_string(),
                severity: "Critical".to_string(),
                message: "Multi-organ failure detected (renal + hepatic impairment)".to_string(),
                evidence_level: "high".to_string(),
                references: vec!["Multi-Organ-Failure-Guidelines".to_string()],
            });
            if !matches!(action, SafetyAction::Contraindicated { .. }) {
                action = SafetyAction::RequireSpecialist {
                    specialty: "nephrology".to_string(),
                    urgency: "urgent".to_string(),
                };
            }
        }

        Ok((action, findings, monitoring))
    }

    fn check_hepatic_safety_kb(&self, rules: &HepaticSafety, dose: f64, patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>, Vec<String>)> {
        let mut findings = Vec::new();
        let mut monitoring = Vec::new();
        let mut action = SafetyAction::Proceed;

        // Check Child-Pugh class if available
        if let Some(ref child_pugh) = patient.hepatic_function.child_pugh_class {
            match child_pugh.as_str() {
                "C" => {
                    findings.push(SafetyFinding {
                        finding_id: "SEVERE_HEPATIC_IMPAIRMENT".to_string(),
                        category: "contraindication".to_string(),
                        severity: "Critical".to_string(),
                        message: "Child-Pugh Class C: Severe hepatic impairment".to_string(),
                        evidence_level: "high".to_string(),
                        references: vec!["Child-Pugh-Guidelines".to_string()],
                    });
                    action = SafetyAction::RequireSpecialist {
                        specialty: "hepatology".to_string(),
                        urgency: "urgent".to_string(),
                    };
                },
                "B" => {
                    findings.push(SafetyFinding {
                        finding_id: "MODERATE_HEPATIC_IMPAIRMENT".to_string(),
                        category: "monitoring".to_string(),
                        severity: "Warning".to_string(),
                        message: "Child-Pugh Class B: Moderate hepatic impairment - dose adjustment may be required".to_string(),
                        evidence_level: "moderate".to_string(),
                        references: vec!["Hepatic-Dosing-Guidelines".to_string()],
                    });
                    if action == SafetyAction::Proceed {
                        action = SafetyAction::ProceedWithMonitoring {
                            parameters: vec!["hepatic_function".to_string(), "liver_enzymes".to_string()]
                        };
                    }
                    monitoring.push("hepatic_function".to_string());
                },
                _ => {}
            }
        }

        // Check elevated liver enzymes
        if let Some(alt) = patient.hepatic_function.alt_u_l {
            if alt > 120.0 { // 3x upper limit of normal (assuming ULN ~40)
                findings.push(SafetyFinding {
                    finding_id: "ELEVATED_LIVER_ENZYMES".to_string(),
                    category: "monitoring".to_string(),
                    severity: "Warning".to_string(),
                    message: format!("Elevated ALT: {:.1} U/L", alt),
                    evidence_level: "moderate".to_string(),
                    references: vec!["Liver-Function-Guidelines".to_string()],
                });
                if action == SafetyAction::Proceed {
                    action = SafetyAction::ProceedWithMonitoring {
                        parameters: vec!["liver_enzymes".to_string()]
                    };
                }
                monitoring.push("liver_enzymes".to_string());
            }
        }

        if let Some(ast) = patient.hepatic_function.ast_u_l {
            if ast > 120.0 { // 3x upper limit of normal
                findings.push(SafetyFinding {
                    finding_id: "ELEVATED_LIVER_ENZYMES".to_string(),
                    category: "monitoring".to_string(),
                    severity: "Warning".to_string(),
                    message: format!("Elevated AST: {:.1} U/L", ast),
                    evidence_level: "moderate".to_string(),
                    references: vec!["Liver-Function-Guidelines".to_string()],
                });
                if action == SafetyAction::Proceed {
                    action = SafetyAction::ProceedWithMonitoring {
                        parameters: vec!["liver_enzymes".to_string()]
                    };
                }
                monitoring.push("liver_enzymes".to_string());
            }
        }

        Ok((action, findings, monitoring))
    }

    fn check_absolute_contraindications(&self, rules: &AbsoluteContraindications, patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>)> {
        let mut findings = Vec::new();
        let mut action = SafetyAction::Proceed;

        // Check pregnancy contraindication
        if rules.pregnancy {
            match &patient.pregnancy_status {
                super::PregnancyStatus::Pregnant { .. } => {
                    findings.push(SafetyFinding {
                        finding_id: "PREGNANCY_CONTRAINDICATION".to_string(),
                        category: "contraindication".to_string(),
                        severity: "Critical".to_string(),
                        message: "This medication is contraindicated in pregnancy".to_string(),
                        evidence_level: "high".to_string(),
                        references: vec!["FDA-Pregnancy-Guidelines".to_string()],
                    });
                    action = SafetyAction::Contraindicated {
                        reason: "Contraindicated in pregnancy".to_string(),
                        alternatives: vec!["Consult specialist for pregnancy-safe alternatives".to_string()]
                    };
                },
                _ => {}
            }
        }

        // Check allergy contraindications
        for allergy_class in &rules.allergy_classes {
            for patient_allergy in &patient.allergies {
                if patient_allergy.allergen.to_lowercase().contains(&allergy_class.to_lowercase()) ||
                   allergy_class.to_lowercase().contains(&patient_allergy.allergen.to_lowercase()) {
                    findings.push(SafetyFinding {
                        finding_id: "ALLERGY_CONTRAINDICATION".to_string(),
                        category: "contraindication".to_string(),
                        severity: "Critical".to_string(),
                        message: format!("Patient has documented allergy to {}", allergy_class),
                        evidence_level: "high".to_string(),
                        references: vec!["Patient-Allergy-History".to_string()],
                    });
                    action = SafetyAction::Contraindicated {
                        reason: format!("Patient allergic to {}", allergy_class),
                        alternatives: vec!["Review allergy history and select alternative medication".to_string()]
                    };
                    break;
                }
            }
        }

        // Check condition contraindications
        for contraindicated_condition in &rules.conditions {
            for patient_condition in &patient.conditions {
                if patient_condition.to_lowercase().contains(&contraindicated_condition.to_lowercase()) ||
                   contraindicated_condition.to_lowercase().contains(&patient_condition.to_lowercase()) {
                    findings.push(SafetyFinding {
                        finding_id: "CONDITION_CONTRAINDICATION".to_string(),
                        category: "contraindication".to_string(),
                        severity: "Critical".to_string(),
                        message: format!("Medication contraindicated with condition: {}", contraindicated_condition),
                        evidence_level: "high".to_string(),
                        references: vec!["Clinical-Guidelines".to_string()],
                    });
                    action = SafetyAction::Contraindicated {
                        reason: format!("Contraindicated with {}", contraindicated_condition),
                        alternatives: vec!["Select alternative medication appropriate for patient's condition".to_string()]
                    };
                    break;
                }
            }
        }

        Ok((action, findings))
    }

    fn check_drug_interactions_kb(&self, rules: &DrugInteractions, patient: &PatientContext) -> Result<(SafetyAction, Vec<SafetyFinding>)> {
        let mut findings = Vec::new();
        let mut action = SafetyAction::Proceed;

        // Get patient's current medications
        let patient_drug_ids: Vec<String> = patient.active_medications
            .iter()
            .map(|med| med.drug_id.clone())
            .collect();

        // Check major interactions
        for interaction in &rules.major {
            for interacting_class in &interaction.interacting_drug_classes {
                // Check if patient is on any drugs in this class
                for patient_drug in &patient_drug_ids {
                    if patient_drug.to_lowercase().contains(&interacting_class.to_lowercase()) ||
                       interacting_class.to_lowercase().contains(&patient_drug.to_lowercase()) {

                        match interaction.action.as_str() {
                            "avoid_concurrent_use" => {
                                findings.push(SafetyFinding {
                                    finding_id: "MAJOR_DRUG_INTERACTION".to_string(),
                                    category: "interaction".to_string(),
                                    severity: "Critical".to_string(),
                                    message: format!("Major interaction with {}: {}", interacting_class, interaction.reason),
                                    evidence_level: "high".to_string(),
                                    references: vec!["Drug-Interaction-Database".to_string()],
                                });
                                action = SafetyAction::Contraindicated {
                                    reason: format!("Major interaction with {}", interacting_class),
                                    alternatives: vec!["Select alternative medication without interaction".to_string()]
                                };
                            },
                            "dose_adjust" => {
                                findings.push(SafetyFinding {
                                    finding_id: "DRUG_INTERACTION_DOSE_ADJUST".to_string(),
                                    category: "interaction".to_string(),
                                    severity: "Warning".to_string(),
                                    message: format!("Interaction with {}: {}", interacting_class, interaction.reason),
                                    evidence_level: "moderate".to_string(),
                                    references: vec!["Drug-Interaction-Database".to_string()],
                                });
                                if action == SafetyAction::Proceed {
                                    action = SafetyAction::ProceedWithMonitoring {
                                        parameters: vec!["drug_levels".to_string(), "clinical_response".to_string()]
                                    };
                                }
                            },
                            "monitor_closely" => {
                                findings.push(SafetyFinding {
                                    finding_id: "DRUG_INTERACTION_MONITORING".to_string(),
                                    category: "interaction".to_string(),
                                    severity: "Warning".to_string(),
                                    message: format!("Interaction with {}: {}", interacting_class, interaction.reason),
                                    evidence_level: "moderate".to_string(),
                                    references: vec!["Drug-Interaction-Database".to_string()],
                                });
                                if action == SafetyAction::Proceed {
                                    action = SafetyAction::ProceedWithMonitoring {
                                        parameters: vec!["clinical_response".to_string(), "adverse_effects".to_string()]
                                    };
                                }
                            },
                            _ => {}
                        }
                        break;
                    }
                }
            }
        }

        // Check moderate interactions
        for interaction in &rules.moderate {
            for interacting_class in &interaction.interacting_drug_classes {
                for patient_drug in &patient_drug_ids {
                    if patient_drug.to_lowercase().contains(&interacting_class.to_lowercase()) ||
                       interacting_class.to_lowercase().contains(&patient_drug.to_lowercase()) {

                        findings.push(SafetyFinding {
                            finding_id: "MODERATE_DRUG_INTERACTION".to_string(),
                            category: "interaction".to_string(),
                            severity: "Warning".to_string(),
                            message: format!("Moderate interaction with {}: {}", interacting_class, interaction.reason),
                            evidence_level: "moderate".to_string(),
                            references: vec!["Drug-Interaction-Database".to_string()],
                        });

                        if action == SafetyAction::Proceed {
                            action = SafetyAction::ProceedWithMonitoring {
                                parameters: vec!["interaction_effects".to_string()]
                            };
                        }
                        break;
                    }
                }
            }
        }

        Ok((action, findings))
    }



    /// Evaluate a mathematical expression in the context of a clinical request
    pub fn evaluate_expression(&self, expression: &str, request: &ClinicalRequest) -> Result<f64> {
        // Parse the mathematical expression
        let parsed_expr = ExpressionParser::parse(expression)?;

        // Create variable substitution from patient context
        let substitution = self.variable_substitution.create_substitution(request)?;

        // Create evaluation context
        let eval_context = EvaluationContext {
            variables: substitution.variables,
            config: EvaluationConfig::default(),
        };

        // Evaluate the expression
        let result = self.expression_evaluator.evaluate(&parsed_expr, &eval_context)?;

        Ok(result.value)
    }

    /// Check if a string contains a mathematical expression (contains operators or functions)
    pub fn is_mathematical_expression(value: &str) -> bool {
        // Simple heuristic: if it contains mathematical operators or function calls, treat as expression
        value.contains('+') || value.contains('-') || value.contains('*') || value.contains('/') ||
        value.contains('^') || value.contains('(') || value.contains('?') || value.contains(':') ||
        value.contains("min") || value.contains("max") || value.contains("sqrt") ||
        value.contains("age") || value.contains("weight") || value.contains("egfr")
    }

    /// Evaluate a value that might be a number or mathematical expression
    pub fn evaluate_numeric_value(&self, value: &str, request: &ClinicalRequest) -> Result<f64> {
        // Try to parse as a simple number first
        if let Ok(number) = value.parse::<f64>() {
            return Ok(number);
        }

        // If not a simple number, treat as mathematical expression
        if Self::is_mathematical_expression(value) {
            self.evaluate_expression(value, request)
        } else {
            Err(anyhow!("Invalid numeric value or expression: {}", value))
        }
    }
}