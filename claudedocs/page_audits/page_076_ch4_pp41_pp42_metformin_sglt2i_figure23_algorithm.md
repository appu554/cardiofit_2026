# Page 76 Audit — Chapter 4 Opening: PP 4.1 (Metformin+SGLT2i First-Line), PP 4.2 (eGFR ≥30 Combination), Figure 23 (Treatment Algorithm)

| Field | Value |
|-------|-------|
| **Page** | 76 (PDF page S75) |
| **Content Type** | Chapter 4 "Glucose-lowering therapies in patients with T2D and CKD" opening, PP 4.1 (lifestyle + metformin + SGLT2i first-line, SGLT2i for eGFR ≥20, metformin when eGFR >30, GLP-1 RA preferred additional therapy for eGFR ≥15), PP 4.2 (metformin + SGLT2i combination safe when eGFR ≥30, SGLT2i weaker HbA1c at eGFR <60), Figure 23 (treatment algorithm with eGFR-based drug management: metformin reduce dose at eGFR <45, discontinue at eGFR <30; SGLT2i do not initiate at eGFR <20, discontinue at dialysis) |
| **Extracted Spans** | 12 total (5 original + 7 added) |
| **Channels** | L1 (Oracle Recovery), B (Drug Dictionary), C (Grammar/Regex), F (NuExtract LLM) |
| **Disagreements** | 4 |
| **Review Status** | CONFIRMED: 5, ADDED: 7, REJECTED: 0, PENDING: 0 |
| **Risk** | Oracle (L1 Recovery present) |
| **Cross-Check** | Counts verified against pipeline DB post-review |
| **Audit Date** | 2026-02-25 (initial), 2026-02-27 (reviewed) |

---

## Source PDF Content

**Chapter 4: Glucose-lowering therapies in patients with T2D and CKD**

**Practice Point 4.1:**
- "Glycemic management for patients with T2D and CKD should include lifestyle therapy, first-line treatment with both metformin and a sodium-glucose cotransporter-2 inhibitor (SGLT2i), and additional drug therapy as needed for glycemic control (Figure 23)"
- Lifestyle therapy = cornerstone of management
- **Metformin + SGLT2i as first-line combination** for most patients with suitable eGFR
- **SGLT2i recommended for eGFR ≥20 ml/min/1.73m²** → reduce CKD progression + major CVD events (especially heart failure)
- SGLT2i benefits not mediated by glycemia; HbA1c improvements modest, diminished at low eGFR
- **Metformin: effective, safe, inexpensive when eGFR >30**
- Additional drugs: **GLP-1 RA preferred** (safe with eGFR as low as 15, reduce ASCVD when eGFR <60, lower albuminuria, may slow eGFR decline)

**Practice Point 4.2:**
- "Most patients with T2D, CKD, and eGFR ≥30 ml/min/1.73m² would benefit from treatment with both metformin and an SGLT2i"
- Metformin: safe, effective, inexpensive foundation; modest long-term complication prevention; low hypoglycemia risk
- **SGLT2i: weaker HbA1c effects particularly with eGFR <60**, but large CKD progression + CVD event reduction, independent of eGFR
- **Together safe and effective when eGFR ≥30**
- "Metformin should not be used for..." (continues on next page)

**Figure 23 — Treatment Algorithm for Glucose-Lowering Drugs:**

| Component | Details |
|-----------|---------|
| **First-line** | Lifestyle therapy + SGLT2i + Metformin |
| **Additional drugs** | GLP-1 RA (preferred), DPP-4i, Insulin, Sulfonylurea, TZD, Alpha-glucosidase inhibitor |
| **Metformin eGFR rules** | Reduce dose at eGFR <45; Discontinue at eGFR <30 |
| **SGLT2i eGFR rules** | Do not initiate at eGFR <20; Discontinue at Dialysis |
| **Patient factors** | Guided by preferences, comorbidities, eGFR, cost |
| **Scope** | Includes patients with eGFR <30 or on dialysis (for additional drugs) |

---

## Key Spans Assessment

