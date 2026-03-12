// Package factstore provides the FactStore Repository for database operations.
// This is LAYER 4 (Derived Facts Store) + LAYER 6 (FactStore) integration.
//
// DESIGN PRINCIPLE: Source-Centric Data Model
// - All facts have complete lineage: source_document → source_section → derived_fact
// - KB-1, KB-4, KB-5, KB-6 are SEMANTIC DOMAINS, not separate databases
// - Single unified store with domain routing
//
// Phase 3 Implementation
package factstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// =============================================================================
// REPOSITORY
// =============================================================================

// Repository provides database operations for the FactStore
type Repository struct {
	db  *sql.DB
	log *logrus.Entry
}

// NewRepository creates a new FactStore repository
func NewRepository(db *sql.DB, log *logrus.Entry) *Repository {
	return &Repository{
		db:  db,
		log: log.WithField("component", "factstore-repository"),
	}
}

// =============================================================================
// SOURCE DOCUMENT OPERATIONS
// =============================================================================

// SourceDocument represents a raw document from an authoritative source
type SourceDocument struct {
	ID               string    `json:"id"`
	SourceType       string    `json:"sourceType"`       // FDA_SPL, CPIC, CREDIBLEMEDS, etc.
	DocumentID       string    `json:"documentId"`       // SetID for SPL, PMID for literature
	VersionNumber    string    `json:"versionNumber"`
	RawContentHash   string    `json:"rawContentHash"`
	FetchedAt        time.Time `json:"fetchedAt"`
	ContentUpdatedAt *time.Time `json:"contentUpdatedAt,omitempty"`
	DrugName         string    `json:"drugName"`
	GenericName      string    `json:"genericName"`
	RxCUI            string    `json:"rxcui"`
	NDCCodes         []string  `json:"ndcCodes,omitempty"`
	ATCCodes         []string  `json:"atcCodes,omitempty"`
	EffectiveDate    *time.Time `json:"effectiveDate,omitempty"`
	Manufacturer     string    `json:"manufacturer,omitempty"`
	LabelerCode      string    `json:"labelerCode,omitempty"`
	ProcessingStatus string    `json:"processingStatus"`
	ProcessingError  string    `json:"processingError,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// CreateSourceDocument inserts a new source document
func (r *Repository) CreateSourceDocument(ctx context.Context, doc *SourceDocument) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	if doc.ProcessingStatus == "" {
		doc.ProcessingStatus = "PENDING"
	}

	// Ensure arrays are not nil (PostgreSQL TEXT[] needs proper array handling)
	ndcCodes := doc.NDCCodes
	if ndcCodes == nil {
		ndcCodes = []string{}
	}
	atcCodes := doc.ATCCodes
	if atcCodes == nil {
		atcCodes = []string{}
	}

	query := `
		INSERT INTO source_documents (
			id, source_type, document_id, version_number, raw_content_hash,
			fetched_at, content_updated_at, drug_name, generic_name, rxcui,
			ndc_codes, atc_codes, effective_date, manufacturer, labeler_code,
			processing_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		ON CONFLICT (source_type, document_id, version_number) DO UPDATE SET
			raw_content_hash = EXCLUDED.raw_content_hash,
			fetched_at = EXCLUDED.fetched_at,
			drug_name = EXCLUDED.drug_name,
			generic_name = EXCLUDED.generic_name,
			rxcui = EXCLUDED.rxcui,
			ndc_codes = EXCLUDED.ndc_codes,
			atc_codes = EXCLUDED.atc_codes,
			manufacturer = EXCLUDED.manufacturer,
			processing_status = EXCLUDED.processing_status,
			updated_at = NOW()
		RETURNING id
	`

	err := r.db.QueryRowContext(ctx, query,
		doc.ID, doc.SourceType, doc.DocumentID, doc.VersionNumber, doc.RawContentHash,
		doc.FetchedAt, doc.ContentUpdatedAt, doc.DrugName, doc.GenericName, doc.RxCUI,
		pq.Array(ndcCodes), pq.Array(atcCodes), doc.EffectiveDate, doc.Manufacturer, doc.LabelerCode,
		doc.ProcessingStatus,
	).Scan(&doc.ID)

	if err != nil {
		return fmt.Errorf("failed to create source document: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"documentId": doc.DocumentID,
		"sourceType": doc.SourceType,
		"drugName":   doc.DrugName,
	}).Debug("Source document created")

	return nil
}

// GetSourceDocument retrieves a source document by ID
func (r *Repository) GetSourceDocument(ctx context.Context, id string) (*SourceDocument, error) {
	query := `
		SELECT id, source_type, document_id, version_number, raw_content_hash,
		       fetched_at, content_updated_at, drug_name, generic_name, rxcui,
		       ndc_codes, atc_codes, effective_date, manufacturer, labeler_code,
		       processing_status, processing_error, created_at, updated_at
		FROM source_documents
		WHERE id = $1
	`

	doc := &SourceDocument{}
	var ndcJSON, atcJSON []byte
	var contentUpdatedAt, effectiveDate sql.NullTime
	var processingError sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID, &doc.SourceType, &doc.DocumentID, &doc.VersionNumber, &doc.RawContentHash,
		&doc.FetchedAt, &contentUpdatedAt, &doc.DrugName, &doc.GenericName, &doc.RxCUI,
		&ndcJSON, &atcJSON, &effectiveDate, &doc.Manufacturer, &doc.LabelerCode,
		&doc.ProcessingStatus, &processingError, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get source document: %w", err)
	}

	if contentUpdatedAt.Valid {
		doc.ContentUpdatedAt = &contentUpdatedAt.Time
	}
	if effectiveDate.Valid {
		doc.EffectiveDate = &effectiveDate.Time
	}
	if processingError.Valid {
		doc.ProcessingError = processingError.String
	}
	_ = json.Unmarshal(ndcJSON, &doc.NDCCodes)
	_ = json.Unmarshal(atcJSON, &doc.ATCCodes)

	return doc, nil
}

// GetSourceDocumentBySetID retrieves a source document by SetID (for SPL)
func (r *Repository) GetSourceDocumentBySetID(ctx context.Context, setID string) (*SourceDocument, error) {
	query := `
		SELECT id, source_type, document_id, version_number, raw_content_hash,
		       fetched_at, content_updated_at, drug_name, generic_name, rxcui,
		       ndc_codes, atc_codes, effective_date, manufacturer, labeler_code,
		       processing_status, processing_error, created_at, updated_at
		FROM source_documents
		WHERE source_type = 'FDA_SPL' AND document_id = $1
		ORDER BY version_number DESC
		LIMIT 1
	`

	doc := &SourceDocument{}
	var ndcJSON, atcJSON []byte
	var contentUpdatedAt, effectiveDate sql.NullTime
	var processingError sql.NullString

	err := r.db.QueryRowContext(ctx, query, setID).Scan(
		&doc.ID, &doc.SourceType, &doc.DocumentID, &doc.VersionNumber, &doc.RawContentHash,
		&doc.FetchedAt, &contentUpdatedAt, &doc.DrugName, &doc.GenericName, &doc.RxCUI,
		&ndcJSON, &atcJSON, &effectiveDate, &doc.Manufacturer, &doc.LabelerCode,
		&doc.ProcessingStatus, &processingError, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get source document: %w", err)
	}

	if contentUpdatedAt.Valid {
		doc.ContentUpdatedAt = &contentUpdatedAt.Time
	}
	if effectiveDate.Valid {
		doc.EffectiveDate = &effectiveDate.Time
	}
	if processingError.Valid {
		doc.ProcessingError = processingError.String
	}
	_ = json.Unmarshal(ndcJSON, &doc.NDCCodes)
	_ = json.Unmarshal(atcJSON, &doc.ATCCodes)

	return doc, nil
}

// UpdateSourceDocumentStatus updates the processing status of a source document
func (r *Repository) UpdateSourceDocumentStatus(ctx context.Context, id string, status string, errorMsg string) error {
	query := `
		UPDATE source_documents
		SET processing_status = $2,
		    processing_error = $3,
		    last_processed_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	var errorPtr *string
	if errorMsg != "" {
		errorPtr = &errorMsg
	}

	_, err := r.db.ExecContext(ctx, query, id, status, errorPtr)
	return err
}

// =============================================================================
// SOURCE SECTION OPERATIONS
// =============================================================================

// SourceSection represents a parsed section from a source document
type SourceSection struct {
	ID                   string          `json:"id"`
	SourceDocumentID     string          `json:"sourceDocumentId"`
	SectionCode          string          `json:"sectionCode"`    // LOINC code
	SectionName          string          `json:"sectionName"`
	TargetKBs            []string        `json:"targetKbs"`
	RawText              string          `json:"rawText,omitempty"`
	RawHTML              string          `json:"rawHtml,omitempty"`
	ParsedTables         json.RawMessage `json:"parsedTables,omitempty"`
	ExtractionMethod     string          `json:"extractionMethod"` // TABLE_PARSE, REGEX_PARSE, LLM_GAP, AUTHORITY
	ExtractionConfidence float64         `json:"extractionConfidence"`
	HasStructuredTables  bool            `json:"hasStructuredTables"`
	TableCount           int             `json:"tableCount"`
	WordCount            int             `json:"wordCount"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

// CreateSourceSection inserts a new source section
func (r *Repository) CreateSourceSection(ctx context.Context, section *SourceSection) error {
	if section.ID == "" {
		section.ID = uuid.New().String()
	}

	// Ensure TargetKBs is not nil for PostgreSQL TEXT[] array
	targetKBs := section.TargetKBs
	if targetKBs == nil {
		targetKBs = []string{}
	}

	query := `
		INSERT INTO source_sections (
			id, source_document_id, section_code, section_name, target_kbs,
			raw_text, raw_html, parsed_tables, extraction_method, extraction_confidence,
			has_structured_tables, table_count, word_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (source_document_id, section_code) DO UPDATE SET
			section_name = EXCLUDED.section_name,
			target_kbs = EXCLUDED.target_kbs,
			raw_text = EXCLUDED.raw_text,
			raw_html = EXCLUDED.raw_html,
			parsed_tables = EXCLUDED.parsed_tables,
			extraction_method = EXCLUDED.extraction_method,
			extraction_confidence = EXCLUDED.extraction_confidence,
			has_structured_tables = EXCLUDED.has_structured_tables,
			table_count = EXCLUDED.table_count,
			word_count = EXCLUDED.word_count,
			updated_at = NOW()
		RETURNING id
	`

	// ── Sanitize text fields: strip NUL bytes and control chars ──
	// PostgreSQL text columns reject \x00; jsonb rejects \u0000.
	sanitize := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r == 0 || (r < 0x20 && r != '\n' && r != '\t' && r != '\r') {
				return -1
			}
			return r
		}, strings.ToValidUTF8(s, ""))
	}

	// ── Handle parsed_tables: explicit NULL vs valid JSON ──
	var parsedTablesParam interface{}
	switch {
	case section.ParsedTables == nil:
		parsedTablesParam = nil // SQL NULL — safe for jsonb
	case len(section.ParsedTables) == 0:
		parsedTablesParam = []byte("[]")
	default:
		parsedTablesParam = []byte(section.ParsedTables)
	}

	err := r.db.QueryRowContext(ctx, query,
		section.ID, section.SourceDocumentID, sanitize(section.SectionCode), sanitize(section.SectionName), pq.Array(targetKBs),
		sanitize(section.RawText), sanitize(section.RawHTML), parsedTablesParam, sanitize(section.ExtractionMethod),
		section.ExtractionConfidence, section.HasStructuredTables, section.TableCount, section.WordCount,
	).Scan(&section.ID)

	if err != nil {
		return fmt.Errorf("failed to create source section: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"sectionId":   section.ID,
		"sectionCode": section.SectionCode,
		"targetKBs":   section.TargetKBs,
	}).Debug("Source section created")

	return nil
}

// GetSectionsByDocument retrieves all sections for a source document
func (r *Repository) GetSectionsByDocument(ctx context.Context, documentID string) ([]*SourceSection, error) {
	query := `
		SELECT id, source_document_id, section_code, section_name, target_kbs,
		       raw_text, raw_html, parsed_tables, extraction_method, extraction_confidence,
		       has_structured_tables, table_count, word_count, created_at, updated_at
		FROM source_sections
		WHERE source_document_id = $1
		ORDER BY section_code
	`

	rows, err := r.db.QueryContext(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sections: %w", err)
	}
	defer rows.Close()

	var sections []*SourceSection
	for rows.Next() {
		section := &SourceSection{}
		var targetKBsJSON []byte
		var rawText, rawHTML sql.NullString
		var parsedTables []byte

		err := rows.Scan(
			&section.ID, &section.SourceDocumentID, &section.SectionCode, &section.SectionName,
			&targetKBsJSON, &rawText, &rawHTML, &parsedTables, &section.ExtractionMethod,
			&section.ExtractionConfidence, &section.HasStructuredTables, &section.TableCount,
			&section.WordCount, &section.CreatedAt, &section.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan section: %w", err)
		}

		_ = json.Unmarshal(targetKBsJSON, &section.TargetKBs)
		if rawText.Valid {
			section.RawText = rawText.String
		}
		if rawHTML.Valid {
			section.RawHTML = rawHTML.String
		}
		section.ParsedTables = parsedTables

		sections = append(sections, section)
	}

	return sections, nil
}

// =============================================================================
// DERIVED FACT OPERATIONS
// =============================================================================

// DerivedFact represents an extracted fact with full lineage
type DerivedFact struct {
	ID                   string          `json:"id"`
	SourceDocumentID     string          `json:"sourceDocumentId"`
	SourceSectionID      string          `json:"sourceSectionId,omitempty"`
	TargetKB             string          `json:"targetKb"`       // KB-1, KB-4, KB-5, KB-6, KB-16
	FactType             string          `json:"factType"`       // RENAL_DOSE_ADJUST, HEPATIC_DOSE, QT_RISK, etc.
	FactKey              string          `json:"factKey"`        // Unique key like "metformin:gfr_band:30-60"
	FactData             json.RawMessage `json:"factData"`       // Structured fact content
	ExtractionMethod     string          `json:"extractionMethod"` // AUTHORITY, TABLE_PARSE, REGEX_PARSE, LLM_CONSENSUS
	ExtractionConfidence float64         `json:"extractionConfidence"`
	EvidenceSpans        json.RawMessage `json:"evidenceSpans,omitempty"` // Quoted source text
	LLMProvider          string          `json:"llmProvider,omitempty"`
	LLMModel             string          `json:"llmModel,omitempty"`
	ConsensusAchieved    bool            `json:"consensusAchieved"`
	ConsensusProviders   []string        `json:"consensusProviders,omitempty"`
	GovernanceStatus     string          `json:"governanceStatus"` // DRAFT, PENDING_REVIEW, APPROVED, REJECTED, SUPERSEDED
	ReviewedBy           string          `json:"reviewedBy,omitempty"`
	ReviewedAt           *time.Time      `json:"reviewedAt,omitempty"`
	ReviewNotes          string          `json:"reviewNotes,omitempty"`
	IsActive             bool            `json:"isActive"`
	SupersededBy         string          `json:"supersededBy,omitempty"`
	Supersedes           string          `json:"supersedes,omitempty"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

// CreateDerivedFact inserts a new derived fact
func (r *Repository) CreateDerivedFact(ctx context.Context, fact *DerivedFact) error {
	if fact.ID == "" {
		fact.ID = uuid.New().String()
	}
	if fact.GovernanceStatus == "" {
		fact.GovernanceStatus = "DRAFT"
	}
	fact.IsActive = true

	// Ensure ConsensusProviders is not nil for PostgreSQL TEXT[] array
	consensusProviders := fact.ConsensusProviders
	if consensusProviders == nil {
		consensusProviders = []string{}
	}

	query := `
		INSERT INTO derived_facts (
			id, source_document_id, source_section_id, target_kb, fact_type, fact_key,
			fact_data, extraction_method, extraction_confidence, evidence_spans,
			llm_provider, llm_model, consensus_achieved, consensus_providers,
			governance_status, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
		RETURNING id, created_at
	`

	var sectionID interface{}
	if fact.SourceSectionID != "" {
		sectionID = fact.SourceSectionID
	}

	// ── Sanitize JSON fields: strip NUL bytes that PostgreSQL jsonb rejects ──
	sanitizeJSON := func(raw json.RawMessage) interface{} {
		if raw == nil {
			return nil
		}
		if len(raw) == 0 {
			return []byte("{}")
		}
		cleaned := strings.ToValidUTF8(string(raw), "")
		cleaned = strings.Map(func(r rune) rune {
			if r == 0 {
				return -1
			}
			return r
		}, cleaned)
		return []byte(cleaned)
	}

	err := r.db.QueryRowContext(ctx, query,
		fact.ID, fact.SourceDocumentID, sectionID, fact.TargetKB, fact.FactType, fact.FactKey,
		sanitizeJSON(fact.FactData), fact.ExtractionMethod, fact.ExtractionConfidence, sanitizeJSON(fact.EvidenceSpans),
		nilIfEmpty(fact.LLMProvider), nilIfEmpty(fact.LLMModel), fact.ConsensusAchieved, pq.Array(consensusProviders),
		fact.GovernanceStatus, fact.IsActive,
	).Scan(&fact.ID, &fact.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create derived fact: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"factId":   fact.ID,
		"factType": fact.FactType,
		"targetKB": fact.TargetKB,
		"method":   fact.ExtractionMethod,
	}).Debug("Derived fact created")

	return nil
}

// GetDerivedFact retrieves a derived fact by ID
func (r *Repository) GetDerivedFact(ctx context.Context, id string) (*DerivedFact, error) {
	query := `
		SELECT id, source_document_id, source_section_id, target_kb, fact_type, fact_key,
		       fact_data, extraction_method, extraction_confidence, evidence_spans,
		       llm_provider, llm_model, consensus_achieved, consensus_providers,
		       governance_status, reviewed_by, reviewed_at, review_notes,
		       is_active, superseded_by, supersedes, created_at, updated_at
		FROM derived_facts
		WHERE id = $1
	`

	fact := &DerivedFact{}
	var sectionID, llmProvider, llmModel, reviewedBy, reviewNotes, supersededBy, supersedes sql.NullString
	var reviewedAt sql.NullTime
	var consensusJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&fact.ID, &fact.SourceDocumentID, &sectionID, &fact.TargetKB, &fact.FactType, &fact.FactKey,
		&fact.FactData, &fact.ExtractionMethod, &fact.ExtractionConfidence, &fact.EvidenceSpans,
		&llmProvider, &llmModel, &fact.ConsensusAchieved, &consensusJSON,
		&fact.GovernanceStatus, &reviewedBy, &reviewedAt, &reviewNotes,
		&fact.IsActive, &supersededBy, &supersedes, &fact.CreatedAt, &fact.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get derived fact: %w", err)
	}

	if sectionID.Valid {
		fact.SourceSectionID = sectionID.String
	}
	if llmProvider.Valid {
		fact.LLMProvider = llmProvider.String
	}
	if llmModel.Valid {
		fact.LLMModel = llmModel.String
	}
	if reviewedBy.Valid {
		fact.ReviewedBy = reviewedBy.String
	}
	if reviewedAt.Valid {
		fact.ReviewedAt = &reviewedAt.Time
	}
	if reviewNotes.Valid {
		fact.ReviewNotes = reviewNotes.String
	}
	if supersededBy.Valid {
		fact.SupersededBy = supersededBy.String
	}
	if supersedes.Valid {
		fact.Supersedes = supersedes.String
	}
	_ = json.Unmarshal(consensusJSON, &fact.ConsensusProviders)

	return fact, nil
}

// GetFactsByTargetKB retrieves all active facts for a specific KB
func (r *Repository) GetFactsByTargetKB(ctx context.Context, targetKB string, limit int) ([]*DerivedFact, error) {
	query := `
		SELECT id, source_document_id, source_section_id, target_kb, fact_type, fact_key,
		       fact_data, extraction_method, extraction_confidence, governance_status,
		       is_active, created_at, updated_at
		FROM derived_facts
		WHERE target_kb = $1 AND is_active = TRUE AND governance_status = 'APPROVED'
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, targetKB, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query facts by KB: %w", err)
	}
	defer rows.Close()

	var facts []*DerivedFact
	for rows.Next() {
		fact := &DerivedFact{}
		var sectionID sql.NullString

		err := rows.Scan(
			&fact.ID, &fact.SourceDocumentID, &sectionID, &fact.TargetKB, &fact.FactType, &fact.FactKey,
			&fact.FactData, &fact.ExtractionMethod, &fact.ExtractionConfidence, &fact.GovernanceStatus,
			&fact.IsActive, &fact.CreatedAt, &fact.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}

		if sectionID.Valid {
			fact.SourceSectionID = sectionID.String
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// GetPendingFactsForGovernance retrieves facts awaiting governance processing
func (r *Repository) GetPendingFactsForGovernance(ctx context.Context, limit int) ([]*DerivedFact, error) {
	query := `
		SELECT id, source_document_id, source_section_id, target_kb, fact_type, fact_key,
		       fact_data, extraction_method, extraction_confidence, governance_status,
		       is_active, created_at, updated_at
		FROM derived_facts
		WHERE governance_status = 'DRAFT' AND is_active = TRUE
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending facts: %w", err)
	}
	defer rows.Close()

	var facts []*DerivedFact
	for rows.Next() {
		fact := &DerivedFact{}
		var sectionID sql.NullString

		err := rows.Scan(
			&fact.ID, &fact.SourceDocumentID, &sectionID, &fact.TargetKB, &fact.FactType, &fact.FactKey,
			&fact.FactData, &fact.ExtractionMethod, &fact.ExtractionConfidence, &fact.GovernanceStatus,
			&fact.IsActive, &fact.CreatedAt, &fact.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fact: %w", err)
		}

		if sectionID.Valid {
			fact.SourceSectionID = sectionID.String
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// UpdateFactGovernanceStatus updates the governance status of a fact
func (r *Repository) UpdateFactGovernanceStatus(ctx context.Context, id string, status string, reviewer string, notes string) error {
	query := `
		UPDATE derived_facts
		SET governance_status = $2,
		    reviewed_by = $3,
		    reviewed_at = NOW(),
		    review_notes = $4,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, status, nilIfEmpty(reviewer), nilIfEmpty(notes))
	if err != nil {
		return fmt.Errorf("failed to update governance status: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"factId":   id,
		"status":   status,
		"reviewer": reviewer,
	}).Debug("Fact governance status updated")

	return nil
}

// FindActiveByFactKey returns the ID of an existing active fact with the given
// canonical key, or "" if none exists. Used by the version chain (P6.2) to detect
// when a new extraction should supersede an existing fact.
func (r *Repository) FindActiveByFactKey(ctx context.Context, factKey string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM derived_facts
		WHERE fact_key = $1 AND is_active = TRUE AND governance_status != 'SUPERSEDED'
		ORDER BY created_at DESC
		LIMIT 1
	`, factKey).Scan(&id)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return "", nil // No existing fact — this is the first extraction
		}
		return "", fmt.Errorf("failed to find active fact by key: %w", err)
	}
	return id, nil
}

