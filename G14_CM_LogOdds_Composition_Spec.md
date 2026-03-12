# G14: CM Composition in Log-Odds Space
## Implementation Specification — CMApplicator

**File**: `kb-22/.../services/cm_applicator.go`
**Go Change**: G14
**Week**: 1 (Day 1 — required before P00 Dizziness YAML can load meaningfully)
**Depends on**: None (self-contained change to CMApplicator)
**Required by**: P00, P01 V2, P02 V1, ALL nodes with polypharmacy CMs
**Scope**: Narrow — CMApplicator only. No BayesianEngine changes. No YAML schema changes.

---

## 1. Problem Statement

`CMApplicator.Apply()` currently adds author-declared `delta_p` values directly to
differential posteriors in probability space:

```go
// CURRENT BROKEN BEHAVIOR (pseudocode)
posterior[dx] += cm.DeltaP   // e.g., 0.25 + 0.20 = 0.45
```

This is mathematically invalid when two or more CMs target the same differential —
which is the **default case** in the DM+HTN polypharmacy cohort, not an edge case.

### Why It Breaks

Probability space addition is not additive. Two independent OR=2.33 evidence sources
on the same hypothesis do not sum to a 0.40 shift — they combine multiplicatively in
odds space. The correct operation is:

```
logodds(final) = logodds(base) + Σ delta_logodds(each CM)
```

### Concrete failure: P00 Dizziness, 3 CMs on OH

A patient aged 68, on ARB + metoprolol + HCTZ, who started a new antihypertensive
14 days ago fires CM01 (ARB → OH +0.20), CM07 (age≥65 + 2 antihypertensives → OH
+0.20), and CM08 (recent med change → OH +0.15). With a base prior of 0.24:

| Method | OH posterior | Valid? |
|---|---|---|
| Naive probability sum | 0.24 + 0.20 + 0.20 + 0.15 = **0.79** | No — inflated |
| Log-odds composition | sigmoid(logit(0.24) + 0.847 + 0.847 + 0.619) = **0.76** | Yes |

The difference widens dramatically with 4+ CMs. With naive addition on 4 CMs,
OH can exceed 1.00 — an impossible probability, breaking the entire posterior
distribution.

---

## 2. The Math

### Conversion formula

Given an author-declared `delta_p` (the probability shift at a 50% baseline prior):

```
delta_logodds = logit(0.50 + delta_p) − logit(0.50)
             = ln((0.50 + delta_p) / (0.50 − delta_p))
```

For negative deltas (DECREASE_PRIOR):

```
delta_logodds = logit(0.50 − |delta_p|) − logit(0.50)
             = ln((0.50 − |delta_p|) / (0.50 + |delta_p|))     [negative result]
```

### Application

All CMs targeting the same differential are summed in log-odds space:

```
logodds_adjusted = logodds_base + Σ delta_logodds_i
p_adjusted = sigmoid(logodds_adjusted) = 1 / (1 + exp(−logodds_adjusted))
```

### Reference conversion table

| delta_p (YAML) | delta_logodds | Equivalent OR |
|---|---|---|
| +0.05 | +0.2007 | 1.22 |
| +0.10 | +0.4055 | 1.50 |
| +0.15 | +0.6190 | 1.86 |
| +0.20 | +0.8473 | 2.33 |
| +0.25 | +1.0986 | 3.00 |
| +0.30 | +1.3863 | 4.00 |
| +0.35 | +1.7346 | 5.67 |
| +0.40 | +2.1972 | 9.00 |
| −0.05 | −0.2007 | 0.82 |
| −0.10 | −0.4055 | 0.67 |
| −0.15 | −0.6190 | 0.54 |
| −0.20 | −0.8473 | 0.43 |
| −0.25 | −1.0986 | 0.33 |

### Why delta_p at baseline 0.50?

Authors express CM magnitude as "this drug increases the probability by 0.20."
That statement is most naturally interpreted as a shift at a neutral 50% prior —
the OR corresponding to a 50%→70% shift is 2.33. Using the 0.50 baseline as the
reference point converts any author-declared delta_p to a consistent OR that can
then be applied to any base prior via log-odds arithmetic.

