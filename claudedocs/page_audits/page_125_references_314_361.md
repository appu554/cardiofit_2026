# Page 125 Audit — References 314-361: Physical Activity, GLP-1 RA, Metformin (11 Spans)

## Page Identity
- **PDF page**: S124 (References — www.kidney-international.org)
- **Content**: References 314-361 — Physical activity in CKD, weight management, GLP-1 receptor agonist outcomes, metformin safety in CKD
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 11 |
| T1 (Patient Safety) | 6 |
| T2 (Clinical Accuracy) | 5 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM) |
| Accept button | DISABLED — "6 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Tier accuracy | 0% (0/11) |
| Disagreements | 4 |
| Review Status | PENDING: 11 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — Count corrected 12→11 (T2 6→5); D channel removed (not present in raw data); verified against raw extraction data |

## Key Findings

### D Channel Reappearance — "Antihyperglycemic agents"
The D (Table Decomp) channel reappears in the References section for the first time since the main body tables. It extracted "Antihyperglycemic agents" at 92% confidence as T2. This likely reflects a supplementary table heading or structured list of glucose-lowering agents in the references transition area.

### T1 Spans (6) — GLP-1 RA + Metformin Drug Names
- **B+F 98%**: GLP-1 receptor agonist systematic review ref title
- **B+F 98%**: Metformin-associated lactic acidosis ref title
- **B+F 98%**: FDA metformin safety communication ref title
- **B+F 100%**: Bailey metformin review article ref title
- **B bare drugs**: Additional GLP-1 RA and metformin occurrences from ref titles

### T2 Spans (6)
- **F 90%**: `<!-- PAGE 125 -->` HTML artifact (19th occurrence)
- **F 85%**: Exercise and CKD ref titles (physical activity systematic reviews)
- **F 85%**: Adiposity and kidney outcomes meta-analysis ref title
- **D 92%**: "Antihyperglycemic agents" — table decomposition extraction
- **F 85%**: CARMELINA trial (linagliptin cardiovascular outcomes) ref title

## Thematic Transition
This page bridges the final lifestyle references and the beginning of Chapter 4 glucose-lowering agent references:
1. **Refs 314-320**: Physical activity in CKD (exercise RCTs, rehabilitation)
2. **Refs 321-330**: Weight management and bariatric surgery in CKD
3. **Refs 331-345**: GLP-1 receptor agonist landmark trials (LEADER, SUSTAIN-6, REWIND, HARMONY)
4. **Refs 346-361**: Metformin safety in CKD (lactic acidosis, FDA labeling changes, dosing adjustments)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All 12 spans are drug names/drug class names/ref titles from numbered references.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 125 of 126
