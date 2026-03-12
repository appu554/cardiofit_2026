package repositories

import (
	"context"
	"time"
	"github.com/google/uuid"
	"medication-service-v2/internal/domain/entities"
)

// MedicationRepository defines the interface for medication data operations
type MedicationRepository interface {
	// Create operations
	CreateProposal(ctx context.Context, proposal *entities.MedicationProposal) error
	CreateBatchProposals(ctx context.Context, proposals []*entities.MedicationProposal) error
	
	// Read operations
	GetProposalByID(ctx context.Context, proposalID uuid.UUID) (*entities.MedicationProposal, error)
	GetProposalsByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicationProposal, error)
	GetProposalsByStatus(ctx context.Context, status entities.ProposalStatus, limit, offset int) ([]*entities.MedicationProposal, error)
	GetActiveProposals(ctx context.Context, limit, offset int) ([]*entities.MedicationProposal, error)
	GetExpiredProposals(ctx context.Context, beforeDate time.Time) ([]*entities.MedicationProposal, error)
	
	// Update operations
	UpdateProposal(ctx context.Context, proposal *entities.MedicationProposal) error
	UpdateProposalStatus(ctx context.Context, proposalID uuid.UUID, status entities.ProposalStatus, updatedBy string) error
	ValidateProposal(ctx context.Context, proposalID uuid.UUID, validatedBy string) error
	
	// Delete operations
	DeleteProposal(ctx context.Context, proposalID uuid.UUID) error
	ArchiveExpiredProposals(ctx context.Context, beforeDate time.Time) (int, error)
	
	// Search operations
	SearchProposals(ctx context.Context, criteria SearchCriteria) ([]*entities.MedicationProposal, error)
	GetProposalsByProtocol(ctx context.Context, protocolID string) ([]*entities.MedicationProposal, error)
	GetProposalsRequiringReview(ctx context.Context) ([]*entities.MedicationProposal, error)
	
	// Analytics operations
	GetProposalStatistics(ctx context.Context, timeRange TimeRange) (*ProposalStatistics, error)
	GetPatientMedicationHistory(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicationProposal, error)
}

// RecipeRepository defines the interface for recipe data operations  
type RecipeRepository interface {
	// Create operations
	CreateRecipe(ctx context.Context, recipe *entities.Recipe) error
	CreateRecipeVersion(ctx context.Context, recipe *entities.Recipe) error
	
	// Read operations
	GetRecipeByID(ctx context.Context, recipeID uuid.UUID) (*entities.Recipe, error)
	GetRecipeByProtocolID(ctx context.Context, protocolID string) (*entities.Recipe, error)
	GetRecipesByIndication(ctx context.Context, indication string) ([]*entities.Recipe, error)
	GetActiveRecipes(ctx context.Context) ([]*entities.Recipe, error)
	GetRecipeVersions(ctx context.Context, protocolID string) ([]*entities.Recipe, error)
	GetLatestRecipeVersion(ctx context.Context, protocolID string) (*entities.Recipe, error)
	
	// Update operations
	UpdateRecipe(ctx context.Context, recipe *entities.Recipe) error
	UpdateRecipeStatus(ctx context.Context, recipeID uuid.UUID, status entities.RecipeStatus, updatedBy string) error
	ApproveRecipe(ctx context.Context, recipeID uuid.UUID, approvedBy string, notes string) error
	
	// Delete operations
	ArchiveRecipe(ctx context.Context, recipeID uuid.UUID, archivedBy string) error
	DeleteRecipe(ctx context.Context, recipeID uuid.UUID) error
	
	// Search operations
	SearchRecipes(ctx context.Context, criteria RecipeSearchCriteria) ([]*entities.Recipe, error)
	GetRecipesByEvidence(ctx context.Context, evidenceLevel entities.EvidenceLevel) ([]*entities.Recipe, error)
	
	// Cache operations
	CacheRecipe(ctx context.Context, recipe *entities.Recipe, ttl time.Duration) error
	GetCachedRecipe(ctx context.Context, protocolID string) (*entities.Recipe, error)
	InvalidateRecipeCache(ctx context.Context, protocolID string) error
}

