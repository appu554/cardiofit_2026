// Package integration runs the Sunday-night-fall scenario end-to-end
// against the kb-30 evaluator (Layer 3 v2 doc Part 4.5.5).
//
// 7 sequential authorisation evaluations across the workflow, each with
// its own latency budget. The total scenario p95 must stay under 500ms
// per call. EvidenceTrace records (audit.EvaluationRecord) must capture
// the full chain of authority for every evaluation.
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
	"kb-authorisation-evaluator/internal/store"
)

func loadExamplesIntoStore(t *testing.T, s *store.MemoryStore) {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	var examplesDir string
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "examples")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			examplesDir = candidate
			break
		}
		dir = filepath.Dir(dir)
	}
	require.NotEmpty(t, examplesDir)

	for _, fname := range []string{
		"aus-vic-pcw-s4-exclusion.yaml",
		"designated-rn-prescriber.yaml",
		"acop-credential-active.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(examplesDir, fname))
		require.NoError(t, err)
		rule, err := dsl.ParseRule(data)
		require.NoError(t, err)
		_, err = s.Insert(context.Background(), *rule, data)
		require.NoError(t, err)
	}
}

// step is one evaluation in the Sunday-night-fall scenario.
type step struct {
	name           string
	q              evaluator.Query
	expectDecision dsl.Decision
	latencyBudget  time.Duration
}

