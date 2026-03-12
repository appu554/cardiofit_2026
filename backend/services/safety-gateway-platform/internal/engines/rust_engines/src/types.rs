// FHIR-compatible data structures for clinical safety evaluation
//
// This module defines the core data structures used for clinical safety
// evaluation, designed to be compatible with FHIR R4 resources and
// optimized for performance in clinical decision support systems.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use chrono::{DateTime, Utc};
use uuid::Uuid;

/// Safety evaluation status
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[repr(C)]
pub enum SafetyStatus {
    Safe,
    Unsafe,
    Warning,
    ManualReview,
}

/// Safety evaluation request from Go layer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyRequest {
    pub patient_id: String,
    pub request_id: String,
    pub medication_ids: Vec<String>,
    pub condition_ids: Vec<String>,
    pub allergy_ids: Vec<String>,
    pub action_type: String,
    pub priority: String,
}

/// Safety evaluation result returned to Go layer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyResult {
    pub status: SafetyStatus,
    pub risk_score: f64,
    pub confidence: f64,
    pub violations: Vec<String>,
    pub warnings: Vec<String>,
    pub processing_time_ms: u64,
    pub metadata: HashMap<String, String>,
}

/// Drug interaction severity levels
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum InteractionSeverity {
    Minor,
    Moderate,
    Major,
    Contraindicated,
}

/// Drug interaction information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugInteraction {
    pub drug_a: String,
    pub drug_b: String,
    pub severity: InteractionSeverity,
    pub description: String,
    pub mechanism: String,
    pub management: String,
    pub risk_score: f64,
}

/// Contraindication severity levels
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum ContraindicationSeverity {
    Relative,
    Absolute,
    Critical,
}

/// Contraindication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Contraindication {
    pub medication_id: String,
    pub condition_id: Option<String>,
    pub allergy_id: Option<String>,
    pub severity: ContraindicationSeverity,
    pub description: String,
    pub risk_score: f64,
}

/// Dosing validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DosingIssue {
    pub medication_id: String,
    pub issue_type: String, // "overdose", "underdose", "frequency", "duration"
    pub description: String,
    pub recommended_dose: Option<String>,
    pub risk_score: f64,
}

/// Clinical context data (mirrors Go types)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalContext {
    pub demographics: Option<PatientDemographics>,
    pub active_medications: Vec<Medication>,
    pub allergies: Vec<Allergy>,
    pub conditions: Vec<Condition>,
    pub recent_vitals: Option<VitalSigns>,
    pub lab_results: Vec<LabResult>,
    pub context_version: String,
}

/// Patient demographics
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientDemographics {
    pub age_years: Option<u32>,
    pub weight_kg: Option<f64>,
    pub height_cm: Option<f64>,
    pub gender: Option<String>,
    pub pregnancy_status: Option<bool>,
}

/// Medication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Medication {
    pub id: String,
    pub name: String,
    pub dose: Option<String>,
    pub frequency: Option<String>,
    pub route: Option<String>,
    pub start_date: Option<DateTime<Utc>>,
    pub end_date: Option<DateTime<Utc>>,
    pub prescriber_id: Option<String>,
}

/// Allergy information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Allergy {
    pub id: String,
    pub allergen: String,
    pub severity: String, // "mild", "moderate", "severe"
    pub reaction: Vec<String>,
    pub verified_date: Option<DateTime<Utc>>,
}

/// Medical condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Condition {
    pub id: String,
    pub code: String, // ICD-10 or similar
    pub display: String,
    pub severity: Option<String>,
    pub onset_date: Option<DateTime<Utc>>,
    pub status: String, // "active", "resolved", "inactive"
}

/// Vital signs
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VitalSigns {
    pub systolic_bp: Option<u32>,
    pub diastolic_bp: Option<u32>,
    pub heart_rate: Option<u32>,
    pub respiratory_rate: Option<u32>,
    pub temperature_c: Option<f64>,
    pub oxygen_saturation: Option<u32>,
    pub recorded_at: DateTime<Utc>,
}

/// Laboratory result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabResult {
    pub test_name: String,
    pub value: f64,
    pub unit: String,
    pub reference_range: Option<String>,
    pub abnormal_flag: Option<String>,
    pub collected_at: DateTime<Utc>,
}

