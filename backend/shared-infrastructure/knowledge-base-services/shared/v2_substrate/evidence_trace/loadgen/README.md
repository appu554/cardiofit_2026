# EvidenceTrace load-test harness (Wave 5.4)

Synthetic graph generator and benchmark harness for the EvidenceTrace
substrate. Production execution and SLO lock-in are deferred to V1; this
package ships the code so an operator can run it on demand.

## What it produces

The default profile (`DefaultProfile()`) generates ~180,000 nodes and
~500,000 edges:

- 200 residents
- 6 months of activity per resident (180 days)
- 5 nodes/day per resident (3 observation Monitoring + 1 Recommendation
  + 1 outcome ClinicalState)
- 4 edges/day per resident (3 derived_from + 1 led_to)

Reproducible via `Profile.Seed`.

## Invocation against an in-memory sink

```go
import (
    "context"
    "github.com/cardiofit/shared/v2_substrate/evidence_trace/loadgen"
)

sink := newYourMemSink()
stats, err := loadgen.Synthesize(context.Background(), sink, loadgen.DefaultProfile())
```

## Invocation against kb-20 PostgreSQL (V1 — deferred)

To run against a real kb-20 instance:

1. Provision a clean PostgreSQL with migration 022 applied.
2. Start kb-20 with `KB20_DATABASE_URL` pointing at it.
3. Adapt the kb-20 `V2SubstrateStore` as a `NodeSink` (it already
   implements `UpsertEvidenceTraceNode` + `InsertEvidenceTraceEdge`).
4. Run `loadgen.Synthesize(ctx, store, loadgen.DefaultProfile())` from a
   short-lived CLI binary.
5. Run the Wave 5.4 benchmark harness (`bench_test.go`) against the same
   store with `-bench=. -benchtime=10s`.

The plan task explicitly defers the production benchmark + index-tune
exercise to V1; the `bench_test.go` shipped here exercises the in-process
BFS traversal on the synthetic graph as a regression-detection floor.

## Acceptance targets (per Wave 5.4 plan)

| Operation | Target p95 |
|-----------|-----------|
| Forward traversal depth=5 | <200ms |
| Backward traversal depth=5 | <200ms |
| Materialised view incremental refresh | <60s |
| Materialised view full refresh | <10min |

These targets are deferred to V1 verification against the production
PostgreSQL deployment. The current bench provides an early-warning floor
on traversal regressions in the pure-Go BFS implementation.

## Files

- `synthesize.go` — graph generator
- `synthesize_test.go` — unit tests + small profile validation
- `bench_test.go` — Go benchmarks for forward+backward depth=5 traversal
  on the small profile
