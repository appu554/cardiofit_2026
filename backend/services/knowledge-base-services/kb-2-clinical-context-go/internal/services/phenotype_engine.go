package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/models"
)

// PhenotypeEngine handles CEL-based phenotype evaluation
type PhenotypeEngine struct {
	config       *config.Config
	celEnv       *cel.Env
	phenotypesMu sync.RWMutex
	phenotypes   map[string]*models.PhenotypeDefinition
}

// NewPhenotypeEngine creates a new phenotype engine with CEL integration
func NewPhenotypeEngine(cfg *config.Config) *PhenotypeEngine {
	// Initialize CEL environment with patient data types
	env, err := cel.NewEnv(
		cel.Declarations(
			// Patient data declarations
			decls.NewVar("patient", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("age", decls.Int),
			decls.NewVar("gender", decls.String),
			decls.NewVar("conditions", decls.NewListType(decls.String)),
			decls.NewVar("medications", decls.NewListType(decls.String)),
			decls.NewVar("labs", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("vitals", decls.NewMapType(decls.String, decls.Any)),
			decls.NewVar("allergies", decls.NewListType(decls.String)),
			
			// Helper functions
			decls.NewFunction("has_condition",
				decls.NewOverload("has_condition_string",
					[]*exprpb.Type{decls.String}, decls.Bool)),
			decls.NewFunction("has_medication",
				decls.NewOverload("has_medication_string",
					[]*exprpb.Type{decls.String}, decls.Bool)),
			decls.NewFunction("lab_value",
				decls.NewOverload("lab_value_string",
					[]*exprpb.Type{decls.String}, decls.Double)),
			decls.NewFunction("vital_value",
				decls.NewOverload("vital_value_string",
					[]*exprpb.Type{decls.String}, decls.Double)),
			decls.NewFunction("has_allergy",
				decls.NewOverload("has_allergy_string",
					[]*exprpb.Type{decls.String}, decls.Bool)),
		),
	)
	
	if err != nil {
		panic(fmt.Sprintf("Failed to create CEL environment: %v", err))
	}

	return &PhenotypeEngine{
		config:     cfg,
		celEnv:     env,
		phenotypes: make(map[string]*models.PhenotypeDefinition),
	}
}

// LoadPhenotypes loads phenotype definitions from knowledge base
func (pe *PhenotypeEngine) LoadPhenotypes(phenotypes []*models.PhenotypeDefinition) error {
	pe.phenotypesMu.Lock()
	defer pe.phenotypesMu.Unlock()
	
	for _, phenotype := range phenotypes {
		// Validate CEL rule
		if err := pe.validateCELRule(phenotype.CELRule); err != nil {
			return fmt.Errorf("invalid CEL rule for phenotype %s: %w", phenotype.Name, err)
		}
		
		pe.phenotypes[phenotype.ID.Hex()] = phenotype
	}
	
	return nil
}

// EvaluatePhenotypes evaluates phenotypes for multiple patients using batch processing
func (pe *PhenotypeEngine) EvaluatePhenotypes(ctx context.Context, request *models.PhenotypeEvaluationRequest) ([]models.PhenotypeEvaluationResult, error) {
	startTime := time.Now()
	results := make([]models.PhenotypeEvaluationResult, 0, len(request.Patients))
	
	// Process patients in batches to avoid overwhelming the system
	batchSize := pe.config.BatchSize
	if batchSize > len(request.Patients) {
		batchSize = len(request.Patients)
	}
	
	// Channel for collecting results
	resultsChan := make(chan models.PhenotypeEvaluationResult, len(request.Patients))
	errorsChan := make(chan error, len(request.Patients))
	
	// Worker pool for parallel processing
	maxWorkers := pe.config.MaxConcurrentRequests
	if maxWorkers > len(request.Patients) {
		maxWorkers = len(request.Patients)
	}
	
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	
	for _, patient := range request.Patients {
		wg.Add(1)
		go func(p models.Patient) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			// Evaluate phenotypes for this patient
			result, err := pe.evaluatePatientPhenotypes(ctx, p, request.PhenotypeIDs, request.IncludeExplanation)
			if err != nil {
				errorsChan <- err
				return
			}
			
			resultsChan <- result
		}(patient)
	}
	
	// Close channels when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()
	
	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}
	
	// Check for errors
	select {
	case err := <-errorsChan:
		return results, fmt.Errorf("phenotype evaluation error: %w", err)
	default:
	}
	
	// Check SLA compliance
	totalDuration := time.Since(startTime)
	slaThreshold := time.Duration(pe.config.PhenotypeEvaluationSLA) * time.Millisecond
	if totalDuration > slaThreshold {
		return results, fmt.Errorf("SLA violation: evaluation took %v, threshold is %v", totalDuration, slaThreshold)
	}
	
	return results, nil
}

