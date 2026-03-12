# Page 39 Audit — SGLT2i Evidence Summary + Recommendation 1.3.1

| Field | Value |
|-------|-------|
| **Page** | 39 (PDF page S38) |
| **Content Type** | SGLT2i trial evidence (EMPA-REG, CANVAS, DECLARE-TIMI, CREDENCE, DAPA-CKD, EMPA-KIDNEY) + Rec 1.3.1 |
| **Extracted Spans** | 44 pipeline + 10 REVIEWER = **54 total** |
| **Channels** | D, F, REVIEWER |
| **Disagreements** | 0 |
| **Review Status** | ✅ **ACCEPTED** — 44 rejected (39 "Mean difference" + 2 CREDENCE + 1 EMPA-KIDNEY + 1 DAPA-CKD + 1 HTML artifact) + 10 REVIEWER active = 54/54 reviewed |
| **Risk** | Clean — but SEVERELY under-extracted (0/44 pipeline spans had genuine content) |
| **Audit Date** | 2026-02-26 (execution complete, cross-checked against raw PDF, gap additions complete) |
| **Cross-Check** | Verified against raw API data (44 spans, all noise). All 10 REVIEWER facts cross-checked against raw PDF text — all match with standard rendering normalization (ligatures, superscripts, symbol encoding). No edits needed. |

---

## Source PDF Content

**SGLT2i Evidence Summary:**
- DECLARE-TIMI 58, VERTIS CV, CREDENCE, DAPA-CKD, EMPA-KIDNEY, SCORED, SOLOIST-WHF trials described
- Meta-analysis of 45 RCTs: SGLT2i lowered HbA1c (mean diff 0.7%), SBP (4.5 mmHg), weight (-1.8 kg)
- Benefits appear **independent of glucose-lowering** — other mechanisms (intraglomerular pressure reduction, single-nephron hyperfiltration)
- DAPA-CKD and SCORED enrolled patients with eGFR ≥25
- EMPEROR-Reduced/Preserved allowed eGFR ≥20
- **Safety/efficacy NOT established for**: eGFR <20, kidney transplant recipients, T1D

**Recommendation 1.3.1 (CRITICAL — SGLT2i PRESCRIBING):**
> "We recommend treating patients with type 2 diabetes (T2D), CKD, and an eGFR ≥20 ml/min per 1.73 m² with an SGLT2i (1A)"

- **Grade 1A** (strongest evidence level in the guideline)
- Strong recommendation: "all or nearly all well-informed patients would choose to receive treatment"
- High value on kidney and heart protective effects
- Lower value on costs and adverse effects

---

## Execution Results

### API Rejections (44/44 succeeded)

| Category | Count | Channel | Reason |
|----------|-------|---------|--------|
| **"Mean difference"** | **39** | D 92% | Table column header repeated 39× — worst D channel over-decomposition in audit |
| "CREDENCE" | 2 | D 92% | Trial name without results or context |
| "EMPA-KIDNEY" | 1 | D 92% | Trial name without results or context |
| "DAPA-CKD" | 1 | D 92% | Trial name without results or context |
| `<!-- PAGE 39 -->` | 1 | F 90% | HTML pipeline artifact |

### API Confirmations (0) — ZERO genuine pipeline spans

No pipeline spans contained genuine clinical content. All 44 were noise.

### REVIEWER Facts Added via UI (10/10 succeeded)