impl SafetyRequest {
    /// Generate a cache key for this request
    pub fn cache_key(&self) -> String {
        use sha2::{Sha256, Digest};
        
        let mut hasher = Sha256::new();
        hasher.update(&self.patient_id);
        hasher.update(&self.action_type);
        
        // Sort medication IDs for consistent hashing
        let mut meds = self.medication_ids.clone();
        meds.sort();
        for med in &meds {
            hasher.update(med);
        }
        
        // Sort condition IDs for consistent hashing
        let mut conditions = self.condition_ids.clone();
        conditions.sort();
        for condition in &conditions {
            hasher.update(condition);
        }
        
        // Sort allergy IDs for consistent hashing
        let mut allergies = self.allergy_ids.clone();
        allergies.sort();
        for allergy in &allergies {
            hasher.update(allergy);
        }
        
        format!("{:x}", hasher.finalize())
    }
}

impl SafetyResult {
    /// Create a new safe result
    pub fn safe(processing_time_ms: u64) -> Self {
        Self {
            status: SafetyStatus::Safe,
            risk_score: 0.0,
            confidence: 1.0,
            violations: Vec::new(),
            warnings: Vec::new(),
            processing_time_ms,
            metadata: HashMap::new(),
        }
    }
    
    /// Create a new unsafe result with violations
    pub fn unsafe_with_violations(violations: Vec<String>, risk_score: f64, processing_time_ms: u64) -> Self {
        Self {
            status: SafetyStatus::Unsafe,
            risk_score,
            confidence: 0.9, // High confidence in safety violations
            violations,
            warnings: Vec::new(),
            processing_time_ms,
            metadata: HashMap::new(),
        }
    }
    
    /// Create a new warning result
    pub fn warning_with_issues(warnings: Vec<String>, risk_score: f64, processing_time_ms: u64) -> Self {
        Self {
            status: SafetyStatus::Warning,
            risk_score,
            confidence: 0.8,
            violations: Vec::new(),
            warnings,
            processing_time_ms,
            metadata: HashMap::new(),
        }
    }
}

impl DrugInteraction {
    /// Calculate risk score based on severity
    pub fn calculate_risk_score(severity: &InteractionSeverity) -> f64 {
        match severity {
            InteractionSeverity::Minor => 0.2,
            InteractionSeverity::Moderate => 0.5,
            InteractionSeverity::Major => 0.8,
            InteractionSeverity::Contraindicated => 1.0,
        }
    }
}

impl Contraindication {
    /// Calculate risk score based on severity
    pub fn calculate_risk_score(severity: &ContraindicationSeverity) -> f64 {
        match severity {
            ContraindicationSeverity::Relative => 0.4,
            ContraindicationSeverity::Absolute => 0.8,
            ContraindicationSeverity::Critical => 1.0,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cache_key_generation() {
        let request1 = SafetyRequest {
            patient_id: "patient-123".to_string(),
            request_id: "req-001".to_string(),
            medication_ids: vec!["med1".to_string(), "med2".to_string()],
            condition_ids: vec!["cond1".to_string()],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };

        let request2 = SafetyRequest {
            patient_id: "patient-123".to_string(),
            request_id: "req-002".to_string(), // Different request ID
            medication_ids: vec!["med2".to_string(), "med1".to_string()], // Different order
            condition_ids: vec!["cond1".to_string()],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };

        // Cache keys should be the same despite different order and request ID
        assert_eq!(request1.cache_key(), request2.cache_key());
    }

    #[test]
    fn test_safety_result_creation() {
        let safe_result = SafetyResult::safe(25);
        assert_eq!(safe_result.status, SafetyStatus::Safe);
        assert_eq!(safe_result.risk_score, 0.0);
        assert_eq!(safe_result.processing_time_ms, 25);

        let unsafe_result = SafetyResult::unsafe_with_violations(
            vec!["Drug interaction: warfarin + aspirin".to_string()],
            0.9,
            30
        );
        assert_eq!(unsafe_result.status, SafetyStatus::Unsafe);
        assert_eq!(unsafe_result.risk_score, 0.9);
        assert_eq!(unsafe_result.violations.len(), 1);
    }

    #[test]
    fn test_risk_score_calculations() {
        assert_eq!(DrugInteraction::calculate_risk_score(&InteractionSeverity::Minor), 0.2);
        assert_eq!(DrugInteraction::calculate_risk_score(&InteractionSeverity::Contraindicated), 1.0);
        
        assert_eq!(Contraindication::calculate_risk_score(&ContraindicationSeverity::Relative), 0.4);
        assert_eq!(Contraindication::calculate_risk_score(&ContraindicationSeverity::Critical), 1.0);
    }
}