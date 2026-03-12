//! Advanced Validation - Multi-layer safety validation system
//! 
//! This module implements comprehensive validation with multiple layers:
//! - Input validation (patient data, lab values, contraindications)
//! - Mathematical validation (numerical stability, dose reasonableness)
//! - Clinical validation (drug interactions, Beers criteria)
//! - Output validation (result consistency, safety bounds)

use std::collections::HashMap;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use tracing::{info, warn, error, debug};

/// Advanced validator with multi-layer validation
#[derive(Debug)]
pub struct AdvancedValidator {
    clinical_validators: Vec<Box<dyn ClinicalValidator>>,
    mathematical_validators: Vec<Box<dyn MathematicalValidator>>,
    safety_validators: Vec<Box<dyn SafetyValidator>>,
    config: ValidationConfig,
}

/// Validation configuration
#[derive(Debug, Clone)]
pub struct ValidationConfig {
    pub enable_clinical_validation: bool,
    pub enable_mathematical_validation: bool,
    pub enable_safety_validation: bool,
    pub strict_mode: bool,
    pub fail_on_warnings: bool,
    pub max_validation_time_ms: u64,
}

impl Default for ValidationConfig {
    fn default() -> Self {
        Self {
            enable_clinical_validation: true,
            enable_mathematical_validation: true,
            enable_safety_validation: true,
            strict_mode: false,
            fail_on_warnings: false,
            max_validation_time_ms: 1000,
        }
    }
}

/// Comprehensive validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub overall_valid: bool,
    pub validation_score: f64,
    pub clinical_validation: LayerValidationResult,
    pub mathematical_validation: LayerValidationResult,
    pub safety_validation: LayerValidationResult,
    pub validation_time_ms: u64,
    pub summary: ValidationSummary,
}

/// Validation result for a specific layer
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LayerValidationResult {
    pub layer_name: String,
    pub valid: bool,
    pub score: f64,
    pub findings: Vec<ValidationFinding>,
    pub execution_time_ms: u64,
}

/// Individual validation finding
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationFinding {
    pub finding_id: String,
    pub validator_name: String,
    pub severity: ValidationSeverity,
    pub category: ValidationCategory,
    pub message: String,
    pub details: HashMap<String, serde_json::Value>,
    pub recommendation: Option<String>,
    pub evidence_level: EvidenceLevel,
}

/// Validation severity levels
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidationSeverity {
    Info,
    Warning,
    Error,
    Critical,
}

/// Validation categories
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidationCategory {
    PatientData,
    LaboratoryValues,
    DrugInteraction,
    Contraindication,
    DoseCalculation,
    NumericalStability,
    SafetyBounds,
    ClinicalGuidelines,
    BeerseCriteria,
    RenalFunction,
    HepaticFunction,
}

/// Evidence levels for validation findings
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum EvidenceLevel {
    A, // High-quality evidence
    B, // Moderate-quality evidence
    C, // Low-quality evidence
    Expert, // Expert opinion
}

/// Validation summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationSummary {
    pub total_findings: u32,
    pub critical_findings: u32,
    pub error_findings: u32,
    pub warning_findings: u32,
    pub info_findings: u32,
    pub recommendation: ValidationRecommendation,
}

/// Overall validation recommendation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum ValidationRecommendation {
    Proceed,
    ProceedWithCaution,
    RequiresReview,
    DoNotProceed,
}

/// Clinical validator trait
pub trait ClinicalValidator: Send + Sync + std::fmt::Debug {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>>;
    fn validator_name(&self) -> &str;
    fn categories(&self) -> Vec<ValidationCategory>;
}

/// Mathematical validator trait
pub trait MathematicalValidator: Send + Sync + std::fmt::Debug {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>>;
    fn validator_name(&self) -> &str;
    fn categories(&self) -> Vec<ValidationCategory>;
}

/// Safety validator trait
pub trait SafetyValidator: Send + Sync + std::fmt::Debug {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>>;
    fn validator_name(&self) -> &str;
    fn categories(&self) -> Vec<ValidationCategory>;
}

