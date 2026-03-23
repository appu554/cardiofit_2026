package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ──────────────────────────────────────────────────────────────────────
// Lab threshold response types — typed structs, not map[string]interface{}.
// Consumers: Flink stream enrichment, V-MCU Channel B safety, ingestion
// service plausibility checks.
// ──────────────────────────────────────────────────────────────────────

// CreatinineThresholds defines AKI staging and worsening-slope thresholds.
type CreatinineThresholds struct {
	PlausibleRange      [2]float64 `json:"plausible_range"`
	NormalRange         [2]float64 `json:"normal_range"`
	AKIStage1Delta48h   float64    `json:"aki_stage1_delta_48h"`
	AKIStage1PctIncrease float64   `json:"aki_stage1_pct_increase"`
	AKIStage2Multiplier float64    `json:"aki_stage2_multiplier"`
	AKIStage3Multiplier float64    `json:"aki_stage3_multiplier"`
	AKIStage3Absolute   float64    `json:"aki_stage3_absolute"`
	WorseningSlope      float64    `json:"worsening_slope"`
}

// PotassiumThresholds separates Flink alert thresholds from V-MCU halt thresholds.
// alert_high (5.5) = Flink notification; halt_high (6.0) = V-MCU Channel B dose HALT.
type PotassiumThresholds struct {
	PlausibleRange [2]float64 `json:"plausible_range"`
	NormalRange    [2]float64 `json:"normal_range"`
	AlertLow       float64   `json:"alert_low"`
	AlertHigh      float64   `json:"alert_high"`
	HaltLow        float64   `json:"halt_low"`
	HaltHigh       float64   `json:"halt_high"`
}

// GlucoseThresholds covers hypo/hyper severity ladder and CV threshold.
type GlucoseThresholds struct {
	PlausibleRange [2]float64 `json:"plausible_range"`
	NormalFasting  [2]float64 `json:"normal_fasting"`
	Hypo           float64    `json:"hypo"`
	SevereHypo     float64    `json:"severe_hypo"`
	SevereHyper    float64    `json:"severe_hyper"`
	CriticalHigh   float64    `json:"critical_high"`
	CVThreshold    float64    `json:"cv_threshold"`
}

// EGFRThresholds separates halt (V-MCU hard stop) from pause (V-MCU soft gate).
// halt (15) = absolute contraindication; pause (30) = dose-hold pending review.
type EGFRThresholds struct {
	PlausibleRange [2]float64 `json:"plausible_range"`
	Halt           float64    `json:"halt"`
	Pause          float64    `json:"pause"`
	CKDStage3a     float64    `json:"ckd_stage3a"`
	CKDStage3b     float64    `json:"ckd_stage3b"`
}

// HbA1cThresholds defines diagnostic cut-points.
type HbA1cThresholds struct {
	PlausibleRange [2]float64 `json:"plausible_range"`
	NormalHigh     float64    `json:"normal_high"`
	Prediabetic    float64    `json:"prediabetic"`
}

// LactateThresholds defines sepsis/tissue-hypoxia thresholds.
type LactateThresholds struct {
	NormalHigh float64 `json:"normal_high"`
	Critical   float64 `json:"critical"`
}

// TroponinThresholds defines cardiac injury markers.
type TroponinThresholds struct {
	NormalHigh float64 `json:"normal_high"`
	Critical   float64 `json:"critical"`
}

// WBCThresholds defines infection/immunosuppression markers.
type WBCThresholds struct {
	CriticalLow  float64 `json:"critical_low"`
	CriticalHigh float64 `json:"critical_high"`
}

// LabThresholdsResponse is the top-level response for GET /api/v1/thresholds/labs.
type LabThresholdsResponse struct {
	Creatinine CreatinineThresholds `json:"creatinine"`
	Potassium  PotassiumThresholds  `json:"potassium"`
	Glucose    GlucoseThresholds    `json:"glucose"`
	EGFR       EGFRThresholds       `json:"egfr"`
	HbA1c      HbA1cThresholds      `json:"hba1c"`
	Lactate    LactateThresholds    `json:"lactate"`
	Troponin   TroponinThresholds   `json:"troponin"`
	WBC        WBCThresholds        `json:"wbc"`
	Version    string               `json:"version"`
}

// labThresholds is the singleton response. Static clinical configuration —
// no database access required. Values sourced from spec Section 5.3.
var labThresholds = LabThresholdsResponse{
	Creatinine: CreatinineThresholds{
		PlausibleRange:       [2]float64{0.2, 20.0},
		NormalRange:          [2]float64{0.6, 1.2},
		AKIStage1Delta48h:    0.3,
		AKIStage1PctIncrease: 50,
		AKIStage2Multiplier:  2.0,
		AKIStage3Multiplier:  3.0,
		AKIStage3Absolute:    4.0,
		WorseningSlope:       0.1,
	},
	Potassium: PotassiumThresholds{
		PlausibleRange: [2]float64{1.5, 9.0},
		NormalRange:    [2]float64{3.5, 5.0},
		AlertLow:       3.0,
		AlertHigh:      5.5,
		HaltLow:        3.0,
		HaltHigh:       6.0,
	},
	Glucose: GlucoseThresholds{
		PlausibleRange: [2]float64{30, 600},
		NormalFasting:  [2]float64{70, 100},
		Hypo:           70,
		SevereHypo:     54,
		SevereHyper:    300,
		CriticalHigh:   400,
		CVThreshold:    36.0,
	},
	EGFR: EGFRThresholds{
		PlausibleRange: [2]float64{0, 200},
		Halt:           15,
		Pause:          30,
		CKDStage3a:     45,
		CKDStage3b:     30,
	},
	HbA1c: HbA1cThresholds{
		PlausibleRange: [2]float64{3.0, 18.0},
		NormalHigh:     5.7,
		Prediabetic:    6.5,
	},
	Lactate: LactateThresholds{
		NormalHigh: 2.0,
		Critical:   4.0,
	},
	Troponin: TroponinThresholds{
		NormalHigh: 0.04,
		Critical:   0.5,
	},
	WBC: WBCThresholds{
		CriticalLow:  4.0,
		CriticalHigh: 15.0,
	},
	Version: "2026-03-23T00:00:00Z",
}

// getLabThresholds returns the full lab threshold dataset for Flink and V-MCU consumers.
// GET /api/v1/thresholds/labs
func (s *Server) getLabThresholds(c *gin.Context) {
	c.JSON(http.StatusOK, labThresholds)
}
