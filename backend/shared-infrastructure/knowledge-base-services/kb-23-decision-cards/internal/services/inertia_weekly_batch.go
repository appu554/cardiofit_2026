package services

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// InertiaActivePatientLister is the narrow dependency the inertia weekly
// batch needs to enumerate active patients for inertia evaluation.
// Production wiring is a Phase 6 follow-up that connects this to a KB-20
// query returning patients with uncontrolled clinical domains. Tests stub
// it directly.
type InertiaActivePatientLister interface {
	ListInertiaActivePatientIDs(ctx context.Context) ([]string, error)
}

// renalListerAsInertia adapts a RenalActivePatientLister (which exposes
// ListRenalActivePatientIDs) to the InertiaActivePatientLister interface
// (which expects ListInertiaActivePatientIDs). Phase 7 P7-D: the initial
// inertia population reuses the renal-active lister since patients on
// renal-sensitive medications are a clinically-meaningful superset of
// the patients where therapeutic inertia actually matters. A broader
// "all-active-CKM-patients" lister is a future refinement.
type renalListerAsInertia struct {
	inner RenalActivePatientLister
}

// ListInertiaActivePatientIDs delegates to the wrapped RenalActivePatientLister.
func (a renalListerAsInertia) ListInertiaActivePatientIDs(ctx context.Context) ([]string, error) {
	return a.inner.ListRenalActivePatientIDs(ctx)
}

// NewRenalListerAsInertiaLister wraps a RenalActivePatientLister so it
// satisfies the InertiaActivePatientLister interface. Used by main.go
// to feed the P7-D inertia batch the same patient population as the
// P7-C renal anticipatory batch.
func NewRenalListerAsInertiaLister(inner RenalActivePatientLister) InertiaActivePatientLister {
	return renalListerAsInertia{inner: inner}
}

// InertiaInputAssembler is the narrow dependency the batch needs to
// construct an InertiaDetectorInput for a given patient. Phase 6
// follow-up wires this to KB-20 (intervention timeline + active
// medications) and KB-26 (glycaemic/hemodynamic/renal target status).
// When nil, the batch runs in heartbeat mode and skips per-patient
// evaluation — the scheduler integration is the abstraction proof.
type InertiaInputAssembler interface {
	AssembleInertiaInput(ctx context.Context, patientID string) (InertiaDetectorInput, error)
}

// InertiaWeeklyBatch is a BatchJob that runs once per week (Sunday 03:00 UTC)
// and evaluates therapeutic inertia across all active patients. Phase 6
// P6-1: replaces the Phase 5 P5-3 KB-26 heartbeat with a KB-23-resident
// batch that lives alongside the inertia detector. The KB-23 BatchScheduler
// (built in Phase 6 P6-5) hosts this as its second consumer.
//
// Scope note — production assembler is deferred. When nil, the batch fetches
// the active patient list but does not per-patient evaluate; instead it
// logs a heartbeat with the count. This is consistent with the P6-5
// RenalAnticipatoryBatch pattern: the abstraction proof (scheduler hosts,
// ShouldRun gates on Sunday, Run iterates active patients) ships now; the
// per-patient orchestrator assembly that the detector needs is a Phase 6
// follow-up tied to KB-20's intervention timeline service and KB-26's
// target-status HTTP exposure.
type InertiaWeeklyBatch struct {
	repo         InertiaActivePatientLister
	assembler    InertiaInputAssembler
	orchestrator *InertiaOrchestrator
	log          *zap.Logger
}

// NewInertiaWeeklyBatch wires the dependencies. assembler and orchestrator
// are optional — when either is nil, the batch runs in heartbeat mode and
// logs the active patient count without per-patient evaluation.
func NewInertiaWeeklyBatch(
	repo InertiaActivePatientLister,
	assembler InertiaInputAssembler,
	orchestrator *InertiaOrchestrator,
	log *zap.Logger,
) *InertiaWeeklyBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &InertiaWeeklyBatch{
		repo:         repo,
		assembler:    assembler,
		orchestrator: orchestrator,
		log:          log,
	}
}

// Name implements BatchJob.
func (j *InertiaWeeklyBatch) Name() string { return "inertia_weekly" }

// ShouldRun implements BatchJob — fires only on Sundays at 03:00 UTC.
// The KB-23 BatchScheduler ticks hourly; ShouldRun filters to one fire
// per week. Multiple ticks within the same hour would all fire, but the
// orchestrator is idempotent per patient per week so repeats are safe.
func (j *InertiaWeeklyBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Weekday() == time.Sunday && now.Hour() == 3
}

// Run iterates active patients and, when the assembler + orchestrator are
// wired, evaluates inertia per patient. In heartbeat mode (assembler or
// orchestrator nil) it logs the patient count and returns without
// per-patient evaluation.
func (j *InertiaWeeklyBatch) Run(ctx context.Context) error {
	if j.repo == nil {
		j.log.Warn("inertia weekly batch: repo nil, skipping")
		return nil
	}
	ids, err := j.repo.ListInertiaActivePatientIDs(ctx)
	if err != nil {
		return err
	}

	if j.assembler == nil || j.orchestrator == nil {
		j.log.Info("inertia weekly heartbeat",
			zap.Int("active_patient_count", len(ids)),
			zap.String("note", "per-patient assembly pending Phase 6 follow-up — needs KB-20 intervention timeline + KB-26 target-status HTTP exposure"))
		return nil
	}

	// Full per-patient evaluation. Errors on individual patients are
	// logged but do not abort the batch — each patient is isolated.
	evaluated := 0
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		input, assembleErr := j.assembler.AssembleInertiaInput(ctx, id)
		if assembleErr != nil {
			j.log.Warn("inertia weekly batch: assembly failed",
				zap.String("patient_id", id),
				zap.Error(assembleErr))
			continue
		}
		j.orchestrator.Evaluate(ctx, input)
		evaluated++
	}
	j.log.Info("inertia weekly batch complete",
		zap.Int("active_patient_count", len(ids)),
		zap.Int("evaluated", evaluated))
	return nil
}
