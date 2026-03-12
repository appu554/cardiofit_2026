// Package temporal provides the scheduling engine for chronic and preventive care
package temporal

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// SchedulingEngine manages scheduled items for patients
type SchedulingEngine struct {
	mu    sync.RWMutex
	items map[string][]models.ScheduledItem // patientID -> items
}

// NewSchedulingEngine creates a new scheduling engine instance
func NewSchedulingEngine() *SchedulingEngine {
	return &SchedulingEngine{
		items: make(map[string][]models.ScheduledItem),
	}
}

// AddScheduledItem adds a new scheduled item for a patient
func (e *SchedulingEngine) AddScheduledItem(patientID string, req models.AddScheduleRequest) (*models.ScheduledItem, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()

	item := models.ScheduledItem{
		ItemID:      uuid.New().String(),
		PatientID:   patientID,
		Type:        req.Type,
		Name:        req.Name,
		Description: req.Description,
		DueDate:     req.DueDate,
		Priority:    req.Priority,
		IsRecurring: req.IsRecurring,
		Recurrence:  req.Recurrence,
		Status:      models.SchedulePending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Default priority if not set
	if item.Priority == 0 {
		item.Priority = 2 // Medium priority
	}

	e.items[patientID] = append(e.items[patientID], item)
	return &item, nil
}

// GetPatientSchedule retrieves all scheduled items for a patient
func (e *SchedulingEngine) GetPatientSchedule(patientID string) ([]models.ScheduledItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items, exists := e.items[patientID]
	if !exists {
		return []models.ScheduledItem{}, nil
	}

	// Update statuses based on current time
	e.updateItemStatuses(patientID)

	// Sort by due date
	sortedItems := make([]models.ScheduledItem, len(items))
	copy(sortedItems, items)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].DueDate.Before(sortedItems[j].DueDate)
	})

	return sortedItems, nil
}

// GetPendingItems returns all pending scheduled items for a patient
func (e *SchedulingEngine) GetPendingItems(patientID string) ([]models.ScheduledItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items, exists := e.items[patientID]
	if !exists {
		return []models.ScheduledItem{}, nil
	}

	e.updateItemStatuses(patientID)

	var pending []models.ScheduledItem
	for _, item := range items {
		if item.Status == models.SchedulePending {
			pending = append(pending, item)
		}
	}

	// Sort by due date
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].DueDate.Before(pending[j].DueDate)
	})

	return pending, nil
}

// GetOverdueItems returns all overdue scheduled items for a patient
func (e *SchedulingEngine) GetOverdueItems(patientID string) ([]models.ScheduledItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items, exists := e.items[patientID]
	if !exists {
		return []models.ScheduledItem{}, nil
	}

	e.updateItemStatuses(patientID)

	var overdue []models.ScheduledItem
	for _, item := range items {
		if item.Status == models.ScheduleOverdue {
			overdue = append(overdue, item)
		}
	}

	// Sort by priority (highest first), then by due date
	sort.Slice(overdue, func(i, j int) bool {
		if overdue[i].Priority != overdue[j].Priority {
			return overdue[i].Priority < overdue[j].Priority
		}
		return overdue[i].DueDate.Before(overdue[j].DueDate)
	})

	return overdue, nil
}

// GetUpcoming returns scheduled items due within the specified number of days
func (e *SchedulingEngine) GetUpcoming(patientID string, days int) ([]models.ScheduledItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items, exists := e.items[patientID]
	if !exists {
		return []models.ScheduledItem{}, nil
	}

	e.updateItemStatuses(patientID)

	now := time.Now()
	cutoff := now.AddDate(0, 0, days)

	var upcoming []models.ScheduledItem
	for _, item := range items {
		if item.Status == models.SchedulePending && !item.DueDate.After(cutoff) {
			upcoming = append(upcoming, item)
		}
	}

	// Sort by due date
	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].DueDate.Before(upcoming[j].DueDate)
	})

	return upcoming, nil
}

// CompleteItem marks a scheduled item as completed
func (e *SchedulingEngine) CompleteItem(patientID, itemID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	items, exists := e.items[patientID]
	if !exists {
		return fmt.Errorf("patient not found: %s", patientID)
	}

	now := time.Now()

	for i := range items {
		if items[i].ItemID == itemID {
			items[i].Status = models.ScheduleCompleted
			items[i].CompletedAt = &now
			items[i].UpdatedAt = now

			// If recurring, create next occurrence
			if items[i].IsRecurring && items[i].Recurrence != nil {
				e.createNextOccurrence(&items[i])
			}

			return nil
		}
	}

	return fmt.Errorf("item not found: %s", itemID)
}

