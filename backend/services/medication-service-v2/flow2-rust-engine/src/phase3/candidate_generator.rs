//! Phase 3a: Candidate Generation with Safety Vetting
//! 
//! Generates medication candidates from therapy options and performs parallel safety vetting
//! including DDI checks, allergy screening, renal safety, and pregnancy considerations.
//! Target: ≤25ms processing time.

use std::sync::Arc;
use std::time::{Duration, Instant};
use anyhow::{Result, anyhow};
use tokio::task::JoinSet;
use tracing::{info, warn, error};

use super::models::*;
use super::apollo_client::ApolloFederationClient;
use super::performance::Phase3Metrics;
use crate::jit_safety::{JitSafetyEngine, SafetyRequest, SafetyResponse};

/// Candidate Generator for Phase 3a
pub struct CandidateGenerator {
    apollo_client: ApolloFederationClient,
    safety_engine: Arc<JitSafetyEngine>,
    metrics: Arc<Phase3Metrics>,
    max_candidates: usize,
    parallel_workers: usize,
}

impl CandidateGenerator {
    /// Create new candidate generator
    pub fn new(
        apollo_client: ApolloFederationClient,
        metrics: Arc<Phase3Metrics>,
    ) -> Result<Self> {
        // Initialize JIT safety engine for parallel safety checks
        let safety_engine = Arc::new(JitSafetyEngine::new()?);
        
        let max_candidates = std::env::var("MAX_CANDIDATES")
            .unwrap_or("20".to_string())
            .parse::<usize>()
            .unwrap_or(20);
            
        let parallel_workers = std::env::var("CANDIDATE_WORKERS")
            .unwrap_or_else(|_| num_cpus::get().min(10).to_string())
            .parse::<usize>()
            .unwrap_or(5);
        
        Ok(Self {
            apollo_client,
            safety_engine,
            metrics,
            max_candidates,
            parallel_workers,
        })
    }
    
    /// Generate and vet medication candidates from therapy options
    pub async fn generate_candidates(&self, input: &Phase3Input) -> Result<CandidateSet> {
        let start_time = Instant::now();
        
        info!(
            "🎯 Phase 3a: Generating candidates for {} therapy options",
            input.manifest.therapy_options.len()
        );
        
        // Step 1: Fetch medication candidates from Apollo Federation
        let raw_candidates = self.fetch_medication_candidates(input).await?;
        
        info!(
            "📚 Fetched {} raw medication candidates from KB",
            raw_candidates.len()
        );
        
        // Step 2: Perform parallel safety vetting
        let vetted_candidates = self.perform_parallel_safety_vetting(
            raw_candidates,
            input
        ).await?;
        
        info!(
            "🛡️ Safety vetting completed: {} candidates passed screening",
            vetted_candidates.len()
        );
        
        // Step 3: Filter absolute contraindications
        let safe_candidates = self.filter_absolute_contraindications(
            vetted_candidates,
            input
        ).await?;
        
        let generation_duration = start_time.elapsed();
        
        // Build candidate set result
        let candidate_set = CandidateSet {
            request_id: input.request_id.clone(),
            initial_count: raw_candidates.len(),
            vetted_count: vetted_candidates.len(),
            safe_count: safe_candidates.len(),
            candidates: safe_candidates,
            generation_duration,
            evidence: self.create_candidate_evidence(input),
        };
        
        // Record metrics
        self.metrics.record_candidate_generation(&candidate_set);
        
        // Check sub-phase SLA (≤25ms target)
        if generation_duration.as_millis() > 25 {
            warn!(
                "⚠️ Phase 3a SLA exceeded: {:?} > 25ms for request_id: {}",
                generation_duration,
                input.request_id
            );
        } else {
            info!(
                "✅ Phase 3a SLA met: {:?} ≤ 25ms for request_id: {}",
                generation_duration,
                input.request_id
            );
        }
        
        Ok(candidate_set)
    }
    
