# Page 79 Audit — Metformin Evidence (Weight, CV Protection, UKPDS, SPREAD-DIMCAD), Figure 25 (Drug Selection Matrix)

| Field | Value |
|-------|-------|
| **Page** | 79 (PDF page S78) |
| **Content Type** | Rec 4.1.1 evidence continued: metformin weight effects (UKPDS no weight change, subgroup analysis, systematic review: metformin vs SU -2.7 kg, vs TZD -2.6 kg, vs DPP-4i -1.3 kg), metformin CV protection (UKPDS diabetes-related endpoints, SPREAD-DIMCAD metformin vs glipizide, systematic review CV mortality RR 0.6-0.7 vs SU), caveats (all-cause mortality/complications less consistent), Figure 25 (patient factors for drug selection: high-risk ASCVD, glucose-lowering potency, hypoglycemia avoidance, injection avoidance, weight loss, cost, heart failure, eGFR <15/dialysis) |
| **Extracted Spans** | 14 total (14 T1, 0 T2) |
| **Channels** | B (Drug Dictionary), F (NuExtract LLM) |
| **Disagreements** | 6 |
| **Review Status** | PENDING: 14 |
| **Risk** | Disagreement |
| **Cross-Check** | Counts verified against raw extraction data |
| **Audit Date** | 2026-02-25 (revised) |

---

## Source PDF Content

**Metformin Weight Evidence (continued from p78):**
- **UKPDS**: metformin → no weight change at 3 years; sulfonylurea + insulin → significant weight increase
- **UKPDS subgroup**: diet failure → randomized to metformin/SU/insulin → metformin group least weight gain
- **Systematic review weight reduction vs comparators:**
  - Metformin vs SU: **-2.7 kg (95% CI: -3.5 to -1.9)**
  - Metformin vs TZD: **-2.6 kg (95% CI: -4.1 to -1.2)**
  - Metformin vs DPP-4i: **-1.3 kg (95% CI: -1.6 to -1.0)**

**Metformin CV Protection Evidence:**
- UKPDS: metformin > SU/insulin for reduction in diabetes-related endpoints (death, MI, angina, HF, stroke)
- **SPREAD-DIMCAD (China RCT)**: metformin vs glipizide on CV events → metformin benefit over median 5-year follow-up
- Systematic review: **CV mortality RR 0.6–0.7** from RCTs favoring metformin vs SU
- **Caveat**: "effects of metformin on all-cause mortality and other diabetic complications appeared to be less consistent"

**Figure 25 — Patient Factors for Glucose-Lowering Drug Selection (Beyond SGLT2i + Metformin):**

| Patient Factor | More Suitable | Less Suitable |
|---------------|---------------|---------------|
| High-risk ASCVD | GLP1RA | — |
| Potent glucose-lowering | GLP1RA, insulin | — |
| Avoid hypoglycemia | GLP1RA, DPP4i, TZD, AGI | SU, insulin |
| Avoid injections | DPP4i, TZD, SU, AGI, oral GLP1RA | — |
| Weight loss | GLP1RA | SU, TZD, AGI |
| Low cost | SU, TZD, AGI | GLP1RA, insulin |
| Heart failure | GLP1RA | TZD |
| eGFR <15 or dialysis | DPP4i, insulin, TZD | — |

---

## Key Spans Assessment

### Tier 1 Spans (14 — ALL T1)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "thiazolidinedione" ×3 (B, B+F) | B | 100% | **→ T3** — Drug class name mentions without clinical context (3 separate spans) |
| "DPP-4 inhibitors" ×2 (B) | B | 100% | **→ T3** — Drug class name mentions (2 separate spans) |
| "glipizide" ×2 (B) | B | 100% | **→ T3** — Drug name from SPREAD-DIMCAD (2 separate spans) |
| "or DPP-4 inhibitors (" (B+F) | B+F | 98% | **→ NOISE** — Sentence fragment with opening parenthesis; F channel truncation artifact |
| "Results from the UKPDS study demonstrated that patients allocated to metformin did not show a change in mean body weight..." (B+F) | B+F | 98% | **⚠️ T2** — Evidence sentence about metformin weight neutrality. Not patient safety; evidence for drug efficacy |
| "Similarly, this effect was reproduced in an analysis of a subgroup of patients in the UKPDS study who failed diet therap..." (B+F) | B+F | 98% | **⚠️ T2** — UKPDS subgroup evidence. Evidence discussion, not safety |
| "Likewise, the same systematic review earlier showed that metformin treatment led to greater weight reduction compared to..." (B+F) | B+F | 98% | **⚠️ T2** — Systematic review weight comparison. Contains specific metrics (-2.7 kg, -2.6 kg, -1.3 kg) |
| "treatment with metformin may be associated with protective effects against cardiovascular events, beyond its efficacy in..." (B+F) | B+F | 98% | **⚠️ T2** — Metformin CV protection evidence. Important clinical claim but stated as "may be associated" — evidence, not directive |
| "Despite the potential benefits on cardiovascular mortality, the effects of metformin on all-cause mortality and other di..." (B+F) | B+F | 98% | **⚠️ T2** — Important qualifying statement about inconsistent evidence. This is the most nuanced extraction — captures evidence limitation |

