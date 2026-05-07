# Wave 6 — Coronial / ACQSC finding rule-intake template

**Audience:** clinical lead + governance + engineering lead triaging a
coronial inquest finding or an ACQSC (Aged Care Quality and Safety
Commission) complaint finding to determine whether the rule library
needs a new rule, an updated rule, or a suppression tweak.

**Authoritative reference:** Layer 3 v2 plan Wave 6 Task 4.

## Intake sources

| Source | Cadence | Severity bias |
|---|---|---|
| Coronial inquest finding | Per-case (irregular) | High — typically a death or near-miss |
| ACQSC complaint finding | Per-case + monthly digest | Variable |
| Internal incident review | Per-incident | Variable |
| Pilot facility safety report | Weekly | Low → use for trend monitoring |

## Per-finding workflow

### Step 1 — Classify the finding

Classify into one of:

- **A — New rule:** the finding identifies a clinical pattern not
  currently covered by any rule (e.g. opioid + benzodiazepine
  co-prescription on a frail resident with no review). Author a new
  rule.
- **B — Existing rule under-fired:** a relevant rule exists but did
  not fire in the index case (suppression mistuned, content drift,
  data gap). Tune Class 5 / Class 6 suppression per the suppression
  tuning runbook.
- **C — Existing rule over-fired and was overridden:** a relevant rule
  fired but was overridden, contributing to alert fatigue that masked
  the index event. Route to retirement / re-author.
- **D — Out of scope:** the finding is not addressable by the rule
  library (e.g. workforce / training issue). Document and route to
  the appropriate governance forum.

### Step 2 — Capture the index case

Required fields per intake row:

```yaml
intake:
  intake_id: <unique id>
  source: coroner | ACQSC | internal | facility
  source_reference: <coronial reference / ACQSC ref / internal id>
  finding_date: ISO8601
  finding_summary: <one-paragraph summary; avoid PHI>
  classification: A | B | C | D
  resident_archetype: <de-identified description; cohort, comorbidity, meds class>
  proposed_action:
    kind: new_rule | tune_existing | retire | none
    rule_id: <existing rule_id if applicable>
  evidence:
    - <link / citation; coronial PDF, ACQSC notice>
  governance:
    triaged_by: <name + role>
    triaged_at: ISO8601
    clinical_lead_signoff: <name + ISO8601>
    governance_signoff: <name + ISO8601>
```

### Step 3 — If classification = A (new rule)

1. Author the spec + CQL define + 3 fixtures (positive / negative /
   suppression) following the Wave 4A / 4B / 5 patterns in the repo.
2. Run the toolchain locally (Stage 1 + two-gate + CompatibilityChecker
   + CDS Hooks emitter).
3. Promote via GovernancePromoter; obtain dual signature.
4. Backlink the rule's audit block:
   ```yaml
   audit:
     legislative_reference: <existing reference>
     intake_reference: "Coronial inquest 2026/NNN — see claudedocs/governance/2026-XX-coronial-NNN-intake.md"
   ```

### Step 4 — If classification = B (tune existing)

1. Identify the affected rule_id and helper.
2. Tune the Class 5 / Class 6 suppression per the suppression tuning
   runbook decision matrix.
3. Re-promote.

### Step 5 — If classification = C (retire / re-author)

1. Add the rule to the retirement queue via
   `rule_retirement_workflow.py`.
2. Optionally author a replacement rule (typically narrower than the
   retired rule) following Step 3.
3. Cross-reference the retirement decision in the governance signoff.

## SLA on intake

| Severity | Triage SLA | Toolchain SLA | Promotion SLA |
|---|---|---|---|
| Coronial — death or critical near-miss | 24h | 7d | 14d |
| ACQSC — formal complaint with finding | 5d | 7d | 21d |
| Internal incident review | 7d | 7d | 21d |
| Pilot facility safety report | 14d | 7d | 30d |

## Quarterly review

Every 3 months the clinical lead + governance review:

1. Total intake count by source + classification.
2. SLA adherence.
3. Rule-library impact: rules added, tuned, retired.
4. Override-rate trend post-intake (did a new rule cause a spike?).
5. Outstanding intakes still in flight; reasons.

The quarterly review feeds the Wave 6 continuous-tuning rhythm and
provides the regulator-defensible audit trail that the rule library is
updated against real clinical events, not just published guideline
revisions.

## Anti-patterns to avoid

- **Authoring a rule for a single coronial finding without cohort
  evidence** — the rule will fire too narrowly and degrade alert
  quality. Confirm the pattern recurs across at least 2-3 archetypes
  before ramping Tier 1 / Tier 2.
- **Skipping suppressions to ensure the rule "always fires"** — alert
  fatigue is the leading cause of override rate spikes. Use Class 5 /
  Class 6 suppressions to ensure surface relevance.
- **Backdating intake with no governance signoff** — the audit chain
  must show triage → clinical signoff → governance signoff in
  chronological order. No retroactive entries.
