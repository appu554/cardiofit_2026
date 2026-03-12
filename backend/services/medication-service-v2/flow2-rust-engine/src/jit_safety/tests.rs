//! # JIT Safety Integration Tests
//!
//! Comprehensive integration tests for the JIT Safety Engine.

use super::*;
use crate::jit_safety::{
    engine::JitEngine,
    rules::MockRuleLoader,
    ddi_adapter::MockDdiAdapter,
};
use std::sync::Arc;
use std::collections::HashMap;

/// Create a test JIT Safety context
fn create_test_context() -> JitSafetyContext {
    JitSafetyContext {
        patient: PatientCtx {
            age_years: 65,
            sex: "female".to_string(),
            weight_kg: 70.0,
            height_cm: Some(165.0),
            pregnancy: false,
            renal: RenalCtx {
                egfr: Some(45.0),
                crcl: None,
            },
            hepatic: HepaticCtx {
                child_pugh: Some('A'),
            },
            qtc_ms: Some(420),
            allergies: vec![],
            conditions: vec!["T2DM".to_string()],
            labs: LabsCtx {
                alt: Some(25.0),
                ast: Some(30.0),
                uacr: Some(100.0),
            },
        },
        concurrent_meds: vec![],
        proposal: ProposedDose {
            drug_id: "lisinopril".to_string(),
            dose_mg: 10.0,
            route: "po".to_string(),
            interval_h: 24,
        },
        kb_versions: HashMap::new(),
        request_id: "test-request-123".to_string(),
    }
}

/// Create a test JIT engine
fn create_test_engine() -> JitEngine {
    let loader = Arc::new(MockRuleLoader::new());
    let ddi = Arc::new(MockDdiAdapter::new());
    JitEngine::new(loader, ddi, "test-1.0.0")
}

#[test]
fn test_basic_allow_scenario() {
    let engine = create_test_engine();
    let ctx = create_test_context();
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    assert_eq!(outcome.decision, Decision::Allow);
    assert_eq!(outcome.final_dose.drug_id, "lisinopril");
    assert_eq!(outcome.final_dose.dose_mg, 10.0);
}

#[test]
fn test_renal_adjustment_scenario() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Set eGFR to 35 (should trigger renal cap to 10mg max)
    ctx.patient.renal.egfr = Some(35.0);
    ctx.proposal.dose_mg = 15.0; // Propose higher than cap
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    // Should be adjusted due to renal function
    assert!(matches!(outcome.decision, Decision::AllowWithAdjustment | Decision::Allow));
    
    // Check that dose was capped
    assert!(outcome.final_dose.dose_mg <= 10.0);
    
    // Should have renal-related reason
    assert!(outcome.reasons.iter().any(|r| r.code.contains("RENAL")));
}

#[test]
fn test_allergy_contraindication() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Add ACE inhibitor allergy
    ctx.patient.allergies.push("ACE_INHIBITOR".to_string());
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    assert_eq!(outcome.decision, Decision::Block);
    
    // Should have allergy-related reason
    assert!(outcome.reasons.iter().any(|r| r.code.contains("ALLERGY")));
}

#[test]
fn test_pregnancy_contraindication() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Set pregnancy to true
    ctx.patient.pregnancy = true;
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    assert_eq!(outcome.decision, Decision::Block);
    
    // Should have pregnancy-related reason
    assert!(outcome.reasons.iter().any(|r| r.code.contains("PREGNANCY")));
}

#[test]
fn test_ddi_major_interaction() {
    let loader = Arc::new(MockRuleLoader::new());
    
    // Create DDI adapter with major interaction
    let ddi_flag = DdiFlag {
        with_drug_id: "losartan".to_string(),
        severity: "major".to_string(),
        action: "monitor closely".to_string(),
        code: "DDI-ACEI-ARB".to_string(),
        rule_id: "DDI-RAAS-001".to_string(),
    };
    let ddi = Arc::new(MockDdiAdapter::with_interactions(vec![ddi_flag]));
    
    let engine = JitEngine::new(loader, ddi, "test-1.0.0");
    
    let mut ctx = create_test_context();
    ctx.concurrent_meds.push(ConcurrentMed {
        drug_id: "losartan".to_string(),
        class_id: "ARB".to_string(),
        dose_mg: 50.0,
        interval_h: 24,
    });
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    // Major DDI should not block, just flag
    assert_ne!(outcome.decision, Decision::Block);
    
    // Should have DDI flag
    assert_eq!(outcome.ddis.len(), 1);
    assert_eq!(outcome.ddis[0].code, "DDI-ACEI-ARB");
    assert_eq!(outcome.ddis[0].severity, "major");
}

