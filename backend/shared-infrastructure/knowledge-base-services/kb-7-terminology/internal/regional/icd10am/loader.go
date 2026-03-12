package icd10am

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ICD10AMLoader handles loading ICD-10-AM (Australian Modification) data
type ICD10AMLoader struct {
	DB             *sql.DB
	Logger         Logger
	BatchSize      int
	ValidateData   bool
	SkipDuplicates bool
	LoadMetrics    *LoadMetrics
	IHACPAConfig   *IHACPAConfig
}

// Logger interface for ICD-10-AM operations
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// IHACPAConfig configuration for IHACPA (Independent Hospital and Aged Care Pricing Authority) access
type IHACPAConfig struct {
	BaseURL           string `json:"base_url"`
	InstitutionID     string `json:"institution_id"`
	AccessKey         string `json:"access_key"`
	CertificatePath   string `json:"certificate_path"`
	PrivateKeyPath    string `json:"private_key_path"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	RetryAttempts     int    `json:"retry_attempts"`
	VerifySSL         bool   `json:"verify_ssl"`
}

// LoadMetrics tracks ICD-10-AM loading statistics
type LoadMetrics struct {
	StartTime           time.Time                 `json:"start_time"`
	EndTime             time.Time                 `json:"end_time"`
	Duration            time.Duration             `json:"duration"`
	TotalFiles          int                       `json:"total_files"`
	ProcessedFiles      int                       `json:"processed_files"`
	SkippedFiles        int                       `json:"skipped_files"`
	ErrorFiles          int                       `json:"error_files"`
	TotalCodes          int                       `json:"total_codes"`
	InsertedCodes       int                       `json:"inserted_codes"`
	UpdatedCodes        int                       `json:"updated_codes"`
	SkippedCodes        int                       `json:"skipped_codes"`
	ErrorCodes          int                       `json:"error_codes"`
	ValidationErrors    int                       `json:"validation_errors"`
	ChapterMetrics      map[string]*ChapterMetrics `json:"chapter_metrics"`
	CategoryMetrics     map[string]*CategoryMetrics `json:"category_metrics"`
	PerformanceMetrics  *PerformanceMetrics       `json:"performance_metrics"`
}

// ChapterMetrics tracks metrics for ICD-10-AM chapters
type ChapterMetrics struct {
	ChapterNumber   int    `json:"chapter_number"`
	ChapterTitle    string `json:"chapter_title"`
	CodeRange       string `json:"code_range"`
	TotalCodes      int    `json:"total_codes"`
	ProcessedCodes  int    `json:"processed_codes"`
	ErrorCodes      int    `json:"error_codes"`
}

// CategoryMetrics tracks metrics for ICD-10-AM categories
type CategoryMetrics struct {
	Category        string `json:"category"`
	CodeCount       int    `json:"code_count"`
	DescriptionCount int    `json:"description_count"`
	ModifierCount   int    `json:"modifier_count"`
	ExclusionCount  int    `json:"exclusion_count"`
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	CodesPerSecond     float64 `json:"codes_per_second"`
	AvgBatchTime       time.Duration `json:"avg_batch_time"`
	MaxBatchTime       time.Duration `json:"max_batch_time"`
	MinBatchTime       time.Duration `json:"min_batch_time"`
	MemoryUsageMB      float64 `json:"memory_usage_mb"`
	DatabaseConnections int     `json:"database_connections"`
}

// ICD10AMCode represents an ICD-10-AM code
type ICD10AMCode struct {
	ID               string               `json:"id"`
	Code             string               `json:"code"`
	Title            string               `json:"title"`
	Description      string               `json:"description"`
	Category         string               `json:"category"`
	Chapter          int                  `json:"chapter"`
	ChapterTitle     string               `json:"chapter_title"`
	CodeType         string               `json:"code_type"` // "category", "subcategory", "code"
	ParentCode       *string              `json:"parent_code,omitempty"`
	Level            int                  `json:"level"`
	IsLeaf           bool                 `json:"is_leaf"`
	Gender           *string              `json:"gender,omitempty"` // "M", "F", null
	AgeRange         *AgeRange            `json:"age_range,omitempty"`
	Includes         []string             `json:"includes"`
	Excludes         []string             `json:"excludes"`
	Notes            []string             `json:"notes"`
	Modifiers        []*ICD10AMModifier   `json:"modifiers"`
	AustralianNotes  []string             `json:"australian_notes"`
	CodingGuidelines []string             `json:"coding_guidelines"`
	DRGRelevant      bool                 `json:"drg_relevant"`
	ValidFrom        time.Time            `json:"valid_from"`
	ValidTo          *time.Time           `json:"valid_to,omitempty"`
	Version          string               `json:"version"`
	LoadedAt         time.Time            `json:"loaded_at"`
	LoadedBy         string               `json:"loaded_by"`
	Metadata         map[string]string    `json:"metadata"`
}

// AgeRange represents an age range restriction
type AgeRange struct {
	MinAge    *int    `json:"min_age,omitempty"`
	MaxAge    *int    `json:"max_age,omitempty"`
	AgeUnit   string  `json:"age_unit"` // "days", "months", "years"
	AgeText   string  `json:"age_text"`
}

// ICD10AMModifier represents a modifier for ICD-10-AM codes
type ICD10AMModifier struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // "seventh_character", "extension", "etiology"
	Mandatory   bool      `json:"mandatory"`
	ValidCodes  []string  `json:"valid_codes"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
}

