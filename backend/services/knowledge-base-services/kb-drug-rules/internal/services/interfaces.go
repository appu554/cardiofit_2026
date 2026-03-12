package services

import (
	"errors"
	"net/http"
	"time"

	"kb-drug-rules/internal/models"
)

// Common errors
var (
	ErrRuleNotFound      = errors.New("rule not found")
	ErrInvalidVersion    = errors.New("invalid version")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrValidationFailed  = errors.New("validation failed")
	ErrApprovalRequired  = errors.New("approval required")
	ErrDuplicateRule     = errors.New("duplicate rule")
)

// RuleService defines the interface for drug rule operations
type RuleService interface {
	// GetDrugRules retrieves drug rules by ID, version, and region
	GetDrugRules(drugID string, version *string, region *string) (*models.DrugRulePack, error)
	
	// SaveRulePack saves a new rule pack to the database
	SaveRulePack(rulePack *models.DrugRulePack) error
	
	// ListVersions lists all versions for a drug
	ListVersions(drugID string) ([]string, error)
	
	// DeleteVersion deletes a specific version of drug rules
	DeleteVersion(drugID, version string) error
	
	// GetLatestVersion gets the latest version for a drug
	GetLatestVersion(drugID string) (string, error)
	
	// UpdateLatestVersion updates the latest version pointer
	UpdateLatestVersion(drugID, version string) error
}

// ValidationService defines the interface for rule validation
type ValidationService interface {
	// ValidateRuleContent validates the content of drug rules
	ValidateRuleContent(content *models.DrugRuleContent, regions []string) (*models.ValidationResponse, error)
	
	// ValidateSchema validates the schema structure
	ValidateSchema(content *models.DrugRuleContent) error
	
	// ValidateExpressions validates mathematical expressions in dose calculations
	ValidateExpressions(doseCalc *models.DoseCalculation) error
	
	// ValidateCrossReferences validates references to other services
	ValidateCrossReferences(content *models.DrugRuleContent) error
	
	// ValidateClinicalSafety validates clinical safety rules
	ValidateClinicalSafety(content *models.DrugRuleContent) error
}

// CacheService defines the interface for caching operations
type CacheService interface {
	// Get retrieves a value from cache
	Get(key string) ([]byte, error)
	
	// Set stores a value in cache with TTL
	Set(key string, value []byte, ttl time.Duration) error
	
	// Delete removes a value from cache
	Delete(key string) error
	
	// InvalidatePattern invalidates all keys matching a pattern
	InvalidatePattern(pattern string) error
	
	// Ping checks if cache is available
	Ping() error
	
	// Close closes the cache connection
	Close() error
	
	// GetStats returns cache statistics
	GetStats() (map[string]interface{}, error)
}

// GovernanceService defines the interface for governance operations
type GovernanceService interface {
	// IsApproved checks if a rule version is approved
	IsApproved(drugID, version string) (bool, error)
	
	// GetApproval gets approval details for a rule version
	GetApproval(drugID, version string) (*ApprovalDetails, error)
	
	// SubmitForApproval submits a rule for approval
	SubmitForApproval(request *ApprovalRequest) (*ApprovalTicket, error)
	
	// ReviewSubmission processes a review for a submission
	ReviewSubmission(ticketID string, review *ReviewSubmission) error
	
	// VerifySignature verifies the digital signature of content
	VerifySignature(content, signature, signer string) (bool, error)
	
	// SignContent signs content with the specified signer
	SignContent(content, signer string) (string, error)
}

// EventService defines the interface for event operations
type EventService interface {
	// EmitRulePackUpdated emits an event when a rule pack is updated
	EmitRulePackUpdated(rulePack *models.DrugRulePack, userInfo map[string]string) error
	
	// EmitRulePackPromoted emits an event when a rule pack is promoted
	EmitRulePackPromoted(drugID, fromVersion, toVersion string, userInfo map[string]string) error
	
	// EmitValidationFailed emits an event when validation fails
	EmitValidationFailed(drugID, version string, errors []string, userInfo map[string]string) error
	
	// EmitGovernanceEvent emits governance-related events
	EmitGovernanceEvent(eventType string, details map[string]interface{}) error
}

