//! Advanced Intent Manifest Generator with Clinical Intelligence

use crate::models::*;
use std::collections::HashMap;
use serde_json;
use tracing::{info, warn, debug};
use chrono::{DateTime, Utc};

/// Advanced Intent Manifest Generator with clinical intelligence
pub struct IntentManifestGenerator {
    knowledge_base: KnowledgeBase,
    clinical_intelligence: ClinicalIntelligenceEngine,
}

/// Clinical Intelligence Engine for enhanced manifest generation
pub struct ClinicalIntelligenceEngine {
    risk_assessment: RiskAssessmentEngine,
    priority_calculator: PriorityCalculator,
    data_optimizer: DataRequirementOptimizer,
}

/// Risk assessment engine for clinical decision support
pub struct RiskAssessmentEngine;

/// Priority calculator for clinical urgency
pub struct PriorityCalculator;

/// Data requirement optimizer for efficient context gathering
pub struct DataRequirementOptimizer;

impl IntentManifestGenerator {
    /// Create a new intent manifest generator
    pub fn new(knowledge_base: KnowledgeBase) -> Self {
        Self {
            knowledge_base,
            clinical_intelligence: ClinicalIntelligenceEngine::new(),
        }
    }

    /// ⭐ MAIN METHOD: Generate enhanced intent manifest with clinical intelligence
    pub async fn generate_enhanced_manifest(
        &self,
        request: &MedicationRequest,
        matched_rule: &ORBRule,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<EnhancedIntentManifest, crate::EngineError> {
        info!("Generating enhanced intent manifest for request: {}", request.request_id);

        // 1. Perform clinical risk assessment
        let risk_assessment = self.clinical_intelligence
            .risk_assessment
            .assess_clinical_risk(request, clinical_context)?;

        // 2. Calculate dynamic priority based on clinical factors
        let dynamic_priority = self.clinical_intelligence
            .priority_calculator
            .calculate_priority(request, &matched_rule, &risk_assessment)?;

        // 3. Optimize data requirements based on clinical context
        let optimized_data_requirements = self.clinical_intelligence
            .data_optimizer
            .optimize_data_requirements(
                &matched_rule.action.generate_manifest.data_manifest.required,
                request,
                clinical_context,
            )?;

        // 4. Generate clinical rationale with detailed reasoning
        let clinical_rationale = self.generate_detailed_rationale(
            request,
            matched_rule,
            &risk_assessment,
            clinical_context,
        )?;

        // 5. Estimate execution complexity and time
        let execution_estimate = self.estimate_execution_complexity(
            matched_rule,
            &optimized_data_requirements,
            &risk_assessment,
        )?;

        // 6. Generate alternative recipes if needed
        let alternative_recipes = self.generate_alternative_recipes(
            request,
            matched_rule,
            &risk_assessment,
        )?;

        // 7. Create enhanced intent manifest
        let enhanced_manifest = EnhancedIntentManifest {
            // Core manifest fields
            request_id: request.request_id.clone(),
            patient_id: request.patient_id.clone(),
            recipe_id: matched_rule.action.generate_manifest.recipe_id.clone(),
            variant: matched_rule.action.generate_manifest.variant.clone(),
            data_requirements: optimized_data_requirements,
            priority: dynamic_priority.level.clone(),
            clinical_rationale: clinical_rationale.summary.clone(),
            estimated_execution_time_ms: execution_estimate.estimated_time_ms,
            rule_id: matched_rule.id.clone(),
            rule_version: "2.0.0".to_string(),
            generated_at: Utc::now(),
            medication_code: request.medication_code.clone(),
            conditions: request.patient_conditions.clone(),

            // Enhanced fields
            risk_assessment,
            priority_details: dynamic_priority,
            clinical_rationale_details: clinical_rationale,
            execution_estimate,
            alternative_recipes: alternative_recipes.clone(),
            clinical_flags: self.generate_clinical_flags(request, clinical_context)?,
            monitoring_requirements: self.generate_monitoring_requirements(request, matched_rule)?,
            safety_considerations: self.generate_safety_considerations(request, clinical_context)?,
            
            // Metadata
            metadata: EnhancedManifestMetadata {
                generator_version: "2.0.0".to_string(),
                clinical_intelligence_enabled: true,
                risk_assessment_performed: true,
                data_optimization_applied: true,
                alternative_analysis_performed: !alternative_recipes.is_empty(),
                generation_time_ms: 0, // Will be set after generation
            },
        };

        info!("Enhanced intent manifest generated: recipe={}, priority={}, risk_level={}", 
              enhanced_manifest.recipe_id, 
              enhanced_manifest.priority, 
              enhanced_manifest.risk_assessment.overall_risk_level);

        Ok(enhanced_manifest)
    }

    /// Generate detailed clinical rationale with reasoning
    fn generate_detailed_rationale(
        &self,
        request: &MedicationRequest,
        rule: &ORBRule,
        risk_assessment: &ClinicalRiskAssessment,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<DetailedClinicalRationale, crate::EngineError> {
        let mut reasoning_steps = Vec::new();
        let mut clinical_factors = Vec::new();

        // Add medication-specific reasoning
        reasoning_steps.push(format!(
            "Medication {} (RxNorm: {}) selected for conditions: {}",
            request.medication_name,
            request.medication_code,
            request.patient_conditions.join(", ")
        ));

        // Add patient-specific factors
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() {
                clinical_factors.push("Elderly patient (≥65 years) - requires dose adjustment consideration".to_string());
                reasoning_steps.push("Age-related pharmacokinetic changes considered".to_string());
            }

            if demographics.has_renal_impairment() {
                clinical_factors.push("Renal impairment detected - dose adjustment required".to_string());
                reasoning_steps.push("Renal function assessment indicates dose modification needed".to_string());
            }

            if demographics.is_obese() {
                clinical_factors.push("Obesity detected - weight-based dosing considerations".to_string());
                reasoning_steps.push("Body weight and composition factors included in dosing".to_string());
            }
        }

        // Add rule-specific reasoning
        reasoning_steps.push(format!("Clinical rule {} matched with priority {}", rule.id, rule.priority));

        // Add risk-specific reasoning
        match risk_assessment.overall_risk_level.as_str() {
            "HIGH" => {
                reasoning_steps.push("High-risk patient identified - enhanced monitoring required".to_string());
                clinical_factors.push("High-risk profile requires additional safety measures".to_string());
            }
            "MEDIUM" => {
                reasoning_steps.push("Moderate risk profile - standard monitoring with vigilance".to_string());
            }
            _ => {
                reasoning_steps.push("Low risk profile - standard monitoring protocols".to_string());
            }
        }

        // Generate summary
        let summary = format!(
            "Clinical decision for {} based on {} with {} risk profile. {}",
            request.medication_name,
            request.patient_conditions.join(" and "),
            risk_assessment.overall_risk_level.to_lowercase(),
            if clinical_factors.is_empty() { 
                "Standard dosing protocol applicable".to_string() 
            } else { 
                "Special considerations required".to_string() 
            }
        );

        Ok(DetailedClinicalRationale {
            summary,
            reasoning_steps,
            clinical_factors,
            evidence_level: "A".to_string(), // TODO: Determine from evidence repository
            confidence_score: 0.85, // TODO: Calculate based on data completeness
        })
    }

    /// Estimate execution complexity and time
    fn estimate_execution_complexity(
        &self,
        rule: &ORBRule,
        data_requirements: &[String],
        risk_assessment: &ClinicalRiskAssessment,
    ) -> Result<ExecutionEstimate, crate::EngineError> {
        let base_time = 50; // Base 50ms

        // Add time for data requirements
        let data_complexity_time = data_requirements.len() as u64 * 5;

        // Add time for risk level
        let risk_complexity_time = match risk_assessment.overall_risk_level.as_str() {
            "HIGH" => 20, // Additional safety checks
            "MEDIUM" => 10,
            _ => 0,
        };

        // Add time for rule complexity
        let rule_complexity_time = (rule.conditions.all_of.len() + rule.conditions.any_of.len()) as u64 * 3;

        let estimated_time_ms = base_time + data_complexity_time + risk_complexity_time + rule_complexity_time;

        Ok(ExecutionEstimate {
            estimated_time_ms,
            complexity_score: self.calculate_complexity_score(rule, data_requirements, risk_assessment),
            resource_requirements: ResourceRequirements {
                cpu_intensive: risk_assessment.overall_risk_level == "HIGH",
                memory_intensive: data_requirements.len() > 10,
                io_intensive: data_requirements.iter().any(|req| req.contains("external")),
                network_calls_required: data_requirements.iter().filter(|req| req.contains("service")).count(),
            },
            parallel_execution_possible: true,
            caching_opportunities: data_requirements.iter()
                .filter(|req| req.contains("demographics") || req.contains("static"))
                .count(),
        })
    }

    /// Calculate complexity score
    fn calculate_complexity_score(
        &self,
        rule: &ORBRule,
        data_requirements: &[String],
        risk_assessment: &ClinicalRiskAssessment,
    ) -> f64 {
        let mut score = 1.0;

        // Rule complexity
        score += (rule.conditions.all_of.len() + rule.conditions.any_of.len()) as f64 * 0.1;

        // Data complexity
        score += data_requirements.len() as f64 * 0.05;

        // Risk complexity
        score += match risk_assessment.overall_risk_level.as_str() {
            "HIGH" => 0.5,
            "MEDIUM" => 0.2,
            _ => 0.0,
        };

        // Priority complexity
        score += if rule.is_safety_rule() { 0.3 } else { 0.0 };

        score.min(5.0) // Cap at 5.0
    }

    /// Generate alternative recipes
    fn generate_alternative_recipes(
        &self,
        request: &MedicationRequest,
        primary_rule: &ORBRule,
        risk_assessment: &ClinicalRiskAssessment,
    ) -> Result<Vec<AlternativeRecipe>, crate::EngineError> {
        let mut alternatives = Vec::new();

        // Find alternative rules for the same medication
        for rule in &self.knowledge_base.orb_rules.rules {
            if rule.id != primary_rule.id && 
               rule.action.generate_manifest.recipe_id.contains(&request.medication_code) {
                
                alternatives.push(AlternativeRecipe {
                    recipe_id: rule.action.generate_manifest.recipe_id.clone(),
                    variant: rule.action.generate_manifest.variant.clone(),
                    rationale: format!("Alternative approach for {}", request.medication_name),
                    suitability_score: self.calculate_suitability_score(rule, request, risk_assessment),
                    trade_offs: vec![
                        "Different calculation methodology".to_string(),
                        "May have different monitoring requirements".to_string(),
                    ],
                });
            }
        }

        // Sort by suitability score
        alternatives.sort_by(|a, b| b.suitability_score.partial_cmp(&a.suitability_score).unwrap());

        // Return top 3 alternatives
        Ok(alternatives.into_iter().take(3).collect())
    }

    /// Calculate suitability score for alternative recipes
    fn calculate_suitability_score(
        &self,
        rule: &ORBRule,
        request: &MedicationRequest,
        risk_assessment: &ClinicalRiskAssessment,
    ) -> f64 {
        let mut score = 0.5; // Base score

        // Higher priority rules get higher scores
        score += (rule.priority as f64) / 1000.0;

        // Adjust for risk level
        if risk_assessment.overall_risk_level == "HIGH" && rule.is_safety_rule() {
            score += 0.3;
        }

        // Adjust for patient factors
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() && rule.id.contains("elderly") {
                score += 0.2;
            }
            if demographics.has_renal_impairment() && rule.id.contains("renal") {
                score += 0.2;
            }
        }

        score.min(1.0) // Cap at 1.0
    }

