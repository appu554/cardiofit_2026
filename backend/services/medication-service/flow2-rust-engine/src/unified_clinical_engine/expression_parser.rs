//! Mathematical Expression Parser for TOML Rules
//! 
//! This module provides a comprehensive mathematical expression parser that can handle
//! complex formulas with variables, conditional logic, and mathematical operations.
//! 
//! Supported features:
//! - Basic arithmetic: +, -, *, /, ^, %
//! - Comparison operators: ==, !=, <, <=, >, >=
//! - Logical operators: &&, ||, !
//! - Conditional expressions: condition ? value1 : value2
//! - Variable substitution: weight, age, egfr, etc.
//! - Mathematical functions: min, max, abs, sqrt, log, exp
//! - Parentheses for grouping
//! - Constants and literals

use std::collections::HashMap;
use std::fmt;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use tracing::{debug, warn};

/// Mathematical expression that can be evaluated with variable substitution
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct MathExpression {
    pub expression: String,
    pub parsed_ast: ExpressionAST,
    pub variables: Vec<String>,
    pub functions: Vec<String>,
}

/// Abstract Syntax Tree for mathematical expressions
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum ExpressionAST {
    // Literals
    Number(f64),
    Variable(String),
    
    // Binary operations
    Add(Box<ExpressionAST>, Box<ExpressionAST>),
    Subtract(Box<ExpressionAST>, Box<ExpressionAST>),
    Multiply(Box<ExpressionAST>, Box<ExpressionAST>),
    Divide(Box<ExpressionAST>, Box<ExpressionAST>),
    Power(Box<ExpressionAST>, Box<ExpressionAST>),
    Modulo(Box<ExpressionAST>, Box<ExpressionAST>),
    
    // Comparison operations
    Equal(Box<ExpressionAST>, Box<ExpressionAST>),
    NotEqual(Box<ExpressionAST>, Box<ExpressionAST>),
    LessThan(Box<ExpressionAST>, Box<ExpressionAST>),
    LessEqual(Box<ExpressionAST>, Box<ExpressionAST>),
    GreaterThan(Box<ExpressionAST>, Box<ExpressionAST>),
    GreaterEqual(Box<ExpressionAST>, Box<ExpressionAST>),
    
    // Logical operations
    And(Box<ExpressionAST>, Box<ExpressionAST>),
    Or(Box<ExpressionAST>, Box<ExpressionAST>),
    Not(Box<ExpressionAST>),
    
    // Conditional expression (ternary operator)
    Conditional {
        condition: Box<ExpressionAST>,
        true_expr: Box<ExpressionAST>,
        false_expr: Box<ExpressionAST>,
    },
    
    // Function calls
    Function {
        name: String,
        args: Vec<ExpressionAST>,
    },
}

/// Token types for lexical analysis
#[derive(Debug, Clone, PartialEq)]
pub enum Token {
    Number(f64),
    Identifier(String),
    
    // Operators
    Plus,
    Minus,
    Multiply,
    Divide,
    Power,
    Modulo,
    
    // Comparison
    Equal,
    NotEqual,
    LessThan,
    LessEqual,
    GreaterThan,
    GreaterEqual,
    
    // Logical
    And,
    Or,
    Not,
    
    // Conditional
    Question,
    Colon,
    
    // Grouping
    LeftParen,
    RightParen,
    Comma,
    
    // End of input
    EOF,
}

/// Lexer for tokenizing mathematical expressions
pub struct Lexer {
    input: Vec<char>,
    position: usize,
    current_char: Option<char>,
}

impl Lexer {
    pub fn new(input: &str) -> Self {
        let chars: Vec<char> = input.chars().collect();
        let current_char = chars.get(0).copied();
        
        Self {
            input: chars,
            position: 0,
            current_char,
        }
    }
    
    fn advance(&mut self) {
        self.position += 1;
        self.current_char = self.input.get(self.position).copied();
    }
    
    fn peek(&self) -> Option<char> {
        self.input.get(self.position + 1).copied()
    }
    