### Tier 1 Spans (5)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "eGFR < 45 eGFR < 30 sis ylai D sis ylai D" (L1) | L1 | 100% | **⚠️ T1 PARTIALLY CORRECT** — L1 Oracle Recovery captures Figure 23 medication discontinuation thresholds (eGFR <45, eGFR <30). The "sis ylai D" is garbled OCR for "Dialysis" (reversed text). Captures critical drug management thresholds but OCR quality is poor. The L1 channel is the ONLY channel that extracted Figure 23 content |
| "In addition, metformin and SGLT2i should be used in combination as first-line treatment for most patients with suitable..." (B+C) | B+C | 98% | **✅ T1 CORRECT** — Dual-channel capture of the core PP 4.1 first-line treatment recommendation. B fires on "metformin" and "SGLT2i", C fires on the sentence structure. This is a genuine T1 clinical assertion |
| "Practice Point 4.2" (C) | C | 98% | **→ T3** — PP label only; PP 4.2 text about metformin+SGLT2i combination at eGFR ≥30 NOT captured. Already CONFIRMED by a reviewer — this is the first confirmed span since the audit began |
| "HbA1c, particularly with an eGFR <60 mL/min/1.73m²" (C) | C | 95% | **⚠️ T2 more appropriate** — Partial sentence fragment from "SGLT2i have weaker effects on HbA1c, particularly with an eGFR <60". Captures eGFR threshold in context of drug efficacy, but missing the drug name and the "weaker effects" qualifier |
| "In most patients with T2D, CKD, and an eGFR ≥30 mL/min/1.73m², metformin and an SGLT2i can be used safely and effectivel..." (B+C+F) | B+C+F | 100% | **✅ T1 CORRECT — EXCELLENT** — **Triple-channel B+C+F capture at 100% confidence!** This is the key PP 4.2 safety assertion: metformin + SGLT2i safe and effective when eGFR ≥30. Three independent channels converge on this extraction. Second B+C+F triple in the audit (first was p67 sodium evidence) |

**Summary: 2/5 T1 genuinely correct (first-line treatment rec + B+C+F safety assertion). 1 partially correct (L1 Figure 23 OCR). 1 PP label → T3. 1 partial sentence → T2.**

---

## Critical Findings

### ✅ B+C+F TRIPLE-CHANNEL — Second Instance, 100% Confidence

"In most patients with T2D, CKD, and an eGFR ≥30 mL/min/1.73m², metformin and an SGLT2i can be used safely and effectively together" — extracted by all three of B (drug names), C (eGFR threshold regex), and F (NuExtract sentence extraction) at 100% confidence. This is the **gold standard extraction pattern**: three independent channels converge on the same clinically critical sentence.

The first B+C+F triple was on page 67 (sodium evidence). Both triples capture sentences that:
- Contain specific drug names (triggering B)
- Contain numeric eGFR thresholds (triggering C)
- Form complete clinical assertions (triggering F)

### ✅ L1 Oracle Recovery — Figure 23 Drug Algorithm

The L1 (Oracle Recovery) channel is the **only channel** that captured any content from Figure 23 — the critical treatment algorithm showing metformin and SGLT2i eGFR-based dosing rules. The D (Table Decomposition) channel did not fire on this figure (consistent with its poor performance on figures throughout the audit).

However, the OCR quality is poor: "sis ylai D" = reversed "Dialysis". This suggests the figure text was rendered in a non-standard direction (possibly rotated or mirrored in the PDF layout) and the OCR engine struggled with it.

### ✅ First Reviewed Span — PP 4.2 CONFIRMED

"Practice Point 4.2" is marked CONFIRMED — the first reviewed span encountered since the audit began. This means a reviewer has already looked at this page, though they confirmed only the PP label (which should be T3). This raises questions about whether the reviewer understood the tier system.

### ❌ GLP-1 RA Recommendation NOT EXTRACTED

PP 4.1 states GLP-1 RA are "generally preferred" as additional therapy with specific clinical rationale:
- Safe with eGFR as low as 15 ml/min/1.73m²
- Reduce ASCVD events even when eGFR <60
- Lower albuminuria
- May slow eGFR decline

This is a T1 clinical recommendation with specific eGFR thresholds. None of the four channels captured it.

### ❌ Metformin eGFR Dosing Rules NOT FULLY EXTRACTED

Figure 23 shows:
- Metformin: reduce dose at eGFR <45, discontinue at eGFR <30
- SGLT2i: do not initiate at eGFR <20, discontinue at dialysis

