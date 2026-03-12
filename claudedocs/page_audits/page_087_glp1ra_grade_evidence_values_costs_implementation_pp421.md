# Page 87 Audit — GLP-1 RA GRADE Evidence Assessment, Values/Preferences, Resource Use/Costs, Implementation, PP 4.2.1

| Field | Value |
|-------|-------|
| **Page** | 87 (PDF page S86) |
| **Content Type** | Rec 4.2.1 GRADE evidence assessment (study design: multiple RCTs with kidney secondary endpoints, risk of bias: low from 8 RCTs/Cochrane tool — some unclear allocation concealment in CKD-focused review, consistency: moderate-high with I²=55% for MACE and I²=86% for HbA1c/I²=70% for eGFR loss, indirectness: good — placebo-controlled with 1 active comparator, precision: good for critical outcomes but imprecise for AKI/hyperkalemia in CKD, publication bias: low — clinicaltrials.gov registered), values/preferences (most patients would choose GLP-1 RA for CV benefits; barriers: GI side effects, injectable, cost/availability), resource use/costs (cost-prohibitive vs SU, preauthorization burden, copayment issues, country/regional availability), implementation (prioritize metformin + SGLT2i first → GLP-1 RA when needed; consistent with ACC/ADA/ESC/EASD; applies to transplant recipients; caution in CKD G5/KRT; GI exacerbation in PD/uremic/cachexic patients), PP 4.2.1 (choose GLP-1 RA with documented CV benefits — ELIXA/EXSCEL did not prove benefit, albiglutide/efpeglenatide unavailable) |
| **Extracted Spans** | 4 total (4 T1, 0 T2) — 0 EDITED |
| **Channels** | B (Drug Dictionary — 3 spans), C (Grammar/Regex — 1 span), F (NuExtract LLM — 1 span) |
| **Risk** | Disagreement |
| **Disagreements** | 1 |
| **Review Status** | **FINAL**: 17 ADDED (P2-ready), 0 PENDING, 30 REJECTED |
| **Audit Date** | 2026-02-24 |
| **Cross-Check** | 2026-02-25 — count corrected 20→4 (16 phantom T2 spans removed), D channel removed (raw data has B, C, F only) |
| **Raw PDF Cross-Check** | 2026-02-28 — 10 agent duplicates rejected, 9 gaps added, 8 agent spans kept |

---

## Source PDF Content

**GRADE Evidence Assessment (Rec 4.2.1):**

| Domain | Assessment |
|--------|-----------|
| **Study Design** | Multiple RCTs with adequate participants; CKD kidney outcomes as secondary/exploratory; meta-analysis of 8 RCTs confirmed CV + kidney composite benefit |
| **Risk of Bias** | Low — 8 large RCTs with good allocation concealment, adequate blinding, complete follow-up; Cochrane Risk of Bias tool: all high quality; BUT updated Cochrane CKD-focused review found unclear allocation concealment/blinding in some trials — downgraded evidence for hypoglycemia, hyperkalemia, HbA1c, eGFR loss, weight, BMI |
| **Consistency** | Moderate-high; **MACE I²=55%** (heterogeneity); no heterogeneity for secondary kidney outcomes across eGFR/ACR groups; **HbA1c I²=86%** and **eGFR loss I²=70%** (high heterogeneity) |
| **Indirectness** | Good — placebo-controlled, well-distributed confounders; 1 active comparator trial (AWARD-7) |
| **Precision** | Good for critical outcomes (large N, acceptable event rates); **downgraded for AKI and hyperkalemia in CKD** — fewer events, did not exclude minimally clinically important difference |
| **Publication Bias** | Low — all registered at clinicaltrials.gov; mostly commercially funded but no evidence of undue industry influence |

**Values and Preferences:**
- Most well-informed T2D+CKD patients who cannot take SGLT2i would choose GLP-1 RA for CV benefits
- **High ASCVD risk or residual albuminuria** + glycemic management → particularly inclined
- **Barriers**: severe GI side effects, inability to administer injectable, unaffordability, unavailability

**Resource Use and Costs:**
- Some models: GLP-1 RA cost-effective in T2D
- **Frequently cost-prohibitive** vs oral agents (e.g., SU)
- **Preauthorization burden** on healthcare professionals and patients (US)
- Large copayment even with insurance
- **Drug availability varies** among countries/regions
- Patients may need to choose cost vs anticipated benefits

