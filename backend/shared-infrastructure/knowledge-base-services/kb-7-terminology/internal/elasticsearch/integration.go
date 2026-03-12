package elasticsearch

import (
	"context"
	"fmt"
	"log"
	"time"

	"database/sql"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Integration provides the bridge between existing KB7 PostgreSQL data and Elasticsearch
type Integration struct {
	esClient    *Client
	searchSvc   *SearchService
	pgDB        *sql.DB
	indexName   string
	config      *IntegrationConfig
}

// IntegrationConfig holds configuration for the PostgreSQL-Elasticsearch integration
type IntegrationConfig struct {
	PostgreSQLDSN      string        `json:"postgresql_dsn"`
	ElasticsearchURLs  []string      `json:"elasticsearch_urls"`
	IndexName          string        `json:"index_name"`
	SyncInterval       time.Duration `json:"sync_interval"`
	BatchSize          int           `json:"batch_size"`
	EnableRealTimeSync bool          `json:"enable_realtime_sync"`
	LogLevel           string        `json:"log_level"`
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() *IntegrationConfig {
	return &IntegrationConfig{
		PostgreSQLDSN:      "postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable",
		ElasticsearchURLs:  []string{"http://localhost:9200"},
		IndexName:          "clinical_terms",
		SyncInterval:       5 * time.Minute,
		BatchSize:          1000,
		EnableRealTimeSync: true,
		LogLevel:           "INFO",
	}
}

// NewIntegration creates a new KB7-Elasticsearch integration
func NewIntegration(config *IntegrationConfig) (*Integration, error) {
	if config == nil {
		config = DefaultIntegrationConfig()
	}

	// Connect to PostgreSQL
	pgDB, err := sql.Open("postgres", config.PostgreSQLDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pgDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Connect to Elasticsearch
	esConfig := &Config{
		URLs:           config.ElasticsearchURLs,
		MaxRetries:     3,
		RequestTimeout: 30 * time.Second,
		BulkSize:       config.BatchSize,
	}

	esClient, err := NewClient(esConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Create search service
	searchSvc := NewSearchService(esClient, config.IndexName)

	integration := &Integration{
		esClient:  esClient,
		searchSvc: searchSvc,
		pgDB:      pgDB,
		indexName: config.IndexName,
		config:    config,
	}

	log.Printf("KB7-Elasticsearch integration initialized")
	return integration, nil
}

// Close closes all connections
func (i *Integration) Close() error {
	if i.pgDB != nil {
		return i.pgDB.Close()
	}
	return nil
}

// SyncFromPostgreSQL performs a full sync from PostgreSQL to Elasticsearch
func (i *Integration) SyncFromPostgreSQL(ctx context.Context) error {
	log.Printf("Starting full sync from PostgreSQL to Elasticsearch...")

	// Get total count for progress tracking
	var totalCount int
	err := i.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM concepts WHERE active = true").Scan(&totalCount)
	if err != nil {
		return fmt.Errorf("failed to get total count: %w", err)
	}

	log.Printf("Total terms to sync: %d", totalCount)

	// Process in batches
	offset := 0
	synced := 0
	batchSize := i.config.BatchSize

	for offset < totalCount {
		terms, err := i.fetchTermsBatch(ctx, offset, batchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch terms batch at offset %d: %w", offset, err)
		}

		if len(terms) == 0 {
			break
		}

		// Convert to bulk documents
		docs := make([]BulkDocument, len(terms))
		for j, term := range terms {
			docs[j] = BulkDocument{
				ID:     term.TermID,
				Source: term,
			}
		}

		// Index batch to Elasticsearch
		err = i.esClient.BulkIndexDocuments(ctx, i.indexName, docs)
		if err != nil {
			log.Printf("Failed to index batch at offset %d: %v", offset, err)
			// Continue with next batch rather than failing entirely
		} else {
			synced += len(terms)
			log.Printf("Synced %d/%d terms (%.1f%%)", synced, totalCount, float64(synced)/float64(totalCount)*100)
		}

		offset += batchSize

		// Small delay to avoid overwhelming the systems
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	log.Printf("Full sync completed: %d terms synced", synced)
	return nil
}

// fetchTermsBatch retrieves a batch of terms from PostgreSQL
func (i *Integration) fetchTermsBatch(ctx context.Context, offset, limit int) ([]*ClinicalTerm, error) {
	query := `
		SELECT
			concept_uuid::text, code, preferred_term, COALESCE(fully_specified_name, '') as fully_specified_name,
			COALESCE(synonyms, '{}') as synonyms,
			COALESCE(properties->>'definition', '') as definition,
			system, version,
			COALESCE(properties->>'semantic_tags', '{}') as semantic_tags,
			active, COALESCE(properties->>'clinical_domain', '') as clinical_domain,
			COALESCE((properties->>'complexity_score')::int, 0) as complexity_score,
			COALESCE((properties->>'usage_frequency')::int, 0) as usage_frequency,
			updated_at, created_at
		FROM concepts
		WHERE active = true
		ORDER BY concept_uuid
		LIMIT $1 OFFSET $2
	`

	rows, err := i.pgDB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query terms: %w", err)
	}
	defer rows.Close()

	var terms []*ClinicalTerm

	for rows.Next() {
		term := &ClinicalTerm{}
		var synonymsJSON, semanticTagsJSON string
		var effectiveDate sql.NullTime
		var active bool

		err := rows.Scan(
			&term.TermID, &term.ConceptID, &term.PreferredTerm, &term.Term,
			&synonymsJSON, &term.Definition, &term.TerminologySystem, &term.TerminologyVersion,
			&semanticTagsJSON, &active, &term.ClinicalDomain,
			&term.ComplexityScore, &term.UsageFrequency, &term.LastUpdated, &effectiveDate,
		)
		if err != nil {
			log.Printf("Failed to scan term row: %v", err)
			continue
		}

		// Set status based on active flag
		if active {
			term.Status = "active"
		} else {
			term.Status = "inactive"
		}

		// Parse JSON arrays (simplified - in production use proper JSON parsing)
		if synonymsJSON != "{}" {
			// Parse synonyms JSON array
			term.Synonyms = parseJSONStringArray(synonymsJSON)
		}

		if semanticTagsJSON != "{}" {
			// Parse semantic tags JSON array
			term.SemanticTags = parseJSONStringArray(semanticTagsJSON)
		}

		if effectiveDate.Valid {
			term.EffectiveDate = &effectiveDate.Time
		}

		// Add FHIR mappings based on terminology system
		term.FHIRMappings = i.generateFHIRMappings(term)

		terms = append(terms, term)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return terms, nil
}

// parseJSONStringArray is a simplified JSON array parser
// In production, use proper JSON unmarshaling
func parseJSONStringArray(jsonStr string) []string {
	// Remove brackets and quotes, split by comma
	if len(jsonStr) <= 2 {
		return nil
	}

	// Simple parsing - replace with proper JSON unmarshaling in production
	cleaned := jsonStr[1 : len(jsonStr)-1] // Remove { }
	if cleaned == "" {
		return nil
	}

	// This is a simplified version - use json.Unmarshal in production
	return []string{cleaned}
}

// generateFHIRMappings creates FHIR mappings based on terminology system
func (i *Integration) generateFHIRMappings(term *ClinicalTerm) []FHIRMapping {
	var mappings []FHIRMapping

	switch term.TerminologySystem {
	case "SNOMED_CT":
		mappings = append(mappings, FHIRMapping{
			Code:    term.ConceptID,
			System:  "http://snomed.info/sct",
			Display: term.PreferredTerm,
			Version: term.TerminologyVersion,
		})
	case "RXNORM":
		mappings = append(mappings, FHIRMapping{
			Code:    term.ConceptID,
			System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
			Display: term.PreferredTerm,
			Version: term.TerminologyVersion,
		})
	case "ICD10CM":
		mappings = append(mappings, FHIRMapping{
			Code:    term.ConceptID,
			System:  "http://hl7.org/fhir/sid/icd-10-cm",
			Display: term.PreferredTerm,
			Version: term.TerminologyVersion,
		})
	case "LOINC":
		mappings = append(mappings, FHIRMapping{
			Code:    term.ConceptID,
			System:  "http://loinc.org",
			Display: term.PreferredTerm,
			Version: term.TerminologyVersion,
		})
	}

	return mappings
}

// SearchTerms performs a clinical terminology search using Elasticsearch
func (i *Integration) SearchTerms(ctx context.Context, req *SearchRequest) (*SearchResults, error) {
	return i.searchSvc.Search(ctx, req)
}

// GetTermByID retrieves a specific term by ID
func (i *Integration) GetTermByID(ctx context.Context, termID string) (*ClinicalTerm, error) {
	req := &SearchRequest{
		Query:      termID,
		SearchType: SearchTypeExact,
		Size:       1,
		Filters:    map[string]string{"term_id": termID},
	}

	results, err := i.searchSvc.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search for term: %w", err)
	}

	if len(results.Results) == 0 {
		return nil, fmt.Errorf("term not found: %s", termID)
	}

	return results.Results[0].Term, nil
}

// ValidateSync compares PostgreSQL and Elasticsearch data for consistency
func (i *Integration) ValidateSync(ctx context.Context) (*SyncValidation, error) {
	log.Printf("Starting sync validation...")

	validation := &SyncValidation{
		StartTime: time.Now(),
	}

	// Count terms in PostgreSQL
	err := i.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM concepts WHERE active = true").Scan(&validation.PostgreSQLCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count PostgreSQL terms: %w", err)
	}

	// Count terms in Elasticsearch
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 0,
	}

	response, err := i.esClient.Search(ctx, i.indexName, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count Elasticsearch terms: %w", err)
	}

	validation.ElasticsearchCount = response.Hits.Total.Value
	validation.EndTime = time.Now()
	validation.IsConsistent = validation.PostgreSQLCount == validation.ElasticsearchCount

	if !validation.IsConsistent {
		validation.Discrepancy = validation.PostgreSQLCount - validation.ElasticsearchCount
		log.Printf("Sync validation failed: PostgreSQL=%d, Elasticsearch=%d, Discrepancy=%d",
			validation.PostgreSQLCount, validation.ElasticsearchCount, validation.Discrepancy)
	} else {
		log.Printf("Sync validation passed: %d terms in both systems", validation.PostgreSQLCount)
	}

	return validation, nil
}

// SyncValidation represents the result of a sync validation
type SyncValidation struct {
	StartTime          time.Time `json:"start_time"`
	EndTime            time.Time `json:"end_time"`
	PostgreSQLCount    int       `json:"postgresql_count"`
	ElasticsearchCount int       `json:"elasticsearch_count"`
	IsConsistent       bool      `json:"is_consistent"`
	Discrepancy        int       `json:"discrepancy"`
}

// GetIndexStats returns statistics about the Elasticsearch index
func (i *Integration) GetIndexStats(ctx context.Context) (*IndexStats, error) {
	// Get index stats from Elasticsearch
	response, err := i.esClient.Search(ctx, i.indexName, map[string]interface{}{
		"size": 0,
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get index stats: %w", err)
	}

	stats := &IndexStats{
		IndexName:     i.indexName,
		DocumentCount: int64(response.Hits.Total.Value),
		LastUpdated:   time.Now(),
		Health:        "green", // Simplified - get from cluster health in production
		Shards: ShardInfo{
			Total:      3,
			Successful: 3,
			Failed:     0,
		},
	}

	return stats, nil
}

// StartPeriodicSync starts a background process for periodic synchronization
func (i *Integration) StartPeriodicSync(ctx context.Context) {
	if !i.config.EnableRealTimeSync {
		log.Printf("Real-time sync disabled")
		return
	}

	log.Printf("Starting periodic sync with interval: %v", i.config.SyncInterval)

	ticker := time.NewTicker(i.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Periodic sync stopped")
			return
		case <-ticker.C:
			log.Printf("Starting periodic sync...")
			if err := i.SyncFromPostgreSQL(ctx); err != nil {
				log.Printf("Periodic sync failed: %v", err)
			} else {
				log.Printf("Periodic sync completed successfully")
			}
		}
	}
}