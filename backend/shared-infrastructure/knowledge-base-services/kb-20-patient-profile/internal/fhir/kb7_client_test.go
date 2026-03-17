package fhir

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
)

func TestIsGlucocorticoidFallback_PositiveCases(t *testing.T) {
	glucocorticoids := []string{
		"prednisolone", "prednisone", "methylprednisolone",
		"dexamethasone", "hydrocortisone", "betamethasone", "deflazacort",
	}
	for _, drug := range glucocorticoids {
		if !isGlucocorticoidFallback(drug) {
			t.Errorf("expected %s to be detected as glucocorticoid", drug)
		}
	}
}

func TestIsGlucocorticoidFallback_NegativeCases(t *testing.T) {
	nonGlucocorticoids := []string{"metformin", "lisinopril", "amlodipine", "insulin glargine"}
	for _, drug := range nonGlucocorticoids {
		if isGlucocorticoidFallback(drug) {
			t.Errorf("expected %s to NOT be detected as glucocorticoid", drug)
		}
	}
}

func TestIsSystemicGlucocorticoid_WithMockKB7(t *testing.T) {
	// Mock KB-7 returning H02AB06 (prednisolone)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"code":"prednisolone","atc_class":"Glucocorticoids","atc_code":"H02AB06","class_name":"Prednisolone"}`))
	}))
	defer server.Close()

	client := NewKB7Client(config.KB7Config{BaseURL: server.URL}, zap.NewNop())
	if !client.IsSystemicGlucocorticoid("prednisolone") {
		t.Error("expected prednisolone to be detected as systemic glucocorticoid via KB-7")
	}
}

func TestIsSystemicGlucocorticoid_KB7Unavailable_FallsBack(t *testing.T) {
	// Mock KB-7 returning 500 error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB7Client(config.KB7Config{BaseURL: server.URL}, zap.NewNop())
	if !client.IsSystemicGlucocorticoid("prednisolone") {
		t.Error("expected prednisolone to be detected via fallback when KB-7 unavailable")
	}
	if client.IsSystemicGlucocorticoid("metformin") {
		t.Error("expected metformin to NOT be detected as glucocorticoid via fallback")
	}
}
