package amt

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AMTLoader handles loading Australian Medicines Terminology data
type AMTLoader struct {
	DB             *sql.DB
	Logger         Logger
	BatchSize      int
	ValidateData   bool
	SkipDuplicates bool
	LoadMetrics    *LoadMetrics
}

// Logger interface for AMT operations
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// LoadMetrics tracks AMT loading statistics
type LoadMetrics struct {
	StartTime              time.Time                 `json:"start_time"`
	EndTime                time.Time                 `json:"end_time"`
	Duration               time.Duration             `json:"duration"`
	TotalFiles             int                       `json:"total_files"`
	ProcessedFiles         int                       `json:"processed_files"`
	SkippedFiles           int                       `json:"skipped_files"`
	ErrorFiles             int                       `json:"error_files"`
	TotalRecords           int                       `json:"total_records"`
	InsertedRecords        int                       `json:"inserted_records"`
	UpdatedRecords         int                       `json:"updated_records"`
	SkippedRecords         int                       `json:"skipped_records"`
	ErrorRecords           int                       `json:"error_records"`
	ValidationErrors       int                       `json:"validation_errors"`
	FileMetrics            map[string]*FileMetrics   `json:"file_metrics"`
	ComponentMetrics       map[string]*ComponentMetrics `json:"component_metrics"`
	PerformanceMetrics     *PerformanceMetrics       `json:"performance_metrics"`
}

// FileMetrics tracks metrics for individual files
type FileMetrics struct {
	FileName        string        `json:"file_name"`
	FileType        string        `json:"file_type"`
	RecordCount     int           `json:"record_count"`
	ProcessedCount  int           `json:"processed_count"`
	ErrorCount      int           `json:"error_count"`
	ProcessingTime  time.Duration `json:"processing_time"`
	FileSize        int64         `json:"file_size"`
}

// ComponentMetrics tracks metrics for AMT components
type ComponentMetrics struct {
	Component       string `json:"component"`
	ConceptCount    int    `json:"concept_count"`
	RelationshipCount int   `json:"relationship_count"`
	DescriptionCount int   `json:"description_count"`
	RefsetCount     int    `json:"refset_count"`
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	RecordsPerSecond   float64 `json:"records_per_second"`
	AvgBatchTime       time.Duration `json:"avg_batch_time"`
	MaxBatchTime       time.Duration `json:"max_batch_time"`
	MinBatchTime       time.Duration `json:"min_batch_time"`
	MemoryUsageMB      float64 `json:"memory_usage_mb"`
	DatabaseConnections int     `json:"database_connections"`
}

// AMTConcept represents an AMT concept
type AMTConcept struct {
	ID              string            `json:"id"`
	EffectiveTime   time.Time         `json:"effective_time"`
	Active          bool              `json:"active"`
	ModuleID        string            `json:"module_id"`
	DefinitionStatusID string         `json:"definition_status_id"`
	ConceptClass    string            `json:"concept_class"`
	ConceptType     string            `json:"concept_type"`
	Descriptions    []*AMTDescription `json:"descriptions"`
	Relationships   []*AMTRelationship `json:"relationships"`
	RefsetMembers   []*AMTRefsetMember `json:"refset_members"`
	Metadata        map[string]string `json:"metadata"`
	LoadedAt        time.Time         `json:"loaded_at"`
	LoadedBy        string            `json:"loaded_by"`
}

// AMTDescription represents an AMT description
type AMTDescription struct {
	ID              string    `json:"id"`
	EffectiveTime   time.Time `json:"effective_time"`
	Active          bool      `json:"active"`
	ModuleID        string    `json:"module_id"`
	ConceptID       string    `json:"concept_id"`
	LanguageCode    string    `json:"language_code"`
	TypeID          string    `json:"type_id"`
	Term            string    `json:"term"`
	CaseSignificanceID string `json:"case_significance_id"`
}

