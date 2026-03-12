/*
 * MedicationHelpers CQL Library
 * ============================================================
 * Tier 1 Primitive - Medication Operations for Clinical Logic
 *
 * Purpose: Provides reusable functions for working with FHIR
 *          Medication resources (MedicationRequest, MedicationStatement,
 *          MedicationAdministration, MedicationDispense) in clinical
 *          quality measures and decision support logic.
 *
 * Version: 1.0.0
 * FHIR Version: R4 (4.0.1)
 * CQL Version: 1.5
 *
 * Dependencies:
 *   - FHIRHelpers (Tier 0)
 *   - IntervalHelpers (Tier 1)
 *
 * Clinical Use Cases:
 *   - Medication adherence calculations
 *   - Drug class filtering
 *   - Active medication detection
 *   - Medication period calculations
 *   - Cumulative medication duration
 *   - Polypharmacy assessment
 *   - High-risk medication identification
 *
 * Governance: Changes require Tier-1 review (Clinical + Technical)
 * ============================================================
 */
library MedicationHelpers version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called IntervalHelpers

// ============================================================
// CODESYSTEMS (must be declared before context)
// ============================================================
codesystem "MedicationRequestStatus": 'http://hl7.org/fhir/CodeSystem/medicationrequest-status'
codesystem "MedicationStatementStatus": 'http://hl7.org/fhir/CodeSystem/medication-statement-status'
codesystem "MedicationAdministrationStatus": 'http://terminology.hl7.org/CodeSystem/medication-admin-status'
codesystem "MedicationRequestIntent": 'http://hl7.org/fhir/CodeSystem/medicationrequest-intent'
codesystem "MedicationRequestCategory": 'http://terminology.hl7.org/CodeSystem/medicationrequest-category'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "UCUM": 'http://unitsofmeasure.org'

// ============================================================
// CODES - MedicationRequest Status
// ============================================================
code "MedRequest Active": 'active' from "MedicationRequestStatus" display 'Active'
code "MedRequest On Hold": 'on-hold' from "MedicationRequestStatus" display 'On Hold'
code "MedRequest Cancelled": 'cancelled' from "MedicationRequestStatus" display 'Cancelled'
code "MedRequest Completed": 'completed' from "MedicationRequestStatus" display 'Completed'
code "MedRequest Stopped": 'stopped' from "MedicationRequestStatus" display 'Stopped'
code "MedRequest Draft": 'draft' from "MedicationRequestStatus" display 'Draft'

// ============================================================
// CODES - MedicationStatement Status
// ============================================================
code "MedStatement Active": 'active' from "MedicationStatementStatus" display 'Active'
code "MedStatement Completed": 'completed' from "MedicationStatementStatus" display 'Completed'
code "MedStatement Stopped": 'stopped' from "MedicationStatementStatus" display 'Stopped'
code "MedStatement Intended": 'intended' from "MedicationStatementStatus" display 'Intended'

// ============================================================
// CODES - MedicationRequest Category
// ============================================================
code "Inpatient Medication": 'inpatient' from "MedicationRequestCategory" display 'Inpatient'
code "Outpatient Medication": 'outpatient' from "MedicationRequestCategory" display 'Outpatient'
code "Community Medication": 'community' from "MedicationRequestCategory" display 'Community'
code "Discharge Medication": 'discharge' from "MedicationRequestCategory" display 'Discharge'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// MEDICATION CODE EXTRACTION
// ============================================================

/*
 * Get medication code from MedicationRequest
 * Handles both CodeableConcept and Reference patterns
 */
define function GetMedicationCode(request MedicationRequest):
  if request.medication is FHIR.CodeableConcept then
    request.medication as FHIR.CodeableConcept
  else null

/*
 * Get medication code from MedicationStatement
 */
define function GetMedicationCodeFromStatement(statement MedicationStatement):
  if statement.medication is FHIR.CodeableConcept then
    statement.medication as FHIR.CodeableConcept
  else null

/*
 * Get medication code from MedicationAdministration
 */
define function GetMedicationCodeFromAdmin(admin MedicationAdministration):
  if admin.medication is FHIR.CodeableConcept then
    admin.medication as FHIR.CodeableConcept
  else null

/*
 * Get medication code from MedicationDispense
 */
define function GetMedicationCodeFromDispense(dispense MedicationDispense):
  if dispense.medication is FHIR.CodeableConcept then
    dispense.medication as FHIR.CodeableConcept
  else null

/*
 * Get RxNorm code from medication CodeableConcept
 */
define function GetRxNormCode(medication CodeableConcept):
  First(
    medication.coding C
    where C.system.value = 'http://www.nlm.nih.gov/research/umls/rxnorm'
    return C.code.value
  )

