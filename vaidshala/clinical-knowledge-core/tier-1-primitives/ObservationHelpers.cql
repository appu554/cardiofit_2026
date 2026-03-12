/*
 * ObservationHelpers CQL Library
 * ============================================================
 * Tier 1 Primitive - Observation Operations for Clinical Logic
 *
 * Purpose: Provides reusable functions for working with FHIR
 *          Observation resources (labs, vitals, assessments)
 *          in clinical quality measures and decision support.
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
 *   - Lab result trending and analysis
 *   - Vital sign monitoring
 *   - Result interpretation
 *   - Critical value detection
 *   - Most recent value retrieval
 *   - Value change calculations
 *   - Quality measure observation criteria
 *
 * Governance: Changes require Tier-1 review (Clinical + Technical)
 * ============================================================
 */
library ObservationHelpers version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called IntervalHelpers

// ============================================================
// CODESYSTEMS (must be declared before context)
// ============================================================
codesystem "ObservationStatus": 'http://hl7.org/fhir/observation-status'
codesystem "ObservationCategory": 'http://terminology.hl7.org/CodeSystem/observation-category'
codesystem "ObservationInterpretation": 'http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation'
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "UCUM": 'http://unitsofmeasure.org'

// ============================================================
// CODES - Observation Status
// ============================================================
code "Observation Registered": 'registered' from "ObservationStatus" display 'Registered'
code "Observation Preliminary": 'preliminary' from "ObservationStatus" display 'Preliminary'
code "Observation Final": 'final' from "ObservationStatus" display 'Final'
code "Observation Amended": 'amended' from "ObservationStatus" display 'Amended'
code "Observation Corrected": 'corrected' from "ObservationStatus" display 'Corrected'
code "Observation Cancelled": 'cancelled' from "ObservationStatus" display 'Cancelled'
code "Observation Entered In Error": 'entered-in-error' from "ObservationStatus" display 'Entered in Error'

// ============================================================
// CODES - Observation Categories
// ============================================================
code "Vital Signs Category": 'vital-signs' from "ObservationCategory" display 'Vital Signs'
code "Laboratory Category": 'laboratory' from "ObservationCategory" display 'Laboratory'
code "Imaging Category": 'imaging' from "ObservationCategory" display 'Imaging'
code "Procedure Category": 'procedure' from "ObservationCategory" display 'Procedure'
code "Survey Category": 'survey' from "ObservationCategory" display 'Survey'
code "Exam Category": 'exam' from "ObservationCategory" display 'Exam'
code "Social History Category": 'social-history' from "ObservationCategory" display 'Social History'
code "Activity Category": 'activity' from "ObservationCategory" display 'Activity'

// ============================================================
// CODES - Interpretation
// ============================================================
code "Normal Interpretation": 'N' from "ObservationInterpretation" display 'Normal'
code "Abnormal Interpretation": 'A' from "ObservationInterpretation" display 'Abnormal'
code "High Interpretation": 'H' from "ObservationInterpretation" display 'High'
code "Low Interpretation": 'L' from "ObservationInterpretation" display 'Low'
code "Critical High Interpretation": 'HH' from "ObservationInterpretation" display 'Critical High'
code "Critical Low Interpretation": 'LL' from "ObservationInterpretation" display 'Critical Low'
code "Positive Interpretation": 'POS' from "ObservationInterpretation" display 'Positive'
code "Negative Interpretation": 'NEG' from "ObservationInterpretation" display 'Negative'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// OBSERVATION EFFECTIVE DATE/TIME HELPERS
// ============================================================

/*
 * Get the effective DateTime of an observation
 * Handles dateTime, Period, instant, and Timing
 */
define function ObservationDateTime(observation Observation):
  case
    when observation.effective is FHIR.dateTime then
      FHIRHelpers.ToDateTime(observation.effective as FHIR.dateTime)
    when observation.effective is FHIR.instant then
      FHIRHelpers.ToDateTime(observation.effective as FHIR.instant)
    when observation.effective is FHIR.Period then
      FHIRHelpers.ToDateTime((observation.effective as FHIR.Period).start)
    else null
  end

