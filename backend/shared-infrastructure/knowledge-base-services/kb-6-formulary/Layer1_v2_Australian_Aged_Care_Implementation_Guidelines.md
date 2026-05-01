# Layer 1 v2 — Australian Aged Care Clinical Knowledge Sources

**Version:** 2.0 — substantial revision of v1.0
**Date:** April 2026
**Status:** Implementation guidelines for the data layer that feeds Vaidshala's clinical reasoning continuity infrastructure

**Companion documents:**
- *Vaidshala Final Product Proposal v2.0 Revision Mapping* (the strategic context for this revision)
- *Layer 1 Australian Aged Care Implementation Guidelines v1.0* (being revised)
- *Layer 3 Rule Encoding Implementation Guidelines* (consumes from Layer 1)

**Audience:** Clinical informatics, data engineering, knowledge curation team

---

## Part 0 — What this document is and what changed

The original Layer 1 guidelines treated Vaidshala as a clinical decision support system with 27 content sources organised into 6 implementation waves. That framing was correct as far as it went, but it understated three things:

**One — Layer 1 is not just a content layer; it is the substrate for clinical reasoning continuity.** The five interlocking state machines (Authorisation, Recommendation, Monitoring, Clinical state, Consent) need source data of *kinds the original document didn't address*: prescribing agreements, credentials, hospital discharge summaries, dispensing pharmacy schedules, baseline observations, jurisdiction-aware regulatory rules. These aren't optional extensions; they're load-bearing for the v2.0 product.

**Two — the Australian regulatory landscape moved substantially between v1.0 and now.** The Aged Care Act 2024 commenced 1 November 2025; the Strengthened Quality Standards are in force; the Modernising My Health Record (Sharing by Default) Act 2025 received Royal Assent 14 February 2025 with mandatory pathology upload from 1 July 2026; the Victorian PCW exclusion legislation is live with grace period to 29 September 2026; the designated RN prescriber endorsement standard took effect 30 September 2025 with first endorsed prescribers expected mid-2026; the Tasmanian aged care pharmacist co-prescribing pilot is in development for trial 2026-2027; the $350M ACOP program requires mandatory APC training from 1 July 2026; the PHARMA-Care National Quality Framework is in active national pilot phase.

Each of these is now a Layer 1 source in its own right, not just a regulatory backdrop.

**Three — the Australian Deprescribing Guideline 2025** was published by UWA in September 2025 with 185 recommendations and 70 good practice statements, RACGP and ANZSGM endorsed, freely available at deprescribing.com under non-commercial reproduction terms. It's the most important single new content source for the Australian aged care product, and v1.0 didn't have it.

This v2 document supersedes v1.0. It covers what changed, what's new, and how the team should sequence the work.

---

## Part 1 — The reframed Layer 1 model

### 1.1 Three categories of source, not one

v1.0 treated all sources as clinical knowledge inputs (guidelines, drug references, terminology services). v2.0 categorises sources into three structurally different types:

**Category A — Clinical Knowledge Sources.** Guidelines, drug references, formularies, terminology services. These feed the Recommendation and Decision layers (KB-1, KB-4, KB-5, KB-6, KB-16, KB-20). This is what v1.0 was about.

**Category B — Patient State Sources.** Real-time data flows about specific residents — eNRMC medication chart, pathology, discharge summaries, nursing observations, MHR continuity data, dispensing pharmacy DAA schedules. These feed the Clinical state machine (running baselines, transition events, active concerns) and the Recommendation engine's trigger surface. Category B is mostly Layer 2 work in v2.0, but it has significant Layer 1 components (the *source definitions* and *ingestion contracts*).

**Category C — Regulatory and Authority Sources.** Quality Standards, restrictive practice rules, prescribing agreements, credentials, jurisdiction-specific scope rules. These feed the Authorisation state machine and the Consent state machine. Almost entirely new in v2.0; v1.0 didn't have this category.

The team should track these separately because they have different update cadences (Category A: months to years; Category B: real-time; Category C: months to years but legally consequential), different quality requirements (A: clinical-evidence-graded; B: provenance-and-timestamp-critical; C: jurisdiction-and-validity-audited), and different KB consumption patterns.

### 1.2 The Source Registry, extended

v1.0 specified a Source Registry with truth-source records carrying identity, location, format, update cadence, license, scope, and trust tier. That structure is correct and continues. v2.0 extends each record with:

- **Category** (A / B / C)
- **State machine consumed by** (Authorisation / Recommendation / Monitoring / Clinical state / Consent — most sources feed multiple)
- **Jurisdiction scope** (national / state / territory / facility-specific) — important because Victorian PCW exclusion rules don't apply outside Victoria
- **Effective period** (start_date, end_date, replaces, replaced_by) — because regulations change and we need version-aware source consumption
- **Authority tier** (1=primary regulator/legislature; 2=peak professional body; 3=academic/research; 4=facility-level policy)
- **Reproduction terms** (free reproduction / non-commercial / licensed / proprietary)

This extension means the Source Registry becomes the operational database that drives all downstream knowledge management, not just a content reference.

---

## Part 2 — Category A: Clinical Knowledge Sources (the v1.0 content, refreshed)

