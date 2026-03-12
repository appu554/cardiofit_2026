package rules

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// DRAFT RULE CREATION TESTS
// =============================================================================

func TestNewDraftRule_CreatesValidRule(t *testing.T) {
	condition := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	action := Action{
		Effect: EffectContraindicated,
	}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
		DocumentID:       "metformin-001",
		SectionCode:      "34068-7",
		ExtractionMethod: "TABLE_PARSE",
	}

	rule := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance)

	if rule.RuleID == uuid.Nil {
		t.Error("Expected RuleID to be generated")
	}

	if rule.Domain != "KB-1" {
		t.Errorf("Expected domain KB-1, got %s", rule.Domain)
	}

	if rule.RuleType != RuleTypeDosing {
		t.Errorf("Expected RuleType DOSING, got %s", rule.RuleType)
	}

	if rule.GovernanceStatus != GovernanceStatusDraft {
		t.Errorf("Expected GovernanceStatus DRAFT, got %s", rule.GovernanceStatus)
	}

	if rule.SemanticFingerprint.Hash == "" {
		t.Error("Expected fingerprint to be computed")
	}
}

func TestNewDosingRule_CreatesDosingRule(t *testing.T) {
	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	rule := NewDosingRule(
		"renal_function.crcl",
		OpBetween,
		30, 60,
		"ml/min",
		EffectDoseAdjust,
		&DoseAdjustment{
			Type:      AdjustTypeReduce,
			Magnitude: 50,
			Unit:      "percent",
		},
		provenance,
	)

	if rule.Condition.Variable != "renal_function.crcl" {
		t.Errorf("Expected variable renal_function.crcl, got %s", rule.Condition.Variable)
	}

	if rule.Condition.Operator != OpBetween {
		t.Errorf("Expected operator BETWEEN, got %s", rule.Condition.Operator)
	}

	if rule.Condition.MinValue == nil || *rule.Condition.MinValue != 30 {
		t.Error("Expected MinValue 30")
	}

	if rule.Condition.MaxValue == nil || *rule.Condition.MaxValue != 60 {
		t.Error("Expected MaxValue 60")
	}

	if rule.Action.Effect != EffectDoseAdjust {
		t.Errorf("Expected effect DOSE_ADJUST, got %s", rule.Action.Effect)
	}
}

func TestNewContraindicationRule_CreatesContraindicationRule(t *testing.T) {
	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	rule := NewContraindicationRule(
		"renal_function.crcl",
		OpLessThan,
		ptrFloat64(30),
		"ml/min",
		provenance,
	)

	if rule.RuleType != RuleTypeContraindication {
		t.Errorf("Expected RuleType CONTRAINDICATION, got %s", rule.RuleType)
	}

	if rule.Domain != "KB-4" {
		t.Errorf("Expected domain KB-4, got %s", rule.Domain)
	}

	if rule.Action.Effect != EffectContraindicated {
		t.Errorf("Expected effect CONTRAINDICATED, got %s", rule.Action.Effect)
	}
}

func TestNewInteractionRule_CreatesInteractionRule(t *testing.T) {
	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	rule := NewInteractionRule(
		"interacting_drug",
		OpEquals,
		"warfarin",
		EffectAvoid,
		map[string]string{"severity": "major"},
		provenance,
	)

	if rule.RuleType != RuleTypeInteraction {
		t.Errorf("Expected RuleType INTERACTION, got %s", rule.RuleType)
	}

	if rule.Domain != "KB-5" {
		t.Errorf("Expected domain KB-5, got %s", rule.Domain)
	}

	if rule.Condition.StringValue == nil || *rule.Condition.StringValue != "warfarin" {
		t.Error("Expected string value warfarin")
	}
}

// =============================================================================
// FINGERPRINT TESTS
// =============================================================================

func TestComputeFingerprint_SameInputSameHash(t *testing.T) {
	condition := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	action := Action{
		Effect: EffectContraindicated,
	}

	provenance1 := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
		DocumentID:       "doc-001",
	}

	provenance2 := Provenance{
		SourceDocumentID: uuid.New(), // Different source
		SourceType:       "CPIC",      // Different type
		DocumentID:       "doc-002",   // Different document
	}

	rule1 := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance1)
	rule2 := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance2)

	// Same condition and action should produce same fingerprint
	// regardless of provenance
	if rule1.SemanticFingerprint.Hash != rule2.SemanticFingerprint.Hash {
		t.Errorf("Expected same fingerprint for semantically identical rules\nRule1: %s\nRule2: %s",
			rule1.SemanticFingerprint.Hash, rule2.SemanticFingerprint.Hash)
	}
}

