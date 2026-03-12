package services

import (
	"math"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

func newTestCMEffectProcessor() *CMEffectProcessor {
	return NewCMEffectProcessor(testLogger())
}

// --- G5: Extract tests ---

func TestExtract_HardBlock(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{
			ModifierID:       "CM06",
			Effect:           models.CMEffectHardBlock,
			BlockedTreatment: "NITRATE_THERAPY",
			DrugClass:        "PDE5I",
			Differentials:    []string{"ACS"},
		},
	}

	result := p.Extract(mods)

	if len(result.Contraindications) != 1 {
		t.Fatalf("expected 1 contraindication, got %d", len(result.Contraindications))
	}
	ci := result.Contraindications[0]
	if ci.ModifierID != "CM06" {
		t.Errorf("expected modifier_id CM06, got %s", ci.ModifierID)
	}
	if ci.BlockedTreatment != "NITRATE_THERAPY" {
		t.Errorf("expected NITRATE_THERAPY, got %s", ci.BlockedTreatment)
	}
	if ci.DrugClass != "PDE5I" {
		t.Errorf("expected drug_class PDE5I, got %s", ci.DrugClass)
	}
	if len(result.Overrides) != 0 {
		t.Errorf("expected 0 overrides, got %d", len(result.Overrides))
	}
}

func TestExtract_HardBlock_MissingTreatment_Skipped(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{ModifierID: "BAD_BLOCK", Effect: models.CMEffectHardBlock},
	}

	result := p.Extract(mods)
	if len(result.Contraindications) != 0 {
		t.Errorf("HARD_BLOCK without blocked_treatment should be skipped")
	}
}

func TestExtract_Override(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{
			ModifierID: "CM10",
			Effect:     models.CMEffectOverride,
			OverrideTargets: map[string]float64{
				"ANAEMIA": 0.20,
			},
		},
	}

	result := p.Extract(mods)

	if len(result.Overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(result.Overrides))
	}
	ovr := result.Overrides[0]
	if ovr.ModifierID != "CM10" {
		t.Errorf("expected modifier_id CM10, got %s", ovr.ModifierID)
	}
	if ovr.DifferentialID != "ANAEMIA" {
		t.Errorf("expected ANAEMIA, got %s", ovr.DifferentialID)
	}
	if math.Abs(ovr.MinPosterior-0.20) > 1e-10 {
		t.Errorf("expected min_posterior 0.20, got %.6f", ovr.MinPosterior)
	}
}

func TestExtract_Override_MissingTargets_Skipped(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{ModifierID: "BAD_OVR", Effect: models.CMEffectOverride},
	}

	result := p.Extract(mods)
	if len(result.Overrides) != 0 {
		t.Errorf("OVERRIDE without override_targets should be skipped")
	}
}

func TestExtract_Override_OutOfRange_Skipped(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{
			ModifierID: "OVR_BAD",
			Effect:     models.CMEffectOverride,
			OverrideTargets: map[string]float64{
				"ACS": 0.0,  // invalid: must be > 0
				"PE":  1.0,  // invalid: must be < 1.0
				"HF":  -0.5, // invalid: must be > 0
			},
		},
	}

	result := p.Extract(mods)
	if len(result.Overrides) != 0 {
		t.Errorf("all override targets out of range, expected 0 overrides, got %d", len(result.Overrides))
	}
}

func TestExtract_MixedEffects(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{ModifierID: "CM01", Effect: "INCREASE_PRIOR", Magnitude: 0.10, Differentials: []string{"ACS"}},
		{ModifierID: "CM06", Effect: models.CMEffectHardBlock, BlockedTreatment: "NITRATE_THERAPY"},
		{ModifierID: "CM10", Effect: models.CMEffectOverride, OverrideTargets: map[string]float64{"ANAEMIA": 0.20}},
		{ModifierID: "CM03", Effect: "INCREASE_PRIOR", Magnitude: 0.05, Differentials: []string{"GERD"}},
	}

	result := p.Extract(mods)

	// Only HARD_BLOCK and OVERRIDE are extracted; INCREASE_PRIOR is ignored
	if len(result.Contraindications) != 1 {
		t.Errorf("expected 1 contraindication, got %d", len(result.Contraindications))
	}
	if len(result.Overrides) != 1 {
		t.Errorf("expected 1 override, got %d", len(result.Overrides))
	}
}

func TestExtract_Empty(t *testing.T) {
	p := newTestCMEffectProcessor()
	result := p.Extract(nil)
	if len(result.Contraindications) != 0 || len(result.Overrides) != 0 {
		t.Errorf("empty input should produce empty result")
	}
}

func TestExtract_MultipleOverrideTargets(t *testing.T) {
	p := newTestCMEffectProcessor()
	mods := []ContextModifier{
		{
			ModifierID: "CM10",
			Effect:     models.CMEffectOverride,
			OverrideTargets: map[string]float64{
				"ANAEMIA":  0.20,
				"PE":       0.15,
			},
		},
	}

	result := p.Extract(mods)
	if len(result.Overrides) != 2 {
		t.Fatalf("expected 2 overrides, got %d", len(result.Overrides))
	}
}

