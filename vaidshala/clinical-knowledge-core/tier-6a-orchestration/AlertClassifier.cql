/*
 * AlertClassifier CQL Library
 * ============================================================
 * Tier 6a Orchestration - Alert Priority Classification
 *
 * PURPOSE: ALERT CLASSIFICATION FOR KB-10 RULES ENGINE
 * This library classifies clinical alerts into priority levels
 * and categories for intelligent alert routing and fatigue reduction.
 *
 * ARCHITECTURE CRITICAL:
 * - CQL OUTPUT: "Alert priority CRITICAL, category SAFETY, requires immediate action"
 * - KB-10 CONSUMES: Priority classification to route/suppress/escalate
 * - KB-10 DOES NOT: Generate alerts (that's CQL output)
 *
 * Priority Levels:
 * - CRITICAL: Immediate action required, interruptive
 * - URGENT: Prompt action required, high visibility
 * - WARNING: Action recommended, moderate visibility
 * - INFORMATIONAL: Awareness only, low visibility
 *
 * Governance: Tier-6a review (Clinical + UX + Safety)
 * ============================================================
 */
library AlertClassifier version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// ALERT PRIORITY CODES
// ============================================================================

codesystem "AlertPriority": 'http://cardiofit.io/fhir/CodeSystem/alert-priority'
codesystem "AlertCategory": 'http://cardiofit.io/fhir/CodeSystem/alert-category'

code "Priority Critical": 'CRITICAL' from "AlertPriority"
code "Priority Urgent": 'URGENT' from "AlertPriority"
code "Priority Warning": 'WARNING' from "AlertPriority"
code "Priority Informational": 'INFORMATIONAL' from "AlertPriority"

code "Category Safety": 'SAFETY' from "AlertCategory"
code "Category Interaction": 'INTERACTION' from "AlertCategory"
code "Category Gap": 'GAP' from "AlertCategory"
code "Category Reminder": 'REMINDER' from "AlertCategory"
code "Category Guideline": 'GUIDELINE' from "AlertCategory"
code "Category Lab": 'LAB' from "AlertCategory"

// ============================================================================
// SECTION 1: DRUG INTERACTION ALERT CLASSIFICATION
// ============================================================================

/*
 * Drug Interaction Alert Priority
 * Classifies severity of drug-drug interactions for KB-10.
 */
define "Drug Interaction Alert Priority":
  case
    when "Has Contraindicated Interaction" then 'CRITICAL'
    when "Has Severe Interaction" then 'URGENT'
    when "Has Moderate Interaction" then 'WARNING'
    when "Has Minor Interaction" then 'INFORMATIONAL'
    else null
  end

define "Has Contraindicated Interaction":
  // From KB-5 snapshot - contraindicated combinations
  Safety."Has Active Drug Interaction"
    and Safety."Drug Interaction Severity" = 'contraindicated'

define "Has Severe Interaction":
  Safety."Has Active Drug Interaction"
    and Safety."Drug Interaction Severity" = 'severe'

define "Has Moderate Interaction":
  Safety."Has Active Drug Interaction"
    and Safety."Drug Interaction Severity" = 'moderate'

define "Has Minor Interaction":
  Safety."Has Active Drug Interaction"
    and Safety."Drug Interaction Severity" = 'minor'

define "Drug Interaction Alert Output":
  if "Drug Interaction Alert Priority" is not null then
    {
      alertType: 'DRUG_INTERACTION',
      priority: "Drug Interaction Alert Priority",
      category: 'INTERACTION',
      requiresImmediateAction: "Drug Interaction Alert Priority" = 'CRITICAL',
      suppressionEligible: "Drug Interaction Alert Priority" = 'INFORMATIONAL',
      interactiveMedicationPair: Safety."Interacting Drug Pair",
      clinicalEffect: Safety."Drug Interaction Effect"
    }
  else null

// ============================================================================
// SECTION 2: ALLERGY ALERT CLASSIFICATION
// ============================================================================

/*
 * Allergy Alert Priority
 * Classifies allergy/adverse reaction alerts.
 */
