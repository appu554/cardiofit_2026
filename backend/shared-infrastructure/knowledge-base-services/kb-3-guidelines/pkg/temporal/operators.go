// Package temporal provides temporal reasoning capabilities for clinical pathways
// Implements Allen's Interval Algebra for CQL-compatible temporal operations
package temporal

import (
	"time"
)

// TemporalOperator represents Allen's Interval Algebra operators
type TemporalOperator string

const (
	// OpBefore: Target ends before reference starts
	OpBefore TemporalOperator = "before"
	// OpAfter: Target starts after reference ends
	OpAfter TemporalOperator = "after"
	// OpSameAs: Target and reference are equivalent
	OpSameAs TemporalOperator = "same_as"
	// OpMeets: Target ends exactly when reference starts
	OpMeets TemporalOperator = "meets"
	// OpOverlaps: Intervals share some time period
	OpOverlaps TemporalOperator = "overlaps"
	// OpWithin: Target is within offset of reference
	OpWithin TemporalOperator = "within"
	// OpWithinBefore: Target is within offset before reference
	OpWithinBefore TemporalOperator = "within_before"
	// OpWithinAfter: Target is within offset after reference
	OpWithinAfter TemporalOperator = "within_after"
	// OpDuring: Target interval contained within reference
	OpDuring TemporalOperator = "during"
	// OpContains: Target interval contains reference
	OpContains TemporalOperator = "contains"
	// OpStarts: Both start at same time
	OpStarts TemporalOperator = "starts"
	// OpEnds: Both end at same time
	OpEnds TemporalOperator = "ends"
	// OpEquals: Intervals are identical
	OpEquals TemporalOperator = "equals"
)

// AllOperators returns all available temporal operators
func AllOperators() []TemporalOperator {
	return []TemporalOperator{
		OpBefore, OpAfter, OpSameAs, OpMeets, OpOverlaps,
		OpWithin, OpWithinBefore, OpWithinAfter,
		OpDuring, OpContains, OpStarts, OpEnds, OpEquals,
	}
}

// Interval represents a time interval with start and end points
type Interval struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// NewInterval creates a new interval with validation
func NewInterval(start, end time.Time) Interval {
	if end.Before(start) {
		// Swap if end is before start
		return Interval{Start: end, End: start}
	}
	return Interval{Start: start, End: end}
}

// NewPointInterval creates an interval representing a single point in time
func NewPointInterval(t time.Time) Interval {
	return Interval{Start: t, End: t}
}

// Duration returns the duration of the interval
func (i Interval) Duration() time.Duration {
	return i.End.Sub(i.Start)
}

// IsPoint returns true if the interval represents a single point
func (i Interval) IsPoint() bool {
	return i.Start.Equal(i.End)
}

// Contains checks if a time point is within the interval
func (i Interval) ContainsTime(t time.Time) bool {
	return !t.Before(i.Start) && !t.After(i.End)
}

// Before: Target ends before reference starts (no overlap)
// [target]     [reference]
func (i Interval) Before(other Interval) bool {
	return i.End.Before(other.Start)
}

// After: Target starts after reference ends (no overlap)
// [reference]     [target]
func (i Interval) After(other Interval) bool {
	return i.Start.After(other.End)
}

// Meets: Target ends exactly when reference starts
// [target][reference]
func (i Interval) Meets(other Interval) bool {
	return i.End.Equal(other.Start)
}

// MetBy: Target starts exactly when reference ends (inverse of Meets)
// [reference][target]
func (i Interval) MetBy(other Interval) bool {
	return i.Start.Equal(other.End)
}

// Overlaps: Target starts before reference but ends during reference
// [target---]
//      [---reference]
func (i Interval) Overlaps(other Interval) bool {
	return i.Start.Before(other.Start) &&
		i.End.After(other.Start) &&
		i.End.Before(other.End)
}

// OverlappedBy: Target is overlapped by reference (inverse of Overlaps)
func (i Interval) OverlappedBy(other Interval) bool {
	return other.Overlaps(i)
}

// During: Target is completely contained within reference
//      [target]
// [----reference----]
func (i Interval) During(other Interval) bool {
	return i.Start.After(other.Start) && i.End.Before(other.End)
}

// Contains: Target completely contains reference (inverse of During)
// [----target----]
//      [reference]
func (i Interval) ContainsInterval(other Interval) bool {
	return i.Start.Before(other.Start) && i.End.After(other.End)
}

// Starts: Both intervals start at the same time, target ends first
// [target---]
// [------reference]
func (i Interval) Starts(other Interval) bool {
	return i.Start.Equal(other.Start) && i.End.Before(other.End)
}

// StartedBy: Both intervals start at the same time, reference ends first
func (i Interval) StartedBy(other Interval) bool {
	return other.Starts(i)
}

// Ends: Both intervals end at the same time, target starts later
//      [---target]
// [reference----]
func (i Interval) Ends(other Interval) bool {
	return i.Start.After(other.Start) && i.End.Equal(other.End)
}

// EndedBy: Both intervals end at the same time, reference starts later
func (i Interval) EndedBy(other Interval) bool {
	return other.Ends(i)
}

// Equals: Intervals are identical (same start and end)
func (i Interval) Equals(other Interval) bool {
	return i.Start.Equal(other.Start) && i.End.Equal(other.End)
}

// Within: Target is within a given offset of reference
func (i Interval) Within(other Interval, offset time.Duration) bool {
	expandedStart := other.Start.Add(-offset)
	expandedEnd := other.End.Add(offset)
	return !i.Start.Before(expandedStart) && !i.End.After(expandedEnd)
}

