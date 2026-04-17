package services

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// RenalActivePatientLister is the narrow dependency the renal anticipatory
// batch needs to enumerate patients on renal-sensitive medications.
// Production wiring is KB20Client.FetchRenalActivePatientIDs; tests can
// inject a simple slice-returning stub.
type RenalActivePatientLister interface {
	ListRenalActivePatientIDs(ctx context.Context) ([]string, error)
}

// Phase 7 P7-C: template IDs for the renal anticipatory cards. These
// match the template_id keys in templates/renal/*.yaml so the
// TemplateLoader can resolve them at card-build time.
const (
	renalThresholdApproachingTemplateID = "dc-renal-threshold-approaching-v1"
	staleEGFRTemplateID                 = "dc-stale-egfr-v1"
)

// RenalAnticipatoryBatch is a BatchJob that runs once per month (1st of
// the month, 04:00 UTC) and finds patients whose projected eGFR will
// cross a clinically significant threshold within the next 6-12 months,
// or whose eGFR surveillance is overdue relative to their CKD stage +
// medication profile.
//
// Phase 6 P6-5 shipped this as a heartbeat that listed active renal
// patients and logged the count. Phase 7 P7-C now wires the real
// per-patient orchestrator, persists DecisionCards for every detected
// alert, and emits Prometheus counters by drug class, horizon, and
// CKD stage.
type RenalAnticipatoryBatch struct {
	repo           RenalActivePatientLister
	orchestrator   *RenalAnticipatoryOrchestrator
	templateLoader *TemplateLoader
	db             *database.Database
	gateCache      *MCUGateCache
	kb19           *KB19Publisher
	fhirNotifier   FHIRCardNotifier
	metrics        *metrics.Collector
	log            *zap.Logger
}

// SetFHIRNotifier injects the FHIR notification client. Phase 10 Gap 9.
func (j *RenalAnticipatoryBatch) SetFHIRNotifier(n FHIRCardNotifier) { j.fhirNotifier = n }

// NewRenalAnticipatoryBatch wires the dependencies.
//
// Phase 7 P7-C: the signature now takes the orchestrator + card-persistence
// plumbing (templateLoader, db, gateCache, kb19, metrics). All are
// optional — a nil templateLoader degrades the job to orchestrator-only
// (alerts computed, no cards persisted) which is useful for bootstrap
// dry-runs. A nil repo still degrades to a no-op heartbeat for tests
// that exercise only the scheduler's ShouldRun contract.
func NewRenalAnticipatoryBatch(
	repo RenalActivePatientLister,
	orchestrator *RenalAnticipatoryOrchestrator,
	templateLoader *TemplateLoader,
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	m *metrics.Collector,
	log *zap.Logger,
) *RenalAnticipatoryBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &RenalAnticipatoryBatch{
		repo:           repo,
		orchestrator:   orchestrator,
		templateLoader: templateLoader,
		db:             db,
		gateCache:      gateCache,
		kb19:           kb19,
		metrics:        m,
		log:            log,
	}
}

// Name implements BatchJob.
func (j *RenalAnticipatoryBatch) Name() string { return "renal_anticipatory_monthly" }

// ShouldRun implements BatchJob — fires only on the 1st of the month at
// 04:00 UTC. The KB-23 BatchScheduler ticks hourly; ShouldRun filters to
// one fire per month per ticker. Multiple ticks within the same hour on
// the 1st would all fire, but the cards are content-deterministic and
// the KB-19 publish path is idempotent so repeats are safe.
func (j *RenalAnticipatoryBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Day() == 1 && now.Hour() == 4
}

