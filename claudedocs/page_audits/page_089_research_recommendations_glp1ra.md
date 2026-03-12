# Page 89 Audit — Research Recommendations (GLP-1 RA)

## Page Identity
- **PDF page**: S88 (Chapter 4 — www.kidney-international.org)
- **Content**: Research recommendations for future GLP-1 RA studies in CKD
- **Clinical tier**: T3 (Informational — research agenda, not actionable clinical guidance)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 12 |
| T1 (Patient Safety) | 10 |
| T2 (Clinical Accuracy) | 2 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), C (Grammar/Regex) |
| Genuinely correct T1 | 0 |
| Tier accuracy | 0% (0/12) |
| Disagreements | 0 |
| Review Status | FINAL: 14 ADDED, 0 PENDING, 12 REJECTED |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts confirmed (12 total: 10 T1, 2 T2), channels confirmed (B, C) |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept, 12 rejected (all noise), 14 gaps added (G89-A–G89-N) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| B (Drug Dictionary) | 10 | 100% | GLP-1 RA ×7, metformin ×1, SGLT2i ×2 |
| C (Grammar/Regex) | 2 | 85% | HbA1c ×2 |

## T1 Spans (10) — ALL MISTIERED

All 10 T1 spans are standalone drug class/name mentions from B channel at 100% confidence. These appear within research recommendation prose — not prescribing guidance, dosing limits, contraindications, or safety warnings.

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 2 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 3 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 4 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 5 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 6 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 7 | "GLP-1 RA" | B | 100% | T3 | Drug name in research context |
| 8 | "metformin" | B | 100% | T3 | Drug name in research context |
| 9 | "SGLT2i" | B | 100% | T3 | Drug name in research context |
| 10 | "SGLT2i" | B | 100% | T3 | Drug name in research context |

**Pattern**: B channel cannot distinguish between drug mentions in prescribing guidance (T1) vs. research recommendation prose (T3). Every drug mention on this page is within "future studies should examine..." framing — purely informational.

## T2 Spans (2) — ALL MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "HbA1c" | C | 85% | T3 | Lab name in research context |
| 2 | "HbA1c" | C | 85% | T3 | Lab name in research context |

**Pattern**: C channel extracts bare lab parameter names without thresholds/comparators. Within research recommendations, these are purely informational mentions.

## PDF Source Content Analysis

The page contains research recommendations for future GLP-1 RA studies, including:

1. **Kidney outcomes as primary endpoint** — Studies with kidney outcomes as primary endpoint for GLP-1 RA in CKD
2. **CKD populations** — Trials in CKD populations specifically
3. **Long-term safety >5 years** — Long-term safety studies beyond 5 years
4. **Severe CKD/dialysis** — Safety in severe CKD and dialysis patients
5. **Kidney transplant** — GLP-1 RA in kidney transplant recipients
6. **Biomarkers** — Biomarker studies for GLP-1 RA response prediction
7. **Primary prevention** — GLP-1 RA for primary kidney disease prevention
8. **T1D** — GLP-1 RA in type 1 diabetes
9. **Controlled HbA1c** — GLP-1 RA benefit when HbA1c already at target
10. **Tirzepatide/GIP-GLP-1 dual agonists** — Emerging combination therapies
11. **Cost-effectiveness** — Health economic analyses
12. **Combined SGLT2i + GLP-1 RA** — Combined therapy benefits
13. **Low-resource settings** — Implementation in resource-limited contexts

### Content Classification
**ALL content on this page is T3 (Informational)** — research agenda items suggesting future studies. None of this constitutes actionable clinical guidance for current patient management.

## Missing Content
- ~~None significant~~ All 13 research recommendations were missing — resolved by Raw PDF Cross-Check (14 gaps added)
- ~~Tirzepatide/GIP-GLP-1 dual agonist mention not captured~~ → Added as G89-K

## Cross-Page Patterns

### B Channel Drug Name Noise in Research Sections
This page is the clearest example of B channel's context-blindness:
- 10/10 T1 spans are drug mentions in research recommendation text
- The word "GLP-1 RA" appears 7 times, each extracted as a separate T1 span
- Research recommendation sections should be excluded from T1 classification entirely

### Channel Coverage
- Only B and C channels active — no D (no tables), no E (GLiNER), no F (NuExtract)
- Absence of D/E/F is appropriate — this is prose content with no structured data

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | LOW | Only drug names and lab names captured from rich research text |
| Tier accuracy | 0% | All 12 spans mistiered (should all be T3) |
| Clinical safety risk | NONE | Research recommendations have no patient safety implications |
| Channel diversity | LOW | Only B and C |
| Noise level | HIGH | 100% of spans are noise in T1/T2 context |

## Decision Recommendation
**ACCEPT** — Despite 0% tier accuracy, this page poses zero clinical risk. All content is T3 research recommendations. The mistiered drug names are systematic B channel noise that should be addressed at the pipeline level, not through individual page review.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 12
- **ADDED**: 0
- **PENDING**: 0
- **REJECTED**: 12 (all pre-existing agent spans were noise — bare drug/lab name mentions)

