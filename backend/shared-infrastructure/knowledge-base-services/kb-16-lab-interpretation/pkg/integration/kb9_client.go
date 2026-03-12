// Package integration provides HTTP clients for KB service integrations
// KB-9 Client for care gap integration with quality measures
package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// KB-9 CARE GAPS CLIENT
// =============================================================================

// KB9Config holds configuration for KB-9 client
type KB9Config struct {
	BaseURL string
	Timeout time.Duration
	Enabled bool
}

// KB9Client provides integration with KB-9 Care Gaps Service
type KB9Client struct {
	config     KB9Config
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB9Client creates a new KB-9 client
func NewKB9Client(config KB9Config, log *logrus.Entry) *KB9Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &KB9Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		log: log.WithField("component", "kb9_client"),
	}
}

// =============================================================================
// CARE GAP TYPES (matching KB-9 models)
// =============================================================================

// MeasureType represents quality measure types
type MeasureType string

const (
	MeasureCMS122DiabetesHbA1c   MeasureType = "CMS122_DIABETES_HBA1C"
	MeasureCMS165BPControl       MeasureType = "CMS165_BP_CONTROL"
	MeasureCMS69BMIScreening     MeasureType = "CMS69_BMI_SCREENING"
	MeasureCMS2DepressionScreen  MeasureType = "CMS2_DEPRESSION_SCREENING"
	MeasureCMS138TobaccoScreen   MeasureType = "CMS138_TOBACCO_SCREENING"
	MeasureCMS130ColorectalScreen MeasureType = "CMS130_COLORECTAL_SCREENING"
)

// GapStatus represents the status of a care gap
type GapStatus string

const (
	GapStatusOpen          GapStatus = "OPEN"
	GapStatusClosed        GapStatus = "CLOSED"
	GapStatusPending       GapStatus = "PENDING"
	GapStatusNotApplicable GapStatus = "NOT_APPLICABLE"
)

// GapPriority represents the priority of a care gap
type GapPriority string

const (
	GapPriorityCritical GapPriority = "CRITICAL"
	GapPriorityUrgent   GapPriority = "URGENT"
	GapPriorityHigh     GapPriority = "HIGH"
	GapPriorityMedium   GapPriority = "MEDIUM"
	GapPriorityLow      GapPriority = "LOW"
)

// Period represents a time period
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CareGap represents an individual care gap from KB-9
type CareGap struct {
	ID              string      `json:"id"`
	MeasureType     MeasureType `json:"measureType"`
	MeasureName     string      `json:"measureName"`
	Status          GapStatus   `json:"status"`
	Priority        GapPriority `json:"priority"`
	Reason          string      `json:"reason"`
	Recommendation  string      `json:"recommendation"`
	IdentifiedDate  time.Time   `json:"identifiedDate"`
	DueDate         *time.Time  `json:"dueDate,omitempty"`
	DaysUntilDue    *int        `json:"daysUntilDue,omitempty"`
	DaysOverdue     *int        `json:"daysOverdue,omitempty"`
	LastLabDate     *time.Time  `json:"lastLabDate,omitempty"`
	LastLabValue    *float64    `json:"lastLabValue,omitempty"`
	TargetLabCode   string      `json:"targetLabCode,omitempty"`
}

// CareGapSummary provides summary statistics for care gaps
type CareGapSummary struct {
	TotalOpenGaps    int      `json:"totalOpenGaps"`
	UrgentGaps       int      `json:"urgentGaps"`
	HighPriorityGaps int      `json:"highPriorityGaps"`
	QualityScore     *float64 `json:"qualityScore,omitempty"`
}

// CareGapReport is the complete care gap report for a patient
type CareGapReport struct {
	PatientID         string          `json:"patientId"`
	ReportDate        time.Time       `json:"reportDate"`
	MeasurementPeriod Period          `json:"measurementPeriod"`
	OpenGaps          []CareGap       `json:"openGaps"`
	ClosedGaps        []CareGap       `json:"closedGaps,omitempty"`
	UpcomingDue       []CareGap       `json:"upcomingDue,omitempty"`
	Summary           CareGapSummary  `json:"summary"`
}

