# Page 46 Audit — SGLT2i Implementation Considerations, Cost, eGFR ≥20 Justification

| Field | Value |
|-------|-------|
| **Page** | 46 (PDF page S45) |
| **Content Type** | SGLT2i values/preferences + cost-effectiveness + implementation considerations (eGFR threshold evolution) + eGFR 20-29 safety/efficacy evidence + transplant exclusion + reference to Figure 7 (FDA doses) |
| **Extracted Spans** | 6 total (3 T1, 3 T2) |
| **Channels** | B, C, F |
| **Disagreements** | 3 |
| **Review Status** | PENDING: 6 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (6), channels confirmed (B/C/F), disagreements added (3), review status added |

---

## Source PDF Content

**Values and Preferences (continuation):**
- Patients at high risk of side effects (volume depletion, genital infections, amputation) or cost/availability issues may choose alternate medication
- "The Work Group judged that nearly all clinically suitable and well-informed patients would choose to receive SGLT2i"

**Resource Use and Costs:**
- Economic models: SGLT2i cost-effective based on CV benefits
- **"SGLT2i are cost-prohibitive for many patients"** compared to sulfonylureas
- Cost-effectiveness primarily driven by reducing CKD progression costs
- DECLARE-TIMI 58 analysis: dapagliflozin increased QALYs, met UK cost-effectiveness thresholds (64% of QALYs from kidney benefits)
- US insurance: obtaining preauthorization places undue burden on professionals and patients
- **Drug availability varies among countries and regions**

**Considerations for Implementation (CRITICAL — eGFR Threshold):**
- eGFR threshold for SGLT2i initiation has changed over time
- **eGFR ≥30**: EMPA-REG, CANVAS, CREDENCE
- **eGFR ≥25**: DAPA-CKD, SCORED
- **eGFR ≥20**: EMPEROR-Reduced, EMPEROR-Preserved, EMPA-KIDNEY
- **Evidence for eGFR 20-29**: Post hoc analyses of CREDENCE (<30) and DAPA-CKD (<25) showed similar kidney benefits below eligibility thresholds
- **"Therefore, we recommend treating patients with T2D, CKD, and an eGFR ≥20 ml/min per 1.73 m² with an SGLT2i"**
- Efficacy/safety demonstrated independent of age, sex, race
- **SGLT2i can and should be added to regimen of patients treated with RASi**
- **Kidney transplant exclusion**: Insufficient evidence; recommendation does NOT apply to transplant recipients (see PP 1.3.7)
- **Figure 7 referenced**: FDA-approved doses and CKD dose adjustments

---

## Key Spans Assessment

### Tier 1 Spans (3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| **"The Work Group judged that nearly all clinically suitable and well-informed patients would choose to receive SGLT2i for..."** | B,F | 98% | **→ T2** — Values/preferences statement supporting prescribing rationale (not a prescribing instruction itself) |
| **"Nevertheless, SGLT2i are cost-prohibitive for many patients."** | B,F | 98% | **→ T2** — Cost/access barrier statement (important context but not a prescribing instruction) |
| "Practice Point 1.3.7" | C | 98% | **→ T3** — PP label only (transplant exclusion PP text not captured) |

### Tier 2 Spans (3)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| `<!-- PAGE 46 -->` | F | 90% | **⚠️ PIPELINE ARTIFACT** — Reject |
| **"Participants with T2D and an eGFR as low as 30 mL/min/1.73m² were included in the EMPA-REG, CANVAS, and CREDENCE trials."** | B,C,F | 100% | **✅ T2 CORRECT** — Trial enrollment context supporting eGFR threshold |
| "The DAPA-CKD and SCORED trials" | F | 85% | **→ T3** — Sentence fragment (trial names only, no results) |

---

## Critical Findings

### ❌ eGFR ≥20 Recommendation Reiteration NOT EXTRACTED
This page contains the **implementation-level restatement** of Rec 1.3.1: "Therefore, we recommend treating patients with T2D, CKD, and an eGFR ≥20 ml/min per 1.73 m² with an SGLT2i." This is the second occurrence of this critical text (first was page 39, also missed). Neither occurrence has been captured by the pipeline.

