# Page 33 Audit — Recommendation 1.2.1: ACEi/ARB Initiation + Evidence

| Field | Value |
|-------|-------|
| **Page** | 33 (PDF page S32) |
| **Content Type** | Rec 1.2.1 (ACEi/ARB for diabetes+HTN+albuminuria) + IRMA-2/INNOVATION/IDNT/RENAAL trial evidence |
| **Extracted Spans** | 76 original + 8 REVIEWER = 84 total |
| **Channels** | B, C, D, F + REVIEWER |
| **Disagreements** | 0 |
| **Review Status** | REJECTED: 76, ADDED: 8 — 84/84 reviewed |
| **Page Decision** | **FLAGGED** |
| **Risk** | 100% false positive — zero genuine spans from pipeline |
| **Audit Date** | 2026-02-25 (pre-audit) → 2026-02-26 (executed) |
| **Execution** | pharma@vaidshala.com — 76 API rejections + 8 UI additions |

---

## Source PDF Content

**Recommendation 1.2.1 (CRITICAL T1 — NOT EXTRACTED):**
> "We recommend that treatment with an ACEi or ARB be initiated in patients with diabetes, hypertension, and albuminuria, and that these medications be titrated to the highest approved dose that is tolerated (1B)"

**Key Clinical Facts on Page:**
- Albuminuria thresholds: moderately increased (30–300 mg/g), severely increased (>300 mg/g)
- IRMA-2: irbesartan 300 mg/day → 3-fold risk reduction in CKD progression at 2 years
- INNOVATION: telmisartan → lower transition to overt nephropathy after 1 year
- IDNT: irbesartan → 33% decrease in doubling of serum creatinine
- RENAAL: losartan → 16% reduction in doubling of serum creatinine, kidney failure, and death
- Cochrane review: ACEi RR 0.45 (0.29–0.69); ARB RR 0.45 (0.35–0.57) for severely increased albuminuria
- **Adverse effects**: Angioedema 0.30% with ACEi; dry cough; consider switching to ARB
- **Dose titration**: Start low, up-titrate to highest approved dose
- Quality of evidence: Moderate (1B grade)

---

## Execution Log — 2026-02-26

### Phase 1: API Rejections (76/76)

All 76 pipeline spans rejected as noise — **100% false positive rate** (worst page audited).

| Category | Channel | Count | Reason |
|----------|---------|-------|--------|
| Standalone drug names (ACEi ×8, ARB ×8, ARBs ×2, irbesartan ×5, losartan ×3, telmisartan ×4, insulin ×1) | B | 33 | Drug names without prescribing context |
| Evidence table drug names (Losartan, Irbesartan, Telmisartan, "ACEi and ARB") | D | 4 | Table cell fragments |
| "Cochrane systematic" ×15 | D | 15 | Evidence source label repeated from table decomposition |
| Albuminuria percentages (3.5%, 44%, 7.9%) | D | 3 | Study results without drug/population context |
| Evidence labels (All-cause mortality, Moderately increased a ×3, Balance of benefits ×2, Observational, Critical outcomes) | D/F | 9 | Evidence table headers/categories |
| Lab names (potassium, serum creatinine ×4) | C | 5 | Lab names without monitoring instructions |
| Dose values (300 mg ×3, 30 mg ×2, 1 g) | C | 6 | Decontextualized values |
| Section label ("Recommendation 1.2.1") | C | 1 | Label only — recommendation TEXT missing |
| **Total rejected** | | **76** | |

### Phase 2: API Confirmations (0/76)

Zero confirms — no genuine prescriptive content survived from pipeline extraction.

### Phase 3: REVIEWER-Added Facts (8 via UI)

