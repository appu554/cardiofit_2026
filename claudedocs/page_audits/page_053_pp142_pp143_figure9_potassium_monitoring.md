# Page 53 Audit — PP 1.4.2 (MRA+SGLT2i Combination), PP 1.4.3 (Potassium Monitoring), Figure 9

| Field | Value |
|-------|-------|
| **Page** | 53 (PDF page S52) |
| **Content Type** | PP 1.4.1 rationale (albuminuria ≥30 mg/g target population), PP 1.4.2 (nonsteroidal MRA + RASi + SGLT2i combination), PP 1.4.3 (potassium monitoring protocol + finerenone dosing), Figure 9 (serum potassium monitoring flowchart during finerenone treatment) |
| **Extracted Spans** | 125 total (75 T1, 50 T2) |
| **Channels** | B, C, E, F |
| **Disagreements** | 2 |
| **Review Status** | EDITED: 1, PENDING: 124 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (125), channels confirmed (B/C/E/F), disagreements added (2), review status added |

---

## Source PDF Content

**PP 1.4.1 Rationale (Continued from Page 52):**
- FIDELIO/FIGARO enrolled T2D + CKD with albuminuria ≥30 mg/g despite standard of care (RASi + glycemia/BP medications)
- Most logical application: patients with high residual risks evidenced by albuminuria despite first-line therapies

**Practice Point 1.4.2:**
- "A nonsteroidal MRA can be added to a RASi and an SGLT2i for treatment of T2D and CKD"
- Strong recommendation for SGLT2i as first-line (Rec 1.3.1); SGLT2i not required in FIDELIO/FIGARO
- 877 participants using SGLT2i at baseline — CV effects of finerenone at least as beneficial
- "SGLT2i may reduce the risk of hyperkalemia for patients treated concomitantly with a RASi and nonsteroidal MRA"
- Complementary mechanisms suggest benefits may be additive
- Finerenone may be added to RASi alone for patients who don't tolerate SGLT2i

**Practice Point 1.4.3 (CRITICAL — Potassium Monitoring Protocol):**
- "Select patients with consistently normal serum potassium concentration and monitor serum potassium regularly after initiation of a nonsteroidal MRA"
- FIDELIO/FIGARO restricted eligibility: serum potassium consistently ≤4.8 mmol/l during screening
- **Monitoring schedule**: K+ at 1 month after initiation, 4 months after initiation, then every 4 months
- **Continue if K+ ≤5.5 mmol/l**
- **Hold if K+ >5.5 mmol/l**: temporarily withhold, recheck within 72 hours
- **Resume when K+ ≤5.0 mmol/l**
- Dietary potassium restriction and concomitant medications (diuretics, potassium binders) allowed

**Figure 9 — Finerenone Potassium Monitoring Flowchart:**

| K+ Level | Action |
|----------|--------|
| **K+ ≤4.8 mmol/l** | Start finerenone: 10 mg daily if eGFR 25-59; 20 mg daily if eGFR ≥60 |
| **K+ 4.9-5.5 mmol/l** | Continue finerenone 10mg or 20mg; Monitor K+ every 4 months; Consider dose increase to 20mg if on 10mg; Restart 10mg if previously held and K+ now ≤5.0 |
| **K+ >5.5 mmol/l** | Hold finerenone; Recheck K+; Consider reinitiation if/when K+ ≤5.0; Consider dietary/medication adjustments |

- FDA approved initiation threshold: K+ <5.0 mmol/l
- Monitor serum creatinine/eGFR concurrently with potassium

---

## Key Spans Assessment

### Tier 1 Spans (75)

