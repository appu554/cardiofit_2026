// =============================================================================
// HFGDMTGuidelines.cql
// Tier 4b: Clinical Guidelines - Heart Failure GDMT per ACC/AHA 2022
// =============================================================================
//
// PURPOSE:
// Implements Guideline-Directed Medical Therapy (GDMT) for Heart Failure
// based on ACC/AHA/HFSA 2022 Heart Failure Guidelines.
//
// KEY COMPONENTS:
// 1. HF Staging (A, B, C, D)
// 2. LVEF Classification (HFrEF, HFmrEF, HFpEF)
// 3. GDMT Recommendations (Quadruple Therapy for HFrEF)
// 4. Target Dose Achievement
// 5. Contraindication Detection
//
// QUADRUPLE THERAPY FOR HFrEF:
// - ACEi/ARB/ARNI (preferably ARNI)
// - Beta-blocker (carvedilol, metoprolol succinate, bisoprolol)
// - MRA (spironolactone or eplerenone)
// - SGLT2i (dapagliflozin or empagliflozin)
//
// SOURCES:
// - ACC/AHA/HFSA 2022 Heart Failure Guidelines
// - ACC Expert Consensus on GDMT Optimization
// =============================================================================

library HFGDMTGuidelines version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'

// Lab and vital codes
code "LVEF": '10230-1' from "LOINC" display 'Left ventricular ejection fraction'
code "NT-proBNP": '33762-6' from "LOINC" display 'NT-proBNP'
code "BNP": '42637-9' from "LOINC" display 'BNP'
code "Potassium": '2823-3' from "LOINC" display 'Potassium'
code "Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'
code "Systolic BP": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Heart Rate": '8867-4' from "LOINC" display 'Heart rate'

// =============================================================================
// VALUESETS - Heart Failure Conditions
// =============================================================================

valueset "Heart Failure Diagnosis": 'http://vaidshala.health/ValueSet/heart-failure'
valueset "HFrEF Codes": 'http://vaidshala.health/ValueSet/hfref'
valueset "HFpEF Codes": 'http://vaidshala.health/ValueSet/hfpef'
valueset "Cardiomyopathy Codes": 'http://vaidshala.health/ValueSet/cardiomyopathy'
valueset "Prior MI Codes": 'http://vaidshala.health/ValueSet/prior-mi'
valueset "Atrial Fibrillation Codes": 'http://vaidshala.health/ValueSet/afib'

// =============================================================================
// VALUESETS - GDMT Medications
// =============================================================================

// ACEi
valueset "ACE Inhibitors": 'http://vaidshala.health/ValueSet/acei'

// ARBs
valueset "ARBs": 'http://vaidshala.health/ValueSet/arb'

// ARNI (sacubitril/valsartan)
valueset "ARNI Medications": 'http://vaidshala.health/ValueSet/arni'

// Beta-blockers (HF-specific)
valueset "HF Beta Blockers": 'http://vaidshala.health/ValueSet/hf-beta-blockers'

// MRAs
valueset "MRA Medications": 'http://vaidshala.health/ValueSet/mra'

// SGLT2i
valueset "SGLT2i Medications": 'http://vaidshala.health/ValueSet/sglt2i'

// Diuretics
valueset "Loop Diuretics": 'http://vaidshala.health/ValueSet/loop-diuretics'

// Other HF medications
valueset "Hydralazine Medications": 'http://vaidshala.health/ValueSet/hydralazine'
valueset "Nitrate Medications": 'http://vaidshala.health/ValueSet/nitrates'
valueset "Digoxin Medications": 'http://vaidshala.health/ValueSet/digoxin'
valueset "Ivabradine Medications": 'http://vaidshala.health/ValueSet/ivabradine'

// =============================================================================
// PARAMETERS
// =============================================================================

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

// LVEF Thresholds
parameter "HFrEF LVEF Threshold" Integer default 40
parameter "HFmrEF LVEF Lower" Integer default 41
parameter "HFmrEF LVEF Upper" Integer default 49
parameter "HFpEF LVEF Threshold" Integer default 50

