package services

import (
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Domain weights (spec §4.3)
const (
	weightGlucose    = 0.35
	weightBodyComp   = 0.25
	weightCardio     = 0.25
	weightBehavioral = 0.15
)

// MRIScorerInput collects all 12 signals needed for MRI computation.
// Decouples scorer from TwinState so it can also be used by simulation projection.
type MRIScorerInput struct {
	FBG         float64 // mg/dL
	PPBG        float64 // mg/dL
	HbA1cTrend  float64 // %/quarter (positive = worsening)
	WaistCm     float64
	WeightTrend float64 // kg/month (positive = gaining)
	MuscleSTS   float64 // 30s sit-to-stand count (or proxy 0-1 scaled to STS range)
	SBP         float64 // mmHg
	SBPTrend    float64 // mmHg/4 weeks
	BPDipping   string  // DIPPER, NON_DIPPER, REVERSE_DIPPER
	Steps       float64 // daily average
	ProteinGKg  float64 // g/kg/day
	SleepScore  float64 // 0-1 quality score
	Sex         string  // M or F
	BMI         float64 // kg/m² — used for BMI-aware weight trend penalty (LS-15)
}

// MRIScorer computes MRI scores and persists them.
type MRIScorer struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewMRIScorer creates a new MRIScorer.
func NewMRIScorer(db *gorm.DB, logger *zap.Logger) *MRIScorer {
	return &MRIScorer{db: db, logger: logger}
}

// ComputeGlucoseDomain computes the glucose control domain sub-score.
// Spec §4.2: 0.40*FBG + 0.40*PPBG + 0.20*HbA1cTrend
func (s *MRIScorer) ComputeGlucoseDomain(fbg, ppbg, hba1cTrend float64) models.DomainScore {
	fbgZ := NormalizeFBG(fbg)
	ppbgZ := NormalizePPBG(ppbg)
	hba1cZ := NormalizeHbA1cTrend(hba1cTrend)

	return models.DomainScore{
		Name:  "Glucose Control",
		Score: 0.40*fbgZ + 0.40*ppbgZ + 0.20*hba1cZ,
		Signals: map[string]float64{
			"FBG":         fbgZ,
			"PPBG":        ppbgZ,
			"HbA1c_trend": hba1cZ,
		},
	}
}

// ComputeBodyCompDomain computes the body composition domain sub-score.
// Spec §4.2: 0.50*waist + 0.25*weightTrend + 0.25*(-muscle)
// bmi is used to apply the LS-15 BMI-aware weight trend penalty (spec Table 2).
func (s *MRIScorer) ComputeBodyCompDomain(waist, weightTrend, muscleSTS float64, sex string, bmi float64) models.DomainScore {
	waistZ := NormalizeWaistSexSpecific(waist, sex)
	weightZ := NormalizeWeightTrendBMI(weightTrend, bmi)
	muscleZ := NormalizeMuscleFunction(muscleSTS)

	return models.DomainScore{
		Name:  "Body Composition",
		Score: 0.50*waistZ + 0.25*weightZ + 0.25*muscleZ,
		Signals: map[string]float64{
			"waist":        waistZ,
			"weight_trend": weightZ,
			"muscle":       muscleZ,
		},
	}
}

// ComputeCardioDomain computes the cardiovascular regulation domain sub-score.
// Spec §4.2: 0.45*SBP + 0.30*SBPTrend + 0.25*dipping
func (s *MRIScorer) ComputeCardioDomain(sbp, sbpTrend float64, bpDipping string) models.DomainScore {
	sbpZ := NormalizeSBP(sbp)
	sbpTrendZ := NormalizeSBPTrend(sbpTrend)
	dipZ := DippingToZScore(bpDipping)

	return models.DomainScore{
		Name:  "Cardiovascular Regulation",
		Score: 0.45*sbpZ + 0.30*sbpTrendZ + 0.25*dipZ,
		Signals: map[string]float64{
			"SBP":       sbpZ,
			"SBP_trend": sbpTrendZ,
			"dipping":   dipZ,
		},
	}
}

// ComputeBehavioralDomain computes the behavioral metabolism domain sub-score.
// Spec §4.2: 0.40*(-activity) + 0.35*(-protein) + 0.25*sleep
func (s *MRIScorer) ComputeBehavioralDomain(steps, proteinGKg, sleepScore float64) models.DomainScore {
	activityZ := NormalizeSteps(steps)
	proteinZ := NormalizeProtein(proteinGKg)
	sleepZ := ComputeSleepZScore(sleepScore)

	return models.DomainScore{
		Name:  "Behavioral Metabolism",
		Score: 0.40*activityZ + 0.35*proteinZ + 0.25*sleepZ,
		Signals: map[string]float64{
			"activity": activityZ,
			"protein":  proteinZ,
			"sleep":    sleepZ,
		},
	}
}

// ScaleToRange converts a raw z-score composite to 0-100 using a sigmoid.
// Spec §4.3: "sigmoid scaling centered at z=0"
// Centered sigmoid: z=0 → 50, z=-3 → ~5, z=+3 → ~95
func ScaleToRange(rawZ float64) float64 {
	sigmoid := 1.0 / (1.0 + math.Exp(-rawZ))
	return sigmoid * 100.0
}

// CategorizeMRI assigns a clinical category based on MRI score.
// Spec §4.4: OPTIMAL (0-25), MILD (26-50), MODERATE (51-75), HIGH (76-100)
func CategorizeMRI(score float64) string {
	switch {
	case score <= 25:
		return models.MRICategoryOptimal
	case score <= 50:
		return models.MRICategoryMildDysregulation
	case score <= 75:
		return models.MRICategoryModerateDeterioration
	default:
		return models.MRICategoryHighDeterioration
	}
}

// ComputeMRITrend determines trend direction from current score and history.
// Uses mean of last N scores vs current. Threshold: ±3 points.
func ComputeMRITrend(current float64, history []float64) string {
	if len(history) < 2 {
		return "STABLE"
	}

	// Average of last scores
	sum := 0.0
	for _, s := range history {
		sum += s
	}
	avg := sum / float64(len(history))

	diff := current - avg
	switch {
	case diff <= -3:
		return "IMPROVING"
	case diff >= 3:
		return "WORSENING"
	default:
		return "STABLE"
	}
}

// HighestDomainContributor returns the name of the domain with the highest score.
func HighestDomainContributor(domains []models.DomainScore) string {
	if len(domains) == 0 {
		return ""
	}
	best := domains[0]
	for _, d := range domains[1:] {
		if d.Score > best.Score {
			best = d
		}
	}
	return best.Name
}

// ComputeMRI runs the full MRI computation pipeline.
// historyScores: previous MRI scores for trend computation (newest last), can be nil.
func (s *MRIScorer) ComputeMRI(input MRIScorerInput, historyScores []float64) models.MRIResult {
	glucose := s.ComputeGlucoseDomain(input.FBG, input.PPBG, input.HbA1cTrend)
	bodyComp := s.ComputeBodyCompDomain(input.WaistCm, input.WeightTrend, input.MuscleSTS, input.Sex, input.BMI)
	cardio := s.ComputeCardioDomain(input.SBP, input.SBPTrend, input.BPDipping)
	behavioral := s.ComputeBehavioralDomain(input.Steps, input.ProteinGKg, input.SleepScore)

	// Weighted composite (spec §4.3)
	rawScore := weightGlucose*glucose.Score + weightBodyComp*bodyComp.Score +
		weightCardio*cardio.Score + weightBehavioral*behavioral.Score

	mri := ScaleToRange(rawScore)

	// Scale each domain to 0-100 for display
	glucose.Scaled = ScaleToRange(glucose.Score)
	bodyComp.Scaled = ScaleToRange(bodyComp.Score)
	cardio.Scaled = ScaleToRange(cardio.Score)
	behavioral.Scaled = ScaleToRange(behavioral.Score)

	domains := []models.DomainScore{glucose, bodyComp, cardio, behavioral}

	return models.MRIResult{
		Score:     mri,
		Category:  CategorizeMRI(mri),
		Trend:     ComputeMRITrend(mri, historyScores),
		TopDriver: HighestDomainContributor(domains),
		Domains:   domains,
	}
}

// PersistScore stores an MRI score in the database.
func (s *MRIScorer) PersistScore(patientID uuid.UUID, result models.MRIResult, twinStateID *uuid.UUID) (*models.MRIScore, error) {
	if s.db == nil {
		return nil, nil
	}

	// Collect all signal z-scores into a flat map
	signalZ := make(map[string]float64)
	for _, d := range result.Domains {
		for k, v := range d.Signals {
			signalZ[k] = v
		}
	}

	score := &models.MRIScore{
		PatientID:        patientID,
		Score:            result.Score,
		Category:         result.Category,
		Trend:            result.Trend,
		TopDriver:        result.TopDriver,
		GlucoseDomain:    result.Domains[0].Scaled,
		BodyCompDomain:   result.Domains[1].Scaled,
		CardioDomain:     result.Domains[2].Scaled,
		BehavioralDomain: result.Domains[3].Scaled,
		SignalZScores:    signalZ,
		TwinStateID:      twinStateID,
		ComputedAt:       time.Now().UTC(),
	}

	if err := s.db.Create(score).Error; err != nil {
		return nil, err
	}
	return score, nil
}

// GetHistoryScores returns recent MRI score values for trend computation.
func (s *MRIScorer) GetHistoryScores(patientID uuid.UUID) []float64 {
	if s.db == nil {
		return nil
	}

	var scores []models.MRIScore
	s.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		Limit(10).
		Select("score").
		Find(&scores)

	result := make([]float64, len(scores))
	// Reverse so oldest is first
	for i, sc := range scores {
		result[len(scores)-1-i] = sc.Score
	}
	return result
}

// GetHistory returns recent MRI score records for the history endpoint.
func (s *MRIScorer) GetHistory(patientID uuid.UUID, limit int) ([]models.MRIScore, error) {
	if s.db == nil {
		return nil, nil
	}
	var scores []models.MRIScore
	result := s.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		Limit(limit).
		Find(&scores)
	return scores, result.Error
}

// GetLatest returns the most recent MRI score for a patient.
func (s *MRIScorer) GetLatest(patientID uuid.UUID) (*models.MRIScore, error) {
	if s.db == nil {
		return nil, fmt.Errorf("no database")
	}
	var score models.MRIScore
	result := s.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}
