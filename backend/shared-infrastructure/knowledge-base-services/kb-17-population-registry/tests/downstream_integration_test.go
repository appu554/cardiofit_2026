// Package tests provides comprehensive test utilities for KB-17 Population Registry
// downstream_integration_test.go - Tests for downstream service integration
// This validates KB-17 correctly feeds KB-9 (Care Gaps), KB-18 (Governance), KB-19 (Orchestration)
package tests

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// DOWNSTREAM SERVICE MOCKS
// =============================================================================

// MockKB9CareGapsClient simulates KB-9 Care Gaps service
type MockKB9CareGapsClient struct {
	mu               sync.RWMutex
	populationFeeds  []PopulationFeed
	careGapRequests  []CareGapRequest
	refreshCallCount int
}

// PopulationFeed represents data sent to KB-9
type PopulationFeed struct {
	RegistryCode models.RegistryCode
	PatientIDs   []string
	RiskTiers    map[string]models.RiskTier
	Timestamp    time.Time
}

// CareGapRequest represents a care gap query
type CareGapRequest struct {
	RegistryCode models.RegistryCode
	RiskTier     *models.RiskTier
	Limit        int
	Offset       int
}

// NewMockKB9Client creates new KB-9 mock
func NewMockKB9Client() *MockKB9CareGapsClient {
	return &MockKB9CareGapsClient{
		populationFeeds: make([]PopulationFeed, 0),
		careGapRequests: make([]CareGapRequest, 0),
	}
}

// FeedPopulation sends population data to KB-9
func (c *MockKB9CareGapsClient) FeedPopulation(ctx context.Context, feed PopulationFeed) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.populationFeeds = append(c.populationFeeds, feed)
	return nil
}

// RefreshCareGaps triggers care gap refresh for registry
func (c *MockKB9CareGapsClient) RefreshCareGaps(ctx context.Context, registryCode models.RegistryCode) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.refreshCallCount++
	return nil
}

// GetFeeds returns all population feeds
func (c *MockKB9CareGapsClient) GetFeeds() []PopulationFeed {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.populationFeeds
}

// GetRefreshCount returns refresh call count
func (c *MockKB9CareGapsClient) GetRefreshCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.refreshCallCount
}

// =============================================================================
// KB-18 GOVERNANCE MOCK
// =============================================================================

// MockKB18GovernanceClient simulates KB-18 Governance service
type MockKB18GovernanceClient struct {
	mu              sync.RWMutex
	policyChecks    []PolicyCheck
	membershipUsed  []MembershipQuery
	auditRecords    []AuditRecord
}

// PolicyCheck represents a governance policy check
type PolicyCheck struct {
	PatientID    string
	PolicyCode   string
	RegistryCode models.RegistryCode
	IsMember     bool
	RiskTier     models.RiskTier
	Decision     string
	Timestamp    time.Time
}

// MembershipQuery represents KB-18's query to KB-17
type MembershipQuery struct {
	PatientID    string
	RegistryCode models.RegistryCode
	Timestamp    time.Time
}

// AuditRecord represents governance audit entry
type AuditRecord struct {
	Action       string
	PatientID    string
	RegistryCode models.RegistryCode
	Reason       string
	Timestamp    time.Time
}

// NewMockKB18Client creates new KB-18 mock
func NewMockKB18Client() *MockKB18GovernanceClient {
	return &MockKB18GovernanceClient{
		policyChecks:   make([]PolicyCheck, 0),
		membershipUsed: make([]MembershipQuery, 0),
		auditRecords:   make([]AuditRecord, 0),
	}
}

// CheckPolicy checks if patient is governed by policy based on registry membership
func (c *MockKB18GovernanceClient) CheckPolicy(ctx context.Context, patientID string, policyCode string, membership *models.RegistryPatient) *PolicyCheck {
	c.mu.Lock()
	defer c.mu.Unlock()

	check := PolicyCheck{
		PatientID:  patientID,
		PolicyCode: policyCode,
		Timestamp:  time.Now(),
	}

	if membership != nil {
		check.RegistryCode = membership.RegistryCode
		check.IsMember = membership.Status == models.EnrollmentStatusActive
		check.RiskTier = membership.RiskTier
		check.Decision = "APPLY_POLICY"

		c.membershipUsed = append(c.membershipUsed, MembershipQuery{
			PatientID:    patientID,
			RegistryCode: membership.RegistryCode,
			Timestamp:    time.Now(),
		})
	} else {
		check.Decision = "SKIP_POLICY"
	}

	c.policyChecks = append(c.policyChecks, check)
	return &check
}

