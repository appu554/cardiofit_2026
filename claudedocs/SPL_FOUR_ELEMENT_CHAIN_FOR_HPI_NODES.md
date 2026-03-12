# SPL Four-Element Chain for HPI Node Context Modifiers

**Version**: 2.0 (Codebase-Verified)
**Date**: 2026-03-10
**Status**: Cross-checked against CardioFit codebase — 5 corrections applied from v1.0

---

## Executive Summary

The four-element chain your Context Modifier layer needs is:

```
drug class --> mechanism of symptom --> onset window --> context modifier rule
```

This is the hardest data structure in the entire product to populate because no single source contains all four elements for any drug-symptom pair. However, the **storage, merge, and consumption infrastructure is already built**. What remains is the systematic population workflow.

### Architecture Reality (from code, not design docs)

The KB-20 `AdverseReactionProfile` struct already stores ALL four elements in a single record:

```go
// File: kb-20-patient-profile/internal/models/adr_profile.go
type AdverseReactionProfile struct {
    // Element 1: Drug --> Symptom Association
    DrugClass       string  // e.g., "BETA_BLOCKER"
    Reaction        string  // e.g., "dizziness"
    ReactionSNOMED  string  // e.g., "10013573" (MedDRA PT)

    // Element 2: Mechanism of Symptom
    Mechanism       string  // e.g., "AT1 receptor blockade --> vasodilation --> postural BP drop"

    // Element 3: Onset Window
    OnsetWindow     string  // e.g., "1-14 days"
    OnsetCategory   string  // IMMEDIATE | ACUTE | SUBACUTE | CHRONIC | DELAYED

    // Element 4: Context Modifier Rule
    ContextModifierRule string  // JSONB: {"cm_id": "CM01", "effect": "INCREASE_PRIOR", ...}

    // Quality gate
    Source            string           // PIPELINE | SPL | MANUAL_CURATED
    Confidence        decimal.Decimal  // [0.0, 1.0]
    CompletenessGrade string           // FULL | PARTIAL | STUB (auto-computed)
}
```

**Completeness grading** (line 78-95 of `adr_profile.go`):
- **FULL**: drug+reaction + onset + mechanism + CM_rule + confidence >= 0.70 --> 1.0x magnitude in Bayesian engine
- **PARTIAL**: drug+reaction + (onset OR mechanism) --> 0.7x magnitude (dampened)
- **STUB**: minimal data only --> 0.0x magnitude (excluded from clinical use)

---

## Element 1: Drug Class --> Symptom Association (the "what")

### What SPL Provides

This is the only element that SPL provides reliably at scale. The DailyMed label lists adverse reactions under LOINC section `34084-4`.

**Codebase implementation**: The SPL pipeline CLI (`shared/cmd/spl-pipeline/main.go`) executes a 9-phase pipeline:

```
Phase A: Verify Spine (drug_master, source_documents, source_sections)
Phase B: Select Scope (target drugs)
Phase C: SPL Acquisition (fetch from DailyMed free API)
Phase D: LOINC Section Routing
Phase E-F: Table Extraction & Rule Generation
Phase G: DraftFact Creation (6 canonical fact types)
Phase H: Governance Handoff (KB-0)
Phase I: KB Projection (to KB-1, KB-4, KB-5, KB-6, KB-16)
```

The LOINC parser (`shared/extraction/spl/loinc_parser.go`) routes sections:

| LOINC Code | Section | Target KBs | Extraction |
|------------|---------|------------|------------|
| 34066-1 | Boxed Warning | KB-4 | SAFETY_SIGNAL fact |
| 34068-7 | Dosage & Administration | KB-1, KB-6, KB-16 | GFR thresholds, doses |
| 34070-3 | Contraindications | KB-4, KB-5 | Safety blocks |
| 34073-7 | Drug Interactions | KB-5 | DDI pairs |
| 34077-8 | Pregnancy | KB-4 | Reproductive safety |
| 34084-4 | Adverse Reactions | KB-4 | **SAFETY_SIGNAL with MedDRA PT** |
| 34090-1 | Clinical Pharmacology | KB-1 | **PK params (half-life, Tmax)** |
| 43685-7 | Warnings | KB-4 | Safety alerts |