define "Allergy Alert Priority":
  case
    when "Has Life Threatening Allergy Alert" then 'CRITICAL'
    when "Has Severe Allergy Alert" then 'URGENT'
    when "Has Moderate Allergy Alert" then 'WARNING'
    when "Has Mild Allergy Alert" then 'INFORMATIONAL'
    else null
  end

define "Has Life Threatening Allergy Alert":
  Safety."Has Known Anaphylaxis History"
    and "Ordering Allergen Medication"

define "Has Severe Allergy Alert":
  Safety."Has Active Allergy"
    and "Ordering Allergen Medication"

define "Has Moderate Allergy Alert":
  Safety."Has Drug Sensitivity"
    and "Ordering Sensitivity Medication"

define "Has Mild Allergy Alert":
  Safety."Has Intolerance"
    and "Ordering Intolerance Medication"

define "Ordering Allergen Medication":
  // Check if current medication order matches documented allergy
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and Safety."Medication Matches Allergy"(MR.medication)
  )

define "Ordering Sensitivity Medication":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and Safety."Medication Matches Sensitivity"(MR.medication)
  )

define "Ordering Intolerance Medication":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and Safety."Medication Matches Intolerance"(MR.medication)
  )

define "Allergy Alert Output":
  if "Allergy Alert Priority" is not null then
    {
      alertType: 'ALLERGY',
      priority: "Allergy Alert Priority",
      category: 'SAFETY',
      requiresImmediateAction: "Allergy Alert Priority" in { 'CRITICAL', 'URGENT' },
      suppressionEligible: false,
      allergen: Safety."Active Allergen",
      reactionType: Safety."Expected Reaction Type"
    }
  else null

// ============================================================================
// SECTION 3: DOSE ALERT CLASSIFICATION
// ============================================================================

/*
 * Dose Alert Priority
 * Classifies dosing-related alerts (overdose, underdose, renal adjustment).
 */
define "Dose Alert Priority":
  case
    when "Has Lethal Dose Alert" then 'CRITICAL'
    when "Has Severe Overdose Alert" then 'CRITICAL'
    when "Has Moderate Overdose Alert" then 'URGENT'
    when "Has Renal Dose Adjustment Alert" then 'URGENT'
    when "Has Hepatic Dose Adjustment Alert" then 'URGENT'
    when "Has Subtherapeutic Dose Alert" then 'WARNING'
    when "Has Weight Based Adjustment Alert" then 'WARNING'
    else null
  end

define "Has Lethal Dose Alert":
  // Dose exceeds maximum safe dose by >200%
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and "Dose Exceeds Maximum By Percentage"(MR, 200)
  )

define "Has Severe Overdose Alert":
  // Dose exceeds maximum safe dose by 100-200%
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and "Dose Exceeds Maximum By Percentage"(MR, 100)
        and not "Dose Exceeds Maximum By Percentage"(MR, 200)
  )

define "Has Moderate Overdose Alert":
  // Dose exceeds maximum safe dose by 50-100%
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and "Dose Exceeds Maximum By Percentage"(MR, 50)
        and not "Dose Exceeds Maximum By Percentage"(MR, 100)
  )

define function "Dose Exceeds Maximum By Percentage"(med MedicationRequest, pct Integer):
  // Placeholder - actual dose calculation from KB-1
  false

define "Has Renal Dose Adjustment Alert":
  (Calculators."Has Severe Renal Impairment" or Calculators."Has Moderate Renal Impairment")
    and "Ordering Renally Cleared Medication"
    and not "Dose Already Adjusted for Renal"

define "Ordering Renally Cleared Medication":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/renal-elimination'
              and E.value >= 0.3  // >=30% renal elimination
        )
  )

define "Dose Already Adjusted for Renal":
  // Check if dose was already adjusted per renal function
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.note N
            where N.text ~ 'renal adjustment applied'
        )
  )

define "Has Hepatic Dose Adjustment Alert":
  (Calculators."Has Severe Hepatic Impairment" or Calculators."Has Moderate Hepatic Impairment")
    and "Ordering Hepatically Metabolized Medication"
    and not "Dose Already Adjusted for Hepatic"

define "Ordering Hepatically Metabolized Medication":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.extension E
            where E.url = 'http://cardiofit.io/fhir/StructureDefinition/hepatic-metabolism'
              and E.value >= 0.5  // >=50% hepatic metabolism
        )
  )

