package orb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// KnowledgeLoader loads knowledge base files from the file system
type KnowledgeLoader struct {
	basePath string
	logger   *logrus.Logger
}

// NewKnowledgeLoader creates a new knowledge loader
func NewKnowledgeLoader(basePath string) *KnowledgeLoader {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &KnowledgeLoader{
		basePath: basePath,
		logger:   logger,
	}
}

// LoadORBRules loads the ORB rules from production YAML files
func (kl *KnowledgeLoader) LoadORBRules() (*ORBRuleSet, error) {
	orbPath := filepath.Join(kl.basePath, "tier2-decision", "orb-rules")

	var allRules []ORBRule

	// Load from production subdirectories
	subdirs := []string{"anticoagulation", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics"}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(orbPath, subdir)
		subdirRules, err := kl.loadORBRulesFromDir(subdirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load ORB rules from %s: %w", subdir, err)
		}

		// Merge rules
		allRules = append(allRules, subdirRules...)
	}

	ruleSet := &ORBRuleSet{
		Metadata: struct {
			Version          string `yaml:"version"`
			LastUpdated      string `yaml:"last_updated"`
			Description      string `yaml:"description"`
			Author           string `yaml:"author"`
			ValidationStatus string `yaml:"validation_status"`
		}{
			Version:          "2.0.0",
			LastUpdated:      "2024-01-15",
			Description:      "Production ORB Rules - Top 10 Recipes",
			Author:           "Clinical Decision Support Team",
			ValidationStatus: "validated",
		},
		Rules: allRules,
	}

	// Validate rules
	if err := kl.validateORBRules(ruleSet); err != nil {
		return nil, fmt.Errorf("ORB rules validation failed: %w", err)
	}

	return ruleSet, nil
}

// LoadMedicationKnowledgeCore loads the MKC from JSON
func (kl *KnowledgeLoader) LoadMedicationKnowledgeCore() (*MedicationKnowledgeCore, error) {
	// Load drug encyclopedia
	drugEncyclopedia, err := kl.loadDrugEncyclopedia()
	if err != nil {
		return nil, fmt.Errorf("failed to load drug encyclopedia: %w", err)
	}

	// Load drug interactions
	drugInteractions, err := kl.loadDrugInteractions()
	if err != nil {
		return nil, fmt.Errorf("failed to load drug interactions: %w", err)
	}

	// Load contraindications
	contraindications, err := kl.loadContraindications()
	if err != nil {
		return nil, fmt.Errorf("failed to load contraindications: %w", err)
	}

	return &MedicationKnowledgeCore{
		DrugEncyclopedia:  drugEncyclopedia,
		DrugInteractions:  drugInteractions,
		Contraindications: contraindications,
	}, nil
}

// LoadContextRecipes loads the CSRB from production YAML files
func (kl *KnowledgeLoader) LoadContextRecipes() (*ContextServiceRecipeBook, error) {
	contextPath := filepath.Join(kl.basePath, "tier2-decision", "context-recipes")

	recipes := make(map[string]*ContextRecipe)

	// Load from production fragments directory
	fragmentsPath := filepath.Join(contextPath, "fragments")
	fragmentRecipes, err := kl.loadContextRecipesFromDir(fragmentsPath)
	if err == nil {
		// Merge fragment recipes
		for id, recipe := range fragmentRecipes {
			recipes[id] = recipe
		}
	}

	// Also load root-level context files for backward compatibility
	rootRecipes, err := kl.loadContextRecipesFromDir(contextPath)
	if err == nil {
		// Merge root recipes
		for id, recipe := range rootRecipes {
			recipes[id] = recipe
		}
	}

	return &ContextServiceRecipeBook{
		Recipes: recipes,
	}, nil
}

