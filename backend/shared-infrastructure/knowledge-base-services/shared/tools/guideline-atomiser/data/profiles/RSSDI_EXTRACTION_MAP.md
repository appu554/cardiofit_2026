# RSSDI 2024 Extraction Map

## Purpose

This document maps every ADA 2026 rule (Tier 1 universal) to its RSSDI 2024 override (Tier 2 India), organized by the 5 overlap zones. Each entry specifies: the ADA rule being overridden, the RSSDI modification, the resolution logic, and the conflict group ID for the reviewer dashboard.

The extraction pipeline should tag every RSSDI span with its corresponding ADA Section node and KDIGO rule ID (where applicable) so the reviewer dashboard presents them side-by-side.

---

## Architecture: 3-Tier Guideline Resolution

```
Tier 1 — Universal Canonical Layer (ADA 2026 + KDIGO 2024)
  │
  ├── Tier 2 — Country Override: India (RSSDI 2024 + CSI 2020 + ICMR)
  │     └── Tier 3 — Patient-Level (KB-20 state machine)
  │
  └── Tier 2 — Country Override: Australia (RACGP + Diabetes Australia + KHA)
        └── Tier 3 — Patient-Level (KB-20 state machine)
```

### Resolution Mechanisms

| Mechanism | Description | Implementation |
|-----------|-------------|----------------|
| **M1: Domain-Scoped Source Priority** | Each clinical domain has one authoritative guideline | KB-23 `source_priority` field per domain |
| **M2: Specificity Cascade** | Most specific rule (highest antecedent count) wins | KB-23 `specificity_score` field (int) |
| **M3: Conflict Tagging** | Unresolvable conflicts tagged for clinical advisory board | `l2_merged_spans.conflict_group_id` |

### Domain Authority Hierarchy

| Clinical Domain | Authoritative Source | Confirmatory | Country Override |
|-----------------|---------------------|--------------|------------------|
| CKD drug rules + monitoring | KDIGO 2024 | ADA S11 | RSSDI (access/cost) |
| Glycemic targets + drug sequencing | ADA 2026 S6/S9 | KDIGO 2022 | RSSDI (population thresholds) |
| BP + lipid management | ADA 2026 S10 | KDIGO 2024 S3.4 | RSSDI + CSI 2020 |
| Age-adjusted modifications | ADA 2026 S13 | — | RSSDI (Indian life expectancy) |
| Monitoring cadences | KDIGO 2024 Fig 11.1 | ADA S4/S11 | RSSDI (infra feasibility) |

---

## Overlap Zone 1: CKD Drug Rules

**Density: HIGH** — Densest overlap. ADA S11 + KDIGO 2024 Ch3 + RSSDI CKD chapter all specify the same drug initiation/titration rules.

### Extraction Targets from RSSDI 2024

| RSSDI Rule | ADA Tier 1 Rule (S11) | KDIGO Tier 1 Rule | Override Type | Conflict Group |
|------------|----------------------|-------------------|---------------|----------------|
| SGLT2i initiation eGFR threshold | ADA: eGFR ≥20 | KDIGO: eGFR ≥20 | **NULL** (aligned) | CG-CKD-SGLT2I-01 |
| SGLT2i: remogliflozin as option | ADA: empa/dapa/cana only | KDIGO: empa/dapa only | **ADDITIVE** — India-only drug | CG-CKD-SGLT2I-02 |
| Finerenone initiation criteria | ADA: eGFR ≥25, K ≤4.8 | KDIGO: eGFR ≥25, K ≤4.8 | **RESTRICTED** — nsMRA pathway closed if finerenone unavailable/unaffordable AND K monitoring infeasible in Tier 2/3; engine continues on max ACEi/ARB + SGLT2i without finerenone layer | CG-CKD-FINER-01 |
| Finerenone monitoring cadence | ADA: K at 1wk, 4wk | KDIGO: K at 1wk, 4wk post-initiation | **MODIFIED** — RSSDI may specify longer intervals if lab access limited | CG-CKD-FINER-02 |
| Metformin eGFR cutoff | ADA: contraindicated eGFR <30 | KDIGO: avoid <30, reduce dose 30-44 | **NULL** (aligned) | CG-CKD-MET-01 |
| ACEi/ARB max tolerated dose | ADA: max tolerated dose | KDIGO: creatinine rise ≤30% acceptable | **COMPLEMENTARY** — ADA says "what", KDIGO says "how" | CG-CKD-RASI-01 |
| ACEi/ARB: specific agents | ADA: class-level | KDIGO: class-level | **RSSDI OVERRIDE** — RSSDI may specify ramipril/telmisartan as preferred (Indian trial data, cost) | CG-CKD-RASI-02 |
| GLP-1 RA in CKD | ADA: recommended for CKD | KDIGO: recommended | **MODIFIED** — RSSDI flags cost barrier (₹8,000-15,000/month vs ₹200-500 target), positions as aspirational | CG-CKD-GLP1-01 |

