//! Additional knowledge base loaders for the remaining knowledge bases

use crate::models::*;
use crate::knowledge::KnowledgeError;
use std::collections::HashMap;
use std::fs;
use std::path::Path;
use tracing::{info, warn};

impl super::loader::KnowledgeLoader {
    /// Load Evidence Repository (ER) - TIER 4
    pub async fn load_evidence_repository(&self) -> Result<EvidenceRepository, KnowledgeError> {
        let er_path = self.base_path.join("tier4-evidence/er");
        info!("Loading Evidence Repository from: {:?}", er_path);

        let mut evidence_entries = HashMap::new();
        
        // Load from subdirectories
        let subdirs = ["anticoagulants", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics"];
        
        for subdir in &subdirs {
            let subdir_path = er_path.join(subdir);
            if subdir_path.exists() {
                let subdir_evidence = self.load_evidence_from_dir(&subdir_path).await?;
                evidence_entries.extend(subdir_evidence);
            }
        }

        Ok(EvidenceRepository {
            metadata: KnowledgeMetadata {
                version: "2.0.0".to_string(),
                last_updated: chrono::Utc::now().to_rfc3339(),
                source: "Production YAML Files".to_string(),
                description: "Evidence Repository with clinical guidelines".to_string(),
                total_entries: Some(evidence_entries.len()),
            },
            evidence_entries,
        })
    }

    /// Load evidence entries from a directory
    async fn load_evidence_from_dir(&self, dir_path: &Path) -> Result<HashMap<String, EvidenceEntry>, KnowledgeError> {
        let mut evidence_entries = HashMap::new();

        if !dir_path.exists() {
            return Ok(evidence_entries);
        }

        let entries = fs::read_dir(dir_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        for entry in entries {
            let entry = entry.map_err(|e| KnowledgeError::Io(e))?;
            let path = entry.path();
            
            if path.extension().and_then(|s| s.to_str()) == Some("yaml") {
                match self.load_evidence_from_file(&path).await {
                    Ok(evidence) => {
                        evidence_entries.insert(evidence.id.clone(), evidence);
                    }
                    Err(e) => {
                        warn!("Failed to load evidence from {:?}: {}", path, e);
                    }
                }
            }
        }

        Ok(evidence_entries)
    }

    /// Load evidence from a single YAML file
    async fn load_evidence_from_file(&self, file_path: &Path) -> Result<EvidenceEntry, KnowledgeError> {
        let content = fs::read_to_string(file_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        // Parse production format
        #[derive(serde::Deserialize)]
        struct ProductionEvidence {
            evidence: ProductionEvidenceData,
        }

        #[derive(serde::Deserialize)]
        struct ProductionEvidenceData {
            id: String,
            source: String,
            title: Option<String>,
            publication_date: Option<String>,
            evidence_level: Option<String>,
            recommendations: Option<Vec<String>>,
        }

        let production_format: ProductionEvidence = serde_yaml::from_str(&content)
            .map_err(|e| KnowledgeError::Yaml(e))?;

        let evidence_data = production_format.evidence;
        
        Ok(EvidenceEntry {
            id: evidence_data.id,
            source: evidence_data.source,
            title: evidence_data.title,
            publication_date: evidence_data.publication_date,
            evidence_level: evidence_data.evidence_level,
            recommendations: evidence_data.recommendations.unwrap_or_default(),
        })
    }

    /// Load Context Service Recipe Book (CSRB) - TIER 2
    pub async fn load_context_recipes(&self) -> Result<ContextServiceRecipeBook, KnowledgeError> {
        let csrb_path = self.base_path.join("tier2-decision/context-recipes/fragments");
        info!("Loading Context Recipes from: {:?}", csrb_path);

        let mut recipes = HashMap::new();

        if csrb_path.exists() {
            let context_recipes = self.load_context_recipes_from_dir(&csrb_path).await?;
            recipes.extend(context_recipes);
        }

        Ok(ContextServiceRecipeBook {
            metadata: KnowledgeMetadata {
                version: "2.0.0".to_string(),
                last_updated: chrono::Utc::now().to_rfc3339(),
                source: "Production YAML Files".to_string(),
                description: "Context Service Recipe Book with data assembly recipes".to_string(),
                total_entries: Some(recipes.len()),
            },
            recipes,
        })
    }

    /// Load context recipes from a directory
    async fn load_context_recipes_from_dir(&self, dir_path: &Path) -> Result<HashMap<String, ContextRecipe>, KnowledgeError> {
        let mut recipes = HashMap::new();

        if !dir_path.exists() {
            return Ok(recipes);
        }

        let entries = fs::read_dir(dir_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        for entry in entries {
            let entry = entry.map_err(|e| KnowledgeError::Io(e))?;
            let path = entry.path();
            
            if path.extension().and_then(|s| s.to_str()) == Some("yaml") {
                match self.load_context_recipe_from_file(&path).await {
                    Ok(recipe) => {
                        recipes.insert(recipe.recipe_id.clone(), recipe);
                    }
                    Err(e) => {
                        warn!("Failed to load context recipe from {:?}: {}", path, e);
                    }
                }
            }
        }

        Ok(recipes)
    }

    /// Load context recipe from a single YAML file
    async fn load_context_recipe_from_file(&self, file_path: &Path) -> Result<ContextRecipe, KnowledgeError> {
        let content = fs::read_to_string(file_path)
            .map_err(|e| KnowledgeError::Io(e))?;

        // Parse production fragment format (array of fragments)
        #[derive(serde::Deserialize)]
        struct ProductionFragment {
            fragment_id: String,
            description: Option<String>,
            source_service: Option<String>,
            source_api_endpoint: Option<String>,
            derivation_formula_id: Option<String>,
            dependencies: Option<Vec<String>>,
        }

        let fragments: Vec<ProductionFragment> = serde_yaml::from_str(&content)
            .map_err(|e| KnowledgeError::Yaml(e))?;

        // Convert fragments to context requirements
        let mut base_requirements = Vec::new();
        for fragment in fragments {
            base_requirements.push(ContextRequirement {
                field: fragment.fragment_id,
                source: fragment.source_service.unwrap_or_default(),
                endpoint: fragment.source_api_endpoint,
                required: true,
                max_age_hours: None,
            });
        }

        // Use filename as recipe ID
        let filename = file_path.file_stem()
            .and_then(|s| s.to_str())
            .unwrap_or("unknown")
            .to_string();

        Ok(ContextRecipe {
            recipe_id: filename.clone(),
            version: "2.0.0".to_string(),
            description: format!("Production context fragments from {}", filename),
            base_requirements,
        })
    }
}