// evaluatePatientPhenotypes evaluates phenotypes for a single patient
func (pe *PhenotypeEngine) evaluatePatientPhenotypes(ctx context.Context, patient models.Patient, phenotypeIDs []string, includeExplanation bool) (models.PhenotypeEvaluationResult, error) {
	startTime := time.Now()
	result := models.PhenotypeEvaluationResult{
		PatientID:  patient.ID,
		Phenotypes: []models.DetectedPhenotype{},
	}
	
	pe.phenotypesMu.RLock()
	defer pe.phenotypesMu.RUnlock()
	
	// Determine which phenotypes to evaluate
	phenotypesToEvaluate := pe.getPhenotypesToEvaluate(phenotypeIDs)
	
	// Create CEL evaluation context
	celContext := pe.createCELContext(patient)
	
	// Evaluate each phenotype
	for _, phenotype := range phenotypesToEvaluate {
		detected, confidence, evidence, err := pe.evaluateSinglePhenotype(ctx, phenotype, celContext)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Error evaluating phenotype %s: %v", phenotype.Name, err))
			continue
		}
		
		detectedPhenotype := models.DetectedPhenotype{
			ID:         phenotype.ID.Hex(),
			Name:       phenotype.Name,
			Category:   phenotype.Category,
			Detected:   detected,
			Confidence: confidence,
			Evidence:   evidence,
			Metadata:   phenotype.Metadata,
		}
		
		result.Phenotypes = append(result.Phenotypes, detectedPhenotype)
	}
	
	// Generate explanation if requested
	if includeExplanation {
		explanation := pe.generatePhenotypeExplanation(patient, result.Phenotypes)
		result.Explanation = &explanation
	}
	
	result.ProcessingTime = time.Since(startTime)
	return result, nil
}

// evaluateSinglePhenotype evaluates a single phenotype against patient data
func (pe *PhenotypeEngine) evaluateSinglePhenotype(ctx context.Context, phenotype *models.PhenotypeDefinition, celContext map[string]interface{}) (bool, float64, []models.EvidenceItem, error) {
	// Parse and compile CEL expression
	ast, issues := pe.celEnv.Compile(phenotype.CELRule)
	if issues != nil && issues.Err() != nil {
		return false, 0, nil, fmt.Errorf("CEL compilation error: %w", issues.Err())
	}
	
	// Create program
	program, err := pe.celEnv.Program(ast)
	if err != nil {
		return false, 0, nil, fmt.Errorf("CEL program creation error: %w", err)
	}
	
	// Evaluate with timeout
	evalCtx, cancel := context.WithTimeout(ctx, time.Duration(pe.config.CELTimeout)*time.Millisecond)
	defer cancel()
	
	// Execute evaluation
	val, _, err := program.ContextEval(evalCtx, celContext)
	if err != nil {
		return false, 0, nil, fmt.Errorf("CEL evaluation error: %w", err)
	}
	
	// Extract result
	detected, ok := val.Value().(bool)
	if !ok {
		return false, 0, nil, fmt.Errorf("CEL expression must return boolean, got %T", val.Value())
	}
	
	// Calculate confidence based on evidence strength
	confidence := pe.calculateConfidence(phenotype, celContext, detected)
	
	// Generate evidence items
	evidence := pe.generateEvidence(phenotype, celContext, detected)
	
	return detected, confidence, evidence, nil
}

