// Package factstore provides integration tests for the FactStore Pipeline
package factstore

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/governance/routing"
)

// TestPipelineConfiguration verifies pipeline config defaults
func TestPipelineConfiguration(t *testing.T) {
	config := DefaultPipelineConfig()

	// Auto-approve DISABLED — all facts route to PENDING_REVIEW for pharmacist review
	if config.AutoApproveThreshold != 2.0 {
		t.Errorf("Expected AutoApproveThreshold=2.0 (disabled), got %.2f", config.AutoApproveThreshold)
	}
	if config.ReviewThreshold != 0.65 {
		t.Errorf("Expected ReviewThreshold=0.65, got %.2f", config.ReviewThreshold)
	}

	// Verify LLM consensus settings
	if config.LLMConsensusRequired != 2 {
		t.Errorf("Expected LLMConsensusRequired=2, got %d", config.LLMConsensusRequired)
	}
}

// TestGovernanceDecisionLogic tests the confidence-based governance rules
func TestGovernanceDecisionLogic(t *testing.T) {
	testCases := []struct {
		name           string
		confidence     float64
		expectedStatus string
	}{
		// Auto-approve DISABLED (threshold=2.0) — all facts route to PENDING_REVIEW
		{"High confidence pending", 0.95, "PENDING_REVIEW"},
		{"Medium-high confidence pending", 0.85, "PENDING_REVIEW"},
		{"Medium confidence pending", 0.75, "PENDING_REVIEW"},
		{"Review queue lower", 0.65, "PENDING_REVIEW"},
		{"Auto-reject below threshold", 0.64, "REJECTED"},
		{"Low confidence reject", 0.50, "REJECTED"},
	}

	config := DefaultPipelineConfig()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var status string
			switch {
			case tc.confidence >= config.AutoApproveThreshold:
				status = "APPROVED"
			case tc.confidence >= config.ReviewThreshold:
				status = "PENDING_REVIEW"
			default:
				status = "REJECTED"
			}

			if status != tc.expectedStatus {
				t.Errorf("Confidence %.2f: expected %s, got %s", tc.confidence, tc.expectedStatus, status)
			}
		})
	}
}

// TestCriticalSafetyFactDetection tests the critical safety fact detection
func TestCriticalSafetyFactDetection(t *testing.T) {
	criticalTypes := []string{
		"BLACK_BOX_WARNING",
		"CONTRAINDICATION",
		"QT_PROLONGATION",
		"HEPATOTOXICITY",
		"SERIOUS_ADVERSE",
		"RENAL_DOSE_ADJUST",
		"HEPATIC_DOSE_ADJUST",
	}

	nonCriticalTypes := []string{
		"PK_PARAMETER",
		"GENERAL_TABLE",
		"GAP_FILL_NEEDED",
	}

	for _, factType := range criticalTypes {
		if !isCriticalSafetyFact(factType) {
			t.Errorf("Expected %s to be critical safety fact", factType)
		}
	}

	for _, factType := range nonCriticalTypes {
		if isCriticalSafetyFact(factType) {
			t.Errorf("Expected %s to NOT be critical safety fact", factType)
		}
	}
}

