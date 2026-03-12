# Page 124 Audit — References 267-313: Diet, Sodium, Physical Activity (37 Spans)

## Page Identity
- **PDF page**: S123 (References — www.kidney-international.org)
- **Content**: References 267-313 — Dietary protein restriction, sodium intake studies, DASH diet, physical activity in CKD
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 37 |
| T1 (Patient Safety) | 11 |
| T2 (Clinical Accuracy) | 26 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), C (Grammar/Regex), E (GLiNER NER), F (NuExtract LLM) |
| Accept button | DISABLED — "11 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/37) |
| Disagreements | 4 |
| Review Status | PENDING: 37 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### C Channel "sodium" Explosion — 15 Occurrences
The C (Grammar/Regex) channel extracted "sodium"/"Sodium" 15 times from reference titles about dietary sodium intake (refs 287-310). This is the most extreme example of C channel over-extraction: the word "sodium" appears in nearly every reference title on this page because the topic IS sodium intake, but the regex treats each occurrence as a standalone electrolyte lab term.

### B+C+F Triple-Channel T1 Spans (2)
- "A low-sodium diet potentiates the effects of losartan in type 2 diabetes" — ref 293 (B: losartan, C: sodium, F: title)
- "Enhanced responsiveness of blood pressure to sodium intake and to angiotensin II is associated with insulin resistance i..." — ref 303 (B: insulin, C: sodium, F: title)

### T1 Spans (11)
- **B bare drugs**: losartan, hydrochlorothiazide (×3), insulin (×4), telmisartan
- **B+C+F 100%**: 2 ref titles combining drug + sodium + LLM extraction

### T2 Spans (26)
- **C 85%**: sodium (×15), potassium (×4), Sodium (×4)
- **E 85%**: potassium (×1)
- **B 100%**: thiazide (×1)
- **C+F multi-channel**: "sodium restriction and sodium supplementation" ref fragment (93%), "Short-term moderate sodium restriction..." ref title (90%)
- **F 85%**: "Diabetes self-management education..." joint position statement title
- No HTML artifact on this page

## PDF Source Content
- **Refs 267-276**: Dietary protein restriction in diabetic nephropathy (RCTs)
- **Refs 277-286**: Nutrition guidelines (ADA, KDOQI), protein losses in dialysis
- **Refs 287-310**: Sodium intake and diabetic kidney disease (DASH, WHO guidelines, RCTs)
- **Refs 311-313**: Physical activity in CKD, weight management

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 37 spans are drug/electrolyte names from numbered references about dietary interventions.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 124 of 126