// Run enumerates renal-active patients, runs the orchestrator per
// patient, and persists a DecisionCard for every approaching-threshold
// alert or stale-eGFR detection. Errors from individual patients are
// logged + isolated so one bad patient can't stop the batch.
//
// The end-to-end batch duration is recorded via
// kb23_renal_anticipatory_batch_duration_seconds.
func (j *RenalAnticipatoryBatch) Run(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if j.metrics != nil {
			j.metrics.RenalAnticipatoryBatchDuration.Observe(time.Since(start).Seconds())
		}
	}()

	if j.repo == nil {
		j.log.Warn("renal anticipatory batch: repo nil, skipping")
		return nil
	}
	ids, err := j.repo.ListRenalActivePatientIDs(ctx)
	if err != nil {
		return err
	}

	if j.orchestrator == nil {
		// Heartbeat mode — scheduler ticks still observable even without
		// the real orchestrator wired. Preserves the P6-5 behaviour for
		// bootstrap paths that haven't fully wired the KB-23 stack.
		j.log.Info("renal anticipatory monthly heartbeat (orchestrator not wired)",
			zap.Int("renal_active_patient_count", len(ids)))
		return nil
	}

	// Phase 9 P9-E: bounded-concurrency fan-out. Same pattern as
	// InertiaWeeklyBatch.Run — channel semaphore limits goroutine
	// fan-out to DefaultBatchConcurrency (4). Per-patient errors
	// are isolated; a goroutine failure does not abort siblings.
	var (
		approachingCardCount int64
		staleCardCount       int64
		evalErrCount         int64
	)
	sem := make(chan struct{}, DefaultBatchConcurrency)
	var wg sync.WaitGroup

	for _, patientID := range ids {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{} // acquire semaphore slot
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			defer func() { <-sem }() // release

			result, err := j.orchestrator.EvaluatePatient(ctx, pid)
			if err != nil {
				atomic.AddInt64(&evalErrCount, 1)
				j.log.Warn("renal anticipatory orchestrator error for patient",
					zap.String("patient_id", pid),
					zap.Error(err))
				return
			}
			if result == nil {
				return
			}

			for _, alert := range result.ApproachingAlerts {
				if err := j.persistApproachingCard(pid, result, alert); err != nil {
					j.log.Error("failed to persist renal-approaching card",
						zap.String("patient_id", pid),
						zap.String("drug_class", alert.DrugClass),
						zap.Error(err))
					continue
				}
				atomic.AddInt64(&approachingCardCount, 1)
				if j.metrics != nil {
					j.metrics.RenalAnticipatoryAlerts.
						WithLabelValues(alert.DrugClass, alert.ThresholdType).Inc()
				}
			}

			if result.StaleEGFRTriggered {
				if err := j.persistStaleEGFRCard(pid, result); err != nil {
					j.log.Error("failed to persist stale-eGFR card",
						zap.String("patient_id", pid),
						zap.Error(err))
					return
				}
				atomic.AddInt64(&staleCardCount, 1)
				if j.metrics != nil {
					j.metrics.StaleEGFRDetected.WithLabelValues(result.CKDStage).Inc()
				}
			}
		}(patientID)
	}
	wg.Wait()

	j.log.Info("renal anticipatory monthly batch completed",
		zap.Int("patients_evaluated", len(ids)),
		zap.Int64("approaching_cards_persisted", approachingCardCount),
		zap.Int64("stale_cards_persisted", staleCardCount),
		zap.Int64("per_patient_errors", evalErrCount),
		zap.Int("concurrency", DefaultBatchConcurrency),
		zap.Duration("duration", time.Since(start)))
	return nil
}

// persistApproachingCard looks up the RENAL_THRESHOLD_APPROACHING template,
// renders fragments with the alert's drug-class + horizon + months,
// assembles a DecisionCard, persists it, and fires the gate-changed
// event. Defensive when templateLoader / db / gateCache / kb19 are nil.
func (j *RenalAnticipatoryBatch) persistApproachingCard(
	patientID string,
	result *RenalAnticipatoryResult,
	alert AnticipatoryAlert,
) error {
	if j.templateLoader == nil || j.db == nil {
		j.log.Debug("renal approaching card persistence skipped: templateLoader or db not wired",
			zap.String("patient_id", patientID))
		return nil
	}

	tmpl, ok := j.templateLoader.Get(renalThresholdApproachingTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", renalThresholdApproachingTemplateID)
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinician, patientEn, patientHi := renderApproachingSummaries(tmpl, result, alert)
	notes := clinician

	// Slope carried through for fragment substitution via runtime data,
	// but MCUGate/SafetyTier stay at the template defaults: MODIFY +
	// Routine — anticipatory, not acute.
	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "RENAL_THRESHOLD_APPROACHING",
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyRoutine,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := j.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save renal approaching card: %w", err)
	}
	if j.gateCache != nil {
		_ = j.gateCache.WriteGate(card)
	}
	if j.kb19 != nil {
		go j.kb19.PublishGateChanged(card)
	}
	notifyFHIR(j.fhirNotifier, card)
	return nil
}

