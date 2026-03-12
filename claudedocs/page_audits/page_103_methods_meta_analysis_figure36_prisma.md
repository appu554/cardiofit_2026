# Page 103 Audit — Methods: Meta-Analysis, Heterogeneity, Figure 36 PRISMA (3 Spans)

## Page Identity
- **PDF page**: S102 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Data synthesis (Mantel-Haenszel random-effects model), heterogeneity assessment (I², chi-squared), publication bias (funnel plots), subgroup analysis, Figure 36 (PRISMA search yield: 346 RCTs + 31 observational + 50 reviews)
- **Clinical tier**: T3 (Informational — statistical methodology)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 3 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 2 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM), B (Drug Dictionary) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/3) |
| Disagreements | 0 |
| Review Status | FINAL: 9 ADDED, 0 CONFIRMED, 3 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 3 rejected: F HTML artifact, F prose, B drug class), 9 gaps added |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "GLP-1 RA" | B | 100% | T1 | T3 | Drug class from subgroup analysis text ("short-acting versus long-acting GLP-1 RA") |
| 2 | `<!-- PAGE 103 -->` | F | 90% | T2 | NOISE/BUG | **HTML artifact — 9th occurrence** |
| 3 | "Data were pooled using the Mantel-Haenszel random-effects model for dichotomous outcomes and the inverse variance random..." | F | 85% | T2 | T3 | Statistical methodology prose |

### Analysis
- **B channel "GLP-1 RA"**: Extracted from subgroup analysis description mentioning "short-acting versus long-acting GLP-1 RA" as an example subgroup — purely methodological context
- **F channel HTML artifact**: 9th confirmed occurrence (pages 91, 92, 94, 95, 98, 99, 100, 101, 103)
- **F channel prose**: Good extraction of meaningful methodology content, but mistiered

## PDF Source Content Analysis

### Content Present on Page
1. **Data synthesis**: Mantel-Haenszel random-effects for dichotomous, inverse variance for continuous, generic inverse variance for time-to-event
2. **Heterogeneity assessment**: Chi-squared test (P<0.05), I² statistic, Higgins et al. 2003 conventions
3. **Publication bias**: Funnel plots when >10 studies, unpublished study searches
4. **Subgroup analysis**: By diabetes type, CKD severity, dialysis modality, age group, intervention type
5. **Figure 36 — PRISMA flow diagram**:
   - Sources: 802 Cochrane Registry + 44 hand-searching + 6 supplement
   - 226 duplicates removed → 626 screened → 346 excluded → 280 assessed
   - Exclusions: wrong population (65), wrong intervention (69), wrong duration (2), wrong design (19), ongoing (23)
   - **102 new RCTs included** in 2022 update
   - **Total: 346 RCTs, 31 observational, 50 systematic reviews**
   - By chapter: Ch1 151, Ch2 16, Ch3 94, Ch4 83, Ch5 11 RCTs

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | VERY LOW | 3 spans for a dense methodology page + PRISMA diagram |
| Tier accuracy | 0% | All should be T3 or NOISE |
| Clinical safety risk | NONE | Statistical methodology and study flow |
| Pipeline bugs | 1 | HTML artifact (9th) |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Statistical methodology and PRISMA flow diagram.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 3
- **ADDED**: 0
- **PENDING**: 0
- **CONFIRMED**: 0
- **REJECTED**: 3 (all original agent spans already rejected)

### Agent Spans Kept: 0
All 3 original spans correctly rejected:
- `8701da6f` — F channel `<!-- PAGE 103 -->` HTML artifact
- `42cc4041` — F channel "Data were pooled using the Mantel-Haenszel..." (partial prose, replaced by complete gap)
- `e89b123b` — B channel "GLP-1 RA" (bare drug class from subgroup analysis text)

