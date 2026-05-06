# MHR gateway error recovery

**Audience:** on-call backend engineer.
**Source contract:** Layer 2 doc §3.x (MHR ingest pathways) +
Wave 1R MHR poller architecture.

## What "MHR gateway" covers

The substrate ingests data from the Australian My Health Record
(MHR) via two pathways:

1. **MHR SOAP poll** (`mhr-poll/`) — periodic SOAP request that pulls
   pathology results, prescriptions, hospital discharge summaries.
   NASH PKI authentication; signed XML envelope.
2. **MHR FHIR push** (V1 deferred) — push subscription via FHIR
   Subscription resources.

Both pathways feed into the kb-20 ingest pipeline and write
substrate rows + EvidenceTrace nodes.

## Triage matrix

| Symptom | Likely cause | First action |
|---------|--------------|--------------|
| All MHR polls failing across all residents | NASH PKI cert expired or DNS issue | Check `mhr_poll_certificates` table; verify DNS via `dig hi.digitalhealth.gov.au` |
| Single-resident polls failing | Per-resident IHI binding stale | Re-bind via the identity_mappings table |
| 401 Unauthorized | Bearer or NASH cert rotated upstream | Rotate kb-20's NASH cert via the certs runbook |
| 503 / 5xx from MHR | MHR-side outage | Check status.digitalhealth.gov.au; circuit-break via `MHR_POLL_DISABLED=true` env var |
| Polls succeed but no data ingested | Document parser failure | Inspect `mhr_poll_audit` table for `parse_error` rows |
| IHI mismatch on inbound payload | Resident moved facility / typo | Route to identity review queue |

## Detailed procedures

### Procedure: NASH PKI certificate rotation

1. Receive new NASH cert + key from the Vaidshala compliance team
   (via the secure channel — never email).
2. Stage to `vault/secrets/nash/` (operator path).
3. Run `kubectl create secret generic nash-cert --from-file=...`.
4. Roll the kb-20 deployment.
5. Verify with a manual poll: `POST /v2/admin/mhr/poll` for one
   test resident.
6. Document the rotation in `docs/security/v2-substrate-security-review.md`'s
   rotation log.

SLA: NASH cert rotation must complete within the 7-day grace period
the AU Digital Health Agency provides before old cert validity ends.

### Procedure: Circuit-breaking MHR polls

When MHR is down or returning consistent 5xx:

1. Set `MHR_POLL_DISABLED=true` in kb-20's environment.
2. Restart the kb-20 deployment.
3. Existing in-flight polls will complete or time out; no new polls
   are scheduled.
4. The MHR poll outbox queue continues to accept retries when the
   flag is later cleared.
5. Communicate to the clinical team that pathology results may be
   delayed.

### Procedure: Recovering from a parse error

1. Inspect the offending payload via `mhr_poll_audit` (the raw XML/JSON
   is captured for audit before parsing).
2. If the payload is malformed, file a ticket with MHR support
   citing the document_id and the parse error.
3. If the payload is well-formed but our parser is stale (new MHR
   field), file an internal ticket against the kb-20 MHR parser
   package.
4. Mark the audit row as `triage_complete=true` so it doesn't
   re-trigger alerts.

### Procedure: Resolving an inbound IHI mismatch

When the MHR poll surfaces an IHI not currently bound to any
resident:

1. The inbound payload is paused at the identity_review_queue.
2. Follow the [identity-match-queue-triage.md](identity-match-queue-triage.md)
   procedure to resolve.
3. Once resolved, the pending payload retries against the new
   binding automatically.

## Monitoring

- **Prometheus metric:** `kb20_mhr_poll_outcome_total{status="..."}`.
- **SLO:** 99% of MHR polls succeed within 30 seconds (excluding
  upstream MHR outages).
- **Alert rule:** trigger on `rate(kb20_mhr_poll_outcome_total{status!="success"}[5m]) > 0.1`.

## Audit trail

Every MHR poll outcome (success or failure) writes one row to
`mhr_poll_audit`. Successful polls additionally write
`evidence_trace_nodes` rows for the resulting substrate writes
(typically Observation insert + ClinicalState transition for any
opened active concern). Failed polls do NOT write EvidenceTrace
nodes — the failure is captured only in the audit table.

## See also

- [identity-match-queue-triage.md](identity-match-queue-triage.md).
- `docs/security/v2-substrate-security-review.md` — NASH PKI rotation
  procedure and security audit log.
- Layer 2 doc §3.x.
