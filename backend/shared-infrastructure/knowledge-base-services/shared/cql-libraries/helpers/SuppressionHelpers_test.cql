library SuppressionHelpers_test version '1.0'

using FHIR version '4.0.1'

include SuppressionHelpers version '1.0' called Suppress

// TODO(wave-1-runtime): execute under HAPI Test Engine.

context Patient

define ResidentActioned: 'fixture:resident-rec-actioned-7d'
define ResidentDeferred: 'fixture:resident-rec-deferred-3d'
define ResidentNoHistory: 'fixture:resident-no-rec-history'

// 1. WasActionedRecently positive — rule actioned 7d ago, look back 14d.
define Test_WasActionedRecently_Positive:
  Suppress."WasActionedRecently"(
    ResidentActioned, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION', 14)

// 2. WasActionedRecently negative — different rule, not actioned.
define Test_WasActionedRecently_Negative:
  not Suppress."WasActionedRecently"(
    ResidentNoHistory, 'STOPP_B1_ASPIRIN_PRIMARY_PREVENTION', 14)

// 3. WasDeferredRecently positive — snoozed 3d ago, look back 7d.
define Test_WasDeferredRecently_Positive:
  Suppress."WasDeferredRecently"(
    ResidentDeferred, 'ADG2025_PPI_STEP_DOWN_PROTOCOL', 7)

// 4. RecentSimilarActionedCount returns >0 for matching prefix.
define Test_RecentSimilarActionedCount_Positive:
  Suppress."RecentSimilarActionedCount"(
    ResidentActioned, 'STOPP_B', 30) > 0

// 5. RecentlyOpenedSimilarConcern positive — concern opened recently.
define Test_RecentlyOpenedSimilarConcern_Positive:
  Suppress."RecentlyOpenedSimilarConcern"(
    ResidentActioned, 'GI_BLEED_WATCH', 14)