    /// Fetch medication candidates via Apollo Federation
    async fn fetch_medication_candidates(
        &self,
        input: &Phase3Input,
    ) -> Result<Vec<MedicationCandidate>> {
        let therapy_classes: Vec<String> = input.manifest.therapy_options
            .iter()
            .map(|opt| opt.therapy_class.clone())
            .collect();
        
        // Build versioned GraphQL query for medication candidates
        let query = r#"
            query GetMedicationCandidates($therapyClasses: [String!]!, $kbVersions: KBVersionInput!) {
                kb_patient_safe_checks(version: $kbVersions.safety) {
                    medicationsByClass(classes: $therapyClasses) {
                        medicationId
                        rxnorm
                        name
                        class
                        subClass
                        contraindications {
                            conditionCode
                            severity
                            type
                        }
                        precautions {
                            condition
                            adjustmentNeeded
                            monitoringRequired
                        }
                        blackBoxWarning
                    }
                }
                
                kb_formulary_stock(version: $kbVersions.formulary) {
                    availability(therapyClasses: $therapyClasses) {
                        medicationId
                        inStock
                        tier
                        preferredAlternative
                        priorAuthRequired
                    }
                }
            }
        "#;
        
        let variables = serde_json::json!({
            "therapyClasses": therapy_classes,
            "kbVersions": {
                "safety": input.evidence_envelope.kb_versions.get("kb_patient_safe_checks"),
                "formulary": input.evidence_envelope.kb_versions.get("kb_formulary_stock")
            }
        });
        
        let response = self.apollo_client.query(query, variables).await?;
        
        // Parse response into medication candidates
        self.parse_medication_candidates(response)
    }
    
    /// Parse Apollo Federation response into medication candidates
    fn parse_medication_candidates(
        &self,
        response: serde_json::Value,
    ) -> Result<Vec<MedicationCandidate>> {
        let mut candidates = Vec::new();
        
        // Extract medications from kb_patient_safe_checks
        if let Some(safety_data) = response.get("data")
            .and_then(|d| d.get("kb_patient_safe_checks"))
            .and_then(|kb| kb.get("medicationsByClass"))
        {
            if let Some(medications) = safety_data.as_array() {
                for med in medications {
                    let candidate = MedicationCandidate {
                        id: med.get("medicationId")
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        rxnorm: med.get("rxnorm")
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        name: med.get("name")
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        class: med.get("class")
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        subclass: med.get("subClass")
                            .and_then(|v| v.as_str())
                            .unwrap_or_default()
                            .to_string(),
                        contraindications: self.parse_contraindications(
                            med.get("contraindications")
                        ),
                        precautions: self.parse_precautions(
                            med.get("precautions")
                        ),
                        black_box_warning: med.get("blackBoxWarning")
                            .and_then(|v| v.as_bool())
                            .unwrap_or(false),
                    };
                    
                    candidates.push(candidate);
                }
            }
        }
        
        // Limit to max candidates
        if candidates.len() > self.max_candidates {
            candidates.truncate(self.max_candidates);
            info!(
                "📊 Limited candidates to {} (from {})",
                self.max_candidates,
                candidates.len()
            );
        }
        
        Ok(candidates)
    }
    
    /// Perform parallel safety vetting on all candidates
    async fn perform_parallel_safety_vetting(
        &self,
        candidates: Vec<MedicationCandidate>,
        input: &Phase3Input,
    ) -> Result<Vec<VettedCandidate>> {
        let mut join_set = JoinSet::new();
        
        // Create safety vetting tasks
        for candidate in candidates {
            let safety_engine = self.safety_engine.clone();
            let enriched_context = input.enriched_context.clone();
            
            join_set.spawn(async move {
                Self::vet_single_candidate(safety_engine, candidate, enriched_context).await
            });
        }
        
        // Collect results as they complete
        let mut vetted_candidates = Vec::new();
        
        while let Some(result) = join_set.join_next().await {
            match result {
                Ok(Ok(vetted_candidate)) => {
                    vetted_candidates.push(vetted_candidate);
                }
                Ok(Err(e)) => {
                    error!("Safety vetting failed for candidate: {}", e);
                    // Continue with other candidates
                }
                Err(e) => {
                    error!("Task join error during safety vetting: {}", e);
                    // Continue with other candidates
                }
            }
        }
        
        Ok(vetted_candidates)
    }
    
