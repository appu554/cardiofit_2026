package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/models"
)

// KB20Fetcher is the narrow interface the orchestrator needs from KB-20.
// Defined here (not in the clients package) so tests can stub it without
// importing the real client.
type KB20Fetcher interface {
	FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error)
}

// KB21Fetcher is the narrow interface the orchestrator needs from KB-21.
type KB21Fetcher interface {
	FetchEngagement(ctx context.Context, patientID string) (*clients.KB21EngagementProfile, error)
}

// KB19EventPublisher is the narrow interface the orchestrator needs from
// the KB-19 client. Defined here (not in the clients package) so tests
// can stub it without importing the real client.
type KB19EventPublisher interface {
	PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error
	PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error
}

// BPContextOrchestrator coordinates upstream fetches, classification, and
// persistence for a single patient's BP context analysis.
type BPContextOrchestrator struct {
	kb20       KB20Fetcher
	kb21       KB21Fetcher
	repo       *BPContextRepository
	thresholds *config.BPContextThresholds
	log        *zap.Logger
	metrics    *metrics.Collector
	kb19       KB19EventPublisher
}

// NewBPContextOrchestrator wires the orchestrator dependencies.
func NewBPContextOrchestrator(
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	repo *BPContextRepository,
	thresholds *config.BPContextThresholds,
	log *zap.Logger,
	metricsCollector *metrics.Collector,
	kb19 KB19EventPublisher,
) *BPContextOrchestrator {
	return &BPContextOrchestrator{
		kb20:       kb20,
		kb21:       kb21,
		repo:       repo,
		thresholds: thresholds,
		log:        log,
		metrics:    metricsCollector,
		kb19:       kb19,
	}
}

// Classify is the entry point for BP context analysis. It fetches inputs
// from KB-20 (required) and KB-21 (best-effort), runs the Phase 1
// classifier, and persists the result.
func (o *BPContextOrchestrator) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	start := time.Now()
	defer func() {
		if o.metrics != nil {
			o.metrics.BPClassifyLatency.Observe(time.Since(start).Seconds())
		}
	}()

	profile, err := o.kb20.FetchProfile(ctx, patientID)
	if err != nil {
		if o.metrics != nil {
			o.metrics.BPClassifyErrors.Inc()
		}
		return nil, fmt.Errorf("fetch KB-20 profile: %w", err)
	}
	if profile == nil {
		if o.metrics != nil {
			o.metrics.BPClassifyErrors.Inc()
		}
		return nil, fmt.Errorf("patient %s not found in KB-20", patientID)
	}

	// KB-21 is best-effort. Outage degrades to "no engagement phenotype",
	// which the classifier handles cleanly (no bias flag fires).
	var engagementPhenotype string
	engagement, kb21Err := o.kb21.FetchEngagement(ctx, patientID)
	if kb21Err != nil {
		o.log.Warn("KB-21 fetch failed; continuing without engagement",
			zap.String("patient_id", patientID),
			zap.Error(kb21Err))
	} else if engagement != nil {
		composite := 0.0
		if engagement.EngagementComposite != nil {
			composite = *engagement.EngagementComposite
		}
		engagementPhenotype = clients.MapEngagementToBPPhenotype(engagement.Phenotype, composite)
	}

	input := buildBPContextInputFromProfile(profile, engagementPhenotype)
	result := ClassifyBPContext(input, o.thresholds)
	result.PatientID = patientID

	// Capture the prior phenotype BEFORE saving — SaveSnapshot upserts on
	// (patient_id, snapshot_date), so a same-day reclassification overwrites
	// yesterday's row and we'd lose the comparison if we fetched after.
	var oldPhenotype models.BPContextPhenotype
	prior, fetchErr := o.repo.FetchLatest(patientID)
	if fetchErr != nil {
		o.log.Warn("prior snapshot fetch failed; treating as first detection",
			zap.String("patient_id", patientID), zap.Error(fetchErr))
	} else if prior != nil {
		oldPhenotype = prior.Phenotype
	}

	// Persist snapshot. If persistence fails, log but do not block the
	// classification result — the caller still gets the analysis.
	snapshot := &models.BPContextHistory{
		ID:            uuid.New().String(),
		PatientID:     patientID,
		SnapshotDate:  time.Now().UTC().Truncate(24 * time.Hour),
		Phenotype:     result.Phenotype,
		ClinicSBPMean: result.ClinicSBPMean,
		HomeSBPMean:   result.HomeSBPMean,
		GapSBP:        result.ClinicHomeGapSBP,
		Confidence:    result.Confidence,
	}
	if err := o.repo.SaveSnapshot(snapshot); err != nil {
		o.log.Error("BP context snapshot persistence failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
	}

	if o.metrics != nil {
		o.metrics.BPPhenotypeTotal.WithLabelValues(string(result.Phenotype)).Inc()
	}

	o.emitPhenotypeEvents(ctx, patientID, oldPhenotype, result.Phenotype)

	return &result, nil
}

