package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// evaluateWithSeries is an internal test-only method on DeteriorationNodeEngine.
// It runs the full evaluation pipeline but accepts an injected time series and
// error instead of calling kb26Client.GetVariableHistory. This avoids the need
// for a real KB-26 HTTP server in unit tests.
// ---------------------------------------------------------------------------

func (e *DeteriorationNodeEngine) evaluateWithSeries(
	ctx context.Context,
	nodeID, patientID, stratumLabel string,
	cascadeCtx *CascadeContext,
	series []models.TimeSeriesPoint,
	seriesErr error,
) (*models.ClinicalSignalEvent, error) {
	// Step 1: Load node definition.
	node := e.loader.Get(nodeID)
	if node == nil {
		return nil, fmt.Errorf("deterioration engine: node %q not found", nodeID)
	}

	// Step 2: Resolve patient data.
	resolved, err := e.resolver.Resolve(ctx, patientID, node.RequiredInputs, node.AggregatedInputs)
	if err != nil {
		return nil, fmt.Errorf("deterioration engine: resolve data for node %s: %w", nodeID, err)
	}

	// Step 3: Check DataSufficiency.
	if resolved.Sufficiency == models.DataInsufficient {
		switch node.InsufficientData.Action {
		case "FLAG_FOR_REVIEW":
			event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
			if err2 := e.persist(ctx, event); err2 != nil {
				e.log.Warn("persist failed for insufficient event", zap.Error(err2))
			}
			return event, nil
		case "USE_SNAPSHOT":
			// Continue evaluation with whatever partial data exists.
		default:
			return nil, nil
		}
	}

	// Step 4: Build working fields map and merge cascade context signals.
	fields := copyFields(resolved.Fields)
	if cascadeCtx != nil {
		for ref, score := range cascadeCtx.PMSignals {
			fields[ref] = score
		}
		for ref, score := range cascadeCtx.MDSignals {
			fields[ref] = score
		}
	}

	// Step 5: Trajectory computation using injected series.
	var trajectoryRate float64
	var trajectoryConfidence float64
	trajectoryComputed := false

	if node.Trajectory != nil {
		if seriesErr != nil {
			if node.InsufficientData.Action == "FLAG_FOR_REVIEW" {
				event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
				if err2 := e.persist(ctx, event); err2 != nil {
					e.log.Warn("persist failed", zap.Error(err2))
				}
				return event, nil
			}
			// USE_SNAPSHOT: continue without trajectory.
		} else {
			result, err := e.trajectory.Compute(series, *node.Trajectory)
			if err != nil {
				if node.InsufficientData.Action == "FLAG_FOR_REVIEW" {
					event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
					if err2 := e.persist(ctx, event); err2 != nil {
						e.log.Warn("persist failed", zap.Error(err2))
					}
					return event, nil
				}
				// USE_SNAPSHOT: continue without trajectory.
			} else {
				trajectoryRate = result.Slope
				trajectoryConfidence = result.RSquared
				trajectoryComputed = true
				fields["rate_of_change"] = trajectoryRate
				fields["trajectory_confidence"] = trajectoryConfidence
			}
		}
	}

	// Step 6: Evaluate computed_fields.
	for _, cf := range node.ComputedFields {
		val, err := e.evaluator.EvaluateNumeric(cf.Formula, fields)
		if err != nil {
			e.log.Warn("computed field evaluation failed",
				zap.String("field", cf.Name),
				zap.Error(err),
			)
			continue
		}
		fields[cf.Name] = val
	}

	// Step 7: Evaluate computed_field_variants (first match per name wins).
	resolvedVariantNames := make(map[string]bool)
	for _, cfv := range node.ComputedFieldVariants {
		if resolvedVariantNames[cfv.Name] {
			continue
		}
		matched := true
		if cfv.Condition != "" {
			matched, err = e.evaluator.EvaluateBool(cfv.Condition, fields)
			if err != nil {
				e.log.Warn("computed field variant condition failed",
					zap.String("name", cfv.Name),
					zap.Error(err),
				)
				continue
			}
		}
		if !matched {
			continue
		}
		val, err := e.evaluator.EvaluateNumeric(cfv.Formula, fields)
		if err != nil {
			e.log.Warn("computed field variant formula failed",
				zap.String("name", cfv.Name),
				zap.Error(err),
			)
			continue
		}
		fields[cfv.Name] = val
		resolvedVariantNames[cfv.Name] = true
	}

	// Step 8: Evaluate thresholds (first match wins).
	var matchedThreshold *models.ThresholdDef
	for i := range node.Thresholds {
		th := &node.Thresholds[i]
		if th.Condition == "" {
			matchedThreshold = th
			break
		}
		matched, err := e.evaluator.EvaluateBool(th.Condition, fields)
		if err != nil {
			e.log.Warn("threshold condition evaluation failed",
				zap.String("signal", th.Signal),
				zap.Error(err),
			)
			continue
		}
		if matched {
			matchedThreshold = th
			break
		}
	}

	// Step 9: Compute projections.
	var projection *models.ThresholdProjection
	if trajectoryComputed && matchedThreshold != nil && len(node.Projections) > 0 {
		projDef := &node.Projections[0]
		if trajectoryConfidence >= projDef.ConfidenceRequired {
			currentValue := fields[node.StateVariable]
			projectedDate, err := e.trajectory.Project(currentValue, trajectoryRate, projDef.Threshold)
			if err == nil {
				projection = &models.ThresholdProjection{
					ThresholdName:  projDef.Name,
					CurrentValue:   currentValue,
					ThresholdValue: projDef.Threshold,
					ProjectedDate:  *projectedDate,
					Confidence:     trajectoryConfidence,
				}
			}
		}
	}

	// Step 10: Build event.
	event := e.buildEvent(node, patientID, stratumLabel, resolved, matchedThreshold, projection,
		trajectoryRate, trajectoryComputed, cascadeCtx, fields)

	// Step 11: Persist (no-op with db=nil in tests).
	if err := e.persist(ctx, event); err != nil {
		e.log.Warn("persist failed", zap.String("event_id", event.EventID), zap.Error(err))
	}

	return event, nil
}

