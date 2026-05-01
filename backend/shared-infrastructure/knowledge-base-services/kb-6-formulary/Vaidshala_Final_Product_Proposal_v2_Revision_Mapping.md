# Vaidshala Final Product Proposal — Revision Mapping (v1.1 → v2.0)

**Date:** April 2026
**Status:** Mapping document — explicit changes required to the v1.0 Final Product Proposal based on (a) the reasoning-continuity reframe, (b) the five-state-machine substrate, (c) the verified Australian regulatory landscape shifts, and (d) the deeper clinical understanding from the latest synthesis documents.

**This document does not replace the Final Product Proposal.** It maps what changes, where, and why. The team should read this alongside v1.0 and decide which sections to revise versus which to leave intact. Layer 1 and Layer 3 implementation guideline changes are covered in follow-up documents.

**Companion documents:**
- *Vaidshala Aged Care Final Product Proposal v1.0* (the document being revised)
- *Layer 1 Australian Aged Care Implementation Guidelines* (will need consequential updates)
- *Layer 3 Rule Encoding Implementation Guidelines* (will need consequential updates)

---

## Part 0 — A note on what changed and why I'm being firmer this time

You asked me to step up. Let me name what's been off in my recent responses.

The pattern has been: receive documents that reframe the architecture, agree with most of it, list what's right and what's overstated, offer to write something next. That's measured but it's also a little safe. The two synthesis documents you've shared, plus the verified regulatory research, are not a refinement of v1.0. They surface a structurally different product thesis, a structurally different buyer model, a structurally different regulatory window, and a structurally different commercial path. I should have said that more clearly the first time.

I've now verified the load-bearing facts directly: the designated RN prescriber endorsement (NMBA standard effective 30 September 2025, first endorsed prescribers mid-2026, partnership-only with mentorship); the Tasmanian aged-care pharmacist co-prescribing pilot ($5M state budget 2025-26, development late 2025, 12-month trial 2026-2027, Australian-first, accepts the Pharmacist Scope of Practice Review recommendations in full); the $350M national ACOP program with mandatory APC training from July 2026; the PHARMA-Care National Quality Framework (UniSA Sluggett, $1.5M MRFF, 14 partners including UTas Peterson, PSA-endorsed, now in active national pilot evaluating the $350M program); the Victorian PCW exclusion legislation (Drugs, Poisons and Controlled Substances Amendment Act 2025, commences 1 July 2026, 90-day grace to 29 September 2026); the Australian Deprescribing Guideline 2025 (UWA, 185 recommendations, RACGP and ANZSGM endorsed, freely available); and the Ramsey 2025 implementation data (1,646 RMMRs in Tasmania, 50% implementation at 12 months, class-specific rates: colecalciferol 37%, calcium 36%, PPIs 43%, cessation 51%, dose reduction 49%).

Each of these changes the product calculus. Together, they say v1.0 is **a good product spec for a market that no longer exists.** Not because v1.0 is wrong — because the market underneath it has moved.

The remainder of this document maps the changes directly. I'll be specific about what stays, what gets revised, what gets replaced, and what gets added. I'll quote v1.0 sections where revision is needed.

---

## Part 1 — What stays from v1.0

These pieces of v1.0 hold up under the new framing and don't need substantive revision:

1. **The L0–L6 pipeline architecture, governance state machine, CompatibilityChecker, source-attribution, and audit trail** are all correct and continue to be the structural foundation. The reasoning-continuity reframe doesn't change the substrate; it changes what we build on top.

2. **The KB-1 through KB-29 substrate** is correct in scope. Some KBs need expanded roles (KB-2 Clinical Context becomes much more important under baseline-aware reasoning; KB-18 Audit Trail becomes the EvidenceTrace graph) but the entity model is right.

3. **The MVP → V1 → V2 sequencing over 12 months** is roughly right in shape, but the *content* of each phase changes substantially. See Part 6.

4. **The three product modes** (ACOP Embedded / RMMR Visiting / Facility Operator) are correct in name but the *workflow patterns within them* need revision — see Part 4.

5. **The seven user surfaces** (Shift Command Center, Worklist, Resident Workspace, GP Communication Hub, Facility Intelligence Dashboard, AN-ACC Revenue Assurance, Learning System) are all valid but they're now *renders of the five-state-machine substrate*, not standalone surfaces. The substrate has to come first.

6. **The MVP feature list (MVP-1 through MVP-5)** is correct in concept but some items need re-scoping. See Part 6.

---

## Part 2 — Strategic positioning revision (v1.0 Part 2 → v2.0)

### What v1.0 said

> *"What we're building: Vaidshala for Aged Care is a Clinical Operating System for Australian residential aged care medication management."*

### What v2.0 should say

**Vaidshala is the clinical reasoning continuity infrastructure for medication-related care in Australian aged care.**

This is not a marketing change. It is a category change. Three reasons it matters:

**Reason 1: Defensibility.** "Clinical Operating System" puts Vaidshala in a category populated by aged care vendors with broader scope (Leecare, AutumnCare, Person Centred Software, etc.). Vaidshala will lose feature comparisons against them on rostering, financials, care planning, etc. — domains we don't own and shouldn't try to. "Clinical reasoning continuity infrastructure" creates a category Vaidshala is alone in; it forces evaluators to assess us on a different axis.

