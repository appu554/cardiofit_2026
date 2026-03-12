//! Mathematical Expression Evaluator
//! 
//! This module provides a safe evaluation engine for mathematical expressions
//! with variable substitution and built-in mathematical functions.
//! 
//! Features:
//! - Safe evaluation with bounds checking
//! - Variable substitution from patient context
//! - Built-in mathematical functions (min, max, abs, sqrt, etc.)
//! - Division by zero protection
//! - Overflow/underflow protection
//! - Execution time limits

use std::collections::HashMap;
use std::time::{Duration, Instant};
use anyhow::{Result, anyhow};
use tracing::{debug, warn, error};
use serde::{Deserialize, Serialize};

use super::expression_parser::{ExpressionAST, MathExpression};
use crate::unified_clinical_engine::{PatientContext, BiologicalSex, PregnancyStatus};

/// Evaluation context containing variables and configuration
#[derive(Debug, Clone)]
pub struct EvaluationContext {
    pub variables: HashMap<String, f64>,
    pub config: EvaluationConfig,
}

/// Configuration for expression evaluation
#[derive(Debug, Clone)]
pub struct EvaluationConfig {
    /// Maximum execution time for evaluation
    pub max_execution_time: Duration,
    /// Maximum recursion depth
    pub max_recursion_depth: usize,
    /// Enable division by zero protection
    pub protect_division_by_zero: bool,
    /// Enable overflow protection
    pub protect_overflow: bool,
    /// Minimum allowed result value
    pub min_result_value: f64,
    /// Maximum allowed result value
    pub max_result_value: f64,
}

impl Default for EvaluationConfig {
    fn default() -> Self {
        Self {
            max_execution_time: Duration::from_millis(100),
            max_recursion_depth: 100,
            protect_division_by_zero: true,
            protect_overflow: true,
            min_result_value: -1e6,
            max_result_value: 1e6,
        }
    }
}

/// Result of expression evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    pub value: f64,
    pub execution_time_ms: u64,
    pub variables_used: Vec<String>,
    pub functions_called: Vec<String>,
    pub warnings: Vec<String>,
}

/// Mathematical expression evaluator
pub struct ExpressionEvaluator {
    config: EvaluationConfig,
    built_in_functions: HashMap<String, fn(&[f64]) -> Result<f64>>,
}

impl ExpressionEvaluator {
    pub fn new(config: EvaluationConfig) -> Self {
        let mut built_in_functions: HashMap<String, fn(&[f64]) -> Result<f64>> = HashMap::new();
        
        // Register built-in mathematical functions
        built_in_functions.insert("min".to_string(), Self::func_min);
        built_in_functions.insert("max".to_string(), Self::func_max);
        built_in_functions.insert("abs".to_string(), Self::func_abs);
        built_in_functions.insert("sqrt".to_string(), Self::func_sqrt);
        built_in_functions.insert("log".to_string(), Self::func_log);
        built_in_functions.insert("ln".to_string(), Self::func_ln);
        built_in_functions.insert("exp".to_string(), Self::func_exp);
        built_in_functions.insert("pow".to_string(), Self::func_pow);
        built_in_functions.insert("round".to_string(), Self::func_round);
        built_in_functions.insert("floor".to_string(), Self::func_floor);
        built_in_functions.insert("ceil".to_string(), Self::func_ceil);
        built_in_functions.insert("clamp".to_string(), Self::func_clamp);
        
        Self {
            config,
            built_in_functions,
        }
    }
    
