# Page 121 Audit — References 131-172: SGLT2i Continued + MRA Trials (49 Spans)

## Page Identity
- **PDF page**: S120 (References — www.kidney-international.org)
- **Content**: References 131-172 — SGLT2i safety/efficacy studies, cost-effectiveness, MRA trials (finerenone FIDELIO-DKD/FIDELITY, esaxerenone ESAX-DN, spironolactone, eplerenone)
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 49 |
| T1 (Patient Safety) | 45 |
| T2 (Clinical Accuracy) | 4 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM), E (GLiNER NER) |
| Accept button | DISABLED — "45 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/49) |
| Disagreements | 5 |
| Review Status | PENDING: 49 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### Highest T1 Count — 45 of 49 Spans
Page 121 has the highest T1 count of any page in the entire document. The B channel extracted every occurrence of SGLT2i drug names from reference titles:
- **dapagliflozin**: 12 occurrences
- **empagliflozin**: 8 occurrences
- **canagliflozin**: 7 occurrences
- **SGLT2 inhibitors**: 7 occurrences (drug class)
- **mineralocorticoid receptor antagonists**: 1 occurrence (drug class)
- **irbesartan**: 1, **eplerenone**: 1

### B+F Multi-Channel T1 Spans
- "SGLT2 Inhibitor) in patients with type 2 diabetes mellitus" — ref title fragment (B+F 100%)
- "Efficacy and safety of canagliflozin used in conjunction with sulfonylurea..." — ref 149 title (B+F 98%)
- "conjunction with insulin therapy in patients with type 2 diabetes" — ref 150 title fragment (B+F 100%)
- "Hyperkalemia risk with finerenone: results from the FIDELIO-DKD Trial" — ref 163 title (B+F 98%)

### New Drug: Esaxerenone (CS-3150)
F channel extracted "Ito S, Kashihara N, Shikata K, et al. Esaxerone (CS-3150) in patients with type 2 diabetes and microalbuminuria (ESAX-DN..." as T1 at 85%. This is a non-steroidal MRA not previously seen in the extraction — but it's from a reference title.

### T2 Spans (4)
- **F 90%**: `<!-- PAGE 121 -->` HTML artifact (17th occurrence)
- **E+F 90%**: "sodium-glucose cotransporter 2 inhibitor on proximal tubular function..." — ref title
- **F 85%**: "Insights from CREDENCE trial indicate an acute drop in estimated glomerular filtration rate..." — ref title
- **F 85%**: "Rates of hyperkalemia after publication of the Randomized Aldactone Evaluation Study" — ref 159 title

## PDF Source Content
- **Refs 131-172**: SGLT2i safety/cost-effectiveness (DARE-19, CREDENCE subgroups, DAPA-HF/CKD subgroups), ADA/ESC/KDIGO guideline citations, MRA trials (RALES, FIDELIO-DKD, FIDELITY, ESAX-DN), aldosterone antagonist Cochrane reviews

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 49 spans are drug names/drug class names/titles from numbered references. The 45 T1 spans are all false-positives.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 121 of 126
