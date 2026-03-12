// Package cql provides CQL (Clinical Quality Language) integration for KB-9.
// This package wraps the vaidshala clinical-runtime-platform for care gap detection.
//
// Per CTO/CMO Architecture:
//   - Uses vaidshala CQL Engine for clinical truth determination
//   - Uses vaidshala Measure Engine for care gap detection
//   - Works with the FROZEN ClinicalExecutionContext contract
package cql

import (
	"context"
	"time"

	"go.uber.org/zap"

	// Import vaidshala contracts and engines
	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines"
)

// Executor wraps the vaidshala CQL and Measure engines for KB-9 use.
// It provides a simplified interface for care gap detection.
type Executor struct {
	cqlEngine     *engines.CQLEngine
	measureEngine *engines.MeasureEngine
	logger        *zap.Logger
	region        string
}

// ExecutorConfig configures the CQL executor.
type ExecutorConfig struct {
	Region string // AU, IN, US
}

// NewExecutor creates a new CQL executor with vaidshala engines.
func NewExecutor(config ExecutorConfig, logger *zap.Logger) *Executor {
	// Initialize CQL Engine with region-specific configuration
	cqlConfig := engines.DefaultCQLEngineConfig()
	cqlConfig.Region = config.Region

	cqlEngine := engines.NewCQLEngine(cqlConfig)

	// Initialize Measure Engine
	measureConfig := engines.DefaultMeasureEngineConfig()
	measureConfig.Region = config.Region

	measureEngine := engines.NewMeasureEngine(measureConfig)

	return &Executor{
		cqlEngine:     cqlEngine,
		measureEngine: measureEngine,
		logger:        logger,
		region:        config.Region,
	}
}

// EvaluationResult contains the results of CQL and Measure evaluation.
type EvaluationResult struct {
	// Clinical facts determined by CQL Engine
	ClinicalFacts []contracts.ClinicalFact `json:"clinical_facts"`

	// Measure results with care gap information
	MeasureResults []contracts.MeasureResult `json:"measure_results"`

	// Total execution time
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// Engine versions for audit
	CQLEngineVersion     string `json:"cql_engine_version"`
	MeasureEngineVersion string `json:"measure_engine_version"`
}

// Evaluate runs the full CQL → Measure Engine pipeline.
//
// Flow:
//  1. CQL Engine evaluates clinical facts (truths) from patient context
//  2. Measure Engine consumes facts to determine care gaps
//  3. Combined results returned for care gap reporting
//
// This implements the CTO/CMO architecture where:
//   - CQL Engine = "What is true about this patient?"
//   - Measure Engine = "Given what's true, are we meeting standards of care?"
func (e *Executor) Evaluate(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
) (*EvaluationResult, error) {
	startTime := time.Now()

	e.logger.Debug("Starting CQL evaluation",
		zap.String("patient_id", execCtx.Patient.Demographics.PatientID),
		zap.String("region", e.region),
	)

	// Step 1: CQL Engine evaluates clinical facts
	cqlResult, err := e.cqlEngine.Evaluate(ctx, execCtx)
	if err != nil {
		e.logger.Error("CQL Engine evaluation failed", zap.Error(err))
		return nil, err
	}

	e.logger.Debug("CQL Engine completed",
		zap.Int("facts_count", len(cqlResult.ClinicalFacts)),
		zap.Int64("cql_time_ms", cqlResult.ExecutionTimeMs),
	)

	// Step 2: Measure Engine evaluates care gaps using facts
	// EvaluateWithFacts takes context, execCtx, and CQL facts
	measureResult, err := e.measureEngine.EvaluateWithFacts(ctx, execCtx, cqlResult.ClinicalFacts)
	if err != nil {
		e.logger.Error("Measure Engine evaluation failed", zap.Error(err))
		return nil, err
	}

	e.logger.Debug("Measure Engine completed",
		zap.Int("measure_count", len(measureResult.MeasureResults)),
		zap.Int64("measure_time_ms", measureResult.ExecutionTimeMs),
	)

	result := &EvaluationResult{
		ClinicalFacts:        cqlResult.ClinicalFacts,
		MeasureResults:       measureResult.MeasureResults,
		ExecutionTimeMs:      time.Since(startTime).Milliseconds(),
		CQLEngineVersion:     "1.0.0",
		MeasureEngineVersion: "1.0.0",
	}

	return result, nil
}

// EvaluateClinicalFacts runs only the CQL Engine for fact determination.
// Use this when you need clinical truths without care gap analysis.
func (e *Executor) EvaluateClinicalFacts(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
) ([]contracts.ClinicalFact, error) {
	result, err := e.cqlEngine.Evaluate(ctx, execCtx)
	if err != nil {
		return nil, err
	}
	return result.ClinicalFacts, nil
}

// EvaluateMeasures runs only the Measure Engine with pre-computed facts.
// Use this when clinical facts are already available.
func (e *Executor) EvaluateMeasures(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
	facts []contracts.ClinicalFact,
) ([]contracts.MeasureResult, error) {
	// Use EvaluateWithFacts which takes 3 arguments: ctx, execCtx, and facts
	result, err := e.measureEngine.EvaluateWithFacts(ctx, execCtx, facts)
	if err != nil {
		return nil, err
	}
	return result.MeasureResults, nil
}

// EvaluateSingleMeasure evaluates a specific measure by ID.
func (e *Executor) EvaluateSingleMeasure(
	ctx context.Context,
	execCtx *contracts.ClinicalExecutionContext,
	measureID string,
) (*contracts.MeasureResult, error) {
	// First get clinical facts
	cqlResult, err := e.cqlEngine.Evaluate(ctx, execCtx)
	if err != nil {
		return nil, err
	}

	// Get the specific evaluator and evaluate directly
	evaluator, exists := e.measureEngine.GetEvaluator(measureID)
	if !exists {
		return nil, nil // Measure not found
	}

	// Inject facts for potential lookup during evaluation
	e.measureEngine.SetClinicalFacts(cqlResult.ClinicalFacts)

	// Execute the pure function evaluator
	measureResult := evaluator.Evaluate(execCtx)
	return &measureResult, nil
}

// GetAvailableFacts returns the list of clinical facts the CQL Engine can evaluate.
func (e *Executor) GetAvailableFacts() []string {
	return e.cqlEngine.AvailableFacts()
}

// GetAvailableMeasures returns the list of quality measures the Measure Engine supports.
func (e *Executor) GetAvailableMeasures() []string {
	return e.measureEngine.AvailableMeasures()
}

// GetFactByID returns a specific fact evaluator.
func (e *Executor) GetFactByID(factID string) (engines.FactEvaluator, bool) {
	return e.cqlEngine.GetFactEvaluator(factID)
}
