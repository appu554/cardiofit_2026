// Package kb3 provides integration with KB-3 Temporal/Guidelines service.
package kb3

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-9-care-gaps/internal/models"

	// Import vaidshala contracts for CQL integration
	"vaidshala/clinical-runtime-platform/contracts"
)

// Integration provides the logic to connect KB-9 care gaps with KB-3 temporal scheduling.
// This is the bridge between the "Accountability Engine" (KB-9) and the "Temporal Brain" (KB-3).
type Integration struct {
	client           *Client
	logger           *zap.Logger
	measureMappings  map[string]MeasureTemporalMapping
}

// NewIntegration creates a new KB-3 integration handler.
func NewIntegration(client *Client, logger *zap.Logger) *Integration {
	return &Integration{
		client:          client,
		logger:          logger,
		measureMappings: DefaultMeasureMappings(),
	}
}

// EnrichGapWithTemporalContext adds temporal information from KB-3 to a care gap.
// This transforms a simple gap into a time-aware gap with due dates and status.
// Returns *models.TemporalInfo for direct assignment to CareGap.TemporalContext.
func (i *Integration) EnrichGapWithTemporalContext(
	ctx context.Context,
	gap *models.CareGap,
	patientID string,
) (*models.TemporalInfo, error) {
	if !i.client.IsEnabled() {
		// Return default temporal info when KB-3 is disabled
		return i.getDefaultTemporalInfo(gap, false), nil
	}

	// Check if there's already a schedule item for this measure
	items, err := i.client.GetScheduleItemsByMeasure(ctx, patientID, gap.Measure.CMSID)
	if err != nil {
		i.logger.Warn("Failed to get schedule items from KB-3, using defaults",
			zap.String("patient_id", patientID),
			zap.String("measure_id", gap.Measure.CMSID),
			zap.Error(err),
		)
		return i.getDefaultTemporalInfo(gap, false), nil
	}

	// If we have existing schedule items, use the most recent pending one
	var relevantItem *ScheduledItem
	for idx, item := range items {
		if item.Status == SchedulePending || item.Status == ScheduleOverdue {
			relevantItem = &items[idx]
			break
		}
	}

	if relevantItem != nil {
		return i.buildTemporalInfoFromItem(relevantItem), nil
	}

	// No existing item, create temporal info from measure mappings
	return i.getDefaultTemporalInfo(gap, false), nil
}

// EnrichGapsWithTemporalContext adds temporal information to multiple care gaps.
func (i *Integration) EnrichGapsWithTemporalContext(
	ctx context.Context,
	gaps []models.CareGap,
	patientID string,
) ([]models.CareGap, error) {
	enrichedGaps := make([]models.CareGap, len(gaps))
	copy(enrichedGaps, gaps)

	for idx := range enrichedGaps {
		temporal, err := i.EnrichGapWithTemporalContext(ctx, &enrichedGaps[idx], patientID)
		if err != nil {
			i.logger.Warn("Failed to enrich gap with temporal context",
				zap.String("gap_id", enrichedGaps[idx].ID),
				zap.Error(err),
			)
			continue
		}

		// Add temporal context to the gap (now uses models.TemporalInfo)
		enrichedGaps[idx].TemporalContext = temporal

		// Update the gap's due date if we have temporal info
		if temporal.DueDate != nil {
			enrichedGaps[idx].DueDate = temporal.DueDate
		}

		// Update priority based on temporal status (using models.ConstraintStatus)
		if temporal.Status == models.ConstraintOverdue || temporal.Status == models.ConstraintMissed {
			// Escalate priority for overdue gaps
			enrichedGaps[idx].Priority = models.GapPriorityCritical
		} else if temporal.Status == models.ConstraintApproaching {
			// Increase priority for approaching deadlines
			if enrichedGaps[idx].Priority != models.GapPriorityCritical {
				enrichedGaps[idx].Priority = models.GapPriorityHigh
			}
		}
	}

	return enrichedGaps, nil
}

// CreateScheduleItemsForGaps creates KB-3 schedule items for newly identified gaps.
// This ensures that KB-3 tracks the temporal aspects of each gap.
func (i *Integration) CreateScheduleItemsForGaps(
	ctx context.Context,
	gaps []models.CareGap,
	patientID string,
) ([]ScheduledItem, error) {
	if !i.client.IsEnabled() {
		i.logger.Debug("KB-3 integration disabled, skipping schedule creation")
		return nil, nil
	}

	var createdItems []ScheduledItem

	for _, gap := range gaps {
		// Check if a schedule item already exists
		existingItems, err := i.client.GetScheduleItemsByMeasure(ctx, patientID, gap.Measure.CMSID)
		if err != nil {
			i.logger.Warn("Failed to check existing schedule items",
				zap.String("measure_id", gap.Measure.CMSID),
				zap.Error(err),
			)
			continue
		}

		// Skip if there's already a pending item
		hasPending := false
		for _, item := range existingItems {
			if item.Status == SchedulePending || item.Status == ScheduleOverdue {
				hasPending = true
				break
			}
		}
		if hasPending {
			i.logger.Debug("Schedule item already exists for measure",
				zap.String("measure_id", gap.Measure.CMSID),
			)
			continue
		}

		// Create schedule request from gap
		request := i.gapToScheduleRequest(&gap)

		item, err := i.client.CreateScheduledItem(ctx, patientID, request)
		if err != nil {
			i.logger.Error("Failed to create schedule item for gap",
				zap.String("gap_id", gap.ID),
				zap.String("measure_id", gap.Measure.CMSID),
				zap.Error(err),
			)
			continue
		}

		if item != nil {
			createdItems = append(createdItems, *item)
			i.logger.Info("Created KB-3 schedule item for care gap",
				zap.String("gap_id", gap.ID),
				zap.String("item_id", item.ItemID),
				zap.String("measure_id", gap.Measure.CMSID),
			)
		}
	}

	return createdItems, nil
}