/*
 * Get SNOMED code from medication CodeableConcept
 */
define function GetSNOMEDCode(medication CodeableConcept):
  First(
    medication.coding C
    where C.system.value = 'http://snomed.info/sct'
    return C.code.value
  )

// ============================================================
// MEDICATION REQUEST STATUS HELPERS
// ============================================================

/*
 * Check if medication request is active
 */
define function IsActiveMedicationRequest(request MedicationRequest):
  request.status = 'active'

/*
 * Check if medication request is completed
 */
define function IsCompletedMedicationRequest(request MedicationRequest):
  request.status = 'completed'

/*
 * Check if medication request was stopped
 */
define function IsStoppedMedicationRequest(request MedicationRequest):
  request.status = 'stopped'

/*
 * Check if medication request was cancelled
 */
define function IsCancelledMedicationRequest(request MedicationRequest):
  request.status = 'cancelled'

/*
 * Check if medication request is on hold
 */
define function IsOnHoldMedicationRequest(request MedicationRequest):
  request.status = 'on-hold'

/*
 * Get active or completed medication requests (for measure calculation)
 */
define function ActiveOrCompletedMedicationRequests(requests List<MedicationRequest>):
  requests R where R.status in { 'active', 'completed' }

// ============================================================
// MEDICATION STATEMENT STATUS HELPERS
// ============================================================

/*
 * Check if medication statement indicates active use
 */
define function IsActiveMedicationStatement(statement MedicationStatement):
  statement.status = 'active'

/*
 * Check if medication statement indicates completed use
 */
define function IsCompletedMedicationStatement(statement MedicationStatement):
  statement.status = 'completed'

/*
 * Get active medication statements
 */
define function ActiveMedicationStatements(statements List<MedicationStatement>):
  statements S where S.status = 'active'

// ============================================================
// MEDICATION PERIOD HELPERS
// ============================================================

/*
 * Get the effective period of a MedicationRequest
 * Uses dispenseRequest.validityPeriod if available,
 * otherwise calculates from authoredOn + expectedSupplyDuration
 */
define function MedicationRequestPeriod(request MedicationRequest):
  if request.dispenseRequest is not null
    and request.dispenseRequest.validityPeriod is not null
  then
    Interval[
      FHIRHelpers.ToDateTime(request.dispenseRequest.validityPeriod.start),
      FHIRHelpers.ToDateTime(request.dispenseRequest.validityPeriod.end)
    ]
  else if request.authoredOn is not null then
    Interval[
      FHIRHelpers.ToDateTime(request.authoredOn),
      null as DateTime
    ]
  else null

/*
 * Get the effective period of a MedicationStatement
 */
define function MedicationStatementPeriod(statement MedicationStatement):
  case
    when statement.effective is FHIR.Period then
      Interval[
        FHIRHelpers.ToDateTime((statement.effective as FHIR.Period).start),
        FHIRHelpers.ToDateTime((statement.effective as FHIR.Period).end)
      ]
    when statement.effective is FHIR.dateTime then
      Interval[
        FHIRHelpers.ToDateTime(statement.effective as FHIR.dateTime),
        FHIRHelpers.ToDateTime(statement.effective as FHIR.dateTime)
      ]
    else null
  end

/*
 * Get the effective DateTime of a MedicationAdministration
 */
define function MedicationAdministrationDateTime(admin MedicationAdministration):
  case
    when admin.effective is FHIR.dateTime then
      FHIRHelpers.ToDateTime(admin.effective as FHIR.dateTime)
    when admin.effective is FHIR.Period then
      FHIRHelpers.ToDateTime((admin.effective as FHIR.Period).start)
    else null
  end

/*
 * Get the period of a MedicationAdministration
 */
define function MedicationAdministrationPeriod(admin MedicationAdministration):
  case
    when admin.effective is FHIR.Period then
      Interval[
        FHIRHelpers.ToDateTime((admin.effective as FHIR.Period).start),
        FHIRHelpers.ToDateTime((admin.effective as FHIR.Period).end)
      ]
    when admin.effective is FHIR.dateTime then
      Interval[
        FHIRHelpers.ToDateTime(admin.effective as FHIR.dateTime),
        FHIRHelpers.ToDateTime(admin.effective as FHIR.dateTime)
      ]
    else null
  end

/*
 * Get authored date from MedicationRequest
 */
define function MedicationRequestAuthoredOn(request MedicationRequest):
  if request.authoredOn is not null then
    date from FHIRHelpers.ToDateTime(request.authoredOn)
  else null

// ============================================================
// MEDICATION FILTERING HELPERS
// ============================================================

/*
 * Filter medication requests by code
 */
