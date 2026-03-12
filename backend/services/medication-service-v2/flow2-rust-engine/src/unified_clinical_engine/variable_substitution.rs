//! Variable Substitution System for Mathematical Expressions
//! 
//! This module provides a comprehensive variable substitution system that can
//! extract patient data, clinical context, and medication information to create
//! variables for use in mathematical expressions.
//! 
//! Supported variable categories:
//! - Patient demographics (age, weight, height, BMI, BSA)
//! - Organ function (renal, hepatic, cardiac)
//! - Laboratory values (creatinine, eGFR, liver enzymes)
//! - Clinical status (pregnancy, comorbidities)
//! - Medication context (current medications, interactions)
//! - Temporal factors (time since diagnosis, treatment duration)

use std::collections::HashMap;
use anyhow::{Result, anyhow};
use serde::{Deserialize, Serialize};
use tracing::{debug, warn};
use chrono::{DateTime, Utc, Duration};

use crate::unified_clinical_engine::{PatientContext, ClinicalRequest};
use crate::unified_clinical_engine::{BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction};

/// Variable substitution engine
pub struct VariableSubstitution {
    /// Custom variable definitions
    custom_variables: HashMap<String, VariableDefinition>,
    /// Variable validation rules
    validation_rules: HashMap<String, VariableValidation>,
}

/// Definition of a custom variable
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VariableDefinition {
    pub name: String,
    pub description: String,
    pub data_type: VariableDataType,
    pub source: VariableSource,
    pub calculation: Option<String>, // Mathematical expression for calculated variables
    pub default_value: Option<f64>,
    pub units: Option<String>,
    pub clinical_significance: String,
}

/// Data type of a variable
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VariableDataType {
    Numeric,
    Boolean,
    Categorical,
    Temporal,
}

/// Source of variable data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum VariableSource {
    PatientDemographics,
    LaboratoryResults,
    VitalSigns,
    OrganFunction,
    ClinicalHistory,
    MedicationHistory,
    CalculatedValue,
    ExternalSystem,
}

/// Variable validation rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VariableValidation {
    pub min_value: Option<f64>,
    pub max_value: Option<f64>,
    pub required: bool,
    pub clinical_range: Option<(f64, f64)>,
    pub warning_thresholds: Option<(f64, f64)>,
}

/// Result of variable substitution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SubstitutionResult {
    pub variables: HashMap<String, f64>,
    pub missing_variables: Vec<String>,
    pub warnings: Vec<String>,
    pub calculated_variables: Vec<String>,
    pub data_sources: HashMap<String, VariableSource>,
}

impl VariableSubstitution {
    pub fn new() -> Self {
        let mut engine = Self {
            custom_variables: HashMap::new(),
            validation_rules: HashMap::new(),
        };
        
        // Register standard clinical variables
        engine.register_standard_variables();
        engine
    }
    