// WithinBefore: Target ends within offset before reference starts
// Clinical example: "tPA must be given within 60 minutes of door time"
func (i Interval) WithinBefore(referenceStart time.Time, offset time.Duration) bool {
	return i.End.Before(referenceStart) &&
		i.End.After(referenceStart.Add(-offset))
}

// WithinAfter: Target starts within offset after reference ends
// Clinical example: "Follow-up within 7 days of discharge"
func (i Interval) WithinAfter(referenceEnd time.Time, offset time.Duration) bool {
	return i.Start.After(referenceEnd) &&
		i.Start.Before(referenceEnd.Add(offset))
}

// HasOverlap returns true if the intervals share any common time
func (i Interval) HasOverlap(other Interval) bool {
	return i.Start.Before(other.End) && i.End.After(other.Start)
}

// EvaluateTemporalRelation evaluates a temporal relationship between intervals
func EvaluateTemporalRelation(target, reference Interval, operator TemporalOperator, offset ...time.Duration) bool {
	switch operator {
	case OpBefore:
		return target.Before(reference)
	case OpAfter:
		return target.After(reference)
	case OpMeets:
		return target.Meets(reference)
	case OpOverlaps:
		return target.HasOverlap(reference)
	case OpDuring:
		return target.During(reference)
	case OpContains:
		return target.ContainsInterval(reference)
	case OpStarts:
		return target.Starts(reference) || target.Start.Equal(reference.Start)
	case OpEnds:
		return target.Ends(reference) || target.End.Equal(reference.End)
	case OpEquals, OpSameAs:
		return target.Equals(reference)
	case OpWithin:
		if len(offset) > 0 {
			return target.Within(reference, offset[0])
		}
		return target.During(reference) || target.Equals(reference)
	case OpWithinBefore:
		if len(offset) > 0 {
			return target.WithinBefore(reference.Start, offset[0])
		}
		return false
	case OpWithinAfter:
		if len(offset) > 0 {
			return target.WithinAfter(reference.End, offset[0])
		}
		return false
	default:
		return false
	}
}

// TemporalRelationRequest for API
type TemporalRelationRequest struct {
	TargetStart    time.Time        `json:"target_start" binding:"required"`
	TargetEnd      time.Time        `json:"target_end" binding:"required"`
	ReferenceStart time.Time        `json:"reference_start" binding:"required"`
	ReferenceEnd   time.Time        `json:"reference_end" binding:"required"`
	Operator       TemporalOperator `json:"operator" binding:"required"`
	Offset         *string          `json:"offset,omitempty"` // Duration string like "1h", "30m"
}

// TemporalRelationResponse for API
type TemporalRelationResponse struct {
	Result   bool             `json:"result"`
	Operator TemporalOperator `json:"operator"`
	Target   Interval         `json:"target"`
	Reference Interval        `json:"reference"`
	Offset   *time.Duration   `json:"offset,omitempty"`
}

// NextOccurrenceRequest for API
type NextOccurrenceRequest struct {
	FromTime   time.Time `json:"from_time" binding:"required"`
	Recurrence struct {
		Frequency string `json:"frequency" binding:"required"` // daily, weekly, monthly, yearly
		Interval  int    `json:"interval" binding:"required"`
	} `json:"recurrence" binding:"required"`
}

// NextOccurrenceResponse for API
type NextOccurrenceResponse struct {
	NextOccurrence time.Time `json:"next_occurrence"`
	FromTime       time.Time `json:"from_time"`
	Frequency      string    `json:"frequency"`
	Interval       int       `json:"interval"`
}

// ValidateConstraintRequest for API
type ValidateConstraintRequest struct {
	ActionTime     time.Time     `json:"action_time" binding:"required"`
	ReferenceTime  time.Time     `json:"reference_time" binding:"required"`
	Deadline       time.Duration `json:"deadline" binding:"required"`
	GracePeriod    time.Duration `json:"grace_period,omitempty"`
}

// ValidateConstraintResponse for API
type ValidateConstraintResponse struct {
	Valid         bool           `json:"valid"`
	Status        string         `json:"status"` // met, approaching, overdue, missed
	TimeRemaining *time.Duration `json:"time_remaining,omitempty"`
	TimeOverdue   *time.Duration `json:"time_overdue,omitempty"`
}

// ValidateConstraintTiming validates if an action meets its time constraint
func ValidateConstraintTiming(actionTime, referenceTime time.Time, deadline, gracePeriod time.Duration) ValidateConstraintResponse {
	deadlineTime := referenceTime.Add(deadline)
	graceEndTime := deadlineTime.Add(gracePeriod)
	now := time.Now()

	resp := ValidateConstraintResponse{}

	// If action is completed
	if !actionTime.IsZero() {
		if actionTime.Before(deadlineTime) || actionTime.Equal(deadlineTime) {
			resp.Valid = true
			resp.Status = "met"
		} else if actionTime.Before(graceEndTime) || actionTime.Equal(graceEndTime) {
			resp.Valid = true
			resp.Status = "met_within_grace"
		} else {
			resp.Valid = false
			resp.Status = "missed"
			overdue := actionTime.Sub(deadlineTime)
			resp.TimeOverdue = &overdue
		}
		return resp
	}

	// Action not yet completed - evaluate current status
	if now.Before(deadlineTime) {
		remaining := deadlineTime.Sub(now)
		resp.Status = "pending"
		resp.TimeRemaining = &remaining

		// Check if approaching (within 20% of deadline)
		threshold := deadline / 5
		if remaining < threshold {
			resp.Status = "approaching"
		}
	} else if now.Before(graceEndTime) {
		overdue := now.Sub(deadlineTime)
		resp.Status = "overdue"
		resp.TimeOverdue = &overdue
	} else {
		overdue := now.Sub(deadlineTime)
		resp.Status = "missed"
		resp.TimeOverdue = &overdue
	}

	return resp
}