// ICD10AMChapter represents an ICD-10-AM chapter
type ICD10AMChapter struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	CodeRange   string    `json:"code_range"`
	Description string    `json:"description"`
	Notes       []string  `json:"notes"`
	ValidFrom   time.Time `json:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`
}

// ICD10AMLoadConfig configuration for ICD-10-AM loading
type ICD10AMLoadConfig struct {
	DataDirectory     string        `json:"data_directory"`
	Version           string        `json:"version"`
	Edition           string        `json:"edition"` // "11th", "12th", etc.
	BatchSize         int           `json:"batch_size"`
	WorkerCount       int           `json:"worker_count"`
	ValidateData      bool          `json:"validate_data"`
	SkipDuplicates    bool          `json:"skip_duplicates"`
	TruncateExisting  bool          `json:"truncate_existing"`
	EnableMetrics     bool          `json:"enable_metrics"`
	ProcessTimeout    time.Duration `json:"process_timeout"`
	IncludeDRGCodes   bool          `json:"include_drg_codes"`
	IncludeModifiers  bool          `json:"include_modifiers"`
	LoadMorbidityOnly bool          `json:"load_morbidity_only"`
}

// ICD10AMDataFile represents an ICD-10-AM data file structure
type ICD10AMDataFile struct {
	FilePath    string `json:"file_path"`
	FileType    string `json:"file_type"` // "xml", "csv", "txt"
	DataType    string `json:"data_type"` // "codes", "modifiers", "chapters", "guidelines"
	Version     string `json:"version"`
	Description string `json:"description"`
}

// XMLChapter represents chapter structure in XML files
type XMLChapter struct {
	XMLName     xml.Name      `xml:"chapter"`
	Number      int           `xml:"number,attr"`
	Title       string        `xml:"title,attr"`
	CodeRange   string        `xml:"range,attr"`
	Description string        `xml:"description"`
	Categories  []XMLCategory `xml:"category"`
}

// XMLCategory represents category structure in XML files
type XMLCategory struct {
	XMLName     xml.Name     `xml:"category"`
	Code        string       `xml:"code,attr"`
	Title       string       `xml:"title,attr"`
	Description string       `xml:"description"`
	Includes    []string     `xml:"include"`
	Excludes    []string     `xml:"exclude"`
	Notes       []string     `xml:"note"`
	Subcodes    []XMLSubcode `xml:"subcode"`
}

// XMLSubcode represents subcode structure in XML files
type XMLSubcode struct {
	XMLName     xml.Name `xml:"subcode"`
	Code        string   `xml:"code,attr"`
	Title       string   `xml:"title,attr"`
	Description string   `xml:"description"`
	Gender      string   `xml:"gender,attr,omitempty"`
	AgeRange    string   `xml:"age,attr,omitempty"`
}