    /// Register standard clinical variables
    fn register_standard_variables(&mut self) {
        // Patient demographics
        self.register_variable(VariableDefinition {
            name: "age".to_string(),
            description: "Patient age in years".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::PatientDemographics,
            calculation: None,
            default_value: None,
            units: Some("years".to_string()),
            clinical_significance: "Age affects drug metabolism and dosing requirements".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "weight".to_string(),
            description: "Patient weight in kilograms".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::PatientDemographics,
            calculation: None,
            default_value: None,
            units: Some("kg".to_string()),
            clinical_significance: "Weight is primary factor for dose calculations".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "height".to_string(),
            description: "Patient height in centimeters".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::PatientDemographics,
            calculation: None,
            default_value: None,
            units: Some("cm".to_string()),
            clinical_significance: "Height used for BSA and BMI calculations".to_string(),
        });
        
        // Calculated demographics
        self.register_variable(VariableDefinition {
            name: "bmi".to_string(),
            description: "Body Mass Index".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::CalculatedValue,
            calculation: Some("weight / (height / 100)^2".to_string()),
            default_value: None,
            units: Some("kg/m²".to_string()),
            clinical_significance: "BMI affects drug distribution and dosing".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "bsa".to_string(),
            description: "Body Surface Area (Mosteller formula)".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::CalculatedValue,
            calculation: Some("sqrt((weight * height) / 3600)".to_string()),
            default_value: None,
            units: Some("m²".to_string()),
            clinical_significance: "BSA used for chemotherapy and pediatric dosing".to_string(),
        });
        
        // Organ function
        self.register_variable(VariableDefinition {
            name: "egfr".to_string(),
            description: "Estimated Glomerular Filtration Rate".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::LaboratoryResults,
            calculation: None,
            default_value: None,
            units: Some("mL/min/1.73m²".to_string()),
            clinical_significance: "eGFR determines renal dose adjustments".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "creatinine".to_string(),
            description: "Serum creatinine".to_string(),
            data_type: VariableDataType::Numeric,
            source: VariableSource::LaboratoryResults,
            calculation: None,
            default_value: None,
            units: Some("mg/dL".to_string()),
            clinical_significance: "Creatinine indicates kidney function".to_string(),
        });
        
        // Gender and status variables
        self.register_variable(VariableDefinition {
            name: "is_male".to_string(),
            description: "1 if male, 0 if female".to_string(),
            data_type: VariableDataType::Boolean,
            source: VariableSource::PatientDemographics,
            calculation: None,
            default_value: None,
            units: None,
            clinical_significance: "Gender affects drug metabolism and dosing".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "is_pregnant".to_string(),
            description: "1 if pregnant, 0 if not pregnant".to_string(),
            data_type: VariableDataType::Boolean,
            source: VariableSource::ClinicalHistory,
            calculation: None,
            default_value: None,
            units: None,
            clinical_significance: "Pregnancy affects drug safety and dosing".to_string(),
        });
        
        // Age categories
        self.register_variable(VariableDefinition {
            name: "is_pediatric".to_string(),
            description: "1 if age < 18, 0 otherwise".to_string(),
            data_type: VariableDataType::Boolean,
            source: VariableSource::CalculatedValue,
            calculation: Some("age < 18 ? 1 : 0".to_string()),
            default_value: None,
            units: None,
            clinical_significance: "Pediatric patients require special dosing considerations".to_string(),
        });
        
        self.register_variable(VariableDefinition {
            name: "is_elderly".to_string(),
            description: "1 if age >= 65, 0 otherwise".to_string(),
            data_type: VariableDataType::Boolean,
            source: VariableSource::CalculatedValue,
            calculation: Some("age >= 65 ? 1 : 0".to_string()),
            default_value: None,
            units: None,
            clinical_significance: "Elderly patients may need dose reductions".to_string(),
        });
        
        // Register validation rules
        self.register_validation_rules();
    }
    
    /// Register validation rules for variables
    fn register_validation_rules(&mut self) {
        self.validation_rules.insert("age".to_string(), VariableValidation {
            min_value: Some(0.0),
            max_value: Some(150.0),
            required: true,
            clinical_range: Some((0.0, 120.0)),
            warning_thresholds: Some((0.1, 110.0)),
        });
        
        self.validation_rules.insert("weight".to_string(), VariableValidation {
            min_value: Some(0.5),
            max_value: Some(500.0),
            required: true,
            clinical_range: Some((2.0, 200.0)),
            warning_thresholds: Some((3.0, 150.0)),
        });
        
        self.validation_rules.insert("height".to_string(), VariableValidation {
            min_value: Some(30.0),
            max_value: Some(250.0),
            required: true,
            clinical_range: Some((40.0, 220.0)),
            warning_thresholds: Some((45.0, 210.0)),
        });
        
        self.validation_rules.insert("egfr".to_string(), VariableValidation {
            min_value: Some(0.0),
            max_value: Some(200.0),
            required: false,
            clinical_range: Some((5.0, 150.0)),
            warning_thresholds: Some((15.0, 120.0)),
        });
        
        self.validation_rules.insert("bmi".to_string(), VariableValidation {
            min_value: Some(10.0),
            max_value: Some(80.0),
            required: false,
            clinical_range: Some((15.0, 50.0)),
            warning_thresholds: Some((16.0, 40.0)),
        });
    }
    
    /// Register a custom variable definition
    pub fn register_variable(&mut self, definition: VariableDefinition) {
        self.custom_variables.insert(definition.name.clone(), definition);
    }
    
