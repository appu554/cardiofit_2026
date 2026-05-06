# ADR: MHR + Pathology Integration Strategy — SOAP/CDA → FHIR Gateway → HL7 Fallback

**Status:** Proposed (Wave 3)
**Date:** 2026-05-06
**Decision-makers:** TBD (Engineering Lead, Clinical Lead, Platform Lead)
**Consulted:** Layer 2 doc §3.2 author, ADHA conformance pack reviewers (deferred to V1)

## Context

The Vaidshala clinical runtime needs pathology results (potassium, eGFR, sodium,
magnesium, INR, HbA1c, etc.) flowing into the v2 substrate so baselines, active
concerns, and trajectory detection (Wave 3.4) operate on real lab data rather
than CSV-only exports.

Two upstream sources exist in Australia:

1. **My Health Record (MHR)** — national PCEHR operated by ADHA. Currently
   exposes a B2B SOAP gateway returning CDA documents (HL7 v2 / CDA R2) and is
   in the process of standing up a FHIR Gateway aligned with AU Core / AU Base.
   ADHA's published roadmap targets July 2026 for FHIR Gateway general
   availability for B2B clients.
2. **Pathology vendors directly** — for facilities whose pathology providers
   either don't post to MHR or post with substantial latency, an HL7 v2.5
   ORU^R01 feed delivered via per-vendor adapters is the only timely path.

A Wave 3 implementation must cover all three modes without prescribing premature
production wiring against credentials, certificates, and conformance packs we
do not yet hold.

## Options Considered

### Option A — Pure FHIR Gateway only (wait for ADHA July 2026)

**Pros:**
- Single integration surface, modern (FHIR R4)
- Aligned with AU Core profile
- Lower long-term maintenance

**Cons:**
- Blocks Wave 3 entirely until ADHA GA
- No fallback for non-MHR pathology
- ADHA timeline is a target, not a contract

### Option B — Pure SOAP/CDA only (current state)

**Pros:**
- Available today against ADHA NASH PKI auth
- Mature, well-documented conformance pack

**Cons:**
- CDA parsing is heavier than FHIR JSON
- ADHA is actively deprecating SOAP roadmap-wise
- Will require migration work in 6-12 months

### Option C — Multi-mode: SOAP/CDA today, FHIR Gateway as it matures, HL7 ORU as direct-vendor fallback (CHOSEN)

**Pros:**
- Both MHR paths converge on the same internal `CDAPathologyResult` /
  `ParsedObservation` DTO so the substrate write path is unified
- Per-facility config (`mhr_gateway_mode`) selects path without code changes
- HL7 vendor fallback covers facilities outside MHR
- Wave 3 lands the architecture; V1 fills production wiring without redesign

**Cons:**
- Three code paths to maintain
- Per-facility configuration adds operational surface

## Decision

**Adopt Option C.** Wave 3 ships:

1. Interface skeletons + stub clients for `MHRSOAPClient` and `MHRFHIRClient`
   (production wiring deferred to V1 — see "What is deferred" below).
2. A working CDA parser, FHIR DiagnosticReport mapper, and HL7 v2.5 ORU parser,
   each with synthetic fixtures so V1 has a contract to test against.
3. A `pathology_ingest_log` idempotency table so the same CDA document, FHIR
   bundle, or HL7 message arriving twice from different paths produces a single
   substrate write.
4. A per-facility `mhr_gateway_mode` configuration with values:
   - `soap_cda` — use the SOAP/CDA path only
   - `fhir_gateway` — use the FHIR Gateway path only
   - `dual` — query both, dedupe on document ID, prefer FHIR when both return
   - `hl7_only` — for non-MHR facilities, use vendor HL7 adapter only
5. The trajectory detector + velocity flag wiring (Wave 3.4) — a full
   implementation, no stubs, since it is pure Go logic with no external
   dependencies.

## Sequencing

| Phase | Trigger | Action |
|-------|---------|--------|
| Wave 3 (now) | Layer 2 substrate plan | Skeletons + parsers + trajectory full impl |
| V1 Phase 1 | NASH PKI cert + ADHA test endpoint provisioned | Wire `MHRSOAPClient` against ADHA conformance pack |
| V1 Phase 2 | ADHA FHIR Gateway GA (target July 2026) | Wire `MHRFHIRClient`; flip configured facilities to `dual` |
| V1 Phase 3 | First pathology vendor agreement signed | Register vendor adapter via `RegisterVendorAdapter` |
| V2 | ADHA SOAP gateway sunset announcement | Migrate facilities from `soap_cda` to `fhir_gateway` |

## What is deferred to V1

The following are explicitly out of scope for Wave 3 and tracked here so V1
implementers can pick them up without re-discovery:

- **NASH PKI authentication** for ADHA SOAP gateway. Stub returns
  `"mhr_soap_cda: production wiring deferred to V1"` from every method.
- **ADHA conformance pack** XSD validation of CDA documents. The Wave 3
  parser handles a synthetic-but-realistic CDA structure; V1 will need to
  exercise it against the real conformance pack and add per-template-id
  branches as required.
- **FHIR Gateway endpoint configuration** + OAuth2 client credentials flow.
  Stub returns the same deferred error.
- **Per-vendor HL7 adapter quirks**. The Wave 3 registry contains a
  `genericVendorAdapter` that pass-throughs OBR/OBX field positions. V1 will
  add per-vendor adapters as agreements are signed (e.g. handling vendor
  X's non-standard OBX-3 sub-component delimiter).
- **Production rate limiting + backoff** against ADHA endpoints. ADHA
  publishes throttling guidelines that V1 will encode against the real
  client.
- **Subject identity reconciliation** when MHR returns a record under a
  different IHI than the one queried (uncommon but specified in the
  conformance pack). The Wave 3 stub does not handle this case.

## Convergence on internal DTO

Both MHR paths and the HL7 vendor path produce a list of
`ParsedObservation` records (`shared/v2_substrate/ingestion`):

```go
type ParsedObservation struct {
    LOINCCode    string
    SNOMEDCode   string
    DisplayName  string
    Value        *float64
    ValueText    string
    Unit         string
    ObservedAt   time.Time
    AbnormalFlag string // "high" | "low" | ""
}
```

Downstream substrate writes (`V2SubstrateStore.UpsertObservation`) are
unified — one code path regardless of source. Source provenance is
captured on the surrounding EvidenceTrace node, not on the
`ParsedObservation` itself.

## Acceptance criteria

- Wave 3 commits land with the skeleton clients, working parsers, and
  synthetic fixtures.
- `go build ./shared/v2_substrate/... ./kb-20-patient-profile/...` clean.
- All existing Wave 2 tests continue to pass.
- The `pathology_ingest_log` migration applies cleanly against the kb-20
  schema.
- This ADR is reviewed when V1 Phase 1 begins so deferred items are
  re-evaluated against the then-current ADHA roadmap.

## Related

- Layer 2 doc §3.2 (lines 644-687) — MHR + hospital ingestion strategy
- Layer 2 doc §1.4 (lines 277-285) — Observation trajectory + velocity
- ADR `2026-05-06-streaming-pipeline-choice.md` — Wave 2.7 streaming
  precedent for skeleton + deferred-wiring pattern
- Plan task `2026-05-04-layer2-substrate-plan.md` lines 463-528 — Wave 3
  sub-task definitions