// createCELContext creates evaluation context from patient data
func (pe *PhenotypeEngine) createCELContext(patient models.Patient) map[string]interface{} {
	// Convert labs to simple map
	labs := make(map[string]interface{})
	for name, labValue := range patient.Labs {
		labs[name] = labValue.Value
	}
	
	// Convert vitals to simple map
	vitals := make(map[string]interface{})
	for name, vitalValue := range patient.Vitals {
		vitals[name] = vitalValue.Value
	}
	
	context := map[string]interface{}{
		"patient":     patient,
		"age":         patient.Age,
		"gender":      patient.Gender,
		"conditions":  patient.Conditions,
		"medications": patient.Medications,
		"labs":        labs,
		"vitals":      vitals,
		"allergies":   patient.Allergies,
		
		// Helper functions
		"has_condition": func(condition string) bool {
			for _, c := range patient.Conditions {
				if c == condition {
					return true
				}
			}
			return false
		},
		"has_medication": func(medication string) bool {
			for _, m := range patient.Medications {
				if m == medication {
					return true
				}
			}
			return false
		},
		"lab_value": func(labName string) float64 {
			if lab, exists := patient.Labs[labName]; exists {
				return lab.Value
			}
			return 0
		},
		"vital_value": func(vitalName string) float64 {
			if vital, exists := patient.Vitals[vitalName]; exists {
				return vital.Value
			}
			return 0
		},
		"has_allergy": func(allergy string) bool {
			for _, a := range patient.Allergies {
				if a == allergy {
					return true
				}
			}
			return false
		},
	}
	
	return context
}

// getPhenotypesToEvaluate returns phenotypes to evaluate based on request
func (pe *PhenotypeEngine) getPhenotypesToEvaluate(phenotypeIDs []string) []*models.PhenotypeDefinition {
	var phenotypes []*models.PhenotypeDefinition
	
	if len(phenotypeIDs) == 0 {
		// Evaluate all phenotypes
		for _, phenotype := range pe.phenotypes {
			phenotypes = append(phenotypes, phenotype)
		}
	} else {
		// Evaluate specific phenotypes
		for _, id := range phenotypeIDs {
			if phenotype, exists := pe.phenotypes[id]; exists {
				phenotypes = append(phenotypes, phenotype)
			}
		}
	}
	
	return phenotypes
}

