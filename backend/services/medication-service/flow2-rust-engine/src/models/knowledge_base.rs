//! Knowledge base models for all 7 knowledge bases

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Complete knowledge base containing all 7 knowledge bases
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeBase {
    pub medication_knowledge_core: MedicationKnowledgeCore,
    pub evidence_repository: EvidenceRepository,
    pub orb_rules: super::rules::ORBRuleSet,
    pub context_recipes: ContextServiceRecipeBook,
    pub clinical_recipes: ClinicalRecipeBook,
    pub formulary_database: FormularyDatabase,
    pub monitoring_database: MonitoringDatabase,
}

/// TIER 1: Medication Knowledge Core (MKC)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MedicationKnowledgeCore {
    pub metadata: KnowledgeMetadata,
    pub medications: HashMap<String, super::medication::Medication>,
}

/// TIER 4: Evidence Repository (ER)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvidenceRepository {
    pub metadata: KnowledgeMetadata,
    pub evidence_entries: HashMap<String, EvidenceEntry>,
}

/// Evidence entry from clinical guidelines
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvidenceEntry {
    pub id: String,
    pub source: String,
    pub title: Option<String>,
    pub publication_date: Option<String>,
    pub evidence_level: Option<String>,
    pub recommendations: Vec<String>,
}

/// TIER 2: Context Service Recipe Book (CSRB)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextServiceRecipeBook {
    pub metadata: KnowledgeMetadata,
    pub recipes: HashMap<String, ContextRecipe>,
}

/// Context recipe for data assembly
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextRecipe {
    pub recipe_id: String,
    pub version: String,
    pub description: String,
    pub base_requirements: Vec<ContextRequirement>,
}

/// Context requirement specification
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContextRequirement {
    pub field: String,
    pub source: String,
    pub endpoint: Option<String>,
    pub required: bool,
    pub max_age_hours: Option<i32>,
}

/// TIER 1: Clinical Recipe Book (CRB)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRecipeBook {
    pub metadata: KnowledgeMetadata,
    pub recipes: HashMap<String, ClinicalRecipe>,
}

/// Clinical recipe for medication calculations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRecipe {
    pub id: String,
    pub name: String,
    pub description: String,
    pub medication_code: String,
    pub indication: String,
    pub calculation_variants: HashMap<String, CalculationVariant>,
    #[serde(default)]
    pub safety_checks: Vec<SafetyCheck>,
}

/// Calculation variant within a clinical recipe
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CalculationVariant {
    pub logic_steps: Vec<LogicStep>,
    #[serde(default)]
    pub adjustment_table: Vec<AdjustmentTableEntry>,
}

/// Individual logic step in calculation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogicStep {
    pub name: String,
    #[serde(rename = "type")]
    pub step_type: Option<String>,
    pub output: String,
    #[serde(default)]
    pub operation: Vec<Operation>,
    pub max_value: Option<f64>,
    pub min_value: Option<f64>,
}

/// Mathematical operation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Operation {
    pub variable: String,
    pub operator: String,
    pub value: Option<f64>,
    pub value_from: Option<String>,
}

/// Adjustment table entry for dose modifications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AdjustmentTableEntry {
    pub condition: String,
    pub dose_multiplier: Option<f64>,
    pub interval_hours: Option<i32>,
    pub special_instructions: Option<String>,
}

/// Safety check within clinical recipe
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyCheck {
    pub name: String,
    pub conditions: Vec<super::rules::RuleCondition>,
    pub action: SafetyAction,
}

/// Safety action to take
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyAction {
    #[serde(rename = "type")]
    pub action_type: String,
    pub message: String,
    pub severity: Option<String>,
}

/// TIER 3: Formulary Database (FCD)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FormularyDatabase {
    pub metadata: KnowledgeMetadata,
    pub formularies: HashMap<String, FormularyEntry>,
}

/// Formulary entry for medication products
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FormularyEntry {
    pub medication_code: String,
    pub generic_name: String,
    pub brand_name: String,
    pub formulary_status: String,
    pub tier: i32,
    pub prior_authorization_required: Option<bool>,
    pub quantity_limits: Option<String>,
}

/// TIER 3: Monitoring Requirements Database (MRD)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringDatabase {
    pub metadata: KnowledgeMetadata,
    pub profiles: HashMap<String, MonitoringProfile>,
}

/// Monitoring profile for medications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringProfile {
    pub medication_code: String,
    pub profile_name: String,
    pub baseline_required: bool,
    pub monitoring_parameters: Vec<MonitoringParameter>,
}

/// Individual monitoring parameter
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringParameter {
    pub parameter: String,
    pub frequency: String,
    pub target_range: Option<String>,
    pub alert_conditions: Vec<String>,
}

/// Generic metadata for knowledge bases
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct KnowledgeMetadata {
    pub version: String,
    pub last_updated: String,
    pub source: String,
    pub description: String,
    pub total_entries: Option<usize>,
}

impl KnowledgeBase {
    /// Get total number of knowledge items across all bases
    pub fn total_items(&self) -> usize {
        self.medication_knowledge_core.medications.len()
            + self.evidence_repository.evidence_entries.len()
            + self.orb_rules.rules.len()
            + self.context_recipes.recipes.len()
            + self.clinical_recipes.recipes.len()
            + self.formulary_database.formularies.len()
            + self.monitoring_database.profiles.len()
    }

    /// Get knowledge base summary
    pub fn summary(&self) -> KnowledgeSummary {
        KnowledgeSummary {
            medications_count: self.medication_knowledge_core.medications.len(),
            evidence_entries_count: self.evidence_repository.evidence_entries.len(),
            orb_rules_count: self.orb_rules.rules.len(),
            context_recipes_count: self.context_recipes.recipes.len(),
            clinical_recipes_count: self.clinical_recipes.recipes.len(),
            formulary_entries_count: self.formulary_database.formularies.len(),
            monitoring_profiles_count: self.monitoring_database.profiles.len(),
            total_items: self.total_items(),
        }
    }
}

/// Summary of knowledge base contents
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnowledgeSummary {
    pub medications_count: usize,
    pub evidence_entries_count: usize,
    pub orb_rules_count: usize,
    pub context_recipes_count: usize,
    pub clinical_recipes_count: usize,
    pub formulary_entries_count: usize,
    pub monitoring_profiles_count: usize,
    pub total_items: usize,
}