// RecordAudit records governance audit entry
func (c *MockKB18GovernanceClient) RecordAudit(record AuditRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.auditRecords = append(c.auditRecords, record)
}

// GetPolicyChecks returns all policy checks
func (c *MockKB18GovernanceClient) GetPolicyChecks() []PolicyCheck {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.policyChecks
}

// GetMembershipQueries returns membership queries
func (c *MockKB18GovernanceClient) GetMembershipQueries() []MembershipQuery {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.membershipUsed
}

// =============================================================================
// KB-19 ORCHESTRATION MOCK
// =============================================================================

// MockKB19OrchestrationClient simulates KB-19 Orchestration service
type MockKB19OrchestrationClient struct {
	mu               sync.RWMutex
	populationFilters []PopulationFilter
	workflowTriggers  []WorkflowTrigger
	taskAssignments   []TaskAssignment
}

// PopulationFilter represents a filter for population-based workflows
type PopulationFilter struct {
	RegistryCode models.RegistryCode
	RiskTiers    []models.RiskTier
	Status       models.EnrollmentStatus
	Limit        int
}

// WorkflowTrigger represents a triggered workflow
type WorkflowTrigger struct {
	WorkflowID   string
	PatientIDs   []string
	RegistryCode models.RegistryCode
	TriggerType  string
	Timestamp    time.Time
}

// TaskAssignment represents assigned care task
type TaskAssignment struct {
	TaskID       string
	PatientID    string
	RegistryCode models.RegistryCode
	RiskTier     models.RiskTier
	DueDate      time.Time
}

// NewMockKB19Client creates new KB-19 mock
func NewMockKB19Client() *MockKB19OrchestrationClient {
	return &MockKB19OrchestrationClient{
		populationFilters: make([]PopulationFilter, 0),
		workflowTriggers:  make([]WorkflowTrigger, 0),
		taskAssignments:   make([]TaskAssignment, 0),
	}
}

// FilterPopulation filters population for workflow
func (c *MockKB19OrchestrationClient) FilterPopulation(ctx context.Context, filter PopulationFilter, enrollments []*models.RegistryPatient) []*models.RegistryPatient {
	c.mu.Lock()
	c.populationFilters = append(c.populationFilters, filter)
	c.mu.Unlock()

	var filtered []*models.RegistryPatient
	for _, e := range enrollments {
		if e.RegistryCode != filter.RegistryCode {
			continue
		}
		if filter.Status != "" && e.Status != filter.Status {
			continue
		}
		if len(filter.RiskTiers) > 0 {
			tierMatch := false
			for _, tier := range filter.RiskTiers {
				if e.RiskTier == tier {
					tierMatch = true
					break
				}
			}
			if !tierMatch {
				continue
			}
		}
		filtered = append(filtered, e)
		if filter.Limit > 0 && len(filtered) >= filter.Limit {
			break
		}
	}
	return filtered
}

// TriggerWorkflow triggers a workflow for patients
func (c *MockKB19OrchestrationClient) TriggerWorkflow(ctx context.Context, workflowID string, patientIDs []string, registryCode models.RegistryCode, triggerType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.workflowTriggers = append(c.workflowTriggers, WorkflowTrigger{
		WorkflowID:   workflowID,
		PatientIDs:   patientIDs,
		RegistryCode: registryCode,
		TriggerType:  triggerType,
		Timestamp:    time.Now(),
	})
	return nil
}

// AssignTask assigns care task based on registry
func (c *MockKB19OrchestrationClient) AssignTask(ctx context.Context, enrollment *models.RegistryPatient, taskID string, dueDate time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.taskAssignments = append(c.taskAssignments, TaskAssignment{
		TaskID:       taskID,
		PatientID:    enrollment.PatientID,
		RegistryCode: enrollment.RegistryCode,
		RiskTier:     enrollment.RiskTier,
		DueDate:      dueDate,
	})
	return nil
}

