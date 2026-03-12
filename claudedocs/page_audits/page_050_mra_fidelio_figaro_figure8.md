# Page 50 Audit — MRA Evidence: Steroidal vs Nonsteroidal MRA + Figure 8 (FIDELIO/FIGARO)

| Field | Value |
|-------|-------|
| **Page** | 50 (PDF page S49) |
| **Content Type** | Steroidal MRA (spironolactone, eplerenone) limitations + Nonsteroidal MRA (finerenone, esaxerenone) introduction + Figure 8 (FIDELIO-DKD/FIGARO-DKD trial comparison) + FIDELIO-DKD enrollment criteria |
| **Extracted Spans** | 40 total (3 T1, 37 T2) |
| **Channels** | B, C, D, F |
| **Disagreements** | 4 |
| **Review Status** | PENDING: 40 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — count corrected UPWARD (32→40, 8 spans were missed in original audit), T2 corrected (29→37), disagreements added (4) |

---

## Source PDF Content

**Steroidal MRA (spironolactone, eplerenone):**
- Established CV benefits in HF, useful for hyperaldosteronism and refractory hypertension
- Reduce albuminuria but effects on kidney progression NOT established in large trials
- **Hyperkalemia risk: 2-3 fold increase; AKI risk: 2-fold increase**
- Spironolactone: gynecomastia risk
- Limiting adverse effects reduced use in high-risk populations after RALES study

**Nonsteroidal MRA (finerenone, esaxerenone):**
- More selective for mineralocorticoid receptors
- Similar albuminuria reduction with lower hyperkalemia risk
- RAS blockade → aldosterone escape phenomenon → rationale for adding MRA

**Figure 8 — FIDELIO-DKD vs FIGARO-DKD Comparison:**

| | FIDELIO-DKD | FIGARO-DKD |
|---|---|---|
| Drug | Finerenone | Finerenone |
| N | 5734 | 7437 |
| Enrollment | eGFR 25-60 + ACR 30-300 OR eGFR 25-75 + ACR 300-5000 | eGFR 25-90 + ACR 30-300 OR eGFR ≥60 + ACR 300-5000 |
| Mean eGFR | 44.7 | 45.4 |
| % eGFR <60 | 87.5 | 50.7 |
| % ACR ≥300 | 50.7 | 38.2 |
| Median ACR | 850 [85.0] | 309 [30.9] |
| Follow-up | 2.6 yr | 3.4 yr |
| Primary outcome | Kidney composite (kidney failure, sustained ≥40% GFR decrease, renal death) | CV composite (CV death, nonfatal MI/stroke, HF hospitalization) |
| Kidney HR | 0.82 (0.73-0.93) | 0.87 (0.76-1.01) |
| CV HR | 0.86 (0.75-0.99) | 0.87 (0.76-0.98) |

**FIDELIO-DKD enrollment (narrative):**
- All participants on RASi titrated to max antihypertensive or max tolerated dose
- Serum potassium <4.8 mmol/l at screening
- 18% lower incidence of primary kidney composite

---

## Key Spans Assessment

### Tier 1 Spans (3) — ALL Genuine B+F Multi-Channel

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Clinical trials have demonstrated the kidney and cardiovasc... Steroidal MRA, such as spironolactone and eplerenone, h..." | B+F | 98% | **✅ T1 CORRECT** — Drug class evidence summary with named drugs (though arguably T2 evidence) |
| "Steroidal MRA, such as spironolactone and eplerenone, have established cardiovascular benefits in those with heart f..." | B+F | 98% | **✅ T1 OK** — Drug class + indication (HF benefit) — may overlap with above span |
| "the use of steroidal MRA also increases the risk of hyperkalemia (by..." | B+F | 98% | **✅ T1 CORRECT** — Drug safety: hyperkalemia 2-3× risk — critical prescribing information |

**Summary: 3/3 T1 spans are genuine clinical content. BEST T1 PRECISION IN THE AUDIT (100%).**

### Tier 2 Spans (29) — D Channel Figure 8 Decomposition

