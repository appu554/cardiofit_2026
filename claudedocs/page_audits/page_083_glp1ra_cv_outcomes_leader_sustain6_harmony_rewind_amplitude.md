# Page 83 Audit — GLP-1 RA Cardiovascular Outcomes Evidence (LEADER, SUSTAIN-6, HARMONY, REWIND, AMPLITUDE-O, ELIXA, EXSCEL)

| Field | Value |
|-------|-------|
| **Page** | 83 (PDF page S82) |
| **Content Type** | Rec 4.2.1 balance of benefits/harms: GLP-1 RA cardiovascular outcome trials — LEADER (liraglutide: 13% MACE reduction, HR 0.87; CKD subgroup HR 0.69; stroke HR 0.51 at eGFR <60), SUSTAIN-6 (semaglutide: 26% MACE reduction, HR 0.74; no heterogeneity by CKD), HARMONY (albiglutide: 22% MACE reduction, HR 0.78; excluded eGFR <30; no longer on market), REWIND (dulaglutide: 12% MACE reduction, HR 0.88; excluded eGFR <15; primary prevention), AMPLITUDE-O (efpeglenatide: 27% MACE reduction, HR 0.73; eGFR <71 subgroup HR 0.67), ELIXA (lixisenatide: no CV benefit, safety confirmed), EXSCEL (exenatide: no CV benefit, safety confirmed) |
| **Extracted Spans** | 6 total (6 T1, 0 T2) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 6 |
| **Risk** | Clean |
| **Cross-Check** | Count corrected 7→6 (T2 1→0); D channel removed (not present in raw data); verified against raw extraction data |
| **Audit Date** | 2026-02-25 (revised) |

---

## Source PDF Content

**GLP-1 RA Cardiovascular Outcome Trials Summary:**

| Trial | Drug | Population | Primary Outcome | MACE Result | CKD Subgroup |
|-------|------|-----------|-----------------|-------------|--------------|
| **LEADER** | Liraglutide | 9340 T2D, HbA1c ≥7%, high CV risk (incl CVD, CKD G3+, age ≥60). Included 220 with eGFR 15–30 | MACE (CV death, MI, stroke) over 3.8 yr | **HR 0.87 (95% CI: 0.78–0.97)** — 13% reduction | **eGFR <60: HR 0.69 (0.57–0.85)** vs eGFR ≥60: HR 0.94 (0.83–1.07), P-interaction=0.01. Stroke: **HR 0.51 (0.33–0.80)** at eGFR <60 |
| **SUSTAIN-6** | Semaglutide (injectable) | 3297 T2D, HbA1c ≥7%, CVD/CKD G3+. 83% had CVD/CKD, 10.7% CKD only | MACE | **HR 0.74 (0.58–0.95)** — 26% reduction | No heterogeneity: eGFR <30 vs ≥30 (P=0.98); eGFR <60 vs ≥60 (P=0.37) |
| **HARMONY** | Albiglutide | 9463 T2D, HbA1c ≥7%, high CV risk. **eGFR <30 excluded** | MACE over 1.6 yr | **HR 0.78 (0.68–0.90)** — 22% reduction | No heterogeneity by eGFR subgroups (P=0.19). **Albiglutide no longer on market** |
| **REWIND** | Dulaglutide | 9901 T2D, HbA1c ≤9.5% (mean 7.2%). **eGFR <15 excluded**. 31.5% established CVD (primarily primary prevention) | MACE over 5.4 yr | **HR 0.88 (0.79–0.99)** — 12% reduction | Similar with/without previous CVD (P=0.97) |
| **AMPLITUDE-O** | Efpeglenatide | 4076 T2D, high CV risk or CKD. 89.6% established CVD | MACE | **HR 0.73 (0.58–0.92)** — 27% reduction | **eGFR <71: HR 0.67 (0.50–0.91)** |
| **ELIXA** | Lixisenatide | Acute coronary syndrome | MACE | **No benefit** — CV safety confirmed | — |
| **EXSCEL** | Exenatide | — | MACE | **No benefit** — CV safety confirmed | — |

**Key Observations:**
- 5/8 injectable GLP-1 RAs show MACE benefit (liraglutide, semaglutide, albiglutide, dulaglutide, efpeglenatide)
- 3 showed safety but no benefit (lixisenatide, exenatide, oral semaglutide)
- CKD subgroups show equal or greater benefit than general population
- LEADER included patients down to eGFR 15 (most inclusive)
- Differences may stem from molecular structures, half-lives, formulations, study design

---

## Key Spans Assessment

### Tier 1 Spans (6)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "liraglutide" (B) | B | 100% | **→ T3** — Drug name from LEADER trial description |
| "semaglutide" (B) | B | 100% | **→ T3** — Drug name from SUSTAIN-6 trial description |
| "Exenatide" (B) | B | 100% | **→ T3** — Drug name from EXSCEL trial description |
| "dulaglutide" (B) | B | 100% | **→ T3** — Drug name from REWIND trial description |
| "eGFR <60" (C) | C | 95% | **→ T3** — Standalone threshold fragment from CKD subgroup analysis |
| "eGFR <15 ml/min" (C) | C | 95% | **→ T3** — Standalone threshold from REWIND exclusion criterion |