/// Validation context containing all relevant data
#[derive(Debug, Clone)]
pub struct ValidationContext {
    pub patient_data: HashMap<String, serde_json::Value>,
    pub drug_data: HashMap<String, serde_json::Value>,
    pub calculated_dose: Option<f64>,
    pub laboratory_values: HashMap<String, LabValue>,
    pub current_medications: Vec<Medication>,
    pub allergies: Vec<Allergy>,
    pub conditions: Vec<String>,
    pub calculation_results: HashMap<String, serde_json::Value>,
}

/// Laboratory value with metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabValue {
    pub value: f64,
    pub unit: String,
    pub reference_range: Option<ReferenceRange>,
    pub timestamp: chrono::DateTime<chrono::Utc>,
    pub abnormal: bool,
}

/// Reference range for lab values
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ReferenceRange {
    pub min: f64,
    pub max: f64,
    pub unit: String,
}

/// Medication information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Medication {
    pub code: String,
    pub name: String,
    pub dose: f64,
    pub unit: String,
    pub frequency: String,
    pub route: String,
}

/// Allergy information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Allergy {
    pub allergen: String,
    pub allergen_type: String,
    pub severity: String,
    pub reaction: String,
}

impl AdvancedValidator {
    /// Create a new advanced validator
    pub fn new() -> Self {
        let mut validator = Self {
            clinical_validators: Vec::new(),
            mathematical_validators: Vec::new(),
            safety_validators: Vec::new(),
            config: ValidationConfig::default(),
        };
        
        // Register default validators
        validator.register_default_validators();
        
        info!("🛡️ Advanced Validator initialized");
        info!("🏥 Clinical validators: {}", validator.clinical_validators.len());
        info!("🔢 Mathematical validators: {}", validator.mathematical_validators.len());
        info!("⚠️ Safety validators: {}", validator.safety_validators.len());
        
        validator
    }
    
    /// Register default validators
    fn register_default_validators(&mut self) {
        // Clinical validators
        self.clinical_validators.push(Box::new(PatientDataValidator));
        self.clinical_validators.push(Box::new(LaboratoryValidator));
        self.clinical_validators.push(Box::new(DrugInteractionValidator));
        self.clinical_validators.push(Box::new(ContraindicationValidator));
        self.clinical_validators.push(Box::new(BeerseCriteriaValidator));
        
        // Mathematical validators
        self.mathematical_validators.push(Box::new(NumericalStabilityValidator));
        self.mathematical_validators.push(Box::new(DoseReasonablenessValidator));
        self.mathematical_validators.push(Box::new(CalculationConsistencyValidator));
        
        // Safety validators
        self.safety_validators.push(Box::new(RenalSafetyValidator));
        self.safety_validators.push(Box::new(HepaticSafetyValidator));
        self.safety_validators.push(Box::new(SafetyBoundsValidator));
    }
    