// =============================================================================
// CONTEXT
// =============================================================================

context Patient

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

define function GetLatestObservation(obsCode Code):
  Last(
    [Observation: obsCode] O
      where O.status in { 'final', 'amended', 'corrected' }
      sort by FHIRHelpers.ToDateTime(effective as dateTime)
  )

define function GetLatestValue(obsCode Code):
  (GetLatestObservation(obsCode).value as Quantity).value

// =============================================================================
// HEART FAILURE DIAGNOSIS
// =============================================================================

define "Has Heart Failure Diagnosis":
  exists ([Condition] C where C.code in "Heart Failure Diagnosis"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Cardiomyopathy":
  exists ([Condition] C where C.code in "Cardiomyopathy Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Prior MI":
  exists ([Condition] C where C.code in "Prior MI Codes")

// =============================================================================
// LVEF CLASSIFICATION
// =============================================================================

define "Latest LVEF":
  GetLatestValue("LVEF")

define "LVEF Category":
  case
    when "Latest LVEF" is null then 'Unknown'
    when "Latest LVEF" <= "HFrEF LVEF Threshold" then 'HFrEF'  // ≤40%
    when "Latest LVEF" >= "HFmrEF LVEF Lower" and "Latest LVEF" <= "HFmrEF LVEF Upper" then 'HFmrEF'  // 41-49%
    when "Latest LVEF" >= "HFpEF LVEF Threshold" then 'HFpEF'  // ≥50%
    else 'Unknown'
  end

define "Has HFrEF":
  "LVEF Category" = 'HFrEF'

define "Has HFmrEF":
  "LVEF Category" = 'HFmrEF'

define "Has HFpEF":
  "LVEF Category" = 'HFpEF'

// =============================================================================
// HF STAGING (ACC/AHA)
// =============================================================================
// Stage A: At risk for HF
// Stage B: Pre-HF (structural heart disease, no symptoms)
// Stage C: Symptomatic HF
// Stage D: Advanced HF

valueset "HF Risk Factors": 'http://vaidshala.health/ValueSet/hf-risk-factors'
valueset "Structural Heart Disease Codes": 'http://vaidshala.health/ValueSet/structural-heart-disease'

define "Has HF Risk Factors":
  exists ([Condition] C where C.code in "HF Risk Factors")

define "Has Structural Heart Disease":
  exists ([Condition] C where C.code in "Structural Heart Disease Codes")
    or "Has Prior MI"
    or "Has Cardiomyopathy"
    or ("Latest LVEF" is not null and "Latest LVEF" < 50)

define "HF Stage":
  case
    when "Has Heart Failure Diagnosis" and "Latest LVEF" is not null and "Latest LVEF" <= 20 then 'D'  // Advanced HF (simplified)
    when "Has Heart Failure Diagnosis" then 'C'  // Symptomatic HF
    when "Has Structural Heart Disease" and not "Has Heart Failure Diagnosis" then 'B'  // Pre-HF
    when "Has HF Risk Factors" then 'A'  // At risk
    else 'None'
  end

// =============================================================================
// CURRENT GDMT STATUS
// =============================================================================

// ACEi/ARB/ARNI
define "On ACEi":
  exists ([MedicationRequest] MR where MR.medication in "ACE Inhibitors" and MR.status = 'active')

define "On ARB":
  exists ([MedicationRequest] MR where MR.medication in "ARBs" and MR.status = 'active')

define "On ARNI":
  exists ([MedicationRequest] MR where MR.medication in "ARNI Medications" and MR.status = 'active')

define "On RAASi":
  "On ACEi" or "On ARB" or "On ARNI"

// Beta-blocker
define "On Beta Blocker":
  exists ([MedicationRequest] MR where MR.medication in "HF Beta Blockers" and MR.status = 'active')

// MRA
define "On MRA":
  exists ([MedicationRequest] MR where MR.medication in "MRA Medications" and MR.status = 'active')

// SGLT2i
define "On SGLT2i":
  exists ([MedicationRequest] MR where MR.medication in "SGLT2i Medications" and MR.status = 'active')

// Loop diuretic
define "On Loop Diuretic":
  exists ([MedicationRequest] MR where MR.medication in "Loop Diuretics" and MR.status = 'active')

// Hydralazine/Nitrate
define "On Hydralazine Nitrate":
  exists ([MedicationRequest] MR where MR.medication in "Hydralazine Medications" and MR.status = 'active')
    and exists ([MedicationRequest] MR where MR.medication in "Nitrate Medications" and MR.status = 'active')

// Digoxin
define "On Digoxin":
  exists ([MedicationRequest] MR where MR.medication in "Digoxin Medications" and MR.status = 'active')

// Ivabradine
define "On Ivabradine":
  exists ([MedicationRequest] MR where MR.medication in "Ivabradine Medications" and MR.status = 'active')

// =============================================================================
// QUADRUPLE THERAPY ASSESSMENT
// =============================================================================

define "Quadruple Therapy Count":
  (if "On RAASi" then 1 else 0)
  + (if "On Beta Blocker" then 1 else 0)
  + (if "On MRA" then 1 else 0)
  + (if "On SGLT2i" then 1 else 0)

define "On Quadruple Therapy":
  "Quadruple Therapy Count" = 4

define "Quadruple Therapy Eligible":
  "Has HFrEF"

define "GDMT Gap Count":
  if "Quadruple Therapy Eligible" then
    4 - "Quadruple Therapy Count"
  else
    null

// =============================================================================
// CONTRAINDICATIONS TO GDMT COMPONENTS
// =============================================================================

define "Latest Potassium":
  GetLatestValue("Potassium")

define "Latest Creatinine":
  GetLatestValue("Creatinine")

define "Latest eGFR":
  GetLatestValue("eGFR")

define "Latest Systolic BP":
  GetLatestValue("Systolic BP")

define "Latest Heart Rate":
  GetLatestValue("Heart Rate")

// ACEi/ARB/ARNI contraindications
define "Hyperkalemia":
  "Latest Potassium" is not null and "Latest Potassium" > 5.5

define "Severe Hypotension":
  "Latest Systolic BP" is not null and "Latest Systolic BP" < 90

define "Severe CKD":
  "Latest eGFR" is not null and "Latest eGFR" < 20

valueset "Angioedema History": 'http://vaidshala.health/ValueSet/angioedema'

define "History of Angioedema":
  exists ([Condition] C where C.code in "Angioedema History")

define "RAASi Contraindication":
  "Hyperkalemia"
    or "Severe Hypotension"
    or "History of Angioedema"
    or "Latest eGFR" < 30

// Beta-blocker contraindications
define "Bradycardia":
  "Latest Heart Rate" is not null and "Latest Heart Rate" < 50

valueset "Asthma Severe Codes": 'http://vaidshala.health/ValueSet/asthma-severe'
valueset "Heart Block Codes": 'http://vaidshala.health/ValueSet/heart-block'

define "Has Severe Asthma":
  exists ([Condition] C where C.code in "Asthma Severe Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Has Heart Block":
  exists ([Condition] C where C.code in "Heart Block Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Beta Blocker Contraindication":
  "Bradycardia"
    or "Severe Hypotension"
    or "Has Severe Asthma"
    or "Has Heart Block"

// MRA contraindications
define "MRA Contraindication":
  "Hyperkalemia"
    or "Latest eGFR" < 30

// SGLT2i contraindications
valueset "Type 1 Diabetes Codes": 'http://vaidshala.health/ValueSet/t1dm'

define "Has Type 1 Diabetes":
  exists ([Condition] C where C.code in "Type 1 Diabetes Codes")

define "SGLT2i Contraindication":
  "Latest eGFR" < 20
    or "Has Type 1 Diabetes"

// =============================================================================
// GDMT RECOMMENDATIONS
// =============================================================================

define "RAASi Recommendation":
  case
    when not "Quadruple Therapy Eligible" then null
    when "RAASi Contraindication" then
      {
        medication: null,
        status: 'CONTRAINDICATED',
        reason: if "History of Angioedema" then 'History of angioedema'
                else if "Hyperkalemia" then 'Hyperkalemia (K+ > 5.5)'
                else if "Severe Hypotension" then 'Hypotension (SBP < 90)'
                else 'Severe renal impairment'
      }
    when "On ARNI" then
      {
        medication: 'ARNI',
        status: 'OPTIMAL',
        reason: 'Preferred RAASi for HFrEF'
      }
    when "On ACEi" or "On ARB" then
      {
        medication: if "On ACEi" then 'ACEi' else 'ARB',
        status: 'SUBOPTIMAL',
        reason: 'Consider transition to ARNI if tolerated',
        action: 'Switch to sacubitril/valsartan after 36h washout from ACEi'
      }
    else
      {
        medication: 'ARNI',
        status: 'NOT_ON_THERAPY',
        reason: 'Initiate ARNI or ACEi/ARB',
        action: 'Start sacubitril/valsartan 24/26mg BID or enalapril 2.5mg BID'
      }
  end

define "Beta Blocker Recommendation":
  case
    when not "Quadruple Therapy Eligible" then null
    when "Beta Blocker Contraindication" then
      {
        medication: null,
        status: 'CONTRAINDICATED',
        reason: if "Bradycardia" then 'Bradycardia (HR < 50)'
                else if "Has Heart Block" then 'Heart block'
                else if "Has Severe Asthma" then 'Severe asthma'
                else 'Hypotension'
      }
    when "On Beta Blocker" then
      {
        medication: 'HF Beta-blocker',
        status: 'ON_THERAPY',
        reason: 'Continue and uptitrate to target dose',
        targetMeds: 'carvedilol 25mg BID, metoprolol succinate 200mg daily, or bisoprolol 10mg daily'
      }
    else
      {
        medication: 'HF Beta-blocker',
        status: 'NOT_ON_THERAPY',
        reason: 'Initiate beta-blocker',
        action: 'Start carvedilol 3.125mg BID or metoprolol succinate 12.5-25mg daily'
      }
  end

define "MRA Recommendation":
  case
    when not "Quadruple Therapy Eligible" then null
    when "MRA Contraindication" then
      {
        medication: null,
        status: 'CONTRAINDICATED',
        reason: if "Hyperkalemia" then 'Hyperkalemia (K+ > 5.5)'
                else 'Severe renal impairment (eGFR < 30)'
      }
    when "On MRA" then
      {
        medication: 'MRA',
        status: 'ON_THERAPY',
        reason: 'Continue spironolactone or eplerenone',
        monitoring: 'Check K+ and creatinine within 1-2 weeks'
      }
    else
      {
        medication: 'MRA',
        status: 'NOT_ON_THERAPY',
        reason: 'Initiate MRA for NYHA II-IV with EF ≤35%',
        action: 'Start spironolactone 12.5-25mg or eplerenone 25mg daily'
      }
  end

define "SGLT2i Recommendation":
  case
    when not "Quadruple Therapy Eligible" then null
    when "SGLT2i Contraindication" then
      {
        medication: null,
        status: 'CONTRAINDICATED',
        reason: if "Has Type 1 Diabetes" then 'Type 1 Diabetes'
                else 'Severe renal impairment (eGFR < 20)'
      }
    when "On SGLT2i" then
      {
        medication: 'SGLT2i',
        status: 'ON_THERAPY',
        reason: 'Continue dapagliflozin or empagliflozin'
      }
    else
      {
        medication: 'SGLT2i',
        status: 'NOT_ON_THERAPY',
        reason: 'Initiate SGLT2i - Class I recommendation for HFrEF',
        action: 'Start dapagliflozin 10mg or empagliflozin 10mg daily'
      }
  end

// =============================================================================
// TARGET DOSE ACHIEVEMENT
// =============================================================================
// Note: Would require actual dose information from MedicationRequest

define "Target Dose Achievement":
  // Simplified - in production would calculate actual vs target dose
  {
    raasi: if "On ARNI" then 'Assess dose' else if "On ACEi" or "On ARB" then 'Consider ARNI transition' else 'Not initiated',
    betaBlocker: if "On Beta Blocker" then 'Assess dose - uptitrate if tolerated' else 'Not initiated',
    mra: if "On MRA" then 'At target if 25-50mg spironolactone' else 'Not initiated',
    sglt2i: if "On SGLT2i" then 'At target (10mg fixed dose)' else 'Not initiated'
  }

// =============================================================================
// ADDITIONAL THERAPY CONSIDERATIONS
// =============================================================================

define "Ivabradine Indicated":
  "Has HFrEF"
    and "On Beta Blocker"
    and "Latest Heart Rate" >= 70
    and "Latest LVEF" <= 35

define "Hydralazine Nitrate Indicated":
  // For African American patients with HFrEF NYHA III-IV, or ACEi/ARB intolerant
  "Has HFrEF"
    and "RAASi Contraindication"

define "Has Atrial Fibrillation":
  exists ([Condition] C where C.code in "Atrial Fibrillation Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Digoxin May Be Considered":
  "Has HFrEF"
    and "Has Atrial Fibrillation"

// =============================================================================
// NATRIURETIC PEPTIDE MONITORING
// =============================================================================

define "Latest NT-proBNP":
  GetLatestValue("NT-proBNP")

define "Latest BNP":
  GetLatestValue("BNP")

define "Elevated Natriuretic Peptide":
  ("Latest NT-proBNP" is not null and "Latest NT-proBNP" > 125)
    or ("Latest BNP" is not null and "Latest BNP" > 35)

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "HF GDMT Output":
  {
    // Diagnosis and Classification
    hasHeartFailure: "Has Heart Failure Diagnosis",
    hfStage: "HF Stage",
    lvef: "Latest LVEF",
    lvefCategory: "LVEF Category",
    hasHFrEF: "Has HFrEF",

    // Current GDMT Status
    currentTherapy: {
      onRAASi: "On RAASi",
      onARNI: "On ARNI",
      onACEi: "On ACEi",
      onARB: "On ARB",
      onBetaBlocker: "On Beta Blocker",
      onMRA: "On MRA",
      onSGLT2i: "On SGLT2i",
      onLoopDiuretic: "On Loop Diuretic"
    },

    // Quadruple Therapy
    quadrupleTherapyEligible: "Quadruple Therapy Eligible",
    quadrupleTherapyCount: "Quadruple Therapy Count",
    onQuadrupleTherapy: "On Quadruple Therapy",
    gdmtGapCount: "GDMT Gap Count",

    // Recommendations
    recommendations: {
      raasi: "RAASi Recommendation",
      betaBlocker: "Beta Blocker Recommendation",
      mra: "MRA Recommendation",
      sglt2i: "SGLT2i Recommendation"
    },

    // Additional Therapies
    ivabradineIndicated: "Ivabradine Indicated",
    hydralazineNitrateIndicated: "Hydralazine Nitrate Indicated",
    digoxinMayBeConsidered: "Digoxin May Be Considered",

    // Contraindications
    contraindications: {
      raasiContraindicated: "RAASi Contraindication",
      betaBlockerContraindicated: "Beta Blocker Contraindication",
      mraContraindicated: "MRA Contraindication",
      sglt2iContraindicated: "SGLT2i Contraindication"
    },

    // Labs
    labs: {
      potassium: "Latest Potassium",
      creatinine: "Latest Creatinine",
      egfr: "Latest eGFR",
      ntProBNP: "Latest NT-proBNP",
      elevatedNatriureticPeptide: "Elevated Natriuretic Peptide"
    },

    // Target Dose Status
    targetDoseAchievement: "Target Dose Achievement"
  }