    /// Generate clinical flags
    fn generate_clinical_flags(
        &self,
        request: &MedicationRequest,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<ClinicalFlag>, crate::EngineError> {
        let mut flags = Vec::new();

        // Patient demographic flags
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() {
                flags.push(ClinicalFlag {
                    flag_type: "DEMOGRAPHIC".to_string(),
                    severity: "MEDIUM".to_string(),
                    message: "Elderly patient - consider age-related pharmacokinetic changes".to_string(),
                    code: "ELDERLY_PATIENT".to_string(),
                });
            }

            if demographics.has_renal_impairment() {
                flags.push(ClinicalFlag {
                    flag_type: "ORGAN_FUNCTION".to_string(),
                    severity: "HIGH".to_string(),
                    message: "Renal impairment detected - dose adjustment required".to_string(),
                    code: "RENAL_IMPAIRMENT".to_string(),
                });
            }

            if demographics.is_obese() {
                flags.push(ClinicalFlag {
                    flag_type: "DEMOGRAPHIC".to_string(),
                    severity: "MEDIUM".to_string(),
                    message: "Obesity detected - consider weight-based dosing".to_string(),
                    code: "OBESITY".to_string(),
                });
            }
        }

        // Condition-specific flags
        for condition in &request.patient_conditions {
            match condition.to_lowercase().as_str() {
                "sepsis" => {
                    flags.push(ClinicalFlag {
                        flag_type: "CONDITION".to_string(),
                        severity: "HIGH".to_string(),
                        message: "Sepsis requires aggressive treatment and monitoring".to_string(),
                        code: "SEPSIS_ALERT".to_string(),
                    });
                }
                "heart_failure" | "chf" => {
                    flags.push(ClinicalFlag {
                        flag_type: "CONDITION".to_string(),
                        severity: "HIGH".to_string(),
                        message: "Heart failure - monitor fluid balance and renal function".to_string(),
                        code: "HEART_FAILURE_ALERT".to_string(),
                    });
                }
                _ => {}
            }
        }

        Ok(flags)
    }

