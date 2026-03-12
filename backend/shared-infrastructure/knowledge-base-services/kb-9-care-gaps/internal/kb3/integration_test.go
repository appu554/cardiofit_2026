// Package kb3 provides integration tests for KB-3 Temporal/Guidelines service.
package kb3

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-9-care-gaps/internal/models"
)

// TestIntegrationEnrichGapWithTemporalContext tests temporal enrichment of care gaps.
func TestIntegrationEnrichGapWithTemporalContext(t *testing.T) {
	logger := zap.NewNop()

	// Create client with KB-3 disabled (will use defaults)
	client := NewClient(ClientConfig{
		BaseURL: "http://localhost:8087",
		Timeout: 10 * time.Second,
		Enabled: false, // Disabled for unit test
	}, logger)

	integration := NewIntegration(client, logger)

	// Create a test care gap
	gap := &models.CareGap{
		ID: "test-gap-001",
		Measure: models.MeasureInfo{
			Type:        models.MeasureCMS122DiabetesHbA1c,
			CMSID:       "CMS122",
			Name:        "Diabetes HbA1c Control",
			Description: "HbA1c > 9%",
		},
		Status:   models.GapStatusOpen,
		Priority: models.GapPriorityHigh,
		Reason:   "HbA1c 10.5% exceeds 9% target",
	}

	ctx := context.Background()
	temporal, err := integration.EnrichGapWithTemporalContext(ctx, gap, "patient-001")

	if err != nil {
		t.Fatalf("EnrichGapWithTemporalContext failed: %v", err)
	}

	if temporal == nil {
		t.Fatal("Expected non-nil temporal info")
	}

	// Verify default temporal info when KB-3 is disabled
	if temporal.DueDate == nil {
		t.Error("Expected DueDate to be set")
	}

	if temporal.OverdueDate == nil {
		t.Error("Expected OverdueDate to be set")
	}

	if temporal.Status != models.ConstraintPending {
		t.Errorf("Expected status PENDING, got %s", temporal.Status)
	}

	// CMS122 should have quarterly (3 month) recurrence
	if !temporal.IsRecurring {
		t.Error("Expected CMS122 to be recurring")
	}

	if temporal.RecurrenceMonths != 3 {
		t.Errorf("Expected 3 month recurrence for CMS122, got %d", temporal.RecurrenceMonths)
	}

	// Since KB-3 is disabled, SourcedFromKB3 should be false
	if temporal.SourcedFromKB3 {
		t.Error("Expected SourcedFromKB3 to be false when KB-3 is disabled")
	}

	t.Logf("Temporal info: DueDate=%v, Status=%s, Recurrence=%d months",
		temporal.DueDate, temporal.Status, temporal.RecurrenceMonths)
}

// TestIntegrationEnrichGapsWithTemporalContext tests batch temporal enrichment.
func TestIntegrationEnrichGapsWithTemporalContext(t *testing.T) {
	logger := zap.NewNop()

	client := NewClient(ClientConfig{
		BaseURL: "http://localhost:8087",
		Timeout: 10 * time.Second,
		Enabled: false,
	}, logger)

	integration := NewIntegration(client, logger)

	// Create multiple test gaps
	gaps := []models.CareGap{
		{
			ID: "gap-cms122",
			Measure: models.MeasureInfo{
				Type:  models.MeasureCMS122DiabetesHbA1c,
				CMSID: "CMS122",
				Name:  "Diabetes HbA1c",
			},
			Status:   models.GapStatusOpen,
			Priority: models.GapPriorityHigh,
		},
		{
			ID: "gap-cms165",
			Measure: models.MeasureInfo{
				Type:  models.MeasureCMS165BPControl,
				CMSID: "CMS165",
				Name:  "BP Control",
			},
			Status:   models.GapStatusOpen,
			Priority: models.GapPriorityHigh,
		},
		{
			ID: "gap-cms130",
			Measure: models.MeasureInfo{
				Type:  models.MeasureCMS130ColorectalScreening,
				CMSID: "CMS130",
				Name:  "Colorectal Screening",
			},
			Status:   models.GapStatusOpen,
			Priority: models.GapPriorityMedium,
		},
	}

	ctx := context.Background()
	enrichedGaps, err := integration.EnrichGapsWithTemporalContext(ctx, gaps, "patient-002")

	if err != nil {
		t.Fatalf("EnrichGapsWithTemporalContext failed: %v", err)
	}

	if len(enrichedGaps) != len(gaps) {
		t.Errorf("Expected %d enriched gaps, got %d", len(gaps), len(enrichedGaps))
	}

	// Verify each gap has temporal context
	for _, gap := range enrichedGaps {
		if gap.TemporalContext == nil {
			t.Errorf("Gap %s missing temporal context", gap.ID)
			continue
		}

		if gap.DueDate == nil {
			t.Errorf("Gap %s missing due date", gap.ID)
		}

		t.Logf("Gap %s: Status=%s, DueDate=%v, Recurrence=%d months",
			gap.ID, gap.TemporalContext.Status,
			gap.TemporalContext.DueDate, gap.TemporalContext.RecurrenceMonths)
	}
}

