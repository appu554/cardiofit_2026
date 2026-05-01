# Layer 1 Implementation Guidelines — Australian Aged Care Clinical Guidelines

**Scope:** This document covers Layer 1 only — the *clinical-knowledge content* the ACOP rule engine consumes. Layers 2 (patient-state plumbing), 3 (rule encoding/CQL), and 4 (pharmacist-facing output) follow as separate documents in the series.

**Audience:** Implementation teams working on the ACOP product wedge into Australian aged care.

**Companion documents:**
- *Vaidshala Phase 3 KB Update Proposal* (with Part 3.5 on CQL coupling)
- *Source-to-KB Routing Map*
- *Implementation Plan Review*
- KB-27 GPCCMP, KB-28 Revenue Assurance, KB-29 Prep Engine — all confirmed real and aged-care-aware

**Author:** Claude (Anthropic), grounded in 2025–26 Australian aged care regulatory and clinical literature.

**Date:** April 2026

---

## Part 0 — Why the ACOP role shapes everything in Layer 1

Before mapping sources to KBs, a clarification that affects every downstream decision: **ACOP and RMMR are not the same workflow, and the knowledge they need is structurally different.**

**RMMR** is a discrete, GP-referred, episodic medication review — one per resident per 12 months (or 24 months under the v3 rules with re-referral exceptions). It produces a structured report sent to the GP. The pharmacist's deliverable is: identify medication-related problems, recommend changes, document rationale, send report. RMMR work is **rule-firing-per-resident-per-review-event**.

**ACOP** is a continuous, embedded, on-site clinical pharmacy role. The DoH ACOP Tier 1 and Tier 2 Rules (current Feb 2026) describe the role as: medication management improvement, clinical governance participation, day-to-day medication review, prompt issue resolution, easy access to pharmacist advice, integration with the GP / nursing / community pharmacy team, training delivery. ACOP is funded at 1 FTE per 250 beds with daily payment ~$619.84 (Tier 1) or salary equivalent (Tier 2). ACOP work is **continuous monitoring + opportunistic intervention + clinical governance contribution**.

**A RACH cannot have both** — ACOP and RMMR/QUM are mutually exclusive funding streams. Once a RACH is in ACOP, RMMR claims are blocked.

The implication for Layer 1: the knowledge base must support both rule-firing patterns. RMMR-style needs explicit, auditable, point-in-time rule evaluation. ACOP-style additionally needs continuous monitoring (lab trends, deterioration signals), trend-aware alerting (eGFR drop *velocity*, not just absolute value), and clinical governance reporting (medication safety incident analysis, Standard 5 evidence, MAC committee inputs). Both share the same knowledge content but consume it through different temporal patterns.

This shapes the Layer 1 schema design that follows.

---

## Part 1 — The ACOP pharmacist workflow, decomposed by knowledge consumption

Before mapping sources, let me decompose the ACOP workflow into the discrete activities that consume Layer 1 knowledge. Each activity is annotated with: who consumes the knowledge, when, and what specific knowledge artifacts are needed.

### Activity 1: Daily medication round review

**When:** Every on-site day, typically 30–60 minutes at the start of shift.
**What the ACOP does:** Reviews any new prescriptions or changes since last on-site day. Identifies issues for same-day intervention.
**Knowledge consumed:**
- Drug-disease contraindications (Beers, Australian PIMs, STOPP)
- Drug-drug interactions (DDI baseline)
- Renal/hepatic dose adjustment thresholds
- Anticholinergic burden (ACB scale) and Drug Burden Index (DBI) scores
- Recent guideline updates that would change a long-standing prescription's appropriateness

**Automation goal:** Generate a "new-events-since-last-visit" worklist with rule-flagged residents at top. Each item ≤30 seconds for ACOP to action.

### Activity 2: New admission medication reconciliation

**When:** Within 7 days of a new resident's admission (per Strengthened Quality Standard 5 expectations).
**What the ACOP does:** Reviews the medication chart against the discharge summary, GP notes, prior pharmacy records. Identifies discrepancies, omissions, inappropriate medications carried over from acute care.
**Knowledge consumed:**
- Full PIM/PPO criteria sets (STOPP/START v3, Australian PIMs, Beers)
- Hospital-to-community discharge medication patterns (e.g., transient PPI started in hospital that was never stopped)
- Deprescribing protocols (AMH Aged Care Companion principles)
- Transition-of-care risk indicators

**Automation goal:** Pre-populate the reconciliation worksheet with: (a) every PIM/PPO flagged on the new chart, (b) every drug whose hospital indication may not apply now, (c) suggested deprescribing candidates. Pharmacist confirms or overrides each.

### Activity 3: Periodic comprehensive medication review (RMMR-equivalent under ACOP)

**When:** Annually per resident, or triggered by clinical change. Under ACOP, this is internal to the on-site role rather than a separate billing event, but the depth of review is the same.
**What the ACOP does:** Full review against complete PIM/PPO criteria, drug-disease, drug-lab, drug-frailty interactions. Produces written recommendations sent to GP. Documents rationale.
**Knowledge consumed:**
- Complete STOPP/START v3 (190 criteria)
- Complete Australian PIMs 2024 list
- Complete AGS Beers 2023
- Complete drug-condition mapping
- Disease-specific deprescribing protocols (e.g., PPI deprescribing, antipsychotic deprescribing in dementia, statin deprescribing in frailty)
- Lab monitoring thresholds (KB-16 CDLs)
- Pregnancy/lactation, hepatic, renal, geriatric-specific dose modifications

**Automation goal:** Produce a structured RMMR-grade review document with every triggered rule, evidence citation, suggested action, and pharmacist confirmation flow. ~5 minutes per resident from start to GP-ready report.

### Activity 4: Adverse drug event monitoring

**When:** Continuous. Triggered by lab results, falls, behavioral changes, hospital admissions.
**What the ACOP does:** Investigates whether a clinical event was medication-related. Documents findings. Recommends preventive action.
**Knowledge consumed:**
- Drug → ADR profiles with onset windows, frequencies, severities (KB-20)
- Causality assessment frameworks (Naranjo Algorithm, WHO-UMC criteria)
- Drug-induced syndromes catalog (serotonin syndrome, NMS, anticholinergic toxidrome, hypoglycemia from sulfonylureas, etc.)

**Automation goal:** When a new event lands (fall, lab abnormality, behavioral change), surface a ranked list of medications that could plausibly explain it, with onset-window plausibility check.

