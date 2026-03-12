//! Protocol Rule Evaluation Engine
//!
//! This module implements high-performance rule evaluation for clinical protocols
//! with support for parallel evaluation, caching, and complex clinical expressions.

use std::collections::HashMap;
use std::sync::Arc;
use std::time::Instant;
use tokio::time::{timeout, Duration};
use evalexpr::{eval_with_context, HashMapContext, Value, ContextWithMutableVariables};
use lru::LruCache;
use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use chrono::{Datelike, Timelike};

use crate::protocol::{
    types::*,
    error::*,
    evaluation::EvaluationContext,
    snapshot::SnapshotContext,
};

/// Rule engine configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleEngineConfig {
    pub max_rule_execution_time_ms: u64,
    pub max_parallel_rules: usize,
    pub enable_rule_caching: bool,
    pub cache_size: usize,
}

/// High-performance rule evaluation engine
pub struct RuleEngine {
    config: RuleEngineConfig,
    
    /// Compiled rule cache for performance
    compiled_rule_cache: Arc<RwLock<LruCache<String, CompiledRule>>>,
    
    /// Expression evaluation cache
    expression_cache: Arc<RwLock<LruCache<String, Value>>>,
    
    /// Rule evaluation metrics
    metrics: Arc<RuleEngineMetrics>,
}

/// Compiled rule for efficient evaluation
#[derive(Debug, Clone)]
struct CompiledRule {
    rule_id: String,
    compiled_condition: String, // Pre-processed condition
    action: RuleAction,
    priority: u32,
    context_dependencies: Vec<String>, // Variables this rule depends on
}

/// Rule engine performance metrics
#[derive(Debug, Default)]
pub struct RuleEngineMetrics {
    pub rules_evaluated: std::sync::atomic::AtomicU64,
    pub rules_passed: std::sync::atomic::AtomicU64,
    pub rules_failed: std::sync::atomic::AtomicU64,
    pub rules_errored: std::sync::atomic::AtomicU64,
    pub cache_hits: std::sync::atomic::AtomicU64,
    pub cache_misses: std::sync::atomic::AtomicU64,
    pub average_rule_time_ms: std::sync::atomic::AtomicU64,
}

impl RuleEngine {
    /// Create a new rule evaluation engine
    pub fn new(config: &RuleEngineConfig) -> ProtocolResult<Self> {
        let compiled_rule_cache = if config.enable_rule_caching {
            Arc::new(RwLock::new(LruCache::new(config.cache_size.try_into()
                .map_err(|_| ProtocolEngineError::ConfigurationError {
                    message: "Invalid rule cache size".to_string()
                })?)))
        } else {
            Arc::new(RwLock::new(LruCache::new(1.try_into().unwrap()))) // Minimal cache
        };
        
        let expression_cache = if config.enable_rule_caching {
            Arc::new(RwLock::new(LruCache::new((config.cache_size * 2).try_into()
                .map_err(|_| ProtocolEngineError::ConfigurationError {
                    message: "Invalid expression cache size".to_string()
                })?)))
        } else {
            Arc::new(RwLock::new(LruCache::new(1.try_into().unwrap()))) // Minimal cache
        };
        
        Ok(Self {
            config: config.clone(),
            compiled_rule_cache,
            expression_cache,
            metrics: Arc::new(RuleEngineMetrics::default()),
        })
    }
    
    /// Evaluate all rules for a protocol
    pub async fn evaluate_rules(
        &self,
        rules: &[ProtocolRule],
        clinical_context: &ProtocolContext,
        snapshot_context: Option<&SnapshotContext>,
        eval_context: &mut EvaluationContext,
    ) -> ProtocolResult<Vec<RuleEvaluationResult>> {
        let start_time = Instant::now();
        
        // Build evaluation context for expressions
        let expression_context = self.build_expression_context(
            clinical_context,
            snapshot_context,
            eval_context,
        )?;
        
        // Sort rules by priority (higher priority first)
        let mut sorted_rules: Vec<_> = rules.iter().collect();
        sorted_rules.sort_by(|a, b| b.priority.cmp(&a.priority));
        
        // Evaluate rules with parallel processing
        let rule_results = if sorted_rules.len() <= self.config.max_parallel_rules {
            // Small number of rules - evaluate all in parallel
            self.evaluate_rules_parallel(&sorted_rules, &expression_context).await?
        } else {
            // Large number of rules - batch processing
            self.evaluate_rules_batched(&sorted_rules, &expression_context).await?
        };
        
        // Update metrics
        eval_context.rules_execution_time_ms = start_time.elapsed().as_millis() as u64;
        
        Ok(rule_results)
    }
    