// =============================================================================
// LAB-BASED CARE GAP DETECTION
// =============================================================================

// LabBasedCareGap represents a care gap identified from lab interpretation
type LabBasedCareGap struct {
	PatientID       string      `json:"patientId"`
	MeasureType     MeasureType `json:"measureType"`
	LabCode         string      `json:"labCode"`
	LabName         string      `json:"labName"`
	CurrentValue    *float64    `json:"currentValue,omitempty"`
	LastTestDate    *time.Time  `json:"lastTestDate,omitempty"`
	DaysOverdue     int         `json:"daysOverdue"`
	Priority        GapPriority `json:"priority"`
	Recommendation  string      `json:"recommendation"`
	SourceService   string      `json:"sourceService"` // KB-16
	IdentifiedAt    time.Time   `json:"identifiedAt"`
}

// LabToMeasureMapping maps LOINC codes to quality measures
var LabToMeasureMapping = map[string]MeasureType{
	"4548-4":  MeasureCMS122DiabetesHbA1c, // HbA1c
	"17856-6": MeasureCMS122DiabetesHbA1c, // HbA1c (alternate)
	"59261-8": MeasureCMS122DiabetesHbA1c, // HbA1c (IFCC)
	"55454-3": MeasureCMS165BPControl,     // BP systolic
	"55284-4": MeasureCMS165BPControl,     // BP diastolic
	"39156-5": MeasureCMS69BMIScreening,   // BMI
	"39156-6": MeasureCMS69BMIScreening,   // BMI percentile
}

// RecommendedTestingFrequency defines recommended testing intervals in days
var RecommendedTestingFrequency = map[MeasureType]int{
	MeasureCMS122DiabetesHbA1c: 90,   // Every 3 months for diabetics
	MeasureCMS165BPControl:     30,   // Monthly for hypertensives
	MeasureCMS69BMIScreening:   365,  // Annually
}

// =============================================================================
// CLIENT METHODS
// =============================================================================

// GetPatientCareGaps retrieves care gaps for a patient
func (c *KB9Client) GetPatientCareGaps(ctx context.Context, patientID string, measures []MeasureType) (*CareGapReport, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-9 integration disabled")
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/api/v1/care-gaps", c.config.BaseURL)

	reqBody := map[string]interface{}{
		"patientId":         patientID,
		"includeClosedGaps": false,
		"includeEvidence":   true,
	}
	if len(measures) > 0 {
		reqBody["measures"] = measures
	}

	var report CareGapReport
	if err := c.doRequest(ctx, "POST", endpoint, reqBody, &report); err != nil {
		return nil, fmt.Errorf("failed to get care gaps: %w", err)
	}

	return &report, nil
}

// GetLabBasedCareGaps retrieves care gaps that are related to lab testing
func (c *KB9Client) GetLabBasedCareGaps(ctx context.Context, patientID string) ([]CareGap, error) {
	report, err := c.GetPatientCareGaps(ctx, patientID, []MeasureType{
		MeasureCMS122DiabetesHbA1c,
		MeasureCMS165BPControl,
		MeasureCMS69BMIScreening,
	})
	if err != nil {
		return nil, err
	}
	if report == nil {
		return nil, nil
	}

	// Filter to lab-based gaps only
	labGaps := make([]CareGap, 0)
	for _, gap := range report.OpenGaps {
		if gap.TargetLabCode != "" {
			labGaps = append(labGaps, gap)
		}
	}

	return labGaps, nil
}

// ReportLabBasedCareGap reports a care gap identified by KB-16 to KB-9
func (c *KB9Client) ReportLabBasedCareGap(ctx context.Context, gap *LabBasedCareGap) error {
	if !c.config.Enabled {
		c.log.Debug("KB-9 integration disabled, logging gap locally")
		c.logCareGap(gap)
		return nil
	}

	endpoint := fmt.Sprintf("%s/api/v1/care-gaps/report", c.config.BaseURL)

	var response map[string]interface{}
	if err := c.doRequest(ctx, "POST", endpoint, gap, &response); err != nil {
		// Log locally if KB-9 is unavailable
		c.log.WithError(err).Warn("Failed to report care gap to KB-9, logging locally")
		c.logCareGap(gap)
		return err
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":   gap.PatientID,
		"measure_type": gap.MeasureType,
		"lab_code":     gap.LabCode,
		"priority":     gap.Priority,
	}).Info("Care gap reported to KB-9")

	return nil
}

