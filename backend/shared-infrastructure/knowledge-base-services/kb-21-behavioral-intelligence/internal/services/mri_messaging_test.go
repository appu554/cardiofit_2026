package services

import "testing"

func TestGenerateMRIMessage_Improved(t *testing.T) {
	msg := GenerateMRIMessage(49.0, 58.0, "MILD_DYSREGULATION", "Glucose Control")
	if msg.Type != MRIMessageImproved {
		t.Errorf("expected IMPROVED, got %s", msg.Type)
	}
	if msg.Tone != "Sathi" {
		t.Errorf("expected Sathi tone, got %s", msg.Tone)
	}
}

func TestGenerateMRIMessage_CrossedModerate(t *testing.T) {
	msg := GenerateMRIMessage(55.0, 48.0, "MODERATE_DETERIORATION", "Behavioral Metabolism")
	if msg.Type != MRIMessageModerate {
		t.Errorf("expected CROSSED_MODERATE, got %s", msg.Type)
	}
	if msg.Tone != "Vaidya" {
		t.Errorf("expected Vaidya tone, got %s", msg.Tone)
	}
}

func TestGenerateMRIMessage_Stable(t *testing.T) {
	msg := GenerateMRIMessage(50.0, 48.0, "MILD_DYSREGULATION", "")
	if msg.Type != MRIMessageStable {
		t.Errorf("expected STABLE, got %s", msg.Type)
	}
}

func TestGenerateMRIMessage_HighNoMessage(t *testing.T) {
	msg := GenerateMRIMessage(82.0, 70.0, "HIGH_DETERIORATION", "Glucose Control")
	if msg.Type != MRIMessageHigh {
		t.Errorf("expected CROSSED_HIGH, got %s", msg.Type)
	}
	if msg.MessageEN != "" {
		t.Error("HIGH_DETERIORATION should have no patient message")
	}
}
