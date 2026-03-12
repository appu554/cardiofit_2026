package ncts

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// NCTS Refset Loader
// ============================================================================
// Loads SNOMED CT-AU refset data from RF2 distribution files into Neo4j.
// Creates :IN_REFSET relationships between Concept and Refset nodes.
// ============================================================================

// NCTSRefsetLoader handles loading NCTS refset data into Neo4j
type NCTSRefsetLoader struct {
	neo4jClient *semantic.Neo4jClient
	logger      *logrus.Logger
	batchSize   int
	workers     int
	stats       *models.RefsetLoaderStats
	mu          sync.Mutex
}

// NCTSLoaderConfig configuration for NCTS loading
type NCTSLoaderConfig struct {
	ZipFilePath  string `json:"zip_file_path"`
	ExtractDir   string `json:"extract_dir"`
	BatchSize    int    `json:"batch_size"`
	Workers      int    `json:"workers"`
	SkipInactive bool   `json:"skip_inactive"`
	ModuleFilter string `json:"module_filter"` // Filter by module ID (empty = all)
}

// DefaultNCTSLoaderConfig returns default loader configuration
func DefaultNCTSLoaderConfig() *NCTSLoaderConfig {
	return &NCTSLoaderConfig{
		BatchSize:    5000,
		Workers:      4,
		SkipInactive: true,
		ModuleFilter: "", // Load all modules
	}
}

// NewNCTSRefsetLoader creates a new NCTS refset loader
func NewNCTSRefsetLoader(neo4jClient *semantic.Neo4jClient, logger *logrus.Logger) *NCTSRefsetLoader {
	return &NCTSRefsetLoader{
		neo4jClient: neo4jClient,
		logger:      logger,
		batchSize:   5000,
		workers:     4,
		stats:       &models.RefsetLoaderStats{},
	}
}

// SetBatchSize sets the batch size for bulk operations
func (l *NCTSRefsetLoader) SetBatchSize(size int) {
	l.batchSize = size
}

// SetWorkers sets the number of parallel workers
func (l *NCTSRefsetLoader) SetWorkers(workers int) {
	l.workers = workers
}

// ============================================================================
// Main Loading Methods
// ============================================================================

// LoadFromZip loads refset data from an NCTS ZIP archive
func (l *NCTSRefsetLoader) LoadFromZip(ctx context.Context, zipPath string, config *NCTSLoaderConfig) (*models.RefsetLoaderStats, error) {
	if config == nil {
		config = DefaultNCTSLoaderConfig()
	}

	l.stats = &models.RefsetLoaderStats{
		StartTime: time.Now(),
	}

	l.logger.WithField("zip_path", zipPath).Info("Starting NCTS refset load from ZIP")

	// Create temp directory for extraction
	extractDir := config.ExtractDir
	if extractDir == "" {
		var err error
		extractDir, err = os.MkdirTemp("", "ncts-extract-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(extractDir)
	}

	// Extract ZIP
	if err := l.extractZip(zipPath, extractDir); err != nil {
		return nil, fmt.Errorf("failed to extract ZIP: %w", err)
	}

	// Find and process RF2 files
	if err := l.processExtractedFiles(ctx, extractDir, config); err != nil {
		return nil, fmt.Errorf("failed to process extracted files: %w", err)
	}

	l.stats.EndTime = time.Now()
	l.stats.Duration = l.stats.EndTime.Sub(l.stats.StartTime).String()

	l.logger.WithFields(logrus.Fields{
		"duration":              l.stats.Duration,
		"files_processed":       l.stats.FilesProcessed,
		"rows_imported":         l.stats.RowsImported,
		"relationships_created": l.stats.RelationshipsCreated,
	}).Info("NCTS refset load complete")

	return l.stats, nil
}

// LoadFromDirectory loads refset data from an extracted directory
func (l *NCTSRefsetLoader) LoadFromDirectory(ctx context.Context, dirPath string, config *NCTSLoaderConfig) (*models.RefsetLoaderStats, error) {
	if config == nil {
		config = DefaultNCTSLoaderConfig()
	}

	l.stats = &models.RefsetLoaderStats{
		StartTime: time.Now(),
	}

	l.logger.WithField("dir_path", dirPath).Info("Starting NCTS refset load from directory")

	if err := l.processExtractedFiles(ctx, dirPath, config); err != nil {
		return nil, fmt.Errorf("failed to process files: %w", err)
	}

	l.stats.EndTime = time.Now()
	l.stats.Duration = l.stats.EndTime.Sub(l.stats.StartTime).String()

	return l.stats, nil
}

// ============================================================================
// ZIP Extraction
// ============================================================================

func (l *NCTSRefsetLoader) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destDir, f.Name)

		// Security check for zip slip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// ============================================================================
// File Processing
// ============================================================================