| # | Fact Text (truncated) | Note | Target KB |
|---|----------------------|------|-----------|
| 1 | "We recommend that treatment with an ACEi or ARB be initiated in patients with diabetes, hypertension, and albuminuria, and that these medications be titrated to the highest approved dose that is tolerated (1B)." | Rec 1.2.1 full text — primary ACEi/ARB prescribing recommendation. Verbatim from PDF S32. | KB-1 dosing |
| 2 | "Initiation should begin at a low dose, with up-titration as tolerated to the highest approved dose." | Dose titration instruction — start low, up-titrate. Verbatim from PDF S32. | KB-1 dosing |
| 3 | "Angioedema has been associated with the use of ACEi, with a weighted incidence of 0.30%. Dry cough is also a known adverse effect of ACEi, affecting about 10% of patients. Consideration can be given to switching affected patients to an ARB." | ACEi adverse effects — angioedema (0.30%) + cough (10%) incidence with ARB switching guidance. Verbatim from PDF S32. | KB-4 safety |
| 4 | "Blood pressure, serum potassium, and serum creatinine should be monitored in patients who are started on RAS blockade or whenever there is a change in the dose." | Monitoring instruction for RAS blockade — BP, potassium, creatinine. Verbatim from PDF S32. | KB-16 monitoring |
| 5 | "Practice Point 1.2.1: For patients with diabetes, albuminuria, and normal blood pressure, treatment with an ACEi or ARB may be considered." | Practice Point 1.2.1 — normotensive patients with albuminuria may still benefit from ACEi/ARB. Verbatim from PDF S32. | KB-1 dosing eligibility |
| 6 | "Women who are planning for pregnancy or who are pregnant while on RAS blockade treatment should have the drug discontinued." | Pregnancy contraindication for ACEi/ARB — mandatory discontinuation. Verbatim from PDF S32. | KB-4 safety |
| 7 | "This recommendation does not apply to patients on dialysis. The evidence does not demonstrate superior efficacy of ACEi over ARB treatment or vice versa." | Population exclusion (dialysis) + therapeutic equivalence (ACEi vs ARB). Verbatim from PDF S32. | KB-1 scope + KB-4 safety |
| 8 | "This recommendation applies to patients with type 1 diabetes (T1D) or type 2 diabetes (T2D)." | Rec 1.2.1 population scope — explicitly covers both T1D and T2D (unlike SGLT2i/GLP-1 RA which are T2D-only). Verbatim from PDF S32. | KB-1 dosing scope |

---

## Coverage Checklist

| Content | Covered | Source |
|---------|---------|--------|
| Rec 1.2.1 full text (ACEi/ARB + diabetes + HTN + albuminuria + titrate to max) | ✅ | REVIEWER Fact 1 |
| Dose titration (start low, up-titrate) | ✅ | REVIEWER Fact 2 |
| Angioedema incidence (0.30%) | ✅ | REVIEWER Fact 3 |
| Dry cough (~10% of patients) | ✅ | REVIEWER Fact 3 |
| ARB switching for adverse effects | ✅ | REVIEWER Fact 3 |
| Monitoring (BP, potassium, creatinine) | ✅ | REVIEWER Fact 4 |
| PP 1.2.1 (normotensive + albuminuria) | ✅ | REVIEWER Fact 5 |
| Pregnancy contraindication | ✅ | REVIEWER Fact 6 |
| Dialysis exclusion | ✅ | REVIEWER Fact 7 |
| ACEi vs ARB therapeutic equivalence | ✅ | REVIEWER Fact 7 |
| T1D and T2D population scope | ✅ | REVIEWER Fact 8 |
| IRMA-2/IDNT/RENAAL trial details | ❌ | T2 evidence — deferred to Pipeline 2 L3 extraction |
| Evidence quality grade (1B, Moderate) | ❌ | Metadata — captured within Rec 1.2.1 parenthetical |

---

## Critical Findings (Pre-Audit Assessment — Confirmed)

### 100% False Positive Rate — WORST PAGE AUDITED
- **76/76 pipeline spans** were noise (standalone drug names, evidence table fragments, repeated labels)
- **ZERO genuine prescriptive content** survived from any channel (B, C, D, F)
- B channel matched every occurrence of ACEi/ARB/irbesartan/losartan/telmisartan in evidence discussion
- D channel extracted "Cochrane systematic" 15 times from evidence table cells
- The most important ACEi/ARB recommendation (Rec 1.2.1) was completely missed by all channels

### Pipeline Gap Root Cause
- **B channel**: Drug dictionary matched drug names in evidence narrative text — no sentence-level filtering
- **C channel**: Grammar/regex extracted lab names and dose values without surrounding prescribing context
- **D channel**: Table decomposition shattered evidence summary table into cell-level fragments
- **F channel**: NuExtract captured only 2 sentences (section heading + evidence summary), missed the actual recommendation

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Pre-audit extraction completeness** | ~0% — Recommendation text completely missing |
| **Post-audit extraction completeness** | ~98% — All prescriptive content captured via 8 REVIEWER facts |
| **Pipeline false positive rate** | **100%** (76/76 spans were noise) |
| **Pipeline genuine content** | **0 spans** |
| **REVIEWER additions** | 8 facts (all prescriptive content for this page) |
| **Overall quality** | **WORST PIPELINE PAGE** — but fully remediated via manual review |
| **Final total** | 84 extractions (76 rejected + 8 added) |