The `SafetySignalContent` struct (in `shared/factstore/models.go`) captures:

```go
type SafetySignalContent struct {
    WarningType    string  // BOXED_WARNING, CONTRAINDICATION, ADVERSE_REACTION
    MedDRAPT       string  // Preferred Term code (e.g., "10013573" for dizziness)
    MedDRASOC      string  // System Organ Class (e.g., "Nervous system disorders")
    ClinicalEffect string  // Patient-level outcome
    TimeToOnset    string  // Onset window (when available in SPL)
    FrequencyBand  string  // Very Common (>10%), Common (1-10%), Uncommon, Rare, Very Rare
    Severity       string  // mild, moderate, severe, life-threatening
}
```

### What This Gives KB-22

For telmisartan, SPL extraction produces:
```json
{
    "fact_type": "SAFETY_SIGNAL",
    "drug": "telmisartan",
    "rxcui": "73160",
    "content": {
        "warning_type": "ADVERSE_REACTION",
        "meddra_pt": "10013573",
        "clinical_effect": "dizziness",
        "frequency_band": "common (1-10%)",
        "time_to_onset": ""
    }
}
```

SPL tells you telmisartan is associated with dizziness. It does NOT tell you how it causes dizziness, when the dizziness appears after initiation, or what clinical context amplifies the risk.

### What SPL Does NOT Capture (per drug class in DM+HTN cohort)

For the controlled-trial AE frequency tables, the pipeline extracts comparative incidence (drug arm vs placebo arm). For telmisartan specifically, dizziness appeared in the postmarketing section as one of the most frequently reported spontaneous events. In the controlled trials, dizziness occurred at >= 1% but was at least as frequent in the placebo group -- meaning the signal strength is low for telmisartan alone, but high when combined with volume depletion context.

---

## Element 2: Mechanism of Symptom (the "why")

### What SPL Does NOT Provide

This element transforms a flat drug-symptom association into a clinically actionable Context Modifier. The mechanism chain for telmisartan -- "first-dose hypotension; vasodilation-induced postural drop; higher risk if volume-depleted" -- does NOT appear anywhere in the SPL.

### Where Mechanism Data Lives (by drug class)

**ARBs (telmisartan, losartan, olmesartan)**:
- Mechanism: AT1 receptor blockade --> vasodilation --> reduced SVR --> postural BP drop
- Critical nuance: "First-dose phenomenon" -- acute vasodilatory effect strongest when RAAS already activated (volume-depleted patients, concurrent diuretics)
- Source: StatPearls ARB monograph (NCBI/NLM-hosted, peer-reviewed, freely accessible)
- NOT in SPL adverse reactions section

**SGLT2 inhibitors (dapagliflozin, empagliflozin)**:
- Mechanism: Osmotic diuresis + natriuresis --> intravascular volume contraction --> orthostatic hypotension
- Critical amplifier: Concurrent diuretic use -- volume depletion risk 2.2-2.7% with diuretics vs 0.9-1.0% without (CANVAS trial subgroup analysis)
- Source: Renal Fellow Network review, PMC RCT analyses, Cleveland Clinic Journal of Medicine

**Beta blockers (metoprolol, atenolol, carvedilol)**:
- Mechanism (DUAL):
  1. Direct: Bradycardia --> reduced cardiac output --> reduced cerebral perfusion
  2. Indirect (DANGEROUS): Masks hypoglycemia warning signs (tachycardia, tremor) because beta-1 blockade suppresses adrenergic responses -- while sweating is preserved (cholinergic pathway)