func (l *NCTSRefsetLoader) processExtractedFiles(ctx context.Context, dirPath string, config *NCTSLoaderConfig) error {
	// Find all RF2 refset files
	var refsetFiles []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		filename := info.Name()
		// Match RF2 refset file patterns
		if strings.Contains(filename, "Refset") && strings.HasSuffix(filename, ".txt") {
			// Prefer Snapshot files over Full/Delta
			if strings.Contains(filename, "Snapshot") {
				refsetFiles = append(refsetFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	l.logger.WithField("file_count", len(refsetFiles)).Info("Found RF2 refset files")

	// Process each file
	for _, filePath := range refsetFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := l.processRefsetFile(ctx, filePath, config); err != nil {
				l.logger.WithError(err).WithField("file", filePath).Error("Failed to process refset file")
				l.stats.Errors = append(l.stats.Errors, err.Error())
				continue
			}
			l.stats.FilesProcessed++
		}
	}

	return nil
}

func (l *NCTSRefsetLoader) processRefsetFile(ctx context.Context, filePath string, config *NCTSLoaderConfig) error {
	filename := filepath.Base(filePath)
	l.logger.WithField("file", filename).Info("Processing refset file")

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Read header
	if !scanner.Scan() {
		return fmt.Errorf("empty file: %s", filename)
	}
	header := scanner.Text()
	columns := strings.Split(header, "\t")

	// Determine refset type based on columns
	refsetType := l.determineRefsetType(columns)
	l.logger.WithFields(logrus.Fields{
		"file":        filename,
		"refset_type": refsetType,
		"columns":     len(columns),
	}).Debug("Detected refset type")

	// Process rows in batches
	var batch []models.RF2RefsetRow
	rowCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		fields := strings.Split(line, "\t")

		if len(fields) < 6 {
			continue
		}

		row := models.RF2RefsetRow{
			ID:                    fields[0],
			EffectiveTime:         fields[1],
			Active:                fields[2],
			ModuleID:              fields[3],
			RefsetID:              fields[4],
			ReferencedComponentID: fields[5],
		}

		// Apply filters
		if config.SkipInactive && !row.IsActive() {
			l.stats.RowsSkipped++
			continue
		}

		if config.ModuleFilter != "" && row.ModuleID != config.ModuleFilter {
			l.stats.RowsSkipped++
			continue
		}

		batch = append(batch, row)
		rowCount++
		l.stats.RowsRead++

		// Process batch
		if len(batch) >= config.BatchSize {
			if err := l.importBatch(ctx, batch, refsetType); err != nil {
				l.logger.WithError(err).Error("Failed to import batch")
				l.stats.Errors = append(l.stats.Errors, err.Error())
			}
			batch = batch[:0]

			if rowCount%10000 == 0 {
				l.logger.WithField("rows", rowCount).Info("Progress update")
			}
		}
	}

	// Process remaining batch
	if len(batch) > 0 {
		if err := l.importBatch(ctx, batch, refsetType); err != nil {
			return err
		}
	}

	l.logger.WithFields(logrus.Fields{
		"file":         filename,
		"rows_read":    rowCount,
		"rows_skipped": l.stats.RowsSkipped,
	}).Info("Completed processing file")

	return scanner.Err()
}

func (l *NCTSRefsetLoader) determineRefsetType(columns []string) string {
	columnCount := len(columns)

	// Simple refset: 6 columns (id, effectiveTime, active, moduleId, refsetId, referencedComponentId)
	if columnCount == 6 {
		return models.RefsetTypeSimple
	}

	// Association refset: 7 columns (+ targetComponentId)
	if columnCount == 7 {
		return models.RefsetTypeAssociation
	}

	// Language refset: 7 columns (+ acceptabilityId)
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col), "acceptability") {
			return models.RefsetTypeLanguage
		}
	}

	// Map refset: varies
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col), "maptarget") {
			return models.RefsetTypeMap
		}
	}

	return models.RefsetTypeSimple
}

// ============================================================================
// Neo4j Import
// ============================================================================

func (l *NCTSRefsetLoader) importBatch(ctx context.Context, batch []models.RF2RefsetRow, refsetType string) error {
	if len(batch) == 0 {
		return nil
	}

	// Build batch Cypher query
	cypher := l.buildBatchCypher(batch, refsetType)

	// Execute in Neo4j
	_, err := l.neo4jClient.ExecuteWrite(ctx, cypher, nil)
	if err != nil {
		return fmt.Errorf("failed to execute batch import: %w", err)
	}

	l.mu.Lock()
	l.stats.RowsImported += len(batch)
	l.stats.RelationshipsCreated += len(batch)
	l.mu.Unlock()

	return nil
}

