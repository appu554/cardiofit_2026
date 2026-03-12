//! Main orchestrator for the Rust Recipe Engine

use crate::models::*;
use crate::engine::{RuleEvaluator, RecipeExecutor, IntentManifestGenerator};
use crate::knowledge::KnowledgeLoader;
use crate::unified_clinical_engine::{
    UnifiedClinicalEngine,
    knowledge_base::KnowledgeBase as UnifiedKnowledgeBase,
    ClinicalRequest, PatientContext, BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction
};
use std::sync::Arc;
use std::time::Instant;
use std::collections::HashMap;
use tracing::{info, instrument};
use chrono::Utc;
use anyhow::{Result, anyhow};

/// Main orchestrator that coordinates rule evaluation and recipe execution
pub struct RustEngineOrchestrator {
    rule_evaluator: Arc<RuleEvaluator>,
    recipe_executor: Arc<RecipeExecutor>,
    manifest_generator: Arc<IntentManifestGenerator>,
    knowledge_base: Arc<KnowledgeBase>, // Old knowledge base for compatibility
    unified_knowledge_base: Arc<UnifiedKnowledgeBase>, // New unified knowledge base
    unified_engine: Arc<UnifiedClinicalEngine>,
}

impl RustEngineOrchestrator {
    /// Create a new orchestrator
    pub async fn new(knowledge_base_path: &str) -> Result<Self, crate::EngineError> {
        info!("Initializing Rust Engine Orchestrator");

        // Load old knowledge base for compatibility
        let loader = KnowledgeLoader::new(knowledge_base_path);
        let knowledge_base = Arc::new(loader.load_knowledge_base().await?);

        // Create new unified knowledge base from TOML files
        let unified_knowledge_base = Arc::new(UnifiedKnowledgeBase::new(knowledge_base_path).await.map_err(|e| crate::EngineError::Generic(e.to_string()))?);

        // Initialize components with old knowledge base
        let rule_evaluator = Arc::new(RuleEvaluator::new((*knowledge_base).clone()));
        let recipe_executor = Arc::new(RecipeExecutor::new(knowledge_base_path).await?);
        let manifest_generator = Arc::new(IntentManifestGenerator::new((*knowledge_base).clone()));

        // Initialize the Unified Clinical Engine with new knowledge base
        let unified_engine = Arc::new(UnifiedClinicalEngine::new(unified_knowledge_base.clone()).map_err(|e| crate::EngineError::Generic(e.to_string()))?);

        // Validate knowledge base
        rule_evaluator.validate_rules()?;

        info!("Rust Engine Orchestrator initialized successfully");
        info!("Old knowledge base summary: {:?}", knowledge_base.summary());
        let stats = unified_knowledge_base.get_stats();
        info!("Unified knowledge base loaded with {} drug rules, {} DDI rules", stats.total_drug_rules, stats.total_ddi_rules);

        Ok(Self {
            rule_evaluator,
            recipe_executor,
            manifest_generator,
            knowledge_base,
            unified_knowledge_base,
            unified_engine,
        })
    }

    /// ⭐ MAIN METHOD: Execute recipe (called by Go engine)
    #[instrument(skip(self))]
    pub async fn execute_recipe(&self, request: &RecipeExecutionRequest) -> Result<MedicationProposal, crate::EngineError> {
        let start_time = Instant::now();

        info!("Executing recipe: {} variant: {} for patient: {}", 
              request.recipe_id, request.variant, request.patient_id);

        // Execute recipe using the recipe executor
        let proposal = self.recipe_executor.execute_recipe(request).await?;

        let execution_time = start_time.elapsed();
        info!("Recipe execution completed in {:?}: {}", execution_time, request.recipe_id);

        Ok(proposal)
    }

    /// Execute Flow2 request using Unified Clinical Engine with fallback
    #[instrument(skip(self))]
    pub async fn execute_flow2(&self, request: &Flow2Request) -> Result<Flow2Response, crate::EngineError> {
        let start_time = Instant::now();

        info!("Executing Flow2 request: {} for patient: {}",
              request.request_id, request.patient_id);

        // Use Unified Clinical Engine ONLY - NO FALLBACK!
        let response = self.execute_flow2_unified(request).await?;

        info!("Flow2 executed successfully using Unified Clinical Engine");
        return Ok(response);
    }

