/*
 * CQMCommon CQL Library
 * ============================================================
 * Tier 2 Infrastructure - Clinical Quality Measure Common Logic
 *
 * Purpose: Provides the foundational infrastructure for building
 *          clinical quality measures (CQMs) following eCQM patterns.
 *          This library implements common patterns used by CMS
 *          and other regulatory quality measures.
 *
 * Version: 1.0.0
 * FHIR Version: R4 (4.0.1)
 * CQL Version: 1.5
 *
 * Dependencies:
 *   - FHIRHelpers (Tier 0)
 *   - IntervalHelpers (Tier 1)
 *   - EncounterHelpers (Tier 1)
 *   - MedicationHelpers (Tier 1)
 *   - ObservationHelpers (Tier 1)
 *   - CommonClinicalCodes (Tier 2)
 *   - ClinicalObservationHelpers (Tier 2)
 *
 * Clinical Use Cases:
 *   - eCQM (electronic Clinical Quality Measures)
 *   - CMS Quality Reporting
 *   - HEDIS Measures
 *   - Custom quality programs
 *   - Population health metrics
 *
 * Measure Types Supported:
 *   - Proportion measures
 *   - Continuous variable measures
 *   - Cohort measures
 *   - Ratio measures
 *
 * Governance: Changes require Tier-2 review (Clinical + Technical)
 * ============================================================
 */
library CQMCommon version '1.0.0'

using FHIR version '4.0.1'

// Tier 0 - FHIR Foundation
include FHIRHelpers version '4.4.000' called FHIRHelpers

// Tier 1 - Grammar Primitives (no clinical meaning)
include IntervalHelpers version '1.0.0' called IntervalHelpers
include EncounterHelpers version '1.0.0' called EncounterHelpers
include MedicationHelpers version '1.0.0' called MedicationHelpers
include ObservationHelpers version '1.0.0' called ObservationHelpers

// Tier 2 - Clinical Codes and Helpers (clinical meaning)
include CommonClinicalCodes version '1.0.0' called Codes
include ClinicalObservationHelpers version '1.0.0' called ClinicalObs

// ============================================================
// CODESYSTEMS (must be declared before context)
// ============================================================
codesystem "ConditionClinicalStatus": 'http://terminology.hl7.org/CodeSystem/condition-clinical'
codesystem "ConditionVerificationStatus": 'http://terminology.hl7.org/CodeSystem/condition-ver-status'
codesystem "ConditionCategory": 'http://terminology.hl7.org/CodeSystem/condition-category'
codesystem "AllergyIntoleranceClinicalStatus": 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical'
codesystem "AllergyIntoleranceVerificationStatus": 'http://terminology.hl7.org/CodeSystem/allergyintolerance-verification'
codesystem "ProcedureStatus": 'http://hl7.org/fhir/event-status'
codesystem "ServiceRequestStatus": 'http://hl7.org/fhir/request-status'
codesystem "DiagnosticReportStatus": 'http://hl7.org/fhir/diagnostic-report-status'
codesystem "ImmunizationStatus": 'http://hl7.org/fhir/event-status'
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "AdministrativeGender": 'http://hl7.org/fhir/administrative-gender'

// ============================================================
// CODES - Condition Clinical Status
// ============================================================
code "Condition Active": 'active' from "ConditionClinicalStatus" display 'Active'
code "Condition Recurrence": 'recurrence' from "ConditionClinicalStatus" display 'Recurrence'
code "Condition Relapse": 'relapse' from "ConditionClinicalStatus" display 'Relapse'
code "Condition Inactive": 'inactive' from "ConditionClinicalStatus" display 'Inactive'
code "Condition Remission": 'remission' from "ConditionClinicalStatus" display 'Remission'
code "Condition Resolved": 'resolved' from "ConditionClinicalStatus" display 'Resolved'

// ============================================================
// CODES - Condition Verification Status
// ============================================================
code "Condition Unconfirmed": 'unconfirmed' from "ConditionVerificationStatus" display 'Unconfirmed'
code "Condition Provisional": 'provisional' from "ConditionVerificationStatus" display 'Provisional'
code "Condition Differential": 'differential' from "ConditionVerificationStatus" display 'Differential'
code "Condition Confirmed": 'confirmed' from "ConditionVerificationStatus" display 'Confirmed'
code "Condition Refuted": 'refuted' from "ConditionVerificationStatus" display 'Refuted'
code "Condition Entered In Error": 'entered-in-error' from "ConditionVerificationStatus" display 'Entered in Error'

