/*
 * IntervalHelpers CQL Library
 * ============================================================
 * Tier 1 Primitive - Interval Operations for Clinical Logic
 *
 * Purpose: Provides reusable interval operations for temporal
 *          clinical calculations (encounters, medication periods,
 *          observation windows, etc.)
 *
 * Version: 1.0.0
 * FHIR Version: R4 (4.0.1)
 * CQL Version: 1.5
 *
 * Dependencies:
 *   - FHIRHelpers (Tier 0)
 *
 * Clinical Use Cases:
 *   - Encounter overlap detection
 *   - Medication administration windows
 *   - Lab result trending periods
 *   - Guideline timing requirements
 *   - Quality measure reporting periods
 *
 * Governance: Changes require Tier-1 review (Clinical + Technical)
 * ============================================================
 */
library IntervalHelpers version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// INTERVAL CREATION HELPERS
// ============================================================

/*
 * Create a closed interval from two dates
 * Common pattern: [start, end] - includes both endpoints
 */
define function ClosedDateInterval(startDate Date, endDate Date):
  Interval[startDate, endDate]

/*
 * Create an open interval from two dates
 * Common pattern: (start, end) - excludes both endpoints
 */
define function OpenDateInterval(startDate Date, endDate Date):
  Interval(startDate, endDate)

/*
 * Create a left-open, right-closed interval
 * Pattern: (start, end] - excludes start, includes end
 */
define function LeftOpenDateInterval(startDate Date, endDate Date):
  Interval(startDate, endDate]

/*
 * Create a left-closed, right-open interval
 * Pattern: [start, end) - includes start, excludes end
 * Used for "up to but not including" semantics
 */
define function RightOpenDateInterval(startDate Date, endDate Date):
  Interval[startDate, endDate)

/*
 * Create a closed interval from two DateTimes
 */
define function ClosedDateTimeInterval(startDateTime DateTime, endDateTime DateTime):
  Interval[startDateTime, endDateTime]

/*
 * Create interval starting from a date extending forward N days
 */
define function DaysFromDate(startDate Date, numberOfDays Integer):
  Interval[startDate, startDate + System.Quantity { value: numberOfDays, unit: 'day' })

/*
 * Create interval starting from a date extending forward N months
 */
define function MonthsFromDate(startDate Date, numberOfMonths Integer):
  Interval[startDate, startDate + System.Quantity { value: numberOfMonths, unit: 'month' })

/*
 * Create interval starting from a date extending forward N years
 */
define function YearsFromDate(startDate Date, numberOfYears Integer):
  Interval[startDate, startDate + System.Quantity { value: numberOfYears, unit: 'year' })

/*
 * Create interval looking back N days from a date
 */
define function DaysBeforeDate(endDate Date, numberOfDays Integer):
  Interval[endDate - System.Quantity { value: numberOfDays, unit: 'day' }, endDate]

/*
 * Create interval looking back N months from a date
 */
define function MonthsBeforeDate(endDate Date, numberOfMonths Integer):
  Interval[endDate - System.Quantity { value: numberOfMonths, unit: 'month' }, endDate]

/*
 * Create interval looking back N years from a date
 */
define function YearsBeforeDate(endDate Date, numberOfYears Integer):
  Interval[endDate - System.Quantity { value: numberOfYears, unit: 'year' }, endDate]

// ============================================================
// MEASUREMENT PERIOD HELPERS
// ============================================================

/*
 * Standard calendar year interval
 * Used for annual quality measure reporting
 */
define function CalendarYear(theYear Integer):
  Interval[Date(theYear, 1, 1), Date(theYear, 12, 31)]

/*
 * Standard calendar quarter interval
 * quarter: 1-4
 */
define function CalendarQuarter(theYear Integer, theQuarter Integer):
  case theQuarter
    when 1 then Interval[Date(theYear, 1, 1), Date(theYear, 3, 31)]
    when 2 then Interval[Date(theYear, 4, 1), Date(theYear, 6, 30)]
    when 3 then Interval[Date(theYear, 7, 1), Date(theYear, 9, 30)]
    when 4 then Interval[Date(theYear, 10, 1), Date(theYear, 12, 31)]
    else null
  end

/*
 * Standard calendar month interval
 * month: 1-12
 * Note: Calculates last day of month by adding 1 month and subtracting 1 day
 */
define function CalendarMonth(theYear Integer, theMonth Integer):
  Interval[Date(theYear, theMonth, 1), Date(theYear, theMonth, 1) + 1 month - 1 day]

