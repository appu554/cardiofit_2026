// Package integration provides HTTP clients for KB service integrations
// KB-9 Client tests
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// KB-9 CLIENT TESTS
// =============================================================================

func TestNewKB9Client(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	config := KB9Config{
		BaseURL: "http://localhost:8094",
		Timeout: 30 * time.Second,
		Enabled: true,
	}

	client := NewKB9Client(config, log)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8094", client.config.BaseURL)
	assert.True(t, client.config.Enabled)
}

func TestNewKB9Client_DefaultTimeout(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	config := KB9Config{
		BaseURL: "http://localhost:8094",
		Enabled: true,
	}

	client := NewKB9Client(config, log)

	assert.Equal(t, 30*time.Second, client.config.Timeout)
}

// =============================================================================
// CARE GAP TYPE TESTS
// =============================================================================

func TestMeasureTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		measure  MeasureType
		expected string
	}{
		{"CMS122 HbA1c", MeasureCMS122DiabetesHbA1c, "CMS122_DIABETES_HBA1C"},
		{"CMS165 BP Control", MeasureCMS165BPControl, "CMS165_BP_CONTROL"},
		{"CMS69 BMI Screening", MeasureCMS69BMIScreening, "CMS69_BMI_SCREENING"},
		{"CMS2 Depression", MeasureCMS2DepressionScreen, "CMS2_DEPRESSION_SCREENING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.measure))
		})
	}
}

func TestGapStatusConstants(t *testing.T) {
	assert.Equal(t, "OPEN", string(GapStatusOpen))
	assert.Equal(t, "CLOSED", string(GapStatusClosed))
	assert.Equal(t, "PENDING", string(GapStatusPending))
	assert.Equal(t, "NOT_APPLICABLE", string(GapStatusNotApplicable))
}

func TestGapPriorityConstants(t *testing.T) {
	assert.Equal(t, "CRITICAL", string(GapPriorityCritical))
	assert.Equal(t, "URGENT", string(GapPriorityUrgent))
	assert.Equal(t, "HIGH", string(GapPriorityHigh))
	assert.Equal(t, "MEDIUM", string(GapPriorityMedium))
	assert.Equal(t, "LOW", string(GapPriorityLow))
}

// =============================================================================
// LAB-TO-MEASURE MAPPING TESTS
// =============================================================================

func TestLabToMeasureMapping(t *testing.T) {
	tests := []struct {
		labCode     string
		expectMeasure MeasureType
		expectFound bool
	}{
		{"4548-4", MeasureCMS122DiabetesHbA1c, true},
		{"17856-6", MeasureCMS122DiabetesHbA1c, true},
		{"55454-3", MeasureCMS165BPControl, true},
		{"39156-5", MeasureCMS69BMIScreening, true},
		{"UNKNOWN-CODE", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.labCode, func(t *testing.T) {
			measure, found := LabToMeasureMapping[tt.labCode]
			assert.Equal(t, tt.expectFound, found)
			if found {
				assert.Equal(t, tt.expectMeasure, measure)
			}
		})
	}
}

func TestRecommendedTestingFrequency(t *testing.T) {
	tests := []struct {
		measure   MeasureType
		frequency int
	}{
		{MeasureCMS122DiabetesHbA1c, 90},
		{MeasureCMS165BPControl, 30},
		{MeasureCMS69BMIScreening, 365},
	}

	for _, tt := range tests {
		t.Run(string(tt.measure), func(t *testing.T) {
			freq, exists := RecommendedTestingFrequency[tt.measure]
			assert.True(t, exists)
			assert.Equal(t, tt.frequency, freq)
		})
	}
}

// =============================================================================
// CARE GAP IDENTIFICATION TESTS
// =============================================================================

func TestIdentifyLabCareGaps_OverdueHbA1c(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	patientID := "patient-123"
	// HbA1c test from 120 days ago (overdue by 30 days)
	labHistory := []LabHistoryEntry{
		{
			Code:        "4548-4",
			Name:        "HbA1c",
			Value:       7.2,
			Unit:        "%",
			CollectedAt: time.Now().AddDate(0, 0, -120),
		},
	}

	gaps, err := client.IdentifyLabCareGaps(context.Background(), patientID, labHistory)

	require.NoError(t, err)
	// Function checks ALL mapped lab codes, so we should find gaps for unmapped ones too
	assert.Greater(t, len(gaps), 0, "Should find at least one gap")

	// Find the specific HbA1c gap for 4548-4
	var hba1cGap *LabBasedCareGap
	for i := range gaps {
		if gaps[i].LabCode == "4548-4" {
			hba1cGap = &gaps[i]
			break
		}
	}

	require.NotNil(t, hba1cGap, "Should find gap for HbA1c 4548-4")
	assert.Equal(t, MeasureCMS122DiabetesHbA1c, hba1cGap.MeasureType)
	assert.Equal(t, 30, hba1cGap.DaysOverdue) // 120 - 90 = 30 days overdue
	assert.Equal(t, "KB-16", hba1cGap.SourceService)
	assert.NotNil(t, hba1cGap.CurrentValue)
	assert.Equal(t, 7.2, *hba1cGap.CurrentValue)
}