**Reason 2: The actual moat.** v1.0 named the moat as "closed-loop tracking with outcome measurement linked to evidence-cited intervention rationale." That's correct as far as it goes, but it understates what the platform actually does. The moat is the **EvidenceTrace graph** — the longitudinal preservation of *why* a clinical decision was made, *what alternatives were considered*, *who acted on it*, *what they observed*, *how it propagated to the next decision* — across actors, shifts, transitions, and time. That dataset doesn't exist anywhere in Australian aged care today. It's what every downstream use (regulatory audit, AN-ACC defensibility, Star Ratings evidence, AI training, longitudinal outcome research) actually needs.

**Reason 3: Alignment with where Australian regulation is heading.** The PHARMA-Care framework (UniSA-led, $350M program evaluator, PSA-endorsed, in active national pilot) is explicitly a *quality monitoring* framework. The ACSQHC Stewardship Framework is explicitly a *stewardship* framework. The Aged Care Quality Standard 5 (Clinical Care, in force 2026) is explicitly an *evidence* requirement. None of these match a "Clinical Operating System" framing. All of them match a "stewardship infrastructure" or "reasoning continuity infrastructure" framing.

**Concrete sentence to use in pitch decks, partnership conversations, and investor materials:**

> *"Vaidshala is the clinical reasoning continuity layer for medication-related care in Australian aged care — preserving the chain of clinical thought across actors, time, and care transitions so that no decision has to be re-reasoned from scratch and no recommendation gets lost in fragmented workflows."*

**Sub-positioning for specific buyers (see Part 5):**

- For consultant pharmacy practices: *"the platform that preserves your pharmacists' clinical reasoning across the GP handoff so it doesn't get lost in translation."*
- For RACH operators: *"the medication stewardship infrastructure that produces your PHARMA-Care indicators, Standard 5 evidence, and AN-ACC defensibility as workflow exhaust."*
- For GP networks and practice managers: *"the clinical complexity capture and authority verification layer that lets your GPs handle aged care safely without becoming the bottleneck."*

---

## Part 3 — System architecture revision (v1.0 Part 3 → v2.0)

### What v1.0 said

The three-layer architecture (Signal → Decision → Conversion) with seven user surfaces sitting on top.

### What v2.0 should say

The three layers are still correct as a *user-experience model*, but they sit on top of a deeper substrate that v1.0 didn't name: **five interlocking state machines** sharing a common substrate of clinical entities. This substrate is the load-bearing piece. Without it, the three layers are workflow tooling. With it, they're clinical reasoning continuity infrastructure.

### The five state machines (new architectural commitment)

```
┌─────────────────────────────────────────────────────────────────────┐
│ THREE-LAYER USER EXPERIENCE (existing v1.0)                          │
│ Signal → Decision → Conversion                                       │
│ Seven user surfaces render off this layer                            │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌─────────────────────────────────────────────────────────────────────┐
│ FIVE INTERLOCKING STATE MACHINES (new in v2.0)                       │
│                                                                      │
│  1. Authorisation — runtime: who may act on this resident now?       │
│  2. Recommendation — proposed clinical action lifecycle              │
│  3. Monitoring — observation obligations with stop criteria          │
│  4. Clinical state — slowly-evolving baseline + active concerns      │
│  5. Consent — regulatory substrate for restrictive practice          │
│                                                                      │
│  All five share one EvidenceTrace graph (queryable bidirectionally) │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌─────────────────────────────────────────────────────────────────────┐
│ SHARED CLINICAL ENTITIES (new in v2.0, partly extends KB substrate)  │
│                                                                      │
│  Resident, Person+Role, MedicineUse (intent + target + stop          │
│  criteria), Observation (with computed delta-from-baseline),         │
│  Event, EvidenceTrace                                                │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
                              │
                ┌─────────────┴───────────────┐
                │ DATA INGESTION              │
                │ CSV (Phase 1) → eNRMC FHIR  │
                │ → Pathology HL7 → Hospital  │
                │ discharge → MHR → Dispensing│
                │ pharmacy → Behavioural notes│
                └─────────────────────────────┘
```

### Why each state machine has to be separate

I want to be explicit about this because the temptation is to collapse them into one and the document model gets ugly fast.

**Authorisation** is separate because the answer to "may this person act now?" changes for reasons that have nothing to do with the action itself. A designated RN prescriber's authority lapses if their mentorship evaluation isn't completed. An ACOP credential expires. A GP's collaborative agreement with an NP is revoked. A consent is withdrawn. A jurisdiction's PCW exclusion takes effect. Authorisation is a runtime evaluation, not a configuration. **This is the new safety primitive that almost nobody is building.**

**Recommendation** is the lifecycle v1.0 already specified. It carries `detected → drafted → submitted → viewed → decided → implemented → monitoring-active → outcome-recorded → closed`. Critical: `deferred` must be an explicit state with a forced review_date and escalation if it expires unconsidered. The Ramsey 2025 data — 50% of RMMR recommendations not implemented by next RMMR 12 months later — is largely this failure mode.

**Monitoring** is separate because monitoring outlives the recommendation that triggered it. The cessation closes Monday; the monitoring plan ("watch for urinary retention 14 days, falls 30 days, cognition 30 days") runs for a month. If observations don't land, monitoring escalates. If they cross threshold, monitoring produces a *new* Event that re-enters the trigger surface for Recommendation. **Most CDS systems treat monitoring as a free-text follow-up note attached to a closed recommendation. That's why the outcome loop never closes.**

