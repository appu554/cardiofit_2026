package rules

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// IN-MEMORY QUEUE TESTS
// =============================================================================

func TestNewInMemoryQueue(t *testing.T) {
	queue := NewInMemoryQueue()

	if queue == nil {
		t.Fatal("Expected queue to be created")
	}
}

func TestInMemoryQueue_Enqueue(t *testing.T) {
	queue := NewInMemoryQueue()

	entry := createTestUntranslatableEntry()

	ctx := context.Background()
	err := queue.Enqueue(ctx, entry)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify entry is in queue
	if len(queue.entries) != 1 {
		t.Errorf("Expected 1 entry in queue, got %d", len(queue.entries))
	}
}

func TestInMemoryQueue_GetPending(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 3; i++ {
		entry := createTestUntranslatableEntry()
		queue.Enqueue(ctx, entry)
	}

	pending, err := queue.GetPending(ctx, 10)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("Expected 3 pending entries, got %d", len(pending))
	}
}

func TestInMemoryQueue_GetPending_Limit(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	// Add 10 entries
	for i := 0; i < 10; i++ {
		entry := createTestUntranslatableEntry()
		queue.Enqueue(ctx, entry)
	}

	// Request only 5
	pending, err := queue.GetPending(ctx, 5)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(pending) != 5 {
		t.Errorf("Expected 5 pending entries (limited), got %d", len(pending))
	}
}

