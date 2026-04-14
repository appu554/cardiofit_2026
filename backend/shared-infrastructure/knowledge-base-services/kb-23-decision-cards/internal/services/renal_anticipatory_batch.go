package services

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// RenalActivePatientLister is the narrow dependency the renal anticipatory
// batch needs to enumerate patients on renal-sensitive medications. Defined
// here rather than referenced via a concrete repository so the job can be
// unit-tested with a stub. Production wiring is a Phase 6 follow-up that
// connects this to KB-20's active-patient query (the lister currently
// returns an empty slice, making the job a no-op heartbeat).
type RenalActivePatientLister interface {
	ListRenalActivePatientIDs(ctx context.Context) ([]string, error)
}

// RenalAnticipatoryBatch is a BatchJob that runs once per month (1st of
// the month, 04:00 UTC) and finds patients whose projected eGFR will cross
// a clinically significant threshold within the next 6-12 months.
// Phase 6 P6-5: first KB-23 BatchJob consumer, proves the Phase 5 P5-3
// scheduler abstraction extracts cleanly to a second host service.
//
// Scope note — this job currently ships as a heartbeat that lists active
// renal patients and logs the count. The per-patient orchestration that
// would call FindApproachingThresholds + DetectStaleEGFR + publish events
// is a Phase 6 follow-up requiring (a) a KB-20 endpoint exposing active
// renal-sensitive patient IDs with their current eGFR + slope + active
// medications, and (b) a small RenalAnticipatoryOrchestrator analogous
// to the inertia orchestrator planned for P6-1. The abstraction proof
// P6-5 needs — that KB-23 hosts a BatchScheduler with its own consumer
// on a monthly cadence — does not depend on the orchestrator existing
// yet. This mirrors the P5-3 InertiaWeeklyBatch heartbeat pattern.
type RenalAnticipatoryBatch struct {
	repo RenalActivePatientLister
	log  *zap.Logger
}

// NewRenalAnticipatoryBatch wires the dependencies.
func NewRenalAnticipatoryBatch(repo RenalActivePatientLister, log *zap.Logger) *RenalAnticipatoryBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &RenalAnticipatoryBatch{repo: repo, log: log}
}

// Name implements BatchJob.
func (j *RenalAnticipatoryBatch) Name() string { return "renal_anticipatory_monthly" }

// ShouldRun implements BatchJob — fires only on the 1st of the month at
// 04:00 UTC. The KB-23 BatchScheduler ticks hourly; ShouldRun filters to
// one fire per month per ticker. Multiple ticks within the same hour on
// the 1st would all fire, but the job is idempotent (per-patient state
// is keyed on month) so repeats are safe.
func (j *RenalAnticipatoryBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Day() == 1 && now.Hour() == 4
}

// Run lists active renal-sensitive patients and logs the count. Phase 6
// follow-up will extend this to per-patient FindApproachingThresholds +
// DetectStaleEGFR invocation; the scheduler/ShouldRun contract proven
// here stays the same.
func (j *RenalAnticipatoryBatch) Run(ctx context.Context) error {
	if j.repo == nil {
		j.log.Warn("renal anticipatory batch: repo nil, skipping")
		return nil
	}
	ids, err := j.repo.ListRenalActivePatientIDs(ctx)
	if err != nil {
		return err
	}
	j.log.Info("renal anticipatory monthly heartbeat",
		zap.Int("renal_active_patient_count", len(ids)),
		zap.String("note", "real per-patient FindApproachingThresholds invocation pending Phase 6 follow-up — needs KB-20 active-renal-patient endpoint + orchestrator"))
	return nil
}
