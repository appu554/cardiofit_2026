//! Enhanced JIT Safety Engine Integration
//! 
//! This module integrates the comprehensive JIT Safety Engine with our existing architecture.
//! It provides a bridge between the enhanced engine and our current data models.

use crate::jit_safety::{domain::*, error::JitSafetyError};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use std::sync::Arc;
use tracing::{debug, info, warn};

// Re-export the enhanced engine types
pub use super::super::super::jit_safety_engine::*;

/// Enhanced JIT Safety Engine wrapper that integrates with our current architecture
pub struct EnhancedJitEngine {
    inner_engine: JITSafetyEngine,
    engine_version: String,
}

impl EnhancedJitEngine {
    /// Create a new enhanced JIT Safety Engine
    pub fn new(
        ddi_kb: Arc<dyn DDIKnowledgeBase + Send + Sync>,
        drug_metadata_kb: Arc<dyn DrugMetadataKB + Send + Sync>,
        clinical_rules_kb: Arc<dyn ClinicalRulesKB + Send + Sync>,
        engine_version: &str,
    ) -> Self {
        let inner_engine = JITSafetyEngine::new(ddi_kb, drug_metadata_kb, clinical_rules_kb);
        
        Self {
            inner_engine,
            engine_version: engine_version.to_string(),
        }
    }

    /// Evaluate safety using our current JitSafetyContext format
    pub fn evaluate(&self, ctx: JitSafetyContext) -> Result<JitSafetyOutcome, JitSafetyError> {
        info!("Starting enhanced JIT Safety evaluation for request {}", ctx.request_id);

        // Convert our context to the enhanced format
        let enhanced_context = self.convert_to_enhanced_context(ctx)?;
        
        // Convert our proposal to enhanced candidate format
        let enhanced_candidate = self.convert_to_enhanced_candidate(&enhanced_context)?;

        // Run the enhanced safety check
        let safety_result = self.inner_engine.run_safety_check(&enhanced_candidate, &enhanced_context);

        // Convert back to our format
        let outcome = self.convert_to_our_outcome(safety_result, enhanced_context.request_id)?;

        info!("Enhanced JIT Safety evaluation completed: {:?}", outcome.decision);
        Ok(outcome)
    }