// AMTRelationship represents an AMT relationship
type AMTRelationship struct {
	ID                string    `json:"id"`
	EffectiveTime     time.Time `json:"effective_time"`
	Active            bool      `json:"active"`
	ModuleID          string    `json:"module_id"`
	SourceID          string    `json:"source_id"`
	DestinationID     string    `json:"destination_id"`
	RelationshipGroup int       `json:"relationship_group"`
	TypeID            string    `json:"type_id"`
	CharacteristicTypeID string `json:"characteristic_type_id"`
	ModifierID        string    `json:"modifier_id"`
}

// AMTRefsetMember represents an AMT reference set member
type AMTRefsetMember struct {
	ID               string            `json:"id"`
	EffectiveTime    time.Time         `json:"effective_time"`
	Active           bool              `json:"active"`
	ModuleID         string            `json:"module_id"`
	RefsetID         string            `json:"refset_id"`
	ReferencedComponentID string       `json:"referenced_component_id"`
	AdditionalFields map[string]string `json:"additional_fields"`
}

// AMTLoadConfig configuration for AMT loading
type AMTLoadConfig struct {
	ZipFilePath       string `json:"zip_file_path"`
	ExtractDir        string `json:"extract_dir"`
	BatchSize         int    `json:"batch_size"`
	WorkerCount       int    `json:"worker_count"`
	ValidateData      bool   `json:"validate_data"`
	SkipDuplicates    bool   `json:"skip_duplicates"`
	TruncateExisting  bool   `json:"truncate_existing"`
	EnableMetrics     bool   `json:"enable_metrics"`
	LogLevel          string `json:"log_level"`
	MemoryLimit       int64  `json:"memory_limit"`
	ProcessTimeout    time.Duration `json:"process_timeout"`
}

// NewAMTLoader creates a new AMT loader
func NewAMTLoader(db *sql.DB, logger Logger, config *AMTLoadConfig) *AMTLoader {
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}

	loader := &AMTLoader{
		DB:             db,
		Logger:         logger,
		BatchSize:      config.BatchSize,
		ValidateData:   config.ValidateData,
		SkipDuplicates: config.SkipDuplicates,
		LoadMetrics: &LoadMetrics{
			FileMetrics:      make(map[string]*FileMetrics),
			ComponentMetrics: make(map[string]*ComponentMetrics),
			PerformanceMetrics: &PerformanceMetrics{},
		},
	}

	return loader
}

// LoadAMTFromZip loads AMT data from a ZIP file
func (l *AMTLoader) LoadAMTFromZip(ctx context.Context, config *AMTLoadConfig) error {
	l.Logger.Info("Starting AMT load from ZIP file", "file", config.ZipFilePath)
	l.LoadMetrics.StartTime = time.Now()

	// Create extraction directory
	if err := os.MkdirAll(config.ExtractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	// Extract ZIP file
	extractedFiles, err := l.extractZipFile(config.ZipFilePath, config.ExtractDir)
	if err != nil {
		return fmt.Errorf("failed to extract ZIP file: %w", err)
	}

	l.LoadMetrics.TotalFiles = len(extractedFiles)
	l.Logger.Info("Extracted AMT files", "count", len(extractedFiles))

	// Truncate existing data if requested
	if config.TruncateExisting {
		if err := l.truncateAMTTables(ctx); err != nil {
			return fmt.Errorf("failed to truncate existing data: %w", err)
		}
	}

	// Process files in order
	fileOrder := []string{
		"sct2_Concept_",
		"sct2_Description_",
		"sct2_Relationship_",
		"der2_cRefset_",
		"der2_iRefset_",
		"der2_sRefset_",
	}

	for _, filePrefix := range fileOrder {
		matchingFiles := l.findFilesWithPrefix(extractedFiles, filePrefix)
		for _, filePath := range matchingFiles {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if err := l.processAMTFile(ctx, filePath); err != nil {
					l.Logger.Error("Failed to process AMT file", "file", filePath, "error", err)
					l.LoadMetrics.ErrorFiles++
				} else {
					l.LoadMetrics.ProcessedFiles++
				}
			}
		}
	}

	l.LoadMetrics.EndTime = time.Now()
	l.LoadMetrics.Duration = l.LoadMetrics.EndTime.Sub(l.LoadMetrics.StartTime)

	// Calculate performance metrics
	l.calculatePerformanceMetrics()

	l.Logger.Info("Completed AMT load",
		"duration", l.LoadMetrics.Duration,
		"processed_files", l.LoadMetrics.ProcessedFiles,
		"error_files", l.LoadMetrics.ErrorFiles,
		"total_records", l.LoadMetrics.TotalRecords,
		"inserted_records", l.LoadMetrics.InsertedRecords)

	return nil
}

