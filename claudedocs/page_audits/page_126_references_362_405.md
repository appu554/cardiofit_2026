# Page 126 Audit — References 362-405: Metformin + GLP-1 RA + Behavioral (53 Spans)

## Page Identity
- **PDF page**: S125 (References — www.kidney-international.org)
- **Content**: References 362-405 — Metformin formulations/transplant/B12, GLP-1 RA landmark trials (REWIND, HARMONY, SUSTAIN-6, LEADER, EXSCEL, ELIXA, AWARD-7, PIONEER 5), GLP-1 RA renal outcomes, liraglutide weight management, diabetes self-management education
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 53 |
| T1 (Patient Safety) | 50 |
| T2 (Clinical Accuracy) | 3 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), C (Grammar/Regex), F (NuExtract LLM) |
| Accept button | DISABLED — "50 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/53) |
| Disagreements | 3 |
| Review Status | PENDING: 53 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### Second-Highest T1 Count — 50 of 53 Spans
Page 126 has the second-highest T1 count of any page (after page 121's 45). The B channel extracted every drug name occurrence from reference titles about glucose-lowering agents:
- **metformin**: 18 occurrences (dominant — refs 362-371 are metformin-specific)
- **liraglutide**: 8 occurrences (LEADER, LIRA-RENAL, weight management)
- **insulin**: 7 occurrences (comparator drug in GLP-1 RA trials)
- **dulaglutide**: 4 occurrences (REWIND, AWARD-7)
- **semaglutide**: 4 occurrences (SUSTAIN-6, oral semaglutide PIONEER 5)
- **exenatide**: 2 occurrences (EXSCEL)
- **GLP-1 receptor agonist**: 1 (drug class name)
- **dapagliflozin**: 1 (comparator in cost-effectiveness study)

### Three B+C+F Triple-Channel T1 Spans (Most on Any Page)
1. "Efficacy, tolerability, and safety of a novel once-daily extended-release metformin in patients with type 2 diabetes." — ref 365 title (B: metformin, C: unknown pattern, F: title)
2. "Microvascular and cardiovascular outcomes according to renal function in patients treated with once-weekly exenatide: in..." — ref 376 EXSCEL title (B: exenatide, C: unknown pattern, F: title)
3. "Effects of once-weekly exenatide on cardiovascular outcomes in type 2 diabetes." — ref 378 EXSCEL title (B: exenatide, C: unknown pattern, F: title)

All at 100% confidence — the highest-confidence false-positives in the entire document.

### T2 Spans (3) — Minimal
- **F 85%**: "results of a double-blind, placebo-controlled, dose-response trial" — ref title fragment
- **C 85%**: "daily" — dosing frequency term extracted from ref title text
- **C 85%**: "3.0 mg" — dose value extracted from liraglutide weight management ref (ref 402: "Liraglutide 3.0 mg for weight management")

### Notable: No HTML Artifact
No `<!-- PAGE 126 -->` HTML artifact — the F channel did not produce one on this page. This is the third consecutive reference page (123, 124, 126) without the HTML artifact.

### Metformin Reference Cluster (Refs 362-371)
The densest metformin reference cluster in the document:
- Refs 362-366: Metformin formulations (immediate-release vs extended-release, CONSENT trial)
- Refs 367-369: Metformin in kidney transplant (observational, Transdiab pilot RCT)
- Refs 370-371: Metformin and vitamin B12 deficiency (NHANES analysis, RCT)

## PDF Source Content
- **Refs 362-371**: Metformin formulations, adherence, kidney transplant use, vitamin B12 deficiency
- **Refs 372-386**: GLP-1 RA landmark trials (REWIND, Harmony, SUSTAIN-6, LEADER, EXSCEL, ELIXA, AWARD-7, PIONEER 5, REMODEL)
- **Refs 387-396**: GLP-1 RA safety/efficacy in renal impairment (lixisenatide, liraglutide in dialysis, exenatide in DKD)
- **Refs 397-402**: GLP-1 RA cost-effectiveness, obesity and kidney disease, liraglutide weight management
- **Refs 403-405**: Diabetes self-management education (structured education, behavioral programs)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 53 spans are drug names/dosing terms/ref titles from numbered references. The 50 T1 spans are all false-positives from B channel drug name extraction.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 126 of 126 (last reference page before supplementary)
