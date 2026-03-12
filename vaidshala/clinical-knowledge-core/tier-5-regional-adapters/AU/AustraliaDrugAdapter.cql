// =============================================================================
// AustraliaDrugAdapter.cql
// Tier 5: Regional Adapters - Australia Drug Classification
// =============================================================================
//
// PURPOSE:
// Implements Australia-specific drug classification based on:
// - PBS (Pharmaceutical Benefits Scheme) listings
// - TGA (Therapeutic Goods Administration) registration
// - Scheduling under Standard for Uniform Scheduling of Medicines (SUSMP)
//
// GOVERNANCE:
// - CQL CLASSIFIES drug regulatory status for Australia
// - Does NOT make prescribing decisions
// - Provides information to support PBS authority prescribing decisions
//
// SOURCES:
// - PBS Schedule (Services Australia)
// - TGA Australian Register of Therapeutic Goods (ARTG)
// - Poisons Standard (SUSMP) 2024
// =============================================================================

library AustraliaDrugAdapter version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "AMT": 'http://snomed.info/sct/32506021000036107'  // Australian Medicines Terminology
codesystem "PBS": 'http://pbs.gov.au'
codesystem "ATC": 'http://www.whocc.no/atc'
codesystem "SNOMED-AU": 'http://snomed.info/sct/32506021000036107'

// =============================================================================
// VALUESETS - PBS Categories
// =============================================================================
// Note: These would be populated from KB-7 Terminology Service in production

// PBS General Schedule (unrestricted)
valueset "PBS General Schedule": 'http://vaidshala.health/ValueSet/pbs-general'

// PBS Restricted Benefit (specific indications)
valueset "PBS Restricted Benefit": 'http://vaidshala.health/ValueSet/pbs-restricted'

// PBS Authority Required (phone/online approval)
valueset "PBS Authority Required": 'http://vaidshala.health/ValueSet/pbs-authority'

// PBS Authority Required (Streamlined)
valueset "PBS Authority Streamlined": 'http://vaidshala.health/ValueSet/pbs-authority-streamlined'

// PBS Section 100 (Highly Specialised Drugs)
valueset "PBS Section 100 HSD": 'http://vaidshala.health/ValueSet/pbs-section100-hsd'

// PBS Section 100 (Remote Area Aboriginal Health Services)
valueset "PBS Section 100 RAAHS": 'http://vaidshala.health/ValueSet/pbs-section100-raahs'

// PBS Efficient Funding of Chemotherapy
valueset "PBS Chemotherapy": 'http://vaidshala.health/ValueSet/pbs-chemotherapy'

// TGA Registered Products
valueset "TGA Registered": 'http://vaidshala.health/ValueSet/tga-registered'

// Schedule 4 (Prescription Only)
valueset "Schedule 4 Prescription Only": 'http://vaidshala.health/ValueSet/schedule4-australia'

// Schedule 8 (Controlled Drugs)
valueset "Schedule 8 Controlled": 'http://vaidshala.health/ValueSet/schedule8-australia'

// Schedule 2 (Pharmacy Only)
valueset "Schedule 2 Pharmacy Only": 'http://vaidshala.health/ValueSet/schedule2-australia'

// Schedule 3 (Pharmacist Only)
valueset "Schedule 3 Pharmacist Only": 'http://vaidshala.health/ValueSet/schedule3-australia'

// Unscheduled (OTC)
valueset "Unscheduled OTC": 'http://vaidshala.health/ValueSet/unscheduled-australia'

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

// =============================================================================
// PBS CLASSIFICATION
// =============================================================================

define "Is PBS Listed":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "PBS General Schedule"
         or MR.medication in "PBS Restricted Benefit"
         or MR.medication in "PBS Authority Required"
         or MR.medication in "PBS Authority Streamlined"
         or MR.medication in "PBS Section 100 HSD"
         or MR.medication in "PBS Section 100 RAAHS"
         or MR.medication in "PBS Chemotherapy"
  )

define "PBS General Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS General Schedule"

define "PBS Restricted Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Restricted Benefit"

define "PBS Authority Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Authority Required"

