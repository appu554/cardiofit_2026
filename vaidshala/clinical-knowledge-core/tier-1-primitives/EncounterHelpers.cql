/*
 * EncounterHelpers CQL Library
 * ============================================================
 * Tier 1 Primitive - Encounter Operations for Clinical Logic
 *
 * Purpose: Provides reusable functions for working with FHIR
 *          Encounter resources in clinical quality measures
 *          and decision support logic.
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
 *   - Encounter filtering by type, class, status
 *   - Inpatient vs outpatient differentiation
 *   - Encounter period calculations
 *   - Length of stay analysis
 *   - Emergency department visits
 *   - Telehealth encounter detection
 *
 * Governance: Changes require Tier-1 review (Clinical + Technical)
 * ============================================================
 */
library EncounterHelpers version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called IntervalHelpers

// ============================================================
// CODESYSTEMS (must be declared before context)
// ============================================================
codesystem "EncounterClass": 'http://terminology.hl7.org/CodeSystem/v3-ActCode'
codesystem "EncounterStatus": 'http://hl7.org/fhir/encounter-status'
codesystem "EncounterType": 'http://terminology.hl7.org/CodeSystem/encounter-type'
codesystem "ServiceType": 'http://terminology.hl7.org/CodeSystem/service-type'
codesystem "AdmitSource": 'http://terminology.hl7.org/CodeSystem/admit-source'
codesystem "DischargeDisposition": 'http://terminology.hl7.org/CodeSystem/discharge-disposition'

// ============================================================
// CODES - Encounter Class
// ============================================================
code "Inpatient Encounter": 'IMP' from "EncounterClass" display 'inpatient encounter'
code "Outpatient Encounter": 'AMB' from "EncounterClass" display 'ambulatory'
code "Emergency Encounter": 'EMER' from "EncounterClass" display 'emergency'
code "Virtual Encounter": 'VR' from "EncounterClass" display 'virtual'
code "Home Health Encounter": 'HH' from "EncounterClass" display 'home health'
code "Short Stay Encounter": 'SS' from "EncounterClass" display 'short stay'
code "Observation Encounter": 'OBSENC' from "EncounterClass" display 'observation encounter'
code "Pre-Admission Encounter": 'PRENC' from "EncounterClass" display 'pre-admission'

// ============================================================
// CODES - Encounter Status
// ============================================================
code "Encounter Planned": 'planned' from "EncounterStatus" display 'Planned'
code "Encounter Arrived": 'arrived' from "EncounterStatus" display 'Arrived'
code "Encounter Triaged": 'triaged' from "EncounterStatus" display 'Triaged'
code "Encounter In Progress": 'in-progress' from "EncounterStatus" display 'In Progress'
code "Encounter On Leave": 'onleave' from "EncounterStatus" display 'On Leave'
code "Encounter Finished": 'finished' from "EncounterStatus" display 'Finished'
code "Encounter Cancelled": 'cancelled' from "EncounterStatus" display 'Cancelled'
code "Encounter Unknown": 'unknown' from "EncounterStatus" display 'Unknown'

// ============================================================
// CODES - Discharge Disposition
// ============================================================
code "Discharge to Home": 'home' from "DischargeDisposition" display 'Home'
code "Discharge to SNF": 'snf' from "DischargeDisposition" display 'Skilled Nursing Facility'
code "Discharge to Rehab": 'rehab' from "DischargeDisposition" display 'Rehabilitation'
code "Discharge to Long Term Care": 'long' from "DischargeDisposition" display 'Long Term Care'
code "Discharge Left AMA": 'aadvice' from "DischargeDisposition" display 'Left Against Medical Advice'
code "Discharge Expired": 'exp' from "DischargeDisposition" display 'Expired'
code "Discharge to Hospice": 'hosp' from "DischargeDisposition" display 'Hospice'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// ENCOUNTER PERIOD HELPERS
// ============================================================

/*
 * Get the effective period of an encounter
 * Returns the period from the Encounter.period element
 */
define function EncounterPeriod(encounter Encounter):
  if encounter.period is not null then
    Interval[
      FHIRHelpers.ToDateTime(encounter.period.start),
      FHIRHelpers.ToDateTime(encounter.period.end)
    ]
  else null

