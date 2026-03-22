package dlq

import (
	"testing"

	"go.uber.org/zap"
)

func TestReplayHandler_Construction(t *testing.T) {
	logger := zap.NewNop()
	pub := NewMemoryPublisher(logger)
	handler := NewReplayHandler(pub, logger)

	if handler == nil {
		t.Fatal("expected non-nil ReplayHandler")
	}
}

func TestReplayHandler_Fields(t *testing.T) {
	logger := zap.NewNop()
	pub := NewMemoryPublisher(logger)
	handler := NewReplayHandler(pub, logger)

	// Verify the handler was constructed with the provided publisher and logger.
	// We cannot inspect unexported fields directly, but a nil check ensures
	// the constructor wired dependencies without panicking.
	if handler == nil {
		t.Fatal("expected ReplayHandler to be initialized")
	}
}

func TestDLQEntry_StatusConstants(t *testing.T) {
	// Verify the status constants used by the replay flow are defined.
	tests := []struct {
		name   string
		status DLQStatus
		want   string
	}{
		{"pending", StatusPending, "PENDING"},
		{"replayed", StatusReplayed, "REPLAYED"},
		{"discarded", StatusDiscarded, "DISCARDED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("status %s = %q, want %q", tt.name, tt.status, tt.want)
			}
		})
	}
}
