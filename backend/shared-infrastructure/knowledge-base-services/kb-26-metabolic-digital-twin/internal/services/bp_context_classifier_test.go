package services

import (
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// === CORE PHENOTYPE CLASSIFICATION ===

func TestClassifyBPContext_SustainedHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 155, DBP: 95, Source: "CLINIC"},
			{SBP: 150, DBP: 92, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 142, 88),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedHTN {
		t.Errorf("expected SUSTAINED_HTN, got %s", result.Phenotype)
	}
	if !result.ClinicAboveThreshold {
		t.Error("expected ClinicAboveThreshold = true")
	}
	if !result.HomeAboveThreshold {
		t.Error("expected HomeAboveThreshold = true")
	}
}

func TestClassifyBPContext_MaskedHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 132, DBP: 82, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 148, 92),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
	if result.ClinicAboveThreshold {
		t.Error("expected ClinicAboveThreshold = false")
	}
	if !result.HomeAboveThreshold {
		t.Error("expected HomeAboveThreshold = true")
	}
	if result.ClinicHomeGapSBP >= 0 {
		t.Errorf("expected negative gap (home > clinic), got %.1f", result.ClinicHomeGapSBP)
	}
}

func TestClassifyBPContext_WhiteCoatHTN(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 158, DBP: 96, Source: "CLINIC"},
			{SBP: 152, DBP: 94, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 125, 78),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeWhiteCoatHTN {
		t.Errorf("expected WHITE_COAT_HTN, got %s", result.Phenotype)
	}
	if result.WhiteCoatEffect < 15 {
		t.Errorf("expected WCE >= 15, got %.1f", result.WhiteCoatEffect)
	}
}

func TestClassifyBPContext_SustainedNormotension(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 122, DBP: 76, Source: "CLINIC"},
			{SBP: 118, DBP: 74, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 120, 75),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedNormotension {
		t.Errorf("expected SUSTAINED_NORMOTENSION, got %s", result.Phenotype)
	}
}

// === TREATED PATIENT PHENOTYPES ===

func TestClassifyBPContext_MaskedUncontrolled(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 132, DBP: 80, Source: "CLINIC"},
			{SBP: 128, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 145, 90),
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedUncontrolled {
		t.Errorf("expected MASKED_UNCONTROLLED, got %s", result.Phenotype)
	}
}

func TestClassifyBPContext_WhiteCoatUncontrolled(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 148, DBP: 92, Source: "CLINIC"},
			{SBP: 145, DBP: 90, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 128, 80),
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeWhiteCoatUncontrolled {
		t.Errorf("expected WHITE_COAT_UNCONTROLLED, got %s", result.Phenotype)
	}
}

// === CROSS-DOMAIN AMPLIFICATION ===

func TestClassifyBPContext_MaskedHTN_DiabeticAmplification(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 125, DBP: 76, Source: "CLINIC"},
			{SBP: 128, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 142, 88),
		IsDiabetic:   true,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
	if !result.DiabetesAmplification {
		t.Error("expected DiabetesAmplification = true")
	}
}

func TestClassifyBPContext_MaskedHTN_CKDAmplification(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 130, DBP: 80, Source: "CLINIC"},
			{SBP: 132, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(14, 144, 90),
		HasCKD:       true,
	}

	result := ClassifyBPContext(input)
	if !result.CKDAmplification {
		t.Error("expected CKDAmplification = true")
	}
}

func TestClassifyBPContext_MaskedHTN_MorningSurgeCompound(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 130, DBP: 80, Source: "CLINIC"},
		},
		HomeReadings:      generateHomeReadings(14, 140, 88),
		MorningSurge7dAvg: 28,
	}

	result := ClassifyBPContext(input)
	if !result.MorningSurgeCompound {
		t.Error("expected MorningSurgeCompound = true")
	}
}

// === SELECTION BIAS DETECTION ===