- Critical insight: Non-selective beta-blockers (propranolol) carry significantly higher masking risk than cardioselective agents (metoprolol)
- Source: AHA Hypertension journal, StatPearls Beta Blockers, Drooracle pharmacovigilance
- Clinical impact on KB-22: When Bayesian engine evaluates DQ01 (autonomic hypoglycemia cluster: tremor + sweating + hunger), it MUST reduce discriminating power of tremor/tachycardia in beta-blocked patients. LR for that question cluster drops from ~4.0 to ~2.0.

**Sulfonylureas (glimepiride, gliclazide)**:
- Mechanism: Beta cell insulin secretion stimulation --> excessive insulin --> glucose < 70 mg/dL --> autonomic symptoms including dizziness
- Onset tied to PK profile: glimepiride peak action 2-3 hours, duration 24 hours
- Amplifiers: Meal skipping, renal impairment (reduced clearance), concurrent ARB (increased insulin sensitivity)

### Verified Source Hierarchy

| Source | Coverage | Density | Access |
|--------|----------|---------|--------|
| StatPearls monographs | Mechanism + AE linkage per drug class | Highest single-source density | Free, NCBI/NLM-hosted |
| PMC systematic reviews | Quantified risk amplification (RCT subgroups) | Trial-specific incidence data | Free, PubMed Central |
| Clinical review journals (CCJM, NKF, AAFP) | Synthesised management perspective | Clinical action guidance | Mixed access |
| Pharmacovigilance databases (FAERS, Drooracle) | Post-market signal strength | Population-level frequency | Free/mixed |

### How Mechanism Data Reaches KB-20

The ADR service merge strategy (`kb-20-patient-profile/internal/services/adr_service.go`, lines 55-95) handles this:

```go
// Dual-path merge:
// If existing=SPL and incoming=PIPELINE --> keep SPL's mechanism+onset, upgrade with PIPELINE's CM rule
// If existing=PIPELINE and incoming=SPL --> keep PIPELINE's CM rule, upgrade with SPL's mechanism+onset
// Always upgrade empty fields from either source
```

**Practical workflow**:
1. SPL pipeline creates STUB record: `{drug_class: "ARB", reaction: "dizziness", source: "SPL", mechanism: "", completeness: "STUB"}`
2. Clinician evidence harvest upgrades: `{mechanism: "AT1 blockade --> vasodilation --> postural drop", source: "MANUAL_CURATED"}`
3. Merge strategy retains SPL's frequency data + clinician's mechanism --> upgrades to PARTIAL
4. Pipeline later adds CM rule --> upgrades to FULL

---

## Element 3: Onset Window (the "when")

### What SPL Provides (MORE than the v1.0 document claimed)

**Correction from v1.0**: The SPL pipeline already extracts PK parameters that enable onset window derivation.

The `TabularHarvester` (`shared/extraction/spl/tabular_harvester.go`) classifies `TableTypePKParams` tables containing half-life, Tmax, Cmax, and AUC data from LOINC section `34090-1` (Clinical Pharmacology). This is routed to KB-1.

```go
// tabular_harvester.go, line 88
TableTypePKParams  TableType = "PK_PARAMETERS"  // half-life, clearance, bioavailability, AUC, Cmax
```

**Onset window derivation from PK data**:
- Telmisartan half-life: ~24 hours --> steady state at 5-7 days (5x half-life)
- Maximum vasodilatory effect expected within first week
- Onset window = "Days 1-14 after initiation or dose increase"

This means the SPL pipeline provides a **partial Element 3** that's already extractable:

| PK Parameter | SPL Section | Derivation Rule | Onset Window |
|-------------|-------------|-----------------|--------------|
| Half-life | 34090-1 | 5x half-life = time-to-steady-state | Onset ceiling |
| Tmax | 34090-1 | Peak plasma = peak acute effect | Acute onset |
| AUC ratio (drug/placebo) | 34084-4 | Higher ratio = stronger signal | Signal strength |