// SkipItem marks a scheduled item as skipped
func (e *SchedulingEngine) SkipItem(patientID, itemID, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	items, exists := e.items[patientID]
	if !exists {
		return fmt.Errorf("patient not found: %s", patientID)
	}

	now := time.Now()

	for i := range items {
		if items[i].ItemID == itemID {
			items[i].Status = models.ScheduleSkipped
			items[i].UpdatedAt = now

			// If recurring, create next occurrence
			if items[i].IsRecurring && items[i].Recurrence != nil {
				e.createNextOccurrence(&items[i])
			}

			return nil
		}
	}

	return fmt.Errorf("item not found: %s", itemID)
}

// CancelItem marks a scheduled item as cancelled
func (e *SchedulingEngine) CancelItem(patientID, itemID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	items, exists := e.items[patientID]
	if !exists {
		return fmt.Errorf("patient not found: %s", patientID)
	}

	now := time.Now()

	for i := range items {
		if items[i].ItemID == itemID {
			items[i].Status = models.ScheduleCancelled
			items[i].UpdatedAt = now
			return nil
		}
	}

	return fmt.Errorf("item not found: %s", itemID)
}

// RescheduleItem updates the due date for a scheduled item
func (e *SchedulingEngine) RescheduleItem(patientID, itemID string, newDueDate time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	items, exists := e.items[patientID]
	if !exists {
		return fmt.Errorf("patient not found: %s", patientID)
	}

	now := time.Now()

	for i := range items {
		if items[i].ItemID == itemID {
			items[i].DueDate = newDueDate
			items[i].Status = models.SchedulePending
			items[i].UpdatedAt = now
			return nil
		}
	}

	return fmt.Errorf("item not found: %s", itemID)
}

// GetScheduleSummary returns a summary of a patient's schedule
func (e *SchedulingEngine) GetScheduleSummary(patientID string) models.ScheduleSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	items, exists := e.items[patientID]
	if !exists {
		return models.ScheduleSummary{PatientID: patientID}
	}

	e.updateItemStatuses(patientID)

	now := time.Now()
	weekFromNow := now.AddDate(0, 0, 7)
	monthFromNow := now.AddDate(0, 1, 0)

	summary := models.ScheduleSummary{
		PatientID:  patientID,
		TotalItems: len(items),
	}

	for _, item := range items {
		switch item.Status {
		case models.SchedulePending:
			summary.PendingItems++
			if !item.DueDate.After(weekFromNow) {
				summary.UpcomingInWeek++
			}
			if !item.DueDate.After(monthFromNow) {
				summary.UpcomingInMonth++
			}
		case models.ScheduleOverdue:
			summary.OverdueItems++
		case models.ScheduleCompleted:
			summary.CompletedItems++
		}
	}

	return summary
}

