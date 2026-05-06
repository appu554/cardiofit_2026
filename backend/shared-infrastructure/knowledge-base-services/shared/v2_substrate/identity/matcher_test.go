package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fixtureLookup is an in-memory IdentityCandidateLookup used by the
// matcher unit tests. The matcher's pure-function shape means we never
// need a database to exercise the algorithm.
type fixtureLookup struct {
	byIHI       map[string]uuid.UUID
	byMedicare  map[string][]ResidentCandidate
	byFacDOB    map[string][]ResidentCandidate // key = facility|RFC3339-date
	failNotFnd  bool                           // when true, returns errors.New("v2_substrate: entity not found") on miss instead of (nil, nil)
}

func newFixtureLookup() *fixtureLookup {
	return &fixtureLookup{
		byIHI:      map[string]uuid.UUID{},
		byMedicare: map[string][]ResidentCandidate{},
		byFacDOB:   map[string][]ResidentCandidate{},
	}
}

func facDOBKey(facility uuid.UUID, dob time.Time) string {
	return facility.String() + "|" + dob.UTC().Format("2006-01-02")
}

func (f *fixtureLookup) LookupByIHI(_ context.Context, ihi string) (*uuid.UUID, error) {
	if rid, ok := f.byIHI[ihi]; ok {
		return &rid, nil
	}
	if f.failNotFnd {
		return nil, errors.New("v2_substrate: entity not found")
	}
	return nil, nil
}

func (f *fixtureLookup) LookupByMedicare(_ context.Context, medicare string) ([]ResidentCandidate, error) {
	return f.byMedicare[medicare], nil
}

func (f *fixtureLookup) LookupByFacilityAndDOB(_ context.Context, facility uuid.UUID, dob time.Time) ([]ResidentCandidate, error) {
	return f.byFacDOB[facDOBKey(facility, dob)], nil
}

// ---------------------------------------------------------------------------
// Levenshtein
// ---------------------------------------------------------------------------

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"smith", "smith", 0},
		{"smith", "smiht", 2}, // transposition = 2 classic Levenshtein edits
		{"", "abc", 3},
		{"abc", "", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3}, // canonical example
		{"Müller", "muller", 2},  // umlaut + case difference (rune-correct)
	}
	for _, tc := range cases {
		got := levenshtein(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("levenshtein(%q,%q) = %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestCombineName(t *testing.T) {
	if got := combineName("  Margaret  ", "  Brown  "); got != "margaret brown" {
		t.Errorf("combineName lowercase/trim failed: %q", got)
	}
	if got := combineName("", "Brown"); got != "brown" {
		t.Errorf("combineName empty given mishandled: %q", got)
	}
	if got := combineName("Margaret", ""); got != "margaret" {
		t.Errorf("combineName empty family mishandled: %q", got)
	}
}

// ---------------------------------------------------------------------------
// Match — IHI exact path
// ---------------------------------------------------------------------------

func TestMatch_IHIExact_High(t *testing.T) {
	rid := uuid.New()
	lookup := newFixtureLookup()
	lookup.byIHI["8003608000000570"] = rid

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		IHI: "8003608000000570",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceHigh {
		t.Errorf("Confidence: got %s want HIGH", got.Confidence)
	}
	if got.Path != MatchPathIHI {
		t.Errorf("Path: got %s want ihi", got.Path)
	}
	if got.RequiresReview {
		t.Errorf("RequiresReview should be false for HIGH")
	}
	if got.ResidentRef == nil || *got.ResidentRef != rid {
		t.Errorf("ResidentRef: got %v want %s", got.ResidentRef, rid)
	}
}

func TestMatch_IHIUnknown_FallsThrough(t *testing.T) {
	rid := uuid.New()
	fac := uuid.New()
	dob := time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC)

	lookup := newFixtureLookup()
	// Provide a name+dob+facility candidate so the IHI miss falls through
	// to LOW (verifying the matcher didn't short-circuit on IHI absence).
	lookup.byFacDOB[facDOBKey(fac, dob)] = []ResidentCandidate{
		{ID: rid, GivenName: "Margaret", FamilyName: "Brown", DOB: dob},
	}

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		IHI:        "8003608000099999", // unknown
		GivenName:  "Margaret",
		FamilyName: "Brown",
		DOB:        dob,
		FacilityID: &fac,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceLow {
		t.Errorf("Confidence: got %s want LOW", got.Confidence)
	}
	if got.Path != MatchPathNameDOBFacility {
		t.Errorf("Path: got %s want name+dob+facility", got.Path)
	}
}

func TestMatch_IHIUnknown_NotFoundError_FallsThrough(t *testing.T) {
	// Same as above but the lookup returns the kb-20 ErrNotFound sentinel
	// rather than (nil, nil). Verifies isNotFoundErr's string match.
	rid := uuid.New()
	fac := uuid.New()
	dob := time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC)

	lookup := newFixtureLookup()
	lookup.failNotFnd = true
	lookup.byFacDOB[facDOBKey(fac, dob)] = []ResidentCandidate{
		{ID: rid, GivenName: "Margaret", FamilyName: "Brown", DOB: dob},
	}

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		IHI:        "8003608000099999",
		GivenName:  "Margaret",
		FamilyName: "Brown",
		DOB:        dob,
		FacilityID: &fac,
	})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if got.Confidence != ConfidenceLow {
		t.Errorf("Confidence: got %s want LOW", got.Confidence)
	}
}