    /// Evaluate a mathematical expression with the given context
    pub fn evaluate(&self, expression: &MathExpression, context: &EvaluationContext) -> Result<EvaluationResult> {
        let start_time = Instant::now();
        
        debug!("Evaluating expression: {}", expression.expression);
        
        // Check if all required variables are available
        for var in &expression.variables {
            if !context.variables.contains_key(var) {
                return Err(anyhow!("Variable '{}' not found in evaluation context", var));
            }
        }
        
        // Check if all required functions are available
        for func in &expression.functions {
            if !self.built_in_functions.contains_key(func) {
                return Err(anyhow!("Function '{}' is not supported", func));
            }
        }
        
        let mut eval_state = EvaluationState {
            variables_used: Vec::new(),
            functions_called: Vec::new(),
            warnings: Vec::new(),
            recursion_depth: 0,
            start_time,
        };
        
        // Evaluate the AST
        let result = self.evaluate_ast(&expression.parsed_ast, context, &mut eval_state)?;
        
        let execution_time = start_time.elapsed();
        
        // Check execution time limit
        if execution_time > self.config.max_execution_time {
            warn!("Expression evaluation exceeded time limit: {:?}", execution_time);
            eval_state.warnings.push(format!("Evaluation took {:?}, which exceeds the limit of {:?}", 
                                            execution_time, self.config.max_execution_time));
        }
        
        // Validate result bounds
        if result < self.config.min_result_value || result > self.config.max_result_value {
            return Err(anyhow!("Result {} is outside allowed bounds [{}, {}]", 
                              result, self.config.min_result_value, self.config.max_result_value));
        }
        
        debug!("Expression evaluated to: {} in {:?}", result, execution_time);
        
        Ok(EvaluationResult {
            value: result,
            execution_time_ms: execution_time.as_millis() as u64,
            variables_used: eval_state.variables_used,
            functions_called: eval_state.functions_called,
            warnings: eval_state.warnings,
        })
    }
    
    /// Create evaluation context from patient data
    pub fn create_context_from_patient(patient: &PatientContext, config: Option<EvaluationConfig>) -> EvaluationContext {
        let mut variables = HashMap::new();
        
        // Basic demographics
        variables.insert("age".to_string(), patient.age_years);
        variables.insert("weight".to_string(), patient.weight_kg);
        variables.insert("height".to_string(), patient.height_cm);
        variables.insert("bmi".to_string(), patient.weight_kg / (patient.height_cm / 100.0).powi(2));
        variables.insert("bsa".to_string(), Self::calculate_bsa(patient.weight_kg, patient.height_cm));
        
        // Organ function
        if let Some(egfr) = patient.renal_function.egfr_ml_min_1_73m2 {
            variables.insert("egfr".to_string(), egfr);
        }
        
        // Gender (0 = female, 1 = male for calculations)
        let is_male = match patient.sex {
            BiologicalSex::Male => 1.0,
            BiologicalSex::Female => 0.0,
            BiologicalSex::Other => 0.5, // Neutral value for other/unknown
        };
        variables.insert("is_male".to_string(), is_male);
        variables.insert("is_female".to_string(), if matches!(patient.sex, BiologicalSex::Female) { 1.0 } else { 0.0 });
        
        // Pregnancy status (0 = not pregnant, 1 = pregnant)
        variables.insert("is_pregnant".to_string(), if matches!(patient.pregnancy_status, PregnancyStatus::Pregnant { .. }) { 1.0 } else { 0.0 });
        
        // Age categories
        variables.insert("is_pediatric".to_string(), if patient.age_years < 18.0 { 1.0 } else { 0.0 });
        variables.insert("is_elderly".to_string(), if patient.age_years >= 65.0 { 1.0 } else { 0.0 });
        variables.insert("is_very_elderly".to_string(), if patient.age_years >= 80.0 { 1.0 } else { 0.0 });
        
        // Weight categories
        variables.insert("is_underweight".to_string(), if patient.weight_kg < 50.0 { 1.0 } else { 0.0 });
        variables.insert("is_overweight".to_string(), if patient.weight_kg > 100.0 { 1.0 } else { 0.0 });
        
        EvaluationContext {
            variables,
            config: config.unwrap_or_default(),
        }
    }
    