    /// Create variable substitution from clinical request
    pub fn create_substitution(&self, request: &ClinicalRequest) -> Result<SubstitutionResult> {
        let mut variables = HashMap::new();
        let mut missing_variables = Vec::new();
        let mut warnings = Vec::new();
        let mut calculated_variables = Vec::new();
        let mut data_sources = HashMap::new();
        
        debug!("Creating variable substitution for patient");
        
        // Extract basic patient demographics
        self.extract_demographics(&request.patient_context, &mut variables, &mut data_sources, &mut warnings);
        
        // Extract organ function data
        self.extract_organ_function(&request.patient_context, &mut variables, &mut data_sources, &mut warnings);
        
        // Extract clinical status
        self.extract_clinical_status(&request.patient_context, &mut variables, &mut data_sources, &mut warnings);
        
        // Calculate derived variables
        self.calculate_derived_variables(&mut variables, &mut calculated_variables, &mut warnings)?;
        
        // Validate all variables
        self.validate_variables(&variables, &mut warnings, &mut missing_variables);
        
        // Check for missing required variables
        self.check_required_variables(&variables, &mut missing_variables);
        
        debug!("Variable substitution complete: {} variables, {} missing, {} warnings", 
               variables.len(), missing_variables.len(), warnings.len());
        
        Ok(SubstitutionResult {
            variables,
            missing_variables,
            warnings,
            calculated_variables,
            data_sources,
        })
    }
    
    /// Extract demographic variables
    fn extract_demographics(&self, patient: &PatientContext, variables: &mut HashMap<String, f64>, 
                           data_sources: &mut HashMap<String, VariableSource>, warnings: &mut Vec<String>) {
        // Basic demographics
        variables.insert("age".to_string(), patient.age_years);
        data_sources.insert("age".to_string(), VariableSource::PatientDemographics);
        
        variables.insert("weight".to_string(), patient.weight_kg);
        data_sources.insert("weight".to_string(), VariableSource::PatientDemographics);
        
        variables.insert("height".to_string(), patient.height_cm);
        data_sources.insert("height".to_string(), VariableSource::PatientDemographics);
        
        // Gender
        let is_male = match patient.sex {
            BiologicalSex::Male => 1.0,
            BiologicalSex::Female => 0.0,
            BiologicalSex::Other => 0.5, // Neutral value for other/unknown
        };
        variables.insert("is_male".to_string(), is_male);
        variables.insert("is_female".to_string(), 1.0 - is_male);
        data_sources.insert("is_male".to_string(), VariableSource::PatientDemographics);
        data_sources.insert("is_female".to_string(), VariableSource::PatientDemographics);
        
        // Pregnancy status
        let is_pregnant = match patient.pregnancy_status {
            PregnancyStatus::Pregnant { .. } => 1.0,
            _ => 0.0,
        };
        variables.insert("is_pregnant".to_string(), is_pregnant);
        data_sources.insert("is_pregnant".to_string(), VariableSource::ClinicalHistory);
        
        // Age categories
        variables.insert("is_pediatric".to_string(), if patient.age_years < 18.0 { 1.0 } else { 0.0 });
        variables.insert("is_elderly".to_string(), if patient.age_years >= 65.0 { 1.0 } else { 0.0 });
        variables.insert("is_very_elderly".to_string(), if patient.age_years >= 80.0 { 1.0 } else { 0.0 });
        data_sources.insert("is_pediatric".to_string(), VariableSource::CalculatedValue);
        data_sources.insert("is_elderly".to_string(), VariableSource::CalculatedValue);
        data_sources.insert("is_very_elderly".to_string(), VariableSource::CalculatedValue);
        
        // Weight categories
        variables.insert("is_underweight".to_string(), if patient.weight_kg < 50.0 { 1.0 } else { 0.0 });
        variables.insert("is_overweight".to_string(), if patient.weight_kg > 100.0 { 1.0 } else { 0.0 });
        data_sources.insert("is_underweight".to_string(), VariableSource::CalculatedValue);
        data_sources.insert("is_overweight".to_string(), VariableSource::CalculatedValue);
    }
    
