# Page 118 Audit — References Start: Refs 1-40 (26 Spans)

## Page Identity
- **PDF page**: S117 (References — www.kidney-international.org)
- **Content**: References 1-40 — numbered bibliography beginning with Arnett 2019 (ACC/AHA primary prevention) through Muirhead 1999 (valsartan/captopril microalbuminuria)
- **Clinical tier**: T3 (Informational — bibliographic citations)
- **Job ID**: df538e50-0170-4ef8-862d-5b0a7c48e4ff
- **Disagreement flag**: YES

## Extraction Summary
| Metric | Value |
|--------|-------|
| Total spans | 26 |
| T1 (Patient Safety) | 17 |
| T2 (Clinical Accuracy) | 9 |
| T3 (Informational) | 0 |
| Channels present | B (Drug Dictionary), F (NuExtract LLM) |
| Multi-channel spans | 6 (B+F at 98%) |
| Tier accuracy | 0% (0/26) |
| **Accept button** | **DISABLED** — "17 Tier 1 (patient safety) spans must be reviewed before ACCEPT" |
| Disagreements | 6 |
| Review Status | FINAL: 12 ADDED, 0 CONFIRMED, 36 REJECTED |
| Raw PDF Cross-Check | 2026-03-01 (rev2) — 0 agent spans kept (26 rejected), 10 wrong-citation evidence objects rejected, 12 exact-PDF evidence objects added |
| Audit Date | 2026-02-25 (revised) |
| Cross-Check | 2026-02-25 — counts verified against raw extraction data |

## Span Breakdown

### T1 Spans (17) — All Should Be T3
| # | Text | Channel | Conf | Issue |
|---|------|---------|------|-------|
| 1 | "The effect of irbesartan on the development of diabetic nephropathy..." | B+F | 98% | Ref 11 title (Parving 2001) |
| 2 | "Prevention of transition from incipient to overt nephropathy with telmisartan..." | B+F | 98% | Ref 12 title (Makino 2007) |
| 3 | "Effects of losartan on renal and cardiovascular outcomes..." | B+F | 98% | Ref 13 title (Brenner 2001) |
| 4 | "enalapril" | B | 100% | Bare drug from ref 16 or 17 |
| 5 | "enalapril" | B | 100% | Duplicate — 2nd occurrence |
| 6 | "ACE inhibitor" | B | 100% | Drug class from ref 19 or 33 |
| 7 | "lisinopril" | B | 100% | From ref 24 or 26 |
| 8 | "ramipril" | B | 100% | From ref 25 or 36 |
| 9 | "lisinopril" | B | 100% | Duplicate — 2nd occurrence |
| 10 | "insulin" | B | 100% | From ref 26 or 29 |
| 11 | "insulin" | B | 100% | Duplicate — 2nd occurrence |
| 12 | "Renoprotective effect of the angiotensin-receptor antagonist irbesartan..." | B+F | 98% | Ref 34 title (Lewis 2001) |
| 13 | "ramipril" | B | 100% | Duplicate — 2nd occurrence |
| 14 | "Efficacy of captopril in postponing nephropathy..." | B+F | 98% | Ref 38 title (Mathiesen 1991) |
| 15 | "enalapril" | B | 100% | 3rd occurrence |
| 16 | "losartan" | B | 100% | From ref 39 |
| 17 | "The effects of valsartan and captopril on reducing microalbuminuria..." | B+F | 98% | Ref 40 title (Muirhead 1999) |

