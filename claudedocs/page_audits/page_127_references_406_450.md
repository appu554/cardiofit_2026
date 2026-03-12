# Page 127 Audit — References 406-450: Self-Management, Multifactorial Care, Epidemiology (9 Spans)

## Page Identity
- **PDF page**: S126 (References — www.kidney-international.org)
- **Content**: References 406-450 — Diabetes self-management education, nurse-coordinated care, multifactorial interventions, structured care programs (JADE, RAMP-DM), hypoglycemia risks, diabetes epidemiology (IDF Atlas), cost-effectiveness
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO (first reference page without disagreement)

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 9 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 8 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM), C (Grammar/Regex) |
| Accept button | DISABLED — "1 Tier 1 (patient safety) span must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/9) |
| Disagreements | 0 |
| Review Status | PENDING: 9 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### No B Channel — First Reference Page Without Drug Dictionary
Page 127 is the first reference page (since page 117 acknowledgments) without the B (Drug Dictionary) channel active. The references on this page cover self-management education, structured care, and epidemiology — topics that rarely mention specific drug names in their titles. This demonstrates how the B channel's contribution to T1 false-positives is directly correlated with drug-name density in reference titles.

### Single T1 Span — F Channel IDF Atlas Reference
- **F 90%**: "432. International Diabetes Federation. IDF Diabetes Altas. Accessed August 14, 2020. https://diabetesatlas.org/en/resou..." — The F channel extracted this IDF Diabetes Atlas reference at 90% and it was classified as T1. This is likely because the classifier detected a clinical guideline/resource URL, but it's simply a bibliographic citation.

### T2 Spans (8) — F and C Channel Mix
- **F 85%**: "A multifactorial intervention to improve blood pressure control in co-existing diabetes and kidney disease: a feasibilit..." — ref 418 title
- **F 90%**: "428. NHS Digital. National Diabetes Audit—Report 1 Care Processes and Treatment Targets. 2017-18." — registry report ref
- **F 85%**: "Severe hypoglycemia identifies vulnerable patients with type 2 diabetes at risk for premature death and all-site cancer:..." — ref 433 title
- **F 85%**: "Severe hypoglycemia and risks of vascular events and death." — ref 435 title (Zoungas NEJM 2010)
- **F 90%**: "441. Funnell MM, Piatt GA. Diabetes quality improvement: beyond glucose control. Lancet. 2012;379:2218–2219." — editorial ref
- **C 85%**: "hemoglobin" — lab term extracted from ref 443 title ("hemoglobin A1c outcomes")
- **C 85%**: "A1C" — lab term extracted from ref 443 title
- **C 85%**: "fasting glucose" — lab term extracted from ref 445 title ("fasting glucose, and risk of cause-specific death")

### No HTML Artifact
No `<!-- PAGE 127 -->` HTML artifact on this page — the F channel did not produce one here.

## PDF Source Content
- **Refs 406-412**: Diabetes-renal multifactorial interventions, self-management education, Cochrane reviews
- **Refs 413-422**: Study quality tools (AMSTAR 2), nurse-coordinated CKD care, structured care RCTs (SURE, TASMIN-SR), self-management in CKD
- **Refs 423-432**: Cost-effectiveness of diabetes education, UK structured education policy, IDF guidelines and atlas
- **Refs 433-441**: Hypoglycemia risks (Hong Kong registry, ADVANCE), multifactorial intervention meta-analyses, exercise in diabetic CKD
- **Refs 442-450**: Interdisciplinary team management, peer support, community health workers, diabetes mortality trends, Steno-2 cost-effectiveness, LMIC guideline gaps

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 9 spans are lab terms/ref titles from numbered references about self-management and epidemiology.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 127 of 126 (penultimate page)
