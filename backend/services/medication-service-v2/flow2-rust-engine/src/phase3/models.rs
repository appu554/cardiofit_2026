//! Phase 3 Data Models
//! 
//! Exact compliance with Phase 3 Clinical Intelligence Engine specification.
//! These models match the Go Phase 1 output and provide input/output contracts
//! for the three-phase workflow.

use std::collections::HashMap;
use std::time::Duration;
use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use uuid::Uuid;

/// Phase 3 Input - receives Intent Manifest and Enriched Context from Phase 1 & 2
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Phase3Input {
    pub request_id: String,
    pub manifest: IntentManifest,
    pub enriched_context: EnrichedContext,
    pub evidence_envelope: EvidenceEnvelope,
}

/// Phase 3 Output - returns ranked medication proposals with evidence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Phase3Output {
    pub request_id: String,
    pub candidate_count: usize,
    pub safety_vetted: usize,
    pub dose_calculated: usize,
    
    // Ranked proposals (primary output)
    pub ranked_proposals: Vec<MedicationProposal>,
    
    // Evidence tracking for audit compliance
    pub candidate_evidence: Vec<CandidateEvidence>,
    pub dose_evidence: Vec<DoseCalculation>,
    pub scoring_evidence: Vec<ScoringEvidence>,
    
    // Performance metrics
    pub phase3_duration: Duration,
    pub sub_phase_timing: HashMap<String, Duration>,
}

/// Intent Manifest from Phase 1 (ORB + Recipe Resolution)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntentManifest {
    pub manifest_id: String,
    pub request_id: String,
    pub generated_at: DateTime<Utc>,
    
    // Classification results
    pub primary_intent: ClinicalIntent,
    pub secondary_intents: Vec<ClinicalIntent>,
    
    // Protocol selection
    pub protocol_id: String,
    pub protocol_version: String,
    pub evidence_grade: String,
    
    // Recipe references
    pub context_recipe_id: String,
    pub clinical_recipe_id: String,
    
    // Computed requirements
    pub required_fields: Vec<FieldRequirement>,
    pub optional_fields: Vec<FieldRequirement>,
    
    // Freshness requirements
    pub data_freshness: FreshnessRequirements,
    pub snapshot_ttl: i64,
    
    // Therapy options determined by ORB (key input for Phase 3)
    pub therapy_options: Vec<TherapyCandidate>,
    
    // Provenance
    pub orb_version: String,
    pub rules_applied: Vec<AppliedRule>,
}

/// Clinical Intent classification from Phase 1
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalIntent {
    pub category: String,        // TREATMENT, PROPHYLAXIS, SYMPTOM_CONTROL
    pub condition: String,       // Coded condition (SNOMED/ICD)
    pub severity: String,        // MILD, MODERATE, SEVERE, CRITICAL
    pub phenotype: String,       // Patient phenotype classification
    pub time_horizon: String,    // ACUTE, CHRONIC, MAINTENANCE
}

/// Therapy candidate from ORB evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TherapyCandidate {
    pub therapy_class: String,      // ACE_INHIBITOR, ARB, etc.
    pub preference_order: i32,
    pub rationale: String,
    pub guideline_source: String,
}

/// Enriched Context from Phase 2 (Context Assembly)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnrichedContext {
    // Patient demographics
    pub demographics: Demographics,
    
    // Clinical data
    pub lab_results: LabResults,
    pub vital_signs: VitalSigns,
    pub current_medications: CurrentMedications,
    pub allergies: Vec<AllergyInfo>,
    pub active_conditions: Vec<String>,
    
    // Clinical phenotyping
    pub phenotype: String,
    pub risk_factors: Vec<String>,
    
    // Preferences and constraints
    pub patient_preferences: Option<PatientPreferences>,
    pub clinical_constraints: Vec<ClinicalConstraint>,
}

/// Patient demographics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Demographics {
    pub age: f64,
    pub sex: String,
    pub weight: f64,
    pub height: f64,
    pub bmi: Option<f64>,
    pub pregnancy_status: String,
    pub region: String,
}

/// Laboratory results
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabResults {
    #[serde(rename = "eGFR")]
    pub egfr: Option<f64>,
    pub creatinine: Option<f64>,
    pub bilirubin: Option<f64>,
    pub albumin: Option<f64>,
    pub alt: Option<f64>,
    pub ast: Option<f64>,
    pub hemoglobin: Option<f64>,
    pub platelet_count: Option<f64>,
    pub inr: Option<f64>,
}