// CloseLabCareGap marks a lab-based care gap as closed
func (c *KB9Client) CloseLabCareGap(ctx context.Context, patientID, gapID, labCode string, labValue float64) error {
	if !c.config.Enabled {
		return nil
	}

	endpoint := fmt.Sprintf("%s/api/v1/care-gaps/%s/addressed", c.config.BaseURL, gapID)

	reqBody := map[string]interface{}{
		"patientId":    patientID,
		"intervention": "LAB_ORDER",
		"notes":        fmt.Sprintf("Lab completed: %s = %.2f", labCode, labValue),
	}

	var response map[string]interface{}
	if err := c.doRequest(ctx, "POST", endpoint, reqBody, &response); err != nil {
		return fmt.Errorf("failed to close care gap: %w", err)
	}

	return nil
}

// =============================================================================
// LAB INTERPRETATION INTEGRATION
// =============================================================================

// CheckLabCareGapStatus checks if a lab result addresses an open care gap
func (c *KB9Client) CheckLabCareGapStatus(ctx context.Context, patientID, labCode string, value float64, collectedAt time.Time) (*CareGapUpdateResult, error) {
	// Map lab code to measure type
	measureType, exists := LabToMeasureMapping[labCode]
	if !exists {
		return nil, nil // Not a care-gap-related lab
	}

	result := &CareGapUpdateResult{
		LabCode:     labCode,
		MeasureType: measureType,
		Value:       value,
		CollectedAt: collectedAt,
	}

	// Get patient's care gaps for this measure
	gaps, err := c.GetLabBasedCareGaps(ctx, patientID)
	if err != nil {
		return nil, err
	}

	// Check if this lab closes any gaps
	for _, gap := range gaps {
		if gap.MeasureType == measureType && gap.Status == GapStatusOpen {
			result.ClosesGap = true
			result.ClosedGapID = gap.ID
			result.GapPriority = gap.Priority

			// Attempt to close the gap
			if err := c.CloseLabCareGap(ctx, patientID, gap.ID, labCode, value); err != nil {
				c.log.WithError(err).Warn("Failed to close care gap")
			}
			break
		}
	}

	return result, nil
}

// IdentifyLabCareGaps identifies potential care gaps from lab history
func (c *KB9Client) IdentifyLabCareGaps(ctx context.Context, patientID string, labHistory []LabHistoryEntry) ([]LabBasedCareGap, error) {
	gaps := make([]LabBasedCareGap, 0)
	now := time.Now()

	for labCode, measureType := range LabToMeasureMapping {
		frequency, exists := RecommendedTestingFrequency[measureType]
		if !exists {
			continue
		}

		// Find most recent test for this code
		var lastTest *LabHistoryEntry
		for i := range labHistory {
			if labHistory[i].Code == labCode {
				if lastTest == nil || labHistory[i].CollectedAt.After(lastTest.CollectedAt) {
					lastTest = &labHistory[i]
				}
			}
		}

		// Calculate days since last test
		var daysSinceTest int
		if lastTest != nil {
			daysSinceTest = int(now.Sub(lastTest.CollectedAt).Hours() / 24)
		} else {
			daysSinceTest = 365 * 10 // Never tested = very overdue
		}

		// Check if overdue
		if daysSinceTest > frequency {
			daysOverdue := daysSinceTest - frequency
			priority := calculateGapPriority(daysOverdue, frequency)

			gap := LabBasedCareGap{
				PatientID:      patientID,
				MeasureType:    measureType,
				LabCode:        labCode,
				LabName:        getLabName(labCode),
				DaysOverdue:    daysOverdue,
				Priority:       priority,
				Recommendation: generateRecommendation(measureType, daysOverdue),
				SourceService:  "KB-16",
				IdentifiedAt:   now,
			}

			if lastTest != nil {
				gap.CurrentValue = &lastTest.Value
				gap.LastTestDate = &lastTest.CollectedAt
			}

			gaps = append(gaps, gap)
		}
	}

	return gaps, nil
}

