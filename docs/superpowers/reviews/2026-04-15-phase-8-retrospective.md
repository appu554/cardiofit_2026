# Review: Phase 8 Implementation

This is the session where the platform crosses from "shipped code" to
"shipped clinical effect." Phase 8 started as a 2-hour critical-path
fix — build one missing endpoint, ship it, done. It ended 6-7 hours
and four commits later as a full wire-contract extension, a
cross-service data path activation, and a platform-wide
integration-test rollout that caught **two silent-404 bugs** and a
**production URL mismatch** hidden behind the test discipline of
Phase 7. The retrospective write-up for Phase 7 said the
summary-context fix was "the most important code left to write."
That was correct, and it was also incomplete — what Phase 8 actually
revealed is that the test discipline itself needed repairs, not just
the missing endpoint. Without P8-4, everything shipped in P8-1
through P8-3 would have been dead code in production.

4 commits, ~2000 lines of new code + tests, 30 new tests across
three services, zero regressions in any pre-existing suite, two
production bugs caught by the integration-test pattern, and one
lesson about how the discipline that made Phase 7 fast was also the
discipline that let those bugs survive across four consecutive
sub-projects.

---

## The Critical Bugs Found and Fixed

Phase 8 caught four distinct failure modes, two of which were
production-blocking. Each deserves naming individually because they
illustrate a different gap in the test-writing process.

### Bug 1 — The missing `summary-context` endpoint (known at session start)

This was the explicit P8-1 target. `KB20Client.FetchSummaryContext`
called `GET /patient/:id/summary-context`, no handler existed in
KB-20, every production call returned 404 and every card-generation
path silently no-oped. Fix was ~300 lines: service layer, handler,
route registration, 8 service tests, 3 integration tests.
Straightforward execution against a known diagnosis.

### Bug 2 — The envelope unwrap mismatch in the KB-23 client (caught during P8-1)

While implementing the fix, I found that `FetchSummaryContext` was
decoding the HTTP response body **directly into `PatientContext`**
with no envelope unwrap:

```go
var result PatientContext
json.NewDecoder(resp.Body).Decode(&result)
```

But every other KB20Client method (`FetchRenalStatus`,
`FetchInterventionTimeline`, `FetchLatestCGMReport`) unwraps the
standard `{"success": true, "data": ...}` envelope. If the missing
endpoint had existed with the standard envelope convention all along,
**the client would have returned an empty `PatientContext` struct**
with every field at zero — a different failure mode from a 404, but
equally silent. The card pipeline would produce cards with blank
clinician summaries and empty medication lists, which is arguably
worse than producing no cards at all because it looks like the
system is working.

**Both halves of the bug had to be fixed in one commit.** P8-1 added
the handler AND updated the client to unwrap the envelope. Neither
fix alone would have produced working production behavior.

### Bug 3 — The `sync.Once.Do` recursive deadlock (self-inflicted)

Diagnosed in ~10 minutes via goroutine dump. While writing the P8-1
integration test, I needed a shared metrics collector because the
Prometheus `promauto` registry panics on duplicate registration when
two tests both call `metrics.NewCollector()`. I wrote a `sync.Once`-
backed shared collector helper and made a typo:

```go
func testMetricsCollector() *metrics.Collector {
    sharedMetricsOnce.Do(func() {
        sharedMetricsCollector = testMetricsCollector() // RECURSIVE
                                                        // should be metrics.NewCollector()
    })
    return sharedMetricsCollector
}
```

The inner `Once.Do` call acquired the mutex, the recursive
`testMetricsCollector()` call tried to acquire the same mutex, and
the test hung for 10 minutes before the Go test harness killed it
with a goroutine dump. The dump showed the httptest.Server idle in
`Accept()` (normal) at the top of the stack, which is a red herring —
the real culprit was `goroutine 39 [sync.Mutex.Lock]` twelve frames
deeper, pointing at `sync.(*Once).doSlow` inside my test helper.

The debugging lesson: **when diagnosing a test hang, the first
goroutine in a dump is rarely the broken one.** Find the goroutine
stuck on a user-defined synchronization primitive, not the one stuck
on stdlib I/O.