func TestInMemoryQueue_Assign(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	entry := createTestUntranslatableEntry()
	queue.Enqueue(ctx, entry)

	err := queue.Assign(ctx, entry.ID, "reviewer@example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify assignment
	assignedEntry := queue.entries[entry.ID]
	if assignedEntry.Status != StatusAssigned {
		t.Errorf("Expected status ASSIGNED, got %s", assignedEntry.Status)
	}

	if assignedEntry.AssignedTo != "reviewer@example.com" {
		t.Errorf("Expected assignee 'reviewer@example.com', got '%s'", assignedEntry.AssignedTo)
	}
}

func TestInMemoryQueue_Resolve_ManualRule(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	entry := createTestUntranslatableEntry()
	queue.Enqueue(ctx, entry)
	queue.Assign(ctx, entry.ID, "reviewer@example.com")

	createdRuleID := uuid.New()
	err := queue.Resolve(ctx, entry.ID, ResolutionManualRule, &createdRuleID, "Created dosing rule manually", "reviewer@example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify resolution
	resolvedEntry := queue.entries[entry.ID]
	if resolvedEntry.Status != StatusResolved {
		t.Errorf("Expected status RESOLVED, got %s", resolvedEntry.Status)
	}

	if resolvedEntry.Resolution != ResolutionManualRule {
		t.Errorf("Expected resolution MANUAL_RULE, got %s", resolvedEntry.Resolution)
	}

	if resolvedEntry.CreatedRuleID == nil || *resolvedEntry.CreatedRuleID != createdRuleID {
		t.Error("Expected created rule ID to be set")
	}
}

func TestInMemoryQueue_Resolve_NotApplicable(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	entry := createTestUntranslatableEntry()
	queue.Enqueue(ctx, entry)
	queue.Assign(ctx, entry.ID, "reviewer@example.com")

	err := queue.Resolve(ctx, entry.ID, ResolutionNotApplicable, nil, "Table is informational only", "reviewer@example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	resolvedEntry := queue.entries[entry.ID]
	if resolvedEntry.Resolution != ResolutionNotApplicable {
		t.Errorf("Expected resolution NOT_APPLICABLE, got %s", resolvedEntry.Resolution)
	}
}

func TestInMemoryQueue_Defer(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	entry := createTestUntranslatableEntry()
	queue.Enqueue(ctx, entry)

	newDeadline := time.Now().Add(168 * time.Hour) // 1 week
	err := queue.Defer(ctx, entry.ID, newDeadline, "Waiting for clinical review")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	deferredEntry := queue.entries[entry.ID]
	if deferredEntry.Status != StatusDeferred {
		t.Errorf("Expected status DEFERRED, got %s", deferredEntry.Status)
	}
}

func TestInMemoryQueue_Escalate(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	entry := createTestUntranslatableEntry()
	queue.Enqueue(ctx, entry)

	err := queue.Escalate(ctx, entry.ID, ResolutionEscalateLLM, "Complex table requires LLM assistance")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	escalatedEntry := queue.entries[entry.ID]
	if escalatedEntry.Status != StatusEscalated {
		t.Errorf("Expected status ESCALATED, got %s", escalatedEntry.Status)
	}

	if escalatedEntry.Resolution != ResolutionEscalateLLM {
		t.Errorf("Expected resolution ESCALATE_LLM, got %s", escalatedEntry.Resolution)
	}
}

// =============================================================================
// UNTRANSLATABLE ENTRY TESTS
// =============================================================================

func TestUntranslatableEntry_Fields(t *testing.T) {
	entry := UntranslatableEntry{
		ID:               uuid.New(),
		TableID:          "table-001",
		Headers:          []string{"Col1", "Col2"},
		RowCount:         5,
		Reason:           "no_condition_column",
		SourceDocumentID: uuid.New(),
		SourceInfo:       "metformin-001/34068-7",
		TableType:        "DOSING",
		Status:           StatusPending,
		SLADeadline:      time.Now().Add(72 * time.Hour),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if entry.TableID != "table-001" {
		t.Errorf("Expected TableID 'table-001', got '%s'", entry.TableID)
	}

	if entry.Reason != "no_condition_column" {
		t.Errorf("Expected reason 'no_condition_column', got '%s'", entry.Reason)
	}

	if entry.Status != StatusPending {
		t.Errorf("Expected status PENDING, got %s", entry.Status)
	}
}

// =============================================================================
// STATUS TESTS
// =============================================================================

func TestEntryStatus_Values(t *testing.T) {
	statuses := []EntryStatus{
		StatusPending,
		StatusAssigned,
		StatusInReview,
		StatusResolved,
		StatusEscalated,
		StatusDeferred,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Expected status to have a value")
		}
	}
}

func TestResolution_Values(t *testing.T) {
	resolutions := []Resolution{
		ResolutionManualRule,
		ResolutionNotApplicable,
		ResolutionEscalateLLM,
		ResolutionMergedExisting,
		ResolutionSplitRules,
	}

	for _, resolution := range resolutions {
		if resolution == "" {
			t.Error("Expected resolution to have a value")
		}
	}
}

// =============================================================================
// SLA TESTS
// =============================================================================

func TestInMemoryQueue_CheckSLABreaches(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	// Add an entry with past deadline
	entry := createTestUntranslatableEntry()
	entry.SLADeadline = time.Now().Add(-24 * time.Hour) // Past deadline
	queue.Enqueue(ctx, entry)

	// Add an entry with future deadline
	entry2 := createTestUntranslatableEntry()
	entry2.SLADeadline = time.Now().Add(24 * time.Hour) // Future deadline
	queue.Enqueue(ctx, entry2)

	breached, err := queue.GetSLABreached(ctx)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(breached) != 1 {
		t.Errorf("Expected 1 SLA breach, got %d", len(breached))
	}
}

func TestInMemoryQueue_DefaultSLADeadline(t *testing.T) {
	entry := createTestUntranslatableEntry()

	// Default SLA should be 72 hours
	expectedDeadline := time.Now().Add(72 * time.Hour)
	timeDiff := entry.SLADeadline.Sub(expectedDeadline)

	// Allow 1 second tolerance
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("Expected SLA deadline around 72 hours from now, got %v", entry.SLADeadline)
	}
}

// =============================================================================
// PRIORITY TESTS
// =============================================================================

func TestAssignPriority_ByReason(t *testing.T) {
	tests := []struct {
		reason   string
		expected int
	}{
		{"no_condition_column", 2}, // Medium priority
		{"mixed_units", 3},          // High priority
		{"free_text", 1},            // Low priority
		{"unknown_reason", 1},       // Default low
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			entry := &UntranslatableEntry{Reason: tt.reason}
			priority := AssignPriority(entry)

			if priority != tt.expected {
				t.Errorf("Expected priority %d for reason '%s', got %d", tt.expected, tt.reason, priority)
			}
		})
	}
}

