# Page 120 Audit — References 86-130: SGLT2i Landmark Trials (25 Spans)

## Page Identity
- **PDF page**: S119 (References — www.kidney-international.org)
- **Content**: References 86-130 — SGLT2i landmark trials (CANVAS, DAPA-HF, EMPA-REG, CREDENCE, DAPA-CKD, VERTIS CV), DPP-4 inhibitors, pioglitazone
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 25 |
| T1 (Patient Safety) | 10 |
| T2 (Clinical Accuracy) | 15 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM), C (Grammar/Regex), E (GLiNER NER) |
| Multi-channel spans | E+F at 90% for SGLT2 ref titles |
| Accept button | DISABLED — "10 Tier 1 spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/25) |
| Disagreements | 8 |
| Review Status | PENDING: 25 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Key Findings

### All 4 Channels Active — First Reference Page
Page 120 is the first reference page with all 4 extraction channels active simultaneously:
- **B**: Drug names (canagliflozin, dapagliflozin, empagliflozin, ertugliflozin, sotagliflozin, saxagliptin, pioglitazone)
- **C**: Lab terms from ref titles (serum creatinine, eGFR patterns)
- **E**: "sodium-glucose co-transporter-2" extracted from SGLT2i ref titles — first E channel appearance in References section
- **F**: Full reference title extraction at 85-90%

### T1 Spans (10) — All False-Positives from SGLT2i Drug Names
- **B bare drugs**: ertugliflozin (×2), canagliflozin, dapagliflozin, empagliflozin, sotagliflozin (×2), saxagliptin (×2), pioglitazone
- All extracted from article titles like "Canagliflozin and renal outcomes in type 2 diabetes..."

### T2 Spans (15) — Mix of Channels
- **F 85-90%**: Reference titles and author lines from SGLT2i landmark trials
- **E+F 90%**: "sodium-glucose co-transporter-2" from ref titles (SGLT2 drug class description)
- **F 90%**: `<!-- PAGE 120 -->` HTML artifact (16th occurrence)
- **C 85%**: Lab terms extracted from ref titles

### SGLT2i Trial References (86-130)
Landmark clinical trial citations present on this page:
- CANVAS (canagliflozin cardiovascular outcomes)
- DAPA-HF (dapagliflozin heart failure)
- EMPA-REG OUTCOME (empagliflozin cardiovascular outcomes)
- CREDENCE (canagliflozin renal outcomes)
- DAPA-CKD (dapagliflozin CKD outcomes)
- VERTIS CV (ertugliflozin cardiovascular outcomes)
- SCORED (sotagliflozin diabetes + CKD)
- SAVOR-TIMI 53 (saxagliptin cardiovascular safety)

## PDF Source Content
- **Refs 86-130**: SGLT2 inhibitor landmark trials; DPP-4 inhibitor safety studies; pioglitazone cardiovascular outcomes; GLP-1 receptor agonist references beginning

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 25 spans are drug names/drug class names/lab terms/titles from numbered references.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 120 of 126
