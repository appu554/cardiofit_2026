# Page 34 Audit — Rec 1.2.1 Cont. (Safety, Pregnancy Warning, PP 1.2.1)

| Field | Value |
|-------|-------|
| **Page** | 34 (PDF page S33) |
| **Content Type** | Rec 1.2.1 evidence continuation + implementation considerations + PP 1.2.1 |
| **Extracted Spans** | 9 original + 9 REVIEWER = 18 total |
| **Channels** | C, D, F + REVIEWER |
| **Disagreements** | 2 |
| **Review Status** | REJECTED: 7, CONFIRMED: 2 (pre-existing), ADDED: 9 — 18/18 reviewed |
| **Page Decision** | **FLAGGED** |
| **Risk** | 9 prescriptive facts missing from pipeline; pregnancy contraindication absent |
| **Audit Date** | 2026-02-25 (pre-audit) → 2026-02-26 (executed) |
| **Execution** | pharma@vaidshala.com — 7 API rejections + 9 UI additions |

---

## Source PDF Content

**Values and Preferences:**
- CKD progression to kidney failure critically important to patients
- Side effects of ACEi/ARB acceptable to majority

**Considerations for Implementation (CRITICAL SAFETY):**
- "ACEi and ARBs are potent medications and can cause **hypotension, hyperkalemia, and a rise in serum creatinine**"
- Renal artery stenosis risk → hyperkalemia and creatinine rise
- "blood pressure, serum potassium, and serum creatinine should be **monitored** in patients who are started on RAS blockade or whenever there is a change in the dose"
- Changes usually reversible if medication stopped or doses reduced
- **Figure 3 reference**: ACEi/ARB starting and maximum doses (dose adjustment with declining kidney function)
- **PREGNANCY WARNING**: "The use of ACEi and ARB treatment has been associated with an increased risk of adverse effects to the fetus during pregnancy. Women who are planning for pregnancy or who are pregnant while on RAS blockade treatment should have the drug discontinued"
- ACEi-induced cough affects ~10% of patients → switch to ARB

**Practice Point 1.2.1:**
> "For patients with diabetes, albuminuria, and normal blood pressure, treatment with an ACEi or ARB may be considered"

**Rationale:**
- Dose titration: "start at a low dose and then up-titrate to the highest tolerated and recommended dose"

---

## Execution Log — 2026-02-26

### Phase 1: API Rejections (7/9)

| # | Span ID | Channel | Conf | Text | Reason |
|---|---------|---------|------|------|--------|
| 1 | b90abfcf | C | 98% | "Practice Point 1.2.1" | T1 label only — PP text missing |
| 2 | fddc3381 | D | 92% | "Kidney outcomes" | Evidence table section label |
| 3 | cad04db4 | D | 92% | "Cardiovascular mortality" | Evidence table section label |
| 4 | 78a8fad6 | D | 92% | "Resource use and costs" | Evidence table section label |
| 5 | 22ce1479 | D | 92% | "Kidney outcomes" | Evidence table section label (duplicate) |
| 6 | 8be3dc95 | D | 92% | "Kidney outcomes" | Evidence table section label (duplicate) |
| 7 | db213ff5 | F | 90% | `<!-- PAGE 34 -->` | Pipeline HTML artifact |

### Phase 2: Pre-Existing Confirmations (2/9)

These spans were already CONFIRMED by a prior reviewer and were left as-is:

| # | Span ID | Channel | Conf | Text (truncated) | Assessment |
|---|---------|---------|------|-------------------|------------|
| 1 | 88eea49c | F | 85% | "The progression of CKD to kidney failure, the avoidance or delay in initiating dialysis therapy..." | ✅ T2 OK — Values/preferences narrative |
| 2 | 8b2024fd | C | 85% | "Consequently, blood pressure, serum potassium, and serum creatinine should be monitored..." | ✅ T2 OK — Drug monitoring instruction (should be T1 but tier correction not available via API) |

### Phase 3: REVIEWER-Added Facts (9 via UI)

