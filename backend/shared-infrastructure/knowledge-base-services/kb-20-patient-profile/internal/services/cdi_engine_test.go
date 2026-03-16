package services

import (
	"fmt"
	"testing"

	"kb-patient-profile/internal/models"
)

func TestComputeGlycaemicScore(t *testing.T) {
	tests := []struct {
		name      string
		fbg       *models.FBGTracking
		hba1c     float64
		hba1cPrev float64
		want      int
	}{
		{"all good", &models.FBGTracking{Trend: "STABLE", CV30d: 20}, 6.5, 6.5, 0},
		{"worsening FBG", &models.FBGTracking{Trend: "WORSENING", CV30d: 20}, 6.5, 6.5, 2},
		{"high CV only", &models.FBGTracking{Trend: "STABLE", CV30d: 40}, 6.5, 6.5, 2},
		{"worsening + high CV", &models.FBGTracking{Trend: "WORSENING", CV30d: 40}, 6.5, 6.5, 4},
		{"worsening + high CV + HbA1c rise", &models.FBGTracking{Trend: "WORSENING", CV30d: 40}, 9.5, 7.0, 6},
		{"nil FBG", nil, 9.5, 7.0, 2},
		{"no previous HbA1c", &models.FBGTracking{Trend: "WORSENING", CV30d: 40}, 9.5, 0, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeGlycaemicScore(tt.fbg, tt.hba1c, tt.hba1cPrev)
			if got != tt.want {
				t.Errorf("computeGlycaemicScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestComputeRenalScore(t *testing.T) {
	tests := []struct {
		name      string
		egfrSlope float64
		acrTrend  string
		acrCat    string
		creatRise float64
		want      int
	}{
		{"stable", -1.0, "STABLE", "A1", 5.0, 0},
		{"declining eGFR", -6.0, "STABLE", "A1", 5.0, 2},
		{"worsening ACR", -1.0, "WORSENING", "A2", 5.0, 2},
		{"A3 category alone", -1.0, "STABLE", "A3", 5.0, 2},
		{"creatinine rise only", -1.0, "STABLE", "A1", 25.0, 2},
		{"rapid decline + A3 ACR", -8.0, "WORSENING", "A3", 25.0, 6},
		{"all zero", 0, "STABLE", "A1", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeRenalScore(tt.egfrSlope, tt.acrTrend, tt.acrCat, tt.creatRise)
			if got != tt.want {
				t.Errorf("computeRenalScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCDIRiskLevel(t *testing.T) {
	tests := []struct {
		score int
		want  string
	}{
		{0, "LOW"}, {6, "LOW"}, {7, "MODERATE"}, {12, "MODERATE"},
		{13, "HIGH"}, {16, "HIGH"}, {17, "CRITICAL"}, {20, "CRITICAL"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%d", tt.score), func(t *testing.T) {
			got := cdiRiskLevel(tt.score)
			if got != tt.want {
				t.Errorf("cdiRiskLevel(%d) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}