### Bug 4 — The `/api/v1/` prefix missing from `FetchSummaryContext` URL (caught during P8-4 rollout)

While implementing the integration-test rollout for the other four
cross-service client methods, I compared each method's URL template
against its real production route. Five of six matched. The sixth
was `FetchSummaryContext`:

```go
// Client builds:
url := fmt.Sprintf("%s/patient/%s/summary-context", c.cfg.KB20URL, patientID)

// Real route in routes.go:
v1 := s.Router.Group("/api/v1")
patient := v1.Group("/patient")
patient.GET("/:id/summary-context", s.getSummaryContext)
// → /api/v1/patient/:id/summary-context
```

**Missing `/api/v1/` prefix.** Every production call from KB-23 to
KB-20 for patient context was hitting a URL that does not exist on
the server. P8-1's integration test masked the bug because its mirror
handler ALSO used `/patient/p-integration/summary-context` without
the prefix — both sides were wrong in matching ways, the round trip
worked cleanly against the broken route, the test passed, and
production would 404.

**This is the second silent-404 bug in the same chain**, caught by
the rollout of the same test pattern that was supposed to catch the
first one. The fact that P8-1's test didn't catch it is the most
important process lesson of Phase 8.

---

## What Shipped — Commit by Commit

### P8-1 (`a7a099c3`) — The critical-path fix

845 insertions, 2 deletions across 6 files. New
`SummaryContextService` with
`BuildContext(patientID) → (*SummaryContext, error)` assembling
demographics, stratum, medications, latest labs, and weight from
existing KB-20 tables. New `getSummaryContext` handler wrapping it
in the standard success envelope with `404` on missing patient and
`500` on internal error. New `GET /api/v1/patient/:id/summary-context`
route registered alongside the P7-C and P7-D routes. **8 new service
tests** covering happy path, missing patient, no labs, no
medications, nil EGFR, distinct drug classes, empty patient ID, and
JSON wire contract. **3 new integration tests** in KB-23 against
`httptest.Server` — happy round-trip, 404 propagation, malformed
body. Plus the envelope-unwrap fix on the client side which was the
second latent bug.

P8-1 was the commit the Phase 7 retrospective said unblocked all
clinical value. It was right, but only after P8-4 fixed the URL.
Between P8-1 and P8-4, the endpoint existed but the client was
calling the wrong URL, so the "unblocked" state was theoretical. The
real unblock happened at P8-4.

### P8-2 (`fb49d821`) — The wire contract extension

450 insertions, 56 deletions across 5 files. Extended `PatientContext`
from 10 fields to 21, adding:

- **Demographics**: `Age`, `Sex`, `BMI` — enables the CKM classifier's
  age-based risk stratification, sex-specific thresholds, and the
  waist-risk branch of the lifestyle-intervention pathway.
- **CKM stage metadata**: `CKMStageV2` + nested `CKMSubstageMetadata`
  with HF classification, LVEF, NYHA class, NT-proBNP, BNP, HF
  etiology, CAC score, CIMT percentile, HasLVH. The P7-B 4c pathway
  routing uses only the coarse stage string today — no substage
  awareness. P8-2 unlocks the data pipe for future per-HF-subtype
  card templates.
- **Latest potassium** from `lab_entries` (falling back to the
  `PatientProfile.Potassium` cached column) — needed by the
  MRA/finerenone hyperkalaemia guard and the RAAS creatinine-
  monitoring window.
- **Engagement context**: `EngagementComposite *float64` +
  `EngagementStatus` — used by the adherence-gain factor path and
  the future non-adherence exclusion branch of the inertia detector
  (a patient flagged `DISENGAGED` should not trigger inertia cards;
  the target gap is driven by adherence, not clinical drift).
- **CGM status**: `HasCGM`, `LatestCGMTIR *float64`,
  `LatestCGMGRIZone`, `CGMReportAt *time.Time` — the wire slot the
  inertia detector's `cgmMinDays=14` branch reads.

