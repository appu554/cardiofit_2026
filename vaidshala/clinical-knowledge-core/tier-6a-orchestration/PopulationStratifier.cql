/*
 * PopulationStratifier CQL Library
 * ============================================================
 * Tier 6a Orchestration - Population Risk Stratification
 *
 * PURPOSE: STRATIFICATION FOR KB-11/KB-17 POPULATION HEALTH
 * This library classifies patients into risk strata for population
 * health management, care prioritization, and resource allocation.
 *
 * ARCHITECTURE CRITICAL:
 * - CQL OUTPUT: "Patient is HIGH risk, intervention priority 8.5"
 * - KB-11 CONSUMES: Risk strata for population analytics
 * - KB-17 CONSUMES: Registry assignment based on risk
 *
 * Risk Strata (per CMS/AHRQ standards):
 * - LOW: Healthy, preventive care focus
 * - MODERATE: 1-2 chronic conditions, stable
 * - HIGH: Multiple conditions, increasing utilization
 * - VERY HIGH: Complex, high-cost, high-risk
 *
 * Governance: Tier-6a review (Clinical + Analytics + Population Health)
 * ============================================================
 */
library PopulationStratifier version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// RISK STRATUM CODES
// ============================================================================

codesystem "RiskStratum": 'http://cardiofit.io/fhir/CodeSystem/risk-stratum'

code "Low Risk": 'LOW' from "RiskStratum"
code "Moderate Risk": 'MODERATE' from "RiskStratum"
code "High Risk": 'HIGH' from "RiskStratum"
code "Very High Risk": 'VERY_HIGH' from "RiskStratum"

// ============================================================================
// SECTION 1: DOMAIN-SPECIFIC RISK STRATA
// ============================================================================

/*
 * Cardiovascular Risk Stratum
 * Uses ASCVD risk score and presence of CVD conditions.
 * OUTPUT: Used by KB-11 for CVD population cohorts.
 */
define "Cardiovascular Risk Stratum":
  case
    when "Has Established ASCVD" then 'VERY_HIGH'
    when Calculators."ASCVD Risk Score" >= 20.0 then 'VERY_HIGH'
    when Calculators."ASCVD Risk Score" >= 10.0 then 'HIGH'
    when Calculators."ASCVD Risk Score" >= 5.0 then 'MODERATE'
    when Calculators."ASCVD Risk Score" >= 0 then 'LOW'
    else 'LOW'
  end

define "Has Established ASCVD":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (
          C.code.coding.code contains '414.9'   // CAD
          or C.code.coding.code contains '433.10' // Carotid stenosis
          or C.code.coding.code contains '440.20' // PAD
          or C.code.coding.code contains '434.91' // Stroke
          or C.code.coding.code contains 'I25.10' // CAD ICD-10
          or C.code.coding.code contains 'I63'    // Stroke ICD-10
        )
  )

/*
 * Metabolic Risk Stratum
 * Diabetes control, obesity, metabolic syndrome markers.
 */
define "Metabolic Risk Stratum":
  case
    when "Has Uncontrolled Diabetes" and "Has Diabetic Complications" then 'VERY_HIGH'
    when "Has Uncontrolled Diabetes" then 'HIGH'
    when "Has Diabetes" and not "Has Good Glycemic Control" then 'MODERATE'
    when "Has Prediabetes" or "Has Metabolic Syndrome" then 'MODERATE'
    when "Has Diabetes" and "Has Good Glycemic Control" then 'MODERATE'
    else 'LOW'
  end

define "Has Diabetes":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (
          C.code.coding.code contains 'E11'  // Type 2 DM
          or C.code.coding.code contains 'E10' // Type 1 DM
          or C.code.coding.code contains '250'  // DM ICD-9
        )
  )

define "Has Prediabetes":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'R73.03'
  )

define "Has Uncontrolled Diabetes":
  "Has Diabetes" and Calculators."HbA1c Value" > 9.0

define "Has Good Glycemic Control":
  "Has Diabetes" and Calculators."HbA1c Value" <= 7.0

define "Has Diabetic Complications":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (
          C.code.coding.code contains 'E11.2'  // Nephropathy
          or C.code.coding.code contains 'E11.3'  // Retinopathy
          or C.code.coding.code contains 'E11.4'  // Neuropathy
          or C.code.coding.code contains 'E11.5'  // PVD
        )
  )