### T2 Spans (9) — All Should Be T3
| # | Text | Channel | Conf | Issue |
|---|------|---------|------|-------|
| 1 | "2016 ACC/AHA guideline focused update on duration of dual antiplatelet therapy..." | F | 85% | Ref 2 title |
| 2 | "Bakris GL, Barnhill BW, Sadler R. Treatment of arterial hypertension..." | F | 85% | Ref 18 citation |
| 3 | "Effects of captopril treatment versus placebo on renal function..." | F | 85% | Ref 21 title |
| 4 | "nifedipine" | B | 100% | From ref 24 or 30 |
| 5 | "perindopril" | B | 100% | From ref 30 |
| 6 | "nifedipine" | B | 100% | Duplicate — 2nd occurrence |
| 7 | "The effect of angiotensin-converting-enzyme inhibition on diabetic nephropathy." | F | 85% | Ref 33 title |
| 8 | "Marre M, Lievre M, Chatellier G, et al." | F | 85% | Ref 36 author list |
| 9 | "Mauer M, Zinman B, Gardiner R, et al." | F | 85% | Ref 39 author list |

### Key Pattern: B+F Multi-Channel on Reference Titles
6 spans have both B and F channels (98% confidence) — these are reference titles that contain drug names. The B channel detects the drug name, the F channel extracts the title text. The merger creates a high-confidence multi-channel span that gets classified as T1 because of the drug presence. These are the **most dangerous false-positives** — they look convincing but are just citation titles.

### B Channel Drug Extraction in References
The B drug dictionary is extracting every drug name mentioned in any reference title or citation: irbesartan, telmisartan, losartan, enalapril (×3), lisinopril (×2), ramipril (×2), insulin (×2), captopril, ACE inhibitor, nifedipine (×2), perindopril, valsartan. These are all from article titles like "The effect of irbesartan on..." — not clinical prescribing instructions.

### T1 Gate Active
The Accept button is disabled with the message: **"17 Tier 1 (patient safety) spans must be reviewed before ACCEPT"**. This confirms the Phase 2B tier-gated ACCEPT feature is working correctly. However, all 17 T1 spans are false-positives from reference titles.

## PDF Source Content
- **References 1-40**: ACEi/ARB clinical trial citations including landmark studies (IDNT, IRMA-2, RENAAL, EUCLID, DIABHYCAR, Collaborative Study Group)
- All content is standard numbered bibliography format: "Author. Title. Journal. Year;Vol:Pages."
- Drug names appear exclusively within article titles, not as clinical recommendations

## Decision Recommendation
**ACCEPT** — Zero clinical safety risk. Bibliographic references only. All T1 spans are false-positives from drug names in article titles.

---

## Raw PDF Cross-Check (2026-03-01)

### Pre-Review State
- **Total spans**: 26
- **ADDED**: 0
- **PENDING**: 26 (all original agent spans)
- **CONFIRMED**: 0
- **REJECTED**: 0

### Agent Spans Kept: 0
All 26 original agent spans rejected in 3 categories:
- **14 bare drug names** (B channel): enalapril ×3, lisinopril ×2, ramipril ×2, insulin ×2, nifedipine ×2, ACE inhibitor, perindopril, losartan — all extracted from reference titles without clinical context
- **2 partial author lists** (F channel): Marre et al., Mauer et al. — bibliographic fragments
- **10 reference titles** (B+F / F channel): IRMA-2, RENAAL, IDNT, Makino telmisartan, ACC/AHA antiplatelet, Bakris hypertension, captopril vs placebo, ACEi nephropathy, Mathiesen captopril, Muirhead valsartan — raw title spans replaced by structured KB-15 evidence objects

### Pipeline 2 L3-L5 Assessment
- **L3 (Claude fact extraction)**: Raw reference spans contain no extractable clinical facts (no thresholds, dosing, monitoring parameters)
- **L4 (RxNorm/LOINC/SNOMED)**: Drug names in reference titles are NOT clinical prescribing data — mapping would create false evidence chains
- **L5 (CQL schema)**: No CQL-compatible content in bibliographic citations
- **Target KBs (KB-1, KB-4, KB-16)**: Not directly applicable — references are provenance metadata, not clinical rules
- **KB-15 routing**: References restructured as evidence metadata objects (see gaps below)