/*
 * Get the effective Date of an observation
 */
define function ObservationDate(observation Observation):
  date from ObservationDateTime(observation)

/*
 * Get the effective Period of an observation (if Period type)
 */
define function ObservationPeriod(observation Observation):
  if observation.effective is FHIR.Period then
    Interval[
      FHIRHelpers.ToDateTime((observation.effective as FHIR.Period).start),
      FHIRHelpers.ToDateTime((observation.effective as FHIR.Period).end)
    ]
  else
    Interval[ObservationDateTime(observation), ObservationDateTime(observation)]

/*
 * Get issued DateTime (when observation was available)
 */
define function ObservationIssuedDateTime(observation Observation):
  FHIRHelpers.ToDateTime(observation.issued)

// ============================================================
// OBSERVATION VALUE EXTRACTION
// ============================================================

/*
 * Get Quantity value from observation
 */
define function ObservationQuantityValue(observation Observation):
  if observation.value is FHIR.Quantity then
    FHIRHelpers.ToQuantity(observation.value as FHIR.Quantity)
  else null

/*
 * Get numeric value from observation (strips unit)
 */
define function ObservationNumericValue(observation Observation):
  if observation.value is FHIR.Quantity then
    (observation.value as FHIR.Quantity).value.value
  else null

/*
 * Get unit from Quantity observation
 */
define function ObservationUnit(observation Observation):
  if observation.value is FHIR.Quantity then
    (observation.value as FHIR.Quantity).unit.value
  else null

/*
 * Get CodeableConcept value from observation
 */
define function ObservationCodeableConceptValue(observation Observation):
  if observation.value is FHIR.CodeableConcept then
    observation.value as FHIR.CodeableConcept
  else null

/*
 * Get String value from observation
 */
define function ObservationStringValue(observation Observation):
  if observation.value is FHIR.string then
    (observation.value as FHIR.string).value
  else null

/*
 * Get Boolean value from observation
 */
define function ObservationBooleanValue(observation Observation):
  if observation.value is FHIR.boolean then
    (observation.value as FHIR.boolean).value
  else null

/*
 * Get Integer value from observation
 */
define function ObservationIntegerValue(observation Observation):
  if observation.value is FHIR.integer then
    (observation.value as FHIR.integer).value
  else null

/*
 * Get Range value from observation
 */
define function ObservationRangeValue(observation Observation):
  if observation.value is FHIR.Range then
    Interval[
      FHIRHelpers.ToQuantity((observation.value as FHIR.Range).low),
      FHIRHelpers.ToQuantity((observation.value as FHIR.Range).high)
    ]
  else null

/*
 * Get Ratio value from observation (numerator/denominator)
 */
define function ObservationRatioNumerator(observation Observation):
  if observation.value is FHIR.Ratio then
    FHIRHelpers.ToQuantity((observation.value as FHIR.Ratio).numerator)
  else null

define function ObservationRatioDenominator(observation Observation):
  if observation.value is FHIR.Ratio then
    FHIRHelpers.ToQuantity((observation.value as FHIR.Ratio).denominator)
  else null

// ============================================================
// OBSERVATION STATUS HELPERS
// ============================================================

/*
 * Check if observation has final status
 */
define function IsFinalObservation(observation Observation):
  observation.status = 'final'

/*
 * Check if observation is amended (corrected final)
 */
define function IsAmendedObservation(observation Observation):
  observation.status = 'amended'

/*
 * Check if observation is corrected
 */
define function IsCorrectedObservation(observation Observation):
  observation.status = 'corrected'

/*
 * Check if observation is preliminary
 */
define function IsPreliminaryObservation(observation Observation):
  observation.status = 'preliminary'

/*
 * Check if observation is valid for clinical use
 * (final, amended, or corrected - not entered-in-error)
 */
define function IsValidObservation(observation Observation):
  observation.status in { 'final', 'amended', 'corrected' }

/*
 * Filter for valid (usable) observations only
 */
define function ValidObservations(observations List<Observation>):
  observations O where IsValidObservation(O)

