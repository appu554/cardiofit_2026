package loaders

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"kb-clinical-context/internal/engines"
)

// PhenotypeLoader handles loading and parsing phenotype definitions from YAML files
type PhenotypeLoader struct {
	logger          *zap.Logger
	phenotypeDir    string
	loadedPhenotypes map[string]engines.PhenotypeDefinitionYAML
}

// PhenotypeLibraryMetadata represents the metadata section of a phenotype library
type PhenotypeLibraryMetadata struct {
	KBName        string   `yaml:"kb_name"`
	Version       string   `yaml:"version"`
	Institution   string   `yaml:"institution"`
	EffectiveDate string   `yaml:"effective_date"`
	ExpiresDate   string   `yaml:"expires_date"`
	ClinicalDomains []string `yaml:"clinical_domains"`
}

// PhenotypeLibrary represents a complete phenotype library file
type PhenotypeLibrary struct {
	Metadata   PhenotypeLibraryMetadata          `yaml:"metadata"`
	Phenotypes []engines.PhenotypeDefinitionYAML `yaml:"phenotypes"`
}

// LoaderConfig contains configuration for the phenotype loader
type LoaderConfig struct {
	PhenotypeDirectory string
	FileExtensions     []string
	ValidateOnLoad     bool
	EnableCaching      bool
}

// NewPhenotypeLoader creates a new phenotype loader
func NewPhenotypeLoader(logger *zap.Logger, config LoaderConfig) *PhenotypeLoader {
	if config.FileExtensions == nil {
		config.FileExtensions = []string{".yaml", ".yml"}
	}

	return &PhenotypeLoader{
		logger:           logger,
		phenotypeDir:     config.PhenotypeDirectory,
		loadedPhenotypes: make(map[string]engines.PhenotypeDefinitionYAML),
	}
}

// LoadAllPhenotypes loads all phenotype definitions from the configured directory
func (p *PhenotypeLoader) LoadAllPhenotypes() ([]engines.PhenotypeDefinitionYAML, error) {
	var allPhenotypes []engines.PhenotypeDefinitionYAML

	// Walk through the phenotype directory
	err := filepath.WalkDir(p.phenotypeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			p.logger.Warn("Error accessing path", zap.String("path", path), zap.Error(err))
			return nil // Continue walking
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file has valid extension
		if !p.hasValidExtension(path) {
			return nil
		}

		p.logger.Info("Loading phenotype file", zap.String("file", path))

		// Load phenotypes from file
		phenotypes, err := p.LoadPhenotypesFromFile(path)
		if err != nil {
			p.logger.Error("Failed to load phenotypes from file", 
				zap.String("file", path), 
				zap.Error(err))
			return nil // Continue with other files
		}

		allPhenotypes = append(allPhenotypes, phenotypes...)
		p.logger.Info("Successfully loaded phenotypes from file", 
			zap.String("file", path), 
			zap.Int("count", len(phenotypes)))

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk phenotype directory: %w", err)
	}

	// Cache loaded phenotypes
	for _, phenotype := range allPhenotypes {
		p.loadedPhenotypes[phenotype.ID] = phenotype
	}

	p.logger.Info("Completed loading all phenotypes", 
		zap.Int("total_phenotypes", len(allPhenotypes)),
		zap.Int("files_processed", len(allPhenotypes)))

	return allPhenotypes, nil
}

// LoadPhenotypesFromFile loads phenotypes from a specific YAML file
func (p *PhenotypeLoader) LoadPhenotypesFromFile(filePath string) ([]engines.PhenotypeDefinitionYAML, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse YAML content
	var library PhenotypeLibrary
	if err := yaml.Unmarshal(content, &library); err != nil {
		return nil, fmt.Errorf("failed to parse YAML from %s: %w", filePath, err)
	}

	// Validate metadata
	if err := p.validateMetadata(library.Metadata); err != nil {
		p.logger.Warn("Invalid metadata in phenotype file", 
			zap.String("file", filePath), 
			zap.Error(err))
	}

	// Process phenotypes
	var validPhenotypes []engines.PhenotypeDefinitionYAML
	for i, phenotype := range library.Phenotypes {
		// Note: Metadata would need to be added to PhenotypeDefinitionYAML struct
		// For now, skip metadata assignment

		// Validate phenotype
		if err := p.validatePhenotype(phenotype); err != nil {
			p.logger.Error("Invalid phenotype definition", 
				zap.String("file", filePath),
				zap.Int("phenotype_index", i),
				zap.String("phenotype_id", phenotype.ID),
				zap.Error(err))
			continue // Skip invalid phenotypes
		}

		validPhenotypes = append(validPhenotypes, phenotype)
	}

	return validPhenotypes, nil
}

// LoadPhenotypeByID loads a specific phenotype by ID
func (p *PhenotypeLoader) LoadPhenotypeByID(phenotypeID string) (*engines.PhenotypeDefinitionYAML, error) {
	// Check cache first
	if phenotype, exists := p.loadedPhenotypes[phenotypeID]; exists {
		return &phenotype, nil
	}

	// If not in cache, load all phenotypes
	phenotypes, err := p.LoadAllPhenotypes()
	if err != nil {
		return nil, fmt.Errorf("failed to load phenotypes: %w", err)
	}

	// Search for the specific phenotype
	for _, phenotype := range phenotypes {
		if phenotype.ID == phenotypeID {
			return &phenotype, nil
		}
	}

	return nil, fmt.Errorf("phenotype with ID %s not found", phenotypeID)
}

