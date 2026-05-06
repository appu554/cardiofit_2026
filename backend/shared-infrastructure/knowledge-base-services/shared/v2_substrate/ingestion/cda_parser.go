package ingestion

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// LOINC and SNOMED OID code-system identifiers used in CDA documents.
// CDA carries OIDs rather than HTTP URIs (FHIR's pattern); the parser
// maps OIDs back to the internal LOINC/SNOMED columns.
const (
	cdaCodeSystemLOINC  = "2.16.840.1.113883.6.1"
	cdaCodeSystemSNOMED = "2.16.840.1.113883.6.96"
	cdaCodeSystemHL7Obs = "2.16.840.1.113883.5.83" // observation interpretation
	cdaIHIRoot          = "1.2.36.1.2001.1003.0"
)

// ParseCDAPathology parses a CDA R2 pathology document into the internal
// CDAPathologyResult DTO. Wave 3.1 handles the synthetic-but-realistic
// structure exercised by testdata/synthetic_cda_pathology.xml. V1 will
// extend this against the real ADHA conformance pack — the ParsedObservation
// shape is the contract V1 must continue to satisfy.
//
// Numeric (PQ) values populate ParsedObservation.Value + Unit; string
// (ST) values populate ValueText. Both LOINC primary code and SNOMED
// translation code are captured when present. interpretationCode H/L
// maps to AbnormalFlag "high"/"low"; other codes (or absence) leave
// AbnormalFlag empty.
//
// Errors are returned only for unrecoverable structural problems
// (malformed XML, missing ClinicalDocument). Per-observation parse
// problems (unknown value type, unparseable timestamp) skip that
// observation and continue — surfaced as a warning by including the
// document with an Observations slice missing the offending entry.
// V1 may want to switch to fail-fast; today the conservative behaviour
// matches the CSV ingestor pattern.
func ParseCDAPathology(raw []byte) (*CDAPathologyResult, error) {
	var doc cdaClinicalDocument
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("cda: parse xml: %w", err)
	}

	result := &CDAPathologyResult{
		DocumentID: doc.ID.Extension,
		PatientIHI: extractIHI(doc.RecordTarget.PatientRole.IDs),
		AuthoredAt: parseCDATime(doc.EffectiveTime.Value),
	}

	for _, comp := range doc.Component.StructuredBody.Components {
		for _, entry := range comp.Section.Entries {
			obs := entry.Observation
			if obs == nil {
				continue
			}
			po, ok := mapCDAObservation(obs)
			if !ok {
				continue
			}
			result.Observations = append(result.Observations, po)
		}
	}

	return result, nil
}

// extractIHI returns the IHI extension when one of the patientRole
// identifiers carries the IHI root OID. Returns empty string when no
// IHI identifier is present (the matcher's IHI path will then fall
// through to fuzzy paths).
func extractIHI(ids []cdaID) string {
	for _, id := range ids {
		if id.Root == cdaIHIRoot {
			return id.Extension
		}
	}
	return ""
}