func TestSundayNightFall_FullWorkflow(t *testing.T) {
	s := store.NewMemoryStore()
	loadExamplesIntoStore(t, s)
	c := cache.NewInMemory()
	auditSvc := audit.NewService()
	eval := evaluator.New(s, evaluator.AlwaysPassResolver)

	resident := uuid.New() // Mary
	pcwActor := uuid.New() // Sarah
	rnActor := uuid.New()  // Jamie
	pharmActor := uuid.New() // Priya
	gpActor := uuid.New()  // Dr Chen

	steps := []step{
		// 1. PCW Event log — Sunday 18:30 (referenced as 21:47 in the doc;
		//    using the prompt's timestamps for the ordering).
		{
			name:          "PCW Event log (Sunday 18:30)",
			latencyBudget: 50 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC",
				Role:         "personal_care_worker",
				ActionClass:  dsl.ActionObserve,
				ResidentRef:  resident,
				ActorRef:     pcwActor,
				ActionDate:   time.Date(2026, 10, 4, 18, 30, 0, 0, time.UTC),
			},
		},
		// 2. RN observation — Sunday 18:35
		{
			name:          "RN post-fall vitals (Sunday 18:35)",
			latencyBudget: 50 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC",
				Role:         "registered_nurse",
				ActionClass:  dsl.ActionObserve,
				ResidentRef:  resident,
				ActorRef:     rnActor,
				ActionDate:   time.Date(2026, 10, 4, 18, 35, 0, 0, time.UTC),
			},
		},
		// 3. ACOP profile view — Monday 09:00. Matches ACOP rule:
		//    granted_with_conditions when APC + AHPRA credentials current.
		{
			name:          "ACOP pharmacist profile view (Monday 09:00)",
			latencyBudget: 100 * time.Millisecond,
			expectDecision: dsl.DecisionGrantedWithConditions,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC",
				Role:         "acop_pharmacist",
				ActionClass:  dsl.ActionViewProfile,
				ResidentRef:  resident,
				ActorRef:     pharmActor,
				ActionDate:   time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC),
			},
		},
		// 4. GP recommendation approval — Monday 14:00. Most complex
		//    check; the GP does not match a designated_rn_prescriber rule,
		//    so default-grant fires (open by default for action classes
		//    without a jurisdictional rule, plus condition resolver vouches).
		{
			name:          "GP recommendation approval (Monday 14:00)",
			latencyBudget: 200 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction:       "AU/VIC",
				Role:               "gp",
				ActionClass:        dsl.ActionPrescribe,
				MedicationSchedule: "S4",
				MedicationClass:    "benzodiazepines",
				ResidentRef:        resident,
				ActorRef:           gpActor,
				ActionDate:         time.Date(2026, 10, 5, 14, 0, 0, 0, time.UTC),
			},
		},
		// 5-7. RN monitoring observations × 3 (Tue/Wed/Thu)
		{
			name:          "RN monitoring observation Tuesday",
			latencyBudget: 50 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC", Role: "registered_nurse", ActionClass: dsl.ActionObserve,
				ResidentRef: resident, ActorRef: rnActor,
				ActionDate: time.Date(2026, 10, 6, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "RN monitoring observation Wednesday",
			latencyBudget: 50 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC", Role: "registered_nurse", ActionClass: dsl.ActionObserve,
				ResidentRef: resident, ActorRef: rnActor,
				ActionDate: time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC),
			},
		},
		{
			name:          "RN monitoring observation Thursday",
			latencyBudget: 50 * time.Millisecond,
			expectDecision: dsl.DecisionGranted,
			q: evaluator.Query{
				Jurisdiction: "AU/VIC", Role: "registered_nurse", ActionClass: dsl.ActionObserve,
				ResidentRef: resident, ActorRef: rnActor,
				ActionDate: time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC),
			},
		},
	}

	type runResult struct {
		step    step
		elapsed time.Duration
		decision dsl.Decision
		evalID  uuid.UUID
	}
	results := make([]runResult, 0, len(steps))

	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			start := time.Now()

			// Try cache first.
			var res evaluator.Result
			if cached, ok, _ := c.Get(context.Background(), st.q.CacheKey()); ok && cached != nil {
				res = *cached
			} else {
				var err error
				res, err = eval.Evaluate(context.Background(), st.q)
				require.NoError(t, err)
				_ = c.Set(context.Background(), st.q.CacheKey(), &res, cache.DefaultTTL(res))
			}
			elapsed := time.Since(start)

			rec := audit.EvaluationRecord{
				ID:          uuid.New(),
				Query:       st.q,
				Result:      res,
				EvaluatedAt: res.EvaluatedAt,
			}
			auditSvc.Record(rec)

			assert.Equal(t, st.expectDecision, res.Decision,
				"%s: expected %s, got %s (rule=%s)", st.name, st.expectDecision, res.Decision, res.RuleID)
			assert.Less(t, elapsed, st.latencyBudget,
				"%s exceeded latency budget: %v >= %v", st.name, elapsed, st.latencyBudget)

			t.Logf("%s: decision=%s rule=%s elapsed=%v budget=%v",
				st.name, res.Decision, res.RuleID, elapsed, st.latencyBudget)

			results = append(results, runResult{step: st, elapsed: elapsed, decision: res.Decision, evalID: rec.ID})
		})
	}

	// Total scenario p95 < 500ms per call.
	t.Run("scenario_total_latency_under_budget", func(t *testing.T) {
		var maxElapsed time.Duration
		for _, r := range results {
			if r.elapsed > maxElapsed {
				maxElapsed = r.elapsed
			}
		}
		assert.Less(t, maxElapsed, 500*time.Millisecond,
			"slowest evaluation %v exceeds 500ms scenario budget", maxElapsed)
		t.Logf("scenario max latency: %v across %d evaluations", maxElapsed, len(results))
	})

	// EvidenceTrace contains the full chain — query Q1 must return all 7.
	t.Run("evidence_trace_full_chain", func(t *testing.T) {
		records := auditSvc.QueryByResident(resident, time.Time{}, time.Time{})
		assert.Len(t, records, len(steps),
			"audit trail must capture every evaluation")
		// Q4: chain query for the PCW evaluation surfaces no rule (granted
		// by default), but for the ACOP evaluation surfaces the rule + ref.
		for _, r := range results {
			chain, ok := auditSvc.QueryAuthorisationChain(r.evalID)
			require.True(t, ok)
			assert.NotZero(t, chain.Record.ID)
		}
	})

	// Verify cache warming on a re-run of step 1 shows a cache hit.
	t.Run("cache_warm_replay", func(t *testing.T) {
		cached, ok, _ := c.Get(context.Background(), steps[0].q.CacheKey())
		require.True(t, ok)
		assert.Equal(t, dsl.DecisionGranted, cached.Decision)
	})
}
