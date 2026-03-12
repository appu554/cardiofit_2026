# Page 88 Audit — PP 4.2.1 Continuation, PP 4.2.2–4.2.5, Figure 29 GLP-1 RA Dosing & CKD Adjustments

| Field | Value |
|-------|-------|
| **Page** | 88 (PDF page S87) |
| **Content Type** | PP 4.2.1 continuation (prioritize liraglutide/semaglutide/dulaglutide; PIONEER 6/SOUL context; treatment heterogeneity), PP 4.2.2 (low dose start, titrate slowly per Figure 29), PP 4.2.3 (GLP-1 RA + DPP-4i prohibition), PP 4.2.4 (hypoglycemia risk with SU/insulin; GLP-1 RA preferred over DPP-4i/TZD/SU/insulin/acarbose; reduce SU/insulin dose when starting GLP-1 RA), PP 4.2.5 (GLP-1 RA for obesity + T2D + CKD; semaglutide/liraglutide approved for nondiabetic obesity; AWARD-7 weight ~5 kg differential; weight loss for transplant qualification), Figure 29 (GLP-1 RA dosing table with CKD dose adjustments for 7 agents) |
| **Extracted Spans** | 202 total (147 T1, 55 T2) — 0 EDITED |
| **Channels** | B (Drug Dictionary — ~133 spans), C (Grammar/Regex — ~63 spans), D (Table Decomp — 3 spans), F (NuExtract LLM — 2 spans via B+C+F and B+F), E (GLiNER NER — 1 span) |
| **Risk** | Disagreement |
| **Disagreements** | 2 |
| **Review Status** | **FINAL**: 23 ADDED (P2-ready), 0 PENDING, 216 REJECTED |
| **Audit Date** | 2026-02-24 |
| **Cross-Check** | 2026-02-25 — counts confirmed (202 total: 147 T1, 55 T2), channels confirmed (B, C, D, E, F) |
| **Raw PDF Cross-Check** | 2026-02-28 — 14 agent duplicates rejected, 5 PENDING rejected, 5 gaps added (G88-A–E), 18 agent spans kept |

---

## Source PDF Content (from UI)

**PP 4.2.1 Continuation:**
- Albiglutide/efpeglenatide currently unavailable → prioritize liraglutide, semaglutide (injectable), dulaglutide
- CV benefit not demonstrated for oral semaglutide (PIONEER 6 powered for non-inferiority only; SOUL NCT03914326 ongoing)
- "Patients with T2D and CKD are a heterogeneous group... Treatment algorithms must be tailored to individuals"

**Practice Point 4.2.2:**
- "To minimize gastrointestinal side effects, start with a low dose of GLP-1 RA, and titrate up slowly (Figure 29)"

**Practice Point 4.2.3:**
- "GLP-1 RA should not be used in combination with dipeptidyl peptidase-4 (DPP-4) inhibitors"
- "DPP-4 inhibitors and GLP-1 RA should not be used together"
- Consider stopping DPP-4i to facilitate GLP-1 RA treatment

**Practice Point 4.2.4:**
- "The risk of hypoglycemia is generally low with GLP-1 RA when used alone, but risk is increased when GLP-1 RA is used concomitantly with other medications such as sulfonylureas or insulin"
- "The doses of sulfonylurea and/or insulin may need to be reduced"
- "GLP-1 RA are preferred over classes of glucose-lowering medications with less evidence supporting reduction of cardiovascular or kidney risks (e.g., DPP-4 inhibitors, thiazolidinediones, sulfonylureas, insulin, and acarbose)"

**Practice Point 4.2.5:**
- "GLP-1 RA may be preferentially used in patients with obesity, T2D, and CKD to promote intentional weight loss"
- Semaglutide and liraglutide approved for weight loss in nondiabetic obesity
- Tirzepatide studied in SURMONT trial for obesity without diabetes
- AWARD-7: dulaglutide 1.5 mg weekly → ~4 kg weight loss over 1 year; insulin users gained >1 kg → **~5 kg differential**
- Weight loss clinically meaningful for CV and CKD risk factors
- **Weight loss may be required to qualify for kidney transplant**

**Figure 29 — GLP-1 RA Dosing and CKD Dose Modifications:**

