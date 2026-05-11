package entry_paths

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFromSearch_HappyPath(t *testing.T) {
	meta, err := FromSearch(context.Background(), uuid.New(), uuid.New(), "Smith, Margaret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Path != EntryPathSearch {
		t.Errorf("Path = %q want search", meta.Path)
	}
	sc, ok := meta.Context.(SearchContext)
	if !ok {
		t.Fatalf("Context not SearchContext: %T", meta.Context)
	}
	if sc.Query != "Smith, Margaret" {
		t.Errorf("Query = %q", sc.Query)
	}
	if sc.MatchedAt.IsZero() {
		t.Errorf("MatchedAt should be populated")
	}
}

func TestFromSearch_RejectsEmptyQuery(t *testing.T) {
	_, err := FromSearch(context.Background(), uuid.New(), uuid.New(), "   ")
	if err == nil {
		t.Fatal("expected error for empty/whitespace query")
	}
}

func TestFromSearch_RejectsZeroIDs(t *testing.T) {
	if _, err := FromSearch(context.Background(), uuid.Nil, uuid.New(), "q"); err == nil {
		t.Error("expected error for zero pharmacist")
	}
	if _, err := FromSearch(context.Background(), uuid.New(), uuid.Nil, "q"); err == nil {
		t.Error("expected error for zero resident")
	}
}
