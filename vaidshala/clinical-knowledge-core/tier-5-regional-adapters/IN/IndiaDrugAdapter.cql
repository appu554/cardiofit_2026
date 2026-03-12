// =============================================================================
// IndiaDrugAdapter.cql
// Tier 5: Regional Adapters - India Drug Classification
// =============================================================================
//
// PURPOSE:
// Implements India-specific drug classification based on:
// - NLEM (National List of Essential Medicines) 2022
// - Drugs and Cosmetics Act scheduling (H, H1, X, G)
// - CDSCO (Central Drugs Standard Control Organization) regulations
//
// GOVERNANCE:
// - CQL CLASSIFIES drug regulatory status
// - Does NOT make prescribing decisions
// - Provides information to support formulary and prescribing decisions
//
// SOURCES:
// - NLEM 2022 (Ministry of Health and Family Welfare)
// - Drugs and Cosmetics Rules 1945 (as amended)
// - DPCO (Drug Price Control Order) 2013
// =============================================================================

library IndiaDrugAdapter version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "ATC": 'http://www.whocc.no/atc'

// =============================================================================
// VALUESETS - NLEM Categories
// =============================================================================
// Note: These would be populated from KB-7 Terminology Service in production

// Primary NLEM medications (P category)
valueset "NLEM Primary Care Medications": 'http://vaidshala.health/ValueSet/nlem-primary-2022'

// Secondary NLEM medications (S category)
valueset "NLEM Secondary Care Medications": 'http://vaidshala.health/ValueSet/nlem-secondary-2022'

// Tertiary NLEM medications (T category)
valueset "NLEM Tertiary Care Medications": 'http://vaidshala.health/ValueSet/nlem-tertiary-2022'

// Schedule H drugs (prescription only)
valueset "Schedule H Drugs India": 'http://vaidshala.health/ValueSet/schedule-h-india'

// Schedule H1 drugs (special restrictions)
valueset "Schedule H1 Drugs India": 'http://vaidshala.health/ValueSet/schedule-h1-india'

// Schedule X drugs (narcotics/controlled)
valueset "Schedule X Drugs India": 'http://vaidshala.health/ValueSet/schedule-x-india'

// Schedule G drugs (restricted to hospitals)
valueset "Schedule G Drugs India": 'http://vaidshala.health/ValueSet/schedule-g-india'

// OTC medications (no prescription required)
valueset "OTC Medications India": 'http://vaidshala.health/ValueSet/otc-india'

// DPCO controlled prices
valueset "DPCO Price Controlled Drugs": 'http://vaidshala.health/ValueSet/dpco-controlled'

// Banned/restricted combinations
valueset "Banned Fixed Dose Combinations India": 'http://vaidshala.health/ValueSet/banned-fdc-india'

// =============================================================================
// CONTEXT
// =============================================================================

context Patient

// =============================================================================
// ACTIVE MEDICATIONS
// =============================================================================

define "Active Medication Requests":
  [MedicationRequest] MR
    where MR.status in { 'active', 'on-hold' }
      and MR.intent in { 'order', 'original-order', 'reflex-order', 'filler-order', 'instance-order' }

define "Active Medication Codes":
  "Active Medication Requests" MR
    return FHIRHelpers.ToCode(MR.medication.coding[0])

// =============================================================================
// NLEM CLASSIFICATION
// =============================================================================

define "Is NLEM Drug":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "NLEM Primary Care Medications"
         or MR.medication in "NLEM Secondary Care Medications"
         or MR.medication in "NLEM Tertiary Care Medications"
  )

define "NLEM Primary Medications":
  "Active Medication Requests" MR
    where MR.medication in "NLEM Primary Care Medications"

define "NLEM Secondary Medications":
  "Active Medication Requests" MR
    where MR.medication in "NLEM Secondary Care Medications"

define "NLEM Tertiary Medications":
  "Active Medication Requests" MR
    where MR.medication in "NLEM Tertiary Care Medications"