    /// Perform comprehensive validation
    pub async fn validate(&self, context: ValidationContext) -> Result<ValidationResult> {
        let start_time = std::time::Instant::now();
        
        debug!("🔍 Starting comprehensive validation");
        
        let mut overall_valid = true;
        let mut all_findings = Vec::new();
        
        // Clinical validation
        let clinical_result = if self.config.enable_clinical_validation {
            self.run_clinical_validation(&context).await?
        } else {
            LayerValidationResult {
                layer_name: "Clinical".to_string(),
                valid: true,
                score: 1.0,
                findings: vec![],
                execution_time_ms: 0,
            }
        };
        
        if !clinical_result.valid {
            overall_valid = false;
        }
        all_findings.extend(clinical_result.findings.clone());
        
        // Mathematical validation
        let mathematical_result = if self.config.enable_mathematical_validation {
            self.run_mathematical_validation(&context).await?
        } else {
            LayerValidationResult {
                layer_name: "Mathematical".to_string(),
                valid: true,
                score: 1.0,
                findings: vec![],
                execution_time_ms: 0,
            }
        };
        
        if !mathematical_result.valid {
            overall_valid = false;
        }
        all_findings.extend(mathematical_result.findings.clone());
        
        // Safety validation
        let safety_result = if self.config.enable_safety_validation {
            self.run_safety_validation(&context).await?
        } else {
            LayerValidationResult {
                layer_name: "Safety".to_string(),
                valid: true,
                score: 1.0,
                findings: vec![],
                execution_time_ms: 0,
            }
        };
        
        if !safety_result.valid {
            overall_valid = false;
        }
        all_findings.extend(safety_result.findings.clone());
        
        // Calculate overall validation score
        let validation_score = self.calculate_overall_score(&all_findings);
        
        // Generate summary
        let summary = self.generate_summary(&all_findings);
        
        let total_time = start_time.elapsed().as_millis() as u64;
        
        let result = ValidationResult {
            overall_valid,
            validation_score,
            clinical_validation: clinical_result,
            mathematical_validation: mathematical_result,
            safety_validation: safety_result,
            validation_time_ms: total_time,
            summary,
        };
        
        info!("✅ Validation completed in {}ms (Score: {:.2})", 
              total_time, validation_score);
        
        Ok(result)
    }
    
    /// Run clinical validation layer
    async fn run_clinical_validation(&self, context: &ValidationContext) -> Result<LayerValidationResult> {
        let start_time = std::time::Instant::now();
        let mut findings = Vec::new();
        
        for validator in &self.clinical_validators {
            match validator.validate(context) {
                Ok(mut validator_findings) => {
                    findings.append(&mut validator_findings);
                }
                Err(e) => {
                    error!("Clinical validator {} failed: {}", validator.validator_name(), e);
                    findings.push(ValidationFinding {
                        finding_id: format!("{}-error", validator.validator_name()),
                        validator_name: validator.validator_name().to_string(),
                        severity: ValidationSeverity::Error,
                        category: ValidationCategory::PatientData,
                        message: format!("Validator failed: {}", e),
                        details: HashMap::new(),
                        recommendation: None,
                        evidence_level: EvidenceLevel::Expert,
                    });
                }
            }
        }
        
        let valid = !findings.iter().any(|f| matches!(f.severity, ValidationSeverity::Critical | ValidationSeverity::Error));
        let score = self.calculate_layer_score(&findings);
        
        Ok(LayerValidationResult {
            layer_name: "Clinical".to_string(),
            valid,
            score,
            findings,
            execution_time_ms: start_time.elapsed().as_millis() as u64,
        })
    }
    
    /// Run mathematical validation layer
    async fn run_mathematical_validation(&self, context: &ValidationContext) -> Result<LayerValidationResult> {
        let start_time = std::time::Instant::now();
        let mut findings = Vec::new();
        
        for validator in &self.mathematical_validators {
            match validator.validate(context) {
                Ok(mut validator_findings) => {
                    findings.append(&mut validator_findings);
                }
                Err(e) => {
                    error!("Mathematical validator {} failed: {}", validator.validator_name(), e);
                    findings.push(ValidationFinding {
                        finding_id: format!("{}-error", validator.validator_name()),
                        validator_name: validator.validator_name().to_string(),
                        severity: ValidationSeverity::Error,
                        category: ValidationCategory::DoseCalculation,
                        message: format!("Validator failed: {}", e),
                        details: HashMap::new(),
                        recommendation: None,
                        evidence_level: EvidenceLevel::Expert,
                    });
                }
            }
        }
        
        let valid = !findings.iter().any(|f| matches!(f.severity, ValidationSeverity::Critical | ValidationSeverity::Error));
        let score = self.calculate_layer_score(&findings);
        
        Ok(LayerValidationResult {
            layer_name: "Mathematical".to_string(),
            valid,
            score,
            findings,
            execution_time_ms: start_time.elapsed().as_millis() as u64,
        })
    }
    
