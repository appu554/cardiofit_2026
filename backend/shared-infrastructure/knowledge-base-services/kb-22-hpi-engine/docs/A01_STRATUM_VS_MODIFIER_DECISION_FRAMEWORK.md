# A01: Stratum vs Modifier Decision Framework

**Gap ID**: A01
**Owner**: HPI Engine Team
**Blocks**: ALL nodes (mandatory pre-authoring step)
**Status**: ACTIVE
**Version**: 1.0.0

## Purpose

When a clinical signal (comorbidity, lab result, medication, demographic) affects differential
diagnosis probabilities, the node author must decide: model it as a **STRATUM** (separate prior
column in the `priors` table) or as a **MODIFIER** (context modifier probability delta).

Wrong choice degrades diagnostic accuracy:
- Overuse of strata creates combinatorial explosion (3 comorbidities x 2 levels = 8 columns)
- Overuse of modifiers loses the ability to capture correlated multi-differential shifts

This 1-page framework resolves the decision via 4 binary questions.

## Decision Flowchart

```
                    Clinical Signal Identified
                              |
                    Q1: Does this signal shift the
                    PRIOR PROBABILITY of >= 3
                    differentials simultaneously?
                              |
                    +----YES----+----NO----+
                    |                      |
            Q2: Does the shift             MODIFIER
            MAGNITUDE differ               (CM with single-
            by >= 2x across                differential
            differentials?                 adjustment)
                    |
            +----YES----+----NO----+
            |                      |
    Q3: Is this signal              MODIFIER
    STABLE for the entire           (CM with multi-
    session duration?               differential
    (not treatment-dependent)       adjustments map)
            |
    +----YES----+----NO----+
    |                      |
Q4: Does published          MODIFIER
evidence provide            (Use CM; priors
STRATUM-SPECIFIC            would be
priors from a               fabricated without
population study?           evidence)
    |
+----YES----+----NO----+
|                      |
STRATUM                MODIFIER
(Add column to         (Use CM until
priors table)          evidence harvest
                       provides stratum
                       data)
```

## The Four Questions

### Q1: Multi-Differential Shift (>= 3 differentials)

**Ask**: "Does this signal change the pre-test probability of 3 or more differentials at once?"

| Answer | Reasoning | Example |
|--------|-----------|---------|
| YES | Signal fundamentally reshapes the diagnostic landscape | CKD stage shifts ACS, ADHF, PE, Anaemia, Pericarditis simultaneously |
| NO | Signal targets 1-2 differentials only | Antiplatelet+NSAID mainly shifts GERD/GI bleed |

**If NO -> MODIFIER.** Single-target or dual-target shifts are cleanly handled by CM adjustments.

### Q2: Magnitude Asymmetry (>= 2x difference)

**Ask**: "Among the shifted differentials, does the largest shift exceed the smallest by >= 2x?"

| Answer | Reasoning | Example |
|--------|-----------|---------|
| YES | Different differentials respond non-uniformly to this signal | CKD: ADHF prior increases 3x (0.12 -> 0.35) while Asthma drops 2x (0.18 -> 0.05) |
| NO | All differentials shift roughly proportionally | Age >= 65 increases most priors by ~1.3-1.5x uniformly |

**If NO -> MODIFIER.** Uniform shifts are well-modelled by a multi-differential CM (each adjustment approximately equal).

### Q3: Session Stability

**Ask**: "Is this signal fixed for the entire HPI session, or could it change based on treatment decisions?"

| Answer | Reasoning | Example |
|--------|-----------|---------|
| YES (stable) | Signal is determined at session start and doesn't change | CKD stage, DM duration, sex, age, known HF history |
| NO (dynamic) | Signal depends on current medications or recent interventions | "On SGLT2i" can change if drug is stopped; "recent med change" is transient |

**If NO -> MODIFIER.** Strata are fixed at session initialization. Dynamic signals must be modifiers that can be toggled by KB-20 updates.

### Q4: Evidence Availability

**Ask**: "Does published literature provide population-level prior probabilities stratified by this signal?"

| Answer | Reasoning | Example |
|--------|-----------|---------|
| YES | Peer-reviewed source with stratum-specific prevalence data | Jaipur Heart Watch provides DM+HTN-specific ACS prevalence; ARIC provides CKD-stratified HF incidence |
| NO | No published stratum-specific data; values would need fabrication | No published "CKD 3b-specific chest pain differential" prevalence study |

**If NO -> MODIFIER.** Do not fabricate stratum priors. Use a CM with the best available magnitude estimate and mark as CALIBRATE. Upgrade to STRATUM when evidence harvest provides real population data.

## Decision Matrix Summary