### Pipeline Instructions
- Tag every RSSDI CKD drug mention with the corresponding ADA S11 recommendation number (e.g., `11.3a`, `11.7b`)
- Tag with KDIGO 2024 recommendation ID (e.g., `Rec 3.7.1`, `Rec 3.8.2`)
- Flag any RSSDI rule where the drug (finerenone, GLP-1 RA) has an India availability caveat → `formulary_context` pattern match

---

## Overlap Zone 2: BP Targets in CKD + Diabetes

**Density: MEDIUM** — Four sources give slightly different BP targets for the same patient.

### Extraction Targets from RSSDI 2024

| RSSDI Rule | ADA Tier 1 Rule | KDIGO Tier 1 Rule | Override Type | Conflict Group |
|------------|----------------|-------------------|---------------|----------------|
| General BP target | ADA S10: <130/80 | KDIGO: SBP <120 when tolerated | **RSSDI POSITION** — extract whether RSSDI adopts BPROAD <120 or stays at <130/80 | CG-BP-GEN-01 |
| BP target: CKD + DM | ADA S11: <130/80 | KDIGO S3.4: SBP <120 | **RESOLUTION NEEDED** — RSSDI + CSI 2020 position | CG-BP-CKD-01 |
| BP target: elderly (>65) | ADA S13: <130/80 healthy, <140/90 complex | KDIGO: individualize | **RSSDI OVERRIDE** — may have different age threshold given Indian life expectancy | CG-BP-AGE-01 |
| Preferred antihypertensive class | ADA S10: ACEi/ARB first | KDIGO: ACEi/ARB first | **RSSDI OVERRIDE** — extract preferred specific agents (amlodipine widely used in India as add-on) | CG-BP-DRUG-01 |
| BPROAD applicability | ADA S10: SBP <120 high-CV-risk | Not yet in KDIGO | **CRITICAL CONFLICT** — BPROAD was primarily Chinese population; RSSDI position on applicability to Indian population needed | CG-BP-BPROAD-01 |

### Resolution Logic
- **M1**: For CKD-specific BP → KDIGO authoritative. For general DM BP → ADA authoritative. RSSDI overrides on agent selection and feasibility.
- **M2**: Patient with T2DM + CKD G3b + ASCVD + age 74 → ADA S13 age-adjusted target (specificity=4) overrides ADA S10 general target (specificity=2).
- **M3**: BPROAD applicability → tag as `UNRESOLVED` for clinical advisory board.

---

## Overlap Zone 3: Glycemic Targets in CKD

**Density: MEDIUM** — Complicated by A1C unreliability at advanced CKD + population-specific thresholds.

### Extraction Targets from RSSDI 2024

| RSSDI Rule | ADA Tier 1 Rule | KDIGO Tier 1 Rule | Override Type | Conflict Group |
|------------|----------------|-------------------|---------------|----------------|
| General A1C target | ADA S6: <7% most adults | KDIGO: individualized | **RSSDI OVERRIDE** — extract RSSDI target (may use <7% but with earlier intensification at lower BMI) | CG-GLYC-A1C-01 |
| A1C target: elderly | ADA S13: <7.5-8.0% complex | KDIGO: individualized | **RSSDI OVERRIDE** — Indian life expectancy and complication rates may shift thresholds | CG-GLYC-AGE-01 |
| A1C reliability in CKD G4-5 | ADA S11: use glycated albumin/fructosamine | KDIGO: recognized inaccuracy | **NULL** (aligned) — extract RSSDI position on alternative markers availability in India | CG-GLYC-CKD-01 |
| FPG/PPG targets | ADA S6: FPG 80-130, PPG <180 | — | **RSSDI OVERRIDE** — RSSDI may specify tighter PPG (<160 or <140) for Indian population with higher PPG-driven risk | CG-GLYC-FPG-01 |
| Intervention threshold (BMI-adjusted) | ADA: standard BMI cutoffs | — | **RSSDI OVERRIDE** — BMI ≥23 overweight (vs 25), earlier drug intensification at lower BMI per WHO Asia-Pacific | CG-GLYC-BMI-01 |
| CGM targets (TIR) | ADA S6: TIR >70% | — | **RSSDI POSITION** — extract whether RSSDI endorses CGM targets (cost/access barrier in India) | CG-GLYC-CGM-01 |

