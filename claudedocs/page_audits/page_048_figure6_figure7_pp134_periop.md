# Page 48 Audit — Figure 6 (SGLT2i Initiation), Figure 7 (FDA Doses), PP 1.3.4, Perioperative Care

| Field | Value |
|-------|-------|
| **Page** | 48 (PDF page S47) |
| **Content Type** | PP 1.3.4 (volume depletion/diuretic management) + Figure 6 (practical SGLT2i initiation flowchart) + Figure 7 (FDA-approved SGLT2i doses/eGFR adjustments) + Perioperative/sick day protocol |
| **Extracted Spans** | 59 total (42 T1, 17 T2) |
| **Channels** | B, C, D, F |
| **Disagreements** | 6 |
| **Review Status** | PENDING: 59 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — count corrected (61→59), tiers corrected (T1: 41→42, T2: 20→17), disagreements added (6) |

---

## Source PDF Content

**Practice Point 1.3.4:**
- "If a patient is at risk for hypovolemia, consider decreasing thiazide or loop diuretic dosages before commencement of SGLT2i treatment"
- Advise patients about symptoms of volume depletion and low blood pressure
- Follow up on volume status after drug initiation
- SGLT2i cause initial natriuresis with weight reduction
- Despite theoretical concern, AKI incidence is DECREASED with SGLT2i vs placebo
- "Caution is prudent when initiating an SGLT2i in patients with tenuous volume status and at high risk of AKI"

**Figure 6 — Practical SGLT2i Initiation Flowchart:**
- **Patient Selection**: eGFR ≥20 ml/min/1.73 m²; high priority: ACR ≥200 mg/g, heart failure
- **Potential Contraindications**: Genital infection risk, DKA, foot ulcers, immunosuppression
- **Assessment — Glycemia**: Hypoglycemia risk with insulin/sulfonylurea → consider dose reduction
- **Assessment — Volume**: Volume depletion risk with concurrent diuretics → consider diuretic dose reduction
- **Intervention**: SGLT2i with proven benefits: canagliflozin 100mg, dapagliflozin 10mg, empagliflozin 10mg
- **Education**: Sick day protocol, perioperative care, foot care
- **Follow-up**: Anticipate acute eGFR drop (not a reason to stop); reassess volume; reduce concomitant diuretic if needed

**Figure 7 — FDA-Approved SGLT2i Doses:**

| SGLT2i | Dose | FDA-Approved eGFR | Trial Enrollment eGFR |
|--------|------|----|----|
| Dapagliflozin | 10 mg daily | eGFR ≥25 | DAPA-CKD ≥25; DAPA-HF+DECLARE ≥30 |
| Empagliflozin | 10 mg daily (can ↑ 25mg for glucose) | eGFR ≥30 (T2D/ASCVD); eGFR ≥20 (HF) | EMPA-REG ≥30; EMPEROR ≥20 |
| Canagliflozin | 100 mg daily (300mg NOT for CKD) | eGFR ≥30 | CREDENCE ≥30 |

**Perioperative/Sick Day Protocol:**
- Sick day: temporarily withhold SGLT2i, keep eating/drinking, check glucose + ketones more often
- Perioperative: withhold SGLT2i day of day-stay procedures; withhold ≥2 days before major surgery; measure glucose + ketones on admission; restart only when eating/drinking normally

---

## Key Spans Assessment

### Tier 1 Spans (41)