// NewICD10AMLoader creates a new ICD-10-AM loader
func NewICD10AMLoader(db *sql.DB, logger Logger, config *ICD10AMLoadConfig) *ICD10AMLoader {
	if config.BatchSize == 0 {
		config.BatchSize = 500
	}

	loader := &ICD10AMLoader{
		DB:             db,
		Logger:         logger,
		BatchSize:      config.BatchSize,
		ValidateData:   config.ValidateData,
		SkipDuplicates: config.SkipDuplicates,
		LoadMetrics: &LoadMetrics{
			ChapterMetrics:      make(map[string]*ChapterMetrics),
			CategoryMetrics:     make(map[string]*CategoryMetrics),
			PerformanceMetrics:  &PerformanceMetrics{},
		},
	}

	return loader
}

// LoadICD10AMFromDirectory loads ICD-10-AM data from a directory containing data files
func (l *ICD10AMLoader) LoadICD10AMFromDirectory(ctx context.Context, config *ICD10AMLoadConfig) error {
	l.Logger.Info("Starting ICD-10-AM load from directory",
		"directory", config.DataDirectory,
		"version", config.Version,
		"edition", config.Edition)

	l.LoadMetrics.StartTime = time.Now()

	// Discover data files
	dataFiles, err := l.discoverDataFiles(config.DataDirectory)
	if err != nil {
		return fmt.Errorf("failed to discover data files: %w", err)
	}

	l.LoadMetrics.TotalFiles = len(dataFiles)
	l.Logger.Info("Discovered ICD-10-AM data files", "count", len(dataFiles))

	// Truncate existing data if requested
	if config.TruncateExisting {
		if err := l.truncateICD10AMTables(ctx); err != nil {
			return fmt.Errorf("failed to truncate existing data: %w", err)
		}
	}

	// Load chapters first
	if err := l.loadChapters(ctx, dataFiles, config); err != nil {
		return fmt.Errorf("failed to load chapters: %w", err)
	}

	// Load codes and categories
	if err := l.loadCodes(ctx, dataFiles, config); err != nil {
		return fmt.Errorf("failed to load codes: %w", err)
	}

	// Load modifiers if enabled
	if config.IncludeModifiers {
		if err := l.loadModifiers(ctx, dataFiles, config); err != nil {
			l.Logger.Warn("Failed to load modifiers", "error", err)
		}
	}

	// Update relationships and hierarchies
	if err := l.updateHierarchies(ctx); err != nil {
		l.Logger.Warn("Failed to update hierarchies", "error", err)
	}

	l.LoadMetrics.EndTime = time.Now()
	l.LoadMetrics.Duration = l.LoadMetrics.EndTime.Sub(l.LoadMetrics.StartTime)

	// Calculate performance metrics
	l.calculatePerformanceMetrics()

	l.Logger.Info("Completed ICD-10-AM load",
		"duration", l.LoadMetrics.Duration,
		"processed_files", l.LoadMetrics.ProcessedFiles,
		"total_codes", l.LoadMetrics.TotalCodes,
		"inserted_codes", l.LoadMetrics.InsertedCodes)

	return nil
}

