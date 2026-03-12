# Page 90 Audit — Chapter 5: Recommendation 5.1.1, Self-Management Education

## Page Identity
- **PDF page**: S89 (Chapter 5 — www.kidney-international.org)
- **Content**: Chapter 5 opening — Recommendation 5.1.1 (structured self-management education program, 1C grade), Key Information, Figure 30
- **Clinical tier**: Mixed — Recommendation itself is T2 (clinical accuracy), supporting prose is T3 (informational)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES (1 span with C+F channel disagreement)

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 5 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 4 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 (see analysis) |
| Tier accuracy | 0% (0/5) |
| Disagreements | 1 |
| Review Status | FINAL: 16 ADDED, 0 CONFIRMED, 7 REJECTED |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 5 original rejected as C/F noise), 2 reviewer spans also rejected, 16 gaps added |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts confirmed (5 total: 1 T1, 4 T2), channels confirmed (C, F) |

## Channel Breakdown
| Channel | Count | Confidence | Content Type |
|---------|-------|------------|-------------|
| C (Grammar/Regex) | 1 solo + 1 multi-channel | 98%, 90% | Recommendation label + evidence summary |
| F (NuExtract LLM) | 3 solo + 1 multi-channel | 85%, 90% | Rationale prose extractions |

## T1 Spans (1) — MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Recommendation 5.1.1" | C | 98% | T3 | Recommendation label/number only — not the recommendation content itself |

**Analysis**: C channel extracted the recommendation header label "Recommendation 5.1.1" and classified it as T1 (patient safety). This is just a section identifier — the actual recommendation text ("We recommend that a structured self-management educational program be implemented...") was NOT extracted as a span. The label alone has no clinical safety content.

## T2 Spans (4) — ALL MISTIERED

| # | Text | Channels | Conf | Disagree? | Correct Tier | Issue |
|---|------|----------|------|-----------|-------------|-------|
| 1 | "group-based diabetes self-management education programs in people with T2D result in improvements in clinical outcomes (..." | C+F | 90% | YES | T3 | Evidence summary from systematic review — informational |
| 2 | "Diabetes self-management education programs are guided by learning and behavior-change theories, are tailored to a perso..." | F | 85% | No | T3 | Descriptive rationale prose |
| 3 | "Self-management programs delivered from diagnosis can promote medication adherence, healthy eating, physical activity, a..." | F | 85% | No | T3 | Rationale prose about program benefits |
| 4 | "There is no expected or anticipated harm to patients if diabetes self-management and education support (DSMES)" | F | 85% | No | T3 | Harm assessment narrative — not a safety threshold |

**Analysis**: All 4 T2 spans are rationale/evidence prose from the "Key Information" and "Balance of benefits and harms" subsections. These are supporting narrative explaining why the recommendation was made — classic T3 informational content. None contain monitoring intervals, titration steps, or lab thresholds that would qualify as T2.

### Disagreement Analysis
Span #1 (C+F 90%) has a disagreement flag. Both C and F channels extracted overlapping text about group-based DSMES outcomes, but likely with different span boundaries or slightly different extracted text. The disagreement is between extraction boundaries, not clinical interpretation. Both channels agree this is evidence summary text.

## PDF Source Content Analysis

### Content Present on Page
1. **Chapter 5 heading**: "Approaches to management of patients with diabetes and CKD"
2. **Section 5.1**: "Self-management education programs"
3. **Recommendation 5.1.1**: "We recommend that a structured self-management educational program be implemented for care of people with diabetes and CKD (Figure 30) (1C)"
4. **Recommendation context**: Places high value on potential benefits of structured education programs, lower value on lack of high-quality evidence specifically in diabetes+CKD
5. **Key Information — Balance of benefits and harms**: Description of DSMES program design, objectives, evidence from systematic review (21 studies, 2833 participants)
6. **Figure 30**: Key objectives of effective diabetes self-management education programs (reprinted from Lancet Diabetes & Endocrinology 2018)
   - Improve emotional and mental well-being
   - Reduce risk of diabetes-related complications
   - Increase engagement with medication/monitoring
   - Improve vascular risk factors
   - Encourage healthy lifestyles
   - Improve self-management and self-motivation
   - Improve diabetes-related knowledge, beliefs, skills

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| Rec 5.1.1 full text ("We recommend structured self-management educational program... 1C") | T2 | NO — only label extracted |
| "1C" evidence grade | T3 | NO |
| Systematic review outcome (21 studies, 2833 participants, improvements in HbA1c, fasting glucose, body weight) | T3 | Partially (summary extracted but mistiered) |
| "contact time of more than 10 hours" threshold | T2 | NO — specific program design parameter |
| Figure 30 objectives list | T3 | NO |

### Critical Missing Extraction
**Recommendation 5.1.1 full text** — The actual recommendation statement is the most important content on the page. Only the label "Recommendation 5.1.1" was extracted (as T1), while the recommendation body text was missed entirely. This is a significant extraction gap.

**"Contact time of more than 10 hours"** — This is a specific clinical threshold for program design (T2) that was not extracted.

## Cross-Page Patterns

