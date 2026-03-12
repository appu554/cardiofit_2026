// Package cohort provides cohort management for KB-11 Population Health.
//
// Cohort Types:
// - STATIC: Fixed membership, manually maintained
// - DYNAMIC: Rule-based, automatically refreshed based on criteria
// - SNAPSHOT: Point-in-time capture for analysis
package cohort

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Cohort represents a group of patients defined by criteria or membership.
type Cohort struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	Name        string            `json:"name" db:"name"`
	Description string            `json:"description" db:"description"`
	Type        models.CohortType `json:"type" db:"type"`

	// Criteria (for DYNAMIC cohorts)
	Criteria []Criterion `json:"criteria,omitempty" db:"-"`
	CriteriaJSON []byte   `json:"-" db:"criteria"`

	// Membership stats (cached)
	MemberCount   int       `json:"member_count" db:"member_count"`
	LastRefreshed *time.Time `json:"last_refreshed,omitempty" db:"last_refreshed"`

	// For SNAPSHOT cohorts
	SnapshotDate *time.Time `json:"snapshot_date,omitempty" db:"snapshot_date"`
	SourceCohortID *uuid.UUID `json:"source_cohort_id,omitempty" db:"source_cohort_id"`

	// Metadata
	CreatedBy string    `json:"created_by" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	IsActive  bool      `json:"is_active" db:"is_active"`
}

// Criterion represents a single criterion for cohort membership.
type Criterion struct {
	ID       uuid.UUID              `json:"id"`
	Field    string                 `json:"field"`    // e.g., "risk_tier", "age", "attributed_pcp"
	Operator models.CriteriaOperator `json:"operator"` // e.g., "eq", "gt", "in"
	Value    interface{}            `json:"value"`    // The comparison value
	Logic    string                 `json:"logic,omitempty"` // "AND" or "OR" for combining with next criterion
}

// CohortMember represents a patient's membership in a cohort.
type CohortMember struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	CohortID  uuid.UUID  `json:"cohort_id" db:"cohort_id"`
	PatientID uuid.UUID  `json:"patient_id" db:"patient_id"`
	FHIRPatientID string `json:"fhir_patient_id" db:"fhir_patient_id"`
	JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
	RemovedAt *time.Time `json:"removed_at,omitempty" db:"removed_at"`
	IsActive  bool       `json:"is_active" db:"is_active"`
	// Snapshot of patient data at join time (for SNAPSHOT cohorts)
	SnapshotData []byte `json:"snapshot_data,omitempty" db:"snapshot_data"`
}

// CohortRefreshResult represents the result of refreshing a dynamic cohort.
type CohortRefreshResult struct {
	CohortID      uuid.UUID     `json:"cohort_id"`
	CohortName    string        `json:"cohort_name"`
	PreviousCount int           `json:"previous_count"`
	NewCount      int           `json:"new_count"`
	Added         int           `json:"added"`
	Removed       int           `json:"removed"`
	Duration      time.Duration `json:"duration"`
	RefreshedAt   time.Time     `json:"refreshed_at"`
}

// CohortStats provides statistics about a cohort.
type CohortStats struct {
	CohortID       uuid.UUID          `json:"cohort_id"`
	CohortName     string             `json:"cohort_name"`
	MemberCount    int                `json:"member_count"`
	RiskDistribution map[models.RiskTier]int `json:"risk_distribution"`
	AverageRiskScore float64          `json:"average_risk_score"`
	HighRiskCount  int                `json:"high_risk_count"`
	ByPractice     map[string]int     `json:"by_practice,omitempty"`
	ByPCP          map[string]int     `json:"by_pcp,omitempty"`
	CalculatedAt   time.Time          `json:"calculated_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Constructors
// ──────────────────────────────────────────────────────────────────────────────

// NewStaticCohort creates a new static cohort.
func NewStaticCohort(name, description, createdBy string) *Cohort {
	now := time.Now()
	return &Cohort{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Type:        models.CohortTypeStatic,
		MemberCount: 0,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsActive:    true,
	}
}

// NewDynamicCohort creates a new dynamic cohort with criteria.
func NewDynamicCohort(name, description, createdBy string, criteria []Criterion) *Cohort {
	now := time.Now()
	cohort := &Cohort{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Type:        models.CohortTypeDynamic,
		Criteria:    criteria,
		MemberCount: 0,
		CreatedBy:   createdBy,
		CreatedAt:   now,
		UpdatedAt:   now,
		IsActive:    true,
	}

	// Serialize criteria to JSON
	if len(criteria) > 0 {
		cohort.CriteriaJSON, _ = json.Marshal(criteria)
	}

	return cohort
}