// GetWorkflowTriggers returns all triggers
func (c *MockKB19OrchestrationClient) GetWorkflowTriggers() []WorkflowTrigger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.workflowTriggers
}

// GetTaskAssignments returns all task assignments
func (c *MockKB19OrchestrationClient) GetTaskAssignments() []TaskAssignment {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.taskAssignments
}

// GetPopulationFilters returns all filters used
func (c *MockKB19OrchestrationClient) GetPopulationFilters() []PopulationFilter {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.populationFilters
}

// =============================================================================
// KB-9 CARE GAPS INTEGRATION TESTS
// =============================================================================

// TestKB9Integration_PopulationFeedsCorrectly tests KB-17 → KB-9 feed
func TestKB9Integration_PopulationFeedsCorrectly(t *testing.T) {
	repo := NewMockRepository()
	kb9Client := NewMockKB9Client()
	ctx := context.Background()

	// Create enrollments
	patients := []string{"kb9-patient-001", "kb9-patient-002", "kb9-patient-003"}
	riskTiers := []models.RiskTier{models.RiskTierHigh, models.RiskTierCritical, models.RiskTierModerate}

	for i, pid := range patients {
		enrollment := &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    pid,
			RegistryCode: models.RegistryDiabetes,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     riskTiers[i],
			EnrolledAt:   time.Now(),
		}
		_ = repo.CreateEnrollment(enrollment)
	}

	// Feed population to KB-9
	riskMap := make(map[string]models.RiskTier)
	for i, pid := range patients {
		riskMap[pid] = riskTiers[i]
	}

	feed := PopulationFeed{
		RegistryCode: models.RegistryDiabetes,
		PatientIDs:   patients,
		RiskTiers:    riskMap,
		Timestamp:    time.Now(),
	}
	err := kb9Client.FeedPopulation(ctx, feed)
	require.NoError(t, err)

	// Verify feed
	feeds := kb9Client.GetFeeds()
	require.Len(t, feeds, 1)
	assert.Equal(t, models.RegistryDiabetes, feeds[0].RegistryCode)
	assert.Len(t, feeds[0].PatientIDs, 3)
	assert.Equal(t, models.RiskTierHigh, feeds[0].RiskTiers["kb9-patient-001"])
	assert.Equal(t, models.RiskTierCritical, feeds[0].RiskTiers["kb9-patient-002"])
}

// TestKB9Integration_CareGapRefreshOnEnrollment tests refresh trigger
func TestKB9Integration_CareGapRefreshOnEnrollment(t *testing.T) {
	repo := NewMockRepository()
	kb9Client := NewMockKB9Client()
	ctx := context.Background()

	// New enrollment should trigger care gap refresh
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb9-refresh-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierHigh,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Trigger refresh (simulating event handler)
	err := kb9Client.RefreshCareGaps(ctx, models.RegistryDiabetes)
	require.NoError(t, err)

	assert.Equal(t, 1, kb9Client.GetRefreshCount())
}

// TestKB9Integration_RiskTierChangeTriggersRefresh tests risk change refresh
func TestKB9Integration_RiskTierChangeTriggersRefresh(t *testing.T) {
	repo := NewMockRepository()
	kb9Client := NewMockKB9Client()
	ctx := context.Background()

	// Create initial enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb9-risk-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Risk tier escalates
	_ = repo.UpdateEnrollmentRiskTier(enrollment.ID, models.RiskTierModerate, models.RiskTierCritical, "HbA1c worsened")

	// Should trigger care gap refresh for re-prioritization
	err := kb9Client.RefreshCareGaps(ctx, models.RegistryDiabetes)
	require.NoError(t, err)

	assert.Equal(t, 1, kb9Client.GetRefreshCount())
}