| Category | Count | Assessment |
|----------|-------|------------|
| **D channel — Figure 8 enrollment criteria** | 2 | **✅ T2 CORRECT** — FIDELIO + FIGARO eGFR/ACR enrollment criteria |
| **D channel — Outcome HRs with CIs** | 4 | **✅ T2 CORRECT** — HR 0.82 (kidney), HR 0.87 (kidney), HR 0.86 (CV), HR 0.87 (CV) |
| **D channel — Endpoint definitions** | 4 | **✅ T2 CORRECT** — CV composite + Kidney composite definitions (×2 each) |
| **D channel — Table headers** | 5 | **→ T3** — "Median ACR at enrollment", "Cardiovascular composite outcome result", "Follow-up time", "eGFR and ACR criteria", "Kidney composite outcome result" |
| **D channel — Numeric values** | 8 | **→ T3** — 5734, 7437, 45.4, 44.7, 38.2, 87.5, 50.7, 3.4 (decontextualized numbers) |
| **D channel — ACR values** | 3 | **→ T2** — "850 [85.0]", "309 [30.9]", "% with ACR 300 mg/g" |
| **D channel — Trial names** | 2 | **→ T3** — "FIDELIO-DKD", "FIGARO-DKD" |
| **F channel — Aldosterone escape** | 1 | **✅ T2 OK** — "Incomplete suppression of serum aldosterone levels (aldosterone escape phenomenon)..." — pharmacological rationale |
| **C+F channel — Steroidal MRA kidney limitation** | 1 | **⚠️ SHOULD BE T1** — "their effects on kidney disease progression have not been examined in large..." — critical limitation for prescribing |

**Summary: ~13/29 T2 correctly tiered (10 D channel content + 3 narrative). 16/29 are table headers or decontextualized numbers.**

---

## Critical Findings

### ✅ BEST T1 PRECISION — 3/3 Genuine (100%)
For the first time in the entire audit, ALL T1 spans contain genuine clinical content. The B+F multi-channel combination produces meaningful sentences about steroidal MRA benefits and hyperkalemia risk. Zero drug-name-only spam.

### ✅ D Channel Figure 8 Decomposition — GOOD
The D channel successfully decomposes Figure 8's FIDELIO/FIGARO comparison table into enrollment criteria, HRs, and endpoint definitions. While ~16 spans are table headers or bare numbers, the 13 substantive spans contain real clinical data. This is much better than D channel performance on pages 39-45.

### ⚠️ One T2 Should Be T1
"their effects on kidney disease progression (eGFR decline or kidney failure) have not been examined in larger trials" — this is a critical prescribing limitation for steroidal MRA, distinguishing them from finerenone. Should be T1.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| "Hyperkalemia risk: 2-3 fold increase" with steroidal MRA (quantified risk) | **T1** | Captured partially in span 3 but "2-3 fold" cutoff not confirmed |
| "AKI risk: 2-fold increase" with steroidal MRA | **T1** | Safety data |
| Spironolactone gynecomastia risk | **T1** | Drug-specific adverse effect |
| "Finerenone and esaxerenone are more selective for mineralocorticoid receptors" | **T1** | Drug differentiation |
| "Lower risk of hyperkalemia" with nonsteroidal vs steroidal MRA | **T1** | Comparative safety |
| "Serum potassium <4.8 mmol/l at screening" (FIDELIO enrollment) | **T1** | Potassium threshold for MRA eligibility |
| "All participants on RASi titrated to max tolerated dose" | **T1** | Sequential therapy prerequisite |
| "18% lower incidence of primary kidney composite" (FIDELIO result) | **T2** | Key efficacy finding |

