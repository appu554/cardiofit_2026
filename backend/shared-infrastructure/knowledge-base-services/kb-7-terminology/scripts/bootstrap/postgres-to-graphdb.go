package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"kb-7-terminology/internal/database"
	"kb-7-terminology/internal/semantic"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

const (
	defaultBatchSize    = 1000
	defaultLogInterval  = 10000
	bootstrapContext    = "http://cardiofit.ai/bootstrap"
)

type Config struct {
	PostgresURL  string
	GraphDBURL   string
	Repository   string
	BatchSize    int
	LogInterval  int
	DryRun       bool
	StartOffset  int
	MaxConcepts  int
}

type MigrationStats struct {
	TotalConcepts     int64
	MigratedConcepts  int64
	FailedConcepts    int64
	TotalTriples      int64
	StartTime         time.Time
	EndTime           time.Time
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	logger.WithFields(logrus.Fields{
		"postgres_url":  maskPassword(config.PostgresURL),
		"graphdb_url":   config.GraphDBURL,
		"repository":    config.Repository,
		"batch_size":    config.BatchSize,
		"dry_run":       config.DryRun,
	}).Info("Starting PostgreSQL to GraphDB migration")

	// Create context
	ctx := context.Background()

	// Connect to PostgreSQL
	logger.Info("Connecting to PostgreSQL...")
	pgDB, err := database.Connect(config.PostgresURL)
	if err != nil {
		logger.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer pgDB.Close()

	// Connect to GraphDB
	logger.Info("Connecting to GraphDB...")
	graphDB := semantic.NewGraphDBClient(config.GraphDBURL, config.Repository, logger)

	// Test GraphDB connection
	if err := graphDB.HealthCheck(ctx); err != nil {
		logger.Fatalf("GraphDB health check failed: %v", err)
	}
	logger.Info("GraphDB connection successful")

	// Initialize migration stats
	stats := &MigrationStats{
		StartTime: time.Now(),
	}

	// Get total concept count
	totalConcepts, err := getTotalConceptCount(pgDB)
	if err != nil {
		logger.Fatalf("Failed to get total concept count: %v", err)
	}
	stats.TotalConcepts = totalConcepts

	logger.WithFields(logrus.Fields{
		"total_concepts": totalConcepts,
		"estimated_time": estimateMigrationTime(totalConcepts, config.BatchSize),
	}).Info("Migration plan prepared")

	if config.DryRun {
		logger.Info("DRY RUN MODE - No data will be migrated")
		return
	}

	// Perform migration
	if err := migrateConcepts(ctx, pgDB, graphDB, config, stats, logger); err != nil {
		logger.Fatalf("Migration failed: %v", err)
	}

	stats.EndTime = time.Now()

	// Validate migration
	logger.Info("Validating migration...")
	if err := validateMigration(ctx, pgDB, graphDB, stats, logger); err != nil {
		logger.Errorf("Validation failed: %v", err)
	}

	// Print final stats
	printStats(stats, logger)

	logger.Info("Migration completed successfully")
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.PostgresURL, "postgres", getEnv("DATABASE_URL", "postgresql://kb7_user:kb7_password@localhost:5433/kb7_terminology"), "PostgreSQL connection URL")
	flag.StringVar(&config.GraphDBURL, "graphdb", getEnv("GRAPHDB_URL", "http://localhost:7200"), "GraphDB base URL")
	flag.StringVar(&config.Repository, "repo", getEnv("GRAPHDB_REPOSITORY", "kb7-terminology"), "GraphDB repository name")
	flag.IntVar(&config.BatchSize, "batch", defaultBatchSize, "Batch size for migration")
	flag.IntVar(&config.LogInterval, "log-interval", defaultLogInterval, "Log progress every N concepts")
	flag.BoolVar(&config.DryRun, "dry-run", false, "Perform dry run without migrating data")
	flag.IntVar(&config.StartOffset, "start", 0, "Start offset for testing partial migration")
	flag.IntVar(&config.MaxConcepts, "max", 0, "Maximum concepts to migrate (0 = all)")

	flag.Parse()

	return config
}