/// Current medications context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CurrentMedications {
    pub medications: Vec<CurrentMedication>,
    pub count: usize,
}

/// Evidence Envelope for KB version tracking and audit compliance
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvidenceEnvelope {
    pub envelope_id: String,
    pub created_at: DateTime<Utc>,
    
    // KB version tracking (critical for deterministic results)
    pub kb_versions: HashMap<String, String>,
    
    // Snapshot integrity
    pub snapshot_hash: String,
    pub signature: Option<String>,
    
    // Audit trail
    pub audit_id: String,
    pub processing_chain: Vec<String>,
}

/// Medication Proposal - final ranked output from Phase 3
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationProposal {
    pub proposal_id: String,
    pub rank: i32,
    pub score: f64,
    
    // Medication details
    pub medication: MedicationDetails,
    
    // Dosing recommendation
    pub dose_calculation: DoseResult,
    
    // Safety assessment
    pub safety_profile: SafetyAssessment,
    
    // Scoring breakdown
    pub score_breakdown: ScoreComponents,
    
    // Supporting evidence
    pub evidence: ProposalEvidence,
}

/// Medication details for proposals
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationDetails {
    pub id: String,
    pub rxnorm: String,
    pub name: String,
    pub generic_name: String,
    pub class: String,
    pub subclass: String,
    pub indication: String,
}

/// Dose calculation result from Rust engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseResult {
    pub calculated_dose: f64,
    pub unit: String,
    pub frequency: String,
    pub route: String,
    
    // Adjustments applied
    pub adjustments_applied: Vec<Adjustment>,
    
    // Safety limits
    pub min_safe_dose: f64,
    pub max_safe_dose: f64,
    
    // Warnings and evidence
    pub warnings: Vec<DoseWarning>,
    pub evidence: DoseEvidence,
    
    // Calculation metadata
    pub calculation_method: String,
    pub confidence: f64,
    pub calculation_time: Duration,
    pub rust_engine_version: String,
}

/// Safety assessment from Phase 3a vetting
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyAssessment {
    pub safety_score: f64,           // 0-100 scale
    pub contraindicated: bool,
    pub safety_checks: Vec<SafetyCheck>,
    pub dose_adjustment_factor: Option<f64>,
}

/// Multi-factor score breakdown
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoreComponents {
    pub total: f64,
    pub guideline_adherence: f64,
    pub patient_specific: f64,
    pub safety_profile: f64,
    pub formulary_preference: f64,
    pub cost_effectiveness: f64,
    pub adherence_likelihood: f64,
}

/// Candidate Set from Phase 3a
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CandidateSet {
    pub request_id: String,
    pub initial_count: usize,
    pub vetted_count: usize,
    pub safe_count: usize,
    pub candidates: Vec<VettedCandidate>,
    pub generation_duration: Duration,
    pub evidence: Vec<CandidateEvidence>,
}

/// Vetted candidate from safety screening
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VettedCandidate {
    pub medication: MedicationCandidate,
    pub safety_score: f64,
    pub contraindicated: bool,
    pub safety_checks: Vec<SafetyCheck>,
    pub dose_adjustment_factor: Option<f64>,
}

/// Medication candidate from KB query
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationCandidate {
    pub id: String,
    pub rxnorm: String,
    pub name: String,
    pub class: String,
    pub subclass: String,
    pub contraindications: Vec<Contraindication>,
    pub precautions: Vec<Precaution>,
    pub black_box_warning: bool,
}

/// Dosed candidate from Phase 3b
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DosedCandidate {
    pub vetted_candidate: VettedCandidate,
    pub dose_result: DoseResult,
}

