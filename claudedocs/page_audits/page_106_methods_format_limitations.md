# Page 106 Audit — Methods: Format for Recommendations, Limitations (4 Spans)

## Page Identity
- **PDF page**: S105 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Format for guideline recommendations (strength level 1/2, quality A-D), limitations of development process (RCT priority, search scope, no qualitative synthesis, no economic evaluations)
- **Clinical tier**: T3 (Informational — methodology format and limitations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 4 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 3 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), F (NuExtract LLM) |
| Tier accuracy | 0% (0/4) |
| Disagreements | 0 |
| Review Status | FINAL: 5 ADDED, 0 CONFIRMED, 9 REJECTED |
| Raw PDF Cross-Check | 2026-03-01 — 0 agent spans kept (4 original rejected + 5 paraphrased rejected and replaced with exact PDF text), 5 exact PDF gaps added |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | `<!-- PAGE 106 --> Format for guideline recommendations` | F | 90% | T1 | NOISE/BUG | HTML artifact variant (10th occurrence) + section title |
| 2 | "level 1" | C | 90% | T2 | T3 | GRADE level from methodology description |
| 3 | "level 2" | C | 90% | T2 | T3 | GRADE level from methodology description |
| 4 | "update (2022), there is unlikely to be practice-changing evidence beyond RCTs." | F | 85% | T2 | T3 | Limitations prose |

## PDF Source Content
- **Recommendation format**: Strength (Level 1 strong / Level 2 weak) + quality (A-D) + key information + rationale + SoF tables
- **Limitations**: RCT-only evidence, short 2020→2022 timeframe, non-exhaustive search, no qualitative evidence synthesis, no economic evaluations
- **Patient participation**: 2 people with diabetes and CKD on Work Group

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Methodology format and limitations disclosure.

---

## Raw PDF Cross-Check (2026-03-01)

### Pre-Review State
- **Total spans**: 4
- **ADDED**: 0
- **PENDING**: 4 (all original agent spans)
- **CONFIRMED**: 0
- **REJECTED**: 0

### Agent Spans Kept: 0
All 4 original agent spans rejected:
- `749f9e81` — F channel `<!-- PAGE 106 -->\nFormat for guideline recommendations` (HTML artifact + section title)
- `6d62b5b6` — C channel "level 1" (bare GRADE label, no clinical context)
- `9535d5e3` — C channel "level 2" (bare GRADE label, no clinical context)
- `0e1a0dd5` — F channel "update (2022), there is unlikely to be practice-changing evidence beyond RCTs." (partial limitations sentence)

### Pipeline 2 L3-L5 Assessment
All 4 original spans fail Pipeline 2 viability:
- **L3 (Claude fact extraction)**: No structured clinical fact extractable — no drug, lab, condition, or threshold
- **L4 (RxNorm/LOINC/SNOMED)**: No entities mappable to any terminology code
- **L5 (CQL schema)**: No content translatable to CQL decision logic
- **Target KBs**: Not applicable to KB-1 (dosing), KB-4 (safety), or KB-16 (monitoring)
- **KB-7 routing**: Gaps added below provide methodology reference data for KB-7 (Terminology)

### Paraphrased Gaps Rejected (5)
Initial gaps G106-A through G106-E were added from audit file content summaries (not exact PDF text). All 5 rejected and replaced with exact PDF versions below:
- `6f90cc10`, `dfef1f03`, `413ab248`, `3460e880`, `ca2d8055` — all rejected with note "Replacing with exact PDF text"

### Gaps Added (5) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G106-A2 | Format for guideline recommendations. Each guideline recommendation provides an assessment of the strength of the recommendation (strong, level 1; weak, level 2) and the quality of the evidence (A, B, C, D). The recommendation statements are followed by key information (benefits and harms, quality of the evidence, values and preferences, resource use and costs, considerations for implementation) and rationale. Each recommendation is linked to relevant SoF tables. In most cases, an underlying rationale supported each practice point. | Recommendation format — strength + quality + key information sections |
| G106-B2 | Limitations of the guideline development process. The evidence review for the guideline update prioritized RCTs as the primary source of evidence, and study types beyond RCTs have not been considered for the update. However, considering the short timeframe between the previous guideline version (2020) and the guideline update (2022), there is unlikely to be practice-changing evidence beyond RCTs. | Limitation 1 — RCT-prioritized evidence, short 2020-2022 timeframe |
| G106-C2 | The search strategy for the guideline update has relied on a well-maintained, expertly controlled database of RCTs in kidney disease. However, the search strategies were not exhaustive, as specialty and regional databases were not searched, and hand-searching of journals was not performed for the included reviews. | Limitation 2 — search scope, non-exhaustive |
| G106-D2 | Two people living with diabetes and CKD were members of the Work Group and provided invaluable perspectives and lived experiences for the development of these guidelines. However, in the development of these guidelines, no scoping exercise with patients, searches of the qualitative literature, or formal qualitative evidence synthesis examining patient experiences and priorities were undertaken. | Limitation 3 — 2 patient reps but no qualitative evidence synthesis |
| G106-E2 | As noted, although resource implications were considered in the formulation of recommendations, no economic evaluations were undertaken. | Limitation 4 — no economic evaluations |

### Post-Review State
- **Total spans**: 14
- **ADDED**: 5 (exact PDF gaps)
- **CONFIRMED**: 0
- **REJECTED**: 9 (4 original agent noise + 5 paraphrased gaps replaced)
- **P2-ready facts**: 5

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G106-A2 | KB-7 (Terminology) | Recommendation format — strength, quality, key information structure |
| G106-B2 | KB-7 (Terminology) | Limitation 1 — RCT focus and update timeframe |
| G106-C2 | KB-7 (Terminology) | Limitation 2 — search scope (non-exhaustive) |
| G106-D2 | KB-7 (Terminology) | Limitation 3 — patient participation, no qualitative synthesis |
| G106-E2 | KB-7 (Terminology) | Limitation 4 — no economic evaluations |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-03-01 (claude-auditor via API)
- **Page in sequence**: 106 of 126
