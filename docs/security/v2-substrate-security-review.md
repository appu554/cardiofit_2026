# Layer 2 substrate — security review

**Status:** V0 security review for production deployment. V1 will
add: HSM-backed NASH key storage, runtime attestation, and a deeper
threat-model review with the Vaidshala compliance team.

This document captures the security posture of the Layer 2 substrate:
PHI access logging, NASH PKI rotation, IHI handling per the Australian
Privacy Act considerations, and the audit posture for ACQSC/OAIC
inquiries.

## Scope

The Layer 2 substrate stores PHI (protected health information):

- Identifying data: full names, DOBs, IHIs, Medicare numbers, DVA
  numbers, facility internal IDs.
- Clinical data: observations, medicine uses, events, baselines,
  active concerns, capacity assessments, CFS scores, EvidenceTrace
  nodes.
- Identity-mapping data: cross-reference between source-system
  identifiers and our canonical Resident UUID.

All PHI is "sensitive personal information" per the Australian Privacy
Act 1988 and the AU Privacy Principles (APPs).

## Authentication

### kb-20 service-to-service

- JWT bearer tokens issued by the platform auth-service, RS256-signed.
- Token validity: 1 hour; refresh against auth-service.
- Service identities are scoped per-KB (kb-22, kb-23, kb-25, etc.);
  each service has a least-privilege role permitting only the
  endpoints it actually needs.
- Operator access via the `kb-20-operator` role through a separate
  short-lived token; operator actions are audited explicitly.

### MHR ingest path

- NASH PKI mutual-TLS with the AU Digital Health Agency.
- NASH cert + private key provisioned via Kubernetes secret;
  rotation procedure documented in
  [mhr-gateway-error-recovery.md](../runbooks/mhr-gateway-error-recovery.md).
- V1: migrate NASH key storage to HSM-backed KMS.

### eNRMC CSV import

- Operator authentication via SSO; CSV upload through the kb-20
  admin API which requires the `kb-20-operator` role.
- Every CSV row landing produces an EvidenceTrace node tagged with
  the operator's role_ref so the audit trail is complete.

## Authorization

### Role-based access

The substrate enforces three primary role tiers:

| Role | Read access | Write access |
|------|-------------|--------------|
| `kb-20-clinical-reader` | All PHI fields | None |
| `kb-20-clinical-writer` | All PHI fields | All clinical writes |
| `kb-20-operator` | All PHI + admin endpoints | Admin endpoints (recompute, queue triage) |

PHI access is logged at the API layer regardless of the operation
outcome (a 404 on a Resident lookup is still an attempted PHI access
and is logged).

### Endpoint scoping

Layer 3 services receive only the scopes they need:

- kb-22 (HPI Engine): read MedicineUse, Observation, Event;
  write EvidenceTrace.
- kb-23 (Decision Cards): read Recommendation lineage and resident
  context; write Recommendation EvidenceTrace transitions.
- kb-25 (Lifestyle Knowledge Graph): read CFS, care intensity;
  no PHI writes.
- kb-26 (Metabolic Digital Twin): read Observation, MedicineUse;
  write derived twin state.

## PHI access logging

Every PHI-touching API call writes one row to `phi_access_log`:

- Caller's role_ref + person_ref (when applicable).
- Endpoint + HTTP method.
- Resource type + ID accessed.
- Timestamp.
- Outcome (200 / 4xx / 5xx).
- Source IP.

The log is append-only; rotation is monthly to cold storage with
indefinite retention (per Australian aged-care record-keeping
requirements).

PHI access logs are reviewable by the Vaidshala compliance lead via
a dedicated read-only endpoint. Bulk export for OAIC submission
follows the standard export procedure.

## IHI handling per Privacy Act

Australian Individual Healthcare Identifiers (IHI) are subject to the
Healthcare Identifiers Act 2010 (Cth) in addition to the general
Privacy Act:

1. **Use limitation.** IHIs are used only for healthcare-record
   linkage, never for marketing or non-healthcare purposes.