// gapToScheduleRequest converts a care gap to a KB-3 schedule request.
func (i *Integration) gapToScheduleRequest(gap *models.CareGap) *ScheduleRequest {
	mapping, exists := i.measureMappings[gap.Measure.CMSID]
	if !exists {
		// Use default mapping
		mapping = MeasureTemporalMapping{
			MeasureID:   gap.Measure.CMSID,
			ItemType:    ItemTypeAssessment,
			GracePeriod: 30 * 24 * time.Hour,
			Priority:    3,
		}
	}

	// Calculate due date (default to 30 days from now)
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	if gap.DueDate != nil {
		dueDate = *gap.DueDate
	}

	request := &ScheduleRequest{
		Type:            mapping.ItemType,
		Name:            fmt.Sprintf("%s - %s", gap.Measure.Name, gap.Reason),
		Description:     gap.Recommendation,
		DueDate:         dueDate,
		Priority:        mapping.Priority,
		IsRecurring:     mapping.Recurrence != nil,
		Recurrence:      mapping.Recurrence,
		SourceProtocol:  gap.Measure.GuidelineSource,
		SourceMeasureID: gap.Measure.CMSID,
	}

	return request
}

// buildTemporalInfoFromItem creates a models.TemporalInfo from a KB-3 ScheduledItem.
// This converts KB-3's internal ScheduledItem to the unified models.TemporalInfo type.
func (i *Integration) buildTemporalInfoFromItem(item *ScheduledItem) *models.TemporalInfo {
	now := time.Now()
	daysUntilDue := int(time.Until(item.DueDate).Hours() / 24)
	daysOverdue := 0

	// Calculate status based on due date - map to models.ConstraintStatus
	var status models.ConstraintStatus
	if item.Status == ScheduleCompleted {
		status = models.ConstraintMet
	} else if now.After(item.DueDate) {
		daysOverdue = int(now.Sub(item.DueDate).Hours() / 24)
		if daysOverdue > 30 { // Grace period exceeded
			status = models.ConstraintMissed
		} else {
			status = models.ConstraintOverdue
		}
	} else if daysUntilDue <= 7 {
		status = models.ConstraintApproaching
	} else {
		status = models.ConstraintPending
	}

	// Calculate overdue date (due date + grace period)
	gracePeriodDuration := 30 * 24 * time.Hour // Default 30 days
	if mapping, exists := i.measureMappings[item.SourceMeasureID]; exists {
		gracePeriodDuration = mapping.GracePeriod
	}
	overdueDate := item.DueDate.Add(gracePeriodDuration)
	gracePeriodDays := int(gracePeriodDuration.Hours() / 24)

	// Calculate recurrence in months
	recurrenceMonths := 0
	isRecurring := item.IsRecurring
	if item.Recurrence != nil {
		switch item.Recurrence.Frequency {
		case FrequencyMonthly:
			recurrenceMonths = item.Recurrence.Interval
		case FrequencyYearly:
			recurrenceMonths = item.Recurrence.Interval * 12
		}
	}

	dueDate := item.DueDate // Make a copy for pointer
	temporal := &models.TemporalInfo{
		DueDate:          &dueDate,
		OverdueDate:      &overdueDate,
		GracePeriodDays:  gracePeriodDays,
		Status:           status,
		DaysUntilDue:     daysUntilDue,
		DaysOverdue:      daysOverdue,
		IsRecurring:      isRecurring,
		RecurrenceMonths: recurrenceMonths,
		SourcedFromKB3:   true, // Data came from KB-3
	}

	if item.CompletedAt != nil {
		temporal.LastCompletedDate = item.CompletedAt
	}

	return temporal
}

