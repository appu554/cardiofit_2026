//! Expression Validation and Safety Checks
//! 
//! This module provides comprehensive validation and safety checks for mathematical
//! expressions used in clinical rules to ensure they are safe, reasonable, and
//! clinically appropriate.
//! 
//! Features:
//! - Syntax validation
//! - Clinical range validation
//! - Security checks (prevent malicious expressions)
//! - Performance validation (complexity limits)
//! - Medical reasonableness checks

use std::collections::HashSet;
use anyhow::{Result, anyhow};
use serde::{Deserialize, Serialize};
use tracing::{debug, warn, error};

use super::expression_parser::{ExpressionParser, MathExpression, ExpressionAST};
use super::expression_evaluator::{ExpressionEvaluator, EvaluationContext, EvaluationConfig};
use super::variable_substitution::VariableSubstitution;
use crate::unified_clinical_engine::ClinicalRequest;

/// Expression validation result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    pub is_valid: bool,
    pub errors: Vec<String>,
    pub warnings: Vec<String>,
    pub security_issues: Vec<String>,
    pub performance_issues: Vec<String>,
    pub clinical_concerns: Vec<String>,
    pub complexity_score: u32,
    pub estimated_execution_time_ms: u64,
}

/// Expression validation configuration
#[derive(Debug, Clone)]
pub struct ValidationConfig {
    /// Maximum allowed complexity score
    pub max_complexity: u32,
    /// Maximum estimated execution time in milliseconds
    pub max_execution_time_ms: u64,
    /// Maximum recursion depth
    pub max_recursion_depth: u32,
    /// Allowed variable names
    pub allowed_variables: HashSet<String>,
    /// Allowed function names
    pub allowed_functions: HashSet<String>,
    /// Clinical value ranges for validation
    pub clinical_ranges: std::collections::HashMap<String, (f64, f64)>,
}

impl Default for ValidationConfig {
    fn default() -> Self {
        let mut allowed_variables = HashSet::new();
        allowed_variables.insert("age".to_string());
        allowed_variables.insert("weight".to_string());
        allowed_variables.insert("height".to_string());
        allowed_variables.insert("bmi".to_string());
        allowed_variables.insert("bsa".to_string());
        allowed_variables.insert("egfr".to_string());
        allowed_variables.insert("creatinine".to_string());
        allowed_variables.insert("is_male".to_string());
        allowed_variables.insert("is_female".to_string());
        allowed_variables.insert("is_pregnant".to_string());
        allowed_variables.insert("is_pediatric".to_string());
        allowed_variables.insert("is_elderly".to_string());
        
        let mut allowed_functions = HashSet::new();
        allowed_functions.insert("min".to_string());
        allowed_functions.insert("max".to_string());
        allowed_functions.insert("abs".to_string());
        allowed_functions.insert("sqrt".to_string());
        allowed_functions.insert("round".to_string());
        allowed_functions.insert("floor".to_string());
        allowed_functions.insert("ceil".to_string());
        allowed_functions.insert("clamp".to_string());
        
        let mut clinical_ranges = std::collections::HashMap::new();
        clinical_ranges.insert("age".to_string(), (0.0, 150.0));
        clinical_ranges.insert("weight".to_string(), (0.5, 500.0));
        clinical_ranges.insert("height".to_string(), (30.0, 250.0));
        clinical_ranges.insert("egfr".to_string(), (0.0, 200.0));
        clinical_ranges.insert("dose".to_string(), (0.001, 10000.0)); // mg
        
        Self {
            max_complexity: 100,
            max_execution_time_ms: 50,
            max_recursion_depth: 20,
            allowed_variables,
            allowed_functions,
            clinical_ranges,
        }
    }
}

/// Expression validator for clinical mathematical expressions
pub struct ExpressionValidator {
    config: ValidationConfig,
    variable_substitution: VariableSubstitution,
    evaluator: ExpressionEvaluator,
}

impl ExpressionValidator {
    pub fn new(config: ValidationConfig) -> Self {
        let variable_substitution = VariableSubstitution::new();
        let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
        
        Self {
            config,
            variable_substitution,
            evaluator,
        }
    }
    
