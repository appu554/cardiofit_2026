library ClinicalStateHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * ClinicalStateHelpers
 * ====================
 *
 * Substrate-aware reads over baseline_state, observations,
 * active_concerns, care_intensity, capacity_assessment.
 *
 * Bodies reference Substrate.* — see MedicationHelpers.cql header.
 *
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 2 — ClinicalStateHelpers).
 */

// BaselineFor — fetch the rolling baseline value for the kind.
// Latency: <30ms p95.
define function "BaselineFor"(residentRef String, observationKind String):
  Substrate.BaselineFor(residentRef, observationKind)

// DeltaFromBaseline — signed delta over `lookbackDays`.
// Latency: <60ms p95.
define function "DeltaFromBaseline"(
  residentRef String, observationKind String, lookbackDays Integer
):
  Substrate.FlaggedBaselineDelta(residentRef, observationKind, lookbackDays)

// IsTrending — direction in {'up', 'down', 'flat'}.
// Latency: <60ms p95.
define function "IsTrending"(
  residentRef String, observationKind String, direction String
):
  Substrate.TrajectoryDirection(residentRef, observationKind) = direction

// VelocityFlag — true when velocity flag is set on baseline.
// Latency: <60ms p95.
define function "VelocityFlag"(residentRef String, observationKind String):
  Substrate.BaselineVelocityFlag(residentRef, observationKind)

// ActiveConcernCount — open active concerns. Latency: <60ms p95.
define function "ActiveConcernCount"(residentRef String):
  Count(Substrate.ListActiveConcerns(residentRef, true))

// HasActiveConcernType — Latency: <60ms p95.
define function "HasActiveConcernType"(
  residentRef String, concernType String
):
  exists(
    Substrate.ListActiveConcerns(residentRef, true) C
      where C.type = concernType
  )

// CurrentCareIntensity — one of {'active_treatment',
// 'comfort_focused', 'palliative'}. Latency: <30ms p95.
define function "CurrentCareIntensity"(residentRef String):
  Substrate.CareIntensity(residentRef)

// CapacityAssessmentFor — domain ∈ {'medical', 'financial', 'sdm'}.
// Latency: <50ms p95.
define function "CapacityAssessmentFor"(
  residentRef String, domain String
):
  Substrate.CapacityAssessment(residentRef, domain)

// LatestObservationValue — Latency: <100ms p95.
define function "LatestObservationValue"(
  residentRef String, kind String
):
  Substrate.LatestObservationValue(residentRef, kind)

// LatestObservationAt — Latency: <100ms p95.
define function "LatestObservationAt"(
  residentRef String, kind String
):
  Substrate.LatestObservationAt(residentRef, kind)

// IsPalliative sugar. Latency: <30ms p95.
define function "IsPalliative"(residentRef String):
  Substrate.CareIntensity(residentRef) = 'palliative'
