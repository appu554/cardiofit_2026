# Page 85 Audit — GLP-1 RA Kidney Outcomes Evidence (PIONEER 6, 2021 Meta-Analysis, LEADER Kidney, SUSTAIN-6 Kidney, REWIND Kidney, AWARD-7)

| Field | Value |
|-------|-------|
| **Page** | 85 (PDF page S84) |
| **Content Type** | GLP-1 RA evidence continued: PIONEER 6 (oral semaglutide: non-inferior MACE, eGFR <60 subgroup HR 0.74; eGFR <30 excluded), 2021 meta-analysis of 8 GLP-1 RA trials (60,080 participants: CV death HR 0.87, stroke HR 0.83, MI HR 0.90, all-cause mortality HR 0.88, HF hospitalization HR 0.90 — first HF benefit for GLP-1 RA class), LEADER kidney outcomes (22% composite kidney reduction HR 0.78, new-onset albuminuria HR 0.74), SUSTAIN-6 kidney (new/worsening nephropathy HR 0.64), REWIND kidney (15% composite reduction HR 0.85, new albuminuria HR 0.77, 40% eGFR decline reduced 30%, 50% decline reduced 46%), AWARD-7 (dulaglutide vs insulin glargine in CKD G3a-G4: eGFR decline -0.7 vs -3.3, macroalbuminuria subgroup -0.5/-0.7 vs -5.5, 40% eGFR decline/KF reduced >50%, macroalbuminuria HR 0.25, FDA approval for eGFR ≥15), meta-analysis kidney composite (new albuminuria, eGFR decline, KRT) |
| **Extracted Spans** | 2 total (1 T1, 1 T2) |
| **Channels** | C (Grammar/Regex) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 2 |
| **Risk** | Clean |
| **Cross-Check** | Counts verified against raw extraction data |
| **Audit Date** | 2026-02-25 (revised) |

---

## Source PDF Content

**PIONEER 6 (Oral Semaglutide):**
- 3183 patients with T2D + high CV risk/CKD/age >50 with CVD risk factor
- **eGFR <30 excluded**
- Oral semaglutide non-inferior to placebo for primary MACE
- eGFR <60 subgroup: **HR 0.74 (95% CI: 0.41–1.33)** — P-interaction 0.80 (no heterogeneity)

**2021 Meta-Analysis (8 GLP-1 RA Trials — 60,080 Participants):**
- Trials: ELIXA, LEADER, SUSTAIN-6, EXSCEL, HARMONY, REWIND, PIONEER 6, AMPLITUDE-O
- **CV death: HR 0.87 (95% CI: 0.80–0.94)**
- **Stroke: HR 0.83 (95% CI: 0.76–0.92)**
- **MI: HR 0.90 (95% CI: 0.83–0.98)**
- **All-cause mortality: HR 0.88 (95% CI: 0.82–0.94)**
- **HF hospitalization: HR 0.90 (95% CI: 0.83–0.98)** — first time HF benefit demonstrated for GLP-1 RA class

**LEADER Kidney Outcomes:**
- Secondary composite: new-onset severely increased albuminuria, doubling SCr, kidney failure, or death from kidney disease
- **Composite kidney: HR 0.78 (95% CI: 0.67–0.92)** — 22% reduction
- Driven by **new-onset severely increased albuminuria: HR 0.74 (95% CI: 0.60–0.91)**
- No difference in SCr doubling or kidney failure

**SUSTAIN-6 Kidney Outcomes:**
- New or worsening nephropathy: **HR 0.64 (95% CI: 0.46–0.88)**
- Composite: persistent severely increased albuminuria, persistent doubling SCr, CrCl <45, or KRT

**REWIND Kidney Outcomes:**
- Secondary microvascular outcome including CKD
- **Composite kidney: HR 0.85 (95% CI: 0.77–0.93)** — 15% reduction
- Definition: new severely increased albuminuria (ACR >33.9 mg/mmol [>339 mg/g]), sustained eGFR decline 30%, or KRT
- **New severely increased albuminuria: HR 0.77 (95% CI: 0.68–0.87)**
- Post hoc: 40% eGFR decline reduced 30%, 50% eGFR decline reduced 46%
- No serious kidney adverse events
- 22.2% had eGFR <60 at baseline, 7.9% severely increased albuminuria
- Kidney benefit similar ≥60 vs <60 (P-interaction 0.65)
- **HbA1c-lowering explained 26% and BP-lowering 15% of kidney benefits** — not all benefit from conventional risk factors
- Kidney benefit similar regardless of ACEi/ARB use