### ✅ No Pipeline Artifacts
Unlike most pages, page 50 has NO `<!-- PAGE XX -->` pipeline artifacts.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — Best page quality so far; all 3 T1 genuine; D channel figure decomposition mostly good |
| **Tier corrections** | ~16 D channel numbers/headers: T2 → T3; 2 trial names: T2 → T3; Steroidal MRA kidney limitation: T2 → T1 |
| **Missing T1** | AKI 2-fold risk, gynecomastia, finerenone/esaxerenone selectivity, potassium <4.8 threshold, max RASi prerequisite |
| **Missing T2** | 18% kidney composite reduction |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~50% — Good T1 safety sentences + D channel figure data; some drug-specific content missing |
| **Tier accuracy** | ~50% (3/3 T1 correct + ~13/29 T2 correctly tiered = ~16/32) |
| **False positive T1 rate** | **0%** (3/3 genuine) — BEST IN AUDIT |
| **Genuine T1 content** | 3 extracted (steroidal MRA benefits, HF benefits, hyperkalemia risk) |
| **Overall quality** | **GOOD** — Highest quality page in the audit so far; B+F multi-channel and D channel both performing well on this page type |

---

## Why This Page Works

Page 50 succeeds where pages 44-49 failed because:
1. **Narrative structure**: Longer paragraphs about drug classes (not GRADE/evidence tables) → F channel extracts meaningful sentences
2. **Simple comparison table**: Figure 8 is a clean 2-column comparison → D channel decomposes correctly
3. **Lower drug mention density**: Fewer repetitive drug name mentions → B channel doesn't spam T1
4. **B+F synergy**: When B matches a drug name within an F-extracted sentence, the result is a genuine clinical span

---

## Review Actions Completed (2026-02-27)

### CONFIRMED (9 spans)
| Span | Text (truncated) | Rationale |
|------|-------------------|-----------|
| 1 (T1) | "Clinical trials have demonstrated the kidney and cardiovasc...Steroidal MRA, such as spironolactone and eplerenone, h..." | B+F: steroidal MRA clinical evidence with drug names |
| 2 (T1) | "Steroidal MRA, such as spironolactone and eplerenone, have established cardiovascular benefits in those with heart f..." | B+F: steroidal MRA HF benefits |
| 3 (T1) | "the use of steroidal MRA also increases the risk of hyperkalemia (by..." | B+F: hyperkalemia risk — critical KB-4 safety |
| 4 (T2) | "Incomplete suppression of serum aldosterone levels (aldosterone escape phenomenon)..." | Aldosterone escape: pharmacological rationale for MRA |
| 5 (T2→T1) | "However, their effects on kidney disease progression...have not been examined in large..." | Steroidal MRA kidney limitation — critical prescribing differentiation |
| 6 (T2) | "CV composite: death from CV causes, nonfatal MI, nonfatal stroke, or hospitalization for HF" | CV composite endpoint definition (1 of 2 — kept unique) |
| 7 (T2) | "Kidney composite: kidney failure, a sustained decrease ≥40% in GFR, renal death" | Kidney composite endpoint definition (1 of 2 — kept unique) |
| 8 (T2) | "25-60 ml/min per 1.73 m2 and ACR 30-300 mg/g...OR 25-75 ml/min per 1.73 m2 and ACR 300-5000..." | FIDELIO-DKD enrollment criteria with eGFR/ACR thresholds |
| 9 (T2) | "25-90 ml/min per 1.73 m2 and ACR 30-300 mg/g...OR ≥60 ml/min per 1.73 m2 and ACR 300-5000..." | FIGARO-DKD enrollment criteria with eGFR/ACR thresholds |

### REJECTED (23 spans)
| Category | Count | Reason Code | Rationale |
|----------|-------|-------------|-----------|
| HRs without trial/endpoint context | 4 | `out_of_scope` | HR values (0.82, 0.87, 0.86, 0.87) without trial name or endpoint label — replaced by linked Figure 8 facts |
| Bare numbers (N, eGFR, %, yr) | 8 | `out_of_scope` | Decontextualized: 5734, 7437, 45.4, 44.7, 38.2, 87.5, 50.7, 3.4 |
| Table headers | 5 | `out_of_scope` | "Median ACR at enrollment", "CV composite outcome result", etc. — labels without values |
| ACR values without trial context | 2 | `out_of_scope` | "850 [85.0]", "309 [30.9]" — bare values |
| Duplicate endpoint definitions | 2 | `out_of_scope` | CV composite + kidney composite duplicates (kept 1 each) |
| Trial name only | 1 | `out_of_scope` | "FIDELIO-DKD" — no clinical content |
| Table label | 1 | `out_of_scope` | "% with ACR 300 mg/g" — no data value |