// --- G5: ApplyOverrides tests ---

func TestApplyOverrides_RaisesBelowMinimum(t *testing.T) {
	p := newTestCMEffectProcessor()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.30},
		{DifferentialID: "ANAEMIA", PosteriorProbability: 0.10},
		{DifferentialID: "GERD", PosteriorProbability: 0.40},
		{DifferentialID: "PE", PosteriorProbability: 0.20},
	}

	overrides := []PosteriorOverride{
		{ModifierID: "CM10", DifferentialID: "ANAEMIA", MinPosterior: 0.20},
	}

	applied := p.ApplyOverrides(posteriors, overrides)
	if !applied {
		t.Fatal("expected override to be applied")
	}

	// ANAEMIA should be raised to 0.20
	for _, entry := range posteriors {
		if entry.DifferentialID == "ANAEMIA" {
			if math.Abs(entry.PosteriorProbability-0.20) > 1e-10 {
				t.Errorf("ANAEMIA posterior should be 0.20, got %.6f", entry.PosteriorProbability)
			}
		}
	}

	// Non-overridden posteriors should sum to original_sum - deficit
	// Original non-override sum: 0.30 + 0.40 + 0.20 = 0.90
	// Deficit: 0.20 - 0.10 = 0.10
	// New non-override sum: 0.90 - 0.10 = 0.80
	nonOvrSum := 0.0
	for _, entry := range posteriors {
		if entry.DifferentialID != "ANAEMIA" {
			nonOvrSum += entry.PosteriorProbability
		}
	}
	if math.Abs(nonOvrSum-0.80) > 1e-6 {
		t.Errorf("non-override sum should be ~0.80, got %.6f", nonOvrSum)
	}
}

func TestApplyOverrides_AlreadyAboveMinimum_NoOp(t *testing.T) {
	p := newTestCMEffectProcessor()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.50},
		{DifferentialID: "ANAEMIA", PosteriorProbability: 0.30},
		{DifferentialID: "GERD", PosteriorProbability: 0.20},
	}

	overrides := []PosteriorOverride{
		{ModifierID: "CM10", DifferentialID: "ANAEMIA", MinPosterior: 0.20},
	}

	applied := p.ApplyOverrides(posteriors, overrides)
	if applied {
		t.Error("ANAEMIA is already 0.30 >= 0.20, override should not apply")
	}
}

func TestApplyOverrides_EmptyOverrides_NoOp(t *testing.T) {
	p := newTestCMEffectProcessor()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.50},
	}

	applied := p.ApplyOverrides(posteriors, nil)
	if applied {
		t.Error("nil overrides should be no-op")
	}
}

func TestApplyOverrides_EmptyPosteriors_NoOp(t *testing.T) {
	p := newTestCMEffectProcessor()
	overrides := []PosteriorOverride{
		{ModifierID: "CM10", DifferentialID: "ANAEMIA", MinPosterior: 0.20},
	}

	applied := p.ApplyOverrides(nil, overrides)
	if applied {
		t.Error("nil posteriors should be no-op")
	}
}

func TestApplyOverrides_MultipleOverrides(t *testing.T) {
	p := newTestCMEffectProcessor()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ACS", PosteriorProbability: 0.50},
		{DifferentialID: "ANAEMIA", PosteriorProbability: 0.05},
		{DifferentialID: "PE", PosteriorProbability: 0.05},
		{DifferentialID: "GERD", PosteriorProbability: 0.40},
	}

	overrides := []PosteriorOverride{
		{ModifierID: "CM10", DifferentialID: "ANAEMIA", MinPosterior: 0.15},
		{ModifierID: "CM11", DifferentialID: "PE", MinPosterior: 0.12},
	}

	applied := p.ApplyOverrides(posteriors, overrides)
	if !applied {
		t.Fatal("expected overrides to apply")
	}

	for _, entry := range posteriors {
		switch entry.DifferentialID {
		case "ANAEMIA":
			if math.Abs(entry.PosteriorProbability-0.15) > 1e-10 {
				t.Errorf("ANAEMIA should be 0.15, got %.6f", entry.PosteriorProbability)
			}
		case "PE":
			if math.Abs(entry.PosteriorProbability-0.12) > 1e-10 {
				t.Errorf("PE should be 0.12, got %.6f", entry.PosteriorProbability)
			}
		}
	}

	// Total should still sum to ~1.0
	total := 0.0
	for _, entry := range posteriors {
		total += entry.PosteriorProbability
	}
	if math.Abs(total-1.0) > 0.02 {
		t.Errorf("total posteriors should be ~1.0, got %.6f", total)
	}
}

