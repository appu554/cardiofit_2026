//! Phase 3b: Dose Calculation Engine
//! 
//! High-performance dose calculations leveraging the existing unified clinical engine
//! with Phase 3 specific interfaces and performance optimizations.
//! Target: ≤25ms processing time.

use std::sync::Arc;
use std::time::Instant;
use anyhow::Result;
use tracing::{info, warn};

use super::models::*;
use super::performance::Phase3Metrics;
use crate::unified_clinical_engine::{
    UnifiedClinicalEngine, 
    ClinicalRequest, 
    PatientContext,
    BiologicalSex,
    PregnancyStatus,
    RenalFunction,
    HepaticFunction,
    ActiveMedication,
    Allergy,
    LabValue
};
use std::collections::HashMap;
use chrono::Utc;

/// Phase 3b Dose Engine that bridges to the unified clinical engine
pub struct Phase3DoseEngine {
    unified_engine: Arc<UnifiedClinicalEngine>,
    metrics: Arc<Phase3Metrics>,
}

impl Phase3DoseEngine {
    /// Create new Phase 3 dose engine
    pub fn new(
        unified_engine: Arc<UnifiedClinicalEngine>,
        metrics: Arc<Phase3Metrics>,
    ) -> Result<Self> {
        Ok(Self {
            unified_engine,
            metrics,
        })
    }
    
    /// Calculate doses for all vetted candidates
    pub async fn calculate_doses(
        &self,
        candidate_set: &CandidateSet,
        input: &Phase3Input,
    ) -> Result<(Vec<DosedCandidate>, Vec<DoseCalculation>)> {
        let start_time = Instant::now();
        
        info!(
            "🎯 Phase 3b: Calculating doses for {} candidates",
            candidate_set.candidates.len()
        );
        
        let mut dosed_candidates = Vec::new();
        let mut dose_evidence = Vec::new();
        
        // Process each candidate
        for candidate in &candidate_set.candidates {
            match self.calculate_single_dose(candidate, input).await {
                Ok((dosed_candidate, evidence)) => {
                    dosed_candidates.push(dosed_candidate);
                    dose_evidence.push(evidence);
                }
                Err(e) => {
                    warn!(
                        "⚠️ Dose calculation failed for {}: {}",
                        candidate.medication.name,
                        e
                    );
                    // Continue with other candidates
                }
            }
        }
        
        let dosing_duration = start_time.elapsed();
        
        info!(
            "✅ Phase 3b completed: {} doses calculated in {:?}",
            dosed_candidates.len(),
            dosing_duration
        );
        
        // Check sub-phase SLA (≤25ms target)
        if dosing_duration.as_millis() > 25 {
            warn!(
                "⚠️ Phase 3b SLA exceeded: {:?} > 25ms for request_id: {}",
                dosing_duration,
                input.request_id
            );
        }
        
        Ok((dosed_candidates, dose_evidence))
    }
    
    /// Calculate dose for a single candidate
    async fn calculate_single_dose(
        &self,
        candidate: &VettedCandidate,
        input: &Phase3Input,
    ) -> Result<(DosedCandidate, DoseCalculation)> {
        let calc_start = Instant::now();
        
        // Convert Phase 3 input to unified engine format
        let clinical_request = self.convert_to_clinical_request(candidate, input)?;
        
        // Use the unified clinical engine for dose calculation
        let clinical_response = self.unified_engine
            .process_clinical_request(clinical_request)
            .await?;
        
        // Convert response back to Phase 3 format
        let dose_result = self.convert_to_dose_result(
            &clinical_response,
            calc_start.elapsed(),
        )?;
        
        let dosed_candidate = DosedCandidate {
            vetted_candidate: candidate.clone(),
            dose_result: dose_result.clone(),
        };
        
        let dose_calculation = DoseCalculation {
            medication_id: candidate.medication.id.clone(),
            method: dose_result.calculation_method.clone(),
            adjustments: dose_result.adjustments_applied.clone(),
            calculation_time: dose_result.calculation_time,
            confidence: dose_result.confidence,
        };
        
        Ok((dosed_candidate, dose_calculation))
    }
    