define "Has Metabolic Syndrome":
  // 3 or more of: abdominal obesity, high triglycerides, low HDL, high BP, high fasting glucose
  Count({
    if "Has Abdominal Obesity" then 1 else null,
    if "Has High Triglycerides" then 1 else null,
    if "Has Low HDL" then 1 else null,
    if "Has Elevated Blood Pressure" then 1 else null,
    if "Has Elevated Fasting Glucose" then 1 else null
  }) >= 3

define "Has Abdominal Obesity":
  Calculators."BMI Value" >= 30

define "Has High Triglycerides":
  exists(
    [Observation: "Triglycerides Code"] O
      where O.status = 'final'
        and O.value.value >= 150
  )

define "Triglycerides Code":
  Code '2571-8' from "LOINC"

define "Has Low HDL":
  exists(
    [Observation: "HDL Code"] O
      where O.status = 'final'
        and O.value.value < 40
  )

define "HDL Code":
  Code '2085-9' from "LOINC"

define "Has Elevated Blood Pressure":
  Calculators."Systolic BP Value" >= 130 or Calculators."Diastolic BP Value" >= 85

define "Has Elevated Fasting Glucose":
  Calculators."Blood Glucose Value" >= 100

codesystem "LOINC": 'http://loinc.org'

/*
 * Renal Risk Stratum
 * CKD staging based on eGFR and proteinuria.
 */
define "Renal Risk Stratum":
  case
    when Calculators."Has Severe Renal Impairment" then 'VERY_HIGH'
    when Calculators."Has Moderate Renal Impairment" then 'HIGH'
    when Calculators."Has Mild Renal Impairment" then 'MODERATE'
    else 'LOW'
  end

/*
 * Safety Risk Stratum
 * From SafetyCommon library - medication/clinical safety risks.
 */
define "Safety Risk Stratum":
  case
    when Safety."Safety Risk Level" = 'CRITICAL' then 'VERY_HIGH'
    when Safety."Safety Risk Level" = 'HIGH' then 'HIGH'
    when Safety."Safety Risk Level" = 'MODERATE' then 'MODERATE'
    else 'LOW'
  end

/*
 * Respiratory Risk Stratum
 * COPD, asthma control, and pulmonary function.
 */
define "Respiratory Risk Stratum":
  case
    when "Has Severe COPD" then 'VERY_HIGH'
    when "Has COPD with Exacerbations" then 'HIGH'
    when "Has Moderate COPD or Uncontrolled Asthma" then 'MODERATE'
    else 'LOW'
  end

define "Has Severe COPD":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'J44'
        and C.severity.coding.code contains 'severe'
  )

define "Has COPD with Exacerbations":
  "Has COPD" and "Recent COPD Exacerbation Count" >= 2

define "Has COPD":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'J44'
  )

define "Recent COPD Exacerbation Count":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.period.start >= Today() - 365 days
        and exists(
          E.reasonCode RC
            where RC.coding.code contains 'J44.1'  // COPD exacerbation
        )
  )

define "Has Moderate COPD or Uncontrolled Asthma":
  "Has COPD" or "Has Uncontrolled Asthma"

define "Has Uncontrolled Asthma":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'J45'
  )
  and "Recent Asthma Exacerbation Count" >= 2

define "Recent Asthma Exacerbation Count":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.period.start >= Today() - 365 days
        and E.type.coding.code contains 'emergency'
        and exists(
          E.reasonCode RC
            where RC.coding.code contains 'J45'
        )
  )

// ============================================================================
// SECTION 2: OVERALL RISK STRATUM (COMPOSITE)
// ============================================================================

/*
 * Overall Risk Stratum
 * Combines domain-specific strata into single risk classification.
 * Highest risk across domains determines overall stratum.
 */
define "Overall Risk Stratum":
  if "Has Very High Risk in Any Domain" then 'VERY_HIGH'
  else if "Has High Risk in Multiple Domains" or "Has High Risk in Any Domain" then 'HIGH'
  else if "Has Moderate Risk in Any Domain" then 'MODERATE'
  else 'LOW'

define "Domain Risk Levels":
  {
    cvRisk: "Cardiovascular Risk Stratum",
    metabolicRisk: "Metabolic Risk Stratum",
    renalRisk: "Renal Risk Stratum",
    safetyRisk: "Safety Risk Stratum",
    respiratoryRisk: "Respiratory Risk Stratum"
  }