| # | Fact Text (truncated) | Note | Target KB |
|---|----------------------|------|-----------|
| 1 | "ACEi and ARBs are potent medications and can cause hypotension, hyperkalemia, and a rise in serum creatinine." | Adverse effects triad — hypotension, hyperkalemia, creatinine rise. All 3 channels missed this complete sentence. Verbatim from PDF S33. | KB-4 safety |
| 2 | "Women who are planning for pregnancy or who are pregnant while on RAS blockade treatment should have the drug discontinued." | Pregnancy contraindication — mandatory drug discontinuation. BLACK BOX level safety fact. Per-page completeness (also on P33). Verbatim from PDF S33. | KB-4 safety |
| 3 | "Practice Point 1.2.1: For patients with diabetes, albuminuria, and normal blood pressure, treatment with an ACEi or ARB may be considered." | Practice Point 1.2.1 — normotensive patients with albuminuria. Per-page completeness (also on P33). Verbatim from PDF S33. | KB-1 dosing eligibility |
| 4 | "Start at a low dose and then up-titrate to the highest tolerated and recommended dose." | Dose titration instruction for ACEi/ARB. Per-page completeness (also on P33). Verbatim from PDF S33 Rationale section. | KB-1 dosing |
| 5 | "ACEi-induced cough is the predominant symptom of intolerance to this class of drug, affecting about 10% of patients. In clinical practice, affected patients are often switched to an ARB so as not to lose the kidney protective effects of RAS blockade." | ACEi cough (~10%) with ARB switching guidance. Per-page completeness (also on P33). Verbatim from PDF S33. | KB-4 safety |
| 6 | "The inhibition of aldosterone action and its effect on efferent arteriole dilatation could result in hyperkalemia and a rise in serum creatinine in patients with renal artery stenosis." | Renal artery stenosis risk — hyperkalemia and creatinine rise mechanism with ACEi/ARB. Safety consideration for specific population. Verbatim from PDF S33. | KB-4 safety |
| 7 | "This recommendation applies to patients with either T1D and T2D, as well as kidney transplant recipients; however, this recommendation does not apply to patients on dialysis." | Population scope (T1D, T2D, kidney transplant) + dialysis exclusion. Per-page completeness (also on P33). Verbatim from PDF S33. | KB-1 dosing scope + KB-4 safety |
| 8 | "The evidence does not demonstrate superior efficacy of ACEi over ARB treatment or vice versa, and the choice between these 2 drug classes will depend on other factors, including patient preferences, cost, availability of generic formulations, and side-effects profiles of individual drugs." | ACEi vs ARB therapeutic equivalence. Per-page completeness (also on P33). Verbatim from PDF S33. | KB-1 dosing + KB-4 safety |
| 9 | "The changes in blood pressure, potassium, and kidney function are usually reversible if medication is stopped or doses are reduced." | Reversibility reassurance — adverse effects reversible on dose reduction/stop. Informs clinical decision to adjust rather than permanently discontinue. Verbatim from PDF S33. | KB-4 safety |

---

## Coverage Checklist

| Content | Covered | Source |
|---------|---------|--------|
| Adverse effects triad (hypotension, hyperkalemia, creatinine rise) | ✅ | REVIEWER Fact 1 |
| Pregnancy warning / fetal risk / discontinuation | ✅ | REVIEWER Fact 2 |
| PP 1.2.1 (normotensive + albuminuria → ACEi/ARB) | ✅ | REVIEWER Fact 3 |
| Dose titration (start low, up-titrate to max) | ✅ | REVIEWER Fact 4 |
| ACEi cough (~10%) → ARB switching | ✅ | REVIEWER Fact 5 |
| Renal artery stenosis risk | ✅ | REVIEWER Fact 6 |
| Values/preferences (CKD progression importance) | ✅ | Pre-existing CONFIRMED (F channel) |
| Monitoring instruction (BP, potassium, creatinine) | ✅ | Pre-existing CONFIRMED (C channel) |
| T1D/T2D + kidney transplant scope + dialysis exclusion | ✅ | REVIEWER Fact 7 |
| ACEi vs ARB therapeutic equivalence | ✅ | REVIEWER Fact 8 |
| Reversibility of adverse effects on dose reduction/stop | ✅ | REVIEWER Fact 9 |
| Figure 3 ACEi/ARB dosing table reference | ❌ | Table content — deferred to Pipeline 2 L3 extraction |

---

## Critical Findings

### Pipeline Performance — 78% False Positive Rate
- **7/9 pipeline spans** were noise (evidence table labels, HTML artifact, PP label without text)
- **2/9 genuine pipeline spans** survived (pre-existing CONFIRMED) — both are clinically meaningful sentences
- D channel extracted 5 evidence table labels with no prescribing context
- F channel included an HTML comment artifact (`<!-- PAGE 34 -->`)
- C channel captured the PP label but missed the PP text

### Pipeline Gap Root Cause
- **C channel**: Extracted "Practice Point 1.2.1" label at 98% confidence but the actual practice point TEXT was missed
- **D channel**: Table decomposition extracted evidence table section headers as if they were prescriptive content
- **F channel**: NuExtract captured 1 genuine sentence + 1 HTML artifact; missed all safety content
- **No channel** extracted the pregnancy warning, adverse effects, or dose titration instructions

### Prior Review Activity
2 of 9 original spans were already CONFIRMED before this audit session, indicating partial prior review coverage on the most meaningful content.

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Pre-audit extraction completeness** | ~18% — 2 genuine sentences from 11 prescriptive facts |
| **Post-audit extraction completeness** | ~99% — All prescriptive content captured (only Figure 3 table deferred) |
| **Pipeline false positive rate** | **78%** (7/9 spans were noise) |
| **Pipeline genuine content** | **2 spans** (pre-existing CONFIRMED) |
| **REVIEWER additions** | 9 facts (all missing safety, prescribing, and scope content) |
| **Overall quality** | **POOR pipeline** — but fully remediated via manual review + cross-check |
| **Final total** | 18 extractions (7 rejected + 2 confirmed + 9 added) |
