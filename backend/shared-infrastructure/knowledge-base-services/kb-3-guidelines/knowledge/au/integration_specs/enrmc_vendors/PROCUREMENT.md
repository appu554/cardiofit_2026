# eNRMC Vendor APIs — Procurement Runbook

**Status (2026-05-04):** 🔒 engagement-required
**Authority:** Per-vendor (commercial)
**Spec type:** Proprietary vendor API documentation
**Reproduction terms:** TBD per vendor — typically NDA-bound
**Layer 1 v2 spec section:** §3.1 (eNRMC integration)
**Spec deadline:** RACF eNRMC adoption mandatory by 2026-12-31

## Why engagement-required

Per ADHA November 2025 conformance status, 8 of 10 eNRMC vendors hold conformance certification. Vendor-specific API documentation, sandbox credentials, FHIR endpoints, and integration sandboxes are not publicly downloadable — they require a developer agreement / partnership.

Per Layer 1 v2 §3.1 sequencing recommendation:
> "MVP supports CSV export for any facility. V1 adds FHIR R4 API integration with the top 2-3 conformant vendors (likely Telstra Health MedPoint, MIMS, ResMed Software). V2 expands to all conformant vendors."

This means **MVP-stage Vaidshala does not strictly need any vendor partnership** — CSV export from any conformant eNRMC works. Vendor partnerships are a V1 / V2 acceleration, not an MVP blocker.

## Target vendors (priority order per spec §3.1)

| Rank | Vendor | Conformance | Engagement path |
|---|---|---|---|
| 1 | Telstra Health MedPoint | Conformant | Telstra Health partnership / developer portal |
| 2 | MIMS Australia | Conformant | MIMS API partner program |
| 3 | ResMed Software (formerly Software of Excellence / SoE) | Conformant | ResMed integration partner program |
| 4-8 | Other conformant vendors (per ADHA register) | Conformant | per-vendor |
| 9-10 | Two non-conformant (extended to 2026-04-01) | Pending | revisit post-conformance |

The 8-of-10 list is the authoritative reference — see ADHA conformance register procured in `adha_fhir/enrmc_conformance/`.

## What we expect to procure once partnership is agreed (per vendor)

1. API specification (Swagger / OpenAPI / FHIR CapabilityStatement)
2. Authentication / authorisation flow documentation
3. Sandbox credentials + test environment URL
4. FHIR profile / HL7 v2 message dictionary used by this vendor
5. Sample messages / payloads
6. Rate limits + SLA documentation
7. Vendor-specific quirks documentation (deviation from ADHA conformance baseline)

These would land in per-vendor subfolders once partnerships are agreed. Until then this folder remains placeholder + this runbook.

## Action required (commercial)

**Not blocked on engineering.** Recommended commercial sequence:
1. **MVP**: Build CSV-shaped eNRMC adapter (no vendor partnership needed). Pilot facility provides CSV export.
2. **V1**: Engage Telstra Health (largest market share for conformant aged-care eNRMC) for FHIR R4 partnership. Timeline: 3-6 months commercial discussions + 4-8 weeks integration.
3. **V2**: Expand to MIMS + ResMed Software once Telstra integration proves out.

## When to expand this folder

When the first vendor partnership lands, create `enrmc_vendors/<vendor_name>/` subfolder with its own PROCUREMENT.md tracking that vendor's API docs, sandbox setup, and integration runbook.

## Status legend

- 🔒 engagement-required — vendor partnership / NDA required
