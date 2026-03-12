# Page 97 Audit — Chapter 5: PP 5.2.1 Team-Based Integrated Care, Figure 35

## Page Identity
- **PDF page**: S96 (Chapter 5 — www.kidney-international.org)
- **Content**: PP 5.2.1 full text (team-based integrated care delivered by physicians and nonphysician personnel), structured care steps (register, assess, review, reinforce, recall), Figure 35 (team-based care schematic), research recommendation
- **Clinical tier**: Mixed T2/T3 — Practice point with actionable monitoring intervals (T2) + implementation guidance (T3)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 8 |
| T1 (Patient Safety) | 1 |
| T2 (Clinical Accuracy) | 7 |
| T3 (Informational) | 0 |
| Channels present | C (Grammar/Regex), F (NuExtract LLM) |
| Genuinely correct T1 | 0 |
| Genuinely correct T2 | 3 |
| Tier accuracy | 37.5% (3/8) |
| Disagreements | 1 |
| Review Status | FINAL: 15 ADDED, 0 PENDING, 8 REJECTED |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts confirmed (8 total: 1 T1, 7 T2), channels confirmed (C, F) |
| Raw PDF Cross-Check | 2026-02-28 — 0 agent spans kept, 8 rejected, 15 gaps total (5 bulk audit + 10 new) |

## Channel Breakdown
| Channel | Count | Confidence | Content |
|---------|-------|------------|---------|
| C (Grammar/Regex) | 6 | 85-98% | PP label + monitoring intervals + lab names |
| F (NuExtract LLM) | 1 | 85% | Decision-makers resource allocation prose |
| C+F (Multi-channel) | 1 | 97% | PP 5.2.1 body text (disagreement) |

## T1 Spans (1) — MISTIERED

| # | Text | Channel | Conf | Correct Tier | Issue |
|---|------|---------|------|-------------|-------|
| 1 | "Practice Point 5.2.1" | C | 98% | T3 | Practice point label only — body text captured separately |

**Pattern**: Fourth occurrence of C channel extracting recommendation/practice point labels as T1 (pages 90, 92, 94, 97). Same regex issue — matches section headers but not body content.

## T2 Spans (7)

| # | Text | Channel | Conf | Status | Correct Tier | Issue |
|---|------|---------|------|--------|-------------|-------|
| 1 | "Team–based integrated care, supported by decision–makers, should be delivered by physicians and nonphysician personnel (..." | C+F | 97% | PENDING | T3 | PP 5.2.1 body text — practice point guidance, not clinical threshold |
| 2 | "every 12–18 months" | C | 90% | PENDING | **T2** ✓ | Monitoring interval for comprehensive risk assessment |
| 3 | "hemoglobin" | C | 85% | PENDING | T3 | Bare lab name without clinical context |
| 4 | "every 2–3 months" | C | 90% | PENDING | **T2** ✓ | Monitoring interval for cardiometabolic risk factors |
| 5 | "eGFR" | C | 85% | PENDING | T3 | Bare lab name without clinical context |
| 6 | "every 3–12 months" | C | 90% | PENDING | **T2** ✓ | Monitoring interval for kidney function assessment |
| 7 | "Decision-makers allocate or redistribute resources, supported by appropriate policies, to facilitate the formation of a ..." | F | 85% | PENDING | T3 | Implementation/policy prose |

### Monitoring Interval Analysis — First Correct T2 in Chapter 5
Three C channel spans correctly capture monitoring intervals:
- **"every 12–18 months"** → Comprehensive risk assessment (blood/urine, eye/foot exam)
- **"every 2–3 months"** → Cardiometabolic risk factors (BP, HbA1c, body weight)
- **"every 3–12 months"** → Kidney function (eGFR and ACR)

These are genuine T2 content: specific monitoring frequencies that guide clinical practice. However, they're extracted without their clinical context (what to monitor at each interval), which reduces their standalone utility.

### C+F Multi-Channel Span
The PP 5.2.1 body text was captured by both C and F channels at 97% aggregate confidence with a disagreement flag. This is the first multi-channel span in Chapter 5 that captures meaningful clinical content (the practice point itself). However, it's a practice point about care delivery models — more T3 (informational/implementation) than T2 (clinical accuracy).

## PDF Source Content Analysis

### Content Present on Page
1. **Continuation from page 96** — Implementation gaps: patient motivation/adherence, system infrastructure, provider knowledge/skills, context-specific factors

2. **Practice Point 5.2.1** (full text):
   > "Team-based integrated care, supported by decision-makers, should be delivered by physicians and nonphysician personnel (e.g., trained nurses and dieticians, pharmacists, healthcare assistants, community workers, and peer supporters) preferably with knowledge of CKD (Figure 35)."

3. **Structured care delivery steps**:
   - Decision-makers: allocate resources, form multidisciplinary teams
   - Greater coordination among specialties (cardiology, endocrinology, nephrology, primary care)
   - Define care processes, re-engineer workflow with decision support