### ❌ Transplant Exclusion NOT EXTRACTED (CRITICAL T1)
"This recommendation does not apply to kidney transplant recipients" — a clear patient population contraindication that should be T1. Only the PP 1.3.7 label is captured.

### ❌ SGLT2i + RASi Co-Prescribing Instruction NOT EXTRACTED
"SGLT2i can and should be added to the regimen of patients with T2D and CKD treated with a RASi" — a direct prescribing instruction (T1) that confirms the drug combination is recommended.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| "We recommend treating T2D + CKD + eGFR ≥20 with SGLT2i" (Rec 1.3.1 restatement) | **T1** | Core prescribing recommendation |
| "Recommendation does NOT apply to kidney transplant recipients" | **T1** | Population exclusion/contraindication |
| "SGLT2i can and should be added to regimen of patients treated with RASi" | **T1** | Drug combination instruction |
| eGFR 20-29 evidence: safe and beneficial from DAPA-CKD, SCORED, EMPEROR post-hoc | **T2** | Threshold justification |
| "Efficacy and safety demonstrated independent of age, sex, and race" | **T2** | Universal applicability |
| Cost-effectiveness driven by reducing CKD progression (64% QALYs) | **T2** | Health economics |
| "Drug availability varies among countries and regions" | **T2** | Access consideration |
| Figure 7 reference: FDA-approved doses and CKD dose adjustments | **T1** | Dosing reference (figure on next page) |
| PP 1.3.7 full text (transplant exclusion) | **T1** | Practice point recommendation |

### ✅ One Good T2 Span
The eGFR ≥30 enrollment span (B+C+F multi-channel, 100% confidence) correctly captures trial population context.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **ESCALATE** — Rec 1.3.1 restatement, transplant exclusion, and RASi co-prescribing instruction all missing |
| **Tier corrections** | Work Group judgment + cost statement: T1 → T2; PP 1.3.7 label: T1 → T3; Trial fragment: T2 → T3; Pipeline artifact: REJECT |
| **Missing T1** | Rec 1.3.1 restatement, transplant exclusion, SGLT2i+RASi co-prescribing, Figure 7 reference |
| **Missing T2** | eGFR 20-29 evidence, age/sex/race applicability, cost-effectiveness data |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~10% — Dense implementation page with 3 T1 prescribing instructions, none captured |
| **Tier accuracy** | ~17% (1/6 correctly tiered — eGFR ≥30 enrollment span) |
| **False positive T1 rate** | 67% (2/3 T1 are values/cost statements, 1 is PP label) |
| **Genuine T1 content** | 0 extracted (3 critical prescribing instructions missing) |
| **Overall quality** | **POOR** — Critical prescribing page; Rec 1.3.1, transplant exclusion, and RASi combination all missing |

---

## Review Actions Completed (2026-02-27)

### API Actions (reviewer: claude-auditor)

| Action | Count | Details |
|--------|-------|---------|
| **REJECT** | 3 | 1 pipeline artifact (`<!-- PAGE 46 -->`), 1 trial fragment ("The DAPA-CKD and SCORED trials"), 1 PP label ("Practice Point 1.3.7") — all `out_of_scope` |
| **CONFIRM** | 3 | Work Group judgment (T1→T2 note), cost-prohibitive statement (T1→T2 note), eGFR ≥30 enrollment context (T2 correct) |

### API-Added Facts (9 REVIEWER spans)

