package framing_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/cardiofit/kb32/internal/framing"
)

// ---------------------------------------------------------------------------
// Stub ObservationSource
// ---------------------------------------------------------------------------

// stubSource implements framing.ObservationSource for tests.
// Zero value: not opted out, no pattern, RecordObservation succeeds.
type stubSource struct {
	optedOut bool
	optErr   error

	pattern    *framing.FramingPattern
	patternErr error

	recordErr error
	recorded  []recordedCall
}

type recordedCall struct {
	gpID, tone, outcome string
}

func (s *stubSource) HasOptedOut(_ context.Context, _ string) (bool, error) {
	return s.optedOut, s.optErr
}

func (s *stubSource) PatternFor(_ context.Context, _ string) (*framing.FramingPattern, error) {
	return s.pattern, s.patternErr
}

func (s *stubSource) RecordObservation(_ context.Context, gpID, tone, outcome string) error {
	s.recorded = append(s.recorded, recordedCall{gpID, tone, outcome})
	return s.recordErr
}

// ---------------------------------------------------------------------------
// Suggest tests
// ---------------------------------------------------------------------------

func TestSuggest_UnderThirtyObsReturnsDefault(t *testing.T) {
	src := &stubSource{
		pattern: &framing.FramingPattern{
			GPID:             "gp-001",
			BestFramingTone:  "concise",
			ObservationCount: 29, // one below threshold
		},
	}
	obs := framing.NewPerGPObserver(src)
	tone, err := obs.Suggest(context.Background(), "gp-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tone != "default" {
		t.Errorf("got %q, want %q for ObservationCount=29", tone, "default")
	}
}

func TestSuggest_AtThirtyObsReturnsLearned(t *testing.T) {
	src := &stubSource{
		pattern: &framing.FramingPattern{
			GPID:             "gp-002",
			BestFramingTone:  "detailed",
			ObservationCount: 30, // exactly at threshold — boundary inclusive
		},
	}
	obs := framing.NewPerGPObserver(src)
	tone, err := obs.Suggest(context.Background(), "gp-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tone != "detailed" {
		t.Errorf("got %q, want %q for ObservationCount=30", tone, "detailed")
	}
}

func TestSuggest_OptedOutReturnsDefault(t *testing.T) {
	// GP has 100 observations but has opted out — must still return error.
	src := &stubSource{
		optedOut: true,
		pattern: &framing.FramingPattern{
			GPID:             "gp-003",
			BestFramingTone:  "collaborative",
			ObservationCount: 100,
		},
	}
	obs := framing.NewPerGPObserver(src)
	_, err := obs.Suggest(context.Background(), "gp-003")
	if !errors.Is(err, framing.ErrFramingOptedOut) {
		t.Errorf("got err=%v, want ErrFramingOptedOut", err)
	}
}

func TestSuggest_OptedOutChecksFirst(t *testing.T) {
	// patternErr would propagate IF PatternFor were called. Since opt-out check
	// happens first, PatternFor should never be reached.
	src := &stubSource{
		optedOut:   true,
		patternErr: errors.New("should not be reached"),
	}
	obs := framing.NewPerGPObserver(src)
	_, err := obs.Suggest(context.Background(), "gp-004")
	if !errors.Is(err, framing.ErrFramingOptedOut) {
		t.Errorf("expected ErrFramingOptedOut before pattern lookup; got %v", err)
	}
	// Confirm patternErr was NOT the error returned (opt-out check was first).
	if err != nil && err.Error() == "should not be reached" {
		t.Error("PatternFor was called despite opt-out — opt-out check must run first")
	}
}

func TestSuggest_OptOutSourceErrorPropagates(t *testing.T) {
	wantErr := errors.New("db unavailable")
	src := &stubSource{
		optErr: wantErr,
	}
	obs := framing.NewPerGPObserver(src)
	_, err := obs.Suggest(context.Background(), "gp-005")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected opt-out source error to propagate; got %v", err)
	}
}

func TestSuggest_PatternSourceErrorPropagates(t *testing.T) {
	wantErr := errors.New("pattern store error")
	src := &stubSource{
		patternErr: wantErr,
	}
	obs := framing.NewPerGPObserver(src)
	_, err := obs.Suggest(context.Background(), "gp-006")
	if !errors.Is(err, wantErr) {
		t.Errorf("expected pattern source error to propagate; got %v", err)
	}
}

func TestSuggest_NilPatternReturnsDefault(t *testing.T) {
	// No observations recorded yet → PatternFor returns nil.
	src := &stubSource{pattern: nil}
	obs := framing.NewPerGPObserver(src)
	tone, err := obs.Suggest(context.Background(), "gp-new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tone != "default" {
		t.Errorf("got %q, want %q for nil pattern", tone, "default")
	}
}

// ---------------------------------------------------------------------------
// Observe tests
// ---------------------------------------------------------------------------

func TestObserve_ValidOutcomes(t *testing.T) {
	outcomes := []string{"accepted", "declined", "deferred"}
	for _, outcome := range outcomes {
		src := &stubSource{}
		obs := framing.NewPerGPObserver(src)
		err := obs.Observe(context.Background(), "gp-007", "concise", outcome)
		if err != nil {
			t.Errorf("Observe with outcome=%q returned unexpected error: %v", outcome, err)
		}
		if len(src.recorded) != 1 {
			t.Errorf("expected 1 recorded call for outcome=%q, got %d", outcome, len(src.recorded))
		}
	}
}

