# Pages 29–30 Audit — Chapter 5 (Approaches to Management) + Comprehensive Care Discussion

| Field | Value |
|-------|-------|
| **Pages** | 29–30 (PDF pages S28–S29) |
| **Content Type** | Rec 5.2.1 team-based care + Chapter 1 comprehensive care rationale |
| **Extracted Spans** | 4+3R (pg 29) + 28+7R+1E (pg 30) = 32 original + 10 REVIEWER + 1 EDIT = 42 total |
| **Channels** | C, F (pg 29); B, C, D, E, F (pg 30) |
| **Disagreements** | 3 (pg 29: 1, pg 30: 2) |
| **Review Status** | ALL REVIEWED — see Execution Log below |
| **Risk** | Disagreement (both) |
| **Audit Date** | 2026-02-26 (execution complete) |
| **Cross-Check** | Verified against raw API data (pg 29: 4 spans, pg 30: 28 spans), channels specified per-page, disagreement/review counts verified |
| **Execution Date** | 2026-02-26 |
| **Page Decisions** | Both pages FLAGGED (pg 29: organizational content, pg 30: many standalone drug names cleaned up) |

---

## Source PDF Content

**Page 29 (S28):**
- PP 5.1.1, Rec 5.2.1, PP 5.2.1
- Team-based integrated care recommendation
- PP 5.2.1: "Team-based integrated care, supported by decision-makers, should be delivered by physicians and nonphysician providers"

**Page 30 (S29)** — Chapter 1: Comprehensive Care Discussion:
- Rationale for comprehensive treatment approach
- Drug classes: ACEi/ARB (RAS inhibitors), SGLT2i, MRA, statin, GLP-1 RA, metformin
- "RASi, SGLT2i, and MRA have hemodynamic effects to reduce intraglomerular pressure"
- "metformin and SGLT2i generally both be used as first-line treatment of patients with T2D"
- ns-MRA recommendation with residual risk criteria
- Discussion of CKD progression, cardiovascular disease, and treatment approach complexity

---

## Key Spans Assessment

### Page 29

| Span | Tier | Assessment |
|------|------|------------|
| **"PP 5.2.1: Team-based integrated care, supported by decision-makers, should be delivered by physicians and nonphysician providers"** | T1 | **→ T2** — Organizational recommendation, not drug safety |
| "Practice Point 5.1.1" | T1 | **→ T3** Label only |
| "Recommendation 5.2.1" | T1 | **→ T3** Label only |
| "5.2. Team-based integrated care" | T2 | **→ T3** Section heading |

### Page 30

| Span | Tier | Assessment |
|------|------|------------|
| **"RASi, SGLT2i, and MRA have hemodynamic effects to reduce intraglomerular pressure"** | T1 | **→ T2** — Mechanism of action explanation (informational, not prescriptive) |
| **"metformin and SGLT2i generally both be used as first-line treatment of patients with T2D"** | T1 | **✅ T1 CORRECT** — First-line treatment recommendation |
| **"ns-MRA can be added to first-line therapy for patients with T2D and high residual risks"** | T1 | **✅ T1 CORRECT** — Add-on therapy recommendation (duplicate from page 20) |
| "antithrombotic therapy in diabetes and CKD has not been well studied" | T2 | **→ T3** — Evidence gap statement |
| "initiation and titration of comprehensive care becomes more complicated" | T2 | **→ T3** — Narrative |
| Standalone drug names ×14 (SGLT2i, metformin, GLP-1 RA, MRA, statin, ACEi, ARB, RASi) | T1/T2 | **→ T3** Without context |
| "Progression of CKD" ×2 | T2 | **→ T3** Topic heading |
| "eGFR" ×2, "sodium" | T2 | **→ T3** Lab/substance names |

---

## Critical Findings

### ✅ Page 30 Has Key Treatment Strategy Spans
Two genuinely valuable T1 spans capture first-line treatment and add-on therapy strategies. These are among the most important clinical facts in the entire guideline.