The L1 channel captures the raw eGFR values but the clinical action verbs ("reduce dose", "discontinue", "do not initiate") are not clearly captured due to OCR quality. The prose text on this page mentions "metformin should not be used for..." (continued on next page) which is also not captured.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| GLP-1 RA preferred additional therapy (eGFR ≥15, ASCVD reduction at eGFR <60) | **T1** | Third drug class recommendation with specific eGFR threshold |
| SGLT2i recommended for eGFR ≥20 (CKD progression + CVD reduction) | **T1** | Key eGFR initiation threshold for SGLT2i |
| Figure 23 drug actions: "reduce dose at eGFR <45", "discontinue at eGFR <30", "do not initiate at eGFR <20" | **T1** | Medication management decision points |
| PP 4.1 full text: lifestyle + metformin + SGLT2i first-line | **T2** | Treatment approach overview |
| SGLT2i benefits "not mediated by glycemia" | **T2** | Mechanism understanding for treatment decisions |
| "Low risk of hypoglycemia" for metformin + SGLT2i | **T2** | Safety profile for first-line agents |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — B+C+F triple-channel safety assertion is excellent (metformin+SGLT2i safe at eGFR ≥30); B+C first-line rec captured; L1 Oracle Recovery on Figure 23 useful despite OCR quality; but GLP-1 RA recommendation and SGLT2i eGFR ≥20 threshold missing |
| **Tier corrections** | PP 4.2 label: T1 → T3 (despite being CONFIRMED); HbA1c + eGFR <60 fragment: T1 → T2 |
| **Missing T1** | GLP-1 RA recommendation (eGFR ≥15), SGLT2i initiation threshold (eGFR ≥20), Figure 23 drug management actions |
| **Missing T2** | PP 4.1 overview text, SGLT2i non-glycemic benefits, hypoglycemia safety profile |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~35% — 2 genuine T1 sentences + L1 figure capture from a page with 2 PPs, GLP-1 RA recommendation, and complete treatment algorithm |
| **Tier accuracy** | ~60% (3/5 T1 correct or partially correct = 60%) |
| **Noise ratio** | ~20% — Only PP label is true noise; HbA1c fragment is partial but has some value |
| **Genuine T1 content** | 2 fully correct (first-line rec + B+C+F safety) + 1 partial (L1 figure) |
| **Prior review** | 1/5 reviewed (PP 4.2 CONFIRMED) |
| **Overall quality** | **GOOD** — Best Chapter 4 page so far; B+C+F triple is audit highlight; L1 Oracle Recovery adds figure content; low noise |

---

## Chapter Transition Assessment

Page 76 marks the **Chapter 3 → Chapter 4 transition**. Comparing extraction quality:

| Chapter | Topic | Avg Extraction | Best Pattern | Worst Pattern |
|---------|-------|---------------|--------------|---------------|
| Ch1 (pp20-53) | Medications (ACEi/ARB, SGLT2i, MRA) | ~40% | B+C drug+threshold | Repetitive drug name noise |
| Ch2 (pp54-63) | Glycemic Monitoring | ~25% | HbA1c thresholds | HbA1c ×100+ noise explosion |
| Ch3 (pp64-75) | Lifestyle (Nutrition, Exercise) | ~15% | F evidence prose | PP labels without text, 0% pages |
| Ch4 (p76) | Glucose-Lowering Drugs | ~35% | B+C+F triple | GLP-1 RA missed |

Chapter 4 immediately shows improvement over Chapter 3 because the content returns to **drug-focused clinical assertions** — the pipeline's strongest content type. The B+C+F triple-channel pattern fires when:
1. Drug names present (B channel)
2. eGFR/numeric thresholds present (C channel)
3. Complete clinical sentences present (F channel)

Chapter 3's lifestyle content (exercise, nutrition) lacked drug names entirely, so B channel was nearly silent and C channel only had threshold fragments.

---

## Raw PDF Gap Analysis (Pipeline 2 L3-L5 Perspective)

### KB-1 (Dosing/Dietary) Relevance — HIGH
- **Metformin eGFR dosing rules** (ADDED): reduce dose at eGFR <45, discontinue at eGFR <30 — mandatory prescribing thresholds
- **SGLT2i eGFR rules** (ADDED): do not initiate at eGFR <20, discontinue at dialysis — mandatory prescribing thresholds
- **Metformin + SGLT2i first-line combination** (CONFIRMED + ADDED): core treatment algorithm
- **GLP-1 RA as preferred third-line** (ADDED): treatment escalation pathway with eGFR >=15 floor
- **Metformin safety profile** (ADDED): safe at eGFR >30, low hypoglycemia risk