// =============================================================================
// QUEUE STATISTICS TESTS
// =============================================================================

func TestInMemoryQueue_GetStats(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	// Add entries with different statuses
	pending := createTestUntranslatableEntry()
	queue.Enqueue(ctx, pending)

	assigned := createTestUntranslatableEntry()
	queue.Enqueue(ctx, assigned)
	queue.Assign(ctx, assigned.ID, "reviewer@example.com")

	resolved := createTestUntranslatableEntry()
	queue.Enqueue(ctx, resolved)
	queue.Assign(ctx, resolved.ID, "reviewer@example.com")
	queue.Resolve(ctx, resolved.ID, ResolutionManualRule, nil, "Resolved", "reviewer@example.com")

	stats, err := queue.GetStats(ctx)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if stats.TotalEntries != 3 {
		t.Errorf("Expected 3 total entries, got %d", stats.TotalEntries)
	}

	if stats.PendingCount != 1 {
		t.Errorf("Expected 1 pending, got %d", stats.PendingCount)
	}

	if stats.AssignedCount != 1 {
		t.Errorf("Expected 1 assigned, got %d", stats.AssignedCount)
	}

	if stats.ResolvedCount != 1 {
		t.Errorf("Expected 1 resolved, got %d", stats.ResolvedCount)
	}
}

// =============================================================================
// ERROR HANDLING TESTS
// =============================================================================

func TestInMemoryQueue_Assign_NotFound(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	err := queue.Assign(ctx, uuid.New(), "reviewer@example.com")

	if err == nil {
		t.Error("Expected error for nonexistent entry")
	}
}

func TestInMemoryQueue_Resolve_NotFound(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	err := queue.Resolve(ctx, uuid.New(), ResolutionManualRule, nil, "Notes", "reviewer")

	if err == nil {
		t.Error("Expected error for nonexistent entry")
	}
}

// =============================================================================
// CONCURRENT ACCESS TESTS
// =============================================================================

func TestInMemoryQueue_ConcurrentAccess(t *testing.T) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	done := make(chan bool, 20)

	// Concurrent enqueues
	for i := 0; i < 10; i++ {
		go func() {
			entry := createTestUntranslatableEntry()
			queue.Enqueue(ctx, entry)
			done <- true
		}()
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			queue.GetPending(ctx, 10)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createTestUntranslatableEntry() *UntranslatableEntry {
	return &UntranslatableEntry{
		ID:               uuid.New(),
		TableID:          "table-" + uuid.New().String()[:8],
		Headers:          []string{"CrCl", "Dose", "Notes"},
		RowCount:         5,
		Reason:           "no_condition_column",
		SourceDocumentID: uuid.New(),
		SourceInfo:       "test-doc/34068-7",
		TableType:        "DOSING",
		Status:           StatusPending,
		SLADeadline:      time.Now().Add(72 * time.Hour),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

// =============================================================================
// BENCHMARKS
// =============================================================================

func BenchmarkInMemoryQueue_Enqueue(b *testing.B) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := createTestUntranslatableEntry()
		queue.Enqueue(ctx, entry)
	}
}

func BenchmarkInMemoryQueue_GetPending(b *testing.B) {
	queue := NewInMemoryQueue()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 1000; i++ {
		entry := createTestUntranslatableEntry()
		queue.Enqueue(ctx, entry)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		queue.GetPending(ctx, 100)
	}
}

func BenchmarkAssignPriority(b *testing.B) {
	entry := &UntranslatableEntry{Reason: "mixed_units"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AssignPriority(entry)
	}
}
