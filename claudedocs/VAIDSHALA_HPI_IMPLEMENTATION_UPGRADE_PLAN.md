# Vaidshala HPI Implementation Upgrade Plan
## P0 Dizziness Pilot + P1 Chest Pain V2 + P2 Acute Dyspnea V1 + P3-P8 Gap Closure

**Date**: 2026-03-10 (Revised v3)
**Source Documents**: P1 Chest Pain V2, P2 Acute Dyspnea V1, P3-P7 Gap Analysis, P2 Adjusted Execution Priority, What P2 Gets Automatically, **Dizziness Evidence Harvest V2 (Feb 2026)**, **M2 Bayesian Engine Critical Review & Counter-Proposal (Feb 2026)**
**Scope**: KB-22 HPI Engine node upgrades, KB-20 stratum infrastructure, pipeline integration, cross-node safety, **M2 Bayesian engine structural fixes**, **Safety Constraint Engine (SCE)**, **noise resilience for Hindi ASR**

---

## 1. Engineering Infrastructure Inventory

This plan builds on verified engineering scaffolding. Every item below maps to running Go code.

### 1.1 What Already Works (YAML-Consumable, No Go Changes)

| Capability | Go Consumer | File | What YAML Controls |
|---|---|---|---|
| Per-stratum priors | `BayesianEngine.InitPriors()` | `bayesian_engine.go:34` | `differentials[].priors` map keyed by stratum label |
| Dual convergence (R-01) | `BayesianEngine.CheckConvergence()` | `bayesian_engine.go:230` | `convergence_threshold`, `posterior_gap_threshold`, `convergence_logic` |
| Cluster dampening (R-02) | `BayesianEngine.Update()` | `bayesian_engine.go:136-148` | `questions[].cluster`, `cluster_dampening` |
| Reliability weighting (R-03) | `BayesianEngine.Update()` | `bayesian_engine.go:163` | Fetched from KB-21 at runtime |
| PATA_NAHI handling (F-04) | `BayesianEngine.Update()` | `bayesian_engine.go:110` | Answer value "PATA_NAHI" = zero update |
| Safety triggers (BOOLEAN) | `SafetyEngine` + `NodeLoader.validate()` | `node_loader.go:222-236` | `safety_triggers[].condition`, `severity`, `recommended_action` |
| R-05 auto-injection | `NodeLoader.injectSafetyGuards()` | `node_loader.go:243-256` | Auto-computed from safety trigger conditions |
| Cross-node triggers (F-07) | `CrossNodeSafety.Load()` | `cross_node_safety.go:48` | `cross_node_triggers.yaml` |
| Guideline adjustments (N-01) | `BayesianEngine.ApplyGuidelineAdjustments()` | `bayesian_engine.go:63` | Additive log-odds from KB-3 |
| LR source provenance (R-07) | `NodeLoader.validate()` warns | `node_loader.go:169,217` | `lr_source`, `lr_evidence_class`, `population_reference` |
| Branch conditions | `QuestionOrchestrator.EvaluateBranchCondition()` | `question_orchestrator.go:215` | `branch_condition` string per question |
| Entropy-based selection | `QuestionOrchestrator.ComputeExpectedIG()` | `question_orchestrator.go:168` | One-step lookahead, automatic |
| CM application (F-01/F-03) | `CMApplicator.Apply()` | `cm_applicator.go:51` | CMs from KB-20 registry at runtime |
| Adherence gain scaling | `computeAdherenceGain()` | `session_context_provider.go:389` | KB-21 tier: HIGH=1.0, MEDIUM=0.7, LOW=0.4 |

### 1.2 What Requires Go Code Changes

| # | Feature | Why Go Change | Estimated Scope | Required By |
|---|---|---|---|---|
| G1 | Safety floor clamping (per-diagnosis posterior floor) | No floor-clamping logic exists in `BayesianEngine.Update()` or `GetPosteriors()` | Add `safety_floors` YAML field to `NodeDefinition`, clamp posteriors after each update | P1 V2 (ACS floor 0.05), P2 V1 (ACS 0.05, PE 0.03, ADHF stratum-specific) |
| G2 | Sex-modifier prior adjustment (SM01-SM03) | Sex modifiers are OR-based prior multipliers in log-odds domain, not LR questions | Add `sex_modifiers` YAML field, `ApplySexModifiers()` in BayesianEngine after InitPriors | P1 V2, P2 V1 |
| G3 | Medication-conditional differentials (DX09, DX10) | Engine must conditionally include/exclude differentials based on active medication list from KB-20 | Add `activation_condition` field to `DifferentialDef`, conditional inclusion in InitPriors | P1 V2 (DX09 SGLT2i, DX10 Metformin), P2 V1 (DX09 ACEi, DX10 SGLT2i) |
| G4 | DM_HTN_CKD_HF stratum | `StratumEngine.determineStratum()` only emits DM_ONLY/DM_HTN/DM_HTN_CKD/HTN_ONLY | Add HF detection in `determineStratum()`, new stratum constant | P2 V1 (4-column priors) |
| G5 | HARD_BLOCK and OVERRIDE CM effect types | `CMApplicator.Apply()` only handles INCREASE_PRIOR / DECREASE_PRIOR | Add HARD_BLOCK (treatment safety gate) and OVERRIDE (force posterior) to Apply() | P1 V2 (CM06 PDE5i), P2 V1 (CM08 PDE5i, CM10 Hb<8) |
| G6 | Stratum-conditional LR values | `QuestionDef.LRPositive` is `map[string]float64` keyed by differential ID, not stratum | Add `lr_positive_by_stratum` / `lr_negative_by_stratum` optional override maps | P2 V1 (DQ01 orthopnea LR+ drops from 2.2 to 1.2 in CKD+HF) |
| G7 | Acuity/context question layer | No acuity scoring track separate from Bayesian differential | New `AcuityEngine` service or acuity tags on questions with parallel scoring | P1 V2 (AC01-AC06), P2 V1 (AC01-AC06) |
| G8 | CM active state in safety trigger conditions | `SafetyEngine` evaluates only question answers, not active CM state | Add CM active flags to condition evaluation context | P2 V1 (ST004 references CM05_ACTIVE) |
| G9 | Conditional priors (P05-style `bp_status == SEVERE` overrides) | NodeDefinition needs conditional prior fields | New struct field + InitPriors logic branch | P5+ (deferred) |
| G10 | CATEGORICAL answer types (beyond YES/NO/PATA_NAHI) | `BayesianEngine.Update()` hard-coded to YES/NO | Add categorical LR lookup map | P7+ (deferred) |
| G11 | Action thresholds (posterior > X triggers recommendation) | No action trigger framework | New ActionEngine service | Deferred — KB-23 handles downstream |
| G12 | COMPOSITE_SCORE safety triggers (R-06 stub) | NodeLoader rejects COMPOSITE_SCORE | Remove guard, implement weighted scoring | Deferred |
| G13 | Node transition protocol | No node transition logic exists | Add `NodeTransition` struct, transition evaluation in SessionService | Deferred |
| G14 | CM composition in log-odds space (BAY-1) | `CMApplicator.Apply()` adds deltas in probability space. Breaks when ≥2 CMs fire on same differential (common in DM+HTN polypharmacy). | Convert author deltas to log-odds shifts: `delta_logodds = logit(0.50 + delta_p) - logit(0.50)`. Apply additively: `logodds_adjusted = logodds_base + Σ(delta_logodds)`. See Section 11.1 for worked example. | P1 V2 (CM01+CM09 both target OH) |
| G15 | 'Other' bucket differential (BAY-2) | No escape hatch for out-of-differential diagnoses. All node priors sum to 1.00 with no residual. | Add implicit 'Other' differential (p=0.15) updating via geometric mean of inverse LRs. If Other > 0.30 → DIFFERENTIAL_INCOMPLETE flag. If Other > 0.45 → soft escalation (specialist referral). | ALL nodes |
| G16 | Pata-nahi cascade protocol (BAY-3) | `BayesianEngine.Update()` handles single PATA_NAHI (c=0) but no cascade tracking. | Track consecutive low-confidence answers (c<0.3). Count=2→rephrase via `alt_prompt` YAML field. Count=3→binary-only mode. Count≥5→PARTIAL_ASSESSMENT terminate. Count≥5 AND any RF among low-conf→ESCALATE immediately. Add `alt_prompt` field to QuestionDef YAML schema. | ALL nodes (critical for Hindi voice pilot) |
| G17 | Contradiction detection matrix (BAY-4) | No detection of contradictory patient answers across logically-related questions. | Add `contradiction_pairs` YAML field to NodeDefinition (e.g., AC02↔DQ03). When contradiction fires: (1) don't apply 2nd LR, (2) re-ask 1st question with confirmation framing, (3) apply confirmed answer's LR, discard contradicted, (4) log `contradiction_event`. | ALL nodes |
| G18 | Closure multi-criteria guard (BAY-13) | `CheckConvergence()` only checks posterior threshold + gap. No answer quality gate. | Closure requires: `top_p > threshold` AND `confidence of decisive answer > 0.75` AND `supporting_answers >= 2`. Prevents premature closure on single low-confidence answer. | ALL nodes |
| G19 | Skip-redundancy rule (BAY-8) | No mechanism to skip questions already answered by CM evaluation. | If a question's primary target differentials ALL have CM delta ≥ 0.30 already applied, skip the question. Shortens interactions without losing information. | ALL nodes |

### 1.3 KB-20 Infrastructure (Already Built)

| Component | File | Status |
|---|---|---|
| Stratum engine | `stratum_engine.go` | Running: DM_ONLY, DM_HTN, DM_HTN_CKD, HTN_ONLY |
| CM registry | `cm_registry.go` | Running: completeness-grade-aware (FULL 1.0x, PARTIAL 0.7x, STUB excluded) |
| ADR profile model | `adr_profile.go` | Running: stores all 4 elements of the drug-symptom chain (drug_class, mechanism, onset_window, context_modifier_rule). Auto-computes completeness grade on create/update via GORM hooks. |
| ADR merge strategy | `adr_service.go:55-95` | Running: dual-path merge -- SPL wins for mechanism+onset, PIPELINE wins for CM rules, MANUAL_CURATED highest priority. Recomputes completeness after merge. |
| Context modifier model | `context_modifier.go` | Running: GORM entity with confidence, source, lab threshold fields |
| eGFR trajectory | `egfr_engine.go` | Running: CKD staging, trajectory classification |

### 1.4 SPLGuard Pipeline Infrastructure (Already Built)

| Component | File | Status |
|---|---|---|
| SPL Pipeline CLI (9-phase) | `shared/cmd/spl-pipeline/main.go` | Running: Phases A-I from FactStore Runbook |
| LOINC Section Parser | `shared/extraction/spl/loinc_parser.go` | Running: routes 34066-1, 34068-7, 34070-3, 34073-7, 34084-4, 34090-1, 43685-7 to appropriate KBs |
| Tabular Harvester | `shared/extraction/spl/tabular_harvester.go` | Running: extracts GFR_DOSE, HEPATIC_DOSE, INTERACTION, ADVERSE_EVENT, **PK_PARAMETERS**, AGE_WEIGHT tables |
| DailyMed Fetcher (Go) | `kb-0-governance-platform/internal/api/dailymed_fetch.go` | Running: on-demand SPL XML fetch by SetID + LOINC section code |
| DailyMed Fetcher (Python) | `kb-5-drug-interactions/scripts/source_fetchers/dailymed_fetcher.py` | Running: search + section fetch via free DailyMed API |
| FactStore models | `shared/factstore/models.go` | Running: 6 canonical fact types (ORGAN_IMPAIRMENT, SAFETY_SIGNAL, REPRODUCTIVE_SAFETY, INTERACTION, FORMULARY, LAB_REFERENCE) with volatility classification + staleness tracking |
| SPL Review UI | `vaidshala/.../governance-dashboard/components/spl-review/` | Running: fact cards, triage dashboard, clinical sign-off workflow |

---

## 2. Three-Category Data Dependency Map

Every HPI node field falls into one of three categories based on where its data originates.

### Category A: Pipeline Supplies Today
Pipeline 1 extracts from KDIGO/ADA/RSSDI PDFs into KB-1 (dosing), KB-4 (safety), KB-16 (monitoring). KB-20's CM registry queries these at runtime. **No HPI node YAML change needed for Category A** -- the engine reads it live.

Examples already flowing:
- eGFR thresholds for metformin contraindication (KDIGO -> KB-1)
- SGLT2i sick day rules (KDIGO -> KB-4)
- Renal dosing adjustment thresholds (KDIGO -> KB-1)

When P2's CM06 fires ("eGFR < 45 -> CKD stratum activation"), the eGFR threshold itself comes from KDIGO extraction in KB-1.

### Category B: Pipeline Should Supply But Can't Yet (The Four-Element Chain)

The four-element chain that KB-20's `AdverseReactionProfile` model stores:

```
drug_class --> mechanism --> onset_window --> context_modifier_rule
```

**KB-20 already stores all four elements** in a single record (`adr_profile.go:14-61`), with auto-computed `CompletenessGrade`:

| Element | KB-20 Field | SPL Provides? | Guideline Pipeline Provides? | Manual Harvest? |
|---|---|---|---|---|
| 1. Drug-->Symptom | `drug_class`, `reaction`, `reaction_snomed` | YES (LOINC 34084-4 AE tables, MedDRA PT + frequency bands) | PARTIAL (Channel G sentence extraction) | NO -- automated |
| 2. Mechanism | `mechanism` | NO (prose only in Clinical Pharmacology) | NO | YES -- StatPearls monographs per drug class |
| 3. Onset Window | `onset_window`, `onset_category` (IMMEDIATE/ACUTE/SUBACUTE/CHRONIC/DELAYED) | PARTIAL (PK tables from LOINC 34090-1 contain half-life, Tmax -- onset derivable via 5x half-life rule) | NO | SEMI -- PK-derivable for most classes, literature for edge cases |
| 4. CM Rule | `context_modifier_rule` (JSONB) | NO | PARTIAL (eGFR thresholds from KDIGO) | YES -- clinician calibration of magnitude |

**Completeness grading** (auto-computed on every write via GORM hooks):
- **FULL**: drug+reaction + onset + mechanism + CM_rule + confidence >= 0.70 --> 1.0x magnitude in Bayesian engine
- **PARTIAL**: drug+reaction + (onset OR mechanism) --> 0.7x magnitude (dampened)
- **STUB**: minimal data only --> 0.0x magnitude (excluded from clinical use)

**Dual-path merge strategy** (`adr_service.go:55-95`): KB-20's `ADRService.Upsert()` implements source-aware merging:
- If existing=`SPL` and incoming=`PIPELINE` --> keep SPL's mechanism+onset, upgrade with PIPELINE's CM rule
- If existing=`PIPELINE` and incoming=`SPL` --> keep PIPELINE's CM rule, upgrade with SPL's mechanism+onset
- `MANUAL_CURATED` source has highest priority for all fields
- Always upgrades empty fields from either source; recomputes completeness after merge

**Source tri-state** controls merge priority and audit provenance:

| Source Value | Who Writes | What It Provides |
|---|---|---|
| `SPL` | SPLGuard pipeline (Phase G DraftFact creation) | Element 1 (drug-->symptom + MedDRA PT + frequency band), partial Element 3 (PK params) |
| `PIPELINE` | Guideline extraction pipeline (Channel G + L3 fact extraction) | Element 4 (CM rules from eGFR thresholds, contraindications) |
| `MANUAL_CURATED` | Clinician evidence harvest (StatPearls, PMC, clinical experience) | Elements 2+3+4 (mechanism, onset windows, CM calibration) |

**Blocking P2**: The breathlessness node needs to know that ACEi-induced cough occurs in 5-20% of patients, onset is weeks 1-12, mechanism is bradykinin accumulation. The SPLGuard pipeline can provide Element 1 (ACEi-->cough from LOINC 34084-4) and partial Element 3 (ACEi half-life from LOINC 34090-1). Elements 2 and 4 must be manually authored.