    /// Validate a mathematical expression for clinical use
    pub fn validate_expression(&self, expression: &str, sample_request: Option<&ClinicalRequest>) -> Result<ValidationResult> {
        let mut result = ValidationResult {
            is_valid: true,
            errors: Vec::new(),
            warnings: Vec::new(),
            security_issues: Vec::new(),
            performance_issues: Vec::new(),
            clinical_concerns: Vec::new(),
            complexity_score: 0,
            estimated_execution_time_ms: 0,
        };
        
        debug!("Validating expression: {}", expression);
        
        // Step 1: Syntax validation
        let parsed_expr = match ExpressionParser::parse(expression) {
            Ok(expr) => expr,
            Err(e) => {
                result.is_valid = false;
                result.errors.push(format!("Syntax error: {}", e));
                return Ok(result);
            }
        };
        
        // Step 2: Security validation
        self.validate_security(&parsed_expr, &mut result);
        
        // Step 3: Performance validation
        self.validate_performance(&parsed_expr, &mut result);
        
        // Step 4: Variable and function validation
        self.validate_identifiers(&parsed_expr, &mut result);
        
        // Step 5: Clinical reasonableness validation
        if let Some(request) = sample_request {
            self.validate_clinical_reasonableness(&parsed_expr, request, &mut result)?;
        }
        
        // Step 6: Test evaluation with sample data
        if result.is_valid && sample_request.is_some() {
            self.test_evaluation(&parsed_expr, sample_request.unwrap(), &mut result)?;
        }
        
        // Final validation decision
        result.is_valid = result.errors.is_empty() && result.security_issues.is_empty();
        
        debug!("Expression validation complete: valid={}, errors={}, warnings={}", 
               result.is_valid, result.errors.len(), result.warnings.len());
        
        Ok(result)
    }
    
    /// Validate expression security (prevent malicious code)
    fn validate_security(&self, expr: &MathExpression, result: &mut ValidationResult) {
        // Check for suspicious patterns
        let expression_lower = expr.expression.to_lowercase();
        
        // Check for potentially dangerous patterns
        let dangerous_patterns = [
            "eval", "exec", "system", "import", "require", "include",
            "file", "read", "write", "delete", "drop", "truncate",
            "while", "for", "loop", "goto", "break", "continue",
        ];
        
        for pattern in &dangerous_patterns {
            if expression_lower.contains(pattern) {
                result.security_issues.push(format!("Potentially dangerous pattern detected: {}", pattern));
            }
        }
        
        // Check for excessive complexity that could be a DoS attack
        if expr.expression.len() > 1000 {
            result.security_issues.push("Expression is excessively long (potential DoS)".to_string());
        }
        
        // Check for deeply nested expressions
        let nesting_depth = self.calculate_nesting_depth(&expr.parsed_ast);
        if nesting_depth > self.config.max_recursion_depth {
            result.security_issues.push(format!("Expression nesting depth {} exceeds limit {}", 
                                               nesting_depth, self.config.max_recursion_depth));
        }
    }
    
    /// Validate expression performance characteristics
    fn validate_performance(&self, expr: &MathExpression, result: &mut ValidationResult) {
        // Calculate complexity score
        result.complexity_score = self.calculate_complexity(&expr.parsed_ast);
        
        if result.complexity_score > self.config.max_complexity {
            result.performance_issues.push(format!("Expression complexity {} exceeds limit {}", 
                                                  result.complexity_score, self.config.max_complexity));
        }
        
        // Estimate execution time based on complexity
        result.estimated_execution_time_ms = (result.complexity_score as u64).saturating_mul(2);
        
        if result.estimated_execution_time_ms > self.config.max_execution_time_ms {
            result.performance_issues.push(format!("Estimated execution time {}ms exceeds limit {}ms", 
                                                  result.estimated_execution_time_ms, self.config.max_execution_time_ms));
        }
        
        // Check for potentially expensive operations
        if expr.functions.iter().any(|f| f == "exp" || f == "pow") {
            result.warnings.push("Expression contains potentially expensive mathematical operations".to_string());
        }
    }
    
