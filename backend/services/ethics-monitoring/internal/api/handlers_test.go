package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeOrch struct{ n int }

func (f fakeOrch) JobCount() int { return f.n }

func TestHealthz_ReturnsExpectedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(fakeOrch{n: 4}).Register(r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status=%v, want ok", body["status"])
	}
	if body["version"] != Version {
		t.Errorf("version=%v, want %s", body["version"], Version)
	}
	// JSON numbers decode as float64.
	if jobs, ok := body["jobs"].(float64); !ok || int(jobs) != 4 {
		t.Errorf("jobs=%v, want 4", body["jobs"])
	}
}

func TestHealthz_NilOrchSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler(nil).Register(r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}
