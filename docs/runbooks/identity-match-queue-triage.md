# Identity match queue triage

**Audience:** ACOP pharmacist, on-call ops engineer.
**Source contract:** Layer 2 doc Part 6 Failure 2 — identity-match
errors.

## Why this queue exists

Inbound identity-bearing payloads (eNRMC CSV imports, MHR pathology
pushes, hospital discharge summaries) sometimes arrive with typo'd or
ambiguous identifiers. The substrate's IdentityMatcher classifies each
incoming identifier into one of four confidence tiers
(HIGH / MEDIUM / LOW / NONE). LOW and NONE results
**must not auto-accept** — they queue for human review.

The defence is enforced in
`shared/v2_substrate/identity/matcher.go`: every MatchResult with
Confidence in {LOW, NONE} carries `RequiresReview=true` and surfaces the
candidate set so the reviewer can disambiguate.

## Where the queue lives

- Database table: `identity_review_queue` (kb-20 migration 010 +
  Wave 1R companion migrations).
- API: `GET /v2/identity/review-queue` — paginated list of pending
  reviews. `POST /v2/identity/review-queue/{id}/resolve` — mark a
  queue item as resolved with the reviewer's chosen ResidentRef and
  rationale.

## Triage decision tree

1. **Open the queue item.** Note the source (eNRMC, MHR, discharge),
   the incoming identifier, and the candidate set.
2. **Verify the resident.** Cross-check the candidate residents'
   names + DOBs + facility against the incoming payload's free-text
   fields (typically a name + DOB on every source).
3. **One candidate dominates.** Choose it; resolve with rationale
   `chose_dominant_candidate`. Log the source-system identifier
   variant (typo, formatting drift, etc.) — repeated patterns inform
   upstream cleanup.
4. **Multiple plausible candidates.** Escalate to clinical lead.
   DO NOT guess — incorrect routing is a clinical safety event.
5. **No candidate matches.** This may be a new admission whose
   substrate row hasn't been created yet, or a true cross-facility
   transfer. Resolve with rationale `no_candidate_create_new` and
   trigger the new-admission workflow.

## SLAs

- **Triage queue depth target:** ≤ 20 items at any instant.
- **Triage time per item:** ≤ 5 minutes for clear single-candidate
  cases; ≤ 30 minutes for escalations.
- **Backlog clearance:** ≥ 95% of items resolved within 4 business
  hours of arriving in queue.

## Common patterns

### Pattern: IHI digit transposition

**Symptom:** A single IHI digit is wrong (typically a transposition
in the middle 8 digits). The fuzzy path matches the candidate by
name+DOB at MEDIUM confidence.

**Resolution:** confirm the candidate matches by name+DOB+facility,
resolve, and log a ticket against the source system if the same
typo recurs.

### Pattern: Hyphenated surname

**Symptom:** The source records "Smith-Jones" while the kb-20
canonical record holds "Smith Jones". Fuzzy matcher returns MEDIUM
or LOW confidence.

**Resolution:** confirm DOB + facility match exactly. Resolve;
update the kb-20 canonical row to the source's preferred form if
clinical informatics agrees.

### Pattern: Cross-facility transfer

**Symptom:** Inbound discharge summary references a resident never
seen at this facility. NONE confidence; no candidates.

**Resolution:** verify with the sending facility; create a new
substrate row through the standard new-admission workflow; resolve
the queue item with the new ResidentRef.

## When to escalate

- Two or more high-confidence candidates (true ambiguity).
- The inbound payload references a deceased resident.
- Repeated identical typos from one source within a 24-hour window
  — likely a source-system data-quality bug.
- Anything that smells like data exfiltration (e.g. an MHR push
  for a resident not in our care).

## Audit trail

Every resolution writes an `evidence_trace_nodes` row
(state_machine=`Authorisation`,
state_change_type=`identity_match_resolved`) so the resolution path is
visible in the reasoning-window query. Resolution rationale is captured
on the node's `reasoning_summary`.

## See also

- [evidencetrace-audit-query.md](evidencetrace-audit-query.md)
- [mhr-gateway-error-recovery.md](mhr-gateway-error-recovery.md)
- Layer 2 doc Part 6 Failure 2.