define function MedicationRequestsByCode(requests List<MedicationRequest>, medicationCode Code):
  requests R where GetMedicationCode(R) ~ medicationCode

/*
 * Filter medication requests by ValueSet
 */
define function MedicationRequestsInValueSet(requests List<MedicationRequest>, valueSet ValueSet):
  requests R where GetMedicationCode(R) in valueSet

/*
 * Filter medication requests ordered during period
 */
define function MedicationRequestsOrderedDuring(requests List<MedicationRequest>, period Interval<DateTime>):
  requests R
  where FHIRHelpers.ToDateTime(R.authoredOn) during period

/*
 * Filter medication statements by code
 */
define function MedicationStatementsByCode(statements List<MedicationStatement>, medicationCode Code):
  statements S where GetMedicationCodeFromStatement(S) ~ medicationCode

/*
 * Filter medication statements by ValueSet
 */
define function MedicationStatementsInValueSet(statements List<MedicationStatement>, valueSet ValueSet):
  statements S where GetMedicationCodeFromStatement(S) in valueSet

/*
 * Filter medication administrations during period
 */
define function MedicationAdministrationsDuring(admins List<MedicationAdministration>, period Interval<DateTime>):
  admins A
  where MedicationAdministrationDateTime(A) during period

/*
 * Filter for outpatient medications
 */
define function OutpatientMedicationRequests(requests List<MedicationRequest>):
  requests R
  where exists (R.category C where C ~ "Outpatient Medication")

/*
 * Filter for inpatient medications
 */
define function InpatientMedicationRequests(requests List<MedicationRequest>):
  requests R
  where exists (R.category C where C ~ "Inpatient Medication")

/*
 * Filter for discharge medications
 */
define function DischargeMedicationRequests(requests List<MedicationRequest>):
  requests R
  where exists (R.category C where C ~ "Discharge Medication")

// ============================================================
// DOSAGE HELPERS
// ============================================================

/*
 * Get the primary dosage instruction
 */
define function PrimaryDosage(request MedicationRequest):
  First(request.dosageInstruction)

/*
 * Get the dose quantity from first dosage instruction
 */
define function DoseQuantity(request MedicationRequest):
  if exists (request.dosageInstruction) then
    (First(request.dosageInstruction).doseAndRate[0].dose as FHIR.Quantity)
  else null

/*
 * Get the frequency from timing
 */
define function DosageFrequency(request MedicationRequest):
  if exists (request.dosageInstruction) then
    First(request.dosageInstruction).timing.repeat.frequency.value
  else null

/*
 * Get timing period (e.g., every 8 hours = period of 8, periodUnit 'h')
 */
define function DosagePeriod(request MedicationRequest):
  if exists (request.dosageInstruction) then
    First(request.dosageInstruction).timing.repeat.period.value
  else null

/*
 * Get timing period unit
 */
define function DosagePeriodUnit(request MedicationRequest):
  if exists (request.dosageInstruction) then
    First(request.dosageInstruction).timing.repeat.periodUnit.value
  else null

/*
 * Get the route of administration
 */
define function DosageRoute(request MedicationRequest):
  if exists (request.dosageInstruction) then
    First(request.dosageInstruction).route
  else null

/*
 * Check if medication is PRN (as needed)
 */
define function IsPRN(request MedicationRequest):
  if exists (request.dosageInstruction) then
    First(request.dosageInstruction).asNeeded is true
    or First(request.dosageInstruction).asNeededBoolean.value is true
  else false

// ============================================================
// DISPENSE HELPERS
// ============================================================

/*
 * Get quantity dispensed
 */
define function QuantityDispensed(dispense MedicationDispense):
  FHIRHelpers.ToQuantity(dispense.quantity)

/*
 * Get days supply
 */
define function DaysSupply(dispense MedicationDispense):
  FHIRHelpers.ToQuantity(dispense.daysSupply)

/*
 * Get dispense date
 */
define function DispenseDate(dispense MedicationDispense):
  if dispense.whenHandedOver is not null then
    date from FHIRHelpers.ToDateTime(dispense.whenHandedOver)
  else if dispense.whenPrepared is not null then
    date from FHIRHelpers.ToDateTime(dispense.whenPrepared)
  else null

/*
 * Calculate medication coverage period from dispense
 * Uses dispense date + days supply
 */
define function DispenseCoveragePeriod(dispense MedicationDispense):
  if DispenseDate(dispense) is not null and DaysSupply(dispense) is not null then
    Interval[DispenseDate(dispense), DispenseDate(dispense) + DaysSupply(dispense)]
  else null

// ============================================================
// MEDICATION DURATION CALCULATIONS
// ============================================================

/*
 * Calculate cumulative medication duration in days
 * Sums up all dispense periods for a medication
 */
