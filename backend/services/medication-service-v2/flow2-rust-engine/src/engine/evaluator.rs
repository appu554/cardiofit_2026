//! Rule evaluation engine for clinical decision support

use crate::models::*;
use crate::EngineError;
use std::collections::HashMap;
use serde_json;
use tracing::{info, warn, debug};

/// Rule evaluation engine for processing ORB rules and clinical conditions
pub struct RuleEvaluator {
    knowledge_base: KnowledgeBase,
}

impl RuleEvaluator {
    /// Create a new rule evaluator
    pub fn new(knowledge_base: KnowledgeBase) -> Self {
        Self {
            knowledge_base,
        }
    }

    /// Evaluate medication request against ORB rules to generate Intent Manifest
    pub fn evaluate_medication_request(&self, request: &MedicationRequest) -> Result<IntentManifest, crate::EngineError> {
        info!("Evaluating medication request for patient: {}, medication: {}", 
              request.patient_id, request.medication_name);

        // Build evaluation context from request
        let context = self.build_evaluation_context(request)?;

        // Find matching ORB rule
        let matching_rule = self.find_matching_rule(&context)?;

        // Generate Intent Manifest from matching rule
        let intent_manifest = self.generate_intent_manifest(request, &matching_rule)?;

        info!("Generated Intent Manifest: recipe_id={}, priority={}", 
              intent_manifest.recipe_id, intent_manifest.priority);

        Ok(intent_manifest)
    }

    /// Build evaluation context from medication request
    fn build_evaluation_context(&self, request: &MedicationRequest) -> Result<HashMap<String, serde_json::Value>, crate::EngineError> {
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

            if let Some(gender) = &demographics.gender {
                context.insert("patient_gender".to_string(), serde_json::Value::String(gender.clone()));
            }
        }

        // Add clinical context if available
        if let Some(clinical_context) = &request.clinical_context {
            // Add active medications
            if !clinical_context.active_medications.is_empty() {
                let med_codes: Vec<serde_json::Value> = clinical_context.active_medications
                    .iter()
                    .map(|med| serde_json::Value::String(med.medication_code.clone()))
                    .collect();
                context.insert("current_medications".to_string(), serde_json::Value::Array(med_codes));
            }

            // Add allergies
            if !clinical_context.allergies.is_empty() {
                let allergens: Vec<serde_json::Value> = clinical_context.allergies
                    .iter()
                    .map(|allergy| serde_json::Value::String(allergy.allergen.clone()))
                    .collect();
                context.insert("known_allergies".to_string(), serde_json::Value::Array(allergens));
            }

            // Add conditions
            if !clinical_context.conditions.is_empty() {
                let condition_codes: Vec<serde_json::Value> = clinical_context.conditions
                    .iter()
                    .map(|condition| serde_json::Value::String(condition.code.clone()))
                    .collect();
                context.insert("active_conditions".to_string(), serde_json::Value::Array(condition_codes));
            }
        }

