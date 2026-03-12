# Page 110 Audit — Biographic & Disclosure: Michos (cont), Navaneethan, Olowu, Sadusky (4 Spans)

## Page Identity
- **PDF page**: S109 (Biographic and Disclosure Information — www.kidney-international.org)
- **Content**: Michos bio conclusion + COI; Sankar Navaneethan bio + COI; Wasiu Olowu bio (no COI); Tami Sadusky bio start (patient representative — pancreas/kidney transplant)
- **Clinical tier**: T3 (Informational — author biographies and disclosures)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 4 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 4 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM) |
| Tier accuracy | 0% (0/4) |
| Disagreements | 0 |
| Review Status | PENDING: 4 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "SDN reports consultancy fees from AstraZeneca, ACI Clinical, Bayer Pharmaceuticals, Boehringer Ingelheim/Lilly, Vertex, ..." | F | 90% | T2 | T3 | Navaneethan COI disclosure |
| 2 | "Wasiu A. Olowu, MBBS, FMCPaed, graduated from the College of Medicine of the University of Lagos, Nigeria in 1985." | F | 85% | T2 | T3 | Olowu bio opening sentence |
| 3 | "the chair of the Pediatric Nephrology and Hypertension Unit, Department of Pediatrics, OAUTHC, since 1994." | F | 90% | T2 | T3 | Olowu bio — title/role |
| 4 | "WAO declared no competing interests." | F | 90% | T2 | T3 | Olowu — no COI declaration |

### F Channel Selection Pattern
Interesting selectivity: F channel extracted 2 fragments from Olowu's bio (opening + title) but nothing from the much longer Michos or Navaneethan bios. It also extracted Navaneethan's COI statement but not Michos's COI. The pattern suggests the F channel is not systematically processing each biography — it selects fragments somewhat unpredictably.

### Biographic Section Summary (Pages 107-110)
| Page | Spans | Authors Covered | COI Extracted? |
|------|-------|----------------|----------------|
| 107 | 11 | de Boer, Rossing, Caramori (start) | Yes — fragmented per category |
| 108 | 7 | Caramori (cont), Chan, Heerspink (start) | Yes — Chan fragmented |
| 109 | 2 | Heerspink (cont), Hurst, Khunti, Liew, Michos (start) | Minimal — only "no COI" + footnote |
| 110 | 4 | Michos (cont), Navaneethan, Olowu, Sadusky (start) | Partial — SDN only |
| **Total** | **24** | **10 authors** | Highly inconsistent |

## PDF Source Content
- **Erin Michos** (continued): Co-Editor-in-Chief American Journal of Preventive Cardiology; associate editor Circulation; ASPC Board of Directors; ACC Prevention Leadership Council; AHA Funding Committee; MESA and ARIC co-investigator; mentored 60+ individuals
- **Michos COI**: AstraZeneca, Bayer, Boehringer Ingelheim, Esperion, Novartis, Novo Nordisk, Pfizer (consultancy)
- **Sankar D. Navaneethan, MD, MS, MPH**: Professor at Baylor College of Medicine; MD from Madras Medical College; MPH from University of South Carolina; 275+ publications; associate editor AJKD since 2017; NephSAP-CKD co-editor 2015-2019
- **Navaneethan COI**: AstraZeneca, ACI Clinical, Bayer, Boehringer Ingelheim/Lilly, Vertex, Vifor (consultancy)
- **Wasiu A. Olowu, MBBS, FMCPaed**: Full professor at Obafemi Awolowo University, Nigeria; chair of Pediatric Nephrology and Hypertension Unit since 1994; APOL1/MYH9 CKD research (H3AFRICA network); 50+ publications. **No competing interests.**
- **Tami Sadusky, MBA** (Patient Representative, start): Pancreas + kidney transplant 1993; second kidney transplant 2011; diagnosed T1D age 13; developed kidney failure within 20 years (continues on next page)

## Quality Assessment
| Dimension | Rating | Notes |
|-----------|--------|-------|
| Tier accuracy | 0% | All should be T3 |
| Clinical safety risk | NONE | Biographies and COI disclosures |
| Key pattern | F channel inconsistently selects bio fragments across pages |
| Pipeline bugs | 0 | No HTML artifact |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Author biographies and conflict-of-interest disclosures.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 110 of 126