func TestComputeFingerprint_DifferentConditionDifferentHash(t *testing.T) {
	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	action := Action{Effect: EffectContraindicated}

	condition1 := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	condition2 := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(15), // Different threshold
		Unit:     "ml/min",
	}

	rule1 := NewDraftRule("KB-1", RuleTypeDosing, condition1, action, provenance)
	rule2 := NewDraftRule("KB-1", RuleTypeDosing, condition2, action, provenance)

	if rule1.SemanticFingerprint.Hash == rule2.SemanticFingerprint.Hash {
		t.Error("Expected different fingerprints for different thresholds")
	}
}

func TestComputeFingerprint_DifferentDomainDifferentHash(t *testing.T) {
	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	condition := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	action := Action{Effect: EffectContraindicated}

	rule1 := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance)
	rule2 := NewDraftRule("KB-4", RuleTypeDosing, condition, action, provenance) // Different domain

	if rule1.SemanticFingerprint.Hash == rule2.SemanticFingerprint.Hash {
		t.Error("Expected different fingerprints for different domains")
	}
}

func TestComputeFingerprint_HashIsDeterministic(t *testing.T) {
	condition := Condition{
		Variable: "renal_function.crcl",
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
		Unit:     "ml/min",
	}

	action := Action{Effect: EffectContraindicated}

	provenance := Provenance{
		SourceDocumentID: uuid.New(),
		SourceType:       "FDA_SPL",
	}

	hashes := make(map[string]bool)

	// Generate the same rule 100 times
	for i := 0; i < 100; i++ {
		rule := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance)
		hashes[rule.SemanticFingerprint.Hash] = true
	}

	// All hashes should be the same
	if len(hashes) != 1 {
		t.Errorf("Expected 1 unique hash, got %d different hashes", len(hashes))
	}
}

func TestComputeFingerprint_Version(t *testing.T) {
	condition := Condition{Variable: "test", Operator: OpEquals, Value: ptrFloat64(1), Unit: "unit"}
	action := Action{Effect: EffectAvoid}
	provenance := Provenance{SourceDocumentID: uuid.New()}

	rule := NewDraftRule("KB-1", RuleTypeDosing, condition, action, provenance)

	if rule.SemanticFingerprint.Version != 1 {
		t.Errorf("Expected fingerprint version 1, got %d", rule.SemanticFingerprint.Version)
	}

	if rule.SemanticFingerprint.Algorithm != "SHA256" {
		t.Errorf("Expected algorithm SHA256, got %s", rule.SemanticFingerprint.Algorithm)
	}
}

// =============================================================================
// CONDITION EVALUATION TESTS
// =============================================================================

func TestCondition_Evaluate_LessThan(t *testing.T) {
	condition := Condition{
		Operator: OpLessThan,
		Value:    ptrFloat64(30),
	}

	// Should match: 29 < 30
	if !condition.Evaluate(ptrFloat64(29), nil) {
		t.Error("Expected 29 < 30 to be true")
	}

	// Should not match: 30 < 30
	if condition.Evaluate(ptrFloat64(30), nil) {
		t.Error("Expected 30 < 30 to be false")
	}

	// Should not match: 31 < 30
	if condition.Evaluate(ptrFloat64(31), nil) {
		t.Error("Expected 31 < 30 to be false")
	}
}

func TestCondition_Evaluate_Between(t *testing.T) {
	condition := Condition{
		Operator: OpBetween,
		MinValue: ptrFloat64(30),
		MaxValue: ptrFloat64(60),
	}

	// Should match: 30 <= 45 < 60
	if !condition.Evaluate(ptrFloat64(45), nil) {
		t.Error("Expected 45 to be between 30 and 60")
	}

	// Should match: 30 <= 30 < 60 (inclusive lower bound)
	if !condition.Evaluate(ptrFloat64(30), nil) {
		t.Error("Expected 30 to be between 30 and 60 (inclusive lower)")
	}

	// Should not match: 60 is at upper bound (exclusive)
	if condition.Evaluate(ptrFloat64(60), nil) {
		t.Error("Expected 60 to NOT be between 30 and 60 (exclusive upper)")
	}

	// Should not match: 29 < 30
	if condition.Evaluate(ptrFloat64(29), nil) {
		t.Error("Expected 29 to NOT be between 30 and 60")
	}
}

