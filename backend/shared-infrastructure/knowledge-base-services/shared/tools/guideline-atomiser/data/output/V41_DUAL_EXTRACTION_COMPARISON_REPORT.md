# V4.1 Dual Extraction Comparison Report
## KDIGO 2022 Full Guide — Pages 58-61 (Chapter 2: Glycemic Monitoring)

**Pipeline Version**: 4.1.0
**Date**: 2026-02-09
**Source PDF**: `KDIGO-2022-Clinical-Practice-Guideline-for-Diabetes-Management-in-CKD.pdf`
**Pages Tested**: 58-61 (4 pages, standard portrait 594x783pt)
**Content**: HbA1c monitoring tables, CGM glossary, drug-hypoglycemia risk table, glycemic targets

---

## 1. Summary Scorecard

| Metric | Marker (L1) | Granite-Docling VlmPipeline (Channel A) |
|---|---|---|
| **Markdown output** | 20,516 chars (134 lines) | 9,781 chars (1,317 lines — inflated by hallucinations) |
| **Usable clinical content** | ~20,000 chars (faithful) | ~1,500 chars (then degenerates) |
| **Processing time** | 19.4 min (Docker CPU) | 44.3 min (native Apple Silicon CPU) |
| **Tables correctly extracted** | 3 (HbA1c, drug-hypo, glycemic targets) | 1 partial (HbA1c only, with errors) |
| **Headings detected** | 11 (TOC with bbox coordinates) | 0 (all 663 children labeled `?`) |
| **Practice Points captured** | 2.1.1 through 2.1.6 + Rec 2.2.1 (all correct) | 2.1.1, 2.1.5, 2.1.6 (paraphrased/hallucinated) |
| **Hallucinated content** | None | ~8,000 chars fabricated |
| **Text extraction method** | surya OCR (pages 0,2,3) + pdftext (page 1) | VLM autoregressive generation from rendered images |
| **Structural metadata** | Rich (block types, polygons, page stats) | Empty (label=`?` on all elements) |

---

## 2. Raw Output: Marker L1

### 2a. Marker Markdown (`marker_fullguide_58_61.md`)

**File**: `data/output/marker_fullguide_58_61.md`
**Size**: 20,516 chars, 134 lines

```markdown
www.kidney-international.org chapter 2

|                                                                      | HbA1c   |                                                                                                              |             |                     |
|----------------------------------------------------------------------|---------|--------------------------------------------------------------------------------------------------------------|-------------|---------------------|
| Population                                                           | Measure | Frequency                                                                                                    | Reliability | GMI                 |
| CKD G1–G3b                                                           | Yes     | <ul><li>Twice per year</li><li>Up to 4 times per year if not achieving target or change in therapy</li></ul> | High        | Occasionally useful |
| CKD G4–G5<br>including treatment by<br>dialysis or kidney transplant | Yes     | <ul><li>Twice per year</li><li>Up to 4 times per year if not achieving target or change in therapy</li></ul> | Low         | Likely useful       |

Figure 11 | Frequency of glycated hemoglobin (HbA1c) measurement and use of glucose management indicator (GMI) in chronic kidney disease (CKD). G1–G3b, estimated glomerular filtration rate (eGFR) ≥30 ml/min per 1.73 m²; G4–G5, eGFR <30 ml/min per 1.73 m².

Practice Point 2.1.1: Monitoring long-term glycemic control by HbA1c twice per year is reasonable for patients with diabetes. HbA1c may be measured as often as 4 times per year if the glycemic target is not met or after a change in glucose-lowering therapy.

[... full clinical text continues faithfully through all 4 pages ...]

# Practice Point 2.1.2: Accuracy and precision of HbA1c measurement declines with advanced CKD (G4–G5)...

Practice Point 2.1.3: A glucose management indicator (GMI) derived from continuous glucose monitoring (CGM) data...

Practice Point 2.1.4: Daily glycemic monitoring with CGM or self-monitoring of blood glucose (SMBG)...

# **Glossary of glucose monitoring terms**
### **Self-monitoring of blood glucose (SMBG)**
# **Continuous glucose monitoring (CGM)**
# **(a) Retrospective CGM**
# **(b) Real-time CGM (rtCGM)**
#### **(c) Intermittently scanned CGM**
# **Glucose management indicator (GMI)**

| Antihyperglycemic agents                                                                                      | Risk of hypoglycemia | Rationale for CGM or SMBG |
|---------------------------------------------------------------------------------------------------------------|----------------------|---------------------------|
| <ul><li>Insulin</li><li>Sulfonylureas</li><li>Meglitinides</li></ul>                                          | Higher               | Higher                    |
| <ul><li>Metformin</li><li>SGLT2 inhibitors</li><li>GLP-1 receptor agonists</li><li>DPP-4 inhibitors</li></ul> | Lower                | Lower                     |

Practice Point 2.1.5: For patients with T2D and CKD who choose not to do daily glycemic monitoring...
Practice Point 2.1.6: CGM devices are rapidly evolving with multiple functionalities...

#### Research recommendations
- Develop methods to identify patients for whom HbA1c produces a biased estimate...
- Develop methods to identify patients at high risk of hypoglycemia...
[7 bullet items]

# 2.2 Glycemic targets
Recommendation 2.2.1: We recommend an individualized HbA1c target ranging from <6.5% to <8.0%...

| < 6.5%       | HbA1c                                         | < 8.0%         |  |
|--------------|-----------------------------------------------|----------------|--|
| CKD G1       | Severity of CKD                               | CKD G5         |  |
| Absent/minor | Macrovascular complications                   | Present/severe |  |
| Few          | Comorbidities                                 | Many           |  |
| Long         | Life expectancy                               | Short          |  |
| Present      | Hypoglycemia awareness                        | Impaired       |  |
| Available    | Resources for hypoglycemia management         | Scarce         |  |
| Low          | Propensity of treatment to cause hypoglycemia | High           |  |

# **Key information**
Balance of benefits and harms. HbA1c targets are central to guide glucose-lowering treatment...
```

