# Restrictive Practice Regulations — Procurement Runbook

**Status (2026-05-04):** ✅ landed (9 PDFs)
**Authority tier:** 1 (Commonwealth legislature + ACQSC)
**Jurisdiction scope:** national
**Effective period:** core legislation in force since 2019; amended by Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022; subsequent regulations iterating.
**Reproduction terms:** Crown copyright Commonwealth. CC BY 4.0 for the legislation; ACQSC guidance documents under similar permissive licensing.
**Layer 1 v2 spec section:** §4.3
**Spec deadline:** none (already in force; load-bearing for Consent state machine in Phase 1C-δ)

## What to download

1. **Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022**
   - Source: https://www.legislation.gov.au/
   - Why: introduced consolidated restrictive-practice authorisation framework
   - Maps to: Consent state machine inputs; Authorisation state machine (psychotropic + chemical restraint authority gating)

2. **Quality of Care Principles 2014 — Part 4A Restrictive Practices** (as amended)
   - Source: https://www.legislation.gov.au/
   - Why: operational definitions of behaviour-support plan, informed consent, monitoring requirements
   - Maps to: KB-29 psychotropic deprescribing template; Consent state machine

3. **ACQSC Guidance — Minimising Restrictive Practices**
   - Source: https://www.agedcarequality.gov.au/providers/standards/restrictive-practices
   - Why: regulator-facing operational guidance on what evidence the platform must produce
   - Maps to: KB-13 quality indicators; audit trail design

## Code path (Playwright)

```
1. browser_navigate → https://www.legislation.gov.au/
2. Search "Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022"
3. browser_evaluate with fetch()+base64 → decode to:
   Aged-Care-Royal-Commission-Response-Act-2022.pdf
4. Search "Quality of Care Principles 2014" → fetch latest compilation → save as:
   Quality-of-Care-Principles-2014-compilation.pdf
5. browser_navigate → https://www.agedcarequality.gov.au/providers/standards/restrictive-practices
6. Download published ACQSC restrictive-practices guidance PDFs; save with descriptive names
```

## Manual fallback

1. Browse https://www.legislation.gov.au/; search by short title; download PDF.
2. Browse https://www.agedcarequality.gov.au/providers/standards/restrictive-practices; download linked PDFs.
3. Save all to: `regulatory/commonwealth/restrictive_practice/`.

## After PDFs land

- [ ] Files verified
- [ ] SHA-256 hashes recorded below
- [ ] MANIFEST.md row 7 updated to ✅ landed
- [ ] Source Registry seed (Phase 1C-β)
- [ ] Consent state machine inputs encoded (Phase 1C-δ)

## File hashes (post-procurement)

| File | SHA-256 | Bytes |
|---|---|---|
| Aged-Care-Royal-Commission-Response-Act-2022.pdf | bb95b7f365c52d777fb68fced13243efb4720f09be2f565eb99a9febab1898a3 | 1751821 |
| Quality-of-Care-Principles-2014-compilation-2025-10-01.pdf | adc8fb57a17b05ce57c5d3f5d6d7917d0b1b01851df11a931d3ae590e100d0aa | 520506 |
| ACQSC-Overview-of-Restrictive-Practices.pdf | 24974f7e05fd55985e195779226085d1c5bf48b514ba8abd4c99735a48623de7 | 465425 |
| ACQSC-Behaviour-Support-Plans-FactSheet.pdf | 45c6c432aa308b9db7f1d420ac7af541c3f850c2d74d460a1ea04f39305f7695 | 1288240 |
| ACQSC-Consent-for-Medication-FactSheet.pdf | 235aae1f0df28bfb380eb260392fb1b7063caae7509ea80f3fa6e1fc0cc1a61c | 460612 |
| ACQSC-FAQ-Consent.pdf | d4ef1f072fadd80100bb625e74654a3dd1fcaf9803cd53880030da4c16ada304 | 433264 |
| ACQSC-RB-2023-22-Senior-Practitioner.pdf | d2e9578dfc8de51e3648babfba66fe636e67266bc8e910be151410dd4135bc66 | 738156 |
| ACQSC-Psychotropic-PRN-Stickers-Guide.pdf | 6fdf68826d9ae3430a5131b25650a152bceba70f9db39974e85a60953b156391 | 692756 |
| ACQSC-Restrictive-Practice-Scenarios.pdf | af886212ceb8a5bcc40444f0a7cc6c21cff8fcac88d180c25df73704d9a84bf1 | 481037 |

## Procurement notes (2026-05-04)

- Royal Commission Response Act 2022 procured as-made from `legislation.gov.au` (`C2022A00034`).
- Quality of Care Principles 2014 latest compilation (2025-10-01, immediately pre-repeal on 2025-11-01 by `F2025L01305`) captured. Subsequent Aged Care Rules 2025 supersede this instrument; both held for transition-period coverage.
- ACQSC operational guidance procured from the agency's resource library (topic 8397 — "Minimising the use of restrictive practices"). Seven guidance PDFs cover Behaviour Support Plans, consent (medication + general), Senior Practitioner role, psychotropic PRN stickers, and scenarios — covering the primary inputs to the platform's Consent state machine and Authorisation state machine for chemical restraint.
