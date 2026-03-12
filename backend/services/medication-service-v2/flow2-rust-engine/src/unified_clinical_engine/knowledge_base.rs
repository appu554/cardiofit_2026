//! Knowledge Base - Dynamic TOML-based clinical knowledge system
//! 
//! This module implements a comprehensive knowledge base that loads and manages
//! clinical rules, drug information, and safety protocols from TOML files.

use std::collections::HashMap;
use std::sync::Arc;
use std::path::Path;
use serde::{Deserialize, Serialize};
use anyhow::{Result, anyhow};
use tokio::fs;
use tracing::{info, warn, error};

/// Main knowledge base structure
#[derive(Debug, Clone)]
pub struct KnowledgeBase {
    drug_rules: HashMap<String, DrugRulePack>,
    ddi_rules: HashMap<String, DDIRulePack>,
    safety_protocols: HashMap<String, SafetyProtocol>,
    version: String,
    last_updated: chrono::DateTime<chrono::Utc>,
}

/// Drug-specific rule pack containing dose calculation and safety rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugRulePack {
    pub meta: DrugMeta,
    pub dose_calculation: super::rule_engine::DoseCalculationRules, // Use unified structure
    pub safety_verification: SafetyVerificationRules,
    pub monitoring_requirements: Vec<MonitoringRequirement>,
}

/// Drug metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugMeta {
    pub drug_id: String,
    pub generic_name: String,
    pub version: String,
    pub evidence_sources: Vec<String>,
    pub last_updated: String,
    pub clinical_reviewer: String,
    #[serde(default = "default_calculation_strategy")]
    pub calculation_strategy: String,
    #[serde(default)]
    pub therapeutic_class: Vec<String>,
}

fn default_calculation_strategy() -> String {
    "standard_rules".to_string()
}

// REMOVED: Old conflicting DoseCalculationRules struct
// Now using the unified structure from rule_engine.rs

// REMOVED: Old conflicting struct definitions
// Now using the unified structures from rule_engine.rs

/// Safety verification rules - UNIFIED STRUCTURE
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyVerificationRules {
    pub absolute_contraindications: AbsoluteContraindications,
    pub renal_safety: Option<RenalSafety>,
    pub hepatic_safety: Option<HepaticSafety>,
    pub interactions: Option<DrugInteractions>,  // UNIFIED: Use 'interactions' instead of 'drug_interactions'
    pub monitoring_requirements: Vec<MonitoringRequirement>,
}

/// Absolute contraindications
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AbsoluteContraindications {
    pub pregnancy: bool,
    pub breastfeeding: bool,
    pub allergy_classes: Vec<String>,
    pub conditions: Vec<String>,
}

/// Renal safety rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalSafety {
    pub bands: Vec<RenalSafetyBand>,
}

/// Renal safety band
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RenalSafetyBand {
    pub min_egfr: f64,
    pub max_egfr: f64,
    pub action: String,
    pub reason: String,
    pub evidence: Vec<String>,
    pub max_dose_mg_per_day: Option<f64>,
    pub monitoring_required: Option<Vec<String>>,
}

/// Hepatic safety rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct HepaticSafety {
    pub child_pugh_restrictions: Vec<ChildPughRestriction>,
}

/// Child-Pugh restriction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChildPughRestriction {
    pub class: String,
    pub action: String,
    pub dose_reduction_factor: Option<f64>,
}

/// Drug-drug interactions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DrugInteractions {
    pub major: Vec<MajorInteraction>,
    pub moderate: Vec<ModerateInteraction>,
}

/// Major drug interaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MajorInteraction {
    pub interacting_drug_classes: Vec<String>,
    pub action: String,
    pub duration_hours: Option<u32>,
    pub reason: String,
}

/// Moderate drug interaction
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModerateInteraction {
    pub interacting_drug_classes: Vec<String>,
    pub action: String,
    pub monitoring_required: Vec<String>,
    pub reason: String,
}

/// Monitoring requirement
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MonitoringRequirement {
    pub lab_test: String,
    pub frequency: String,
    pub alert_threshold_low: Option<f64>,
    pub alert_threshold_high: Option<f64>,
    pub action_on_alert: String,
    pub reason: Option<String>,
}

/// DDI rule pack for drug-drug interactions
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DDIRulePack {
    pub meta: DDIMeta,
    pub interactions: Vec<DDIRule>,
}

/// DDI metadata
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DDIMeta {
    pub rule_pack_id: String,
    pub version: String,
    pub drug_class: String,
    pub last_updated: String,
}

/// DDI rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DDIRule {
    pub drug_a: String,
    pub drug_b: String,
    pub severity: String,
    pub mechanism: String,
    pub clinical_effect: String,
    pub management: String,
    pub evidence_level: String,
}

/// Safety protocol
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyProtocol {
    pub protocol_id: String,
    pub name: String,
    pub version: String,
    pub rules: Vec<SafetyRule>,
}

/// Safety rule
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SafetyRule {
    pub rule_id: String,
    pub condition: String,
    pub action: String,
    pub severity: String,
    pub rationale: String,
}