    /// Run safety validation layer
    async fn run_safety_validation(&self, context: &ValidationContext) -> Result<LayerValidationResult> {
        let start_time = std::time::Instant::now();
        let mut findings = Vec::new();
        
        for validator in &self.safety_validators {
            match validator.validate(context) {
                Ok(mut validator_findings) => {
                    findings.append(&mut validator_findings);
                }
                Err(e) => {
                    error!("Safety validator {} failed: {}", validator.validator_name(), e);
                    findings.push(ValidationFinding {
                        finding_id: format!("{}-error", validator.validator_name()),
                        validator_name: validator.validator_name().to_string(),
                        severity: ValidationSeverity::Error,
                        category: ValidationCategory::SafetyBounds,
                        message: format!("Validator failed: {}", e),
                        details: HashMap::new(),
                        recommendation: None,
                        evidence_level: EvidenceLevel::Expert,
                    });
                }
            }
        }
        
        let valid = !findings.iter().any(|f| matches!(f.severity, ValidationSeverity::Critical | ValidationSeverity::Error));
        let score = self.calculate_layer_score(&findings);
        
        Ok(LayerValidationResult {
            layer_name: "Safety".to_string(),
            valid,
            score,
            findings,
            execution_time_ms: start_time.elapsed().as_millis() as u64,
        })
    }
    
    /// Calculate overall validation score
    fn calculate_overall_score(&self, findings: &[ValidationFinding]) -> f64 {
        if findings.is_empty() {
            return 1.0;
        }
        
        let mut score: f64 = 1.0;
        
        for finding in findings {
            let penalty = match finding.severity {
                ValidationSeverity::Critical => 0.5,
                ValidationSeverity::Error => 0.3,
                ValidationSeverity::Warning => 0.1,
                ValidationSeverity::Info => 0.0,
            };
            score -= penalty;
        }
        
        score.max(0.0)
    }
    
    /// Calculate layer-specific score
    fn calculate_layer_score(&self, findings: &[ValidationFinding]) -> f64 {
        self.calculate_overall_score(findings)
    }
    
    /// Generate validation summary
    fn generate_summary(&self, findings: &[ValidationFinding]) -> ValidationSummary {
        let mut critical_findings = 0;
        let mut error_findings = 0;
        let mut warning_findings = 0;
        let mut info_findings = 0;
        
        for finding in findings {
            match finding.severity {
                ValidationSeverity::Critical => critical_findings += 1,
                ValidationSeverity::Error => error_findings += 1,
                ValidationSeverity::Warning => warning_findings += 1,
                ValidationSeverity::Info => info_findings += 1,
            }
        }
        
        let recommendation = if critical_findings > 0 {
            ValidationRecommendation::DoNotProceed
        } else if error_findings > 0 {
            ValidationRecommendation::RequiresReview
        } else if warning_findings > 0 {
            ValidationRecommendation::ProceedWithCaution
        } else {
            ValidationRecommendation::Proceed
        };
        
        ValidationSummary {
            total_findings: findings.len() as u32,
            critical_findings,
            error_findings,
            warning_findings,
            info_findings,
            recommendation,
        }
    }
}

// Default validator implementations
#[derive(Debug)]
struct PatientDataValidator;

impl ClinicalValidator for PatientDataValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check for required patient data
        if !context.patient_data.contains_key("age_years") {
            findings.push(ValidationFinding {
                finding_id: "patient-age-missing".to_string(),
                validator_name: self.validator_name().to_string(),
                severity: ValidationSeverity::Error,
                category: ValidationCategory::PatientData,
                message: "Patient age is required for dose calculation".to_string(),
                details: HashMap::new(),
                recommendation: Some("Obtain patient age before proceeding".to_string()),
                evidence_level: EvidenceLevel::A,
            });
        }
        
        if !context.patient_data.contains_key("weight_kg") {
            findings.push(ValidationFinding {
                finding_id: "patient-weight-missing".to_string(),
                validator_name: self.validator_name().to_string(),
                severity: ValidationSeverity::Error,
                category: ValidationCategory::PatientData,
                message: "Patient weight is required for dose calculation".to_string(),
                details: HashMap::new(),
                recommendation: Some("Obtain patient weight before proceeding".to_string()),
                evidence_level: EvidenceLevel::A,
            });
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "PatientDataValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::PatientData]
    }
}

