package services

import (
	"time"

	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// pairingWindow is the maximum time delta allowed between SBP and DBP
// measurements for them to be considered part of the same BP reading.
const pairingWindow = 5 * time.Minute

// BPReading is a paired SBP+DBP measurement from one observation event.
// Source is passed through from LabEntry.Source unchanged — callers
// interpret the string (CLINIC/OFFICE/HOSPITAL for clinic, HOME_CUFF/
// HOME_WRIST for home, empty string for unknown).
type BPReading struct {
	PatientID  string    `json:"patient_id"`
	SBP        float64   `json:"sbp"`
	DBP        float64   `json:"dbp"`
	Source     string    `json:"source"`
	MeasuredAt time.Time `json:"measured_at"`
}

// BPReadingQuery fetches paired SBP+DBP readings from LabEntry rows.
// SBP and DBP are stored as separate LabEntry rows; this service pairs
// them by MeasuredAt (within a 5-minute window) and Source.
//
// Phase 4 limitation: LabEntry does not persist BPMeasurementContext
// (only LabEntry.Source is available at query time). Callers that need
// clinic-vs-home distinction must interpret the Source string. Historical
// data with empty Source is returned as-is; callers decide how to treat it.
type BPReadingQuery struct {
	db *gorm.DB
}

// NewBPReadingQuery constructs a query service.
func NewBPReadingQuery(db *gorm.DB) *BPReadingQuery {
	return &BPReadingQuery{db: db}
}

// FetchSince returns all paired BP readings for a patient since the given
// time. Unpaired readings (SBP without matching DBP within 5 minutes from
// the same source) are dropped. Returns an empty slice (not an error) for
// unknown patients.
func (q *BPReadingQuery) FetchSince(patientID string, since time.Time) ([]BPReading, error) {
	var entries []models.LabEntry
	err := q.db.Where(
		"patient_id = ? AND lab_type IN (?, ?) AND measured_at > ?",
		patientID, models.LabTypeSBP, models.LabTypeDBP, since,
	).Order("measured_at ASC").Find(&entries).Error
	if err != nil {
		return nil, err
	}

	return pairEntries(entries, patientID), nil
}

// pairEntries groups SBP and DBP entries by (Source, MeasuredAt ± pairingWindow).
// Uses a greedy single-pass match: each SBP is paired with the nearest DBP
// from the same source within the window.
func pairEntries(entries []models.LabEntry, patientID string) []BPReading {
	if len(entries) == 0 {
		return []BPReading{}
	}

	var paired []BPReading
	used := make(map[int]bool)

	for i, sbp := range entries {
		if used[i] || sbp.LabType != models.LabTypeSBP {
			continue
		}
		// Find the closest matching DBP within the window from the same source.
		bestJ := -1
		bestDelta := pairingWindow + time.Second
		for j, dbp := range entries {
			if used[j] || j == i || dbp.LabType != models.LabTypeDBP {
				continue
			}
			if dbp.Source != sbp.Source {
				continue
			}
			delta := absDuration(dbp.MeasuredAt.Sub(sbp.MeasuredAt))
			if delta > pairingWindow {
				continue
			}
			if delta < bestDelta {
				bestDelta = delta
				bestJ = j
			}
		}
		if bestJ == -1 {
			continue
		}
		dbp := entries[bestJ]
		sbpVal, _ := sbp.Value.Float64()
		dbpVal, _ := dbp.Value.Float64()
		paired = append(paired, BPReading{
			PatientID:  patientID,
			SBP:        sbpVal,
			DBP:        dbpVal,
			Source:     sbp.Source,
			MeasuredAt: sbp.MeasuredAt,
		})
		used[i] = true
		used[bestJ] = true
	}

	if paired == nil {
		return []BPReading{}
	}
	return paired
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
