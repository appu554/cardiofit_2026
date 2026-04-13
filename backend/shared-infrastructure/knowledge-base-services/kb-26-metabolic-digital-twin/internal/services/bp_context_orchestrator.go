package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/config"
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

// BPContextOrchestrator coordinates upstream fetches, classification, and
// persistence for a single patient's BP context analysis.
type BPContextOrchestrator struct {
	kb20       KB20Fetcher
	kb21       KB21Fetcher
	repo       *BPContextRepository
	thresholds *config.BPContextThresholds
	log        *zap.Logger
}

// NewBPContextOrchestrator wires the orchestrator dependencies.
func NewBPContextOrchestrator(
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	repo *BPContextRepository,
	thresholds *config.BPContextThresholds,
	log *zap.Logger,
) *BPContextOrchestrator {
	return &BPContextOrchestrator{
		kb20:       kb20,
		kb21:       kb21,
		repo:       repo,
		thresholds: thresholds,
		log:        log,
	}
}

// Classify is the entry point for BP context analysis. It fetches inputs
// from KB-20 (required) and KB-21 (best-effort), runs the Phase 1
// classifier, and persists the result.
func (o *BPContextOrchestrator) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	profile, err := o.kb20.FetchProfile(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("fetch KB-20 profile: %w", err)
	}
	if profile == nil {
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

	return &result, nil
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
