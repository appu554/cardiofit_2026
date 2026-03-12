# Page 42 Audit — SGLT2i Cardiovascular Trial Evidence Narrative

| Field | Value |
|-------|-------|
| **Page** | 42 (PDF page S41) |
| **Content Type** | SGLT2i CV trial narrative: EMPA-REG, CANVAS, DECLARE-TIMI 58, VERTIS-CV, CREDENCE, DAPA-CKD, SCORED trial results + HF outcomes + meta-analyses |
| **Extracted Spans** | 21 total (11 pipeline + 10 REVIEWER) |
| **Channels** | B, D, F, REVIEWER |
| **Disagreements** | 4 |
| **Review Status** | 4 confirmed + 6 rejected + 1 EDITED + 10 REVIEWER added = 21/21 reviewed |
| **Risk** | Disagreement |
| **Page Decision** | **ACCEPTED** |
| **Audit Date** | 2026-02-27 (executed + cross-check pass 2) |
| **Cross-Check** | Verified against raw PDF text — 2 passes. Pass 1: API review + 6 REVIEWER facts. Pass 2: 4 additional REVIEWER facts added (overall trial HRs). All 15 data points confirmed captured. |

---

## Source PDF Content

**SGLT2i Cardiovascular Trial Evidence (narrative text):**

Key trial results described on this page:
- **EMPA-REG**: Empagliflozin 10/25mg reduced MACE by 14% (HR 0.86; 0.74-0.99). In eGFR <60 subgroup: CV death HR 0.71 (0.52-0.98), all-cause mortality HR 0.76 (0.59-0.99), HF hospitalization HR 0.61 (0.42-0.87)
- **CANVAS**: Canagliflozin 100/300mg reduced MACE by 14% (HR 0.86; 0.75-0.97). eGFR 30-60 subgroup: MACE HR 0.70 (0.55-0.90)
- **DECLARE-TIMI 58**: Dapagliflozin 10mg, CV death/HF HR 0.83 (0.73-0.95). Primary prevention trial (59% had risk factors, not established CVD)
- **VERTIS-CV**: Ertugliflozin, noninferiority for MACE; CV death/HF HR 0.88 (0.75-1.03) — not statistically significant
- **CREDENCE**: Canagliflozin, HF hospitalization HR 0.61 (0.47-0.80), MACE HR 0.80 (0.67-0.95)
- **DAPA-CKD**: Dapagliflozin, CV death/HF HR 0.71 (0.55-0.92)
- **SCORED**: Sotagliflozin reduced primary outcome by 26% (HR 0.74; 0.63-0.88); original coprimary HR 0.77 (0.66-0.91)
- **Meta-analysis (eGFR 30-60)**: HF hospitalization HR 0.60 (0.47-0.77), MACE HR 0.82 (0.70-0.95)
- **Meta-analysis (CKD exclusive)**: CREDENCE, DAPA-CKD, SCORED composite CV HR 0.73 (0.65-0.82)
- **Heart failure outcomes**: Consistent HF hospitalization reduction across EMPA-REG, CANVAS, DECLARE-TIMI 58; confirmed in real-world registry

---

## Key Spans Assessment

### Tier 1 Spans (5)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "ertugliflozin" | B | 100% | **REJECTED** — Drug name only — no dose, indication, or threshold |
| "ertugliflozin" (duplicate) | B | 100% | **REJECTED** — Drug name only (duplicate) |
| "ertugliflozin" (triplicate) | B | 100% | **REJECTED** — Drug name only (triplicate) |
| **"Heart failure outcomes. In the original cardiovascular outcome trials with SGLT2i among patients with T2D, there was a s..."** | B,F | 98% | **CONFIRMED** — Drug class + population + outcome (HF hospitalization reduction consistent across trials) |
| **"This result was also confirmed in a real-world registry, with the reduction in risk of hospitalization for heart failure..."** | B,F | 98% | **CONFIRMED** — Real-world registry confirmation of HF benefit |

### Tier 2 Spans (6)

| Span | Channel | Conf | Status | Assessment |
|------|---------|------|--------|------------|
| "Total number of participants" | D | 92% | **REJECTED** | Table column header — no clinical content |
| "Total number of participants" (duplicate) | D | 92% | **REJECTED** | Table column header (duplicate) |
| `<!-- PAGE 42 -->` | F | 90% | **REJECTED** | Pipeline HTML artifact |
| **"In the DAPA CKD trial which enrolled patients with albuminuric CKD with and without T2D"** | F | 85% | **CONFIRMED** | Trial enrollment context (DAPA-CKD population description) |
| **"The primary cardiovasc_endpoint was changed during the trial to a composite of cardiovascular death, heart failure hospi..."** | F | 85% | **EDITED** | SCORED trial endpoint change (prior reviewer) |
| **"sotagliflozin reduced this primary outcome by 26% (HR: 0.74; 95% CI: 0.63–0.88); of note, sotagliflozin also reduced the..."** | B,F | 98% | **CONFIRMED** | Drug + specific HR + CI (sotagliflozin efficacy) |

---

## REVIEWER Facts Added (10)

### Phase 1: Original Review (6) — subgroup HRs + meta-analyses

