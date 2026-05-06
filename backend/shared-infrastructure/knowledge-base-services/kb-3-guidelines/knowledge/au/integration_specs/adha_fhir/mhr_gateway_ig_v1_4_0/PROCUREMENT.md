# ADHA MHR FHIR Gateway Implementation Guide v1.4.0 (R4) — Procurement Runbook

**Status (2026-05-04):** ✅ landed
**Authority:** Australian Digital Health Agency (ADHA)
**Spec type:** FHIR R4 Implementation Guide
**Effective period:** v1.4.0 published; SOAP/CDA path is current production interface, FHIR Gateway is modern transition path per spec §3.2.
**Reproduction terms:** ADHA-published material is generally Crown copyright Commonwealth; check the IG's licence section. Reference and quotation permitted with attribution.
**Layer 1 v2 spec section:** §3.2 (MHR pathology integration)
**Spec deadline:** mandatory pathology upload to MHR commences 2026-07-01 (~8 weeks from spec date)

## What to download

1. **MHR FHIR Gateway Implementation Guide v1.4.0** (R4)
   - Source: https://developer.digitalhealth.gov.au/specifications/api-library/myhealthrecord
   - Why: defines FHIR R4 capability statements, profile constraints, and operation contracts for the Sharing-by-Default pathology + diagnostic imaging upload pathway
   - Maps to: Layer 1B MHR FHIR Gateway adapter (downstream phase)

2. **ADHA FHIR Implementation Guide bundle** (`package.tgz` if published)
   - Source: same domain or HL7 AU Confluence
   - Why: machine-readable IG bundle for `npm install`-style consumption by FHIR validators

3. **MHR Conformance Profile** for DiagnosticReport, Observation, MedicationRequest, AllergyIntolerance
   - Source: ADHA developer portal
   - Why: the constraints we must respect when querying MHR for resident clinical state

## Code path (Playwright)

```
1. browser_navigate → https://developer.digitalhealth.gov.au/specifications/api-library/myhealthrecord
2. browser_snapshot → identify "FHIR Gateway Implementation Guide v1.4.0" or latest version link
3. browser_navigate → IG landing page
4. browser_snapshot → locate downloadable PDF + package.tgz / .zip
5. browser_evaluate → fetch()+base64 → decode locally
```

## Manual fallback

1. Browse https://developer.digitalhealth.gov.au/specifications/api-library/myhealthrecord
2. Locate FHIR Gateway IG (latest); download PDF + package bundle
3. Save to: `integration_specs/adha_fhir/mhr_gateway_ig_v1_4_0/`

## After files land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 1 updated to ✅ landed
- [ ] Layer 1B MHR FHIR Gateway adapter (deferred to 1B-δ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| EP_4232_2026_MyHealthRecordFHIRGateway_v5.0.zip | bb15fc8e4802164c2ced497120bdd891bfc2a49c8abb478200fe5442c09a9f13 | 2114433 |

## Procurement notes (2026-05-04)

- **Version mismatch:** Folder named `mhr_gateway_ig_v1_4_0` but ADHA Implementer Hub published version is **v5.0** (effective Mar 2026). v1.4.0 was superseded long ago — current spec captured.
- **Package contents:** ZIP bundles four artifacts (extracted on demand):
  - `DH_4234_2026_MyHealthRecordFHIRGateway_APISpecification_v5.0.pdf` (1.58 MB) — primary FHIR R4 capability statements + profile constraints
  - `DH_4233_2026_MyHealthRecordFHIRGateway_ReleaseNote_v5.0.pdf` (444 KB)
  - `DH_4020_2024_MyHealthRecordFHIRGateway_DataMapping_v3.0.xlsx` (186 KB)
  - `DH_4021_2024_MyHealthRecordFHIRGateway_ErrorMapping_v3.0.xlsx` (104 KB)
- **No separate `package.tgz`:** ADHA does not publish an `npm`-style FHIR package for the MHR FHIR Gateway; capability statement / profiles are inline in the PDF API Specification. For machine-readable AU FHIR profiles, see `hl7_au/base_ig_r4/` (Source 3).
- **Source URL:** https://implementer.digitalhealth.gov.au/resources/my-health-record-fhir-gateway-v5-0
- **Procurement method:** Playwright fetch+base64 (cookies-included), zip verified by `file(1)` and `unzip -l`.
