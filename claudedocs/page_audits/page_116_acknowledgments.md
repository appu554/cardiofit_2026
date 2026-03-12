# Page 116 Audit — Acknowledgments (1 Span)

## Page Identity
- **PDF page**: S115 (Acknowledgments — www.kidney-international.org)
- **Content**: KDIGO acknowledgments — thanks to Co-Chairs, ERT, Work Group; **200+ external reviewer names** from 2022 and 2020 public review rounds
- **Clinical tier**: T3 (Informational — acknowledgments and reviewer listing)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 1 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 1 |
| T3 (Informational) | 0 |
| Channels present | F (NuExtract LLM) |
| Tier accuracy | 0% (0/1) |
| Disagreements | 0 |
| Review Status | PENDING: 1 |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## All Spans

| # | Text | Channel | Conf | Tier | Correct Tier | Issue |
|---|------|---------|------|------|-------------|-------|
| 1 | "2020:" | F | 90% | T2 | NOISE | Bare year marker from "2020:" reviewer list subheading |

### Minimal Extraction on Dense Page
This is one of the densest pages in the entire document — containing 200+ reviewer names across two public review rounds (2022 and 2020). Yet the pipeline extracted only a single span: the bare text "2020:" (a year subheading separating the two review rounds). This suggests:
1. The F channel LLM correctly identified that lists of proper names are not clinical content
2. But it was triggered by "2020:" as a potential temporal/date entity
3. No B, C, D, or E channels fired on this page at all

### No HTML Artifact
No `<!-- PAGE 116 -->` HTML artifact — the F channel's sole extraction was the "2020:" marker.

## PDF Source Content
- **Acknowledgments section** thanking KDIGO Co-Chairs (Jadoul, Winkelmayer), ERT (Craig, Strippoli, Howell, Tunnicliffe), Tonelli and Lytvyn for evidence-recommendation linkage, Debbie Maizels for artwork, and Work Group members
- **2022 External Reviewers**: ~80 names (Baris Afsar through Carmine Zoccali)
- **2020 External Reviewers**: ~120+ names (Georgi Abraham through Evgueniy Vazelov, continues next page)

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Acknowledgments and external reviewer name listing.

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Page in sequence**: 116 of 126
