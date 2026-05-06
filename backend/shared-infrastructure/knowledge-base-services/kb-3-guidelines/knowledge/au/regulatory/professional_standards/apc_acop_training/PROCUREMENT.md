# APC Accreditation Standards for ACOP (Aged Care On-site Pharmacist) Training Programs — Procurement Runbook

**Status (2026-05-04):** ✅ landed (6 PDFs)
**Authority tier:** 2 (peak professional body — Australian Pharmacy Council)
**Jurisdiction scope:** national
**Effective period:** APC-accredited ACOP training mandatory for ACOP measure participation from 2026-07-01.
**Reproduction terms:** © APC. Reference and quotation permitted with attribution. Commercial reproduction may require permission. Verify on download.
**Layer 1 v2 spec section:** §4.7
**Spec deadline:** mandatory training requirement begins 2026-07-01

## What to download

1. **APC Accreditation Standards for ACOP Training Programs**
   - Source: https://www.pharmacycouncil.org.au/
   - Why: defines required training content + duration + assessment for ACOP credentialing
   - Maps to: Credential ledger (APC training completion: valid_from, valid_to, evidence) per spec §4.7

2. **APC accredited program list** (registry of approved providers)
   - Source: https://www.pharmacycouncil.org.au/
   - Why: operational lookup for credential verification
   - Maps to: Credential ledger evidence_url validation

3. **$350M ACOP Program operational rules** (PSA / Department of Health joint material)
   - Source: https://www.psa.org.au/ + https://www.health.gov.au/
   - Why: defines Tier 1 (community pharmacy) vs Tier 2 (facility-employed) ACOP claim scope; bed-allocation rules; daily rate (verified AUD 619.84/day per FTE Feb 2026)
   - Maps to: Credential ledger ACOP measure participation; KB-13 PHARMA-Care indicators

## Code path (Playwright)

```
1. browser_navigate → https://www.pharmacycouncil.org.au/
2. Locate ACOP training accreditation standards page
3. Download accreditation standard PDF; fetch()+base64 → decode to:
   APC-ACOP-Training-Accreditation-Standard.pdf
4. Locate accredited program list / register; fetch where available
5. browser_navigate → https://www.psa.org.au/ ACOP program pages
6. Download ACOP operational rules / measure guidelines
7. browser_navigate → https://www.health.gov.au/our-work/aged-care-on-site-pharmacist
8. Download Department of Health ACOP measure documentation
```

## Manual fallback

1. Browse https://www.pharmacycouncil.org.au/; locate ACOP accreditation page.
2. Download standard PDF; save as `APC-ACOP-Training-Accreditation-Standard.pdf`.
3. Browse https://www.psa.org.au/; locate ACOP program documentation.
4. Browse https://www.health.gov.au/; locate ACOP measure pages.
5. Save all to: `regulatory/professional_standards/apc_acop_training/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 5 updated to ✅ landed
- [ ] Credential ledger ACOP fields (Phase 1C-δ)
- [ ] Source Registry seed (Phase 1C-β)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| APC-ACOP-Accreditation-Standards.pdf | db41019dcc36a065cfddb4ed5693828d5a5ad8fe86162a7d0019b1c2cdc340b7 | 1255749 |
| APC-ACOP-Performance-Outcomes-Framework.pdf | 93f72de84e61079d46780d3bec574b0b6ba0f81c35498e7417fbc7955929ea87 | 1231866 |
| APC-ACOP-Indicative-Role-Description.pdf | afc7c755b64f58726a8bbd122f95648f3c33f0cc696560d6e62a0bf3e66dfae0 | 205351 |
| APC-ACOP-Evidence-Guide-2023.pdf | 224764a979eb4e6bf5260ac27319126827b16a97669498e830066d441c939583 | 1293594 |
| DH-ACOP-Pharmacist-and-RACH-Guide.pdf | 4826793617d398473da8d0218d5563a7748d9d5fe64500d16dceccc6fa82ebab | 650147 |
| DH-ACOP-PHN-Grant-Program-Guide.pdf | a48717a2d8fde3baf54873350494b043088d493af31ca41ea3ac6be7ef29cec0 | 849212 |

## Procurement notes (2026-05-04)

- APC Accreditation Standards, Performance Outcomes Framework, Indicative Role Description, and Evidence Guide procured directly from `pharmacycouncil.org.au/resources/pharmacist-education-programs-standards/`. These are the joint MMR + Aged Care On-site Pharmacist (ACOP) education program standards.
- Department of Health ACOP measure operational documentation procured from `health.gov.au` (Pharmacist + RACH guide and PHN Grant Program guide). These define Tier 1/Tier 2 claim scope and bed-allocation rules referenced by spec §4.7.
- APC accredited program list (live registry at `pharmacycouncil.org.au/education-provider/accreditation/pharmacist-education-programs/accredited-pharmacist-education-programs/`) intentionally not snapshotted as a PDF — it is an HTML registry that must be queried at credential-verification time, not cached. Phase 1C-δ Credential ledger should call it directly.
- PSA-published ACOP program operational material referenced in the original runbook — PSA's site is fragmented; the load-bearing operational rules are already captured in the DH guides above. Re-procure if PSA publishes a consolidated operational document later.
