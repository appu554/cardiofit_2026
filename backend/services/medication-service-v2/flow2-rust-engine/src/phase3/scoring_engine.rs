//! Phase 3c: Scoring and Ranking Engine
//! 
//! Multi-factor scoring system that evaluates medication candidates based on:
//! - Guideline adherence (25%)
//! - Patient-specific factors (20%)
//! - Safety profile (20%)
//! - Formulary preference (15%)
//! - Cost effectiveness (10%)
//! - Adherence likelihood (10%)
//! 
//! Target: ≤25ms processing time.

use std::sync::Arc;
use std::time::Instant;
use std::collections::HashMap;
use anyhow::{Result, anyhow};
use tracing::{info, warn};
use serde_json::json;

use super::models::*;
use super::apollo_client::ApolloFederationClient;
use super::performance::Phase3Metrics;

/// Scoring and Ranking Engine for Phase 3c
pub struct ScoringRankingEngine {
    apollo_client: ApolloFederationClient,
    metrics: Arc<Phase3Metrics>,
    scoring_weights: ScoringWeights,
}

impl ScoringRankingEngine {
    /// Create new scoring and ranking engine
    pub fn new(
        apollo_client: ApolloFederationClient,
        metrics: Arc<Phase3Metrics>,
    ) -> Result<Self> {
        let scoring_weights = ScoringWeights::default();
        
        Ok(Self {
            apollo_client,
            metrics,
            scoring_weights,
        })
    }
    
    /// Score and rank medication proposals
    pub async fn score_and_rank(
        &self,
        dosed_candidates: &[DosedCandidate],
        input: &Phase3Input,
    ) -> Result<Vec<MedicationProposal>> {
        let start_time = Instant::now();
        
        info!(
            "🎯 Phase 3c: Scoring {} dosed candidates",
            dosed_candidates.len()
        );
        
        // Step 1: Fetch scoring data from knowledge bases
        let scoring_data = self.fetch_scoring_data(dosed_candidates, input).await?;
        
        // Step 2: Calculate scores for each candidate
        let mut scored_proposals = Vec::new();
        
        for (index, candidate) in dosed_candidates.iter().enumerate() {
            let score_breakdown = self.calculate_comprehensive_score(
                candidate,
                &scoring_data,
                input,
            )?;
            
            let proposal = MedicationProposal::new(
                (index + 1) as i32, // Temporary rank, will be updated after sorting
                score_breakdown.total,
                MedicationDetails {
                    id: candidate.vetted_candidate.medication.id.clone(),
                    rxnorm: candidate.vetted_candidate.medication.rxnorm.clone(),
                    name: candidate.vetted_candidate.medication.name.clone(),
                    generic_name: candidate.vetted_candidate.medication.name.clone(),
                    class: candidate.vetted_candidate.medication.class.clone(),
                    subclass: candidate.vetted_candidate.medication.subclass.clone(),
                    indication: input.manifest.primary_intent.condition.clone(),
                },
                candidate.dose_result.clone(),
                SafetyAssessment {
                    safety_score: candidate.vetted_candidate.safety_score,
                    contraindicated: candidate.vetted_candidate.contraindicated,
                    safety_checks: candidate.vetted_candidate.safety_checks.clone(),
                    dose_adjustment_factor: candidate.vetted_candidate.dose_adjustment_factor,
                },
                score_breakdown,
                self.create_proposal_evidence(&candidate.vetted_candidate.medication, &scoring_data),
            );
            
            scored_proposals.push(proposal);
        }
        
        // Step 3: Sort by total score (descending)
        scored_proposals.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        
        // Step 4: Update ranks
        for (index, proposal) in scored_proposals.iter_mut().enumerate() {
            proposal.rank = (index + 1) as i32;
        }
        
        let scoring_duration = start_time.elapsed();
        
        info!(
            "✅ Phase 3c completed: {} proposals ranked in {:?}",
            scored_proposals.len(),
            scoring_duration
        );
        
        // Check sub-phase SLA (≤25ms target)
        if scoring_duration.as_millis() > 25 {
            warn!(
                "⚠️ Phase 3c SLA exceeded: {:?} > 25ms for request_id: {}",
                scoring_duration,
                input.request_id
            );
        }
        
        Ok(scored_proposals)
    }
    
