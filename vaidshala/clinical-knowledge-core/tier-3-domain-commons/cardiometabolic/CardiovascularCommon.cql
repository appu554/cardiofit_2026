/*
 * CardiovascularCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Cardiovascular Clinical Patterns
 *
 * Purpose: Shared cardiovascular definitions consumed by:
 *   - ICU Intelligence (Tier 7)
 *   - KB-3 Guidelines (Cardiology)
 *   - KB-4 Patient Safety (Anticoagulation)
 *   - CMS Cardiovascular Measures (CMS347, etc.)
 *
 * Clinical Domains:
 *   - Blood Pressure (HTN, Hypotension, Crisis)
 *   - Heart Rhythm (AFib, Brady, Tachy)
 *   - Cardiac Biomarkers (Troponin, BNP, NT-proBNP)
 *   - Lipid Panel (LDL, HDL, TG, TC)
 *   - Acute Coronary Syndrome (ACS, STEMI, NSTEMI)
 *   - Heart Failure (HFrEF, HFpEF, HFmrEF)
 *   - Risk Scoring (ASCVD, CHADS2-VASc components)
 *
 * Governance: Tier-3 review required (Clinical + Technical + Cardiology)
 * ============================================================
 */
library CardiovascularCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// Cardiovascular Conditions
valueset "Hypertension Conditions": 'http://cardiofit.io/fhir/ValueSet/hypertension'
valueset "Atrial Fibrillation Conditions": 'http://cardiofit.io/fhir/ValueSet/atrial-fibrillation'
valueset "Heart Failure Conditions": 'http://cardiofit.io/fhir/ValueSet/heart-failure'
valueset "Coronary Artery Disease Conditions": 'http://cardiofit.io/fhir/ValueSet/cad'
valueset "Acute Coronary Syndrome Conditions": 'http://cardiofit.io/fhir/ValueSet/acs'
valueset "STEMI Conditions": 'http://cardiofit.io/fhir/ValueSet/stemi'
valueset "NSTEMI Conditions": 'http://cardiofit.io/fhir/ValueSet/nstemi'
valueset "Peripheral Vascular Disease": 'http://cardiofit.io/fhir/ValueSet/pvd'
valueset "Stroke Conditions": 'http://cardiofit.io/fhir/ValueSet/stroke'
valueset "TIA Conditions": 'http://cardiofit.io/fhir/ValueSet/tia'
valueset "Thromboembolism Conditions": 'http://cardiofit.io/fhir/ValueSet/thromboembolism'
valueset "Deep Vein Thrombosis": 'http://cardiofit.io/fhir/ValueSet/dvt'
valueset "Pulmonary Embolism": 'http://cardiofit.io/fhir/ValueSet/pe'

// Cardiovascular Procedures
valueset "PCI Procedures": 'http://cardiofit.io/fhir/ValueSet/pci-procedures'
valueset "CABG Procedures": 'http://cardiofit.io/fhir/ValueSet/cabg-procedures'
valueset "Cardiac Catheterization": 'http://cardiofit.io/fhir/ValueSet/cardiac-cath'
valueset "Valve Replacement Procedures": 'http://cardiofit.io/fhir/ValueSet/valve-replacement'

// Cardiovascular Medications
valueset "Anticoagulants": 'http://cardiofit.io/fhir/ValueSet/anticoagulants'
valueset "Antiplatelets": 'http://cardiofit.io/fhir/ValueSet/antiplatelets'
valueset "Beta Blockers": 'http://cardiofit.io/fhir/ValueSet/beta-blockers'
valueset "ACE Inhibitors": 'http://cardiofit.io/fhir/ValueSet/ace-inhibitors'
valueset "ARBs": 'http://cardiofit.io/fhir/ValueSet/arbs'
valueset "Calcium Channel Blockers": 'http://cardiofit.io/fhir/ValueSet/ccbs'
valueset "Diuretics": 'http://cardiofit.io/fhir/ValueSet/diuretics'
valueset "Statins": 'http://cardiofit.io/fhir/ValueSet/statins'
valueset "Antiarrhythmics": 'http://cardiofit.io/fhir/ValueSet/antiarrhythmics'
valueset "Vasopressors": 'http://cardiofit.io/fhir/ValueSet/vasopressors'
valueset "Inotropes": 'http://cardiofit.io/fhir/ValueSet/inotropes'
valueset "SGLT2 Inhibitors": 'http://cardiofit.io/fhir/ValueSet/sglt2-inhibitors'