// ---------------------------------------------------------------------------
// buildDeteriorationEngine creates a DeteriorationNodeEngine with db=nil for tests.
// ---------------------------------------------------------------------------

func buildDeteriorationEngine(
	node *models.DeteriorationNodeDefinition,
	resolver DataResolver,
) *DeteriorationNodeEngine {
	loader := NewDeteriorationNodeLoader("", testLogger())
	if node != nil {
		loader.mu.Lock()
		loader.nodes[node.NodeID] = node
		loader.mu.Unlock()
	}
	return NewDeteriorationNodeEngine(
		loader,
		resolver,
		NewTrajectoryComputer(testLogger()),
		nil, // kb26Client: nil for tests (series injected via evaluateWithSeries)
		NewExpressionEvaluator(),
		nil, // db: nil (no persistence in tests)
		testLogger(),
	)
}

// ---------------------------------------------------------------------------
// Test node builders
// ---------------------------------------------------------------------------

// testMD01ISDeclineNode returns a DeteriorationNodeDefinition simulating MD-01.
// Field "is_value" is used for the IS scalar (to avoid uppercase/lowercase conflicts).
func testMD01ISDeclineNode() *models.DeteriorationNodeDefinition {
	return &models.DeteriorationNodeDefinition{
		NodeID:        "MD-01",
		Version:       "1.0.0",
		Type:          "DETERIORATION",
		TitleEN:       "Insulin Sensitivity Decline",
		StateVariable: "is_value", // field name in resolved fields map
		Trajectory: &models.TrajectoryConfig{
			Method:        "LINEAR_REGRESSION",
			WindowDays:    90,
			MinDataPoints: 4,
			RateUnit:      "per_month",
			DataSource:    "KB-26",
		},
		Thresholds: []models.ThresholdDef{
			{
				Signal:            "IS_CRITICAL_DECLINE",
				Condition:         "rate_of_change < -0.05 AND is_value < 0.30",
				Severity:          "CRITICAL",
				Trajectory:        "DECLINING",
				MCUGateSuggestion: "PAUSE",
				Actions: []models.RecommendedAction{
					{ActionID: "ACT-MD01-01", Type: "CLINICAL_REVIEW", Description: "Review IS trajectory", Urgency: "HIGH"},
				},
			},
			{
				Signal:            "IS_STABLE",
				Condition:         "",
				Severity:          "NONE",
				Trajectory:        "STABLE",
				MCUGateSuggestion: "SAFE",
			},
		},
		Projections: []models.ProjectionDef{
			{
				Name:               "IS_CRITICAL_THRESHOLD",
				Variable:           "is_value",
				Threshold:          0.20,
				Method:             "LINEAR_EXTRAPOLATION",
				ConfidenceRequired: 0.70,
			},
		},
		InsufficientData: models.InsufficientDataPolicy{
			Action: "USE_SNAPSHOT",
		},
	}
}