### Pipeline Instructions
- Every RSSDI glycemic threshold → tag with ADA S6 recommendation number
- BMI cutoffs → tag as `anthropometric_threshold` with `population_context` co-tag
- FPG-based monitoring recommendations → tag as monitoring cadence override (Zone 5)

---

## Overlap Zone 4: Drug Sequencing — Multi-Comorbidity

**Density: VERY HIGH** — The three-body problem. ADA S9/S10/S11/S13 + KDIGO Ch3 + RSSDI all fire for patients with T2DM + ASCVD + CKD + age.

### Extraction Targets from RSSDI 2024

| RSSDI Rule | ADA Tier 1 Rule (S9) | Override Type | Conflict Group |
|------------|---------------------|---------------|----------------|
| First-line: metformin | ADA: metformin (unless contraindicated) | **NULL** (aligned) | CG-SEQ-LINE1-01 |
| Second-line: ASCVD pathway | ADA: GLP-1 RA or SGLT2i | **RSSDI OVERRIDE** — GLP-1 RA cost-prohibitive → SGLT2i preferred; if SGLT2i contraindicated → pioglitazone (not in ADA pathway) | CG-SEQ-ASCVD-01 |
| Second-line: HF pathway | ADA: SGLT2i (proven benefit) | **MODIFIED** — aligned on SGLT2i, but RSSDI may specify empa/dapa preference based on Indian pricing | CG-SEQ-HF-01 |
| Second-line: CKD pathway | ADA: SGLT2i, then finerenone, then GLP-1 RA | **MODIFIED** — SGLT2i aligned; finerenone conditional on availability; GLP-1 RA aspirational | CG-SEQ-CKD-01 |
| Second-line: weight management | ADA: tirzepatide/semaglutide | **RSSDI OVERRIDE** — tirzepatide unavailable/expensive → SGLT2i for modest weight loss, or metformin XR dose optimization | CG-SEQ-WEIGHT-01 |
| Second-line: cost-constrained | Not in ADA (assumes insurance) | **RSSDI ADDITIVE** — entire cost-first pathway: metformin → SU (gliclazide preferred for lower hypo risk) → DPP-4i (teneligliptin ₹5-8/day) | CG-SEQ-COST-01 |
| Third-line add-on | ADA: based on comorbidity | **RSSDI OVERRIDE** — voglibose as PPG-targeting add-on (not in ADA algorithms), pioglitazone as insulin-sparing add-on | CG-SEQ-LINE3-01 |
| Insulin initiation | ADA: basal insulin when oral triple fails | **RSSDI OVERRIDE** — human insulin (NPH) still first-line insulin in India (cost: ₹80-120/vial vs ₹800-1500 for analogs); premixed 70/30 for patients needing basal+bolus but can't afford MDI | CG-SEQ-INSULIN-01 |
| Insulin analogs | ADA: glargine/degludec preferred | **RSSDI CONDITIONAL** — analogs preferred if affordable; biosimilar glargine (Basalog/Glaritus ₹350-500) makes this more accessible | CG-SEQ-ANALOG-01 |
| De-intensification | ADA S13: deintensify in elderly | **RSSDI POSITION** — extract RSSDI's deintensification protocol (may differ given Indian complication patterns) | CG-SEQ-DEINT-01 |

