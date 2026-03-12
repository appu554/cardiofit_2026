// Clinical Rules Engine - Drug Interactions, Contraindications, and Dosing
//
// This module implements the clinical decision logic for safety evaluation.
// It provides high-performance rule evaluation for drug interactions,
// contraindications, dosing validation, and other clinical safety checks.

use std::collections::{HashMap, HashSet};
use crate::types::{SafetyRequest, DrugInteraction, Contraindication, DosingIssue, 
                   InteractionSeverity, ContraindicationSeverity};
use crate::cae::{CAEConfig, CAEError, CAEResult};

/// Clinical rules engine that evaluates safety rules
pub struct RuleEngine {
    drug_interactions_db: DrugInteractionDB,
    contraindications_db: ContraindicationDB,
    dosing_rules_db: DosingRuleDB,
    duplicate_therapy_db: DuplicateTherapyDB,
}

/// Drug interaction database
pub struct DrugInteractionDB {
    interactions: HashMap<String, Vec<DrugInteraction>>,
    interaction_pairs: HashSet<(String, String)>,
}

/// Contraindications database
pub struct ContraindicationDB {
    medication_conditions: HashMap<String, Vec<Contraindication>>,
    medication_allergies: HashMap<String, Vec<String>>,
}

/// Dosing rules database
pub struct DosingRuleDB {
    dosing_rules: HashMap<String, DosingRule>,
    age_based_rules: HashMap<String, AgeBasedRule>,
    weight_based_rules: HashMap<String, WeightBasedRule>,
}

/// Duplicate therapy detection database
pub struct DuplicateTherapyDB {
    therapeutic_classes: HashMap<String, String>, // medication -> class
    class_interactions: HashMap<String, Vec<String>>, // class -> conflicting classes
}

/// Result of interaction checking
#[derive(Debug, Clone)]
pub struct InteractionResult {
    pub violations: Vec<String>,
    pub warnings: Vec<String>,
    pub risk_score: f64,
    pub interactions: Vec<DrugInteraction>,
}

/// Result of contraindication checking
#[derive(Debug, Clone)]
pub struct ContraindicationResult {
    pub violations: Vec<String>,
    pub warnings: Vec<String>,
    pub risk_score: f64,
    pub contraindications: Vec<Contraindication>,
}

/// Result of dosing validation
#[derive(Debug, Clone)]
pub struct DosingResult {
    pub violations: Vec<String>,
    pub warnings: Vec<String>,
    pub risk_score: f64,
    pub dosing_issues: Vec<DosingIssue>,
}

/// Dosing rule for a medication
#[derive(Debug, Clone)]
struct DosingRule {
    medication_id: String,
    min_dose_mg: Option<f64>,
    max_dose_mg: Option<f64>,
    max_daily_dose_mg: Option<f64>,
    frequency_hours: Option<u32>,
}

/// Age-based dosing rule
#[derive(Debug, Clone)]
struct AgeBasedRule {
    medication_id: String,
    min_age_years: Option<u32>,
    max_age_years: Option<u32>,
    pediatric_dose_factor: Option<f64>,
    geriatric_dose_factor: Option<f64>,
}

/// Weight-based dosing rule
#[derive(Debug, Clone)]
struct WeightBasedRule {
    medication_id: String,
    dose_per_kg: Option<f64>,
    max_dose_per_kg: Option<f64>,
}

impl RuleEngine {
    /// Create a new rules engine with the given configuration
    pub fn new(config: &CAEConfig) -> CAEResult<Self> {
        let drug_interactions_db = DrugInteractionDB::load(&config.database.drug_interactions_path)?;
        let contraindications_db = ContraindicationDB::load(&config.database.contraindications_path)?;
        let dosing_rules_db = DosingRuleDB::load(&config.database.dosing_rules_path)?;
        let duplicate_therapy_db = DuplicateTherapyDB::load(&config.rules_path)?;
        
        Ok(Self {
            drug_interactions_db,
            contraindications_db,
            dosing_rules_db,
            duplicate_therapy_db,
        })
    }
    