### What SPL Does NOT Provide

Explicit onset window statements. The telmisartan SPL says dizziness occurs -- it doesn't say "typically appears within 1-14 days of initiation." The temporal data comes from:

**Post-marketing timing data (sparse)**:
- SGLT2i: >50% of ~100 renal failure cases reported onset within 1 month of starting (PMC)
- SGLT2i volume depletion: Renal compensation rapid and nearly complete within days/weeks (AHA Circulation)
- ARB first-dose phenomenon: Days 1-3 peak risk (derived from PK + clinical pharmacology)

**Expert consensus / clinical experience**:
- Beta-blocker masking of hypoglycemia: Onset essentially immediate upon therapeutic plasma levels. Danger persists indefinitely.
- Close blood glucose monitoring recommended for first 3-4 weeks after beta-blocker initiation (Drooracle)

### OnsetCategory Enum (from KB-20 schema)

```sql
-- kb-20-patient-profile/migrations/001_initial_schema.sql
onset_category VARCHAR(20) CHECK (onset_category IN (
    'IMMEDIATE',  -- Minutes (anaphylaxis, acute hypotension)
    'ACUTE',      -- Hours to days (first-dose phenomenon, acute volume depletion)
    'SUBACUTE',   -- Days to weeks (steady-state ADRs, metabolic effects)
    'CHRONIC',    -- Weeks to months (cumulative toxicity, metabolic changes)
    'DELAYED'     -- Months to years (organ damage, carcinogenicity)
))
```

### Onset Window Population Strategy

| Drug Class | Onset Window | Category | PK-Derivable? | Source |
|-----------|-------------|----------|---------------|--------|
| ARB | Days 1-14 | ACUTE | YES (half-life 24h, steady state ~7d) | SPL PK + StatPearls |
| SGLT2i | Days 1-28 | SUBACUTE | PARTIAL (PK + RCT timing data) | CANVAS/EMPA-REG + PMC |
| Beta-blocker (masking) | Immediate, indefinite | IMMEDIATE | YES (therapeutic levels = masking) | AHA + Drooracle |
| Sulfonylurea (hypo) | Hours 2-24 | ACUTE | YES (Tmax = peak hypoglycemia risk) | SPL PK section |
| CCB | Days 1-7 | ACUTE | YES (half-life varies: amlodipine ~40h) | SPL PK |
| Thiazide (electrolyte) | Days 7-28 | SUBACUTE | PARTIAL (cumulative effect) | KDIGO + StatPearls |
| Insulin (hypo) | Minutes 15-120 | IMMEDIATE | YES (Tmax = onset of action) | SPL PK section |

---

## Element 4: Context Modifier Rule (the "so what")

### Architecture (from code)

Context Modifier rules live in two places:

1. **KB-20 `context_modifiers` table** -- structured rules with drug_class_trigger, effect, magnitude, target_differential
2. **KB-20 `adverse_reaction_profiles` table** -- `context_modifier_rule` JSONB field linking ADR to CM

The CM Applicator in KB-22 (`kb-22-hpi-engine/internal/services/cm_applicator.go`) converts CMs to log-odds deltas:

```go
// F-01: Log-odds conversion
// INCREASE_PRIOR: delta = log(1 + adj_magnitude)
// DECREASE_PRIOR: delta = log(1 - adj_magnitude)
//
// F-03: Adherence scaling
// adj_magnitude = magnitude * min(1.0, adherence_score / 0.70)
```

### Evidence Chain for Three Critical CMs

**CM01: ARB + recent initiation --> increase OH probability**

| Element | Value | Source |
|---------|-------|--------|
| 1. Drug-->Symptom | telmisartan --> dizziness | SPL LOINC 34084-4 (MedDRA PT 10013573) |
| 2. Mechanism | First-dose vasodilation, amplified by volume depletion | StatPearls ARBs |
| 3. Onset Window | Days 1-14 (OnsetCategory: ACUTE) | SPL PK (half-life 24h --> 5-7d steady state) |
| 4. CM Rule | `IF med_class='ARB' AND days_since_start<30 THEN increase_prob('OH', +0.20)` | Calibration: 25.5% OH prevalence (South India DM study) + ~2x risk amplification from recent initiation |

