//! Knowledge base loader for production YAML files

use crate::models::*;
use crate::knowledge::KnowledgeError;
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};
use serde_yaml;
use tracing::{info, warn, error};

/// Knowledge base loader for all 7 knowledge bases
pub struct KnowledgeLoader {
    pub base_path: PathBuf,
}

impl KnowledgeLoader {
    /// Create a new knowledge loader
    pub fn new<P: AsRef<Path>>(base_path: P) -> Self {
        Self {
            base_path: base_path.as_ref().to_path_buf(),
        }
    }

    /// Load the complete knowledge base from YAML files
    pub async fn load_knowledge_base(&self) -> Result<KnowledgeBase, KnowledgeError> {
        info!("Loading complete knowledge base from: {:?}", self.base_path);

        let medication_knowledge_core = self.load_medication_knowledge_core().await?;
        let evidence_repository = self.load_evidence_repository().await?;
        let orb_rules = self.load_orb_rules().await?;
        let context_recipes = self.load_context_recipes().await?;
        let context_recipes = self.load_context_recipes().await?;
        let clinical_recipes = ClinicalRecipeBook {
            metadata: context_recipes.metadata.clone(),
            recipes: std::collections::HashMap::new(), // Convert later if needed
        };
        let formulary_database = FormularyDatabase {
            metadata: Default::default(),
            formularies: std::collections::HashMap::new(),
        };
        let monitoring_database = MonitoringDatabase {
            metadata: Default::default(),
            profiles: std::collections::HashMap::new(),
        };

        let knowledge_base = KnowledgeBase {
            medication_knowledge_core,
            evidence_repository,
            orb_rules,
            context_recipes,
            clinical_recipes,
            formulary_database,
            monitoring_database,
        };

        info!("Knowledge base loaded successfully: {} total items", knowledge_base.total_items());
        Ok(knowledge_base)
    }

