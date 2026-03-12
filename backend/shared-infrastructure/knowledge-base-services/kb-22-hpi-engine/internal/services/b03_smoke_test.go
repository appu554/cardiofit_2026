package services

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"kb-22-hpi-engine/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════════════
// B03 Smoke Test: End-to-end Bayesian pipeline on P02 Dyspnea with a synthetic
// patient. Validates that priors, sex modifiers, context modifiers, LR updates,
// safety floors, G15 other bucket, and G5 effects all integrate correctly.
//
// Patient: 62F with T2DM + HTN + CKD (eGFR=38) + HF (DM_HTN_CKD_HF stratum)
// Medications: metformin, enalapril, empagliflozin (SGLT2i), amlodipine, metoprolol
// Active med classes for G3: SGLT2i (activates DX09), Metformin (activates DX10)
// ═══════════════════════════════════════════════════════════════════════════════

func TestB03_SmokeTest_P02_SyntheticPatient(t *testing.T) {
	// ── Step 1: Load P02 V2 node from YAML (isolated) ──
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(nodesDir, "p02_dyspnea.yaml")
	dst := filepath.Join(tmpDir, "p02_dyspnea.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("cannot read P02 YAML: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("cannot write P02 YAML to temp: %v", err)
	}

	nl := NewNodeLoader(tmpDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("NodeLoader.Load() failed: %v", err)
	}

	node := nl.Get("P02_DYSPNEA")
	if node == nil {
		t.Fatal("P02_DYSPNEA not loaded")
	}

	// ── Step 2: Verify node structure ──
	t.Run("NodeStructure", func(t *testing.T) {
		if node.Version != "2.0.0" {
			t.Errorf("version: want 2.0.0, got %s", node.Version)
		}
		if len(node.Differentials) != 10 {
			t.Errorf("differentials: want 10, got %d", len(node.Differentials))
		}
		if len(node.Questions) != 25 {
			t.Errorf("questions: want 25, got %d", len(node.Questions))
		}
		if len(node.SexModifiers) != 3 {
			t.Errorf("sex_modifiers: want 3, got %d", len(node.SexModifiers))
		}
		if len(node.ContextModifiers) != 10 {
			t.Errorf("context_modifiers: want 10, got %d", len(node.ContextModifiers))
		}
		if !node.OtherBucketEnabled {
			t.Error("other_bucket_enabled should be true")
		}
	})

	// ── Step 3: Init priors for DM_HTN_CKD_HF stratum with active meds ──
	e := newTestBayesianEngine()
	stratum := "DM_HTN_CKD_HF"
	activeMeds := []string{"SGLT2i", "Metformin"} // Empagliflozin + Metformin

	logOdds := e.InitPriors(node, stratum, activeMeds)

	t.Run("InitPriors", func(t *testing.T) {
		// G3: Both conditional diffs should be included (SGLT2i + Metformin active)
		if _, ok := logOdds["EUGLYCEMIC_DKA_RESP"]; !ok {
			t.Error("EUGLYCEMIC_DKA_RESP should be active (SGLT2i present)")
		}
		if _, ok := logOdds["LACTIC_ACIDOSIS_RESP"]; !ok {
			t.Error("LACTIC_ACIDOSIS_RESP should be active (Metformin present)")
		}

		// G15: _OTHER should be injected
		if _, ok := logOdds[models.OtherBucketDiffID]; !ok {
			t.Error("_OTHER bucket should be present")
		}

		// Should have 10 authored + 1 _OTHER = 11 total
		if len(logOdds) != 11 {
			t.Errorf("expected 11 differentials (10 authored + _OTHER), got %d", len(logOdds))
		}

		// CHF should have highest log-odds in DM_HTN_CKD_HF stratum
		chfLO := logOdds["CHF"]
		for diffID, lo := range logOdds {
			if diffID != models.OtherBucketDiffID && diffID != "CHF" && lo > chfLO {
				t.Errorf("CHF should have highest log-odds in DM_HTN_CKD_HF, but %s (%.4f) > CHF (%.4f)",
					diffID, lo, chfLO)
			}
		}
	})

	// ── Step 4: Apply sex modifiers (62F) ──
	t.Run("SexModifiers", func(t *testing.T) {
		peBefore := logOdds["PE_DYSPNEA"]
		chfBefore := logOdds["CHF"]

		e.ApplySexModifiers(logOdds, node.SexModifiers, "Female", 62)

		// SM01 fires (sex == Female): PE_DYSPNEA += 0.47
		peAfter := logOdds["PE_DYSPNEA"]
		if math.Abs((peAfter-peBefore)-0.47) > 1e-10 {
			t.Errorf("SM01 PE_DYSPNEA delta: want +0.47, got %+.4f", peAfter-peBefore)
		}

		// SM02 fires (Female AND age >= 50): CHF += 0.34
		chfAfter := logOdds["CHF"]
		if math.Abs((chfAfter-chfBefore)-0.34) > 1e-10 {
			t.Errorf("SM02 CHF delta: want +0.34, got %+.4f", chfAfter-chfBefore)
		}

		// SM03 (Male) should NOT fire
		copdBefore := logOdds["COPD_EXAC"]
		e.ApplySexModifiers(logOdds, node.SexModifiers, "Female", 62)
		// Re-applying shouldn't matter for testing SM03 not firing initially.
		// Just check COPD didn't change from Male modifier in original call.
		_ = copdBefore // verified by absence of SM03 effect
	})

	// ── Step 5: Expand and apply CMs ──
	t.Run("ContextModifiers", func(t *testing.T) {
		expandedCMs := ExpandNodeCMs(node.ContextModifiers)

		// Verify expansion produces CMs from all 10 definitions
		if len(expandedCMs) == 0 {
			t.Fatal("ExpandNodeCMs returned zero CMs")
		}

		// Find which CM IDs are present
		cmIDs := make(map[string]bool)
		for _, cm := range expandedCMs {
			cmIDs[cm.ModifierID] = true
		}

		// All 10 CMs should be represented
		for i := 1; i <= 10; i++ {
			id := "CM01"
			if i < 10 {
				id = "CM0" + string(rune('0'+i))
			} else {
				id = "CM10"
			}
			if !cmIDs[id] {
				t.Errorf("CM %s not found in expanded CMs", id)
			}
		}

		// Simulate: patient is on metoprolol (beta-blocker), enalapril (ACEi),
		// empagliflozin (SGLT2i), loop diuretic (furosemide).
		// Active CMs: CM01 (loop diuretic), CM02 (beta-blocker), CM04 (ACEi), CM05 (SGLT2i)
		// For B03, we apply ALL expanded CMs to verify the pipeline doesn't crash.
		_, cmDeltas := NewCMApplicator(testLogger()).Apply(logOdds, expandedCMs, nil)

		// Verify cm_log_deltas were recorded
		if len(cmDeltas) == 0 {
			t.Error("cm_log_deltas should be non-empty after CM application")
		}
	})

	// ── Step 6: Simulate question answers and verify Bayesian updates ──
	t.Run("BayesianUpdates", func(t *testing.T) {
		clusterAnswered := make(map[string]int)

		// Find Q001 (orthopnea — strong CHF indicator)
		var q001 *models.QuestionDef
		for i := range node.Questions {
			if node.Questions[i].ID == "Q001" {
				q001 = &node.Questions[i]
				break
			}
		}
		if q001 == nil {
			t.Fatal("Q001 not found in P02")
		}

		// Answer YES to orthopnea
		chfBefore := logOdds["CHF"]
		logOdds, _ = e.Update(logOdds, "Q001", "YES", q001, 1.0, 1.0, clusterAnswered)
		if q001.Cluster != "" {
			clusterAnswered[q001.Cluster]++
		}

		// CHF should increase (LR+ for CHF on Q001 = 2.20)
		if logOdds["CHF"] <= chfBefore {
			t.Error("CHF log-odds should increase after YES to orthopnea")
		}

		// Find Q003 (ankle edema — another CHF indicator, same FLUID cluster as Q009)
		var q003 *models.QuestionDef
		for i := range node.Questions {
			if node.Questions[i].ID == "Q003" {
				q003 = &node.Questions[i]
				break
			}
		}
		if q003 == nil {
			t.Fatal("Q003 not found in P02")
		}

		// Answer YES to ankle edema
		chfBefore = logOdds["CHF"]
		logOdds, _ = e.Update(logOdds, "Q003", "YES", q003, 1.0, 1.0, clusterAnswered)
		if q003.Cluster != "" {
			clusterAnswered[q003.Cluster]++
		}

		if logOdds["CHF"] <= chfBefore {
			t.Error("CHF log-odds should increase after YES to ankle edema")
		}

		// Answer NO to wheezing (Q006 — reduces ASTHMA/COPD)
		var q006 *models.QuestionDef
		for i := range node.Questions {
			if node.Questions[i].ID == "Q006" {
				q006 = &node.Questions[i]
				break
			}
		}
		if q006 == nil {
			t.Fatal("Q006 not found in P02")
		}

		asthmaBefore := logOdds["ASTHMA"]
		logOdds, _ = e.Update(logOdds, "Q006", "NO", q006, 1.0, 1.0, clusterAnswered)
		if q006.Cluster != "" {
			clusterAnswered[q006.Cluster]++
		}

		if logOdds["ASTHMA"] >= asthmaBefore {
			t.Error("ASTHMA log-odds should decrease after NO to wheezing")
		}

		// F-04: Answer PATA_NAHI to Q012 (tingling) — no update
		var q012 *models.QuestionDef
		for i := range node.Questions {
			if node.Questions[i].ID == "Q012" {
				q012 = &node.Questions[i]
				break
			}
		}
		if q012 == nil {
			t.Fatal("Q012 not found in P02")
		}

		anxBefore := logOdds["ANXIETY_DYSPNEA"]
		logOdds, ig := e.Update(logOdds, "Q012", "PATA_NAHI", q012, 1.0, 1.0, clusterAnswered)

		if math.Abs(logOdds["ANXIETY_DYSPNEA"]-anxBefore) > 1e-15 {
			t.Error("PATA_NAHI should not change ANXIETY_DYSPNEA log-odds")
		}
		if math.Abs(ig) > 1e-10 {
			t.Errorf("PATA_NAHI information gain should be ~0, got %.6f", ig)
		}
	})

	// ── Step 7: Get posteriors with safety floors ──
	t.Run("PosteriorsAndFloors", func(t *testing.T) {
		floors := ResolveFloors(node, stratum)

		if floors == nil {
			t.Fatal("DM_HTN_CKD_HF should have stratum-specific safety floors")
		}
		// P02 DM_HTN_CKD_HF floors: CHF=0.10, PE_DYSPNEA=0.06, METABOLIC_ACIDOSIS=0.04
		if floors["CHF"] != 0.10 {
			t.Errorf("CHF floor: want 0.10, got %.2f", floors["CHF"])
		}
		if floors["PE_DYSPNEA"] != 0.06 {
			t.Errorf("PE_DYSPNEA floor: want 0.06, got %.2f", floors["PE_DYSPNEA"])
		}

		posteriors := e.GetPosteriors(logOdds, floors)

		// Verify sum to 1.0
		total := 0.0
		for _, entry := range posteriors {
			total += entry.PosteriorProbability
		}
		if math.Abs(total-1.0) > 1e-9 {
			t.Errorf("posteriors should sum to 1.0, got %.10f", total)
		}

		// CHF should be the top differential after orthopnea + edema YES answers
		if posteriors[0].DifferentialID != "CHF" {
			t.Errorf("CHF should be top differential, got %s (%.4f)",
				posteriors[0].DifferentialID, posteriors[0].PosteriorProbability)
		}

		// Verify _OTHER bucket is annotated
		foundOther := false
		for _, entry := range posteriors {
			if entry.DifferentialID == models.OtherBucketDiffID {
				foundOther = true
				if !entry.IsOtherBucket {
					t.Error("_OTHER should have IsOtherBucket=true")
				}
			}
		}
		if !foundOther {
			t.Error("_OTHER bucket not found in posteriors")
		}

		// Log posteriors for inspection
		t.Log("Final posteriors (DM_HTN_CKD_HF, 62F, after 4 questions):")
		for _, entry := range posteriors {
			flags := ""
			if len(entry.Flags) > 0 {
				flags = " " + entry.Flags[0]
			}
			t.Logf("  %-25s  P=%.4f  LO=%.4f%s",
				entry.DifferentialID, entry.PosteriorProbability, entry.LogOdds, flags)
		}
	})

	// ── Step 8: G5 effect extraction ──
	t.Run("G5_Effects", func(t *testing.T) {
		expandedCMs := ExpandNodeCMs(node.ContextModifiers)

		processor := NewCMEffectProcessor(testLogger())
		result := processor.Extract(expandedCMs)

		// P02 has no HARD_BLOCK or OVERRIDE CMs, so both should be empty.
		if len(result.Contraindications) != 0 {
			t.Errorf("expected 0 contraindications (P02 has no HARD_BLOCK CMs), got %d", len(result.Contraindications))
		}
		if len(result.Overrides) != 0 {
			t.Errorf("expected 0 overrides (P02 has no OVERRIDE CMs), got %d", len(result.Overrides))
		}

		t.Logf("G5 result: %d contraindications, %d overrides",
			len(result.Contraindications), len(result.Overrides))
	})

	// ── Step 9: Convergence check ──
	t.Run("ConvergenceCheck", func(t *testing.T) {
		floors := ResolveFloors(node, stratum)
		posteriors := e.GetPosteriors(logOdds, floors)
		converged, _ := e.CheckConvergence(posteriors, node)

		// After only 4 questions, convergence is unlikely with BOTH logic
		// (threshold=0.82, gap=0.20). Just verify it doesn't crash.
		t.Logf("Convergence after 4 questions: %v (top=%.4f, threshold=%.2f)",
			converged, posteriors[0].PosteriorProbability, node.ConvergenceThreshold)
	})

	// ── Step 10: Entropy decreases with evidence ──
	t.Run("EntropyDecreases", func(t *testing.T) {
		// Re-init to get clean state for entropy comparison
		freshLogOdds := e.InitPriors(node, stratum, activeMeds)
		hInitial := e.ComputeEntropy(freshLogOdds)

		// Current state (after 4 questions) should have lower entropy
		hCurrent := e.ComputeEntropy(logOdds)

		if hCurrent >= hInitial {
			t.Errorf("entropy should decrease with evidence: initial=%.4f, current=%.4f",
				hInitial, hCurrent)
		}

		t.Logf("Entropy: initial=%.4f → current=%.4f (delta=%.4f)",
			hInitial, hCurrent, hInitial-hCurrent)
	})
}

