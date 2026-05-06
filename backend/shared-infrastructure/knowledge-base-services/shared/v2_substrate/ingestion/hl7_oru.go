// Package ingestion — HL7 v2.5 ORU^R01 parser for the per-vendor
// pathology fallback path (Wave 3.3).
//
// HL7 v2 messages are pipe-delimited records, one segment per line:
//   MSH|<field-1>|<field-2>|...
//   PID|<field-1>|<field-2>|...
//   OBR|<field-1>|...
//   OBX|<field-1>|...
//
// Within a field, sub-components are caret-delimited; repeats are
// tilde-delimited. The standard MSH-1 / MSH-2 carry the actual
// delimiters in use; the parser here observes them at parse time
// rather than hard-coding | and ^ — this is the one HL7 quirk that
// vendor adapters genuinely need to be liberal about.
//
// Wave 3.3 (full impl): the parser is real. Per-vendor adapter table
// (hl7_vendor_adapters.go) is a stub registry with one default
// pass-through adapter; per-vendor quirks are V1 work as agreements
// are signed.
package ingestion

import (
	"bufio"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HL7 v2 LOINC namespace identifier as used in OBX-3 / OBR-4
// component-2 (the coding system field). Vendors typically emit "LN"
// for LOINC; some emit "L" or the OID 2.16.840.1.113883.6.1.
var hl7LOINCSystems = map[string]bool{
	"LN":                       true,
	"L":                        true,
	"2.16.840.1.113883.6.1":    true,
	"http://loinc.org":         true,
}

// HL7 v2 SNOMED-CT namespace identifier.
var hl7SNOMEDSystems = map[string]bool{
	"SCT":                      true,
	"SNM":                      true,
	"2.16.840.1.113883.6.96":   true,
	"http://snomed.info/sct":   true,
}

// ParseORUR01 parses an HL7 v2.5 ORU^R01 message into a CDAPathologyResult.
// Reusing the CDAPathologyResult DTO across paths keeps the substrate
// write code unified (the ADR's convergence claim).
//
// vendorName selects the per-vendor adapter from the registry; pass
// "generic" or empty for the default pass-through adapter. Unknown
// vendor names also fall through to generic (so the runtime never
// hard-fails on a misconfigured vendor identifier — it logs and
// processes with the generic adapter instead).
func ParseORUR01(raw []byte, vendorName string) (*CDAPathologyResult, error) {
	adapter := lookupVendorAdapter(vendorName)

	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	// HL7 nominally uses CR (\r) as the segment separator, but real-
	// world feeds (and our hand-crafted fixture) use LF. Normalise by
	// splitting on either.
	scanner.Split(scanLinesAnyEOL)

	var (
		fieldSep     byte = '|'
		componentSep byte = '^'
		result            = &CDAPathologyResult{}
		segments     []hl7Segment
	)

	first := true
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if first {
			// MSH-1 / MSH-2 carry the delimiters. MSH-1 is the field
			// separator (the character between MSH and the next field);
			// MSH-2 is the four-character encoding-characters string
			// (component, repetition, escape, sub-component).
			first = false
			if !strings.HasPrefix(line, "MSH") {
				return nil, errors.New("hl7: first segment must be MSH")
			}
			if len(line) < 8 {
				return nil, errors.New("hl7: MSH segment too short")
			}
			fieldSep = line[3]
			componentSep = line[4] // first encoding char
		}
		seg, err := parseSegment(line, fieldSep, componentSep)
		if err != nil {
			return nil, fmt.Errorf("hl7: parse segment: %w", err)
		}
		segments = append(segments, seg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("hl7: scan: %w", err)
	}

	// Walk segments to populate header fields + observations.
	for _, seg := range segments {
		switch seg.name {
		case "MSH":
			// MSH-1 is the field separator itself (the | between MSH
			// and the next field), so after parts := Split(line, "|")
			// and stripping parts[0]==MSH, fields[0] holds MSH-2 (the
			// encoding-characters string ^~\&). Therefore MSH-N for
			// N>=2 lives at fields[N-2]. MSH-7 = fields[5] (timestamp);
			// MSH-10 = fields[8] (message control id / DocumentID).
			if len(seg.fields) >= 6 {
				result.AuthoredAt = parseHL7Time(seg.fields[5])
			}
			if len(seg.fields) >= 9 {
				result.DocumentID = seg.fields[8]
			}
		case "PID":
			// PID-3 carries patient identifiers; we look for an IHI
			// component (component-5 = "IHI" assigning authority).
			if len(seg.fields) >= 3 {
				result.PatientIHI = extractIHIFromPID3(seg.fields[2], componentSep)
			}
		case "OBX":
			po, ok := parseOBX(seg.fields, componentSep)
			if !ok {
				continue
			}
			po = adapter.Adapt(po)
			result.Observations = append(result.Observations, po)
		}
	}

	return result, nil
}

// hl7Segment is a parsed segment with its name and field slice.
// Fields are 1-indexed in HL7 specs; we store them 0-indexed and
// callers consult seg.fields[N-1] when reading per-spec.
type hl7Segment struct {
	name   string
	fields []string
}

func parseSegment(line string, fieldSep, _ byte) (hl7Segment, error) {
	parts := strings.Split(line, string(fieldSep))
	if len(parts) == 0 {
		return hl7Segment{}, errors.New("empty segment")
	}
	return hl7Segment{name: parts[0], fields: parts[1:]}, nil
}

// scanLinesAnyEOL splits on \r, \n, or \r\n so HL7 messages from
// either CR-only feeds or unix-converted fixtures parse uniformly.
func scanLinesAnyEOL(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\r' || data[i] == '\n' {
			// Consume any immediately following LF in a CRLF pair.
			adv := i + 1
			if i+1 < len(data) && data[i] == '\r' && data[i+1] == '\n' {
				adv = i + 2
			}
			return adv, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// extractIHIFromPID3 walks a PID-3 field's repeats (tilde-delimited)
// and returns the identifier whose component-5 (assigning authority)
// is "IHI". Returns empty string when no IHI repeat is present.
func extractIHIFromPID3(field string, componentSep byte) string {
	repeats := strings.Split(field, "~")
	for _, rep := range repeats {
		comps := strings.Split(rep, string(componentSep))
		// PID-3 (CX) component layout: 1=ID, 2=CheckDigit, 3=CDScheme,
		// 4=Assigning Authority (HD), 5=Identifier Type Code. The IHI
		// label appears in either component 4 or component 5 depending
		// on vendor convention; accept either to be liberal.
		for _, idx := range []int{3, 4} {
			if len(comps) > idx && strings.EqualFold(strings.TrimSpace(comps[idx]), "IHI") {
				return strings.TrimSpace(comps[0])
			}
		}
	}
	return ""
}

// parseOBX maps an OBX segment's fields into a ParsedObservation.
// Field indices (0-based; HL7-1-based subtract 1):
//   OBX-2 (idx 1) : value type — NM (numeric) | ST (string) | ...
//   OBX-3 (idx 2) : observation identifier (CWE) — code^display^system
//   OBX-5 (idx 4) : observation value
//   OBX-6 (idx 5) : units (UCUM in ANZ)
//   OBX-8 (idx 7) : abnormal flags — H | L | HH | LL | N
//   OBX-14 (idx 13): observation date/time (override; falls back to OBR-7)
func parseOBX(fields []string, componentSep byte) (ParsedObservation, bool) {
	if len(fields) < 5 {
		return ParsedObservation{}, false
	}
	po := ParsedObservation{}

	// OBX-3 — code^display^system
	idComps := strings.Split(fields[2], string(componentSep))
	codeStr := safeIdx(idComps, 0)
	display := safeIdx(idComps, 1)
	system := safeIdx(idComps, 2)
	po.DisplayName = display
	switch {
	case hl7LOINCSystems[system]:
		po.LOINCCode = codeStr
	case hl7SNOMEDSystems[system]:
		po.SNOMEDCode = codeStr
	default:
		// Unknown system — best-effort assume LOINC. ADHA + ANZ pathology
		// vendors overwhelmingly use LOINC for OBX-3; SNOMED appears in
		// component-4/5/6 (alternate identifier) which we don't consume.
		po.LOINCCode = codeStr
	}
	if po.LOINCCode == "" && po.SNOMEDCode == "" {
		return ParsedObservation{}, false
	}

	// OBX-2 dispatches value parsing.
	valueType := strings.ToUpper(safeIdx(fields, 1))
	rawVal := safeIdx(fields, 4)
	switch valueType {
	case "NM", "SN":
		if v, err := strconv.ParseFloat(strings.TrimSpace(rawVal), 64); err == nil {
			po.Value = &v
			po.Unit = strings.TrimSpace(safeIdx(fields, 5))
		}
	case "ST", "TX", "FT":
		po.ValueText = strings.TrimSpace(rawVal)
	default:
		// Unknown OBX-2 — fall back to text capture so the data isn't
		// silently dropped. V1 may expand the dispatch table.
		po.ValueText = strings.TrimSpace(rawVal)
	}

	// OBX-8 abnormal flag.
	flag := strings.ToUpper(strings.TrimSpace(safeIdx(fields, 7)))
	switch flag {
	case "H", "HH":
		po.AbnormalFlag = "high"
	case "L", "LL":
		po.AbnormalFlag = "low"
	}

	// OBX-14 observation timestamp; HL7 v2 is YYYYMMDDhhmmss[+TZ].
	if t := parseHL7Time(safeIdx(fields, 13)); !t.IsZero() {
		po.ObservedAt = t
	}

	return po, true
}

// safeIdx is a bounds-safe slice access for whitespace-tolerant
// per-field reads. Returns empty string for out-of-range indices.
func safeIdx(s []string, i int) string {
	if i < 0 || i >= len(s) {
		return ""
	}
	return s[i]
}

// parseHL7Time accepts HL7 v2 timestamp formats. Returns the zero time
// for empty / unparseable input.
func parseHL7Time(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	formats := []string{
		"20060102150405-0700",
		"20060102150405Z0700",
		"20060102150405",
		"200601021504",
		"20060102",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