### Activity 5: Quality Use of Medicines (QUM) governance contribution

**When:** Monthly or quarterly. Contribution to Medication Advisory Committee (MAC), incident reviews, training delivery.
**What the ACOP does:** Aggregates facility-level patterns. Identifies systemic issues (e.g., over-reliance on PRN psychotropics, antibiotic stewardship gaps). Produces reports.
**Knowledge consumed:**
- Aged care quality indicators (National Aged Care Mandatory Quality Indicator Program — QI Program)
- Strengthened Quality Standards Standard 5 evidence requirements
- Antimicrobial stewardship principles
- Psychotropic minimisation principles

**Automation goal:** Auto-generated facility-level KPI dashboards mapped to Standard 5, plus monthly MAC report drafts.

### Activity 6: Clinical event response

**When:** Acute. The ACOP is on-site when something happens.
**What the ACOP does:** Real-time advice on dose adjustment, drug stopping, escalation. Liaises with GP and community pharmacy.
**Knowledge consumed:**
- Real-time drug-disease, drug-lab interactions
- Clinical decision limits (KB-16)
- Acute management protocols (e.g., AKI nephrotoxin hold protocols)

**Automation goal:** Sub-second lookup. The ACOP types a drug name + a clinical state and gets an answer.

### Activity 7: Resident/family medication education

**When:** Opportunistic during on-site rounds.
**What the ACOP does:** Explains medications, side effects, deprescribing rationale to residents and families.
**Knowledge consumed:**
- Consumer Medicines Information (CMI) — TGA-registered patient-facing summaries
- Plain-language deprescribing rationale templates

**Automation goal:** Pre-formatted family-facing summaries generated from the underlying clinical recommendation.

### Activity 8: Continuous monitoring (the ACOP-distinctive activity)

**When:** Background, daily.
**What the ACOP does:** Watches lab trends, weight changes, behavioral patterns across the resident population. Detects subtle deterioration before it becomes acute.
**Knowledge consumed:**
- Lab trend interpretation (eGFR drop velocity, weight gain rate, etc.)
- Disease progression patterns
- Deterioration risk signals

**Automation goal:** Streaming alert engine — when a lab/weight/behavioral trend crosses a threshold over a window, surface to ACOP.

### Workflow-to-KB consumption matrix

| Activity | KB-1 | KB-4 | KB-5 | KB-7 | KB-16 | KB-20 | KB-9 | KB-13 | KB-22 | KB-23 |
|---|---|---|---|---|---|---|---|---|---|---|
| 1. Daily round review | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |  |  |  | ✓ |
| 2. Admission reconciliation | ✓ | ✓ | ✓ | ✓ |  | ✓ |  |  |  | ✓ |
| 3. Comprehensive review | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |  |  |  | ✓ |
| 4. ADE monitoring |  | ✓ |  | ✓ | ✓ | ✓ |  |  | ✓ | ✓ |
| 5. QUM governance |  | ✓ |  |  |  |  |  | ✓ |  | ✓ |
| 6. Acute event response | ✓ | ✓ | ✓ | ✓ | ✓ |  |  |  |  | ✓ |
| 7. Family education |  |  |  | ✓ |  | ✓ |  |  |  | ✓ |
| 8. Continuous monitoring |  |  |  | ✓ | ✓ | ✓ | ✓ |  |  | ✓ |

KB-23 (Decision Cards) is the rendering layer for every activity. KB-7 (Terminology) is the foundation for every activity. The other KBs supply the substantive clinical content.

---

## Part 2 — The complete Layer 1 source map

Every source listed here, with the routing decision, the extraction pipeline, the target KB(s), the schema requirement, the licensing posture, and the update mechanism. This is the master spec.

### 2.1 Authoritative explicit-criteria sources (the rule backbone)

These are the sources from which most ACOP rules will be structurally derived. They're explicit, structured, and (mostly) extractable.

#### Source A — STOPP/START v3 (O'Mahony et al., Eur Geriatr Med 2023)

**Authority tier:** 2 (international clinical guideline)
**DOI:** 10.1007/s41999-023-00777-y
**Content:** 190 explicit criteria — 133 STOPP (drugs to stop) + 57 START (drugs to start). Physiological-systems-based, validated by 11-country Delphi panel.
**Licensing:** The criteria themselves are published in an open-access journal (Springer European Geriatric Medicine) and are widely reproduced in clinical literature. The criteria are factual clinical statements not subject to copyright in the same way as prose; the *paper* describing them is copyright Springer but the criteria are freely usable for clinical decision support per multiple licensed implementations (e.g., the CGAKit Plus practice tool, multiple national health service deployments). **Verify with legal counsel before extraction.**

**Routing:**
| Criterion type | Target KB | Fact type |
|---|---|---|
| STOPP criteria with absolute "avoid" rule (e.g., long-term NSAIDs in eGFR <50) | KB-4 | CONTRAINDICATION |
| STOPP criteria with conditional "avoid in" (e.g., benzodiazepines + falls history) | KB-4 | WARNING_SPECIAL_POPULATION |
| STOPP criteria with dose-adjustment guidance | KB-1 | RENAL_OR_HEPATIC_ADJUSTMENT |
| START criteria (omissions: drug X should be considered when condition Y) | KB-4 (new fact_type) | PRESCRIBING_OMISSION |
| Criteria with explicit lab thresholds | KB-16 | CLINICAL_DECISION_LIMIT |
| DDI-flavored criteria | KB-5 | DDI_GUIDELINE_ANCHORED |

**Pipeline:** Pipeline 2 with a **new schema** `KB4ExtractionResult_STOPP_START_v3` extending the existing safety schema with: `criterion_id` (e.g., "STOPP-A1"), `physiological_system` (cardiovascular, renal, etc.), `omission_or_inappropriate` (PIM | PPO), `evidence_strength_v3`. Cost estimate: ~$2-3.

**A new fact_type the existing schema doesn't have: `PRESCRIBING_OMISSION`** (the START half). Schema work needed before extraction. ~half day.

**Update cadence:** ~8-year cycle. v3 published 2023; next version expected ~2030. No need for active monitoring beyond annual check.

#### Source B — Australian PIMs list 2024 (Wang et al., Internal Medicine Journal Feb 2024)

