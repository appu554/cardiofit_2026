package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

type PerturbationService struct {
	db      *database.Database
	cache   *cache.CacheClient
	metrics *metrics.Collector
	log     *zap.Logger
}

func NewPerturbationService(db *database.Database, c *cache.CacheClient, m *metrics.Collector, log *zap.Logger) *PerturbationService {
	return &PerturbationService{db: db, cache: c, metrics: m, log: log}
}

// Store saves a treatment perturbation and updates the Redis cache.
func (s *PerturbationService) Store(ctx context.Context, p *models.TreatmentPerturbation) error {
	if p.PerturbationID == uuid.Nil {
		p.PerturbationID = uuid.New()
	}
	p.CreatedAt = time.Now()

	if err := s.db.DB.Create(p).Error; err != nil {
		return fmt.Errorf("save perturbation: %w", err)
	}

	// Refresh the active perturbation cache for this patient
	if err := s.refreshCache(ctx, p.PatientID.String()); err != nil {
		s.log.Warn("perturbation cache refresh failed", zap.Error(err))
	}

	s.log.Info("perturbation stored",
		zap.String("perturbation_id", p.PerturbationID.String()),
		zap.String("patient_id", p.PatientID.String()),
		zap.String("type", string(p.InterventionType)),
		zap.Float64("dose_delta", p.DoseDelta),
	)

	return nil
}

// GetActive returns all active perturbations for a patient (effect_window_end > now).
// Tries Redis first, falls back to DB.
func (s *PerturbationService) GetActive(ctx context.Context, patientID string) ([]models.TreatmentPerturbation, error) {
	// Try cache
	var cached []models.TreatmentPerturbation
	if err := s.cache.GetPerturbations(patientID, &cached); err == nil {
		return cached, nil
	}

	// Fallback to DB
	var perturbations []models.TreatmentPerturbation
	result := s.db.DB.Where("patient_id = ? AND effect_window_end > ?", patientID, time.Now()).
		Order("effect_window_start DESC").
		Find(&perturbations)

	if result.Error != nil {
		return nil, fmt.Errorf("query active perturbations: %w", result.Error)
	}

	// Write back to cache with TTL based on nearest window end
	if len(perturbations) > 0 {
		ttl := s.computeCacheTTL(perturbations)
		if err := s.cache.SetPerturbations(patientID, perturbations, ttl); err != nil {
			s.log.Warn("perturbation cache write failed", zap.Error(err))
		}
	}

	return perturbations, nil
}

func (s *PerturbationService) refreshCache(ctx context.Context, patientID string) error {
	var perturbations []models.TreatmentPerturbation
	result := s.db.DB.Where("patient_id = ? AND effect_window_end > ?", patientID, time.Now()).
		Order("effect_window_start DESC").
		Find(&perturbations)

	if result.Error != nil {
		return result.Error
	}

	if len(perturbations) == 0 {
		return s.cache.Delete("kb23:perturbation:" + patientID)
	}

	ttl := s.computeCacheTTL(perturbations)
	return s.cache.SetPerturbations(patientID, perturbations, ttl)
}

// ════════════════════════════════════════════════════════════════════════
// HTN PERTURBATION WINDOW DEFINITIONS (Amendment 2)
//
// Each drug class has defined expected effects when added, stopped, or
// titrated. These windows enable Channel B and C to distinguish expected
// pharmacodynamic changes from pathological events.
// ════════════════════════════════════════════════════════════════════════

// HTNPerturbationSpec defines the expected physiological effect of an
// antihypertensive drug class change. Used by CreateHTNPerturbation to
// populate the TreatmentPerturbation with correct windows and directions.
type HTNPerturbationSpec struct {
	DrugClass            string
	InterventionType     models.InterventionType
	AffectedObservables  []string
	WindowDays           int
	ExpectedDirection    string  // UP | DOWN
	ExpectedMagnitudeMin float64
	ExpectedMagnitudeMax float64
	StabilityFactor      float64
	CausalNote           string
}

