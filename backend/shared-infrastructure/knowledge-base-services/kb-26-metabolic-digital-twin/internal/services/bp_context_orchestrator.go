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
	"kb-26-metabolic-digital-twin/pkg/stability"
)

// KB20Fetcher is the narrow interface the orchestrator needs from KB-20.
// Defined here (not in the clients package) so tests can stub it without
// importing the real client.
type KB20Fetcher interface {
	FetchProfile(ctx context.Context, patientID string) (*clients.KB20PatientProfile, error)
	FetchBPReadings(ctx context.Context, patientID string, since time.Time) ([]clients.KB20BPReading, error)
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

// KB23CompositeTrigger is the narrow interface the orchestrator needs from
// the KB-23 client for Phase 4 P9: trigger composite card synthesis after
// a successful classification so masked HTN + medication timing +
// selection bias cards fold into one CompositeCardSignal with the most
// restrictive MCU gate. Best-effort — failures are logged, not returned.
type KB23CompositeTrigger interface {
	TriggerCompositeSynthesize(ctx context.Context, patientID string) error
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
	kb23       KB23CompositeTrigger
	stability  *stability.Engine
}

// NewBPContextOrchestrator wires the orchestrator dependencies. kb23 is
// optional — pass nil to disable the Phase 4 P9 composite synthesis
// trigger (for tests or local dev without a KB-23 dependency).
func NewBPContextOrchestrator(
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	repo *BPContextRepository,
	thresholds *config.BPContextThresholds,
	log *zap.Logger,
	metricsCollector *metrics.Collector,
	kb19 KB19EventPublisher,
	stabilityEngine *stability.Engine,
	kb23 KB23CompositeTrigger,
) *BPContextOrchestrator {
	return &BPContextOrchestrator{
		kb20:       kb20,
		kb21:       kb21,
		repo:       repo,
		thresholds: thresholds,
		log:        log,
		metrics:    metricsCollector,
		kb19:       kb19,
		kb23:       kb23,
		stability:  stabilityEngine,
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

	// Phase 4 P3: Fetch real per-reading BP data from KB-20. If available,
	// this replaces the Phase 2 synthetic-readings hack and enables the
	// medication timing hypothesis to fire correctly. If the fetch fails
	// or returns empty, fall back to the synthetic path so the classifier
	// still produces a result — this keeps the feature working during
	// rollout when KB-20's /bp-readings endpoint may not yet be deployed.
	var realReadings []clients.KB20BPReading
	since := time.Now().UTC().AddDate(0, 0, -30)
	realReadings, fetchErr := o.kb20.FetchBPReadings(ctx, patientID, since)
	if fetchErr != nil {
		o.log.Warn("real BP reading fetch failed; falling back to synthetic",
			zap.String("patient_id", patientID), zap.Error(fetchErr))
		realReadings = nil
	}

	var input BPContextInput
	if len(realReadings) > 0 {
		input = buildBPContextInputFromReadings(profile, realReadings, engagementPhenotype)
	} else {
		input = buildBPContextInputFromProfile(profile, engagementPhenotype)
	}
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

	// Phase 4 P2: Stability check. Fetch recent history and evaluate
	// the proposed transition. If damped, keep the prior phenotype.
	// The stability engine is nil-safe — no engine means no dampening.
	if o.stability != nil && oldPhenotype != "" {
		historyRows, histErr := o.repo.FetchHistorySince(patientID, time.Now().UTC().AddDate(0, 0, -60))
		if histErr != nil {
			o.log.Warn("stability history fetch failed; skipping dampening",
				zap.String("patient_id", patientID), zap.Error(histErr))
		} else {
			history := buildStabilityHistory(historyRows)
			override := detectOverrideEvent(profile)
			decision := o.stability.Evaluate(
				history,
				string(result.Phenotype),
				time.Now().UTC(),
				override,
			)
			if decision.Decision == stability.DecisionDamp {
				o.log.Info("BP phenotype transition damped",
					zap.String("patient_id", patientID),
					zap.String("proposed", string(result.Phenotype)),
					zap.String("current", string(oldPhenotype)),
					zap.String("reason", decision.Reason))
				// Revert to the prior phenotype. The snapshot will be saved
				// with the old phenotype, preserving stability.
				result.Phenotype = oldPhenotype
				result.Confidence = "DAMPED"
			}
		}
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

	// Phase 4 P9: ask KB-23 to fold any active cards into a single
	// composite signal. Best-effort — composite failures never block
	// classification. The snapshot + KB-19 event remain the source of
	// truth for downstream consumers.
	o.triggerCompositeSynthesis(ctx, patientID)

	return &result, nil
}

// triggerCompositeSynthesis calls KB-23 to aggregate active cards for the
// patient into a CompositeCardSignal. Phase 4 P9 wiring — see
// KB23CompositeTrigger. Failures are logged and swallowed.
func (o *BPContextOrchestrator) triggerCompositeSynthesis(ctx context.Context, patientID string) {
	if o.kb23 == nil {
		return
	}
	if err := o.kb23.TriggerCompositeSynthesize(ctx, patientID); err != nil {
		o.log.Warn("KB-23 composite synthesise trigger failed",
			zap.String("patient_id", patientID), zap.Error(err))
	}
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

// buildStabilityHistory converts repository snapshot rows into the generic
// History shape the stability engine consumes. Snapshots are assumed to
// arrive oldest-first from FetchHistorySince.
func buildStabilityHistory(rows []models.BPContextHistory) stability.History {
	entries := make([]stability.Entry, 0, len(rows))
	for _, r := range rows {
		entries = append(entries, stability.Entry{
			State:     string(r.Phenotype),
			EnteredAt: r.SnapshotDate,
		})
	}
	return stability.History{Entries: entries}
}

// detectOverrideEvent returns true when the patient profile indicates a
// clinical event that should bypass stability dwell/flap checks. Phase 4
// stub: always returns false. Phase 5 wires this to a real medication
// change signal from KB-20 (see the MedicationChangeDate field when it
// exists on KB20PatientProfile).
func detectOverrideEvent(profile *clients.KB20PatientProfile) bool {
	// Phase 4 P2 placeholder — no override detection yet.
	return false
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

// buildBPContextInputFromReadings constructs a BPContextInput from real
// per-reading data fetched from KB-20's /bp-readings endpoint. This is
// the Phase 4 replacement for buildBPContextInputFromProfile's synthetic
// readings. It correctly populates TimeContext (MORNING/EVENING) so the
// medication timing hypothesis can fire, and distinguishes clinic vs
// home readings by the KB-20 Source field.
func buildBPContextInputFromReadings(
	profile *clients.KB20PatientProfile,
	readings []clients.KB20BPReading,
	engagementPhenotype string,
) BPContextInput {
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

	for _, r := range readings {
		bp := BPReading{
			SBP:       r.SBP,
			DBP:       r.DBP,
			Source:    r.Source,
			Timestamp: r.MeasuredAt,
		}
		// Tag time context for the medication timing hypothesis.
		// Morning window: 05:00-11:00 local (but we use UTC hour here;
		// Phase 5 should apply patient timezone).
		hour := r.MeasuredAt.Hour()
		switch {
		case hour >= 5 && hour < 11:
			bp.TimeContext = "MORNING"
		case hour >= 17 && hour < 23:
			bp.TimeContext = "EVENING"
		}

		// Source-based clinic vs home routing.
		switch r.Source {
		case "CLINIC", "OFFICE", "HOSPITAL":
			input.ClinicReadings = append(input.ClinicReadings, bp)
		default:
			// HOME_CUFF, HOME_WRIST, PATIENT_REPORTED, and empty (unknown)
			// all go to the home bucket. This matches KB-20's source mapping
			// in market-configs/shared/bp_context_thresholds.yaml.
			input.HomeReadings = append(input.HomeReadings, bp)
		}
	}
	return input
}
