# Page 54 Audit — PP 1.4.4 (Finerenone Dosing), PP 1.4.5 (Steroidal MRA for HF), MRA Research, Rec 1.5.1 (Smoking)

| Field | Value |
|-------|-------|
| **Page** | 54 (PDF page S53) |
| **Content Type** | PP 1.4.4 (nonsteroidal MRA agent selection + finerenone dosing by eGFR), PP 1.4.5 (steroidal MRA for HF/hyperaldosteronism/refractory HTN), MRA research recommendations, Section 1.5 Smoking Cessation, Rec 1.5.1 (quit tobacco, 1D) |
| **Extracted Spans** | 28 total (18 T1, 10 T2) |
| **Channels** | B, C, F |
| **Disagreements** | 6 |
| **Review Status** | EDITED: 2, PENDING: 26 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (28), channels confirmed (B/C/F), disagreements added (6), review status added |

---

## Source PDF Content

**Practice Point 1.4.4:**
- "The choice of a nonsteroidal MRA should prioritize agents with documented kidney or cardiovascular benefits"
- Currently, only finerenone has rigorous long-term outcome data
- **Finerenone dosing**: 20 mg daily if eGFR ≥60; 10 mg daily if eGFR 25-59; uptitrate to 20 mg if K+ ≤4.8 mmol/l
- Steroidal MRA do NOT have documented clinical kidney/CV benefits, except when heart failure is present

**Practice Point 1.4.5:**
- "A steroidal MRA should be used for treatment of heart failure, hyperaldosteronism, or refractory hypertension, but may cause hyperkalemia or a reversible decline in GFR, particularly among patients with a low GFR"
- Steroidal MRA = standard of care for HF (reduced ejection fraction) and primary hyperaldosteronism
- Useful for refractory hypertension
- **No evidence that switching steroidal → nonsteroidal MRA will improve outcome**
- **Adding nonsteroidal MRA to steroidal MRA likely increases adverse effects — should not be done**
- If indications for both (e.g., T2D + HF + albuminuria): most clinically pressing indication drives MRA selection
- **"Nonsteroidal MRA cannot be a replacement for steroidal MRA for HF and hyperaldosteronism"**

**MRA Research Recommendations:**
- MRA effects on CKD progression + kidney failure + CVD in diabetes patients
- Combining MRA with SGLT2i and GLP-1 RA
- Trials in: T2D + normal albumin, T1D + CKD, transplant, CKD without T2D, dialysis
- Comparative steroidal vs nonsteroidal MRA studies
- Real-world data on nonsteroidal MRA outcomes
- Health economic evaluation

**Section 1.5 — Smoking Cessation:**

**Recommendation 1.5.1 (1D — Strong/Very Low):**
- "We recommend advising patients with diabetes and CKD who use tobacco to quit using tobacco products"
- High value on well-documented general population benefits
- No RCTs on smoking cessation in CKD specifically
- Applies to T1D and T2D
- E-cigarettes: increase risk of lung disease and CVD; sparse data in kidney disease
- Evidence quality: very low (1 small crossover trial, 25 participants, 8-hour sessions)

---

## Key Spans Assessment

### Tier 1 Spans (18)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| **"Practice Point 1.4.5: A steroidal MRA should be used for treatment of heart failure, hyperaldosteronism, or refractory h..."** | B+C | 100% | **✅ T1 CORRECT** — FIRST TIME a PP label + full text is captured together! Critical prescribing instruction for steroidal MRA indications |
| **"Steroidal MRA are standard of care for treatment of heart failure (particularly with reduced ejection fraction) and prim..."** | B+F | 98% | **✅ T1 CORRECT** — Drug class indication statement |
| **"When a steroidal MRA is already used for one of these indications, there is no evidence that switching to a nonstero..."** | B+C+F | 100% | **✅ T1 CORRECT** — Critical: no evidence for switching steroidal → nonsteroidal; adding both increases adverse effects |
| **"Currently, a nonsteroidal MRA cannot be a replacement for steroidal MRA for the indications of heart failure and hyperal..."** | B+F | 98% | **✅ T1 CORRECT** — Drug class non-substitutability statement |
| "Practice Point 1.4.4" | C | 98% | **→ T3** — PP label only (text NOT captured) |
| "Recommendation 1.5.1" | C | 98% | **→ T3** — Rec label only |
| "MRA" (B channel) ×9 | B | 100% | **ALL → T3** — Drug class name only |
| "finerenone" (B channel) ×2 | B | 100% | **→ T3** — Drug name only |
| "SGLT2i" (B channel) ×1 | B | 100% | **→ T3** — Drug class name only |