### Pipeline Instructions
- Every RSSDI drug sequencing recommendation → tag with ADA S9 Figure 9.4 pathway node
- Cost mentions → `formulary_context` pattern tag
- When RSSDI mentions a drug NOT in ADA's algorithm (voglibose, teneligliptin, human insulin NPH, premixed insulin) → tag as `ADDITIVE` override
- When RSSDI substitutes one drug for another (GLP-1 RA → pioglitazone, analog insulin → human insulin) → tag as `SUBSTITUTION` override

### Resolution Logic for Multi-Comorbidity Patient

For a patient with T2DM + CKD G3b + ASCVD + age 74:

```
Step 1: Collect all matching rules from ADA S9, S10, S11, S13 + KDIGO Ch3 + RSSDI
Step 2: Group by conflict_group_id
Step 3: Within each group, resolve by:
  a. Country override (India → RSSDI layer active)
  b. Specificity cascade (S13 age rule, specificity=4, overrides S9 general, specificity=1)
  c. Domain authority (CKD-specific → KDIGO; glycemic → ADA; formulary → RSSDI)
Step 4: Log all non-winning rules in decision trace for audit
```

---

## Overlap Zone 5: Monitoring Cadences

**Density: LOW-MEDIUM** — Not conflicting but layered. RSSDI adjusts for Indian lab infrastructure.

### Extraction Targets from RSSDI 2024

| RSSDI Rule | ADA Tier 1 Rule | KDIGO Tier 1 Rule | Override Type | Conflict Group |
|------------|----------------|-------------------|---------------|----------------|
| HbA1c frequency | ADA: Q3M uncontrolled, Q6M at target | — | **RSSDI OVERRIDE** — may specify FPG-based interim monitoring when quarterly HbA1c not feasible (Tier 2/3 cities) | CG-MON-A1C-01 |
| eGFR + UACR frequency | ADA S11: annually | KDIGO: by risk category (1-4x/year) | **RSSDI OVERRIDE** — KDIGO cadence ideal but RSSDI may specify minimum feasible cadence | CG-MON-EGFR-01 |
| Potassium monitoring (finerenone) | ADA: 1wk + 4wk post-initiation | KDIGO: 1wk + 4wk | **RSSDI CONDITIONAL** — if finerenone prescribed, same cadence; but flag if lab turnaround >48h | CG-MON-K-01 |
| Lipid panel frequency | ADA S10: annually or post-statin initiation | — | **NULL** (likely aligned) | CG-MON-LIPID-01 |
| Self-monitoring blood glucose | ADA: per clinical need | — | **RSSDI OVERRIDE** — extract RSSDI's structured SMBG protocol (Indian cost of test strips: ₹8-15/strip affects frequency) | CG-MON-SMBG-01 |
| Retinal screening | ADA S12: annually | — | **RSSDI POSITION** — extract frequency and whether teleophthalmology is recommended for Tier 2/3 | CG-MON-RETINA-01 |
| Foot examination | ADA S12: annually | — | **RSSDI POSITION** — Indian diabetic foot complication rates higher; may recommend more frequent | CG-MON-FOOT-01 |

### Resolution Logic
- Take the **more frequent** cadence when multiple sources apply
- RSSDI downward adjustments are `CONDITIONAL` — apply only when infrastructure constraint is flagged in KB-20 patient profile (care_setting = "tier2" or "tier3")
- For tier1 urban settings, use ADA/KDIGO cadences unchanged

---

## KB-23 Rule Schema Extension

Each rule in KB-23 decision card templates should carry these metadata fields for multi-guideline resolution:

```yaml
rule_metadata:
  source: "RSSDI-2022-Ch7-Rec7.3"          # Guideline + section + recommendation ID
  conflict_group: "CG-SEQ-ASCVD-01"         # Links all rules addressing same clinical decision
  domain: "drug-sequencing"                  # CKD-drug | glycemic-target | BP-target | monitoring-cadence | drug-sequencing
  specificity_score: 3                       # Count of antecedent conditions
  tier: 2                                    # 1=universal, 2=country-override, 3=patient-level
  country_code: "IN"                         # ISO 3166-1 alpha-2
  override_type: "SUBSTITUTION"              # NULL | ADDITIVE | MODIFIED | CONDITIONAL | SUBSTITUTION | RESTRICTED
  overrides:                                 # Rules this rule overrides
    - rule_id: "ADA-2026-S9-Rec9.5a"
      justification: "GLP-1 RA cost-prohibitive in Indian formulary context"
  ada_cross_ref: "S9-Rec9.5a"               # ADA rule this corresponds to
  kdigo_cross_ref: "Rec-3.7.1"              # KDIGO rule this corresponds to (if applicable)
  formulary_constraint: true                 # Whether this override is driven by formulary/cost
  population_specific: false                 # Whether this override is driven by population genetics/anthropometry
  infrastructure_dependent: false            # Whether this override depends on care setting tier
  source_lag: false                          # True when RSSDI 2022 predates evidence (tirzepatide, BPROAD, CONFIDENCE)
  source_lag_note: null                      # e.g., "Tirzepatide post-dates RSSDI 2022 — clinical advisory board review required"
```

