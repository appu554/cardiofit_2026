//! Core engine module for rule evaluation and decision making

pub mod evaluator;
pub mod orchestrator;
pub mod recipe_executor;
pub mod manifest_generator;  // ⭐ NEW: Intent manifest generator

pub use evaluator::*;
pub use orchestrator::*;
pub use recipe_executor::*;
pub use manifest_generator::*;  // ⭐ NEW: Export manifest generator