// ============================================================
// CODES - Procedure Status
// ============================================================
code "Procedure Completed": 'completed' from "ProcedureStatus" display 'Completed'
code "Procedure In Progress": 'in-progress' from "ProcedureStatus" display 'In Progress'
code "Procedure Not Done": 'not-done' from "ProcedureStatus" display 'Not Done'

// ============================================================
// CODES - Administrative Gender
// ============================================================
code "Male": 'male' from "AdministrativeGender" display 'Male'
code "Female": 'female' from "AdministrativeGender" display 'Female'

// ============================================================
// VALUESETS (for exclusion patterns)
// KB-7 Terminology Service (localhost:8087) - uses exact PostgreSQL name column for lookup
// ============================================================
valueset "Hospice Encounter": 'http://localhost:8087/fhir/ValueSet/Hospice Encounter'
valueset "Hospice Care Ambulatory": 'http://localhost:8087/fhir/ValueSet/Hospice Care Ambulatory'
valueset "Hospice Diagnosis": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.1165'
valueset "Palliative Care Encounter": 'http://localhost:8087/fhir/ValueSet/Palliative Care Encounter'
valueset "Palliative Care Intervention": 'http://cts.nlm.nih.gov/fhir/ValueSet/2.16.840.1.113883.3.464.1003.198.12.1135'

// ============================================================
// MEASUREMENT PERIOD PARAMETER
// ============================================================
/*
 * The Measurement Period is the time interval for quality measure calculation.
 * This parameter MUST be supplied at runtime by the CQL engine.
 *
 * Standard values:
 * - Annual measures: January 1 to December 31 of the reporting year
 * - Quarterly measures: Start and end of the fiscal quarter
 */
parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

// ============================================================
// CONTEXT
// ============================================================
context Patient

/*
 * Measurement Period as Date interval (for date-based comparisons)
 */
define "Measurement Period Date":
  Interval[date from start of "Measurement Period", date from end of "Measurement Period"]

// ============================================================
// PATIENT DEMOGRAPHICS
// ============================================================

/*
 * Patient birth date
 */
define "Patient Birth Date":
  FHIRHelpers.ToDate(Patient.birthDate)

/*
 * Patient gender
 */
define "Patient Gender":
  Patient.gender.value

/*
 * Patient is male
 */
define "Is Male":
  Patient.gender = 'male'

/*
 * Patient is female
 */
define "Is Female":
  Patient.gender = 'female'

/*
 * Age at start of measurement period
 * Standard for most quality measures
 */
define "Age at Start of Measurement Period":
  AgeInYearsAt(date from start of "Measurement Period")

/*
 * Age at end of measurement period
 */
define "Age at End of Measurement Period":
  AgeInYearsAt(date from end of "Measurement Period")

/*
 * Age in years at specific date
 */
define function AgeInYearsAt(asOfDate Date):
  if "Patient Birth Date" is not null then
    difference in years between "Patient Birth Date" and asOfDate
  else null

/*
 * Age in months at specific date
 */
define function AgeInMonthsAt(asOfDate Date):
  if "Patient Birth Date" is not null then
    difference in months between "Patient Birth Date" and asOfDate
  else null

/*
 * Age in days at specific date
 */
define function AgeInDaysAt(asOfDate Date):
  if "Patient Birth Date" is not null then
    difference in days between "Patient Birth Date" and asOfDate
  else null

// ============================================================
// AGE STRATIFICATION HELPERS
// ============================================================

/*
 * Standard adult age brackets (18-44, 45-64, 65+)
 */
define "Age 18 to 44":
  "Age at Start of Measurement Period" >= 18
  and "Age at Start of Measurement Period" <= 44

define "Age 45 to 64":
  "Age at Start of Measurement Period" >= 45
  and "Age at Start of Measurement Period" <= 64

define "Age 65 and Older":
  "Age at Start of Measurement Period" >= 65

/*
 * Pediatric age brackets
 */
define "Age 0 to 2":
  "Age at Start of Measurement Period" >= 0
  and "Age at Start of Measurement Period" <= 2

define "Age 3 to 11":
  "Age at Start of Measurement Period" >= 3
  and "Age at Start of Measurement Period" <= 11