| Category | Count | Assessment |
|----------|-------|------------|
| **D channel — Figure 7 table rows** (eGFR thresholds per drug) | 6 | **✅ T1 CORRECT** — Genuine prescribing data: drug-specific eGFR thresholds from FDA dosing table |
| **B+F multi-channel — Perioperative care protocol** | 1 | **✅ T1 CORRECT** — "withhold SGLT2i the day of day-stay procedures..." — critical perioperative prescribing instruction |
| **B+F multi-channel — Volume depletion/AKI concern** | 1 | **✅ T1 CORRECT** — "concern for volume depletion and AKI, particularly among patients treated concurrently with diuretics" |
| **B+F multi-channel — AKI decreased with SGLT2i** | 1 | **✅ T1 CORRECT** — "the incidence of AKI is decreased with SGLT2i, compared with placebo" — safety finding |
| **B+F multi-channel — Caution for tenuous volume** | 1 | **✅ T1 CORRECT** — "caution is prudent when initiating an SGLT2i in patients with tenuous volume status and at high risk of AKI" |
| **B+F multi-channel — Practical provider guide title** | 1 | **→ T3** — Figure title, not prescribing content |
| **"diuretics"/"diuretic"/"Diuretics"** (B channel) | 8 | **ALL → T3** — Drug class name only, no context |
| **"loop diuretic"** (B channel) | 1 | **→ T2** — Drug class with specificity (from PP 1.3.4) but no full sentence |
| **"SGLT2 inhibitors"/"SGLT2 inhibitor"/"SGLT2i"** (B channel) | 4 | **ALL → T3** — Drug class name only |
| **"dapagliflozin"** (B channel) | 2 | **ALL → T3** — Drug name only |
| **"empagliflozin"** (B channel) | 1 | **→ T3** — Drug name only |
| **"canagliflozin"** (B channel) | 1 | **→ T3** — Drug name only |
| **"Practice Point 1.3.4"** (C channel) | 1 | **→ T3** — PP label only (actual PP text NOT captured) |
| **eGFR thresholds** (C channel) | ~10 | **→ T2** — ≥20 ×2, ≥25 ×2, ≥30 ×5 — decontextualized thresholds |
| **"not recommended"** (C channel) | 1 | **→ T2** — Partial phrase from canagliflozin 300mg CKD warning |

**Summary: 10/41 T1 spans are genuine (6 D channel table rows + 4 B+F multi-channel clinical sentences). 31/41 T1 are drug names, PP labels, or eGFR fragments.**

### Tier 2 Spans (20)

| Category | Count | Assessment |
|----------|-------|------------|
| **D channel — Figure 7 dosing rows** | 5 | **MIXED** — "10 mg daily" ✅ T2, "Dosing approved by US FDA" → T3 header, "100 mg daily (300mg not recommended for CKD)" **⚠️ SHOULD BE T1**, "10 mg daily (can increase to 25mg)" **⚠️ SHOULD BE T1**, "Kidney function eligible..." → T3 header |
| **"hemoglobin"** (C channel) | 2 | **→ T3** — Lab name only |
| **"creatinine"** (C channel) | 1 | **→ T3** — Lab name only |
| **"thiazide"** (B channel) | 1 | **→ T3** — Drug class name only |
| **"eGFR Ω mL/min/1.73m2"** (F channel) | 1 | **→ T3** — Corrupted OCR artifact (Ω character) |
| **Dose fragments**: "10 mg" ×2, "25 mg", "100 mg", "300 mg" (C channel) | 5 | **ALL → T3** — Decontextualized dose numbers |
| **"daily"** (C channel) | 4 | **ALL → T3** — Frequency word without drug context |

---

## Critical Findings

### ✅ BEST PAGE SINCE PAGE 40 — D Channel Table Decomposition Works on Figure 7

For the first time since the Figure 5 drug dosing table (page 40), the D channel successfully decomposes a structured figure into meaningful clinical data rows. The 6 eGFR threshold spans from Figure 7 contain genuine prescribing information:
- "eGFR ≥30 ml/min per 1.73 m² for T2D and ASCVD for glucose control; eGFR ≥20 ml/min per 1.73 m² for HF" (empagliflozin)
- "eGFR ≥25 ml/min per 1.73 m² in DAPA-CKD; eGFR ≥30 ml/min per 1.73 m² in DAPA-HF" (dapagliflozin)
- "eGFR ≥30 ml/min per 1.73 m² in CREDENCE" (canagliflozin)

### ✅ B+F Multi-Channel Captures 4 Clinical Sentences

The B+F combination captures the most important clinical sentences:
1. Perioperative care protocol (withhold SGLT2i for surgery)
2. Volume depletion/AKI concern with diuretics
3. AKI actually decreased with SGLT2i vs placebo
4. Caution for tenuous volume status patients

### ⚠️ Two Dosing Spans Should Be T1 (Currently T2)
- "100 mg daily (The higher dose of 300 mg is not recommended for CKD)" — canagliflozin dose + CKD restriction = T1
- "10 mg daily (Can increase to 25 mg daily if needed for glucose control)" — empagliflozin dose + titration guidance = T1

### ❌ PP 1.3.4 Full Text NOT EXTRACTED (Same Pattern as PP 1.3.1-1.3.3)
"If a patient is at risk for hypovolemia, consider decreasing thiazide or loop diuretic dosages before commencement of SGLT2i treatment" — the actual practice point text is missing. Only the label "Practice Point 1.3.4" is captured.

