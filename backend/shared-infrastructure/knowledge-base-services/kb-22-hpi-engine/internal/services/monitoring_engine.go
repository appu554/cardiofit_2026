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
// DB models for clinical_signals and clinical_signals_latest
// ---------------------------------------------------------------------------

// clinicalSignalRow maps to the clinical_signals table (migration 006).
type clinicalSignalRow struct {
	SignalID   string    `gorm:"column:signal_id;primaryKey"`
	PatientID  string    `gorm:"column:patient_id"`
	NodeID     string    `gorm:"column:node_id"`
	NodeVersion string   `gorm:"column:node_version"`
	SignalType string    `gorm:"column:signal_type"`
	StratumLabel string  `gorm:"column:stratum_label"`

	// PM node fields
	ClassificationCategory string  `gorm:"column:classification_category"`
	ClassificationValue    float64 `gorm:"column:classification_value"`
	ClassificationUnit     string  `gorm:"column:classification_unit"`
	DataSufficiency        string  `gorm:"column:data_sufficiency"`

	// Shared
	SafetyFlagsJSON      []byte  `gorm:"column:safety_flags"`
	MCUGateSuggestion    string  `gorm:"column:mcu_gate_suggestion"`
	PublishedToKB23      bool    `gorm:"column:published_to_kb23"`
	EvaluatedAt          time.Time `gorm:"column:evaluated_at"`
}

func (clinicalSignalRow) TableName() string { return "clinical_signals" }

// clinicalSignalLatestRow maps to clinical_signals_latest (migration 006).
type clinicalSignalLatestRow struct {
	PatientID   string    `gorm:"column:patient_id;primaryKey"`
	NodeID      string    `gorm:"column:node_id;primaryKey"`
	SignalID    string    `gorm:"column:signal_id"`
	EvaluatedAt time.Time `gorm:"column:evaluated_at"`
}

func (clinicalSignalLatestRow) TableName() string { return "clinical_signals_latest" }

// ---------------------------------------------------------------------------
// MonitoringNodeEngine
// ---------------------------------------------------------------------------

// MonitoringNodeEngine evaluates Layer-2 PM monitoring nodes.
// Given a nodeID and patientID it:
//  1. Loads the MonitoringNodeDefinition via loader.
//  2. Resolves patient data via resolver.
//  3. Handles insufficient data according to the node's InsufficientData policy.
//  4. Evaluates ComputedFields with the expression evaluator.
//  5. Iterates Classifications top-to-bottom (first match wins).
//  6. Evaluates SafetyTriggers.
//  7. Builds and optionally persists a ClinicalSignalEvent.
type MonitoringNodeEngine struct {
	loader     *MonitoringNodeLoader
	resolver   DataResolver
	evaluator  *ExpressionEvaluator
	trajectory *TrajectoryComputer
	db         *gorm.DB
	log        *zap.Logger
}

// NewMonitoringNodeEngine creates a MonitoringNodeEngine.
// db may be nil; when nil, persistence is skipped (useful in tests).
func NewMonitoringNodeEngine(
	loader *MonitoringNodeLoader,
	resolver DataResolver,
	evaluator *ExpressionEvaluator,
	trajectory *TrajectoryComputer,
	db *gorm.DB,
	log *zap.Logger,
) *MonitoringNodeEngine {
	return &MonitoringNodeEngine{
		loader:     loader,
		resolver:   resolver,
		evaluator:  evaluator,
		trajectory: trajectory,
		db:         db,
		log:        log.With(zap.String("component", "monitoring-engine")),
	}
}