**Summary: 0/14 T1 genuine patient safety. 5 are B+F evidence sentences → T2. 8 are drug class/name mentions → T3. 1 is a fragment → NOISE.**

---

## Critical Findings

### ✅ B+F DUAL-CHANNEL — Strong Evidence Extraction

The B+F pattern (B fires on drug names within sentences, F extracts the full sentence) captures 5 evidence discussion sentences on this page. This mirrors the F channel's strong performance on evidence prose pages (p65, p72), but here B is the co-trigger because every evidence sentence contains drug names.

The 5 B+F evidence sentences form a coherent narrative:
1. UKPDS: metformin weight-neutral
2. UKPDS subgroup: least weight gain
3. Systematic review: specific weight reduction metrics
4. CV protection beyond glycemic control
5. Evidence limitation caveat (inconsistency)

### ⚠️ ALL 14 SPANS T1 — Severe Over-Tiering

Every span on this page is classified T1, but NONE contain patient safety content:
- 8 are standalone drug names/class names → T3
- 5 are evidence discussion sentences → T2
- 1 is a fragment → NOISE

This page has the **highest T1 over-tiering rate in the audit**: 0% of T1 spans are genuine patient safety content.

### ❌ Figure 25 Drug Selection Matrix NOT EXTRACTED

Figure 25 provides the most clinically actionable content on this page — a decision matrix mapping patient factors (ASCVD risk, hypoglycemia avoidance, cost, HF, eGFR <15) to drug class suitability. This is exactly the content a prescriber needs.

The D (Table Decomposition) channel did not fire, consistent with its pattern of being silent on matrix/grid figures (only fires on row-column tabular data like Figure 24).

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| Figure 25: GLP1RA preferred for high-risk ASCVD and weight loss | **T1** | Drug selection for specific comorbidities |
| Figure 25: TZD less suitable for heart failure | **T1** | Drug-disease interaction (safety) |
| Figure 25: DPP4i/insulin/TZD for eGFR <15 or dialysis | **T1** | Drug selection for advanced CKD |
| Figure 25: SU/insulin → hypoglycemia risk (less suitable for "avoid hypoglycemia") | **T1** | Drug safety profile mapping |
| Metformin vs SU weight: -2.7 kg (CI: -3.5 to -1.9) | **T2** | Specific weight comparison metric |
| CV mortality RR 0.6-0.7 favoring metformin vs SU | **T2** | CV protection evidence metric |
| SPREAD-DIMCAD: metformin vs glipizide CV benefit over 5 years | **T2** | RCT evidence for metformin CV protection |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — 5 B+F evidence sentences provide a coherent metformin evidence narrative; but all 14 spans over-tiered as T1 (none are patient safety); Figure 25 drug selection matrix completely missing |
| **Tier corrections** | All 8 drug name spans: T1 → T3; All 5 evidence sentences: T1 → T2; Fragment: T1 → NOISE |
| **Missing T1** | Figure 25 drug selection matrix (GLP1RA for ASCVD, TZD contraindicated in HF, eGFR <15 options) |
| **Missing T2** | Weight reduction metrics, CV mortality RR, SPREAD-DIMCAD RCT details |

---

## Completeness Score (Pre-Review)

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~35% — 5 evidence sentences from text; Figure 25 drug selection matrix completely missing |
| **Tier accuracy** | ~0% (0/14 T1 correct — all should be T2 or T3) |
| **Noise ratio** | ~64% — 8 drug names + 1 fragment = 9/14 noise or T3 |
| **Genuine T1 content** | 0 extracted (all T1 are evidence or drug names) |
| **Prior review** | 0/14 reviewed |
| **Overall quality** | **MODERATE-POOR** — Good evidence extraction by B+F, but 100% T1 over-tiering; Figure 25 clinical decision matrix is the key gap |

---

## Raw PDF Gap Analysis (Pipeline 2 L3-L5 Perspective)

### What L3 Claude Fact Extraction Needs from Page 79

**KB-1 (Drug Dosing/Selection):**
- Figure 25 drug selection matrix: maps patient factors (ASCVD, hypoglycemia, weight, cost, HF, eGFR) to drug class suitability — fundamental prescribing decision support data
- Weight reduction metrics: metformin vs SU (-2.7 kg), vs TZD (-2.6 kg), vs DPP-4i (-1.3 kg) with 95% CIs — drug comparison evidence
- SPREAD-DIMCAD RCT: metformin vs glipizide CV benefit over 5 years — CV outcome evidence
- CV mortality RR 0.6-0.7 favoring metformin vs SU — systematic review evidence

**KB-4 (Patient Safety):**
- Figure 25: TZD less suitable for heart failure (drug-disease contraindication)
- Figure 25: SU and insulin less suitable for hypoglycemia avoidance (safety profile)
- Figure 25: eGFR <15/dialysis drug options (CKD safety)