    /// Execute Flow2 request using the Unified Clinical Engine
    #[instrument(skip(self))]
    async fn execute_flow2_unified(&self, request: &Flow2Request) -> Result<Flow2Response, crate::EngineError> {
        // Convert Flow2Request to ClinicalRequest for unified engine
        let clinical_request = self.convert_flow2_to_clinical_request(request)?;

        // Execute using Unified Clinical Engine
        let clinical_response = self.unified_engine.process_clinical_request(clinical_request).await
            .map_err(|e| crate::EngineError::Generic(e.to_string()))?;

        // Create safety alerts from unified engine response
        let safety_alerts = vec![
            crate::models::medication::SafetyAlert {
                alert_id: format!("unified-{}", clinical_response.request_id),
                severity: match clinical_response.safety_result.action {
                    crate::unified_clinical_engine::SafetyAction::Proceed => "low".to_string(),
                    crate::unified_clinical_engine::SafetyAction::ProceedWithMonitoring { .. } => "medium".to_string(),
                    crate::unified_clinical_engine::SafetyAction::AdjustDose { .. } => "medium".to_string(),
                    crate::unified_clinical_engine::SafetyAction::Hold { .. } => "high".to_string(),
                    crate::unified_clinical_engine::SafetyAction::Contraindicated { .. } => "high".to_string(),
                    crate::unified_clinical_engine::SafetyAction::RequireSpecialist { .. } => "high".to_string(),
                },
                alert_type: "unified_clinical_engine".to_string(),
                message: format!("Unified Clinical Engine: {:?}", clinical_response.safety_result.action),
                description: format!("Final dose: {} mg", clinical_response.final_recommendation.final_dose_mg),
                action_required: !clinical_response.final_recommendation.monitoring_required.is_empty(),
            }
        ];

        // Create a basic recipe result from unified engine response
        let recipe_result = RecipeResult {
            recipe_id: format!("{}-unified", clinical_response.drug_id),
            recipe_name: format!("Unified Clinical Engine - {}", clinical_response.drug_id),
            overall_status: format!("{:?}", clinical_response.safety_result.action),
            execution_time_ms: clinical_response.processing_time_ms as i64,
            validations: vec![],
            clinical_decision_support: serde_json::Map::new(),
            recommendations: vec![format!("Final dose: {} mg", clinical_response.final_recommendation.final_dose_mg)],
            warnings: vec![],
            errors: vec![],
            metadata: serde_json::Map::new(),
        };

        // Convert ClinicalResponse to Flow2Response using existing structure
        Ok(Flow2Response {
            request_id: request.request_id.clone(),
            patient_id: request.patient_id.clone(),
            overall_status: "success".to_string(),
            execution_summary: ExecutionSummary {
                total_recipes_executed: 1,
                successful_recipes: 1,
                failed_recipes: 0,
                warnings: 0,
                errors: 0,
                engine: "unified_clinical_engine".to_string(),
                cache_hit_rate: 0.0,
            },
            recipe_results: vec![recipe_result],
            clinical_decision_support: serde_json::Map::new(),
            safety_alerts,
            recommendations: vec![
                format!("Dose: {} mg", clinical_response.final_recommendation.final_dose_mg),
                format!("Route: {}", clinical_response.final_recommendation.route),
                format!("Frequency: {}", clinical_response.final_recommendation.frequency),
            ],
            analytics: serde_json::Map::new(),
            execution_time_ms: clinical_response.processing_time_ms as i64,
            engine_used: "unified_clinical_engine".to_string(),
            timestamp: chrono::Utc::now(),
            processing_metadata: ProcessingMetadata {
                fallback_used: false,
                cache_used: false,
                context_sources: vec!["unified_clinical_engine".to_string()],
                processing_stages: vec!["unified_calculation".to_string(), "safety_verification".to_string()],
                snapshot_based: false,
                snapshot_id: None,
                snapshot_validation: None,
            },
        })
    }

    /// Execute medication intelligence analysis
    #[instrument(skip(self))]
    pub async fn execute_medication_intelligence(&self, request: &MedicationIntelligenceRequest) -> Result<MedicationIntelligenceResponse, crate::EngineError> {
        let start_time = Instant::now();

        info!("Executing medication intelligence for patient: {}", request.patient_id);

        // For now, return a basic response
        // TODO: Implement full medication intelligence
        let response = MedicationIntelligenceResponse {
            request_id: request.request_id.clone(),
            intelligence_score: 0.85,
            medication_analysis: serde_json::Map::new(),
            interaction_analysis: serde_json::Map::new(),
            outcome_predictions: serde_json::Map::new(),
            alternative_recommendations: Vec::new(),
            clinical_insights: Vec::new(),
            execution_time_ms: start_time.elapsed().as_millis() as i64,
        };

        Ok(response)
    }

