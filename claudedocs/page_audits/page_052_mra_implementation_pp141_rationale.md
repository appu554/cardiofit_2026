# Page 52 Audit — MRA Implementation, Hyperkalemia Management, PP 1.4.1, Rationale

| Field | Value |
|-------|-------|
| **Page** | 52 (PDF page S51) |
| **Content Type** | MRA values/preferences (continued), resource use/costs, considerations for implementation (hyperkalemia management, potassium thresholds, SGLT2i combination, steroidal+nonsteroidal MRA contraindication, pregnancy), rationale for MRA section, Practice Point 1.4.1 |
| **Extracted Spans** | 12 total (7 T1, 5 T2) |
| **Channels** | B, C, F |
| **Disagreements** | 3 |
| **Review Status** | PENDING: 12 |
| **Risk** | Disagreement |
| **Audit Date** | 2026-02-25 (revised) |
| **Cross-Check** | Verified against raw spans — counts confirmed (12), channels confirmed (B/C/F), disagreements added (3) |

---

## Source PDF Content

**Values and Preferences (Continued from Page 51):**
- Lack of definitive data on benefits when nonsteroidal MRA added to SGLT2i
- Limited representation of moderate albuminuria patients in FIDELIO/FIGARO
- Only finerenone has rigorous outcome data — restriction to 1 drug in class
- Patients with severely increased albuminuria (ACR ≥300 mg/g) might be particularly inclined to choose nonsteroidal MRA
- Some patients with history of severe hyperkalemia may choose to avoid added risk

**Resource Use and Costs:**
- Nonsteroidal MRA not yet available in many countries
- Cost has yet to be determined — likely priced significantly higher than generic medications
- Costs may be prohibitive; lower priority in low-resource settings
- Monitoring of potassium during treatment already indicated for CKD + ACEi/ARB patients
- Increased hyperkalemia rate may lead to higher healthcare costs (more frequent visits)

**Considerations for Implementation (CRITICAL PRESCRIBING CONTENT):**
- **Tested population**: CKD + T2D with residual cardiorenal risk, albuminuria ≥30 mg/g despite max tolerated RAS blockade
- **Only finerenone** has demonstrated clinical CV and kidney benefits
- **Hyperkalemia management**: "Treatment dose and monitoring should be in accordance with the clinical trials, as described in Practice Point 1.4.3"
- **Potassium threshold**: "Treatment should not be initiated if serum potassium is elevated (4.8 mmol/l was the threshold at screening in the finerenone trials, but per FDA label, serum potassium should not be >5 mmol/l)"
- **Hyperkalemia management**: "Most incidents can be managed with treatment pauses of 72 hours" (short half-life)
- **BP effect**: Small reduction in systolic BP (3 mmHg) with finerenone vs placebo
- **No HbA1c effect**: No increase in hypo-/hyperglycemia, no sexual side effects
- **SGLT2i combination**: "Beneficial effects similar among participants also treated with SGLT2i or GLP-1 RA... potentially lower risk of hyperkalemia when finerenone combined with SGLT2i"
- **Combination not yet tested**: "Randomized studies have not explicitly tested whether benefits are additive"
- **CRITICAL**: "Steroidal and nonsteroidal MRA should not be combined due to risk of hyperkalemia"
- **Pregnancy**: "Steroidal MRA are currently contraindicated in pregnancy. For nonsteroidal MRA, no experience with pregnancy — discontinue if pregnant/planning pregnancy"

**Rationale:**
- Adding MRA to ACEi/ARB reduces albuminuria
- Steroidal MRA (spironolactone, eplerenone): reduce albuminuria but NO clinical outcome data
- Nonsteroidal MRA (finerenone, esaxerenone): also reduce albuminuria; finerenone reduced kidney + CV outcomes in 2 pivotal trials

**Practice Point 1.4.1:**
- "Nonsteroidal MRA are most appropriate for patients with T2D who are at high risk of CKD progression and cardiovascular events, as demonstrated by persistent albuminuria despite other standard-of-care therapies"

---

## Key Spans Assessment

### Tier 1 Spans (7)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Monitoring of potassium during treatment is already indicated for patients with CKD treated with an ACEi or ARB" | B+C+F | 100% | **✅ T1 CORRECT** — Potassium monitoring instruction for MRA + RASi patients |
| "Nonsteroidal MRA can cause hyperkalemia, and treatment dose and monitoring should be in accordance with the clinical tri..." | B+C | 100% | **✅ T1 CORRECT** — Drug safety warning + dosing guidance reference |
| "Steroidal and nonsteroidal MRA should not be combined due to risk of hyperkalemia." | B+C | 100% | **✅ T1 CORRECT** — CRITICAL CONTRAINDICATION — drug combination prohibition |
| "GLP-1 RA" | B | 100% | **→ T3** — Drug class name only |
| "spironolactone" | B | 100% | **→ T3** — Drug name only |
| "eplerenone" | B | 100% | **→ T3** — Drug name only |
| "Practice Point 1.4.1" | C | 98% | **→ T3** — PP label only (but this time the PP text IS captured in the PDF panel) |

