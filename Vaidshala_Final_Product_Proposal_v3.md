# Vaidshala Final Product Proposal — v3.0

**Date:** May 2026
**Supersedes:** *Vaidshala Final Product Proposal v1.0* (foundational architecture) and *Vaidshala Final Product Proposal v2.0 Revision Mapping* (April 2026 reasoning-continuity reframe).
**Purpose:** Consolidate v1.0 + v2.0 into a single working canon, incorporating the strategic, commercial, clinical-craft, and ethical-architecture work completed across the May 2026 strategic exploration thread. From this document, pilot design proceeds.

**Reading order:** This document replaces v1.0 and v2.0 as the working specification. Layer 1 / Layer 2 / Layer 3 implementation guidelines remain valid as referenced; their consequential updates are noted in Part 16.

---

## Part 0 — What changed in v3.0 and why

The v2.0 mapping (April 2026) reframed Vaidshala from "Clinical Operating System" to "clinical reasoning continuity infrastructure" and introduced the five-state-machine substrate. That work survives stress-testing and is preserved intact. What v3.0 adds is the layer of strategic and operational understanding that comes from systematically walking the platform through every reviewer in the aged-care medication-management pyramid and every craft layer of the pharmacist's actual work.

The May 2026 thread produced eight additions that v2.0 did not yet contain:

1. **The three-nested-layer strategic position** — architecture, purpose, and measurement, each used in a different audience conversation. v2.0 had the architectural framing; v3.0 adds the purpose framing (continuous accountable stewardship) and the measurement framing (Recommendation Implementation Rate as operational North Star).

2. **The reviewer pyramid as the platform's structural location** — every actor in the system is simultaneously reviewer and reviewed. The platform's strategic position is the longitudinal evidence substrate every reviewer uses. This produces multi-sided lock-in that no single-stakeholder tool can replicate.

3. **A four-buyer commercial model** (with emerging fifth) — the consultant pharmacy practice, the RACH operator, the GP network, and now explicitly the **pharmacist as bottom-up buyer**, with the PHN as a longer-horizon regional-observability buyer.

4. **Pricing structure corrections** — federal ACOP funding is $141,323.52 per FTE per year, not the $220K figure that recurred in earlier strategic notes. Funding scales by beds (1 FTE per 250 beds), not by homes. Pricing anchor at 5–10% of recovered funding = $7K–$14K per FTE per year, or roughly $30–$40 per bed per month for facility-mode pricing.

5. **The recommendation craft engine** — the engineering of GP acceptance through structured templates, recommendation-type ordering, urgency tiers, evidence anchoring, brevity enforcement, and per-GP framing adaptation. This is a substantial new product module that v1.0/v2.0 treated as a single feature ("decision packet generator").

6. **The expertise gap closure model** — a five-dimension analysis of what separates a new pharmacist from an experienced geriatric pharmacist, with explicit assessment of how much of each dimension the platform can credibly close. This shapes both feature priorities and the workforce-development positioning.

7. **The ethical architecture** — a set of explicit design principles: frame adapts, content doesn't; clinical appropriateness checked alongside acceptance; restraint signals as well as action signals; pharmacist autonomy preserved; reviewability of the platform itself; trust architecture controlling who sees whose data. v2.0 implied some of this; v3.0 makes it architecturally explicit.

8. **Workforce development & CPD integration as deliberate design principles** — the platform as teacher (accelerated competence development), CPD evidence support, RPL-evidence pathway for the 30 June 2026 credentialing cliff, and pharmacist self-visibility as the bottom-up adoption mechanism.

These additions don't replace v2.0; they extend it across dimensions v2.0 didn't address. The five-state-machine substrate remains the architectural commitment. The reasoning-continuity reframe remains the strategic positioning. What v3.0 does is make the platform commercial, ethical, and clinically humane as well as architecturally sound.

---

## Part 1 — Strategic positioning, three nested layers

### What v2.0 said

> *"Vaidshala is the clinical reasoning continuity infrastructure for medication-related care in Australian aged care."*

That stays. v3.0 adds two layers underneath, used in different audience conversations.

### v3.0 strategic position — three nested layers

**Layer A — Architecture (for investors, regulators, category-defining narrative):**

> *"The longitudinal clinical-operations evidence substrate that every reviewer in aged-care medication management — pharmacist, pharmacy, home, network, regulator — uses to perform their review with evidence rather than impression."*

This is the version used when the audience is asking "what is this category?" Investors and regulators need to understand that Vaidshala isn't a CDS tool, an EMR, or an aged-care platform. It's the evidence substrate that sits beneath all of those, queryable bidirectionally, accumulated longitudinally.

**Layer B — Purpose (for buyers, especially RACHs and chain executives):**

> *"Continuous accountable stewardship of longitudinal medication-related stability — reducing the time residents spend in unresolved instability, reducing the surprises every stakeholder fears."*

This is the version used in buyer conversations. RACHs don't buy "evidence substrates"; they buy reduction in surprises (preventable hospitalisations, complaints, audit findings, incidents). Chain executives don't buy "infrastructure"; they buy account retention and operational margin. The purpose framing translates the architecture into outcomes the buyer cares about.

**Layer C — Measurement (for pilots, pharmacists, pricing structures):**

> *"Recommendation Implementation Rate as the operational North Star, sustained at +12 percentage points above baseline, with downstream outcomes tracked but not used as primary commercial structure."*

This is the version used when the conversation needs to land on a specific testable claim. RIR is measurable today, converts cleanly into all stakeholder languages, and is the necessary precondition for the deeper outcomes the platform eventually claims credit for.

The three layers point at the same thing. The buyer hears Layer B or C in their language. The platform internally builds for Layer A. The strategic moat lives in the gap between what each layer says and what they collectively are.

### Three sub-positionings stay valid (from v2.0)

- **For consultant pharmacy practices:** *"the platform that preserves your pharmacists' clinical reasoning across the GP handoff so it doesn't get lost in translation, and produces the PHARMA-Care indicators that prove your service quality."*
- **For RACH operators:** *"the medication stewardship infrastructure that produces your PHARMA-Care indicators, Standard 5 evidence, and AN-ACC defensibility as workflow exhaust."*
- **For GP networks and practice managers:** *"the clinical complexity capture and authority verification layer that lets your GPs handle aged care safely without becoming the bottleneck."*

v3.0 adds a fourth:

- **For pharmacists themselves:** *"the platform that makes your clinical reasoning, recommendation acceptance, and outcome contribution visible — to yourself first, in service of your career, your CPD, and your professional development."*

The fourth sub-positioning is the bottom-up adoption motion. See Part 5 for the buyer mechanics.

---

## Part 2 — The reviewer pyramid

### Why this matters

The strategic location of the platform is not "above the pharmacist's workflow" or "between pharmacist and GP" or "inside the RACH's clinical governance." It sits across all of these because every actor in the medication-management chain is *simultaneously a reviewer of the layer below and reviewed by the layer above*, and each performs their reviewing role with partial evidence today.

### The pyramid

