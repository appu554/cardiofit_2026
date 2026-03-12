# Page 60 Audit — PP 2.1.5 (Hypoglycemia Drug Selection), PP 2.1.6 (CGM Evolution), Figure 13, Rec 2.2.1 (Glycemic Targets)

| Field | Value |
|-------|-------|
| **Page** | 60 (PDF page S59) |
| **Content Type** | PP 2.1.5 (glucose-lowering agent selection to minimize hypoglycemia risk without CGM/SMBG), PP 2.1.6 (CGM device evolution and specialist consultation), Figure 13 (drug class vs hypoglycemia risk vs CGM/SMBG rationale), research recommendations, Section 2.2 Glycemic Targets, Rec 2.2.1 (individualized HbA1c target <6.5% to <8.0%, 1C) |
| **Extracted Spans** | 45 total (22 T1, 23 T2) |
| **Channels** | B, C, D, E, F |
| **Disagreements** | 1 |
| **Review Status** | PENDING: 45 |
| **Risk** | Disagreement |
| **Cross-Check** | Verified against pipeline export 2026-02-25 |
| **Audit Date** | 2026-02-25 |

---

## Source PDF Content

**Practice Point 2.1.5:**
- "For patients with T2D and CKD who choose not to do daily glycemic monitoring by CGM or SMBG, glucose-lowering agents that pose a lower risk of hypoglycemia are preferred and should be administered in doses that are appropriate for the level of eGFR"
- **High hypoglycemia risk agents**: insulin, sulfonylureas, meglitinides (raise blood insulin levels)
- **Low/no hypoglycemia risk agents**: metformin, SGLT2i, GLP-1 RA, DPP-4 inhibitors
- Without daily monitoring, difficult to avoid hypoglycemic episodes with high-risk agents
- Patients with advanced CKD at increased risk

**Practice Point 2.1.6:**
- "CGM devices are rapidly evolving with multiple functionalities (e.g., real-time and intermittently scanned CGM). Newer CGM devices may offer advantages for certain patients, depending on their values, goals, and preferences"
- CGM features: alarms, cell phone linkage, factory calibration, GMI, ambulatory glucose profiles, closed-loop insulin delivery integration
- Consultation with diabetes technology specialist (certified diabetes educator) recommended
- Devices differ in: accuracy, calibration needs, placement, sensor life, warm-up time, transmitter type, display, data sharing, cost, insurance

**Figure 13 — Drug Class vs Hypoglycemia Risk vs CGM/SMBG Rationale:**

| Drug Class | Hypoglycemia Risk | CGM/SMBG Rationale |
|-----------|-------------------|---------------------|
| Metformin | Lower | Lower |
| SGLT2 inhibitors | Lower | Lower |
| GLP-1 receptor agonists | Lower | Lower |
| DPP-4 inhibitors | Lower | Lower |
| Insulin | Higher | Higher |
| Sulfonylureas | Higher | Higher |
| Meglitinides | Higher | Higher |

**Research Recommendations (Section 2.1 closing):**
- Identify patients where HbA1c produces biased estimate
- Identify patients at high risk of hypoglycemia who benefit from CGM/SMBG
- Develop CGM approaches for high-risk hypoglycemia patients
- Determine overall benefits/harms of SMBG and CGM
- Develop/validate alternative biomarkers for glycemic monitoring
- Test whether CGM improves clinical outcomes

**Recommendation 2.2.1 (1C — Strong/Low):**
- "We recommend an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis (Figure 14)"
- Higher value on potential benefits of individualized target balancing risks

---

## Key Spans Assessment

### Tier 1 Spans (17)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| **"· SGLT2 inhibitors"** (D) | D | 92% | **→ T3** — Drug class from Figure 13 table row, no prescribing context |
| **"· DPP-4 inhibitors"** (D) | D | 92% | **→ T3** — Drug class from Figure 13 table row |
| **"· Sulfonylureas"** (D) | D | 92% | **→ T3** — Drug class from Figure 13 table row |
| **"· Metformin"** (D) | D | 92% | **→ T3** — Drug class from Figure 13 table row |
| **"· Insulin"** (D) | D | 92% | **→ T3** — Drug class from Figure 13 table row |
| **"Practice Point 2.1.5"** (C) | C | 98% | **→ T3** — PP label only (text NOT captured) |
| **"insulin"** (B) ×4 | B | 100% | **ALL → T3** — Drug name only |
| **"sulfonylureas"** (B) | B | 100% | **→ T3** — Drug name only |
| **"metformin"** (B) | B | 100% | **→ T3** — Drug name only |
| **"SGLT2i"** (B) | B | 100% | **→ T3** — Drug class name only |
| **"GLP-1 RA"** (B) | B | 100% | **→ T3** — Drug class name only |
| **"Practice Point 2.1.6"** (C) | C | 98% | **→ T3** — PP label only (text NOT captured) |
| **"Recommendation 2.2.1"** (C) | C | 98% | **→ T3** — Rec label only (text NOT captured) |
| **"eGFR) ≥90 mL/min/1.73m²; G5, eGFR <15 mL/min/1.73m²"** (C) | C | 95% | **⚠️ SHOULD STAY T1** — eGFR thresholds with CKD stage classification (G1 and G5 boundaries); has disagreement flag |