    /// Generate monitoring requirements
    fn generate_monitoring_requirements(
        &self,
        request: &MedicationRequest,
        rule: &ORBRule,
    ) -> Result<Vec<MonitoringRequirement>, crate::EngineError> {
        let mut requirements = Vec::new();

        // Get monitoring profile from knowledge base
        if let Some(monitoring_profile) = self.knowledge_base
            .monitoring_database
            .profiles
            .get(&request.medication_code) {
            
            for param in &monitoring_profile.monitoring_parameters {
                requirements.push(MonitoringRequirement {
                    parameter: param.parameter.clone(),
                    frequency: param.frequency.clone(),
                    target_range: param.target_range.clone(),
                    alert_conditions: param.alert_conditions.clone(),
                    rationale: format!("Standard monitoring for {}", request.medication_name),
                    priority: if param.parameter.contains("creatinine") || param.parameter.contains("renal") {
                        "HIGH".to_string()
                    } else {
                        "MEDIUM".to_string()
                    },
                });
            }
        }

        // Add patient-specific monitoring
        if let Some(demographics) = &request.patient_demographics {
            if demographics.has_renal_impairment() {
                requirements.push(MonitoringRequirement {
                    parameter: "serum_creatinine".to_string(),
                    frequency: "daily".to_string(),
                    target_range: Some("baseline or improving".to_string()),
                    alert_conditions: vec!["increase >0.5 mg/dL from baseline".to_string()],
                    rationale: "Enhanced renal monitoring due to impaired function".to_string(),
                    priority: "HIGH".to_string(),
                });
            }
        }

        Ok(requirements)
    }

