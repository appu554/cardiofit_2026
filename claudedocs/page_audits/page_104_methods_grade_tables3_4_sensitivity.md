# Page 104 Audit — Methods: GRADE Tables 3-4, Sensitivity Analysis (29 Spans)

## Page Identity
- **PDF page**: S103 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Subgroup analysis continuation, sensitivity analyses, GRADE methodology, Table 3 (quality classification A-D), Table 4 (GRADE grading system — downgrade/upgrade factors), SoF tables, recommendation update process
- **Clinical tier**: T3 (Informational — GRADE methodology tables)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 31 |
| T1 (Patient Safety) | 2 |
| T2 (Clinical Accuracy) | 29 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/31) |
| Disagreements | 0 |
| Review Status | FINAL: 10 ADDED, 0 CONFIRMED, 29 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 29 rejected: 18 D table cells, 3 C bare terms, 8 F prose/headers), 10 gaps added |
| Cross-Check | 2026-02-25 — Count corrected 29→31, T2 27→29; verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## T1 Spans (2) — BOTH MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Table 4 \| GRADE system for grading quality of evidence" | F | 90% | T3 | Table title |
| 2 | "GRADE, Grading of Recommendations, Assessment, Development, and Evaluation; RCT, randomized controlled trial." | F | 90% | T3 | Abbreviation legend |

## T2 Spans (27) — ALL MISTIERED

### D Channel — Table 4 GRADE Cells (18 spans)
| Content Pattern | Count | Source |
|----------------|-------|--------|
| "High" | 4× | GRADE quality levels from Tables 3-4 |
| "Low" | 4× | GRADE quality levels |
| GRADE downgrade factors (-1 serious) | 5× | Study limitations, inconsistency, indirectness, imprecision, publication bias |
| GRADE upgrade factors | 3× | Strength of association, dose-response gradient, plausible confounding |
| GRADE process steps | 3× | Starting grade, Step 2 lower, Step 3 raise |

### C Channel (3 spans)
- "HbA1c" (85%) — bare lab name from methods context
- "Level 1" (90%) — from GRADE evidence levels
- "Level 2" (90%) — from GRADE evidence levels

### F Channel (6 spans including 2 T1)
- Table 3 title, subgroup analysis prose, glucose-lowering therapy subgroup methodology, sensitivity analyses heading

## PDF Source Content
- **Subgroup analysis**: I² statistic, P=0.1; CREDENCE (CKD-specific) vs DECLARE TIMI 58 (T2D with CKD subgroups)
- **Sensitivity analyses**: 4 types (unpublished exclusion, risk of bias, large study dominance, language/funding/country filters)
- **Table 3**: Quality classification (A=High through D=Very low) with meaning descriptions
- **Table 4**: GRADE system — starting grades, 5 downgrade factors (-1/-2), 3 upgrade factors (+1/+2)
- **SoF tables**: Available in Data Supplement Appendix C and D
- **Recommendation updates**: Virtual meetings 2021-2022, external public review

## Quality Assessment
| Dimension | Rating | Notes |
|-----------|--------|-------|
| Tier accuracy | 0% | All should be T3 |
| Clinical safety risk | NONE | GRADE methodology tables |
| Noise level | HIGH | "High" ×4, "Low" ×4 from D channel |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. GRADE methodology and evidence quality classification tables.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 29
- **ADDED**: 0
- **PENDING**: 0
- **CONFIRMED**: 0
- **REJECTED**: 29 (all original agent spans already rejected — D table cells, C bare terms, F prose fragments)

### Agent Spans Kept: 0
All 29 original spans correctly rejected:
- 18 D channel Table 3/4 cell fragments ("High" ×4, "Low" ×4, GRADE criteria cells)
- 3 C channel bare terms ("HbA1c", "Level 1", "Level 2")
- 8 F channel prose fragments and table headers