**Summary: 3/7 T1 spans are genuine clinical sentences (43%). The 3 genuine spans are EXCELLENT — complete prescribing instructions with drug names + clinical context. The remaining 4 are drug names or labels.**

### Tier 2 Spans (5)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| `<!-- PAGE 52 -->` | F | 90% | **⚠️ PIPELINE ARTIFACT** — Reject |
| "efforts will be made to optimize the use of less expensive drugs" | F | 85% | **✅ T2 OK** — Cost/access consideration |
| "an increased rate of hyperkalemia may lead to higher healthcare costs due to more frequent patient visits" | F | 85% | **✅ T2 OK** — Healthcare cost impact of MRA |
| "Consequently, the cost of these drugs has yet to be determined" | F | 85% | **✅ T2 OK** — Cost availability statement |
| "contraindicated" | C | 95% | **⚠️ SHOULD BE T1** — From "Steroidal MRA are currently contraindicated in pregnancy" — pregnancy contraindication text should be T1 with full context |

**Summary: 3/5 T2 correctly tiered (cost-related F channel extractions). 1 pipeline artifact. 1 should be upgraded to T1 with full pregnancy context.**

---

## Critical Findings

### ✅ THREE GENUINE T1 PRESCRIBING SENTENCES — BEST MULTI-CHANNEL PERFORMANCE

For only the second time in the entire audit (after page 50), the pipeline produces multiple genuine T1 clinical sentences:

1. **Potassium monitoring** (B+C+F triple-channel): Complete sentence about monitoring already indicated for CKD + ACEi/ARB
2. **Hyperkalemia dose warning** (B+C): MRA causes hyperkalemia, dose per clinical trials
3. **Combination prohibition** (B+C): "Steroidal and nonsteroidal MRA should not be combined" — a critical drug interaction contraindication

These succeed because the sentences contain drug names (B channel match), clinical keywords like "contraindicated"/"hyperkalemia" (C channel match), and are extractable LLM sentences (F channel match).

### ✅ F Channel Produces 3 Genuine T2 Cost Sentences
The F channel (NuExtract LLM) successfully extracts 3 cost-related sentences from the resource use section. These are properly tiered as T2.

### ⚠️ PP 1.4.1 Text NOT CAPTURED (Same Pattern)
The Practice Point 1.4.1 text — "Nonsteroidal MRA are most appropriate for patients with T2D who are at high risk of CKD progression and cardiovascular events, as demonstrated by persistent albuminuria despite other standard-of-care therapies" — is visible in the PDF panel but only the label "Practice Point 1.4.1" is extracted. Same systemic PP label-vs-text failure.

### ⚠️ "Contraindicated" T2 Should Be T1 with Context
The C channel extracted "contraindicated" which comes from "Steroidal MRA are currently contraindicated in pregnancy" — a T1 pregnancy safety warning. The word alone is insufficient; the full sentence with drug class + population is needed.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| "Treatment should not be initiated if serum potassium >5 mmol/l" (FDA label) | **T1** | Potassium initiation threshold |
| "4.8 mmol/l was the threshold at screening in finerenone trials" | **T1** | Trial-based potassium cutoff |
| "Most incidents of hyperkalemia managed with treatment pauses of 72 hours" | **T1** | Hyperkalemia management protocol |
| "Steroidal MRA contraindicated in pregnancy; discontinue nonsteroidal MRA if pregnant" | **T1** | Pregnancy safety |
| PP 1.4.1 full text: "Nonsteroidal MRA most appropriate for T2D + high CKD progression risk + persistent albuminuria" | **T1** | Practice point recommendation |
| "Beneficial effects similar with concurrent SGLT2i or GLP-1 RA; potentially lower hyperkalemia risk with SGLT2i" | **T1** | Co-prescribing safety |
| "Randomized studies have not explicitly tested whether benefits are additive" | **T2** | Evidence limitation |
| "Only finerenone has demonstrated clinical CV and kidney benefits" | **T1** | Drug-specific limitation |
| "Small reduction in systolic BP (3 mmHg) with finerenone vs placebo" | **T2** | Expected BP effect |