/*
 * Get the effective period as a Date interval
 * Useful for date-based quality measure calculations
 */
define function EncounterDatePeriod(encounter Encounter):
  if encounter.period is not null then
    Interval[
      date from FHIRHelpers.ToDateTime(encounter.period.start),
      date from FHIRHelpers.ToDateTime(encounter.period.end)
    ]
  else null

/*
 * Get the start date of an encounter
 */
define function EncounterStartDate(encounter Encounter):
  if encounter.period is not null and encounter.period.start is not null then
    date from FHIRHelpers.ToDateTime(encounter.period.start)
  else null

/*
 * Get the end date of an encounter
 */
define function EncounterEndDate(encounter Encounter):
  if encounter.period is not null and encounter.period.end is not null then
    date from FHIRHelpers.ToDateTime(encounter.period.end)
  else null

/*
 * Get the admission DateTime
 */
define function AdmissionDateTime(encounter Encounter):
  if encounter.period is not null then
    FHIRHelpers.ToDateTime(encounter.period.start)
  else null

/*
 * Get the discharge DateTime
 */
define function DischargeDateTime(encounter Encounter):
  if encounter.period is not null then
    FHIRHelpers.ToDateTime(encounter.period.end)
  else null

// ============================================================
// LENGTH OF STAY CALCULATIONS
// ============================================================

/*
 * Calculate length of stay in days
 * Standard hospital LOS calculation
 */
define function LengthOfStayInDays(encounter Encounter):
  if encounter.period is not null
    and encounter.period.start is not null
    and encounter.period.end is not null
  then
    difference in days between
      FHIRHelpers.ToDateTime(encounter.period.start) and
      FHIRHelpers.ToDateTime(encounter.period.end)
  else null

/*
 * Calculate length of stay in hours
 * Used for ED and observation encounters
 */
define function LengthOfStayInHours(encounter Encounter):
  if encounter.period is not null
    and encounter.period.start is not null
    and encounter.period.end is not null
  then
    difference in hours between
      FHIRHelpers.ToDateTime(encounter.period.start) and
      FHIRHelpers.ToDateTime(encounter.period.end)
  else null

/*
 * Calculate length of stay in minutes
 * Used for procedure and short encounters
 */
define function LengthOfStayInMinutes(encounter Encounter):
  if encounter.period is not null
    and encounter.period.start is not null
    and encounter.period.end is not null
  then
    difference in minutes between
      FHIRHelpers.ToDateTime(encounter.period.start) and
      FHIRHelpers.ToDateTime(encounter.period.end)
  else null

/*
 * Check if encounter exceeds specified days threshold
 * Used for extended stay identification
 */
define function LengthOfStayExceeds(encounter Encounter, numDays Integer):
  LengthOfStayInDays(encounter) > numDays

/*
 * Check if encounter is short stay (< 24 hours)
 */
define function IsShortStay(encounter Encounter):
  LengthOfStayInHours(encounter) < 24

/*
 * Check if encounter is extended stay (> 7 days)
 */
define function IsExtendedStay(encounter Encounter):
  LengthOfStayInDays(encounter) > 7

// ============================================================
// ENCOUNTER CLASS HELPERS
// ============================================================

/*
 * Get the encounter class code
 */
define function EncounterClass(encounter Encounter):
  encounter.class.code

/*
 * Check if encounter is inpatient
 */
define function IsInpatient(encounter Encounter):
  encounter.class ~ "Inpatient Encounter"

/*
 * Check if encounter is outpatient/ambulatory
 */
define function IsOutpatient(encounter Encounter):
  encounter.class ~ "Outpatient Encounter"

/*
 * Check if encounter is emergency department visit
 */
define function IsEmergency(encounter Encounter):
  encounter.class ~ "Emergency Encounter"

/*
 * Check if encounter is virtual/telehealth
 */
define function IsVirtual(encounter Encounter):
  encounter.class ~ "Virtual Encounter"

/*
 * Check if encounter is home health
 */
define function IsHomeHealth(encounter Encounter):
  encounter.class ~ "Home Health Encounter"

/*
 * Check if encounter is observation status
 */
define function IsObservation(encounter Encounter):
  encounter.class ~ "Observation Encounter"