| Category | Count | Assessment |
|----------|-------|------------|
| **B+C+F: "Continue finerenone 10 mg or 20 mg / Monitor K+ every 4 months"** | 1 | **✅ T1 CORRECT** — Genuine dosing + monitoring instruction from Figure 9 |
| **C: "potassium ≤5.5 mmol/L. With serum potassium > 5.5 mmol/L"** (EDITED) | 1 | **✅ T1 CORRECT** — Critical potassium threshold for hold/continue decision (prior reviewer engaged!) |
| **"MRA"** (B channel) | ~20 | **ALL → T3** — Drug class name only |
| **"SGLT2i"** (B channel) | ~18 | **ALL → T3** — Drug class name only |
| **"finerenone"** (B channel) | ~14 | **ALL → T3** — Drug name only |
| **"Practice Point 1.4.2"** (C channel) | 2 | **→ T3** — PP label (text NOT captured) |
| **"Practice Point 1.4.3"** (C channel) | 3 | **→ T3** — PP label (text NOT captured) |
| **"Recommendation 1.3.1"** (C channel) | 1 | **→ T3** — Rec label only |
| **"strong recommendation"** (C channel) | 2 | **→ T2** — GRADE strength label without context |
| **"ACEi"** (B channel) | 1 | **→ T3** — Drug class name |
| **"ARB"** (B channel) | 1 | **→ T3** — Drug class name |
| **"diuretics"** (B channel) | 1 | **→ T3** — Drug class name |
| **"eGFR ≥60 mL/min/1.73m²"** (C channel) | 1 | **→ T2** — Decontextualized threshold |

**Summary: 2/75 T1 spans are genuine (2.7%). 73/75 are drug/class names (MRA ×20, SGLT2i ×18, finerenone ×14) or PP/Rec labels.**

### Tier 2 Spans (50)

| Category | Count | Assessment |
|----------|-------|------------|
| **"potassium"** (C channel) | ~13 | **ALL → T3** — Electrolyte name repeated 13 times |
| **"potassium"** (E channel) | 1 | **→ T3** — GLiNER NER duplicate |
| **Dose fragments**: "10 mg" ×3, "20 mg" ×3, "30 mg" ×4, "3 mg" ×3, "300 mg", "299 mg", "29.9 mg" | ~17 | **ALL → T3** — Decontextualized dose/ACR numbers |
| **"daily"** (C channel) | ~5 | **→ T3** — Frequency word without drug context |
| **"RASi"** (B channel) | ~6 | **→ T3** — Drug class abbreviation only |
| **"at baseline"** (C channel) | 2 | **→ T3** — Temporal fragment |
| **"HbA1c"** (C channel) | 1 | **→ T3** — Lab name only |
| **"K+ ≤4.8 mmol/L"** (C channel) | 1 | **⚠️ SHOULD BE T1** — Finerenone initiation potassium threshold |
| **"K+ > 5.5 mmol/L"** (C channel) | 1 | **⚠️ SHOULD BE T1** — Finerenone hold threshold (critical safety) |
| **"eGFR 25-59 mL/min/1.73m²"** (C channel) | 1 | **✅ T2 OK** — Dosing eGFR range from Figure 9 |
| **"Hold"** (C channel) | 1 | **→ T3** — Action verb without drug context |
| **"avoid"** (E channel) | 1 | **→ T3** — GLiNER NER action word |

**Summary: ~3/50 T2 correctly tiered. 2 should be T1 (potassium thresholds). ~45/50 are noise.**

---

## Critical Findings

### ✅ Two Genuine T1 Spans (Both from Figure 9)

1. **Finerenone dosing + monitoring** (B+C+F triple-channel): "Continue finerenone 10 mg or 20 mg / Monitor K+ every 4 months" — the first complete drug+dose+monitoring instruction captured from Figure 9
2. **Potassium threshold** (C channel, EDITED): "potassium ≤5.5 mmol/L. With serum potassium > 5.5 mmol/L" — the continue/hold decision threshold. **Prior reviewer engaged with this span.**

### ⚠️ Two T2 Potassium Thresholds Should Be T1
- "K+ ≤4.8 mmol/L" — finerenone initiation eligibility threshold (T1 prescribing criterion)
- "K+ > 5.5 mmol/L" — finerenone hold threshold (T1 safety action trigger)