// ============================================================
// OBSERVATION CATEGORY HELPERS
// ============================================================

/*
 * Get observation category
 */
define function ObservationCategory(observation Observation):
  First(observation.category)

/*
 * Check if observation is a vital sign
 */
define function IsVitalSign(observation Observation):
  exists (observation.category C where C ~ "Vital Signs Category")

/*
 * Check if observation is a lab result
 */
define function IsLabResult(observation Observation):
  exists (observation.category C where C ~ "Laboratory Category")

/*
 * Check if observation is from survey/questionnaire
 */
define function IsSurveyResult(observation Observation):
  exists (observation.category C where C ~ "Survey Category")

/*
 * Check if observation is social history
 */
define function IsSocialHistory(observation Observation):
  exists (observation.category C where C ~ "Social History Category")

/*
 * Filter for vital sign observations
 */
define function VitalSigns(observations List<Observation>):
  observations O where IsVitalSign(O)

/*
 * Filter for lab observations
 */
define function LabResults(observations List<Observation>):
  observations O where IsLabResult(O)

// ============================================================
// OBSERVATION FILTERING BY CODE
// ============================================================

/*
 * Get observation code
 */
define function ObservationCode(observation Observation):
  observation.code

/*
 * Filter observations by code
 */
define function ObservationsByCode(observations List<Observation>, code Code):
  observations O where O.code ~ code

/*
 * Filter observations by LOINC code
 */
define function ObservationsByLOINC(observations List<Observation>, loincCode String):
  observations O
  where exists (O.code.coding C where C.system.value = 'http://loinc.org' and C.code.value = loincCode)

/*
 * Filter observations by ValueSet
 */
define function ObservationsInValueSet(observations List<Observation>, valueSet ValueSet):
  observations O where O.code in valueSet

/*
 * Filter observations during period
 */
define function ObservationsDuringPeriod(observations List<Observation>, period Interval<DateTime>):
  observations O where ObservationDateTime(O) during period

/*
 * Filter observations during date period
 */
define function ObservationsDuringDatePeriod(observations List<Observation>, datePeriod Interval<Date>):
  observations O where ObservationDate(O) during datePeriod

/*
 * Filter for valid observations by code during period
 */
define function ValidObservationsByCodeDuringPeriod(
  observations List<Observation>,
  code Code,
  period Interval<DateTime>
):
  ValidObservations(
    ObservationsDuringPeriod(
      ObservationsByCode(observations, code),
      period
    )
  )

// ============================================================
// MOST RECENT / FIRST OBSERVATION HELPERS
// ============================================================

/*
 * Get most recent observation from a list
 */
define function MostRecentObservation(observations List<Observation>):
  Last(
    observations O
    sort by ObservationDateTime(O)
  )

/*
 * Get first observation from a list
 */
define function FirstObservation(observations List<Observation>):
  First(
    observations O
    sort by ObservationDateTime(O)
  )

/*
 * Get most recent observation by code
 */
define function MostRecentObservationByCode(observations List<Observation>, code Code):
  MostRecentObservation(
    ValidObservations(ObservationsByCode(observations, code))
  )

/*
 * Get most recent observation value (Quantity)
 */
define function MostRecentObservationValue(observations List<Observation>):
  ObservationQuantityValue(MostRecentObservation(ValidObservations(observations)))

/*
 * Get most recent numeric value by code
 */
define function MostRecentNumericValueByCode(observations List<Observation>, code Code):
  ObservationNumericValue(MostRecentObservationByCode(observations, code))

/*
 * Sort observations by date (ascending)
 */
define function SortObservationsByDate(observations List<Observation>):
  observations O sort by ObservationDateTime(O)

/*
 * Sort observations by date (descending)
 */
define function SortObservationsByDateDesc(observations List<Observation>):
  observations O sort by ObservationDateTime(O) desc

// ============================================================
// INTERPRETATION HELPERS
// ============================================================

/*
 * Get observation interpretation
 */
define function ObservationInterpretation(observation Observation):
  First(observation.interpretation)

/*
 * Check if observation is abnormal
 */
define function IsAbnormal(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value in { 'A', 'AA', 'H', 'HH', 'L', 'LL' })
  )