### Override Resolution States

| State | Meaning | Dashboard Action |
|-------|---------|-----------------|
| `HARMONIZED` | Tier 1 and Tier 2 rules agree | Confirm both sources, cite both |
| `OVERRIDDEN` | Tier 2 takes precedence per domain hierarchy | Mark with resolution logic |
| `CONDITIONAL` | Tier 2 applies only under specific conditions (cost, infra, population) | Tag conditions in KB-20 |
| `ADDITIVE` | Tier 2 adds rules not present in Tier 1 (India-only drugs) | No conflict — add directly |
| `SUBSTITUTION` | Tier 2 replaces Tier 1 drug with alternative | Requires clinical review |
| `RESTRICTED` | Tier 1 drug/class unavailable or inaccessible in country market | Engine falls back to next tier in sequencing algorithm; no substitute exists |
| `UNRESOLVED` | Genuine clinical disagreement | Escalate to clinical advisory board |

---

## Source-Lag Warning: RSSDI 2022 vs ADA 2026

**CRITICAL**: RSSDI 2022 predates ADA 2026 by 4 years. Drugs and evidence that entered clinical practice after October 2022 are NOT addressed by RSSDI:

| Drug/Evidence | ADA 2026 Position | RSSDI 2022 Position | Source-Lag Risk |
|---------------|-------------------|---------------------|-----------------|
| Tirzepatide (dual GIP/GLP-1 RA) | First-line for weight, ASCVD, MASH | **Not mentioned** (approved 2022, post-publication) | HIGH — no RSSDI override exists; clinical advisory board must decide Indian positioning |
| Finerenone (nsMRA) | Recommended for CKD+albuminuria | **Minimal coverage** (FIDELIO-DKD published 2020, but RSSDI 2022 may have early mention) | MEDIUM — verify RSSDI 2022 Ch8 coverage |
| BPROAD (SBP <120) | Endorsed for high-CV-risk | **Not mentioned** (published 2023) | HIGH — CSI 2020 also predates this |
| SGLT2i eGFR ≥20 extension | Initiate at eGFR ≥20 | **May use older eGFR ≥25 threshold** | MEDIUM — check RSSDI 2022 Ch8 |
| CONFIDENCE trial (finerenone+SGLT2i) | Simultaneous use endorsed | **Not mentioned** (published 2024) | HIGH |

**Pipeline instruction**: Tag every RSSDI override on these drugs/evidence with `source_lag: true` metadata flag. The reviewer dashboard should display a warning banner: "RSSDI 2022 predates this evidence — clinical advisory board review required."

---

## RSSDI Chapter-to-ADA Section Cross-Reference

This mapping tells the pipeline which RSSDI chapters to tag against which ADA sections:

| RSSDI 2022 Chapter | ADA 2026 Section | KDIGO 2024 Chapter | Overlap Zone |
|--------------------|------------------|--------------------|--------------|
| Ch 1: Screening & Diagnosis | S2: Classification/Diagnosis | — | — (minimal overlap) |
| Ch 2: Glycemic Targets | S6: Glycemic Goals | — | Zone 3 |
| Ch 3: Lifestyle Management | S5: Lifestyle Management | — | — (minimal overlap) |
| Ch 4: Pharmacotherapy | S9: Pharmacologic Approaches | Ch 4: Medication Mgmt | Zone 4 |
| Ch 5: Insulin Therapy | S9: Pharmacologic (insulin section) | — | Zone 4 |
| Ch 6: Hypertension | S10: CVD Risk Management | S3.4: BP Management | Zone 2 |
| Ch 7: Dyslipidemia | S10: CVD Risk Management | — | Zone 2 |
| Ch 8: CKD in Diabetes | S11: CKD & Risk Management | Ch 3: Medication Mgmt | Zone 1 |
| Ch 9: CVD Prevention | S10: CVD Risk Management | S3.15-3.16: CVD | Zone 2 |
| Ch 10: Elderly | S13: Older Adults | — | Zone 3, 4 |
| Ch 11: Monitoring | S4/S6: Comprehensive Eval | Fig 11.1: Monitoring | Zone 5 |
| Ch 12: Special Populations | S14/S15: Special Populations | — | — |