    /// Fetch scoring data from multiple knowledge bases
    async fn fetch_scoring_data(
        &self,
        candidates: &[DosedCandidate],
        input: &Phase3Input,
    ) -> Result<ScoringData> {
        let medication_ids: Vec<String> = candidates
            .iter()
            .map(|c| c.vetted_candidate.medication.id.clone())
            .collect();
        
        let query = r#"
            query GetScoringData($medIds: [String!]!, $phenotype: String!, $region: String!, $versions: ScoringVersionInput!) {
                kb_guidelines(version: $versions.guidelines) {
                    guidelineRecommendations(
                        medicationIds: $medIds,
                        phenotype: $phenotype
                    ) {
                        medicationId
                        recommendationLevel
                        evidenceGrade
                        firstLine
                        alternativeTo
                        phenotypeSpecific
                    }
                }
                
                kb_formulary_stock(version: $versions.formulary) {
                    formularyDetails(medicationIds: $medIds) {
                        medicationId
                        inStock
                        tier
                        copay
                        priorAuthRequired
                        preferredStatus
                        stockLevel
                        averageCost
                    }
                }
                
                kb_resistance_profiles(version: $versions.resistance) {
                    resistanceData(
                        medicationIds: $medIds,
                        region: $region
                    ) {
                        medicationId
                        susceptibilityRate
                        resistancePattern
                        lastUpdated
                    }
                }
                
                kb_clinical_context(version: $versions.context) {
                    patientSpecificFactors(
                        phenotype: $phenotype,
                        medicationIds: $medIds
                    ) {
                        medicationId
                        phenotypeSpecificEfficacy
                        expectedOutcome
                        timeToEffect
                    }
                }
            }
        "#;
        
        let variables = json!({
            "medIds": medication_ids,
            "phenotype": input.enriched_context.phenotype,
            "region": input.enriched_context.demographics.region,
            "versions": {
                "guidelines": input.evidence_envelope.kb_versions.get("kb_guidelines"),
                "formulary": input.evidence_envelope.kb_versions.get("kb_formulary_stock"),
                "resistance": input.evidence_envelope.kb_versions.get("kb_resistance_profiles"),
                "context": input.evidence_envelope.kb_versions.get("kb_clinical_context")
            }
        });
        
        let response = self.apollo_client.query(query, variables).await?;
        self.parse_scoring_data(response)
    }
    
    /// Calculate comprehensive multi-factor score
    fn calculate_comprehensive_score(
        &self,
        candidate: &DosedCandidate,
        scoring_data: &ScoringData,
        input: &Phase3Input,
    ) -> Result<ScoreComponents> {
        let mut components = ScoreComponents::default();
        let med_id = &candidate.vetted_candidate.medication.id;
        
        // 1. Guideline Adherence Score (25%)
        components.guideline_adherence = self.calculate_guideline_score(
            med_id,
            scoring_data,
            &input.enriched_context.phenotype,
        )?;
        
        // 2. Patient-Specific Score (20%)
        components.patient_specific = self.calculate_patient_specific_score(
            candidate,
            input,
        )?;
        
        // 3. Safety Profile Score (20%)
        components.safety_profile = candidate.vetted_candidate.safety_score / 100.0;
        
        // 4. Formulary Preference Score (15%)
        components.formulary_preference = self.calculate_formulary_score(
            med_id,
            scoring_data,
        )?;
        
        // 5. Cost Effectiveness Score (10%)
        components.cost_effectiveness = self.calculate_cost_effectiveness_score(
            med_id,
            scoring_data,
        )?;
        
        // 6. Adherence Likelihood Score (10%)
        components.adherence_likelihood = self.calculate_adherence_score(
            &candidate.dose_result,
            input,
        )?;
        
        // Calculate weighted total
        components.total = 
            components.guideline_adherence * self.scoring_weights.guideline_adherence +
            components.patient_specific * self.scoring_weights.patient_specific +
            components.safety_profile * self.scoring_weights.safety_profile +
            components.formulary_preference * self.scoring_weights.formulary_preference +
            components.cost_effectiveness * self.scoring_weights.cost_effectiveness +
            components.adherence_likelihood * self.scoring_weights.adherence_likelihood;
        
        // Ensure total is within valid range [0.0, 1.0]
        components.total = components.total.max(0.0).min(1.0);
        
        Ok(components)
    }
    
