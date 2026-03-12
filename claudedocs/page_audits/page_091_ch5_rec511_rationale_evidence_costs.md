# Page 91 Audit — Chapter 5: Rec 5.1.1 Rationale, Evidence Quality, Costs

## Page Identity
- **PDF page**: S90 (Chapter 5 — www.kidney-international.org)
- **Content**: Continuation of Rec 5.1.1 rationale — NICE guideline components, quality of evidence (low), values and preferences, resource use and costs
- **Clinical tier**: T3 (Informational — evidence review, rationale, and cost-effectiveness discussion)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 4 |
| T1 (Patient Safety) | 0 |
| T2 (Clinical Accuracy) | 4 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), F (NuExtract LLM) |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/4) |
| Disagreements | 0 |
| Review Status | FINAL: 16 ADDED, 0 CONFIRMED, 8 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 8 rejected: 4 D "Usual care" duplicates, 1 F HTML artifact, 1 C bare temporal, 2 F prose), 16 gaps added |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — count corrected 8→4, D channel removed (raw data has C, F only) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | 4 | 92% | "Usual care" ×4 (comparator term) |
| F (NuExtract LLM) | 3 | 85-90% | HTML comment artifact + 2 prose passages |
| C (Grammar/Regex) | 1 | 85% | "annually" (bare temporal word) |

## T2 Spans (8) — ALL MISTIERED OR NOISE

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Usual care" | D | 92% | NOISE | Comparator term from supplementary table — not clinical parameter |
| 2 | "Usual care" | D | 92% | NOISE | Duplicate of #1 |
| 3 | "Usual care" | D | 92% | NOISE | Duplicate of #1 |
| 4 | "Usual care" | D | 92% | NOISE | Duplicate of #1 |
| 5 | "<!-- PAGE 91 -->" | F | 90% | NOISE/BUG | **HTML comment artifact** — pipeline metadata leaked into extraction |
| 6 | "programs are commissioned and delivered according to evidenced-based guidelines." | F | 85% | T3 | Rationale prose from NICE guidelines section |
| 7 | "annually" | C | 85% | NOISE | Bare temporal word without context |
| 8 | "The evidence review included RCTs that focused on educational programs in patients with diabetes and CKD to prevent the ..." | F | 85% | T3 | Evidence review narrative |

### Pipeline Bug: HTML Comment Extraction
Span #5 is a **pipeline artifact** — the F (NuExtract LLM) channel extracted `<!-- PAGE 91 -->`, which is an HTML comment marker from the document processing pipeline. This should never appear as a clinical span. At 90% confidence, this suggests the LLM model is not filtering out markup artifacts before extraction. **This is a bug that should be reported.**

### D Channel Quadruple Duplicate
The D (Table Decomp) channel extracted "Usual care" 4 times at 92% confidence. This likely comes from Supplementary Tables S24-S26 referenced on this page, where "Usual care" appears as a comparator arm label. Table Decomp correctly identified table content but:
- "Usual care" is a study arm label, not a clinical parameter
- 4 identical extractions suggest no deduplication in the D channel pipeline
- Correct classification would be T3 (informational) at most

### C Channel Bare Temporal Word
"annually" is extracted from the NICE guidelines list item: "is available to patients at critical times (i.e., at diagnosis, **annually**, when complications arise, and when transitions in care occur)". The C channel extracted the single word without its clinical context. With the full sentence, this would be T2 (monitoring interval), but the bare word "annually" alone is insufficient — it lacks what should happen annually.

## PDF Source Content Analysis

### Content Present on Page
1. **NICE guideline components for self-management education** — 12-item checklist (evidence-based, individualized, structured curriculum, trained educators, group/individual settings, local alignment, family support, core content, availability at critical times, progress monitoring, quality assurance)
2. **Quality of evidence**: Overall LOW
   - 2 RCTs comparing self-management with multifactorial care (Supplementary Table S24)
   - 1 RCT comparing self-management + routine treatment vs. routine treatment alone (Supplementary Table S25)
   - Systematic review of 8 RCTs on self-management support in CKD (Supplementary Table S26, Figures 31-32)
   - Evidence downgraded for heterogeneity, self-report reliance, lack of blinding
3. **Values and preferences**: Work Group judged strong recommendation — patients would choose self-management as cornerstone of chronic care
4. **Resource use and costs**:
   - Self-management education likely cost-effective long-term (systematic review of 8 RCTs)
   - More cost-effective than usual care (review of 22 studies)
   - Telemedical delivery potentially not cost-effective
   - Over half of self-management approaches associated with cost savings (review of 26 studies)

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| "contact time of more than 10 hours" (from previous page context) | T2 | NO |
| "available to patients at diagnosis, annually, when complications arise, transitions in care" | T2 | Partially — only bare "annually" extracted |
| NICE 12-component checklist | T3 | NO |
| Evidence quality: LOW | T3 | NO |
| Cost-effectiveness conclusions | T3 | NO |
| Supplementary Table S24-S26 references | T3 | Indirectly (via "Usual care" comparator) |