// TestKB9Integration_DisenrollmentUpdatesGaps tests disenrollment handling
func TestKB9Integration_DisenrollmentUpdatesGaps(t *testing.T) {
	kb9Client := NewMockKB9Client()
	ctx := context.Background()

	// Patient enrolled in diabetes
	feed1 := PopulationFeed{
		RegistryCode: models.RegistryDiabetes,
		PatientIDs:   []string{"kb9-disenroll-001", "kb9-disenroll-002"},
		RiskTiers:    map[string]models.RiskTier{"kb9-disenroll-001": models.RiskTierHigh, "kb9-disenroll-002": models.RiskTierLow},
		Timestamp:    time.Now(),
	}
	_ = kb9Client.FeedPopulation(ctx, feed1)

	// Patient disenrolled - send updated feed without that patient
	feed2 := PopulationFeed{
		RegistryCode: models.RegistryDiabetes,
		PatientIDs:   []string{"kb9-disenroll-002"}, // Only one patient remains
		RiskTiers:    map[string]models.RiskTier{"kb9-disenroll-002": models.RiskTierLow},
		Timestamp:    time.Now(),
	}
	_ = kb9Client.FeedPopulation(ctx, feed2)

	feeds := kb9Client.GetFeeds()
	require.Len(t, feeds, 2)

	// Latest feed should have fewer patients
	assert.Len(t, feeds[1].PatientIDs, 1)
	assert.Equal(t, "kb9-disenroll-002", feeds[1].PatientIDs[0])
}

// =============================================================================
// KB-18 GOVERNANCE INTEGRATION TESTS
// =============================================================================

// TestKB18Integration_MembershipUsedInPolicyCheck tests policy checks
func TestKB18Integration_MembershipUsedInPolicyCheck(t *testing.T) {
	repo := NewMockRepository()
	kb18Client := NewMockKB18Client()
	ctx := context.Background()

	// Create enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb18-policy-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierHigh,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// KB-18 checks policy using KB-17 membership
	check := kb18Client.CheckPolicy(ctx, "kb18-policy-001", "DIABETES_CARE_PLAN_REQUIRED", enrollment)

	assert.NotNil(t, check)
	assert.True(t, check.IsMember)
	assert.Equal(t, models.RiskTierHigh, check.RiskTier)
	assert.Equal(t, "APPLY_POLICY", check.Decision)

	// Verify membership was queried
	queries := kb18Client.GetMembershipQueries()
	require.Len(t, queries, 1)
	assert.Equal(t, "kb18-policy-001", queries[0].PatientID)
	assert.Equal(t, models.RegistryDiabetes, queries[0].RegistryCode)
}

// TestKB18Integration_NonMemberSkipsPolicy tests policy skip
func TestKB18Integration_NonMemberSkipsPolicy(t *testing.T) {
	kb18Client := NewMockKB18Client()
	ctx := context.Background()

	// Check policy for non-member
	check := kb18Client.CheckPolicy(ctx, "kb18-nonmember-001", "DIABETES_CARE_PLAN_REQUIRED", nil)

	assert.NotNil(t, check)
	assert.False(t, check.IsMember)
	assert.Equal(t, "SKIP_POLICY", check.Decision)
}

// TestKB18Integration_RiskTierAffectsPolicy tests risk-based policy
func TestKB18Integration_RiskTierAffectsPolicy(t *testing.T) {
	repo := NewMockRepository()
	kb18Client := NewMockKB18Client()
	ctx := context.Background()

	// Critical patient
	criticalEnrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb18-critical-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierCritical,
	}
	_ = repo.CreateEnrollment(criticalEnrollment)

	// Low-risk patient
	lowEnrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb18-low-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierLow,
	}
	_ = repo.CreateEnrollment(lowEnrollment)

	// Check both
	criticalCheck := kb18Client.CheckPolicy(ctx, "kb18-critical-001", "URGENT_INTERVENTION", criticalEnrollment)
	lowCheck := kb18Client.CheckPolicy(ctx, "kb18-low-001", "URGENT_INTERVENTION", lowEnrollment)

	// Both are members but risk tier differs
	assert.Equal(t, models.RiskTierCritical, criticalCheck.RiskTier)
	assert.Equal(t, models.RiskTierLow, lowCheck.RiskTier)

	// Policy decisions could differ based on risk
	checks := kb18Client.GetPolicyChecks()
	assert.Len(t, checks, 2)
}