    fn skip_whitespace(&mut self) {
        while let Some(ch) = self.current_char {
            if ch.is_whitespace() {
                self.advance();
            } else {
                break;
            }
        }
    }
    
    fn read_number(&mut self) -> Result<f64> {
        let mut number_str = String::new();
        
        while let Some(ch) = self.current_char {
            if ch.is_ascii_digit() || ch == '.' {
                number_str.push(ch);
                self.advance();
            } else {
                break;
            }
        }
        
        number_str.parse::<f64>()
            .map_err(|e| anyhow!("Invalid number format: {}", e))
    }
    
    fn read_identifier(&mut self) -> String {
        let mut identifier = String::new();
        
        while let Some(ch) = self.current_char {
            if ch.is_alphanumeric() || ch == '_' {
                identifier.push(ch);
                self.advance();
            } else {
                break;
            }
        }
        
        identifier
    }
    
    pub fn next_token(&mut self) -> Result<Token> {
        loop {
            match self.current_char {
                None => return Ok(Token::EOF),
                Some(ch) if ch.is_whitespace() => {
                    self.skip_whitespace();
                    continue;
                }
                Some(ch) if ch.is_ascii_digit() => {
                    return Ok(Token::Number(self.read_number()?));
                }
                Some(ch) if ch.is_alphabetic() || ch == '_' => {
                    let identifier = self.read_identifier();
                    return Ok(match identifier.as_str() {
                        "and" | "AND" => Token::And,
                        "or" | "OR" => Token::Or,
                        "not" | "NOT" => Token::Not,
                        _ => Token::Identifier(identifier),
                    });
                }
                Some('+') => {
                    self.advance();
                    return Ok(Token::Plus);
                }
                Some('-') => {
                    self.advance();
                    return Ok(Token::Minus);
                }
                Some('*') => {
                    self.advance();
                    return Ok(Token::Multiply);
                }
                Some('/') => {
                    self.advance();
                    return Ok(Token::Divide);
                }
                Some('^') => {
                    self.advance();
                    return Ok(Token::Power);
                }
                Some('%') => {
                    self.advance();
                    return Ok(Token::Modulo);
                }
                Some('=') => {
                    self.advance();
                    if self.current_char == Some('=') {
                        self.advance();
                        return Ok(Token::Equal);
                    } else {
                        return Err(anyhow!("Unexpected character '=' (use '==' for equality)"));
                    }
                }
                Some('!') => {
                    self.advance();
                    if self.current_char == Some('=') {
                        self.advance();
                        return Ok(Token::NotEqual);
                    } else {
                        return Ok(Token::Not);
                    }
                }
                Some('<') => {
                    self.advance();
                    if self.current_char == Some('=') {
                        self.advance();
                        return Ok(Token::LessEqual);
                    } else {
                        return Ok(Token::LessThan);
                    }
                }
                Some('>') => {
                    self.advance();
                    if self.current_char == Some('=') {
                        self.advance();
                        return Ok(Token::GreaterEqual);
                    } else {
                        return Ok(Token::GreaterThan);
                    }
                }
                Some('&') => {
                    self.advance();
                    if self.current_char == Some('&') {
                        self.advance();
                        return Ok(Token::And);
                    } else {
                        return Err(anyhow!("Unexpected character '&' (use '&&' for logical AND)"));
                    }
                }
                Some('|') => {
                    self.advance();
                    if self.current_char == Some('|') {
                        self.advance();
                        return Ok(Token::Or);
                    } else {
                        return Err(anyhow!("Unexpected character '|' (use '||' for logical OR)"));
                    }
                }
                Some('?') => {
                    self.advance();
                    return Ok(Token::Question);
                }
                Some(':') => {
                    self.advance();
                    return Ok(Token::Colon);
                }
                Some('(') => {
                    self.advance();
                    return Ok(Token::LeftParen);
                }
                Some(')') => {
                    self.advance();
                    return Ok(Token::RightParen);
                }
                Some(',') => {
                    self.advance();
                    return Ok(Token::Comma);
                }
                Some(ch) => {
                    return Err(anyhow!("Unexpected character: '{}'", ch));
                }
            }
        }
    }
}