**This means YAML authors never change their delta_p values.** The conversion
is internal to the engine. CM01 ARB delta_p: 0.20 remains 0.20 in every YAML
forever — G14 just changes how the engine interprets it.

---

## 3. Scope Boundaries

**G14 touches**: `CMApplicator.Apply()` — the single method where probability-space
arithmetic currently happens.

**G14 does NOT touch**:
- `BayesianEngine.Update()` — LR updates already operate in log-odds space (correct)
- `BayesianEngine.InitPriors()` — priors stored as probabilities, converted to
  log-odds at init time (already correct per existing architecture)
- `BayesianEngine.GetPosteriors()` — converts log-odds → probability for output
  (already correct)
- `HARD_BLOCK` and `OVERRIDE` CM effect types — these are G5 features that bypass
  Bayesian arithmetic entirely. G14 must explicitly skip them.
- Sex modifiers — processed by `ApplySexModifiers()` via G2, separate call path,
  same log-odds math already specified in G2 spec

**Call sequence** (unchanged by G14):

```
InitPriors() → ApplySexModifiers() [G2] → CMApplicator.Apply() [G14 changes here]
→ QuestionOrchestrator → BayesianEngine.Update() [per answer] → GetPosteriors()
```

---

## 4. Detailed Go Change

### 4.1 New helper functions

Add to `cm_applicator.go` (or a `math_util.go` in the same package):

```go
import "math"

// deltaLogOdds converts an author-declared probability-space delta (at a 0.50 baseline)
// to a log-odds shift. This is the core G14 conversion.
//
// The 0.50 baseline means: "if the prior were 50%, this CM shifts it by delta_p."
// That defines an odds ratio of (0.50+delta_p)/(0.50-delta_p), which is applied
// multiplicatively (additively in log-odds space) to any base prior.
//
// Valid range: delta_p in (-0.49, +0.49). Values at or beyond ±0.50 are undefined
// (logit approaches ±infinity). YAML validation must enforce |delta_p| < 0.49.
func deltaLogOdds(deltaP float64) float64 {
    // Guard: clamp to safe range. Should not happen if YAML validation is correct.
    if deltaP >= 0.49 {
        deltaP = 0.49
    }
    if deltaP <= -0.49 {
        deltaP = -0.49
    }
    return math.Log((0.50+deltaP)/(0.50-deltaP))
}

// logit converts a probability to log-odds. p must be in (0, 1).
// If p is exactly 0 or 1 (should never happen with valid priors), returns ±MaxFloat64.
func logit(p float64) float64 {
    if p <= 0 {
        return -math.MaxFloat64 / 2
    }
    if p >= 1 {
        return math.MaxFloat64 / 2
    }
    return math.Log(p / (1 - p))
}

// sigmoid converts log-odds to probability.
func sigmoid(logOdds float64) float64 {
    return 1.0 / (1.0 + math.Exp(-logOdds))
}
```

### 4.2 CMApplicator.Apply() rewrite

Current `Apply()` signature (preserved — no interface change):

```go
func (a *CMApplicator) Apply(
    cms []ContextModifier,
    posteriors map[string]float64,
    patientContext PatientContext,
) (map[string]float64, []FiredCM, error)
```

**Replace the inner accumulation loop** with log-odds composition.

The change is localized to the section that currently does:

```go
// BEFORE (broken)
for _, cm := range activeCMs {
    for _, target := range cm.Targets {
        posteriors[target.DifferentialID] += target.DeltaP
    }
}
```

**New implementation**:

```go
// AFTER (G14): accumulate delta_logodds per differential, apply once

// Step 1: Convert current posteriors to log-odds working space.
// Posteriors are maintained as probabilities in the map; we work in log-odds
// locally and write back probabilities.
type cmAccumulator struct {
    baseLogOdds      float64
    totalDeltaLogOdds float64
    cmCount           int
}

accumulators := make(map[string]*cmAccumulator)
for dxID, p := range posteriors {
    accumulators[dxID] = &cmAccumulator{
        baseLogOdds: logit(p),
    }
}

// Step 2: For each active CM, compute delta_logodds and accumulate.
// IMPORTANT: Skip HARD_BLOCK and OVERRIDE — handled by G5, not G14.
for _, cm := range activeCMs {
    if cm.EffectType == EffectTypeHardBlock || cm.EffectType == EffectTypeOverride {
        // G5 handles these. G14 must not touch them.
        continue
    }
    if cm.EffectType == EffectTypeSymptomModification {
        // CM04 BB-masking type — no prior change. Handled by G8 (question weighting).
        // Log the firing for the clinician output, but apply no log-odds shift.
        continue
    }

    for _, target := range cm.Targets {
        acc, ok := accumulators[target.DifferentialID]
        if !ok {
            // CM targets a differential not in this node's set — log and skip.
            a.logger.Warn("CM targets unknown differential",
                "cm_id", cm.ID, "differential_id", target.DifferentialID)
            continue
        }

        dl := deltaLogOdds(target.DeltaP)
        acc.totalDeltaLogOdds += dl
        acc.cmCount++
    }
}

// Step 3: Apply cap and warn. Cap total CM log-odds shift at ±2.0 per differential.
// This prevents extreme posterior values from CM stacking on a single differential.
// ±2.0 corresponds to an OR ceiling of ~7.4, sufficient for any single risk factor cluster.
const cmLogOddsCap = 2.0
const cmStackWarningThreshold = 3

for dxID, acc := range accumulators {
    if acc.cmCount >= cmStackWarningThreshold {
        a.logger.Warn("CM_STACKED: multiple CMs targeting same differential",
            "differential_id", dxID,
            "cm_count", acc.cmCount,
            "total_delta_logodds", acc.totalDeltaLogOdds,
        )
        // Emit to hpi.session.events Kafka topic for Tier A clinical review
        a.emitCMStackedEvent(dxID, acc.cmCount, acc.totalDeltaLogOdds)
    }

    if acc.totalDeltaLogOdds > cmLogOddsCap {
        a.logger.Info("CM log-odds cap applied",
            "differential_id", dxID,
            "uncapped", acc.totalDeltaLogOdds,
            "capped_to", cmLogOddsCap,
        )
        acc.totalDeltaLogOdds = cmLogOddsCap
    }
    if acc.totalDeltaLogOdds < -cmLogOddsCap {
        a.logger.Info("CM log-odds cap applied (negative)",
            "differential_id", dxID,
            "uncapped", acc.totalDeltaLogOdds,
            "capped_to", -cmLogOddsCap,
        )
        acc.totalDeltaLogOdds = -cmLogOddsCap
    }
}

// Step 4: Compute adjusted posteriors in probability space.
// Do NOT normalize here — normalization happens in BayesianEngine.GetPosteriors()
// after safety floor clamping (G1) has run. Normalizing twice (here and in
// GetPosteriors) would double-normalize. The pre-normalization posterior sum
// will be > 1.0 when CMs increase multiple differentials simultaneously — this
// is expected and correct; GetPosteriors handles it.
adjusted := make(map[string]float64, len(posteriors))
for dxID, acc := range accumulators {
    adjusted[dxID] = sigmoid(acc.baseLogOdds + acc.totalDeltaLogOdds)
}

return adjusted, firedCMs, nil
```

### 4.3 What does NOT change in the return contract

- Return type: `(map[string]float64, []FiredCM, error)` — unchanged
- `FiredCM` structs: still populated with `cm_id`, `target_differential`, `delta_p`
  (the original author value, not the converted delta_logodds). The delta_logodds
  is an implementation detail; callers see the semantic value.
- `HARD_BLOCK` CMs: still returned in `FiredCM` list with `effect_type: HARD_BLOCK`
  so the clinician output template and G5 handler can process them.

### 4.4 Provenance logging addition

After Step 4, append per-differential CM provenance to the session audit trail.
This feeds the `element_attributions` table in the Clinical Source Registry (E07):

```go
for dxID, acc := range accumulators {
    if acc.cmCount > 0 {
        a.auditLog.RecordCMApplication(CMApplicationRecord{
            SessionID:         ctx.SessionID,
            DifferentialID:    dxID,
            BaseProbability:   posteriors[dxID],
            AdjustedProbability: adjusted[dxID],
            BaseLogOdds:       acc.baseLogOdds,
            TotalDeltaLogOdds: acc.totalDeltaLogOdds,
            CMCount:           acc.cmCount,
            WasCapped:         math.Abs(acc.totalDeltaLogOdds) == cmLogOddsCap,
        })
    }
}
```

