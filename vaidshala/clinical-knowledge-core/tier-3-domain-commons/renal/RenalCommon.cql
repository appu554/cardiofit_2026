/*
 * RenalCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Renal/Nephrology Clinical Patterns
 *
 * Purpose: Shared renal definitions consumed by:
 *   - ICU Intelligence (Tier 7) - AKI detection, electrolytes
 *   - KB-3 Guidelines (CKD management)
 *   - KB-4 Patient Safety (Nephrotoxic drugs)
 *   - KB-5 Drug Interactions (Renal dosing)
 *   - KB-8 Calculator Service (eGFR, CrCl)
 *
 * Clinical Domains:
 *   - Acute Kidney Injury (KDIGO criteria)
 *   - Chronic Kidney Disease (eGFR staging)
 *   - Electrolyte Disorders
 *   - Dialysis Status
 *   - Nephrotoxic Drug Monitoring
 *   - Contrast-Induced Nephropathy Risk
 *
 * Governance: Tier-3 review required (Clinical + Technical + Nephrology)
 * ============================================================
 */
library RenalCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// Renal Conditions
valueset "Acute Kidney Injury": 'http://cardiofit.io/fhir/ValueSet/aki-conditions'
valueset "Chronic Kidney Disease": 'http://cardiofit.io/fhir/ValueSet/ckd-conditions'
valueset "CKD Stage 3": 'http://cardiofit.io/fhir/ValueSet/ckd-stage3'
valueset "CKD Stage 4": 'http://cardiofit.io/fhir/ValueSet/ckd-stage4'
valueset "CKD Stage 5": 'http://cardiofit.io/fhir/ValueSet/ckd-stage5'
valueset "ESRD Conditions": 'http://cardiofit.io/fhir/ValueSet/esrd'
valueset "Nephrotic Syndrome": 'http://cardiofit.io/fhir/ValueSet/nephrotic-syndrome'
valueset "Glomerulonephritis": 'http://cardiofit.io/fhir/ValueSet/glomerulonephritis'
valueset "Polycystic Kidney Disease": 'http://cardiofit.io/fhir/ValueSet/pkd'

// Dialysis
valueset "Hemodialysis Procedures": 'http://cardiofit.io/fhir/ValueSet/hemodialysis'
valueset "Peritoneal Dialysis Procedures": 'http://cardiofit.io/fhir/ValueSet/peritoneal-dialysis'
valueset "CRRT Procedures": 'http://cardiofit.io/fhir/ValueSet/crrt'

// Nephrotoxic Medications
valueset "Nephrotoxic Medications": 'http://cardiofit.io/fhir/ValueSet/nephrotoxic-meds'
valueset "NSAIDs": 'http://cardiofit.io/fhir/ValueSet/nsaids'
valueset "Aminoglycosides": 'http://cardiofit.io/fhir/ValueSet/aminoglycosides'
valueset "Vancomycin": 'http://cardiofit.io/fhir/ValueSet/vancomycin'
valueset "Contrast Media": 'http://cardiofit.io/fhir/ValueSet/contrast-media'
valueset "ACE Inhibitors": 'http://cardiofit.io/fhir/ValueSet/ace-inhibitors'
valueset "ARBs": 'http://cardiofit.io/fhir/ValueSet/arbs'
valueset "Diuretics": 'http://cardiofit.io/fhir/ValueSet/diuretics'

// ============================================================
// CODESYSTEMS
// ============================================================
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

// ============================================================
// OBSERVATION CODES (LOINC)
// ============================================================

// Creatinine
code "Serum Creatinine Code": '2160-0' from "LOINC" display 'Creatinine [Mass/volume] in Serum or Plasma'
code "Urine Creatinine Code": '2161-8' from "LOINC" display 'Creatinine [Mass/volume] in Urine'

// eGFR
code "eGFR CKD-EPI Code": '62238-1' from "LOINC" display 'GFR CKD-EPI'
code "eGFR MDRD Code": '33914-3' from "LOINC" display 'GFR MDRD'
code "eGFR Code": '69405-9' from "LOINC" display 'Glomerular filtration rate'

// BUN
code "BUN Code": '3094-0' from "LOINC" display 'Urea nitrogen [Mass/volume] in Serum or Plasma'