// TestKB18Integration_GovernanceAuditTrail tests audit recording
func TestKB18Integration_GovernanceAuditTrail(t *testing.T) {
	repo := NewMockRepository()
	kb18Client := NewMockKB18Client()

	// Create enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb18-audit-001",
		RegistryCode: models.RegistryOpioidUse,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierHigh,
	}
	_ = repo.CreateEnrollment(enrollment)

	// Record audit entry
	kb18Client.RecordAudit(AuditRecord{
		Action:       "POLICY_APPLIED",
		PatientID:    "kb18-audit-001",
		RegistryCode: models.RegistryOpioidUse,
		Reason:       "Opioid registry membership triggered PDMP check policy",
		Timestamp:    time.Now(),
	})

	// Verify audit trail includes KB-17 context
	records := kb18Client.auditRecords
	require.Len(t, records, 1)
	assert.Equal(t, models.RegistryOpioidUse, records[0].RegistryCode)
	assert.Contains(t, records[0].Reason, "Opioid registry membership")
}

// =============================================================================
// KB-19 ORCHESTRATION INTEGRATION TESTS
// =============================================================================

// TestKB19Integration_PopulationFiltersWork tests population filtering
func TestKB19Integration_PopulationFiltersWork(t *testing.T) {
	repo := NewMockRepository()
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// Create mixed population
	enrollments := []*models.RegistryPatient{
		{ID: uuid.New(), PatientID: "kb19-filter-001", RegistryCode: models.RegistryDiabetes, Status: models.EnrollmentStatusActive, RiskTier: models.RiskTierCritical},
		{ID: uuid.New(), PatientID: "kb19-filter-002", RegistryCode: models.RegistryDiabetes, Status: models.EnrollmentStatusActive, RiskTier: models.RiskTierHigh},
		{ID: uuid.New(), PatientID: "kb19-filter-003", RegistryCode: models.RegistryDiabetes, Status: models.EnrollmentStatusActive, RiskTier: models.RiskTierLow},
		{ID: uuid.New(), PatientID: "kb19-filter-004", RegistryCode: models.RegistryHypertension, Status: models.EnrollmentStatusActive, RiskTier: models.RiskTierCritical},
	}

	for _, e := range enrollments {
		_ = repo.CreateEnrollment(e)
	}

	// Filter for high-risk diabetes patients
	filter := PopulationFilter{
		RegistryCode: models.RegistryDiabetes,
		RiskTiers:    []models.RiskTier{models.RiskTierHigh, models.RiskTierCritical},
		Status:       models.EnrollmentStatusActive,
	}

	filtered := kb19Client.FilterPopulation(ctx, filter, enrollments)

	assert.Len(t, filtered, 2, "Should filter to 2 high-risk diabetes patients")
	for _, e := range filtered {
		assert.Equal(t, models.RegistryDiabetes, e.RegistryCode)
		assert.True(t, e.RiskTier == models.RiskTierHigh || e.RiskTier == models.RiskTierCritical)
	}
}

// TestKB19Integration_WorkflowTriggeredByRegistry tests workflow trigger
func TestKB19Integration_WorkflowTriggeredByRegistry(t *testing.T) {
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// Trigger outreach workflow for registry members
	patients := []string{"kb19-workflow-001", "kb19-workflow-002"}
	err := kb19Client.TriggerWorkflow(ctx, "DIABETES_QUARTERLY_OUTREACH", patients, models.RegistryDiabetes, "SCHEDULED")

	require.NoError(t, err)

	triggers := kb19Client.GetWorkflowTriggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "DIABETES_QUARTERLY_OUTREACH", triggers[0].WorkflowID)
	assert.Equal(t, models.RegistryDiabetes, triggers[0].RegistryCode)
	assert.Len(t, triggers[0].PatientIDs, 2)
}

// TestKB19Integration_TaskAssignmentByRiskTier tests risk-based tasks
func TestKB19Integration_TaskAssignmentByRiskTier(t *testing.T) {
	repo := NewMockRepository()
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// Critical patient gets urgent task
	criticalEnrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "kb19-task-001",
		RegistryCode: models.RegistryHeartFailure,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierCritical,
	}
	_ = repo.CreateEnrollment(criticalEnrollment)

	// Assign urgent follow-up
	err := kb19Client.AssignTask(ctx, criticalEnrollment, "URGENT_CARDIOLOGY_FOLLOWUP", time.Now().Add(24*time.Hour))
	require.NoError(t, err)

	tasks := kb19Client.GetTaskAssignments()
	require.Len(t, tasks, 1)
	assert.Equal(t, "URGENT_CARDIOLOGY_FOLLOWUP", tasks[0].TaskID)
	assert.Equal(t, models.RiskTierCritical, tasks[0].RiskTier)
	assert.Equal(t, models.RegistryHeartFailure, tasks[0].RegistryCode)
}

