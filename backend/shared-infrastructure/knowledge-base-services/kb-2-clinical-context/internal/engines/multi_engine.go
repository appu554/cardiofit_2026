package engines

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-clinical-context/internal/models"
)

// LogicEngine represents different logic engine types
type LogicEngine string

const (
	LogicEngineCEL    LogicEngine = "cel"
	LogicEngineRego   LogicEngine = "rego" 
	LogicEnginePython LogicEngine = "python"
	LogicEngineSQL    LogicEngine = "sql"
	LogicEngineCustom LogicEngine = "custom"
)

// ExpressionEvaluator interface defines the contract for all logic engines
type ExpressionEvaluator interface {
	EvaluateExpression(expression string, context models.PatientContext) (bool, float64, error)
	ValidateExpression(expression string) error
	GetEngineType() LogicEngine
}

// MultiEngineEvaluator orchestrates multiple logic engines for phenotype evaluation
type MultiEngineEvaluator struct {
	celEngine    *CELEngine
	regoEngine   ExpressionEvaluator // Future: OPA Rego engine
	pythonEngine ExpressionEvaluator // Future: Python expression engine
	sqlEngine    ExpressionEvaluator // Future: SQL query engine
	logger       *zap.Logger
	config       MultiEngineConfig
}

// MultiEngineConfig contains configuration for the multi-engine evaluator
type MultiEngineConfig struct {
	DefaultEngine         LogicEngine
	FallbackEngine        LogicEngine
	MaxEvaluationTime     time.Duration
	EnableParallelExecution bool
	EnableFallback        bool
}

// PhenotypeDefinitionYAML represents a phenotype definition loaded from YAML
type PhenotypeDefinitionYAML struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Domain      string `yaml:"domain"`
	Version     string `yaml:"version"`
	Status      string `yaml:"status"`
	Description string `yaml:"description"`
	Criteria    struct {
		LogicEngine        string                 `yaml:"logic_engine"`
		Expression         string                 `yaml:"expression"`
		DataRequirements   []DataRequirement      `yaml:"data_requirements"`
	} `yaml:"criteria"`
	Priority            int                    `yaml:"priority"`
	Outputs             map[string]string      `yaml:"outputs"`
	ClinicalImplications []ClinicalImplication `yaml:"clinical_implications"`
	EvidenceLinks       EvidenceLinks          `yaml:"evidence_links"`
}

// DataRequirement represents a data requirement for phenotype evaluation
type DataRequirement struct {
	Field           string            `yaml:"field"`
	Type            string            `yaml:"type"`
	Required        bool              `yaml:"required"`
	Source          string            `yaml:"source"`
	Units           string            `yaml:"units,omitempty"`
	TimeWindow      string            `yaml:"time_window,omitempty"`
	ValidationRules map[string]interface{} `yaml:"validation_rules,omitempty"`
}

// ClinicalImplication represents clinical implications of a phenotype
type ClinicalImplication struct {
	Implication    string `yaml:"implication"`
	Severity       string `yaml:"severity"`
	ActionRequired bool   `yaml:"action_required"`
	Timeframe      string `yaml:"timeframe"`
	ActionType     string `yaml:"action_type"`
}

// EvidenceLinks represents links to evidence and guidelines
type EvidenceLinks struct {
	KB3Guidelines []string                   `yaml:"kb3_guidelines"`
	KB4SafetyRules []string                  `yaml:"kb4_safety_rules"`
	LiteratureRefs []map[string]interface{}  `yaml:"literature_refs"`
}

