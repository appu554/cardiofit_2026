# Pages 15–16 Audit — Work Group Membership & Evidence Review Team

| Field | Value |
|-------|-------|
| **Pages** | 15–16 (PDF pages S14–S15) |
| **Content Type** | Work Group membership list and Evidence Review Team |
| **Extracted Spans** | 15 (pg 15) + 9 (pg 16) = 24 total |
| **Channels** | F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | COMPLETE: 24/24 REJECTED |
| **Risk** | No clinical content — personnel listing |
| **Audit Date** | 2026-02-25 (created), 2026-02-26 (review executed) |
| **Cross-Check** | Verified against raw spans — counts confirmed (pg 15: 15, pg 16: 9, total: 24), channels confirmed (F only), all T2 confirmed |
| **Page Decisions** | Page 15: FLAGGED, Page 16: FLAGGED |
| **Reviewer** | Claude (automated via KB0 Governance Dashboard UI) |

---

## Source PDF Content

**Page 15 (S14) — Work Group Membership:**
- Work Group Co-Chairs: Ian H. de Boer (U. of Washington), M. Luiza Caramori (U. of Minnesota)
- Work Group Members: ~15 named researchers with institutional affiliations
- Including: Hiddo J.L. Heerspink, Peter Rossing, Vlado Perkovic, etc.

**Page 16 (S15) — Evidence Review Team:**
- Cochrane Kidney and Transplant, Sydney, Australia
- ERT Director: Jonathan C. Craig
- ERT Co-Director: Giovanni F.M. Strippoli
- Project Team Leader: David J. Tunnicliffe
- Additional ERT members

---

## Extraction Analysis

### What Was Extracted

**Page 15 (15 spans):**

| Category | Count | Channel | Current Tier | Issue |
|----------|-------|---------|-------------|-------|
| "<!-- PAGE 15 --> Work Group membership" | 1 | F | T2 | Pipeline artifact + heading |
| "WORK GROUP CO-CHAIRS" | 1 | F | T2 | Section heading |
| Names with affiliations | 13 | F | T2 | Researcher names and institutional affiliations |

**Page 16 (9 spans):**

| Category | Count | Channel | Current Tier | Issue |
|----------|-------|---------|-------------|-------|
| "<!-- PAGE 16 --> EVIDENCE REVIEW TEAM" | 1 | F | T2 | Pipeline artifact + heading |
| "Cochrane Kidney and Transplant, Sydney, Australia" | 1 | F | T2 | Organization name |
| ERT member names with credentials | 7 | F | T2 | Personnel listings |

### Tier Assignment Issues

| Current Tier | Count | Should Be | Reason |
|-------------|-------|-----------|--------|
| **T1** | 0 | — | Correct |
| **T2** | 24 | **T3** | Personnel listings — no clinical content |

### Severity: LOW (correct that T1=0, low impact)

---

## Specific Problems

### Problem 1: Personnel Names as T2
Researcher names and institutional affiliations have zero clinical decision-making value. They are authorship metadata.

### Problem 2: Pipeline Artifacts
Both pages have "<!-- PAGE X -->" HTML comments in the first span.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page Decision** | **FLAG** — Personnel listings, no clinical content |
| **T2 spans** | All 24 → T3 or REJECT |
| **Missing content** | None expected |

---

## Pipeline 2 L3–L5 Analysis

### Why These Pages Have Zero Pipeline 2 Value

Pipeline 2 converts verified spans into structured KB facts through a layered process:

| Layer | Function | Pages 15–16 Outcome |
|-------|----------|---------------------|
| **Dossier Assembly** | Groups spans by drug anchor (Channel B match) | No Channel B drug anchors on personnel pages — cannot create DrugDossier objects |
| **L2.5 RxNorm** | Normalizes drug names to RxCUI codes | No drug names present — nothing to normalize |
| **L3 Claude** | Extracts structured facts (dosing, thresholds, conditions) | No dossiers arrive — layer never invoked |
| **L4 Terminology** | Maps to SNOMED/LOINC/ICD-10 codes | No facts to code |
| **L5 CQL** | Generates executable CDS rules | No coded facts — no CQL output |

**Root cause**: Personnel metadata contains zero drug-anchored content. The F channel (NuExtract LLM) lacks domain awareness to distinguish authorship listings from clinical recommendations, so it extracted every paragraph indiscriminately. Without Channel B drug dictionary matches, Dossier Assembly has no anchor to group these spans around, and the entire Pipeline 2 chain is never triggered.

### Target KB Impact Assessment

| Target KB | Relevant Data on Pages 15–16 | Impact |
|-----------|-------------------------------|--------|
| **KB-1** (Drug Dosing Rules) | None — no dosing values, no drug names | Zero |
| **KB-4** (Patient Safety / Contraindications) | None — no contraindication or safety data | Zero |
| **KB-16** (Lab Monitoring / Reference Ranges) | None — no lab values, thresholds, or monitoring intervals | Zero |

---

## Review Actions Executed (2026-02-26)

### What Was Done

All 24 spans across Pages 15–16 were **REJECTED** via the KB0 Governance Dashboard UI with reason **"Out of guideline scope"**. Both pages were **FLAGGED** as non-clinical content.

