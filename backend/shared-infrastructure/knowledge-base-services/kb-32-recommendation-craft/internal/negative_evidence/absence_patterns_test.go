package negative_evidence_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cardiofit/kb32/internal/negative_evidence"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// AbsencePattern String() tests
// ---------------------------------------------------------------------------

func TestAbsencePattern_String(t *testing.T) {
	cases := []struct {
		pattern negative_evidence.AbsencePattern
		want    string
	}{
		{negative_evidence.PatternBoundedWindow, "bounded_window"},
		{negative_evidence.PatternPeriodicReview, "periodic_review"},
		{negative_evidence.PatternIndicationDocumentation, "indication_documentation"},
	}
	for _, tc := range cases {
		got := tc.pattern.String()
		if got != tc.want {
			t.Errorf("pattern %d String() = %q, want %q", tc.pattern, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// IsValidPattern tests
// ---------------------------------------------------------------------------

func TestIsValidPattern(t *testing.T) {
	valid := []negative_evidence.AbsencePattern{
		negative_evidence.PatternBoundedWindow,
		negative_evidence.PatternPeriodicReview,
		negative_evidence.PatternIndicationDocumentation,
	}
	for _, p := range valid {
		if !negative_evidence.IsValidPattern(p) {
			t.Errorf("IsValidPattern(%d) = false, want true", p)
		}
	}

	// Invalid sentinel values.
	invalid := []negative_evidence.AbsencePattern{0, 4, 99, -1}
	for _, p := range invalid {
		if negative_evidence.IsValidPattern(p) {
			t.Errorf("IsValidPattern(%d) = true, want false", p)
		}
	}
}

// ---------------------------------------------------------------------------
// AbsenceQuery.Validate() tests
// ---------------------------------------------------------------------------

func TestAbsenceQuery_Validate(t *testing.T) {
	residentID := uuid.New()

	t.Run("valid bounded-window query", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "fall",
			WindowDays:      90,
		}
		if err := q.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid periodic-review query (WindowDays may be zero)", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternPeriodicReview,
			ResidentID:      residentID,
			ObservationKind: "medication_review",
			WindowDays:      0, // Conventionally 365 days; caller may leave 0.
		}
		if err := q.Validate(); err != nil {
			t.Fatalf("unexpected error for periodic-review with WindowDays=0: %v", err)
		}
	})

	t.Run("valid indication-documentation query (WindowDays may be zero)", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternIndicationDocumentation,
			ResidentID:      residentID,
			ObservationKind: "ppi_indication",
			WindowDays:      0,
		}
		if err := q.Validate(); err != nil {
			t.Fatalf("unexpected error for indication-documentation with WindowDays=0: %v", err)
		}
	})

	t.Run("invalid pattern", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.AbsencePattern(99),
			ResidentID:      residentID,
			ObservationKind: "fall",
			WindowDays:      90,
		}
		if err := q.Validate(); err == nil {
			t.Fatal("expected error for invalid pattern, got nil")
		}
	})

	t.Run("empty ObservationKind", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "",
			WindowDays:      90,
		}
		if err := q.Validate(); err == nil {
			t.Fatal("expected error for empty ObservationKind, got nil")
		}
	})

	t.Run("bounded-window with WindowDays=0 is invalid", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "fall",
			WindowDays:      0,
		}
		if err := q.Validate(); err == nil {
			t.Fatal("expected error for bounded-window with WindowDays=0, got nil")
		}
	})

	t.Run("bounded-window with negative WindowDays is invalid", func(t *testing.T) {
		q := negative_evidence.AbsenceQuery{
			Pattern:         negative_evidence.PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "fall",
			WindowDays:      -5,
		}
		if err := q.Validate(); err == nil {
			t.Fatal("expected error for bounded-window with negative WindowDays, got nil")
		}
	})
}

// ---------------------------------------------------------------------------
// InMemoryQuerier happy-path tests — all 3 patterns
// ---------------------------------------------------------------------------

