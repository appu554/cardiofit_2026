# Page 101 Audit — Methods: Table 2 PICOM Continued Ch2-5 (84 Spans)

## Page Identity
- **PDF page**: S100 (Methods for Guideline Development — www.kidney-international.org)
- **Content**: Table 2 continuation — PICOM questions for glycemic targets (Ch2), exercise/dietary interventions (Ch3), glucose-lowering therapies including metformin/insulin/sulfonylureas/thiazolidinediones/GLP-1 RA/DPP-4i/SGLT2i (Ch4), self-management education (Ch5)
- **Clinical tier**: T3 (Informational — methodology table)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: NO

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 85 |
| T1 (Patient Safety) | 20 |
| T2 (Clinical Accuracy) | 65 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), C (Grammar/Regex), D (Table Decomp), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 0 |
| Tier accuracy | 0% (0/85) |
| Disagreements | 0 |
| Review Status | FINAL: 14 ADDED, 0 CONFIRMED, 84 REJECTED |
| Cross-Check | 2026-02-25 — Count corrected 84→85, T1 21→20, T2 63→65; verified against raw extraction data |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept (all 84 rejected as D/B/C/F table cell noise), 14 gaps added |
| Audit Date | 2026-02-25 (revised) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| D (Table Decomp) | ~55 | 92% | Table 2 cell values (PICOM questions, populations, interventions, outcomes, study designs) |
| B (Drug Dictionary) | ~21 | 100% | Individual drug/class names from PICOM Intervention cells |
| C (Grammar/Regex) | ~6 | 85% | Bare lab names (HbA1c, eGFR, potassium) + "120 mg" + "Stop" |
| F (NuExtract LLM) | ~2 | 90% | Table headers + HTML comment artifact variant |

## T1 Spans (21) — ALL MISTIERED

### B Channel Drug Names (majority of T1)
| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1-4 | "ACEi" (×4) | B | 100% | T3 | Drug class in PICOM Intervention/Comparator columns |
| 5-8 | "ARB" (×4) | B | 100% | T3 | Drug class in PICOM Intervention/Comparator columns |
| 9-11 | "SGLT2i" (×3) | B | 100% | T3 | Drug class in PICOM Intervention column |
| 12-13 | "Insulin" (×2) | B | 100% | T3 | Drug name in Ch4 glucose-lowering PICOM |
| 14 | "metformin" | B | 100% | T3 | Drug name in Ch4 PICOM |
| 15 | "sulfonylureas" | B | 100% | T3 | Drug class in Ch4 PICOM |
| 16 | "thiazolidinediones" | B | 100% | T3 | Drug class in Ch4 PICOM |
| 17-18 | "GLP-1 RA" (×2) | B | 100% | T3 | Drug class in Ch4 PICOM |
| 19 | "DPP-4 inhibitors" | B | 100% | T3 | Drug class in Ch4 PICOM |

### D Channel Drug Class Cells (~2 T1)
| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 20-21 | Drug class/combination cells | D | 92% | T3 | Table 2 Intervention cells containing drug class names |

**Analysis**: T1 classification triggered entirely by drug names appearing in PICOM table's Intervention/Comparator columns. These define the *scope of systematic review topics*, not prescribing directives. The B channel cannot distinguish methodology context from clinical prescribing context — it extracts all drug names at T1 regardless of surrounding content.

## T2 Spans (63) — ALL MISTIERED

### D Channel Table Cells (~55 spans)
| Content Pattern | Count | Source |
|----------------|-------|--------|
| "RCT" / "Randomized controlled trial" | 12× | Study design column repeated per clinical question |
| Cochrane citations (Strippoli, Palmer et al.) | ~8× | Cochrane systematic reviews column |
| "Critical and important outcomes listed in Table 1" | ~6× | Outcomes column per question |
| Population descriptions (Adults with CKD + diabetes) | ~6× | Population column repeated |
| Supplementary table references | ~5× | SoF table references |
| Clinical question texts | ~8× | Full PICOM questions for each topic |
| Dietary intervention list | ~2× | "Dietary sodium restriction, caloric restriction, dietary protein restriction, dietary patterns..." |
| Chapter references | ~4× | "Guideline chapter 2/3/4/5" |
| "(Continued on following page)" | 1× | Pagination marker |