// =============================================================================
// HELPER TYPES AND FUNCTIONS
// =============================================================================

// CareGapUpdateResult contains the result of checking/updating care gaps
type CareGapUpdateResult struct {
	LabCode     string      `json:"labCode"`
	MeasureType MeasureType `json:"measureType"`
	Value       float64     `json:"value"`
	CollectedAt time.Time   `json:"collectedAt"`
	ClosesGap   bool        `json:"closesGap"`
	ClosedGapID string      `json:"closedGapId,omitempty"`
	GapPriority GapPriority `json:"gapPriority,omitempty"`
}

// LabHistoryEntry represents a lab test from patient history
type LabHistoryEntry struct {
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	CollectedAt time.Time `json:"collectedAt"`
}

func calculateGapPriority(daysOverdue, recommendedFrequency int) GapPriority {
	overdueFactor := float64(daysOverdue) / float64(recommendedFrequency)

	switch {
	case overdueFactor >= 2.0:
		return GapPriorityCritical
	case overdueFactor >= 1.5:
		return GapPriorityUrgent
	case overdueFactor >= 1.0:
		return GapPriorityHigh
	case overdueFactor >= 0.5:
		return GapPriorityMedium
	default:
		return GapPriorityLow
	}
}

func getLabName(code string) string {
	names := map[string]string{
		"4548-4":  "Hemoglobin A1c",
		"17856-6": "Hemoglobin A1c/Hemoglobin.total",
		"59261-8": "Hemoglobin A1c (IFCC)",
		"55454-3": "Blood Pressure Systolic",
		"55284-4": "Blood Pressure Diastolic",
		"39156-5": "Body Mass Index",
	}
	if name, exists := names[code]; exists {
		return name
	}
	return code
}

func generateRecommendation(measureType MeasureType, daysOverdue int) string {
	switch measureType {
	case MeasureCMS122DiabetesHbA1c:
		return fmt.Sprintf("HbA1c test is %d days overdue. Order HbA1c to assess glycemic control.", daysOverdue)
	case MeasureCMS165BPControl:
		return fmt.Sprintf("Blood pressure check is %d days overdue. Schedule BP measurement.", daysOverdue)
	case MeasureCMS69BMIScreening:
		return fmt.Sprintf("BMI screening is %d days overdue. Document height and weight.", daysOverdue)
	default:
		return fmt.Sprintf("Screening is %d days overdue. Schedule appropriate testing.", daysOverdue)
	}
}

func (c *KB9Client) logCareGap(gap *LabBasedCareGap) {
	c.log.WithFields(logrus.Fields{
		"patient_id":   gap.PatientID,
		"measure_type": gap.MeasureType,
		"lab_code":     gap.LabCode,
		"days_overdue": gap.DaysOverdue,
		"priority":     gap.Priority,
		"source":       gap.SourceService,
	}).Info("Lab-based care gap identified (logged locally)")
}

// =============================================================================
// HTTP HELPER
// =============================================================================

func (c *KB9Client) doRequest(ctx context.Context, method, url string, body interface{}, response interface{}) error {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if len(reqBody) > 0 {
		req.Header.Set("Content-Type", "application/json")
		req.Body = readCloser{reader: reqBody}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("KB-9 returned status %d", resp.StatusCode)
	}

	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// readCloser wraps bytes for http.Request.Body
type readCloser struct {
	reader []byte
	offset int
}

func (r readCloser) Read(p []byte) (n int, err error) {
	if r.offset >= len(r.reader) {
		return 0, nil
	}
	n = copy(p, r.reader[r.offset:])
	r.offset += n
	return n, nil
}

func (r readCloser) Close() error {
	return nil
}
