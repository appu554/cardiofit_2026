// Package dailymed provides storage layer for SPL documents.
//
// Phase 3a.3: Storage Layer for DailyMed SPL
// Key Feature: Persist SPL documents and sections to PostgreSQL + S3
//
// Database Tables:
// - source_documents: Raw SPL document metadata and storage paths
// - source_sections: Parsed LOINC sections with KB routing
package dailymed

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// =============================================================================
// STORAGE CONFIGURATION
// =============================================================================

// StorageConfig contains configuration for the storage layer
type StorageConfig struct {
	// PostgreSQL connection
	DatabaseURL string

	// File storage (S3 or local filesystem)
	StorageType    string // "s3" or "filesystem"
	StoragePath    string // Base path for file storage
	S3Bucket       string // S3 bucket name (if using S3)
	S3Prefix       string // S3 key prefix
	S3Region       string // AWS region

	// Options
	EnableCompression bool          // Compress XML before storage
	RetentionDays     int           // How long to keep old versions
	MaxConcurrency    int           // Max concurrent DB operations
	QueryTimeout      time.Duration // Database query timeout
}

// DefaultStorageConfig returns sensible defaults
func DefaultStorageConfig() StorageConfig {
	return StorageConfig{
		StorageType:       "filesystem",
		StoragePath:       "/var/lib/dailymed/spl",
		EnableCompression: true,
		RetentionDays:     365,
		MaxConcurrency:    10,
		QueryTimeout:      30 * time.Second,
	}
}

// =============================================================================
// DATABASE MODELS
// =============================================================================

// SourceDocument represents a row in the source_documents table
type SourceDocument struct {
	ID                uuid.UUID
	SourceType        string
	SourceAuthority   string
	SourceJurisdiction string

	// SPL-specific identifiers
	SetID         string
	SPLVersion    int
	DocumentID    string

	// Document metadata
	Title         string
	EffectiveDate *time.Time
	PublishedDate *time.Time

	// Drug identifiers
	DrugName   string
	RxCUI      string
	NDCCodes   []string

	// Content storage
	RawXMLPath   string
	RawXMLHash   string
	RawXMLSize   int64

	// Parsed content cache
	ParsedHeader  json.RawMessage
	ParsedAt      *time.Time
	ParserVersion string

	// Sync tracking
	DailyMedLastUpdated *time.Time
	FetchedAt           time.Time
	FetchSource         string // "API", "BULK_FTP", "DELTA"

	// Status
	Status       string // "FETCHED", "PARSED", "ERROR"
	ErrorMessage string
}

// SourceSection represents a row in the source_sections table
type SourceSection struct {
	ID               uuid.UUID
	SourceDocumentID uuid.UUID

	// Section identification
	SectionID       string
	LOINCCode       string
	LOINCDisplay    string
	SectionTitle    string
	SectionSequence int

	// Parent section
	ParentSectionID *uuid.UUID
	NestingLevel    int

	// Content
	RawXML     string
	RawText    string
	HasTables  bool
	TableCount int

	// Structured extraction
	TablesJSON json.RawMessage
	ListsJSON  json.RawMessage

	// KB routing
	TargetKBs          []string
	ExtractionPriority string

	// Processing status
	ExtractedAt time.Time
}

// =============================================================================
// STORAGE MANAGER
// =============================================================================

// StorageManager handles persistence of SPL documents and sections
type StorageManager struct {
	config StorageConfig
	db     *sql.DB
}

// NewStorageManager creates a new storage manager
func NewStorageManager(config StorageConfig) (*StorageManager, error) {
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConcurrency)
	db.SetMaxIdleConns(config.MaxConcurrency / 2)
	db.SetConnMaxLifetime(time.Hour)

	sm := &StorageManager{
		config: config,
		db:     db,
	}

	return sm, nil
}

// Close closes database connections
func (sm *StorageManager) Close() error {
	return sm.db.Close()
}

// =============================================================================
// DOCUMENT STORAGE
// =============================================================================

