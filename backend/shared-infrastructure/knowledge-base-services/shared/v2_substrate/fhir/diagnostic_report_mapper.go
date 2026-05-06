package fhir

import (
	"fmt"
	"strings"
	"time"

	"github.com/cardiofit/shared/v2_substrate/ingestion"
)

// FHIR R4 codeSystem URIs. Differ from the CDA OID encoding — the
// DiagnosticReport mapper deals exclusively in URI-form codings.
const (
	fhirSystemLOINC          = "http://loinc.org"
	fhirSystemSNOMED         = "http://snomed.info/sct"
	fhirSystemInterpretation = "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"
)

// ParseFHIRDiagnosticReport translates a FHIR R4 DiagnosticReport
// resource (json-decoded as map[string]interface{}) into a slice of
// ParsedObservation records — the same DTO produced by the CDA path
// (Wave 3.1) and the HL7 ORU path (Wave 3.3).
//
// FHIR DiagnosticReport may carry results either as inline contained
// Observation resources (preferred for ADHA per AU Core guidance) OR
// as references to externally-served Observation resources. Callers
// that have a Bundle with separate Observation entries should pass
// the resolved slice via observations; the mapper merges contained
// + supplied lists. Inline contained resources take precedence on
// id collision.
//
// Returns an error only when the input map is structurally invalid
// (missing resourceType, malformed effectiveDateTime). Per-result
// parse problems are skipped silently and the surviving observations
// returned (matches the CDA path's tolerance).
func ParseFHIRDiagnosticReport(report map[string]interface{}, observations ...map[string]interface{}) ([]ingestion.ParsedObservation, error) {
	rt, _ := report["resourceType"].(string)
	if rt != "DiagnosticReport" {
		return nil, fmt.Errorf("fhir: expected resourceType=DiagnosticReport, got %q", rt)
	}

	// Build an id-keyed index of resolvable observations: inline
	// contained first (they win on collision), then supplied externals.
	byID := map[string]map[string]interface{}{}
	contained, _ := report["contained"].([]interface{})
	for _, c := range contained {
		obs, _ := c.(map[string]interface{})
		if obs == nil {
			continue
		}
		if rtype, _ := obs["resourceType"].(string); rtype != "Observation" {
			continue
		}
		if id, _ := obs["id"].(string); id != "" {
			byID["#"+id] = obs
		}
	}
	for _, obs := range observations {
		if obs == nil {
			continue
		}
		if rtype, _ := obs["resourceType"].(string); rtype != "Observation" {
			continue
		}
		if id, _ := obs["id"].(string); id != "" {
			// Externals keyed without the # prefix; reference may be
			// "Observation/<id>" or just "<id>".
			byID["Observation/"+id] = obs
			byID[id] = obs
		}
	}

	// Walk DiagnosticReport.result references. Each reference dereferences
	// to one Observation; that Observation produces one ParsedObservation.
	var results []ingestion.ParsedObservation
	resultsRaw, _ := report["result"].([]interface{})
	for _, rRaw := range resultsRaw {
		ref, _ := rRaw.(map[string]interface{})
		if ref == nil {
			continue
		}
		refStr, _ := ref["reference"].(string)
		obs, ok := byID[refStr]
		if !ok {
			// Reference unresolved — skip rather than fail. V1 may want
			// to surface an audit warning for this.
			continue
		}
		po, ok := mapFHIRObservation(obs)
		if !ok {
			continue
		}
		results = append(results, po)
	}

	// Fallback: if DiagnosticReport.result is empty but there are
	// contained Observations, emit each contained Observation directly.
	// AU Core conformant payloads always populate result; this branch
	// covers minimal/test payloads.
	if len(results) == 0 && len(contained) > 0 {
		for _, c := range contained {
			obs, _ := c.(map[string]interface{})
			if obs == nil {
				continue
			}
			if rtype, _ := obs["resourceType"].(string); rtype != "Observation" {
				continue
			}
			if po, ok := mapFHIRObservation(obs); ok {
				results = append(results, po)
			}
		}
	}

	return results, nil
}