// discoverDataFiles discovers ICD-10-AM data files in the directory
func (l *ICD10AMLoader) discoverDataFiles(directory string) ([]*ICD10AMDataFile, error) {
	var dataFiles []*ICD10AMDataFile

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Detect file type and purpose based on filename patterns
		fileName := strings.ToLower(filepath.Base(path))
		ext := strings.ToLower(filepath.Ext(path))

		dataFile := &ICD10AMDataFile{
			FilePath: path,
			FileType: ext[1:], // Remove leading dot
		}

		// Determine data type based on filename patterns
		switch {
		case strings.Contains(fileName, "chapter"):
			dataFile.DataType = "chapters"
		case strings.Contains(fileName, "modifier") || strings.Contains(fileName, "extension"):
			dataFile.DataType = "modifiers"
		case strings.Contains(fileName, "guideline") || strings.Contains(fileName, "instruction"):
			dataFile.DataType = "guidelines"
		case strings.Contains(fileName, "icd10am") || strings.Contains(fileName, "tabular"):
			dataFile.DataType = "codes"
		case strings.Contains(fileName, "index"):
			dataFile.DataType = "index"
		default:
			// Try to infer from file content if possible
			dataFile.DataType = "unknown"
		}

		// Only include relevant file types
		if dataFile.FileType == "xml" || dataFile.FileType == "csv" || dataFile.FileType == "txt" {
			dataFiles = append(dataFiles, dataFile)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return dataFiles, nil
}

// loadChapters loads ICD-10-AM chapters
func (l *ICD10AMLoader) loadChapters(ctx context.Context, dataFiles []*ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	l.Logger.Info("Loading ICD-10-AM chapters")

	// Find chapter files
	chapterFiles := l.filterFilesByDataType(dataFiles, "chapters")
	if len(chapterFiles) == 0 {
		l.Logger.Warn("No chapter files found, creating default chapters")
		return l.createDefaultChapters(ctx)
	}

	for _, file := range chapterFiles {
		if err := l.processChapterFile(ctx, file); err != nil {
			l.Logger.Error("Failed to process chapter file", "file", file.FilePath, "error", err)
			continue
		}
	}

	return nil
}

// loadCodes loads ICD-10-AM codes and categories
func (l *ICD10AMLoader) loadCodes(ctx context.Context, dataFiles []*ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	l.Logger.Info("Loading ICD-10-AM codes")

	// Find code files
	codeFiles := l.filterFilesByDataType(dataFiles, "codes")
	if len(codeFiles) == 0 {
		return fmt.Errorf("no code files found")
	}

	for _, file := range codeFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := l.processCodeFile(ctx, file, config); err != nil {
				l.Logger.Error("Failed to process code file", "file", file.FilePath, "error", err)
				l.LoadMetrics.ErrorFiles++
				continue
			}
			l.LoadMetrics.ProcessedFiles++
		}
	}

	return nil
}

// loadModifiers loads ICD-10-AM modifiers
func (l *ICD10AMLoader) loadModifiers(ctx context.Context, dataFiles []*ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	l.Logger.Info("Loading ICD-10-AM modifiers")

	// Find modifier files
	modifierFiles := l.filterFilesByDataType(dataFiles, "modifiers")
	if len(modifierFiles) == 0 {
		l.Logger.Info("No modifier files found, skipping modifier loading")
		return nil
	}

	for _, file := range modifierFiles {
		if err := l.processModifierFile(ctx, file); err != nil {
			l.Logger.Error("Failed to process modifier file", "file", file.FilePath, "error", err)
			continue
		}
	}

	return nil
}

