// Package storage — DischargeDocumentStore is the kb-20 implementation of
// the discharge document persistence layer (Wave 4.1 of the Layer 2
// substrate plan; Layer 2 doc §3.2). It owns the discharge_documents +
// discharge_medication_lines tables created by migration 021.
//
// The store is intentionally narrow: parsing happens upstream in
// shared/v2_substrate/ingestion. This layer persists the parsed
// document + lines and exposes them by id / by resident.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
)

// ErrDuplicateDocument is returned when (source, document_id) collides
// with an existing row. Callers handle re-ingestion via the same
// document_id by treating this as 409 Conflict.
var ErrDuplicateDocument = errors.New("storage: duplicate discharge document for (source, document_id)")

// DischargeDocumentStore implements interfaces.DischargeDocumentStore
// against the discharge_documents + discharge_medication_lines tables.
type DischargeDocumentStore struct {
	db *sql.DB
}

// NewDischargeDocumentStore wires a *sql.DB into the store. The caller
// owns the database lifecycle.
func NewDischargeDocumentStore(db *sql.DB) *DischargeDocumentStore {
	return &DischargeDocumentStore{db: db}
}

// CreateDischargeDocument inserts the document + its medication lines
// in a single transaction. Returns ErrDuplicateDocument when the
// (source, document_id) idempotency key collides with an existing row.
func (s *DischargeDocumentStore) CreateDischargeDocument(ctx context.Context, doc interfaces.DischargeDocument) (*interfaces.DischargeDocument, error) {
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}
	if doc.ResidentRef == uuid.Nil {
		return nil, errors.New("CreateDischargeDocument: resident_ref required")
	}
	if doc.Source == "" {
		return nil, errors.New("CreateDischargeDocument: source required")
	}
	if doc.DischargeDate.IsZero() {
		return nil, errors.New("CreateDischargeDocument: discharge_date required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	const insDoc = `
		INSERT INTO discharge_documents
			(id, resident_ref, source, document_id, discharge_date,
			 discharging_facility_name, raw_text, structured_payload, ingested_at)
		VALUES ($1, $2, $3, $4, $5,
		        $6, $7, $8, NOW())
		RETURNING ingested_at`

	var structPayload interface{}
	if len(doc.StructuredPayload) > 0 {
		structPayload = []byte(doc.StructuredPayload)
	}
	var docIDArg interface{}
	if s := strings.TrimSpace(doc.DocumentID); s != "" {
		docIDArg = s
	}

	row := tx.QueryRowContext(ctx, insDoc,
		doc.ID, doc.ResidentRef, doc.Source, docIDArg, doc.DischargeDate,
		nilIfEmpty(doc.DischargingFacilityName), nilIfEmpty(doc.RawText),
		structPayload,
	)
	if err := row.Scan(&doc.IngestedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicateDocument
		}
		return nil, fmt.Errorf("insert discharge_document: %w", err)
	}

	const insLine = `
		INSERT INTO discharge_medication_lines
			(id, discharge_document_ref, line_number, medication_name_raw,
			 amt_code, dose_raw, frequency_raw, route_raw,
			 indication_text, notes)
		VALUES ($1, $2, $3, $4,
		        $5, $6, $7, $8,
		        $9, $10)`

	for i := range doc.MedicationLines {
		line := &doc.MedicationLines[i]
		if line.ID == uuid.Nil {
			line.ID = uuid.New()
		}
		if line.LineNumber == 0 {
			line.LineNumber = i + 1
		}
		line.DischargeDocumentRef = doc.ID
		if _, err := tx.ExecContext(ctx, insLine,
			line.ID, line.DischargeDocumentRef, line.LineNumber, line.MedicationNameRaw,
			nilIfEmpty(line.AMTCode), nilIfEmpty(line.DoseRaw),
			nilIfEmpty(line.FrequencyRaw), nilIfEmpty(line.RouteRaw),
			nilIfEmpty(line.IndicationText), nilIfEmpty(line.Notes),
		); err != nil {
			return nil, fmt.Errorf("insert discharge_medication_line %d: %w", line.LineNumber, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	committed = true

	return s.GetDischargeDocument(ctx, doc.ID)
}

// GetDischargeDocument loads one discharge_documents row + its
// medication lines.
func (s *DischargeDocumentStore) GetDischargeDocument(ctx context.Context, id uuid.UUID) (*interfaces.DischargeDocument, error) {
	const q = `
		SELECT id, resident_ref, source, COALESCE(document_id, ''),
		       discharge_date, COALESCE(discharging_facility_name, ''),
		       COALESCE(raw_text, ''), structured_payload, ingested_at
		FROM discharge_documents
		WHERE id = $1`
	var (
		out         interfaces.DischargeDocument
		structRaw   []byte
	)
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&out.ID, &out.ResidentRef, &out.Source, &out.DocumentID,
		&out.DischargeDate, &out.DischargingFacilityName,
		&out.RawText, &structRaw, &out.IngestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, interfaces.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query discharge_document: %w", err)
	}
	if len(structRaw) > 0 {
		out.StructuredPayload = json.RawMessage(structRaw)
	}
	lines, err := s.ListDischargeMedicationLines(ctx, out.ID)
	if err != nil {
		return nil, err
	}
	out.MedicationLines = lines
	return &out, nil
}

// ListDischargeDocumentsByResident returns documents for a resident
// newest-first, paginated. Each row carries its medication_lines.
func (s *DischargeDocumentStore) ListDischargeDocumentsByResident(ctx context.Context, residentRef uuid.UUID, limit, offset int) ([]interfaces.DischargeDocument, error) {
	if limit <= 0 {
		limit = 50
	}
	const q = `
		SELECT id FROM discharge_documents
		WHERE resident_ref = $1
		ORDER BY discharge_date DESC, ingested_at DESC
		LIMIT $2 OFFSET $3`
	rows, err := s.db.QueryContext(ctx, q, residentRef, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list discharge_documents: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	out := make([]interfaces.DischargeDocument, 0, len(ids))
	for _, id := range ids {
		doc, err := s.GetDischargeDocument(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *doc)
	}
	return out, nil
}

// ListDischargeMedicationLines returns the discharge_medication_lines
// rows for a document, ordered by line_number.
func (s *DischargeDocumentStore) ListDischargeMedicationLines(ctx context.Context, docRef uuid.UUID) ([]interfaces.DischargeMedicationLine, error) {
	const q = `
		SELECT id, discharge_document_ref, line_number, medication_name_raw,
		       COALESCE(amt_code,''), COALESCE(dose_raw,''),
		       COALESCE(frequency_raw,''), COALESCE(route_raw,''),
		       COALESCE(indication_text,''), COALESCE(notes,'')
		FROM discharge_medication_lines
		WHERE discharge_document_ref = $1
		ORDER BY line_number ASC`
	rows, err := s.db.QueryContext(ctx, q, docRef)
	if err != nil {
		return nil, fmt.Errorf("list discharge_medication_lines: %w", err)
	}
	defer rows.Close()
	out := []interfaces.DischargeMedicationLine{}
	for rows.Next() {
		var line interfaces.DischargeMedicationLine
		if err := rows.Scan(
			&line.ID, &line.DischargeDocumentRef, &line.LineNumber, &line.MedicationNameRaw,
			&line.AMTCode, &line.DoseRaw, &line.FrequencyRaw, &line.RouteRaw,
			&line.IndicationText, &line.Notes,
		); err != nil {
			return nil, err
		}
		out = append(out, line)
	}
	return out, nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505). Used by CreateDischargeDocument to
// surface ErrDuplicateDocument.
func isUniqueViolation(err error) bool {
	var pqe *pq.Error
	if errors.As(err, &pqe) {
		return pqe.Code == "23505"
	}
	return false
}
