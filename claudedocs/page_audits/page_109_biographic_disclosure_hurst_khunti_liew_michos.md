# Page 109 Audit — Biographic & Disclosure: Hurst, Khunti, Liew, Michos (2 Spans)

## Page Identity
- **PDF page**: S108 (Biographic and Disclosure Information — www.kidney-international.org)
- **Content**: Heerspink COI conclusion; Clint Hurst bio (patient representative — kidney transplant recipient, no COI); Kamlesh Khunti bio + COI; Adrian Liew bio + COI; Erin Michos bio (start)
- **Clinical tier**: T3 (Informational — author biographies and disclosures)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 2 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 2 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM) |
| Tier accuracy | 0% (0/2) |
| Disagreements | 0 |
| Review Status | PENDING: 2 |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |
| Audit Date | 2026-02-25 (revised) |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "CH declared no competing interests." | F | 90% | T2 | T3 | Clint Hurst — no COI declaration |
| 2 | "*Monies paid to institution." | F | 90% | T2 | T3 | Disclosure footnote (3rd occurrence: pp107, 108, 109) |

### Minimal Extraction on Dense Page
This page contains 5 author biographies and 4 COI disclosures — far more content than pages 107-108. Yet only 2 spans were extracted, compared to 11 spans on page 107 and 7 on page 108. The F channel appears to have reduced its extraction aggressiveness on this page, possibly due to the absence of triggering patterns (the two extracted spans are short, formulaic statements rather than long COI lists).

### Notable: "*Monies paid to institution." Recurrence
This disclosure footnote has now appeared on 3 consecutive pages (107, 108, 109), extracted independently each time at 90% confidence. It serves as a recurrent boilerplate extraction — the F channel treats each page's instance as a new entity.

### No HTML Artifact
Second consecutive biographic page without the `<!-- PAGE XX -->` artifact.

## PDF Source Content
- **Heerspink COI** (conclusion from p108): Novo Nordisk*, Travere Pharmaceuticals* (consultancy); Abbvie*, AstraZeneca*, Boehringer Ingelheim*, Janssen*, Novo Nordisk* (grants); AstraZeneca, Bayer (speaker)
- **Clint Hurst, BS** (Patient Representative): Retired special education teacher; kidney transplant June 13, 2017 at DeBakey VA Medical Center; Vietnam War veteran. **No competing interests declared.**
- **Kamlesh Khunti, MD, PhD, FRCP**: Professor at University of Leicester; NIHR ARC East Midlands director; SAGE member + Ethnicity Sub-panel chair; 1100+ publications
- **Khunti COI**: Amgen, AstraZeneca, Bayer, Boehringer Ingelheim, Eli Lilly, Janssen, MSD, Novartis, Novo Nordisk, Roche, Sanofi, Servier (consultancy); + Berlin-Chemie AG/Menarini, Napp (speaker additional); AstraZeneca*, Boehringer Ingelheim*, Eli Lilly*, Janssen*, MSD*, Novartis*, Novo Nordisk*, Roche*, Sanofi* (grants); NIHR ARC EM + Leicester BRC (general)
- **Adrian Liew, MBBS, MRCP**: Senior consultant nephrologist at Mount Elizabeth Novena Hospital, Singapore; ISN Executive Committee; ISPD Honorary Secretary; John Maher Award 2020; associate editor Nephrology
- **Liew COI**: Alnylam, AstraZeneca, Baxter, Bayer, Boehringer Ingelheim, Chinook, DaVita, Eledon, George Clinical, Otsuka, ProKidney (consultancy); Baxter, Chinook, DKSH, Otsuka (speaker)
- **Erin D. Michos, MD, MHS** (starts): Associate professor at Johns Hopkins; director of women's cardiovascular health; 550+ publications; research in CV disease in women, coronary artery calcium, lipids, diabetes/cardiometabolic disease (continues on next page)

## Quality Assessment
| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | VERY LOW | 2 spans for dense 5-biography page |
| Tier accuracy | 0% | Both should be T3 |
| Clinical safety risk | NONE | Biographies and COI disclosures |
| Pipeline bugs | 0 | No HTML artifact |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Author biographies and conflict-of-interest disclosures.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 109 of 126