// filterFilesByDataType filters files by data type
func (l *ICD10AMLoader) filterFilesByDataType(files []*ICD10AMDataFile, dataType string) []*ICD10AMDataFile {
	var filtered []*ICD10AMDataFile
	for _, file := range files {
		if file.DataType == dataType {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// processCodeFile processes a single ICD-10-AM code file
func (l *ICD10AMLoader) processCodeFile(ctx context.Context, file *ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	l.Logger.Info("Processing ICD-10-AM code file", "file", file.FilePath, "type", file.FileType)

	switch file.FileType {
	case "xml":
		return l.processXMLCodeFile(ctx, file, config)
	case "csv":
		return l.processCSVCodeFile(ctx, file, config)
	case "txt":
		return l.processTextCodeFile(ctx, file, config)
	default:
		return fmt.Errorf("unsupported file type: %s", file.FileType)
	}
}

// processXMLCodeFile processes XML-formatted ICD-10-AM files
func (l *ICD10AMLoader) processXMLCodeFile(ctx context.Context, file *ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	xmlFile, err := os.Open(file.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open XML file: %w", err)
	}
	defer xmlFile.Close()

	decoder := xml.NewDecoder(xmlFile)

	// Process XML tokens
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode XML: %w", err)
		}

		switch element := token.(type) {
		case xml.StartElement:
			if element.Name.Local == "chapter" {
				var chapter XMLChapter
				if err := decoder.DecodeElement(&chapter, &element); err != nil {
					l.Logger.Warn("Failed to decode chapter", "error", err)
					continue
				}

				if err := l.processXMLChapter(ctx, &chapter, config); err != nil {
					l.Logger.Error("Failed to process chapter", "chapter", chapter.Number, "error", err)
				}
			}
		}
	}

	return nil
}

// processXMLChapter processes a single XML chapter
func (l *ICD10AMLoader) processXMLChapter(ctx context.Context, chapter *XMLChapter, config *ICD10AMLoadConfig) error {
	// Process categories within the chapter
	for _, category := range chapter.Categories {
		icdCode := &ICD10AMCode{
			ID:              uuid.New().String(),
			Code:            category.Code,
			Title:           category.Title,
			Description:     category.Description,
			Category:        "category",
			Chapter:         chapter.Number,
			ChapterTitle:    chapter.Title,
			CodeType:        "category",
			Level:           1,
			IsLeaf:          len(category.Subcodes) == 0,
			Includes:        category.Includes,
			Excludes:        category.Excludes,
			Notes:           category.Notes,
			ValidFrom:       time.Now(),
			Version:         config.Version,
			LoadedAt:        time.Now(),
			LoadedBy:        "icd10am-loader",
			Metadata:        make(map[string]string),
		}

		// Add chapter information to metadata
		icdCode.Metadata["chapter_range"] = chapter.CodeRange
		icdCode.Metadata["edition"] = config.Edition

		// Insert the category code
		if err := l.insertICD10AMCode(ctx, icdCode); err != nil {
			return fmt.Errorf("failed to insert category code %s: %w", category.Code, err)
		}

		l.LoadMetrics.TotalCodes++
		l.LoadMetrics.InsertedCodes++

		// Process subcodes
		for _, subcode := range category.Subcodes {
			subcodeObj := &ICD10AMCode{
				ID:           uuid.New().String(),
				Code:         subcode.Code,
				Title:        subcode.Title,
				Description:  subcode.Description,
				Category:     category.Code,
				Chapter:      chapter.Number,
				ChapterTitle: chapter.Title,
				CodeType:     "code",
				ParentCode:   &category.Code,
				Level:        2,
				IsLeaf:       true,
				ValidFrom:    time.Now(),
				Version:      config.Version,
				LoadedAt:     time.Now(),
				LoadedBy:     "icd10am-loader",
				Metadata:     make(map[string]string),
			}

			// Parse gender restriction
			if subcode.Gender != "" {
				subcodeObj.Gender = &subcode.Gender
			}

			// Parse age range
			if subcode.AgeRange != "" {
				ageRange := l.parseAgeRange(subcode.AgeRange)
				subcodeObj.AgeRange = ageRange
			}

			// Insert the subcode
			if err := l.insertICD10AMCode(ctx, subcodeObj); err != nil {
				l.Logger.Error("Failed to insert subcode", "code", subcode.Code, "error", err)
				l.LoadMetrics.ErrorCodes++
				continue
			}

			l.LoadMetrics.TotalCodes++
			l.LoadMetrics.InsertedCodes++
		}
	}

	return nil
}

// insertICD10AMCode inserts an ICD-10-AM code into the database
func (l *ICD10AMLoader) insertICD10AMCode(ctx context.Context, code *ICD10AMCode) error {
	query := `
		INSERT INTO icd10am_codes (
			id, code, title, description, category, chapter, chapter_title,
			code_type, parent_code, level, is_leaf, gender, age_range,
			includes, excludes, notes, australian_notes, coding_guidelines,
			drg_relevant, valid_from, valid_to, version, loaded_at, loaded_by, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25
		) ON CONFLICT (code, version) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			chapter = EXCLUDED.chapter,
			chapter_title = EXCLUDED.chapter_title,
			code_type = EXCLUDED.code_type,
			parent_code = EXCLUDED.parent_code,
			level = EXCLUDED.level,
			is_leaf = EXCLUDED.is_leaf,
			gender = EXCLUDED.gender,
			age_range = EXCLUDED.age_range,
			includes = EXCLUDED.includes,
			excludes = EXCLUDED.excludes,
			notes = EXCLUDED.notes,
			australian_notes = EXCLUDED.australian_notes,
			coding_guidelines = EXCLUDED.coding_guidelines,
			drg_relevant = EXCLUDED.drg_relevant,
			valid_from = EXCLUDED.valid_from,
			valid_to = EXCLUDED.valid_to,
			loaded_at = EXCLUDED.loaded_at,
			loaded_by = EXCLUDED.loaded_by,
			metadata = EXCLUDED.metadata
	`

	// Serialize arrays to JSON
	includesJSON := l.serializeStringArray(code.Includes)
	excludesJSON := l.serializeStringArray(code.Excludes)
	notesJSON := l.serializeStringArray(code.Notes)
	australianNotesJSON := l.serializeStringArray(code.AustralianNotes)
	guidelinesJSON := l.serializeStringArray(code.CodingGuidelines)
	metadataJSON := l.serializeMetadata(code.Metadata)

	// Serialize age range
	var ageRangeJSON string
	if code.AgeRange != nil {
		ageRangeJSON = l.serializeAgeRange(code.AgeRange)
	}

	_, err := l.DB.ExecContext(ctx, query,
		code.ID, code.Code, code.Title, code.Description, code.Category,
		code.Chapter, code.ChapterTitle, code.CodeType, code.ParentCode,
		code.Level, code.IsLeaf, code.Gender, ageRangeJSON,
		includesJSON, excludesJSON, notesJSON, australianNotesJSON,
		guidelinesJSON, code.DRGRelevant, code.ValidFrom, code.ValidTo,
		code.Version, code.LoadedAt, code.LoadedBy, metadataJSON,
	)

	return err
}

// parseAgeRange parses age range text into structured data
func (l *ICD10AMLoader) parseAgeRange(ageText string) *AgeRange {
	ageRange := &AgeRange{
		AgeText: ageText,
		AgeUnit: "years", // default
	}

	// Parse common patterns
	// Examples: "0-17 years", "18+ years", "newborn", "adult"
	agePattern := regexp.MustCompile(`(\d+)(?:-(\d+))?\s*(year|month|day|week)s?`)
	matches := agePattern.FindStringSubmatch(strings.ToLower(ageText))

	if len(matches) >= 2 {
		if minAge, err := strconv.Atoi(matches[1]); err == nil {
			ageRange.MinAge = &minAge
		}

		if len(matches) >= 3 && matches[2] != "" {
			if maxAge, err := strconv.Atoi(matches[2]); err == nil {
				ageRange.MaxAge = &maxAge
			}
		}

		if len(matches) >= 4 && matches[3] != "" {
			ageRange.AgeUnit = matches[3] + "s"
		}
	}

	return ageRange
}

// processCSVCodeFile processes CSV-formatted ICD-10-AM files
func (l *ICD10AMLoader) processCSVCodeFile(ctx context.Context, file *ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	csvFile, err := os.Open(file.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	l.Logger.Debug("CSV header", "fields", header)

	// Process records
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
			l.Logger.Warn("Failed to read CSV record", "error", err)
			continue
		}

		// Parse record based on CSV structure
		code, err := l.parseCSVRecord(record, header)
		if err != nil {
			l.Logger.Warn("Failed to parse CSV record", "record", record, "error", err)
			l.LoadMetrics.ErrorCodes++
			continue
		}

		if err := l.insertICD10AMCode(ctx, code); err != nil {
			l.Logger.Error("Failed to insert CSV code", "code", code.Code, "error", err)
			l.LoadMetrics.ErrorCodes++
			continue
		}

		l.LoadMetrics.TotalCodes++
		l.LoadMetrics.InsertedCodes++
	}

	return nil
}

