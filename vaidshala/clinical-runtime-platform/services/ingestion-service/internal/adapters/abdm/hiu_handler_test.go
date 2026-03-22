package abdm

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cardiofit/ingestion-service/internal/crypto"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// mockConsentStore returns a valid GRANTED consent for any consent ID.
type mockConsentStore struct{}

func (m *mockConsentStore) GetConsent(consentID string) (*crypto.ConsentArtifact, error) {
	return &crypto.ConsentArtifact{
		ConsentID:    consentID,
		PatientID:    "patient-test-001",
		HIURequestID: "hiu-req-001",
		Purpose:      "CAREMGT",
		HITypes:      []string{"OPConsultation"},
		DateFrom:     time.Now().Add(-24 * time.Hour),
		DateTo:       time.Now().Add(24 * time.Hour),
		ExpiresAt:    time.Now().Add(72 * time.Hour),
		Signature:    "valid-test-signature",
		Status:       "GRANTED",
	}, nil
}

func TestHIUHandler_HandleDataPush(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Generate sender (HIP) and receiver (our HIU) key pairs.
	sender, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("sender GenerateKeyPair() error: %v", err)
	}
	receiver, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("receiver GenerateKeyPair() error: %v", err)
	}

	handler := NewHIUHandler(receiver, &mockConsentStore{}, zap.NewNop())

	// Build a minimal FHIR Bundle with one Observation.
	fhirBundle := `{
		"resourceType": "Bundle",
		"entry": [
			{
				"resource": {
					"resourceType": "Observation",
					"code": {"coding": [{"system": "http://loinc.org", "code": "2339-0"}]},
					"valueQuantity": {"value": 95, "unit": "mg/dL"}
				}
			}
		]
	}`

	encrypted, nonce, err := sender.Encrypt([]byte(fhirBundle), receiver.PublicKey)
	if err != nil {
		t.Fatalf("Encrypt() error: %v", err)
	}

	payload := ABDMDataPushPayload{
		TransactionID: "txn-test-001",
		ConsentID:     "consent-test-001",
		KeyMaterial: ABDMKeyMaterial{
			SenderPublicKey: base64.StdEncoding.EncodeToString(sender.PublicKey[:]),
			Nonce:           base64.StdEncoding.EncodeToString(nonce[:]),
		},
		Entries: []ABDMDataEntry{
			{
				CareContextRef: "visit-001",
				Content:        base64.StdEncoding.EncodeToString(encrypted),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload error: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/abdm/data/push", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleDataPush(c)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}

	obsCount, ok := resp["observation_count"].(float64)
	if !ok || int(obsCount) != 1 {
		t.Errorf("expected observation_count=1, got %v", resp["observation_count"])
	}
}

func TestHIUHandler_BadPublicKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	receiver, err := crypto.GenerateKeyPair()
	if err != nil {
		t.Fatalf("receiver GenerateKeyPair() error: %v", err)
	}

	handler := NewHIUHandler(receiver, &mockConsentStore{}, zap.NewNop())

	payload := ABDMDataPushPayload{
		TransactionID: "txn-bad-key",
		ConsentID:     "consent-test-002",
		KeyMaterial: ABDMKeyMaterial{
			SenderPublicKey: "not-valid-base64!!!",
			Nonce:           base64.StdEncoding.EncodeToString(make([]byte, 24)),
		},
		Entries: []ABDMDataEntry{
			{
				CareContextRef: "visit-001",
				Content:        base64.StdEncoding.EncodeToString([]byte("encrypted-data")),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload error: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/v1/abdm/data/push", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.HandleDataPush(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
