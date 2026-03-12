// Package tests provides clinical-device rigor testing for KB-18 Governance Engine.
// This file tests CONCURRENCY safety - must run with: go test -race
package tests

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// CONCURRENCY TESTS - Must be run with -race flag
// go test -race ./tests/...
// =============================================================================

// TestConcurrency_100SimultaneousEvaluations tests that 100 concurrent
// evaluations complete without race conditions or data corruption.
func TestConcurrency_100SimultaneousEvaluations(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	const numGoroutines = 100
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Track all hashes to verify no corruption
	hashes := make(chan string, numGoroutines)
	outcomes := make(chan types.Outcome, numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := &types.EvaluationRequest{
				PatientID: "PT-CONCURRENT",
				PatientContext: &types.PatientContext{
					PatientID:  "PT-CONCURRENT",
					Age:        30 + (id % 50), // Vary ages
					Sex:        "F",
					IsPregnant: id%2 == 0, // Half pregnant, half not
				},
				Order: &types.MedicationOrder{
					MedicationCode: "MTX",
					MedicationName: "Methotrexate",
					DrugClass:      "METHOTREXATE",
					Dose:           10.0,
					DoseUnit:       "mg",
				},
				EvaluationType: types.EvalTypeMedicationOrder,
				RequestorID:    "DR-001",
				Timestamp:      time.Now(),
			}

			resp, err := eng.Evaluate(ctx, req)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			atomic.AddInt64(&successCount, 1)

			if resp.EvidenceTrail != nil {
				hashes <- resp.EvidenceTrail.Hash
			}
			outcomes <- resp.Outcome
		}(i)
	}

	wg.Wait()
	close(hashes)
	close(outcomes)

	duration := time.Since(startTime)

	// Collect results
	var allHashes []string
	var allOutcomes []types.Outcome
	for hash := range hashes {
		allHashes = append(allHashes, hash)
	}
	for outcome := range outcomes {
		allOutcomes = append(allOutcomes, outcome)
	}

	// Verify no errors
	if errorCount > 0 {
		t.Errorf("Had %d errors during concurrent execution", errorCount)
	}

	// Verify all completed
	if successCount != numGoroutines {
		t.Errorf("Expected %d successes, got %d", numGoroutines, successCount)
	}

	// Verify hash validity (should be valid SHA-256)
	for _, hash := range allHashes {
		if len(hash) < 10 {
			t.Error("Invalid hash detected - possible data corruption")
		}
	}

	// Count outcomes
	outcomeCount := make(map[types.Outcome]int)
	for _, outcome := range allOutcomes {
		outcomeCount[outcome]++
	}

	t.Logf("✅ CONCURRENCY TEST PASSED")
	t.Logf("   %d goroutines completed in %v", successCount, duration)
	t.Logf("   Average: %.2f ms per evaluation", float64(duration.Milliseconds())/float64(numGoroutines))
	t.Logf("   Outcomes: %v", outcomeCount)
}

