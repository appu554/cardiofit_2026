package algorithmic_distinction

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestOverridePathwayAvailable is a Phase 3 (tightened) Task 5 CI invariance
// gate per Ethical Architecture Implementation Guidelines v1.0 Principle 4
// (pharmacist autonomy / override pathway availability).
//
// Every PlatformSuggestion observation MUST support .Confirm() and transition
// cleanly to ClassHybrid. This is the structural invariant: future code that
// changes Confirm to error for valid PlatformSuggestion inputs, or removes
// the Hybrid transition, breaks this test and blocks merge.
//
// Located here rather than the plan's shared/v2_substrate/ethics/ci_gates/
// because algorithmic_distinction is an internal package of the
// pharmacist-self-visibility module. The CI-gate property is preserved:
// any failing go test blocks merge.
func TestOverridePathwayAvailable(t *testing.T) {
	origin := "pattern-detector:adherence-drift"
	obs := Observation{
		ID:                uuid.New(),
		Class:             ClassPlatformSuggestion,
		PharmacistID:      uuid.New(),
		Body:              "Suggestion: review patient adherence pattern over the last 14 days.",
		AlgorithmicOrigin: &origin,
		CreatedAt:         time.Now().UTC(),
	}

	confirmed, err := obs.Confirm(uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("PlatformSuggestion must support Confirm; got error: %v", err)
	}
	if confirmed.Class != ClassHybrid {
		t.Errorf("Confirm must transition to ClassHybrid; got %v", confirmed.Class)
	}
	if !confirmed.IsConfirmed() {
		t.Errorf("post-Confirm IsConfirmed() must return true (Class=Hybrid + ConfirmedBy + ConfirmedAt)")
	}
}

// TestOverridePathwayNonSuggestionsRejected is a sibling guard: the three
// other classes must NOT be confirmable. This pins the boundary so future
// "helpful" extensions don't accidentally let any class transition to Hybrid.
func TestOverridePathwayNonSuggestionsRejected(t *testing.T) {
	nonSuggestions := []Class{
		ClassSubstrateFact,
		ClassPharmacistReflection,
		ClassHybrid,
	}
	for _, c := range nonSuggestions {
		obs := Observation{
			ID:           uuid.New(),
			Class:        c,
			PharmacistID: uuid.New(),
			Body:         "non-suggestion observation",
			CreatedAt:    time.Now().UTC(),
		}
		if _, err := obs.Confirm(uuid.New(), time.Now()); err == nil {
			t.Errorf("class %q must not be confirmable; got nil error", c)
		}
	}
}