// TestIntegrationGapToScheduleRequest tests conversion from gap to KB-3 schedule request.
func TestIntegrationGapToScheduleRequest(t *testing.T) {
	logger := zap.NewNop()

	client := NewClient(ClientConfig{
		Enabled: false,
	}, logger)

	integration := NewIntegration(client, logger)

	gap := &models.CareGap{
		ID: "gap-001",
		Measure: models.MeasureInfo{
			Type:            models.MeasureCMS122DiabetesHbA1c,
			CMSID:           "CMS122",
			Name:            "Diabetes HbA1c Control",
			GuidelineSource: "NCQA",
		},
		Priority:       models.GapPriorityHigh,
		Reason:         "HbA1c not tested in measurement period",
		Recommendation: "Order HbA1c test",
	}

	request := integration.gapToScheduleRequest(gap)

	if request == nil {
		t.Fatal("Expected non-nil schedule request")
	}

	// Verify request fields
	if request.Type != ItemTypeLab {
		t.Errorf("Expected item type 'lab' for CMS122, got %s", request.Type)
	}

	if request.SourceMeasureID != "CMS122" {
		t.Errorf("Expected source measure ID 'CMS122', got %s", request.SourceMeasureID)
	}

	if request.SourceProtocol != "NCQA" {
		t.Errorf("Expected source protocol 'NCQA', got %s", request.SourceProtocol)
	}

	if !request.IsRecurring {
		t.Error("Expected CMS122 to have recurring schedule")
	}

	if request.Recurrence == nil {
		t.Error("Expected recurrence pattern to be set")
	} else {
		if request.Recurrence.Frequency != FrequencyMonthly {
			t.Errorf("Expected monthly frequency, got %s", request.Recurrence.Frequency)
		}
		if request.Recurrence.Interval != 3 {
			t.Errorf("Expected 3-month interval (quarterly), got %d", request.Recurrence.Interval)
		}
	}

	if request.Priority != 2 {
		t.Errorf("Expected priority 2 (high) for CMS122, got %d", request.Priority)
	}

	t.Logf("Schedule request: Type=%s, Recurring=%v, Interval=%d months",
		request.Type, request.IsRecurring, request.Recurrence.Interval)
}

// TestIntegrationBuildTemporalInfoFromItem tests conversion from KB-3 item to temporal info.
func TestIntegrationBuildTemporalInfoFromItem(t *testing.T) {
	logger := zap.NewNop()

	client := NewClient(ClientConfig{
		Enabled: false,
	}, logger)

	integration := NewIntegration(client, logger)

	// Create a scheduled item from KB-3
	dueDate := time.Now().Add(5 * 24 * time.Hour) // Due in 5 days
	item := &ScheduledItem{
		ItemID:          "item-001",
		PatientID:       "patient-001",
		Type:            ItemTypeLab,
		Name:            "HbA1c Test",
		DueDate:         dueDate,
		Priority:        2,
		IsRecurring:     true,
		Status:          SchedulePending,
		SourceMeasureID: "CMS122",
		Recurrence: &RecurrencePattern{
			Frequency: FrequencyMonthly,
			Interval:  3,
		},
	}

	temporal := integration.buildTemporalInfoFromItem(item)

	if temporal == nil {
		t.Fatal("Expected non-nil temporal info")
	}

	// Should be approaching since due in 5 days (within 7-day threshold)
	if temporal.Status != models.ConstraintApproaching {
		t.Errorf("Expected status APPROACHING for 5-day deadline, got %s", temporal.Status)
	}

	if temporal.DaysUntilDue > 6 || temporal.DaysUntilDue < 4 {
		t.Errorf("Expected ~5 days until due, got %d", temporal.DaysUntilDue)
	}

	if temporal.DaysOverdue != 0 {
		t.Errorf("Expected 0 days overdue, got %d", temporal.DaysOverdue)
	}

	if !temporal.IsRecurring {
		t.Error("Expected IsRecurring to be true")
	}

	if temporal.RecurrenceMonths != 3 {
		t.Errorf("Expected 3-month recurrence, got %d", temporal.RecurrenceMonths)
	}

	if !temporal.SourcedFromKB3 {
		t.Error("Expected SourcedFromKB3 to be true")
	}

	t.Logf("Temporal info from item: Status=%s, DaysUntilDue=%d, Recurrence=%d months",
		temporal.Status, temporal.DaysUntilDue, temporal.RecurrenceMonths)
}