| Q1 (>=3 diffs) | Q2 (>=2x asymmetry) | Q3 (stable) | Q4 (evidence) | Decision |
|:-:|:-:|:-:|:-:|:--|
| NO | - | - | - | **MODIFIER** (single/dual-target CM) |
| YES | NO | - | - | **MODIFIER** (multi-target CM, uniform shifts) |
| YES | YES | NO | - | **MODIFIER** (dynamic signal, CM with toggle) |
| YES | YES | YES | NO | **MODIFIER** (no evidence; mark CALIBRATE) |
| YES | YES | YES | YES | **STRATUM** (add prior column) |

**Only when ALL FOUR answers are YES does a signal warrant a new stratum column.**

## Worked Examples

### Example 1: CKD Stage -> STRATUM

| Question | Answer | Evidence |
|----------|--------|----------|
| Q1: Shifts >= 3 diffs? | YES | CKD shifts ACS, ADHF, PE, Pericarditis, Anaemia (5 diffs) |
| Q2: >= 2x asymmetry? | YES | ADHF: 0.12 -> 0.35 (2.9x); Asthma: 0.18 -> 0.05 (0.28x) |
| Q3: Stable for session? | YES | CKD stage doesn't change during a 10-minute HPI |
| Q4: Published evidence? | YES | ARIC CKD substudy; KDIGO 2024 provides CKD-stratified cardiovascular event rates |
| **Decision** | **STRATUM** | Add DM_HTN_CKD column to prior table |

### Example 2: SGLT2i use -> MODIFIER

| Question | Answer | Evidence |
|----------|--------|----------|
| Q1: Shifts >= 3 diffs? | NO | Primarily activates EUGLYCEMIC_DKA (1 differential) |
| **Decision** | **MODIFIER** | CM05: activation_condition + prior delta |

### Example 3: Age >= 65 + polypharmacy -> MODIFIER

| Question | Answer | Evidence |
|----------|--------|----------|
| Q1: Shifts >= 3 diffs? | YES | Shifts OH, MSK, Anxiety, drug-induced differentials |
| Q2: >= 2x asymmetry? | NO | All shifts are moderate (~1.3-1.8x), roughly proportional |
| **Decision** | **MODIFIER** | CM09: multi-differential CM with uniform-ish adjustments |

### Example 4: Heart Failure history -> STRATUM (P02) or MODIFIER (P01)?

| Question | P02 (Dyspnea) | P01 (Chest Pain) |
|----------|---------------|-----------------|
| Q1: Shifts >= 3 diffs? | YES (ADHF, PE, Anaemia, Pneumonia) | YES (ACS, PE, ADHF-related) |
| Q2: >= 2x asymmetry? | YES (ADHF 3x; Asthma 0.3x) | NO (ACS 1.5x; PE 1.3x — similar) |
| Q3: Stable? | YES | YES |
| Q4: Evidence? | YES (Wang JAMA 2005) | NO (no chest-pain-in-HF stratum data) |
| **Decision** | **STRATUM** (DM_HTN_CKD_HF column) | **MODIFIER** (CM with HF adjustments) |

This demonstrates that the **same clinical signal** (HF) can be a STRATUM in one node and a MODIFIER in another, depending on impact magnitude and evidence availability.

## Integration with A02 and A03

- **A02 (Parameterised Strata Schema)**: When A01 decides STRATUM, A02 defines the YAML schema for the new prior column, LR overrides, and stratum activation rules.
- **A03 (Safety Floor Pattern)**: When A01 decides STRATUM, A03 requires stratum-specific safety floors (not a single row). When A01 decides single-stratum (or MODIFIER-only), a single safety floor row is sufficient.

## Pre-Authoring Checklist

Before authoring any new HPI node, the clinical team MUST:

1. List all clinical signals that affect differential probabilities
2. Run each signal through the A01 flowchart (4 questions)
3. Document the decision and evidence for each signal
4. For STRATUM decisions: verify evidence source provides per-stratum priors
5. For MODIFIER decisions: author CM entry with magnitude, source, and confidence tier
6. Submit A01 worksheet for Canon Framework review before YAML authoring begins

## Current Node A01 Audit

| Node | Strata | Modifiers | A01 Compliant | Notes |
|------|--------|-----------|:---:|-------|
| P01 V2 | DM_HTN_base (single) | CM01-CM10 | YES | CKD/HF columns deferred (no Q4 evidence yet) |
| P02 V1 | DM_ONLY, DM_HTN, DM_HTN_CKD | CM01-CM10 | PARTIAL | Needs DM_HTN_CKD_HF (G4); has Q4 evidence from ARIC/Wang |
| P00 V2 | DM_HTN_base (single) | CM01-CM08 | YES | Dizziness has limited stratum evidence |
| P03-P08 | TBD | TBD | PENDING | Requires A01 worksheet before authoring |
