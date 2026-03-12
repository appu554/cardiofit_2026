# Page 43 Audit — SGLT2i Heart Failure Trials + Kidney Outcomes Narrative

| Field | Value |
|-------|-------|
| **Page** | 43 (PDF page S42) |
| **Content Type** | SGLT2i HF dedicated trials (DAPA-HF, EMPEROR-Reduced, EMPEROR-Preserved, SOLOIST, DELIVER) + kidney outcomes (EMPA-REG, CANVAS, DECLARE-TIMI 58, DAPA-HF) + meta-analyses |
| **Pipeline Spans** | 9 original (2 T1, 7 T2) |
| **Channels** | C, F — NO D (Table Decomp) on this page |
| **Disagreements** | 2 |
| **Review Status** | **COMPLETED** — Page ACCEPTED |
| **Risk** | Disagreement (resolved) |
| **Audit Date** | 2026-02-25 (pre-review) → 2026-02-27 (review completed) |
| **Reviewer** | pharma@vaidshala.com |

---

## Review Actions Summary

| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 6 | Noise spans: decontextualized thresholds, pipeline artifacts, fragments |
| **CONFIRMED** | 3 | Genuine T2 spans: DAPA-HF enrollment, diabetes-independent benefit, EMPEROR-Reduced enrollment |
| **ADDED (REVIEWER)** | 10 | Verbatim PDF trial outcome facts covering all major trials on this page |
| **EDITED** | 8 of 10 | Corrected from paraphrased to verbatim PDF text |
| **Final extractions** | 19 | 3 confirmed pipeline + 10 REVIEWER + 6 rejected (not counted) |

---

## Source PDF Content

**Heart Failure Dedicated Trials:**
- **DAPA-HF**: Dapagliflozin, 4744 patients, HFrEF (EF ≤40%), eGFR ≥30. Primary outcome HR 0.74 (0.65-0.85). eGFR ≥60 subgroup HR 0.76 (0.63-0.92); eGFR <60 HR 0.72 (0.59-0.86). Benefit similar with/without diabetes.
- **EMPEROR-Reduced**: Empagliflozin, 3730 patients, HFrEF (EF ≤40%), eGFR ≥20. Primary outcome HR 0.75 (0.65-0.86). Composite kidney outcome HR 0.50 (0.32-0.77).
- **Meta-analysis DAPA-HF + EMPEROR-Reduced**: eGFR ≥60 HR 0.72 (0.62-0.82); eGFR <60 HR 0.77 (0.68-0.88); kidney outcome HR 0.62 (0.43-0.90)
- **EMPEROR-Preserved**: Empagliflozin, 5988 patients, EF ≥40%. Primary outcome HR 0.79 (0.69-0.90). 50% had eGFR <60.
- **SOLOIST**: Sotagliflozin, 70% had eGFR <60. Primary outcome HR 0.67 (0.52-0.85). Stopped early.
- **DELIVER**: Dapagliflozin, HFpEF (LVEF >40%). Met primary endpoint (announced May 2022).

**Kidney Outcomes:**
- **EMPA-REG**: Nephropathy HR 0.61 (0.53-0.70); 12.7% vs 18.8%
- **CANVAS**: Albuminuria progression HR 0.73 (0.67-0.79); composite kidney HR 0.60 (0.47-0.77); doubling SCr/kidney failure/death HR 0.53 (0.33-0.84)
- **DECLARE-TIMI 58**: Secondary kidney outcome HR 0.76 (0.67-0.87)
- **DAPA-HF**: Worsening kidney function HR 0.71 (0.44-1.16) — not significant (P=0.17), trial only 18.2 months

---

## Pipeline Span Disposition

### REJECTED (6 spans)

| Span | Channel | Conf | Reason |
|------|---------|------|--------|
| "eGFR <60 ml/min" | C | 95% | Decontextualized threshold — no drug/action context |
| "eGFR <60" | C | 95% | Decontextualized threshold — no drug/action context |
| "Main secondary outcome" | D | 92% | Table column header fragment |
| `<!-- PAGE 43 -->` | F | 90% | Pipeline artifact |
| "hospitalizations and urgent visits for heart failure (first and subsequent events)" | F | 85% | Endpoint definition fragment (no trial result) |
| "The aforementioned 2019 meta-analysis pooled data from the EMPA-REG, CANVAS program, and DECLARE-TIMI 58" | F | 90% | Introductory sentence fragment (no results included) |

### CONFIRMED (3 spans)