| Agent | Dose | CKD Adjustment |
|-------|------|----------------|
| **Dulaglutide** | 0.75 mg and 1.5 mg once weekly | No dosage adjustment; Use with eGFR >15 ml/min per 1.73 m² |
| **Exenatide** | 10 μg twice daily | Use with CrCl >30 ml/min |
| **Exenatide extended-release** | 2 mg once weekly | Use with eGFR >45 ml/min per 1.73 m² |
| **Liraglutide** | 1.2 mg and 1.8 mg once daily | No dosage adjustment; Limited data for severe CKD |
| **Lixisenatide** | 10 μg and 20 μg once daily | Not recommended with eGFR <15 ml/min per 1.73 m² |
| **Semaglutide (injection)** | 0.5 mg and 1 mg once weekly | No dosage adjustment; Limited data for severe CKD |
| **Semaglutide (oral)** | 3 mg, 7 mg, or 14 mg daily | No dosage adjustment; Limited data for severe CKD |

---

## Key Spans Assessment

### Tier 1 Spans (147)

#### Genuinely Correct T1 (5 spans)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| **"DPP-4 inhibitors and GLP-1 RA should not be used together."** | B+C+F | 100% | **✅ T1 CORRECT** — PP 4.2.3 drug interaction prohibition. **B+C+F triple-channel gold standard** — the 6th instance in the audit. Complete clinical sentence with actionable prohibition |
| **"GLP-1 RA are preferred over classes of glucose-lowering medications with less evidence supporting reduction of cardiovas..."** | B+F | 98% | **✅ T1 CORRECT** — PP 4.2.4 drug class hierarchy. Truncated but captures the key assertion. Lists DPP-4i, TZD, SU, insulin, acarbose as inferior |
| "Use with CrCl > 30 ml/min" | D | 92% | **✅ T1 CORRECT** — Exenatide CKD dosing threshold from Figure 29 |
| "No dosage adjustment Use with eGFR > 15 ml/min per 1.73 m²" | D | 92% | **✅ T1 CORRECT** — Dulaglutide CKD adjustment from Figure 29 |
| "Use with eGFR > 45 ml/min per 1.73 m²" | D | 92% | **✅ T1 CORRECT** — Exenatide extended-release CKD threshold from Figure 29 |

#### PP Labels Without Text (4 spans → T3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Practice Point 4.2.2" | C | 98% | **→ T3** — PP label only. Text "start with a low dose of GLP-1 RA, and titrate up slowly" NOT captured |
| "Practice Point 4.2.3" | C | 98% | **→ T3** — PP label only (actual text captured by B+C+F span above) |
| "Practice Point 4.2.4" | C | 98% | **→ T3** — PP label only |
| "Practice Point 4.2.5" | C | 98% | **→ T3** — PP label only |

#### eGFR Threshold Fragments (5 spans → T3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "eGFR <30 mL/min/1.73m²" | C | 95% | **→ T3** — Standalone threshold; no drug association |
| "eGFR <60 mL/min/1.73m²" ×2 | C | 95% | **→ T3** — Standalone threshold fragments |
| "eGFR ≥60 mL/min/1.73m²" | C | 95% | **→ T3** — Standalone threshold fragment |
| "eGFR ≥20 mL/min/1.73m²" | C | 95% | **→ T3** — SGLT2i reference threshold fragment |

#### Drug Name Mentions (~133 spans → T3)

| Drug | Count | Assessment |
|------|-------|------------|
| GLP-1 RA | ~40 | **→ T3** — Drug class name mentions without clinical sentence context |
| dulaglutide | ~15 | **→ T3** — Drug name mentions from PP text and Figure 29 |
| semaglutide | ~12 | **→ T3** — Drug name mentions |
| insulin | ~10 | **→ T3** — Drug name mentions from PP 4.2.4 hypoglycemia text |
| liraglutide | ~8 | **→ T3** — Drug name mentions |
| sulfonylureas/sulfonylurea | ~6 | **→ T3** — Drug name mentions from PP 4.2.4 |
| SGLT2i | ~6 | **→ T3** — Drug class mentions |
| ACEi | 3 | **→ T3** — Drug class mentions |
| ARB | 3 | **→ T3** — Drug class mentions |
| Exenatide/exenatide | ~4 | **→ T3** — Drug name mentions |
| DPP-4 inhibitors | 1 | **→ T3** — Drug class mention |
| thiazolidinediones | 1 | **→ T3** — Drug class mention |

**Summary: 5/147 T1 genuinely correct (3.4%). 4 PP labels + 5 eGFR fragments + ~133 drug names = 142/147 noise or T3.**

### Tier 2 Spans (55)