// SupersedeFact marks a fact as superseded by a newer version
func (r *Repository) SupersedeFact(ctx context.Context, oldFactID string, newFactID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Mark old fact as superseded
	_, err = tx.ExecContext(ctx, `
		UPDATE derived_facts
		SET governance_status = 'SUPERSEDED',
		    is_active = FALSE,
		    superseded_by = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, oldFactID, newFactID)
	if err != nil {
		return fmt.Errorf("failed to supersede old fact: %w", err)
	}

	// Link new fact to old
	_, err = tx.ExecContext(ctx, `
		UPDATE derived_facts
		SET supersedes = $2,
		    updated_at = NOW()
		WHERE id = $1
	`, newFactID, oldFactID)
	if err != nil {
		return fmt.Errorf("failed to link new fact: %w", err)
	}

	return tx.Commit()
}

// =============================================================================
// EXTRACTION AUDIT LOG
// =============================================================================

// ExtractionAuditEntry represents an audit log entry for extraction
type ExtractionAuditEntry struct {
	ID                    string          `json:"id"`
	SourceDocumentID      string          `json:"sourceDocumentId,omitempty"`
	SourceSectionID       string          `json:"sourceSectionId,omitempty"`
	DerivedFactID         string          `json:"derivedFactId,omitempty"`
	ExtractionMethod      string          `json:"extractionMethod"`
	ExtractionStartedAt   time.Time       `json:"extractionStartedAt"`
	ExtractionCompletedAt *time.Time      `json:"extractionCompletedAt,omitempty"`
	ExtractionDurationMs  int             `json:"extractionDurationMs"`
	LLMProvider           string          `json:"llmProvider,omitempty"`
	LLMModel              string          `json:"llmModel,omitempty"`
	LLMPromptTokens       int             `json:"llmPromptTokens"`
	LLMCompletionTokens   int             `json:"llmCompletionTokens"`
	LLMRawResponse        string          `json:"llmRawResponse,omitempty"`
	ConsensusRequired     bool            `json:"consensusRequired"`
	ConsensusAchieved     bool            `json:"consensusAchieved"`
	ProvidersAgreed       []string        `json:"providersAgreed,omitempty"`
	ProvidersDisagreed    []string        `json:"providersDisagreed,omitempty"`
	DisagreementDetails   json.RawMessage `json:"disagreementDetails,omitempty"`
	Success               bool            `json:"success"`
	ErrorMessage          string          `json:"errorMessage,omitempty"`
	ConfidenceScore       float64         `json:"confidenceScore"`
	CreatedAt             time.Time       `json:"createdAt"`
}

// LogExtraction creates an audit log entry for an extraction operation
func (r *Repository) LogExtraction(ctx context.Context, entry *ExtractionAuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	agreed := entry.ProvidersAgreed
	if agreed == nil {
		agreed = []string{}
	}
	disagreed := entry.ProvidersDisagreed
	if disagreed == nil {
		disagreed = []string{}
	}

	query := `
		INSERT INTO extraction_audit_log (
			id, source_document_id, source_section_id, derived_fact_id,
			extraction_method, extraction_started_at, extraction_completed_at, extraction_duration_ms,
			llm_provider, llm_model, llm_prompt_tokens, llm_completion_tokens, llm_raw_response,
			consensus_required, consensus_achieved, providers_agreed, providers_disagreed, disagreement_details,
			success, error_message, confidence_score
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, nilIfEmpty(entry.SourceDocumentID), nilIfEmpty(entry.SourceSectionID), nilIfEmpty(entry.DerivedFactID),
		entry.ExtractionMethod, entry.ExtractionStartedAt, entry.ExtractionCompletedAt, entry.ExtractionDurationMs,
		nilIfEmpty(entry.LLMProvider), nilIfEmpty(entry.LLMModel), entry.LLMPromptTokens, entry.LLMCompletionTokens, nilIfEmpty(entry.LLMRawResponse),
		entry.ConsensusRequired, entry.ConsensusAchieved, pq.Array(agreed), pq.Array(disagreed), nilIfEmptyJSON(entry.DisagreementDetails),
		entry.Success, nilIfEmpty(entry.ErrorMessage), entry.ConfidenceScore,
	)

	if err != nil {
		return fmt.Errorf("failed to log extraction: %w", err)
	}

	return nil
}

// =============================================================================
// HUMAN ESCALATION QUEUE
// =============================================================================

// EscalationEntry represents an item in the human escalation queue
type EscalationEntry struct {
	ID                string          `json:"id"`
	DerivedFactID     string          `json:"derivedFactId,omitempty"`
	SourceDocumentID  string          `json:"sourceDocumentId,omitempty"`
	EscalationReason  string          `json:"escalationReason"` // CONSENSUS_NOT_ACHIEVED, LOW_CONFIDENCE, CRITICAL_SAFETY, MANUAL_REQUEST
	EscalationDetails json.RawMessage `json:"escalationDetails,omitempty"`
	Priority          string          `json:"priority"` // LOW, NORMAL, HIGH, CRITICAL
	AssignedTo        string          `json:"assignedTo,omitempty"`
	AssignedAt        *time.Time      `json:"assignedAt,omitempty"`
	Status            string          `json:"status"` // PENDING, IN_REVIEW, RESOLVED, DEFERRED
	Resolution        string          `json:"resolution,omitempty"` // APPROVED, REJECTED, MODIFIED, DEFERRED
	ResolutionNotes   string          `json:"resolutionNotes,omitempty"`
	ResolvedBy        string          `json:"resolvedBy,omitempty"`
	ResolvedAt        *time.Time      `json:"resolvedAt,omitempty"`
	SLADeadline       *time.Time      `json:"slaDeadline,omitempty"`
	SLABreached       bool            `json:"slaBreached"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

// CreateEscalation creates a new escalation entry
func (r *Repository) CreateEscalation(ctx context.Context, entry *EscalationEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Status == "" {
		entry.Status = "PENDING"
	}
	if entry.Priority == "" {
		entry.Priority = "NORMAL"
	}

	// Calculate SLA deadline based on priority
	var slaDeadline time.Time
	switch entry.Priority {
	case "CRITICAL":
		slaDeadline = time.Now().Add(4 * time.Hour)
	case "HIGH":
		slaDeadline = time.Now().Add(24 * time.Hour)
	case "NORMAL":
		slaDeadline = time.Now().Add(72 * time.Hour)
	default:
		slaDeadline = time.Now().Add(168 * time.Hour) // 1 week
	}
	entry.SLADeadline = &slaDeadline

	query := `
		INSERT INTO human_escalation_queue (
			id, derived_fact_id, source_document_id, escalation_reason, escalation_details,
			priority, status, sla_deadline
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, nilIfEmpty(entry.DerivedFactID), nilIfEmpty(entry.SourceDocumentID),
		entry.EscalationReason, entry.EscalationDetails, entry.Priority, entry.Status, entry.SLADeadline,
	)

	if err != nil {
		return fmt.Errorf("failed to create escalation: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"escalationId": entry.ID,
		"reason":       entry.EscalationReason,
		"priority":     entry.Priority,
	}).Info("Escalation created")

	return nil
}

// GetPendingEscalations retrieves pending escalations
func (r *Repository) GetPendingEscalations(ctx context.Context, limit int) ([]*EscalationEntry, error) {
	query := `
		SELECT id, derived_fact_id, source_document_id, escalation_reason, escalation_details,
		       priority, assigned_to, assigned_at, status, sla_deadline, sla_breached,
		       created_at, updated_at
		FROM human_escalation_queue
		WHERE status = 'PENDING'
		ORDER BY
			CASE priority
				WHEN 'CRITICAL' THEN 1
				WHEN 'HIGH' THEN 2
				WHEN 'NORMAL' THEN 3
				ELSE 4
			END,
			created_at
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query escalations: %w", err)
	}
	defer rows.Close()

	var entries []*EscalationEntry
	for rows.Next() {
		entry := &EscalationEntry{}
		var factID, docID, assignedTo sql.NullString
		var assignedAt, slaDeadline sql.NullTime

		err := rows.Scan(
			&entry.ID, &factID, &docID, &entry.EscalationReason, &entry.EscalationDetails,
			&entry.Priority, &assignedTo, &assignedAt, &entry.Status, &slaDeadline, &entry.SLABreached,
			&entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan escalation: %w", err)
		}

		if factID.Valid {
			entry.DerivedFactID = factID.String
		}
		if docID.Valid {
			entry.SourceDocumentID = docID.String
		}
		if assignedTo.Valid {
			entry.AssignedTo = assignedTo.String
		}
		if assignedAt.Valid {
			entry.AssignedAt = &assignedAt.Time
		}
		if slaDeadline.Valid {
			entry.SLADeadline = &slaDeadline.Time
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// ResolveEscalation resolves an escalation entry
func (r *Repository) ResolveEscalation(ctx context.Context, id string, resolution string, resolvedBy string, notes string) error {
	query := `
		UPDATE human_escalation_queue
		SET status = 'RESOLVED',
		    resolution = $2,
		    resolved_by = $3,
		    resolution_notes = $4,
		    resolved_at = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, id, resolution, resolvedBy, notes)
	if err != nil {
		return fmt.Errorf("failed to resolve escalation: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"escalationId": id,
		"resolution":   resolution,
		"resolvedBy":   resolvedBy,
	}).Info("Escalation resolved")

	return nil
}

