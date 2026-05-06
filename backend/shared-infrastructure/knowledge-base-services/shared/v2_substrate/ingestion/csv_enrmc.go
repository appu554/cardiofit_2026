// Package ingestion implements the v2 substrate's batch source-of-truth
// ingestion path (Layer 2 doc §3.2 — eNRMC Phase 1 CSV strategy).
//
// The package is split into three concerns:
//
//   - csv_enrmc.go   — Telstra MedPoint CSV row parser
//   - normaliser.go  — AMT / SNOMED-CT-AU lookup behind interfaces
//   - runner.go      — orchestrator: parse → match → normalise → write
//
// The orchestrator depends on KB20Client, IdentityMatcher, AMTLookup, and
// SNOMEDLookup as interfaces so the binary can wire real implementations
// while tests use in-memory fakes.
//
// The CSV schema documented here is the assumed Telstra MedPoint v1.x
// export shape; revise when an authoritative sample is available. The
// parser tolerates missing optional fields (indication_text,
// prescriber_ahpra, end_date, ihi or medicare alone) so that real-world
// eNRMC exports — which routinely omit indication and prescriber
// identifiers — still ingest with degraded-but-valid downstream rows.
package ingestion

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// CSVRow is a parsed Telstra MedPoint export row. Field semantics:
//
//   - PatientID, FacilityID — source-of-truth identifiers (informational)
//   - IHI, Medicare         — Australian healthcare identifiers used by
//     IdentityMatcher; either may be empty
//   - FamilyName, GivenName, DOB — demographics for fuzzy-match fallback
//   - MedicationName, Strength, Form, Route, Frequency — what was prescribed
//   - StartDate, EndDate    — therapy window; EndDate may be empty for
//     long-term meds
//   - PrescriberName, PrescriberAHPRA — prescriber attribution; AHPRA may
//     be empty (it commonly is in eNRMC exports)
//   - IndicationText        — free-text indication; commonly empty
//
// Date strings are kept in their raw on-the-wire form here; the runner
// performs format detection and parsing.
type CSVRow struct {
	PatientID       string
	FacilityID      string
	IHI             string
	Medicare        string
	FamilyName      string
	GivenName       string
	DOB             string
	MedicationName  string
	Strength        string
	Form            string
	Route           string
	Frequency       string
	StartDate       string
	EndDate         string
	PrescriberName  string
	PrescriberAHPRA string
	IndicationText  string
	// LineNumber is the 1-indexed source line (header = 1, first data row = 2).
	LineNumber int
}

// ParseError describes a non-fatal per-row issue. Fatal CSV errors
// (malformed quotes, header mismatch) return early from ParseCSV with a
// non-nil error instead.
type ParseError struct {
	LineNumber int
	Field      string
	Reason     string
}

// Error formats a ParseError for display.
func (e ParseError) Error() string {
	return fmt.Sprintf("line %d: field %q: %s", e.LineNumber, e.Field, e.Reason)
}

// expectedHeaders is the canonical column set for the assumed Telstra
// MedPoint v1.x export shape. Header validation is exact-match (after
// lowercase + trim + BOM-strip): unknown columns are rejected so that
// schema drift produces a fatal error rather than silently dropping data.
var expectedHeaders = []string{
	"patient_id",
	"facility_id",
	"ihi",
	"medicare",
	"family_name",
	"given_name",
	"dob",
	"medication_name",
	"strength",
	"form",
	"route",
	"frequency",
	"start_date",
	"end_date",
	"prescriber_name",
	"prescriber_ahpra",
	"indication_text",
}