**Summary: 4/18 T1 spans are genuine clinical sentences (22%). ALL 4 are multi-channel (B+C, B+F, B+C+F). This is the BEST genuine T1 count on a page with <30 spans.**

### Tier 2 Spans (10)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "eGFR was ≥60 mL/min/1.73m² or at a dose of 10 mg" (EDITED) | C | 85% | **⚠️ SHOULD BE T1** — Finerenone dosing by eGFR threshold (prior reviewer engaged) |
| "eGFR was 25-59 mL/min/1.73m², with uptitration to 20 mg" (EDITED) | C | 85% | **⚠️ SHOULD BE T1** — Finerenone dosing by eGFR threshold (prior reviewer engaged) |
| "except when heart failure is present." | C | 90% | **✅ T2 OK** — Important exception to steroidal MRA limitation (contextual) |
| "20 mg" | C | 85% | **→ T3** — Dose fragment |
| "daily" ×3 | C | 85% | **→ T3** — Frequency word |
| "potassium" | C | 85% | **→ T3** — Electrolyte name |
| "urine albumin" | C | 85% | **→ T3** — Lab name |
| "stop" | C | 90% | **→ T3** — Action verb without context |

**Summary: 3/10 T2 correctly tiered or meaningful. 2 should be T1 (eGFR-based dosing with drug context). 5 are noise.**

---

## Critical Findings

### ✅ FOUR GENUINE T1 PRESCRIBING SENTENCES — EXCELLENT PAGE

This page produces 4 genuine multi-channel T1 spans, the highest count for any page with <30 spans:

1. **PP 1.4.5 full text** (B+C): First time in the entire audit that a Practice Point label AND its full text are captured in the same span
2. **Steroidal MRA standard of care** (B+F): HF (reduced EF) and primary hyperaldosteronism
3. **No switching evidence** (B+C+F): "No evidence that switching to a nonsteroidal MRA will improve outcome" — critical for preventing inappropriate drug substitution
4. **Non-substitutability** (B+F): "Nonsteroidal MRA cannot be a replacement for steroidal MRA for HF and hyperaldosteronism"

### ✅ PP 1.4.5 TEXT CAPTURED (First PP Text in Audit!)
Unlike PP 1.3.1-1.3.7 and PP 1.4.1-1.4.3 which only had labels captured, PP 1.4.5 has its full text in a B+C multi-channel span. This likely succeeds because the PP text starts with a drug class name ("A steroidal MRA should be used...") which triggers B channel, and the C channel matches the PP label pattern.

### ⚠️ Two EDITED T2 Dosing Spans Should Be T1
- "eGFR was ≥60 mL/min/1.73m² or at a dose of 10 mg" — finerenone dosing criterion
- "eGFR was 25-59 mL/min/1.73m², with uptitration to 20 mg" — finerenone titration guidance
Both prior-reviewer-engaged spans contain drug-specific dosing information and should be T1.

### ❌ PP 1.4.4 Text NOT EXTRACTED
"The choice of a nonsteroidal MRA should prioritize agents with documented kidney or cardiovascular benefits" — the PP label is captured but the full text with finerenone dosing by eGFR is missing (only captured as decontextualized T2 fragments).

### ❌ Rec 1.5.1 Text NOT EXTRACTED
"We recommend advising patients with diabetes and CKD who use tobacco to quit using tobacco products" — only the "Recommendation 1.5.1" label is captured.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 1.4.4 full text: prioritize agents with documented benefits | **T1** | Agent selection principle |
| "Only finerenone has rigorous long-term outcome data" | **T1** | Drug-specific limitation |
| "Adding nonsteroidal MRA to steroidal MRA likely increases adverse effects — should not be done" | **T1** | Drug combination prohibition (partially captured in switching span) |
| Rec 1.5.1 full text: advise tobacco cessation | **T1** | Smoking cessation recommendation |
| "E-cigarettes increase risk of lung disease and CVD" | **T2** | E-cigarette safety |
| MRA research recommendations (combining with SGLT2i/GLP-1 RA, populations) | **T3** | Research priorities |