// =============================================================================
// STATISTICS
// =============================================================================

// FactStoreStats contains statistics about the FactStore
type FactStoreStats struct {
	TotalDocuments      int            `json:"totalDocuments"`
	TotalSections       int            `json:"totalSections"`
	TotalFacts          int            `json:"totalFacts"`
	FactsByKB           map[string]int `json:"factsByKb"`
	FactsByStatus       map[string]int `json:"factsByStatus"`
	FactsByMethod       map[string]int `json:"factsByMethod"`
	PendingEscalations  int            `json:"pendingEscalations"`
	AvgConfidence       float64        `json:"avgConfidence"`
}

// GetStats returns statistics about the FactStore
func (r *Repository) GetStats(ctx context.Context) (*FactStoreStats, error) {
	stats := &FactStoreStats{
		FactsByKB:     make(map[string]int),
		FactsByStatus: make(map[string]int),
		FactsByMethod: make(map[string]int),
	}

	// Total counts
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM source_documents").Scan(&stats.TotalDocuments)
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM source_sections").Scan(&stats.TotalSections)
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM derived_facts").Scan(&stats.TotalFacts)
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM human_escalation_queue WHERE status = 'PENDING'").Scan(&stats.PendingEscalations)
	r.db.QueryRowContext(ctx, "SELECT COALESCE(AVG(extraction_confidence), 0) FROM derived_facts").Scan(&stats.AvgConfidence)

	// Facts by KB
	rows, _ := r.db.QueryContext(ctx, "SELECT target_kb, COUNT(*) FROM derived_facts GROUP BY target_kb")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var kb string
			var count int
			rows.Scan(&kb, &count)
			stats.FactsByKB[kb] = count
		}
	}

	// Facts by status
	rows, _ = r.db.QueryContext(ctx, "SELECT governance_status, COUNT(*) FROM derived_facts GROUP BY governance_status")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var status string
			var count int
			rows.Scan(&status, &count)
			stats.FactsByStatus[status] = count
		}
	}

	// Facts by extraction method
	rows, _ = r.db.QueryContext(ctx, "SELECT extraction_method, COUNT(*) FROM derived_facts GROUP BY extraction_method")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var method string
			var count int
			rows.Scan(&method, &count)
			stats.FactsByMethod[method] = count
		}
	}

	return stats, nil
}

