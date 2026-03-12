// Package cohort provides cohort management for KB-11 Population Health.
package cohort

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/config"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Service provides cohort management operations.
type Service struct {
	repo           *Repository
	config         *config.Config
	logger         *logrus.Entry
	refreshLocks   map[uuid.UUID]*sync.Mutex
	refreshLocksMu sync.RWMutex
}

// NewService creates a new cohort service.
func NewService(repo *Repository, cfg *config.Config, logger *logrus.Entry) *Service {
	return &Service{
		repo:         repo,
		config:       cfg,
		logger:       logger.WithField("component", "cohort-service"),
		refreshLocks: make(map[uuid.UUID]*sync.Mutex),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Lifecycle Operations
// ──────────────────────────────────────────────────────────────────────────────

// CreateStaticCohort creates a new static cohort.
func (s *Service) CreateStaticCohort(ctx context.Context, name, description, createdBy string) (*Cohort, error) {
	// Check for duplicate name
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing cohort: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("cohort with name '%s' already exists", name)
	}

	cohort := NewStaticCohort(name, description, createdBy)

	if err := s.repo.Create(ctx, cohort); err != nil {
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"cohort_id": cohort.ID,
		"name":      name,
		"type":      "STATIC",
	}).Info("Static cohort created")

	return cohort, nil
}

// CreateDynamicCohort creates a new dynamic cohort with criteria.
func (s *Service) CreateDynamicCohort(ctx context.Context, name, description, createdBy string, criteria []Criterion) (*Cohort, error) {
	if len(criteria) == 0 {
		return nil, fmt.Errorf("dynamic cohort requires at least one criterion")
	}

	// Validate criteria
	if err := s.validateCriteria(criteria); err != nil {
		return nil, fmt.Errorf("invalid criteria: %w", err)
	}

	// Check for duplicate name
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing cohort: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("cohort with name '%s' already exists", name)
	}

	cohort := NewDynamicCohort(name, description, createdBy, criteria)

	if err := s.repo.Create(ctx, cohort); err != nil {
		return nil, err
	}

	// Perform initial refresh to populate members
	_, refreshErr := s.RefreshDynamicCohort(ctx, cohort.ID)
	if refreshErr != nil {
		s.logger.WithError(refreshErr).Warn("Initial cohort refresh failed")
	}

	s.logger.WithFields(logrus.Fields{
		"cohort_id":      cohort.ID,
		"name":           name,
		"type":           "DYNAMIC",
		"criteria_count": len(criteria),
	}).Info("Dynamic cohort created")

	return cohort, nil
}

// CreateSnapshotCohort creates a snapshot of an existing cohort.
func (s *Service) CreateSnapshotCohort(ctx context.Context, sourceCohortID uuid.UUID, createdBy string) (*Cohort, error) {
	// Get source cohort
	sourceCohort, err := s.repo.GetByID(ctx, sourceCohortID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source cohort: %w", err)
	}
	if sourceCohort == nil {
		return nil, fmt.Errorf("source cohort not found: %s", sourceCohortID)
	}

	// Create snapshot cohort
	snapshot := NewSnapshotCohort(sourceCohort, createdBy)

	if err := s.repo.Create(ctx, snapshot); err != nil {
		return nil, err
	}

	// Copy members with snapshot data
	members, err := s.repo.GetMembers(ctx, sourceCohortID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get source members: %w", err)
	}

	snapshotMembers := make([]*CohortMember, len(members))
	for i, m := range members {
		// Create snapshot data (could include patient state at snapshot time)
		snapshotData, _ := json.Marshal(map[string]interface{}{
			"original_joined_at": m.JoinedAt,
			"snapshot_time":      time.Now(),
		})

		snapshotMembers[i] = &CohortMember{
			ID:            uuid.New(),
			CohortID:      snapshot.ID,
			PatientID:     m.PatientID,
			FHIRPatientID: m.FHIRPatientID,
			JoinedAt:      time.Now(),
			IsActive:      true,
			SnapshotData:  snapshotData,
		}
	}

	if err := s.repo.BulkAddMembers(ctx, snapshotMembers); err != nil {
		return nil, fmt.Errorf("failed to copy members to snapshot: %w", err)
	}

	// Update member count
	if err := s.repo.UpdateMemberCount(ctx, snapshot.ID); err != nil {
		s.logger.WithError(err).Warn("Failed to update snapshot member count")
	}

	s.logger.WithFields(logrus.Fields{
		"snapshot_id":    snapshot.ID,
		"source_id":      sourceCohortID,
		"member_count":   len(snapshotMembers),
		"snapshot_date":  snapshot.SnapshotDate,
	}).Info("Snapshot cohort created")

	return snapshot, nil
}

