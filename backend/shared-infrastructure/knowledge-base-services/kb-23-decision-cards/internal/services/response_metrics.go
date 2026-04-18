package services

import (
	"math"
	"sort"

	"kb-23-decision-cards/internal/models"
)

// ResponseMetricsService computes aggregate response metrics from lifecycle data.
// It is stateless — all methods are pure functions operating on slices.
type ResponseMetricsService struct{}

// NewResponseMetricsService creates a new ResponseMetricsService.
func NewResponseMetricsService() *ResponseMetricsService {
	return &ResponseMetricsService{}
}

// ComputeClinicianMetrics computes rolling-window metrics from lifecycle data
// for a single clinician. windowDays is stored in the result but filtering
// by date window is the caller's responsibility.
func (s *ResponseMetricsService) ComputeClinicianMetrics(
	lifecycles []models.DetectionLifecycle,
	clinicianID string,
	windowDays int,
) models.ClinicianResponseMetrics {
	// Filter to this clinician
	var filtered []models.DetectionLifecycle
	for _, lc := range lifecycles {
		if lc.AssignedClinicianID == clinicianID {
			filtered = append(filtered, lc)
		}
	}

	total := len(filtered)
	result := models.ClinicianResponseMetrics{
		ClinicianID:     clinicianID,
		WindowDays:      windowDays,
		TotalDetections: total,
	}
	if total == 0 {
		return result
	}

	// Collect latency values
	var deliveryVals, ackVals, actionVals []int64
	var actionedCount, resolvedCount int

	for _, lc := range filtered {
		if lc.DeliveryLatencyMs != nil {
			deliveryVals = append(deliveryVals, *lc.DeliveryLatencyMs)
		}
		if lc.AcknowledgmentLatencyMs != nil {
			ackVals = append(ackVals, *lc.AcknowledgmentLatencyMs)
		}
		if lc.ActionLatencyMs != nil {
			actionVals = append(actionVals, *lc.ActionLatencyMs)
		}
		if lc.ActionedAt != nil {
			actionedCount++
		}
		if lc.ResolvedAt != nil {
			resolvedCount++
		}
	}

	result.MedianDeliveryMs = medianInt64(deliveryVals)
	result.MedianAcknowledgmentMs = medianInt64(ackVals)
	result.MedianActionMs = medianInt64(actionVals)
	result.ActionCompletionRate = float64(actionedCount) / float64(total)
	if actionedCount > 0 {
		result.OutcomeRate = float64(resolvedCount) / float64(actionedCount)
	}

	return result
}

// ComputeSystemMetrics computes system-level aggregate metrics across all
// lifecycles in the provided slice.
func (s *ResponseMetricsService) ComputeSystemMetrics(
	lifecycles []models.DetectionLifecycle,
	windowDays int,
) models.SystemResponseMetrics {
	total := len(lifecycles)
	result := models.SystemResponseMetrics{
		WindowDays:      windowDays,
		TotalDetections: total,
		ByTier:          make(map[string]models.TierMetrics),
	}
	if total == 0 {
		return result
	}

	// T0→T2 (DetectedAt → AcknowledgedAt) and T0→T3 (DetectedAt → ActionedAt)
	var t0t2Vals, t0t3Vals []int64
	var actionedCount, resolvedCount, timedOutCount int

	// Per-tier collection
	type tierAccum struct {
		ackVals    []int64
		actionVals []int64
		actioned   int
		total      int
	}
	tierMap := make(map[string]*tierAccum)

	for _, lc := range lifecycles {
		// System-level latencies
		if lc.AcknowledgedAt != nil {
			latency := lc.AcknowledgedAt.Sub(lc.DetectedAt).Milliseconds()
			t0t2Vals = append(t0t2Vals, latency)
		}
		if lc.ActionedAt != nil {
			latency := lc.ActionedAt.Sub(lc.DetectedAt).Milliseconds()
			t0t3Vals = append(t0t3Vals, latency)
			actionedCount++
		}
		if lc.ResolvedAt != nil {
			resolvedCount++
		}
		if lc.CurrentState == string(models.LifecycleTimedOut) {
			timedOutCount++
		}

		// Per-tier accumulation
		tier := lc.TierAtDetection
		if tier == "" {
			tier = "UNKNOWN"
		}
		ta, ok := tierMap[tier]
		if !ok {
			ta = &tierAccum{}
			tierMap[tier] = ta
		}
		ta.total++
		if lc.AcknowledgmentLatencyMs != nil {
			ta.ackVals = append(ta.ackVals, *lc.AcknowledgmentLatencyMs)
		}
		if lc.ActionLatencyMs != nil {
			ta.actionVals = append(ta.actionVals, *lc.ActionLatencyMs)
		}
		if lc.ActionedAt != nil {
			ta.actioned++
		}
	}

	result.MedianT0toT2Ms = medianInt64(t0t2Vals)
	result.MedianT0toT3Ms = medianInt64(t0t3Vals)
	result.ActionCompletionRate = float64(actionedCount) / float64(total)
	if actionedCount > 0 {
		result.OutcomeRate = float64(resolvedCount) / float64(actionedCount)
	}
	result.TimeoutRate = roundTo3(float64(timedOutCount) / float64(total))

	// Build per-tier metrics
	for tier, ta := range tierMap {
		tm := models.TierMetrics{
			Count:      ta.total,
			MedianAckMs: medianInt64(ta.ackVals),
			MedianActionMs: medianInt64(ta.actionVals),
		}
		if ta.total > 0 {
			tm.ActionCompletionRate = float64(ta.actioned) / float64(ta.total)
		}
		result.ByTier[tier] = tm
	}

	return result
}

