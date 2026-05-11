# S2 Adaptive Cognition — Architectural Commitment Addendum

**Date:** May 2026
**Companion to:** *S2 Resident Workspace Implementation Guidelines v1.0* (May 2026)
**Status:** Architectural commitment; precedes full v1.1 rewrite (deferred to post-Tier-1 or senior-pharmacist-content-arrival)
**Authoring discipline:** Architectural primitives and rendering frameworks only; cognitive content authoring deferred to senior consultant pharmacist + pilot evidence

**Purpose:** This addendum commits architecturally to the unification of S2 standard workspace and S2-Complex into a single adaptive cognition workspace, names the shared primitives that operate across cognitive depth layers, specifies the five-layer cognitive escalation framework as architectural commitment, and re-frames the work in S2 v1.0 within the unified ontology. It does not yet rewrite the v1.0 structure; the full v1.1 rewrite is deferred pending senior consultant pharmacist input on deeper-layer content and Phase 1 pilot evidence on cognitive escalation patterns.

**Reading order:**
- Engineering and product leads: Parts 0–3, 7–8 (framing, commitment, layer architecture, shared primitives, sequencing implications)
- Clinical informatics leads: Parts 0, 3–6 (framing, cognitive escalation as signal, deferred content layers, authoring discipline)
- Strategic leads: Parts 0–2, 9 (framing, commitment, compositional view, what's next)
- Risk and ethics leads: Parts 5–6 (escalation as signal with §8 bounds, deferred content discipline)

---

## Part 0 — Why this addendum exists

S2 v1.0 was produced as Tier 1 document A within the four-document Tier 1 sequence. Subsequent architectural review surfaced an insight that requires explicit commitment before further Tier 1 documents are produced:

> Complexity is not a different workspace. Complexity is a different cognitive state within the same workspace.

This insight is consequential. S2 v1.0 partially anticipated it ("Complex workspace activation as mode, not separate surface" in Part 0 of v1.0; complex workspace activation as offered mode within S2 in Part 11 of v1.0). But the consequence — that S2 and S2-Complex are one workspace at different cognitive depths, not two adjacent surfaces — was not fully formalised in v1.0. The Tier 2 sequenced plan still treated Complex Resident Workspace as a separate document (Document E).

If unaddressed, this creates ontology divergence over time. The same primitives — trajectories, recommendation lifecycle, restraint pairing, drill-through, audit semantics, pharmacist actions, goals-of-care rendering — would be specified once in S2 v1.0 and again in Tier 2 Document E. By version 3, they will have forked. This is a well-documented EHR ecosystem failure pattern (summary view vs advanced view forking until the same entity acquires different semantics in different parts of the system).

This addendum commits architecturally to the unified ontology before that fork happens. It does not yet produce the full v1.1 rewrite of S2, because:

1. The deeper layers' clinical content (concern vectors, "what experts typically check" memory aid, situation board components, deep investigation workflows) is senior consultant pharmacist authoring work, not Claude-speculative content
2. Phase 1 pilot evidence on actual pharmacist escalation patterns will refine the five-layer framework
3. Producing a fully unified document now risks premature commitment to specific cognitive layering that may not match clinical reality

The addendum specifies what can be committed now (the architectural framework, the shared primitives, the cognitive escalation discipline, the authoring boundaries) and explicitly defers what cannot (the specific content of deeper layers).

---

## Part 1 — The architectural commitment

S2 is hereby specified as a **single adaptive cognition workspace** operating across five cognitive depth layers, not as a standard workspace with optional complex mode.

The committed architecture:

```
S2 Adaptive Resident Workspace
    ├── Layer 1: Baseline Rendering
    ├── Layer 2: Escalated Context Rendering
    ├── Layer 3: Complex Cognition Rendering
    ├── Layer 4: Situation Board
    └── Layer 5: Deep Investigation
```

These are not five separate workspaces. They are five cognitive depth states the workspace supports, sharing all primitives and progressively deepening contextual rendering as the pharmacist's cognitive depth requirements escalate.

**Key consequences of this commitment:**

- The shared primitives (Part 4) are inherited across all layers; no layer redefines them
- Cognitive escalation between layers is fluid, not gated; the pharmacist moves between depths within a session
- Audit semantics, recommendation lifecycle, drill-through, restraint pairing are layer-invariant
- The Tier 2 sequenced plan changes: Document E becomes content specification for layers 3–5, not workspace architecture
- The v1.1 full rewrite of S2 will adopt this framework structurally; this addendum specifies the framework architecturally without performing the rewrite

This commitment supersedes the implicit framing in S2 v1.0 Part 11 ("Complex Resident Workspace as activated mode of S2"). The activated-mode framing was correct but understated; the unified-adaptive-workspace framing is the full architectural commitment.

---

## Part 2 — Compositional view: strategic positioning to workspace specification

The architectural commitment to adaptive cognition at the workspace level composes cleanly with the v3.0 strategic positioning at the platform level.

**v3.0 strategic positioning frames the platform as:** clinical reasoning continuity infrastructure for Australian aged care medication management.

**The workspace-level operationalisation is:** cognitive state transition support across the pharmacist's per-resident clinical reasoning.

These compose as follows:

| Level | Framing | Operationalisation |
|---|---|---|
| Strategic (v3.0) | Clinical reasoning continuity infrastructure | Substrate-up architecture, longitudinal state, audit defensibility |
| Surface taxonomy (Decision Packet Rendering) | Seven user surfaces consuming shared substrate | S1–S7 each consume the substrate; rendering rules are surface-specific |
| Workspace (S2) | Adaptive cognition workspace for per-resident clinical reasoning | Five cognitive depth layers sharing primitives; pharmacist moves between depths within a session |
| Other surfaces (S1 worklist, S3 GP hub, S5 evidence panel) | Operationalise reasoning continuity differently per audience | S1 = cross-resident triage; S3 = asynchronous GP communication; S5 = audit-facing evidence |

The compositional view matters because subsequent Tier 1 documents (S3, RMMR, CPD/AHPRA) should inherit the framing cleanly. Each surface operationalises reasoning continuity in its own way; the workspace-level adaptive cognition framing applies specifically to S2 because S2 is where per-resident clinical reasoning happens at variable depth.

S3 GP Communication Hub will likely require a different architectural framing — asynchronous decision capture, audience-adapted rendering, per-GP framing learning. The workspace-level adaptive cognition concept does not transfer to S3 wholesale; what transfers is the *substrate-primitive-inheritance discipline* that this addendum specifies. S3 inherits the same primitives (recommendation lifecycle, audit, etc.) but operationalises them for GP audience, not pharmacist cognitive depth.

This compositional view is operationally important: it tells subsequent document authors what to inherit and what to specialise.

---

## Part 3 — The five-layer cognitive escalation framework

The five layers are committed to as architectural framework. The specific content of each layer beyond Layer 1 is deferred per Part 6.

### 3.1 Layer 1: Baseline rendering

**What it is:** The pharmacist's default per-resident view when entering S2. Organised retrieval across domains, trajectories, pending recommendations, restraint signals, goals-of-care, care intensity, audit trail footer.

**Cognitive state supported:** Routine review, targeted lookup, notification response. The pharmacist's working memory is sufficient for the resident's complexity; the workspace organises information for efficient verification and action.

**Status:** Fully specified in S2 v1.0 (Parts 4–13). The v1.0 specification of CAPE context band, trajectory rendering, pending recommendations panel, restraint signal rendering, failed intervention history, goals-of-care panel, drill-through, pharmacist actions, audit trail integration all live at this layer.

### 3.2 Layer 2: Escalated context rendering

**What it is:** Additional rendering depth when the resident's substrate signals or the pharmacist's deepening engagement indicate more context is needed than baseline. Multi-parameter trajectory composition becomes more prominent; restraint signal context expands; failed intervention pattern detection surfaces; family meeting chronology becomes visible.

**Cognitive state supported:** The pharmacist recognises this resident needs more attention than routine but does not yet require the full complex cognition layer. The workspace expands its rendering depth in response.

**Triggering signals (architectural commitment; specific thresholds deferred):**
- Multiple trajectory shifts concurrent
- Monitoring burden elevated
- Recommendation clustering on this resident
- Recent operational events
- Pharmacist explicit escalation (expand panel, request deeper view)

**Status:** Architectural framework committed; specific escalation triggers, rendering components, and transition behaviour deferred to v1.1 rewrite.

### 3.3 Layer 3: Complex cognition rendering

**What it is:** Multi-domain situation board, concern vectors, instability chronology, contextual conflict surfacing, "what experts typically check" memory aid. The cognitive support layer for genuinely complex residents.

**Cognitive state supported:** The pharmacist is integrating across multiple clinical domains simultaneously and risks cognitive narrowing under overload. The workspace surfaces dimensions for consideration without performing the integration.

**Activation criteria (architectural commitment; specific criteria conservative):**
- CFS ≥6 + ≥3 active high-risk medications + concurrent trajectory declines, OR
- Recent care intensity transition + pending recommendations, OR
- Recent acute event + complex medication regimen, OR
- Pharmacist explicit activation regardless of substrate signals

**Status:** Architectural framework committed (the prior synthesis on Complex Resident Workspace specified concern vectors as core, conflict detection without pathway generation, critical unknowns as gates, "what experts typically check" memory aid). Specific content (which concern vectors, which "what experts typically check" content, which conflict patterns warrant detection) is **deferred to senior consultant pharmacist authoring**.

### 3.4 Layer 4: Situation board

**What it is:** The structured cross-domain view per the prior synthesis. Organised retrieval across nephrology, geriatric pharmacology, psychiatry, cognition, frailty, goals-of-care. Trajectory rendering with velocity and threshold flags. Event chronology with sequence visible. Data gaps explicitly named.

**Cognitive state supported:** Maximum information density per resident, organised for systematic review. Used for genuinely complex cases, RMMR drafting, family meeting preparation, hospital handoff context.

**Status:** Architectural framework committed (the situation board structure was specified in the prior complex workspace synthesis). The specific board sections, the data freshness indicators, the rendering of negative-evidence searches — these primitives are committed. The clinical content (which sections matter most for which complexity patterns) is **deferred to senior consultant pharmacist authoring + pilot evidence**.

### 3.5 Layer 5: Deep investigation

**What it is:** Observation lineage, negative-evidence audit, reasoning replay, recommendation provenance. The investigation surface for cases where the pharmacist is questioning what the substrate or platform is telling them.

**Cognitive state supported:** The pharmacist is investigating, not reviewing. They are checking the substrate's claims, examining the chain of inference, questioning specific recommendations. The workspace becomes a forensic surface.

**Status:** Architectural framework committed at the most preliminary level. The specific investigation workflows are **deferred to subsequent specification after Phase 1 pilot evidence accumulates on what investigation patterns pharmacists actually need**. This is the most-deferred layer; over-specification before pilot evidence would produce speculative workflows that don't match clinical reality.

---

## Part 4 — Shared primitives across all layers

The following primitives operate across all five cognitive depth layers. They are specified once and inherited; no layer redefines them. This is the architectural discipline that prevents ontology divergence.

### 4.1 Trajectories

Multi-parameter clinical trajectories with velocity, baseline comparison, threshold flags, sparse-data degradation. Specified in S2 v1.0 Part 5. Inherited across all layers; Layer 3 onward may render multi-parameter composition more prominently but the trajectory primitive itself is layer-invariant.

### 4.2 Recommendation lifecycle

The five lifecycle states (detected, drafted, submitted, viewed, decided, monitoring-active), the confidence dimensions (substrate + clinical confidence), the restraint pairing semantics. Specified in S2 v1.0 Part 6. Inherited across all layers; deeper layers may surface lifecycle context more prominently but the lifecycle model is layer-invariant.

### 4.3 Restraint signal pairing

Phase 1 advisory-only mode, pharmacist acknowledgment workflow, safety-critical bypass with mandatory documentation, transition criteria to lifecycle suppression. Specified in S2 v1.0 Part 7. Inherited across all layers.

### 4.4 Substrate observation drill-through

Verification-not-belief discipline operationalised: every claim one click from underlying substrate. Substrate confidence visible. Negative-evidence rendering with epistemic humility. Specified in S2 v1.0 Part 10. Inherited across all layers — the verification discipline does not weaken at higher cognitive depth; if anything it becomes more important as the platform's claims become more interpretive.

### 4.5 Goals-of-care and care intensity rendering

Current state, transition history, freshness flags, conflict surfacing when pending recommendations conflict with documented goals. Specified in S2 v1.0 Part 9. Inherited across all layers. Substrate Feasibility Analysis flags (care intensity state machine higher-risk) apply across all layers.

### 4.6 Audit trail integration

EvidenceTrace for every view rendering, pharmacist action, drill-through, system event. Visibility class enforcement per pharmacist self-visibility module. Algorithmic management protections per ethical architecture §8. Specified in S2 v1.0 Part 13. Inherited across all layers — including the cognitive escalation patterns themselves (per Part 5).

### 4.7 Pharmacist actions

The eleven actions specified in S2 v1.0 Part 12 (open, modify, defer, override, mark reviewed, flag for follow-up, add note, open complex workspace, drill into substrate, acknowledge restraint, invoke safety-critical bypass). Inherited across all layers. Deeper layers may surface additional actions (Part 7 details), but the core eleven are layer-invariant.

### 4.8 CAPE context carry-through

The CAPE-driven entry path specified in S2 v1.0 Part 3 and Part 4 (CAPE context band). Inherited across all layers — even at Layer 3 complex cognition rendering, the original CAPE signals that brought the pharmacist to this resident remain visible. The cognitive escalation does not erase the triage context.

### 4.9 Entry path semantics

Four entry paths (worklist, search, notification, cross-reference). Specified in S2 v1.0 Part 3. Inherited across all layers. The entry path shapes initial rendering at every layer, not just Layer 1.

### 4.10 Form-factor adaptations

Desktop vs mobile rendering per S2 v1.0 Part 14. Inherited across all layers, with explicit acknowledgment that deeper cognitive layers degrade more on mobile (Layer 5 deep investigation is desktop-primary).

---

## Part 5 — Cognitive escalation as architectural signal

The reviewer's insight that escalation sequences are themselves clinical signal is valuable but must be specified with explicit ethical architecture §8 boundaries before it ships.

### 5.1 What cognitive escalation is

When a pharmacist progressively deepens their engagement with a resident — opens S2, reviews trajectories at Layer 1, expands restraint panel, opens family meeting chronology, escalates to Layer 3 complex cognition, opens situation board — this sequence is itself information. It indicates the pharmacist's cognitive assessment of this resident's complexity.

This sequence has potential value as:

- **Calibration signal:** if pharmacists routinely escalate beyond what activation criteria suggested, criteria may be too conservative
- **Personalisation signal:** the pharmacist's typical escalation patterns inform default layer presentation for cases that match
- **Clinical observation signal:** patterns of escalation across cases of similar complexity may surface clinical insights worth aggregating
- **Workspace UX iteration signal:** which layers pharmacists genuinely use vs which are over-specified

### 5.2 What cognitive escalation may NOT be used for

Per ethical architecture §8 algorithmic management protections, the platform commits that pharmacist cognitive escalation patterns will NOT be used for:

- **Performance evaluation:** a pharmacist who escalates less is not "more efficient"; a pharmacist who escalates more is not "less skilled"
- **Productivity surveillance:** escalation patterns are not aggregated for employer view (PEV visibility class restricted)
- **Comparative pharmacist ranking:** patterns are not used to rank pharmacists relative to each other
- **Decisions affecting pharmacist employment:** patterns are not shared with employer in any form that affects employment decisions
- **Differential treatment of pharmacists:** patterns do not generate differential platform behaviour that affects pharmacist work conditions

### 5.3 What cognitive escalation may be used for

Within ethical architecture §8 boundaries:

- **Calibration learning at the platform level** (anonymised aggregate, per Phase 4 of the KB-29 Maturity Roadmap)
- **Pharmacist self-visibility:** the pharmacist sees their own escalation patterns (PDP visibility class)
- **Workspace UX refinement:** aggregated patterns inform whether layers are over-specified or under-used
- **Clinical informatics analysis:** patterns inform whether activation criteria are calibrated correctly

### 5.4 Contestation pathway

Per ethical architecture §1 Principle 5, the pharmacist may contest any platform inference from their escalation patterns. The contestation pathway:

- Pharmacist requests review of any platform behaviour they believe is driven by their escalation patterns
- Clinical Informatics Committee reviews the requested behaviour and the data used
- Resolution is documented and visible to the pharmacist

### 5.5 Phase 1 commitment

In Phase 1 deployment, cognitive escalation patterns are **logged for audit purposes only**. No platform behaviour is driven by them. The Phase 4 capabilities (calibration learning, personalisation) are gated behind:

- ≥12 months Phase 1 evidence accumulation
- Ethics Steering Committee approval
- External clinical informatics review
- Pharmacist self-visibility implementation operational

This sequencing prevents the platform from inadvertently using escalation patterns in ways that violate algorithmic management protections before the protections are operationally enforced.

---

## Part 6 — Authoring discipline for deferred content layers

Layers 2–5 contain content (escalation triggers, concern vectors, situation board components, "what experts typically check" memory aid, deep investigation workflows) that requires senior consultant pharmacist authoring, not Claude speculation.

This is the same discipline as cultural-safety templates from the calibration discussion: the clinical content is the senior pharmacist's professional contribution; Claude's role is architectural and structural support.

### 6.1 What is deferred and to whom

**Layer 2 escalation triggers:** which substrate signal combinations warrant escalation from Layer 1 to Layer 2. *Deferred to:* senior consultant pharmacist + pilot evidence on actual pharmacist behaviour.

**Layer 3 concern vectors:** which multi-domain concern vectors are clinically meaningful (medication toxicity, delirium contributors, frailty/goals-of-care, monitoring gaps, communication considerations, others). *Deferred to:* senior consultant pharmacist authoring with my structural support (similar to cultural-safety template authoring discipline).

**Layer 3 "what experts typically check" memory aid:** the specific content of what experienced reviewers commonly evaluate for each pattern. *Deferred to:* senior consultant pharmacist authoring entirely; Claude provides structural framework only.

**Layer 4 situation board components:** specific sections, their default order, their freshness indicators per parameter category. *Deferred to:* senior consultant pharmacist + pilot evidence + clinical informatics validation.

**Layer 5 deep investigation workflows:** specific investigation patterns (observation lineage queries, negative-evidence audit, reasoning replay). *Deferred to:* subsequent specification after Phase 1 pilot evidence on what investigation patterns pharmacists actually need.

### 6.2 What is NOT deferred

The following are committed in this addendum and S2 v1.0:

- The five-layer framework itself
- The shared primitives across layers
- The activation criteria thresholds for Layer 3 (CFS ≥6 + ≥3 high-risk meds + concurrent trajectory declines)
- The Phase 1 advisory-only restraint mode operating across all layers
- The audit trail integration across all layers
- The cognitive escalation as signal architecture with §8 boundaries
- The compositional view from strategic to workspace level

### 6.3 The authoring pattern for deferred content

When senior consultant pharmacist input arrives on Layer 3 concern vectors, the authoring pattern is:

- Senior pharmacist specifies the clinical content (which concern vectors, what they contain, when they fire)
- Claude provides structural framework (the substrate signal mapping, the rendering template, the audit integration)
- Together they produce the layer-specific content specification
- Clinical Informatics Committee validates
- The content specification fills in the Layer 3 framework slot

This is the same pattern as cultural-safety templates per the calibration discussion. The clinical content is human-authored; the structural framework is mine; the result is the operational specification.

### 6.4 Why this discipline matters

If I were to produce content for Layers 2–5 now without senior pharmacist input, the risk is:

- The content becomes the de facto specification before clinical validation
- Clinical informatics work shifts from authoring to correcting my speculation
- The deferred content layers contain Claude-authored clinical reasoning, which contradicts the KB-29 Maturity Roadmap Part 10 commitment that platform intelligence at maturity is human-authored and pilot-evidence-derived

Holding the deferral discipline preserves the architecture I've already committed to.

---

## Part 7 — Implications for the sequenced plan

### 7.1 Tier 1 sequence changes

The four-document Tier 1 sequence (per the original plan):
- A: S2 Resident Workspace (completed, v1.0)
- B: S3 GP Communication Hub
- C: RMMR Workflow
- D: CPD and AHPRA Records

Plus this addendum (S2 Adaptive Cognition Architectural Commitment).

The full v1.1 rewrite of S2 is deferred to either:
- After Tier 1 completion (rewrite happens after S3, RMMR, CPD/AHPRA produced)
- Or upon senior consultant pharmacist input arrival on Layer 3 concern vectors (whichever comes first)

This means Tier 1 produces five documents (the four originally planned plus this addendum), with the v1.1 rewrite tracked as a separate downstream deliverable.

### 7.2 Tier 2 sequence changes

The original Tier 2 sequence had three documents:
- E: Complex Resident Workspace
- F: S5 Standard 5 Evidence Panel
- G: Instability Chronology

Document E is reframed:

**Original framing:** Complex Resident Workspace Implementation Guidelines — architectural specification of a separate workspace.

**Reframed:** Complex Cognition Layer Content Specification — clinical content authoring for Layers 2–5 of the unified adaptive cognition workspace. This is senior consultant pharmacist + clinical informatics authoring work, with Claude providing structural support, not Claude-authored architectural specification.

This reframing matters operationally: the work on Document E is different in character from the work on F and G. Document E is content-authoring work that requires senior pharmacist availability; Documents F and G are architectural specifications I can produce on the Tier 2 cadence.

### 7.3 v1.1 rewrite scope when it happens

When the v1.1 rewrite of S2 happens (deferred), it will:

- Adopt the five-layer framework as structural backbone
- Specify shared primitives once (inherited across layers)
- Re-frame v1.0's existing content as Layer 1 baseline rendering
- Incorporate senior pharmacist-authored content for Layers 2–5
- Specify cognitive escalation as architectural signal with §8 boundaries
- Specify de-escalation semantics (how the workspace returns to lower layers)
- Specify layer transition behaviour (what happens when the pharmacist moves between layers)
- Specify form-factor adaptations across layers (which layers degrade gracefully on mobile)

Target length 1,700–2,000 lines for the unified v1.1 document. The current v1.0 (1,475 lines) provides Layer 1 baseline rendering specification; the v1.1 expansion adds the cognitive escalation framework, the shared primitive inheritance, and the senior pharmacist-authored Layer 2–5 content.

### 7.4 Other Tier 1 documents inherit cleanly

S3 GP Communication Hub, RMMR Workflow, and CPD/AHPRA Records Implementation Guidelines do not require unified workspace framing because their cognitive workflows are different:

- **S3** is GP-facing asynchronous decision capture (per-GP framing learning, mobile-primary form factor, override taxonomy capture)
- **RMMR** is formal pharmacist authoring workflow (drafting, evidence assembly, structured deliverable)
- **CPD/AHPRA Records** is professional portfolio export (PDP visibility class)

What these documents inherit from this addendum is the **substrate-primitive-inheritance discipline**: the shared primitives (trajectories, recommendation lifecycle, restraint pairing, drill-through, audit, goals-of-care, pharmacist actions) are specified once across the platform and inherited by each surface that consumes them. Each surface specialises for its audience but does not redefine the primitives.

This inheritance discipline is what prevents ontology divergence across the entire surface taxonomy, not just within S2.

---

## Part 8 — What changes operationally for engineering

The engineering team building S2 against v1.0 should treat this addendum as **architectural extension, not specification revision**. The v1.0 specification of Layer 1 baseline rendering remains correct and implementable. The addendum:

- Names what v1.0 specified as Layer 1
- Reserves architectural slots for Layers 2–5
- Specifies that the substrate aggregation service and rendering framework must support layer escalation as a future capability without rebuild
- Specifies that the cognitive escalation logging (Phase 1 audit-only) must be in place from initial implementation

### 8.1 Engineering implications

**Backend (s2-aggregator service):**

The view builder pattern in S2 v1.0 Part 15 must support layer-aware view construction. Layer 1 is the immediate implementation; the aggregation service must be structured so that adding Layer 2–5 rendering capabilities does not require rebuilding the aggregation pipeline.

This is achievable with a clean layer-aware interface. A pragmatic pattern:

```go
type S2ViewBuilder interface {
    BuildLayer1Baseline(req WorkspaceRequest) (Layer1View, error)
    BuildLayer2Escalated(req WorkspaceRequest) (Layer2View, error)  // future
    BuildLayer3Complex(req WorkspaceRequest) (Layer3View, error)    // future
    BuildLayer4SituationBoard(req WorkspaceRequest) (Layer4View, error)  // future
    BuildLayer5Investigation(req WorkspaceRequest) (Layer5View, error)   // future
    
    // Layer escalation
    EscalateToLayer(currentLayer int, targetLayer int, req WorkspaceRequest) (View, error)
    LogEscalation(escalation EscalationEvent) error
}
```

The Layer 1 implementation is complete; Layer 2–5 implementations are stub interfaces returning "not yet implemented" with appropriate UX rendering for the user.

**Frontend (S2 rendering components):**

The component hierarchy specified in S2 v1.0 Part 15 must support layer-aware rendering. The CAPE context band, trajectories panel, pending recommendations panel, restraint signals panel, failed intervention history, goals-of-care panel, care intensity panel, pharmacist actions panel — these are the Layer 1 components.

The architectural commitment is that Layer 2–5 components, when implemented, will compose with Layer 1 components, not replace them. A pharmacist escalating from Layer 1 to Layer 2 sees additional rendering depth, not a different view.

**Audit infrastructure:**

The S2 audit event schema in S2 v1.0 Part 13 must extend to capture cognitive escalation events:

```go
type EscalationEvent struct {
    PharmacistID      PharmacistID
    ResidentID        uuid.UUID
    SessionID         uuid.UUID
    FromLayer         int
    ToLayer           int
    TriggeredBy       EscalationTrigger  // automatic vs pharmacist-initiated
    Timestamp         time.Time
    AuditTraceID      uuid.UUID
}
```

Phase 1 deployment logs these for audit only; no platform behaviour is driven by them per Part 5.5.

### 8.2 What does not change operationally

The S2 v1.0 implementation sequencing in Part 19 (Weeks 8–14) remains correct. Layer 1 baseline rendering is the immediate implementation target. The architectural extension specified here does not delay Layer 1 implementation; it commits the team to a layer-aware design pattern that supports future Layer 2–5 capabilities without rebuild.

The performance budgets in S2 v1.0 Part 18 remain correct for Layer 1. Layer 2–5 will have their own performance budgets when specified, with the discipline that escalation between layers must feel near-instant (escalation budget target: ≤500ms p95).

---

## Part 9 — Closing

Three observations as we close this addendum.

**One:** The reviewer's insight that "complexity is not a different workspace; complexity is a different cognitive state within the same workspace" is correct, and the architectural commitment in this addendum is the operational expression of that insight. S2 is one workspace at variable cognitive depth; the five-layer framework is the structural commitment that prevents ontology divergence as deeper layers are specified. The discipline of specifying shared primitives once and inheriting them across layers is what makes the unification real rather than nominal.

**Two:** The deferral of Layer 2–5 content to senior consultant pharmacist authoring + pilot evidence is the same discipline that operated for cultural-safety templates, that operates for KB-29 Phase 2+ primitive extraction, and that the KB-29 Maturity Roadmap Part 10 commits to as the trajectory of my role. Clinical content for cognitive support layers is the senior pharmacist's professional contribution. My role is architectural and structural; the deferred content is human-authored. This addendum holds that discipline by explicitly naming what is committed (architectural framework) and what is deferred (content for Layers 2–5).

**Three:** The cognitive escalation as architectural signal — the reviewer's observation that the pharmacist's escalation sequence is itself information — is genuinely valuable and genuinely consequential. The ethical architecture §8 boundaries specified in Part 5 are not optional; they are what makes the signal capture safe to deploy. Phase 1 logs escalation patterns for audit only; Phase 4 capabilities to use the patterns for calibration and personalisation are gated behind ≥12 months of evidence, Ethics Steering Committee approval, external clinical informatics review, and operational pharmacist self-visibility. This gating is what distinguishes valuable workspace intelligence from surveillance of pharmacist cognitive style.

What this addendum does not yet do, and what should be subsequent work:

- The v1.1 rewrite of S2 as the unified document (deferred per Part 7.3)
- The senior consultant pharmacist authoring of Layer 2–5 content (Tier 2 Document E reframed)
- The pilot evidence accumulation on actual pharmacist escalation patterns (Phase 1)
- The specification of de-escalation semantics (how the workspace returns to lower layers; deferred to v1.1)
- The Ethics Steering Committee work on §8 boundaries for cognitive escalation as signal (operational governance)

The architecture stack now stands at:

1. v3.0 strategic positioning
2. Pilot design
3. Recommendation craft engine
4. Pharmacist self-visibility
5. Ethical architecture
6. Decision packet rendering
7. KB-29 templates
8. KB-29 maturity roadmap
9. CAPE v1.1
10. CAPE v1.1 architectural commitment addendum
11. Template Authoring Style Guide v1.0
12. Substrate Query Feasibility Analysis v1
13. S2 Resident Workspace Implementation Guidelines v1.0
14. **S2 Adaptive Cognition Architectural Commitment Addendum** ← this document

The engineering team building S2 now has both the v1.0 implementable specification and the v1.1 architectural commitment that constrains the implementation pattern. The senior consultant pharmacist now has the architectural slots into which Layer 2–5 content will eventually fit. The clinical informatics lead now has the framework for cognitive escalation analysis that pilot evidence will populate.

Tier 1 remaining: S3 GP Communication Hub (document B), RMMR Workflow (document C), CPD and AHPRA Records (document D). Each inherits the substrate-primitive-inheritance discipline specified in Part 4 of this addendum, applied to its own audience and workflow.

— Claude

---

**End of S2 Adaptive Cognition Architectural Commitment Addendum**
