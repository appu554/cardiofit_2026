package services

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
)

// mockKafkaWriter captures messages for test verification.
type mockKafkaWriter struct {
	mu       sync.Mutex
	messages []mockKafkaMessage
}

type mockKafkaMessage struct {
	Topic string
	Key   string
	Value []byte
}

func (m *mockKafkaWriter) WriteMessage(ctx context.Context, topic, key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, mockKafkaMessage{Topic: topic, Key: key, Value: value})
	return nil
}

func (m *mockKafkaWriter) Close() error { return nil }

func TestKafkaOutboxRelay_PollAndPublish(t *testing.T) {
	writer := &mockKafkaWriter{}
	mapper := NewEventSignalMapper()
	log := zap.NewNop()

	relay := &KafkaOutboxRelay{
		mapper: mapper,
		writer: writer,
		log:    log,
	}

	payload, _ := json.Marshal(models.LabResultPayload{
		LabType: "FBG", Value: 6.0, Unit: "mmol/L",
		MeasuredAt: time.Now().Format(time.RFC3339),
	})

	entries := []models.EventOutboxEntry{
		{
			ID:        uuid.New(),
			EventType: models.EventLabResult,
			PatientID: "patient-1",
			Payload:   payload,
			CreatedAt: time.Now(),
		},
	}

	published := relay.processEntries(context.Background(), entries)
	if len(published) != 1 {
		t.Fatalf("expected 1 published, got %d", len(published))
	}
	if len(writer.messages) != 1 {
		t.Fatalf("expected 1 kafka message, got %d", len(writer.messages))
	}
	if writer.messages[0].Topic != "clinical.observations.v1" {
		t.Errorf("expected observations topic, got %s", writer.messages[0].Topic)
	}
	if writer.messages[0].Key != "patient-1" {
		t.Errorf("expected patient-1 key, got %s", writer.messages[0].Key)
	}
}

func TestKafkaOutboxRelay_StateChangeGoesToStateChangeTopic(t *testing.T) {
	writer := &mockKafkaWriter{}
	mapper := NewEventSignalMapper()
	log := zap.NewNop()

	relay := &KafkaOutboxRelay{
		mapper: mapper,
		writer: writer,
		log:    log,
	}

	payload, _ := json.Marshal(models.MedicationChangePayload{
		ChangeType: "ADD", DrugName: "metformin",
	})

	entries := []models.EventOutboxEntry{
		{
			ID:        uuid.New(),
			EventType: models.EventMedicationChange,
			PatientID: "patient-2",
			Payload:   payload,
			CreatedAt: time.Now(),
		},
	}

	published := relay.processEntries(context.Background(), entries)
	if len(published) != 1 {
		t.Fatalf("expected 1 published, got %d", len(published))
	}
	if writer.messages[0].Topic != "clinical.state-changes.v1" {
		t.Errorf("expected state-changes topic, got %s", writer.messages[0].Topic)
	}
}

func TestKafkaOutboxRelay_SkipsUnmappedEvents(t *testing.T) {
	writer := &mockKafkaWriter{}
	mapper := NewEventSignalMapper()
	log := zap.NewNop()

	relay := &KafkaOutboxRelay{
		mapper: mapper,
		writer: writer,
		log:    log,
	}

	entries := []models.EventOutboxEntry{
		{
			ID:        uuid.New(),
			EventType: "SOME_UNKNOWN_EVENT",
			PatientID: "patient-3",
			Payload:   json.RawMessage(`{}`),
			CreatedAt: time.Now(),
		},
	}

	published := relay.processEntries(context.Background(), entries)
	if len(published) != 1 {
		t.Fatalf("expected 1 published (skipped), got %d", len(published))
	}
	if len(writer.messages) != 0 {
		t.Errorf("expected 0 kafka messages for unmapped event, got %d", len(writer.messages))
	}
}
