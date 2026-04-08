# Clinical Pipeline E2E Test Harness & M13 Debugger

## Architecture

```
┌─────────────────┐     ┌──────────────┐     ┌──────────────────┐
│  E2E Harness    │────▶│  Kafka       │────▶│  Production      │
│  (generates     │     │  (Testcontainer)   │  Flink Jobs      │
│   test data)    │     └──────┬───────┘     │  (UNMODIFIED)    │
└────────┬────────┘            │             └──────┬───────────┘
         │                     │                    │
         │    ┌────────────────▼────────────────────▼──┐
         └───▶│  Output Collectors + Assertions        │
              │  + M13 Velocity Debugger               │
              └────────────────────────────────────────┘
```

**Zero production code changes.** The harness tests modules 7-13 by:
- Spinning up real Kafka via Testcontainers
- Injecting test data into input topics
- Collecting outputs from output topics
- Running assertions + diagnostic analysis

## Files

| File | Purpose |
|------|---------|
| `ClinicalPipelineE2EHarness.java` | Main test class — M7 through M13 |
| `build.gradle` | Dependencies and test configuration |
| `m13_velocity_debugger.py` | Standalone diagnostic for the CARDIOVASCULAR=0.0 bug |

## Solving the Timer/Window Problem (Without Code Changes)

### M9 (Engagement) — Daily timer at 23:59 UTC
- **Fast CI:** Tagged `@Tag("slow")`, skipped by default
- **Full E2E:** Run `./gradlew testSlow` near 23:55 UTC
- The harness calculates exact ms until 23:59 and waits

### M10 (Meal Response) — 3h05m session window
- **Strategy:** Inject meal event, then `Thread.sleep(11_100_000)` (3h05m + buffer)
- Processing-time session window closes → output fires
- Tagged `@Tag("slow")`

### M11 (Activity Response) — Same session window pattern
- Same strategy as M10

### Running

```bash
# Fast tests only (M7, M8, M13, lineage) — ~2 minutes
./gradlew test

# Full E2E including timer-dependent modules — ~4 hours
./gradlew testE2EFull

# Run M13 debugger against existing test output
python3 m13_velocity_debugger.py e2e-14day-all-modules-io.json
```

## M13 CARDIOVASCULAR Velocity Bug — Diagnosis

**Root Cause (92% confidence):**

M13's CKM velocity calculator computes CARDIOVASCULAR velocity from
`bp_control_status` **transitions only**. Since the status was
`STAGE_2_UNCONTROLLED` for all 54 M7 outputs (no transitions),
velocity = 0.0.

The calculator ignores these M7 signals that DID change:
- `variability_classification_7d`: ELEVATED → HIGH
- `sbp_7d_avg`: 153 → 167 mmHg (+14 over 14 days)
- `crisis_flag`: multiple activations (SBP ≥ 180)
- `surge_classification`: escalated to ELEVATED

**Same bug affects METABOLIC=0.0** — FBG of 165/185 mg/dL not mapped.

**Contributing factor (70% confidence):** Race condition — M13 fires
`DATA_ABSENCE_CRITICAL` before M7 data is fully committed to state
(burst injection artifact). `data_completeness=0.667` at CKM time.

### Recommended Fix

Update `CKMVelocityCalculator.computeCardiovascularVelocity()`:

| M7 Signal | Weight | Threshold |
|-----------|--------|-----------|
| `variability_classification` transition | 0.30 | any → HIGH |
| `sbp_7d_avg` slope (mmHg/week) | 0.25 | > 5 mmHg/wk |
| `crisis_flag` frequency | 0.25 | ≥ 1 in 7d |
| `surge_classification` escalation | 0.10 | → ELEVATED |
| `bp_control_status` transition | 0.10 | any change |