// TestTableTypeMapping tests the mapping from table types to fact types
// Maps to canonical KB fact types: ORGAN_IMPAIRMENT, SAFETY_SIGNAL, INTERACTION, etc.
// Some table types are deliberately disabled (return "") — see pipeline.go comments
func TestTableTypeMapping(t *testing.T) {
	testCases := []struct {
		tableType        dailymed.TableType
		expectedFactType string
		description      string
	}{
		{dailymed.TableTypeGFRDosing, "ORGAN_IMPAIRMENT", "P5.1: GFR dosing re-enabled with 0.70 confidence cap"},
		{dailymed.TableTypeHepaticDosing, "ORGAN_IMPAIRMENT", "P5.1: Hepatic dosing re-enabled with 0.70 confidence cap"},
		{dailymed.TableTypeDosing, "", "General dosing tables dropped — not extractable"},
		{dailymed.TableTypeAdverseEvents, "SAFETY_SIGNAL", "AE tables → SAFETY_SIGNAL for KB-4"},
		{dailymed.TableTypeDDI, "INTERACTION", "DDI tables → INTERACTION for KB-5"},
		{dailymed.TableTypePK, "", "PK tables disabled — LAB_REFERENCE extraction parked"},
		{dailymed.TableTypeContraindications, "SAFETY_SIGNAL", "Contraindications → SAFETY_SIGNAL for KB-4"},
		{dailymed.TableTypeUnknown, "SAFETY_SIGNAL", "Unknown defaults to SAFETY_SIGNAL"},
	}

	for _, tc := range testCases {
		result := mapTableTypeToFactType(tc.tableType)
		if result != tc.expectedFactType {
			t.Errorf("TableType %v (%s): expected %q, got %q", tc.tableType, tc.description, tc.expectedFactType, result)
		}
	}
}

// TestPriorityCalculation tests the priority calculation logic
func TestPriorityCalculation(t *testing.T) {
	// Critical safety facts should always be HIGH priority
	priority := calculatePriority(0.50, "BLACK_BOX_WARNING")
	if priority != "HIGH" {
		t.Errorf("Critical safety fact should be HIGH priority, got %s", priority)
	}

	// Non-critical facts use confidence-based priority
	priority = calculatePriority(0.85, "GENERAL_TABLE")
	if priority != "NORMAL" {
		t.Errorf("High confidence non-critical should be NORMAL priority, got %s", priority)
	}

	priority = calculatePriority(0.60, "GENERAL_TABLE")
	if priority != "LOW" {
		t.Errorf("Low confidence non-critical should be LOW priority, got %s", priority)
	}
}

// TestSourceDocumentModel tests the source document model
func TestSourceDocumentModel(t *testing.T) {
	doc := &SourceDocument{
		SourceType:       "FDA_SPL",
		DocumentID:       "test-setid-123",
		VersionNumber:    "1",
		RawContentHash:   "abc123",
		FetchedAt:        time.Now(),
		DrugName:         "Test Drug",
		RxCUI:            "12345",
		ProcessingStatus: "PENDING",
	}

	// Verify JSON marshaling
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal source document: %v", err)
	}

	var unmarshaled SourceDocument
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal source document: %v", err)
	}

	if unmarshaled.SourceType != doc.SourceType {
		t.Errorf("SourceType mismatch: expected %s, got %s", doc.SourceType, unmarshaled.SourceType)
	}
	if unmarshaled.DrugName != doc.DrugName {
		t.Errorf("DrugName mismatch: expected %s, got %s", doc.DrugName, unmarshaled.DrugName)
	}
}

// TestDerivedFactModel tests the derived fact model
func TestDerivedFactModel(t *testing.T) {
	factData := map[string]interface{}{
		"gfr_range": "30-60",
		"dose":      "500mg",
		"frequency": "once daily",
	}
	factDataJSON, _ := json.Marshal(factData)

	fact := &DerivedFact{
		SourceDocumentID:     "doc-123",
		TargetKB:             "KB-1",
		FactType:             "RENAL_DOSE_ADJUST",
		FactKey:              "metformin:gfr:30-60",
		FactData:             factDataJSON,
		ExtractionMethod:     "TABLE_PARSE",
		ExtractionConfidence: 0.90,
		ConsensusAchieved:    true,
		GovernanceStatus:     "DRAFT",
		IsActive:             true,
	}

	// Verify JSON marshaling
	data, err := json.Marshal(fact)
	if err != nil {
		t.Fatalf("Failed to marshal derived fact: %v", err)
	}

	var unmarshaled DerivedFact
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal derived fact: %v", err)
	}

	if unmarshaled.TargetKB != fact.TargetKB {
		t.Errorf("TargetKB mismatch: expected %s, got %s", fact.TargetKB, unmarshaled.TargetKB)
	}
	if unmarshaled.ExtractionConfidence != fact.ExtractionConfidence {
		t.Errorf("Confidence mismatch: expected %.2f, got %.2f", fact.ExtractionConfidence, unmarshaled.ExtractionConfidence)
	}
}