| Span Category | Count | Channel | Conf | Assessment |
|---------------|-------|---------|------|------------|
| "eGFR" (standalone) | ~13 | C | 85% | **→ NOISE** — Bare lab name |
| "HbA1c" | 3 | C | 85% | **→ NOISE** — Bare lab name |
| "HbA1c" | 1 | E | 85% | **→ NOISE** — E channel fires on lab name (first E in Ch4) |
| "creatinine" | 1 | C | 85% | **→ NOISE** — Bare lab name |
| "Weekly"/"weekly" | ~6 | C | 85% | **→ NOISE** — Frequency word from Figure 29 |
| "daily" | ~4 | C | 85% | **→ NOISE** — Frequency word from Figure 29 |
| Dose amounts (0.75/1/1.2/1.5/1.8/2/3/7/14/0.5 mg) | ~14 | C | 85% | **⚠️ Borderline** — Dosing amounts from Figure 29; individually they're fragments, but they ARE clinical dosing data |
| "33.9 mg" and "339 mg" | 2 | C | 85% | **→ T2 OK** — Albuminuria thresholds (ACR >33.9 mg/mmol / >339 mg/g) from PP text |
| "eGFR >15 mL/min/1.73m²" | 1 | C | 95% | **→ T1** — Dulaglutide CKD threshold (duplicates D channel extraction) |
| "CrCl >30 ml/min" | 1 | C | 95% | **→ T1** — Exenatide CKD threshold |
| "eGFR >45 mL/min/1.73m²" | 1 | C | 95% | **→ T1** — Exenatide ER CKD threshold |
| "eGFR <15 mL/min/1.73m²" | 1 | C | 95% | **→ T1** — Lixisenatide contraindication threshold |
| "not recommended" / "Not recommended" | 2 | C | 95% | **→ T1** — Safety directive (lixisenatide at eGFR <15) but without drug context |
| "should not be used" | 1 | C | 95% | **→ T1** — Prohibition fragment from PP 4.2.3, missing drug names |
| "stop" | 1 | C | 90% | **→ NOISE** — Single word fragment from "stopping the gliptin" |
| "1 mg" (standalone) | 1 | C | 85% | **→ NOISE** — Ambiguous dose fragment |

**Summary: 0/55 T2 correctly tiered. 4 CKD thresholds → T1; 3 directive fragments → T1 (missing context); ~14 dosing amounts → borderline T2; ~34 standalone lab/frequency words → NOISE.**

---

## Critical Findings

### ✅ B+C+F Triple-Channel Returns — PP 4.2.3 Drug Prohibition

"DPP-4 inhibitors and GLP-1 RA should not be used together." is the **6th B+C+F triple-channel span** in the audit (after pp67, 76, 78, 80, 81). It captures a complete, actionable drug interaction prohibition with 100% confidence. This is the first B+C+F span since page 81 (7-page gap) and the only one in the GLP-1 RA section.

The pattern holds: B fires on drug names (DPP-4 inhibitors, GLP-1 RA), C fires on "should not be used" prohibition pattern, F extracts the complete sentence. When all three channels converge, the result is always genuine T1 patient safety content.

### ✅ B+F Captures PP 4.2.4 Drug Class Hierarchy

"GLP-1 RA are preferred over classes of glucose-lowering medications with less evidence..." (B+F, 98%) is a genuine clinical hierarchy statement. B fires on multiple drug class names, F extracts the comparative sentence. Truncated but the key assertion is preserved.

### ✅ D Channel Extracts 3 Figure 29 CKD Dosing Thresholds

The D channel captures 3 of the 7 CKD adjustment cells from Figure 29:
- Dulaglutide: "No dosage adjustment Use with eGFR > 15"
- Exenatide: "Use with CrCl > 30 ml/min"
- Exenatide ER: "Use with eGFR > 45 ml/min per 1.73 m²"

These are genuinely useful T1 extractions — drug-specific renal dosing thresholds. However, D misses 4 of 7 rows (liraglutide, lixisenatide, semaglutide injection, semaglutide oral).

### ⚠️ E Channel First Appearance in Chapter 4

The E (GLiNER NER) channel fires once on "HbA1c" as T2. This is the first E channel activity since the earlier chapters. Its single extraction is a bare lab name with no clinical context — consistent with E's pattern of extracting entity names without surrounding sentences.

### ❌ B Channel Produces 133 Drug Name Mentions — Worst in Audit

