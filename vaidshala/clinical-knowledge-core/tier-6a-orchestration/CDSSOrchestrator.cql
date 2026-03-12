/*
 * CDSSOrchestrator CQL Library
 * ============================================================
 * Tier 6a Orchestration - Protocol Selection & CDSS Orchestration
 *
 * PURPOSE: PROTOCOL SELECTION AND ORCHESTRATION FOR KB-19
 * This library identifies applicable protocols, ranks them by
 * priority, detects conflicts, and recommends optimal protocols.
 *
 * ARCHITECTURE CRITICAL (CTO/CMO DIRECTIVE):
 * - CQL OUTPUT: "Sepsis Protocol is recommended, evidence grade A"
 * - KB-19 CONSUMES: Protocol recommendation for execution
 * - KB-19 CONSTRAINT: "Guideline reasoning + explanation" ONLY
 * - ICU DOMINANCE: ICU Intelligence can VETO any KB-19 recommendation
 *
 * Protocol Selection Hierarchy:
 * 1. Safety protocols (always applicable if criteria met)
 * 2. Condition-specific protocols (evidence-ranked)
 * 3. Preventive protocols (population health)
 *
 * Governance: Tier-6a review (Clinical + Informatics + Quality)
 * ============================================================
 */
library CDSSOrchestrator version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// PROTOCOL CODE DEFINITIONS
// ============================================================================

codesystem "ProtocolCategory": 'http://cardiofit.io/fhir/CodeSystem/protocol-category'
codesystem "EvidenceGrade": 'http://cardiofit.io/fhir/CodeSystem/evidence-grade'

code "Category Safety": 'SAFETY' from "ProtocolCategory"
code "Category Sepsis": 'SEPSIS' from "ProtocolCategory"
code "Category Cardiac": 'CARDIAC' from "ProtocolCategory"
code "Category Respiratory": 'RESPIRATORY' from "ProtocolCategory"
code "Category Metabolic": 'METABOLIC' from "ProtocolCategory"
code "Category Renal": 'RENAL' from "ProtocolCategory"
code "Category VTE": 'VTE' from "ProtocolCategory"
code "Category Preventive": 'PREVENTIVE' from "ProtocolCategory"

code "Grade A": 'A' from "EvidenceGrade"
code "Grade B": 'B' from "EvidenceGrade"
code "Grade C": 'C' from "EvidenceGrade"
code "Grade D": 'D' from "EvidenceGrade"

// ============================================================================
// SECTION 1: PROTOCOL APPLICABILITY DEFINITIONS
// ============================================================================

/*
 * Sepsis Protocol Applicable
 * Based on Surviving Sepsis Campaign 2021 criteria.
 */
define "Sepsis Protocol Applicable":
  "Has Suspected Infection" and "Meets Sepsis Screening Criteria"

define "Has Suspected Infection":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (
          C.code.coding.code contains 'R65'  // SIRS
          or C.code.coding.code contains 'A41'  // Sepsis
          or C.code.coding.code contains 'R78.81'  // Bacteremia
        )
  )
  or "Has Active Antimicrobial Order"

define "Has Active Antimicrobial Order":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.category C
            where C.coding.code contains 'antimicrobial'
        )
  )

define "Meets Sepsis Screening Criteria":
  // qSOFA >= 2
  "qSOFA Score" >= 2

define "qSOFA Score":
  Count({
    if Calculators."Systolic BP Value" < 100 then 1 else null,
    if Calculators."Respiratory Rate Value" >= 22 then 1 else null,
    if "Has Altered Mental Status" then 1 else null
  })

define "Has Altered Mental Status":
  exists(
    [Observation] O
      where O.status = 'final'
        and O.code.coding.code contains '72172-0'  // GCS
        and O.value.value < 15
  )

define "Sepsis Protocol Priority":
  // Sepsis is high priority, especially with shock
  if "Meets Septic Shock Criteria" then 10
  else if "Sepsis Protocol Applicable" then 9
  else 0

define "Meets Septic Shock Criteria":
  "Sepsis Protocol Applicable"
    and "Requires Vasopressors"
    and Calculators."Lactate Value" > 2.0

define "Requires Vasopressors":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and (
          MR.medication.coding.code contains 'norepinephrine'
          or MR.medication.coding.code contains 'vasopressin'
          or MR.medication.coding.code contains 'epinephrine'
        )
  )