// ============================================================
// ENCOUNTER STATUS HELPERS
// ============================================================

/*
 * Get the encounter status
 */
define function EncounterStatus(encounter Encounter):
  encounter.status.value

/*
 * Check if encounter is finished/completed
 */
define function IsFinished(encounter Encounter):
  encounter.status = 'finished'

/*
 * Check if encounter is currently in progress
 */
define function IsInProgress(encounter Encounter):
  encounter.status = 'in-progress'

/*
 * Check if encounter is planned (future)
 */
define function IsPlanned(encounter Encounter):
  encounter.status = 'planned'

/*
 * Check if encounter was cancelled
 */
define function IsCancelled(encounter Encounter):
  encounter.status = 'cancelled'

/*
 * Check if encounter is active (arrived, triaged, or in-progress)
 */
define function IsActive(encounter Encounter):
  encounter.status in { 'arrived', 'triaged', 'in-progress' }

/*
 * Check if encounter is closed (finished, cancelled)
 */
define function IsClosed(encounter Encounter):
  encounter.status in { 'finished', 'cancelled' }

// ============================================================
// ENCOUNTER FILTERING HELPERS
// ============================================================

/*
 * Filter encounters by class
 */
define function EncountersByClass(encounters List<Encounter>, classCode Code):
  encounters E where E.class ~ classCode

/*
 * Filter encounters by status
 */
define function EncountersByStatus(encounters List<Encounter>, status String):
  encounters E where E.status.value = status

/*
 * Filter encounters by type
 */
define function EncountersByType(encounters List<Encounter>, encounterType Code):
  encounters E where exists (E.type T where T ~ encounterType)

/*
 * Filter encounters by period (overlapping with measurement period)
 */
define function EncountersInPeriod(encounters List<Encounter>, measurementPeriod Interval<DateTime>):
  encounters E where EncounterPeriod(E) overlaps measurementPeriod

/*
 * Filter encounters by date period
 */
define function EncountersInDatePeriod(encounters List<Encounter>, datePeriod Interval<Date>):
  encounters E where EncounterDatePeriod(E) overlaps datePeriod

/*
 * Filter encounters that started during period
 */
define function EncountersStartedDuring(encounters List<Encounter>, period Interval<DateTime>):
  encounters E where AdmissionDateTime(E) during period

/*
 * Filter encounters that ended during period
 */
define function EncountersEndedDuring(encounters List<Encounter>, period Interval<DateTime>):
  encounters E where DischargeDateTime(E) during period

/*
 * Filter for finished inpatient encounters
 */
define function FinishedInpatientEncounters(encounters List<Encounter>):
  encounters E where IsInpatient(E) and IsFinished(E)

/*
 * Filter for finished ED encounters
 */
define function FinishedEDEncounters(encounters List<Encounter>):
  encounters E where IsEmergency(E) and IsFinished(E)

// ============================================================
// DISCHARGE HELPERS
// ============================================================

/*
 * Get discharge disposition code
 */
define function DischargeDisposition(encounter Encounter):
  if encounter.hospitalization is not null then
    encounter.hospitalization.dischargeDisposition
  else null

/*
 * Check if patient discharged to home
 */
define function DischargedToHome(encounter Encounter):
  DischargeDisposition(encounter) ~ "Discharge to Home"

/*
 * Check if patient discharged to SNF
 */
define function DischargedToSNF(encounter Encounter):
  DischargeDisposition(encounter) ~ "Discharge to SNF"

/*
 * Check if patient left against medical advice
 */
define function LeftAgainstMedicalAdvice(encounter Encounter):
  DischargeDisposition(encounter) ~ "Discharge Left AMA"

/*
 * Check if patient expired during encounter
 */
define function ExpiredDuringEncounter(encounter Encounter):
  DischargeDisposition(encounter) ~ "Discharge Expired"

/*
 * Check if patient discharged to hospice
 */
define function DischargedToHospice(encounter Encounter):
  DischargeDisposition(encounter) ~ "Discharge to Hospice"

/*
 * Check if patient discharged to post-acute care (SNF, Rehab, LTC)
 */