With ~133 standalone drug name T1 spans, page 88 has the **highest B channel noise count** in the entire audit. This surpasses p81's 72 metformin mentions. The B channel fires on every occurrence of GLP-1 RA (~40×), dulaglutide (~15×), semaglutide (~12×), insulin (~10×), etc. — all classified T1 with zero clinical sentence context.

### ❌ PP 4.2.2 and PP 4.2.5 Text NOT Captured

While PP 4.2.3 text was captured (B+C+F) and PP 4.2.4 hierarchy was partially captured (B+F), the text of PP 4.2.2 and PP 4.2.5 is entirely missing:
- **PP 4.2.2**: "start with a low dose of GLP-1 RA, and titrate up slowly" — dosing safety guidance
- **PP 4.2.5**: "GLP-1 RA may be preferentially used in patients with obesity" — population selection guidance

### ❌ Figure 29 Incomplete — 4 of 7 Agents Missing CKD Adjustments

D channel captured CKD adjustments for dulaglutide, exenatide, and exenatide ER. Missing:
- **Liraglutide**: "No dosage adjustment; Limited data for severe CKD"
- **Lixisenatide**: "Not recommended with eGFR <15 ml/min per 1.73 m²" — critical safety threshold
- **Semaglutide (injection)**: "No dosage adjustment; Limited data for severe CKD"
- **Semaglutide (oral)**: "No dosage adjustment; Limited data for severe CKD"

### ❌ Critical Missing Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 4.2.2: "start with low dose, titrate up slowly" (full text) | **T1** | Dosing safety — prevents GI side effects |
| PP 4.2.5: "preferentially used in patients with obesity" (full text) | **T1** | Population selection directive |
| Lixisenatide not recommended at eGFR <15 (as complete fact) | **T1** | Drug-specific safety threshold |
| "Limited data for severe CKD" for liraglutide/semaglutide | **T1** | Evidence limitation warning |
| "Reduce dose of SU/insulin when starting GLP-1 RA" (full sentence) | **T1** | Hypoglycemia prevention |
| Consider stopping DPP-4i to facilitate GLP-1 RA | **T1** | Drug switching guidance |
| CV benefit not demonstrated for oral semaglutide | **T1** | Prescribing limitation |
| Weight differential ~5 kg dulaglutide vs insulin (AWARD-7) | **T2** | Clinical outcome metric |
| Weight loss may be required for kidney transplant qualification | **T2** | Treatment goal context |
| Figure 29 complete drug-dose-CKD rows (4 missing agents) | **T2** | Dosing reference data |
| Treatment must be tailored to individual patients | **T3** | Clinical context statement |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **FLAG** — 202 spans with only 5 genuinely correct T1 (3.4%); B+C+F captures PP 4.2.3 prohibition (gold standard); B+F captures PP 4.2.4 hierarchy; D captures 3/7 Figure 29 CKD thresholds; but 133 B channel drug name mentions flood the page; PP 4.2.2 and PP 4.2.5 text missing; 4/7 Figure 29 agents missing CKD data |
| **Tier corrections** | ~133 drug names: T1 → T3; 4 PP labels: T1 → T3; 5 eGFR thresholds: T1 → T3; 4 T2 CKD thresholds → T1; ~34 T2 standalone fragments → NOISE |
| **Missing T1** | PP 4.2.2 text, PP 4.2.5 text, lixisenatide eGFR <15, "limited data severe CKD" warnings, reduce SU/insulin dose guidance, oral semaglutide CV limitation |
| **Missing T2** | AWARD-7 weight differential, transplant qualification context, 4 missing Figure 29 rows |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~15% — B+C+F captures PP 4.2.3, B+F captures PP 4.2.4 hierarchy, D captures 3 dosing thresholds; but PP 4.2.2/4.2.5 text missing, Figure 29 incomplete, weight/transplant data absent |
| **Tier accuracy** | ~2.5% (5/147 T1 correct + 0/55 T2 correct = 5/202) |
| **Noise ratio** | ~85% — 133 drug names + 34 standalone fragments + 4 PP labels + 5 bare thresholds = ~176/202 noise or T3 |
| **Genuine T1 content** | 5 extracted (B+C+F prohibition, B+F hierarchy, 3 D dosing thresholds) |
| **Prior review** | 0/202 reviewed |
| **Overall quality** | **MODERATE-POOR — FLAG** — The B+C+F and B+F spans are excellent; D captures useful dosing thresholds; but the page is overwhelmed by 133 B channel drug name mentions masking the 5 genuine extractions. PP 4.2.2/4.2.5 text and 4/7 Figure 29 CKD rows remain uncaptured |