    /// Check for drug interactions between medications
    pub fn check_drug_interactions(&self, medication_ids: &[String]) -> Result<InteractionResult, Box<dyn std::error::Error>> {
        let mut interactions = Vec::new();
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0;
        
        // Check all pairs of medications
        for (i, med1) in medication_ids.iter().enumerate() {
            for med2 in medication_ids.iter().skip(i + 1) {
                if let Some(interaction) = self.drug_interactions_db.get_interaction(med1, med2) {
                    let risk_score = DrugInteraction::calculate_risk_score(&interaction.severity);
                    max_risk_score = max_risk_score.max(risk_score);
                    
                    match interaction.severity {
                        InteractionSeverity::Contraindicated | InteractionSeverity::Major => {
                            violations.push(format!(
                                "Drug interaction: {} + {} - {} ({})",
                                interaction.drug_a,
                                interaction.drug_b,
                                interaction.description,
                                format!("{:?}", interaction.severity)
                            ));
                        },
                        InteractionSeverity::Moderate => {
                            warnings.push(format!(
                                "Moderate drug interaction: {} + {} - {} (Monitor: {})",
                                interaction.drug_a,
                                interaction.drug_b,
                                interaction.description,
                                interaction.management
                            ));
                        },
                        InteractionSeverity::Minor => {
                            // Minor interactions are typically just informational
                            // Don't add to warnings unless specifically requested
                        },
                    }
                    
                    interactions.push(interaction);
                }
            }
        }
        
        Ok(InteractionResult {
            violations,
            warnings,
            risk_score: max_risk_score,
            interactions,
        })
    }
    
    /// Check for contraindications based on conditions
    pub fn check_contraindications(&self, request: &SafetyRequest) -> Result<ContraindicationResult, Box<dyn std::error::Error>> {
        let mut contraindications = Vec::new();
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0;
        
        for medication_id in &request.medication_ids {
            // Check against medical conditions
            for condition_id in &request.condition_ids {
                if let Some(contraindication) = self.contraindications_db.get_contraindication(medication_id, condition_id) {
                    let risk_score = Contraindication::calculate_risk_score(&contraindication.severity);
                    max_risk_score = max_risk_score.max(risk_score);
                    
                    match contraindication.severity {
                        ContraindicationSeverity::Critical | ContraindicationSeverity::Absolute => {
                            violations.push(format!(
                                "Contraindication: {} with {} - {}",
                                medication_id,
                                condition_id,
                                contraindication.description
                            ));
                        },
                        ContraindicationSeverity::Relative => {
                            warnings.push(format!(
                                "Relative contraindication: {} with {} - {} (Use with caution)",
                                medication_id,
                                condition_id,
                                contraindication.description
                            ));
                        },
                    }
                    
                    contraindications.push(contraindication);
                }
            }
        }
        
        Ok(ContraindicationResult {
            violations,
            warnings,
            risk_score: max_risk_score,
            contraindications,
        })
    }
    
    /// Check for allergy contraindications
    pub fn check_allergy_contraindications(&self, request: &SafetyRequest) -> Result<ContraindicationResult, Box<dyn std::error::Error>> {
        let mut contraindications = Vec::new();
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0;
        
        for medication_id in &request.medication_ids {
            for allergy_id in &request.allergy_ids {
                if self.contraindications_db.has_allergy_contraindication(medication_id, allergy_id) {
                    // Allergy contraindications are always critical
                    let contraindication = Contraindication {
                        medication_id: medication_id.clone(),
                        condition_id: None,
                        allergy_id: Some(allergy_id.clone()),
                        severity: ContraindicationSeverity::Critical,
                        description: format!("Allergy contraindication: {} with known allergy to {}", medication_id, allergy_id),
                        risk_score: 1.0,
                    };
                    
                    violations.push(format!(
                        "ALLERGY ALERT: {} contraindicated due to {} allergy",
                        medication_id,
                        allergy_id
                    ));
                    
                    max_risk_score = 1.0; // Maximum risk for allergies
                    contraindications.push(contraindication);
                }
            }
        }
        
        Ok(ContraindicationResult {
            violations,
            warnings,
            risk_score: max_risk_score,
            contraindications,
        })
    }
    