define "Non NLEM Medications":
  "Active Medication Requests" MR
    where not (MR.medication in "NLEM Primary Care Medications")
      and not (MR.medication in "NLEM Secondary Care Medications")
      and not (MR.medication in "NLEM Tertiary Care Medications")

define "NLEM Coverage Percentage":
  if Count("Active Medication Requests") > 0 then
    (Count("NLEM Primary Medications") + Count("NLEM Secondary Medications") + Count("NLEM Tertiary Medications"))
    / Count("Active Medication Requests") * 100
  else
    null

define "NLEM Category Summary":
  {
    primary: Count("NLEM Primary Medications"),
    secondary: Count("NLEM Secondary Medications"),
    tertiary: Count("NLEM Tertiary Medications"),
    nonNLEM: Count("Non NLEM Medications"),
    coveragePercent: "NLEM Coverage Percentage"
  }

// =============================================================================
// DRUG SCHEDULING (Drugs and Cosmetics Act)
// =============================================================================

define "Is Scheduled Drug India":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Schedule H Drugs India"
         or MR.medication in "Schedule H1 Drugs India"
         or MR.medication in "Schedule X Drugs India"
         or MR.medication in "Schedule G Drugs India"
  )

define "Schedule H Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule H Drugs India"

define "Schedule H1 Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule H1 Drugs India"

define "Schedule X Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule X Drugs India"

define "Schedule G Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule G Drugs India"

define "OTC Medications":
  "Active Medication Requests" MR
    where MR.medication in "OTC Medications India"

define "Requires Prescription India":
  exists("Schedule H Medications")
    or exists("Schedule H1 Medications")
    or exists("Schedule X Medications")
    or exists("Schedule G Medications")

// =============================================================================
// SPECIAL RESTRICTIONS
// =============================================================================

// Schedule H1 requires special documentation
define "Has H1 Restricted Medications":
  exists("Schedule H1 Medications")

define "H1 Documentation Required":
  if "Has H1 Restricted Medications" then
    {
      requiresPatientRecord: true,
      requiresDispenseRecord: true,
      retentionPeriodYears: 3,
      antibioticStewardship: true
    }
  else
    null

// Schedule X requires narcotic license
define "Has Controlled Substances India":
  exists("Schedule X Medications")

define "Narcotic Control Requirements":
  if "Has Controlled Substances India" then
    {
      requiresNDPSLicense: true,
      requiresSpecialPrescription: true,
      quantityLimitDays: 7,
      noRefillsAllowed: true,
      requiresWitnessing: true
    }
  else
    null

// Schedule G restricted to hospitals
define "Has Hospital Restricted Medications":
  exists("Schedule G Medications")

// =============================================================================
// PRICE CONTROL (DPCO 2013)
// =============================================================================

define "DPCO Controlled Medications":
  "Active Medication Requests" MR
    where MR.medication in "DPCO Price Controlled Drugs"

define "Has Price Controlled Medications":
  exists("DPCO Controlled Medications")

// =============================================================================
// BANNED COMBINATIONS
// =============================================================================

define "Banned FDC Medications":
  "Active Medication Requests" MR
    where MR.medication in "Banned Fixed Dose Combinations India"

define "Has Banned Combinations":
  exists("Banned FDC Medications")

define "Banned Combination Alert":
  if "Has Banned Combinations" then
    {
      alert: 'REGULATORY_ALERT',
      severity: 'CRITICAL',
      message: 'Patient has medication(s) containing banned fixed-dose combinations per CDSCO notification',
      action: 'Immediate review and substitution required',
      regulatoryReference: 'CDSCO FDC Ban Notification'
    }
  else
    null

// =============================================================================
// AVAILABILITY CHECK
// =============================================================================
// Note: In production, this would query real-time availability from distribution systems

define "Available in India":
  true  // Placeholder - all medications in formulary assumed available