---

## B+C+F Triple-Channel Tracker (Audit-Wide)

| # | Page | Span Text | Confidence |
|---|------|-----------|------------|
| 1 | 67 | "kidney protective effects... independent of glucose lowering" | 100% |
| 2 | 76 | SGLT2i eGFR/kidney function statement | 100% |
| 3 | 78 | Finerenone MRA evidence statement | 100% |
| 4 | 80 | Metformin dosing/safety assertion | 100% |
| 5 | 81 | Metformin clinical directive | 100% |
| 6 | **88** | **"DPP-4 inhibitors and GLP-1 RA should not be used together."** | **100%** |

All 6 B+C+F spans: 100% confidence, all genuine T1 patient safety content, zero false positives.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Cross-Check State
- **Original extraction**: 202 spans (147 T1, 55 T2) — all rejected by agent pass (drug names, fragments, noise)
- **Agent-added**: 32 ADDED spans + 5 PENDING from parallel agent processing
- **Total before cross-check**: 239 spans (202 REJECTED + 32 ADDED + 5 PENDING)

### Duplicate Rejections (14 ADDED)

| # | ID | Duplicate Text | Kept Span |
|---|-----|---------------|-----------|
| 1 | `5b172824` | Liraglutide: 1.2 mg and 1.8 mg once daily… | `8cd61e8c` (full dosing) |
| 2 | `d9810d33` | Semaglutide (injection): 0.5 mg and 1 mg once weekly… | `34a142f2` (full dosing) |
| 3 | `f2aa9a66` | Semaglutide (oral): 3 mg, 7 mg, or 14 mg daily… | `4917f932` (full dosing) |
| 4 | `d1b8fdfc` | Lixisenatide: Not recommended eGFR <15 (fragment) | Subset of `d74edb81` |
| 5 | `a6faf254` | Liraglutide: No dosage adjustment (fragment) | Subset of `8cd61e8c` |
| 6 | `33a8a177` | Semaglutide (inj): No dosage adjustment (fragment) | Subset of `34a142f2` |
| 7 | `74684a52` | Semaglutide (oral): No dosage adjustment (fragment) | Subset of `4917f932` |
| 8 | `cada9ad8` | PP 4.2.2 GI titration (duplicate) | `5ae78944` |
| 9 | `b8e2abbd` | "Consider stopping DPP-4i…" (variant) | `bc791f0d` |
| 10 | `97f41693` | "The doses of sulfonylurea and/or insulin may need to be reduced" | `f6373ef4` |
| 11 | `5f04befe` | PP 4.2.5 obesity (duplicate) | `00bd1dcc` |
| 12 | `03ed142b` | Oral semaglutide CV caveat (shortened) | `cc8576d4` (fuller version) |
| 13 | `98606f78` | AWARD-7 weight differential (shortened) | `a633b269` (fuller version) |
| 14 | `fa3b6b80` | "Weight loss may be required to qualify for kidney transplant" | `72485526` |

**Duplication rate**: 14/32 agent-added = **44%**

### PENDING Rejections (5)

| # | ID | Text | Reason |
|---|-----|------|--------|
| 1 | `a8ce9221` | "Use with CrCl > 30 ml/min" | D-channel fragment, covered by `32bd4bbb` |
| 2 | `43d9780b` | "No dosage adjustment Use with eGFR > 15 ml/min per 1.73 m 2" | D-channel fragment, covered by `811832bd` |
| 3 | `7d7861c9` | "Use with eGFR > 45 ml/min per 1.73 m 2" | D-channel fragment, covered by `55d02f5f` |
| 4 | `2fcd9550` | "DPP-4 inhibitors and GLP-1 RA should not be used together." | Redundant with `a3aba8f7` (PP 4.2.3) |
| 5 | `3f47dc8b` | "GLP-1 RA are preferred over classes of glucose-lowering medications…" (truncated) | Replaced by fuller gap G88-A including drug class list |

### Gap Additions (4)

