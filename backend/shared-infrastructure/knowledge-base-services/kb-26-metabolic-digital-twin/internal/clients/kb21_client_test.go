package clients

import "testing"

func TestMapEngagementToBPPhenotype_DormantToAvoidant(t *testing.T) {
	if got := MapEngagementToBPPhenotype("DORMANT", 0.4); got != "MEASUREMENT_AVOIDANT" {
		t.Errorf("DORMANT should map to MEASUREMENT_AVOIDANT, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_ChurnedToAvoidant(t *testing.T) {
	if got := MapEngagementToBPPhenotype("CHURNED", 0.2); got != "MEASUREMENT_AVOIDANT" {
		t.Errorf("CHURNED should map to MEASUREMENT_AVOIDANT, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_SporadicLowEngagementToCrisis(t *testing.T) {
	if got := MapEngagementToBPPhenotype("SPORADIC", 0.45); got != "CRISIS_ONLY_MEASURER" {
		t.Errorf("SPORADIC + low engagement should map to CRISIS_ONLY_MEASURER, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_SporadicNormalEngagementToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("SPORADIC", 0.6); got != "" {
		t.Errorf("SPORADIC + normal engagement should map to empty, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_ChampionToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("CHAMPION", 0.95); got != "" {
		t.Errorf("CHAMPION should map to empty, got %s", got)
	}
}

func TestMapEngagementToBPPhenotype_UnknownToEmpty(t *testing.T) {
	if got := MapEngagementToBPPhenotype("MYSTERY", 0.5); got != "" {
		t.Errorf("unknown phenotype should map to empty, got %s", got)
	}
}
