/*
 * GovernanceClassifier CQL Library
 * ============================================================
 * Tier 6a Orchestration - Governance Classification
 *
 * PURPOSE: CLASSIFICATION ONLY - NOT ENFORCEMENT
 * This library determines WHAT level of approval an action requires.
 * KB-18 Governance Engine ENFORCES the actual approval workflow.
 *
 * ARCHITECTURE CRITICAL:
 * - CQL OUTPUT: "This action requires Medical Director approval"
 * - CQL DOES NOT: Actually route to Medical Director (that's KB-18)
 *
 * Approval Levels:
 * - LEVEL_1: Auto-approved (no human required)
 * - LEVEL_2: RN/Pharmacist approval
 * - LEVEL_3: Attending Physician approval
 * - LEVEL_4: Department Head approval
 * - LEVEL_5: Medical Director approval (IRB for research)
 * - LEVEL_6: C-Suite/Board approval (institutional risk)
 *
 * Governance: Tier-6a review (Clinical + Technical + Compliance)
 * ============================================================
 */
library GovernanceClassifier version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// APPROVAL LEVEL CODE DEFINITIONS
// ============================================================================

code "Level 1 Auto": 'LEVEL_1' from "http://cardiofit.io/fhir/CodeSystem/approval-level"
code "Level 2 RN Pharmacist": 'LEVEL_2' from "http://cardiofit.io/fhir/CodeSystem/approval-level"
code "Level 3 Attending": 'LEVEL_3' from "http://cardiofit.io/fhir/CodeSystem/approval-level"
code "Level 4 Department": 'LEVEL_4' from "http://cardiofit.io/fhir/CodeSystem/approval-level"
code "Level 5 Medical Director": 'LEVEL_5' from "http://cardiofit.io/fhir/CodeSystem/approval-level"
code "Level 6 Institutional": 'LEVEL_6' from "http://cardiofit.io/fhir/CodeSystem/approval-level"

// ============================================================================
// SECTION 1: MEDICATION ORDER CLASSIFICATION
// ============================================================================

/*
 * Classify medication order approval requirements.
 * OUTPUT: Approval level string for KB-18 consumption.
 *
 * Classification hierarchy (highest wins):
 * 1. Experimental/Compassionate use → LEVEL_6/5
 * 2. Controlled substances with duration concerns → LEVEL_5
 * 3. High cost without authorization → LEVEL_5
 * 4. Off-label without evidence → LEVEL_4
 * 5. Formulary limit exceeded → LEVEL_4
 * 6. High-alert medications → LEVEL_3
 * 7. Drug interactions → LEVEL_3
 * 8. Renal dose adjustment needed → LEVEL_3
 * 9. PRN first administration → LEVEL_2
 * 10. Standard medications → LEVEL_1
 */
define "Medication Approval Level":
  case
    // LEVEL 6: Institutional risk - experimental drugs
    when "Is Experimental Drug" then 'LEVEL_6'

    // LEVEL 5: Medical Director review
    when "Is Compassionate Use" then 'LEVEL_5'
    when "Is Controlled Substance Schedule II" and "Exceeds Standard Opioid Duration" then 'LEVEL_5'
    when "Is High Cost Medication" and "No Prior Authorization" then 'LEVEL_5'
    when "Is Chemotherapy Agent" and "First Cycle" then 'LEVEL_5'

    // LEVEL 4: Department Head
    when "Is Off Label Use" and "No Evidence Support" then 'LEVEL_4'
    when "Exceeds Formulary Limits" then 'LEVEL_4'
    when "Is Restricted Antimicrobial" then 'LEVEL_4'

    // LEVEL 3: Attending Physician
    when Safety."Has Active High Alert Medication" then 'LEVEL_3'
    when "Has Drug Interaction Warning" then 'LEVEL_3'
    when "Requires Renal Dose Adjustment" and not "Adjustment Applied" then 'LEVEL_3'
    when "Requires Hepatic Dose Adjustment" and not "Adjustment Applied" then 'LEVEL_3'
    when Safety."Is Pregnant" and "Pregnancy Category Risk" then 'LEVEL_3'

    // LEVEL 2: RN/Pharmacist
    when "Is PRN Medication" and "First Administration" then 'LEVEL_2'
    when "Requires Double Check" then 'LEVEL_2'
    when "Is High Alert Infusion" then 'LEVEL_2'

    // LEVEL 1: Auto-approved
    else 'LEVEL_1'
  end