// getDefaultTemporalInfo creates a default temporal info when KB-3 data is unavailable.
// The sourcedFromKB3 flag indicates whether this is a fallback (false) or KB-3 derived (true).
func (i *Integration) getDefaultTemporalInfo(gap *models.CareGap, sourcedFromKB3 bool) *models.TemporalInfo {
	now := time.Now()

	// Get mapping for this measure
	mapping, exists := i.measureMappings[gap.Measure.CMSID]
	if !exists {
		mapping = MeasureTemporalMapping{
			MeasureID:   gap.Measure.CMSID,
			GracePeriod: 30 * 24 * time.Hour,
			Priority:    3,
		}
	}

	// Calculate default due date based on recurrence
	var dueDate time.Time
	recurrenceMonths := 0
	isRecurring := mapping.Recurrence != nil

	if mapping.Recurrence != nil {
		switch mapping.Recurrence.Frequency {
		case FrequencyMonthly:
			dueDate = now.AddDate(0, mapping.Recurrence.Interval, 0)
			recurrenceMonths = mapping.Recurrence.Interval
		case FrequencyYearly:
			dueDate = now.AddDate(mapping.Recurrence.Interval, 0, 0)
			recurrenceMonths = mapping.Recurrence.Interval * 12
		default:
			dueDate = now.Add(30 * 24 * time.Hour)
		}
	} else {
		dueDate = now.Add(30 * 24 * time.Hour)
	}

	overdueDate := dueDate.Add(mapping.GracePeriod)
	gracePeriodDays := int(mapping.GracePeriod.Hours() / 24)

	return &models.TemporalInfo{
		DueDate:          &dueDate,
		OverdueDate:      &overdueDate,
		GracePeriodDays:  gracePeriodDays,
		Status:           models.ConstraintPending,
		DaysUntilDue:     int(time.Until(dueDate).Hours() / 24),
		DaysOverdue:      0,
		IsRecurring:      isRecurring,
		RecurrenceMonths: recurrenceMonths,
		SourcedFromKB3:   sourcedFromKB3,
	}
}

// SyncGapClosureWithKB3 marks KB-3 schedule items as complete when a gap is closed.
// This is called when KB-9 detects that a patient now meets a measure.
func (i *Integration) SyncGapClosureWithKB3(
	ctx context.Context,
	closedGap *models.CareGap,
	patientID string,
) error {
	if !i.client.IsEnabled() {
		return nil
	}

	// Find the corresponding schedule item
	items, err := i.client.GetScheduleItemsByMeasure(ctx, patientID, closedGap.Measure.CMSID)
	if err != nil {
		return fmt.Errorf("failed to get schedule items: %w", err)
	}

	// Complete any pending items for this measure
	for _, item := range items {
		if item.Status == SchedulePending || item.Status == ScheduleOverdue {
			if err := i.client.CompleteScheduledItem(ctx, patientID, item.ItemID); err != nil {
				i.logger.Error("Failed to complete schedule item",
					zap.String("item_id", item.ItemID),
					zap.Error(err),
				)
				continue
			}
			i.logger.Info("Synced gap closure to KB-3",
				zap.String("gap_id", closedGap.ID),
				zap.String("item_id", item.ItemID),
				zap.String("measure_id", closedGap.Measure.CMSID),
			)
		}
	}

	return nil
}

// GetOverdueGapsFromKB3 retrieves care gaps that are overdue according to KB-3.
// This provides a temporal view of gaps - not just what's missing, but what's urgently missing.
func (i *Integration) GetOverdueGapsFromKB3(ctx context.Context, patientID string) ([]OverdueAlert, error) {
	if !i.client.IsEnabled() {
		return nil, nil
	}

	alerts, err := i.client.GetPatientOverdueAlerts(ctx, patientID)
	if err != nil {
		return nil, err
	}

	return alerts.Alerts, nil
}

// MeasureResultToScheduleRequest converts a vaidshala MeasureResult to KB-3 schedule request.
// This is used when CQL evaluation identifies a gap that needs temporal tracking.
func (i *Integration) MeasureResultToScheduleRequest(
	measureResult *contracts.MeasureResult,
	patientID string,
) *ScheduleRequest {
	mapping, exists := i.measureMappings[measureResult.MeasureID]
	if !exists {
		mapping = MeasureTemporalMapping{
			ItemType:    ItemTypeAssessment,
			GracePeriod: 30 * 24 * time.Hour,
			Priority:    3,
		}
	}

	// Calculate due date
	dueDate := time.Now().Add(30 * 24 * time.Hour)
	if mapping.Recurrence != nil {
		switch mapping.Recurrence.Frequency {
		case FrequencyMonthly:
			dueDate = time.Now().AddDate(0, mapping.Recurrence.Interval, 0)
		case FrequencyYearly:
			dueDate = time.Now().AddDate(mapping.Recurrence.Interval, 0, 0)
		}
	}

	return &ScheduleRequest{
		PatientID:       patientID,
		Type:            mapping.ItemType,
		Name:            measureResult.MeasureName,
		Description:     measureResult.Rationale,
		DueDate:         dueDate,
		Priority:        mapping.Priority,
		IsRecurring:     mapping.Recurrence != nil,
		Recurrence:      mapping.Recurrence,
		SourceMeasureID: measureResult.MeasureID,
	}
}
