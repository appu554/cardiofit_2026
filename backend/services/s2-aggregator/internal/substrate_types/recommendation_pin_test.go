package substrate_types

import (
	"reflect"
	"testing"
)

// TestRecommendationPacketFieldPinning pins the field names of
// RecommendationPacket so drift against kb-32 generator.Packet is caught
// at CI time.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/generator/recommendation.go (Packet).
func TestRecommendationPacketFieldPinning(t *testing.T) {
	want := []string{
		"RecommendationID", "AuthorID", "Type", "Sections",
		"AppliedRule", "SnapshotRef",
	}
	got := fieldNames(RecommendationPacket{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("RecommendationPacket fields drifted: want %v got %v\n"+
			"if canonical Packet changed, update local copy + SOURCE OF TRUTH comment",
			want, got)
	}
}

// TestAppliedRuleFieldPinning pins the slim AppliedRule projection.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/reasoning (ApplicableRule).
func TestAppliedRuleFieldPinning(t *testing.T) {
	want := []string{"RuleID", "Type", "Urgency"}
	got := fieldNames(AppliedRule{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("AppliedRule fields drifted: want %v got %v", want, got)
	}
}

// TestAssessmentScoresFieldPinning pins the 5-dimension appropriateness rubric.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/appropriateness/checker.go (Assessment).
func TestAssessmentScoresFieldPinning(t *testing.T) {
	want := []string{
		"ClinicalWarrant", "EvidenceSolidity", "AlternativesConsidered",
		"RestraintConsidered", "GoalsOfCareAlignment",
	}
	got := fieldNames(AssessmentScores{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("AssessmentScores fields drifted: want %v got %v", want, got)
	}
}

// TestCitationFieldPinning pins the fire-time citation shape.
//
// SOURCE OF TRUTH: kb-32-recommendation-craft/internal/citations/versioning.go (RecommendationCitation).
func TestCitationFieldPinning(t *testing.T) {
	want := []string{"RecommendationID", "SourceID", "Version", "PinnedAt"}
	got := fieldNames(Citation{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("Citation fields drifted: want %v got %v", want, got)
	}
}
