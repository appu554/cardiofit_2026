# KB-15 Reference Page Extraction Pattern

## Purpose
Reusable extraction rules for KDIGO 2022 Diabetes & CKD guideline reference pages (118-128).
References are NOT clinical facts — they are **evidence metadata** for KB-15.

## Core Principle
**Extract as KB-15 evidence metadata, NOT as bibliography.**
Text must be **exact PDF citation** — no resentencing, no constructed citations from memory.

---

## Evidence Object Classification

### 1. Landmark Trial (extract individually)
**Criteria**: Pivotal RCT with named trial acronym, published in top-tier journal, directly referenced by KDIGO recommendations, has extractable effect sizes.

**What to capture**: Full citation + trial name + population + primary endpoint + key effect size in the note field.

**Target KBs**: KB-15 (always) + KB-1 (if dosing) + KB-4 (if safety) + KB-16 (if monitoring)

**Page 118 examples**: IRMA-2 (Ref 11), RENAAL (Ref 13), IDNT (Ref 34), Steno-2 (Refs 8-9), Rawshani (Ref 6), Ueki (Ref 7)

### 2. Canonical Cluster Summary (extract individually)
**Criteria**: Cochrane review, major meta-analysis, or systematic review that subsumes multiple individual RCTs in the reference list.

**What to capture**: Full citation + trial count + aggregate effect + list of subsumed ref numbers.

**Schema**: This becomes the `evidence_object_id` canonical node. Individual cluster members become `supporting_refs[]`.

**Target KBs**: KB-15 (always)

**Page 118 example**: Strippoli Cochrane (Ref 15) — subsumes refs 16-33, 35, 37-40

### 3. Evidence Cluster Object (synthesized)
**Criteria**: Group of homogeneous RCTs all covered by a single Cochrane/meta-analysis. These share drug class, population, and outcome.

**What to capture**: Synthesized cluster description listing all ref numbers and first-author/year for each. NOT an exact PDF citation (there is no single PDF passage for this).

**Schema**: Single weighted node. Not independently scored. `supporting_refs[]` for provenance tracing.

**Target KBs**: KB-15 (always)

**Page 118 example**: ACEi/ARB cluster (Refs 16-33, 35, 37-40) under Cochrane Ref 15

### 4. Cross-Guideline Linkage Node (extract individually)
**Criteria**: External guideline (ACC/AHA, ADA, ESC, NICE, etc.) referenced by KDIGO to inform its recommendations.

**What to capture**: Full citation. Note should specify the linkage: which external guideline → which KDIGO recommendation area.

**Target KBs**: KB-15 + KB-7 (Terminology)

**Page 118 examples**: ACC/AHA prevention (Ref 1), AHA/ACC cholesterol (Ref 5)

### 5. Meta-Recommendation Source (extract individually)
**Criteria**: Previous KDIGO version, KDIGO Controversies Conference, or other KDIGO consensus document that the current guideline updates or builds upon.

**What to capture**: Full citation. Note should specify the evolution: predecessor → current guideline relationship.

**Target KBs**: KB-15 + KB-7

**Page 118 example**: KDIGO 2016 Controversies Conference (Ref 4)

### 6. Skip (do not extract)
**Criteria**:
- Forward-looking reviews of future therapeutics (no current clinical application)
- Novel agents with low formulary relevance
- Conditional extracts (only relevant if CDS covers specific edge-case decisions)
- Smaller trials already subsumed by a Cochrane review (these go into the cluster object)

**Page 118 examples**: Breyer 2016 next-gen therapeutics (Ref 10), Jardine aspirin post-hoc (Ref 3), Ito imarikiren (Ref 29)

---

## Decision Flowchart

For each reference on a page:

```
Is it a Cochrane/meta-analysis?
├─ YES → Extract as Canonical Cluster Summary
│        List all subsumed ref numbers
│        Create companion Evidence Cluster Object
└─ NO ↓

Is it a landmark RCT (named trial, top-tier journal, effect sizes)?
├─ YES → Extract as Landmark Trial
│        Include trial name, population, endpoint, effect size in note
└─ NO ↓

Is it an external guideline (ACC/AHA, ADA, ESC, NICE)?
├─ YES → Extract as Cross-Guideline Linkage Node
│        Specify linkage direction in note
└─ NO ↓

Is it a previous KDIGO version or KDIGO consensus document?
├─ YES → Extract as Meta-Recommendation Source
│        Specify predecessor relationship in note
└─ NO ↓

Is it a member of a homogeneous trial corpus covered by a Cochrane review?
├─ YES → Include in Evidence Cluster Object (supporting_refs[])
│        Do NOT extract individually
└─ NO ↓

Is it forward-looking, low-relevance, or conditional?
├─ YES → SKIP (document reason in audit file "Refs Not Extracted" table)
└─ NO → Extract as Landmark Trial (default for unclear cases)
```

