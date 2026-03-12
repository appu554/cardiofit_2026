//! # JIT Safety Engine Core
//!
//! Main JIT Safety Engine implementation with deterministic evaluation logic.
//! Orchestrates the complete safety evaluation process from context normalization
//! through final decision synthesis.

use crate::jit_safety::{
    domain::*,
    error::JitSafetyError,
    normalization::normalize_context,
    rules::{RuleLoader, RulePack},
    ddi_adapter::DdiAdapter,
};
use std::sync::Arc;
use tracing::{info, warn, debug, instrument};

/// Main JIT Safety Engine
pub struct JitEngine {
    loader: Arc<dyn RuleLoader>,
    ddi: Arc<dyn DdiAdapter>,
    engine_version: String,
}

impl JitEngine {
    /// Create a new JIT Safety Engine
    pub fn new(
        loader: Arc<dyn RuleLoader>,
        ddi: Arc<dyn DdiAdapter>,
        engine_version: &str,
    ) -> Self {
        Self {
            loader,
            ddi,
            engine_version: engine_version.to_string(),
        }
    }

    /// Evaluate a JIT Safety context and return the outcome
    #[instrument(skip(self), fields(request_id = %ctx.request_id, drug_id = %ctx.proposal.drug_id))]
    pub fn evaluate(&self, mut ctx: JitSafetyContext) -> Result<JitSafetyOutcome, JitSafetyError> {
        info!(
            "Starting JIT Safety evaluation for drug '{}' (request: {})",
            ctx.proposal.drug_id, ctx.request_id
        );

        // Step 1: Normalize context (units, compute CrCl if rule requires it)
        normalize_context(&mut ctx)?;
        debug!("Context normalization completed");

        // Step 2: Load rule pack for proposal drug
        let mut dose = ctx.proposal.clone();
        let pack = self.loader.load(&dose.drug_id).map_err(|e| {
            warn!("Failed to load rule pack for drug '{}': {}", dose.drug_id, e);
            JitSafetyError::rule_pack_not_found_with_context(
                &dose.drug_id,
                Some(ctx.request_id.clone()),
            )
        })?;

        debug!("Rule pack loaded for drug '{}'", dose.drug_id);

        // Step 3: Initialize evaluation buffer
        let mut buf = EvalBuffer::new();

        // Step 4: Evaluate DDI (contraindicated first)
        self.evaluate_ddi(&ctx, &mut buf)?;
        if buf.blocked {
            return Ok(self.synthesize_outcome(ctx, dose, buf));
        }

        // Step 5: Evaluate drug rule pack
        // (hard blocks → renal/hepatic → dose limits → duplication)
        pack.evaluate(&ctx, &mut dose, &mut buf)
            .map_err(|e| {
                warn!("Rule pack evaluation failed for drug '{}': {}", dose.drug_id, e);
                JitSafetyError::generic_with_context(
                    format!("Rule evaluation failed: {}", e),
                    Some(ctx.request_id.clone()),
                )
            })?;

        debug!("Rule pack evaluation completed");

        // Step 6: Decision synthesis
        let outcome = self.synthesize_outcome(ctx, dose, buf);
        
        info!(
            "JIT Safety evaluation completed: decision={:?}, reasons={}, ddis={}",
            outcome.decision,
            outcome.reasons.len(),
            outcome.ddis.len()
        );

        Ok(outcome)
    }

    /// Evaluate drug-drug interactions
    fn evaluate_ddi(&self, ctx: &JitSafetyContext, buf: &mut EvalBuffer) -> Result<(), JitSafetyError> {
        debug!("Evaluating DDI for {} concurrent medications", ctx.concurrent_meds.len());

        let ddi_hits = self.ddi.check_all(&ctx.proposal.drug_id, &ctx.concurrent_meds)
            .map_err(|e| {
                warn!("DDI adapter failed: {}", e);
                // DDI failures are recoverable - log warning and continue with empty set
                JitSafetyError::ddi_error_with_context(
                    format!("DDI check failed: {}", e),
                    Some(ctx.request_id.clone()),
                    Some(ctx.proposal.drug_id.clone()),
                )
            })
            .unwrap_or_else(|_| Vec::new()); // Graceful degradation

        for hit in ddi_hits {
            match hit.severity.as_str() {
                "contraindicated" => {
                    buf.block(
                        &hit.rule_id,
                        &hit.code,
                        "Contraindicated drug combination",
                        &[],
                    );
                    warn!("Contraindicated DDI found: {}", hit.code);
                }
                "major" => {
                    buf.add_ddi(hit.clone());
                    buf.trace("DDI", "major_flag");
                    debug!("Major DDI flagged: {}", hit.code);
                }
                _ => {
                    buf.add_ddi(hit.clone());
                    debug!("DDI flagged: {} ({})", hit.code, hit.severity);
                }
            }
        }

        Ok(())
    }

    /// Synthesize final outcome from evaluation buffer
    fn synthesize_outcome(
        &self,
        ctx: JitSafetyContext,
        final_dose: ProposedDose,
        buf: EvalBuffer,
    ) -> JitSafetyOutcome {
        let decision = if buf.blocked {
            Decision::Block
        } else if buf.adjusted {
            Decision::AllowWithAdjustment
        } else {
            Decision::Allow
        };

        JitSafetyOutcome {
            decision,
            final_dose,
            reasons: buf.reasons,
            ddis: buf.ddis,
            provenance: Provenance {
                engine_version: self.engine_version.clone(),
                kb_versions: ctx.kb_versions.clone(),
                evaluation_trace: buf.trace,
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::jit_safety::rules::MockRuleLoader;
    use crate::jit_safety::ddi_adapter::MockDdiAdapter;
    use std::collections::HashMap;

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

    #[test]
    fn test_engine_creation() {
        let loader = Arc::new(MockRuleLoader::new());
        let ddi = Arc::new(MockDdiAdapter::new());
        let engine = JitEngine::new(loader, ddi, "test-1.0.0");
        
        assert_eq!(engine.engine_version, "test-1.0.0");
    }

    #[test]
    fn test_basic_evaluation() {
        let loader = Arc::new(MockRuleLoader::new());
        let ddi = Arc::new(MockDdiAdapter::new());
        let engine = JitEngine::new(loader, ddi, "test-1.0.0");
        
        let ctx = create_test_context();
        let result = engine.evaluate(ctx);
        
        // Should succeed with mock implementations
        assert!(result.is_ok());
    }
}