    /// Calculate body surface area using Mosteller formula
    fn calculate_bsa(weight_kg: f64, height_cm: f64) -> f64 {
        ((weight_kg * height_cm) / 3600.0).sqrt()
    }
}

/// Internal state during evaluation
struct EvaluationState {
    variables_used: Vec<String>,
    functions_called: Vec<String>,
    warnings: Vec<String>,
    recursion_depth: usize,
    start_time: Instant,
}

impl ExpressionEvaluator {
    /// Evaluate an AST node
    fn evaluate_ast(&self, ast: &ExpressionAST, context: &EvaluationContext, state: &mut EvaluationState) -> Result<f64> {
        // Check recursion depth
        if state.recursion_depth > self.config.max_recursion_depth {
            return Err(anyhow!("Maximum recursion depth exceeded"));
        }
        
        // Check execution time
        if state.start_time.elapsed() > self.config.max_execution_time {
            return Err(anyhow!("Evaluation time limit exceeded"));
        }
        
        state.recursion_depth += 1;
        
        let result = match ast {
            ExpressionAST::Number(value) => Ok(*value),
            
            ExpressionAST::Variable(name) => {
                state.variables_used.push(name.clone());
                context.variables.get(name)
                    .copied()
                    .ok_or_else(|| anyhow!("Variable '{}' not found", name))
            }
            
            ExpressionAST::Add(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_add(left_val, right_val)
            }
            
            ExpressionAST::Subtract(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_subtract(left_val, right_val)
            }
            
            ExpressionAST::Multiply(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_multiply(left_val, right_val)
            }
            
            ExpressionAST::Divide(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_divide(left_val, right_val)
            }
            
            ExpressionAST::Power(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_power(left_val, right_val)
            }
            
            ExpressionAST::Modulo(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                self.safe_modulo(left_val, right_val)
            }
            
            ExpressionAST::Equal(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if (left_val - right_val).abs() < f64::EPSILON { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::NotEqual(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if (left_val - right_val).abs() >= f64::EPSILON { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::LessThan(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if left_val < right_val { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::LessEqual(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if left_val <= right_val { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::GreaterThan(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if left_val > right_val { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::GreaterEqual(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                let right_val = self.evaluate_ast(right, context, state)?;
                Ok(if left_val >= right_val { 1.0 } else { 0.0 })
            }
            
            ExpressionAST::And(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                // Short-circuit evaluation
                if self.is_truthy(left_val) {
                    let right_val = self.evaluate_ast(right, context, state)?;
                    Ok(if self.is_truthy(right_val) { 1.0 } else { 0.0 })
                } else {
                    Ok(0.0)
                }
            }
            
            ExpressionAST::Or(left, right) => {
                let left_val = self.evaluate_ast(left, context, state)?;
                // Short-circuit evaluation
                if self.is_truthy(left_val) {
                    Ok(1.0)
                } else {
                    let right_val = self.evaluate_ast(right, context, state)?;
                    Ok(if self.is_truthy(right_val) { 1.0 } else { 0.0 })
                }
            }
            
            ExpressionAST::Not(expr) => {
                let val = self.evaluate_ast(expr, context, state)?;
                Ok(if self.is_truthy(val) { 0.0 } else { 1.0 })
            }
            
            ExpressionAST::Conditional { condition, true_expr, false_expr } => {
                let condition_val = self.evaluate_ast(condition, context, state)?;
                if self.is_truthy(condition_val) {
                    self.evaluate_ast(true_expr, context, state)
                } else {
                    self.evaluate_ast(false_expr, context, state)
                }
            }
            
            ExpressionAST::Function { name, args } => {
                state.functions_called.push(name.clone());
                
                let mut arg_values = Vec::new();
                for arg in args {
                    arg_values.push(self.evaluate_ast(arg, context, state)?);
                }
                
                if let Some(func) = self.built_in_functions.get(name) {
                    func(&arg_values)
                } else {
                    Err(anyhow!("Unknown function: {}", name))
                }
            }
        };
        
        state.recursion_depth -= 1;
        result
    }
    
    /// Check if a value is considered "truthy" (non-zero)
    fn is_truthy(&self, value: f64) -> bool {
        value.abs() > f64::EPSILON
    }
    
    /// Safe arithmetic operations with overflow protection
    fn safe_add(&self, a: f64, b: f64) -> Result<f64> {
        let result = a + b;
        if self.config.protect_overflow && (result.is_infinite() || result.is_nan()) {
            Err(anyhow!("Addition overflow: {} + {}", a, b))
        } else {
            Ok(result)
        }
    }
    
    fn safe_subtract(&self, a: f64, b: f64) -> Result<f64> {
        let result = a - b;
        if self.config.protect_overflow && (result.is_infinite() || result.is_nan()) {
            Err(anyhow!("Subtraction overflow: {} - {}", a, b))
        } else {
            Ok(result)
        }
    }
    
    fn safe_multiply(&self, a: f64, b: f64) -> Result<f64> {
        let result = a * b;
        if self.config.protect_overflow && (result.is_infinite() || result.is_nan()) {
            Err(anyhow!("Multiplication overflow: {} * {}", a, b))
        } else {
            Ok(result)
        }
    }
    
    fn safe_divide(&self, a: f64, b: f64) -> Result<f64> {
        if self.config.protect_division_by_zero && b.abs() < f64::EPSILON {
            Err(anyhow!("Division by zero: {} / {}", a, b))
        } else {
            let result = a / b;
            if self.config.protect_overflow && (result.is_infinite() || result.is_nan()) {
                Err(anyhow!("Division overflow: {} / {}", a, b))
            } else {
                Ok(result)
            }
        }
    }
    
    fn safe_power(&self, base: f64, exponent: f64) -> Result<f64> {
        let result = base.powf(exponent);
        if self.config.protect_overflow && (result.is_infinite() || result.is_nan()) {
            Err(anyhow!("Power overflow: {} ^ {}", base, exponent))
        } else {
            Ok(result)
        }
    }
    
    fn safe_modulo(&self, a: f64, b: f64) -> Result<f64> {
        if self.config.protect_division_by_zero && b.abs() < f64::EPSILON {
            Err(anyhow!("Modulo by zero: {} % {}", a, b))
        } else {
            Ok(a % b)
        }
    }
}

// Built-in mathematical functions
impl ExpressionEvaluator {
    fn func_min(args: &[f64]) -> Result<f64> {
        if args.is_empty() {
            return Err(anyhow!("min() requires at least one argument"));
        }
        Ok(args.iter().fold(f64::INFINITY, |a, &b| a.min(b)))
    }
    
    fn func_max(args: &[f64]) -> Result<f64> {
        if args.is_empty() {
            return Err(anyhow!("max() requires at least one argument"));
        }
        Ok(args.iter().fold(f64::NEG_INFINITY, |a, &b| a.max(b)))
    }
    
    fn func_abs(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("abs() requires exactly one argument"));
        }
        Ok(args[0].abs())
    }
    
    fn func_sqrt(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("sqrt() requires exactly one argument"));
        }
        if args[0] < 0.0 {
            return Err(anyhow!("sqrt() of negative number: {}", args[0]));
        }
        Ok(args[0].sqrt())
    }
    
    fn func_log(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("log() requires exactly one argument"));
        }
        if args[0] <= 0.0 {
            return Err(anyhow!("log() of non-positive number: {}", args[0]));
        }
        Ok(args[0].log10())
    }
    
    fn func_ln(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("ln() requires exactly one argument"));
        }
        if args[0] <= 0.0 {
            return Err(anyhow!("ln() of non-positive number: {}", args[0]));
        }
        Ok(args[0].ln())
    }
    
    fn func_exp(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("exp() requires exactly one argument"));
        }
        let result = args[0].exp();
        if result.is_infinite() {
            return Err(anyhow!("exp() overflow: {}", args[0]));
        }
        Ok(result)
    }
    