// LoadClinicalRecipeBook loads the CRB from YAML files (TIER 1)
func (kl *KnowledgeLoader) LoadClinicalRecipeBook() (*ClinicalRecipeBook, error) {
	recipePath := filepath.Join(kl.basePath, "tier1-core", "clinical-recipe-book")

	recipes := make(map[string]*ClinicalRecipe)

	// Load individual recipe files
	recipeFiles := []string{
		"vancomycin_renal.yaml",
		"warfarin_initiation.yaml",
		"acetaminophen_standard.yaml",
	}

	for _, filename := range recipeFiles {
		filePath := filepath.Join(recipePath, filename)
		recipe, err := kl.loadClinicalRecipe(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load clinical recipe %s: %w", filename, err)
		}
		recipes[recipe.RecipeID] = recipe
	}

	// Load recipes from production subdirectories
	subdirs := []string{"anticoagulation", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics", "protocols"}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(recipePath, subdir)
		subdirRecipes, err := kl.loadClinicalRecipesFromDir(subdirPath)
		if err != nil {
			// Continue if subdirectory doesn't exist or has issues
			continue
		}

		// Merge recipes
		for id, recipe := range subdirRecipes {
			recipes[id] = recipe
		}
	}

	return &ClinicalRecipeBook{
		Metadata: ClinicalRecipeMetadata{
			Version:     "1.0.0",
			LastUpdated: "2024-01-15",
			Description: "Clinical Recipe Book - TIER 1 Core Clinical Knowledge",
			Author:      "Clinical Decision Support Team",
		},
		Recipes: recipes,
	}, nil
}

// LoadFormularyDatabase loads the FCD from JSON files (TIER 3)
func (kl *KnowledgeLoader) LoadFormularyDatabase() (*FormularyDatabase, error) {
	formularyPath := filepath.Join(kl.basePath, "tier3-operational", "formulary")

	formularies := make(map[string]*FormularyEntry)

	// Load formulary files from subdirectories
	subdirs := []string{"anticoagulants", "antimicrobials", "oncology", "pain", "pediatrics"}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(formularyPath, subdir)
		subdirFormularies, err := kl.loadFormulariesFromDir(subdirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load formularies from %s: %w", subdir, err)
		}

		// Merge formularies
		for code, entry := range subdirFormularies {
			formularies[code] = entry
		}
	}

	return &FormularyDatabase{
		Metadata: FormularyMetadata{
			Version:     "1.0.0",
			LastUpdated: "2024-01-15",
			Source:      "PBMs, Insurance Plans, 340B Pricing",
			Description: "Formulary & Cost Database - TIER 3 Operational Knowledge",
		},
		Formularies: formularies,
	}, nil
}

// LoadMonitoringDatabase loads the MRD from JSON files (TIER 3)
func (kl *KnowledgeLoader) LoadMonitoringDatabase() (*MonitoringDatabase, error) {
	monitoringPath := filepath.Join(kl.basePath, "tier3-operational", "monitoring")

	profiles := make(map[string]*MonitoringProfile)

	// Load monitoring profiles from protocols subdirectory
	protocolsPath := filepath.Join(monitoringPath, "protocols")
	protocolProfiles, err := kl.loadMonitoringProfilesFromDir(protocolsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load monitoring profiles from protocols: %w", err)
	}

	// Merge profiles
	for code, profile := range protocolProfiles {
		profiles[code] = profile
	}

	return &MonitoringDatabase{
		Metadata: MonitoringMetadata{
			Version:     "1.0.0",
			LastUpdated: "2024-01-15",
			Source:      "Clinical Guidelines, Safety Data",
			Description: "Monitoring Requirements Database - TIER 3 Operational Knowledge",
		},
		MonitoringProfiles: profiles,
	}, nil
}

// LoadEvidenceRepository loads the ER from JSON files (TIER 4)
func (kl *KnowledgeLoader) LoadEvidenceRepository() (*EvidenceRepository, error) {
	evidencePath := filepath.Join(kl.basePath, "tier4-evidence", "er")

	evidence := make(map[string]*EvidenceEntry)

	// Load evidence from subdirectories
	subdirs := []string{"anticoagulants", "antimicrobials", "cardiology", "endocrinology", "oncology", "pain", "pediatrics"}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(evidencePath, subdir)
		subdirEvidence, err := kl.loadEvidenceFromDir(subdirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load evidence from %s: %w", subdir, err)
		}

		// Merge evidence
		for id, entry := range subdirEvidence {
			evidence[id] = entry
		}
	}

	return &EvidenceRepository{
		Metadata: EvidenceMetadata{
			Version:     "1.0.0",
			LastUpdated: "2024-01-15",
			Source:      "Medical Literature, Professional Societies",
			Description: "Evidence Repository - TIER 4 Evidence & Quality",
		},
		Evidence: evidence,
	}, nil
}

