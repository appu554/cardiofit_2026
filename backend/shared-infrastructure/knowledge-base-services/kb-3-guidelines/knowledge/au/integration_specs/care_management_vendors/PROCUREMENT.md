# Care Management Software Vendor APIs — Procurement Runbook

**Status (2026-05-04):** 🔒 engagement-required
**Authority:** Per-vendor (commercial)
**Spec type:** Proprietary vendor API documentation
**Reproduction terms:** TBD per vendor — typically NDA-bound
**Layer 1 v2 spec section:** §3.5 (nursing observations and behavioural notes)
**Spec deadline:** spec describes this as "may be the single hardest data engineering problem in Layer 1+2"

## Why engagement-required

Per Layer 1 v2 §3.5, the Australian aged care care-management vendor market is fragmented; integration is per-vendor and effortful. Per spec verbatim:

> "Sequencing:
> - **MVP:** CSV export of structured observations from one or two pilot facilities' systems
> - **V1:** API integration with the top 2-3 care management vendors
> - **V2:** broader vendor coverage; NLP on free-text progress notes"

CSV export at MVP stage requires **facility partnership**, not vendor partnership — the facility administrator extracts CSV from whichever care-management system they happen to use.

## Target vendors (priority order per spec §3.5)

| Rank | Vendor | Market position | Engagement path |
|---|---|---|---|
| 1 | Leecare | Established large-share | Leecare partnership |
| 2 | AutumnCare | Major mid-market | AutumnCare partner program |
| 3 | Person Centred Software (PCS) | Growing share | PCS API partnership |
| 4 | Mirus Australia | Mid-market | Mirus partner program |
| 5+ | Other (Manad Plus, ECase, etc.) | Niche | per-vendor |

## Data sources within each vendor's system

Per spec §3.5 the structured data Vaidshala needs for KB-26 baselines + Clinical state machine:

1. **eMAR** (electronic Medication Administration Record) — administration events, refusals, PRN use, missed doses
2. **Structured observations** — vital signs, weight, mobility scores, behavioural events, falls, infections
3. **Behavioural charts** for residents on antipsychotics — required under restrictive practice regulations
4. **Free-text progress notes** — clinical narrative; harder to use; deferred to Wave 4-5 (NLP)

## What we expect to procure once partnership is agreed (per vendor)

1. API specification (typically REST, sometimes SOAP for older systems)
2. Data dictionary for observation types + behavioural chart structures
3. eMAR event schema
4. Authentication flow
5. Sandbox access

## Action required (commercial + facility)

**Not blocked on engineering for MVP.** Recommended sequence per spec §3.5:
1. **MVP**: Identify 1-2 pilot facilities. Have facility admin export structured observations to CSV daily. Vaidshala adapter parses CSV. **No vendor partnership required at MVP.**
2. **V1**: Engage Leecare (largest market share) for API partnership. Timeline 6-12 months commercial.
3. **V2**: AutumnCare + PCS.

**Critical clinical point** (spec verbatim §3.5):
> "The running baseline computation that powers the Clinical state machine depends on having structured observations over time. CSV monthly snapshots are insufficient — the platform needs at minimum daily observation flows for vital signs, behavioural events, mobility, weight."

This shapes facility partnership requirements: at MVP stage we need **daily** CSV export, not monthly. That's a facility-process commitment, not just a data export.

## Status legend

- 🔒 engagement-required — vendor partnership / NDA required