func TestApplyOverrides_ConflictingOverrides_MaxWins(t *testing.T) {
	p := newTestCMEffectProcessor()

	posteriors := []models.DifferentialEntry{
		{DifferentialID: "ANAEMIA", PosteriorProbability: 0.05},
		{DifferentialID: "ACS", PosteriorProbability: 0.95},
	}

	// Two overrides targeting the same differential — max min_posterior should win
	overrides := []PosteriorOverride{
		{ModifierID: "CM10", DifferentialID: "ANAEMIA", MinPosterior: 0.15},
		{ModifierID: "CM11", DifferentialID: "ANAEMIA", MinPosterior: 0.25},
	}

	applied := p.ApplyOverrides(posteriors, overrides)
	if !applied {
		t.Fatal("expected override to apply")
	}

	for _, entry := range posteriors {
		if entry.DifferentialID == "ANAEMIA" {
			if math.Abs(entry.PosteriorProbability-0.25) > 1e-10 {
				t.Errorf("ANAEMIA should be 0.25 (max of conflicting overrides), got %.6f", entry.PosteriorProbability)
			}
		}
	}
}

// --- G5: ExpandNodeCMs with effect_type tests ---

func TestExpandNodeCMs_HardBlock(t *testing.T) {
	defs := []models.ContextModifierDef{
		{
			ID:               "CM06",
			Name:             "PDE5i active",
			EffectType:       models.CMEffectHardBlock,
			BlockedTreatment: "NITRATE_THERAPY",
			Adjustments:      map[string]float64{"ACS": 0.01},
		},
	}

	expanded := ExpandNodeCMs(defs)

	if len(expanded) != 1 {
		t.Fatalf("expected 1 expanded CM, got %d", len(expanded))
	}
	cm := expanded[0]
	if cm.Effect != models.CMEffectHardBlock {
		t.Errorf("expected effect HARD_BLOCK, got %s", cm.Effect)
	}
	if cm.BlockedTreatment != "NITRATE_THERAPY" {
		t.Errorf("expected blocked_treatment NITRATE_THERAPY, got %s", cm.BlockedTreatment)
	}
	if cm.ModifierID != "CM06" {
		t.Errorf("expected modifier_id CM06, got %s", cm.ModifierID)
	}
}

func TestExpandNodeCMs_Override(t *testing.T) {
	defs := []models.ContextModifierDef{
		{
			ID:         "CM10",
			Name:       "Hb<8 anaemia override",
			EffectType: models.CMEffectOverride,
			OverrideTargets: map[string]float64{
				"ANAEMIA": 0.20,
			},
		},
	}

	expanded := ExpandNodeCMs(defs)

	if len(expanded) != 1 {
		t.Fatalf("expected 1 expanded CM, got %d", len(expanded))
	}
	cm := expanded[0]
	if cm.Effect != models.CMEffectOverride {
		t.Errorf("expected effect OVERRIDE, got %s", cm.Effect)
	}
	if len(cm.OverrideTargets) != 1 {
		t.Fatalf("expected 1 override target, got %d", len(cm.OverrideTargets))
	}
	if math.Abs(cm.OverrideTargets["ANAEMIA"]-0.20) > 1e-10 {
		t.Errorf("ANAEMIA override target should be 0.20, got %.6f", cm.OverrideTargets["ANAEMIA"])
	}
}

func TestExpandNodeCMs_MixedEffectTypes(t *testing.T) {
	defs := []models.ContextModifierDef{
		{ID: "CM01", Adjustments: map[string]float64{"ACS": 0.10}},                                                                     // default INCREASE_PRIOR
		{ID: "CM06", EffectType: models.CMEffectHardBlock, BlockedTreatment: "NITRATE", Adjustments: map[string]float64{"ACS": 0.01}}, // HARD_BLOCK
		{ID: "CM10", EffectType: models.CMEffectOverride, OverrideTargets: map[string]float64{"ANAEMIA": 0.20}},                        // OVERRIDE
		{ID: "CM02", Adjustments: map[string]float64{"GERD": 0.05, "MSK": 0.03}},                                                      // default, 2 entries
	}

	expanded := ExpandNodeCMs(defs)

	// CM01 -> 1, CM06 -> 1, CM10 -> 1, CM02 -> 2 = 5 total
	if len(expanded) != 5 {
		t.Fatalf("expected 5 expanded CMs, got %d", len(expanded))
	}

	// Verify effect types
	effectCounts := make(map[string]int)
	for _, cm := range expanded {
		effectCounts[cm.Effect]++
	}
	if effectCounts[models.CMEffectIncreasePrior] != 3 {
		t.Errorf("expected 3 INCREASE_PRIOR, got %d", effectCounts[models.CMEffectIncreasePrior])
	}
	if effectCounts[models.CMEffectHardBlock] != 1 {
		t.Errorf("expected 1 HARD_BLOCK, got %d", effectCounts[models.CMEffectHardBlock])
	}
	if effectCounts[models.CMEffectOverride] != 1 {
		t.Errorf("expected 1 OVERRIDE, got %d", effectCounts[models.CMEffectOverride])
	}
}
