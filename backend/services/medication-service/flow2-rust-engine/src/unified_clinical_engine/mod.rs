//! Unified Clinical Engine - Rule-Driven by Default, Compiled for Complexity
//! 
//! This module implements the definitive hybrid architecture that combines:
//! 1. Rule-driven dose calculation and safety verification (95% of cases)
//! 2. Compiled models for complex mathematical scenarios (5% of cases)

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use anyhow::{Result, anyhow};

pub mod rule_engine;
pub mod compiled_models;
pub mod knowledge_base;
pub mod monitoring;
pub mod model_sandbox;
pub mod advanced_validation;
pub mod hot_loader;
pub mod parallel_rule_engine;
pub mod titration_engine;
pub mod cumulative_risk;
pub mod risk_aware_titration;
pub mod expression_parser;
pub mod expression_evaluator;
pub mod variable_substitution;
pub mod expression_validator;

// Tests are in the tests/ directory

use rule_engine::RuleEngine;
use compiled_models::CompiledModelRegistry;
use knowledge_base::KnowledgeBase;
use model_sandbox::{ModelSandbox, SandboxExecutionResult};
use advanced_validation::{AdvancedValidator, ValidationResult};
use hot_loader::{HotLoader, DeploymentStatus};
use parallel_rule_engine::{ParallelRuleEngine, RuleDefinition};
use titration_engine::{TitrationEngine, TitrationSchedule, TitrationRequest};
use cumulative_risk::{CumulativeRiskAssessment, CumulativeRiskProfile};
use risk_aware_titration::{RiskAwareTitrationEngine, RiskAdjustedTitrationSchedule};
use expression_parser::{ExpressionParser, MathExpression};
use expression_evaluator::{ExpressionEvaluator, EvaluationContext, EvaluationConfig, EvaluationResult};
use variable_substitution::{VariableSubstitution, SubstitutionResult};

/// Unified Clinical Engine that handles both dose calculation and safety verification
pub struct UnifiedClinicalEngine {
    rule_engine: RuleEngine,
    parallel_rule_engine: ParallelRuleEngine,
    compiled_models: CompiledModelRegistry,
    model_sandbox: ModelSandbox,
    advanced_validator: AdvancedValidator,
    hot_loader: Option<HotLoader>,
    titration_engine: TitrationEngine,
    cumulative_risk_engine: CumulativeRiskAssessment,
    risk_aware_titration: RiskAwareTitrationEngine,
    knowledge_base: Arc<KnowledgeBase>,
    expression_evaluator: ExpressionEvaluator,
    variable_substitution: VariableSubstitution,
}

/// Clinical request containing patient context and drug information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRequest {
    pub request_id: String,
    pub drug_id: String,
    pub indication: String,
    pub patient_context: PatientContext,
    pub timestamp: DateTime<Utc>,
}

/// Patient clinical context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientContext {
    pub age_years: f64,
    pub weight_kg: f64,
    pub height_cm: f64,
    pub sex: BiologicalSex,
    pub pregnancy_status: PregnancyStatus,
    pub renal_function: RenalFunction,
    pub hepatic_function: HepaticFunction,
    pub active_medications: Vec<ActiveMedication>,
    pub allergies: Vec<Allergy>,
    pub conditions: Vec<String>, // ICD-10 codes
    pub lab_values: HashMap<String, LabValue>,
}

/// Clinical response with calculated dose and safety verification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalResponse {
    pub request_id: String,
    pub drug_id: String,
    pub calculation_result: CalculationResult,
    pub safety_result: SafetyResult,
    pub final_recommendation: FinalRecommendation,
    pub audit_trail: AuditTrail,
    pub processing_time_ms: u64,
}

/// Dose calculation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalculationResult {
    pub proposed_dose_mg: f64,
    pub calculation_strategy: String,
    pub calculation_steps: Vec<CalculationStep>,
    pub confidence_score: f64,
}

/// Safety verification result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyResult {
    pub action: SafetyAction,
    pub findings: Vec<SafetyFinding>,
    pub adjusted_dose_mg: Option<f64>,
    pub monitoring_parameters: Vec<String>,
}

/// Final clinical recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FinalRecommendation {
    pub action: RecommendationAction,
    pub final_dose_mg: f64,
    pub route: String,
    pub frequency: String,
    pub duration: Option<String>,
    pub special_instructions: Vec<String>,
    pub monitoring_required: Vec<String>,
    pub contraindications: Vec<String>,
    pub alternatives: Vec<AlternativeDrug>,
}

/// Calculation strategy enumeration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum CalculationStrategy {
    StandardRules,
    CustomModel(String),
}