### 2b. Marker Structural Metadata (`marker_fullguide_58_61_meta.json`)

**File**: `data/output/marker_fullguide_58_61_meta.json`
**Size**: 8,205 bytes

**Table of Contents (11 headings with bounding-box polygons):**

| # | Title | Page | Coordinates |
|---|---|---|---|
| 1 | Practice Point 2.1.2: Accuracy and precision of HbA1c... | 0 | (42.75, 577.5) → (291.0, 624.3) |
| 2 | Glossary of glucose monitoring terms | 1 | (47.6, 66.9) → (188.2, 83.0) |
| 3 | Self-monitoring of blood glucose (SMBG) | 1 | (49.6, 85.6) → (183.7, 94.6) |
| 4 | Continuous glucose monitoring (CGM) | 1 | (51.3, 135.1) → (175.4, 142.6) |
| 5 | (a) Retrospective CGM | 1 | (50.5, 165.5) → (127.0, 175.5) |
| 6 | (b) Real-time CGM (rtCGM) | 1 | (160.4, 165.9) → (246.2, 175.0) |
| 7 | (c) Intermittently scanned CGM | 1 | (350.1, 167.5) → (453.9, 175.1) |
| 8 | Glucose management indicator (GMI) | 1 | (51.3, 291.3) → (172.6, 299.7) |
| 9 | Research recommendations | 2 | (302.2, 373.5) → (414.8, 381.9) |
| 10 | 2.2 Glycemic targets | 2 | (302.2, 624.0) → (402.0, 634.7) |
| 11 | Key information | 3 | (42.8, 358.5) → (110.3, 367.4) |

**Per-Page Block Counts:**

| Page | Text Method | Tables | SectionHeaders | Text Blocks | Captions | Pictures | ListItems |
|---|---|---|---|---|---|---|---|
| 0 (p58) | surya | 1 | 1 | 9 | 1 | 0 | 0 |
| 1 (p59) | pdftext | 0 | 7 | 9 | 1 | 2 | 0 |
| 2 (p60) | surya | 1 | 2 | 11 | 1 | 0 | 7 |
| 3 (p61) | surya | 1 | 1 | 9 | 1 | 0 | 0 |

---

## 3. Raw Output: Granite-Docling VlmPipeline

### 3a. Granite-Docling Markdown (`granite_docling_fullguide_58_61.md`)

