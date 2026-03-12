# Page 31 Audit — Chapter 1 Comprehensive Care + Figure 1 Algorithm

| Field | Value |
|-------|-------|
| **Page** | 31 (PDF page S30) |
| **Content Type** | Chapter 1 discussion continuation + Figure 1: Kidney-heart risk factor management |
| **Extracted Spans** | 8 original + 6 REVIEWER = 14 total |
| **Channels** | B, C, F |
| **Disagreements** | 5 |
| **Review Status** | ALL REVIEWED — see Execution Log below |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw API data (8 spans), channels confirmed (B/C/F), disagreements (5), all spans reviewed via API |
| **Execution Date** | 2026-02-26 |
| **Page Decision** | FLAGGED (Figure 1 caption fully captured via 6 REVIEWER facts + 2 confirmed spans) |

---

## Source PDF Content

**Narrative text (top half):**
- Discussion of AKI risk from low eGFR or concurrent medications causing kidney hypoperfusion (diuretics)
- Sequencing of treatments: "critically important to ensure all effective and indicated treatments are implemented in an expeditious manner"
- T1D vs T2D: different pharmacologic management approaches
- GLP-1 RA recommended only in T2D population
- SGLT2i benefits in CKD with or without diabetes
- ns-MRA benefits demonstrated only in T2D with CKD
- Guideline defers T1D insulin treatment to diabetes organization guidelines

**Figure 1 — Kidney-heart risk factor management (HIGH VALUE):**
- **First-line drug therapy**: Metformin (T2D), SGLT2i (T2D), RAS blockade (HTN)
- **Additional drugs**: ns-MRA (T2D), GLP-1 RA (T2D), Statin
- **Lifestyle**: Diet, Exercise, Smoking cessation, Weight
- **Other**: Antiplatelet therapies (ASCVD), Lipid management, Blood pressure control

**Figure 1 Caption (CRITICAL T1 CONTENT):**
- "Metformin may be given when eGFR ≥30 ml/min per 1.73 m²"
- "SGLT2i should be initiated when eGFR is ≥20 ml/min per 1.73 m² and continued as tolerated, until dialysis or transplantation"
- "RAS inhibition is recommended for patients with albuminuria and hypertension"
- "A statin is recommended for all patients with T1D or T2D and CKD"
- "GLP-1 RA are preferred glucose-lowering drugs for people T2D if SGLT2i and metformin are insufficient"
- "ns-MRA can be added to first-line therapy for patients with T2D and high residual risks... persistent albuminuria (>30 mg/g [>3 mg/mmol])"

---

## Key Spans Assessment

### Tier 1 Spans (6)

| Span | Channels | Conf | Assessment |
|------|----------|------|------------|
| **"Glycemic control is based on insulin for T1D and a combination of metformin and SGLT2i for T2D..."** | B,C | 98% | **✅ T1 CORRECT** — Drug + population + treatment protocol from Figure 1 caption |
| **"risk of acute kidney injury due to low eGFR or concurrent use of medications that may contribute to kidney hypoperfusion..."** | B,C,F | 100% | **→ T2** — AKI risk discussion, narrative safety context but no specific threshold or contraindication |
| **"The GLP-1 RA are also recommended only in the T2D population."** | B,F | 98% | **✅ T1 CORRECT** — Drug class + population restriction (prescribing boundary) |
| **"The benefits of SGLT2i have been demonstrated in persons with CKD with or without diabetes."** | B,F | 98% | **→ T2/T3** — Efficacy evidence summary, not prescriptive |
| **"There is a substantial difference in the evidence base; thus, this guideline includes evidence-based recommendations for..."** | F | 85% | **→ T3** — Guideline methodology prose |
| **"However, this guideline defers pharmacologic glucose-lowering treatment of T1D, based on insulin, to existing guidelines..."** | B,F | 98% | **→ T3** — Guideline scope limitation statement |

### Tier 2 Spans (2)

| Span | Channels | Conf | Assessment |
|------|----------|------|------------|
| **`<!-- PAGE 31 -->`** | F | 90% | **⚠️ PIPELINE ARTIFACT** — HTML comment, not clinical content. Should be REJECTED |
| **"are both addressed, with differences in approach to management highlighted as appropriate."** | F | 85% | **→ T3** — Incomplete sentence fragment from narrative |

---

## Critical Findings

### ✅ Two Genuine T1 Spans
1. **Glycemic control treatment protocol** — Correctly captures metformin + SGLT2i for T2D, insulin for T1D
2. **GLP-1 RA population restriction** — T2D only, a safety-relevant prescribing boundary