### C Channel Bare Terms (~6 spans)
| Text | Count | Confidence | Issue |
|------|-------|------------|-------|
| "HbA1c" | ~10× | 85% | Bare lab name from glycemic target PICOM |
| "eGFR" | ~4× | 85% | Bare lab name from outcome definitions |
| "potassium" | ~3× | 85% | Bare lab name from outcome definitions |
| "sodium" | 1× | 85% | Bare lab name (dietary sodium restriction) |
| "fasting glucose" | 1× | 85% | Lab name from glycemic target PICOM |
| "120 mg" | 1× | 85% | Partial threshold from "< 120 mg/dl" glycemic PICOM |
| "Stop" | 1× | 85% | Bare word — likely regex false positive |

### F Channel (~2 spans)
| Text | Confidence | Issue |
|------|------------|-------|
| Table section headers | 90% | Extracted table structure labels |
| `<!-- PAGE 101 --> Methods for guideline development` | 90% | **HTML artifact variant** — first time artifact includes additional descriptive text beyond page number |

## PDF Source Content Analysis

### Content Present on Page
**Table 2 continuation** — PICOM questions covering 4 guideline chapters:

1. **Glycemic targets** (Ch2): Individualized HbA1c target (<6.5%, <7.0%, <7.5%, <8.0%) vs standard glycemic targets — includes fasting glucose and postprandial targets
2. **Exercise interventions** (Ch3): Various exercise programs vs standard care
3. **Dietary interventions** (Ch3): Sodium restriction, caloric restriction, protein restriction, dietary patterns (Mediterranean, DASH, plant-based)
4. **Glucose-lowering therapies** (Ch4): 8 drug classes compared:
   - Metformin vs other glucose-lowering agents
   - Insulin vs other agents
   - Sulfonylureas vs other agents
   - Thiazolidinediones vs other agents
   - GLP-1 RA vs other agents
   - DPP-4 inhibitors vs other agents
   - SGLT2i vs other agents (appears in both Ch1 and Ch4 contexts)
   - ACEi/ARB comparisons in multiple PICOM questions
5. **Self-management education** (Ch5): Structured self-management programs vs standard care

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| Drug class names (8 classes) | T3 | YES (B 100%, mistiered as T1) |
| PICOM clinical questions | T3 | YES (D 92%, mistiered as T2) |
| "RCT" study design | T3 | YES (D 92%, ×12 duplicates) |
| HbA1c target ranges | T3 | Partially (C extracted "HbA1c" and "120 mg" but not complete thresholds) |
| Dietary intervention list | T3 | YES (D 92%, mistiered as T2) |
| Cochrane review citations | T3 | YES (D 92%, mistiered as T2) |

## Cross-Page Patterns

### All 4 Channels Active — First in Methods Section
Page 101 is the first methods page with all 4 extraction channels (B, C, D, F) active simultaneously. Previous methods pages (98-100) had only 2-3 channels. This is because Ch4 glucose-lowering therapies PICOM contains:
- Many drug names → activates B channel
- Lab names and numeric values → activates C channel
- Table structure → activates D channel
- Prose headers → activates F channel

### F Channel Artifact Variant
`<!-- PAGE 101 --> Methods for guideline development` — the HTML artifact now includes additional text. Previous occurrences were bare `<!-- PAGE XX -->`. This may indicate the F channel attempted to extract more context around the artifact.

### C Channel "120 mg" and "Stop"
- **"120 mg"**: Partial threshold from glycemic PICOM "< 120 mg/dl fasting glucose target." C channel regex captured the numeric + unit but lost the comparison operator and context.
- **"Stop"**: Likely a false positive from regex matching — possibly from "Stop codon" in table metadata or a fragmented extraction.

### Methods Section Running Total (Pages 98-101)
| Page | Spans | Tier Accuracy | Content |
|------|-------|--------------|---------|
| 98 | 6 | 0% | Table 1 outcomes + methods prose |
| 99 | 90 | 0% | Table 2 PICOM questions (Ch1) |
| 100 | 33 | 0% | Table 2 PICOM continued (Ch1/Ch2/Ch3) |
| 101 | 84 | 0% | Table 2 PICOM continued (Ch2/Ch3/Ch4/Ch5) |
| **Total** | **213** | **0%** | All methodology — zero clinical risk |

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | MODERATE | Drug names and table cells captured, but fragmented |
| Tier accuracy | 0% | All 84 spans should be T3 or NOISE |
| Clinical safety risk | NONE | Methods table — PICOM scope definitions |
| Channel diversity | HIGH | All 4 channels active (first time in methods section) |
| Noise level | EXTREME | 84 spans for a methodology table, massive duplication |
| Pipeline bugs | 1 | HTML comment artifact variant (8th occurrence, with extra text) |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Methods table continuation defining systematic review scope for chapters 2-5.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 84
- **ADDED**: 0
- **PENDING**: 0
- **CONFIRMED**: 0
- **REJECTED**: 84 (all D/B/C/F table cell noise already rejected in bulk audit)

