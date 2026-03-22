package slots

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
)

// SlotSnapshot holds all current slot values for a patient, keyed by slot name.
type SlotSnapshot struct {
	PatientID uuid.UUID            `json:"patient_id"`
	Values    map[string]SlotValue `json:"values"`
	Filled    int                  `json:"filled"`
	Total     int                  `json:"total"`
	Required  int                  `json:"required"`
	Missing   []string             `json:"missing_required"`
}

// BuildSnapshot constructs a SlotSnapshot from current values.
func BuildSnapshot(patientID uuid.UUID, values map[string]SlotValue) SlotSnapshot {
	allSlots := AllSlots()
	requiredSlots := RequiredSlots()

	var missingRequired []string
	for _, s := range requiredSlots {
		if _, ok := values[s.Name]; !ok {
			missingRequired = append(missingRequired, s.Name)
		}
	}

	return SlotSnapshot{
		PatientID: patientID,
		Values:    values,
		Filled:    len(values),
		Total:     len(allSlots),
		Required:  len(requiredSlots),
		Missing:   missingRequired,
	}
}

// IsComplete returns true if all required slots are filled.
func (ss SlotSnapshot) IsComplete() bool {
	return len(ss.Missing) == 0
}

// GetFloat64 extracts a float64 value from the snapshot by slot name.
// Returns 0 and false if the slot is not filled or not a valid number.
func (ss SlotSnapshot) GetFloat64(slotName string) (float64, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return 0, false
	}
	var v float64
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		// Try string-encoded number
		var s string
		if err2 := json.Unmarshal(sv.Value, &s); err2 == nil {
			if f, err3 := strconv.ParseFloat(s, 64); err3 == nil {
				return f, true
			}
		}
		return 0, false
	}
	return v, true
}

// GetBool extracts a boolean value from the snapshot by slot name.
func (ss SlotSnapshot) GetBool(slotName string) (bool, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return false, false
	}
	var v bool
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		return false, false
	}
	return v, true
}

// GetInt extracts an integer value from the snapshot by slot name.
func (ss SlotSnapshot) GetInt(slotName string) (int, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return 0, false
	}
	var v int
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		// Try float (JSON numbers may decode as float)
		var f float64
		if err2 := json.Unmarshal(sv.Value, &f); err2 == nil {
			return int(f), true
		}
		return 0, false
	}
	return v, true
}

// GetString extracts a string value from the snapshot by slot name.
func (ss SlotSnapshot) GetString(slotName string) (string, bool) {
	sv, ok := ss.Values[slotName]
	if !ok {
		return "", false
	}
	var v string
	if err := json.Unmarshal(sv.Value, &v); err != nil {
		return "", false
	}
	return v, true
}

// FilledSlotNames returns the names of all currently filled slots.
func FilledSlotNames(ctx context.Context, store EventStore, patientID uuid.UUID) ([]string, error) {
	values, err := store.CurrentValues(ctx, patientID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	return names, nil
}