### KB-4 (Patient Safety) Relevance — HIGH
- **Drug combination safety**: metformin + SGLT2i safe at eGFR >=30 (B+C+F triple CONFIRMED)
- **eGFR-based discontinuation**: mandatory stop points for patient safety
- **Low hypoglycemia risk**: metformin + SGLT2i combination safety profile
- **GLP-1 RA cardiovascular protection**: ASCVD reduction at eGFR <60

### KB-16 (Lab Monitoring) Relevance — HIGH
- **eGFR monitoring triggers**: eGFR <45 (metformin dose reduce), eGFR <30 (metformin stop), eGFR <20 (SGLT2i stop)
- **HbA1c expectations**: SGLT2i weaker HbA1c effect at eGFR <60 (CONFIRMED)
- **Treatment response monitoring**: eGFR trajectory as drug management decision point

### What Pipeline 2 L3 Needs from This Page
This is the **pharmacological heart** of KDIGO 2022. Every span added is high-value for L3 fact extraction:
- 3 drug classes with specific eGFR thresholds (metformin, SGLT2i, GLP-1 RA)
- Figure 23 treatment algorithm codified as structured decision rules
- Drug combination safety assertions with quantified eGFR floors
- Treatment escalation pathway (first-line -> additional drugs)

---

## Post-Review State (2026-02-27)

| Metric | Value |
|--------|-------|
| **Total spans** | 12 (5 original + 7 added) |
| **REJECTED** | 0 |
| **CONFIRMED** | 5 (L1 Figure 23 OCR, B+C first-line rec, PP 4.2 label, HbA1c/eGFR<60 fragment, B+C+F safety assertion) |
| **ADDED** | 7 (PP 4.1 full text, SGLT2i eGFR>=20, GLP-1 RA preferred, GLP-1 RA eGFR>=15, Figure 23 metformin rules, Figure 23 SGLT2i rules, metformin safety profile) |
| **PENDING** | 0 |
| **Review completeness** | 100% — all 12 spans decided |
| **Updated completeness** | ~90% — 12 spans now cover PP 4.1, PP 4.2, Figure 23 drug rules, GLP-1 RA recommendation, and drug safety profiles |
| **Reviewer** | claude-auditor |
| **Review date** | 2026-02-27 |

### Post-Review Remaining Gaps
- SGLT2i benefits "not mediated by glycemia" — mechanistic nuance not added (lower priority)
- Additional drug classes from Figure 23 (DPP-4i, insulin, SU, TZD, alpha-glucosidase inhibitors) — mentioned but not individually extracted (these are on subsequent pages)
- "Patient factors: guided by preferences, comorbidities, eGFR, cost" — Figure 23 patient-centered note not added

---

## Raw PDF Cross-Check Gap Analysis (2026-02-28)

| # | Gap Text (Exact PDF) | Priority | Rationale |
|---|---------------------|----------|-----------|
| 1 | "Lifestyle therapy is the cornerstone of management for patients with T2D and CKD." | **MODERATE** | Foundation principle for Chapter 4 treatment approach. KB-1. |
| 2 | "These benefits of SGLT2i do not appear to be mediated by glycemia. Nonetheless, SGLT2i do also lower blood glucose, with improvements in HbA1c that are modest and diminished at low eGFR." | **MODERATE** | SGLT2i mechanism: benefits non-glycemic; HbA1c effect modest and eGFR-dependent. KB-1, KB-16. |
| 3 | "Therefore, a combination of metformin and SGLT2i is a logical foundation for glycemic control in suitable patients with T2D. Additional glucose-lowering drugs can be added to this base drug therapy as needed to achieve glycemic targets." | **MODERATE** | Treatment escalation rationale: metformin+SGLT2i foundation + add-on pathway. KB-1. |
| 4 | "These recommendations are guided in large part by results of recent large RCTs, summarized in Figure 24 and detailed in Sections 1.3, 4.1, and 4.2." | **MODERATE** | Evidence provenance: RCT-guided recommendations with cross-references. KB-1. |
| 5 | "Most patients with T2D, CKD, and eGFR ≥30 ml/min per 1.73 m2 would benefit from treatment with both metformin and an SGLT2i." | **MODERATE** | PP 4.2 full text — only label was confirmed, not the recommendation. KB-1. |
| 6 | "Both metformin and SGLT2i agents are preferred glucose-lowering medications for patients with T2D, CKD, and suitable eGFR. Metformin and SGLT2i each reduce the risk of developing diabetes complications with a low risk of hypoglycemia." | **MODERATE** | Both drugs preferred + low hypoglycemia risk for combination. KB-1, KB-4. |
| 7 | "Metformin has been proven to be a safe, effective, and inexpensive foundation for glycemic control in T2D, with modest long-term benefits for the prevention of diabetes complications." | **MODERATE** | Metformin evidence: modest long-term complication prevention. KB-1. |
| 8 | "In comparison, SGLT2i have weaker effects on HbA1c, particularly with an eGFR <60 ml/min per 1.73 m2, but they have large effects on reducing CKD progression and CVD events, especially heart failure, which appear to be independent of eGFR." | **MODERATE** | SGLT2i full comparison: weaker HbA1c but large CKD/CVD effects independent of eGFR. KB-1, KB-16. |
| 9 | "Figure 23. Additional drug therapy as needed for glycemic control: GLP-1 receptor agonist (preferred), DPP-4 inhibitor, Insulin, Sulfonylurea, TZD, Alpha-glucosidase inhibitor. Guided by patient preferences, comorbidities, eGFR, and cost. Includes patients with eGFR <30 ml/min per 1.73 m2 or treated with dialysis." | **MODERATE** | Figure 23 additional drug classes and patient-centered decision factors. KB-1. |