| # | Gap Type | Verbatim Text (truncated) | Reviewer Note |
|---|----------|---------------------------|---------------|
| 1 | EMPA-REG CKD subgroup HRs | "In a prespecified analysis from EMPA-REG of patients with prevalent kidney disease... CV death HR 0.71, mortality HR 0.76, HF HR 0.61" | KB-4 safety evidence for empagliflozin in CKD |
| 2 | CANVAS CKD subgroup MACE | "In subgroup analyses from the CANVAS trial, those with eGFR 30-60... MACE HR 0.70 (0.55-0.90)" | KB-4 safety evidence for canagliflozin |
| 3 | DECLARE-TIMI 58 CV death/HF | "dapagliflozin did reduce... CV death or HF hospitalization HR 0.83 (0.73-0.95)" | KB-1 dosing, KB-4 safety for dapagliflozin |
| 4 | VERTIS-CV MACE noninferiority | "VERTIS CV trial enrolled 8246 patients... MACE noninferiority... CV death/HF HR 0.88 (0.75-1.03)" | KB-4 safety — important negative result |
| 5 | Meta-analysis eGFR 30-60 | "eGFR 30 to <60... HF HR 0.60 (0.47-0.77) and MACE HR 0.82 (0.70-0.95)" | SGLT2i class-level CKD subgroup pooled evidence |
| 6 | Meta-analysis CKD exclusive | "3 trials exclusively CKD (CREDENCE, DAPA-CKD, SCORED)... composite CV HR 0.73 (0.65-0.82)" | SGLT2i class composite CV outcome |

### Phase 2: Cross-Check Pass 2 (4) — overall trial primary HRs

| # | Gap Type | Verbatim Text (truncated) | Reviewer Note |
|---|----------|---------------------------|---------------|
| 7 | EMPA-REG overall MACE | "In the overall trial, empagliflozin reduced 3-point major adverse cardiovascular events (MACE) by 14% (HR: 0.86; 95% CI: 0.74–0.99)." | KB-4 safety — EMPA-REG overall MACE HR, empagliflozin 10/25mg primary CV outcome |
| 8 | CANVAS overall MACE | "As in EMPA-REG, the SGLT2i canagliflozin also reduced MACE by 14% (HR: 0.86; 95% CI: 0.75–0.97)." | KB-4 safety — CANVAS overall MACE HR, canagliflozin 100/300mg primary CV outcome |
| 9 | CREDENCE HF + MACE (completely missing) | "In the CREDENCE trial among patients with T2D with albuminuric CKD, canagliflozin reduced the risk of the secondary cardiovascular outcomes of hospitalization for heart failure and MACE by 39% (HR: 0.61; 95% CI: 0.47–0.80) and 20% (HR: 0.80; 95% CI: 0.67–0.95), respectively." | KB-4 safety — CREDENCE landmark CKD trial, canagliflozin HF + MACE HRs, completely missing from pipeline |
| 10 | DAPA-CKD CV death/HF | "In the DAPA-CKD trial, dapagliflozin reduced the risk of the secondary cardiovascular outcome of death from cardiovascular cause or hospitalization for heart failure by 29% (HR: 0.71; 95% CI: 0.55–0.92)." | KB-4 safety — DAPA-CKD secondary CV outcome HR, dapagliflozin in albuminuric CKD |

---

## Critical Findings

### ❌ "ertugliflozin" Extracted 3 Times as T1 — Classic B Channel Pattern
The B (Drug Dictionary) channel matched the drug name "ertugliflozin" three separate times on the page. Each is a standalone drug name without any dose, indication, threshold, or clinical instruction. All three REJECTED via API.

### ✅ Four Genuine Narrative Spans Confirmed
- HF outcomes summary (B+F 98%) — SGLT2i class HF hospitalization reduction
- Real-world registry confirmation (B+F 98%) — HF benefit mirrored in registry data
- DAPA-CKD enrollment context (F 85%) — trial population description
- Sotagliflozin HR (B+F 98%) — drug + specific HR + CI

### ✅ 10 REVIEWER Facts Fill All Trial HR Gaps (2 passes)
- **Pass 1 (6 facts)**: Subgroup analyses and meta-analyses — EMPA-REG CKD, CANVAS CKD, DECLARE-TIMI 58, VERTIS-CV, 2 meta-analyses
- **Pass 2 (4 facts)**: Overall trial primary HRs — EMPA-REG overall, CANVAS overall, CREDENCE (completely missing!), DAPA-CKD
- CREDENCE was the most critical gap: a landmark CKD trial with both HF (HR 0.61) and MACE (HR 0.80) endpoints, entirely absent from pipeline + original review

### ✅ Prior Review Activity
1/11 pipeline spans EDITED (SCORED endpoint change) — prior reviewer engaged with trial methodology content.

### ✅ 2nd-Pass Verification — All 15 Data Points Confirmed
Cross-checked all 15 distinct data points from raw PDF text against captured spans. Zero remaining gaps after Phase 2 additions.

---

## Final Disposition

| Action | Details |
|--------|---------|
| **Decision** | **ACCEPTED** |
| **Total Spans** | 21 (11 pipeline + 10 REVIEWER) |
| **Pipeline Confirmed** | 4 (HF outcomes, real-world registry, DAPA-CKD context, sotagliflozin HR) |
| **Pipeline Rejected** | 6 (ertugliflozin ×3, "Total number" ×2, pipeline artifact) |
| **Pipeline EDITED** | 1 (SCORED endpoint — prior reviewer) |
| **REVIEWER Added** | 10 (Phase 1: 6 subgroup/meta-analysis HRs + Phase 2: 4 overall trial HRs) |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | **~95%** — 10 REVIEWER additions fill all major trial HR gaps; 2nd-pass verified all 15 data points captured |
| **Tier accuracy (pipeline)** | 36% (4/11 pipeline spans genuine) |
| **Genuine content** | 14 total (4 pipeline confirmed + 1 EDITED + 10 REVIEWER added) — was 5 pre-review |
| **REVIEWER Added** | 10 (in 2 phases) |
| **Prior review** | 1/11 EDITED |
| **Cross-check passes** | 2 (raw PDF verification) |
| **Overall quality** | **EXCELLENT** — Dense narrative page required massive REVIEWER supplementation; all 15 data points now captured including CREDENCE (previously completely missing) |