```
                ┌───────────────────────────────────────┐
                │ POLITICAL/PUBLIC                      │
                │ Inspector-General, Parliament,        │
                │ media, families, royal commission risk│
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ REGULATOR                             │
                │ ACQSC (Standard 5, Standard 2)        │
                │ AHPRA (registration, CPD)             │
                │ APC (ACOP credentialing)              │
                │ PPA (claim integrity)                 │
                │ IHACPA (AN-ACC pricing)               │
                │ AIHW/PHARMA-Care (quality indicators) │
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ RACH GOVERNING BODY                   │
                │ Aged Care Act 2024 personal liability │
                │ Reviews pharmacy partner monthly      │
                │ Star Ratings, occupancy, AN-ACC       │
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ RACH OPERATIONAL                      │
                │ DON, Quality Lead, Facility Manager   │
                │ MAC reviews ACOP performance          │
                │ Standard 5 evidence, QI Program       │
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ PHARMACY EMPLOYER                     │
                │ Reviews pharmacist annually           │
                │ Contract retention, FTE economics     │
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ PHARMACIST                            │
                │ Reviews own practice via CPD          │
                │ Self-visibility into KPI trajectory   │
                └───────────────────────────────────────┘
                                ▲
                ┌───────────────────────────────────────┐
                │ RESIDENT/FAMILY                       │
                │ Reviews home via Star Ratings,        │
                │ complaints, occupancy decisions       │
                └───────────────────────────────────────┘
```

### Three structural implications

**Multi-sided lock-in.** When a single platform serves the pharmacist, the employer, the RACH, and indirectly the regulator from one substrate, each stakeholder accumulates value that's hard to migrate. A competitor building a single-stakeholder tool — even if technically superior at one slice — cannot replicate the cross-stakeholder substrate. Defensibility comes from the interlocking nature of the reviews, not from any single feature.

**Network effects across reviewers.** The pharmacist's evidence track improves their employability. The pharmacy's evidence track improves their RACH retention. The RACH's audit track improves their Star Ratings and admissions. Each layer's data quality benefits the layer above and below. Compounding value — each new participant strengthens the substrate for everyone else.

**Pricing power expands across the pyramid.** The current pitch is to the pharmacy as productivity tool. Once the platform demonstrably serves the RACH (compliance, audit defensibility) and the regulator (evidence quality), additional revenue lines open: per-RACH facility module, RACH-group enterprise license, regulator-certified evidence packs, pharmacist career-portfolio premium tier. The chain pitch is the entry. The pyramid is the long game.

### What this means architecturally

The platform's architecture must support five distinct view-types over the same substrate:

1. **Pharmacist view** — own KPI trajectory, recommendation pipeline, CPD-relevant case portfolio
2. **Pharmacy employer view** — pharmacist comparative performance, RACH satisfaction signals, contract retention drivers (with appropriate ethical limits — see Part 9)
3. **RACH view** — pharmacy-partner performance, polypharmacy/antipsychotic trajectories, MAC reporting cadence, Standard 5 evidence packs
4. **Chain head office view** (longer-horizon) — network-level performance across franchisees and homes
5. **Regulator view** — audit-defensible evidence of clinical governance in operation

Same data, five views, each consequential to the reviewer who consumes it. The view-permission architecture is therefore foundational, not bolt-on. Trust architecture (see Part 9) governs who sees whose data.

---

## Part 3 — System architecture (carries forward v2.0 with extensions)

### Five-state-machine substrate (v2.0, preserved)

```
┌─────────────────────────────────────────────────────────────────────┐
│ THREE-LAYER USER EXPERIENCE                                          │
│ Signal → Decision → Conversion                                       │
│ Seven user surfaces render off this layer                            │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
┌─────────────────────────────────────────────────────────────────────┐
│ FIVE INTERLOCKING STATE MACHINES                                     │
│  1. Authorisation — runtime: who may act on this resident now?       │
│  2. Recommendation — proposed clinical action lifecycle              │
│  3. Monitoring — observation obligations with stop criteria          │
│  4. Clinical state — slowly-evolving baseline + active concerns      │
│  5. Consent — regulatory substrate for restrictive practice          │
│  All five share one EvidenceTrace graph (queryable bidirectionally) │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
┌─────────────────────────────────────────────────────────────────────┐
│ SHARED CLINICAL ENTITIES                                             │
│  Resident, Person+Role, MedicineUse (intent + target + stop          │
│  criteria), Observation (with computed delta-from-baseline),         │
│  Event, EvidenceTrace                                                │
└─────────────────────────────────────────────────────────────────────┘
                              ▲
┌─────────────────────────────────────────────────────────────────────┐
│ DATA INGESTION                                                       │
│ CSV (Phase 1) → eNRMC FHIR → Pathology HL7 → Hospital discharge     │
│ → MHR → Dispensing pharmacy → Behavioural notes                     │
└─────────────────────────────────────────────────────────────────────┘
```

The five-state-machine substrate, the EvidenceTrace graph, the shared clinical entities, and the data ingestion strategy from v2.0 are preserved verbatim. See v2.0 Part 3 for the rationale on why each state machine must be separate and how they interlock.

### v3.0 architectural additions

Two architectural commitments are added in v3.0 that v2.0 didn't yet specify:

**Architectural Commitment 6: View-permission layer.** Five view-types over one substrate (pharmacist self-view, pharmacy employer view, RACH view, chain network view, regulator view). View-permissions are first-class entities, not afterthoughts. Pharmacist data is pharmacist-controlled by default; aggregation upward is permission-gated; identifiable cross-comparison is restricted. The trust architecture (Part 9) defines the rules.

**Architectural Commitment 7: Recommendation craft engine.** The recommendation lifecycle is a state machine; the recommendation *itself* needs to be assembled to maximise acceptance probability while preserving clinical appropriateness. The craft engine (Part 7) is a separate subsystem that takes the pharmacist's clinical intent and renders it into a recommendation packet structured for the receiving prescriber, with framing adapted but content invariant. This is more product engineering than v1.0/v2.0 specified for the "decision packet generator."

**Architectural Commitment 8: Clinical appropriateness check.** Acceptance optimisation without appropriateness anchoring is a clinical safety failure waiting to happen. The platform must track recommendation appropriateness *alongside* acceptance and flag any pattern where high-quality framing is carrying through clinically marginal recommendations. This is both a product feature and a metric integrity guard (Part 11).

---

## Part 4 — The four-role authority model (carries forward v2.0)

The four-role authority model from v2.0 is preserved in full. The role table (GP / NP / ACOP-credentialed pharmacist / Designated RN prescriber / RN / EN / PCW / Pharmacist co-prescriber TAS / Dispensing pharmacy / Hospital / SDM / Resident / Regulator) and the bottleneck-shifts-to-authority-verification framing remain correct.

### v3.0 additions to the authority model

**The 30 June 2026 credentialing cliff** is now a near-term commercial wedge. From that date, every ACOP pharmacist must hold APC-accredited credentials; the MMR grandfathering ends. Pharmacies whose ACOP-eligible workforce includes pharmacists yet to complete training or RPL face a workforce-availability crisis. The platform's evidence trail can serve as **RPL-evidence substrate** — granular, longitudinal, auditable demonstration of competency in real ACOP work, supporting credential-progression applications. This is a time-bound differentiated value proposition that closes when the cliff passes. See Part 10.

**Algorithmic performance management constraints.** When platform-generated KPIs feed pay decisions, AHPRA professional accountability and Fair Work Commission emerging guidance both apply. The platform must:
- preserve the pharmacist's individual clinical judgment as the ultimate authority on each recommendation
- show the pharmacist their data before any aggregated upward sharing
- provide formal contestation pathway for any KPI feeding employment decisions
- log algorithmic-vs-human decisions distinctly so the audit trail shows where the platform supported the pharmacist and where it tried to direct them

This is both a legal requirement and a trust-architecture requirement. See Part 9.

---

## Part 5 — Buyer model: four buyers + emerging fifth

### Buyer 1 — Consultant pharmacy practice (Tier 1 ACOP buyer)

**Verified pricing reality:**
- ACOP Tier 1 pays community pharmacy AUD 619.84/day per FTE (Feb 2026 rules)
- 1 FTE = 228 days/year max claimable = AUD 141,323.52 federal funding cap per FTE
- Funding scales by beds: 1 FTE per 250 beds, in 50-bed increments
- A 250-bed home generates 1.0 FTE entitlement (~$565/bed/year recovered)
- A 100-bed home generates 0.4 FTE entitlement (~$56K/year)