define "Has Very High Risk in Any Domain":
  "Cardiovascular Risk Stratum" = 'VERY_HIGH'
  or "Metabolic Risk Stratum" = 'VERY_HIGH'
  or "Renal Risk Stratum" = 'VERY_HIGH'
  or "Safety Risk Stratum" = 'VERY_HIGH'
  or "Respiratory Risk Stratum" = 'VERY_HIGH'

define "Has High Risk in Any Domain":
  "Cardiovascular Risk Stratum" = 'HIGH'
  or "Metabolic Risk Stratum" = 'HIGH'
  or "Renal Risk Stratum" = 'HIGH'
  or "Safety Risk Stratum" = 'HIGH'
  or "Respiratory Risk Stratum" = 'HIGH'

define "Has High Risk in Multiple Domains":
  Count({
    if "Cardiovascular Risk Stratum" in { 'HIGH', 'VERY_HIGH' } then 1 else null,
    if "Metabolic Risk Stratum" in { 'HIGH', 'VERY_HIGH' } then 1 else null,
    if "Renal Risk Stratum" in { 'HIGH', 'VERY_HIGH' } then 1 else null,
    if "Safety Risk Stratum" in { 'HIGH', 'VERY_HIGH' } then 1 else null,
    if "Respiratory Risk Stratum" in { 'HIGH', 'VERY_HIGH' } then 1 else null
  }) >= 2

define "Has Moderate Risk in Any Domain":
  "Cardiovascular Risk Stratum" = 'MODERATE'
  or "Metabolic Risk Stratum" = 'MODERATE'
  or "Renal Risk Stratum" = 'MODERATE'
  or "Safety Risk Stratum" = 'MODERATE'
  or "Respiratory Risk Stratum" = 'MODERATE'

// ============================================================================
// SECTION 3: INTERVENTION PRIORITY SCORING
// ============================================================================

/*
 * Intervention Priority Score
 * Numeric score (0-10) for ranking patients by intervention urgency.
 * Used by KB-11/KB-17 for outreach prioritization.
 */
define "Intervention Priority Score":
  Min({
    10.0,
    "Base Risk Score"
    + "Gap Burden Score"
    + "Utilization Score"
    + "Rising Risk Bonus"
  })

define "Base Risk Score":
  case
    when "Overall Risk Stratum" = 'VERY_HIGH' then 4.0
    when "Overall Risk Stratum" = 'HIGH' then 3.0
    when "Overall Risk Stratum" = 'MODERATE' then 2.0
    else 1.0
  end

define "Gap Burden Score":
  // Add points for each open care gap
  Min({
    3.0,
    ("Open Care Gap Count" * 0.5)
  })

define "Open Care Gap Count":
  // Count of open quality measure gaps - placeholder
  // Actual value from KB-9 snapshot
  0

define "Utilization Score":
  // Points for recent high utilization
  case
    when "Recent ED Visits" >= 3 then 2.0
    when "Recent ED Visits" >= 1 then 1.0
    when "Recent Hospitalizations" >= 1 then 1.5
    else 0.0
  end

define "Recent ED Visits":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.class.code = 'EMER'
        and E.period.start >= Today() - 180 days
  )

define "Recent Hospitalizations":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.class.code = 'IMP'
        and E.period.start >= Today() - 365 days
  )

define "Rising Risk Bonus":
  // Additional points for trending toward higher risk
  if "Rising Risk Indicator" then 1.0 else 0.0

// ============================================================================
// SECTION 4: RISING RISK INDICATOR
// ============================================================================

/*
 * Rising Risk Indicator
 * Identifies patients trending toward higher risk category.
 * Key for proactive intervention before deterioration.
 */
define "Rising Risk Indicator":
  "Has Worsening Glycemic Control"
  or "Has Declining Renal Function"
  or "Has Increasing Utilization"
  or "Has New High Risk Condition"

define "Has Worsening Glycemic Control":
  "Has Diabetes" and (
    exists(
      [Observation: "HbA1c Code"] Current
        let Prior: Last(
          [Observation: "HbA1c Code"] P
            where P.effective.value < Current.effective.value - 90 days
        )
        where Current.status = 'final'
          and Prior is not null
          and Current.value.value > Prior.value.value + 0.5
    )
  )

define "HbA1c Code":
  Code '4548-4' from "LOINC"

define "Has Declining Renal Function":
  exists(
    [Observation: "eGFR Code"] Current
      let Prior: Last(
        [Observation: "eGFR Code"] P
          where P.effective.value < Current.effective.value - 180 days
      )
      where Current.status = 'final'
        and Prior is not null
        and Current.value.value < Prior.value.value - 10
  )

