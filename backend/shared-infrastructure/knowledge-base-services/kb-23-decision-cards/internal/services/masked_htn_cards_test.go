package services

import (
	"strings"
	"testing"

	"kb-23-decision-cards/internal/models"
)

func TestMaskedHTNCards_MaskedHTN_Diabetic_Immediate(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:             models.PhenotypeMaskedHTN,
		ClinicSBPMean:         128,
		ClinicDBPMean:         78,
		HomeSBPMean:           148,
		HomeDBPMean:           92,
		ClinicHomeGapSBP:      -20,
		DiabetesAmplification: true,
		IsDiabetic:            true,
		Confidence:            "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_HYPERTENSION" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency for DM amplification, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "3.2") {
				t.Error("expected rationale to mention 3.2x risk multiplier")
			}
		}
	}
	if !found {
		t.Error("expected MASKED_HYPERTENSION card")
	}
}

func TestMaskedHTNCards_WhiteCoatHTN_AvoidOvertreatment(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:        models.PhenotypeWhiteCoatHTN,
		ClinicSBPMean:    155,
		ClinicDBPMean:    96,
		HomeSBPMean:      125,
		HomeDBPMean:      78,
		ClinicHomeGapSBP: 30,
		WhiteCoatEffect:  30,
		Confidence:       "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "WHITE_COAT_HYPERTENSION" {
			found = true
			if c.Urgency != "ROUTINE" {
				t.Errorf("expected ROUTINE urgency, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "overtreatment") {
				t.Error("expected rationale to mention overtreatment")
			}
		}
	}
	if !found {
		t.Error("expected WHITE_COAT_HYPERTENSION card")
	}
}

func TestMaskedHTNCards_MUCH_TreatedPatient(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:           models.PhenotypeMaskedUncontrolled,
		ClinicSBPMean:       130,
		ClinicDBPMean:       80,
		HomeSBPMean:         150,
		HomeDBPMean:         92,
		OnAntihypertensives: true,
		CKDAmplification:    true,
		Confidence:          "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_UNCONTROLLED" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency, got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "controlled in clinic but not at home") {
				t.Error("expected rationale to mention 'controlled in clinic but not at home'")
			}
		}
	}
	if !found {
		t.Error("expected MASKED_UNCONTROLLED card")
	}
}

func TestMaskedHTNCards_CompoundRisk_MH_MorningSurge(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:            models.PhenotypeMaskedHTN,
		ClinicSBPMean:        128,
		HomeSBPMean:          142,
		MorningSurgeCompound: true,
		Confidence:           "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MASKED_HTN_MORNING_SURGE_COMPOUND" {
			found = true
			if c.Urgency != "IMMEDIATE" {
				t.Errorf("expected IMMEDIATE urgency, got %s", c.Urgency)
			}
		}
	}
	if !found {
		t.Error("expected MASKED_HTN_MORNING_SURGE_COMPOUND card")
	}
}

func TestMaskedHTNCards_SelectionBias_FlagsUncertainty(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:           models.PhenotypeMaskedHTN,
		SelectionBiasRisk:   true,
		Confidence:          "LOW",
		HomeReadingCount:    8,
		EngagementPhenotype: "MEASUREMENT_AVOIDANT",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "SELECTION_BIAS_WARNING" {
			found = true
			if !strings.Contains(strings.ToLower(c.Rationale), "avoidant") ||
				!strings.Contains(strings.ToLower(c.Rationale), "measurement") {
				t.Error("expected rationale to reference measurement avoidant behaviour")
			}
		}
	}
	if !found {
		t.Error("expected SELECTION_BIAS_WARNING card")
	}
}

func TestMaskedHTNCards_MedicationTiming(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:                  models.PhenotypeMaskedUncontrolled,
		OnAntihypertensives:        true,
		MedicationTimingHypothesis: "Morning BP significantly higher than evening — consider evening dosing",
		Confidence:                 "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "MEDICATION_TIMING" {
			found = true
			if !strings.Contains(c.Rationale, "evening dosing") {
				t.Error("expected rationale to mention evening dosing")
			}
		}
	}
	if !found {
		t.Error("expected MEDICATION_TIMING card")
	}
}

func TestMaskedHTNCards_Normotensive_NoUrgentCards(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:  models.PhenotypeSustainedNormotension,
		Confidence: "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	for _, c := range cards {
		if c.Urgency == "IMMEDIATE" || c.Urgency == "URGENT" {
			t.Errorf("normotensive patient should not receive urgent cards, got %s (%s)", c.Urgency, c.CardType)
		}
	}
}

func TestMaskedHTNCards_WhiteCoatUncontrolled_AvoidEscalation(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:           models.PhenotypeWhiteCoatUncontrolled,
		ClinicSBPMean:       150,
		ClinicDBPMean:       92,
		HomeSBPMean:         128,
		HomeDBPMean:         80,
		WhiteCoatEffect:     22,
		OnAntihypertensives: true,
		Confidence:          "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "WHITE_COAT_UNCONTROLLED" {
			found = true
			if c.Urgency != "ROUTINE" {
				t.Errorf("expected ROUTINE urgency, got %s", c.Urgency)
			}
			// Must warn against escalation
			foundEscalationWarning := false
			for _, a := range c.Actions {
				if strings.Contains(a, "NOT escalate") || strings.Contains(a, "REDUCING") {
					foundEscalationWarning = true
				}
			}
			if !foundEscalationWarning {
				t.Error("expected action warning against escalation or recommending dose reduction")
			}
		}
	}
	if !found {
		t.Error("expected WHITE_COAT_UNCONTROLLED card")
	}
}

func TestMaskedHTNCards_SustainedHTN_MorningSurge(t *testing.T) {
	classification := &models.BPContextClassification{
		Phenotype:            models.PhenotypeSustainedHTN,
		ClinicSBPMean:        155,
		ClinicDBPMean:        94,
		HomeSBPMean:          148,
		HomeDBPMean:          90,
		MorningSurgeCompound: true,
		Confidence:           "HIGH",
	}

	cards := EvaluateMaskedHTNCards(classification)
	found := false
	for _, c := range cards {
		if c.CardType == "SUSTAINED_HTN_MORNING_SURGE" {
			found = true
			if c.Urgency != "URGENT" {
				t.Errorf("expected URGENT urgency (not IMMEDIATE — both contexts already elevated), got %s", c.Urgency)
			}
			if !strings.Contains(c.Rationale, "morning surge") {
				t.Error("expected rationale to mention morning surge")
			}
		}
	}
	if !found {
		t.Error("expected SUSTAINED_HTN_MORNING_SURGE card")
	}
}
