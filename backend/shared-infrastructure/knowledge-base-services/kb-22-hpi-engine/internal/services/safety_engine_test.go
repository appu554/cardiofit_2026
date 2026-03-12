package services

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"kb-22-hpi-engine/internal/models"
)

func newTestSafetyEngine() *SafetyEngine {
	return NewSafetyEngine(testLogger(), testMetrics())
}

func TestParseCondition_SimpleAND(t *testing.T) {
	e := newTestSafetyEngine()
	answers := map[string]string{"Q001": "YES", "Q002": "YES", "Q003": "NO"}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"all_true", "Q001=YES AND Q002=YES", true},
		{"one_false", "Q001=YES AND Q003=YES", false},
		{"single_true", "Q001=YES", true},
		{"single_false", "Q003=YES", false},
		{"unanswered", "Q001=YES AND Q999=YES", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.ParseCondition(tt.condition, answers)
			if result != tt.expected {
				t.Errorf("ParseCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestParseCondition_OR(t *testing.T) {
	e := newTestSafetyEngine()
	answers := map[string]string{"Q001": "YES", "Q002": "NO"}

	tests := []struct {
		name      string
		condition string
		expected  bool
	}{
		{"first_true", "Q001=YES OR Q002=YES", true},
		{"second_true", "Q001=NO OR Q002=NO", true},
		{"neither", "Q001=NO OR Q002=YES", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.ParseCondition(tt.condition, answers)
			if result != tt.expected {
				t.Errorf("ParseCondition(%q) = %v, want %v", tt.condition, result, tt.expected)
			}
		})
	}
}

func TestParseCondition_CaseInsensitive(t *testing.T) {
	e := newTestSafetyEngine()
	answers := map[string]string{"Q001": "YES"}

	if !e.ParseCondition("Q001=yes", answers) {
		t.Error("ParseCondition should be case-insensitive for answer values")
	}
}

func TestParseCondition_TripleAND(t *testing.T) {
	e := newTestSafetyEngine()

	// Simulates ST001 from P01: Q001=YES AND Q002=YES AND Q008=YES
	answers := map[string]string{"Q001": "YES", "Q002": "YES", "Q008": "YES"}
	if !e.ParseCondition("Q001=YES AND Q002=YES AND Q008=YES", answers) {
		t.Error("triple AND should be true when all atoms match")
	}

	answers["Q008"] = "NO"
	if e.ParseCondition("Q001=YES AND Q002=YES AND Q008=YES", answers) {
		t.Error("triple AND should be false when one atom is false")
	}
}

func TestEvaluateTriggers_FiresCorrectFlags(t *testing.T) {
	e := newTestSafetyEngine()

	triggers := []models.SafetyTriggerDef{
		{ID: "ST001", Condition: "Q001=YES AND Q002=YES", Severity: "IMMEDIATE", Action: "STAT ECG"},
		{ID: "ST002", Condition: "Q007=YES AND Q009=YES", Severity: "URGENT", Action: "PE workup"},
		{ID: "ST003", Condition: "Q001=YES AND Q013=YES", Severity: "WARN", Action: "Monitor"},
	}

	answers := map[string]string{
		"Q001": "YES",
		"Q002": "YES",
		"Q013": "YES",
	}

	flags := e.EvaluateTriggers(triggers, answers)

	if len(flags) != 2 {
		t.Fatalf("expected 2 fired triggers, got %d", len(flags))
	}

	firedIDs := make(map[string]bool)
	for _, f := range flags {
		firedIDs[f.FlagID] = true
	}

	if !firedIDs["ST001"] {
		t.Error("ST001 should have fired (Q001=YES AND Q002=YES)")
	}
	if firedIDs["ST002"] {
		t.Error("ST002 should NOT have fired (Q007 and Q009 not answered)")
	}
	if !firedIDs["ST003"] {
		t.Error("ST003 should have fired (Q001=YES AND Q013=YES)")
	}
}

func TestEvaluateTriggers_NoFlagsWhenNoMatch(t *testing.T) {
	e := newTestSafetyEngine()

	triggers := []models.SafetyTriggerDef{
		{ID: "ST001", Condition: "Q001=YES AND Q002=YES", Severity: "IMMEDIATE", Action: "Action"},
	}

	answers := map[string]string{"Q001": "NO", "Q002": "YES"}

	flags := e.EvaluateTriggers(triggers, answers)
	if len(flags) != 0 {
		t.Errorf("expected 0 fired triggers, got %d", len(flags))
	}
}

func TestSafetyEngine_StartGoroutine(t *testing.T) {
	e := newTestSafetyEngine()
	sessionID := uuid.New()

	triggers := []models.SafetyTriggerDef{
		{ID: "ST001", Condition: "Q001=YES AND Q002=YES", Severity: "IMMEDIATE", Action: "STAT ECG"},
	}

	answerChan := make(chan AnswerEvent, 10)
	flagChan := make(chan models.SafetyFlag, 10)

	e.Start(triggers, nil, answerChan, flagChan, sessionID)

	// Send answers that should trigger ST001
	answerChan <- AnswerEvent{QuestionID: "Q001", Answer: "YES", SessionID: sessionID}
	answerChan <- AnswerEvent{QuestionID: "Q002", Answer: "YES", SessionID: sessionID}
	close(answerChan)

	// Wait for flag with timeout
	select {
	case flag := <-flagChan:
		if flag.FlagID != "ST001" {
			t.Errorf("expected ST001, got %s", flag.FlagID)
		}
		if flag.Severity != models.SafetyImmediate {
			t.Errorf("expected IMMEDIATE, got %s", flag.Severity)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for safety flag from goroutine")
	}
}

func TestSafetyEngine_NoDuplicateFlags(t *testing.T) {
	e := newTestSafetyEngine()
	sessionID := uuid.New()

	triggers := []models.SafetyTriggerDef{
		{ID: "ST001", Condition: "Q001=YES", Severity: "WARN", Action: "Monitor"},
	}

	answerChan := make(chan AnswerEvent, 10)
	flagChan := make(chan models.SafetyFlag, 10)

	e.Start(triggers, nil, answerChan, flagChan, sessionID)

	// Send the triggering answer twice — flag should only fire once
	answerChan <- AnswerEvent{QuestionID: "Q001", Answer: "YES", SessionID: sessionID}
	answerChan <- AnswerEvent{QuestionID: "Q002", Answer: "YES", SessionID: sessionID}
	close(answerChan)

	time.Sleep(100 * time.Millisecond)

	count := 0
	for len(flagChan) > 0 {
		<-flagChan
		count++
	}
	if count != 1 {
		t.Errorf("expected exactly 1 flag (no duplicates), got %d", count)
	}
}

func TestSafetyEngine_CrossNodeTriggers(t *testing.T) {
	e := newTestSafetyEngine()
	sessionID := uuid.New()

	crossTriggers := []models.CrossNodeTrigger{
		{
			TriggerID:         "XN001",
			Condition:         "Q001=YES AND Q007=YES",
			Severity:          "IMMEDIATE",
			RecommendedAction: "Consider ACS with pulmonary congestion",
			Active:            true,
		},
		{
			TriggerID:         "XN002",
			Condition:         "Q099=YES",
			Severity:          "URGENT",
			RecommendedAction: "Test",
			Active:            false, // inactive
		},
	}

	answerChan := make(chan AnswerEvent, 10)
	flagChan := make(chan models.SafetyFlag, 10)

	e.Start(nil, crossTriggers, answerChan, flagChan, sessionID)

	answerChan <- AnswerEvent{QuestionID: "Q001", Answer: "YES", SessionID: sessionID}
	answerChan <- AnswerEvent{QuestionID: "Q007", Answer: "YES", SessionID: sessionID}
	close(answerChan)

	select {
	case flag := <-flagChan:
		if flag.FlagID != "XN001" {
			t.Errorf("expected XN001, got %s", flag.FlagID)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for cross-node trigger")
	}

	// XN002 is inactive, should not fire
	select {
	case flag := <-flagChan:
		t.Errorf("unexpected flag from inactive trigger: %s", flag.FlagID)
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}