2. **Storage.** The IHI is stored in the canonical Resident row and
   in the `identity_mappings` table for cross-source linkage. It is
   NEVER stored in derived analytics tables, logs, or caches.
3. **Logging.** PHI access logs reference the canonical Resident UUID,
   never the raw IHI.
4. **Egress.** The `GET /v2/residents/{id}` endpoint returns the IHI
   only to consumers with the `kb-20-clinical-writer` or
   `kb-20-operator` role. Read-only consumers receive a redacted
   response.
5. **Mismatch handling.** A typo'd or ambiguous IHI lands in the
   identity-review queue; never auto-binds (Failure 2 defence).
6. **Rotation.** IHIs do not rotate; they are lifetime identifiers.
   But the binding from inbound source identifier → canonical
   Resident is mutable and audited via `identity_mappings`.

## NASH PKI rotation

### Rotation cadence

- Standard cadence: every 24 months (matching AU DHA cert validity).
- Emergency rotation: within 24 hours of a key-compromise event.

### Procedure

1. Receive new cert + private key bundle from compliance team via
   the secure channel (Vaidshala vault).
2. Stage in `vault/secrets/nash/{rotation-date}/`.
3. Apply via `kubectl create secret generic nash-cert
   --from-file=...` to the kb-20 namespace.
4. Restart kb-20 pods; verify with one test poll.
5. Update the rotation log below.
6. Securely destroy the previous key material per Vaidshala secure-
   destruction procedure.

### Rotation log

| Date | Operator | Reason | Notes |
|------|----------|--------|-------|
| TBD  | TBD      | initial production cert | First cert provisioned at V0 deploy |

(Append every rotation here. Don't remove rows.)

## EvidenceTrace as audit substrate

The Layer 2 EvidenceTrace IS the substrate-side audit trail for
clinical reasoning. Properties:

- **Append-only:** no row is ever deleted (Wave 5.4 plan: deletion is
  a correctness bug).
- **Bidirectional:** forward + backward traversal supported from
  day 1 (Recommendation 3 of Layer 2 doc Part 7).
- **FHIR-aligned:** each node maps to exactly one FHIR Provenance OR
  AuditEvent (Wave 5.3 dispatcher; Layer 2 doc §1.6).
- **Regulator-queryable:** the reasoning-window query
  (`GET /v2/residents/{id}/reasoning-window`) returns an
  ACQSC-submission-ready JSON envelope.

The phi_access_log captures access to PHI; EvidenceTrace captures
the reasoning that PHI fed into. Together they form the complete
audit record.

## OAIC notifiable-data-breach posture

The substrate is designed so that a single-resident PHI exposure
event has a bounded blast radius:

- Per-resident row-level encryption at rest (V1 — currently DB-level
  encryption only).
- Audit access logs make any unusual access pattern detectable
  within 24 hours.
- Outbox events do NOT carry full PHI payloads; consumers fetch the
  full row via the authenticated read API.

In the event of a confirmed PHI breach, the OAIC notifiable-data-
breach playbook (separate document, Vaidshala compliance) governs
the response. The substrate's role is to provide the access log
evidence within the OAIC-required 30-day window.

## Open security gaps for V1

1. HSM-backed NASH key storage (currently Kubernetes secret).
2. Per-row encryption at rest (currently DB-level encryption only).
3. Runtime attestation for kb-20 pods.
4. Penetration testing pass with an AU-accredited auditor.
5. Threat model walkthrough with the Vaidshala compliance team
   (Christensen-style "what job is this attacker hired to do" — not
   yet completed for the substrate).

## See also

- Layer 2 doc §1.6 + Part 6 (failure modes).
- `docs/runbooks/identity-match-queue-triage.md` — Failure 2 defence.
- `docs/runbooks/mhr-gateway-error-recovery.md` — NASH PKI rotation.
- `docs/runbooks/evidencetrace-audit-query.md` — audit query patterns.
- Privacy Act 1988 (Cth); Healthcare Identifiers Act 2010 (Cth);
  Australian Privacy Principles 1-13.