/*
 * Check if observation is high
 */
define function IsHigh(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value in { 'H', 'HH' })
  )

/*
 * Check if observation is low
 */
define function IsLow(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value in { 'L', 'LL' })
  )

/*
 * Check if observation is critical (critically high or low)
 */
define function IsCritical(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value in { 'HH', 'LL' })
  )

/*
 * Check if observation is positive
 */
define function IsPositive(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value = 'POS')
  )

/*
 * Check if observation is negative
 */
define function IsNegative(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value = 'NEG')
  )

/*
 * Check if observation is normal
 */
define function IsNormal(observation Observation):
  exists (
    observation.interpretation I
    where exists (I.coding C where C.code.value = 'N')
  )

/*
 * Filter for abnormal observations
 */
define function AbnormalObservations(observations List<Observation>):
  observations O where IsAbnormal(O)

/*
 * Filter for critical observations
 */
define function CriticalObservations(observations List<Observation>):
  observations O where IsCritical(O)

// ============================================================
// REFERENCE RANGE HELPERS
// ============================================================

/*
 * Get the reference range from observation
 */
define function ObservationReferenceRange(observation Observation):
  First(observation.referenceRange)

/*
 * Get reference range low
 */
define function ReferenceRangeLow(observation Observation):
  if exists (observation.referenceRange) then
    FHIRHelpers.ToQuantity(First(observation.referenceRange).low)
  else null

/*
 * Get reference range high
 */
define function ReferenceRangeHigh(observation Observation):
  if exists (observation.referenceRange) then
    FHIRHelpers.ToQuantity(First(observation.referenceRange).high)
  else null

/*
 * Check if observation value is within reference range
 * Note: Inlined expressions to avoid CQL let syntax issues
 */
define function IsWithinReferenceRange(observation Observation):
  if ObservationNumericValue(observation) is not null
     and ((ReferenceRangeLow(observation)).value is not null or (ReferenceRangeHigh(observation)).value is not null) then
    ((ReferenceRangeLow(observation)).value is null or ObservationNumericValue(observation) >= (ReferenceRangeLow(observation)).value)
    and ((ReferenceRangeHigh(observation)).value is null or ObservationNumericValue(observation) <= (ReferenceRangeHigh(observation)).value)
  else null

/*
 * Check if observation value is above reference range
 */
define function IsAboveReferenceRange(observation Observation):
  if ObservationNumericValue(observation) is not null and (ReferenceRangeHigh(observation)).value is not null then
    ObservationNumericValue(observation) > (ReferenceRangeHigh(observation)).value
  else null

/*
 * Check if observation value is below reference range
 */
define function IsBelowReferenceRange(observation Observation):
  if ObservationNumericValue(observation) is not null and (ReferenceRangeLow(observation)).value is not null then
    ObservationNumericValue(observation) < (ReferenceRangeLow(observation)).value
  else null

// ============================================================
// THRESHOLD COMPARISON HELPERS
// ============================================================

/*
 * Check if observation value is greater than threshold
 */
define function ValueGreaterThan(observation Observation, threshold Decimal):
  ObservationNumericValue(observation) > threshold

/*
 * Check if observation value is greater than or equal to threshold
 */
define function ValueGreaterOrEqual(observation Observation, threshold Decimal):
  ObservationNumericValue(observation) >= threshold

/*
 * Check if observation value is less than threshold
 */
define function ValueLessThan(observation Observation, threshold Decimal):
  ObservationNumericValue(observation) < threshold

/*
 * Check if observation value is less than or equal to threshold
 */
define function ValueLessOrEqual(observation Observation, threshold Decimal):
  ObservationNumericValue(observation) <= threshold

/*
 * Check if observation value is between thresholds (inclusive)
 */
define function ValueBetween(observation Observation, lowThreshold Decimal, highThreshold Decimal):
  ObservationNumericValue(observation) >= lowThreshold
  and ObservationNumericValue(observation) <= highThreshold

/*
 * Filter observations with values above threshold
 */