**Resolution**: Category B data gets populated through a **three-stage upgrade path**:
1. **Stage 1**: SPLGuard pipeline creates STUB records with drug-->symptom + MedDRA PT + frequency band (automated)
2. **Stage 2**: PK-derived onset windows auto-upgrade STUBs to PARTIAL (automatable -- see Fix 5 in Section 6)
3. **Stage 3**: Clinician evidence harvest adds mechanism + CM rule --> upgrades to FULL (manual, Canon Framework validated)

When the KB-24 extraction target is built (Fix 1 in Section 6), guideline-extracted data validates against SPL entries and the merge strategy combines the best data from both sources.

**KB-18 naming collision (RESOLVED)**: KB-18 is confirmed as the **Audit Trail & Compliance Registry** per both P1 and P2 source document footers ("KB-22 → KB-23 → KB-19 → KB-18 Audit"). The contextual modifier extraction target is hereby assigned **KB-24** (next unassigned slot). All references below use KB-24 for the ADR extraction target. The L3 template, pipeline integration lines (59, 852, 860), and all documentation now reference KB-24. **Note**: P2 source doc footer "KB-22 → KB-23 → KB-19 → KB-18 Audit" correctly refers to KB-18 as Audit; this does not conflict with our resolution.

### Category C: Never From Pipeline
Clinician-authored evidence harvest products:
- Base priors per stratum (from Jaipur Heart Watch, ICMR-INDIAB, Framingham, THFR, ASIAN-HF)
- Likelihood ratios (from McGee, JAMA Rational Clinical Exam, Fanaroff, Wang JAMA 2005)
- Sex-specific atypical presentation rates (from INTERHEART, Canto JAMA 2012)
- Stratum-conditional LR adjustments (e.g., orthopnea LR+ drops from 2.2 in DM_HTN_base to 1.2 in CKD+HF)
- Context modifier rules (CM01-CM10 per node)
- Safety floor posterior values

These go through: literature harvest -> Canon Framework review -> YAML commit.

---

## 3. Tiered Implementation Plan

### TIER 0: P00 Dizziness V2 — Pilot Node (Evidence Harvest Complete, YAML-Ready)
**Duration**: 1 week (YAML authoring + Go changes + validation) | **Risk**: Low | **Go changes**: G14 (log-odds CM composition), G15 (Other bucket), G16 (pata-nahi cascade) — these apply to ALL nodes but Dizziness is the first to exercise them. G16 is critical for Hindi voice pilot.
**Source**: *Dizziness Evidence Harvest V2 (Feb 2026)* — complete harvest-first output with provenance.

P00 Dizziness is the **prototype node** proving the harvest-first, build-second methodology. Unlike P1-P7 which require evidence harvest in parallel with Go changes, P00's evidence is **already harvested**. The Dizziness Evidence Harvest V2 document provides:

- **8 differentials** with Indian DM+HTN prevalence: OH (20-30%), Hypoglycemia (15-20%), Drug-Induced (10-15%), BPPV (10-15%), TIA (3-5%), Arrhythmia (3-5%), Anemia (5-8%), Volume Depletion (variable). **Note**: Ranges are from pre-G15 source doc. After G15 rebalancing, all priors scale down proportionally to sum to 0.85 (0.15 reserved for 'Other' bucket).
- **6 red flags** with SNOMED codes and Hindi prompts (focal neuro deficit, syncope, thunderclap headache, chest pain+dizziness, severe palpitations, glucose <54)
- **6 acuity questions** using AAFP 2023 **TiTrATE framework** (Timing/Triggers/Targeted Exam) — the structural reference for dizziness question sequencing. TiTrATE is a reusable acuity methodology applicable to multiple nodes.
- **8 discriminating questions** with harvested LRs: DQ01 autonomic cluster LR+ ~4.0, DQ02 Whipple confirmation LR+ ~6.0, DQ04 brief positional LR+ ~4.5, DQ07 continuous >24h triggers HINTS pathway (Kattah LR+ 25)
- **8 context modifiers** with epidemiological sources: CM01 ARB +0.20 (StatPearls), CM02 SGLT2i+diuretic +0.25 (NKF/CANVAS), CM03 SU/Insulin +0.30/+0.35 (ADA), CM04 BB masked hypo, CM05 HbA1c>9/DM>10y +0.15, CM06 eGFR<45 +0.15, CM07 age≥65+2 antihypertensives +0.20, CM08 recent med change +0.25
- **Hindi/Hinglish prompts** for every question — ready for Sarvam ASR
- **Pertinent negatives** with clinical reasoning
- **Completion criteria**: 6/6 RF screened, ≥4/6 acuity, ≥4/8 discriminating, max 12 questions, top posterior ≥0.60 OR top-2 sum ≥0.85

**Critical insight from harvest**: In the DM+HTN cohort, the top 3 dizziness differentials (OH, hypoglycemia, drug-induced) are ALL medication-related. The Context Modifier layer carries disproportionate diagnostic weight compared to general population — making G14 (log-odds CM composition) essential for this node.

**LR provenance gap**: Many dizziness LRs are marked CALIBRATE (consensus estimates). The Dizziness doc explicitly flags where formal LRs don't exist in published literature and recommends calibration after N=200. These feed into the three-tier calibration strategy (Section 11.5).

**P00 Dizziness CMs should be the first entries in the shared CM registry (B01)**: CM01-CM08 from the Dizziness doc overlap substantially with P1/P2 CMs (ARB, SGLT2i, BB, eGFR thresholds).

---

### TIER 1: P01 Chest Pain V2 Upgrade (YAML + Go Changes G1-G3)
**Duration**: 2 weeks | **Risk**: Medium | **Go changes**: G1 (safety floors), G2 (sex modifiers), G3 (medication-conditional differentials)

The current P01 YAML (`p01_chest_pain.yaml`) is V1-era with 7 differentials and Western-only priors. V2 spec requires a 10-differential closed set with Indian epidemiological priors summing to 1.00.

#### 1a. Replace Differential Set with V2 10-Diagnosis Closed Set

Current differentials: ACS, STABLE_ANGINA, GERD, MSK, PE, PERICARDITIS, ANXIETY (7 total).
V2 specifies 10 differentials with Indian DM+HTN base priors:

```yaml
differentials:
  - id: ACS
    label: "Acute Coronary Syndrome"
    population_reference: "Jaipur Heart Watch 2017; INTERHEART 2004 SA substudy"
    priors:
      DM_HTN_base: 0.18  # Indian DM has 2-4x CAD risk; Jaipur Heart Watch 18.2% in DM+HTN males
    # Source: Category C evidence harvest

  - id: GERD
    label: "GERD / Esophagitis"
    population_reference: "Indian J Gastroenterol 2019; ICMR dietary survey"
    priors:
      DM_HTN_base: 0.25  # Indian diet (spice-heavy, late dinners) drives 18-28% GERD prevalence

  - id: STABLE_ANGINA
    label: "Stable Angina / Chronic Coronary Syndrome"
    population_reference: "ACC/AHA 2021; India Heart Study 2018"
    priors:
      DM_HTN_base: 0.20

  - id: MSK
    label: "Musculoskeletal (Costochondritis)"
    population_reference: "McGee Ch. 21; Bruyninckx 2008"
    priors:
      DM_HTN_base: 0.18

  - id: ANXIETY
    label: "Anxiety / Panic Attack"
    population_reference: "ACC/AHA 2021"
    priors:
      DM_HTN_base: 0.08

  - id: PE
    label: "Pulmonary Embolism"
    population_reference: "ESC 2019 PE; Wells Score validation"
    priors:
      DM_HTN_base: 0.05
    acuity: CATASTROPHIC

  - id: AORTIC_DISSECTION
    label: "Aortic Dissection"
    population_reference: "ACC/AHA 2021; Hagan IRAD 2000"
    priors:
      DM_HTN_base: 0.02  # HTN is #1 risk factor
    acuity: CATASTROPHIC

  - id: PERICARDITIS
    label: "Pericarditis"
    population_reference: "McGee Ch. 21; ESC 2015 Pericarditis"
    priors:
      DM_HTN_base: 0.02

  - id: EUGLYCEMIC_DKA
    label: "Euglycemic DKA (SGLT2i-related)"
    population_reference: "StatPearls SGLT2i; ADA 2024"
    priors:
      DM_HTN_base: 0.01
    acuity: CATASTROPHIC
    activation_condition: "med_class == SGLT2i"  # GO CHANGE G3: medication-conditional

  - id: LACTIC_ACIDOSIS
    label: "Lactic Acidosis (Metformin-related)"
    population_reference: "StatPearls Metformin; KDIGO 2024"
    priors:
      DM_HTN_base: 0.01
    acuity: CATASTROPHIC
    activation_condition: "med_class == Metformin AND eGFR < 30"  # GO CHANGE G3
```

**PRIORS SUM TO 1.00**: 0.18+0.25+0.20+0.18+0.08+0.05+0.02+0.02+0.01+0.01 = 1.00.

> **⚠️ Pre-G15 YAML.** After G15 ('Other' bucket) implementation, reduce all priors proportionally so the set sums to **0.85**, leaving 0.15 for the implicit 'Other' differential. Example: ACS 0.18 → 0.153, GERD 0.25 → 0.2125, etc.

**V2 spec uses a single-column prior table** (Indian DM+HTN base). Multi-column strata (DM_ONLY, DM_HTN_CKD) are not specified in the V2 doc -- the existing 3-column priors in current YAML should be replaced with V2's single-column values. CKD/HF strata columns can be added later through evidence harvest; do NOT fabricate extrapolated values.

**DX09 and DX10 are medication-conditional**: When not activated, their prior (0.02 total) redistributes proportionally across DX01-DX08. This requires Go change G3 -- adding an `activation_condition` field to `DifferentialDef` and conditional inclusion logic in `InitPriors()`.

#### 1b. Red Flag Sweep (RF01-RF06 with SNOMED)

V2 specifies 6 red flags matching ACC/AHA 2021, ESC 2019, and IRAD registry:

```yaml
safety_triggers:
  - id: RF01
    type: BOOLEAN
    condition: "Q_RF01_TEARING=YES"
    severity: IMMEDIATE
    recommended_action: "Aortic dissection protocol. STAT CT angiography. SNOMED: 95436002"
    source: "ACC/AHA 2021; Hagan IRAD 2000"

  - id: RF02
    type: BOOLEAN
    condition: "Q_RF02_SUDDEN_DYSPNEA=YES"
    severity: IMMEDIATE
    recommended_action: "PE + ACS dual pathway. SNOMED: 267036007"
    source: "ESC 2019 PE; ACC/AHA 2021"

  - id: RF03
    type: BOOLEAN
    condition: "Q_RF03_SYNCOPE=YES"
    severity: IMMEDIATE
    recommended_action: "Cardiac syncope protocol. SNOMED: 271594007"
    source: "ACC/AHA 2021; IRAD"

  - id: RF04
    type: BOOLEAN
    condition: "Q_RF04_FOCAL_NEURO=YES"
    severity: IMMEDIATE
    recommended_action: "Stroke + dissection protocol. SNOMED: 373606000"
    source: "NICE Stroke; ACC/AHA 2021"

  - id: RF05
    type: BOOLEAN
    condition: "Q_RF05_GLUCOSE_LOW=YES"
    severity: IMMEDIATE
    recommended_action: "Level 2+ hypoglycemia + cardiac eval. SNOMED: 302866003"
    source: "ADA Standards 2024"

  - id: RF06
    type: BOOLEAN
    condition: "Q_RF06_HEMOPTYSIS=YES"
    severity: IMMEDIATE
    recommended_action: "PE + malignancy evaluation. SNOMED: 66857006"
    source: "ESC 2019 PE"
```

#### 1c. Acuity/Context Questions (AC01-AC06) -- GO CHANGE G7

V2 specifies an acuity layer between red flags and discriminating questions:

```yaml
acuity_questions:
  - id: AC01
    text_en: "When did the chest pain start? Sudden or gradual?"
    text_hi: "Yeh seene ka dard kab se hai? Achanak shuru hua ya dheere-dheere?"
    variable: onset_timing
    options: [sudden_minutes, hours, days, weeks_chronic]
    maps_to: "Acute (<48h) vs subacute vs chronic"
    source: "ACC/AHA 2021"

  - id: AC02
    text_en: "What does the pain feel like -- pressure, burning, or sharp?"
    text_hi: "Dard kaisa lag raha hai -- dabaav jaisa, jalan jaisi, ya chubhan jaisi?"
    variable: pain_quality
    options: [pressure_heaviness, burning, sharp_stabbing, aching_dull]
    source: "McGee: pressure LR+ 1.3 for ACS; sharp LR- 0.3 for ACS"

  - id: AC03
    text_en: "Does walking, climbing stairs, or exertion make the pain worse?"
    text_hi: "Kya chalne, seedhi chadhne, ya mehnat karne se dard badhta hai?"
    variable: exertional_relation
    options: [yes_clearly, sometimes, no, at_rest_only]
    source: "Fanaroff JAMA 2015: Exertional chest pain LR+ 2.4 for ischemia"

  - id: AC04
    text_en: "How long does each episode last?"
    text_hi: "Ek baar mein dard kitni der rehta hai?"
    variable: episode_duration
    options: [seconds, 2_20_min, 20_60_min, hours_continuous]
    source: "ACC/AHA: 2-20 min is classic anginal window"

  - id: AC05
    text_en: "Does pain worsen after eating or when lying down?"
    text_hi: "Kya dard khaana khaane ke baad badhta hai ya lete hue?"
    variable: meal_relation
    options: [postprandial, lying_down, no_relation]
    source: "Critical in Indian cohort: GERD and ACS overlap"

  - id: AC06
    text_en: "Have any medications been started, stopped, or changed recently?"
    text_hi: "Kya haal hi mein koi nayi dawai shuru hui hai ya dose badla hai?"
    variable: recent_med_change
    options: [new_med_started, dose_increased, no_change]
    source: "Highest yield in polypharmacy cohort"
```

The acuity layer narrows the differential category (acute/subacute/chronic, ischemic/pleuritic/GI) before discriminating questions refine individual posteriors. This requires Go change G7 -- a parallel acuity scoring track or acuity tags on questions.

#### 1d. Sex-Specific Atypical Presentation Modifiers (SM01-SM03) -- GO CHANGE G2

V2 specifies sex modifiers as **OR-based prior multipliers in the log-odds domain**, NOT LR questions:

```yaml
sex_modifiers:
  - id: SM01
    condition: "sex == Female"
    effect: "Increase anginal equivalent weight: DQ04 nausea LR+ 1.9->2.4; DQ02 jaw radiation LR+ 1.5->2.0; ACS prior +0.10"
    confidence: "HIGH_CONF"
    source: "INTERHEART 2004; Canto JAMA 2012 (women <55 with MI: 42% no chest pain)"

  - id: SM02
    condition: "sex == Female AND age >= 50"
    effect: "Post-menopausal: further increase ACS prior by OR 1.5 (log-odds +0.41)"
    confidence: "HIGH_CONF"
    source: "Framingham Heart Study; AHA 2016 Women + CVD"

  - id: SM03
    condition: "sex == Male AND pain_quality == burning"
    effect: "Do NOT rule out ACS on burning quality alone. Apply LR- ceiling of 0.7 for GERD"
    confidence: "CALIBRATE"
    source: "ACC/AHA 2021; Indian clinical experience"
```

**Go change G2**: Add `ApplySexModifiers()` after `InitPriors()` in the session initialization. Sex modifiers multiply odds (add to log-odds) rather than being processed as question-answer LR updates. SM01's OR 1.8 for female ACS = +0.59 log-odds to ACS prior; SM02's OR 1.5 = +0.41 additional. This is fundamentally different from an LR question.

#### 1e. Context Modifiers (CM01-CM10)

V2 specifies 10 CMs, all multiplicative in log-odds domain:

| CM ID | Condition | Effect (Log-Odds Shift) | Safety Flag | Source |
|---|---|---|---|---|
| CM01 | Known CAD / Prior MI / Prior PCI/CABG | ACS odds x2.0 (+0.69) | None | HIGH_CONF / Fanaroff JAMA 2015 |
| CM02 | DM duration > 10 years | ACS odds x1.5 (+0.41) | FLAG: Consider silent ischemia | HIGH_CONF / UKPDS; ADA 2024 |
| CM03 | eGFR < 45 (CKD 3b+) | ACS odds x1.8 (+0.59); activates DX10 if on metformin | FLAG: Check creatinine/eGFR | HIGH_CONF / ARIC CKD; KDIGO 2024 |
| CM04 | On beta-blockers | Reduce HR discriminating power; bidirectional | FLAG: HR may be misleadingly normal | HIGH_CONF / StatPearls |
| CM05 | SGLT2i + recent illness/fasting/surgery | Activate DX09; set prior 0.03 | SAFETY: Order ketone check | HIGH_CONF / ADA 2024 |
| CM06 | PDE5i (sildenafil, tadalafil) | No diagnostic change. **HARD_BLOCK** on nitrate. | HARD_BLOCK: No nitroglycerine | HIGH_CONF / SPL. **GO CHANGE G5** |
| CM07 | Metformin + eGFR<30 or illness/dehydration | Activate DX10; set prior 0.02 | SAFETY: Order lactate level | HIGH_CONF / KDIGO 2024 |
| CM08 | Antiplatelet + concurrent NSAID | GERD/GI bleed odds x2.0 (+0.69) | FLAG: Check Hb; ask about melena | HIGH_CONF / SPL Aspirin |
| CM09 | Age >= 65 + antihypertensives >= 3 | OH odds x1.8 (+0.59) | None | CONSENSUS / India OH study |
| CM10 | Current smoker or quit < 1 year | ACS odds x2.0 (+0.69) | None | HIGH_CONF / INTERHEART 2004 |

#### 1f. Safety Floors -- GO CHANGE G1

V2 specifies a non-negotiable ACS safety floor:

```yaml
safety_floors:
  ACS: 0.05  # ACS posterior NEVER drops below 0.05 regardless of negative findings
```

**Rationale**: Remote symptom assessment cannot rule out ACS in DM patients. Only in-person ECG + troponin can. This prevents false reassurance from maximally negative answers.

**Go change G1**: After each LR update in `BayesianEngine.Update()`, and in `GetPosteriors()` normalization, clamp specified differentials' posteriors to their floor values. The `safety_floors` map is read from `NodeDefinition` YAML.

#### 1g. Pertinent Negatives Documentation

V2 requires the clinician HPI output to explicitly document pertinent negatives:

| Pertinent Negative | Rules Out / Reduces | LR- | Source |
|---|---|---|---|
| No radiation to arms/jaw/neck | ACS | 0.7 | McGee |
| No exertional worsening | Stable angina | -- | ACC/AHA 2021 |
| No diaphoresis, no nausea | ACS (autonomic features) | 0.6 | Fanaroff. In DM with autonomic neuropathy, this negative is LESS reliable |
| Not pleuritic | PE, pericarditis | -- | Bruyninckx 2008 |
| No reproducibility on palpation | MSK | -- | Fanaroff 2015 |
| No leg swelling, no immobilisation | PE | -- | Wells Score |
| No medication change | Drug-induced causes | -- | N/A |
| No postprandial worsening | GERD | -- | McGee |

#### 1h. Convergence and Completion Criteria

```yaml
version: "2.0.0"
convergence_threshold: 0.65      # V2 spec: top differential >= 0.65
posterior_gap_threshold: 0.20     # Top-2 sum >= 0.85
convergence_logic: BOTH
max_questions: 14                 # 6 RF + 2 acuity + 6 discriminating max
guideline_prior_source: "ACC_AHA_2021,Jaipur_Heart_Watch_2017,ICMR_INDIAB,INTERHEART_2004"
```

**Closure guard**: If top differential < 0.50 after all questions -> YELLOW alert: "Atypical presentation. Insufficient data for remote rule-out of ACS. Recommend in-person evaluation."

---

### TIER 2: P02 Acute Dyspnea V1 (YAML + Go Change G4 + Manual CM Authoring)
**Duration**: 2 weeks | **Risk**: Medium | **Go changes**: G4 (DM_HTN_CKD_HF stratum)

P2 introduces the **parameterised strata architecture**: three prior table columns (DM_HTN_base, DM_HTN_CKD, DM_HTN_CKD_HF) with stratum-conditional LR values. Base stratum is fully populated; CKD and HF columns are schema-ready placeholders for Phase 2/3.

#### 2a. 10-Differential Closed Set with Parameterised Strata

```yaml
differentials:
  - id: ADHF
    label: "Acute Decompensated HF"
    population_reference: "THFR; ASIAN-HF Registry; REPORT-HF SE Asia"
    priors:
      DM_HTN_base: 0.18
      DM_HTN_CKD: 0.28        # placeholder -- directional
      DM_HTN_CKD_HF: 0.35     # placeholder -- directional

  - id: ACS_EQUIVALENT
    label: "ACS Equivalent (dyspnea-only ischemia)"
    population_reference: "INTERHEART SA; Canto JAMA 2012"
    priors:
      DM_HTN_base: 0.12
      DM_HTN_CKD: 0.10
      DM_HTN_CKD_HF: 0.08

  - id: PNEUMONIA
    label: "Pneumonia / Lower Respiratory Infection"
    priors:
      DM_HTN_base: 0.20
      DM_HTN_CKD: 0.15
      DM_HTN_CKD_HF: 0.12

  - id: COPD_EXAC
    label: "COPD / Asthma Exacerbation"
    priors:
      DM_HTN_base: 0.15
      DM_HTN_CKD: 0.12
      DM_HTN_CKD_HF: 0.08

  - id: PE_DYSPNEA
    label: "Pulmonary Embolism"
    priors:
      DM_HTN_base: 0.06
      DM_HTN_CKD: 0.06
      DM_HTN_CKD_HF: 0.05
    acuity: CATASTROPHIC

  - id: SEVERE_ANAEMIA
    label: "Severe Anaemia"
    priors:
      DM_HTN_base: 0.10
      DM_HTN_CKD: 0.15
      DM_HTN_CKD_HF: 0.12

  - id: ANXIETY_DYSPNEA
    label: "Anxiety / Hyperventilation"
    priors:
      DM_HTN_base: 0.07
      DM_HTN_CKD: 0.05
      DM_HTN_CKD_HF: 0.04

  - id: FLASH_OEDEMA
    label: "Hypertensive Pulmonary Oedema (Flash)"
    priors:
      DM_HTN_base: 0.06
      DM_HTN_CKD: 0.05
      DM_HTN_CKD_HF: 0.04

  - id: ACEI_COUGH_DYSPNEA
    label: "ACEi-Induced Cough/Dyspnea"
    priors:
      DM_HTN_base: 0.03
      DM_HTN_CKD: 0.02
      DM_HTN_CKD_HF: 0.02
    activation_condition: "med_class == ACEi"  # GO CHANGE G3

  - id: EUGLYCEMIC_DKA_DYSPNEA
    label: "Euglycemic DKA (SGLT2i-related)"
    priors:
      DM_HTN_base: 0.03
      DM_HTN_CKD: 0.02
      DM_HTN_CKD_HF: 0.02
    acuity: CATASTROPHIC
    activation_condition: "med_class == SGLT2i"  # GO CHANGE G3
```

**PRIORS SUM TO 1.00 (Base)**: 0.18+0.12+0.20+0.15+0.06+0.10+0.07+0.06+0.03+0.03 = 1.00. CKD/HF placeholder columns show directional shifts only; exact values pending Phase 2/3 evidence harvest. Each stratum column must independently sum to 1.00 (pre-G15) or 0.85 (post-G15).

> **⚠️ Pre-G15 YAML.** After G15 ('Other' bucket) implementation, reduce all priors proportionally so each stratum column sums to **0.85**, leaving 0.15 for the implicit 'Other' differential. Placeholder CKD/HF columns are directional only — exact values and normalization pending Phase 2/3 evidence harvest.

#### 2b. DM_HTN_CKD_HF Stratum -- GO CHANGE G4

**Current state**: `stratum_engine.go:145-168` only resolves DM_ONLY, DM_HTN, DM_HTN_CKD, HTN_ONLY, NONE.

**Required change**: Add HF detection in `determineStratum()`:

```go
hasHF := false
for _, c := range profile.Comorbidities {
    if c == "HF" || c == "HFrEF" || c == "HFpEF" || c == "HFmrEF" {
        hasHF = true
        break
    }
}

switch {
case hasDM && hasHTN && hasCKD && hasHF:
    return "DM_HTN_CKD_HF"
case hasDM && hasHTN && hasCKD:
    return models.StratumDMHTNCKD
// ... rest unchanged
}
```

**Impact**: Once this stratum exists, P02 YAML declares 4-column priors. `InitPriors()` looks up `diff.Priors["DM_HTN_CKD_HF"]` automatically. All existing nodes with fewer columns get uniform fallback (logged as warning).

#### 2c. Red Flag Sweep (RF01-RF06 with Cross-Node Safety)

```yaml
safety_triggers:
  - id: RF01
    condition: "Q_RF01_SUDDEN_SEVERE=YES"
    severity: IMMEDIATE
    recommended_action: "PE + ACS + Flash oedema triple pathway. SNOMED: 267036007"

  - id: RF02
    condition: "Q_RF02_SYNCOPE=YES"
    severity: IMMEDIATE
    recommended_action: "Massive PE / Cardiac tamponade / ACS. SNOMED: 271594007"

  - id: RF03
    condition: "Q_RF03_HEMOPTYSIS=YES"
    severity: IMMEDIATE
    recommended_action: "PE + malignancy evaluation. SNOMED: 66857006"

  - id: RF04
    condition: "Q_RF04_SPO2_LOW=YES"
    severity: IMMEDIATE
    recommended_action: "Immediate supplemental O2 + dual pathway. SNOMED: 431314004"

  - id: RF05
    condition: "Q_RF05_STRIDOR=YES"
    severity: IMMEDIATE
    recommended_action: "Airway emergency. SNOMED: 70407001"

  - id: RF06
    condition: "Q_RF06_CHEST_PAIN_COMBO=YES"
    severity: IMMEDIATE
    recommended_action: "Dual-node: CHEST_PAIN_V2 + DYSPNEA_V1 concurrent. SNOMED: 29857009+267036007"
    cross_node_action: "ACS safety floor escalates from 0.05 to 0.10 in BOTH nodes"
```

**Cross-node interaction (RF06)**: When chest pain + dyspnea co-occur, both nodes activate concurrently and the ACS safety floor increases from 0.05 to 0.10. This is the cross-node safety protocol.

#### 2d. Acuity/Context Questions (AC01-AC06)

P2's acuity layer mirrors P1 structure but with dyspnea-specific content:

| ID | Question | Variable | Options | Rationale |
|---|---|---|---|---|
| AC01 | Onset timing | onset_timing | sudden_minutes / hours / days / weeks_chronic | Sudden=PE/ACS/Flash; Hours=ADHF/Pneumonia; Weeks=Anaemia/COPD |
| AC02 | Positional component | positional | worse_lying / 2plus_pillows / no_positional / worse_upright | Orthopnea=HF (LR+ 2.2); Platypnea=shunt |
| AC03 | Exertional vs rest | exertional_rest | exertional_only / both / rest_only | Exertional=stable; Rest=acute emergency |
| AC04 | Associated cough character | cough_character | dry_persistent / productive_coloured / productive_frothy_pink / no_cough | Dry=ACEi; Productive=pneumonia; Frothy pink=pulm oedema |
| AC05 | Fever present | fever_present | yes_high / low_grade / no | Fever+dyspnea strongly favours infectious cause |
| AC06 | Recent medication change or missed doses | recent_med_change | new_med / stopped_med / missed_doses / dose_changed / no_change | Missed diuretic is #1 ADHF precipitant |

#### 2e. Discriminating Questions with Stratum-Conditional LRs -- GO CHANGE G6

P2 specifies stratum-conditional LR values for key questions:

| ID | Question | Primary Hypothesis | LR+ Base | Stratum Note | Source |
|---|---|---|---|---|---|
| DQ01 | Orthopnea (2+ pillows) | ADHF | 2.2; LR- 0.65 | CKD: ~1.6; CKD+HF: ~1.2 (near-universal) | Wang JAMA 2005; HIGH_CONF |
| DQ02 | PND | ADHF | 2.6; LR- 0.70 | CKD: ~2.0; CKD+HF: ~1.4 | Wang JAMA 2005; HIGH_CONF |
| DQ03 | Peripheral oedema | ADHF/DVT-PE | 2.1 bilateral; 2.8 unilateral | CKD: HF LR+ drops to ~1.4 (nephrotic oedema) | Wang; Wells; HIGH_CONF |
| DQ04 | Weight gain >=2kg/week | ADHF | 2.5 | Dominant discriminator in CKD+HF stratum | ACC/AHA 2022; CALIBRATE |
| DQ05 | Wheezing | COPD/Asthma | 2.1 | CKD+HF: cardiac asthma confounds; COPD LR drops | McGee; CALIBRATE |
| DQ06 | Fever + productive coloured sputum | Pneumonia | 3.0 | Consistent across strata | JAMA RCE; HIGH_CONF |
| DQ07 | Sudden onset + pleuritic pain | PE | 3.5 | Consistent across strata | ESC 2019; HIGH_CONF |
| DQ08 | Pallor + fatigue + exercise intolerance | Severe Anaemia | 2.5 | CKD: ~3.0 (EPO deficiency makes LR more specific) | CALIBRATE |
| DQ09 | Dry persistent cough (on ACEi) | ACEi-Cough | 4.0 | Drug-conditional (ACEi only) | ACCP 2006; HIGH_CONF |
| DQ10 | Nausea + rapid deep breathing (on SGLT2i) | Euglycemic DKA | 4.5 | Drug-conditional (SGLT2i only) | ADA 2024; HIGH_CONF |
| DQ11 | Missed diuretic >=2 days | ADHF decompensation | 3.0 | Drug-conditional (diuretics). #1 discriminator in CKD+HF | ACC/AHA 2022; CALIBRATE |
| DQ12 | Anxiety + perioral tingling + carpopedal spasm | Hyperventilation | 3.5 | Consistent across strata | JAMA RCE; HIGH_CONF |

**Adaptive selection note**: The table lists DQ01-DQ12 (12 questions) but the engine selects the **top 7 by information gain** from this pool per session. P2 completion criteria specifies "≥ 4 of 12 (adaptive selection)" and "7 discriminating max" in the question budget. Not all 12 are asked in any given session.

**Stratum re-ranking note**: In CKD+HF stratum, question ordering should shift: DQ04 (weight gain) and DQ11 (missed diuretic) become the dominant discriminators because DQ01/DQ02 (orthopnea/PND) lose discriminating power. The entropy-based selector partially achieves this automatically (because stratum-specific priors shift the posterior landscape), but explicit stratum-conditional LRs (Go change G6) are needed for full fidelity.

#### 2f. Context Modifiers (CM01-CM10)

| CM | Condition | Effect | Safety Flag | Source |
|---|---|---|---|---|
| CM01 | ACEi + cough/dry_cough | Activate DX09; prior 0.08. Bradykinin mechanism, onset hours-months | Suggest ARB switch | HIGH_CONF / ACCP 2006. Cat B. |
| CM02 | Beta-blocker + wheezing | COPD/Asthma odds x1.5 (+0.41). Mask tachycardia. | FLAG if new after BB initiation | HIGH_CONF / StatPearls. Cat B. |
| CM03 | SGLT2i + illness/fasting/surgery | Activate DX10; prior 0.05. Kussmaul breathing. Shared w/ P1 CM05. | SAFETY: Order ketone check | HIGH_CONF / ADA 2024. Cat A+B. |
| CM04 | Metformin + eGFR<30/illness/dehydration | Add lactic acidosis DX11 conditional; prior 0.02. Shared w/ P1 CM07. | SAFETY: Order lactate | HIGH_CONF / KDIGO 2024. Cat A. |
| CM05 | Known HF / prior HF admission | ADHF odds x2.5 (+0.92). Single strongest predictor. | None | HIGH_CONF / Wang JAMA 2005 |
| CM06 | eGFR < 45 (CKD 3b+) | ADHF odds x1.6 (+0.47). DX06 anaemia odds x1.5. TRIGGERS STRATUM SWITCH. | FLAG: Check creatinine | HIGH_CONF / ARIC; KDIGO 2024. Cat A. |
| CM07 | Loop diuretic + missed doses/vomiting/diarrhoea | ADHF odds x2.0 (+0.69). Missed diuretic = #1 ADHF precipitant. | Assess adherence/hydration | HIGH_CONF / ACC/AHA 2022. |
| CM08 | PDE5i + concurrent presentation | No diagnostic change. **HARD_BLOCK** on nitrate. Shared w/ P1 CM06. | HARD_BLOCK: No nitroglycerine | HIGH_CONF / SPL. **GO CHANGE G5** |
| CM09 | Antihypertensive recent uptitration (<=14 days) | Bidirectional: HTN pulm oedema if BP still high; hypotension-dyspnea if BP dropped too far | None | CONSENSUS. Cat B. |
| CM10 | Haemoglobin < 8 g/dL (lab available) | DX06 Anaemia prior: **OVERRIDE to 0.20**. Lab-confirmed severe anaemia. | FLAG: Transfusion threshold assessment | HIGH_CONF / WHO. **GO CHANGE G5** |

