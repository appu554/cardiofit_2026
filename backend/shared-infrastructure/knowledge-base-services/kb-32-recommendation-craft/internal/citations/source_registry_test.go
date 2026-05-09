package citations

import (
	"context"
	"testing"
	"time"
)

// testTime is a fixed reference time for deterministic test scenarios.
var testTime = time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// IsValidStatus boundary tests
// ---------------------------------------------------------------------------

func TestIsValidStatus(t *testing.T) {
	valid := []string{"active", "amended", "retracted", "superseded"}
	for _, s := range valid {
		if !IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = false, want true", s)
		}
	}

	invalid := []string{"", "Active", "ACTIVE", "unknown", "pending", "deleted"}
	for _, s := range invalid {
		if IsValidStatus(s) {
			t.Errorf("IsValidStatus(%q) = true, want false", s)
		}
	}
}

// ---------------------------------------------------------------------------
// SourceVersion.ActiveAt boundary tests
// ---------------------------------------------------------------------------

func TestActiveAt(t *testing.T) {
	eff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	sv := SourceVersion{
		SourceID:      "src-1",
		Version:       "1",
		EffectiveFrom: eff,
		EffectiveTo:   &exp,
		Status:        StatusActive,
	}

	cases := []struct {
		name string
		asOf time.Time
		want bool
	}{
		{"before EffectiveFrom", eff.Add(-time.Second), false},
		{"equal EffectiveFrom (inclusive)", eff, true},
		{"mid window", eff.Add(24 * time.Hour), true},
		{"equal EffectiveTo (exclusive)", exp, false},
		{"after EffectiveTo", exp.Add(time.Second), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sv.ActiveAt(tc.asOf)
			if got != tc.want {
				t.Errorf("ActiveAt(%v) = %v, want %v", tc.asOf, got, tc.want)
			}
		})
	}

	// Open interval (EffectiveTo == nil).
	open := SourceVersion{
		SourceID:      "src-2",
		Version:       "1",
		EffectiveFrom: eff,
		EffectiveTo:   nil,
		Status:        StatusActive,
	}
	if !open.ActiveAt(exp.Add(100 * 24 * time.Hour)) {
		t.Error("open-interval version: ActiveAt far future = false, want true")
	}
	if open.ActiveAt(eff.Add(-time.Second)) {
		t.Error("open-interval version: ActiveAt before EffectiveFrom = true, want false")
	}
}

// ---------------------------------------------------------------------------
// SourceVersion roundtrip via InMemoryRegistry
// ---------------------------------------------------------------------------

