// Package fixtures provides test data generators for KB-11 Population Health.
package fixtures

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb-11-population-health/internal/models"
	"github.com/cardiofit/kb-11-population-health/internal/risk"
)

// GenerateSyntheticPatients creates N synthetic patient records for scale testing.
// Distribution: 45% low, 30% moderate, 12% high, 5% very high, 3.5% rising, 4.5% unscored.
func GenerateSyntheticPatients(n int) []*risk.RiskFeatures {
	patients := make([]*risk.RiskFeatures, n)
	rng := rand.New(rand.NewSource(42)) // Deterministic seed for reproducible tests

	for i := 0; i < n; i++ {
		patients[i] = generatePatient(i, rng)
	}

	return patients
}

// GenerateSyntheticPatientProjections creates N patient projections for analytics testing.
func GenerateSyntheticPatientProjections(n int) []*models.PatientProjection {
	projections := make([]*models.PatientProjection, n)
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < n; i++ {
		projections[i] = generateProjection(i, rng)
	}

	return projections
}

// generatePatient creates a single synthetic patient with realistic distribution.
func generatePatient(index int, rng *rand.Rand) *risk.RiskFeatures {
	now := time.Now()

	// Age distribution: 18-95, skewed toward 45-75
	age := 45 + rng.Intn(30) // 45-74 base
	if rng.Float64() < 0.2 { // 20% chance of very old
		age = 75 + rng.Intn(20)
	}
	if rng.Float64() < 0.1 { // 10% chance of young
		age = 18 + rng.Intn(27)
	}

	// Gender distribution
	gender := models.GenderMale
	if rng.Float64() < 0.52 {
		gender = models.GenderFemale
	}

	// Generate conditions (higher count = higher risk)
	numConditions := rng.Intn(5)
	conditions := generateConditions(numConditions, rng)

	// Generate medications
	numMeds := rng.Intn(8)
	medications := generateMedications(numMeds, rng)

	// Generate encounters
	numEncounters := rng.Intn(6)
	encounters := generateEncounters(numEncounters, now, rng)

	// Generate labs
	numLabs := rng.Intn(5)
	labs := generateLabs(numLabs, rng)

	// Previous scores for rising risk detection
	previousScores := generatePreviousScores(rng)

	return &risk.RiskFeatures{
		PatientFHIRID: fmt.Sprintf("patient-%08d", index),
		Timestamp:     now,
		Age:           age,
		Gender:        gender,
		Conditions:    conditions,
		Medications:   medications,
		Encounters:    encounters,
		LabValues:     labs,
		PreviousScores: previousScores,
	}
}

// generateProjection creates a synthetic patient projection for analytics.
func generateProjection(index int, rng *rand.Rand) *models.PatientProjection {
	now := time.Now()

	// Risk tier distribution
	tierRoll := rng.Float64()
	var tier models.RiskTier
	var score float64

	switch {
	case tierRoll < 0.45: // 45% Low
		tier = models.RiskTierLow
		score = 0.1 + rng.Float64()*0.3
	case tierRoll < 0.75: // 30% Moderate
		tier = models.RiskTierModerate
		score = 0.4 + rng.Float64()*0.2
	case tierRoll < 0.87: // 12% High
		tier = models.RiskTierHigh
		score = 0.6 + rng.Float64()*0.2
	case tierRoll < 0.92: // 5% Very High
		tier = models.RiskTierVeryHigh
		score = 0.8 + rng.Float64()*0.2
	case tierRoll < 0.955: // 3.5% Rising
		tier = models.RiskTierRising
		score = 0.4 + rng.Float64()*0.3
	default: // 4.5% Unscored
		tier = models.RiskTierUnscored
		score = 0.0
	}

	// Care gap distribution (0-5 gaps, 30% have at least one)
	careGapCount := 0
	if rng.Float64() < 0.3 {
		careGapCount = 1 + rng.Intn(5)
	}

	practices := []string{"Primary Care West", "Downtown Medical", "Health Plus", "Community Clinic", "University Health"}
	pcps := []string{"Dr. Smith", "Dr. Johnson", "Dr. Williams", "Dr. Brown", "Dr. Davis"}

	// Create pointer values for optional fields
	mrn := fmt.Sprintf("MRN%08d", index)
	pcp := pcps[rng.Intn(len(pcps))]
	practice := practices[rng.Intn(len(practices))]

	return &models.PatientProjection{
		ID:                 uuid.New(),
		FHIRID:             fmt.Sprintf("patient-%08d", index),
		MRN:                &mrn,
		CurrentRiskTier:    tier,
		LatestRiskScore:    &score,
		CareGapCount:       careGapCount,
		AttributedPCP:      &pcp,
		AttributedPractice: &practice,
		LastSyncedAt:       now.Add(-time.Duration(rng.Intn(24)) * time.Hour),
		SyncSource:         models.SyncSourceFHIR,
		SyncVersion:        1,
	}
}