func migrateConcepts(ctx context.Context, pgDB *sql.DB, graphDB *semantic.GraphDBClient, config *Config, stats *MigrationStats, logger *logrus.Logger) error {
	offset := config.StartOffset
	maxConcepts := config.MaxConcepts
	if maxConcepts == 0 {
		maxConcepts = int(stats.TotalConcepts)
	}

	for offset < maxConcepts {
		batchStart := time.Now()

		// Fetch batch of concepts from PostgreSQL
		concepts, err := fetchConceptBatch(pgDB, offset, config.BatchSize)
		if err != nil {
			return fmt.Errorf("failed to fetch batch at offset %d: %w", offset, err)
		}

		if len(concepts) == 0 {
			break // No more concepts
		}

		// Convert concepts to RDF/Turtle format
		turtleContent, tripleCount := convertToTurtle(concepts)

		// Load to GraphDB
		if err := graphDB.LoadTurtleData(ctx, []byte(turtleContent), bootstrapContext); err != nil {
			logger.WithFields(logrus.Fields{
				"offset": offset,
				"batch_size": len(concepts),
			}).Errorf("Failed to load batch to GraphDB: %v", err)
			stats.FailedConcepts += int64(len(concepts))
			offset += config.BatchSize
			continue
		}

		// Update stats
		stats.MigratedConcepts += int64(len(concepts))
		stats.TotalTriples += int64(tripleCount)

		// Log progress
		if stats.MigratedConcepts%int64(config.LogInterval) == 0 || offset+config.BatchSize >= maxConcepts {
			elapsed := time.Since(stats.StartTime)
			remaining := estimateRemainingTime(stats.MigratedConcepts, stats.TotalConcepts, elapsed)

			logger.WithFields(logrus.Fields{
				"migrated":        stats.MigratedConcepts,
				"total":           stats.TotalConcepts,
				"progress":        fmt.Sprintf("%.2f%%", float64(stats.MigratedConcepts)/float64(stats.TotalConcepts)*100),
				"batch_time":      time.Since(batchStart).Round(time.Millisecond),
				"elapsed":         elapsed.Round(time.Second),
				"remaining_est":   remaining.Round(time.Second),
				"triples":         stats.TotalTriples,
			}).Info("Migration progress")
		}

		offset += config.BatchSize
	}

	return nil
}