// testMD04CompositeNode returns a node that aggregates PM signals via computed_field_variants.
// Uses underscore-based signal keys (e.g. "pm_04") because the expression evaluator
// treats hyphen as arithmetic subtraction.
func testMD04CompositeNode() *models.DeteriorationNodeDefinition {
	return &models.DeteriorationNodeDefinition{
		NodeID:  "MD-04",
		Version: "1.0.0",
		Type:    "DETERIORATION",
		TitleEN: "Composite Metabolic Score",
		ComputedFieldVariants: []models.ComputedFieldVariant{
			{
				// Variant 1: both pm_04 and pm_05 available.
				Condition: "pm_04 > 0 AND pm_05 > 0",
				Name:      "composite_score",
				Formula:   "pm_04 * 0.6 + pm_05 * 0.4",
			},
			{
				// Variant 2: only pm_04 available.
				Condition: "pm_04 > 0",
				Name:      "composite_score",
				Formula:   "pm_04",
			},
			{
				// Catch-all.
				Condition: "",
				Name:      "composite_score",
				Formula:   "0",
			},
		},
		Thresholds: []models.ThresholdDef{
			{
				Signal:            "COMPOSITE_CRITICAL",
				Condition:         "composite_score >= 2.0",
				Severity:          "CRITICAL",
				MCUGateSuggestion: "PAUSE",
			},
			{
				Signal:            "COMPOSITE_MODERATE",
				Condition:         "composite_score >= 1.0",
				Severity:          "MODERATE",
				MCUGateSuggestion: "MODIFY",
			},
			{
				Signal:            "COMPOSITE_STABLE",
				Condition:         "",
				Severity:          "NONE",
				MCUGateSuggestion: "SAFE",
			},
		},
		InsufficientData: models.InsufficientDataPolicy{
			Action: "USE_SNAPSHOT",
		},
	}
}

// testMD06HaltGateNode simulates MD-06, which is the only node that emits HALT.
// Uses underscore-based MD signal keys because the expression evaluator treats
// hyphen as subtraction.
func testMD06HaltGateNode() *models.DeteriorationNodeDefinition {
	return &models.DeteriorationNodeDefinition{
		NodeID:  "MD-06",
		Version: "1.0.0",
		Type:    "DETERIORATION",
		TitleEN: "Multi-System Failure Gate",
		ComputedFieldVariants: []models.ComputedFieldVariant{
			{
				Condition: "",
				Name:      "multi_system_score",
				Formula:   "md_01 + md_02",
			},
		},
		Thresholds: []models.ThresholdDef{
			{
				Signal:            "MULTI_SYSTEM_HALT",
				Condition:         "multi_system_score >= 4.0",
				Severity:          "CRITICAL",
				MCUGateSuggestion: "HALT",
			},
			{
				Signal:            "MULTI_SYSTEM_STABLE",
				Condition:         "",
				Severity:          "NONE",
				MCUGateSuggestion: "SAFE",
			},
		},
		InsufficientData: models.InsufficientDataPolicy{
			Action: "USE_SNAPSHOT",
		},
	}
}