// ============================================================
// CODESYSTEMS
// ============================================================
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

// ============================================================
// OBSERVATION CODES (LOINC)
// ============================================================

// Blood Pressure
code "Systolic BP Code": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP Code": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "BP Panel Code": '85354-9' from "LOINC" display 'Blood pressure panel'
code "MAP Code": '8478-0' from "LOINC" display 'Mean arterial pressure'

// Heart Rate
code "Heart Rate Code": '8867-4' from "LOINC" display 'Heart rate'

// Cardiac Biomarkers
code "Troponin I Code": '10839-9' from "LOINC" display 'Troponin I.cardiac'
code "Troponin T Code": '6598-7' from "LOINC" display 'Troponin T.cardiac'
code "High Sensitivity Troponin I": '89579-7' from "LOINC" display 'Troponin I.cardiac HS'
code "High Sensitivity Troponin T": '67151-1' from "LOINC" display 'Troponin T.cardiac HS'
code "BNP Code": '30934-4' from "LOINC" display 'BNP'
code "NT-proBNP Code": '33762-6' from "LOINC" display 'NT-proBNP'
code "CK-MB Code": '13969-1' from "LOINC" display 'Creatine kinase.MB'

// Lipid Panel
code "Total Cholesterol Code": '2093-3' from "LOINC" display 'Cholesterol'
code "LDL Code": '13457-7' from "LOINC" display 'Cholesterol in LDL'
code "HDL Code": '2085-9' from "LOINC" display 'Cholesterol in HDL'
code "Triglycerides Code": '2571-8' from "LOINC" display 'Triglyceride'
code "Non-HDL Cholesterol Code": '43396-1' from "LOINC" display 'Cholesterol non HDL'

// Cardiac Function
code "Ejection Fraction Code": '10230-1' from "LOINC" display 'Left ventricular Ejection fraction'

// Anticoagulation Monitoring
code "INR Code": '34714-6' from "LOINC" display 'INR'
code "PTT Code": '3173-2' from "LOINC" display 'Activated partial thromboplastin time'
code "Anti-Xa Code": '3218-5' from "LOINC" display 'Heparin anti-Xa'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// BLOOD PRESSURE RETRIEVAL
// ============================================================

/*
 * Get most recent Systolic BP observation
 */