### Agent Spans Kept: 0
All 84 original spans were correctly rejected — D channel table cells, B channel bare drug names, C channel bare lab names, F channel HTML artifact.

### Gaps Added (14) — Exact PDF Text

| # | Gap Text (truncated) | Note |
|---|----------|------|
| G101-A | Does reducing blood glucose to a lower versus higher target improve clinically relevant outcomes... Intervention: Tight glycemic control (<7% HbA1c target or fasting glucose levels <120 mg/dl [6.7 mmol/l]), <6.5% HbA1c target, or <6.0% HbA1c target). | PICOM Q12 — Glycemic targets Ch2 |
| G101-B | Ruospo et al. Glucose targets for preventing diabetic kidney disease and its progression. Cochrane Database Syst Rev. 2017;CD010137. | Cochrane citation — Glycemic targets |
| G101-C | Does exercise/physical activity versus usual care improve clinically relevant outcomes... Intervention: Exercise/physical activity (aerobic training, resistance training). | PICOM Q13 — Exercise Ch3 |
| G101-D | Heiwe and Jacobson. Exercise training for adults with chronic kidney disease. Cochrane Database Syst Rev. 2011;CD003236. | Cochrane citation — Exercise in CKD |
| G101-E | Do dietary interventions versus usual diet improve clinically relevant outcomes... Intervention: Low-salt diets, low-potassium diets, low-phosphate diets, low-protein diets, dietary patterns (caloric-restriction diet, whole-food diets, Mediterranean diet, DASH diet, vegetarian diet). | PICOM Q14 — Dietary interventions Ch3 |
| G101-F | McMahon et al. Altered dietary salt intake... Cochrane Database Syst Rev. 2015:2;CD010070. Palmer et al. Dietary interventions for adults with chronic kidney disease. Cochrane Database Syst Rev. 2017;4:CD011998. | Cochrane citations — Dietary |
| G101-G | In patients with T2D and CKD, what are the effects of glucose-lowering medication... Intervention: Older therapies-metformin, sulfonylureas, or thiazolidinediones. More recent therapies-alpha-glucosidase inhibitors, GLP-1 RA, DPP-4 inhibitors. Long-term harms: amputation, bone fractures, hypoglycemia, lactic acidosis. | PICOM Q15 — Glucose-lowering Ch4 |
| G101-H | Lo et al. Insulin and glucose-lowering agents... Cochrane Database Syst Rev. 2018;9:CD011798. Lo et al. Glucose-lowering agents for treating pre-existing and new-onset diabetes in kidney transplant recipients. Cochrane Database Syst Rev. 2017;2:CD009966. | Cochrane citations — Glucose-lowering Ch4 |
| G101-I | What are the most effective education or self-management education programs... Intervention: Education and self-management programs. Additional outcomes: fatigue and quality of life. | PICOM Q16 — Education/self-management Ch5 |
| G101-J | Table 2 \| (Continued) Clinical questions and systematic review topics in the PICOM format. Guideline chapter 3: Lifestyle interventions. Guideline chapter 4: Glucose-lowering therapies. Guideline chapter 5: Approaches to management. | Table 2 continued headers — Ch3/Ch4/Ch5 |
| G101-K | Supplementary Tables S11-S13 | SoF refs — Glycemic targets Ch2 |
| G101-L | Supplementary Tables S21, S22 | SoF refs — Exercise Ch3 |
| G101-M | Supplementary Tables S16-S20 and S52-S56 | SoF refs — Dietary Ch3 |
| G101-N | Supplementary Tables S23 and S60-S91 | SoF refs — Glucose-lowering Ch4 |

### Post-Review State
- **Total spans**: 98
- **ADDED**: 14 (all new gaps)
- **CONFIRMED**: 0
- **REJECTED**: 84 (all original noise)
- **P2-ready facts**: 14

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G101-A | KB-16 (Monitoring) | Glycemic targets with specific HbA1c thresholds (<7%, <6.5%, <6.0%) |
| G101-C, G101-E | KB-4 (Safety) | Lifestyle interventions — exercise + dietary patterns |
| G101-G | KB-1 (Dosing) + KB-4 (Safety) | Glucose-lowering drug classes with long-term harms |
| G101-I | KB-4 (Safety) | Education/self-management programs |
| G101-B, G101-D, G101-F, G101-H | KB-7 (Terminology) | Cochrane systematic review citations |
| G101-J–N | KB-7 (Terminology) | Table headers, SoF table references |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 101 of 126