    /// Extract organ function variables
    fn extract_organ_function(&self, patient: &PatientContext, variables: &mut HashMap<String, f64>,
                             data_sources: &mut HashMap<String, VariableSource>, warnings: &mut Vec<String>) {
        // Renal function
        if let Some(egfr) = patient.renal_function.egfr_ml_min_1_73m2 {
            variables.insert("egfr".to_string(), egfr);
            data_sources.insert("egfr".to_string(), VariableSource::LaboratoryResults);
            
            // Renal function categories
            variables.insert("normal_renal".to_string(), if egfr >= 90.0 { 1.0 } else { 0.0 });
            variables.insert("mild_renal_impairment".to_string(), if egfr >= 60.0 && egfr < 90.0 { 1.0 } else { 0.0 });
            variables.insert("moderate_renal_impairment".to_string(), if egfr >= 30.0 && egfr < 60.0 { 1.0 } else { 0.0 });
            variables.insert("severe_renal_impairment".to_string(), if egfr >= 15.0 && egfr < 30.0 { 1.0 } else { 0.0 });
            variables.insert("kidney_failure".to_string(), if egfr < 15.0 { 1.0 } else { 0.0 });
        } else {
            warnings.push("eGFR not available - renal dose adjustments may not be accurate".to_string());
        }
        
        if let Some(creatinine) = patient.renal_function.creatinine_mg_dl {
            variables.insert("creatinine".to_string(), creatinine);
            data_sources.insert("creatinine".to_string(), VariableSource::LaboratoryResults);
        }
        
        // Hepatic function
        if let Some(alt) = patient.hepatic_function.alt_u_l {
            variables.insert("alt".to_string(), alt);
            data_sources.insert("alt".to_string(), VariableSource::LaboratoryResults);
            
            // Hepatic impairment indicators
            variables.insert("elevated_alt".to_string(), if alt > 40.0 { 1.0 } else { 0.0 });
        }
        
        if let Some(ast) = patient.hepatic_function.ast_u_l {
            variables.insert("ast".to_string(), ast);
            data_sources.insert("ast".to_string(), VariableSource::LaboratoryResults);
            
            variables.insert("elevated_ast".to_string(), if ast > 40.0 { 1.0 } else { 0.0 });
        }
        
        if let Some(bilirubin) = patient.hepatic_function.bilirubin_mg_dl {
            variables.insert("bilirubin".to_string(), bilirubin);
            data_sources.insert("bilirubin".to_string(), VariableSource::LaboratoryResults);
            
            variables.insert("elevated_bilirubin".to_string(), if bilirubin > 1.2 { 1.0 } else { 0.0 });
        }
    }
    
    /// Extract clinical status variables
    fn extract_clinical_status(&self, patient: &PatientContext, variables: &mut HashMap<String, f64>,
                              data_sources: &mut HashMap<String, VariableSource>, _warnings: &mut Vec<String>) {
        // Additional clinical status variables can be added here
        // For now, we'll add some basic derived status indicators
        
        // Renal status summary
        if let Some(egfr) = patient.renal_function.egfr_ml_min_1_73m2 {
            variables.insert("renal_dose_adjustment_needed".to_string(), if egfr < 60.0 { 1.0 } else { 0.0 });
            data_sources.insert("renal_dose_adjustment_needed".to_string(), VariableSource::CalculatedValue);
        }
        
        // Hepatic status summary
        let alt_elevated = patient.hepatic_function.alt_u_l.map_or(false, |alt| alt > 40.0);
        let ast_elevated = patient.hepatic_function.ast_u_l.map_or(false, |ast| ast > 40.0);
        let bilirubin_elevated = patient.hepatic_function.bilirubin_mg_dl.map_or(false, |bil| bil > 1.2);
        
        let hepatic_impairment = if alt_elevated || ast_elevated || bilirubin_elevated { 1.0 } else { 0.0 };
        variables.insert("hepatic_impairment".to_string(), hepatic_impairment);
        data_sources.insert("hepatic_impairment".to_string(), VariableSource::CalculatedValue);
    }
    