define "Age 12 to 17":
  "Age at Start of Measurement Period" >= 12
  and "Age at Start of Measurement Period" <= 17

/*
 * Generic age range check
 */
define function AgeInRange(minAge Integer, maxAge Integer):
  "Age at Start of Measurement Period" >= minAge
  and "Age at Start of Measurement Period" <= maxAge

// ============================================================
// CONDITION HELPERS
// ============================================================

/*
 * Get condition onset as DateTime
 */
define function ConditionOnset(condition Condition):
  case
    when condition.onset is FHIR.dateTime then
      FHIRHelpers.ToDateTime(condition.onset as FHIR.dateTime)
    when condition.onset is FHIR.Period then
      FHIRHelpers.ToDateTime((condition.onset as FHIR.Period).start)
    when condition.onset is FHIR.Age then
      Patient.birthDate + FHIRHelpers.ToQuantity(condition.onset as FHIR.Age)
    else null
  end

/*
 * Get condition onset as Date
 */
define function ConditionOnsetDate(condition Condition):
  date from ConditionOnset(condition)

/*
 * Get condition abatement as DateTime
 */
define function ConditionAbatement(condition Condition):
  case
    when condition.abatement is FHIR.dateTime then
      FHIRHelpers.ToDateTime(condition.abatement as FHIR.dateTime)
    when condition.abatement is FHIR.Period then
      FHIRHelpers.ToDateTime((condition.abatement as FHIR.Period).end)
    when condition.abatement is FHIR.Age then
      Patient.birthDate + FHIRHelpers.ToQuantity(condition.abatement as FHIR.Age)
    else null
  end

/*
 * Get condition prevalence period (onset to abatement)
 */
define function ConditionPrevalencePeriod(condition Condition):
  Interval[ConditionOnset(condition), ConditionAbatement(condition)]

/*
 * Check if condition is active
 */
define function IsActiveCondition(condition Condition):
  condition.clinicalStatus ~ "Condition Active"
  and condition.verificationStatus ~ "Condition Confirmed"

/*
 * Check if condition was confirmed
 */
define function IsConfirmedCondition(condition Condition):
  condition.verificationStatus ~ "Condition Confirmed"

/*
 * Check if condition is in remission
 */
define function IsConditionInRemission(condition Condition):
  condition.clinicalStatus ~ "Condition Remission"

/*
 * Check if condition is resolved
 */
define function IsResolvedCondition(condition Condition):
  condition.clinicalStatus ~ "Condition Resolved"

/*
 * Filter for active, confirmed conditions
 */
define function ActiveConfirmedConditions(conditions List<Condition>):
  conditions C
  where IsActiveCondition(C)

/*
 * Filter conditions by code
 */
define function ConditionsByCode(conditions List<Condition>, code Code):
  conditions C where C.code ~ code

/*
 * Filter conditions by ValueSet
 */
define function ConditionsInValueSet(conditions List<Condition>, valueSet ValueSet):
  conditions C where C.code in valueSet

/*
 * Filter conditions overlapping with measurement period
 */
define function ConditionsDuringMeasurementPeriod(conditions List<Condition>):
  conditions C
  where ConditionPrevalencePeriod(C) overlaps "Measurement Period"

/*
 * Check if patient has active condition from ValueSet
 */
define function HasActiveCondition(conditions List<Condition>, valueSet ValueSet):
  exists (
    ActiveConfirmedConditions(ConditionsInValueSet(conditions, valueSet))
  )

/*
 * Check if condition was present during measurement period
 */
define function ConditionPresentDuringMeasurementPeriod(conditions List<Condition>, valueSet ValueSet):
  exists (
    ConditionsDuringMeasurementPeriod(
      ConditionsInValueSet(ActiveConfirmedConditions(conditions), valueSet)
    )
  )

// ============================================================
// PROCEDURE HELPERS
// ============================================================

/*
 * Get procedure performed DateTime
 */
define function ProcedurePerformedDateTime(procedure Procedure):
  case
    when procedure.performed is FHIR.dateTime then
      FHIRHelpers.ToDateTime(procedure.performed as FHIR.dateTime)
    when procedure.performed is FHIR.Period then
      FHIRHelpers.ToDateTime((procedure.performed as FHIR.Period).start)
    else null
  end

/*
 * Get procedure performed Date
 */
define function ProcedurePerformedDate(procedure Procedure):
  date from ProcedurePerformedDateTime(procedure)

