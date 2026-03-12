package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// MedicationService handles medication CRUD with FDC decomposition (F-01 RED).
type MedicationService struct {
	db       *database.Database
	cache    *cache.Client
	logger   *zap.Logger
	eventBus *EventBus
}

// NewMedicationService creates a medication service.
func NewMedicationService(db *database.Database, cacheClient *cache.Client, logger *zap.Logger, eventBus *EventBus) *MedicationService {
	return &MedicationService{db: db, cache: cacheClient, logger: logger, eventBus: eventBus}
}

// Add stores a new medication. If fdc_components is non-empty, the medication
// is treated as a fixed-dose combination and CM activation will evaluate ALL
// component drug classes (F-01 RED).
func (s *MedicationService) Add(patientID string, req models.AddMedicationRequest) (*models.MedicationState, error) {
	startDate := time.Now()
	if req.StartDate != "" {
		parsed, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			startDate = parsed
		}
	}

	med := &models.MedicationState{
		ID:            uuid.New(),
		PatientID:     patientID,
		DrugName:      req.DrugName,
		DrugClass:     req.DrugClass,
		DoseMg:        decimal.NewFromFloat(req.DoseMg),
		Frequency:     req.Frequency,
		Route:         req.Route,
		PrescribedBy:  req.PrescribedBy,
		FDCComponents: req.FDCComponents,
		IsActive:      true,
		StartDate:     startDate,
	}

	if med.Route == "" {
		med.Route = "ORAL"
	}

	if err := s.db.DB.Create(med).Error; err != nil {
		return nil, fmt.Errorf("failed to add medication: %w", err)
	}

	// Publish MEDICATION_CHANGE event
	s.eventBus.Publish(models.EventMedicationChange, patientID, models.MedicationChangePayload{
		ChangeType: "ADD",
		DrugName:   med.DrugName,
		DrugClass:  med.DrugClass,
		NewDoseMg:  med.DoseMg.String(),
	})

	// Invalidate cache
	s.cache.Delete(cache.PatientProfilePrefix + patientID)

	s.logger.Info("Medication added",
		zap.String("patient_id", patientID),
		zap.String("drug", med.DrugName),
		zap.String("class", med.DrugClass),
		zap.Int("fdc_components", len(med.FDCComponents)))

	return med, nil
}

// Update modifies a medication's dose, frequency, or active status.
func (s *MedicationService) Update(patientID string, medID string, req models.UpdateMedicationRequest) error {
	parsedID, err := uuid.Parse(medID)
	if err != nil {
		return fmt.Errorf("invalid medication ID: %w", err)
	}

	var med models.MedicationState
	if err := s.db.DB.Where("id = ? AND patient_id = ?", parsedID, patientID).First(&med).Error; err != nil {
		return fmt.Errorf("medication not found: %w", err)
	}

	oldDose := med.DoseMg.String()
	updates := map[string]interface{}{}
	if req.DoseMg != nil {
		updates["dose_mg"] = decimal.NewFromFloat(*req.DoseMg)
	}
	if req.Frequency != nil {
		updates["frequency"] = *req.Frequency
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
		if !*req.IsActive {
			now := time.Now()
			updates["end_date"] = &now
		}
	}

	if err := s.db.DB.Model(&med).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update medication: %w", err)
	}

	changeType := "UPDATE"
	if req.IsActive != nil && !*req.IsActive {
		changeType = "DISCONTINUE"
	}

	newDose := oldDose
	if req.DoseMg != nil {
		newDose = decimal.NewFromFloat(*req.DoseMg).String()
	}

	s.eventBus.Publish(models.EventMedicationChange, patientID, models.MedicationChangePayload{
		ChangeType: changeType,
		DrugName:   med.DrugName,
		DrugClass:  med.DrugClass,
		OldDoseMg:  oldDose,
		NewDoseMg:  newDose,
	})

	s.cache.Delete(cache.PatientProfilePrefix + patientID)
	return nil
}

// GetActive retrieves all active medications for a patient.
func (s *MedicationService) GetActive(patientID string) ([]models.MedicationState, error) {
	var meds []models.MedicationState
	if err := s.db.DB.Where("patient_id = ? AND is_active = true", patientID).Find(&meds).Error; err != nil {
		return nil, fmt.Errorf("failed to get medications: %w", err)
	}
	return meds, nil
}

// GetRAASChangeRecency returns how recently the patient's ACEi/ARB therapy was
// initiated or titrated. V-MCU Channel B uses this via PG-14 to suppress B-03
// (AKI alarm) during the expected post-RAAS creatinine rise window.
//
// Logic:
//   - Query active ACEi/ARB medications
//   - Find the most recently modified one (UpdatedAt)
//   - If StartDate ≈ UpdatedAt (within 48h), classify as INITIATION; else TITRATION
//   - Return days since change + classification
func (s *MedicationService) GetRAASChangeRecency(patientID string) (*models.RAASChangeRecency, error) {
	raasClasses := []string{models.DrugClassACEInhibitor, models.DrugClassARB}

	var meds []models.MedicationState
	if err := s.db.DB.Where("patient_id = ? AND is_active = true AND drug_class IN ?",
		patientID, raasClasses).
		Order("updated_at DESC").Find(&meds).Error; err != nil {
		return nil, fmt.Errorf("failed to query RAAS medications: %w", err)
	}

	result := &models.RAASChangeRecency{
		InitiationOrTitration: "NONE",
	}

	if len(meds) == 0 {
		return result, nil
	}

	// Most recently changed RAAS medication
	latest := meds[0]
	result.LastACEiARBChangeAt = &latest.UpdatedAt
	result.DaysSinceChange = int(time.Since(latest.UpdatedAt).Hours() / 24)

	// Classify: if UpdatedAt is within 48h of StartDate → INITIATION (new drug)
	// Otherwise → TITRATION (dose change on existing drug)
	if latest.UpdatedAt.Sub(latest.StartDate).Hours() < 48 {
		result.InitiationOrTitration = "INITIATION"
	} else {
		result.InitiationOrTitration = "TITRATION"
	}

	return result, nil
}

// GetAllDrugClasses returns all effective drug classes for a patient (including FDC decomposition).
func (s *MedicationService) GetAllDrugClasses(patientID string) ([]string, error) {
	meds, err := s.GetActive(patientID)
	if err != nil {
		return nil, err
	}

	classSet := make(map[string]bool)
	for _, med := range meds {
		for _, dc := range med.EffectiveDrugClasses() {
			classSet[dc] = true
		}
	}

	var classes []string
	for dc := range classSet {
		classes = append(classes, dc)
	}
	return classes, nil
}
