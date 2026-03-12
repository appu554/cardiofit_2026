//! # JIT Safety Rule System
//!
//! TOML-based rule pack loading and evaluation system.
//! Provides drug-specific safety rules with versioning and validation.

use crate::jit_safety::{domain::*, error::JitSafetyError};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::Path;
use tracing::{debug, warn};

/// Rule pack metadata
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RulePackMeta {
    pub drug_id: String,
    pub version: String,
    pub evidence: Vec<String>,
}

/// Hard contraindication rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct HardContraindications {
    pub allergy_codes: Option<Vec<String>>,
    pub pregnancy: Option<bool>,
    pub angioedema_history: Option<bool>,
}

/// Hepatic adjustment rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct HepaticRules {
    pub max_child_pugh: Option<String>, // "A", "B", "C"
}

/// Renal band definition
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RenalBand {
    pub min: f64,
    pub max: f64,
    pub action: String, // "allow", "cap", "block"
    pub max_dose_mg: Option<f64>,
    pub max_dose_mg_per_day: Option<f64>,
    pub min_interval_h: Option<u32>,
    pub reason: Option<String>,
    pub split: Option<String>, // "BID", "TID"
}

/// Renal adjustment rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct RenalRules {
    pub bands: Vec<RenalBand>,
    pub egfr_metric: String, // "egfr_only", "crcl_only", "crcl_or_egfr"
}

/// Dose limit rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct DoseLimits {
    pub absolute_max_mg_per_day: Option<f64>,
    pub absolute_min_mg: Option<f64>,
}

/// Duplicate class rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct DuplicateClass {
    pub class_id: String,
    pub block_combination: bool,
    pub flag_severity: String, // "minor", "moderate", "major"
}

/// QT prolongation rules
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct QtRules {
    pub applies: bool,
    pub threshold_ms: Option<u32>,
}

/// Complete TOML rule pack structure
#[derive(Debug, Clone, Deserialize, Serialize)]
pub struct TomlRulePack {
    pub meta: RulePackMeta,
    pub hard_contraindications: Option<HardContraindications>,
    pub hepatic: Option<HepaticRules>,
    pub renal: Option<RenalRules>,
    pub dose_limits: Option<DoseLimits>,
    pub duplicate_class: Option<DuplicateClass>,
    pub qt_rules: Option<QtRules>,
}

/// Rule pack trait for evaluation
pub trait RulePack: Send + Sync {
    fn drug_id(&self) -> &str;
    fn evaluate(
        &self,
        ctx: &JitSafetyContext,
        dose: &mut ProposedDose,
        buf: &mut EvalBuffer,
    ) -> Result<(), JitSafetyError>;
}

/// TOML-based rule pack implementation
impl RulePack for TomlRulePack {
    fn drug_id(&self) -> &str {
        &self.meta.drug_id
    }

    fn evaluate(
        &self,
        ctx: &JitSafetyContext,
        dose: &mut ProposedDose,
        buf: &mut EvalBuffer,
    ) -> Result<(), JitSafetyError> {
        debug!("Evaluating rule pack for drug '{}'", self.drug_id());

        // Step 1: Hard contraindications
        self.evaluate_hard_contraindications(ctx, buf);
        if buf.blocked {
            return Ok(());
        }

        // Step 2: Renal banding
        self.evaluate_renal_rules(ctx, dose, buf)?;
        if buf.blocked {
            return Ok(());
        }

        // Step 3: Hepatic constraints
        self.evaluate_hepatic_rules(ctx, buf);
        if buf.blocked {
            return Ok(());
        }

        // Step 4: Dose limits
        self.evaluate_dose_limits(dose, buf);

        // Step 5: Duplicate class
        self.evaluate_duplicate_class(ctx, buf);

        // Step 6: QT rules
        self.evaluate_qt_rules(ctx, buf);

        Ok(())
    }
}

impl TomlRulePack {
    /// Evaluate hard contraindications
    fn evaluate_hard_contraindications(&self, ctx: &JitSafetyContext, buf: &mut EvalBuffer) {
        if let Some(ref contraindications) = self.hard_contraindications {
            let rule_id = format!("{}-HARD-CONTRAINDICATIONS", self.meta.drug_id.to_uppercase());

            // Check allergies
            if let Some(ref allergy_codes) = contraindications.allergy_codes {
                for allergy in &ctx.patient.allergies {
                    if allergy_codes.contains(allergy) {
                        buf.block(
                            &rule_id,
                            "ALLERGY_CONTRAINDICATION",
                            &format!("Patient allergic to {}", allergy),
                            &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                        );
                        return;
                    }
                }
            }

            // Check pregnancy
            if contraindications.pregnancy == Some(true) && ctx.patient.pregnancy {
                buf.block(
                    &rule_id,
                    "PREGNANCY_CONTRAINDICATION",
                    "Drug contraindicated in pregnancy",
                    &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                );
                return;
            }

            // Check angioedema history (simplified - would check patient history in real implementation)
            if contraindications.angioedema_history == Some(true) {
                // In real implementation, check patient.conditions for angioedema history
                if ctx.patient.conditions.iter().any(|c| c.contains("angioedema")) {
                    buf.block(
                        &rule_id,
                        "ANGIOEDEMA_HISTORY",
                        "History of angioedema with this drug class",
                        &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                    );
                    return;
                }
            }

            buf.trace(&rule_id, "no_contraindications");
        }
    }

