# Page 63 Audit — PP 2.2.1 (Safe Lower HbA1c via CGM/Drug Selection), PP 2.2.2 (CGM Metrics as Alternatives), Research Recs

| Field | Value |
|-------|-------|
| **Page** | 63 (PDF page S62) |
| **Content Type** | Rec 2.2.1 rationale continued, PP 2.2.1 (safe achievement of lower HbA1c targets via CGM/SMBG + drug selection), PP 2.2.2 (CGM metrics TIR/TBR as alternatives to HbA1c for glycemic targets), research recommendations (CGM vs HbA1c, lower targets in CKD, dialysis targets) |
| **Extracted Spans** | 18 total (4 T1, 14 T2) |
| **Channels** | C only |
| **Disagreements** | 2 |
| **Review Status** | CONFIRMED: 2, REJECTED: 16, ADDED: 5 (all 18 original PENDING reviewed) |
| **Risk** | Disagreement |
| **Cross-Check** | Verified against pipeline export 2026-02-25 |
| **Audit Date** | 2026-02-25 |
| **Review Date** | 2026-02-27 |
| **Reviewer** | claude-auditor |

---

## Source PDF Content

**Rec 2.2.1 Rationale (Continued):**
- High rate of hypoglycemic events in lower HbA1c range may relate to strategies used, not targets per se

**Practice Point 2.2.1:**
- "Safe achievement of lower HbA1c targets (e.g., <6.5% or <7.0%) may be facilitated by CGM or SMBG and by selection of glucose-lowering agents that are not associated with hypoglycemia"
- CGM and SMBG not biased by CKD or its treatments (dialysis, transplant)
- GMI as proxy for long-term glycemia alongside HbA1c
- GMI useful for advanced CKD/dialysis where HbA1c reliability low

**Practice Point 2.2.2:**
- "CGM metrics, such as time in range and time in hypoglycemia, may be considered as alternatives to HbA1c for defining glycemic targets in some patients"
- HbA1c accuracy/precision similar to general population for eGFR ≥30
- HbA1c accuracy/precision REDUCED for eGFR <30
- CGM can index HbA1c via GMI and adjust targets
- **CGM metrics for clinical care**:
  - TIR: 70–180 mg/dl (3.9–10.0 mmol/l)
  - TBR Level 1: <70 mg/dl (3.9 mmol/l)
  - TBR Level 2: <54 mg/dl (3.0 mmol/l)
- CGM metrics studied most in T1D (greater glycemic variability, higher hypo risk)

**Research Recommendations:**
- Evaluate CGM TIR/mean glucose as alternatives to HbA1c for CKD patients
- Safety of lower glycemic target with non-hypoglycemia-risk agents
- Lower target and CKD progression
- Optimal glycemic targets in dialysis population

---

## Key Spans Assessment

### Tier 1 Spans (4)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Practice Point 2.2.1" (C) | C | 98% | **→ T3** — PP label only (text NOT captured) |
| "Practice Point 2.2.2" (C) | C | 98% | **→ T3** — PP label only (text NOT captured) |
| "HbA1c among patients with CKD and an eGFR ≥30 mL/min/1.73m²" (C) | C | 95% | **⚠️ T2 more appropriate** — eGFR-stratified HbA1c accuracy statement fragment; has disagreement flag; clinically relevant but decontextualized |
| "HbA1c are reduced among patients with CKD and an eGFR <30" (C) | C | 95% | **⚠️ T2 more appropriate** — eGFR-stratified HbA1c precision reduction; has disagreement flag; clinically relevant but decontextualized |

**Summary: 0/4 T1 spans are genuine prescribing/safety content. 2 PP labels → T3. 2 eGFR-stratified fragments are clinically relevant but should be T2 (monitoring accuracy, not prescribing action).**

### Tier 2 Spans (14)

| Category | Count | Assessment |
|----------|-------|------------|
| **"HbA1c"** (C) ×11 | 11 | **ALL → T3** — Lab test name repetition |
| **"180 mg"** (C) | 1 | **→ T3** — Dose/threshold fragment (from TIR 70-180 mg/dl, but without "dl" unit or clinical context) |
| **"70 mg"** (C) | 1 | **→ T3** — Threshold fragment (from TBR <70 mg/dl, without unit) |
| **"54 mg"** (C) | 1 | **→ T3** — Threshold fragment (from TBR <54 mg/dl, without unit) |

**Summary: 0/14 T2 spans are genuine clinical content. HbA1c ×11 + 3 mg-only threshold fragments.**

---

## Critical Findings