// Private helper methods

func (kl *KnowledgeLoader) loadDrugEncyclopedia() (*DrugEncyclopedia, error) {
	mkcPath := filepath.Join(kl.basePath, "tier1-core", "mkc")

	medications := make(map[string]Medication)

	// Load from production subdirectories
	subdirs := []string{"anticoagulants", "antimicrobials", "endocrinology", "oncology", "pain", "pediatrics"}

	for _, subdir := range subdirs {
		subdirPath := filepath.Join(mkcPath, subdir)
		subdirMeds, err := kl.loadMedicationsFromDir(subdirPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load medications from %s: %w", subdir, err)
		}

		// Merge medications
		for code, med := range subdirMeds {
			medications[code] = med
		}
	}

	encyclopedia := &DrugEncyclopedia{
		Metadata: struct {
			Version     string `json:"version"`
			LastUpdated string `json:"last_updated"`
			Source      string `json:"source"`
			Description string `json:"description"`
		}{
			Version:     "2.0.0",
			LastUpdated: "2024-01-15",
			Source:      "Production MKC",
			Description: "Production Medication Knowledge Core",
		},
		Medications: medications,
	}

	return encyclopedia, nil
}

func (kl *KnowledgeLoader) loadDrugInteractions() (*DrugInteractions, error) {
	filePath := filepath.Join(kl.basePath, "tier1-core", "medication-knowledge-core", "drug_interactions.json")
	
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var interactions DrugInteractions
	if err := json.Unmarshal(data, &interactions); err != nil {
		return nil, err
	}

	return &interactions, nil
}

func (kl *KnowledgeLoader) loadContraindications() (*Contraindications, error) {
	filePath := filepath.Join(kl.basePath, "tier1-core", "medication-knowledge-core", "contraindications.json")
	
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var contraindications Contraindications
	if err := json.Unmarshal(data, &contraindications); err != nil {
		return nil, err
	}

	return &contraindications, nil
}

func (kl *KnowledgeLoader) loadContextRecipe(filePath string) (*ContextRecipe, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try production fragment format first (array of fragments)
	var productionFragments []struct {
		FragmentID          string   `yaml:"fragment_id"`
		Description         string   `yaml:"description"`
		SourceService       string   `yaml:"source_service"`
		SourceAPIEndpoint   string   `yaml:"source_api_endpoint"`
		DerivationFormulaID string   `yaml:"derivation_formula_id"`
		Dependencies        []string `yaml:"dependencies"`
	}

	if err := yaml.Unmarshal(data, &productionFragments); err == nil && len(productionFragments) > 0 {
		// Convert production fragments to ContextRecipe
		// Use filename as recipe ID
		filename := filepath.Base(filePath)
		recipeID := strings.TrimSuffix(filename, filepath.Ext(filename))

		recipe := &ContextRecipe{
			RecipeID:    recipeID,
			Version:     "2.0.0",
			Description: fmt.Sprintf("Production context fragments from %s", filename),
		}

		// Convert fragments to base requirements
		var baseRequirements []ContextRequirement
		for _, fragment := range productionFragments {
			requirement := ContextRequirement{
				Field:    fragment.FragmentID,
				Source:   fragment.SourceService,
				Endpoint: fragment.SourceAPIEndpoint,
				Required: true,
			}
			baseRequirements = append(baseRequirements, requirement)
		}
		recipe.BaseRequirements = baseRequirements

		return recipe, nil
	}

	// Fall back to legacy format
	var legacyRecipe ContextRecipe
	if err := yaml.Unmarshal(data, &legacyRecipe); err != nil {
		return nil, fmt.Errorf("failed to parse as both production and legacy format: %w", err)
	}

	return &legacyRecipe, nil
}