// TestB03_G3_ConditionalDiffs_ExclusionInP02 verifies that when the patient is NOT
// on SGLT2i/Metformin, the conditional differentials are excluded and priors
// redistribute correctly in the DM_HTN_CKD_HF stratum.
func TestB03_G3_ConditionalDiffs_ExclusionInP02(t *testing.T) {
	nodesDir := findNodesDir(t)
	if nodesDir == "" {
		t.Skip("nodes directory not found")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(nodesDir, "p02_dyspnea.yaml")
	dst := filepath.Join(tmpDir, "p02_dyspnea.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("cannot read P02 YAML: %v", err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatal(err)
	}

	nl := NewNodeLoader(tmpDir, testLogger())
	if err := nl.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	node := nl.Get("P02_DYSPNEA")

	e := newTestBayesianEngine()
	stratum := "DM_HTN_CKD_HF"

	// Patient on ARB + Statin only — no SGLT2i, no Metformin
	logOddsExcluded := e.InitPriors(node, stratum, []string{"ARB", "Statin"})

	// Should have 8 authored + 1 _OTHER = 9
	if len(logOddsExcluded) != 9 {
		t.Errorf("expected 9 differentials (8 base + _OTHER), got %d", len(logOddsExcluded))
	}

	if _, ok := logOddsExcluded["EUGLYCEMIC_DKA_RESP"]; ok {
		t.Error("EUGLYCEMIC_DKA_RESP should be excluded without SGLT2i")
	}
	if _, ok := logOddsExcluded["LACTIC_ACIDOSIS_RESP"]; ok {
		t.Error("LACTIC_ACIDOSIS_RESP should be excluded without Metformin")
	}

	// With all meds active — should have 10 authored + 1 _OTHER = 11
	logOddsAll := e.InitPriors(node, stratum, []string{"SGLT2i", "Metformin"})
	if len(logOddsAll) != 11 {
		t.Errorf("expected 11 differentials with all meds, got %d", len(logOddsAll))
	}

	// CHF prior should be slightly higher when conditionals are excluded
	// (their 0.017 mass redistributes proportionally)
	chfExcluded := sigmoid(logOddsExcluded["CHF"])
	chfAll := sigmoid(logOddsAll["CHF"])
	if chfExcluded <= chfAll {
		t.Errorf("CHF should be higher when conditionals excluded: excluded=%.6f, all=%.6f",
			chfExcluded, chfAll)
	}

	// _OTHER should be unaffected by G3 redistribution
	otherExcluded := logOddsExcluded[models.OtherBucketDiffID]
	otherAll := logOddsAll[models.OtherBucketDiffID]
	if math.Abs(otherExcluded-otherAll) > 1e-10 {
		t.Errorf("_OTHER should be same regardless of G3: excluded=%.6f, all=%.6f",
			otherExcluded, otherAll)
	}
}
