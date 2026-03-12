# Page 105 Audit — Methods: Recommendation Grading, Tables 5-6, Practice Points (15 Spans)

## Page Identity
- **PDF page**: S104 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Grading recommendation strength (strong vs weak), Table 5 (KDIGO nomenclature Level 1/2), Table 6 (determinants of strength — benefits/harms, evidence quality, values/preferences, costs), practice points definition
- **Clinical tier**: T3 (Informational — methodology for grading)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 14 |
| T1 (Patient Safety) | 9 |
| T2 (Clinical Accuracy) | 5 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/14) |
| Disagreements | 0 |
| Review Status | FINAL: 10 ADDED, 0 CONFIRMED, 15 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 15 rejected: 4 D Table 6 cells, 4 C "strong recommendation", 2 C "weak recommendation", 2 D column headers, 2 F table titles, 1 F abbreviation), 10 gaps added |
| Cross-Check | 2026-02-25 — Count corrected 15→14, T1 12→9, T2 3→5; verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## T1 Spans (12) — ALL MISTIERED — Highest T1 in Methods

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "The higher the quality of the evidence, the more likely a strong recommendation is warranted..." | D | 92% | T3 | Table 6 cell — methodology prose |
| 2 | "The more variability or the more uncertainty in values and preferences, the more likely a weak recommendation is warrant..." | D | 92% | T3 | Table 6 cell |
| 3 | "The higher the cost of an intervention — that is, the more resources consumed — the less likely a strong recommendation..." | D | 92% | T3 | Table 6 cell |
| 4 | "The larger the difference between the desirable and undesirable effects, the more likely a strong recommendation is prov..." | D | 92% | T3 | Table 6 cell |
| 5 | "Table 5 \| KDIGO nomenclature and description for grading recommendations" | F | 90% | T3 | Table title |
| 6 | "Table 6 \| Determinants of the strength of recommendation" | F | 90% | T3 | Table title |
| 7-10 | "strong recommendation" (×4) | C | 90% | T3 | Phrase from Tables 5-6 methodology |
| 11-12 | "weak recommendation" (×2) | C | 90% | T3 | Phrase from Tables 5-6 methodology |

**Critical classifier failure**: The C channel regex matches "strong recommendation" and "weak recommendation" as T1 patient safety content. These phrases appear in the methodology section describing HOW KDIGO grades its recommendations — not actual clinical recommendations themselves. Similarly, D channel Table 6 cells discuss *when* to use strong vs weak grades, which the tier classifier misinterprets as clinical guidance.

## T2 Spans (3) — ALL MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Comment" | D | 92% | NOISE | Table 6 column header |
| 2 | "Factors" | D | 92% | NOISE | Table 6 column header |
| 3 | "KDIGO, Kidney Disease: Improving Global Outcomes." | F | 90% | T3 | Organization abbreviation |

## PDF Source Content
- **Recommendation grading**: Strong (Level 1 "We recommend") vs Weak (Level 2 "We suggest")
- **Table 5**: KDIGO nomenclature — implications for patients, clinicians, policy
- **Table 6**: 4 determinants — benefits/harms balance, evidence quality, values/preferences, resource use
- **Practice points**: Defined as consensus statements supplementing graded recommendations; based on expert judgment when insufficient evidence for grading
- **Patient representatives**: 2 people living with diabetes and CKD on Work Group

## Quality Assessment
| Dimension | Rating | Notes |
|-----------|--------|-------|
| Tier accuracy | 0% | 12 T1 on methodology page — worst mistiering in audit |
| Clinical safety risk | NONE | Methodology for grading recommendations |
| Key issue | C regex "strong/weak recommendation" triggers T1 in non-clinical context |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. KDIGO grading methodology and nomenclature tables.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 15
- **ADDED**: 0
- **PENDING**: 0
- **CONFIRMED**: 0
- **REJECTED**: 15 (all original agent spans already rejected)

### Agent Spans Kept: 0
All 15 original spans correctly rejected:
- 4 D channel Table 6 cells (benefits/harms, evidence quality, values/preferences, costs — methodology prose)
- 4 C channel "strong recommendation" (repeated phrase from Tables 5-6)
- 2 C channel "weak recommendation" (repeated phrase from Tables 5-6)
- 2 D channel column headers ("Comment", "Factors")
- 2 F channel table titles ("Table 5 | KDIGO nomenclature...", "Table 6 | Determinants...")
- 1 F channel abbreviation ("KDIGO, Kidney Disease: Improving Global Outcomes.")