// SaveDocument persists an SPL document to storage
func (sm *StorageManager) SaveDocument(ctx context.Context, doc *SPLDocument, rawXML []byte, fetchSource string) (*SourceDocument, error) {
	// Generate UUID
	docID := uuid.New()

	// Save raw XML to file storage
	xmlPath, err := sm.saveRawXML(doc.SetID.Root, doc.VersionNumber.Value, rawXML)
	if err != nil {
		return nil, fmt.Errorf("saving raw XML: %w", err)
	}

	// Parse effective time
	var effectiveDate *time.Time
	if doc.EffectiveTime.Value != "" {
		t, err := parseHL7Time(doc.EffectiveTime.Value)
		if err == nil {
			effectiveDate = &t
		}
	}

	// Build source document
	sourceDoc := &SourceDocument{
		ID:                 docID,
		SourceType:         "FDA_SPL",
		SourceAuthority:    "FDA",
		SourceJurisdiction: "US",
		SetID:              doc.SetID.Root,
		SPLVersion:         doc.VersionNumber.Value,
		DocumentID:         doc.ID.Extension,
		Title:              doc.Title,
		EffectiveDate:      effectiveDate,
		RawXMLPath:         xmlPath,
		RawXMLHash:         doc.ContentHash,
		RawXMLSize:         int64(len(rawXML)),
		FetchedAt:          time.Now(),
		FetchSource:        fetchSource,
		Status:             "FETCHED",
	}

	// Insert into database
	query := `
		INSERT INTO source_documents (
			id, source_type, source_authority, source_jurisdiction,
			set_id, spl_version, document_id, title, effective_date,
			raw_xml_path, raw_xml_hash, raw_xml_size,
			fetched_at, fetch_source, status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (set_id, spl_version) DO UPDATE SET
			raw_xml_path = EXCLUDED.raw_xml_path,
			raw_xml_hash = EXCLUDED.raw_xml_hash,
			raw_xml_size = EXCLUDED.raw_xml_size,
			fetched_at = EXCLUDED.fetched_at,
			fetch_source = EXCLUDED.fetch_source,
			status = EXCLUDED.status
		RETURNING id`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	err = sm.db.QueryRowContext(ctx, query,
		sourceDoc.ID, sourceDoc.SourceType, sourceDoc.SourceAuthority, sourceDoc.SourceJurisdiction,
		sourceDoc.SetID, sourceDoc.SPLVersion, sourceDoc.DocumentID, sourceDoc.Title, sourceDoc.EffectiveDate,
		sourceDoc.RawXMLPath, sourceDoc.RawXMLHash, sourceDoc.RawXMLSize,
		sourceDoc.FetchedAt, sourceDoc.FetchSource, sourceDoc.Status,
	).Scan(&sourceDoc.ID)

	if err != nil {
		return nil, fmt.Errorf("inserting document: %w", err)
	}

	return sourceDoc, nil
}

