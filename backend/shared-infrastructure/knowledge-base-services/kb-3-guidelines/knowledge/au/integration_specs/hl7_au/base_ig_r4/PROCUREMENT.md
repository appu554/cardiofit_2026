# HL7 AU Base Implementation Guide (FHIR R4) — Procurement Runbook

**Status (2026-05-04):** ✅ landed
**Authority:** HL7 Australia
**Spec type:** FHIR R4 Implementation Guide (jurisdictional base)
**Effective period:** continuous; check for latest published version
**Reproduction terms:** © HL7 Australia. Generally permissive for implementer use; verify on download.
**Layer 1 v2 spec section:** §3.1 (eNRMC), §3.2 (MHR), §3.3 (discharge)
**Spec deadline:** baseline FHIR profile work feeds all Layer 1B adapter design

## What to download

1. **HL7 AU Base Implementation Guide** (FHIR R4)
   - Source: https://confluence.hl7australia.com/ or https://hl7.org.au/
   - Why: Australian-localised FHIR R4 base profiles — Patient, Practitioner, Organization, Encounter, Observation with AU extensions (IHI, HPI-I, HPI-O, AMT bindings)
   - Maps to: every Layer 1B adapter (canonical AU FHIR shape)

2. **AU FHIR core package bundle** (`hl7.fhir.au.base.tgz`)
   - Source: https://hl7.org.au/fhir/ or HL7 AU Confluence package server
   - Why: machine-readable IG for FHIR validators

3. **Australian Core Implementation Guide** (if separately published — captures common Australian extension patterns)
   - Source: HL7 AU
   - Why: extension definitions for IHI, HPI, MBS/PBS codes used throughout AU FHIR

## Code path (Playwright)

```
1. browser_navigate → https://hl7.org.au/fhir/ (or confluence.hl7australia.com)
2. browser_snapshot → locate AU Base IG download links + package bundle
3. browser_evaluate → fetch()+base64 → decode locally
```

## Manual fallback

1. Browse https://hl7.org.au/fhir/
2. Locate AU Base IG; download published version PDF/HTML + package bundle
3. Save to: `integration_specs/hl7_au/base_ig_r4/`

## After files land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 3 updated to ✅ landed
- [ ] Substrate entity design (Phase 1B-β) references AU FHIR profiles for Resident → AU Patient mapping

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| hl7.fhir.au.base-v6.0.0-full-ig.zip | 279fabd318e7223a750451196205233fb51230a5ddc6113c2421aae1a50fb127 | 40932704 |
| hl7.fhir.au.base-v6.0.0-package.tgz | ac2ebba3e9363b3387926f75542e57e378aecdd2b79fe955a0974e3a2f10a163 | 830695 |

## Procurement notes (2026-05-04)

- **Published version captured: v6.0.0** (current at procurement). Spec mentions "R4" generically; the v6.0.0 IG is the latest published HL7 AU Base IG.
- **Two artifacts captured:**
  - `full-ig.zip` (39 MB) — complete static HTML IG + examples + StructureDefinitions + ValueSets
  - `package.tgz` (811 KB) — `npm`-style FHIR package consumable by FHIR validators (`hl7.fhir.au.base#6.0.0`)
- **R4B variant** (`package.r4b.tgz`) not pulled — Vaidshala adapters target FHIR R4 baseline; R4B can be pulled later if needed.
- **No separate "Australian Core IG"** (the previously distinct "AU Core" project has been merged into AU Base 6.0.0; AU Core 0.2.0-preview remains for FHIR R4 reference profiles such as `Encounter-discharge-1`).
- **Source URLs:** https://hl7.org.au/fhir/ + https://hl7.org.au/fhir/downloads.html
- **Procurement method:** Playwright fetch+base64; archives verified by `file(1)`.