    /// Check dosing appropriateness
    pub fn check_dosing(&self, request: &SafetyRequest) -> Result<DosingResult, Box<dyn std::error::Error>> {
        let mut dosing_issues = Vec::new();
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0;
        
        for medication_id in &request.medication_ids {
            // Check basic dosing rules
            if let Some(rule) = self.dosing_rules_db.get_rule(medication_id) {
                // For now, we'll create placeholder dosing validation
                // In a real implementation, this would check actual prescribed doses
                // against the dosing rules based on patient demographics
                
                // Example: Check if medication requires age-specific dosing
                if let Some(age_rule) = self.dosing_rules_db.get_age_rule(medication_id) {
                    // This would require patient age from clinical context
                    // For now, just check if age-sensitive medication
                    if age_rule.pediatric_dose_factor.is_some() || age_rule.geriatric_dose_factor.is_some() {
                        warnings.push(format!(
                            "Age-sensitive dosing: {} requires age-appropriate dosing considerations",
                            medication_id
                        ));
                        max_risk_score = max_risk_score.max(0.3);
                    }
                }
                
                // Example: Check if medication requires weight-based dosing
                if let Some(weight_rule) = self.dosing_rules_db.get_weight_rule(medication_id) {
                    if weight_rule.dose_per_kg.is_some() {
                        warnings.push(format!(
                            "Weight-based dosing: {} requires weight-based dose calculation",
                            medication_id
                        ));
                        max_risk_score = max_risk_score.max(0.2);
                    }
                }
            }
        }
        
        Ok(DosingResult {
            violations,
            warnings,
            risk_score: max_risk_score,
            dosing_issues,
        })
    }
    
    /// Check for duplicate therapy
    pub fn check_duplicate_therapy(&self, medication_ids: &[String]) -> Result<InteractionResult, Box<dyn std::error::Error>> {
        let mut violations = Vec::new();
        let mut warnings = Vec::new();
        let mut max_risk_score = 0.0;
        
        // Group medications by therapeutic class
        let mut class_meds: HashMap<String, Vec<String>> = HashMap::new();
        for med_id in medication_ids {
            if let Some(class) = self.duplicate_therapy_db.get_therapeutic_class(med_id) {
                class_meds.entry(class).or_insert_with(Vec::new).push(med_id.clone());
            }
        }
        
        // Check for multiple medications in the same class
        for (class, meds) in class_meds {
            if meds.len() > 1 {
                warnings.push(format!(
                    "Duplicate therapy: Multiple {} medications: {}",
                    class,
                    meds.join(", ")
                ));
                max_risk_score = max_risk_score.max(0.4);
            }
        }
        
        Ok(InteractionResult {
            violations,
            warnings,
            risk_score: max_risk_score,
            interactions: Vec::new(), // No specific interactions for duplicate therapy
        })
    }
}

// Database implementations

impl DrugInteractionDB {
    fn load(path: &str) -> CAEResult<Self> {
        // In a real implementation, this would load from a database file
        // For now, create a mock database with common interactions
        let mut interactions = HashMap::new();
        let mut interaction_pairs = HashSet::new();
        
        // Add common drug interactions
        let warfarin_aspirin = DrugInteraction {
            drug_a: "warfarin".to_string(),
            drug_b: "aspirin".to_string(),
            severity: InteractionSeverity::Major,
            description: "Increased bleeding risk".to_string(),
            mechanism: "Additive anticoagulant effect".to_string(),
            management: "Monitor INR closely, consider dose adjustment".to_string(),
            risk_score: 0.8,
        };
        
        interactions.entry("warfarin".to_string()).or_insert_with(Vec::new).push(warfarin_aspirin.clone());
        interactions.entry("aspirin".to_string()).or_insert_with(Vec::new).push(warfarin_aspirin);
        interaction_pairs.insert(("warfarin".to_string(), "aspirin".to_string()));
        interaction_pairs.insert(("aspirin".to_string(), "warfarin".to_string()));
        
        // Add more common interactions...
        let digoxin_furosemide = DrugInteraction {
            drug_a: "digoxin".to_string(),
            drug_b: "furosemide".to_string(),
            severity: InteractionSeverity::Moderate,
            description: "Increased digoxin toxicity risk".to_string(),
            mechanism: "Furosemide-induced hypokalemia enhances digoxin toxicity".to_string(),
            management: "Monitor potassium levels and digoxin concentration".to_string(),
            risk_score: 0.6,
        };
        
        interactions.entry("digoxin".to_string()).or_insert_with(Vec::new).push(digoxin_furosemide.clone());
        interactions.entry("furosemide".to_string()).or_insert_with(Vec::new).push(digoxin_furosemide);
        interaction_pairs.insert(("digoxin".to_string(), "furosemide".to_string()));
        interaction_pairs.insert(("furosemide".to_string(), "digoxin".to_string()));
        
        Ok(Self {
            interactions,
            interaction_pairs,
        })
    }
    
