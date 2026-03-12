//! ⭐ MISSING IMPLEMENTATION: Recipe Execution Engine
//! This is what we need to implement to handle the recipe-driven data flow from Go

use crate::models::*;
use crate::knowledge::KnowledgeLoader;
use std::collections::HashMap;
use serde_json;
use tracing::{info, warn, error};

/// Recipe Execution Engine - The core missing piece
pub struct RecipeExecutor {
    knowledge_base: KnowledgeBase,
}

impl RecipeExecutor {
    /// Create a new recipe executor
    pub async fn new(knowledge_base_path: &str) -> Result<Self, crate::EngineError> {
        let loader = KnowledgeLoader::new(knowledge_base_path);
        let knowledge_base = loader.load_knowledge_base().await?;
        
        Ok(Self {
            knowledge_base,
        })
    }

    /// ⭐ MAIN METHOD: Execute specific recipe (what Go engine calls)
    pub async fn execute_recipe(&self, request: &RecipeExecutionRequest) -> Result<MedicationProposal, crate::EngineError> {
        info!("Executing recipe: {} variant: {} for patient: {}", 
              request.recipe_id, request.variant, request.patient_id);

        // 1. Parse clinical context JSON from Go
        let clinical_data: serde_json::Value = serde_json::from_str(&request.clinical_context)
            .map_err(|e| crate::EngineError::Serialization(e))?;

        // 2. Load the specific recipe by ID (from Go's Intent Manifest)
        let clinical_recipe = self.knowledge_base
            .clinical_recipes
            .recipes
            .get(&request.recipe_id)
            .ok_or_else(|| crate::EngineError::RuleEvaluation(
                format!("Recipe not found: {}", request.recipe_id)
            ))?;

        // 3. Get the specific calculation variant (from Go's Intent Manifest)
        let calculation_variant = clinical_recipe
            .calculation_variants
            .get(&request.variant)
            .ok_or_else(|| crate::EngineError::RuleEvaluation(
                format!("Variant not found: {} for recipe: {}", request.variant, request.recipe_id)
            ))?;

        // 4. Extract clinical data for calculations
        let clinical_context = self.parse_clinical_context(&clinical_data)?;

        // 5. Execute the specific recipe calculations
        let calculation_results = self.execute_calculation_steps(
            &calculation_variant.logic_steps,
            &clinical_context
        )?;

        // 6. Perform safety checks
        let safety_assessment = self.perform_safety_checks(
            &clinical_recipe.safety_checks,
            &clinical_context,
            &calculation_results
        )?;

        // 7. Generate medication proposal
        let proposal = self.generate_medication_proposal(
            request,
            clinical_recipe,
            &calculation_results,
            &safety_assessment
        )?;

        info!("Recipe execution completed: {} in {}ms", 
              request.recipe_id, proposal.execution_time_ms);

        Ok(proposal)
    }

    /// Parse clinical context from Go engine
    fn parse_clinical_context(&self, clinical_data: &serde_json::Value) -> Result<ClinicalContextData, crate::EngineError> {
        let fields = clinical_data.get("fields")
            .ok_or_else(|| crate::EngineError::RuleEvaluation("Missing fields in clinical context".to_string()))?;

        Ok(ClinicalContextData {
            age: fields.get("demographics.age").and_then(|v| v.as_f64()),
            weight_kg: fields.get("demographics.weight.actual_kg").and_then(|v| v.as_f64()),
            height_cm: fields.get("demographics.height_cm").and_then(|v| v.as_f64()),
            gender: fields.get("demographics.gender").and_then(|v| v.as_str()).map(|s| s.to_string()),
            egfr: fields.get("labs.egfr[latest]").and_then(|v| v.as_f64()),
            creatinine: fields.get("labs.serum_creatinine[latest]").and_then(|v| v.as_f64()),
            conditions: self.extract_conditions(fields),
            allergies: self.extract_allergies(fields),
            current_medications: self.extract_medications(fields),
        })
    }

    /// Execute calculation steps from clinical recipe
    fn execute_calculation_steps(
        &self,
        logic_steps: &[LogicStep],
        clinical_context: &ClinicalContextData
    ) -> Result<HashMap<String, f64>, crate::EngineError> {
        let mut results = HashMap::new();

        for step in logic_steps {
            let result = match step.name.as_str() {
                "calculate_loading_dose" => {
                    let weight = clinical_context.weight_kg.unwrap_or(70.0);
                    let loading_dose = weight * 25.0; // 25 mg/kg for Vancomycin
                    
                    // Apply constraints
                    let final_dose = if let Some(max_value) = step.max_value {
                        loading_dose.min(max_value)
                    } else {
                        loading_dose
                    };
                    
                    info!("Calculated loading dose: {} mg (weight: {} kg)", final_dose, weight);
                    final_dose
                }
                "calculate_maintenance_dose" => {
                    let loading_dose = results.get("loading_dose").copied().unwrap_or(0.0);
                    let egfr = clinical_context.egfr.unwrap_or(90.0);
                    
                    // Renal adjustment
                    let adjustment_factor = if egfr < 30.0 { 0.5 } else if egfr < 60.0 { 0.75 } else { 1.0 };
                    let maintenance_dose = (loading_dose * 0.6) * adjustment_factor;
                    
                    info!("Calculated maintenance dose: {} mg (eGFR: {}, adjustment: {})", 
                          maintenance_dose, egfr, adjustment_factor);
                    maintenance_dose
                }
                _ => {
                    warn!("Unknown calculation step: {}", step.name);
                    0.0
                }
            };

            results.insert(step.output.clone(), result);
        }

        Ok(results)
    }