**Clinical state** is separate because *change* is the primary clinical question. The slowly-evolving baseline (per-observation-type baseline with confidence interval, active concerns, care intensity tag) turns observation noise into signal. "Resident drowsy today" is noise; "drowsy 4 of 7 vs baseline 0 of 7, plus benzo PRN increase, plus DBI escalation" is signal. The delta is computed on write, not on read.

**Consent** is separate because the Aged Care Quality Standards (in force 2026), the restrictive practice legislation (since 2019), and the antipsychotic Quality Indicator Program reporting requirements make consent state the *legal substrate* underneath every psychotropic and restrictive-practice recommendation. Treating consent as a portal is treating compliance as decoration. Consent has its own state machine: `requested → discussed → granted | refused | granted-with-conditions → active → under_review → withdrawn | expired`. When a Recommendation enters `submitted` and falls within a class requiring Consent, it cannot advance to `decided` without an active matching Consent — and the platform makes this gap visible, not blocking, with the option to authorise initiation pending consent if clinically urgent.

### The shared EvidenceTrace graph

All five state machines write to one graph, not five logs. Every state transition records:
- the action taken (state change)
- the actor who took it (Person + Role + Authorisation evaluation)
- the inputs that justified it (links to Observations, prior Recommendations, guideline references, rule fires)
- the alternatives considered (even if rejected)
- the reasoning summary

The graph is queryable in both directions: forward (given a recommendation, what did it produce?) and backward (given an outcome, what reasoning produced it?). This is what makes Vaidshala different from a workflow tool. It's also non-trivial to implement — concurrency, transactional integrity, schema evolution all become harder. The team should expect this to be the hardest engineering piece in V1, and the highest-value architectural commitment.

### Implementation order

The Sunday-night-fall walkthrough in the second synthesis document is the right test case. **Before any further architectural work, walk through this scenario in person with engineering and clinical leads** and identify where current Vaidshala primitives can already handle each state machine and where new infrastructure is needed.

I'd suggest this sequencing for V1 build:

1. **Substrate entities first** (Resident, Person, Role, MedicineUse with intent+target+stop criteria, Observation with delta-on-write, Event, EvidenceTrace). About 4 weeks.
2. **Authorisation evaluator** (rule format, cache invalidation, audit query API). About 4 weeks. Latency budget: aim for 200ms but plan for 300-500ms in V1.
3. **Recommendation lifecycle** as a thin layer to validate the substrate. 2 weeks.
4. **Monitoring as a separate lifecycle.** 3 weeks.
5. **Clinical state with running baselines.** 4 weeks.
6. **Consent as regulatory substrate.** 3 weeks.

Total substrate build: ~20 weeks. This is a real investment. It's also what makes the rest of the product work.

---

## Part 4 — Three product modes — what changes

### What v1.0 said

Three product modes (ACOP Embedded / RMMR Visiting / Facility Operator) with shared data substrate.

### What v2.0 should add

**The four-role authority model.** v1.0 modeled four actors (pharmacist, RN, GP, resident/family). The verified regulatory landscape now requires a structurally larger model:

| Role | Status (April 2026) | What changes for the platform |
|---|---|---|
| GP | Existing primary prescriber | Remains primary; UI must visibly *strengthen* GP authority, not route around it (RACGP friction is real) |
| Nurse Practitioner | Autonomous since November 2024 (Collaborative Arrangements removed for MBS/PBS access) | NPs are now the **lowest-friction prescriber** for ACOP-style workflows; treat as first-class equal to GP |
| ACOP-credentialed pharmacist | All require APC training by 1 July 2026 | Already first-class; credential verification becomes important |
| Designated RN prescriber | Endorsement standard live since 30 Sept 2025; first cohort mid-2026; partnership-only with prescribing agreement and 6-month mentorship | New role; platform must track prescribing agreement, scope, mentorship status |
| RN (non-prescriber) | Existing | Remains; assessment, monitoring, escalation |
| Enrolled Nurse | Existing | Remains; administration under supervision |
| Personal Care Worker | Victorian PCW exclusion live 1 July 2026 (90-day grace to 29 Sept 2026) | Jurisdiction-aware: VIC PCWs cannot administer S4/S8/S9 + drugs of dependence to non-self-administering residents |
| Pharmacist co-prescriber (Tasmania) | Pilot 2026-2027, Australian-first, in collaboration with GP per treatment plan | New role in TAS only; platform should be ready as the digital substrate |
| Dispensing pharmacy | Existing community pharmacy | First-class execution actor (DAA timing) |
| Hospital | Existing | First-class transition counterparty |
| Substitute decision-maker | Existing under restrictive practice regs | First-class consent state actor |
| Resident | Existing | First-class signal where capacity allows |
| Regulator (ACQSC, AIHW, PHARMA-Care, ACSQHC) | Existing | Ghost user — shapes buyer's pain |

**The product implication is clear and uncomfortable:** v1.0 said "make our pharmacists 3× more effective" and "improve GP acceptance rate from 51.5% to 65%." That framing assumed a fixed authority structure. The actual authority structure is shifting: NPs autonomous, designated RN prescribers from mid-2026, pharmacist co-prescribers in Tasmania from 2026-2027, autonomous pharmacist prescribing nationally from ~2027-2028.

