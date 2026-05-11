package entry_paths

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFromNotification_HappyPath(t *testing.T) {
	notifID := uuid.New()
	dispatched := time.Now().Add(-30 * time.Second).UTC()
	meta, err := FromNotification(context.Background(), uuid.New(), uuid.New(), NotificationContext{
		NotificationID: notifID,
		ReasonText:     "Pathology result received: lithium level 0.92 mmol/L",
		DispatchedAt:   dispatched,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Path != EntryPathNotification {
		t.Errorf("Path = %q", meta.Path)
	}
	nc := meta.Context.(NotificationContext)
	if nc.NotificationID != notifID {
		t.Errorf("NotificationID not preserved")
	}
	if !nc.DispatchedAt.Equal(dispatched) {
		t.Errorf("DispatchedAt overwritten")
	}
	if nc.Kind() != EntryPathNotification {
		t.Errorf("Kind() = %q", nc.Kind())
	}
}

func TestFromNotification_DefaultsDispatchedAt(t *testing.T) {
	meta, err := FromNotification(context.Background(), uuid.New(), uuid.New(), NotificationContext{
		NotificationID: uuid.New(),
		ReasonText:     "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Context.(NotificationContext).DispatchedAt.IsZero() {
		t.Error("DispatchedAt should have defaulted")
	}
}

func TestFromNotification_RejectsZeroNotificationID(t *testing.T) {
	_, err := FromNotification(context.Background(), uuid.New(), uuid.New(), NotificationContext{
		ReasonText: "x",
	})
	if err == nil {
		t.Fatal("expected error for zero notification_id")
	}
}

func TestFromNotification_RejectsEmptyReason(t *testing.T) {
	_, err := FromNotification(context.Background(), uuid.New(), uuid.New(), NotificationContext{
		NotificationID: uuid.New(),
		ReasonText:     "  ",
	})
	if err == nil {
		t.Fatal("expected error for empty reason_text")
	}
}