func TestInMemoryQuerier_BoundedWindowAbsenceConfirmed(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.NewInMemoryQuerier(nil) // nil = absence for all queries

	query := negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternBoundedWindow,
		ResidentID:      residentID,
		ObservationKind: "fall",
		WindowDays:      90,
	}
	result, err := q.QueryAbsence(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Confirmed {
		t.Error("expected Confirmed=true for absence fixture")
	}
	if result.LastSeenAt != nil {
		t.Errorf("expected LastSeenAt=nil, got %v", result.LastSeenAt)
	}
	if result.EvidenceText == "" {
		t.Error("EvidenceText must be non-empty")
	}
	if result.QueriedAt.IsZero() {
		t.Error("QueriedAt must be populated")
	}
}

func TestInMemoryQuerier_PeriodicReviewAbsenceConfirmed(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.NewInMemoryQuerier(nil)

	query := negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternPeriodicReview,
		ResidentID:      residentID,
		ObservationKind: "medication_review",
		WindowDays:      365,
	}
	result, err := q.QueryAbsence(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Confirmed {
		t.Error("expected Confirmed=true")
	}
	if result.LastSeenAt != nil {
		t.Errorf("expected LastSeenAt=nil, got %v", result.LastSeenAt)
	}
	if result.EvidenceText == "" {
		t.Error("EvidenceText must be non-empty")
	}
}

func TestInMemoryQuerier_IndicationDocumentationAbsenceConfirmed(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.NewInMemoryQuerier(nil)

	query := negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternIndicationDocumentation,
		ResidentID:      residentID,
		ObservationKind: "ppi_indication",
		WindowDays:      0,
	}
	result, err := q.QueryAbsence(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Confirmed {
		t.Error("expected Confirmed=true")
	}
	if result.LastSeenAt != nil {
		t.Errorf("expected LastSeenAt=nil, got %v", result.LastSeenAt)
	}
	if result.EvidenceText == "" {
		t.Error("EvidenceText must be non-empty")
	}
}

// ---------------------------------------------------------------------------
// InMemoryQuerier presence-detected test
// ---------------------------------------------------------------------------

func TestInMemoryQuerier_PresenceDetected(t *testing.T) {
	residentID := uuid.New()
	lastSeen := time.Now().UTC().Add(-48 * time.Hour) // 2 days ago — within 90-day window

	q := negative_evidence.NewInMemoryQuerier(&lastSeen) // non-nil = presence detected

	query := negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternBoundedWindow,
		ResidentID:      residentID,
		ObservationKind: "fall",
		WindowDays:      90,
	}
	result, err := q.QueryAbsence(context.Background(), query)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Confirmed {
		t.Error("expected Confirmed=false when presence is detected")
	}
	if result.LastSeenAt == nil {
		t.Fatal("expected LastSeenAt to be set when presence detected")
	}
	if !result.LastSeenAt.Equal(lastSeen) {
		t.Errorf("LastSeenAt = %v, want %v", result.LastSeenAt, lastSeen)
	}
	if result.EvidenceText == "" {
		t.Error("EvidenceText must be non-empty even when presence detected")
	}
}

// ---------------------------------------------------------------------------
// QueriedAt is populated as time.Now().UTC()
// ---------------------------------------------------------------------------

func TestInMemoryQuerier_QueriedAtPopulated(t *testing.T) {
	before := time.Now().UTC()
	q := negative_evidence.NewInMemoryQuerier(nil)

	result, err := q.QueryAbsence(context.Background(), negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternBoundedWindow,
		ResidentID:      uuid.New(),
		ObservationKind: "fall",
		WindowDays:      90,
	})
	after := time.Now().UTC()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.QueriedAt.Before(before) || result.QueriedAt.After(after) {
		t.Errorf("QueriedAt %v not within [%v, %v]", result.QueriedAt, before, after)
	}
}

// ---------------------------------------------------------------------------
// InMemoryQuerier error propagation
// ---------------------------------------------------------------------------

func TestInMemoryQuerier_ErrorPropagation(t *testing.T) {
	sentinel := errors.New("injected querier error")
	q := negative_evidence.NewInMemoryQuerierWithError(sentinel)

	_, err := q.QueryAbsence(context.Background(), negative_evidence.AbsenceQuery{
		Pattern:         negative_evidence.PatternBoundedWindow,
		ResidentID:      uuid.New(),
		ObservationKind: "fall",
		WindowDays:      90,
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