### ✅ Prior Review Activity
2/28 spans reviewed (both EDITED) — prior reviewer engaged with the eGFR-based dosing spans.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — 4 genuine T1 prescribing sentences including PP 1.4.5 full text (first in audit!); prior reviewer engaged with dosing spans |
| **Tier corrections** | ~12 drug names: T1 → T3; PP 1.4.4 + Rec 1.5.1 labels: T1 → T3; 2 eGFR dosing spans: T2 → T1; ~5 dose/frequency fragments: T2 → T3 |
| **Missing T1** | PP 1.4.4 text, "only finerenone" limitation, Rec 1.5.1 text |
| **Missing T2** | E-cigarette safety warning |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~45% — 4 genuine T1 sentences + 2 EDITED dosing spans + 1 exception clause capture significant prescribing content |
| **Tier accuracy** | ~25% (4/18 T1 correct + 3/10 T2 correct = 7/28) |
| **False positive T1 rate** | 78% (14/18 T1 are drug names or labels) |
| **Genuine T1 content** | 4 extracted (PP 1.4.5, steroidal MRA standard of care, no switching evidence, non-substitutability) |
| **Prior review** | 2/28 EDITED |
| **Overall quality** | **GOOD** — Third-best page in audit; multi-channel captures complete PP text for first time; prior reviewer actively engaged |

---

## Why PP 1.4.5 Succeeds Where PP 1.3.1-1.4.3 Failed

PP 1.4.5 text starts with: **"A steroidal MRA should be used for treatment of..."**
- **B channel** matches "steroidal MRA" (drug class name)
- **C channel** matches "Practice Point 1.4.5" (PP label pattern)
- Both fire on the same text block → multi-channel merge produces the full sentence

Previous PPs (1.3.1-1.4.3) started with phrases like:
- "SGLT2i can be added to an existing treatment regimen..." — B matches "SGLT2i" but as a standalone mention, not merged with C's PP label
- "Select patients with consistently normal serum potassium..." — no drug name in the sentence → B doesn't fire

**Key insight**: Practice Points that start with a drug class name get captured; those that start with clinical instructions without a drug name do not.

---

## Review Actions Completed

| Field | Value |
|-------|-------|
| **Review Date** | 2026-02-27 |
| **Reviewer** | claude-auditor |
| **Actions** | 7 confirmed, 21 rejected, 5 added |

### CONFIRMED Spans (7)

| Span ID | Text | Tier | Reason |
|---------|------|------|--------|
| `2bfcb834` | Practice Point 1.4.5: A steroidal MRA should be used for treatment of heart failure, hyperaldosteronism, or refractory h... | T1 | PP 1.4.5 full text — first PP with label+text captured together. KB-1, KB-4. |
| `2999ccee` | Steroidal MRA are standard of care for treatment of heart failure (particularly with reduced ejection fraction) and primary hyperaldosteronism. | T1 | Drug class indication statement. KB-1 prescribing indication. |
| `66779e06` | When a steroidal MRA is already used for one of these indications, there is no evidence that switching to a nonsteroidal MRA will improve outcome... | T1 | No switching evidence + adding both increases adverse effects. KB-4 safety, KB-5 interaction. |
| `b3d5b1c3` | Currently, a nonsteroidal MRA cannot be a replacement for steroidal MRA for the indications of heart failure and hyperaldosteronism. | T1 | Drug class non-substitutability statement. KB-1. |
| `a6167731` | eGFR was ≥60 mL/min/1.73m² or at a dose of 10 mg | T2 (EDITED) | Finerenone dosing by eGFR threshold. Prior reviewer engaged. KB-1 dosing. |
| `4f2967e1` | eGFR was 25-59 mL/min/1.73m², with uptitration to 20 mg | T2 (EDITED) | Finerenone titration guidance. Prior reviewer engaged. KB-1 dosing. |
| `b3fdbefd` | except when heart failure is present. | T2 | Important exception to steroidal MRA limitation. KB-1 prescribing context. |

### REJECTED Spans (21)