// =============================================================================
// HELPERS
// =============================================================================

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nilIfEmptyJSON returns nil (SQL NULL) for nil/empty json.RawMessage,
// otherwise returns the raw JSON bytes. PostgreSQL jsonb columns reject
// empty byte slices as invalid JSON.
func nilIfEmptyJSON(b json.RawMessage) interface{} {
	if len(b) == 0 {
		return nil
	}
	return []byte(b)
}

// =============================================================================
// PIPELINE RUNNER SUPPORT METHODS
// =============================================================================

// Health checks database connectivity
func (r *Repository) Health(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

// CountDrugMaster returns the count of drugs in drug_master table
func (r *Repository) CountDrugMaster(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM drug_master").Scan(&count)
	if err != nil {
		// Table may not exist, return 0
		return 0, nil
	}
	return count, nil
}

// LookupRxCUIByName looks up RxCUI from drug_master by drug name (case-insensitive)
// Returns empty string if not found
func (r *Repository) LookupRxCUIByName(ctx context.Context, drugName string) (string, error) {
	// Try exact match first, then ILIKE for case-insensitive partial match
	query := `
		SELECT rxcui FROM drug_master
		WHERE LOWER(drug_name) = LOWER($1)
		   OR LOWER(generic_name) = LOWER($1)
		LIMIT 1
	`
	var rxcui string
	err := r.db.QueryRowContext(ctx, query, drugName).Scan(&rxcui)
	if err != nil {
		if err == sql.ErrNoRows {
			// Try partial match with ILIKE
			query = `
				SELECT rxcui FROM drug_master
				WHERE drug_name ILIKE '%' || $1 || '%'
				   OR generic_name ILIKE '%' || $1 || '%'
				ORDER BY LENGTH(drug_name) ASC
				LIMIT 1
			`
			err = r.db.QueryRowContext(ctx, query, drugName).Scan(&rxcui)
			if err != nil {
				if err == sql.ErrNoRows {
					return "", nil // Not found
				}
				return "", err
			}
		} else {
			return "", err
		}
	}

	r.log.WithFields(logrus.Fields{
		"drugName": drugName,
		"rxcui":    rxcui,
	}).Debug("RxCUI lookup result")

	return rxcui, nil
}

// VerifyTableExists checks if a table exists in the database
func (r *Repository) VerifyTableExists(ctx context.Context, tableName string) error {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	var exists bool
	if err := r.db.QueryRowContext(ctx, query, tableName).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("table %s does not exist", tableName)
	}
	return nil
}