    /// Calculate guideline adherence score
    fn calculate_guideline_score(
        &self,
        medication_id: &str,
        scoring_data: &ScoringData,
        phenotype: &str,
    ) -> Result<f64> {
        if let Some(guideline) = scoring_data.guidelines.get(medication_id) {
            let mut score = match guideline.recommendation_level.as_str() {
                "STRONG" => 1.0,
                "MODERATE" => 0.8,
                "WEAK" => 0.6,
                "INSUFFICIENT" => 0.3,
                _ => 0.5,
            };
            
            // Bonus for first-line therapy
            if guideline.first_line {
                score = (score * 1.2).min(1.0);
            }
            
            // Evidence grade adjustment
            let evidence_multiplier = match guideline.evidence_grade.as_str() {
                "A" => 1.0,
                "B" => 0.9,
                "C" => 0.8,
                "D" => 0.7,
                _ => 0.6,
            };
            
            score *= evidence_multiplier;
            
            // Phenotype-specific bonus
            if guideline.phenotype_specific == phenotype {
                score = (score * 1.1).min(1.0);
            }
            
            Ok(score)
        } else {
            Ok(0.5) // Default moderate score when no guideline data available
        }
    }
    
    /// Calculate patient-specific factors score
    fn calculate_patient_specific_score(
        &self,
        candidate: &DosedCandidate,
        input: &Phase3Input,
    ) -> Result<f64> {
        let mut score = 1.0;
        let context = &input.enriched_context;
        
        // Age appropriateness
        if context.demographics.age > 65.0 {
            // Check for geriatric considerations in dose adjustments
            let has_age_adjustment = candidate.dose_result.adjustments_applied
                .iter()
                .any(|adj| adj.adjustment_type == "GERIATRIC" || adj.adjustment_type == "AGE");
            
            if has_age_adjustment {
                score *= 1.0; // Properly adjusted
            } else {
                score *= 0.9; // May need age consideration
            }
        }
        
        if context.demographics.age < 18.0 {
            // Check for pediatric considerations
            let has_pediatric_adjustment = candidate.dose_result.adjustments_applied
                .iter()
                .any(|adj| adj.adjustment_type == "PEDIATRIC");
            
            if has_pediatric_adjustment {
                score *= 1.0; // Properly adjusted
            } else {
                score *= 0.8; // Pediatric use requires special consideration
            }
        }
        
        // Renal function appropriateness
        if let Some(egfr) = context.lab_results.egfr {
            if egfr < 60.0 {
                let has_renal_adjustment = candidate.dose_result.adjustments_applied
                    .iter()
                    .any(|adj| adj.adjustment_type == "RENAL");
                
                if has_renal_adjustment {
                    score *= 1.0; // Properly adjusted
                } else {
                    score *= 0.7; // May need renal adjustment
                }
            }
        }
        
        // Pregnancy considerations
        if context.demographics.pregnancy_status != "not_applicable" && context.demographics.pregnancy_status != "not_pregnant" {
            // Pregnancy requires special consideration
            score *= 0.6;
        }
        
        // Polypharmacy penalty
        if context.current_medications.count > 5 {
            score *= 0.95; // Small penalty for complexity
        }
        
        Ok(score.max(0.0).min(1.0))
    }
    
    /// Calculate formulary preference score
    fn calculate_formulary_score(
        &self,
        medication_id: &str,
        scoring_data: &ScoringData,
    ) -> Result<f64> {
        if let Some(formulary) = scoring_data.formulary.get(medication_id) {
            let mut score = if formulary.in_stock { 1.0 } else { 0.3 };
            
            // Tier preference (lower tier = higher score)
            let tier_score = match formulary.tier.as_str() {
                "1" => 1.0,
                "2" => 0.8,
                "3" => 0.6,
                "4" => 0.4,
                _ => 0.5,
            };
            
            score *= tier_score;
            
            // Prior authorization penalty
            if formulary.prior_auth_required {
                score *= 0.7;
            }
            
            // Preferred status bonus
            if formulary.preferred_status == "preferred" {
                score = (score * 1.2).min(1.0);
            }
            
            Ok(score)
        } else {
            Ok(0.5) // Default when formulary data unavailable
        }
    }
    