// MetricsService defines the interface for metrics collection
type MetricsService interface {
	// IncrementCounter increments a counter metric
	IncrementCounter(name string, labels map[string]string)
	
	// RecordHistogram records a histogram metric
	RecordHistogram(name string, value float64, labels map[string]string)
	
	// RecordGauge records a gauge metric
	RecordGauge(name string, value float64, labels map[string]string)
	
	// RecordHTTPRequest records HTTP request metrics
	RecordHTTPRequest(method, path string, statusCode int, duration time.Duration)
	
	// Handler returns the metrics HTTP handler
	Handler() func(http.ResponseWriter, *http.Request)
}

// AuditService defines the interface for audit logging
type AuditService interface {
	// LogRuleChange logs a rule change event
	LogRuleChange(drugID, version, action string, userInfo map[string]string, details map[string]interface{}) error
	
	// LogGovernanceAction logs a governance action
	LogGovernanceAction(action string, userInfo map[string]string, details map[string]interface{}) error
	
	// LogSecurityEvent logs a security-related event
	LogSecurityEvent(eventType string, userInfo map[string]string, details map[string]interface{}) error
	
	// GetAuditLog retrieves audit log entries
	GetAuditLog(filters map[string]interface{}, limit, offset int) ([]AuditLogEntry, error)
}

// Supporting types

// ApprovalDetails contains details about an approval
type ApprovalDetails struct {
	DrugID               string    `json:"drug_id"`
	Version              string    `json:"version"`
	Status               string    `json:"status"`
	ClinicalReviewer     string    `json:"clinical_reviewer"`
	ClinicalReviewDate   time.Time `json:"clinical_review_date"`
	ClinicalApproved     bool      `json:"clinical_approved"`
	TechnicalReviewer    string    `json:"technical_reviewer"`
	TechnicalReviewDate  time.Time `json:"technical_review_date"`
	TechnicalApproved    bool      `json:"technical_approved"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ApprovalRequest represents a request for approval
type ApprovalRequest struct {
	DrugID      string                 `json:"drug_id"`
	Version     string                 `json:"version"`
	Content     string                 `json:"content"`
	Regions     []string               `json:"regions"`
	Submitter   string                 `json:"submitter"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ApprovalTicket represents an approval ticket
type ApprovalTicket struct {
	ID          string                 `json:"id"`
	DrugID      string                 `json:"drug_id"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Submitter   string                 `json:"submitter"`
	Description string                 `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ReviewSubmission represents a review submission
type ReviewSubmission struct {
	TicketID    string `json:"ticket_id"`
	ReviewType  string `json:"review_type"` // clinical, technical
	Reviewer    string `json:"reviewer"`
	Approved    bool   `json:"approved"`
	Comments    string `json:"comments"`
	ReviewDate  time.Time `json:"review_date"`
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	ID          string                 `json:"id"`
	EntityType  string                 `json:"entity_type"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	UserID      string                 `json:"user_id"`
	UserRole    string                 `json:"user_role"`
	ClientIP    string                 `json:"client_ip"`
	UserAgent   string                 `json:"user_agent"`
	OldValues   map[string]interface{} `json:"old_values,omitempty"`
	NewValues   map[string]interface{} `json:"new_values,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int64   `json:"size"`
	MaxSize     int64   `json:"max_size"`
	Evictions   int64   `json:"evictions"`
	Connections int     `json:"connections"`
}

// ServiceStats represents overall service statistics
type ServiceStats struct {
	TotalRules      int64                  `json:"total_rules"`
	TotalVersions   int64                  `json:"total_versions"`
	ActiveRegions   []string               `json:"active_regions"`
	RecentActivity  []ActivitySummary      `json:"recent_activity"`
	CacheStats      CacheStats             `json:"cache_stats"`
	DatabaseStats   map[string]interface{} `json:"database_stats"`
	Uptime          time.Duration          `json:"uptime"`
	Version         string                 `json:"version"`
}

// ActivitySummary represents recent activity summary
type ActivitySummary struct {
	Action    string    `json:"action"`
	DrugID    string    `json:"drug_id"`
	Version   string    `json:"version"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}