**Implementation Considerations:**
- **Treatment hierarchy**: Lifestyle → metformin + SGLT2i → GLP-1 RA (when additional needed)
- Consistent with ACC, ADA, ESC/EASD recommendations
- **CV benefits sustained** regardless of age, sex, race/ethnicity
- **Applies to kidney transplant recipients** (no evidence of different outcomes)
- **Caution in CKD G5 or on KRT** — less safety data
- **May exacerbate GI symptoms** in: peritoneal dialysis, uremic/underdialyzed, cachexia/malnutrition

**Practice Point 4.2.1:**
- **"The choice of GLP-1 RA should prioritize agents with documented cardiovascular benefits"**
- ELIXA (lixisenatide) and EXSCEL (exenatide) did not prove CV benefit
- Albiglutide and efpeglenatide are [no longer available/not widely available]
- → Prioritize: **liraglutide, semaglutide, dulaglutide**

---

## Key Spans Assessment

### Tier 1 Spans (4)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "metformin" ×2 | B | 100% | **→ T3** — Drug name mentions from implementation section ("prioritizing metformin and an SGLT2i") — no clinical assertion context |
| "Practice Point 4.2.1" | C | 98% | **→ T3** — PP label only. The actual practice point text ("The choice of GLP-1 RA should prioritize agents with documented cardiovascular benefits") NOT captured |
| **"When the decision has been made to add a GLP-1 RA"** | B+F | 98% | **⚠️ T2** — B fires on "GLP-1 RA", F extracts the sentence beginning. This is the start of PP 4.2.1's implementation guidance. The full sentence continues: "given that the ELIXA (lixisenatide) and EXSCEL (exenatide) trials did not prove cardiovascular benefit with these agents..." — which IS clinically actionable. But as extracted, it's truncated to just the opening clause |

**Summary: 0/4 T1 genuinely correct. 2 drug names → T3. 1 PP label → T3. 1 B+F sentence opening → T2 (truncated but contextually relevant).**

### Tier 2 Spans (16)

| Span | Channel | Count | Conf | Assessment |
|------|---------|-------|------|------------|
| "Comparator" ×16 | D | 16 | 92% | **→ NOISE** — Column header "Comparator" from Supplementary Table S23, extracted 16 times (once per trial row). This is a table header label with zero clinical content |

**Summary: 0/16 T2 correct. All 16 are identical "Comparator" column header noise.**

---

## Critical Findings

### ✅ B+F Returns — First F Channel Activity Since Page 81

The B+F span "When the decision has been made to add a GLP-1 RA" marks the first F (NuExtract LLM) channel activity since page 81. The F channel was silent across pages 82-86 (5 pages). Its return on page 87 — on a PP implementation guidance sentence rather than evidence prose — suggests F may respond to directive language patterns.

### ❌ "Comparator" ×16 — Worst D Channel Noise Pattern

The D channel extracted the column header "Comparator" 16 times from Supplementary Table S23, once for each trial row in the table. This is the most extreme example of D channel noise: a single word repeated 16 times, all classified T2. This is worse than the "CV death, nonfatal MI, or nonfatal stroke" ×6 on page 84 because at least that was a clinical definition.

### ❌ PP 4.2.1 Label Without Text — Recurring Pattern

C channel captures "Practice Point 4.2.1" as a label but not the actual practice point: "The choice of GLP-1 RA should prioritize agents with documented cardiovascular benefits." This is the same pattern seen throughout the audit (PP 4.1.1-4.1.4 labels on pp81-82, Rec 4.2.1 label on p82).