**CM08 HARD_BLOCK** and **CM10 OVERRIDE** require Go change G5 -- new CM effect types in `CMApplicator.Apply()`.

> **CM ID cross-reference note**: P2 source doc metadata references shared CMs using P1's IDs (CM05, CM06, CM07). In P2's own schema above, those same modifiers are CM03, CM08, and CM04 respectively. See P2 source doc §13.2 cross-reference table for correct mapping. The shared CM registry (B01) uses canonical IDs independent of per-node numbering.

#### 2g. Safety Floors (Stratum-Specific)

| Differential | Floor: DM_HTN Base | Floor: +CKD | Floor: +CKD+HF | Rationale |
|---|---|---|---|---|
| ACS Equivalent (DX02) | 0.05 | 0.05 | 0.05 | Cannot rule out ACS remotely in DM |
| PE (DX05) | 0.03 | 0.03 | 0.03 | Catastrophic if missed |
| ADHF (DX01) | 0.03 | 0.08 | 0.12 | **Stratum-specific**: CKD+HF = high floor to prevent false reassurance |
| Euglycemic DKA (DX10) | 0.02 (if on SGLT2i) | 0.02 | 0.02 | Drug-conditional catastrophic |
| Cross-node ACS (if RF06 fires) | 0.10 | 0.10 | 0.10 | Chest pain + dyspnea co-occurrence |

These use Go change G1 (safety floor clamping). **Stratum-specific floors** follow Gap Analysis A03: if a node uses strata, safety floors MUST be stratum-specific.

#### 2h. Sex Modifiers (SM01-SM03)

| ID | Condition | Effect | Source |
|---|---|---|---|
| SM01 | sex == Female | DX02 (ACS-eq) prior: odds x1.5 (+0.41). Dyspnea-only MI 42% more common in women <55. | Canto JAMA 2012; INTERHEART |
| SM02 | sex == Female AND age >= 50 | DX02 ACS-eq further OR 1.3 (+0.26). DX06 anaemia OR 1.4 (post-menopausal + Indian dietary deficiency) | Framingham; AHA 2016; NFHS-5 |
| SM03 | sex == Male AND age >= 55 AND smoker | COPD prior: OR 1.5 (+0.41). Biomass + tobacco dual exposure in rural India. | ICMR COPD; CALIBRATE |

#### 2i. Pertinent Negatives

| Pertinent Negative | Reduces | LR- | Reliability Note |
|---|---|---|---|
| No orthopnea, no PND | ADHF | 0.65 | In DM with autonomic neuropathy, LESS reliable (A06 gap) |
| No peripheral oedema | ADHF, nephrotic | 0.6 | Early ADHF may not show oedema yet |
| No fever, no productive cough | Pneumonia | -- | DM patients can have afebrile pneumonia |
| No pleuritic component | PE | -- | 50% of PE patients have no pleuritic pain |
| No unilateral leg swelling | PE (Wells) | -- | PE can occur without DVT signs |
| No wheezing | COPD/Asthma | -- | Silent chest = ominous (severe bronchospasm) |
| No medication change | Drug-induced causes | -- | Strongly reduces ACEi-cough, BB-bronchospasm |
| No pallor, no fatigue | Severe anaemia | -- | Pallor assessment less reliable in Indian cohort (skin pigmentation) |

#### 2j. Convergence and Completion Criteria

```yaml
version: "1.0.0"
convergence_threshold: 0.60      # P2 spec: lower than P1 (0.65) because dyspnea differentials overlap more
posterior_gap_threshold: 0.20     # Top-2 sum >= 0.80
convergence_logic: BOTH
max_questions: 16                 # 6 RF + 3 acuity + 7 discriminating max (from 12-question pool)
guideline_prior_source: "ACC_AHA_2022,THFR,ASIAN_HF,ESC_2019_PE,KDIGO_2024,INTERHEART_2004"
```

**Closure guard**: If top differential < 0.45 after all questions → YELLOW alert: "Overlapping respiratory and cardiac differentials. Recommend in-person evaluation with SpO2, CXR, BNP."

**Why P2 thresholds differ from P1**: Dyspnea has more overlapping differentials than chest pain. ADHF, pneumonia, anaemia, and COPD all share exercise intolerance and orthopnea — the engine needs more questions (16 vs 14) and accepts lower convergence (0.60 vs 0.65) before reaching a clinically useful conclusion.

---

### TIER 3: P3-P7 New Node Construction
**Duration**: Weeks 5-13 (after Phase 0 infrastructure sprint) | **Risk**: Medium

#### Phase 0: Infrastructure Sprint (Weeks 1-2, parallel with Tier 1)

The Gap Analysis identifies 6 BLOCKING gaps that must be resolved before ANY P3-P7 node enters clinical authoring:

| Gap ID | Gap | Owner | Blocks |
|---|---|---|---|
| A01 | Stratum-vs-Modifier Decision Framework: 1-page flowchart (4 binary questions -> STRATUM or MODIFIER) | HPI Engine Team | ALL nodes |
| A02 | Parameterised Strata YAML Schema: `strata_definitions[]`, `prior_tables{stratum: {dx: prior}}`, `lr_overrides{stratum: {q: {dx: lr}}}`, `stratum_activation_rules[]` | HPI Engine Team | P3, possibly P7 |
| A03 | Safety Floor Specification Pattern: if node uses strata -> floors MUST be stratum-specific; if single-stratum -> single floor row | HPI Engine Team | ALL nodes |
| B01 | Shared Context Modifier Registry (`shared_cm_registry.yaml`): canonical CM ID, trigger, effect, consuming nodes, version. Check before adding any CM. | KB-19 | P3 (immediate), ALL |
| B03 | Medication List Integration Smoke Test: synthetic patient on metformin+enalapril+empagliflozin+amlodipine+metoprolol. Verify all CMs fire. Map KB-6 drug names to CM med_class. | KB-20/KB-22 | ALL nodes (critical) |
| B04 | Stratum Activation from KB-20 Profile: stratum_selector module reads KB-20 patient profile, outputs stratum_id, loads correct prior_table column | KB-22 | P3 (critical), all stratified |

**Resolve A01 first** -- it determines whether A02/A03 apply to each node.

#### Revised Build Sequence (Dependency-Driven)

The original priority (P3->P4->P5->P6->P7) assumed no blocking dependencies. Gap analysis reveals P3 has 7 BLOCKING gaps while P4/P5 have 2-3 each. Revised sequence:

| Phase | Nodes | Architecture | Duration |
|---|---|---|---|
| Phase 0 | Infrastructure sprint (no nodes) | Resolve A01, A02, A03, B01, B03, B04 | Weeks 1-2 |
| Phase 1a | P4 Fatigue + P5 Numbness-Tingling (parallel) | Single-stratum (P1-style) for both. Lighter authoring burden. | Weeks 3-6 |
| Phase 1b | P3 Pedal Oedema | Parameterised strata (P2-style). Benefits from Phase 0 + P4/P5 shared CM discoveries. | Weeks 5-9 (overlaps 1a tail) |
| Phase 2 | P7 GI Disturbance | Single-stratum but heaviest medication overlay (12+ CMs). | Weeks 8-11 |
| Phase 3 | P6 Visual Changes | Most unique design: acuity classification not differential resolution. Build last. | Weeks 10-13 |

**KEY CHANGE**: P4+P5 move BEFORE P3. P3 has 7 BLOCKING dependencies and the most complex evidence harvest (CKD sub-stratification C03). Building P4+P5 first resolves shared evidence (metformin B12: C08+C11), populates the Shared CM Registry (B01), and validates single-stratum architecture at scale.

#### Per-Node Evidence Harvest Requirements

| Node | Differentials | Harvest Gaps | Shared Harvests | Architecture |
|---|---|---|---|---|
| P3 Pedal Oedema | CCB-induced, HF, CKD volume, nephrotic, DVT, hypothyroid (myxoedema) | C01-C05. C03 (CKD sub-stratification) is most complex. | CKD-Anaemia-Oedema, CCB ADR | Parameterised strata. CKD sub-strat decision pending. |
| P4 Fatigue | Hypothyroid, uncontrolled DM, B12 deficiency, depression, anaemia, deconditioning, sleep apnea | C06-C09. C06 (depression in Indian DM) requires cultural adaptation. | Metformin B12, Hypothyroidism, Depression | Single-stratum |
| P5 Numbness-Tingling | DPN, B12 neuropathy, CTS, peripheral vascular, CKD neuropathy | C10-C12. C10 (MNSI per-question LR decomposition) is most complex. A06 CRITICAL. | Metformin B12 | Single-stratum. A06 pertinent negative reliability is BLOCKING. |
| P6 Visual Changes | DR (symptomatic), glaucoma, retinal detachment, CRVO/CRAO, hypo/hyperglycemia visual | C13-C15. Most novel design: triage mode not differential mode. A05 BLOCKING. | Hypoglycaemia (from P0) | Single-stratum. Fundamentally different closure criteria. |
| P7 GI Disturbance | Metformin intolerance, gastroparesis, GERD, PUD, pancreatitis, celiac | C16-C18. C16 (metformin dose-response) is heaviest CM. D01 BLOCKING (KB-24 ADR). | SGLT2i comprehensive, CCB ADR | Single-stratum, heaviest CM overlay |
| P8 Urinary Symptoms | UTI (SGLT2i-associated), BPH, neuropathic bladder, CKD progression, SGLT2i genital mycosis | C19-C21. C19 (SGLT2i UTI/genital mycosis mechanism) is unique to this node. | SGLT2i comprehensive (from P3/P7) | Single-stratum. **Source**: Dizziness Evidence Harvest V2 Section 5 scaling roadmap. SGLT2i UTI data, IPSS score, KDIGO. |

#### Shared Evidence Harvest Optimization

Harvesting shared evidence once saves **35-40%** of total effort:

| Shared Harvest | Sources | Feeds | Est. Days |
|---|---|---|---|
| Metformin B12 Depletion | de Jager 2010, ADA 2024, StatPearls | P4 (B12 fatigue), P5 (B12 neuropathy) | 3-4 |
| CKD-Anaemia-Oedema Nexus | KDIGO 2024 anaemia, NFHS-5, WHO Hb | P3 (CKD oedema), P4 (anaemia fatigue) | 3-4 |
| Hypothyroidism in Indian DM | Unnikrishnan 2013, ATA 2012, ICMR | P3 (myxoedema), P4 (hypothyroid fatigue) | 2-3 |
| SGLT2i Comprehensive ADR | StatPearls, CANVAS, NKF, SPL | P3 (volume depletion masking), P7 (GI effects) | 2-3 |
| Depression in Indian DM | Patel 2008, Poongothai 2009, PHQ Hindi | P4 (depression) | 5-7 (cultural adaptation) |
| CCB ADR Profile | ACCOMPLISH, Makani 2015, Indian PV | P3 (CCB oedema), P7 (CCB nausea) | 4-5 |

**Total**: ~20-26 days shared + ~15-20 days node-specific = 35-46 days. Naive approach: ~55-70 days.

---

### TIER 4: Cross-Node Infrastructure Gaps (Full Registry)

#### Category A: Architecture Gaps

| Gap | Description | Resolution | Blocks |
|---|---|---|---|
| A01 | Stratum Decision Framework not codified | 1-page flowchart: 4 binary questions -> STRATUM or MODIFIER. Canon Framework mandatory pre-authoring step. | ALL |
| A02 | Parameterised Strata YAML Schema | Publish `parameterised_strata_schema.yaml` with field names, types, validation rules. Validate against P2. | P3, P7 |
| A03 | Safety Floor Pattern inconsistent (P1 single-row vs P2 stratum-specific) | Extend A01: if node uses strata -> stratum-specific floors required. | ALL |
| A04 | Question Ordering Re-Ranking by Stratum | Engine spec for `question_priority_override{stratum: [ordered_ids]}`. Currently entropy partially achieves this. | P3, all stratified |
| A05 | Confidence Threshold per Node not standardised | Define formula: base 0.65 minus 0.025 per differential beyond 8 (floor 0.50). Canon Framework. Calibrate after N=200. | P6 (BLOCKING), all quality |
| A06 | Pertinent Negative Reliability Modifiers | Add `reliability_modifier{condition: adjustment}` to pertinent negative schema. Conditions: autonomic_neuropathy, cultural_stoicism, cognitive_impairment. | P5 (BLOCKING), P4, P7 |

#### Category B: Cross-Node Infrastructure

| Gap | Description | Resolution | Blocks |
|---|---|---|---|
| B01 | Shared CM Registry | Build `shared_cm_registry.yaml`: canonical ID, trigger, effect, consuming nodes, version. KB-19 hosts at runtime. | P3 (immediate), P7, ALL |
| B02 | Cross-Node Safety Protocol Engine | KB-19 spec: node_activation_trigger, safety_floor_propagation, concurrent_output_merger. Min viable: two-node (P1+P2). | P3 (needs two-node), P4 (three-node eventually) |
| B03 | Medication List Integration Test | Smoke test: synthetic patient, verify all CMs fire, map KB-6 drug names to CM med_class. | ALL (critical) |
| B04 | Stratum Activation from KB-20 | KB-22 stratum_selector module: KB-20 profile -> stratum_id -> prior_table column. | P3 (critical), all stratified |
| B05 | Node Transition Protocol | Define three modes: CONCURRENT (P1+P2), HANDOFF (suspend/resume), FLAG_ONLY (clinician attention). | P3 (immediate), ALL |

#### Category D: Pipeline Dependencies

| Gap | Description | Pipeline Phase | Blocks |
|---|---|---|---|
| D01 | KB-24 ADR Profiles (four-element chain) -- Fix 1 in Section 6 | Phase 5 Week 1 | ALL nodes (manual workaround exists via KB-20 ADRService.Upsert) |
| D02 | ADA Standards extraction via GuidelineProfile | Phase 2 Week 2 | P4, P5, P6, P7 |
| D03 | RSSDI 2024 Indian-specific extraction | Phase 2 Week 2 | P4, P7 |
| D04 | RIE cross-system threshold validation | Phase 4 Week 3 | P3, P7 |
| D05 | SPLGuard PK-derived onset windows -- Fix 5 in Section 6. SPL pipeline already extracts PK_PARAMETERS tables (half-life, Tmax). Adding DeriveOnsetWindow() auto-upgrades STUB ADR profiles to PARTIAL. | Week 3-4 (low effort, high impact) | ALL nodes (reduces manual harvest from 4 elements to 2) |
| D06 | KB-24 completeness_grade field -- **already built** in KB-20 `adr_profile.go:78-95` | N/A (exists) | ALL |

#### Category E: Calibration Infrastructure (ALL MISSING — Three-Tier Strategy Adopted)

**Per BAY-9 (M2 Counter-Proposal §8.2)**: N=200 adjudicated examples is unreachable for 20+ months at pilot scale. Adopt **three-tier calibration strategy** (full details in Section 11.5). The infrastructure gaps below are re-prioritized by tier:

| Gap | Description | What P1/P2 Assumed | Resolution | Tier |
|---|---|---|---|---|
| E01 | Clinician Adjudication Interface | "After N=200 with clinician adjudication" -- no interface exists | Build minimal: clinician sees HPI output, selects actual Dx from dropdown, optional comment. Mobile-accessible. | Tier A (Month 0 — needed for expert panel review) |
| E02 | Per-Question Information Gain Tracker | "Prune questions with IG < 0.05 bits" -- no infrastructure computes this | KB-22 emits posterior_snapshot after each question via `hpi.session.events` Kafka topic. Pipeline computes KL(posterior_after \|\| posterior_before). Weekly report. | Tier B (Month 6+ — requires observed data) |
| E03 | Stratum-Specific Calibration Isolation | "Separate concordance for base vs CKD vs CKD+HF" -- requires partitioning calibration data | Min 50 per stratum for Tier B calibration, 200 for Tier C. CKD+HF will be underpowered for months. Use Tier A expert panel for stratum-specific adjustments until data accumulates. | Tier B/C (Month 6-18+) |
| E04 | PATA_NAHI Rate Tracking | "Reword questions with >25% unanswered rate" -- no tracking | ASR UNKNOWN mapping + per-question rate aggregation via `hpi.session.events`. **Critical for G16 pata-nahi cascade tuning.** Track from Day 1. | Tier A (Month 0 — immediate) |
| E05 | Cross-Node Concordance Measurement | P1+P2 concurrent -- which node gets "credit" for correct diagnosis? | Three-metric model: P0/P1/P2 concordance + MERGED concordance. Conflict Arbiter effectiveness measurement (Section 11.7). | Tier B (Month 6+) |
| E06 | **Contradiction Rate per Question Pair** (NEW, from BAY-4) | No tracking of contradiction rates between logically-related questions | `contradiction_event` logging (G17) feeds per-pair rate dashboard. Questions with >15% contradiction rate flagged for rewording. | Tier A (Month 0 — immediate via G17 logging) |
| E07 | **Clinical Source Registry** (NEW, from DIZ-7) | No LR provenance tracking separate from KB-20 ADR profiles | Deploy `clinical_sources`, `element_attributions`, `calibration_events` tables. Every YAML LR/prior linked to source record. See Section 11.12. | Tier A (Month 0 — Week 0-1 build) |

**Calibration KPI targets** (from Dizziness Evidence Harvest V2 §6):

| Metric | Target | Action if Off-Target |
|---|---|---|
| Concordance with physician top-1 diagnosis | ≥80% (long-term 85-90%) | Review LR weights; check CM firing rates |
| Red flag sensitivity | **100%** (non-negotiable) | Any miss → immediate node freeze and review |
| Closure rate (confident differential) | ≥85% | Add questions or adjust threshold |
| Median interaction length | 8-10 questions | Prune low-utility questions |
| False positive escalation rate | <5% | Raise RF thresholds |
| Pata-nahi rate per question | <25% | Reword question using `alt_prompt`; if persistent, simplify to binary |

---

## 4. Execution Timeline

### Week 0-1: P00 Dizziness Pilot + Core Engine Hardening (G14-G16)

| Day | Task | Type | Verification |
|---|---|---|---|
| 1 | **Implement G14** (CM composition in log-odds space) in CMApplicator | Go change | Unit test: patient with CM01+CM07 both targeting OH → p_adjusted=0.644, NOT 0.65 (naive sum). Test with 3+ CMs: sum never exceeds 1.0. |
| 1-2 | **Implement G15** ('Other' bucket differential) in BayesianEngine | Go change | Unit test: evidence weakening ALL listed differentials → Other rises. Other > 0.30 → DIFFERENTIAL_INCOMPLETE flag. |
| 2 | **Implement G16** (pata-nahi cascade protocol) — critical for P00 Hindi voice pilot. Add `alt_prompt` YAML field. Consecutive low-conf tracking, binary-only mode switch, PARTIAL_ASSESSMENT termination. | Go change + YAML schema | Unit test: 5 consecutive c<0.3 answers → PARTIAL_ASSESSMENT output. RF among low-conf → ESCALATE. |
| 2-3 | P00 DIZZINESS_V2 YAML authoring from Evidence Harvest V2 doc: 8 differentials, 6 RF, 6 acuity (TiTrATE), 8 DQ, 8 CMs, pertinent negatives, `alt_prompt` fields for all questions | YAML | NodeLoader validates; priors sum to 0.85 (0.15 for Other); all Hindi prompts present; pata-nahi cascade exercised |
| 3 | Register P00 CMs in shared CM registry (B01): CM01-CM08 from Dizziness harvest | DB + YAML | CMs queryable via CM registry; cross-referenced with P01/P02 CMs for overlap |
| 4 | Clinical Source Registry deployment: `clinical_sources`, `element_attributions`, `calibration_events` tables | Postgres | Schema deployed; Dizziness LR provenance records loaded (McGee, Kattah, AASK sources) |
| 4-5 | CC-1 (SCE) and CC-2 (question selection) decisions already resolved — see §12 | Architecture | Verify decisions documented in Section 12 |

### Week 1-2: Go Engine Changes + P01 V2

| Day | Task | Type | Verification |
|---|---|---|---|
| 1-2 | Implement G1 (safety floor clamping) in BayesianEngine | Go change | Unit test: maximally negative answers -> ACS posterior stays >= 0.05 |
| 2-3 | Implement G3 (medication-conditional differentials) in InitPriors | Go change | Unit test: DX09 included only when SGLT2i active; excluded otherwise with prior redistribution |
| 3-4 | Implement G2 (sex-modifier prior adjustment) | Go change | Unit test: female patient -> ACS log-odds shifted by +0.59 |
| 4-5 | P01 YAML V2 upgrade (3a-3h): 10 differentials, RF01-RF06, AC01-AC06, SM01-SM03, CM01-CM10, safety floors | YAML | NodeLoader validates on restart; all 9 rules pass; priors sum to 0.85 (0.15 for Other) |
| 5 | Phase 0 infrastructure: A01 (stratum decision tree) draft | Spec | Canon Framework review |

### Week 2-3: P02 Base Launch + Infrastructure

| Day | Task | Type | Verification |
|---|---|---|---|
| 1 | Implement G4 (DM_HTN_CKD_HF stratum) | Go change | Unit test: HF patient returns correct stratum |
| 1-2 | Implement G5 (HARD_BLOCK, OVERRIDE CM types) | Go change | Unit test: CM06 PDE5i blocks nitrate; CM10 Hb<8 overrides anaemia posterior to 0.20 |
| 2-4 | P02 YAML expansion: 10 differentials, RF01-RF06, AC01-AC06, DQ01-DQ12, SM01-SM03, safety floors | YAML | NodeLoader validates; cross-reference LR sources |
| 4-5 | B03 Smoke test: synthetic patient (DM+HTN, metformin, eGFR 38, breathlessness) | Test fixture | CM06 fires (eGFR<45 stratum), CM04 fires (metformin+eGFR check), correct prior table loads |

**P2 Launch Gate**: Synthetic patient -> KB-22 initialisation -> verify: correct stratum, CMs fire, correct prior table, valid JSON output. P2 launches with DM_HTN_base stratum fully populated; CKD/HF columns as schema-ready placeholders.

### Week 3-4: CM Authoring + Pipeline Fixes + SPLGuard Integration + SCE Build

| Day | Task | Type | Verification |
|---|---|---|---|
| 0-1 | **SCE Build** (if CC-1 decision = separate service): scaffold Go service at port 8201, RF pattern matching from YAML, `/v1/session/escalate` webhook, independent health check. Add KB-19 dual routing. | Go (new service) | SCE catches all 6 P00 red flags independently of M2. Health check at :8201/health. M2 kill → SCE still processes answers. |
| 0-1 | **Implement G18** (closure multi-criteria guard) + **G19** (skip-redundancy rule) | Go change | Closure requires top_p + confidence > 0.75 + 2 supporting answers. Skip-redundancy fires when CM delta ≥ 0.30 already covers question's target differentials. |
| 1 | Run SPLGuard for P01+P02 drug sets (~10 drugs). Create STUB ADR profiles in KB-20. | SPL pipeline | SAFETY_SIGNAL facts in FactStore; STUB records in KB-20 adverse_reaction_profiles |
| 1-2 | Implement Fix 5 (DeriveOnsetWindow from PK data). Auto-upgrade STUBs to PARTIAL. | Go (SPL pipeline enrichment) | STUB records with PK data upgrade to PARTIAL; onset_window + onset_category populated |
| 2-4 | Clinician mechanism harvest for P01 CMs: StatPearls monographs for ARB, BB, SGLT2i, PDE5i, SU | Manual + DB insert | Mechanism field populated; merge strategy preserves SPL frequency + pipeline onset; completeness grade upgrades toward FULL |
| 3-5 | Manual CM rule authoring for P01 (10 CMs) and P02 (10 CMs) into KB-20 DB | DB insert | All CMs queryable via CM registry; records with all 4 elements = FULL |
| 5-6 | KB-24 extraction target (Fix 1): add "contextual"/"adverse_effects" to target_kbs. L3 template for 4-element chain (source-agnostic for SPL + guideline + manual). | Python pipeline | Pipeline runs on KDIGO with new target; L3 produces valid records; dual-path merge validates against SPL entries |
| 6-7 | Backward compatibility diff test: full pipeline on all 3 KDIGO sources | Pipeline test | Byte-identical output for KB-1/KB-4/KB-16 artifacts. **Budget 2 full days.** |

### Week 4-5: Phase 0 Infrastructure Sprint + G6/G7 + Contradiction Detection (G17)

| Day | Task | Type | Verification |
|---|---|---|---|
| 0-1 | **Implement G17** (contradiction detection matrix). Add `contradiction_pairs` YAML field. Implement re-ask protocol and `contradiction_event` logging. Max 1 re-ask per pair per session. | Go change + YAML schema | Unit test: AC02=YES then DQ03=NO → contradiction fires → re-ask AC02 → confirmed answer's LR applied |
| 1-2 | Complete Phase 0: A01 flowchart, A02 schema, A03 floor spec, B01 shared CM registry, B04 stratum activation | Spec + Go | Canon Framework review; B01 populated with P0+P1+P2 shared CMs |
| 2-3 | Implement G6 (stratum-conditional LRs) -- optional override maps | Go change | Unit test: orthopnea LR+ = 2.2 in base, 1.2 in CKD+HF |
| 3-5 | Implement G7 (acuity question layer) | Go change | Unit test: AC01-AC06 processed as acuity scoring, not Bayesian update |
| 5 | RIE build (Fix 3): monotonic severity + cross-system threshold consistency | Python pipeline | eGFR thresholds form coherent ordering |

### Week 5-6: API + Kafka + Provenance (Counter-Proposal Sprint 2-3)

| Day | Task | Type | Verification |
|---|---|---|---|
| 1-2 | Implement `/v1/session/escalate`, `/v1/session/multi-init`, `/v1/node/validate` endpoints (BAY-10) | Go API | POST to /escalate → session interrupted. POST to /multi-init → two linked sessions created. POST to /validate → schema errors returned for bad YAML. |
| 2-3 | Kafka topic setup: `hpi.session.events`, `hpi.escalation.events`, `hpi.calibration.data` (BAY-11). Wire M2 and SCE producers. | Kafka + Go | SessionInitialized/AnswerProcessed events on Kafka. Escalation events from SCE on separate topic. |
| 3-4 | Provenance logging: per-update audit trail to Postgres (question_id, answer_code, confidence, old_logodds, ln(LR), new_logodds). Session replay at `/v1/session/status/{session_id}`. | Go + Postgres | Full provenance chain replayable for any session. |
| 4-5 | Clinician output template: structured HPI note (GP-facing) + patient summary (Hindi, WhatsApp-ready via M6) | Template | GP sees: CC, key positives/negatives, top differential with posterior, red flag status, suggested actions, CMs fired. |
| 5 | Establish Tier A calibration: identify 3-clinician panel (diabetologist, internist, GP). Schedule first quarterly review for P00 Dizziness. | Governance | Panel appointed; first review date set. |

### Weeks 6-7: Conflict Arbiter (Counter-Proposal Sprint 4)

| Day | Task | Type | Verification |
|---|---|---|---|
| 1-3 | Implement Conflict Arbiter in KB-19 (BAY-7, replaces B02): BOOST/FLAG/REPORT/RED_FLAG_WINS rules. Post-processing after both node sessions complete. | Go (KB-19) | Two-node test: P00 Dizziness + P01 Chest Pain concurrent. Shared hypoglycemia → BOOST. Contradictory evidence → FLAG. Independent top-Dx → REPORT BOTH. RF in one node → escalate regardless of other. |
| 3-4 | `/v1/node/validate` CI/CD integration: run validate on every YAML change before merge | CI pipeline | PR with bad YAML → validate endpoint rejects → PR blocked. |

### Weeks 7-15: P3-P8 Construction (Revised Sequence)

| Week | Phase | Nodes | Evidence Harvest |
|---|---|---|---|
| 7-8 | Shared harvest | Metformin B12, CKD-Anaemia-Oedema, Hypothyroidism, Depression (cultural adaptation) | 20-26 days shared |
| 7-10 | Phase 1a | P4 Fatigue + P5 Numbness-Tingling (parallel, single-stratum) | C06-C12 |
| 9-13 | Phase 1b | P3 Pedal Oedema (parameterised strata, CKD sub-strat) | C01-C05 |
| 12-15 | Phase 2 | P7 GI Disturbance (heaviest CM overlay) | C16-C18 |
| 14-17 | Phase 3a | P6 Visual Changes (triage mode, unique design) | C13-C15 |
| 16-18 | Phase 3b | **P8 Urinary Symptoms** (SGLT2i UTI/mycosis, BPH, neuropathic bladder) | C19-C21 |

### Week 14+: Deferred Items (Go Changes + Infrastructure)

| Item | Type | Blocking? | Notes |
|---|---|---|---|
| G8 CM active state in safety conditions | Go change | No -- P2 ST004 can use question-only conditions | Enables CM-aware safety triggers |
| G9 Conditional priors (bp_status overrides) | Go change | No -- only HTN nodes need this | P5+ only |
| G10 CATEGORICAL answer types | Go change | No -- all current nodes use YES/NO/PATA_NAHI | P7+ only |
| G11 Action thresholds | Go change | No -- KB-23 handles downstream | Separate ActionEngine service |
| G12 COMPOSITE_SCORE safety triggers | Go change | No | Remove NodeLoader guard, implement weighted scoring |
| G13 Node transition protocol (B05) | Go change | No -- CONCURRENT mode (RF06) works; HANDOFF mode deferred | NodeTransition struct + evaluation logic |
| ADA/RSSDI pipeline parameterisation (Fix 2) | Python pipeline | Blocks P4-P7 pipeline automation, not manual authoring | Channel A/C/G guideline_profile config |

### Week 16+: Deferred Items (Counter-Proposal Phase 2)

| Item | Type | Blocking? | Notes |
|---|---|---|---|
| Tier B calibration (beta-binomial shrinkage) | Engineering | Blocks Month 6-18 calibration upgrades, not launch | Requires 200+ sessions per node for meaningful shrinkage |
| Tier C calibration (full data-driven) | Engineering | Blocks Month 18+ calibration, not launch | Multi-site pooling, per-stratum Bayesian updates |
| Phase 2 question selection (greedy LR-variance) | Go change | No -- author-ordered (Phase 1) works for launch | See §11.8; requires LR variance metadata in YAML |
| Phase 3 question selection (full IG) | Go change | No -- Phase 1 sufficient | See §11.8; compute-intensive, ≥500 session training data |
| P8 Urinary Symptoms node | YAML + Evidence harvest | No -- P3-P7 take priority | Tier 3b (Phase 3b Week 15-16); DX01-DX08 authored |
| SCE Phase 2 (multi-node SCE coordination) | Go service | No -- single-node SCE covers P0-P2 | Requires multi-node session protocol (G13) first |
| Conflict Arbiter tuning (false-positive BOOST) | Calibration | No -- conservative REPORT-default safe for launch | Need clinical review of BOOST→FLAG boundaries after 500+ multi-node sessions |

---

## 5. Version Pinning Technical Debt Note