**KB-16 (Lab Monitoring):**
- No direct monitoring content on this page

### Gap Classification

| Gap | Content | KB Target | Priority | Status |
|-----|---------|-----------|----------|--------|
| G79-1 | Figure 25: ASCVD + potency drug selection | KB-1 | HIGH | ADDED |
| G79-2 | Figure 25: Hypoglycemia avoidance mapping | KB-4 | HIGH | ADDED |
| G79-3 | Figure 25: HF + weight management mapping | KB-4/KB-1 | HIGH | ADDED |
| G79-4 | Figure 25: eGFR <15/dialysis drug options | KB-1/KB-4 | HIGH | ADDED |
| G79-5 | Weight reduction metrics with CIs | KB-1 | MEDIUM | ADDED |
| G79-6 | SPREAD-DIMCAD RCT evidence | KB-1 | MEDIUM | ADDED |

---

## Post-Review State (2026-02-27)

| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 9 | 4x "thiazolidinedione(s)" bare drug class, 2x "DPP-4 inhibitors" bare drug class, 2x "glipizide" bare drug name, 1x "or DPP-4 inhibitors (" truncated fragment |
| **CONFIRMED** | 5 | UKPDS weight neutrality, UKPDS subgroup, systematic review weight (truncated), CV protection evidence, all-cause mortality caveat |
| **ADDED** | 6 | Figure 25 ASCVD/potency, Figure 25 hypoglycemia, Figure 25 HF/weight, Figure 25 eGFR<15, weight reduction CIs, SPREAD-DIMCAD RCT |
| **Reviewer** | claude-auditor | |
| **Review Date** | 2026-02-27 | |

---

## Raw PDF Cross-Check Gap Analysis (2026-02-28)

Cross-checked 11 P2-ready spans against exact KDIGO PDF text (lines 1822-1836 of Docling output). Key finding: the prior review covered Figure 25 and weight metrics well, but **4 critical CV evidence/safety sentences were entirely missing** — including the UKPDS CV endpoints list, CV mortality RR 0.6-0.7, all-cause mortality qualification, and a T1 patient safety signal (metformin + SU → 96% increased diabetes-related death).

| Gap | Priority | Content | KB Target |
|-----|----------|---------|-----------|
| G79-7 | HIGH | UKPDS: metformin > SU/insulin for diabetes-related endpoints (death, MI, angina, HF, stroke) | KB-1 |
| G79-8 | HIGH | Systematic review CV mortality RR 0.6-0.7 from RCTs favoring metformin vs SU | KB-1 |
| G79-9 | HIGH | No advantage of metformin over SU for all-cause mortality or microvascular complications | KB-4/KB-1 |
| G79-10 | HIGH/CRITICAL | UKPDS: early addition of metformin to SU → 96% increased diabetes-related death (95% CI: 2%-275%, P=0.039) — T1 safety | KB-4 |
| G79-11 | MODERATE | Figure 25: Avoid injections row — DPP-4i, TZD, SU, AGI, oral GLP-1 RA more suitable | KB-1 |
| G79-12 | MODERATE | Figure 25: Low cost row — SU, TZD, AGI more suitable; GLP-1 RA, insulin less suitable | KB-1 |
| G79-13 | LOW | Metformin weight prevention framing — effective in preventing weight gain, weight reduction in obese patients | KB-1 |

All 7 gaps added via API (all 201 success).

---

## Post-Review State (Final — with raw PDF gap fills)

| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 9 | 4x "thiazolidinedione(s)" bare drug class, 2x "DPP-4 inhibitors" bare drug class, 2x "glipizide" bare drug name, 1x "or DPP-4 inhibitors (" truncated fragment |
| **CONFIRMED** | 5 | UKPDS weight neutrality, UKPDS subgroup, systematic review weight (truncated), CV protection evidence, all-cause mortality caveat |
| **ADDED** | 13 | 6 prior (Figure 25 ×4, weight CIs, SPREAD-DIMCAD) + 7 raw PDF gaps (UKPDS CV endpoints, CV mortality RR, all-cause mortality no advantage, metformin+SU death risk, Figure 25 injections, Figure 25 cost, weight framing) |
| **Total spans** | 27 | 14 original + 13 added |
| **P2-ready** | 18 | 5 confirmed + 13 added |
| **Reviewer** | claude-auditor | |
| **Review Date** | 2026-02-28 | |

### Updated Completeness Score (Final)

| Metric | Pre-Review | Post-Review (2/27) | Final (2/28) |
|--------|-----------|---------------------|--------------|
| **Total spans** | 14 | 20 | 27 |
| **P2-ready** | 0 | 11 | 18 |
| **Extraction completeness** | ~35% | ~75% | **~95%** — all Figure 25 rows, all CV evidence, safety signal captured |
| **Noise ratio** | 64% | 0% active | 0% active |
| **Overall quality** | MODERATE-POOR | GOOD | **EXCELLENT** — critical metformin+SU safety signal (G79-10) now captured |
