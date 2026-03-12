package fhir

import (
	"encoding/json"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// Publisher subscribes to KB-20 events and writes them back to the FHIR Store
// as DetectedIssue (threshold crossings) and Condition (CKD status) resources.
// Only active when FHIR_WRITE_BACK=true.
type Publisher struct {
	client *FHIRClient
	db     *gorm.DB
	logger *zap.Logger
}

// NewPublisher creates a FHIR write-back publisher.
func NewPublisher(client *FHIRClient, db *gorm.DB, logger *zap.Logger) *Publisher {
	return &Publisher{
		client: client,
		db:     db,
		logger: logger,
	}
}

// HandleThresholdCrossed writes a FHIR DetectedIssue when a medication threshold is crossed.
func (p *Publisher) HandleThresholdCrossed(event models.Event) {
	// Unmarshal the payload
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		p.logger.Error("Failed to marshal threshold event payload", zap.Error(err))
		return
	}

	var payload models.MedicationThresholdCrossedPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		p.logger.Error("Failed to unmarshal threshold event", zap.Error(err))
		return
	}

	// Resolve FHIR Patient ID
	fhirPatientID := p.resolveFHIRPatientID(event.PatientID)
	if fhirPatientID == "" {
		p.logger.Warn("Cannot write-back DetectedIssue: no FHIR Patient ID",
			zap.String("patient_id", event.PatientID))
		return
	}

	issue := ThresholdCrossingToDetectedIssue(fhirPatientID, &payload)

	if err := p.client.UpsertDetectedIssue(issue); err != nil {
		p.logger.Error("Failed to write DetectedIssue to FHIR Store",
			zap.String("patient_id", event.PatientID),
			zap.Error(err))
		return
	}

	p.logger.Info("DetectedIssue written to FHIR Store",
		zap.String("patient_id", event.PatientID),
		zap.Float64("threshold", payload.ThresholdCrossed))
}

// HandleStratumChange writes a FHIR Condition when CKD status is confirmed.
func (p *Publisher) HandleStratumChange(event models.Event) {
	// Look up the patient profile to check CKD status
	var profile models.PatientProfile
	if err := p.db.Where("patient_id = ?", event.PatientID).First(&profile).Error; err != nil {
		p.logger.Warn("Cannot write-back CKD Condition: patient not found",
			zap.String("patient_id", event.PatientID))
		return
	}

	if profile.CKDStatus != "CONFIRMED" && profile.CKDStatus != "SUSPECTED" {
		return // No CKD to report
	}

	if profile.FHIRPatientID == "" {
		p.logger.Warn("Cannot write-back CKD Condition: no FHIR Patient ID",
			zap.String("patient_id", event.PatientID))
		return
	}

	condition := CKDStatusToFHIRCondition(&profile)

	if err := p.client.UpsertCondition(condition); err != nil {
		p.logger.Error("Failed to write CKD Condition to FHIR Store",
			zap.String("patient_id", event.PatientID),
			zap.Error(err))
		return
	}

	p.logger.Info("CKD Condition written to FHIR Store",
		zap.String("patient_id", event.PatientID),
		zap.String("ckd_status", profile.CKDStatus),
		zap.String("ckd_stage", profile.CKDStage))
}

// resolveFHIRPatientID looks up the FHIR Patient ID from the KB-20 patient profile.
func (p *Publisher) resolveFHIRPatientID(patientID string) string {
	var profile models.PatientProfile
	if err := p.db.Select("fhir_patient_id").
		Where("patient_id = ?", patientID).
		First(&profile).Error; err != nil {
		return ""
	}
	return profile.FHIRPatientID
}
