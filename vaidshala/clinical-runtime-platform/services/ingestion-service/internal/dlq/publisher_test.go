package dlq

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	l, _ := zap.NewDevelopment()
	return l
}

func TestDLQEntry_Validate(t *testing.T) {
	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		SourceID:     "thyrocare",
		RawPayload:   []byte(`{"invalid": "json`),
		ErrorMessage: "unexpected end of JSON input",
	}

	err := entry.Validate()
	require.NoError(t, err)
}

func TestDLQEntry_ValidateEmptyPayload(t *testing.T) {
	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		ErrorMessage: "some error",
	}

	err := entry.Validate()
	assert.Error(t, err) // Missing raw payload
}

func TestDLQEntry_ValidateEmptyErrorClass(t *testing.T) {
	entry := &DLQEntry{
		SourceType:   "LAB",
		RawPayload:   []byte("data"),
		ErrorMessage: "some error",
	}

	err := entry.Validate()
	assert.Error(t, err) // Missing error class
}

func TestErrorClasses(t *testing.T) {
	classes := []ErrorClass{
		ErrorClassParse,
		ErrorClassNormalization,
		ErrorClassValidation,
		ErrorClassMapping,
		ErrorClassPublish,
		ErrorClassFHIRWrite,
	}
	assert.Len(t, classes, 6)
}

func TestPublisher_PublishToMemory(t *testing.T) {
	p := NewMemoryPublisher(testLogger())

	entry := &DLQEntry{
		ErrorClass:   ErrorClassParse,
		SourceType:   "LAB",
		SourceID:     "thyrocare",
		RawPayload:   []byte(`{"bad": "data"`),
		ErrorMessage: "invalid JSON",
	}

	err := p.Publish(context.Background(), entry)
	require.NoError(t, err)

	entries := p.ListPending(context.Background())
	require.Len(t, entries, 1)
	assert.Equal(t, ErrorClassParse, entries[0].ErrorClass)
	assert.Equal(t, StatusPending, entries[0].Status)
}

func TestPublisher_ReplayEntry(t *testing.T) {
	p := NewMemoryPublisher(testLogger())

	entry := &DLQEntry{
		ErrorClass:   ErrorClassValidation,
		SourceType:   "DEVICE",
		RawPayload:   []byte(`{"value": -1}`),
		ErrorMessage: "negative value",
	}

	_ = p.Publish(context.Background(), entry)
	entries := p.ListPending(context.Background())
	require.Len(t, entries, 1)

	err := p.MarkReplayed(context.Background(), entries[0].ID)
	require.NoError(t, err)

	pending := p.ListPending(context.Background())
	assert.Len(t, pending, 0)
}
