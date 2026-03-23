// Package thresholds provides static clinical threshold endpoints for
// vital signs and early-warning scoring systems (NEWS2, MEWS).
// These are consumed by the Flink analytical pipeline via BroadcastState
// hot-swap as specified in the outbox-flink-kb-centralization design.
package thresholds

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// vitalThresholdsVersion is the canonical version tag for the threshold
// dataset.  Consumers (Flink BroadcastState) use this to detect changes.
const vitalThresholdsVersion = "2026-03-23T00:00:00Z"

// HeartRateThresholds defines clinical heart-rate classification boundaries.
type HeartRateThresholds struct {
	BradycardiaSevere   int `json:"bradycardia_severe"`
	BradycardiaModerate int `json:"bradycardia_moderate"`
	NormalLow           int `json:"normal_low"`
	NormalHigh          int `json:"normal_high"`
	TachycardiaModerate int `json:"tachycardia_moderate"`
	TachycardiaSevere   int `json:"tachycardia_severe"`
}

// SystolicBPThresholds defines systolic blood-pressure classification.
type SystolicBPThresholds struct {
	HypotensionSevere int `json:"hypotension_severe"`
	Hypotension       int `json:"hypotension"`
	NormalHigh        int `json:"normal_high"`
	Stage2HTN         int `json:"stage2_htn"`
	Crisis            int `json:"crisis"`
}

// DiastolicBPThresholds defines diastolic blood-pressure classification.
type DiastolicBPThresholds struct {
	NormalHigh int `json:"normal_high"`
	Crisis     int `json:"crisis"`
}

// SpO2Thresholds defines oxygen saturation classification.
type SpO2Thresholds struct {
	Critical  int `json:"critical"`
	Low       int `json:"low"`
	NormalLow int `json:"normal_low"`
}

// RespiratoryRateThresholds defines respiratory-rate classification.
type RespiratoryRateThresholds struct {
	CriticalLow  int `json:"critical_low"`
	NormalLow    int `json:"normal_low"`
	NormalHigh   int `json:"normal_high"`
	CriticalHigh int `json:"critical_high"`
}

// TemperatureThresholds defines body-temperature classification.
type TemperatureThresholds struct {
	Hypothermia float64 `json:"hypothermia"`
	NormalLow   float64 `json:"normal_low"`
	NormalHigh  float64 `json:"normal_high"`
	HighFever   float64 `json:"high_fever"`
}

// VitalThresholdsResponse is the top-level response for GET /v1/thresholds/vitals.
type VitalThresholdsResponse struct {
	HeartRate       HeartRateThresholds       `json:"heart_rate"`
	SystolicBP      SystolicBPThresholds      `json:"systolic_bp"`
	DiastolicBP     DiastolicBPThresholds     `json:"diastolic_bp"`
	SpO2            SpO2Thresholds            `json:"spo2"`
	RespiratoryRate RespiratoryRateThresholds `json:"respiratory_rate"`
	Temperature     TemperatureThresholds     `json:"temperature"`
	Version         string                    `json:"version"`
}

// vitalThresholds is the singleton response returned by the endpoint.
// Values are sourced from the KB-4 clinical safety dataset (spec Section 5.3).
var vitalThresholds = VitalThresholdsResponse{
	HeartRate: HeartRateThresholds{
		BradycardiaSevere:   40,
		BradycardiaModerate: 50,
		NormalLow:           60,
		NormalHigh:          100,
		TachycardiaModerate: 110,
		TachycardiaSevere:   120,
	},
	// NormalHigh and Stage2HTN are intentionally both 140, matching the KB-4
	// spec (Section 5.3).  Stage 1 HTN (130-139 mmHg) is not separately
	// tracked here because this endpoint serves Flink alerting thresholds, not
	// full clinical BP staging.
	SystolicBP: SystolicBPThresholds{
		HypotensionSevere: 70,
		Hypotension:       90,
		NormalHigh:        140,
		Stage2HTN:         140,
		Crisis:            180,
	},
	DiastolicBP: DiastolicBPThresholds{
		NormalHigh: 90,
		Crisis:     120,
	},
	SpO2: SpO2Thresholds{
		Critical:  90,
		Low:       92,
		NormalLow: 95,
	},
	RespiratoryRate: RespiratoryRateThresholds{
		CriticalLow:  8,
		NormalLow:    12,
		NormalHigh:   20,
		CriticalHigh: 30,
	},
	Temperature: TemperatureThresholds{
		Hypothermia: 35.0,
		NormalLow:   36.1,
		NormalHigh:  37.8,
		HighFever:   39.5,
	},
	Version: vitalThresholdsVersion,
}

// HandleGetVitalThresholds returns the canonical vital-sign threshold
// dataset.  The response is static and safe for aggressive HTTP caching
// by Flink's BroadcastState poller.
func HandleGetVitalThresholds(c *gin.Context) {
	c.JSON(http.StatusOK, vitalThresholds)
}