    /// Perform safety checks from clinical recipe
    fn perform_safety_checks(
        &self,
        safety_checks: &[SafetyCheck],
        clinical_context: &ClinicalContextData,
        calculation_results: &HashMap<String, f64>
    ) -> Result<SafetyAssessment, crate::EngineError> {
        let mut safety_alerts = Vec::new();
        let mut contraindications = Vec::new();
        let mut overall_status = "SAFE".to_string();

        for check in safety_checks {
            // Evaluate safety conditions
            let context_map = self.build_context_map(clinical_context, calculation_results);
            
            let conditions_met = check.conditions.iter().all(|condition| {
                if let Some(actual_value) = context_map.get(&condition.fact) {
                    condition.evaluate(actual_value)
                } else {
                    false
                }
            });

            if conditions_met {
                match check.action.action_type.as_str() {
                    "warning" => {
                        safety_alerts.push(check.action.message.clone());
                        if overall_status == "SAFE" {
                            overall_status = "WARNING".to_string();
                        }
                    }
                    "contraindication" => {
                        contraindications.push(check.action.message.clone());
                        overall_status = "UNSAFE".to_string();
                    }
                    _ => {}
                }
            }
        }

        Ok(SafetyAssessment {
            overall_status,
            safety_alerts,
            contraindications,
        })
    }

    /// Generate final medication proposal
    fn generate_medication_proposal(
        &self,
        request: &RecipeExecutionRequest,
        recipe: &ClinicalRecipe,
        calculation_results: &HashMap<String, f64>,
        safety_assessment: &SafetyAssessment
    ) -> Result<MedicationProposal, crate::EngineError> {
        let calculated_dose = calculation_results.get("loading_dose").copied().unwrap_or(0.0);

        Ok(MedicationProposal {
            medication_code: request.medication_code.clone(),
            medication_name: recipe.name.clone(),
            calculated_dose,
            dose_unit: "mg".to_string(),
            frequency: "q12h".to_string(),
            duration: Some("7 days".to_string()),
            safety_status: safety_assessment.overall_status.clone(),
            safety_alerts: safety_assessment.safety_alerts.clone(),
            contraindications: safety_assessment.contraindications.clone(),
            clinical_rationale: format!(
                "Calculated using recipe {} variant {} - {}",
                request.recipe_id, request.variant, recipe.description
            ),
            monitoring_plan: vec![
                "Monitor serum creatinine daily".to_string(),
                "Target trough level 15-20 mg/L".to_string(),
                "Monitor for nephrotoxicity".to_string()
            ],
            alternatives: Vec::new(), // TODO: Implement alternatives
            execution_time_ms: 5, // TODO: Measure actual execution time
            recipe_version: "v1.0".to_string(),
        })
    }

    // Helper methods
    fn extract_conditions(&self, fields: &serde_json::Value) -> Vec<String> {
        fields.get("conditions.active")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter().filter_map(|v| v.as_str().map(|s| s.to_string())).collect())
            .unwrap_or_default()
    }

    fn extract_allergies(&self, fields: &serde_json::Value) -> Vec<String> {
        fields.get("allergies.active")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter().filter_map(|v| {
                v.get("allergen").and_then(|a| a.as_str().map(|s| s.to_string()))
            }).collect())
            .unwrap_or_default()
    }

    fn extract_medications(&self, fields: &serde_json::Value) -> Vec<String> {
        fields.get("medications.current")
            .and_then(|v| v.as_array())
            .map(|arr| arr.iter().filter_map(|v| {
                v.get("name").and_then(|n| n.as_str().map(|s| s.to_string()))
            }).collect())
            .unwrap_or_default()
    }

    fn build_context_map(&self, clinical_context: &ClinicalContextData, calculation_results: &HashMap<String, f64>) -> HashMap<String, serde_json::Value> {
        let mut context = HashMap::new();
        
        if let Some(age) = clinical_context.age {
            context.insert("patient_age".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(age).unwrap()));
        }
        if let Some(weight) = clinical_context.weight_kg {
            context.insert("patient_weight".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(weight).unwrap()));
        }
        if let Some(egfr) = clinical_context.egfr {
            context.insert("patient_egfr".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(egfr).unwrap()));
        }

        for (key, value) in calculation_results {
            context.insert(key.clone(), serde_json::Value::Number(serde_json::Number::from_f64(*value).unwrap()));
        }

        context
    }
}

/// Clinical context data extracted from Go engine
#[derive(Debug, Clone)]
pub struct ClinicalContextData {
    pub age: Option<f64>,
    pub weight_kg: Option<f64>,
    pub height_cm: Option<f64>,
    pub gender: Option<String>,
    pub egfr: Option<f64>,
    pub creatinine: Option<f64>,
    pub conditions: Vec<String>,
    pub allergies: Vec<String>,
    pub current_medications: Vec<String>,
}

/// Safety assessment result
#[derive(Debug, Clone)]
pub struct SafetyAssessment {
    pub overall_status: String,
    pub safety_alerts: Vec<String>,
    pub contraindications: Vec<String>,
}
