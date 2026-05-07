# Layer 3 Wave 5 Task 1 — Tier 4 surveillance rule queue

**Status:** queue manifest for the 44 Tier 4 surveillance rules deferred from
the Wave 5 vertical slice.

**Vertical slice (this Wave 5 commit):** 6 of 50 rules shipped — 3 trajectory
defines (`Trajectory.cql`) + 3 lifecycle defines (`Lifecycle.cql`). All 6 pass
the toolchain (Stage 1 + two-gate + CompatibilityChecker + CDS Hooks emitter)
and are exercised by `tests/test_tier4_wave5_batch.py`.

**Wave-extension batch (2026-05):** 8 more shipped — 2 trajectory defines
(sodium delta-90d, BMI delta-180d) + 6 lifecycle defines (PPI / statin /
opioid / antimicrobial review-cycle + prescriber credential / prescribing
agreement expiry). Total Tier 4 shipped: **14** of ~50 target rules.

| rule_id | criterion_id | citation |
|---|---|---|
| `VAIDSHALA_T4_SODIUM_TRAJECTORY_DELTA_90D` | VAIDSHALA-T4-SODIUM-DELTA-90D | KDIGO/AHA dysnatraemia surveillance (TODO layer1-bind) |
| `VAIDSHALA_T4_BMI_TRAJECTORY_DELTA_180D` | VAIDSHALA-T4-BMI-DELTA-180D | AN-ACC v1.1 + Aged Care Quality Standard 5 nutrition |
| `VAIDSHALA_T4_PPI_REVIEW_OVERDUE_6MO` | VAIDSHALA-T4-PPI-REVIEW-OVERDUE-PLACEHOLDER | ADG 2025 PPI cycle (TODO layer1-bind) |
| `VAIDSHALA_T4_STATIN_REVIEW_OVERDUE_12MO` | VAIDSHALA-T4-STATIN-REVIEW-OVERDUE-PLACEHOLDER | ADG 2025 statin cycle (TODO layer1-bind) |
| `VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_3MO` | VAIDSHALA-T4-OPIOID-REVIEW-OVERDUE-PLACEHOLDER | ADG 2025 opioid cycle (TODO layer1-bind) |
| `VAIDSHALA_T4_ANTIMICROBIAL_REVIEW_OVERDUE_7D` | VAIDSHALA-T4-ANTIMICROBIAL-REVIEW-OVERDUE-PLACEHOLDER | ADG 2025 antimicrobial cycle + ACSQHC AMS (TODO layer1-bind) |
| `VAIDSHALA_T4_PRESCRIBER_CREDENTIAL_EXPIRING_WITHIN_30D` | VAIDSHALA-T4-PRESCRIBER-CREDENTIAL-EXPIRING-30D | Aged Care Act 2024 + Quality of Care Principles 2025 |
| `VAIDSHALA_T4_PRESCRIBING_AGREEMENT_EXPIRING_WITHIN_30D` | VAIDSHALA-T4-PRESCRIBING-AGREEMENT-EXPIRING-30D | RACGP / PSA aged-care care-plan agreement |

The remaining 44 rules are queued with a clear authoring path: each entry below
identifies the surveillance theme, the published source, the helper(s) the rule
will call, and the suppression class governing surface behaviour. Authors lift
each row into a spec + CQL define + 3 fixtures, mirroring the Wave 5 vertical
slice.

## Coverage estimate

| Cluster | Rules in slice | Rules queued | Total target |
|---|---|---|---|
| Trajectory | 3 | ~17 | ~20 |
| Lifecycle / deadline | 3 | ~12 | ~15 |
| Outcome monitoring | 0 | ~10 | ~10 |
| Hospital-discharge transition | 0 | ~5 | ~5 |
| **Total** | **6** | **44** | **50** |

## Trajectory cluster (queued — 17 rules)

