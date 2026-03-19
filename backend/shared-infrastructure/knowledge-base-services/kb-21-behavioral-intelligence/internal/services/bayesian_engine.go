package services

import (
	"math"
	"math/rand"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BayesianEngine implements Thompson Sampling for per-patient technique learning.
// Each patient × technique pair has a Beta(α, β) posterior.
// On delivery: observe 7-day adherence delta → success if positive → α++, else β++.
type BayesianEngine struct {
	db     *gorm.DB
	logger *zap.Logger
	rng    *rand.Rand
}

func NewBayesianEngine(db *gorm.DB, logger *zap.Logger) *BayesianEngine {
	return &BayesianEngine{
		db:     db,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// BuildDefaultRecords creates 12 TechniqueEffectiveness records with population priors.
func (be *BayesianEngine) BuildDefaultRecords(patientID string) []models.TechniqueEffectiveness {
	techniques := models.AllTechniques()
	records := make([]models.TechniqueEffectiveness, 0, len(techniques))
	for _, tech := range techniques {
		alpha, beta := GetDefaultPriors(tech)
		records = append(records, models.TechniqueEffectiveness{
			PatientID:     patientID,
			Technique:     tech,
			Alpha:         alpha,
			Beta:          beta,
			PosteriorMean: be.PosteriorMean(alpha, beta),
		})
	}
	return records
}

// BuildPhenotypeRecords creates 12 TechniqueEffectiveness records using phenotype-calibrated priors (E1).
// Falls back to population priors if phenotype is unknown.
func (be *BayesianEngine) BuildPhenotypeRecords(patientID string, phenotype models.ColdStartPhenotype) []models.TechniqueEffectiveness {
	cs := NewColdStartEngine(nil, nil)
	priors := cs.GetPhenotypePriors(phenotype)

	techniques := models.AllTechniques()
	records := make([]models.TechniqueEffectiveness, 0, len(techniques))
	for _, tech := range techniques {
		alpha, beta := priors[tech].Alpha, priors[tech].Beta
		if alpha == 0 {
			alpha, beta = GetDefaultPriors(tech) // fallback
		}
		records = append(records, models.TechniqueEffectiveness{
			PatientID:     patientID,
			Technique:     tech,
			Alpha:         alpha,
			Beta:          beta,
			PosteriorMean: be.PosteriorMean(alpha, beta),
		})
	}
	return records
}

// PosteriorMean returns α / (α + β).
func (be *BayesianEngine) PosteriorMean(alpha, beta float64) float64 {
	if alpha+beta == 0 {
		return 0.5
	}
	return alpha / (alpha + beta)
}

// UpdatePosterior updates the Beta distribution after observing a technique outcome.
// success=true: adherence improved within 7-day observation window after delivery.
func (be *BayesianEngine) UpdatePosterior(rec *models.TechniqueEffectiveness, success bool) {
	if success {
		rec.Alpha += 1.0
		rec.Successes++
	} else {
		rec.Beta += 1.0
	}
	rec.Deliveries++
	rec.PosteriorMean = be.PosteriorMean(rec.Alpha, rec.Beta)
}

// ThompsonSelect samples from each technique's Beta posterior and returns the one
// with the highest sample. Phase multipliers (E5) are applied before comparison.
func (be *BayesianEngine) ThompsonSelect(
	records []*models.TechniqueEffectiveness,
	phaseMultipliers map[models.TechniqueID]float64,
) *models.TechniqueEffectiveness {
	if len(records) == 0 {
		return nil
	}

	var bestRecord *models.TechniqueEffectiveness
	bestSample := -1.0

	for _, rec := range records {
		sample := be.betaSample(rec.Alpha, rec.Beta)

		// Apply phase multiplier (E5) if present
		if phaseMultipliers != nil {
			if mult, ok := phaseMultipliers[rec.Technique]; ok {
				sample *= mult
			}
		}

		if sample > bestSample {
			bestSample = sample
			bestRecord = rec
		}
	}
	return bestRecord
}

// betaSample draws a sample from Beta(α, β) using the Gamma distribution method.
// Beta(α,β) = Gamma(α,1) / (Gamma(α,1) + Gamma(β,1))
func (be *BayesianEngine) betaSample(alpha, beta float64) float64 {
	if alpha <= 0 {
		alpha = 0.01
	}
	if beta <= 0 {
		beta = 0.01
	}
	x := be.gammaSample(alpha)
	y := be.gammaSample(beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// gammaSample draws from Gamma(α, 1) using Marsaglia and Tsang's method.
func (be *BayesianEngine) gammaSample(alpha float64) float64 {
	if alpha < 1.0 {
		// For α < 1, use the relation: Gamma(α) = Gamma(α+1) * U^(1/α)
		return be.gammaSample(alpha+1.0) * math.Pow(be.rng.Float64(), 1.0/alpha)
	}
	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		var x, v float64
		for {
			x = be.rng.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := be.rng.Float64()
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// EnsurePatientRecords loads or creates technique effectiveness records for a patient.
func (be *BayesianEngine) EnsurePatientRecords(patientID string) ([]*models.TechniqueEffectiveness, error) {
	if be.db == nil {
		return nil, nil
	}

	var existing []models.TechniqueEffectiveness
	if err := be.db.Where("patient_id = ?", patientID).Find(&existing).Error; err != nil {
		return nil, err
	}

	// If all 12 exist, return them
	if len(existing) == 12 {
		ptrs := make([]*models.TechniqueEffectiveness, len(existing))
		for i := range existing {
			ptrs[i] = &existing[i]
		}
		return ptrs, nil
	}

	// Create missing techniques
	existingMap := map[models.TechniqueID]bool{}
	for _, e := range existing {
		existingMap[e.Technique] = true
	}

	defaults := be.BuildDefaultRecords(patientID)
	for _, d := range defaults {
		if !existingMap[d.Technique] {
			if err := be.db.Create(&d).Error; err != nil {
				if be.logger != nil {
					be.logger.Warn("failed to create technique record",
						zap.String("patient_id", patientID),
						zap.String("technique", string(d.Technique)),
						zap.Error(err))
				}
			}
		}
	}

	// Reload all
	var all []models.TechniqueEffectiveness
	if err := be.db.Where("patient_id = ?", patientID).Find(&all).Error; err != nil {
		return nil, err
	}
	ptrs := make([]*models.TechniqueEffectiveness, len(all))
	for i := range all {
		ptrs[i] = &all[i]
	}
	return ptrs, nil
}

// SaveRecord persists an updated technique effectiveness record.
func (be *BayesianEngine) SaveRecord(rec *models.TechniqueEffectiveness) error {
	if be.db == nil {
		return nil
	}
	return be.db.Save(rec).Error
}