// SnapshotRepository defines the interface for clinical snapshot operations
type SnapshotRepository interface {
	// Create operations
	CreateSnapshot(ctx context.Context, snapshot *entities.ClinicalSnapshot) error
	CreateBatchSnapshots(ctx context.Context, snapshots []*entities.ClinicalSnapshot) error
	
	// Read operations  
	GetSnapshotByID(ctx context.Context, snapshotID uuid.UUID) (*entities.ClinicalSnapshot, error)
	GetSnapshotsByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.ClinicalSnapshot, error)
	GetActiveSnapshots(ctx context.Context, patientID uuid.UUID) ([]*entities.ClinicalSnapshot, error)
	GetSnapshotHistory(ctx context.Context, patientID uuid.UUID, limit int) ([]*entities.ClinicalSnapshot, error)
	GetExpiredSnapshots(ctx context.Context, beforeDate time.Time) ([]*entities.ClinicalSnapshot, error)
	
	// Update operations
	UpdateSnapshot(ctx context.Context, snapshot *entities.ClinicalSnapshot) error
	UpdateSnapshotStatus(ctx context.Context, snapshotID uuid.UUID, status entities.SnapshotStatus) error
	SupersedeSnapshot(ctx context.Context, oldSnapshotID, newSnapshotID uuid.UUID, reason string) error
	
	// Delete operations
	DeleteSnapshot(ctx context.Context, snapshotID uuid.UUID) error
	CleanupExpiredSnapshots(ctx context.Context, beforeDate time.Time) (int, error)
	
	// Validation operations
	ValidateSnapshot(ctx context.Context, snapshotID uuid.UUID, validatedBy string) error
	GetSnapshotValidationResults(ctx context.Context, snapshotID uuid.UUID) (*entities.ValidationResults, error)
	
	// Search operations
	SearchSnapshots(ctx context.Context, criteria SnapshotSearchCriteria) ([]*entities.ClinicalSnapshot, error)
	GetSnapshotsByType(ctx context.Context, snapshotType entities.SnapshotType) ([]*entities.ClinicalSnapshot, error)
	
	// Analytics operations
	GetSnapshotStatistics(ctx context.Context, timeRange TimeRange) (*SnapshotStatistics, error)
	GetDataQualityMetrics(ctx context.Context, patientID uuid.UUID) (*DataQualityMetrics, error)
}

// Search criteria structures
type SearchCriteria struct {
	PatientID     *uuid.UUID                   `json:"patient_id,omitempty"`
	Status        *entities.ProposalStatus     `json:"status,omitempty"`
	Indication    *string                      `json:"indication,omitempty"`
	CreatedAfter  *time.Time                   `json:"created_after,omitempty"`
	CreatedBefore *time.Time                   `json:"created_before,omitempty"`
	CreatedBy     *string                      `json:"created_by,omitempty"`
	DrugName      *string                      `json:"drug_name,omitempty"`
	Limit         int                          `json:"limit"`
	Offset        int                          `json:"offset"`
	SortBy        string                       `json:"sort_by"`
	SortOrder     SortOrder                    `json:"sort_order"`
}

type RecipeSearchCriteria struct {
	ProtocolID    *string                     `json:"protocol_id,omitempty"`
	Name          *string                     `json:"name,omitempty"`
	Indication    *string                     `json:"indication,omitempty"`
	Status        *entities.RecipeStatus      `json:"status,omitempty"`
	EvidenceLevel *entities.EvidenceLevel     `json:"evidence_level,omitempty"`
	CreatedAfter  *time.Time                  `json:"created_after,omitempty"`
	CreatedBefore *time.Time                  `json:"created_before,omitempty"`
	CreatedBy     *string                     `json:"created_by,omitempty"`
	Limit         int                         `json:"limit"`
	Offset        int                         `json:"offset"`
	SortBy        string                      `json:"sort_by"`
	SortOrder     SortOrder                   `json:"sort_order"`
}

type SnapshotSearchCriteria struct {
	PatientID     *uuid.UUID                  `json:"patient_id,omitempty"`
	RecipeID      *uuid.UUID                  `json:"recipe_id,omitempty"`
	SnapshotType  *entities.SnapshotType      `json:"snapshot_type,omitempty"`
	Status        *entities.SnapshotStatus    `json:"status,omitempty"`
	CreatedAfter  *time.Time                  `json:"created_after,omitempty"`
	CreatedBefore *time.Time                  `json:"created_before,omitempty"`
	ValidOnly     bool                        `json:"valid_only"`
	Limit         int                         `json:"limit"`
	Offset        int                         `json:"offset"`
	SortBy        string                      `json:"sort_by"`
	SortOrder     SortOrder                   `json:"sort_order"`
}

