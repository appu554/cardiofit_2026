/*
 * MaternalHealthCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Maternal/Obstetric Clinical Patterns
 *
 * Purpose: Shared maternal health definitions consumed by:
 *   - ICU Intelligence (Tier 7) - Obstetric emergencies
 *   - KB-3 Guidelines (Prenatal care, L&D protocols)
 *   - KB-4 Patient Safety (Teratogenic drugs, pregnancy categories)
 *   - Regional Adapters (India ICMR, Australia RACGP)
 *
 * Clinical Domains:
 *   - Pregnancy Status and Gestational Age
 *   - Preeclampsia/Eclampsia (ACOG criteria)
 *   - Obstetric Hemorrhage
 *   - Gestational Diabetes
 *   - Fetal Monitoring
 *   - High-Risk Pregnancy Indicators
 *   - Postpartum Complications
 *
 * CRITICAL SAFETY NOTE:
 *   This library directly impacts medication safety for pregnant
 *   patients. All teratogenic drug alerts depend on accurate
 *   pregnancy status detection.
 *
 * Governance: Tier-3 review required (Clinical + Technical + OB/GYN)
 * ============================================================
 */
library MaternalHealthCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// Pregnancy Conditions
valueset "Pregnancy Conditions": 'http://cardiofit.io/fhir/ValueSet/pregnancy'
valueset "Normal Pregnancy": 'http://cardiofit.io/fhir/ValueSet/pregnancy-normal'
valueset "High Risk Pregnancy": 'http://cardiofit.io/fhir/ValueSet/pregnancy-high-risk'
valueset "Multiple Gestation": 'http://cardiofit.io/fhir/ValueSet/multiple-gestation'
valueset "Ectopic Pregnancy": 'http://cardiofit.io/fhir/ValueSet/ectopic-pregnancy'

// Hypertensive Disorders of Pregnancy
valueset "Gestational Hypertension": 'http://cardiofit.io/fhir/ValueSet/gestational-htn'
valueset "Preeclampsia Conditions": 'http://cardiofit.io/fhir/ValueSet/preeclampsia'
valueset "Severe Preeclampsia": 'http://cardiofit.io/fhir/ValueSet/preeclampsia-severe'
valueset "Eclampsia Conditions": 'http://cardiofit.io/fhir/ValueSet/eclampsia'
valueset "HELLP Syndrome": 'http://cardiofit.io/fhir/ValueSet/hellp-syndrome'

// Gestational Diabetes
valueset "Gestational Diabetes": 'http://cardiofit.io/fhir/ValueSet/gdm'

// Hemorrhage
valueset "Antepartum Hemorrhage": 'http://cardiofit.io/fhir/ValueSet/antepartum-hemorrhage'
valueset "Postpartum Hemorrhage": 'http://cardiofit.io/fhir/ValueSet/postpartum-hemorrhage'
valueset "Placenta Previa": 'http://cardiofit.io/fhir/ValueSet/placenta-previa'
valueset "Placental Abruption": 'http://cardiofit.io/fhir/ValueSet/placental-abruption'

// Labor and Delivery
valueset "Preterm Labor": 'http://cardiofit.io/fhir/ValueSet/preterm-labor'
valueset "PROM Conditions": 'http://cardiofit.io/fhir/ValueSet/prom'
valueset "Cesarean Delivery": 'http://cardiofit.io/fhir/ValueSet/cesarean-delivery'
valueset "Vaginal Delivery": 'http://cardiofit.io/fhir/ValueSet/vaginal-delivery'

// Postpartum
valueset "Postpartum Conditions": 'http://cardiofit.io/fhir/ValueSet/postpartum'
valueset "Postpartum Depression": 'http://cardiofit.io/fhir/ValueSet/postpartum-depression'

// Teratogenic Medications (KB-4 input)
valueset "Teratogenic Medications Category X": 'http://cardiofit.io/fhir/ValueSet/teratogenic-category-x'
valueset "Teratogenic Medications Category D": 'http://cardiofit.io/fhir/ValueSet/teratogenic-category-d'
valueset "ACE Inhibitors": 'http://cardiofit.io/fhir/ValueSet/ace-inhibitors'
valueset "ARBs": 'http://cardiofit.io/fhir/ValueSet/arbs'
valueset "Warfarin": 'http://cardiofit.io/fhir/ValueSet/warfarin'
valueset "Statins": 'http://cardiofit.io/fhir/ValueSet/statins'
valueset "Methotrexate": 'http://cardiofit.io/fhir/ValueSet/methotrexate'