```go
// In KB-22 Bayesian engine:
// CM01 applied: delta = log(1 + 0.20) = +0.182
// log_odds["Orthostatic Hypotension"] += 0.182
```

**CM02: SGLT2i + concurrent diuretic --> increase volume depletion probability**

| Element | Value | Source |
|---------|-------|--------|
| 1. Drug-->Symptom | empagliflozin --> orthostatic hypotension | SPL LOINC 34084-4 |
| 2. Mechanism | Osmotic diuresis + natriuresis --> volume contraction | StatPearls SGLT2i + Renal Fellow Network |
| 3. Onset Window | Days 1-28 (OnsetCategory: SUBACUTE) | CANVAS trial timing + PMC practical approach |
| 4. CM Rule | `IF med_class='SGLT2I' AND concurrent_diuretic=true THEN increase_prob('Volume Depletion', +0.25)` | CANVAS/EMPA-REG: 2.2-2.7% with diuretics vs 0.9-1.0% without (~2.5x RR) |

**CM04: Beta-blocker + SU/Insulin --> flag masked hypoglycemia**

| Element | Value | Source |
|---------|-------|--------|
| 1. Drug-->Symptom | metoprolol --> dizziness | SPL LOINC 34084-4 |
| 2. Mechanism | Beta-1 blockade suppresses tremor + tachycardia (warning signs); sweating preserved (cholinergic) | AHA Hypertension, StatPearls BB |
| 3. Onset Window | IMMEDIATE (upon therapeutic levels), indefinite | Drooracle: monitor glucose first 3-4 weeks |
| 4. CM Rule | `IF med_class='BB' AND (concurrent='SU' OR concurrent='Insulin') THEN flag_masked_hypo=true; adjust symptom_weights` | Clinical inference: LR drops from ~4.0 to ~2.0 for autonomic cluster |

The "adjust symptom weights" is operationalised in the beta-blocker modifier YAML:
```yaml
# kb-22-hpi-engine/nodes/beta_blocker_hypo_modifier.yaml
modifications:
  - symptom: palpitations
    weight_multiplier: 0.05    # Beta-blockers abolish tachycardia
  - symptom: tremor
    weight_multiplier: 0.10    # Marked reduction
  - symptom: diaphoresis
    weight_multiplier: 2.5     # Cholinergic, preserved -- AMPLIFIED as discriminator
  - symptom: cognitive_symptoms
    weight_multiplier: 3.0     # Neuroglycopaenic, unaffected
  - symptom: hunger
    weight_multiplier: 2.0     # Hypothalamic, preserved
```

---

## The Population Workflow: How to Fill All Four Elements

### Current State (what's built)

```
SPL Pipeline (Phase A-I)
    |
    v
FactStore (6 canonical types)
    |                              Clinician Evidence Harvest
    |                                     |
    v                                     v
KB-20 ADRService.Upsert()  <--- dual-path merge strategy
    |
    v
AdverseReactionProfile {
    Element 1: from SPL (drug_class + reaction + MedDRA PT)     [AUTOMATED]
    Element 2: from MANUAL_CURATED or SPL Clinical Pharm        [SEMI-AUTOMATED]
    Element 3: from SPL PK tables + literature                  [SEMI-AUTOMATED]
    Element 4: from PIPELINE or MANUAL_CURATED                  [MANUAL]
    Completeness: auto-computed on every write                   [AUTOMATED]
}
    |
    v
KB-22 SessionContextProvider (4 parallel goroutines, 40ms timeout)
    |
    v
Bayesian Engine (F-01 log-odds delta, F-03 adherence scaling)
```

