package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestAttention_90DaysNoClinician_Critical(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		DaysSinceLastClinician: 90,
	}

	score := ComputeAttentionScore(input, cfg)

	if score < 90 {
		t.Errorf("expected score >=90 for 90 days no clinician, got %.2f", score)
	}
}

func TestAttention_RecentReview_Low(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		DaysSinceLastClinician: 5,
		HasUnacknowledgedCards: false,
	}

	score := ComputeAttentionScore(input, cfg)

	if score >= 10 {
		t.Errorf("expected score <10 for recent review with no unacked cards, got %.2f", score)
	}
}

func TestAttention_UnacknowledgedCards(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		DaysSinceLastClinician:   0,
		HasUnacknowledgedCards:   true,
		UnacknowledgedCardCount:  3,
		OldestUnacknowledgedDays: 10,
	}

	score := ComputeAttentionScore(input, cfg)

	// cardScore = min(3*10 + 10*3, 50) = min(60, 50) = 50
	// clinicianScore for 0 days = scaleLinear(0, 0, 14, 0, 10) = 0
	// combined = max(50, 0) + 0.2 * min(50, 0) = 50
	if score < 45 || score > 55 {
		t.Errorf("expected score ~50 for 3 unacked cards oldest 10 days, got %.2f", score)
	}
}

func TestAttention_Combined_High(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{
		DaysSinceLastClinician:   45,
		HasUnacknowledgedCards:   true,
		UnacknowledgedCardCount:  2,
		OldestUnacknowledgedDays: 7,
	}

	score := ComputeAttentionScore(input, cfg)

	// clinicianScore: 45 days in [30,60] -> scaleLinear(45, 30, 60, 30, 60) = 30 + (15/30)*30 = 45
	// cardScore: min(2*10 + 7*3, 50) = min(41, 50) = 41
	// combined = max(45, 41) + 0.2 * min(45, 41) = 45 + 8.2 = 53.2
	if score < 40 {
		t.Errorf("expected elevated score (>=40) for combined attention gaps, got %.2f", score)
	}
	if score > 70 {
		t.Errorf("expected score <=70 for moderate combined gaps, got %.2f", score)
	}
}

func TestAttention_NoData_Zero(t *testing.T) {
	cfg := testPAIConfig()
	input := models.PAIDimensionInput{}

	score := ComputeAttentionScore(input, cfg)

	if score != 0 {
		t.Errorf("expected score 0 for all-zero input, got %.2f", score)
	}
}