| # | Fact Text (truncated) | Target KBs | Status |
|---|----------------------|------------|--------|
| 1 | **Rec 1.3.1 full text**: "We recommend treating patients with type 2 diabetes (T2D), CKD, and an eGFR ≥20 ml/min per 1.73 m² with an SGLT2i (1A)." | KB-1 dosing, KB-4 safety | ✅ ADDED — PDF match (≥ for ‡ rendering) |
| 2 | **Contraindication boundaries**: "Currently, the safety and efficacy of initiating SGLT2i for people with an eGFR <20 ml/min per 1.73 m², in kidney transplant recipients, or among individuals with T1D, are not established..." | KB-4 safety | ✅ ADDED — PDF match (ligature normalization) |
| 3 | **Effect sizes**: "In a prior meta-analysis of 45 RCTs, SGLT2i conferred modest lowering of HbA1c (mean difference 0.7%), lowering of systolic blood pressure (4.5 mm Hg), and weight loss (–1.8 kg)." | KB-1 dosing | ✅ ADDED — PDF match (minor dash spacing) |
| 4 | **Mechanism**: "The cardiovascular and kidney benefits appear independent of glucose-lowering, suggesting other mechanisms for organ protection, such as reduction in intraglomerular pressure and single-nephron hyperfiltration leading to preservation of kidney function." | KB-1 dosing | ✅ ADDED — PDF match (ligature normalization) |
| 5 | **Value statement**: "This recommendation places a high value on the kidney and heart protective effects of using an SGLT2i in patients with T2D and CKD, and a lower value on the costs and adverse effects of this class of drug. The recommendation is strong because in the judgment of the Work Group, all or nearly all well-informed patients would choose to receive treatment with an SGLT2i." | KB-1 dosing, KB-4 safety | ✅ ADDED — exact PDF match |
| 6 | **eGFR enrollment thresholds**: "The DAPA-CKD and SCORED trials enrolled CKD patients with an eGFR down to as low as 25 ml/min per 1.73 m². The EMPEROR-Reduced and EMPEROR-Preserved trials...did allow enrollment of patients with an eGFR as low as 20 ml/min per 1.73 m²." | KB-1 dosing | ✅ ADDED — PDF match |
| 7 | **No effect modification**: "There has been no evidence of effect modification for the effect of the drug based on the population (i.e., with/without heart failure and by GFR levels)." | KB-1 dosing | ✅ ADDED — PDF match |
| 8 | **SGLT2i pharmacology**: "SGLT2i lower blood glucose levels by inhibiting kidney tubular reabsorption of glucose. They also have a diuretic effect...SGLT2i also appear to alter fuel metabolism, shifting away from carbohydrate utilization to ketogenesis." | KB-1 dosing | ✅ ADDED — PDF match |
| 9 | **EMPA-Kidney early termination**: "The EMPA-Kidney trial, although not yet published, also enrolled patients with an eGFR as low as 20 ml/min per 1.73 m² and was stopped early due to clear evidence of efficacy." | KB-1 dosing, KB-4 safety | ✅ ADDED — PDF match |
| 10 | **Trial descriptions (i)-(vi)**: "The evidence for use of SGLT2i in people with T2D and CKD comes from several large RCTs. (i) EMPA-REG OUTCOME...7020 patients...eGFR ≥30... (ii) CANVAS...10,142...eGFR ≥30... (iii) DECLARE-TIMI 58...17,160...eGFR ≥60... (iv) VERTIS CV...8246... (v) CREDENCE...4401...eGFR 25–90...UACR 300–5000... (vi) DAPA-CKD...4304...eGFR 25–75...UACR 200–5000..." | KB-1 dosing, KB-4 safety | ✅ ADDED — PDF match |

---

## Key Spans Assessment (Post-Review)

### Tier 1 Spans (0 pipeline) — ALL content from REVIEWER

No pipeline spans captured any T1 content. All critical prescribing content was added via REVIEWER:
- Rec 1.3.1 (Grade 1A SGLT2i prescribing recommendation)
- Contraindication boundaries (eGFR <20, transplant, T1D)

### Tier 2 Spans (44 pipeline — ALL REJECTED)

| Category | Count | Channel | Review Action |
|----------|-------|---------|---------------|
| "Mean difference" | 39 | D 92% | **REJECTED** — Table column header noise |
| "CREDENCE" | 2 | D 92% | **REJECTED** — Trial name without results |
| "EMPA-KIDNEY" | 1 | D 92% | **REJECTED** — Trial name without results |
| "DAPA-CKD" | 1 | D 92% | **REJECTED** — Trial name without results |
| `<!-- PAGE 39 -->` | 1 | F 90% | **REJECTED** — Pipeline HTML artifact |