func TestIdentifyLabCareGaps_UpToDate(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	patientID := "patient-456"
	// Recent HbA1c test (within 90 days)
	labHistory := []LabHistoryEntry{
		{
			Code:        "4548-4",
			Name:        "HbA1c",
			Value:       6.5,
			Unit:        "%",
			CollectedAt: time.Now().AddDate(0, 0, -30),
		},
	}

	gaps, err := client.IdentifyLabCareGaps(context.Background(), patientID, labHistory)

	require.NoError(t, err)
	// Should not find gaps for lab codes that are up to date
	// Note: gaps may be found for OTHER measure types not in history
	for _, gap := range gaps {
		assert.NotEqual(t, "4548-4", gap.LabCode, "Should not have gap for recent HbA1c")
	}
}

func TestIdentifyLabCareGaps_NeverTested(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	patientID := "patient-789"
	labHistory := []LabHistoryEntry{} // No lab history

	gaps, err := client.IdentifyLabCareGaps(context.Background(), patientID, labHistory)

	require.NoError(t, err)
	// Should find gaps for all mapped lab codes since none have been tested
	assert.Greater(t, len(gaps), 0, "Should find care gaps when no labs on record")
}

func TestIdentifyLabCareGaps_MultipleMeasures(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	patientID := "patient-multi"
	labHistory := []LabHistoryEntry{
		{
			Code:        "4548-4",
			Name:        "HbA1c",
			Value:       7.2,
			CollectedAt: time.Now().AddDate(0, 0, -30), // Up to date
		},
		{
			Code:        "55454-3",
			Name:        "BP Systolic",
			Value:       145,
			CollectedAt: time.Now().AddDate(0, 0, -60), // 60 days ago, overdue (30 day freq)
		},
	}

	gaps, err := client.IdentifyLabCareGaps(context.Background(), patientID, labHistory)

	require.NoError(t, err)
	// Should find gap for BP (overdue) but not HbA1c (up to date)
	var bpGapFound bool
	for _, gap := range gaps {
		if gap.LabCode == "55454-3" {
			bpGapFound = true
			assert.Equal(t, 30, gap.DaysOverdue) // 60 - 30 = 30 days overdue
		}
		assert.NotEqual(t, "4548-4", gap.LabCode, "HbA1c should not be overdue")
	}
	assert.True(t, bpGapFound, "Should find BP gap")
}

// =============================================================================
// PRIORITY CALCULATION TESTS
// =============================================================================

func TestCalculateGapPriority(t *testing.T) {
	tests := []struct {
		daysOverdue   int
		frequency     int
		expectPriority GapPriority
	}{
		{180, 90, GapPriorityCritical},  // 2x overdue
		{135, 90, GapPriorityUrgent},    // 1.5x overdue
		{100, 90, GapPriorityHigh},      // 1.1x overdue
		{45, 90, GapPriorityMedium},     // 0.5x overdue
		{20, 90, GapPriorityLow},        // 0.2x overdue
	}

	for _, tt := range tests {
		t.Run(string(tt.expectPriority), func(t *testing.T) {
			priority := calculateGapPriority(tt.daysOverdue, tt.frequency)
			assert.Equal(t, tt.expectPriority, priority)
		})
	}
}

// =============================================================================
// HELPER FUNCTION TESTS
// =============================================================================

func TestGetLabName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"4548-4", "Hemoglobin A1c"},
		{"55454-3", "Blood Pressure Systolic"},
		{"39156-5", "Body Mass Index"},
		{"UNKNOWN", "UNKNOWN"}, // Returns code if not found
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			name := getLabName(tt.code)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestGenerateRecommendation(t *testing.T) {
	tests := []struct {
		measure     MeasureType
		daysOverdue int
		contains    string
	}{
		{MeasureCMS122DiabetesHbA1c, 30, "HbA1c test is 30 days overdue"},
		{MeasureCMS165BPControl, 45, "Blood pressure check is 45 days overdue"},
		{MeasureCMS69BMIScreening, 100, "BMI screening is 100 days overdue"},
	}

	for _, tt := range tests {
		t.Run(string(tt.measure), func(t *testing.T) {
			rec := generateRecommendation(tt.measure, tt.daysOverdue)
			assert.Contains(t, rec, tt.contains)
		})
	}
}

// =============================================================================
// CLIENT METHODS WITH MOCK SERVER TESTS
// =============================================================================

func TestGetPatientCareGaps_Disabled(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	report, err := client.GetPatientCareGaps(context.Background(), "patient-123", nil)

	require.NoError(t, err)
	assert.Nil(t, report, "Should return nil when disabled")
}