**Problem**: When the pipeline extracts "eGFR < 45" from KDIGO and writes it to KB-1, and P2's CM06 YAML hardcodes "eGFR < 45", there's an implicit coupling. If KDIGO updates the threshold to 40, the pipeline would extract 40 into KB-1, but P2's YAML still says 45.

**Resolution options**:
1. **Runtime reference**: P2 YAML references `kb1:eGFR_metformin_threshold` instead of hardcoding 45. KB-22 resolves this at runtime from KB-1.
2. **Pipeline-sync check**: Canon Framework review process includes a "pipeline-sync check" that flags when P2 YAML thresholds don't match KB-1 extracted values.

This isn't a Week 1-4 build item, but without it, guideline updates create silent inconsistencies between pipeline output and HPI node content. **Owner: KB-22 HPI Engine Team. Resolve before any KDIGO or ADA guideline version is upgraded in the pipeline.** For a system making clinical recommendations, guideline-version drift is a patient safety issue, not just technical debt.

---

## 6. Pipeline Integration (Fix 1-4)

### Fix 1: KB-24 Extraction Target + SPLGuard Integration (Highest Impact)

Add "contextual" and "adverse_effects" to Pipeline 2's `target_kbs` mapping (line 852). Create L3 template for context modifier facts matching the four-element chain. **The L3 template is designed for THREE sources**:

1. **Guideline pipeline** (`fact_extractor.py`): Extracts eGFR thresholds, contraindication conditions, monitoring requirements from KDIGO/ADA/RSSDI
2. **SPLGuard pipeline** (`shared/cmd/spl-pipeline/main.go`): Extracts drug-->symptom associations (LOINC 34084-4), PK parameters for onset derivation (LOINC 34090-1), DDI signals (LOINC 34073-7)
3. **Manual curated**: Clinician evidence harvest from StatPearls, PMC reviews, clinical experience

Template must be source-agnostic. All three sources write to KB-20's `adverse_reaction_profiles` and `context_modifiers` tables via the dual-path merge strategy (`adr_service.go:55-95`).

**What each source provides for the four-element chain**:

| Source | Element 1 (Drug-->Symptom) | Element 2 (Mechanism) | Element 3 (Onset Window) | Element 4 (CM Rule) |
|---|---|---|---|---|
| SPLGuard | YES: MedDRA PT + frequency band from LOINC 34084-4 | NO | PARTIAL: PK tables (half-life, Tmax) from LOINC 34090-1 --> derivable via 5x half-life | NO |
| Guideline pipeline | PARTIAL: Channel G sentence extraction captures drug-symptom mentions | NO | NO | PARTIAL: eGFR thresholds, contraindication conditions |
| Manual curated | SUPPLEMENTAL: literature-specific frequency data | YES: StatPearls mechanism-of-action per drug class | YES: literature synthesis + PK derivation | YES: clinician-calibrated magnitude + clinical judgment |

**Resulting completeness after each source**:
- SPLGuard alone --> STUB (drug+symptom only, no mechanism or CM rule)
- SPLGuard + PK derivation (Fix 5) --> PARTIAL (drug+symptom+onset, no mechanism or CM rule)
- SPLGuard + Manual curated --> FULL (all 4 elements, mechanism + onset from literature, CM rule calibrated)
- Guideline pipeline + SPLGuard --> PARTIAL (merge strategy combines both; SPL wins for onset, pipeline wins for CM rule)
- All three sources merged --> FULL at highest confidence

**KB-20 merge sequence** (operational):
```
Step 1: SPLGuard creates record {source: "SPL", drug_class, reaction, meddra_pt, frequency} --> STUB
Step 2: PK derivation adds onset_window from half-life --> auto-upgrade to PARTIAL
Step 3: Guideline pipeline adds context_modifier_rule from eGFR thresholds --> stays PARTIAL (missing mechanism)
Step 4: Clinician adds mechanism from StatPearls --> upgrade to FULL (all 4 elements + confidence >= 0.70)
```

The merge strategy ensures no data is lost: SPL's frequency band is retained when pipeline adds CM rule; pipeline's eGFR threshold is retained when clinician adds mechanism.

### Fix 2: ADA/RSSDI Pipeline Parameterisation

Channel A and Channel C need `guideline_profile` configuration for non-KDIGO sources. Pipeline lines: 51-54 (source choices), 143-152 (guideline_context), 314-320 (heading patterns "Standard X.Y"), 331-335 (regex profiles). Channel G also needs the guideline_profile parameter. P2 needs ADA Section 10 (CVD) for ACS-equivalent prevalence data.

### Fix 3: Range Integrity Engine (RIE)

Four checks: (1) eGFR intervals monotonic severity, (2) no overlapping thresholds within drug, (3) no coverage gaps, (4) **cross-system threshold consistency** -- stratum activation boundaries (eGFR >= 60 -> DM_HTN_base, eGFR < 60 -> DM_HTN_CKD) must be consistent with dosing thresholds from KB-1. Include cross-system slot in `RangeIntegrityReport` schema from day one.

### Fix 4: Channel G + H (Deferred, not blocking P2)

Sentence extraction (G) and cross-channel recovery (H) improve completeness but P2's Category B data comes from SPL/StatPearls harvest, not sentence extraction.

### Fix 5: SPLGuard PK-Derived Onset Windows (High Value, Low Effort)

The SPL pipeline's `TabularHarvester` (`shared/extraction/spl/tabular_harvester.go`) already classifies and extracts `PK_PARAMETERS` tables (half-life, Tmax, Cmax, AUC) from LOINC section 34090-1 (Clinical Pharmacology). **No code exists to convert these PK parameters into onset windows for ADR profiles.** Adding this transform would auto-upgrade every STUB ADR record (created by Phase G DraftFact) to PARTIAL -- the single highest-ROI automation for Category B data.

**Proposed addition** (in Phase G DraftFact creation or as a post-Phase I enrichment step):

```go
// DeriveOnsetWindow converts PK parameters to clinical onset window for ADR profiles.
// Based on pharmacological principle: ADRs peak around Cmax/steady-state (5x half-life).
func DeriveOnsetWindow(halfLifeHours float64, tmaxHours float64) (onsetWindow string, onsetCategory string) {
    if halfLifeHours <= 0 {
        return "", ""
    }
    steadyStateDays := (halfLifeHours * 5) / 24.0
    if steadyStateDays <= 1 {
        return fmt.Sprintf("Hours %.0f-%.0f", tmaxHours, halfLifeHours*5), "IMMEDIATE"
    } else if steadyStateDays <= 7 {
        return fmt.Sprintf("Days 1-%d", int(math.Ceil(steadyStateDays*2))), "ACUTE"
    } else if steadyStateDays <= 28 {
        return fmt.Sprintf("Days 1-%d", int(math.Ceil(steadyStateDays*2))), "SUBACUTE"
    }
    return fmt.Sprintf("Weeks 1-%d", int(math.Ceil(steadyStateDays/7))), "CHRONIC"
}
```

**Drug class onset windows derivable from existing SPL PK data**:

| Drug Class | SPL Half-Life | Derived Onset Window | OnsetCategory | Verified Against Literature |
|---|---|---|---|---|
| ARB (telmisartan) | ~24h | Days 1-14 | ACUTE | YES -- StatPearls: first-dose phenomenon Days 1-7 |
| SGLT2i (dapagliflozin) | ~13h | Days 1-6 (acute), Days 1-28 (volume depletion) | ACUTE/SUBACUTE | PARTIAL -- acute PK-derivable; chronic needs CANVAS data |
| Beta-blocker (metoprolol) | ~3-7h | Hours to Days 1-4 (bradycardia); IMMEDIATE (hypo masking) | IMMEDIATE | YES -- masking is immediate on therapeutic levels |
| Sulfonylurea (glimepiride) | ~5-8h | Hours 2-24 | ACUTE | YES -- SPL Tmax = peak hypoglycemia risk |
| CCB (amlodipine) | ~40h | Days 1-24 | SUBACUTE | YES -- slow onset matches clinical observation |
| Thiazide (HCTZ) | ~6-15h | Days 1-28 (electrolyte) | SUBACUTE | PARTIAL -- cumulative effect, not pure PK |
| Metformin | ~6h | Days 1-4 (GI); Weeks (lactic acidosis) | ACUTE/DELAYED | PARTIAL -- GI is PK-derivable; lactic acidosis is not |

**Implementation timeline**: Week 3-4 (parallel with CM authoring). Requires: (1) extract PK_PARAMETERS from existing SPL runs, (2) apply DeriveOnsetWindow(), (3) write onset_window + onset_category to KB-20 ADR profiles via pipeline API, (4) merge strategy auto-upgrades STUBs to PARTIAL.

**Impact**: For the ~10 high-value drugs in DM+HTN cohort, this automates Element 3 for 70-80% of drug-symptom pairs, reducing manual evidence harvest from 4 elements to 2 (mechanism + CM rule only).

---

## 7. Go Change Summary

**19 total Go changes identified; 13 required for Weeks 1-6, 6 deferred:**

| # | File | Change | Week | Required By |
|---|---|---|---|---|
| G1 | `kb-22/.../services/bayesian_engine.go` | Safety floor clamping: read `safety_floors` from NodeDef, clamp posteriors after each update | 1 | P1 V2, P2 V1 |
| G2 | `kb-22/.../services/bayesian_engine.go` | Sex-modifier prior adjustment: `ApplySexModifiers()` after InitPriors, OR-based log-odds shifts | 1 | P1 V2, P2 V1 |
| G3 | `kb-22/.../models/node.go` + `bayesian_engine.go` | Medication-conditional differentials: `activation_condition` field, conditional inclusion in InitPriors | 1 | P1 V2, P2 V1 |
| G4 | `kb-20/.../services/stratum_engine.go` + models | DM_HTN_CKD_HF stratum: HF detection in determineStratum() | 2 | P2 V1 |
| G5 | `kb-22/.../services/cm_applicator.go` | HARD_BLOCK + OVERRIDE CM effect types in Apply() | 2 | P1 V2 (CM06), P2 V1 (CM08, CM10) |
| G6 | `kb-22/.../models/node.go` + `bayesian_engine.go` | Stratum-conditional LR overrides: `lr_positive_by_stratum` map | 4 | P2 V1 stratum fidelity |
| G7 | `kb-22/.../services/` (new file) | Acuity question layer: parallel scoring track | 4 | P1 V2, P2 V1 |
| G8 | `kb-22/.../services/safety_engine.go` | CM active state in condition evaluator | Deferred | P2 ST004 |
| G9 | `kb-22/.../models/node.go` + `bayesian_engine.go` | Conditional priors (bp_status overrides) | Deferred | P5+ |
| G10 | `kb-22/.../services/bayesian_engine.go` | CATEGORICAL answer types | Deferred | P7+ |
| G11 | New service | Action thresholds engine | Deferred | KB-23 handles |
| G12 | `kb-22/.../services/node_loader.go` | COMPOSITE_SCORE safety triggers | Deferred | None |
| G13 | `kb-22/.../models/node.go` + session service | Node transition protocol (CONCURRENT/HANDOFF/FLAG) | Deferred | B05 |
| G14 | `kb-22/.../services/cm_applicator.go` | CM composition in log-odds space: convert author deltas via logit-shift, apply additively. See §11.1. | 1 | ALL (critical for polypharmacy) |
| G15 | `kb-22/.../services/bayesian_engine.go` + models | 'Other' bucket differential: geometric-mean inverse LR update, DIFFERENTIAL_INCOMPLETE flag at p>0.30, soft escalation at p>0.45 | 2 | ALL nodes |
| G16 | `kb-22/.../services/bayesian_engine.go` + session | Pata-nahi cascade: consecutive low-conf tracking, rephrase→binary→terminate→escalate protocol. YAML `alt_prompt` field. | 1 | ALL (critical for Hindi voice — P00 pilot exercises immediately) |
| G17 | `kb-22/.../models/node.go` + `bayesian_engine.go` | Contradiction detection matrix: `contradiction_pairs` YAML field, re-ask protocol, `contradiction_event` logging | 4 | ALL nodes |
| G18 | `kb-22/.../services/bayesian_engine.go` | Closure multi-criteria guard: top_p > threshold + decisive answer confidence > 0.75 + supporting answers ≥ 2 | 2 | ALL nodes |
| G19 | `kb-22/.../services/question_orchestrator.go` | Skip-redundancy rule: skip questions whose target differentials have CM delta ≥ 0.30 already applied | 2 | ALL nodes |

---

## 8. Risk Matrix

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| G1-G3 Go changes delay Tier 1 beyond 2 weeks | Medium | High | G1 (floor clamping) is simple -- 20 lines. G3 (conditional differentials) is the hardest. Prototype G3 first. |
| Prior rebalancing causes convergence regression | Medium | High | Run existing P01/P02 test fixtures before and after YAML change; compare convergence speed |
| ~~KB-18 naming collision~~ (RESOLVED → KB-24) | ~~High~~ | ~~High~~ | **Resolved**: KB-18 = Audit Trail. ADR extraction target reassigned to KB-24. |
| Manual CM entries conflict with future pipeline output | Low | Medium | Completeness grading ensures manual CALIBRATE < pipeline HIGH_CONF; pipeline wins on upgrade |
| ADA pipeline parameterisation breaks KDIGO extraction | Medium | High | Backward compatibility diff test on all 3 KDIGO sources (budgeted 2 days in Week 3-4) |
| DM_HTN_CKD_HF stratum breaks existing node priors | Low | Low | `InitPriors()` falls back to uniform with warning for missing stratum columns |
| Safety floor clamping distorts posterior distribution | Low | Medium | Clamp AFTER normalization; use soft floor (asymptotic approach, not hard cutoff) if distribution distortion is significant |
| CKD+HF stratum calibration sample insufficient | High | Medium | Min 50 per stratum; may need targeted recruitment. Launch with base stratum calibrated. |
| Acuity layer (G7) scope creep delays core engine | Medium | Medium | Start with acuity tags on existing question struct (minimal Go change) rather than full AcuityEngine service |
| SCE separate service adds deployment complexity (BAY-6) | Medium | Medium | SCE port 8201 runs as sidecar on same host. Health check + circuit breaker in KB-19 routing. Fallback: inline safety check if SCE unreachable. |
| Pata-nahi cascade Hindi ASR tuning inadequate (BAY-3) | High | Medium | `alt_prompt` YAML field allows per-question rephrase text. Binary-only fallback (count=3) eliminates ASR ambiguity. Pilot with P00 Dizziness 8 questions first. |
| Conflict Arbiter false-positive BOOSTs (BAY-7) | Medium | Medium | Default to REPORT (informational) not BOOST. Require 500+ multi-node sessions before enabling BOOST rules. Clinical review gate. |
| 'Other' bucket inflates prematurely on noisy input (BAY-2) | Medium | High | Geometric-mean inverse LR update is conservative. DIFFERENTIAL_INCOMPLETE flag at 0.30 is warning-only; soft escalation at 0.45 requires clinician confirmation. |
| Log-odds CM composition produces extreme posteriors (BAY-1) | Low | High | Cap total CM log-odds shift at ±2.0 (equivalent to ~0.12 to 0.88 probability range). Warn if >3 CMs fire on same differential. |
| Contradiction detection re-ask loop frustrates patients (BAY-4) | Low | Medium | Max 1 re-ask per contradiction pair per session. If 2nd answer still contradicts, accept most recent and log `contradiction_unresolved`. |

---

## 9. Gap-to-Node Dependency Matrix

From P3-P7 Gap Analysis Section 7. BLOCKING = cannot author until resolved. REQUIRED = must resolve before deployment. BENEFICIAL = improves quality.

