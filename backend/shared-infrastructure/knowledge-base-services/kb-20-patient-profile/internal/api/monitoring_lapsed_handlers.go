package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// MonitoringLapsedEntry describes a patient who was actively
// monitoring home BP and stopped. Phase 9 P9-B.
type MonitoringLapsedEntry struct {
	PatientID              string    `json:"patient_id"`
	LastHomeBPReadingAt    time.Time `json:"last_home_bp_reading_at"`
	DaysSinceLastReading   int       `json:"days_since_last_reading"`
	ReadingsInPrior28Days  int       `json:"readings_in_prior_28_days"`
}

// listMonitoringLapsedPatients returns patients who had >=7 home BP
// readings in the 28-day window ending 14 days ago AND 0 readings in
// the last 14 days. This set-based query runs once per weekly batch
// and returns the full lapsed population in a single HTTP call.
//
// Definition of "lapsed":
//   - The patient was ACTIVELY monitoring (>=7 readings in 28 days
//     proves a pattern, not a one-off)
//   - The patient STOPPED (0 readings in the last 14 days proves
//     the pattern broke)
//
// Phase 9 P9-B.
func (s *Server) listMonitoringLapsedPatients(c *gin.Context) {
	now := time.Now().UTC()
	recentCutoff := now.AddDate(0, 0, -14)          // no readings in last 14 days
	priorWindowStart := now.AddDate(0, 0, -14-28)    // 28-day window ending 14 days ago
	priorWindowEnd := now.AddDate(0, 0, -14)
	minReadingsInPriorWindow := 7

	// Step 1: find patients with 0 SBP readings in the last 14 days
	// Step 2: among those, find patients with >=7 SBP readings in
	//         the 28-day window ending 14 days ago
	// Implemented as a subquery to keep it to one round-trip.
	//
	// Note: uses "SBP" as the home BP lab type. Home BP readings are
	// distinguished from clinic readings by the "HOME_BP" source tag,
	// but the Source field is not always populated consistently across
	// FHIR sync origins. For P9-B we use ANY SBP reading as a proxy
	// for home monitoring activity — a future refinement can filter
	// by Source="HOME_BP" when the FHIR sync populates it reliably.

	type lapsedRow struct {
		PatientID     string
		LastReadingAt time.Time
		PriorCount    int
	}

	var rows []lapsedRow
	err := s.db.DB.Raw(`
		SELECT
			le.patient_id,
			MAX(le.measured_at) AS last_reading_at,
			(SELECT COUNT(*) FROM lab_entries le2
			 WHERE le2.patient_id = le.patient_id
			   AND le2.lab_type = 'SBP'
			   AND le2.validation_status = 'ACCEPTED'
			   AND le2.measured_at >= ?
			   AND le2.measured_at < ?
			) AS prior_count
		FROM lab_entries le
		WHERE le.lab_type = 'SBP'
		  AND le.validation_status = 'ACCEPTED'
		GROUP BY le.patient_id
		HAVING MAX(le.measured_at) < ?
	`, priorWindowStart, priorWindowEnd, recentCutoff).Scan(&rows).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to query monitoring-lapsed patients: " + err.Error(),
		})
		return
	}

	entries := make([]MonitoringLapsedEntry, 0)
	for _, row := range rows {
		if row.PriorCount < minReadingsInPriorWindow {
			continue // not actively monitoring in the prior window
		}
		entries = append(entries, MonitoringLapsedEntry{
			PatientID:             row.PatientID,
			LastHomeBPReadingAt:   row.LastReadingAt,
			DaysSinceLastReading:  int(now.Sub(row.LastReadingAt).Hours() / 24),
			ReadingsInPrior28Days: row.PriorCount,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": entries})
}
