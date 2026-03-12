# Page 113 Audit — KDIGO Chairs: Jadoul, Winkelmayer, Tonelli (5 Spans)

## Page Identity
- **PDF page**: S112 (Biographic and Disclosure Information — www.kidney-international.org)
- **Content**: KDIGO Co-Chairs (Michel Jadoul + COI, Wolfgang Winkelmayer + COI) and Methods Chair (Marcello Tonelli + COI)
- **Clinical tier**: T3 (Informational — KDIGO leadership biographies and disclosures)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 5 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 5 |
| T3 (Informational) | 0 |
| Channels present | E (GLiNER NER), C (Grammar/Regex), F (NuExtract LLM) |
| Tier accuracy | 0% (0/5) |
| Disagreements | 0 |
| Review Status | PENDING: 5 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "sodium" | E | 85% | T2 | NOISE | Bare element name — likely from "sodium-glucose cotransporter-2" in text |
| 2 | "Weekly" | C | 85% | T2 | NOISE | Bare temporal word from journal title context |
| 3 | "Michel Jadoul, MD, received his MD degree in 1983 at the Université Catholique de Louvain (UCLouvain), Brussels, Belgium." | F | 85% | T2 | T3 | Jadoul bio opening |
| 4 | "Dr. Tonelli's research focuses on improving the care of people with" | F | 90% | T2 | T3 | Tonelli bio — research description (truncated) |
| 5 | "MAT reports speaker fees from AstraZeneca." | F | 90% | T2 | T3 | Tonelli COI — minimal (single disclosure) |

### Notable: E Channel Returns
First appearance of E (GLiNER NER) channel since the clinical chapters. "sodium" is likely extracted from the phrase "sodium-glucose cotransporter-2 inhibitor" which appears in biographical context. GLiNER recognized it as a chemical entity but lacks context awareness.

### Notable: 3-Channel Diversity on Biographic Page
This is the first biographic page with 3 channels active (E+C+F). The E and C channels contribute only noise words ("sodium", "Weekly") while F provides the substantive biographical content.

## PDF Source Content
- **Michel Jadoul, MD** (KDIGO Co-Chair): Professor at UCLouvain; 330+ papers; b2-microglobulin amyloidosis, hepatitis C, hemodialysis complications research; NKF International Distinguished Medal 2008
- **Jadoul COI**: Astellas, AstraZeneca, Bayer, Boehringer Ingelheim, Fresenius, Mundipharma, Vifor (consultancy); Amgen*, AstraZeneca* (grants); Astellas, AstraZeneca, Mundipharma, Vifor (speaker)
- **Wolfgang C. Winkelmayer, MD, MPH, ScD** (KDIGO Co-Chair): Gordon A. Cain Chair at Baylor; MD from University of Vienna; MPH + ScD from Harvard; 350+ publications; JAMA associate editor; KDIGO Co-Chair since 2016
- **Winkelmayer COI**: Akebia/Otsuka, AstraZeneca, Bayer, Boehringer Ingelheim/Lilly, GSK, Merck, Pharmacosmos, Reata, Zydus (consultancy)
- **Marcello A. Tonelli, MD, SM, MSc** (Methods Chair): Senior Associate Dean at University of Calgary; WHO Collaborating Centre director; "Highly Cited" researcher since 2015; NKF Medal 2013
- **Tonelli COI**: AstraZeneca (speaker fees only — minimal COI)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. KDIGO leadership biographies and COI disclosures.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 113 of 126
