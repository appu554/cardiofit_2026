//! Protocol Evaluation Context and Decision Aggregation
//!
//! This module provides evaluation context management and decision aggregation
//! for protocol evaluation results.

use std::time::Instant;
use uuid::Uuid;
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use crate::protocol::{
    types::*,
    error::*,
};

/// Evaluation context that tracks the state of a protocol evaluation
#[derive(Debug, Clone)]
pub struct EvaluationContext {
    /// Unique evaluation ID
    pub evaluation_id: Uuid,
    
    /// Original evaluation request
    pub request: ProtocolEvaluationRequest,
    
    /// Evaluation start time
    pub start_time: Instant,
    
    /// Conditions that have been met during evaluation
    pub conditions_met: Vec<String>,
    
    /// Conditions that have failed during evaluation
    pub conditions_failed: Vec<String>,
    
    /// Warnings generated during evaluation
    pub warnings: Vec<String>,
    
    /// Informational messages generated during evaluation
    pub information: Vec<String>,
    
    /// Performance tracking
    pub rules_execution_time_ms: u64,
    pub constraints_execution_time_ms: u64,
    pub state_machine_time_ms: u64,
    pub snapshot_resolution_time_ms: u64,
}

impl EvaluationContext {
    /// Create new evaluation context
    pub fn new(
        evaluation_id: Uuid,
        request: ProtocolEvaluationRequest,
        start_time: Instant,
    ) -> Self {
        Self {
            evaluation_id,
            request,
            start_time,
            conditions_met: Vec::new(),
            conditions_failed: Vec::new(),
            warnings: Vec::new(),
            information: Vec::new(),
            rules_execution_time_ms: 0,
            constraints_execution_time_ms: 0,
            state_machine_time_ms: 0,
            snapshot_resolution_time_ms: 0,
        }
    }
    
    /// Add a condition that was met
    pub fn add_condition_met(&mut self, condition: String) {
        self.conditions_met.push(condition);
    }
    
    /// Add a condition that failed
    pub fn add_condition_failed(&mut self, condition: String) {
        self.conditions_failed.push(condition);
    }
    
    /// Add a warning message
    pub fn add_warning(&mut self, warning: String) {
        self.warnings.push(warning);
    }
    
    /// Add an informational message
    pub fn add_information(&mut self, info: String) {
        self.information.push(info);
    }
    
    /// Get total elapsed time
    pub fn elapsed_time_ms(&self) -> u64 {
        self.start_time.elapsed().as_millis() as u64
    }
}

/// Evaluation result aggregation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    pub decision: ProtocolDecision,
    pub confidence: f64,
    pub reasoning: Vec<String>,
    pub recommendations: Vec<ProtocolRecommendation>,
}

/// Decision aggregator for combining multiple evaluation results
pub struct DecisionAggregator {
    /// Configuration for decision aggregation
    config: DecisionAggregatorConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DecisionAggregatorConfig {
    /// Minimum confidence threshold for decisions
    pub min_confidence_threshold: f64,
    
    /// Weight given to rule results
    pub rule_weight: f64,
    
    /// Weight given to constraint results  
    pub constraint_weight: f64,
    
    /// Weight given to temporal constraint results
    pub temporal_weight: f64,
    
    /// Weight given to state machine results
    pub state_machine_weight: f64,
}

impl Default for DecisionAggregatorConfig {
    fn default() -> Self {
        Self {
            min_confidence_threshold: 0.7,
            rule_weight: 0.4,
            constraint_weight: 0.3,
            temporal_weight: 0.2,
            state_machine_weight: 0.1,
        }
    }
}

impl DecisionAggregator {
    /// Create new decision aggregator
    pub fn new() -> ProtocolResult<Self> {
        Ok(Self {
            config: DecisionAggregatorConfig::default(),
        })
    }
    
    /// Create decision aggregator with custom configuration
    pub fn with_config(config: DecisionAggregatorConfig) -> ProtocolResult<Self> {
        Ok(Self { config })
    }
    