**Authority tier:** 2 (national clinical guideline, Australia-specific)
**DOI:** 10.1111/imj.16322
**Content:** Delphi-developed Australian PIMs list. Two-section structure: (a) medicines to avoid in all older people, (b) medicines to avoid in specific clinical contexts, with safer alternatives.
**Licensing:** Published in Internal Medicine Journal (Wiley). Same legal posture as STOPP/START — the criteria are factual; verify before extraction. The supplementary tables are typically open-access for clinical use.

**Routing:**
| Criterion type | Target KB | Fact type |
|---|---|---|
| "Avoid in all older people" | KB-4 | CONTRAINDICATION (with `age_threshold` ≥65) |
| "Avoid in specific contexts" | KB-4 | CONTRAINDICATION (with condition codes) |
| Safer alternatives | KB-4 (linked) | ALTERNATIVE_RECOMMENDATION |

**Pipeline:** Pipeline 2 with the Australian PIMs schema. The schema can mostly reuse the STOPP/START schema. Cost: ~$1-2.

**Update cadence:** First edition 2024. Likely 5–7 year cycle. Re-check 2027.

#### Source C — AGS Beers Criteria 2023 (American Geriatrics Society)

**Authority tier:** 2 (international clinical guideline, US-origin but globally referenced)
**Content:** ~30 medication categories with explicit "avoid" / "use with caution" / "modify dose" guidance. Last updated 2023.
**Licensing:** Already accessible via OHDSI concept set per your routing map. Beers is the most-implemented PIM tool globally; multiple structured machine-readable representations exist.

**Routing:**
| Beers section | Target KB | Fact type |
|---|---|---|
| "Avoid" recommendations | KB-4 | CONTRAINDICATION |
| "Use with caution" | KB-4 | WARNING_PRECAUTION |
| "Avoid in specific conditions" | KB-4 | CONTRAINDICATION (conditional) |
| "Renal dose modification" | KB-1 | RENAL_DOSE_ADJUSTMENT |
| "Drug-disease interactions" | KB-4 | DRUG_DISEASE_INTERACTION |
| "Drug-drug interactions" | KB-5 | DDI_BEERS_FLAGGED |

**Pipeline:** Already partially loaded via OHDSI vocabulary. **Verification step needed:** confirm the Beers concept set is loaded with sufficient field detail. If only the concept IDs are loaded (drug class memberships) without the specific recommendation text, the actual rules need an additional ingestion step from the AGS publication. Half-day verification, plus possible Pipeline 2 enrichment run if recommendation text is missing.

**Update cadence:** 3-year cycle. Next update expected 2026.

### 2.2 Australian gold-standard prescribing references

#### Source D — AMH Aged Care Companion 2024

**Authority tier:** 1 (Australian de facto standard for aged care prescribing)
**Publisher:** Australian Medicines Handbook Pty Ltd / Pharmaceutical Society of Australia
**Content:** >70 conditions covered with deprescribing principles, condition-specific guidance, age-related pharmacokinetic adjustments. The reference Australian aged care pharmacists trust most.
**Licensing:** **Commercial publication. Requires a licensing agreement with AMH for content extraction rights.** This is non-negotiable — extracting AMH content into a clinical decision support product without a license is a copyright violation regardless of how the product is delivered.

**Recommendation:** Pursue an AMH commercial license. The commercial value of having AMH-grade content in your engine is substantial; the cost of the license is small relative to the lift it gives the product. **Flag this as a P0 commercial action item.**

**If license is obtained:**

| AMH content type | Target KB | Fact type |
|---|---|---|
| Condition management chapters with prescribing recommendations | KB-1, KB-4 | DOSING, CONTRAINDICATION |
| Age-related pharmacokinetic adjustments | KB-1 | GERIATRIC_DOSE_ADJUSTMENT |
| Deprescribing protocols (taper schedules, monitoring requirements) | KB-1, KB-4 | DEPRESCRIBING_PROTOCOL (new fact_type) |
| Drug-frailty interactions | KB-4 | FRAILTY_INTERACTION (new fact_type) |
| Non-drug treatment alternatives | KB-4 (linked) | NON_DRUG_ALTERNATIVE |

**Pipeline:** Pipeline 2 with new schemas for deprescribing protocols and frailty interactions. AMH is a structured publication with consistent chapter formatting, so extraction quality should be high. Cost depends on scope — full extraction of all 70+ chapters: ~$8-12.

**Two new fact types the existing schemas don't have:**
- `DEPRESCRIBING_PROTOCOL` — taper schedule, monitoring frequency, expected withdrawal symptoms, success criteria. Important: this is structurally different from "stop drug X" rules; it's a *workflow* with phases.
- `FRAILTY_INTERACTION` — needs a frailty score input (CFS), drug, condition, modified recommendation. Different from age-based PIM rules because frailty isn't equivalent to age.

**If license is NOT obtained:** the rule engine works on STOPP/START + Australian PIMs + Beers + open-access guidelines. Coverage of ~85% of medication safety surface, but lacks AMH's clinical nuance and brand association. The product is still credible; it's just not "carries AMH content."

**Update cadence:** Print biennial (April), online annual (April). Set up a yearly extraction refresh.

#### Source E — Therapeutic Guidelines: Geriatric (eTG complete)

**Authority tier:** 1 (Australian therapeutic guidelines)
**Publisher:** Therapeutic Guidelines Limited
**Content:** Authoritative Australian therapeutic guidance, geriatric-specific subset.
**Licensing:** **Subscription-based. Same posture as AMH — license needed for extraction.**

**Routing:** Similar to AMH but with stronger emphasis on therapeutic protocols. Maps primarily to KB-1 (dosing), KB-4 (safety), and a potential KB-3 enrichment (treatment pathways).

**Recommendation:** Lower priority than AMH. eTG is excellent but AMH is more directly aged-care-focused. If commercial budget allows both, license both. If only one, AMH first.

**Update cadence:** Continuous online updates.

#### Source F — Australian Medicines Handbook (AMH, the parent reference)

**Authority tier:** 1 (Australian formulary)
**Publisher:** Australian Medicines Handbook Pty Ltd
**Content:** Full Australian drug formulary with prescribing guidance.
**Licensing:** Commercial license required.

**Recommendation:** If you license AMH Aged Care Companion, the parent AMH is usually bundled or co-licensed. Use AMH for general drug information across all ages; use AMH Aged Care Companion for the geriatric-specific layer.

### 2.3 Drug-burden scoring scales

#### Source G — Drug Burden Index (DBI)

