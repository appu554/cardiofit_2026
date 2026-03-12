//! Medication-related data models

use serde::{Deserialize, Serialize};
use uuid::Uuid;
use chrono::{DateTime, Utc};

/// Represents a medication request from a clinical system
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationRequest {
    pub request_id: String,
    pub patient_id: String,
    pub medication_code: String,
    pub medication_name: String,
    pub patient_conditions: Vec<String>,
    #[serde(default)]
    pub patient_demographics: Option<PatientDemographics>,
    #[serde(default)]
    pub clinical_context: Option<ClinicalContext>,
    #[serde(default = "Utc::now")]
    pub timestamp: DateTime<Utc>,
}

/// Flow2Request - Complete compatibility with Go engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Flow2Request {
    pub request_id: String,
    pub patient_id: String,
    pub action_type: String,                                    // "MEDICATION_ANALYSIS", "DOSE_OPTIMIZATION"
    pub medication_data: serde_json::Map<String, serde_json::Value>,  // Flexible medication info
    pub patient_data: serde_json::Map<String, serde_json::Value>,     // Flexible patient info
    pub clinical_context: serde_json::Map<String, serde_json::Value>, // Complete clinical data
    pub processing_hints: serde_json::Map<String, serde_json::Value>, // Processing instructions
    #[serde(default)]
    pub priority: Option<String>,
    #[serde(default)]
    pub enable_ml_inference: bool,
    #[serde(default)]
    pub timeout: Option<u64>,                                   // Duration in milliseconds
    pub timestamp: DateTime<Utc>,
}

/// Patient demographic information - 100% compatible with Go engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientDemographics {
    pub age_years: Option<f64>,
    pub weight_kg: Option<f64>,
    pub height_cm: Option<f64>,
    pub gender: Option<String>,
    pub bmi: Option<f64>,
    pub bsa_m2: Option<f64>,        // Body Surface Area - from Go
    pub race: Option<String>,       // From Go
    pub ethnicity: Option<String>,  // From Go
    pub egfr: Option<f64>,
    pub creatinine_clearance: Option<f64>,
}

impl PatientDemographics {
    /// Calculate BMI if height and weight are available
    pub fn calculate_bmi(&self) -> Option<f64> {
        match (self.weight_kg, self.height_cm) {
            (Some(weight), Some(height)) => {
                let height_m = height / 100.0;
                Some(weight / (height_m * height_m))
            }
            _ => None,
        }
    }

    /// Check if patient is elderly (>= 65 years)
    pub fn is_elderly(&self) -> bool {
        self.age_years.map_or(false, |age| age >= 65.0)
    }

    /// Check if patient has renal impairment (eGFR < 60)
    pub fn has_renal_impairment(&self) -> bool {
        self.egfr.map_or(false, |egfr| egfr < 60.0)
    }

    /// Check if patient is obese (BMI >= 30)
    pub fn is_obese(&self) -> bool {
        self.bmi.or_else(|| self.calculate_bmi())
            .map_or(false, |bmi| bmi >= 30.0)
    }
}

/// Clinical context information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalContext {
    pub active_medications: Vec<ActiveMedication>,
    pub allergies: Vec<Allergy>,
    pub lab_values: Vec<LabValue>,
    pub conditions: Vec<Condition>,
    pub dialysis_status: Option<String>,
}

/// Active medication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ActiveMedication {
    pub medication_code: String,
    pub medication_name: String,
    pub dose: Option<String>,
    pub frequency: Option<String>,
    pub route: Option<String>,
    pub start_date: Option<DateTime<Utc>>,
}

/// Allergy information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Allergy {
    pub allergen: String,
    pub reaction: Option<String>,
    pub severity: Option<String>,
}

/// Laboratory value
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabValue {
    pub code: String,
    pub name: String,
    pub value: f64,
    pub unit: String,
    pub reference_range: Option<String>,
    pub timestamp: DateTime<Utc>,
}

/// Medical condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Condition {
    pub code: String,
    pub name: String,
    pub status: String,
    pub onset_date: Option<DateTime<Utc>>,
}

/// Medication from knowledge base
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Medication {
    pub rxnorm_code: String,
    pub generic_name: String,
    pub brand_names: Vec<String>,
    pub therapeutic_class: String,
    pub mechanism: String,
    pub indications: Vec<String>,
    pub safety_profile: SafetyProfile,
}