// GetSourceDocumentsByStatus returns all source documents with a given status
func (r *Repository) GetSourceDocumentsByStatus(ctx context.Context, status string) ([]*SourceDocument, error) {
	query := `
		SELECT id, source_type, document_id, version_number, raw_content_hash,
			   fetched_at, drug_name, generic_name, rxcui, processing_status,
			   created_at, updated_at
		FROM source_documents
		WHERE processing_status = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*SourceDocument
	for rows.Next() {
		doc := &SourceDocument{}
		err := rows.Scan(
			&doc.ID, &doc.SourceType, &doc.DocumentID, &doc.VersionNumber,
			&doc.RawContentHash, &doc.FetchedAt, &doc.DrugName, &doc.GenericName,
			&doc.RxCUI, &doc.ProcessingStatus, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// CountFactsByType returns count of facts grouped by fact_type
func (r *Repository) CountFactsByType(ctx context.Context) (map[string]int, error) {
	result := make(map[string]int)

	query := `SELECT fact_type, COUNT(*) FROM derived_facts GROUP BY fact_type`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return result, nil // Return empty map on error
	}
	defer rows.Close()

	for rows.Next() {
		var factType string
		var count int
		if err := rows.Scan(&factType, &count); err != nil {
			continue
		}
		result[factType] = count
	}

	return result, nil
}

// GovernanceStats holds governance statistics
type GovernanceStats struct {
	AutoApproved  int `json:"autoApproved"`
	PendingReview int `json:"pendingReview"`
	Rejected      int `json:"rejected"`
}

// GetGovernanceStats returns governance decision statistics
func (r *Repository) GetGovernanceStats(ctx context.Context) (*GovernanceStats, error) {
	stats := &GovernanceStats{}

	// Count by governance status
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN governance_status = 'APPROVED' THEN 1 ELSE 0 END), 0) as approved,
			COALESCE(SUM(CASE WHEN governance_status = 'PENDING_REVIEW' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN governance_status = 'REJECTED' THEN 1 ELSE 0 END), 0) as rejected
		FROM derived_facts
	`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.AutoApproved,
		&stats.PendingReview,
		&stats.Rejected,
	)
	if err != nil {
		return stats, nil // Return zero stats on error
	}

	return stats, nil
}

