package services

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// DeteriorationEvaluator — interface for cascade dependency on deterioration engine.
// Using an interface makes the cascade cleanly testable without a real engine.
// ---------------------------------------------------------------------------

// DeteriorationEvaluator is the subset of DeteriorationNodeEngine required by SignalCascade.
type DeteriorationEvaluator interface {
	Evaluate(ctx context.Context, nodeID, patientID, stratumLabel string, cascadeCtx *CascadeContext) (*models.ClinicalSignalEvent, error)
}

// ---------------------------------------------------------------------------
// SignalCascade
// ---------------------------------------------------------------------------

// SignalCascade coordinates the two-pass PM→MD and MD→MD-06 cascade evaluation
// described in spec Section 8.7.
//
// Pass 1: For each MD node listed in the source PM node's cascade_to field,
//
//	call DeteriorationEvaluator.Evaluate with a CascadeContext carrying the
//	PM node's severity score in PMSignals.
//
// Pass 2: If any Pass-1 result fired (non-nil event with a DeteriorationSignal or
//
//	Classification) and the firing MD node is listed in MD-06's
//	contributing_signals, evaluate MD-06 with the full CascadeContext
//	(both PM and MD signals accumulated from Pass 1).
//
// Cascade failures are non-fatal: they are logged and skipped.
type SignalCascade struct {
	pmToMD   map[string][]string // PM node_id → []MD node_ids to evaluate (from cascade_to)
	mdToMD06 map[string]bool     // MD node_ids that are in MD-06's contributing_signals

	deterEngine DeteriorationEvaluator
	log         *zap.Logger
}

// NewSignalCascade builds a SignalCascade from the loaded node definitions.
//
//   - pmToMD is built from all PM nodes' cascade_to fields.
//   - mdToMD06 is built from MD-06's contributing_signals (identified by node_id == "MD-06").
func NewSignalCascade(
	monLoader *MonitoringNodeLoader,
	deterLoader *DeteriorationNodeLoader,
	deterEngine DeteriorationEvaluator,
	log *zap.Logger,
) *SignalCascade {
	sc := &SignalCascade{
		pmToMD:      make(map[string][]string),
		mdToMD06:    make(map[string]bool),
		deterEngine: deterEngine,
		log:         log.With(zap.String("component", "signal-cascade")),
	}

	// Build pmToMD from all monitoring nodes' cascade_to fields.
	for id, node := range monLoader.All() {
		if len(node.CascadeTo) > 0 {
			sc.pmToMD[id] = append(sc.pmToMD[id], node.CascadeTo...)
		}
	}

	// Build mdToMD06 from MD-06's contributing_signals.
	md06 := deterLoader.Get("MD-06")
	if md06 != nil {
		for _, sig := range md06.ContributingSignals {
			sc.mdToMD06[sig] = true
		}
	}

	return sc
}

// nodeIDToFieldKey converts a node ID (e.g. "PM-04", "MD-01") to the underscore
// key expected by the expression evaluator (e.g. "pm_04", "md_01").
// The expression evaluator treats '-' as subtraction, so hyphens must not appear
// in field map keys.
func nodeIDToFieldKey(nodeID string) string {
	return strings.ToLower(strings.ReplaceAll(nodeID, "-", "_"))
}

// severityScore maps a severity string to a numeric score for cascade context.
//
//	NONE=0, MILD=1, MODERATE=2, CRITICAL=3. Unknown values map to 0.
func severityScore(severity string) float64 {
	switch severity {
	case "MILD":
		return 1.0
	case "MODERATE":
		return 2.0
	case "CRITICAL":
		return 3.0
	default: // NONE or unknown
		return 0.0
	}
}

