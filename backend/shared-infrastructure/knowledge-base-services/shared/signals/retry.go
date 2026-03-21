package signals

import (
	"context"
	"time"
)

// DLQPublisher is called when retries are exhausted.
type DLQPublisher func(ctx context.Context, consumerGroup string, data []byte, err error)

// RetryHandler wraps a message handler with exponential backoff retry and DLQ publishing.
type RetryHandler struct {
	handler       func(ctx context.Context, data []byte) error
	maxRetries    int
	backoffs      []time.Duration
	dlqPublisher  DLQPublisher
	consumerGroup string
}

// NewRetryHandler creates a handler with exponential backoff retries (1s, 5s, 30s).
func NewRetryHandler(handler func(ctx context.Context, data []byte) error, maxRetries int, consumerGroup string, dlqPublisher DLQPublisher) *RetryHandler {
	return &RetryHandler{
		handler:       handler,
		maxRetries:    maxRetries,
		backoffs:      []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second},
		dlqPublisher:  dlqPublisher,
		consumerGroup: consumerGroup,
	}
}

// Handle executes the handler with retries. On exhaustion, publishes to DLQ.
func (r *RetryHandler) Handle(ctx context.Context, data []byte) error {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		lastErr = r.handler(ctx, data)
		if lastErr == nil {
			return nil
		}
		if attempt < r.maxRetries && attempt < len(r.backoffs) {
			select {
			case <-time.After(r.backoffs[attempt]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	// Exhausted retries — publish to DLQ
	if r.dlqPublisher != nil {
		r.dlqPublisher(ctx, r.consumerGroup, data, lastErr)
	}
	return lastErr
}
