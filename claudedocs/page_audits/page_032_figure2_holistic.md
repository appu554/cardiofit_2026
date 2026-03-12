# Page 32 Audit — Practice Point 1.1.1 + Figure 2: Holistic Approach Algorithm

| Field | Value |
|-------|-------|
| **Page** | 32 (PDF page S31) |
| **Content Type** | PP 1.1.1 (comprehensive strategy) + Figure 2 (holistic treatment algorithm) + Research recommendations |
| **Extracted Spans** | 20 original + 10 REVIEWER = 30 total |
| **Channels** | B, C, D, E, F, L1_RECOVERY (all 6 channels) |
| **Disagreements** | 3 |
| **Review Status** | ALL REVIEWED — see Execution Log below |
| **Risk** | Oracle (L1_RECOVERY present) |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw API data (20 spans), channels confirmed (all 6), disagreements (3), all spans reviewed via API |
| **Execution Date** | 2026-02-26 |
| **Page Decision** | FLAGGED (Figure 2 algorithm fully captured via 10 REVIEWER facts + 3 confirmed spans) |

---

## Source PDF Content

**Narrative text:**
- CKD definition: persistently elevated urine albumin (≥30 mg/g [≥3 mg/mmol]), persistently reduced eGFR (<60), or both, for >3 months
- **Practice Point 1.1.1**: "Patients with diabetes and CKD should be treated with a comprehensive strategy to reduce risks of kidney disease progression and cardiovascular disease"
- As kidney function deteriorates, medication doses/types need adjustment
- Management of anemia, bone/mineral disorders, fluid/electrolyte disturbances
- Research recommendations for future studies

**Figure 2 — Holistic approach (HIGHEST VALUE FIGURE IN GUIDELINE):**

| Category | Interventions |
|----------|---------------|
| **Lifestyle** | Weight management, healthy diet, smoking cessation, physical activity |
| **First-line (All)** | Metformin (if eGFR ≥30), SGLT2i (Initiate eGFR ≥20; continue until dialysis/transplant), RAS inhibitor at max tolerated dose (if HTN*), Moderate/high-intensity statin |
| **Additional risk-based** | GLP-1 RA for glycemic target, **ns-MRA if ACR ≥30 mg/g [≥3 mg/mmol] and normal potassium** |
| **Other** | Dihydropyridine CCB/diuretic for BP, Antiplatelet for ASCVD, Other glucose-lowering drugs, **Steroidal MRA for resistant HTN if eGFR ≥45**, Ezetimibe/PCSK9i/icosapent ethyl |
| **Scope** | T2D only vs All patients (T1D and T2D) clearly distinguished |

**Figure 2 Caption (T1 content):**
- "ACEi or ARB should be first-line therapy for HTN when albuminuria is present, otherwise dihydropyridine CCB or diuretic can also be considered"
- "Finerenone is currently the only nonsteroidal MRA with proven clinical kidney and cardiovascular benefits"

---

## Key Spans Assessment

### Tier 1 Spans (11)

| Span | Channels | Conf | Assessment |
|------|----------|------|------------|
| **"Metformin (if eGFR ≥30) SGLT2i (Initiate eGFR ≥20; continue until dialysis or transplant)"** | L1 | 100% | **✅ T1 CORRECT — GOLD STANDARD** Drug + threshold + action from Figure 2 |
| "MRA" | B | 100% | **→ T3** Standalone drug class name |
| "SGLT2i" | B | 100% | **→ T3** Standalone drug class name |
| "ACEi" | B | 100% | **→ T3** Standalone drug class name |
| "ARB" | B | 100% | **→ T3** Standalone drug class name |
| "diuretic" | B | 100% | **→ T3** Standalone drug class name |
| **"finerenone is currently the only nonsteroidal MRA with proven clinical kidney an..."** | B,C,D,E | 100% | **✅ T1 CORRECT** Drug-specific efficacy/safety statement |
| **"eGFR <60 mL/min/1.73m²"** | C | 95% | **→ T2** CKD definition threshold, not a drug prescribing threshold |
| "Practice Point 1.1.1" | C | 98% | **→ T3** Label only, not the practice point text |
| "SGLT2i" (duplicate) | B | 100% | **→ T3** Duplicate standalone drug name |
| "MRA" (duplicate) | B | 100% | **→ T3** Duplicate standalone drug name |

### Tier 2 Spans (9)