    /// Generate safety considerations
    fn generate_safety_considerations(
        &self,
        request: &MedicationRequest,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<SafetyConsideration>, crate::EngineError> {
        let mut considerations = Vec::new();

        // Medication-specific safety considerations
        match request.medication_code.as_str() {
            "11124" => { // Vancomycin
                considerations.push(SafetyConsideration {
                    category: "NEPHROTOXICITY".to_string(),
                    severity: "HIGH".to_string(),
                    description: "Vancomycin can cause nephrotoxicity, especially with prolonged use".to_string(),
                    mitigation_strategies: vec![
                        "Monitor serum creatinine daily".to_string(),
                        "Maintain adequate hydration".to_string(),
                        "Avoid concurrent nephrotoxic agents".to_string(),
                    ],
                    monitoring_parameters: vec!["serum_creatinine".to_string(), "urine_output".to_string()],
                });

                considerations.push(SafetyConsideration {
                    category: "OTOTOXICITY".to_string(),
                    severity: "MEDIUM".to_string(),
                    description: "Risk of hearing loss with high doses or prolonged therapy".to_string(),
                    mitigation_strategies: vec![
                        "Monitor trough levels".to_string(),
                        "Assess hearing if prolonged therapy".to_string(),
                    ],
                    monitoring_parameters: vec!["vancomycin_trough".to_string()],
                });
            }
            _ => {
                // Generic safety considerations
                considerations.push(SafetyConsideration {
                    category: "GENERAL".to_string(),
                    severity: "MEDIUM".to_string(),
                    description: "Standard medication safety monitoring".to_string(),
                    mitigation_strategies: vec!["Monitor for adverse effects".to_string()],
                    monitoring_parameters: vec!["clinical_response".to_string()],
                });
            }
        }

        Ok(considerations)
    }
}

impl ClinicalIntelligenceEngine {
    /// Create a new clinical intelligence engine
    pub fn new() -> Self {
        Self {
            risk_assessment: RiskAssessmentEngine,
            priority_calculator: PriorityCalculator,
            data_optimizer: DataRequirementOptimizer,
        }
    }
}

impl RiskAssessmentEngine {
    /// Assess clinical risk based on patient and medication factors
    pub fn assess_clinical_risk(
        &self,
        request: &MedicationRequest,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<ClinicalRiskAssessment, crate::EngineError> {
        let mut risk_factors = Vec::new();
        let mut total_risk_score: f32 = 0.0;

        // Assess demographic risk factors
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() {
                risk_factors.push(RiskFactor {
                    factor_type: "DEMOGRAPHIC".to_string(),
                    description: "Elderly patient (≥65 years)".to_string(),
                    severity: "MEDIUM".to_string(),
                    impact_score: 0.3,
                    evidence_level: "A".to_string(),
                });
                total_risk_score += 0.3;
            }

            if demographics.has_renal_impairment() {
                risk_factors.push(RiskFactor {
                    factor_type: "ORGAN_FUNCTION".to_string(),
                    description: "Renal impairment (eGFR < 60)".to_string(),
                    severity: "HIGH".to_string(),
                    impact_score: 0.5,
                    evidence_level: "A".to_string(),
                });
                total_risk_score += 0.5;
            }

            if demographics.is_obese() {
                risk_factors.push(RiskFactor {
                    factor_type: "DEMOGRAPHIC".to_string(),
                    description: "Obesity (BMI ≥ 30)".to_string(),
                    severity: "MEDIUM".to_string(),
                    impact_score: 0.2,
                    evidence_level: "B".to_string(),
                });
                total_risk_score += 0.2;
            }
        }

        // Assess condition-based risk factors
        for condition in &request.patient_conditions {
            match condition.to_lowercase().as_str() {
                "sepsis" => {
                    risk_factors.push(RiskFactor {
                        factor_type: "CONDITION".to_string(),
                        description: "Sepsis - critical condition requiring aggressive treatment".to_string(),
                        severity: "HIGH".to_string(),
                        impact_score: 0.6,
                        evidence_level: "A".to_string(),
                    });
                    total_risk_score += 0.6;
                }
                "heart_failure" | "chf" => {
                    risk_factors.push(RiskFactor {
                        factor_type: "CONDITION".to_string(),
                        description: "Heart failure - affects drug clearance and fluid balance".to_string(),
                        severity: "HIGH".to_string(),
                        impact_score: 0.4,
                        evidence_level: "A".to_string(),
                    });
                    total_risk_score += 0.4;
                }
                "diabetes" => {
                    risk_factors.push(RiskFactor {
                        factor_type: "CONDITION".to_string(),
                        description: "Diabetes - may affect drug metabolism and wound healing".to_string(),
                        severity: "MEDIUM".to_string(),
                        impact_score: 0.2,
                        evidence_level: "B".to_string(),
                    });
                    total_risk_score += 0.2;
                }
                _ => {}
            }
        }

        // Assess medication-specific risk factors
        match request.medication_code.as_str() {
            "11124" => { // Vancomycin
                risk_factors.push(RiskFactor {
                    factor_type: "MEDICATION".to_string(),
                    description: "Vancomycin - nephrotoxic and ototoxic potential".to_string(),
                    severity: "MEDIUM".to_string(),
                    impact_score: 0.3,
                    evidence_level: "A".to_string(),
                });
                total_risk_score += 0.3;
            }
            _ => {}
        }

        // Normalize risk score (cap at 1.0)
        total_risk_score = total_risk_score.min(1.0);

        // Determine overall risk level
        let overall_risk_level = match total_risk_score {
            score if score >= 0.8 => "CRITICAL",
            score if score >= 0.6 => "HIGH",
            score if score >= 0.3 => "MEDIUM",
            _ => "LOW",
        }.to_string();

        // Generate assessment rationale
        let assessment_rationale = if risk_factors.is_empty() {
            "Low risk profile with no significant risk factors identified".to_string()
        } else {
            format!(
                "Risk assessment based on {} factors: {}",
                risk_factors.len(),
                risk_factors.iter()
                    .map(|rf| rf.description.clone())
                    .collect::<Vec<_>>()
                    .join(", ")
            )
        };

        // Generate mitigation strategies
        let mitigation_strategies = self.generate_mitigation_strategies(&risk_factors, &overall_risk_level);

        Ok(ClinicalRiskAssessment {
            overall_risk_level,
            risk_factors,
            risk_score: total_risk_score as f64,
            assessment_rationale,
            mitigation_strategies,
        })
    }

