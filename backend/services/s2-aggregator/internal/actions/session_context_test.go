package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestStartSessionPersists(t *testing.T) {
	store := NewInMemorySessionStore()
	pharm := uuid.New()
	s, err := StartSession(context.Background(), pharm, store)
	if err != nil {
		t.Fatalf("StartSession err = %v", err)
	}
	if s.PharmacistID != pharm {
		t.Errorf("PharmacistID = %v, want %v", s.PharmacistID, pharm)
	}
	if s.StartedAt.IsZero() {
		t.Error("StartedAt zero")
	}
	if s.EndedAt != nil {
		t.Errorf("EndedAt = %v, want nil", s.EndedAt)
	}
	got, err := store.Get(context.Background(), s.SessionID)
	if err != nil || got.SessionID != s.SessionID {
		t.Errorf("Get after Create -> (%v,%v), want session present", got, err)
	}
}

func TestEndSessionStampsEndedAt(t *testing.T) {
	store := NewInMemorySessionStore()
	s, _ := StartSession(context.Background(), uuid.New(), store)
	ended, err := EndSession(context.Background(), s.SessionID, store)
	if err != nil {
		t.Fatalf("EndSession err = %v", err)
	}
	if ended.EndedAt == nil {
		t.Fatal("EndedAt nil after EndSession")
	}
}

func TestEndSessionRejectsDoubleClose(t *testing.T) {
	store := NewInMemorySessionStore()
	s, _ := StartSession(context.Background(), uuid.New(), store)
	if _, err := EndSession(context.Background(), s.SessionID, store); err != nil {
		t.Fatalf("first EndSession err = %v", err)
	}
	if _, err := EndSession(context.Background(), s.SessionID, store); !errors.Is(err, ErrSessionAlreadyEnded) {
		t.Errorf("double-close err = %v, want ErrSessionAlreadyEnded", err)
	}
}

func TestEndSessionUnknown(t *testing.T) {
	store := NewInMemorySessionStore()
	if _, err := EndSession(context.Background(), uuid.New(), store); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("EndSession on unknown id err = %v, want ErrSessionNotFound", err)
	}
}

func TestRecordActionInSession(t *testing.T) {
	store := NewInMemorySessionStore()
	s, _ := StartSession(context.Background(), uuid.New(), store)
	for i := 0; i < 3; i++ {
		if err := store.RecordActionInSession(context.Background(), s.SessionID, ActionOpen); err != nil {
			t.Fatalf("RecordActionInSession err = %v", err)
		}
	}
	got, _ := store.Get(context.Background(), s.SessionID)
	if got.ActionCount != 3 {
		t.Errorf("ActionCount = %d, want 3", got.ActionCount)
	}
}