4. **Team-based structured care checklist**:
   - **Register**: Comprehensive risk assessment including blood/urine and eye/foot examination **every 12–18 months**
   - **Assess**: Cardiometabolic risk factors (BP, glycated hemoglobin, body weight) **every 2–3 months**
   - **Assess**: Kidney function (eGFR and ACR) **every 3–12 months**
   - **Review**: Treatment targets and organ-protective medications at each visit
   - **Reinforce**: Self-management (self-monitoring of BP, blood glucose, body weight)
   - **Recall**: Counseling on diet, exercise, self-monitoring with ongoing support

5. **Periodic audits**: Administrators should conduct system-level audits to identify care gaps

6. **Research recommendation**: Need for implementation research evaluating context-relevant team-based integrated care

7. **Figure 35**: Team-based integrated care schematic:
   - Register → Risk assessment → Risk stratification (1-8 steps)
   - Review: Risk factor control, organ-protective drugs, treat to multiple targets
   - Relay → Reinforce → Recall: Ongoing support for self-care
   - Uncoordinated care → Coordinated care = Empowered patients with optimal control

### What Should Have Been Extracted
| Content | Correct Tier | Extracted? |
|---------|-------------|------------|
| PP 5.2.1 full text | T3 | YES (C+F 97%, mistiered as T2) |
| "every 12–18 months" comprehensive risk assessment | T2 | **YES** ✓ (C 90%) |
| "every 2–3 months" cardiometabolic assessment | T2 | **YES** ✓ (C 90%) |
| "every 3–12 months" kidney function assessment | T2 | **YES** ✓ (C 90%) |
| "eGFR and ACR" kidney function markers | T2 | Partially (eGFR yes, ACR missed) |
| "glycated hemoglobin" as monitoring target | T3 | Partially ("hemoglobin" bare word) |
| "organ-protective medications at each visit" | T2 | NO |
| "self-monitoring of blood pressure, blood glucose, body weight" | T3 | NO |
| Figure 35 care model components | T3 | NO |
| Team composition (nurses, dieticians, pharmacists, etc.) | T3 | NO |
| Audit recommendation for care gap identification | T3 | NO |

## Cross-Page Patterns

### First Correct T2 Extractions in Chapter 5
Pages 90-96 had 0% tier accuracy across 379 spans. Page 97 breaks this streak with 3 genuinely correct T2 spans (monitoring intervals). The C channel's regex patterns for temporal expressions ("every X–Y months") successfully captured clinically meaningful monitoring frequencies.

### No F Channel HTML Artifact
Like page 96, page 97 does NOT have the `<!-- PAGE XX -->` F channel artifact. The artifact appeared on pages 91, 92, 94, 95 but not 96 or 97. This suggests the artifact is intermittent rather than universal.

### C Channel Dual Pattern
C channel shows two distinct extraction behaviors on this page:
1. **Temporal regex** (90% confidence): "every 12–18 months", "every 2–3 months", "every 3–12 months" — genuinely useful T2 extractions
2. **Lab name regex** (85% confidence): "hemoglobin", "eGFR" — bare terms without context (noise)

The temporal regex is more clinically valuable than the lab name regex.

### Chapter 5 Running Total (Pages 90-97)
| Page | Spans | Tier Accuracy | Genuine T2 | Notes |
|------|-------|--------------|------------|-------|
| 90 | 5 | 0% | 0 | Rec 5.1.1 label only |
| 91 | 8 | 0% | 0 | D "Usual care" ×4 + HTML artifact |
| 92 | 13 | 0% | 0 | 9 CI ranges + HTML artifact |
| 93 | 338 | 0% | 0 | Forest plot explosion |
| 94 | 3 | 0% | 0 | Rec 5.2.1 label + HTML artifact |
| 95 | 5 | 0% | 0 | F-only prose + HTML artifact |
| 96 | 7 | 0% | 0 | Drug names + prose |
| 97 | 8 | 37.5% | 3 | **Monitoring intervals correctly captured** |
| **Total** | **387** | **0.8%** | **3** | Chapter 5 improves slightly |

## Quality Assessment

| Dimension | Rating | Notes |
|-----------|--------|-------|
| Extraction completeness | MODERATE | Monitoring intervals captured, PP body captured, but context missing |
| Tier accuracy | 37.5% | 3/8 correct (monitoring intervals) — best in Chapter 5 |
| Clinical safety risk | NONE | Practice point guidance, monitoring recommendations, care model |
| Channel diversity | MODERATE | C + F, with one multi-channel C+F span |
| Noise level | MODERATE | 2 bare lab names + 1 label, but 3 useful intervals |
| Prior review status | 0/8 reviewed | No previous reviewer activity |

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Practice point guidance with correctly captured monitoring intervals. The 3 genuine T2 extractions are the best extraction quality in Chapter 5 so far.

---

## Raw PDF Cross-Check (2026-02-28)