### ❌ Figure 1 Caption Under-Extracted (CRITICAL GAP)
The Figure 1 caption contains **6+ distinct T1-quality clinical facts** with drug + threshold + population, but only 1 was extracted as a span. Missing:
- **"Metformin may be given when eGFR ≥30"** — T1 drug threshold
- **"SGLT2i should be initiated when eGFR is ≥20... continued until dialysis or transplantation"** — T1 drug threshold + continuation rule
- **"RAS inhibition is recommended for patients with albuminuria and hypertension"** — T1 drug indication
- **"A statin is recommended for all patients with T1D or T2D and CKD"** — T1 drug recommendation
- **"ns-MRA can be added... persistent albuminuria (>30 mg/g)"** — T1 add-on therapy threshold

### ⚠️ Pipeline Artifact
`<!-- PAGE 31 -->` is an HTML page marker that leaked through the extraction pipeline into a T2 span. This should be automatically filtered.

### ⚠️ 4 of 6 T1 Spans are Mistiered
Only 2 of 6 T1 spans meet T1 criteria. The others are narrative prose (T3), evidence summaries (T2/T3), or scope statements (T3).

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — Critical Figure 1 content severely under-extracted |
| **Tier corrections** | AKI risk narrative: T1 → T2; SGLT2i evidence: T1 → T2; Guideline prose ×2: T1 → T3; Pipeline artifact: REJECT; Fragment: T2 → T3 |
| **Missing T1** | 5+ drug thresholds from Figure 1 caption (eGFR ≥30, eGFR ≥20, albuminuria >30 mg/g, statin rec, RAS inhibition) |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~20% — Figure 1 caption has 6+ T1 facts, only 1 captured |
| **Tier accuracy** | ~25% (2/8 correctly tiered) |
| **False positive T1 rate** | 67% (4/6 T1 spans are prose/evidence, not safety) |
| **Pipeline artifacts** | 1 (`<!-- PAGE 31 -->`) |
| **Overall quality** | **POOR** — Most important content (Figure 1 algorithm) is under-extracted |

---

## Execution Log — 2026-02-26

### Phase 1: API Rejections (6 spans)

| Span ID | Text (truncated) | Reason | Note |
|---------|-------------------|--------|------|
| `c7152168` | `<!-- PAGE 31 -->` | out_of_scope | Pipeline artifact — HTML page marker, not clinical content |
| `3bff5890` | "risk of acute kidney injury due to low eGFR or concurrent use of medications..." | out_of_scope | AKI risk fragment — already captured with full context on page 30 |
| `b72c86b6` | "are both addressed, with differences in approach to management highlighted as appropriate." | out_of_scope | Incomplete sentence fragment — no standalone clinical meaning |
| `e20e0f9f` | "The benefits of SGLT2i have been demonstrated in persons with CKD with or without diabetes." | out_of_scope | Evidence summary — not prescriptive, no drug/threshold/action pattern for L3 |
| `b31283d4` | "There is a substantial difference in the evidence base; thus, this guideline includes..." | out_of_scope | Guideline methodology prose — no clinical content for L3 extraction |
| `e2487055` | "However, this guideline defers pharmacologic glucose-lowering treatment of T1D..." | out_of_scope | Guideline scope limitation — organizational statement, not prescriptive |

### Phase 2: API Confirmations (2 spans)

| Span ID | Text (truncated) | Action | Note |
|---------|-------------------|--------|------|
| `9394ad03` | "Glycemic control is based on insulin for type 1 diabetes (T1D) and a combination of metformin and SGLT2i for T2D. metformin may be given when eGFR ≥30... SGLT2i should be initiated when eGFR ≥20..." | **CONFIRM** | Best span on this page — Figure 1 caption combining T1D/T2D treatment protocol with specific eGFR thresholds |
| `20d2fdc3` | "The GLP-1 RA are also recommended only in the T2D population." | **CONFIRM** | Population restriction — safety-relevant prescribing boundary for GLP-1 RA class |

### Phase 3: Facts Added via UI (6 total)

