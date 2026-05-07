library ADG2025 version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include SuppressionHelpers version '1.0' called Suppress

/*
 * ADG2025
 * =======
 * Tier 2 deprescribing rules from the Australian Deprescribing
 * Guideline 2025 (ADG 2025). Wave 3 Task 3 vertical slice ships TWO
 * rules covering the most clinically-important ADG 2025 themes that
 * map to verified Wave 2 substrate primitives:
 *
 *   - ADG2025_PPI_STEP_DOWN_PROTOCOL
 *   - ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW
 *
 * IMPORTANT — Layer 1 binding pending
 * -----------------------------------
 * The ADG 2025 source PDFs are landed in the kb-3 corpus
 * (Layer 1 Wave 5 free-alternative path) but the Pipeline-2
 * recommendation extraction has not yet emitted machine-readable
 * recommendation IDs. Every reference to an ADG 2025 recommendation
 * in this file carries a `TODO(layer1-bind)` marker so the V1 author
 * can grep + replace once Pipeline-2 completes.
 *
 * Remaining ~18 ADG 2025 rules are queued — see
 * claudedocs/clinical/2026-05-Layer3-Wave3-Task3-adg2025-rule-queue.md.
 */

context Patient

// -----------------------------------------------------------------
// ADG2025_PPI_STEP_DOWN_PROTOCOL
// -----------------------------------------------------------------
// Source: ADG 2025 PPI step-down theme — proton pump inhibitor active
// >8 weeks (>56 days) without a documented GI indication concern
// (GORD, peptic ulcer disease, Barrett's, NSAID prophylaxis indication).
// Drafts a step-down recommendation per ADG 2025.
//
// TODO(layer1-bind): replace with real ADG 2025 recommendation ID once
// kb-3 Pipeline-2 extraction completes. Current criterion_id
// 'ADG2025-PPI-STEPDOWN-PLACEHOLDER' is a synthetic identifier.
define "ADG 2025 PPI Step Down Protocol":
  exists(MedicationRequest M
    where Meds."HasActiveAtcPrefix"(Patient.id, 'A02BC')
      and Meds."DurationDaysActive"(M) > 56
  )
  and not State."HasActiveConcernType"(Patient.id, 'GORD')
  and not State."HasActiveConcernType"(Patient.id, 'PEPTIC_ULCER_DISEASE')
  and not State."HasActiveConcernType"(Patient.id, 'BARRETTS_OESOPHAGUS')
  and not State."HasActiveConcernType"(Patient.id, 'NSAID_PROPHYLAXIS_INDICATED')
  and not State."HasActiveConcernType"(Patient.id, 'GI_BLEED_WATCH_14D')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'ADG2025_PPI_STEP_DOWN_PROTOCOL', 90)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ADG2025_PPI_STEP_DOWN_PROTOCOL',
        Interval[Now() - 90 day, Now()])) = 0

// -----------------------------------------------------------------
// ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW
// -----------------------------------------------------------------
// Source: ADG 2025 antipsychotic-BPSD review theme — antipsychotic
// (ATC N05A) active for BPSD AND no documented BPSD review concern
// closed within last 84 days (12 weeks). Drafts a review recommendation
// per ADG 2025 protocol.
//
// Differs from Wang2024 (90-day cycle) by the 12-week (84-day) interval
// and the ADG 2025 review-protocol routing. Both rules can co-fire;
// downstream Recommendation deduplication (Layer 2 Wave 4) is the
// ultimate gate.
//
// TODO(layer1-bind): replace with real ADG 2025 recommendation ID once
// kb-3 Pipeline-2 extraction completes. Current criterion_id
// 'ADG2025-APS-12W-PLACEHOLDER' is a synthetic identifier.
define "ADG 2025 Antipsychotic BPSD 12W Review":
  Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
  and State."HasActiveConcernType"(Patient.id, 'BPSD')
  and not State."HasActiveConcernType"(Patient.id, 'BPSD_REVIEW_DONE_12W')
  and State."CurrentCareIntensity"(Patient.id) = 'active_treatment'
  and not Suppress."WasActionedRecently"(
        Patient.id, 'ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW', 84)
  and Count(Trace."RuleFiredFor"(
        Patient.id, 'ADG2025_ANTIPSYCHOTIC_BPSD_12W_REVIEW',
        Interval[Now() - 84 day, Now()])) = 0