### ❌ C CHANNEL ONLY — Zero F Channel, Zero B Channel
This page has only C channel spans. The F channel (NuExtract) does NOT fire on PP 2.2.1 or PP 2.2.2 text, despite both containing clinically important guidance about CGM as an alternative to HbA1c. This is consistent with the pattern: F channel works on evidence summary prose but fails on Practice Point text.

### ❌ PP 2.2.1 and PP 2.2.2 Texts NOT EXTRACTED
Two important Practice Points with only labels captured:
1. PP 2.2.1: Safe lower targets via CGM + non-hypoglycemia drugs
2. PP 2.2.2: CGM metrics (TIR, TBR) as alternatives to HbA1c for targets

### ❌ Glucose Thresholds Extracted as "mg" Fragments
The C channel captures "180 mg", "70 mg", "54 mg" — the TIR/TBR threshold values from the CGM metrics — but strips the "dl" unit and all clinical context. On page 59, the L1 Oracle Recovery channel captured these same thresholds as complete sentences ("70–180 mg/dl (3.9–10.0 mmol/l) at >70% of readings"). The C channel's regex pattern only matches the "number + mg" portion.

### ⚠️ eGFR-Stratified HbA1c Accuracy Fragments — Interesting
Two T1 spans capture eGFR-stratified clinical content:
- "HbA1c among patients with CKD and an eGFR ≥30 mL/min/1.73m²" (accuracy similar to general population)
- "HbA1c are reduced among patients with CKD and an eGFR <30" (accuracy reduced)

These are genuinely useful clinical fragments but should be T2 (monitoring accuracy information, not prescribing safety).

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 2.2.1 text: safe lower targets via CGM + drug selection | **T1** | Target achievement strategy |
| PP 2.2.2 text: CGM metrics as alternatives to HbA1c targets | **T1** | Alternative monitoring for target setting |
| "Hypoglycemic events may relate to strategies used, not targets per se" | **T1** | Critical nuance about target safety |
| TIR target: 70-180 mg/dl at >70% (captured on p59 by L1, not here) | **T1** | CGM interpretation threshold |
| TBR thresholds: <70 mg/dl (Level 1), <54 mg/dl (Level 2) | **T1** | Hypoglycemia safety thresholds |
| "CGM and SMBG not biased by CKD or dialysis/transplant" | **T2** | Monitoring advantage (repeats from p58) |
| Research recommendations (4 items) | **T3** | Research priorities |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — 2 Practice Points missing text; C channel only (no F, no B); glucose thresholds extracted as "mg" fragments without units or context; zero genuine clinical content |
| **Tier corrections** | 2 PP labels: T1 → T3; 2 eGFR fragments: T1 → T2; HbA1c ×11: T2 → T3; "180 mg"/"70 mg"/"54 mg": T2 → T3 |
| **Missing T1** | PP 2.2.1 text, PP 2.2.2 text, TIR/TBR complete thresholds, target safety nuance |
| **Missing T2** | CGM/SMBG CKD advantage, research recommendations |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~5% pre-review; ~65% post-review (5 critical spans added covering PP 2.2.1, PP 2.2.2, rationale, TIR/TBR thresholds, CGM/SMBG CKD advantage) |
| **Tier accuracy** | ~0% (0/4 T1 correct + 0/14 T2 correct = 0/18) |
| **Noise ratio** | ~83% pre-review; 0% post-review (all 16 noise spans rejected) |
| **Genuine T1 content** | 0 extracted; 4 added (PP 2.2.1, PP 2.2.2, target safety rationale, TIR/TBR thresholds) |
| **Prior review** | 0/18 reviewed |
| **Overall quality** | **POOR pre-review; MODERATE post-review** — All noise rejected, 5 critical spans added. Remaining gap: research recommendations (T3, not added as lower priority) |

---

## Post-Review State (2026-02-27)

| Action | Count | Details |
|--------|-------|---------|
| **Confirmed** | 2 | eGFR ≥30 HbA1c accuracy fragment, eGFR <30 HbA1c precision reduction fragment (both noted as T2-appropriate, not T1) |
| **Rejected** | 16 | 2 PP labels (out_of_scope), 11 bare "HbA1c" (out_of_scope), 3 mg-only fragments (out_of_scope) |
| **Added** | 5 | PP 2.2.1 full text, PP 2.2.2 full text, target safety rationale, TIR/TBR complete thresholds, CGM/SMBG CKD advantage |
| **Errors** | 0 | All API calls returned 200/201 |

