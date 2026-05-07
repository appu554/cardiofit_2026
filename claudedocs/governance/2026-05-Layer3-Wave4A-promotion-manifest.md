# Wave 4A Governance Promotion Manifest

This manifest documents the 8 Tier 3 quality-gap rules promoted
through the Stage 5 governance pipeline as part of the Wave 4A
vertical slice. Each rule round-trips through:

1. Stage 1 — `rule_specification_validator` (schema + 4 classic-error checks)
2. Stage 2 — `two_gate_validator` (snapshot-semantics + substrate-semantics gates)
3. CompatibilityChecker — must show ACTIVE
4. CDS Hooks v2.0 emitter — emit a valid CDS Hooks response
5. GovernancePromoter — Ed25519-sign + write signed package to KB-4

## Promoted rules

| # | rule_id | criterion_set | criterion_id | citation source | Ed25519 signed package |
|---|---|---|---|---|---|
| 1 | `VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE` | VAIDSHALA_TIER3 | `VAIDSHALA-PC-D2-ANTIPSYCHOTIC-PREVALENCE` (TODO(clinical-author)) | PHARMA-Care v1 D2 (UniSA Sluggett-led, published) | `kb-4-patient-safety/governance/signed/VAIDSHALA_PC_D2_ANTIPSYCHOTIC_PREVALENCE-2.0.json` |
| 2 | `VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS` | VAIDSHALA_TIER3 | `VAIDSHALA-PC-D1-POLYPHARMACY-10PLUS` (TODO(clinical-author)) | PHARMA-Care v1 D1 | `kb-4-patient-safety/governance/signed/VAIDSHALA_PC_D1_POLYPHARMACY_10PLUS-2.0.json` |
| 3 | `VAIDSHALA_PC_D3_ACB_ABOVE_3` | VAIDSHALA_TIER3 | `VAIDSHALA-PC-D3-ACB-ABOVE-3` (TODO(clinical-author)) | PHARMA-Care v1 D3 | `kb-4-patient-safety/governance/signed/VAIDSHALA_PC_D3_ACB_ABOVE_3-2.0.json` |
| 4 | `VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM` | VAIDSHALA_TIER3 | `VAIDSHALA-PC-D4-BPSD-FIRST-LINE-NONPHARM` (TODO(clinical-author)) | PHARMA-Care v1 D4 | `kb-4-patient-safety/governance/signed/VAIDSHALA_PC_D4_BPSD_FIRST_LINE_NONPHARM-2.0.json` |
| 5 | `VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H` | VAIDSHALA_TIER3 | `VAIDSHALA-TX-DISCHARGE-NOT-RECONCILED-72H` | Aged Care Quality Standard 5 (Clinical Care 2026) — transitions of care | `kb-4-patient-safety/governance/signed/VAIDSHALA_TX_DISCHARGE_NOT_RECONCILED_72H-2.0.json` |
| 6 | `VAIDSHALA_TX_RMMR_OVERDUE_6MO` | VAIDSHALA_TIER3 | `VAIDSHALA-TX-RMMR-OVERDUE-6MO` | RACGP / Pharmaceutical Society of Australia 6-month RMMR follow-up cycle (published) | `kb-4-patient-safety/governance/signed/VAIDSHALA_TX_RMMR_OVERDUE_6MO-2.0.json` |
| 7 | `VAIDSHALA_ANACC_FUNCTIONAL_DECLINE` | VAIDSHALA_TIER3 | `VAIDSHALA-ANACC-FUNCTIONAL-DECLINE` | AN-ACC v1.1 AKPS functional-status indicator (Australian Government published) | `kb-4-patient-safety/governance/signed/VAIDSHALA_ANACC_FUNCTIONAL_DECLINE-2.0.json` |
| 8 | `VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE` | VAIDSHALA_TIER3 | `VAIDSHALA-ANACC-BEHAVIOURAL-EVIDENCE` | AN-ACC v1.1 class 9-13 behavioural indicators | `kb-4-patient-safety/governance/signed/VAIDSHALA_ANACC_BEHAVIOURAL_EVIDENCE-2.0.json` |

## Signing keys

- **Platform key**: Ed25519 private key seeded from
  `L3_PROMOTE_PLATFORM_KEY_B64` (32-byte base64 seed). Test runs in
  `test_tier3_wave4a_batch.py` use a freshly-generated key per pytest
  fixture.
- **Approver signatures (dual)**: `CLINICAL_REVIEWER` +
  `MEDICAL_DIRECTOR`. Production signing is gated on a real reviewer
  workflow (Layer 4); test runs use placeholder signer IDs.

## Verification

The batch test
`shared/cql-toolchain/tests/test_tier3_wave4a_batch.py::test_wave4a_governance_promotion_eight_signed_packages`
asserts:

1. All 8 rules produce a signed package on disk.
2. All 8 packages have distinct `content_sha` values.
3. Each package contains both reviewer signatures.

## Clinical-author binding follow-up

The 4 PHARMA-Care rules (#1-4) carry `TODO(clinical-author)` markers
on each `criterion_id`. Once the published PHARMA-Care v1 framework
PDF is checked into `docs/clinical/`, the binding sequence is:

1. Replace the `VAIDSHALA-PC-Dn-*` placeholder ids with the canonical
   PHARMA-Care v1 published indicator ids in the spec YAMLs.
2. Re-promote affected rules — the `content_sha` will change because
   `criterion_id` is part of the canonicalised hash.
3. Update this manifest to reflect the new `criterion_id` values.

## Next wave

The remaining 42 Tier 3 rules (21 PHARMA-Care/S5 + 13 transition + 8
AN-ACC) are queued in:

- `claudedocs/clinical/2026-05-Layer3-Wave4A-Task1-quality-gap-rule-queue.md`
- `claudedocs/clinical/2026-05-Layer3-Wave4A-Task2-transition-rule-queue.md`
- `claudedocs/clinical/2026-05-Layer3-Wave4A-Task3-anacc-rule-queue.md`

Promotion of the queued rules is gated on clinical-author input.
