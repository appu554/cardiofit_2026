# Page 82 Audit — PP 4.1.4 B12 Cont, Research Recs (Metformin in CKD), Section 4.2 GLP-1 RA, Rec 4.2.1 (Long-Acting GLP-1 RA, 1B), Figure 27 (Metformin Dosing Algorithm)

| Field | Value |
|-------|-------|
| **Page** | 82 (PDF page S81) |
| **Content Type** | PP 4.1.4 continuation (B12 deficiency clinical consequences uncommon, no routine supplementation, increased reduction with longer metformin therapy, monitoring after >4 years or risk factors), Research Recommendations ×2 (metformin safety/efficacy in CKD including eGFR <30/dialysis; metformin in kidney transplant), Section 4.2 GLP-1 RA (incretin mechanism: insulin stimulation + glucagon suppression + gastric slowing + weight loss, glycemic control + weight loss + MACE reduction + albuminuria reduction + eGFR decline slowing), Rec 4.2.1 (long-acting GLP-1 RA for T2D+CKD when metformin+SGLT2i insufficient, 1B), Figure 27 (metformin dosing algorithm by eGFR: initiation, titration, eGFR-based adjustments, monitoring frequency) |
| **Extracted Spans** | 21 total (18 T1, 3 T2) |
| **Channels** | B (Drug Dictionary), C (Grammar/Regex), F (NuExtract LLM) |
| **Disagreements** | 0 |
| **Review Status** | PENDING: 21 |
| **Risk** | Clean |
| **Cross-Check** | Counts verified against raw extraction data |
| **Audit Date** | 2026-02-25 (revised) |

---

## Source PDF Content

**PP 4.1.4 Continuation (Vitamin B12):**
- "Clinical consequences of vitamin B12 deficiency with metformin treatment are uncommon"
- Work Group judgment: "routine concurrent supplementation with vitamin B12 is unnecessary"
- B12 reduction increases with increasing duration of metformin therapy
- **Monitoring recommendation**: Consider B12 monitoring in patients on long-term metformin (>4 years) OR at risk of low B12 (malabsorption syndrome, vegans)

**Research Recommendations (Metformin):**
1. **RCTs needed**: Evaluate safety, efficacy, and potential CV and kidney protective benefits of metformin in T2D+CKD, **including those with eGFR <30 or on dialysis**
2. **RCTs needed**: Evaluate safety and efficacy of metformin in **kidney transplant recipients**

**Section 4.2 — GLP-1 Receptor Agonists:**
- GLP-1 = incretin hormone secreted from intestine after glucose/nutrient ingestion
- Mechanism: stimulates glucose-dependent insulin release (beta cells) + suppresses glucagon (alpha cells) + slows gastric emptying + decreases appetite → weight loss
- Incretin effects reduced/absent in diabetes patients
- Long-acting GLP-1 RA: substantially improve glycemic control + confer weight loss
- **MACE reduction** in T2D patients with HbA1c >7.0% at high CV risk
- **Kidney benefits**: reducing albuminuria + slowing eGFR decline

**Recommendation 4.2.1 (1B — Strong/Moderate):**
- "In patients with T2D and CKD who have not achieved individualized glycemic targets despite use of metformin and SGLT2i treatment, or who are unable to use those medications, we recommend a long-acting GLP-1 RA"
- High value on: CV and kidney benefits of long-acting GLP-1 RA
- Lower value on: costs and adverse effects

**Figure 27 — Metformin Dosing Algorithm by Kidney Function:**

| eGFR Range | Action |
|-----------|--------|
| **Dose Initiation** | |
| eGFR ≥60 | IR: Initial 500 mg or 850 mg once daily, titrate by 500 mg/d or 850 mg/d every 7 days to max; ER: Initial 500 mg daily, titrate by 500 mg/d every 7 days to max |
| eGFR 30–44 | Initiate at half the dose, titrate to half of maximum recommended dose |
| eGFR <30 | **Stop metformin; do not initiate metformin** |
| **Subsequent Dose Adjustment** | |
| eGFR ≥60 | Continue same dose |
| eGFR 45–59 | Continue same dose. Consider dose reduction in certain conditions (hypoperfusion/hypoxemia) |
| eGFR 30–44 | Halve the dose |
| **Monitoring** | |
| eGFR ≥60 | At least annually |
| eGFR <60 | At least every 3–6 months |
| **B12 Monitoring** | Monitor vitamin B12 |

---

## Key Spans Assessment