    /// Aggregate decision from multiple evaluation results
    pub async fn aggregate_decision(
        &self,
        rule_results: &[RuleEvaluationResult],
        constraint_results: &[ConstraintEvaluationResult],
        temporal_results: &[AppliedTemporalConstraint],
        state_changes: &[StateChange],
        eval_context: &EvaluationContext,
    ) -> ProtocolResult<ProtocolDecision> {
        // Calculate weighted scores for each component
        let rule_score = self.calculate_rule_score(rule_results);
        let constraint_score = self.calculate_constraint_score(constraint_results);
        let temporal_score = self.calculate_temporal_score(temporal_results);
        let state_score = self.calculate_state_score(state_changes);
        
        // Calculate overall weighted score
        let total_score = (rule_score * self.config.rule_weight) +
                         (constraint_score * self.config.constraint_weight) +
                         (temporal_score * self.config.temporal_weight) +
                         (state_score * self.config.state_machine_weight);
        
        // Normalize confidence (0.0 to 1.0)
        let confidence = total_score.max(0.0).min(1.0);
        
        // Determine decision type based on results
        let decision_type = self.determine_decision_type(
            rule_results,
            constraint_results,
            temporal_results,
            confidence,
        );
        
        // Build reasoning explanation
        let reasoning = self.build_reasoning(
            rule_results,
            constraint_results,
            temporal_results,
            state_changes,
            &decision_type,
        );
        
        // Determine if approval is required
        let requires_approval = self.requires_approval(
            &decision_type,
            rule_results,
            constraint_results,
        );
        
        // Determine if override is available
        let override_available = self.override_available(
            &decision_type,
            rule_results,
            constraint_results,
        );
        
        Ok(ProtocolDecision {
            decision_type,
            confidence,
            reasoning,
            requires_approval,
            override_available,
        })
    }
    
    /// Calculate rule evaluation score
    fn calculate_rule_score(&self, rule_results: &[RuleEvaluationResult]) -> f64 {
        if rule_results.is_empty() {
            return 1.0; // No rules = neutral score
        }
        
        let mut total_score = 0.0;
        let mut total_weight = 0.0;
        
        for result in rule_results {
            let weight = 1.0; // Could be based on rule priority
            total_weight += weight;
            
            let score = match &result.result {
                RuleResult::Pass => 1.0,
                RuleResult::Fail => 0.0,
                RuleResult::Warning => 0.5,
                RuleResult::NotApplicable => 1.0, // Don't penalize
                RuleResult::Error(_) => 0.0,
            };
            
            total_score += score * weight;
        }
        
        if total_weight > 0.0 {
            total_score / total_weight
        } else {
            1.0
        }
    }
    
    /// Calculate constraint evaluation score
    fn calculate_constraint_score(&self, constraint_results: &[ConstraintEvaluationResult]) -> f64 {
        if constraint_results.is_empty() {
            return 1.0;
        }
        
        let mut hard_constraints_passed = 0;
        let mut hard_constraints_total = 0;
        let mut soft_constraints_passed = 0;
        let mut soft_constraints_total = 0;
        
        for result in constraint_results {
            match result.constraint_type {
                ConstraintType::Hard => {
                    hard_constraints_total += 1;
                    if result.satisfied {
                        hard_constraints_passed += 1;
                    }
                },
                ConstraintType::Soft => {
                    soft_constraints_total += 1;
                    if result.satisfied {
                        soft_constraints_passed += 1;
                    }
                },
                _ => {} // Handle other constraint types as needed
            }
        }
        
        // Hard constraints are critical - if any fail, score is very low
        let hard_score = if hard_constraints_total > 0 {
            hard_constraints_passed as f64 / hard_constraints_total as f64
        } else {
            1.0
        };
        
        // Soft constraints contribute but are not critical
        let soft_score = if soft_constraints_total > 0 {
            soft_constraints_passed as f64 / soft_constraints_total as f64
        } else {
            1.0
        };
        
        // Weight hard constraints more heavily
        (hard_score * 0.8) + (soft_score * 0.2)
    }
    
    /// Calculate temporal constraint score
    fn calculate_temporal_score(&self, temporal_results: &[AppliedTemporalConstraint]) -> f64 {
        if temporal_results.is_empty() {
            return 1.0;
        }
        
        let satisfied_count = temporal_results.iter()
            .filter(|tc| tc.satisfied)
            .count();
        
        satisfied_count as f64 / temporal_results.len() as f64
    }
    