**AWARD-7 (Dulaglutide vs Insulin Glargine in CKD G3a-G4):**
- 577 patients with T2D + CKD G3a-G4 (mean eGFR 38 ml/min per 1.73 m²)
- All on ACEi or ARB
- Primary: glycemic indices; Main secondary: eGFR and ACR
- **eGFR decline: -0.7 ml/min per 1.73 m² (dulaglutide) vs -3.3 ml/min per 1.73 m² (insulin glargine)**
- 0.75 mg weekly and 1.5 mg weekly both beneficial
- **Severely increased albuminuria subgroup: -0.7/-0.5 vs -5.5 ml/min per 1.73 m²**
- Similar HbA1c improvement (~1%), comparable BP
- **Symptomatic hypoglycemia reduced by half** vs insulin glargine
- Higher GI side effects but overall safety confirmed in CKD G3a-G4
- **FDA approval: dulaglutide for glycemic control with eGFR ≥15 ml/min per 1.73 m²**
- Exploratory: **40% eGFR decline or KF reduced >50%**; macroalbuminuria subgroup **HR 0.25 (95% CI: 0.10–0.68)** — 75% risk reduction

**2021 Meta-Analysis Kidney Composite:**
- GLP-1 RA treatment reduces broad composite kidney outcome (new severely increased albuminuria, eGFR decline, KRT)

---

## Key Spans Assessment

### Tier 1 Spans (1)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "eGFR <60" | C | 95% | **→ T3** — Standalone eGFR threshold from PIONEER 6 subgroup or REWIND baseline. No clinical directive context |

**Summary: 0/1 T1 genuine patient safety content. Bare eGFR threshold → T3.**

### Tier 2 Spans (1)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "at baseline" | C | 90% | **→ NOISE** — 2-word fragment from "22.2% had eGFR <60 at baseline". No clinical information |

**Summary: 0/1 T2 correct. Fragment → NOISE.**

---

## Critical Findings

### 🚨 2 SPANS FROM 2,500+ WORDS OF DENSE CLINICAL EVIDENCE — Worst Extraction Ratio in Audit

Page 85 contains approximately 2,500 words of continuous evidence prose — one of the most content-dense pages in the entire KDIGO 2022 guideline. It covers:
- 1 new trial (PIONEER 6)
- 1 meta-analysis (8 trials, 60,080 patients)
- 4 trial kidney outcome analyses (LEADER, SUSTAIN-6, REWIND, AWARD-7)
- 15+ hazard ratios with confidence intervals
- 1 FDA approval decision
- Multiple clinical insights (HF benefit, kidney mechanism, hypoglycemia)

Yet only **2 spans** were extracted — both C channel fragments ("eGFR <60" and "at baseline") that carry zero clinical information. This is the **worst extraction-to-content ratio** in the entire audit.

### ❌ No B Channel Despite 10+ Drug Names

The PDF text contains: semaglutide, liraglutide, dulaglutide, exenatide, insulin glargine, GLP-1 RA (×multiple), ACEi, ARB — yet B channel fired 0 times. This is extremely unusual given that B fires on every drug name mention on other pages (e.g., metformin ×72 on p81).

**Possible explanations:**
1. The PDF text on this page may not be in the extraction pipeline's OCR text layer (rendering issue)
2. The page boundary may be misaligned — content assigned to page 84 instead
3. B channel processing may have failed on this specific page

### ❌ No F Channel Despite Perfect Evidence Prose

Page 85 is exactly the type of content F (NuExtract LLM) excels at — continuous clinical evidence prose with specific claims, hazard ratios, and conclusions. F captured similar content on pages 65, 72, 79, 80. Its absence here, combined with B's absence, strongly suggests a **pipeline processing failure** rather than a content recognition issue.