func TestClassifyBPContext_SelectionBias_MeasurementAvoidant(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 78, Source: "CLINIC"},
			{SBP: 130, DBP: 80, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(13, 155, 95),
		EngagementPhenotype: "MEASUREMENT_AVOIDANT",
	}

	result := ClassifyBPContext(input)
	if !result.SelectionBiasRisk {
		t.Error("expected SelectionBiasRisk = true")
	}
	if result.Confidence != "LOW" {
		t.Errorf("expected LOW confidence, got %s", result.Confidence)
	}
}

// === INSUFFICIENT DATA ===

func TestClassifyBPContext_InsufficientHome(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 145, DBP: 92, Source: "CLINIC"},
			{SBP: 148, DBP: 90, Source: "CLINIC"},
		},
		HomeReadings: generateHomeReadings(2, 138, 86),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeInsufficientData {
		t.Errorf("expected INSUFFICIENT_DATA, got %s", result.Phenotype)
	}
	if result.SufficientHome {
		t.Error("expected SufficientHome = false")
	}
}

func TestClassifyBPContext_InsufficientClinic(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{},
		HomeReadings:   generateHomeReadings(14, 138, 86),
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeInsufficientData {
		t.Errorf("expected INSUFFICIENT_DATA, got %s", result.Phenotype)
	}
	if result.SufficientClinic {
		t.Error("expected SufficientClinic = false")
	}
}

// === MEDICATION TIMING HYPOTHESIS ===

func TestClassifyBPContext_MedicationTimingHypothesis(t *testing.T) {
	now := time.Now()
	var allHome []BPReading
	for i := 0; i < 7; i++ {
		allHome = append(allHome, BPReading{
			SBP: 148, DBP: 92, Source: "HOME_CUFF", TimeContext: "MORNING",
			Timestamp: now.Add(time.Duration(-i*24) * time.Hour),
		})
		allHome = append(allHome, BPReading{
			SBP: 125, DBP: 78, Source: "HOME_CUFF", TimeContext: "EVENING",
			Timestamp: now.Add(time.Duration(-i*24+12) * time.Hour),
		})
	}

	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 128, DBP: 80, Source: "CLINIC"},
			{SBP: 130, DBP: 78, Source: "CLINIC"},
		},
		HomeReadings:        allHome,
		OnAntihypertensives: true,
	}

	result := ClassifyBPContext(input)
	if result.MedicationTimingHypothesis == "" {
		t.Error("expected non-empty MedicationTimingHypothesis")
	}
}

// === RAJESH KUMAR INTEGRATION ===

func TestClassifyBPContext_RajeshKumar(t *testing.T) {
	input := BPContextInput{
		ClinicReadings: []BPReading{
			{SBP: 170, DBP: 104, Source: "CLINIC"},
			{SBP: 168, DBP: 100, Source: "CLINIC"},
		},
		HomeReadings:        generateHomeReadings(14, 158, 96),
		IsDiabetic:          true,
		HasCKD:              true,
		OnAntihypertensives: true,
		MorningSurge7dAvg:   28,
	}

	result := ClassifyBPContext(input)
	if result.Phenotype != models.PhenotypeSustainedHTN {
		t.Errorf("expected SUSTAINED_HTN, got %s", result.Phenotype)
	}
	if !result.DiabetesAmplification {
		t.Error("expected DiabetesAmplification = true")
	}
	if !result.CKDAmplification {
		t.Error("expected CKDAmplification = true")
	}
	if !result.MorningSurgeCompound {
		t.Error("expected MorningSurgeCompound = true")
	}
}

// === HELPER ===

func generateHomeReadings(count int, avgSBP, avgDBP float64) []BPReading {
	readings := make([]BPReading, count)
	now := time.Now()
	for i := 0; i < count; i++ {
		readings[i] = BPReading{
			SBP:       avgSBP + float64(i%3-1)*3,
			DBP:       avgDBP + float64(i%3-1)*2,
			Source:    "HOME_CUFF",
			Timestamp: now.Add(time.Duration(-i*12) * time.Hour),
		}
	}
	return readings
}