// EvaluationResult represents the result of phenotype expression evaluation
type EvaluationResult struct {
	Matched       bool              `json:"matched"`
	Confidence    float64           `json:"confidence"`
	EngineUsed    LogicEngine       `json:"engine_used"`
	ExecutionTime time.Duration     `json:"execution_time"`
	Evidence      []EvidenceItem    `json:"evidence"`
	Errors        []string          `json:"errors,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// EvidenceItem represents a piece of evidence supporting the phenotype match
type EvidenceItem struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Value       interface{} `json:"value,omitempty"`
	Source      string      `json:"source"`
	Timestamp   time.Time   `json:"timestamp"`
	Confidence  float64     `json:"confidence"`
}

// NewMultiEngineEvaluator creates a new multi-engine evaluator
func NewMultiEngineEvaluator(logger *zap.Logger) (*MultiEngineEvaluator, error) {
	config := MultiEngineConfig{
		DefaultEngine:           LogicEngineCEL,
		FallbackEngine:          LogicEngineCEL,
		MaxEvaluationTime:       10 * time.Second,
		EnableParallelExecution: false, // Start with sequential for safety
		EnableFallback:          true,
	}

	// Initialize CEL engine
	celEngine, err := NewCELEngine(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize CEL engine: %w", err)
	}

	return &MultiEngineEvaluator{
		celEngine: celEngine,
		logger:    logger,
		config:    config,
	}, nil
}

// EvaluatePhenotype evaluates a phenotype definition against patient context
func (m *MultiEngineEvaluator) EvaluatePhenotype(
	phenotypeDef PhenotypeDefinitionYAML, 
	patientContext models.PatientContext,
) (*EvaluationResult, error) {
	startTime := time.Now()
	
	// Validate phenotype definition
	if err := m.validatePhenotypeDefinition(phenotypeDef); err != nil {
		return nil, fmt.Errorf("invalid phenotype definition: %w", err)
	}

	// Determine which engine to use
	engineType := LogicEngine(phenotypeDef.Criteria.LogicEngine)
	if engineType == "" {
		engineType = m.config.DefaultEngine
	}

	// Get the appropriate engine
	engine, err := m.getEngine(engineType)
	if err != nil {
		if m.config.EnableFallback && engineType != m.config.FallbackEngine {
			m.logger.Warn("Engine not available, falling back to default",
				zap.String("requested_engine", string(engineType)),
				zap.String("fallback_engine", string(m.config.FallbackEngine)),
				zap.Error(err))
			
			engine, err = m.getEngine(m.config.FallbackEngine)
			if err != nil {
				return nil, fmt.Errorf("fallback engine also failed: %w", err)
			}
			engineType = m.config.FallbackEngine
		} else {
			return nil, fmt.Errorf("failed to get engine %s: %w", engineType, err)
		}
	}

	// Set up timeout context
	ctx, cancel := context.WithTimeout(context.Background(), m.config.MaxEvaluationTime)
	defer cancel()

	// Evaluate expression with timeout
	resultChan := make(chan *EvaluationResult, 1)
	errorChan := make(chan error, 1)

	go func() {
		matched, confidence, err := engine.EvaluateExpression(phenotypeDef.Criteria.Expression, patientContext)
		if err != nil {
			errorChan <- err
			return
		}

		evidence := m.buildEvidence(phenotypeDef, patientContext, matched)
		
		result := &EvaluationResult{
			Matched:       matched,
			Confidence:    confidence,
			EngineUsed:    engineType,
			ExecutionTime: time.Since(startTime),
			Evidence:      evidence,
			Metadata: map[string]interface{}{
				"phenotype_id":   phenotypeDef.ID,
				"phenotype_name": phenotypeDef.Name,
				"version":        phenotypeDef.Version,
				"priority":       phenotypeDef.Priority,
			},
		}

		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		return &EvaluationResult{
			Matched:       false,
			Confidence:    0.0,
			EngineUsed:    engineType,
			ExecutionTime: time.Since(startTime),
			Errors:        []string{"evaluation timeout"},
		}, nil
	case err := <-errorChan:
		return &EvaluationResult{
			Matched:       false,
			Confidence:    0.0,
			EngineUsed:    engineType,
			ExecutionTime: time.Since(startTime),
			Errors:        []string{err.Error()},
		}, nil
	case result := <-resultChan:
		m.logger.Info("Phenotype evaluation completed",
			zap.String("phenotype_id", phenotypeDef.ID),
			zap.Bool("matched", result.Matched),
			zap.Float64("confidence", result.Confidence),
			zap.String("engine", string(result.EngineUsed)),
			zap.Duration("execution_time", result.ExecutionTime))
		
		return result, nil
	}
}

// ValidatePhenotype validates a phenotype definition's expression
func (m *MultiEngineEvaluator) ValidatePhenotype(phenotypeDef PhenotypeDefinitionYAML) error {
	// Validate phenotype definition structure
	if err := m.validatePhenotypeDefinition(phenotypeDef); err != nil {
		return err
	}

	// Determine engine type
	engineType := LogicEngine(phenotypeDef.Criteria.LogicEngine)
	if engineType == "" {
		engineType = m.config.DefaultEngine
	}

	// Get engine and validate expression
	engine, err := m.getEngine(engineType)
	if err != nil {
		return fmt.Errorf("failed to get engine %s: %w", engineType, err)
	}

	return engine.ValidateExpression(phenotypeDef.Criteria.Expression)
}

// getEngine returns the appropriate engine based on type
func (m *MultiEngineEvaluator) getEngine(engineType LogicEngine) (ExpressionEvaluator, error) {
	switch engineType {
	case LogicEngineCEL:
		if m.celEngine == nil {
			return nil, fmt.Errorf("CEL engine not initialized")
		}
		return m.celEngine, nil
	case LogicEngineRego:
		if m.regoEngine == nil {
			return nil, fmt.Errorf("Rego engine not implemented yet")
		}
		return m.regoEngine, nil
	case LogicEnginePython:
		if m.pythonEngine == nil {
			return nil, fmt.Errorf("Python engine not implemented yet")
		}
		return m.pythonEngine, nil
	case LogicEngineSQL:
		if m.sqlEngine == nil {
			return nil, fmt.Errorf("SQL engine not implemented yet")
		}
		return m.sqlEngine, nil
	default:
		return nil, fmt.Errorf("unsupported engine type: %s", engineType)
	}
}

// validatePhenotypeDefinition validates the structure of a phenotype definition
func (m *MultiEngineEvaluator) validatePhenotypeDefinition(phenotypeDef PhenotypeDefinitionYAML) error {
	if phenotypeDef.ID == "" {
		return fmt.Errorf("phenotype ID is required")
	}
	if phenotypeDef.Name == "" {
		return fmt.Errorf("phenotype name is required")
	}
	if phenotypeDef.Criteria.Expression == "" {
		return fmt.Errorf("phenotype expression is required")
	}
	if phenotypeDef.Status != "active" {
		return fmt.Errorf("only active phenotypes can be evaluated")
	}

	// Validate logic engine type
	engineType := LogicEngine(phenotypeDef.Criteria.LogicEngine)
	if engineType != "" && !m.isEngineSupported(engineType) {
		return fmt.Errorf("unsupported logic engine: %s", engineType)
	}

	// Validate data requirements
	for _, req := range phenotypeDef.Criteria.DataRequirements {
		if req.Field == "" {
			return fmt.Errorf("data requirement field is required")
		}
		if req.Type == "" {
			return fmt.Errorf("data requirement type is required")
		}
	}

	return nil
}

// isEngineSupported checks if an engine type is supported
func (m *MultiEngineEvaluator) isEngineSupported(engineType LogicEngine) bool {
	switch engineType {
	case LogicEngineCEL:
		return m.celEngine != nil
	case LogicEngineRego:
		return m.regoEngine != nil
	case LogicEnginePython:
		return m.pythonEngine != nil
	case LogicEngineSQL:
		return m.sqlEngine != nil
	default:
		return false
	}
}

// buildEvidence builds evidence for the phenotype evaluation
func (m *MultiEngineEvaluator) buildEvidence(
	phenotypeDef PhenotypeDefinitionYAML,
	patientContext models.PatientContext,
	matched bool,
) []EvidenceItem {
	var evidence []EvidenceItem
	now := time.Now()

	// Add condition evidence
	for _, condition := range patientContext.ActiveConditions {
		evidence = append(evidence, EvidenceItem{
			Type:        "condition",
			Description: fmt.Sprintf("Active condition: %s", condition.Name),
			Value:       condition.Code,
			Source:      "ehr",
			Timestamp:   condition.OnsetDate,
			Confidence:  0.9,
		})
	}

	// Add lab evidence
	for _, lab := range patientContext.RecentLabs {
		evidence = append(evidence, EvidenceItem{
			Type:        "laboratory",
			Description: fmt.Sprintf("Lab result: %s", lab.LOINCCode),
			Value:       lab.Value,
			Source:      "laboratory",
			Timestamp:   lab.ResultDate,
			Confidence:  0.95,
		})
	}

	// Add medication evidence
	for _, med := range patientContext.CurrentMeds {
		evidence = append(evidence, EvidenceItem{
			Type:        "medication",
			Description: fmt.Sprintf("Current medication: %s", med.Name),
			Value:       med.RxNormCode,
			Source:      "pharmacy",
			Timestamp:   med.StartDate,
			Confidence:  0.9,
		})
	}

	// Add demographic evidence
	evidence = append(evidence, EvidenceItem{
		Type:        "demographics",
		Description: fmt.Sprintf("Patient age: %d, sex: %s", patientContext.Demographics.AgeYears, patientContext.Demographics.Sex),
		Value: map[string]interface{}{
			"age": patientContext.Demographics.AgeYears,
			"sex": patientContext.Demographics.Sex,
		},
		Source:     "demographics",
		Timestamp:  now,
		Confidence: 1.0,
	})

	return evidence
}

// GetEngineStats returns statistics about all engines
func (m *MultiEngineEvaluator) GetEngineStats() map[string]interface{} {
	stats := map[string]interface{}{
		"default_engine":     string(m.config.DefaultEngine),
		"fallback_engine":    string(m.config.FallbackEngine),
		"supported_engines":  m.getSupportedEngines(),
		"max_evaluation_time": m.config.MaxEvaluationTime.String(),
	}

	// Add CEL-specific stats
	if m.celEngine != nil {
		stats["cel_engine"] = m.celEngine.GetCacheStats()
	}

	return stats
}

// getSupportedEngines returns a list of supported engine types
func (m *MultiEngineEvaluator) getSupportedEngines() []string {
	var engines []string
	
	if m.celEngine != nil {
		engines = append(engines, string(LogicEngineCEL))
	}
	if m.regoEngine != nil {
		engines = append(engines, string(LogicEngineRego))
	}
	if m.pythonEngine != nil {
		engines = append(engines, string(LogicEnginePython))
	}
	if m.sqlEngine != nil {
		engines = append(engines, string(LogicEngineSQL))
	}
	
	return engines
}

// CELEngine adapter to implement ExpressionEvaluator interface
func (c *CELEngine) GetEngineType() LogicEngine {
	return LogicEngineCEL
}