---

## API Pattern

```javascript
// Text: EXACT PDF citation text
// Note: gap ID + KB-15 evidence type + clinical significance + target KBs
await fetch(`/api/v2/pipeline1/jobs/${JOB_ID}/spans/add`, {
  method: 'POST', headers,
  body: JSON.stringify({
    reviewerId: 'claude-auditor',
    text: 'Author. Title. Journal. Year;Vol:Pages.',  // EXACT PDF
    note: 'G{page}-{letter} | KB-15 evidence object: {type} — Ref {n}, {description}. Target: {KBs}',
    pageNumber: pageNum
  })
});
```

---

## Naming Convention

- **Gap IDs**: `G{page}-{letter}` (e.g., G119-A, G119-B)
- **Replacement IDs**: `G{page}-{letter}2` if correcting a previous wrong entry
- **Evidence types in notes**: `Landmark trial`, `Canonical cluster summary`, `Cross-guideline linkage`, `Meta-recommendation source`, `Evidence cluster`

---

## Audit File Template for Reference Pages

```markdown
### Gaps Added (N) — KB-15 Evidence Objects

| # | Ref | Type | Note |
|---|-----|------|------|
| G{page}-A | {ref#} | {evidence type} | {clinical significance} |

### Refs Not Extracted
| Ref | Citation | Reason |
|-----|----------|--------|
| {ref#} | {author year} | {skip reason} |

### KB Routing
| Gap | Target KB | Rationale |
|-----|-----------|-----------|
```

---

## Pages Covered

| Page | PDF Page | Content | Status |
|------|----------|---------|--------|
| 118 | S117 | Refs 1-40 (ACEi/ARB, multifactorial, CV guidelines) | COMPLETE — 12 evidence objects |
| 119 | S118 | Refs 41-? | PENDING |
| 120 | S119 | Refs ?-? | PENDING |
| 121 | S120 | Refs ?-? | PENDING |
| 122 | S121 | Refs ?-? | PENDING |
| 123 | S122 | Refs ?-? | PENDING |
| 124 | S123 | Refs ?-? | PENDING |
| 125 | S124 | Refs ?-? | PENDING |
| 126 | S125 | Refs ?-? | PENDING |
| 127 | S126 | Refs ?-? | PENDING |
| 128 | S127 | Refs ?-? (final) | PENDING |

---

## KB-15 Schema Extension

Evidence objects require `evidence_object_id` field enabling:
- Multiple references → single weighted node (cluster collapse)
- Cochrane review as canonical `evidence_object`; individual trials as `supporting_refs[]`
- Landmark trials carry independent weight; cluster members do not
- Cross-guideline linkage nodes connect KDIGO to external guideline evidence

### Evidence Object Types
| Type | Weight | Schema |
|------|--------|--------|
| Landmark trial | Independent — carries own weight in Bayesian engine | `{ evidence_object_id, source_type: 'landmark_trial', citation, evidence_class, effect_size, population, outcome }` |
| Canonical cluster summary | Aggregated — carries weight of entire corpus | `{ evidence_object_id, source_type: 'cluster_summary', citation, supporting_refs[], trial_count, aggregate_effect }` |
| Cross-guideline linkage | Contextual — informs but doesn't independently weight | `{ evidence_object_id, source_type: 'cross_guideline', citation, linkage_from, linkage_to }` |
| Meta-recommendation source | Provenance — tracks guideline evolution | `{ evidence_object_id, source_type: 'meta_recommendation', citation, predecessor_of, update_scope }` |
| Evidence cluster | Collapsed — NOT independently weighted | `{ evidence_object_id, source_type: 'cluster', canonical_ref, supporting_refs[], aggregate_description }` |

---

*Created: 2026-03-01 | Applies to: KDIGO 2022 Diabetes & CKD guideline pages 118-128*