// CountFactsByTargetKB returns count of facts for a specific target KB
func (r *Repository) CountFactsByTargetKB(ctx context.Context, targetKB string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM derived_facts WHERE target_kb = $1 AND is_active = true`
	err := r.db.QueryRowContext(ctx, query, targetKB).Scan(&count)
	if err != nil {
		return 0, nil
	}
	return count, nil
}

// =============================================================================
// KB PROJECTION (derived_facts → clinical_facts)
// =============================================================================

// ClinicalFactProjection represents a fact ready for projection to clinical_facts

// ClinicalFactProjection represents a fact ready for projection to clinical_facts
type ClinicalFactProjection struct {
	FactID               string
	FactType             string // Must match fact_type enum
	RxCUI                string
	DrugName             string
	Content              json.RawMessage
	SourceType           string // Must match source_type enum
	SourceID             string
	ExtractionMethod     string
	ConfidenceScore      float64
	ConfidenceBand       string // HIGH, MEDIUM, LOW
	Status               string // DRAFT, ACTIVE
	DerivedFactID        string // FK back to derived_facts for audit
}

// ProjectApprovedFactsToClinical projects approved derived_facts to clinical_facts
// This is the key Phase I operation that bridges ingestion → consumption
func (r *Repository) ProjectApprovedFactsToClinical(ctx context.Context) (int, error) {
	// Get all APPROVED facts that haven't been projected yet.
	// HARD DEDUP: When multiple derived_facts share the same fact_key (canonical key),
	// only project the highest-confidence instance to clinical_facts.
	// This collapses duplicates from different sections/tables while preserving
	// full provenance in derived_facts for audit.
	query := `
		SELECT DISTINCT ON (df.fact_key)
			df.id,
			df.target_kb,
			df.fact_type,
			df.fact_key,
			df.fact_data,
			df.extraction_method,
			df.extraction_confidence,
			sd.drug_name,
			sd.rxcui
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		WHERE df.governance_status = 'APPROVED'
		AND df.is_active = true
		AND NOT EXISTS (
			SELECT 1 FROM clinical_facts cf
			WHERE cf.source_id = df.id::text
		)
		AND NOT EXISTS (
			SELECT 1 FROM clinical_facts cf2
			WHERE cf2.fact_type = df.fact_type::fact_type
			AND cf2.rxcui = sd.rxcui
			AND cf2.content = df.fact_data
		)
		ORDER BY df.fact_key, df.extraction_confidence DESC, df.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to query approved facts: %w", err)
	}
	defer rows.Close()

	projected := 0
	for rows.Next() {
		var (
			derivedID           string
			targetKB            string
			factType            string
			factKey             string
			factData            json.RawMessage
			extractionMethod    string
			extractionConfidence float64
			drugName            sql.NullString
			rxcui               sql.NullString
		)

		err := rows.Scan(
			&derivedID, &targetKB, &factType, &factKey, &factData,
			&extractionMethod, &extractionConfidence, &drugName, &rxcui,
		)
		if err != nil {
			r.log.WithError(err).Warn("Failed to scan derived fact for projection")
			continue
		}

		// Skip if no RxCUI (required FK constraint)
		if !rxcui.Valid || rxcui.String == "" {
			r.log.WithField("derivedId", derivedID).Debug("Skipping projection: no RxCUI")
			continue
		}

		// Map derived_facts.fact_type → clinical_facts.fact_type (enum)
		clinicalFactType := mapFactTypeToClinical(factType)
		if clinicalFactType == "" {
			r.log.WithFields(logrus.Fields{
				"derivedId": derivedID,
				"factType":  factType,
			}).Debug("Skipping projection: unmapped fact type")
			continue
		}

		// Map extraction method → source_type enum
		sourceType := mapExtractionToSourceType(extractionMethod)

		// Determine confidence band
		confidenceBand := "LOW"
		if extractionConfidence >= 0.85 {
			confidenceBand = "HIGH"
		} else if extractionConfidence >= 0.65 {
			confidenceBand = "MEDIUM"
		}

		// Insert into clinical_facts
		// The NOT EXISTS check in the SELECT already prevents duplicates
		// We use source_id = derived_fact.id to track provenance
		insertQuery := `
			INSERT INTO clinical_facts (
				fact_type, rxcui, drug_name, content,
				source_type, source_id, extraction_method,
				confidence_score, confidence_band, status
			) VALUES (
				$1::fact_type, $2, $3, $4,
				$5::source_type, $6, $7,
				$8, $9::confidence_band, 'ACTIVE'::fact_status
			)
			RETURNING fact_id
		`

		var clinicalFactID string
		err = r.db.QueryRowContext(ctx, insertQuery,
			clinicalFactType, rxcui.String, drugName.String, factData,
			sourceType, derivedID, extractionMethod,
			extractionConfidence, confidenceBand,
		).Scan(&clinicalFactID)

		if err != nil {
			r.log.WithError(err).WithFields(logrus.Fields{
				"derivedId": derivedID,
				"factType":  clinicalFactType,
				"rxcui":     rxcui.String,
			}).Warn("Failed to project fact to clinical_facts")
			continue
		}

		r.log.WithFields(logrus.Fields{
			"derivedId":      derivedID,
			"clinicalFactId": clinicalFactID,
			"factType":       clinicalFactType,
			"rxcui":          rxcui.String,
		}).Debug("Fact projected to clinical_facts")

		projected++
	}

	return projected, nil
}