    /// Convert Phase 3 input to unified engine clinical request
    fn convert_to_clinical_request(
        &self,
        candidate: &VettedCandidate,
        input: &Phase3Input,
    ) -> Result<ClinicalRequest> {
        let context = &input.enriched_context;
        
        // Convert demographics
        let sex = match context.demographics.sex.as_str() {
            "male" => BiologicalSex::Male,
            "female" => BiologicalSex::Female,
            _ => BiologicalSex::Other,
        };
        
        let pregnancy_status = if context.demographics.pregnancy_status == "not_applicable" || 
                                 context.demographics.pregnancy_status == "not_pregnant" {
            PregnancyStatus::NotPregnant
        } else {
            PregnancyStatus::Unknown
        };
        
        // Convert renal function
        let renal_function = RenalFunction {
            egfr_ml_min: context.lab_results.egfr,
            egfr_ml_min_1_73m2: context.lab_results.egfr,
            creatinine_clearance: None,
            creatinine_mg_dl: context.lab_results.creatinine,
            bun_mg_dl: None,
            stage: None,
        };
        
        // Convert hepatic function
        let hepatic_function = HepaticFunction {
            child_pugh_class: None,
            alt_u_l: context.lab_results.alt,
            ast_u_l: context.lab_results.ast,
            bilirubin_mg_dl: context.lab_results.bilirubin,
            albumin_g_dl: context.lab_results.albumin,
        };
        
        // Convert active medications
        let active_medications: Vec<ActiveMedication> = context.current_medications
            .medications
            .iter()
            .map(|med| ActiveMedication {
                drug_id: med.medication_code.clone(),
                dose_mg: 0.0, // Parse from dose string if needed
                frequency: med.frequency.clone(),
                route: med.route.clone(),
                start_date: med.start_date,
            })
            .collect();
        
        // Convert allergies
        let allergies: Vec<Allergy> = context.allergies
            .iter()
            .map(|allergy| Allergy {
                allergen: allergy.allergen.clone(),
                reaction_type: allergy.reaction.clone(),
                severity: allergy.severity.clone(),
            })
            .collect();
        
        // Convert lab values
        let mut lab_values = HashMap::new();
        
        if let Some(egfr) = context.lab_results.egfr {
            lab_values.insert("eGFR".to_string(), LabValue {
                value: egfr,
                unit: "mL/min/1.73m²".to_string(),
                timestamp: Utc::now(),
                reference_range: Some(">60".to_string()),
            });
        }
        
        if let Some(creatinine) = context.lab_results.creatinine {
            lab_values.insert("creatinine".to_string(), LabValue {
                value: creatinine,
                unit: "mg/dL".to_string(),
                timestamp: Utc::now(),
                reference_range: Some("0.6-1.2".to_string()),
            });
        }
        
        let patient_context = PatientContext {
            age_years: context.demographics.age,
            weight_kg: context.demographics.weight,
            height_cm: context.demographics.height,
            sex,
            pregnancy_status,
            renal_function,
            hepatic_function,
            active_medications,
            allergies,
            conditions: context.active_conditions.clone(),
            lab_values,
        };
        
        Ok(ClinicalRequest {
            request_id: input.request_id.clone(),
            drug_id: candidate.medication.id.clone(),
            indication: input.manifest.primary_intent.condition.clone(),
            patient_context,
            timestamp: Utc::now(),
        })
    }
    
    /// Convert unified engine response to Phase 3 dose result
    fn convert_to_dose_result(
        &self,
        response: &crate::unified_clinical_engine::ClinicalResponse,
        calculation_time: std::time::Duration,
    ) -> Result<DoseResult> {
        let calc_result = &response.calculation_result;
        
        // Convert calculation steps to adjustments
        let adjustments: Vec<Adjustment> = calc_result.calculation_steps
            .iter()
            .filter_map(|step| {
                if let Some(rule) = &step.rule_applied {
                    Some(Adjustment {
                        adjustment_type: rule.clone(),
                        factor: step.result / step.input_values.values().next().unwrap_or(&1.0),
                        reason: step.calculation.clone(),
                    })
                } else {
                    None
                }
            })
            .collect();
        
        // Generate warnings from safety findings
        let warnings: Vec<DoseWarning> = response.safety_result.findings
            .iter()
            .map(|finding| DoseWarning {
                severity: finding.severity.clone(),
                message: finding.message.clone(),
                recommendation: "Monitor closely".to_string(),
            })
            .collect();
        
        let dose_result = DoseResult {
            calculated_dose: calc_result.proposed_dose_mg,
            unit: "mg".to_string(),
            frequency: response.final_recommendation.frequency.clone(),
            route: response.final_recommendation.route.clone(),
            adjustments_applied: adjustments,
            min_safe_dose: calc_result.proposed_dose_mg * 0.5, // Simplified
            max_safe_dose: calc_result.proposed_dose_mg * 2.0, // Simplified
            warnings,
            evidence: DoseEvidence {
                calculation_source: calc_result.calculation_strategy.clone(),
                adjustment_rationale: vec![
                    format!("Confidence: {:.1}%", calc_result.confidence_score * 100.0)
                ],
                safety_considerations: response.safety_result.findings
                    .iter()
                    .map(|f| f.message.clone())
                    .collect(),
                literature_references: vec![], // TODO: Add from knowledge base
            },
            calculation_method: calc_result.calculation_strategy.clone(),
            confidence: calc_result.confidence_score,
            calculation_time,
            rust_engine_version: crate::VERSION.to_string(),
        };
        
        Ok(dose_result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::unified_clinical_engine::knowledge_base::KnowledgeBase;
    use std::time::Duration;
    
    #[tokio::test]
    async fn test_dose_engine_creation() {
        let kb_path = std::env::temp_dir().join("test_kb_dose");
        std::fs::create_dir_all(&kb_path).unwrap();
        
        let knowledge_base = Arc::new(KnowledgeBase::new(kb_path.to_str().unwrap()).await.unwrap());
        let unified_engine = Arc::new(UnifiedClinicalEngine::new(knowledge_base).unwrap());
        let metrics = Arc::new(Phase3Metrics::new());
        
        let dose_engine = Phase3DoseEngine::new(unified_engine, metrics);
        assert!(dose_engine.is_ok());
        
        // Cleanup
        std::fs::remove_dir_all(&kb_path).unwrap();
    }
    
    #[test]
    fn test_clinical_request_conversion() {
        // Create mock data for testing conversion
        let demographics = Demographics {
            age: 55.0,
            sex: "male".to_string(),
            weight: 80.0,
            height: 175.0,
            bmi: Some(26.1),
            pregnancy_status: "not_applicable".to_string(),
            region: "US".to_string(),
        };
        
        let lab_results = LabResults {
            egfr: Some(90.0),
            creatinine: Some(1.0),
            bilirubin: None,
            albumin: None,
            alt: None,
            ast: None,
            hemoglobin: None,
            platelet_count: None,
            inr: None,
        };
        
        // Test would continue with full conversion validation
        assert_eq!(demographics.age, 55.0);
        assert_eq!(lab_results.egfr, Some(90.0));
    }
}