    /// Evaluate rules in parallel (for small rule sets)
    async fn evaluate_rules_parallel(
        &self,
        rules: &[&ProtocolRule],
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<Vec<RuleEvaluationResult>> {
        use futures::future::join_all;
        
        let evaluation_futures: Vec<_> = rules
            .iter()
            .map(|rule| self.evaluate_single_rule(rule, expression_context))
            .collect();
        
        let results = join_all(evaluation_futures).await;
        
        // Collect results and handle errors
        let mut rule_results = Vec::new();
        let mut errors = Vec::new();
        
        for result in results {
            match result {
                Ok(rule_result) => rule_results.push(rule_result),
                Err(error) => errors.push(error),
            }
        }
        
        if !errors.is_empty() {
            return Err(ProtocolEngineError::MultipleErrors {
                count: errors.len(),
                errors,
            });
        }
        
        Ok(rule_results)
    }
    
    /// Evaluate rules in batches (for large rule sets)
    async fn evaluate_rules_batched(
        &self,
        rules: &[&ProtocolRule],
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<Vec<RuleEvaluationResult>> {
        let mut all_results = Vec::new();
        let batch_size = self.config.max_parallel_rules;
        
        for batch in rules.chunks(batch_size) {
            let batch_results = self.evaluate_rules_parallel(batch, expression_context).await?;
            all_results.extend(batch_results);
        }
        
        Ok(all_results)
    }
    
    /// Evaluate a single rule
    async fn evaluate_single_rule(
        &self,
        rule: &ProtocolRule,
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<RuleEvaluationResult> {
        let start_time = Instant::now();
        
        // Skip disabled rules
        if !rule.enabled {
            return Ok(RuleEvaluationResult {
                rule_id: rule.rule_id.clone(),
                rule_name: rule.name.clone(),
                result: RuleResult::NotApplicable,
                execution_time_ms: 0,
                details: Some("Rule disabled".to_string()),
            });
        }
        
        // Evaluate with timeout
        let result = timeout(
            Duration::from_millis(self.config.max_rule_execution_time_ms),
            self.evaluate_rule_condition(rule, expression_context)
        ).await;
        
        let execution_time_ms = start_time.elapsed().as_millis() as u64;
        
        // Update metrics
        self.metrics.rules_evaluated.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        
        match result {
            Ok(Ok(condition_result)) => {
                let rule_result = if condition_result {
                    self.metrics.rules_passed.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    RuleResult::Pass
                } else {
                    self.metrics.rules_failed.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    RuleResult::Fail
                };
                
                Ok(RuleEvaluationResult {
                    rule_id: rule.rule_id.clone(),
                    rule_name: rule.name.clone(),
                    result: rule_result,
                    execution_time_ms,
                    details: None,
                })
            },
            Ok(Err(error)) => {
                self.metrics.rules_errored.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Ok(RuleEvaluationResult {
                    rule_id: rule.rule_id.clone(),
                    rule_name: rule.name.clone(),
                    result: RuleResult::Error(error.to_string()),
                    execution_time_ms,
                    details: Some(format!("Rule evaluation error: {}", error)),
                })
            },
            Err(_timeout) => {
                self.metrics.rules_errored.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                Ok(RuleEvaluationResult {
                    rule_id: rule.rule_id.clone(),
                    rule_name: rule.name.clone(),
                    result: RuleResult::Error("Timeout".to_string()),
                    execution_time_ms,
                    details: Some(format!("Rule evaluation timeout after {}ms", 
                        self.config.max_rule_execution_time_ms)),
                })
            },
        }
    }
    
    /// Evaluate rule condition (using boxed futures to handle recursion)
    async fn evaluate_rule_condition(
        &self,
        rule: &ProtocolRule,
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<bool> {
        self.evaluate_rule_condition_internal(rule, expression_context).await
    }
    
    /// Internal rule condition evaluation with proper async recursion handling
    fn evaluate_rule_condition_internal<'a>(
        &'a self,
        rule: &'a ProtocolRule,
        expression_context: &'a ExpressionContext,
    ) -> std::pin::Pin<Box<dyn std::future::Future<Output = ProtocolResult<bool>> + Send + 'a>> {
        Box::pin(async move {
            match &rule.condition {
                RuleCondition::Expression(expr) => {
                    self.evaluate_expression(expr, expression_context).await
                },
                RuleCondition::And(conditions) => {
                    for condition in conditions {
                        let sub_rule = ProtocolRule {
                            rule_id: format!("{}_and", rule.rule_id),
                            name: "AND condition".to_string(),
                            description: "".to_string(),
                            condition: condition.clone(),
                            action: rule.action.clone(),
                            priority: rule.priority,
                            enabled: true,
                        };
                        if !self.evaluate_rule_condition_internal(&sub_rule, expression_context).await? {
                            return Ok(false);
                        }
                    }
                    Ok(true)
                },
                RuleCondition::Or(conditions) => {
                    for condition in conditions {
                        let sub_rule = ProtocolRule {
                            rule_id: format!("{}_or", rule.rule_id),
                            name: "OR condition".to_string(),
                            description: "".to_string(),
                            condition: condition.clone(),
                            action: rule.action.clone(),
                            priority: rule.priority,
                            enabled: true,
                        };
                        if self.evaluate_rule_condition_internal(&sub_rule, expression_context).await? {
                            return Ok(true);
                        }
                    }
                    Ok(false)
                },
                RuleCondition::Not(condition) => {
                    let sub_rule = ProtocolRule {
                        rule_id: format!("{}_not", rule.rule_id),
                        name: "NOT condition".to_string(),
                        description: "".to_string(),
                        condition: (**condition).clone(),
                        action: rule.action.clone(),
                        priority: rule.priority,
                        enabled: true,
                    };
                    let result = self.evaluate_rule_condition_internal(&sub_rule, expression_context).await?;
                    Ok(!result)
                },
            }
        })
    }
    
    /// Evaluate a clinical expression
    async fn evaluate_expression(
        &self,
        expression: &str,
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<bool> {
        // Check expression cache
        if self.config.enable_rule_caching {
            let cache_key = format!("{}:{}", expression, expression_context.get_hash());
            
            {
                let cache = self.expression_cache.read();
                if let Some(cached_value) = cache.peek(&cache_key) {
                    self.metrics.cache_hits.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                    return match cached_value {
                        Value::Boolean(b) => Ok(*b),
                        _ => Ok(false),
                    };
                }
            }
            self.metrics.cache_misses.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
        }
        
        // Evaluate expression
        let result = self.evaluate_expression_internal(expression, expression_context)?;
        
        // Cache result
        if self.config.enable_rule_caching {
            let cache_key = format!("{}:{}", expression, expression_context.get_hash());
            let mut cache = self.expression_cache.write();
            cache.put(cache_key, Value::Boolean(result));
        }
        
        Ok(result)
    }
    
    /// Internal expression evaluation
    fn evaluate_expression_internal(
        &self,
        expression: &str,
        expression_context: &ExpressionContext,
    ) -> ProtocolResult<bool> {
        // Create evalexpr context
        let mut eval_context = HashMapContext::new();
        
        // Add clinical context variables
        for (key, value) in &expression_context.variables {
            eval_context.set_value(key.clone(), value.clone())
                .map_err(|e| ProtocolEngineError::ExpressionError {
                    context: "variable_binding".to_string(),
                    message: e.to_string(),
                })?;
        }
        
        // Evaluate expression
        let result = eval_with_context(expression, &eval_context)
            .map_err(|e| ProtocolEngineError::ExpressionError {
                context: "evaluation".to_string(),
                message: format!("Failed to evaluate expression '{}': {}", expression, e),
            })?;
        
        // Convert result to boolean
        match result {
            Value::Boolean(b) => Ok(b),
            Value::Float(f) => Ok(f != 0.0),
            Value::Int(i) => Ok(i != 0),
            Value::String(s) => Ok(!s.is_empty() && s != "false" && s != "0"),
            _ => Ok(false),
        }
    }
    
    /// Build expression evaluation context from clinical data
    fn build_expression_context(
        &self,
        clinical_context: &ProtocolContext,
        snapshot_context: Option<&SnapshotContext>,
        eval_context: &EvaluationContext,
    ) -> ProtocolResult<ExpressionContext> {
        let mut variables = HashMap::new();
        
        // Patient demographics
        if let Some(age) = clinical_context.patient_demographics.age {
            variables.insert("patient_age".to_string(), Value::Int(age as i64));
        }
        if let Some(weight) = clinical_context.patient_demographics.weight_kg {
            variables.insert("patient_weight_kg".to_string(), Value::Float(weight));
        }
        if let Some(height) = clinical_context.patient_demographics.height_cm {
            variables.insert("patient_height_cm".to_string(), Value::Float(height));
        }
        if let Some(ref gender) = clinical_context.patient_demographics.gender {
            variables.insert("patient_gender".to_string(), Value::String(gender.clone()));
        }
        
        // Calculate derived values
        if let (Some(weight), Some(height)) = (
            clinical_context.patient_demographics.weight_kg,
            clinical_context.patient_demographics.height_cm,
        ) {
            let height_m = height / 100.0;
            let bmi = weight / (height_m * height_m);
            variables.insert("patient_bmi".to_string(), Value::Float(bmi));
        }
        
        // Active medications
        variables.insert("medication_count".to_string(), 
            Value::Int(clinical_context.medications.len() as i64));
        
        let active_meds: Vec<_> = clinical_context.medications
            .iter()
            .filter(|m| matches!(m.status, MedicationStatus::Active))
            .collect();
        variables.insert("active_medication_count".to_string(), 
            Value::Int(active_meds.len() as i64));
        
        // Check for specific medication types
        for med in &clinical_context.medications {
            let var_name = format!("has_medication_{}", med.name.to_lowercase().replace(' ', "_"));
            variables.insert(var_name, Value::Boolean(matches!(med.status, MedicationStatus::Active)));
        }
        
        // Active conditions
        variables.insert("condition_count".to_string(), 
            Value::Int(clinical_context.conditions.len() as i64));
        
        let active_conditions: Vec<_> = clinical_context.conditions
            .iter()
            .filter(|c| matches!(c.status, ConditionStatus::Active))
            .collect();
        variables.insert("active_condition_count".to_string(), 
            Value::Int(active_conditions.len() as i64));
        
        // Check for specific conditions
        for condition in &clinical_context.conditions {
            let var_name = format!("has_condition_{}", condition.code.to_lowercase().replace('-', "_"));
            variables.insert(var_name, Value::Boolean(matches!(condition.status, ConditionStatus::Active)));
        }
        
        // Allergies
        variables.insert("allergy_count".to_string(), 
            Value::Int(clinical_context.allergies.len() as i64));
        
        for allergy in &clinical_context.allergies {
            let var_name = format!("allergic_to_{}", allergy.substance.to_lowercase().replace(' ', "_"));
            variables.insert(var_name, Value::Boolean(true));
        }
        
        // Latest lab results
        for lab in &clinical_context.lab_results {
            let var_name = format!("lab_{}", lab.test_name.to_lowercase().replace(' ', "_"));
            match &lab.value {
                LabValue::Numeric(n) => {
                    variables.insert(var_name, Value::Float(*n));
                },
                LabValue::Boolean(b) => {
                    variables.insert(var_name, Value::Boolean(*b));
                },
                LabValue::Text(t) => {
                    variables.insert(var_name, Value::String(t.clone()));
                },
            }
        }
        
        // Latest vital signs
        for vital in &clinical_context.vital_signs {
            let var_name = match vital.vital_type {
                VitalSignType::BloodPressureSystolic => "bp_systolic",
                VitalSignType::BloodPressureDiastolic => "bp_diastolic",
                VitalSignType::HeartRate => "heart_rate",
                VitalSignType::RespiratoryRate => "respiratory_rate",
                VitalSignType::Temperature => "temperature",
                VitalSignType::OxygenSaturation => "oxygen_saturation",
                VitalSignType::Pain => "pain_score",
            };
            variables.insert(var_name.to_string(), Value::Float(vital.value));
        }
        
        // Encounter context
        if let Some(ref department) = clinical_context.encounter_context.department {
            variables.insert("department".to_string(), Value::String(department.clone()));
        }
        
        match clinical_context.encounter_context.encounter_type {
            EncounterType::Inpatient => {
                variables.insert("is_inpatient".to_string(), Value::Boolean(true));
            },
            EncounterType::Emergency => {
                variables.insert("is_emergency".to_string(), Value::Boolean(true));
            },
            _ => {},
        }
        
        // Snapshot context
        if let Some(snapshot) = snapshot_context {
            variables.insert("snapshot_id".to_string(), Value::String(snapshot.snapshot_id.clone()));
            // Add snapshot-specific variables as needed
        }
        
        // Time-based variables
        let now = chrono::Utc::now();
        variables.insert("current_hour".to_string(), Value::Int(now.hour() as i64));
        variables.insert("current_day_of_week".to_string(), Value::Int(now.weekday().number_from_monday() as i64));
        
        // Evaluation context
        variables.insert("evaluation_timestamp".to_string(), 
            Value::String(eval_context.request.evaluation_timestamp.to_rfc3339()));
        
        Ok(ExpressionContext {
            variables,
            hash: 0, // Will be calculated when needed
        })
    }
    
    /// Get rule engine metrics
    pub fn get_metrics(&self) -> RuleEngineMetrics {
        RuleEngineMetrics {
            rules_evaluated: std::sync::atomic::AtomicU64::new(
                self.metrics.rules_evaluated.load(std::sync::atomic::Ordering::Relaxed)
            ),
            rules_passed: std::sync::atomic::AtomicU64::new(
                self.metrics.rules_passed.load(std::sync::atomic::Ordering::Relaxed)
            ),
            rules_failed: std::sync::atomic::AtomicU64::new(
                self.metrics.rules_failed.load(std::sync::atomic::Ordering::Relaxed)
            ),
            rules_errored: std::sync::atomic::AtomicU64::new(
                self.metrics.rules_errored.load(std::sync::atomic::Ordering::Relaxed)
            ),
            cache_hits: std::sync::atomic::AtomicU64::new(
                self.metrics.cache_hits.load(std::sync::atomic::Ordering::Relaxed)
            ),
            cache_misses: std::sync::atomic::AtomicU64::new(
                self.metrics.cache_misses.load(std::sync::atomic::Ordering::Relaxed)
            ),
            average_rule_time_ms: std::sync::atomic::AtomicU64::new(
                self.metrics.average_rule_time_ms.load(std::sync::atomic::Ordering::Relaxed)
            ),
        }
    }
}

/// Expression evaluation context
#[derive(Debug, Clone)]
pub struct ExpressionContext {
    pub variables: HashMap<String, Value>,
    hash: u64,
}

impl ExpressionContext {
    pub fn get_hash(&self) -> u64 {
        if self.hash == 0 {
            // Calculate hash lazily (in production, this would be more sophisticated)
            format!("{:?}", self.variables).chars().map(|c| c as u64).sum()
        } else {
            self.hash
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Utc;

    #[tokio::test]
    async fn test_rule_engine_creation() {
        let config = RuleEngineConfig {
            max_rule_execution_time_ms: 1000,
            max_parallel_rules: 10,
            enable_rule_caching: true,
            cache_size: 100,
        };
        
        let engine = RuleEngine::new(&config);
        assert!(engine.is_ok());
    }

    #[tokio::test]
    async fn test_simple_expression_evaluation() {
        let config = RuleEngineConfig {
            max_rule_execution_time_ms: 1000,
            max_parallel_rules: 10,
            enable_rule_caching: true,
            cache_size: 100,
        };
        
        let engine = RuleEngine::new(&config).unwrap();
        
        let mut context = ExpressionContext {
            variables: HashMap::new(),
            hash: 0,
        };
        context.variables.insert("patient_age".to_string(), Value::Int(45));
        
        let result = engine.evaluate_expression("patient_age > 40", &context).await;
        assert!(result.is_ok());
        assert_eq!(result.unwrap(), true);
        
        let result2 = engine.evaluate_expression("patient_age < 30", &context).await;
        assert!(result2.is_ok());
        assert_eq!(result2.unwrap(), false);
    }

    #[tokio::test]
    async fn test_rule_evaluation() {
        let config = RuleEngineConfig {
            max_rule_execution_time_ms: 1000,
            max_parallel_rules: 10,
            enable_rule_caching: true,
            cache_size: 100,
        };
        
        let engine = RuleEngine::new(&config).unwrap();
        
        let rule = ProtocolRule {
            rule_id: "age-check".to_string(),
            name: "Age Check".to_string(),
            description: "Check if patient is adult".to_string(),
            condition: RuleCondition::Expression("patient_age >= 18".to_string()),
            action: RuleAction {
                action_type: RuleActionType::Allow,
                parameters: HashMap::new(),
                message: None,
            },
            priority: 1,
            enabled: true,
        };
        
        let clinical_context = ProtocolContext {
            patient_demographics: PatientDemographics {
                age: Some(45),
                ..Default::default()
            },
            ..Default::default()
        };
        
        let mut eval_context = EvaluationContext::new(
            uuid::Uuid::new_v4(),
            ProtocolEvaluationRequest {
                protocol_id: "test".to_string(),
                patient_id: "test-patient".to_string(),
                clinical_context: clinical_context.clone(),
                snapshot_id: None,
                evaluation_timestamp: Utc::now(),
                metadata: None,
            },
            Instant::now(),
        );
        
        let results = engine.evaluate_rules(
            &[rule],
            &clinical_context,
            None,
            &mut eval_context,
        ).await;
        
        assert!(results.is_ok());
        let results = results.unwrap();
        assert_eq!(results.len(), 1);
        assert!(matches!(results[0].result, RuleResult::Pass));
    }
}