    /// Load Medication Knowledge Core (MKC) - TIER 1
    async fn load_medication_knowledge_core(&self) -> Result<MedicationKnowledgeCore, KnowledgeError> {
        let mkc_path = self.base_path.join("tier1-core/mkc");
        info!("Loading MKC from: {:?}", mkc_path);

        let mut medications = HashMap::new();
        
        // Load from subdirectories
        let subdirs = ["anticoagulants", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics"];
        
        for subdir in &subdirs {
            let subdir_path = mkc_path.join(subdir);
            if subdir_path.exists() {
                let subdir_medications = self.load_medications_from_dir(&subdir_path).await?;
                medications.extend(subdir_medications);
            }
        }

        Ok(MedicationKnowledgeCore {
            metadata: KnowledgeMetadata {
                version: "2.0.0".to_string(),
                last_updated: chrono::Utc::now().to_rfc3339(),
                source: "Production YAML Files".to_string(),
                description: "Medication Knowledge Core with production medication data".to_string(),
                total_entries: Some(medications.len()),
            },
            medications,
        })
    }

    /// Load medications from a directory
    async fn load_medications_from_dir(&self, dir_path: &Path) -> Result<HashMap<String, Medication>, KnowledgeError> {
        let mut medications = HashMap::new();

        if !dir_path.exists() {
            return Ok(medications);
        }

        let entries = fs::read_dir(dir_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        for entry in entries {
            let entry = entry.map_err(|e| KnowledgeError::Io(e))?;
            let path = entry.path();
            
            if path.extension().and_then(|s| s.to_str()) == Some("yaml") {
                match self.load_medication_from_file(&path).await {
                    Ok(medication) => {
                        // Index by both RxNorm code and name for flexible lookup
                        if !medication.rxnorm_code.is_empty() {
                            medications.insert(medication.rxnorm_code.clone(), medication.clone());
                        }
                        medications.insert(medication.generic_name.clone(), medication);
                    }
                    Err(e) => {
                        warn!("Failed to load medication from {:?}: {}", path, e);
                    }
                }
            }
        }

        Ok(medications)
    }

    /// Load a single medication from YAML file
    async fn load_medication_from_file(&self, file_path: &Path) -> Result<Medication, KnowledgeError> {
        let content = fs::read_to_string(file_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        // Parse production format
        #[derive(serde::Deserialize)]
        struct ProductionMedication {
            medication: ProductionMedicationData,
        }

        #[derive(serde::Deserialize)]
        struct ProductionMedicationData {
            rxnorm_code: Option<String>,
            name: String,
            therapeutic_class: Option<String>,
            is_high_alert: Option<bool>,
            is_narrow_therapeutic_index: Option<bool>,
            tdm_required: Option<bool>,
            controlled_substance_schedule: Option<i32>,
            pharmacogenomics: Option<Vec<String>>,
            pediatric_formulations: Option<Vec<serde_json::Value>>,
            dosing_parameter: Option<String>,
            renally_cleared: Option<bool>,
        }

        let production_format: ProductionMedication = serde_yaml::from_str(&content)
            .map_err(|e| KnowledgeError::Yaml(e))?;

        let med_data = production_format.medication;

        Ok(Medication {
            rxnorm_code: med_data.rxnorm_code.unwrap_or_default(),
            generic_name: med_data.name,
            brand_names: Vec::new(), // Could be extracted from additional fields
            therapeutic_class: med_data.therapeutic_class.unwrap_or_default(),
            mechanism: String::new(), // Could be added to production format
            indications: Vec::new(), // Could be added to production format
            safety_profile: SafetyProfile {
                requires_monitoring: med_data.tdm_required.unwrap_or(false),
                black_box_warning: None,
                contraindications: Vec::new(),
                drug_interactions: Vec::new(),
            },
        })
    }

    /// Load ORB Rules - TIER 2
    async fn load_orb_rules(&self) -> Result<ORBRuleSet, KnowledgeError> {
        let orb_path = self.base_path.join("tier2-decision/orb-rules");
        info!("Loading ORB rules from: {:?}", orb_path);

        let mut rules = Vec::new();

        // Load from subdirectories
        let subdirs = ["anticoagulation", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics"];

        for subdir in &subdirs {
            let subdir_path = orb_path.join(subdir);
            if subdir_path.exists() {
                let subdir_rules = self.load_rules_from_dir(&subdir_path).await?;
                rules.extend(subdir_rules);
            }
        }

        // Sort rules by priority (highest first)
        rules.sort_by(|a, b| b.priority.cmp(&a.priority));

        Ok(ORBRuleSet {
            metadata: ORBMetadata {
                version: "2.0.0".to_string(),
                last_updated: chrono::Utc::now().to_rfc3339(),
                description: "Production ORB rules for clinical decision support".to_string(),
                total_rules: rules.len(),
            },
            rules,
        })
    }

    /// Load rules from a directory
    async fn load_rules_from_dir(&self, dir_path: &Path) -> Result<Vec<ORBRule>, KnowledgeError> {
        let mut rules = Vec::new();

        if !dir_path.exists() {
            return Ok(rules);
        }

        let entries = fs::read_dir(dir_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        for entry in entries {
            let entry = entry.map_err(|e| KnowledgeError::Io(e))?;
            let path = entry.path();

            if path.extension().and_then(|s| s.to_str()) == Some("yaml") {
                match self.load_rules_from_file(&path).await {
                    Ok(mut file_rules) => {
                        rules.append(&mut file_rules);
                    }
                    Err(e) => {
                        warn!("Failed to load rules from {:?}: {}", path, e);
                    }
                }
            }
        }

        Ok(rules)
    }

    /// Load rules from a single YAML file
    async fn load_rules_from_file(&self, file_path: &Path) -> Result<Vec<ORBRule>, KnowledgeError> {
        let content = fs::read_to_string(file_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        // Production format is an array of rules
        let rules: Vec<ORBRule> = serde_yaml::from_str(&content)
            .map_err(|e| KnowledgeError::Yaml(e))?;

        Ok(rules)
    }
}