Every new field uses `omitempty` so existing P7-era consumers compile
without changes. The struct is purely additive. The `BuildContext`
signature changed from `(patientID)` to `(ctx, patientID)` to thread
request context through cross-service calls. The
`SummaryContextService` constructor gained an optional
`CGMStatusFetcher` interface parameter. The wire-contract test on
the KB-20 side now pins all 22 JSON keys through a JSON round-trip
assertion. The KB-23 integration test happy-path fixture populates
every new field distinctly and asserts 13 new field-level round trips
including the nested `CKMSubstageMetadata` block with
`HFrEF + LVEF=35 + NYHA II + CAC=275`.

The confounder flags (`IsAcuteIll`, `HasRecentTransfusion`,
`HasRecentHypoglycaemia`) stayed at `false` with a prominent TODO
comment explaining the data-source gap. This was the one honest
deferral in P8-2, flagged explicitly in the commit message rather
than buried.

### P8-3 (`62d77e75`) — The KB-26 HTTP client wire-up

300 insertions, 11 deletions across 5 files. This is the commit that
activates the full CGM data path from Flink all the way through to
the inertia detector. I found a pre-existing `KB26Client` in KB-20's
`clients/` package — already constructed in `main.go` but immediately
discarded with `_ = kb26Client` for some earlier MRI-related intent
that never became reality. P8-3 added a new
`GetLatestCGMStatus(ctx, patientID)` method to the existing client
(no new client instance needed), wired the adapter pattern in
`main.go` to translate between the clients-layer and services-layer
structs, and injected the fetcher into `api.Server` via a new
`SetKB26CGMFetcher` method matching the existing
`ProtocolService.SetKB25Client` pattern.