    /// Calculate state machine score
    fn calculate_state_score(&self, _state_changes: &[StateChange]) -> f64 {
        // For now, state changes are considered neutral
        // In a more sophisticated implementation, this could consider
        // whether state changes are valid/expected
        1.0
    }
    
    /// Determine the overall decision type
    fn determine_decision_type(
        &self,
        rule_results: &[RuleEvaluationResult],
        constraint_results: &[ConstraintEvaluationResult],
        temporal_results: &[AppliedTemporalConstraint],
        confidence: f64,
    ) -> ProtocolDecisionType {
        // Check for any blocking conditions first
        
        // Hard constraint violations = Block
        for constraint in constraint_results {
            if matches!(constraint.constraint_type, ConstraintType::Hard) && !constraint.satisfied {
                return ProtocolDecisionType::Block;
            }
        }
        
        // Critical temporal constraint violations = Block
        for temporal in temporal_results {
            if !temporal.satisfied && temporal.constraint_type.contains("critical") {
                return ProtocolDecisionType::Block;
            }
        }
        
        // Rule failures that require blocking
        for rule in rule_results {
            if let RuleResult::Fail = rule.result {
                // Check if this rule blocks the action (would depend on rule configuration)
                // For now, assume critical rules block
                if rule.rule_id.contains("critical") || rule.rule_id.contains("block") {
                    return ProtocolDecisionType::Block;
                }
            }
        }
        
        // Check for approval requirements
        let needs_approval = rule_results.iter().any(|r| {
            matches!(r.result, RuleResult::Warning) || r.rule_id.contains("approval")
        }) || constraint_results.iter().any(|c| {
            matches!(c.constraint_type, ConstraintType::Soft) && !c.satisfied
        });
        
        if needs_approval {
            return ProtocolDecisionType::RequireApproval;
        }
        
        // Check confidence level
        if confidence < self.config.min_confidence_threshold {
            return ProtocolDecisionType::Indeterminate;
        }
        
        // Check for modifications needed
        let has_warnings = rule_results.iter().any(|r| matches!(r.result, RuleResult::Warning));
        if has_warnings {
            return ProtocolDecisionType::Modify;
        }
        
        // Default to allow if no blocking conditions
        ProtocolDecisionType::Allow
    }
    
    /// Build reasoning explanation
    fn build_reasoning(
        &self,
        rule_results: &[RuleEvaluationResult],
        constraint_results: &[ConstraintEvaluationResult],
        temporal_results: &[AppliedTemporalConstraint],
        _state_changes: &[StateChange],
        decision_type: &ProtocolDecisionType,
    ) -> String {
        let mut reasoning_parts = Vec::new();
        
        // Add decision type explanation
        reasoning_parts.push(match decision_type {
            ProtocolDecisionType::Allow => "Protocol evaluation passed all requirements".to_string(),
            ProtocolDecisionType::Block => "Protocol evaluation identified blocking conditions".to_string(),
            ProtocolDecisionType::Modify => "Protocol evaluation identified modifications needed".to_string(),
            ProtocolDecisionType::RequireApproval => "Protocol evaluation requires manual approval".to_string(),
            ProtocolDecisionType::Indeterminate => "Protocol evaluation could not determine outcome".to_string(),
        });
        
        // Add rule evaluation summary
        let passed_rules = rule_results.iter().filter(|r| matches!(r.result, RuleResult::Pass)).count();
        let failed_rules = rule_results.iter().filter(|r| matches!(r.result, RuleResult::Fail)).count();
        let warning_rules = rule_results.iter().filter(|r| matches!(r.result, RuleResult::Warning)).count();
        
        if !rule_results.is_empty() {
            reasoning_parts.push(format!(
                "Rules: {} passed, {} failed, {} warnings",
                passed_rules, failed_rules, warning_rules
            ));
        }
        
        // Add constraint summary
        let satisfied_constraints = constraint_results.iter().filter(|c| c.satisfied).count();
        let violated_constraints = constraint_results.len() - satisfied_constraints;
        
        if !constraint_results.is_empty() {
            reasoning_parts.push(format!(
                "Constraints: {} satisfied, {} violated",
                satisfied_constraints, violated_constraints
            ));
        }
        
        // Add temporal constraint summary
        let satisfied_temporal = temporal_results.iter().filter(|t| t.satisfied).count();
        let violated_temporal = temporal_results.len() - satisfied_temporal;
        
        if !temporal_results.is_empty() {
            reasoning_parts.push(format!(
                "Temporal: {} satisfied, {} violated",
                satisfied_temporal, violated_temporal
            ));
        }
        
        reasoning_parts.join(". ")
    }
    