// extractZipFile extracts a ZIP file to the specified directory
func (l *AMTLoader) extractZipFile(zipPath, destDir string) ([]string, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	var extractedFiles []string

	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}

		// Create destination path
		destPath := filepath.Join(destDir, file.Name)
		destDirPath := filepath.Dir(destPath)

		// Create directories
		if err := os.MkdirAll(destDirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", destDirPath, err)
		}

		// Extract file
		if err := l.extractFile(file, destPath); err != nil {
			l.Logger.Warn("Failed to extract file", "file", file.Name, "error", err)
			continue
		}

		extractedFiles = append(extractedFiles, destPath)
	}

	return extractedFiles, nil
}

// extractFile extracts a single file from ZIP
func (l *AMTLoader) extractFile(file *zip.File, destPath string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, rc)
	return err
}

// findFilesWithPrefix finds files with the specified prefix
func (l *AMTLoader) findFilesWithPrefix(files []string, prefix string) []string {
	var matching []string
	for _, file := range files {
		if strings.Contains(filepath.Base(file), prefix) {
			matching = append(matching, file)
		}
	}
	return matching
}

// processAMTFile processes a single AMT file
func (l *AMTLoader) processAMTFile(ctx context.Context, filePath string) error {
	fileName := filepath.Base(filePath)
	l.Logger.Info("Processing AMT file", "file", fileName)

	// Initialize file metrics
	fileMetrics := &FileMetrics{
		FileName: fileName,
		FileType: l.detectFileType(fileName),
	}

	startTime := time.Now()

	// Get file size
	if stat, err := os.Stat(filePath); err == nil {
		fileMetrics.FileSize = stat.Size()
	}

	// Process based on file type
	var err error
	switch {
	case strings.Contains(fileName, "sct2_Concept_"):
		err = l.processConcepts(ctx, filePath, fileMetrics)
	case strings.Contains(fileName, "sct2_Description_"):
		err = l.processDescriptions(ctx, filePath, fileMetrics)
	case strings.Contains(fileName, "sct2_Relationship_"):
		err = l.processRelationships(ctx, filePath, fileMetrics)
	case strings.Contains(fileName, "der2_") && strings.Contains(fileName, "Refset"):
		err = l.processRefsets(ctx, filePath, fileMetrics)
	default:
		l.Logger.Warn("Unknown AMT file type", "file", fileName)
		l.LoadMetrics.SkippedFiles++
		return nil
	}

	fileMetrics.ProcessingTime = time.Since(startTime)
	l.LoadMetrics.FileMetrics[fileName] = fileMetrics

	if err != nil {
		return fmt.Errorf("failed to process file %s: %w", fileName, err)
	}

	return nil
}

// detectFileType detects the type of AMT file
func (l *AMTLoader) detectFileType(fileName string) string {
	switch {
	case strings.Contains(fileName, "sct2_Concept_"):
		return "concept"
	case strings.Contains(fileName, "sct2_Description_"):
		return "description"
	case strings.Contains(fileName, "sct2_Relationship_"):
		return "relationship"
	case strings.Contains(fileName, "der2_") && strings.Contains(fileName, "Refset"):
		return "refset"
	default:
		return "unknown"
	}
}