/*
 * VTE Prophylaxis Protocol Applicable
 * Based on CHEST guidelines for hospitalized patients.
 */
define "VTE Prophylaxis Protocol Applicable":
  "Is Hospitalized" and "VTE Risk Score" >= 4

define "Is Hospitalized":
  exists(
    [Encounter] E
      where E.status = 'in-progress'
        and E.class.code = 'IMP'
  )

define "VTE Risk Score":
  // Padua Prediction Score
  Count({
    if "Has Active Cancer" then 3 else null,
    if "Has Prior VTE" then 3 else null,
    if "Has Reduced Mobility" then 3 else null,
    if "Has Known Thrombophilia" then 3 else null,
    if "Recent Trauma or Surgery" then 2 else null,
    if AgeInYears() >= 70 then 1 else null,
    if "Has Heart Failure or Respiratory Failure" then 1 else null,
    if "Has AMI or Stroke" then 1 else null,
    if "Has Acute Infection or Rheumatologic Disorder" then 1 else null,
    if Calculators."BMI Value" >= 30 then 1 else null,
    if "On Hormonal Therapy" then 1 else null
  })

define "Has Active Cancer":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'C'  // ICD-10 malignancy
  )

define "Has Prior VTE":
  exists(
    [Condition] C
      where C.code.coding.code contains 'I26'  // PE
        or C.code.coding.code contains 'I82'  // DVT
  )

define "Has Reduced Mobility":
  exists(
    [Observation] O
      where O.code.coding.code contains 'mobility-status'
        and O.value.coding.code contains 'bedbound'
  )

define "Has Known Thrombophilia":
  exists(
    [Condition] C
      where C.code.coding.code contains 'D68'  // Coagulation defects
  )

define "Recent Trauma or Surgery":
  exists(
    [Procedure] P
      where P.status = 'completed'
        and P.performed.end >= Today() - 30 days
  )

define "Has Heart Failure or Respiratory Failure":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (C.code.coding.code contains 'I50' or C.code.coding.code contains 'J96')
  )

define "Has AMI or Stroke":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (C.code.coding.code contains 'I21' or C.code.coding.code contains 'I63')
  )

define "Has Acute Infection or Rheumatologic Disorder":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (C.code.coding.code contains 'A' or C.code.coding.code contains 'M')  // Infection or Rheumatic
  )

define "On Hormonal Therapy":
  exists(
    [MedicationRequest] MR
      where MR.status = 'active'
        and MR.medication.coding.code contains 'hormone'
  )

define "VTE Prophylaxis Protocol Priority":
  if "VTE Prophylaxis Protocol Applicable" and not "Has Bleeding Contraindication" then 7
  else if "VTE Prophylaxis Protocol Applicable" then 5  // Mechanical prophylaxis only
  else 0

define "Has Bleeding Contraindication":
  Safety."Has Active Bleeding"
    or Safety."Has Coagulopathy"
    or Calculators."Platelet Count Value" < 50

/*
 * Heart Failure GDMT Protocol Applicable
 * Based on ACC/AHA 2022 guidelines.
 */
define "HF GDMT Protocol Applicable":
  "Has Heart Failure" and "LVEF Category Known"

define "Has Heart Failure":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'I50'
  )

define "LVEF Category Known":
  exists(
    [Observation] O
      where O.status = 'final'
        and O.code.coding.code contains '10230-1'  // LVEF
  )

define "LVEF Value":
  Last(
    [Observation] O
      where O.status = 'final'
        and O.code.coding.code contains '10230-1'
      sort by effective.value
  ).value.value

define "HF GDMT Protocol Priority":
  case
    when "HF GDMT Protocol Applicable" and "LVEF Value" < 40 then 8  // HFrEF - most guideline-directed
    when "HF GDMT Protocol Applicable" and "LVEF Value" <= 50 then 6  // HFmrEF
    when "HF GDMT Protocol Applicable" then 5  // HFpEF
    else 0
  end

/*
 * Diabetes Management Protocol Applicable
 * Based on ADA Standards 2024.
 */
define "Diabetes Protocol Applicable":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (C.code.coding.code contains 'E11' or C.code.coding.code contains 'E10')
  )

define "Diabetes Protocol Priority":
  case
    when "Diabetes Protocol Applicable" and Calculators."HbA1c Value" > 10.0 then 8
    when "Diabetes Protocol Applicable" and Calculators."HbA1c Value" > 9.0 then 7
    when "Diabetes Protocol Applicable" and Calculators."HbA1c Value" > 8.0 then 6
    when "Diabetes Protocol Applicable" then 5
    else 0
  end