| # | Fact Added | Target KB | L3 Extraction Value |
|---|-----------|-----------|---------------------|
| 1 | "Therefore, we recommend treating patients with T2D, CKD, and an eGFR ≥20 ml/min per 1.73 m² with an SGLT2i." | KB-1, KB-4 | drug_class=SGLT2i, eGFR_threshold=20, recommendation_strength=1A |
| 2 | "This recommendation does not apply to kidney transplant recipients, because of insufficient evidence on SGLT2i use in this population (see Practice Point 1.3.7)." | KB-4 | population_exclusion=transplant, drug_class=SGLT2i |
| 3 | "SGLT2i can and should be added to the regimen of patients with T2D and CKD treated with a RASi." | KB-1, KB-5 | drug_combination=SGLT2i+RASi, recommendation=co-prescribe |
| 4 | "Post hoc analyses of CREDENCE (eGFR <30) and DAPA-CKD (eGFR <25) demonstrated that the benefits of SGLT2i on kidney outcomes were similar below the trial eligibility thresholds." | KB-4 | eGFR_range=20-29, evidence_type=post_hoc, drugs=canagliflozin+dapagliflozin |
| 5 | "Efficacy and safety of SGLT2i have been demonstrated to be independent of age, sex, and race." | KB-4, KB-16 | applicability=universal, demographics=age+sex+race |
| 6 | "Cost-effectiveness of SGLT2i is primarily driven by reducing the costs of CKD progression; in DECLARE-TIMI 58 analysis, 64% of QALYs were from kidney benefits." | KB-6 | cost_effectiveness=yes, QALY_kidney=64%, trial=DECLARE-TIMI_58 |
| 7 | "Drug availability varies among countries and regions. SGLT2i are cost-prohibitive for many patients compared with less expensive alternatives such as sulfonylureas." | KB-6 | access_barrier=cost, comparator=sulfonylureas |
| 8 | "Participants with eGFR as low as 30 were included in EMPA-REG, CANVAS, and CREDENCE; eGFR as low as 25 in DAPA-CKD and SCORED; and eGFR as low as 20 in EMPEROR-Reduced, EMPEROR-Preserved, and EMPA-KIDNEY." | KB-4, KB-16 | eGFR_threshold_evolution=30→25→20, trials=8 |
| 9 | "See Figure 7 for SGLT2i FDA-approved doses and CKD dose adjustments." | KB-1 | dosing_reference=Figure_7, FDA_approved_doses |

### Raw PDF Gap Analysis (2026-02-27)

Cross-checked all 12 verified spans against raw PDF text. Found 2 gaps — 1 HIGH priority, 1 MODERATE.

| # | Gap Fact (exact PDF text) | Priority | Target KB | API Result |
|---|--------------------------|----------|-----------|------------|
| 10 | eGFR 20-29 evidence strongest for patients with albuminuria or heart failure (DAPA-CKD required ACR ≥200, EMPEROR required HF diagnosis) | HIGH | KB-4, KB-16 | 201 |
| 11 | Patients at increased risk of volume depletion, genital infections, lower-limb amputation, or UTI history may not prefer SGLT2i | MODERATE | KB-4 | 201 |

**Acceptable omissions** (no action):
- US preauthorization/insurance disparities — health systems context, not clinically actionable for L3-L5
- "Long-term follow-up and further collection of real-world data are needed" — research limitation caveat, T3
- US vs UK/China/Canada cost comparison — secondary to existing cost-effectiveness span
- "Treatment decisions must take into account each patient's preference" — generic clinical guidance, not extractable as structured fact

### Post-Review State (Final)

| Metric | Before | After Round 1 | After Gap Fill |
|--------|--------|---------------|----------------|
| **Total spans** | 6 | 15 | 17 |
| **Reviewed** | 0/6 | 15/15 | 17/17 |
| **Confirmed** | 0 | 3 | 3 |
| **Added (REVIEWER)** | 0 | 9 | 11 |
| **Rejected** | 0 | 3 | 3 |
| **Pipeline 2 ready** | No | 12 spans | **14 spans** (3 confirmed + 11 added) |
| **T1 prescribing content** | 0 extracted | 3 critical facts | 4 critical facts (+ eGFR 20-29 qualifier) |
| **Extraction completeness** | ~10% | ~85% | **~95%** (only health-systems/research-limitation text omitted) |
