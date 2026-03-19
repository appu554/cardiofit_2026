package services

import (
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PopulationLearningEngine aggregates technique response data across patients
// to continuously improve cold-start priors (E3 §5).
// Pipeline: Aggregate → Cluster → Update Priors → Validate → Deploy.
type PopulationLearningEngine struct {
	db            *gorm.DB
	logger        *zap.Logger
	minDeliveries int
}

func NewPopulationLearningEngine(db *gorm.DB, logger *zap.Logger, minDeliveries int) *PopulationLearningEngine {
	if minDeliveries <= 0 {
		minDeliveries = 8
	}
	return &PopulationLearningEngine{
		db:            db,
		logger:        logger,
		minDeliveries: minDeliveries,
	}
}

// AggregateTechniqueStats computes mean Alpha and Beta from eligible patient records.
// Only includes records with Deliveries >= minDeliveries (spec §5.1 step 1).
func (pl *PopulationLearningEngine) AggregateTechniqueStats(records []models.TechniqueEffectiveness) (float64, float64) {
	var sumAlpha, sumBeta float64
	count := 0
	for _, r := range records {
		if r.Deliveries >= pl.minDeliveries {
			sumAlpha += r.Alpha
			sumBeta += r.Beta
			count++
		}
	}
	if count == 0 {
		return 1.0, 1.0
	}
	return sumAlpha / float64(count), sumBeta / float64(count)
}

// ComputeClusterPriors aggregates all technique records grouped by cold-start phenotype.
// Returns a map of phenotype → technique → (Alpha, Beta).
func (pl *PopulationLearningEngine) ComputeClusterPriors() (map[models.ColdStartPhenotype]map[models.TechniqueID]models.TechniquePrior, error) {
	if pl.db == nil {
		return nil, nil
	}

	var intakes []models.IntakeProfile
	if err := pl.db.Find(&intakes).Error; err != nil {
		return nil, err
	}

	cs := NewColdStartEngine(nil, nil)
	patientPhenotype := map[string]models.ColdStartPhenotype{}
	for _, intake := range intakes {
		patientPhenotype[intake.PatientID] = cs.AssignPhenotype(intake)
	}

	var allRecords []models.TechniqueEffectiveness
	if err := pl.db.Where("deliveries >= ?", pl.minDeliveries).Find(&allRecords).Error; err != nil {
		return nil, err
	}

	type key struct {
		phenotype models.ColdStartPhenotype
		technique models.TechniqueID
	}
	grouped := map[key][]models.TechniqueEffectiveness{}
	for _, rec := range allRecords {
		phenotype, ok := patientPhenotype[rec.PatientID]
		if !ok {
			phenotype = models.PhenotypeRoutineBuilder
		}
		k := key{phenotype: phenotype, technique: rec.Technique}
		grouped[k] = append(grouped[k], rec)
	}

	result := map[models.ColdStartPhenotype]map[models.TechniqueID]models.TechniquePrior{}
	for k, records := range grouped {
		if len(records) < 3 {
			continue
		}
		alpha, beta := pl.AggregateTechniqueStats(records)
		if _, ok := result[k.phenotype]; !ok {
			result[k.phenotype] = map[models.TechniqueID]models.TechniquePrior{}
		}
		result[k.phenotype][k.technique] = models.TechniquePrior{Alpha: alpha, Beta: beta}
	}

	return result, nil
}

// ValidateImprovement checks if the accuracy improvement meets the 5% threshold (spec §5.1 step 4).
func (pl *PopulationLearningEngine) ValidateImprovement(improvement float64) bool {
	return improvement >= 0.05
}

// DeployPriors persists updated population priors to the database.
func (pl *PopulationLearningEngine) DeployPriors(priors map[models.ColdStartPhenotype]map[models.TechniqueID]models.TechniquePrior) error {
	if pl.db == nil {
		return nil
	}

	for phenotype, techPriors := range priors {
		for tech, prior := range techPriors {
			popPrior := models.PopulationPrior{
				Phenotype:  phenotype,
				Technique:  tech,
				Alpha:      prior.Alpha,
				Beta:       prior.Beta,
				SampleSize: 0,
			}
			pl.db.Where("phenotype = ? AND technique = ?", phenotype, tech).
				Assign(models.PopulationPrior{Alpha: prior.Alpha, Beta: prior.Beta}).
				FirstOrCreate(&popPrior)
		}
	}

	pl.db.Create(&models.PriorCalibrationLog{
		RunAt:   time.Now().UTC(),
		Adopted: true,
	})

	return nil
}

// GetPopulationPriors loads the latest population-derived priors from DB.
func (pl *PopulationLearningEngine) GetPopulationPriors(phenotype models.ColdStartPhenotype, tech models.TechniqueID) *models.TechniquePrior {
	if pl.db == nil {
		return nil
	}

	var prior models.PopulationPrior
	err := pl.db.Where("phenotype = ? AND technique = ?", phenotype, tech).First(&prior).Error
	if err != nil {
		return nil
	}
	return &models.TechniquePrior{Alpha: prior.Alpha, Beta: prior.Beta}
}