define "Most Recent Systolic BP":
  Observations.MostRecentObservation(
    [Observation: "Systolic BP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Systolic BP Value":
  Observations.ObservationNumericValue("Most Recent Systolic BP")

/*
 * Get most recent Diastolic BP observation
 */
define "Most Recent Diastolic BP":
  Observations.MostRecentObservation(
    [Observation: "Diastolic BP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Diastolic BP Value":
  Observations.ObservationNumericValue("Most Recent Diastolic BP")

/*
 * Get most recent MAP
 */
define "Most Recent MAP":
  Observations.MostRecentObservation(
    [Observation: "MAP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent MAP Value":
  Observations.ObservationNumericValue("Most Recent MAP")

// ============================================================
// BLOOD PRESSURE CLASSIFICATIONS (JNC 8 / ACC/AHA 2017)
// ============================================================

/*
 * Normal BP: SBP <120 AND DBP <80
 */
define "Has Normal BP":
  "Most Recent Systolic BP Value" < 120 and "Most Recent Diastolic BP Value" < 80

/*
 * Elevated BP: SBP 120-129 AND DBP <80
 */
define "Has Elevated BP":
  "Most Recent Systolic BP Value" >= 120
  and "Most Recent Systolic BP Value" < 130
  and "Most Recent Diastolic BP Value" < 80

/*
 * Stage 1 Hypertension: SBP 130-139 OR DBP 80-89
 */
define "Has Stage 1 Hypertension":
  ("Most Recent Systolic BP Value" >= 130 and "Most Recent Systolic BP Value" < 140)
  or ("Most Recent Diastolic BP Value" >= 80 and "Most Recent Diastolic BP Value" < 90)

/*
 * Stage 2 Hypertension: SBP >= 140 OR DBP >= 90
 */
define "Has Stage 2 Hypertension":
  "Most Recent Systolic BP Value" >= 140
  or "Most Recent Diastolic BP Value" >= 90

/*
 * Hypertensive Crisis: SBP > 180 OR DBP > 120
 * CRITICAL: ICU Intelligence input
 */
define "Has Hypertensive Crisis":
  "Most Recent Systolic BP Value" > 180
  or "Most Recent Diastolic BP Value" > 120

/*
 * Hypotension: SBP < 90 OR MAP < 65
 * CRITICAL: Shock detection for ICU Intelligence
 */
define "Has Hypotension":
  "Most Recent Systolic BP Value" < 90
  or "Most Recent MAP Value" < 65

/*
 * Severe Hypotension: SBP < 70
 * CRITICAL: Cardiogenic shock indicator
 */
define "Has Severe Hypotension":
  "Most Recent Systolic BP Value" < 70

// ============================================================
// HEART RATE AND RHYTHM
// ============================================================

define "Most Recent Heart Rate":
  Observations.MostRecentObservation(
    [Observation: "Heart Rate Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Heart Rate Value":
  Observations.ObservationNumericValue("Most Recent Heart Rate")

/*
 * Bradycardia: HR < 60 bpm
 */
define "Has Bradycardia":
  "Most Recent Heart Rate Value" < 60

/*
 * Significant Bradycardia: HR < 50 bpm (pacer consideration)
 */
define "Has Significant Bradycardia":
  "Most Recent Heart Rate Value" < 50

/*
 * Tachycardia: HR > 100 bpm
 */
define "Has Tachycardia":
  "Most Recent Heart Rate Value" > 100

/*
 * Significant Tachycardia: HR > 120 bpm
 */
define "Has Significant Tachycardia":
  "Most Recent Heart Rate Value" > 120

/*
 * Extreme Tachycardia: HR > 150 bpm
 * CRITICAL: ICU Intelligence input for hemodynamic instability
 */
define "Has Extreme Tachycardia":
  "Most Recent Heart Rate Value" > 150

// ============================================================
// ATRIAL FIBRILLATION STATUS
// ============================================================

/*
 * Active Atrial Fibrillation diagnosis
 */
define "Has Active AFib":
  exists [Condition: "Atrial Fibrillation Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * History of AFib (active or resolved)
 */
define "Has History of AFib":
  exists [Condition: "Atrial Fibrillation Conditions"]

/*
 * AFib with RVR: AFib + HR > 100
 */
define "Has AFib with RVR":
  "Has Active AFib" and "Has Tachycardia"

// ============================================================
// CARDIAC BIOMARKERS
// ============================================================

/*
 * Most Recent Troponin (High-sensitivity preferred)
 */
define "Most Recent Troponin":
  Observations.MostRecentObservation(
    ([Observation: "High Sensitivity Troponin I"]
     union [Observation: "High Sensitivity Troponin T"]
     union [Observation: "Troponin I Code"]
     union [Observation: "Troponin T Code"]) O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Troponin Value":
  Observations.ObservationNumericValue("Most Recent Troponin")

/*
 * Elevated Troponin (>99th percentile URL)
 * Note: Threshold varies by assay, using common hs-cTnI cutoff
 */
define "Has Elevated Troponin":
  "Most Recent Troponin Value" > 0.04

/*
 * Significantly Elevated Troponin (ACS consideration)
 */
define "Has Significantly Elevated Troponin":
  "Most Recent Troponin Value" > 0.10

/*
 * Most Recent BNP/NT-proBNP
 */
define "Most Recent BNP":
  Observations.MostRecentObservation(
    [Observation: "BNP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent BNP Value":
  Observations.ObservationNumericValue("Most Recent BNP")

define "Most Recent NT-proBNP":
  Observations.MostRecentObservation(
    [Observation: "NT-proBNP Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent NT-proBNP Value":
  Observations.ObservationNumericValue("Most Recent NT-proBNP")

/*
 * Elevated BNP (>100 pg/mL) - HF consideration
 */
define "Has Elevated BNP":
  "Most Recent BNP Value" > 100

/*
 * Elevated NT-proBNP (>300 pg/mL) - HF consideration
 */
define "Has Elevated NT-proBNP":
  "Most Recent NT-proBNP Value" > 300

/*
 * Significantly Elevated BNP (>400 pg/mL) - HF likely
 */
define "Has Significantly Elevated BNP":
  "Most Recent BNP Value" > 400

// ============================================================
// LIPID PANEL
// ============================================================

define "Most Recent LDL":
  Observations.MostRecentObservation(
    [Observation: "LDL Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent LDL Value":
  Observations.ObservationNumericValue("Most Recent LDL")

define "Most Recent HDL":
  Observations.MostRecentObservation(
    [Observation: "HDL Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent HDL Value":
  Observations.ObservationNumericValue("Most Recent HDL")

define "Most Recent Total Cholesterol":
  Observations.MostRecentObservation(
    [Observation: "Total Cholesterol Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Total Cholesterol Value":
  Observations.ObservationNumericValue("Most Recent Total Cholesterol")

define "Most Recent Triglycerides":
  Observations.MostRecentObservation(
    [Observation: "Triglycerides Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Triglycerides Value":
  Observations.ObservationNumericValue("Most Recent Triglycerides")

/*
 * LDL Classifications
 */
define "Has Optimal LDL":
  "Most Recent LDL Value" < 100

define "Has Near Optimal LDL":
  "Most Recent LDL Value" >= 100 and "Most Recent LDL Value" < 130

define "Has Borderline High LDL":
  "Most Recent LDL Value" >= 130 and "Most Recent LDL Value" < 160

define "Has High LDL":
  "Most Recent LDL Value" >= 160 and "Most Recent LDL Value" < 190

define "Has Very High LDL":
  "Most Recent LDL Value" >= 190

/*
 * HDL Classifications
 */
define "Has Low HDL":
  "Most Recent HDL Value" < 40

define "Has Protective HDL":
  "Most Recent HDL Value" >= 60

/*
 * Triglyceride Classifications
 */
define "Has High Triglycerides":
  "Most Recent Triglycerides Value" >= 200

define "Has Very High Triglycerides":
  "Most Recent Triglycerides Value" >= 500

// ============================================================
// EJECTION FRACTION / HEART FAILURE CLASSIFICATION
// ============================================================

define "Most Recent Ejection Fraction":
  Observations.MostRecentObservation(
    [Observation: "Ejection Fraction Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent EF Value":
  Observations.ObservationNumericValue("Most Recent Ejection Fraction")

/*
 * Heart Failure with Reduced EF (HFrEF): EF <= 40%
 */
define "Has HFrEF":
  "Most Recent EF Value" <= 40

/*
 * Heart Failure with Mildly Reduced EF (HFmrEF): EF 41-49%
 */
define "Has HFmrEF":
  "Most Recent EF Value" > 40 and "Most Recent EF Value" < 50

/*
 * Heart Failure with Preserved EF (HFpEF): EF >= 50%
 */
define "Has HFpEF":
  "Most Recent EF Value" >= 50

/*
 * Severely Reduced EF: EF <= 30%
 * CRITICAL: ICD consideration, advanced HF
 */
define "Has Severely Reduced EF":
  "Most Recent EF Value" <= 30

/*
 * Active Heart Failure diagnosis
 */
define "Has Active Heart Failure":
  exists [Condition: "Heart Failure Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// ANTICOAGULATION STATUS
// ============================================================

define "Most Recent INR":
  Observations.MostRecentObservation(
    [Observation: "INR Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent INR Value":
  Observations.ObservationNumericValue("Most Recent INR")

/*
 * On Anticoagulation Therapy
 */
define "On Anticoagulation":
  exists [MedicationRequest: "Anticoagulants"] MR
    where Medications.IsMedicationActive(MR)

/*
 * Therapeutic INR (2.0 - 3.0 for most indications)
 */
define "Has Therapeutic INR":
  "Most Recent INR Value" >= 2.0 and "Most Recent INR Value" <= 3.0

/*
 * Supratherapeutic INR (>3.0) - Bleeding risk
 */
define "Has Supratherapeutic INR":
  "Most Recent INR Value" > 3.0

/*
 * High INR (>4.0) - Significant bleeding risk
 * CRITICAL: ICU Intelligence / KB-4 input
 */
define "Has High INR":
  "Most Recent INR Value" > 4.0

/*
 * Critical INR (>5.0) - Reversal consideration
 * CRITICAL: ICU Intelligence input
 */
define "Has Critical INR":
  "Most Recent INR Value" > 5.0

// ============================================================
// CARDIOVASCULAR CONDITION FLAGS
// ============================================================

define "Has Active Hypertension":
  exists [Condition: "Hypertension Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has CAD":
  exists [Condition: "Coronary Artery Disease Conditions"]

define "Has History of ACS":
  exists [Condition: "Acute Coronary Syndrome Conditions"]

define "Has History of STEMI":
  exists [Condition: "STEMI Conditions"]

define "Has History of NSTEMI":
  exists [Condition: "NSTEMI Conditions"]

define "Has History of Stroke":
  exists [Condition: "Stroke Conditions"]

define "Has History of TIA":
  exists [Condition: "TIA Conditions"]

define "Has PVD":
  exists [Condition: "Peripheral Vascular Disease"]

define "Has History of VTE":
  exists ([Condition: "Deep Vein Thrombosis"] union [Condition: "Pulmonary Embolism"])

define "Has History of PE":
  exists [Condition: "Pulmonary Embolism"]

define "Has History of DVT":
  exists [Condition: "Deep Vein Thrombosis"]

// ============================================================
// CARDIAC PROCEDURE FLAGS
// ============================================================

define "Has History of PCI":
  exists [Procedure: "PCI Procedures"]

define "Has History of CABG":
  exists [Procedure: "CABG Procedures"]

define "Has Prosthetic Valve":
  exists [Procedure: "Valve Replacement Procedures"]

// ============================================================
// CARDIOVASCULAR MEDICATION STATUS
// ============================================================

define "On Beta Blocker":
  exists [MedicationRequest: "Beta Blockers"] MR
    where Medications.IsMedicationActive(MR)

define "On ACE Inhibitor":
  exists [MedicationRequest: "ACE Inhibitors"] MR
    where Medications.IsMedicationActive(MR)

define "On ARB":
  exists [MedicationRequest: "ARBs"] MR
    where Medications.IsMedicationActive(MR)

define "On ACEI or ARB":
  "On ACE Inhibitor" or "On ARB"

define "On Statin":
  exists [MedicationRequest: "Statins"] MR
    where Medications.IsMedicationActive(MR)

define "On Antiplatelet":
  exists [MedicationRequest: "Antiplatelets"] MR
    where Medications.IsMedicationActive(MR)

define "On Diuretic":
  exists [MedicationRequest: "Diuretics"] MR
    where Medications.IsMedicationActive(MR)

define "On SGLT2 Inhibitor":
  exists [MedicationRequest: "SGLT2 Inhibitors"] MR
    where Medications.IsMedicationActive(MR)

define "On Vasopressors":
  exists [MedicationRequest: "Vasopressors"] MR
    where Medications.IsMedicationActive(MR)

define "On Inotropes":
  exists [MedicationRequest: "Inotropes"] MR
    where Medications.IsMedicationActive(MR)

// ============================================================
// CHADS2-VASc COMPONENTS (For AFib Stroke Risk)
// ============================================================

/*
 * CHADS2-VASc Score Components (0-9 scale)
 * NOTE: This provides individual components for ICU/KB consumption.
 * Full scoring should be done by KB-8 Calculator Service.
 */

// Age-related components
define "CHADS2-VASc Age 65-74":
  AgeInYears() >= 65 and AgeInYears() < 75

define "CHADS2-VASc Age 75 Plus":
  AgeInYears() >= 75

// Condition components
define "CHADS2-VASc Has CHF":
  "Has Active Heart Failure" or "Has HFrEF"

define "CHADS2-VASc Has Hypertension":
  "Has Active Hypertension"

define "CHADS2-VASc Has Diabetes":
  exists [Condition] C
    where exists (C.code.coding CC where CC.system.value = 'http://snomed.info/sct'
      and CC.code.value in { '73211009', '44054006' }) // Diabetes mellitus codes
    and C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "CHADS2-VASc Has Stroke TIA":
  "Has History of Stroke" or "Has History of TIA"

define "CHADS2-VASc Has Vascular Disease":
  "Has CAD" or "Has PVD" or "Has History of ACS"

define "CHADS2-VASc Is Female":
  Patient.gender = 'female'

// ============================================================
// CARDIOVASCULAR RISK INDICATORS (For ICU Intelligence)
// ============================================================

/*
 * Acute Cardiac Risk Score (Composite)
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. ICU DECIDES actions.
 */
define "Acute Cardiac Risk Score":
  (if "Has Elevated Troponin" then 3 else 0) +
  (if "Has Significantly Elevated Troponin" then 2 else 0) +
  (if "Has Hypotension" then 3 else 0) +
  (if "Has Hypertensive Crisis" then 2 else 0) +
  (if "Has AFib with RVR" then 2 else 0) +
  (if "Has Extreme Tachycardia" then 2 else 0) +
  (if "Has Critical INR" then 3 else 0) +
  (if "Has Significantly Elevated BNP" then 2 else 0) +
  (if "Has Severely Reduced EF" then 2 else 0)

define "Acute Cardiac Risk Level":
  case
    when "Acute Cardiac Risk Score" >= 8 then 'CRITICAL'
    when "Acute Cardiac Risk Score" >= 5 then 'HIGH'
    when "Acute Cardiac Risk Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// GUIDELINE-DIRECTED THERAPY FLAGS
// ============================================================

/*
 * Heart Failure Guideline-Directed Medical Therapy (GDMT) Components
 * Used by KB-3 Guidelines for HF management
 */
define "HFrEF GDMT - On ACEI or ARB or ARNI":
  "On ACEI or ARB" // Note: ARNI valueset would need to be added

define "HFrEF GDMT - On Beta Blocker":
  "On Beta Blocker"

define "HFrEF GDMT - On MRA":
  exists [MedicationRequest] MR
    where Medications.IsMedicationActive(MR)
    and exists (MR.medication.coding C
      where C.code.value in { '6916', '152692' }) // Spironolactone, Eplerenone

define "HFrEF GDMT - On SGLT2i":
  "On SGLT2 Inhibitor"

define "HFrEF On Complete GDMT":
  "HFrEF GDMT - On ACEI or ARB or ARNI"
  and "HFrEF GDMT - On Beta Blocker"
  and "HFrEF GDMT - On MRA"
  and "HFrEF GDMT - On SGLT2i"

// ============================================================
// SUMMARY OUTPUTS FOR ICU INTELLIGENCE
// ============================================================

/*
 * CardiovascularFacts - structured output for Go consumption
 * This is what the KnowledgeSnapshotBuilder will capture
 */
define "ICU Cardiovascular Facts":
  {
    // Blood Pressure
    systolicBP: "Most Recent Systolic BP Value",
    diastolicBP: "Most Recent Diastolic BP Value",
    map: "Most Recent MAP Value",
    hasHypertensiveCrisis: "Has Hypertensive Crisis",
    hasHypotension: "Has Hypotension",
    hasSevereHypotension: "Has Severe Hypotension",

    // Heart Rate/Rhythm
    heartRate: "Most Recent Heart Rate Value",
    hasExtremeTachycardia: "Has Extreme Tachycardia",
    hasSignificantBradycardia: "Has Significant Bradycardia",
    hasActiveAFib: "Has Active AFib",
    hasAFibWithRVR: "Has AFib with RVR",

    // Biomarkers
    troponinValue: "Most Recent Troponin Value",
    hasElevatedTroponin: "Has Elevated Troponin",
    hasSignificantlyElevatedTroponin: "Has Significantly Elevated Troponin",
    bnpValue: "Most Recent BNP Value",
    hasElevatedBNP: "Has Elevated BNP",

    // Cardiac Function
    ejectionFraction: "Most Recent EF Value",
    hasHFrEF: "Has HFrEF",
    hasSeverelyReducedEF: "Has Severely Reduced EF",

    // Anticoagulation
    inrValue: "Most Recent INR Value",
    onAnticoagulation: "On Anticoagulation",
    hasCriticalINR: "Has Critical INR",

    // Risk
    acuteCardiacRiskLevel: "Acute Cardiac Risk Level",
    acuteCardiacRiskScore: "Acute Cardiac Risk Score",

    // Vasopressor/Inotrope Status
    onVasopressors: "On Vasopressors",
    onInotropes: "On Inotropes"
  }

