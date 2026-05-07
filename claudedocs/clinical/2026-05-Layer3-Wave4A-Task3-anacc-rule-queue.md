# Wave 4A Task 3 — AN-ACC Defensibility Rule Queue

This manifest documents the AN-ACC (Australian National Aged Care
Classification) defensibility rules **queued** for authoring, **not**
yet shipped. Wave 4A Task 3 ships 2 grounded rules as a vertical
slice; the remaining 8 are listed here for clinical-author input.

AN-ACC is the published Australian Government residential aged-care
funding classification (replaced ACFI in October 2022). The 13-class
taxonomy is the canonical funding-determination model. These rules
do **not** recommend a class — the rule output is workflow exhaust
that surfaces evidence packets supporting AN-ACC class reassessment.

## Rules Shipped (Wave 4A Task 3 vertical slice)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `VAIDSHALA_ANACC_FUNCTIONAL_DECLINE` | VAIDSHALA_TIER3 | VAIDSHALA-ANACC-FUNCTIONAL-DECLINE | AN-ACC v1.1 functional-status indicator (AKPS) |
| `VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE` | VAIDSHALA_TIER3 | VAIDSHALA-ANACC-BEHAVIOURAL-EVIDENCE | AN-ACC class 9-13 behavioural indicators |

## Rules Queued (Deferred to clinical-author input)

Status legend:
- `queued` — AN-ACC class indicator is established, awaiting
  clinical-author input on Vaidshala-specific evidence-Event bindings
- `blocked_substrate` — needs additional baseline_state primitive
- `blocked_clinical_author` — needs clinical-author confirmation of
  AN-ACC v1.1 vs v1.2 indicator semantics

### AN-ACC class-evidence rules (queued — 8 rules)

| rule_id_candidate | description | anacc_class_target | wave4a_status |
|---|---|---|---|
| `VAIDSHALA_ANACC_COGNITIVE_DECLINE` | PAS-CIS / RUDAS score decline >2 points in 90 days | classes 5-8 | blocked_substrate |
| `VAIDSHALA_ANACC_ADL_DECLINE` | ADL assessment score decline (Barthel / Modified Barthel) | classes 6-8 | blocked_substrate |
| `VAIDSHALA_ANACC_PALLIATIVE_TRANSITION` | Care-intensity transition to palliative without class reassessment | class 1 | queued |
| `VAIDSHALA_ANACC_FALLS_FREQUENCY` | >3 fall Events in 90 days without class reassessment | classes 7-13 | queued |
| `VAIDSHALA_ANACC_PRESSURE_INJURY_DEVELOPMENT` | New stage 2+ pressure injury Event without class reassessment | classes 6-13 | blocked_substrate |
| `VAIDSHALA_ANACC_WEIGHT_LOSS_SIGNIFICANT` | >10% weight loss in 90 days without class reassessment | classes 8-13 | blocked_substrate |
| `VAIDSHALA_ANACC_WANDERING_INCIDENT_FREQUENCY` | >3 wandering incident Events in 30 days | classes 9-13 | queued |
| `VAIDSHALA_ANACC_MEDICATION_BURDEN_INCREASE` | Active medication count increased >3 in 30 days without class reassessment | classes 7-8 | queued |

## Notes

- Total queued: 8 rules.
- AN-ACC class taxonomy (1-13) is **real** and citable from the
  Australian Government Department of Health AN-ACC v1.1 published
  classification model. Class 1 = palliative; classes 2-4 =
  independent / low-care; classes 5-8 = standard residential care
  with cognitive / functional gradients; classes 9-13 = high-acuity
  behavioural and complex care.
- AKPS (Australian-modified Karnofsky Performance Scale) is the
  published AN-ACC v1.1 functional-status indicator.
- Output of these rules is **evidence**, not a class assignment.
  AIHW assessor expectations require that surfaced evidence packets
  contain timestamped events and observed score deltas — the rule
  bodies above conform to that contract.
