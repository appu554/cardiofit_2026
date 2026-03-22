package kafka

import (
	"testing"
)

func TestAllTopics_Count(t *testing.T) {
	topics := AllTopics()
	if len(topics) != 8 {
		t.Errorf("expected 8 intake topics, got %d", len(topics))
	}
}

func TestAllTopics_NamingConvention(t *testing.T) {
	for _, topic := range AllTopics() {
		if len(topic) < 8 || topic[:7] != "intake." {
			t.Errorf("topic %q does not follow intake.* naming convention", topic)
		}
	}
}

func TestTopicConstants(t *testing.T) {
	expected := map[string]bool{
		"intake.patient-lifecycle":  true,
		"intake.slot-events":       true,
		"intake.safety-alerts":     true,
		"intake.safety-flags":      true,
		"intake.completions":       true,
		"intake.checkin-events":    true,
		"intake.session-lifecycle": true,
		"intake.lab-orders":        true,
	}
	for _, topic := range AllTopics() {
		if !expected[topic] {
			t.Errorf("unexpected topic: %s", topic)
		}
	}
}