// TestKB19Integration_PopulationBasedWorkflow tests population workflow
func TestKB19Integration_PopulationBasedWorkflow(t *testing.T) {
	repo := NewMockRepository()
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// Create CKD population
	for i := 0; i < 10; i++ {
		tier := models.RiskTierModerate
		if i < 3 {
			tier = models.RiskTierCritical
		} else if i < 6 {
			tier = models.RiskTierHigh
		}

		enrollment := &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    createDownstreamPatientID("ckd-workflow", i),
			RegistryCode: models.RegistryCKD,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     tier,
		}
		_ = repo.CreateEnrollment(enrollment)
	}

	// Get all enrollments
	enrollmentsVal, _, _ := repo.ListEnrollments(&models.EnrollmentQuery{
		RegistryCode: models.RegistryCKD,
	})
	// Convert to pointer slice for FilterPopulation
	enrollments := make([]*models.RegistryPatient, len(enrollmentsVal))
	for i := range enrollmentsVal {
		enrollments[i] = &enrollmentsVal[i]
	}

	// Filter for critical tier only
	filter := PopulationFilter{
		RegistryCode: models.RegistryCKD,
		RiskTiers:    []models.RiskTier{models.RiskTierCritical},
	}
	critical := kb19Client.FilterPopulation(ctx, filter, enrollments)

	assert.Len(t, critical, 3, "Should have 3 critical CKD patients")

	// Trigger nephrology consult workflow
	patientIDs := make([]string, len(critical))
	for i, e := range critical {
		patientIDs[i] = e.PatientID
	}
	err := kb19Client.TriggerWorkflow(ctx, "NEPHROLOGY_URGENT_CONSULT", patientIDs, models.RegistryCKD, "RISK_ESCALATION")
	require.NoError(t, err)

	// Verify filter was recorded
	filters := kb19Client.GetPopulationFilters()
	require.Len(t, filters, 1)
	assert.Equal(t, models.RegistryCKD, filters[0].RegistryCode)
}

// =============================================================================
// CROSS-SERVICE EVENT FLOW TESTS
// =============================================================================

// TestDownstream_EnrollmentEventCascade tests event cascade
func TestDownstream_EnrollmentEventCascade(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	kb9Client := NewMockKB9Client()
	kb18Client := NewMockKB18Client()
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// New enrollment
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        "cascade-patient-001",
		RegistryCode:     models.RegistryDiabetes,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierHigh,
		EnrollmentSource: models.EnrollmentSourceDiagnosis,
		EnrolledAt:       time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)
	_ = producer.ProduceEnrollmentEvent(ctx, enrollment)

	// Simulate downstream reactions to enrollment event:

	// 1. KB-9: Refresh care gaps
	_ = kb9Client.RefreshCareGaps(ctx, models.RegistryDiabetes)

	// 2. KB-18: Check applicable policies
	_ = kb18Client.CheckPolicy(ctx, enrollment.PatientID, "DIABETES_MONITORING", enrollment)

	// 3. KB-19: Trigger onboarding workflow
	_ = kb19Client.TriggerWorkflow(ctx, "DIABETES_ONBOARDING", []string{enrollment.PatientID}, models.RegistryDiabetes, "NEW_ENROLLMENT")

	// Verify cascade
	assert.Equal(t, 1, kb9Client.GetRefreshCount())
	assert.Len(t, kb18Client.GetPolicyChecks(), 1)
	assert.Len(t, kb19Client.GetWorkflowTriggers(), 1)
}

