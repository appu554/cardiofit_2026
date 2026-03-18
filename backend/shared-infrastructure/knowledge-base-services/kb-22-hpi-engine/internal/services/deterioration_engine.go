package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// CascadeContext
// ---------------------------------------------------------------------------

// CascadeContext carries PM classification results for composite MD nodes
// (MD-04, MD-05, MD-06). PMSignals maps PM node_id → severity score
// (0=NONE, 1=MILD, 2=MODERATE, 3=CRITICAL). MDSignals maps MD node_id →
// severity score for the MD→MD-06 pass.
type CascadeContext struct {
	PMSignals map[string]float64 // node_id → severity score
	MDSignals map[string]float64 // node_id → severity score (for MD→MD-06 pass)
}

// ---------------------------------------------------------------------------
// DeteriorationNodeEngine
// ---------------------------------------------------------------------------

// DeteriorationNodeEngine evaluates Layer-3 MD deterioration nodes.
// Given a nodeID, patientID, stratumLabel and an optional CascadeContext it:
//  1. Loads the DeteriorationNodeDefinition via loader.
//  2. Resolves patient data via resolver.
//  3. Handles insufficient data according to the node's InsufficientData policy.
//  4. Merges cascade PM/MD severity scores into the fields map.
//  5. Runs trajectory computation via kb26Client + TrajectoryComputer when configured.
//  6. Evaluates ComputedFields and ComputedFieldVariants with the expression evaluator.
//  7. Iterates Thresholds top-to-bottom (first match wins).
//  8. Computes projections if configured.
//  9. Builds and optionally persists a ClinicalSignalEvent.
type DeteriorationNodeEngine struct {
	loader     *DeteriorationNodeLoader
	resolver   DataResolver
	trajectory *TrajectoryComputer
	kb26Client *KB26Client
	evaluator  *ExpressionEvaluator
	db         *gorm.DB
	log        *zap.Logger
}

// NewDeteriorationNodeEngine creates a DeteriorationNodeEngine.
// db may be nil; when nil, persistence is skipped (useful in tests).
// kb26Client may be nil when trajectory computation is not needed.
func NewDeteriorationNodeEngine(
	loader *DeteriorationNodeLoader,
	resolver DataResolver,
	trajectory *TrajectoryComputer,
	kb26Client *KB26Client,
	evaluator *ExpressionEvaluator,
	db *gorm.DB,
	log *zap.Logger,
) *DeteriorationNodeEngine {
	return &DeteriorationNodeEngine{
		loader:     loader,
		resolver:   resolver,
		trajectory: trajectory,
		kb26Client: kb26Client,
		evaluator:  evaluator,
		db:         db,
		log:        log.With(zap.String("component", "deterioration-engine")),
	}
}