type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

type TimeRange struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// Statistics structures
type ProposalStatistics struct {
	TotalProposals      int                            `json:"total_proposals"`
	ProposalsByStatus   map[entities.ProposalStatus]int `json:"proposals_by_status"`
	ProposalsByIndication map[string]int               `json:"proposals_by_indication"`
	AverageProcessingTime time.Duration                `json:"average_processing_time"`
	SuccessRate         float64                        `json:"success_rate"`
	TopDrugs           []DrugStatistic                 `json:"top_drugs"`
	TimeSeriesData     []TimeSeriesPoint               `json:"time_series_data"`
}

type SnapshotStatistics struct {
	TotalSnapshots       int                             `json:"total_snapshots"`
	SnapshotsByType      map[entities.SnapshotType]int   `json:"snapshots_by_type"`
	SnapshotsByStatus    map[entities.SnapshotStatus]int `json:"snapshots_by_status"`
	AverageQualityScore  float64                         `json:"average_quality_score"`
	AverageCreationTime  time.Duration                   `json:"average_creation_time"`
	ExpirationRate       float64                         `json:"expiration_rate"`
	DataFreshnessMetrics *DataFreshnessMetrics           `json:"data_freshness_metrics"`
}

type DataQualityMetrics struct {
	PatientID         uuid.UUID                    `json:"patient_id"`
	OverallScore      float64                      `json:"overall_score"`
	CompletenessScore float64                      `json:"completeness_score"`
	AccuracyScore     float64                      `json:"accuracy_score"`
	ConsistencyScore  float64                      `json:"consistency_score"`
	TimelinessScore   float64                      `json:"timeliness_score"`
	QualityTrend      []QualityTrendPoint          `json:"quality_trend"`
	IssuesByCategory  map[entities.QualityCategory]int `json:"issues_by_category"`
}

type DrugStatistic struct {
	DrugName        string  `json:"drug_name"`
	ProposalCount   int     `json:"proposal_count"`
	SuccessRate     float64 `json:"success_rate"`
	AverageConfidence float64 `json:"average_confidence"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time   `json:"timestamp"`
	Count     int         `json:"count"`
	Value     float64     `json:"value"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

type QualityTrendPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	QualityScore float64   `json:"quality_score"`
	IssueCount   int       `json:"issue_count"`
}

type DataFreshnessMetrics struct {
	AverageFreshness map[string]time.Duration      `json:"average_freshness"`
	FreshnessScores  map[string]float64            `json:"freshness_scores"`
	StaleDataCount   map[string]int                `json:"stale_data_count"`
	RefreshRates     map[string]time.Duration      `json:"refresh_rates"`
}

// Repository error types
type RepositoryError struct {
	Type    RepositoryErrorType `json:"type"`
	Message string              `json:"message"`
	Cause   error               `json:"cause,omitempty"`
}

type RepositoryErrorType string

const (
	RepositoryErrorTypeNotFound        RepositoryErrorType = "not_found"
	RepositoryErrorTypeConflict        RepositoryErrorType = "conflict"
	RepositoryErrorTypeValidation      RepositoryErrorType = "validation"
	RepositoryErrorTypeDatabase        RepositoryErrorType = "database"
	RepositoryErrorTypeTimeout         RepositoryErrorType = "timeout"
	RepositoryErrorTypeUnauthorized    RepositoryErrorType = "unauthorized"
	RepositoryErrorTypeConstraintViolation RepositoryErrorType = "constraint_violation"
)

func (e *RepositoryError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func NewRepositoryError(errorType RepositoryErrorType, message string, cause error) *RepositoryError {
	return &RepositoryError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if the error is a not found error
func IsNotFound(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		return repoErr.Type == RepositoryErrorTypeNotFound
	}
	return false
}

// IsConflict checks if the error is a conflict error
func IsConflict(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		return repoErr.Type == RepositoryErrorTypeConflict
	}
	return false
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	if repoErr, ok := err.(*RepositoryError); ok {
		return repoErr.Type == RepositoryErrorTypeValidation
	}
	return false
}