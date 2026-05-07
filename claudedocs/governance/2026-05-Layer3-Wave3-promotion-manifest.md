# Wave 3 Governance Promotion Manifest

This manifest documents the 6 Tier 2 deprescribing rules promoted
through the Stage 5 governance pipeline as part of the Wave 3 vertical
slice. Each rule round-trips through:

1. Stage 1 — `rule_specification_validator` (schema + 4 classic-error checks)
2. Stage 2 — `two_gate_validator` (snapshot-semantics + substrate-semantics gates)
3. CompatibilityChecker — must show ACTIVE
4. CDS Hooks v2.0 emitter — emit a valid CDS Hooks response
5. GovernancePromoter — Ed25519-sign + write signed package to KB-4

## Promoted rules

| # | rule_id | criterion_set | criterion_id | citation source | Ed25519 signed package |
|---|---|---|---|---|---|
| 1 | `STOPP_B1_ASPIRIN_PRIMARY_PREVENTION` | STOPP_V3 | STOPP-V3-B1 | O'Mahony et al., Eur Geriatr Med 2023 §B1 (published) | `kb-4-patient-safety/governance/signed/STOPP_B1_ASPIRIN_PRIMARY_PREVENTION-2.0.json` |
| 2 | `START_A1_ANTICOAGULATION_AF` | START_V3 | START-V3-A1 | O'Mahony et al., Eur Geriatr Med 2023 §A1 (published) | `kb-4-patient-safety/governance/signed/START_A1_ANTICOAGULATION_AF-2.0.json` |
| 3 | `BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE` | BEERS_2023 | BEERS-2023-T2-ANTIHISTAMINE | AGS Beers Criteria 2023 Table 2 (J Am Geriatr Soc 2023; 71:2052-2081) | `kb-4-patient-safety/governance/signed/BEERS_2023_ANTICHOLINERGIC_CHRONIC_USE-2.0.json` |
| 4 | `WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU` | PIMS_WANG | WANG-2024-AU-PIMS-3 | Wang 2024 AU-PIMs §3 (published) | `kb-4-patient-safety/governance/signed/WANG_2024_ANTIPSYCHOTIC_DEMENTIA_AU-2.0.json` |
| 5 | `ADG2025_PPI_STEP_DOWN_PROTOCOL` | VAIDSHALA_TIER2 | `ADG2025-PPI-STEPDOWN-PLACEHOLDER` (TODO(layer1-bind)) | ADG 2025 PPI step-down theme (recommendation_id pending Pipeline-2) | `kb-4-patient-safety/governance/signed/ADG2025_PPI_STEP_DOWN_PROTOCOL-2.0.json` |
| 6 | `ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW` | VAIDSHALA_TIER2 | `ADG2025-APS-12W-PLACEHOLDER` (TODO(layer1-bind)) | ADG 2025 antipsychotic-BPSD 12-week review (recommendation_id pending Pipeline-2) | `kb-4-patient-safety/governance/signed/ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW-2.0.json` |

## Signing keys

- **Platform key**: Ed25519 private key seeded from
  `L3_PROMOTE_PLATFORM_KEY_B64` (32-byte base64 seed). Test runs in
  `test_tier2_wave3_batch.py` use a freshly-generated key per pytest
  fixture.
- **Approver signatures (dual)**: `CLINICAL_REVIEWER` +
  `MEDICAL_DIRECTOR`. Production signing is gated on a real reviewer
  workflow (Layer 4); test runs use placeholder signer IDs.

## Verification

The batch test
`shared/cql-toolchain/tests/test_tier2_wave3_batch.py::test_wave3_governance_promotion_six_signed_packages`
asserts:

1. All 6 rules produce a signed package on disk.
2. All 6 packages have distinct `content_sha` values (no duplicate hashing).
3. Each package contains both reviewer signatures.

## Layer 1 binding follow-up (rules 5, 6)

Once kb-3 Pipeline-2 emits real ADG 2025 recommendation IDs:

1. Replace `ADG2025-PPI-STEPDOWN-PLACEHOLDER` and
   `ADG2025-APS-12W-PLACEHOLDER` with the canonical ADG 2025 IDs in
   the spec YAMLs.
2. Re-promote both rules — the `content_sha` will change because
   `criterion_id` is part of the canonicalised hash.
3. Update this manifest to reflect the new `criterion_id` values.
4. Mark the queued ADG 2025 rules in
   `claudedocs/clinical/2026-05-Layer3-Wave3-Task3-adg2025-rule-queue.md`
   as ready-to-author.

## Next wave

The remaining ~70 STOPP/START/Beers/Wang criteria + ~18 ADG 2025 themes
are queued in:

- `claudedocs/clinical/2026-05-Layer3-Wave3-Task2-rule-queue.md`
- `claudedocs/clinical/2026-05-Layer3-Wave3-Task3-adg2025-rule-queue.md`

Promotion of the queued rules is gated on clinical-author input
(Task 2) and Pipeline-2 completion (Task 3).