// htnPerturbationSpecs defines the 6 drug class perturbation windows
// from Amendment 2 of the HTN integration plan.
var htnPerturbationSpecs = map[string][]HTNPerturbationSpec{
	"SGLT2I_ADDED": {{
		DrugClass:            "SGLT2I",
		InterventionType:     models.IntDrugStart,
		AffectedObservables:  []string{"SBP"},
		WindowDays:           28,
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 3.0,
		ExpectedMagnitudeMax: 5.0,
		StabilityFactor:      0.5,
		CausalNote:           "SGLT2i osmotic diuresis typically lowers SBP by 3-5 mmHg over 4 weeks.",
	}},
	"SGLT2I_STOPPED": {{
		DrugClass:            "SGLT2I",
		InterventionType:     models.IntDrugStop,
		AffectedObservables:  []string{"SBP"},
		WindowDays:           28,
		ExpectedDirection:    "UP",
		ExpectedMagnitudeMin: 3.0,
		ExpectedMagnitudeMax: 5.0,
		StabilityFactor:      0.5,
		CausalNote:           "Stopping SGLT2i removes osmotic diuresis. Expect SBP rise of 3-5 mmHg over 4 weeks.",
	}},
	"ACEI_ARB_STARTED": {{
		DrugClass:            "ACE_INHIBITOR",
		InterventionType:     models.IntDrugStart,
		AffectedObservables:  []string{"CREATININE"},
		WindowDays:           14,
		ExpectedDirection:    "UP",
		ExpectedMagnitudeMin: 10.0,
		ExpectedMagnitudeMax: 30.0,
		StabilityFactor:      0.4,
		CausalNote:           "ACEi/ARB reduces efferent arteriolar tone → expected creatinine rise 10-30% within 14 days. Not AKI.",
	}},
	"ACEI_ARB_STOPPED": {{
		DrugClass:            "ACE_INHIBITOR",
		InterventionType:     models.IntDrugStop,
		AffectedObservables:  []string{"CREATININE"},
		WindowDays:           7,
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.0,
		ExpectedMagnitudeMax: 30.0,
		StabilityFactor:      0.5,
		CausalNote:           "Stopping ACEi/ARB restores efferent tone. Creatinine should normalise within 7 days.",
	}},
	"THIAZIDE_ADDED": {{
		DrugClass:            "THIAZIDE",
		InterventionType:     models.IntDrugStart,
		AffectedObservables:  []string{"POTASSIUM"},
		WindowDays:           21,
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.3,
		ExpectedMagnitudeMax: 0.5,
		StabilityFactor:      0.5,
		CausalNote:           "Thiazide-induced kaliuresis: expect K+ drop of 0.3-0.5 mmol/L over 3 weeks.",
	}},
	"BETA_BLOCKER_ADDED": {{
		DrugClass:            "BETA_BLOCKER",
		InterventionType:     models.IntDrugStart,
		AffectedObservables:  []string{"HR", "GLUCOSE"},
		WindowDays:           14,
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.0,
		ExpectedMagnitudeMax: 0.0, // variable per patient
		StabilityFactor:      0.4,
		CausalNote:           "Beta-blocker reduces HR and may impair glycogenolysis, raising fasting glucose.",
	}},
}

// CreateHTNPerturbation creates a perturbation from a known HTN drug class event.
// Returns nil if the event key is not recognized.
func (s *PerturbationService) CreateHTNPerturbation(ctx context.Context, patientID uuid.UUID, eventKey string, doseDelta, baselineDose float64) (*models.TreatmentPerturbation, error) {
	specs, ok := htnPerturbationSpecs[eventKey]
	if !ok {
		return nil, fmt.Errorf("unknown HTN perturbation event: %s", eventKey)
	}

	spec := specs[0]
	now := time.Now()
	p := &models.TreatmentPerturbation{
		PatientID:            patientID,
		InterventionType:     spec.InterventionType,
		DoseDelta:            doseDelta,
		BaselineDose:         baselineDose,
		EffectWindowStart:    now,
		EffectWindowEnd:      now.AddDate(0, 0, spec.WindowDays),
		AffectedObservables:  models.StringArray(spec.AffectedObservables),
		StabilityFactor:      spec.StabilityFactor,
		ExpectedDirection:    spec.ExpectedDirection,
		ExpectedMagnitudeMin: spec.ExpectedMagnitudeMin,
		ExpectedMagnitudeMax: spec.ExpectedMagnitudeMax,
		CausalNote:           spec.CausalNote,
	}

	if err := s.Store(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// computeCacheTTL sets TTL to the nearest EffectWindowEnd (dynamic per plan).
func (s *PerturbationService) computeCacheTTL(perturbations []models.TreatmentPerturbation) time.Duration {
	now := time.Now()
	minTTL := 24 * time.Hour // max fallback
	for _, p := range perturbations {
		ttl := p.EffectWindowEnd.Sub(now)
		if ttl > 0 && ttl < minTTL {
			minTTL = ttl
		}
	}
	return minTTL
}