    /// Evaluate renal adjustment rules
    fn evaluate_renal_rules(
        &self,
        ctx: &JitSafetyContext,
        dose: &mut ProposedDose,
        buf: &mut EvalBuffer,
    ) -> Result<(), JitSafetyError> {
        if let Some(ref renal_rules) = self.renal {
            let rule_id = format!("{}-RENAL", self.meta.drug_id.to_uppercase());

            // Select renal metric based on rule preference
            let renal_value = match renal_rules.egfr_metric.as_str() {
                "egfr_only" => ctx.patient.renal.egfr,
                "crcl_only" => ctx.patient.renal.crcl,
                "crcl_or_egfr" => ctx.patient.renal.crcl.or(ctx.patient.renal.egfr),
                _ => {
                    warn!("Unknown egfr_metric: {}", renal_rules.egfr_metric);
                    ctx.patient.renal.egfr
                }
            };

            if let Some(renal_val) = renal_value {
                // Find matching band
                for band in &renal_rules.bands {
                    if renal_val >= band.min && renal_val <= band.max {
                        match band.action.as_str() {
                            "block" => {
                                let reason = band.reason.as_deref()
                                    .unwrap_or("Renal function insufficient for this medication");
                                buf.block(&rule_id, "RENAL_BLOCK", reason, &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>());
                                return Ok(());
                            }
                            "cap" => {
                                let mut adjusted = false;
                                
                                // Apply dose cap
                                if let Some(max_dose) = band.max_dose_mg {
                                    if dose.dose_mg > max_dose {
                                        dose.dose_mg = max_dose;
                                        adjusted = true;
                                    }
                                }

                                // Apply interval adjustment
                                if let Some(min_interval) = band.min_interval_h {
                                    if dose.interval_h < min_interval {
                                        dose.interval_h = min_interval;
                                        adjusted = true;
                                    }
                                }

                                if adjusted {
                                    buf.cap(
                                        &rule_id,
                                        "RENAL_CAP_APPLIED",
                                        &format!("Dose adjusted for renal function ({}={:.1})", 
                                                renal_rules.egfr_metric, renal_val),
                                        &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                                    );
                                } else {
                                    buf.info(
                                        &rule_id,
                                        "RENAL_BAND_CHECKED",
                                        &format!("Renal function acceptable ({}={:.1})", 
                                                renal_rules.egfr_metric, renal_val),
                                        &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                                    );
                                }
                            }
                            "allow" => {
                                buf.info(
                                    &rule_id,
                                    "RENAL_ALLOW",
                                    &format!("Normal renal function ({}={:.1})", 
                                            renal_rules.egfr_metric, renal_val),
                                    &self.meta.evidence.iter().map(|s| s.as_str()).collect::<Vec<_>>(),
                                );
                            }
                            _ => {
                                warn!("Unknown renal band action: {}", band.action);
                            }
                        }
                        break;
                    }
                }
            } else {
                buf.info(
                    &rule_id,
                    "RENAL_DATA_MISSING",
                    "Renal function data not available",
                    &[],
                );
            }
        }

        Ok(())
    }

    /// Evaluate hepatic rules (placeholder)
    fn evaluate_hepatic_rules(&self, _ctx: &JitSafetyContext, _buf: &mut EvalBuffer) {
        // Implementation would check Child-Pugh class against max_child_pugh
        // For now, just trace that hepatic rules were checked
        if self.hepatic.is_some() {
            let rule_id = format!("{}-HEPATIC", self.meta.drug_id.to_uppercase());
            // buf.trace(&rule_id, "hepatic_checked");
        }
    }

    /// Evaluate dose limits (placeholder)
    fn evaluate_dose_limits(&self, _dose: &mut ProposedDose, _buf: &mut EvalBuffer) {
        // Implementation would check absolute_max_mg_per_day and absolute_min_mg
        if self.dose_limits.is_some() {
            let rule_id = format!("{}-DOSE-LIMITS", self.meta.drug_id.to_uppercase());
            // buf.trace(&rule_id, "dose_limits_checked");
        }
    }

