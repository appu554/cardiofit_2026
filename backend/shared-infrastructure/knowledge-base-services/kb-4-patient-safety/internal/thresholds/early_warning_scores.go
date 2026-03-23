package thresholds

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// earlyWarningVersion is the canonical version tag for the early-warning
// scoring dataset.
const earlyWarningVersion = "2026-03-23T00:00:00Z"

// EWSThresholds defines the alert-escalation thresholds for an early-warning
// scoring system.  Critical and High are the aggregate score values that
// trigger the respective escalation tiers.  LowMediumSingle3 flags whether a
// single parameter scoring 3 alone triggers a medium escalation (NEWS2 only).
type EWSThresholds struct {
	Critical         int  `json:"critical"`
	High             int  `json:"high"`
	LowMediumSingle3 bool `json:"low_medium_single3,omitempty"`
}

// ScoreBand represents a single scoring band in an early-warning system.
// Min and Max are inclusive boundaries; Points is the score awarded when
// the observed value falls within [Min, Max].
type ScoreBand struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Points int     `json:"points"`
}

// NEWS2Scores holds the complete NEWS2 (National Early Warning Score 2)
// scoring parameters including both SpO2 scales.
type NEWS2Scores struct {
	RespiratoryRate []ScoreBand            `json:"respiratory_rate"`
	SpO2Scale1      []ScoreBand            `json:"spo2_scale1"`
	SpO2Scale2      []ScoreBand            `json:"spo2_scale2"`
	SystolicBP      []ScoreBand            `json:"systolic_bp"`
	HeartRate       []ScoreBand            `json:"heart_rate"`
	Temperature     []ScoreBand            `json:"temperature"`
	Consciousness   map[string]int `json:"consciousness"`
	SupplementalO2  map[string]int `json:"supplemental_o2"`
	Thresholds      EWSThresholds  `json:"thresholds"`
}

// MEWSScores holds the complete MEWS (Modified Early Warning Score)
// scoring parameters.
type MEWSScores struct {
	RespiratoryRate []ScoreBand    `json:"respiratory_rate"`
	HeartRate       []ScoreBand    `json:"heart_rate"`
	SystolicBP      []ScoreBand    `json:"systolic_bp"`
	Temperature     []ScoreBand    `json:"temperature"`
	Consciousness   map[string]int `json:"consciousness"`
	Thresholds      EWSThresholds  `json:"thresholds"`
}

// EarlyWarningScoresResponse is the top-level response for
// GET /v1/thresholds/early-warning-scores.
type EarlyWarningScoresResponse struct {
	NEWS2   NEWS2Scores `json:"news2"`
	MEWS    MEWSScores  `json:"mews"`
	Version string      `json:"version"`
}

// earlyWarningScores is the singleton response.  Values are sourced from
// the KB-4 clinical safety dataset (spec Section 5.3).
var earlyWarningScores = EarlyWarningScoresResponse{
	NEWS2: NEWS2Scores{
		RespiratoryRate: []ScoreBand{
			{Min: 0, Max: 8, Points: 3},
			{Min: 9, Max: 11, Points: 1},
			{Min: 12, Max: 20, Points: 0},
			{Min: 21, Max: 24, Points: 2},
			{Min: 25, Max: 999, Points: 3},
		},
		SpO2Scale1: []ScoreBand{
			{Min: 0, Max: 91, Points: 3},
			{Min: 92, Max: 93, Points: 2},
			{Min: 94, Max: 95, Points: 1},
			{Min: 96, Max: 100, Points: 0},
		},
		SpO2Scale2: []ScoreBand{
			{Min: 0, Max: 92, Points: 3},
			{Min: 93, Max: 94, Points: 2},
			{Min: 95, Max: 96, Points: 1},
			{Min: 97, Max: 100, Points: 3},
		},
		SystolicBP: []ScoreBand{
			{Min: 0, Max: 90, Points: 3},
			{Min: 91, Max: 100, Points: 2},
			{Min: 101, Max: 110, Points: 1},
			{Min: 111, Max: 219, Points: 0},
			{Min: 220, Max: 999, Points: 3},
		},
		HeartRate: []ScoreBand{
			{Min: 0, Max: 40, Points: 3},
			{Min: 41, Max: 50, Points: 1},
			{Min: 51, Max: 90, Points: 0},
			{Min: 91, Max: 110, Points: 1},
			{Min: 111, Max: 130, Points: 2},
			{Min: 131, Max: 999, Points: 3},
		},
		Temperature: []ScoreBand{
			{Min: 0, Max: 35.0, Points: 3},
			{Min: 35.1, Max: 36.0, Points: 1},
			{Min: 36.1, Max: 38.0, Points: 0},
			{Min: 38.1, Max: 39.0, Points: 1},
			{Min: 39.1, Max: 99, Points: 2},
		},
		Consciousness: map[string]int{
			"alert":        0,
			"voice":        3,
			"pain":         3,
			"unresponsive": 3,
		},
		SupplementalO2: map[string]int{
			"on_oxygen": 2,
		},
		Thresholds: EWSThresholds{
			Critical:         7,
			High:             5,
			LowMediumSingle3: true,
		},
	},
	MEWS: MEWSScores{
		RespiratoryRate: []ScoreBand{
			{Min: 0, Max: 8, Points: 2},
			{Min: 9, Max: 14, Points: 0},
			{Min: 15, Max: 20, Points: 1},
			{Min: 21, Max: 29, Points: 2},
			{Min: 30, Max: 999, Points: 3},
		},
		HeartRate: []ScoreBand{
			{Min: 0, Max: 40, Points: 2},
			{Min: 41, Max: 50, Points: 1},
			{Min: 51, Max: 100, Points: 0},
			{Min: 101, Max: 110, Points: 1},
			{Min: 111, Max: 129, Points: 2},
			{Min: 130, Max: 999, Points: 3},
		},
		SystolicBP: []ScoreBand{
			{Min: 0, Max: 70, Points: 3},
			{Min: 71, Max: 80, Points: 2},
			{Min: 81, Max: 100, Points: 1},
			{Min: 101, Max: 199, Points: 0},
			{Min: 200, Max: 999, Points: 2},
		},
		Temperature: []ScoreBand{
			{Min: 0, Max: 35.0, Points: 2},
			{Min: 35.1, Max: 38.4, Points: 0},
			{Min: 38.5, Max: 99, Points: 2},
		},
		Consciousness: map[string]int{
			"alert":        0,
			"voice":        1,
			"pain":         2,
			"unresponsive": 3,
		},
		Thresholds: EWSThresholds{
			Critical: 5,
			High:     3,
		},
	},
	Version: earlyWarningVersion,
}

// HandleGetEarlyWarningScores returns the canonical NEWS2 and MEWS
// scoring parameters.  The response is static and safe for aggressive
// HTTP caching by Flink's BroadcastState poller.
func HandleGetEarlyWarningScores(c *gin.Context) {
	c.JSON(http.StatusOK, earlyWarningScores)
}