| Gap | P3 Oedema | P4 Fatigue | P5 Numbness | P6 Visual | P7 GI | P8 Urinary |
|---|---|---|---|---|---|---|
| A01 Stratum Decision | BLOCKING | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED |
| A02 Strata Schema | BLOCKING | -- | -- | -- | TBD | -- |
| A03 Safety Floor Pattern | BLOCKING | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED |
| A04 Question Reranking | BLOCKING | -- | -- | -- | TBD | -- |
| A05 Confidence Threshold | REQUIRED | REQUIRED | REQUIRED | **BLOCKING** | REQUIRED | REQUIRED |
| A06 Pertinent Neg Reliability | BENEFICIAL | REQUIRED | **BLOCKING** | BENEFICIAL | BENEFICIAL | BENEFICIAL |
| B01 Shared CM Registry | BLOCKING | REQUIRED | REQUIRED | BENEFICIAL | BLOCKING | REQUIRED |
| B02 Cross-Node Safety Engine | REQUIRED | BENEFICIAL | -- | -- | -- | BENEFICIAL |
| B03 Med List Test | BLOCKING | BLOCKING | BLOCKING | BLOCKING | BLOCKING | BLOCKING |
| B04 Stratum Activation | BLOCKING | -- | -- | -- | TBD | -- |
| B05 Node Transition | REQUIRED | REQUIRED | BENEFICIAL | BENEFICIAL | REQUIRED | BENEFICIAL |
| D01 KB-24 ADR (Fix 1) | REQUIRED | REQUIRED | REQUIRED | BENEFICIAL | BLOCKING | REQUIRED |
| D02 ADA Extraction | BENEFICIAL | REQUIRED | REQUIRED | REQUIRED | REQUIRED | BENEFICIAL |
| D04 RIE Cross-System | REQUIRED | -- | -- | -- | REQUIRED | -- |
| D05 SPLGuard PK Onset (Fix 5) | BENEFICIAL | BENEFICIAL | BENEFICIAL | -- | BENEFICIAL | BENEFICIAL |
| D06 KB-24 Completeness (exists) | -- | -- | -- | -- | -- | -- |
| E01 Adjudication Interface | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED |
| G14 CM Log-Odds Composition | REQUIRED | REQUIRED | REQUIRED | BENEFICIAL | REQUIRED | REQUIRED |
| G15 Other Bucket | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED | REQUIRED |
| G16 Pata-Nahi Cascade | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL |
| G17 Contradiction Detection | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL |
| SCE (BAY-6) | REQUIRED | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL | BENEFICIAL |

---

## 10. SPL Four-Element Chain: Per-Node Population Protocol

This section specifies how to populate the four-element ADR chain for each HPI node using the SPLGuard pipeline, PK derivation, and clinician evidence harvest. See `claudedocs/SPL_FOUR_ELEMENT_CHAIN_FOR_HPI_NODES.md` for the full technical deep-dive with codebase verification.

### 10.1 Population Steps (per node)

**Step 1: Run SPLGuard for node-relevant drugs** (automated)
```bash
# Create drugs file for the node's differential set
# Example for P01 Chest Pain: all drugs whose ADR profiles affect chest pain differentials
go run shared/cmd/spl-pipeline/main.go --drugs-file node_p01_drugs.csv
```
This creates SAFETY_SIGNAL facts with MedDRA PT, frequency bands from LOINC 34084-4. KB projection routes to KB-4 (safety signals). A separate ETL writes to KB-20 `adverse_reaction_profiles` with `source: "SPL"`, `completeness_grade: "STUB"`.

**Step 2: Derive onset windows from PK data** (automatable -- Fix 5)
```bash
# Extract PK_PARAMETERS from LOINC 34090-1, apply DeriveOnsetWindow()
# Write onset_window + onset_category to KB-20 ADR profiles
# Merge strategy: SPL source retains existing fields, adds onset_window
```
Result: STUB records auto-upgrade to PARTIAL (drug+symptom+onset, missing mechanism and CM rule).

**Step 3: Clinician evidence harvest** (manual, Canon Framework validated)

For each drug-symptom pair relevant to the node:
1. Look up mechanism in StatPearls monograph for the drug class
2. Verify onset window against clinical literature (adjust PK-derived if needed)
3. Author CM rule with calibrated magnitude from epidemiological data
4. Write to KB-20 via `POST /api/v1/pipeline/adr-profiles` with `source: "MANUAL_CURATED"`

Result: PARTIAL records upgrade to FULL (all 4 elements + confidence >= 0.70).

**Step 4: Verify completeness** (automated check)
```bash
GET /api/v1/modifiers/node/{node_id}
# Verify: all drug-symptom pairs have completeness_grade = FULL or PARTIAL
# No STUBs should remain for drugs in active prescribing for the node's patient cohort
```

### 10.2 Per-Node Drug-Symptom Matrix

#### P01 Chest Pain V2

> **Source doc routing note**: P1 source doc §11 routes "Drug ADR map + onset windows → KB-6 (Formulary)." This is outdated. KB-20 `adverse_reaction_profiles` is the canonical ADR store per codebase verification (`adr_service.go`, `adr_profile.go`). KB-6 is formulary only.

| Drug Class | Key ADR | SPL Element 1 | PK Element 3 | Mechanism (Manual) | CM Rule |
|---|---|---|---|---|---|
| ARB | OH, dizziness | telmisartan LOINC 34084-4 | half-life 24h --> Days 1-14 ACUTE | AT1 blockade --> vasodilation --> postural drop. First-dose phenomenon. | CM09: age>=65 + >=3 antihypertensives --> OH odds x1.8 |
| Beta-blocker | Masked hypoglycemia | metoprolol LOINC 34084-4 | half-life 3-7h --> IMMEDIATE | Beta-1 blockade suppresses tremor+tachycardia; sweating preserved (cholinergic) | CM04: BB + SU/Insulin --> flag_masked_hypo; symptom weight modification (palpitations 0.05x, diaphoresis 2.5x) |
| SGLT2i | Volume depletion, DKA | empagliflozin LOINC 34084-4 | half-life 13h --> Days 1-6 acute | Osmotic diuresis + natriuresis --> volume contraction | CM05: SGLT2i + illness/fasting --> activate DX09; prior 0.03 |
| PDE5i | Hypotension (with nitrate) | sildenafil LOINC 34084-4 | half-life 4h --> Hours 1-24 | cGMP-mediated vasodilation + nitrate synergy | CM06: PDE5i --> HARD_BLOCK on nitrate (GO CHANGE G5) |
| Antiplatelet+NSAID | GI bleed | aspirin LOINC 34084-4 | N/A (chronic) | Dual COX inhibition + mucosal damage | CM08: antiplatelet+NSAID --> GERD/GI bleed odds x2.0 |

#### P02 Acute Dyspnea V1

| Drug Class | Key ADR | SPL Element 1 | PK Element 3 | Mechanism (Manual) | CM Rule |
|---|---|---|---|---|---|
| ACEi | Cough, dyspnea | enalapril LOINC 34084-4 | half-life 11h --> Days 1-84 SUBACUTE | Bradykinin accumulation --> airway irritation --> chronic dry cough | CM01: ACEi + cough --> activate DX09; prior 0.08. Suggest ARB switch. |
| Beta-blocker | Bronchospasm, masked hypo | metoprolol LOINC 34084-4 | half-life 3-7h --> IMMEDIATE | Beta-2 blockade --> bronchospasm (non-selective); beta-1 masks tachycardia | CM02: BB + wheezing --> COPD/Asthma odds x1.5; flag if new after BB initiation |
| SGLT2i | DKA (Kussmaul breathing) | empagliflozin LOINC 34084-4 | half-life 13h --> acute; DKA onset Days 1-28 | Ketogenesis from glucosuria + insulin insufficiency | CM03: SGLT2i + illness/fasting --> activate DX10; prior 0.05. Order ketones. |
| Loop diuretic | Volume depletion, decompensation on missed dose | furosemide LOINC 34084-4 | half-life 2h --> IMMEDIATE | Missed doses --> fluid reaccumulation --> ADHF decompensation | CM07: Loop + missed doses --> ADHF odds x2.0. #1 precipitant. |
| PDE5i | Hypotension (with nitrate) | sildenafil LOINC 34084-4 | half-life 4h --> Hours 1-24 | cGMP vasodilation + nitrate synergy | CM08: PDE5i --> HARD_BLOCK on nitrate (GO CHANGE G5) |

### 10.3 Evidence Source Map for Mechanism Harvest

For each drug class in the DM+HTN cohort, the primary StatPearls monograph and supplementary sources:

| Drug Class | StatPearls Monograph | Supplementary PMC/Journal Sources | Key Mechanism Chain |
|---|---|---|---|
| ARB | "Angiotensin II Receptor Blockers" | AJMC 2005, South India OH prevalence study | AT1 blockade --> vasodilation --> SVR reduction --> postural BP drop |
| SGLT2i | "Sodium-Glucose Cotransporter 2 Inhibitors" | CANVAS, EMPA-REG, Renal Fellow Network, Cleveland Clinic JM | Osmotic diuresis + natriuresis --> volume contraction; ketogenesis risk |
| Beta-blocker | "Beta Blockers" | AHA Hypertension, Drooracle pharmacovigilance | Beta-1: bradycardia + hypoglycemia masking; Beta-2: bronchospasm (non-selective) |
| ACEi | "Angiotensin-Converting Enzyme Inhibitors" | ACCP 2006, StatPearls ACEi Cough | Bradykinin accumulation --> airway C-fiber sensitization --> cough |
| Sulfonylurea | "Sulfonylureas" | ADA Standards 2024, UKPDS | Beta-cell insulin secretion --> excessive insulin --> glucose < 70 |
| CCB | "Calcium Channel Blockers" | ACCOMPLISH, Makani 2015 | Peripheral vasodilation (dihydropyridine) --> dependent oedema; negative chronotropy (non-DHP) |
| Thiazide | "Thiazide Diuretics" | ALLHAT, StatPearls Electrolyte Disorders | Na/K/Mg depletion --> electrolyte imbalance --> dizziness/weakness |
| Metformin | "Metformin" | KDIGO 2024, StatPearls Lactic Acidosis | GI intolerance (direct mucosal effect); lactic acidosis (renal accumulation if eGFR < 30) |

### 10.4 Timeline Integration

| Week | SPL Activity | Parallel With |
|---|---|---|
| 1-2 | Run SPLGuard for P01+P02 drug sets (~10 drugs). Create STUB ADR profiles in KB-20. | Tier 1: G1-G3 Go changes + P01 YAML V2 |
| 3-4 | Implement Fix 5 (DeriveOnsetWindow). Auto-upgrade STUBs to PARTIAL. Begin clinician mechanism harvest for P01 CMs. | Week 3-4: CM authoring + pipeline fixes |
| 4-5 | Complete clinician harvest for P01+P02 (10+10 CMs). Verify FULL completeness. | Week 4-5: Phase 0 infrastructure + G6/G7 |
| 5-8 | Run SPLGuard for P4+P5 drug sets. Harvest shared evidence (Metformin B12, CKD-Anaemia). | Phase 1a: P4 Fatigue + P5 Numbness |
| 7-11 | Run SPLGuard for P3 drug sets. CCB ADR profile is critical (oedema mechanism). | Phase 1b: P3 Pedal Oedema |
| 10-13 | Run SPLGuard for P7 drug sets. Heaviest CM overlay (12+ CMs) -- most ADR profiles needed. | Phase 2: P7 GI Disturbance |
| 12-15 | Run SPLGuard for P8 drug sets (SGLT2i UTI/mycosis, neuropathic bladder agents). | Phase 3: P8 Urinary Symptoms |

---

## 11. M2 Bayesian Engine: Counter-Proposal Integration

**Source**: *M2 Bayesian Engine Critical Review & Counter-Proposal (Feb 2026)*. This document identifies 6 structural problems in the Bayesian engine spec and provides concrete fixes. The core log-odds math (§1-§4 of spec) is **SOUND and should ship as specified**. The following subsections capture what MUST change.

### 11.1 Fix: CM Composition in Log-Odds Space (BAY-1, Go Change G14)

**Problem**: `CMApplicator.Apply()` adds author deltas (+0.20) in probability space. When ≥2 CMs fire on the same differential — which is the **default case** in DM+HTN polypharmacy — naive addition breaks:
- Patient on ARB + ≥2 antihypertensives: CM01 (+0.20) + CM09 (+0.20) targeting OH → naive sum: 0.25 + 0.40 = **0.65** before any questions asked
- With 3+ modifiers, sum can exceed 1.0 (mathematically impossible as probability)

**Fix**: Convert deltas to log-odds shifts and apply additively:

```
Step 1: delta_logodds = logit(0.50 + delta_p) − logit(0.50) = ln((0.50 + delta_p) / (0.50 − delta_p))
Step 2: logodds_adjusted = logodds_base + Σ(delta_logodds) for all fired CMs
Step 3: p_adjusted = sigmoid(logodds_adjusted)
```

**Worked example** (from counter-proposal §10):
- OH base prior p0 = 0.25 → logodds_base = ln(0.25/0.75) = −1.099
- CM01 delta +0.20 → delta_logodds = ln(0.70/0.30) = +0.847
- CM07 delta +0.20 → same +0.847
- Combined: logodds_adjusted = −1.099 + 0.847 + 0.847 = **+0.595** → p_adjusted = **0.644**
- Compare naive: 0.25 + 0.20 + 0.20 = 0.65 (similar here but diverges significantly with 3+ modifiers)

**Authoring impact**: Minimal. Authors continue writing `+0.20` deltas in YAML. Engine converts internally using logit-shift. A validation experiment should compare both approaches on 50 synthetic patient profiles spanning the polypharmacy spectrum.

### 11.2 Fix: 'Other' Bucket Differential (BAY-2, Go Change G15)

**Problem**: If the true diagnosis is NOT in the differential list — e.g., horizontal canal BPPV variant, cervicogenic dizziness, medication-induced QT prolongation — the system force-fits evidence to listed differentials. No escape hatch.

**Fix**:
- 'Other' starts at initial prior (p=0.15 recommended)
- Updates using **geometric mean of inverse LRs** across listed differentials: when evidence simultaneously weakens ALL listed diagnoses, Other naturally rises
- If Other posterior > **0.30** → flag `DIFFERENTIAL_INCOMPLETE`: "Available evidence does not strongly support any listed differential. Recommend clinical examination."
- If Other posterior > **0.45** → soft escalation to specialist referral

**Node authoring change**: All differential sets should sum to 0.85 (leaving 0.15 for Other). Existing P1/P2 sets summing to 1.00 need rebalancing (proportional reduction of 15%).

### 11.3 Fix: Pata-Nahi Cascade Protocol (BAY-3, Go Change G16)

**Problem**: Engine handles single PATA_NAHI (c=0, zero update). But elderly DM patients communicating in Hindi via phone routinely give 4-5 consecutive low-confidence answers. The engine reaches question budget with posteriors barely moved from context-modified priors — essentially outputting the PRIOR dressed as a diagnosis.

**Protocol**:

| Consecutive Low-Conf Count (c < 0.3) | Action |
|---|---|
| 2 | Rephrase current question using `alt_prompt` field from YAML (simpler Hindi, different framing) |
| 3 | Switch to **binary-only mode**. Replace multi-option questions with simple yes/no. Recompute question priority for binary format. |
| ≥5 | Terminate with **PARTIAL_ASSESSMENT** flag. Clinician note: "Patient unable to provide reliable symptom history via remote interaction. Clinical examination recommended. Partial assessment based on [N] answered questions and medication context." Output context-modifier-adjusted priors with explicit confidence band: "Low confidence — based primarily on medication profile, not patient-reported symptoms." |
| ≥5 AND any RF among low-conf | **ESCALATE immediately**. Cannot accept low-confidence 'no' to "Do you have weakness on one side of your body?" |

**YAML schema addition**: `alt_prompt` field per question (simpler Hindi rephrasing).

### 11.4 Fix: Contradiction Detection (BAY-4, Go Change G17)

**Problem**: Patients contradict themselves — "haan, khade hone par chakkar aata hai" (AC02) then later "nahi, position se koi farak nahi padta" (DQ03). Bayesian engine applies LR+ then LR−, partially cancelling. Mathematically correct — clinically naive. One answer is wrong; the average is NOT the truth.

**Fix**: `contradiction_pairs` field in YAML schema defines logically related questions:
```yaml
contradiction_pairs:
  - [AC02, DQ03]   # postural relationship ↔ position change trigger
  - [DQ01, DQ02]   # autonomic cluster ↔ sugar relief (both probe hypoglycemia)
```