    /// Generate mitigation strategies based on risk factors
    fn generate_mitigation_strategies(&self, risk_factors: &[RiskFactor], risk_level: &str) -> Vec<String> {
        let mut strategies = Vec::new();

        // General strategies based on risk level
        match risk_level {
            "CRITICAL" | "HIGH" => {
                strategies.push("Enhanced monitoring with frequent assessments".to_string());
                strategies.push("Consider dose reduction or alternative therapy".to_string());
                strategies.push("Multidisciplinary team consultation recommended".to_string());
            }
            "MEDIUM" => {
                strategies.push("Standard monitoring with increased vigilance".to_string());
                strategies.push("Consider dose adjustment based on response".to_string());
            }
            _ => {
                strategies.push("Standard monitoring protocols".to_string());
            }
        }

        // Specific strategies based on risk factors
        for risk_factor in risk_factors {
            match risk_factor.factor_type.as_str() {
                "ORGAN_FUNCTION" if risk_factor.description.contains("renal") => {
                    strategies.push("Daily renal function monitoring".to_string());
                    strategies.push("Dose adjustment based on creatinine clearance".to_string());
                }
                "CONDITION" if risk_factor.description.contains("sepsis") => {
                    strategies.push("Aggressive infection control measures".to_string());
                    strategies.push("Hemodynamic monitoring".to_string());
                }
                "DEMOGRAPHIC" if risk_factor.description.contains("elderly") => {
                    strategies.push("Start with lower doses and titrate carefully".to_string());
                    strategies.push("Monitor for drug accumulation".to_string());
                }
                _ => {}
            }
        }

        // Remove duplicates and return
        strategies.sort();
        strategies.dedup();
        strategies
    }
}

impl PriorityCalculator {
    /// Calculate dynamic priority based on clinical factors
    pub fn calculate_priority(
        &self,
        request: &MedicationRequest,
        rule: &ORBRule,
        risk_assessment: &ClinicalRiskAssessment,
    ) -> Result<DynamicPriority, crate::EngineError> {
        let base_priority = self.rule_priority_to_string(rule.priority);
        let mut adjustments = Vec::new();
        let mut priority_score = self.priority_to_score(&base_priority);

        // Adjust based on risk assessment
        match risk_assessment.overall_risk_level.as_str() {
            "CRITICAL" => {
                adjustments.push(PriorityAdjustment {
                    factor: "Critical risk level".to_string(),
                    adjustment: 0.4,
                    rationale: "Critical risk factors require immediate attention".to_string(),
                });
                priority_score += 0.4;
            }
            "HIGH" => {
                adjustments.push(PriorityAdjustment {
                    factor: "High risk level".to_string(),
                    adjustment: 0.2,
                    rationale: "High risk factors require elevated priority".to_string(),
                });
                priority_score += 0.2;
            }
            _ => {}
        }

        // Adjust based on patient conditions
        for condition in &request.patient_conditions {
            if condition.to_lowercase().contains("sepsis") {
                adjustments.push(PriorityAdjustment {
                    factor: "Sepsis condition".to_string(),
                    adjustment: 0.3,
                    rationale: "Sepsis is a life-threatening condition requiring urgent treatment".to_string(),
                });
                priority_score += 0.3;
                break; // Only add once
            }
        }

        // Adjust based on patient demographics
        if let Some(demographics) = &request.patient_demographics {
            if demographics.is_elderly() && demographics.has_renal_impairment() {
                adjustments.push(PriorityAdjustment {
                    factor: "Elderly with renal impairment".to_string(),
                    adjustment: 0.15,
                    rationale: "Combination of age and renal impairment increases complexity".to_string(),
                });
                priority_score += 0.15;
            }
        }

        // Normalize priority score
        priority_score = priority_score.min(1.0);

        // Determine final priority level
        let final_level = match priority_score {
            score if score >= 0.9 => "EMERGENCY",
            score if score >= 0.7 => "CRITICAL",
            score if score >= 0.5 => "HIGH",
            score if score >= 0.3 => "MEDIUM",
            _ => "LOW",
        }.to_string();

        // Generate rationale
        let rationale = if adjustments.is_empty() {
            format!("Priority maintained at {} based on rule priority", base_priority)
        } else {
            format!(
                "Priority elevated from {} to {} due to: {}",
                base_priority,
                final_level,
                adjustments.iter()
                    .map(|adj| adj.factor.clone())
                    .collect::<Vec<_>>()
                    .join(", ")
            )
        };

        Ok(DynamicPriority {
            level: final_level,
            base_priority,
            adjustments,
            final_score: priority_score,
            rationale,
        })
    }