    /// Convert our JitSafetyContext to the enhanced PatientContext format
    fn convert_to_enhanced_context(&self, ctx: JitSafetyContext) -> Result<PatientContext, JitSafetyError> {
        let patient = &ctx.patient;
        
        // Convert pregnancy status
        let pregnancy_status = if patient.pregnancy {
            PregnancyStatus::Pregnant { trimester: 2 } // Default to 2nd trimester if not specified
        } else {
            PregnancyStatus::NotPregnant
        };

        // Convert biological sex
        let sex = match patient.sex.to_lowercase().as_str() {
            "male" => BiologicalSex::Male,
            "female" => BiologicalSex::Female,
            _ => BiologicalSex::Other,
        };

        // Convert lab results
        let labs = LabResults {
            egfr_ml_min: patient.renal.egfr.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "mL/min/1.73m²".to_string(),
            }),
            serum_creatinine_mg_dl: None, // Not available in our current format
            serum_potassium_mmol_l: patient.labs.potassium.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "mmol/L".to_string(),
            }),
            serum_sodium_mmol_l: patient.labs.sodium.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "mmol/L".to_string(),
            }),
            serum_magnesium_mmol_l: None,
            alt_u_l: patient.labs.alt.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "U/L".to_string(),
            }),
            ast_u_l: patient.labs.ast.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "U/L".to_string(),
            }),
            total_bilirubin_mg_dl: None,
            albumin_g_dl: None,
            inr: None,
            platelet_count: None,
            hemoglobin_g_dl: None,
            hba1c_percent: patient.labs.hba1c.map(|v| LabValue {
                value: v,
                timestamp: Utc::now(),
                unit: "%".to_string(),
            }),
            tsh_miu_l: None,
            qtc_ms: patient.qtc_ms.map(|v| LabValue {
                value: v as f64,
                timestamp: Utc::now(),
                unit: "ms".to_string(),
            }),
        };

        // Convert allergies
        let allergies: Vec<Allergy> = patient.allergies.iter().map(|allergy_code| {
            Allergy {
                allergen: allergy_code.clone(),
                reaction_type: AllergyReactionType::Other("Unknown".to_string()),
                severity: AllergySeverity::Moderate, // Default severity
            }
        }).collect();

        // Convert conditions to ICD-10 codes
        let conditions: HashSet<String> = patient.conditions.iter().cloned().collect();

        // Convert concurrent medications
        let active_medications: Vec<ActiveMedication> = ctx.concurrent_meds.iter().map(|med| {
            ActiveMedication {
                drug_id: med.drug_id.clone(),
                name: med.drug_id.clone(), // Use drug_id as name for now
                generic_name: med.drug_id.clone(),
                dose_mg: med.dose_mg,
                route: self.convert_route(&med.route),
                frequency: Frequency {
                    times_per_day: (24 / med.interval_h.max(1)) as u8,
                    schedule: Some(format!("q{}h", med.interval_h)),
                    with_food: None,
                },
                start_date: Utc::now() - chrono::Duration::days(30), // Assume started 30 days ago
                last_taken: Some(Utc::now() - chrono::Duration::hours(med.interval_h as i64)),
                therapeutic_class: vec![med.class_id.clone()],
                mechanism_of_action: vec![],
            }
        }).collect();

        Ok(PatientContext {
            patient_id: ctx.request_id.clone(),
            age_years: patient.age_years as u8,
            sex,
            weight_kg: patient.weight_kg,
            height_cm: patient.height_cm.unwrap_or(170.0), // Default height if not provided
            pregnancy_status,
            breastfeeding: false, // Not available in our current format
            labs,
            conditions,
            recent_procedures: vec![], // Not available in our current format
            active_medications,
            allergies,
            pharmacogenomics: None, // Not available in our current format
            kb_versions: ctx.kb_versions,
            request_id: ctx.request_id,
            timestamp: Utc::now(),
        })
    }

    /// Convert our ProposedDose to enhanced DrugCandidate format
    fn convert_to_enhanced_candidate(&self, ctx: &PatientContext) -> Result<DrugCandidate, JitSafetyError> {
        // Extract proposal from the original context (we need to pass it separately)
        // For now, create a basic candidate - in real implementation, this would come from the original context
        Ok(DrugCandidate {
            drug_id: "example_drug".to_string(), // This should come from the original proposal
            name: "Example Drug".to_string(),
            generic_name: "example_drug".to_string(),
            proposed_dose_mg: 10.0, // This should come from the original proposal
            proposed_dose_unit: "mg".to_string(),
            route: Route::Oral, // This should come from the original proposal
            frequency: Frequency {
                times_per_day: 1,
                schedule: Some("q24h".to_string()),
                with_food: None,
            },
            duration_days: None,
            formulation: None,
            provenance: HashMap::new(),
        })
    }

    /// Convert enhanced SafetyResult back to our JitSafetyOutcome format
    fn convert_to_our_outcome(&self, result: SafetyResult, request_id: String) -> Result<JitSafetyOutcome, JitSafetyError> {
        // Convert decision
        let decision = match result.action {
            SafetyAction::Proceed => Decision::Allow,
            SafetyAction::ProceedWithMonitoring { .. } => Decision::Allow,
            SafetyAction::AdjustDose { .. } => Decision::AllowWithAdjustment,
            SafetyAction::HoldForClinician { .. } => Decision::Block,
            SafetyAction::AbortAndSwitch { .. } => Decision::Block,
            SafetyAction::RequireSpecialistReview { .. } => Decision::Block,
        };

        // Convert findings to reasons
        let reasons: Vec<Reason> = result.findings.iter().map(|finding| {
            let severity = match finding.severity {
                FindingSeverity::Info => "info",
                FindingSeverity::Warning => "warn",
                FindingSeverity::Major => "error",
                FindingSeverity::Critical => "blocker",
                FindingSeverity::Contraindicated => "blocker",
            };

            Reason {
                code: finding.code.clone(),
                severity: severity.to_string(),
                message: finding.message.clone(),
                evidence: finding.references.clone(),
                rule_id: finding.finding_id.clone(),
            }
        }).collect();

        // Convert DDI findings to DDI flags
        let ddis: Vec<DdiFlag> = result.findings.iter()
            .filter(|f| f.category == FindingCategory::DrugDrugInteraction)
            .map(|finding| {
                let severity = match finding.severity {
                    FindingSeverity::Info => "minor",
                    FindingSeverity::Warning => "moderate", 
                    FindingSeverity::Major => "major",
                    FindingSeverity::Critical => "contraindicated",
                    FindingSeverity::Contraindicated => "contraindicated",
                };

                DdiFlag {
                    with_drug_id: finding.details.get("interacting_drug").cloned().unwrap_or_default(),
                    severity: severity.to_string(),
                    action: finding.clinical_significance.clone(),
                    code: finding.code.clone(),
                    rule_id: finding.finding_id.clone(),
                }
            }).collect();

        // Create final dose (placeholder - should be extracted from action if adjusted)
        let final_dose = ProposedDose {
            drug_id: "example_drug".to_string(),
            dose_mg: 10.0,
            route: "po".to_string(),
            interval_h: 24,
        };

        // Convert provenance
        let provenance = Provenance {
            engine_version: self.engine_version.clone(),
            kb_versions: result.audit_trail.kb_versions,
            evaluation_trace: result.audit_trail.checks_performed.iter().map(|check| {
                EvalStep {
                    rule_id: check.clone(),
                    result: "completed".to_string(),
                }
            }).collect(),
        };

        Ok(JitSafetyOutcome {
            decision,
            final_dose,
            reasons,
            ddis,
            provenance,
        })
    }

    /// Convert route string to enhanced Route enum
    fn convert_route(&self, route: &str) -> Route {
        match route.to_lowercase().as_str() {
            "po" | "oral" => Route::Oral,
            "iv" => Route::IV,
            "im" => Route::IM,
            "sc" | "subcutaneous" => Route::Subcutaneous,
            "topical" => Route::Topical,
            "inhaled" => Route::Inhaled,
            "rectal" | "pr" => Route::Rectal,
            _ => Route::Other(route.to_string()),
        }
    }
}

