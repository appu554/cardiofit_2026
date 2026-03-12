package kb4_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"kb-1-drug-rules/pkg/kb4"
)

// TestClientCheck tests the KB-4 safety check client
func TestClientCheck(t *testing.T) {
	// Create mock KB-4 server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/check" {
			t.Errorf("Expected path /v1/check, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Return mock safety response
		response := kb4.SafetyCheckResponse{
			Safe:             true,
			RequiresAction:   false,
			BlockPrescribing: false,
			TotalAlerts:      1,
			CriticalAlerts:   0,
			HighAlerts:       1,
			IsHighAlertDrug:  true,
			CheckedAt:        time.Now(),
			RequestID:        "test-request-123",
			Alerts: []kb4.SafetyAlert{
				{
					Type:     kb4.AlertTypeHighAlert,
					Severity: kb4.SeverityHigh,
					Title:    "High-Alert Medication",
					Message:  "Warfarin is an ISMP High-Alert Medication",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create client pointing to mock server
	log := logrus.NewEntry(logrus.New())
	client := kb4.NewClient(mockServer.URL, log)

	// Test safety check
	req := &kb4.SafetyCheckRequest{
		Drug: kb4.DrugInfo{
			RxNormCode: "11289",
			DrugName:   "Warfarin",
		},
		ProposedDose: 5.0,
		DoseUnit:     "mg",
		Frequency:    "DAILY",
		Route:        "PO",
		Patient: kb4.PatientContext{
			Age:      65,
			AgeUnit:  "years",
			WeightKg: 70,
			Gender:   "M",
			EGFR:     60,
		},
	}

	resp, err := client.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Verify response
	if !resp.Safe {
		t.Error("Expected Safe=true")
	}
	if resp.BlockPrescribing {
		t.Error("Expected BlockPrescribing=false")
	}
	if !resp.IsHighAlertDrug {
		t.Error("Expected IsHighAlertDrug=true")
	}
	if resp.TotalAlerts != 1 {
		t.Errorf("Expected TotalAlerts=1, got %d", resp.TotalAlerts)
	}
	if resp.HighAlerts != 1 {
		t.Errorf("Expected HighAlerts=1, got %d", resp.HighAlerts)
	}
}

// TestClientCheckBlocked tests KB-4 blocking a prescription
func TestClientCheckBlocked(t *testing.T) {
	// Create mock KB-4 server that returns a blocked prescription
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := kb4.SafetyCheckResponse{
			Safe:             false,
			RequiresAction:   true,
			BlockPrescribing: true,
			TotalAlerts:      1,
			CriticalAlerts:   1,
			CheckedAt:        time.Now(),
			Alerts: []kb4.SafetyAlert{
				{
					Type:                   kb4.AlertTypeContraindication,
					Severity:               kb4.SeverityCritical,
					Title:                  "Absolute Contraindication",
					Message:                "Metformin contraindicated in severe renal impairment (eGFR < 30)",
					RequiresAcknowledgment: true,
					CanOverride:            false,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	log := logrus.NewEntry(logrus.New())
	client := kb4.NewClient(mockServer.URL, log)

	req := &kb4.SafetyCheckRequest{
		Drug: kb4.DrugInfo{
			RxNormCode: "6809",
			DrugName:   "Metformin",
		},
		Patient: kb4.PatientContext{
			Age:  70,
			EGFR: 25, // Severe renal impairment
		},
	}

	resp, err := client.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	// Verify blocked response
	if resp.Safe {
		t.Error("Expected Safe=false for contraindication")
	}
	if !resp.BlockPrescribing {
		t.Error("Expected BlockPrescribing=true for absolute contraindication")
	}
	if resp.CriticalAlerts != 1 {
		t.Errorf("Expected CriticalAlerts=1, got %d", resp.CriticalAlerts)
	}
}

// TestClientDisabled tests that disabled client returns safe verdict
func TestClientDisabled(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := kb4.Config{
		BaseURL: "http://localhost:8088",
		Enabled: false,
	}
	client := kb4.NewClientWithConfig(cfg, log)

	if client.IsEnabled() {
		t.Error("Expected client to be disabled")
	}

	req := &kb4.SafetyCheckRequest{
		Drug: kb4.DrugInfo{RxNormCode: "11289"},
	}

	resp, err := client.Check(context.Background(), req)
	if err != nil {
		t.Fatalf("Check should not fail when disabled: %v", err)
	}

	// Disabled client should return safe (pass-through)
	if !resp.Safe {
		t.Error("Disabled client should return Safe=true")
	}
}

// TestSafetyVerdictConversion tests converting response to verdict
func TestSafetyVerdictConversion(t *testing.T) {
	resp := &kb4.SafetyCheckResponse{
		Safe:             false,
		BlockPrescribing: true,
		RequiresAction:   true,
		IsHighAlertDrug:  true,
		TotalAlerts:      3,
		CriticalAlerts:   1,
		HighAlerts:       2,
		CheckedAt:        time.Now(),
		RequestID:        "test-123",
		Alerts: []kb4.SafetyAlert{
			{Type: kb4.AlertTypeBlackBox, Severity: kb4.SeverityCritical},
			{Type: kb4.AlertTypeHighAlert, Severity: kb4.SeverityHigh},
			{Type: kb4.AlertTypeBeers, Severity: kb4.SeverityHigh},
		},
	}

	verdict := resp.ToVerdict()

	if verdict.Safe != resp.Safe {
		t.Error("Verdict.Safe mismatch")
	}
	if verdict.BlockPrescribing != resp.BlockPrescribing {
		t.Error("Verdict.BlockPrescribing mismatch")
	}
	if verdict.TotalAlerts != resp.TotalAlerts {
		t.Error("Verdict.TotalAlerts mismatch")
	}
	if len(verdict.Alerts) != len(resp.Alerts) {
		t.Error("Verdict.Alerts count mismatch")
	}
	if verdict.KB4RequestID != resp.RequestID {
		t.Error("Verdict.KB4RequestID mismatch")
	}
}

// TestClientHealth tests the health check endpoint
func TestClientHealth(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "healthy"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mockServer.Close()

	log := logrus.NewEntry(logrus.New())
	client := kb4.NewClient(mockServer.URL, log)

	err := client.Health(context.Background())
	if err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}