// LoadPhenotypesByDomain loads all phenotypes for a specific clinical domain
func (p *PhenotypeLoader) LoadPhenotypesByDomain(domain string) ([]engines.PhenotypeDefinitionYAML, error) {
	allPhenotypes, err := p.LoadAllPhenotypes()
	if err != nil {
		return nil, fmt.Errorf("failed to load phenotypes: %w", err)
	}

	var domainPhenotypes []engines.PhenotypeDefinitionYAML
	domainLower := strings.ToLower(domain)

	for _, phenotype := range allPhenotypes {
		if strings.ToLower(phenotype.Domain) == domainLower {
			domainPhenotypes = append(domainPhenotypes, phenotype)
		}
	}

	return domainPhenotypes, nil
}

// GetLoadedPhenotypes returns all currently loaded phenotypes
func (p *PhenotypeLoader) GetLoadedPhenotypes() map[string]engines.PhenotypeDefinitionYAML {
	return p.loadedPhenotypes
}

// ReloadPhenotypes clears cache and reloads all phenotypes
func (p *PhenotypeLoader) ReloadPhenotypes() ([]engines.PhenotypeDefinitionYAML, error) {
	p.loadedPhenotypes = make(map[string]engines.PhenotypeDefinitionYAML)
	return p.LoadAllPhenotypes()
}

// hasValidExtension checks if a file has a valid extension
func (p *PhenotypeLoader) hasValidExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	validExtensions := []string{".yaml", ".yml"}
	
	for _, validExt := range validExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

// validateMetadata validates the metadata section of a phenotype library
func (p *PhenotypeLoader) validateMetadata(metadata PhenotypeLibraryMetadata) error {
	if metadata.KBName == "" {
		return fmt.Errorf("kb_name is required in metadata")
	}
	if metadata.Version == "" {
		return fmt.Errorf("version is required in metadata")
	}
	if len(metadata.ClinicalDomains) == 0 {
		return fmt.Errorf("at least one clinical domain must be specified")
	}
	return nil
}

// validatePhenotype validates a single phenotype definition
func (p *PhenotypeLoader) validatePhenotype(phenotype engines.PhenotypeDefinitionYAML) error {
	if phenotype.ID == "" {
		return fmt.Errorf("phenotype ID is required")
	}
	if phenotype.Name == "" {
		return fmt.Errorf("phenotype name is required")
	}
	if phenotype.Domain == "" {
		return fmt.Errorf("phenotype domain is required")
	}
	if phenotype.Criteria.Expression == "" {
		return fmt.Errorf("phenotype expression is required")
	}
	if phenotype.Status == "" {
		return fmt.Errorf("phenotype status is required")
	}

	// Validate logic engine if specified
	if phenotype.Criteria.LogicEngine != "" {
		validEngines := []string{"cel", "rego", "python", "sql", "custom"}
		engineValid := false
		for _, validEngine := range validEngines {
			if phenotype.Criteria.LogicEngine == validEngine {
				engineValid = true
				break
			}
		}
		if !engineValid {
			return fmt.Errorf("invalid logic engine: %s", phenotype.Criteria.LogicEngine)
		}
	}

	// Validate data requirements
	for i, req := range phenotype.Criteria.DataRequirements {
		if req.Field == "" {
			return fmt.Errorf("data requirement %d: field is required", i)
		}
		if req.Type == "" {
			return fmt.Errorf("data requirement %d: type is required", i)
		}
		
		// Validate type
		validTypes := []string{"integer", "float", "boolean", "string", "datetime"}
		typeValid := false
		for _, validType := range validTypes {
			if req.Type == validType {
				typeValid = true
				break
			}
		}
		if !typeValid {
			return fmt.Errorf("data requirement %d: invalid type %s", i, req.Type)
		}
	}

	return nil
}

// GetPhenotypeStats returns statistics about loaded phenotypes
func (p *PhenotypeLoader) GetPhenotypeStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_loaded": len(p.loadedPhenotypes),
		"phenotype_dir": p.phenotypeDir,
	}

	// Count by domain
	domainCounts := make(map[string]int)
	engineCounts := make(map[string]int)
	statusCounts := make(map[string]int)

	for _, phenotype := range p.loadedPhenotypes {
		domainCounts[phenotype.Domain]++
		
		engine := phenotype.Criteria.LogicEngine
		if engine == "" {
			engine = "default"
		}
		engineCounts[engine]++
		
		statusCounts[phenotype.Status]++
	}

	stats["by_domain"] = domainCounts
	stats["by_engine"] = engineCounts
	stats["by_status"] = statusCounts

	return stats
}

// ValidateAllPhenotypes validates all loaded phenotypes using the multi-engine evaluator
func (p *PhenotypeLoader) ValidateAllPhenotypes(evaluator *engines.MultiEngineEvaluator) []ValidationResult {
	var results []ValidationResult

	for _, phenotype := range p.loadedPhenotypes {
		result := ValidationResult{
			PhenotypeID: phenotype.ID,
			Valid:       true,
		}

		if err := evaluator.ValidatePhenotype(phenotype); err != nil {
			result.Valid = false
			result.Error = err.Error()
		}

		results = append(results, result)
	}

	return results
}

// ValidationResult represents the result of phenotype validation
type ValidationResult struct {
	PhenotypeID string `json:"phenotype_id"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}