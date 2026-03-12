# Page 111 Audit — Biographic & Disclosure: Sadusky (cont), Tandon, Tuttle, Wanner (6 Spans)

## Page Identity
- **PDF page**: S110 (Biographic and Disclosure Information — www.kidney-international.org)
- **Content**: Sadusky bio conclusion + no COI; Nikhil Tandon bio + COI; Katherine Tuttle bio + COI; Christoph Wanner bio (start)
- **Clinical tier**: T3 (Informational — author biographies and disclosures)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 6 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 5 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM), B (Drug Dictionary) |
| Tier accuracy | 0% (0/6) |
| Disagreements | 0 |
| Review Status | PENDING: 6 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "statin" | B | 100% | T1 | T3 | Drug class from Wanner bio ("statin treatment in hemodialysis patients") |
| 2 | "TS declared no competing interests." | F | 90% | T2 | T3 | Sadusky — no COI |
| 3 | "NT reports grant support from Government of India; Indian Council of Medical Research; National Heart, Lung, and Blood I..." | F | 85% | T2 | T3 | Tandon COI — grants |
| 4 | "Dr. Tuttle's major research interests are in clinical and translational science for diabetes and CKD." | F | 85% | T2 | T3 | Tuttle bio — research interests |
| 5 | "participated in the leadership of several implementation research studies, funded through the National Institutes of Hea..." | F | 85% | T2 | T3 | Tandon bio — research description |
| 6 | "KRT reports consultancy fees from AstraZeneca, Boehringer Ingelheim, Eli Lilly and Company, Gilead, Goldfinch Bio, Novo ..." | F | 85% | T2 | T3 | Tuttle COI disclosure |

### B Channel "statin" as T1
The B drug dictionary extracted "statin" from Wanner's biographical description: "recognized for his contributions to the field of cardiovascular disease, lipid disorders, and **statin** treatment in hemodialysis patients." This is a biographical description of research expertise, not a clinical recommendation. The B channel at 100% confidence + T1 classification means this page is **ACCEPT-gated** — the T1 span blocks the Accept button until reviewed.

## PDF Source Content
- **Tami Sadusky** (continued): Executive Director at UW for 22 years; Transplant House board; established Sadusky Endowed Fund for Diabetes, Kidney, and Transplant Research; KDIGO contributor since 2020. **No COI.**
- **Nikhil Tandon, MBBS, MD, PhD**: Professor at AIIMS New Delhi; 550+ publications (56,000+ citations); Padma Shri awardee; National Academy of Medical Sciences fellow; MCI Board of Governors
- **Tandon COI**: Government of India, ICMR, NHLBI/NIH, Novo Nordisk (grants only)
- **Katherine R. Tuttle, MD, FASN**: Executive director for research at Providence Health Care; professor at University of Washington Spokane; 300+ publications; ASN DKD Collaborative Task Force chair
- **Tuttle COI**: AstraZeneca, Boehringer Ingelheim, Eli Lilly, Gilead, Goldfinch Bio, Novo Nordisk, Travere (consultancy); Bayer*, Goldfinch Bio*, Novo Nordisk*, Travere (grants); AstraZeneca, Eli Lilly, Gilead, Goldfinch Bio, Janssen, Novo Nordisk (speaker)
- **Christoph Wanner, MD** (start): Professor and head of Nephrology at University Hospital Würzburg; cardiovascular disease, lipid disorders, statin treatment in hemodialysis (continues next page)

## Quality Assessment
| Dimension | Rating | Notes |
|-----------|--------|-------|
| Tier accuracy | 0% | All should be T3 |
| Clinical safety risk | NONE | Biographies and COI disclosures |
| Key issue | B "statin" from bio research description → T1 gates Accept |
| Pipeline bugs | 0 | No HTML artifact |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Author biographies and conflict-of-interest disclosures.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 111 of 126