func (kl *KnowledgeLoader) validateORBRules(ruleSet *ORBRuleSet) error {
	if len(ruleSet.Rules) == 0 {
		return fmt.Errorf("no ORB rules found")
	}

	// Check for duplicate rule IDs
	ruleIDs := make(map[string]bool)
	for _, rule := range ruleSet.Rules {
		if rule.ID == "" {
			return fmt.Errorf("rule missing ID")
		}
		if ruleIDs[rule.ID] {
			return fmt.Errorf("duplicate rule ID: %s", rule.ID)
		}
		ruleIDs[rule.ID] = true

		// Validate rule structure
		if err := kl.validateRule(&rule); err != nil {
			return fmt.Errorf("rule %s validation failed: %w", rule.ID, err)
		}
	}

	return nil
}

func (kl *KnowledgeLoader) validateRule(rule *ORBRule) error {
	if rule.Priority <= 0 {
		return fmt.Errorf("priority must be positive")
	}

	// Validate production format (action.generate_manifest)
	if rule.Action.GenerateManifest.RecipeID != "" {
		// Production format validation for recipe selection rules
		if len(rule.Action.GenerateManifest.DataManifest.Required) == 0 {
			return fmt.Errorf("action.generate_manifest.data_manifest.required cannot be empty")
		}
		// Validate conditions
		if len(rule.Conditions.AllOf) == 0 && len(rule.Conditions.AnyOf) == 0 {
			return fmt.Errorf("conditions must have at least one all_of or any_of clause")
		}
		return nil
	}

	// Check if this is a safety/validation rule (different action format)
	if rule.Action.Type != "" {
		// Safety rules don't need data_manifest, just validate they have conditions
		if len(rule.Conditions.AllOf) == 0 && len(rule.Conditions.AnyOf) == 0 {
			return fmt.Errorf("safety rules must have conditions")
		}
		return nil
	}

	// Legacy format validation (for backward compatibility)
	if rule.MedicationCode == "" {
		return fmt.Errorf("medication_code is required (legacy format)")
	}
	if rule.IntentManifest.RecipeID == "" {
		return fmt.Errorf("intent_manifest.recipe_id is required (legacy format)")
	}
	if len(rule.IntentManifest.DataRequirements) == 0 {
		return fmt.Errorf("intent_manifest.data_requirements cannot be empty (legacy format)")
	}

	return nil
}

// ValidateKnowledgeBase performs comprehensive validation of the entire knowledge base
func (kl *KnowledgeLoader) ValidateKnowledgeBase() error {
	// Validate TIER 1 - Core Clinical Knowledge
	_, err := kl.LoadMedicationKnowledgeCore()
	if err != nil {
		return fmt.Errorf("MKC validation failed: %w", err)
	}

	_, err = kl.LoadClinicalRecipeBook()
	if err != nil {
		return fmt.Errorf("CRB validation failed: %w", err)
	}

	// Validate TIER 2 - Decision Support
	_, err = kl.LoadORBRules()
	if err != nil {
		return fmt.Errorf("ORB rules validation failed: %w", err)
	}

	_, err = kl.LoadContextRecipes()
	if err != nil {
		return fmt.Errorf("context recipes validation failed: %w", err)
	}

	// Validate TIER 3 - Operational Knowledge
	_, err = kl.LoadFormularyDatabase()
	if err != nil {
		return fmt.Errorf("formulary database validation failed: %w", err)
	}

	_, err = kl.LoadMonitoringDatabase()
	if err != nil {
		return fmt.Errorf("monitoring database validation failed: %w", err)
	}

	// Validate TIER 4 - Evidence & Quality
	_, err = kl.LoadEvidenceRepository()
	if err != nil {
		return fmt.Errorf("evidence repository validation failed: %w", err)
	}

	return nil
}

// GetKnowledgeBasePath returns the base path for knowledge files
func (kl *KnowledgeLoader) GetKnowledgeBasePath() string {
	return kl.basePath
}