**v1.0's framing of "the pharmacist proposes, the GP authorises" understates the platform.** The accurate framing is: **"any of four prescribing roles can authorise, the platform tracks who's allowed to do what, and the pharmacist's clinical reasoning survives the handoff regardless of who acts."** That's a much harder product to build, but it's the product that fits the world arriving in 2026-2028.

### The bottleneck moves, not disappears

When more people can prescribe, the bottleneck moves from *authority delay* to *authority verification*. The new question every action attempts is:

> *For this resident, this medicine, this moment, who is authorised to do what?*

Specifically:
- Is there a current prescribing agreement on file between the RN prescriber and an authorised practitioner?
- Does the agreement cover this resident, this medicine class, this dose change?
- Is the six-month mentorship period complete?
- Has the pharmacist's ACOP credential been verified for the current period?
- Does the GP's collaborative agreement with the NP exclude any medicines?
- Has the resident's substitute decision-maker consented to this class of changes?
- Does the jurisdiction permit this person to administer this scheduled medicine?

These are PDFs in shared drives, paper agreements in filing cabinets, MOUs nobody can find. They could become structured data with scope, expiry, audit trail, machine-readable authorisation logic. **Almost nothing in the current product landscape does this.** This is the new safety primitive Vaidshala should own.

---

## Part 5 — Buyer model revision (v1.0 Part 7 → v2.0)

### What v1.0 said

Three buyers (consultant pharmacy practice, RACH operator, visiting RMMR pharmacist) with three independent value stories.

### What v2.0 should say

The three buyers are correct, but the *value stories* and *commercial sequencing* need substantial revision based on the verified landscape.

### Buyer 1 — Consultant pharmacy practice (Tier 1 ACOP buyer) — REVISED

**v1.0 framing:** "Make our pharmacists 3× more effective" — productivity multiplier.

**v2.0 framing:** "Preserve your pharmacists' clinical reasoning across the GP handoff so it doesn't get lost in translation, and produce the PHARMA-Care indicators that prove your service quality."

The shift matters because:
1. The 51.5% acceptance rate ceiling isn't moved primarily by speed — it's moved by *what makes it across the handoff*. The Ramsey data shows class-specific implementation rates (colecalciferol 37%, calcium 36%, PPIs 43%) reflecting how well the recommendation rationale survives translation, not how fast the pharmacist works.
2. PHARMA-Care indicators are about to become the standard by which ACOP services are evaluated. Practices that can produce them as workflow exhaust have a structural commercial advantage.

**Verified pricing anchor:** ACOP Tier 1 pays the community pharmacy AUD 619.84/day per FTE (Feb 2026 rules). For 1 FTE working 228 days/year = AUD 141K/year of subsidized pharmacist time per facility. Vaidshala license at AUD 30/bed/month = AUD 360/bed/year. Per 250-bed facility (1 FTE coverage) = AUD 90K/year, or 64% of the subsidized pharmacist cost. The ROI calculation is whether the platform makes the subsidized pharmacist >64% more valuable. With the productivity multiplier + acceptance-rate improvement + PHARMA-Care indicator generation + audit defensibility, this is defensible.

### Buyer 2 — RACH operator (Tier 2 ACOP + Facility Operator) — REVISED

**v1.0 framing:** "Standard 5 audit defensibility + AN-ACC revenue assurance + Star Ratings improvement."

**v2.0 framing:** *all three remain valid, plus:* **"PHARMA-Care indicators produced as workflow exhaust" + "Victorian PCW exclusion compliance infrastructure" (where applicable) + "designated RN prescriber scope verification" (from mid-2026).**

The two new commercial levers are real:

1. **From 1 July 2026 in Victoria** (with 90-day grace to 29 Sept 2026), RACHs need to redesign their administration workflow to ensure RN/EN coverage for S4/S8 + drugs of dependence at every administration round. Many facilities will struggle. A platform that *visibly maintains the legal-administration trail* — who administered, under whose authority, with what scope, with what observation — is not just clinical infrastructure but **compliance infrastructure for a brand-new regulatory regime**. Other states will likely follow. This is a wedge into Victorian RACHs that didn't exist when v1.0 was written.

2. **The ACSQHC Medication Management at Transitions of Care Stewardship Framework** (published 2024) and the **PHARMA-Care National Quality Framework** (UniSA-led, in active national pilot evaluating the $350M ACOP program) together define what regulators are about to measure. A platform that produces these indicators automatically is making the buyer's purchase justification for them.

### Buyer 3 — GP networks and practice managers — NEW POSITIONING

**v1.0 framing:** Visiting RMMR pharmacist (a third buyer, smaller wedge).

**v2.0 framing:** v1.0 had this nearly right but framed it as a small wedge. The updated framing is sharper: **GP networks and practice managers buy the platform as the clinical complexity capture and authority verification layer that lets their GPs handle aged care safely without becoming the bottleneck.**

This matters because of the AMA/RACGP narrative. The AMA was unambiguous about the Tasmania pharmacist prescribing pilot ("further care fragmentation," "the $5M would be far better spent supporting GPs"). The RACGP's ACOP submission was explicit that pharmacists lack the diagnostic skills required, that minor symptoms could indicate deeper health issues particularly for older people, and that duplication of unnecessary primary care services will lead to fragmentation of care.

