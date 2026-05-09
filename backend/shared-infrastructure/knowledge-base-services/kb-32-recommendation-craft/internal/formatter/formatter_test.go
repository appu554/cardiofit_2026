package formatter

import (
	"errors"
	"strings"
	"testing"
)

// helper: generate a string with exactly n words.
func nWords(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = "word"
	}
	return strings.Join(words, " ")
}

// ---------------------------------------------------------------------------
// WordCount tests
// ---------------------------------------------------------------------------

func TestWordCount_EmptyString(t *testing.T) {
	if got := WordCount(""); got != 0 {
		t.Fatalf("WordCount(\"\") = %d, want 0", got)
	}
}

func TestWordCount_SingleWord(t *testing.T) {
	if got := WordCount("hello"); got != 1 {
		t.Fatalf("WordCount(\"hello\") = %d, want 1", got)
	}
}

func TestWordCount_MultipleWords(t *testing.T) {
	if got := WordCount("one two three"); got != 3 {
		t.Fatalf("WordCount(\"one two three\") = %d, want 3", got)
	}
}

func TestWordCount_LeadingAndTrailingWhitespace(t *testing.T) {
	if got := WordCount("  hello world  "); got != 2 {
		t.Fatalf("WordCount(\"  hello world  \") = %d, want 2", got)
	}
}

func TestWordCount_MultipleInternalSpaces(t *testing.T) {
	if got := WordCount("one   two     three"); got != 3 {
		t.Fatalf("WordCount(\"one   two     three\") = %d, want 3", got)
	}
}

func TestWordCount_Unicode(t *testing.T) {
	// Non-ASCII content: two words, each containing accented characters.
	// strings.Fields splits on Unicode whitespace, so this must count as 2.
	if got := WordCount("résumé hôpital"); got != 2 {
		t.Fatalf("WordCount(\"résumé hôpital\") = %d, want 2", got)
	}
}

// ---------------------------------------------------------------------------
// IsLayerWithinBudget tests
// ---------------------------------------------------------------------------

func TestIsLayerWithinBudget_AllLayers(t *testing.T) {
	tests := []struct {
		name    string
		layer   int
		content string
		want    bool
	}{
		// Layer 1 — cap 25 words
		{"L1 within budget", 1, nWords(25), true},
		{"L1 one word under", 1, nWords(24), true},
		{"L1 over budget", 1, nWords(26), false},
		{"L1 empty", 1, "", true},

		// Layer 2 — cap 100 words
		{"L2 within budget", 2, nWords(100), true},
		{"L2 one word under", 2, nWords(99), true},
		{"L2 over budget", 2, nWords(101), false},
		{"L2 empty", 2, "", true},

		// Layer 3 — unbounded (always true)
		{"L3 always true small", 3, nWords(1000), true},
		{"L3 always true empty", 3, "", true},

		// Layer 4 — unbounded (always true)
		{"L4 always true large", 4, nWords(5000), true},
		{"L4 always true empty", 4, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsLayerWithinBudget(tc.layer, tc.content)
			if got != tc.want {
				t.Fatalf("IsLayerWithinBudget(%d, <content>) = %v, want %v", tc.layer, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Validate tests
// ---------------------------------------------------------------------------

func TestValidate_Layer1OverBudget(t *testing.T) {
	out := LayerOutput{
		L1Signal:    nWords(26), // one over the 25-word cap
		L2Reasoning: nWords(10),
	}
	err := Validate(out)
	if !errors.Is(err, ErrLayer1OverBudget) {
		t.Fatalf("Validate() = %v, want ErrLayer1OverBudget", err)
	}
}

func TestValidate_Layer1AtBoundary(t *testing.T) {
	out := LayerOutput{
		L1Signal:    nWords(25), // exactly at the cap — must pass
		L2Reasoning: nWords(10),
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil (25 words is within budget)", err)
	}
}

func TestValidate_Layer1Empty(t *testing.T) {
	out := LayerOutput{
		L1Signal:    "", // 0 words — within 25
		L2Reasoning: "rationale",
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil (empty string is within budget)", err)
	}
}

func TestValidate_Layer2OverBudget(t *testing.T) {
	out := LayerOutput{
		L1Signal:    nWords(10),
		L2Reasoning: nWords(101), // one over the 100-word cap
	}
	err := Validate(out)
	if !errors.Is(err, ErrLayer2OverBudget) {
		t.Fatalf("Validate() = %v, want ErrLayer2OverBudget", err)
	}
}

func TestValidate_Layer2AtBoundary(t *testing.T) {
	out := LayerOutput{
		L1Signal:    nWords(10),
		L2Reasoning: nWords(100), // exactly at the cap — must pass
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil (100 words is within budget)", err)
	}
}

func TestValidate_BothLayer1And2OverBudget(t *testing.T) {
	// Layer 1 error must be returned first (checked before Layer 2).
	out := LayerOutput{
		L1Signal:    nWords(26),
		L2Reasoning: nWords(101),
	}
	err := Validate(out)
	if !errors.Is(err, ErrLayer1OverBudget) {
		t.Fatalf("Validate() = %v, want ErrLayer1OverBudget (L1 checked first)", err)
	}
}

func TestValidate_Layer3UnboundedPasses(t *testing.T) {
	// 1000 provenance entries — no budget cap on Layer 3.
	provenance := make([]string, 1000)
	for i := range provenance {
		provenance[i] = "anchor-ref"
	}
	out := LayerOutput{
		L1Signal:     nWords(10),
		L2Reasoning:  nWords(50),
		L3Provenance: provenance,
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil (Layer 3 is unbounded)", err)
	}
}

func TestValidate_Layer4UnboundedPasses(t *testing.T) {
	// Layer 4 is unbounded — an extremely long string must not fail.
	out := LayerOutput{
		L1Signal:    nWords(10),
		L2Reasoning: nWords(50),
		L4DeepAudit: strings.Repeat("evidence-trace-token ", 10_000),
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil (Layer 4 is unbounded)", err)
	}
}

func TestValidate_HappyPath(t *testing.T) {
	// All four layers within budget — complete happy path.
	out := LayerOutput{
		L1Signal:    nWords(20),
		L2Reasoning: nWords(80),
		L3Provenance: []string{
			"STOPP-B3",
			"BEERS-2023-§4.2",
			"KB-23-rule-042",
		},
		L4DeepAudit: strings.Repeat("audit-token ", 500),
	}
	if err := Validate(out); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}