### KB-15 Evidence Architecture Applied
Per KB-15 evidence metadata schema, references split into 3 categories:
1. **Landmark trials** — extracted individually with effect sizes, population, outcome annotations
2. **Cluster objects** — homogeneous trial corpora collapsed under canonical Cochrane review
3. **Cross-guideline linkage nodes** — external guidelines that inform KDIGO recommendations

### Round 1 Gaps (10) — REJECTED (wrong citations)
Initial 10 evidence objects added with constructed citations (not exact PDF text). All 10 rejected:
- G118-A: Ref 1 — wrong journal (said JACC, PDF says JAMA Cardiol) and wrong pages
- G118-B: Ref 5 — wrong journal (said JACC, PDF says Circulation) and wrong pages
- G118-C: Ref 6 — correct citation (kept concept, re-added as G118-D2)
- G118-D: Ref 11 — correct citation (kept concept, re-added as G118-G2)
- G118-E: Ref 13 — correct citation (kept concept, re-added as G118-H2)
- G118-F: Ref 34 — correct citation (kept concept, re-added as G118-K2)
- G118-G: Ref 15 — wrong journal/year (said BMJ 2004, PDF says Cochrane Database Syst Rev 2006)
- G118-H: Ref 4 — wrong author/title/year (said de Boer KDIGO 2020, PDF says Perkovic KDIGO 2016 Controversies)
- G118-I: Refs 8-9 — wrong year for Ref 9 (said NEJM 2008, PDF says NEJM 2003)
- G118-J: ACEi/ARB cluster — synthesized (concept preserved as G118-L2)

### Round 2 Gaps Added (12) — Exact PDF Text

| # | Ref | Evidence Object (exact PDF citation) | Type | Note |
|---|-----|--------------------------------------|------|------|
| G118-A2 | 1 | Arnett DK, Khera A, Blumenthal RS. 2019 ACC/AHA guideline... JAMA Cardiol. 2019;4:1043–1044. | Cross-guideline linkage | ACC/AHA primary prevention → KDIGO CV risk |
| G118-B2 | 4 | Perkovic V, Agarwal R, Fioretto P, et al. ...KDIGO Controversies Conference. Kidney Int. 2016;90:1175–1183. | Meta-recommendation source | KDIGO 2016 Controversies, predecessor to 2022 |
| G118-C2 | 5 | Grundy SM, Stone NJ, Bailey AL, et al. ...blood cholesterol... Circulation. 2019;139:e1082–e1143. | Cross-guideline linkage | AHA/ACC cholesterol → KDIGO CV risk in CKD |
| G118-D2 | 6 | Rawshani A, Rawshani A, Franzen S, et al. ...type 2 diabetes. N Engl J Med. 2018;379:633–644. | Landmark trial | T2DM risk factor priors, n=271K Swedish NDR |
| G118-E2 | 7 | Ueki K, Sasako T, Okazaki Y, et al. ...diabetic kidney disease... Kidney Int. 2021;99:256–266. | Landmark trial | Multifactorial intervention T2D kidney disease |
| G118-F2 | 8+9 | Gaede P, Oellgaard J... Diabetologia. 2016;59:2298–2307. // Gaede P, Vedel P... N Engl J Med. 2003;348:383–393. | Landmark trial | Steno-2 + 21yr follow-up, 7.9yr life gained |
| G118-G2 | 11 | Parving HH, Lehnert H... irbesartan... N Engl J Med. 2001;345:870–878. | Landmark trial | IRMA-2, RR 0.30 progression to overt nephropathy |
| G118-H2 | 13 | Brenner BM, Cooper ME, de Zeeuw D, et al. ...losartan... N Engl J Med. 2001;345:861–869. | Landmark trial | RENAAL, 16% risk reduction doubling SCr |
| G118-I2 | 14 | Keane WF, Brenner BM, de Zeeuw D, et al. ...RENAAL study. Kidney Int. 2003;63:1499–1507. | Landmark trial | RENAAL ESRD risk, feeds Bayesian risk engine |
| G118-J2 | 15 | Strippoli GF, Bonifati C, Craig M, et al. ...Cochrane Database Syst Rev. 2006;6:CD006257. | Canonical cluster summary | Cochrane ACEi/ARB, subsumes ~20 RCTs (refs 16-33, 35, 37-40) |
| G118-K2 | 34 | Lewis EJ, Hunsicker LG, Clarke WR, et al. ...irbesartan... N Engl J Med. 2001;345:851–860. | Landmark trial | IDNT, 20% risk reduction primary composite |
| G118-L2 | 16-33,35,37-40 | ACEi/ARB microalbuminuria trial cluster: ~20 RCTs subsumed by Cochrane ref 15 (Strippoli 2006). | Evidence cluster | Single weighted node, supporting_refs[] for provenance |