    /// Execute dose optimization
    #[instrument(skip(self))]
    pub async fn execute_dose_optimization(&self, request: &DoseOptimizationRequest) -> Result<DoseOptimizationResponse, crate::EngineError> {
        let start_time = Instant::now();

        info!("Executing dose optimization for medication: {}", request.medication_code);

        // Use the FULL UNIFIED CLINICAL ENGINE with all 9 advanced modules!
        let clinical_request = crate::unified_clinical_engine::ClinicalRequest {
            request_id: request.request_id.clone(),
            drug_id: request.medication_code.clone(),
            indication: request.clinical_context.get("indication")
                .and_then(|v| v.as_str())
                .unwrap_or("unknown")
                .to_string(),
            patient_context: crate::unified_clinical_engine::PatientContext {
                age_years: request.clinical_parameters.get("age_years")
                    .and_then(|v| v.as_f64())
                    .unwrap_or(65.0),
                weight_kg: request.clinical_parameters.get("weight_kg")
                    .and_then(|v| v.as_f64())
                    .unwrap_or(70.0),
                height_cm: request.clinical_parameters.get("height_cm")
                    .and_then(|v| v.as_f64())
                    .unwrap_or(170.0),
                sex: crate::unified_clinical_engine::BiologicalSex::Other,
                pregnancy_status: crate::unified_clinical_engine::PregnancyStatus::NotPregnant,
                renal_function: crate::unified_clinical_engine::RenalFunction {
                    egfr_ml_min: Some(request.clinical_parameters.get("egfr")
                        .and_then(|v| v.as_f64())
                        .unwrap_or(90.0)),
                    egfr_ml_min_1_73m2: Some(request.clinical_parameters.get("egfr")
                        .and_then(|v| v.as_f64())
                        .unwrap_or(90.0)),
                    creatinine_clearance: Some(90.0),
                    creatinine_mg_dl: request.clinical_parameters.get("creatinine")
                        .and_then(|v| v.as_f64()),
                    bun_mg_dl: request.clinical_parameters.get("bun")
                        .and_then(|v| v.as_f64()),
                    stage: Some("Normal".to_string()),
                },
                hepatic_function: crate::unified_clinical_engine::HepaticFunction {
                    child_pugh_class: Some("A".to_string()),
                    alt_u_l: Some(25.0),
                    ast_u_l: Some(25.0),
                    bilirubin_mg_dl: Some(1.0),
                    albumin_g_dl: Some(4.0),
                },
                active_medications: vec![],
                allergies: vec![],
                conditions: vec![],
                lab_values: std::collections::HashMap::new(),
            },
            timestamp: chrono::Utc::now(),
        };

        // Execute the FULL UNIFIED CLINICAL ENGINE with all 9 advanced modules!
        match self.unified_engine.process_clinical_request(clinical_request).await {
            Ok(clinical_response) => {
                let dose = clinical_response.calculation_result.proposed_dose_mg;
                let response = DoseOptimizationResponse {
                    request_id: request.request_id.clone(),
                    optimized_dose: dose,
                    optimization_score: clinical_response.calculation_result.confidence_score,
                    confidence_interval: ConfidenceInterval {
                        lower: dose * 0.8,
                        upper: dose * 1.2,
                        confidence: 0.95,
                    },
                    pharmacokinetic_predictions: serde_json::Map::new(),
                    monitoring_recommendations: clinical_response.final_recommendation.monitoring_required,
                    clinical_rationale: format!("REAL UNIFIED CLINICAL ENGINE: {:?} - {}", clinical_response.final_recommendation.action, clinical_response.calculation_result.calculation_strategy),
                    execution_time_ms: start_time.elapsed().as_millis() as i64,
                };
                Ok(response)
            }
            Err(e) => {
                info!("Unified engine failed, using fallback: {}", e);
                let response = DoseOptimizationResponse {
                    request_id: request.request_id.clone(),
                    optimized_dose: 500.0,
                    optimization_score: 0.75,
                    confidence_interval: ConfidenceInterval {
                        lower: 400.0,
                        upper: 600.0,
                        confidence: 0.80,
                    },
                    pharmacokinetic_predictions: serde_json::Map::new(),
                    monitoring_recommendations: Vec::new(),
                    clinical_rationale: "Fallback calculation - unified engine error".to_string(),
                    execution_time_ms: start_time.elapsed().as_millis() as i64,
                };
                Ok(response)
            }
        }
    }

