package services

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// InertiaActivePatientLister is the narrow repo dependency the weekly
// inertia batch needs — defined here rather than referenced via the
// concrete BPContextRepository so the job can be unit-tested with a stub.
type InertiaActivePatientLister interface {
	ListActivePatientIDs(window time.Duration) ([]string, error)
}

// InertiaWeeklyBatch is a BatchJob that runs once per week (Mondays) and
// logs how many active patients would be scanned for therapeutic inertia.
// Phase 5 P5-3: second consumer of BatchScheduler, used to prove the
// abstraction hosts jobs with cadences other than the daily BP context.
//
// Scope note — this job is intentionally a heartbeat, not a full inertia
// scan. Real inertia detection needs domain-specific input assembly
// (glycaemic + hemodynamic + renal data pulls from KB-20) that lives
// outside the P5 scope. When that wiring ships in Phase 6, the Run
// method is extended to call kb23Client.TriggerInertiaScan(patientID)
// per patient; the scheduler/ShouldRun contract proven here stays the
// same. See docs/superpowers/plans/2026-04-14-masked-htn-phase-5-*.md
// for the deferred scope.
type InertiaWeeklyBatch struct {
	repo InertiaActivePatientLister
	log  *zap.Logger
}

// NewInertiaWeeklyBatch wires the dependencies.
func NewInertiaWeeklyBatch(repo InertiaActivePatientLister, log *zap.Logger) *InertiaWeeklyBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &InertiaWeeklyBatch{repo: repo, log: log}
}

// Name implements BatchJob.
func (j *InertiaWeeklyBatch) Name() string { return "inertia_weekly" }

// ShouldRun implements BatchJob — fires only on Mondays. The scheduler
// ticks hourly (Phase 5 P5-3); ShouldRun filters to one day per week.
// Note: will fire on every tick on Monday, not just once — tracking
// "did today's run already happen" is left to the job or the scheduler
// as a follow-up. For a weekly heartbeat job, 24 runs on a Monday is
// harmless because Run is cheap.
func (j *InertiaWeeklyBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Weekday() == time.Monday
}

// Run lists active patients and logs how many would be scanned for
// therapeutic inertia. Phase 6 will extend this to per-patient scan
// invocation via a KB-23 HTTP endpoint; the scheduler/ShouldRun
// contract proven here stays the same.
func (j *InertiaWeeklyBatch) Run(ctx context.Context) error {
	ids, err := j.repo.ListActivePatientIDs(60 * 24 * time.Hour)
	if err != nil {
		return err
	}
	j.log.Info("inertia weekly heartbeat",
		zap.Int("active_patient_count", len(ids)),
		zap.String("note", "real per-patient scan pending Phase 6 inertia-data-assembly"))
	return nil
}