    /// Calculate derived variables
    fn calculate_derived_variables(&self, variables: &mut HashMap<String, f64>, 
                                  calculated_variables: &mut Vec<String>, warnings: &mut Vec<String>) -> Result<()> {
        // Calculate BMI
        if let (Some(&weight), Some(&height)) = (variables.get("weight"), variables.get("height")) {
            let height_m = height / 100.0;
            let bmi = weight / (height_m * height_m);
            variables.insert("bmi".to_string(), bmi);
            calculated_variables.push("bmi".to_string());
            
            // BMI categories
            variables.insert("underweight".to_string(), if bmi < 18.5 { 1.0 } else { 0.0 });
            variables.insert("normal_weight".to_string(), if bmi >= 18.5 && bmi < 25.0 { 1.0 } else { 0.0 });
            variables.insert("overweight".to_string(), if bmi >= 25.0 && bmi < 30.0 { 1.0 } else { 0.0 });
            variables.insert("obese".to_string(), if bmi >= 30.0 { 1.0 } else { 0.0 });
        } else {
            warnings.push("Cannot calculate BMI - weight or height missing".to_string());
        }
        
        // Calculate BSA (Mosteller formula)
        if let (Some(&weight), Some(&height)) = (variables.get("weight"), variables.get("height")) {
            let bsa = ((weight * height) / 3600.0).sqrt();
            variables.insert("bsa".to_string(), bsa);
            calculated_variables.push("bsa".to_string());
        } else {
            warnings.push("Cannot calculate BSA - weight or height missing".to_string());
        }
        
        // Calculate ideal body weight (Devine formula)
        if let (Some(&height), Some(&is_male)) = (variables.get("height"), variables.get("is_male")) {
            let ibw = if is_male > 0.5 {
                50.0 + 2.3 * ((height - 152.4) / 2.54) // Male
            } else {
                45.5 + 2.3 * ((height - 152.4) / 2.54) // Female
            };
            variables.insert("ideal_body_weight".to_string(), ibw.max(0.0));
            calculated_variables.push("ideal_body_weight".to_string());
        }
        
        Ok(())
    }
    
    /// Validate variables against defined rules
    fn validate_variables(&self, variables: &HashMap<String, f64>, warnings: &mut Vec<String>, 
                         _missing_variables: &mut Vec<String>) {
        for (var_name, &value) in variables {
            if let Some(validation) = self.validation_rules.get(var_name) {
                // Check bounds
                if let Some(min_val) = validation.min_value {
                    if value < min_val {
                        warnings.push(format!("Variable '{}' value {} is below minimum {}", var_name, value, min_val));
                    }
                }
                
                if let Some(max_val) = validation.max_value {
                    if value > max_val {
                        warnings.push(format!("Variable '{}' value {} is above maximum {}", var_name, value, max_val));
                    }
                }
                
                // Check clinical range
                if let Some((min_clinical, max_clinical)) = validation.clinical_range {
                    if value < min_clinical || value > max_clinical {
                        warnings.push(format!("Variable '{}' value {} is outside normal clinical range [{}, {}]", 
                                             var_name, value, min_clinical, max_clinical));
                    }
                }
                
                // Check warning thresholds
                if let Some((warn_low, warn_high)) = validation.warning_thresholds {
                    if value < warn_low {
                        warnings.push(format!("Variable '{}' value {} is below warning threshold {}", 
                                             var_name, value, warn_low));
                    } else if value > warn_high {
                        warnings.push(format!("Variable '{}' value {} is above warning threshold {}", 
                                             var_name, value, warn_high));
                    }
                }
            }
        }
    }
    
    /// Check for missing required variables
    fn check_required_variables(&self, variables: &HashMap<String, f64>, missing_variables: &mut Vec<String>) {
        for (var_name, validation) in &self.validation_rules {
            if validation.required && !variables.contains_key(var_name) {
                missing_variables.push(var_name.clone());
            }
        }
    }
    
    /// Get variable definition
    pub fn get_variable_definition(&self, name: &str) -> Option<&VariableDefinition> {
        self.custom_variables.get(name)
    }
    
    /// List all available variables
    pub fn list_available_variables(&self) -> Vec<&VariableDefinition> {
        self.custom_variables.values().collect()
    }
}

