package dashboards

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// fakePortSrc — test double for PortfolioSource
// ---------------------------------------------------------------------------

type fakePortSrc struct {
	narrative    string
	scenarios    int
	narrativeErr error
	scenarioErr  error
}

func (f *fakePortSrc) Narrative(_ context.Context, _ uuid.UUID) (string, error) {
	return f.narrative, f.narrativeErr
}
func (f *fakePortSrc) ScenarioCount(_ context.Context, _ uuid.UUID) (int, error) {
	return f.scenarios, f.scenarioErr
}

// ---------------------------------------------------------------------------
// Plan verbatim tests (Task 9)
// ---------------------------------------------------------------------------

func TestPortfolio_AnonymisesByDefault(t *testing.T) {
	pharm := uuid.New()
	src := &fakePortSrc{narrative: "Worked at RACH-ABC with Dr Smith on resident John Doe.", scenarios: 3}
	p := NewPortfolio(src)
	view, _ := p.For(context.Background(), pharm, false /* not consented to identify */)
	if strings.Contains(view.Narrative, "John Doe") || strings.Contains(view.Narrative, "RACH-ABC") {
		t.Errorf("narrative should be anonymised by default; got %q", view.Narrative)
	}
}

func TestPortfolio_ScenarioCount(t *testing.T) {
	src := &fakePortSrc{narrative: "x", scenarios: 12}
	p := NewPortfolio(src)
	view, _ := p.For(context.Background(), uuid.New(), false)
	if view.ScenarioCount != 12 {
		t.Errorf("got %d", view.ScenarioCount)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 2: narrative preserved verbatim when consent given
// ---------------------------------------------------------------------------

func TestPortfolio_PreservesNarrativeWithConsent(t *testing.T) {
	original := "Worked at RACH-ABC with Dr Smith on resident John Doe."
	src := &fakePortSrc{narrative: original, scenarios: 3}
	p := NewPortfolio(src)
	view, err := p.For(context.Background(), uuid.New(), true /* identifiableConsented */)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Narrative != original {
		t.Errorf("narrative with consent must be verbatim; got %q, want %q", view.Narrative, original)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 3: regex edge cases
// ---------------------------------------------------------------------------

func TestPortfolio_RegexHandlesEdgeCases(t *testing.T) {
	// Each sub-test: input narrative, identifiableConsented=false, expected behaviour.
	cases := []struct {
		name     string
		input    string
		redacted bool   // true → the pattern must NOT appear in output
		pattern  string // the exact string that should / should not appear
	}{
		{
			name:     "two-word proper name redacted",
			input:    "Seen by John Doe last week.",
			redacted: true,
			pattern:  "John Doe",
		},
		{
			name:     "RACH facility code redacted",
			input:    "Placement at RACH-ABC was instructive.",
			redacted: true,
			pattern:  "RACH-ABC",
		},
		{
			name:     "lowercase name not redacted",
			input:    "lowercase name should remain.",
			redacted: false,
			pattern:  "lowercase name",
		},
		{
			// All-caps "ALL CAPS" — neither token matches [A-Z][a-z]+ (no lowercase
			// chars after the first capital), so it must NOT be redacted.
			name:     "all-caps not redacted",
			input:    "The ALL CAPS text stays.",
			redacted: false,
			pattern:  "ALL CAPS",
		},
		{
			// Single letter — "X" doesn't match [A-Z][a-z]+ (needs ≥1 lowercase after),
			// so a single letter followed by a word like "X Ray" would only be redacted
			// if "Ray" is also [A-Z][a-z]+. Test a lone single-cap letter by itself.
			name:     "single capital letter word not redacted",
			input:    "Exhibit A was reviewed.",
			redacted: false,
			pattern:  "A was",
		},
		{
			// "Dr Smith" — "Dr" is [A-Z][a-z]+ (D + r) and "Smith" is [A-Z][a-z]+.
			// Therefore the regex DOES match "Dr Smith" and it is redacted.
			// This is intentional: clinical titles paired with surnames are identifiable.
			name:     "Dr Smith is redacted (title+surname matches proper-noun pattern)",
			input:    "Supervised by Dr Smith at the clinic.",
			redacted: true,
			pattern:  "Dr Smith",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := &fakePortSrc{narrative: tc.input}
			p := NewPortfolio(src)
			view, err := p.For(context.Background(), uuid.New(), false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			contains := strings.Contains(view.Narrative, tc.pattern)
			if tc.redacted && contains {
				t.Errorf("pattern %q should be redacted but was found in output %q", tc.pattern, view.Narrative)
			}
			if !tc.redacted && !contains {
				t.Errorf("pattern %q should be preserved but was absent from output %q", tc.pattern, view.Narrative)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Augmentation 4: source errors propagate
// ---------------------------------------------------------------------------

func TestPortfolio_PropagatesSourceError(t *testing.T) {
	t.Run("narrative error propagates", func(t *testing.T) {
		sentinel := errors.New("narrative unavailable")
		src := &fakePortSrc{narrativeErr: sentinel}
		p := NewPortfolio(src)
		_, err := p.For(context.Background(), uuid.New(), false)
		if !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel narrative error; got %v", err)
		}
	})

	t.Run("scenario count error propagates", func(t *testing.T) {
		sentinel := errors.New("scenario count unavailable")
		src := &fakePortSrc{narrative: "some text", scenarioErr: sentinel}
		p := NewPortfolio(src)
		_, err := p.For(context.Background(), uuid.New(), false)
		if !errors.Is(err, sentinel) {
			t.Errorf("expected sentinel scenario count error; got %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Augmentation 5: context cancellation handled defensively
// ---------------------------------------------------------------------------

type cancelPortSrc struct{}

func (c *cancelPortSrc) Narrative(ctx context.Context, _ uuid.UUID) (string, error) {
	return "", ctx.Err()
}
func (c *cancelPortSrc) ScenarioCount(ctx context.Context, _ uuid.UUID) (int, error) {
	return 0, ctx.Err()
}

func TestPortfolio_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	p := NewPortfolio(&cancelPortSrc{})
	_, err := p.For(ctx, uuid.New(), false)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled; got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Augmentation 6: empty narrative returns empty PortfolioView, not an error
// ---------------------------------------------------------------------------

func TestPortfolio_EmptyNarrativeReturnsEmpty(t *testing.T) {
	src := &fakePortSrc{narrative: "", scenarios: 0}
	p := NewPortfolio(src)
	view, err := p.For(context.Background(), uuid.New(), false)
	if err != nil {
		t.Fatalf("unexpected error for empty narrative: %v", err)
	}
	if view.Narrative != "" {
		t.Errorf("empty narrative should yield empty Narrative field; got %q", view.Narrative)
	}
	if view.ScenarioCount != 0 {
		t.Errorf("ScenarioCount should be 0; got %d", view.ScenarioCount)
	}
}
