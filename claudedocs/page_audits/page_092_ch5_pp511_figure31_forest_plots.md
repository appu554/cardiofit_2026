# Page 92 Audit — Chapter 5: PP 5.1.1, Figure 31 Forest Plots (Meta-Analysis)

## Page Identity
- **PDF page**: S91 (Chapter 5 — www.kidney-international.org)
- **Content**: Continuation of Rec 5.1.1 — implementation considerations, rationale, Practice Point 5.1.1, Figure 31 (6-panel forest plot meta-analysis: SBP, DBP, eGFR, HbA1c, SM activity, HRQOL)
- **Clinical tier**: T3 (Informational — meta-analysis results, practice point, implementation guidance)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 4 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 3 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/4) |
| Disagreements | 0 |
| Review Status | FINAL: 10 ADDED, 0 CONFIRMED, 13 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 13 rejected: 9 D forest plot CIs, 1 F HTML artifact, 1 C label, 2 F prose), 10 gaps added |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — count corrected 13→4, D channel removed (raw data has C, F only) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | 9 | 92% | Confidence interval ranges from forest plots |
| F (NuExtract LLM) | 3 | 85-90% | HTML artifact + cost prose + rationale prose |
| C (Grammar/Regex) | 1 | 98% | Practice point label |

## T1 Spans (1) — MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Practice Point 5.1.1" | C | 98% | T3 | Practice point label only — not the practice point content |

**Pattern**: Same as page 90 where "Recommendation 5.1.1" was extracted as T1. C channel's regex captures section headers/labels and classifies them as T1, but the actual clinical content in the body text is not extracted.

## T2 Spans (12) — ALL MISTIERED OR NOISE

### D Channel — Forest Plot CI Values (9 spans)

| # | Text | Channel | Conf | Source | Correct Tier |
|---|------|---------|------|--------|-------------|
| 1 | "-0.59, 0.30" | D | 92% | Figure 31f: HRQOL, Patient education CI | T3 |
| 2 | "-13.01, -1.99" | D | 92% | Figure 31b: DBP, Provider education CI | T3 |
| 3 | "-7.81, -0.70" | D | 92% | Figure 31a: SBP, All interventions CI | T3 |
| 4 | "-6.68, 0.68" | D | 92% | Figure 31b: DBP, Provider reminders CI | T3 |
| 5 | "-6.19, 0.78" | D | 92% | Figure 31b: DBP, All interventions CI | T3 |
| 6 | "-6.08, 0.88" | D | 92% | Figure 31c: eGFR, Provider reminders CI | T3 |
| 7 | "-0.65, 7.65" | D | 92% | Figure 31c: eGFR, Provider education CI | T3 |
| 8 | "-0.25, 0.85" | D | 92% | Figure 31d: HbA1c, Provider reminders CI | T3 |
| 9 | "-0.75, 0.19" | D | 92% | Figure 31d: HbA1c, Provider education CI | T3 |

**Analysis**: D (Table Decomp) channel interpreted the forest plot figure data as table content and extracted 9 individual confidence interval ranges. These are meta-analysis effect sizes from a systematic review — purely statistical results (T3). They are not clinical thresholds, dosing limits, or safety parameters (T1/T2).

**Notable**: D channel extracted only the CI ranges without their labels (which intervention type, which outcome). Without context, "-0.59, 0.30" is meaningless. The channel successfully parsed the numerical structure of forest plots but failed to capture the semantic meaning.

### F Channel (3 spans)

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 10 | "<!-- PAGE 92 -->" | F | 90% | NOISE/BUG | **HTML comment artifact** — 2nd consecutive occurrence |
| 11 | "cost-savings, cost-effectiveness, reduced cost, or positive in-vestment returns. 424" | F | 90% | T3 | Cost-effectiveness conclusion from previous page continuation |
| 12 | "Diabetes self-management education programs should be individualized and tailored to the changing biomedical and psychos..." | F | 85% | T3 | Rationale prose about program individualization |

### Pipeline Bug Confirmation
`<!-- PAGE 92 -->` is the **second consecutive** F channel HTML artifact (after `<!-- PAGE 91 -->` on page 91). This confirms a systematic bug: the NuExtract LLM receives HTML-processed documents with page marker comments, and the model treats these as extractable clinical content at 90% confidence.

## PDF Source Content Analysis

### Content Present on Page
1. **Implementation considerations** (continuation from page 91):
   - Need for trained workforce
   - Limited evidence for CKD-specific programs
   - ADA and NICE definitions of self-management education
   - NICE recommendations for program delivery (multidisciplinary team, one-on-one/groups, telephone/web platforms)
   - NICE: offer at diagnosis with ongoing maintenance sessions

2. **Rationale**: Work Group judgment on individualized, tailored DSMES programs

3. **Practice Point 5.1.1**: "Healthcare systems should consider implementing a structured self-management program for patients with diabetes and CKD, taking into consideration local context, cultures, and availability of resources. Diabetes self-management education programs should be individualized and tailored to the changing biomedical and psychosocial needs of the person with diabetes."

