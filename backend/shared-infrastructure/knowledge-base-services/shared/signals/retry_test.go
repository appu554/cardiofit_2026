package signals

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryHandler_SucceedsFirstAttempt(t *testing.T) {
	attempts := 0
	handler := func(ctx context.Context, data []byte) error {
		attempts++
		return nil
	}
	rh := NewRetryHandler(handler, 3, "test-consumer", nil)
	err := rh.Handle(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryHandler_FailsAllRetries_PublishesDLQ(t *testing.T) {
	handler := func(ctx context.Context, data []byte) error {
		return errors.New("permanent failure")
	}
	var dlqData []byte
	dlqPublisher := func(ctx context.Context, consumerGroup string, data []byte, err error) {
		dlqData = data
	}
	rh := NewRetryHandler(handler, 3, "test-consumer", dlqPublisher)
	// Override backoffs to zero so the test doesn't wait 36 seconds.
	rh.backoffs = []time.Duration{0, 0, 0}
	retErr := rh.Handle(context.Background(), []byte(`{"test":true}`))
	if retErr == nil {
		t.Error("expected error after exhausted retries")
	}
	if dlqData == nil {
		t.Error("expected DLQ publish after exhausted retries")
	}
}
