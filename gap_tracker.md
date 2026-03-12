# Vaidshala HPI Upgrade Gap Tracker

_Start date reference: plan revised March 10, 2026. Week windows below align to that calendar (Week 0 begins March 10, 2026)._ 

| Gap ID | Description | Owner | Closure Artifact / Evidence | Planned Date or Window | Status Notes |
|--------|-------------|-------|-----------------------------|------------------------|--------------|
| A01 | Stratum vs Modifier decision framework | HPI Engine Team | 1-page Canon Framework flowchart answering 4 binary questions | Phase 0 Weeks 1-2 (Mar 17–Mar 30, 2026) | Listed as BLOCKING; must be completed before any new node per plan §3 (Phase 0) |
| A02 | Parameterised strata YAML schema | HPI Engine Team | `parameterised_strata_schema.yaml` spec + validator | Phase 0 Weeks 1-2 (Mar 17–Mar 30, 2026) | BLOCKING for P3/P7; completion paired with A01 |
| A03 | Safety floor specification pattern | HPI Engine Team | Canon spec defining single vs stratum-specific floor tables | Phase 0 Weeks 1-2 (Mar 17–Mar 30, 2026) | BLOCKING for all nodes |
| A04 | Question ordering re-ranking by stratum | HPI Engine Team | Engine spec for `question_priority_override{stratum: [...]}` | Not yet scheduled – required before P3 kickoff (target ≤ Week 5) | Still marked BLOCKING in matrix |
| A05 | Node confidence-threshold standard | HPI Engine Team | Formula doc (base 0.65 − 0.025 per dx beyond 8, floor 0.50) | Not yet scheduled – must be ready before P6 Phase 3 (Week 10) | BLOCKING for P6 |
| A06 | Pertinent negative reliability modifiers | Clinical authoring + HPI | Schema extension `reliability_modifier{condition: adjustment}` | Not yet scheduled – required before P5 start (Week 7) | BLOCKING for P5; beneficial elsewhere |
| B01 | Shared CM registry | KB-19 | `shared_cm_registry.yaml` populated with P0/P1/P2 CMs | Phase 0 Weeks 1-2; verification checkpoint Week 4 Day 1 | BLOCKING (P3 immediately) |
| B02 | Cross-node safety protocol engine | KB-19 | Conflict Arbiter service (BOOST/FLAG/REPORT/RED_FLAG_WINS) + KB-19 routing | Weeks 6-7 (Apr 21–May 4, 2026) | Implemented during Counter-Proposal Sprint 4 |
| B03 | Medication list integration smoke test | KB-20 / KB-22 | Synthetic patient run (metformin+enalapril+empagliflozin+amlodipine+metoprolol) | Week 2-3 Day 4-5 (Mar 24–Mar 28, 2026) | Defined as launch gate for P2 |
| B04 | Stratum activation from KB-20 profile | KB-22 | `stratum_selector` module wiring KB-20 profile → prior table column | Phase 0 Weeks 1-2 with verification Week 4 Day 1 | BLOCKING for stratified nodes |
| B05 | Node transition protocol | KB-22 (HPI Engine Team) | Spec + `NodeTransition` struct covering CONCURRENT/HANDOFF/FLAG modes | Phase 3 kickoff (target May 5, 2026) | Transition evaluator code exists for CONCURRENT; need HANDOFF/FLAG wiring + operational runbook |
| D01 | KB-24 ADR extraction target & SPLGuard integration | Pipeline team | Updated target_kbs + L3 template; pipeline run on KDIGO | Week 3-4 Day 5-6 (Mar 31–Apr 4, 2026) | Highest impact fix; prerequisite for full CM completeness |
| D02 | ADA Standards extraction (guideline profiles) | KB-24 Pipeline (Pipeline team) | Channel A/C/G `guideline_profile` configs for ADA/RSSDI; validation diff report | Pipeline Phase 2 Week 2 (target Apr 28–May 4, 2026) | Blocks P4-P7 automation; owner: Priya (Pipeline PM) |
| D03 | RSSDI 2024 extraction | KB-24 Pipeline (Pipeline team) | RSSDI profile + dual-source reconciliation script | Pipeline Phase 2 Week 2 (target Apr 28–May 4, 2026) | Blocks P4 & P7; same owner/timeline as D02 |
| D04 | Range Integrity Engine cross-system validation | Pipeline team | RIE module covering monotonic thresholds + cross-system slot | Week 4-5 Day 5 (Apr 7–Apr 11, 2026) | Required for P3/P7 |
| D05 | SPLGuard PK-derived onset windows | Pipeline team | `DeriveOnsetWindow()` enrichment + PK data reprocessing | Week 3-4 Day 1-2 (Mar 31–Apr 1, 2026) | Upgrades ADR completeness (Category B) |
| D06 | KB-24 completeness_grade | KB-20 | `adr_profile.go` field + hooks (already implemented) | Complete (pre-plan) | No further action needed |
| E01 | Clinician adjudication interface | Governance + KB-22 UI | Minimal web/mobile reviewer UI logging adjudicated Dx | Tier A Month 0 (Mar 10–Apr 9, 2026) | Required before pilot review panel |
| E02 | Per-question information gain tracker | KB-22 + Data | `hpi.session.events` Kafka topic + KL computation pipeline | Weeks 5-6 (Apr 14–Apr 28, 2026) with Kafka build | Tier B (Month 6+) analytics dependent |
| E03 | Stratum-specific calibration isolation | Governance/Data + Analytics | Partitioned calibration pipeline (≥50 per stratum), ClickHouse materialized views | Tier B kick-off (target Sept 2026) | Requires instrumentation of `hpi.session.events`; owner: Calibration Working Group |
| E04 | Pata-nahi rate tracking | KB-22 | `pata_nahi` counters via G16 + event logging | Week 0-1 Day 2 (Mar 11–Mar 13, 2026) | Immediate need for Hindi voice pilot |
| E05 | Cross-node concordance measurement | Governance/Data | Metrics pipeline aggregating P0/P1/P2 + merged outputs | Tier B (Month 6+) – depends on multi-node sessions post Week 7 | Need Kafka consumer wired to `hpi.session.events` + Conflict Arbiter output; owner: Analytics Ops |
| E06 | Contradiction rate per question pair | KB-22/Data | `contradiction_event` logging from G17 + dashboard | Week 4-5 Day 0-1 (Apr 7–Apr 8, 2026) | Enables Tier A monitoring |
| E07 | Clinical source registry | Data Governance | `clinical_sources`, `element_attributions`, `calibration_events` tables | Week 0-1 Day 4 (Mar 13–Mar 16, 2026) | Must be live before P00 YAML load |
