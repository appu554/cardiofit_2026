package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"

	"medication-service-v2/internal/domain/entities"
)

// RecipeResolverRepository defines repository operations for recipe resolvers
type RecipeResolverRepository interface {
	Save(ctx context.Context, resolver *entities.RecipeResolver) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.RecipeResolver, error)
	GetByRecipeAndPatient(ctx context.Context, recipeID uuid.UUID, patientID string) (*entities.RecipeResolver, error)
	List(ctx context.Context, filters ResolverFilters) ([]*entities.RecipeResolver, error)
	Update(ctx context.Context, resolver *entities.RecipeResolver) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}

// ResolverFilters defines filters for resolver queries
type ResolverFilters struct {
	RecipeID    *uuid.UUID `json:"recipe_id,omitempty"`
	PatientID   string     `json:"patient_id,omitempty"`
	ProtocolID  string     `json:"protocol_id,omitempty"`
	CreatedAfter *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	Limit       int        `json:"limit,omitempty"`
	Offset      int        `json:"offset,omitempty"`
}

// ConditionalRuleRepository defines repository operations for conditional rules
type ConditionalRuleRepository interface {
	Save(ctx context.Context, rule *entities.ConditionalRule) error
	GetByID(ctx context.Context, id uuid.UUID) (*entities.ConditionalRule, error)
	GetByProtocol(ctx context.Context, protocolID string) ([]*entities.ConditionalRule, error)
	List(ctx context.Context, filters RuleFilters) ([]*entities.ConditionalRule, error)
	Update(ctx context.Context, rule *entities.ConditionalRule) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveRulesByPriority(ctx context.Context, protocolID string) ([]*entities.ConditionalRule, error)
}

// RecipeTemplateRepository defines repository operations for recipe templates
type RecipeTemplateRepository interface {
	Save(ctx context.Context, template *RecipeTemplate) error
	GetByID(ctx context.Context, id uuid.UUID) (*RecipeTemplate, error)
	GetByProtocol(ctx context.Context, protocolID string) ([]*RecipeTemplate, error)
	List(ctx context.Context, filters TemplateFilters) ([]*RecipeTemplate, error)
	Update(ctx context.Context, template *RecipeTemplate) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetActiveTemplatesByProtocol(ctx context.Context, protocolID string) ([]*RecipeTemplate, error)
}

// RecipeResolutionHistoryRepository defines repository for resolution audit trail
type RecipeResolutionHistoryRepository interface {
	Save(ctx context.Context, history *ResolutionHistory) error
	GetByResolutionID(ctx context.Context, resolutionID uuid.UUID) (*ResolutionHistory, error)
	GetByPatient(ctx context.Context, patientID string, limit int) ([]*ResolutionHistory, error)
	GetByProvider(ctx context.Context, providerID string, limit int) ([]*ResolutionHistory, error)
	SearchResolutions(ctx context.Context, filters ResolutionHistoryFilters) ([]*ResolutionHistory, error)
	GetResolutionStatistics(ctx context.Context, filters ResolutionStatisticsFilters) (*ResolutionStatistics, error)
}

// ResolutionHistory represents the audit trail for recipe resolutions
type ResolutionHistory struct {
	ID                  uuid.UUID                    `json:"id"`
	ResolutionID        uuid.UUID                    `json:"resolution_id"`
	RecipeID            uuid.UUID                    `json:"recipe_id"`
	PatientID           string                       `json:"patient_id"`
	ProviderID          string                       `json:"provider_id"`
	EncounterID         string                       `json:"encounter_id"`
	ResolutionTimestamp time.Time                    `json:"resolution_timestamp"`
	ProcessingTimeMs    int64                        `json:"processing_time_ms"`
	CacheUsed           bool                         `json:"cache_used"`
	ProtocolID          string                       `json:"protocol_id"`
	FieldsResolved      int                          `json:"fields_resolved"`
	RulesEvaluated      int                          `json:"rules_evaluated"`
	SafetyViolations    int                          `json:"safety_violations"`
	CalculatedDoses     int                          `json:"calculated_doses"`
	ConfidenceScore     float64                      `json:"confidence_score"`
	ContextSnapshot     map[string]interface{}       `json:"context_snapshot"`
	Resolution          *entities.RecipeResolution   `json:"resolution"`
	AuditTrail          []AuditEvent                 `json:"audit_trail"`
	CreatedAt           time.Time                    `json:"created_at"`
}