### ❌ PP 1.4.2 Full Text NOT EXTRACTED
"A nonsteroidal MRA can be added to a RASi and an SGLT2i for treatment of T2D and CKD" — critical drug combination guidance. Only "Practice Point 1.4.2" labels captured (×2).

### ❌ PP 1.4.3 Full Text NOT EXTRACTED
"Select patients with consistently normal serum potassium concentration and monitor serum potassium regularly after initiation of a nonsteroidal MRA" — critical safety instruction. Only "Practice Point 1.4.3" labels captured (×3).

### ❌ Figure 9 Dosing Protocol Largely Missing
The complete finerenone dosing by eGFR (10mg if eGFR 25-59, 20mg if eGFR ≥60) and the hold/resume protocol (hold >5.5, resume ≤5.0, recheck within 72 hours) are only partially captured.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 1.4.2 full text: MRA + RASi + SGLT2i combination | **T1** | Drug combination guidance |
| PP 1.4.3 full text: potassium monitoring instruction | **T1** | Safety monitoring protocol |
| "10 mg daily if eGFR 25-59; 20 mg daily if eGFR ≥60" (with drug context) | **T1** | Finerenone dosing by eGFR |
| "Hold if K+ >5.5, recheck within 72 hours, resume when ≤5.0" | **T1** | Hyperkalemia management protocol |
| "SGLT2i may reduce hyperkalemia risk when combined with nonsteroidal MRA" | **T1** | Combination safety benefit |
| "Finerenone may be added to RASi alone if SGLT2i not tolerated" | **T1** | Alternative prescribing pathway |
| "FDA approved initiation: K+ <5.0 mmol/l" | **T1** | FDA threshold (differs from trial threshold) |
| "Monitor serum creatinine/eGFR concurrently with potassium" | **T2** | Monitoring completeness |

### ✅ Prior Review Activity
1/125 spans EDITED — "potassium ≤5.5 mmol/L. With serum potassium > 5.5 mmol/L" — prior reviewer engaged with the critical potassium threshold.

### ⚠️ "Potassium" ×14 — Worst C Channel Repetition Yet
With PP 1.4.3 entirely about potassium monitoring, the C channel fires on every "potassium" mention. Combined with dose fragments (×17) and "daily" (×5), the T2 tier is overwhelmed with noise.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — PP 1.4.2 and PP 1.4.3 text completely missing; Figure 9 protocol partially missing; 2 genuine T1 spans buried under 73 noise spans |
| **Tier corrections** | ~52 drug names (MRA+SGLT2i+finerenone): T1 → T3; 5 PP/Rec labels: T1 → T3; K+ ≤4.8 and K+ >5.5: T2 → T1; ~45 dose/lab/potassium fragments: T2 → T3 |
| **Missing T1** | PP 1.4.2 text, PP 1.4.3 text, finerenone dosing by eGFR, hold/resume protocol, SGLT2i hyperkalemia reduction, FDA K+ threshold |
| **Missing T2** | Concurrent creatinine/eGFR monitoring |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~10% — Dense prescribing page with 2 PPs + Figure 9 potassium protocol; 2 genuine T1 spans captured |
| **Tier accuracy** | ~4% (2/75 T1 correct + ~3/50 T2 correct = ~5/125) |
| **Noise ratio** | ~96% — Drug names ×52, potassium ×14, dose fragments ×17, "daily" ×5 |
| **Genuine T1 content** | 2 extracted (finerenone dosing+monitoring, potassium ≤5.5/>5.5 threshold) |
| **Prior review** | 1/125 EDITED |
| **Overall quality** | **POOR — ESCALATE** — PP 1.4.2/1.4.3 critical prescribing text missing despite 125 spans; Figure 9 protocol partially missing; noise ratio 96% |