// buildISTimeSeries generates a time series for IS values with a given slope per month.
// startIS is the IS value at t=0; slopePerMonth is the per-month change.
// nPoints points are spaced 15 days apart.
func buildISTimeSeries(startIS, slopePerMonth float64, nPoints int) []models.TimeSeriesPoint {
	series := make([]models.TimeSeriesPoint, nPoints)
	base := time.Now().UTC().Add(-time.Duration(nPoints*15) * 24 * time.Hour)
	for i := 0; i < nPoints; i++ {
		monthsElapsed := float64(i*15) / 30.0
		series[i] = models.TimeSeriesPoint{
			Timestamp: base.Add(time.Duration(i*15) * 24 * time.Hour),
			Value:     startIS + slopePerMonth*monthsElapsed,
		}
	}
	return series
}

// resolvedDataFor builds a ResolvedData with SUFFICIENT or INSUFFICIENT sufficiency.
func resolvedDataFor(fields map[string]float64) *models.ResolvedData {
	suf := models.DataSufficient
	if len(fields) == 0 {
		suf = models.DataInsufficient
	}
	return &models.ResolvedData{
		Fields:      fields,
		Sufficiency: suf,
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_CriticalDecline
// IS slope=-0.10/month, current IS=0.25
// Condition: rate_of_change < -0.05 AND is < 0.30 → IS_CRITICAL_DECLINE, CRITICAL, PAUSE
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_CriticalDecline(t *testing.T) {
	// 6 points at 15-day intervals with slope -0.10/month.
	// Start=0.50, after 2.5 months IS ≈ 0.50 - 0.25 = 0.25.
	series := buildISTimeSeries(0.50, -0.10, 6)
	currentIS := series[len(series)-1].Value // ≈ 0.25

	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{
			"is_value": currentIS,
		}),
	}

	node := testMD01ISDeclineNode()
	eng := buildDeteriorationEngine(node, resolver)

	event, err := eng.evaluateWithSeries(context.Background(), "MD-01", "patient-d001", "T2D", nil, series, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.DeteriorationSignal == nil {
		t.Fatal("DeteriorationSignal must not be nil")
	}

	det := event.DeteriorationSignal
	if det.Signal != "IS_CRITICAL_DECLINE" {
		t.Errorf("Signal: expected IS_CRITICAL_DECLINE, got %q", det.Signal)
	}
	if det.Severity != "CRITICAL" {
		t.Errorf("Severity: expected CRITICAL, got %q", det.Severity)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "PAUSE" {
		t.Errorf("MCUGateSuggestion: expected PAUSE, got %v", event.MCUGateSuggestion)
	}
	if det.RateOfChange >= -0.05 {
		t.Errorf("RateOfChange should be < -0.05, got %.4f", det.RateOfChange)
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_StableTrajectory
// IS slope=0.01/month → STABLE threshold matches.
// Expected: IS_STABLE, NONE, SAFE, no actions
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_StableTrajectory(t *testing.T) {
	series := buildISTimeSeries(0.50, 0.01, 6) // gently rising IS
	currentIS := series[len(series)-1].Value

	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{
			"is_value": currentIS,
		}),
	}

	node := testMD01ISDeclineNode()
	eng := buildDeteriorationEngine(node, resolver)

	event, err := eng.evaluateWithSeries(context.Background(), "MD-01", "patient-d002", "T2D", nil, series, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.DeteriorationSignal == nil {
		t.Fatal("DeteriorationSignal must not be nil")
	}

	det := event.DeteriorationSignal
	// rate_of_change ≈ +0.01 → first threshold "rate_of_change < -0.05" fails.
	// Catch-all IS_STABLE matches.
	if det.Signal != "IS_STABLE" {
		t.Errorf("Signal: expected IS_STABLE, got %q", det.Signal)
	}
	if det.Severity != "NONE" {
		t.Errorf("Severity: expected NONE, got %q", det.Severity)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "SAFE" {
		t.Errorf("MCUGateSuggestion: expected SAFE, got %v", event.MCUGateSuggestion)
	}
	if len(event.RecommendedActions) != 0 {
		t.Errorf("expected no RecommendedActions for STABLE, got %d", len(event.RecommendedActions))
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_Projection
// IS=0.35, slope=-0.05/month → projected crossing of 0.20 ~3 months out.
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_Projection(t *testing.T) {
	// IS starts at 0.50, slope=-0.05/month → after 2.5 months IS ≈ 0.375.
	series := buildISTimeSeries(0.50, -0.05, 6)
	currentIS := series[len(series)-1].Value // ≈ 0.375

	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{
			"is_value": currentIS,
		}),
	}

	// Build node with a threshold that fires for is_value < 0.40 AND declining rate.
	node := testMD01ISDeclineNode()
	node.Thresholds = []models.ThresholdDef{
		{
			Signal:            "IS_DECLINING",
			Condition:         "rate_of_change < -0.04 AND is_value < 0.40",
			Severity:          "MODERATE",
			Trajectory:        "DECLINING",
			MCUGateSuggestion: "MODIFY",
		},
		{
			Signal:            "IS_STABLE",
			Condition:         "",
			Severity:          "NONE",
			Trajectory:        "STABLE",
			MCUGateSuggestion: "SAFE",
		},
	}
	// Projection: threshold=0.20, confidence_required=0.0 (always project).
	node.Projections = []models.ProjectionDef{
		{
			Name:               "IS_CRITICAL_THRESHOLD",
			Variable:           "is_value",
			Threshold:          0.20,
			Method:             "LINEAR_EXTRAPOLATION",
			ConfidenceRequired: 0.0,
		},
	}

	eng := buildDeteriorationEngine(node, resolver)

	event, err := eng.evaluateWithSeries(context.Background(), "MD-01", "patient-d003", "T2D", nil, series, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if event.ProjectedThreshold == nil {
		t.Fatal("expected ProjectedThreshold to be set")
	}

	proj := event.ProjectedThreshold
	if proj.ThresholdName != "IS_CRITICAL_THRESHOLD" {
		t.Errorf("ProjectedThreshold.ThresholdName: expected IS_CRITICAL_THRESHOLD, got %q", proj.ThresholdName)
	}
	if proj.ThresholdValue != 0.20 {
		t.Errorf("ProjectedThreshold.ThresholdValue: expected 0.20, got %.4f", proj.ThresholdValue)
	}

	// The projected date should be at least 30 days in the future.
	daysUntil := time.Until(proj.ProjectedDate).Hours() / 24
	if daysUntil < 30 {
		t.Errorf("ProjectedDate should be at least 30 days out, got %.1f days", daysUntil)
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_InsufficientHistory
// Only 3 data points (node requires MinDataPoints=4).
// Policy=USE_SNAPSHOT → continue without trajectory; event still built.
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_InsufficientHistory(t *testing.T) {
	// Only 3 data points — node requires 4.
	series := buildISTimeSeries(0.50, -0.08, 3)

	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{
			"is_value": 0.50,
		}),
	}

	node := testMD01ISDeclineNode() // USE_SNAPSHOT policy
	eng := buildDeteriorationEngine(node, resolver)

	event, err := eng.evaluateWithSeries(context.Background(), "MD-01", "patient-d004", "T2D", nil, series, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event (USE_SNAPSHOT continues without trajectory)")
	}
	if event.DeteriorationSignal == nil {
		t.Fatal("DeteriorationSignal must not be nil")
	}

	det := event.DeteriorationSignal
	// No trajectory computed → RateOfChange should be zero.
	if det.RateOfChange != 0 {
		t.Errorf("RateOfChange should be 0 when trajectory not computed, got %.4f", det.RateOfChange)
	}

	// No projection without trajectory.
	if event.ProjectedThreshold != nil {
		t.Error("ProjectedThreshold should be nil when trajectory not computed")
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_MD04CompositeScore
// CascadeContext: PM-04=2.0, PM-05=1.0
// composite_score = 2.0*0.6 + 1.0*0.4 = 1.6
// Threshold: composite_score >= 1.0 → COMPOSITE_MODERATE, MODIFY
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_MD04CompositeScore(t *testing.T) {
	// Empty fields → DataInsufficient but USE_SNAPSHOT continues.
	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{}),
	}

	node := testMD04CompositeNode()
	eng := buildDeteriorationEngine(node, resolver)

	// Use underscore-based keys matching the node's expression field references.
	cascadeCtx := &CascadeContext{
		PMSignals: map[string]float64{
			"pm_04": 2.0,
			"pm_05": 1.0,
		},
	}

	event, err := eng.evaluateWithSeries(context.Background(), "MD-04", "patient-d005", "T2D", cascadeCtx, nil, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.DeteriorationSignal == nil {
		t.Fatal("DeteriorationSignal must not be nil")
	}

	det := event.DeteriorationSignal
	// composite_score = 2.0*0.6 + 1.0*0.4 = 1.6
	// Threshold[0]: composite_score >= 2.0 → 1.6 >= 2.0 → false
	// Threshold[1]: composite_score >= 1.0 → 1.6 >= 1.0 → true → COMPOSITE_MODERATE
	if det.Signal != "COMPOSITE_MODERATE" {
		t.Errorf("Signal: expected COMPOSITE_MODERATE, got %q", det.Signal)
	}
	if det.Severity != "MODERATE" {
		t.Errorf("Severity: expected MODERATE, got %q", det.Severity)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "MODIFY" {
		t.Errorf("MCUGateSuggestion: expected MODIFY, got %v", event.MCUGateSuggestion)
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_MD06HaltGate
// MD-06 is the only node that emits HALT.
// MDSignals: MD-01=3.0, MD-02=2.0 → multi_system_score = 5.0 >= 4.0 → HALT
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_MD06HaltGate(t *testing.T) {
	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{}),
	}

	node := testMD06HaltGateNode()
	eng := buildDeteriorationEngine(node, resolver)

	// Use underscore-based keys matching the node's expression field references.
	cascadeCtx := &CascadeContext{
		MDSignals: map[string]float64{
			"md_01": 3.0,
			"md_02": 2.0,
		},
	}

	event, err := eng.evaluateWithSeries(context.Background(), "MD-06", "patient-d006", "T2D", cascadeCtx, nil, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.DeteriorationSignal == nil {
		t.Fatal("DeteriorationSignal must not be nil")
	}

	det := event.DeteriorationSignal
	// multi_system_score = 3.0 + 2.0 = 5.0 >= 4.0 → MULTI_SYSTEM_HALT
	if det.Signal != "MULTI_SYSTEM_HALT" {
		t.Errorf("Signal: expected MULTI_SYSTEM_HALT, got %q", det.Signal)
	}
	if event.MCUGateSuggestion == nil || *event.MCUGateSuggestion != "HALT" {
		t.Errorf("MCUGateSuggestion: expected HALT, got %v", event.MCUGateSuggestion)
	}
	if det.Severity != "CRITICAL" {
		t.Errorf("Severity: expected CRITICAL, got %q", det.Severity)
	}
}

// ---------------------------------------------------------------------------
// TestDeteriorationEngine_ContributingSignals
// CascadeContext with PM signals → event.ContributingSignals populated.
// ---------------------------------------------------------------------------

func TestDeteriorationEngine_ContributingSignals(t *testing.T) {
	series := buildISTimeSeries(0.60, 0.01, 6) // stable IS
	currentIS := series[len(series)-1].Value

	resolver := &mockDataResolver{
		resolvedData: resolvedDataFor(map[string]float64{
			"is_value": currentIS,
		}),
	}

	node := testMD01ISDeclineNode()
	eng := buildDeteriorationEngine(node, resolver)

	cascadeCtx := &CascadeContext{
		PMSignals: map[string]float64{
			"pm_04": 1.0,
			"pm_07": 2.0,
		},
	}

	event, err := eng.evaluateWithSeries(context.Background(), "MD-01", "patient-d007", "T2D", cascadeCtx, series, nil)
	if err != nil {
		t.Fatalf("Evaluate() unexpected error: %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if len(event.ContributingSignals) == 0 {
		t.Fatal("expected ContributingSignals to be populated from CascadeContext")
	}

	// Both PM signal keys should be present.
	sigMap := make(map[string]bool)
	for _, s := range event.ContributingSignals {
		sigMap[s] = true
	}
	if !sigMap["pm_04"] {
		t.Error("ContributingSignals should include pm_04")
	}
	if !sigMap["pm_07"] {
		t.Error("ContributingSignals should include pm_07")
	}
}