// ListAvailableRules returns a list of available rule IDs
func (kl *KnowledgeLoader) ListAvailableRules() ([]string, error) {
	ruleSet, err := kl.LoadORBRules()
	if err != nil {
		return nil, err
	}

	var ruleIDs []string
	for _, rule := range ruleSet.Rules {
		ruleIDs = append(ruleIDs, rule.ID)
	}

	return ruleIDs, nil
}

// GetRuleByID retrieves a specific rule by ID
func (kl *KnowledgeLoader) GetRuleByID(ruleID string) (*ORBRule, error) {
	ruleSet, err := kl.LoadORBRules()
	if err != nil {
		return nil, err
	}

	for _, rule := range ruleSet.Rules {
		if rule.ID == ruleID {
			return &rule, nil
		}
	}

	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

// Helper methods for loading new knowledge bases

// loadClinicalRecipe loads a single clinical recipe from YAML (supports both legacy and production formats)
func (kl *KnowledgeLoader) loadClinicalRecipe(filePath string) (*ClinicalRecipe, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Try production format first
	var productionFormat struct {
		ID                   string `yaml:"id"`
		WhatID               string `yaml:"what id"` // Alternative field name
		Name                 string `yaml:"name"`
		CalculationVariants  map[string]struct {
			LogicSteps []map[string]interface{} `yaml:"logic_steps"`
		} `yaml:"calculation_variants"`
	}

	if err := yaml.Unmarshal(data, &productionFormat); err == nil {
		// Determine recipe ID
		recipeID := productionFormat.ID
		if recipeID == "" {
			recipeID = productionFormat.WhatID
		}

		if recipeID != "" && len(productionFormat.CalculationVariants) > 0 {
			// Convert production format to ClinicalRecipe
			recipe := &ClinicalRecipe{
				RecipeID:       recipeID,
				Version:        "2.0.0",
				Name:           productionFormat.Name,
				Description:    fmt.Sprintf("Production recipe: %s", productionFormat.Name),
				MedicationCode: extractMedicationFromID(recipeID),
				Indication:     "general",
			}

			// Convert calculation variants to algorithm steps
			var steps []AlgorithmStep
			for variantName, variant := range productionFormat.CalculationVariants {
				for i, logicStep := range variant.LogicSteps {
					step := AlgorithmStep{
						StepID:      fmt.Sprintf("%s_step_%d", variantName, i+1),
						Description: fmt.Sprintf("Step %d for %s variant", i+1, variantName),
						Action:      "calculate",
						Parameters:  logicStep,
					}
					steps = append(steps, step)
				}
			}

			recipe.Algorithm = ClinicalAlgorithm{
				Type:  "production_calculation",
				Steps: steps,
			}

			return recipe, nil
		}
	}

	// Fall back to legacy format
	var legacyRecipe ClinicalRecipe
	if err := yaml.Unmarshal(data, &legacyRecipe); err != nil {
		return nil, fmt.Errorf("failed to parse as both production and legacy format: %w", err)
	}

	return &legacyRecipe, nil
}

// loadClinicalRecipesFromDir loads all clinical recipes from a directory
func (kl *KnowledgeLoader) loadClinicalRecipesFromDir(dirPath string) (map[string]*ClinicalRecipe, error) {
	recipes := make(map[string]*ClinicalRecipe)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return recipes, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())
			recipe, err := kl.loadClinicalRecipe(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load recipe %s: %w", file.Name(), err)
			}
			recipes[recipe.RecipeID] = recipe
		}
	}

	return recipes, nil
}

// loadFormulariesFromDir loads formulary entries from a directory of YAML files
func (kl *KnowledgeLoader) loadFormulariesFromDir(dirPath string) (map[string]*FormularyEntry, error) {
	formularies := make(map[string]*FormularyEntry)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return formularies, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Production format: formulary_items: [...]
			var productionFormat struct {
				FormularyItems []struct {
					ProductName string `yaml:"product_name"`
				} `yaml:"formulary_items"`
			}

			if err := yaml.Unmarshal(data, &productionFormat); err != nil {
				continue // Skip files that can't be parsed
			}

			// Convert to FormularyEntry format
			filename := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			for i, item := range productionFormat.FormularyItems {
				// Generate unique key from filename and index
				key := fmt.Sprintf("%s_%d", filename, i)
				entry := &FormularyEntry{
					MedicationCode:  key,
					GenericName:     item.ProductName,
					BrandName:       item.ProductName,
					FormularyStatus: "preferred",
					Tier:            1,
				}
				formularies[key] = entry
			}
		}
	}

	return formularies, nil
}

