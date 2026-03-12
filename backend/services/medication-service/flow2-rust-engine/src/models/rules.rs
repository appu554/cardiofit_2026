//! Rule evaluation models for the ORB (Orchestrator Rule Base)

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Orchestrator Rule Base - contains all clinical decision rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ORBRuleSet {
    pub metadata: ORBMetadata,
    pub rules: Vec<ORBRule>,
}

/// Metadata for the ORB rule set
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ORBMetadata {
    pub version: String,
    pub last_updated: String,
    pub description: String,
    pub total_rules: usize,
}

/// Individual ORB rule for clinical decision making
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ORBRule {
    pub id: String,
    pub priority: i32,
    #[serde(default)]
    pub conditions: RuleConditions,
    pub action: RuleAction,
    #[serde(default)]
    pub metadata: Option<RuleMetadata>,
}

/// Rule conditions (production format)
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct RuleConditions {
    #[serde(default)]
    pub all_of: Vec<RuleCondition>,
    #[serde(default)]
    pub any_of: Vec<RuleCondition>,
}

/// Individual rule condition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleCondition {
    pub fact: String,
    pub operator: String,
    pub value: serde_json::Value,
}

/// Rule action (what to do when rule matches)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleAction {
    #[serde(default)]
    pub generate_manifest: GenerateManifestAction,
    #[serde(default)]
    pub workflow_interrupt: Option<WorkflowInterruptAction>,
    #[serde(default)]
    pub escalate_for_review: Option<EscalateAction>,
}

/// Generate manifest action
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct GenerateManifestAction {
    pub recipe_id: String,
    pub variant: String,
    pub data_manifest: DataManifest,
}

/// Data manifest specifying required clinical data
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct DataManifest {
    pub required: Vec<String>,
    #[serde(default)]
    pub optional: Vec<String>,
}

/// Workflow interrupt action
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WorkflowInterruptAction {
    pub severity: String,
    pub message: String,
    #[serde(rename = "type")]
    pub interrupt_type: String,
}

/// Escalate for review action
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EscalateAction {
    pub severity: String,
    pub message: String,
    pub reviewer_role: Option<String>,
}

/// Rule metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleMetadata {
    pub created_by: Option<String>,
    pub created_at: Option<String>,
    pub last_modified: Option<String>,
    pub tags: Vec<String>,
    pub clinical_domain: Option<String>,
}

impl ORBRule {
    /// Check if this rule has any conditions
    pub fn has_conditions(&self) -> bool {
        !self.conditions.all_of.is_empty() || !self.conditions.any_of.is_empty()
    }

    /// Get the clinical domain from metadata
    pub fn clinical_domain(&self) -> Option<&str> {
        self.metadata.as_ref()?.clinical_domain.as_deref()
    }

    /// Check if this is a safety rule (high priority)
    pub fn is_safety_rule(&self) -> bool {
        self.priority >= 1000
    }
}

impl RuleCondition {
    /// Evaluate this condition against a given value
    pub fn evaluate(&self, actual_value: &serde_json::Value) -> bool {
        match self.operator.as_str() {
            "equal" => actual_value == &self.value,
            "not_equal" => actual_value != &self.value,
            "greater_than" | "gt" => {
                if let (Some(actual), Some(expected)) = (actual_value.as_f64(), self.value.as_f64()) {
                    actual > expected
                } else {
                    false
                }
            }
            "less_than" | "lt" => {
                if let (Some(actual), Some(expected)) = (actual_value.as_f64(), self.value.as_f64()) {
                    actual < expected
                } else {
                    false
                }
            }
            "greater_than_or_equal" | "gte" => {
                if let (Some(actual), Some(expected)) = (actual_value.as_f64(), self.value.as_f64()) {
                    actual >= expected
                } else {
                    false
                }
            }
            "less_than_or_equal" | "lte" => {
                if let (Some(actual), Some(expected)) = (actual_value.as_f64(), self.value.as_f64()) {
                    actual <= expected
                } else {
                    false
                }
            }
            "contains" => {
                if let (Some(actual), Some(expected)) = (actual_value.as_str(), self.value.as_str()) {
                    actual.to_lowercase().contains(&expected.to_lowercase())
                } else {
                    false
                }
            }
            "in" => {
                if let Some(array) = self.value.as_array() {
                    array.contains(actual_value)
                } else {
                    false
                }
            }
            "exists" => !actual_value.is_null(),
            "not_exists" => actual_value.is_null(),
            _ => false,
        }
    }

    /// Create a new rule condition
    pub fn new(fact: String, operator: String, value: serde_json::Value) -> Self {
        Self { fact, operator, value }
    }
}

impl RuleConditions {
    /// Evaluate all conditions using AND logic for all_of and OR logic for any_of
    pub fn evaluate(&self, context: &HashMap<String, serde_json::Value>) -> bool {
        // If no conditions, rule matches (catch-all rule)
        if self.all_of.is_empty() && self.any_of.is_empty() {
            return true;
        }

        // Evaluate all_of conditions (AND logic)
        let all_of_result = if self.all_of.is_empty() {
            true
        } else {
            self.all_of.iter().all(|condition| {
                if let Some(actual_value) = context.get(&condition.fact) {
                    condition.evaluate(actual_value)
                } else {
                    // If fact is not in context, condition fails
                    false
                }
            })
        };

        // Evaluate any_of conditions (OR logic)
        let any_of_result = if self.any_of.is_empty() {
            true
        } else {
            self.any_of.iter().any(|condition| {
                if let Some(actual_value) = context.get(&condition.fact) {
                    condition.evaluate(actual_value)
                } else {
                    false
                }
            })
        };

        all_of_result && any_of_result
    }
}