### ❌ No D Channel Despite Table-Adjacent Content

Some of the content (AWARD-7 eGFR decline comparison, meta-analysis pooled HRs) is structured enough for D channel extraction. Its absence further supports pipeline failure on this page.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| 2021 meta-analysis: CV death HR 0.87, stroke HR 0.83, MI HR 0.90, all-cause mortality HR 0.88, HF hospitalization HR 0.90 | **T1** | Class-level evidence for GLP-1 RA CV benefit (60,080 patients) |
| First demonstration of HF hospitalization benefit for GLP-1 RA class | **T1** | Novel drug class benefit — changes prescribing for HF patients |
| AWARD-7: Dulaglutide approved for eGFR ≥15 ml/min per 1.73 m² | **T1** | FDA regulatory threshold — prescribing eligibility |
| AWARD-7: eGFR decline -0.7 vs -3.3 (dulaglutide vs insulin glargine in CKD G3a-G4) | **T1** | Drug comparison in advanced CKD — kidney protection |
| AWARD-7: Symptomatic hypoglycemia reduced by half vs insulin glargine | **T1** | Safety advantage in CKD population |
| AWARD-7 macroalbuminuria: 40% eGFR decline/KF HR 0.25 (0.10-0.68) — 75% reduction | **T1** | Strongest kidney protection signal |
| LEADER kidney: composite HR 0.78 (0.67-0.92), albuminuria HR 0.74 (0.60-0.91) | **T2** | Trial-specific kidney outcomes |
| SUSTAIN-6 kidney: nephropathy HR 0.64 (0.46-0.88) | **T2** | Trial-specific kidney outcomes |
| REWIND kidney: composite HR 0.85 (0.77-0.93), albuminuria HR 0.77 (0.68-0.87) | **T2** | Trial-specific kidney outcomes |
| REWIND: HbA1c-lowering explains 26%, BP-lowering 15% of kidney benefit | **T2** | Mechanistic insight — not all benefit from conventional risk factors |
| PIONEER 6: oral semaglutide non-inferior for MACE, eGFR <60 HR 0.74 (0.41-1.33) | **T2** | Oral GLP-1 RA evidence |
| REWIND: kidney benefit similar regardless of ACEi/ARB use | **T2** | Drug combination evidence |
| AWARD-7: dulaglutide safety confirmed in CKD G3a-G4 | **T2** | Drug safety in advanced CKD |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — 2 spans from ~2,500 words of evidence prose; 0/2 correctly tiered; suspected pipeline processing failure; page contains critical meta-analysis results, FDA approval, AWARD-7 CKD evidence, and first GLP-1 RA HF benefit |
| **Tier corrections** | "eGFR <60": T1 → T3; "at baseline": T2 → NOISE |
| **Missing T1** | Meta-analysis class-level HRs (5 endpoints), AWARD-7 CKD results, FDA approval for eGFR ≥15, HF hospitalization benefit |
| **Missing T2** | All 4 trial kidney outcome HRs, mechanistic insights, PIONEER 6 data |
| **Pipeline investigation** | Why did B, D, F channels all fail on this page? Content is clearly visible in PDF viewer |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~1% — 2 fragments from ~2,500 words of dense clinical evidence |
| **Tier accuracy** | 0% (0/1 T1 correct + 0/1 T2 correct = 0/2) |
| **Noise ratio** | 100% — Both spans are fragments (eGFR threshold, "at baseline") |
| **Genuine T1 content** | 0 extracted |
| **Prior review** | 0/2 reviewed |
| **Overall quality** | **VERY POOR — FLAG** — Suspected pipeline failure; the richest evidence page in Chapter 4 with zero meaningful extraction; contains the 2021 meta-analysis (60,080 patients, 5 CV endpoints), AWARD-7 (CKD G3a-G4 trial), FDA approval decision, and first GLP-1 RA HF benefit |

---

## Pipeline Failure Investigation Needed