**File**: `data/output/granite_docling_fullguide_58_61.md`
**Size**: 9,781 chars, 1,317 lines

**Usable content (lines 1-11):**
```markdown
Figure 11: Frequency of glycated hemoglobin (HbA1c) measurements in chronic kidney disease (CKD) and non-CKD patients.

|                                                      | HbA1c   | HbA1c                                                         | HbA1c       | HbA1c               |
|------------------------------------------------------|---------|---------------------------------------------------------------|-------------|---------------------|
| Population                                           | Measure | Frequency                                                     | Reliability | GMM                 |
| CKD G1-G3b                                           | Yes     | · Twice per year                                              | High        | Occasionally useful |
| CKD G4-G5                                            | Yes     | · Twice per year if not achieving target or change in therapy | Low         | Likely useful       |
| including treatment by dialysis or kidney transplant | Yes     | · Twice per year if not achieving target or change in therapy | Low         | Likely useful       |

Practice Point 2.1.1: Monitoring long-term glycemic control by HbA1c twice per year is reasonable for patients with diabetes. HbA1c may be measured as often as 4 times per year if the glycemic targget is not met or after a change in glucose-lowering therapy.

Copyright © 2019 Kidney International. All rights reserved.
```

**Hallucination begins (line 14):**
```markdown
## S58 2019

S58 2019

S58 2019
[... "S58 2019" repeated 290 times through line 580 ...]
```

**Fabricated clinical content (line 594):**
```markdown
Chapter 2 An important feature of the drug is the ability of the drug to raise blood glucose
levels. The drug is able to raise blood glucose levels by increasing the amount of glucose in
the blood. This is because the blood glucose levels are higher than the levels of glucose in
the urine.
```
> **CRITICAL**: This text does NOT exist in the PDF. It is entirely fabricated by the VLM.

**Incorrect drug table (lines 596-603):**
```markdown
| Anthiyperglycemic agents   | Risk of hypoglycemia   | Rationale for CGM or SMBG   |
|----------------------------|------------------------|-----------------------------|
| Insulin                    | Higher                 | Higher                      |
| Sulfonylureas              | Meglitinides           | Meglitinides                |
| Metformin                  | Lower                  | Lower                       |
| SGLT2 inhibitors           | Higher                 | Higher                      |
| CGP-1 receptor agonists    | Higher                 | Higher                      |
| DPP-4 inhibitors           | Higher                 | Higher                      |
```

> **ERRORS**: "Anthiyperglycemic" (typo), "CGP-1" should be "GLP-1", Sulfonylureas row has "Meglitinides" in wrong columns, SGLT2i/GLP-1 RA/DPP-4i all incorrectly show "Higher" risk (should be "Lower")

**Single-character hallucination loop (lines 619-1317):**
```
S
M
J
D
N
N
E
N
D
E
N
N
E
[... "N" repeated 330+ times ...]
```

### 3b. Granite-Docling Structural JSON (`granite_docling_fullguide_58_61.json`)

**File**: `data/output/granite_docling_fullguide_58_61.json`
**Size**: 2,062,824 bytes (2.0 MB — bloated by 663 hallucinated children)

**Schema**: DoclingDocument v1.9.0

**Structure Summary:**
```json
{
  "schema_name": "DoclingDocument",
  "version": "1.9.0",
  "body": {
    "children": [663 refs — all "#/texts/N"]
  },
  "tables": [1 table — the HbA1c table with errors],
  "texts": [662 text nodes — most contain "S58 2019" or single "N"],
  "pages": {
    "1": {"size": {"width": 594, "height": 783}},
    "2": {"size": {"width": 594, "height": 783}},
    "3": {"size": {"width": 594, "height": 783}},
    "4": {"size": {"width": 594, "height": 783}}
  }
}
```

**Critical structural failures:**
- **Headings**: 0 detected (all elements labeled `?` or unclassified)
- **Tables**: 1 detected (HbA1c table only, with column header errors)
- **Missing tables**: Drug-hypoglycemia table (Figure 13), HbA1c targets table (Figure 14)
- **663 body children**: ~10 are real content, ~653 are hallucinated repetitions

---