func TestObserve_InvalidOutcomeRejected(t *testing.T) {
	src := &stubSource{}
	obs := framing.NewPerGPObserver(src)
	err := obs.Observe(context.Background(), "gp-008", "concise", "garbage")
	if !errors.Is(err, framing.ErrInvalidOutcome) {
		t.Errorf("expected ErrInvalidOutcome for outcome=%q; got %v", "garbage", err)
	}
	if len(src.recorded) != 0 {
		t.Error("RecordObservation must not be called for invalid outcome")
	}
}

func TestObserve_RecordPassesThrough(t *testing.T) {
	src := &stubSource{}
	obs := framing.NewPerGPObserver(src)
	if err := obs.Observe(context.Background(), "gp-009", "detailed", "accepted"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(src.recorded) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(src.recorded))
	}
	got := src.recorded[0]
	if got.gpID != "gp-009" || got.tone != "detailed" || got.outcome != "accepted" {
		t.Errorf("RecordObservation called with wrong args: %+v", got)
	}
}

// ---------------------------------------------------------------------------
// IsValidFramingTone
// ---------------------------------------------------------------------------

func TestIsValidFramingTone(t *testing.T) {
	cases := []struct {
		tone   string
		wantOK bool
	}{
		{"concise", true},
		{"detailed", true},
		{"collaborative", true},
		{"default", true},
		{"", false},
		{"Concise", false},   // case-sensitive
		{"DETAILED", false},
		{"informal", false},
		{"urgent", false},
	}
	for _, tc := range cases {
		got := framing.IsValidFramingTone(tc.tone)
		if got != tc.wantOK {
			t.Errorf("IsValidFramingTone(%q) = %v, want %v", tc.tone, got, tc.wantOK)
		}
	}
}

// ---------------------------------------------------------------------------
// Schema integrity: migration 040 must NOT contain pharmacist_id
// ---------------------------------------------------------------------------

// TestSchema_NoPharmacistIDColumn opens migration 040 and fails if it contains
// the string "pharmacist_id". This is the architectural toxicity guard test:
// if a future engineer accidentally adds a pharmacist_id column to the
// per_gp_framing_observations migration it is caught here before it ships.
func TestSchema_NoPharmacistIDColumn(t *testing.T) {
	// Walk up from the test file's package directory to find the migrations directory.
	// The migrations directory lives at:
	//   backend/shared-infrastructure/knowledge-base-services/migrations/
	// relative to the kb-32 module root. We locate it via the well-known path
	// segment used in this repository.
	migrationPath := findMigration040(t)
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("could not read migration 040: %v", err)
	}
	// Check each non-comment line for "pharmacist_id" as a column definition.
	// We look specifically for "pharmacist_id" appearing as a column identifier
	// in a CREATE TABLE body — i.e., a line that looks like a column definition.
	// Comment lines ("--") and COMMENT ON ... string continuations are skipped
	// because they may legitimately mention the prohibition notice.
	//
	// The strict check: does any non-comment line contain "pharmacist_id" followed
	// by a SQL type keyword? This catches accidental column addition while allowing
	// the prohibition notice in the COMMENT ON TABLE statement.
	lines := strings.Split(string(data), "\n")
	inCommentOn := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Skip pure SQL comment lines.
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		// Track multi-line COMMENT ON TABLE statement (starts with COMMENT, ends with ;).
		if strings.HasPrefix(strings.ToUpper(trimmed), "COMMENT ON") {
			inCommentOn = true
		}
		if inCommentOn {
			if strings.HasSuffix(trimmed, ";") {
				inCommentOn = false
			}
			continue // skip all lines of the COMMENT ON block
		}
		// For all other lines, check whether pharmacist_id appears as a column name.
		// Column definitions look like: `    pharmacist_id    UUID ...`
		// We check that the token appears and is followed by a space then a type.
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "pharmacist_id") {
			t.Errorf(
				"TOXICITY GUARD FAILURE: migration 040 DDL contains 'pharmacist_id'.\n"+
					"File: %s\n"+
					"Line: %q\n"+
					"The per_gp_framing_observations table MUST NOT define a pharmacist_id column.\n"+
					"Per Guidelines §8, per-pharmacist attribution is architecturally prohibited.\n"+
					"Remove the pharmacist_id column from the migration immediately.",
				migrationPath, line,
			)
		}
	}
}

// findMigration040 searches for the migration file by walking up from the test
// binary's working directory. In `go test` the cwd is the package directory.
func findMigration040(t *testing.T) string {
	t.Helper()
	// Candidate relative paths from kb-32 package directory upward.
	candidates := []string{
		// Relative from kb-32-recommendation-craft/internal/framing/
		"../../../../../../../shared-infrastructure/knowledge-base-services/migrations/040_per_gp_framing_observations.sql",
		// From kb-32 module root (go test -v ./internal/framing/...)
		"../../../../migrations/040_per_gp_framing_observations.sql",
		// Additional fallback: relative to knowledge-base-services root
		"../migrations/040_per_gp_framing_observations.sql",
	}
	// Also try absolute-ish discovery by going up from cwd.
	cwd, _ := os.Getwd()
	// Build path from cwd by traversing up to find migrations/
	// cwd is typically: .../kb-32-recommendation-craft/internal/framing
	// We need:         .../migrations/040_...
	base := cwd
	for i := 0; i < 6; i++ {
		candidate := base + "/migrations/040_per_gp_framing_observations.sql"
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		// Go up one level.
		parent := parentDir(base)
		if parent == base {
			break
		}
		base = parent
	}
	// Try the static relative candidates.
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Fatal("could not locate migration 040 file; ensure tests are run from the kb-32 module directory")
	return ""
}

// parentDir returns the parent directory of path.
func parentDir(path string) string {
	if path == "" || path == "/" {
		return path
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return path
}