define "PBS Authority Streamlined Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Authority Streamlined"

define "PBS Section 100 HSD Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Section 100 HSD"

define "PBS Section 100 RAAHS Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Section 100 RAAHS"

define "PBS Chemotherapy Medications":
  "Active Medication Requests" MR
    where MR.medication in "PBS Chemotherapy"

define "Non PBS Medications":
  "Active Medication Requests" MR
    where not (MR.medication in "PBS General Schedule")
      and not (MR.medication in "PBS Restricted Benefit")
      and not (MR.medication in "PBS Authority Required")
      and not (MR.medication in "PBS Authority Streamlined")
      and not (MR.medication in "PBS Section 100 HSD")
      and not (MR.medication in "PBS Section 100 RAAHS")
      and not (MR.medication in "PBS Chemotherapy")

define "PBS Restriction Code":
  case
    when exists("PBS Section 100 HSD Medications") then 'S100_HSD'
    when exists("PBS Section 100 RAAHS Medications") then 'S100_RAAHS'
    when exists("PBS Chemotherapy Medications") then 'CHEMOTHERAPY'
    when exists("PBS Authority Medications") then 'AUTHORITY'
    when exists("PBS Authority Streamlined Medications") then 'STREAMLINED'
    when exists("PBS Restricted Medications") then 'RESTRICTED'
    when exists("PBS General Medications") then 'GENERAL'
    else 'NOT_PBS'
  end

define "PBS Category Summary":
  {
    general: Count("PBS General Medications"),
    restricted: Count("PBS Restricted Medications"),
    authority: Count("PBS Authority Medications"),
    authorityStreamlined: Count("PBS Authority Streamlined Medications"),
    section100HSD: Count("PBS Section 100 HSD Medications"),
    section100RAAHS: Count("PBS Section 100 RAAHS Medications"),
    chemotherapy: Count("PBS Chemotherapy Medications"),
    nonPBS: Count("Non PBS Medications")
  }

// =============================================================================
// AUTHORITY PRESCRIPTION REQUIREMENTS
// =============================================================================

define "Requires Authority Script":
  exists("PBS Authority Medications")

define "Requires Streamlined Authority":
  exists("PBS Authority Streamlined Medications")

define "Authority Script Requirements":
  if "Requires Authority Script" then
    {
      requiresApproval: true,
      approvalType: 'Phone or Online',
      approvalAuthority: 'Services Australia',
      pbsOnlineUrl: 'https://pbsonline.com.au',
      phoneNumber: '1800 888 333',
      documentIndication: true,
      includeApprovalNumber: true
    }
  else if "Requires Streamlined Authority" then
    {
      requiresApproval: true,
      approvalType: 'Streamlined (4-digit code on script)',
      approvalAuthority: 'Self-assessed by prescriber',
      documentIndication: true,
      includeStreamlinedCode: true
    }
  else
    null

// =============================================================================
// TGA REGISTRATION
// =============================================================================

define "Is TGA Registered":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "TGA Registered"
  )

define "TGA Registered Medications":
  "Active Medication Requests" MR
    where MR.medication in "TGA Registered"

define "Non TGA Registered Medications":
  "Active Medication Requests" MR
    where not (MR.medication in "TGA Registered")

define "Has Unregistered Medications":
  exists("Non TGA Registered Medications")

define "TGA Registration Alert":
  if "Has Unregistered Medications" then
    {
      alert: 'REGULATORY_WARNING',
      severity: 'WARNING',
      message: 'Patient has medication(s) not registered with TGA',
      action: 'Verify SAS/CTN exemption or consider alternatives',
      regulatoryNote: 'Unregistered medications may require Special Access Scheme approval'
    }
  else
    null

// =============================================================================
// SCHEDULING (SUSMP - Poisons Standard)
// =============================================================================

define "Is Schedule 8 Controlled":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Schedule 8 Controlled"
  )

define "Is Schedule 4 Prescription Only":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Schedule 4 Prescription Only"
  )

define "Is Schedule 3 Pharmacist Only":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Schedule 3 Pharmacist Only"
  )