### ❌ Figure 6 Flowchart Content Partially Missing

| Missing from Figure 6 | Tier | Clinical Importance |
|------------------------|------|---------------------|
| "Eligible patients: eGFR ≥20 ml/min/1.73 m²" | **T1** | Patient selection criterion |
| "High priority: ACR ≥200 mg/g; Heart failure" | **T1** | Prioritization guidance |
| "Potential contraindications: genital infection risk, DKA, foot ulcers, immunosuppression" | **T1** | Contraindication list |
| "SGLT2i with proven benefits: canagliflozin 100mg, dapagliflozin 10mg, empagliflozin 10mg" | **T1** | Recommended agents + doses |
| "Anticipate acute eGFR drop — not a reason to stop" | **T1** | Expected initial eGFR dip (critical for prescriber education) |
| Sick day protocol details | **T1** | Safety protocol |

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 1.3.4 full text (diuretic dose reduction before SGLT2i) | **T1** | Practice point prescribing instruction |
| Figure 6 patient selection criteria | **T1** | Initiation eligibility |
| Figure 6 contraindication list | **T1** | Safety screening |
| Figure 6 "anticipate eGFR drop — not a reason to stop" | **T1** | Prescriber reassurance (prevents inappropriate discontinuation) |
| Sick day protocol (temporarily withhold, check ketones) | **T1** | Safety protocol |
| Drug names linked to doses in Figure 7 (dapagliflozin 10mg, empagliflozin 10/25mg, canagliflozin 100mg) | **T1** | Complete drug-dose pairs |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — Good D channel table extraction + B+F clinical sentences, but PP 1.3.4 text missing + Figure 6 flowchart content partially missing |
| **Tier corrections** | 8 "diuretics" B channel: T1 → T3; 4 SGLT2i names: T1 → T3; PP 1.3.4 label: T1 → T3; canagliflozin 100mg dose + empagliflozin titration: T2 → T1; 5 dose fragments + 4 "daily" + 2 "hemoglobin" + 1 "creatinine": T2 → T3 |
| **Missing T1** | PP 1.3.4 text, Figure 6 eligibility/contraindications/sick day protocol, "eGFR drop not a reason to stop" |
| **Upgrades** | "100mg daily (300mg not recommended for CKD)": T2 → T1; "10mg daily (can increase to 25mg)": T2 → T1 |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~40% — Best so far in Chapter 1; D channel table + B+F clinical sentences capture meaningful content |
| **Tier accuracy** | ~24% (10/41 T1 correctly tiered + 1/20 T2 correctly tiered) |
| **False positive T1 rate** | 76% (31/41 T1 are drug names, labels, or thresholds without context) |
| **Genuine T1 content** | 10 extracted (6 D channel dosing rows + 4 B+F safety sentences) — BEST PAGE |
| **Overall quality** | **FAIR-GOOD** — Significant genuine content captured via D+F channels; Figure 6 flowchart and PP 1.3.4 text still missing |

---

## D Channel Success Pattern

Page 48 demonstrates when D channel table decomposition succeeds:
- **Figure 7** = simple 3-column table (Drug | Dose | eGFR threshold) → D channel produces meaningful rows
- **Contrast with Figure 5 (pages 40-41)**: Complex multi-column trial summary → D channel produces column headers and fragments
- **Key insight**: D channel works best on simple drug-dose-threshold tables (the most clinically important table type) but fails on complex evidence/trial summary tables

---

## Review Actions Completed (2026-02-27)

### API Actions (reviewer: claude-auditor)

| Action | Count | Details |
|--------|-------|---------|
| **CONFIRM** | 11 | 5 D channel Figure 7 eGFR rows (empagliflozin ×2, dapagliflozin ×2, canagliflozin ×1) + 2 D channel dosing rows (canagliflozin 100mg/300mg CKD, empagliflozin 10/25mg) + 4 B+F clinical sentences (perioperative, volume/AKI concern, AKI decreased, caution tenuous volume) |
| **REJECT** | 50 | 10× diuretic/Diuretics, 4× SGLT2i/SGLT2 inhibitor, 4× daily, 4× eGFR ≥30, 2× dapagliflozin, 2× hemoglobin, 2× eGFR ≥25, 2× eGFR ≥20, 2× 10mg, plus drug names, dose fragments, PP label, table headers, OCR artifact |

### API-Added Facts (7 REVIEWER spans)

