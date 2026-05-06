package reconciliation

import "strings"

// IntentClass is the reconciliation-specific classification of WHY a
// discharge line exists (Layer 2 §3.2 step 5). Distinct from
// models.Intent.Category (which is the long-lived MedicineUse intent
// taxonomy: therapeutic / preventive / symptomatic / trial /
// deprescribing). The reconciliation IntentClass is a transient
// classification used only at the discharge-reconciliation boundary;
// the writeback layer maps it onto a MedicineUse.Intent value.
type IntentClass string

const (
	// IntentAcuteTemporary — started for an acute illness / event during
	// admission; should be reviewed for cessation after the acute phase.
	IntentAcuteTemporary IntentClass = "acute_illness_temporary"
	// IntentNewChronic — started as a long-term therapy; carries forward
	// after discharge.
	IntentNewChronic IntentClass = "new_chronic"
	// IntentReconciledChange — explicit dose / frequency adjustment with
	// rationale recorded on the discharge document.
	IntentReconciledChange IntentClass = "reconciled_change"
	// IntentUnclear — conservative default when no signal is present.
	// ACOP must clarify during reconciliation.
	IntentUnclear IntentClass = "unclear"
)

// IsValidIntentClass reports whether s is a recognised IntentClass value.
func IsValidIntentClass(s string) bool {
	switch IntentClass(s) {
	case IntentAcuteTemporary, IntentNewChronic, IntentReconciledChange, IntentUnclear:
		return true
	}
	return false
}

// acuteKeywords are markers that the line was started for an acute event
// during admission. Sourced from Layer 2 doc §3.2 step 5. Short tokens
// (ami, dvt, pe) are word-boundary matched via containsWord; long
// keywords use plain substring contains.
var acuteKeywords = []string{
	"infection", "sepsis", "post-op", "post operative",
	"surgery", "myocardial infarction",
	"deep vein thrombosis",
	"pulmonary embolism",
	"exacerbation", "acute",
}

// acuteWordKeywords are short tokens that MUST match on a word boundary
// (otherwise "pe" would match "hypertension", etc.).
var acuteWordKeywords = []string{"ami", "dvt", "pe"}

// chronicMarkers are markers that the line was started as long-term
// therapy.
var chronicMarkers = []string{
	"started for ongoing", "long-term", "long term",
	"maintenance", "lifelong", "chronic",
}

// reconciledMarkers are signals that a dose change carries an explicit
// rationale on the discharge document.
var reconciledMarkers = []string{
	"increased", "reduced", "decreased", "uptitrated", "downtitrated",
	"adjusted", "titrated",
}

// ClassifyIntent inspects the discharge text near a diff entry and
// returns the IntentClass. dischargeText is the per-line free-text
// context (typically IndicationText + Notes concatenated, but the
// caller composes it) — the function lowercases internally.
//
// Heuristic priority:
//
//  1. acute_illness_temporary  (NEW or DOSE_CHANGE + acute keyword)
//  2. new_chronic              (NEW + chronic marker)
//  3. reconciled_change        (DOSE_CHANGE + reconciled marker)
//  4. unclear                  (default)
//
// CEASED + UNCHANGED entries always classify as IntentUnclear — the
// reconciliation engine has no opinion on why a medicine was stopped /
// kept; that's the ACOP's call.
func ClassifyIntent(diff DiffEntry, dischargeText string) IntentClass {
	if diff.Class == DiffCeasedMedication || diff.Class == DiffUnchanged {
		return IntentUnclear
	}
	text := strings.ToLower(dischargeText)
	if text == "" {
		return IntentUnclear
	}

	if diff.Class == DiffNewMedication || diff.Class == DiffDoseChange {
		for _, kw := range acuteKeywords {
			if strings.Contains(text, kw) {
				return IntentAcuteTemporary
			}
		}
		for _, kw := range acuteWordKeywords {
			if containsWord(text, kw) {
				return IntentAcuteTemporary
			}
		}
	}

	if diff.Class == DiffNewMedication {
		for _, kw := range chronicMarkers {
			if strings.Contains(text, kw) {
				return IntentNewChronic
			}
		}
	}

	if diff.Class == DiffDoseChange {
		for _, kw := range reconciledMarkers {
			if strings.Contains(text, kw) {
				return IntentReconciledChange
			}
		}
	}

	return IntentUnclear
}

// containsWord reports whether `word` appears in `haystack` on a
// word boundary (non-letter characters or string ends on either side).
// Used for short ambiguous tokens (ami, dvt, pe) where a substring
// match would yield false positives ("pe" inside "hypertension").
func containsWord(haystack, word string) bool {
	if word == "" {
		return false
	}
	idx := 0
	for {
		i := strings.Index(haystack[idx:], word)
		if i < 0 {
			return false
		}
		start := idx + i
		end := start + len(word)
		leftOK := start == 0 || !isWordChar(haystack[start-1])
		rightOK := end == len(haystack) || !isWordChar(haystack[end])
		if leftOK && rightOK {
			return true
		}
		idx = start + 1
		if idx >= len(haystack) {
			return false
		}
	}
}

// isWordChar reports whether b is an ASCII letter or digit. Underscores
// are intentionally treated as non-word so "pe_form" still word-boundary
// matches "pe".
func isWordChar(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z':
		return true
	case b >= 'A' && b <= 'Z':
		return true
	case b >= '0' && b <= '9':
		return true
	}
	return false
}

// ComposeDischargeText joins IndicationText + Notes from a discharge
// line into the single-string view ClassifyIntent consumes. Convenience
// helper so callers don't reinvent the join.
func ComposeDischargeText(line *DischargeLineSummary) string {
	if line == nil {
		return ""
	}
	parts := []string{}
	if s := strings.TrimSpace(line.IndicationText); s != "" {
		parts = append(parts, s)
	}
	if s := strings.TrimSpace(line.Notes); s != "" {
		parts = append(parts, s)
	}
	return strings.Join(parts, " ")
}
