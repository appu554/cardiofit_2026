library SuppressionHelpers version '1.0'

using FHIR version '4.0.1'

include Vaidshala.Substrate version '1.0' called Substrate

/*
 * SuppressionHelpers
 * ==================
 *
 * Class 2 (recently-actioned) suppression primitives — Wave 3 Task 4.
 *
 * Each function backs a `recommendation_state.*` fact_type that the
 * Wave 1 rule_specification_validator recognises. Bodies fan out to
 * the recommendation-state substrate API (read-only queries over the
 * Recommendation state machine + EvidenceTrace v0 graph).
 *
 *   WasActionedRecently            → recommendation_state.recently_actioned
 *   WasDeferredRecently            → recommendation_state.recently_deferred
 *   RecentSimilarActionedCount     → recommendation_state.recent_similar_actioned_count
 *   RecentlyOpenedSimilarConcern   → recommendation_state.recently_opened_similar_concern
 *
 * Latency budget (per helper, p95): <80ms — backed by the same
 * EvidenceTrace v0 indices used by RuleFiredFor (<100ms).
 *
 * "Actioned" semantics (substrate convention):
 *   - decided      — user accepted/dismissed the recommendation
 *   - refused      — SDM/clinician explicitly declined
 *   - superseded   — replaced by a newer recommendation in same class
 * "Deferred" semantics:
 *   - snooze, requeue_later — non-terminal closures that suppress
 *     re-fire for the chosen window without resolving the underlying
 *     concern.
 *
 * TODO(wave-1-runtime): execute under HAPI Test Engine.
 *
 * Spec: docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md
 *       (Wave 3 Task 4 — Suppression-by-recent-action library).
 */

// WasActionedRecently — true if a recommendation originating from
// `ruleId` has been closed in {decided, refused, superseded} within
// `withinDays` days for `residentRef`.
define function "WasActionedRecently"(
  residentRef String, ruleId String, withinDays Integer
):
  Substrate.RecommendationActionedWithin(residentRef, ruleId, withinDays)

// WasDeferredRecently — true if a recommendation originating from
// `ruleId` has been closed in {snooze, requeue_later} within
// `withinDays` days. Distinct from WasActionedRecently because the
// underlying clinical concern is unresolved; callers may choose to
// suppress for shorter windows.
define function "WasDeferredRecently"(
  residentRef String, ruleId String, withinDays Integer
):
  Substrate.RecommendationDeferredWithin(residentRef, ruleId, withinDays)

// RecentSimilarActionedCount — count of distinct rules sharing the
// `ruleClassPrefix` (e.g. 'STOPP_B' for all primary-prevention
// antiplatelet rules) that have been actioned for `residentRef`
// within `withinDays`. Lets a rule suppress when the resident has
// already received 2+ deprescribing actions in the same therapeutic
// class — avoids alert-cascade.
define function "RecentSimilarActionedCount"(
  residentRef String, ruleClassPrefix String, withinDays Integer
):
  Count(
    Substrate.RecommendationsActionedByPrefix(
      residentRef, ruleClassPrefix, withinDays))

// RecentlyOpenedSimilarConcern — true if an active_concern of type
// `concernType` was opened (transitioned to active) within
// `withinDays`. Distinct from `HasActiveConcernType` (which is a
// snapshot read) — this is a transition-window read used to
// suppress rules whose intervention overlaps with a fresh concern
// (e.g. don't fire a deprescribing rule when a "GI_BLEED_WATCH"
// was just opened).
define function "RecentlyOpenedSimilarConcern"(
  residentRef String, concernType String, withinDays Integer
):
  Substrate.ActiveConcernOpenedWithin(residentRef, concernType, withinDays)