### Pre-Review State
- **Total spans**: 8
- **ADDED**: 5 (from agent bulk audit: comprehensive risk assessment, cardiometabolic risk factors, kidney function, treatment targets, self-management)
- **PENDING**: 0
- **REJECTED**: 8 (PP label, truncated PP body, bare intervals ×3, bare lab names ×2, decision-makers prose fragment)

### Agent Spans Kept (5 — from earlier bulk audit)
| # | ID | Status | Text (truncated) | Note |
|---|-----|--------|----------|------|
| G97-A | `8e9305a1` | ADDED | Comprehensive risk assessment including blood/urine and eye/foot examination every 12-18 months | Register step — monitoring interval |
| G97-B | `b279c1e6` | ADDED | Cardiometabolic risk factors (BP, glycated hemoglobin, body weight) every 2-3 months | Assess step 1 — cardiometabolic interval |
| G97-C | `dd68c7a2` | ADDED | Kidney function (eGFR and ACR) every 3-12 months | Assess step 2 — kidney function interval |
| G97-D | `8ddb701a` | ADDED | Review treatment targets and organ-protective medications at each visit | Review step — treatment targets |
| G97-E | `89a3cf83` | ADDED | Self-management (self-monitoring of blood pressure, blood glucose, body weight) | Reinforce step — self-management |

### Gaps Added (10) — Exact PDF Text

| # | ID | Gap Text (truncated) | Note |
|---|-----|----------|------|
| G97-F | `b45c9258` | Practice Point 5.2.1: Team-based integrated care, supported by decision-makers, should be delivered by physicians and nonphysician personnel (e.g., trained nurses and dieticians, pharmacists, healthcare assistants, community workers, and peer supporters) preferably with knowledge of CKD (Figure 35). | PP 5.2.1 full text |
| G97-G | `79222e1d` | Decision-makers allocate or redistribute resources, supported by appropriate policies, to facilitate the formation of a multidisciplinary team including physicians and nonphysician personnel to deliver structured care in order to stratify risk, identify needs, and individualize targets and treatment strategies. | Decision-makers — team formation |
| G97-H | `835d5ce0` | Greater communication and more closely coordinated care among different specialties (e.g., cardiology, endocrinology, nephrology, primary care) and other allied health professionals should be a key pillar of this team-based integrated care. | Coordinated care — specialty coordination |
| G97-I | `ab52a3ad` | we emphasize that these recommendations and practice points should be viewed collectively as key components for general holistic management of patients with CKD and diabetes. | Holistic management emphasis |
| G97-J | `b5a86a2f` | Within team-based structured care, practitioners should define care processes and re-engineer workflow, supported by an information system with decision support, to deliver team-based structured care... | Care processes + workflow definition |
| G97-K | `72bfd2c7` | Provide counseling on diet, exercise, and self-monitoring with ongoing support, and recall defaulters at the clinic visit. | Recall step — counseling + defaulters |
| G97-L | `a33daf10` | Administrators or managers should conduct periodic audits on a system level to identify care gaps and provide feedback to practitioners with support to improve the quality of care. | Periodic audits — care gap identification |
| G97-M | `e805de3e` | There is a need for funding agencies to support implementation research or naturalistic experiments to evaluate context-relevant, team-based integrated care, taking into consideration local settings, cultures, and resources in order to inform practices and policies. | Research recommendation |
| G97-N | `9282ee26` | Figure 35 \| Team-based integrated care delivered by physicians and nonphysician personnel supported by decision-makers. BP, blood pressure. | Figure 35 caption |
| G97-O | `512aff57` | The relative importance of these factors is often context-specific and may vary among and within countries, as well as over time, depending on socioeconomic development and healthcare provision... and payment (social or private insurance) policies. | Implementation gaps — context variability |

### Post-Review State
- **Total spans**: 23
- **ADDED**: 15 (5 from bulk audit + 10 new gaps)
- **PENDING**: 0
- **REJECTED**: 8 (all original agent noise)
- **P2-ready facts**: 15

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G97-A–C | KB-16 (Monitoring) | Monitoring intervals: 12-18mo risk assessment, 2-3mo cardiometabolic, 3-12mo kidney function |
| G97-D, G97-E | KB-16 (Monitoring) | Treatment target review + self-management monitoring |
| G97-F | KB-4 (Safety) | PP 5.2.1 — care delivery model definition |
| G97-G, G97-H | KB-4 (Safety) | Decision-makers, multidisciplinary team, specialty coordination |
| G97-I, G97-J | KB-4 (Safety) | Holistic management, care process workflow |
| G97-K | KB-4 (Safety) | Counseling + defaulter recall |
| G97-L | KB-4 (Safety) | Periodic audits for care gap identification |
| G97-M | KB-4 (Safety) | Research recommendation — implementation research |
| G97-N | KB-7 (Terminology) | Figure 35 source attribution |
| G97-O | KB-4 (Safety) | Implementation gaps — context-specific factors |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-02-28 (claude-auditor via API)
- **Page in sequence**: 97 of 126
