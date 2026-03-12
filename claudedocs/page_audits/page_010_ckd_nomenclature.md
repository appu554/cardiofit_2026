# Page 10 Audit — CKD Nomenclature (GFR & Albuminuria Categories)

| Field | Value |
|-------|-------|
| **Page** | 10 (PDF page S9) |
| **Content Type** | CKD staging nomenclature — GFR categories (G1–G5) and albuminuria categories (A1–A3) |
| **Extracted Spans** | 49 |
| **Channels** | C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 49 |
| **Risk** | Reference table — CKD staging definitions used throughout guideline |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (49), channels reordered, `<fcel>`/`<ecel>` table artifacts documented, D-channel heatmap multi-pass documented |

---

## Source PDF Content

Page 10 presents the **KDIGO 2012 CKD classification** heatmap:

**GFR Categories (ml/min/1.73 m²):**
| Category | GFR Range | Description |
|----------|-----------|-------------|
| G1 | ≥90 | Normal or high |
| G2 | 60–89 | Mildly decreased |
| G3a | 45–59 | Mildly to moderately decreased |
| G3b | 30–44 | Moderately to severely decreased |
| G4 | 15–29 | Severely decreased |
| G5 | <15 | Kidney failure |

**Albuminuria Categories:**
| Category | AER (mg/g) | ACR (mg/mmol) | Description |
|----------|-----------|---------------|-------------|
| A1 | <30 | <3 | Normal to mildly increased |
| A2 | 30–300 | 3–30 | Moderately increased |
| A3 | >300 | >30 | Severely increased |

This is a **reference classification table** used throughout the guideline when specifying which CKD stage a recommendation applies to.

---

## Extraction Analysis

### Channel Breakdown (49 spans)

| Channel | Spans | Count | Notes |
|---------|-------|-------|-------|
| **D (Table Decomp)** | #1–40 | 40 | CKD heatmap cells extracted individually with multi-pass |
| **F (NuExtract LLM)** | #41 | 1 | Pipeline artifact `<!-- PAGE 10 -->` heading |
| **C (Grammar/Regex)** | #42–49 | 8 | eGFR (2×), mg values (5×), mg unit |

### What Was Extracted (49 spans)

| Category | Count | Channel | Current Tier | Span #s |
|----------|-------|---------|-------------|---------|
| GFR range values (≥90, 60-89, 45-59, 30-44, 15-29) | 4 | D | T2 | #2–5 (first pass), #25 (≥90 duplicate) |
| Albuminuria threshold values (<30, <3, >300 mg) | ~6 | D | T2 | #1,#6,#7,#8,#24,#40 |
| CKD stage labels (G1–G5, G3a, G3b) | 12 | D | T2 | #10,#11,#16–19 (first pass), #26,#27,#33–36 (second pass) |
| Albuminuria category labels (A1, A2, A3) | 12 | D | T2 | #13–15, #20–22, #30–32, #37–39 (4× each from heatmap positions) |
| Stage descriptions ("Mildly to moderately decreased", "Mildly decreased") | 2 | D | T2 | #23, #28 |
| `<fcel>` artifact values | 4 | D | T2 | #1,#6,#7,#40: values with `<fcel>` table cell marker |
| `<ecel>` artifact values | 2 | D | T2 | #12,#29: `<15<ecel>` — end-cell marker |
| "eGFR" | 2 | C | T2 | #42, #43 |
| Unit/threshold strings (30 mg, 3 mg, 300 mg) | 5 | C | T2 | #44–48 + #49 |
| Pipeline artifact heading | 1 | F | T2 | #41: `<!-- PAGE 10 --> CURRENT CHRONIC KIDNEY DISEASE (CKD) NOMENCLATURE...` |

### D-Channel Heatmap Multi-Pass

The CKD staging heatmap (a matrix with GFR rows × albuminuria columns) was extracted through multiple passes:

- **G-stage labels** appear 2× each: G3a (#10, #26), G3b (#11, #27), G1 (#16, #33), G2 (#17, #34), G4 (#18, #35), G5 (#19, #36)
- **A-category labels** appear 4× each: A1 (#13, #20, #30, #37), A2 (#14, #21, #31, #38), A3 (#15, #22, #32, #39)
- The heatmap matrix structure caused D-channel to extract category labels from row headers, column headers, and individual cells, producing more duplication than simple double-pass

### `<fcel>` / `<ecel>` Table Parsing Artifacts

6 spans contain table cell boundary markers that leaked into the extracted text:
- `<fcel>` (field-cell start): spans #1 (`<30 mg/g<fcel>`), #6 (`<3 mg/mmol<fcel>`), #7 (`<30 mg/g<fcel>`), #40 (`<3 mg/mmol<fcel>`)
- `<ecel>` (end-cell): spans #12 (`<15<ecel>`), #29 (`<15<ecel>`)

These markers corrupt the extracted values and should be stripped during post-processing.

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 0 | — | Correct — no drug dosing or contraindications |
| **T2** | 49 | **T2 or T3** | **MIXED** — see analysis below |
| **T3** | 0 | — | — |

### Severity: MODERATE — Nuanced Assessment Required

**This page is unique among front matter pages because the CKD staging system IS clinically referenced throughout the guideline.** However, the staging table itself is a reference classification, not a monitoring instruction or dose adjustment threshold.

**Correct tiering analysis:**
- The GFR/albuminuria ranges (G1–G5, A1–A3) could be **T2** if treated as lab thresholds for staging decisions
- However, they are more accurately **T3** (informational reference table) because they don't prescribe actions — the guideline text on later pages says "for patients with G3a–G5..." and THAT usage creates the T1/T2 clinical fact
- Column headers, labels, and duplicates are clearly T3

---

## Specific Problems

### Problem 1: Classification vs Prescription
The CKD staging table defines categories (e.g., "GFR 30-44 = G3b = Moderately to severely decreased"). This is a **definition**, not a prescriptive threshold. The clinical action tied to G3b (e.g., "reduce SGLT2i dose when eGFR <30") appears on other pages.

### Problem 2: Pipeline Artifact
"<!-- PAGE 10 -->" HTML comment extracted as a span.

### Problem 3: Fragmented Heatmap Decomposition with Multi-Pass
The D channel decomposed the CKD heatmap into individual cell values without preserving row/column relationships. "30-44" alone is meaningless without knowing it corresponds to G3b. The matrix structure caused A-category labels to be extracted 4× each (from 4 different cell positions).

### Problem 4: `<fcel>` / `<ecel>` Table Boundary Artifacts
6 spans contain leaked table cell markers: `<30 mg/g<fcel>`, `<3 mg/mmol<fcel>`, `<15<ecel>`. These corrupt the extracted threshold values and should be stripped.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Reference classification table, not prescriptive clinical content |
| **T2 spans** | Re-tier all 49 to T3. The staging definitions are informational — clinical actions using these stages appear on later pages |
| **Missing content** | None — classification table is complete |
| **Root cause** | 1. D channel treats classification heatmap as clinical data; 2. Matrix structure causes multi-pass extraction (4× for A-categories); 3. `<fcel>`/`<ecel>` markers leak into span text; 4. No distinction between "definition" and "instruction" |
| **Pipeline recommendation** | 1. Strip `<fcel>`/`<ecel>` markers; 2. Deduplicate heatmap cell extraction; 3. Detect classification tables vs prescriptive tables |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | ~60% (staging categories extracted but relationships lost due to fragmentation) |
| **Tier accuracy** | 0% (all should be T3, currently T2) |
| **False positive rate** | 100% for T1 (0 correctly T1); ~100% for T2 (should all be T3) |
| **Missing T1 content** | 0 (none expected) |
| **Missing T2 content** | 0 (none expected on this page) |
| **D-channel multi-pass** | Confirmed — A-categories extracted 4× each, G-stages 2× each |
| **Pipeline artifacts** | 7 (1× `<!-- PAGE 10 -->`, 4× `<fcel>`, 2× `<ecel>`) |
| **Overall page quality** | PARTIAL — content extracted but fragmented, mistiered, and artifact-corrupted |