        debug!("Built evaluation context with {} facts", context.len());
        Ok(context)
    }

    /// Find matching ORB rule based on context
    fn find_matching_rule(&self, context: &HashMap<String, serde_json::Value>) -> Result<&ORBRule, crate::EngineError> {
        // Rules are pre-sorted by priority (highest first)
        for rule in &self.knowledge_base.orb_rules.rules {
            if self.evaluate_rule_conditions(rule, context) {
                info!("Rule matched: {} (priority: {})", rule.id, rule.priority);
                return Ok(rule);
            }
        }

        // If no specific rule matches, look for catch-all rule
        for rule in &self.knowledge_base.orb_rules.rules {
            if !rule.has_conditions() {
                info!("Using catch-all rule: {}", rule.id);
                return Ok(rule);
            }
        }

        Err(crate::EngineError::RuleEvaluation("No matching rule found".to_string()))
    }

    /// Evaluate rule conditions against context
    fn evaluate_rule_conditions(&self, rule: &ORBRule, context: &HashMap<String, serde_json::Value>) -> bool {
        // Handle rules with no conditions (catch-all rules)
        if !rule.has_conditions() {
            return true;
        }

        // Evaluate using the enhanced condition evaluation
        rule.conditions.evaluate(context)
    }

    /// Generate Intent Manifest from matching rule
    fn generate_intent_manifest(&self, request: &MedicationRequest, rule: &ORBRule) -> Result<IntentManifest, crate::EngineError> {
        let manifest = IntentManifestBuilder::new()
            .with_request_info(request.request_id.clone(), request.patient_id.clone())
            .with_recipe(rule.action.generate_manifest.recipe_id.clone())
            .with_variant(rule.action.generate_manifest.variant.clone())
            .with_data_requirements(rule.action.generate_manifest.data_manifest.required.clone())
            .with_priority(self.determine_priority(rule, request))
            .with_rationale(format!("Rule matched: {} - {}", rule.id, self.generate_rationale(rule, request)))
            .with_rule_info(rule.id.clone(), "2.0.0".to_string())
            .with_medication_info(
                request.medication_code.clone(),
                request.medication_name.clone(),
                request.patient_conditions.clone()
            )
            .with_estimated_time(self.estimate_execution_time(rule))
            .build().map_err(|e| EngineError::ValidationError(e))?;

        Ok(manifest)
    }

    /// Determine priority based on rule and request
    fn determine_priority(&self, rule: &ORBRule, request: &MedicationRequest) -> String {
        // High priority for safety rules
        if rule.is_safety_rule() {
            return "critical".to_string();
        }

        // High priority for elderly patients
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() {
                return "high".to_string();
            }
        }

        // High priority for renal impairment
        if let Some(demographics) = &request.patient_demographics {
            if demographics.has_renal_impairment() {
                return "high".to_string();
            }
        }

        // Default priority
        "medium".to_string()
    }

    /// Generate clinical rationale
    fn generate_rationale(&self, rule: &ORBRule, request: &MedicationRequest) -> String {
        let mut rationale_parts = Vec::new();

        // Add medication-specific rationale
        rationale_parts.push(format!("Medication: {}", request.medication_name));

        // Add condition-specific rationale
        if !request.patient_conditions.is_empty() {
            rationale_parts.push(format!("Conditions: {}", request.patient_conditions.join(", ")));
        }

        // Add demographic-specific rationale
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() {
                rationale_parts.push("Elderly patient requires special consideration".to_string());
            }
            if demographics.has_renal_impairment() {
                rationale_parts.push("Renal impairment requires dose adjustment".to_string());
            }
            if demographics.is_obese() {
                rationale_parts.push("Obesity may affect dosing calculations".to_string());
            }
        }

        // Add rule-specific rationale
        if let Some(metadata) = &rule.metadata {
            if let Some(domain) = &metadata.clinical_domain {
                rationale_parts.push(format!("Clinical domain: {}", domain));
            }
        }

        rationale_parts.join("; ")
    }

    /// Estimate execution time based on rule complexity
    fn estimate_execution_time(&self, rule: &ORBRule) -> u64 {
        let base_time = 50; // Base 50ms

        // Add time for complex conditions
        let condition_complexity = rule.conditions.all_of.len() + rule.conditions.any_of.len();
        let condition_time = condition_complexity as u64 * 5;

        // Add time for data requirements
        let data_complexity = rule.action.generate_manifest.data_manifest.required.len();
        let data_time = data_complexity as u64 * 10;

        base_time + condition_time + data_time
    }

    /// Get knowledge base summary
    pub fn get_knowledge_summary(&self) -> KnowledgeSummary {
        self.knowledge_base.summary()
    }

    /// Validate rule set integrity
    pub fn validate_rules(&self) -> Result<(), crate::EngineError> {
        let rules = &self.knowledge_base.orb_rules.rules;

        if rules.is_empty() {
            return Err(crate::EngineError::RuleEvaluation("No rules loaded".to_string()));
        }

        // Check for duplicate rule IDs
        let mut rule_ids = std::collections::HashSet::new();
        for rule in rules {
            if !rule_ids.insert(&rule.id) {
                return Err(crate::EngineError::RuleEvaluation(
                    format!("Duplicate rule ID: {}", rule.id)
                ));
            }
        }

        // Check for missing recipe references
        for rule in rules {
            let recipe_id = &rule.action.generate_manifest.recipe_id;
            if !self.knowledge_base.clinical_recipes.recipes.contains_key(recipe_id) {
                warn!("Rule {} references missing recipe: {}", rule.id, recipe_id);
            }
        }

        info!("Rule validation completed: {} rules validated", rules.len());
        Ok(())
    }
}
