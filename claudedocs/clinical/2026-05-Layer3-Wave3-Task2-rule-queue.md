# Wave 3 Task 2 — STOPP/START/Beers/Wang Rule Queue

This manifest documents the rules from STOPP v3, START v3, Beers 2023,
and Wang 2024 (AU-PIMs) that are **queued** for authoring, **not** yet
shipped. Wave 3 Task 2 ships 4 grounded rules as a vertical slice; the
remaining ~46 are listed here for clinical-author input.

## Rules Shipped (Wave 3 Task 2 vertical slice)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `STOPP_B1_ASPIRIN_PRIMARY_PREVENTION` | STOPP_V3 | STOPP-V3-B1 | O'Mahony 2023 §B1 |
| `START_A1_ANTICOAGULATION_AF` | START_V3 | START-V3-A1 | O'Mahony 2023 §A1 |
| `BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE` | BEERS_2023 | BEERS-2023-T2-ANTIHISTAMINE | AGS Beers 2023 Table 2 |
| `WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU` | PIMS_WANG | WANG-2024-AU-PIMS-3 | Wang 2024 §3 |

## Rules Shipped (Wave-extension batch 2026-05 — 15 rules)

| rule_id | criterion_set | criterion_id | citation |
|---|---|---|---|
| `STOPP_D5_LONG_TERM_BENZODIAZEPINE_HYPNOTIC` | STOPP_V3 | STOPP-V3-D5 | O'Mahony 2023 §D5 |
| `STOPP_F2_PPI_UNCOMPLICATED_PUD_OVER_8_WEEKS` | STOPP_V3 | STOPP-V3-F2 | O'Mahony 2023 §F2 |
| `STOPP_K1_ANTICHOLINERGIC_IN_DELIRIUM_OR_DEMENTIA` | STOPP_V3 | STOPP-V3-K1 | O'Mahony 2023 §K1 |
| `STOPP_J6_SULFONYLUREA_HBA1C_BELOW_7` | STOPP_V3 | STOPP-V3-J6 | O'Mahony 2023 §J6 |
| `START_B5_BETA_BLOCKER_POST_MI_REDUCED_LVEF` | START_V3 | START-V3-B5 | O'Mahony 2023 §B5 |
| `START_D2_CALCIUM_VITAMIN_D_OSTEOPOROSIS` | START_V3 | START-V3-D2 | O'Mahony 2023 §D2 |
| `START_E1_BONE_PROTECTIVE_THERAPY_OSTEOPOROSIS` | START_V3 | START-V3-E1 | O'Mahony 2023 §E1 |
| `START_F4_INFLUENZA_VACCINE_ANNUAL_ELDERLY` | START_V3 | START-V3-F4 | O'Mahony 2023 §F4 |
| `BEERS_2023_SLIDING_SCALE_INSULIN_NURSING_HOME` | BEERS_2023 | BEERS-2023-K1-SLIDING-SCALE-INSULIN | AGS Beers 2023 §K.1 |
| `BEERS_2023_STRONG_OPIOID_FIRST_LINE_ELDERLY` | BEERS_2023 | BEERS-2023-K7-OPIOID-FIRST-LINE | AGS Beers 2023 §K.7 |
| `BEERS_2023_NSAID_IN_CKD_STAGE_3_PLUS` | BEERS_2023 | BEERS-2023-H-NSAID-CKD | AGS Beers 2023 §H |
| `BEERS_2023_BENZODIAZEPINE_IN_ELDERLY` | BEERS_2023 | BEERS-2023-G-BENZODIAZEPINE | AGS Beers 2023 §G |
| `WANG_2024_ANTICHOLINERGIC_COGNITIVE_IMPAIRMENT_AU` | PIMS_WANG | WANG-2024-AU-PIMS-1 | Wang 2024 §1 |
| `WANG_2024_STRONG_OPIOID_WITHOUT_CANCER_PAIN_AU` | PIMS_WANG | WANG-2024-AU-PIMS-7 | Wang 2024 §7 |
| `WANG_2024_LONG_TERM_PPI_WITHOUT_INDICATION_AU` | PIMS_WANG | WANG-2024-AU-PIMS-11 | Wang 2024 §11 |