    /// Convert rule priority to string
    fn rule_priority_to_string(&self, priority: i32) -> String {
        match priority {
            p if p >= 1000 => "CRITICAL".to_string(),
            p if p >= 500 => "HIGH".to_string(),
            p if p >= 100 => "MEDIUM".to_string(),
            _ => "LOW".to_string(),
        }
    }

    /// Convert priority string to score
    fn priority_to_score(&self, priority: &str) -> f64 {
        match priority {
            "EMERGENCY" => 0.9,
            "CRITICAL" => 0.7,
            "HIGH" => 0.5,
            "MEDIUM" => 0.3,
            _ => 0.1,
        }
    }
}

impl DataRequirementOptimizer {
    /// Optimize data requirements based on clinical context and patient factors
    pub fn optimize_data_requirements(
        &self,
        base_requirements: &[String],
        request: &MedicationRequest,
        clinical_context: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, crate::EngineError> {
        let mut optimized_requirements = base_requirements.to_vec();

        // Add patient-specific requirements
        if let Some(demographics) = &request.patient_demographics {
            if demographics.has_renal_impairment() {
                // Add enhanced renal monitoring
                if !optimized_requirements.iter().any(|req| req.contains("creatinine")) {
                    optimized_requirements.push("labs.serum_creatinine[latest]".to_string());
                    optimized_requirements.push("labs.bun[latest]".to_string());
                }
                if !optimized_requirements.iter().any(|req| req.contains("egfr")) {
                    optimized_requirements.push("labs.egfr[latest]".to_string());
                }
            }

            if demographics.is_elderly() {
                // Add geriatric-specific requirements
                if !optimized_requirements.iter().any(|req| req.contains("cognitive")) {
                    optimized_requirements.push("assessments.cognitive_status[latest]".to_string());
                }
            }
        }

        // Add condition-specific requirements
        for condition in &request.patient_conditions {
            match condition.to_lowercase().as_str() {
                "sepsis" => {
                    optimized_requirements.push("vitals.temperature[latest]".to_string());
                    optimized_requirements.push("labs.lactate[latest]".to_string());
                    optimized_requirements.push("labs.procalcitonin[latest]".to_string());
                }
                "heart_failure" | "chf" => {
                    optimized_requirements.push("vitals.weight[daily]".to_string());
                    optimized_requirements.push("labs.bnp[latest]".to_string());
                }
                _ => {}
            }
        }

        // Remove duplicates and sort
        optimized_requirements.sort();
        optimized_requirements.dedup();

        Ok(optimized_requirements)
    }
}