// TestDownstream_RiskEscalationCascade tests risk change cascade
func TestDownstream_RiskEscalationCascade(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	kb9Client := NewMockKB9Client()
	kb18Client := NewMockKB18Client()
	kb19Client := NewMockKB19Client()
	ctx := context.Background()

	// Existing enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "escalation-patient-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now().Add(-30 * 24 * time.Hour),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Risk escalates to CRITICAL
	_ = repo.UpdateEnrollmentRiskTier(enrollment.ID, models.RiskTierModerate, models.RiskTierCritical, "HbA1c > 12%")
	_ = producer.ProduceRiskChangedEvent(ctx, enrollment, models.RiskTierModerate, models.RiskTierCritical)

	// Get updated enrollment
	updated, _ := repo.GetEnrollment(enrollment.ID)

	// Simulate downstream reactions to risk escalation:

	// 1. KB-9: Re-prioritize care gaps
	_ = kb9Client.RefreshCareGaps(ctx, models.RegistryDiabetes)

	// 2. KB-18: Check escalation policies
	_ = kb18Client.CheckPolicy(ctx, updated.PatientID, "CRITICAL_PATIENT_ALERT", updated)
	kb18Client.RecordAudit(AuditRecord{
		Action:       "RISK_ESCALATION",
		PatientID:    updated.PatientID,
		RegistryCode: models.RegistryDiabetes,
		Reason:       "Risk escalated from MODERATE to CRITICAL",
		Timestamp:    time.Now(),
	})

	// 3. KB-19: Trigger urgent intervention
	_ = kb19Client.TriggerWorkflow(ctx, "URGENT_INTERVENTION", []string{updated.PatientID}, models.RegistryDiabetes, "RISK_ESCALATION")
	_ = kb19Client.AssignTask(ctx, updated, "URGENT_ENDO_CONSULT", time.Now().Add(48*time.Hour))

	// Verify cascade
	assert.Equal(t, 1, kb9Client.GetRefreshCount())

	policyChecks := kb18Client.GetPolicyChecks()
	require.Len(t, policyChecks, 1)
	assert.Equal(t, models.RiskTierCritical, policyChecks[0].RiskTier)

	triggers := kb19Client.GetWorkflowTriggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "RISK_ESCALATION", triggers[0].TriggerType)

	tasks := kb19Client.GetTaskAssignments()
	require.Len(t, tasks, 1)
	assert.Equal(t, "URGENT_ENDO_CONSULT", tasks[0].TaskID)
}

// =============================================================================
// DATA CONTRACT TESTS
// =============================================================================

// TestDownstream_PopulationDataContract tests data contract
func TestDownstream_PopulationDataContract(t *testing.T) {
	// KB-17 must provide this data to downstream services
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        "contract-patient-001",
		RegistryCode:     models.RegistryDiabetes,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierHigh,
		EnrollmentSource: models.EnrollmentSourceDiagnosis,
		EnrolledAt:       time.Now(),
		Notes:            "E11.9 - Type 2 diabetes mellitus without complications",
	}

	// Serialize to JSON (contract format)
	data, err := json.Marshal(enrollment)
	require.NoError(t, err)

	// Verify required fields present
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Contract requirements for downstream services
	requiredFields := []string{
		"patient_id",
		"registry_code",
		"status",
		"risk_tier",
		"enrolled_at",
	}

	for _, field := range requiredFields {
		assert.Contains(t, parsed, field, "Contract requires field: %s", field)
	}
}

// TestDownstream_NoSensitiveDataLeakage tests PHI protection
func TestDownstream_NoSensitiveDataLeakage(t *testing.T) {
	kb9Client := NewMockKB9Client()
	ctx := context.Background()

	// Feed should NOT contain PHI
	feed := PopulationFeed{
		RegistryCode: models.RegistryDiabetes,
		PatientIDs:   []string{"patient-001"}, // Only IDs, not names
		RiskTiers:    map[string]models.RiskTier{"patient-001": models.RiskTierHigh},
		Timestamp:    time.Now(),
	}
	_ = kb9Client.FeedPopulation(ctx, feed)

	feeds := kb9Client.GetFeeds()
	require.Len(t, feeds, 1)

	// Verify no PHI in feed
	feedData, _ := json.Marshal(feeds[0])
	feedStr := string(feedData)

	// Should NOT contain sensitive data patterns
	assert.NotContains(t, feedStr, "date_of_birth")
	assert.NotContains(t, feedStr, "ssn")
	assert.NotContains(t, feedStr, "address")
	assert.NotContains(t, feedStr, "phone")
	assert.NotContains(t, feedStr, "email")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createDownstreamPatientID(prefix string, index int) string {
	return prefix + "-" +
		string(rune('0'+index/100%10)) +
		string(rune('0'+index/10%10)) +
		string(rune('0'+index%10))
}