**Authority tier:** 2 (validated clinical metric)
**Original publication:** Hilmer et al., Arch Intern Med 2007. Multiple subsequent updates and Australian-specific drug list extensions.
**Content:** Anticholinergic + sedative weighting per drug, computing a cumulative DBI score per patient. Score ≥1 associated with falls, cognitive decline, mortality.
**Licensing:** Open methodology. The drug-weight list is published in academic literature; multiple Australian-specific extensions exist (notably from Monash). Verify that your specific drug-weight list is the most current Australian version.

**Routing:**

| DBI content | Target KB | Fact type |
|---|---|---|
| Drug → DBI weight (anticholinergic component) | KB-20 | DBI_ANTICHOLINERGIC_WEIGHT |
| Drug → DBI weight (sedative component) | KB-20 | DBI_SEDATIVE_WEIGHT |

**Pipeline:** Direct CSV load. No LLM needed. ~30 minutes of work to load + verify against the published Monash list.

**Note:** DBI is a *patient-level computed score*, not a per-rule trigger. The rule engine computes DBI from the patient's medication list and triggers a rule if DBI ≥1 (falls/cognitive risk threshold). The KB stores the per-drug weights; the patient-level computation lives in the rule layer (Layer 3).

**Update cadence:** Stable methodology. Drug list grows as new drugs are added to the Australian formulary; annual review.

#### Source H — Anticholinergic Cognitive Burden (ACB) Scale

**Authority tier:** 2 (validated clinical metric)
**Original publication:** Boustani et al. 2008, with multiple subsequent updates (Salahudeen 2015, etc.). Australian-validated extensions exist.
**Content:** Drug → ACB score (0/1/2/3) with cumulative patient-level score. Score ≥3 associated with cognitive impairment, falls, mortality.
**Licensing:** Open methodology, drug-score list freely available.

**Routing:**

| ACB content | Target KB | Fact type |
|---|---|---|
| Drug → ACB score | KB-20 | ACB_SCORE |

**Pipeline:** Direct CSV load. ~30 minutes.

**Update cadence:** Periodic drug list updates; annual review.

### 2.4 Australian regulatory and program rules

#### Source I — Aged Care Quality Standards (Strengthened, 2026)

**Authority tier:** 1 (statutory)
**Publisher:** Aged Care Quality and Safety Commission
**Content:** Standards 1–7. Standard 5 (Clinical Care) is most ACOP-relevant; Standard 6 (Food and Nutrition), Standard 7 (Residential Community) also relevant.
**Licensing:** Public domain (Australian Government).

**Routing:** This isn't drug knowledge — it's the *evidence framework* the ACOP rule engine must produce evidence against. Routes to KB-28 (Revenue Assurance) for defensibility evidence templates and KB-29 (Prep Engine) for the audit-evidence-bundle generation mode.

**Pipeline:** Manual ingestion. Extract Standard 5 evidence requirements as structured facts: "Standard 5.X requires evidence that…" → mapped to which KBs/queries can supply that evidence.

**Update cadence:** Statutory; track for amendments. Currently in force from 2026.

#### Source J — Aged Care Rules 2025 (s166-112 care and services plan)

**Authority tier:** 1 (statutory)
**Publisher:** Australian Government, Federal Register of Legislation
**Content:** Statutory basis for care plans, reassessment triggers, quality requirements.
**Licensing:** Public domain.

**Routing:** Primarily KB-28 (Revenue Assurance — already references this) and KB-29 (Prep Engine — already references this). Some sections feed KB-13 (Quality Measures) for compliance reporting.

**Pipeline:** Manual ingestion. Already partially done per KB-28/KB-29 documents.

**Update cadence:** Statutory; track for amendments.

#### Source K — National Aged Care Mandatory Quality Indicator Program (QI Program)

**Authority tier:** 1 (mandatory reporting)
**Publisher:** Department of Health, Disability and Ageing / AIHW
**Content:** Required quality indicators every RACF reports quarterly. Includes medication management indicators.
**Licensing:** Public domain.

**Routing:** KB-13 (Quality Measures) — every QI Program indicator becomes a measure definition. Also routes to KB-28 (defensibility) and KB-23 (Decision Cards — facility-level dashboards).

**Pipeline:** Manual ingestion of indicator definitions; runtime computation against patient state.

**Update cadence:** Periodic indicator additions/refinements.

#### Source L — PSA RMMR Guidelines + ACOP Measure Tier 1/Tier 2 Rules

**Authority tier:** 1 (operational/program rules)
**Publishers:** Pharmaceutical Society of Australia (RMMR Guidelines), Department of Health (ACOP Measure Rules), Pharmacy Programs Administrator (operational)
**Content:** Defines the workflow, deliverables, and audit requirements for RMMR and ACOP services.
**Licensing:** Public domain (program rules); PSA Guidelines may have copyright but are freely available for professional use.

**Routing:** Primarily KB-29 (Prep Engine) for output template design — every RMMR/ACOP deliverable specification defines a prep-pack template. Also routes to KB-23 (Decision Cards) for the pharmacist-facing rendering.

**Pipeline:** Manual ingestion of workflow rules; design constraint for KB-29 templates.

**Update cadence:** Periodic updates; ACOP Tier 1/Tier 2 rules updated Feb 2026.

### 2.5 Drug terminology and identifiers (foundation layer)

#### Source M — AMT (Australian Medicines Terminology)

**Authority tier:** 1 (Australian official terminology)
**Publisher:** National Clinical Terminology Service (NCTS) — Australian Digital Health Agency
**Content:** Australian medicines code system. Maps trade names, generic names, dose forms, strengths, packs to canonical codes. RxNorm equivalent for Australia.
**Licensing:** Free for clinical use, requires NCTS account.

**Routing:** **KB-7 (Terminology) — foundation.** Every drug code in every other KB references AMT. Without AMT loaded, code validation fails for Australian-deployed product (just as KB-7 must reference RxNorm for US deployment).

**Pipeline:** Direct ingestion via NCTS monthly bulk download. Same pattern as RxNorm bulk import. ~1 day setup, then automated monthly refresh.

**Update cadence:** Monthly via NCTS.

#### Source N — SNOMED CT-AU

**Authority tier:** 1 (Australian official terminology)
**Publisher:** NCTS
**Content:** Australian extension to SNOMED CT, including aged-care-specific concepts.
**Licensing:** Free for clinical use, NCTS account required.

**Routing:** KB-7 (Terminology) — foundation for condition codes referenced throughout KB-4 (contraindications), KB-11 (cohorts), and elsewhere.

**Pipeline:** NCTS monthly bulk download.