/*
 * Get procedure performed Period
 */
define function ProcedurePerformedPeriod(procedure Procedure):
  case
    when procedure.performed is FHIR.Period then
      Interval[
        FHIRHelpers.ToDateTime((procedure.performed as FHIR.Period).start),
        FHIRHelpers.ToDateTime((procedure.performed as FHIR.Period).end)
      ]
    when procedure.performed is FHIR.dateTime then
      Interval[
        FHIRHelpers.ToDateTime(procedure.performed as FHIR.dateTime),
        FHIRHelpers.ToDateTime(procedure.performed as FHIR.dateTime)
      ]
    else null
  end

/*
 * Check if procedure was completed
 */
define function IsCompletedProcedure(procedure Procedure):
  procedure.status = 'completed'

/*
 * Filter for completed procedures
 */
define function CompletedProcedures(procedures List<Procedure>):
  procedures P where IsCompletedProcedure(P)

/*
 * Filter procedures by code
 */
define function ProceduresByCode(procedures List<Procedure>, code Code):
  procedures P where P.code ~ code

/*
 * Filter procedures by ValueSet
 */
define function ProceduresInValueSet(procedures List<Procedure>, valueSet ValueSet):
  procedures P where P.code in valueSet

/*
 * Filter procedures during measurement period
 */
define function ProceduresDuringMeasurementPeriod(procedures List<Procedure>):
  procedures P
  where ProcedurePerformedPeriod(P) during "Measurement Period"

/*
 * Check if patient had procedure during measurement period
 */
define function HadProcedureDuringMeasurementPeriod(procedures List<Procedure>, valueSet ValueSet):
  exists (
    ProceduresDuringMeasurementPeriod(
      CompletedProcedures(ProceduresInValueSet(procedures, valueSet))
    )
  )

// ============================================================
// ENCOUNTER HELPERS (Quality Measure Specific)
// ============================================================

/*
 * Qualifying encounters during measurement period
 * Common pattern for eCQM Initial Population
 */
define function QualifyingEncountersDuringMeasurementPeriod(encounters List<Encounter>, encounterTypes ValueSet):
  encounters E
  where E.status = 'finished'
  and E.type in encounterTypes
  and EncounterHelpers.EncounterPeriod(E) during "Measurement Period"

/*
 * Inpatient encounters during measurement period
 */
define function InpatientEncountersDuringMeasurementPeriod(encounters List<Encounter>):
  encounters E
  where EncounterHelpers.IsInpatient(E)
  and EncounterHelpers.IsFinished(E)
  and EncounterHelpers.EncounterPeriod(E) during "Measurement Period"

/*
 * ED encounters during measurement period
 */
define function EDEncountersDuringMeasurementPeriod(encounters List<Encounter>):
  encounters E
  where EncounterHelpers.IsEmergency(E)
  and EncounterHelpers.IsFinished(E)
  and EncounterHelpers.EncounterPeriod(E) during "Measurement Period"

/*
 * Outpatient encounters during measurement period
 */
define function OutpatientEncountersDuringMeasurementPeriod(encounters List<Encounter>):
  encounters E
  where EncounterHelpers.IsOutpatient(E)
  and EncounterHelpers.IsFinished(E)
  and EncounterHelpers.EncounterPeriod(E) during "Measurement Period"

// ============================================================
// DIAGNOSTIC REPORT HELPERS
// ============================================================

/*
 * Get diagnostic report effective DateTime
 */
define function DiagnosticReportDateTime(report DiagnosticReport):
  case
    when report.effective is FHIR.dateTime then
      FHIRHelpers.ToDateTime(report.effective as FHIR.dateTime)
    when report.effective is FHIR.Period then
      FHIRHelpers.ToDateTime((report.effective as FHIR.Period).start)
    else null
  end

/*
 * Check if diagnostic report is final
 */
define function IsFinalReport(report DiagnosticReport):
  report.status in { 'final', 'amended', 'corrected' }

/*
 * Filter for final diagnostic reports
 */
define function FinalDiagnosticReports(reports List<DiagnosticReport>):
  reports R where IsFinalReport(R)

/*
 * Filter diagnostic reports by code
 */
define function DiagnosticReportsByCode(reports List<DiagnosticReport>, code Code):
  reports R where R.code ~ code

/*
 * Filter diagnostic reports by ValueSet
 */