define "Dose Already Adjusted for Hepatic":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and exists(
          MR.note N
            where N.text ~ 'hepatic adjustment applied'
        )
  )

define "Has Subtherapeutic Dose Alert":
  // Dose below therapeutic range
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and "Dose Below Therapeutic Range"(MR)
  )

define function "Dose Below Therapeutic Range"(med MedicationRequest):
  // Placeholder - actual dose calculation from KB-1
  false

define "Has Weight Based Adjustment Alert":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and "Requires Weight Based Dosing"(MR)
        and not "Dose Calculated By Weight"(MR)
  )

define function "Requires Weight Based Dosing"(med MedicationRequest):
  // Placeholder - check medication properties
  false

define function "Dose Calculated By Weight"(med MedicationRequest):
  // Placeholder - check if dose was calculated with weight
  false

define "Dose Alert Output":
  if "Dose Alert Priority" is not null then
    {
      alertType: 'DOSING',
      priority: "Dose Alert Priority",
      category: 'SAFETY',
      requiresImmediateAction: "Dose Alert Priority" = 'CRITICAL',
      suppressionEligible: false,
      adjustmentType: "Dose Adjustment Type Needed",
      organFunction: "Relevant Organ Function"
    }
  else null

define "Dose Adjustment Type Needed":
  case
    when "Has Renal Dose Adjustment Alert" then 'RENAL'
    when "Has Hepatic Dose Adjustment Alert" then 'HEPATIC'
    when "Has Weight Based Adjustment Alert" then 'WEIGHT'
    else 'DOSE_RANGE'
  end

define "Relevant Organ Function":
  case
    when "Has Renal Dose Adjustment Alert" then Calculators."eGFR Value"
    when "Has Hepatic Dose Adjustment Alert" then Calculators."Child Pugh Score"
    else null
  end

// ============================================================================
// SECTION 4: LAB ALERT CLASSIFICATION
// ============================================================================

/*
 * Lab Alert Priority
 * Classifies abnormal lab value alerts.
 */
define "Lab Alert Priority":
  case
    when "Has Critical Lab Value" then 'CRITICAL'
    when "Has Panic Lab Value" then 'URGENT'
    when "Has Abnormal Lab Value" then 'WARNING'
    when "Has Borderline Lab Value" then 'INFORMATIONAL'
    else null
  end

define "Has Critical Lab Value":
  // Life-threatening lab values requiring immediate intervention
  Calculators."Potassium Value" < 2.5 or Calculators."Potassium Value" > 6.5
  or Calculators."Sodium Value" < 120 or Calculators."Sodium Value" > 160
  or Calculators."Blood Glucose Value" < 40 or Calculators."Blood Glucose Value" > 500
  or Calculators."Hemoglobin Value" < 5.0

define "Has Panic Lab Value":
  // Significant abnormality requiring prompt action
  (Calculators."Potassium Value" >= 2.5 and Calculators."Potassium Value" < 3.0)
  or (Calculators."Potassium Value" > 5.5 and Calculators."Potassium Value" <= 6.5)
  or (Calculators."Blood Glucose Value" >= 40 and Calculators."Blood Glucose Value" < 60)
  or (Calculators."Blood Glucose Value" > 300 and Calculators."Blood Glucose Value" <= 500)

define "Has Abnormal Lab Value":
  // Outside normal range but not critical
  Safety."Has Abnormal Lab Results"

define "Has Borderline Lab Value":
  // Trending toward abnormal
  Safety."Has Borderline Lab Results"

define "Lab Alert Output":
  if "Lab Alert Priority" is not null then
    {
      alertType: 'LAB_VALUE',
      priority: "Lab Alert Priority",
      category: 'LAB',
      requiresImmediateAction: "Lab Alert Priority" = 'CRITICAL',
      suppressionEligible: "Lab Alert Priority" = 'INFORMATIONAL',
      criticalLabs: "Critical Lab List"
    }
  else null