| Span | Channels | Conf | Assessment |
|------|----------|------|------------|
| "calcium channel blocker" | B | 100% | **→ T3** Drug class name without prescribing context |
| "urine albumin" | C | 85% | **→ T3** Lab test name only |
| "30 mg" | C | 85% | **Partial** Threshold value (albuminuria) but decontextualized |
| "3 mg" | C | 85% | **Partial** Threshold value (albuminuria in mmol) but decontextualized |
| "eGFR" | C | 85% | **→ T3** Lab abbreviation only |
| **"Patients with diabetes and CKD should be treated with a comprehensive strategy to reduce risks..."** | F | 85% | **✅ T2 CORRECT** Full PP 1.1.1 recommendation text |
| "management of anemia, bone and mineral disorders, fluid and electrolyte disturbanz" | C,F | 90% | **→ T3** Narrative + OCR artifact ("disturbanz" = "disturbances") |
| "changes to types and doses of medications often need to be adjusted" | F | 85% | **→ T3** General narrative statement |
| "monitoring, glycemia management, and RAS blockade, as well as lifestyle factors for all CKD severities." | F | 90% | **→ T2 OK** Scope statement covering clinical topics |

---

## Critical Findings

### ✅ L1_RECOVERY Captures Gold Standard Span (Again)
The L1_RECOVERY channel extracted "Metformin (if eGFR ≥30) SGLT2i (Initiate eGFR ≥20; continue until dialysis or transplant)" — the single most valuable clinical span, capturing 2 drug thresholds in one span. This is a duplicate of the same span on page 21 (Figure 1 had same data; Figure 2 repeats it).

### ✅ Finerenone Statement Well-Captured
"finerenone is currently the only nonsteroidal MRA with proven clinical kidney and cardiovascular benefits" correctly extracted by 4 channels (B,C,D,E).

### ❌ Figure 2 Algorithm SEVERELY Under-Extracted (CRITICAL GAP)
Figure 2 is the **most comprehensive treatment algorithm in the entire guideline** — a step-by-step prescribing guide. Yet 8+ T1-quality facts from the figure are MISSING:

| Missing T1 Content | Clinical Importance |
|--------------------|---------------------|
| "RAS inhibitor at maximum tolerated dose (if HTN)" | First-line drug + dosing instruction |
| "Moderate- or high-intensity statin" | First-line drug for all patients |
| "GLP-1 RA if needed to achieve individualized glycemic target" | Additional therapy indication |
| "ns-MRA if ACR ≥30 mg/g [≥3 mg/mmol] and normal potassium" | Drug + threshold + lab prerequisite |
| "Steroidal MRA for resistant hypertension if eGFR ≥45" | Drug + indication + eGFR threshold |
| "Ezetimibe, PCSK9i, or icosapent ethyl if indicated based on ASCVD risk" | Lipid drugs + indications |
| "Antiplatelet agent for clinical ASCVD" | Drug class + indication |
| "ACEi or ARB should be first-line therapy for HTN when albuminuria is present" | First-line prescribing rule from caption |

### ⚠️ B Channel Drug Name Inflation
7 of 11 T1 spans are standalone drug names from B channel (MRA ×2, SGLT2i ×2, ACEi, ARB, diuretic). These are T3, inflating the T1 count from 2 genuine to 11 reported.

### ⚠️ OCR Artifact
"disturbanz" should be "disturbances" — OCR error from PDF extraction.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — Figure 2 algorithm is the guideline's most important prescribing guide; 8+ T1 facts missing |
| **Tier corrections** | 7 standalone drug names: T1 → T3; eGFR <60: T1 → T2; PP label: T1 → T3; CCB/urine albumin/eGFR: T2 → T3 |
| **Missing T1** | 8+ prescribing instructions from Figure 2 (RAS inhibitor, statin, GLP-1 RA, ns-MRA criteria, steroidal MRA, lipid drugs, antiplatelet, ACEi/ARB first-line rule) |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~15% — Figure 2 has 10+ T1 facts, only 2 captured |
| **Tier accuracy** | ~15% (3/20 correctly tiered) |
| **False positive T1 rate** | 82% (9/11 T1 spans are standalone names or labels) |
| **Genuine T1 spans** | 2 (L1 algorithm + finerenone statement) |
| **OCR artifacts** | 1 ("disturbanz") |
| **Overall quality** | **POOR** — Most important prescribing algorithm severely under-extracted |

---

## Execution Log — 2026-02-26

### Phase 1: API Rejections (17 spans)

