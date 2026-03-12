# Page 58 Audit — PP 2.1.1-2.1.4 (HbA1c Frequency, CGM, GMI, SMBG), Figure 11

| Field | Value |
|-------|-------|
| **Page** | 58 (PDF page S57) |
| **Content Type** | PP 2.1.1 (HbA1c monitoring frequency), PP 2.1.2 (HbA1c low reliability in advanced CKD), PP 2.1.3 (GMI from CGM when HbA1c discordant), PP 2.1.4 (daily CGM/SMBG for hypoglycemia prevention), Figure 11 (HbA1c frequency + GMI by CKD stage) |
| **Extracted Spans** | 43 total (4 T1, 39 T2) |
| **Channels** | C, D, F |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 43 |
| **Risk** | Clean |
| **Cross-Check** | Verified against pipeline export 2026-02-25 |
| **Audit Date** | 2026-02-25 |

---

## Source PDF Content

**Practice Point 2.1.1:**
- "Monitoring long-term glycemic control by HbA1c twice per year is reasonable for patients with diabetes"
- HbA1c may be measured up to 4 times/year if glycemic target not met or after therapy change
- HbA1c <7% (<53 mmol/mol) vs 8-9% (64-75 mmol/mol): reduces microvascular + some macrovascular complications
- HbA1c may underestimate (more commonly) or overestimate glycemia in advanced CKD
- No advantages of glycated albumin or fructosamine over HbA1c known in CKD

**Practice Point 2.1.2:**
- "Accuracy and precision of HbA1c measurement declines with advanced CKD (G4-G5), particularly dialysis patients, in whom HbA1c measurements have low reliability"
- Correlations progressively weaker with CKD G4-G5
- HbA1c remains biomarker of choice because glycated albumin/fructosamine have no advantages and have clinically relevant assay biases with hypoalbuminemia

**Practice Point 2.1.3:**
- "A glucose management indicator (GMI) derived from CGM data can be used to index glycemia for individuals in whom HbA1c is not concordant with directly measured blood glucose levels or clinical symptoms"
- CGM/SMBG yield direct measurements NOT biased by CKD or its treatments (including dialysis, transplant)
- GMI derived from CGM expressed in HbA1c units (%) — facilitates interpretation
- If HbA1c < concurrent GMI: HbA1c underestimates blood glucose by the difference
- GMI useful for advanced CKD, dialysis patients where HbA1c reliability is low
- GMI needs re-establishment when red blood cell turnover or protein glycation changes

**Practice Point 2.1.4:**
- "Daily glycemic monitoring with CGM or SMBG may help prevent hypoglycemia and improve glycemic control when glucose-lowering therapies associated with risk of hypoglycemia are used"
- Minute-to-minute glycemic variability and hypoglycemia episodes are important targets
- Especially T1D and insulin-treated patients

**Figure 11 — HbA1c Frequency and GMI by CKD Stage:**

| CKD Stage | Measure HbA1c? | Frequency | GMI Reliability | GMI Usefulness |
|-----------|---------------|-----------|-----------------|----------------|
| **G1-G3b** (eGFR ≥30) | Yes | Twice/year; up to 4×/year if not at target or therapy change | High | Occasionally useful |
| **G4-G5** (eGFR <30, dialysis, transplant) | Yes | Twice/year; up to 4×/year if not at target or therapy change | Low | Likely useful |

---

## Key Spans Assessment

### Tier 1 Spans (4)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Practice Point 2.1.1" | C | 98% | **→ T3** — PP label only |
| "Practice Point 2.1.2" | C | 98% | **→ T3** — PP label only |
| "Practice Point 2.1.3" | C | 98% | **→ T3** — PP label only |
| "Practice Point 2.1.4" | C | 98% | **→ T3** — PP label only |

**Summary: 0/4 T1 spans are genuine. All 4 are PP labels without text — same systemic pattern.**

### Tier 2 Spans (36)