    fn get_interaction(&self, med1: &str, med2: &str) -> Option<DrugInteraction> {
        // Check if this pair has a known interaction
        if self.interaction_pairs.contains(&(med1.to_string(), med2.to_string())) {
            if let Some(interactions) = self.interactions.get(med1) {
                return interactions.iter()
                    .find(|interaction| interaction.drug_b == med2)
                    .cloned();
            }
        }
        None
    }
}

impl ContraindicationDB {
    fn load(path: &str) -> CAEResult<Self> {
        // Mock contraindications database
        let mut medication_conditions = HashMap::new();
        let mut medication_allergies = HashMap::new();
        
        // Add common contraindications
        let metformin_kidney = Contraindication {
            medication_id: "metformin".to_string(),
            condition_id: Some("chronic_kidney_disease".to_string()),
            allergy_id: None,
            severity: ContraindicationSeverity::Absolute,
            description: "Metformin contraindicated in severe kidney disease due to lactic acidosis risk".to_string(),
            risk_score: 0.9,
        };
        
        medication_conditions.entry("metformin".to_string()).or_insert_with(Vec::new).push(metformin_kidney);
        
        // Add allergy contraindications
        medication_allergies.insert("penicillin".to_string(), vec!["penicillin_allergy".to_string()]);
        medication_allergies.insert("aspirin".to_string(), vec!["aspirin_allergy".to_string(), "nsaid_allergy".to_string()]);
        
        Ok(Self {
            medication_conditions,
            medication_allergies,
        })
    }
    
    fn get_contraindication(&self, medication_id: &str, condition_id: &str) -> Option<Contraindication> {
        self.medication_conditions.get(medication_id)?
            .iter()
            .find(|c| c.condition_id.as_ref() == Some(&condition_id.to_string()))
            .cloned()
    }
    
    fn has_allergy_contraindication(&self, medication_id: &str, allergy_id: &str) -> bool {
        self.medication_allergies.get(medication_id)
            .map(|allergies| allergies.contains(&allergy_id.to_string()))
            .unwrap_or(false)
    }
}

impl DosingRuleDB {
    fn load(path: &str) -> CAEResult<Self> {
        // Mock dosing rules database
        let mut dosing_rules = HashMap::new();
        let mut age_based_rules = HashMap::new();
        let mut weight_based_rules = HashMap::new();
        
        // Add common dosing rules
        let warfarin_rule = DosingRule {
            medication_id: "warfarin".to_string(),
            min_dose_mg: Some(1.0),
            max_dose_mg: Some(10.0),
            max_daily_dose_mg: Some(15.0),
            frequency_hours: Some(24),
        };
        dosing_rules.insert("warfarin".to_string(), warfarin_rule);
        
        // Age-based rules
        let pediatric_aspirin = AgeBasedRule {
            medication_id: "aspirin".to_string(),
            min_age_years: Some(12), // Avoid in children due to Reye's syndrome
            max_age_years: None,
            pediatric_dose_factor: None,
            geriatric_dose_factor: Some(0.5), // Reduce dose in elderly
        };
        age_based_rules.insert("aspirin".to_string(), pediatric_aspirin);
        
        Ok(Self {
            dosing_rules,
            age_based_rules,
            weight_based_rules,
        })
    }
    
    fn get_rule(&self, medication_id: &str) -> Option<&DosingRule> {
        self.dosing_rules.get(medication_id)
    }
    