### Gaps Added (9) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G103-A | Previous studies: Studies included in previous version of the guideline (n = 244 RCTs, 31 observational studies, 50 reviews). Identification of new studies via databases and registers: Records identified from Cochrane Kidney and Transplant Registry of studies (n = 802), Hand-searching (n = 44), Supplement searching (n = 6). Records removed before screening: Duplicate records removed (n = 226). | Figure 36 — Identification phase |
| G103-B | Records screened (n = 626). Records excluded (n = 346). Reports sought for retrieval (n = 280). Reports not retrieved (n = 0). Reports assessed for eligibility (n = 280). Reports excluded: Wrong population (n = 65), Wrong intervention (n = 69), Wrong duration of therapy (n = 2), Wrong study design (n = 19), Ongoing studies (n = 23). New studies included in review (n = 102). | Figure 36 — Screening and eligibility |
| G103-C | Total studies included in review (n = 346 RCTs, 31 observational studies, 50 reviews). Chapter 1 Comprehensive care (n = 151 RCTs). Chapter 2 Glycemic targets (n = 16 RCTs). Chapter 3 Lifestyle (n = 94 RCTs). Chapter 4 Antihyperglycemic therapies (n = 83 RCTs). Chapter 5 Models of care (n = 11 RCTs). | Figure 36 — Total evidence base by chapter |
| G103-D | Figure 36 \| Search yield and study flow diagram. A number of randomized controlled trials (RCTs) overlap across chapters in the guidelines. Screening for RCTs only. | Figure 36 caption with footnotes |
| G103-E | the effects of treatment, such as HbA1c, etc., the mean difference (MD) with 95% CI was used. | Measures of treatment effect — MD for continuous (continued from p102) |
| G103-F | Data synthesis. Data were pooled using the Mantel–Haenszel random-effects model for dichotomous outcomes and the inverse variance random-effects model for continuous outcomes. The random-effects model was chosen because it provides a conservative estimate of effect in the presence of known and unknown heterogeneity. The generic inverse variance random-effects analysis was used for time-to-event data. | Data synthesis — statistical pooling methods |
| G103-G | Assessment of heterogeneity. Heterogeneity was assessed by visual inspection of forest plots of standardized mean effect sizes and of risk ratios, and chi-squared tests. A P < 0.05 was used to denote statistical heterogeneity, with an I-squared calculated to measure the proportion of total variation in the estimates of treatment effect that was due to heterogeneity beyond chance. | Assessment of heterogeneity — I-squared criteria |
| G103-H | Assessment of publication bias. We made every attempt to minimize publication bias by including unpublished studies (e.g., by searching online trial registries and conference abstracts). To assess publication bias, we used funnel plots of the log odds ratio (effect vs. standard error of the effect size) when a sufficient number of studies were available (i.e., more than 10 studies). Other reasons for the asymmetry of funnel plots were considered. | Assessment of publication bias — funnel plots |
| G103-I | Subgroup analysis and investigation of heterogeneity. Subgroup analysis was undertaken to explore whether clinical differences between the studies may have systematically influenced the differences that were observed in the critical and important outcomes. However, subgroup analyses are hypothesis-forming, rather than hypothesis-testing, and should be interpreted with caution. The following subgroups were considered: type of diabetes, severity of CKD, dialysis modality, age group (pediatric or older adults), and type of intervention—for example, short-acting versus long-acting GLP-1 RA. | Subgroup analysis — 5 subgroup categories |

### Post-Review State
- **Total spans**: 12
- **ADDED**: 9 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 3 (all original noise)
- **P2-ready facts**: 9

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G103-A–D | KB-7 (Terminology) | Figure 36 PRISMA flow — search yield, exclusion criteria, chapter breakdown |
| G103-E–H | KB-7 (Terminology) | Statistical methodology — treatment effect measures, data synthesis, heterogeneity, publication bias |
| G103-I | KB-7 (Terminology) + KB-1 (Dosing) | Subgroup analysis categories — includes intervention type (GLP-1 RA subtypes) |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 103 of 126