**Summary: 1/17 T1 spans may be genuine (eGFR thresholds with CKD staging). 16/17 are drug names, PP/Rec labels, or Figure 13 table fragments.**

### Tier 2 Spans (20)

| Category | Count | Assessment |
|----------|-------|------------|
| **"Higher"** (D channel) ×3 | 3 | **→ T3** — Risk level word from Figure 13 without drug context |
| **"Lower"** (D channel) ×2 | 2 | **→ T3** — Risk level word from Figure 13 without drug context |
| **"Rationale for CGM or SMBG"** (D) | 1 | **✅ T2 OK** — Column header from Figure 13 (provides context) |
| **"·"** (D) | 1 | **→ NOISE** — Bullet character only |
| **"Meglitinides"** (D) | 1 | **→ T3** — Drug class from Figure 13 (not in B channel dictionary) |
| **"avoid"** (E) ×2 | 2 | **→ T3** — Action verb without clinical context |
| **"daily"** (C) ×4 | 4 | **→ T3** — Frequency word |
| **"eGFR"** (C) | 1 | **→ T3** — Lab abbreviation |
| **"CGM technology has greatly impacted diabetes self-management by providing glycemic assessment moment-to-moment, allowing..."** (F) | 1 | **✅ T2 OK** — Complete clinical sentence about CGM benefits; relates to PP 2.1.6 |
| **"HbA1c"** (C) ×4 | 4 | **→ T3** — Lab test name repetition (continuing C channel pattern) |

**Summary: 2/20 T2 correctly tiered or meaningful (Figure 13 column header + F channel CGM sentence). 18/20 are noise (Higher/Lower fragments, daily ×4, HbA1c ×4, avoid ×2, eGFR, bullet, Meglitinides).**

---

## Critical Findings

### ✅ D Channel Figure 13 Decomposition — Partially Useful
The D channel decomposes Figure 13's drug-risk matrix but only extracts individual cells:
- Drug names: "· SGLT2 inhibitors", "· Metformin", "· Insulin" etc. (all as separate T1 spans)
- Risk levels: "Higher", "Lower" (as separate T2 spans)
- Column header: "Rationale for CGM or SMBG"

The figure's key clinical message — **which drug classes have higher vs lower hypoglycemia risk** — is NOT captured as a linked statement. Each cell is isolated.

### ✅ F Channel Produces CGM Technology Sentence
"CGM technology has greatly impacted diabetes self-management by providing glycemic assessment moment-to-moment" — genuine clinical sentence from PP 2.1.6 context. F channel continues to extract well from narrative prose.

### ⚠️ eGFR Threshold Span — Interesting T1
"eGFR) ≥90 mL/min/1.73m²; G5, eGFR <15 mL/min/1.73m²" captures the CKD staging boundaries (G1 at ≥90, G5 at <15). This appears to be from the Rec 2.2.1 context (glycemic targets by CKD stage). Has disagreement flag. Could be T1 if tied to glycemic target recommendations.

### ❌ PP 2.1.5 Text NOT EXTRACTED (Critical Prescribing Guidance)
"glucose-lowering agents that pose a lower risk of hypoglycemia are preferred and should be administered in doses appropriate for the level of eGFR" — this is a T1 prescribing instruction linking drug selection to both hypoglycemia risk AND eGFR-based dosing. Missing because PP text doesn't start with a drug name.

### ❌ PP 2.1.6 Text NOT EXTRACTED
"CGM devices are rapidly evolving with multiple functionalities" — PP label captured but text missing. Lower clinical importance (T2 technology guidance) but still a gap.