define function ObservationsAboveThreshold(observations List<Observation>, threshold Decimal):
  observations O where ValueGreaterThan(O, threshold)

/*
 * Filter observations with values below threshold
 */
define function ObservationsBelowThreshold(observations List<Observation>, threshold Decimal):
  observations O where ValueLessThan(O, threshold)

// ============================================================
// TRENDING AND CHANGE ANALYSIS
// ============================================================

/*
 * Get the two most recent observations for trending
 */
define function LastTwoObservations(observations List<Observation>):
  Take(SortObservationsByDateDesc(ValidObservations(observations)), 2)

/*
 * Calculate change between most recent and previous observation
 * Positive = increased, Negative = decreased
 * Note: Uses inline expressions to avoid CQL let syntax issues
 */
define function ValueChange(observations List<Observation>):
  if Count(LastTwoObservations(observations)) = 2 then
    ObservationNumericValue(First(LastTwoObservations(observations)))
    - ObservationNumericValue(Last(LastTwoObservations(observations)))
  else null

/*
 * Calculate percent change between observations
 * Note: Uses inline expressions to avoid CQL let syntax issues
 */
define function PercentChange(observations List<Observation>):
  if Count(LastTwoObservations(observations)) = 2
     and ObservationNumericValue(Last(LastTwoObservations(observations))) != 0 then
    ((ObservationNumericValue(First(LastTwoObservations(observations)))
      - ObservationNumericValue(Last(LastTwoObservations(observations))))
     / ObservationNumericValue(Last(LastTwoObservations(observations)))) * 100
  else null

/*
 * Check if value is increasing (positive trend)
 */
define function IsIncreasing(observations List<Observation>):
  ValueChange(observations) > 0

/*
 * Check if value is decreasing (negative trend)
 */
define function IsDecreasing(observations List<Observation>):
  ValueChange(observations) < 0

/*
 * Check if value is stable (change within threshold %)
 */
define function IsStable(observations List<Observation>, percentThreshold Decimal):
  Abs(PercentChange(observations)) <= percentThreshold

/*
 * Get maximum value from observation list
 */
define function MaxObservationValue(observations List<Observation>):
  Max(
    from ValidObservations(observations) O
    return ObservationNumericValue(O)
  )

/*
 * Get minimum value from observation list
 */
define function MinObservationValue(observations List<Observation>):
  Min(
    from ValidObservations(observations) O
    return ObservationNumericValue(O)
  )

/*
 * Get average value from observation list
 */
define function AvgObservationValue(observations List<Observation>):
  Avg(
    from ValidObservations(observations) O
    return ObservationNumericValue(O)
  )

/*
 * Calculate value range (max - min)
 */
define function ValueRange(observations List<Observation>):
  MaxObservationValue(observations) - MinObservationValue(observations)

// ============================================================
// COMPONENT HELPERS (For multi-component observations like BP)
// ============================================================

/*
 * Get component by code
 */
define function GetComponent(observation Observation, componentCode Code):
  First(
    observation.component C
    where C.code ~ componentCode
  )

/*
 * Get component value as Quantity
 */
define function ComponentQuantityValue(observation Observation, componentCode Code):
  let component: GetComponent(observation, componentCode)
  return
    if component.value is FHIR.Quantity then
      FHIRHelpers.ToQuantity(component.value as FHIR.Quantity)
    else null

/*
 * Get component numeric value
 */
define function ComponentNumericValue(observation Observation, componentCode Code):
  let component: GetComponent(observation, componentCode)
  return
    if component.value is FHIR.Quantity then
      (component.value as FHIR.Quantity).value.value
    else null

// ============================================================
// END OF TIER-1 OBSERVATION HELPERS
// ============================================================
//
// NOTE: Named clinical codes (HbA1c, BP, eGFR, etc.) and their
// convenience functions have been moved to Tier-2:
//   - CommonClinicalCodes.cql (code definitions)
//   - ClinicalObservationHelpers.cql (convenience functions)
//
// This maintains the Grammar Lock principle where Tier 0-1
// contains only grammar (accepts codes as parameters), while
// Tier 2+ contains clinical meaning (named codes).
// ============================================================
