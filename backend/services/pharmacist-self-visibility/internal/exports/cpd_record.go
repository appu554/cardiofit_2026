package exports

// VisibilityClass: pharmacist-controlled — platform never submits on pharmacist's behalf.

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ActivityRow represents a single CPD activity retrieved from the data store.
type ActivityRow struct {
	// ID is the stable identifier for this activity record.
	ID string

	// Category is the AHPRA CPD category label
	// (e.g. "educational_activities", "reviewing_performance", "measuring_outcomes").
	Category string

	// Hours is the number of CPD hours credited for this activity.
	Hours float64

	// Confirmed indicates whether the pharmacist has confirmed this activity.
	// Only confirmed activities are included in export totals.
	Confirmed bool
}

// CPDRecord is the output of CPDRecordGenerator.Generate.
// It is a submission-ready snapshot; the platform does not forward or submit it.
type CPDRecord struct {
	// ID uniquely identifies this generated record (UUID v4).
	ID string

	// PharmacistID is the identifier of the pharmacist who requested the export.
	PharmacistID string

	// CycleStart is the AHPRA registration year marking the start of the CPD cycle.
	CycleStart int

	// CycleEnd is the AHPRA registration year marking the end of the CPD cycle.
	CycleEnd int

	// HoursByCategory maps each AHPRA CPD category to the total confirmed hours.
	// Always non-nil; categories with no confirmed hours are omitted.
	HoursByCategory map[string]float64

	// GeneratedAt is the UTC instant at which the record was assembled.
	GeneratedAt time.Time
}

// CPDExportSource is the data-access interface that CPDRecordGenerator depends upon.
type CPDExportSource interface {
	// ActivitiesInCycle returns all CPD activity rows for the pharmacist within
	// the given cycle (cycleStart and cycleEnd are registration years, inclusive).
	ActivitiesInCycle(ctx context.Context, pharmacistID string, cycleStart, cycleEnd int) ([]ActivityRow, error)

	// ReflectionsForActivity returns reflective entry content linked to a given
	// activity. Reflections are POA-class; this interface exposes them only to
	// the pharmacist's own export flow.
	ReflectionsForActivity(ctx context.Context, activityID string) ([]string, error)
}

// CPDRecordGenerator assembles AHPRA-format CPD records from a CPDExportSource.
type CPDRecordGenerator struct {
	source CPDExportSource
}

// NewCPDRecordGenerator returns a CPDRecordGenerator backed by the provided source.
func NewCPDRecordGenerator(s CPDExportSource) *CPDRecordGenerator {
	return &CPDRecordGenerator{source: s}
}

// Generate builds a CPDRecord for the pharmacist covering the given cycle years.
//
// Only confirmed activities (ActivityRow.Confirmed == true) contribute to
// HoursByCategory. Unconfirmed activities are silently skipped.
//
// Returns a non-nil HoursByCategory map even when no confirmed activities exist.
func (g *CPDRecordGenerator) Generate(ctx context.Context, pharmacistID string, cycleStart, cycleEnd int) (CPDRecord, error) {
	activities, err := g.source.ActivitiesInCycle(ctx, pharmacistID, cycleStart, cycleEnd)
	if err != nil {
		return CPDRecord{}, err
	}

	hoursByCategory := make(map[string]float64)
	for _, a := range activities {
		if !a.Confirmed {
			continue
		}
		hoursByCategory[a.Category] += a.Hours
	}

	return CPDRecord{
		ID:              uuid.New().String(),
		PharmacistID:    pharmacistID,
		CycleStart:      cycleStart,
		CycleEnd:        cycleEnd,
		HoursByCategory: hoursByCategory,
		GeneratedAt:     time.Now().UTC(),
	}, nil
}
