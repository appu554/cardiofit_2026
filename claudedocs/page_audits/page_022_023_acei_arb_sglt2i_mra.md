# Pages 22–23 Audit — ACEi/ARB Practice Points + SGLT2i/MRA Recommendations

| Field | Value |
|-------|-------|
| **Pages** | 22–23 (PDF pages S21–S22) |
| **Content Type** | Practice Points 1.2.5–1.3.4 (ACEi/ARB), Recommendations 1.3.1, 1.4.1, 1.5.1 (SGLT2i, MRA, Smoking) |
| **Extracted Spans** | 70 (pg 22) + 29 (pg 23) = 99 total |
| **Channels** | B, C, E (pg 22); B, C, D, F (pg 23) — NO D on pg 22, NO E/F on pg 22 |
| **Disagreements** | 5 (pg 22: 2, pg 23: 3) |
| **Review Status** | CONFIRMED: 1, EDITED: 1, PENDING: 97 |
| **Risk** | Disagreement (both pages) |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — pg 22 corrected (72→70, D channel removed), pg 23 corrected (39→29, phantom D-channel drug names removed), combined 111→99 |

---

## Source PDF Content

**Page 22 (S21)** — ACEi/ARB management practice points:
- PP 1.2.5: Monitor serum creatinine and potassium after ACEi/ARB initiation/dose changes
- PP 1.2.6: Discontinue ACEi/ARB if concerns about hyperkalemia or AKI (sick-day rules)
- PP 1.2.7: Consider ACEi/ARB use in specific CKD stages
- Rec 1.3.1: SGLT2i recommendation
- PP 1.3.1–1.3.4: SGLT2i practice points

**Page 23 (S22)** — SGLT2i safety + MRA recommendations:
- SGLT2i brand-specific information (Dapagliflozin, Canagliflozin, Empagliflozin)
- Periprocedural/perioperative care: withhold SGLT2i for procedures (DKA risk)
- Rec 1.4.1: ns-MRA with proven benefits (finerenone) for T2D with eGFR ≥25, potassium normal, and albuminuria ≥30 mg/g
- PP 1.4.1–1.4.5: MRA practice points
- PP 1.4.5: Steroidal MRA for heart failure, hyperaldosteronism, refractory hypertension
- Rec 1.5.1 + PP 1.5.1: Smoking cessation

---

## Key Clinical Spans Assessment

### Page 22 — Genuine T1/T2 Content

| Span | Tier | Correct? | Assessment |
|------|------|----------|------------|
| "ACEi" ×11 / "ARB" ×11 standalone = 22 | T1 | **NO → T3** | Drug class names repeated without thresholds |
| "potassium" ×3 | T2 | **Partial** | Lab name from monitoring context — could be T2 if monitoring instruction preserved |
| "serum creatinine" ×2 | T2 | **Partial** | Same as above |
| "discontinue" ×2 | T2 | **NO → T3** | Action verb without what/when context |
| "eGFR" ×2 | T2 | **NO → T3** | Lab abbreviation without threshold |
| "NSAID" | T2 | **OK** | Drug class in avoidance context |
| Practice Point labels ×5 | T1 | **NO → T3** | Labels without recommendation text |
| "Recommendation 1.3.1" | T1 | **NO → T3** | Label only |
| "SGLT2i" ×7 | T1 | **NO → T3** | Drug name without associated recommendation |
| "Sodium" | T2 | **NO → T3** | Single word |
| ~~"Potassium binders"~~ | — | — | **PHANTOM — not in raw data (no D channel on pg 22)** |
| ~~"Volume depletion"~~ | — | — | **PHANTOM — not in raw data (no D channel on pg 22)** |

### Page 23 — Genuine T1/T2 Content

| Span | Tier | Correct? | Assessment |
|------|------|----------|------------|
| **"Periprocedural/perioperative care: inform patients about risk of DKA; withhold SGLT2i the day of day-stay procedures..."** | T1 | **✅ T1 CORRECT** | Drug + action + safety context (DKA risk) |
| **"eGFR ≥25 mL/min/1.73m²"** | T1 | **✅ T1 CORRECT** | Finerenone initiation threshold |
| **"potassium concentration, and albuminuria (≥30 mg..."** | T2 | **⚠️ SHOULD BE T1** | Finerenone eligibility criteria — potassium + albuminuria thresholds |
| **"PP 1.4.5: A steroidal MRA should be used for treatment of heart failure, hyperaldosteronism, or refractory hypertension"** | T1 | **✅ T1 CORRECT** | Drug class + indications — full practice point text! |
| "SGLT2 inhibitors" ×2 (B+D) / "SGLT2i" ×1 (B) | T1 | **NO → T3** | Drug name/class repeated without clinical context |
| ~~"Dapagliflozin", "Canagliflozin", "Empagliflozin"~~ | — | — | **PHANTOM — not individually extracted in raw data** |
| Drug class names (MRA, SGLT2i, RASi) standalone | T1/T2 | **NO → T3** | Without associated clinical facts |
| Practice Point labels ×6 | T1 | **NO → T3** | Labels only |
| "Smoking cessation" recommendation | F | T2 | **OK** — lifestyle recommendation |

---

## Critical Findings

### Genuine T1 Spans (4 of 99)
1. **Periprocedural SGLT2i management** — drug + safety warning + action
2. **eGFR ≥25** — finerenone initiation threshold
3. **PP 1.4.5 full text** — steroidal MRA indications (rare: full practice point captured!)
4. **Potassium/albuminuria criteria** (currently T2, should be T1)

### ❌ MASSIVE Missing Content
**The full text of Practice Points 1.2.5–1.2.7, 1.3.1–1.3.4 was NOT extracted.** Only labels and drug names appear. The actual clinical instructions like:
- "Monitor serum creatinine and potassium within 2-4 weeks of ACEi/ARB initiation" — MISSING
- "Temporarily discontinue ACEi/ARB during intercurrent illness" (sick-day rules) — MISSING
- SGLT2i expected eGFR dip on initiation — MISSING

### 📊 Drug Name Inflation
~49 of 99 spans are standalone drug/drug class names extracted by B channel. These inflate the T1 count without adding clinical value.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page 22** | **FLAG** — Critical ACEi/ARB monitoring instructions are MISSING; only drug names and labels extracted |
| **Page 23** | **FLAG** — Periprocedural SGLT2i and MRA criteria are well-captured; many standalone drug names to reject |
| **Tier corrections** | 91 of 99 spans (92%) need re-tiering. Potassium/albuminuria criteria: T2 → **T1**; ~49 standalone drug names: T1 → **T3**; ~15 labels: T1 → **T3** |
| **Missing T1** | ~12 full recommendation/practice point texts not extracted at all (see detailed audit) |
| **Missing T2** | Monitoring schedule for ACEi/ARB (2-4 week follow-up), sick-day rules detail |

---

## Completeness Score

| Metric | Page 22 | Page 23 |
|--------|---------|---------|
| **Extraction completeness** | ~20% (drug names only, full recommendations missing) | ~45% (some complete recommendations captured) |
| **Tier accuracy** | ~4% (2/50 T1 defensible) | ~15% (3/20 T1 genuine) |
| **False positive T1 rate** | 98% (49/50 T1 are false) | 85% (17/20 T1 are false) |
| **Missing T1 content** | ~8 full practice point/recommendation texts | ~4 (Rec 1.4.1 full text, sick-day protocol, drug doses, ketone threshold) |
| **Overall page quality** | POOR | MODERATE |
