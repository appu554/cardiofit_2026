// Wave 6.1 — Failure Mode 3: intent field sparseness.
//
// Layer 2 doc Part 6 Failure 3: "if Intent.Category is unknown, rules
// that depend on intent (e.g. 'no antipsychotic for primary insomnia
// without explicit indication') would either misfire or silently
// suppress. Defence: rules with intent_required must SUPPRESS (not
// fire) when intent_class='unknown'."
//
// Pure-logic test: a mock rule predicate enforces the intent_required
// contract; we drive it with three MedicineUse fixtures (intent set,
// intent unknown, intent missing) and assert the suppress rule.
package failure_modes

import (
	"testing"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// intentRequiredRule simulates a rule predicate that fires only when
// MedicineUse.Intent.Category is non-empty AND not "unknown". Returns
// true if the rule WOULD fire; false if it suppresses.
//
// This mirrors the contract Layer 3 will publish for rules tagged
// `intent_required: true`.
func intentRequiredRule(mu models.MedicineUse) (fires bool, suppressedReason string) {
	if mu.Intent.Category == "" {
		return false, "intent.category empty — suppress (Failure 3 defence)"
	}
	if mu.Intent.Category == "unknown" {
		return false, "intent.category=unknown — suppress (Failure 3 defence)"
	}
	return true, ""
}

func TestFailure3_IntentRequiredRule_FiresWhenIntentSet(t *testing.T) {
	mu := models.MedicineUse{
		Intent: models.Intent{Category: "primary_indication", Indication: "Hypertension"},
	}
	fires, reason := intentRequiredRule(mu)
	if !fires {
		t.Fatalf("with explicit intent the rule must fire; suppressed: %s", reason)
	}
}

func TestFailure3_IntentRequiredRule_SuppressesWhenIntentUnknown(t *testing.T) {
	mu := models.MedicineUse{
		Intent: models.Intent{Category: "unknown", Indication: ""},
	}
	fires, reason := intentRequiredRule(mu)
	if fires {
		t.Fatal("rule MUST suppress when intent_class='unknown' (Failure 3 defence)")
	}
	if reason == "" {
		t.Fatal("suppression reason should be non-empty for audit visibility")
	}
}

func TestFailure3_IntentRequiredRule_SuppressesWhenIntentEmpty(t *testing.T) {
	mu := models.MedicineUse{Intent: models.Intent{}}
	fires, reason := intentRequiredRule(mu)
	if fires {
		t.Fatal("rule MUST suppress when intent.category is empty (Failure 3 defence)")
	}
	if reason == "" {
		t.Fatal("expected suppression reason")
	}
}