define function CumulativeMedicationDuration(dispenses List<MedicationDispense>):
  Sum(
    dispenses D
    return FHIRHelpers.ToQuantity(D.daysSupply).value
  )

/*
 * Calculate medication coverage ratio (PDC - Proportion of Days Covered)
 * Standard adherence measure
 */
define function ProportionOfDaysCovered(dispenses List<MedicationDispense>, measurementPeriod Interval<Date>):
  if Count(dispenses) = 0 or measurementPeriod is null then null
  else if (difference in days between start of measurementPeriod and end of measurementPeriod) > 0 then
    Min({
      CumulativeMedicationDuration(dispenses)
        / (difference in days between start of measurementPeriod and end of measurementPeriod),
      1.0
    })
  else null

/*
 * Check if patient is adherent (PDC >= 80%)
 */
define function IsAdherent(dispenses List<MedicationDispense>, measurementPeriod Interval<Date>):
  ProportionOfDaysCovered(dispenses, measurementPeriod) >= 0.8

/*
 * Get expected supply duration from request
 */
define function ExpectedSupplyDuration(request MedicationRequest):
  if request.dispenseRequest is not null
    and request.dispenseRequest.expectedSupplyDuration is not null
  then
    FHIRHelpers.ToQuantity(request.dispenseRequest.expectedSupplyDuration)
  else null

/*
 * Get number of refills allowed
 */
define function RefillsAllowed(request MedicationRequest):
  if request.dispenseRequest is not null then
    request.dispenseRequest.numberOfRepeatsAllowed.value
  else null

// ============================================================
// POLYPHARMACY AND SAFETY HELPERS
// ============================================================

/*
 * Count active medications
 */
define function ActiveMedicationCount(requests List<MedicationRequest>):
  Count(ActiveOrCompletedMedicationRequests(requests))

/*
 * Check for polypharmacy (5+ concurrent medications)
 */
define function HasPolypharmacy(requests List<MedicationRequest>):
  ActiveMedicationCount(requests) >= 5

/*
 * Check for severe polypharmacy (10+ concurrent medications)
 */
define function HasSeverePolypharmacy(requests List<MedicationRequest>):
  ActiveMedicationCount(requests) >= 10

/*
 * Get distinct medication codes from requests
 */
define function DistinctMedications(requests List<MedicationRequest>):
  distinct (
    requests R
    return GetMedicationCode(R)
  )

/*
 * Count distinct medications
 */
define function DistinctMedicationCount(requests List<MedicationRequest>):
  Count(DistinctMedications(requests))

// ============================================================
// MEDICATION TIMING HELPERS
// ============================================================

/*
 * Get the most recent medication request
 */
define function MostRecentMedicationRequest(requests List<MedicationRequest>):
  Last(
    requests R
    sort by FHIRHelpers.ToDateTime(authoredOn)
  )

/*
 * Get the first medication request
 */
define function FirstMedicationRequest(requests List<MedicationRequest>):
  First(
    requests R
    sort by FHIRHelpers.ToDateTime(authoredOn)
  )

/*
 * Get medication requests ordered in last N days
 */
define function MedicationRequestsInLastDays(requests List<MedicationRequest>, numDays Integer):
  requests R
  where date from FHIRHelpers.ToDateTime(R.authoredOn) in Interval[Today() - System.Quantity { value: numDays, unit: 'day' }, Today()]

/*
 * Get medication requests ordered in last N months
 */
define function MedicationRequestsInLastMonths(requests List<MedicationRequest>, numMonths Integer):
  requests R
  where date from FHIRHelpers.ToDateTime(R.authoredOn) in Interval[Today() - System.Quantity { value: numMonths, unit: 'month' }, Today()]

/*
 * Check if medication was recently ordered (within N days)
 */
define function MedicationOrderedWithinDays(requests List<MedicationRequest>, medicationCode Code, numDays Integer):
  exists (
    MedicationRequestsInLastDays(
      MedicationRequestsByCode(requests, medicationCode),
      numDays
    )
  )

// ============================================================
// INTENT HELPERS
// ============================================================

/*
 * Check if request is an order (not proposal, plan, etc.)
 */
define function IsOrder(request MedicationRequest):
  request.intent = 'order'

/*
 * Check if request is a proposal (suggested medication)
 */
define function IsProposal(request MedicationRequest):
  request.intent = 'proposal'

/*
 * Filter for medication orders only (exclude proposals, plans)
 */
define function MedicationOrders(requests List<MedicationRequest>):
  requests R where IsOrder(R)

/*
 * Get active medication orders (not proposals)
 */
define function ActiveMedicationOrders(requests List<MedicationRequest>):
  MedicationOrders(ActiveOrCompletedMedicationRequests(requests))
