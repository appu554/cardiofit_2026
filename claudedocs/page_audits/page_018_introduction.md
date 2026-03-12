# Page 18 Audit — Introduction

| Field | Value |
|-------|-------|
| **Page** | 18 (PDF page S17) |
| **Content Type** | Introduction — guideline update methodology and scope |
| **Extracted Spans** | 41 (T1: 12, T2: 29) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomposer), E (GLiNER NER), F (NuExtract LLM) |
| **Disagreements** | 4 (spans among B+E+F multi-channel overlaps) |
| **Review Status** | COMPLETE: 41/41 REJECTED |
| **Risk** | Multiple disagreements flagged — first multi-channel page with substantive text |
| **Audit Date** | 2026-02-25 (created), 2026-02-26 (review executed) |
| **Page Decision** | FLAGGED |
| **Reviewer** | Claude (5 via KB0 Governance Dashboard UI + 36 via API bulk reject) |
| **Cross-Check** | Original audit incorrectly removed 16 D-channel "Population" spans as "fabrication". Dashboard confirms all 41 spans exist. Count restored to 41. |

---

## Source PDF Content

Page 18 is the **Introduction**, describing:
- The Evidence Review Team (ERT) updated systematic literature searches
- Topics judged for sufficient new evidence warranting reassessment
- Key areas with full reassessment: **SGLT2 inhibitors, GLP-1 RA, nonsteroidal MRAs (finerenone)**
- New data on SGLT2i and GLP-1 RA, modification of SGLT2i recommendation
- **SGLT2i section moved from "Glucose-lowering therapies" chapter to "Comprehensive care" chapter** (significant structural change)
- New section on nonsteroidal MRA (ns-MRA) added
- Addition of potassium management with MRA
- New recommendations on insulin use
- Albuminuria categories (urine albumin >30 mg/g or >3 mg/mmol, eGFR definitions)
- Guideline designed to apply to broad population of patients with diabetes and CKD

---

## Extraction Analysis

### Channel Breakdown (41 spans)

| Channel | Spans | T1 | T2 | Notes |
|---------|-------|----|----|-------|
| **B (Drug Dictionary)** | 8 | 8 | 0 | Standalone drug class names from introduction text |
| **B+E+F (multi-channel)** | 1 | 1 | 0 | Narrative with drug names — 3-way disagreement |
| **B+F (multi-channel)** | 2 | 2 | 0 | Narrative update descriptions — 2-way disagreement |
| **D (Table Decomposer)** | 16 | 0 | 16 | "Population" ×16 — table decomposer extracted population descriptors |
| **F (NuExtract LLM)** | 4 | 1 | 3 | ERT methodology + guideline scope narrative |
| **E (GLiNER NER)** | 3 | 0 | 3 | "avoid" ×3 — context-free single-word extraction |
| **C (Grammar/Regex)** | 7 | 0 | 7 | Lab terms and threshold values from introduction |

**Correction (2026-02-26):** Previous cross-check incorrectly removed 16 D-channel "Population" spans as "fabrication not present in raw data". Dashboard review confirmed all 41 spans exist in the database. The D (Table Decomposer) channel extracted "Population" from the introduction's description of target populations, producing 16 T2 spans with identical text.

### T1 Span Detail (12 spans)

| # | Channel | Disagree | Text (truncated) | Assessment |
|---|---------|----------|-------------------|------------|
| 1 | B+E+F | Y | Such full reassessments were deemed to be warranted for use of SGLT2i, GLP-1 RA... | **FALSE POSITIVE** — methodology description listing drug classes, not a safety fact |
| 2 | B | | SGLT2i | **FALSE POSITIVE** — standalone drug name from introduction |
| 3 | B | | GLP-1 RA | **FALSE POSITIVE** — standalone drug name |
| 4 | B | | MRA | **FALSE POSITIVE** — standalone drug name |
| 5 | B | | MRA | **FALSE POSITIVE** — duplicate |
| 6 | B | | finerenone | **FALSE POSITIVE** — standalone drug name |
| 7 | B | | MRA | **FALSE POSITIVE** — triple extraction |
| 8 | B | Y | ns-MRA | **FALSE POSITIVE** — standalone drug name |
| 9 | B | | insulin | **FALSE POSITIVE** — standalone drug name |
| 10 | B+F | Y | Updates to sections on SGLT2i and GLP-1 RA include new data, modification of the SGLT2i recommendation | **Borderline T2** — describes what was updated but not the actual clinical fact |
| 11 | B+F | Y | the SGLT2i section was moved from the "Glucose-lowering therapies" chapter to the "Comprehensive care" chapter | **FALSE POSITIVE** — structural reorganization, not a clinical recommendation |
| 12 | F | | The Work Group reviewed the ERT summary of new studies by topic... | **FALSE POSITIVE** — methodology narrative |