// --- Medication Classification Helpers ---

define "Is Experimental Drug":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.category C
            where C.coding.code contains 'experimental'
        )
  )

define "Is Compassionate Use":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.category C
            where C.coding.code contains 'compassionate-use'
              or C.coding.code contains 'expanded-access'
        )
  )

define "Is Controlled Substance Schedule II":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and (
          MR.medication.text ~ 'Schedule II'
          or exists(
            MR.extension E
              where E.url = 'http://cardiofit.io/fhir/StructureDefinition/dea-schedule'
                and E.value = 'II'
          )
        )
  )

define "Exceeds Standard Opioid Duration":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and MR.dispenseRequest.expectedSupplyDuration.value > 7
  )

define "Is High Cost Medication":
  // From KB-6 Formulary - cost exceeds threshold (e.g., >$1000/month)
  // Placeholder - actual value from KB-6 snapshot
  false

define "No Prior Authorization":
  // From KB-6 Formulary - PA required but not obtained
  // Placeholder - actual value from KB-6 snapshot
  false

define "Is Chemotherapy Agent":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.category C
            where C.coding.code contains 'antineoplastic'
              or C.coding.code contains 'chemotherapy'
        )
  )

define "First Cycle":
  // Check medication administration history
  not exists(
    [MedicationAdministration] MA
      where MA.status = 'completed'
        and MA.effective.end > Today() - 90 days
  )

define "Is Off Label Use":
  // Indication not in approved uses - from KB-3
  // Placeholder - requires indication matching
  false

define "No Evidence Support":
  // KB-15 Evidence Engine - no supporting evidence for use
  // Placeholder - actual evidence lookup
  false

define "Exceeds Formulary Limits":
  // KB-6 Formulary quantity/duration limits exceeded
  // Placeholder
  false

define "Is Restricted Antimicrobial":
  // Antimicrobial stewardship restricted list
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/antimicrobial-restriction'
              and E.value = 'restricted'
        )
  )

define "Has Drug Interaction Warning":
  // From KB-5 Drug Interactions snapshot
  Safety."Has Active Drug Interaction"

define "Requires Renal Dose Adjustment":
  Calculators."Has Severe Renal Impairment"
  or Calculators."Has Moderate Renal Impairment"

define "Requires Hepatic Dose Adjustment":
  Calculators."Has Severe Hepatic Impairment"
  or Calculators."Has Moderate Hepatic Impairment"

define "Adjustment Applied":
  // Check if dose was already adjusted per renal/hepatic function
  // Placeholder - requires dose comparison
  false

define "Pregnancy Category Risk":
  // Category D or X medications
  Safety."On Category X Medication"

define "Is PRN Medication":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.dosageInstruction DI
            where DI.asNeeded is not null
              and DI.asNeeded = true
        )
  )

define "First Administration":
  // No prior administration of this medication for this patient
  not exists(
    [MedicationAdministration] MA
      where MA.status = 'completed'
  )

define "Requires Double Check":
  Safety."Has Active High Alert Medication"

define "Is High Alert Infusion":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and MR.dosageInstruction.route.coding.code contains 'intravenous'
        and Safety."Has Active High Alert Medication"
  )

// ============================================================================
// SECTION 2: PROCEDURE ORDER CLASSIFICATION
// ============================================================================