| # | Fact Added | Target KB | L3 Extraction Value |
|---|-----------|-----------|---------------------|
| 1 | PP 1.3.4: "If at risk for hypovolemia, consider decreasing thiazide or loop diuretic dosages before SGLT2i; advise about volume depletion symptoms" | KB-1, KB-4, KB-5 | drug_class=SGLT2i, co_prescribing=diuretic_reduction, patient_education=volume_depletion |
| 2 | Figure 6: Eligible eGFR ≥20; high priority ACR ≥200 or HF; contraindications: genital infection, DKA, foot ulcers, immunosuppression | KB-1, KB-4 | eligibility=eGFR≥20, priority=ACR≥200+HF, contraindications=4 |
| 3 | Figure 6: SGLT2i with proven benefits: canagliflozin 100mg, dapagliflozin 10mg, empagliflozin 10mg | KB-1 | drug_dose_pairs=3_agents |
| 4 | Figure 6: Anticipate acute eGFR drop — not a reason to discontinue; reassess volume; reduce diuretic if needed | KB-4, KB-16 | expected_eGFR_dip=normal, monitoring=volume_status |
| 5 | Sick day + perioperative protocol: withhold SGLT2i; day-stay = day of; major surgery = ≥2 days before; check glucose+ketones; restart when eating normally | KB-4 | sick_day_rules=complete_protocol, perioperative=tiered |
| 6 | Figure 6: Pre-initiation assessment — hypoglycemia risk with insulin/SU → dose reduction; volume risk with diuretics → diuretic reduction | KB-1, KB-5 | pre_initiation=glycemia+volume_assessment |
| 7 | Figure 7: Dapagliflozin 10mg daily, FDA eGFR ≥25; DAPA-CKD ≥25, DAPA-HF+DECLARE ≥30 | KB-1 | drug=dapagliflozin, dose=10mg, eGFR_threshold=25 |

### Raw PDF Gap Analysis (2026-02-27)

Cross-checked all 18 verified spans against raw PDF text. Found 2 gaps — 1 HIGH priority, 1 MODERATE.

| # | Gap Fact (exact PDF text) | Priority | Target KB | API Result |
|---|--------------------------|----------|-----------|------------|
| 8 | Perioperative: measure glucose + ketones on admission; proceed if clinically well and **ketones <1.0 mmol/l** | HIGH | KB-4, KB-16 | 201 |
| 9 | Sick day triggers include **excessive exercise or alcohol intake** (not just illness) | MODERATE | KB-4 | 201 |

**Acceptable omissions** (no action):
- "SGLT2i cause an initial natriuresis with accompanying weight reduction" — mechanism context, T3
- "Take note of country-to-country variation" in Figure 7 caption — generic note
- "Assess adverse effects, review knowledge" from Figure 6 — generic education items

### Figure 7 Drug-Linked Row Fix (2026-02-27)

D channel confirmed spans contained eGFR thresholds and doses but NOT drug names (row headers extracted separately as B channel noise). Added complete linked rows for empagliflozin and canagliflozin (dapagliflozin already had one).

| # | Fact Added | Target KB | API Result |
|---|-----------|-----------|------------|
| 10 | Figure 7 — Empagliflozin: 10mg (↑25mg), FDA ≥30 T2D/ASCVD + ≥20 HF, trials EMPA-REG ≥30 + EMPEROR ≥20 | KB-1 | 201 |
| 11 | Figure 7 — Canagliflozin: 100mg (300mg NOT CKD), FDA ≥30, CREDENCE ≥30 | KB-1 | 201 |

### Post-Review State (Final)

| Metric | Before | After Round 1 | After Gap Fill | After Fig 7 Fix |
|--------|--------|---------------|----------------|-----------------|
| **Total spans** | 61 | 68 | 70 | 72 |
| **Reviewed** | 0/61 | 68/68 | 70/70 | 72/72 |
| **Confirmed** | 0 | 11 | 11 | 11 |
| **Added (REVIEWER)** | 0 | 7 | 9 | 11 |
| **Rejected** | 0 | 50 | 50 | 50 |
| **Pipeline 2 ready** | No | 18 spans | 20 spans | **22 spans** (11 confirmed + 11 added) |
| **Figure 7 coverage** | Fragmented (no drug names) | Dapagliflozin linked | + ketone/sick day | **All 3 drugs fully linked** |
| **Extraction completeness** | ~40% | ~80% | ~92% | **~95%** |