/*
 * AKI Protocol Applicable
 * Based on KDIGO 2012 guidelines.
 */
define "AKI Protocol Applicable":
  "Meets AKI Criteria"

define "Meets AKI Criteria":
  // KDIGO AKI staging
  "Has Creatinine Rise" or "Has Urine Output Decrease"

define "Has Creatinine Rise":
  exists(
    [Observation: "Creatinine Code"] Current
      let Baseline: First(
        [Observation: "Creatinine Code"] B
          where B.effective.value < Current.effective.value - 48 hours
          sort by effective.value descending
      )
      where Current.status = 'final'
        and Baseline is not null
        and (
          Current.value.value >= Baseline.value.value * 1.5  // Stage 1+
          or Current.value.value >= Baseline.value.value + 0.3  // Stage 1
        )
  )

define "Creatinine Code":
  Code '2160-0' from "LOINC"

codesystem "LOINC": 'http://loinc.org'

define "Has Urine Output Decrease":
  exists(
    [Observation] O
      where O.code.coding.code contains '9192-6'  // Urine output
        and O.value.value < 0.5  // mL/kg/hr
        and O.effective.value >= Today() - 6 hours
  )

define "AKI Protocol Priority":
  case
    when "AKI Protocol Applicable" and Calculators."eGFR Value" < 15 then 9  // AKI Stage 3
    when "AKI Protocol Applicable" and Calculators."eGFR Value" < 30 then 8  // AKI Stage 2
    when "AKI Protocol Applicable" then 7  // AKI Stage 1
    else 0
  end

// ============================================================================
// SECTION 2: APPLICABLE PROTOCOLS LIST
// ============================================================================

/*
 * Applicable Protocols
 * List of all protocols that apply to current patient context.
 */
define "Applicable Protocols":
  {
    sepsis: if "Sepsis Protocol Applicable" then {
      protocolId: 'SEPSIS_SSC_2021',
      name: 'Surviving Sepsis Campaign 2021',
      category: 'SEPSIS',
      priority: "Sepsis Protocol Priority",
      evidenceGrade: 'A',
      isCritical: "Meets Septic Shock Criteria"
    } else null,

    vte: if "VTE Prophylaxis Protocol Applicable" then {
      protocolId: 'VTE_CHEST_2021',
      name: 'VTE Prophylaxis (CHEST Guidelines)',
      category: 'VTE',
      priority: "VTE Prophylaxis Protocol Priority",
      evidenceGrade: 'A',
      hasBleedingRisk: "Has Bleeding Contraindication"
    } else null,

    hf: if "HF GDMT Protocol Applicable" then {
      protocolId: 'HF_GDMT_2022',
      name: 'Heart Failure GDMT (ACC/AHA 2022)',
      category: 'CARDIAC',
      priority: "HF GDMT Protocol Priority",
      evidenceGrade: 'A',
      lvefCategory: case
        when "LVEF Value" < 40 then 'HFrEF'
        when "LVEF Value" <= 50 then 'HFmrEF'
        else 'HFpEF'
      end
    } else null,

    diabetes: if "Diabetes Protocol Applicable" then {
      protocolId: 'T2DM_ADA_2024',
      name: 'Type 2 Diabetes (ADA Standards 2024)',
      category: 'METABOLIC',
      priority: "Diabetes Protocol Priority",
      evidenceGrade: 'A',
      glycemicUrgency: Calculators."HbA1c Value" > 9.0
    } else null,

    aki: if "AKI Protocol Applicable" then {
      protocolId: 'AKI_KDIGO_2012',
      name: 'AKI Management (KDIGO 2012)',
      category: 'RENAL',
      priority: "AKI Protocol Priority",
      evidenceGrade: 'B',
      akiStage: case
        when Calculators."eGFR Value" < 15 then 3
        when Calculators."eGFR Value" < 30 then 2
        else 1
      end
    } else null
  }

// ============================================================================
// SECTION 3: PROTOCOL PRIORITY RANKING
// ============================================================================

/*
 * Protocol Priority Ranking
 * Orders applicable protocols by priority score (highest first).
 */
