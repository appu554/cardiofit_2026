# RIR Semantics — Implementation vs "Touched" Rate

**RIR** (Recommendation Implementation Rate) per v3 §11 line 588 is the
v3 Layer-C operational North Star metric.

## What "actioned" means

A recommendation counts as **actioned** ONLY when its current state is
one of:

- `implemented`
- `monitoring-active`
- `outcome-recorded`

…AND `decided_at` is populated within the rolling window (28 days by
convention).

It does NOT count:

- `closed` — even if the closure followed implementation, the
  `recommendations` table doesn't preserve enough state history to
  distinguish closed-after-implemented from closed-via-decided-no-action.
  The full state history is in EvidenceTrace. RIR over the substrate
  table is conservative: `closed` = uncounted.
- `decided` — represents a prescriber decision but not implementation.
- `deferred`, `submitted`, `viewed`, `drafted`, `detected` — not actioned
  by definition.

## Why this differs from the matview

Migration `023_recommendation_lifecycle.sql` defines a materialised view
`recommendation_rir_28d` whose actioned filter is looser:

```sql
state IN ('decided','implemented','monitoring-active',
          'outcome-recorded','closed')
```

The matview was the original Plan 0.1 design but is INCORRECT per the
Task 3 review's concern: a `decided-no-action → closed` path inflates
the rate. The Go `ComputeRIR` function in `rir.go` is the AUTHORITATIVE
computation; the matview is retained for backward compatibility but
should be tightened in a follow-up migration to:

```sql
state IN ('implemented','monitoring-active','outcome-recorded')
```

Until that follow-up, callers who use `recommendation_rir_28d` directly
will see slightly inflated rates. New consumers MUST use
`ComputeRIR(ctx, db, authorID, 28*24*time.Hour)` instead.

## Why "implementation" not "touched"

RIR is conventionally an implementation rate (Ramsey 2025 baseline,
PHARMA-Care framework). A "touched" rate (any prescriber action including
no-action close) measures different behaviour and is not what stakeholders
ask about when discussing RIR. Conflating the two would silently inflate
the headline North Star metric.

## Caveat: closed-after-implemented

The substrate table compresses the lifecycle: a recommendation that
passed through `implemented` and then onward to `closed` will only show
its terminal `closed` state. This means `ComputeRIR` slightly
under-counts true implementations. The correction for this is to
consult EvidenceTrace, which preserves the full transition history.
For the operational RIR dashboard, the substrate-level approximation is
acceptable; for audit-grade attribution, EvidenceTrace is required.