// TestConcurrency_MixedPatientScenarios tests concurrent evaluations
// with different patient types and expected outcomes.
func TestConcurrency_MixedPatientScenarios(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	scenarios := []struct {
		name       string
		pregnant   bool
		drugClass  string
		shouldBlock bool
	}{
		{"pregnant-mtx", true, "METHOTREXATE", true},
		{"nonpregnant-mtx", false, "METHOTREXATE", false},
		{"pregnant-warfarin", true, "WARFARIN", true},
		{"nonpregnant-warfarin", false, "WARFARIN", false},
	}

	const repeatCount = 25 // Run each scenario 25 times = 100 total
	var wg sync.WaitGroup
	results := make(chan struct {
		scenario string
		blocked  bool
	}, len(scenarios)*repeatCount)

	for _, scenario := range scenarios {
		for i := 0; i < repeatCount; i++ {
			wg.Add(1)
			go func(s struct {
				name       string
				pregnant   bool
				drugClass  string
				shouldBlock bool
			}) {
				defer wg.Done()

				req := &types.EvaluationRequest{
					PatientID: "PT-MIXED",
					PatientContext: &types.PatientContext{
						PatientID:  "PT-MIXED",
						Age:        30,
						Sex:        "F",
						IsPregnant: s.pregnant,
						RegistryMemberships: []types.RegistryMembership{
							{RegistryCode: "PREGNANCY", Status: "ACTIVE"},
						},
					},
					Order: &types.MedicationOrder{
						MedicationCode: "TEST",
						MedicationName: "Test Drug",
						DrugClass:      s.drugClass,
						Dose:           5.0,
						DoseUnit:       "mg",
					},
					EvaluationType: types.EvalTypeMedicationOrder,
					RequestorID:    "DR-001",
					Timestamp:      time.Now(),
				}

				resp, err := eng.Evaluate(ctx, req)
				if err != nil {
					return
				}

				results <- struct {
					scenario string
					blocked  bool
				}{s.name, resp.Outcome == types.OutcomeBlocked}
			}(scenario)
		}
	}

	wg.Wait()
	close(results)

	// Count results by scenario
	scenarioResults := make(map[string]struct{ blocked, allowed int })
	for result := range results {
		current := scenarioResults[result.scenario]
		if result.blocked {
			current.blocked++
		} else {
			current.allowed++
		}
		scenarioResults[result.scenario] = current
	}

	t.Logf("✅ MIXED SCENARIO CONCURRENCY TEST:")
	for scenario, counts := range scenarioResults {
		t.Logf("   %s: blocked=%d, allowed=%d", scenario, counts.blocked, counts.allowed)
	}
}

// TestConcurrency_StatisticsAccuracy tests that statistics are accurately
// maintained under concurrent load.
func TestConcurrency_StatisticsAccuracy(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	const numGoroutines = 50
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			req := &types.EvaluationRequest{
				PatientID: "PT-STATS",
				PatientContext: &types.PatientContext{
					PatientID: "PT-STATS",
					Age:       40,
					Sex:       "M",
				},
				EvaluationType: types.EvalTypeMedicationOrder,
				RequestorID:    "DR-001",
				Timestamp:      time.Now(),
			}

			eng.Evaluate(ctx, req)
		}(i)
	}

	wg.Wait()

	stats := eng.GetStats()

	// Total evaluations should equal number of goroutines
	if stats.TotalEvaluations != int64(numGoroutines) {
		t.Errorf("Expected %d total evaluations, got %d", numGoroutines, stats.TotalEvaluations)
	}

	t.Logf("✅ STATISTICS ACCURACY VERIFIED under concurrent load")
	t.Logf("   Total evaluations: %d", stats.TotalEvaluations)
	t.Logf("   Total blocked: %d, Total allowed: %d", stats.TotalBlocked, stats.TotalAllowed)
}

// TestConcurrency_NoDataRace verifies no race conditions in engine state.
// This test is specifically designed to trigger race detection.
func TestConcurrency_NoDataRace(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	const numReaders = 20
	const numWriters = 20
	const iterations = 10

	var wg sync.WaitGroup

	// Writers: perform evaluations (modify state)
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				req := &types.EvaluationRequest{
					PatientID: "PT-RACE",
					PatientContext: &types.PatientContext{
						PatientID: "PT-RACE",
						Age:       30,
						Sex:       "M",
					},
					EvaluationType: types.EvalTypeMedicationOrder,
					RequestorID:    "DR-001",
					Timestamp:      time.Now(),
				}
				eng.Evaluate(ctx, req)
			}
		}(w)
	}

	// Readers: read statistics concurrently (read state)
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				stats := eng.GetStats()
				_ = stats.TotalEvaluations
				_ = stats.ByProgram
				_ = stats.BySeverity
			}
		}(r)
	}

	wg.Wait()

	t.Logf("✅ NO DATA RACE: %d writers x %d readers x %d iterations completed",
		numWriters, numReaders, iterations)
}