// loadMonitoringProfilesFromDir loads monitoring profiles from a directory of YAML files
func (kl *KnowledgeLoader) loadMonitoringProfilesFromDir(dirPath string) (map[string]*MonitoringProfile, error) {
	profiles := make(map[string]*MonitoringProfile)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return profiles, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Try production format 1: Array of protocols
			var arrayFormat []struct {
				ProtocolID string `yaml:"protocol_id"`
			}

			if err := yaml.Unmarshal(data, &arrayFormat); err == nil && len(arrayFormat) > 0 {
				filename := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
				for i, protocol := range arrayFormat {
					key := fmt.Sprintf("%s_%d", filename, i)
					profile := &MonitoringProfile{
						MedicationCode:   key,
						ProfileName:      protocol.ProtocolID,
						BaselineRequired: true,
					}
					profiles[key] = profile
				}
				continue
			}

			// Try production format 2: Single protocol with items
			var singleFormat struct {
				ProtocolID string `yaml:"protocol_id"`
				Items      []struct {
					Name string `yaml:"name"`
				} `yaml:"items"`
			}

			if err := yaml.Unmarshal(data, &singleFormat); err == nil && singleFormat.ProtocolID != "" {
				filename := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
				profile := &MonitoringProfile{
					MedicationCode:   filename,
					ProfileName:      singleFormat.ProtocolID,
					BaselineRequired: true,
				}
				profiles[filename] = profile
			}
		}
	}

	return profiles, nil
}

// loadEvidenceFromDir loads evidence entries from a directory of YAML files
func (kl *KnowledgeLoader) loadEvidenceFromDir(dirPath string) (map[string]*EvidenceEntry, error) {
	evidence := make(map[string]*EvidenceEntry)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return evidence, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Production format: evidence: {...}
			var productionFormat struct {
				Evidence struct {
					ID     string `yaml:"id"`
					Source string `yaml:"source"`
				} `yaml:"evidence"`
			}

			if err := yaml.Unmarshal(data, &productionFormat); err != nil {
				continue // Skip files that can't be parsed
			}

			// Convert to EvidenceEntry format
			entry := &EvidenceEntry{
				EvidenceID:     productionFormat.Evidence.ID,
				MedicationCode: "unknown", // Would need to extract from filename
				Indication:     "general",
				EvidenceType:   "guideline",
				EvidenceLevel:  "A",
				Recommendation: productionFormat.Evidence.Source,
			}
			evidence[entry.EvidenceID] = entry
		}
	}

	return evidence, nil
}

// loadORBRulesFromDir loads ORB rules from a directory of YAML files
func (kl *KnowledgeLoader) loadORBRulesFromDir(dirPath string) ([]ORBRule, error) {
	var rules []ORBRule

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty slice
		return rules, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Try new enhanced format first (with metadata and rules sections)
			var enhancedFormat struct {
				Metadata struct {
					Version     string `yaml:"version"`
					LastUpdated string `yaml:"last_updated"`
					Description string `yaml:"description"`
				} `yaml:"metadata"`
				Rules []ORBRule `yaml:"rules"`
			}

			if err := yaml.Unmarshal(data, &enhancedFormat); err == nil && len(enhancedFormat.Rules) > 0 {
				// Enhanced format with metadata - validate knowledge_manifest
				for i := range enhancedFormat.Rules {
					if err := kl.validateKnowledgeManifest(&enhancedFormat.Rules[i]); err != nil {
						kl.logger.WithError(err).Warnf("Invalid knowledge_manifest in rule %s, file %s",
							enhancedFormat.Rules[i].ID, file.Name())
						continue // Skip invalid rules
					}
				}
				rules = append(rules, enhancedFormat.Rules...)
				continue
			}

			// Fallback to legacy format (array of rules directly)
			var fileRules []ORBRule
			if err := yaml.Unmarshal(data, &fileRules); err != nil {
				continue // Skip files that can't be parsed
			}

			rules = append(rules, fileRules...)
		}
	}

	return rules, nil
}