**Update cadence:** Monthly.

#### Source O — LOINC AU subset

**Authority tier:** 1 (Australian official terminology)
**Publisher:** NCTS
**Content:** Australian pathology test codes.
**Licensing:** Free.

**Routing:** KB-7 (Terminology) — every lab in KB-16 references LOINC. **Critical for fixing the LOINC=UNKNOWN problem flagged in earlier reviews** — the 13 unresolved KB-16 entries need this layer populated.

**Pipeline:** NCTS continuous sync.

**Update cadence:** Continuous.

#### Source P — ICD-10-AM (Australian Modification)

**Authority tier:** 1 (Australian hospital coding)
**Publisher:** Independent Health and Aged Care Pricing Authority (IHACPA)
**Content:** Diagnosis codes used in Australian hospital coding.
**Licensing:** Free for clinical use.

**Routing:** KB-7 (Terminology). Relevant when patient state includes hospital discharge data feeding the rule engine.

**Pipeline:** Periodic IHACPA update.

**Update cadence:** Periodic (typically annual).

#### Source Q — PBS (Pharmaceutical Benefits Scheme) Schedule

**Authority tier:** 1 (Australian formulary)
**Publisher:** Department of Health, Disability and Ageing
**Content:** Current PBS items, restrictions, authority requirements, prescriber types, pricing.
**Licensing:** Public domain.

**Routing:** **KB-6 (Formulary)** — primary source for Australian formulary content. Also feeds KB-1 prescribing context (whether a drug requires authority script affects deprescribing/initiation pathways) and KB-23 (Decision Cards — Smart Form integration with PBS criteria pre-populated).

**Pipeline:** Monthly XML/CSV download from PBS website. Direct structured ingestion.

**Update cadence:** Monthly.

#### Source R — TGA Product Information (PI) and Consumer Medicine Information (CMI)

**Authority tier:** 1 (Australian regulatory drug labels)
**Publisher:** Therapeutic Goods Administration
**Content:** Australian equivalent of FDA SPL. Registered indications, contraindications, warnings, dosing per drug.
**Licensing:** Public domain.

**Routing:** Same routing logic as FDA SPL in your existing routing map, just Australian-sourced. Feeds KB-1 (dosing — primary baseline for Australian-registered drugs), KB-4 (contraindications, warnings — primary baseline), KB-5 (per-drug DDI sections), KB-20 (ADRs).

**CMI specifically routes to** the family-education output layer (Activity 7 in the workflow). CMI is plain-language patient-facing content; it's the Australian source for resident/family-facing drug summaries.

**Pipeline:** TGA website scrape — there's no clean API equivalent to DailyMed. This is a real engineering challenge. TGA PI/CMI documents are typically PDF; extraction will need an SPL-Pipeline-equivalent for Australian sources. Reuse the existing SPL Pipeline infrastructure with TGA-specific scraping.

**Update cadence:** Continuous; weekly diff against TGA's product list is a reasonable cadence.

### 2.6 Disease-specific Australian guidelines

These are condition-specific guidelines that supplement the core PIM/PPO sources for specific clinical contexts the ACOP encounters.

#### Source S — Heart Foundation Cardiovascular Disease guidelines

**Authority:** 2 (Australian)
**Routing:** KB-1, KB-4, KB-16 — anti-hypertensive dosing, statin use, anticoagulation thresholds.
**Pipeline:** Pipeline 2 extraction. Cost: ~$1.

#### Source T — Diabetes Australia / ADS-ADEA guidelines

**Authority:** 2 (Australian — Australian Diabetes Society)
**Note:** Your existing 32-dossier ADA SoC 2026 extraction is ADA US, not the Australian ADS guidelines. The two largely align, but Australian-specific guidance (e.g., PBS-aligned medication sequencing) lives in ADS materials.
**Routing:** KB-1, KB-4, KB-16, KB-20 — diabetes management in older adults, hypoglycemia thresholds, deprescribing as frailty progresses.
**Pipeline:** Pipeline 2. Cost: ~$1.

#### Source U — Kidney Health Australia / KHA-CARI guidelines

**Authority:** 2 (Australian, KDIGO-aligned but with Australian-specific overlays)
**Routing:** KB-1 (renal dosing), KB-16 (eGFR thresholds), KB-4 (contraindications). Largely aligned with KDIGO 2024 already in your CDL seed; overlay for Australian-specific recommendations.
**Pipeline:** Pipeline 2. Cost: ~$1.

#### Source V — Psychotropic Expert Group / RANZCP psychotropic guidelines for older adults

**Authority:** 2 (Australian psychiatry — Royal Australian and New Zealand College of Psychiatrists)
**Content:** Antipsychotic, antidepressant, anxiolytic prescribing in older adults. Critical for Rules 2 (sedatives + falls) and 8 (antipsychotic in dementia) of your initial 20-rule set.
**Routing:** KB-4, KB-20 — psychotropic-specific contraindications, deprescribing protocols, ADR profiles.
**Pipeline:** Pipeline 2. Cost: ~$1.

#### Source W — Australian Antimicrobial Stewardship guidelines (Therapeutic Guidelines: Antibiotic + ACSQHC AMS standards)

**Authority:** 2 (Australian)
**Content:** Antibiotic prescribing for older adults, course-length guidance, AMS principles.
**Routing:** KB-4 (Rule 18 — prolonged antibiotic use), KB-13 (QUM antimicrobial indicator).
**Pipeline:** Therapeutic Guidelines is licensed; AMS standards are public. Mixed approach.

#### Source X — Royal Commission into Aged Care Quality and Safety Recommendations

**Authority:** 1 (foundational policy basis)
**Content:** Recommendation 38 (the basis for ACOP) and other medication-management recommendations.
**Routing:** KB-13 (Quality Measures — recommendations operationalized as indicators), KB-28 (defensibility — Royal Commission alignment as an evidence dimension).
**Pipeline:** Manual ingestion of relevant recommendations.

### 2.7 Sources for specific failure-mode analysis

These are the empirical-completion and adversarial-completion sources from the methodology in my previous response.

#### Source Y — Coronial findings (state coroners)

**Authority:** Variable (case-by-case)
**Content:** Public coronial findings related to medication-related deaths in aged care.
**Routing:** Not direct KB content — used as input to Approach C (failure-mode rule generation). Each finding may surface a rule that should fire to prevent recurrence.
**Pipeline:** Manual review with structured templating.

#### Source Z — Aged Care Quality and Safety Commission audit reports

