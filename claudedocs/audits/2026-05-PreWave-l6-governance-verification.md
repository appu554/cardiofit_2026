# Pre-Wave Task 2 — L6 governance audit-trail verifier (runbook)

**Date:** 2026-05-06
**Plan:** [2026-05-04-layer3-rule-encoding-plan.md](../../docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md) — Pre-Wave Task 2
**Status:** Tool delivered. Live verification deferred to the operator
running the verifier against a populated kb-4 PG instance.

---

## What the verifier does

`verify-au-chain` walks the KB-4 Ed25519 signing chain across the 8
criterion sets currently in scope:

```
STOPP_V3, START_V3, BEERS_2023, BEERS_RENAL,
ACB,      PIMS_WANG, AU_APINCHS, AU_TGA_BLACKBOX
```

For every signed rule, it checks:

| Check | Failure mode |
|---|---|
| Ed25519 signature is present and 64 bytes | exit 1 with `[criterion_set/criterion_id]: invalid signature length` |
| `content_sha` (current row) equals `signed_content_sha` (signing-time digest) | exit 1 with `… content_sha drift: stored=… signed=…` |
| `content_sha` is valid hex of length 32 bytes | exit 1 with `content_sha not valid hex` or `decoded length …` |
| Ed25519 signature verifies against the platform pubkey | exit 1 with `Ed25519 signature did not verify` |
| Approval rows include both `CLINICAL_REVIEWER` and `MEDICAL_DIRECTOR` | exit 1 with `missing required approval role …` |
| Each criterion set has at least one signed rule | exit 1 with `[…]: no signed rules found — chain incomplete` |

Exit codes:

| Code | Meaning |
|---|---|
| 0 | every signed rule passes |
| 1 | one or more rules failed; the first failure is reported with `[criterion_set/criterion_id]` prefix |
| 2 | configuration / connectivity error before any rule was checked |

---

## Build

```
cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety
go build -o bin/verify-au-chain ./cmd/verify-au-chain
```

---

## Run

The verifier requires the kb-4 PG DSN and the Ed25519 verification
public key.

```bash
export KB4_DATABASE_URL='postgresql://kb_patient_safety_user:...@localhost:5433/kb_patient_safety'
export KB4_VERIFY_PUBKEY='<64-hex-character Ed25519 public key>'

bin/verify-au-chain
```

Override the env-var sources with explicit flags if needed:

```
bin/verify-au-chain \
  -pubkey-hex=$KB4_VERIFY_PUBKEY \
  -timeout=120
```

A dry-run is available to confirm the public key parses without
touching a database — useful in CI before staging credentials are
attached:

```
bin/verify-au-chain -dry-run -pubkey-hex=...
```

---

## Schema assumptions

The verifier reads from two tables:

* **`kb4_explicit_criteria`** (existing, migration 005). Provides
  `criterion_set`, `criterion_id`, `content_sha`.
* **`kb4_rule_signatures`** (assumed present per the L6 governance
  design). Provides `signature_bytes BYTEA`, `signed_content_sha
  TEXT`, joined on `(criterion_set, criterion_id)`.
* **`kb4_rule_approvals`** (assumed present per the L6 governance
  design). Provides `reviewer_role TEXT`, `reviewer_id TEXT`,
  one row per approver per rule.

If your kb-4 deployment uses different table or column names, edit
`internal/governance/verify_au_chain.go::SQLChainStore.ListSignedRules`
and `ListApprovals` accordingly. The verifier API itself is
schema-agnostic — only the SQL is schema-specific.

---

## Interpreting failures

The first error returned is the failing rule (verifier short-circuits
on first failure to keep the operator focused). To find every failing
rule in one pass, iterate:

1. Run the verifier; note `[criterion_set/criterion_id]` from stderr.
2. Patch (re-sign or repair) the failing rule via the KB-4 governance
   UI or `kb-4-patient-safety/internal/governance` Go API.
3. Re-run; repeat until exit 0.

For a bulk failure inventory before remediation, copy
`internal/governance/verify_au_chain.go::VerifyChain` into a fork
that collects failures rather than returning the first; this is a
small change but kept out of the production verifier so the default
behaviour is the safer fail-fast.

---

## Test coverage

Unit tests at
`internal/governance/verify_au_chain_test.go` cover:

* happy path under all 8 criterion sets (`TestVerifyChain_HappyAllSets`)
* signature drift detection (`TestVerifyRuleSignature_Drift`)
* corrupted signature (`TestVerifyRuleSignature_BadSignature`)
* wrong public key (`TestVerifyRuleSignature_WrongPubKey`)
* missing dual-approval role (`TestVerifyDualApproval_MissingMD`)
* empty criterion set (`TestVerifyChain_EmptyCriterionSetFails`)
* error-prefix correctness (`TestVerifyChain_PrefixesCriterionSetAndID`)
* invalid pub-key length (`TestVerifyChain_RejectsBadPubKeyLength`)

Run with:

```
cd backend/shared-infrastructure/knowledge-base-services/kb-4-patient-safety
go test ./internal/governance/...
```

The CLI itself is a thin wrapper; coverage of `cmd/verify-au-chain`
is left to the integration step where a live kb-4 PG instance is in
play.

---

## Acceptance evidence

* `internal/governance/verify_au_chain.go` compiles cleanly under
  `go build ./internal/governance/...`.
* `internal/governance/verify_au_chain_test.go` passes under
  `go test ./internal/governance/...` (8 tests, all green).
* `cmd/verify-au-chain/main.go` compiles cleanly under
  `go build ./cmd/verify-au-chain/...`.
* This runbook documents invocation, schema assumptions, exit codes,
  and remediation flow.

The actual execution against a live kb-4 DB is out of scope for the
Pre-Wave dispatch (no live kb-4 credentials available in the dev
environment); the operator picks this up before Wave 2 production
deploy per the plan's Pre-Wave exit criterion.
