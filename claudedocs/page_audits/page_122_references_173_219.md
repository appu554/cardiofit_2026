# Page 122 Audit — References 173-219: MRA Continued + Glycemic Monitoring (22 Spans)

## Page Identity
- **PDF page**: S121 (References — www.kidney-international.org)
- **Content**: References 173-219 — MRA trials (spironolactone, finerenone, apararenone), smoking/CKD, glycemic control (HbA1c, CGM, glycated albumin, fructosamine), dialysis glucose monitoring
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 22 |
| T1 (Patient Safety) | 11 |
| T2 (Clinical Accuracy) | 11 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM), C (Grammar/Regex) |
| Accept button | DISABLED — "11 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/22) |
| Disagreements | 5 |
| Review Status | PENDING: 22 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## T1 Spans (11) — All False-Positives
- **B bare drugs**: spironolactone (×4), finerenone (×4)
- **B+F ref titles**: "Effect of finerenone on albuminuria..." (ref 167, 98%), "Beneficial impact of spironolactone on nephrotic range albuminuria..." (ref 173, 98%), "spironolactone in type 2 diabetic nephropathy..." (ref 174, 98%)

## T2 Spans (11) — All Reference Content
- **F 90%**: `<!-- PAGE 122 -->` HTML artifact (18th occurrence)
- **F 85%**: Author lines (Wada, Pan), ref titles (smoking/CKD, e-cigarettes, diabetes treatment, hypoglycemia)
- **C+F 93%**: "Hemoglobin A1C (5 Challenge) GHS-C 2019" — CAP reference (C channel matching lab term)
- **C+F 90%**: "Use of continuous glucose monitoring in patients with diabetes mellitus on peritoneal dialysis..." — C channel matching CGM term

## Thematic Transition
This page bridges two guideline chapters in the references:
1. **Refs 173-178**: MRA trials conclusion (spironolactone/finerenone albuminuria studies)
2. **Refs 179-188**: Smoking and CKD (Ch3 Lifestyle references)
3. **Refs 189-219**: Glycemic monitoring in CKD (HbA1c accuracy in CKD, glycated albumin, CGM on dialysis)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 22 spans are drug names/lab terms/titles from numbered references.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 122 of 126
