package services

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// RenalContextFetcher is the narrow dependency RenalAnticipatoryOrchestrator
// needs to retrieve per-patient renal state. Defined as an interface so
// tests can inject a stub fetcher without an HTTP server. Production wires
// this to KB20Client.FetchRenalStatus.
type RenalContextFetcher interface {
	FetchRenalStatus(ctx context.Context, patientID string) (*KB20RenalStatus, error)
}

// RenalAnticipatoryResult is the per-patient output of the orchestrator.
// Carries both the proactive threshold alerts and the staleness verdict
// so the batch consumer can decide which cards to persist.
type RenalAnticipatoryResult struct {
	PatientID          string
	ApproachingAlerts  []AnticipatoryAlert
	StaleEGFR          StaleEGFRResult
	StaleEGFRTriggered bool
	EGFR               float64
	CKDStage           string
}

// RenalAnticipatoryOrchestrator evaluates one patient per call: fetches
// their renal context from KB-20, runs FindApproachingThresholds against
// the renal formulary, and runs DetectStaleEGFR against the staleness
// config. Pure wiring — the clinical logic lives in the existing
// FindApproachingThresholds + DetectStaleEGFR functions. Phase 7 P7-C.
type RenalAnticipatoryOrchestrator struct {
	fetcher   RenalContextFetcher
	formulary *RenalFormulary
	log       *zap.Logger
}

// NewRenalAnticipatoryOrchestrator wires the dependencies. formulary is
// required — without it the projection math has nothing to compare
// against. fetcher may be nil in tests that want to exercise only the
// pure-logic path via EvaluateWithContext.
func NewRenalAnticipatoryOrchestrator(fetcher RenalContextFetcher, formulary *RenalFormulary, log *zap.Logger) *RenalAnticipatoryOrchestrator {
	if log == nil {
		log = zap.NewNop()
	}
	return &RenalAnticipatoryOrchestrator{
		fetcher:   fetcher,
		formulary: formulary,
		log:       log,
	}
}

// EvaluatePatient fetches the patient's renal context from KB-20 and
// delegates to EvaluateWithContext. Returns an error only if the fetch
// itself fails; a successful fetch producing no alerts is the normal
// steady-state and returns a zero-value result with nil error.
func (o *RenalAnticipatoryOrchestrator) EvaluatePatient(ctx context.Context, patientID string) (*RenalAnticipatoryResult, error) {
	if o.fetcher == nil {
		return nil, fmt.Errorf("renal anticipatory orchestrator: fetcher not wired")
	}
	renalStatus, err := o.fetcher.FetchRenalStatus(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("fetch renal status for %s: %w", patientID, err)
	}
	if renalStatus == nil {
		return &RenalAnticipatoryResult{PatientID: patientID}, nil
	}
	return o.EvaluateWithContext(patientID, renalStatus), nil
}

// EvaluateWithContext runs the pure-logic portion of the orchestrator
// against an already-fetched KB20RenalStatus. Exported so tests can
// construct synthetic patient contexts without needing a stub HTTP
// server. Returns a non-nil result even when no alerts are found — the
// batch consumer checks ApproachingAlerts and StaleEGFRTriggered to
// decide which (if any) cards to persist.
func (o *RenalAnticipatoryOrchestrator) EvaluateWithContext(patientID string, renalStatus *KB20RenalStatus) *RenalAnticipatoryResult {
	result := &RenalAnticipatoryResult{
		PatientID: patientID,
		EGFR:      renalStatus.EGFR,
		CKDStage:  renalStatus.CKDStage,
	}
	if o.formulary == nil {
		o.log.Warn("renal anticipatory orchestrator: formulary nil, skipping",
			zap.String("patient_id", patientID))
		return result
	}

	// Build the ActiveMedication slice from the KB20 response. Only
	// active meds with a non-empty drug class are retained — any entries
	// with empty DrugClass would be silently skipped by
	// FindApproachingThresholds anyway (the formulary lookup returns nil),
	// but filtering here keeps the on-renal-sensitive-med flag honest.
	meds := make([]ActiveMedication, 0, len(renalStatus.ActiveMedications))
	onRenalSensitive := false
	for _, m := range renalStatus.ActiveMedications {
		if !m.IsActive || m.DrugClass == "" {
			continue
		}
		meds = append(meds, ActiveMedication{DrugClass: m.DrugClass})
		if o.formulary.GetRule(m.DrugClass) != nil {
			onRenalSensitive = true
		}
	}

	// 1. Projection-based approaching-threshold alerts.
	result.ApproachingAlerts = FindApproachingThresholds(
		o.formulary,
		renalStatus.EGFR,
		renalStatus.EGFRSlope,
		meds,
	)

	// 2. Staleness check. Only fires for patients on a renal-sensitive
	// medication — an unmedicated patient without a recent eGFR isn't
	// actionable here and is better left to a separate lab-planning
	// flow in a future phase.
	staleRenal := models.RenalStatus{
		EGFR:           renalStatus.EGFR,
		EGFRMeasuredAt: renalStatus.EGFRMeasuredAt,
	}
	staleResult := DetectStaleEGFR(staleRenal, o.formulary.StaleEGFR, onRenalSensitive)
	result.StaleEGFR = staleResult
	result.StaleEGFRTriggered = staleResult.IsStale && onRenalSensitive

	o.log.Debug("renal anticipatory orchestrator evaluated patient",
		zap.String("patient_id", patientID),
		zap.Float64("egfr", renalStatus.EGFR),
		zap.Float64("slope", renalStatus.EGFRSlope),
		zap.Int("approaching_alerts", len(result.ApproachingAlerts)),
		zap.Bool("stale_triggered", result.StaleEGFRTriggered))

	return result
}
