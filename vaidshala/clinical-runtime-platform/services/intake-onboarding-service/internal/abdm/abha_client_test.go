package abdm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func abdmTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v3/token/generate":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"accessToken": "test-token",
				"expiresIn":   3600,
			})
		case "/api/v3/enrollment/request/otp":
			json.NewEncoder(w).Encode(map[string]string{"txnId": "txn-123"})
		case "/api/v3/enrollment/enrol/byAadhaar":
			json.NewEncoder(w).Encode(ABHAAccount{
				ABHANumber:  "91-1234-5678-9012",
				ABHAAddress: "patient@abdm",
				Name:        "Test Patient",
			})
		case "/api/v3/profile/link/init":
			json.NewEncoder(w).Encode(map[string]string{"txnId": "link-txn-456"})
		case "/api/v3/profile/link/confirm":
			json.NewEncoder(w).Encode(ABHAProfile{
				ABHANumber:  "91-1234-5678-9012",
				ABHAAddress: "patient@abdm",
				Name:        "Test Patient",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestInitAadhaarOTP(t *testing.T) {
	srv := abdmTestServer(t)
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	client := NewABHAClient(ABHAConfig{
		BaseURL:      srv.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
	}, logger)

	txnID, err := client.InitAadhaarOTP("123456789012")
	if err != nil {
		t.Fatalf("InitAadhaarOTP failed: %v", err)
	}
	if txnID != "txn-123" {
		t.Errorf("expected txn-123, got %s", txnID)
	}
}

func TestVerifyAadhaarOTP(t *testing.T) {
	srv := abdmTestServer(t)
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	client := NewABHAClient(ABHAConfig{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"}, logger)

	account, err := client.VerifyAadhaarOTP("txn-123", "123456")
	if err != nil {
		t.Fatalf("VerifyAadhaarOTP failed: %v", err)
	}
	if account.ABHANumber != "91-1234-5678-9012" {
		t.Errorf("expected 91-1234-5678-9012, got %s", account.ABHANumber)
	}
}

func TestLinkABHA(t *testing.T) {
	srv := abdmTestServer(t)
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	client := NewABHAClient(ABHAConfig{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"}, logger)

	txnID, err := client.LinkABHA("91-1234-5678-9012")
	if err != nil {
		t.Fatalf("LinkABHA failed: %v", err)
	}
	if txnID != "link-txn-456" {
		t.Errorf("expected link-txn-456, got %s", txnID)
	}
}

func TestConfirmLinkABHA(t *testing.T) {
	srv := abdmTestServer(t)
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	client := NewABHAClient(ABHAConfig{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"}, logger)

	profile, err := client.ConfirmLinkABHA("link-txn-456", "654321")
	if err != nil {
		t.Fatalf("ConfirmLinkABHA failed: %v", err)
	}
	if profile.ABHANumber != "91-1234-5678-9012" {
		t.Errorf("expected 91-1234-5678-9012, got %s", profile.ABHANumber)
	}
}

func TestEnsureToken_Caching(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/token/generate" {
			callCount++
			json.NewEncoder(w).Encode(map[string]interface{}{
				"accessToken": "cached-token",
				"expiresIn":   3600,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"txnId": "txn"})
	}))
	defer srv.Close()

	logger, _ := zap.NewDevelopment()
	client := NewABHAClient(ABHAConfig{BaseURL: srv.URL, ClientID: "c", ClientSecret: "s"}, logger)

	client.InitAadhaarOTP("111")
	client.InitAadhaarOTP("222")

	if callCount != 1 {
		t.Errorf("expected 1 token call (cached), got %d", callCount)
	}
}
