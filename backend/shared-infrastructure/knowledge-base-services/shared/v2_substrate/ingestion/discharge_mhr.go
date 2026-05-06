package ingestion

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LOINC document-type codes for AU MHR Discharge Summary documents.
// The CDA root code is one of these; the parser uses the code to choose
// "discharge summary" routing without re-inspecting the title.
const (
	loincDischargeSummary       = "18842-5"  // Discharge summary
	loincDischargeMedicationsLn = "10183-2"  // Hospital discharge medications
)

// ParseMHRDischargeCDA parses an AU MHR (My Health Record) Discharge
// Summary CDA R2 document into a ParsedDischargeDocument.
//
// Wave 4.1 handles the synthetic-but-realistic structure exercised by
// testdata/synthetic_cda_discharge.xml. The parser pulls:
//
//   - Document id / discharge date / patient IHI
//   - Discharging facility name (custodian)
//   - The Hospital Discharge Medications section's <substanceAdministration>
//     entries; each yields one ParsedDischargeMedicationLine with
//     AMT code (SNOMED-CT-AU), display name, dose / frequency / route.
//
// Errors are returned only for unrecoverable structural problems
// (malformed XML, missing ClinicalDocument). Per-entry parse problems
// skip the offending entry. ResidentRef must be assigned by the caller
// AFTER identity matching against the patient IHI.
func ParseMHRDischargeCDA(raw []byte) (*ParsedDischargeDocument, error) {
	var doc cdaDischargeClinicalDocument
	if err := xml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("discharge_mhr: parse xml: %w", err)
	}

	out := &ParsedDischargeDocument{
		Source:                  DischargeSourceMHRCDA,
		DocumentID:              doc.ID.Extension,
		DischargeDate:           parseCDATime(doc.EffectiveTime.Value),
		DischargingFacilityName: extractCustodianName(doc.Custodian),
	}

	// PatientIHI lives on the recordTarget; surfaced via StructuredPayload
	// so the caller can run identity matching before assigning ResidentRef.
	ihi := extractIHI(doc.RecordTarget.PatientRole.IDs)
	out.StructuredPayload = map[string]interface{}{
		"patient_ihi":     ihi,
		"document_loinc":  doc.Code.Code,
		"document_title":  strings.TrimSpace(doc.Title),
	}

	for _, comp := range doc.Component.StructuredBody.Components {
		if !isMedicationsSection(comp.Section.Code) {
			continue
		}
		for i, entry := range comp.Section.Entries {
			if entry.SubstanceAdministration == nil {
				continue
			}
			line, ok := mapDischargeMedication(entry.SubstanceAdministration, i+1)
			if !ok {
				continue
			}
			out.MedicationLines = append(out.MedicationLines, line)
		}
	}

	return out, nil
}

// extractCustodianName returns the discharging facility name from the
// CDA custodian/representedCustodianOrganization/name element.
func extractCustodianName(c cdaCustodian) string {
	return strings.TrimSpace(c.RepresentedCustodianOrganization.Name)
}

// isMedicationsSection reports whether code identifies the Hospital
// Discharge Medications section. We accept the LOINC code; the
// displayName is informational only.
func isMedicationsSection(c cdaCode) bool {
	return c.CodeSystem == cdaCodeSystemLOINC && c.Code == loincDischargeMedicationsLn
}

// mapDischargeMedication translates one substanceAdministration element
// into a ParsedDischargeMedicationLine.
//
// Returns ok=false when the entry lacks a recognisable medication code
// (no AMT/SNOMED) — such entries are not actionable downstream and are
// silently skipped.
func mapDischargeMedication(sa *cdaSubstanceAdministration, lineNumber int) (ParsedDischargeMedicationLine, bool) {
	matCode := sa.Consumable.ManufacturedProduct.ManufacturedMaterial.Code
	displayName := strings.TrimSpace(matCode.DisplayName)
	amt := ""
	if matCode.CodeSystem == cdaCodeSystemSNOMED {
		amt = matCode.Code
	} else {
		// Fall through to a translation if present.
		for _, t := range matCode.Translations {
			if t.CodeSystem == cdaCodeSystemSNOMED {
				amt = t.Code
				break
			}
		}
	}
	if amt == "" && displayName == "" {
		return ParsedDischargeMedicationLine{}, false
	}
	line := ParsedDischargeMedicationLine{
		LineNumber:        lineNumber,
		MedicationNameRaw: displayName,
		AMTCode:           amt,
		DoseRaw:           formatDose(sa.DoseQuantity),
		FrequencyRaw:      formatFrequency(sa.EffectiveTimes),
		RouteRaw:          strings.TrimSpace(sa.RouteCode.DisplayName),
	}
	if sa.Text != "" {
		line.Notes = strings.TrimSpace(sa.Text)
	}
	// Indication: when present as a separate <entryRelationship> observation,
	// the displayName of that observation populates IndicationText. Keep
	// it minimal here — V1 may want richer extraction.
	for _, er := range sa.EntryRelationships {
		if er.Observation != nil && er.Observation.Code.DisplayName != "" {
			line.IndicationText = strings.TrimSpace(er.Observation.Code.DisplayName)
			break
		}
	}
	return line, true
}