define "Protocol Priority Ranking":
  {
    if "Sepsis Protocol Priority" > 0 then { protocol: 'SEPSIS', priority: "Sepsis Protocol Priority" } else null,
    if "AKI Protocol Priority" > 0 then { protocol: 'AKI', priority: "AKI Protocol Priority" } else null,
    if "HF GDMT Protocol Priority" > 0 then { protocol: 'HF_GDMT', priority: "HF GDMT Protocol Priority" } else null,
    if "VTE Prophylaxis Protocol Priority" > 0 then { protocol: 'VTE', priority: "VTE Prophylaxis Protocol Priority" } else null,
    if "Diabetes Protocol Priority" > 0 then { protocol: 'DIABETES', priority: "Diabetes Protocol Priority" } else null
  }

/*
 * Recommended Protocol
 * Top-ranked applicable protocol for current context.
 * NOTE: ICU Intelligence can override this recommendation.
 */
define "Recommended Protocol":
  case
    when "Sepsis Protocol Priority" >= "Maximum Protocol Priority" and "Sepsis Protocol Priority" > 0
      then 'SEPSIS_SSC_2021'
    when "AKI Protocol Priority" >= "Maximum Protocol Priority" and "AKI Protocol Priority" > 0
      then 'AKI_KDIGO_2012'
    when "HF GDMT Protocol Priority" >= "Maximum Protocol Priority" and "HF GDMT Protocol Priority" > 0
      then 'HF_GDMT_2022'
    when "VTE Prophylaxis Protocol Priority" >= "Maximum Protocol Priority" and "VTE Prophylaxis Protocol Priority" > 0
      then 'VTE_CHEST_2021'
    when "Diabetes Protocol Priority" >= "Maximum Protocol Priority" and "Diabetes Protocol Priority" > 0
      then 'T2DM_ADA_2024'
    else null
  end

define "Maximum Protocol Priority":
  Max({
    "Sepsis Protocol Priority",
    "AKI Protocol Priority",
    "HF GDMT Protocol Priority",
    "VTE Prophylaxis Protocol Priority",
    "Diabetes Protocol Priority"
  })

// ============================================================================
// SECTION 4: PROTOCOL CONFLICTS
// ============================================================================

/*
 * Protocol Conflicts
 * Detects conflicts between applicable protocols.
 */
define "Protocol Conflicts":
  {
    // VTE vs AKI: Anticoagulation may worsen renal function in some cases
    vteAkiConflict: if "VTE Prophylaxis Protocol Applicable" and "AKI Protocol Applicable"
      then {
        conflictType: 'ANTICOAGULATION_RENAL',
        protocols: { 'VTE_CHEST_2021', 'AKI_KDIGO_2012' },
        severity: 'MODERATE',
        recommendation: 'Consider reduced anticoagulation dosing or mechanical prophylaxis'
      }
      else null,

    // VTE vs Bleeding: Contraindication to pharmacologic prophylaxis
    vteBleedingConflict: if "VTE Prophylaxis Protocol Applicable" and "Has Bleeding Contraindication"
      then {
        conflictType: 'ANTICOAGULATION_BLEEDING',
        protocols: { 'VTE_CHEST_2021' },
        severity: 'HIGH',
        recommendation: 'Use mechanical prophylaxis only'
      }
      else null,

    // Diabetes vs AKI: SGLT2i and metformin concerns
    diabetesAkiConflict: if "Diabetes Protocol Applicable" and "AKI Protocol Applicable"
      then {
        conflictType: 'NEPHROTOXIC_ANTIDIABETIC',
        protocols: { 'T2DM_ADA_2024', 'AKI_KDIGO_2012' },
        severity: 'HIGH',
        recommendation: 'Hold metformin and SGLT2 inhibitors during AKI'
      }
      else null,

    // HF vs AKI: Diuretic management tension
    hfAkiConflict: if "HF GDMT Protocol Applicable" and "AKI Protocol Applicable"
      then {
        conflictType: 'DIURETIC_RENAL',
        protocols: { 'HF_GDMT_2022', 'AKI_KDIGO_2012' },
        severity: 'MODERATE',
        recommendation: 'Balance volume status with renal perfusion; consider nephrology consult'
      }
      else null
  }

define "Has Protocol Conflicts":
  "Protocol Conflicts".vteAkiConflict is not null
  or "Protocol Conflicts".vteBleedingConflict is not null
  or "Protocol Conflicts".diabetesAkiConflict is not null
  or "Protocol Conflicts".hfAkiConflict is not null

