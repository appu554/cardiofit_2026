# ADHA eNRMC Conformance Specifications — Procurement Runbook

**Status (2026-05-04):** ✅ landed
**Authority:** Australian Digital Health Agency (ADHA)
**Spec type:** Vendor conformance profile
**Effective period:** ADHA conformance certification ongoing; per Nov 2025 status, 8 of 10 eNRMC vendors are conformant for electronic prescribing. Two remaining vendors extended to 2026-04-01. RACF adoption deadline: 2026-12-31.
**Reproduction terms:** Crown copyright Commonwealth via ADHA. Reference and quotation permitted with attribution.
**Layer 1 v2 spec section:** §3.1 (eNRMC integration)
**Spec deadline:** RACF eNRMC adoption mandatory by 2026-12-31

## What to download

1. **eNRMC Conformance Profile** (FHIR R4 + HL7 v2 messaging)
   - Source: https://developer.digitalhealth.gov.au/specifications/api-library/electronic-prescribing
   - Why: defines what "conformant eNRMC" means at the protocol level — message structures, MedicationRequest profiles, MedicationAdministration events
   - Maps to: Layer 1B eNRMC adapter (1B-γ priority — recommended first adapter end-to-end)

2. **Conformant Vendor Register / Status Tracker**
   - Source: https://www.digitalhealth.gov.au/healthcare-providers/initiatives-and-programs/electronic-prescribing
   - Why: live register of which vendors hold conformance; informs partnership prioritisation
   - Maps to: enrmc_vendors/PROCUREMENT.md vendor selection

3. **eNRMC Implementation Guide** (operational guidance for integrators)
   - Source: same domain
   - Why: integration patterns, common pitfalls, certification pathway

## Code path (Playwright)

```
1. browser_navigate → https://developer.digitalhealth.gov.au/specifications/api-library/electronic-prescribing
2. browser_snapshot → locate eNRMC / electronic prescribing conformance documents
3. browser_evaluate → fetch()+base64 → decode locally
```

## Manual fallback

1. Browse https://developer.digitalhealth.gov.au/
2. Search "eNRMC conformance" or "electronic prescribing conformance"
3. Download all linked PDFs + IG bundles
4. Save to: `integration_specs/adha_fhir/enrmc_conformance/`

## After files land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 2 updated to ✅ landed
- [ ] Conformant-vendor list extracted and surfaced in `enrmc_vendors/PROCUREMENT.md`
- [ ] Layer 1B eNRMC adapter design (Phase 1B-γ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| EP_4153_2025_ElectronicPrescribing-TechnicalFrameworkDocuments_v3.7.zip | 437261c09097052b89ce7d504bb1322f439753847f02e99ba7d898f0598460d8 | 4891676 |
| EP_4150_2025_ElectronicPrescribing-ConformanceTestSpecifications_v3.0.6.zip | f8f074e45bc9141013dec1de1ac6dfb7ba219955a89dc69040ce56560de00f91 | 2590290 |
| ep-conformance-register-20260408.pdf | e1ea5cbde71638de14727e9736209e3eb21a5ba58f7d4e855a44d990fc889607 | 1332654 |
| transitional-enrmc-conformance-register-20260323.pdf | f8ce4ced1ff9744a605f1446b0e008b94f2a1b5fdc152d17c12ecc1edf4b2b3b | 127844 |

## Procurement notes (2026-05-04)

- **No standalone "eNRMC Conformance Specifications" published.** Per ADHA Implementer Hub, eNRMC conformance is governed by the umbrella **Electronic Prescribing Technical Framework v3.7** + **Conformance Test Specifications v3.0.6**. The eNRMC vendor cohort is tracked separately in the **Transitional eNRMC Conformance Register**.
- **Source URLs:**
  - https://implementer.digitalhealth.gov.au/resources/electronic-prescribing-technical-framework-documents-v3-7
  - https://implementer.digitalhealth.gov.au/resources/electronic-prescribing-conformance-test-specifications-v3-0-6
  - https://www.digitalhealth.gov.au/about-us/policies-privacy-and-reporting/registers (PDF registers)
- **Conformant eNRMC vendors (Transitional eNRMC Register, 23-Mar-2026 generation, 10 products, all valid until 31-Dec-2026):**

| Reg # | Product | Provider | Version |
|---|---|---|---|
| 00001 | emma | Compact Business Systems Australia | 1.4 |
| 00002 | BESTMED Medication Management | Best Health Solutions Pty Ltd | 2.2 |
| 00003 | Webstercare MedCare | Webstercare - Manrex | 2.0 |
| 00004 | MedPoint | Telstra Health Pty Ltd | 1.1 |
| 00005 | HealthStream | Medication Packaging Systems (MPS) | 2022.6 |
| 00006 | Acredia Care (Rx) | Acredia Pty Ltd | 1.0 |
| 00007 | Leecare P6Med | LeeCare Solutions | 22.05 |
| 00008 | Medi-Map Medication Management | Medi-Map Group Pty Limited | 17.8 |
| 00009 | StrongCare | SRSPV Pty LTD | 1.0 |
| 00010 | SimpleMed Electronic Medication Management System | SimpleMed | 1.0.1.0 |

  All entries are categorised as **"Medication Charts (MC)"** product type — i.e. transitional eNRMC. Note: the broader ADHA EP Conformance Register (`ep-conformance-register-20260408.pdf`) covers prescribing/dispensing software conformance and is the authoritative artifact for downstream pharmacy & GP integration prioritisation.
- **Aged-care priority short-list for 1B-γ adapter:** Telstra Health MedPoint (largest RACF footprint), LeeCare P6Med (already a Vaidshala care-management vendor candidate per source #8), Webstercare MedCare (DAA workflow integration). To be confirmed via engagement workstream.
- **Procurement method:** Playwright fetch+base64; ZIPs verified by `file(1)`; PDFs verified by `pdftotext`.