**Authority:** Public regulator findings
**Content:** Aggregated and individual facility audit findings.
**Routing:** Same as coronial findings — input to Approach C and to KB-13 indicator design.
**Pipeline:** Manual review.

#### Source AA — Published Australian RMMR analyses (e.g., Ramsey 2025, Frontiers in Pharmacology)

**Authority:** Academic
**Content:** Empirical distributions of what Australian pharmacists actually flag in RMMRs.
**Routing:** Input to Approach B (empirical validation) and to prioritization tuning.
**Pipeline:** Literature review with structured extraction.

---

## Part 3 — Sequencing the Layer 1 implementation

Six waves, ordered by leverage and dependency.

### Wave 1 — Foundation terminology (Week 1, blocking everything else)

Goal: KB-7 populated for Australian deployment.

| Task | Source | Effort | Cost | Dependency |
|---|---|---|---|---|
| Set up NCTS account and bulk-download pipeline | NCTS | 1 day | $0 | Anthropic API key not needed |
| Load AMT bulk into KB-7 | Source M (AMT) | 0.5 day | $0 | NCTS account |
| Load SNOMED CT-AU into KB-7 | Source N | 0.5 day | $0 | NCTS account |
| Load LOINC AU into KB-7 | Source O | 0.5 day | $0 | NCTS account |
| Load ICD-10-AM into KB-7 | Source P | 0.5 day | $0 | IHACPA download |
| Set up monthly refresh cron | All NCTS sources | 0.5 day | $0 | All above loaded |
| Verify KB-7 resolves all codes referenced by existing KB-1/4/16/20 ADA-derived rows | Verification | 1 day | $0 | All above loaded |

**Wave 1 exit criterion:** Every drug code, condition code, lab code in any existing KB row resolves via KB-7. **The LOINC=UNKNOWN issue is closed in this wave.**

### Wave 2 — Australian formulary and drug labels (Week 2)

Goal: KB-6 (formulary) populated with PBS; KB-1/KB-4/KB-5/KB-20 supplemented with TGA PI/CMI for Australian-registered drugs.

| Task | Source | Effort | Cost |
|---|---|---|---|
| PBS Schedule monthly bulk loader | Source Q (PBS) | 1 day | $0 |
| Build TGA PI/CMI scraper (adapter on existing SPL Pipeline) | Source R (TGA) | 2-3 days | $0 |
| Run TGA PI extraction on top 100 RACF drugs | Source R | 1 day runtime | ~$2-3 |
| Run CMI extraction for family-facing summaries | Source R | 1 day runtime | ~$1 |

**Wave 2 exit criterion:** Australian regulatory baseline established for KB-1, KB-4, KB-5, KB-6, KB-20.

**Note:** This is the wave where the constitutional DDI projection from your earlier action queue should also run, populating KB-5 with the 2,527 unprojected DDI definitions. Sequence it within Wave 2.

### Wave 3 — Explicit-criteria rule sources (Weeks 3–4)

Goal: STOPP/START v3, Australian PIMs 2024, Beers 2023 all extracted and live as governance-tracked KB rows.

| Task | Source | Effort | Cost |
|---|---|---|---|
| Author `KB4ExtractionResult_STOPP_START_v3` schema (extends existing KB-4 schema with new fact types `PRESCRIBING_OMISSION`) | Schema | 0.5 day | $0 |
| Author Australian PIMs 2024 schema (largely reuses STOPP/START schema) | Schema | 0.25 day | $0 |
| Run Pipeline 2 extraction on STOPP/START v3 paper + supplementary tables | Source A | 1 day runtime | ~$2-3 |
| Run Pipeline 2 extraction on Australian PIMs 2024 paper + supplementary | Source B | 0.5 day runtime | ~$1-2 |
| Verify Beers 2023 OHDSI concept set has full recommendation text; if not, run Pipeline 2 enrichment | Source C | 1 day | ~$1 if needed |
| L6 loader runs through governance for all extracted rules | All | 0.5 day | $0 |
| CompatibilityChecker pass — every extracted rule should have a candidate CQL define already in tier-4-guidelines/au/ | Verification | 0.5 day | $0 |

**Wave 3 exit criterion:** ~350 explicit PIM/PPO rules in KB-4 with full provenance, governance audit trail, and CQL-ready structure.

### Wave 4 — Drug-burden scoring scales (Week 4, parallel with Wave 3)

Goal: DBI and ACB scores loaded as patient-state-computable per-drug references.

| Task | Source | Effort | Cost |
|---|---|---|---|
| Author KB-20 schema extension for DBI weights | Schema | 0.25 day | $0 |
| Author KB-20 schema extension for ACB scores | Schema | 0.25 day | $0 |
| CSV load of DBI weights (Australian Monash extension) | Source G (DBI) | 0.25 day | $0 |
| CSV load of ACB scores | Source H (ACB) | 0.25 day | $0 |
| Verify DBI/ACB drugs map to AMT codes via KB-7 | Verification | 0.25 day | $0 |

**Wave 4 exit criterion:** Patient-level DBI and ACB scores can be computed at runtime from a medication list.

### Wave 5 — Australian gold-standard prescribing references (Weeks 5–8, gated on licensing)

Goal: AMH Aged Care Companion (and possibly eTG Geriatric) extracted into KB-1/KB-4 with deprescribing protocols and frailty interactions.

**This wave is gated on commercial licensing decisions.** If licenses aren't secured, skip this wave; the product is still credible without it.

| Task | Source | Effort | Cost | Gate |
|---|---|---|---|---|
| Pursue AMH commercial license | — | Commercial timeline | License fee | **Required for extraction** |
| Author `DEPRESCRIBING_PROTOCOL` fact type schema | Schema | 0.5 day | $0 | License obtained |
| Author `FRAILTY_INTERACTION` fact type schema | Schema | 0.5 day | $0 | License obtained |
| Pipeline 2 extraction on AMH Aged Care Companion 2024 (~70 chapters) | Source D | 2-3 days runtime | ~$8-12 | License obtained |
| Pipeline 2 extraction on eTG Geriatric (if licensed) | Source E | 2-3 days runtime | ~$5-8 | License obtained |
| L6 loader + governance | All | 1 day | $0 | Above |
| CQL define authoring for AMH-derived deprescribing protocols | CQL | 3-5 days | $0 | Above |

**Wave 5 exit criterion:** AMH-grade deprescribing and frailty content live in KBs.