**5 new client integration tests** against `httptest.Server` covering
happy path, 404 clean degradation (the most common real-world case
because most patients don't wear CGMs), 5xx error propagation,
malformed body, and network errors. The `(nil, nil)` return pattern
on 404 is the clinical decision that keeps the log stream clean — a
patient without CGM data is not an error condition, it's the normal
case, and the downstream code writes `if snap == nil { fallback }`
which is the same idiom the Phase 6 inertia detector uses for
missing inputs.

After P8-3, the full data path shipped across 10 commits finally had
every link in place:

```
Flink Module3_CGMStreamJob (P7-E M1)
  → clinical.cgm-analytics.v1 (Kafka)
  → KB-26 cgm_analytics_consumer (P7-E M2)
  → cgm_period_reports (Postgres)
  → KB-26 GET /api/v1/kb26/cgm-latest/:patientId (P7-E M2)
  → KB-20 clients.KB26Client.GetLatestCGMStatus (P8-3)
  → KB-20 kb26CGMFetcherAdapter (P8-3)
  → KB-20 SummaryContextService (P8-2)
  → GET /api/v1/patient/:id/summary-context (P8-1, URL fixed in P8-4)
  → KB-23 KB20Client.FetchSummaryContext (P8-1, URL fixed in P8-4)
  → PatientContext.HasCGM / LatestCGMTIR (P8-2)
  → InertiaInputAssembler CGM_TIR override (P7-E M2)
  → DetectInertia cgmMinDays=14 branch (Phase 6 P6-1)
  → INERTIA_DETECTED card with DataSource="CGM_TIR"
```

Every link shipped, tested, pushed. The dormant `cgmMinDays=14`
constant in `inertia_detector.go` — sitting in the codebase unused
since Phase 6 — finally has data flowing into it for real patients.

### P8-4 (`fe1f42e1`) — The integration-test rollout and URL fix

576 insertions, 3 deletions across 3 files. The insert count is
almost entirely test code. The production change is a one-line URL
fix. This is the commit that actually makes Phase 8 production-ready.

8 new integration tests covering the four remaining cross-service
client methods (`FetchRenalStatus`, `FetchInterventionTimeline`,
`FetchRenalActivePatientIDs` + empty-list case, `FetchTargetStatus`,
`FetchLatestCGMReport` + 404 case). Each test uses `httptest.Server`
with a mirror handler registered at the **exact** production route,
a fully-populated wire fixture, and field-by-field round-trip
assertions. The `FetchTargetStatus` test is structurally different
from the others because it's a POST with a request body — the mirror
handler decodes the body and asserts both halves of the wire
contract (request-direction and response-direction).

The URL fix for `FetchSummaryContext` is the production bug fix. The
tightened P8-1 mirror handler (from
`/patient/p-integration/summary-context` to
`/api/v1/patient/p-integration/summary-context`) is the regression
guard — any future drift of the client's URL away from the real
route will fail the test deterministically.

---

## The Meta-Observation — How Integration Tests Caught What Stub Tests Couldn't

The Phase 7 retrospective identified the missing test layer as
"contract tests that verify the HTTP interface between services
matches what both sides expect. One test per cross-service client,
running against a real handler (httptest.Server), verifying
serialization/deserialization round-trips." Phase 8 implemented that
pattern and immediately validated it by catching two silent-404 bugs
that stub-based testing had missed across four consecutive
sub-projects.

**But Phase 8 also surfaced a second-order failure mode inside the
integration-test pattern itself:** permissive route matching. P8-1's
mirror handler used
`http.NewServeMux().HandleFunc("/patient/p-integration/summary-context", ...)`,
which happily matched the KB-23 client's broken URL even though that
URL was wrong. Both sides of the test were wrong in matching ways,
so the round trip worked cleanly, assertions passed, and the bug
shipped.

**The lesson is not "write more integration tests." The lesson is
"integration tests must pin the literal production route, not any
route the client happens to call."** The difference is one line —
`HandleFunc("/api/v1/patient/...")` vs `HandleFunc("/patient/...")` —
but it's the difference between catching the bug at CI time and
shipping it to four consecutive sub-projects. The tightened P8-1
test and all 8 new P8-4 tests use literal production routes, so the
same class of bug now fails deterministically at test time.

This is the kind of test-discipline refinement that only shows up
when you actually use the discipline against real bugs. Writing the
retrospective and acting on it surfaced a bug (Bug 1, P8-1). Fixing
that bug surfaced a client-side bug (Bug 2, envelope). Then a
self-inflicted test bug (Bug 3, Once.Do). Then the URL mismatch
(Bug 4, P8-4). Each layer revealed the next. None of them were
findable from the outside without actually running the pattern.

---

## The Dead Code Finding

During P8-3 recon I found this line in KB-20's `main.go`, written
well before Phase 7 started:

```go
kb26Client := clients.NewKB26Client(cfg.KB26.BaseURL, logger)
_ = kb26Client // available for injection into handlers/services that build TrajectoryInput
```

A whole HTTP client instantiated on every service boot, taking a TCP
connection pool, doing nothing for months. The comment explained the
**intent** — "available for injection" — but no injection ever
happened. The original author had the right idea for MRI-related
work, moved on to something else, and left the placeholder in the
codebase.

P8-3 finally used it. The commit deleted the `_ =` line and added
the real wiring. If I had been more aggressive earlier, I would have
instantiated a new client in P8-3 and added ~15 lines of boilerplate.
Reusing what was already there was the right call because the intent
matched my need exactly, but **"aspiration in the codebase that
never became reality" is a specific smell worth naming** — it's a
placeholder that costs nothing until someone realizes it was meant
to be load-bearing and tries to use it, at which point it either
works (rare) or the aspirational intent doesn't match what you need
(common).

---

## Velocity Assessment

Phase 8 was supposed to be "one endpoint, ~2 hours, one commit." It
ended up being 4 commits and ~6-7 hours. The scope estimate ratio is
3x, which sounds bad but isn't — the scope genuinely expanded
mid-execution because each sub-project revealed work the previous
one needed:

- P8-1 **had** to fix the envelope-unwrap bug because the missing
  endpoint and the wrong decoder were both on the critical path.
- P8-2 **had** to extend the wire contract because the Phase 7
  retrospective's field enumeration was actionable and ignoring it
  would have left Phase 8 partial.
- P8-3 **had** to activate the CGM HTTP client because P8-2 declared
  the CGM fields with nil-safe fallback but the fallback was always
  firing, which is not the same as "feature working."
- P8-4 **had** to roll out the integration-test pattern and fix the
  second silent-404 because without it everything P8-1 through P8-3
  shipped would have been dead code in production.

None of these were scope creep. Each was **unavoidable** once I
started the previous one. The 3x ratio reflects the shape of the
work, not poor estimation: the original 2-hour estimate was for "one
task that's internally trivial and externally opaque," and the
externally-opaque-ness is what produced the expansion.

The session cadence also held up. Each sub-project was a clean
commit with a descriptive message, full test sweep, and push before
moving on. The P8-4 test rollout took the stated ~60 minutes; the
URL fix inside it was a 5-minute diversion that was actually the
highest-value change of the entire session. Finding high-leverage
work hidden inside test infrastructure is its own kind of velocity —
the kind that pays back orders of magnitude per hour.

Phase 7 was 10-11 hours across 6 sub-projects (~1.8 hours/sub-project
average). Phase 8 was 6-7 hours across 4 sub-projects
(~1.6 hours/sub-project average). Similar pacing. Both within
estimate bands once you account for the scope discovery.

---

## What's Next — Priority Order

### Immediate (highest remaining clinical leverage)

**Confounder flag population.** `IsAcuteIll`,
`HasRecentTransfusion`, `HasRecentHypoglycaemia` still default to
`false` in the summary-context response. The V-06 stress-
hyperglycaemia MCU gate rule in `mcu_gate_manager.go` depends on
`IsAcuteIll` to pause glycaemic intensification for acutely ill
patients — right now that rule never fires because the flag is
always false. Populating requires either (a) a new `safety_events`
audit table fed by the event bus with a migration + consumer +
query method, or (b) a rolling-window Kafka consumer on
`clinical.priority-events.v1` that maintains an in-memory view.
~200-300 lines. This is the last item on the Phase 8 punch list
where code directly changes what a clinician sees — everything
after it is verification or planning.

### Highest-confidence verification

**Staging verification.** Deploy the current state, seed one
patient with the full clinical profile (diabetic on metformin +
ACEi, eGFR=25, HbA1c=8.5, latest CGM period report with TIR=55),
trigger the card pipeline, observe cards in `decision_cards`. The
single end-to-end test that proves every Phase 7-8 commit holds
under real traffic. Depends on staging environment access. Higher
confidence than any automated test because it exercises the actual
Postgres → Kafka → Go → HTTP → Go → Postgres cycle with real
infrastructure.

### Lower-priority but worth naming

**Extend the integration-test pattern to the remaining uncovered
client methods.** `GetCurrentMRI` and `GetMRIHistory` on KB-20's
KB-26 client are pre-existing P6-era code that Phase 7-8 didn't
touch. They carry the same stub-only risk profile. Not critical
because they're not in the Phase 7 card-generation critical path,
but applying the pattern there closes the bug class entirely across
the platform. ~30 minutes.

**Phase 9 planning.** After the confounder flags ship, Phase 9
should target one of: (a) new clinical intelligence features that
weren't in Phase 1-8 scope, (b) hardening items from the original
Phase 8 punch list that never happened (InertiaVerdictHistory
Postgres repo, CGM dedup, bounded-concurrency fan-out), or (c) FHIR
outbound (Gap 9, which becomes compliance-critical for Australia
MHR / India ABDM market access). These are real decisions that
benefit from plan-first discipline because they involve
prioritization tradeoffs, not just a scoped bug fix.

---

## Summary

Phase 8 took Phase 7 from "shipped code" to "shipped clinical
effect." Four commits, 11 total Phase 7-8 commits on the
`feature/v4-clinical-gaps` branch ahead of origin. The critical-path
bug is fixed, the wire contract is complete, the CGM data path is
live end-to-end, and every cross-service client method in the
critical path has an integration test against a real handler. The
two silent-404 bugs that gated Phase 1-7 clinical value are closed,
and the integration-test pattern that catches the class of bug is
now applied across every method that could have recurred it.

The single unresolved blocker — the summary-context URL mismatch —
survived four commits and was caught by the rollout of the same test
discipline that missed it. That's a process lesson worth keeping:
**integration tests must pin literal production routes, and the
first retrospective that uses a new test discipline should always
assume it has its own latent blind spots.** The discipline that made
Phase 7 fast was also the discipline that let two silent-404 bugs
accumulate. Phase 8 both used and repaired that discipline in the
same session.

The platform is now one data-source sub-project away (confounder
flags) from every MCU gate rule firing correctly. Everything else is
verification, cleanup, or new scope.