// Condition codes with risk weights
var chronicConditionCodes = []struct {
	code   string
	system string
	name   string
}{
	{"E11", "ICD-10", "Type 2 Diabetes"},
	{"I10", "ICD-10", "Essential Hypertension"},
	{"I50", "ICD-10", "Heart Failure"},
	{"J44", "ICD-10", "COPD"},
	{"N18", "ICD-10", "Chronic Kidney Disease"},
	{"E78", "ICD-10", "Hyperlipidemia"},
	{"I25", "ICD-10", "Ischemic Heart Disease"},
	{"F32", "ICD-10", "Major Depressive Disorder"},
}

func generateConditions(n int, rng *rand.Rand) []risk.ConditionFeature {
	if n == 0 {
		return nil
	}

	conditions := make([]risk.ConditionFeature, n)
	used := make(map[int]bool)

	for i := 0; i < n; i++ {
		idx := rng.Intn(len(chronicConditionCodes))
		for used[idx] {
			idx = rng.Intn(len(chronicConditionCodes))
		}
		used[idx] = true

		c := chronicConditionCodes[idx]
		conditions[i] = risk.ConditionFeature{
			Code:     c.code,
			System:   c.system,
			Display:  c.name,
			IsActive: true,
		}
	}

	return conditions
}

// Medication examples with high-risk flags
var medicationCodes = []struct {
	code     string
	display  string
	highRisk bool
}{
	{"6809", "Metformin", false},
	{"104894", "Lisinopril", false},
	{"197361", "Atorvastatin", false},
	{"7804", "Oxycodone", true},
	{"11289", "Warfarin", true},
	{"237159", "Insulin Glargine", true},
	{"6135", "Carvedilol", false},
	{"67108", "Furosemide", false},
}

func generateMedications(n int, rng *rand.Rand) []risk.MedicationFeature {
	if n == 0 {
		return nil
	}

	meds := make([]risk.MedicationFeature, n)
	used := make(map[int]bool)

	for i := 0; i < n; i++ {
		idx := rng.Intn(len(medicationCodes))
		for used[idx] {
			idx = rng.Intn(len(medicationCodes))
		}
		used[idx] = true

		m := medicationCodes[idx]
		meds[i] = risk.MedicationFeature{
			Code:     m.code,
			Display:  m.display,
			IsActive: true,
			HighRisk: m.highRisk,
		}
	}

	return meds
}

func generateEncounters(n int, now time.Time, rng *rand.Rand) []risk.EncounterFeature {
	if n == 0 {
		return nil
	}

	encounterTypes := []string{"outpatient", "inpatient", "emergency"}
	encounters := make([]risk.EncounterFeature, n)

	for i := 0; i < n; i++ {
		daysAgo := rng.Intn(180) // Last 6 months
		encType := encounterTypes[0] // Mostly outpatient
		if rng.Float64() < 0.15 {
			encType = encounterTypes[1] // 15% inpatient
		} else if rng.Float64() < 0.1 {
			encType = encounterTypes[2] // 10% ED
		}

		encounters[i] = risk.EncounterFeature{
			Type: encType,
			Date: now.AddDate(0, 0, -daysAgo),
		}
	}

	return encounters
}