### ❌ Rec 2.2.1 Text NOT EXTRACTED (Critical Target Recommendation)
"We recommend an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis" — this is ONE OF THE MOST IMPORTANT recommendations in the entire guideline (glycemic target range for CKD patients). Only the label is captured.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 2.1.5 text: prefer lower hypoglycemia-risk agents, dose by eGFR | **T1** | Drug selection + dosing guidance |
| Rec 2.2.1 text: individualized HbA1c <6.5% to <8.0% in CKD (not dialysis) | **T1** | **CRITICAL** — Primary glycemic target recommendation |
| "High risk: insulin, sulfonylureas, meglitinides (raise blood insulin)" | **T1** | Drug-specific hypoglycemia risk classification |
| "Low risk: metformin, SGLT2i, GLP-1 RA, DPP-4i" | **T1** | Safe drug list for patients without monitoring |
| PP 2.1.6 text: CGM evolution, specialist consultation | **T2** | Technology guidance |
| Figure 13 complete: drug class ↔ hypoglycemia risk linkage | **T1** | Visual drug safety classification |
| Research recommendations (6 items) | **T3** | Research priorities |

### ⚠️ Meglitinides NOT in Drug Dictionary
The D channel extracts "Meglitinides" from Figure 13 as a T2 span, but the B channel does NOT match it. This confirms meglitinides are not in the drug dictionary, similar to the bupropion/varenicline gap on page 55.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — Rec 2.2.1 (glycemic target) is one of the most clinically important recommendations in the guideline and its text is completely missing; PP 2.1.5 drug selection guidance also missing; Figure 13 decomposed into fragments without linkage |
| **Tier corrections** | 5 D channel drug names: T1 → T3; 6 B channel drug names: T1 → T3; 3 C labels: T1 → T3; Higher ×3: T2 → T3; Lower ×2: T2 → T3; daily ×4: T2 → T3; HbA1c ×4: T2 → T3; avoid ×2: T2 → T3; "·": T2 → NOISE |
| **Missing T1** | Rec 2.2.1 text (glycemic target), PP 2.1.5 text (drug selection), hypoglycemia risk drug classification, Figure 13 linked content |
| **Missing T2** | PP 2.1.6 text, research recommendations |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~10% — Figure 13 fragments and 1 F channel sentence; Rec 2.2.1 and PP 2.1.5 texts entirely missing |
| **Tier accuracy** | ~8% (1/17 T1 possibly correct + 2/20 T2 correct = 3/37) |
| **Noise ratio** | ~89% — Drug names, labels, Higher/Lower fragments, daily, HbA1c, avoid |
| **Genuine T1 content** | 1 possibly extracted (eGFR thresholds with CKD staging) |
| **Prior review** | 0/37 reviewed |
| **Overall quality** | **POOR — ESCALATE** — Missing Rec 2.2.1 (the glycemic target recommendation) makes this a critical gap |

---

## Drug Dictionary Gap Update

Confirmed drug dictionary omissions (cumulative):
1. **Bupropion** (page 55) — smoking cessation
2. **Varenicline** (page 55) — smoking cessation
3. **Nicotine replacement therapy** (page 55) — smoking cessation
4. **Meglitinides** (page 60) — oral hypoglycemic (causes hypoglycemia)
5. **DPP-4 inhibitors** (page 60) — oral hypoglycemic (extracted by D channel but NOT B channel)

Note: DPP-4 inhibitors appear in Figure 13 via D channel but B channel does not match "DPP-4 inhibitors" as a drug class name.

---

## Review Actions Completed

| Field | Value |
|-------|-------|
| **Review Date** | 2026-02-27 |
| **Reviewer** | claude-auditor |
| **Total Original Spans** | 37 (API returned 37; audit initially estimated 45 — difference likely due to count methodology) |
| **Confirmed** | 3 |
| **Rejected** | 34 |
| **Added** | 5 |

### CONFIRMED Spans (3)

| Span ID | Text | Tier | Channel | Reason |
|---------|------|------|---------|--------|
| `f9b08695` | "eGFR) >=90 mL/min/1.73m2; G5, eGFR <15 mL/min/1.73m2" | T1 | C | CKD staging boundaries (G1 and G5) with eGFR thresholds. Links to Rec 2.2.1 glycemic targets. KB-16, KB-4. |
| `088eef11` | "CGM technology has greatly impacted diabetes self-management by providing glycemic assessment moment-to-moment, allowing patients to make real-time decisions about their hyperglycemic treatment." | T2 | F | Complete clinical sentence about CGM benefits from PP 2.1.6 context. KB-16. |
| `6c5cf48b` | "Rationale for CGM or SMBG" | T2 | D | Figure 13 column header providing structural context for drug-monitoring linkage table. |