// AuditEvent represents an audit event in the resolution process
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Phase       string                 `json:"phase"`
	Action      string                 `json:"action"`
	Component   string                 `json:"component"`
	Details     map[string]interface{} `json:"details"`
	Duration    time.Duration          `json:"duration"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
}

// ResolutionHistoryFilters defines filters for resolution history queries
type ResolutionHistoryFilters struct {
	PatientID           string     `json:"patient_id,omitempty"`
	ProviderID          string     `json:"provider_id,omitempty"`
	RecipeID            *uuid.UUID `json:"recipe_id,omitempty"`
	ProtocolID          string     `json:"protocol_id,omitempty"`
	EncounterID         string     `json:"encounter_id,omitempty"`
	StartDate           *time.Time `json:"start_date,omitempty"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	MinProcessingTime   *int64     `json:"min_processing_time,omitempty"`
	MaxProcessingTime   *int64     `json:"max_processing_time,omitempty"`
	MinConfidenceScore  *float64   `json:"min_confidence_score,omitempty"`
	MaxConfidenceScore  *float64   `json:"max_confidence_score,omitempty"`
	HasSafetyViolations *bool      `json:"has_safety_violations,omitempty"`
	CacheUsed           *bool      `json:"cache_used,omitempty"`
	Limit               int        `json:"limit,omitempty"`
	Offset              int        `json:"offset,omitempty"`
}

// ResolutionStatisticsFilters defines filters for statistics queries
type ResolutionStatisticsFilters struct {
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	ProtocolID   string     `json:"protocol_id,omitempty"`
	ProviderID   string     `json:"provider_id,omitempty"`
	GroupBy      string     `json:"group_by,omitempty"` // day, week, month, protocol, provider
}

// ResolutionStatistics represents aggregated statistics for resolutions
type ResolutionStatistics struct {
	TotalResolutions       int64         `json:"total_resolutions"`
	SuccessfulResolutions  int64         `json:"successful_resolutions"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	MedianProcessingTime   time.Duration `json:"median_processing_time"`
	P95ProcessingTime      time.Duration `json:"p95_processing_time"`
	P99ProcessingTime      time.Duration `json:"p99_processing_time"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
	AverageConfidenceScore float64       `json:"average_confidence_score"`
	SafetyViolationRate    float64       `json:"safety_violation_rate"`
	PerformanceTargetRate  float64       `json:"performance_target_rate"`
	ProtocolBreakdown      map[string]ProtocolStats `json:"protocol_breakdown"`
	ProviderBreakdown      map[string]ProviderStats `json:"provider_breakdown"`
	TimeSeriesData         []RecipeMetricsPoint     `json:"time_series_data"`
	GeneratedAt            time.Time     `json:"generated_at"`
}

// ProtocolStats represents statistics for a specific protocol
type ProtocolStats struct {
	ProtocolID             string        `json:"protocol_id"`
	ResolutionCount        int64         `json:"resolution_count"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	AverageConfidenceScore float64       `json:"average_confidence_score"`
	SafetyViolationRate    float64       `json:"safety_violation_rate"`
	CacheHitRate           float64       `json:"cache_hit_rate"`
}

// ProviderStats represents statistics for a specific provider
type ProviderStats struct {
	ProviderID             string        `json:"provider_id"`
	ResolutionCount        int64         `json:"resolution_count"`
	UniquePatients         int64         `json:"unique_patients"`
	AverageProcessingTime  time.Duration `json:"average_processing_time"`
	AverageConfidenceScore float64       `json:"average_confidence_score"`
	SafetyViolationRate    float64       `json:"safety_violation_rate"`
}

// RecipeMetricsPoint represents a point in recipe resolver time series data
type RecipeMetricsPoint struct {
	Timestamp             time.Time     `json:"timestamp"`
	ResolutionCount       int64         `json:"resolution_count"`
	AverageProcessingTime time.Duration `json:"average_processing_time"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	SafetyViolationRate   float64       `json:"safety_violation_rate"`
}

