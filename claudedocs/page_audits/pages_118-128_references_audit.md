# Pages 118-128 Reference Block Audit — KB-15 Evidence Metadata Extraction

**Job ID**: `df538e50-0170-4ef8-862d-5b0a7c48e4ff`
**Reviewer**: `claude-auditor`
**Date**: 2026-03-01
**Source**: KDIGO 2022 Clinical Practice Guideline for Diabetes Management in CKD
**Reference Range**: Refs 41-468 (pages 118-128)
**Sessions**: 3 (context continuations due to volume)

---

## Final Status Summary

| Page | Refs | ADDED | REJECTED | PENDING | Status |
|------|------|-------|----------|---------|--------|
| 118 | 41-85 | 12 | 36 | 0 | COMPLETE |
| 119 | 86-130 | 17 | 22 | 0 | COMPLETE |
| 120 | 131-172 | 30 | 25 | 0 | COMPLETE |
| 121 | 173-219 | 22 | 49 | 0 | COMPLETE |
| 122 | 220-265 | 13 | 22 | 0 | COMPLETE |
| 123 | 266-313 | 22 | 50 | 0 | COMPLETE |
| 124 | 314-361 | 11 | 37 | 0 | COMPLETE |
| 125 | 314-361 | 16 | 12 | 0 | COMPLETE |
| 126 | 362-405 | 17 | 53 | 0 | COMPLETE |
| 127 | 406-451 | 17 | 9 | 0 | COMPLETE |
| 128 | 452-468 | 5 | 4 | 0 | COMPLETE |
| **TOTAL** | **41-468** | **182** | **319** | **0** | **ALL COMPLETE** |

---

## KB-15 Evidence Object Classification Breakdown

### By Category (across all pages)

| Category | Count | Description |
|----------|-------|-------------|
| Landmark trial | ~65 | Pivotal RCTs with named trial acronyms (CREDENCE, DAPA-CKD, EMPA-REG, LEADER, etc.) |
| Canonical cluster summary | ~45 | Cochrane/systematic reviews/meta-analyses subsuming multiple RCTs |
| Evidence cluster | ~25 | Synthesized groupings of studies under canonical reviews |
| Cross-guideline linkage | ~30 | External guidelines (ADA, ACC/AHA, ESC, WHO, NICE, FDA, IDF) informing KDIGO |
| Meta-recommendation source | ~10 | Previous KDIGO versions or Controversies Conferences |
| Skip | 0 | All reference-page objects classified as one of the 5 active categories |

### Key Therapeutic Domains Covered

1. **SGLT2 inhibitors** (pp 118-120): CREDENCE, DAPA-CKD, EMPA-REG OUTCOME, CANVAS, DECLARE-TIMI 58, SCORED, VERTIS CV
2. **GLP-1 receptor agonists** (pp 125-126): LEADER, SUSTAIN-6, REWIND, Harmony Outcomes, EXSCEL, ELIXA, PIONEER 5/6, AMPLITUDE-O, AWARD-7
3. **Finerenone/MRAs** (pp 121): FIDELIO-DKD, FIGARO-DKD, FIDELITY, TOPCAT, RALES, ESAX-DN
4. **RAS blockade** (pp 119-122): RENAAL, IDNT, Lewis 1993, ONTARGET, VA NEPHRON-D, ALTITUDE
5. **Metformin** (pp 125-126): UKPDS 13, DeFronzo 1995, Salpeter Cochrane, Inzucchi 2014, Crowley 2017
6. **DPP-4 inhibitors** (p 125): CARMELINA
7. **Lifestyle/exercise** (pp 124-125): Look AHEAD, Heiwe Cochrane, exercise/PA cluster
8. **Self-management/education** (pp 126-127): Li Cochrane, Zimbudzi SR, Lim MA, SURE, TASMIN-SR
9. **Safety signals** (pp 121, 125-127): Hypoglycemia (ADVANCE), B12 deficiency (de Jager), lactic acidosis (Salpeter Cochrane)

---

## Rejection Pattern

All PENDING spans on reference pages were bare citation fragments — partial author names, journal abbreviations, volume/page numbers — produced by the PDF extraction pipeline splitting references at line boundaries. These carry no clinical semantic value and were rejected with note: "Reference page noise — bare citation fragment, not a clinically actionable span."

---

## Evidence Object Naming Convention

- Format: `G{page}-{letter}` (e.g., G118-A, G127-Q)
- Sequential within each page: A through the last letter needed
- Each object's `note` field contains: tag, KB-15 category, reference number, trial name/description, clinical relevance, target KBs

---

## Cross-KB Target Distribution

| Target KB | Description | Frequency |
|-----------|-------------|-----------|
| KB-15 | Evidence metadata (primary) | ALL objects |
| KB-1 | Dosing rules | ~40 objects |
| KB-4 | Patient safety | ~35 objects |
| KB-7 | Terminology/linkages | ~25 objects |
| KB-3 | Guidelines | ~15 objects |
| KB-16 | Monitoring | ~10 objects |

---

## Processing Methodology

1. **Raw PDF text** extracted to `claudedocs/raw_pdf_119-128/raw_pdf_119-128.md`
2. **Classification agents** (subagent_type=general-purpose) launched per page to classify each reference against the 6-category KB-15 framework
3. **Noise rejection** via Playwright browser_evaluate: bulk reject all PENDING spans per page
4. **Evidence addition** via Playwright browser_evaluate: batch add (6 per call) classified evidence objects with exact PDF text + structured notes
5. **Verification**: API query per page confirming 0 PENDING remaining

### API Pattern
- Auth: `GET /auth/access-token` (browser context, cookie-based)
- Add: `POST /api/v2/pipeline1/jobs/{JOB_ID}/spans/add` → 201
- Reject: `POST /api/v2/pipeline1/jobs/{JOB_ID}/spans/{id}/reject` → 200
- All calls via Playwright `browser_evaluate` (node-fetch unavailable in Node v24.7.0)
