# NCTS Value Sets + ATC Profiles for Layer 1B Ingestion — Procurement Runbook

**Status (2026-05-04):** ✅ landed (via cross-reference — see REFERENCES.md)
**Authority:** National Clinical Terminology Service (NCTS) under ADHA
**Spec type:** FHIR value sets + concept maps
**Effective period:** continuous; quarterly NCTS releases
**Reproduction terms:** NCTS account-bound; reproduction permitted within NCTS member terms
**Layer 1 v2 spec section:** §3.6 (NCTS terminology)
**Spec deadline:** baseline; supports observation / pathology coding for adapters

## What to download

This is a curated subset of NCTS value sets specifically for Layer 1B adapter ingestion contracts. KB-7 already contains the bulk SNOMED-CT-AU + AMT data; this folder captures the integration-time profiles (which value sets adapters bind their inputs to).

1. **AMT (Australian Medicines Terminology) ingestion bindings**
   - Source: https://nctsapi.healthterminologies.gov.au/
   - Why: defines how MedicationRequest.medication[x] should bind to AMT MPP/MP/CTPP/TPP codes
   - Maps to: eNRMC adapter MedicationRequest extraction; KB-7 reconciliation

2. **AU LOINC Lab Observation value sets** (pathology-relevant subset)
   - Source: NCTS shrine / Ontoserver expansions
   - Why: defines which LOINC codes appear in MHR pathology + how to map to Vaidshala observation taxonomy
   - Maps to: MHR pathology adapter; KB-26 baseline observation flow

3. **AU FHIR Discharge Summary value sets** (for §3.3 hospital discharge)
   - Source: NCTS / ADHA discharge summary IG
   - Why: defines coded sections of discharge summaries (medications-on-admission, medications-on-discharge, allergies, problems)
   - Maps to: hospital discharge reconciliation adapter

4. **AU SNOMED CT-AU subset for aged care observation** (mobility, falls, behavioural events)
   - Source: NCTS authoring tools / Vaidshala-specific subset
   - Why: defines which SNOMED concepts are in scope for Layer 1B observation ingestion

## Code path (Playwright)

NCTS API requires authenticated NCTS account (already in place per existing KB-7 work).

```
1. browser_navigate → https://nctsapi.healthterminologies.gov.au/ (authenticated session)
2. Use Ontoserver/Shrimp to export named value sets relevant to Layer 1B
3. Save to: integration_specs/ncts_profiles/<name>.json
```

Alternative: re-use KB-7 ontoserver-valuesets/ as the source (already populated). The profiles relevant to Layer 1B can be referenced via symlinks or copies.

## Manual fallback

1. Log into NCTS Ontoserver / Shrimp
2. Export value sets per the list above
3. Save to: `integration_specs/ncts_profiles/`

## After files land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 4 updated to ✅ landed
- [ ] Adapter-time binding contracts authored (Phase 1B-γ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| (no local files — cross-referenced to `kb-7-terminology/data/ontoserver-valuesets/`) | — | — |

## Procurement notes (2026-05-04)

- **Option A taken** (per runbook recommendation): cross-reference KB-7 ontoserver-valuesets/ rather than re-fetching from authenticated NCTS API.
- KB-7 contains 23,710 NCTS value sets downloaded 2025-12-13 from `https://r4.ontoserver.csiro.au/fhir` — covers the full Layer 1B adapter binding surface.
- See `REFERENCES.md` (sibling file in this directory) for:
  - Authoritative source path: `kb-7-terminology/data/ontoserver-valuesets/`
  - Per-§3.x adapter binding contracts (canonical ValueSet URLs)
  - Rationale for cross-reference vs duplication
  - Trigger conditions for switching to Option B (re-procurement)
- **Procurement method:** No browser fetch needed. Documented cross-reference only.