define "Procedure Approval Level":
  case
    // LEVEL 5: High-risk surgical procedures
    when "Is Major Surgical Procedure" and "High Risk Patient" then 'LEVEL_5'

    // LEVEL 4: Invasive procedures in high-risk patients
    when "Is Invasive Procedure" and "High Risk Patient" then 'LEVEL_4'
    when "Is Contrast Required" and Safety."Has Contrast Allergy" then 'LEVEL_4'
    when "Is Radiation Procedure" and "Cumulative Radiation Exceeded" then 'LEVEL_4'

    // LEVEL 3: Standard invasive or sedation procedures
    when "Is Sedation Required" then 'LEVEL_3'
    when "Is Radiation Procedure" then 'LEVEL_3'
    when "Is Invasive Procedure" then 'LEVEL_3'

    // LEVEL 2: Standard non-invasive procedures
    else 'LEVEL_2'
  end

define "Is Major Surgical Procedure":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'major-surgery'
        )
  )

define "Is Invasive Procedure":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'invasive'
        )
  )

define "High Risk Patient":
  Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' }

define "Is Sedation Required":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/sedation-required'
              and E.value = true
        )
  )

define "Is Radiation Procedure":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'ct'
              or C.coding.code contains 'nuclear'
              or C.coding.code contains 'fluoroscopy'
        )
  )

define "Cumulative Radiation Exceeded":
  // Track cumulative radiation dose - exceeds annual limit
  // Placeholder - requires radiation tracking
  false

define "Is Contrast Required":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/contrast-required'
              and E.value = true
        )
  )

// ============================================================================
// SECTION 3: DISCHARGE CLASSIFICATION
// ============================================================================

define "Discharge Approval Level":
  case
    // LEVEL 5: AMA discharge with critical conditions
    when "Against Medical Advice" and "High Risk Patient" then 'LEVEL_5'

    // LEVEL 4: AMA discharge or high-risk discharge
    when "Against Medical Advice" then 'LEVEL_4'
    when "High Risk Discharge" then 'LEVEL_4'

    // LEVEL 3: Incomplete care plan
    when "Incomplete Follow Up Plan" then 'LEVEL_3'
    when "Pending Critical Results" then 'LEVEL_3'
    when "Unstable Vital Signs at Discharge" then 'LEVEL_3'

    // LEVEL 2: Standard discharge
    else 'LEVEL_2'
  end

define "Against Medical Advice":
  exists(
    [Encounter] E
      where E.status = 'in-progress'
        and E.hospitalization.dischargeDisposition.coding.code contains 'ama'
  )

define "High Risk Discharge":
  Safety."Safety Risk Level" = 'CRITICAL'
  or Calculators."SOFA Score" >= 4

define "Incomplete Follow Up Plan":
  // Check for required follow-up orders
  not exists(
    [ServiceRequest] SR
      where SR.status = 'active'
        and SR.intent = 'order'
        and SR.category.coding.code contains 'follow-up'
  )

define "Pending Critical Results":
  // Check for outstanding critical lab results
  exists(
    [DiagnosticReport] DR
      where DR.status in { 'registered', 'partial', 'preliminary' }
        and exists(
          DR.category C
            where C.coding.code contains 'critical'
        )
  )

define "Unstable Vital Signs at Discharge":
  // Recent vital signs indicate instability
  Safety."Has Hemodynamic Instability"
  or Safety."Has Respiratory Compromise"

// ============================================================================
// SECTION 4: OVERRIDE CLASSIFICATION
// ============================================================================

/*
 * When a clinical decision support alert is overridden,
 * classify the approval level needed for the override.
 */
define "Override Approval Level":
  case
    when Safety."Safety Risk Level" = 'CRITICAL' then 'LEVEL_5'
    when Safety."Safety Risk Level" = 'HIGH' then 'LEVEL_4'
    when Safety."Safety Risk Level" = 'MODERATE' then 'LEVEL_3'
    when Safety."Safety Risk Level" = 'LOW' then 'LEVEL_2'
    else 'LEVEL_1'
  end

// ============================================================================
// SECTION 5: TRANSFUSION CLASSIFICATION
// ============================================================================