func TestSourceVersionRoundtrip(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	sv := SourceVersion{
		SourceID:      "guideline-ACC-2024",
		Version:       "1",
		EffectiveFrom: testTime,
		EffectiveTo:   nil,
		ContentHash:   "abc123",
		Status:        StatusActive,
	}

	if err := reg.Register(ctx, sv); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := reg.Get(ctx, sv.SourceID, sv.Version)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SourceID != sv.SourceID || got.Version != sv.Version ||
		got.ContentHash != sv.ContentHash || got.Status != sv.Status {
		t.Errorf("roundtrip mismatch: got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Register: duplicate key returns ErrVersionExists
// ---------------------------------------------------------------------------

func TestRegisterDuplicate(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	sv := SourceVersion{
		SourceID: "src-dup", Version: "1",
		EffectiveFrom: testTime, Status: StatusActive,
	}
	if err := reg.Register(ctx, sv); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := reg.Register(ctx, sv)
	if err == nil {
		t.Fatal("second Register: expected ErrVersionExists, got nil")
	}
}

// ---------------------------------------------------------------------------
// Amend workflow
// ---------------------------------------------------------------------------

func TestAmend(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	t0 := testTime
	t1 := t0.Add(30 * 24 * time.Hour) // 30 days later

	// Register initial version.
	if err := reg.Register(ctx, SourceVersion{
		SourceID: "src-a", Version: "1",
		EffectiveFrom: t0, Status: StatusActive, ContentHash: "hash-v1",
	}); err != nil {
		t.Fatalf("Register v1: %v", err)
	}

	// Amend: create v2 starting at t1.
	if err := reg.Amend(ctx, "src-a", "2", "hash-v2", t1); err != nil {
		t.Fatalf("Amend: %v", err)
	}

	// v1 should be amended with EffectiveTo = t1.
	v1, err := reg.Get(ctx, "src-a", "1")
	if err != nil {
		t.Fatalf("Get v1 after amend: %v", err)
	}
	if v1.Status != StatusAmended {
		t.Errorf("v1.Status = %q, want %q", v1.Status, StatusAmended)
	}
	if v1.EffectiveTo == nil || !v1.EffectiveTo.Equal(t1) {
		t.Errorf("v1.EffectiveTo = %v, want %v", v1.EffectiveTo, t1)
	}

	// v2 should be active.
	v2, err := reg.Get(ctx, "src-a", "2")
	if err != nil {
		t.Fatalf("Get v2: %v", err)
	}
	if v2.Status != StatusActive {
		t.Errorf("v2.Status = %q, want %q", v2.Status, StatusActive)
	}
	if v2.EffectiveTo != nil {
		t.Errorf("v2.EffectiveTo = %v, want nil (open)", v2.EffectiveTo)
	}

	// ActiveAt(beforeAmend) should return v1.
	beforeAmend := t1.Add(-time.Second)
	if !v1.ActiveAt(beforeAmend) {
		t.Error("v1.ActiveAt(beforeAmend) = false, want true")
	}

	// ActiveAt(afterAmend) should return v2.
	afterAmend := t1.Add(time.Second)
	if !v2.ActiveAt(afterAmend) {
		t.Error("v2.ActiveAt(afterAmend) = false, want true")
	}

	// At exactly t1: v1 closed (exclusive), v2 open (inclusive).
	if v1.ActiveAt(t1) {
		t.Error("v1.ActiveAt(t1) = true, want false (t1 is exclusive upper bound)")
	}
	if !v2.ActiveAt(t1) {
		t.Error("v2.ActiveAt(t1) = false, want true (t1 is inclusive lower bound)")
	}
}

// ---------------------------------------------------------------------------
// Retract workflow
// ---------------------------------------------------------------------------

func TestRetract(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	t0 := testTime
	t1 := t0.Add(30 * 24 * time.Hour)

	// Register two versions.
	if err := reg.Register(ctx, SourceVersion{
		SourceID: "src-r", Version: "1",
		EffectiveFrom: t0, EffectiveTo: &t1, Status: StatusActive,
	}); err != nil {
		t.Fatalf("Register v1: %v", err)
	}
	if err := reg.Register(ctx, SourceVersion{
		SourceID: "src-r", Version: "2",
		EffectiveFrom: t1, Status: StatusActive,
	}); err != nil {
		t.Fatalf("Register v2: %v", err)
	}

	// Retract.
	if err := reg.Retract(ctx, "src-r", "safety concern identified"); err != nil {
		t.Fatalf("Retract: %v", err)
	}

	// Both versions should be retracted.
	for _, ver := range []string{"1", "2"} {
		sv, err := reg.Get(ctx, "src-r", ver)
		if err != nil {
			t.Fatalf("Get v%s after retract: %v", ver, err)
		}
		if sv.Status != StatusRetracted {
			t.Errorf("v%s.Status = %q after retract, want %q", ver, sv.Status, StatusRetracted)
		}
	}
}

// ---------------------------------------------------------------------------
// Supersede workflow
// ---------------------------------------------------------------------------

func TestSupersede(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	t0 := testTime
	t1 := t0.Add(60 * 24 * time.Hour)

	// Register old source.
	if err := reg.Register(ctx, SourceVersion{
		SourceID: "old-src", Version: "1",
		EffectiveFrom: t0, Status: StatusActive,
	}); err != nil {
		t.Fatalf("Register old: %v", err)
	}

	// Supersede with new source.
	if err := reg.Supersede(ctx, "old-src", "new-src", t1); err != nil {
		t.Fatalf("Supersede: %v", err)
	}

	// Old source should be superseded.
	old, err := reg.Get(ctx, "old-src", "1")
	if err != nil {
		t.Fatalf("Get old after supersede: %v", err)
	}
	if old.Status != StatusSuperseded {
		t.Errorf("old.Status = %q, want %q", old.Status, StatusSuperseded)
	}
	if old.EffectiveTo == nil || !old.EffectiveTo.Equal(t1) {
		t.Errorf("old.EffectiveTo = %v, want %v", old.EffectiveTo, t1)
	}

	// New source version "1" should be active.
	newSV, err := reg.Get(ctx, "new-src", "1")
	if err != nil {
		t.Fatalf("Get new-src v1: %v", err)
	}
	if newSV.Status != StatusActive {
		t.Errorf("new-src v1.Status = %q, want %q", newSV.Status, StatusActive)
	}
	if !newSV.EffectiveFrom.Equal(t1) {
		t.Errorf("new-src v1.EffectiveFrom = %v, want %v", newSV.EffectiveFrom, t1)
	}
}

// ---------------------------------------------------------------------------
// ActiveVersion via Registry
// ---------------------------------------------------------------------------

func TestActiveVersion(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	t0 := testTime
	t1 := t0.Add(30 * 24 * time.Hour)

	if err := reg.Register(ctx, SourceVersion{
		SourceID: "src-av", Version: "1",
		EffectiveFrom: t0, EffectiveTo: &t1, Status: StatusActive,
	}); err != nil {
		t.Fatalf("Register v1: %v", err)
	}
	if err := reg.Register(ctx, SourceVersion{
		SourceID: "src-av", Version: "2",
		EffectiveFrom: t1, Status: StatusActive,
	}); err != nil {
		t.Fatalf("Register v2: %v", err)
	}

	// Before t1 → v1.
	sv, err := reg.ActiveVersion(ctx, "src-av", t0.Add(time.Hour))
	if err != nil {
		t.Fatalf("ActiveVersion before t1: %v", err)
	}
	if sv.Version != "1" {
		t.Errorf("ActiveVersion before t1: version = %q, want %q", sv.Version, "1")
	}

	// After t1 → v2.
	sv, err = reg.ActiveVersion(ctx, "src-av", t1.Add(time.Hour))
	if err != nil {
		t.Fatalf("ActiveVersion after t1: %v", err)
	}
	if sv.Version != "2" {
		t.Errorf("ActiveVersion after t1: version = %q, want %q", sv.Version, "2")
	}

	// No active version before t0 → ErrNoActiveVersion.
	_, err = reg.ActiveVersion(ctx, "src-av", t0.Add(-time.Second))
	if err == nil {
		t.Fatal("ActiveVersion before t0: expected ErrNoActiveVersion, got nil")
	}
}

// ---------------------------------------------------------------------------
// ListVersions ordering
// ---------------------------------------------------------------------------

func TestListVersions(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	t0 := testTime
	t1 := t0.Add(10 * 24 * time.Hour)
	t2 := t0.Add(20 * 24 * time.Hour)

	for _, sv := range []SourceVersion{
		{SourceID: "src-lv", Version: "2", EffectiveFrom: t1, Status: StatusAmended},
		{SourceID: "src-lv", Version: "1", EffectiveFrom: t0, Status: StatusAmended},
		{SourceID: "src-lv", Version: "3", EffectiveFrom: t2, Status: StatusActive},
	} {
		if err := reg.Register(ctx, sv); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	list, err := reg.ListVersions(ctx, "src-lv")
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("ListVersions: got %d items, want 3", len(list))
	}
	want := []string{"1", "2", "3"}
	for i, sv := range list {
		if sv.Version != want[i] {
			t.Errorf("ListVersions[%d].Version = %q, want %q", i, sv.Version, want[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Citation save/get/list
// ---------------------------------------------------------------------------

func TestCitationRoundtrip(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	c := RecommendationCitation{
		RecommendationID: "rec-001",
		SourceID:         "src-1",
		Version:          "1",
		PinnedAt:         testTime,
	}
	if err := reg.SaveCitation(ctx, c); err != nil {
		t.Fatalf("SaveCitation: %v", err)
	}

	got, err := reg.GetCitation(ctx, c.RecommendationID, c.SourceID, c.Version)
	if err != nil {
		t.Fatalf("GetCitation: %v", err)
	}
	if got != c {
		t.Errorf("GetCitation: got %+v, want %+v", got, c)
	}

	list, err := reg.ListCitations(ctx, c.RecommendationID)
	if err != nil {
		t.Fatalf("ListCitations: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListCitations: got %d, want 1", len(list))
	}
}