define function DiagnosticReportsInValueSet(reports List<DiagnosticReport>, valueSet ValueSet):
  reports R where R.code in valueSet

// ============================================================
// IMMUNIZATION HELPERS
// ============================================================

/*
 * Get immunization date
 */
define function ImmunizationDate(immunization Immunization):
  case
    when immunization.occurrence is FHIR.dateTime then
      date from FHIRHelpers.ToDateTime(immunization.occurrence as FHIR.dateTime)
    when immunization.occurrence is FHIR.string then
      null // Cannot parse string dates
    else null
  end

/*
 * Check if immunization was completed
 */
define function IsCompletedImmunization(immunization Immunization):
  immunization.status = 'completed'

/*
 * Filter for completed immunizations
 */
define function CompletedImmunizations(immunizations List<Immunization>):
  immunizations I where IsCompletedImmunization(I)

/*
 * Filter immunizations by vaccine code
 */
define function ImmunizationsByVaccine(immunizations List<Immunization>, vaccineCode Code):
  immunizations I where I.vaccineCode ~ vaccineCode

/*
 * Filter immunizations by ValueSet
 */
define function ImmunizationsInValueSet(immunizations List<Immunization>, valueSet ValueSet):
  immunizations I where I.vaccineCode in valueSet

// ============================================================
// SERVICE REQUEST HELPERS
// ============================================================

/*
 * Get service request authored date
 */
define function ServiceRequestAuthoredOn(request ServiceRequest):
  if request.authoredOn is not null then
    date from FHIRHelpers.ToDateTime(request.authoredOn)
  else null

/*
 * Check if service request is active
 */
define function IsActiveServiceRequest(request ServiceRequest):
  request.status in { 'active', 'completed' }

/*
 * Filter for active service requests
 */
define function ActiveServiceRequests(requests List<ServiceRequest>):
  requests R where IsActiveServiceRequest(R)

/*
 * Filter service requests by code
 */
define function ServiceRequestsByCode(requests List<ServiceRequest>, code Code):
  requests R where R.code ~ code

// ============================================================
// ALLERGY INTOLERANCE HELPERS
// ============================================================

/*
 * Check if allergy is active
 */
define function IsActiveAllergy(allergy AllergyIntolerance):
  allergy.clinicalStatus ~ (Code 'active' from "AllergyIntoleranceClinicalStatus")
  and allergy.verificationStatus ~ (Code 'confirmed' from "AllergyIntoleranceVerificationStatus")

/*
 * Filter for active allergies
 */
define function ActiveAllergies(allergies List<AllergyIntolerance>):
  allergies A where IsActiveAllergy(A)

/*
 * Filter allergies by code
 */
define function AllergiesByCode(allergies List<AllergyIntolerance>, code Code):
  allergies A where A.code ~ code

/*
 * Filter allergies by ValueSet
 */
define function AllergiesInValueSet(allergies List<AllergyIntolerance>, valueSet ValueSet):
  allergies A where A.code in valueSet

/*
 * Check if patient has active allergy
 */
define function HasActiveAllergy(allergies List<AllergyIntolerance>, valueSet ValueSet):
  exists (ActiveAllergies(AllergiesInValueSet(allergies, valueSet)))

// ============================================================
// HOSPICE EXCLUSION (Common eCQM Pattern)
// ============================================================

/*
 * Standard hospice exclusion logic
 * Used by most CMS quality measures
 * Note: ValueSets defined in declarations section above
 */
define "Has Hospice Services":
  exists (
    [Encounter: "Hospice Encounter"] HospiceEncounter
    where HospiceEncounter.status = 'finished'
    and EncounterHelpers.EncounterPeriod(HospiceEncounter) overlaps "Measurement Period"
  )
  or exists (
    [Procedure: "Hospice Care Ambulatory"] HospiceProcedure
    where HospiceProcedure.status = 'completed'
    and ProcedurePerformedPeriod(HospiceProcedure) overlaps "Measurement Period"
  )
  or exists (
    [Condition: "Hospice Diagnosis"] HospiceCondition
    where IsActiveCondition(HospiceCondition)
    and ConditionPrevalencePeriod(HospiceCondition) overlaps "Measurement Period"
  )

// ============================================================
// PALLIATIVE CARE EXCLUSION (Common eCQM Pattern)
// ============================================================