Page 85 has the **lowest span count (2) relative to content density** in the entire audit. Every other evidence-heavy page (pp65, 72, 79, 80, 83) produced at least 5-28 spans. The complete absence of B (drug names), F (evidence extraction), and D (structured data) channels — despite clearly visible PDF text in the viewer — suggests:

1. **OCR/text extraction failure**: The page text may not have been properly OCR'd or included in the extraction pipeline
2. **Page boundary error**: Content may have been assigned to page 84 or 86 during PDF parsing
3. **Pipeline timeout**: The page may have exceeded processing time limits (though this is unlikely for text-only content)
4. **Encoding issue**: The PDF text contains special characters (≥, ², fi ligatures) that may have caused parsing failure

This should be investigated at the pipeline level — this page alone contains more clinically important content than pages 81-84 combined.

---

## Raw PDF Cross-Check (2026-02-28)

### Methodology
User provided exact PDF text covering PDF pages S84-S85: ELIXA dropout caveat, PIONEER 6 oral semaglutide details, 2021 meta-analysis of 8 GLP-1 RA trials (60,080 participants), LEADER/SUSTAIN-6/REWIND/AWARD-7 kidney outcomes, meta-analysis kidney composite intro. Cross-checked all 32 ADDED spans (from prior agent pass) against exact PDF text.

### Duplicate ADDED Spans Rejected (19)

Parallel agents created 2-3 copies of each fact. Kept the version with trial name attribution; rejected duplicates and unnamed versions:

| # | Reject ID | Kept ID | Content | Reason |
|---|-----------|---------|---------|--------|
| 1 | `19846b56` | `febd1c14` | Meta-analysis CV outcomes | No context vs full meta-analysis intro |
| 2 | `3c983198` | `b43e6810` | HF hospitalization HR 0.90 | Duplicate |
| 3 | `3daea903` | `98dbb130` | PIONEER 6 | Duplicate |
| 4 | `4d798e06` | `98dbb130` | PIONEER 6 | No trial name |
| 5 | `adbe17f9` | `f74d6ad3` | LEADER kidney | Duplicate |
| 6 | `e9afc434` | `f74d6ad3` | LEADER kidney | No trial name |
| 7 | `20cf08ae` | `3215edec` | SUSTAIN-6 kidney | Duplicate |
| 8 | `1d08f175` | `3215edec` | SUSTAIN-6 kidney | No trial name |
| 9 | `59ac0eed` | `8b7ef0a4` | REWIND kidney | Duplicate |
| 10 | `0d5327ce` | `8b7ef0a4` | REWIND kidney | No trial name |
| 11 | `b9f86dd0` | `c05bc3f5` | REWIND HbA1c/BP | Duplicate |
| 12 | `1588b3cf` | `c05bc3f5` | REWIND HbA1c/BP | No trial name |
| 13 | `ab104868` | `15eb61f7` | REWIND ACEi/ARB | Duplicate |
| 14 | `ef4dc056` | `15eb61f7` | REWIND ACEi/ARB | No trial name |
| 15 | `9763dca7` | `099414f4` | AWARD-7 eGFR decline | No trial name |
| 16 | `46f7a678` | `c7faf4a9` | Hypoglycemia | No trial name |
| 17 | `840742e7` | `50153fe8` | AWARD-7 exploratory | No trial name |
| 18 | `7ffe83c3` | `ba1944df` | AWARD-7 GI safety | No trial name |
| 19 | `bab0b006` | `f9ac29d9` | FDA approval | Worse formatting (>= vs ≥) |

### Missing Gaps Added (10)