| Rule ID candidate | Surveillance signal | Source | Primary helper | Suppression class |
|---|---|---|---|---|
| `VAIDSHALA_T4_HBA1C_TRAJECTORY_RISE_180D` | HbA1c rising over 180 days | RACGP T2DM 2024 | `IsTrending` + `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_BP_SYSTOLIC_RISE_60D` | Systolic BP rising over 60 days | NHFA 2024 hypertension | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_BP_DIASTOLIC_RISE_60D` | Diastolic BP rising over 60 days | NHFA 2024 hypertension | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_POTASSIUM_RISE_30D` | Potassium drift up over 30 days | KDIGO 2024 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_SODIUM_DRIFT_30D` | Sodium drift > +/- 5 mmol/L over 30 days | RCPA 2023 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_AKPS_TRAJECTORY_DECLINE_90D` | AKPS dropping over 90 days | AN-ACC v1.1 | `DeltaFromBaseline('akps', 90)` | substrate_state |
| `VAIDSHALA_T4_INR_TRAJECTORY_VOLATILITY_30D` | INR volatility (range > 1.0 over 30 days) | RACGP anticoag 2024 | `IsTrending('volatile')` | substrate_state |
| `VAIDSHALA_T4_BGL_TRAJECTORY_RISE_90D` | BGL trending up over 90 days | RACGP T2DM 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_FALL_FREQUENCY_CHANGE_30D` | Fall count > 1.5x baseline over 30 days | ACQSC fall reporting 2025 | `EventCountSince('fall')` | substrate_state |
| `VAIDSHALA_T4_DELIRIUM_INDICATOR_RISE_14D` | Delirium-screen indicator rising over 14 days | NSQHS Delirium 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_WEIGHT_LOSS_TRAJECTORY_180D` | Weight loss > 10% over 180 days | AN-ACC v1.1 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_PRESSURE_INJURY_INDICATOR_60D` | Pressure-injury indicator drift | NSQHS Pressure Injury 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_PAIN_SCORE_RISE_14D` | Pain-score trending up over 14 days | RACGP pain 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_NEURO_DECLINE_30D` | Neuro-check trajectory decline | RACGP dementia 2024 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_ORAL_INTAKE_DROP_14D` | Oral-intake drop over 14 days | AN-ACC nutrition 2025 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_INFECTION_INDICATOR_FREQUENCY_30D` | Infection-event uptick over 30 days | ACSQHC AMS 2024 | `EventCountSince('infection')` | substrate_state |
| `VAIDSHALA_T4_PSYCHOTROPIC_DOSE_DRIFT_60D` | Psychotropic MED drift over 60 days | RACGP BPSD 2024 | `IsTrending('mg_per_day')` | substrate_state |

## Lifecycle / deadline cluster (queued — 12 rules)

| Rule ID candidate | Lifecycle signal | Source | Primary helper | Suppression class |
|---|---|---|---|---|
| `VAIDSHALA_T4_RMMR_OVERDUE_24MO` | RMMR overdue beyond 24 months | RACGP RMMR 2024 | `LatestHospitalDischargeOlderThan` + `HasRecentEventOfType` | recently_actioned |
| `VAIDSHALA_T4_BPSD_REVIEW_OVERDUE_12W` | BPSD review overdue 12 weeks | RACGP BPSD 2024 | `HasActiveConcernType('BPSD_REVIEW_DONE_12W')` | recently_actioned |
| `VAIDSHALA_T4_OPIOID_REVIEW_OVERDUE_28D` | Opioid trial review overdue 28 days | RACGP opioid 2024 | `HasActiveConcernType('OPIOID_TRIAL_REVIEW_DONE')` | recently_actioned |
| `VAIDSHALA_T4_BENZO_REVIEW_OVERDUE_28D` | Benzodiazepine review overdue 28 days | RACGP benzodiazepine 2024 | `HasActiveConcernType('BENZO_REVIEW_DONE')` | recently_actioned |
| `VAIDSHALA_T4_INSULIN_TITRATION_REVIEW_OVERDUE` | Insulin titration review overdue | RACGP T2DM 2024 | `HasActiveConcernType` | recently_actioned |
| `VAIDSHALA_T4_ANTICOAG_INR_OVERDUE` | Anticoag INR check overdue | RACGP anticoag 2024 | `ObservationOverdueBy` | recently_actioned |
| `VAIDSHALA_T4_CONSENT_EXPIRING_WITHIN_60D` | Restrictive practice consent within 60 days | Quality of Care Principles 2025 | `ConsentExpiringWithin` | recently_actioned |
| `VAIDSHALA_T4_CARE_PLAN_REVIEW_OVERDUE_3MO` | Care plan review overdue 3 months | Aged Care Act 2024 | `HasActiveConcernType` | recently_actioned |
| `VAIDSHALA_T4_GP_VISIT_OVERDUE_90D` | GP visit overdue 90 days | RACGP RMMR 2024 | `LatestHospitalDischargeOlderThan` | recently_actioned |
| `VAIDSHALA_T4_PHARMACIST_REVIEW_OVERDUE_180D` | Pharmacist review overdue 180 days | ACOP 2026 | `HasActiveConcernType('ACOP_REVIEW_DONE')` | recently_actioned |
| `VAIDSHALA_T4_DIETITIAN_REVIEW_OVERDUE_90D` | Dietitian review overdue 90 days | AN-ACC nutrition 2025 | `HasActiveConcernType` | recently_actioned |
| `VAIDSHALA_T4_PALLIATIVE_TRANSITION_REVIEW_OVERDUE` | Palliative-transition review overdue | RACGP palliative 2024 | `IsPalliative` + `HasActiveConcernType` | recently_actioned |

## Outcome-monitoring cluster (queued — 10 rules)

| Rule ID candidate | Outcome monitored | Source | Primary helper | Suppression class |
|---|---|---|---|---|
| `VAIDSHALA_T4_BP_OUTCOME_OFF_TARGET_30D` | BP off-target for 30 days post Tier-2 BP rec | NHFA 2024 | `DeltaFromBaseline` after `RuleFiredFor` | substrate_state |
| `VAIDSHALA_T4_K_OUTCOME_OFF_TARGET_30D` | Potassium not back in range 30 days post Tier-1 fire | KDIGO 2024 | `DeltaFromBaseline` | substrate_state |
| `VAIDSHALA_T4_BGL_OUTCOME_OFF_TARGET_60D` | BGL off-target post titration | RACGP T2DM 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_FALL_OUTCOME_RECURRENCE_30D` | Fall recurrence within 30 days of Tier-2 fall-prevent fire | ACQSC fall 2025 | `EventCountSince('fall')` | substrate_state |
| `VAIDSHALA_T4_BPSD_OUTCOME_NO_IMPROVEMENT_28D` | BPSD episodes still rising 28 days after Tier-2 BPSD fire | RACGP BPSD 2024 | `EventCountSince` | substrate_state |
| `VAIDSHALA_T4_PAIN_OUTCOME_NOT_CONTROLLED_14D` | Pain score still rising 14 days post analgesic | RACGP pain 2024 | `IsTrending` | substrate_state |
| `VAIDSHALA_T4_ANTIPSYCHOTIC_TAPER_OUTCOME_60D` | Antipsychotic dose not tapered 60 days post Tier-2 | RACGP BPSD 2024 | `MorphineEquivalentMgPerDay` analog | substrate_state |
| `VAIDSHALA_T4_PPI_DEPRESCRIBE_OUTCOME_90D` | PPI not deprescribed 90 days post Tier-2 deprescribe fire | RACGP deprescribing 2024 | `IsActiveOnDate` + `RuleFiredFor` | substrate_state |
| `VAIDSHALA_T4_ANTIBIOTIC_AMS_OUTCOME_14D` | Antibiotic still active 14 days post AMS prompt | ACSQHC AMS 2024 | `IsActiveOnDate` | substrate_state |
| `VAIDSHALA_T4_RESTRAINT_OUTCOME_REDUCTION_30D` | Restrictive practice not reduced 30 days post Tier-3 fire | Quality of Care Principles 2025 | `HasActiveConcernType` | substrate_state |