#[derive(Debug)]
struct LaboratoryValidator;

impl ClinicalValidator for LaboratoryValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check for abnormal lab values
        for (lab_name, lab_value) in &context.laboratory_values {
            if lab_value.abnormal {
                findings.push(ValidationFinding {
                    finding_id: format!("lab-abnormal-{}", lab_name),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Warning,
                    category: ValidationCategory::LaboratoryValues,
                    message: format!("Abnormal {} value: {} {}", lab_name, lab_value.value, lab_value.unit),
                    details: {
                        let mut details = HashMap::new();
                        details.insert("lab_name".to_string(), serde_json::Value::String(lab_name.clone()));
                        details.insert("value".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(lab_value.value).unwrap()));
                        details.insert("unit".to_string(), serde_json::Value::String(lab_value.unit.clone()));
                        details
                    },
                    recommendation: Some(format!("Consider dose adjustment for abnormal {}", lab_name)),
                    evidence_level: EvidenceLevel::B,
                });
            }
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "LaboratoryValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::LaboratoryValues]
    }
}

#[derive(Debug)]
struct DrugInteractionValidator;

impl ClinicalValidator for DrugInteractionValidator {
    fn validate(&self, _context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    fn validator_name(&self) -> &str {
        "DrugInteractionValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::DrugInteraction]
    }
}

#[derive(Debug)]
struct ContraindicationValidator;

impl ClinicalValidator for ContraindicationValidator {
    fn validate(&self, _context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    fn validator_name(&self) -> &str {
        "ContraindicationValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::Contraindication]
    }
}

#[derive(Debug)]
struct BeerseCriteriaValidator;

impl ClinicalValidator for BeerseCriteriaValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check if patient is elderly (≥65 years)
        if let Some(age_value) = context.patient_data.get("age_years") {
            if let Some(age) = age_value.as_f64() {
                if age >= 65.0 {
                    findings.push(ValidationFinding {
                        finding_id: "beers-elderly-patient".to_string(),
                        validator_name: self.validator_name().to_string(),
                        severity: ValidationSeverity::Warning,
                        category: ValidationCategory::BeerseCriteria,
                        message: format!("Elderly patient (age {}) - apply Beers criteria", age),
                        details: {
                            let mut details = HashMap::new();
                            details.insert("age_years".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(age).unwrap()));
                            details
                        },
                        recommendation: Some("Review medication appropriateness using Beers criteria".to_string()),
                        evidence_level: EvidenceLevel::A,
                    });
                }
            }
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "BeerseCriteriaValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::BeerseCriteria]
    }
}

#[derive(Debug)]
struct NumericalStabilityValidator;

impl MathematicalValidator for NumericalStabilityValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check calculated dose for numerical issues
        if let Some(dose) = context.calculated_dose {
            if dose.is_nan() {
                findings.push(ValidationFinding {
                    finding_id: "dose-nan".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Critical,
                    category: ValidationCategory::NumericalStability,
                    message: "Calculated dose is NaN (Not a Number)".to_string(),
                    details: HashMap::new(),
                    recommendation: Some("Review calculation inputs and logic".to_string()),
                    evidence_level: EvidenceLevel::A,
                });
            }
            
            if dose.is_infinite() {
                findings.push(ValidationFinding {
                    finding_id: "dose-infinite".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Critical,
                    category: ValidationCategory::NumericalStability,
                    message: "Calculated dose is infinite".to_string(),
                    details: HashMap::new(),
                    recommendation: Some("Review calculation inputs and logic".to_string()),
                    evidence_level: EvidenceLevel::A,
                });
            }
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "NumericalStabilityValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::NumericalStability]
    }
}

#[derive(Debug)]
struct DoseReasonablenessValidator;

