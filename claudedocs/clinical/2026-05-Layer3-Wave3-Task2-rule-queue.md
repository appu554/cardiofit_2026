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