// extractSeverity returns the numeric severity score from a ClinicalSignalEvent.
// For PM events (Classification != nil) the Category field is not used directly;
// instead the Severity field on the matched ClassificationDef is stored in the
// event. However, ClinicalSignalEvent.Classification only carries Category and
// Value (not Severity). We fall back to DeteriorationSignal.Severity for MD
// events, and use Classification.Category as proxy for PM events by matching
// the Category name to severity levels.
//
// In practice the MonitoringNodeEngine does not store severity on the event's
// Classification directly (it stores Category = e.g. "CRITICAL", "MODERATE"...).
// So for PM events we check Classification.Category for severity keywords, and
// for MD events we use DeteriorationResult.Severity.
func extractSeverityFromEvent(event *models.ClinicalSignalEvent) float64 {
	if event == nil {
		return 0.0
	}
	// MD node: use DeteriorationResult.Severity
	if event.DeteriorationSignal != nil {
		return severityScore(event.DeteriorationSignal.Severity)
	}
	// PM node: Classification.Category often holds the severity string.
	if event.Classification != nil {
		return severityScore(event.Classification.Category)
	}
	return 0.0
}

// Trigger executes the two-pass PM→MD→MD-06 cascade starting from sourceNodeID.
//
//   - classificationSeverity is the numeric severity (0-3) of the PM event that
//     triggered the cascade (passed in CascadeContext.PMSignals for MD evaluation).
//
// Returns all emitted ClinicalSignalEvents (Pass 1 + Pass 2). The slice is empty
// (not nil) when no MD nodes fire.
func (sc *SignalCascade) Trigger(
	ctx context.Context,
	sourceNodeID, patientID, stratumLabel string,
	classificationSeverity float64,
) []*models.ClinicalSignalEvent {
	mdNodeIDs, ok := sc.pmToMD[sourceNodeID]
	if !ok || len(mdNodeIDs) == 0 {
		sc.log.Debug("no cascade targets for PM node",
			zap.String("source_node_id", sourceNodeID),
		)
		return []*models.ClinicalSignalEvent{}
	}

	var results []*models.ClinicalSignalEvent

	// Build the base CascadeContext carrying the PM signal.
	pmKey := nodeIDToFieldKey(sourceNodeID)
	baseCtx := &CascadeContext{
		PMSignals: map[string]float64{
			pmKey: classificationSeverity,
		},
		MDSignals: make(map[string]float64),
	}

	// Pass 1: Evaluate each MD node in cascade_to.
	// Track which MD nodes fired (for Pass 2 trigger decision).
	firedMDNodeIDs := make(map[string]bool)

	for _, mdNodeID := range mdNodeIDs {
		sc.log.Info("cascade pass-1: evaluating MD node",
			zap.String("source_pm", sourceNodeID),
			zap.String("md_node", mdNodeID),
			zap.String("patient_id", patientID),
		)

		event, err := sc.deterEngine.Evaluate(ctx, mdNodeID, patientID, stratumLabel, baseCtx)
		if err != nil {
			sc.log.Warn("cascade pass-1: MD evaluation failed (non-fatal)",
				zap.String("md_node", mdNodeID),
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
			continue
		}
		if event == nil {
			// Node returned no event (e.g. SKIP policy with insufficient data).
			continue
		}

		results = append(results, event)

		// Determine if this event "fired" (has a meaningful signal).
		// We consider an event fired when it has a DeteriorationSignal or Classification.
		eventFired := event.DeteriorationSignal != nil || event.Classification != nil
		if eventFired {
			firedMDNodeIDs[mdNodeID] = true
			// Accumulate MD severity score into baseCtx.MDSignals for Pass 2.
			mdKey := nodeIDToFieldKey(mdNodeID)
			score := extractSeverityFromEvent(event)
			baseCtx.MDSignals[mdKey] = score
		}
	}

	// Pass 2: Evaluate MD-06 if any Pass-1 fired MD node is in MD-06's contributing_signals.
	shouldEvaluateMD06 := false
	for mdNodeID := range firedMDNodeIDs {
		if sc.mdToMD06[mdNodeID] {
			shouldEvaluateMD06 = true
			break
		}
	}

	if shouldEvaluateMD06 {
		sc.log.Info("cascade pass-2: evaluating MD-06",
			zap.String("source_pm", sourceNodeID),
			zap.String("patient_id", patientID),
		)

		md06Event, err := sc.deterEngine.Evaluate(ctx, "MD-06", patientID, stratumLabel, baseCtx)
		if err != nil {
			sc.log.Warn("cascade pass-2: MD-06 evaluation failed (non-fatal)",
				zap.String("patient_id", patientID),
				zap.Error(err),
			)
		} else if md06Event != nil {
			results = append(results, md06Event)
		}
	}

	return results
}