// Lab codes with normal ranges
var labCodes = []struct {
	code   string
	name   string
	normal float64
}{
	{"2345-7", "Glucose", 100},
	{"4548-4", "HbA1c", 5.7},
	{"2160-0", "Creatinine", 1.0},
	{"17861-6", "Calcium", 9.5},
	{"2093-3", "Total Cholesterol", 200},
}

func generateLabs(n int, rng *rand.Rand) []risk.LabFeature {
	if n == 0 {
		return nil
	}

	labs := make([]risk.LabFeature, n)
	used := make(map[int]bool)

	for i := 0; i < n; i++ {
		idx := rng.Intn(len(labCodes))
		for used[idx] {
			idx = rng.Intn(len(labCodes))
		}
		used[idx] = true

		l := labCodes[idx]
		// 30% chance of abnormal
		value := l.normal * (0.9 + rng.Float64()*0.2) // Normal range
		isAbnormal := false
		if rng.Float64() < 0.3 {
			value = l.normal * (1.2 + rng.Float64()*0.3) // Abnormal high
			isAbnormal = true
		}

		labs[i] = risk.LabFeature{
			Code:       l.code,
			Display:    l.name,
			Value:      value,
			IsAbnormal: isAbnormal,
		}
	}

	return labs
}

func generatePreviousScores(rng *rand.Rand) []risk.HistoricalScore {
	// 60% have history
	if rng.Float64() > 0.6 {
		return nil
	}

	now := time.Now()
	numScores := 1 + rng.Intn(3)
	scores := make([]risk.HistoricalScore, numScores)

	baseScore := 0.3 + rng.Float64()*0.4

	for i := 0; i < numScores; i++ {
		daysAgo := 30 * (i + 1) // 30, 60, 90 days ago
		// Slight variation in historical scores
		scores[i] = risk.HistoricalScore{
			Score:        baseScore + (rng.Float64()*0.1 - 0.05),
			CalculatedAt: now.AddDate(0, 0, -daysAgo),
		}
	}

	return scores
}

// GenerateHighRiskPatients creates N patients with high risk factors.
// Used for testing cohort creation and risk stratification.
func GenerateHighRiskPatients(n int) []*risk.RiskFeatures {
	patients := make([]*risk.RiskFeatures, n)
	rng := rand.New(rand.NewSource(99)) // Different seed for high-risk cohort

	for i := 0; i < n; i++ {
		patients[i] = generateHighRiskPatient(i, rng)
	}

	return patients
}

func generateHighRiskPatient(index int, rng *rand.Rand) *risk.RiskFeatures {
	now := time.Now()

	// Older population for high risk
	age := 70 + rng.Intn(25)

	// Multiple chronic conditions
	conditions := generateConditions(3+rng.Intn(3), rng)

	// High-risk medications
	meds := []risk.MedicationFeature{
		{Code: "11289", Display: "Warfarin", IsActive: true, HighRisk: true},
		{Code: "237159", Display: "Insulin", IsActive: true, HighRisk: true},
	}
	if rng.Float64() < 0.5 {
		meds = append(meds, risk.MedicationFeature{Code: "7804", Display: "Oxycodone", IsActive: true, HighRisk: true})
	}

	// Recent hospitalizations
	encounters := []risk.EncounterFeature{
		{Type: "inpatient", Date: now.AddDate(0, 0, -15)},
		{Type: "emergency", Date: now.AddDate(0, 0, -45)},
	}

	// Abnormal labs
	labs := []risk.LabFeature{
		{Code: "2160-0", Display: "Creatinine", Value: 2.5, IsAbnormal: true},
		{Code: "4548-4", Display: "HbA1c", Value: 9.2, IsAbnormal: true},
	}

	// Rising scores
	previousScores := []risk.HistoricalScore{
		{Score: 0.55, CalculatedAt: now.AddDate(0, 0, -30)},
		{Score: 0.45, CalculatedAt: now.AddDate(0, 0, -60)},
	}

	return &risk.RiskFeatures{
		PatientFHIRID:  fmt.Sprintf("high-risk-%08d", index),
		Timestamp:      now,
		Age:            age,
		Gender:         models.GenderMale,
		Conditions:     conditions,
		Medications:    meds,
		Encounters:     encounters,
		LabValues:      labs,
		PreviousScores: previousScores,
	}
}