/*
 * Trailing 12-month interval from reference date
 * Commonly used for rolling annual measures
 */
define function TrailingTwelveMonths(referenceDate Date):
  Interval[referenceDate - 12 months, referenceDate]

/*
 * Trailing 90-day interval from reference date
 * Common for medication adherence, recent labs
 */
define function TrailingNinetyDays(referenceDate Date):
  Interval[referenceDate - 90 days, referenceDate]

/*
 * Trailing 30-day interval from reference date
 * Common for recent activity checks
 */
define function TrailingThirtyDays(referenceDate Date):
  Interval[referenceDate - 30 days, referenceDate]

// ============================================================
// INTERVAL DURATION CALCULATIONS
// ============================================================

/*
 * Calculate duration of an interval in days
 * Returns null if interval is null or has no defined bounds
 */
define function DurationInDays(interval Interval<Date>):
  if interval is null then null
  else if start of interval is null or end of interval is null then null
  else difference in days between start of interval and end of interval

/*
 * Calculate duration of an interval in months
 */
define function DurationInMonths(interval Interval<Date>):
  if interval is null then null
  else if start of interval is null or end of interval is null then null
  else difference in months between start of interval and end of interval

/*
 * Calculate duration of an interval in years
 */
define function DurationInYears(interval Interval<Date>):
  if interval is null then null
  else if start of interval is null or end of interval is null then null
  else difference in years between start of interval and end of interval

/*
 * Calculate duration of a DateTime interval in hours
 * Used for encounter duration, procedure timing
 */
define function DurationInHours(interval Interval<DateTime>):
  if interval is null then null
  else if start of interval is null or end of interval is null then null
  else difference in hours between start of interval and end of interval

/*
 * Calculate duration of a DateTime interval in minutes
 * Used for procedure timing, medication administration
 */
define function DurationInMinutes(interval Interval<DateTime>):
  if interval is null then null
  else if start of interval is null or end of interval is null then null
  else difference in minutes between start of interval and end of interval

// ============================================================
// INTERVAL RELATIONSHIP HELPERS
// ============================================================

/*
 * Check if two intervals overlap (have any common time)
 * Returns true if intervals share at least one point in time
 */
define function IntervalsOverlap(interval1 Interval<Date>, interval2 Interval<Date>):
  interval1 overlaps interval2

/*
 * Check if interval1 contains interval2 completely
 * Returns true if all of interval2 is within interval1
 */
define function IntervalContains(container Interval<Date>, contained Interval<Date>):
  container includes contained

/*
 * Check if a date falls within an interval
 */
define function DateInInterval(checkDate Date, interval Interval<Date>):
  checkDate in interval

/*
 * Check if a DateTime falls within an interval
 */
define function DateTimeInInterval(checkDateTime DateTime, interval Interval<DateTime>):
  checkDateTime in interval

/*
 * Check if interval1 meets interval2 (ends exactly when other starts)
 * Used for continuous coverage detection
 */
define function IntervalsMeet(interval1 Interval<Date>, interval2 Interval<Date>):
  interval1 meets interval2

/*
 * Get the overlapping portion of two intervals
 * Returns null if intervals do not overlap
 */
define function GetOverlap(interval1 Interval<Date>, interval2 Interval<Date>):
  interval1 intersect interval2

/*
 * Combine two overlapping or meeting intervals into one
 * Returns null if intervals cannot be combined
 */
define function CombineIntervals(interval1 Interval<Date>, interval2 Interval<Date>):
  if interval1 overlaps interval2 or interval1 meets interval2 or interval2 meets interval1
    then interval1 union interval2
  else null

// ============================================================
// GAP ANALYSIS HELPERS
// ============================================================

/*
 * Calculate the gap in days between two intervals
 * Returns positive if gap exists, 0 if adjacent, negative if overlap
 */
define function GapInDays(interval1 Interval<Date>, interval2 Interval<Date>):
  if end of interval1 < start of interval2 then
    difference in days between end of interval1 and start of interval2
  else if end of interval2 < start of interval1 then
    difference in days between end of interval2 and start of interval1
  else
    0 // Overlapping or adjacent

/*
 * Check if a gap exists between two intervals
 * Gap must be greater than the specified threshold
 */
define function HasGapGreaterThan(interval1 Interval<Date>, interval2 Interval<Date>, thresholdDays Integer):
  GapInDays(interval1, interval2) > thresholdDays

/*
 * Check for continuous coverage between consecutive intervals
 * Returns true if no gaps exceed maxGapDays between first two intervals
 * Note: Full coverage check requires runtime iteration (not expressible in pure CQL)
 */
