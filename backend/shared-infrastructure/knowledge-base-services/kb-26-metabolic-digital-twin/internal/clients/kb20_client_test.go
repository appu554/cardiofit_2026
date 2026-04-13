package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestKB20Client_FetchBPReadings_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/patient/p1/bp-readings" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.URL.Query().Get("since") == "" {
			t.Error("expected 'since' query parameter")
		}
		resp := map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{
					"patient_id":  "p1",
					"sbp":         142.0,
					"dbp":         88.0,
					"source":      "HOME_CUFF",
					"measured_at": "2026-04-01T09:00:00Z",
				},
				{
					"patient_id":  "p1",
					"sbp":         128.0,
					"dbp":         78.0,
					"source":      "CLINIC",
					"measured_at": "2026-04-05T14:30:00Z",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewKB20Client(server.URL, 1*time.Second, zap.NewNop())
	since := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	readings, err := client.FetchBPReadings(context.Background(), "p1", since)
	if err != nil {
		t.Fatalf("FetchBPReadings failed: %v", err)
	}
	if len(readings) != 2 {
		t.Fatalf("expected 2 readings, got %d", len(readings))
	}
	if readings[0].SBP != 142 || readings[0].DBP != 88 {
		t.Errorf("first reading: expected 142/88, got %.0f/%.0f", readings[0].SBP, readings[0].DBP)
	}
	if readings[0].Source != "HOME_CUFF" {
		t.Errorf("first reading: expected source HOME_CUFF, got %s", readings[0].Source)
	}
	if readings[1].Source != "CLINIC" {
		t.Errorf("second reading: expected source CLINIC, got %s", readings[1].Source)
	}
}

func TestKB20Client_FetchBPReadings_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": true,
			"data":    []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewKB20Client(server.URL, 1*time.Second, zap.NewNop())
	readings, err := client.FetchBPReadings(context.Background(), "p1", time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("FetchBPReadings failed: %v", err)
	}
	if len(readings) != 0 {
		t.Errorf("expected empty slice, got %d", len(readings))
	}
}

func TestKB20Client_FetchBPReadings_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB20Client(server.URL, 1*time.Second, zap.NewNop())
	_, err := client.FetchBPReadings(context.Background(), "p1", time.Now().AddDate(0, 0, -30))
	if err == nil {
		t.Error("expected error on 500")
	}
}

func TestKB20Client_FetchBPReadings_NetworkError(t *testing.T) {
	client := NewKB20Client("http://127.0.0.1:1", 100*time.Millisecond, zap.NewNop())
	_, err := client.FetchBPReadings(context.Background(), "p1", time.Now().AddDate(0, 0, -30))
	if err == nil {
		t.Error("expected error from unreachable server")
	}
}