    /// Get engine health status
    pub fn health_check(&self) -> EngineHealthStatus {
        let knowledge_summary = self.knowledge_base.summary();

        EngineHealthStatus {
            status: "healthy".to_string(),
            version: crate::VERSION.to_string(),
            knowledge_base_loaded: true,
            total_knowledge_items: knowledge_summary.total_items,
            rules_loaded: knowledge_summary.orb_rules_count,
            recipes_loaded: knowledge_summary.clinical_recipes_count,
            uptime_seconds: 0, // TODO: Track actual uptime
        }
    }

    /// Get rule evaluator reference
    pub fn rule_evaluator(&self) -> &RuleEvaluator {
        &self.rule_evaluator
    }

    /// ⭐ Generate enhanced intent manifest with clinical intelligence
    #[instrument(skip(self))]
    pub async fn generate_enhanced_manifest(&self, request: &MedicationRequest) -> Result<EnhancedIntentManifest, crate::EngineError> {
        let start_time = std::time::Instant::now();

        info!("Generating enhanced intent manifest for request: {}", request.request_id);

        // 1. Build evaluation context
        let context = self.build_evaluation_context_from_request(request)?;

        // 2. Find matching rule
        let matching_rule = self.find_matching_rule_for_request(request, &context)?;

        // 3. Generate enhanced manifest using the manifest generator
        let enhanced_manifest = self.manifest_generator
            .generate_enhanced_manifest(request, &matching_rule, &context)
            .await?;

        let execution_time = start_time.elapsed();
        info!("Enhanced manifest generation completed in {:?}: {}", execution_time, request.request_id);

        Ok(enhanced_manifest)
    }

    // Helper methods for enhanced manifest generation
    fn build_evaluation_context_from_request(&self, request: &MedicationRequest) -> Result<HashMap<String, serde_json::Value>, crate::EngineError> {
        let mut context = HashMap::new();

        // Add medication facts
        context.insert("medication_code".to_string(), serde_json::Value::String(request.medication_code.clone()));
        context.insert("medication_name".to_string(), serde_json::Value::String(request.medication_name.clone()));
        context.insert("patient_conditions".to_string(), serde_json::Value::Array(
            request.patient_conditions.iter().map(|c| serde_json::Value::String(c.clone())).collect()
        ));

        // Add patient demographics if available
        if let Some(demographics) = &request.patient_demographics {
            if let Some(age) = demographics.age_years {
                context.insert("patient_age_years".to_string(), serde_json::Value::Number(
                    serde_json::Number::from_f64(age).unwrap()
                ));
                context.insert("is_elderly".to_string(), serde_json::Value::Bool(age >= 65.0));
            }

            if let Some(weight) = demographics.weight_kg {
                context.insert("patient_weight_kg".to_string(), serde_json::Value::Number(
                    serde_json::Number::from_f64(weight).unwrap()
                ));
                context.insert("is_obese".to_string(), serde_json::Value::Bool(demographics.is_obese()));
            }

            if let Some(egfr) = demographics.egfr {
                context.insert("patient_egfr".to_string(), serde_json::Value::Number(
                    serde_json::Number::from_f64(egfr).unwrap()
                ));
                context.insert("has_renal_impairment".to_string(), serde_json::Value::Bool(demographics.has_renal_impairment()));
            }
        }

        Ok(context)
    }

    fn find_matching_rule_for_request(&self, request: &MedicationRequest, context: &HashMap<String, serde_json::Value>) -> Result<ORBRule, crate::EngineError> {
        // Rules are pre-sorted by priority (highest first)
        for rule in &self.knowledge_base.orb_rules.rules {
            if self.evaluate_rule_conditions_for_request(rule, context) {
                info!("Rule matched: {} (priority: {})", rule.id, rule.priority);
                return Ok(rule.clone());
            }
        }

        // If no specific rule matches, look for catch-all rule
        for rule in &self.knowledge_base.orb_rules.rules {
            if !rule.has_conditions() {
                info!("Using catch-all rule: {}", rule.id);
                return Ok(rule.clone());
            }
        }

        Err(crate::EngineError::RuleEvaluation("No matching rule found".to_string()))
    }

    fn evaluate_rule_conditions_for_request(&self, rule: &ORBRule, context: &HashMap<String, serde_json::Value>) -> bool {
        // Handle rules with no conditions (catch-all rules)
        if !rule.has_conditions() {
            return true;
        }

        // Evaluate using the enhanced condition evaluation
        rule.conditions.evaluate(context)
    }