### ⚠️ Pipeline Artifact Present
`<!-- PAGE 52 -->` F channel artifact continues.

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — 3 genuine T1 prescribing sentences (potassium monitoring, hyperkalemia warning, combination prohibition) are high-quality; F channel cost T2s are appropriate |
| **Tier corrections** | 3 drug names: T1 → T3; PP 1.4.1 label: T1 → T3; "contraindicated": T2 → T1 (with context); pipeline artifact: REJECT |
| **Missing T1** | Potassium >5 mmol/l threshold, 72-hour treatment pause protocol, pregnancy contraindication, PP 1.4.1 text, SGLT2i combination safety, "only finerenone" limitation |
| **Missing T2** | Additive benefits not tested, 3 mmHg BP reduction |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~35% — 3 genuine T1 prescribing sentences + 3 T2 cost sentences captured from dense implementation page |
| **Tier accuracy** | ~50% (3/7 T1 correct + 3/5 T2 correct = 6/12) |
| **False positive T1 rate** | 57% (4/7 T1 are drug names or labels) |
| **Genuine T1 content** | 3 extracted (potassium monitoring, hyperkalemia dosing, combination prohibition) |
| **Overall quality** | **GOOD** — Second-best page in audit (after page 50); multi-channel B+C combination produces genuine contraindication and safety sentences |

---

## Why Page 52 Works (B+C Multi-Channel Pattern)

Page 52 demonstrates the **B+C dual-channel pattern** as the most reliable extraction mechanism:
1. **B channel** matches drug class name ("MRA", "ACEi", "ARB") within a sentence
2. **C channel** matches clinical keyword ("hyperkalemia", "contraindicated", "monitoring")
3. When both fire on the same sentence, the result is a genuine prescribing instruction

This is similar to page 50's B+F pattern but uses C (Grammar/Regex) instead of F (NuExtract LLM) as the co-matching channel. Both patterns succeed because they require two independent signals to confirm clinical relevance.

**Key difference from noise pages**: On pages 47-49, B and C fire independently — B on every drug mention, C on every lab term — producing isolated noise. On page 52, B and C fire on the *same span*, producing genuine multi-word clinical content.

---

## Review Actions Completed — 2026-02-27

**Reviewer**: claude-auditor

### Notable Discovery During Review

The span `dc74f8a8` was far more comprehensive than the audit anticipated. It captured the full hyperkalemia warning sentence AND the potassium initiation thresholds in a single span: "Nonsteroidal MRA can cause hyperkalemia, and treatment dose and monitoring should be in accordance with the clinical trials, as described in Practice Point 1.4.3. Treatment should not be initiated if serum potassium is elevated (4.8 mmol/L was the threshold at screening in the finerenone trials, but per FDA label, serum potassium should not be >5 mmol/L)." This means two of the audit's "missing" items (potassium >5 mmol/L threshold and 4.8 mmol/L trial threshold) were already captured — an excellent pipeline result.

### CONFIRMED Spans (6)

| Span ID | Text | Tier | Reason |
|---------|------|------|--------|
| dbe0eae8 | "Monitoring of potassium during treatment is already indicated for patients with CKD treated with an ACEi or ARB" | T1 | Genuine prescribing instruction — potassium monitoring for CKD + ACEi/ARB patients (KB-4, KB-16) |
| dc74f8a8 | "Nonsteroidal MRA can cause hyperkalemia, and treatment dose and monitoring should be in accordance with the clinical trials...Treatment should not be initiated if serum potassium is elevated (4.8 mmol/L...should not be >5 mmol/L)." | T1 | Excellent combined span — hyperkalemia warning + dose guidance + potassium initiation thresholds (KB-1, KB-4, KB-5, KB-16) |
| a94bc6eb | "Steroidal and nonsteroidal MRA should not be combined due to risk of hyperkalemia." | T1 | Critical contraindication — steroidal + nonsteroidal MRA combination prohibition (KB-5) |
| f1f182aa | "efforts will be made to optimize the use of less expensive drugs" | T2 | Cost/access consideration — useful implementation context |
| 99c03840 | "an increased rate of hyperkalemia may lead to higher healthcare costs due to more frequent patient visits" | T2 | Healthcare cost impact of MRA hyperkalemia |
| 7198a73c | "Consequently, the cost of these drugs has yet to be determined" | T2 | Cost availability statement |

### REJECTED Spans (6)

| Span ID | Text | Tier | Reason | Reject Code |
|---------|------|------|--------|-------------|
| e7aaaa08 | `<!-- PAGE 52 -->` | T2 | Pipeline artifact — HTML comment, not clinical content | other |
| 3afc8fa1 | "GLP-1 RA" | T1 | Drug class name only — no clinical sentence | out_of_scope |
| 93c256c1 | "spironolactone" | T1 | Drug name only — no clinical sentence | out_of_scope |
| 51591fb2 | "eplerenone" | T1 | Drug name only — no clinical sentence | out_of_scope |
| cfaff928 | "Practice Point 1.4.1" | T1 | PP label only — practice point text not captured | out_of_scope |
| ac0a74c8 | "contraindicated" | T2 | Single word without context — from pregnancy contraindication sentence but word alone is insufficient | out_of_scope |