// TestIntegrationOverdueStatus tests that overdue items are correctly identified.
func TestIntegrationOverdueStatus(t *testing.T) {
	logger := zap.NewNop()

	client := NewClient(ClientConfig{
		Enabled: false,
	}, logger)

	integration := NewIntegration(client, logger)

	// Create an overdue item
	dueDate := time.Now().Add(-10 * 24 * time.Hour) // Due 10 days ago
	item := &ScheduledItem{
		ItemID:          "item-overdue",
		PatientID:       "patient-001",
		Type:            ItemTypeLab,
		Name:            "Overdue HbA1c",
		DueDate:         dueDate,
		Priority:        1,
		Status:          ScheduleOverdue,
		SourceMeasureID: "CMS122",
	}

	temporal := integration.buildTemporalInfoFromItem(item)

	if temporal.Status != models.ConstraintOverdue {
		t.Errorf("Expected status OVERDUE, got %s", temporal.Status)
	}

	if temporal.DaysOverdue < 9 || temporal.DaysOverdue > 11 {
		t.Errorf("Expected ~10 days overdue, got %d", temporal.DaysOverdue)
	}

	if temporal.DaysUntilDue >= 0 {
		t.Errorf("Expected negative days until due, got %d", temporal.DaysUntilDue)
	}

	t.Logf("Overdue item: Status=%s, DaysOverdue=%d", temporal.Status, temporal.DaysOverdue)
}

// TestIntegrationMissedStatus tests that missed items (past grace period) are identified.
func TestIntegrationMissedStatus(t *testing.T) {
	logger := zap.NewNop()

	client := NewClient(ClientConfig{
		Enabled: false,
	}, logger)

	integration := NewIntegration(client, logger)

	// Create a missed item (past 30-day grace period)
	dueDate := time.Now().Add(-45 * 24 * time.Hour) // Due 45 days ago
	item := &ScheduledItem{
		ItemID:          "item-missed",
		PatientID:       "patient-001",
		Type:            ItemTypeLab,
		Name:            "Missed HbA1c",
		DueDate:         dueDate,
		Priority:        1,
		Status:          ScheduleOverdue,
		SourceMeasureID: "CMS122",
	}

	temporal := integration.buildTemporalInfoFromItem(item)

	if temporal.Status != models.ConstraintMissed {
		t.Errorf("Expected status MISSED (past 30-day grace), got %s", temporal.Status)
	}

	t.Logf("Missed item: Status=%s, DaysOverdue=%d", temporal.Status, temporal.DaysOverdue)
}

// TestDefaultMeasureMappings verifies the default temporal mappings for CMS measures.
func TestDefaultMeasureMappings(t *testing.T) {
	mappings := DefaultMeasureMappings()

	// CMS122 - Diabetes HbA1c (quarterly)
	if m, ok := mappings["CMS122"]; ok {
		if m.ItemType != ItemTypeLab {
			t.Errorf("CMS122 should be lab type, got %s", m.ItemType)
		}
		if m.Recurrence.Frequency != FrequencyMonthly || m.Recurrence.Interval != 3 {
			t.Errorf("CMS122 should be quarterly (3 months), got %s/%d",
				m.Recurrence.Frequency, m.Recurrence.Interval)
		}
	} else {
		t.Error("Missing CMS122 mapping")
	}

	// CMS165 - BP Control (annual)
	if m, ok := mappings["CMS165"]; ok {
		if m.ItemType != ItemTypeAppointment {
			t.Errorf("CMS165 should be appointment type, got %s", m.ItemType)
		}
		if m.Recurrence.Frequency != FrequencyYearly || m.Recurrence.Interval != 1 {
			t.Errorf("CMS165 should be annual, got %s/%d",
				m.Recurrence.Frequency, m.Recurrence.Interval)
		}
	} else {
		t.Error("Missing CMS165 mapping")
	}

	// CMS130 - Colorectal Screening (10 years)
	if m, ok := mappings["CMS130"]; ok {
		if m.ItemType != ItemTypeScreening {
			t.Errorf("CMS130 should be screening type, got %s", m.ItemType)
		}
		if m.Recurrence.Frequency != FrequencyYearly || m.Recurrence.Interval != 10 {
			t.Errorf("CMS130 should be 10-year, got %s/%d",
				m.Recurrence.Frequency, m.Recurrence.Interval)
		}
	} else {
		t.Error("Missing CMS130 mapping")
	}

	// CMS2 - Depression Screening (annual)
	if m, ok := mappings["CMS2"]; ok {
		if m.ItemType != ItemTypeAssessment {
			t.Errorf("CMS2 should be assessment type, got %s", m.ItemType)
		}
		if m.Recurrence.Frequency != FrequencyYearly || m.Recurrence.Interval != 1 {
			t.Errorf("CMS2 should be annual, got %s/%d",
				m.Recurrence.Frequency, m.Recurrence.Interval)
		}
	} else {
		t.Error("Missing CMS2 mapping")
	}

	t.Logf("Verified %d default measure mappings", len(mappings))
}