    fn func_pow(args: &[f64]) -> Result<f64> {
        if args.len() != 2 {
            return Err(anyhow!("pow() requires exactly two arguments"));
        }
        let result = args[0].powf(args[1]);
        if result.is_infinite() || result.is_nan() {
            return Err(anyhow!("pow() overflow: {} ^ {}", args[0], args[1]));
        }
        Ok(result)
    }
    
    fn func_round(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("round() requires exactly one argument"));
        }
        Ok(args[0].round())
    }
    
    fn func_floor(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("floor() requires exactly one argument"));
        }
        Ok(args[0].floor())
    }
    
    fn func_ceil(args: &[f64]) -> Result<f64> {
        if args.len() != 1 {
            return Err(anyhow!("ceil() requires exactly one argument"));
        }
        Ok(args[0].ceil())
    }
    
    fn func_clamp(args: &[f64]) -> Result<f64> {
        if args.len() != 3 {
            return Err(anyhow!("clamp() requires exactly three arguments: clamp(value, min, max)"));
        }
        let value = args[0];
        let min_val = args[1];
        let max_val = args[2];
        
        if min_val > max_val {
            return Err(anyhow!("clamp() min value {} is greater than max value {}", min_val, max_val));
        }
        
        Ok(value.max(min_val).min(max_val))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::unified_clinical_engine::expression_parser::ExpressionParser;
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

    #[test]
    fn test_simple_arithmetic() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("2 + 3 * 4").unwrap();
        let context = EvaluationContext {
            variables: HashMap::new(),
            config: EvaluationConfig::default(),
        };
        
        let result = evaluator.evaluate(&expr, &context).unwrap();
        assert_eq!(result.value, 14.0);
    }

    #[test]
    fn test_variable_substitution() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("weight * 0.5").unwrap();
        let patient = create_test_patient();
        let context = ExpressionEvaluator::create_context_from_patient(&patient, None);
        
        let result = evaluator.evaluate(&expr, &context).unwrap();
        assert_eq!(result.value, 35.0); // 70 * 0.5
        assert!(result.variables_used.contains(&"weight".to_string()));
    }

    #[test]
    fn test_conditional_expression() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("age > 65 ? 10 : 5").unwrap();
        let patient = create_test_patient();
        let context = ExpressionEvaluator::create_context_from_patient(&patient, None);
        
        let result = evaluator.evaluate(&expr, &context).unwrap();
        assert_eq!(result.value, 5.0); // age is 45, so condition is false
    }

    #[test]
    fn test_function_calls() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("min(weight, 100)").unwrap();
        let patient = create_test_patient();
        let context = ExpressionEvaluator::create_context_from_patient(&patient, None);
        
        let result = evaluator.evaluate(&expr, &context).unwrap();
        assert_eq!(result.value, 70.0); // min(70, 100) = 70
        assert!(result.functions_called.contains(&"min".to_string()));
    }

    #[test]
    fn test_complex_expression() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("(weight * 0.5) + (age > 65 ? 10 : 0) + min(egfr / 60, 1.0)").unwrap();
        let patient = create_test_patient();
        let context = ExpressionEvaluator::create_context_from_patient(&patient, None);
        
        let result = evaluator.evaluate(&expr, &context).unwrap();
        // (70 * 0.5) + 0 + min(90/60, 1.0) = 35 + 0 + 1.0 = 36.0
        assert_eq!(result.value, 36.0);
    }

    #[test]
    fn test_division_by_zero_protection() {
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        let expr = ExpressionParser::parse("10 / 0").unwrap();
        let context = EvaluationContext {
            variables: HashMap::new(),
            config: EvaluationConfig::default(),
        };
        
        let result = evaluator.evaluate(&expr, &context);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("Division by zero"));
    }
}
