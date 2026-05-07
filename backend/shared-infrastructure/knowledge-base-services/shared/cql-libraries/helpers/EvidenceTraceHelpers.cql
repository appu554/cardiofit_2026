library EvidenceTraceHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * EvidenceTraceHelpers
 * ====================
 *
 * Read-only queries over the EvidenceTrace v0 graph (Layer 2 plan
 * Wave 1R.2). Writes happen as a side-effect of rule fire and are
 * the engine's job, not the rule body's.
 *
 * `ReasoningWindowSummary` is async — it returns a Tuple stream that
 * the engine resolves before downstream defines evaluate. Validator
 * (Wave 1) warns if it appears inside a Tier 1 rule (not in the hot
 * path).
 *
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/specs/2026-05-Layer3-Wave0-cql-helper-surface.md
 *       (File 6 — EvidenceTraceHelpers).
 */

// LineageOf — bidirectional edges. Latency: <100ms p95.
define function "LineageOf"(traceNodeRef String, depth Integer):
  Substrate.EvidenceTraceLineage(traceNodeRef, depth)

// ConsequencesOf — forward traversal. Latency: <100ms p95.
define function "ConsequencesOf"(traceNodeRef String, depth Integer):
  Substrate.EvidenceTraceConsequences(traceNodeRef, depth)

// ReasoningWindowSummary (ASYNC, <200ms p95) — Wave 5.2 query API.
// Returns Tuple{rule_fires, recommendations_drafted, observations_used}.
define function "ReasoningWindowSummary"(
  residentRef String, sinceWindow Interval<DateTime>
):
  Substrate.EvidenceTraceWindowSummary(residentRef, sinceWindow)

// RuleFiredFor — Latency: <100ms p95.
define function "RuleFiredFor"(
  residentRef String, ruleId String, sinceWindow Interval<DateTime>
):
  Substrate.EvidenceTraceRuleFires(residentRef, ruleId, sinceWindow)

// RecommendationOriginRule — Latency: <50ms p95.
define function "RecommendationOriginRule"(recommendationRef String):
  Substrate.EvidenceTraceRecommendationOrigin(recommendationRef)