---

## Cumulative MRA Section Summary (Pages 49-53)

| Page | Spans | Genuine T1 | Quality | Key Finding |
|------|-------|------------|---------|-------------|
| 49 | 90 | 1 (transplant exclusion) | POOR | Rec 1.4.1 missing; potassium ×18 |
| 50 | 32 | 3/3 (100% precision) | GOOD | Best page; B+F narrative works |
| 51 | 75 | 0 | POOR | FIDELITY HRs missing; dose fragments ×20 |
| 52 | 12 | 3 (monitoring, hyperkalemia, combination) | GOOD | B+C multi-channel works on prescribing sentences |
| 53 | 125 | 2 (dosing+monitoring, K+ threshold) | POOR | PP 1.4.2/1.4.3 missing; potassium ×14 |

**Pattern**: Low-span pages (32, 12) with narrative text produce the best quality. High-span pages (90, 75, 125) with dense prescribing content are overwhelmed by B channel drug name noise. The pipeline's signal-to-noise ratio is inversely correlated with page clinical density.

---

## Review Actions Completed

| Field | Value |
|-------|-------|
| **Review Date** | 2026-02-27 |
| **Reviewer** | claude-auditor |
| **Actions** | 5 confirmed, 120 rejected, 8 added |

### CONFIRMED Spans (5)

| Span ID | Text | Tier | Reason |
|---------|------|------|--------|
| `ae0e8c3c` | potassium ≤5.5 mmol/L. With serum potassium > 5.5 mmol/L | T1 (EDITED) | Critical potassium threshold for finerenone continue/hold decision. KB-4 safety. |
| `fae01682` | Continue finerenone 10 mg or 20 mg / Monitor K+ every 4 months | T1 | Complete drug+dose+monitoring from Figure 9. KB-1 dosing, KB-16 monitoring. |
| `21b5b999` | K+ ≤4.8 mmol/L | T2 | Finerenone initiation threshold from Figure 9. KB-4 safety prerequisite. |
| `172abb87` | K+ > 5.5 mmol/L | T2 | Finerenone hold threshold from Figure 9. KB-4 safety action trigger. |
| `1f853cf7` | eGFR 25-59 mL/min/1.73m² | T2 | Finerenone dosing eGFR range from Figure 9. KB-1 dosing criterion. |

### REJECTED Spans (120)

| Category | Count | Reject Reason |
|----------|-------|---------------|
| "MRA" (drug class name) | ~20 | out_of_scope — drug class name only |
| "SGLT2i" (drug class name) | ~18 | out_of_scope — drug class name only |
| "finerenone" (drug name) | ~14 | out_of_scope — drug name only |
| "potassium" (electrolyte name) | ~14 | out_of_scope — electrolyte name only |
| Dose fragments (10mg, 20mg, 30mg, 3mg, 300mg, 299mg, 29.9mg) | ~17 | out_of_scope — decontextualized dose/ACR numbers |
| "daily" (frequency word) | ~5 | out_of_scope — frequency word without drug context |
| "RASi" (abbreviation) | ~6 | out_of_scope — drug class abbreviation only |
| PP/Rec labels (PP 1.4.2 x2, PP 1.4.3 x3, Rec 1.3.1) | 6 | out_of_scope — label without practice point text |
| "strong recommendation" | 2 | out_of_scope — GRADE label without context |
| "ACEi", "ARB", "diuretics" | 3 | out_of_scope — drug class name only |
| "at baseline" x2, "HbA1c", "Hold", "avoid" | 5 | out_of_scope — decontextualized fragments |
| "eGFR ≥60 mL/min/1.73m²" (T1), misc dose/daily fragments | ~10 | out_of_scope — decontextualized threshold or fragment |

### ADDED Facts (8)