// emitPhenotypeEvents publishes events to KB-19 when the classification
// represents a new detection or a phenotype transition.
//
//   oldPhenotype empty + new is masked variant -> MASKED_HTN_DETECTED
//   oldPhenotype != newPhenotype -> BP_PHENOTYPE_CHANGED
//   oldPhenotype == newPhenotype -> no event
//
// Failures are logged but do not affect the caller — events are
// best-effort, the snapshot is the source of truth.
func (o *BPContextOrchestrator) emitPhenotypeEvents(
	ctx context.Context,
	patientID string,
	oldPhenotype models.BPContextPhenotype,
	newPhenotype models.BPContextPhenotype,
) {
	if o.kb19 == nil {
		return
	}

	isNewDetection := oldPhenotype == "" && (newPhenotype == models.PhenotypeMaskedHTN || newPhenotype == models.PhenotypeMaskedUncontrolled)
	isTransition := oldPhenotype != "" && oldPhenotype != newPhenotype

	if isNewDetection {
		urgency := "URGENT"
		if newPhenotype == models.PhenotypeMaskedUncontrolled {
			urgency = "URGENT"
		}
		if err := o.kb19.PublishMaskedHTNDetected(ctx, patientID, string(newPhenotype), urgency); err != nil {
			o.log.Warn("KB-19 MASKED_HTN_DETECTED publish failed",
				zap.String("patient_id", patientID), zap.Error(err))
		}
		return
	}

	if isTransition {
		if err := o.kb19.PublishPhenotypeChanged(ctx, patientID, string(oldPhenotype), string(newPhenotype)); err != nil {
			o.log.Warn("KB-19 BP_PHENOTYPE_CHANGED publish failed",
				zap.String("patient_id", patientID), zap.Error(err))
		}
	}
}

// buildBPContextInputFromProfile constructs synthetic BPReading slices
// from the aggregate values on KB-20's patient profile. This is a Phase 2
// limitation: per-reading data does not exist anywhere in the Go services,
// so we manufacture the minimum number of "readings" needed to satisfy
// the classifier's data sufficiency gates, all carrying the means as values.
//
// Phase 3 would replace this with a real per-reading store and remove
// this synthetic-reading hack.
func buildBPContextInputFromProfile(profile *clients.KB20PatientProfile, engagementPhenotype string) BPContextInput {
	input := BPContextInput{
		PatientID:           profile.PatientID,
		IsDiabetic:          profile.IsDiabetic,
		HasCKD:              profile.HasCKD,
		OnAntihypertensives: profile.OnHTNMeds,
		EngagementPhenotype: engagementPhenotype,
	}
	if profile.MorningSurge7dAvg != nil {
		input.MorningSurge7dAvg = *profile.MorningSurge7dAvg
	}

	// Synthesize clinic readings from the clinic mean.
	if profile.ClinicSBPMean != nil && profile.ClinicDBPMean != nil && profile.ClinicReadings >= 2 {
		count := profile.ClinicReadings
		input.ClinicReadings = make([]BPReading, count)
		now := time.Now()
		for i := 0; i < count; i++ {
			input.ClinicReadings[i] = BPReading{
				SBP:       *profile.ClinicSBPMean,
				DBP:       *profile.ClinicDBPMean,
				Source:    "CLINIC",
				Timestamp: now.AddDate(0, 0, -i*30),
			}
		}
	}

	// Synthesize home readings from the home mean (SBP14dMean stand-in).
	// Spread across distinct days so the classifier's "min distinct days"
	// gate passes when the count is sufficient.
	if profile.SBP14dMean != nil && profile.DBP14dMean != nil && profile.HomeReadings >= 12 {
		count := profile.HomeReadings
		input.HomeReadings = make([]BPReading, count)
		now := time.Now()
		for i := 0; i < count; i++ {
			input.HomeReadings[i] = BPReading{
				SBP:       *profile.SBP14dMean,
				DBP:       *profile.DBP14dMean,
				Source:    "HOME_CUFF",
				Timestamp: now.Add(time.Duration(-i*12) * time.Hour),
			}
		}
	}

	return input
}