## 4. Error Catalog

### 4a. Granite-Docling Errors

| # | Error Type | Location | Detail |
|---|---|---|---|
| 1 | **Hallucination: repetitive loop** | Lines 14-580 | "S58 2019" repeated 290 times (page number artifact) |
| 2 | **Hallucination: fabricated text** | Line 594 | "An important feature of the drug is the ability..." — not in PDF |
| 3 | **Hallucination: repetitive loop** | Lines 619-1317 | Single character "N" repeated 330+ times |
| 4 | **Table header error** | Line 5 | "GMM" should be "GMI" |
| 5 | **Table header error** | Line 3 | All sub-columns labeled "HbA1c" instead of proper headers |
| 6 | **Table row split** | Lines 7-8 | "CKD G4-G5 including treatment by dialysis" split into 2 rows |
| 7 | **Table content error** | Line 599 | Sulfonylureas row: "Meglitinides" in Risk and Rationale columns |
| 8 | **Table content error** | Lines 601-603 | SGLT2i, GLP-1 RA, DPP-4i all show "Higher" risk (should be "Lower") |
| 9 | **Drug name error** | Line 602 | "CGP-1 receptor agonists" should be "GLP-1 receptor agonists" |
| 10 | **Spelling error** | Line 596 | "Anthiyperglycemic" should be "Antihyperglycemic" |
| 11 | **Spelling error** | Line 10 | "targget" should be "target" |
| 12 | **Paraphrased content** | Lines 611-617 | Practice Points 2.1.5, 2.1.6 are hallucinated paraphrases |
| 13 | **Missing content** | N/A | Pages 59 (glossary), 61 (glycemic targets) almost entirely missing |
| 14 | **Missing tables** | N/A | Drug-hypoglycemia table (Fig 13) and HbA1c targets table (Fig 14) not extracted |
| 15 | **No structural labels** | JSON | All 663 body children have label=`?` (unknown) |

### 4b. Marker Minor Issues

| # | Issue Type | Location | Detail |
|---|---|---|---|
| 1 | Heading level inconsistency | Lines 29-55 | Glossary terms use `#` (h1) instead of `###` (h3) — Marker promotes all headings |
| 2 | Page header/footer leakage | Lines 1, 27, 65, 104 | "www.kidney-international.org chapter 2" appears as body text |
| 3 | CID character artifacts | Not in markdown (resolved) | pdfplumber showed `(cid:129)` but Marker's surya OCR resolved them to bullet points |
| 4 | LaTeX notation | Lines 9, 116, 132 | `$\geq$` and `$\leq$` instead of Unicode ≥ and ≤ |

---

## 5. V4.1 Architecture Validation

This comparison validates the V4.1 hybrid design:

1. **Marker = L1 text-of-record** — Faithful, comprehensive, no hallucinations. Minor heading-level issues are easily correctable by Channel A regex.

2. **Granite-Docling VlmPipeline = structural oracle only** — Cannot be trusted for text content. Its autoregressive generation enters degenerate repetition loops on clinical documents. Only useful for detecting document structure (headings, table boundaries) from the first few tokens before hallucination onset.

3. **Channel A regex fallback = essential** — When VlmPipeline returns garbage (label=`?`, repetitive text), the regex parser must take over for heading/section detection.

4. **`_is_suspicious()` validation = critical** — The repetition detection in Channel A and Channel D must catch hallucinated table content before it enters the fact store.

---

## 6. Files Reference

| File | Size | Description |
|---|---|---|
| `marker_fullguide_58_61.md` | 20,516 bytes | Marker raw markdown output |
| `marker_fullguide_58_61_meta.json` | 8,205 bytes | Marker structural metadata (TOC, block counts, polygons) |
| `granite_docling_fullguide_58_61.md` | 9,785 bytes | Granite-Docling raw markdown output (contains hallucinations) |
| `granite_docling_fullguide_58_61.json` | 2,062,824 bytes | Granite-Docling full DoclingDocument JSON |
| `KDIGO-2022-full-guide-pages-58-61.pdf` | subset | 4-page PDF extract used for this comparison |

---

*Report generated by V4.1 Clinical Guideline Curation Pipeline dual extraction comparison.*