---

## Extraction Priority Order

Extract RSSDI chapters in this order (mirrors ADA priority, maximizes overlap resolution):

1. **Ch 2 + Ch 4 + Ch 5** (glycemic targets + drug sequencing + insulin) → resolves Zones 3 + 4
2. **Ch 8** (CKD) → resolves Zone 1 against KDIGO
3. **Ch 6 + Ch 7 + Ch 9** (BP + lipids + CVD) → resolves Zone 2
4. **Ch 11** (monitoring) → resolves Zone 5
5. **Ch 10** (elderly) → resolves age-adjusted overrides across all zones
6. **Ch 1 + Ch 3 + Ch 12** (screening, lifestyle, special populations) → low overlap, extract last

---

## Reviewer Dashboard Integration

When the reviewer dashboard presents spans for review, ADA and RSSDI spans sharing a `conflict_group_id` should appear side-by-side:

```
┌─────────────────────────────────────────────────────────────────┐
│ Conflict Group: CG-SEQ-ASCVD-01                                │
│ Domain: drug-sequencing | Override Type: SUBSTITUTION           │
├─────────────────────┬───────────────────────────────────────────┤
│ TIER 1 (ADA S9)     │ TIER 2 (RSSDI Ch4)                       │
│                     │                                           │
│ "For T2DM with      │ "In Indian setting where GLP-1 RA is     │
│  ASCVD, GLP-1 RA    │  cost-prohibitive, SGLT2i is preferred   │
│  or SGLT2i with     │  second-line; if SGLT2i contraindicated,  │
│  proven CV benefit   │  consider pioglitazone 15-30mg."         │
│  is recommended."   │                                           │
├─────────────────────┴───────────────────────────────────────────┤
│ Resolution: [ ] HARMONIZED  [x] OVERRIDDEN  [ ] UNRESOLVED     │
│ Justification: Indian formulary constraint (₹200-500/month)    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Post-Extraction: KB Schema Changes Required

1. **`l2_merged_spans`** — Add columns:
   - `conflict_group_id VARCHAR(30)` — links overlapping spans across guidelines
   - `override_type VARCHAR(20)` — NULL/ADDITIVE/MODIFIED/CONDITIONAL/SUBSTITUTION/RESTRICTED
   - `tier_level SMALLINT DEFAULT 1` — 1=universal, 2=country-override
   - `country_code VARCHAR(2)` — ISO country code for Tier 2 rules

2. **`l2_extraction_jobs`** — Add column:
   - `guideline_tier SMALLINT DEFAULT 1` — distinguishes universal vs override jobs

3. **New table: `l2_conflict_groups`**:
   ```sql
   CREATE TABLE l2_conflict_groups (
     conflict_group_id VARCHAR(30) PRIMARY KEY,
     domain VARCHAR(30) NOT NULL,
     resolution_state VARCHAR(20) NOT NULL DEFAULT 'PENDING'
       CHECK (resolution_state IN ('PENDING', 'HARMONIZED', 'OVERRIDDEN', 'CONDITIONAL', 'RESTRICTED', 'UNRESOLVED')),
     resolution_justification TEXT,
     resolved_by VARCHAR(100),
     resolved_at TIMESTAMPTZ,
     winning_span_id UUID REFERENCES l2_merged_spans(span_id),
     source_lag BOOLEAN DEFAULT FALSE,
     source_lag_note TEXT
   );
   ```

This schema extension supports the 3-tier resolution architecture without requiring changes to the existing KDIGO/ADA extraction data.