// formatDose joins the value + unit of a CDA <doseQuantity>, e.g.
// "500 mg". Returns the trimmed text content if the structured fields
// are absent.
func formatDose(d cdaDoseQuantity) string {
	v := strings.TrimSpace(d.Value)
	u := strings.TrimSpace(d.Unit)
	switch {
	case v != "" && u != "":
		return v + " " + u
	case v != "":
		return v
	case u != "":
		return u
	}
	return strings.TrimSpace(d.Text)
}

// formatFrequency picks a usable frequency string out of a
// substanceAdministration's effectiveTimes. The first PIVL_TS@period
// (e.g. 12h) wins; absent that, the first text body wins. Empty when
// nothing parseable is present.
func formatFrequency(ts []cdaEffectiveTime) string {
	for _, t := range ts {
		if v := strings.TrimSpace(t.Period.Value); v != "" {
			u := strings.TrimSpace(t.Period.Unit)
			if u == "" {
				u = "h"
			}
			return "every " + v + u
		}
		if v := strings.TrimSpace(t.Text); v != "" {
			return v
		}
	}
	return ""
}

// ============================================================================
// CDA XML structural types — discharge-medications subset (kept distinct
// from the pathology cdaClinicalDocument so encoding/xml can dispatch on
// XMLName per file).
// ============================================================================

type cdaDischargeClinicalDocument struct {
	XMLName       xml.Name             `xml:"ClinicalDocument"`
	ID            cdaID                `xml:"id"`
	Code          cdaCode              `xml:"code"`
	Title         string               `xml:"title"`
	EffectiveTime cdaTimeValue         `xml:"effectiveTime"`
	RecordTarget  cdaRecordTarget      `xml:"recordTarget"`
	Custodian     cdaCustodian         `xml:"custodian"`
	Component     cdaTopComponentDisch `xml:"component"`
}

type cdaCustodian struct {
	RepresentedCustodianOrganization cdaCustodianOrg `xml:"assignedCustodian>representedCustodianOrganization"`
}

type cdaCustodianOrg struct {
	ID   cdaID  `xml:"id"`
	Name string `xml:"name"`
}

type cdaTopComponentDisch struct {
	StructuredBody cdaStructuredBodyDisch `xml:"structuredBody"`
}

type cdaStructuredBodyDisch struct {
	Components []cdaSectionComponentDisch `xml:"component"`
}

type cdaSectionComponentDisch struct {
	Section cdaSectionDisch `xml:"section"`
}

type cdaSectionDisch struct {
	Code    cdaCode               `xml:"code"`
	Entries []cdaEntryDischMedAdm `xml:"entry"`
}

type cdaEntryDischMedAdm struct {
	SubstanceAdministration *cdaSubstanceAdministration `xml:"substanceAdministration"`
}

type cdaSubstanceAdministration struct {
	Text               string                       `xml:"text"`
	EffectiveTimes     []cdaEffectiveTime           `xml:"effectiveTime"`
	RouteCode          cdaCodeRef                   `xml:"routeCode"`
	DoseQuantity       cdaDoseQuantity              `xml:"doseQuantity"`
	Consumable         cdaConsumable                `xml:"consumable"`
	EntryRelationships []cdaEntryRelationship       `xml:"entryRelationship"`
}

type cdaEffectiveTime struct {
	Text   string             `xml:",chardata"`
	Period cdaEffectivePeriod `xml:"period"`
}

type cdaEffectivePeriod struct {
	Value string `xml:"value,attr"`
	Unit  string `xml:"unit,attr"`
}

type cdaDoseQuantity struct {
	Value string `xml:"value,attr"`
	Unit  string `xml:"unit,attr"`
	Text  string `xml:",chardata"`
}

type cdaConsumable struct {
	ManufacturedProduct cdaManufacturedProduct `xml:"manufacturedProduct"`
}

type cdaManufacturedProduct struct {
	ManufacturedMaterial cdaManufacturedMaterial `xml:"manufacturedMaterial"`
}

type cdaManufacturedMaterial struct {
	Code cdaCode `xml:"code"`
}

type cdaEntryRelationship struct {
	Observation *cdaObservation `xml:"observation"`
}

// Use a special time helper — IngestDischargePDFFromMHR converts a
// ParsedDischargeDocument to the storage layer's contract; storage
// validates DischargeDate / ResidentRef.

// AssignResident attaches a resident_ref to a parsed document after the
// caller has run identity matching. Returns the same pointer for
// chaining convenience.
func AssignResident(doc *ParsedDischargeDocument, residentRef uuid.UUID) *ParsedDischargeDocument {
	if doc != nil {
		doc.ResidentRef = residentRef
	}
	return doc
}

// Compile-time guards — DischargeDate hits time.Time helpers; force the
// import to remain referenced even if future edits remove the only use.
var _ = time.Time{}