### Gaps Added (10) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G104-A | Table 3 \| Classification for certainty and quality of the evidence. Grade A, High: We are confident that the true effect is close to the estimate of the effect. Grade B, Moderate: The true effect is likely to be close to the estimate of the effect, but there is a possibility that it is substantially different. Grade C, Low: The true effect may be substantially different from the estimate of the effect. Grade D, Very low: The estimate of effect is very uncertain, and often will be far from the true effect. | Table 3 — GRADE evidence quality classification (A-D) |
| G104-B | of subgroup differences used the I2 statistic and a P value of 0.1 (noting that this is a weak test). | Subgroup differences — I-squared and P = 0.1 threshold (continued from p103) |
| G104-C | For glucose-lowering therapies, subgroup analysis was undertaken to assess effect modification of the population of the included studies. Studies that were designed specifically to assess the effects of glucose-lowering therapy in people with CKD and T2D (e.g., CREDENCE) were compared to studies in people with T2D that reported subgroups of people with CKD (e.g., DECLARE TIMI 58) to assess any subgroup differences. | Glucose-lowering subgroup — CREDENCE vs DECLARE TIMI 58 |
| G104-D | Sensitivity analyses. The following sensitivity analyses were considered: Repeating the analysis excluding unpublished studies. Repeating the analysis taking account of the risk of bias, as specified. Repeating the analysis excluding any very long or large studies to establish how much they dominate the results. Repeating the analysis excluding studies using the following filters: language of publication, source of funding (industry vs. other), and country in which the study was conducted. | Sensitivity analyses — 4 robustness checks |
| G104-E | Grading the quality of the evidence and the strength of a guideline recommendation. The overall quality of the evidence related to each critical and important outcome was assessed using the GRADE approach... The quality of the evidence is lowered in the event of study limitations; important inconsistencies in results across studies; indirectness of the results... imprecision in the evidence review results; and concerns about publication bias. | GRADE approach — 5 quality-lowering domains |
| G104-F | For imprecision, data were benchmarked against optimal information size, low event rates in either arm, CIs that indicate appreciable benefit and harm (25% decrease and 25% increase in the outcome of interest), and sparse data (only 1 study), all indicating concerns about the precision of the results. The final grade for the quality of the evidence for an outcome could be high, moderate, low, or very low (Tables 3 and 4). | Imprecision criteria — optimal information size, 25% threshold |
| G104-G | Table 4 \| GRADE system for grading quality of evidence. Starting grade: RCT = High, Observational = Low. Step 2—lower the grade: Study limitations (-1 serious, -2 very serious), Inconsistency (-1/-2), Indirectness (-1/-2), Imprecision (-1/-2), Publication bias (-1/-2). Step 3—raise the grade for observational studies: Strength of association (+1 large effect, +2 very large), Evidence of a dose-response gradient, All plausible confounding would reduce the demonstrated effect. | Table 4 — GRADE step-down/step-up system |
| G104-H | GRADE, Grading of Recommendations, Assessment, Development, and Evaluation; RCT, randomized controlled trial. | Table 4 abbreviation legend |
| G104-I | Summary of findings (SoF) tables. The SoF tables were developed to include a description of the population and the intervention and comparator. In addition, the SoF tables include results from the data synthesis as relative and absolute effect estimates. The grading of the quality of the evidence for each critical and important outcome is also provided in these tables. The SoF tables are available in the Data Supplement Appendix C and Appendix D. | Summary of Findings tables — structure and availability |
| G104-J | Updating and developing the recommendations. The guideline statements from the KDIGO 2020 Clinical Practice Guideline for Diabetes Management in CKD were considered in the context of new evidence by the Work Group Co-Chairs and Work Group members, and updated as appropriate. Recommendations were revised during virtual meetings in 2021–2022 and by e-mail communication. The final draft was sent for external public review. | Updating recommendations — 2020 guideline revision process |

### Post-Review State
- **Total spans**: 39
- **ADDED**: 10 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 29 (all original noise)
- **P2-ready facts**: 10

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G104-A | KB-7 (Terminology) | Table 3 — GRADE quality classification definitions (A-D) |
| G104-B, G104-C | KB-7 (Terminology) + KB-1 (Dosing) | Subgroup analysis — CREDENCE/DECLARE comparison methodology |
| G104-D | KB-7 (Terminology) | Sensitivity analyses — robustness methodology |
| G104-E, G104-F | KB-7 (Terminology) | GRADE approach — quality domains and imprecision criteria |
| G104-G, G104-H | KB-7 (Terminology) | Table 4 — GRADE grading system with step-down/step-up factors |
| G104-I | KB-7 (Terminology) | SoF table structure and availability |
| G104-J | KB-7 (Terminology) | Recommendation update process |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 104 of 126