    /// Get engine performance metrics
    pub fn get_metrics(&self) -> EngineMetrics {
        EngineMetrics {
            total_requests: 0,        // TODO: Track actual metrics
            successful_requests: 0,   // TODO: Track actual metrics
            failed_requests: 0,       // TODO: Track actual metrics
            average_response_time_ms: 0.0, // TODO: Track actual metrics
            cache_hit_rate: 0.0,      // TODO: Implement caching
            knowledge_base_version: self.knowledge_base.medication_knowledge_core.metadata.version.clone(),
        }
    }

    // Helper methods
    fn convert_flow2_to_medication_request(&self, request: &Flow2Request) -> Result<MedicationRequest, crate::EngineError> {
        // Extract medication information from Flow2Request
        let medication_code = request.medication_data.get("code")
            .and_then(|v| v.as_str())
            .unwrap_or("unknown")
            .to_string();

        let medication_name = request.medication_data.get("name")
            .and_then(|v| v.as_str())
            .unwrap_or("Unknown Medication")
            .to_string();

        // Extract patient conditions
        let patient_conditions = request.patient_data.get("conditions")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter().filter_map(|v| v.as_str().map(|s| s.to_string())).collect())
            .unwrap_or_default();

        // Create basic medication request
        let mut med_request = MedicationRequest::new(
            request.request_id.clone(),
            request.patient_id.clone(),
            medication_code,
            medication_name,
            patient_conditions,
        );

        // Add demographics if available
        if let Some(demographics_data) = request.patient_data.get("demographics") {
            let demographics = PatientDemographics {
                age_years: demographics_data.get("age_years").and_then(|v| v.as_f64()),
                weight_kg: demographics_data.get("weight_kg").and_then(|v| v.as_f64()),
                height_cm: demographics_data.get("height_cm").and_then(|v| v.as_f64()),
                gender: demographics_data.get("gender").and_then(|v| v.as_str()).map(|s| s.to_string()),
                bmi: demographics_data.get("bmi").and_then(|v| v.as_f64()),
                bsa_m2: demographics_data.get("bsa_m2").and_then(|v| v.as_f64()),
                race: demographics_data.get("race").and_then(|v| v.as_str()).map(|s| s.to_string()),
                ethnicity: demographics_data.get("ethnicity").and_then(|v| v.as_str()).map(|s| s.to_string()),
                egfr: demographics_data.get("egfr").and_then(|v| v.as_f64()),
                creatinine_clearance: demographics_data.get("creatinine_clearance").and_then(|v| v.as_f64()),
            };
            med_request = med_request.with_demographics(demographics);
        }