**A platform that visibly *strengthens* GP authority** — one-click approval, one-click revocation, comprehensive audit trail, explicit scope verification for non-GP prescribers — is much easier to defend than one that visibly routes around them. **Get this framing right early; it's the difference between RACGP being a partner and being an antagonist.**

This third buyer becomes more important than v1.0 framed because GP practice managers are the ones who run the aged-care visit logistics for their GPs. They feel the pain of the current system most acutely.

---

## Part 6 — MVP → V1 → V2 sequencing revision (v1.0 Part 4 → v2.0)

### What v1.0 said

12-month sequencing: MVP (1-3 months), V1 (4-6 months), V2 (7-12 months) with feature lists per phase.

### What v2.0 should say

The 12-month shape stays, but the *content* of each phase shifts substantially. The five-state-machine substrate has to land in MVP or V1; the regulatory windows (Victorian PCW exclusion, designated RN prescriber, Tasmanian pilot, $350M ACOP mandatory training) all hit between July 2026 and 2027. The product has to be ready.

### MVP (Months 1-3) — REVISED

**v1.0 MVP features:** Unified Medication Timeline, ACOP Worklist with priority clusters, Resident Workspace with trajectory panel, GP recommendation generator, Audit trail.

**v2.0 MVP features (all new or substantially revised):**

- **MVP-1: Substrate entities + Authorisation evaluator** — the foundation. Without this, V1 can't build the rest. Resident, Person, Role, MedicineUse (with intent+target+stop criteria), Observation (with delta-on-write), Event, EvidenceTrace, plus the runtime authorisation rule format and cache. *This is new vs v1.0.*
- **MVP-2: Unified Medication Timeline with eNRMC + CSV ingestion** — same as v1.0 but explicitly designed against the conformant eNRMC vendors that exist by April 2026 (8 of 10 conformant per the November 2025 status), with CSV fallback for non-conformant facilities and for Phase 1 deployments.
- **MVP-3: Recommendation lifecycle as thin layer over substrate** — v1.0's "ACOP Worklist with priority clusters" rebuilt as a render of the Recommendation state machine, with `deferred` as explicit state.
- **MVP-4: Monitoring lifecycle as separate object** — *new vs v1.0; this is the architecturally important addition.*
- **MVP-5: Decision packet generator with explicit guideline tension flags** — v1.0's "GP recommendation generator" upgraded to include guideline tensions per the Ramsey data, plus PBS authority criteria pre-population, plus alternative considerations explicit.
- **MVP-6: Standard 5 evidence panel** — v1.0's audit trail upgraded to produce Standard 5 evidence as workflow exhaust, not separate report.

**MVP exit criterion (revised):** One ACOP pharmacist can complete a full medication review for one resident in 5 minutes from cold start. The recommendation reaches the GP with rationale preserved (not just "cease oxybutynin"). The audit trail is intact. The Standard 5 evidence bundle is generated. **The five-state-machine substrate is in place even if some surfaces aren't built yet.**

### V1 (Months 4-6) — REVISED

**v1.0 V1 features:** GP Communication Hub, Facility Intelligence Dashboard, Counterfactual simulator, GP behavior model, Continuous monitoring, Multi-facility deployment.

**v2.0 V1 features:**

- **V1-1: Clinical state with running baselines** — *new vs v1.0; the baseline-aware reasoning layer.* Per-observation-type baselines computed continuously, deltas surfaced as signal.
- **V1-2: Consent state machine** — *new vs v1.0; the regulatory substrate for restrictive practice.* SubstituteDecisionMaker entity, capacity assessment integration, consent gating for psychotropic recommendations.
- **V1-3: GP Communication Hub** — same as v1.0 but with the closed-loop tracking explicit, recommendation-supersedes-recommendation logic, and Smart Form/structured-input integration where prescriber system supports it.
- **V1-4: Facility Intelligence Dashboard with PHARMA-Care indicators** — v1.0's dashboard upgraded to produce the PHARMA-Care five-domain indicators automatically.
- **V1-5: Hospital discharge reconciliation workflow** — *new vs v1.0; v1.0 had no hospital channel.* Discharge summary ingestion, pre-admission medication chart diff, change-flagging, prioritization, ACOP routing within 24 hours, pre-fill recommendation packet for GP/NP.
- **V1-6: Dispensing pharmacy integration** — *new vs v1.0; the DAA timing layer.* Structured cessation/change alerts to dispensing pharmacy, DAA packing schedule as state, latency surfaced to ACOP.
- **V1-7: Jurisdiction-aware ScopeRules infrastructure** — *new vs v1.0; the Victorian PCW exclusion + designated RN prescriber + Tasmanian pilot infrastructure as data, not code.* This is what makes V2's role-aware UI feasible without a rebuild.
- **V1-8: Multi-facility deployment** — same as v1.0.