/// Mock implementations for testing (replace with real implementations)
pub struct MockDDIKnowledgeBase;
pub struct MockDrugMetadataKB;
pub struct MockClinicalRulesKB;

impl DDIKnowledgeBase for MockDDIKnowledgeBase {
    fn check_interaction(&self, _drug_a: &str, _drug_b: &str, _dose_a_mg: f64, _dose_b_mg: f64) -> Result<Option<DDIRecord>, String> {
        Ok(None)
    }
    
    fn batch_check(&self, _candidate: &DrugCandidate, _active_meds: &[ActiveMedication]) -> Result<Vec<DDIRecord>, String> {
        Ok(vec![])
    }
    
    fn version(&self) -> String {
        "mock-1.0.0".to_string()
    }
}

impl DrugMetadataKB for MockDrugMetadataKB {
    fn get_metadata(&self, _drug_id: &str) -> Result<DrugMetadata, String> {
        Ok(DrugMetadata {
            drug_id: _drug_id.to_string(),
            pregnancy_category: Some("B".to_string()),
            lactation_risk: Some("L2".to_string()),
            renal_dosing: RenalDosingInfo {
                adjustments: vec![],
                contraindicated_below_egfr: None,
            },
            hepatic_dosing: HepaticDosingInfo {
                child_pugh_adjustments: HashMap::new(),
                contraindicated_in_cirrhosis: false,
            },
            geriatric_considerations: vec![],
            pediatric_dosing: None,
            black_box_warnings: vec![],
            beers_criteria: None,
            qt_prolongation_risk: QtRisk::None,
            anticholinergic_burden_score: 0,
            serotonergic_activity: false,
            narrow_therapeutic_index: false,
            cyp_interactions: CypInteractions {
                substrate_of: vec![],
                inhibits: vec![],
                induces: vec![],
            },
            therapeutic_monitoring: None,
        })
    }
    
    fn version(&self) -> String {
        "mock-1.0.0".to_string()
    }
}

impl ClinicalRulesKB for MockClinicalRulesKB {
    fn get_threshold(&self, key: &str) -> Option<f64> {
        match key {
            "k_high_threshold" => Some(5.5),
            "k_low_threshold" => Some(3.5),
            "qtc_threshold" => Some(500.0),
            "sglt2_surgery_hold_days" => Some(3.0),
            _ => None,
        }
    }
    
    fn get_rule(&self, _key: &str) -> Option<ClinicalRule> {
        None
    }
    
    fn version(&self) -> String {
        "mock-1.0.0".to_string()
    }
}