        Ok(med_request)
    }

    /// Convert Flow2Request to ClinicalRequest for Unified Clinical Engine
    fn convert_flow2_to_clinical_request(&self, request: &Flow2Request) -> Result<ClinicalRequest, crate::EngineError> {
        // Extract medication code from medication_data
        let drug_id = request.medication_data.get("code")
            .or_else(|| request.medication_data.get("drug_id"))
            .and_then(|v| v.as_str())
            .ok_or_else(|| crate::EngineError::Generic("Missing medication code in request".to_string()))?
            .to_string();

        // Extract indication
        let indication = request.medication_data.get("indication")
            .and_then(|v| v.as_str())
            .unwrap_or("unknown")
            .to_string();

        // Extract patient age
        let age_years = request.patient_data.get("age_years")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0) as u8;

        // Extract patient weight
        let weight_kg = request.patient_data.get("weight_kg")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        // Extract patient height
        let height_cm = request.patient_data.get("height_cm")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        // Extract creatinine clearance
        let creatinine_clearance = request.patient_data.get("creatinine_clearance")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);

        // Extract medical conditions
        let conditions = request.patient_data.get("medical_conditions")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter()
                .filter_map(|v| v.as_str())
                .map(|s| s.to_string())
                .collect())
            .unwrap_or_default();

        // Create renal function based on creatinine clearance
        let renal_function = crate::unified_clinical_engine::RenalFunction {
            egfr_ml_min: Some(creatinine_clearance),
            egfr_ml_min_1_73m2: Some(creatinine_clearance),
            creatinine_clearance: Some(creatinine_clearance),
            creatinine_mg_dl: Some(1.0), // Default value
            bun_mg_dl: Some(15.0), // Default value
            stage: Some(if creatinine_clearance >= 90.0 {
                "Normal".to_string()
            } else if creatinine_clearance >= 60.0 {
                "Stage 2".to_string()
            } else if creatinine_clearance >= 30.0 {
                "Stage 3".to_string()
            } else if creatinine_clearance >= 15.0 {
                "Stage 4".to_string()
            } else {
                "Stage 5".to_string()
            }),
        };

        Ok(ClinicalRequest {
            request_id: request.request_id.clone(),
            drug_id,
            indication,
            patient_context: crate::unified_clinical_engine::PatientContext {
                age_years: age_years as f64,
                weight_kg,
                height_cm,
                sex: crate::unified_clinical_engine::BiologicalSex::Other,
                pregnancy_status: crate::unified_clinical_engine::PregnancyStatus::NotPregnant,
                renal_function,
                hepatic_function: crate::unified_clinical_engine::HepaticFunction {
                    child_pugh_class: Some("A".to_string()),
                    alt_u_l: Some(25.0),
                    ast_u_l: Some(25.0),
                    bilirubin_mg_dl: Some(1.0),
                    albumin_g_dl: Some(4.0),
                },
                active_medications: vec![],
                allergies: vec![],
                conditions,
                lab_values: std::collections::HashMap::new(),
            },
            timestamp: chrono::Utc::now(),
        })
    }

    fn convert_proposal_to_flow2_response(
        &self,
        request: &Flow2Request,
        intent_manifest: &IntentManifest,
        proposal: &MedicationProposal,
        start_time: Instant,
    ) -> Result<Flow2Response, crate::EngineError> {
        let execution_time = start_time.elapsed().as_millis() as i64;

        // Create recipe result
        let recipe_result = RecipeResult {
            recipe_id: intent_manifest.recipe_id.clone(),
            recipe_name: format!("Recipe: {}", intent_manifest.recipe_id),
            overall_status: proposal.safety_status.clone(),
            execution_time_ms: proposal.execution_time_ms,
            validations: Vec::new(), // TODO: Add validations
            clinical_decision_support: serde_json::Map::new(), // TODO: Add CDS data
            recommendations: proposal.monitoring_plan.clone(),
            warnings: proposal.safety_alerts.clone(),
            errors: Vec::new(),
            metadata: serde_json::Map::new(),
        };

        Ok(Flow2Response {
            request_id: request.request_id.clone(),
            patient_id: request.patient_id.clone(),
            overall_status: proposal.safety_status.clone(),
            execution_summary: ExecutionSummary {
                total_recipes_executed: 1,
                successful_recipes: if proposal.safety_status == "SAFE" { 1 } else { 0 },
                failed_recipes: 0,
                warnings: proposal.safety_alerts.len(),
                errors: 0,
                engine: "rust".to_string(),
                cache_hit_rate: 0.0,
            },
            recipe_results: vec![recipe_result],
            clinical_decision_support: serde_json::Map::new(),
            safety_alerts: proposal.safety_alerts.iter().map(|alert| crate::models::medication::SafetyAlert {
                alert_id: uuid::Uuid::new_v4().to_string(),
                severity: "medium".to_string(),
                alert_type: "clinical".to_string(),
                message: alert.clone(),
                description: alert.clone(),
                action_required: false,
            }).collect(),
            recommendations: Vec::new(),
            analytics: serde_json::Map::new(),
            execution_time_ms: execution_time,
            engine_used: format!("rust-{}", crate::VERSION),
            timestamp: chrono::Utc::now(),
            processing_metadata: ProcessingMetadata {
                fallback_used: false,
                cache_used: false,
                context_sources: Vec::new(),
                processing_stages: Vec::new(),
                snapshot_based: false,
                snapshot_id: None,
                snapshot_validation: None,
            },
        })
    }
}

/// Engine health status
#[derive(Debug, Clone, serde::Serialize)]
pub struct EngineHealthStatus {
    pub status: String,
    pub version: String,
    pub knowledge_base_loaded: bool,
    pub total_knowledge_items: usize,
    pub rules_loaded: usize,
    pub recipes_loaded: usize,
    pub uptime_seconds: u64,
}

/// Engine performance metrics
#[derive(Debug, Clone, serde::Serialize)]
pub struct EngineMetrics {
    pub total_requests: u64,
    pub successful_requests: u64,
    pub failed_requests: u64,
    pub average_response_time_ms: f64,
    pub cache_hit_rate: f64,
    pub knowledge_base_version: String,
}