### Wave 6 — Australian disease-specific guidelines (Weeks 6–10, parallel)

Goal: Disease-specific guidelines for the conditions ACOPs encounter most.

| Task | Source | Effort | Cost |
|---|---|---|---|
| Pipeline 2 on Heart Foundation CV guidelines | Source S | 0.5 day runtime | ~$1 |
| Pipeline 2 on ADS-ADEA diabetes guidelines (Australian) | Source T | 0.5 day runtime | ~$1 |
| Pipeline 2 on KHA-CARI renal guidelines | Source U | 0.5 day runtime | ~$1 |
| Pipeline 2 on RANZCP psychotropic guidelines | Source V | 0.5 day runtime | ~$1 |
| Therapeutic Guidelines: Antibiotic (if licensed) | Source W | 0.5 day | ~$1 |
| Manual ingestion of Strengthened Quality Standards Standard 5 evidence requirements | Source I | 1 day | $0 |
| Manual ingestion of QI Program indicator definitions | Source K | 1 day | $0 |

**Wave 6 exit criterion:** Disease-specific Australian context layered onto the explicit-criteria backbone.

### Total Layer 1 implementation

| Wave | Effort | API cost | License cost | Output |
|---|---|---|---|---|
| 1. Foundation terminology | 4-5 days | $0 | NCTS free | KB-7 populated |
| 2. Australian formulary + drug labels | 4-5 days | ~$3-4 | $0 | KB-1/4/5/6/20 Australian baseline |
| 3. Explicit-criteria rules | 4-5 days | ~$4-6 | $0 | ~350 PIM/PPO rules |
| 4. DBI + ACB | 1 day | $0 | $0 | Drug-burden scoring active |
| 5. AMH/eTG licensed content | 8-12 days | ~$13-20 | License fee | Australian gold-standard layer (gated) |
| 6. Disease-specific Australian | 4-6 days | ~$5-7 | possibly some | Layered context |
| **Total** | **25-34 days** | **~$25-37** | **License-dependent** | **Layer 1 complete for ACOP** |

This is roughly 5–7 weeks of focused engineering work, $25-37 in API spend (well within the budget already proven in the ADA work), plus commercial licensing decisions for AMH/eTG/Therapeutic Guidelines.

---

## Part 4 — The new schemas Layer 1 needs

The existing Pipeline 2 schemas (KB1ExtractionResult / KB4ExtractionResult / KB16ExtractionResult / KB20ExtractionResult / KB5InteractionsResult) cover most of the ACOP work, but five new fact types and three new schema variants are required.

### 4.1 New fact types

| Fact type | Where it lives | What it captures | Required by source |
|---|---|---|---|
| `PRESCRIBING_OMISSION` | KB-4 | Drug X *should be* considered when condition Y; absence = quality issue. The START half of STOPP/START. | STOPP/START v3, AMH |
| `DEPRESCRIBING_PROTOCOL` | KB-1 (or new KB) | Structured taper schedule with phases, monitoring requirements, expected withdrawal symptoms, success criteria. Different from "stop drug X" — it's a *workflow*. | AMH Aged Care Companion |
| `FRAILTY_INTERACTION` | KB-4 | Drug + frailty score + condition → modified recommendation. Different from age-based PIMs because frailty isn't a function of age alone. | AMH, STOPP/START v3 (some criteria), AGS Beers (some) |
| `DRUG_BURDEN_WEIGHT` | KB-20 | Drug → DBI weight (anticholinergic + sedative components) | DBI source |
| `ANTICHOLINERGIC_BURDEN_SCORE` | KB-20 | Drug → ACB score (0/1/2/3) | ACB Scale |

### 4.2 New schema variants

**Variant 1: `KB4ExtractionResult_STOPP_START_v3`** — extends the existing safety schema with criterion ID, physiological system, omission/inappropriate flag, evidence strength.

**Variant 2: `KB4ExtractionResult_AustralianPIMs_2024`** — extends safety schema with Australian PIMs section type (avoid in all / avoid in context), safer alternative drug references.

**Variant 3: `KB1ExtractionResult_DeprescribingProtocol`** — new schema for taper schedules. Fields: drug, indication being deprescribed, taper phases (each with duration, dose, monitoring), expected withdrawal symptoms, success criteria, fallback if symptoms emerge.

### 4.3 Schema authoring effort

About 2 days total to author all five new fact types and three new schema variants, plus update the prompt branches in `fact_extractor.py`. Same pattern as the existing schemas; mechanical work.

---

## Part 5 — Mapping ACOP activities to KB queries

This section closes the loop: every ACOP workflow activity, what KB queries it generates, what data flows into Layer 4 (the pharmacist-facing output).

### Activity 1: Daily medication round review

**Trigger:** ACOP arrives on-site; system detects last on-site visit timestamp; pulls all events since.

**Queries:**
1. `KB-4 WHERE drug_rxcui IN patient.current_meds AND condition_code IN patient.diagnoses` — surface PIMs
2. `KB-5 WHERE pair_a_rxcui IN patient.current_meds AND pair_b_rxcui IN patient.current_meds` — surface DDIs
3. `KB-1 WHERE drug_rxcui IN patient.current_meds AND egfr_min > patient.current_egfr` — surface dose-adjustment-needed
4. Compute patient.DBI from KB-20 weights × patient.current_meds
5. Compute patient.ACB from KB-20 scores × patient.current_meds
6. Surface if DBI ≥ 1 OR ACB ≥ 3

**Output:** KB-23 Decision Card per resident — top 3 medication concerns with specific actions.

### Activity 2: New admission medication reconciliation

**Trigger:** New resident admission event.

**Queries:**
- All KB-4 PIM rules against the new med chart
- All KB-1 dose-adjustment rules against the patient's renal/hepatic state
- KB-20 transition-of-care risk indicators (drugs commonly carried over inappropriately from acute care, e.g., transient PPI, transient opioid)
- KB-1 deprescribing protocols (Wave 5 content) for any drug whose hospital indication may not apply now

**Output:** Reconciliation worksheet template (KB-29 prep pack mode) pre-populated with flags.

### Activity 3: Comprehensive medication review (RMMR-equivalent under ACOP)

**Trigger:** Annual review or clinical-change-triggered review.

**Queries:** Full sweep of all KB-4 PIM rules, all KB-1 dosing rules, all KB-5 DDI rules, all KB-16 lab thresholds, all KB-20 ADR profiles, plus AMH deprescribing protocols if licensed.