func TestCondition_Evaluate_StringEquals(t *testing.T) {
	condition := Condition{
		Operator:    OpEquals,
		StringValue: ptrString("A"),
	}

	// Should match
	if !condition.Evaluate(nil, ptrString("A")) {
		t.Error("Expected 'A' == 'A' to be true")
	}

	// Should not match
	if condition.Evaluate(nil, ptrString("B")) {
		t.Error("Expected 'B' == 'A' to be false")
	}
}

func TestCondition_Evaluate_In(t *testing.T) {
	condition := Condition{
		Operator:   OpIn,
		ListValues: []string{"A", "B", "C"},
	}

	// Should match
	if !condition.Evaluate(nil, ptrString("B")) {
		t.Error("Expected 'B' to be IN [A, B, C]")
	}

	// Should not match
	if condition.Evaluate(nil, ptrString("D")) {
		t.Error("Expected 'D' to NOT be IN [A, B, C]")
	}
}

// =============================================================================
// GOVERNANCE STATUS TESTS
// =============================================================================

func TestGovernanceStatus_Transitions(t *testing.T) {
	tests := []struct {
		name   string
		status GovernanceStatus
		valid  bool
	}{
		{"draft", GovernanceStatusDraft, true},
		{"review", GovernanceStatusReview, true},
		{"approved", GovernanceStatusApproved, true},
		{"active", GovernanceStatusActive, true},
		{"rejected", GovernanceStatusRejected, true},
		{"superseded", GovernanceStatusSuperseded, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.status == "" {
				t.Error("Expected status to be non-empty")
			}
		})
	}
}

// =============================================================================
// DOSE ADJUSTMENT TESTS
// =============================================================================

func TestDoseAdjustment_ReducePercent(t *testing.T) {
	adj := DoseAdjustment{
		Type:      AdjustTypeReduce,
		Magnitude: 50,
		Unit:      "percent",
	}

	if adj.Type != AdjustTypeReduce {
		t.Errorf("Expected type REDUCE, got %s", adj.Type)
	}

	if adj.Magnitude != 50 {
		t.Errorf("Expected magnitude 50, got %f", adj.Magnitude)
	}
}

func TestDoseAdjustment_MaxDose(t *testing.T) {
	adj := DoseAdjustment{
		Type:      AdjustTypeMaxDose,
		Magnitude: 1000,
		Unit:      "mg/day",
	}

	if adj.Type != AdjustTypeMaxDose {
		t.Errorf("Expected type MAX_DOSE, got %s", adj.Type)
	}
}

func TestDoseAdjustment_Interval(t *testing.T) {
	adj := DoseAdjustment{
		Type:     AdjustTypeInterval,
		Interval: "every 48 hours",
	}

	if adj.Interval != "every 48 hours" {
		t.Errorf("Expected interval 'every 48 hours', got %s", adj.Interval)
	}
}

// =============================================================================
// PROVENANCE TESTS
// =============================================================================

func TestProvenance_FullLineage(t *testing.T) {
	docID := uuid.New()
	sectionID := uuid.New()

	provenance := Provenance{
		SourceDocumentID: docID,
		SourceSectionID:  &sectionID,
		SourceType:       "FDA_SPL",
		DocumentID:       "metformin-001",
		SectionCode:      "34068-7",
		TableID:          "table-001",
		EvidenceSpan:     "CrCl < 30 mL/min: Contraindicated",
		ExtractionMethod: "TABLE_PARSE",
		Confidence:       0.95,
		ExtractedAt:      time.Now(),
	}

	if provenance.SourceDocumentID == uuid.Nil {
		t.Error("Expected SourceDocumentID to be set")
	}

	if provenance.SourceSectionID == nil || *provenance.SourceSectionID == uuid.Nil {
		t.Error("Expected SourceSectionID to be set")
	}

	if provenance.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", provenance.Confidence)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrString(s string) *string {
	return &s
}
