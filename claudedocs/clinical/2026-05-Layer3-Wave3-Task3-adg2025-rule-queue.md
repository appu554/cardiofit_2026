# Wave 3 Task 3 — ADG 2025 Rule Queue

This manifest documents the ADG 2025 (Australian Deprescribing Guideline
2025) rules that are **queued** for authoring, **not** yet shipped.

## Critical blocker

**Authoring of the queued rules is BLOCKED on Layer 1 Pipeline-2
extraction completing.** The ADG 2025 source PDFs landed in the kb-3
corpus per the Layer 1 audit (Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_
2026-04-30.md), but Pipeline-2 has NOT yet emitted machine-readable
recommendation IDs. Without those IDs:

- We cannot cite ADG 2025 recommendations with verifiable identifiers.
- Synthetic IDs would be fabrication and fail clinical-review at L6.

Wave 3 Task 3 vertical slice ships TWO grounded rules that map to the
most clinically-important ADG 2025 themes (PPI step-down,
antipsychotic-BPSD review). Both carry `TODO(layer1-bind)` markers on
their `criterion_id` fields so V1 can grep + replace once Pipeline-2
delivers real IDs.

## Rules Shipped (Wave 3 Task 3 vertical slice)

| rule_id | criterion_id (placeholder) | ADG 2025 theme |
|---|---|---|
| `ADG2025_PPI_STEP_DOWN_PROTOCOL` | `ADG2025-PPI-STEPDOWN-PLACEHOLDER` (TODO(layer1-bind)) | PPI step-down |
| `ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW` | `ADG2025-APS-12W-PLACEHOLDER` (TODO(layer1-bind)) | Antipsychotic-BPSD review |

## Rules Queued (~18) — blocked on Pipeline-2

These are the ADG 2025 themes that are publicly known but for which we
do not yet have Pipeline-2-extracted recommendation IDs. Each row's
`criterion_id` is `TODO(layer1-bind)` until Pipeline-2 emits.

| ADG 2025 theme | proposed rule_id | wave3_status | notes |
|---|---|---|---|
| Statin primary prevention >75y | `ADG2025_STATIN_PRIMARY_PREVENTION_75Y` | queued (blocked Pipeline-2) | Overlaps Tier 1 STATIN_PALLIATIVE_PRIMARY_PREVENTION |
| Bisphosphonate beyond 5y | `ADG2025_BISPHOSPHONATE_BEYOND_5Y` | queued (blocked Pipeline-2) | Needs duration helper extension |
| Calcium without osteoporosis | `ADG2025_CALCIUM_NO_OSTEOPOROSIS` | queued (blocked Pipeline-2) | Often paired with vit D |
| Vitamin D without deficiency/osteoporosis | `ADG2025_VITAMIND_NO_INDICATION` | queued (blocked Pipeline-2) | — |
| Anticholinergic burden score >3 | `ADG2025_ANTICHOLINERGIC_BURDEN_HIGH` | queued (blocked Pipeline-2) | Needs ACB scoring substrate; overlaps Beers 2023 |
| Opioid taper post-acute (chronic non-cancer) | `ADG2025_OPIOID_TAPER_POST_ACUTE` | queued (blocked Pipeline-2) | Schedule 8 routing required |
| Benzodiazepine taper in elderly | `ADG2025_BENZODIAZEPINE_TAPER_ELDERLY` | queued (blocked Pipeline-2) | Overlaps Wave 2 BENZODIAZEPINE_BPSD consent slice |
| Z-drug taper in elderly | `ADG2025_ZDRUG_TAPER_ELDERLY` | queued (blocked Pipeline-2) | — |
| Antihypertensive over-treatment (non-palliative) | `ADG2025_ANTIHTN_OVERTREATMENT` | queued (blocked Pipeline-2; deferred Wave 4 substrate) | Overlaps Tier 1 palliative variant |
| SSRI chronic without symptom benefit | `ADG2025_SSRI_NO_RESPONSE_12M` | queued (blocked Pipeline-2) | Needs depression-response observation kind |
| NSAID chronic in elderly | `ADG2025_NSAID_CHRONIC_ELDERLY` | queued (blocked Pipeline-2) | Overlaps Wave 2 AKI rules |
| Diuretic without HF/HTN indication | `ADG2025_DIURETIC_NO_INDICATION` | queued (blocked Pipeline-2) | — |
| Cholinesterase inhibitor in advanced dementia | `ADG2025_CHEI_ADVANCED_DEMENTIA` | queued (blocked Pipeline-2) | Needs dementia-stage substrate |
| Memantine in advanced dementia | `ADG2025_MEMANTINE_ADVANCED_DEMENTIA` | queued (blocked Pipeline-2) | — |
| Sulfonylurea taper in elderly | `ADG2025_SULFONYLUREA_TAPER` | queued (blocked Pipeline-2) | Hypoglycaemia risk |
| Inhaled corticosteroid in COPD without exacerbations | `ADG2025_ICS_COPD_NO_EXAC` | queued (blocked Pipeline-2) | Needs exacerbation observation kind |
| Iron supplement after replenishment | `ADG2025_IRON_AFTER_REPLENISH` | queued (blocked Pipeline-2) | Needs ferritin baseline-aware substrate |
| Multivitamin without indication | `ADG2025_MULTIVITAMIN_NO_INDICATION` | out_of_scope | Wave 3 limited to medications with measurable harm |

## Acceptance gate

Before any of the queued rules may be authored:

1. kb-3 Pipeline-2 extraction MUST emit a JSON manifest with:
   - `recommendation_id` (canonical ADG 2025 ID, e.g. `ADG2025-R042`)
   - `recommendation_text` (verbatim or paraphrased per ADG 2025 licence)
   - `evidence_grade`
   - `medication_class`
2. The Wave 3 ADG 2025 mapping CSV
   (`claudedocs/clinical/2026-05-Layer3-ADG2025-mapping.csv`) MUST be
   regenerated with real IDs replacing every `TODO(layer1-bind)` token.
3. The two shipped rules' `criterion_id` fields MUST be patched with
   real IDs (single greppable replacement per rule).

Until Pipeline-2 completes, attempts to author the queued rules
constitute fabrication and MUST be rejected at clinical review.
