package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

// renalSensitiveDrugClasses is the set of drug classes whose active
// prescription makes a patient "renal-active" for the purposes of the
// KB-23 anticipatory monthly batch. Keyed on the drug classes that the
// shared renal_dose_rules.yaml formulary actually tracks (metformin,
// SGLT2, RAAS, MRA, DOACs). Used by listRenalActivePatients below.
var renalSensitiveDrugClasses = []string{
	"METFORMIN",
	"SGLT2",
	"SGLT2_INHIBITOR",
	"ACE_INHIBITOR",
	"ARB",
	"ARNI",
	"MRA",
	"FINERENONE",
	"DOAC",
	"DIGOXIN",
}

// ---------------------------------------------------------------------------
// RenalStatusResponse — renal snapshot for KB-23 decision-card consumption
// ---------------------------------------------------------------------------

// MedSummary is a lightweight medication reference returned in renal status.
type MedSummary struct {
	DrugName  string `json:"drug_name"`
	DrugClass string `json:"drug_class"`
	DoseMg    string `json:"dose_mg"`
	IsActive  bool   `json:"is_active"`
}

// RenalStatusResponse aggregates renal-relevant data for KB-23.
type RenalStatusResponse struct {
	PatientID         string       `json:"patient_id"`
	EGFR              float64      `json:"egfr"`
	EGFRSlope         float64      `json:"egfr_slope"`
	EGFRMeasuredAt    time.Time    `json:"egfr_measured_at"`
	EGFRDataPoints    int          `json:"egfr_data_points"`
	Potassium         *float64     `json:"potassium,omitempty"`
	ACR               *float64     `json:"acr,omitempty"`
	CKDStage          string       `json:"ckd_stage"`
	IsRapidDecliner   bool         `json:"is_rapid_decliner"`
	ActiveMedications []MedSummary `json:"active_medications"`
}

// ---------------------------------------------------------------------------
// classifyCKDStage — KDIGO 2024 eGFR-based CKD staging
// ---------------------------------------------------------------------------

// classifyCKDStage maps an eGFR value to a KDIGO CKD stage label.
func classifyCKDStage(egfr float64) string {
	switch {
	case egfr >= 90:
		return "G1"
	case egfr >= 60:
		return "G2"
	case egfr >= 45:
		return "G3a"
	case egfr >= 30:
		return "G3b"
	case egfr >= 15:
		return "G4"
	default:
		return "G5"
	}
}

// ---------------------------------------------------------------------------
// getRenalStatus — GET /:id/renal-status handler
// ---------------------------------------------------------------------------

// getRenalStatus returns a renal snapshot for the given patient.
// It queries the latest eGFR trajectory, potassium, ACR, and active meds.
func (s *Server) getRenalStatus(c *gin.Context) {
	patientID := c.Param("id")

	// 1. Fetch patient profile for potassium, UACR, eGFR.
	var profile models.PatientProfile
	if err := s.db.DB.Where("patient_id = ? AND active = true", patientID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
		return
	}

	// 2. Get eGFR trajectory from lab service.
	trajectory, err := s.labService.GetEGFRTrajectory(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute eGFR trajectory: " + err.Error()})
		return
	}

	// 3. Derive renal fields from trajectory response.
	var latestEGFR float64
	var egfrMeasuredAt time.Time
	dataPoints := len(trajectory.Points)
	if dataPoints > 0 {
		last := trajectory.Points[dataPoints-1]
		latestEGFR = last.Value
		egfrMeasuredAt = last.MeasuredAt
	} else if profile.EGFR != nil {
		latestEGFR = *profile.EGFR
	}

	// AnnualChange from EGFRTrajectoryResponse is the slope (mL/min/year).
	var slope float64
	if trajectory.AnnualChange != nil {
		slope = *trajectory.AnnualChange
	}

	isRapidDecliner := slope <= -5.0

	// 4. Fetch active medications.
	var meds []models.MedicationState
	s.db.DB.Where("patient_id = ? AND is_active = true", patientID).Find(&meds)

	medSummaries := make([]MedSummary, 0, len(meds))
	for _, m := range meds {
		medSummaries = append(medSummaries, MedSummary{
			DrugName:  m.DrugName,
			DrugClass: m.DrugClass,
			DoseMg:    m.DoseMg.String(),
			IsActive:  m.IsActive,
		})
	}

	resp := RenalStatusResponse{
		PatientID:         patientID,
		EGFR:              latestEGFR,
		EGFRSlope:         slope,
		EGFRMeasuredAt:    egfrMeasuredAt,
		EGFRDataPoints:    dataPoints,
		Potassium:         profile.Potassium,
		ACR:               profile.UACR,
		CKDStage:          classifyCKDStage(latestEGFR),
		IsRapidDecliner:   isRapidDecliner,
		ActiveMedications: medSummaries,
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// ---------------------------------------------------------------------------
// listRenalActivePatients — GET /api/v1/patients/renal-active handler
// ---------------------------------------------------------------------------

// RenalActivePatientEntry is the minimal projection returned by the
// renal-active list endpoint. Consumers (KB-23 RenalAnticipatoryBatch)
// call back to GET /patient/:id/renal-status for the full snapshot of
// any patient they care about — keeping this list-endpoint payload
// small avoids scanning join tables for every patient on every run.
type RenalActivePatientEntry struct {
	PatientID string `json:"patient_id"`
}

// listRenalActivePatients returns all active patients who have at least
// one active medication in the renal-sensitive drug class set. Used by
// KB-23's monthly renal anticipatory batch to enumerate the population
// worth running FindApproachingThresholds + DetectStaleEGFR over.
// Phase 7 P7-C.
func (s *Server) listRenalActivePatients(c *gin.Context) {
	// SELECT DISTINCT patient_id FROM medication_states WHERE is_active = true
	//   AND drug_class IN (renalSensitiveDrugClasses)
	var patientIDs []string
	err := s.db.DB.
		Table("medication_states").
		Distinct("patient_id").
		Where("is_active = ? AND drug_class IN ?", true, renalSensitiveDrugClasses).
		Pluck("patient_id", &patientIDs).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query renal-active patients: " + err.Error()})
		return
	}

	entries := make([]RenalActivePatientEntry, 0, len(patientIDs))
	for _, pid := range patientIDs {
		entries = append(entries, RenalActivePatientEntry{PatientID: pid})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": entries})
}
