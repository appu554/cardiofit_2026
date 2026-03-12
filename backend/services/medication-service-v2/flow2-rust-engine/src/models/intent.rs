//! Intent Manifest models - the output of clinical decision making

use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};
use uuid::Uuid;

/// Intent Manifest - the complete output of clinical decision support
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntentManifest {
    pub request_id: String,
    pub patient_id: String,
    pub recipe_id: String,
    pub variant: String,
    pub data_requirements: Vec<String>,
    pub priority: String,
    pub clinical_rationale: String,
    pub estimated_execution_time_ms: u64,
    pub rule_id: String,
    pub rule_version: String,
    pub generated_at: DateTime<Utc>,
    pub medication_code: String,
    pub conditions: Vec<String>,
    pub metadata: IntentMetadata,
}

/// Metadata for the intent manifest
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntentMetadata {
    pub medication_name: String,
    pub rule_type: String,
    #[serde(default)]
    pub confidence_score: Option<f64>,
    #[serde(default)]
    pub alternative_recipes: Vec<String>,
    #[serde(default)]
    pub warnings: Vec<String>,
    #[serde(default)]
    pub safety_alerts: Vec<SafetyAlert>,
}

/// Safety alert information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyAlert {
    pub alert_type: String,
    pub severity: String,
    pub message: String,
    pub source_rule: String,
}

/// Builder for creating Intent Manifests
pub struct IntentManifestBuilder {
    request_id: Option<String>,
    patient_id: Option<String>,
    recipe_id: Option<String>,
    variant: Option<String>,
    data_requirements: Vec<String>,
    priority: String,
    clinical_rationale: Option<String>,
    estimated_execution_time_ms: u64,
    rule_id: Option<String>,
    rule_version: String,
    medication_code: Option<String>,
    conditions: Vec<String>,
    metadata: IntentMetadata,
}

impl IntentManifestBuilder {
    /// Create a new builder
    pub fn new() -> Self {
        Self {
            request_id: None,
            patient_id: None,
            recipe_id: None,
            variant: None,
            data_requirements: Vec::new(),
            priority: "medium".to_string(),
            clinical_rationale: None,
            estimated_execution_time_ms: 100,
            rule_id: None,
            rule_version: "1.0.0".to_string(),
            medication_code: None,
            conditions: Vec::new(),
            metadata: IntentMetadata {
                medication_name: String::new(),
                rule_type: "production".to_string(),
                confidence_score: None,
                alternative_recipes: Vec::new(),
                warnings: Vec::new(),
                safety_alerts: Vec::new(),
            },
        }
    }

    /// Set request information
    pub fn with_request_info(mut self, request_id: String, patient_id: String) -> Self {
        self.request_id = Some(request_id);
        self.patient_id = Some(patient_id);
        self
    }

    /// Set recipe information
    pub fn with_recipe(mut self, recipe_id: String) -> Self {
        self.recipe_id = Some(recipe_id);
        self
    }

    /// Set variant
    pub fn with_variant(mut self, variant: String) -> Self {
        self.variant = Some(variant);
        self
    }

    /// Set data requirements
    pub fn with_data_requirements(mut self, requirements: Vec<String>) -> Self {
        self.data_requirements = requirements;
        self
    }

    /// Set priority
    pub fn with_priority(mut self, priority: String) -> Self {
        self.priority = priority;
        self
    }

    /// Set clinical rationale
    pub fn with_rationale(mut self, rationale: String) -> Self {
        self.clinical_rationale = Some(rationale);
        self
    }

    /// Set rule information
    pub fn with_rule_info(mut self, rule_id: String, rule_version: String) -> Self {
        self.rule_id = Some(rule_id);
        self.rule_version = rule_version;
        self
    }

    /// Set medication information
    pub fn with_medication_info(mut self, medication_code: String, medication_name: String, conditions: Vec<String>) -> Self {
        self.medication_code = Some(medication_code);
        self.metadata.medication_name = medication_name;
        self.conditions = conditions;
        self
    }

    /// Set estimated execution time
    pub fn with_estimated_time(mut self, time_ms: u64) -> Self {
        self.estimated_execution_time_ms = time_ms;
        self
    }

    /// Add a safety alert
    pub fn with_safety_alert(mut self, alert: SafetyAlert) -> Self {
        self.metadata.safety_alerts.push(alert);
        self
    }

    /// Build the intent manifest
    pub fn build(self) -> Result<IntentManifest, String> {
        Ok(IntentManifest {
            request_id: self.request_id.ok_or("request_id is required")?,
            patient_id: self.patient_id.ok_or("patient_id is required")?,
            recipe_id: self.recipe_id.ok_or("recipe_id is required")?,
            variant: self.variant.unwrap_or_else(|| "standard".to_string()),
            data_requirements: self.data_requirements,
            priority: self.priority,
            clinical_rationale: self.clinical_rationale.unwrap_or_else(|| "Clinical rule matched".to_string()),
            estimated_execution_time_ms: self.estimated_execution_time_ms,
            rule_id: self.rule_id.ok_or("rule_id is required")?,
            rule_version: self.rule_version,
            generated_at: Utc::now(),
            medication_code: self.medication_code.ok_or("medication_code is required")?,
            conditions: self.conditions,
            metadata: self.metadata,
        })
    }
}