    fn get_age_rule(&self, medication_id: &str) -> Option<&AgeBasedRule> {
        self.age_based_rules.get(medication_id)
    }
    
    fn get_weight_rule(&self, medication_id: &str) -> Option<&WeightBasedRule> {
        self.weight_based_rules.get(medication_id)
    }
}

impl DuplicateTherapyDB {
    fn load(path: &str) -> CAEResult<Self> {
        // Mock therapeutic class database
        let mut therapeutic_classes = HashMap::new();
        let mut class_interactions = HashMap::new();
        
        // Common therapeutic classes
        therapeutic_classes.insert("aspirin".to_string(), "nsaid".to_string());
        therapeutic_classes.insert("ibuprofen".to_string(), "nsaid".to_string());
        therapeutic_classes.insert("naproxen".to_string(), "nsaid".to_string());
        
        therapeutic_classes.insert("lisinopril".to_string(), "ace_inhibitor".to_string());
        therapeutic_classes.insert("enalapril".to_string(), "ace_inhibitor".to_string());
        
        therapeutic_classes.insert("metoprolol".to_string(), "beta_blocker".to_string());
        therapeutic_classes.insert("propranolol".to_string(), "beta_blocker".to_string());
        
        Ok(Self {
            therapeutic_classes,
            class_interactions,
        })
    }
    
    fn get_therapeutic_class(&self, medication_id: &str) -> Option<String> {
        self.therapeutic_classes.get(medication_id).cloned()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cae::CAEConfig;
    
    #[test]
    fn test_rule_engine_creation() {
        let config = CAEConfig::test_config();
        let rule_engine = RuleEngine::new(&config);
        assert!(rule_engine.is_ok());
    }
    
    #[test]
    fn test_drug_interaction_detection() {
        let config = CAEConfig::test_config();
        let rule_engine = RuleEngine::new(&config).unwrap();
        
        let medications = vec!["warfarin".to_string(), "aspirin".to_string()];
        let result = rule_engine.check_drug_interactions(&medications).unwrap();
        
        assert!(!result.violations.is_empty());
        assert!(result.risk_score > 0.5);
        assert_eq!(result.interactions.len(), 1);
        assert_eq!(result.interactions[0].severity, InteractionSeverity::Major);
    }
    
    #[test]
    fn test_contraindication_detection() {
        let config = CAEConfig::test_config();
        let rule_engine = RuleEngine::new(&config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-123".to_string(),
            request_id: "req-001".to_string(),
            medication_ids: vec!["metformin".to_string()],
            condition_ids: vec!["chronic_kidney_disease".to_string()],
            allergy_ids: vec![],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        let result = rule_engine.check_contraindications(&request).unwrap();
        
        assert!(!result.violations.is_empty());
        assert!(result.risk_score > 0.8);
        assert_eq!(result.contraindications.len(), 1);
    }
    
    #[test]
    fn test_allergy_detection() {
        let config = CAEConfig::test_config();
        let rule_engine = RuleEngine::new(&config).unwrap();
        
        let request = SafetyRequest {
            patient_id: "patient-456".to_string(),
            request_id: "req-002".to_string(),
            medication_ids: vec!["penicillin".to_string()],
            condition_ids: vec![],
            allergy_ids: vec!["penicillin_allergy".to_string()],
            action_type: "medication_order".to_string(),
            priority: "normal".to_string(),
        };
        
        let result = rule_engine.check_allergy_contraindications(&request).unwrap();
        
        assert!(!result.violations.is_empty());
        assert_eq!(result.risk_score, 1.0); // Maximum risk for allergies
        assert!(result.violations[0].contains("ALLERGY ALERT"));
    }
    
    #[test]
    fn test_duplicate_therapy_detection() {
        let config = CAEConfig::test_config();
        let rule_engine = RuleEngine::new(&config).unwrap();
        
        let medications = vec!["aspirin".to_string(), "ibuprofen".to_string()]; // Both NSAIDs
        let result = rule_engine.check_duplicate_therapy(&medications).unwrap();
        
        assert!(!result.warnings.is_empty());
        assert!(result.risk_score > 0.0);
        assert!(result.warnings[0].contains("Duplicate therapy"));
    }
}