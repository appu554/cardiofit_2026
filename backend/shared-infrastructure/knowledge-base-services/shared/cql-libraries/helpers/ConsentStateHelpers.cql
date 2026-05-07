library ConsentStateHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * ConsentStateHelpers
 * ===================
 *
 * Reads over the Consent state machine API (Layer 2 doc §3).
 *
 * Bodies reference Substrate.* — see MedicationHelpers.cql header.
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 3 — ConsentStateHelpers).
 */

// HasActiveConsentForClass — Latency: <50ms p95.
define function "HasActiveConsentForClass"(
  residentRef String, consentClass String
):
  exists(
    Substrate.ListConsents(residentRef, consentClass) C
      where C.status = 'active'
        and (C.expiresAt is null or C.expiresAt > Now())
  )

// ConsentExpiringWithin — Latency: <50ms p95.
define function "ConsentExpiringWithin"(
  residentRef String, consentClass String, days Integer
):
  exists(
    Substrate.ListConsents(residentRef, consentClass) C
      where C.status = 'active'
        and C.expiresAt is not null
        and C.expiresAt <= (Now() + (days * 1 'day'))
  )

// NeedsConsentRefresh — composition of HasActiveConsentForClass and
// ConsentExpiringWithin(14d). Latency: <50ms p95.
define function "NeedsConsentRefresh"(
  residentRef String, consentClass String
):
  ConsentStateHelpers."HasActiveConsentForClass"(residentRef, consentClass)
    and ConsentStateHelpers."ConsentExpiringWithin"(residentRef, consentClass, 14)

// ConsentExpiryDate — null if no active consent of that class.
// Latency: <50ms p95.
define function "ConsentExpiryDate"(
  residentRef String, consentClass String
):
  First(
    Substrate.ListConsents(residentRef, consentClass) C
      where C.status = 'active'
      return C.expiresAt
  )

// SdmReferenceFor — resident → SDM relationship. Latency: <30ms p95.
define function "SdmReferenceFor"(residentRef String):
  Substrate.SdmFor(residentRef)