impl Default for IntentManifestBuilder {
    fn default() -> Self {
        Self::new()
    }
}

/// ⭐ Enhanced Intent Manifest with Clinical Intelligence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnhancedIntentManifest {
    // Core manifest fields
    pub request_id: String,
    pub patient_id: String,
    pub recipe_id: String,
    pub variant: String,
    pub data_requirements: Vec<String>,
    pub priority: String,
    pub clinical_rationale: String,
    pub estimated_execution_time_ms: u64,
    pub rule_id: String,
    pub rule_version: String,
    pub generated_at: DateTime<Utc>,
    pub medication_code: String,
    pub conditions: Vec<String>,

    // Enhanced fields with clinical intelligence
    pub risk_assessment: ClinicalRiskAssessment,
    pub priority_details: DynamicPriority,
    pub clinical_rationale_details: DetailedClinicalRationale,
    pub execution_estimate: ExecutionEstimate,
    pub alternative_recipes: Vec<AlternativeRecipe>,
    pub clinical_flags: Vec<ClinicalFlag>,
    pub monitoring_requirements: Vec<MonitoringRequirement>,
    pub safety_considerations: Vec<SafetyConsideration>,
    pub metadata: EnhancedManifestMetadata,
}

/// Clinical risk assessment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalRiskAssessment {
    pub overall_risk_level: String, // "LOW", "MEDIUM", "HIGH", "CRITICAL"
    pub risk_factors: Vec<RiskFactor>,
    pub risk_score: f64, // 0.0 to 1.0
    pub assessment_rationale: String,
    pub mitigation_strategies: Vec<String>,
}

/// Individual risk factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RiskFactor {
    pub factor_type: String, // "DEMOGRAPHIC", "CONDITION", "MEDICATION", "LAB_VALUE"
    pub description: String,
    pub severity: String, // "LOW", "MEDIUM", "HIGH"
    pub impact_score: f64, // 0.0 to 1.0
    pub evidence_level: String, // "A", "B", "C", "D"
}

/// Dynamic priority calculation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DynamicPriority {
    pub level: String, // "LOW", "MEDIUM", "HIGH", "CRITICAL", "EMERGENCY"
    pub base_priority: String, // From rule
    pub adjustments: Vec<PriorityAdjustment>,
    pub final_score: f64, // 0.0 to 1.0
    pub rationale: String,
}

/// Priority adjustment factor
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PriorityAdjustment {
    pub factor: String,
    pub adjustment: f64, // -0.5 to +0.5
    pub rationale: String,
}

/// Detailed clinical rationale
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DetailedClinicalRationale {
    pub summary: String,
    pub reasoning_steps: Vec<String>,
    pub clinical_factors: Vec<String>,
    pub evidence_level: String,
    pub confidence_score: f64, // 0.0 to 1.0
}

/// Execution estimate with complexity analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExecutionEstimate {
    pub estimated_time_ms: u64,
    pub complexity_score: f64, // 1.0 to 5.0
    pub resource_requirements: ResourceRequirements,
    pub parallel_execution_possible: bool,
    pub caching_opportunities: usize,
}

/// Resource requirements for execution
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ResourceRequirements {
    pub cpu_intensive: bool,
    pub memory_intensive: bool,
    pub io_intensive: bool,
    pub network_calls_required: usize,
}

/// Alternative recipe option
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AlternativeRecipe {
    pub recipe_id: String,
    pub variant: String,
    pub rationale: String,
    pub suitability_score: f64, // 0.0 to 1.0
    pub trade_offs: Vec<String>,
}

/// Clinical flag for important considerations
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClinicalFlag {
    pub flag_type: String, // "DEMOGRAPHIC", "CONDITION", "MEDICATION", "LAB_VALUE"
    pub severity: String, // "LOW", "MEDIUM", "HIGH", "CRITICAL"
    pub message: String,
    pub code: String, // Standardized code
}

/// Monitoring requirement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRequirement {
    pub parameter: String,
    pub frequency: String,
    pub target_range: Option<String>,
    pub alert_conditions: Vec<String>,
    pub rationale: String,
    pub priority: String, // "LOW", "MEDIUM", "HIGH"
}

/// Safety consideration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyConsideration {
    pub category: String, // "NEPHROTOXICITY", "HEPATOTOXICITY", etc.
    pub severity: String, // "LOW", "MEDIUM", "HIGH"
    pub description: String,
    pub mitigation_strategies: Vec<String>,
    pub monitoring_parameters: Vec<String>,
}

/// Enhanced manifest metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnhancedManifestMetadata {
    pub generator_version: String,
    pub clinical_intelligence_enabled: bool,
    pub risk_assessment_performed: bool,
    pub data_optimization_applied: bool,
    pub alternative_analysis_performed: bool,
    pub generation_time_ms: u64,
}
