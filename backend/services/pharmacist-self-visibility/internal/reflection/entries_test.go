package reflection

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEntry_AuthorOnlyRead(t *testing.T) {
	store := NewInMemoryStore()
	author := uuid.New()
	other := uuid.New()

	entry, err := store.Create(context.Background(), Entry{
		PharmacistID: author,
		Body:         "Worked on a complex deprescribing case today.",
		PromptID:     nil,
		Tags:         []string{"deprescribing", "complex_case"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Author can read.
	got, err := store.Get(context.Background(), author, entry.ID)
	if err != nil || got == nil {
		t.Fatalf("author read: err=%v entry=%v", err, got)
	}

	// Non-author gets ErrNotAuthorized, not the entry.
	_, err = store.Get(context.Background(), other, entry.ID)
	if err != ErrNotAuthorized {
		t.Errorf("expected ErrNotAuthorized for non-author, got %v", err)
	}
}

func TestEntry_ListByAuthorOnly(t *testing.T) {
	store := NewInMemoryStore()
	a := uuid.New()
	b := uuid.New()
	for i := 0; i < 3; i++ {
		_, _ = store.Create(context.Background(), Entry{PharmacistID: a, Body: "a"})
	}
	_, _ = store.Create(context.Background(), Entry{PharmacistID: b, Body: "b"})

	listA, _ := store.ListByAuthor(context.Background(), a, 50)
	listB, _ := store.ListByAuthor(context.Background(), b, 50)
	if len(listA) != 3 || len(listB) != 1 {
		t.Errorf("listA=%d listB=%d (want 3 / 1)", len(listA), len(listB))
	}
	// Cross-author list with mismatched pharmacist returns 0.
	if list, _ := store.ListByAuthor(context.Background(), uuid.New(), 50); len(list) != 0 {
		t.Errorf("unknown pharmacist should see 0 entries, got %d", len(list))
	}
	_ = time.Now() // suppress unused if test grows
}