define function HasContinuousCoverageForTwo(interval1 Interval<Date>, interval2 Interval<Date>, maxGapDays Integer):
  not HasGapGreaterThan(interval1, interval2, maxGapDays)

/*
 * Note: For sorting intervals, use the native CQL sort pattern:
 *   intervals I sort by start of I
 * This function is deprecated - use native sort instead.
 */

// ============================================================
// CLINICAL TIMING HELPERS
// ============================================================

/*
 * Check if an event occurred within N days before a reference date
 * Common for pre-operative requirements, recent lab checks
 */
define function OccurredWithinDaysBefore(eventDate Date, referenceDate Date, numDays Integer):
  eventDate in Interval[referenceDate - System.Quantity { value: numDays, unit: 'day' }, referenceDate]

/*
 * Check if an event occurred within N days after a reference date
 * Common for follow-up requirements
 */
define function OccurredWithinDaysAfter(eventDate Date, referenceDate Date, numDays Integer):
  eventDate in Interval[referenceDate, referenceDate + System.Quantity { value: numDays, unit: 'day' }]

/*
 * Check if an event occurred within a window around a reference date
 * Common for screening intervals with acceptable variance
 */
define function OccurredWithinWindow(eventDate Date, referenceDate Date, numDaysBefore Integer, numDaysAfter Integer):
  eventDate in Interval[referenceDate - System.Quantity { value: numDaysBefore, unit: 'day' }, referenceDate + System.Quantity { value: numDaysAfter, unit: 'day' }]

/*
 * Get age in years at a specific date
 * Uses birth date from Patient context
 */
define function AgeInYearsAt(asOfDate Date):
  if Patient.birthDate is null then null
  else difference in years between FHIRHelpers.ToDate(Patient.birthDate) and asOfDate

/*
 * Get age in months at a specific date
 * Useful for pediatric calculations
 */
define function AgeInMonthsAt(asOfDate Date):
  if Patient.birthDate is null then null
  else difference in months between FHIRHelpers.ToDate(Patient.birthDate) and asOfDate

/*
 * Get age in days at a specific date
 * Useful for neonatal calculations
 */
define function AgeInDaysAt(asOfDate Date):
  if Patient.birthDate is null then null
  else difference in days between FHIRHelpers.ToDate(Patient.birthDate) and asOfDate

// ============================================================
// LOOKBACK PERIOD HELPERS
// ============================================================

/*
 * Standard lookback periods used in clinical guidelines
 */
define function LastOneYear():
  Interval[Today() - 1 year, Today()]

define function LastTwoYears():
  Interval[Today() - 2 years, Today()]

define function LastThreeYears():
  Interval[Today() - 3 years, Today()]

define function LastFiveYears():
  Interval[Today() - 5 years, Today()]

define function LastSixMonths():
  Interval[Today() - 6 months, Today()]

define function LastThreeMonths():
  Interval[Today() - 3 months, Today()]

define function LastThirtyDays():
  Interval[Today() - 30 days, Today()]

define function LastSevenDays():
  Interval[Today() - 7 days, Today()]

define function LastTwentyFourHours():
  Interval[Now() - 24 hours, Now()]

define function LastFortyEightHours():
  Interval[Now() - 48 hours, Now()]

define function LastSeventyTwoHours():
  Interval[Now() - 72 hours, Now()]

// ============================================================
// UTILITY HELPERS
// ============================================================

/*
 * Note: CQL does not support recursive functions or imperative loops.
 * Range generation must be done at runtime or with fixed-size lists.
 *
 * For iteration patterns, use the built-in CQL list comprehension:
 *   from item in list return ...
 *
 * Or use explicit small ranges like:
 *   { 0, 1, 2, 3, 4 } where each <= someLimit
 */

/*
 * Check if all values in a list are true
 */
define function AllTrue(values List<Boolean>):
  not exists (values V where V is false)

/*
 * Check if any value in a list is true
 */
define function AnyTrue(values List<Boolean>):
  exists (values V where V is true)

/*
 * Get the earliest date from a list
 */
define function EarliestDate(dates List<Date>):
  Min(dates)

/*
 * Get the latest date from a list
 */
define function LatestDate(dates List<Date>):
  Max(dates)

/*
 * Coalesce interval - return first non-null interval
 */
define function CoalesceInterval(interval1 Interval<Date>, interval2 Interval<Date>):
  Coalesce(interval1, interval2)