### Tier 1 Spans (18)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "GLP-1 RA" ×8 (B) | B | 100% | **→ T3** — Drug class abbreviation mentions (8 separate spans). No clinical context |
| "GLP-1 receptor agonists" ×1 (B) | B | 100% | **→ T3** — Full drug class name, section header text |
| "metformin" ×5 (B) | B | 100% | **→ T3** — Drug name mentions across PP 4.1.4, research recs, and Rec 4.2.1 |
| "insulin" ×1 (B) | B | 100% | **→ T3** — Drug name in GLP-1 mechanism description |
| "SGLT2i" ×1 (B) | B | 100% | **→ T3** — Drug class name in Rec 4.2.1 context |
| "eGFR <30" (C) | C | 95% | **→ T3** — Standalone threshold from research rec. Without the sentence "Evaluate safety...including those with eGFR <30" it's a bare number |
| "Recommendation 4.2.1" (C) | C | 98% | **→ T3** — Rec label only. The actual recommendation text about GLP-1 RA NOT captured separately |

**Summary: 0/18 T1 genuine patient safety content. 16 are standalone drug name mentions → T3. 1 is eGFR threshold fragment → T3. 1 is Rec label → T3. ALL 18 T1 SPANS ARE T3.**

### Tier 2 Spans (3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "RCTs are needed to:" (F) | F | 90% | **→ T3** — Incomplete sentence fragment. The F channel captured the heading but not the actual research recommendations that follow (metformin in CKD eGFR <30, metformin in transplant) |
| "HbA1c" (C) | C | 85% | **→ NOISE** — Standalone lab name, same Ch2 noise pattern |
| "eGFR" (C) | C | 85% | **→ NOISE** — Bare abbreviation without threshold |

**Summary: 0/3 T2 correct. 1 is incomplete F sentence heading → T3. 2 are single-word noise.**

---

## Critical Findings

### 🚨 0/21 SPANS CORRECTLY TIERED — 100% Misclassification

This is only the second page in the audit (after page 79) where **zero spans are correctly tiered**. Every single T1 span is a drug name or threshold fragment (T3), and every T2 span is noise. Despite having clinically rich content (Rec 4.2.1, Figure 27, PP 4.1.4, research recs), the pipeline extracted only labels and drug mentions.

### ❌ Rec 4.2.1 Text NOT CAPTURED

"In patients with T2D and CKD who have not achieved individualized glycemic targets despite use of metformin and SGLT2i treatment, or who are unable to use those medications, we recommend a long-acting GLP-1 RA" — this is a **1B formal recommendation** (Strong/Moderate evidence) and the core prescribing directive for GLP-1 RA. The C channel captured only the label "Recommendation 4.2.1" without the recommendation text.

This is the same pattern seen throughout the audit: C fires on "Recommendation X.Y.Z" labels but does not capture the actual recommendation sentence.

### ❌ Figure 27 Dosing Algorithm NOT EXTRACTED

Figure 27 is the critical prescriber tool for metformin dose management by eGFR. It contains:
- **Stop metformin at eGFR <30** (patient safety)
- Halve dose at eGFR 30–44 (dosing guidance)
- Initiation at half dose for eGFR 30–44 (dosing guidance)
- IR vs ER titration schedules (clinical accuracy)
- Monitoring frequency by eGFR (monitoring)

The figure text IS visible in the PDF viewer (the snapshot shows the full algorithm text), but no channel extracted it. The D channel did not fire (consistent with D being silent on algorithmic/flowchart figures), and no other channel processed the figure content.

### ❌ GLP-1 RA Mechanism and Benefits NOT CAPTURED