// Lactation-Related
valueset "Breastfeeding": 'http://cardiofit.io/fhir/ValueSet/breastfeeding'

// ============================================================
// CODESYSTEMS
// ============================================================
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

// ============================================================
// OBSERVATION CODES (LOINC)
// ============================================================

// Pregnancy Tests
code "Pregnancy Test Serum Code": '2106-3' from "LOINC" display 'Choriogonadotropin (pregnancy test) in Serum'
code "Pregnancy Test Urine Code": '2118-8' from "LOINC" display 'Choriogonadotropin (pregnancy test) in Urine'
code "hCG Quantitative Code": '19080-1' from "LOINC" display 'Choriogonadotropin [Units/volume] in Serum'

// Gestational Age
code "Gestational Age Code": '18185-9' from "LOINC" display 'Gestational age'
code "EDD Code": '11778-8' from "LOINC" display 'Estimated date of delivery'
code "LMP Code": '8665-2' from "LOINC" display 'Last menstrual period start date'

// Blood Pressure (duplicate for maternal context)
code "Systolic BP Code": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP Code": '8462-4' from "LOINC" display 'Diastolic blood pressure'

// Preeclampsia Labs
code "Urine Protein Code": '5804-0' from "LOINC" display 'Protein [Mass/volume] in Urine'
code "Urine Protein Creatinine Ratio": '34366-5' from "LOINC" display 'Protein/Creatinine ratio in Urine'
code "24h Urine Protein Code": '2889-4' from "LOINC" display 'Protein [Mass/24 hours] in 24 hour Urine'
code "Platelet Count Code": '777-3' from "LOINC" display 'Platelets [#/volume] in Blood'
code "AST Code": '1920-8' from "LOINC" display 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum'
code "ALT Code": '1742-6' from "LOINC" display 'Alanine aminotransferase [Enzymatic activity/volume] in Serum'
code "Creatinine Code": '2160-0' from "LOINC" display 'Creatinine [Mass/volume] in Serum'
code "LDH Code": '2532-0' from "LOINC" display 'Lactate dehydrogenase [Enzymatic activity/volume] in Serum'
code "Uric Acid Code": '3084-1' from "LOINC" display 'Urate [Mass/volume] in Serum'

// Fetal Monitoring
code "Fetal Heart Rate Code": '55283-6' from "LOINC" display 'Fetal Heart rate'
code "Fetal Movement Code": '57088-7' from "LOINC" display 'Fetal movement'

// Hemoglobin (for hemorrhage)
code "Hemoglobin Code": '718-7' from "LOINC" display 'Hemoglobin [Mass/volume] in Blood'

// GDM Screening
code "Glucose 1h Post 50g Code": '20437-0' from "LOINC" display 'Glucose [Mass/volume] in Serum 1 hour post 50 g glucose'
code "Glucose 1h Post 75g Code": '20438-8' from "LOINC" display 'Glucose [Mass/volume] in Serum 1 hour post 75 g glucose'
code "Glucose Fasting Code": '1558-6' from "LOINC" display 'Fasting glucose [Mass/volume] in Serum'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// PREGNANCY STATUS DETECTION (CRITICAL FOR DRUG SAFETY)
// ============================================================

/*
 * Is Female of Childbearing Age (15-50)
 */
define "Is Female of Childbearing Age":
  Patient.gender = 'female'
  and AgeInYears() >= 15
  and AgeInYears() <= 50

/*
 * Active Pregnancy Condition
 */
