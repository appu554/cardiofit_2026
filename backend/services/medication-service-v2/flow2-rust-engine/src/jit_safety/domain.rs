//! # JIT Safety Domain Types
//!
//! Core data structures for the JIT Safety Engine.
//! These types define the input/output contracts and internal data models
//! used throughout the safety evaluation process.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Proposed medication dose for safety evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProposedDose {
    pub drug_id: String,
    pub dose_mg: f64,
    pub route: String,       // "po", "iv", "im", "sc"
    pub interval_h: u32,     // q24h => 24, q12h => 12
}

/// Patient context for safety evaluation
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PatientCtx {
    pub age_years: u32,
    pub sex: String,
    pub weight_kg: f64,
    pub height_cm: Option<f64>,
    pub pregnancy: bool,
    pub renal: RenalCtx,
    pub hepatic: HepaticCtx,
    pub qtc_ms: Option<u32>,
    pub allergies: Vec<String>,  // class/drug codes
    pub conditions: Vec<String>,
    pub labs: LabsCtx,
}

/// Renal function context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalCtx {
    pub egfr: Option<f64>,  // eGFR in mL/min/1.73m²
    pub crcl: Option<f64>,  // CrCl in mL/min
}

/// Hepatic function context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticCtx {
    pub child_pugh: Option<char>, // 'A', 'B', 'C'
}

/// Laboratory values context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LabsCtx {
    pub alt: Option<f64>,   // ALT in U/L
    pub ast: Option<f64>,   // AST in U/L
    pub uacr: Option<f64>,  // UACR in mg/g
}

/// Concurrent medication
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ConcurrentMed {
    pub drug_id: String,
    pub class_id: String,
    pub dose_mg: f64,
    pub interval_h: u32,
}

/// Complete JIT Safety evaluation context
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JitSafetyContext {
    pub patient: PatientCtx,
    pub concurrent_meds: Vec<ConcurrentMed>,
    pub proposal: ProposedDose,
    pub kb_versions: HashMap<String, String>,
    pub request_id: String,
}

/// Safety evaluation decision
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum Decision {
    Allow,
    AllowWithAdjustment,
    Block,
}

/// Safety evaluation reason
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Reason {
    pub code: String,       // e.g., "RENAL_CAP_APPLIED"
    pub severity: String,   // info|warn|error|blocker
    pub message: String,
    pub evidence: Vec<String>,
    pub rule_id: String,
}

/// Drug-drug interaction flag
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DdiFlag {
    pub with_drug_id: String,
    pub severity: String,   // minor|moderate|major|contraindicated
    pub action: String,
    pub code: String,
    pub rule_id: String,
}

/// Evaluation provenance and audit trail
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Provenance {
    pub engine_version: String,
    pub kb_versions: HashMap<String, String>,
    pub evaluation_trace: Vec<EvalStep>,
}

/// Individual evaluation step for audit trail
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvalStep {
    pub rule_id: String,
    pub result: String,
}

/// Complete JIT Safety evaluation outcome
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JitSafetyOutcome {
    pub decision: Decision,
    pub final_dose: ProposedDose,
    pub reasons: Vec<Reason>,
    pub ddis: Vec<DdiFlag>,
    pub provenance: Provenance,
}

/// Internal evaluation buffer for building results
#[derive(Debug, Default)]
pub struct EvalBuffer {
    pub reasons: Vec<Reason>,
    pub ddis: Vec<DdiFlag>,
    pub trace: Vec<EvalStep>,
    pub blocked: bool,
    pub adjusted: bool,
}

impl EvalBuffer {
    pub fn new() -> Self {
        Self::default()
    }

    /// Record a blocking safety issue
    pub fn block(&mut self, rule_id: &str, code: &str, msg: &str, evidence: &[&str]) {
        self.blocked = true;
        self.reasons.push(Reason {
            code: code.to_string(),
            severity: "blocker".to_string(),
            message: msg.to_string(),
            evidence: evidence.iter().map(|s| s.to_string()).collect(),
            rule_id: rule_id.to_string(),
        });
        self.trace.push(EvalStep {
            rule_id: rule_id.to_string(),
            result: "block".to_string(),
        });
    }

    /// Record a dose adjustment
    pub fn cap(&mut self, rule_id: &str, code: &str, msg: &str, evidence: &[&str]) {
        self.adjusted = true;
        self.reasons.push(Reason {
            code: code.to_string(),
            severity: "warn".to_string(),
            message: msg.to_string(),
            evidence: evidence.iter().map(|s| s.to_string()).collect(),
            rule_id: rule_id.to_string(),
        });
        self.trace.push(EvalStep {
            rule_id: rule_id.to_string(),
            result: "cap".to_string(),
        });
    }

    /// Record an informational finding
    pub fn info(&mut self, rule_id: &str, code: &str, msg: &str, evidence: &[&str]) {
        self.reasons.push(Reason {
            code: code.to_string(),
            severity: "info".to_string(),
            message: msg.to_string(),
            evidence: evidence.iter().map(|s| s.to_string()).collect(),
            rule_id: rule_id.to_string(),
        });
        self.trace.push(EvalStep {
            rule_id: rule_id.to_string(),
            result: "info".to_string(),
        });
    }

    /// Add a DDI flag
    pub fn add_ddi(&mut self, ddi: DdiFlag) {
        self.ddis.push(ddi);
    }

    /// Record a trace step without adding a reason
    pub fn trace(&mut self, rule_id: &str, result: &str) {
        self.trace.push(EvalStep {
            rule_id: rule_id.to_string(),
            result: result.to_string(),
        });
    }
}