define "Jan Aushadhi Equivalent Available":
  // Jan Aushadhi: Government generic pharmacy scheme
  exists("NLEM Primary Medications") or exists("NLEM Secondary Medications")

// =============================================================================
// SUBSTITUTION RECOMMENDATIONS
// =============================================================================

define "Generic Substitution Candidates":
  "Non NLEM Medications" MR
    where exists(
      "NLEM Primary Medications" NLEM
        // In production: would check therapeutic equivalence
    )

define "Can Substitute with NLEM":
  exists("Generic Substitution Candidates")

// =============================================================================
// AYUSH INTEGRATION
// =============================================================================
// Ayurveda, Yoga, Unani, Siddha, Homeopathy - regulated differently

valueset "AYUSH Medications": 'http://vaidshala.health/ValueSet/ayush-medications'

define "AYUSH Medications Active":
  "Active Medication Requests" MR
    where MR.medication in "AYUSH Medications"

define "Has AYUSH Medications":
  exists("AYUSH Medications Active")

define "AYUSH Regulatory Note":
  if "Has AYUSH Medications" then
    {
      regulatoryBody: 'Ministry of AYUSH',
      notAllopathic: true,
      interactionCheckRequired: true,
      separateFormularyApplies: true
    }
  else
    null

// =============================================================================
// ANTIBIOTIC STEWARDSHIP (H1 REQUIREMENT)
// =============================================================================
// Schedule H1 includes antibiotics with stewardship requirements

define "H1 Antibiotics":
  "Schedule H1 Medications" MR
    // Additional antibiotic-specific filtering would be done here

define "Antibiotic Stewardship Applicable":
  exists("H1 Antibiotics")

define "Antibiotic Stewardship Requirements":
  if "Antibiotic Stewardship Applicable" then
    {
      requiresSensitivityTesting: true,
      requiresCultureBefore: true,
      maxDurationWithoutReview: 5,  // days
      documentationRequired: 'Indication and culture results',
      escalationIfNoImprovement: true
    }
  else
    null

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "India Drug Adapter Output":
  {
    region: 'IN',
    regulatoryFramework: 'Drugs and Cosmetics Act 1940',

    // NLEM Status
    hasNLEMDrugs: "Is NLEM Drug",
    nlemCategorySummary: "NLEM Category Summary",
    nlemCoveragePercent: "NLEM Coverage Percentage",

    // Scheduling
    isScheduledDrug: "Is Scheduled Drug India",
    requiresPrescription: "Requires Prescription India",
    scheduleHCount: Count("Schedule H Medications"),
    scheduleH1Count: Count("Schedule H1 Medications"),
    scheduleXCount: Count("Schedule X Medications"),
    scheduleGCount: Count("Schedule G Medications"),
    otcCount: Count("OTC Medications"),

    // Special Requirements
    hasH1Restrictions: "Has H1 Restricted Medications",
    h1Documentation: "H1 Documentation Required",
    hasControlledSubstances: "Has Controlled Substances India",
    narcoticRequirements: "Narcotic Control Requirements",
    hasHospitalRestricted: "Has Hospital Restricted Medications",

    // Price Control
    hasPriceControlled: "Has Price Controlled Medications",
    priceControlledCount: Count("DPCO Controlled Medications"),

    // Safety Alerts
    hasBannedCombinations: "Has Banned Combinations",
    bannedCombinationAlert: "Banned Combination Alert",

    // Availability
    availableInIndia: "Available in India",
    janAushadhiAvailable: "Jan Aushadhi Equivalent Available",
    canSubstituteWithNLEM: "Can Substitute with NLEM",

    // AYUSH
    hasAYUSHMedications: "Has AYUSH Medications",
    ayushNote: "AYUSH Regulatory Note",

    // Stewardship
    antibioticStewardshipApplicable: "Antibiotic Stewardship Applicable",
    antibioticStewardshipRequirements: "Antibiotic Stewardship Requirements"
  }