    /// Validate variables and functions used in the expression
    fn validate_identifiers(&self, expr: &MathExpression, result: &mut ValidationResult) {
        // Validate variables
        for var in &expr.variables {
            if !self.config.allowed_variables.contains(var) {
                result.errors.push(format!("Unknown or disallowed variable: {}", var));
            }
        }
        
        // Validate functions
        for func in &expr.functions {
            if !self.config.allowed_functions.contains(func) {
                result.errors.push(format!("Unknown or disallowed function: {}", func));
            }
        }
        
        // Check for missing critical variables
        if expr.variables.is_empty() && !expr.expression.chars().any(|c| c.is_ascii_digit()) {
            result.warnings.push("Expression contains no variables or constants".to_string());
        }
    }
    
    /// Validate clinical reasonableness of the expression
    fn validate_clinical_reasonableness(&self, expr: &MathExpression, request: &ClinicalRequest, result: &mut ValidationResult) -> Result<()> {
        // Check if expression uses clinically relevant variables
        let clinical_vars = ["age", "weight", "height", "egfr", "bmi", "bsa"];
        let uses_clinical_vars = expr.variables.iter().any(|v| clinical_vars.contains(&v.as_str()));
        
        if !uses_clinical_vars && !expr.expression.chars().any(|c| c.is_ascii_digit()) {
            result.clinical_concerns.push("Expression does not use any clinical variables".to_string());
        }
        
        // Check for clinically unreasonable combinations
        if expr.variables.contains(&"is_pregnant".to_string()) && expr.variables.contains(&"is_male".to_string()) {
            result.warnings.push("Expression uses both pregnancy and male gender variables".to_string());
        }
        
        // Check for age-related logic consistency
        if expr.variables.contains(&"is_pediatric".to_string()) && expr.variables.contains(&"is_elderly".to_string()) {
            result.warnings.push("Expression uses both pediatric and elderly age categories".to_string());
        }
        
        Ok(())
    }
    
    /// Test expression evaluation with sample data
    fn test_evaluation(&self, expr: &MathExpression, request: &ClinicalRequest, result: &mut ValidationResult) -> Result<()> {
        // Create evaluation context
        let substitution = self.variable_substitution.create_substitution(request)?;
        let eval_context = EvaluationContext {
            variables: substitution.variables,
            config: EvaluationConfig::default(),
        };
        
        // Try to evaluate the expression
        match self.evaluator.evaluate(expr, &eval_context) {
            Ok(eval_result) => {
                // Check if result is clinically reasonable
                if eval_result.value.is_nan() || eval_result.value.is_infinite() {
                    result.errors.push("Expression evaluates to NaN or infinity".to_string());
                } else if eval_result.value < 0.0 {
                    result.clinical_concerns.push("Expression evaluates to negative value (may be inappropriate for dose)".to_string());
                } else if eval_result.value > 10000.0 {
                    result.clinical_concerns.push("Expression evaluates to very large value (may be inappropriate for dose)".to_string());
                }
                
                // Update actual execution time
                result.estimated_execution_time_ms = eval_result.execution_time_ms;
            }
            Err(e) => {
                result.errors.push(format!("Expression evaluation failed: {}", e));
            }
        }
        
        Ok(())
    }
    
    /// Calculate the complexity score of an expression
    fn calculate_complexity(&self, ast: &ExpressionAST) -> u32 {
        match ast {
            ExpressionAST::Number(_) => 1,
            ExpressionAST::Variable(_) => 2,
            ExpressionAST::Add(left, right) |
            ExpressionAST::Subtract(left, right) |
            ExpressionAST::Multiply(left, right) |
            ExpressionAST::Divide(left, right) |
            ExpressionAST::Equal(left, right) |
            ExpressionAST::NotEqual(left, right) |
            ExpressionAST::LessThan(left, right) |
            ExpressionAST::LessEqual(left, right) |
            ExpressionAST::GreaterThan(left, right) |
            ExpressionAST::GreaterEqual(left, right) |
            ExpressionAST::And(left, right) |
            ExpressionAST::Or(left, right) => {
                3 + self.calculate_complexity(left) + self.calculate_complexity(right)
            }
            ExpressionAST::Power(left, right) => {
                5 + self.calculate_complexity(left) + self.calculate_complexity(right)
            }
            ExpressionAST::Modulo(left, right) => {
                4 + self.calculate_complexity(left) + self.calculate_complexity(right)
            }
            ExpressionAST::Not(expr) => {
                2 + self.calculate_complexity(expr)
            }
            ExpressionAST::Conditional { condition, true_expr, false_expr } => {
                5 + self.calculate_complexity(condition) + 
                self.calculate_complexity(true_expr) + 
                self.calculate_complexity(false_expr)
            }
            ExpressionAST::Function { name: _, args } => {
                let base_cost = match args.len() {
                    0 => 3,
                    1 => 4,
                    2 => 5,
                    _ => 6 + args.len() as u32,
                };
                base_cost + args.iter().map(|arg| self.calculate_complexity(arg)).sum::<u32>()
            }
        }
    }
    