This part covers what was in v1.0 with corrections, additions, and licensing clarifications.

### 2.1 The v1.0 sources that hold up

These continue largely as v1.0 specified, with minor updates:

**STOPP/START v3** (O'Mahony et al. 2023, Eur Geriatr Med). 190 criteria. DOI 10.1007/s41999-023-00777-y. **Reproduction terms**: published under standard journal terms; the criteria themselves are factual clinical statements that can be encoded as rules with proper attribution, consistent with established commercial CDS practice (same posture as Beers and other rule sets).

**Australian PIMs 2024** (Wang et al., Internal Medicine Journal). DOI 10.1111/imj.16322. Australian-specific PIM list. **Reproduction terms**: standard journal terms; same posture as STOPP/START.

**AGS Beers Criteria 2023.** Used widely in Australian clinical practice despite US origin. **Reproduction terms**: AGS holds copyright; commercial use requires licensing for reproduction of the text, but the underlying clinical determinations (which drug is a PIM in which context) are factual and have been encoded in dozens of commercial CDS systems globally with proper attribution.

**TGA Product Information.** Updated continuously per product. **Reproduction terms**: TGA-managed, generally free for reference but commercial reproduction restrictions apply to specific manufacturer texts.

**PBS Schedule.** Monthly updates. **Reproduction terms**: government-published, free for reference.

**KDIGO, AHA/ACC, ADS-ADEA, ESC** disease-specific guidelines. **Reproduction terms**: vary by guideline body; standard posture as above.

### 2.2 The major addition v1.0 didn't have

**Australian Deprescribing Guideline 2025** (Quek/Page/Etherton-Beer/Lee, UWA, September 2025).

This is the most important new content source for the product. 185 consensus-based recommendations + 70 good practice statements, structured into four areas (when to deprescribe, ongoing treatment needs, how to deprescribe, monitoring requirements). RACGP and ANZSGM endorsed. Australian-specific. Freely available at deprescribing.com.

**Reproduction terms (verified directly from the guideline):** "You may reproduce this work, in whole or in part, in its original, unaltered form for your own personal use or, if you are part of an organisation, internal use within your organisation, provided that: 1) The reproduction is not used for commercial purposes."

**The licensing posture is genuinely contested for a commercial CDS product:**

A strict reading says commercial CDS is exactly what the non-commercial clause excludes. A more permissive reading says the recommendations are factual clinical statements (similar to Beers, STOPP/START) and encoding equivalent rules in our own words with proper attribution is established commercial practice that the non-commercial clause doesn't preclude.

**My recommendation:** before any production CQL is authored that's traceable to ADG 2025, the team gets:
- A formal legal opinion (commercial CDS counsel, not generic IP)
- A direct conversation with the UWA team (Quek, Page, Etherton-Beer, Lee) to clarify intent and explore licensing paths
- Either a written license agreement or written confirmation from UWA that the team's intended use does not require one

The cost of getting this wrong is a copyright dispute that destroys credibility with Australian clinical and regulatory stakeholders. The cost of getting it right is a meeting and a written legal position. The risk asymmetry is very large; do not skip this step.

If licensing or written clarification is not achievable, the fallback is to encode rules from STOPP/START v3, AGS Beers 2023, and Australian PIMs 2024 (all with established commercial CDS encoding precedents) and reference ADG 2025 only where its recommendations align with these sources, never as the sole evidence basis.

### 2.3 AMH Aged Care Companion

Australian gold-standard reference for aged care medication, biennial print plus annual online updates. **Commercial license required** — AMH negotiates licenses individually. License negotiations typically take 3-9 months.

**Recommendation:** start license negotiation now (P0 commercial action). The product can launch MVP without AMH content (substituting STOPP/START + ADG + Beers + Australian PIMs) and integrate AMH as a content upgrade in V1 or V2 once licensed. **Do not block MVP on AMH licensing.** Do not launch claiming "AMH-based" content until the license is signed.

### 2.4 KDIGO, ESC, ADS-ADEA, AHA/ACC, ANZSGM updates

These disease-specific guideline bodies publish on independent cadences. The Source Registry tracks each with its own update process. None require licensing changes from v1.0.

**One addition worth flagging:** the ANZSGM (Australia and New Zealand Society for Geriatric Medicine) endorsement of ADG 2025 and the existence of the ANZSGM-published *Position Statements* on specific aged care medication topics make ANZSGM a secondary authority source the team should track. Position statements update faster than guidelines and are aged-care-specific.

### 2.5 The "failure-mode learning" sources

v1.0 mentioned these briefly. v2.0 should be explicit:

**Coronial findings (state coroners' courts).** Australian state coroners regularly publish findings on medication-related deaths in aged care. These are publicly available, jurisdiction-specific, and provide failure-mode evidence that no guideline captures. **Quarterly review cadence.** Findings feed into rule additions, suppression refinements, and audit-trail design.

**ACQSC compliance and complaint reports.** The Aged Care Quality and Safety Commission publishes Sector Performance Reports quarterly and individual non-compliance findings. **Quarterly review cadence.**

**ACSQHC clinical care standards and audit reports.** The Australian Commission on Safety and Quality in Health Care publishes the Psychotropic Medicines in Cognitive Disability or Impairment Clinical Care Standard (relevant to aged care), the Stewardship Framework for Medication Management at Transitions of Care (2024), and audit reports. **Continuous monitoring.**

These three sources together tell the team what's actually going wrong in Australian aged care medication management — which is what the rule library should be designed to prevent.

---

## Part 3 — Category B: Patient State Sources (mostly new in v2.0)

This is where v2.0 differs most from v1.0. The Clinical state machine, the Authorisation state machine, and the Recommendation engine's trigger surface all need real-time data flows from sources v1.0 didn't address.

### 3.1 eNRMC integration (refined from v1.0)

v1.0 treated eNRMC as a data source. v2.0 reframes it correctly:

**eNRMC is the legal medication execution ledger, not just a data source.**

Only medications present on the eNRMC may legally be administered in the RACF. Vaidshala consumes from eNRMC; we do not write authoritative prescribing data back. This boundary is non-negotiable.

**Status as of April 2026:** 8 of 10 eNRMC vendors are conformant for electronic prescribing per the November 2025 status tracker. Implementation deadline for RACFs to adopt conformant systems remains 31 December 2026. Vendor conformance has been extended to 1 April 2026 for the remaining two vendors.

**What Vaidshala ingests from eNRMC:**
- Current MedicationRequest list (active prescriptions)
- MedicationAdministration events (what was actually given when)
- Medication changes (start, stop, dose change events)
- Prescriber identity (who authorised the change)
- Indication free-text (often empty, but where present, valuable)

**What Vaidshala does NOT ingest from eNRMC and shouldn't try to:**
- The legal authoritative chart (that's the eNRMC's role)
- Prescribing decisions made within eNRMC (we sit on top, not within)

**Integration approach by vendor:**

| Approach | When | Effort |
|---|---|---|
| **Direct FHIR R4 API** (if vendor exposes one) | Conformant vendors with mature APIs | 4-8 weeks per vendor |
| **HL7 v2 ORM/RDE messaging** (legacy) | Some conformant vendors | 6-10 weeks per vendor |
| **CSV export + nightly sync** | All vendors as fallback | 2-4 weeks per vendor |
| **Direct database read** | Where vendor cooperates | 4 weeks per vendor (but creates dependency) |

**Sequencing recommendation:** MVP supports CSV export for any facility. V1 adds FHIR R4 API integration with the top 2-3 conformant vendors (likely Telstra Health MedPoint, MIMS, ResMed Software). V2 expands to all conformant vendors.

### 3.2 Pathology — significantly easier in v2.0 because of MHR Sharing by Default

This is the change v1.0 didn't anticipate.

**Until July 2026:** Pathology integration was per-vendor, per-facility, per-pathology-provider — fragmented, expensive, and slow. v1.0 specified HL7 v2 ORU integration as the canonical approach.

**From 1 July 2026:** Pathology providers must upload pathology and diagnostic imaging reports to My Health Record by default for all consumers (with limited exceptions). Civil penalties apply for non-compliance. Medicare benefits may be recovered for non-compliant services.

**The Layer 1 implication:** for any RACF resident with an MHR, pathology results become available through a single integration point (MHR FHIR Gateway) rather than per-pathology-provider integration. The data flow becomes:

```
Pathology lab → MHR (mandatory upload from July 2026) → MHR FHIR Gateway → Vaidshala
                                                       ↓
                                           Most reports available immediately
                                           5-day delay for anatomical pathology,
                                           cytopathology, genetic tests
```

**This is genuinely transformational for the product.** v1.0's pathology integration estimate (~6-10 weeks per pathology vendor, per facility) is replaced by a single MHR FHIR Gateway integration that serves all facilities and all consenting residents.

**Caveats the team should hold:**

- **Consent.** Residents (or their substitute decision-makers) can request specific reports not be uploaded, can hide or restrict access after upload. Vaidshala must respect MHR access controls.
- **Coverage.** Residents without an MHR (small but non-zero — 14% of Australians as of 2024) have no MHR pathology available. The product needs CSV/HL7 fallback for these residents.
- **Latency.** Most pathology available immediately; some types have 5-day delay. This affects how quickly Layer 2 trigger surfaces fire on abnormal results.
- **MHR API maturity.** Production MHR integration is currently SOAP/CDA; FHIR Gateway is the modern path but still maturing. The Australian Digital Health Agency publishes the FHIR Implementation Guide v1.4.0 (R4) for the FHIR Gateway. **Plan for SOAP/CDA in V1 with FHIR transition in V2.**

### 3.3 Hospital discharge summaries — the highest-yield channel v1.0 missed

The original synthesis documents made this clear: hospital discharges are the highest-risk medication moments. >50% of medication errors occur at transitions of care. The original Layer 1 didn't have hospital discharge as a first-class source.

**v2.0 adds this as a Wave 2 priority.**

**Source structure:**
- **Hospital discharge summary** — currently mostly PDF, increasingly available as MHR-uploaded structured documents. The Australian Digital Health Agency's MHR profile includes discharge summaries.
- **Pre-admission medication chart** — from the RACF eNRMC at admission time
- **GP notes during admission** — from MHR or direct GP system integration
- **Hospital prescribing data** — varies by hospital; some have MHR upload, some don't

**The reconciliation challenge:** discharge summaries use generic + brand drug names; eNRMC uses AMT (Australian Medicines Terminology) codes; GP systems use MIMS or other coding. Three different coding systems, three different timestamp regimes, three different signers. Reconciling these is genuinely hard work — probably 6-8 weeks of focused engineering for a robust v1 implementation.

**Sequencing recommendation:**
- **MVP:** discharge summary PDF upload + manual reconciliation interface for the pharmacist
- **V1:** MHR-pulled structured discharge documents + auto-reconciliation against eNRMC + change-flagging + ACOP routing within 24 hours
- **V2:** direct hospital ADT (Admission, Discharge, Transfer) feed integration where available; this is hard to obtain because hospital integration is jurisdiction-specific and politically complex

This is also where the ACSQHC Stewardship Framework (Medication Management at Transitions of Care, published 2024) becomes the canonical reference. The platform's hospital discharge reconciliation workflow should explicitly produce evidence aligned to this Framework.

### 3.4 Dispensing pharmacy DAA timing — the gap that nobody else is closing

v1.0 didn't address this at all. v2.0 makes it a first-class Category B source.

**The problem:** the dispensing community pharmacy packs the resident's Dose Administration Aid (DAA) on a weekly cycle (typically Saturday or Sunday). When the GP approves a medication change Monday at 0900, the change does not reach the resident's body until either:
- The DAA is unpacked (a known nursing time-cost), or
- The next packing cycle (up to 6 days later)

Almost no platform models this. v2.0 makes the dispensing pharmacy a first-class execution actor and ingests:
- DAA packing schedule (per resident, per pharmacy)
- Dispensing events (when meds are physically supplied to the facility)
- DAA composition (what's actually in this week's pack)
- Supply delays / partial pack changes / urgent re-dispensing events

**Integration approach:** Australian community pharmacy software is a fragmented market — FRED, Z Solutions, Minfos, LOTS, Aquarius, others. None has a clean modern API for this. Realistic posture:

- **MVP:** structured cessation/change alerts to dispensing pharmacy via fax/email/portal; manual DAA timing entry by pharmacist
- **V1:** API integration with the top 2-3 vendors (FRED is the largest); DAA packing schedule as state
- **V2:** broader vendor coverage; full DAA composition tracking

**Why this matters strategically:** owning the dispensing pharmacy coordination layer is unfashionable but defensible. The community pharmacy is structurally inside the ACOP Tier 1 model already (Tier 1 ACOP is a community pharmacy claim and engages an ACOP through the community pharmacy). A platform that gets dispensing pharmacies adopting it — even passively — gets a network effect that no on-site-only platform can match.

### 3.5 Nursing observations and behavioural notes

v1.0 mentioned this peripherally. v2.0 makes it explicit because the Clinical state machine depends on it.

**Sources:**
- **eMAR** (electronic Medication Administration Record) — administration events, refusals, PRN use, missed doses
- **Care management system** (Leecare, AutumnCare, Person Centred Software, Mirus Australia, etc.) — structured observations: vital signs, weight, mobility scores, behavioural events, falls, infections
- **Nursing free-text progress notes** — clinical narrative; harder to use but rich
- **Behavioural charts** for residents on antipsychotics — required under restrictive practice regulations; structured data about target behaviours, frequency, intensity

**The Australian aged care care management system market is fragmented** with no single dominant vendor. Integration is per-vendor and effortful.

**Sequencing:**
- **MVP:** CSV export of structured observations from one or two pilot facilities' systems
- **V1:** API integration with the top 2-3 care management vendors
- **V2:** broader vendor coverage; NLP on free-text progress notes for OTC/complementary medicine detection and behavioural change signals

**One critical clinical point:** the running baseline computation that powers the Clinical state machine depends on having structured observations over time. CSV monthly snapshots are insufficient — the platform needs at minimum daily observation flows for vital signs, behavioural events, mobility, weight. **This may be the single hardest data engineering problem in Layer 1+2.**

### 3.6 NCTS and Australian terminology

v1.0 specified AMT (Australian Medicines Terminology), SNOMED-CT-AU, LOINC AU, ICD-10-AM via the National Clinical Terminology Service (NCTS). This continues unchanged.

**Action items still pending from v1.0:**
- NCTS account application (the team has been pending this; it gates all of KB-7)
- AMT bulk download and integration
- SNOMED-CT-AU subset for aged care (pruned from full SNOMED-CT-AU)
- LOINC AU subset for typical RACF pathology

These remain Wave 1 priorities. They block KB-7 (Terminology) which is currently empty. Without KB-7 populated, all rule encoding is degraded because drug-class membership and condition-code matching rely on it.

---

## Part 4 — Category C: Regulatory and Authority Sources (entirely new in v2.0)

This category didn't exist in v1.0. It's the substrate for the Authorisation and Consent state machines.

### 4.1 Aged Care Act 2024 and Strengthened Quality Standards

**Status (verified April 2026):** Aged Care Act 2024 commenced 1 November 2025 (delayed from original 1 July 2025). Seven Strengthened Quality Standards in force. Aged Care Rules 2025 are 900+ pages, released iteratively from September 2024.

**The seven strengthened standards** (relevant for the platform):
1. The Individual
2. The Organisation
3. The Care and Services
4. The Environment
5. **Clinical Care** (most relevant for medication management)
6. Food and Nutrition
7. The Residential Community

**Standard 5 (Clinical Care) is the primary driver of platform-facility alignment.** It requires evidence of safe medication management, governance of clinical care, restrictive practice oversight, and continuous quality improvement. The platform's audit trail and EvidenceTrace graph are designed to produce Standard 5 evidence as workflow exhaust.

**Layer 1 implication:** the Aged Care Rules 2025 sections relevant to medication management need to be parsed into the platform as structured rule data, not just referenced. Specifically:
- Restrictive practice authorisation requirements (consent state machine input)
- Clinical governance requirements (audit trail design)
- Quality Indicator Program reporting requirements (KB-13 inputs)
- Worker screening requirements from mid-2026 (authorisation state machine input)

**Update cadence:** Aged Care Rules 2025 will iterate. The Source Registry must track Rule version and effective date. Rule changes propagate to the platform's regulatory rule engine.

### 4.2 Quality Indicator (QI) Program requirements

**Status:** Mandatory reporting in Australian residential aged care since 2019, expanded scope. Currently includes (relevant to medication):
- **Antipsychotic medication use** — measured through 7-day medication chart and/or administration record review every quarter
- **Polypharmacy** — % of residents prescribed 9+ medications
- **Falls and major injury** — quarterly
- **Pressure injuries**, **physical restraint**, **unplanned weight loss** — also reported

**Platform implication:** the QI Program indicators should be produced *automatically* as workflow exhaust, not as separate quarterly reports. KB-13 (Quality Measures) is responsible for this.

**Key clinical detail:** the antipsychotic indicator measures use, not appropriateness. A facility with high antipsychotic use that's all clinically justified looks the same in the indicator as one with widespread inappropriate use. The platform produces the indicator AND the *justification analysis* — which residents are on antipsychotics, what's the documented BPSD trial, what's the consent status, what's the deprescribing review status. This is what RACH operators actually need for Standard 5 audit defensibility.

### 4.3 Restrictive practice / chemical restraint legislation

**Status:** Legislation in force since 2019. Aged Care and Other Legislation Amendment (Royal Commission Response) Act 2022 and subsequent regulations require:
- Documented best-practice behaviour supports trial *before* psychotropics
- Informed consent (resident or substitute decision-maker)
- Documentation of decisions to use restraint
- Regular review and monitoring
- Reporting under the Quality Indicator Program

**Platform implication:** any deprescribing recommendation that touches a medication currently authorized as a restrictive practice must:
- Surface the restrictive practice authorisation status
- Flag concurrent behaviour-support plan review requirement
- Not recommend simple cessation without behavioural plan review
- Track consent state separately from medication state

The KB-29 template for psychotropic deprescribing recommendations needs explicit "this medication is currently authorised as a restrictive practice; deprescribing requires concurrent behaviour-support plan review" language.

### 4.4 Victorian PCW exclusion legislation (jurisdiction-specific)

**Status (verified April 2026):** Drugs, Poisons and Controlled Substances Amendment (Medication Administration in Residential Aged Care) Act 2025 passed September 2025. Commences 1 July 2026, with 90-day grace period to 29 September 2026 before enforcement. Applies in Victoria only.

**What changes:** From 1 July 2026, only registered nurses, enrolled nurses, pharmacists, or medical practitioners may administer Schedule 4, 8, and 9 medications and drugs of dependence to residents who do not self-administer their own medication. PCWs may continue to assist competent self-administering residents. This includes antibiotics, opioid analgesics, benzodiazepines, and clinical trial medications.

**Platform implication:** the Authorisation state machine must be jurisdiction-aware. ScopeRules for Victorian RACFs must encode this restriction; ScopeRules for other states do not. The Authorisation evaluator must check (jurisdiction × role × medication schedule × resident self-administration status) at every administration attempt.

**Strategic implication:** other states will likely follow Victoria. ANMF branches in NSW, QLD, and SA have advocated for similar restrictions. The platform's ScopeRules architecture must be data-not-code so additional jurisdictions can be added without engineering work.

### 4.5 Designated RN prescriber endorsement

**Status (verified April 2026):** NMBA Registration Standard: Endorsement for Scheduled Medicines — Designated Registered Nurse Prescriber took effect 30 September 2025. Enables endorsed RNs to prescribe Schedule 2, 3, 4, and 8 medicines in partnership with an authorised health practitioner under a prescribing agreement, within their scope of practice and per relevant state/territory legislation. Eligibility: 5,000 hours clinical experience in past 6 years, NMBA-approved postgraduate qualification, six-month clinical mentorship post-endorsement.

**Education program status:** ANMAC commenced assessment of education programs July 2025. No NMBA-approved units of study available as of late 2025. First endorsed prescribers expected mid-2026.

**Platform implications (significant):**

The Authorisation state machine must track for each potential designated RN prescriber:
- **Credential**: Endorsement valid_from, valid_to, evidence_url
- **PrescribingAgreement**: linked to authorised health practitioner, scope (medicine classes, residents covered), validity period, mentorship_status (active/complete/breached), signed_packet_url
- **MentorshipStatus**: complete (post-six-month) vs in-progress
- **ScopeMatch**: per-action verification that the proposed action falls within the prescribing agreement's scope

These are PDFs in shared drives today. The platform converts them to structured data with audit trail.

**Why this matters strategically:** the platform that maintains structured prescribing-agreement and credential data with audit query API has built **the new safety primitive that almost nobody else is building**. Regulators will be watching designated RN prescribing very closely from mid-2026. A platform that can answer "show me every Schedule 4 prescription by RN X in Q3 2026 and the prescribing agreement that authorised each one" in structured form has a defensible commercial position.

### 4.6 Tasmanian aged care pharmacist co-prescribing pilot

**Status (verified April 2026):** Tasmania's 2025-26 Budget includes $5M for pharmacist scope expansion, including an Australian-first aged care pharmacist collaborative prescribing pilot. Development late 2025, 12-month trial through 2026 and 2027. Pharmacists prescribe medication to Tasmanian aged care residents in collaboration with their GP, per a treatment plan. UTas Pharmacy School (Salahudeen, Peterson, Curtain) likely involved.

**Platform implications:**

The Authorisation state machine must support a new role: **Pharmacist co-prescriber (Tasmanian pilot only)**. ScopeRules for this role:
- Jurisdiction: Tasmania only
- Validity period: pilot duration (2026-2027)
- Authorised medication classes: per pilot design (likely deprescribing, dose adjustments, specific class additions per agreed treatment plan)
- Co-prescription requirement: GP authorisation per treatment plan

**Strategic implication:** the pilot needs a digital substrate to track pharmacist-GP co-prescribing per treatment plan. This is the most natural Vaidshala partnership opportunity in Australia. **Action item from the v2 product proposal:** engage UTas (Salahudeen, Peterson) and Tasmanian Department of Health Pharmacy Projects team within 30-60 days. Timing window closes mid-2026.

### 4.7 ACOP credentialing and APC training requirements

**Status (verified April 2026):** From 1 July 2026, all pharmacists participating in the $350M ACOP measure must have completed an APC-accredited aged care on-site pharmacist training program. Tier 1 (community pharmacy claims) operates since July 2024 at 1 FTE per 250 beds; daily rate AUD 619.84/day per FTE.

**Platform implications:**

The Authorisation state machine must track for each ACOP pharmacist:
- **Credential**: APC training completion (valid_from, valid_to, evidence)
- **ACOP measure participation**: which Tier (1 or 2), per-facility allocation
- **Bed-allocation**: how many residents this pharmacist is credentialed for at this facility

The system should refuse to attribute ACOP-billable activity to a pharmacist without current APC credential evidence. This is both a safety primitive and a billing-defensibility primitive.

### 4.8 PHARMA-Care National Quality Framework

**Status (verified April 2026):** UniSA-led (Sluggett), $1.5M MRFF-funded, 14 project partners, PSA-endorsed, in active national pilot phase as of November 2025. Formally evaluating the $350M ACOP program. EOI open for aged care providers and on-site pharmacists at ALH-PHARMA-Care@unisa.edu.au.

**Platform implications:**

PHARMA-Care defines the *quality indicators* by which ACOP services are evaluated. The framework structures evaluation across five domains relating to medication management. **A platform that produces these indicators automatically as workflow exhaust has a structural commercial advantage** because it makes the buyer's purchase justification for them.

The framework is in active pilot phase, so indicator definitions may refine. **Build the platform's indicator computation as configurable, not hardcoded.** Re-evaluate alignment quarterly with the UniSA team.

**Strategic action item:** engage Janet Sluggett and Sara Javanparast at UniSA within 30-60 days to position Vaidshala as both a deployed platform in the pilot and a measurement substrate. Reference the v2 product proposal for the full positioning.

### 4.9 ACSQHC Stewardship Framework

**Status:** Australian Commission on Safety and Quality in Health Care published Medication Management at Transitions of Care Stewardship Framework in 2024. Four elements: effective communication, person-centred care, digital enablers, governance.

**Platform implications:** the framework provides the structural anchor for the v2 product positioning ("medication stewardship infrastructure for aged care"). The platform's hospital discharge reconciliation workflow, monitoring lifecycle, and audit trail should explicitly produce evidence aligned to this framework.

**Update cadence:** ACSQHC frameworks update on multi-year cycles. Source Registry tracks framework version.

### 4.10 Modernising My Health Record (Sharing by Default) Act 2025

**Status:** Royal Assent 14 February 2025. Pathology and diagnostic imaging mandatory upload from 1 July 2026. Civil penalties for non-compliance (250 penalty units = AUD 82,500 for non-registration; 30 penalty units = AUD 9,900 for non-compliant upload). Framework extends to other information types over time.

**Platform implications (significant):**

This is the regulatory backbone behind the pathology integration simplification described in section 3.2. Beyond pathology:
- The framework can extend to discharge summaries, GP notes, specialist letters, immunisation records — any health information type
- "Sharing by default" creates a presumption of MHR availability for all consenting consumers
- Vaidshala benefits from this without doing per-source integration work, but must respect MHR access controls and consent

**Action item:** monitor the Department of Health's "Better and Faster Access to health information" pages quarterly for Sharing by Default Rule extensions. Each extension is potentially a new Layer 1 source.

---

## Part 5 — The implementation waves, revised

v1.0 had six implementation waves. v2.0 reorganises around the new categories and the regulatory timeline.

### Wave 1 — Foundation (Weeks 1-4)

**Category A:** STOPP/START v3, AGS Beers 2023, Australian PIMs 2024 ingested into KB-1/KB-4. ADG 2025 prepared for ingestion pending licensing review.

**Category B:** NCTS account application processed; AMT, SNOMED-CT-AU, LOINC AU bulk download into KB-7 (foundation gap closure).

**Category C:** Aged Care Act 2024 + Strengthened Quality Standards parsed into structured regulatory data; Restrictive Practice regulations parsed into Consent state machine inputs; Victorian PCW exclusion ScopeRules drafted (data not code, ready for July 2026 activation).

**Wave 1 exit criterion:** KB-7 populated; KB-1/4 have 200+ rule sources; regulatory rule engine has Aged Care Act 2024 + Standard 5 + Restrictive Practice + Victorian ScopeRules as data.

### Wave 2 — Patient state plumbing (Weeks 5-12)

**Category B:** eNRMC integration via CSV (MVP); MHR FHIR Gateway integration for pathology (preparing for July 2026 mandatory upload); hospital discharge summary PDF ingestion + reconciliation interface; eMAR / care management system CSV export from pilot facilities.

**Category C:** ACOP credential ledger; designated RN prescriber credential structure (preparing for mid-2026 first cohort); PHARMA-Care indicator definitions encoded as KB-13 measures.

**Wave 2 exit criterion:** Layer 2 has running baselines for pilot residents from at least one pilot facility; pathology flows from MHR FHIR Gateway are operational for residents with MHR; hospital discharge reconciliation produces structured change-flag output.

### Wave 3 — Clinical knowledge depth (Weeks 13-20)

**Category A:** ADG 2025 ingestion (post-licensing); KDIGO, AHA/ACC, ESC, ANZSGM disease-specific guidelines for KB-1; coronial findings + ACQSC compliance reports + ACSQHC audit reports as failure-mode learning sources.

**Category C:** Tasmanian pilot ScopeRules drafted (preparing for 2026-2027 trial); ACOP APC training credential infrastructure operational.

**Wave 3 exit criterion:** rule library covers ~200 production rules across STOPP/START + Beers + Australian PIMs + ADG 2025 + KDIGO/AHA/ESC subsets; failure-mode learning loop active.

### Wave 4 — Dispensing pharmacy + advanced patient state (Weeks 21-28)

**Category B:** Dispensing pharmacy DAA timing integration for top 2-3 vendors; behavioural chart structured ingestion for psychotropic-treated residents; NLP on nursing progress notes for OTC/complementary medicine detection.

**Category C:** Pharmacist co-prescriber role infrastructure operational (Tasmanian pilot ready); Authorisation evaluator <500ms p95 latency target.

**Wave 4 exit criterion:** dispensing pharmacy DAA latency surfaced to ACOP for at least one pilot pharmacy; behavioural data flows operational; Authorisation evaluator handling production query load.

### Wave 5 — AMH integration + breadth expansion (Weeks 29-40)

**Category A:** AMH Aged Care Companion integration (post-licensing); ANZSGM Position Statements as secondary sources; expanded coverage of disease-specific guidelines.

**Category B:** Direct hospital ADT feed integration where partnerships allow; broader eNRMC vendor coverage (4-6 vendors); broader care management system integration.

**Category C:** Multi-jurisdiction ScopeRules (preparing for non-Victorian states adopting PCW exclusion); QI Program automated reporting at scale.

**Wave 5 exit criterion:** Layer 1 covers ~300 rules; data ingestion at scale across 10+ facilities; multi-jurisdiction support live.

### Wave 6 — Continuous tuning (ongoing from Week 41)

The work shifts to continuous improvement: source update tracking (7-day SLA from publication to deployed rule); coverage audit (monthly, comparing rule fires against expected prevalence); failure-mode mining from coronial findings and ACQSC reports; PHARMA-Care framework alignment as the framework refines.

This is where the moat compounds. Without continuous tuning, the rule library degrades within 12 months.

---

## Part 6 — Revised cost and effort estimates

v1.0 estimated 5-7 weeks and AUD 25-37 in API costs for the six waves. **That estimate covered Category A only.** v2.0 spans more categories and more sources.

**Realistic effort estimates (calendar weeks, 2-3 dedicated authors):**

| Wave | Effort | API + integration costs |
|---|---|---|
| 1. Foundation | 4 weeks | AUD ~50 in API + NCTS account (free) |
| 2. Patient state plumbing | 8 weeks | AUD ~200 + per-vendor integration costs (variable) |
| 3. Clinical knowledge depth | 8 weeks | AUD ~150 |
| 4. Dispensing pharmacy + state | 8 weeks | per-vendor integration |
| 5. AMH + breadth expansion | 12 weeks | AMH license (negotiate) + per-vendor |
| 6. Continuous tuning | Ongoing | AUD ~50/month |
| **Total to comprehensive Layer 1** | **~40 weeks** | **AUD ~500 + license + integration** |

This is significantly more than v1.0's estimate but is honest about the scope. The team should sequence ruthlessly: Wave 1 + Wave 2 + Wave 3 cover MVP (about 20 weeks); Waves 4-5 cover V1 and V2.

**Critical cost item that v1.0 underestimated:** AMH Aged Care Companion commercial licensing. AMH licenses are negotiated individually and can range from low five figures to mid six figures annually depending on use case. **Start negotiation now; do not assume a number until the licensee has quoted.**

---

## Part 7 — The three commercial actions Layer 1 work depends on

These are referenced in the v2 product proposal and bear repeating here because they directly affect Layer 1 source availability:

**Action 1: ADG 2025 licensing review.** P0 commercial action. Either secure a written license or written confirmation from UWA that the team's intended use does not require one. **Block any production CQL traceable to ADG 2025 until this is resolved.**

**Action 2: AMH Aged Care Companion license negotiation.** P0 commercial action. 3-9 month timeline. Start now.

**Action 3: NCTS account application.** P0 technical action. Free, but processing time is weeks. Without this, KB-7 cannot be populated and the rule library is degraded.

**Action 4 (if pursuing the Tasmanian pilot partnership):** UTas Pharmacy School (Salahudeen, Peterson, Curtain) and Tasmanian Department of Health Pharmacy Projects team engagement. Window closes mid-2026.

**Action 5 (if pursuing PHARMA-Care alignment):** UniSA team (Sluggett, Javanparast) engagement at ALH-PHARMA-Care@unisa.edu.au. Window is currently open.

---

## Part 8 — What's hardest, and what to defer

Three pieces of Layer 1 work are genuinely hard and the team should not underestimate them:

**Hardest 1: Hospital discharge reconciliation.** Three coding systems, three timestamp regimes, three signers. Currently mostly PDF. Plan for 6-8 weeks of focused engineering for v1; do not underestimate.

**Hardest 2: Real-time observation flows from care management systems.** The Australian aged care care management vendor market is fragmented; integration is per-vendor. Daily-frequency observation flows are needed for the Clinical state machine's running baselines. This may be the single hardest data engineering problem in Layer 1+2.

**Hardest 3: Dispensing pharmacy DAA timing.** Fragmented vendor market, no clean APIs, but the data is genuinely valuable. Defer to Wave 4 but do not skip.

**What to defer (without losing the product):**

- **Direct hospital ADT integration**: defer to V2/V3. PDF + MHR-pulled discharge documents cover MVP and V1.
- **Broader eNRMC vendor coverage**: MVP supports CSV; V1 covers top 2-3 conformant vendors; V2 broadens.
- **NLP on free-text progress notes**: defer to Wave 4-5. Structured behavioural data covers most of the value.
- **AMH Aged Care Companion integration**: defer pending license. Substitute STOPP/START + Beers + Australian PIMs + ADG (post-licensing) for MVP.
- **Multi-jurisdiction ScopeRules expansion**: build the infrastructure in Wave 1; activate per-jurisdiction as legislation passes.

---

## Part 9 — Closing

Three things to register before this document leaves the room.

**One:** Layer 1 in v2.0 is a structurally larger scope than v1.0. The shift from "clinical knowledge sources" to "the substrate for clinical reasoning continuity infrastructure" expands what we ingest, why we ingest it, and how it's organised. The team should expect this to be roughly 40 weeks of work to comprehensive coverage, not 5-7 weeks. Most of that work pays off in V1 and V2; Wave 1+2+3 (~20 weeks) covers MVP.

**Two:** the regulatory timeline is creating a window. Mandatory eNRMC by 31 December 2026, mandatory MHR pathology upload from 1 July 2026, mandatory APC training for ACOP from 1 July 2026, Strengthened Quality Standards in force, designated RN prescribers from mid-2026, Tasmanian pilot 2026-2027. By mid-2027, every RACF in Australia will be operating in a transformed regulatory environment. **A platform that's ready for that environment in mid-2026 has a structural commercial advantage that's hard to replicate later.** The Layer 1 work is on the critical path for that advantage.

**Three:** the three P0 commercial actions (ADG licensing review, AMH licensing negotiation, NCTS account) are blocking work the team has been deferring. **Stop deferring them.** Each is inexpensive in absolute terms; each is genuinely blocking; each compounds in cost the longer it's deferred. The team should treat these as critical-path commercial actions, not nice-to-haves.

What the platform becomes when this layer works: the canonical Australian aged care medication knowledge substrate, with source-attributed governance, audit-ready evidence flows, jurisdiction-aware regulatory rule support, real-time patient state plumbing, and a 7-day SLA from authoritative source publication to deployed rule. Most competitors are doing per-source integration, manual rule authoring, and quarterly reporting. The team is building the substrate beneath all of that.

In follow-up documents we'll do the same exercise for Layer 3 — what changes in the rule encoding methodology now that Layer 1 has expanded categories, Authorisation state machine has been added, and the suppression model has to handle jurisdiction-aware ScopeRules.

— Claude