**Summary: 0/6 T1 genuine patient safety content. 4 are drug names → T3. 2 are eGFR threshold fragments → T3.**

### Tier 2 Spans (1)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "HR: 0.87; 95% CI: 0.78-0.97" (D) | D | 92% | **✅ T2 CORRECT** — LEADER trial primary MACE outcome hazard ratio. The D channel extracted this from Figure 28 or inline text. Genuine clinical evidence metric |

**Summary: 1/1 T2 correctly tiered. The D channel captured the LEADER HR — the only genuinely useful extraction on this page.**

---

## Critical Findings

### ✅ D Channel Extracts LEADER HR — Single Useful Data Point

The D channel captures "HR: 0.87; 95% CI: 0.78-0.97" — the LEADER trial primary outcome. This suggests D can extract statistical results formatted as table cells or structured data, even from dense evidence prose. However, only 1 of the 7 trial HRs on this page was captured.

### ⚠️ B Channel Captures 4/7 GLP-1 RA Drug Names

The B channel fires on liraglutide, semaglutide, exenatide, and dulaglutide but MISSES:
- **albiglutide** (HARMONY) — possibly not in the drug dictionary
- **efpeglenatide** (AMPLITUDE-O) — likely not in the drug dictionary
- **lixisenatide** (ELIXA) — possibly not in the drug dictionary

This reveals a **drug dictionary coverage gap**: the B channel's dictionary doesn't include all GLP-1 RA agents. Albiglutide, efpeglenatide, and lixisenatide are less commonly prescribed (albiglutide is withdrawn), which explains their absence, but they're clinically important for evidence interpretation.

### ❌ No F Channel on This Page

The F (NuExtract LLM) channel did not fire on page 83 despite extensive evidence prose. This page contains 7 complete trial summaries with specific HR data, CKD subgroup analyses, and clinical conclusions — exactly the type of evidence content F extracted well on pages 78-80. Its absence here is unexplained.

### ❌ 6/7 Trial HR Values NOT EXTRACTED

| Missing HR | Trial | Value | Clinical Importance |
|-----------|-------|-------|---------------------|
| SUSTAIN-6 MACE | Semaglutide | HR 0.74 (0.58–0.95) | 26% MACE reduction |
| HARMONY MACE | Albiglutide | HR 0.78 (0.68–0.90) | 22% MACE reduction |
| REWIND MACE | Dulaglutide | HR 0.88 (0.79–0.99) | 12% MACE reduction |
| AMPLITUDE-O MACE | Efpeglenatide | HR 0.73 (0.58–0.92) | 27% MACE reduction |
| LEADER CKD subgroup | Liraglutide | HR 0.69 (0.57–0.85) at eGFR <60 | Greater benefit in CKD |
| LEADER stroke CKD | Liraglutide | HR 0.51 (0.33–0.80) at eGFR <60 | 49% stroke reduction in CKD |

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| 5/8 GLP-1 RAs show MACE benefit (liraglutide, semaglutide, albiglutide, dulaglutide, efpeglenatide) | **T1** | Drug class efficacy summary — guides prescribing |
| LEADER: 220 patients with eGFR 15–30 included (most inclusive trial for advanced CKD) | **T1** | Evidence basis for GLP-1 RA use at very low eGFR |
| CKD subgroups show equal or greater CV benefit | **T1** | Supports GLP-1 RA use specifically in CKD population |
| Albiglutide no longer available on market | **T1** | Drug availability — prescribing constraint |
| ELIXA/EXSCEL: CV safety confirmed but no benefit | **T2** | Not all GLP-1 RAs are equal for CV outcomes |
| SUSTAIN-6: 26% MACE reduction HR 0.74 | **T2** | Key evidence metric |
| REWIND: primary prevention trial (31.5% established CVD) | **T2** | Trial design context — broader applicability |
| AMPLITUDE-O: CKD subgroup HR 0.67 at eGFR <71 | **T2** | CKD-specific benefit evidence |
| Differences may stem from molecular structures, half-lives, formulations | **T3** | Explanatory context for variable results |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — 7 spans with only 1 useful (D channel LEADER HR); all 6 T1 are drug names/thresholds (T3); but page is pure evidence prose (T2 content), so low T1 expectation is appropriate; main concern is that 6/7 trial HRs and the CKD subgroup benefits are missing |
| **Tier corrections** | All 4 drug names: T1 → T3; both eGFR thresholds: T1 → T3 |
| **Missing T1** | Drug class efficacy summary, CKD subgroup benefit statement, albiglutide market withdrawal |
| **Missing T2** | 6/7 trial HRs, trial design details, CKD-specific subgroup analyses |

---

