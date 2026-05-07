library MonitoringHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * MonitoringHelpers
 * =================
 *
 * Reads over the MonitoringPlan API (Layer 2 doc §2.4).
 *
 * Bodies reference Substrate.* — see MedicationHelpers.cql header.
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 5 — MonitoringHelpers).
 */

// ExpectedObservationCount — Latency: <50ms p95.
define function "ExpectedObservationCount"(planRef String, asOf DateTime):
  Substrate.MonitoringPlanExpectedCount(planRef, asOf)

// ObservationOverdueBy — hours overdue. Latency: <50ms p95.
define function "ObservationOverdueBy"(
  planRef String, kind String, asOf DateTime
):
  Substrate.MonitoringObservationOverdueHours(planRef, kind, asOf)

// MonitoringPlanIsActive — Latency: <30ms p95.
define function "MonitoringPlanIsActive"(planRef String, asOf DateTime):
  Substrate.MonitoringPlanActiveAt(planRef, asOf)

// ActivePlansFor — Latency: <50ms p95.
define function "ActivePlansFor"(residentRef String):
  Substrate.ListMonitoringPlans(residentRef, true)

// HasOpenPlanOfType — Latency: <50ms p95.
define function "HasOpenPlanOfType"(
  residentRef String, planType String
):
  exists(
    Substrate.ListMonitoringPlans(residentRef, true) P
      where P.type = planType
  )

// LastThresholdCrossingAt — Latency: <50ms p95.
define function "LastThresholdCrossingAt"(planRef String):
  Substrate.MonitoringLastThresholdCrossing(planRef)