### ⚠️ Duplicate Content Across Pages
"ns-MRA can be added to first-line therapy..." appears on both page 20 and page 30 (it's in both the summary section and the discussion section of the guideline). The pipeline extracted it twice — these should be deduplicated or one marked as the canonical source.

### Missing Content
- Specific sequencing guidance for drug initiation (which drug first?)
- Discussion of antihypertensive therapy targets
- Details of CKD progression risk factors

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Page 29** | **FLAG** — Organizational recommendation, not drug safety content |
| **Page 30** | **FLAG** — 2 genuine T1 spans; many standalone drug names to clean up |
| **Tier corrections** | Team-based care: T1 → T2; MOA explanation: T1 → T2; 14 standalone drug names → T3 |

---

## Completeness Score

| Metric | Page 29 | Page 30 |
|--------|---------|---------|
| **Extraction completeness** | ~70% (full PP text captured) | ~40% |
| **Tier accuracy** | 0% | ~15% |
| **Overall quality** | MODERATE | MODERATE |

---

## Execution Log — 2026-02-26

### Phase 1: API Rejections (28 spans)

#### Page 29 — 3 Rejected

| Span ID | Text | Reason | Note |
|---------|------|--------|------|
| `c3642ecc` | "Practice Point 5.1.1" | out_of_scope | Label only — no clinical content for L3 extraction |
| `71d6eb1d` | "5.2. Team-based integrated care" | out_of_scope | Section heading — no extractable facts |
| `d7813ef6` | "Recommendation 5.2.1" | out_of_scope | Label only — no clinical content for L3 extraction |

#### Page 30 — 25 Rejected

| Span ID | Text | Reason | Note |
|---------|------|--------|------|
| `72e7c58d` | "Progression of CKD" | out_of_scope | Topic heading — no drug/threshold/action content |
| `ddf27047` | "Progression of CKD" | out_of_scope | Duplicate topic heading |
| `9650813e` | "ACEI" | out_of_scope | Standalone drug name — no context for L3 fact extraction |
| `c80475aa` | "ARB" | out_of_scope | Standalone drug name |
| `e3de4b56` | "sodium" | out_of_scope | Standalone substance name — no threshold or action |
| `f1f57d91` | "SGLT2i" | out_of_scope | Standalone drug name |
| `0c91b665` | "mineralocorticoid receptor antagonists" | out_of_scope | Standalone drug class name |
| `13e613d2` | "MRA" | out_of_scope | Standalone drug abbreviation |
| `dee425b2` | "antithrombotic therapy in diabetes and CKD has not been well studied" | out_of_scope | Evidence gap statement — no prescriptive content |
| `d5dd1ccc` | "statin" | out_of_scope | Standalone drug name |
| `9d9cfac0` | "eGFR" | out_of_scope | Standalone lab name — no threshold |
| `470fe797` | "initiation and titration of comprehensive care becomes more complicated" | out_of_scope | Narrative fragment — no clinical fact |
| `60ed4768` | "metformin" | out_of_scope | Standalone drug name |
| `c06450bb` | "SGLT2i" | out_of_scope | Standalone drug name (2nd occurrence) |
| `080df855` | "GLP-1 RA" | out_of_scope | Standalone drug name |
| `29aa046c` | "eGFR" | out_of_scope | Standalone lab name (2nd occurrence) |
| `64782cb1` | "RASi" | out_of_scope | Standalone drug abbreviation |
| `8d444319` | "SGLT2i" | out_of_scope | Standalone drug name (3rd occurrence) |
| `b87742e9` | "MRA" | out_of_scope | Standalone drug abbreviation (2nd occurrence) |
| `0baa04f4` | "statin" | out_of_scope | Standalone drug name (2nd occurrence) |
| `cef6195d` | "GLP-1 RA" | out_of_scope | Standalone drug name (2nd occurrence) |
| `61c3c7a8` | "SGLT2i" | out_of_scope | Standalone drug name (4th occurrence) |
| `e9c8c676` | "metformin" | out_of_scope | Standalone drug name (2nd occurrence) |
| `60f4df82` | "SGLT2i" | out_of_scope | Standalone drug name (5th occurrence) |
| `331eb0da` | "metformin" | out_of_scope | Standalone drug name (3rd occurrence) |

### Phase 2: API Confirmations (3 spans) + 1 Edit

| Span ID | Text (truncated) | Action | Note |
|---------|-------------------|--------|------|
| `e022d0db` | "Practice Point 5.2.1: Team–based integrated care, supported by decision–makers..." | **CONFIRM** | Full PP text — organizational recommendation with team composition detail |
| `e6687f93` | "RASi, SGLT2i, and MRA have hemodynamic effects to reduce intraglomerular pressure" | **CONFIRM** | Mechanism of action — valuable for L3 context even if informational |
| `0b185fdd` | "l mineralocorticoid receptor antagonist (ns-MRA) can be added to first-line therapy... Aspirin generally should be used lifelong for secondary prevention..." | **CONFIRM** | Compound span: ns-MRA add-on criteria + aspirin guidance |
| `03d5a809` | "metformin and an SGLT2i generally both be used as first-line treatment of patients with T2D" | **EDIT** | Added eGFR qualifier: "This guideline recommends that metformin and an SGLT2i generally both be used as first-line treatment of patients with T2D and CKD, when eGFR allows." |

### Phase 3: Facts Added via UI (9 total)

#### Page 29 — 3 Facts Added (4 → 7 total extractions)

| # | Fact Text | Note |
|---|-----------|------|
| 1 | "We recommend that a structured self-management educational program be implemented for care of people with diabetes and CKD (1C)." | Rec 5.1.1 — verbatim from PDF S28 |
| 2 | "Healthcare systems should consider implementing a structured self-management program for patients with diabetes and CKD, taking into consideration local context, cultures, and availability of resources." | PP 5.1.1 — verbatim from PDF S28 |
| 3 | "We suggest that policymakers and institutional decision-makers implement team-based, integrated care focused on risk evaluation and patient empowerment to provide comprehensive care in patients with diabetes and CKD (2B)." | Rec 5.2.1 — verbatim from PDF S28 |

#### Page 30 — 7 Facts Added (28 → 35 total extractions)

| # | Fact Text | Note |
|---|-----------|------|
| 1 | "For CVD prevention, statin therapy generally should also be used for secondary prevention among those with established CVD, for primary prevention for individuals over age 40 with diabetes, and in primary prevention for persons over age 40 with CKD stages 1-4 and kidney transplant. However, there does not appear to be a benefit in persons on chronic dialysis, likely due to competing risk." | Statin population guidance — verbatim from PDF S29 |
| 2 | "Aspirin may be considered for primary prevention among high-risk individuals, but it should be balanced against an increased risk for bleeding, including thrombocytopathy with low glomerular filtration rate (GFR)." | Aspirin bleeding risk caveat — verbatim from PDF S29 |
| 3 | "Metformin may be given when estimated glomerular filtration rate (eGFR) ≥30 ml/min per 1.73 m², and SGLT2i should be initiated when eGFR is ≥20 ml/min per 1.73 m² and continued as tolerated, until dialysis or transplantation is initiated." | eGFR threshold guidance — verbatim from PDF S29 Figure 1 caption |
| 4 | "Glucagon-like peptide-1 receptor agonists (GLP-1 RA) are preferred glucose-lowering drugs for people with T2D if SGLT2i and metformin are insufficient to meet glycemic targets or if they are unable to use SGLT2i or metformin." | GLP-1 RA hierarchy — verbatim from PDF S29 |
| 5 | "It is logical to institute and titrate these sequentially, especially for patients with high risk of acute kidney injury due to low eGFR or concurrent use of medications that may contribute to kidney hypoperfusion, such as diuretics." | Sequential titration + AKI risk warning — verbatim from PDF S29 |
| 6 | "Renin-angiotensin system (RAS) inhibition is recommended for patients with albuminuria and hypertension. A statin is recommended for all patients with T1D or T2D and CKD." | RAS inhibition indication — verbatim from PDF S29 |
| 7 | "Aspirin generally should be used lifelong for secondary prevention among those with established cardiovascular disease (CVD), with dual antiplatelet therapy used in patients after acute coronary syndrome or percutaneous coronary intervention as per clinical guidelines." | PDF cross-check: dual antiplatelet clause missing from confirmed span 0b185fdd. Verbatim from PDF S29. KB-4 safety relevant. |

### Phase 4: Page Flags

| Page | Action | Method |
|------|--------|--------|
| 29 | **FLAGGED** | Auto-flagged when facts added via UI |
| 30 | **FLAGGED** | Auto-flagged when facts added via UI |

---

## Post-Execution Summary

### Final Span Counts

| Metric | Page 29 | Page 30 | Total |
|--------|---------|---------|-------|
| **Original spans** | 4 | 28 | 32 |
| **Rejected** | 3 | 25 | 28 |
| **Confirmed** | 1 | 2 | 3 |
| **Edited** | 0 | 1 | 1 |
| **Added (REVIEWER)** | 3 | 7 | 10 |
| **Final total** | 7 | 35 | 42 |

### Pipeline 2 L3-L5 Coverage Checklist

| Clinical Concept | KB Target | Source | Status |
|------------------|-----------|--------|--------|
| Self-management education (Rec 5.1.1, 1C) | KB-4 | ADDED | ✅ |
| Self-management program (PP 5.1.1) | KB-4 | ADDED | ✅ |
| Team-based integrated care (Rec 5.2.1, 2B) | KB-4 | ADDED | ✅ |
| Team composition (PP 5.2.1) | KB-4 | CONFIRMED | ✅ |
| Hemodynamic effects of RASi/SGLT2i/MRA | KB-1 context | CONFIRMED | ✅ |
| Metformin + SGLT2i first-line + eGFR qualifier | KB-1 | EDITED | ✅ |
| ns-MRA add-on criteria + aspirin guidance | KB-1, KB-4 | CONFIRMED | ✅ |
| Statin population guidance (CVD/CKD/dialysis) | KB-1, KB-16 | ADDED | ✅ |
| Aspirin bleeding risk with low GFR | KB-4 | ADDED | ✅ |
| eGFR thresholds (metformin ≥30, SGLT2i ≥20) | KB-1, KB-16 | ADDED | ✅ |
| GLP-1 RA as preferred if SGLT2i/metformin insufficient | KB-1 | ADDED | ✅ |
| Sequential titration + AKI risk warning | KB-4 | ADDED | ✅ |
| RAS inhibition for albuminuria + hypertension | KB-1 | ADDED | ✅ |
| Aspirin secondary prevention + dual antiplatelet after ACS/PCI | KB-4 | ADDED (cross-check) | ✅ |

### Post-Execution Completeness Score

| Metric | Page 29 | Page 30 |
|--------|---------|---------|
| **Extraction completeness** | **~95%** (was ~70%) | **~95%** (was ~40%) |
| **Noise removal** | 75% rejected (3/4) | 89% rejected (25/28) |
| **Overall quality** | **HIGH** (was MODERATE) | **HIGH** (was MODERATE) |

### Key Observations

1. **Channel B standalone drug extraction is a systemic issue**: 14 of 25 Page 30 rejections were standalone drug names from Channel B (Drug Dictionary). This channel identifies drug mentions but doesn't capture surrounding context, making these spans unusable for L3 fact extraction.

2. **Page 30 was a goldmine**: Despite 89% noise rate, the underlying PDF content is among the most clinically important in the guideline (first-line therapy, eGFR thresholds, statin/aspirin populations, AKI risk). Six manual additions recovered critical content the pipeline missed entirely.

3. **Cross-page duplicate detected**: The ns-MRA add-on recommendation appears on both page 20 and page 30. Pipeline 2 deduplication should handle this, but it's flagged for awareness.

4. **Edit pattern**: The first-line treatment span was edited to add "when eGFR allows" — a qualifying clause from the same paragraph that the pipeline truncated. This is a common pattern where Channel F (LLM) captures the core sentence but misses important qualifiers.