// GetCohort retrieves a cohort by ID.
func (s *Service) GetCohort(ctx context.Context, id uuid.UUID) (*Cohort, error) {
	return s.repo.GetByID(ctx, id)
}

// ListCohorts retrieves all cohorts with optional filtering.
func (s *Service) ListCohorts(ctx context.Context, filter *CohortFilter) ([]*Cohort, error) {
	return s.repo.List(ctx, filter)
}

// UpdateCohort updates a cohort's metadata.
func (s *Service) UpdateCohort(ctx context.Context, id uuid.UUID, name, description string) (*Cohort, error) {
	cohort, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cohort == nil {
		return nil, fmt.Errorf("cohort not found: %s", id)
	}

	if name != "" {
		cohort.Name = name
	}
	if description != "" {
		cohort.Description = description
	}

	if err := s.repo.Update(ctx, cohort); err != nil {
		return nil, err
	}

	return cohort, nil
}

// DeleteCohort soft-deletes a cohort.
func (s *Service) DeleteCohort(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// ──────────────────────────────────────────────────────────────────────────────
// Dynamic Cohort Refresh
// ──────────────────────────────────────────────────────────────────────────────

// RefreshDynamicCohort refreshes a dynamic cohort by re-evaluating criteria.
func (s *Service) RefreshDynamicCohort(ctx context.Context, cohortID uuid.UUID) (*CohortRefreshResult, error) {
	startTime := time.Now()

	// Get cohort
	cohort, err := s.repo.GetByID(ctx, cohortID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cohort: %w", err)
	}
	if cohort == nil {
		return nil, fmt.Errorf("cohort not found: %s", cohortID)
	}
	if !cohort.IsDynamic() {
		return nil, fmt.Errorf("cohort is not dynamic: %s", cohortID)
	}

	// Acquire refresh lock for this cohort
	lock := s.getRefreshLock(cohortID)
	lock.Lock()
	defer lock.Unlock()

	previousCount := cohort.MemberCount

	// Find all patients matching criteria
	matches, err := s.repo.FindPatientsMatchingCriteria(ctx, cohort.Criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching patients: %w", err)
	}

	// Build new members list
	now := time.Now()
	newMembers := make([]*CohortMember, len(matches))
	keepPatientIDs := make([]uuid.UUID, len(matches))

	for i, m := range matches {
		newMembers[i] = &CohortMember{
			ID:            uuid.New(),
			CohortID:      cohortID,
			PatientID:     m.ID,
			FHIRPatientID: m.FHIRPatientID,
			JoinedAt:      now,
			IsActive:      true,
		}
		keepPatientIDs[i] = m.ID
	}

	// Remove members not matching criteria anymore
	removed, err := s.repo.BulkRemoveMembers(ctx, cohortID, keepPatientIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to remove old members: %w", err)
	}

	// Add new/re-add existing members
	if err := s.repo.BulkAddMembers(ctx, newMembers); err != nil {
		return nil, fmt.Errorf("failed to add new members: %w", err)
	}

	// Update timestamps and counts
	if err := s.repo.UpdateLastRefreshed(ctx, cohortID); err != nil {
		s.logger.WithError(err).Warn("Failed to update last_refreshed")
	}
	if err := s.repo.UpdateMemberCount(ctx, cohortID); err != nil {
		s.logger.WithError(err).Warn("Failed to update member count")
	}

	// Get updated count
	newCount, _ := s.repo.GetMemberCount(ctx, cohortID)

	result := &CohortRefreshResult{
		CohortID:      cohortID,
		CohortName:    cohort.Name,
		PreviousCount: previousCount,
		NewCount:      newCount,
		Added:         len(matches) - previousCount + removed,
		Removed:       removed,
		Duration:      time.Since(startTime),
		RefreshedAt:   now,
	}

	s.logger.WithFields(logrus.Fields{
		"cohort_id":      cohortID,
		"previous_count": previousCount,
		"new_count":      newCount,
		"added":          result.Added,
		"removed":        removed,
		"duration":       result.Duration.String(),
	}).Info("Dynamic cohort refreshed")

	return result, nil
}

// RefreshAllDynamicCohorts refreshes all dynamic cohorts that need refreshing.
func (s *Service) RefreshAllDynamicCohorts(ctx context.Context, refreshInterval time.Duration) ([]CohortRefreshResult, error) {
	// Get all dynamic cohorts
	cohorts, err := s.repo.List(ctx, &CohortFilter{Type: models.CohortTypeDynamic})
	if err != nil {
		return nil, fmt.Errorf("failed to list dynamic cohorts: %w", err)
	}

	var results []CohortRefreshResult
	for _, cohort := range cohorts {
		if cohort.NeedsRefresh(refreshInterval) {
			result, err := s.RefreshDynamicCohort(ctx, cohort.ID)
			if err != nil {
				s.logger.WithError(err).WithField("cohort_id", cohort.ID).Error("Failed to refresh cohort")
				continue
			}
			results = append(results, *result)
		}
	}

	s.logger.WithFields(logrus.Fields{
		"total_dynamic": len(cohorts),
		"refreshed":     len(results),
	}).Info("Dynamic cohort refresh cycle completed")

	return results, nil
}

// getRefreshLock gets or creates a mutex for a specific cohort refresh.
func (s *Service) getRefreshLock(cohortID uuid.UUID) *sync.Mutex {
	s.refreshLocksMu.RLock()
	lock, ok := s.refreshLocks[cohortID]
	s.refreshLocksMu.RUnlock()

	if ok {
		return lock
	}

	s.refreshLocksMu.Lock()
	defer s.refreshLocksMu.Unlock()

	// Double-check after acquiring write lock
	if lock, ok := s.refreshLocks[cohortID]; ok {
		return lock
	}

	lock = &sync.Mutex{}
	s.refreshLocks[cohortID] = lock
	return lock
}

// ──────────────────────────────────────────────────────────────────────────────
// Membership Operations
// ──────────────────────────────────────────────────────────────────────────────

// AddMemberToStaticCohort adds a patient to a static cohort.
func (s *Service) AddMemberToStaticCohort(ctx context.Context, cohortID, patientID uuid.UUID, fhirPatientID string) error {
	cohort, err := s.repo.GetByID(ctx, cohortID)
	if err != nil {
		return fmt.Errorf("failed to get cohort: %w", err)
	}
	if cohort == nil {
		return fmt.Errorf("cohort not found: %s", cohortID)
	}
	if !cohort.IsStatic() {
		return fmt.Errorf("can only manually add members to static cohorts")
	}

	member := &CohortMember{
		ID:            uuid.New(),
		CohortID:      cohortID,
		PatientID:     patientID,
		FHIRPatientID: fhirPatientID,
		JoinedAt:      time.Now(),
		IsActive:      true,
	}

	if err := s.repo.AddMember(ctx, member); err != nil {
		return err
	}

	if err := s.repo.UpdateMemberCount(ctx, cohortID); err != nil {
		s.logger.WithError(err).Warn("Failed to update member count")
	}

	return nil
}

// RemoveMemberFromStaticCohort removes a patient from a static cohort.
func (s *Service) RemoveMemberFromStaticCohort(ctx context.Context, cohortID, patientID uuid.UUID) error {
	cohort, err := s.repo.GetByID(ctx, cohortID)
	if err != nil {
		return fmt.Errorf("failed to get cohort: %w", err)
	}
	if cohort == nil {
		return fmt.Errorf("cohort not found: %s", cohortID)
	}
	if !cohort.IsStatic() {
		return fmt.Errorf("can only manually remove members from static cohorts")
	}

	if err := s.repo.RemoveMember(ctx, cohortID, patientID); err != nil {
		return err
	}

	if err := s.repo.UpdateMemberCount(ctx, cohortID); err != nil {
		s.logger.WithError(err).Warn("Failed to update member count")
	}

	return nil
}

// GetCohortMembers retrieves members of a cohort.
func (s *Service) GetCohortMembers(ctx context.Context, cohortID uuid.UUID, limit, offset int) ([]*CohortMember, error) {
	return s.repo.GetMembers(ctx, cohortID, limit, offset)
}

// IsMemberOfCohort checks if a patient is a member of a cohort.
func (s *Service) IsMemberOfCohort(ctx context.Context, cohortID, patientID uuid.UUID) (bool, error) {
	return s.repo.IsMember(ctx, cohortID, patientID)
}

// ──────────────────────────────────────────────────────────────────────────────
// Analytics
// ──────────────────────────────────────────────────────────────────────────────

// GetCohortStats retrieves statistics for a cohort.
func (s *Service) GetCohortStats(ctx context.Context, cohortID uuid.UUID) (*CohortStats, error) {
	return s.repo.GetCohortStats(ctx, cohortID)
}

// CompareCohorts compares two cohorts and returns overlap statistics.
func (s *Service) CompareCohorts(ctx context.Context, cohortID1, cohortID2 uuid.UUID) (*CohortComparison, error) {
	// Get members of both cohorts
	members1, err := s.repo.GetMembers(ctx, cohortID1, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get members of cohort 1: %w", err)
	}

	members2, err := s.repo.GetMembers(ctx, cohortID2, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get members of cohort 2: %w", err)
	}

	// Build sets for comparison
	set1 := make(map[uuid.UUID]bool)
	for _, m := range members1 {
		set1[m.PatientID] = true
	}

	set2 := make(map[uuid.UUID]bool)
	for _, m := range members2 {
		set2[m.PatientID] = true
	}

	// Calculate overlap
	var overlap int
	var only1 int
	var only2 int

	for pid := range set1 {
		if set2[pid] {
			overlap++
		} else {
			only1++
		}
	}

	for pid := range set2 {
		if !set1[pid] {
			only2++
		}
	}

	// Get cohort info
	cohort1, _ := s.repo.GetByID(ctx, cohortID1)
	cohort2, _ := s.repo.GetByID(ctx, cohortID2)

	var name1, name2 string
	if cohort1 != nil {
		name1 = cohort1.Name
	}
	if cohort2 != nil {
		name2 = cohort2.Name
	}

	return &CohortComparison{
		CohortID1:    cohortID1,
		CohortName1:  name1,
		CohortID2:    cohortID2,
		CohortName2:  name2,
		Count1:       len(members1),
		Count2:       len(members2),
		Overlap:      overlap,
		OnlyIn1:      only1,
		OnlyIn2:      only2,
		JaccardIndex: calculateJaccardIndex(len(set1), len(set2), overlap),
		CalculatedAt: time.Now(),
	}, nil
}

// CohortComparison represents a comparison between two cohorts.
type CohortComparison struct {
	CohortID1    uuid.UUID `json:"cohort_id_1"`
	CohortName1  string    `json:"cohort_name_1"`
	CohortID2    uuid.UUID `json:"cohort_id_2"`
	CohortName2  string    `json:"cohort_name_2"`
	Count1       int       `json:"count_1"`
	Count2       int       `json:"count_2"`
	Overlap      int       `json:"overlap"`
	OnlyIn1      int       `json:"only_in_1"`
	OnlyIn2      int       `json:"only_in_2"`
	JaccardIndex float64   `json:"jaccard_index"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// calculateJaccardIndex calculates the Jaccard similarity index.
func calculateJaccardIndex(size1, size2, overlap int) float64 {
	union := size1 + size2 - overlap
	if union == 0 {
		return 0
	}
	return float64(overlap) / float64(union)
}

// ──────────────────────────────────────────────────────────────────────────────
// Validation
// ──────────────────────────────────────────────────────────────────────────────

// validateCriteria validates cohort criteria.
func (s *Service) validateCriteria(criteria []Criterion) error {
	validFields := map[string]bool{
		"current_risk_tier":   true,
		"risk_tier":           true,
		"current_risk_score":  true,
		"risk_score":          true,
		"age":                 true,
		"gender":              true,
		"attributed_pcp":      true,
		"attributed_practice": true,
		"care_gap_count":      true,
		"last_encounter_date": true,
	}

	for i, c := range criteria {
		if c.Field == "" {
			return fmt.Errorf("criterion %d: field is required", i)
		}
		if !validFields[c.Field] {
			return fmt.Errorf("criterion %d: invalid field '%s'", i, c.Field)
		}
		if !c.Operator.IsValid() {
			return fmt.Errorf("criterion %d: invalid operator '%s'", i, c.Operator)
		}
		if c.Value == nil {
			return fmt.Errorf("criterion %d: value is required", i)
		}
		if c.Logic != "" && c.Logic != "AND" && c.Logic != "OR" {
			return fmt.Errorf("criterion %d: invalid logic '%s' (must be AND or OR)", i, c.Logic)
		}
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Predefined Cohort Helpers
// ──────────────────────────────────────────────────────────────────────────────

// CreateHighRiskCohort creates a predefined high-risk cohort.
func (s *Service) CreateHighRiskCohort(ctx context.Context, createdBy string) (*Cohort, error) {
	return s.CreateDynamicCohort(
		ctx,
		"High Risk Patients",
		"Patients with HIGH or VERY_HIGH risk tier requiring intensive management",
		createdBy,
		HighRiskCriteria(),
	)
}

// CreateRisingRiskCohort creates a predefined rising-risk cohort.
func (s *Service) CreateRisingRiskCohort(ctx context.Context, createdBy string) (*Cohort, error) {
	return s.CreateDynamicCohort(
		ctx,
		"Rising Risk Patients",
		"Patients with RISING risk trend requiring early intervention",
		createdBy,
		RisingRiskCriteria(),
	)
}

// CreateCareGapCohort creates a cohort of patients with significant care gaps.
func (s *Service) CreateCareGapCohort(ctx context.Context, minGaps int, createdBy string) (*Cohort, error) {
	return s.CreateDynamicCohort(
		ctx,
		fmt.Sprintf("Care Gap Patients (>=%d gaps)", minGaps),
		fmt.Sprintf("Patients with %d or more open care gaps", minGaps),
		createdBy,
		CareGapCriteria(minGaps),
	)
}

// CreatePCPCohort creates a cohort of patients attributed to a specific PCP.
func (s *Service) CreatePCPCohort(ctx context.Context, pcp, createdBy string) (*Cohort, error) {
	return s.CreateDynamicCohort(
		ctx,
		fmt.Sprintf("PCP Panel: %s", pcp),
		fmt.Sprintf("Patients attributed to %s", pcp),
		createdBy,
		PCPCriteria(pcp),
	)
}

// CreatePracticeCohort creates a cohort of patients in a specific practice.
func (s *Service) CreatePracticeCohort(ctx context.Context, practice, createdBy string) (*Cohort, error) {
	return s.CreateDynamicCohort(
		ctx,
		fmt.Sprintf("Practice: %s", practice),
		fmt.Sprintf("Patients attributed to %s practice", practice),
		createdBy,
		PracticeCriteria(practice),
	)
}
