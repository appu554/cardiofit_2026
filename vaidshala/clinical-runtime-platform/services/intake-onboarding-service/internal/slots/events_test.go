package slots

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// MockEventStore implements EventStore for testing without PostgreSQL.
type MockEventStore struct {
	events []SlotEvent
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{events: make([]SlotEvent, 0)}
}

func (m *MockEventStore) Append(ctx context.Context, event SlotEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now().UTC()
	m.events = append(m.events, event)
	return nil
}

func (m *MockEventStore) CurrentValues(ctx context.Context, patientID uuid.UUID) (map[string]SlotValue, error) {
	latest := make(map[string]SlotEvent)
	for _, e := range m.events {
		if e.PatientID == patientID {
			if existing, ok := latest[e.SlotName]; !ok || e.CreatedAt.After(existing.CreatedAt) {
				latest[e.SlotName] = e
			}
		}
	}
	result := make(map[string]SlotValue, len(latest))
	for name, e := range latest {
		result[name] = SlotValue{
			Value:          e.Value,
			ExtractionMode: e.ExtractionMode,
			Confidence:     e.Confidence,
			FHIRResourceID: e.FHIRResourceID,
			UpdatedAt:      e.CreatedAt,
		}
	}
	return result, nil
}

func (m *MockEventStore) SlotHistory(ctx context.Context, patientID uuid.UUID, slotName string) ([]SlotEvent, error) {
	var history []SlotEvent
	for _, e := range m.events {
		if e.PatientID == patientID && e.SlotName == slotName {
			history = append(history, e)
		}
	}
	return history, nil
}

func TestEventStore_AppendAndRetrieve(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// Append FBG slot event
	err := store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`178`),
		ExtractionMode: "BUTTON",
		Confidence:     1.0,
		SourceChannel:  "APP",
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	values, err := store.CurrentValues(ctx, patientID)
	if err != nil {
		t.Fatalf("CurrentValues failed: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(values))
	}
	if string(values["fbg"].Value) != "178" {
		t.Errorf("expected fbg=178, got %s", string(values["fbg"].Value))
	}
}

func TestEventStore_LatestWins(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// First FBG value
	_ = store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`178`),
		ExtractionMode: "BUTTON",
		Confidence:     1.0,
		SourceChannel:  "APP",
	})

	// Updated FBG value (correction)
	time.Sleep(time.Millisecond) // ensure different timestamp
	_ = store.Append(ctx, SlotEvent{
		PatientID:      patientID,
		SlotName:       "fbg",
		Domain:         "glycemic",
		Value:          json.RawMessage(`165`),
		ExtractionMode: "REGEX",
		Confidence:     0.95,
		SourceChannel:  "WHATSAPP",
	})

	values, _ := store.CurrentValues(ctx, patientID)
	if string(values["fbg"].Value) != "165" {
		t.Errorf("expected latest fbg=165, got %s", string(values["fbg"].Value))
	}
}

func TestEventStore_MultipleSlots(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	slots := []struct {
		name   string
		domain string
		value  string
	}{
		{"fbg", "glycemic", "178"},
		{"hba1c", "glycemic", "8.2"},
		{"egfr", "renal", "42"},
		{"systolic_bp", "cardiac", "145"},
	}

	for _, s := range slots {
		_ = store.Append(ctx, SlotEvent{
			PatientID:      patientID,
			SlotName:       s.name,
			Domain:         s.domain,
			Value:          json.RawMessage(s.value),
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			SourceChannel:  "APP",
		})
	}

	values, _ := store.CurrentValues(ctx, patientID)
	if len(values) != 4 {
		t.Errorf("expected 4 slots, got %d", len(values))
	}
}

func TestEventStore_SlotHistory(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patientID := uuid.New()

	// Three FBG entries
	for _, v := range []string{"178", "165", "150"} {
		_ = store.Append(ctx, SlotEvent{
			PatientID:      patientID,
			SlotName:       "fbg",
			Domain:         "glycemic",
			Value:          json.RawMessage(v),
			ExtractionMode: "BUTTON",
			Confidence:     1.0,
			SourceChannel:  "APP",
		})
	}

	history, _ := store.SlotHistory(ctx, patientID, "fbg")
	if len(history) != 3 {
		t.Errorf("expected 3 history entries, got %d", len(history))
	}
}

func TestEventStore_PatientIsolation(t *testing.T) {
	store := NewMockEventStore()
	ctx := context.Background()
	patient1 := uuid.New()
	patient2 := uuid.New()

	_ = store.Append(ctx, SlotEvent{
		PatientID: patient1, SlotName: "fbg", Domain: "glycemic",
		Value: json.RawMessage(`178`), ExtractionMode: "BUTTON",
		Confidence: 1.0, SourceChannel: "APP",
	})
	_ = store.Append(ctx, SlotEvent{
		PatientID: patient2, SlotName: "fbg", Domain: "glycemic",
		Value: json.RawMessage(`110`), ExtractionMode: "BUTTON",
		Confidence: 1.0, SourceChannel: "APP",
	})

	values1, _ := store.CurrentValues(ctx, patient1)
	values2, _ := store.CurrentValues(ctx, patient2)
	if string(values1["fbg"].Value) != "178" {
		t.Errorf("patient1 fbg should be 178")
	}
	if string(values2["fbg"].Value) != "110" {
		t.Errorf("patient2 fbg should be 110")
	}
}