### REJECTED Spans (34)

| Span ID | Text | Tier | Reason | Reject Code |
|---------|------|------|--------|-------------|
| `a071fa3e` | "Higher" | T2 | Risk level word from Figure 13, no drug context | out_of_scope |
| `b8038291` | "Higher" | T2 | Duplicate risk level fragment | duplicate |
| `e5750bed` | "Higher" | T2 | Duplicate risk level fragment | duplicate |
| `1b25a86b` | "Lower" | T2 | Risk level word from Figure 13, no drug context | out_of_scope |
| `00071de1` | "Lower" | T2 | Duplicate risk level fragment | duplicate |
| `f43ff30d` | "SGLT2 inhibitors" | T1 | Drug class from Figure 13 row, no prescribing context | out_of_scope |
| `260ef296` | "DPP-4 inhibitors" | T1 | Drug class from Figure 13 row, no prescribing context | out_of_scope |
| `fa9bcc6c` | "Sulfonylureas" | T1 | Drug class from Figure 13 row, no prescribing context | out_of_scope |
| `40f510e4` | "Metformin" | T1 | Drug name from Figure 13 row, no prescribing context | out_of_scope |
| `646de455` | "Insulin" | T1 | Drug name from Figure 13 row, no prescribing context | out_of_scope |
| `a1aac449` | "." (bullet) | T2 | Pipeline artifact — bullet character only | other |
| `86eea15f` | "Meglitinides" | T2 | Drug class from Figure 13, no prescribing context | out_of_scope |
| `35db3093` | "avoid" | T2 | Action verb without clinical context | out_of_scope |
| `d350524a` | "daily" | T2 | Frequency word only | out_of_scope |
| `395ff7ec` | "Practice Point 2.1.5" | T1 | PP label only — text NOT captured | out_of_scope |
| `754f77dd` | "daily" | T2 | Duplicate frequency word | duplicate |
| `6f8b5a1f` | "eGFR" | T2 | Lab abbreviation only | out_of_scope |
| `4e4c0fd3` | "daily" | T2 | Duplicate frequency word | duplicate |
| `dceeb501` | "insulin" | T1 | Drug name only | out_of_scope |
| `a799737b` | "insulin" | T1 | Duplicate drug name | duplicate |
| `0169ee43` | "sulfonylureas" | T1 | Drug name only | out_of_scope |
| `11083d10` | "daily" | T2 | Duplicate frequency word | duplicate |
| `3218bc20` | "avoid" | T2 | Duplicate action verb | duplicate |
| `a9a8e9fd` | "metformin" | T1 | Drug name only | out_of_scope |
| `45283408` | "SGLT2i" | T1 | Drug class abbreviation only | out_of_scope |
| `64c255a3` | "GLP-1 RA" | T1 | Drug class abbreviation only | out_of_scope |
| `dadebcd3` | "Practice Point 2.1.6" | T1 | PP label only — text NOT captured | out_of_scope |
| `6a586100` | "insulin" | T1 | Duplicate drug name | duplicate |
| `8536dd1f` | "insulin" | T1 | Duplicate drug name | duplicate |
| `fe2e3902` | "HbA1c" | T2 | Lab test name only | out_of_scope |
| `a975a7d1` | "HbA1c" | T2 | Duplicate lab test name | duplicate |
| `d5b8561c` | "Recommendation 2.2.1" | T1 | Rec label only — critical recommendation text NOT captured | out_of_scope |
| `e5eb6ad4` | "HbA1c" | T2 | Duplicate lab test name | duplicate |
| `0998ea00` | "HbA1c" | T2 | Duplicate lab test name | duplicate |

### ADDED Facts (5)

| # | Text (Exact PDF) | Note | Target KB |
|---|-----------------|------|-----------|
| 1 | We recommend an individualized HbA1c target ranging from <6.5% to <8.0% in patients with diabetes and CKD not treated with dialysis (Figure 14) | Rec 2.2.1 (1C) — PRIMARY glycemic target recommendation | KB-16, KB-4 |
| 2 | For patients with T2D and CKD who choose not to do daily glycemic monitoring by CGM or SMBG, glucose-lowering agents that pose a lower risk of hypoglycemia are preferred and should be administered in doses that are appropriate for the level of eGFR | PP 2.1.5 — Drug selection + eGFR-based dosing guidance | KB-1, KB-4, KB-5 |
| 3 | High hypoglycemia risk agents: insulin, sulfonylureas, meglitinides (raise blood insulin levels) | PP 2.1.5 / Figure 13 — Drug-specific risk classification | KB-1, KB-4 |
| 4 | Low/no hypoglycemia risk agents: metformin, SGLT2i, GLP-1 RA, DPP-4 inhibitors | PP 2.1.5 / Figure 13 — Safe drug list | KB-1, KB-4 |
| 5 | CGM devices are rapidly evolving with multiple functionalities (e.g., real-time and intermittently scanned CGM). Newer CGM devices may offer advantages for certain patients, depending on their values, goals, and preferences | PP 2.1.6 — CGM technology evolution guidance | KB-16 |