---

## 5. YAML Schema: No Changes Required

**Authors continue writing delta_p exactly as before.** The conversion is internal.

```yaml
# YAML author writes this — unchanged before and after G14
context_modifiers:
  - id: CM01_DIZ
    target_differential: OH
    delta_p: +0.20        # <-- author declares probability-space shift at 0.50 baseline
    effect_type: INCREASE_PRIOR
```

The engine converts `+0.20` to `+0.8473` log-odds internally during `Apply()`.
YAML authors never see or write log-odds values.

**What YAML validation must enforce (NodeLoader.validate()):**

```go
// Add to existing CM validation in node_loader.go
if math.Abs(cm.DeltaP) >= 0.49 {
    return ValidationError{
        Field:   fmt.Sprintf("cm[%s].delta_p", cm.ID),
        Message: "delta_p must be in (-0.49, +0.49); |delta_p| >= 0.49 causes logit overflow",
        Value:   cm.DeltaP,
    }
}
```

**Also validate at load time**: if `effect_type` is `INCREASE_PRIOR` or `DECREASE_PRIOR`,
`delta_p` must be present and non-zero. If `effect_type` is `HARD_BLOCK` or `OVERRIDE`,
`delta_p` must be absent (G5 handles those).

---

## 6. Cap Design Rationale

The cap of ±2.0 total delta_logodds per differential is a clinical safety constraint,
not a mathematical requirement. Here is what it means in practice:

| Base prior | Max posterior after +2.0 cap | Interpretation |
|---|---|---|
| 0.04 (TIA) | 0.24 | CMs can raise a rare diagnosis to moderate; not to dominant |
| 0.10 | 0.45 | CMs raise an uncommon to near-even |
| 0.18 (ACS) | 0.62 | CMs raise to probable but not certain |
| 0.24 (OH) | 0.70 | Maximum OH prior from CM stacking |

The cap prevents a scenario where 4–5 CMs on a single differential push its posterior
to 0.90+ before any questions are asked, rendering the entire question sequence
clinically meaningless (the node would immediately converge on the CM-driven
differential regardless of patient answers).

**The cap is a log-odds sum cap, not a probability cap.** Applying a probability cap
(e.g., "posterior never exceeds 0.80 from CMs") is wrong because the same cap would
behave differently depending on what base prior it was applied to. The log-odds cap
is uniform in evidential weight regardless of base prior.