func fetchConceptBatch(db *sql.DB, offset, limit int) ([]Concept, error) {
	query := `
		SELECT
			id,
			system_id,
			code,
			display,
			COALESCE(definition, '') as definition,
			status,
			COALESCE(clinical_domain, '') as clinical_domain,
			COALESCE(specialty, '') as specialty
		FROM terminology_concepts
		WHERE status = 'active'
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var concepts []Concept
	for rows.Next() {
		var c Concept
		if err := rows.Scan(
			&c.ID,
			&c.SystemID,
			&c.Code,
			&c.Display,
			&c.Definition,
			&c.Status,
			&c.ClinicalDomain,
			&c.Specialty,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		concepts = append(concepts, c)
	}

	return concepts, rows.Err()
}

func convertToTurtle(concepts []Concept) (string, int) {
	var sb strings.Builder
	tripleCount := 0

	// Write prefixes
	sb.WriteString("@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .\n")
	sb.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	sb.WriteString("@prefix skos: <http://www.w3.org/2004/02/skos/core#> .\n")
	sb.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
	sb.WriteString("@prefix dc: <http://purl.org/dc/elements/1.1/> .\n")
	sb.WriteString("\n")

	for _, concept := range concepts {
		conceptURI := fmt.Sprintf("<http://cardiofit.ai/kb7/concepts/%s>", concept.ID)

		// Type declaration
		sb.WriteString(fmt.Sprintf("%s a kb7:ClinicalConcept ;\n", conceptURI))
		tripleCount++

		// Core properties
		sb.WriteString(fmt.Sprintf("    kb7:code \"%s\" ;\n", escapeTurtle(concept.Code)))
		sb.WriteString(fmt.Sprintf("    kb7:system \"%s\" ;\n", escapeTurtle(concept.SystemID)))
		sb.WriteString(fmt.Sprintf("    rdfs:label \"%s\" ;\n", escapeTurtle(concept.Display)))
		tripleCount += 3

		// Optional properties
		if concept.Definition != "" {
			sb.WriteString(fmt.Sprintf("    skos:definition \"%s\" ;\n", escapeTurtle(concept.Definition)))
			tripleCount++
		}

		if concept.ClinicalDomain != "" {
			sb.WriteString(fmt.Sprintf("    kb7:clinicalDomain \"%s\" ;\n", escapeTurtle(concept.ClinicalDomain)))
			tripleCount++
		}

		if concept.Specialty != "" {
			sb.WriteString(fmt.Sprintf("    kb7:specialty \"%s\" ;\n", escapeTurtle(concept.Specialty)))
			tripleCount++
		}

		// Status (remove trailing semicolon on last property)
		sb.WriteString(fmt.Sprintf("    kb7:status \"%s\" .\n\n", escapeTurtle(concept.Status)))
		tripleCount++
	}

	return sb.String(), tripleCount
}

func getTotalConceptCount(db *sql.DB) (int64, error) {
	var count int64
	err := db.QueryRow("SELECT COUNT(*) FROM terminology_concepts WHERE status = 'active'").Scan(&count)
	return count, err
}

func validateMigration(ctx context.Context, pgDB *sql.DB, graphDB *semantic.GraphDBClient, stats *MigrationStats, logger *logrus.Logger) error {
	// Count concepts in GraphDB
	query := &semantic.SPARQLQuery{
		Query: `
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			SELECT (COUNT(?concept) AS ?count) WHERE {
				?concept a kb7:ClinicalConcept .
			}
		`,
	}

	results, err := graphDB.ExecuteSPARQL(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to count GraphDB concepts: %w", err)
	}

	if len(results.Results.Bindings) == 0 {
		return fmt.Errorf("no results from GraphDB count query")
	}

	graphDBCount := results.Results.Bindings[0]["count"].Value

	logger.WithFields(logrus.Fields{
		"postgresql_count": stats.MigratedConcepts,
		"graphdb_count":    graphDBCount,
		"match":            stats.MigratedConcepts == parseCount(graphDBCount),
	}).Info("Validation results")

	if stats.MigratedConcepts != parseCount(graphDBCount) {
		return fmt.Errorf("concept count mismatch: PostgreSQL=%d, GraphDB=%s", stats.MigratedConcepts, graphDBCount)
	}

	return nil
}

func printStats(stats *MigrationStats, logger *logrus.Logger) {
	duration := stats.EndTime.Sub(stats.StartTime)
	rate := float64(stats.MigratedConcepts) / duration.Seconds()

	logger.WithFields(logrus.Fields{
		"total_concepts":    stats.TotalConcepts,
		"migrated":          stats.MigratedConcepts,
		"failed":            stats.FailedConcepts,
		"total_triples":     stats.TotalTriples,
		"duration":          duration.Round(time.Second),
		"concepts_per_sec":  fmt.Sprintf("%.2f", rate),
		"success_rate":      fmt.Sprintf("%.2f%%", float64(stats.MigratedConcepts)/float64(stats.TotalConcepts)*100),
	}).Info("=== Migration Complete ===")
}

// Helper functions

type Concept struct {
	ID             string
	SystemID       string
	Code           string
	Display        string
	Definition     string
	Status         string
	ClinicalDomain string
	Specialty      string
}

func escapeTurtle(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func maskPassword(dbURL string) string {
	if strings.Contains(dbURL, "@") {
		parts := strings.Split(dbURL, "@")
		if len(parts) == 2 {
			userParts := strings.Split(parts[0], "://")
			if len(userParts) == 2 {
				return userParts[0] + "://***:***@" + parts[1]
			}
		}
	}
	return dbURL
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func estimateMigrationTime(totalConcepts int64, batchSize int) string {
	// Conservative estimate: 2 seconds per batch
	batches := (totalConcepts + int64(batchSize) - 1) / int64(batchSize)
	seconds := batches * 2
	duration := time.Duration(seconds) * time.Second
	return duration.Round(time.Minute).String()
}

func estimateRemainingTime(migrated, total int64, elapsed time.Duration) time.Duration {
	if migrated == 0 {
		return 0
	}
	rate := float64(migrated) / elapsed.Seconds()
	remaining := total - migrated
	return time.Duration(float64(remaining)/rate) * time.Second
}

func parseCount(countStr string) int64 {
	var count int64
	fmt.Sscanf(countStr, "%d", &count)
	return count
}