    /// Calculate the nesting depth of an expression
    fn calculate_nesting_depth(&self, ast: &ExpressionAST) -> u32 {
        match ast {
            ExpressionAST::Number(_) | ExpressionAST::Variable(_) => 1,
            ExpressionAST::Add(left, right) |
            ExpressionAST::Subtract(left, right) |
            ExpressionAST::Multiply(left, right) |
            ExpressionAST::Divide(left, right) |
            ExpressionAST::Power(left, right) |
            ExpressionAST::Modulo(left, right) |
            ExpressionAST::Equal(left, right) |
            ExpressionAST::NotEqual(left, right) |
            ExpressionAST::LessThan(left, right) |
            ExpressionAST::LessEqual(left, right) |
            ExpressionAST::GreaterThan(left, right) |
            ExpressionAST::GreaterEqual(left, right) |
            ExpressionAST::And(left, right) |
            ExpressionAST::Or(left, right) => {
                1 + self.calculate_nesting_depth(left).max(self.calculate_nesting_depth(right))
            }
            ExpressionAST::Not(expr) => {
                1 + self.calculate_nesting_depth(expr)
            }
            ExpressionAST::Conditional { condition, true_expr, false_expr } => {
                1 + self.calculate_nesting_depth(condition)
                    .max(self.calculate_nesting_depth(true_expr))
                    .max(self.calculate_nesting_depth(false_expr))
            }
            ExpressionAST::Function { name: _, args } => {
                if args.is_empty() {
                    1
                } else {
                    1 + args.iter().map(|arg| self.calculate_nesting_depth(arg)).max().unwrap_or(0)
                }
            }
        }
    }
}

impl Default for ExpressionValidator {
    fn default() -> Self {
        Self::new(ValidationConfig::default())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::unified_clinical_engine::{BiologicalSex, PregnancyStatus, RenalFunction, HepaticFunction};

    fn create_test_request() -> ClinicalRequest {
        ClinicalRequest {
            request_id: "test-123".to_string(),
            patient_context: crate::unified_clinical_engine::PatientContext {
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
                lab_values: std::collections::HashMap::new(),
            },
            drug_id: "metformin".to_string(),
            indication: "diabetes".to_string(),
            timestamp: chrono::Utc::now(),
        }
    }

    #[test]
    fn test_valid_expression() {
        let validator = ExpressionValidator::default();
        let request = create_test_request();
        
        let result = validator.validate_expression("weight * 0.5", Some(&request)).unwrap();
        assert!(result.is_valid);
        assert!(result.errors.is_empty());
    }

    #[test]
    fn test_invalid_syntax() {
        let validator = ExpressionValidator::default();
        
        let result = validator.validate_expression("weight * (", None).unwrap();
        assert!(!result.is_valid);
        assert!(!result.errors.is_empty());
    }

    #[test]
    fn test_security_validation() {
        let validator = ExpressionValidator::default();
        
        let result = validator.validate_expression("eval(malicious_code)", None).unwrap();
        assert!(!result.is_valid);
        assert!(!result.security_issues.is_empty());
    }

    #[test]
    fn test_unknown_variable() {
        let validator = ExpressionValidator::default();
        
        let result = validator.validate_expression("unknown_var * 2", None).unwrap();
        assert!(!result.is_valid);
        assert!(result.errors.iter().any(|e| e.contains("unknown_var")));
    }

    #[test]
    fn test_complex_valid_expression() {
        let validator = ExpressionValidator::default();
        let request = create_test_request();
        
        let result = validator.validate_expression(
            "(weight * 0.5) + (age > 65 ? 10 : 0) + min(egfr / 60, 1.0)", 
            Some(&request)
        ).unwrap();
        
        assert!(result.is_valid);
        assert!(result.complexity_score > 0);
    }
}