**v2.0 framing held:** "Preserve your pharmacists' clinical reasoning across the GP handoff so it doesn't get lost in translation, and produce the PHARMA-Care indicators that prove your service quality."

**v3.0 commercial sharpening:**

The buyer's deeper fear is **operational commoditisation** — RACHs treating ACOP pharmacists as interchangeable labour. The platform's competitive value to the chain is differentiation: "our pharmacists deliver measurably higher recommendation acceptance, audit-defensible documentation, and PHARMA-Care indicators built into the workflow." That's a tender-defensible claim no competitor pharmacy can match without similar tooling.

**Pricing structure (revised from v2.0):**

The 5-10% of recovered ACOP funding heuristic produces:
- $7K–$14K per FTE per year (AUD)
- Or equivalently: $30–$40 per bed per month at facility-mode pricing
- Tiered: base platform license + per-active-pharmacist adjustment

Pure outcome-share pricing was rejected in v3.0 strategic exploration as fragile (attribution wars, vendor-controlled meters). Outcome share remains as a modest upper-tier on top of base + performance, not as primary commercial backbone.

**Dispensing-pharmacy displacement risk** — when a chain holds both the ACOP and the supply contract, an ACOP recommending deprescribing reduces the supply pharmacy's dispensing revenue. The platform's audit trail and outcome documentation defend the chain against this conflict-of-interest critique by separating clinical reasoning from commercial impact in the record.

### Buyer 2 — RACH operator (Tier 2 ACOP + Facility Operator)

**v2.0 framing held:** Standard 5 evidence + AN-ACC defensibility + Star Ratings + PHARMA-Care indicators + Victorian PCW exclusion compliance + designated RN prescriber scope verification.

**v3.0 sharpening:**

The RACH carries **ultimate clinical-care accountability** under the Aged Care Act 2024. Funding flows through the pharmacy (Tier 1) but accountability lands on the RACH. Under Standard 2, the governing body holds personal liability. This means RACH boards are increasingly involved in ACOP partner selection, and pitches at board / governing-body level land harder than at operational level for high-quality partners.

**The "fewer surprises" KPI is the unstated buyer-want.** RACHs don't fear pharmacist underperformance as much as they fear *silent longitudinal deterioration* — the slow drift no one notices until coronial, complaint, or audit. The platform's value isn't safety improvement; it's reduction in unpredictable escalation events. Different category, much sharper.

**Multi-vendor eNRMC reality** at group operators (8 RACHs across 3+ eNRMC vendors) makes the platform's vendor-agnostic ScopeRules infrastructure a *procurement-level* feature for groups, not a nice-to-have. Single-vendor competitor tools cannot service multi-site operators.

**MAC participation as ongoing performance review.** Procurement signs the Service Authorisation; the Medication Advisory Committee judges performance month-to-month. The platform's MAC-ready reports must be designed for that audience, not just for procurement-time pitches.

### Buyer 3 — GP networks and practice managers

**v2.0 framing held:** Clinical complexity capture and authority verification layer that lets GPs handle aged care safely without becoming the bottleneck.

**v3.0 sharpening:**

The platform's tone with GP networks must visibly *strengthen* GP authority — never imply "bottleneck" or "delay" or "routing around." Use messaging that emphasises what the platform does *for* the GP: structured decision packet, scope verification of non-GP prescribers, comprehensive audit trail for medico-legal protection, time recovery from documentation duplication.

The recommendation-craft module (Part 7) is specifically designed to land well with GPs: structured templates, clinical-context anchoring, ready-to-action language, monitoring plans embedded, brevity enforced. The platform should be the friend of busy GPs, not their judge.

### Buyer 4 — Pharmacists themselves (NEW in v3.0)

**Framing:** *"The platform that makes your clinical reasoning, recommendation acceptance, and outcome contribution visible — to yourself first, in service of your career, your CPD, and your professional development."*

**Why this is now an explicit buyer:**

The KPIs employer-pharmacies measure pharmacists on (recommendation acceptance rate at 30% weight; reviews completed; documentation quality; RACH satisfaction) are exactly what the platform makes legibly better. A pharmacist using the platform produces work that scores higher on annual review and supports faster pay-band progression. This creates a *bottom-up adoption motion*: pharmacists ask their employers to deploy the platform because it makes them look good at review time.