### ADDED Facts (7)

| # | Added Text | Target KBs | Note |
|---|-----------|------------|------|
| 1 | "Most incidents of hyperkalemia can be managed with treatment pauses of 72 hours due to the short half-life of finerenone" | KB-4, KB-16 | Hyperkalemia management protocol — 72-hour treatment pause |
| 2 | "Steroidal MRA are currently contraindicated in pregnancy. For nonsteroidal MRA, there is no experience with use in pregnancy, and nonsteroidal MRA should be discontinued if the patient becomes pregnant or is planning pregnancy" | KB-4 | Pregnancy safety — steroidal contraindicated, nonsteroidal discontinue |
| 3 | "Practice Point 1.4.1: Nonsteroidal MRA are most appropriate for patients with T2D who are at high risk of CKD progression and cardiovascular events, as demonstrated by persistent albuminuria despite other standard-of-care therapies" | KB-1 | Practice Point 1.4.1 full text — target population for nonsteroidal MRA |
| 4 | "Beneficial effects of finerenone were similar among participants also treated with SGLT2i or GLP-1 RA, with potentially lower risk of hyperkalemia when finerenone is combined with SGLT2i" | KB-5 | Co-prescribing safety — SGLT2i may reduce finerenone hyperkalemia risk |
| 5 | "Randomized studies have not explicitly tested whether the benefits of finerenone and SGLT2i are additive when used in combination" | KB-1, KB-5 | Evidence limitation — additive benefits not tested |
| 6 | "Only finerenone has demonstrated clinical cardiovascular and kidney benefits among nonsteroidal MRA" | KB-1 | Drug-specific limitation — only finerenone has outcome data |
| 7 | "Finerenone was associated with a small reduction in systolic blood pressure (3 mmHg) compared with placebo" | KB-1, KB-16 | Expected BP effect of finerenone |

---

## Raw PDF Gap Analysis (2026-02-27)

### Gap-Fill Facts Added (4)
| # | Text | Priority | KB Target | Note |
|---|------|----------|-----------|------|
| 8 | "no effect on HbA1c, no increase in hypo- or hyperglycemia, and no sexual side effects due to the specificity for the MRA" | HIGH | KB-4 | Finerenone safety reassurances — differentiates from steroidal MRA |
| 9 | "Patients with severely increased albuminuria (ACR ≥300 mg/g)...might be particularly inclined to choose a nonsteroidal MRA" | MODERATE | KB-1 | ACR ≥300 as preferred MRA target population |
| 10 | "Monitoring of potassium during treatment is already indicated for patients with CKD treated with an ACEi or ARB..." | MODERATE | KB-16 | K+ monitoring already required for RASi — MRA adds incremental burden |
| 11 | "Some patients who...have a history of severe hyperkalemia or highly variable serum potassium may choose to avoid the added risk" | MODERATE | KB-4 | History of severe hyperkalemia as relative exclusion |

### Not Added (Low Priority)
| Content | Reason |
|---------|--------|
| Factors influencing patient choice (lack of real-world data, limited moderate albuminuria representation) | Patient decision context — not prescribing rules |
| SGLT2i became first-line for T2D with CKD | Already established in earlier pages (46-49) |
| Direct head-to-head comparisons not available | Covered by "additive benefits not tested" fact |
| Cost-effectiveness evaluations not yet available | Covered by confirmed T2 cost spans |

---

## Post-Review State (Final)

| Metric | Value |
|--------|-------|
| **Original spans** | 12 |
| **Confirmed** | 6 (3 T1 prescribing sentences, 3 T2 cost sentences) |
| **Rejected** | 6 (3 drug names, 1 PP label, 1 pipeline artifact, 1 decontextualized word) |
| **Added** | 11 (7 from initial review + 4 from raw PDF gap analysis) |
| **Total spans** | 23 (12 original + 11 added) |
| **Total reviewed** | 23/23 (100%) |
| **Pipeline 2 ready** | 17 (6 confirmed + 11 added) |
| **Completeness (post-review)** | ~95% — Finerenone safety profile (no HbA1c/glycemia/sexual effects), ACR ≥300 targeting, K+ monitoring context, severe hyperkalemia exclusion now captured |
| **Review date** | 2026-02-27 |
| **Reviewer** | claude-auditor |
