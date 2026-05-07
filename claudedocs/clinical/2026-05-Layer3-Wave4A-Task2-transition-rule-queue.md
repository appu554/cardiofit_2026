# Wave 4A Task 2 — Care-Transition Quality-Gap Rule Queue

This manifest documents the care-transition quality-gap rules
**queued** for authoring, **not** yet shipped. Wave 4A Task 2 ships
2 grounded rules as a vertical slice; the remaining 13 are listed
here for clinical-author input.

## Rules Shipped (Wave 4A Task 2 vertical slice)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H` | VAIDSHALA_TIER3 | VAIDSHALA-TX-DISCHARGE-NOT-RECONCILED-72H | Aged Care Quality Standard 5 — Clinical Care 2026, transitions of care |
| `VAIDSHALA_TX_RMMR_OVERDUE_6MO` | VAIDSHALA_TIER3 | VAIDSHALA-TX-RMMR-OVERDUE-6MO | RACGP / Pharmaceutical Society of Australia 6-month RMMR follow-up cycle (published) |

## Rules Queued (Deferred to clinical-author input)

Status legend:
- `queued` — care-transition gap definition is established, awaiting
  clinical-author input on Vaidshala-specific Event-taxonomy bindings
- `blocked_substrate` — needs additional substrate primitive
- `blocked_layer2` — depends on Layer 2 Event taxonomy extension

### Hospital discharge transition gaps (queued — 5 rules)

| rule_id_candidate | description | wave4a_status |
|---|---|---|
| `VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_24H` | Tighter 24h reconciliation SLA (high-risk meds) | queued |
| `VAIDSHALA_TX_DISCHARGE_HIGH_RISK_MED_NEW` | New high-risk med (anticoagulant / opioid / insulin) on discharge without prescriber notification Event | queued |
| `VAIDSHALA_TX_DISCHARGE_DUPLICATE_PRESCRIPTION` | Discharge med list duplicates a pre-admission med (different brand/strength) | queued |
| `VAIDSHALA_TX_DISCHARGE_DOSE_CHANGE_UNCONFIRMED` | Discharge dose differs from pre-admission without explicit confirmation Event | queued |
| `VAIDSHALA_TX_DISCHARGE_FOLLOWUP_GP_OVERDUE` | No GP follow-up Event within 14 days post-discharge | queued |

### RACF admission transition gaps (queued — 4 rules)

| rule_id_candidate | description | wave4a_status |
|---|---|---|
| `VAIDSHALA_TX_NEW_ADMISSION_REVIEW_14D` | New RACF admission Event with no medication review within 14 days | queued |
| `VAIDSHALA_TX_NEW_ADMISSION_HIGH_RISK_PROFILE` | New admission with high anticholinergic burden + no review scheduled | queued |
| `VAIDSHALA_TX_NEW_ADMISSION_BPSD_FIRST_LINE` | New admission with antipsychotic active + no non-pharm trial documented | queued |
| `VAIDSHALA_TX_RESPITE_TO_PERMANENT_RECONCILIATION` | Respite-to-permanent-stay transition without medication reconciliation | blocked_layer2 |

### Care-planning + RMMR follow-up gaps (queued — 4 rules)

| rule_id_candidate | description | wave4a_status |
|---|---|---|
| `VAIDSHALA_TX_RMMR_OVERDUE_12MO` | Escalation tier — RMMR overdue 12 months | queued |
| `VAIDSHALA_TX_CARE_PLAN_REVIEW_OVERDUE` | Care plan review Event >6mo ago | queued |
| `VAIDSHALA_TX_DEPRESCRIBING_RECOMMENDATION_DEFERRED_4WK` | Deprescribing recommendation deferred >4 weeks without follow-up | queued |
| `VAIDSHALA_TX_PRESCRIBER_HANDOVER_INCOMPLETE` | Prescriber change Event without medication handover completion | blocked_layer2 |

## Notes

- Total queued: 13 rules.
- All rules consume real Layer 2 Event taxonomy values
  (`hospital_discharge`, `admission_to_facility`, `care_planning_meeting`,
  `rmmr_followup`, `behavioural_incident`, `anacc_reassessment`).
- Standard 5 (Clinical Care 2026) is the operative quality-of-care
  framework reference; transitions-of-care evidence is one of its
  required categories.
