# Page 102 Audit — Methods: Table 2 End, Literature Search, Critical Appraisal (14 Spans)

## Page Identity
- **PDF page**: S101 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Table 2 conclusion (Ch5 healthcare delivery PICOM + abbreviation legend), literature search methodology (Cochrane Registry, MEDLINE, Embase → 346 RCTs + 31 observational + 50 systematic reviews), data extraction, critical appraisal (Cochrane Risk of Bias tool), evidence synthesis, measures of treatment effect
- **Clinical tier**: T3 (Informational — methodology)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES (C+F on abbreviation legend)

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 11 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 11 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/11) |
| Disagreements | 1 |
| Review Status | FINAL: 12 ADDED, 0 CONFIRMED, 14 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 14 rejected as C/D/F noise), 12 gaps added |
| Cross-Check | 2026-02-25 — Count corrected 14→11, T2 14→11; verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | 4 | 92% | Table 2 final cells (citations, interventions, SoF refs) |
| C (Grammar/Regex) | 8 | 85% | Bare lab names (serum creatinine ×3, creatinine, hemoglobin) + temporal words (monthly, weekly) |
| F (NuExtract LLM) | 2 | 85% | Methods prose (critical appraisal) + section heading |
| C+F (Multi-channel) | 1 | 93% | Abbreviation legend (disagreement) |

## T2 Spans (14) — ALL MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Tables S24-S25, S92, S93" | D | 92% | T3 | SoF table references |
| 2 | "Li et al. Education programmes for people with diabetic kidney disease..." | D | 92% | T3 | Cochrane citation |
| 3 | "Health service delivery programs/models of care" | D | 92% | T3 | PICOM Intervention cell |
| 4 | "Supplementary Tables S26-S28 and S94" | D | 92% | T3 | SoF table references |
| 5 | "serum creatinine" | C | 85% | T3 | Bare lab name from outcome definitions |
| 6 | "creatinine" | C | 85% | T3 | Bare lab name |
| 7 | "eGFR, estimated glomerular filtration rate; HbA1c, glycated hemoglobin; MACE, major cardiovascular events." | C+F | 93% | T3 | **Abbreviation legend** — DISAGREEMENT |
| 8 | "hemoglobin" | C | 85% | T3 | Bare lab name from abbreviation legend |
| 9 | "serum creatinine" | C | 85% | T3 | Duplicate |
| 10 | "monthly" | C | 85% | NOISE | Bare temporal word ("monthly searches of MEDLINE") |
| 11 | "weekly" | C | 85% | NOISE | Bare temporal word ("weekly searches of Cochrane Central") |
| 12 | "All critical appraisal was conducted independently by 2 members of the ERT..." | F | 85% | T3 | Methods prose — critical appraisal methodology |
| 13 | "Measures of treatment effect." | F | 85% | T3 | Section heading |
| 14 | "serum creatinine" | C | 85% | T3 | Bare lab name (3rd occurrence) |

### Disagreement Analysis
The C+F span captures the Table 2 abbreviation legend — a glossary of acronyms used throughout the table. Both channels extracted it independently: C regex matched the lab names (eGFR, HbA1c) while F recognized it as structured content. The combined 93% confidence is meaningless for a non-clinical abbreviation key.

### C Channel "monthly" and "weekly"
These bare temporal words come from the literature search methodology: "populated by **monthly** searches of the Cochrane Central Register of Controlled Trials, **weekly** searches of MEDLINE OVID." The C temporal regex captured them without context — they describe search frequency, not clinical monitoring intervals.

## PDF Source Content Analysis

### Content Present on Page
1. **Table 2 conclusion**: Ch5 healthcare delivery PICOM (models of care vs standard care, "None relevant" Cochrane reviews) + full abbreviation legend
2. **Literature search**: Cochrane Kidney and Transplant Registry, monthly Cochrane Central, weekly MEDLINE, yearly Embase, conference proceedings, trial registries
3. **Search results**: 2020 guideline → 5,667 citations; 2022 update → 1,078 citations screened → 102 RCTs included; **Total: 346 RCTs + 31 observational + 50 systematic reviews**
4. **Data extraction**: Independent by ERT member, confirmed by second, author contact for unclear data
5. **Critical appraisal**: Cochrane Risk of Bias tool — 7 domains (selection, detection, performance, attrition, reporting, other bias, sponsor involvement)
6. **Evidence synthesis**: Followed 2020 guideline methods
7. **Measures of treatment effect**: Dichotomous → RR with 95% CI; Time-to-event → HR with 95% CI; Continuous → [continues on next page]
8. **Figure 36 reference**: Study selection flow diagram

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| "346 RCTs, 31 observational studies, 50 systematic reviews" | T3 | NO |
| Cochrane Risk of Bias tool 7 domains | T3 | NO |
| "RR with 95% CI" treatment effect measure | T3 | NO |
| "HR with 95% CI" for time-to-event | T3 | NO |
| Abbreviation legend | T3 | YES (C+F 93%, mistiered as T2) |
| "serum creatinine" in outcome definitions | T3 | YES (C 85%, ×3) |

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | LOW | Key methodology stats (346 RCTs, risk of bias domains) missed |
| Tier accuracy | 0% | All 14 spans should be T3 or NOISE |
| Clinical safety risk | NONE | Methods section — literature search and appraisal methodology |
| Channel diversity | MODERATE | D + C + F (3 channels) |
| Noise level | HIGH | serum creatinine ×3, bare temporal words |
| Pipeline bugs | 0 | No HTML artifact on this page |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Methods section covering literature search, critical appraisal, and evidence synthesis methodology.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 14
- **ADDED**: 0
- **PENDING**: 0
- **CONFIRMED**: 0
- **REJECTED**: 14 (all C/D/F noise — bare lab names ×3, temporal words, table cells, section headings)