| Span ID | Text (truncated) | Reason | Note |
|---------|-------------------|--------|------|
| `d225d241` | "MRA" | out_of_scope | Standalone drug class name — no prescribing context for L3 |
| `3da46ee6` | "SGLT2i" | out_of_scope | Standalone drug class name — no prescribing context for L3 |
| `9df56761` | "ACEi" | out_of_scope | Standalone drug class name — no prescribing context for L3 |
| `abd7892b` | "ARB" | out_of_scope | Standalone drug class name — no prescribing context for L3 |
| `ddbfdb8c` | "calcium channel blocker" | out_of_scope | Drug class name without prescribing context |
| `4598665e` | "diuretic" | out_of_scope | Standalone drug class name — no prescribing context for L3 |
| `4a8b9070` | "urine albumin" | out_of_scope | Lab test name only — no threshold or clinical action |
| `587948fd` | "30 mg" | out_of_scope | Decontextualized threshold value — no drug or condition |
| `09751e8b` | "3 mg" | out_of_scope | Decontextualized threshold value — no drug or condition |
| `a1f6e268` | "eGFR" | out_of_scope | Lab abbreviation only — no threshold or clinical action |
| `e046344d` | "eGFR <60 mL/min/1.73m²" | out_of_scope | CKD definition threshold, not a drug prescribing threshold |
| `23ba41a8` | "Practice Point 1.1.1" | out_of_scope | Label only — not the practice point content |
| `48c016f8` | "management of anemia, bone and mineral disorders, fluid and electrolyte disturbanz" | out_of_scope | Narrative fragment with OCR artifact ("disturbanz") |
| `4353b761` | "changes to types and doses of medications often need to be adjusted" | out_of_scope | General narrative statement — no specific prescribing content |
| `87f15c58` | "monitoring, glycemia management, and RAS blockade..." | out_of_scope | Narrative scope statement — no specific drug/threshold/action |
| `3fae5344` | "SGLT2i" (duplicate) | out_of_scope | Duplicate standalone drug class name in Research section |
| `c54d2d64` | "MRA" (duplicate) | out_of_scope | Duplicate standalone drug class name in Research section |

### Phase 2: API Confirmations (3 spans)

| Span ID | Text (truncated) | Action | Note |
|---------|-------------------|--------|------|
| `7b081a08` | "Metformin (if eGFR ≥30) SGLT2i (Initiate eGFR ≥20; continue until dialysis or transplant)" | **CONFIRM** | Gold standard Figure 2 span — L1_RECOVERY captures 2 drug thresholds + continuation rule |
| `29ebb5f1` | "finerenone is currently the only nonsteroidal mineralocorticoid receptor antagonist (MRA)..." | **CONFIRM** | Figure 2 caption — drug-specific efficacy statement from 4 channels (B,C,D,E) |
| `24d3d87d` | "Patients with diabetes and chronic kidney disease (CKD) should be treated with a comprehensive strategy..." | **CONFIRM** | Full Practice Point 1.1.1 recommendation text — comprehensive treatment strategy |

### Phase 3: Facts Added via UI (10 total)

| # | Fact Text | Note |
|---|-----------|------|
| 1 | "RAS inhibitor at maximum tolerated dose (if HTN)." | Figure 2 first-line therapy — RAS inhibitor dosing instruction for hypertensive patients. Verbatim from Figure 2 algorithm box, PDF S31. |
| 2 | "Moderate- or high-intensity statin." | Figure 2 first-line therapy — statin intensity recommendation for all patients. Verbatim from Figure 2 algorithm box, PDF S31. |
| 3 | "GLP-1 RA if needed to achieve individualized glycemic target." | Figure 2 additional therapy — GLP-1 RA indication for glycemic control. Verbatim from Figure 2 algorithm box, PDF S31. |
| 4 | "Nonsteroidal MRA if ACR ≥30 mg/g [≥3 mg/mmol] and normal potassium." | Figure 2 additional therapy — ns-MRA criteria with albuminuria threshold + potassium prerequisite. Verbatim from Figure 2 algorithm box, PDF S31. |
| 5 | "Antiplatelet agent for clinical ASCVD." | Figure 2 other therapy — antiplatelet indication. Verbatim from Figure 2 algorithm box, PDF S31. |
| 6 | "Steroidal MRA if needed for resistant hypertension if eGFR ≥45." | Figure 2 other therapy — steroidal MRA indication with eGFR threshold. Verbatim from Figure 2 algorithm box, PDF S31. |
| 7 | "Ezetimibe, PCSK9i, or icosapent ethyl if indicated based on ASCVD risk and lipids." | Figure 2 other therapy — lipid-lowering drugs for ASCVD risk. Verbatim from Figure 2 algorithm box, PDF S31. |
| 8 | "ACEi or ARB should be first-line therapy for hypertension (HTN) when albuminuria is present; otherwise dihydropyridine calcium channel blocker (CCB) or diuretic can also be considered. All 3 classes are often needed to attain blood pressure (BP) targets." | Figure 2 caption — ACEi/ARB first-line prescribing rule for HTN with albuminuria, plus CCB/diuretic alternative. Verbatim from PDF S31. |
| 9 | "Regular risk factor reassessment (every 3–6 months)." | Figure 2 monitoring instruction — risk factor reassessment frequency. Verbatim from Figure 2 algorithm box, PDF S31. KB-16 monitoring schedule. |
| 10 | "Regular reassessment of glycemia, albuminuria, blood pressure (BP), cardiovascular disease (CVD) risk, and lipids." | Figure 2 monitoring instruction — parameters for regular reassessment. Verbatim from Figure 2 algorithm box, PDF S31. KB-16 monitoring parameters. |

