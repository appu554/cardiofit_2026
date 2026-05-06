# AU FHIR Discharge Summary Profile — Procurement Runbook

**Status (2026-05-04):** ✅ landed
**Authority:** ADHA / HL7 Australia
**Spec type:** FHIR R4 Composition profile + Discharge Summary IG
**Effective period:** continuous; subject to ADHA / HL7 AU revisions
**Reproduction terms:** ADHA-published material is Crown copyright Commonwealth; HL7 AU material per HL7 AU licensing.
**Layer 1 v2 spec section:** §3.3 (hospital discharge reconciliation — "highest-yield channel v1.0 missed")
**Spec deadline:** Wave 2 priority. Spec estimates 6-8 weeks of focused engineering for the discharge reconciliation MVP — this phase only procures the spec.

## What to download

1. **AU FHIR Discharge Summary Implementation Guide**
   - Source: https://developer.digitalhealth.gov.au/specifications/clinical-documents/discharge-summary
   - Why: structured discharge summary profile — Composition + sections (medications-on-admission, medications-on-discharge, allergies, problems, follow-up plan)
   - Maps to: hospital discharge reconciliation adapter; ACOP routing within 24 hours per spec §3.3

2. **CDA Discharge Summary spec** (legacy MHR upload format)
   - Source: ADHA developer portal
   - Why: most current MHR-uploaded discharge summaries are still CDA-format until FHIR transition completes; adapter must handle both
   - Maps to: PDF / CDA fallback path in discharge reconciliation adapter

3. **Medication reconciliation profile** (if separately published)
   - Source: HL7 AU
   - Why: structured reconciliation between three coding systems — discharge summary (generic + brand), eNRMC (AMT codes), GP system (MIMS or other)
   - Maps to: spec §3.3 reconciliation challenge

## Code path (Playwright)

```
1. browser_navigate → https://developer.digitalhealth.gov.au/specifications/clinical-documents/discharge-summary
2. browser_snapshot → locate IG + CDA profile downloads
3. browser_evaluate → fetch()+base64 → decode locally
```

## Manual fallback

1. Browse ADHA developer portal → Specifications → Clinical Documents → Discharge Summary
2. Download IG + supporting profile docs + CDA spec
3. Save to: `integration_specs/hospital_transitions/au_fhir_discharge_summary/`

## After files land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 5 updated to ✅ landed
- [ ] Hospital discharge reconciliation adapter design (Phase 1B-δ — 6-8 weeks)
- [ ] ACSQHC Stewardship Framework alignment (already in Layer 1A wave6/acsqhc_ams/)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| EP_4226_2025_DischargeSummary_v1.7.zip | ce10b7a833491a42393f8b7656ef0d05c1d5277494f587fa4f0325f1abfb2c51 | 17251438 |
| EP_3851_2023_AgedCareTransferSummary_v1.1.zip | 7a75781297d22a244534ffbc2f1fb91e5ee5a427292ced508508781fcf79dfff | 1316952 |

## Procurement notes (2026-05-04)

- **AU FHIR Discharge Summary IG not separately published.** ADHA's current Discharge Summary spec is **v1.7 (CDA-format)** — the legacy MHR upload format. The "modern FHIR R4 path" referenced in the runbook is *not* yet a distinct ADHA-published IG; FHIR-based discharge document profiles are expressed via:
  - `Composition`, `Encounter`, `MedicationStatement`, `AllergyIntolerance` profiles in **HL7 AU Base IG v6.0.0** (`hl7_au/base_ig_r4/`, Source 3) — including `Encounter-discharge-1` reference profile from AU Core 0.2.0-preview
  - ADHA-native discharge content is still primarily CDA at v1.7
- **Aged Care Transfer Summary v1.1** captured as companion artifact: defines content/format for resident transfer between RACF ↔ hospital — directly relevant to spec §3.3 reconciliation (medications-on-admission, medications-on-discharge, allergies, problems).
- **Source URLs:**
  - https://implementer.digitalhealth.gov.au/resources/discharge-summary-v1-7
  - https://implementer.digitalhealth.gov.au/resources/aged-care-transfer-summary-v1-1
- **Adapter implication:** the discharge reconciliation adapter MUST handle BOTH paths (CDA v1.7 today, FHIR R4 via AU Base profiles for emerging integrations). Three-coding-system reconciliation challenge (discharge generic+brand → eNRMC AMT → GP MIMS) remains the §3.3 engineering risk; no separate "Medication reconciliation profile" IG was found in HL7 AU.
- **Procurement method:** Playwright fetch+base64; ZIPs verified by `file(1)`.