// TestEscalationEntryModel tests the escalation entry model
func TestEscalationEntryModel(t *testing.T) {
	entry := &EscalationEntry{
		DerivedFactID:    "fact-123",
		EscalationReason: "LOW_CONFIDENCE",
		Priority:         "HIGH",
		Status:           "PENDING",
	}

	// Verify JSON marshaling
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Failed to marshal escalation entry: %v", err)
	}

	var unmarshaled EscalationEntry
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal escalation entry: %v", err)
	}

	if unmarshaled.EscalationReason != entry.EscalationReason {
		t.Errorf("EscalationReason mismatch: expected %s, got %s", entry.EscalationReason, unmarshaled.EscalationReason)
	}
	if unmarshaled.Priority != entry.Priority {
		t.Errorf("Priority mismatch: expected %s, got %s", entry.Priority, unmarshaled.Priority)
	}
}

// TestPipelineMetrics tests the pipeline metrics structure
func TestPipelineMetrics(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	config := DefaultPipelineConfig()

	sectionRouter := dailymed.NewSectionRouter()
	authorityRouter := routing.NewAuthorityRouter()

	pipeline := NewPipeline(config, nil, sectionRouter, authorityRouter, log, nil, nil, nil)

	metrics := pipeline.GetMetrics()

	// Initial metrics should be zero
	if metrics.DocumentsProcessed != 0 {
		t.Errorf("Expected 0 documents processed, got %d", metrics.DocumentsProcessed)
	}
	if metrics.FactsExtracted != 0 {
		t.Errorf("Expected 0 facts extracted, got %d", metrics.FactsExtracted)
	}
}

// TestHashComputation tests the content hash computation
func TestHashComputation(t *testing.T) {
	content1 := "Test content for hashing"
	content2 := "Different content"

	hash1 := computeHash(content1)
	hash2 := computeHash(content2)
	hash1Again := computeHash(content1)

	// Same content should produce same hash
	if hash1 != hash1Again {
		t.Error("Same content produced different hashes")
	}

	// Different content should produce different hash
	if hash1 == hash2 {
		t.Error("Different content produced same hash")
	}

	// Hash should be 64 characters (SHA-256 hex)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

// TestTruncateString tests the string truncation helper
func TestTruncateString(t *testing.T) {
	longString := "This is a very long string that should be truncated"
	shortString := "Short"

	truncated := truncateString(longString, 10)
	if len(truncated) > 13 { // 10 + "..."
		t.Errorf("Truncation failed: %s", truncated)
	}
	if !contains(truncated, "...") {
		t.Error("Truncated string should contain ...")
	}

	notTruncated := truncateString(shortString, 100)
	if notTruncated != shortString {
		t.Errorf("Short string was modified: %s", notTruncated)
	}
}

// TestProcessingResultModel tests the processing result model
func TestProcessingResultModel(t *testing.T) {
	result := &ProcessingResult{
		SourceDocumentID:  "doc-123",
		StartedAt:         time.Now(),
		TotalSections:     10,
		SectionsProcessed: 8,
		FactsExtracted:    15,
		FactsApproved:     12,
		FactsQueued:       2,
		FactsRejected:     1,
		Errors:            []string{"Error 1", "Error 2"},
	}

	// Verify JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal processing result: %v", err)
	}

	var unmarshaled ProcessingResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal processing result: %v", err)
	}

	if unmarshaled.TotalSections != result.TotalSections {
		t.Errorf("TotalSections mismatch: expected %d, got %d", result.TotalSections, unmarshaled.TotalSections)
	}
	if unmarshaled.FactsExtracted != result.FactsExtracted {
		t.Errorf("FactsExtracted mismatch: expected %d, got %d", result.FactsExtracted, unmarshaled.FactsExtracted)
	}
	if len(unmarshaled.Errors) != len(result.Errors) {
		t.Errorf("Errors count mismatch: expected %d, got %d", len(result.Errors), len(unmarshaled.Errors))
	}
}

