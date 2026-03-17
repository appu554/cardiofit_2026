package services

import (
	"go.uber.org/zap"

	"kb-patient-profile/internal/cache"
	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/metrics"
	"kb-patient-profile/internal/models"
)

// StratumEngine evaluates patient state to determine the active stratum
// and returns CKD substage visibility (F-03 RED).
type StratumEngine struct {
	db       *database.Database
	cache    *cache.Client
	logger   *zap.Logger
	metrics  *metrics.Collector
	egfr     *EGFREngine
	cmReg    *CMRegistry
	eventBus *EventBus
}

// NewStratumEngine creates the stratum activation engine.
func NewStratumEngine(
	db *database.Database,
	cacheClient *cache.Client,
	logger *zap.Logger,
	metricsCollector *metrics.Collector,
	cmReg *CMRegistry,
	eventBus *EventBus,
) *StratumEngine {
	return &StratumEngine{
		db:       db,
		cache:    cacheClient,
		logger:   logger,
		metrics:  metricsCollector,
		egfr:     NewEGFREngine(),
		cmReg:    cmReg,
		eventBus: eventBus,
	}
}

// GetStratum evaluates the patient's current state and returns the stratum response
// for a given HPI node, including ckd_substage and active modifiers.
func (se *StratumEngine) GetStratum(patientID string, nodeID string) (*models.StratumResponse, error) {
	se.metrics.StratumQueries.Inc()

	// Try cache
	cacheKey := cache.StratumPrefix + patientID + ":" + nodeID
	var cached models.StratumResponse
	if err := se.cache.Get(cacheKey, &cached); err == nil {
		return &cached, nil
	}

	// Get patient profile
	var profile models.PatientProfile
	if err := se.db.DB.Where("patient_id = ? AND active = true", patientID).First(&profile).Error; err != nil {
		return nil, err
	}

	// Get latest eGFR
	var latestEGFR models.LabEntry
	hasEGFR := false
	var egfrValue float64
	if err := se.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
		patientID, models.LabTypeEGFR, models.ValidationAccepted).
		Order("measured_at DESC").First(&latestEGFR).Error; err == nil {
		hasEGFR = true
		egfrValue, _ = latestEGFR.Value.Float64()
	}

	// Determine stratum label
	stratumLabel := se.determineStratum(profile, hasEGFR, egfrValue)

	// Determine CKD substage (F-03 RED)
	ckdSubstage := ""
	if hasEGFR && egfrValue < 60 {
		ckdSubstage = se.egfr.CKDStageFromEGFR(egfrValue)
	}

	// Get active medications
	var medications []models.MedicationState
	se.db.DB.Where("patient_id = ? AND is_active = true", patientID).Find(&medications)

	// Get active context modifiers for this node
	activeModifiers := se.cmReg.GetActiveModifiers(nodeID, medications, patientID)

	// Get safety overrides
	var safetyOverrides []models.SafetyOverride
	if hasEGFR {
		safetyOverrides = se.egfr.CheckMedicationAlerts(egfrValue, medications)
	}

	// Include eGFR value in response when available
	var egfrPtr *float64
	if hasEGFR {
		egfrPtr = &egfrValue
	}

	// Compute eGFR trajectory from historical lab data
	var egfrSlope float64
	var egfrTrajectoryClass string
	if hasEGFR {
		var egfrEntries []models.LabEntry
		se.db.DB.Where("patient_id = ? AND lab_type = ? AND validation_status = ?",
			patientID, models.LabTypeEGFR, models.ValidationAccepted).
			Order("measured_at ASC").Find(&egfrEntries)

		var points []models.EGFRTrajectoryPoint
		for _, entry := range egfrEntries {
			val, _ := entry.Value.Float64()
			points = append(points, models.EGFRTrajectoryPoint{
				Value:      val,
				MeasuredAt: entry.MeasuredAt,
				CKDStage:   se.egfr.CKDStageFromEGFR(val),
			})
		}

		trend, annualChange := se.egfr.ClassifyTrajectory(points)
		egfrTrajectoryClass = trend
		if annualChange != nil {
			egfrSlope = *annualChange
		}
	}

	response := &models.StratumResponse{
		PatientID:           patientID,
		NodeID:              nodeID,
		StratumLabel:        stratumLabel,
		EGFR:                egfrPtr,
		EGFRSlope:           egfrSlope,
		EGFRTrajectoryClass: egfrTrajectoryClass,
		CKDSubstage:         ckdSubstage,
		IsProvisional:       profile.CKDStatus == "SUSPECTED",
		ActiveModifiers:     activeModifiers,
		SafetyOverrides:     safetyOverrides,
	}

	// Emit STRATUM_CHANGE when stratum transitions (Gap #21)
	if cached.StratumLabel != "" && cached.StratumLabel != stratumLabel {
		se.eventBus.Publish(models.EventStratumChange, patientID, models.StratumChangePayload{
			OldStratum:     cached.StratumLabel,
			NewStratum:     stratumLabel,
			OldCKDSubstage: cached.CKDSubstage,
			NewCKDSubstage: ckdSubstage,
			Trigger:        "STRATUM_RECOMPUTATION",
		})
	}

	se.cache.Set(cacheKey, response, cache.DefaultStratumTTL)
	return response, nil
}

// determineStratum resolves the stratum label from patient state.
// G4: Added HF detection — DM+HTN+CKD+HF is the highest-acuity stratum,
// supported by ARIC CKD substudy and Wang JAMA 2005 evidence (A01-Q4 satisfied).
func (se *StratumEngine) determineStratum(profile models.PatientProfile, hasEGFR bool, egfr float64) string {
	hasDM := profile.DMType == "T1DM" || profile.DMType == "T2DM"
	hasHTN := false
	hasHF := false
	for _, c := range profile.Comorbidities {
		switch c {
		case "UNCONTROLLED_HTN", "HTN":
			hasHTN = true
		case "HF", "HEART_FAILURE", "HFrEF", "HFpEF", "HFmrEF":
			hasHF = true
		}
	}
	hasCKD := hasEGFR && egfr < 60

	switch {
	case hasDM && hasHTN && hasCKD && hasHF:
		return models.StratumDMHTNCKDHF
	case hasDM && hasHTN && hasCKD:
		return models.StratumDMHTNCKD
	case hasDM && hasHTN:
		return models.StratumDMHTN
	case hasDM:
		return models.StratumDMOnly
	case hasHTN:
		return models.StratumHTNOnly
	default:
		return "NONE"
	}
}