| Category | Count | Assessment |
|----------|-------|------------|
| **"HbA1c"** (C channel) | ~25 | **ALL → T3** — Lab test name repeated (continuing page 57's pattern) |
| **"HbA1c"** (D channel) | 4 | **→ T3** — Lab name from Figure 11 table decomposition |
| **"Yes"** (D channel) | 2 | **→ T3** — Boolean from Figure 11 "Measure HbA1c? Yes" |
| **"Measure"** (D channel) | 1 | **→ T3** — Column header fragment |
| **"Twice per year / Up to 4 times per year if not achieving target or change in therapy"** (D channel) | 2 | **✅ T2 CORRECT** — HbA1c monitoring frequency from Figure 11 (for both CKD G1-G3b and G4-G5) |
| **"CKD G4-G5 including treatment by dialysis or kidney transplant"** (D channel) | 1 | **✅ T2 CORRECT** — Population definition from Figure 11 |
| **"CKD G1-G3b"** (D channel) | 1 | **✅ T2 OK** — Population segment |
| **"Daily"** (C channel) | 1 | **→ T3** — Frequency word without context |
| **"In addition to long-term glycemic control, minute-to-minute glycemic variability and episodes of hypoglycemia are import..."** (F channel) | 1 | **⚠️ SHOULD BE T1** — Complete clinical sentence about hypoglycemia monitoring targets; directly relates to PP 2.1.4 |

**Summary: ~5/36 T2 correctly tiered (4 D channel Figure 11 content + 1 F channel sentence). 1 F channel sentence should be T1. ~31/36 are HbA1c repetitions or table fragments.**

---

## Critical Findings

### ✅ D Channel Figure 11 Decomposition — USEFUL
The D channel successfully extracts the key HbA1c monitoring frequency guidance from Figure 11:
- "Twice per year / Up to 4 times per year if not achieving target or change in therapy" (×2, for both CKD populations)
- "CKD G4-G5 including treatment by dialysis or kidney transplant"
- "CKD G1-G3b"

This is clinically actionable monitoring guidance, properly tiered as T2.

### ✅ F Channel Sentence — Genuine but Should Be T1
"In addition to long-term glycemic control, minute-to-minute glycemic variability and episodes of hypoglycemia are important therapeutic targets" — directly supports PP 2.1.4 and should be T1 (hypoglycemia prevention is patient safety).

### ❌ PP 2.1.1-2.1.4 ALL Labels Only (4 Practice Points Missing Text)
This is the most PPs on a single page in the audit, and none have their text captured. Key missing PP texts:
1. PP 2.1.1: HbA1c twice per year; up to 4×/year if not at target
2. PP 2.1.2: HbA1c low reliability in CKD G4-G5/dialysis
3. PP 2.1.3: GMI from CGM when HbA1c discordant
4. PP 2.1.4: Daily CGM/SMBG for hypoglycemia prevention

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 2.1.1 text: HbA1c frequency (twice/year, up to 4×/year) | **T1** | Monitoring schedule |
| PP 2.1.2 text: HbA1c low reliability in CKD G4-G5 | **T1** | Monitoring interpretation warning |
| PP 2.1.3 text: GMI from CGM when HbA1c discordant | **T1** | Alternative monitoring method |
| PP 2.1.4 text: Daily CGM/SMBG for hypoglycemia prevention | **T1** | Hypoglycemia safety monitoring |
| "HbA1c <7% reduces microvascular + macrovascular complications" | **T1** | Glycemic target evidence |
| "HbA1c may underestimate glycemia in advanced CKD" | **T1** | Monitoring bias direction |
| "CGM/SMBG not biased by CKD or dialysis/transplant" | **T1** | Alternative monitoring advantage |
| "If HbA1c < GMI: underestimates blood glucose by the difference" | **T2** | Practical interpretation guidance |
| GMI reliability: High for G1-G3b, Low for G4-G5 | **T2** | From Figure 11 (partially captured) |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — 4 Practice Points with only labels captured; HbA1c ×29 noise; Figure 11 D channel provides useful monitoring frequency data but is buried |
| **Tier corrections** | 4 PP labels: T1 → T3; HbA1c ×29: T2 → T3; F channel sentence: T2 → T1; "Yes" ×2: T2 → T3; "Measure": T2 → T3; "Daily": T2 → T3 |
| **Missing T1** | PP 2.1.1-2.1.4 full text, HbA1c target evidence, HbA1c bias direction, CGM/SMBG CKD advantage |
| **Missing T2** | GMI interpretation guidance, GMI reliability by CKD stage |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~15% — D channel captures monitoring frequency; 4 PP texts entirely missing |
| **Tier accuracy** | ~13% (0/4 T1 correct + ~5/36 T2 correct = 5/40) |
| **Noise ratio** | ~88% — HbA1c ×29, table fragments ×7 |
| **Genuine T1 content** | 0 extracted (1 F channel sentence should be T1) |
| **Prior review** | 0/40 reviewed |
| **Overall quality** | **POOR — ESCALATE** — 4 Practice Points missing text; page is a dense monitoring protocol |

---

## Chapter 2 Section 2.1 Summary (Pages 56-58)

| Page | Spans | Genuine T1 | Quality | Key Finding |
|------|-------|------------|---------|-------------|
| 56 | 6 | 0 (but 5 genuine T2) | GOOD | Best F channel page; biomarker limitation sentences |
| 57 | 41 | 0 | POOR | HbA1c ×35; zero clinical sentences |
| 58 | 40 | 0 (1 F should be T1) | POOR | 4 PP labels only; D channel Figure 11 useful |

**Pattern**: Section 2.1 (Glycemic Monitoring) is poorly served by the pipeline because:
1. No drugs are discussed → B channel silent
2. HbA1c is mentioned constantly → C channel explodes with lab name repetition
3. Practice Points about monitoring don't contain drug names → no B+C multi-channel rescue
4. Only F channel produces genuine sentences, and only on narrative evidence pages (56), not on PP-dense pages (57-58)

---

## Review Actions Completed

| Field | Value |
|-------|-------|
| **Review Date** | 2026-02-27 |
| **Reviewer** | claude-auditor |
| **Review Scope** | All 40 spans triaged; missing critical facts added |

### CONFIRMED Spans (5)

| Span ID (short) | Text | Channel | Note |
|------------------|------|---------|------|
| 77703b46 | "Twice per year / Up to 4 times per year if not achieving target or change in therapy" | D | Figure 11 HbA1c monitoring frequency for CKD G1-G3b. Target: KB-16. |
| 02cc683d | "Twice per year / Up to 4 times per year if not achieving target or change in therapy" | D | Figure 11 HbA1c monitoring frequency for CKD G4-G5. Target: KB-16. |
| 92d45aff | "CKD G4-G5 including treatment by dialysis or kidney transplant" | D | Figure 11 population definition. Target: KB-16. |
| 4850b69f | "CKD G1-G3b" | D | Figure 11 population segment. Target: KB-16. |
| c48a42ba | "In addition to long-term glycemic control, minute-to-minute glycemic variability and episodes of hypoglycemia are important therapeutic targets" | F | Clinical sentence supporting PP 2.1.4 (should be T1 for patient safety). Target: KB-4, KB-16. |

### REJECTED Spans (35)

| Category | Count | Reason | Note |
|----------|-------|--------|------|
| PP labels ("Practice Point 2.1.1" through "Practice Point 2.1.4") | 4 | out_of_scope | Labels only without accompanying text; no clinical content |
| "HbA1c" (C/D channel) | 27 | out_of_scope | Lab name single-token repetition; no clinical context |
| "Yes" (D channel) | 2 | out_of_scope | Decontextualized boolean from Figure 11 table |
| "Measure" (D channel) | 1 | out_of_scope | Column header fragment from Figure 11 |
| "Daily" (C channel) | 1 | out_of_scope | Frequency word without context |

### ADDED Facts (8)

| # | Text | Note | Target KBs |
|---|------|------|------------|
| 1 | Monitoring long-term glycemic control by HbA1c twice per year is reasonable for patients with diabetes | T1: PP 2.1.1 full text - HbA1c monitoring frequency | KB-16 |
| 2 | Accuracy and precision of HbA1c measurement declines with advanced CKD (G4-G5), particularly dialysis patients, in whom HbA1c measurements have low reliability | T1: PP 2.1.2 full text - HbA1c reliability warning | KB-16, KB-4 |
| 3 | A glucose management indicator (GMI) derived from CGM data can be used to index glycemia for individuals in whom HbA1c is not concordant with directly measured blood glucose levels or clinical symptoms | T1: PP 2.1.3 full text - GMI from CGM alternative | KB-16 |
| 4 | Daily glycemic monitoring with CGM or SMBG may help prevent hypoglycemia and improve glycemic control when glucose-lowering therapies associated with risk of hypoglycemia are used | T1: PP 2.1.4 full text - Daily CGM/SMBG for safety | KB-4, KB-16 |
| 5 | HbA1c may be measured up to 4 times per year if glycemic target not met or after therapy change | T1: PP 2.1.1 supplementary - increased frequency threshold | KB-16 |
| 6 | HbA1c may underestimate or overestimate glycemia in advanced CKD | T1: HbA1c bias direction in advanced CKD | KB-16 |
| 7 | CGM and SMBG yield direct glucose measurements not biased by CKD or its treatments including dialysis and transplant | T1: CGM/SMBG advantage over HbA1c in CKD | KB-16 |
| 8 | If HbA1c is lower than concurrent GMI, HbA1c underestimates blood glucose by the difference | T2: Practical interpretation guidance for HbA1c vs GMI discordance | KB-16 |

---

## Raw PDF Gap Analysis

| # | Gap Text | Priority | Rationale |
|---|----------|----------|-----------|
| 1 | "An HbA1c target of <7% (<53 mmol/mol) compared with 8-9% (64-75 mmol/mol) reduces the risk of microvascular diabetes complications and in some studies macrovascular complications" | **MODERATE** | HbA1c target evidence — glycemic threshold reduces microvascular + some macrovascular complications. KB-16, KB-4. |
| 2 | "No advantages of glycated albumin or fructosamine over HbA1c are known for monitoring glycemia in patients with CKD" | **MODERATE** | Confirms HbA1c primacy despite CKD limitations — alternative biomarkers have no demonstrated advantages. KB-16. |
| 3 | "HbA1c remains the biomarker of choice because glycated albumin and fructosamine have no demonstrated advantages and have clinically relevant assay biases with hypoalbuminemia" | **MODERATE** | Full justification for HbA1c primacy — alternatives worse due to hypoalbuminemia bias. KB-16. |
| 4 | "GMI derived from CGM is expressed in HbA1c-equivalent units (%) which facilitates interpretation" | **MODERATE** | GMI practical utility — expressed in same units as HbA1c for clinical ease. KB-16. |
| 5 | "GMI needs to be re-established when changes occur in factors affecting red blood cell turnover or protein glycation" | **MODERATE** | GMI re-calibration requirement — not a static measurement, must update when RBC turnover changes. KB-16. |

**All 5 gaps added via API (all 201).**

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Original spans** | 40 |
| **Confirmed** | 5 |
| **Rejected** | 35 |
| **Added (agent)** | 8 |
| **Added (gap fill)** | 5 |
| **Total Added** | 13 |
| **Total post-review** | 18 (5 confirmed + 13 added) |
| **Reviewed** | 40/40 original + 13 added = 53 total |
| **Pipeline 2 ready** | 18 spans |
| **Completeness (post-review)** | ~95% — All 4 PP full texts; HbA1c target evidence (<7% reduces complications); monitoring frequency (twice/year, up to 4×); HbA1c bias direction in advanced CKD; no advantages of alternative biomarkers; HbA1c primacy justification; CGM/SMBG unbiased by CKD; GMI from CGM when HbA1c discordant; GMI expressed in HbA1c units; GMI re-establishment requirement; HbA1c vs GMI interpretation; glycemic variability and hypoglycemia as therapeutic targets |
| **Remaining gaps** | Figure 11 GMI reliability column (High for G1-G3b, Low for G4-G5 — partially implied by confirmed D channel population segments + PP 2.1.2 text); "professional" vs real-time CGM distinction (T3 informational) |
| **Review Status** | COMPLETE |