// parseCSVRecord parses a CSV record into an ICD-10-AM code
func (l *ICD10AMLoader) parseCSVRecord(record []string, header []string) (*ICD10AMCode, error) {
	if len(record) != len(header) {
		return nil, fmt.Errorf("record length %d doesn't match header length %d", len(record), len(header))
	}

	// Create field map
	fields := make(map[string]string)
	for i, fieldName := range header {
		fields[strings.ToLower(fieldName)] = record[i]
	}

	// Extract required fields
	code := &ICD10AMCode{
		ID:        uuid.New().String(),
		Code:      fields["code"],
		Title:     fields["title"],
		LoadedAt:  time.Now(),
		LoadedBy:  "icd10am-loader",
		Metadata:  make(map[string]string),
		ValidFrom: time.Now(),
	}

	// Parse optional fields
	if desc, exists := fields["description"]; exists {
		code.Description = desc
	}

	if chapter, exists := fields["chapter"]; exists {
		if chapterNum, err := strconv.Atoi(chapter); err == nil {
			code.Chapter = chapterNum
		}
	}

	if category, exists := fields["category"]; exists {
		code.Category = category
	}

	if codeType, exists := fields["type"]; exists {
		code.CodeType = codeType
	} else {
		code.CodeType = "code"
	}

	if level, exists := fields["level"]; exists {
		if levelNum, err := strconv.Atoi(level); err == nil {
			code.Level = levelNum
		}
	}

	// Validate required fields
	if code.Code == "" {
		return nil, fmt.Errorf("code cannot be empty")
	}

	return code, nil
}