    /// Check if approval is required
    fn requires_approval(
        &self,
        decision_type: &ProtocolDecisionType,
        rule_results: &[RuleEvaluationResult],
        constraint_results: &[ConstraintEvaluationResult],
    ) -> bool {
        // Always require approval for RequireApproval decision
        if matches!(decision_type, ProtocolDecisionType::RequireApproval) {
            return true;
        }
        
        // Check for specific rules that require approval
        for rule in rule_results {
            if rule.rule_id.contains("require_approval") || rule.rule_id.contains("approval_required") {
                return true;
            }
        }
        
        // Check for constraint violations that require approval
        for constraint in constraint_results {
            if !constraint.satisfied && constraint.constraint_id.contains("approval") {
                return true;
            }
        }
        
        false
    }
    
    /// Check if override is available
    fn override_available(
        &self,
        decision_type: &ProtocolDecisionType,
        _rule_results: &[RuleEvaluationResult],
        constraint_results: &[ConstraintEvaluationResult],
    ) -> bool {
        // Hard constraints typically don't allow override
        let has_hard_constraint_violation = constraint_results.iter().any(|c| {
            matches!(c.constraint_type, ConstraintType::Hard) && !c.satisfied
        });
        
        if has_hard_constraint_violation {
            return false;
        }
        
        // Most other cases allow override
        !matches!(decision_type, ProtocolDecisionType::Allow)
    }
}

/// Rule evaluator for individual protocol rules
pub struct RuleEvaluator {
    // Rule evaluation implementation would go here
}

/// Constraint evaluator for protocol constraints
pub struct ConstraintEvaluator {
    // Constraint evaluation implementation would go here
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::time::Instant;
    use chrono::Utc;

    #[tokio::test]
    async fn test_evaluation_context_creation() {
        let eval_id = Uuid::new_v4();
        let request = ProtocolEvaluationRequest {
            protocol_id: "test-protocol".to_string(),
            patient_id: "test-patient".to_string(),
            clinical_context: ProtocolContext::default(),
            snapshot_id: None,
            evaluation_timestamp: Utc::now(),
            metadata: None,
        };
        
        let context = EvaluationContext::new(eval_id, request.clone(), Instant::now());
        
        assert_eq!(context.evaluation_id, eval_id);
        assert_eq!(context.request.protocol_id, request.protocol_id);
        assert!(context.conditions_met.is_empty());
        assert!(context.conditions_failed.is_empty());
    }

    #[tokio::test]
    async fn test_decision_aggregator() {
        let aggregator = DecisionAggregator::new().unwrap();
        
        // Create test data
        let rule_results = vec![
            RuleEvaluationResult {
                rule_id: "test-rule".to_string(),
                rule_name: "Test Rule".to_string(),
                result: RuleResult::Pass,
                execution_time_ms: 10,
                details: None,
            }
        ];
        
        let constraint_results = vec![];
        let temporal_results = vec![];
        let state_changes = vec![];
        
        let eval_context = EvaluationContext::new(
            Uuid::new_v4(),
            ProtocolEvaluationRequest {
                protocol_id: "test".to_string(),
                patient_id: "test-patient".to_string(),
                clinical_context: ProtocolContext::default(),
                snapshot_id: None,
                evaluation_timestamp: Utc::now(),
                metadata: None,
            },
            Instant::now(),
        );
        
        let decision = aggregator.aggregate_decision(
            &rule_results,
            &constraint_results,
            &temporal_results,
            &state_changes,
            &eval_context,
        ).await;
        
        assert!(decision.is_ok());
        let decision = decision.unwrap();
        assert_eq!(decision.decision_type, ProtocolDecisionType::Allow);
        assert!(decision.confidence > 0.0);
    }
}