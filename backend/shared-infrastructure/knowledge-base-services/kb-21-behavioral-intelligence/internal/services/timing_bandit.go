package services

import (
	"math"
	"math/rand"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TimingBandit implements a contextual multi-armed bandit for message delivery time optimization (E4 §6).
// Each arm is a time-of-day slot. Thompson Sampling on response latency reward:
// responded within 30 min = success (α++), otherwise failure (β++).
type TimingBandit struct {
	db     *gorm.DB
	logger *zap.Logger
	rng    *rand.Rand
}

func NewTimingBandit(db *gorm.DB, logger *zap.Logger) *TimingBandit {
	return &TimingBandit{
		db:     db,
		logger: logger,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectDeliveryTime samples from each slot's Beta posterior and returns the slot
// with the highest sample (Thompson Sampling).
func (tb *TimingBandit) SelectDeliveryTime(profiles []*models.PatientTimingProfile) models.TimingSlot {
	if len(profiles) == 0 {
		slots := models.AllTimingSlots()
		return slots[tb.rng.Intn(len(slots))]
	}

	bestSlot := profiles[0].Slot
	bestSample := -1.0

	for _, p := range profiles {
		sample := tb.betaSample(p.Alpha, p.Beta)
		if sample > bestSample {
			bestSample = sample
			bestSlot = p.Slot
		}
	}
	return bestSlot
}

// ObserveReward updates the timing profile after observing a response.
// responded: true if patient responded within 30 min.
func (tb *TimingBandit) ObserveReward(profile *models.PatientTimingProfile, responded bool) {
	profile.Deliveries++
	if responded {
		profile.Alpha += 1.0
		profile.Responses++
	} else {
		profile.Beta += 1.0
	}
	if tb.db != nil {
		tb.db.Save(profile)
	}
}

// BuildDefaultProfiles creates 7 timing profiles with uniform priors for a new patient.
func (tb *TimingBandit) BuildDefaultProfiles(patientID string) []models.PatientTimingProfile {
	slots := models.AllTimingSlots()
	profiles := make([]models.PatientTimingProfile, 0, len(slots))
	for _, slot := range slots {
		profiles = append(profiles, models.PatientTimingProfile{
			PatientID: patientID,
			Slot:      slot,
			Alpha:     1.0,
			Beta:      1.0,
		})
	}
	return profiles
}

// EnsurePatientProfiles loads or creates timing profiles for a patient.
func (tb *TimingBandit) EnsurePatientProfiles(patientID string) ([]*models.PatientTimingProfile, error) {
	if tb.db == nil {
		defaults := tb.BuildDefaultProfiles(patientID)
		ptrs := make([]*models.PatientTimingProfile, len(defaults))
		for i := range defaults {
			ptrs[i] = &defaults[i]
		}
		return ptrs, nil
	}

	var existing []models.PatientTimingProfile
	if err := tb.db.Where("patient_id = ?", patientID).Find(&existing).Error; err != nil {
		return nil, err
	}

	if len(existing) == 7 {
		ptrs := make([]*models.PatientTimingProfile, len(existing))
		for i := range existing {
			ptrs[i] = &existing[i]
		}
		return ptrs, nil
	}

	existingMap := map[models.TimingSlot]bool{}
	for _, e := range existing {
		existingMap[e.Slot] = true
	}

	defaults := tb.BuildDefaultProfiles(patientID)
	for _, d := range defaults {
		if !existingMap[d.Slot] {
			tb.db.Create(&d)
		}
	}

	var all []models.PatientTimingProfile
	if err := tb.db.Where("patient_id = ?", patientID).Find(&all).Error; err != nil {
		return nil, err
	}
	ptrs := make([]*models.PatientTimingProfile, len(all))
	for i := range all {
		ptrs[i] = &all[i]
	}
	return ptrs, nil
}

// GetOptimalTime returns the best delivery time for a patient.
func (tb *TimingBandit) GetOptimalTime(patientID string) (models.TimingSlot, error) {
	profiles, err := tb.EnsurePatientProfiles(patientID)
	if err != nil {
		return models.Slot9AM, err
	}
	return tb.SelectDeliveryTime(profiles), nil
}

// betaSample draws from Beta(α, β) using the Gamma method.
func (tb *TimingBandit) betaSample(alpha, beta float64) float64 {
	if alpha <= 0 {
		alpha = 0.01
	}
	if beta <= 0 {
		beta = 0.01
	}
	x := tb.gammaSample(alpha)
	y := tb.gammaSample(beta)
	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

func (tb *TimingBandit) gammaSample(alpha float64) float64 {
	if alpha < 1.0 {
		return tb.gammaSample(alpha+1.0) * math.Pow(tb.rng.Float64(), 1.0/alpha)
	}
	d := alpha - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		var x, v float64
		for {
			x = tb.rng.NormFloat64()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}
		v = v * v * v
		u := tb.rng.Float64()
		if u < 1.0-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}