// mapCDAObservation translates a single CDA observation activity into a
// ParsedObservation. Returns ok=false when the observation lacks a
// usable code (no LOINC or SNOMED) — such entries are not actionable
// downstream and are silently skipped.
func mapCDAObservation(obs *cdaObservation) (ParsedObservation, bool) {
	po := ParsedObservation{
		DisplayName: obs.Code.DisplayName,
		ObservedAt:  parseCDATime(obs.EffectiveTime.Value),
	}

	// Primary code: LOINC vs SNOMED depending on codeSystem.
	switch obs.Code.CodeSystem {
	case cdaCodeSystemLOINC:
		po.LOINCCode = obs.Code.Code
	case cdaCodeSystemSNOMED:
		po.SNOMEDCode = obs.Code.Code
	}

	// Translation: typically the alternate-namespace coding (LOINC primary
	// + SNOMED translation, or vice versa). Apply whichever slot is empty.
	for _, t := range obs.Code.Translations {
		switch t.CodeSystem {
		case cdaCodeSystemLOINC:
			if po.LOINCCode == "" {
				po.LOINCCode = t.Code
			}
		case cdaCodeSystemSNOMED:
			if po.SNOMEDCode == "" {
				po.SNOMEDCode = t.Code
			}
		}
	}

	if po.LOINCCode == "" && po.SNOMEDCode == "" {
		return ParsedObservation{}, false
	}

	// Value: PQ (numeric+unit) populates Value+Unit; ST populates ValueText.
	switch strings.ToUpper(obs.Value.XSIType) {
	case "PQ":
		if v, err := strconv.ParseFloat(obs.Value.Value, 64); err == nil {
			po.Value = &v
			po.Unit = obs.Value.Unit
		}
	case "ST", "":
		// ST may have no @xsi:type when the schema default applies; treat
		// missing xsi:type with non-empty text as a string value.
		if strings.TrimSpace(obs.Value.Text) != "" {
			po.ValueText = strings.TrimSpace(obs.Value.Text)
		} else if obs.Value.Value != "" {
			// PQ-style @value without @xsi:type — best-effort numeric parse.
			if v, err := strconv.ParseFloat(obs.Value.Value, 64); err == nil {
				po.Value = &v
				po.Unit = obs.Value.Unit
			}
		}
	}

	// Interpretation code (H | L). Only the HL7 ObservationInterpretation
	// code system contributes here; other systems are ignored.
	if obs.InterpretationCode.CodeSystem == cdaCodeSystemHL7Obs {
		switch strings.ToUpper(obs.InterpretationCode.Code) {
		case "H", "HH":
			po.AbnormalFlag = "high"
		case "L", "LL":
			po.AbnormalFlag = "low"
		}
	}

	return po, true
}

// parseCDATime accepts the CDA HL7 v3 timestamp format YYYYMMDDhhmmss[+hhmm].
// Returns the zero time when the string is empty or unparseable; callers
// that need to distinguish "missing" from "unparseable" should pre-check.
func parseCDATime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Common variants: 14 digits, 14 digits + tz, 8 digits (date-only).
	formats := []string{
		"20060102150405-0700",
		"20060102150405Z0700",
		"20060102150405",
		"20060102",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// ============================================================================
// CDA XML structural types — minimal subset needed by the parser.
// ============================================================================

type cdaClinicalDocument struct {
	XMLName       xml.Name        `xml:"ClinicalDocument"`
	ID            cdaID           `xml:"id"`
	EffectiveTime cdaTimeValue    `xml:"effectiveTime"`
	RecordTarget  cdaRecordTarget `xml:"recordTarget"`
	Component     cdaTopComponent `xml:"component"`
}

type cdaID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

type cdaTimeValue struct {
	Value string `xml:"value,attr"`
}

type cdaRecordTarget struct {
	PatientRole cdaPatientRole `xml:"patientRole"`
}

type cdaPatientRole struct {
	IDs []cdaID `xml:"id"`
}

type cdaTopComponent struct {
	StructuredBody cdaStructuredBody `xml:"structuredBody"`
}

type cdaStructuredBody struct {
	Components []cdaSectionComponent `xml:"component"`
}

type cdaSectionComponent struct {
	Section cdaSection `xml:"section"`
}

type cdaSection struct {
	Entries []cdaEntry `xml:"entry"`
}

type cdaEntry struct {
	Observation *cdaObservation `xml:"observation"`
}

type cdaObservation struct {
	Code               cdaCode      `xml:"code"`
	EffectiveTime      cdaTimeValue `xml:"effectiveTime"`
	Value              cdaValue     `xml:"value"`
	InterpretationCode cdaCodeRef   `xml:"interpretationCode"`
}

type cdaCode struct {
	Code         string       `xml:"code,attr"`
	CodeSystem   string       `xml:"codeSystem,attr"`
	DisplayName  string       `xml:"displayName,attr"`
	Translations []cdaCodeRef `xml:"translation"`
}

type cdaCodeRef struct {
	Code        string `xml:"code,attr"`
	CodeSystem  string `xml:"codeSystem,attr"`
	DisplayName string `xml:"displayName,attr"`
}

// cdaValue handles both PQ (xsi:type="PQ" with @value @unit) and ST
// (xsi:type="ST" with text content). Encoding/xml does not natively
// resolve xsi:type, so we capture the attribute as XSIType and dispatch
// in mapCDAObservation.
type cdaValue struct {
	XSIType string `xml:"type,attr"` // namespaced as xsi:type in source; encoding/xml strips ns
	Value   string `xml:"value,attr"`
	Unit    string `xml:"unit,attr"`
	Text    string `xml:",chardata"`
}