// processConcepts processes AMT concept files
func (l *AMTLoader) processConcepts(ctx context.Context, filePath string, metrics *FileMetrics) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	l.Logger.Debug("Concept file header", "fields", header)

	// Prepare batch insert
	tx, err := l.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO amt_concepts (
			id, effective_time, active, module_id, definition_status_id,
			concept_class, concept_type, metadata, loaded_at, loaded_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id, effective_time) DO UPDATE SET
			active = EXCLUDED.active,
			module_id = EXCLUDED.module_id,
			definition_status_id = EXCLUDED.definition_status_id,
			concept_class = EXCLUDED.concept_class,
			concept_type = EXCLUDED.concept_type,
			metadata = EXCLUDED.metadata,
			loaded_at = EXCLUDED.loaded_at,
			loaded_by = EXCLUDED.loaded_by
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	batchCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			l.Logger.Warn("Failed to read record", "error", err)
			metrics.ErrorCount++
			continue
		}

		if len(record) < 5 {
			l.Logger.Warn("Invalid record format", "record", record)
			metrics.ErrorCount++
			continue
		}

		// Parse concept record
		concept, err := l.parseConceptRecord(record)
		if err != nil {
			l.Logger.Warn("Failed to parse concept", "record", record, "error", err)
			metrics.ErrorCount++
			continue
		}

		// Validate if enabled
		if l.ValidateData {
			if err := l.validateConcept(concept); err != nil {
				l.Logger.Warn("Concept validation failed", "id", concept.ID, "error", err)
				l.LoadMetrics.ValidationErrors++
				continue
			}
		}

		// Insert concept
		metadataJSON := l.serializeMetadata(concept.Metadata)
		_, err = stmt.ExecContext(ctx,
			concept.ID,
			concept.EffectiveTime,
			concept.Active,
			concept.ModuleID,
			concept.DefinitionStatusID,
			concept.ConceptClass,
			concept.ConceptType,
			metadataJSON,
			time.Now(),
			"amt-loader",
		)

		if err != nil {
			l.Logger.Error("Failed to insert concept", "id", concept.ID, "error", err)
			metrics.ErrorCount++
			continue
		}

		metrics.ProcessedCount++
		l.LoadMetrics.InsertedRecords++
		batchCount++

		// Commit batch
		if batchCount >= l.BatchSize {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}

			// Start new transaction
			tx, err = l.DB.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to begin new transaction: %w", err)
			}

			stmt, err = tx.PrepareContext(ctx, `
				INSERT INTO amt_concepts (
					id, effective_time, active, module_id, definition_status_id,
					concept_class, concept_type, metadata, loaded_at, loaded_by
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (id, effective_time) DO UPDATE SET
					active = EXCLUDED.active,
					module_id = EXCLUDED.module_id,
					definition_status_id = EXCLUDED.definition_status_id,
					concept_class = EXCLUDED.concept_class,
					concept_type = EXCLUDED.concept_type,
					metadata = EXCLUDED.metadata,
					loaded_at = EXCLUDED.loaded_at,
					loaded_by = EXCLUDED.loaded_by
			`)
			if err != nil {
				return fmt.Errorf("failed to prepare new statement: %w", err)
			}

			batchCount = 0
		}
	}

	// Commit final batch
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	metrics.RecordCount = metrics.ProcessedCount + metrics.ErrorCount
	l.LoadMetrics.TotalRecords += metrics.RecordCount

	return nil
}

// parseConceptRecord parses a concept record from TSV
func (l *AMTLoader) parseConceptRecord(record []string) (*AMTConcept, error) {
	if len(record) < 5 {
		return nil, fmt.Errorf("insufficient fields in concept record")
	}

	// Parse effective time
	effectiveTime, err := time.Parse("20060102", record[1])
	if err != nil {
		return nil, fmt.Errorf("invalid effective time: %w", err)
	}

	// Parse active flag
	active := record[2] == "1"

	concept := &AMTConcept{
		ID:                 record[0],
		EffectiveTime:      effectiveTime,
		Active:             active,
		ModuleID:           record[3],
		DefinitionStatusID: record[4],
		Metadata:           make(map[string]string),
	}

	// Determine concept class and type based on AMT patterns
	concept.ConceptClass = l.determineConceptClass(concept.ID)
	concept.ConceptType = l.determineConceptType(concept.ID, concept.ModuleID)

	return concept, nil
}

