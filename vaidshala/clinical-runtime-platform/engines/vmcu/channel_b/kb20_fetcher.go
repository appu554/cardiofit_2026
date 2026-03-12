// Package channel_b — KB-20 raw lab fetcher interface (Phase 2.3).
//
// The V-MCU engine does not call KB-20 directly. Instead, the runtime layer
// implements this interface and provides data to the engine via TitrationCycleInput.
//
// During rule evaluation, NO network calls — Channel B reads from RawPatientData only.
package channel_b

import "context"

// KB20Fetcher defines the contract for fetching raw lab values from KB-20.
// The runtime layer implements this to populate the local safety cache.
//
// Endpoint: GET /patient/:id/labs?types=CREATININE,EGFR,FBG,HBA1C,SBP,POTASSIUM
// Data is cached locally (refreshed hourly + on KB-19 events).
type KB20Fetcher interface {
	// FetchRawLabs retrieves current lab values for a patient from KB-20.
	// Returns populated RawPatientData with current + historical values.
	// Timeout: 200ms default.
	FetchRawLabs(ctx context.Context, patientID string) (*RawPatientData, error)

	// FetchActiveMedications retrieves the patient's active medication list.
	// Used by Channel C (ProtocolGuard) for medication-conditional rules.
	// Endpoint: GET /patient/:id/medications
	FetchActiveMedications(ctx context.Context, patientID string) ([]string, error)
}