// ApplyChronicSchedule applies a chronic disease schedule to a patient
func (e *SchedulingEngine) ApplyChronicSchedule(patientID string, schedule models.ChronicSchedule, startDate time.Time) ([]models.ScheduledItem, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var createdItems []models.ScheduledItem
	now := time.Now()

	for _, item := range schedule.MonitoringItems {
		dueDate := item.Recurrence.CalculateNextOccurrence(startDate)

		scheduledItem := models.ScheduledItem{
			ItemID:         uuid.New().String(),
			PatientID:      patientID,
			Type:           item.Type,
			Name:           item.Name,
			DueDate:        dueDate,
			Priority:       2, // Default to medium
			IsRecurring:    true,
			Recurrence:     &item.Recurrence,
			Status:         models.SchedulePending,
			SourceProtocol: schedule.ScheduleID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		e.items[patientID] = append(e.items[patientID], scheduledItem)
		createdItems = append(createdItems, scheduledItem)
	}

	return createdItems, nil
}

// ApplyPreventiveSchedule applies a preventive care schedule based on patient demographics
func (e *SchedulingEngine) ApplyPreventiveSchedule(patientID string, schedule models.PreventiveSchedule, patientAge int, patientSex string, startDate time.Time) ([]models.ScheduledItem, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var createdItems []models.ScheduledItem
	now := time.Now()

	for _, item := range schedule.ScreeningItems {
		// Check if patient meets age criteria
		if patientAge < item.StartAge || patientAge > item.EndAge {
			continue
		}

		// Check if patient meets sex criteria
		if item.Sex != "any" && item.Sex != patientSex {
			continue
		}

		dueDate := item.Interval.CalculateNextOccurrence(startDate)

		scheduledItem := models.ScheduledItem{
			ItemID:         uuid.New().String(),
			PatientID:      patientID,
			Type:           models.ScheduleScreening,
			Name:           item.Name,
			Description:    item.Recommendation,
			DueDate:        dueDate,
			Priority:       getPriorityFromGrade(item.EvidenceGrade),
			IsRecurring:    item.Interval.MaxOccurrences != 1,
			Recurrence:     &item.Interval,
			Status:         models.SchedulePending,
			SourceProtocol: schedule.ScheduleID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		e.items[patientID] = append(e.items[patientID], scheduledItem)
		createdItems = append(createdItems, scheduledItem)
	}

	return createdItems, nil
}

// CalculateNextOccurrence calculates the next occurrence based on recurrence pattern
func (e *SchedulingEngine) CalculateNextOccurrence(from time.Time, pattern models.RecurrencePattern) time.Time {
	return pattern.CalculateNextOccurrence(from)
}

// GetAllOverdueItems returns all overdue items across all patients
func (e *SchedulingEngine) GetAllOverdueItems() []models.ScheduledItem {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var allOverdue []models.ScheduledItem

	for patientID := range e.items {
		e.updateItemStatuses(patientID)

		for _, item := range e.items[patientID] {
			if item.Status == models.ScheduleOverdue {
				allOverdue = append(allOverdue, item)
			}
		}
	}

	// Sort by priority, then by due date
	sort.Slice(allOverdue, func(i, j int) bool {
		if allOverdue[i].Priority != allOverdue[j].Priority {
			return allOverdue[i].Priority < allOverdue[j].Priority
		}
		return allOverdue[i].DueDate.Before(allOverdue[j].DueDate)
	})

	return allOverdue
}

// updateItemStatuses updates item statuses based on current time
func (e *SchedulingEngine) updateItemStatuses(patientID string) {
	items, exists := e.items[patientID]
	if !exists {
		return
	}

	now := time.Now()

	for i := range items {
		item := &items[i]

		// Skip non-pending items
		if item.Status != models.SchedulePending {
			continue
		}

		// Check if overdue
		if now.After(item.DueDate) {
			item.Status = models.ScheduleOverdue
		}
	}
}

// createNextOccurrence creates the next scheduled occurrence for a recurring item
func (e *SchedulingEngine) createNextOccurrence(item *models.ScheduledItem) {
	if item.Recurrence == nil {
		return
	}

	// Check max occurrences
	if item.Recurrence.MaxOccurrences > 0 {
		// Count existing occurrences for this protocol/item combination
		count := 0
		for _, existing := range e.items[item.PatientID] {
			if existing.SourceProtocol == item.SourceProtocol && existing.Name == item.Name {
				count++
			}
		}
		if count >= item.Recurrence.MaxOccurrences {
			return
		}
	}

	// Check end date
	if item.Recurrence.EndDate != nil && time.Now().After(*item.Recurrence.EndDate) {
		return
	}

	now := time.Now()
	nextDue := item.Recurrence.CalculateNextOccurrence(item.DueDate)

	newItem := models.ScheduledItem{
		ItemID:         uuid.New().String(),
		PatientID:      item.PatientID,
		Type:           item.Type,
		Name:           item.Name,
		Description:    item.Description,
		DueDate:        nextDue,
		Priority:       item.Priority,
		IsRecurring:    true,
		Recurrence:     item.Recurrence,
		Status:         models.SchedulePending,
		SourceProtocol: item.SourceProtocol,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	e.items[item.PatientID] = append(e.items[item.PatientID], newItem)
}

// getPriorityFromGrade converts evidence grade to priority
func getPriorityFromGrade(grade string) int {
	switch grade {
	case "A":
		return 1 // Highest priority
	case "B":
		return 2
	case "C":
		return 3
	case "D", "I":
		return 4
	default:
		return 2 // Default to medium
	}
}

// BatchCreateSchedules creates schedules for multiple patients
func (e *SchedulingEngine) BatchCreateSchedules(schedules []BatchScheduleRequest) ([]models.ScheduledItem, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var created []models.ScheduledItem
	now := time.Now()

	for _, req := range schedules {
		item := models.ScheduledItem{
			ItemID:         uuid.New().String(),
			PatientID:      req.PatientID,
			Type:           req.Type,
			Name:           req.Name,
			Description:    req.Description,
			DueDate:        req.DueDate,
			Priority:       req.Priority,
			IsRecurring:    req.IsRecurring,
			Recurrence:     req.Recurrence,
			Status:         models.SchedulePending,
			SourceProtocol: req.SourceProtocol,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		if item.Priority == 0 {
			item.Priority = 2
		}

		e.items[req.PatientID] = append(e.items[req.PatientID], item)
		created = append(created, item)
	}

	return created, nil
}

// BatchScheduleRequest for batch schedule creation
type BatchScheduleRequest struct {
	PatientID      string                  `json:"patient_id"`
	Type           models.ScheduleItemType `json:"type"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description,omitempty"`
	DueDate        time.Time               `json:"due_date"`
	Priority       int                     `json:"priority,omitempty"`
	IsRecurring    bool                    `json:"is_recurring,omitempty"`
	Recurrence     *models.RecurrencePattern `json:"recurrence,omitempty"`
	SourceProtocol string                  `json:"source_protocol,omitempty"`
}

// Global scheduling engine instance
var globalSchedulingEngine *SchedulingEngine
var schedulingEngineOnce sync.Once

// GetSchedulingEngine returns the global scheduling engine instance
func GetSchedulingEngine() *SchedulingEngine {
	schedulingEngineOnce.Do(func() {
		globalSchedulingEngine = NewSchedulingEngine()
	})
	return globalSchedulingEngine
}