define "Critical Lab List":
  {
    potassium: if Calculators."Potassium Value" < 3.0 or Calculators."Potassium Value" > 5.5
               then Calculators."Potassium Value" else null,
    sodium: if Calculators."Sodium Value" < 125 or Calculators."Sodium Value" > 150
            then Calculators."Sodium Value" else null,
    glucose: if Calculators."Blood Glucose Value" < 60 or Calculators."Blood Glucose Value" > 300
             then Calculators."Blood Glucose Value" else null
  }

// ============================================================================
// SECTION 5: CARE GAP ALERT CLASSIFICATION
// ============================================================================

/*
 * Care Gap Alert Priority
 * Classifies overdue preventive care and quality measure gaps.
 */
define "Care Gap Alert Priority":
  case
    when "Has Critical Overdue Screening" then 'URGENT'
    when "Has Overdue Immunization" then 'WARNING'
    when "Has Quality Measure Gap" then 'WARNING'
    when "Has Preventive Care Due" then 'INFORMATIONAL'
    else null
  end

define "Has Critical Overdue Screening":
  // Significantly overdue cancer screenings or critical preventive care
  "Mammogram Overdue by Years" >= 3
  or "Colonoscopy Overdue by Years" >= 5
  or "Diabetic Eye Exam Overdue" and "Has Diabetes"

define "Mammogram Overdue by Years":
  // Calculate years since last mammogram for eligible patients
  if "Eligible for Mammogram" then
    years between Last([Procedure: "Mammogram Codes"].performed) and Today()
  else 0

define "Eligible for Mammogram":
  AgeInYears() >= 50 and AgeInYears() <= 74
    and Patient.gender = 'female'

define "Mammogram Codes":
  Code '77067' from "CPT"  // Screening mammography

codesystem "CPT": 'http://www.ama-assn.org/go/cpt'

define "Colonoscopy Overdue by Years":
  if "Eligible for Colonoscopy" then
    years between Last([Procedure: "Colonoscopy Codes"].performed) and Today()
  else 0

define "Eligible for Colonoscopy":
  AgeInYears() >= 45 and AgeInYears() <= 75

define "Colonoscopy Codes":
  Code '45378' from "CPT"  // Colonoscopy

define "Has Diabetes":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'E11'
  )

define "Diabetic Eye Exam Overdue":
  "Has Diabetes" and (
    not exists([Procedure: "Eye Exam Codes"] P where P.performed >= Today() - 365 days)
  )

define "Eye Exam Codes":
  Code '92004' from "CPT"  // Eye exam

define "Has Overdue Immunization":
  // Overdue flu shot, pneumonia vaccine, etc.
  "Flu Vaccine Overdue" or "Pneumonia Vaccine Overdue"

define "Flu Vaccine Overdue":
  AgeInYears() >= 65 and (
    not exists([Immunization] I
      where I.status = 'completed'
        and I.vaccineCode.coding.code contains 'flu'
        and I.occurrence >= Today() - 365 days)
  )

define "Pneumonia Vaccine Overdue":
  AgeInYears() >= 65 and (
    not exists([Immunization] I
      where I.status = 'completed'
        and I.vaccineCode.coding.code contains 'pneumococcal')
  )

define "Has Quality Measure Gap":
  // Open quality measure gaps from KB-9 snapshot
  // Placeholder - actual value from KB-9
  false

define "Has Preventive Care Due":
  // Routine preventive care coming due
  "Annual Wellness Visit Due" or "Routine Lab Work Due"

define "Annual Wellness Visit Due":
  not exists(
    [Encounter] E
      where E.status = 'finished'
        and E.type.coding.code contains 'wellness'
        and E.period.start >= Today() - 365 days
  )

define "Routine Lab Work Due":
  // Check for annual lab work
  not exists(
    [Observation] O
      where O.status = 'final'
        and O.code.coding.code contains '55231-5'  // Lipid panel
        and O.effective.value >= Today() - 365 days
  )

define "Care Gap Alert Output":
  if "Care Gap Alert Priority" is not null then
    {
      alertType: 'CARE_GAP',
      priority: "Care Gap Alert Priority",
      category: 'GAP',
      requiresImmediateAction: false,
      suppressionEligible: "Care Gap Alert Priority" = 'INFORMATIONAL',
      gapType: "Primary Care Gap Type"
    }
  else null