## Hospital-discharge transition cluster (queued — 5 rules; depends on Layer 2 Wave 4)

| Rule ID candidate | Transition surveillance | Source | Primary helper | Suppression class |
|---|---|---|---|---|
| `VAIDSHALA_T4_DISCHARGE_MED_RECONCILIATION_OUTCOME_72H` | Reconciliation not started within 72h of discharge event | Aged Care Act 2024 + RACGP RMMR 2024 | `HasCompletedReconciliationFor` | recently_actioned |
| `VAIDSHALA_T4_DISCHARGE_HIGH_RISK_MED_DRIFT_14D` | New high-risk med (Beers) appears within 14 days of discharge | Beers 2023 | `IsHighRiskFromBeers` + `HasRecentEventOfType('discharge')` | substrate_state |
| `VAIDSHALA_T4_DISCHARGE_OBSERVATION_GAP_7D` | No vital observations recorded within 7 days of discharge | NSQHS Recognising Deterioration 2024 | `LatestObservationAt` | substrate_state |
| `VAIDSHALA_T4_DISCHARGE_GP_HANDOVER_OVERDUE_14D` | No GP letter / handover within 14 days of discharge | RACGP discharge 2024 | `HasRecentEventOfType('gp_handover')` | recently_actioned |
| `VAIDSHALA_T4_DISCHARGE_FALL_RISK_RESCORE_14D` | Fall risk not re-scored within 14 days of discharge | ACQSC fall 2025 | `HasRecentEventOfType('fall_risk_score')` | recently_actioned |

## Authoring procedure (per queued rule)

1. Copy a Wave 5 vertical-slice spec as a template (e.g.
   `egfr-trajectory-decline-90d.yaml`).
2. Fill in `rule_id`, `criterion_id`, summary, trigger_sources,
   state_machine_references, suppressions, test_cases.
3. Add the CQL define to `Trajectory.cql` (trajectory cluster) or
   `Lifecycle.cql` (lifecycle cluster); create `Outcome.cql` and
   `DischargeTransition.cql` for those clusters when authoring lands.
4. Author 3 fixtures (positive / negative / suppression) under
   `tier-4-surveillance/fixtures/`.
5. Add the rule_id to `EXPECTED_RULE_IDS` in
   `tests/test_tier4_wave5_batch.py` (or a Wave-5b sibling file once
   the corpus exceeds ~15 rules).
6. Run `pytest tests/test_tier4_wave5_batch.py -q` until green; then
   `pytest -q` for full-suite confirmation.

## Wave-level acceptance status

- Vertical slice (6 rules): **green** — Stage 1 + two-gate +
  CompatibilityChecker + CDS Hooks emitter all pass.
- Queue (44 rules): authored offline / next sprint; this manifest is
  the authoritative tracking document for that work.
- Total target by Wave 5 exit: 50 Tier 4 surveillance rules ACTIVE,
  bringing total rule-library size to ~200 (per plan Wave 5 exit
  criterion).