// NewSnapshotCohort creates a snapshot of an existing cohort.
func NewSnapshotCohort(sourceCohort *Cohort, createdBy string) *Cohort {
	now := time.Now()
	return &Cohort{
		ID:             uuid.New(),
		Name:           sourceCohort.Name + " - Snapshot " + now.Format("2006-01-02"),
		Description:    "Snapshot of " + sourceCohort.Name,
		Type:           models.CohortTypeSnapshot,
		SnapshotDate:   &now,
		SourceCohortID: &sourceCohort.ID,
		MemberCount:    0,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsActive:       true,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Methods
// ──────────────────────────────────────────────────────────────────────────────

// LoadCriteria deserializes criteria from JSON.
func (c *Cohort) LoadCriteria() error {
	if len(c.CriteriaJSON) == 0 {
		return nil
	}
	return json.Unmarshal(c.CriteriaJSON, &c.Criteria)
}

// SaveCriteria serializes criteria to JSON.
func (c *Cohort) SaveCriteria() error {
	if len(c.Criteria) == 0 {
		c.CriteriaJSON = nil
		return nil
	}
	var err error
	c.CriteriaJSON, err = json.Marshal(c.Criteria)
	return err
}

// IsDynamic returns true if this is a dynamic cohort.
func (c *Cohort) IsDynamic() bool {
	return c.Type == models.CohortTypeDynamic
}

// IsStatic returns true if this is a static cohort.
func (c *Cohort) IsStatic() bool {
	return c.Type == models.CohortTypeStatic
}

// IsSnapshot returns true if this is a snapshot cohort.
func (c *Cohort) IsSnapshot() bool {
	return c.Type == models.CohortTypeSnapshot
}

// NeedsRefresh returns true if a dynamic cohort needs to be refreshed.
func (c *Cohort) NeedsRefresh(refreshInterval time.Duration) bool {
	if !c.IsDynamic() {
		return false
	}
	if c.LastRefreshed == nil {
		return true
	}
	return time.Since(*c.LastRefreshed) > refreshInterval
}

// ──────────────────────────────────────────────────────────────────────────────
// Criterion Builder (Fluent API)
// ──────────────────────────────────────────────────────────────────────────────

// CriterionBuilder helps build criteria with a fluent API.
type CriterionBuilder struct {
	criteria []Criterion
}

// NewCriterionBuilder creates a new criterion builder.
func NewCriterionBuilder() *CriterionBuilder {
	return &CriterionBuilder{
		criteria: []Criterion{},
	}
}

// Where adds a criterion.
func (b *CriterionBuilder) Where(field string, op models.CriteriaOperator, value interface{}) *CriterionBuilder {
	b.criteria = append(b.criteria, Criterion{
		ID:       uuid.New(),
		Field:    field,
		Operator: op,
		Value:    value,
		Logic:    "AND",
	})
	return b
}

// And adds an AND criterion.
func (b *CriterionBuilder) And(field string, op models.CriteriaOperator, value interface{}) *CriterionBuilder {
	return b.Where(field, op, value)
}

// Or adds an OR criterion.
func (b *CriterionBuilder) Or(field string, op models.CriteriaOperator, value interface{}) *CriterionBuilder {
	if len(b.criteria) > 0 {
		b.criteria[len(b.criteria)-1].Logic = "OR"
	}
	b.criteria = append(b.criteria, Criterion{
		ID:       uuid.New(),
		Field:    field,
		Operator: op,
		Value:    value,
		Logic:    "AND",
	})
	return b
}

// Build returns the built criteria.
func (b *CriterionBuilder) Build() []Criterion {
	return b.criteria
}

// ──────────────────────────────────────────────────────────────────────────────
// Predefined Cohort Criteria
// ──────────────────────────────────────────────────────────────────────────────

// HighRiskCriteria returns criteria for high-risk patients.
func HighRiskCriteria() []Criterion {
	return NewCriterionBuilder().
		Where("current_risk_tier", models.OpIn, []string{"HIGH", "VERY_HIGH"}).
		Build()
}

// RisingRiskCriteria returns criteria for rising-risk patients.
func RisingRiskCriteria() []Criterion {
	return NewCriterionBuilder().
		Where("current_risk_tier", models.OpEquals, "RISING").
		Build()
}

// CareGapCriteria returns criteria for patients with care gaps.
func CareGapCriteria(minGaps int) []Criterion {
	return NewCriterionBuilder().
		Where("care_gap_count", models.OpGreaterEq, minGaps).
		Build()
}

// PCPCriteria returns criteria for patients attributed to a specific PCP.
func PCPCriteria(pcp string) []Criterion {
	return NewCriterionBuilder().
		Where("attributed_pcp", models.OpEquals, pcp).
		Build()
}

// PracticeCriteria returns criteria for patients in a specific practice.
func PracticeCriteria(practice string) []Criterion {
	return NewCriterionBuilder().
		Where("attributed_practice", models.OpEquals, practice).
		Build()
}
