# Dispensing Pharmacy Software Vendor APIs — Procurement Runbook

**Status (2026-05-04):** 🔒 engagement-required
**Authority:** Per-vendor (commercial)
**Spec type:** Proprietary vendor API documentation
**Reproduction terms:** TBD per vendor — typically NDA-bound
**Layer 1 v2 spec section:** §3.4 (dispensing pharmacy DAA timing)
**Spec deadline:** Wave 4 priority — DAA timing layer is Vaidshala's defensible commercial moat per spec ("owning the dispensing pharmacy coordination layer is unfashionable but defensible")

## Why engagement-required

Per Layer 1 v2 §3.4, Australian community pharmacy software is a fragmented market with no clean modern API. Per spec verbatim:

> "Australian community pharmacy software is a fragmented market — FRED, Z Solutions, Minfos, LOTS, Aquarius, others. None has a clean modern API for this. Realistic posture:
> - **MVP:** structured cessation/change alerts to dispensing pharmacy via fax/email/portal; manual DAA timing entry by pharmacist
> - **V1:** API integration with the top 2-3 vendors (FRED is the largest); DAA packing schedule as state
> - **V2:** broader vendor coverage; full DAA composition tracking"

This means **MVP-stage Vaidshala uses fax/email/portal for cessation alerts**, not vendor APIs. Vendor partnerships are a V1+ acceleration.

## Target vendors (priority order per spec §3.4)

| Rank | Vendor | Market position | Engagement path |
|---|---|---|---|
| 1 | FRED Dispense | Largest community pharmacy market share | FRED IT partnership |
| 2 | Z Solutions (Z Dispense) | Major mid-market | Z Solutions partner program |
| 3 | Minfos | Established mid-market | Minfos partner program |
| 4 | LOTS | Smaller share | per-vendor |
| 5 | Aquarius / Corum | Smaller share | per-vendor |
| 6+ | Other (POS-Works, etc.) | Niche | per-vendor |

## What we expect to procure once partnership is agreed (per vendor)

1. API specification or integration protocol (often vendor-specific RPC, not standard FHIR)
2. DAA packing schedule data model
3. Dispensing event message format
4. Authentication / authorisation flow
5. Sandbox / test environment access
6. Rate limits + SLA

## Action required (commercial)

**Not blocked on engineering for MVP.** Recommended commercial sequence per spec §3.4:
1. **MVP**: Structured cessation/change alerts via fax/email/portal. No vendor partnership.
2. **V1**: Engage FRED IT first (largest market share). Timeline likely 6-12 months commercial discussions + integration.
3. **V2**: Z Solutions + Minfos.

## Status legend

- 🔒 engagement-required — vendor partnership / NDA required
