// Package visualization provides chart data generation for lab results
package visualization

import (
	"context"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/baseline"
	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// WindowDays maps window names to days
var WindowDays = map[string]int{
	"7d":  7,
	"30d": 30,
	"90d": 90,
	"1yr": 365,
}

// Service generates visualization data for lab results
type Service struct {
	resultStore     *store.ResultStore
	baselineTracker *baseline.Tracker
	refDB           *reference.Database
	log             *logrus.Entry
}

// NewService creates a new visualization service
func NewService(
	resultStore *store.ResultStore,
	baselineTracker *baseline.Tracker,
	refDB *reference.Database,
	log *logrus.Entry,
) *Service {
	return &Service{
		resultStore:     resultStore,
		baselineTracker: baselineTracker,
		refDB:           refDB,
		log:             log.WithField("component", "visualization_service"),
	}
}

// GenerateChartData generates chart data for a specific test
func (s *Service) GenerateChartData(ctx context.Context, patientID, code, window string) (*types.ChartData, error) {
	// Get window days
	days, exists := WindowDays[window]
	if !exists {
		days = 30 // Default to 30 days
	}

	// Get results
	results, err := s.resultStore.GetByPatientAndCode(ctx, patientID, code, days)
	if err != nil {
		return nil, err
	}

	// Get test definition
	testDef := s.refDB.GetTest(code)
	testName := code
	unit := ""
	if testDef != nil {
		testName = testDef.Name
		unit = testDef.Unit
	}

	// Get reference range
	ranges := s.refDB.GetRanges(code, 0, "")

	// Convert to data points
	dataPoints := s.toChartDataPoints(results)

	// Sort by timestamp
	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	chartData := &types.ChartData{
		TestCode:   code,
		TestName:   testName,
		Unit:       unit,
		DataPoints: dataPoints,
		Window:     window,
		WindowDays: days,
	}

	// Add reference range
	if ranges != nil {
		chartData.ReferenceRange = &types.ChartReferenceRange{}
		if ranges.Low != nil {
			chartData.ReferenceRange.Low = *ranges.Low
		}
		if ranges.High != nil {
			chartData.ReferenceRange.High = *ranges.High
		}
		if ranges.CriticalLow != nil {
			chartData.ReferenceRange.CriticalLow = ranges.CriticalLow
		}
		if ranges.CriticalHigh != nil {
			chartData.ReferenceRange.CriticalHigh = ranges.CriticalHigh
		}
	}

	// Add baseline if available
	if s.baselineTracker != nil {
		bl, err := s.baselineTracker.GetBaseline(ctx, patientID, code)
		if err == nil && bl != nil {
			chartData.Baseline = &types.ChartBaseline{
				Mean:   bl.Mean,
				StdDev: bl.StdDev,
				Upper:  bl.Mean + 2*bl.StdDev,
				Lower:  bl.Mean - 2*bl.StdDev,
			}
		}
	}

	// Generate annotations for significant points
	chartData.Annotations = s.generateAnnotations(results, ranges)

	return chartData, nil
}

// GenerateSparkline generates minimal sparkline data
func (s *Service) GenerateSparkline(ctx context.Context, patientID, code string, days int) (*types.SparklineData, error) {
	if days <= 0 {
		days = 30
	}

	results, err := s.resultStore.GetByPatientAndCode(ctx, patientID, code, days)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	// Extract values
	values := make([]float64, 0, len(results))
	for _, r := range results {
		if r.ValueNumeric != nil {
			values = append(values, *r.ValueNumeric)
		}
	}

	if len(values) == 0 {
		return nil, nil
	}

	// Calculate min/max for normalization
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// Normalize to 0-100 scale
	normalizedValues := make([]float64, len(values))
	valueRange := max - min
	if valueRange == 0 {
		valueRange = 1 // Avoid division by zero
	}
	for i, v := range values {
		normalizedValues[i] = ((v - min) / valueRange) * 100
	}

	// Get reference range for trend indicator
	ranges := s.refDB.GetRanges(code, 0, "")
	var trend string
	if len(values) >= 2 {
		lastValue := values[len(values)-1]
		prevValue := values[len(values)-2]
		if lastValue > prevValue {
			trend = "up"
		} else if lastValue < prevValue {
			trend = "down"
		} else {
			trend = "stable"
		}
	}

	// Determine status
	status := "normal"
	if ranges != nil && len(values) > 0 {
		lastValue := values[len(values)-1]
		if ranges.Low != nil && lastValue < *ranges.Low {
			status = "low"
		} else if ranges.High != nil && lastValue > *ranges.High {
			status = "high"
		}
	}

	return &types.SparklineData{
		Code:             code,
		Values:           normalizedValues,
		Min:              min,
		Max:              max,
		Latest:           values[len(values)-1],
		Trend:            trend,
		Status:           status,
		DataPointCount:   len(values),
	}, nil
}

// GenerateDashboard generates a patient dashboard with multiple tests
func (s *Service) GenerateDashboard(ctx context.Context, patientID string, days int) (*types.DashboardData, error) {
	if days <= 0 {
		days = 30
	}

	// Get all unique codes for patient
	codes, err := s.resultStore.GetDistinctCodes(ctx, patientID)
	if err != nil {
		return nil, err
	}

	dashboard := &types.DashboardData{
		PatientID:  patientID,
		GeneratedAt: time.Now(),
		Panels:     make([]types.DashboardPanel, 0),
	}

	// Group tests by category
	categoryTests := make(map[string][]string)
	for _, code := range codes {
		testDef := s.refDB.GetTest(code)
		category := "Other"
		if testDef != nil {
			category = testDef.Category
		}
		categoryTests[category] = append(categoryTests[category], code)
	}

	// Generate sparklines for each category
	for category, tests := range categoryTests {
		panel := types.DashboardPanel{
			Category:   category,
			Tests:      make([]types.DashboardTestSummary, 0, len(tests)),
		}

		for _, code := range tests {
			sparkline, err := s.GenerateSparkline(ctx, patientID, code, days)
			if err != nil || sparkline == nil {
				continue
			}

			testDef := s.refDB.GetTest(code)
			testName := code
			unit := ""
			if testDef != nil {
				testName = testDef.Name
				unit = testDef.Unit
			}

			summary := types.DashboardTestSummary{
				Code:      code,
				Name:      testName,
				Unit:      unit,
				Latest:    sparkline.Latest,
				Trend:     sparkline.Trend,
				Status:    sparkline.Status,
				Sparkline: sparkline.Values,
			}

			panel.Tests = append(panel.Tests, summary)
		}

		if len(panel.Tests) > 0 {
			dashboard.Panels = append(dashboard.Panels, panel)
		}
	}

	// Count alerts
	for _, panel := range dashboard.Panels {
		for _, test := range panel.Tests {
			if test.Status == "high" || test.Status == "low" {
				dashboard.AlertCount++
			}
		}
	}

	return dashboard, nil
}

// toChartDataPoints converts lab results to chart data points
func (s *Service) toChartDataPoints(results []types.LabResult) []types.ChartDataPoint {
	points := make([]types.ChartDataPoint, 0, len(results))

	for _, r := range results {
		if r.ValueNumeric != nil {
			points = append(points, types.ChartDataPoint{
				Timestamp: r.CollectedAt,
				Value:     *r.ValueNumeric,
				ResultID:  r.ID.String(),
				Status:    string(r.Status),
			})
		}
	}

	return points
}

// generateAnnotations creates annotations for significant points
func (s *Service) generateAnnotations(results []types.LabResult, ranges *types.ReferenceRange) []types.ChartAnnotation {
	annotations := make([]types.ChartAnnotation, 0)

	for _, r := range results {
		if r.ValueNumeric == nil {
			continue
		}

		value := *r.ValueNumeric
		var annotation *types.ChartAnnotation

		if ranges != nil {
			// Check for critical values
			if ranges.CriticalLow != nil && value < *ranges.CriticalLow {
				annotation = &types.ChartAnnotation{
					Timestamp: r.CollectedAt,
					Type:      "critical_low",
					Label:     "Critical Low",
					Value:     value,
				}
			} else if ranges.CriticalHigh != nil && value > *ranges.CriticalHigh {
				annotation = &types.ChartAnnotation{
					Timestamp: r.CollectedAt,
					Type:      "critical_high",
					Label:     "Critical High",
					Value:     value,
				}
			} else if ranges.PanicLow != nil && value < *ranges.PanicLow {
				annotation = &types.ChartAnnotation{
					Timestamp: r.CollectedAt,
					Type:      "panic_low",
					Label:     "PANIC Low",
					Value:     value,
				}
			} else if ranges.PanicHigh != nil && value > *ranges.PanicHigh {
				annotation = &types.ChartAnnotation{
					Timestamp: r.CollectedAt,
					Type:      "panic_high",
					Label:     "PANIC High",
					Value:     value,
				}
			}
		}

		if annotation != nil {
			annotations = append(annotations, *annotation)
		}
	}

	return annotations
}