### Gaps Added (10) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G105-A | Table 5 \| KDIGO nomenclature and description for grading recommendations. Level 1, strong, "We recommend": Most people in your situation would want the recommended course of action, and only a small proportion would not. Most patients should receive the recommended course of action. The recommendation can be evaluated as a candidate for developing a policy or a performance measure. | Table 5 — Level 1 strong recommendation nomenclature |
| G105-B | Level 2, weak, "We suggest": The majority of people in your situation would want the recommended course of action, but many would not. Different choices will be appropriate for different patients. Each patient needs help to arrive at a management decision consistent with her or his values and preferences. The recommendation is likely to require substantial debate and involvement of stakeholders before policy can be determined. | Table 5 — Level 2 weak recommendation nomenclature |
| G105-C | Based on feedback, the guideline was further revised by the Work Group, as appropriate. All Work Group members provided input on initial and final drafts of the guideline statements and guideline text, and approved the final version of the guideline. The ERT also provided a descriptive summary of the evidence quality in support of the graded recommendations. | Guideline revision and approval process |
| G105-D | Grading the strength of the recommendations. The strength of a recommendation is graded as strong or weak (Table 5). The strength of a recommendation was determined by the balance of benefits and harms across all critical and important outcomes, the grading of the overall quality of the evidence, patient values and preferences, resource use and costs, and other considerations (Table 6). | Recommendation strength grading — 4 determinants |
| G105-E | Balance of benefits and harms. The Work Group and ERT determined the anticipated net health benefit on the basis of expected benefits and harms across all critical and important outcomes from the underlying evidence review. | Determinant 1 — benefits/harms balance |
| G105-F | The overall quality of evidence. The overall quality of the evidence was based on the quality of evidence for all critical and important outcomes, taking into account the relative importance of each outcome to the population of interest. The overall quality of the evidence was graded A, B, C, or D (Table 3). | Determinant 2 — evidence quality graded A-D |
| G105-G | Patient preferences and values. The Work Group included 2 people living with diabetes and CKD. These members' unique perspectives and lived experience, in addition to the Work Group's understanding of patient preferences and priorities, also informed decisions about the strength of the recommendations. A systematic review of qualitative studies on patient priorities and preferences was not undertaken for this guideline. | Determinant 3 — patient preferences, 2 patient reps |
| G105-H | Resource use and costs. Healthcare and non-healthcare resources, including all inputs in the treatment management pathway, were considered in grading the strength of a recommendation. The following resources were considered: direct healthcare costs, non-healthcare resources (such as transportation and social services), informal caregiver resources (e.g., time of family and caregivers), and changes in productivity. No formal economic evaluations, including cost-effectiveness analysis, were conducted. | Determinant 4 — resource use, 4 resource types |
| G105-I | Practice points. In addition to graded recommendations, KDIGO guidelines now include "practice points" to help clinicians better evaluate and implement the guidance from the expert Work Group. Practice points are consensus statements about a specific aspect of care and supplement recommendations for which a larger quality of evidence was identified. These were developed when no formal systematic evidence review was undertaken, or if there was insufficient evidence to provide a graded recommendation. Practice points represent the expert judgment of the guideline Work Group, but they may be based on limited evidence. | Practice points definition — consensus statements |
| G105-J | Table 6 \| Determinants of the strength of recommendation. Balance of benefits and harms: The larger the difference between the desirable and undesirable effects, the more likely a strong recommendation is provided. Quality of evidence: The higher the quality of the evidence, the more likely a strong recommendation is warranted. However, there are exceptions for which low or very low quality of the evidence will warrant a strong recommendation. Values and preferences: The more variability or the more uncertainty in values and preferences, the more likely a weak recommendation is warranted. Resource use and costs: The higher the cost of an intervention—the more resources consumed—the less likely a strong recommendation is warranted. | Table 6 — 4 determinants with full descriptions |

### Post-Review State
- **Total spans**: 25
- **ADDED**: 10 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 15 (all original noise)
- **P2-ready facts**: 10

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G105-A, G105-B | KB-7 (Terminology) | Table 5 — KDIGO recommendation nomenclature (Level 1/2) |
| G105-C | KB-7 (Terminology) | Guideline revision and approval process |
| G105-D, G105-E | KB-7 (Terminology) | Recommendation strength grading methodology |
| G105-F | KB-7 (Terminology) | Evidence quality grading A-D (references Table 3) |
| G105-G | KB-7 (Terminology) | Patient preferences — Work Group composition |
| G105-H | KB-7 (Terminology) | Resource use and costs — 4 resource types |
| G105-I | KB-7 (Terminology) | Practice points definition and scope |
| G105-J | KB-7 (Terminology) | Table 6 — 4 determinants of recommendation strength |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 105 of 126
