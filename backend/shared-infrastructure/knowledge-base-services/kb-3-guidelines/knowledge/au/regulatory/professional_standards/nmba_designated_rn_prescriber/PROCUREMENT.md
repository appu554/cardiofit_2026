# NMBA Registration Standard — Endorsement for Scheduled Medicines (Designated Registered Nurse Prescriber) — Procurement Runbook

**Status (2026-05-04):** ✅ landed (3 PDFs)
**Authority tier:** 2 (peak professional body — Nursing and Midwifery Board of Australia under AHPRA)
**Jurisdiction scope:** national (subject to state legislation parity)
**Effective period:** standard took effect 2025-09-30. First endorsed prescribers expected mid-2026.
**Reproduction terms:** © NMBA / AHPRA. Permitted use generally allows reference and quotation with attribution; commercial reproduction may require permission. Capture verbatim from source page.
**Layer 1 v2 spec section:** §4.5
**Spec deadline:** first cohort lands mid-2026 (~2-3 months from spec date)

## What to download

1. **NMBA Registration Standard: Endorsement for Scheduled Medicines — Designated Registered Nurse Prescriber**
   - Source: https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx
   - Why: definitional document for credential schema (eligibility, scope, partnership requirements, mentorship)
   - Maps to: Credential ledger schema (Phase 1C-δ); Authorisation state machine; spec's "new safety primitive almost nobody is building"

2. **NMBA Guidelines for Scheduled Medicines** (companion guidance)
   - Source: same domain
   - Why: operational interpretation of scope-of-practice for designated prescribers
   - Maps to: ScopeRules data; Credential ledger constraints

3. **ANMAC Accreditation Standards for the relevant postgraduate programs** (referenced by NMBA standard)
   - Source: https://www.anmac.org.au/
   - Why: defines "NMBA-approved postgraduate qualification" prerequisite tracked in Credential ledger
   - Maps to: Credential ledger qualification verification

## What enables (reminder for downstream phase)

Per Layer 1 v2 §4.5, the Authorisation state machine must track for each potential designated RN prescriber:
- Credential: Endorsement valid_from, valid_to, evidence_url
- PrescribingAgreement: linked to authorised health practitioner, scope (medicine classes, residents covered), validity period, mentorship_status (active/complete/breached), signed_packet_url
- MentorshipStatus: complete (post-six-month) vs in-progress
- ScopeMatch: per-action verification that the proposed action falls within the prescribing agreement's scope

These structures are Phase 1C-δ; this phase only procures the source.

## Code path (Playwright)

```
1. browser_navigate → https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx
2. Locate "Endorsement for scheduled medicines — designated registered nurse prescriber" link
3. Navigate to detail page; locate Standard PDF + companion guidance
4. fetch()+base64 → decode to:
   NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf
   NMBA-Scheduled-Medicines-Guidelines.pdf
5. browser_navigate → https://www.anmac.org.au/standards-and-review
6. Locate accreditation standards relevant to prescribing programs; fetch+decode
```

## Manual fallback

1. Browse https://www.nursingmidwiferyboard.gov.au/Registration-Standards.aspx.
2. Locate the designated RN prescriber endorsement standard; download PDF.
3. Save as `NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf`.
4. Download accompanying guidelines.
5. Browse https://www.anmac.org.au/; locate accreditation standards.
6. Save all to: `regulatory/professional_standards/nmba_designated_rn_prescriber/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 3 updated to ✅ landed
- [ ] Credential ledger schema authoring (Phase 1C-δ)
- [ ] Source Registry seed (Phase 1C-β)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| NMBA-Designated-RN-Prescriber-Standard-2025-09-30.pdf | 902b71327c668b6332fdd2594d8af39445669a6e152b4f30872e825b795a5e0b | 154775 |
| NMBA-Designated-RN-Prescriber-Guidelines-2025-09-30.pdf | e5032fd84cf73c74942e7933b72a4cb2b3bb487c5fc53b93fa8d82c9776830b7 | 442083 |
| NMBA-Designated-RN-Prescriber-FactSheet-2026-01.pdf | 74f0e0f1af026a7ea958b551f4797746d552ac3fd001aa15f15efb3f484333da | 341367 |

## Procurement notes (2026-05-04)

- NMBA standard, companion guidelines, and updated fact sheet procured via Playwright real-browser session through the AHPRA document distribution endpoint (`ahpra.gov.au/documents/default.aspx?record=...`). The endpoint is fronted by a TrustSec / Cloudflare anti-bot challenge that blocks `curl` (returns HTML interstitial); only browser-context download succeeds.
- Standard came into effect 2025-09-30. Fact sheet was updated 2026-01.
- ANMAC accreditation standards for prescribing programs (third entry in the original runbook) deferred — the relevant ANMAC standards are those for the Master-level postgraduate qualifications which the NMBA Standard cites by reference. Not required for Phase 1C-α; revisit during Phase 1C-δ Credential ledger schema authoring once a real candidate program is identified.
