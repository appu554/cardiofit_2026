package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------------------------------------------------------------------------
// CGMStatusResponse — CGM availability snapshot for KB-23 card generation
// ---------------------------------------------------------------------------

// CGMStatusResponse aggregates CGM-relevant data for decision-card context.
type CGMStatusResponse struct {
	PatientID        string     `json:"patient_id"`
	HasCGM           bool       `json:"has_cgm"`
	DeviceType       string     `json:"device_type,omitempty"`
	LatestReportDate *time.Time `json:"latest_report_date,omitempty"`
	DataFreshDays    int        `json:"data_fresh_days"`
	LatestTIR        *float64   `json:"latest_tir,omitempty"`
	LatestGRIZone    string     `json:"latest_gri_zone,omitempty"`
	SufficientData   bool       `json:"sufficient_data"`
}

// classifyGRIZone maps a GRI score to a risk zone label.
// GRI (Glycaemia Risk Index): A < 20, B 20–40, C 40–60, D 60–80, E ≥ 80.
func classifyGRIZone(gri float64) string {
	switch {
	case gri < 20:
		return "A"
	case gri < 40:
		return "B"
	case gri < 60:
		return "C"
	case gri < 80:
		return "D"
	default:
		return "E"
	}
}

// ---------------------------------------------------------------------------
// getCGMStatus — GET /:id/cgm-status handler
// ---------------------------------------------------------------------------

// getCGMStatus returns CGM availability, device type, data freshness, and
// latest TIR/GRI zone for card generation context.
func (s *Server) getCGMStatus(c *gin.Context) {
	patientID := c.Param("id")

	// 1. Fetch patient profile to determine data tier.
	var profile struct {
		DataTier   string  `gorm:"column:data_tier"`
		HbA1c      *float64 `gorm:"column:hba1c"`
	}
	if err := s.db.DB.Table("patient_profiles").
		Select("data_tier, hba1c").
		Where("patient_id = ? AND active = true", patientID).
		First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
		return
	}

	// 2. Determine CGM availability from data tier.
	hasCGM := strings.HasPrefix(profile.DataTier, "TIER_1") ||
		strings.HasPrefix(profile.DataTier, "TIER_2")

	resp := CGMStatusResponse{
		PatientID:      patientID,
		HasCGM:         hasCGM,
		SufficientData: false,
	}

	if !hasCGM {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
		return
	}

	// 3. Derive device type heuristic from data tier.
	switch {
	case strings.Contains(profile.DataTier, "CGM"):
		resp.DeviceType = "CGM_CONTINUOUS"
	case strings.Contains(profile.DataTier, "FLASH"):
		resp.DeviceType = "FLASH_GLUCOSE"
	default:
		resp.DeviceType = "CGM_UNKNOWN"
	}

	// 4. Query latest CGM report from lab tracker.
	var latestReport struct {
		MeasuredAt time.Time `gorm:"column:measured_at"`
		Value      float64   `gorm:"column:value"`
		LoincCode  string    `gorm:"column:loinc_code"`
	}
	if err := s.db.DB.Table("lab_trackers").
		Where("patient_id = ? AND loinc_code IN (?, ?)", patientID, "97507-8", "97506-0"). // TIR and GRI LOINC codes
		Order("measured_at DESC").
		First(&latestReport).Error; err == nil {
		resp.LatestReportDate = &latestReport.MeasuredAt
		daysSince := int(time.Since(latestReport.MeasuredAt).Hours() / 24)
		resp.DataFreshDays = daysSince
		resp.SufficientData = daysSince <= 14

		// Assign based on LOINC: 97507-8 = TIR, 97506-0 = GRI
		if latestReport.LoincCode == "97507-8" {
			v := latestReport.Value
			resp.LatestTIR = &v
		}
	}

	// 5. Query latest GRI for zone classification.
	var griReport struct {
		Value float64 `gorm:"column:value"`
	}
	if err := s.db.DB.Table("lab_trackers").
		Where("patient_id = ? AND loinc_code = ?", patientID, "97506-0").
		Order("measured_at DESC").
		First(&griReport).Error; err == nil {
		resp.LatestGRIZone = classifyGRIZone(griReport.Value)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}