/// Parser for building AST from tokens
pub struct Parser {
    lexer: Lexer,
    current_token: Token,
}

impl Parser {
    pub fn new(input: &str) -> Result<Self> {
        let mut lexer = Lexer::new(input);
        let current_token = lexer.next_token()?;
        
        Ok(Self {
            lexer,
            current_token,
        })
    }
    
    fn advance(&mut self) -> Result<()> {
        self.current_token = self.lexer.next_token()?;
        Ok(())
    }
    
    fn expect_token(&mut self, expected: Token) -> Result<()> {
        if std::mem::discriminant(&self.current_token) == std::mem::discriminant(&expected) {
            self.advance()
        } else {
            Err(anyhow!("Expected {:?}, found {:?}", expected, self.current_token))
        }
    }
    
    pub fn parse(&mut self) -> Result<ExpressionAST> {
        self.parse_conditional()
    }
}

/// Expression parser implementation
impl Parser {
    // Parse conditional expressions (ternary operator): condition ? true_expr : false_expr
    fn parse_conditional(&mut self) -> Result<ExpressionAST> {
        let condition = self.parse_logical_or()?;
        
        if matches!(self.current_token, Token::Question) {
            self.advance()?; // consume '?'
            let true_expr = self.parse_conditional()?;
            self.expect_token(Token::Colon)?;
            let false_expr = self.parse_conditional()?;
            
            Ok(ExpressionAST::Conditional {
                condition: Box::new(condition),
                true_expr: Box::new(true_expr),
                false_expr: Box::new(false_expr),
            })
        } else {
            Ok(condition)
        }
    }
    
    // Parse logical OR expressions
    fn parse_logical_or(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_logical_and()?;
        
        while matches!(self.current_token, Token::Or) {
            self.advance()?;
            let right = self.parse_logical_and()?;
            left = ExpressionAST::Or(Box::new(left), Box::new(right));
        }
        
        Ok(left)
    }
    
    // Parse logical AND expressions
    fn parse_logical_and(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_equality()?;
        
        while matches!(self.current_token, Token::And) {
            self.advance()?;
            let right = self.parse_equality()?;
            left = ExpressionAST::And(Box::new(left), Box::new(right));
        }
        
        Ok(left)
    }
    
    // Parse equality expressions
    fn parse_equality(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_comparison()?;
        
        while matches!(self.current_token, Token::Equal | Token::NotEqual) {
            let op = self.current_token.clone();
            self.advance()?;
            let right = self.parse_comparison()?;
            
            left = match op {
                Token::Equal => ExpressionAST::Equal(Box::new(left), Box::new(right)),
                Token::NotEqual => ExpressionAST::NotEqual(Box::new(left), Box::new(right)),
                _ => unreachable!(),
            };
        }
        
        Ok(left)
    }
    
    // Parse comparison expressions
    fn parse_comparison(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_addition()?;
        
        while matches!(self.current_token, Token::LessThan | Token::LessEqual | Token::GreaterThan | Token::GreaterEqual) {
            let op = self.current_token.clone();
            self.advance()?;
            let right = self.parse_addition()?;
            
            left = match op {
                Token::LessThan => ExpressionAST::LessThan(Box::new(left), Box::new(right)),
                Token::LessEqual => ExpressionAST::LessEqual(Box::new(left), Box::new(right)),
                Token::GreaterThan => ExpressionAST::GreaterThan(Box::new(left), Box::new(right)),
                Token::GreaterEqual => ExpressionAST::GreaterEqual(Box::new(left), Box::new(right)),
                _ => unreachable!(),
            };
        }
        
        Ok(left)
    }
    
    // Parse addition and subtraction
    fn parse_addition(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_multiplication()?;
        
        while matches!(self.current_token, Token::Plus | Token::Minus) {
            let op = self.current_token.clone();
            self.advance()?;
            let right = self.parse_multiplication()?;
            
            left = match op {
                Token::Plus => ExpressionAST::Add(Box::new(left), Box::new(right)),
                Token::Minus => ExpressionAST::Subtract(Box::new(left), Box::new(right)),
                _ => unreachable!(),
            };
        }
        
        Ok(left)
    }
    