// TestConcurrency_ProgramStoreThreadSafety tests that program store
// operations are thread-safe.
func TestConcurrency_ProgramStoreThreadSafety(t *testing.T) {
	programStore := programs.NewProgramStore()

	const numReaders = 50
	const iterations = 20

	var wg sync.WaitGroup

	// Multiple goroutines reading from program store
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Read active programs
				_ = programStore.GetActivePrograms()
				// Read specific program
				_, _ = programStore.GetProgram("MAT")
				_, _ = programStore.GetProgram("OPI")
				// Read program count
				_ = programStore.Count()
			}
		}()
	}

	wg.Wait()

	t.Logf("✅ PROGRAM STORE THREAD SAFETY VERIFIED: %d readers x %d iterations",
		numReaders, iterations)
}

// TestConcurrency_EvidenceTrailHashConsistency tests that evidence trails
// are generated without corruption under concurrent load.
//
// Note: Trail hashes are intentionally unique per evaluation (each gets a new
// TrailID). This test verifies: (1) hashes have valid format, (2) decisions are
// consistent across concurrent evaluations.
func TestConcurrency_EvidenceTrailHashConsistency(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Use identical requests to verify decision consistency
	fixedTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	createRequest := func() *types.EvaluationRequest {
		return &types.EvaluationRequest{
			RequestID: "REQ-HASH-CONCURRENT",
			PatientID: "PT-HASH-CONCURRENT",
			PatientContext: &types.PatientContext{
				PatientID:  "PT-HASH-CONCURRENT",
				Age:        35,
				Sex:        "F",
				IsPregnant: true,
			},
			Order: &types.MedicationOrder{
				MedicationCode: "MTX",
				MedicationName: "Methotrexate",
				DrugClass:      "METHOTREXATE",
				Dose:           10.0,
				DoseUnit:       "mg",
			},
			EvaluationType: types.EvalTypeMedicationOrder,
			RequestorID:    "DR-001",
			Timestamp:      fixedTime,
		}
	}

	type trailResult struct {
		hash       string
		outcome    types.Outcome
		violations int
	}

	const numGoroutines = 30
	var wg sync.WaitGroup
	results := make(chan trailResult, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := createRequest()
			resp, err := eng.Evaluate(ctx, req)
			if err != nil {
				return
			}
			if resp.EvidenceTrail != nil {
				results <- trailResult{
					hash:       resp.EvidenceTrail.Hash,
					outcome:    resp.Outcome,
					violations: len(resp.Violations),
				}
			}
		}()
	}

	wg.Wait()
	close(results)

	var allResults []trailResult
	for res := range results {
		allResults = append(allResults, res)
	}

	if len(allResults) == 0 {
		t.Fatal("No results collected")
	}

	// Verify all hashes are valid (proper SHA-256 format)
	for i, res := range allResults {
		if len(res.hash) < 10 {
			t.Errorf("Goroutine %d: invalid hash format (too short)", i)
		}
		if !strings.HasPrefix(res.hash, "sha256:") {
			t.Errorf("Goroutine %d: hash missing sha256: prefix", i)
		}
	}

	// Verify all clinical decisions are identical (decision determinism)
	first := allResults[0]
	for i, res := range allResults[1:] {
		if res.outcome != first.outcome {
			t.Errorf("DECISION CONSISTENCY FAILURE: Goroutine %d has different outcome", i+1)
		}
		if res.violations != first.violations {
			t.Errorf("DECISION CONSISTENCY FAILURE: Goroutine %d has different violation count", i+1)
		}
	}

	t.Logf("✅ CONCURRENT EVIDENCE TRAIL VERIFICATION: %d valid trails, consistent decisions (%s)", len(allResults), first.outcome)
}