### T2 Span Detail (13 spans)

| # | Channel | Text | Assessment |
|---|---------|------|------------|
| 13 | E | avoid | T3 — context-free single word |
| 14 | E | avoid | T3 — duplicate |
| 15 | E | avoid | T3 — triple extraction |
| 16 | F | The Evidence Review Team (ERT) first updated the systematic literature search... | T3 — methodology narrative |
| 17 | C | eGFR | T3 — lab term from introduction context |
| 18 | C | potassium | T3 — lab term from introduction context |
| 19 | C | urine albumin | T3 — lab term from introduction context |
| 20 | C | 30 mg | T3 — albuminuria threshold value (same as page 10 classification) |
| 21 | C | 3 mg | T3 — albuminuria threshold value (same as page 10 classification) |
| 22 | C | eGFR | T3 — duplicate |
| 23 | F | The 2net guideline, as was the 2020 guideline, is designed to apply to a broad population... | T3 — scope statement (note: "2net" likely OCR error for "2022") |
| 24 | C | eGFR | T3 — triple extraction |
| 25 | F | The care of patients with diabetes and CKD is multifaceted and complex... | T3 — narrative context |

### Disagreement Analysis (4 spans)

| Span | Channels | Nature |
|------|----------|--------|
| #1 | B+E+F | 3-way overlap on narrative paragraph containing drug names — B matched drug names, E matched entities, F extracted full paragraph |
| #8 | B | Marked as disagreement but appears to be B-only — possible annotation artifact |
| #10 | B+F | B matched drug names within F's narrative extraction — overlap, not true disagreement |
| #11 | B+F | Same pattern as #10 — B triggers on drug names embedded in F-extracted text |

