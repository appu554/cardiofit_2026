# Page 114 Audit — Evidence Review Team Biographies (7 Spans)

## Page Identity
- **PDF page**: S113 (Biographic and Disclosure Information — www.kidney-international.org)
- **Content**: Evidence Review Team: Jonathan Craig (director), Giovanni Strippoli (co-director), David Tunnicliffe (project leader), Gail Higgins (information specialist), Patrizia Natale (research associate)
- **Clinical tier**: T3 (Informational — ERT biographies and no-COI declarations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 7 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 7 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM) |
| Tier accuracy | 0% (0/7) |
| Disagreements | 0 |
| Review Status | PENDING: 7 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | `<!-- PAGE 114 --> Evidence Review Team` | F | 90% | T2 | NOISE/BUG | **HTML artifact (13th occurrence)** + section heading |
| 2 | "DJT declared no competing interests." | F | 90% | T2 | T3 | Tunnicliffe — no COI |
| 3 | "JCC declared no competing interests." | F | 90% | T2 | T3 | Craig — no COI |
| 4 | "Gail Y. Higgins, BA, Grad Ed, Grad Dip LibSc, Information Specialist" | F | 85% | T2 | T3 | Higgins bio — title/credentials |
| 5 | "GYH declared no competing interests." | F | 90% | T2 | T3 | Higgins — no COI |
| 6 | "GFMS declared no competing interests." | F | 90% | T2 | T3 | Strippoli — no COI |
| 7 | "designed and conducted multiple Cochrane systematic" | F | 90% | T2 | T3 | Natale bio — research description (truncated) |

### Evidence Review Team — All Clean COI
Notable: All 5 ERT members declared **no competing interests**. This is in stark contrast to the Work Group members (pages 107-113) who had extensive pharmaceutical industry COI disclosures. The pipeline extracted 4 of the 5 "no COI" declarations (missing Natale's — "PN declared no competing interests" not captured).

### HTML Artifact Returns
After being absent on pages 108-113 (6 pages), the `<!-- PAGE 114 -->` HTML artifact returns. Now confirmed on 13 pages total.

### F Channel "No COI" Pattern
The F channel consistently extracts "X declared no competing interests" statements at 90% confidence. These formulaic sentences appear to be highly triggering for the LLM extractor — likely because they contain pharmaceutical/medical disclosure keywords.

## PDF Source Content
- **Jonathan C. Craig, MBChB, DipCH, FRACP, M Med, PhD** (ERT Director): VP and executive dean at Flinders University; CKD clinical research; Cochrane Steering Group past-chairman; NHMRC, PBAC, MSAC advisory roles. **No COI.**
- **Giovanni F.M. Strippoli, MD, MPH, M Med, PhD** (ERT Co-Director): CKD prevention, dialysis, transplantation research; ISN and Italian Society of Nephrology positions; systematic reviews and RCTs. **No COI.**
- **David J. Tunnicliffe, PhD** (ERT Project Leader/Manager): Research fellow at University of Sydney; NHMRC Emerging Leadership grant; evidence synthesis, living evidence, clinical practice guidelines; coordinated KDIGO 2022 evidence review. **No COI.**
- **Gail Y. Higgins, BA, Grad Ed, Grad Dip LibSc** (Information Specialist): University of Sydney Library background; Cochrane information specialist; WHO ICTRP secondment (2007-2008). **No COI.**
- **Patrizia Natale, PhD, MSc** (Research Associate): University of Sydney, University of Bari, University of Foggia; epidemiological studies and evidence syntheses; Cochrane systematic reviews. **No COI declared** (but extraction truncated mid-sentence).

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Evidence Review Team biographies and no-COI declarations.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 114 of 126