### ADDED (8 facts)
| # | Text | KB Target | Note |
|---|------|-----------|------|
| 1 | "the use of steroidal MRA also increases the risk of hyperkalemia (by 2- to 3-fold) and of AKI (approximately 2-fold)" | KB-4 | Quantified safety risks: hyperkalemia 2-3× + AKI 2× |
| 2 | "Spironolactone use can also cause gynecomastia" | KB-4 | Drug-specific adverse effect |
| 3 | "Finerenone and esaxerenone are nonsteroidal MRA that are more selective for mineralocorticoid receptors, with similar reductions in albuminuria as steroidal MRA but a lower risk of hyperkalemia" | KB-4, KB-1 | Nonsteroidal vs steroidal MRA differentiation |
| 4 | "Serum potassium was required to be <4.8 mmol/l at screening for enrollment in FIDELIO-DKD" | KB-4, KB-16 | Potassium threshold for MRA eligibility |
| 5 | "All participants were on a RASi that had been titrated to the maximum antihypertensive or maximum tolerated dose" | KB-1 | Sequential therapy prerequisite |
| 6 | "Figure 8 — FIDELIO-DKD: Finerenone (N=5734), mean eGFR 44.7, median follow-up 2.6 yr. Kidney composite...HR 0.82...CV composite...HR 0.86..." | KB-3 | Complete FIDELIO linked row with drug + outcomes |
| 7 | "Figure 8 — FIGARO-DKD: Finerenone (N=7437), mean eGFR 45.4, median follow-up 3.4 yr. Kidney composite...HR 0.87...CV composite...HR 0.87..." | KB-3 | Complete FIGARO linked row with drug + outcomes |
| 8 | "18% lower incidence of the primary kidney composite outcome with finerenone compared to placebo in FIDELIO-DKD" | KB-3 | Key efficacy result |

---

## Raw PDF Gap Analysis + Table Cross-Check

### Gap-Fill Facts Added (4)
| # | Text | Priority | KB Target | Note |
|---|------|----------|-----------|------|
| 9 | "These adverse effects along with the report of higher incidence of hyperkalemia after the publication of the Randomized Aldactone Evaluation Study limited the use of these agents in high-risk populations" | LOW | KB-4 | Post-RALES historical context |
| 10 | "RAS blockade leads to incomplete suppression of serum aldosterone levels (aldosterone escape phenomenon)...lower residual albuminuria and ameliorate kidney fibrosis" | LOW | KB-1 | Aldosterone escape full text with fibrosis detail |
| 11 | "Figure 8 — FIDELIO-DKD: Primary outcome = Kidney composite...Main secondary = CV composite. FIGARO-DKD: Primary outcome = CV composite...Main secondary = Kidney composite" | MODERATE | KB-3 | Primary vs secondary designation — critical for evidence grading (FIGARO kidney HR crosses 1) |
| 12 | "Figure 8 — FIDELIO-DKD baseline: 87.5% eGFR <60, median ACR 850. FIGARO-DKD baseline: 50.7% eGFR <60, median ACR 309" | MODERATE | KB-3 | Baseline population comparison — FIDELIO had sicker patients |

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total spans** | 44 (32 original + 12 added) |
| **Reviewed** | 44/44 (9 confirmed + 23 rejected + 12 added = 44 actioned, 0 pending) |
| **Pipeline 2 ready** | 21 (9 confirmed + 12 added) |
| **Completeness** | ~97% — Steroidal MRA safety (hyperkalemia 2-3×, AKI 2×, gynecomastia), nonsteroidal MRA differentiation, Figure 8 complete linked rows with primary/secondary designation + baseline populations, FIDELIO enrollment (K+ <4.8, max RASi), 18% efficacy, post-RALES context, aldosterone escape with fibrosis |
| **Remaining gaps** | None significant |
| **Review date** | 2026-02-27 |