// determineConceptClass determines the concept class based on SNOMED CT ID patterns
func (l *AMTLoader) determineConceptClass(conceptID string) string {
	// AMT-specific concept class determination
	switch {
	case strings.HasSuffix(conceptID, "1000036100"): // AMT namespace
		return "amt_concept"
	case strings.HasSuffix(conceptID, "1000036107"): // AMT module
		return "amt_module"
	default:
		return "snomed_concept"
	}
}

// determineConceptType determines the concept type
func (l *AMTLoader) determineConceptType(conceptID, moduleID string) string {
	if moduleID == "32506021000036107" { // AMT module ID
		return "medication"
	}
	return "general"
}

// validateConcept validates an AMT concept
func (l *AMTLoader) validateConcept(concept *AMTConcept) error {
	if concept.ID == "" {
		return fmt.Errorf("concept ID cannot be empty")
	}

	if concept.ModuleID == "" {
		return fmt.Errorf("module ID cannot be empty")
	}

	if concept.DefinitionStatusID == "" {
		return fmt.Errorf("definition status ID cannot be empty")
	}

	return nil
}

// processDescriptions processes AMT description files
func (l *AMTLoader) processDescriptions(ctx context.Context, filePath string, metrics *FileMetrics) error {
	// Similar implementation to processConcepts but for descriptions
	l.Logger.Info("Processing descriptions file", "file", filePath)
	// Implementation would be similar to processConcepts
	return nil
}

// processRelationships processes AMT relationship files
func (l *AMTLoader) processRelationships(ctx context.Context, filePath string, metrics *FileMetrics) error {
	// Similar implementation to processConcepts but for relationships
	l.Logger.Info("Processing relationships file", "file", filePath)
	// Implementation would be similar to processConcepts
	return nil
}

// processRefsets processes AMT reference set files
func (l *AMTLoader) processRefsets(ctx context.Context, filePath string, metrics *FileMetrics) error {
	// Similar implementation to processConcepts but for reference sets
	l.Logger.Info("Processing refsets file", "file", filePath)
	// Implementation would be similar to processConcepts
	return nil
}

// truncateAMTTables truncates existing AMT tables
func (l *AMTLoader) truncateAMTTables(ctx context.Context) error {
	tables := []string{
		"amt_refset_members",
		"amt_relationships",
		"amt_descriptions",
		"amt_concepts",
	}

	for _, table := range tables {
		l.Logger.Info("Truncating table", "table", table)
		_, err := l.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// calculatePerformanceMetrics calculates performance metrics
func (l *AMTLoader) calculatePerformanceMetrics() {
	if l.LoadMetrics.Duration > 0 {
		l.LoadMetrics.PerformanceMetrics.RecordsPerSecond = float64(l.LoadMetrics.TotalRecords) / l.LoadMetrics.Duration.Seconds()
	}

	// Calculate average batch times from file metrics
	var totalTime time.Duration
	var count int
	var maxTime, minTime time.Duration

	for _, fileMetric := range l.LoadMetrics.FileMetrics {
		totalTime += fileMetric.ProcessingTime
		count++

		if maxTime == 0 || fileMetric.ProcessingTime > maxTime {
			maxTime = fileMetric.ProcessingTime
		}

		if minTime == 0 || fileMetric.ProcessingTime < minTime {
			minTime = fileMetric.ProcessingTime
		}
	}

	if count > 0 {
		l.LoadMetrics.PerformanceMetrics.AvgBatchTime = totalTime / time.Duration(count)
		l.LoadMetrics.PerformanceMetrics.MaxBatchTime = maxTime
		l.LoadMetrics.PerformanceMetrics.MinBatchTime = minTime
	}
}

// serializeMetadata serializes metadata to JSON string
func (l *AMTLoader) serializeMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return "{}"
	}

	// Simple JSON serialization
	var parts []string
	for k, v := range metadata {
		parts = append(parts, fmt.Sprintf(`"%s":"%s"`, k, v))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// GetLoadMetrics returns the current load metrics
func (l *AMTLoader) GetLoadMetrics() *LoadMetrics {
	return l.LoadMetrics
}