## Completeness Score (Pre-Review)

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~10% — 1 HR value from 7 complete trial summaries; drug names but no clinical sentences; CKD subgroup insights entirely missing |
| **Tier accuracy** | ~14% (0/6 T1 correct + 1/1 T2 correct = 1/7) |
| **Noise ratio** | ~86% — 6/7 spans are drug names or threshold fragments |
| **Genuine T1 content** | 0 extracted (page is evidence prose, T1 not primary content type) |
| **Prior review** | 0/7 reviewed |
| **Overall quality** | **MODERATE-POOR** — D channel captures 1 useful HR; page is evidence-heavy (appropriate for T2) but 6 trial summaries with CKD subgroup data entirely unextracted |

---

## Raw PDF Cross-Check (2026-02-28)

Cross-checked 25 ADDED spans against exact KDIGO PDF text (pages S82-S83). Found **13 duplicates** from parallel agent overlap and **6 missing gaps**.

### Duplicates Rejected (13)

Parallel agents added near-identical spans for each trial. For each, kept the more detailed version:
- LEADER main (kept `1c209710`), CKD subgroup (kept `758bb4b7`), stroke (kept `54a7392f`)
- SUSTAIN-6 (kept `30f40e94`), HARMONY (kept `26e65e9f`), REWIND (kept `8e5bce21`)
- AMPLITUDE-O (kept `11ea6572`), ELIXA/EXSCEL (kept `3ff36f79`)
- 5/8 MACE benefit (kept `1a3b6f62`), LEADER 220 eGFR 15-30 (kept `9fceca04`)
- CKD subgroups equal benefit (kept `0d8546e7`)

### Gaps Added (6)

| Gap | Priority | Content | KB Target |
|-----|----------|---------|-----------|
| G83-A | HIGH | SUSTAIN-6 CKD subgroup: no heterogeneity at eGFR <30 (P-interaction=0.98) and eGFR <60 (P=0.37) — exact text | KB-4 |
| G83-B | HIGH | LEADER subgroup caveat: "efficacy among individuals with CKD is at least as great as that for those without CKD" | KB-4 |
| G83-C | HIGH | REWIND exact: median 5.4 years, MACE definition, HR 0.88, CVD interaction P=0.97 | KB-1/KB-4 |
| G83-D | MEDIUM | HARMONY eGFR subgroups: no heterogeneity across <60, 60-90, ≥90 (P=0.19) | KB-4 |
| G83-E | MEDIUM | Non-benefit agents complete: lixisenatide, exenatide, AND oral semaglutide | KB-1/KB-4 |
| G83-F | MEDIUM | REWIND primary prevention context: 31.5% established CVD, significant CKD inclusion | KB-1 |

All 13 duplicates rejected via API (all 200). All 6 gaps added via API (all 201).

---

## Post-Review State (Final — with raw PDF cross-check)

| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 19 | 6 original noise + 13 duplicate ADDED spans |
| **ADDED** | 18 | 25 agent-added − 13 duplicates + 6 cross-check gaps |
| **PENDING** | 1 | D channel HR 0.87 (LEADER) — correctly tiered T2 |
| **Total spans** | 38 | 32 + 6 new |
| **P2-ready** | 18 | All ADDED |
| **Reviewer** | claude-auditor | |
| **Review Date** | 2026-02-28 | |

### Updated Completeness Score (Final)

| Metric | Pre-Review | Final (2/28) |
|--------|-----------|--------------|
| **Total spans** | 7 | 38 |
| **P2-ready** | 0 | 18 + 1 PENDING |
| **Extraction completeness** | ~10% | **~95%** — all 7 trial summaries with HRs/CIs, all CKD subgroup P-interactions, subgroup caveat, non-benefit agents, primary prevention context |
| **Noise ratio** | 86% | 0% active (19 rejected) |
| **Overall quality** | MODERATE-POOR | **EXCELLENT** — comprehensive CV outcome evidence for all GLP-1 RA trials with CKD-specific safety data |

---

## Drug Dictionary Coverage Gap (Audit-Wide)

Page 83 reveals that the B channel drug dictionary is incomplete for GLP-1 RA agents:

| Drug | In Dictionary? | Evidence |
|------|---------------|----------|
| liraglutide | ✅ Yes | Extracted on p83 |
| semaglutide | ✅ Yes | Extracted on p83 |
| dulaglutide | ✅ Yes | Extracted on p83 |
| exenatide | ✅ Yes | Extracted on p83 |
| albiglutide | ❌ No | Not extracted (HARMONY trial) |
| efpeglenatide | ❌ No | Not extracted (AMPLITUDE-O trial) |
| lixisenatide | ❌ No | Not extracted (ELIXA trial) |

Previously identified missing drugs:
- **glipizide**: Extracted on p79 ✅
- **phenformin**: Not extracted on p80 ❌ (withdrawn drug)
- **sotagliflozin**: Corrupted OCR on p77 ("Slaglifcin") — unclear if in dictionary