### ❌ Critical Missing Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 4.2.1: "prioritize agents with documented CV benefits" (full text) | **T1** | Drug selection directive |
| Prioritize liraglutide, semaglutide, dulaglutide (from PP 4.2.1 context) | **T1** | Specific drug selection |
| **Caution in CKD G5 or on KRT** — less safety data | **T1** | Patient safety limitation |
| **May exacerbate GI symptoms in PD/uremic/cachexic patients** | **T1** | Patient safety warning for specific populations |
| Applies to kidney transplant recipients | **T2** | Population applicability |
| Treatment hierarchy: lifestyle → metformin + SGLT2i → GLP-1 RA | **T2** | Drug sequencing guidance |
| GRADE: I²=55% for MACE, I²=86% for HbA1c, I²=70% for eGFR loss | **T2** | Evidence quality metrics |
| Precision downgraded for AKI and hyperkalemia in CKD | **T2** | Evidence limitation |
| Cost-prohibitive vs SU; preauthorization burden (US) | **T3** | Implementation barriers |
| Drug availability varies by country | **T3** | Access considerations |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — 20 spans with 0% correct tiering; 16/20 are "Comparator" column header noise; PP 4.2.1 text missing; CKD G5/KRT caution and PD/uremic GI warnings missing; B+F captures sentence start but truncated |
| **Tier corrections** | metformin ×2: T1 → T3; "Practice Point 4.2.1": T1 → T3; B+F sentence: T1 → T2; "Comparator" ×16: T2 → NOISE |
| **Missing T1** | PP 4.2.1 full text, drug prioritization, CKD G5/KRT caution, PD/uremic/cachexia GI warning |
| **Missing T2** | Treatment hierarchy, GRADE evidence metrics, kidney transplant applicability |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~5% — 1 truncated B+F sentence from rich implementation/GRADE/PP content; all 16 D spans are column header noise |
| **Tier accuracy** | 0% (0/4 T1 correct + 0/16 T2 correct = 0/20) |
| **Noise ratio** | 90% — 16 "Comparator" + 2 drug names + 1 PP label = 19/20 noise or T3 |
| **Genuine T1 content** | 0 extracted |
| **Prior review** | 0/20 reviewed |
| **Overall quality** | **VERY POOR — FLAG** — D channel extracts "Comparator" ×16 as only table data; PP 4.2.1 drug selection directive missing; CKD G5/KRT caution and PD-specific GI warning (critical safety) missing |

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Cross-Check State
- **Original extraction**: 20 spans (4 T1 + 16 T2 "Comparator" noise) — all rejected by agent pass
- **Agent-added**: 18 ADDED spans from parallel agent processing
- **Total before cross-check**: 38 spans (20 REJECTED + 18 ADDED)

### Duplicate Rejections (10)

Parallel agents created duplicate copies of the same facts. Kept 1 copy, rejected duplicates:

| # | Duplicate Text (truncated) | Reason |
|---|---------------------------|--------|
| 1 | "CV benefits sustained regardless of age, sex, race/ethnicity" (dup) | Duplicate of kept `9eacb375` |
| 2 | "The choice of GLP-1 RA should prioritize agents with documented cardiovascular benefits" (dup) | Duplicate of kept `c0be48ec` |
| 3 | "ELIXA (lixisenatide) and EXSCEL (exenatide) trials did not prove cardiovascular benefit…" (dup) | Duplicate of kept `e1520348` |
| 4 | "Treatment hierarchy: Lifestyle → metformin + SGLT2i → GLP-1 RA" (dup) | Duplicate of kept `a24da7a2` |
| 5 | "Caution in CKD G5 or on KRT — less safety data" (dup) | Duplicate of kept `2ed218d9` |
| 6 | "May exacerbate GI symptoms in peritoneal dialysis…" (dup) | Duplicate of kept `24c49632` |
| 7 | "Applies to kidney transplant recipients" (dup) | Duplicate of kept `97f45ad2` |
| 8 | "GRADE: MACE I²=55%…" (dup) | Duplicate of kept `3cb266a4` |
| 9 | Agent variant of PP 4.2.1 drug selection text | Duplicate/overlap with `c0be48ec` + `e1520348` |
| 10 | Agent variant of treatment hierarchy/implementation text | Duplicate/overlap with `a24da7a2` |

**Duplication rate**: 10/18 agent-added = **56%** (consistent with pp84-86 pattern)

### Gap Additions (9)

All gaps represent exact PDF text missed by both extraction channels and agent pass. GRADE assessment, values/preferences, and resource use sections were entirely uncaptured.

