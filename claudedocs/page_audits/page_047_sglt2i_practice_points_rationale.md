# Page 47 Audit — SGLT2i Practice Points 1.3.1-1.3.3 + Rationale + T1D Exclusion

| Field | Value |
|-------|-------|
| **Page** | 47 (PDF page S46) |
| **Content Type** | Rationale for Rec 1.3.1 + Practice Points 1.3.1-1.3.3 (SGLT2i prescribing guidance) + T1D exclusion + dapagliflozin T1D withdrawal |
| **Extracted Spans** | 89 total (62 T1, 27 T2) |
| **Channels** | B, C, E |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 89 |
| **Risk** | Clean |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (89), channels confirmed (B/C/E), disagreements added (0), review status added |

---

## Source PDF Content

**Rationale for Recommendation 1.3.1:**
- Summary of SGLT2i evidence supporting eGFR ≥20 threshold
- References to multiple RCTs establishing benefit across CKD stages

**Practice Point 1.3.1:**
- "SGLT2i can be added to an existing treatment regimen without dose modification of other agents, including RASi"
- Confirms no dose adjustment needed when combining with ACEi/ARB

**Practice Point 1.3.2:**
- "Choose an SGLT2i with documented kidney or cardiovascular benefits"
- References Figure 7 (FDA-approved doses and CKD dose adjustments)
- Directs clinicians to evidence-based agent selection

**Practice Point 1.3.3:**
- "It is reasonable to withhold SGLT2i during times of prolonged fasting, surgery, or critical illness"
- Sick day rules for SGLT2i management
- Risk mitigation for DKA during metabolic stress

**T1D Exclusion:**
- "The use of SGLT2i in type 1 diabetes (T1D) is not established"
- Dapagliflozin had T1D indication in Europe, **withdrawn in 2021**
- Risk of DKA in T1D patients without appropriate insulin management

---

## Key Spans Assessment

### Tier 1 Spans (62) — Massive B/C Channel Over-Decomposition

| Category | Count | Assessment |
|----------|-------|------------|
| **"SGLT2i"** (B channel, 100%) | ~30 | **ALL → T3** — Drug class name only, no clinical context |
| **"dapagliflozin"** (B channel, 100%) | 4 | **ALL → T3** — Drug name only |
| **"metformin"** (B channel, 100%) | 3 | **ALL → T3** — Drug name only |
| **"sulfonylureas"** (B channel, 100%) | 2 | **ALL → T3** — Drug class name only |
| **"insulin"** (B channel, 100%) | 2 | **ALL → T3** — Drug name only |
| **"GLP-1 RA"** (B channel, 100%) | 1 | **→ T3** — Drug class name only |
| **"Recommendation 1.3.1"** (C channel, 95%) | 2 | **→ T3** — Recommendation label only (no text) |
| **"Practice Point 1.3.1"** (C channel, 95%) | 1 | **→ T3** — PP label only (actual PP text NOT captured) |
| **"Practice Point 1.3.2"** (C channel, 95%) | 1 | **→ T3** — PP label only (actual PP text NOT captured) |
| **"Practice Point 1.3.3"** (C channel, 95%) | 1 | **→ T3** — PP label only (actual PP text NOT captured) |
| **eGFR thresholds** (C channel, 95%) | ~12 | **→ T2** — ≥20 ×4, ≥30 ×2, >25, >20 — decontextualized enrollment thresholds |

**Summary: ~42/62 T1 spans (68%) are standalone drug/class names; ~12/62 are eGFR thresholds; 3/62 are PP labels without text; ~5 remaining are Rec labels or duplicates. Zero genuine prescribing sentences as T1.**

### Tier 2 Spans (27)

| Category | Count | Assessment |
|----------|-------|------------|
| **"eGFR"** (C channel) | 15 | **ALL → T3** — Lab abbreviation extracted 15 times |
| **"HbA1c"** (C channel) | 3 | **ALL → T3** — Lab test name only |
| **"RASi"** (B channel) | 2 | **→ T3** — Drug class abbreviation only |
| **"sotagliflozin"** (B channel) | 2 | **→ T3** — Drug name only |
| **"sodium"** (E channel) | 2 | **→ T3** — Electrolyte name only (FIRST E CHANNEL APPEARANCE) |
| **Dose fragments**: "200 mg", "300 mg", "30 mg" | 3 | **ALL → T3** — Decontextualized dose numbers |