// Urinalysis
code "Urine Protein Code": '5804-0' from "LOINC" display 'Protein [Mass/volume] in Urine'
code "UACR Code": '9318-7' from "LOINC" display 'Albumin/Creatinine [Mass ratio] in Urine'
code "Urine RBC Code": '5808-1' from "LOINC" display 'Erythrocytes [#/area] in Urine sediment'
code "Urine WBC Code": '5821-4' from "LOINC" display 'Leukocytes [#/area] in Urine sediment'

// Urine Output
code "Urine Output Code": '9192-6' from "LOINC" display 'Urine output'

// Electrolytes
code "Potassium Code": '2823-3' from "LOINC" display 'Potassium [Moles/volume] in Serum or Plasma'
code "Sodium Code": '2951-2' from "LOINC" display 'Sodium [Moles/volume] in Serum or Plasma'
code "Chloride Code": '2075-0' from "LOINC" display 'Chloride [Moles/volume] in Serum or Plasma'
code "Bicarbonate Code": '1963-8' from "LOINC" display 'Bicarbonate [Moles/volume] in Serum or Plasma'
code "Phosphorus Code": '2777-1' from "LOINC" display 'Phosphate [Mass/volume] in Serum or Plasma'
code "Calcium Code": '17861-6' from "LOINC" display 'Calcium [Mass/volume] in Serum or Plasma'
code "Magnesium Code": '19123-9' from "LOINC" display 'Magnesium [Mass/volume] in Serum or Plasma'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// CREATININE AND eGFR RETRIEVAL
// ============================================================

