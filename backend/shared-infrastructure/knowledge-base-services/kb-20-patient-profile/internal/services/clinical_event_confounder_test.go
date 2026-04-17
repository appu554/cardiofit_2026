package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfounderWeights() *ConfounderWeights {
	return &ConfounderWeights{
		SteroidWeight: 0.35, SteroidWashoutDays: 28,
		HospWeight: 0.30, HospWashoutDays: 42,
		InfectionWeight: 0.20, InfectionWashoutDays: 21,
		AKIWeight: 0.40, AKIWashoutDays: 90,
		SurgeryWeight: 0.30, SurgeryWashoutDays: 56,
	}
}

func TestClinicalEvent_SteroidCourse(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	events := []PatientClinicalEvent{
		{Type: "MEDICATION_START", DrugName: "Prednisolone", Date: now.AddDate(0, 0, -20)},
		{Type: "MEDICATION_STOP", DrugName: "Prednisolone", Date: now.AddDate(0, 0, -6)},
	}

	windowStart := now.AddDate(0, 0, -30)
	windowEnd := now

	factors := detector.DetectConfounders(events, windowStart, windowEnd)

	var steroid *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "STEROID_COURSE" {
			steroid = &factors[i]
			break
		}
	}
	require.NotNil(t, steroid, "expected STEROID_COURSE factor")
	assert.Equal(t, "IATROGENIC", string(steroid.Category))
	assert.GreaterOrEqual(t, steroid.Weight, 0.30)
	assert.Contains(t, steroid.AffectedOutcomes, "DELTA_HBA1C")
}

func TestClinicalEvent_SteroidWashout(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	// Prednisolone ended 10 days ago — within 28d washout
	events := []PatientClinicalEvent{
		{Type: "MEDICATION_START", DrugName: "Prednisolone", Date: now.AddDate(0, 0, -20)},
		{Type: "MEDICATION_STOP", DrugName: "Prednisolone", Date: now.AddDate(0, 0, -10)},
	}

	// Window is only last 5 days — steroid itself doesn't overlap,
	// but washout (28d from stop) still does.
	windowStart := now.AddDate(0, 0, -5)
	windowEnd := now

	factors := detector.DetectConfounders(events, windowStart, windowEnd)

	found := false
	for _, f := range factors {
		if f.Name == "STEROID_COURSE" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected STEROID_COURSE via washout overlap")
}

func TestClinicalEvent_Hospitalization(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	events := []PatientClinicalEvent{
		{Type: "HOSPITALIZATION", Date: now.AddDate(0, 0, -15), Duration: 5},
	}

	windowStart := now.AddDate(0, 0, -30)
	windowEnd := now

	factors := detector.DetectConfounders(events, windowStart, windowEnd)

	var hosp *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "HOSPITALIZATION" {
			hosp = &factors[i]
			break
		}
	}
	require.NotNil(t, hosp, "expected HOSPITALIZATION factor")
	assert.GreaterOrEqual(t, hosp.Weight, 0.25)
}

func TestClinicalEvent_AcuteInfection_DetectedByAntibiotic(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	events := []PatientClinicalEvent{
		{Type: "MEDICATION_START", DrugName: "Amoxicillin", Date: now.AddDate(0, 0, -12)},
	}

	windowStart := now.AddDate(0, 0, -30)
	windowEnd := now

	factors := detector.DetectConfounders(events, windowStart, windowEnd)

	var infection *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "ACUTE_INFECTION" {
			infection = &factors[i]
			break
		}
	}
	require.NotNil(t, infection, "expected ACUTE_INFECTION factor")
	assert.Equal(t, "ACUTE_ILLNESS", string(infection.Category))
}

func TestClinicalEvent_AKI_DetectedByCreatinineSpike(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	events := []PatientClinicalEvent{
		{Type: "LAB_RESULT", LabType: "CREATININE", Value: 1.2, Date: now.AddDate(0, 0, -60)},
		{Type: "LAB_RESULT", LabType: "CREATININE", Value: 2.8, Date: now.AddDate(0, 0, -10)},
	}

	windowStart := now.AddDate(0, 0, -30)
	windowEnd := now

	factors := detector.DetectConfounders(events, windowStart, windowEnd)

	var aki *confounderFactorResult
	for i := range factors {
		if factors[i].Name == "ACUTE_KIDNEY_INJURY" {
			aki = &factors[i]
			break
		}
	}
	require.NotNil(t, aki, "expected ACUTE_KIDNEY_INJURY factor")
	assert.GreaterOrEqual(t, aki.Weight, 0.35)
	assert.Contains(t, aki.AffectedOutcomes, "DELTA_EGFR")
}

func TestClinicalEvent_NoEvents_EmptyFactors(t *testing.T) {
	now := time.Now().UTC().Truncate(24 * time.Hour)
	detector := NewClinicalEventDetector(testConfounderWeights())

	factors := detector.DetectConfounders(nil, now.AddDate(0, 0, -30), now)

	assert.Empty(t, factors, "expected empty factors for nil events")
}