    /// Vet a single candidate for safety
    async fn vet_single_candidate(
        safety_engine: Arc<JitSafetyEngine>,
        candidate: MedicationCandidate,
        enriched_context: EnrichedContext,
    ) -> Result<VettedCandidate> {
        // Create safety request from candidate and context
        let safety_request = SafetyRequest {
            medication_id: candidate.id.clone(),
            medication_name: candidate.name.clone(),
            patient_age: enriched_context.demographics.age,
            patient_sex: enriched_context.demographics.sex.clone(),
            patient_weight: enriched_context.demographics.weight,
            current_medications: enriched_context.current_medications.medications
                .iter()
                .map(|med| med.medication_name.clone())
                .collect(),
            allergies: enriched_context.allergies
                .iter()
                .map(|allergy| allergy.allergen.clone())
                .collect(),
            renal_function: enriched_context.lab_results.egfr,
            pregnancy_status: enriched_context.demographics.pregnancy_status.clone(),
            active_conditions: enriched_context.active_conditions.clone(),
        };
        
        // Perform safety evaluation
        let safety_response = safety_engine.evaluate_safety(safety_request).await?;
        
        // Convert to vetted candidate
        let vetted_candidate = VettedCandidate {
            medication: candidate,
            safety_score: safety_response.safety_score,
            contraindicated: safety_response.contraindicated,
            safety_checks: safety_response.findings
                .into_iter()
                .map(|finding| SafetyCheck {
                    check_type: finding.check_type,
                    severity: finding.severity,
                    finding: finding.message,
                    action_required: finding.action_required,
                    evidence_level: finding.evidence_level,
                })
                .collect(),
            dose_adjustment_factor: safety_response.dose_adjustment_factor,
        };
        
        Ok(vetted_candidate)
    }
    
    /// Filter candidates with absolute contraindications
    async fn filter_absolute_contraindications(
        &self,
        candidates: Vec<VettedCandidate>,
        input: &Phase3Input,
    ) -> Result<Vec<VettedCandidate>> {
        let safe_candidates: Vec<VettedCandidate> = candidates
            .into_iter()
            .filter(|candidate| {
                // Keep candidates that are not absolutely contraindicated
                !candidate.contraindicated && candidate.safety_score > 0.0
            })
            .collect();
        
        info!(
            "🚫 Filtered out {} absolutely contraindicated candidates",
            candidates.len() - safe_candidates.len()
        );
        
        Ok(safe_candidates)
    }
    
    /// Create evidence trail for candidate generation
    fn create_candidate_evidence(&self, input: &Phase3Input) -> Vec<CandidateEvidence> {
        vec![
            CandidateEvidence {
                candidate_id: "all_candidates".to_string(),
                generation_method: "apollo_federation_query".to_string(),
                kb_versions_used: input.evidence_envelope.kb_versions.clone(),
                safety_checks_performed: vec![
                    "drug_drug_interactions".to_string(),
                    "allergy_screening".to_string(),
                    "renal_safety".to_string(),
                    "pregnancy_safety".to_string(),
                    "contraindication_filtering".to_string(),
                ],
                filtering_criteria: vec![
                    format!("therapy_classes: {:?}", 
                        input.manifest.therapy_options.iter()
                            .map(|opt| &opt.therapy_class)
                            .collect::<Vec<_>>()),
                    format!("max_candidates: {}", self.max_candidates),
                    "absolute_contraindications_excluded".to_string(),
                    "safety_score_threshold: 0.0".to_string(),
                ],
            }
        ]
    }
    
    /// Parse contraindications from GraphQL response
    fn parse_contraindications(
        &self,
        contraindications: Option<&serde_json::Value>,
    ) -> Vec<Contraindication> {
        let mut result = Vec::new();
        
        if let Some(contras) = contraindications.and_then(|v| v.as_array()) {
            for contra in contras {
                let contraindication = Contraindication {
                    condition_code: contra.get("conditionCode")
                        .and_then(|v| v.as_str())
                        .unwrap_or_default()
                        .to_string(),
                    severity: contra.get("severity")
                        .and_then(|v| v.as_str())
                        .unwrap_or_default()
                        .to_string(),
                    contraindication_type: contra.get("type")
                        .and_then(|v| v.as_str())
                        .unwrap_or("relative")
                        .to_string(),
                };
                
                result.push(contraindication);
            }
        }
        
        result
    }
    