The bottom-up motion is structurally important because:
- It runs in parallel with the top-down chain-executive sales motion, doubling adoption surface area
- It makes the platform more durable inside the buyer organisation (pharmacists defend tools that benefit them, abandon tools that don't)
- It aligns the platform with the pharmacist's professional identity rather than with employer surveillance, which is the cultural difference between "tool I use" and "tool that watches me"

**Product implication:** Build the pharmacist self-visibility dashboard as Layer-A architecture, not as a Layer-C feature. The pharmacist sees their KPI trajectory, recommendation pipeline, CPD-relevant case portfolio, and per-GP acceptance patterns in their own dashboard *before* any aggregation upward to the employer happens. This is a foundational architectural commitment, not an afterthought.

**Commercial implication:** Pharmacist-side features create demand independent of employer adoption. Free tier with self-visibility for individual pharmacists is a viable distribution wedge. The pharmacy chain that wants the data has to deploy the platform; the pharmacist already has it personally.

### Buyer 5 (longer-horizon) — PHN as regional observability layer

PHNs today are connectors and matchmakers between RACHs and pharmacies. As the PHARMA-Care framework matures and the 2026–2027 ACOP evaluation produces regional findings, PHNs are likely to evolve into the **regional medication-governance visibility layer**. They'll want cohort-level data on which homes in their catchment are improving, which are stalling, where workforce gaps cluster, where quality varies.

This is a pre-procurement buyer — they don't pay per home but they *recommend* who gets engaged by homes. PHN endorsement is therefore commercially valuable. Build the regional dashboard as a v2/v3 capability; engage PHN aged-care leads progressively from V1 onward.

---

## Part 6 — Pricing structure

### Federal funding context (corrected from earlier strategic notes)

The recurring claim of $220K per FTE Tier 1 ACOP funding is incorrect. The correct figures, verified against the PPA Tier 1 Rules (February 2026):

- **Daily on-site rate:** AUD 619.84
- **Maximum claimable days per FTE per year:** 228 (= 261 weekdays − 20 leave − 13 public holidays)
- **Federal funding cap per FTE per year:** AUD 141,323.52
- **Practical realisation (with personal leave assumed):** AUD 138,224.32
- **Bed-based scaling:** 1 FTE per 250 beds, blocked into 50-bed increments
- **Per-bed annual recovery (250-bed home, 1 FTE):** ~AUD 565

This federal cap is a hard ceiling. Pharmacies cannot charge RACHs in addition; there is no extraction lever beyond the federal funding flow.

### Vaidshala pricing structure

**Base platform license** (per pharmacy organisation, scaled by size):
- Small consultant practice (1–5 FTE pharmacists): $30K–$60K/year
- Mid-sized practice (6–15 FTE): $60K–$150K/year
- Chain or large group (16+ FTE, multi-site): $150K+/year, negotiated

**Per-bed-per-month** (for facility-mode deployments, multi-RACH groups):
- $30–$40 per bed per month
- Translates to ~5–7% of recovered ACOP funding per facility
- 250-bed home: ~$90K–$120K/year
- 100-bed home: ~$36K–$48K/year

**Per-active-pharmacist supplement** (for credential-tracking, CPD substrate, RPL evidence):
- $1,500–$3,000 per pharmacist per year
- Adds individual self-visibility and credentialing-evidence module
- Optional in V1, default-included in V2

**Performance tier** (optional uplift, year 2+):
- 10–15% premium for sustained RIR uplift of +12 percentage points or more above baseline, measured over rolling 12 months
- Measurement transparency is required: the meter must be visible to both parties, with shared definitions and audit access
- Outcome-share pricing on hospitalisation avoidance / AN-ACC recovery: not recommended for primary structure; reserve for selected enterprise customers as a year-3+ option

**Pharmacist self-tier (new in v3.0)**:
- Individual pharmacist subscription: $50–$100/month
- Self-visibility dashboard, CPD evidence, recommendation history portfolio
- Distribution channel for organic adoption ahead of employer purchase
- Bottom-up demand creation; pharmacist takes their dashboard with them across employers

### Pricing principles

**The base license must fit inside recoverable funding.** If a 250-bed home generates ~$565/bed/year of ACOP funding (= $141K total), the platform pricing at $30–$40/bed/month ($90–$120/bed/year) consumes 16–21% of recovered funding. The buyer's productivity uplift must therefore exceed that share. The ROI argument is: fewer FTE required for same coverage (workforce leverage) + higher recommendation acceptance (clinical value uplift) + audit-defensibility (regulatory protection) + Star Ratings movement (occupancy/AN-ACC) + RPL-evidence substrate (workforce credentialing).

**Outcome-share pricing is fragile.** Multi-causal outcomes, contested attribution, vendor-controlled meters, and aged-care buyers' historical scepticism of vendor-favourable outcome deals all argue for keeping outcome share as a small upper tier on top of solid base + performance pricing.

**The pharmacist tier is strategic distribution.** Direct margin from individual pharmacist subscriptions is small. Strategic value is large: organic adoption that pulls employer purchase, plus pharmacist career portability that takes the dashboard across employers and creates pull demand at each new workplace.

---

## Part 7 — Recommendation craft engine (NEW MODULE)

### Why this is its own module

v1.0 and v2.0 treated recommendation generation as a single feature ("decision packet generator"). The May 2026 thread established that recommendation acceptance is a substantial engineering domain in its own right, with multiple distinct mechanisms each producing measurable acceptance lift. The craft engine is therefore promoted to a distinct subsystem.

### What constitutes a high-acceptance recommendation

The literature converges on a consistent set of constituents. Each is encodable into the platform.

**Structured template.** Issue → Clinical Context → Rationale → Evidence → Proposed Plan → Monitoring → Urgency. Free-text recommendations reliably underperform structured ones. The structure is enforced at recommendation-creation time; the platform won't allow a recommendation to enter the lifecycle without all sections populated (or marked explicitly as not-applicable).

**Clinical specificity beats vague suggestion.** "Reduce escitalopram from 20mg to 10mg over 4 weeks; reassess mood and orthostatic blood pressure at week 6" lands materially better than "consider reducing antidepressant." The platform's recommendation generator pre-populates specific dose, taper schedule, monitoring schedule, and re-evaluation criteria — leaving the pharmacist to confirm or adjust, not to write from blank.

**Resident-specific context, not generic flagging.** "In this 87-year-old with eGFR 32, recent fall, and on three other anticholinergics, drug X contributes 0.8 to her DBI and is the most modifiable contributor to her fall risk" beats "Drug X is potentially inappropriate." Context is auto-assembled from the substrate (current labs, recent events, frailty signals, goals-of-care tag, current burden scores) at recommendation-creation time.

**Evidence anchoring with audience-appropriate sources.** Australian GPs respect AMH, Therapeutic Guidelines, RACGP-condition-specific guidance, Australian Deprescribing Guideline 2025. International references (Beers, STOPP/START) work as supplementary anchors. The platform cites one or two strong sources, not a wall of references. Source selection adapts to recommendation type and clinical context.

**Time-anchored monitoring plan.** What to check, when, with what threshold for escalation. This converts the recommendation from one-shot suggestion to managed plan. The Monitoring state machine (substrate component 3) makes this enforceable; observations land or escalate.

**Brevity.** The recommendation must fit on a single screen for the receiving prescriber. Long recommendations are read less. The craft engine enforces a length budget, optimising for one-screen consumption with full reasoning available on click-through if the GP wants to drill in.

**Risk-benefit framing for this resident's goals.** Especially for deprescribing, framing the *patient-specific* benefit (reduced fall risk, simplified regimen for frail resident with months-to-years prognosis) lands better than rule-based framing. The craft engine pulls goals-of-care from the Clinical state machine's `care_intensity` tag and frames recommendations consistent with the resident's stated priorities.

### Recommendation type ordering

Acceptance varies systematically by recommendation type. Direction (not specific percentages) is well-supported in the literature:

- **STOP** recommendations have the highest acceptance — low GP effort, low downside, deprescribing aligns with national priorities
- **MONITOR** recommendations follow — low GP effort, low risk
- **DOSE CHANGE** recommendations are mid-range — moderate effort, no new safety concerns
- **ADD** recommendations have the lowest acceptance — GP concerns about polypharmacy, side effects, cost

The platform orders multi-recommendation packets to lead with low-effort/high-value items (typically STOPs), building GP momentum across the packet. ADD recommendations are placed last and only when clinically necessary. **The platform does not suppress clinically indicated ADD recommendations because acceptance is statistically lower** — see Part 9 on metric integrity. Ordering is craft; suppression is clinical safety failure.

### Urgency tiers

Three tiers, visible to both pharmacist and prescriber:

- **Red — URGENT (response within 24–48 hours):** AKI, hyperkalaemia, hypoglycaemia risk, recent fall on CNS-active medications, QTc prolongation, drug-induced acute confusion
- **Amber — IMPORTANT (response within 1–2 weeks):** Deprescribing opportunity, dose optimisation, monitoring overdue, evidence-based guideline alignment with active clinical context
- **Green — ROUTINE (discuss at next review):** Preventive measures, cost-saving switches, minor optimisations, opportunistic suggestions

Tiering supports the receiving prescriber's triage. The platform also tracks per-GP urgency-acceptance pattern: GPs may have characteristic responses (some accept urgent recommendations near-universally but ignore green; others read everything; others respond only to phone calls for red). The platform learns and prompts the pharmacist on the most-likely-effective channel for each tier per GP.

### Per-GP framing learning (with ethical limits)

The platform observes recommendation-acceptance patterns per receiving prescriber over time. Patterns include:
- Recommendation types that land vs. don't (deprescribing vs. dose adjustment vs. add)
- Evidence sources that resonate (AMH vs. Beers vs. RACGP vs. local guideline)
- Communication channels that work (structured email vs. phone vs. embedded eNRMC note)
- Framing styles that land (clinical-urgency-first vs. patient-context-first vs. monitoring-burden-first)

The platform surfaces these patterns to the pharmacist preparing a recommendation as **gentle suggestions**, not as scorecards. The framing adaptation is offered: *"recommendations to Dr Smith have landed better when monitoring plans are explicit up front"* — not *"Dr Smith has 42% acceptance rate, increase your framing intensity."* The latter is professionally toxic and politically risky; the former is the same adaptive communication an experienced pharmacist does instinctively.

### The frame-vs-content principle

**Framing adapts to receiving prescriber. Content does not.** The clinical recommendation — what to do, why, what to monitor — is invariant across audiences. Only the way it's communicated adapts. This distinction must be auditable in the platform: the EvidenceTrace for any recommendation shows both the clinical content and the framing-adaptation layer, separately. A regulator querying back a recommendation must be able to see that the clinical reasoning was identical across all framings; only language adapted.

This is both ethical architecture (Part 9) and operational protection: it prevents the platform from being accused of varying clinical advice by audience.

### Restraint signals

Sometimes the right clinical answer is to *not* recommend. The resident is stable on a suboptimal regimen, intervening risks destabilisation, the family isn't ready, the timing isn't right, the GP isn't ready, the resident is end-stage frail with minimal upside from optimisation. Junior pharmacists tend to over-intervene because every potentially inappropriate medication looks like a recommendation opportunity. Senior pharmacists choose battles.

The platform supports restraint by surfacing context that argues for it as well as for action: *"this resident is at end-stage frailty per care_intensity = palliative; current regimen tolerated; three potential deprescribing opportunities exist but family decision-maker is processing recent decline; consider deferring — schedule re-review in 6 weeks"*. This is unusual for CDS platforms, which tend to maximise alerts. It's exactly what distinguishes the platform from a more aggressive recommendation engine and is a key differentiator with experienced pharmacists.

### Clinical appropriateness check

Acceptance optimisation without appropriateness anchoring is metric corruption. The craft engine pairs every recommendation with an appropriateness assessment:
- Is this recommendation clinically warranted given current substrate state?
- Is the evidence base solid for this resident's profile?
- Have alternatives been considered?
- Has restraint been considered?
- Does the recommendation align with the resident's goals-of-care?

If acceptance rates rise without corresponding appropriateness scores rising in parallel, the platform flags potential metric distortion to the pharmacy employer. This is metric integrity (Part 11) made architectural.

---

## Part 8 — Expertise gap closure model

### What separates a new pharmacist from an experienced geriatric pharmacist

Five dimensions, each closing differently. The platform's role differs at each. This model shapes feature priorities and the workforce-development positioning.

| Dimension | Gap Description | Platform Closure | Mechanism |
|---|---|---|---|
| Clinical Knowledge | Beers, STOPP/START, deprescribing guideline, geriatric pharmacology, dose-adjustments | Substantially closeable | KB-1 through KB-5 substrate; embedded CDS at point of action |
| Pattern Recognition | Trajectories, cascades, subtle deprescribing opportunities, temporal associations | Partially closeable | Clinical state machine baselines + delta computation; substrate-level pattern surfacing |
| Communication & Persuasion | GP relationship capital, framing skill, negotiation, channel optimisation | Partially closeable | Recommendation craft engine (Part 7); per-GP framing learning |
| Prioritisation & Triage | Risk-stratified worklist, urgency-vs-routine sorting, cognitive bandwidth allocation | Substantially closeable | Risk-scored daily queue; urgency tiering across substrate |
| Systems Thinking | Facility-level QUM patterns, prescriber-mix issues, cohort drift | Substantially closeable | Facility Intelligence Dashboard; PHARMA-Care indicators |

The residual gap, deliberately left to the pharmacist:

- **Calibration / clinical gestalt** — knowing when to push, when to defer, when not to intervene; reading the family's readiness; reading the GP's mood; recognising that the chart problem isn't always the real problem. This is what experience builds. The platform supports it through accumulated implementation intelligence (Part 11) and restraint signals (Part 7) but does not replace it.

### Honest positioning

The platform substantially closes the knowledge and structure gaps, meaningfully closes the communication-craft gap, and explicitly leaves untouched the parts that should remain in pharmacist hands — judgment, restraint, relational work, and individual clinical accountability.

A new pharmacist using the platform produces work whose clinical content approaches senior-level, with recommendations whose acceptance rate sits well above the unaided baseline. They do not become a senior pharmacist overnight. The platform does not pretend otherwise.

### The platform as teacher

A pharmacist using the platform for two years sees:
- How cases are systematically reasoned through
- Which of their recommendations get accepted and why
- Patterns in their own practice that improve or stagnate
- Per-GP framing patterns that worked or didn't

This accelerates the journey from new to experienced — informally estimated at 18–36 months of compression, based on how decision-support tools have shortened skill acquisition in other clinical domains. This is a meaningful workforce-development claim that strengthens the platform's positioning under workforce constraints (the 30 June 2026 credentialing cliff, the 1 FTE per 250 beds funding ratio, the scarcity of credentialed geriatric pharmacists).

### Workforce-development implications

The platform's design supports workforce *capability* rather than workforce *dependence*:

- The pharmacist's reasoning is visible to themselves and they can see how it evolves
- The substrate explains *why* recommendations are surfaced, not just *what* to do
- Restraint signals teach when not to intervene
- Per-GP framing learning is presented as observation and suggestion, not directive
- Over time, a skilled pharmacist uses the platform increasingly as data infrastructure rather than guidance

This is a deliberate design choice. A platform that creates dependence is a workforce risk; a platform that builds capability is a workforce asset.

---

## Part 9 — Ethical architecture

### Why this is its own section

The May 2026 thread surfaced a set of ethical principles that the platform must architect for, not bolt on. v1.0 and v2.0 had implicit ethics; v3.0 makes them explicit and architectural.

### The seven principles

**Principle 1: Frame adapts, content doesn't.** Recommendation framing varies by audience; clinical content is invariant. Auditable separation of clinical-content layer from framing-adaptation layer in EvidenceTrace.

**Principle 2: Acceptance follows appropriateness.** Recommendation acceptance rate is optimised only when paired with maintained or rising appropriateness scores. The platform flags any pattern where framing carries marginal recommendations through.

**Principle 3: Restraint is a clinical answer.** The platform supports "watchful wait" recommendations, surfaces context arguing for non-intervention, does not maximise alerts.

**Principle 4: Pharmacist autonomy preserved.** AHPRA accountability is individual. The platform supports judgment; it does not replace it. Audit trail clearly shows pharmacist's reasoning at each step. Algorithmic suggestions are distinguishable in the record from pharmacist clinical decisions.

**Principle 5: Self-visibility before aggregation.** Pharmacist sees own data first. Aggregation upward to employer requires permission. Identifiable cross-comparison is restricted. The bottom-up adoption motion depends on this trust architecture.

**Principle 6: Reviewability of the platform itself.** The platform is a reviewer of clinical work; its own outputs are subject to review. Explainability, audit trail of platform-side decisions, override pathways, formal contestation mechanisms must exist for any KPI feeding employment decisions or audit findings.

**Principle 7: GP authority strengthened, not routed around.** No messaging implies bottleneck or delay. The platform's tone with GPs is collaborative, evidence-supportive, time-saving. The audit trail strengthens medico-legal protection for GPs, not weakens it.

### What this means architecturally

**Trust architecture** (view-permissions): five view-types, default-private to subject, explicit permission for aggregation upward. Pharmacy chain cannot see individual pharmacist KPIs without pharmacist opt-in or contractual notice. RACH cannot see pharmacist comparative performance. Regulator-view permissions are governed by formal data-sharing agreement with the regulator, not by the buyer.

**Algorithmic vs human distinction in audit trail:** every recommendation, every framing adaptation, every restraint suggestion is logged with attribution: was this the platform suggesting, the pharmacist accepting, the pharmacist over-riding, the platform supporting? The audit trail is the substrate for both clinical accountability and algorithmic-management compliance.

**Contestation pathway:** any KPI surfaced to employer that affects pay, retention, or contract decisions is contestable by the pharmacist. The contestation creates a formal record visible to both parties, and the algorithmic determination cannot be the sole basis for an adverse employment decision.

**Persuasion guardrails:** the recommendation craft engine is auditable for cases where high-quality framing carries clinically marginal recommendations to acceptance. If detected as a pattern, the platform's framing intensity is dampened and clinical appropriateness is foregrounded.

### Two failure modes to architect against

**Persuasive framing of inappropriate recommendations.** Higher acceptance of bad recommendations is worse than lower acceptance of good ones. The clinical appropriateness check (Part 7) is the architectural guard.

**Suppression of clinically necessary recommendations because they're predicted to be rejected.** If the pharmacist is judged on acceptance rate and the platform shows "this recommendation type has 42% acceptance with this GP," the rational response is to not make the recommendation. But sometimes the recommendation needs to be made anyway, rejected, made again next quarter. The platform should not optimise this discomfort away. The KPI framework (Part 11) tracks recommendation appropriateness alongside acceptance, and any pattern of suppression is flagged to the pharmacy employer.

---

## Part 10 — Workforce development & CPD integration

### The 30 June 2026 credentialing cliff

From that date, every ACOP pharmacist must hold APC-accredited credentials. The MMR grandfathering ends. The path to credentialing is either:
- Complete an APC-accredited ACOP training program
- Complete an APC-accredited Recognition of Prior Learning (RPL) process

The RPL process requires evidence of competency in real ACOP work. The platform's longitudinal evidence trail — granular, auditable, demonstrating clinical reasoning across cases — is exactly what RPL applications need.

**Commercial wedge:** for the next ~12 months, a pharmacy whose ACOP-eligible workforce includes pharmacists yet to credential faces workforce-availability risk. The platform's RPL-evidence module makes this risk navigable. Time-bound, defensible, and almost no competitor will have it.

### CPD integration

AHPRA mandates 40 hours of CPD per year for general registration. ACOP credentialing requires aged-care-specific CPD beyond that. The platform's case-based clinical work is itself CPD-eligible: each comprehensive review, with documented clinical reasoning, evidence engagement, recommendation crafting, and outcome tracking, is CPD activity.

The CPD module:
- Tags eligible activities automatically as they occur in normal workflow
- Surfaces CPD-relevant cases for reflective writing
- Generates CPD records ready for AHPRA submission
- Links each CPD entry to the underlying EvidenceTrace for verification

This converts the platform's clinical work from purely operational into a triple-purpose data layer: clinical work product, audit defensibility, CPD documentation.

### Pharmacist career portfolio

The pharmacist self-tier (Part 6) gives the pharmacist a portable record of their clinical work across employers:
- Recommendation history with outcomes
- Per-GP acceptance trajectory
- Clinical scenarios handled, with category and complexity
- CPD log
- Credential status and progression

This is portable across employers — a pharmacist who changes pharmacies takes their portfolio with them. The portfolio supports pay negotiation, credential progression, and eventually, recognition by the profession (PSA, AdPha, College of Geriatric Pharmacy if it emerges).

### The teaching dimension

The platform's design supports the pharmacist's competence development (Part 8). This is a deliberate counter to the workforce concerns raised by the 1 FTE per 250 beds ratio: the platform helps pharmacy chains scale ACOP services with the workforce that exists, partly by accelerating new pharmacists toward senior-level performance.

---

## Part 11 — KPI framework, three-layer

### The three measurement layers

Aligned with the three-nested-layer strategic positioning:

**Layer A (Architecture) metrics — for the platform's strategic moat:**
- EvidenceTrace graph query depth (max actor-handoffs preserved in a single recommendation chain)
- Reasoning continuity rate (% of recommendations where clinical rationale survives the prescriber handoff intact)
- View-permission integrity (% of cross-stakeholder data access compliant with permission rules)
- North Star North Star: existence of a queryable Australian aged-care medication-management dataset where, given any outcome, the full clinical reasoning chain that produced it can be queried back

**Layer B (Purpose) metrics — for buyers and regulators:**
- Reduction in unpredictable escalation events (preventable hospital transfers, SIRS-reportable medication incidents, family complaints)
- Time-in-instability per resident — the integral of medication-state drift without resolution. This is the deep KPI the platform aspires to own. PHARMA-Care framework definition pending; the platform's substrate produces the data when the framework lands.
- PHARMA-Care five-domain indicator scores (produced as workflow exhaust)
- Standard 5 evidence pack completeness and audit-readiness
- Antipsychotic and polypharmacy QI Program movement

**Layer C (Measurement) metrics — for pilots, pharmacists, pricing:**
- Recommendation Implementation Rate (RIR) — % of pharmacist recommendations with documented prescriber action within agreed window. Sustained +12 percentage points above baseline is the platform's commitment.
- Class-specific implementation rates vs Ramsey 2025 baseline (colecalciferol target >40% vs 37% baseline; calcium >40% vs 36%; PPIs >50% vs 43%; cessation overall >55% vs 51%; dose reduction >55% vs 49%)
- Time per comprehensive review (target ~40–50% reduction from baseline)
- Context-assembly time (target <8 minutes vs 20–25 minute baseline)
- Pharmacist self-visibility metrics (own RIR trajectory, own per-GP patterns, own CPD progress)

### Metric integrity guards

**Recommendation appropriateness paired with acceptance.** Acceptance optimisation without appropriateness anchoring is metric corruption. The platform tracks both, in parallel, and flags divergence.

**Suppression detection.** If recommendation-type frequency drops without corresponding clinical-context change, the platform flags potential suppression of clinically indicated recommendations.

**Quality of recommendation, not volume.** Reviews completed and recommendations made are operational metrics, not quality metrics. The KPI framework foregrounds quality (acceptance + appropriateness + outcome linkage) over volume.

**View-permission audit.** Algorithmic decisions affecting employment are auditable, contestable, and distinguishable from human decisions in the record.

### Pharmacist-level KPIs (revised from v2.0)

- RIR per pharmacist, with appropriateness paired (target +12pp above baseline)
- Class-specific implementation rates vs Ramsey 2025
- Recommendation rationale survival rate
- Context-assembly time
- CPD activity completion
- Per-GP framing pattern observed (own dashboard only; not aggregated upward without consent)

### Facility-level KPIs (revised from v2.0)

- PHARMA-Care five-domain indicator scores
- Antipsychotic and polypharmacy QI Program movement
- Standard 5 evidence pack completeness
- Victorian PCW exclusion compliance rate (where applicable)
- Time-in-instability per resident (when PHARMA-Care framework defines)
- Hospital-discharge medication reconciliation completion within 72 hours

### System-level KPIs

- Authorisation evaluator latency (p95 <500ms in V1, <200ms in V2)
- EvidenceTrace graph query depth
- Reasoning continuity rate across handoffs
- View-permission integrity rate

### North Star

Preventable adverse drug events per 1,000 resident-days (carry forward from v1.0).

### North Star North Star

The existence of an Australian aged-care medication-management dataset where, given any outcome (fall, hospitalization, ADE), the full clinical reasoning chain that produced or failed to prevent it can be queried back. This dataset doesn't exist anywhere globally. Building it is the strategic moat.

---

## Part 12 — MVP → V1 → V2 sequencing

### Shape preserved from v2.0

12-month sequencing: MVP (1–3 months), V1 (4–6 months), V2 (7–12 months). The five-state-machine substrate must land in MVP or early V1; the regulatory windows (Victorian PCW exclusion, designated RN prescriber, Tasmanian pilot, $350M ACOP mandatory training, 30 June 2026 credentialing cliff) all hit between July 2026 and 2027. Product must be ready.

### MVP (Months 1–3)

**MVP-1: Substrate entities + Authorisation evaluator** — foundation. Resident, Person, Role, MedicineUse (with intent+target+stop criteria), Observation (with delta-on-write), Event, EvidenceTrace, plus runtime authorisation rule format and cache. Latency budget 200–500ms.

**MVP-2: Unified Medication Timeline with eNRMC + CSV ingestion** — designed against conformant eNRMC vendors with CSV fallback for non-conformant facilities and Phase 1 deployments.

**MVP-3: Recommendation lifecycle as thin layer over substrate** — render of Recommendation state machine with `deferred` as explicit state, forced review-date, escalation if expires unconsidered.

**MVP-4: Monitoring lifecycle as separate object** — observation obligations with stop criteria, escalation if observations don't land or cross threshold.

**MVP-5: Recommendation craft engine v1** — *new in v3.0; promoted from "decision packet generator" to subsystem.* Structured template, clinical-context auto-assembly, evidence anchoring, brevity enforcement, recommendation-type ordering, urgency tiering. **Per-GP framing learning is V1, not MVP.**

**MVP-6: Standard 5 evidence panel** — audit trail produces Standard 5 evidence as workflow exhaust, not separate report.

**MVP-7: Pharmacist self-visibility dashboard** — *new in v3.0; foundational architectural commitment for bottom-up adoption.* Pharmacist sees own RIR trajectory, recommendation pipeline, CPD-relevant case portfolio, per-GP acceptance patterns. Permission-gated against upward aggregation by default.

**MVP exit criterion (revised):** One ACOP pharmacist can complete a full medication review for one resident in 5 minutes from cold start. The recommendation reaches the GP with rationale preserved, structured per craft engine v1, ordered and tiered. The audit trail is intact. Standard 5 evidence is generated. The pharmacist's own dashboard reflects the work. The five-state-machine substrate is in place.

### V1 (Months 4–6)

**V1-1: Clinical state with running baselines** — per-observation-type baselines computed continuously, deltas surfaced as signal.

**V1-2: Consent state machine** — SubstituteDecisionMaker entity, capacity assessment integration, consent gating for psychotropic recommendations.

**V1-3: GP Communication Hub** — closed-loop tracking, recommendation-supersedes-recommendation logic, Smart Form / structured-input integration.

**V1-4: Facility Intelligence Dashboard with PHARMA-Care indicators** — dashboard produces PHARMA-Care five-domain indicators automatically.

**V1-5: Hospital discharge reconciliation workflow** — discharge summary ingestion, pre-admission medication chart diff, change-flagging, prioritization, ACOP routing within 24 hours.

**V1-6: Dispensing pharmacy integration** — structured cessation/change alerts to dispensing pharmacy, DAA packing schedule as state, latency surfaced to ACOP.

**V1-7: Jurisdiction-aware ScopeRules infrastructure** — VIC PCW exclusion + designated RN prescriber + Tasmanian pilot infrastructure as data, not code.

**V1-8: Per-GP framing learning module** — *new in v3.0.* Observed acceptance patterns surfaced to pharmacist as gentle suggestions (Part 7). Ethical limits architected per Principle 5 and Principle 6 (Part 9).

**V1-9: Restraint signals + clinical appropriateness check** — *new in v3.0.* Surfaces context arguing for non-intervention; pairs every recommendation with appropriateness score; flags divergence.

**V1-10: RPL-evidence module for credentialing** — *new in v3.0; time-bound commercial wedge for the 30 June 2026 cliff.* Generates structured competency-evidence packs from longitudinal clinical work for APC RPL applications.

**V1-11: Multi-facility deployment** — same as v1.0.

**V1 exit criterion (revised):** A consultant pharmacy practice running 5 ACOPs across 8 RACHs sees recommendation acceptance rates climb from baseline toward +12pp target, with measurable improvement in fraction of recommendations whose clinical rationale survives the handoff. PHARMA-Care indicators produced automatically. Hospital discharge reconciliation operational. Dispensing pharmacy integrated for at least one major vendor. Pharmacist self-visibility live for all platform users; bottom-up adoption signal observable.

### V2 (Months 7–12)

**V2-1: AN-ACC Revenue Assurance (KB-28)** — supports multidisciplinary AN-ACC reassessment workflow by surfacing medication-complexity change signals. Honest framing: not a pharmacy-attributable revenue line; a workflow-support module.

**V2-2: Designated RN prescriber workflow** — role-aware UI for the new prescribing endorsement.

**V2-3: Tasmanian pharmacist co-prescribing pilot integration** — if Vaidshala becomes the digital substrate.

**V2-4: Victorian PCW exclusion compliance infrastructure** — legal-administration trail, regulator-queryable.

**V2-5: CPD module** — *new in v3.0.* Auto-tags CPD-eligible activities, surfaces cases for reflective writing, generates AHPRA-ready records.

**V2-6: Pharmacist career portfolio** — *new in v3.0.* Portable record across employers; supports pay negotiation, credential progression, professional recognition.

**V2-7: Clinical appropriateness audit module** — *new in v3.0.* Pattern detection for metric distortion; flags suppression patterns and persuasive-framing-of-marginal-recommendations to pharmacy employer and (with consent) to regulator.

**V2-8: Causality engine for ADE attribution** — same as v1.0.

**V2-9: Goals-of-care alignment** — expanded to Clinical state machine's `care_intensity` tag; reshapes every recommendation downstream.

**V2-10: Learning System surfaced selectively** — same as v1.0.

**V2-11: PHN regional dashboard (early)** — *new in v3.0.* Regional cohort visibility for PHN aged-care leads. Foundational for Buyer 5 commercial relationship.

**V2 exit criterion (revised):** A RACH operator sees AN-ACC reassessment-driven revenue recovery (multidisciplinary, not pharmacy-attributable), Standard 5 audit defensibility, measurable medication safety improvement, PHARMA-Care indicators automatically produced, Victorian PCW exclusion compliance evidence (where applicable), within 12 months of deployment. A pharmacy chain reports retention of pharmacist talent attributable to platform-supported career development. A PHN aged-care lead requests regional aggregation access.

---

## Part 13 — Go-to-market

### Three concrete commercial moves (carry forward from v2.0)

**Move 1: Engage the Tasmanian pharmacist co-prescribing pilot as digital substrate partner.** $5M state budget, 2026–2027 trial, structurally needs digital substrate. Contact Mohammed Salahudeen and Gregory Peterson (UTas), Duncan McKenzie (Tas DOH). Offer Vaidshala at no cost for pilot duration with publication and reference rights.

**Move 2: Engage the PHARMA-Care framework consortium for indicator alignment.** UniSA-led, $1.5M MRFF, evaluating $350M ACOP program. Contact Janet Sluggett and Sara Javanparast (ALH-PHARMA-Care@unisa.edu.au). Position Vaidshala as implementation partner producing PHARMA-Care indicators automatically.

**Move 3: Reframe the AMA/RACGP narrative.** Engage RACGP aged care interest group early. Position platform as visibly *strengthening* GP authority. Avoid bottleneck/delay/routing-around messaging.

### Three additional moves (new in v3.0)

**Move 4: Engage the ACQSC for evidence-quality recognition.** If the regulator publicly recognises the platform's evidence as audit-grade, every layer below adopts faster. This is a regulator-engagement strategy, not a sales effort. Engage through formal submissions on Standard 5 evidence requirements and through participation in any consultative bodies on aged-care medication management measurement.

**Move 5: Engage the Inspector-General of Aged Care.** Independent statutory office reporting to Parliament. Powers to investigate systemic issues. The platform's evidence substrate is exactly what an Inspector-General investigation would value. Position the platform as available to support Inspector-General work on medication-related systemic issues.

**Move 6: Engage the Australian Pharmacy Council on RPL pathway.** APC accredits ACOP credentialing including the RPL pathway. The platform's RPL-evidence module is structurally aligned with what APC will need to assess RPL applications. Engage APC on whether the platform's evidence packs can be referenced in RPL submission guidance. Time-bound: 12 months until grandfathering ends.

### Sequencing

Moves 1–3 are inexpensive and high-signal; should happen in parallel within 30–60 days. Moves 4–6 are longer-cycle (regulator engagement runs 6–18 months) but should be *initiated* in parallel; even early conversations create relationship substrate that pays back over 2–3 years.

---

## Part 14 — Risk register

### Risks 1–11 (carry forward from v1.0 + v2.0)

1. ADG 2025 licensing — assumed open access with appropriate citation; verify with UWA team
2. AMH licensing timeline — commercial relationship needed for V1
3. GP integration tooling fragility — Smart Form / structured input integration depends on prescriber-system support
4. Acceptance rate may not move as much as hoped — Ramsey baseline is 50%, +12pp target is meaningful but achievable
5. eNRMC vendor cooperation — 8 of 10 conformant per Nov 2025 status; FHIR-based integration plans
6. Restrictive practice legal exposure — consent gating on psychotropic recommendations is non-trivial
7. Jurisdictional regulatory fragmentation — VIC PCW exclusion may be first of several state changes
8. Designated RN prescriber rollout uncertainty — first cohort mid-2026; uptake unknown
9. Pharmacist autonomous prescribing timeline acceleration — 2027–2028 nationally; opportunity not threat but moving target
10. Hospital integration depth — V1 PDF discharge summary ingestion; V2 MHR; V3 ADT feeds
11. PHARMA-Care framework evolution — indicators may refine in active national pilot; build configurable

### New risks added in v3.0

**Risk 12: Algorithmic performance management legal exposure.** When platform-generated KPIs feed pay decisions, AHPRA professional accountability and Fair Work Commission emerging guidance both apply. Consultation, transparency, contestation pathways are required. Architectural commitment per Part 9 mitigates; legal review needed before any KPI-to-pay feature ships.

**Risk 13: Metric corruption — RIR optimisation distorting clinical practice.** If recommendation acceptance becomes the headline pharmacist KPI, the rational pharmacist response is to suppress recommendations predicted to be rejected, regardless of clinical appropriateness. The clinical appropriateness check (Part 7, Part 11) is the architectural guard. Operational guardrail: any pattern of recommendation suppression flags to the pharmacy employer for review.

**Risk 14: Pharmacist data-trust violation.** Pharmacists who perceive the platform as surveillance abandon it; bottom-up adoption motion collapses. Trust architecture (Part 9) must hold from the first deployment. Pharmacist-side data is pharmacist-controlled by default; aggregation upward requires permission; identifiable cross-comparison restricted. Any breach of this principle is a structural product failure, not a feature regression.

**Risk 15: Persuasive framing of inappropriate recommendations.** The recommendation craft engine could carry clinically marginal recommendations to acceptance through high-quality framing. Higher acceptance of bad recommendations is worse than lower acceptance of good ones. Architectural guard: appropriateness paired with acceptance, divergence flagged. Clinical safety review of any framing-engine output before broad deployment.

**Risk 16: Dispensing-pharmacy displacement risk for chains.** When a chain holds both ACOP and supply contracts at a RACH, an ACOP recommending deprescribing reduces dispensing revenue. The chain may be commercially conflicted in adopting platform features that maximise deprescribing recommendations. Mitigation: position platform's audit trail as conflict-of-interest *defence* for the chain, not as additional commercial risk. The chain that adopts the platform is the chain that demonstrates clinical-commercial separation.

**Risk 17: 30 June 2026 credentialing cliff exposure for buyers.** Pharmacy chains whose ACOP-eligible workforce has not credentialed by that date face workforce-availability risk. Some buyers may delay platform adoption while resolving credentialing; others may accelerate to capture the RPL-evidence wedge. Sales strategy must distinguish these two postures and tailor pitches accordingly. Time-bound advantage closes 30 June 2026.

**Risk 18: GP-college antagonism intensification.** AMA's response to Tasmania pilot is on record; RACGP's ACOP submission is on record. If the platform appears to enable non-GP prescribing at scale before college positions soften, college antagonism may harden. Mitigation per Move 3 (Part 13). The frame-vs-content principle (Part 9) prevents the platform from differentially supporting non-GP prescribers; framing adapts but content is invariant.

---

## Part 15 — What this means for v1.0 and v2.0 documents

**Vaidshala Final Product Proposal v1.0:** Superseded by v3.0. v3.0 incorporates all v1.0 content that remains valid (L0–L6 pipeline, KB-1 through KB-29 substrate, governance state machine, CompatibilityChecker, source attribution).

**Vaidshala Final Product Proposal v2.0 Revision Mapping:** Superseded by v3.0. v3.0 incorporates all v2.0 strategic positioning, five-state-machine substrate, four-role authority model, and three commercial moves. v2.0 is preserved as historical record of the April 2026 reframe.

**Layer 1 Australian Aged Care Implementation Guidelines** (will need consequential updates):
- All v2.0 additions remain valid
- Add: pharmacist self-visibility data access patterns
- Add: trust architecture / view-permission rules
- Add: RPL-evidence pack templates
- Add: CPD-tagging activity definitions

**Layer 2 Implementation Guidelines** (still pending; significantly more important after v3.0):
- Clinical state machine (running baselines, delta computation on write, transition events) — Layer 2 deliverable
- Pharmacist self-visibility dashboard architecture — Layer 2
- Trust architecture / permission-rule engine — Layer 2

**Layer 3 Rule Encoding Implementation Guidelines** (will need consequential updates):
- All v2.0 additions remain valid
- Add: recommendation craft engine — separate subsystem from CQL rule firing
- Add: clinical appropriateness check — paired with every recommendation rule
- Add: restraint signal generation — surfaced from substrate state, not a rule output
- Add: framing-vs-content separation in EvidenceTrace

---

## Part 16 — Closing

Three things to register as v3.0 closes.

**One:** The platform's strategic position is structurally clear and consistent across every dimension we've examined — architectural, regulatory, commercial, clinical-craft, ethical, workforce. v3.0 brings these into one document. The position survives stress-testing because every layer of value maps to a specific accountability mechanism the government has already specified or is currently specifying. Nothing in the platform's positioning is speculative.

**Two:** The five-state-machine substrate (v2.0) plus the recommendation craft engine, pharmacist self-visibility, and ethical architecture (v3.0) constitute a substantial engineering investment. Roughly 30–32 weeks across MVP and V1 to land the foundation. This is what makes the platform structurally different from CDS tools, EMRs, or aged-care platforms. Without it, Vaidshala is workflow tooling. With it, Vaidshala is the longitudinal evidence substrate for Australian aged-care medication management.

**Three:** The commercial windows are time-bound and several close in the next 12–14 months. The Tasmanian pilot digital-substrate decision; the PHARMA-Care national pilot framework alignment; the 30 June 2026 credentialing cliff and RPL-evidence wedge; the Victorian PCW exclusion compliance window (1 July 2026 effective, 90-day grace to 29 September 2026); the AMA/RACGP narrative shaping before positions ossify. The product Vaidshala is now positioned to build is structurally different from what v1.0 described, structurally extended from what v2.0 mapped, and structurally better-fit to the regulatory and clinical world that's arriving.

The reviewer thread that ran through the May 2026 strategic conversations closes the loop on this work. Every actor in the system is reviewer and reviewed; each performs their reviewing role with partial evidence today; the platform supplies the evidence substrate from which all of them can review better. The strategic moat is the substrate; the commercial moat is the multi-sided lock-in; the cultural moat is the trust architecture that protects each layer's data while enabling the whole pyramid to lift.

From here, pilot design proceeds. The next document specifies how Vaidshala enters its first deployments — site selection, baseline measurement, success criteria, evaluation methodology, contractual structure, and the operational playbook for proving the v3.0 thesis under real conditions.

— Claude