| Category | Count | Reject Reason |
|----------|-------|---------------|
| "MRA" (drug class name) | 9 | out_of_scope — drug class name only |
| "finerenone" (drug name) | 2 | out_of_scope — drug name only |
| "SGLT2i" (drug class name) | 1 | out_of_scope — drug class name only |
| "Practice Point 1.4.4" (PP label) | 1 | out_of_scope — PP label without text |
| "Recommendation 1.5.1" (Rec label) | 1 | out_of_scope — Rec label without text |
| "20 mg" (dose fragment) | 1 | out_of_scope — decontextualized dose |
| "daily" x3 (frequency word) | 3 | out_of_scope — frequency word without drug context |
| "potassium" (electrolyte name) | 1 | out_of_scope — electrolyte name only |
| "urine albumin" (lab name) | 1 | out_of_scope — lab name only |
| "stop" (action verb) | 1 | out_of_scope — action verb without context |

### ADDED Facts (5)

| # | Added Text | Target KB | Note |
|---|-----------|-----------|------|
| 1 | The choice of a nonsteroidal MRA should prioritize agents with documented kidney or cardiovascular benefits | KB-1 | PP 1.4.4 full text — agent selection principle |
| 2 | Currently, only finerenone has rigorous long-term outcome data | KB-1 | PP 1.4.4 rationale — drug-specific limitation |
| 3 | Adding nonsteroidal MRA to steroidal MRA likely increases adverse effects and should not be done | KB-4, KB-5 | PP 1.4.5 rationale — drug combination prohibition |
| 4 | We recommend advising patients with diabetes and CKD who use tobacco to quit using tobacco products | KB-4 | Rec 1.5.1 full text (1D strong/very low) |
| 5 | E-cigarettes increase risk of lung disease and cardiovascular disease; sparse data in kidney disease | KB-4 | Rec 1.5.1 rationale — e-cigarette safety warning |

---

## Raw PDF Gap Analysis

| # | Gap Text | Priority | Rationale |
|---|----------|----------|-----------|
| 1 | "Steroidal MRA do not have documented clinical kidney or cardiovascular benefits, except when heart failure is present" | **HIGH** | PP 1.4.4 rationale — steroidal MRA limitation vs nonsteroidal. KB-1 agent selection. |
| 2 | "When a patient is treated with neither a steroidal MRA nor a nonsteroidal MRA but has indications for both (e.g., T2D with heart failure and albuminuria on first-line therapies), the most clinically pressing indication should drive the selection of MRA" | **HIGH** | PP 1.4.5 rationale — dual-indication decision rule for MRA class selection. KB-1 prescribing. |
| 3 | "Steroidal MRA are also useful for reducing blood pressure in the setting of refractory hypertension" | **MODERATE** | PP 1.4.5 rationale — additional steroidal MRA indication. KB-1 prescribing. |
| 4 | "Tobacco use remains a leading cause of death across the globe and is also a known risk factor for the development of CKD" | **MODERATE** | Rec 1.5.1 rationale — CKD-specific harm from tobacco. KB-4. |
| 5 | "no RCTs have examined the impact of smoking cessation on cardiovascular risk in those with CKD" | **MODERATE** | Rec 1.5.1 evidence limitation — explains 1D (very low) grade. KB-4. |
| 6 | "More data are needed on combining MRA with other effective classes of medications, including SGLT2i and GLP-1 RA" | **LOW** | MRA research recommendation — combination therapy evidence gaps. KB-5. |
| 7 | "Trials are needed to examine the benefits and risks of MRA in patients with T2D and normal urine albumin excretion, patients with T1D and CKD, patients who have received a kidney transplant, patients with CKD but without T2D, and patients who are treated with dialysis" | **LOW** | MRA research recommendation — population gaps. |

**All 7 gaps added via API (all 201).**

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total original spans** | 28 |
| **Confirmed** | 7 |
| **Rejected** | 21 |
| **Added (agent)** | 5 |
| **Added (gap fill)** | 7 |
| **Total Added** | 12 |
| **Total Pipeline 2 ready** | 19 (7 confirmed + 12 added) |
| **Review completeness** | 28/28 (100%) |
| **Pipeline 2 completeness** | ~97% — PP 1.4.4 full text + steroidal MRA limitation; PP 1.4.5 full text + dual-indication decision rule + refractory HTN indication; finerenone dosing spans; non-substitutability; Rec 1.5.1 text + tobacco/CKD risk + no RCT evidence + e-cigarette warning; MRA research recommendations (combination + population gaps) |
| **Remaining gaps** | Health economic evaluation of nonsteroidal MRA (T3, administrative); secondhand smoke relationship with kidney disease (ref 180, very low priority) |