// Evaluate runs the full PM node evaluation pipeline for the given node and patient.
// Returns (*ClinicalSignalEvent, nil) on success.
// Returns (nil, nil) if the node's InsufficientData.Action is SKIP and data is insufficient.
// Returns (nil, error) if the node is not found or a resolver/evaluator error occurs.
func (e *MonitoringNodeEngine) Evaluate(
	ctx context.Context,
	nodeID, patientID, stratumLabel string,
) (*models.ClinicalSignalEvent, error) {
	// Step 1: Load node definition.
	node := e.loader.Get(nodeID)
	if node == nil {
		return nil, fmt.Errorf("monitoring engine: node %q not found", nodeID)
	}

	e.log.Info("evaluating monitoring node",
		zap.String("node_id", nodeID),
		zap.String("patient_id", patientID),
		zap.String("stratum", stratumLabel),
	)

	// Step 2: Resolve patient data.
	resolved, err := e.resolver.Resolve(ctx, patientID, node.RequiredInputs, node.AggregatedInputs)
	if err != nil {
		return nil, fmt.Errorf("monitoring engine: resolve data for node %s: %w", nodeID, err)
	}

	// Step 3: Check DataSufficiency.
	if resolved.Sufficiency == models.DataInsufficient {
		switch node.InsufficientData.Action {
		case "SKIP", "":
			e.log.Info("skipping node evaluation: insufficient data + SKIP policy",
				zap.String("node_id", nodeID),
				zap.String("patient_id", patientID),
			)
			return nil, nil

		case "FLAG_FOR_REVIEW":
			// Emit an event that signals insufficient data, no classification.
			e.log.Info("emitting insufficient data flag event",
				zap.String("node_id", nodeID),
				zap.String("patient_id", patientID),
			)
			event := e.buildInsufficientEvent(node, patientID, stratumLabel, resolved)
			if err := e.persist(ctx, event); err != nil {
				e.log.Warn("persist failed for insufficient event", zap.Error(err))
			}
			return event, nil

		default:
			e.log.Warn("unknown InsufficientData.Action, defaulting to SKIP",
				zap.String("action", node.InsufficientData.Action),
			)
			return nil, nil
		}
	}

	// Step 3b: If time-series data is present, compute trajectory (slope) and add to fields.
	fields := copyFields(resolved.Fields)
	for fieldName, series := range resolved.TimeSeries {
		if len(series) < 2 {
			continue
		}
		cfg := models.TrajectoryConfig{
			Method:        "LINEAR_REGRESSION",
			MinDataPoints: 2,
			RateUnit:      "per_month",
		}
		result, err := e.trajectory.Compute(series, cfg)
		if err != nil {
			e.log.Warn("trajectory computation failed",
				zap.String("field", fieldName),
				zap.Error(err),
			)
			continue
		}
		// Store slope as rate_of_change for the field.
		fields[fieldName+"_slope"] = result.Slope
		fields[fieldName+"_r2"] = result.RSquared
	}

	// Step 4: Evaluate computed_fields and add to fields map.
	for _, cf := range node.ComputedFields {
		val, err := e.evaluator.EvaluateNumeric(cf.Formula, fields)
		if err != nil {
			e.log.Warn("computed field evaluation failed",
				zap.String("field", cf.Name),
				zap.String("formula", cf.Formula),
				zap.Error(err),
			)
			// Skip this computed field; do not fail entire evaluation.
			continue
		}
		fields[cf.Name] = val
	}

	// Step 5: Iterate classifications top-to-bottom, first match wins.
	var matchedClass *models.ClassificationDef
	for i := range node.Classifications {
		cls := &node.Classifications[i]
		if cls.Condition == "" {
			// Empty condition always matches (catch-all).
			matchedClass = cls
			break
		}
		matched, err := e.evaluator.EvaluateBool(cls.Condition, fields)
		if err != nil {
			e.log.Warn("classification condition evaluation failed",
				zap.String("category", cls.Category),
				zap.String("condition", cls.Condition),
				zap.Error(err),
			)
			continue
		}
		if matched {
			matchedClass = cls
			break
		}
	}

	if matchedClass == nil {
		e.log.Warn("no classification matched for node",
			zap.String("node_id", nodeID),
			zap.String("patient_id", patientID),
		)
	}

	// Step 6: Evaluate safety_triggers.
	var safetyFlags []models.SignalSafetyFlag
	for _, trigger := range node.SafetyTriggers {
		fired, err := e.evaluator.EvaluateBool(trigger.Condition, fields)
		if err != nil {
			e.log.Warn("safety trigger evaluation failed",
				zap.String("trigger_id", trigger.ID),
				zap.String("condition", trigger.Condition),
				zap.Error(err),
			)
			continue
		}
		if fired {
			safetyFlags = append(safetyFlags, models.SignalSafetyFlag{
				FlagID:    trigger.ID,
				Severity:  trigger.Severity,
				Action:    trigger.Action,
				Condition: trigger.Condition,
			})
		}
	}

	// Step 7: Build ClinicalSignalEvent.
	event := e.buildEvent(node, patientID, stratumLabel, resolved, matchedClass, safetyFlags, fields)

	// Step 8: Persist (skipped when db is nil, e.g. in tests).
	if err := e.persist(ctx, event); err != nil {
		e.log.Warn("persist failed", zap.String("event_id", event.EventID), zap.Error(err))
	}

	// Step 9: Return event (caller handles publishing + cascade).
	return event, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// buildEvent constructs a ClinicalSignalEvent from the evaluated results.
func (e *MonitoringNodeEngine) buildEvent(
	node *models.MonitoringNodeDefinition,
	patientID, stratumLabel string,
	resolved *models.ResolvedData,
	matched *models.ClassificationDef,
	safetyFlags []models.SignalSafetyFlag,
	fields map[string]float64,
) *models.ClinicalSignalEvent {
	now := time.Now().UTC()
	event := &models.ClinicalSignalEvent{
		EventID:      uuid.New().String(),
		EventType:    "CLINICAL_SIGNAL",
		SignalType:   models.SignalMonitoringClassification,
		PatientID:    patientID,
		NodeID:       node.NodeID,
		NodeVersion:  node.Version,
		StratumLabel: stratumLabel,
		EmittedAt:    now,
		SafetyFlags:  safetyFlags,
	}

	// Build classification result.
	classResult := &models.ClassificationResult{
		DataSufficiency: string(resolved.Sufficiency),
		Confidence:      1.0, // deterministic rule-based; no probabilistic uncertainty
	}

	if matched != nil {
		classResult.Category = matched.Category

		// Use the first computed field's value as the primary value.
		if len(node.ComputedFields) > 0 {
			if v, ok := fields[node.ComputedFields[0].Name]; ok {
				classResult.Value = v
			}
		}

		gate := matched.MCUGateSuggestion
		event.MCUGateSuggestion = &gate
	}

	event.Classification = classResult
	return event
}

// buildInsufficientEvent builds a minimal event for FLAG_FOR_REVIEW insufficient-data case.
func (e *MonitoringNodeEngine) buildInsufficientEvent(
	node *models.MonitoringNodeDefinition,
	patientID, stratumLabel string,
	resolved *models.ResolvedData,
) *models.ClinicalSignalEvent {
	now := time.Now().UTC()
	return &models.ClinicalSignalEvent{
		EventID:      uuid.New().String(),
		EventType:    "CLINICAL_SIGNAL",
		SignalType:   models.SignalMonitoringClassification,
		PatientID:    patientID,
		NodeID:       node.NodeID,
		NodeVersion:  node.Version,
		StratumLabel: stratumLabel,
		EmittedAt:    now,
		Classification: &models.ClassificationResult{
			DataSufficiency: string(resolved.Sufficiency),
		},
	}
}

// persist writes the event to clinical_signals and upserts clinical_signals_latest.
// No-ops when e.db is nil (test mode).
func (e *MonitoringNodeEngine) persist(ctx context.Context, event *models.ClinicalSignalEvent) error {
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

	classCategory := ""
	classValue := 0.0
	classUnit := ""
	dataSufficiency := ""
	if event.Classification != nil {
		classCategory = event.Classification.Category
		classValue = event.Classification.Value
		classUnit = event.Classification.Unit
		dataSufficiency = event.Classification.DataSufficiency
	}

	row := clinicalSignalRow{
		SignalID:               event.EventID,
		PatientID:              event.PatientID,
		NodeID:                 event.NodeID,
		NodeVersion:            event.NodeVersion,
		SignalType:             string(event.SignalType),
		StratumLabel:           event.StratumLabel,
		ClassificationCategory: classCategory,
		ClassificationValue:    classValue,
		ClassificationUnit:     classUnit,
		DataSufficiency:        dataSufficiency,
		SafetyFlagsJSON:        safeFlagsJSON,
		MCUGateSuggestion:      mcuGate,
		PublishedToKB23:        false,
		EvaluatedAt:            event.EmittedAt,
	}

	if err := e.db.WithContext(ctx).Create(&row).Error; err != nil {
		return fmt.Errorf("persist clinical signal: %w", err)
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

// copyFields makes a shallow copy of the fields map so computed fields don't
// mutate the original ResolvedData.Fields.
func copyFields(src map[string]float64) map[string]float64 {
	dst := make(map[string]float64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