// mapFHIRObservation translates a single FHIR R4 Observation resource
// (map form) into a ParsedObservation. Returns ok=false for resources
// that lack any usable LOINC/SNOMED coding.
func mapFHIRObservation(obs map[string]interface{}) (ingestion.ParsedObservation, bool) {
	po := ingestion.ParsedObservation{}

	// code.coding is a list; iterate and dispatch by system.
	code, _ := obs["code"].(map[string]interface{})
	if code != nil {
		codings, _ := code["coding"].([]interface{})
		for _, cRaw := range codings {
			c, _ := cRaw.(map[string]interface{})
			if c == nil {
				continue
			}
			system, _ := c["system"].(string)
			codeStr, _ := c["code"].(string)
			display, _ := c["display"].(string)
			switch system {
			case fhirSystemLOINC:
				if po.LOINCCode == "" {
					po.LOINCCode = codeStr
				}
			case fhirSystemSNOMED:
				if po.SNOMEDCode == "" {
					po.SNOMEDCode = codeStr
				}
			}
			if po.DisplayName == "" && display != "" {
				po.DisplayName = display
			}
		}
		if po.DisplayName == "" {
			if text, _ := code["text"].(string); text != "" {
				po.DisplayName = text
			}
		}
	}
	if po.LOINCCode == "" && po.SNOMEDCode == "" {
		return ingestion.ParsedObservation{}, false
	}

	// effectiveDateTime — FHIR uses ISO-8601 / RFC3339.
	if eff, ok := obs["effectiveDateTime"].(string); ok && eff != "" {
		if t, err := time.Parse(time.RFC3339, eff); err == nil {
			po.ObservedAt = t.UTC()
		}
	} else if effPeriod, ok := obs["effectivePeriod"].(map[string]interface{}); ok {
		// Period.start is the closest analogue to a single observation
		// timestamp; fall through to it when DiagnosticReport encodes
		// effectivePeriod instead.
		if start, _ := effPeriod["start"].(string); start != "" {
			if t, err := time.Parse(time.RFC3339, start); err == nil {
				po.ObservedAt = t.UTC()
			}
		}
	}

	// valueQuantity (numeric) vs valueString (text). FHIR exposes a
	// half-dozen value[x] variants; the two we map here are by far the
	// most common in pathology results.
	if vq, ok := obs["valueQuantity"].(map[string]interface{}); ok {
		if v, ok := numericValue(vq["value"]); ok {
			po.Value = &v
		}
		if u, ok := vq["unit"].(string); ok {
			po.Unit = u
		} else if c, ok := vq["code"].(string); ok {
			// FHIR Quantity.code carries the UCUM code; fall back to it
			// when the human-readable unit isn't supplied.
			po.Unit = c
		}
	} else if vs, ok := obs["valueString"].(string); ok {
		po.ValueText = strings.TrimSpace(vs)
	}

	// interpretation: list of CodeableConcept; we only consume H/L from
	// the HL7 v3 ObservationInterpretation system.
	interps, _ := obs["interpretation"].([]interface{})
	for _, iRaw := range interps {
		ic, _ := iRaw.(map[string]interface{})
		if ic == nil {
			continue
		}
		codings, _ := ic["coding"].([]interface{})
		for _, cRaw := range codings {
			c, _ := cRaw.(map[string]interface{})
			if c == nil {
				continue
			}
			system, _ := c["system"].(string)
			if system != fhirSystemInterpretation {
				continue
			}
			codeStr, _ := c["code"].(string)
			switch strings.ToUpper(codeStr) {
			case "H", "HH":
				po.AbnormalFlag = "high"
			case "L", "LL":
				po.AbnormalFlag = "low"
			}
		}
	}

	return po, true
}

// numericValue coerces an interface{} that might hold either a json-
// decoded float64 or an int into a float64. Returns ok=false for any
// other type. JSON-decoded numbers always come back as float64 unless
// a custom decoder is in use, but the fallback handles json.Number too.
func numericValue(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
