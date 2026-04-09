package services

import (
	"strings"
	"testing"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// TestBuildEvidenceChain_HbA1cInertia
// ---------------------------------------------------------------------------

// Verifies that a glycaemic inertia verdict with 200 days, HbA1c 8.5 vs
// target 7.0 produces a non-empty summary, a risk statement mentioning
// microvascular risk, and a guideline reference citing ADA.
func TestBuildEvidenceChain_HbA1cInertia(t *testing.T) {
	verdict := models.InertiaVerdict{
		Domain:              models.DomainGlycaemic,
		Pattern:             models.PatternHbA1cInertia,
		Detected:            true,
		InertiaDurationDays: 200,
		CurrentValue:        8.5,
		TargetValue:         7.0,
		Severity:            models.SeverityModerate,
	}

	chain := BuildEvidenceChain(verdict)

	// Summary must not be empty
	if chain.Summary == "" {
		t.Error("Summary: want non-empty")
	}

	// Summary should mention the HbA1c values
	if !strings.Contains(chain.Summary, "8.5") {
		t.Errorf("Summary should contain current value 8.5, got: %s", chain.Summary)
	}
	if !strings.Contains(chain.Summary, "7.0") {
		t.Errorf("Summary should contain target value 7.0, got: %s", chain.Summary)
	}

	// RiskStatement must mention microvascular
	if !strings.Contains(chain.RiskStatement, "microvascular") {
		t.Errorf("RiskStatement should contain 'microvascular', got: %s", chain.RiskStatement)
	}

	// GuidelineRef must mention ADA
	if !strings.Contains(chain.GuidelineRef, "ADA") {
		t.Errorf("GuidelineRef should contain 'ADA', got: %s", chain.GuidelineRef)
	}
}
