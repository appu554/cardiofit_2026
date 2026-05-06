# NCTS / ATC Profiles — Cross-Reference to KB-7

**Status (2026-05-04):** ✅ landed via cross-reference (Option A per PROCUREMENT.md)
**Approach:** Re-use existing KB-7 ontoserver downloads rather than re-fetching from NCTS API (which requires authenticated NCTS account session).

## Authoritative source location

```
backend/shared-infrastructure/knowledge-base-services/kb-7-terminology/data/ontoserver-valuesets/
├── definitions/         # raw FHIR ValueSet resources (GUID-named .json)
├── expansions/          # pre-expanded ValueSet results
├── _summary.json        # 23,710 value sets total (downloaded 2025-12-13)
├── _progress.json       # download progress markers
└── download.log
```

Generated from `https://r4.ontoserver.csiro.au/fhir` on 2025-12-13. Coverage:
- 23,710 value sets total available
- 23,710 downloaded (100%)
- 22,007 expanded (92.8%)
- 0 failures

## Layer 1B-α adapter binding contracts

The NCTS value sets that Layer 1B adapters bind their inputs to are **not separately materialised here** — the adapter-time binding contract is implemented as a *named subset* against the KB-7 store. Rather than duplicating storage, adapters resolve value-set references at runtime via the KB-7 terminology service.

Layer 1B adapters reference these NCTS value sets (canonical URLs; resolution via KB-7):

| Spec section | Adapter | NCTS / FHIR ValueSet canonical URL | Use |
|---|---|---|---|
| §3.1 eNRMC | MedicationRequest substrate | `https://healthterminologies.gov.au/fhir/ValueSet/amt-medicinal-product-pack-1` | AMT MPP for unit-of-prescribing |
| §3.1 eNRMC | MedicationRequest substrate | `https://healthterminologies.gov.au/fhir/ValueSet/amt-trade-product-pack-1` | AMT TPP brand-level |
| §3.2 MHR pathology | DiagnosticReport / Observation | `http://hl7.org/fhir/ValueSet/observation-codes` (LOINC subset, Australian usage) | Pathology coding |
| §3.2 MHR pathology | Observation | `https://healthterminologies.gov.au/fhir/ValueSet/loinc-pathology-codes-1` | AU LOINC pathology subset |
| §3.3 Hospital discharge | Composition.section coding | `https://healthterminologies.gov.au/fhir/ValueSet/discharge-summary-section-codes-1` | Discharge summary section codes |
| §3.3 Hospital discharge | AllergyIntolerance.code | `https://healthterminologies.gov.au/fhir/ValueSet/indicator-hypersensitivity-intolerance-to-substance-1` | AU allergy/intolerance |
| §3.3 Hospital discharge | Condition.code | `https://healthterminologies.gov.au/fhir/ValueSet/reason-for-encounter-1` | Reason for encounter / problem list |
| §3.5 Care management | Observation (mobility, falls, behaviour) | SNOMED CT-AU subset (aged care vital signs + ADL + behavioural) | Resident observation taxonomy |
| §3.6 NCTS terminology | All adapters | KB-7 `/v1/valuesets/expand` API | Runtime binding resolution |

## Why cross-reference (not copy)

1. **DRY** — KB-7 already provides terminology service infrastructure; duplicating files into `integration_specs/` would violate single-source-of-truth.
2. **Authenticated access** — NCTS API requires authenticated NCTS account; KB-7 already manages that credential and quarterly refresh cycle.
3. **Refresh cadence** — KB-7 is the team's terminology refresh substrate; pinning duplicated copies in `integration_specs/` would silently drift from the canonical source.
4. **Volume** — 23,710 value sets totalling many GB; in-tree duplication is wasteful when adapter binding only needs ~20 specific value sets at runtime.

## When to reconsider (Option B trigger)

Re-procure into this folder ONLY if:
- KB-7 terminology service goes offline for any 1B adapter unit-test environment
- Adapter contract requires a *frozen snapshot* of a specific NCTS value-set version (regulatory pinning)
- KB-7 changes ontoserver source (e.g. moves off CSIRO ontoserver) and value-set GUIDs change

## Phase 1B-γ adapter contract (forthcoming)

When Phase 1B-γ implements the eNRMC adapter, the adapter contract MUST:
1. Declare each value-set canonical URL it binds to (table above)
2. Resolve at runtime via KB-7 `/v1/valuesets/$expand` operation
3. Cache resolved expansions per (canonical URL + version + region=AU)
4. Validate inbound coded values against the resolved expansion membership
5. Surface unmappable codes via `EvidenceTrace` for clinical review (per Vaidshala MVP-1)

## File hashes (cross-reference; no local files)

This folder intentionally contains no `.json` / `.tgz` / `.pdf` artifacts. The procurement contract is satisfied by the cross-reference above. See `kb-7-terminology/data/ontoserver-valuesets/_summary.json` for the canonical hash manifest of the underlying value-set corpus.