### Page 15 — Work Group Membership (15 spans rejected)

| # | Span Content | Action | Reason |
|---|-------------|--------|--------|
| 1 | `<!-- PAGE 15 --> Work Group membership` | REJECTED | Pipeline artifact + section heading |
| 2 | `WORK GROUP CO-CHAIRS` | REJECTED | Section heading, no clinical content |
| 3 | Ian H. de Boer, MD, MS — U. of Washington | REJECTED | Researcher name + affiliation |
| 4 | M. Luiza Caramori, MD, PhD, MSc — U. of Minnesota | REJECTED | Researcher name + affiliation |
| 5 | Hiddo J.L. Heerspink, PhD, PharmD — U. of Groningen | REJECTED | Researcher name + affiliation |
| 6 | Clint Hurst, BS — Patient Representative | REJECTED | Patient representative name |
| 7 | Kamlesh Khunti, MD, PhD — U. of Leicester | REJECTED | Researcher name + affiliation |
| 8 | Adrian Liew, MBBS — Mt Elizabeth Novena Hospital | REJECTED | Researcher name + affiliation |
| 9 | Peter Rossing, MD, DMSc — Steno Diabetes Center | REJECTED | Researcher name + affiliation |
| 10 | Wasiu A. Olowu, MBBS — Obafemi Awolowo U. | REJECTED | Researcher name + affiliation |
| 11 | Tami Sadusky, MBA — Patient Representative | REJECTED | Patient representative name |
| 12 | Nikhil Tandon, MBBS, MD, PhD — AIIMS New Delhi | REJECTED | Researcher name + affiliation |
| 13 | Christoph Wanner, MD — U. Hospital of Würzburg | REJECTED | Researcher name + affiliation |
| 14 | Katy G. Wilkens, MS, RD — Northwest Kidney Centers | REJECTED | Researcher name + affiliation |
| 15 | Sophia Zoungas, MBBS, FRACP, PhD — Monash U. | REJECTED | Researcher name + affiliation |

**Page 15 Decision**: FLAGGED (no clinical content — personnel listing)

### Page 16 — Evidence Review Team (9 spans rejected)

| # | Span Content | Action | Reason |
|---|-------------|--------|--------|
| 1 | `<!-- PAGE 16 --> EVIDENCE REVIEW TEAM` | REJECTED | Pipeline artifact + section heading |
| 2 | Cochrane Kidney and Transplant, Sydney, Australia | REJECTED | Organization name |
| 3 | Jonathan C. Craig, MBChB — ERT Director | REJECTED | ERT personnel name + role |
| 4 | Giovanni F.M. Strippoli, MD — ERT Co-Director | REJECTED | ERT personnel name + role |
| 5 | David J. Tunnicliffe, PhD — Project Team Leader | REJECTED | ERT personnel name + role |
| 6 | Gail Y. Higgins, BA — Information Specialist | REJECTED | ERT personnel name + role |
| 7 | Patrizia Natale, PhD — Research Associate | REJECTED | ERT personnel name + role |
| 8 | Tess E. Cooper, MPH, MSc — Managing Editor | REJECTED | ERT personnel name + role |
| 9 | Narelle S. Willis, BSc, MSc — Managing Editor | REJECTED | ERT personnel name + role |

**Page 16 Decision**: FLAGGED (no clinical content — personnel listing)

### Why "Out of Guideline Scope" Was Chosen

The reject reason "Out of guideline scope" was selected (rather than "Not present in source" or "Hallucinated content") because:

1. **The text IS present in the source PDF** — the F channel accurately extracted what was on the page
2. **The content is NOT hallucinated** — these are real researcher names from the actual KDIGO document
3. **The problem is scope** — personnel metadata falls outside the scope of what the extraction pipeline should capture for clinical decision support. Work Group membership lists and Evidence Review Team rosters have zero value for drug dosing (KB-1), patient safety (KB-4), or lab monitoring (KB-16)

### Dashboard State After Review

| Metric | Before | After |
|--------|--------|-------|
| T2 Reviewed | 798/3242 | 822/3242 (25%) |
| Pages Decided | 12/126 | 14/126 |
| Pages Flagged | 2 | 4 |

---

## Completeness Score

| Metric | Score |
|--------|-------|
| **Extraction completeness** | N/A (no clinical content) |
| **Tier accuracy** | 0% (0/24 correctly tiered) |
| **False positive rate** | 100% |
| **Overall page quality** | FAIL — but low impact (personnel pages) |
| **Review completion** | 100% (24/24 spans decided) |

---

## Lessons for Pipeline Improvement

1. **Page classifier needed**: A pre-extraction page classifier that tags pages as "front matter", "table of contents", "references", or "clinical content" would prevent the F channel from extracting non-clinical pages entirely
2. **F channel over-extraction**: NuExtract LLM treats every paragraph as potentially relevant — it needs domain-specific prompting or post-extraction filtering to skip personnel metadata
3. **Tier assignment gap**: All 24 spans were assigned T2 (Clinical Accuracy) when they should have been T3 (Informational) at most — the tier classifier lacks awareness of content type vs. page type