// Supporting types

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FieldRequirement {
    pub field_name: String,
    pub field_type: String,
    pub required: bool,
    pub max_age_hours: i32,
    pub source: String,
    pub clinical_reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FreshnessRequirements {
    pub max_age: Duration,
    pub critical_fields: Vec<String>,
    pub preferred_sources: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppliedRule {
    pub rule_id: String,
    pub rule_name: String,
    pub confidence: f64,
    pub applied_at: DateTime<Utc>,
    pub evidence_level: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VitalSigns {
    pub systolic_bp: Option<f64>,
    pub diastolic_bp: Option<f64>,
    pub heart_rate: Option<f64>,
    pub temperature: Option<f64>,
    pub respiratory_rate: Option<f64>,
    pub oxygen_saturation: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CurrentMedication {
    pub medication_code: String,
    pub medication_name: String,
    pub dose: String,
    pub frequency: String,
    pub route: String,
    pub start_date: DateTime<Utc>,
    pub indication: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AllergyInfo {
    pub allergen: String,
    pub allergen_type: String,
    pub reaction: String,
    pub severity: String,
    pub verified_by: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientPreferences {
    pub route_preferences: Vec<String>,
    pub frequency_preference: String,
    pub cost_constraints: String,
    pub lifestyle: HashMap<String, serde_json::Value>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalConstraint {
    pub constraint_type: String,
    pub value: String,
    pub severity: String,
    pub source: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Contraindication {
    pub condition_code: String,
    pub severity: String,
    pub contraindication_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Precaution {
    pub condition: String,
    pub adjustment_needed: bool,
    pub monitoring_required: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyCheck {
    pub check_type: String,
    pub severity: String,
    pub finding: String,
    pub action_required: bool,
    pub evidence_level: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Adjustment {
    pub adjustment_type: String,
    pub factor: f64,
    pub reason: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseWarning {
    pub severity: String,
    pub message: String,
    pub recommendation: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseEvidence {
    pub calculation_source: String,
    pub adjustment_rationale: Vec<String>,
    pub safety_considerations: Vec<String>,
    pub literature_references: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProposalEvidence {
    pub guideline_source: GuidelineRecommendation,
    pub formulary_data: FormularyData,
    pub resistance_data: ResistanceData,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GuidelineRecommendation {
    pub recommendation_level: String,
    pub evidence_grade: String,
    pub first_line: bool,
    pub alternative_to: Vec<String>,
    pub phenotype_specific: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FormularyData {
    pub in_stock: bool,
    pub tier: String,
    pub copay: Option<f64>,
    pub prior_auth_required: bool,
    pub preferred_status: String,
    pub average_cost: Option<f64>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResistanceData {
    pub susceptibility_rate: f64,
    pub resistance_pattern: String,
    pub last_updated: DateTime<Utc>,
}

// Evidence tracking types

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CandidateEvidence {
    pub candidate_id: String,
    pub generation_method: String,
    pub kb_versions_used: HashMap<String, String>,
    pub safety_checks_performed: Vec<String>,
    pub filtering_criteria: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseCalculation {
    pub medication_id: String,
    pub method: String,
    pub adjustments: Vec<Adjustment>,
    pub calculation_time: Duration,
    pub confidence: f64,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoringEvidence {
    pub medication_id: String,
    pub scoring_method: String,
    pub factors_considered: Vec<String>,
    pub weight_distribution: ScoreComponents,
    pub kb_sources: Vec<String>,
}

// Utility functions for model creation

impl Phase3Input {
    /// Create a new Phase3Input with generated IDs
    pub fn new(
        manifest: IntentManifest,
        enriched_context: EnrichedContext,
        evidence_envelope: EvidenceEnvelope,
    ) -> Self {
        Self {
            request_id: manifest.request_id.clone(),
            manifest,
            enriched_context,
            evidence_envelope,
        }
    }
}

impl MedicationProposal {
    /// Create a new proposal with generated ID
    pub fn new(
        rank: i32,
        score: f64,
        medication: MedicationDetails,
        dose_calculation: DoseResult,
        safety_profile: SafetyAssessment,
        score_breakdown: ScoreComponents,
        evidence: ProposalEvidence,
    ) -> Self {
        Self {
            proposal_id: Uuid::new_v4().to_string(),
            rank,
            score,
            medication,
            dose_calculation,
            safety_profile,
            score_breakdown,
            evidence,
        }
    }
}

impl Default for ScoreComponents {
    fn default() -> Self {
        Self {
            total: 0.0,
            guideline_adherence: 0.0,
            patient_specific: 0.0,
            safety_profile: 0.0,
            formulary_preference: 0.0,
            cost_effectiveness: 0.0,
            adherence_likelihood: 0.0,
        }
    }
}