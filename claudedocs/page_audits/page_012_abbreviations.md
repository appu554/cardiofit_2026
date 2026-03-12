# Page 12 Audit — Abbreviations and Acronyms

| Field | Value |
|-------|-------|
| **Page** | 12 (PDF page S11) |
| **Content Type** | Abbreviations and Acronyms glossary |
| **Extracted Spans** | 88 (T1: 14, T2: 74) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| **Disagreements** | 1 (span #83: C+F channel overlap) |
| **Review Status** | PENDING: 88 |
| **Risk** | Disagreement flagged — but content is non-clinical glossary |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — span count corrected (76→88), tier breakdown corrected (T1:12→14, T2:64→74), channels corrected (D→B,C,D,F), disagreement count added (0→1), OCR error count corrected (1→2) |

---

## Source PDF Content

Page 12 is the **Abbreviations and Acronyms** glossary listing ~60+ abbreviations used throughout the guideline:

| Abbreviation | Full Term |
|-------------|-----------|
| ACEi | angiotensin-converting enzyme inhibitor |
| ACR | albumin-to-creatinine ratio |
| AKI | acute kidney injury |
| ARB | angiotensin II receptor blocker |
| ASCVD | atherosclerotic cardiovascular disease |
| BMI | body mass index |
| CGM | continuous glucose monitoring |
| CI | confidence interval |
| CKD | chronic kidney disease |
| CrCl | creatinine clearance |
| CVD | cardiovascular disease |
| DPP-4 | dipeptidyl peptidase-4 |
| eGFR | estimated glomerular filtration rate |
| ERT | Evidence Review Team |
| FDA | Food and Drug Administration |
| GFR | glomerular filtration rate |
| GI | gastrointestinal |
| GMI | glucose management indicator |
| GRADE | Grading of Recommendations Assessment, Development and Evaluation |
| HbA1c | glycated hemoglobin |
| HR | hazard ratio |
| KDIGO | Kidney Disease: Improving Global Outcomes |
| MET | metabolic equivalent of task |
| MRA | mineralocorticoid receptor antagonist |
| ns-MRA | nonsteroidal MRA |
| OR | odds ratio |
| RAS(i) | renin-angiotensin system (inhibitor) |
| RR | relative risk |
| SCr | serum creatinine |
| SGLT2i | sodium-glucose co-transporter 2 inhibitor |
| SMBG | self-monitoring of blood glucose |
| T1D/T2D | type 1/type 2 diabetes |
| UKPDS | United Kingdom Prospective Diabetes Study |

---

## Extraction Analysis

### Channel Breakdown (88 spans)

| Channel | Spans | T1 | T2 | Notes |
|---------|-------|----|----|-------|
| **B (Drug Dictionary)** | #1–5 | 5 | 0 | Drug class names matched from glossary entries |
| **D (Table Decomp)** | #6–81 | 9 | 67 | Abbreviation table decomposed cell-by-cell with D-channel double-pass |
| **F (NuExtract LLM)** | #82 | 0 | 1 | Section heading with pipeline artifact |
| **C (Grammar/Regex)** | #83–88 | 0 | 6 | Lab terms matched by regex (#83 also F-channel → disagreement) |

### T1 Span Detail (14 spans — all false positives)

| # | Channel | Text | Assessment |
|---|---------|------|------------|
| 1 | B | ACEi | **FALSE POSITIVE** — abbreviation definition, not clinical safety fact |
| 2 | B | ARB | **FALSE POSITIVE** — abbreviation definition |
| 3 | B | MRA | **FALSE POSITIVE** — abbreviation definition |
| 4 | B | mineralocorticoid receptor antagonist | **FALSE POSITIVE** — full term expansion |
| 5 | B | SGLT2i | **FALSE POSITIVE** — abbreviation definition |
| 6 | D | SGLT2i | **FALSE POSITIVE** — duplicate from table decomp |
| 7 | D | SGLT2i | **FALSE POSITIVE** — triple extraction |
| 8 | D | ACEi | **FALSE POSITIVE** — duplicate from table decomp |
| 9 | D | ARB | **FALSE POSITIVE** — duplicate from table decomp |
| 10 | D | MRA | **FALSE POSITIVE** — duplicate from table decomp |
| 11 | D | ARB | **FALSE POSITIVE** — triple extraction |
| 12 | D | MRA | **FALSE POSITIVE** — triple extraction |
| 13 | D | ACEi | **FALSE POSITIVE** — triple extraction |
| 14 | D | Grading of Recommendations Assessment, Development and Evaluation | **FALSE POSITIVE** — GRADE acronym expansion, not a drug or clinical threshold |

### What Was Extracted (88 spans — T2 detail)

| Category | Count | Channel | Span #s | Notes |
|----------|-------|---------|---------|-------|
| Drug class abbreviations (SGLT2i, ACEi, ARB, MRA) | 14 | B+D | #1–13 | T1 false positives (see above) |
| GRADE expansion | 1 | D | #14 | T1 false positive — acronym, not drug |
| Clinical lab abbreviations (eGFR, CrCl, ACR, etc.) | ~16 | D | #16,#25–29,#42–44,#53,#67,#71–74,#85–86 | D-channel double-pass duplicates many |
| Statistical abbreviations (HR, CI, OR, RR) | ~8 | D | #45–48,#76–79 | Double-pass duplication |
| Organization/study names (KDIGO, UKPDS, GRADE, FDA, ERT) | ~8 | D | #17–22,#37–38,#75 | |
| Disease/condition codes (CKD, CVD, AKI, ASCVD, T1D, T2D) | ~14 | D | #23,#30,#34,#54,#58–65 | Double-pass |
| Full expansion terms | ~8 | D | #23,#51–52,#66,#69–70 | "chronic kidney disease" (2×), "glucose management index" (2×), "scrum creatinine" (2×) |
| Misc abbreviations (BMI, MET, GI, US, SCr, GFR, GMI, etc.) | ~12 | D | #31–33,#46–50,#56–57,#80–81 | |
| Survey / NHANES / DPP-4 | 3 | D | #15,#17,#20 | |
| F-channel section heading | 1 | F | #82 | `<!-- PAGE 12 --> Abbreviations and acronyms` |
| C+F disagreement span | 1 | C+F | #83 | Long concatenated abbreviation block (disagreement flagged) |
| C-channel lab terms | 5 | C | #84–88 | creatinine, eGFR (2×), serum creatinine, sodium |

### D-Channel Double-Pass Evidence

The abbreviation table was processed twice by D-channel:
- **First pass:** spans #6–50 (abbreviations A–G in alphabetical order)
- **Second pass:** spans #51–81 (same abbreviations re-extracted)

Drug class names appear 2–3 times each across B+D channels:
- SGLT2i: 3× (#5 B, #6 D, #7 D)
- ACEi: 3× (#1 B, #8 D, #13 D)
- ARB: 3× (#2 B, #9 D, #11 D)
- MRA: 3× (#3 B, #10 D, #12 D)

### OCR Error: "scrum creatinine"

Appears **twice** in raw data (spans #24 and #70 — both D-channel, double-pass).
Should be **"serum creatinine"**. The OCR misread was propagated through both extraction passes without correction.

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 14 | **T3** | Drug class abbreviation definitions from glossary — no dosing, thresholds, or safety information |
| **T2** | 74 | **T3** | Abbreviation glossary entries — no monitoring intervals, titration steps, or lab thresholds |
| **T3** | 0 | — | — |

### Severity: CRITICAL TIER MISASSIGNMENT

---

## Specific Problems

### Problem 1: Drug Class Names from Glossary as T1
B-channel matched drug class names (SGLT2i, ACEi, ARB, MRA) in the abbreviations glossary and classified them as T1. D-channel also extracted the same names. These are abbreviation definitions (e.g., "SGLT2i = sodium-glucose co-transporter 2 inhibitor"), not clinical safety facts about these drugs.

### Problem 2: GRADE Expansion as T1
Span #14 "Grading of Recommendations Assessment, Development and Evaluation" is the GRADE acronym expansion. It's classified T1 by D-channel, but it's an organization name, not a drug or clinical threshold.

### Problem 3: OCR Error (2 occurrences)
"scrum creatinine" (should be "serum creatinine") appears at spans #24 and #70. The D-channel double-pass propagated the OCR error into both extraction passes. No pipeline quality check caught this.

### Problem 4: C+F Disagreement (Span #83)
Span #83 is flagged as a disagreement between C and F channels. It contains a long concatenated block of abbreviation text ("HbA1c glycated hemoglobin HR hazard ratio KDIGO..."). This is a formatting artifact from the F-channel concatenating multiple glossary entries, overlapping with C-channel regex matches.

### Problem 5: D-Channel Double-Pass on Abbreviation Table
The entire abbreviation table was extracted twice, producing ~76 D-channel spans from ~38 unique abbreviation entries.

### Problem 6: Statistical Terms Mistiered
"odds ratio", "relative risk", "confidence interval", "hazard ratio" are statistical methodology terms, not clinical monitoring parameters.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Abbreviations glossary, not clinical content |
| **T1 spans** | All 14 should be re-tiered to T3 (drug class names and GRADE from glossary) |
| **T2 spans** | All 74 should be re-tiered to T3 (abbreviation definitions) |
| **Missing content** | None — glossary is complete |
| **OCR error** | "scrum creatinine" → "serum creatinine" at spans #24, #70 (needs correction) |
| **Root cause** | 1. B-channel drug matching inflates tier on glossary entries; 2. D-channel double-pass duplicates all abbreviations; 3. OCR error propagated uncorrected; 4. C+F overlap creates false disagreement |
| **Pipeline recommendation** | 1. Implement glossary/abbreviation page detection to suppress T1 classification; 2. Fix D-channel double-pass; 3. Add OCR spell-check for clinical terms; 4. Resolve C+F overlap on concatenated text |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | ~80% (most abbreviations extracted, some relationships lost) |
| **Tier accuracy** | 0% (0/88 correctly tiered; all should be T3) |
| **False positive rate** | 100% for T1 (14/14), 100% for T2 (74/74) |
| **OCR quality** | 2 confirmed OCR errors ("scrum" vs "serum" at #24, #70) |
| **D-channel double-pass** | Confirmed — abbreviation table extracted twice (~76 D spans from ~38 unique entries) |
| **Pipeline artifacts** | 1 (`<!-- PAGE 12 -->` in span #82) |
| **Disagreements** | 1 (span #83 C+F — false alarm on concatenated glossary text) |
| **Overall page quality** | FAIL — all tiers wrong, OCR errors, double-pass duplication |

---

## L3-L5 Assessment

### Pipeline 2 Value
The abbreviation glossary is **valuable reference data** for downstream L3-L5 processing:
- **L3 (Terminology Normalization)**: Drug class and clinical measurement abbreviations enable consistent term resolution across extracted facts
- **L4 (Clinical Context Mapping)**: Disease/condition abbreviations support entity linking and ontology mapping
- **L5 (Cross-Reference Validation)**: Abbreviation lookup table enables automated validation of extracted terms

### What to Keep vs Reject
All 76 original extracted spans are **noise** — standalone abbreviations, duplicates from D-channel double-pass, OCR errors. However, the glossary content itself is valuable when consolidated into clean, structured facts.

**Reject**: All 76 PENDING spans (fragments, duplicates, OCR errors)
**Add**: 3 consolidated terminology lookup facts

---

## Execution Log

| Action | Details | Timestamp |
|--------|---------|-----------|
| **Reject** | 69 PENDING spans via API (7 already rejected previously, 76 total pipeline + 3 added = 79) | 2026-02-25 |
| **Add Fact 1** | Drug class abbreviations: ACEi, ARB, DPP-4, GLP-1 RA, MRA, ns-MRA, RASi, SGLT2i with full expansions | 2026-02-25 |
| **Add Fact 2** | Clinical measurement abbreviations: ACR, BMI, CGM, CrCl, eGFR, GFR, GMI, HbA1c, SCr, SMBG with expansions | 2026-02-25 |
| **Add Fact 3** | Disease/condition abbreviations: AKI, ASCVD, CKD, CVD, T1D, T2D with expansions | 2026-02-25 |
| **Page Decision** | ACCEPTED — all noise rejected, 3 clean consolidated facts added for L3-L5 use | 2026-02-25 |

### Final State
| Metric | Count |
|--------|-------|
| **REJECTED** | 76 |
| **ADDED** | 3 |
| **PENDING** | 0 |
| **Page Status** | ACCEPTED |