    // Parse multiplication, division, and modulo
    fn parse_multiplication(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_power()?;
        
        while matches!(self.current_token, Token::Multiply | Token::Divide | Token::Modulo) {
            let op = self.current_token.clone();
            self.advance()?;
            let right = self.parse_power()?;
            
            left = match op {
                Token::Multiply => ExpressionAST::Multiply(Box::new(left), Box::new(right)),
                Token::Divide => ExpressionAST::Divide(Box::new(left), Box::new(right)),
                Token::Modulo => ExpressionAST::Modulo(Box::new(left), Box::new(right)),
                _ => unreachable!(),
            };
        }
        
        Ok(left)
    }
    
    // Parse power expressions
    fn parse_power(&mut self) -> Result<ExpressionAST> {
        let mut left = self.parse_unary()?;
        
        while matches!(self.current_token, Token::Power) {
            self.advance()?;
            let right = self.parse_unary()?;
            left = ExpressionAST::Power(Box::new(left), Box::new(right));
        }
        
        Ok(left)
    }
    
    // Parse unary expressions
    fn parse_unary(&mut self) -> Result<ExpressionAST> {
        match &self.current_token {
            Token::Minus => {
                self.advance()?;
                let expr = self.parse_unary()?;
                Ok(ExpressionAST::Subtract(
                    Box::new(ExpressionAST::Number(0.0)),
                    Box::new(expr),
                ))
            }
            Token::Not => {
                self.advance()?;
                let expr = self.parse_unary()?;
                Ok(ExpressionAST::Not(Box::new(expr)))
            }
            _ => self.parse_primary(),
        }
    }
    
    // Parse primary expressions (numbers, variables, function calls, parentheses)
    fn parse_primary(&mut self) -> Result<ExpressionAST> {
        match &self.current_token.clone() {
            Token::Number(value) => {
                let value = *value;
                self.advance()?;
                Ok(ExpressionAST::Number(value))
            }
            Token::Identifier(name) => {
                let name = name.clone();
                self.advance()?;
                
                // Check if this is a function call
                if matches!(self.current_token, Token::LeftParen) {
                    self.advance()?; // consume '('
                    let mut args = Vec::new();
                    
                    if !matches!(self.current_token, Token::RightParen) {
                        args.push(self.parse_conditional()?);
                        
                        while matches!(self.current_token, Token::Comma) {
                            self.advance()?; // consume ','
                            args.push(self.parse_conditional()?);
                        }
                    }
                    
                    self.expect_token(Token::RightParen)?;
                    Ok(ExpressionAST::Function { name, args })
                } else {
                    Ok(ExpressionAST::Variable(name))
                }
            }
            Token::LeftParen => {
                self.advance()?; // consume '('
                let expr = self.parse_conditional()?;
                self.expect_token(Token::RightParen)?;
                Ok(expr)
            }
            _ => Err(anyhow!("Unexpected token: {:?}", self.current_token)),
        }
    }
}

/// Main expression parser interface
pub struct ExpressionParser;

impl ExpressionParser {
    /// Parse a mathematical expression string into an AST
    pub fn parse(expression: &str) -> Result<MathExpression> {
        debug!("Parsing mathematical expression: {}", expression);
        
        let mut parser = Parser::new(expression)?;
        let ast = parser.parse()?;
        
        // Extract variables and functions from the AST
        let mut variables = Vec::new();
        let mut functions = Vec::new();
        Self::extract_identifiers(&ast, &mut variables, &mut functions);
        
        // Remove duplicates
        variables.sort();
        variables.dedup();
        functions.sort();
        functions.dedup();
        
        debug!("Parsed expression with {} variables and {} functions", 
               variables.len(), functions.len());
        
        Ok(MathExpression {
            expression: expression.to_string(),
            parsed_ast: ast,
            variables,
            functions,
        })
    }
    