### Phase 4: Page Flag

| Page | Action | Method |
|------|--------|--------|
| 32 | **FLAGGED** | Auto-flagged when facts added via UI |

---

## Post-Execution Summary

### Final Span Counts

| Metric | Count |
|--------|-------|
| **Original spans** | 20 |
| **Rejected** | 17 |
| **Confirmed** | 3 |
| **Edited** | 0 |
| **Added (REVIEWER)** | 10 |
| **Final total** | 30 |

### Pipeline 2 L3-L5 Coverage Checklist

| Clinical Concept | KB Target | Source | Status |
|------------------|-----------|--------|--------|
| Metformin eGFR ≥30 + SGLT2i eGFR ≥20 continuation rule | KB-1, KB-16 | CONFIRMED (`7b081a08`) | ✅ |
| Finerenone = only ns-MRA with proven benefits | KB-1 | CONFIRMED (`29ebb5f1`) | ✅ |
| PP 1.1.1 comprehensive strategy for diabetes + CKD | KB-1 | CONFIRMED (`24d3d87d`) | ✅ |
| RAS inhibitor at max tolerated dose (if HTN) | KB-1 | ADDED (Fact 1) | ✅ |
| Moderate/high-intensity statin | KB-1 | ADDED (Fact 2) | ✅ |
| GLP-1 RA for glycemic target | KB-1 | ADDED (Fact 3) | ✅ |
| ns-MRA if ACR ≥30 mg/g and normal potassium | KB-1, KB-16 | ADDED (Fact 4) | ✅ |
| Antiplatelet for clinical ASCVD | KB-4 | ADDED (Fact 5) | ✅ |
| Steroidal MRA for resistant HTN if eGFR ≥45 | KB-1, KB-16 | ADDED (Fact 6) | ✅ |
| Ezetimibe/PCSK9i/icosapent ethyl for ASCVD risk | KB-1 | ADDED (Fact 7) | ✅ |
| ACEi/ARB first-line for HTN with albuminuria + CCB/diuretic alternative | KB-1 | ADDED (Fact 8) | ✅ |
| Risk factor reassessment every 3–6 months | KB-16 | ADDED (Fact 9) | ✅ |
| Monitoring parameters: glycemia, albuminuria, BP, CVD risk, lipids | KB-16 | ADDED (Fact 10) | ✅ |

### Post-Execution Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | **~98%** (was ~15%) — All Figure 2 algorithm prescriptive facts + monitoring instructions captured via 3 confirmed spans + 10 REVIEWER facts |
| **Noise removal** | 85% rejected (17/20) |
| **Overall quality** | **HIGH** (was POOR) — Complete Figure 2 prescribing algorithm + monitoring schedule now captured on page 32 |

### Key Observations

1. **L1_RECOVERY gold standard confirmed**: The `7b081a08` span captures "Metformin (if eGFR ≥30) SGLT2i (Initiate eGFR ≥20; continue until dialysis or transplant)" — same critical content as page 21/31. Per-page completeness requires it on this page too.

2. **85% noise rate**: 17 of 20 original spans were noise — the highest noise rate so far. B channel alone contributed 9 standalone drug names (MRA×2, SGLT2i×2, ACEi, ARB, diuretic, CCB, plus duplicates in Research section). C channel added 4 decontextualized values (urine albumin, 30 mg, 3 mg, eGFR). None usable for L3 structured extraction.

3. **Figure 2 is now fully captured**: The 10 REVIEWER facts cover the complete prescribing hierarchy: first-line (RAS inhibitor, statin), additional (GLP-1 RA, ns-MRA with thresholds), other (antiplatelet, steroidal MRA with eGFR ≥45, lipid drugs), the caption's ACEi/ARB prescribing rule, plus monitoring instructions (reassessment frequency every 3–6 months, monitoring parameters). Combined with the 3 confirmed spans, all prescriptive content from Figure 2 is now available for Pipeline 2.

4. **Per-page completeness applied**: Even though some Figure 2 content overlaps with pages 20, 21, and 31, all facts are captured directly on page 32. Pipeline 2 dedup handles cross-page references.
