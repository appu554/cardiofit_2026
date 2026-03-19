package services

import (
	"context"
	"encoding/json"
	"time"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// EventProcessor handles incoming events from KB-20, KB-21, and V-MCU,
// updating the twin state accordingly.
type EventProcessor struct {
	twinUpdater *TwinUpdater
	mriScorer   *MRIScorer
	kb22Client  *clients.KB22Client
	logger      *zap.Logger
}

func NewEventProcessor(twinUpdater *TwinUpdater, mriScorer *MRIScorer, kb22Client *clients.KB22Client, logger *zap.Logger) *EventProcessor {
	return &EventProcessor{
		twinUpdater: twinUpdater,
		mriScorer:   mriScorer,
		kb22Client:  kb22Client,
		logger:      logger,
	}
}

// ProcessObservation applies a lab/vital observation to the twin state.
func (ep *EventProcessor) ProcessObservation(ctx context.Context, event models.ObservationEvent) error {
	patientID, err := uuid.Parse(event.PatientID)
	if err != nil {
		return err
	}

	existing, err := ep.twinUpdater.GetLatest(patientID)
	if err != nil {
		// No existing twin — create a new one seeded from this observation.
		existing = &models.TwinState{
			PatientID: patientID,
		}
	}

	newTwin := *existing
	newTwin.ID = uuid.New()
	newTwin.UpdateSource = "KB20_OBSERVATION"
	newTwin.UpdatedAt = time.Now().UTC()

	// Map observation code to twin state field
	switch event.Code {
	case "FBG", "fbg", "fasting_blood_glucose":
		newTwin.FBG7dMean = &event.Value
	case "PPBG", "ppbg", "postprandial_blood_glucose":
		newTwin.PPBG7dMean = &event.Value
	case "HbA1c", "hba1c":
		newTwin.HbA1c = &event.Value
		ts := event.Timestamp
		newTwin.HbA1cDate = &ts
	case "SBP", "sbp", "systolic_bp":
		newTwin.SBP14dMean = &event.Value
	case "DBP", "dbp", "diastolic_bp":
		newTwin.DBP14dMean = &event.Value
	case "eGFR", "egfr":
		newTwin.EGFR = &event.Value
		ts := event.Timestamp
		newTwin.EGFRDate = &ts
	case "waist_cm", "waist":
		newTwin.WaistCm = &event.Value
	case "weight_kg", "weight":
		newTwin.WeightKg = &event.Value
	case "bmi":
		newTwin.BMI = &event.Value
	case "resting_hr", "heart_rate":
		newTwin.RestingHR = &event.Value
	case "daily_steps":
		newTwin.DailySteps7dMean = &event.Value
	case "sleep_quality", "sleep_score":
		newTwin.SleepQuality = &event.Value
	default:
		ep.logger.Debug("unrecognised observation code — skipping twin update",
			zap.String("code", event.Code))
		return nil
	}

	// Re-derive MAP if both SBP and DBP present
	if newTwin.SBP14dMean != nil && newTwin.DBP14dMean != nil {
		mapVal := ComputeMAP(*newTwin.SBP14dMean, *newTwin.DBP14dMean)
		newTwin.MAPValue = &mapVal
	}

	if err := ep.twinUpdater.CreateSnapshot(&newTwin); err != nil {
		return err
	}

	// Recompute MRI after twin update.
	// Capture values by parameter to avoid data race on stack variables.
	if ep.mriScorer != nil {
		twinCopy := newTwin
		pidStr := event.PatientID
		go func(pid uuid.UUID, twin models.TwinState) {
			input := TwinToMRIScorerInput(&twin)
			ep.enrichMRIInputFromKB22(context.Background(), pidStr, &input)
			history := ep.mriScorer.GetHistoryScores(pid)
			result := ep.mriScorer.ComputeMRI(input, history)
			if _, err := ep.mriScorer.PersistScore(pid, result, &twin.ID); err != nil {
				ep.logger.Error("failed to recompute MRI on observation", zap.Error(err))
			}
		}(patientID, twinCopy)
	}
	return nil
}

// ProcessCheckin applies a patient self-report check-in to the twin state.
func (ep *EventProcessor) ProcessCheckin(ctx context.Context, event models.CheckinEvent) error {
	patientID, err := uuid.Parse(event.PatientID)
	if err != nil {
		return err
	}

	existing, err := ep.twinUpdater.GetLatest(patientID)
	if err != nil {
		existing = &models.TwinState{
			PatientID: patientID,
		}
	}

	newTwin := *existing
	newTwin.ID = uuid.New()
	newTwin.UpdateSource = "KB21_CHECKIN"
	newTwin.UpdatedAt = time.Now().UTC()

	// Update Tier 2 lifestyle fields from check-in data
	if event.MealQuality > 0 {
		newTwin.DietQualityScore = &event.MealQuality
	}
	if event.StepCount > 0 {
		steps := float64(event.StepCount)
		newTwin.DailySteps7dMean = &steps
	}
	if event.ExerciseDone {
		compliance := 1.0
		newTwin.ExerciseCompliance = &compliance
	}

	return ep.twinUpdater.CreateSnapshot(&newTwin)
}

// ProcessMedChange records a medication change event in the twin state.
func (ep *EventProcessor) ProcessMedChange(ctx context.Context, event models.MedChangeEvent) error {
	patientID, err := uuid.Parse(event.PatientID)
	if err != nil {
		return err
	}

	existing, err := ep.twinUpdater.GetLatest(patientID)
	if err != nil {
		existing = &models.TwinState{
			PatientID: patientID,
		}
	}

	newTwin := *existing
	newTwin.ID = uuid.New()
	newTwin.UpdateSource = "VMCU_MED_CHANGE"
	newTwin.UpdatedAt = time.Now().UTC()

	// Med changes don't directly update twin fields, but we snapshot the state
	// with the new source so the timeline reflects the change point.
	return ep.twinUpdater.CreateSnapshot(&newTwin)
}

// enrichMRIInputFromKB22 overlays KB-22 authoritative PM-04/PM-07 values
// onto an MRIScorerInput. Falls back to existing values if KB-22 unavailable.
func (ep *EventProcessor) enrichMRIInputFromKB22(ctx context.Context, patientID string, input *MRIScorerInput) {
	if ep.kb22Client == nil {
		return
	}

	// PM-04: BP dipping classification (authoritative source)
	if dipping := ep.kb22Client.GetBPDipping(ctx, patientID); dipping != "" {
		input.BPDipping = dipping
	}

	// PM-07: Sleep quality score (authoritative source)
	if sleepQ := ep.kb22Client.GetSleepQuality(ctx, patientID); sleepQ >= 0 {
		input.SleepScore = sleepQ
	}
}

// MarshalEstimated serializes an EstimatedVariable to JSONB.
func MarshalEstimated(ev models.EstimatedVariable) datatypes.JSON {
	data, _ := json.Marshal(ev)
	return data
}

// TwinToMRIScorerInput maps TwinState to MRIScorerInput for use within services package.
func TwinToMRIScorerInput(twin *models.TwinState) MRIScorerInput {
	input := MRIScorerInput{Sex: "M"}

	if twin.FBG7dMean != nil {
		input.FBG = *twin.FBG7dMean
	}
	if twin.PPBG7dMean != nil {
		input.PPBG = *twin.PPBG7dMean
	}
	if twin.HbA1cTrend != nil {
		input.HbA1cTrend = *twin.HbA1cTrend
	}
	if twin.WaistCm != nil {
		input.WaistCm = *twin.WaistCm
	}
	if twin.WeightTrend != nil {
		input.WeightTrend = *twin.WeightTrend
	}
	if twin.SBP14dMean != nil {
		input.SBP = *twin.SBP14dMean
	}
	if twin.SBPTrend != nil {
		input.SBPTrend = *twin.SBPTrend
	}
	if twin.BPDippingPattern != nil {
		input.BPDipping = *twin.BPDippingPattern
	}
	if twin.DailySteps7dMean != nil {
		input.Steps = *twin.DailySteps7dMean
	}
	if twin.ProteinAdequacy != nil {
		input.ProteinGKg = *twin.ProteinAdequacy
	}
	if twin.SleepQuality != nil {
		input.SleepScore = *twin.SleepQuality
	}
	if twin.BMI != nil {
		input.BMI = *twin.BMI
	}

	if len(twin.MuscleMassProxy) > 0 {
		var ev models.EstimatedVariable
		if err := json.Unmarshal(twin.MuscleMassProxy, &ev); err == nil {
			input.MuscleSTS = 6 + ev.Value*12
		}
	}

	return input
}