| ID | Gap | Text | Target KB | Priority |
|----|-----|------|-----------|----------|
| G88-A | GLP-1 RA class preference | "GLP-1 RA are preferred over classes of glucose-lowering medications with less evidence supporting reduction of cardiovascular or kidney risks (e.g., DPP-4 inhibitors, thiazolidinediones, sulfonylureas, insulin, and acarbose)." | KB-1 | High |
| G88-B | Stop/reduce SU/insulin on GLP-1 RA start | "it is reasonable to stop or reduce the dose of sulfonylurea or insulin when starting a GLP-1 RA if the combination may lead to an unacceptable risk of hypoglycemia." | KB-1, KB-4 | High |
| G88-C | Tirzepatide SURMOUNT trial | "tirzepatide has also been studied for obesity in patients without diabetes in the SURMOUNT trial" | KB-1 | Medium |
| G88-D | Weight loss clinical significance | "This magnitude of weight loss is clinically meaningful from the perspectives of improving cardiovascular and CKD risk factors and for kidney and heart protection." | KB-4 | Medium |
| G88-E | Lixisenatide complete dosing row | "Lixisenatide: 10 μg and 20 μg once daily; No dosage adjustment; Limited data for severe CKD; Not recommended with eGFR <15 ml/min per 1.73 m2" | KB-1 | High |

### Agent-Kept Spans (18)

**Figure 29 Dosing Table (7 spans — all 7 drugs):**

| ID | Agent | Dose + CKD Adjustment |
|----|-------|-----------------------|
| `811832bd` | Dulaglutide | 0.75/1.5 mg weekly; No adjustment; eGFR >15 |
| `32bd4bbb` | Exenatide | 10 mcg BID; CrCl >30 |
| `55d02f5f` | Exenatide ER | 2 mg weekly; eGFR >45 |
| `8cd61e8c` | Liraglutide | 1.2/1.8 mg daily; No adjustment; Limited data severe CKD |
| `d74edb81` | Lixisenatide | 10/20 mcg daily; Not recommended eGFR <15 |
| `34a142f2` | Semaglutide (inj) | 0.5/1 mg weekly; No adjustment; Limited data severe CKD |
| `4917f932` | Semaglutide (oral) | 3/7/14 mg daily; No adjustment; Limited data severe CKD |

**Practice Points (6 spans):**

| ID | PP | Text |
|----|-----|------|
| `5ae78944` | 4.2.2 | "To minimize gastrointestinal side effects, start with a low dose of GLP-1 RA, and titrate up slowly (Figure 29)" |
| `a3aba8f7` | 4.2.3 | "GLP-1 RA should not be used in combination with dipeptidyl peptidase-4 (DPP-4) inhibitors" |
| `bc791f0d` | 4.2.3 | "Consider stopping DPP-4 inhibitor to facilitate GLP-1 RA treatment" |
| `f90aad7e` | 4.2.4 | "The risk of hypoglycemia is generally low with GLP-1 RA when used alone, but risk is increased when GLP-1 RA is used concomitantly with other medications such as sulfonylureas or insulin" |
| `f6373ef4` | 4.2.4 | "The doses of sulfonylurea and/or insulin may need to be reduced" |
| `00bd1dcc` | 4.2.5 | "GLP-1 RA may be preferentially used in patients with obesity, T2D, and CKD to promote intentional weight loss" |

**Clinical Evidence & Context (5 spans):**

| ID | Text |
|----|------|
| `cc8576d4` | "Cardiovascular benefit has not been demonstrated for oral semaglutide (PIONEER 6 was powered for non-inferiority only; SOUL NCT03914326 ongoing)" |
| `b0e3b63e` | "Semaglutide and liraglutide approved for weight loss in nondiabetic obesity" |
| `a633b269` | "AWARD-7: dulaglutide 1.5 mg weekly resulted in approximately 4 kg weight loss over 1 year; insulin users gained more than 1 kg, approximately 5 kg differential" |
| `72485526` | "Weight loss may be required to qualify for kidney transplant" |
| `011e973b` | "Patients with T2D and CKD are a heterogeneous group. Treatment algorithms must be tailored to individuals" |

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total spans** | 239 |
| **ADDED (P2-ready)** | 23 (18 agent-kept + 5 gap additions) |
| **PENDING** | 0 |
| **REJECTED** | 216 (197 original extraction noise + 14 agent duplicates + 5 PENDING) |
| **Extraction completeness** | ~95% — all 5 PPs captured, all 7 Figure 29 drugs with CKD adjustments, AWARD-7 weight data, oral semaglutide caveat, drug class hierarchy, SU/insulin dose guidance, tirzepatide/SURMOUNT reference |
| **Duplication rate** | 44% of agent-added spans (14/32) |
| **Cross-check date** | 2026-02-28 |
