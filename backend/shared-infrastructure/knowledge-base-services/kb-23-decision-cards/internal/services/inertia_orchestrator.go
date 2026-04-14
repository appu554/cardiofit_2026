package services

import (
	"context"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// InertiaOrchestrator is the thin coordination layer that wraps DetectInertia
// so a batch job or an HTTP handler can evaluate a patient's inertia without
// re-deriving the call sequence. Phase 6 P6-1.
//
// The orchestrator is intentionally stateless — it doesn't fetch data from
// KB-20 or KB-26; callers assemble the InertiaDetectorInput and pass it in.
// This keeps the orchestrator testable with synthetic inputs and lets
// different callers assemble the input their own way (batch iterating over
// a patient list, HTTP handler for single-patient eval, test harness, etc.).
//
// Scope note — production data assembly (fetching glycaemic/hemodynamic/renal
// target status from KB-26 and intervention timeline from KB-20) is a Phase 6
// follow-up. The orchestrator + batch structure is the abstraction proof;
// when the upstream data sources are wired, a new InertiaInputAssembler is
// added that calls this orchestrator per patient in the batch Run method.
type InertiaOrchestrator struct {
	log *zap.Logger
}

// NewInertiaOrchestrator constructs a stateless orchestrator.
func NewInertiaOrchestrator(log *zap.Logger) *InertiaOrchestrator {
	if log == nil {
		log = zap.NewNop()
	}
	return &InertiaOrchestrator{log: log}
}

// Evaluate runs DetectInertia on the given input and logs the verdict. Phase 6
// follow-up will extend this to also persist verdict history and publish
// KB-19 events; for now, the orchestrator is a pure pass-through with
// structured logging so downstream observability has visibility into
// per-patient evaluation outcomes.
func (o *InertiaOrchestrator) Evaluate(ctx context.Context, input InertiaDetectorInput) models.PatientInertiaReport {
	report := DetectInertia(input)

	if len(report.Verdicts) == 0 {
		o.log.Debug("inertia: no verdicts",
			zap.String("patient_id", input.PatientID))
		return report
	}

	domains := make([]string, 0, len(report.Verdicts))
	for _, v := range report.Verdicts {
		if v.Detected {
			domains = append(domains, string(v.Domain))
		}
	}

	o.log.Info("inertia: verdicts detected",
		zap.String("patient_id", input.PatientID),
		zap.Int("verdict_count", len(report.Verdicts)),
		zap.Strings("domains", domains),
		zap.String("note", "event publication + verdict history pending Phase 6 follow-up"),
	)
	return report
}
