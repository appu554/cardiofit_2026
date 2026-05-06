package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestMHRSOAPClient_InterfaceContract verifies the interface signature
// shape via the stub. Production implementations must satisfy the same
// contract: deferred-error returns from the stub are intentional and
// signal that wiring is V1 work.
func TestMHRSOAPClient_InterfaceContract(t *testing.T) {
	var c MHRSOAPClient = NewStubMHRSOAPClient()

	_, err := c.GetPathologyDocumentList(context.Background(), "8003608000000001", time.Now().Add(-24*time.Hour))
	if !errors.Is(err, ErrMHRWiringDeferred) {
		t.Fatalf("GetPathologyDocumentList err = %v, want ErrMHRWiringDeferred", err)
	}

	_, err = c.FetchCDADocument(context.Background(), "DOC-SYN-0001")
	if !errors.Is(err, ErrMHRWiringDeferred) {
		t.Fatalf("FetchCDADocument err = %v, want ErrMHRWiringDeferred", err)
	}
}