define "Conflict Count":
  Count({
    if "Protocol Conflicts".vteAkiConflict is not null then 1 else null,
    if "Protocol Conflicts".vteBleedingConflict is not null then 1 else null,
    if "Protocol Conflicts".diabetesAkiConflict is not null then 1 else null,
    if "Protocol Conflicts".hfAkiConflict is not null then 1 else null
  })

// ============================================================================
// SECTION 5: PROTOCOL CONTRAINDICATIONS
// ============================================================================

/*
 * Protocol Contraindications
 * Protocols that should be excluded based on patient factors.
 */
define "Protocol Contraindications":
  {
    // Sepsis: No absolute contraindications (treat or die)
    sepsisContraindicated: false,

    // VTE pharmacologic: Active bleeding, severe thrombocytopenia
    vtePharmacologicContraindicated: "Has Bleeding Contraindication",

    // HF GDMT: Hypotension, severe renal failure for some medications
    hfGdmtPartialContraindication: Calculators."Systolic BP Value" < 90
      or Calculators."Has Severe Renal Impairment"
      and Calculators."Potassium Value" > 5.5,

    // Diabetes intensification: Hypoglycemia risk
    diabetesIntensificationContraindicated: "Has Hypoglycemia History"
  }

define "Has Hypoglycemia History":
  exists(
    [Condition] C
      where C.code.coding.code contains 'E16.2'  // Drug-induced hypoglycemia
  )
  or exists(
    [Observation] O
      where O.code.coding.code contains '2345-7'  // Glucose
        and O.value.value < 70
        and O.effective.value >= Today() - 30 days
  )

// ============================================================================
// SECTION 6: CDSS ORCHESTRATION OUTPUT
// ============================================================================

/*
 * CDSS Orchestration Output for KB-19 consumption.
 *
 * CRITICAL ARCHITECTURE NOTE:
 * - KB-19 uses this output for RECOMMENDATIONS only
 * - ICU Intelligence has VETO authority over all recommendations
 * - deferToICU flag indicates high-risk actions requiring ICU check
 */
define "CDSS Orchestration Output":
  {
    // Primary recommendation
    recommendedProtocol: "Recommended Protocol",
    recommendedProtocolPriority: "Maximum Protocol Priority",
    protocolCount: "Applicable Protocol Count",

    // All applicable protocols
    applicableProtocols: "Applicable Protocols",

    // Conflict analysis
    hasConflicts: "Has Protocol Conflicts",
    conflicts: "Protocol Conflicts",
    conflictCount: "Conflict Count",

    // Contraindications
    contraindications: "Protocol Contraindications",

    // ICU deferral (REQUIRED per CTO/CMO directive)
    deferToICU: "Sepsis Protocol Applicable" or "AKI Protocol Applicable"
                or Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' },
    canBeOverriddenByICU: true,  // Always true

    // Evidence context for KB-15 integration
    highestEvidenceGrade: "Highest Evidence Grade",

    // Workflow context
    requiresMultidisciplinary: "Conflict Count" >= 2,
    suggestedConsults: "Suggested Consults"
  }

define "Applicable Protocol Count":
  Count({
    if "Sepsis Protocol Applicable" then 1 else null,
    if "VTE Prophylaxis Protocol Applicable" then 1 else null,
    if "HF GDMT Protocol Applicable" then 1 else null,
    if "Diabetes Protocol Applicable" then 1 else null,
    if "AKI Protocol Applicable" then 1 else null
  })

define "Highest Evidence Grade":
  // Most protocols we implement are Grade A
  if "Sepsis Protocol Applicable" or "VTE Prophylaxis Protocol Applicable"
     or "HF GDMT Protocol Applicable" or "Diabetes Protocol Applicable"
  then 'A'
  else if "AKI Protocol Applicable"
  then 'B'
  else null

define "Suggested Consults":
  {
    nephrology: "AKI Protocol Applicable" and Calculators."eGFR Value" < 30,
    cardiology: "HF GDMT Protocol Applicable" and "LVEF Value" < 40,
    infectiousDisease: "Sepsis Protocol Applicable" and "Has Active Antimicrobial Order",
    endocrinology: "Diabetes Protocol Applicable" and Calculators."HbA1c Value" > 10.0,
    hematology: "Has Bleeding Contraindication" and "VTE Prophylaxis Protocol Applicable"
  }