    /// Calculate cost effectiveness score
    fn calculate_cost_effectiveness_score(
        &self,
        medication_id: &str,
        scoring_data: &ScoringData,
    ) -> Result<f64> {
        if let Some(formulary) = scoring_data.formulary.get(medication_id) {
            if let Some(copay) = formulary.copay {
                // Simple cost scoring - lower copay = higher score
                let score = match copay {
                    c if c < 10.0 => 1.0,
                    c if c < 25.0 => 0.9,
                    c if c < 50.0 => 0.8,
                    c if c < 100.0 => 0.6,
                    _ => 0.4,
                };
                Ok(score)
            } else {
                Ok(0.7) // Default when cost data unavailable
            }
        } else {
            Ok(0.5) // Default when formulary data unavailable
        }
    }
    
    /// Calculate adherence likelihood score
    fn calculate_adherence_score(
        &self,
        dose_result: &DoseResult,
        input: &Phase3Input,
    ) -> Result<f64> {
        let mut score = 1.0;
        
        // Frequency impact on adherence
        let frequency_score = match dose_result.frequency.as_str() {
            "ONCE_DAILY" => 1.0,
            "TWICE_DAILY" => 0.9,
            "THREE_TIMES_DAILY" => 0.7,
            "FOUR_TIMES_DAILY" => 0.5,
            _ => 0.8,
        };
        
        score *= frequency_score;
        
        // Route impact on adherence
        let route_score = match dose_result.route.as_str() {
            "ORAL" => 1.0,
            "SUBLINGUAL" => 0.9,
            "SUBCUTANEOUS" => 0.8,
            "INTRAMUSCULAR" => 0.7,
            "INTRAVENOUS" => 0.5,
            _ => 0.8,
        };
        
        score *= route_score;
        
        // Polypharmacy impact
        if input.enriched_context.current_medications.count > 5 {
            score *= 0.9;
        }
        
        // Patient preferences if available
        if let Some(preferences) = &input.enriched_context.patient_preferences {
            if preferences.route_preferences.contains(&dose_result.route) {
                score = (score * 1.1).min(1.0);
            }
            
            if preferences.frequency_preference == dose_result.frequency {
                score = (score * 1.1).min(1.0);
            }
        }
        
        Ok(score.max(0.0).min(1.0))
    }
    
    /// Parse scoring data from GraphQL response
    fn parse_scoring_data(&self, response: serde_json::Value) -> Result<ScoringData> {
        let mut guidelines = HashMap::new();
        let mut formulary = HashMap::new();
        let mut resistance = HashMap::new();
        
        // Parse guidelines data
        if let Some(guideline_data) = response.get("data")
            .and_then(|d| d.get("kb_guidelines"))
            .and_then(|kb| kb.get("guidelineRecommendations"))
            .and_then(|gr| gr.as_array())
        {
            for guideline in guideline_data {
                if let Some(med_id) = guideline.get("medicationId").and_then(|v| v.as_str()) {
                    let recommendation = GuidelineRecommendation {
                        recommendation_level: guideline.get("recommendationLevel")
                            .and_then(|v| v.as_str())
                            .unwrap_or("MODERATE")
                            .to_string(),
                        evidence_grade: guideline.get("evidenceGrade")
                            .and_then(|v| v.as_str())
                            .unwrap_or("C")
                            .to_string(),
                        first_line: guideline.get("firstLine")
                            .and_then(|v| v.as_bool())
                            .unwrap_or(false),
                        alternative_to: vec![], // Simplified for now
                        phenotype_specific: guideline.get("phenotypeSpecific")
                            .and_then(|v| v.as_str())
                            .unwrap_or("")
                            .to_string(),
                    };
                    
                    guidelines.insert(med_id.to_string(), recommendation);
                }
            }
        }
        
        // Parse formulary data
        if let Some(formulary_data) = response.get("data")
            .and_then(|d| d.get("kb_formulary_stock"))
            .and_then(|kb| kb.get("formularyDetails"))
            .and_then(|fd| fd.as_array())
        {
            for formulary_entry in formulary_data {
                if let Some(med_id) = formulary_entry.get("medicationId").and_then(|v| v.as_str()) {
                    let formulary_info = FormularyData {
                        in_stock: formulary_entry.get("inStock")
                            .and_then(|v| v.as_bool())
                            .unwrap_or(true),
                        tier: formulary_entry.get("tier")
                            .and_then(|v| v.as_str())
                            .unwrap_or("2")
                            .to_string(),
                        copay: formulary_entry.get("copay")
                            .and_then(|v| v.as_f64()),
                        prior_auth_required: formulary_entry.get("priorAuthRequired")
                            .and_then(|v| v.as_bool())
                            .unwrap_or(false),
                        preferred_status: formulary_entry.get("preferredStatus")
                            .and_then(|v| v.as_str())
                            .unwrap_or("standard")
                            .to_string(),
                        average_cost: formulary_entry.get("averageCost")
                            .and_then(|v| v.as_f64()),
                    };
                    
                    formulary.insert(med_id.to_string(), formulary_info);
                }
            }
        }
        
        Ok(ScoringData {
            guidelines,
            formulary,
            resistance,
        })
    }
    