/// Medication structure compatible with Go engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GoMedication {
    pub code: String,                                           // RxNorm code
    pub name: String,                                           // Medication name
    pub dose: f64,                                              // Dose amount
    pub unit: String,                                           // Dose unit (mg, mcg, etc.)
    pub frequency: String,                                      // Dosing frequency
    pub route: String,                                          // Administration route
    pub duration: String,                                       // Treatment duration
    pub indication: String,                                     // Clinical indication
    pub properties: serde_json::Map<String, serde_json::Value>, // Additional properties
}

/// ⭐ MISSING: RecipeExecutionRequest - What Go engine actually sends
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecipeExecutionRequest {
    pub request_id: String,
    pub recipe_id: String,        // ⭐ KEY: "vancomycin-dosing-v1.0"
    pub variant: String,          // ⭐ KEY: "standard_auc"
    pub patient_id: String,
    pub medication_code: String,  // ⭐ KEY: "11124"
    pub clinical_context: String, // ⭐ KEY: JSON string with all clinical data
    pub timeout_ms: i64,
    #[serde(default)]
    pub snapshot_id: Option<String>, // Optional snapshot ID for snapshot-based processing
}

/// Snapshot-based recipe execution request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotBasedRequest {
    pub request_id: String,
    pub recipe_id: String,
    pub variant: String,
    pub patient_id: String,
    pub medication_code: String,
    pub snapshot_id: String,      // Required snapshot ID
    pub timeout_ms: i64,
    #[serde(default)]
    pub integrity_verification_required: bool,
}

/// Snapshot validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SnapshotValidation {
    pub is_valid: bool,
    pub snapshot_id: String,
    pub validation_timestamp: DateTime<Utc>,
    pub checksum_verified: bool,
    pub signature_verified: bool,
    pub is_expired: bool,
    pub validation_errors: Vec<String>,
    pub data_quality_score: Option<f64>,
}

/// ⭐ MISSING: MedicationProposal - What Go engine expects back
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationProposal {
    pub medication_code: String,
    pub medication_name: String,

    // Dosing information
    pub calculated_dose: f64,
    pub dose_unit: String,
    pub frequency: String,
    pub duration: Option<String>,

    // Safety assessment
    pub safety_status: String,    // "SAFE", "WARNING", "UNSAFE"
    pub safety_alerts: Vec<String>,
    pub contraindications: Vec<String>,

    // Clinical decision support
    pub clinical_rationale: String,
    pub monitoring_plan: Vec<String>,

    // Alternatives if needed
    pub alternatives: Vec<MedicationAlternative>,

    // Execution metadata
    pub execution_time_ms: i64,
    pub recipe_version: String,
}

/// ⭐ MISSING: MedicationAlternative
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationAlternative {
    pub medication_code: String,
    pub medication_name: String,
    pub rationale: String,
    pub safety_profile: String,
}

/// Flow2Response - Complete response to Go engine
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Flow2Response {
    pub request_id: String,
    pub patient_id: String,
    pub overall_status: String,
    pub execution_summary: ExecutionSummary,
    pub recipe_results: Vec<RecipeResult>,
    pub clinical_decision_support: serde_json::Map<String, serde_json::Value>,
    pub safety_alerts: Vec<SafetyAlert>,
    pub recommendations: Vec<String>,
    pub analytics: serde_json::Map<String, serde_json::Value>,
    pub execution_time_ms: i64,
    pub engine_used: String,
    pub timestamp: DateTime<Utc>,
    pub processing_metadata: ProcessingMetadata,
}

/// Execution summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionSummary {
    pub total_recipes_executed: i32,
    pub successful_recipes: i32,
    pub failed_recipes: i32,
    pub warnings: usize,
    pub errors: usize,
    pub engine: String,
    pub cache_hit_rate: f64,
}

/// Recipe execution result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RecipeResult {
    pub recipe_id: String,
    pub recipe_name: String,
    pub overall_status: String,
    pub execution_time_ms: i64,
    pub validations: Vec<String>,
    pub clinical_decision_support: serde_json::Map<String, serde_json::Value>,
    pub recommendations: Vec<String>,
    pub warnings: Vec<String>,
    pub errors: Vec<String>,
    pub metadata: serde_json::Map<String, serde_json::Value>,
}

/// Safety alert
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyAlert {
    pub alert_id: String,
    pub severity: String,
    pub alert_type: String,
    pub message: String,
    pub description: String,
    pub action_required: bool,
}