## Cross-Page Patterns

### Zero T1 Pages
Page 91 has zero T1 spans, meaning the Accept button is enabled without any gating requirement. This is correct — there is genuinely no patient safety content on this page.

### D Channel on Evidence Tables
This is the first time D (Table Decomp) has appeared extracting from supplementary table references rather than in-page tables. The "Usual care" comparator extraction suggests D channel is finding table-like content in the reference citations, producing noise.

### F Channel Artifact Leak
The `<!-- PAGE 91 -->` HTML comment extraction is the first documented pipeline artifact in the audit. This should be flagged as a P1 bug — markup metadata should be stripped before LLM extraction.

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | VERY LOW | NICE checklist, evidence quality, costs all missed |
| Tier accuracy | 0% | All 8 spans mistiered (should be T3 or NOISE) |
| Clinical safety risk | NONE | Evidence review and cost-effectiveness content |
| Channel diversity | MODERATE | D + F + C all present |
| Noise level | VERY HIGH | 5/8 spans are pure noise (4 duplicates + 1 HTML artifact) |
| Pipeline bugs | 1 | HTML comment artifact extraction (F channel) |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. All content is evidence review, guidelines rationale, and cost-effectiveness analysis (T3). The HTML artifact bug should be logged separately.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 8
- **ADDED**: 0
- **PENDING**: 8 (all original agent spans)
- **CONFIRMED**: 0
- **REJECTED**: 0

### Agent Spans Kept: 0
All 8 original agent spans rejected:
- `2235f1df`, `6b0e80c9`, `4bf49df1`, `eae32c60` — D channel "Usual care" ×4
- `353b5527` — F channel `<!-- PAGE 91 -->` HTML artifact
- `28f0922e` — F channel "programs are commissioned and delivered..." (partial prose)
- `85319068` — C channel "annually" (bare temporal word)
- `0dcbdf82` — F channel "The evidence review included RCTs..." (partial prose)

### Gaps Added (16) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G91-A | is evidence-based; | NICE checklist item 1 |
| G91-B | is individualized to the needs of the person, including language and culture; | NICE checklist item 2 |
| G91-C | has a structured theory-driven written curriculum with supporting materials; | NICE checklist item 3 |
| G91-D | is delivered by trained and competent individuals (educators) who are quality-assured; | NICE checklist item 4 |
| G91-E | is delivered in group or individual settings; | NICE checklist item 5 |
| G91-F | aligns with the local population needs; | NICE checklist item 6 |
| G91-G | supports patients and their families in developing attitudes, beliefs, knowledge, and skills to self-manage diabetes; | NICE checklist item 7 |
| G91-H | includes core content (i.e., diabetes pathophysiology and treatment options; medication usage; monitoring, preventing, detecting, and treating complications...); | NICE checklist item 8 |
| G91-I | is available to patients at critical times (i.e., at diagnosis, annually, when complications arise, and when transitions in care occur); | NICE checklist item 9 — includes "annually" monitoring |
| G91-J | includes monitoring of patient progress, including health status, and quality of life; and | NICE checklist item 10 |
| G91-K | has a quality assurance program. | NICE checklist item 11 |
| G91-L | Overall, the quality of the evidence was low because many critical and important outcomes were not reported, and surrogate outcomes were reported. | Evidence quality — LOW |
| G91-M | The recommendation is strong, as the Work Group felt that all or nearly all well-informed patients would choose self-management as a cornerstone of chronic care. | Values and preferences |
| G91-N | One recent systematic review of 8 RCTs concluded that the reduction of clinical risk factors in self-management education for CKD was likely cost-effective. | Cost-effectiveness evidence 1 |
| G91-O | Another review of 22 studies suggested that self-management education programs are more cost-effective than or superior to usual care. | Cost-effectiveness evidence 2 |
| G91-P | The review also found that telemedical methods of delivering programs were potentially not cost-effective. | Telemedical cost-effectiveness |

### Post-Review State
- **Total spans**: 24
- **ADDED**: 16 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 8 (all original noise)
- **P2-ready facts**: 16

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G91-A–K | KB-4 (Safety) | NICE guideline 12-component checklist for DSMES programs |
| G91-I | KB-16 (Monitoring) | Includes "annually" monitoring interval with full context |
| G91-L | KB-7 (Terminology) | Evidence quality rating |
| G91-M | KB-4 (Safety) | Values and preferences — strong recommendation rationale |
| G91-N–P | KB-4 (Safety) | Cost-effectiveness evidence for self-management education |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 91 of 126