// Evaluate runs the full MD node evaluation pipeline for the given node and patient.
// Returns (*ClinicalSignalEvent, nil) on success.
// Returns (nil, error) if the node is not found or a resolver/evaluator error occurs.
func (e *DeteriorationNodeEngine) Evaluate(
	ctx context.Context,
	nodeID, patientID, stratumLabel string,
	cascadeCtx *CascadeContext,
) (*models.ClinicalSignalEvent, error) {
	// Step 1: Load node definition.
	node := e.loader.Get(nodeID)
	if node == nil {
		return nil, fmt.Errorf("deterioration engine: node %q not found", nodeID)
	}

	e.log.Info("evaluating deterioration node",
		zap.String("node_id", nodeID),
		zap.String("patient_id", patientID),
		zap.String("stratum", stratumLabel),
	)

	// Step 2: Resolve patient data.
	resolved, err := e.resolver.Resolve(ctx, patientID, node.RequiredInputs, node.AggregatedInputs)
	if err != nil {
		return nil, fmt.Errorf("deterioration engine: resolve data for node %s: %w", nodeID, err)
	}

	// Step 3: Check DataSufficiency.
	if resolved.Sufficiency == models.DataInsufficient {
		switch node.InsufficientData.Action {
		case "FLAG_FOR_REVIEW":
			e.log.Info("emitting insufficient data flag event",
				zap.String("node_id", nodeID),
				zap.String("patient_id", patientID),
			)
			event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
			if err := e.persist(ctx, event); err != nil {
				e.log.Warn("persist failed for insufficient event", zap.Error(err))
			}
			return event, nil

		case "USE_SNAPSHOT":
			// Continue with partial data — fall through to evaluation below.
			e.log.Info("using snapshot for insufficient data",
				zap.String("node_id", nodeID),
				zap.String("patient_id", patientID),
			)

		default:
			// SKIP or unknown: return nil event (no emission).
			e.log.Info("skipping node evaluation: insufficient data",
				zap.String("node_id", nodeID),
				zap.String("patient_id", patientID),
				zap.String("action", node.InsufficientData.Action),
			)
			return nil, nil
		}
	}

	// Step 4: Build working fields map and merge cascade context signals.
	fields := copyFields(resolved.Fields)

	if cascadeCtx != nil {
		for nodeRef, score := range cascadeCtx.PMSignals {
			fields[nodeRef] = score
		}
		for nodeRef, score := range cascadeCtx.MDSignals {
			fields[nodeRef] = score
		}
	}

	// Step 5: Trajectory computation (if node has trajectory config).
	var trajectoryRate float64
	var trajectoryConfidence float64
	trajectoryComputed := false

	if node.Trajectory != nil && e.kb26Client != nil {
		series, err := e.kb26Client.GetVariableHistory(ctx, patientID, node.StateVariable, node.Trajectory.WindowDays)
		if err != nil {
			e.log.Warn("kb26 GetVariableHistory failed, applying insufficient data policy",
				zap.String("node_id", nodeID),
				zap.String("variable", node.StateVariable),
				zap.Error(err),
			)
			// Apply insufficient data policy for trajectory failure.
			if node.InsufficientData.Action == "FLAG_FOR_REVIEW" {
				event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
				if err2 := e.persist(ctx, event); err2 != nil {
					e.log.Warn("persist failed", zap.Error(err2))
				}
				return event, nil
			}
			// USE_SNAPSHOT or default: continue without trajectory data.
		} else {
			result, err := e.trajectory.Compute(series, *node.Trajectory)
			if err != nil {
				e.log.Warn("trajectory computation failed, applying insufficient data policy",
					zap.String("node_id", nodeID),
					zap.Error(err),
				)
				// Insufficient data points for trajectory.
				if node.InsufficientData.Action == "FLAG_FOR_REVIEW" {
					event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
					if err2 := e.persist(ctx, event); err2 != nil {
						e.log.Warn("persist failed", zap.Error(err2))
					}
					return event, nil
				}
				// USE_SNAPSHOT: continue without trajectory data.
			} else {
				trajectoryRate = result.Slope
				trajectoryConfidence = result.RSquared
				trajectoryComputed = true

				fields["rate_of_change"] = trajectoryRate
				fields["trajectory_confidence"] = trajectoryConfidence

				e.log.Info("trajectory computed",
					zap.String("node_id", nodeID),
					zap.Float64("rate_of_change", trajectoryRate),
					zap.Float64("confidence", trajectoryConfidence),
				)
			}
		}
	}

	// Step 6: Evaluate computed_fields (standard) via EvaluateNumeric.
	for _, cf := range node.ComputedFields {
		val, err := e.evaluator.EvaluateNumeric(cf.Formula, fields)
		if err != nil {
			e.log.Warn("computed field evaluation failed",
				zap.String("field", cf.Name),
				zap.String("formula", cf.Formula),
				zap.Error(err),
			)
			continue
		}
		fields[cf.Name] = val
	}

	// Step 7: Evaluate computed_field_variants top-to-bottom (first match per name wins).
	// Tracks which field names have already been resolved by a matched variant.
	resolvedVariantNames := make(map[string]bool)
	for _, cfv := range node.ComputedFieldVariants {
		// Skip if we already have a match for this field name.
		if resolvedVariantNames[cfv.Name] {
			continue
		}

		// Evaluate condition (empty condition always matches).
		matched := true
		if cfv.Condition != "" {
			var err error
			matched, err = e.evaluator.EvaluateBool(cfv.Condition, fields)
			if err != nil {
				e.log.Warn("computed field variant condition evaluation failed",
					zap.String("name", cfv.Name),
					zap.String("condition", cfv.Condition),
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
			e.log.Warn("computed field variant formula evaluation failed",
				zap.String("name", cfv.Name),
				zap.String("formula", cfv.Formula),
				zap.Error(err),
			)
			continue
		}
		fields[cfv.Name] = val
		resolvedVariantNames[cfv.Name] = true
	}

	// Step 8: Evaluate thresholds top-to-bottom (first match wins).
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
				zap.String("condition", th.Condition),
				zap.Error(err),
			)
			continue
		}
		if matched {
			matchedThreshold = th
			break
		}
	}

	if matchedThreshold == nil {
		e.log.Warn("no threshold matched for node",
			zap.String("node_id", nodeID),
			zap.String("patient_id", patientID),
		)
	}

	// Step 9: Compute projections if configured.
	var projection *models.ThresholdProjection
	if trajectoryComputed && matchedThreshold != nil && len(node.Projections) > 0 {
		projDef := &node.Projections[0]

		// Only project if trajectory confidence meets the requirement.
		if trajectoryConfidence >= projDef.ConfidenceRequired {
			currentValue := fields[node.StateVariable]
			projectedDate, err := e.trajectory.Project(currentValue, trajectoryRate, projDef.Threshold)
			if err != nil {
				e.log.Info("projection not computable",
					zap.String("node_id", nodeID),
					zap.Error(err),
				)
			} else {
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

	// Step 10: Build ClinicalSignalEvent.
	event := e.buildEvent(node, patientID, stratumLabel, resolved, matchedThreshold, projection,
		trajectoryRate, trajectoryComputed, cascadeCtx, fields)

	// Step 11: Persist (skipped when db is nil, e.g. in tests).
	if err := e.persist(ctx, event); err != nil {
		e.log.Warn("persist failed", zap.String("event_id", event.EventID), zap.Error(err))
	}

	// Step 12: Return event.
	return event, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// buildEvent constructs a ClinicalSignalEvent from the evaluated results.
func (e *DeteriorationNodeEngine) buildEvent(
	node *models.DeteriorationNodeDefinition,
	patientID, stratumLabel string,
	resolved *models.ResolvedData,
	matched *models.ThresholdDef,
	projection *models.ThresholdProjection,
	trajectoryRate float64,
	trajectoryComputed bool,
	cascadeCtx *CascadeContext,
	fields map[string]float64,
) *models.ClinicalSignalEvent {
	now := time.Now().UTC()
	event := &models.ClinicalSignalEvent{
		EventID:      uuid.New().String(),
		EventType:    "CLINICAL_SIGNAL",
		SignalType:   models.SignalDeteriorationSignal,
		PatientID:    patientID,
		NodeID:       node.NodeID,
		NodeVersion:  node.Version,
		StratumLabel: stratumLabel,
		EmittedAt:    now,
	}

	// Build deterioration result.
	detResult := &models.DeteriorationResult{
		StateVariable: node.StateVariable,
	}
	if trajectoryComputed {
		detResult.RateOfChange = trajectoryRate
	}
	if matched != nil {
		detResult.Signal = matched.Signal
		detResult.Severity = matched.Severity
		detResult.Trajectory = matched.Trajectory

		gate := matched.MCUGateSuggestion
		event.MCUGateSuggestion = &gate

		// Copy recommended actions from matched threshold.
		if len(matched.Actions) > 0 {
			event.RecommendedActions = make([]models.RecommendedAction, len(matched.Actions))
			copy(event.RecommendedActions, matched.Actions)
		}
	}
	event.DeteriorationSignal = detResult

	// Attach projection if computed.
	event.ProjectedThreshold = projection

	// Populate ContributingSignals from cascade context PM signals.
	if cascadeCtx != nil && len(cascadeCtx.PMSignals) > 0 {
		for nodeRef := range cascadeCtx.PMSignals {
			event.ContributingSignals = append(event.ContributingSignals, nodeRef)
		}
	}

	return event
}

// buildInsufficientEvent builds a minimal event for FLAG_FOR_REVIEW / trajectory-missing cases.
func (e *DeteriorationNodeEngine) buildInsufficientEvent(
	node *models.DeteriorationNodeDefinition,
	patientID, stratumLabel string,
	resolved *models.ResolvedData,
) *models.ClinicalSignalEvent {
	now := time.Now().UTC()
	return &models.ClinicalSignalEvent{
		EventID:      uuid.New().String(),
		EventType:    "CLINICAL_SIGNAL",
		SignalType:   models.SignalDeteriorationSignal,
		PatientID:    patientID,
		NodeID:       node.NodeID,
		NodeVersion:  node.Version,
		StratumLabel: stratumLabel,
		EmittedAt:    now,
		// Classification carries the INSUFFICIENT flag for downstream consumers.
		Classification: &models.ClassificationResult{
			DataSufficiency: string(models.DataInsufficient),
		},
	}
}

// persist writes the event to clinical_signals and upserts clinical_signals_latest.
// No-ops when e.db is nil (test mode).
func (e *DeteriorationNodeEngine) persist(ctx context.Context, event *models.ClinicalSignalEvent) error {
	if e.db == nil {
		return nil
	}

	var safeFlagsJSON []byte
	if len(event.SafetyFlags) > 0 {
		b, err := json.Marshal(event.SafetyFlags)
		if err != nil {
			return fmt.Errorf("marshal safety flags: %w", err)
		}
		safeFlagsJSON = b
	}

	mcuGate := ""
	if event.MCUGateSuggestion != nil {
		mcuGate = *event.MCUGateSuggestion
	}

	// Use Classification for data_sufficiency (set on insufficient events).
	dataSufficiency := string(models.DataSufficient)
	if event.Classification != nil {
		dataSufficiency = event.Classification.DataSufficiency
	}

	row := clinicalSignalRow{
		SignalID:        event.EventID,
		PatientID:       event.PatientID,
		NodeID:          event.NodeID,
		NodeVersion:     event.NodeVersion,
		SignalType:      string(event.SignalType),
		StratumLabel:    event.StratumLabel,
		DataSufficiency: dataSufficiency,
		SafetyFlagsJSON: safeFlagsJSON,
		MCUGateSuggestion: mcuGate,
		PublishedToKB23: false,
		EvaluatedAt:     event.EmittedAt,
	}

	if err := e.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("persist deterioration signal: %w", err)
	}

	// Upsert clinical_signals_latest.
	latest := clinicalSignalLatestRow{
		PatientID:   event.PatientID,
		NodeID:      event.NodeID,
		SignalID:    event.EventID,
		EvaluatedAt: event.EmittedAt,
	}
	if err := e.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "patient_id"}, {Name: "node_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"signal_id", "evaluated_at"}),
		}).
		Create(&latest).Error; err != nil {
		return fmt.Errorf("upsert clinical_signals_latest: %w", err)
	}

	return nil
}