define "Is Schedule 2 Pharmacy Only":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Schedule 2 Pharmacy Only"
  )

define "Schedule 8 Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule 8 Controlled"

define "Schedule 4 Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule 4 Prescription Only"

define "Schedule 3 Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule 3 Pharmacist Only"

define "Schedule 2 Medications":
  "Active Medication Requests" MR
    where MR.medication in "Schedule 2 Pharmacy Only"

define "OTC Medications":
  "Active Medication Requests" MR
    where MR.medication in "Unscheduled OTC"

define "Scheduling Summary":
  {
    schedule8: Count("Schedule 8 Medications"),
    schedule4: Count("Schedule 4 Medications"),
    schedule3: Count("Schedule 3 Medications"),
    schedule2: Count("Schedule 2 Medications"),
    unscheduled: Count("OTC Medications")
  }

// =============================================================================
// SCHEDULE 8 CONTROLLED SUBSTANCES
// =============================================================================

define "Has Controlled Substances Australia":
  "Is Schedule 8 Controlled"

define "Schedule 8 Requirements":
  if "Has Controlled Substances Australia" then
    {
      requiresPermit: true,  // State/Territory dependent
      prescriptionFormat: 'Duplicate or computer-generated',
      maxQuantityWeeks: 2,  // Generally 2 weeks supply without authority
      repeatLimitation: 'Varies by state - check local regulations',
      realTimeMonitoring: 'SafeScript/RTPM required in VIC, TAS, ACT, SA',
      handwritingRequired: 'Quantity, repeats must be in words and figures',
      stateVariation: true
    }
  else
    null

// State-specific Schedule 8 notes
define "Schedule 8 State Notes":
  if "Has Controlled Substances Australia" then
    {
      vic: 'SafeScript check required before prescribing',
      nsw: 'Authority required for > 2 months treatment',
      qld: 'Drug dependency treatment requires approval',
      wa: 'Medicines and Poisons Regulations apply',
      sa: 'DASSA approval may be required',
      tas: 'RTPM check mandatory',
      act: 'RTPM mandatory',
      nt: 'Chief Health Officer approval may be required'
    }
  else
    null

// =============================================================================
// SECTION 100 REQUIREMENTS
// =============================================================================

define "Has Section 100 Medications":
  exists("PBS Section 100 HSD Medications")
    or exists("PBS Section 100 RAAHS Medications")

define "Section 100 Requirements":
  case
    when exists("PBS Section 100 HSD Medications") then
      {
        program: 'Highly Specialised Drugs Program',
        requiresSpecialistInitiation: true,
        dispensingLocation: 'Hospital or approved community pharmacy',
        patientRegistration: 'Required for some medications',
        reportingRequired: true
      }
    when exists("PBS Section 100 RAAHS Medications") then
      {
        program: 'Remote Area Aboriginal Health Services',
        eligibilityCriteria: 'Patient accessing care through approved Aboriginal health service',
        bulkSupply: true,
        specialPackSizes: true
      }
    else
      null
  end

// =============================================================================
// BRAND SUBSTITUTION
// =============================================================================
// Note: In Australia, pharmacists can substitute brands unless prescriber ticks 'a' or 'brand substitution not permitted'

define "Brand Substitution Permitted":
  true  // Default - unless explicitly marked otherwise on script

define "Price Disclosure Brands":
  // Medications affected by price disclosure - may have equivalent generics
  "PBS General Medications"

// =============================================================================
// CTG (CLOSING THE GAP) CO-PAYMENT
// =============================================================================
// Aboriginal and Torres Strait Islander patients may access PBS medicines with reduced co-payment

define "Is Indigenous Australian":
  exists (
    Patient.extension E
      where E.url = 'http://hl7.org.au/fhir/StructureDefinition/indigenous-status'
        and (E.value as CodeableConcept).coding[0].code in { '1', '2', '3' }
  )

define "CTG Eligible":
  "Is Indigenous Australian" and "Is PBS Listed"