**V1 exit criterion (revised):** A consultant pharmacy practice running 5 ACOPs across 8 RACHs sees recommendation acceptance rates climb from baseline (publish whatever the actual baseline is; don't assume 51.5%) toward higher numbers, with measurable improvement in the *fraction of recommendations whose clinical rationale survives the handoff*. PHARMA-Care indicators produced automatically. Hospital discharge reconciliation operational. Dispensing pharmacy integrated for at least one large vendor.

### V2 (Months 7-12) — REVISED

**v1.0 V2 features:** AN-ACC Revenue Assurance, Pricing risk assessment readiness, Learning System, Causality engine, Goals-of-care alignment.

**v2.0 V2 features:**

- **V2-1: AN-ACC Revenue Assurance (KB-28)** — same as v1.0.
- **V2-2: Designated RN prescriber workflow** — *new vs v1.0; the role-aware UI for the new prescribing endorsement.* Prescribing agreement ledger, mentorship status tracking, scope-match-per-action verification, audit trail for regulator.
- **V2-3: Tasmanian pharmacist co-prescribing pilot integration** — *new vs v1.0; if Vaidshala is the digital substrate for the pilot.* This requires partnership with UTas (Salahudeen, Peterson, Curtain) or Tasmanian Department of Health. See Part 7.
- **V2-4: Victorian PCW exclusion compliance infrastructure** — *new vs v1.0; the legal-administration trail.* Who administered, under whose authority, with what scope, with what observation — visibly maintained, regulator-queryable.
- **V2-5: Learning System surfaced selectively** — same as v1.0.
- **V2-6: Causality engine for ADE attribution** — same as v1.0.
- **V2-7: Goals-of-care alignment** — same as v1.0, but now expanded to the Clinical state machine's `care_intensity` tag (palliative / comfort / active treatment / rehabilitation) which reshapes every recommendation downstream.

**V2 exit criterion (revised):** A RACH operator sees AN-ACC reassessment-driven revenue recovery, Standard 5 audit defensibility, measurable medication safety improvement (reduction in mandatory-reported antipsychotic use per QI Program), PHARMA-Care indicators automatically produced, and Victorian PCW exclusion compliance evidence (where applicable) within 12 months of deployment.

---

## Part 7 — Go-to-market revision (v1.0 Part 7.3 → v2.0)

### What v1.0 said

12-month sequencing: 1-2 friendly pilots in Months 1-3, expand to 5-10 facilities in Months 4-6, scale to 20+ paying RACHs and 3-5 consultant pharmacy practices by Month 12. Target AUD 5-8M ARR by month 12.

### What v2.0 should add

The same shape, but **three concrete commercial moves should happen in the next 30-60 days** that v1.0 didn't surface:

**Move 1: Engage the Tasmanian pharmacist co-prescribing pilot as the digital substrate partner.**

The pilot is Australian-first, $5M state budget, in development late 2025, trialled 2026-2027, accepted in full by the Tasmanian Government, structurally needs a digital substrate to track pharmacist-GP co-prescribing per treatment plan. The pilot timing aligns with Vaidshala's V2 build window. The workflow it needs (pharmacist proposes, GP authorises, RN monitors) is exactly what Vaidshala wants to be.

**Concrete actions:**
- Contact Mohammed Salahudeen and Gregory Peterson at the University of Tasmania School of Pharmacy. They authored the Ramsey 2025 implementation paper and are likely involved in the pilot design.
- Contact the Tasmanian Department of Health Pharmacy Projects team (Duncan McKenzie is named in the budget announcement).
- Offer Vaidshala as the digital substrate at no cost for the pilot duration, with publication rights for outcomes and reference-customer rights for commercial use.

This is the **single highest-leverage commercial move available** because it lands Vaidshala inside an Australian-first regulatory innovation with academic, government, and PSA backing.

**Move 2: Engage the PHARMA-Care framework consortium for indicator alignment.**

PHARMA-Care is UniSA-led (Sluggett), $1.5M MRFF-funded, 14 project partners, formally evaluating the $350M ACOP program, PSA-endorsed, in active national pilot phase as of November 2025 with EOI open for aged care providers and on-site pharmacists.

**Concrete actions:**
- Contact Janet Sluggett and Sara Javanparast at UniSA (ALH-PHARMA-Care@unisa.edu.au is the published EOI address).
- Position Vaidshala as the implementation partner that produces PHARMA-Care indicators automatically.
- Offer participation in the pilot evaluation as both a deployed platform and a measurement substrate.

This produces three things at once: (a) PHARMA-Care framework alignment for product design, (b) a published academic evaluation of the platform's effectiveness, (c) endorsement-quality positioning with regulators and buyers.

**Move 3: Reframe the AMA/RACGP narrative before it ossifies.**

The AMA's response to the Tasmania pilot is on the public record: "further care fragmentation," "the $5M would be far better spent supporting GPs." The RACGP's ACOP submission is on the public record: "duplication of unnecessary primary care services will lead to fragmentation of care," concerns about pharmacist diagnostic limitations.

**Concrete actions:**
- Engage RACGP's aged care interest group early. Position Vaidshala as the platform that visibly *strengthens* GP authority through scope verification and audit trail.
- Engage AMA's primary care division (recognising they will be more skeptical than RACGP).
- Avoid any messaging that implies GP "bottleneck" or "delay" or "routing around" — even when it's structurally true.
- Use messaging that focuses on what the platform does *for* the GP: structured decision packet, scope verification of non-GP prescribers, comprehensive audit trail for medico-legal protection, time recovery from documentation duplication.

**These three moves should happen in parallel and within 30-60 days.** They're inexpensive, high-signal, and they lock in three commercial dependencies (academic credibility, regulator alignment, GP-college non-opposition) that are much harder to acquire later.

---

## Part 8 — Risk register revision (v1.0 Part 8 → v2.0)

### What v1.0 said

Six risks: ADG 2025 licensing, AMH licensing timeline, GP integration tooling fragility, acceptance rate may not move as much as hoped, eNRMC vendor cooperation, restrictive practice legal exposure.

### What v2.0 should add

All six remain. Five new risks from the verified landscape:

**Risk 7: Jurisdictional regulatory fragmentation.** Victorian PCW exclusion is the first of likely several state-level changes. Other states may follow with different timelines, different scope, different rules. The platform's ScopeRules infrastructure must be data-not-code from V1, or every state-level change becomes an engineering project.

**Risk 8: Designated RN prescriber rollout uncertainty.** The endorsement standard is live but no NMBA-approved education programs exist as of late 2025. First endorsed prescribers expected mid-2026. Actual uptake by RACFs is unknown — RNs need employer authorization and a prescribing agreement, both of which are operationally complex. Build the infrastructure but **don't assume designated RN prescribers will be a meaningful population in V1**. Plan for them to be a meaningful population in V2.

**Risk 9: Pharmacist autonomous prescribing timeline acceleration.** The joint AdPha/Pharmacy Guild/PSA submission to the Pharmacy Board of Australia in October 2025 proposes a national framework for pharmacist autonomous prescribing. Realistic timeline: 2027-2028 nationally, sooner in some states. **If this accelerates, the four-role authority model expands to five and the platform must adapt.** This is an opportunity, not a threat — but it's a moving target.

**Risk 10: Hospital integration depth.** The hospital discharge channel is genuinely the highest-yield intervention point, but it requires My Health Record integration and ideally direct hospital ADT feeds. MHR integration is non-trivial; hospital ADT feeds are difficult to obtain. **Plan for V1 hospital reconciliation to work from PDF discharge summaries with manual upload; V2 to add MHR integration; V3 to add ADT feeds.** Don't promise V1 hospital integration depth that requires V3 technical work.

**Risk 11: PHARMA-Care framework evolution.** The framework is in active national pilot phase. Indicators may be refined or changed based on pilot findings. **Build the platform's indicator computation as configurable, not hardcoded.** Re-evaluate alignment quarterly with UniSA team.

---

## Part 9 — Success metrics revision (v1.0 Part 9 → v2.0)

### What v1.0 said

Pharmacist-level (time on context assembly 43% → 25%, reviews completed per FTE per month, recommendation acceptance rate 51.5% → 65%), facility-level (antipsychotic prevalence reduction, polypharmacy reduction, AN-ACC reassessment, Standard 5 evidence availability), system-level (recommendation lifecycle closure rate, time from source update to deployed CQL define, CompatibilityChecker pass rate). North Star metric: preventable adverse drug events per 1,000 resident-days.

### What v2.0 should add

All v1.0 metrics remain. Five new metrics aligned with the verified landscape:

**Pharmacist-level (new):**
- **Recommendation rationale survival rate** — % of recommendations where the GP/prescriber's response cites or addresses the specific clinical reasoning, not just the proposed action. This measures reasoning continuity directly.
- **Class-specific implementation rates vs Ramsey 2025 baseline** — colecalciferol target >40% (vs 37%), calcium >40% (vs 36%), PPIs >50% (vs 43%), cessation overall >55% (vs 51%), dose reduction >55% (vs 49%). These are measurable improvements over the published Australian baseline.

**Facility-level (new):**
- **PHARMA-Care five-domain indicator scores** — produced automatically as workflow exhaust. The framework is the standard; track against it.
- **Victorian PCW exclusion compliance rate** (Victorian facilities only) — % of S4/S8 administrations to non-self-administering residents performed by RN/EN/pharmacist/medical practitioner. Target: 100% by 29 September 2026 (end of grace period).

**System-level (new):**
- **Authorisation evaluator latency** — p95 < 500ms in V1, < 200ms in V2.
- **EvidenceTrace graph query depth** — max actor-handoffs preserved in a single recommendation chain. This measures the moat directly.

**The North Star metric stays:** preventable adverse drug events per 1,000 resident-days. **Add a North Star North Star:** *the existence of an Australian aged care medication management dataset where, given any outcome (fall, hospitalization, ADE), the full clinical reasoning chain that produced or failed to prevent it can be queried back.* This dataset doesn't exist anywhere globally. Building it is the strategic moat.

---

## Part 10 — A summary of what changes

A one-page summary for the team, mapping v1.0 → v2.0:

| Section | v1.0 | v2.0 |
|---|---|---|
| Strategic positioning | "ACOP Clinical Operating System" | **"Clinical reasoning continuity infrastructure for medication-related care in Australian aged care"** |
| System architecture | 3-layer (Signal/Decision/Conversion) + 7 surfaces | 3-layer + 7 surfaces **on top of 5-state-machine substrate** sharing one **EvidenceTrace graph** |
| State machines | Recommendation lifecycle only | **Five interlocking machines:** Authorisation, Recommendation, Monitoring, Clinical state, Consent |
| Actor model | 4 actors (pharmacist, RN, GP, family) | **8-12 roles** (adds NP, designated RN prescriber, EN, PCW, dispensing pharmacy, hospital, SDM, regulator) — staged in V1/V2 |
| MVP | 5 features over substrate-light platform | **6 features over five-state-machine substrate**; substrate is itself MVP work |
| V1 | GP Hub, Facility Dashboard, simulator, behavior model | **Adds:** Clinical state, Consent, Hospital discharge reconciliation, Dispensing pharmacy, Jurisdiction-aware ScopeRules |
| V2 | AN-ACC, Pricing risk, Learning, Causality, Goals-of-care | **Adds:** Designated RN prescriber workflow, Tasmanian pilot integration, Victorian PCW exclusion compliance |
| Buyer 1 (pharmacy) | "3× more effective" | "Reasoning preservation across handoff + PHARMA-Care indicators" |
| Buyer 2 (RACH) | "Standard 5 + AN-ACC + Star Ratings" | **Adds:** "PHARMA-Care indicators + VIC PCW compliance + designated RN scope verification" |
| Buyer 3 (GP) | Visiting RMMR pharmacist (small wedge) | **GP networks/practice managers as clinical complexity capture + authority verification layer** |
| GTM moves | 1-2 friendly pilots in months 1-3 | **Plus three concrete moves in next 30-60 days:** Tasmanian pilot, PHARMA-Care framework, AMA/RACGP narrative |
| Risks | 6 risks | 6 + **5 new:** jurisdictional fragmentation, RN prescriber uncertainty, pharmacist prescribing acceleration, hospital integration depth, PHARMA-Care framework evolution |
| Success metrics | Pharmacist + facility + system + 1 North Star | **Adds:** rationale survival rate, class-specific implementation vs Ramsey baseline, PHARMA-Care scores, VIC compliance, authorisation latency, EvidenceTrace depth + **North Star North Star** (queryable reasoning chain dataset) |

---

## Part 11 — What this means for v1.0 documents already written

Three documents need consequential revision:

**Final Product Proposal v1.0:**
- Replace Part 2 (positioning) with the v2.0 reasoning-continuity framing
- Replace Part 3 (system architecture) with the five-state-machine substrate + three-layer model
- Augment Part 4 (product features) with V1/V2 additions per Part 6 above
- Augment Part 7 (commercial positioning) with the three concrete commercial moves
- Augment Part 8 (risks) with the five new risks
- Augment Part 9 (success metrics) with the new metrics

**Layer 1 Australian Aged Care Implementation Guidelines** (will need follow-up document):
- Add hospital discharge summary + MHR continuity sources
- Add dispensing pharmacy DAA timing as a new data source
- Add jurisdiction-aware ScopeRules as part of the regional shim (AU/VIC subset, AU/TAS subset)
- Add designated RN prescriber credential and prescribing agreement structures
- Add PHARMA-Care framework indicator definitions

**Layer 3 Rule Encoding Implementation Guidelines** (will need follow-up document):
- Add the five-state-machine substrate as Part 0.5 (before any rule authoring)
- Add Authorisation evaluator as a separate subsystem (not part of CQL rule firing — it gates whether rules can produce actions)
- Add ScopeRules-as-data architecture (cannot be hardcoded in CQL; needs a separate jurisdiction-aware rules engine)
- Update the four-bucket priority taxonomy to recognize that some rules can only fire if the appropriate Authorisation is granted (changing how priority is computed)

**Layer 2 Implementation Guidelines** (not yet written, now significantly more important):
- The Clinical state machine — running baselines, delta computation on write, transition events — is a Layer 2 deliverable, not Layer 3
- The structural commitment to baseline-aware reasoning has to be made before any rule authoring can rely on it
- This is where the patient-state plumbing (CSV → eNRMC → pathology → frailty/palliative status) lands

---

## Part 12 — Closing

Three things to register as we close.

**One:** The reasoning-continuity reframe is the right thesis. It survives stress-testing against the verified regulatory landscape. It gives Vaidshala a category to itself. It points at the actual moat (the EvidenceTrace graph as a longitudinal dataset). It aligns with where Australian aged care regulation is heading (PHARMA-Care, ACSQHC stewardship, Strengthened Quality Standards, restrictive practice). The team should adopt it as the strategic positioning and update all external-facing materials accordingly.

**Two:** The five-state-machine substrate is the right architectural commitment. Building it is a real investment — roughly 20 weeks of MVP+V1 work to get all five live. But it's what makes the rest of the product work. Without it, Vaidshala is a workflow tool with good UX. With it, Vaidshala is clinical reasoning continuity infrastructure with a defensible moat. The team should commit to the substrate even though it pushes some surface features from MVP to V1.

**Three:** The three commercial moves in the next 30-60 days (Tasmanian pilot, PHARMA-Care framework, AMA/RACGP narrative) are inexpensive, high-signal, and lock in commercial dependencies that are much harder to acquire later. They should happen in parallel, not sequenced. The Tasmanian pilot in particular is a window that closes — the pilot is being designed late 2025, trialled 2026-2027, and the digital substrate decision is likely made within the next 6 months. **If Vaidshala isn't in that conversation by mid-2026, someone else will be.**

The product Vaidshala is now positioned to build is structurally different from what v1.0 described, and structurally better-fit to the world that's arriving. The work to revise v1.0 is substantial but the foundation is sound. What changes is the framing, the substrate, and the sequencing — not the underlying capabilities the team has been building.

In follow-up documents we'll do the same exercise for Layer 1 and Layer 3 Implementation Guidelines, mapping what changes in each based on this revised foundation.

— Claude