func (l *NCTSRefsetLoader) buildBatchCypher(batch []models.RF2RefsetRow, refsetType string) string {
	var sb strings.Builder

	sb.WriteString("UNWIND $rows AS row\n")
	sb.WriteString("MATCH (c:Concept {code: row.referencedComponentId})\n")
	sb.WriteString("MERGE (r:Refset {id: row.refsetId})\n")
	sb.WriteString("ON CREATE SET r.name = 'Unknown Refset'\n")

	switch refsetType {
	case models.RefsetTypeSimple:
		sb.WriteString("CREATE (c)-[:IN_REFSET {\n")
		sb.WriteString("  memberId: row.id,\n")
		sb.WriteString("  effectiveTime: date(substring(row.effectiveTime, 0, 4) + '-' + substring(row.effectiveTime, 4, 2) + '-' + substring(row.effectiveTime, 6, 2)),\n")
		sb.WriteString("  active: true,\n")
		sb.WriteString("  moduleId: row.moduleId\n")
		sb.WriteString("}]->(r)\n")

	default:
		// Default to simple refset relationship
		sb.WriteString("CREATE (c)-[:IN_REFSET {\n")
		sb.WriteString("  memberId: row.id,\n")
		sb.WriteString("  effectiveTime: date(substring(row.effectiveTime, 0, 4) + '-' + substring(row.effectiveTime, 4, 2) + '-' + substring(row.effectiveTime, 6, 2)),\n")
		sb.WriteString("  active: true,\n")
		sb.WriteString("  moduleId: row.moduleId\n")
		sb.WriteString("}]->(r)\n")
	}

	return sb.String()
}

// ============================================================================
// Index Management
// ============================================================================

// CreateIndexes creates required indexes for refset queries
func (l *NCTSRefsetLoader) CreateIndexes(ctx context.Context) error {
	indexes := []string{
		"CREATE INDEX refset_id_idx IF NOT EXISTS FOR (r:Refset) ON (r.id)",
		"CREATE INDEX import_metadata_idx IF NOT EXISTS FOR (m:ImportMetadata) ON (m.type, m.version)",
		"CREATE CONSTRAINT refset_unique IF NOT EXISTS FOR (r:Refset) REQUIRE r.id IS UNIQUE",
	}

	for _, indexQuery := range indexes {
		if _, err := l.neo4jClient.ExecuteWrite(ctx, indexQuery, nil); err != nil {
			l.logger.WithError(err).WithField("query", indexQuery).Warn("Failed to create index")
			// Continue on error - index might already exist
		}
	}

	l.logger.Info("Indexes created/verified")
	return nil
}

// ============================================================================
// Version Management
// ============================================================================

// RecordImportMetadata records import metadata in Neo4j
func (l *NCTSRefsetLoader) RecordImportMetadata(ctx context.Context, version string, stats *models.RefsetLoaderStats) error {
	cypher := `
		MERGE (m:ImportMetadata {type: 'NCTS_REFSET', version: $version})
		SET m.importedAt = datetime(),
			m.fileCount = $fileCount,
			m.relationshipCount = $relationshipCount,
			m.importedBy = 'NCTSRefsetLoader'
	`

	params := map[string]interface{}{
		"version":           version,
		"fileCount":         stats.FilesProcessed,
		"relationshipCount": stats.RelationshipsCreated,
	}

	_, err := l.neo4jClient.ExecuteWrite(ctx, cypher, params)
	return err
}

// GetCurrentVersion returns the currently imported version
func (l *NCTSRefsetLoader) GetCurrentVersion(ctx context.Context) (string, error) {
	cypher := `
		MATCH (m:ImportMetadata {type: 'NCTS_REFSET'})
		RETURN m.version AS version
		ORDER BY m.importedAt DESC
		LIMIT 1
	`

	result, err := l.neo4jClient.ExecuteRead(ctx, cypher, nil)
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", nil
	}

	if version, ok := result[0]["version"].(string); ok {
		return version, nil
	}

	return "", nil
}

// ============================================================================
// Cleanup Operations
// ============================================================================

// DeleteAllRefsetRelationships deletes all IN_REFSET relationships
func (l *NCTSRefsetLoader) DeleteAllRefsetRelationships(ctx context.Context) error {
	cypher := `
		CALL apoc.periodic.iterate(
			'MATCH ()-[r:IN_REFSET]->() RETURN r',
			'DELETE r',
			{batchSize: 10000, parallel: false}
		) YIELD batches, total
		RETURN batches, total
	`

	result, err := l.neo4jClient.ExecuteWrite(ctx, cypher, nil)
	if err != nil {
		return err
	}

	l.logger.WithField("result", result).Info("Deleted all IN_REFSET relationships")
	return nil
}

// DeleteRefsetNodes deletes all Refset nodes
func (l *NCTSRefsetLoader) DeleteRefsetNodes(ctx context.Context) error {
	cypher := "MATCH (r:Refset) DETACH DELETE r"
	_, err := l.neo4jClient.ExecuteWrite(ctx, cypher, nil)
	return err
}

// DeleteImportMetadata deletes import metadata
func (l *NCTSRefsetLoader) DeleteImportMetadata(ctx context.Context) error {
	cypher := "MATCH (m:ImportMetadata {type: 'NCTS_REFSET'}) DELETE m"
	_, err := l.neo4jClient.ExecuteWrite(ctx, cypher, nil)
	return err
}
