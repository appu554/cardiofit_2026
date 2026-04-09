package services

import "testing"

func TestCGMCards_HighTBR_OnInsulin_Immediate(t *testing.T) {
	cards := GenerateCGMCards(CGMCardInput{
		SufficientData: true,
		TBRL2Pct:       2.0,
		TBRL1Pct:       6.0,
		OnSUOrInsulin:  true,
		TIRPct:         55,
		CVPct:          30,
	})
	found := false
	for _, c := range cards {
		if c.CardType == CardHypoglycaemiaRisk {
			found = true
			if c.Urgency != UrgencyImmediate {
				t.Errorf("expected IMMEDIATE urgency for hypo on insulin, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected HYPOGLYCAEMIA_RISK card for high TBR + SU/insulin")
	}
}

func TestCGMCards_HighTAR_Urgent(t *testing.T) {
	cards := GenerateCGMCards(CGMCardInput{
		SufficientData: true,
		TARL2Pct:       8.0,
		TIRPct:         60,
		CVPct:          30,
	})
	found := false
	for _, c := range cards {
		if c.CardType == CardSustainedHyperglycaemia {
			found = true
			if c.Urgency != UrgencyUrgent {
				t.Errorf("expected URGENT urgency for sustained hyperglycaemia, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected SUSTAINED_HYPERGLYCAEMIA card for TAR L2 >5%")
	}
}

func TestCGMCards_HighCV_GlucoseVariability(t *testing.T) {
	cards := GenerateCGMCards(CGMCardInput{
		SufficientData: true,
		CVPct:          40,
		TIRPct:         65,
	})
	found := false
	for _, c := range cards {
		if c.CardType == CardGlucoseVariability {
			found = true
		}
	}
	if !found {
		t.Error("expected GLUCOSE_VARIABILITY card for CV >36%")
	}
}

func TestCGMCards_InsufficientData_OnlyQualityCard(t *testing.T) {
	cards := GenerateCGMCards(CGMCardInput{
		SufficientData: false,
		TBRL2Pct:       5.0, // should be ignored
		TARL2Pct:       10.0,
		CVPct:          50,
	})
	if len(cards) != 1 {
		t.Fatalf("expected exactly 1 card for insufficient data, got %d", len(cards))
	}
	if cards[0].CardType != CardCGMDataQuality {
		t.Errorf("expected CGM_DATA_QUALITY card, got %s", cards[0].CardType)
	}
}

func TestCGMCards_WellManaged_NoUrgentCards(t *testing.T) {
	cards := GenerateCGMCards(CGMCardInput{
		SufficientData: true,
		TIRPct:         75,
		TBRL1Pct:       2.0,
		TBRL2Pct:       0.3,
		TARL1Pct:       8.0,
		TARL2Pct:       2.0,
		CVPct:          30,
	})
	for _, c := range cards {
		if c.Urgency == UrgencyImmediate || c.Urgency == UrgencyUrgent {
			t.Errorf("well-managed patient should have no IMMEDIATE/URGENT cards, got %s (%s)", c.Urgency, c.CardType)
		}
	}
}