### Agent Spans Rejected (12)
All 12 original extraction spans were bare drug class names ("GLP-1 RA" ×7, "metformin" ×1, "SGLT2i" ×2) or lab parameter names ("HbA1c" ×2) from B and C channels. These are single-token mentions without clinical context — rejected as pipeline noise before this cross-check session.

### Gaps Added (14) — Exact PDF Text

| # | ID | Gap Text | Note |
|---|-----|----------|------|
| G89-A | `b607c4fd` | GLP-1 RA promotes weight loss in these individuals and can be a valuable tool to increase rates of pre-emptive and overall kidney transplants. | PP 4.2.5 conclusion — weight loss enables transplant eligibility |
| G89-B | `e12f056b` | Future GLP-1 RA studies should consider evaluating kidney outcomes as the primary outcome. | CRITICAL evidence gap — kidney outcomes as primary endpoint |
| G89-C | `85a0f67f` | Future evidence should confirm clinical evidence of cardiovascular outcome and kidney benefit of GLP-1 RA among patients with T2D in a population selected for CKD, as prior studies have examined only CKD subgroups enrolled in the main trials. | CKD-selected population needed, not just subgroup analysis |
| G89-D | `1a0ddbac` | Future studies should focus on long-term (>5 years) safety and efficacy of using GLP-1 RA among patients with T2D and CKD. We need continued longer safety follow-up data and post-marketing surveillance including real-world evidence studies. | Long-term safety >5 years + real-world evidence |
| G89-E | `688ca006` | Future studies should confirm the safety and clinical benefit of GLP-1 RA for patients with T2D with severe CKD, including those who are on dialysis, for whom there are limited data, and provide more data on CKD G4. | Severe CKD/dialysis safety gap |
| G89-F | `44eb960b` | Future studies should confirm the safety and clinical benefit of GLP-1 RA for patients with T2D and kidney transplant. | Kidney transplant safety gap |
| G89-G | `74763db2` | Future studies should examine what biomarkers are appropriate to follow to assess the clinical benefit of GLP-1 RA (i.e., HbA1c, body weight, blood pressure, albuminuria, etc.). | Biomarker selection for benefit assessment |
| G89-H | `e070bfdb` | Although the REWIND trial provided encouraging results about the cardiovascular outcome benefit of GLP-1 RA among patients with T2D and CKD without established CVD (i.e., exclusively primary prevention population), more population or trial data would be useful to confirm their role, as most studies have focused on secondary prevention. | REWIND primary prevention — more data needed |
| G89-I | `ae5d837e` | Future studies should focus on kidney and heart protective benefits of GLP-1 RA, as well as their safety, for use in patients with T1D. | T1D applicability gap |
| G89-J | `21feb0c9` | Future studies should examine whether there are safety and efficacy issues of GLP-1 RA among individuals with a history of T2D and CKD who now have controlled HbA1c <6.5%. | Controlled HbA1c — benefit question |
| G89-K | `78e95b9f` | Future studies are needed on the efficacy and safety of the newly FDA-approved tirzepatide in patients with diabetes and CKD. The dual agonists, glucose-dependent insulinotropic peptide-glucagon-like peptide 1 (GIP/GLP-1), are emerging as an additional therapeutic option, but currently, data are limited in this population. | Tirzepatide/GIP-GLP-1 dual agonists |
| G89-L | `7e726450` | Future studies should further investigate whether the cardiovascular and kidney benefits are increased when GLP-1 RA are combined with SGLT2i treatment. | GLP-1 RA + SGLT2i combination benefit |
| G89-M | `8a754255` | Future studies should report on the cost-effectiveness of this strategy that prioritizes adding a GLP-1 RA as a second-line pharmacologic agent, after metformin and an SGLT2i, among patients with T2D and CKD, rather than other antiglycemic medications, while factoring in cardiovascular and kidney benefits against the cost of medications and the potential for adverse effects. | Cost-effectiveness analysis |
| G89-N | `49d09319` | Future work should address how to better implement these treatment algorithms in clinical practice and how to improve availability and uptake in low-resource settings. | Implementation & low-resource access gap |

### Post-Review State
- **Total spans**: 26
- **ADDED**: 14 (all gap additions — 0 agent spans kept)
- **PENDING**: 0
- **REJECTED**: 12 (all original agent noise)
- **P2-ready facts**: 14

### Critical Finding
This page had **0% extraction utility** from the automated pipeline — all 12 original spans were bare drug/lab name tokens with no clinical content. The entire page's value (13 research recommendations + 1 practice point conclusion) was captured exclusively through manual gap addition. This represents the worst-case scenario for B/C channel extraction on prose-heavy research recommendation pages.

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G89-A | KB-1 (Dosing) | Weight loss → transplant eligibility clinical pathway |
| G89-B–G89-N | KB-4 (Safety) | Evidence gaps, safety unknowns, future study priorities |
| G89-K | KB-7 (Terminology) | GIP/GLP-1 dual agonist class definition |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 89 of 126