**All 9 gaps added via API (all 201).**

| # | Gap Text (Exact PDF) | Priority | Rationale |
|---|---------------------|----------|-----------|
| 10 | "Figure 23. Treatment algorithm for selecting glucose-lowering drugs for patients with T2D and CKD. Lifestyle therapy (Physical activity, Nutrition, Weight loss). First-line therapy: Metformin + SGLT2 inhibitor. Metformin: Reduce dose when eGFR <45 ml/min per 1.73 m2; Discontinue when eGFR <30 ml/min per 1.73 m2. SGLT2 inhibitor: Do not initiate when eGFR <20 ml/min per 1.73 m2; Discontinue when patient starts dialysis. Additional drug therapy as needed for glycemic control: GLP-1 receptor agonist (preferred), DPP-4 inhibitor, Insulin, Sulfonylurea, Thiazolidinedione (TZD), Alpha-glucosidase inhibitor. Guided by patient preferences, comorbidities, eGFR, and cost. Includes patients with eGFR <30 ml/min per 1.73 m2 or treated with dialysis. See Figure 25." | **MODERATE** | Complete Figure 23 treatment algorithm flowchart — all decision nodes including lifestyle therapy sub-items, first-line Metformin+SGLT2i, eGFR-based dose rules, additional drug classes, and Figure 25 cross-reference. KB-1, KB-4, KB-16. |

**Gap 10 added via API (201).**

---

## Post-Review State (Final — with raw PDF gap fills + verbatim Figure 23)

| Metric | Value |
|--------|-------|
| **Total spans (post-review)** | 22 (5 original + 7 agent-added + 10 gap-fill) |
| **Reviewed** | 22/22 (100%) |
| **REJECTED** | 0 |
| **CONFIRMED** | 5 |
| **ADDED (agent)** | 7 |
| **ADDED (gap fill)** | 10 |
| **Total ADDED** | 17 |
| **Pipeline 2 ready** | 22 (5 confirmed + 17 added) |
| **Completeness (post-review)** | ~97% — PP 4.1 full text (lifestyle + metformin + SGLT2i first-line); PP 4.2 full text (metformin + SGLT2i benefit at eGFR ≥30); lifestyle cornerstone principle; metformin+SGLT2i first-line combination (B+C confirmed); metformin+SGLT2i safe at eGFR ≥30 (B+C+F triple); SGLT2i eGFR ≥20 for CKD/CVD; SGLT2i non-glycemic benefits + modest HbA1c; SGLT2i weaker HbA1c at eGFR <60 but large CKD/CVD effects; metformin safe/effective/inexpensive at eGFR >30 + modest long-term complication prevention; combination foundation + escalation pathway; GLP-1 RA preferred additional (eGFR ≥15, ASCVD, albuminuria, eGFR decline); both drugs preferred + low hypoglycemia; Figure 23 metformin rules (reduce eGFR <45, stop eGFR <30); Figure 23 SGLT2i rules (no initiate eGFR <20, stop dialysis); Figure 23 additional drug classes + patient factors; Figure 23 complete verbatim flowchart (all nodes: lifestyle → first-line → eGFR rules → additional drugs → Figure 25 ref); RCT evidence provenance |
| **Remaining gaps** | Section cross-references (T3) |
| **Review Status** | COMPLETE |