// TestFactStoreStatsModel tests the stats model
func TestFactStoreStatsModel(t *testing.T) {
	stats := &FactStoreStats{
		TotalDocuments:     100,
		TotalSections:      500,
		TotalFacts:         1500,
		FactsByKB:          map[string]int{"KB-1": 500, "KB-4": 800, "KB-5": 200},
		FactsByStatus:      map[string]int{"APPROVED": 1200, "PENDING_REVIEW": 200, "REJECTED": 100},
		FactsByMethod:      map[string]int{"AUTHORITY": 600, "TABLE_PARSE": 700, "LLM_GAP": 200},
		PendingEscalations: 25,
		AvgConfidence:      0.82,
	}

	// Verify JSON marshaling
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Failed to marshal stats: %v", err)
	}

	var unmarshaled FactStoreStats
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal stats: %v", err)
	}

	if unmarshaled.TotalFacts != stats.TotalFacts {
		t.Errorf("TotalFacts mismatch: expected %d, got %d", stats.TotalFacts, unmarshaled.TotalFacts)
	}
	if unmarshaled.FactsByKB["KB-1"] != stats.FactsByKB["KB-1"] {
		t.Errorf("FactsByKB[KB-1] mismatch")
	}
}

// =============================================================================
// INTEGRATION SCENARIO TESTS
// =============================================================================

// TestEndToEndScenario_AuthorityPath tests the authority extraction path
func TestEndToEndScenario_AuthorityPath(t *testing.T) {
	t.Skip("Requires database connection - run with integration test flag")

	// This test would:
	// 1. Create a mock SPL document with NURSING MOTHERS section
	// 2. Process through pipeline
	// 3. Verify LactMed authority was consulted
	// 4. Verify fact was auto-approved (confidence = 1.0 for authority)
}

// TestEndToEndScenario_TableParsePath tests the table parsing path
func TestEndToEndScenario_TableParsePath(t *testing.T) {
	t.Skip("Requires database connection - run with integration test flag")

	// This test would:
	// 1. Create a mock SPL document with renal dosing table
	// 2. Process through pipeline
	// 3. Verify table was parsed
	// 4. Verify fact was auto-approved (confidence = 0.90 for table parse)
}

// TestEndToEndScenario_ReviewQueuePath tests the review queue path
func TestEndToEndScenario_ReviewQueuePath(t *testing.T) {
	t.Skip("Requires database connection - run with integration test flag")

	// This test would:
	// 1. Create a mock fact with confidence 0.75
	// 2. Process through governance
	// 3. Verify fact was queued for review
	// 4. Verify escalation entry was created
}

// TestEndToEndScenario_CriticalSafetyEscalation tests critical safety escalation
func TestEndToEndScenario_CriticalSafetyEscalation(t *testing.T) {
	t.Skip("Requires database connection - run with integration test flag")

	// This test would:
	// 1. Create a BLACK_BOX_WARNING fact with medium confidence
	// 2. Process through governance
	// 3. Verify CRITICAL escalation was created
}

// =============================================================================
// HELPERS
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// BenchmarkGovernanceDecision benchmarks the governance decision logic
func BenchmarkGovernanceDecision(b *testing.B) {
	config := DefaultPipelineConfig()
	confidences := []float64{0.90, 0.85, 0.75, 0.65, 0.50, 0.30}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		confidence := confidences[i%len(confidences)]
		switch {
		case confidence >= config.AutoApproveThreshold:
			_ = "APPROVED"
		case confidence >= config.ReviewThreshold:
			_ = "PENDING_REVIEW"
		default:
			_ = "REJECTED"
		}
	}
}

// BenchmarkHashComputation benchmarks content hash computation
func BenchmarkHashComputation(b *testing.B) {
	content := "This is sample content for benchmarking the hash computation function"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computeHash(content)
	}
}