### Post-Review State (Final)
- **Total spans**: 48
- **ADDED**: 12 (exact PDF KB-15 evidence objects)
- **CONFIRMED**: 0
- **REJECTED**: 36 (26 original agent noise + 10 round-1 wrong-citation evidence objects)
- **P2-ready facts**: 12

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
| G118-A2, G118-C2 | KB-15 + KB-7 | Cross-guideline linkage — ACC/AHA prevention, AHA/ACC cholesterol |
| G118-B2 | KB-15 + KB-7 | Meta-recommendation source — KDIGO 2016 Controversies |
| G118-D2 | KB-15 | Rawshani T2DM risk factor priors — Bayesian engine input |
| G118-E2 | KB-15 + KB-4 | Ueki multifactorial — complements Steno-2 |
| G118-F2 | KB-15 + KB-4 | Steno-2 multifactorial — anchors comprehensive care recommendations |
| G118-G2, G118-H2, G118-K2 | KB-15 + KB-1 | Landmark ARB trials — IRMA-2, RENAAL, IDNT |
| G118-I2 | KB-15 + KB-1 + KB-4 | RENAAL ESRD risk — Bayesian risk engine |
| G118-J2 | KB-15 | Canonical Cochrane cluster — subsumes ACEi/ARB corpus |
| G118-L2 | KB-15 | ACEi/ARB cluster object — provenance tracing, not independently weighted |

### KB-15 Schema Extension Notes
Evidence objects require `evidence_object_id` field enabling:
- Multiple references → single weighted node (cluster collapse)
- Cochrane review as canonical `evidence_object`; individual trials as `supporting_refs[]`
- Landmark trials carry independent weight; cluster members do not
- Cross-guideline linkage nodes connect KDIGO to ACC/AHA, AHA/ACC external evidence

### Refs Not Extracted (by design)
| Ref | Citation | Reason |
|-----|----------|--------|
| 2 | Levine 2016, ACC/AHA antiplatelet | Not directly relevant to KDIGO DM+CKD core recommendations |
| 3 | Jardine 2010, aspirin post-hoc | Conditional — only if CDS covers antiplatelet in CKD |
| 10 | Breyer 2016, next-gen therapeutics | Forward-looking review, no current clinical application |
| 12 | Makino 2007, telmisartan | Smaller trial, not landmark; falls between individually-extracted refs |
| 29 | Ito 2019, imarikiren | Novel agent, low relevance for India formulary (included in cluster) |
| 36 | Marre/DIABHYCAR 2004 | Negative trial (low-dose ramipril), excluded from cluster range |

## Audit Metadata
- **Audited**: 2026-02-24
- **Method**: Playwright browser UI (Phase 2B Page Browse)
- **Auditor**: Claude (automated)
- **Raw PDF Cross-Check**: 2026-03-01 rev1 (claude-auditor via API) — 10 evidence objects, 5 with wrong citations
- **Raw PDF Cross-Check**: 2026-03-01 rev2 (claude-auditor via API) — all 10 rejected, 12 exact-PDF objects added
- **Page in sequence**: 118 of 126