// mapFactTypeToClinical maps derived_facts.fact_type to clinical_facts.fact_type enum
func mapFactTypeToClinical(derivedType string) string {
	mapping := map[string]string{
		"DOSING":              "ORGAN_IMPAIRMENT",
		"ORGAN_IMPAIRMENT":    "ORGAN_IMPAIRMENT",
		"RENAL_DOSING":        "ORGAN_IMPAIRMENT",
		"HEPATIC_DOSING":      "ORGAN_IMPAIRMENT",
		"ADVERSE_REACTION":    "SAFETY_SIGNAL",
		"SAFETY_SIGNAL":       "SAFETY_SIGNAL",
		"BOXED_WARNING":       "SAFETY_SIGNAL",
		"DRUG_INTERACTION":    "INTERACTION",
		"INTERACTION":         "INTERACTION",
		"DDI":                 "INTERACTION",
		"REPRODUCTIVE_SAFETY": "REPRODUCTIVE_SAFETY",
		"PREGNANCY":           "REPRODUCTIVE_SAFETY",
		"LACTATION":           "REPRODUCTIVE_SAFETY",
		"FORMULARY":           "FORMULARY",
		"COVERAGE":            "FORMULARY",
		"LAB_REFERENCE":       "LAB_REFERENCE",
		"LAB_INTERPRETATION":  "LAB_REFERENCE",
	}
	return mapping[derivedType]
}