### Chapter 5 Transition
Page 90 marks the transition from Chapter 4 (Glucose-Lowering Therapies) to Chapter 5 (Approaches to Management). The extraction quality shifts:
- Chapter 4 had heavy B channel drug name extraction (often noisy)
- Chapter 5 opens with F channel dominance (3/5 spans are F-only) — NuExtract LLM extracting prose passages
- This suggests F channel is better suited to narrative/educational content but still misclassifies tier

### F Channel Prose Extraction Pattern
This is the most F-heavy page since Chapter 4 began. F channel extracts meaningful sentence-level prose rather than single terms, but:
- All F extractions are rationale/evidence prose (T3), not clinical parameters (T2)
- F channel's 85% confidence on prose is consistent with its uncertainty about clinical relevance
- The C+F multi-channel span at 90% confidence shows channel agreement boosting confidence

### Recommendation Extraction Gap
The extraction pipeline captures the recommendation label but misses the recommendation body. This pattern may recur throughout Chapter 5 if the C channel regex matches "Recommendation X.Y.Z" headers but doesn't capture the following sentence.

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | LOW | Rec 5.1.1 body text missing, Figure 30 not captured, "10 hours" threshold missed |
| Tier accuracy | 0% | 1 T1 should be T3, 4 T2 should all be T3 |
| Clinical safety risk | NONE | Self-management education content has no patient safety implications |
| Channel diversity | MODERATE | C + F present (no B drug names on this page) |
| Noise level | HIGH | All 5 spans mistiered |
| Disagreement handling | ACCEPTABLE | C+F disagreement is boundary-level, not clinical |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. This is educational/programmatic content about self-management education. All spans are informational prose mistiered as T1/T2. The missing recommendation body text is a pipeline improvement item, not a safety concern.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 5
- **ADDED**: 0
- **PENDING**: 5 (all original agent spans)
- **CONFIRMED**: 0
- **REJECTED**: 0

### Agent Spans Kept: 0
All 5 original agent spans rejected:
- `e4488dc0` "Recommendation 5.1.1" — C channel label only (T1 mistiered)
- `a7f12f5e` "group-based diabetes self-management education programs..." — C+F disagreement span
- `6f20141f` "Diabetes self-management education programs are guided..." — F channel prose
- `005d25db` "Self-management programs delivered from diagnosis..." — F channel prose
- `e3a9ca11` "There is no expected or anticipated harm..." — F channel harm assessment

Additionally 2 reviewer-added spans were rejected as duplicates:
- `7873db7a` "contact time of more than 10 hours" — subsumed into larger gap
- `22b5fb0c` "We recommend that a structured self-management..." — duplicate of kept version

### Gaps Added (16) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G90-A | We recommend that a structured self-management educational program be implemented for care of people with diabetes and CKD (Figure 30) (1C). | Rec 5.1.1 full text — KB-4 |
| G90-B | This recommendation applies to patients with T1D or T2D. | Rec 5.1.1 applicability |
| G90-C | The best outcomes are achieved in those programs with a theory-based and structured curriculum and with a contact time of more than 10 hours. | Program design threshold — KB-4 |
| G90-D | Diabetes self-management education programs are guided by learning and behavior-change theories, are tailored to a person's needs... | DSMES program description |
| G90-E | Potential benefits are summarized in a systematic review of 21 studies (26 publications, 2833 participants), which showed improvements in HbA1c, fasting glucose, body weight... | Evidence summary — KB-16 |
| G90-F | Lifestyle management, including medical nutrition therapy, physical activity, weight loss, counseling for smoking cessation... | Lifestyle management scope |
| G90-G | Self-management programs delivered from diagnosis can promote medication adherence, healthy eating, physical activity, and reduce complications... | Program benefits |
| G90-H | Although online programs may reinforce learning, there is little evidence to date that they are effective when used alone. | Online program evidence |
| G90-I | There is no expected or anticipated harm to patients if diabetes self-management and education support (DSMES)... | Harm assessment |
| G90-J | Improve emotional and mental well-being, treatment satisfaction, and quality of life | Figure 30 objective 1 |
| G90-K | Improve diabetes-related knowledge, beliefs, and skills | Figure 30 objective 2 |
| G90-L | Improve self-management and self-motivation | Figure 30 objective 3 |
| G90-M | Encourage adoption and maintenance of healthy lifestyles | Figure 30 objective 4 |
| G90-N | Improve vascular risk factors | Figure 30 objective 5 |
| G90-O | Increase engagement with medication, glucose monitoring, and complication screening programs | Figure 30 objective 6 |
| G90-P | Reduce risk to prevent (or better manage) diabetes-related complications | Figure 30 objective 7 |

### Post-Review State
- **Total spans**: 23
- **ADDED**: 16 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 7 (5 original agent + 2 reviewer duplicates)
- **P2-ready facts**: 16

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G90-A, G90-B, G90-C | KB-4 (Safety) | Rec 5.1.1 — self-management education recommendation with grade and threshold |
| G90-D–I | KB-4 (Safety) | Evidence, benefits, harms of DSMES programs |
| G90-E | KB-16 (Monitoring) | Systematic review with HbA1c, fasting glucose outcomes |
| G90-J–P | KB-4 (Safety) | Figure 30 objectives — key program goals |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 90 of 126