define "Most Recent Creatinine":
  Observations.MostRecentObservation(
    [Observation: "Serum Creatinine Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Creatinine Value":
  Observations.ObservationNumericValue("Most Recent Creatinine")

define "Most Recent Creatinine DateTime":
  Observations.ObservationDateTime("Most Recent Creatinine")

define "Most Recent eGFR":
  Observations.MostRecentObservation(
    ([Observation: "eGFR CKD-EPI Code"]
     union [Observation: "eGFR MDRD Code"]
     union [Observation: "eGFR Code"]) O
      where Observations.IsValidObservation(O)
  )

define "Most Recent eGFR Value":
  Observations.ObservationNumericValue("Most Recent eGFR")

define "Most Recent BUN":
  Observations.MostRecentObservation(
    [Observation: "BUN Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent BUN Value":
  Observations.ObservationNumericValue("Most Recent BUN")

/*
 * BUN/Creatinine Ratio (prerenal azotemia indicator)
 */
define "BUN Creatinine Ratio":
  if "Most Recent Creatinine Value" > 0 then
    "Most Recent BUN Value" / "Most Recent Creatinine Value"
  else null

// ============================================================
// CKD STAGING BY eGFR (KDIGO 2012)
// ============================================================

/*
 * CKD Stage G1: eGFR >= 90 (Normal or high)
 */
define "CKD Stage G1":
  "Most Recent eGFR Value" >= 90

/*
 * CKD Stage G2: eGFR 60-89 (Mildly decreased)
 */
define "CKD Stage G2":
  "Most Recent eGFR Value" >= 60 and "Most Recent eGFR Value" < 90

/*
 * CKD Stage G3a: eGFR 45-59 (Mild-moderately decreased)
 */
define "CKD Stage G3a":
  "Most Recent eGFR Value" >= 45 and "Most Recent eGFR Value" < 60

/*
 * CKD Stage G3b: eGFR 30-44 (Moderate-severely decreased)
 */
define "CKD Stage G3b":
  "Most Recent eGFR Value" >= 30 and "Most Recent eGFR Value" < 45

/*
 * CKD Stage G4: eGFR 15-29 (Severely decreased)
 * CRITICAL: Dialysis preparation, nephrology referral
 */
define "CKD Stage G4":
  "Most Recent eGFR Value" >= 15 and "Most Recent eGFR Value" < 30

/*
 * CKD Stage G5: eGFR < 15 (Kidney failure)
 * CRITICAL: Dialysis/transplant consideration
 */
define "CKD Stage G5":
  "Most Recent eGFR Value" < 15

/*
 * CKD Stage G3+ (Moderate-to-severe CKD)
 * Used for drug dosing adjustments
 */
define "Has CKD Stage 3 or Higher":
  "Most Recent eGFR Value" < 60

/*
 * CKD Stage G4+ (Severe CKD)
 * CRITICAL: Major drug contraindications
 */
define "Has CKD Stage 4 or Higher":
  "Most Recent eGFR Value" < 30

/*
 * CKD Stage String for Display
 */
define "CKD Stage":
  case
    when "CKD Stage G5" then 'G5'
    when "CKD Stage G4" then 'G4'
    when "CKD Stage G3b" then 'G3b'
    when "CKD Stage G3a" then 'G3a'
    when "CKD Stage G2" then 'G2'
    when "CKD Stage G1" then 'G1'
    else null
  end

// ============================================================
// ACUTE KIDNEY INJURY DETECTION (KDIGO)
// ============================================================

/*
 * Baseline Creatinine (best estimate from recent stable values)
 * Using value from 7-90 days ago as baseline
 */
define "Baseline Creatinine Candidates":
  [Observation: "Serum Creatinine Code"] O
    where Observations.IsValidObservation(O)
    and Observations.ObservationDateTime(O) >= Now() - 90 days
    and Observations.ObservationDateTime(O) <= Now() - 7 days

define "Baseline Creatinine":
  Observations.MinObservationValue("Baseline Creatinine Candidates")

/*
 * Creatinine in Last 48 Hours (for AKI staging)
 */
define "Creatinine Last 48h":
  [Observation: "Serum Creatinine Code"] O
    where Observations.IsValidObservation(O)
    and Observations.ObservationDateTime(O) >= Now() - 48 hours

define "Creatinine 48h Ago":
  Observations.FirstObservation("Creatinine Last 48h")

define "Creatinine 48h Ago Value":
  Observations.ObservationNumericValue("Creatinine 48h Ago")

/*
 * Creatinine Change from Baseline
 */
define "Creatinine Rise from Baseline":
  if "Baseline Creatinine" is not null and "Baseline Creatinine" > 0 then
    "Most Recent Creatinine Value" - "Baseline Creatinine"
  else null

define "Creatinine Ratio to Baseline":
  if "Baseline Creatinine" is not null and "Baseline Creatinine" > 0 then
    "Most Recent Creatinine Value" / "Baseline Creatinine"
  else null

/*
 * Creatinine Rise in 48 Hours
 */
define "Creatinine Rise 48h":
  if "Creatinine 48h Ago Value" is not null then
    "Most Recent Creatinine Value" - "Creatinine 48h Ago Value"
  else null

/*
 * KDIGO AKI Stage 1: Cr rise >= 0.3 mg/dL in 48h OR 1.5-1.9x baseline
 */
define "AKI Stage 1 Criteria":
  ("Creatinine Rise 48h" >= 0.3)
  or ("Creatinine Ratio to Baseline" >= 1.5 and "Creatinine Ratio to Baseline" < 2.0)

/*
 * KDIGO AKI Stage 2: Cr 2.0-2.9x baseline
 */
define "AKI Stage 2 Criteria":
  "Creatinine Ratio to Baseline" >= 2.0 and "Creatinine Ratio to Baseline" < 3.0

/*
 * KDIGO AKI Stage 3: Cr >= 3x baseline OR Cr >= 4.0 OR initiation of RRT
 * CRITICAL for ICU Intelligence
 */
define "AKI Stage 3 Criteria":
  "Creatinine Ratio to Baseline" >= 3.0
  or "Most Recent Creatinine Value" >= 4.0
  or "On CRRT"

/*
 * Has Any AKI
 */
define "Has AKI":
  "AKI Stage 1 Criteria" or "AKI Stage 2 Criteria" or "AKI Stage 3 Criteria"

/*
 * AKI Stage String
 */
define "AKI Stage":
  case
    when "AKI Stage 3 Criteria" then 'Stage 3'
    when "AKI Stage 2 Criteria" then 'Stage 2'
    when "AKI Stage 1 Criteria" then 'Stage 1'
    else null
  end

/*
 * Active AKI Diagnosis
 */
define "Has Active AKI Diagnosis":
  exists [Condition: "Acute Kidney Injury"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// URINE OUTPUT (AKI Criteria Component)
// ============================================================

define "Recent Urine Output":
  Observations.MostRecentObservation(
    [Observation: "Urine Output Code"] O
      where Observations.IsValidObservation(O)
  )

define "Recent Urine Output Value":
  Observations.ObservationNumericValue("Recent Urine Output")

/*
 * Oliguria: UO < 0.5 mL/kg/h for 6-12h (AKI Stage 1-2)
 * Note: Requires weight-adjusted calculation in KB-8
 */
define "Has Oliguria":
  "Recent Urine Output Value" < 400  // Simplified: <400 mL/day

/*
 * Anuria: UO < 100 mL/day or < 0.3 mL/kg/h for >= 24h
 * CRITICAL for ICU Intelligence
 */
define "Has Anuria":
  "Recent Urine Output Value" < 100

// ============================================================
// ELECTROLYTE MONITORING
// ============================================================

// Potassium
define "Most Recent Potassium":
  Observations.MostRecentObservation(
    [Observation: "Potassium Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Potassium Value":
  Observations.ObservationNumericValue("Most Recent Potassium")

/*
 * Potassium Classifications
 */
define "Has Hypokalemia":
  "Most Recent Potassium Value" < 3.5

define "Has Severe Hypokalemia":
  "Most Recent Potassium Value" < 3.0

define "Has Hyperkalemia":
  "Most Recent Potassium Value" > 5.0

define "Has Moderate Hyperkalemia":
  "Most Recent Potassium Value" > 5.5 and "Most Recent Potassium Value" <= 6.5

/*
 * Severe Hyperkalemia: K > 6.5 mEq/L
 * CRITICAL for ICU Intelligence - cardiac arrhythmia risk
 */
define "Has Severe Hyperkalemia":
  "Most Recent Potassium Value" > 6.5

/*
 * Critical Hyperkalemia: K > 7.0 mEq/L
 * CRITICAL - emergent treatment required
 */
define "Has Critical Hyperkalemia":
  "Most Recent Potassium Value" > 7.0

// Sodium
define "Most Recent Sodium":
  Observations.MostRecentObservation(
    [Observation: "Sodium Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Sodium Value":
  Observations.ObservationNumericValue("Most Recent Sodium")

/*
 * Sodium Classifications
 */
define "Has Hyponatremia":
  "Most Recent Sodium Value" < 135

define "Has Moderate Hyponatremia":
  "Most Recent Sodium Value" < 130

define "Has Severe Hyponatremia":
  "Most Recent Sodium Value" < 125

/*
 * Critical Hyponatremia: Na < 120 mEq/L
 * CRITICAL - seizure risk
 */
define "Has Critical Hyponatremia":
  "Most Recent Sodium Value" < 120

define "Has Hypernatremia":
  "Most Recent Sodium Value" > 145

define "Has Severe Hypernatremia":
  "Most Recent Sodium Value" > 155

// Phosphorus
define "Most Recent Phosphorus":
  Observations.MostRecentObservation(
    [Observation: "Phosphorus Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Phosphorus Value":
  Observations.ObservationNumericValue("Most Recent Phosphorus")

define "Has Hyperphosphatemia":
  "Most Recent Phosphorus Value" > 4.5

define "Has Severe Hyperphosphatemia":
  "Most Recent Phosphorus Value" > 6.5

// Calcium
define "Most Recent Calcium":
  Observations.MostRecentObservation(
    [Observation: "Calcium Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Calcium Value":
  Observations.ObservationNumericValue("Most Recent Calcium")

define "Has Hypocalcemia":
  "Most Recent Calcium Value" < 8.5

define "Has Severe Hypocalcemia":
  "Most Recent Calcium Value" < 7.0

define "Has Hypercalcemia":
  "Most Recent Calcium Value" > 10.5

define "Has Severe Hypercalcemia":
  "Most Recent Calcium Value" > 12.0

// Magnesium
define "Most Recent Magnesium":
  Observations.MostRecentObservation(
    [Observation: "Magnesium Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Magnesium Value":
  Observations.ObservationNumericValue("Most Recent Magnesium")

define "Has Hypomagnesemia":
  "Most Recent Magnesium Value" < 1.5

define "Has Severe Hypomagnesemia":
  "Most Recent Magnesium Value" < 1.0

// ============================================================
// DIALYSIS STATUS
// ============================================================

define "On Hemodialysis":
  exists [Procedure: "Hemodialysis Procedures"] P
    where P.status = 'in-progress'
    or (P.status = 'completed'
        and FHIRHelpers.ToDateTime(P.performed as FHIR.dateTime) >= Now() - 7 days)

define "On Peritoneal Dialysis":
  exists [Procedure: "Peritoneal Dialysis Procedures"] P
    where P.status = 'in-progress'
    or (P.status = 'completed'
        and FHIRHelpers.ToDateTime(P.performed as FHIR.dateTime) >= Now() - 7 days)

define "On CRRT":
  exists [Procedure: "CRRT Procedures"] P
    where P.status = 'in-progress'

define "On Any Dialysis":
  "On Hemodialysis" or "On Peritoneal Dialysis" or "On CRRT"

define "Has ESRD":
  exists [Condition: "ESRD Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// NEPHROTOXIC DRUG EXPOSURE
// ============================================================

define "On NSAIDs":
  exists [MedicationRequest: "NSAIDs"] MR
    where Medications.IsMedicationActive(MR)

define "On Aminoglycosides":
  exists [MedicationRequest: "Aminoglycosides"] MR
    where Medications.IsMedicationActive(MR)

define "On Vancomycin":
  exists [MedicationRequest: "Vancomycin"] MR
    where Medications.IsMedicationActive(MR)

define "On ACE Inhibitor":
  exists [MedicationRequest: "ACE Inhibitors"] MR
    where Medications.IsMedicationActive(MR)

define "On ARB":
  exists [MedicationRequest: "ARBs"] MR
    where Medications.IsMedicationActive(MR)

define "On Diuretics":
  exists [MedicationRequest: "Diuretics"] MR
    where Medications.IsMedicationActive(MR)

/*
 * On Any Nephrotoxic Medication
 */
define "On Nephrotoxic Medications":
  "On NSAIDs" or "On Aminoglycosides" or "On Vancomycin"

/*
 * Triple Whammy Risk (NSAID + ACEi/ARB + Diuretic)
 * CRITICAL: High AKI risk combination
 */
define "Has Triple Whammy Risk":
  "On NSAIDs"
  and ("On ACE Inhibitor" or "On ARB")
  and "On Diuretics"

/*
 * Recent Contrast Exposure (within 72h)
 * Contrast-Induced Nephropathy risk
 */
define "Recent Contrast Exposure":
  exists [MedicationAdministration: "Contrast Media"] MA
    where MA.status = 'completed'
    and FHIRHelpers.ToDateTime(MA.effective as FHIR.dateTime) >= Now() - 72 hours

// ============================================================
// ALBUMINURIA (CKD Staging A Categories)
// ============================================================

define "Most Recent UACR":
  Observations.MostRecentObservation(
    [Observation: "UACR Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent UACR Value":
  Observations.ObservationNumericValue("Most Recent UACR")

/*
 * Albuminuria Categories (KDIGO)
 */
define "Albuminuria A1":
  "Most Recent UACR Value" < 30

define "Albuminuria A2":
  "Most Recent UACR Value" >= 30 and "Most Recent UACR Value" < 300

define "Albuminuria A3":
  "Most Recent UACR Value" >= 300

define "Albuminuria Category":
  case
    when "Albuminuria A3" then 'A3'
    when "Albuminuria A2" then 'A2'
    when "Albuminuria A1" then 'A1'
    else null
  end

// ============================================================
// RENAL RISK SCORING (For ICU Intelligence)
// ============================================================

/*
 * Acute Renal Risk Score
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. ICU DECIDES actions.
 */
define "Acute Renal Risk Score":
  (if "AKI Stage 3 Criteria" then 5 else
   if "AKI Stage 2 Criteria" then 3 else
   if "AKI Stage 1 Criteria" then 2 else 0) +
  (if "Has Critical Hyperkalemia" then 5 else
   if "Has Severe Hyperkalemia" then 3 else
   if "Has Hyperkalemia" then 1 else 0) +
  (if "Has Anuria" then 4 else
   if "Has Oliguria" then 2 else 0) +
  (if "Has Critical Hyponatremia" then 3 else
   if "Has Severe Hyponatremia" then 2 else 0) +
  (if "On CRRT" then 2 else 0) +
  (if "Has Triple Whammy Risk" then 2 else
   if "On Nephrotoxic Medications" then 1 else 0) +
  (if "Recent Contrast Exposure" and "Has CKD Stage 3 or Higher" then 2 else 0)

define "Acute Renal Risk Level":
  case
    when "Acute Renal Risk Score" >= 10 then 'CRITICAL'
    when "Acute Renal Risk Score" >= 6 then 'HIGH'
    when "Acute Renal Risk Score" >= 3 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// CONTRAST-INDUCED NEPHROPATHY RISK
// ============================================================

/*
 * CIN Risk Factors
 */
define "CIN Risk Score":
  (if "Has CKD Stage 4 or Higher" then 3 else
   if "Has CKD Stage 3 or Higher" then 2 else 0) +
  (if "Has Any Diabetes" then 1 else 0) +
  (if AgeInYears() >= 75 then 1 else 0) +
  (if "Has AKI" then 2 else 0) +
  (if "On Nephrotoxic Medications" then 1 else 0)

// Helper for diabetes check (cross-reference)
define "Has Any Diabetes":
  exists [Condition] C
    where exists (C.code.coding CC
      where CC.system.value = 'http://snomed.info/sct'
      and CC.code.value in { '73211009', '44054006' })
    and C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "High CIN Risk":
  "CIN Risk Score" >= 3

// ============================================================
// DRUG DOSING FLAGS (For KB-5 / KB-1)
// ============================================================

/*
 * Requires Renal Dose Adjustment
 */
define "Requires Renal Dose Adjustment":
  "Has CKD Stage 3 or Higher" or "Has AKI" or "On Any Dialysis"

/*
 * Severe Renal Impairment (eGFR < 30)
 * Major drug contraindications
 */
define "Has Severe Renal Impairment":
  "Most Recent eGFR Value" < 30

/*
 * Avoid Nephrotoxic Drugs Flag
 */
define "Should Avoid Nephrotoxic Drugs":
  "Has AKI" or "Has CKD Stage 4 or Higher" or "On Any Dialysis"

// ============================================================
// CONDITION FLAGS
// ============================================================

define "Has Active CKD":
  exists [Condition: "Chronic Kidney Disease"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Nephrotic Syndrome":
  exists [Condition: "Nephrotic Syndrome"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// SUMMARY OUTPUTS FOR ICU INTELLIGENCE
// ============================================================

/*
 * RenalFacts - structured output for Go consumption
 * This is what the KnowledgeSnapshotBuilder will capture
 */
define "ICU Renal Facts":
  {
    // Kidney Function
    creatinineValue: "Most Recent Creatinine Value",
    eGFRValue: "Most Recent eGFR Value",
    bunValue: "Most Recent BUN Value",
    bunCreatinineRatio: "BUN Creatinine Ratio",
    ckdStage: "CKD Stage",

    // AKI
    hasAKI: "Has AKI",
    akiStage: "AKI Stage",
    creatinineRiseFromBaseline: "Creatinine Rise from Baseline",
    creatinineRatioToBaseline: "Creatinine Ratio to Baseline",

    // Urine Output
    hasOliguria: "Has Oliguria",
    hasAnuria: "Has Anuria",

    // Electrolytes
    potassiumValue: "Most Recent Potassium Value",
    hasCriticalHyperkalemia: "Has Critical Hyperkalemia",
    hasSevereHyperkalemia: "Has Severe Hyperkalemia",
    hasHyperkalemia: "Has Hyperkalemia",
    sodiumValue: "Most Recent Sodium Value",
    hasCriticalHyponatremia: "Has Critical Hyponatremia",
    hasSevereHyponatremia: "Has Severe Hyponatremia",

    // Dialysis
    onAnyDialysis: "On Any Dialysis",
    onCRRT: "On CRRT",
    hasESRD: "Has ESRD",

    // Nephrotoxicity
    onNephrotoxicMedications: "On Nephrotoxic Medications",
    hasTripleWhammyRisk: "Has Triple Whammy Risk",
    recentContrastExposure: "Recent Contrast Exposure",

    // Dosing
    requiresRenalDoseAdjustment: "Requires Renal Dose Adjustment",
    shouldAvoidNephrotoxicDrugs: "Should Avoid Nephrotoxic Drugs",

    // Risk
    acuteRenalRiskLevel: "Acute Renal Risk Level",
    acuteRenalRiskScore: "Acute Renal Risk Score",
    highCINRisk: "High CIN Risk"
  }