| Gap ID | Content Added (exact PDF text) | Note | Target KB |
|--------|-------------------------------|------|-----------|
| **G85-A** | the ELIXA trial had a high discontinuation and dropout rate | ELIXA evidence quality caveat — trial reliability limitation | KB-4 |
| **G85-B** | An eGFR <30 ml/min per 1.73 m2 was an exclusion criterion | PIONEER 6 excluded eGFR <30 — limits applicability to advanced CKD | KB-4 |
| **G85-C** | There was no difference between liraglutide and placebo in serum creatinine or kidney failure, and few deaths attributed to kidney disease occurred in the study. | LEADER kidney — benefit driven by albuminuria only, no hard kidney endpoint benefit | KB-4, KB-16 |
| **G85-D** | persistent severely increased albuminuria, persistent doubling of serum creatinine, a creatinine clearance of <45 ml/min, or need for kidney replacement therapy | SUSTAIN-6 kidney composite outcome definition — 4-component endpoint specification | KB-7 |
| **G85-E** | in post hoc exploratory analyses, eGFR decline thresholds of 40% and 50% were significantly reduced by 30% and 46%, respectively. Of course, exploratory results must be interpreted cautiously and regarded as hypothesis-generating. | REWIND exploratory eGFR decline with appropriate caveat | KB-4 |
| **G85-F** | Among the 9901 participants, 22.2% had an eGFR <60 ml/min per 1.73 m2 at baseline, and 7.9% had severely increased albuminuria. The benefit on the composite kidney outcome was similar among those with an eGFR ≥60 ml/min per 1.73 m2 or <60 ml/min per 1.73 m2 (P-interaction= 0.65) | REWIND CKD subgroup distribution and P-interaction=0.65 | KB-4 |
| **G85-G** | There were no serious adverse events for kidney disease in the REWIND trial. | REWIND kidney safety — no serious AEs | KB-4 |
| **G85-H** | patients with T2D and CKD G3a–G4 (mean eGFR 38 ml/min per 1.73 m2) who were being treated with an ACEi or ARB | AWARD-7 study population — mean eGFR 38, ACEi/ARB background | KB-4 |
| **G85-I** | The benefits on eGFR were most evident in the severely increased albuminuria subgroup (mean: –5.5 ml/min per 1.73 m2 vs. –0.7 ml/min per 1.73 m2 and –0.5 ml/min per 1.73 m2 over 52 weeks) with the lower and higher doses of dulaglutide, respectively. | AWARD-7 albuminuria subgroup — greatest eGFR preservation | KB-4, KB-16 |
| **G85-J** | similar improvement in HbA1c (mean 1%) and comparable blood pressure levels between the dulaglutide and insulin glargine groups | AWARD-7 kidney benefit independent of glycemic and BP control | KB-4 |

### Post-Review State (Final)

| Metric | Before Cross-Check | After Cross-Check | Change |
|--------|-------------------|-------------------|--------|
| **Total spans** | 35 (2 original + 32 agent ADDED + 3 rejected) | 26 total | -9 net |
| **CONFIRMED** | 0 | 0 | — |
| **ADDED (P2-ready)** | 32 (from agents) | 23 (32 - 19 dupes + 10 gaps) | -9 net |
| **PENDING** | 0 (2 original rejected by agents) | 0 | — |
| **REJECTED** | 3 (agents) | 22 (3 + 19 cross-check) | +19 |
| **Completeness** | ~80% (agent pass with duplicates) | ~95% (cross-checked, deduplicated) | +15% |

### Key Findings from Cross-Check

1. **Worst duplication ratio in audit**: 19/32 ADDED spans were duplicates (59%), meaning parallel agents created nearly 2x redundant data. Every fact had 2-3 copies. The unnamed versions (no trial attribution) are strictly worse for P2 processing.

2. **10 genuine gaps despite dense agent coverage**: Even with 32 agent-added spans, 10 clinically meaningful facts were missed — particularly nuanced caveats (ELIXA dropout, LEADER no hard endpoints, REWIND exploratory caveat), outcome definitions (SUSTAIN-6 composite), subgroup details (AWARD-7 albuminuria subgroup, REWIND P-interaction), and equivalence context (AWARD-7 HbA1c/BP).

3. **Pattern: agents capture headline HRs but miss caveats and context**: The agents reliably extracted the primary HR values (0.78, 0.64, 0.85, 0.25) but missed the qualifying statements that determine how those HRs should be interpreted — e.g., LEADER benefit was albuminuria-only (no hard kidney endpoints), REWIND exploratory results are hypothesis-generating only.

4. **Pipeline failure confirmed**: Original extraction produced only 2 fragments ("eGFR <60" and "at baseline") from ~2,500 words of dense evidence prose. All meaningful content came from agent additions.