// FieldResolutionCacheRepository defines caching operations for field resolution
type FieldResolutionCacheRepository interface {
	GetResolvedField(ctx context.Context, key string) (*entities.ResolvedField, error)
	SetResolvedField(ctx context.Context, key string, field *entities.ResolvedField, ttl time.Duration) error
	InvalidateFieldsByPattern(ctx context.Context, pattern string) error
	GetCacheStatistics(ctx context.Context) (*CacheStatistics, error)
}

// CacheStatistics represents cache performance statistics
type CacheStatistics struct {
	TotalRequests   int64     `json:"total_requests"`
	CacheHits       int64     `json:"cache_hits"`
	CacheMisses     int64     `json:"cache_misses"`
	HitRate         float64   `json:"hit_rate"`
	EntriesCount    int64     `json:"entries_count"`
	MemoryUsage     int64     `json:"memory_usage"`
	LastUpdated     time.Time `json:"last_updated"`
}

// RuleEvaluationCacheRepository defines caching operations for rule evaluations
type RuleEvaluationCacheRepository interface {
	GetRuleEvaluation(ctx context.Context, key string) (*EvaluationResult, error)
	SetRuleEvaluation(ctx context.Context, key string, result *EvaluationResult, ttl time.Duration) error
	InvalidateRulesByProtocol(ctx context.Context, protocolID string) error
	GetEvaluationStatistics(ctx context.Context) (*EvaluationCacheStatistics, error)
}

// EvaluationCacheStatistics represents rule evaluation cache statistics
type EvaluationCacheStatistics struct {
	CacheStatistics
	RulesEvaluated       int64   `json:"rules_evaluated"`
	UniqueRules          int64   `json:"unique_rules"`
	AverageEvaluationTime time.Duration `json:"average_evaluation_time"`
	ProtocolBreakdown    map[string]int64 `json:"protocol_breakdown"`
}

// ProtocolDataRepository defines operations for protocol-specific data storage
type ProtocolDataRepository interface {
	GetProtocolData(ctx context.Context, protocolID, dataType string) (interface{}, error)
	SetProtocolData(ctx context.Context, protocolID, dataType string, data interface{}) error
	ListProtocolDataTypes(ctx context.Context, protocolID string) ([]string, error)
	DeleteProtocolData(ctx context.Context, protocolID, dataType string) error
	GetProtocolMetadata(ctx context.Context, protocolID string) (*ProtocolMetadata, error)
}

// ProtocolMetadata represents metadata about a protocol
type ProtocolMetadata struct {
	ProtocolID    string            `json:"protocol_id"`
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description"`
	Category      string            `json:"category"`
	DataTypes     []string          `json:"data_types"`
	LastUpdated   time.Time         `json:"last_updated"`
	IsActive      bool              `json:"is_active"`
	Configuration map[string]interface{} `json:"configuration"`
}

// RuleFilters defines filters for rule queries
type RuleFilters struct {
	ProtocolID string `json:"protocol_id,omitempty"`
	Active     *bool  `json:"active,omitempty"`
	Priority   *int   `json:"priority,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// TemplateFilters defines filters for template queries
type TemplateFilters struct {
	ProtocolID string `json:"protocol_id,omitempty"`
	Active     *bool  `json:"active,omitempty"`
	Category   string `json:"category,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// RecipeTemplate represents a recipe template
type RecipeTemplate struct {
	ID           uuid.UUID              `json:"id"`
	ProtocolID   string                 `json:"protocol_id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category"`
	Template     map[string]interface{} `json:"template"`
	IsActive     bool                   `json:"is_active"`
	Priority     int                    `json:"priority"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// EvaluationResult represents the result of a rule evaluation
type EvaluationResult struct {
	RuleID      uuid.UUID              `json:"rule_id"`
	Success     bool                   `json:"success"`
	Result      map[string]interface{} `json:"result"`
	Error       string                 `json:"error,omitempty"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
}