impl Default for VariableSubstitution {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::unified_clinical_engine::{BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction};

    fn create_test_patient() -> PatientContext {
        PatientContext {
            age_years: 45.0,
            sex: BiologicalSex::Male,
            weight_kg: 70.0,
            height_cm: 175.0,
            pregnancy_status: PregnancyStatus::NotPregnant,
            renal_function: RenalFunction {
                egfr_ml_min: Some(90.0),
                egfr_ml_min_1_73m2: Some(90.0),
                creatinine_clearance: None,
                creatinine_mg_dl: Some(1.0),
                bun_mg_dl: Some(15.0),
                stage: None,
            },
            hepatic_function: HepaticFunction {
                child_pugh_class: None,
                alt_u_l: Some(25.0),
                ast_u_l: Some(30.0),
                bilirubin_mg_dl: Some(0.8),
                albumin_g_dl: Some(4.0),
            },
            active_medications: Vec::new(),
            allergies: Vec::new(),
            conditions: Vec::new(),
            lab_values: HashMap::new(),
        }
    }

    fn create_test_request() -> ClinicalRequest {
        ClinicalRequest {
            request_id: "test-123".to_string(),
            patient_context: create_test_patient(),
            drug_id: "metformin".to_string(),
            indication: "diabetes".to_string(),
            timestamp: Utc::now(),
        }
    }

    #[test]
    fn test_basic_variable_extraction() {
        let substitution = VariableSubstitution::new();
        let request = create_test_request();
        
        let result = substitution.create_substitution(&request).unwrap();
        
        assert_eq!(result.variables.get("age"), Some(&45.0));
        assert_eq!(result.variables.get("weight"), Some(&70.0));
        assert_eq!(result.variables.get("height"), Some(&175.0));
        assert_eq!(result.variables.get("is_male"), Some(&1.0));
        assert_eq!(result.variables.get("is_pregnant"), Some(&0.0));
    }

    #[test]
    fn test_calculated_variables() {
        let substitution = VariableSubstitution::new();
        let request = create_test_request();
        
        let result = substitution.create_substitution(&request).unwrap();
        
        // BMI should be calculated: 70 / (1.75^2) ≈ 22.86
        let bmi = result.variables.get("bmi").unwrap();
        assert!((bmi - 22.86).abs() < 0.1);
        
        // BSA should be calculated: sqrt((70 * 175) / 3600) ≈ 1.85
        let bsa = result.variables.get("bsa").unwrap();
        assert!((bsa - 1.85).abs() < 0.1);
        
        assert!(result.calculated_variables.contains(&"bmi".to_string()));
        assert!(result.calculated_variables.contains(&"bsa".to_string()));
    }

    #[test]
    fn test_organ_function_variables() {
        let substitution = VariableSubstitution::new();
        let request = create_test_request();
        
        let result = substitution.create_substitution(&request).unwrap();
        
        assert_eq!(result.variables.get("egfr"), Some(&90.0));
        assert_eq!(result.variables.get("creatinine"), Some(&1.0));
        assert_eq!(result.variables.get("normal_renal"), Some(&1.0));
        assert_eq!(result.variables.get("renal_dose_adjustment_needed"), Some(&0.0));
    }

    #[test]
    fn test_age_categories() {
        let substitution = VariableSubstitution::new();
        let request = create_test_request();
        
        let result = substitution.create_substitution(&request).unwrap();
        
        assert_eq!(result.variables.get("is_pediatric"), Some(&0.0)); // 45 years old
        assert_eq!(result.variables.get("is_elderly"), Some(&0.0)); // 45 years old
        assert_eq!(result.variables.get("is_very_elderly"), Some(&0.0)); // 45 years old
    }

    #[test]
    fn test_variable_validation() {
        let substitution = VariableSubstitution::new();
        let mut request = create_test_request();
        
        // Set an extreme age value
        request.patient_context.age_years = 200.0;
        
        let result = substitution.create_substitution(&request).unwrap();
        
        // Should have warnings about age being outside normal range
        assert!(!result.warnings.is_empty());
        assert!(result.warnings.iter().any(|w| w.contains("age")));
    }
}