func TestGetPatientCareGaps_WithMockServer(t *testing.T) {
	// Create mock KB-9 server
	mockReport := CareGapReport{
		PatientID:  "patient-123",
		ReportDate: time.Now(),
		OpenGaps: []CareGap{
			{
				ID:          "gap-1",
				MeasureType: MeasureCMS122DiabetesHbA1c,
				MeasureName: "Diabetes HbA1c Control",
				Status:      GapStatusOpen,
				Priority:    GapPriorityHigh,
			},
		},
		Summary: CareGapSummary{
			TotalOpenGaps:    1,
			HighPriorityGaps: 1,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/care-gaps", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockReport)
	}))
	defer server.Close()

	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Enabled: true,
	}, log)

	report, err := client.GetPatientCareGaps(context.Background(), "patient-123", []MeasureType{
		MeasureCMS122DiabetesHbA1c,
	})

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "patient-123", report.PatientID)
	assert.Len(t, report.OpenGaps, 1)
	assert.Equal(t, GapPriorityHigh, report.OpenGaps[0].Priority)
}

func TestReportLabBasedCareGap_Disabled(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	gap := &LabBasedCareGap{
		PatientID:   "patient-123",
		MeasureType: MeasureCMS122DiabetesHbA1c,
		LabCode:     "4548-4",
		DaysOverdue: 30,
	}

	err := client.ReportLabBasedCareGap(context.Background(), gap)

	require.NoError(t, err, "Should succeed when disabled (logs locally)")
}

func TestReportLabBasedCareGap_WithMockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/care-gaps/report", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "created"})
	}))
	defer server.Close()

	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Enabled: true,
	}, log)

	gap := &LabBasedCareGap{
		PatientID:      "patient-123",
		MeasureType:    MeasureCMS122DiabetesHbA1c,
		LabCode:        "4548-4",
		LabName:        "HbA1c",
		DaysOverdue:    30,
		Priority:       GapPriorityHigh,
		Recommendation: "Order HbA1c test",
		SourceService:  "KB-16",
		IdentifiedAt:   time.Now(),
	}

	err := client.ReportLabBasedCareGap(context.Background(), gap)

	require.NoError(t, err)
}

// =============================================================================
// CARE GAP UPDATE RESULT TESTS
// =============================================================================

func TestCheckLabCareGapStatus_NotMappedLab(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	client := NewKB9Client(KB9Config{Enabled: false}, log)

	result, err := client.CheckLabCareGapStatus(
		context.Background(),
		"patient-123",
		"UNKNOWN-LAB",
		100.0,
		time.Now(),
	)

	require.NoError(t, err)
	assert.Nil(t, result, "Should return nil for unmapped lab codes")
}

// =============================================================================
// SERIALIZATION TESTS
// =============================================================================

func TestCareGapSerialization(t *testing.T) {
	gap := CareGap{
		ID:          "gap-123",
		MeasureType: MeasureCMS122DiabetesHbA1c,
		MeasureName: "Diabetes HbA1c Control",
		Status:      GapStatusOpen,
		Priority:    GapPriorityCritical,
		Reason:      "HbA1c not measured in 6 months",
		Recommendation: "Order HbA1c test",
	}

	data, err := json.Marshal(gap)
	require.NoError(t, err)

	var decoded CareGap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, gap.ID, decoded.ID)
	assert.Equal(t, gap.MeasureType, decoded.MeasureType)
	assert.Equal(t, gap.Status, decoded.Status)
	assert.Equal(t, gap.Priority, decoded.Priority)
}

func TestLabBasedCareGapSerialization(t *testing.T) {
	now := time.Now().UTC()
	value := 7.5
	lastTest := now.AddDate(0, 0, -120)

	gap := LabBasedCareGap{
		PatientID:      "patient-xyz",
		MeasureType:    MeasureCMS122DiabetesHbA1c,
		LabCode:        "4548-4",
		LabName:        "Hemoglobin A1c",
		CurrentValue:   &value,
		LastTestDate:   &lastTest,
		DaysOverdue:    30,
		Priority:       GapPriorityHigh,
		Recommendation: "Order HbA1c",
		SourceService:  "KB-16",
		IdentifiedAt:   now,
	}

	data, err := json.Marshal(gap)
	require.NoError(t, err)

	var decoded LabBasedCareGap
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, gap.PatientID, decoded.PatientID)
	assert.Equal(t, gap.MeasureType, decoded.MeasureType)
	assert.Equal(t, gap.LabCode, decoded.LabCode)
	assert.Equal(t, gap.DaysOverdue, decoded.DaysOverdue)
	assert.NotNil(t, decoded.CurrentValue)
	assert.Equal(t, *gap.CurrentValue, *decoded.CurrentValue)
}
