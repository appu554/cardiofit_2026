//! Phase 3: Clinical Intelligence Engine
//! 
//! This module implements Phase 3 compliance for the Clinical Intelligence Engine
//! as specified in the Phase 3 Clinical Intelligence Engine documentation.
//! 
//! Phase 3 executes through three sequential stages:
//! - Phase 3a: Candidate Generation with Safety Vetting (≤25ms)
//! - Phase 3b: Rust Dose Calculation Engine (≤25ms)  
//! - Phase 3c: Scoring and Ranking Engine (≤25ms)
//! 
//! Total Phase 3 SLA: ≤75ms with deterministic, auditable clinical decisions.

use std::collections::HashMap;
use std::sync::Arc;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use anyhow::{Result, anyhow};
use async_trait::async_trait;

// Sub-modules for Phase 3 components
pub mod models;
pub mod candidate_generator;
pub mod dose_engine;
pub mod scoring_engine;
pub mod apollo_client;
pub mod ffi_bridge;
pub mod performance;
pub mod evidence;

// Re-export commonly used types
pub use models::*;
pub use candidate_generator::CandidateGenerator;
pub use dose_engine::Phase3DoseEngine;
pub use scoring_engine::ScoringRankingEngine;
pub use apollo_client::ApolloFederationClient;
pub use performance::Phase3Metrics;
pub use evidence::EvidenceEnvelope;

use crate::unified_clinical_engine::UnifiedClinicalEngine;

/// Main Phase 3 Clinical Intelligence Engine
pub struct ClinicalIntelligenceEngine {
    // Core engines
    candidate_generator: CandidateGenerator,
    dose_engine: Phase3DoseEngine,
    scoring_engine: ScoringRankingEngine,
    
    // External integrations
    apollo_client: ApolloFederationClient,
    
    // Evidence and performance tracking
    metrics: Arc<Phase3Metrics>,
    
    // Bridge to existing unified engine
    unified_engine: Arc<UnifiedClinicalEngine>,
}

impl ClinicalIntelligenceEngine {
    /// Create new Phase 3 Clinical Intelligence Engine
    pub fn new(unified_engine: Arc<UnifiedClinicalEngine>) -> Result<Self> {
        // Initialize Apollo Federation client for KB access
        let apollo_client = ApolloFederationClient::new(
            std::env::var("APOLLO_FEDERATION_URL")
                .unwrap_or_else(|_| "http://localhost:4000/graphql".to_string())
        )?;
        
        // Initialize performance metrics
        let metrics = Arc::new(Phase3Metrics::new());
        
        // Initialize sub-engines
        let candidate_generator = CandidateGenerator::new(
            apollo_client.clone(),
            metrics.clone()
        )?;
        
        let dose_engine = Phase3DoseEngine::new(
            unified_engine.clone(),
            metrics.clone()
        )?;
        
        let scoring_engine = ScoringRankingEngine::new(
            apollo_client.clone(),
            metrics.clone()
        )?;
        
        Ok(Self {
            candidate_generator,
            dose_engine,
            scoring_engine,
            apollo_client,
            metrics,
            unified_engine,
        })
    }
    
    /// Execute complete Phase 3 workflow
    pub async fn execute_phase3(&self, input: Phase3Input) -> Result<Phase3Output> {
        let phase3_start = std::time::Instant::now();
        let mut sub_phase_timing = HashMap::new();
        
        tracing::info!(
            "🎯 Starting Phase 3 execution for request_id: {}",
            input.request_id
        );
        
        // Phase 3a: Generate and vet candidates (≤25ms target)
        let candidates_start = std::time::Instant::now();
        let candidate_set = self.candidate_generator
            .generate_candidates(&input)
            .await?;
        let candidates_duration = candidates_start.elapsed();
        sub_phase_timing.insert("3a_candidates".to_string(), candidates_duration);
        
        tracing::info!(
            "✅ Phase 3a completed: {} candidates generated, {} safety vetted in {:?}",
            candidate_set.initial_count,
            candidate_set.vetted_count,
            candidates_duration
        );
        
        // Phase 3b: Calculate doses using Rust engine (≤25ms target)
        let dosing_start = std::time::Instant::now();
        let (dosed_candidates, dose_evidence) = self.dose_engine
            .calculate_doses(&candidate_set, &input)
            .await?;
        let dosing_duration = dosing_start.elapsed();
        sub_phase_timing.insert("3b_dosing".to_string(), dosing_duration);
        
        tracing::info!(
            "✅ Phase 3b completed: {} doses calculated in {:?}",
            dosed_candidates.len(),
            dosing_duration
        );
        
        // Phase 3c: Score and rank proposals (≤25ms target)
        let scoring_start = std::time::Instant::now();
        let ranked_proposals = self.scoring_engine
            .score_and_rank(&dosed_candidates, &input)
            .await?;
        let scoring_duration = scoring_start.elapsed();
        sub_phase_timing.insert("3c_scoring".to_string(), scoring_duration);
        
        tracing::info!(
            "✅ Phase 3c completed: {} proposals ranked in {:?}",
            ranked_proposals.len(),
            scoring_duration
        );
        
        // Build Phase 3 output
        let phase3_duration = phase3_start.elapsed();
        let output = Phase3Output {
            request_id: input.request_id.clone(),
            candidate_count: candidate_set.initial_count,
            safety_vetted: candidate_set.vetted_count,
            dose_calculated: dosed_candidates.len(),
            ranked_proposals,
            candidate_evidence: candidate_set.evidence,
            dose_evidence,
            scoring_evidence: vec![], // TODO: Implement scoring evidence
            phase3_duration,
            sub_phase_timing,
        };
        
        // Record metrics
        self.metrics.record_phase3_completion(&output);
        
        // Check SLA compliance
        if phase3_duration.as_millis() > 75 {
            tracing::warn!(
                "⚠️ Phase 3 SLA exceeded: {:?} > 75ms for request_id: {}",
                phase3_duration,
                input.request_id
            );
        } else {
            tracing::info!(
                "🎯 Phase 3 SLA met: {:?} ≤ 75ms for request_id: {}",
                phase3_duration,
                input.request_id
            );
        }
        
        Ok(output)
    }
    