### Source Field Tri-State (controls merge priority)

| Source Value | Who Writes | What It Provides | Merge Behavior |
|-------------|-----------|------------------|----------------|
| `SPL` | SPLGuard pipeline | Element 1 (drug-->symptom), partial Element 3 (PK params) | Wins for mechanism + onset over empty fields |
| `PIPELINE` | Guideline extraction pipeline | Element 4 (CM rules from Channel G sentence extraction) | Wins for context_modifier_rule over empty fields |
| `MANUAL_CURATED` | Clinician evidence harvest | Elements 2+3+4 (mechanism, onset windows, CM calibration) | Highest priority for all fields |

### Step-by-Step Population Protocol (per drug-symptom pair)

**Step 1: SPLGuard creates STUB record**
```
Run: go run shared/cmd/spl-pipeline/main.go --drug telmisartan
Result: SAFETY_SIGNAL fact with MedDRA PT, frequency band
ADR Profile: {drug_class: "ARB", reaction: "dizziness", source: "SPL", completeness: "STUB"}
```

**Step 2: Derive onset window from PK data (semi-automated)**
```
SPL Clinical Pharmacology (LOINC 34090-1) --> TabularHarvester --> PK_PARAMETERS table
Extract: half-life = 24h, Tmax = 0.5-1h
Derive: onset_window = "1-14 days", onset_category = "ACUTE"
ADR Profile upgrades to: {completeness: "PARTIAL"}
```

**Step 3: Clinician harvests mechanism from StatPearls (manual)**
```
Source: StatPearls ARB monograph
Harvest: "AT1 receptor blockade --> vasodilation --> reduced SVR --> postural BP drop.
          First-dose phenomenon amplified by volume depletion."
Write: mechanism field via KB-20 API POST /api/v1/pipeline/adr-profiles
ADR Profile: {completeness: "PARTIAL"} (still needs CM rule)
```

**Step 4: Author CM rule and calibrate magnitude (manual)**
```
Source: Clinical reasoning from Elements 1-3 + epidemiological data
Author: context_modifier_rule = {"cm_id": "CM01", "effect": "INCREASE_PRIOR",
         "target_differential": "OH", "magnitude": 0.20,
         "condition": "days_since_start < 30"}
Write: context_modifier_rule field
ADR Profile upgrades to: {completeness: "FULL"} (all 4 elements + confidence >= 0.70)
```

### Automation Opportunities (reducing manual effort)

| Step | Current | Automatable? | How |
|------|---------|-------------|-----|
| 1. Drug-->Symptom | Automated (SPLGuard) | Already done | Phase G DraftFact creation |
| 2. Mechanism | Manual | PARTIALLY -- StatPearls has structured "Mechanism of Action" sections | LLM extraction from StatPearls HTML with schema-guided prompting |
| 3. Onset Window | Semi-automated | YES -- PK-derivable for most drug classes | Codify the "5x half-life" rule as a deterministic transform in Phase G |
| 4. CM Rule | Manual | NO -- requires clinical judgment for magnitude calibration | Clinician-authored, Canon Framework validated |

### Priority: 5x Half-Life Automation

The single highest-value automation is codifying onset window derivation from PK data:

```go
// Proposed addition to loinc_parser.go or Phase G
func DeriveOnsetWindow(pkParams PKParameters) (string, string) {
    if pkParams.HalfLife > 0 {
        steadyStateDays := (pkParams.HalfLife * 5) / 24.0  // hours to days
        if steadyStateDays <= 1 {
            return "Hours to 1 day", "IMMEDIATE"
        } else if steadyStateDays <= 7 {
            return fmt.Sprintf("Days 1-%d", int(steadyStateDays*2)), "ACUTE"
        } else if steadyStateDays <= 28 {
            return fmt.Sprintf("Days 1-%d", int(steadyStateDays*2)), "SUBACUTE"
        }
        return fmt.Sprintf("Weeks 1-%d", int(steadyStateDays/7)), "CHRONIC"
    }
    return "", ""
}
```