### Added Span IDs
| Span ID | Content |
|---------|---------|
| c5d31026-99ad-4f09-b1e5-831226c5850e | PP 2.2.1: Safe lower HbA1c targets via CGM/SMBG + drug selection |
| 06c8af99-36c6-43d4-bf1a-9a1104206157 | PP 2.2.2: CGM metrics (TIR/TBR) as alternatives to HbA1c |
| cd2cc8fd-2dfc-4b59-b390-b967f643332f | Rationale: Hypoglycemia relates to strategies, not targets per se |
| 26464135-896c-46b2-9548-a279d31aca0d | TIR/TBR complete threshold definitions with proper units |
| 18603e9d-9c7f-46ec-857c-51b51de7eb8c | CGM/SMBG not biased by CKD or dialysis/transplant |

---

---

## Raw PDF Gap Analysis (Cross-Check 2026-02-27)

| # | Gap Text (Exact PDF) | Priority | Rationale |
|---|---------------------|----------|-----------|
| 1 | "A GMI may be generated as a proxy for long-term glycemia in conjunction with the HbA1c measurement in individual patients, allowing adjustment of glycemic goals accordingly. GMI may commonly be useful for patients with advanced CKD, including those treated with dialysis, for whom the reliability of HbA1c is low." | **MODERATE** | GMI as HbA1c complement for advanced CKD/dialysis where HbA1c unreliable. KB-16. |
| 2 | "on average, HbA1c may be inaccurate for an individual patient and does not reflect glycemic variability and hypoglycemia" | **MODERATE** | HbA1c individual-level limitation justifying CGM/GMI use. KB-16. |
| 3 | "CGM metrics such as time in range and time in hypoglycemia have been studied most often among patients with T1D, who tend to have greater glycemic variability than patients with T2D and are at higher risk of hypoglycemia" | **MODERATE** | CGM evidence limitation, predominantly T1D studies. KB-16. |
| 4 | "Evaluate the value of CGM and metrics such as 'time in range' and mean glucose levels as alternatives to HbA1c level for adjustment of glycemic treatment and for predicting risk for long-term complications in CKD patients with diabetes. Establish the safety of a lower glycemic target when achieved by using glucose-lowering agents not associated with increased hypoglycemia risk. Establish whether a lower glycemic target is associated with slower progression of established CKD. Establish optimal glycemic targets in the dialysis population with diabetes." | **MODERATE** | Four research recommendations: CGM alternatives, lower target safety, CKD progression, dialysis targets. KB-1. |

**All 4 gaps added via API (all 201).**

---

## Post-Review State (Final — with gap fills)

| Metric | Value |
|--------|-------|
| **Total spans (post-review)** | 27 (18 original + 5 agent-added + 4 gap-fill) |
| **Reviewed** | 27/27 (100%) |
| **CONFIRMED** | 2 |
| **REJECTED** | 16 |
| **ADDED (agent)** | 5 |
| **ADDED (gap fill)** | 4 |
| **Total ADDED** | 9 |
| **Pipeline 2 ready** | 11 (2 confirmed + 9 added) |
| **Completeness (post-review)** | ~85% — PP 2.2.1 and PP 2.2.2 full text; TIR/TBR thresholds; CGM/SMBG CKD advantage; target safety rationale; GMI as proxy for advanced CKD/dialysis; HbA1c individual inaccuracy limitation; CGM T1D evidence limitation; all 4 research recommendations; eGFR-stratified HbA1c accuracy fragments |
| **Remaining gaps** | CGM-to-GMI indexing mechanism detail (LOW); specific citation references (T3) |
| **Review Status** | COMPLETE |

---

## Chapter 2 Complete Summary (Pages 56-63)

| Page | Spans | Genuine | Quality | Key Finding |
|------|-------|---------|---------|-------------|
| 56 | 6 | 5 (T2 F channel) | **GOOD** | Best F channel; biomarker limitations |
| 57 | 41 | 0 | **POOR** | HbA1c ×35; worst noise |
| 58 | 40 | ~5 (D channel) | **POOR** | 4 PP labels only; D channel Figure 11 useful |
| 59 | 19 | ~5 (L1 + F) | **MODERATE** | L1 first genuine; 5-channel diversity |
| 60 | 37 | ~2 (eGFR + F) | **POOR** | Rec 2.2.1 text missing; Figure 13 fragmented |
| 61 | 25 | 4 (T1 + C+F) | **GOOD** | Landmark T1 RCT evidence statement |
| 62 | 32 | 7 (F evidence) | **MODERATE** | Evidence fragments; ≤6.0% mortality missing |
| 63 | 18 | 0 | **POOR** | C-only; PP 2.2.1-2.2.2 missing; chapter end |

**Chapter 2 Pattern**: Evidence prose pages (56, 61, 62) work well for F channel. PP-dense pages (57-58, 63) and figure-heavy pages (60) produce noise. The chapter's most important outputs are the landmark T1 on page 61 and the D channel Figure 11 monitoring frequencies on page 58.