define "Primary Care Gap Type":
  case
    when "Has Critical Overdue Screening" then 'CRITICAL_SCREENING'
    when "Has Overdue Immunization" then 'IMMUNIZATION'
    when "Has Quality Measure Gap" then 'QUALITY_MEASURE'
    else 'PREVENTIVE_CARE'
  end

// ============================================================================
// SECTION 6: AGGREGATE ALERT CLASSIFICATION OUTPUT
// ============================================================================

/*
 * Maximum Alert Priority
 * Returns highest priority across all alert types.
 */
define "Maximum Alert Priority":
  case
    when "Drug Interaction Alert Priority" = 'CRITICAL'
         or "Allergy Alert Priority" = 'CRITICAL'
         or "Dose Alert Priority" = 'CRITICAL'
         or "Lab Alert Priority" = 'CRITICAL' then 'CRITICAL'
    when "Drug Interaction Alert Priority" = 'URGENT'
         or "Allergy Alert Priority" = 'URGENT'
         or "Dose Alert Priority" = 'URGENT'
         or "Lab Alert Priority" = 'URGENT'
         or "Care Gap Alert Priority" = 'URGENT' then 'URGENT'
    when "Drug Interaction Alert Priority" = 'WARNING'
         or "Allergy Alert Priority" = 'WARNING'
         or "Dose Alert Priority" = 'WARNING'
         or "Lab Alert Priority" = 'WARNING'
         or "Care Gap Alert Priority" = 'WARNING' then 'WARNING'
    else 'INFORMATIONAL'
  end

/*
 * Requires Immediate Action
 * True if any alert requires immediate clinical response.
 */
define "Requires Immediate Action":
  "Maximum Alert Priority" = 'CRITICAL'

/*
 * Alert Suppression Eligible
 * True if all alerts can be acknowledged without action.
 */
define "Alert Suppression Eligible":
  not "Requires Immediate Action"
    and "Maximum Alert Priority" in { 'INFORMATIONAL', 'WARNING' }
    and not ("Allergy Alert Priority" is not null)  // Never suppress allergy alerts

/*
 * Alert Escalation Path
 * Determines who should receive escalated alerts.
 */
define "Alert Escalation Path":
  case
    when "Maximum Alert Priority" = 'CRITICAL' then 'ATTENDING_PHYSICIAN'
    when "Maximum Alert Priority" = 'URGENT'
         and ("Dose Alert Priority" = 'URGENT' or "Allergy Alert Priority" = 'URGENT') then 'PHARMACIST'
    when "Maximum Alert Priority" = 'URGENT' then 'ORDERING_PROVIDER'
    else 'ORDERING_PROVIDER'
  end

/*
 * Complete alert classification for KB-10 consumption.
 */
define "Alert Classification Output":
  {
    // Aggregate classification
    maximumPriority: "Maximum Alert Priority",
    requiresImmediateAction: "Requires Immediate Action",
    suppressionEligible: "Alert Suppression Eligible",
    escalationPath: "Alert Escalation Path",

    // Individual alert outputs
    drugInteractionAlert: "Drug Interaction Alert Output",
    allergyAlert: "Allergy Alert Output",
    doseAlert: "Dose Alert Output",
    labAlert: "Lab Alert Output",
    careGapAlert: "Care Gap Alert Output",

    // Alert counts by category
    safetyAlertCount: "Safety Alert Count",
    interactionAlertCount: "Interaction Alert Count",
    gapAlertCount: "Gap Alert Count",

    // Fatigue indicators
    totalAlertCount: "Total Active Alert Count",
    alertFatigueRisk: "Total Active Alert Count" > 5
  }

define "Safety Alert Count":
  Count({
    if "Allergy Alert Priority" is not null then 1 else null,
    if "Dose Alert Priority" is not null then 1 else null,
    if "Lab Alert Priority" = 'CRITICAL' then 1 else null
  })

define "Interaction Alert Count":
  if "Drug Interaction Alert Priority" is not null then 1 else 0

define "Gap Alert Count":
  if "Care Gap Alert Priority" is not null then 1 else 0

define "Total Active Alert Count":
  "Safety Alert Count" + "Interaction Alert Count" + "Gap Alert Count"