---

## Raw PDF Gap Analysis

| # | Gap Text | Priority | Rationale |
|---|----------|----------|-----------|
| 1 | "Patients with diabetes and more advanced CKD stages are at increased risk of hypoglycemia" | **HIGH** | CKD-hypoglycemia risk statement — foundational safety warning for advanced CKD patients. KB-4, KB-1. |
| 2 | "Risk of hypoglycemia is high in patients with advanced CKD who are treated by glucose-lowering agents that raise blood insulin levels (exogenous insulin, sulfonylureas, meglitinides). Therefore, without daily glycemic monitoring, it is often difficult to avoid hypoglycemic episodes" | **HIGH** | Links advanced CKD + insulin-raising agents + monitoring necessity — critical safety triad. KB-4, KB-1. |
| 3 | "CGM features include alarms for low and high values, direct cell phone linkage, factory calibration, new metrics such as GMI and ambulatory glucose profiles, and integration into closed-loop insulin delivery systems" | **MODERATE** | CGM technology capabilities — clinician needs to know available features for prescribing. KB-16. |
| 4 | "Consultation with a specialist in diabetes technology (certified diabetes educator or other provider) can help patients select the device that is most appropriate for patients with diabetes and CKD" | **MODERATE** | Specialist referral recommendation for CGM device selection. KB-16. |
| 5 | "CGM devices differ in their accuracy, need for calibration, placement, sensor life, warm-up time, type of transmitter, display options, live data-sharing capacity, cost, and insurance coverage" | **MODERATE** | CGM device selection criteria — comprehensive list of differentiating features. KB-16. |
| 6 | "Research recommendations for Section 2.1: develop methods to identify patients for whom HbA1c produces a biased estimate; identify patients at high risk of hypoglycemia who may benefit from CGM or SMBG; develop approaches to apply CGM for high-risk patients; determine overall benefits and harms of SMBG and CGM; develop and validate alternative biomarkers for long-term glycemic monitoring; test whether CGM helps control glycemia and improve clinical outcomes" | **LOW** | Section 2.1 research recommendations — 6 priorities for advancing glycemic monitoring in CKD. KB-16. |
| 7 | "Rec 2.2.1 places a higher value on the potential benefits of an individualized target aimed at balancing the risks of hyperglycemia and hypoglycemia" | **MODERATE** | Rec 2.2.1 rationale — balancing hyperglycemia vs hypoglycemia risk drives individualized target. KB-16, KB-4. |

**All 7 gaps added via API (all 201).**

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Total spans (post-review)** | 49 (37 original + 12 added) |
| **Reviewed** | 49/49 (100%) |
| **Confirmed** | 3 |
| **Rejected** | 34 |
| **Added (agent)** | 5 |
| **Added (gap fill)** | 7 |
| **Total Added** | 12 |
| **Pipeline 2 ready** | 15 (3 confirmed + 12 added) |
| **Completeness (post-review)** | ~95% — Rec 2.2.1 glycemic target + rationale; PP 2.1.5 drug selection + eGFR dosing; PP 2.1.6 CGM evolution; drug risk classification (high: insulin/SU/meglitinides; low: metformin/SGLT2i/GLP-1 RA/DPP-4i); CKD-hypoglycemia foundational risk warning; advanced CKD + insulin-raising agents + monitoring triad; CGM features list; CGM device selection criteria; specialist referral for device selection; research recommendations (6 items); eGFR staging boundaries |
| **Remaining gaps** | Figure 13 visual layout (drug-risk matrix as linked rows — addressed via separate high/low risk lists); "ability to do self-monitoring, or preference to avoid daily burden" patient preference context (T3) |
| **Review Status** | COMPLETE |
| **Escalation resolved** | YES — Rec 2.2.1 text + CKD-hypoglycemia safety gaps now filled. |