// processTextCodeFile processes text-formatted ICD-10-AM files
func (l *ICD10AMLoader) processTextCodeFile(ctx context.Context, file *ICD10AMDataFile, config *ICD10AMLoadConfig) error {
	textFile, err := os.Open(file.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open text file: %w", err)
	}
	defer textFile.Close()

	scanner := bufio.NewScanner(textFile)

	// Process line by line
	lineNumber := 0
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		lineNumber++
		line := scanner.Text()

		// Skip empty lines and comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Parse line based on expected format
		code, err := l.parseTextLine(line, lineNumber)
		if err != nil {
			l.Logger.Warn("Failed to parse text line", "line", lineNumber, "content", line, "error", err)
			l.LoadMetrics.ErrorCodes++
			continue
		}

		if code != nil {
			if err := l.insertICD10AMCode(ctx, code); err != nil {
				l.Logger.Error("Failed to insert text code", "code", code.Code, "error", err)
				l.LoadMetrics.ErrorCodes++
				continue
			}

			l.LoadMetrics.TotalCodes++
			l.LoadMetrics.InsertedCodes++
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error scanning text file: %w", err)
	}

	return nil
}

// parseTextLine parses a single line from a text file
func (l *ICD10AMLoader) parseTextLine(line string, lineNumber int) (*ICD10AMCode, error) {
	// Expected format: CODE|TITLE|DESCRIPTION|CHAPTER|...
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("insufficient fields in line")
	}

	code := &ICD10AMCode{
		ID:        uuid.New().String(),
		Code:      strings.TrimSpace(parts[0]),
		Title:     strings.TrimSpace(parts[1]),
		LoadedAt:  time.Now(),
		LoadedBy:  "icd10am-loader",
		Metadata:  make(map[string]string),
		ValidFrom: time.Now(),
		CodeType:  "code",
	}

	if len(parts) > 2 {
		code.Description = strings.TrimSpace(parts[2])
	}

	if len(parts) > 3 {
		if chapter, err := strconv.Atoi(strings.TrimSpace(parts[3])); err == nil {
			code.Chapter = chapter
		}
	}

	// Additional metadata
	code.Metadata["source_line"] = strconv.Itoa(lineNumber)

	return code, nil
}

// Additional helper methods would continue here...
// Including processChapterFile, processModifierFile, createDefaultChapters,
// updateHierarchies, truncateICD10AMTables, calculatePerformanceMetrics,
// and various serialization methods

// serializeStringArray serializes a string array to JSON
func (l *ICD10AMLoader) serializeStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}

	var parts []string
	for _, item := range arr {
		parts = append(parts, `"`+strings.ReplaceAll(item, `"`, `\"`)+`"`)
	}

	return "[" + strings.Join(parts, ",") + "]"
}