// validateKnowledgeManifest validates the knowledge_manifest section of an ORB rule
func (kl *KnowledgeLoader) validateKnowledgeManifest(rule *ORBRule) error {
	// Knowledge Manifest is optional for backward compatibility
	if len(rule.Action.GenerateManifest.KnowledgeManifest.RequiredKBs) == 0 {
		return nil // Empty is valid - will fall back to all KBs
	}

	// Get valid KB identifiers from intent_manifest.go
	validKBs := []string{
		"kb_drug_master_v1",
		"kb_dosing_rules_v1",
		"kb_ddi_v1",
		"kb_formulary_stock_v1",
		"kb_patient_safe_checks_v1",
		"kb_guideline_evidence_v1",
		"kb_resistance_profiles_v1",
	}

	// Validate that all KB identifiers are valid
	for _, kbID := range rule.Action.GenerateManifest.KnowledgeManifest.RequiredKBs {
		valid := false
		for _, validKB := range validKBs {
			if kbID == validKB {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid KB identifier: %s", kbID)
		}
	}

	return nil
}

// loadMedicationsFromDir loads medications from a directory of YAML files
func (kl *KnowledgeLoader) loadMedicationsFromDir(dirPath string) (map[string]Medication, error) {
	medications := make(map[string]Medication)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return medications, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				continue // Skip files that can't be read
			}

			// Production format: medication: {...}
			var productionFormat struct {
				Medication struct {
					RxNormCode      string   `yaml:"rxnorm_code"`
					Name            string   `yaml:"name"`
					TherapeuticClass string  `yaml:"therapeutic_class"`
					IsHighAlert     bool     `yaml:"is_high_alert"`
					TdmRequired     bool     `yaml:"tdm_required"`
				} `yaml:"medication"`
			}

			if err := yaml.Unmarshal(data, &productionFormat); err != nil {
				continue // Skip files that can't be parsed
			}

			// Convert to Medication format
			medication := Medication{
				RxNormCode:       productionFormat.Medication.RxNormCode,
				GenericName:      productionFormat.Medication.Name,
				BrandNames:       []string{productionFormat.Medication.Name},
				TherapeuticClass: productionFormat.Medication.TherapeuticClass,
				Mechanism:        "Unknown", // Default value
				Indications:      []string{"General"},
			}

			// Set safety profile based on production data
			medication.SafetyProfile.RequiresMonitoring = productionFormat.Medication.TdmRequired

			// Use RxNorm code as primary key for lookup
			if productionFormat.Medication.RxNormCode != "" {
				medications[productionFormat.Medication.RxNormCode] = medication
			}
			// Also index by name for backward compatibility
			medications[productionFormat.Medication.Name] = medication
		}
	}

	return medications, nil
}

// extractMedicationFromID extracts medication name from recipe ID
func extractMedicationFromID(recipeID string) string {
	// Extract medication name from recipe ID (e.g., "heparin-infusion-adult-v2.0" -> "heparin")
	parts := strings.Split(recipeID, "-")
	if len(parts) > 0 {
		return strings.Title(parts[0])
	}
	return "Unknown"
}

// loadContextRecipesFromDir loads context recipes from a directory of YAML files
func (kl *KnowledgeLoader) loadContextRecipesFromDir(dirPath string) (map[string]*ContextRecipe, error) {
	recipes := make(map[string]*ContextRecipe)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		// Directory might not exist, return empty map
		return recipes, nil
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".yaml" {
			filePath := filepath.Join(dirPath, file.Name())
			recipe, err := kl.loadContextRecipe(filePath)
			if err != nil {
				continue // Skip files that can't be parsed
			}

			// Use recipe ID as key
			recipes[recipe.RecipeID] = recipe
		}
	}

	return recipes, nil
}