This would automatically populate Element 3 for every drug where the SPL contains PK tables -- upgrading STUB records to PARTIAL without any manual intervention.

---

## Comprehensive Four-Element Source Map

### Corrected from v1.0 (with codebase verification)

| Element | SPL Provides? | Primary Verified Source | Extraction Method | KB-20 Field | Automatable? |
|---------|--------------|----------------------|-------------------|-------------|-------------|
| 1. Drug-->Symptom | YES (AE tables, LOINC 34084-4) | DailyMed SPL XML | SPLGuard pipeline Phase G | `drug_class`, `reaction`, `reaction_snomed` | YES (existing) |
| 2. Mechanism | NO (prose only in Clinical Pharm) | StatPearls monographs per drug class | Manual harvest + LLM-assisted extraction | `mechanism` | PARTIALLY (LLM from StatPearls) |
| 3. Onset Window | PARTIAL (PK data in 34090-1) | SPL PK section + clinical literature | TabularHarvester PK_PARAMS + 5x half-life derivation | `onset_window`, `onset_category` | YES (PK derivation automatable) |
| 4. CM Rule | NO | Synthesised from Elements 1-3 by clinician | Clinician-authored, Canon Framework validated | `context_modifier_rule` (JSONB) | NO (clinical judgment required) |

### Data Flow Through the System

```
DailyMed API (free, no key)
    |
    v
SPL Pipeline (9 phases)
    |
    +---> Phase D: LOINC 34084-4 (AE) ---> SAFETY_SIGNAL fact ---> Element 1
    +---> Phase D: LOINC 34090-1 (PK) ---> PK_PARAMETERS table --> Element 3 (derived)
    +---> Phase E-F: Table extraction ---> GFR/hepatic thresholds --> KB-1
    |
    v
FactStore (KB-0 canonical_facts)
    |
    v
KB Projection (Phase I)
    |
    +---> KB-4 (SAFETY_SIGNAL, REPRODUCTIVE_SAFETY, BOXED_WARNING)
    +---> KB-5 (INTERACTION)
    +---> KB-1 (ORGAN_IMPAIRMENT)
    +---> KB-16 (LAB_REFERENCE)
    |
    v
KB-20 ADRService.Upsert() [dual-path merge]
    |
    v
AdverseReactionProfile (4 elements, completeness graded)
    |
    v
KB-22 SessionContextProvider.Fetch()
    |
    +---> goroutine 1: KB-20 /api/v1/patient/{id}/stratum/{node_id}  [REQUIRED, 40ms]
    |     Returns: stratum_label, ckd_substage, active_modifiers[], safety_overrides[]
    |
    +---> goroutine 2: KB-21 adherence-weights                        [optional, 40ms]
    +---> goroutine 3: KB-21 answer-reliability                       [optional, 40ms]
    +---> goroutine 4: KB-23 treatment-perturbations                  [optional, 40ms]
    |
    v
SessionContext --> BayesianEngine
    |
    +---> InitPriors(node, stratum)  [stratum-specific priors from YAML]
    +---> ApplyGuidelineAdjustments() [N-01: KB-1 prior injection]
    +---> CMApplicator.Apply()       [F-01: log-odds delta, F-03: adherence scaling]
    +---> Update(answer)             [R-02: cluster dampening, R-03: reliability]
    +---> SafetyEngine.CheckTriggers() [F-02: cross-node safety]
    |
    v
Posterior probabilities --> KB-23 Decision Cards --> KB-19 Commit
```

---

## Node Update Checklist: What Each HPI Node Needs

For each HPI Presentation Node (P01 Chest Pain, P02 Dyspnea, P08 Dizziness, etc.), the following ADR profiles must be populated:

### Per Drug Class in DM+HTN Cohort