| Span | Channel | Conf | Rationale |
|------|---------|------|-----------|
| "The DAPA-HF trial enrolled 4744 patients with symptomatic HFrEF defined as ejection fraction ≤40%, with an eGFR ≥30 ml/min/1...." | C,F | 95% | Trial enrollment description with eGFR criterion |
| "The primary outcome was similarly reduced for individuals with or without diabetes, with no effect of heterogeneity by d..." | F | 85% | Important clinical finding: benefit independent of diabetes status |
| "The EMPEROR-Reduced trial enrolled 3730 patients with HFrEF, defined as ejection fraction ≤40%, with an eGFR ≥20 mL/min/..." | C,F | 95% | Trial enrollment with eGFR ≥20 criterion (supports Rec 1.3.1 threshold) |

---

## REVIEWER-Added Facts (10 total, 8 edited to verbatim)

All facts use verbatim text from the KDIGO 2022 PDF (page S42). Facts #1–2 were added with correct text in session 1. Facts #3–10 were initially paraphrased, then edited to verbatim PDF text in session 2.

### Fact #1 — DAPA-HF Primary (ADDED, not edited)
> Over a median of 18.2 months, the primary outcome of cardiovascular death, heart failure hospitalization, or urgent heart failure visit occurred in 16.3% of the dapagliflozin group and 21.2% of the placebo group (HR: 0.74; 95% CI: 0.65–0.85). The primary outcome was similarly reduced for individuals with or without diabetes, with no effect of heterogeneity by diabetes status. The primary outcome was also similar among those with an eGFR ≥60 ml/min per 1.73 m2 (HR: 0.76; 95% CI: 0.63–0.92) or <60 ml/min per 1.73 m2 (HR: 0.72; 95% CI: 0.59–0.86).

### Fact #2 — EMPEROR-Reduced Primary (ADDED, not edited)
> Over a median of 16 months, the primary outcome of cardiovascular death or heart failure hospitalization occurred in 19.4% of the empagliflozin group and 24.7% of the placebo group (HR: 0.75; 95% CI: 0.65–0.86). As seen in DAPA-HF, the primary outcome was similarly reduced for individuals with and without diabetes. The primary outcome among those with an eGFR ≥60 ml/min per 1.73 m2 was HR: 0.67; 95% CI: 0.55–0.83 and for those with eGFR <60 ml/min per 1.73 m2 was HR: 0.83; 95% CI: 0.69–1.00. A composite kidney outcome HR of 0.50 (95% CI: 0.32–0.77) was also reported.

### Fact #3 — Meta-analysis DAPA-HF + EMPEROR-Reduced (EDITED to verbatim)
> A recent meta-analysis of both the DAPA-HF and EMPEROR-Reduced trials further revealed a composite outcome on first hospitalization for heart failure or cardiovascular death of HR: 0.72 (95% CI: 0.62–0.82) for an eGFR ≥60 ml/min per 1.73 m2 and HR: 0.77 (95% CI: 0.68–0.88) for eGFR <60 ml/min per 1.73 m2; a composite kidney outcome HR: 0.62; 95% CI: 0.43–0.90 (P = 0.013) was also reported.

### Fact #4 — EMPEROR-Preserved (EDITED to verbatim)
> The EMPEROR-Preserved trial enrolled 5988 patients, with or without T2D, with class II-IV heart failure symptoms and an ejection fraction ≥40%. Empagliflozin, compared to placebo, reduced the risk of the primary outcome of cardiovascular death or hospitalization for heart failure by 21% (HR: 0.79; 95% CI: 0.69–0.90). This benefit was again similar among patients with or without diabetes. Fifty percent of study participants had an eGFR <60 ml/min per 1.73 m2, and there was no significant interaction by eGFR status (≥60 vs. <60 ml/min per 1.73 m2) for the primary cardiovascular outcome.

### Fact #5 — SOLOIST (EDITED to verbatim)
> The SOLOIST trial enrolled patients with T2D who had recently been hospitalized for worsening heart failure (with or without reduced ejection fraction), of which 70% of patients had an eGFR <60 ml/min per 1.73 m2. The primary outcome was deaths from cardiovascular causes and hospitalizations and urgent visits for heart failure (first and subsequent events). The trial was stopped early, but sotagliflozin did reduce the primary outcome by 33% (HR: 0.67; 95% CI: 0.52–0.85). There was no significant interaction by eGFR status for the primary outcome.

### Fact #6 — DELIVER (EDITED to verbatim)
> The ongoing phase III Dapagliflozin Evaluation to Improve the LIVEs of Patients with PReserved Ejection Fraction Heart Failure (DELIVER) trial randomized patients with heart failure with mildly reduced or preserved ejection fraction (left ventricular ejection fraction [LVEF] >40%) with or without T2D to treatment with dapagliflozin 10 mg or placebo. On May 5, 2022, it was announced that the results reached a statistically significant and clinically meaningful reduction in the primary composite endpoint of cardiovascular death or worsening heart failure. Results are expected to be reported later in 2022.