    /// Get Phase 3 performance metrics
    pub fn get_metrics(&self) -> Arc<Phase3Metrics> {
        self.metrics.clone()
    }
    
    /// Health check for Phase 3 engine
    pub async fn health_check(&self) -> Result<Phase3HealthStatus> {
        let apollo_healthy = self.apollo_client.health_check().await?;
        let unified_engine_healthy = true; // TODO: Add unified engine health check
        
        Ok(Phase3HealthStatus {
            overall_healthy: apollo_healthy && unified_engine_healthy,
            apollo_federation: apollo_healthy,
            unified_engine: unified_engine_healthy,
            candidate_generator: true,
            dose_engine: true,
            scoring_engine: true,
            last_check: Utc::now(),
        })
    }
}

/// Phase 3 health status
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Phase3HealthStatus {
    pub overall_healthy: bool,
    pub apollo_federation: bool,
    pub unified_engine: bool,
    pub candidate_generator: bool,
    pub dose_engine: bool,
    pub scoring_engine: bool,
    pub last_check: DateTime<Utc>,
}

/// Phase 3 configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Phase3Config {
    pub apollo_federation_url: String,
    pub max_candidates: usize,
    pub target_sla_ms: u64,
    pub sub_phase_targets: SubPhaseTargets,
    pub scoring_weights: ScoringWeights,
    pub parallel_workers: usize,
}

impl Default for Phase3Config {
    fn default() -> Self {
        Self {
            apollo_federation_url: "http://localhost:4000/graphql".to_string(),
            max_candidates: 20,
            target_sla_ms: 75,
            sub_phase_targets: SubPhaseTargets::default(),
            scoring_weights: ScoringWeights::default(),
            parallel_workers: num_cpus::get().min(10),
        }
    }
}

/// Sub-phase timing targets
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubPhaseTargets {
    pub candidates_ms: u64,
    pub dosing_ms: u64,
    pub scoring_ms: u64,
}

impl Default for SubPhaseTargets {
    fn default() -> Self {
        Self {
            candidates_ms: 25,
            dosing_ms: 25,
            scoring_ms: 25,
        }
    }
}

/// Multi-factor scoring weights
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoringWeights {
    pub guideline_adherence: f64,
    pub patient_specific: f64,
    pub safety_profile: f64,
    pub formulary_preference: f64,
    pub cost_effectiveness: f64,
    pub adherence_likelihood: f64,
}

impl Default for ScoringWeights {
    fn default() -> Self {
        Self {
            guideline_adherence: 0.25,   // 25% - Clinical guidelines
            patient_specific: 0.20,      // 20% - Patient-specific factors
            safety_profile: 0.20,        // 20% - Safety considerations
            formulary_preference: 0.15,  // 15% - Formulary status
            cost_effectiveness: 0.10,    // 10% - Cost considerations
            adherence_likelihood: 0.10,  // 10% - Patient adherence
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::unified_clinical_engine::knowledge_base::KnowledgeBase;
    use std::sync::Arc;
    
    #[tokio::test]
    async fn test_phase3_engine_creation() {
        // Mock knowledge base for testing
        let kb_path = std::env::temp_dir().join("test_kb");
        std::fs::create_dir_all(&kb_path).unwrap();
        
        let knowledge_base = Arc::new(KnowledgeBase::new(kb_path.to_str().unwrap()).await.unwrap());
        let unified_engine = Arc::new(UnifiedClinicalEngine::new(knowledge_base).unwrap());
        
        let phase3_engine = ClinicalIntelligenceEngine::new(unified_engine);
        assert!(phase3_engine.is_ok());
        
        // Cleanup
        std::fs::remove_dir_all(&kb_path).unwrap();
    }
    
    #[tokio::test]
    async fn test_health_check() {
        let kb_path = std::env::temp_dir().join("test_kb_health");
        std::fs::create_dir_all(&kb_path).unwrap();
        
        let knowledge_base = Arc::new(KnowledgeBase::new(kb_path.to_str().unwrap()).await.unwrap());
        let unified_engine = Arc::new(UnifiedClinicalEngine::new(knowledge_base).unwrap());
        let phase3_engine = ClinicalIntelligenceEngine::new(unified_engine).unwrap();
        
        // Health check might fail due to Apollo not being available in test
        let _health = phase3_engine.health_check().await;
        // We don't assert success since Apollo might not be available
        
        // Cleanup
        std::fs::remove_dir_all(&kb_path).unwrap();
    }
}