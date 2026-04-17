package services

import (
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// IORConfounderAssessor combines the confounder calendar, clinical event
// detector, and enhanced scorer into a single Assess call.
type IORConfounderAssessor struct {
	calendar      *ConfounderCalendar
	eventDetector *ClinicalEventDetector
	scorer        *EnhancedConfounderScorer
	db            *gorm.DB
	log           *zap.Logger
}

func NewIORConfounderAssessor(
	calendar *ConfounderCalendar,
	eventDetector *ClinicalEventDetector,
	db *gorm.DB,
	log *zap.Logger,
) *IORConfounderAssessor {
	return &IORConfounderAssessor{
		calendar:      calendar,
		eventDetector: eventDetector,
		scorer:        NewEnhancedConfounderScorer(),
		db:            db,
		log:           log,
	}
}

// Assess computes the full confounder result for a patient's outcome window.
func (a *IORConfounderAssessor) Assess(
	patientID string,
	windowStart, windowEnd time.Time,
	outcomeType string,
	religiousAffiliation string,
	region string,
	concurrentMedCount int,
	adherenceDrop float64,
) models.EnhancedConfounderResult {
	// 1. Calendar confounders
	var calendarFactors []models.ConfounderFactor
	if a.calendar != nil {
		calendarFactors = a.calendar.FindActiveConfounders(
			windowStart, windowEnd, religiousAffiliation, region)
	}

	// 2. Clinical event confounders from safety_events + medication records
	var clinicalFactors []models.ConfounderFactor
	if a.eventDetector != nil && a.db != nil {
		events := a.fetchPatientClinicalEvents(patientID, windowStart, windowEnd)
		clinicalFactors = a.eventDetector.DetectConfounders(events, windowStart, windowEnd)
	}

	// 3. Compute enhanced score
	return a.scorer.Compute(EnhancedConfounderInput{
		ConcurrentMedCount:   concurrentMedCount,
		AdherenceDrop:        adherenceDrop,
		CalendarFactors:      calendarFactors,
		ClinicalEventFactors: clinicalFactors,
		OutcomeType:          outcomeType,
		DeferOnRamadan:       true,
		DeferOnSteroid:       true,
	})
}

// fetchPatientClinicalEvents queries safety_events and medication_states
// to build the clinical event list for the detector.
func (a *IORConfounderAssessor) fetchPatientClinicalEvents(
	patientID string,
	windowStart, windowEnd time.Time,
) []PatientClinicalEvent {
	var events []PatientClinicalEvent
	lookback := windowStart.AddDate(0, 0, -90) // extend lookback for baselines

	// Safety events (hospitalization, acute illness)
	var safetyEvents []models.SafetyEvent
	if err := a.db.Where("patient_id = ? AND observed_at BETWEEN ? AND ?",
		patientID, lookback, windowEnd).Find(&safetyEvents).Error; err != nil {
		a.log.Warn("IOR assessor: failed to fetch safety events",
			zap.String("patient_id", patientID), zap.Error(err))
	}
	for _, se := range safetyEvents {
		events = append(events, PatientClinicalEvent{
			Type: se.EventType,
			Date: se.ObservedAt,
		})
	}

	// Medication records (steroid courses, antibiotics)
	var meds []models.MedicationState
	if err := a.db.Where("patient_id = ? AND start_date BETWEEN ? AND ?",
		patientID, lookback, windowEnd).Find(&meds).Error; err != nil {
		a.log.Warn("IOR assessor: failed to fetch medication states",
			zap.String("patient_id", patientID), zap.Error(err))
	}
	for _, m := range meds {
		events = append(events, PatientClinicalEvent{
			Type:     "MEDICATION_START",
			DrugName: m.DrugName,
			Date:     m.StartDate,
		})
	}

	// Lab results (creatinine for AKI detection)
	var labs []models.LabEntry
	if err := a.db.Where("patient_id = ? AND lab_type = ? AND measured_at BETWEEN ? AND ?",
		patientID, models.LabTypeCreatinine, lookback, windowEnd).Find(&labs).Error; err != nil {
		a.log.Warn("IOR assessor: failed to fetch lab entries",
			zap.String("patient_id", patientID), zap.Error(err))
	}
	for _, l := range labs {
		val, _ := l.Value.Float64()
		events = append(events, PatientClinicalEvent{
			Type:    "LAB_RESULT",
			LabType: models.LabTypeCreatinine,
			Value:   val,
			Date:    l.MeasuredAt,
		})
	}

	return events
}
