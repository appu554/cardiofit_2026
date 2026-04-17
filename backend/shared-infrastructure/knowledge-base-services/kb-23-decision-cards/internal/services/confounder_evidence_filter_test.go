package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterIOR_HighConfidenceOnly(t *testing.T) {
	outcomes := []IOROutcomeResult{
		{DeltaValue: -0.8, ConfidenceLevel: "HIGH"},
		{DeltaValue: -0.5, ConfidenceLevel: "MODERATE"},
		{DeltaValue: -1.2, ConfidenceLevel: "LOW"},
		{DeltaValue: -0.9, ConfidenceLevel: "HIGH"},
	}
	filtered := FilterIORByConfidence(outcomes, "HIGH")
	assert.Len(t, filtered, 2)
}

func TestFilterIOR_ModerateAndAbove(t *testing.T) {
	outcomes := []IOROutcomeResult{
		{DeltaValue: -0.8, ConfidenceLevel: "HIGH"},
		{DeltaValue: -0.5, ConfidenceLevel: "MODERATE"},
		{DeltaValue: -1.2, ConfidenceLevel: "LOW"},
	}
	filtered := FilterIORByConfidence(outcomes, "MODERATE")
	assert.Len(t, filtered, 2)
}

func TestFilterIOR_AllOutcomes(t *testing.T) {
	outcomes := []IOROutcomeResult{
		{DeltaValue: -0.8, ConfidenceLevel: "HIGH"},
		{DeltaValue: -0.5, ConfidenceLevel: "MODERATE"},
		{DeltaValue: -1.2, ConfidenceLevel: "LOW"},
	}
	filtered := FilterIORByConfidence(outcomes, "LOW")
	assert.Len(t, filtered, 3)
}

func TestAnnotateEvidence_HighQuality(t *testing.T) {
	result := AnnotateEvidenceWithConfounderContext("Median delta -0.8", 10, 8, 1, 1)
	assert.Contains(t, result, "HIGH")
}

func TestAnnotateEvidence_ModerateQuality(t *testing.T) {
	result := AnnotateEvidenceWithConfounderContext("Median delta -0.8", 10, 5, 3, 2)
	assert.Contains(t, result, "MODERATE")
}

func TestAnnotateEvidence_LowQuality(t *testing.T) {
	result := AnnotateEvidenceWithConfounderContext("Median delta -0.8", 10, 2, 3, 5)
	assert.Contains(t, result, "LOW")
	assert.Contains(t, result, "caution")
}

func TestAnnotateEvidence_EmptyOutcomes(t *testing.T) {
	result := AnnotateEvidenceWithConfounderContext("No data", 0, 0, 0, 0)
	assert.Equal(t, "No data", result)
}