// timelyThresholdMs returns the "timely" acknowledgment threshold per tier.
func timelyThresholdMs(tier string) int64 {
	switch tier {
	case "CRITICAL":
		return 15 * 60 * 1000 // 15 min
	case "URGENT":
		return 60 * 60 * 1000 // 1 hour
	case "STANDARD":
		return 4 * 60 * 60 * 1000 // 4 hours
	default:
		return 24 * 60 * 60 * 1000 // 24 hours
	}
}

// ComputePilotMetrics computes HCF CHF pilot-specific KPIs.
func (s *ResponseMetricsService) ComputePilotMetrics(
	lifecycles []models.DetectionLifecycle,
) models.PilotMetrics {
	result := models.PilotMetrics{
		TotalDetections: len(lifecycles),
	}
	if len(lifecycles) == 0 {
		return result
	}

	var actionLatencies []int64
	timelyPatients := make(map[string]bool)
	untimelyPatients := make(map[string]bool)

	for _, lc := range lifecycles {
		// Acknowledged in time?
		if lc.AcknowledgmentLatencyMs != nil {
			threshold := timelyThresholdMs(lc.TierAtDetection)
			if *lc.AcknowledgmentLatencyMs <= threshold {
				result.DetectionsAcknowledgedInTime++
			}
		}

		// Actioned?
		if lc.ActionedAt != nil {
			result.DetectionsWithAction++
		}

		// Action type counting
		switch lc.ActionType {
		case "CALL_PATIENT", "TELECONSULT":
			result.OutreachCalls++
		case "MEDICATION_REVIEW", "PRESCRIPTION_REVIEW":
			result.MedicationChanges++
		case "SCHEDULE_APPOINTMENT", "SCHEDULE_CLINIC":
			result.AppointmentsScheduled++
		}

		// Collect action latencies for median
		if lc.ActionLatencyMs != nil {
			actionLatencies = append(actionLatencies, *lc.ActionLatencyMs)
		}

		// Timely action tracking per patient
		if lc.ActionedAt != nil && lc.ActionLatencyMs != nil {
			threshold := timelyThresholdMs(lc.TierAtDetection)
			if *lc.ActionLatencyMs <= threshold*2 { // action within 2x ack threshold
				timelyPatients[lc.PatientID] = true
			} else {
				if !timelyPatients[lc.PatientID] {
					untimelyPatients[lc.PatientID] = true
				}
			}
		}
	}

	// Median detection-to-action in hours
	if med := medianInt64(actionLatencies); med != nil {
		result.MedianDetectionToActionHrs = roundTo3(float64(*med) / 3600000.0)
	}

	result.PatientsWithTimelyAction = len(timelyPatients)
	result.PatientsWithoutTimelyAction = len(untimelyPatients)

	return result
}

// medianInt64 computes the median of a slice of int64 values.
// Returns nil for empty slices. For even-length slices, returns the
// average of the two middle values (integer division).
func medianInt64(values []int64) *int64 {
	if len(values) == 0 {
		return nil
	}
	sorted := make([]int64, len(values))
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	var med int64
	if n%2 == 1 {
		med = sorted[n/2]
	} else {
		med = (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return &med
}

// roundTo3 rounds a float to 3 decimal places.
func roundTo3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