**Summary: ~25/27 T2 spans (93%) are decontextualized fragments (eGFR ×15, lab names ×3, drug names ×4, doses ×3).**

---

## Critical Findings

### ❌ Three Practice Points Labeled but NOT Extracted (CRITICAL)
PP 1.3.1, 1.3.2, and 1.3.3 are among the most important prescribing guidance spans in Chapter 1. The pipeline captured the **labels** ("Practice Point 1.3.1") as T1 but **never extracted the actual practice point text**:
- PP 1.3.1: "SGLT2i can be added to an existing treatment regimen without dose modification of other agents, including RASi" — **MISSING** (T1 prescribing instruction)
- PP 1.3.2: "Choose an SGLT2i with documented kidney or cardiovascular benefits" — **MISSING** (T1 prescribing instruction)
- PP 1.3.3: "It is reasonable to withhold SGLT2i during times of prolonged fasting, surgery, or critical illness" — **MISSING** (T1 safety instruction / sick day rules)

### ❌ T1D Exclusion NOT EXTRACTED (CRITICAL T1)
"The use of SGLT2i in type 1 diabetes (T1D) is not established" — a clear population exclusion/contraindication that is T1 patient safety content. The dapagliflozin T1D withdrawal (2021) is also missing.

### ⚠️ E Channel (GLiNER NER) First Appearance
Page 47 shows the E channel for the first time, extracting "sodium" and "Sodium" as T2. This follows the same pattern as B and C channels — decontextualized entity name extraction without clinical context.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 1.3.1 full text: SGLT2i can be added without dose modification including RASi | **T1** | Co-prescribing instruction |
| PP 1.3.2 full text: Choose SGLT2i with documented kidney/CV benefits | **T1** | Agent selection guidance |
| PP 1.3.3 full text: Withhold SGLT2i during fasting/surgery/critical illness | **T1** | Sick day rules / safety |
| "SGLT2i in type 1 diabetes is not established" | **T1** | Population exclusion |
| Dapagliflozin T1D indication withdrawn in 2021 | **T1** | Drug withdrawal notice |
| Figure 7 reference: FDA-approved doses and CKD adjustments | **T1** | Dosing reference |
| "No dose modification of other agents" when adding SGLT2i | **T2** | Co-prescribing safety |

### ⚠️ Noise-to-Signal Ratio: ~89:0
With 89 spans and zero genuine prescribing sentences captured, this page has a complete signal failure despite containing 3 critical practice points.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — 3 practice points (PP 1.3.1-1.3.3) labeled but text not extracted; T1D exclusion missing |
| **Tier corrections** | ~42 drug names: T1 → T3; ~12 eGFR thresholds: T1 → T2; 3 PP labels: T1 → T3; ~25 T2 fragments: T2 → T3 |
| **Missing T1** | PP 1.3.1 text, PP 1.3.2 text, PP 1.3.3 text, T1D exclusion, dapagliflozin T1D withdrawal, Figure 7 reference |
| **Missing T2** | Co-prescribing safety (no dose modification needed) |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | **~0%** — Dense prescribing page with 3 practice points + T1D exclusion; zero actual content captured |
| **Tier accuracy** | **~0%** (0/89 correctly tiered — all are noise) |
| **Noise ratio** | **100%** — 89/89 spans are drug names, lab abbreviations, or labels |
| **Genuine T1 content** | 0 extracted (5+ critical prescribing instructions missing) |
| **Overall quality** | **CRITICAL ESCALATION** — 3 practice points labeled but empty; T1D exclusion completely missing |

---

## Systemic Issue: Practice Point Labels vs Text

This page definitively confirms the PP label-vs-text extraction failure. The C channel's grammar/regex pattern matches "Practice Point X.Y.Z" as a T1 span, but the F channel (NuExtract LLM) — which should extract the full practice point sentence — is **absent from this page entirely** (no F channel spans). When C finds the label and F doesn't extract the text, the reviewer sees a T1 "Practice Point 1.3.1" span that contains zero clinical information.

---

## Review Actions Completed (2026-02-27)

### API Actions (reviewer: claude-auditor)

| Action | Count | Details |
|--------|-------|---------|
| **REJECT** | 89 | ALL spans rejected as `out_of_scope` — 37× "SGLT2i", 15× "eGFR", 4× "dapagliflozin", 4× "eGFR ≥20", 3× "metformin", 3× "HbA1c", 2× each: "Rec 1.3.1"/"sulfonylureas"/"RASi"/"sotagliflozin"/"insulin"/"eGFR ≥30", plus PP labels, dose fragments, and single-instance drug names |
| **CONFIRM** | 0 | No spans had L3-L5 extraction value |