/*
 * Standard palliative care exclusion logic
 * Note: ValueSets defined in declarations section above
 */
define "Has Palliative Care":
  exists (
    [Encounter: "Palliative Care Encounter"] PalliativeEncounter
    where PalliativeEncounter.status = 'finished'
    and EncounterHelpers.EncounterPeriod(PalliativeEncounter) overlaps "Measurement Period"
  )
  or exists (
    [Procedure: "Palliative Care Intervention"] PalliativeProcedure
    where PalliativeProcedure.status = 'completed'
    and ProcedurePerformedPeriod(PalliativeProcedure) overlaps "Measurement Period"
  )

// ============================================================
// QUALITY MEASURE POPULATION HELPERS
// ============================================================

/*
 * Check if patient qualifies for measure (has qualifying encounter)
 * Base pattern for Initial Population
 */
define function QualifiesForMeasure(encounters List<Encounter>, encounterTypes ValueSet, minAge Integer, maxAge Integer):
  AgeInRange(minAge, maxAge)
  and exists (QualifyingEncountersDuringMeasurementPeriod(encounters, encounterTypes))

/*
 * Standard denominator exclusions (hospice + palliative)
 */
define "Standard Denominator Exclusions":
  "Has Hospice Services"
  or "Has Palliative Care"

// ============================================================
// DATE ARITHMETIC HELPERS
// ============================================================

/*
 * Calculate days between two dates
 */
define function DaysBetween(startDate Date, endDate Date):
  difference in days between startDate and endDate

/*
 * Calculate months between two dates
 */
define function MonthsBetween(startDate Date, endDate Date):
  difference in months between startDate and endDate

/*
 * Calculate years between two dates
 */
define function YearsBetween(startDate Date, endDate Date):
  difference in years between startDate and endDate

/*
 * Start of measurement period as Date
 */
define "Start of Measurement Period":
  date from start of "Measurement Period"

/*
 * End of measurement period as Date
 */
define "End of Measurement Period":
  date from end of "Measurement Period"

/*
 * One year before measurement period
 */
define "One Year Before Measurement Period":
  Interval[
    start of "Measurement Period" - 1 year,
    start of "Measurement Period"
  ]

/*
 * Two years before measurement period
 */
define "Two Years Before Measurement Period":
  Interval[
    start of "Measurement Period" - 2 years,
    start of "Measurement Period"
  ]

// ============================================================
// CONTINUOUS VARIABLE MEASURE HELPERS
// ============================================================

/*
 * Count of events (for continuous variable measures)
 */
define function EventCount(events List<Any>):
  Count(events)

/*
 * Sum of durations (for cumulative measures)
 */
define function SumDurations(durations List<Integer>):
  Sum(durations)

/*
 * Average of values
 */
define function AverageValue(values List<Decimal>):
  Avg(values)

/*
 * Median of values
 */
define function MedianValue(values List<Decimal>):
  Median(values)

// ============================================================
// NULL HANDLING HELPERS
// ============================================================

/*
 * Coalesce for dates - return first non-null
 */
define function CoalesceDate(date1 Date, date2 Date):
  Coalesce(date1, date2)

/*
 * Coalesce for DateTimes
 */
define function CoalesceDateTime(dt1 DateTime, dt2 DateTime):
  Coalesce(dt1, dt2)

/*
 * Coalesce for Integers
 */
define function CoalesceInteger(int1 Integer, int2 Integer):
  Coalesce(int1, int2)

/*
 * Default value if null
 */
define function DefaultIfNull(value Integer, defaultValue Integer):
  if value is null then defaultValue else value

// ============================================================
// LATEST/EARLIEST HELPERS
// ============================================================

/*
 * Get latest of two dates
 */
define function LatestOf(date1 Date, date2 Date):
  if date1 is null then date2
  else if date2 is null then date1
  else if date1 > date2 then date1
  else date2

/*
 * Get earliest of two dates
 */
define function EarliestOf(date1 Date, date2 Date):
  if date1 is null then date2
  else if date2 is null then date1
  else if date1 < date2 then date1
  else date2

/*
 * Normalize interval to measurement period
 * Clamps interval to start/end of measurement period
 */
define function NormalizeToMeasurementPeriod(interval Interval<DateTime>):
  Interval[
    LatestOf(date from start of interval, "Start of Measurement Period"),
    EarliestOf(date from end of interval, "End of Measurement Period")
  ]