/// Safety action enumeration
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum SafetyAction {
    Proceed,
    ProceedWithMonitoring { parameters: Vec<String> },
    AdjustDose { new_dose_mg: f64, reason: String },
    Hold { reason: String, duration: Option<String> },
    Contraindicated { reason: String, alternatives: Vec<String> },
    RequireSpecialist { specialty: String, urgency: String },
}

/// Recommendation action enumeration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum RecommendationAction {
    Prescribe,
    PrescribeWithMonitoring,
    RequireApproval,
    Contraindicated,
    ReferToSpecialist,
}

// Supporting types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum BiologicalSex { Male, Female, Other }

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum PregnancyStatus { 
    NotPregnant, 
    Pregnant { trimester: u8 }, 
    Unknown 
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalFunction {
    pub egfr_ml_min: Option<f64>,
    pub egfr_ml_min_1_73m2: Option<f64>,
    pub creatinine_clearance: Option<f64>,
    pub creatinine_mg_dl: Option<f64>,
    pub bun_mg_dl: Option<f64>,
    pub stage: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticFunction {
    pub child_pugh_class: Option<String>,
    pub alt_u_l: Option<f64>,
    pub ast_u_l: Option<f64>,
    pub bilirubin_mg_dl: Option<f64>,
    pub albumin_g_dl: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActiveMedication {
    pub drug_id: String,
    pub dose_mg: f64,
    pub frequency: String,
    pub route: String,
    pub start_date: DateTime<Utc>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Allergy {
    pub allergen: String,
    pub reaction_type: String,
    pub severity: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabValue {
    pub value: f64,
    pub unit: String,
    pub timestamp: DateTime<Utc>,
    pub reference_range: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalculationStep {
    pub step_name: String,
    pub input_values: HashMap<String, f64>,
    pub calculation: String,
    pub result: f64,
    pub rule_applied: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyFinding {
    pub finding_id: String,
    pub category: String,
    pub severity: String,
    pub message: String,
    pub evidence_level: String,
    pub references: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlternativeDrug {
    pub drug_id: String,
    pub name: String,
    pub rationale: String,
    pub relative_efficacy: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuditTrail {
    pub calculation_kb_version: String,
    pub safety_kb_version: String,
    pub engine_version: String,
    pub processing_steps: Vec<String>,
    pub rule_files_loaded: Vec<String>,
    pub models_invoked: Vec<String>,
}

impl UnifiedClinicalEngine {
    /// Create a new unified clinical engine with advanced features
    pub fn new(knowledge_base: Arc<KnowledgeBase>) -> Result<Self> {
        let rule_engine = RuleEngine::new(knowledge_base.clone())?;
        let parallel_rule_engine = ParallelRuleEngine::new()?;
        let compiled_models = CompiledModelRegistry::new()?;
        let model_sandbox = ModelSandbox::new();
        let advanced_validator = AdvancedValidator::new();
        let titration_engine = TitrationEngine::new()?;
        let cumulative_risk_engine = CumulativeRiskAssessment::new();
        let risk_aware_titration = RiskAwareTitrationEngine::new()?;

        // Initialize expression evaluation components
        let expression_evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let variable_substitution = VariableSubstitution::new();

        // Hot-loader is optional for production environments
        let hot_loader = if std::env::var("ENABLE_HOT_LOADING").unwrap_or_default() == "true" {
            Some(HotLoader::new(knowledge_base.clone(), Arc::new(compiled_models.clone()))?)
        } else {
            None
        };

        Ok(Self {
            rule_engine,
            parallel_rule_engine,
            compiled_models,
            model_sandbox,
            advanced_validator,
            hot_loader,
            titration_engine,
            cumulative_risk_engine,
            risk_aware_titration,
            knowledge_base,
            expression_evaluator,
            variable_substitution,
        })
    }

    /// Process a clinical request through the unified workflow
    pub async fn process_clinical_request(&self, request: ClinicalRequest) -> Result<ClinicalResponse> {
        let start_time = std::time::Instant::now();
        
        // Step 1: Load drug knowledge
        let drug_rules = self.knowledge_base.get_drug_rules(&request.drug_id)
            .ok_or_else(|| anyhow!("Drug rules not found for: {}", request.drug_id))?;

        // Convert to DrugKnowledge format
        let drug_knowledge = DrugKnowledge {
            meta: DrugMeta {
                drug_id: request.drug_id.clone(),
                name: drug_rules.meta.generic_name.clone(),
                generic_name: drug_rules.meta.generic_name.clone(),
                calculation_strategy: "standard".to_string(),
                therapeutic_class: vec![],
                mechanism_of_action: vec![],
            },
            calculation_rules: serde_json::to_value(&drug_rules.dose_calculation)?,
            safety_rules: serde_json::to_value(&drug_rules.safety_verification)?,
            default_route: "oral".to_string(),
            default_frequency: "daily".to_string(),
            default_duration: None,
            special_instructions: vec![],
            alternatives: vec![],
            calculation_version: drug_rules.meta.version.clone(),
            safety_version: drug_rules.meta.version.clone(),
        };
        
        // Step 2: Early safety screening for absolute contraindications
        let early_safety_result = self.rule_engine.check_early_safety_contraindications(
            &request,
            &drug_knowledge
        ).await?;

        // If there are absolute contraindications, return immediately without dose calculation
        if let SafetyAction::Contraindicated { .. } = early_safety_result.action {
            let final_recommendation = self.generate_final_recommendation(
                &request,
                &CalculationResult {
                    proposed_dose_mg: 0.0,
                    calculation_strategy: "blocked_by_safety".to_string(),
                    calculation_steps: vec![],
                    confidence_score: 0.0,
                },
                &early_safety_result,
                &drug_knowledge,
            )?;

            let audit_trail = self.create_audit_trail(&request, &drug_knowledge, &CalculationStrategy::StandardRules)?;
            let processing_time = start_time.elapsed().as_millis().max(1) as u64;

            return Ok(ClinicalResponse {
                request_id: request.request_id,
                drug_id: request.drug_id,
                calculation_result: CalculationResult {
                    proposed_dose_mg: 0.0,
                    calculation_strategy: "blocked_by_safety".to_string(),
                    calculation_steps: vec![],
                    confidence_score: 0.0,
                },
                safety_result: early_safety_result,
                final_recommendation,
                audit_trail,
                processing_time_ms: processing_time,
            });
        }

        // Step 3: Determine calculation strategy
        let calculation_strategy = self.determine_calculation_strategy(&drug_knowledge)?;

        // Step 4: Calculate dose based on strategy (only if safe to proceed)
        let calculation_result = match calculation_strategy {
            CalculationStrategy::StandardRules => {
                self.rule_engine.calculate_dose(&request, &drug_knowledge).await?
            },
            CalculationStrategy::CustomModel(ref model_name) => {
                self.compiled_models.calculate_dose(model_name, &request).await?
            },
        };

        // Step 5: Perform comprehensive safety verification (always rule-driven)
        let safety_result = self.rule_engine.verify_safety(
            &request,
            &calculation_result,
            &drug_knowledge
        ).await?;
        
        // Step 6: Generate final recommendation
        let final_recommendation = self.generate_final_recommendation(
            &request,
            &calculation_result,
            &safety_result,
            &drug_knowledge,
        )?;

        // Step 7: Create audit trail
        let audit_trail = self.create_audit_trail(&request, &drug_knowledge, &calculation_strategy)?;
        
        let processing_time = start_time.elapsed().as_millis().max(1) as u64;
        
        Ok(ClinicalResponse {
            request_id: request.request_id,
            drug_id: request.drug_id,
            calculation_result,
            safety_result,
            final_recommendation,
            audit_trail,
            processing_time_ms: processing_time,
        })
    }

    /// Determine which calculation strategy to use for a drug
    fn determine_calculation_strategy(&self, drug_knowledge: &DrugKnowledge) -> Result<CalculationStrategy> {
        // DEBUG: Log what we're actually getting
        tracing::info!("🔍 DEBUG: Drug ID: {}, Calculation Strategy: '{}'",
                      drug_knowledge.meta.drug_id,
                      drug_knowledge.meta.calculation_strategy);

        match drug_knowledge.meta.calculation_strategy.as_str() {
            "standard_rules" => Ok(CalculationStrategy::StandardRules),
            "standard" => {
                tracing::warn!("⚠️ Found 'standard' strategy, converting to 'standard_rules'");
                Ok(CalculationStrategy::StandardRules)
            },
            strategy if strategy.starts_with("custom_model_") => {
                Ok(CalculationStrategy::CustomModel(strategy.to_string()))
            },
            unknown => Err(anyhow!("Unknown calculation strategy: {}", unknown)),
        }
    }

    /// Generate final clinical recommendation
    fn generate_final_recommendation(
        &self,
        request: &ClinicalRequest,
        calculation: &CalculationResult,
        safety: &SafetyResult,
        drug_knowledge: &DrugKnowledge,
    ) -> Result<FinalRecommendation> {
        let (action, final_dose_mg) = match &safety.action {
            SafetyAction::Proceed => (RecommendationAction::Prescribe, calculation.proposed_dose_mg),
            SafetyAction::ProceedWithMonitoring { .. } => (RecommendationAction::PrescribeWithMonitoring, calculation.proposed_dose_mg),
            SafetyAction::AdjustDose { new_dose_mg, .. } => (RecommendationAction::Prescribe, *new_dose_mg),
            SafetyAction::Hold { .. } => (RecommendationAction::RequireApproval, 0.0),
            SafetyAction::Contraindicated { .. } => (RecommendationAction::Contraindicated, 0.0),
            SafetyAction::RequireSpecialist { .. } => (RecommendationAction::ReferToSpecialist, 0.0),
        };

        Ok(FinalRecommendation {
            action,
            final_dose_mg,
            route: drug_knowledge.default_route.clone(),
            frequency: drug_knowledge.default_frequency.clone(),
            duration: drug_knowledge.default_duration.clone(),
            special_instructions: drug_knowledge.special_instructions.clone(),
            monitoring_required: safety.monitoring_parameters.clone(),
            contraindications: safety.findings.iter()
                .filter(|f| f.severity == "contraindicated")
                .map(|f| f.message.clone())
                .collect(),
            alternatives: drug_knowledge.alternatives.clone(),
        })
    }

    /// Create comprehensive audit trail
    fn create_audit_trail(
        &self,
        request: &ClinicalRequest,
        drug_knowledge: &DrugKnowledge,
        strategy: &CalculationStrategy,
    ) -> Result<AuditTrail> {
        let models_invoked = match strategy {
            CalculationStrategy::StandardRules => vec![],
            CalculationStrategy::CustomModel(model_name) => vec![model_name.clone()],
        };

        Ok(AuditTrail {
            calculation_kb_version: drug_knowledge.calculation_version.clone(),
            safety_kb_version: drug_knowledge.safety_version.clone(),
            engine_version: "unified-clinical-engine-1.0.0".to_string(),
            processing_steps: vec![
                "load_drug_knowledge".to_string(),
                "determine_strategy".to_string(),
                "calculate_dose".to_string(),
                "verify_safety".to_string(),
                "generate_recommendation".to_string(),
            ],
            rule_files_loaded: vec![
                format!("calculation/{}.toml", request.drug_id),
                format!("safety/{}.toml", request.drug_id),
            ],
            models_invoked,
        })
    }

    /// Generate titration schedule for chronic medication management
    pub async fn generate_titration_schedule(
        &self,
        request: TitrationRequest
    ) -> Result<TitrationSchedule> {
        Ok(self.titration_engine.generate_titration_schedule(request)?)
    }

    /// Evaluate a mathematical expression with patient context
    pub async fn evaluate_expression(
        &self,
        expression: &str,
        request: &ClinicalRequest
    ) -> Result<EvaluationResult> {
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

        Ok(result)
    }

    /// Parse and validate a mathematical expression without evaluation
    pub fn parse_expression(&self, expression: &str) -> Result<MathExpression> {
        ExpressionParser::parse(expression)
    }

    /// Get available variables for a patient context
    pub fn get_available_variables(&self, request: &ClinicalRequest) -> Result<SubstitutionResult> {
        self.variable_substitution.create_substitution(request)
    }

    // Commented out functions that reference missing modules
    /*
    /// Assess cumulative risk for polypharmacy
    pub async fn assess_cumulative_risk(
        &self,
        patient: &titration_engine::PatientData,
        medications: &[cumulative_risk::Medication],
        clinical_context: &titration_engine::ClinicalContext
    ) -> Result<CumulativeRiskProfile> {
        self.cumulative_risk_engine.assess_cumulative_risk(patient, medications, clinical_context).await
    }

    /// Generate risk-adjusted titration schedule
    pub async fn generate_risk_adjusted_titration(
        &self,
        request: TitrationRequest,
        all_medications: Vec<cumulative_risk::Medication>
    ) -> Result<RiskAdjustedTitrationSchedule> {
        self.risk_aware_titration.generate_risk_adjusted_schedule(request, all_medications).await
    }
    */
}

/// Comprehensive dose response with titration and risk assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComprehensiveDoseResponse {
    pub primary_dose: ClinicalResponse,
    pub titration_schedule: Option<TitrationSchedule>,
    pub cumulative_risk: Option<CumulativeRiskProfile>,
    pub integrated_recommendations: Vec<String>,
}

/// Drug knowledge loaded from TOML files
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugKnowledge {
    pub meta: DrugMeta,
    pub calculation_rules: serde_json::Value, // Flexible structure for rules
    pub safety_rules: serde_json::Value,      // Flexible structure for rules
    pub default_route: String,
    pub default_frequency: String,
    pub default_duration: Option<String>,
    pub special_instructions: Vec<String>,
    pub alternatives: Vec<AlternativeDrug>,
    pub calculation_version: String,
    pub safety_version: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugMeta {
    pub drug_id: String,
    pub name: String,
    pub generic_name: String,
    pub calculation_strategy: String,
    pub therapeutic_class: Vec<String>,
    pub mechanism_of_action: Vec<String>,
}