    /// Parse precautions from GraphQL response
    fn parse_precautions(
        &self,
        precautions: Option<&serde_json::Value>,
    ) -> Vec<Precaution> {
        let mut result = Vec::new();
        
        if let Some(precauts) = precautions.and_then(|v| v.as_array()) {
            for precaut in precauts {
                let precaution = Precaution {
                    condition: precaut.get("condition")
                        .and_then(|v| v.as_str())
                        .unwrap_or_default()
                        .to_string(),
                    adjustment_needed: precaut.get("adjustmentNeeded")
                        .and_then(|v| v.as_bool())
                        .unwrap_or(false),
                    monitoring_required: precaut.get("monitoringRequired")
                        .and_then(|v| v.as_bool())
                        .unwrap_or(false),
                };
                
                result.push(precaution);
            }
        }
        
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::phase3::models::*;
    use chrono::Utc;
    use std::collections::HashMap;
    
    fn create_test_input() -> Phase3Input {
        let intent_manifest = IntentManifest {
            manifest_id: "test_manifest".to_string(),
            request_id: "test_request".to_string(),
            generated_at: Utc::now(),
            primary_intent: ClinicalIntent {
                category: "TREATMENT".to_string(),
                condition: "hypertension".to_string(),
                severity: "MODERATE".to_string(),
                phenotype: "standard".to_string(),
                time_horizon: "CHRONIC".to_string(),
            },
            secondary_intents: vec![],
            protocol_id: "hypertension_protocol".to_string(),
            protocol_version: "1.0".to_string(),
            evidence_grade: "A".to_string(),
            context_recipe_id: "context_recipe_1".to_string(),
            clinical_recipe_id: "clinical_recipe_1".to_string(),
            required_fields: vec![],
            optional_fields: vec![],
            data_freshness: FreshnessRequirements {
                max_age: Duration::from_secs(3600),
                critical_fields: vec![],
                preferred_sources: vec![],
            },
            snapshot_ttl: 3600,
            therapy_options: vec![
                TherapyCandidate {
                    therapy_class: "ACE_INHIBITOR".to_string(),
                    preference_order: 1,
                    rationale: "First-line therapy".to_string(),
                    guideline_source: "AHA/ACC".to_string(),
                }
            ],
            orb_version: "1.0".to_string(),
            rules_applied: vec![],
        };
        
        let enriched_context = EnrichedContext {
            demographics: Demographics {
                age: 55.0,
                sex: "male".to_string(),
                weight: 80.0,
                height: 175.0,
                bmi: Some(26.1),
                pregnancy_status: "not_applicable".to_string(),
                region: "US".to_string(),
            },
            lab_results: LabResults {
                egfr: Some(90.0),
                creatinine: Some(1.0),
                bilirubin: None,
                albumin: None,
                alt: None,
                ast: None,
                hemoglobin: None,
                platelet_count: None,
                inr: None,
            },
            vital_signs: VitalSigns {
                systolic_bp: Some(145.0),
                diastolic_bp: Some(92.0),
                heart_rate: Some(75.0),
                temperature: None,
                respiratory_rate: None,
                oxygen_saturation: None,
            },
            current_medications: CurrentMedications {
                medications: vec![],
                count: 0,
            },
            allergies: vec![],
            active_conditions: vec!["I10".to_string()],
            phenotype: "standard_hypertension".to_string(),
            risk_factors: vec![],
            patient_preferences: None,
            clinical_constraints: vec![],
        };
        
        let evidence_envelope = EvidenceEnvelope {
            envelope_id: "evidence_1".to_string(),
            created_at: Utc::now(),
            kb_versions: HashMap::new(),
            snapshot_hash: "hash123".to_string(),
            signature: None,
            audit_id: "audit_1".to_string(),
            processing_chain: vec!["phase1".to_string(), "phase2".to_string()],
        };
        
        Phase3Input::new(intent_manifest, enriched_context, evidence_envelope)
    }
    
    #[test]
    fn test_candidate_generator_creation() {
        let apollo_client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string()).unwrap();
        let metrics = Arc::new(Phase3Metrics::new());
        
        let generator = CandidateGenerator::new(apollo_client, metrics);
        assert!(generator.is_ok());
    }
    
    #[test]
    fn test_parse_contraindications() {
        let apollo_client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string()).unwrap();
        let metrics = Arc::new(Phase3Metrics::new());
        let generator = CandidateGenerator::new(apollo_client, metrics).unwrap();
        
        let contraindications_json = serde_json::json!([
            {
                "conditionCode": "N18",
                "severity": "absolute",
                "type": "absolute"
            }
        ]);
        
        let parsed = generator.parse_contraindications(Some(&contraindications_json));
        assert_eq!(parsed.len(), 1);
        assert_eq!(parsed[0].condition_code, "N18");
        assert_eq!(parsed[0].severity, "absolute");
    }
}