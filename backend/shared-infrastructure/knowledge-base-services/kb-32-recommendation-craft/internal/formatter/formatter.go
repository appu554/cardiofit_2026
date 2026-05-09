package formatter

import (
	"errors"
	"strings"
)

// ErrLayer1OverBudget is returned by Validate when L1Signal exceeds
// Layer1MaxWords words. The caller must trim the signal and revalidate
// before surfacing the recommendation.
var ErrLayer1OverBudget = errors.New("formatter: L1Signal exceeds Layer1MaxWords word budget")

// ErrLayer2OverBudget is returned by Validate when L2Reasoning exceeds
// Layer2MaxWords words. The caller must trim the reasoning and revalidate
// before surfacing the recommendation.
var ErrLayer2OverBudget = errors.New("formatter: L2Reasoning exceeds Layer2MaxWords word budget")

// wordCount returns the number of whitespace-delimited tokens in s.
// strings.Fields splits on any Unicode whitespace (U+0009 through U+000D,
// U+0020, U+0085, U+00A0, U+1680, U+2000–U+200A, U+2028, U+2029, U+202F,
// U+205F, U+3000) producing a deterministic count across locales. An empty
// or all-whitespace string returns 0.
func wordCount(s string) int {
	return len(strings.Fields(s))
}

// WordCount is the exported word-count helper for pre-flight checks.
// Callers may use this to verify that their content will satisfy the
// Layer 1 and Layer 2 budgets before constructing a LayerOutput.
//
// Counting is performed with strings.Fields, which splits on Unicode
// whitespace, making it locale-independent and deterministic. Leading,
// trailing, and repeated internal whitespace are all normalised away.
// An empty or all-whitespace string returns 0.
func WordCount(s string) int {
	return wordCount(s)
}

// IsLayerWithinBudget reports whether content satisfies the word budget
// for the given layer number (1–4). Layers 3 and 4 are unbounded and
// always return true. Layer 1 budget is Layer1MaxWords; Layer 2 budget is
// Layer2MaxWords.
//
// An unrecognised layer number (outside 1–4) returns false as a safe default
// to prevent accidental budget bypass.
func IsLayerWithinBudget(layer int, content string) bool {
	switch layer {
	case 1:
		return wordCount(content) <= Layer1MaxWords
	case 2:
		return wordCount(content) <= Layer2MaxWords
	case 3, 4:
		// Layers 3 and 4 are unbounded by design; see layer descriptions in
		// layers.go. Truncation of provenance or deep-audit lineage would break
		// audit defensibility.
		return true
	default:
		// Unknown layer — return false as a safe default.
		return false
	}
}

// Validate checks that the L1Signal and L2Reasoning fields of out are within
// their respective word budgets. Layer 3 (L3Provenance) and Layer 4
// (L4DeepAudit) are unbounded and are not checked.
//
// Layer 1 is checked before Layer 2; when both exceed their budgets only
// ErrLayer1OverBudget is returned so callers fix layers in order.
//
// Returns nil when both layers are within budget.
func Validate(out LayerOutput) error {
	if wordCount(out.L1Signal) > Layer1MaxWords {
		return ErrLayer1OverBudget
	}
	if wordCount(out.L2Reasoning) > Layer2MaxWords {
		return ErrLayer2OverBudget
	}
	return nil
}