    /// Evaluate duplicate class rules (placeholder)
    fn evaluate_duplicate_class(&self, _ctx: &JitSafetyContext, _buf: &mut EvalBuffer) {
        // Implementation would check for therapeutic duplication
        if self.duplicate_class.is_some() {
            let rule_id = format!("{}-DUPLICATE-CLASS", self.meta.drug_id.to_uppercase());
            // buf.trace(&rule_id, "duplicate_class_checked");
        }
    }

    /// Evaluate QT rules (placeholder)
    fn evaluate_qt_rules(&self, _ctx: &JitSafetyContext, _buf: &mut EvalBuffer) {
        // Implementation would check QTc prolongation risk
        if let Some(ref qt_rules) = self.qt_rules {
            if qt_rules.applies {
                let rule_id = format!("{}-QT", self.meta.drug_id.to_uppercase());
                // buf.trace(&rule_id, "qt_checked");
            }
        }
    }
}

/// Rule loader trait
pub trait RuleLoader: Send + Sync {
    fn load(&self, drug_id: &str) -> Result<Box<dyn RulePack>, JitSafetyError>;
}

/// File-based TOML rule loader
pub struct TomlRuleLoader {
    rule_pack_dir: String,
}

impl TomlRuleLoader {
    pub fn new(rule_pack_dir: impl Into<String>) -> Self {
        Self {
            rule_pack_dir: rule_pack_dir.into(),
        }
    }
}

impl RuleLoader for TomlRuleLoader {
    fn load(&self, drug_id: &str) -> Result<Box<dyn RulePack>, JitSafetyError> {
        let file_path = Path::new(&self.rule_pack_dir).join(format!("{}.toml", drug_id));
        
        let content = std::fs::read_to_string(&file_path)
            .map_err(|e| {
                if e.kind() == std::io::ErrorKind::NotFound {
                    JitSafetyError::rule_pack_not_found(drug_id)
                } else {
                    JitSafetyError::Io(e)
                }
            })?;

        let rule_pack: TomlRulePack = toml::from_str(&content)
            .map_err(|e| JitSafetyError::rule_pack_parse(drug_id, e.to_string()))?;

        debug!("Loaded rule pack for drug '{}' version '{}'", drug_id, rule_pack.meta.version);
        
        Ok(Box::new(rule_pack))
    }
}

/// Mock rule loader for testing
#[cfg(test)]
pub struct MockRuleLoader {
    rules: HashMap<String, TomlRulePack>,
}

#[cfg(test)]
impl MockRuleLoader {
    pub fn new() -> Self {
        let mut rules = HashMap::new();
        
        // Add a basic lisinopril rule pack for testing
        let lisinopril_pack = TomlRulePack {
            meta: RulePackMeta {
                drug_id: "lisinopril".to_string(),
                version: "1.0.0".to_string(),
                evidence: vec!["TEST-EVIDENCE".to_string()],
            },
            hard_contraindications: Some(HardContraindications {
                allergy_codes: Some(vec!["ACE_INHIBITOR".to_string()]),
                pregnancy: Some(true),
                angioedema_history: Some(true),
            }),
            renal: Some(RenalRules {
                bands: vec![
                    RenalBand {
                        min: 0.0,
                        max: 29.0,
                        action: "cap".to_string(),
                        max_dose_mg: Some(5.0),
                        max_dose_mg_per_day: None,
                        min_interval_h: Some(24),
                        reason: None,
                        split: None,
                    },
                    RenalBand {
                        min: 30.0,
                        max: 44.0,
                        action: "cap".to_string(),
                        max_dose_mg: Some(10.0),
                        max_dose_mg_per_day: None,
                        min_interval_h: Some(24),
                        reason: None,
                        split: None,
                    },
                    RenalBand {
                        min: 45.0,
                        max: 300.0,
                        action: "allow".to_string(),
                        max_dose_mg: None,
                        max_dose_mg_per_day: None,
                        min_interval_h: None,
                        reason: None,
                        split: None,
                    },
                ],
                egfr_metric: "crcl_or_egfr".to_string(),
            }),
            hepatic: None,
            dose_limits: None,
            duplicate_class: None,
            qt_rules: None,
        };
        
        rules.insert("lisinopril".to_string(), lisinopril_pack);
        
        Self { rules }
    }
}

#[cfg(test)]
impl RuleLoader for MockRuleLoader {
    fn load(&self, drug_id: &str) -> Result<Box<dyn RulePack>, JitSafetyError> {
        self.rules
            .get(drug_id)
            .map(|pack| Box::new(pack.clone()) as Box<dyn RulePack>)
            .ok_or_else(|| JitSafetyError::rule_pack_not_found(drug_id))
    }
}