**When to revisit the cap**: if Tier A expert panel review finds that 3-CM stacking
on OH in the Indian elderly DM+HTN cohort is consistently underpredicting OH (i.e.,
clinicians adjudicate OH but the engine's capped posterior was lower), the cap may
need raising to ±2.5 or ±3.0. The `CM_STACKED` warning event provides exactly the
data needed for this review.

---

## 7. Unit Tests

All tests go in `cm_applicator_test.go`. The test philosophy: test the math, not the
mocks. Use real `deltaLogOdds()` and `sigmoid()` calls in assertions.

### Test 1: Single CM, INCREASE_PRIOR — baseline correctness

```go
func TestApply_SingleCM_IncreasesPriorInLogOddsSpace(t *testing.T) {
    posteriors := map[string]float64{"OH": 0.25, "HYPOGLYCEMIA": 0.17}
    cms := []ContextModifier{{
        ID: "CM01_DIZ", EffectType: EffectTypeIncreasePrior,
        Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}},
    }}

    result, _, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    expectedLogOdds := logit(0.25) + deltaLogOdds(0.20)
    expectedP := sigmoid(expectedLogOdds)

    // Tolerance: 1e-6 (float64 arithmetic precision)
    assert.InDelta(t, expectedP, result["OH"], 1e-6,
        "Single CM must apply log-odds shift, not probability addition")

    // Untargeted differential must be unchanged
    assert.InDelta(t, 0.17, result["HYPOGLYCEMIA"], 1e-6,
        "Non-targeted differential must not be modified")
}
```

### Test 2: Two CMs on same differential — the core G14 case

```go
func TestApply_TwoCMs_SameDifferential_LogOddsAdditive(t *testing.T) {
    posteriors := map[string]float64{"OH": 0.25}
    cms := []ContextModifier{
        {ID: "CM01", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM07", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
    }

    result, _, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    // Counter-proposal §10 canonical example
    // logodds(-1.099) + 0.847 + 0.847 = +0.595 → sigmoid = 0.6447
    assert.InDelta(t, 0.6447, result["OH"], 0.001,
        "Two CMs in log-odds space must give 0.6447, not naive sum 0.65")

    // Explicitly verify it is NOT the naive sum
    naiveSum := 0.25 + 0.20 + 0.20
    assert.NotEqual(t, naiveSum, result["OH"],
        "Result must NOT be naive probability sum 0.65")
}
```

### Test 3: Four CMs — naive sum exceeds 1.0, log-odds stays valid

```go
func TestApply_FourCMs_NaiveSumExceedsOne_LogOddsStaysValid(t *testing.T) {
    // Patient: ARB + HCTZ + SU + recent med change — all target OH/HYPOGLYCEMIA
    posteriors := map[string]float64{
        "OH": 0.24, "HYPOGLYCEMIA": 0.17, "DRUG_INDUCED": 0.12,
        "VOLUME_DEPLETION": 0.06, "OTHER": 0.15,
        // ... remaining differentials
    }
    cms := []ContextModifier{
        {ID: "CM01", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{
                {DifferentialID: "OH", DeltaP: 0.20},
                {DifferentialID: "VOLUME_DEPLETION", DeltaP: 0.10},
            }},
        {ID: "CM03a", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "HYPOGLYCEMIA", DeltaP: 0.30}}},
        {ID: "CM07", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM08", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{
                {DifferentialID: "DRUG_INDUCED", DeltaP: 0.25},
                {DifferentialID: "OH", DeltaP: 0.15},
            }},
    }

    result, _, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    // All values must be valid probabilities
    for dxID, p := range result {
        assert.Greater(t, p, 0.0, "Posterior for %s must be > 0", dxID)
        assert.Less(t, p, 1.0, "Posterior for %s must be < 1.0", dxID)
    }

    // OH specifically: 3 CMs stacked — should be capped
    // OH: logit(0.24) + 0.847 + 0.847 + 0.619 = 2.313 → exceeds cap → capped to 2.0
    // sigmoid(logit(0.24) + 2.0) = 0.700
    assert.InDelta(t, 0.700, result["OH"], 0.005,
        "OH with 3 CMs exceeding +2.0 cap must be capped at sigmoid(logit(0.24)+2.0)")
}
```

### Test 4: HARD_BLOCK and OVERRIDE are passed through without log-odds math

```go
func TestApply_HardBlockAndOverride_NotTouchedByG14(t *testing.T) {
    posteriors := map[string]float64{"ACS": 0.18}
    cms := []ContextModifier{
        {ID: "CM06_PDE5i", EffectType: EffectTypeHardBlock,
            Targets: []CMTarget{{DifferentialID: "ACS", DeltaP: 0}}},
    }

    result, fired, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    // HARD_BLOCK must not change ACS posterior — G5 handles it downstream
    assert.InDelta(t, 0.18, result["ACS"], 1e-6,
        "HARD_BLOCK CM must not modify posterior in G14; G5 handles separately")

    // But it must appear in the fired CMs list for G5 to process
    require.Len(t, fired, 1)
    assert.Equal(t, EffectTypeHardBlock, fired[0].EffectType)
}
```

### Test 5: DECREASE_PRIOR — negative delta_p produces negative delta_logodds

```go
func TestApply_DecreasePrior_NegativeDeltaLogOdds(t *testing.T) {
    posteriors := map[string]float64{"GERD": 0.25}
    cms := []ContextModifier{{
        ID: "CM_SM03_GERD_CEILING", EffectType: EffectTypeDecreasePrior,
        Targets: []CMTarget{{DifferentialID: "GERD", DeltaP: -0.20}},
    }}

    result, _, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    expectedLogOdds := logit(0.25) + deltaLogOdds(-0.20)
    expectedP := sigmoid(expectedLogOdds)

    assert.InDelta(t, expectedP, result["GERD"], 1e-6)
    assert.Less(t, result["GERD"], 0.25, "DECREASE_PRIOR must reduce posterior")
}
```

### Test 6: Order independence — CM application order must not affect result

```go
func TestApply_OrderIndependence(t *testing.T) {
    posteriors := map[string]float64{"OH": 0.24}

    cms_order1 := []ContextModifier{
        {ID: "CM01", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM07", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM08", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.15}}},
    }
    cms_order2 := []ContextModifier{cms_order1[2], cms_order1[0], cms_order1[1]}
    cms_order3 := []ContextModifier{cms_order1[1], cms_order1[2], cms_order1[0]}

    r1, _, _ := applicator.Apply(cms_order1, map[string]float64{"OH": 0.24}, ctx)
    r2, _, _ := applicator.Apply(cms_order2, map[string]float64{"OH": 0.24}, ctx)
    r3, _, _ := applicator.Apply(cms_order3, map[string]float64{"OH": 0.24}, ctx)

    assert.InDelta(t, r1["OH"], r2["OH"], 1e-10, "Order must not affect result")
    assert.InDelta(t, r1["OH"], r3["OH"], 1e-10, "Order must not affect result")
}
```

### Test 7: Cap fires at +2.0 total delta_logodds

```go
func TestApply_Cap_FiresAtTwoLogOdds(t *testing.T) {
    // 3 CMs of +0.20 each: total delta_logodds = 3 × 0.8473 = 2.542 → capped to 2.0
    posteriors := map[string]float64{"OH": 0.24}
    cms := []ContextModifier{
        {ID: "CM01", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM07", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
        {ID: "CM_EXTRA", EffectType: EffectTypeIncreasePrior,
            Targets: []CMTarget{{DifferentialID: "OH", DeltaP: 0.20}}},
    }

    result, _, err := applicator.Apply(cms, posteriors, ctx)
    require.NoError(t, err)

    // sigmoid(logit(0.24) + 2.0) = 0.700
    expected := sigmoid(logit(0.24) + 2.0)
    uncapped := sigmoid(logit(0.24) + 3*deltaLogOdds(0.20))

    assert.InDelta(t, expected, result["OH"], 0.001,
        "Result must be capped value (0.700), not uncapped (%.4f)", uncapped)
    assert.Less(t, result["OH"], uncapped,
        "Capped result must be less than uncapped result")
}
```

### Test 8: deltaLogOdds() helper — conversion accuracy

```go
func TestDeltaLogOdds_ConversionAccuracy(t *testing.T) {
    cases := []struct {
        deltaP         float64
        expectedDeltaLO float64
        expectedOR     float64
    }{
        {0.20, 0.8473, 2.33},
        {0.25, 1.0986, 3.00},
        {0.30, 1.3863, 4.00},
        {-0.20, -0.8473, 0.43},
    }
    for _, tc := range cases {
        dl := deltaLogOdds(tc.deltaP)
        assert.InDelta(t, tc.expectedDeltaLO, dl, 0.001,
            "deltaLogOdds(%v)", tc.deltaP)
        assert.InDelta(t, tc.expectedOR, math.Exp(dl), 0.01,
            "OR for deltaP=%v", tc.deltaP)
    }
}
```

---

## 8. Integration Test: Full P00 Session Smoke Test

This is the Day 1 verification that G14 works end-to-end with the P00 YAML.

**Scenario**: Patient, male, 68 years old, on ARB + SU + metoprolol + HCTZ. No
answers given yet (CM-only posterior).

**Expected CMs to fire**: CM01_DIZ (ARB→OH), CM03a_DIZ (SU→HYPOGLYCEMIA),
CM04_DIZ (BB→symptom modification only, no prior change), CM07_DIZ (age≥65 +
2 antihypertensives→OH).

**Expected posterior order** (before questions): OH > HYPOGLYCEMIA > DRUG_INDUCED
> BPPV, with OH substantially elevated from dual CM01+CM07 stacking.

```go
func TestIntegration_P00_CMOnlyPosterior(t *testing.T) {
    node := loadNode(t, "p00_dizziness_v2.yaml")
    patient := PatientContext{
        Age: 68, Sex: "Male",
        Medications: []string{"ARB", "SULFONYLUREA", "BETA_BLOCKER", "THIAZIDE"},
        HbA1c: 8.2, EGfr: 62,
    }

    session := engine.InitSession(node, patient)
    posteriors := session.GetPosteriors()

    // OH must be top differential — two CMs firing
    assert.Equal(t, "OH", topDifferential(posteriors),
        "OH must be top differential when ARB + polypharmacy CMs fire")

    // HYPOGLYCEMIA must be second — SU CM firing
    assert.Equal(t, "HYPOGLYCEMIA", secondDifferential(posteriors))

    // CM04 (BB) must appear in fired CMs but must NOT have shifted posteriors
    firedIDs := getFiredCMIDs(session)
    assert.Contains(t, firedIDs, "CM04_DIZ")

    // No posterior must be invalid
    for dxID, p := range posteriors {
        assert.Greater(t, p, 0.0, "dx %s must have positive posterior", dxID)
        assert.LessOrEqual(t, p, 1.0, "dx %s must not exceed 1.0", dxID)
    }

    // Posteriors must sum to approximately 1.0 (after GetPosteriors normalization)
    total := sumValues(posteriors)
    assert.InDelta(t, 1.0, total, 0.001, "Normalized posteriors must sum to 1.0")
}
```

---

## 9. Validation Experiment

Before merging G14 to main, run the following comparison on 50 synthetic patient
profiles spanning the polypharmacy spectrum. This validates that G14 improves
clinical fidelity without introducing regressions.

**Profile set construction**:
- 10 profiles: 1 CM firing (G14 and naive should agree within 0.005)
- 15 profiles: 2 CMs on same differential (divergence expected; log-odds correct)
- 15 profiles: 3 CMs on same differential (divergence large; cap may fire)
- 10 profiles: CMs on different differentials (minimal divergence expected)

**Pass criteria**:
- All 10 single-CM profiles: naive vs G14 within 0.005 ✓ (expected — minimal
  divergence at single CM)
- All multi-CM profiles: G14 posteriors are valid probabilities (0,1) ✓
- All naive-sum profiles: naive posterior sums checked — confirm G14 sum stays
  closer to 1.0 before normalization than naive approach
- Zero posteriors exceed 1.0 after G14 (naive approach fails this for 3+ CMs)

---

## 10. Deployment Checklist

- [ ] `deltaLogOdds()`, `logit()`, `sigmoid()` helpers added and unit tested
- [ ] `Apply()` inner loop replaced with log-odds accumulator
- [ ] Cap logic at ±2.0 with `CM_STACKED` warning at count≥3
- [ ] `HARD_BLOCK` and `OVERRIDE` explicitly skipped in G14 loop
- [ ] `SYMPTOM_MODIFICATION` (CM04 type) explicitly skipped in G14 loop
- [ ] `delta_p` bounds validation added to `NodeLoader.validate()`
- [ ] All 8 unit tests pass
- [ ] Integration smoke test on P00 YAML passes
- [ ] 50-profile validation experiment run and documented
- [ ] `CMApplicationRecord` provenance logging wired to audit trail
- [ ] `CM_STACKED` Kafka event wired to `hpi.session.events`
- [ ] No change to `Apply()` return type or `FiredCM` struct
- [ ] `HARD_BLOCK` CMs still appear in `FiredCM` list for G5 processing
- [ ] Backward compatibility: existing P01 V1 YAML with single-CM entries still loads and behaves identically (single-CM case: G14 ≈ naive within 0.005)

---

## 11. What G14 Deliberately Does Not Solve

These are in scope for other Go changes and are called out here to prevent scope creep
into G14:

| Item | Correct Go Change |
|---|---|
| HARD_BLOCK (PDE5i nitrate veto) | G5 |
| OVERRIDE (Hb<8 forces anaemia posterior) | G5 |
| BB suppression of DQ05 LR contribution | G8 |
| Safety floor clamping after CM application | G1 (runs in GetPosteriors, after Apply) |
| Sex modifier prior shifts | G2 (runs before Apply in InitSession) |
| 'Other' bucket receiving log-odds shifts from CMs | G15 (Other bucket is implicit; Apply targets listed differentials only) |

---

*G14 Implementation Spec | Vaidshala KB-22 HPI Engine | March 2026*
*File: `kb-22/.../services/cm_applicator.go` | Estimated scope: ~150 lines net change*