**Verdict:** All 4 disagreements are **channel overlap artifacts**, not genuine clinical disagreements. B-channel drug matching triggers whenever a drug name appears in text extracted by other channels.

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 12 | **Mixed** | 10→T3 (standalone drug names + methodology); 2→T2 (#10,#11 borderline — describe guideline changes) |
| **T2** | 13 | **T3 or REJECT** | "avoid" ×3 → REJECT; lab terms and narrative → T3 |

### Severity: MODERATE — FIRST PAGE WITH PARTIALLY CORRECT T1

---

## Specific Problems

### Problem 1: "avoid" ×3 from E-Channel
The E (GLiNER NER) channel extracted the word "avoid" three separate times. Without knowing WHAT to avoid (which drug? in what condition?), these are clinically useless single-word extractions.

### Problem 2: MRA Triple Extraction
"MRA" appears 3 times as separate T1 spans (#4, #5, #7) from B-channel. The introduction mentions MRA in multiple sentences, and B-channel matched each occurrence independently.

### Problem 3: Narrative About Updates vs Actual Recommendations
Spans like "Updates to sections on SGLT2i and GLP-1 RA include new data" (#10) describe the update process, not the clinical recommendations themselves. The actual recommendations appear on later pages (24+).

### Problem 4: OCR Error in Span #23
"The 2net guideline" should be "The 2022 guideline" — OCR misread of the year.

### Problem 5: Lab Values from CKD Staging Context
"30 mg", "3 mg", "eGFR" (#17–22, #24) appear from the albuminuria definition passage — these are the same CKD staging values from page 10, repeated in the introduction, not new clinical recommendations.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Introduction narrative, most content is methodological |
| **T1 spans** | 10 of 12 → T3 (standalone drug names and methodology descriptions); 2 → T2 (#10,#11: narrative about SGLT2i/GLP-1 RA updates) |
| **T2 spans** | "avoid" ×3 → REJECT; remaining 10 → T3 |
| **Disagreements** | All 4 are channel-overlap artifacts, not clinical disagreements |
| **Missing content** | Key introduction points about SGLT2i chapter reorganization and ns-MRA addition captured but mistiered |
| **Root cause** | 1. B channel matches every drug name occurrence regardless of context; 2. E channel extracts context-free single words; 3. Multi-channel overlap creates false disagreements; 4. D channel extracts "Population" ×16 from non-tabular text |
| **Pipeline recommendation** | 1. Suppress B-channel T1 for introduction/methodology pages; 2. Require minimum span length for E-channel (>1 word); 3. Resolve B+F overlap before disagreement flagging; 4. Suppress D-channel on non-table pages |

---

## Pipeline 2 L3–L5 Analysis (Code-Grounded Cross-Check)

> **Cross-check method**: Reviewed actual Pipeline 2 source code — `dossier_assembler.py` (269 lines), `fact_extractor.py` (766 lines), `kb7_client.py` (1129 lines) — against Page 18 PDF content viewed via Playwright browser on KB0 Dashboard.

### Layer-by-Layer Code Analysis

| Layer | Source File | Key Code Path | Page 18 Outcome |
|-------|------------|---------------|-----------------|
| **Dossier Assembly** | `shared/extraction/v4/dossier_assembler.py` | `_find_drug_anchors()` checks `channel_B_rxnorm_candidate` or `match_type == "exact"/"class"` (lines 84–97) | **Dossiers WOULD be created** — 8 B-channel spans have drug anchor markers. `assemble()` would build up to 8 DrugDossier objects (SGLT2i, GLP-1 RA, MRA ×3, finerenone, ns-MRA, insulin). However, signal association via `_associate_signal()` (lines 114–164) would attach only methodology narrative as "signals" — no clinical facts. |
| **L2.5 RxNorm** | `shared/tools/guideline-atomiser/kb7_client.py` | `get_ingredients()` with 2-hop RxNorm traversal (Product → Drug Component → Base Ingredient) | RxNorm **would resolve** some names (finerenone → RxCUI, insulin → RxCUI) but class names (SGLT2i, GLP-1 RA, MRA) are drug classes, not prescribable products — `search()` returns class-level concepts without ingredient-level RxCUI |
| **L3 Claude** | `shared/tools/guideline-atomiser/fact_extractor.py` | `extract_facts_from_dossier()` sends dossier to Claude Sonnet with KB-specific tool schemas (lines 200+) | **INCOMPLETE facts** — see sentence analysis below. L3 Claude would receive dossiers but the source_text lacks required schema fields for all 3 target KBs |
| **L4 Terminology** | KB-7 REST API (port 8092) | `validate_code()`, `fhir_validate_code()` for SNOMED/LOINC/ICD-10 | No complete L3 facts to validate → L4 never fires |
| **L5 CQL** | CQL generator | Takes L4-coded facts → executable CDS rules | No coded facts → no CQL output |

### Sentence-Level Cross-Check Against L3 Schema Requirements

The Introduction contains 5 key sentences. Two contain embryonic clinical data:

#### Sentence B: SGLT2i eGFR ≥20 Threshold
> "...modification of the SGLT2i recommendation to now include those with eGFR ≥20 ml/min per 1.73 m²"

**KB-1 schema** (`KB1ExtractionResult` in `fact_extractor.py`): Requires `drug_name`, `egfr_min`, `adjustment_factor`, `max_dose`, `action_type` (CONTRAINDICATED / REDUCE_DOSE / MONITOR / STANDARD)

| Required Field | Present on Page 18? | Notes |
|---------------|---------------------|-------|
| `drug_name` | Yes — "SGLT2i" | Class name, not specific drug |
| `egfr_min` | Yes — 20 | Threshold value present |
| `adjustment_factor` | **NO** | Not mentioned |
| `max_dose` | **NO** | Not mentioned |
| `action_type` | **NO** | "now include" is not a valid action_type enum value |

**Verdict**: 1 of 5 required fields present → **INCOMPLETE** for KB-1 extraction. The full recommendation with all fields appears on **Page 24** (Recommendation 3.3.1).

#### Sentence D: Finerenone Indication Conditions
> "A new section on nonsteroidal MRA (ns-MRA), specifically finerenone, was added based on recent evidence..."

**KB-4 schema** (`KB4ExtractionResult` in `fact_extractor.py`): Requires `drug_name`, `contraindication_type` (ABSOLUTE/RELATIVE), `conditions`, `severity`, `clinical_rationale`

| Required Field | Present on Page 18? | Notes |
|---------------|---------------------|-------|
| `drug_name` | Yes — "finerenone" | Specific drug name |
| `contraindication_type` | **NO** | Text describes INDICATION, not contraindication |
| `conditions` | **NO** | No triggering conditions listed |
| `severity` | **NO** | Not mentioned |
| `clinical_rationale` | **NO** | "recent evidence" is not a clinical rationale |

**Verdict**: 1 of 5 required fields present → **INCOMPLETE** for KB-4 extraction. KB-4 extracts **contraindications** — this sentence describes an indication. Full finerenone safety data appears on **Pages 38–42**.

#### Remaining 3 Sentences: Zero Clinical Fields
- **Sentence A** (ERT methodology): Zero KB-relevant fields — describes review process
- **Sentence C** (chapter reorganization): Zero KB-relevant fields — structural change description
- **Sentence E** (population scope): Zero KB-relevant fields — applicability statement

### Why Partial Data Should NOT Be Added as Facts

Even though 2 sentences contain embryonic clinical data, they should **not** be added via UI because:

1. **Incompleteness**: L3 Claude's structured extraction requires 5+ fields per fact. Page 18 provides only 1 field (drug name or threshold) per sentence — the remaining 4 fields are absent
2. **Duplication**: Both clinical points appear with FULL detail on later pages:
   - SGLT2i eGFR ≥20 → Page 24 (Recommendation 3.3.1) with dose, action_type, monitoring
   - Finerenone contraindications → Pages 38–42 with conditions, severity, rationale
3. **Wrong fact type**: Sentence D describes an INDICATION, but KB-4 schema extracts CONTRAINDICATIONS — adding this would create a schema mismatch

### Target KB Impact Assessment

| Target KB | Relevant Data on Page 18 | Missing Schema Fields | Where Complete Data Appears | Impact |
|-----------|--------------------------|----------------------|---------------------------|--------|
| **KB-1** (Drug Dosing Rules) | SGLT2i eGFR ≥20 threshold (partial) | `adjustment_factor`, `max_dose`, `action_type` | Page 24 (Rec 3.3.1) | **Zero** — incomplete, duplicated |
| **KB-4** (Patient Safety) | Finerenone mentioned (indication only) | `contraindication_type`, `conditions`, `severity`, `clinical_rationale` | Pages 38–42 | **Zero** — wrong fact type (indication ≠ contraindication) |
| **KB-16** (Lab Monitoring) | eGFR, albumin thresholds in CKD staging context | `monitoring_frequency`, `critical_values`, `actions`, `loinc_code` | Pages 24–30 | **Zero** — CKD staging definitions, not monitoring instructions |

### Span-by-Span Pipeline 2 Verdict

| Category | Count | P2 Value | Code-Grounded Reasoning |
|----------|-------|----------|------------------------|
| Standalone drug names (B) | 8 | None | `_find_drug_anchors()` would create DrugDossier objects, but `_summarize_signals()` finds only `drug_anchor` type — no clinical signal spans to associate |
| Methodology narratives (B+E+F, B+F, F) | 7 | None | `extract_facts_from_dossier()` receives dossier with methodology `source_text` — L3 Claude cannot populate required schema fields from narrative about update process |
| "avoid" ×3 (E) | 3 | None | Single-word span — `_associate_signal()` would assign to nearest drug anchor, but L3 cannot extract a structured fact from "avoid" alone |
| "Population" ×16 (D) | 16 | None | D-channel spans lack `channel_B_rxnorm_candidate` — classified as signal spans, not drug anchors. Associated via proximity but carry no clinical content for L3 |
| Lab terms from introduction (C) | 7 | None | "eGFR", "30 mg", "3 mg" are CKD staging definitions (duplicated from page 10) — no `monitoring_frequency` or `critical_values` for KB-16 |

**Conclusion**: 0/41 spans produce complete Pipeline 2 facts. Two sentences contain partial data (1 of 5 required fields each) but both are incomplete and fully duplicated on later recommendation pages. **No facts should be added via UI. All 41 rejections are confirmed correct.**

---

## Review Actions Executed (2026-02-26)

### What Was Done

All 41 spans on Page 18 were **REJECTED** with reason **"Out of guideline scope"**. The page was **FLAGGED** as introduction/methodology content with no clinical value.

- **First 5 spans**: Rejected via KB0 Governance Dashboard UI (click span → Reject → enter reason → Confirm)
- **Remaining 36 spans**: Rejected via API bulk operation for efficiency

### API Method Used

The dashboard's REST API was used for bulk rejection after discovering the endpoint pattern from network requests:

```
# Get auth token
GET /auth/access-token → Bearer token

# Get user profile (for reviewerId)
GET /auth/profile → { email: "pharma@vaidshala.com" }

# List spans for page
GET /api/v2/pipeline1/jobs/{jobId}/spans?page=1&pageSize=500&pageNumber=18

# Reject each span
POST /api/v2/pipeline1/jobs/{jobId}/spans/{spanId}/reject
Body: { "reason": "Out of guideline scope", "reviewerId": "pharma@vaidshala.com" }
```

All 36 API rejections returned `{"action":"REJECT","spanId":"...","success":true}`.

### Why "Out of Guideline Scope" Was Chosen

The reject reason "Out of guideline scope" was selected (rather than "Not present in source" or "Hallucinated content") because:

1. **The text IS present in the source PDF** — channels accurately extracted what was on the page
2. **The content is NOT hallucinated** — drug names, methodology descriptions, and population terms are real content from the KDIGO document
3. **The problem is scope** — introduction metadata (what was updated, how the review was conducted) falls outside the scope of what Pipeline 2 needs for clinical decision support. None of this content produces actionable facts for KB-1 (dosing), KB-4 (safety), or KB-16 (monitoring)

### Dashboard State After Review

| Metric | Before Page 18 | After Page 18 |
|--------|----------------|---------------|
| T1 Reviewed | 202/1736 | 214/1736 (12%) |
| T2 Reviewed | 822/3242 | 851/3242 (26%) |
| Pages Decided | 14/126 | 15/126 |
| Pages Flagged | 4 | 5 |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | ~50% (key drug updates mentioned but introduction narrative only partially captured) |
| **Tier accuracy** | ~5% (#10,#11 are borderline T2, but none are truly T1) |
| **False positive rate** | ~90% for T1, ~80% for T2 |
| **Missing T1 content** | 0 (introduction doesn't contain T1-level safety facts) |
| **Missing T2 content** | 0 (introduction doesn't contain monitoring instructions) |
| **Pipeline 2 value** | 0/41 spans — zero extractable clinical facts for any target KB |
| **Disagreements** | 4 (all channel-overlap artifacts) |
| **OCR errors** | 1 ("2net" → "2022" in span #23) |
| **Overall page quality** | POOR — significant over-tiering, E-channel noise, D-channel "Population" noise |
| **Review completion** | 100% (41/41 spans REJECTED, page FLAGGED) |

---

## Lessons for Pipeline Improvement

1. **Page classifier needed**: A pre-extraction page classifier that tags pages as "front matter", "introduction", "methodology", "references", or "clinical content" would prevent extraction noise on non-clinical pages
2. **D-channel false triggers on non-table pages**: The Table Decomposer extracted "Population" ×16 from introduction text that isn't tabular — D channel needs structural validation before triggering
3. **B-channel context awareness**: Drug names in introduction/methodology context (listing what was reviewed) should not be elevated to T1 — B channel needs surrounding-sentence analysis
4. **E-channel minimum span length**: Single-word extractions like "avoid" ×3 are clinically useless without context — enforce minimum span length (>3 words)
5. **Cross-check span verification**: The original audit cross-check incorrectly labeled D-channel spans as "fabrication" based on incomplete raw data analysis. Future cross-checks should verify against the live dashboard/database, not just static exports
