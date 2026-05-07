# Wave 4A Task 1 — PHARMA-Care + Standard 5 Quality-Gap Rule Queue

This manifest documents the PHARMA-Care 5-domain quality indicator and
Aged Care Quality Standard 5 (Clinical Care, in force 2026) evidence
rules **queued** for authoring, **not** yet shipped. Wave 4A Task 1
ships 4 grounded rules as a vertical slice; the remaining 21 are
listed here for clinical-author input.

## Rules Shipped (Wave 4A Task 1 vertical slice)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D2-ANTIPSYCHOTIC-PREVALENCE | PHARMA-Care v1 D2 (UniSA Sluggett-led, published) |
| `VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D1-POLYPHARMACY-10PLUS | PHARMA-Care v1 D1 |
| `VAIDSHALA_PC_D3_ACB_ABOVE_3` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D3-ACB-ABOVE-3 | PHARMA-Care v1 D3 |
| `VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D4-BPSD-FIRST-LINE-NONPHARM | PHARMA-Care v1 D4 |

All criterion_ids carry a `TODO(clinical-author)` marker pending
confirmation against the published PHARMA-Care v1 framework PDF.

## Rules Shipped (Wave-extension batch 2026-05 — 4 rules)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `VAIDSHALA_PC_D5_PAIN_ASSESSMENT_DOCUMENTATION_GAP` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D5-PAIN-ASSESSMENT-GAP | PHARMA-Care v1 D5 (TODO clinical-author) |
| `VAIDSHALA_PC_D1_POLYPHARMACY_5PLUS_HIGH_RISK` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D1-POLYPHARMACY-5PLUS-HIGH-RISK | PHARMA-Care v1 D1 (lower-threshold variant) |
| `VAIDSHALA_PC_D3_FALLS_RISK_DRUG_BURDEN` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D3-FALLS-RISK-DRUG-BURDEN | PHARMA-Care v1 D3 (falls-risk drug taxonomy) |
| `VAIDSHALA_PC_D4_RESTRAINT_WITHOUT_SDM_CONSENT` | VAIDSHALA_TIER3 | VAIDSHALA-PC-D4-RESTRAINT-WITHOUT-SDM-CONSENT | Quality of Care Principles 2025 + Aged Care Act 2024 |

## Rules Queued (Deferred to clinical-author input)

Status legend:
- `queued` — published indicator definition exists, awaiting clinical-author input on Vaidshala-specific predicate scoping
- `blocked_substrate` — needs additional substrate primitive not yet in helper surface
- `blocked_clinical_author` — needs clinical-author input on threshold definition

### PHARMA-Care 5-domain (queued — 16 rules)

| domain | indicator | description | wave4a_status |
|---|---|---|---|
| D1 | Polypharmacy >=15 (severe) | Severe polypharmacy threshold | queued |
| D1 | Therapeutic duplication | Two active meds within same ATC class | queued |
| D1 | Drug-disease interaction count | Joins to KB-5 interaction registry | queued |
| D2 | Antipsychotic chronic >12w | Antipsychotic active >12 weeks without renewal review | queued |
| D2 | Benzodiazepine chronic >4w | Chronic benzodiazepine without renewal review | queued |
| D2 | PRN psychotropic frequency | PRN psychotropic dispensed >3 times per week | queued |
| D3 | ACB score >5 (high burden) | Higher-severity threshold | queued |
| D3 | Recent fall + sedative active | Falls-risk medication after recent fall (overlaps Tier 1) | queued |
| D3 | Orthostatic hypotension + antihypertensive intensified | Antihypertensive change + orthostatic concern open | blocked_substrate |
| D4 | Pain non-pharm trial gap | Chronic pain + analgesic + no non-pharm trial | queued |
| D4 | Sleep non-pharm trial gap | Hypnotic + no sleep-hygiene trial | queued |
| D4 | Constipation non-pharm trial gap | Laxative chronic + no fluid/fibre trial | queued |
| D5 | RACF admission medication review | New admission >14 days without medication review | queued |
| D5 | RMMR overdue 12 months | RMMR not done in 12 months (escalation tier) | queued |
| D5 | Discharge reconciliation 24h | Tighter SLA (24h vs 72h) | blocked_clinical_author |
| D5 | Inter-facility transfer reconciliation | Transfer event without paired reconciliation | queued |

### Standard 5 (Clinical Care 2026) evidence categories (queued — 5 rules)

Standard 5 of the Aged Care Quality and Safety Commission Standards
(2026) requires evidence across multiple clinical-care categories.
The 5 below are not yet authored:

| evidence_category | description | wave4a_status |
|---|---|---|
| S5-DETERIORATION | Clinical deterioration response evidence (vital-signs delta + escalation Event) | blocked_substrate |
| S5-INFECTION | Infection control evidence (suspected infection + antimicrobial stewardship trail) | queued |
| S5-NUTRITION | Nutrition/hydration risk evidence (MNA score decline + intervention trail) | blocked_substrate |
| S5-CONTINENCE | Continence assessment evidence (continence Event + reassessment cycle) | queued |
| S5-PRESSURE_INJURY | Pressure injury prevention evidence (Braden score + reposition trail) | blocked_substrate |

## Notes

- Total queued: 21 rules (16 PHARMA-Care + 5 Standard 5).
- All shipped rules cite the real PHARMA-Care v1 5-domain framework
  (UniSA Sluggett-led). Final domain identifiers will be confirmed
  against the published framework PDF — `TODO(clinical-author)`
  markers in the spec YAMLs and CQL files mark each location.
- Standard 5 evidence categories above are **real** Standard 5 (2026)
  clinical-care categories from the Aged Care Quality and Safety
  Commission published standards.