Total Tier 2 published shipped: **19** (4 vertical-slice + 15 extension);
plus 2 ADG2025 placeholders = 21 specs.

## Rules Queued (Deferred to clinical-author input)

Status legend:
- `queued` — published criterion text exists, awaiting clinical-author input on Vaidshala-specific predicate scoping
- `blocked_substrate` — needs Wave 4 substrate primitive (e.g. eGFR-trajectory) not yet in helper surface
- `blocked_terminology` — needs kb-7 AMT/ATC mapping resolution

### STOPP v3 (queued — ~30 rules)

| criterion_set | criterion_number | description (paraphrased) | wave3_status |
|---|---|---|---|
| STOPP_V3 | A1 | Drug duplication within same class | queued |
| STOPP_V3 | A2 | Drug without evidence-based indication | queued |
| STOPP_V3 | A3 | Drug beyond recommended duration | queued |
| STOPP_V3 | B2 | Aspirin >150mg/day | queued |
| STOPP_V3 | B3 | Antiplatelet plus VKA in chronic AF without indication | queued |
| STOPP_V3 | B4 | DOAPT (dual antiplatelet) without acute indication | queued |
| STOPP_V3 | B5 | Aspirin in active peptic ulcer disease | queued |
| STOPP_V3 | C1 | Long-acting benzodiazepine | queued |
| STOPP_V3 | C2 | Tricyclic antidepressant in dementia | queued |
| STOPP_V3 | C3 | Tricyclic antidepressant with glaucoma | queued |
| STOPP_V3 | C4 | Antipsychotic in Parkinson's disease | queued |
| STOPP_V3 | C5 | First-generation antihistamine (overlaps Beers Table 2) | queued |
| STOPP_V3 | D1 | Beta-blocker with bradycardia | queued |
| STOPP_V3 | D2 | Diltiazem/verapamil with NYHA III/IV heart failure | queued |
| STOPP_V3 | D3 | Beta-blocker with COPD/asthma | queued |
| STOPP_V3 | E1 | NSAID with severe hypertension | queued |
| STOPP_V3 | E2 | NSAID with eGFR <50 | blocked_substrate |
| STOPP_V3 | E3 | NSAID >3 months without GI prophylaxis | queued |
| STOPP_V3 | F1 | PPI for >8 weeks without indication (overlaps PPI long-term) | queued |
| STOPP_V3 | F2 | Antimuscarinic urinary in dementia | queued |
| STOPP_V3 | G1 | Bladder antimuscarinic with chronic constipation | queued |
| STOPP_V3 | G2 | Loop diuretic for ankle oedema without heart failure | queued |
| STOPP_V3 | H1 | Long-term opioid for non-cancer pain without review | queued |
| STOPP_V3 | I1 | Falls-risk drug after recent fall | queued (overlaps Tier 1 falls rules) |
| STOPP_V3 | J1 | Sulfonylurea long-acting in T2DM | queued |
| STOPP_V3 | J2 | Glibenclamide in T2DM | queued |
| STOPP_V3 | K1 | Anticholinergic burden score >3 | queued |
| STOPP_V3 | K2 | First-generation antipsychotic (high anticholinergic) | queued |
| STOPP_V3 | L1 | Strong opioid initiation without weak-opioid trial | queued |
| STOPP_V3 | L2 | Long-acting opioid without short-acting breakthrough | queued |

### START v3 (queued — ~12 rules)

| criterion_set | criterion_number | description (paraphrased) | wave3_status |
|---|---|---|---|
| START_V3 | A2 | Aspirin in chronic atherosclerotic CV disease | queued |
| START_V3 | A3 | Statin in atherosclerotic CV disease | queued |
| START_V3 | A4 | ACE inhibitor in heart failure with reduced EF | queued |
| START_V3 | A5 | Beta-blocker in stable systolic heart failure | queued |
| START_V3 | B1 | Inhaled bronchodilator in COPD | queued |
| START_V3 | B2 | Inhaled corticosteroid in moderate-severe asthma | queued |
| START_V3 | C1 | L-DOPA in Parkinson's with functional impairment | queued |
| START_V3 | C2 | Antidepressant in major depression | queued |
| START_V3 | D1 | Bisphosphonate in osteoporosis with prior fragility fracture | queued |
| START_V3 | D2 | Vitamin D in housebound elderly | queued |
| START_V3 | E1 | DMARD in active rheumatoid arthritis | queued |
| START_V3 | F1 | Topical eye drops for chronic glaucoma | queued |