    /// Extract variable and function names from AST
    fn extract_identifiers(ast: &ExpressionAST, variables: &mut Vec<String>, functions: &mut Vec<String>) {
        match ast {
            ExpressionAST::Variable(name) => variables.push(name.clone()),
            ExpressionAST::Function { name, args } => {
                functions.push(name.clone());
                for arg in args {
                    Self::extract_identifiers(arg, variables, functions);
                }
            }
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
                Self::extract_identifiers(left, variables, functions);
                Self::extract_identifiers(right, variables, functions);
            }
            ExpressionAST::Not(expr) => {
                Self::extract_identifiers(expr, variables, functions);
            }
            ExpressionAST::Conditional { condition, true_expr, false_expr } => {
                Self::extract_identifiers(condition, variables, functions);
                Self::extract_identifiers(true_expr, variables, functions);
                Self::extract_identifiers(false_expr, variables, functions);
            }
            ExpressionAST::Number(_) => {} // No identifiers in numbers
        }
    }
}

impl fmt::Display for ExpressionAST {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ExpressionAST::Number(n) => write!(f, "{}", n),
            ExpressionAST::Variable(name) => write!(f, "{}", name),
            ExpressionAST::Add(left, right) => write!(f, "({} + {})", left, right),
            ExpressionAST::Subtract(left, right) => write!(f, "({} - {})", left, right),
            ExpressionAST::Multiply(left, right) => write!(f, "({} * {})", left, right),
            ExpressionAST::Divide(left, right) => write!(f, "({} / {})", left, right),
            ExpressionAST::Power(left, right) => write!(f, "({} ^ {})", left, right),
            ExpressionAST::Modulo(left, right) => write!(f, "({} % {})", left, right),
            ExpressionAST::Equal(left, right) => write!(f, "({} == {})", left, right),
            ExpressionAST::NotEqual(left, right) => write!(f, "({} != {})", left, right),
            ExpressionAST::LessThan(left, right) => write!(f, "({} < {})", left, right),
            ExpressionAST::LessEqual(left, right) => write!(f, "({} <= {})", left, right),
            ExpressionAST::GreaterThan(left, right) => write!(f, "({} > {})", left, right),
            ExpressionAST::GreaterEqual(left, right) => write!(f, "({} >= {})", left, right),
            ExpressionAST::And(left, right) => write!(f, "({} && {})", left, right),
            ExpressionAST::Or(left, right) => write!(f, "({} || {})", left, right),
            ExpressionAST::Not(expr) => write!(f, "(!{})", expr),
            ExpressionAST::Conditional { condition, true_expr, false_expr } => {
                write!(f, "({} ? {} : {})", condition, true_expr, false_expr)
            }
            ExpressionAST::Function { name, args } => {
                write!(f, "{}(", name)?;
                for (i, arg) in args.iter().enumerate() {
                    if i > 0 { write!(f, ", ")?; }
                    write!(f, "{}", arg)?;
                }
                write!(f, ")")
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_simple_arithmetic() {
        let expr = ExpressionParser::parse("2 + 3 * 4").unwrap();
        assert_eq!(expr.expression, "2 + 3 * 4");
        assert!(expr.variables.is_empty());
        assert!(expr.functions.is_empty());
    }

    #[test]
    fn test_variables() {
        let expr = ExpressionParser::parse("weight * 0.5 + age").unwrap();
        assert_eq!(expr.variables, vec!["age", "weight"]);
        assert!(expr.functions.is_empty());
    }

    #[test]
    fn test_conditional() {
        let expr = ExpressionParser::parse("age > 65 ? 10 : 5").unwrap();
        assert_eq!(expr.variables, vec!["age"]);
        assert!(expr.functions.is_empty());
    }

    #[test]
    fn test_functions() {
        let expr = ExpressionParser::parse("min(weight * 0.5, max_dose)").unwrap();
        assert_eq!(expr.variables, vec!["max_dose", "weight"]);
        assert_eq!(expr.functions, vec!["min"]);
    }

    #[test]
    fn test_complex_expression() {
        let expr = ExpressionParser::parse("(weight * 0.5) + (age > 65 ? 10 : 0) + min(egfr / 60, 1.0)").unwrap();
        assert_eq!(expr.variables, vec!["age", "egfr", "weight"]);
        assert_eq!(expr.functions, vec!["min"]);
    }
}