| Drug Class | Key ADR for HPI | Elements Needed | Population Priority |
|-----------|----------------|-----------------|-------------------|
| ARB | Orthostatic hypotension, hyperkalemia, dizziness | All 4 | HIGH -- CM01 (recent initiation amplifier) |
| SGLT2i | Volume depletion, orthostatic hypotension, DKA | All 4 | HIGH -- CM02 (concurrent diuretic amplifier) |
| Beta-blocker | Masked hypoglycemia, bradycardia, fatigue | All 4 | CRITICAL -- CM04 (symptom weight modification) |
| Sulfonylurea | Hypoglycemia | Elements 1-3 (CM via CM04 interaction) | MEDIUM |
| CCB | Peripheral edema, dizziness, flushing | Elements 1-3 | MEDIUM |
| Thiazide | Electrolyte imbalance, orthostatic hypotension | All 4 | HIGH -- CM with CKD amplification |
| Metformin | Lactic acidosis (rare), GI effects | Elements 1-3 | LOW (rare, well-known) |
| Insulin | Hypoglycemia | Elements 1-3 + CM04 interaction | HIGH |

### Steps to Update a Node

1. **Run SPLGuard** for all drugs relevant to the node's differential set
   ```bash
   go run shared/cmd/spl-pipeline/main.go --drugs-file node_p01_drugs.csv
   ```

2. **Verify Element 1 extraction** -- check FactStore for SAFETY_SIGNAL facts with MedDRA PTs matching the node's differential symptoms

3. **Derive Element 3** -- extract PK parameters from LOINC 34090-1, apply 5x half-life rule, populate onset_window and onset_category

4. **Harvest Element 2** -- for each drug-symptom pair, extract mechanism from StatPearls monograph and write to KB-20 via:
   ```bash
   POST /api/v1/pipeline/adr-profiles
   {
       "drug_class": "ARB",
       "reaction": "dizziness",
       "mechanism": "AT1 receptor blockade --> vasodilation --> postural BP drop",
       "source": "MANUAL_CURATED",
       "source_authority": "StatPearls",
       "source_document": "Angiotensin II Receptor Blockers"
   }
   ```

5. **Author Element 4** -- calibrate CM magnitude from epidemiological data and write context_modifier_rule JSONB

6. **Verify completeness** -- query KB-20 for all ADR profiles for the node's drug classes, confirm FULL/PARTIAL grading:
   ```bash
   GET /api/v1/modifiers/node/P01_CHEST_PAIN
   ```

7. **Test in KB-22** -- initialise an HPI session with a mock patient on the relevant medications, verify CM application in Bayesian engine log-odds

---

## Corrections Log (v1.0 --> v2.0)

| # | v1.0 Claim | Correction | Evidence |
|---|-----------|------------|----------|
| 1 | "SPL almost never provides onset window data" | SPL PK tables (LOINC 34090-1) already extracted by TabularHarvester as PK_PARAMETERS; onset derivable via 5x half-life | `tabular_harvester.go:88` TableTypePKParams |
| 2 | Four elements treated as independent silos | ADRService dual-path merge strategy already combines SPL + PIPELINE + MANUAL_CURATED in a single record | `adr_service.go:55-95` mergeProfiles() |
| 3 | "No single pipeline handles all four" | KB-20 AdverseReactionProfile stores all 4 elements; completeness grading gates consumption | `adr_profile.go:14-61` struct definition |
| 4 | Missing `source` tri-state provenance | Source field (PIPELINE/SPL/MANUAL_CURATED) controls merge priority; critical for audit trail | `adr_profile.go:43` Source field |
| 5 | Document focused on KB-5 TimeToOnset | KB-20 has BOTH OnsetWindow + OnsetCategory (IMMEDIATE/ACUTE/SUBACUTE/CHRONIC/DELAYED) -- this is what KB-22 queries | `adr_profile.go:29-30`, `001_initial_schema.sql:141-182` |
