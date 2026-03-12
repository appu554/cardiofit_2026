//! Simple test to verify mathematical expression functionality

use flow2_rust_engine::unified_clinical_engine::expression_parser::ExpressionParser;
use flow2_rust_engine::unified_clinical_engine::expression_evaluator::{ExpressionEvaluator, EvaluationContext, EvaluationConfig};
use std::collections::HashMap;

#[test]
fn test_simple_arithmetic() {
    let expr = ExpressionParser::parse("2 + 3 * 4").unwrap();
    assert_eq!(expr.expression, "2 + 3 * 4");
    
    let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
    let context = EvaluationContext {
        variables: HashMap::new(),
        config: EvaluationConfig::default(),
    };
    
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 14.0);
}

#[test]
fn test_variable_expression() {
    let expr = ExpressionParser::parse("weight * 0.5").unwrap();
    assert_eq!(expr.variables, vec!["weight"]);
    
    let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
    let mut variables = HashMap::new();
    variables.insert("weight".to_string(), 70.0);
    
    let context = EvaluationContext {
        variables,
        config: EvaluationConfig::default(),
    };
    
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 35.0);
}

#[test]
fn test_conditional_expression() {
    let expr = ExpressionParser::parse("age > 65 ? 10 : 5").unwrap();
    assert_eq!(expr.variables, vec!["age"]);
    
    let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
    
    // Test with age 45 (should return 5)
    let mut variables = HashMap::new();
    variables.insert("age".to_string(), 45.0);
    let context = EvaluationContext {
        variables,
        config: EvaluationConfig::default(),
    };
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 5.0);
    
    // Test with age 75 (should return 10)
    let mut variables = HashMap::new();
    variables.insert("age".to_string(), 75.0);
    let context = EvaluationContext {
        variables,
        config: EvaluationConfig::default(),
    };
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 10.0);
}

#[test]
fn test_function_call() {
    let expr = ExpressionParser::parse("min(weight, 100)").unwrap();
    assert_eq!(expr.variables, vec!["weight"]);
    assert_eq!(expr.functions, vec!["min"]);
    
    let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
    let mut variables = HashMap::new();
    variables.insert("weight".to_string(), 70.0);
    
    let context = EvaluationContext {
        variables,
        config: EvaluationConfig::default(),
    };
    
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 70.0); // min(70, 100) = 70
}

#[test]
fn test_complex_expression() {
    let expr = ExpressionParser::parse("(weight * 0.5) + (age > 65 ? 10 : 0)").unwrap();
    
    let evaluator = ExpressionEvaluator::new(EvaluationConfig::default());
    let mut variables = HashMap::new();
    variables.insert("weight".to_string(), 70.0);
    variables.insert("age".to_string(), 45.0);
    
    let context = EvaluationContext {
        variables,
        config: EvaluationConfig::default(),
    };
    
    let result = evaluator.evaluate(&expr, &context).unwrap();
    assert_eq!(result.value, 35.0); // (70 * 0.5) + 0 = 35
}