| # | Added Text | Target KB | Note |
|---|-----------|-----------|------|
| 1 | A nonsteroidal MRA can be added to a RASi and an SGLT2i for treatment of T2D and CKD | KB-1, KB-5 | PP 1.4.2 full text — drug combination guidance |
| 2 | Select patients with consistently normal serum potassium concentration and monitor serum potassium regularly after initiation of a nonsteroidal MRA | KB-4, KB-16 | PP 1.4.3 full text — potassium monitoring safety instruction |
| 3 | Start finerenone: 10 mg daily if eGFR 25-59; 20 mg daily if eGFR ≥60 | KB-1 | Figure 9 finerenone dosing by eGFR |
| 4 | Hold finerenone if K+ >5.5 mmol/l; recheck K+ within 72 hours; resume when K+ ≤5.0 mmol/l | KB-4 | Figure 9 hyperkalemia management protocol |
| 5 | SGLT2i may reduce the risk of hyperkalemia for patients treated concomitantly with a RASi and nonsteroidal MRA | KB-5 | PP 1.4.2 rationale — combination safety benefit |
| 6 | Finerenone may be added to RASi alone for patients who do not tolerate SGLT2i | KB-1 | PP 1.4.2 alternative prescribing pathway |
| 7 | FDA approved initiation threshold: serum potassium <5.0 mmol/l | KB-4, KB-1 | FDA threshold differs from trial threshold of ≤4.8 |
| 8 | Monitor serum creatinine and eGFR concurrently with potassium during finerenone treatment | KB-16 | Figure 9 concurrent monitoring requirement |

---

## Raw PDF Gap Analysis

| # | Gap Text | Priority | Rationale |
|---|----------|----------|-----------|
| 1 | "877 participants were using an SGLT2i at baseline, and the cardiovascular effects of finerenone, compared with placebo, appeared to be at least as beneficial among people using versus not using an SGLT2i" | **HIGH** | Key evidence for PP 1.4.2 combination therapy — N=877 subgroup + CV benefit preserved. KB-5. |
| 2 | "These data, combined with complementary mechanisms of action, suggest that the benefits of SGLT2i and finerenone may be additive" | **MODERATE** | Additive efficacy rationale justifying triple combination (RASi+SGLT2i+MRA). KB-5. |
| 3 | "the FIDELIO-DKD and FIGARO-DKD trials restricted eligibility to patients with normal serum potassium concentration (after maximizing RASi) and implemented a standardized potassium-monitoring protocol" | **MODERATE** | Critical prerequisite: maximize RASi BEFORE adding finerenone. KB-4 safety. |
| 4 | "Use of dietary potassium restriction and concomitant medications, such as diuretics and dietary potassium binders, was allowed" | **MODERATE** | Hyperkalemia mitigation strategies from trial protocol. KB-4 safety management. |
| 5 | "Increase dose to 20 mg daily, if on 10 mg daily. Restart 10 mg daily if previously held for hyperkalemia and K+ now ≤5.0 mmol/l" | **MODERATE** | Figure 9 K+ 4.9-5.5 column — dose escalation and restart protocol. KB-1 dosing. |

**All 5 gaps added via API (all 201).**

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total original spans** | 125 |
| **Confirmed** | 5 |
| **Rejected** | 120 |
| **Added (agent)** | 8 |
| **Added (gap fill)** | 5 |
| **Total Added** | 13 |
| **Total Pipeline 2 ready** | 18 (5 confirmed + 13 added) |
| **Review completeness** | 125/125 (100%) |
| **Pipeline 2 completeness** | ~93% — PP 1.4.2 full text + 877 subgroup evidence + additive benefit rationale; PP 1.4.3 full text + maximizing RASi prerequisite + mitigation strategies; Figure 9 complete (dosing, hold/resume, escalation/restart, FDA threshold, concurrent monitoring) |
| **Remaining gaps** | MRA research recommendations (T3, lower priority); finerenone event reduction specifics "particularly CKD progression and heart failure" (covered by Rec 1.4.1 on page 49) |