| ID | Gap | Text | Target KB | Priority |
|----|-----|------|-----------|----------|
| G87-A | Risk of bias low | "The risk of bias is low, as the 8 large RCTs studies demonstrated good allocation concealment and adequate blinding, with complete accounting for all patients and outcome events." | KB-4 | High |
| G87-B | Cochrane CKD downgrade | "the updated Cochrane review that focused on people with diabetes and CKD found unclear reporting of allocation concealment and blinding in other included trials which downgraded the evidence for hypoglycemia requiring third-party assistance, hyperkalemia, HbA1c, eGFR loss, change in body weight, and body mass index." | KB-4 | High |
| G87-C | **No heterogeneity for kidney outcomes** | "No heterogeneity was observed for secondary kidney outcomes across baseline eGFR and baseline ACR groups." | KB-4, KB-16 | **CRITICAL** |
| G87-D | Publication bias low | "All the published RCTs were registered at clinicaltrials.gov. The majority of studies were commercially funded, but overall, there was no evidence of undue industry influence on the included RCT findings." | KB-4 | Medium |
| G87-E | Values: most patients would choose | "the majority of well-informed patients with T2D and CKD who cannot take an SGLT2i because of intolerance or a contraindication would choose to receive a GLP-1 RA because of the cardiovascular benefits associated with this class of medications." | KB-1 | Medium |
| G87-F | Values: barriers to GLP-1 RA | "patients who experience severe gastrointestinal side effects or are unable to administer an injectable medication, or those for whom GLP-1 RA are unaffordable or unavailable, will be less inclined to choose these agents." | KB-1, KB-4 | Medium |
| G87-G | Cost-prohibitive + preauthorization | "these medications are frequently cost-prohibitive for many patients compared to other oral glucose-lowering agents (e.g., sulfonylureas), which do not have evidence for cardiovascular and kidney benefits. In many cases in the US, obtaining preauthorization from insurance companies for GLP-1 RA places an undue burden on healthcare professionals and patients." | KB-1 | Medium |
| G87-H | Guideline concordance | "This approach is consistent with the recommendations from other professional societies, including the ACC, ADA, and ESC/EASD." | KB-4 | Medium |
| G87-I | Long-term follow-up needed | "long-term follow-up and ongoing collection of real-world data are needed to validate effectiveness and potential harms." | KB-4 | Medium |

### Critical Finding: G87-C — Kidney Outcome Consistency

**"No heterogeneity was observed for secondary kidney outcomes across baseline eGFR and baseline ACR groups."**

This is the most important evidence quality finding on this page. While MACE shows I²=55% (moderate heterogeneity), kidney outcomes are **consistent across all CKD subgroups**. This means:
- CQL rules for GLP-1 RA kidney benefit do NOT need eGFR/ACR stratification
- The kidney benefit signal is robust regardless of CKD stage
- Contrasts sharply with HbA1c (I²=86%) and eGFR loss (I²=70%) which DO show heterogeneity

### Agent-Kept Spans (8)

| ID | Text | Assessment |
|----|------|------------|
| `9eacb375` | "CV benefits sustained regardless of age, sex, race/ethnicity" | ✅ Correct — implementation fact |
| `c0be48ec` | "The choice of GLP-1 RA should prioritize agents with documented cardiovascular benefits" | ✅ Correct — PP 4.2.1 directive |
| `e1520348` | "ELIXA (lixisenatide) and EXSCEL (exenatide) trials did not prove cardiovascular benefit with these agents, and albiglutide and efpeglenatide are not widely available" | ✅ Correct — drug deprioritization |
| `3cb266a4` | "GRADE: MACE I²=55% (heterogeneity); HbA1c I²=86% and eGFR loss I²=70% (high heterogeneity); precision downgraded for AKI and hyperkalemia in CKD" | ✅ Correct — GRADE metrics |
| `24c49632` | "May exacerbate GI symptoms in peritoneal dialysis, uremic or underdialyzed patients, and those with cachexia or malnutrition" | ✅ Correct — safety warning |
| `2ed218d9` | "Caution in CKD G5 or on KRT — less safety data" | ✅ Correct — safety limitation |
| `a24da7a2` | "Treatment hierarchy: Lifestyle → metformin + SGLT2i → GLP-1 RA (when additional needed)" | ✅ Correct — drug sequencing |
| `97f45ad2` | "Applies to kidney transplant recipients (no evidence of different outcomes)" | ✅ Correct — population applicability |

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total spans** | 47 |
| **ADDED (P2-ready)** | 17 (8 agent-kept + 9 gap additions) |
| **PENDING** | 0 |
| **REJECTED** | 30 (20 original extraction noise + 10 agent duplicates) |
| **Extraction completeness** | ~95% — all GRADE domains, values/preferences, resource use, implementation, and PP 4.2.1 now captured |
| **Duplication rate** | 56% of agent-added spans (10/18) |
| **Cross-check date** | 2026-02-28 |