// calculateConfidence calculates confidence score for phenotype detection
func (pe *PhenotypeEngine) calculateConfidence(phenotype *models.PhenotypeDefinition, context map[string]interface{}, detected bool) float64 {
	if !detected {
		return 0.0
	}
	
	// Base confidence starts at 0.5
	confidence := 0.5
	
	// Increase confidence based on available evidence
	if patient, ok := context["patient"].(models.Patient); ok {
		// More conditions = higher confidence (up to 0.3 boost)
		conditionBoost := float64(len(patient.Conditions)) * 0.05
		if conditionBoost > 0.3 {
			conditionBoost = 0.3
		}
		confidence += conditionBoost
		
		// More recent lab values = higher confidence (up to 0.2 boost)
		labBoost := float64(len(patient.Labs)) * 0.02
		if labBoost > 0.2 {
			labBoost = 0.2
		}
		confidence += labBoost
	}
	
	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

// generateEvidence generates evidence items for phenotype detection
func (pe *PhenotypeEngine) generateEvidence(phenotype *models.PhenotypeDefinition, context map[string]interface{}, detected bool) []models.EvidenceItem {
	var evidence []models.EvidenceItem
	
	if patient, ok := context["patient"].(models.Patient); ok {
		// Add condition evidence
		for _, condition := range patient.Conditions {
			evidence = append(evidence, models.EvidenceItem{
				Type:        "condition",
				Value:       condition,
				Description: fmt.Sprintf("Patient has condition: %s", condition),
				Weight:      0.8,
			})
		}
		
		// Add lab evidence
		for name, lab := range patient.Labs {
			evidence = append(evidence, models.EvidenceItem{
				Type:        "lab",
				Value:       lab.Value,
				Description: fmt.Sprintf("Lab %s: %.2f %s", name, lab.Value, lab.Unit),
				Weight:      0.6,
			})
		}
		
		// Add medication evidence
		for _, medication := range patient.Medications {
			evidence = append(evidence, models.EvidenceItem{
				Type:        "medication",
				Value:       medication,
				Description: fmt.Sprintf("Patient taking: %s", medication),
				Weight:      0.7,
			})
		}
	}
	
	return evidence
}

// generatePhenotypeExplanation generates reasoning explanation
func (pe *PhenotypeEngine) generatePhenotypeExplanation(patient models.Patient, phenotypes []models.DetectedPhenotype) models.PhenotypeExplanation {
	explanation := models.PhenotypeExplanation{
		PatientID:       patient.ID,
		ReasoningChains: []models.ReasoningChain{},
		GeneratedAt:     time.Now(),
	}
	
	for _, phenotype := range phenotypes {
		if phenotype.Detected {
			chain := models.ReasoningChain{
				PhenotypeID:   phenotype.ID,
				PhenotypeName: phenotype.Name,
				Steps:         pe.generateReasoningSteps(phenotype, patient),
				Conclusion:    fmt.Sprintf("Phenotype %s detected with confidence %.2f", phenotype.Name, phenotype.Confidence),
			}
			explanation.ReasoningChains = append(explanation.ReasoningChains, chain)
		}
	}
	
	return explanation
}

// generateReasoningSteps generates reasoning steps for explanation
func (pe *PhenotypeEngine) generateReasoningSteps(phenotype models.DetectedPhenotype, patient models.Patient) []models.ReasoningStep {
	var steps []models.ReasoningStep
	
	// Step 1: Evidence gathering
	steps = append(steps, models.ReasoningStep{
		Rule:        "evidence_gathering",
		Evaluation:  "Collected patient clinical data",
		Result:      len(phenotype.Evidence),
		Explanation: fmt.Sprintf("Found %d evidence items supporting the phenotype", len(phenotype.Evidence)),
	})
	
	// Step 2: Evidence evaluation
	steps = append(steps, models.ReasoningStep{
		Rule:        "evidence_evaluation",
		Evaluation:  "Evaluated evidence strength",
		Result:      phenotype.Confidence,
		Explanation: fmt.Sprintf("Evidence supports detection with confidence %.2f", phenotype.Confidence),
	})
	
	// Step 3: Final determination
	steps = append(steps, models.ReasoningStep{
		Rule:        "final_determination",
		Evaluation:  "Applied phenotype criteria",
		Result:      phenotype.Detected,
		Explanation: fmt.Sprintf("Phenotype %s based on available evidence", map[bool]string{true: "detected", false: "not detected"}[phenotype.Detected]),
	})
	
	return steps
}

// validateCELRule validates a CEL rule syntax
func (pe *PhenotypeEngine) validateCELRule(rule string) error {
	_, issues := pe.celEnv.Compile(rule)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}
	return nil
}

// GetAvailablePhenotypes returns list of available phenotypes
func (pe *PhenotypeEngine) GetAvailablePhenotypes() []models.PhenotypeDefinition {
	pe.phenotypesMu.RLock()
	defer pe.phenotypesMu.RUnlock()
	
	phenotypes := make([]models.PhenotypeDefinition, 0, len(pe.phenotypes))
	for _, phenotype := range pe.phenotypes {
		phenotypes = append(phenotypes, *phenotype)
	}
	
	return phenotypes
}