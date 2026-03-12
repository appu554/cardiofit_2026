# Page 123 Audit — References 220-266: Glycemic Control + Diet (50 Spans)

## Page Identity
- **PDF page**: S122 (References — www.kidney-international.org)
- **Content**: References 220-266 — CGM/glycemic markers in CKD, intensive glucose control trials (DCCT, UKPDS, ADVANCE, ACCORD, Steno), dietary protein restriction
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 50 |
| T1 (Patient Safety) | 16 |
| T2 (Clinical Accuracy) | 34 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM), C (Grammar/Regex), E (GLiNER NER) |
| Accept button | DISABLED — "16 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/50) |
| Disagreements | 3 |
| Review Status | PENDING: 50 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### C and E Channel Explosion — Lab Term Extraction
The C (Grammar/Regex) and E (GLiNER NER) channels dominate T2 on this page, extracting individual lab terms from reference titles:
- **"sodium"/"Sodium"**: 9 occurrences (C ×2, E ×7) — from "sodium-glucose cotransporter" in ref titles
- **"HbA1c"**: 4 occurrences (C 85%)
- **"hemoglobin"/"Hemoglobin"**: 6 occurrences (C 85%)
- **"A1C"**: 7 occurrences (C 85%)

This is a pathological case of the regex channel: it matches the word "sodium" in "sodium-glucose cotransporter" as if it were a standalone electrolyte lab value. Similarly, "hemoglobin" from "hemoglobin A1c" is matched as a standalone CBC component.

### Notable T1 Span: C 95% "eGFR <30 mL/min/1.73m²"
The C channel extracted "eGFR <30 mL/min/1.73m²" at 95% confidence as T1. This pattern matches a genuine clinical threshold format — but it's from a reference title, not a clinical recommendation. This span demonstrates how the C channel's regex for renal function thresholds fires regardless of document context.

### Triple-Channel Span: B+C+F 100%
"Continuous glucose monitoring vs conventional therapy for glycemic control in adults with type 1 diabetes treated with m..." — a reference title that triggered all three of B (drug: insulin from title), C (lab term: glucose), and F (title extraction). This is the first B+C+F triple-channel span in the References section.

### T1 Spans (16)
- **B bare drugs**: insulin (×9), metformin (×3)
- **C 95%**: "eGFR <30 mL/min/1.73m²" (threshold pattern from ref title)
- **B+F 98%**: UKPDS ref titles mentioning sulphonylureas/insulin, DIAMOND CGM trial
- **B+C+F 100%**: GOLD CGM trial (triple-channel)

### No HTML Artifact
Unusually, no `<!-- PAGE 123 -->` HTML artifact on this page — the F channel did not produce one here.

## PDF Source Content
- **Refs 220-230**: CGM in CKD/dialysis, glycemic markers (glycated albumin, fructosamine, 1,5-AG), time in range
- **Refs 231-256**: Intensive glucose control landmark trials (DCCT, UKPDS 33/34, ADVANCE, ACCORD, VADT, Steno, Stockholm)
- **Refs 257-260**: CGM trials (DIAMOND, GOLD, closed-loop)
- **Refs 261-266**: Dietary patterns and protein restriction in CKD

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 50 spans are drug names/lab terms/ref titles from numbered references.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 123 of 126