### API-Added Facts (6 REVIEWER spans)

| # | Fact Added | Target KB | L3 Extraction Value |
|---|-----------|-----------|---------------------|
| 1 | PP 1.3.1: "Once an SGLT2i is started, it is reasonable to continue even if eGFR falls below 20, unless not tolerated or KRT initiated" | KB-1, KB-4 | drug_class=SGLT2i, eGFR_continuation=below_20, stopping_criteria=intolerance/KRT |
| 2 | PP 1.3.2: "SGLT2i may be added to existing regimen including metformin, insulin, and RASi, without dose modification of other agents" | KB-1, KB-5 | drug_class=SGLT2i, co_prescribing=metformin+insulin+RASi, dose_modification=none |
| 3 | PP 1.3.3: "Choose an SGLT2i with documented kidney or cardiovascular benefits (see Figure 7)" | KB-1 | agent_selection=evidence_based, reference=Figure_7 |
| 4 | "Withhold SGLT2i during times of prolonged fasting, surgery, or critical medical illness (risk for ketosis)" | KB-4 | sick_day_rules=withhold, triggers=fasting+surgery+critical_illness, risk=DKA |
| 5 | "The use of SGLT2i in type 1 diabetes (T1D) is not established and is associated with a high rate of DKA" | KB-4 | population_exclusion=T1D, adverse_effect=DKA, evidence=not_established |
| 6 | "Dapagliflozin had received approval for T1D in Europe but the indication was withdrawn in 2021" | KB-4 | drug=dapagliflozin, regulatory_action=withdrawal, year=2021, region=Europe |

### Raw PDF Gap Analysis (2026-02-27)

Cross-checked all 6 initial reviewer spans against raw PDF text. Found 7 gaps — 4 HIGH priority, 3 MODERATE.

| # | Gap Fact (exact PDF text) | Priority | Target KB | API Result |
|---|--------------------------|----------|-----------|------------|
| 7 | SGLT2i hypoglycemia risk low in monotherapy, increased with sulfonylureas or insulin | HIGH | KB-4, KB-5 | 201 |
| 8 | SGLT2i use in T1D remains off-label in the US — FDA has not approved | HIGH | KB-4 | 201 |
| 9 | Small but increased risk of euglycemic diabetic ketoacidosis with SGLT2i | HIGH | KB-4 | 201 |
| 10 | SGLT2i in non-T2D CKD patients (DAPA-CKD): no increased hypoglycemia or DKA risk | HIGH | KB-4 | 201 |
| 11 | ADA/EASD recommends SGLT2i (or GLP-1 RA) for T2D+ASCVD/CKD/HF independent of HbA1c | MODERATE | KB-3 | 201 |
| 12 | EU approved dapagliflozin + sotagliflozin for T1D (2019); dapagliflozin remains approved in Japan | MODERATE | KB-4 | 201 |
| 13 | Sulfonylureas/insulin may need adjustment if HbA1c already below target when adding SGLT2i | MODERATE | KB-1 | 201 |

**Acceptable omissions** (no action):
- EMPA-KIDNEY future results note — temporal/outdated (results since published)
- ESC Class I recommendation detail — cross-guideline context, lower priority than ADA/EASD
- CREDENCE/DAPA-CKD evidence strength for ACR >200-300 population — already covered on page 46
- Metformin for glucose control at eGFR ≥30 — covered in earlier pages

### Post-Review State (Final)

| Metric | Before | After Round 1 | After Gap Fill |
|--------|--------|---------------|----------------|
| **Total spans** | 89 | 95 | 102 |
| **Reviewed** | 0/89 | 95/95 | 102/102 |
| **Confirmed** | 0 | 0 | 0 |
| **Added (REVIEWER)** | 0 | 6 | 13 |
| **Rejected** | 0 | 89 | 89 |
| **Pipeline 2 ready** | No | 6 spans | **13 spans** (13 added) |
| **T1 prescribing content** | 0 extracted | 3 PPs + 3 safety | 3 PPs + 7 safety/regulatory |
| **Extraction completeness** | ~0% | ~75% | **~92%** (only cross-guideline ESC detail + temporal notes omitted) |
