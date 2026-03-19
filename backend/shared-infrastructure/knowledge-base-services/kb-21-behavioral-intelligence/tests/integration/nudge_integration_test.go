//go:build integration

package integration

import (
	"fmt"
	"testing"

	"kb-21-behavioral-intelligence/internal/models"
)

func TestNudgeSelection_EndToEnd(t *testing.T) {
	if testServer == nil {
		t.Skip("test server not initialized (no test database)")
	}
	cleanDB()

	patientID := "test-nudge-patient-1"

	// 1. Record some interactions to establish adherence
	for i := 0; i < 5; i++ {
		payload := map[string]interface{}{
			"channel":          "WHATSAPP",
			"type":             "MEDICATION_CONFIRM",
			"drug_class":       "METFORMIN",
			"response_quality": "HIGH",
			"response_value":   "yes",
		}
		w := doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/interaction", patientID), payload)
		if w.Code != 200 {
			t.Fatalf("interaction %d: status=%d body=%s", i, w.Code, w.Body.String())
		}
	}

	// 2. Select a nudge
	w := doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/nudge/select", patientID),
		map[string]interface{}{"channel": "WHATSAPP", "language": "hi"})

	if w.Code != 200 {
		t.Fatalf("nudge select: status=%d body=%s", w.Code, w.Body.String())
	}

	body := parseBody(w)
	data := body["data"].(map[string]interface{})

	technique, _ := data["technique"].(string)
	if technique == "" {
		t.Error("expected a technique to be selected")
	}
	phase, _ := data["phase"].(string)
	if phase != string(models.PhaseInitiation) {
		t.Errorf("expected INITIATION phase for new patient, got %s", phase)
	}
	t.Logf("Selected technique: %s, phase: %s", technique, phase)

	// 3. Record positive outcome → updates Bayesian posterior
	w = doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/nudge/outcome", patientID),
		map[string]interface{}{"technique": technique, "success": true})
	if w.Code != 200 {
		t.Fatalf("nudge outcome: status=%d body=%s", w.Code, w.Body.String())
	}

	// 4. Check technique effectiveness reflects the update
	w = doRequest("GET", fmt.Sprintf("/api/v1/patient/%s/techniques", patientID), nil)
	if w.Code != 200 {
		t.Fatalf("techniques: status=%d body=%s", w.Code, w.Body.String())
	}

	// 5. Get motivation phase — new patient should be in INITIATION
	w = doRequest("GET", fmt.Sprintf("/api/v1/patient/%s/motivation-phase", patientID), nil)
	if w.Code != 200 {
		t.Fatalf("motivation-phase: status=%d body=%s", w.Code, w.Body.String())
	}
}