/// Processing metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProcessingMetadata {
    pub fallback_used: bool,
    pub cache_used: bool,
    pub context_sources: Vec<String>,
    pub processing_stages: Vec<String>,
    #[serde(default)]
    pub snapshot_based: bool,
    #[serde(default)]
    pub snapshot_id: Option<String>,
    #[serde(default)]
    pub snapshot_validation: Option<SnapshotValidation>,
}

/// Medication Intelligence Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationIntelligenceRequest {
    pub request_id: String,
    pub patient_id: String,
    pub medications: Vec<GoMedication>,
    pub intelligence_type: String,
    pub analysis_depth: String,
    pub clinical_context: serde_json::Map<String, serde_json::Value>,
}

/// Medication Intelligence Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationIntelligenceResponse {
    pub request_id: String,
    pub intelligence_score: f64,
    pub medication_analysis: serde_json::Map<String, serde_json::Value>,
    pub interaction_analysis: serde_json::Map<String, serde_json::Value>,
    pub outcome_predictions: serde_json::Map<String, serde_json::Value>,
    pub alternative_recommendations: Vec<MedicationAlternative>,
    pub clinical_insights: Vec<String>,
    pub execution_time_ms: i64,
}

/// Dose Optimization Request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseOptimizationRequest {
    pub request_id: String,
    pub patient_id: String,
    pub medication_code: String,
    pub clinical_parameters: serde_json::Map<String, serde_json::Value>,
    pub optimization_type: String,
    pub clinical_context: serde_json::Map<String, serde_json::Value>,
    pub processing_hints: serde_json::Map<String, serde_json::Value>,
}

/// Dose Optimization Response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DoseOptimizationResponse {
    pub request_id: String,
    pub optimized_dose: f64,
    pub optimization_score: f64,
    pub confidence_interval: ConfidenceInterval,
    pub pharmacokinetic_predictions: serde_json::Map<String, serde_json::Value>,
    pub monitoring_recommendations: Vec<String>,
    pub clinical_rationale: String,
    pub execution_time_ms: i64,
}

/// Confidence interval
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConfidenceInterval {
    pub lower: f64,
    pub upper: f64,
    pub confidence: f64,
}

/// Safety profile for medications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyProfile {
    pub requires_monitoring: bool,
    pub black_box_warning: Option<String>,
    pub contraindications: Vec<String>,
    pub drug_interactions: Vec<String>,
}

impl MedicationRequest {
    /// Create a new medication request
    pub fn new(
        request_id: String,
        patient_id: String,
        medication_code: String,
        medication_name: String,
        patient_conditions: Vec<String>,
    ) -> Self {
        Self {
            request_id,
            patient_id,
            medication_code,
            medication_name,
            patient_conditions,
            patient_demographics: None,
            clinical_context: None,
            timestamp: Utc::now(),
        }
    }

    /// Add patient demographics to the request
    pub fn with_demographics(mut self, demographics: PatientDemographics) -> Self {
        self.patient_demographics = Some(demographics);
        self
    }

    /// Add clinical context to the request
    pub fn with_clinical_context(mut self, context: ClinicalContext) -> Self {
        self.clinical_context = Some(context);
        self
    }

    /// Validate the medication request
    pub fn validate(&self) -> Result<(), String> {
        if self.request_id.is_empty() {
            return Err("request_id cannot be empty".to_string());
        }
        if self.patient_id.is_empty() {
            return Err("patient_id cannot be empty".to_string());
        }
        if self.medication_code.is_empty() {
            return Err("medication_code cannot be empty".to_string());
        }
        if self.medication_name.is_empty() {
            return Err("medication_name cannot be empty".to_string());
        }
        Ok(())
    }

    /// Get patient age in years if available
    pub fn patient_age(&self) -> Option<f64> {
        self.patient_demographics.as_ref()?.age_years
    }

    /// Get patient weight in kg if available
    pub fn patient_weight(&self) -> Option<f64> {
        self.patient_demographics.as_ref()?.weight_kg
    }

    /// Check if patient has a specific condition
    pub fn has_condition(&self, condition: &str) -> bool {
        self.patient_conditions.iter()
            .any(|c| c.to_lowercase().contains(&condition.to_lowercase()))
    }

    /// Get active medications if available
    pub fn active_medications(&self) -> Vec<&ActiveMedication> {
        self.clinical_context.as_ref()
            .map(|ctx| ctx.active_medications.iter().collect())
            .unwrap_or_default()
    }
}