define "Has Active Pregnancy Condition":
  exists [Condition: "Pregnancy Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Recent Positive Pregnancy Test (within 40 weeks)
 */
define "Recent Positive Pregnancy Test":
  exists ([Observation: "Pregnancy Test Serum Code"]
          union [Observation: "Pregnancy Test Urine Code"]) O
    where Observations.IsValidObservation(O)
    and Observations.ObservationDateTime(O) >= Now() - 280 days
    and (O.value as FHIR.CodeableConcept) ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding {
        system: FHIR.uri { value: 'http://snomed.info/sct' },
        code: FHIR.code { value: '10828004' } // Positive
      } }
    })

/*
 * Is Currently Pregnant (CRITICAL FLAG)
 * Used by KB-4 for teratogenic drug alerts
 */
define "Is Currently Pregnant":
  "Has Active Pregnancy Condition" or "Recent Positive Pregnancy Test"

/*
 * Possible Pregnancy (needs confirmation)
 * For drug safety: treat as pregnant until proven otherwise
 */
define "Possible Pregnancy":
  "Is Female of Childbearing Age"
  and not exists [Observation] O
    where O.code ~ "Pregnancy Test Serum Code"
    and Observations.ObservationDateTime(O) >= Now() - 7 days

// ============================================================
// GESTATIONAL AGE
// ============================================================

define "Most Recent Gestational Age":
  Observations.MostRecentObservation(
    [Observation: "Gestational Age Code"] O
      where Observations.IsValidObservation(O)
  )

define "Gestational Age Weeks":
  Observations.ObservationNumericValue("Most Recent Gestational Age")

/*
 * Trimester Classification
 */
define "First Trimester":
  "Gestational Age Weeks" < 14

define "Second Trimester":
  "Gestational Age Weeks" >= 14 and "Gestational Age Weeks" < 28

define "Third Trimester":
  "Gestational Age Weeks" >= 28

/*
 * Preterm Pregnancy (< 37 weeks)
 */
define "Is Preterm":
  "Gestational Age Weeks" < 37 and "Gestational Age Weeks" >= 20

/*
 * Term Pregnancy (37-41 weeks)
 */
define "Is Term":
  "Gestational Age Weeks" >= 37 and "Gestational Age Weeks" <= 41

/*
 * Post-term Pregnancy (> 41 weeks)
 */
define "Is Post Term":
  "Gestational Age Weeks" > 41

/*
 * Viable Pregnancy (> 24 weeks typically)
 */
define "Is Viable Gestation":
  "Gestational Age Weeks" >= 24

// ============================================================
// BLOOD PRESSURE IN PREGNANCY
// ============================================================

