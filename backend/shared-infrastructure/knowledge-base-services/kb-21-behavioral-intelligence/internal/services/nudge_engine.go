package services

import (
	"fmt"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NudgeEngine orchestrates the BCE v1.0 coaching pipeline:
// 1. Evaluate patient phase (E5)
// 2. Check barrier signals
// 3. Select technique (Bayesian + phase multipliers)
// 4. Enforce fatigue rules (cooldown, daily limit)
// 5. Create NudgeRecord for delivery
// 6. Track response and update posterior
type NudgeEngine struct {
	db               *gorm.DB
	logger           *zap.Logger
	bayesian         *BayesianEngine
	phaseEngine      *PhaseEngine
	barrierDiag      *BarrierDiagnostic
	coldStart        *ColdStartEngine
	gamification     *GamificationEngine
	timing           *TimingBandit
	seasonCoach      *SeasonCoach
	maxNudgesPerDay  int
	cooldownDuration time.Duration
}

func NewNudgeEngine(
	db *gorm.DB,
	logger *zap.Logger,
	bayesian *BayesianEngine,
	phaseEngine *PhaseEngine,
	barrierDiag *BarrierDiagnostic,
	coldStart *ColdStartEngine,
	gamification *GamificationEngine,
	timing *TimingBandit,
	maxPerDay int,
	cooldownHours int,
) *NudgeEngine {
	if bayesian == nil {
		bayesian = NewBayesianEngine(db, logger)
	}
	if phaseEngine == nil {
		phaseEngine = NewPhaseEngine(db, logger)
	}
	if barrierDiag == nil {
		barrierDiag = NewBarrierDiagnostic(db, logger)
	}
	if coldStart == nil {
		coldStart = NewColdStartEngine(db, logger)
	}
	if gamification == nil {
		gamification = NewGamificationEngine(db, logger)
	}
	if timing == nil {
		timing = NewTimingBandit(db, logger)
	}
	return &NudgeEngine{
		db:               db,
		logger:           logger,
		bayesian:         bayesian,
		phaseEngine:      phaseEngine,
		barrierDiag:      barrierDiag,
		coldStart:        coldStart,
		gamification:     gamification,
		timing:           timing,
		maxNudgesPerDay:  maxPerDay,
		cooldownDuration: time.Duration(cooldownHours) * time.Hour,
	}
}

// SetSeasonCoach injects a SeasonCoach for season-aware cadence modulation.
func (ne *NudgeEngine) SetSeasonCoach(sc *SeasonCoach) {
	ne.seasonCoach = sc
}

// NudgeRequest contains the context for selecting a nudge.
type NudgeRequest struct {
	PatientID string
	Channel   models.InteractionChannel
	Language  string

	// Current behavioral state (from adherence/engagement services)
	AdherenceScore   float64
	AdherenceScore7d float64 // 7-day unweighted — used for recovery exit (spec: "adherence returns to >= 0.50 for 7 days")
	AdherenceTrend   models.AdherenceTrend
	Phenotype        models.BehavioralPhenotype

	// Barrier signals
	Signals BarrierSignals

	// Engagement season context (Patient Engagement Loop)
	Season          models.EngagementSeason `json:"season,omitempty"`
	HasTriggerEvent bool                    `json:"has_trigger_event,omitempty"`
}

// StreakContext provides streak data to include in nudge messages.
type StreakContext struct {
	Behavior      string `json:"behavior"`
	CurrentStreak int    `json:"current_streak"`
	AtRisk        bool   `json:"at_risk"` // streak will break tomorrow if no action
}

// NudgeResult is the selected nudge ready for delivery.
type NudgeResult struct {
	Technique     models.TechniqueID
	TechniqueName string
	NudgeType     models.NudgeType
	Phase         models.MotivationPhase
	Barrier       *models.BarrierCode // nil if no barrier detected
	Reason        string

	// E2: Gamification context (nil if gamification not active for this patient)
	StreakInfo *StreakContext `json:"streak_info,omitempty"`

	// E4: Optimal delivery time
	OptimalSlot models.TimingSlot `json:"optimal_slot,omitempty"`
}

// SelectNudge chooses the best technique for a patient at this moment.
func (ne *NudgeEngine) SelectNudge(req NudgeRequest) (*NudgeResult, error) {
	// 0. Season-aware event trigger gate: in event-triggered seasons (S3-S5),
	// skip nudge entirely unless a trigger event occurred
	if ne.seasonCoach != nil && req.Season != "" {
		if ne.seasonCoach.IsEventTriggered(req.Season) && !req.HasTriggerEvent {
			return nil, nil // event-triggered season without event — skip
		}
	}

	// 1. Check daily nudge limit (season-aware)
	if ne.db != nil {
		todayCount, err := ne.nudgesToday(req.PatientID)
		if err != nil {
			return nil, fmt.Errorf("nudge count check: %w", err)
		}
		maxNudges := ne.maxNudgesPerDay
		if ne.seasonCoach != nil && req.Season != "" {
			maxNudges = ne.seasonCoach.GetMaxNudgesPerDay(req.Season)
		}
		if todayCount >= maxNudges {
			return nil, nil // daily limit reached — no nudge
		}
	}

	// 2. Get or create motivation phase
	phase, err := ne.phaseEngine.GetOrCreatePhase(req.PatientID)
	if err != nil {
		return nil, fmt.Errorf("phase lookup: %w", err)
	}

	// 3. Check if recovery phase should be entered
	daysInactive := 0
	if req.Phenotype == models.PhenotypeDormant || req.Phenotype == models.PhenotypeChurned {
		daysInactive = 14
	}
	if phase.Phase != models.PhaseRecovery && ne.phaseEngine.ShouldEnterRecovery(req.AdherenceScore, req.AdherenceTrend, daysInactive) {
		ne.phaseEngine.TransitionPhase(phase, models.PhaseRecovery)
	}

	// 4. Check if recovery should exit (spec: "adherence returns to >= 0.50 for 7 days")
	// Uses 7-day unweighted score, not 30-day weighted, per spec Section 7.1.
	if phase.Phase == models.PhaseRecovery && ne.phaseEngine.ShouldExitRecovery(req.AdherenceScore7d) {
		// Drop one phase below pre-recovery phase (spec Section 7.1)
		exitPhase := ne.dropOnePhase(phase.PreRecoveryPhase)
		ne.phaseEngine.TransitionPhase(phase, exitPhase)
	}

	// 5. Diagnose barriers
	barriers := ne.barrierDiag.Diagnose(req.Signals)

	// 6. Get technique records (with cold-start phenotype priors if available)
	records, err := ne.bayesian.EnsurePatientRecords(req.PatientID)
	if err != nil {
		return nil, fmt.Errorf("technique records: %w", err)
	}
	if len(records) == 0 {
		// Cold-start: use phenotype-calibrated priors if engine available
		if ne.coldStart != nil {
			phenotype, _ := ne.coldStart.GetOrAssignPhenotype(req.PatientID)
			defaults := ne.bayesian.BuildPhenotypeRecords(req.PatientID, phenotype)
			for i := range defaults {
				records = append(records, &defaults[i])
			}
		} else {
			defaults := ne.bayesian.BuildDefaultRecords(req.PatientID)
			for i := range defaults {
				records = append(records, &defaults[i])
			}
		}
	}

	// 7. Filter out fatigued techniques
	available := ne.filterFatigued(records)
	if len(available) == 0 {
		return nil, nil // all techniques fatigued — skip this nudge window
	}

	// 8. Select technique via Thompson Sampling with phase multipliers
	selected := ne.selectTechniqueForPhase(available, phase)
	if selected == nil {
		return nil, nil
	}

	// 8a. Gamification context (E2) — only if active for this patient
	var streakCtx *StreakContext
	if ne.gamification != nil {
		// Determine activation from cold-start phenotype and T-06 posterior
		phenotype := models.PhenotypeRoutineBuilder // default
		if ne.coldStart != nil {
			phenotype, _ = ne.coldStart.GetOrAssignPhenotype(req.PatientID)
		}
		var t06Posterior float64
		for _, r := range records {
			if r.Technique == models.TechProgressVisualization {
				t06Posterior = r.PosteriorMean
				break
			}
		}
		if ne.gamification.ShouldActivate(phenotype, t06Posterior) {
			streak, _ := ne.gamification.GetOrCreateStreak(req.PatientID, "MEDICATION_TAKEN")
			if streak != nil && streak.CurrentStreak > 0 {
				streakCtx = &StreakContext{
					Behavior:      streak.Behavior,
					CurrentStreak: streak.CurrentStreak,
					AtRisk:        true, // streak active means it's at risk each day
				}
			}
		}
	}

	// 8b. Optimal delivery time (E4)
	var optimalSlot models.TimingSlot
	if ne.timing != nil {
		slot, _ := ne.timing.GetOptimalTime(req.PatientID)
		optimalSlot = slot
	}

	// 9. Build result
	result := &NudgeResult{
		Technique:     selected.Technique,
		TechniqueName: TechniqueLibrary[selected.Technique].Name,
		NudgeType:     ne.mapToNudgeType(selected.Technique, barriers),
		Phase:         phase.Phase,
		Reason:        fmt.Sprintf("phase=%s, posterior_mean=%.3f", phase.Phase, selected.PosteriorMean),
		StreakInfo:    streakCtx,
		OptimalSlot:   optimalSlot,
	}

	if len(barriers) > 0 {
		result.Barrier = &barriers[0].Barrier
		result.Reason += fmt.Sprintf(", barrier=%s", barriers[0].Barrier)
	}

	return result, nil
}

// RecordDelivery creates a NudgeRecord and updates the technique's last_delivered timestamp.
func (ne *NudgeEngine) RecordDelivery(patientID string, result *NudgeResult, channel models.InteractionChannel, lang string) (*models.NudgeRecord, error) {
	now := time.Now().UTC()
	record := &models.NudgeRecord{
		PatientID:     patientID,
		NudgeType:     result.NudgeType,
		Technique:     result.Technique,
		Channel:       channel,
		Language:      lang,
		TriggerReason: result.Reason,
		SentAt:        now,
	}
	if result.Barrier != nil {
		record.BarrierCode = *result.Barrier
	}

	if ne.db != nil {
		if err := ne.db.Create(record).Error; err != nil {
			return nil, err
		}

		// Update last_delivered on the technique record
		ne.db.Model(&models.TechniqueEffectiveness{}).
			Where("patient_id = ? AND technique = ?", patientID, result.Technique).
			Update("last_delivered", now)
	}

	return record, nil
}

// ObserveOutcome updates the Bayesian posterior after observing the 7-day adherence delta.
func (ne *NudgeEngine) ObserveOutcome(patientID string, technique models.TechniqueID, success bool) error {
	if ne.db == nil {
		return nil
	}

	var rec models.TechniqueEffectiveness
	if err := ne.db.Where("patient_id = ? AND technique = ?", patientID, technique).First(&rec).Error; err != nil {
		return err
	}

	ne.bayesian.UpdatePosterior(&rec, success)
	return ne.bayesian.SaveRecord(&rec)
}

// GetPatientTechniques returns all technique effectiveness records for a patient.
func (ne *NudgeEngine) GetPatientTechniques(patientID string) ([]*models.TechniqueEffectiveness, error) {
	return ne.bayesian.EnsurePatientRecords(patientID)
}

// GetPatientPhase returns the patient's current motivation phase.
func (ne *NudgeEngine) GetPatientPhase(patientID string) (*models.PatientMotivationPhase, error) {
	return ne.phaseEngine.GetOrCreatePhase(patientID)
}

// --- Internal helpers ---

func (ne *NudgeEngine) selectTechniqueForPhase(records []*models.TechniqueEffectiveness, phase *models.PatientMotivationPhase) *models.TechniqueEffectiveness {
	multipliers := ne.phaseEngine.GetMultipliers(phase.Phase)
	return ne.bayesian.ThompsonSelect(records, multipliers)
}

func (ne *NudgeEngine) isFatigued(rec *models.TechniqueEffectiveness, cooldown time.Duration) bool {
	if rec.LastDelivered == nil {
		return false
	}
	return time.Since(*rec.LastDelivered) < cooldown
}

func (ne *NudgeEngine) filterFatigued(records []*models.TechniqueEffectiveness) []*models.TechniqueEffectiveness {
	var available []*models.TechniqueEffectiveness
	for _, r := range records {
		if !ne.isFatigued(r, ne.cooldownDuration) {
			available = append(available, r)
		}
	}
	return available
}

func (ne *NudgeEngine) nudgesToday(patientID string) (int, error) {
	var count int64
	today := time.Now().UTC().Truncate(24 * time.Hour)
	err := ne.db.Model(&models.NudgeRecord{}).
		Where("patient_id = ? AND sent_at >= ?", patientID, today).
		Count(&count).Error
	return int(count), err
}

func (ne *NudgeEngine) dropOnePhase(phase models.MotivationPhase) models.MotivationPhase {
	switch phase {
	case models.PhaseMastery:
		return models.PhaseConsolidation
	case models.PhaseConsolidation:
		return models.PhaseExploration
	case models.PhaseExploration:
		return models.PhaseInitiation
	default:
		return models.PhaseInitiation
	}
}

func (ne *NudgeEngine) mapToNudgeType(tech models.TechniqueID, barriers []DiagnosedBarrier) models.NudgeType {
	if tech == models.TechRecoveryProtocol {
		return models.NudgeReEngagement
	}
	if len(barriers) > 0 {
		return models.NudgeBarrierSupport
	}
	switch tech {
	case models.TechMicroEducation:
		return models.NudgeEducational
	case models.TechProgressVisualization:
		return models.NudgePositiveReinforce
	default:
		return models.NudgeReminder
	}
}