### Fact #7 — EMPA-REG Kidney (EDITED to verbatim)
> EMPA-REG (empagliflozin vs. placebo) also evaluated a prespecified kidney outcome of incident or worsening nephropathy, defined as progression to severely increased albuminuria (ACR >300 mg/g [>30 mg/mmol]), doubling of serum creatinine, accompanied by an eGFR ≤45 ml/min per 1.73 m2, initiation of kidney replacement therapy, or death from kidney causes (i.e., "renal death"). This incident or worsening nephropathy outcome was lower in the empagliflozin group — 12.7% versus 18.8% — with a HR of 0.61 (95% CI: 0.53–0.70).

### Fact #8 — CANVAS Kidney (EDITED to verbatim)
> In the CANVAS program (overall cohort including those with and without baseline CKD), canagliflozin also conferred kidney benefit, with a 27% lower risk of progression of albuminuria (HR: 0.73; 95% CI: 0.67–0.79) and a 40% lower risk of a composite kidney outcome (≥40% reduction in eGFR, need for kidney replacement therapy, or death from kidney cause; HR: 0.60; 95% CI: 0.47–0.77). The CANVAS program further reported additional prespecified kidney outcomes. The composite kidney outcome of doubling of serum creatinine, kidney failure, and death from kidney causes occurred in 1.5 versus 2.8 per 1000 patient-years in the canagliflozin versus placebo groups (HR: 0.53; 95% CI: 0.33–0.84). There was also a reduction in albuminuria and an attenuation of eGFR decline.

### Fact #9 — DECLARE-TIMI 58 Kidney (EDITED to verbatim)
> In the DECLARE-TIMI 58 trial (dapagliflozin vs. placebo), there was a 1.3% absolute and 24% relative risk reduction in the secondary kidney outcome (a composite of a ≥40% decrease in eGFR to <60 ml/min per 1.73 m2, kidney failure, and cardiovascular death or death from kidney causes: HR: 0.76; 95% CI: 0.67–0.87).

### Fact #10 — DAPA-HF Worsening Kidney (EDITED to verbatim)
> In the DAPA-HF trial, the secondary outcome of worsening kidney function (defined as a sustained ≥50% reduction in eGFR, kidney failure, or death from kidney causes) occurred in 1.2% of the dapagliflozin arm and 1.6% of the placebo arm (HR: 0.71; 95% CI: 0.44–1.16), which was not statistically significant (P = 0.17). However, the median duration of the DAPA-HF trial was only 18.2 months, which may not have been long enough to accumulate kidney endpoints.

---

## Critical Findings

### ✅ Good Trial Enrollment Spans (C+F Multi-Channel)
The two best pipeline spans capture DAPA-HF and EMPEROR-Reduced enrollment criteria, including eGFR thresholds (≥30 and ≥20 respectively). The EMPEROR-Reduced eGFR ≥20 criterion is particularly important — it directly supports the Rec 1.3.1 eGFR ≥20 threshold that was completely missing from page 39.

### ✅ Diabetes-Independent Benefit Captured
The F channel captured the important finding that SGLT2i benefit is independent of diabetes status — clinically significant for prescribing in non-diabetic CKD.

### ✅ All Trial Outcomes Now Covered (REVIEWER)
All 10 major trial outcomes from this page are now represented as REVIEWER facts with verbatim PDF text, covering DAPA-HF, EMPEROR-Reduced, meta-analysis, EMPEROR-Preserved, SOLOIST, DELIVER, EMPA-REG kidney, CANVAS kidney, DECLARE-TIMI 58 kidney, and DAPA-HF worsening kidney.

### ❌ Zero Genuine T1 Content (Appropriate)
Both original T1 spans were decontextualized C-channel eGFR threshold extractions — correctly rejected. The page is pure evidence narrative with no prescribing instructions, so the absence of T1 content is appropriate.

---

## Post-Review Completeness Score

| Metric | Pre-Review | Post-Review |
|--------|------------|-------------|
| **Extraction completeness** | ~25% (3/12 facts) | **~100%** (13 spans covering all major trial outcomes) |
| **Tier accuracy** | ~33% (3/9 correct) | **100%** (3 confirmed pipeline T2 + 10 REVIEWER T2) |
| **False positive T1 rate** | 100% (2/2) | **0%** (both rejected) |
| **Genuine T1 content** | 0 (appropriate) | 0 (appropriate — evidence page) |
| **Overall quality** | FAIR | **EXCELLENT** — Complete trial evidence coverage with verbatim PDF text |
| **Page status** | PENDING | **ACCEPTED** |
