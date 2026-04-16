package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// monitoringLapsedTemplateID matches the YAML template file's
// template_id. Phase 9 P9-B.
const monitoringLapsedTemplateID = "dc-monitoring-lapsed-v1"

// MonitoringLapsedPatient mirrors the KB-20 endpoint response entry.
type MonitoringLapsedPatient struct {
	PatientID             string    `json:"patient_id"`
	LastHomeBPReadingAt   time.Time `json:"last_home_bp_reading_at"`
	DaysSinceLastReading  int       `json:"days_since_last_reading"`
	ReadingsInPrior28Days int       `json:"readings_in_prior_28_days"`
}

// DetectMonitoringLapse is the pure function that evaluates whether a
// patient has lapsed from active monitoring. Exported for unit testing
// without HTTP dependencies. Phase 9 P9-B.
//
// A patient has lapsed when:
//   - They had >= minPriorReadings readings in the 28 days before
//     the gap window (proves active monitoring pattern)
//   - Their last reading is older than gapDays days (proves the
//     pattern broke)
//
// Returns true when both conditions are met. False means either
// the patient was never actively monitoring (no pattern to lapse
// from) or they're still monitoring (latest reading is fresh).
func DetectMonitoringLapse(
	readingsInPrior28Days int,
	daysSinceLastReading int,
	minPriorReadings int,
	gapDays int,
) bool {
	return readingsInPrior28Days >= minPriorReadings && daysSinceLastReading >= gapDays
}

// MonitoringEngagementBatch is a BatchJob that runs weekly (Wednesday
// 04:00 UTC — offset from the Sunday 03:00 inertia batch to spread
// load) and checks for patients who've stopped home BP monitoring.
// Phase 9 P9-B.
type MonitoringEngagementBatch struct {
	cfg            *config.Config
	templateLoader *TemplateLoader
	db             *database.Database
	gateCache      *MCUGateCache
	kb19           *KB19Publisher
	metrics        *metrics.Collector
	log            *zap.Logger
}

// NewMonitoringEngagementBatch wires the dependencies. All optional
// for degraded / test modes (matching the convention from
// RenalAnticipatoryBatch and InertiaWeeklyBatch).
func NewMonitoringEngagementBatch(
	cfg *config.Config,
	templateLoader *TemplateLoader,
	db *database.Database,
	gateCache *MCUGateCache,
	kb19 *KB19Publisher,
	m *metrics.Collector,
	log *zap.Logger,
) *MonitoringEngagementBatch {
	if log == nil {
		log = zap.NewNop()
	}
	return &MonitoringEngagementBatch{
		cfg:            cfg,
		templateLoader: templateLoader,
		db:             db,
		gateCache:      gateCache,
		kb19:           kb19,
		metrics:        m,
		log:            log,
	}
}

// Name implements BatchJob.
func (j *MonitoringEngagementBatch) Name() string { return "monitoring_engagement_weekly" }

// ShouldRun implements BatchJob — fires on Wednesdays at 04:00 UTC.
// Offset from the Sunday 03:00 inertia batch to spread Kafka +
// HTTP load across the week. Multiple ticks within the same hour
// on a Wednesday would all fire, but the cards are patient-keyed
// and the template is idempotent.
func (j *MonitoringEngagementBatch) ShouldRun(ctx context.Context, now time.Time) bool {
	return now.Weekday() == time.Wednesday && now.Hour() == 4
}

// Run fetches the monitoring-lapsed patient list from KB-20 and
// generates one MONITORING_LAPSED card per patient. Phase 9 P9-B.
func (j *MonitoringEngagementBatch) Run(ctx context.Context) error {
	start := time.Now()
	defer func() {
		if j.metrics != nil {
			j.metrics.MonitoringLapsedBatchDuration.Observe(time.Since(start).Seconds())
		}
	}()

	if j.cfg == nil {
		j.log.Warn("monitoring engagement batch: config nil, skipping")
		return nil
	}

	patients, err := j.fetchLapsedPatients(ctx)
	if err != nil {
		return fmt.Errorf("fetch monitoring-lapsed patients: %w", err)
	}

	if len(patients) == 0 {
		j.log.Info("monitoring engagement batch: no lapsed patients found")
		return nil
	}

	cardCount := 0
	for _, p := range patients {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := j.persistMonitoringLapsedCard(p); err != nil {
			j.log.Warn("monitoring engagement batch: card persistence failed",
				zap.String("patient_id", p.PatientID),
				zap.Error(err))
			continue
		}
		cardCount++
		if j.metrics != nil {
			j.metrics.MonitoringLapsedDetected.WithLabelValues("HOME_BP").Inc()
		}
	}

	j.log.Info("monitoring engagement batch completed",
		zap.Int("lapsed_patients", len(patients)),
		zap.Int("cards_persisted", cardCount),
		zap.Duration("duration", time.Since(start)))
	return nil
}

// fetchLapsedPatients calls KB-20's GET /api/v1/patients/monitoring-lapsed
// endpoint. Returns the pre-computed list of patients who stopped
// home BP monitoring.
func (j *MonitoringEngagementBatch) fetchLapsedPatients(ctx context.Context) ([]MonitoringLapsedPatient, error) {
	url := fmt.Sprintf("%s/api/v1/patients/monitoring-lapsed", j.cfg.KB20URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: j.cfg.KB20Timeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-20 monitoring-lapsed fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-20 monitoring-lapsed returned status %d: %s", resp.StatusCode, string(body))
	}

	var env struct {
		Success bool                      `json:"success"`
		Data    []MonitoringLapsedPatient `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode monitoring-lapsed response: %w", err)
	}
	return env.Data, nil
}

// persistMonitoringLapsedCard builds + persists a MONITORING_LAPSED
// card from the template. Uses the same persistence pattern as the
// P7-A renal and P7-D inertia card paths.
func (j *MonitoringEngagementBatch) persistMonitoringLapsedCard(patient MonitoringLapsedPatient) error {
	if j.templateLoader == nil || j.db == nil {
		return nil
	}
	tmpl, ok := j.templateLoader.Get(monitoringLapsedTemplateID)
	if !ok {
		return fmt.Errorf("template %s not loaded", monitoringLapsedTemplateID)
	}

	pid, err := uuid.Parse(patient.PatientID)
	if err != nil {
		return fmt.Errorf("invalid patient_id: %w", err)
	}

	data := struct {
		DaysSince    string
		PreviousRate string
	}{
		DaysSince:    fmt.Sprintf("%d", patient.DaysSinceLastReading),
		PreviousRate: fmt.Sprintf("%d", patient.ReadingsInPrior28Days),
	}

	var clinician, patientEn, patientHi string
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
		clinician = fmt.Sprintf("Patient stopped home BP monitoring — last reading %d days ago (was %d readings/28d)",
			patient.DaysSinceLastReading, patient.ReadingsInPrior28Days)
	}

	notes := clinician
	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                pid,
		TemplateID:               tmpl.TemplateID,
		NodeID:                   tmpl.NodeID,
		PrimaryDifferentialID:    "MONITORING_LAPSED",
		DiagnosticConfidenceTier: models.TierProbable,
		MCUGate:                  models.GateSafe,
		MCUGateRationale:         clinician,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityModerate,
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
		return fmt.Errorf("save monitoring lapsed card: %w", err)
	}
	if j.gateCache != nil {
		_ = j.gateCache.WriteGate(card)
	}
	if j.kb19 != nil {
		go j.kb19.PublishGateChanged(card)
	}
	return nil
}
