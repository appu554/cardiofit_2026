// Package adapters provides ingestion adapters for regulatory data sources.
// These adapters are 70% reusable across all Knowledge Bases.
package adapters

import (
	"context"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// ADAPTER INTERFACE
// =============================================================================

// Adapter defines the interface for regulatory data source adapters.
// Each KB implements specific adapters based on their data sources.
type Adapter interface {
	// Name returns the adapter name (e.g., "FDA_DAILYMED", "TGA_PI")
	Name() string

	// Authority returns the regulatory authority (FDA, TGA, CMS, etc.)
	Authority() models.Authority

	// SupportedKBs returns which KBs this adapter supports
	SupportedKBs() []models.KB

	// FetchUpdates retrieves items updated since the given timestamp
	FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error)

	// Transform converts raw data to KnowledgeItem
	Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error)

	// Validate performs adapter-specific validation
	Validate(ctx context.Context, item *models.KnowledgeItem) error
}

// RawItem represents data fetched from external source before transformation.
type RawItem struct {
	ID           string            // External identifier (e.g., FDA SetID)
	Authority    models.Authority  // Source authority
	RawData      []byte            // Raw data (XML, JSON, PDF content)
	ContentType  string            // MIME type
	FetchedAt    time.Time         // When fetched
	SourceURL    string            // Source URL
	Metadata     map[string]string // Additional metadata
}

// =============================================================================
// BASE ADAPTER
// =============================================================================

// BaseAdapter provides common functionality for all adapters.
type BaseAdapter struct {
	name         string
	authority    models.Authority
	supportedKBs []models.KB
}

// NewBaseAdapter creates a base adapter.
func NewBaseAdapter(name string, authority models.Authority, kbs []models.KB) *BaseAdapter {
	return &BaseAdapter{
		name:         name,
		authority:    authority,
		supportedKBs: kbs,
	}
}

// Name returns the adapter name.
func (b *BaseAdapter) Name() string {
	return b.name
}

// Authority returns the regulatory authority.
func (b *BaseAdapter) Authority() models.Authority {
	return b.authority
}

// SupportedKBs returns which KBs this adapter supports.
func (b *BaseAdapter) SupportedKBs() []models.KB {
	return b.supportedKBs
}

// =============================================================================
// ADAPTER REGISTRY
// =============================================================================

// Registry manages adapter instances.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a new adapter registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds an adapter to the registry.
func (r *Registry) Register(adapter Adapter) {
	r.adapters[adapter.Name()] = adapter
}

// Get retrieves an adapter by name.
func (r *Registry) Get(name string) (Adapter, bool) {
	adapter, ok := r.adapters[name]
	return adapter, ok
}

// GetForAuthority returns all adapters for a given authority.
func (r *Registry) GetForAuthority(authority models.Authority) []Adapter {
	var result []Adapter
	for _, adapter := range r.adapters {
		if adapter.Authority() == authority {
			result = append(result, adapter)
		}
	}
	return result
}

// GetForKB returns all adapters that support a given KB.
func (r *Registry) GetForKB(kb models.KB) []Adapter {
	var result []Adapter
	for _, adapter := range r.adapters {
		for _, supportedKB := range adapter.SupportedKBs() {
			if supportedKB == kb {
				result = append(result, adapter)
				break
			}
		}
	}
	return result
}

// List returns all registered adapter names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// INGESTION JOB
// =============================================================================

// IngestionJob represents a single ingestion run.
type IngestionJob struct {
	ID          string           `json:"id"`
	AdapterName string           `json:"adapter_name"`
	KB          models.KB        `json:"kb"`
	StartedAt   time.Time        `json:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty"`
	Status      IngestionStatus  `json:"status"`
	Stats       IngestionStats   `json:"stats"`
	Errors      []IngestionError `json:"errors,omitempty"`
}

// IngestionStatus represents job status.
type IngestionStatus string

const (
	IngestionStatusPending   IngestionStatus = "PENDING"
	IngestionStatusRunning   IngestionStatus = "RUNNING"
	IngestionStatusCompleted IngestionStatus = "COMPLETED"
	IngestionStatusFailed    IngestionStatus = "FAILED"
)

// IngestionStats contains statistics for an ingestion run.
type IngestionStats struct {
	TotalFetched  int `json:"total_fetched"`
	Transformed   int `json:"transformed"`
	Validated     int `json:"validated"`
	Created       int `json:"created"`
	Updated       int `json:"updated"`
	Skipped       int `json:"skipped"`
	Failed        int `json:"failed"`
}

// IngestionError records an error during ingestion.
type IngestionError struct {
	ItemID    string    `json:"item_id,omitempty"`
	Phase     string    `json:"phase"` // FETCH, TRANSFORM, VALIDATE, STORE
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