| # | Fact Text | Note |
|---|-----------|------|
| 1 | "The benefits of nonsteroidal MRA have been demonstrated only in T2D with CKD." | Population restriction for ns-MRA class — verbatim from PDF S30 narrative. KB-4 safety relevant: ns-MRA should NOT be assumed effective in T1D+CKD. |
| 2 | "SGLT2i have not been studied in outcome trials of patients with T1D; however, studies have shown some promise, but also some risk, in this population." | SGLT2i evidence limitation in T1D — verbatim from PDF S30 narrative. KB-4 safety relevant: prescribing boundary for off-label use consideration. |
| 3 | "Renin-angiotensin system (RAS) inhibition is recommended for patients with albuminuria and hypertension (HTN). A statin is recommended for all patients with T1D or T2D and CKD." | Figure 1 caption — RAS inhibition indication + statin universal recommendation. Verbatim from PDF S30. Per-page completeness. |
| 4 | "Glucagon-like peptide-1 receptor agonists (GLP-1 RA) are preferred glucose-lowering drugs for people T2D if SGLT2i and metformin are insufficient to meet glycemic targets or if they are unable to use SGLT2i or metformin." | Figure 1 caption — GLP-1 RA prescribing hierarchy with conditions. Verbatim from PDF S30. Per-page completeness. |
| 5 | "A nonsteroidal mineralocorticoid receptor antagonist (ns-MRA) can be added to first-line therapy for patients with T2D and high residual risks of kidney disease progression and cardiovascular events, as evidenced by persistent albuminuria (>30 mg/g [>3 mg/mmol])." | Figure 1 caption — ns-MRA add-on criteria with albuminuria threshold. Verbatim from PDF S30. Per-page completeness. |
| 6 | "Aspirin generally should be used lifelong for secondary prevention among those with established cardiovascular disease and may be considered for primary prevention among patients with high risk of atherosclerotic cardiovascular disease (ASCVD)." | Figure 1 caption — Aspirin secondary + primary prevention guidance. Verbatim from PDF S30. Per-page completeness. |

### Phase 4: Page Flag

| Page | Action | Method |
|------|--------|--------|
| 31 | **FLAGGED** | Auto-flagged when facts added via UI |

---

## Post-Execution Summary

### Final Span Counts

| Metric | Count |
|--------|-------|
| **Original spans** | 8 |
| **Rejected** | 6 |
| **Confirmed** | 2 |
| **Edited** | 0 |
| **Added (REVIEWER)** | 6 |
| **Final total** | 14 |

### Pipeline 2 L3-L5 Coverage Checklist

| Clinical Concept | KB Target | Source | Status |
|------------------|-----------|--------|--------|
| T1D/T2D glycemic control protocol (insulin vs metformin+SGLT2i) | KB-1 | CONFIRMED (`9394ad03`) | ✅ |
| Metformin eGFR ≥30 threshold | KB-1, KB-16 | CONFIRMED (in `9394ad03`) | ✅ |
| SGLT2i eGFR ≥20 threshold + continuation rule | KB-1, KB-16 | CONFIRMED (in `9394ad03`) | ✅ |
| GLP-1 RA restricted to T2D population | KB-4 | CONFIRMED (`20d2fdc3`) | ✅ |
| ns-MRA demonstrated only in T2D with CKD | KB-4 | ADDED | ✅ |
| SGLT2i not studied in T1D outcome trials | KB-4 | ADDED | ✅ |
| RAS inhibition for albuminuria + HTN | KB-1 | ADDED (Fact 3) | ✅ |
| Statin for all T1D/T2D + CKD patients | KB-1 | ADDED (Fact 3) | ✅ |
| GLP-1 RA preferred if SGLT2i/metformin insufficient | KB-1 | ADDED (Fact 4) | ✅ |
| ns-MRA add-on criteria (albuminuria >30 mg/g) | KB-1 | ADDED (Fact 5) | ✅ |
| Aspirin secondary/primary prevention | KB-4 | ADDED (Fact 6) | ✅ |

### Post-Execution Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | **~95%** (was ~20%) — All Figure 1 caption prescriptive facts captured on this page via 2 confirmed spans + 6 REVIEWER facts. Full per-page completeness achieved. |
| **Noise removal** | 75% rejected (6/8) |
| **Overall quality** | **HIGH** (was POOR) — All prescriptive content from Figure 1 caption and narrative now captured directly on page 31. Per-page completeness principle applied. |

### Key Observations

1. **Confirmed span `9394ad03` is excellent**: Captures the first 2 sentences of the Figure 1 caption with T1D/T2D protocol and specific eGFR thresholds (≥30 metformin, ≥20 SGLT2i). One of the best single spans in the entire job.

2. **Full per-page completeness achieved**: All Figure 1 caption prescriptive facts now added directly on page 31 as REVIEWER facts (3-6), in addition to the 2 unique narrative facts (1-2). Per-page completeness principle: every prescriptive fact on a page must be captured on THAT page; Pipeline 2 dedup handles cross-page references later.

3. **Two unique narrative facts added**: The ns-MRA population restriction ("only in T2D with CKD") and SGLT2i T1D evidence gap are safety-relevant prescribing boundaries not extracted by any pipeline channel. Critical for KB-4 patient safety.

4. **75% noise rate**: 6 of 8 original spans were noise (artifact, fragments, methodology prose, evidence summaries). Only 2 original spans contained genuine prescriptive content — consistent with Channel F (LLM) over-extracting narrative context.
