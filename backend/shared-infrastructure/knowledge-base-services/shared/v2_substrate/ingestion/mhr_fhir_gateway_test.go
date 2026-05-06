package ingestion

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMHRFHIRClient_StubReturnsDeferred(t *testing.T) {
	var c MHRFHIRClient = NewStubMHRFHIRClient()
	_, err := c.GetDiagnosticReports(context.Background(), "8003608000000001", time.Now().Add(-24*time.Hour))
	if !errors.Is(err, ErrMHRFHIRWiringDeferred) {
		t.Fatalf("GetDiagnosticReports err = %v, want ErrMHRFHIRWiringDeferred", err)
	}
}