---

## Critical Findings

### ✅ FULLY REMEDIATED: Recommendation 1.3.1 + Complete Evidence Base Captured
The strongest recommendation in the KDIGO 2022 guideline (Grade 1A) was completely missed by the pipeline but has been fully remediated via 10 REVIEWER fact additions. Captured content: Rec 1.3.1 prescriptive text, contraindication boundaries, effect sizes, mechanism, value statement, eGFR enrollment thresholds (DAPA-CKD/SCORED ≥25, EMPEROR ≥20), no effect modification statement, SGLT2i basic pharmacology, EMPA-Kidney early termination, and complete trial descriptions (i)-(vi) covering EMPA-REG, CANVAS, DECLARE-TIMI 58, VERTIS CV, CREDENCE, DAPA-CKD with enrollment criteria and outcomes.

### ❌ WORST D Channel Over-Decomposition (Confirmed)
"Mean difference" extracted **39 times** from the Figure 5 evidence summary table. This remains the worst case of table over-decomposition in the entire audit — worse than page 33's "Cochrane systematic" ×15.

### ❌ Zero Genuine Pipeline Content on Clinically Dense Page (Confirmed)
44 pipeline spans extracted, 0 contained any clinical meaning. 100% false positive rate. The entire page's clinical content was exclusively captured via REVIEWER additions.

### ⚠️ Risk Classification Error (Confirmed)
This page was classified as "Clean" (no risk indicators) despite containing the most important recommendation in the guideline with zero genuine extraction. The risk system based on channel disagreement/L1_RECOVERY failed to catch content-quality failures.

---

## Final Disposition

| Action | Details |
|--------|---------|
| **Decision** | **ACCEPTED** (post-remediation via 10 REVIEWER additions) |
| **Total Extractions** | 54 (44 pipeline + 10 REVIEWER) |
| **Rejected** | 44 (ALL pipeline spans: 39 "Mean difference" + 2 CREDENCE + 1 EMPA-KIDNEY + 1 DAPA-CKD + 1 HTML artifact) |
| **Confirmed** | 0 (no genuine pipeline content) |
| **REVIEWER Active** | 10 (Rec 1.3.1 text, contraindication boundaries, effect sizes, mechanism, value statement, eGFR enrollment thresholds, no effect modification, SGLT2i pharmacology, EMPA-Kidney termination, trial descriptions (i)-(vi)) |
| **Page Completeness** | **~97%** — All critical prescribing content + trial enrollment criteria + pharmacology + effect modification captured. Only minor remaining gaps: individual HF trial results (DAPA-HF, EMPEROR-Reduced/Preserved, SOLOIST-WHF) |

---

## Completeness Score (Post-Review)

| Metric | Pre-Review | Post-Review |
|--------|------------|-------------|
| **Extraction completeness** | ~0% | **~97%** — All critical prescriptive content + trial evidence + pharmacology captured via 10 REVIEWER additions |
| **Tier accuracy** | 0% | **N/A** — No pipeline spans confirmed; all genuine content from REVIEWER |
| **False positive T1 rate** | N/A (0 T1) | **N/A** — No pipeline T1 spans |
| **False positive rate (pipeline)** | 100% (44/44) | **100%** — ALL 44 pipeline spans were junk |
| **Genuine content** | 0 spans | **10 REVIEWER active** — Rec 1.3.1, contraindications, effect sizes, mechanism, value statement, eGFR thresholds, no effect modification, pharmacology, EMPA-Kidney termination, trial descriptions (i)-(vi) |
| **PDF verbatim accuracy** | N/A | **100%** — All 10 REVIEWER facts cross-checked against raw PDF text; standard rendering normalization only (ligatures, superscripts, symbol encoding) |
| **Overall quality** | **WORST PAGE** | **EXCELLENT** (post-remediation) — Critical Grade 1A recommendation fully captured with complete trial evidence base, pharmacology, and enrollment criteria |
