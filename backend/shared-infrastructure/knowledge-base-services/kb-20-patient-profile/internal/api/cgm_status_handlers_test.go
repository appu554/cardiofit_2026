package api

import (
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestCGMStatusResponse_Structure — verify field assignment
// ---------------------------------------------------------------------------

func TestCGMStatusResponse_Structure(t *testing.T) {
	now := time.Now()
	tir := 72.5

	resp := CGMStatusResponse{
		PatientID:        "CGM-TEST-001",
		HasCGM:           true,
		DeviceType:       "FREESTYLE_LIBRE_2",
		LatestReportDate: &now,
		DataFreshDays:    3,
		LatestTIR:        &tir,
		LatestGRIZone:    "B",
		SufficientData:   true,
	}

	if resp.PatientID != "CGM-TEST-001" {
		t.Errorf("PatientID = %s, want CGM-TEST-001", resp.PatientID)
	}
	if !resp.HasCGM {
		t.Error("HasCGM = false, want true")
	}
	if resp.DeviceType != "FREESTYLE_LIBRE_2" {
		t.Errorf("DeviceType = %s, want FREESTYLE_LIBRE_2", resp.DeviceType)
	}
	if resp.LatestReportDate == nil || !resp.LatestReportDate.Equal(now) {
		t.Error("LatestReportDate not set correctly")
	}
	if resp.DataFreshDays != 3 {
		t.Errorf("DataFreshDays = %d, want 3", resp.DataFreshDays)
	}
	if resp.LatestTIR == nil || *resp.LatestTIR != 72.5 {
		t.Error("LatestTIR not set correctly")
	}
	if resp.LatestGRIZone != "B" {
		t.Errorf("LatestGRIZone = %s, want B", resp.LatestGRIZone)
	}
	if !resp.SufficientData {
		t.Error("SufficientData = false, want true")
	}
}

// ---------------------------------------------------------------------------
// TestCGMStatusResponse_NoCGM — minimal response for non-CGM patient
// ---------------------------------------------------------------------------

func TestCGMStatusResponse_NoCGM(t *testing.T) {
	resp := CGMStatusResponse{
		PatientID:      "SMBG-001",
		HasCGM:         false,
		SufficientData: false,
	}

	if resp.HasCGM {
		t.Error("HasCGM = true, want false for SMBG patient")
	}
	if resp.DeviceType != "" {
		t.Errorf("DeviceType = %s, want empty for non-CGM", resp.DeviceType)
	}
	if resp.LatestTIR != nil {
		t.Error("LatestTIR should be nil for non-CGM patient")
	}
	if resp.SufficientData {
		t.Error("SufficientData should be false for non-CGM patient")
	}
}

// ---------------------------------------------------------------------------
// TestClassifyGRIZone — GRI zone classification
// ---------------------------------------------------------------------------

func TestClassifyGRIZone(t *testing.T) {
	tests := []struct {
		name     string
		gri      float64
		expected string
	}{
		{name: "Zone A — excellent", gri: 10, expected: "A"},
		{name: "Zone B — good", gri: 25, expected: "B"},
		{name: "Zone C — moderate", gri: 50, expected: "C"},
		{name: "Zone D — poor", gri: 70, expected: "D"},
		{name: "Zone E — very poor", gri: 90, expected: "E"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGRIZone(tc.gri)
			if got != tc.expected {
				t.Errorf("classifyGRIZone(%.0f) = %s, want %s", tc.gri, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestClassifyGRIZone_Boundaries — edge cases at zone boundaries
// ---------------------------------------------------------------------------

func TestClassifyGRIZone_Boundaries(t *testing.T) {
	tests := []struct {
		name     string
		gri      float64
		expected string
	}{
		{name: "exactly 0", gri: 0, expected: "A"},
		{name: "exactly 20", gri: 20, expected: "B"},
		{name: "exactly 40", gri: 40, expected: "C"},
		{name: "exactly 60", gri: 60, expected: "D"},
		{name: "exactly 80", gri: 80, expected: "E"},
		{name: "just below 20", gri: 19.99, expected: "A"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGRIZone(tc.gri)
			if got != tc.expected {
				t.Errorf("classifyGRIZone(%.2f) = %s, want %s", tc.gri, got, tc.expected)
			}
		})
	}
}
