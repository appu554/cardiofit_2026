library PharmaCareIndicators version '1.0'

using FHIR version '4.0.1'

include MedicationHelpers version '1.0' called Meds
include ClinicalStateHelpers version '1.0' called State
include EvidenceTraceHelpers version '1.0' called Trace
include QualityGapHelpers version '1.0' called QG

/*
 * PharmaCareIndicators
 * ====================
 * PHARMA-Care 5-domain indicator computation scaffold (Wave 4A Task 1
 * acceptance criterion). The PHARMA-Care framework is the UniSA
 * Sluggett-led published 5-domain quality-indicator framework for
 * residential aged-care medicines optimisation.
 *
 * This file defines the computation surface for the 5 published
 * domains. Each define returns a structured indicator value that the
 * Layer 2 indicator aggregator consumes as workflow exhaust. The
 * domain headers below reference the published PHARMA-Care v1
 * framework; exact published label confirmation is tagged
 * TODO(clinical-author) until the framework PDF is checked in.
 *
 * Output schema convention:
 *   - per-resident point-in-time indicator value (numerator/denominator
 *     or count/percentage/score)
 *   - the Layer 2 aggregator rolls up to facility-level percentages
 *
 * Domains (PHARMA-Care v1, published):
 *   D1 — Medicines optimisation / polypharmacy
 *   D2 — Restraint / antipsychotic prescribing
 *   D3 — Adverse-drug-event risk / anticholinergic burden
 *   D4 — Behavioural-care first-line non-pharm
 *   D5 — Medication review cycle / RMMR + reconciliation gaps
 *
 * TODO(clinical-author): confirm exact published domain identifiers
 * and labels against the PHARMA-Care v1 framework once it lands in
 * docs/clinical/.
 */

context Patient

// -----------------------------------------------------------------
// PharmaCare D1 — Medicines optimisation / polypharmacy
// -----------------------------------------------------------------
// Returns the resident's active MedicineUse count. The Layer 2
// aggregator computes:
//   * facility-level % residents with >=10 active meds (polypharmacy)
//   * facility-level % residents with >=15 active meds (severe)
// TODO(clinical-author): confirm published D1 thresholds.
define "PharmaCare D1 ActiveMedicineCount":
  QG."ActiveMedicineUseCount"(Patient.id)

// -----------------------------------------------------------------
// PharmaCare D2 — Restraint / antipsychotic prescribing
// -----------------------------------------------------------------
// Returns 1 when an antipsychotic (ATC N05A) is active without a
// documented BPSD diagnosis or 12-week BPSD review concern; 0 otherwise.
// The Layer 2 aggregator rolls this to the facility-level antipsychotic
// prevalence-without-indication percentage.
// TODO(clinical-author): confirm published D2 numerator definition.
define "PharmaCare D2 AntipsychoticWithoutIndicationFlag":
  if Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
    and not State."HasActiveConcernType"(Patient.id, 'BPSD_DIAGNOSIS')
    and not State."HasActiveConcernType"(Patient.id, 'BPSD_REVIEW_DONE_12W')
  then 1
  else 0

// -----------------------------------------------------------------
// PharmaCare D3 — Adverse-drug-event risk / anticholinergic burden
// -----------------------------------------------------------------
// Returns the resident's computed ACB score. The Layer 2 aggregator
// computes facility-level percentage with ACB >3.
// TODO(clinical-author): confirm published D3 ACB threshold.
define "PharmaCare D3 AcbScore":
  QG."AcbScoreFor"(Patient.id)

// -----------------------------------------------------------------
// PharmaCare D4 — Behavioural-care first-line non-pharm
// -----------------------------------------------------------------
// Returns 1 when the resident has BPSD diagnosis + psychotropic active
// + no documented non-pharm intervention trial concern; 0 otherwise.
// The Layer 2 aggregator rolls this to the facility-level
// non-pharm-first-line gap percentage.
// TODO(clinical-author): confirm published D4 numerator definition.
define "PharmaCare D4 NonPharmFirstLineGapFlag":
  if State."HasActiveConcernType"(Patient.id, 'BPSD_DIAGNOSIS')
    and (
      Meds."HasActiveAtcPrefix"(Patient.id, 'N05A')
      or Meds."HasActiveAtcPrefix"(Patient.id, 'N05B')
    )
    and not State."HasActiveConcernType"(Patient.id, 'NONPHARM_INTERVENTION_TRIAL')
  then 1
  else 0

// -----------------------------------------------------------------
// PharmaCare D5 — Medication review cycle / RMMR + reconciliation gaps
// -----------------------------------------------------------------
// Returns 1 when the resident has any active care-transition gap:
//   * latest hospital_discharge >72h ago without completed reconciliation
//   * RMMR overdue (no rmmr_followup Event in last 180 days)
// 0 otherwise. The Layer 2 aggregator rolls this to the facility-level
// review-cycle compliance percentage.
// TODO(clinical-author): confirm published D5 lookback windows.
define "PharmaCare D5 ReviewCycleGapFlag":
  if (
    QG."LatestHospitalDischargeOlderThan"(Patient.id, 72)
    and not QG."HasCompletedReconciliationFor"(
      Patient.id, QG."LatestHospitalDischargeId"(Patient.id))
  ) or (
    not QG."HasRecentEventOfType"(Patient.id, 'rmmr_followup', 180)
  )
  then 1
  else 0

// -----------------------------------------------------------------
// PharmaCare ResidentIndicatorBundle
// -----------------------------------------------------------------
// Returns a tuple bundling all 5 domain indicator values for the
// resident. Layer 2 aggregator joins this output across the facility
// and emits the PHARMA-Care v1 5-domain percentage report.
define "PharmaCare ResidentIndicatorBundle":
  Tuple {
    resident: Patient.id,
    d1_active_medicine_count: "PharmaCare D1 ActiveMedicineCount",
    d2_antipsychotic_without_indication: "PharmaCare D2 AntipsychoticWithoutIndicationFlag",
    d3_acb_score: "PharmaCare D3 AcbScore",
    d4_nonpharm_first_line_gap: "PharmaCare D4 NonPharmFirstLineGapFlag",
    d5_review_cycle_gap: "PharmaCare D5 ReviewCycleGapFlag"
  }