**Output:** RMMR-grade report (KB-29 prep pack mode `AGED_CARE_COMPREHENSIVE_REVIEW`).

### Activity 4: Adverse drug event monitoring

**Trigger:** New event detected (lab abnormality, fall, behavioral change, hospital admission).

**Queries:**
- KB-20 ADR profiles WHERE reaction matches event AND drug_rxcui IN patient.current_meds
- Onset window plausibility check (event date vs drug start date)
- KB-22 (HPI Engine — derived from KB-20) Bayesian posterior for drug-causality given event

**Output:** Ranked list of medications that could plausibly explain the event.

### Activity 5: QUM governance contribution

**Trigger:** Monthly or quarterly.

**Queries:**
- KB-13 (Quality Measures) — facility-level computation across all residents
- Standard 5 evidence aggregation per KB-28 evidence templates
- Trend analysis — has psychotropic prescribing ↑ or ↓ over the period?

**Output:** MAC committee report draft (KB-29 prep pack mode), facility QI dashboard.

### Activity 6: Acute event response

**Trigger:** Real-time clinical event.

**Queries:** Sub-second lookup against KB-1, KB-4, KB-5, KB-16 by drug + clinical state.

**Output:** Single-screen clinical answer.

### Activity 7: Resident/family education

**Trigger:** Pharmacist requests family-facing summary for a recommendation.

**Queries:**
- TGA CMI (Source R extracted to KB-20 plain-language layer) for the relevant drug
- KB-23 Decision Card rendering with family-facing template

**Output:** Plain-language summary, printable.

### Activity 8: Continuous monitoring

**Trigger:** Streaming event (lab landed, weight measured, behavior reported).

**Queries:**
- KB-16 CDLs against new lab value
- Trend computation: is patient.lab_X moving past a threshold *velocity* (not just absolute)?
- KB-9 (Care Gaps) computation: has required monitoring been done?

**Output:** Push notification to ACOP if threshold crossed; otherwise silent monitoring.

---

## Part 6 — Risks and dependencies for Layer 1

Three risks worth naming explicitly before Wave 1 starts.

**Risk 1: AMH/eTG/Therapeutic Guidelines licensing timeline.** Commercial licensing can take weeks to months. If Wave 5 is on the critical path for product launch, the licensing conversation should start now, in parallel with Waves 1-4. If Wave 5 isn't on the critical path, the product launches without it and adds it later.

**Risk 2: TGA PI/CMI scraping fragility.** Unlike DailyMed, TGA doesn't have an RSS feed. The scraper from Source R will be the most engineering-heavy piece of Layer 1. Allocate 2-3 days for scraper development and expect to spend ongoing time on maintenance as TGA updates its publication formats. Consider whether commercial alternatives (MIMS, AusDI) with proper APIs are worth the licensing cost as an alternative source.

**Risk 3: STOPP/START v3 and Australian PIMs 2024 copyright posture.** The criteria themselves are factual and widely implemented in clinical decision support tools, but extraction and redistribution may have copyright considerations. **Verify with legal counsel before Wave 3 extraction.** Mitigation: even if extraction is restricted, *referencing* the criteria (citing them, then encoding them as your own rule statements) is universally accepted practice.

**Dependency: KB-7 must be populated before Waves 2–6.** Every other wave depends on KB-7 for code resolution. Wave 1 cannot slip without slipping everything else.

**Dependency: NCTS account approval timeline.** NCTS accounts are typically approved within 1-2 weeks. Apply at Wave 0 (immediately) so the account is ready when Wave 1 starts.

---

## Part 7 — One-page summary

**What Layer 1 provides:** The clinical-knowledge content the ACOP rule engine consumes — drug rules, safety signals, DDIs, lab thresholds, ADR profiles, drug-burden scores, all routed to the appropriate KB with full provenance and Australian-localized.

**Sources, by priority:**

1. **Foundation terminology (NCTS-sourced):** AMT, SNOMED CT-AU, LOINC AU, ICD-10-AM. Free, blocking, must come first.
2. **Australian regulatory baseline:** PBS Schedule, TGA PI/CMI. Free, second.
3. **Explicit-criteria backbone:** STOPP/START v3, Australian PIMs 2024, Beers 2023. Open-access (verify licensing). Third.
4. **Drug-burden scoring:** DBI, ACB. Open methodology. Fourth (parallel with #3).
5. **Australian gold-standard:** AMH Aged Care Companion, eTG Geriatric. Licensed. Fifth (gated on commercial agreement).
6. **Disease-specific:** Heart Foundation, ADS-ADEA, KHA-CARI, RANZCP, ACSQHC AMS. Mostly free. Sixth (parallel).

**Routing summary:**
- KB-7 ← AMT, SNOMED CT-AU, LOINC AU, ICD-10-AM
- KB-1 ← STOPP/START, Australian PIMs, Beers (renal/hepatic), AMH (deprescribing protocols, gated), TGA PI, disease-specific
- KB-4 ← STOPP/START, Australian PIMs, Beers (PIMs), AMH (frailty interactions, gated), TGA PI, disease-specific
- KB-5 ← Beers DDI section, TGA PI DDI, constitutional DDI projection (already pending)
- KB-6 ← PBS Schedule
- KB-16 ← Already covered by Migration 002 7-authority seed; ADA → ADS supplements where Australia differs
- KB-20 ← DBI weights, ACB scores, TGA CMI (family-education layer), TGA ADR sections
- KB-13 ← QI Program indicators, Royal Commission recommendations operationalized
- KB-23 ← All of the above (rendering layer)
- KB-28 ← Strengthened Quality Standards, Aged Care Rules 2025 (already partial)
- KB-29 ← PSA RMMR Guidelines, ACOP Measure Rules, Standard 5 evidence requirements (already partial)

**Total cost:** ~5-7 weeks engineering, ~$25-37 in Claude API spend, plus AMH/eTG license fees if pursued.

**Critical path:** Wave 1 (KB-7 foundation) blocks everything. Start NCTS account application immediately. Pursue AMH licensing in parallel.

**Output for ACOP product:** Layer 1 complete means the rule engine has the substantive clinical knowledge for all 8 ACOP workflow activities, in Australian-localized form, with full governance trail. Layers 2-4 then plug in patient state, rule encoding, and pharmacist-facing output to complete the product.

---

*Next document in series: Layer 2 — Patient-state plumbing for ACOP (CSV ingestion → eNRMC integration → pathology integration → frailty/palliative status capture).*

— Claude