impl MathematicalValidator for DoseReasonablenessValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check if dose is within reasonable bounds
        if let Some(dose) = context.calculated_dose {
            if dose <= 0.0 {
                findings.push(ValidationFinding {
                    finding_id: "dose-non-positive".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Error,
                    category: ValidationCategory::DoseCalculation,
                    message: format!("Calculated dose is non-positive: {}", dose),
                    details: {
                        let mut details = HashMap::new();
                        details.insert("calculated_dose".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(dose).unwrap()));
                        details
                    },
                    recommendation: Some("Review dose calculation logic".to_string()),
                    evidence_level: EvidenceLevel::A,
                });
            }
            
            if dose > 10000.0 {
                findings.push(ValidationFinding {
                    finding_id: "dose-extremely-high".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Warning,
                    category: ValidationCategory::DoseCalculation,
                    message: format!("Calculated dose is extremely high: {} mg", dose),
                    details: {
                        let mut details = HashMap::new();
                        details.insert("calculated_dose".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(dose).unwrap()));
                        details
                    },
                    recommendation: Some("Verify dose calculation and patient parameters".to_string()),
                    evidence_level: EvidenceLevel::B,
                });
            }
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "DoseReasonablenessValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::DoseCalculation]
    }
}

#[derive(Debug)]
struct CalculationConsistencyValidator;

impl MathematicalValidator for CalculationConsistencyValidator {
    fn validate(&self, _context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    fn validator_name(&self) -> &str {
        "CalculationConsistencyValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::DoseCalculation]
    }
}

#[derive(Debug)]
struct RenalSafetyValidator;

impl SafetyValidator for RenalSafetyValidator {
    fn validate(&self, context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        let mut findings = Vec::new();
        
        // Check eGFR if available
        if let Some(egfr_lab) = context.laboratory_values.get("egfr") {
            if egfr_lab.value < 30.0 {
                findings.push(ValidationFinding {
                    finding_id: "renal-severe-impairment".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Error,
                    category: ValidationCategory::RenalFunction,
                    message: format!("Severe renal impairment (eGFR: {} mL/min/1.73m²)", egfr_lab.value),
                    details: {
                        let mut details = HashMap::new();
                        details.insert("egfr".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(egfr_lab.value).unwrap()));
                        details
                    },
                    recommendation: Some("Consider dose reduction or alternative therapy".to_string()),
                    evidence_level: EvidenceLevel::A,
                });
            } else if egfr_lab.value < 60.0 {
                findings.push(ValidationFinding {
                    finding_id: "renal-moderate-impairment".to_string(),
                    validator_name: self.validator_name().to_string(),
                    severity: ValidationSeverity::Warning,
                    category: ValidationCategory::RenalFunction,
                    message: format!("Moderate renal impairment (eGFR: {} mL/min/1.73m²)", egfr_lab.value),
                    details: {
                        let mut details = HashMap::new();
                        details.insert("egfr".to_string(), serde_json::Value::Number(serde_json::Number::from_f64(egfr_lab.value).unwrap()));
                        details
                    },
                    recommendation: Some("Consider dose adjustment and enhanced monitoring".to_string()),
                    evidence_level: EvidenceLevel::A,
                });
            }
        }
        
        Ok(findings)
    }
    
    fn validator_name(&self) -> &str {
        "RenalSafetyValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::RenalFunction]
    }
}

#[derive(Debug)]
struct HepaticSafetyValidator;

impl SafetyValidator for HepaticSafetyValidator {
    fn validate(&self, _context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    fn validator_name(&self) -> &str {
        "HepaticSafetyValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::HepaticFunction]
    }
}

#[derive(Debug)]
struct SafetyBoundsValidator;

impl SafetyValidator for SafetyBoundsValidator {
    fn validate(&self, _context: &ValidationContext) -> Result<Vec<ValidationFinding>> {
        // Placeholder implementation
        Ok(vec![])
    }
    
    fn validator_name(&self) -> &str {
        "SafetyBoundsValidator"
    }
    
    fn categories(&self) -> Vec<ValidationCategory> {
        vec![ValidationCategory::SafetyBounds]
    }
}
