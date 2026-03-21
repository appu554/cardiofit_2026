package canonical

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanonicalObservation_SourceTypes(t *testing.T) {
	sources := []SourceType{
		SourceEHR, SourceABDM, SourceLab,
		SourcePatientReported, SourceHPI, SourceDevice, SourceWearable,
	}
	if len(sources) != 7 {
		t.Errorf("expected 7 source types, got %d", len(sources))
	}
}

func TestCanonicalObservation_Construct(t *testing.T) {
	obs := CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		TenantID:        uuid.New(),
		SourceType:      SourceLab,
		SourceID:        "thyrocare",
		ObservationType: ObsLabs,
		LOINCCode:       "33914-3",
		Value:           42.0,
		Unit:            "mL/min/1.73m2",
		Timestamp:       time.Now(),
		QualityScore:    0.95,
		Flags:           []Flag{FlagCriticalValue},
	}
	if obs.SourceType != SourceLab {
		t.Errorf("expected LAB source, got %s", obs.SourceType)
	}
	if obs.LOINCCode != "33914-3" {
		t.Errorf("expected LOINC 33914-3, got %s", obs.LOINCCode)
	}
	if len(obs.Flags) != 1 || obs.Flags[0] != FlagCriticalValue {
		t.Errorf("expected CRITICAL_VALUE flag")
	}
}

func TestFlags_Constants(t *testing.T) {
	flags := []Flag{
		FlagCriticalValue, FlagImplausible, FlagLowQuality,
		FlagUnmappedCode, FlagStale, FlagDuplicate, FlagManualEntry,
	}
	if len(flags) != 7 {
		t.Errorf("expected 7 flag constants, got %d", len(flags))
	}
}