define "CTG Benefits":
  if "CTG Eligible" then
    {
      eligibleForCTG: true,
      reducedCopayment: true,
      copaymentAmount: '$0.00 for concessional, reduced for general',
      registrationRequired: 'Patient must be registered for CTG',
      prescriptionEndorsement: 'CTG must be marked on prescription'
    }
  else
    null

// =============================================================================
// SAFETY NET
// =============================================================================
// PBS Safety Net - reduced costs after threshold reached

define "Safety Net Applicable":
  "Is PBS Listed"

define "Safety Net Info":
  if "Safety Net Applicable" then
    {
      generalThreshold: '$1617.10 (2024)',
      concessionalThreshold: '$262.80 (2024)',
      afterThreshold: 'Reduced co-payment or free medicines',
      trackingMethod: 'Prescription Record Form or SafeScript'
    }
  else
    null

// =============================================================================
// CHEMOTHERAPY MEDICATIONS
// =============================================================================

define "Has Chemotherapy Medications":
  exists("PBS Chemotherapy Medications")

define "Chemotherapy Requirements":
  if "Has Chemotherapy Medications" then
    {
      program: 'Efficient Funding of Chemotherapy',
      prescribingRequirements: 'Oncologist/haematologist or delegated',
      dispensingLocation: 'Hospital pharmacy or approved day hospital',
      protocolDocumentation: 'Chemotherapy protocol must be documented',
      cycleDosing: 'Dose calculated per cycle',
      wasteMinimization: 'Dose banding may apply'
    }
  else
    null

// =============================================================================
// BIOSIMILAR CONSIDERATIONS
// =============================================================================

valueset "Biosimilar Medications Australia": 'http://vaidshala.health/ValueSet/biosimilars-australia'

define "Has Biosimilar Options":
  exists (
    "Active Medication Requests" MR
      where MR.medication in "Biosimilar Medications Australia"
  )

define "Biosimilar Note":
  if "Has Biosimilar Options" then
    {
      biosimilarAvailable: true,
      substitutionAllowed: false,  // Biosimilar substitution requires prescriber decision in Australia
      costSavings: 'PBS price reduction when biosimilar available',
      requiresNewPrescription: true  // Cannot substitute like generics
    }
  else
    null

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "Australia Drug Adapter Output":
  {
    region: 'AU',
    regulatoryFramework: 'TGA/PBS/SUSMP',

    // PBS Status
    isPBSListed: "Is PBS Listed",
    pbsRestrictionCode: "PBS Restriction Code",
    pbsCategorySummary: "PBS Category Summary",
    nonPBSCount: Count("Non PBS Medications"),

    // Authority Requirements
    requiresAuthorityScript: "Requires Authority Script",
    requiresStreamlinedAuthority: "Requires Streamlined Authority",
    authorityRequirements: "Authority Script Requirements",

    // TGA Registration
    isTGARegistered: "Is TGA Registered",
    hasUnregisteredMedications: "Has Unregistered Medications",
    tgaAlert: "TGA Registration Alert",

    // Scheduling
    hasControlledSubstances: "Has Controlled Substances Australia",
    isSchedule4: "Is Schedule 4 Prescription Only",
    isSchedule3: "Is Schedule 3 Pharmacist Only",
    isSchedule2: "Is Schedule 2 Pharmacy Only",
    schedulingSummary: "Scheduling Summary",

    // Schedule 8 Requirements
    schedule8Requirements: "Schedule 8 Requirements",
    schedule8StateNotes: "Schedule 8 State Notes",

    // Section 100
    hasSection100: "Has Section 100 Medications",
    section100Requirements: "Section 100 Requirements",

    // Chemotherapy
    hasChemotherapy: "Has Chemotherapy Medications",
    chemotherapyRequirements: "Chemotherapy Requirements",

    // Patient-specific
    ctgEligible: "CTG Eligible",
    ctgBenefits: "CTG Benefits",
    safetyNetInfo: "Safety Net Info",

    // Substitution
    brandSubstitutionPermitted: "Brand Substitution Permitted",
    hasBiosimilarOptions: "Has Biosimilar Options",
    biosimilarNote: "Biosimilar Note"
  }