Section 4.2's description of GLP-1 RA mechanism (incretin pathway, insulin stimulation, glucagon suppression, gastric slowing, weight loss) and clinical benefits (MACE reduction, albuminuria reduction, eGFR decline slowing) — none captured. This is evidence/rationale content (T2) that the F channel should excel at, but F only captured "RCTs are needed to:" (a heading fragment).

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| Rec 4.2.1: Long-acting GLP-1 RA when metformin+SGLT2i insufficient (1B) | **T1** | Formal recommendation with evidence grade |
| Figure 27: Stop metformin at eGFR <30, do not initiate | **T1** | Patient safety: drug discontinuation threshold |
| Figure 27: Halve dose at eGFR 30–44, initiate at half dose | **T1** | Dosing modification for renal impairment |
| Figure 27: Consider dose reduction at eGFR 45–59 with hypoperfusion/hypoxemia risk | **T1** | Conditional dose modification |
| PP 4.1.4 closing: Routine B12 supplementation unnecessary; monitor after >4 years | **T2** | Monitoring guidance (modifies the B12 concern) |
| Research rec: Evaluate metformin safety in eGFR <30/dialysis | **T2** | Evidence gap identification |
| Research rec: Evaluate metformin in kidney transplant | **T2** | Evidence gap identification |
| GLP-1 RA mechanism: incretin pathway, MACE reduction, albuminuria/eGFR benefits | **T2** | Drug class rationale for Rec 4.2.1 |
| GLP-1 RA: benefits in HbA1c >7.0% with high CV risk | **T2** | Target population for GLP-1 RA |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — 21 spans with 0% correct tiering; all T1 are drug names/labels (T3); Rec 4.2.1 text missing; Figure 27 dosing algorithm missing; GLP-1 RA section not captured |
| **Tier corrections** | All 16 B drug names: T1 → T3; "eGFR <30": T1 → T3; "Recommendation 4.2.1": T1 → T3; "RCTs are needed to:": T2 → T3; "HbA1c": T2 → NOISE; "eGFR": T2 → NOISE |
| **Missing T1** | Rec 4.2.1 full text, Figure 27 dosing algorithm (especially eGFR <30 discontinuation and eGFR 30–44 dose halving) |
| **Missing T2** | PP 4.1.4 closing, research recs, GLP-1 RA mechanism/benefits |

---

## Completeness Score (Pre-Review)

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~5% — Only drug names and labels extracted; all clinical content (Rec 4.2.1, Figure 27, GLP-1 RA benefits, PP 4.1.4 guidance) missing |
| **Tier accuracy** | 0% (0/18 T1 correct + 0/3 T2 correct = 0/21) |
| **Noise ratio** | 100% — Every span is a drug name, label, or bare abbreviation without clinical context |
| **Genuine T1 content** | 0 extracted |
| **Prior review** | 0/21 reviewed |
| **Overall quality** | **VERY POOR — FLAG** — Zero clinical content extracted from a page with a formal 1B recommendation, a complete dosing algorithm, and a new drug class section |

---

## Raw PDF Cross-Check (2026-02-28)

Cross-checked 12 ADDED spans against exact KDIGO PDF text (page S81). Found **4 missing gaps** — Rec 4.2.1 rationale, B12 duration-dependent effect, B12 monitoring exact text with risk groups, and Rec 4.2.1 with evidence grade (1B).

### Gaps Added (4)

| Gap | Priority | Content | KB Target |
|-----|----------|---------|-----------|
| G82-A | MEDIUM | Rec 4.2.1 rationale: "places a high value on the cardiovascular and kidney benefits... and a lower value on the costs and adverse effects" | KB-1 |
| G82-B | MEDIUM | B12 duration-dependent: "reduction in vitamin B12 concentration is increased with increasing duration of metformin therapy" | KB-16 |
| G82-C | HIGH | B12 monitoring exact text: "patients who have been on long-term metformin treatment (e.g., >4 years) or in those who are at risk of low vitamin B12 levels (e.g., patients with malabsorption syndrome, or reduced dietary intake [vegans])" | KB-16/KB-4 |
| G82-D | HIGH | Rec 4.2.1 complete with evidence grade: "we recommend a long-acting GLP-1 RA (1B)" | KB-1 |

All 4 gaps added via API (all 201 success).

---

## Post-Review State (Final — with raw PDF cross-check)

| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 21 | All original noise spans (bare drug names, labels, abbreviations) |
| **ADDED** | 16 | 12 agent-added + 4 cross-check gaps |
| **Total spans** | 37 | 33 + 4 new |
| **P2-ready** | 16 | All ADDED |
| **Reviewer** | claude-auditor | |
| **Review Date** | 2026-02-28 | |

### Updated Completeness Score (Final)

| Metric | Pre-Review | Final (2/28) |
|--------|-----------|--------------|
| **Total spans** | 21 | 37 |
| **P2-ready** | 0 | 16 |
| **Extraction completeness** | ~5% | **~95%** — Rec 4.2.1 with (1B) grade, Figure 27 dosing algorithm (all eGFR tiers + initiation + monitoring), B12 monitoring with risk groups, GLP-1 RA mechanism/benefits, research recs |
| **Noise ratio** | 100% | 0% active (21 rejected) |
| **Overall quality** | VERY POOR — FLAG | **EXCELLENT** — all critical prescribing content captured including evidence grade, B12 duration-dependent effect, and exact monitoring criteria |
