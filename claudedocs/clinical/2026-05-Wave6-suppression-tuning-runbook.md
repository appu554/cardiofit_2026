# Wave 6 — Suppression-class 5 + 6 tuning runbook

**Audience:** clinical lead + engineering lead pairing tuning the
Vaidshala / CardioFit Layer 3 rule library against pilot override-rate
data.

**Cadence:** weekly during pilot rollout; monthly once the library-wide
override rate is stable below the 5% target.

**Authoritative references:**
- Layer 3 v2 doc Part 2.1 (Suppression class 5 — substrate-state) and
  Part 2.2 (Suppression class 6 — authorisation-context)
- This repo: `shared/cql-libraries/helpers/SuppressionHelpers.cql`
- This repo: `kb-30-authorisation-evaluator/internal/analytics/override_tracker.go`
- This repo: `shared/cql-toolchain/rule_retirement_workflow.py`

## Why Class 5 + 6 in particular

Class 1 (eligibility), Class 2 (recently_actioned), Class 3 (cohort),
Class 4 (workflow) are static-shape suppressions whose tuning is
relatively mechanical. Class 5 + 6 are the substrate- and authorisation-
context suppressions that depend on the live state machines — they are
both the most powerful (they can convert a noisy alert into useful
exhaust without losing the underlying signal) and the most fragile
(misconfiguration creates silent under-fire). This runbook gives the
clinical-lead + engineering-lead pair a structured weekly process.

## Inputs

1. **Library-wide override rate** from kb-30 analytics tracker.
   Target: < 5%. Read via:
   ```go
   tracker.LibraryWideOverrideRate(30) // 30-day rolling
   ```
2. **Per-rule override-rate report** for the trailing 30 days.
3. **Suppression-stat per rule** (which suppression class fired most
   often per rule) — V2 instrumentation; for now, read-only via
   `EvidenceTrace` audit query Q4 (chain) on a sample of fires.

## Decision matrix per noisy rule

| Override rate | Suppression-stat tells us | Action |
|---|---|---|
| < 30% | suppressions firing as designed | KEEP — no action |
| 30-50% | Class 5 (substrate-state) under-firing | TIGHTEN Class 5 predicate |
| 30-50% | Class 6 (authorisation-context) under-firing | TIGHTEN Class 6 predicate |
| 50-70% | Class 5 + 6 both contributing | RE-AUTHOR rule body, not just suppressions |
| > 70% with N >= 5 | Tracker has FlagRetire=true | ROUTE to retirement workflow |

## Tuning Class 5 — substrate-state

Class 5 reads the live substrate state via ClinicalStateHelpers
primitives (`HasActiveConcernType`, `CurrentCareIntensity`, `IsTrending`,
`DeltaFromBaseline`, etc.). The most common failure modes:

1. **Stale concern type:** the substrate has `BPSD_REVIEW_DONE_12W` but
   the suppression checks `BPSD_REVIEW_DONE` (without the duration
   suffix). Fix: align the concern type literal.
2. **Care intensity gate too narrow:** the rule fires only on
   `active_treatment` but the suppression also checks `palliative` →
   miss. Fix: align the suppression to the same care intensity set the
   rule body uses.
3. **Trajectory direction mismatch:** the rule body uses
   `IsTrending(...,'down')` but the suppression checks `'up'`.

## Tuning Class 6 — authorisation-context

Class 6 reads from the authorisation state machine (kb-30) and the
ScopeRule store (kb-31). The most common failure modes:

1. **Stale ScopeRule reference** — when kb-31 publishes a new ScopeRule
   version, CompatibilityChecker marks the rule STALE, but if the
   suppression copy of the scope_rule_ref id is hard-coded, it can
   silently drift. Fix: route changes through the kb-31 publication
   pipeline, never edit `scope_rule_refs` inline.
2. **Fallback path missing** — Class 6 should suppress when the
   authorisation surface routes the action to a fallback role. If the
   rule does not check `HasAvailablePrescriberForClass`, it will
   double-fire. Fix: add the fallback check.
3. **Grace period not honoured** — when a ScopeRule is in its grace
   period (e.g. Victorian PCW 1 Jul - 29 Sep 2026), the Class 6
   suppression must surface a soft fire only. Fix: the rule body must
   read `InGracePeriod` and downgrade the indicator.

## Weekly cadence

```
Monday    — engineering lead pulls override-rate report + 24h roll-up.
Tuesday   — clinical lead reviews flagged rules; marks clinical
            overrides where appropriate (must record rationale per
            rule_retirement_workflow.ClinicalOverride).
Wednesday — pair authoring session: tune Class 5 + 6 predicates per
            decision matrix above.
Thursday  — toolchain run (run_two_gate + CompatibilityChecker +
            governance promoter); promote tuned rules.
Friday    — release window; deploy via the standard governance signing
            chain. Record outcomes for next week's review.
```

## Volume budget

Operational target per Layer 3 v2 doc Part 2.4: **<= 5 actionable alerts
per resident per day.** When the budget is exceeded for a cohort:

1. Run the Class 5/6 decision matrix for the top 5 rules contributing
   to the over-budget cohort.
2. If suppressions can't bring the budget back, route to the rule
   retirement workflow (`shared/cql-toolchain/rule_retirement_workflow.py`)
   for clinical lead review.
3. Track the action via `kb-30 analytics` tracker; expect a measurable
   drop in override rate within 7 days of suppression tuning.

## Audit chain

Every suppression edit must:
- Update the rule's `content_sha` in the YAML spec.
- Re-run the toolchain (Stage 1 + two-gate + CompatibilityChecker).
- Re-sign the rule via the GovernancePromoter.
- Record the tuning rationale in the rule-retirement workflow's
  ClinicalOverride list (even if the action is "KEEP" rather than
  "RETIRE") so the audit chain reflects every tuning decision.

## Failure modes to escalate

- A rule's override rate drops below 5% but its suppression coverage
  rose above 60% — likely silent under-fire. Escalate to a clinical
  lead audit before promoting further.
- A ScopeRule change marks > 20 rules STALE in a single Event D fan-out.
  Pause promotion; do an architectural review before re-validating in
  bulk (suggests a missing helper or a misshapen scope_rule_refs[]).
- The library-wide override rate trend reverses (from below 5% back
  toward 10%). Treat as a regression — pull the previous week's
  changes and review.