// SaveRoutedSections persists routed sections to the database
func (sm *StorageManager) SaveRoutedSections(ctx context.Context, docID uuid.UUID, sections []*RoutedSection) error {
	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout*time.Duration(len(sections)))
	defer cancel()

	tx, err := sm.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO source_sections (
			id, source_document_id, section_id, loinc_code, loinc_display,
			section_title, section_sequence, nesting_level,
			raw_text, has_tables, table_count, tables_json,
			target_kbs, extraction_priority, extracted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (source_document_id, section_id) DO UPDATE SET
			raw_text = EXCLUDED.raw_text,
			has_tables = EXCLUDED.has_tables,
			table_count = EXCLUDED.table_count,
			tables_json = EXCLUDED.tables_json,
			target_kbs = EXCLUDED.target_kbs,
			extraction_priority = EXCLUDED.extraction_priority,
			extracted_at = EXCLUDED.extracted_at`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for i, section := range sections {
		sectionID := uuid.New()

		// Serialize tables to JSON
		var tablesJSON json.RawMessage
		if len(section.ExtractedTables) > 0 {
			tablesJSON, _ = json.Marshal(section.ExtractedTables)
		}

		// Get section ID from XML
		xmlSectionID := section.Section.ID.Extension
		if xmlSectionID == "" {
			xmlSectionID = section.Section.Code.Code + "_" + fmt.Sprintf("%d", i)
		}

		_, err := stmt.ExecContext(ctx,
			sectionID, docID, xmlSectionID,
			section.Section.Code.Code, section.Section.Code.DisplayName,
			section.Section.Title, i, section.NestingLevel,
			section.PlainText, section.HasTables, section.TableCount, tablesJSON,
			pq.Array(section.TargetKBs), section.Priority, time.Now(),
		)

		if err != nil {
			return fmt.Errorf("inserting section %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// =============================================================================
// DOCUMENT RETRIEVAL
// =============================================================================

// GetDocumentBySetID retrieves a document by its SetID
func (sm *StorageManager) GetDocumentBySetID(ctx context.Context, setID string) (*SourceDocument, error) {
	query := `
		SELECT id, source_type, source_authority, source_jurisdiction,
			set_id, spl_version, document_id, title, effective_date, published_date,
			drug_name, rxcui, ndc_codes,
			raw_xml_path, raw_xml_hash, raw_xml_size,
			parsed_header, parsed_at, parser_version,
			dailymed_last_updated, fetched_at, fetch_source,
			status, error_message
		FROM source_documents
		WHERE set_id = $1
		ORDER BY spl_version DESC
		LIMIT 1`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	doc := &SourceDocument{}
	err := sm.db.QueryRowContext(ctx, query, setID).Scan(
		&doc.ID, &doc.SourceType, &doc.SourceAuthority, &doc.SourceJurisdiction,
		&doc.SetID, &doc.SPLVersion, &doc.DocumentID, &doc.Title, &doc.EffectiveDate, &doc.PublishedDate,
		&doc.DrugName, &doc.RxCUI, pq.Array(&doc.NDCCodes),
		&doc.RawXMLPath, &doc.RawXMLHash, &doc.RawXMLSize,
		&doc.ParsedHeader, &doc.ParsedAt, &doc.ParserVersion,
		&doc.DailyMedLastUpdated, &doc.FetchedAt, &doc.FetchSource,
		&doc.Status, &doc.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, fmt.Errorf("querying document: %w", err)
	}

	return doc, nil
}

// GetDocumentByRxCUI retrieves a document by RxCUI
func (sm *StorageManager) GetDocumentByRxCUI(ctx context.Context, rxcui string) (*SourceDocument, error) {
	query := `
		SELECT id, source_type, source_authority, source_jurisdiction,
			set_id, spl_version, document_id, title, effective_date, published_date,
			drug_name, rxcui, ndc_codes,
			raw_xml_path, raw_xml_hash, raw_xml_size,
			parsed_header, parsed_at, parser_version,
			dailymed_last_updated, fetched_at, fetch_source,
			status, error_message
		FROM source_documents
		WHERE rxcui = $1
		ORDER BY spl_version DESC
		LIMIT 1`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	doc := &SourceDocument{}
	err := sm.db.QueryRowContext(ctx, query, rxcui).Scan(
		&doc.ID, &doc.SourceType, &doc.SourceAuthority, &doc.SourceJurisdiction,
		&doc.SetID, &doc.SPLVersion, &doc.DocumentID, &doc.Title, &doc.EffectiveDate, &doc.PublishedDate,
		&doc.DrugName, &doc.RxCUI, pq.Array(&doc.NDCCodes),
		&doc.RawXMLPath, &doc.RawXMLHash, &doc.RawXMLSize,
		&doc.ParsedHeader, &doc.ParsedAt, &doc.ParserVersion,
		&doc.DailyMedLastUpdated, &doc.FetchedAt, &doc.FetchSource,
		&doc.Status, &doc.ErrorMessage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying document: %w", err)
	}

	return doc, nil
}

// GetSectionsByDocument retrieves all sections for a document
func (sm *StorageManager) GetSectionsByDocument(ctx context.Context, docID uuid.UUID) ([]*SourceSection, error) {
	query := `
		SELECT id, source_document_id, section_id, loinc_code, loinc_display,
			section_title, section_sequence, parent_section_id, nesting_level,
			raw_xml, raw_text, has_tables, table_count,
			tables_json, lists_json, target_kbs, extraction_priority, extracted_at
		FROM source_sections
		WHERE source_document_id = $1
		ORDER BY section_sequence`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	rows, err := sm.db.QueryContext(ctx, query, docID)
	if err != nil {
		return nil, fmt.Errorf("querying sections: %w", err)
	}
	defer rows.Close()

	var sections []*SourceSection
	for rows.Next() {
		section := &SourceSection{}
		err := rows.Scan(
			&section.ID, &section.SourceDocumentID, &section.SectionID,
			&section.LOINCCode, &section.LOINCDisplay,
			&section.SectionTitle, &section.SectionSequence,
			&section.ParentSectionID, &section.NestingLevel,
			&section.RawXML, &section.RawText, &section.HasTables, &section.TableCount,
			&section.TablesJSON, &section.ListsJSON,
			pq.Array(&section.TargetKBs), &section.ExtractionPriority, &section.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning section: %w", err)
		}
		sections = append(sections, section)
	}

	return sections, nil
}

// GetSectionsByLOINC retrieves all sections with a specific LOINC code
func (sm *StorageManager) GetSectionsByLOINC(ctx context.Context, loincCode string) ([]*SourceSection, error) {
	query := `
		SELECT s.id, s.source_document_id, s.section_id, s.loinc_code, s.loinc_display,
			s.section_title, s.section_sequence, s.parent_section_id, s.nesting_level,
			s.raw_xml, s.raw_text, s.has_tables, s.table_count,
			s.tables_json, s.lists_json, s.target_kbs, s.extraction_priority, s.extracted_at
		FROM source_sections s
		JOIN source_documents d ON s.source_document_id = d.id
		WHERE s.loinc_code = $1 AND d.status = 'PARSED'
		ORDER BY d.published_date DESC`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	rows, err := sm.db.QueryContext(ctx, query, loincCode)
	if err != nil {
		return nil, fmt.Errorf("querying sections: %w", err)
	}
	defer rows.Close()

	var sections []*SourceSection
	for rows.Next() {
		section := &SourceSection{}
		err := rows.Scan(
			&section.ID, &section.SourceDocumentID, &section.SectionID,
			&section.LOINCCode, &section.LOINCDisplay,
			&section.SectionTitle, &section.SectionSequence,
			&section.ParentSectionID, &section.NestingLevel,
			&section.RawXML, &section.RawText, &section.HasTables, &section.TableCount,
			&section.TablesJSON, &section.ListsJSON,
			pq.Array(&section.TargetKBs), &section.ExtractionPriority, &section.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning section: %w", err)
		}
		sections = append(sections, section)
	}

	return sections, nil
}

// GetSectionsForKB retrieves all sections targeted at a specific KB
func (sm *StorageManager) GetSectionsForKB(ctx context.Context, kbName string) ([]*SourceSection, error) {
	query := `
		SELECT s.id, s.source_document_id, s.section_id, s.loinc_code, s.loinc_display,
			s.section_title, s.section_sequence, s.parent_section_id, s.nesting_level,
			s.raw_xml, s.raw_text, s.has_tables, s.table_count,
			s.tables_json, s.lists_json, s.target_kbs, s.extraction_priority, s.extracted_at
		FROM source_sections s
		JOIN source_documents d ON s.source_document_id = d.id
		WHERE $1 = ANY(s.target_kbs) AND d.status = 'PARSED'
		ORDER BY s.extraction_priority, d.published_date DESC`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	rows, err := sm.db.QueryContext(ctx, query, kbName)
	if err != nil {
		return nil, fmt.Errorf("querying sections: %w", err)
	}
	defer rows.Close()

	var sections []*SourceSection
	for rows.Next() {
		section := &SourceSection{}
		err := rows.Scan(
			&section.ID, &section.SourceDocumentID, &section.SectionID,
			&section.LOINCCode, &section.LOINCDisplay,
			&section.SectionTitle, &section.SectionSequence,
			&section.ParentSectionID, &section.NestingLevel,
			&section.RawXML, &section.RawText, &section.HasTables, &section.TableCount,
			&section.TablesJSON, &section.ListsJSON,
			pq.Array(&section.TargetKBs), &section.ExtractionPriority, &section.ExtractedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning section: %w", err)
		}
		sections = append(sections, section)
	}

	return sections, nil
}

// =============================================================================
// SYNC TRACKING
// =============================================================================

// GetLastSyncTime returns the most recent fetch timestamp
func (sm *StorageManager) GetLastSyncTime(ctx context.Context) (time.Time, error) {
	query := `
		SELECT COALESCE(MAX(fetched_at), '1970-01-01'::timestamptz)
		FROM source_documents
		WHERE source_type = 'FDA_SPL'`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	var lastSync time.Time
	err := sm.db.QueryRowContext(ctx, query).Scan(&lastSync)
	if err != nil {
		return time.Time{}, fmt.Errorf("getting last sync time: %w", err)
	}

	return lastSync, nil
}

// GetDocumentHashBySetID returns the content hash for a SetID if it exists
func (sm *StorageManager) GetDocumentHashBySetID(ctx context.Context, setID string, version int) (string, bool, error) {
	query := `
		SELECT raw_xml_hash
		FROM source_documents
		WHERE set_id = $1 AND spl_version = $2`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	var hash string
	err := sm.db.QueryRowContext(ctx, query, setID, version).Scan(&hash)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("getting document hash: %w", err)
	}

	return hash, true, nil
}

// UpdateDocumentStatus updates the status of a document
func (sm *StorageManager) UpdateDocumentStatus(ctx context.Context, docID uuid.UUID, status string, errorMsg string) error {
	query := `
		UPDATE source_documents
		SET status = $2, error_message = $3, parsed_at = CASE WHEN $2 = 'PARSED' THEN NOW() ELSE parsed_at END
		WHERE id = $1`

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout)
	defer cancel()

	_, err := sm.db.ExecContext(ctx, query, docID, status, errorMsg)
	if err != nil {
		return fmt.Errorf("updating status: %w", err)
	}

	return nil
}

// =============================================================================
// RAW XML FILE STORAGE
// =============================================================================

func (sm *StorageManager) saveRawXML(setID string, version int, rawXML []byte) (string, error) {
	if sm.config.StorageType == "s3" {
		return sm.saveToS3(setID, version, rawXML)
	}
	return sm.saveToFilesystem(setID, version, rawXML)
}

func (sm *StorageManager) saveToFilesystem(setID string, version int, rawXML []byte) (string, error) {
	// Create directory structure: /base/se/tid/setid_v{version}.xml
	prefix := setID[:2]
	if len(setID) < 2 {
		prefix = setID
	}

	dir := filepath.Join(sm.config.StoragePath, prefix, setID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating directory: %w", err)
	}

	filename := fmt.Sprintf("%s_v%d.xml", setID, version)
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, rawXML, 0644); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return path, nil
}

func (sm *StorageManager) saveToS3(setID string, version int, rawXML []byte) (string, error) {
	// S3 storage implementation would go here
	// Using aws-sdk-go to upload to S3
	key := fmt.Sprintf("%s/%s/%s_v%d.xml", sm.config.S3Prefix, setID[:2], setID, version)

	// Placeholder - actual S3 implementation would use aws-sdk-go
	return fmt.Sprintf("s3://%s/%s", sm.config.S3Bucket, key), nil
}

// LoadRawXML loads raw XML from storage
func (sm *StorageManager) LoadRawXML(path string) ([]byte, error) {
	if strings.HasPrefix(path, "s3://") {
		return sm.loadFromS3(path)
	}
	return sm.loadFromFilesystem(path)
}

func (sm *StorageManager) loadFromFilesystem(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	return io.ReadAll(file)
}

func (sm *StorageManager) loadFromS3(path string) ([]byte, error) {
	// S3 loading implementation would go here
	// Placeholder - actual S3 implementation would use aws-sdk-go
	return nil, fmt.Errorf("S3 loading not implemented")
}

// =============================================================================
// HELPERS
// =============================================================================

// parseHL7Time parses HL7 date format (YYYYMMDD or YYYYMMDDHHMMSS)
func parseHL7Time(value string) (time.Time, error) {
	value = strings.TrimSpace(value)

	// Try different formats
	formats := []string{
		"20060102150405",     // Full timestamp
		"200601021504",       // Without seconds
		"20060102",           // Date only
		"2006-01-02",         // ISO date
		"2006-01-02T15:04:05", // ISO timestamp
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse HL7 time: %s", value)
}

// =============================================================================
// MIGRATIONS
// =============================================================================

// CreateTables creates the required database tables
func (sm *StorageManager) CreateTables(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS source_documents (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			source_type VARCHAR(50) NOT NULL DEFAULT 'FDA_SPL',
			source_authority VARCHAR(50) NOT NULL DEFAULT 'FDA',
			source_jurisdiction VARCHAR(10) NOT NULL DEFAULT 'US',
			set_id VARCHAR(100) NOT NULL,
			spl_version INTEGER NOT NULL,
			document_id VARCHAR(100),
			title VARCHAR(1000),
			effective_date DATE,
			published_date DATE,
			drug_name VARCHAR(500),
			rxcui VARCHAR(20),
			ndc_codes TEXT[],
			raw_xml_path TEXT NOT NULL,
			raw_xml_hash VARCHAR(64) NOT NULL,
			raw_xml_size BIGINT,
			parsed_header JSONB,
			parsed_at TIMESTAMPTZ,
			parser_version VARCHAR(20),
			dailymed_last_updated TIMESTAMPTZ,
			fetched_at TIMESTAMPTZ DEFAULT NOW(),
			fetch_source VARCHAR(50),
			status VARCHAR(20) DEFAULT 'FETCHED',
			error_message TEXT,
			UNIQUE(set_id, spl_version)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_source_docs_set_id ON source_documents(set_id)`,
		`CREATE INDEX IF NOT EXISTS idx_source_docs_rxcui ON source_documents(rxcui)`,
		`CREATE INDEX IF NOT EXISTS idx_source_docs_status ON source_documents(status)`,
		`CREATE INDEX IF NOT EXISTS idx_source_docs_published ON source_documents(published_date DESC)`,

		`CREATE TABLE IF NOT EXISTS source_sections (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			source_document_id UUID NOT NULL REFERENCES source_documents(id) ON DELETE CASCADE,
			section_id VARCHAR(100),
			loinc_code VARCHAR(20) NOT NULL,
			loinc_display VARCHAR(200),
			section_title VARCHAR(500),
			section_sequence INTEGER,
			parent_section_id UUID REFERENCES source_sections(id),
			nesting_level INTEGER DEFAULT 0,
			raw_xml TEXT,
			raw_text TEXT,
			has_tables BOOLEAN DEFAULT FALSE,
			table_count INTEGER DEFAULT 0,
			tables_json JSONB,
			lists_json JSONB,
			target_kbs TEXT[] NOT NULL,
			extraction_priority VARCHAR(20),
			extracted_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(source_document_id, section_id)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_sections_document ON source_sections(source_document_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_loinc ON source_sections(loinc_code)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_target_kbs ON source_sections USING GIN (target_kbs)`,
		`CREATE INDEX IF NOT EXISTS idx_sections_has_tables ON source_sections(has_tables) WHERE has_tables = TRUE`,

		`CREATE TABLE IF NOT EXISTS sync_log (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			source_type VARCHAR(50) NOT NULL,
			started_at TIMESTAMPTZ NOT NULL,
			completed_at TIMESTAMPTZ,
			documents_synced INTEGER DEFAULT 0,
			documents_skipped INTEGER DEFAULT 0,
			documents_failed INTEGER DEFAULT 0,
			errors JSONB,
			sync_strategy VARCHAR(20)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_sync_log_source ON sync_log(source_type, started_at DESC)`,
	}

	ctx, cancel := context.WithTimeout(ctx, sm.config.QueryTimeout*time.Duration(len(queries)))
	defer cancel()

	for _, query := range queries {
		if _, err := sm.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("executing migration: %w", err)
		}
	}

	return nil
}
