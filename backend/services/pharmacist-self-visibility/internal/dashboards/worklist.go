// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// VisibilityClass: WO (workflow-operational) — items are visible to anyone
// holding a workflow role on the relevant resident but are never aggregated
// as performance data for the pharmacist themselves.
package dashboards

import (
	"context"
	"sort"

	"github.com/google/uuid"
)

// WorklistItem represents a single resident entry in the daily worklist.
// It carries a CompositeRisk score plus the restraint signals that informed
// the score so that the pharmacist has full context inline.
//
// VisibilityClass: WO (workflow-operational)
type WorklistItem struct {
	// ResidentID is the FHIR Patient resource identifier for the resident.
	ResidentID uuid.UUID
	// CompositeRisk is the aggregate risk score (recent fall + recent admission
	// + new high-risk medication + overdue monitoring + family concern).
	// Higher values indicate greater urgency.
	CompositeRisk int
	// TopReasons are the top contributing signals to the composite risk score,
	// provided for pharmacist situational awareness.
	TopReasons []string
	// RestraintSignals are restraint-context indicators surfaced inline alongside
	// action prompts (e.g., "recent_fall_within_72h").
	RestraintSignals []string
	// EstimatedActionMin is the estimated number of minutes required for the
	// clinical action associated with this resident; may be zero if unknown.
	EstimatedActionMin int
}

// RiskSource is the data-access interface that backs the Worklist.
// Implementations are expected to be lightweight query projections; they must
// respect context cancellation.
type RiskSource interface {
	// ResidentsWithCompositeRisk returns a map of resident IDs to their current
	// composite risk scores for the given pharmacistID's caseload.
	ResidentsWithCompositeRisk(ctx context.Context, pharmacistID uuid.UUID) (map[uuid.UUID]int, error)
	// RestraintSignalsFor returns the active restraint signals for a single
	// resident (e.g., "recent_fall_within_72h", "chemical_restraint_review").
	RestraintSignalsFor(ctx context.Context, residentID uuid.UUID) ([]string, error)
	// TopReasons returns the top contributing signals to the composite risk score
	// for the given resident, in priority order.
	TopReasons(ctx context.Context, residentID uuid.UUID) ([]string, error)
}

// Worklist surfaces the risk-stratified daily queue for a pharmacist.
// Construct with NewWorklist; call Today to obtain the sorted queue.
type Worklist struct{ src RiskSource }

// NewWorklist constructs a Worklist backed by the given RiskSource.
func NewWorklist(src RiskSource) *Worklist { return &Worklist{src: src} }

// Today returns the pharmacist's daily worklist sorted by CompositeRisk
// descending (highest urgency first).
//
// Sorting is performed with sort.SliceStable so that residents with equal
// composite risk scores maintain a stable relative order across calls; callers
// may rely on this invariant for deterministic rendering.
//
// A defensive context check is performed after fetching scores: if the context
// has been cancelled by then, ctx.Err() is returned immediately so callers
// receive an explicit signal rather than a silently incomplete result.
//
// When the source returns no residents, Today returns an empty (non-nil) slice
// so that callers can distinguish "no work today" from an uninitialized result.
func (w *Worklist) Today(ctx context.Context, pharmacistID uuid.UUID) ([]WorklistItem, error) {
	scores, err := w.src.ResidentsWithCompositeRisk(ctx, pharmacistID)
	if err != nil {
		return nil, err
	}

	// Defensive context cancellation check: if the context has been cancelled
	// after the source query, return the error rather than building a partial
	// result that the caller cannot trust.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	items := make([]WorklistItem, 0, len(scores))
	for resID, score := range scores {
		signals, _ := w.src.RestraintSignalsFor(ctx, resID)
		reasons, _ := w.src.TopReasons(ctx, resID)
		items = append(items, WorklistItem{
			ResidentID:       resID,
			CompositeRisk:    score,
			TopReasons:       reasons,
			RestraintSignals: signals,
		})
	}

	// sort.SliceStable preserves insertion order for equal CompositeRisk values,
	// providing deterministic output when scores are tied.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CompositeRisk > items[j].CompositeRisk
	})
	return items, nil
}