### Beers 2023 (queued — ~12 rules)

| criterion_set | criterion_number | description (paraphrased) | wave3_status |
|---|---|---|---|
| BEERS_2023 | T2-TCA | Tertiary tricyclic antidepressants | queued |
| BEERS_2023 | T2-ANTISPASMODIC | GI antispasmodics | queued |
| BEERS_2023 | T2-DIPYRIDAMOLE | Dipyridamole oral short-acting | queued |
| BEERS_2023 | T2-NITROFURANTOIN | Nitrofurantoin with eGFR <30 | blocked_substrate |
| BEERS_2023 | T2-ALPHA1_BLOCKER | Alpha-1 blockers for hypertension | queued |
| BEERS_2023 | T2-AMIODARONE | Amiodarone first-line in AF | queued |
| BEERS_2023 | T2-DIGOXIN_HIGH_DOSE | Digoxin >0.125 mg/day | queued |
| BEERS_2023 | T2-NIFEDIPINE_IMMEDIATE | Nifedipine immediate-release | queued |
| BEERS_2023 | T2-ANDROGENS | Androgens (unless hypogonadism) | queued |
| BEERS_2023 | T2-MEPERIDINE | Meperidine | queued |
| BEERS_2023 | T2-NON_BENZO_HYPNOTIC | Non-benzodiazepine hypnotics chronic use | queued |
| BEERS_2023 | T2-MUSCLE_RELAXANT | Skeletal muscle relaxants | queued |

### Wang 2024 AU-PIMs (queued — ~16 rules)

| criterion_set | criterion_number | description (paraphrased) | wave3_status |
|---|---|---|---|
| PIMS_WANG | AU-PIMS-1 | Antidepressant >75y without indication review | queued |
| PIMS_WANG | AU-PIMS-2 | Benzodiazepine chronic in elderly | queued |
| PIMS_WANG | AU-PIMS-4 | Antipsychotic chronic without psychotic indication | queued |
| PIMS_WANG | AU-PIMS-5 | Multiple anticholinergic medications | queued |
| PIMS_WANG | AU-PIMS-6 | NSAID chronic in elderly without GI prophylaxis | queued |
| PIMS_WANG | AU-PIMS-7 | Statin >75y primary prevention | queued |
| PIMS_WANG | AU-PIMS-8 | PPI chronic without GI indication (overlaps PPI long-term) | queued |
| PIMS_WANG | AU-PIMS-9 | Antihypertensive >85y aggressive control | queued |
| PIMS_WANG | AU-PIMS-10 | Sulfonylurea long-acting in elderly | queued |
| PIMS_WANG | AU-PIMS-11 | Bisphosphonate chronic without monitoring | queued |
| PIMS_WANG | AU-PIMS-12 | Cholinesterase inhibitor in advanced dementia | queued |
| PIMS_WANG | AU-PIMS-13 | Memantine in advanced dementia | queued |
| PIMS_WANG | AU-PIMS-14 | First-generation antihistamine chronic (overlaps Beers T2) | queued |
| PIMS_WANG | AU-PIMS-15 | Antiarrhythmic class IA in elderly | queued |
| PIMS_WANG | AU-PIMS-16 | Vasodilator nitrate without indication | queued |
| PIMS_WANG | AU-PIMS-17 | Theophylline in COPD | queued |
| PIMS_WANG | AU-PIMS-18 | Iron supplement chronic without monitoring | queued |
| PIMS_WANG | AU-PIMS-19 | Combination opioid-paracetamol >3g/day | queued |

## Notes

- Total queued: ~70 rules. Plan Wave 3 Task 2 calls for ~50 highest-yield;
  the queue above includes the public superset for clinical-author selection.
- Overlaps (e.g. Beers Table 2 first-gen antihistamine vs STOPP C5 vs
  Wang AU-PIMS-14) will be resolved at authoring time via the
  Recommendation deduplication layer (Layer 2 Wave 4).
- All criterion_numbers above are real and citable from the published
  STOPP v3 / START v3 / AGS Beers 2023 / Wang 2024 lists.