// ---------------------------------------------------------------------------
// Match — Medicare+name+DOB MEDIUM path
// ---------------------------------------------------------------------------

func TestMatch_MedicareNameDOB_Exact_Medium(t *testing.T) {
	rid := uuid.New()
	dob := time.Date(1942, 7, 30, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byMedicare["2950 47120 1"] = []ResidentCandidate{
		{ID: rid, GivenName: "Helen", FamilyName: "Yang", DOB: dob},
	}

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		Medicare:   "2950 47120 1",
		GivenName:  "Helen",
		FamilyName: "Yang",
		DOB:        dob,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceMedium {
		t.Errorf("Confidence: got %s want MEDIUM", got.Confidence)
	}
	if got.Path != MatchPathMedicareNameDOB {
		t.Errorf("Path: got %s want medicare+name+dob", got.Path)
	}
	if got.NameDistance != 0 {
		t.Errorf("NameDistance: got %d want 0", got.NameDistance)
	}
	if got.RequiresReview {
		t.Errorf("RequiresReview should be false for MEDIUM")
	}
}

func TestMatch_MedicareNameDOB_OneCharTypo_Medium(t *testing.T) {
	rid := uuid.New()
	dob := time.Date(1942, 7, 30, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byMedicare["X"] = []ResidentCandidate{
		{ID: rid, GivenName: "Helen", FamilyName: "Yang", DOB: dob},
	}

	// "Helen Yamg" vs "Helen Yang" → distance 1
	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		Medicare:   "X",
		GivenName:  "Helen",
		FamilyName: "Yamg",
		DOB:        dob,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceMedium {
		t.Errorf("Confidence: got %s want MEDIUM", got.Confidence)
	}
	if got.NameDistance != 1 {
		t.Errorf("NameDistance: got %d want 1", got.NameDistance)
	}
}

func TestMatch_MedicareNameDOB_OverThreshold_FallsThrough(t *testing.T) {
	rid := uuid.New()
	dob := time.Date(1942, 7, 30, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byMedicare["X"] = []ResidentCandidate{
		{ID: rid, GivenName: "Helen", FamilyName: "Yang", DOB: dob},
	}
	// No facility provided so step 3 also can't run → expect NONE.
	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		Medicare:   "X",
		GivenName:  "Hugo",        // distance("hugo yang","helen yang") = 4
		FamilyName: "Yang",
		DOB:        dob,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceNone {
		t.Errorf("Confidence: got %s want NONE (distance=4 over threshold)", got.Confidence)
	}
	if !got.RequiresReview {
		t.Errorf("RequiresReview should be true for NONE")
	}
}

func TestMatch_MedicareNameDOB_DOBMismatch_FallsThrough(t *testing.T) {
	rid := uuid.New()
	dob := time.Date(1942, 7, 30, 0, 0, 0, 0, time.UTC)
	otherDOB := time.Date(1942, 7, 31, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byMedicare["X"] = []ResidentCandidate{
		{ID: rid, GivenName: "Helen", FamilyName: "Yang", DOB: dob},
	}

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		Medicare:   "X",
		GivenName:  "Helen",
		FamilyName: "Yang",
		DOB:        otherDOB,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceNone {
		t.Errorf("Confidence: got %s want NONE (DOB mismatch + no facility)", got.Confidence)
	}
}

// ---------------------------------------------------------------------------
// Match — name+DOB+facility LOW path
// ---------------------------------------------------------------------------

func TestMatch_NameDOBFacility_TwoCharTypo_Low(t *testing.T) {
	rid := uuid.New()
	fac := uuid.New()
	dob := time.Date(1955, 3, 15, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byFacDOB[facDOBKey(fac, dob)] = []ResidentCandidate{
		{ID: rid, GivenName: "Robert", FamilyName: "OConnor", DOB: dob},
	}

	// Distance("robert oconnor","robert oconor") = 1, comfortably ≤ 3.
	// Use a 2-char difference to hit the threshold.
	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		GivenName:  "Robert",
		FamilyName: "OConor", // dropped one 'n' (distance 1)
		DOB:        dob,
		FacilityID: &fac,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceLow {
		t.Errorf("Confidence: got %s want LOW", got.Confidence)
	}
	if got.Path != MatchPathNameDOBFacility {
		t.Errorf("Path: got %s want name+dob+facility", got.Path)
	}
	if !got.RequiresReview {
		t.Errorf("RequiresReview must be true for LOW")
	}
	if got.ResidentRef == nil || *got.ResidentRef != rid {
		t.Errorf("ResidentRef mismatch")
	}
	if len(got.Candidates) != 1 || got.Candidates[0] != rid {
		t.Errorf("Candidates: got %v want [%s]", got.Candidates, rid)
	}
}

func TestMatch_NameDOBFacility_OverThreshold_NoMatch(t *testing.T) {
	rid := uuid.New()
	fac := uuid.New()
	dob := time.Date(1955, 3, 15, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byFacDOB[facDOBKey(fac, dob)] = []ResidentCandidate{
		{ID: rid, GivenName: "Robert", FamilyName: "OConnor", DOB: dob},
	}

	// "robert oconnor" vs "william smith" — distance > 3.
	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		GivenName:  "William",
		FamilyName: "Smith",
		DOB:        dob,
		FacilityID: &fac,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceNone {
		t.Errorf("Confidence: got %s want NONE", got.Confidence)
	}
	if got.Path != MatchPathNoMatch {
		t.Errorf("Path: got %s want no_match", got.Path)
	}
}

func TestMatch_NameDOBFacility_MultipleCandidates_BestWins(t *testing.T) {
	closeID := uuid.New()
	farID := uuid.New()
	fac := uuid.New()
	dob := time.Date(1955, 3, 15, 0, 0, 0, 0, time.UTC)
	lookup := newFixtureLookup()
	lookup.byFacDOB[facDOBKey(fac, dob)] = []ResidentCandidate{
		// far candidate — distance 3 (within threshold but worse)
		{ID: farID, GivenName: "Roberx", FamilyName: "OConnxr", DOB: dob},
		// close candidate — distance 0
		{ID: closeID, GivenName: "Robert", FamilyName: "OConnor", DOB: dob},
	}

	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		GivenName:  "Robert",
		FamilyName: "OConnor",
		DOB:        dob,
		FacilityID: &fac,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceLow {
		t.Errorf("Confidence: got %s want LOW", got.Confidence)
	}
	if got.ResidentRef == nil || *got.ResidentRef != closeID {
		t.Errorf("Should have chosen the closest candidate (distance=0)")
	}
	if len(got.Candidates) != 2 {
		t.Errorf("Candidates: got %d want 2 (both within threshold)", len(got.Candidates))
	}
}

// ---------------------------------------------------------------------------
// Match — no usable inputs at all
// ---------------------------------------------------------------------------

func TestMatch_NoUsableInputs_None(t *testing.T) {
	lookup := newFixtureLookup()
	got, err := NewMatcher(lookup).Match(context.Background(), IncomingIdentifier{
		// completely empty
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Confidence != ConfidenceNone {
		t.Errorf("Confidence: got %s want NONE", got.Confidence)
	}
	if !got.RequiresReview {
		t.Errorf("RequiresReview must be true for NONE")
	}
	if got.ResidentRef != nil {
		t.Errorf("ResidentRef should be nil for NONE")
	}
}

func TestMatch_NilLookup_ReturnsErrNoLookup(t *testing.T) {
	_, err := NewMatcher(nil).Match(context.Background(), IncomingIdentifier{IHI: "x"})
	if !errors.Is(err, ErrNoLookup) {
		t.Errorf("err: got %v want ErrNoLookup", err)
	}
}

// ---------------------------------------------------------------------------
// Validators
// ---------------------------------------------------------------------------

func TestIsValidConfidence(t *testing.T) {
	for _, ok := range []string{"high", "medium", "low", "none"} {
		if !IsValidConfidence(ok) {
			t.Errorf("expected %q valid", ok)
		}
	}
	for _, bad := range []string{"", "HIGH", "unknown"} {
		if IsValidConfidence(bad) {
			t.Errorf("expected %q invalid", bad)
		}
	}
}

func TestIsValidMatchPath(t *testing.T) {
	for _, ok := range []string{"ihi", "medicare+name+dob", "name+dob+facility", "no_match"} {
		if !IsValidMatchPath(ok) {
			t.Errorf("expected %q valid", ok)
		}
	}
	if IsValidMatchPath("foo") {
		t.Errorf("expected 'foo' invalid")
	}
}