define function DischargedToPostAcuteCare(encounter Encounter):
  DischargeDisposition(encounter) in {
    "Discharge to SNF",
    "Discharge to Rehab",
    "Discharge to Long Term Care"
  }

// ============================================================
// ADMIT SOURCE HELPERS
// ============================================================

/*
 * Get admit source code
 */
define function AdmitSource(encounter Encounter):
  if encounter.hospitalization is not null then
    encounter.hospitalization.admitSource
  else null

/*
 * Check if admission was from emergency department
 */
define function AdmittedFromED(encounter Encounter):
  exists (
    from AdmitSource(encounter).coding C
    where C.code.value = 'emd'
    return C
  )

/*
 * Check if admission was a transfer
 */
define function IsTransferAdmission(encounter Encounter):
  exists (
    from AdmitSource(encounter).coding C
    where C.code.value in { 'hosp-trans', 'other' }
    return C
  )

// ============================================================
// ENCOUNTER RELATIONSHIP HELPERS
// ============================================================

/*
 * Get the most recent encounter from a list
 */
define function MostRecentEncounter(encounters List<Encounter>):
  Last(
    encounters E
    sort by start of EncounterPeriod(E)
  )

/*
 * Get the first encounter from a list
 */
define function FirstEncounter(encounters List<Encounter>):
  First(
    encounters E
    sort by start of EncounterPeriod(E)
  )

/*
 * Sort encounters by start date (ascending)
 */
define function SortEncountersByStart(encounters List<Encounter>):
  encounters E sort by start of EncounterPeriod(E)

/*
 * Sort encounters by end date (descending)
 */
define function SortEncountersByEndDesc(encounters List<Encounter>):
  encounters E sort by end of EncounterPeriod(E) desc

/*
 * Check if encounters overlap (potential duplicate or concurrent)
 */
define function EncountersOverlap(encounter1 Encounter, encounter2 Encounter):
  EncounterPeriod(encounter1) overlaps EncounterPeriod(encounter2)

/*
 * Get encounters that occurred within N days of a reference encounter
 */
define function EncountersWithinDaysOf(encounters List<Encounter>, referenceEncounter Encounter, numDays Integer):
  encounters E
  where EncounterStartDate(E) in Interval[
    EncounterEndDate(referenceEncounter),
    EncounterEndDate(referenceEncounter) + System.Quantity { value: numDays, unit: 'day' }
  ]

/*
 * Check if patient had ED visit that led to inpatient admission
 * (ED visit ends within same day as inpatient admission starts)
 */
define function EDToInpatientAdmission(edEncounter Encounter, inpatientEncounter Encounter):
  IsEmergency(edEncounter)
  and IsInpatient(inpatientEncounter)
  and EncounterEndDate(edEncounter) same day as EncounterStartDate(inpatientEncounter)

// ============================================================
// ENCOUNTER COUNT HELPERS
// ============================================================

/*
 * Count encounters by class
 */
define function CountEncountersByClass(encounters List<Encounter>, classCode Code):
  Count(EncountersByClass(encounters, classCode))

/*
 * Check if patient has had N or more encounters in period
 */
define function HasNOrMoreEncounters(encounters List<Encounter>, n Integer):
  Count(encounters) >= n

/*
 * Check if patient is high utilizer (>= 4 ED visits in 12 months)
 */
define function IsEDHighUtilizer(edEncounters List<Encounter>):
  Count(
    edEncounters E
    where IsEmergency(E)
    and EncounterStartDate(E) in IntervalHelpers.LastOneYear()
  ) >= 4

/*
 * Check for readmission (return admission within N days)
 */
define function HasReadmissionWithinDays(inpatientEncounters List<Encounter>, numDays Integer):
  exists (
    from FinishedInpatientEncounters(inpatientEncounters) E1,
         FinishedInpatientEncounters(inpatientEncounters) E2
    where E1 != E2
    and EncounterStartDate(E2) in Interval[
      EncounterEndDate(E1),
      EncounterEndDate(E1) + System.Quantity { value: numDays, unit: 'day' }
    ]
    return true
  )

/*
 * Check for 30-day readmission (standard CMS measure)
 */
define function Has30DayReadmission(inpatientEncounters List<Encounter>):
  HasReadmissionWithinDays(inpatientEncounters, 30)