// persistStaleEGFRCard looks up the STALE_EGFR template, renders the
// days-since / expected-max fragment variables, and persists a routine
// DecisionCard. Uses GateSafe — this is purely a surveillance gap, not
// a safety event.
func (j *RenalAnticipatoryBatch) persistStaleEGFRCard(
	patientID string,
	result *RenalAnticipatoryResult,
) error {
	if j.templateLoader == nil || j.db == nil {
		j.log.Debug("stale-eGFR card persistence skipped: templateLoader or db not wired",
			zap.String("patient_id", patientID))
		return nil
	}

	tmpl, ok := j.templateLoader.Get(staleEGFRTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", staleEGFRTemplateID)
	}

	pid, err := uuid.Parse(patientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	clinician, patientEn, patientHi := renderStaleEGFRSummaries(tmpl, result)
	notes := clinician

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "STALE_EGFR",
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateSafe,
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyRoutine,
		CardSource:               models.SourceClinicalSignal,
		Status:                   models.StatusActive,
		ClinicianSummary:         clinician,
		PatientSummaryEn:         patientEn,
		PatientSummaryHi:         patientHi,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := j.db.DB.Create(card).Error; err != nil {
		return fmt.Errorf("save stale-eGFR card: %w", err)
	}
	if j.gateCache != nil {
		_ = j.gateCache.WriteGate(card)
	}
	if j.kb19 != nil {
		go j.kb19.PublishGateChanged(card)
	}
	notifyFHIR(j.fhirNotifier, card)
	return nil
}

// renderApproachingSummaries substitutes {{.EGFR}} / {{.SlopePerYear}} /
// {{.DrugClass}} / {{.ThresholdType}} / {{.ThresholdValue}} /
// {{.MonthsToThreshold}} into the template fragments. Pure function —
// exported for unit testing without a database.
func renderApproachingSummaries(tmpl *models.CardTemplate, result *RenalAnticipatoryResult, alert AnticipatoryAlert) (clinician, patientEn, patientHi string) {
	data := struct {
		EGFR              string
		SlopePerYear      string
		DrugClass         string
		ThresholdType     string
		ThresholdValue    string
		MonthsToThreshold string
	}{
		EGFR:              fmt.Sprintf("%.1f", result.EGFR),
		SlopePerYear:      "",  // filled below
		DrugClass:         alert.DrugClass,
		ThresholdType:     alert.ThresholdType,
		ThresholdValue:    fmt.Sprintf("%.0f", alert.ThresholdValue),
		MonthsToThreshold: fmt.Sprintf("%.1f", alert.MonthsToThreshold),
	}

	for _, frag := range tmpl.Fragments {
		switch frag.FragmentType {
		case models.FragClinician:
			clinician = executeRenalTemplate(frag.TextEn, data)
		case models.FragPatient:
			patientEn = executeRenalTemplate(frag.TextEn, data)
			patientHi = executeRenalTemplate(frag.TextHi, data)
		}
	}
	if clinician == "" {
		clinician = fmt.Sprintf("Renal anticipatory: %s approaching %s at eGFR %.1f in ~%.1f months",
			alert.DrugClass, alert.ThresholdType, result.EGFR, alert.MonthsToThreshold)
	}
	return clinician, patientEn, patientHi
}

// renderStaleEGFRSummaries substitutes {{.DaysSince}} / {{.ExpectedMaxDays}} /
// {{.Severity}} / {{.EGFR}} / {{.CKDStage}} into the stale-eGFR template
// fragments. Pure function — exported for unit testing.
func renderStaleEGFRSummaries(tmpl *models.CardTemplate, result *RenalAnticipatoryResult) (clinician, patientEn, patientHi string) {
	data := struct {
		DaysSince       string
		ExpectedMaxDays string
		Severity        string
		EGFR            string
		CKDStage        string
	}{
		DaysSince:       fmt.Sprintf("%d", result.StaleEGFR.DaysSince),
		ExpectedMaxDays: fmt.Sprintf("%d", result.StaleEGFR.ExpectedMaxDays),
		Severity:        result.StaleEGFR.Severity,
		EGFR:            fmt.Sprintf("%.1f", result.EGFR),
		CKDStage:        result.CKDStage,
	}

	for _, frag := range tmpl.Fragments {
		switch frag.FragmentType {
		case models.FragClinician:
			clinician = executeRenalTemplate(frag.TextEn, data)
		case models.FragPatient:
			patientEn = executeRenalTemplate(frag.TextEn, data)
			patientHi = executeRenalTemplate(frag.TextHi, data)
		}
	}
	if clinician == "" {
		clinician = fmt.Sprintf("Stale eGFR: %d days old (expected within %d days); order renal panel",
			result.StaleEGFR.DaysSince, result.StaleEGFR.ExpectedMaxDays)
	}
	return clinician, patientEn, patientHi
}

// renalActivePatientListerFunc adapts a plain function to the
// RenalActivePatientLister interface. Lets main.go wrap
// KB20Client.FetchRenalActivePatientIDs without declaring a named
// struct just for the adapter.
type renalActivePatientListerFunc func(ctx context.Context) ([]string, error)

func (f renalActivePatientListerFunc) ListRenalActivePatientIDs(ctx context.Context) ([]string, error) {
	return f(ctx)
}

// NewKB20RenalActivePatientLister wraps a KB20Client into the
// RenalActivePatientLister interface expected by RenalAnticipatoryBatch.
// Phase 7 P7-C: production wiring path.
func NewKB20RenalActivePatientLister(client *KB20Client) RenalActivePatientLister {
	return renalActivePatientListerFunc(func(ctx context.Context) ([]string, error) {
		ids, err := client.FetchRenalActivePatientIDs(ctx)
		if err != nil {
			return nil, err
		}
		// Defensive: drop empty entries that might slip through from
		// a stale FHIR sync or an unusual test seed.
		cleaned := make([]string, 0, len(ids))
		for _, id := range ids {
			if strings.TrimSpace(id) != "" {
				cleaned = append(cleaned, id)
			}
		}
		return cleaned, nil
	})
}