// serializeMetadata serializes metadata to JSON string
func (l *ICD10AMLoader) serializeMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return "{}"
	}

	var parts []string
	for k, v := range metadata {
		parts = append(parts, fmt.Sprintf(`"%s":"%s"`, k, v))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// serializeAgeRange serializes age range to JSON string
func (l *ICD10AMLoader) serializeAgeRange(ageRange *AgeRange) string {
	if ageRange == nil {
		return "{}"
	}

	parts := []string{
		fmt.Sprintf(`"age_text":"%s"`, ageRange.AgeText),
		fmt.Sprintf(`"age_unit":"%s"`, ageRange.AgeUnit),
	}

	if ageRange.MinAge != nil {
		parts = append(parts, fmt.Sprintf(`"min_age":%d`, *ageRange.MinAge))
	}

	if ageRange.MaxAge != nil {
		parts = append(parts, fmt.Sprintf(`"max_age":%d`, *ageRange.MaxAge))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// truncateICD10AMTables truncates existing ICD-10-AM tables
func (l *ICD10AMLoader) truncateICD10AMTables(ctx context.Context) error {
	tables := []string{
		"icd10am_modifiers",
		"icd10am_codes",
		"icd10am_chapters",
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

// createDefaultChapters creates default ICD-10-AM chapters
func (l *ICD10AMLoader) createDefaultChapters(ctx context.Context) error {
	defaultChapters := []ICD10AMChapter{
		{Number: 1, Title: "Certain infectious and parasitic diseases", CodeRange: "A00-B99"},
		{Number: 2, Title: "Neoplasms", CodeRange: "C00-D48"},
		{Number: 3, Title: "Diseases of the blood and blood-forming organs and certain disorders involving the immune mechanism", CodeRange: "D50-D89"},
		// ... additional chapters would be defined here
	}

	for _, chapter := range defaultChapters {
		query := `
			INSERT INTO icd10am_chapters (number, title, code_range, description, notes, valid_from)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (number) DO UPDATE SET
				title = EXCLUDED.title,
				code_range = EXCLUDED.code_range,
				description = EXCLUDED.description,
				notes = EXCLUDED.notes,
				valid_from = EXCLUDED.valid_from
		`

		_, err := l.DB.ExecContext(ctx, query,
			chapter.Number,
			chapter.Title,
			chapter.CodeRange,
			chapter.Description,
			l.serializeStringArray(chapter.Notes),
			time.Now(),
		)

		if err != nil {
			return fmt.Errorf("failed to insert chapter %d: %w", chapter.Number, err)
		}
	}

	return nil
}

// updateHierarchies updates code hierarchies and relationships
func (l *ICD10AMLoader) updateHierarchies(ctx context.Context) error {
	l.Logger.Info("Updating ICD-10-AM code hierarchies")

	// Update parent-child relationships based on code patterns
	query := `
		UPDATE icd10am_codes
		SET parent_code = CASE
			WHEN LENGTH(code) = 4 AND SUBSTRING(code, 4, 1) != '.' THEN SUBSTRING(code, 1, 3)
			WHEN LENGTH(code) = 5 AND SUBSTRING(code, 4, 1) = '.' THEN SUBSTRING(code, 1, 3)
			WHEN LENGTH(code) > 5 THEN SUBSTRING(code, 1, 4)
			ELSE NULL
		END,
		level = CASE
			WHEN LENGTH(code) = 3 THEN 1
			WHEN LENGTH(code) = 4 OR (LENGTH(code) = 5 AND SUBSTRING(code, 4, 1) = '.') THEN 2
			WHEN LENGTH(code) > 5 THEN 3
			ELSE 1
		END,
		is_leaf = NOT EXISTS (
			SELECT 1 FROM icd10am_codes c2
			WHERE c2.parent_code = icd10am_codes.code
		)
		WHERE parent_code IS NULL OR level = 0
	`

	_, err := l.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to update hierarchies: %w", err)
	}

	return nil
}

// calculatePerformanceMetrics calculates performance metrics
func (l *ICD10AMLoader) calculatePerformanceMetrics() {
	if l.LoadMetrics.Duration > 0 {
		l.LoadMetrics.PerformanceMetrics.CodesPerSecond = float64(l.LoadMetrics.TotalCodes) / l.LoadMetrics.Duration.Seconds()
	}
}

// GetLoadMetrics returns the current load metrics
func (l *ICD10AMLoader) GetLoadMetrics() *LoadMetrics {
	return l.LoadMetrics
}