// mapExtractionToSourceType maps extraction_method to source_type enum
func mapExtractionToSourceType(method string) string {
	switch method {
	case "TABLE_PARSE", "NARRATIVE_PARSE", "AUTHORITY_LOOKUP":
		return "ETL"
	case "LLM_CONSENSUS", "LLM_EXTRACTION":
		return "LLM"
	case "API_SYNC":
		return "API_SYNC"
	default:
		return "ETL"
	}
}

// GetProjectionStats returns statistics about projected facts
func (r *Repository) GetProjectionStats(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT fact_type::text, COUNT(*)
		FROM clinical_facts
		WHERE status = 'ACTIVE'
		GROUP BY fact_type
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var factType string
		var count int
		if err := rows.Scan(&factType, &count); err != nil {
			continue
		}
		stats[factType] = count
	}

	return stats, nil
}

// =============================================================================
// COMPLETENESS REPORTS
// =============================================================================

// GradeToGateVerdict converts a letter grade to a gate verdict.
// A/B = PASS (good quality), C = WARNING (review recommended), D/F = BLOCK (quality too low).
func GradeToGateVerdict(grade string) string {
	switch grade {
	case "A", "B":
		return "PASS"
	case "C":
		return "WARNING"
	default:
		return "BLOCK"
	}
}

// SaveCompletenessReport persists a per-drug quality report from the completeness checker.
func (r *Repository) SaveCompletenessReport(ctx context.Context, report *CompletenessReport) error {
	factCountsJSON, err := json.Marshal(report.FactCounts)
	if err != nil {
		factCountsJSON = []byte("{}")
	}
	skipReasonJSON, err := json.Marshal(report.SkipReasonBreakdown)
	if err != nil {
		skipReasonJSON = []byte("{}")
	}

	gateVerdict := GradeToGateVerdict(report.Grade)

	// Ensure arrays are not nil for PostgreSQL
	sectionsCovered := report.SectionsCovered
	if sectionsCovered == nil {
		sectionsCovered = []string{}
	}
	sectionsMissing := report.SectionsMissing
	if sectionsMissing == nil {
		sectionsMissing = []string{}
	}
	warnings := report.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	query := `
		INSERT INTO completeness_reports (
			drug_name, rxcui,
			sections_covered, sections_missing, section_coverage_pct,
			fact_counts, total_facts, fact_types_covered,
			meddra_match_rate, frequency_cov_rate, interaction_qual,
			total_source_rows, extracted_rows, skipped_rows, row_coverage_pct, skip_reason_breakdown,
			structured_count, llm_count, grammar_count, deterministic_pct,
			warnings, grade, gate_verdict
		) VALUES (
			$1, $2,
			$3, $4, $5,
			$6, $7, $8,
			$9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20,
			$21, $22, $23
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		report.DrugName, report.RxCUI,
		pq.Array(sectionsCovered), pq.Array(sectionsMissing), report.SectionCoveragePct,
		factCountsJSON, report.TotalFacts, report.FactTypesCovered,
		report.MedDRAMatchRate, report.FrequencyCovRate, report.InteractionQual,
		report.TotalSourceRows, report.ExtractedRows, report.SkippedRows, report.RowCoveragePct, skipReasonJSON,
		report.StructuredCount, report.LLMCount, report.GrammarCount, report.DeterministicPct,
		pq.Array(warnings), report.Grade, gateVerdict,
	)
	if err != nil {
		return fmt.Errorf("failed to save completeness report: %w", err)
	}

	r.log.WithFields(logrus.Fields{
		"drug":    report.DrugName,
		"grade":   report.Grade,
		"verdict": gateVerdict,
		"facts":   report.TotalFacts,
	}).Info("Completeness report persisted")

	return nil
}
