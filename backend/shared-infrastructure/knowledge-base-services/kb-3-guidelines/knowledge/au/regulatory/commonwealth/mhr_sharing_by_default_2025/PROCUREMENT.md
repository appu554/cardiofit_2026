# Modernising My Health Record (Sharing by Default) Act 2025 — Procurement Runbook

**Status (2026-05-04):** ✅ landed (2 PDFs)
**Authority tier:** 1 (Commonwealth legislature)
**Jurisdiction scope:** national
**Effective period:** Royal Assent 2025-02-14. Mandatory pathology + diagnostic imaging upload to MHR commences 2026-07-01. Civil penalties: 250 penalty units (~AUD 82,500) non-registration; 30 penalty units (~AUD 9,900) non-compliant upload.
**Reproduction terms:** Crown copyright Commonwealth. CC BY 4.0.
**Layer 1 v2 spec section:** §4.10
**Spec deadline:** mandatory pathology upload begins 2026-07-01 (~8 weeks from spec date)

## What to download

1. **Modernising My Health Record (Sharing by Default) Act 2025**
   - Source: https://www.legislation.gov.au/
   - Why: regulatory backbone behind pathology integration simplification per spec §3.2. Removes the per-pathology-vendor integration burden by mandating MHR upload by default.
   - Maps to: Layer 1B MHR FHIR Gateway integration (deferred); Source Registry; Authorisation state machine (consent-aware MHR access)

2. **Sharing by Default Rules** (subordinate instrument, expected iterative release)
   - Source: https://www.legislation.gov.au/
   - Why: operational definitions of which information types fall under "default sharing" and how consent overrides apply
   - Maps to: ingestion contracts for MHR FHIR Gateway

3. **Department of Health implementation guidance**
   - Source: https://www.health.gov.au/ — "Better and Faster Access to health information"
   - Why: spec §4.10 directs quarterly monitoring of this page for Sharing by Default Rule extensions; each extension is potentially a new Layer 1 source

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.gov.au/
2. Search "Modernising My Health Record" → locate Act PDF
3. fetch()+base64 → decode to: MHR-Sharing-by-Default-Act-2025.pdf
4. Search for associated subordinate Rules → fetch+decode if available
5. browser_navigate → Department of Health "Better and Faster Access" landing page
6. Download published implementation guidance PDFs
```

## Manual fallback

1. Browse https://www.legislation.gov.au/; search "Modernising My Health Record".
2. Download Act + any Rules PDFs.
3. Browse https://www.health.gov.au/; locate Better and Faster Access pages; download guidance PDFs.
4. Save to: `regulatory/commonwealth/mhr_sharing_by_default_2025/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 8 updated to ✅ landed
- [ ] Source Registry seed (Phase 1C-β)
- [ ] Layer 1B MHR FHIR Gateway integration (separate workstream)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| MHR-Sharing-by-Default-Act-2025-asmade.pdf | a10080cf0eec4320c3b6b6f61b7a70c3feea4aede7103c94cd5af7b4465cc862 | 381654 |
| MHR-Share-by-Default-Rules-2025.pdf | fbe06d936f4e4fb7e3b5b7e257637f3a421fa63c9b528fbf2264704b19db650e | 246144 |

## Procurement notes (2026-05-04)

- The Act was registered as `C2025A00008` under the title "Health Legislation Amendment (Modernising My Health Record—Sharing by Default) Act 2025"; spec referenced informally as "Modernising MHR (Sharing by Default) Act".
- Subordinate "My Health Record (Share by Default) Rules 2025" registered as `F2025L01569` on 2025-12-09.
- Act Explanatory Memorandum (`/es/original/pdf` URL) returned 404 — only the as-made text PDF is published via the Federal Register API for this Act. EM is in Hansard / parlinfo. Not load-bearing for current spec section §4.10.
- DH "Better and Faster Access" implementation guidance landing page yielded no separate PDFs at procurement time — the page is a navigation hub. Quarterly monitoring (per spec §4.10) will pick up any future guidance Rules.