impl KnowledgeBase {
    /// Create a new knowledge base by loading from directory
    pub async fn new(kb_path: &str) -> Result<Self> {
        // Normalize: allow passing either the knowledge base root (containing kb_drug_rules, kb_ddi_rules)
        // or a direct path to kb_drug_rules/kb_ddi_rules. If a subdir is passed, use its parent.
        let input_path = Path::new(kb_path);
        let normalized_base = if input_path.ends_with("kb_drug_rules") || input_path.ends_with("kb_ddi_rules") {
            input_path.parent().unwrap_or(input_path).to_path_buf()
        } else {
            input_path.to_path_buf()
        };
        let normalized_base_str = normalized_base.to_string_lossy().to_string();

        info!("🧠 Initializing Knowledge Base from: {}", normalized_base_str);

        let mut kb = KnowledgeBase {
            drug_rules: HashMap::new(),
            ddi_rules: HashMap::new(),
            safety_protocols: HashMap::new(),
            version: "1.0.0".to_string(),
            last_updated: chrono::Utc::now(),
        };

        // Load drug rules
        kb.load_drug_rules(&normalized_base_str).await?;

        // Load DDI rules
        kb.load_ddi_rules(&normalized_base_str).await?;

        // Load safety protocols
        kb.load_safety_protocols(&normalized_base_str).await?;

        info!("✅ Knowledge Base loaded successfully");
        info!("📊 Drug rules: {}", kb.drug_rules.len());
        info!("📊 DDI rules: {}", kb.ddi_rules.len());
        info!("📊 Safety protocols: {}", kb.safety_protocols.len());

        Ok(kb)
    }
    
    /// Load drug rules from TOML files
    async fn load_drug_rules(&mut self, kb_path: &str) -> Result<()> {
        let drug_rules_path = Path::new(kb_path).join("kb_drug_rules");
        
        if !drug_rules_path.exists() {
            warn!("Drug rules directory not found: {:?}", drug_rules_path);
            return Ok(());
        }
        
        let mut entries = fs::read_dir(&drug_rules_path).await?;
        
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();

            if path.extension().and_then(|s| s.to_str()) == Some("toml") {
                info!("Attempting to load drug rule file: {:?}", path);
                match self.load_drug_rule_file(&path).await {
                    Ok(rule_pack) => {
                        let drug_id = rule_pack.meta.drug_id.clone();
                        info!("Successfully loaded drug rule: {}", drug_id);
                        self.drug_rules.insert(drug_id, rule_pack);
                    }
                    Err(e) => {
                        error!("Failed to load drug rule file {:?}: {}", path, e);
                        // Print more detailed error information
                        eprintln!("DETAILED ERROR for {:?}: {}", path, e);
                    }
                }
            }
        }
        
        Ok(())
    }
    
    /// Load a single drug rule file
    async fn load_drug_rule_file(&self, path: &Path) -> Result<DrugRulePack> {
        let content = fs::read_to_string(path).await?;
        let rule_pack: DrugRulePack = toml::from_str(&content)
            .map_err(|e| anyhow!("Failed to parse TOML file {:?}: {}", path, e))?;
        
        Ok(rule_pack)
    }
    
    /// Load DDI rules from TOML files
    async fn load_ddi_rules(&mut self, kb_path: &str) -> Result<()> {
        let ddi_rules_path = Path::new(kb_path).join("kb_ddi_rules");
        
        if !ddi_rules_path.exists() {
            warn!("DDI rules directory not found: {:?}", ddi_rules_path);
            return Ok(());
        }
        
        let mut entries = fs::read_dir(&ddi_rules_path).await?;
        
        while let Some(entry) = entries.next_entry().await? {
            let path = entry.path();
            
            if path.extension().and_then(|s| s.to_str()) == Some("toml") {
                match self.load_ddi_rule_file(&path).await {
                    Ok(ddi_pack) => {
                        let pack_id = ddi_pack.meta.rule_pack_id.clone();
                        self.ddi_rules.insert(pack_id, ddi_pack);
                    }
                    Err(e) => {
                        error!("Failed to load DDI rule file {:?}: {}", path, e);
                    }
                }
            }
        }
        
        Ok(())
    }
    
    /// Load a single DDI rule file
    async fn load_ddi_rule_file(&self, path: &Path) -> Result<DDIRulePack> {
        let content = fs::read_to_string(path).await?;
        let ddi_pack: DDIRulePack = toml::from_str(&content)
            .map_err(|e| anyhow!("Failed to parse DDI TOML file {:?}: {}", path, e))?;
        
        Ok(ddi_pack)
    }
    
    /// Load safety protocols (placeholder for now)
    async fn load_safety_protocols(&mut self, _kb_path: &str) -> Result<()> {
        // Placeholder implementation
        Ok(())
    }
    
    /// Get drug rule pack by drug ID
    pub fn get_drug_rules(&self, drug_id: &str) -> Option<&DrugRulePack> {
        self.drug_rules.get(drug_id)
    }
    
    /// Get DDI rules for a drug class
    pub fn get_ddi_rules(&self, drug_class: &str) -> Option<&DDIRulePack> {
        self.ddi_rules.get(drug_class)
    }
    
    /// Get all drug IDs
    pub fn get_all_drug_ids(&self) -> Vec<String> {
        self.drug_rules.keys().cloned().collect()
    }
    
    /// Get knowledge base statistics
    pub fn get_stats(&self) -> KnowledgeBaseStats {
        KnowledgeBaseStats {
            total_drug_rules: self.drug_rules.len(),
            total_ddi_rules: self.ddi_rules.len(),
            total_safety_protocols: self.safety_protocols.len(),
            version: self.version.clone(),
            last_updated: self.last_updated,
        }
    }
}

/// Knowledge base statistics
#[derive(Debug, Clone, Serialize)]
pub struct KnowledgeBaseStats {
    pub total_drug_rules: usize,
    pub total_ddi_rules: usize,
    pub total_safety_protocols: usize,
    pub version: String,
    pub last_updated: chrono::DateTime<chrono::Utc>,
}