define "eGFR Code":
  Code '33914-3' from "LOINC"

define "Has Increasing Utilization":
  "Recent ED Visits" > "Prior Period ED Visits"
  or "Recent Hospitalizations" > "Prior Period Hospitalizations"

define "Prior Period ED Visits":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.class.code = 'EMER'
        and E.period.start >= Today() - 365 days
        and E.period.start < Today() - 180 days
  )

define "Prior Period Hospitalizations":
  Count(
    [Encounter] E
      where E.status = 'finished'
        and E.class.code = 'IMP'
        and E.period.start >= Today() - 730 days
        and E.period.start < Today() - 365 days
  )

define "Has New High Risk Condition":
  // New diagnosis of high-risk condition in past 90 days
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.onset.value >= Today() - 90 days
        and (
          C.code.coding.code contains 'I50'   // Heart failure
          or C.code.coding.code contains 'N18' // CKD
          or C.code.coding.code contains 'J44' // COPD
          or C.code.coding.code contains 'E11' // Diabetes
        )
  )

// ============================================================================
// SECTION 5: CARE MANAGEMENT ELIGIBILITY
// ============================================================================

/*
 * Eligible for Care Management
 * Intensive care coordination for highest risk patients.
 */
define "Eligible for Care Management":
  "Overall Risk Stratum" = 'VERY_HIGH'
  or ("Overall Risk Stratum" = 'HIGH' and "Intervention Priority Score" >= 7.0)
  or "Has Complex Care Needs"

define "Has Complex Care Needs":
  // Multiple chronic conditions + high utilization + social determinants
  "Active Chronic Condition Count" >= 5
  or ("Active Chronic Condition Count" >= 3 and "Recent Hospitalizations" >= 2)
  or "Has High Social Risk"

define "Active Chronic Condition Count":
  Count(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.category.coding.code contains 'problem-list-item'
  )

define "Has High Social Risk":
  // SDOH risk indicators
  exists(
    [Observation: "Social Risk Codes"] O
      where O.status = 'final'
        and O.value.coding.code contains 'at-risk'
  )

define "Social Risk Codes":
  Code '71802-3' from "LOINC"  // Housing instability

/*
 * Eligible for Disease Management
 * Condition-specific programs for stable chronic disease.
 */
define "Eligible for Disease Management":
  not "Eligible for Care Management"
  and (
    "Eligible for Diabetes Management"
    or "Eligible for Heart Failure Management"
    or "Eligible for COPD Management"
    or "Eligible for CKD Management"
  )

define "Eligible for Diabetes Management":
  "Has Diabetes"
  and "Metabolic Risk Stratum" in { 'MODERATE', 'HIGH' }
  and not "Has Good Glycemic Control"

define "Eligible for Heart Failure Management":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'I50'
  )

define "Eligible for COPD Management":
  "Has COPD"
  and "Respiratory Risk Stratum" in { 'MODERATE', 'HIGH' }

define "Eligible for CKD Management":
  Calculators."Has Moderate Renal Impairment"
  and not Calculators."Has Severe Renal Impairment"

// ============================================================================
// SECTION 6: POPULATION STRATIFICATION OUTPUT
// ============================================================================

/*
 * Complete population stratification for KB-11/KB-17 consumption.
 */
define "Population Stratification Output":
  {
    // Overall classification
    overallStratum: "Overall Risk Stratum",
    priorityScore: "Intervention Priority Score",
    risingRisk: "Rising Risk Indicator",

    // Domain-specific strata
    cardiovascularStratum: "Cardiovascular Risk Stratum",
    metabolicStratum: "Metabolic Risk Stratum",
    renalStratum: "Renal Risk Stratum",
    safetyStratum: "Safety Risk Stratum",
    respiratoryStratum: "Respiratory Risk Stratum",

    // Program eligibility
    eligibleCareManagement: "Eligible for Care Management",
    eligibleDiseaseManagement: "Eligible for Disease Management",

    // Component scores
    utilizationScore: "Utilization Score",
    gapBurdenScore: "Gap Burden Score",

    // Key indicators
    edVisits180Days: "Recent ED Visits",
    hospitalizations365Days: "Recent Hospitalizations",
    activeConditionCount: "Active Chronic Condition Count",

    // Flags
    hasComplexNeeds: "Has Complex Care Needs",
    hasHighSocialRisk: "Has High Social Risk"
  }