#[test]
fn test_ddi_contraindicated_interaction() {
    let loader = Arc::new(MockRuleLoader::new());
    
    // Create DDI adapter with contraindicated interaction
    let ddi_flag = DdiFlag {
        with_drug_id: "sacubitril_valsartan".to_string(),
        severity: "contraindicated".to_string(),
        action: "block".to_string(),
        code: "DDI-ACEI-ARNI".to_string(),
        rule_id: "DDI-RAAS-002".to_string(),
    };
    let ddi = Arc::new(MockDdiAdapter::with_interactions(vec![ddi_flag]));
    
    let engine = JitEngine::new(loader, ddi, "test-1.0.0");
    
    let mut ctx = create_test_context();
    ctx.concurrent_meds.push(ConcurrentMed {
        drug_id: "sacubitril_valsartan".to_string(),
        class_id: "ARNI".to_string(),
        dose_mg: 49.0,
        interval_h: 12,
    });
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    // Contraindicated DDI should block
    assert_eq!(outcome.decision, Decision::Block);
    
    // Should have blocking reason
    assert!(outcome.reasons.iter().any(|r| r.severity == "blocker"));
}

#[test]
fn test_invalid_input_handling() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Set invalid dose
    ctx.proposal.dose_mg = -5.0;
    
    let result = engine.evaluate(ctx);
    assert!(result.is_err());
    
    // Should be input validation error
    assert!(matches!(result.unwrap_err(), JitSafetyError::InputValidation { .. }));
}

#[test]
fn test_missing_rule_pack() {
    let loader = Arc::new(MockRuleLoader::new());
    let ddi = Arc::new(MockDdiAdapter::new());
    let engine = JitEngine::new(loader, ddi, "test-1.0.0");
    
    let mut ctx = create_test_context();
    ctx.proposal.drug_id = "unknown_drug".to_string();
    
    let result = engine.evaluate(ctx);
    assert!(result.is_err());
    
    // Should be rule pack not found error
    assert!(matches!(result.unwrap_err(), JitSafetyError::RulePackNotFound { .. }));
}

#[test]
fn test_provenance_tracking() {
    let engine = create_test_engine();
    let ctx = create_test_context();
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    
    // Should have provenance information
    assert_eq!(outcome.provenance.engine_version, "test-1.0.0");
    assert!(!outcome.provenance.evaluation_trace.is_empty());
    
    // Should have trace steps
    let trace_rule_ids: Vec<&String> = outcome.provenance.evaluation_trace
        .iter()
        .map(|step| &step.rule_id)
        .collect();
    
    // Should include hard contraindications check
    assert!(trace_rule_ids.iter().any(|id| id.contains("HARD-CONTRAINDICATIONS")));
}

#[test]
fn test_context_normalization() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Start without CrCl
    assert!(ctx.patient.renal.crcl.is_none());
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    // Context should have been normalized during evaluation
    // (Note: This test verifies that normalization doesn't fail,
    // but the original context is consumed by evaluate())
}

#[test]
fn test_multiple_concurrent_medications() {
    let engine = create_test_engine();
    let mut ctx = create_test_context();
    
    // Add multiple concurrent medications
    ctx.concurrent_meds = vec![
        ConcurrentMed {
            drug_id: "metformin".to_string(),
            class_id: "BIGUANIDE".to_string(),
            dose_mg: 1000.0,
            interval_h: 12,
        },
        ConcurrentMed {
            drug_id: "atorvastatin".to_string(),
            class_id: "STATIN".to_string(),
            dose_mg: 20.0,
            interval_h: 24,
        },
    ];
    
    let result = engine.evaluate(ctx);
    assert!(result.is_ok());
    
    let outcome = result.unwrap();
    // Should complete successfully with multiple concurrent meds
    assert!(matches!(outcome.decision, Decision::Allow | Decision::AllowWithAdjustment));
}