When contradiction fires:
1. Do NOT apply 2nd LR
2. Re-ask earlier question with confirmation framing: "Aapne pehle bataya tha ki khade hone par chakkar aata hai. Kya yeh sahi hai?"
3. Apply CONFIRMED answer's LR. Discard contradicted.
4. Log `contradiction_event` with both answers, confirmation result, final LR applied → feeds calibration (high rates indicate questions need rewording)

### 11.5 Fix: Three-Tier Calibration Strategy (BAY-9, replaces E01-E05 assumptions)

**Problem**: The plan assumes N=200 adjudicated examples for calibration. At pilot scale (50 patients, ~10 dizziness encounters/month, ~60% adjudication rate), that's **28-50 months**. Two to four years.

**Three-tier strategy** (from counter-proposal §8.2):

| Tier | When | Method | Approver | Logged As |
|---|---|---|---|---|
| **A: Expert Panel** | Month 0-6 | Panel of 3 clinicians (1 diabetologist, 1 internist, 1 GP) reviews each YAML node quarterly. Max ±30% LR adjustment per review cycle. | Panel consensus (2/3 agreement) | `calibration_event` with `source='EXPERT_PANEL'`, panel_members, rationale, version bump |
| **B: Small-Sample Bayesian** | Month 6-18 (N=30-200) | Beta-Binomial shrinkage: `new_lr = w × literature_lr + (1-w) × observed_lr` where `w = max(0.3, 1 − sqrt(n/200))`. Starts heavily anchored to literature (w=0.87 at n=30), gradually releases. | Lead clinician + data scientist | `calibration_event` with `source='BAYESIAN_BLEND'`, sample_size, w_factor, prior_lr, observed_lr, blended_lr |
| **C: Full Data-Driven** | Month 18+ (N>200) | Logistic regression with hierarchical shrinkage priors for rare differentials. | Governance committee, quarterly review | `calibration_event` with `source='DATA_DRIVEN'`, model_version, full regression output |

**Post-deployment KPI targets** (per-node, from respective source documents):

| Metric | P0 Dizziness | P1 Chest Pain | P2 Dyspnea | Action if Off-Target |
|---|---|---|---|---|
| Concordance with physician top-1 | ≥80% (long-term 85-90%) | ≥75% | ≥70% | Review LR weights; check CM firing rates |
| Red flag sensitivity | **100%** | **100%** | **100%** | Any miss → immediate node freeze and review |
| Closure rate (confident differential) | ≥85% | ≥80% | ≥75% | Add questions or adjust threshold |
| Median interaction length | 8-10 questions | 8-12 questions | 9-14 questions | Prune low-utility questions |
| False positive escalation rate | <5% | <10% | — | Raise RF thresholds |

**Why targets differ**: P2 Dyspnea has lower concordance and closure targets because dyspnea differentials overlap more than chest pain or dizziness (e.g., ADHF vs pneumonia vs anaemia share orthopnea/exercise intolerance). P1 allows higher FP escalation because ACS rule-out carries higher liability. P3-P8 targets TBD during evidence harvest — inherit from the most similar existing node as starting point.

### 11.6 Safety Constraint Engine (SCE) Architecture (BAY-6)

**Counter-proposal requirement**: Red flags MUST NOT be inside the Bayesian loop. If M2 crashes, times out, or enters error state, red flag detection is disabled — single point of failure.

**Proposed architecture**: SCE as a **separate Go service** (port 8201):

| Property | M2 Bayesian Engine (KB-22, port 8132 — per CLAUDE.md registry) | Safety Constraint Engine (SCE, port 8201) |
|---|---|---|
| Input | answer_code + answer_confidence | Same (duplicated by KB-19 router) |
| Logic | Log-odds update, question selection, termination | Pattern matching against red flag rules ONLY. No Bayesian math. |
| Output | Updated posteriors + next_question | CLEAR or ESCALATE with reason + evidence |
| If M2 crashes | No posteriors. Session stuck. | **SCE still evaluates red flags independently.** Patient safety maintained. |
| Authority | Recommends diagnostic probabilities | Can **VETO** M2 and force-escalate at any point |
| Code review | Standard engineering review | Requires **clinical safety officer sign-off** for any change |

**Port clarification**: P1 and P2 source documents incorrectly state "KB-22 | Port 8133." Per CLAUDE.md port registry and codebase, **KB-22 HPI Engine = port 8132** and **KB-21 Behavioral Intelligence = port 8133**. Source docs have the ports swapped. This plan uses the correct assignment.

**KB-19 routing**: Every answer routed to BOTH M2 and SCE simultaneously. If SCE returns ESCALATE, KB-19 overrides M2's "ask next question" response and triggers escalation pathway (30 min business hours, 2 hr after-hours, 4 hr failsafe to ER referral).

**Decision required (CC-1)**: Current `SafetyEngine` lives in KB-22 (same process). Counter-proposal says SEPARATE process is non-negotiable. **Options**: (a) Extract to separate service as proposed, (b) keep in KB-22 but add independent health-check watchdog that escalates if KB-22 becomes unresponsive. **Document decision before Sprint 1.** See Section 12 Architectural Decisions Register.

### 11.7 Conflict Arbiter for Multi-Node Sessions (BAY-7)

**Counter-proposal**: Do NOT use Option B (heuristic multiply-down). It suppresses real co-morbid pathology — e.g., dizziness from OH and GI disturbance from metformin intolerance are INDEPENDENT conditions sharing a patient but not a mechanism.

**Proposed**: Independent nodes + Conflict Arbiter in KB-19 (post-processing, NOT a third Bayesian engine):

| Scenario | Detection Rule | Arbiter Action |
|---|---|---|
| Shared diagnosis high in both nodes | Same dx_id in top-3 of both nodes (e.g., Hypoglycemia in Dizziness + GI) | **BOOST**: diagnosis over-determined by independent evidence. Report once with combined evidence. Increase confidence flag. |
| Contradictory evidence | Node A supports dx X, Node B opposes same dx X | **FLAG**: "Conflicting evidence for [dx]. Recommend clinical evaluation to resolve." Do NOT auto-resolve — GP's job. |
| Independent top diagnoses (most common) | No overlap in top-3 | **REPORT BOTH** independently. Normal case for multi-complaint patients. |
| One node red-flags, other doesn't | SCE ESCALATE from Node A, Node B normal | **RED FLAG ALWAYS WINS**. Escalate from Node A immediately. Node B output is supplementary. |

**Decision required (CC-3)**: Plan's B02 lists "Cross-Node Safety Protocol Engine" without specifying mechanism. Counter-proposal explicitly rejects Option B. **Adopt Conflict Arbiter as the B02 implementation.** See Section 12.

### 11.8 Three-Phase Question Selection (BAY-8)

**Counter-proposal**: Full expected information gain (`ComputeExpectedIG()` at `question_orchestrator.go:168`) requires `Pr(answer|dx)` for every (question, answer, differential) triple. This data does NOT exist at launch — it comes from pilot data.

| Phase | When | Method | Data Required |
|---|---|---|---|
| **Phase 1: Author-Ordered** | Launch → 200 patients | Authors rank questions by expected clinical utility. Engine asks in that order, skipping CMs-already-covered questions (G19). | None beyond author judgment. |
| **Phase 2: Greedy Heuristic** | 200+ patients with adjudication | For each candidate question, compute LR variance across current top-3 differentials. Highest variance = most discriminating. | Only LR table from YAML (already available). |
| **Phase 3: Full IG** | 1000+ patients | Estimate Pr(answer\|dx) from observed data. Full expected information gain computation. | Answer frequencies per question per final diagnosis from 1000+ adjudicated encounters. |

**Decision required (CC-2)**: `ComputeExpectedIG()` is already built. Counter-proposal says it should NOT be used at launch. **Options**: (a) Keep IG as default but fall back to author order when Pr(answer\|dx) data is missing (hybrid), (b) disable IG for Sprint 1, use author-ordered only. **Recommend option (a)** — the code exists, just ensure it degrades gracefully when conditional probabilities are absent. See Section 12.

### 11.9 API Contract Additions (BAY-10)

Three endpoints missing from KB-22's current API:

| Endpoint | Method | Purpose |
|---|---|---|
| `/v1/session/escalate` | POST | SCE webhook for force-escalation. Payload: session_id, reason_code, evidence_snapshot, urgency_level. Without this, SCE has no channel to interrupt M2. Escalation Latency SLA starts from this event. |
| `/v1/session/multi-init` | POST | Initialize multiple nodes simultaneously for multi-complaint patient ("mujhe chakkar bhi aata hai aur pet bhi kharab rehta hai"). Links sessions for Conflict Arbiter. |
| `/v1/node/validate` | POST | Pre-deployment YAML validation: schema checks, LR bounds (no LR < 0.01 or > 100 without override), red flag presence verification, simulated walk-through with synthetic profiles. Enables CI/CD for clinical content. |

### 11.10 Kafka Topic Contract for M2 (BAY-11)

| Kafka Topic | Producer | Consumer | Event Type |
|---|---|---|---|
| `hpi.session.events` | M2 Bayesian Engine | KB-19 Protocol Orchestrator, Audit Layer | SessionInitialized, AnswerProcessed, SessionTerminated, ClosureReached |
| `hpi.escalation.events` | Safety Constraint Engine | KB-19 (immediate), Notification Service, Audit Layer | RedFlagDetected, EscalationTriggered, PhysicianNotified |
| `hpi.calibration.data` | M2 (on finalize) | Calibration Pipeline (Flink), ClickHouse | SessionOutcome with full provenance chain |
| `protocol.state.transitions` | KB-19 (existing) | ClickHouse, Neo4j | HPI_COMPLETE state transition with node_id, top_dx, confidence, evidence_count |

### 11.11 ASR-Specific Failure Mode Handling (BAY-5)

These rules are implemented in **M0 NLU layer** (not M2), but M2's behavior depends on them. M0's contract: "Give M2 reliable answer_codes or honestly report uncertainty."

| ASR Failure Mode | Impact | Handling Rule |
|---|---|---|
| Word substitution ('haan' → 'naa') | LR applied WRONG direction. Catastrophic for RF. | RF questions: if c < 0.85, NEVER accept voice. Switch to WhatsApp button (Haan/Nahi). Discriminating: if c < 0.7, apply half-LR (c = c × 0.5). |
| Partial utterance (<500ms cutoff) | Ambiguous; NLU may default to 'yes' | If duration < 500ms AND c < 0.7 → NO_ANSWER (c=0). Re-prompt: "Kripya poora jawab dijiye." |
| Code-mixing (Hindi+English) | ASR model confusion | If code-mixing detected (language-ID toggle within utterance) → confidence floor c = max(c, 0.5). Use bilingual NLU extraction path. |
| Background noise | Consistently low c across ALL answers | If mean(c) across last 3 answers < 0.5 → offer mode switch: "Aapki awaaz saaf nahi aa rahi. Kya aap WhatsApp par type karke jawab de sakte hain?" |

### 11.12 Clinical Source Registry (DIZ-7)

**Missing infrastructure**: The Dizziness Evidence Harvest V2 references a Clinical Source Registry (separate from KB-20 ADR profiles) that tracks LR provenance:

| Table | Purpose |
|---|---|
| `clinical_sources` | Canonical source records: PubMed ID, journal, year, study type, population, quality grade |
| `element_attributions` | Links each YAML field (LR, prior, CM delta) to its source record with confidence level |
| `calibration_events` | Immutable log of every LR/prior adjustment: old_value, new_value, sample_size, deviation, approver, source tier (A/B/C) |

This registry ensures every number in every YAML node has traceable provenance — critical for regulatory compliance and clinical governance. **Build as part of Phase 0 infrastructure sprint (Week 1-2).**

---

## 12. Architectural Decisions Register

Four architectural conflicts between the M2 Bayesian Engine Counter-Proposal and current plan/codebase. **All four decisions resolved (2026-03-10).**

### CC-1: Safety Engine — Same Process vs Separate Service

| | Option A: Separate SCE Service (Counter-Proposal) | Option B: Keep in KB-22 (Current) |
|---|---|---|
| **Safety** | Independent failure domain. If M2 crashes, SCE still catches red flags. | Single failure domain. M2 crash disables safety checking. |
| **Complexity** | New service, new deployment, KB-19 dual routing, separate health checks. | Zero infrastructure change. |
| **Latency** | Network hop adds ~1-5ms per answer routing. | In-process, sub-microsecond. |
| **Recommendation** | Counter-proposal's argument is compelling for production. | Acceptable for pilot if watchdog added. |

**DECISION**: **Option A — Extract SCE to separate service (port 8201).** Counter-proposal's argument for independent failure domain is compelling: if M2 crashes, red flag detection must continue independently. KB-19 routes answers to both M2 and SCE in parallel. SCE runs as sidecar on same host to minimize network latency (~1-2ms). Independent health-check circuit breaker in KB-19 — if SCE unreachable, fall back to inline safety check with degraded-mode alert.

### CC-2: Question Selection — Author-Ordered vs Existing IG

| | Option A: Disable IG, Use Author-Ordered (Counter-Proposal) | Option B: Keep IG with Graceful Degradation |
|---|---|---|
| **Rationale** | IG requires Pr(answer\|dx) which doesn't exist yet. | IG code exists. Can fall back when data absent. |
| **Risk** | Wastes existing engineering. | May give false precision from missing conditional probs. |
| **Recommendation** | **Hybrid (B)**: IG as default, author-order fallback when Pr(answer\|dx)=NULL. Log which mode was used per session for later analysis. |

**DECISION**: **Option B — Hybrid.** Keep existing `ComputeExpectedIG()` as default. When `Pr(answer|dx)` data is absent (NULL), fall back to author-ordered sequence. Log `question_selection_mode: "IG" | "AUTHOR_ORDER"` per session for later analysis. This preserves existing engineering investment while degrading gracefully at pilot scale.

### CC-3: Cross-Node Normalization — Option B vs Conflict Arbiter

| | Option A: Conflict Arbiter (Counter-Proposal) | Option B: Heuristic Multiply-Down (Spec) |
|---|---|---|
| **Safety** | Preserves independent co-morbid pathology. | Suppresses real pathology in polypharmacy patients. |
| **Complexity** | Post-processing layer in KB-19. Medium effort. | Simple multiplication. Low effort. |
| **Recommendation** | **Adopt Conflict Arbiter.** Option B is clinically unsafe per counter-proposal analysis. |

**DECISION**: **Option A — Adopt Conflict Arbiter.** Option B (heuristic multiply-down) suppresses real co-morbid pathology in polypharmacy patients (documented as clinically unsafe). Conflict Arbiter runs as post-processing layer in KB-19 with conservative defaults: REPORT (not BOOST) for shared diagnoses until 500+ multi-node sessions validate BOOST rules. RED_FLAG_WINS is unconditional.

### CC-4: Calibration Timeline — N=200 vs Three-Tier

| | Option A: Three-Tier (Counter-Proposal) | Option B: Wait for N=200 (Spec) |
|---|---|---|
| **Feasibility** | Expert panel immediately actionable. | 28-50 months at pilot scale. |
| **Quality** | Expert+literature anchoring; beta-binomial blending. | Statistically rigorous but requires unavailable data. |
| **Recommendation** | **Adopt three-tier.** Tier A starts Month 0. Build Tier B infrastructure by Month 6. Reserve Tier C for Month 18+. |

**DECISION**: **Option A — Three-tier calibration.** N=200 requirement is infeasible at pilot scale (28-50 months). Tier A (expert panel) starts at Month 0 with quarterly review cycles. Build Tier B infrastructure (beta-binomial shrinkage) by Month 6. Reserve Tier C (full data-driven) for Month 18+ when multi-site pooling provides sufficient data.

### CC-5: Dizziness Doc Contradictions (Resolved)

The Dizziness Evidence Harvest V2 contains three outdated claims:
- **C-1**: "Drug ADR data → KB-6 (Formulary)" — **RESOLVED**: ADR profiles live in KB-20 `adverse_reaction_profiles` per codebase verification. KB-6 is formulary only.
- **C-2/C-3**: "HPI Engine / Bayesian Posterior Engine NOT YET BUILT" — **RESOLVED**: KB-22's `bayesian_engine.go`, `safety_engine.go`, `node_loader.go`, `cm_applicator.go` are running Go code per Section 1.1. The Dizziness doc predates KB-22 implementation.
