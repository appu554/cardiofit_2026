package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Client represents a PostgreSQL client for terminology queries
type Client struct {
	db     *sql.DB
	logger *logrus.Logger
}

// Concept represents a terminology concept
type Concept struct {
	ID              string            `json:"id"`
	Code            string            `json:"code"`
	System          string            `json:"system"`
	Display         string            `json:"display"`
	Definition      string            `json:"definition,omitempty"`
	Status          string            `json:"status"`
	LastUpdated     time.Time         `json:"last_updated"`
	Properties      map[string]string `json:"properties,omitempty"`
	Designations    []Designation     `json:"designations,omitempty"`
	Relationships   []Relationship    `json:"relationships,omitempty"`
}

// Designation represents a concept designation
type Designation struct {
	Language string `json:"language"`
	Use      string `json:"use"`
	Value    string `json:"value"`
}

// Relationship represents a concept relationship
type Relationship struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Code   string `json:"code"`
}

// ConceptMapping represents a cross-terminology mapping
type ConceptMapping struct {
	ID         string    `json:"id"`
	FromSystem string    `json:"from_system"`
	FromCode   string    `json:"from_code"`
	ToSystem   string    `json:"to_system"`
	ToCode     string    `json:"to_code"`
	Equivalence string   `json:"equivalence"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SearchResult represents a search result
type SearchResult struct {
	Concepts    []Concept `json:"concepts"`
	TotalCount  int       `json:"total_count"`
	SearchQuery string    `json:"search_query"`
	SearchTime  time.Time `json:"search_time"`
}

// NewClient creates a new PostgreSQL client
func NewClient(databaseURL string) (*Client, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &Client{
		db:     db,
		logger: logger,
	}, nil
}

// GetConcept retrieves a concept by system and code
func (c *Client) GetConcept(ctx context.Context, system, code string) (*Concept, error) {
	start := time.Now()
	defer func() {
		c.logger.WithFields(logrus.Fields{
			"system":   system,
			"code":     code,
			"duration": time.Since(start),
		}).Debug("GetConcept completed")
	}()

	query := `
		SELECT 
			id, code, system, display, definition, status, last_updated
		FROM concepts 
		WHERE system = $1 AND code = $2 AND status = 'active'
	`

	row := c.db.QueryRowContext(ctx, query, system, code)

	var concept Concept
	var definition sql.NullString

	err := row.Scan(
		&concept.ID,
		&concept.Code,
		&concept.System,
		&concept.Display,
		&definition,
		&concept.Status,
		&concept.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Concept not found
		}
		return nil, fmt.Errorf("failed to scan concept: %w", err)
	}

	if definition.Valid {
		concept.Definition = definition.String
	}

	// Load designations
	designations, err := c.getDesignations(ctx, concept.ID)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load designations")
	} else {
		concept.Designations = designations
	}

	// Load relationships
	relationships, err := c.getConceptRelationships(ctx, concept.ID)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to load relationships")
	} else {
		concept.Relationships = relationships
	}

	return &concept, nil
}

// GetMapping retrieves a cross-terminology mapping
func (c *Client) GetMapping(ctx context.Context, fromSystem, fromCode, toSystem string) (*ConceptMapping, error) {
	start := time.Now()
	defer func() {
		c.logger.WithFields(logrus.Fields{
			"from_system": fromSystem,
			"from_code":   fromCode,
			"to_system":   toSystem,
			"duration":    time.Since(start),
		}).Debug("GetMapping completed")
	}()

	query := `
		SELECT 
			id, from_system, from_code, to_system, to_code, equivalence, created_at, updated_at
		FROM concept_mappings 
		WHERE from_system = $1 AND from_code = $2 AND to_system = $3
		ORDER BY updated_at DESC
		LIMIT 1
	`

	row := c.db.QueryRowContext(ctx, query, fromSystem, fromCode, toSystem)

	var mapping ConceptMapping
	err := row.Scan(
		&mapping.ID,
		&mapping.FromSystem,
		&mapping.FromCode,
		&mapping.ToSystem,
		&mapping.ToCode,
		&mapping.Equivalence,
		&mapping.CreatedAt,
		&mapping.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Mapping not found
		}
		return nil, fmt.Errorf("failed to scan mapping: %w", err)
	}

	return &mapping, nil
}

// GetRelationships retrieves concept relationships
func (c *Client) GetRelationships(ctx context.Context, system, code, relType string) ([]Relationship, error) {
	start := time.Now()
	defer func() {
		c.logger.WithFields(logrus.Fields{
			"system":   system,
			"code":     code,
			"rel_type": relType,
			"duration": time.Since(start),
		}).Debug("GetRelationships completed")
	}()

	// First get the concept ID
	conceptID, err := c.getConceptID(ctx, system, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get concept ID: %w", err)
	}

	if conceptID == "" {
		return []Relationship{}, nil
	}

	return c.getConceptRelationships(ctx, conceptID)
}

// SearchConcepts performs fuzzy text search on concepts
func (c *Client) SearchConcepts(ctx context.Context, query, system string, limit int) (*SearchResult, error) {
	start := time.Now()
	defer func() {
		c.logger.WithFields(logrus.Fields{
			"query":    query,
			"system":   system,
			"limit":    limit,
			"duration": time.Since(start),
		}).Debug("SearchConcepts completed")
	}()

	// Build search query with full-text search
	sqlQuery := `
		SELECT 
			id, code, system, display, definition, status, last_updated,
			ts_rank(search_vector, plainto_tsquery($1)) as rank
		FROM concepts 
		WHERE search_vector @@ plainto_tsquery($1)
	`

	args := []interface{}{query}
	argIndex := 2

	// Add system filter if specified
	if system != "all" && system != "" {
		sqlQuery += fmt.Sprintf(" AND system = $%d", argIndex)
		args = append(args, system)
		argIndex++
	}

	sqlQuery += " AND status = 'active' ORDER BY rank DESC"

	// Add limit
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := c.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var concepts []Concept
	for rows.Next() {
		var concept Concept
		var definition sql.NullString
		var rank float64

		err := rows.Scan(
			&concept.ID,
			&concept.Code,
			&concept.System,
			&concept.Display,
			&definition,
			&concept.Status,
			&concept.LastUpdated,
			&rank,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}

		if definition.Valid {
			concept.Definition = definition.String
		}

		concepts = append(concepts, concept)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	// Get total count for pagination
	totalCount, err := c.getSearchCount(ctx, query, system)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to get search count")
		totalCount = len(concepts)
	}

	return &SearchResult{
		Concepts:    concepts,
		TotalCount:  totalCount,
		SearchQuery: query,
		SearchTime:  time.Now(),
	}, nil
}

// Helper methods

func (c *Client) getConceptID(ctx context.Context, system, code string) (string, error) {
	query := "SELECT id FROM concepts WHERE system = $1 AND code = $2 AND status = 'active'"
	row := c.db.QueryRowContext(ctx, query, system, code)

	var id string
	err := row.Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

func (c *Client) getDesignations(ctx context.Context, conceptID string) ([]Designation, error) {
	query := `
		SELECT language, use_code, value 
		FROM concept_designations 
		WHERE concept_id = $1
		ORDER BY language, use_code
	`

	rows, err := c.db.QueryContext(ctx, query, conceptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designations []Designation
	for rows.Next() {
		var d Designation
		err := rows.Scan(&d.Language, &d.Use, &d.Value)
		if err != nil {
			return nil, err
		}
		designations = append(designations, d)
	}

	return designations, rows.Err()
}

func (c *Client) getConceptRelationships(ctx context.Context, conceptID string) ([]Relationship, error) {
	query := `
		SELECT cr.relationship_type, tc.code, tc.system
		FROM concept_relationships cr
		JOIN concepts tc ON cr.target_concept_id = tc.id
		WHERE cr.source_concept_id = $1
		ORDER BY cr.relationship_type, tc.code
	`

	rows, err := c.db.QueryContext(ctx, query, conceptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var relationships []Relationship
	for rows.Next() {
		var r Relationship
		err := rows.Scan(&r.Type, &r.Code, &r.Target)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}

	return relationships, rows.Err()
}

func (c *Client) getSearchCount(ctx context.Context, query, system string) (int, error) {
	sqlQuery := "SELECT COUNT(*) FROM concepts WHERE search_vector @@ plainto_tsquery($1)"
	args := []interface{}{query}

	if system != "all" && system != "" {
		sqlQuery += " AND system = $2"
		args = append(args, system)
	}

	sqlQuery += " AND status = 'active'"

	row := c.db.QueryRowContext(ctx, sqlQuery, args...)
	var count int
	err := row.Scan(&count)
	return count, err
}

// GetStats returns database statistics
func (c *Client) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get concept counts by system
	query := `
		SELECT system, COUNT(*) as count
		FROM concepts 
		WHERE status = 'active'
		GROUP BY system
		ORDER BY count DESC
	`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	systemCounts := make(map[string]int)
	totalConcepts := 0

	for rows.Next() {
		var system string
		var count int
		if err := rows.Scan(&system, &count); err != nil {
			return nil, err
		}
		systemCounts[system] = count
		totalConcepts += count
	}

	stats["concept_counts_by_system"] = systemCounts
	stats["total_concepts"] = totalConcepts

	// Get mapping counts
	mappingCountQuery := "SELECT COUNT(*) FROM concept_mappings"
	row := c.db.QueryRowContext(ctx, mappingCountQuery)
	var mappingCount int
	if err := row.Scan(&mappingCount); err != nil {
		c.logger.WithError(err).Warn("Failed to get mapping count")
	} else {
		stats["total_mappings"] = mappingCount
	}

	return stats, nil
}

// Ping tests the database connection
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.db.PingContext(ctx)
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// BeginTx starts a new transaction
func (c *Client) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, nil)
}

// BulkInsertConcepts performs bulk insertion of concepts
func (c *Client) BulkInsertConcepts(ctx context.Context, concepts []Concept) error {
	tx, err := c.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO concepts (id, code, system, display, definition, status, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (system, code) DO UPDATE SET
			display = EXCLUDED.display,
			definition = EXCLUDED.definition,
			status = EXCLUDED.status,
			last_updated = EXCLUDED.last_updated
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, concept := range concepts {
		_, err := stmt.ExecContext(ctx,
			concept.ID,
			concept.Code,
			concept.System,
			concept.Display,
			concept.Definition,
			concept.Status,
			concept.LastUpdated,
		)
		if err != nil {
			return fmt.Errorf("failed to insert concept %s:%s: %w", concept.System, concept.Code, err)
		}
	}

	return tx.Commit()
}