// ParseCSV parses an io.Reader yielding rows in the documented Telstra
// MedPoint shape. It returns the parsed rows, a list of non-fatal per-row
// ParseErrors (for inclusion in the run report), and a fatal error if the
// CSV is malformed or the header set does not match expectedHeaders.
//
// Tolerated conditions (no error):
//   - UTF-8 BOM at start of file
//   - empty rows (skipped)
//   - missing optional values: ihi, medicare, end_date, prescriber_ahpra,
//     indication_text
//
// Required-but-empty conditions yield a ParseError but do NOT abort the
// whole parse; the row is still returned so the runner can report it
// uniformly with downstream errors.
func ParseCSV(r io.Reader) ([]CSVRow, []ParseError, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // tolerate ragged rows; we validate width manually
	cr.TrimLeadingSpace = true

	headerRec, err := cr.Read()
	if err == io.EOF {
		return nil, nil, errors.New("csv: empty input")
	}
	if err != nil {
		return nil, nil, fmt.Errorf("csv: read header: %w", err)
	}

	// Strip BOM from the first cell if present.
	if len(headerRec) > 0 {
		headerRec[0] = stripBOM(headerRec[0])
	}
	idx, err := indexHeaders(headerRec)
	if err != nil {
		return nil, nil, err
	}

	var (
		rows       []CSVRow
		parseErrs  []ParseError
		lineNumber = 1
	)

	for {
		lineNumber++
		rec, readErr := cr.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, nil, fmt.Errorf("csv: read row at line %d: %w", lineNumber, readErr)
		}
		if isEmptyRecord(rec) {
			continue
		}

		row := CSVRow{LineNumber: lineNumber}
		row.PatientID = pick(rec, idx["patient_id"])
		row.FacilityID = pick(rec, idx["facility_id"])
		row.IHI = pick(rec, idx["ihi"])
		row.Medicare = pick(rec, idx["medicare"])
		row.FamilyName = pick(rec, idx["family_name"])
		row.GivenName = pick(rec, idx["given_name"])
		row.DOB = pick(rec, idx["dob"])
		row.MedicationName = pick(rec, idx["medication_name"])
		row.Strength = pick(rec, idx["strength"])
		row.Form = pick(rec, idx["form"])
		row.Route = pick(rec, idx["route"])
		row.Frequency = pick(rec, idx["frequency"])
		row.StartDate = pick(rec, idx["start_date"])
		row.EndDate = pick(rec, idx["end_date"])
		row.PrescriberName = pick(rec, idx["prescriber_name"])
		row.PrescriberAHPRA = pick(rec, idx["prescriber_ahpra"])
		row.IndicationText = pick(rec, idx["indication_text"])

		// Required-field checks: collect ParseErrors but still emit the row
		// so the runner reports a single per-line outcome.
		if row.MedicationName == "" {
			parseErrs = append(parseErrs, ParseError{
				LineNumber: lineNumber, Field: "medication_name",
				Reason: "required field is empty",
			})
		}
		if row.StartDate == "" {
			parseErrs = append(parseErrs, ParseError{
				LineNumber: lineNumber, Field: "start_date",
				Reason: "required field is empty",
			})
		}
		if row.FamilyName == "" && row.GivenName == "" {
			parseErrs = append(parseErrs, ParseError{
				LineNumber: lineNumber, Field: "family_name|given_name",
				Reason: "at least one name field must be present",
			})
		}

		rows = append(rows, row)
	}
	return rows, parseErrs, nil
}

// stripBOM removes a leading UTF-8 BOM from s, if any.
func stripBOM(s string) string {
	const bom = "\ufeff"
	return strings.TrimPrefix(s, bom)
}

// indexHeaders builds a name → column-index map. Header normalisation:
// lowercase + trim. Returns an error if any expected column is missing or
// any unexpected column is present (schema-drift detection).
func indexHeaders(rec []string) (map[string]int, error) {
	got := make(map[string]int, len(rec))
	for i, h := range rec {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, dup := got[key]; dup {
			return nil, fmt.Errorf("csv: duplicate header %q", key)
		}
		got[key] = i
	}
	for _, want := range expectedHeaders {
		if _, ok := got[want]; !ok {
			return nil, fmt.Errorf("csv: missing required header %q (expected: %v)",
				want, expectedHeaders)
		}
	}
	for k := range got {
		if !contains(expectedHeaders, k) {
			return nil, fmt.Errorf("csv: unexpected header %q (expected only: %v)",
				k, expectedHeaders)
		}
	}
	return got, nil
}

// pick returns the trimmed value at column i, or "" if i is out of range
// (which can happen for ragged rows under FieldsPerRecord=-1).
func pick(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}

// isEmptyRecord reports whether every cell in rec is whitespace.
func isEmptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// contains is a tiny helper because the standard library still does not
// expose a generic slices.Contains for strings in this module's go.mod
// minimum supported toolchain configuration.
func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}