    /// Create proposal evidence
    fn create_proposal_evidence(
        &self,
        medication: &MedicationCandidate,
        scoring_data: &ScoringData,
    ) -> ProposalEvidence {
        let guideline_source = scoring_data.guidelines
            .get(&medication.id)
            .cloned()
            .unwrap_or_else(|| GuidelineRecommendation {
                recommendation_level: "MODERATE".to_string(),
                evidence_grade: "C".to_string(),
                first_line: false,
                alternative_to: vec![],
                phenotype_specific: "".to_string(),
            });
        
        let formulary_data = scoring_data.formulary
            .get(&medication.id)
            .cloned()
            .unwrap_or_else(|| FormularyData {
                in_stock: true,
                tier: "2".to_string(),
                copay: None,
                prior_auth_required: false,
                preferred_status: "standard".to_string(),
                average_cost: None,
            });
        
        let resistance_data = ResistanceData {
            susceptibility_rate: 95.0, // Default assumption
            resistance_pattern: "standard".to_string(),
            last_updated: chrono::Utc::now(),
        };
        
        ProposalEvidence {
            guideline_source,
            formulary_data,
            resistance_data,
        }
    }
}

/// Scoring data from knowledge bases
struct ScoringData {
    guidelines: HashMap<String, GuidelineRecommendation>,
    formulary: HashMap<String, FormularyData>,
    resistance: HashMap<String, ResistanceData>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::phase3::models::*;
    use chrono::Utc;
    use std::time::Duration;
    
    #[test]
    fn test_scoring_engine_creation() {
        let apollo_client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string()).unwrap();
        let metrics = Arc::new(Phase3Metrics::new());
        
        let engine = ScoringRankingEngine::new(apollo_client, metrics);
        assert!(engine.is_ok());
    }
    
    #[test]
    fn test_guideline_score_calculation() {
        let apollo_client = ApolloFederationClient::new("http://localhost:4000/graphql".to_string()).unwrap();
        let metrics = Arc::new(Phase3Metrics::new());
        let engine = ScoringRankingEngine::new(apollo_client, metrics).unwrap();
        
        let mut scoring_data = ScoringData {
            guidelines: HashMap::new(),
            formulary: HashMap::new(),
            resistance: HashMap::new(),
        };
        
        scoring_data.guidelines.insert("med1".to_string(), GuidelineRecommendation {
            recommendation_level: "STRONG".to_string(),
            evidence_grade: "A".to_string(),
            first_line: true,
            alternative_to: vec![],
            phenotype_specific: "standard".to_string(),
        });
        
        let score = engine.calculate_guideline_score("med1", &scoring_data, "standard").unwrap();
        assert!(score > 0.8); // Should be high score for strong recommendation with grade A evidence
    }
}