4. **Figure 31**: 6-panel forest plot meta-analysis (Zimbudzi et al. 2018) showing:
   - (a) SBP: Overall ES -4.02 (-6.39, -1.65), I²=0.0% — **favors intervention**
   - (b) DBP: Overall ES -2.94 (-4.88, -0.99), I²=13.1% — **favors intervention**
   - (c) eGFR: Overall ES -0.40 (-3.15, 2.35), I²=50.7% — **no significant effect**
   - (d) HbA1c: Overall ES -0.37 (-0.85, 0.10), I²=86.9% — **trend, not significant**
   - (e) SM activity: Overall ES 0.54 (0.37, 0.70), I²=0.0% — **favors intervention**
   - (f) HRQOL: Overall ES -0.06 (-0.27, 0.15), I²=0.0% — **no significant effect**

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| PP 5.1.1 full text (individualized self-management programs...) | T3 | NO — only label extracted |
| Figure 31 overall effect sizes with significance | T3 | Partially — 9 CI ranges extracted without labels |
| SBP/DBP improvements favor intervention | T3 | NO (meta-analysis conclusion) |
| I² heterogeneity values (0%, 13.1%, 50.7%, 86.9%) | T3 | NO |
| ADA/NICE program definitions | T3 | NO |
| "multidisciplinary team including trained/accredited practitioner" | T3 | NO |

## Cross-Page Patterns

### D Channel Forest Plot Parsing
This is the first time D (Table Decomp) has encountered a forest plot figure. It extracted CI ranges as if they were table cells — a reasonable structural interpretation, but semantically wrong. Forest plots are statistical visualizations, not clinical parameter tables. The D channel needs a figure-vs-table discriminator.

### F Channel HTML Artifact Pattern
Two consecutive pages (91, 92) with `<!-- PAGE XX -->` artifacts. Pattern: F channel receives HTML-preprocessed content with page markers intact. Bug priority: MEDIUM — doesn't affect clinical accuracy but inflates span counts and creates noise.

### C Channel Header-Only Pattern
Three consecutive pages (90, 91, 92) where C channel extracts recommendation/practice point labels as T1 but misses the body text. This is a consistent C channel regex limitation — it matches "Recommendation X.Y.Z" or "Practice Point X.Y.Z" patterns but doesn't capture the following sentence(s).

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | LOW | PP 5.1.1 body missed, Figure 31 partially captured (CIs only) |
| Tier accuracy | 0% | All 13 spans mistiered (should all be T3 or NOISE) |
| Clinical safety risk | NONE | Meta-analysis results and practice point guidance |
| Channel diversity | MODERATE | D + F + C all present |
| Noise level | HIGH | 10/13 spans are noise (9 decontextualized CIs + 1 HTML artifact) |
| Pipeline bugs | 1 | HTML comment artifact (F channel, 2nd occurrence) |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All content is meta-analysis statistics, implementation guidance, and a practice point about program design. The D channel CI extractions are technically correct number parsing but clinically meaningless without labels.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 13
- **ADDED**: 0
- **PENDING**: 13 (all original agent spans)
- **CONFIRMED**: 0
- **REJECTED**: 0

### Agent Spans Kept: 0
All 13 original agent spans rejected:
- 9 D channel forest plot CI ranges (decontextualized numbers: "-0.75, 0.19", "-13.01, -1.99", etc.)
- `ec3bca17` — F channel `<!-- PAGE 92 -->` HTML artifact
- `7d6f0563` — F channel "cost-savings, cost-effectiveness..." (partial prose)
- `0666ef6b` — C channel "Practice Point 5.1.1" (label only)
- `4e9bc9d5` — F channel "Diabetes self-management education programs should be individualized..." (partial prose)

### Gaps Added (10) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G92-A | Healthcare systems should consider implementing a structured self-management program for patients with diabetes and CKD, taking into consideration local context, cultures, and availability of resources. | PP 5.1.1 sentence 1 |
| G92-B | Diabetes self-management education programs should be individualized and tailored to the changing biomedical and psychosocial needs of the person with diabetes. | PP 5.1.1 sentence 2 |
| G92-C | There is very little evidence on specific self-management programs for people with different severities of CKD and in people receiving dialysis or with a kidney transplant. | Implementation consideration |
| G92-D | Healthcare organizations need to have a trained workforce to deliver self-management programs for people with diabetes and CKD. | Implementation — workforce |
| G92-E | NICE recommends that a multidisciplinary team that includes at least 1 trained or accredited healthcare practitioner, such as a dietitian, pharmacist, diabetes specialist nurse... | NICE team recommendation |
| G92-F | NICE recommends that self-management education be offered to people with diabetes at diagnosis, with ongoing maintenance sessions. | NICE timing recommendation |
| G92-G | Overall SBP effect: -4.02 (95% CI -6.39, -1.65), I-squared = 0.0% | Figure 31a — SBP favors intervention |
| G92-H | Overall DBP effect: -2.94 (95% CI -4.88, -0.99), I-squared = 13.1% | Figure 31b — DBP favors intervention |
| G92-I | Overall HbA1c effect: -0.37 (95% CI -0.85, 0.10), I-squared = 86.9% | Figure 31d — HbA1c trend, not significant |
| G92-J | Overall SM activity effect: 0.54 (95% CI 0.37, 0.70), I-squared = 0.0% | Figure 31e — SM activity favors intervention |

### Post-Review State
- **Total spans**: 23
- **ADDED**: 10 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 13 (all original noise)
- **P2-ready facts**: 10

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G92-A, G92-B | KB-4 (Safety) | PP 5.1.1 full text — practice point for self-management |
| G92-C–F | KB-4 (Safety) | Implementation considerations — workforce, NICE recommendations |
| G92-G–J | KB-16 (Monitoring) | Figure 31 meta-analysis effect sizes — SBP, DBP, HbA1c, SM activity |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 92 of 126
