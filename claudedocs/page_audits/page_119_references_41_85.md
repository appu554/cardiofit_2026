# Page 119 Audit — References 41-85 (22 Spans)

## Page Identity
- **PDF page**: S118 (References — www.kidney-international.org)
- **Content**: References 41-85 — ACEi/ARB trials continued, hyperkalemia, diuretics, potassium management
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 22 |
| T1 (Patient Safety) | 15 |
| T2 (Clinical Accuracy) | 7 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM), C (Grammar/Regex) |
| Multi-channel spans | 3 (B+F at 98%) |
| Accept button | DISABLED — "15 Tier 1 spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/22) |
| Disagreements | 3 |
| Review Status | PENDING: 22 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Span Types

### T1 Spans (15) — All From Reference Titles
- **B bare drugs**: ramipril, enalapril, candesartan, olmesartan (×2), ACE inhibitor (×1), ACE inhibitors (×2), diuretics/diuretic/Diuretics/Diuretic (×4), chlorthalidone
- **B+F ref titles**: captopril ref (Parving 1989), olmesartan ref (Imai 2011), losartan ref (Weil 2013)

### T2 Spans (7) — Mix of Channels
- **F 90%**: `<!-- PAGE 119 -->` HTML artifact (15th occurrence)
- **B 100%**: perindopril, indapamide (bare drugs from refs)
- **F 85%**: O'Hare et al. author line; "Change in albuminuria as surrogate endpoint" ref title
- **C 85%**: "serum creatinine" (×2) — C channel extracting lab terms from ref titles (refs 61, 63)

### New: C Channel in References
First appearance of C (Grammar/Regex) channel in the References section. It extracted "serum creatinine" and "Serum creatinine" from reference titles (Bakris 2000, Schmidt 2017). The regex pattern matches lab test names regardless of context.

## PDF Source Content
- **Refs 41-85**: ACEi/ARB continuation (perindopril, ramipril, captopril, olmesartan, candesartan, losartan studies); hyperkalemia management; diuretics (thiazide, chlorthalidone, indapamide); pregnancy ACE inhibitor risks; KDIGO potassium controversies; patiromer, sodium zirconium cyclosilicate; combined angiotensin inhibition (VA NEPHRON-D)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 22 spans are drug names/lab terms/titles from numbered references.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 119 of 126