define "Most Recent Systolic BP":
  Observations.MostRecentObservation(
    [Observation: "Systolic BP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Systolic BP Value":
  Observations.ObservationNumericValue("Most Recent Systolic BP")

define "Most Recent Diastolic BP":
  Observations.MostRecentObservation(
    [Observation: "Diastolic BP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Diastolic BP Value":
  Observations.ObservationNumericValue("Most Recent Diastolic BP")

/*
 * Hypertension in Pregnancy: SBP >= 140 OR DBP >= 90
 */
define "Has Hypertension in Pregnancy":
  "Is Currently Pregnant"
  and ("Most Recent Systolic BP Value" >= 140 or "Most Recent Diastolic BP Value" >= 90)

/*
 * Severe Hypertension in Pregnancy: SBP >= 160 OR DBP >= 110
 * CRITICAL: Urgent intervention needed
 */
define "Has Severe Hypertension in Pregnancy":
  "Is Currently Pregnant"
  and ("Most Recent Systolic BP Value" >= 160 or "Most Recent Diastolic BP Value" >= 110)

// ============================================================
// PREECLAMPSIA DETECTION (ACOG Criteria)
// ============================================================

define "Most Recent Urine Protein Creatinine Ratio":
  Observations.MostRecentObservation(
    [Observation: "Urine Protein Creatinine Ratio"] O
      where Observations.IsValidObservation(O)
  )

define "UPCR Value":
  Observations.ObservationNumericValue("Most Recent Urine Protein Creatinine Ratio")

define "Most Recent 24h Urine Protein":
  Observations.MostRecentObservation(
    [Observation: "24h Urine Protein Code"] O
      where Observations.IsValidObservation(O)
  )

define "24h Protein Value":
  Observations.ObservationNumericValue("Most Recent 24h Urine Protein")

/*
 * Significant Proteinuria: UPCR >= 0.3 OR 24h protein >= 300mg
 */
define "Has Significant Proteinuria":
  "UPCR Value" >= 0.3 or "24h Protein Value" >= 300

define "Most Recent Platelet Count":
  Observations.MostRecentObservation(
    [Observation: "Platelet Count Code"] O
      where Observations.IsValidObservation(O)
  )

define "Platelet Count Value":
  Observations.ObservationNumericValue("Most Recent Platelet Count")

define "Most Recent AST":
  Observations.MostRecentObservation(
    [Observation: "AST Code"] O
      where Observations.IsValidObservation(O)
  )

define "AST Value":
  Observations.ObservationNumericValue("Most Recent AST")

define "Most Recent ALT":
  Observations.MostRecentObservation(
    [Observation: "ALT Code"] O
      where Observations.IsValidObservation(O)
  )

define "ALT Value":
  Observations.ObservationNumericValue("Most Recent ALT")

define "Most Recent Creatinine":
  Observations.MostRecentObservation(
    [Observation: "Creatinine Code"] O
      where Observations.IsValidObservation(O)
  )

define "Creatinine Value":
  Observations.ObservationNumericValue("Most Recent Creatinine")

/*
 * Preeclampsia Biochemical Criteria (ACOG 2020)
 * HTN + Proteinuria OR end-organ dysfunction
 */
define "Meets Preeclampsia Criteria":
  "Is Currently Pregnant"
  and "Gestational Age Weeks" >= 20
  and "Has Hypertension in Pregnancy"
  and ("Has Significant Proteinuria"
       or "Platelet Count Value" < 100000
       or "Creatinine Value" > 1.1
       or "AST Value" > 70 or "ALT Value" > 70)

/*
 * Active Preeclampsia Diagnosis
 */
define "Has Active Preeclampsia":
  exists [Condition: "Preeclampsia Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Severe Preeclampsia Criteria
 * CRITICAL for ICU Intelligence
 */
define "Meets Severe Preeclampsia Criteria":
  "Meets Preeclampsia Criteria"
  and ("Has Severe Hypertension in Pregnancy"
       or "Platelet Count Value" < 100000
       or "AST Value" > 70 or "ALT Value" > 70
       or "Creatinine Value" > 1.1)

define "Has Active Severe Preeclampsia":
  exists [Condition: "Severe Preeclampsia"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * HELLP Syndrome
 * CRITICAL: ICU-level emergency
 */
define "Meets HELLP Criteria":
  "Is Currently Pregnant"
  and "Platelet Count Value" < 100000
  and ("AST Value" > 70 or "ALT Value" > 70)
  // Hemolysis indicators would need LDH, peripheral smear

define "Has Active HELLP":
  exists [Condition: "HELLP Syndrome"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Active Eclampsia (seizures in preeclampsia)
 * CRITICAL: Immediate ICU emergency
 */
define "Has Active Eclampsia":
  exists [Condition: "Eclampsia Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// OBSTETRIC HEMORRHAGE (CRITICAL FOR ICU)
// ============================================================

define "Most Recent Hemoglobin":
  Observations.MostRecentObservation(
    [Observation: "Hemoglobin Code"] O
      where Observations.IsValidObservation(O)
  )

define "Hemoglobin Value":
  Observations.ObservationNumericValue("Most Recent Hemoglobin")

/*
 * Anemia in Pregnancy: Hgb < 11 g/dL (1st/3rd trimester)
 */
define "Has Anemia in Pregnancy":
  "Is Currently Pregnant" and "Hemoglobin Value" < 11

/*
 * Severe Anemia: Hgb < 7 g/dL
 * CRITICAL: Transfusion likely needed
 */
define "Has Severe Anemia":
  "Hemoglobin Value" < 7

/*
 * Active Antepartum Hemorrhage
 */
define "Has Active Antepartum Hemorrhage":
  exists [Condition: "Antepartum Hemorrhage"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Active Postpartum Hemorrhage
 * CRITICAL: ICU emergency
 */
define "Has Active Postpartum Hemorrhage":
  exists [Condition: "Postpartum Hemorrhage"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Placenta Previa
 */
define "Has Placenta Previa":
  exists [Condition: "Placenta Previa"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Placental Abruption
 * CRITICAL: Emergency delivery consideration
 */
define "Has Placental Abruption":
  exists [Condition: "Placental Abruption"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// HIGH-RISK PREGNANCY FLAGS
// ============================================================

define "Has High Risk Pregnancy Diagnosis":
  exists [Condition: "High Risk Pregnancy"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Multiple Gestation":
  exists [Condition: "Multiple Gestation"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Gestational Diabetes":
  exists [Condition: "Gestational Diabetes"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Gestational Hypertension":
  exists [Condition: "Gestational Hypertension"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Preterm Labor":
  exists [Condition: "Preterm Labor"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has PROM":
  exists [Condition: "PROM Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Is High Risk Pregnancy (composite)
 */
define "Is High Risk Pregnancy":
  "Has High Risk Pregnancy Diagnosis"
  or "Has Multiple Gestation"
  or "Has Gestational Diabetes"
  or "Has Active Preeclampsia"
  or "Has Placenta Previa"
  or AgeInYears() >= 35
  or AgeInYears() < 18

// ============================================================
// TERATOGENIC DRUG EXPOSURE (CRITICAL FOR KB-4)
// ============================================================

/*
 * On Category X Medications (CONTRAINDICATED in pregnancy)
 * CRITICAL SAFETY FLAG
 */
define "On Category X Medication":
  exists [MedicationRequest: "Teratogenic Medications Category X"] MR
    where Medications.IsMedicationActive(MR)

/*
 * On Warfarin (Category X except mechanical valve)
 */
define "On Warfarin":
  exists [MedicationRequest: "Warfarin"] MR
    where Medications.IsMedicationActive(MR)

/*
 * On ACE Inhibitor or ARB (Category D - 2nd/3rd trimester)
 */
define "On ACEi or ARB":
  exists ([MedicationRequest: "ACE Inhibitors"]
          union [MedicationRequest: "ARBs"]) MR
    where Medications.IsMedicationActive(MR)

/*
 * On Statin (Category X)
 */
define "On Statin":
  exists [MedicationRequest: "Statins"] MR
    where Medications.IsMedicationActive(MR)

/*
 * On Methotrexate (Category X - abortifacient)
 */
define "On Methotrexate":
  exists [MedicationRequest: "Methotrexate"] MR
    where Medications.IsMedicationActive(MR)

/*
 * Pregnant on Teratogenic Medication
 * CRITICAL SAFETY ALERT for KB-4
 */
define "Pregnant On Teratogenic Medication":
  "Is Currently Pregnant"
  and ("On Category X Medication" or "On Warfarin" or "On Statin" or "On Methotrexate")

/*
 * Pregnant on ACEi/ARB in 2nd/3rd Trimester
 * CRITICAL: Fetal renal damage risk
 */
define "Pregnant On ACEi ARB Late Term":
  "Is Currently Pregnant"
  and "On ACEi or ARB"
  and ("Second Trimester" or "Third Trimester")

// ============================================================
// LACTATION STATUS
// ============================================================

define "Is Breastfeeding":
  exists [Condition: "Breastfeeding"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Is Postpartum":
  exists [Condition: "Postpartum Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// FETAL MONITORING
// ============================================================

define "Most Recent Fetal Heart Rate":
  Observations.MostRecentObservation(
    [Observation: "Fetal Heart Rate Code"] O
      where Observations.IsValidObservation(O)
  )

define "Fetal Heart Rate Value":
  Observations.ObservationNumericValue("Most Recent Fetal Heart Rate")

/*
 * Fetal Bradycardia: FHR < 110 bpm
 */
define "Has Fetal Bradycardia":
  "Fetal Heart Rate Value" < 110

/*
 * Fetal Tachycardia: FHR > 160 bpm
 */
define "Has Fetal Tachycardia":
  "Fetal Heart Rate Value" > 160

/*
 * Absent Fetal Heart Tones (FHR = 0 or not detected)
 * CRITICAL: Potential fetal demise
 */
define "Has Absent Fetal Heart Tones":
  "Is Currently Pregnant"
  and "Is Viable Gestation"
  and ("Fetal Heart Rate Value" = 0 or "Fetal Heart Rate Value" is null)

// ============================================================
// MATERNAL EMERGENCY RISK SCORING
// ============================================================

/*
 * Maternal Emergency Risk Score
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. ICU DECIDES actions.
 */
define "Maternal Emergency Risk Score":
  (if "Has Active Eclampsia" then 5 else
   if "Has Active Severe Preeclampsia" or "Meets Severe Preeclampsia Criteria" then 4 else
   if "Has Active Preeclampsia" or "Meets Preeclampsia Criteria" then 2 else 0) +
  (if "Has Active HELLP" or "Meets HELLP Criteria" then 5 else 0) +
  (if "Has Active Postpartum Hemorrhage" then 5 else
   if "Has Active Antepartum Hemorrhage" then 4 else 0) +
  (if "Has Placental Abruption" then 4 else 0) +
  (if "Has Severe Anemia" then 3 else 0) +
  (if "Has Severe Hypertension in Pregnancy" then 3 else 0) +
  (if "Pregnant On Teratogenic Medication" then 3 else 0) +
  (if "Has Absent Fetal Heart Tones" then 5 else 0)

define "Maternal Emergency Risk Level":
  case
    when "Maternal Emergency Risk Score" >= 8 then 'CRITICAL'
    when "Maternal Emergency Risk Score" >= 5 then 'HIGH'
    when "Maternal Emergency Risk Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// SUMMARY OUTPUTS FOR ICU INTELLIGENCE
// ============================================================

/*
 * MaternalFacts - structured output for Go consumption
 * This is what the KnowledgeSnapshotBuilder will capture
 */
define "ICU Maternal Facts":
  {
    // Pregnancy Status (CRITICAL FOR DRUG SAFETY)
    isCurrentlyPregnant: "Is Currently Pregnant",
    isFemaleOfChildbearingAge: "Is Female of Childbearing Age",
    possiblePregnancy: "Possible Pregnancy",
    gestationalAgeWeeks: "Gestational Age Weeks",
    trimester: case
      when "First Trimester" then '1st'
      when "Second Trimester" then '2nd'
      when "Third Trimester" then '3rd'
      else null
    end,

    // Hypertensive Disorders
    hasHypertensionInPregnancy: "Has Hypertension in Pregnancy",
    hasSevereHypertensionInPregnancy: "Has Severe Hypertension in Pregnancy",
    hasActivePreeclampsia: "Has Active Preeclampsia",
    meetsPreeclampsiaCriteria: "Meets Preeclampsia Criteria",
    meetsSeverePreeclampsiaCriteria: "Meets Severe Preeclampsia Criteria",
    hasActiveEclampsia: "Has Active Eclampsia",
    hasActiveHELLP: "Has Active HELLP",

    // Hemorrhage
    hasActiveAntepartumHemorrhage: "Has Active Antepartum Hemorrhage",
    hasActivePostpartumHemorrhage: "Has Active Postpartum Hemorrhage",
    hasPlacenalAbruption: "Has Placental Abruption",
    hasSevereAnemia: "Has Severe Anemia",

    // High Risk Indicators
    isHighRiskPregnancy: "Is High Risk Pregnancy",
    hasMultipleGestation: "Has Multiple Gestation",
    hasGestationalDiabetes: "Has Gestational Diabetes",
    hasPretermLabor: "Has Preterm Labor",

    // Teratogenic Drug Safety (CRITICAL)
    pregnantOnTeratogenicMedication: "Pregnant On Teratogenic Medication",
    pregnantOnACEiARBLateTerm: "Pregnant On ACEi ARB Late Term",
    onCategoryXMedication: "On Category X Medication",

    // Fetal Status
    fetalHeartRateValue: "Fetal Heart Rate Value",
    hasAbsentFetalHeartTones: "Has Absent Fetal Heart Tones",

    // Lactation
    isBreastfeeding: "Is Breastfeeding",
    isPostpartum: "Is Postpartum",

    // Risk
    maternalEmergencyRiskLevel: "Maternal Emergency Risk Level",
    maternalEmergencyRiskScore: "Maternal Emergency Risk Score"
  }