define "Transfusion Approval Level":
  case
    // Emergency transfusion - immediate, document later
    when "Is Emergency Transfusion" then 'LEVEL_2'

    // Massive transfusion protocol
    when "Is Massive Transfusion" then 'LEVEL_3'

    // Standard transfusion
    when "Is Blood Product Transfusion" then 'LEVEL_2'

    else 'LEVEL_1'
  end

define "Is Emergency Transfusion":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and SR.priority = 'stat'
        and exists(
          SR.category C
            where C.coding.code contains 'blood-product'
        )
  )

define "Is Massive Transfusion":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'massive-transfusion'
        )
  )

define "Is Blood Product Transfusion":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'blood-product'
              or C.coding.code contains 'transfusion'
        )
  )

// ============================================================================
// SECTION 6: RESTRAINT CLASSIFICATION
// ============================================================================

define "Restraint Approval Level":
  case
    // Physical restraints always require physician order
    when "Is Physical Restraint Order" then 'LEVEL_3'

    // Chemical restraint (sedation for agitation)
    when "Is Chemical Restraint Order" then 'LEVEL_3'

    // Seclusion
    when "Is Seclusion Order" then 'LEVEL_3'

    else 'LEVEL_1'
  end

define "Is Physical Restraint Order":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'physical-restraint'
        )
  )

define "Is Chemical Restraint Order":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/restraint-indication'
        )
  )

define "Is Seclusion Order":
  exists(
    [ServiceRequest] SR
      where SR.status in { 'active', 'draft' }
        and exists(
          SR.category C
            where C.coding.code contains 'seclusion'
        )
  )

// ============================================================================
// SECTION 7: AGGREGATE GOVERNANCE OUTPUT
// ============================================================================

/*
 * Maximum approval level across all action types.
 * Used when multiple actions are being classified together.
 */
define "Maximum Approval Level":
  if "Medication Approval Level" = 'LEVEL_6' then 'LEVEL_6'
  else if "Medication Approval Level" = 'LEVEL_5'
       or "Procedure Approval Level" = 'LEVEL_5'
       or "Discharge Approval Level" = 'LEVEL_5'
       or "Override Approval Level" = 'LEVEL_5' then 'LEVEL_5'
  else if "Medication Approval Level" = 'LEVEL_4'
       or "Procedure Approval Level" = 'LEVEL_4'
       or "Discharge Approval Level" = 'LEVEL_4'
       or "Override Approval Level" = 'LEVEL_4' then 'LEVEL_4'
  else if "Medication Approval Level" = 'LEVEL_3'
       or "Procedure Approval Level" = 'LEVEL_3'
       or "Discharge Approval Level" = 'LEVEL_3'
       or "Override Approval Level" = 'LEVEL_3' then 'LEVEL_3'
  else if "Medication Approval Level" = 'LEVEL_2'
       or "Procedure Approval Level" = 'LEVEL_2'
       or "Discharge Approval Level" = 'LEVEL_2'
       or "Override Approval Level" = 'LEVEL_2' then 'LEVEL_2'
  else 'LEVEL_1'

/*
 * Complete governance classification for KB-18 consumption.
 * This tuple provides all information needed for governance workflow routing.
 */
define "Governance Classification Output":
  {
    // Per-action-type levels
    medicationLevel: "Medication Approval Level",
    procedureLevel: "Procedure Approval Level",
    dischargeLevel: "Discharge Approval Level",
    overrideLevel: "Override Approval Level",
    transfusionLevel: "Transfusion Approval Level",
    restraintLevel: "Restraint Approval Level",

    // Aggregate
    maximumLevel: "Maximum Approval Level",

    // Safety context
    safetyRiskLevel: Safety."Safety Risk Level",

    // Audit flags
    requiresAudit: Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' },
    requiresWitness: Safety."On Multiple High Alert Categories",
    requiresDocumentation: "Maximum Approval Level" in { 'LEVEL_4', 'LEVEL_5', 'LEVEL_6' },

    // Escalation indicators
    highRiskPatient: "High Risk Patient",
    hasContraindications: Safety."Has Active Contraindication"
  }