### Agent Spans Kept: 0
All 14 original spans were correctly rejected — C channel bare lab names (serum creatinine ×3, creatinine, hemoglobin), temporal words (monthly, weekly), D channel table cells, F channel section headings, C+F abbreviation legend fragment.

### Gaps Added (12) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G102-A | In patients with diabetes and CKD, what are the most effective health service delivery programs/models of care... Intervention: Health service delivery programs/models of care. Comparator: Standard care. Outcomes: Critical and important outcomes listed in Table 1. | PICOM Q17 — Healthcare delivery Ch5 |
| G102-B | Li et al. Education programmes for people with diabetic kidney disease. Cochrane Database Syst Rev. 2011;CD007374. | Cochrane citation — Education Ch5 |
| G102-C | ACEi, angiotensin-converting enzyme inhibitor; ARB, angiotensin II receptor blocker; CKD, chronic kidney disease; DPP-4, dipeptidyl peptidase-4; eGFR, estimated glomerular filtration rate; GLP-1 RA, glucagon-like peptide-1 receptor agonist; HbA1c, glycated hemoglobin; MACE, major adverse cardiovascular events; MRA, mineralocorticoid receptor antagonist; PICOM, Population, Intervention, Comparator, Outcome, Methods; RAS, renin-angiotensin system; RCT, randomized controlled trial; SCr, serum creatinine; SGLT2i, sodium-glucose cotransporter-2 inhibitor; SMBG, self-monitoring of blood glucose; SoF, Summary of Findings; T1D, type 1 diabetes; T2D, type 2 diabetes. | Table 2 abbreviation legend — critical for KB-7 L4 mapping |
| G102-D | Supplementary Tables S24, S25, S92, S93 | SoF refs — Education Ch5 |
| G102-E | Supplementary Tables S26-S28 and S94 | SoF refs — Healthcare delivery Ch5 |
| G102-F | Literature search: The Cochrane Kidney and Transplant Register of Studies was searched... populated by monthly searches of the Cochrane Central Register of Controlled Trials, weekly searches of MEDLINE OVID SP, and yearly searches of Embase OVID SP. | Search methodology — databases and frequency |
| G102-G | The searches from the 2020 guideline retrieved 5,667 citations. For the 2022 guideline update, an additional 1,078 citations were screened... a total of 102 RCTs reported in 189 manuscripts were included. Overall, 346 RCTs reported in 609 manuscripts, 31 observational studies, and 50 systematic reviews were included for review. | Search yield — quantitative results |
| G102-H | Data extraction was performed independently by an ERT member using a standardized data extraction form. Data entry was confirmed by a second ERT member. Authors of studies were contacted when information was unclear or missing. | Data extraction methodology |
| G102-I | All critical appraisal was conducted independently by 2 members of the ERT using the Cochrane Risk of Bias tool, assessing the following domains: sequence generation, allocation concealment (selection bias), blinding of participants and personnel (performance bias), blinding of outcome assessment (detection bias), incomplete outcome data (attrition bias), selective outcome reporting (reporting bias), other bias, and source of funding/sponsor involvement. | Critical appraisal — Cochrane Risk of Bias tool 7 domains |
| G102-J | Any discrepancies were resolved by consensus with assistance from an additional ERT member. Screening of citations and full-text articles was performed in duplicate using Covidence. | Screening methodology and conflict resolution |
| G102-K | Measures of treatment effect. For dichotomous outcomes, results were expressed as risk ratios (RR) with 95% confidence intervals (CI). For time-to-event outcomes, hazard ratios (HR) with 95% CI were used. | Measures of treatment effect — RR and HR definitions |
| G102-L | Table 2 \| (Continued) Clinical questions and systematic review topics in the PICOM format. Guideline chapter 5: Approaches to comprehensive management. None relevant (Cochrane systematic reviews for Q17). | Table 2 final header + Q17 Cochrane status |

### Post-Review State
- **Total spans**: 26
- **ADDED**: 12 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 14 (all original noise)
- **P2-ready facts**: 12

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G102-A | KB-4 (Safety) | Healthcare delivery PICOM — models of care |
| G102-B | KB-7 (Terminology) | Cochrane systematic review citation |
| G102-C | KB-7 (Terminology) | Full abbreviation legend — critical for L4 RxNorm/LOINC/SNOMED mapping |
| G102-D, G102-E | KB-7 (Terminology) | SoF table references |
| G102-F, G102-G | KB-7 (Terminology) | Search methodology and quantitative yield |
| G102-H | KB-7 (Terminology) | Data extraction process |
| G102-I, G102-J | KB-7 (Terminology) | Critical appraisal methodology — Cochrane Risk of Bias domains |
| G102-K | KB-7 (Terminology) | Statistical measures — RR and HR definitions |
| G102-L | KB-7 (Terminology) | Table 2 final header |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 102 of 126
