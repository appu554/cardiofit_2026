package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleGetHighRiskCategories(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a minimal server (no DB needed — static endpoint)
	s := &Server{}

	router := gin.New()
	router.GET("/v1/high-risk/categories", s.handleGetHighRiskCategories)

	t.Run("returns 200 with expected structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/high-risk/categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp HighRiskCategoriesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Validate categories
		if len(resp.HighRiskCategories) == 0 {
			t.Fatal("expected non-empty high_risk_categories")
		}

		expectedCategories := map[string]bool{
			"anticoagulants":  true,
			"insulin":         true,
			"opioids":         true,
			"chemotherapy":    true,
			"antiarrhythmics": true,
		}
		for _, cat := range resp.HighRiskCategories {
			if !expectedCategories[cat] {
				t.Errorf("unexpected category: %s", cat)
			}
		}
		if len(resp.HighRiskCategories) != len(expectedCategories) {
			t.Errorf("expected %d categories, got %d", len(expectedCategories), len(resp.HighRiskCategories))
		}

		// Validate drugs
		if len(resp.HighRiskDrugs) == 0 {
			t.Fatal("expected non-empty high_risk_drugs")
		}

		// Check that each drug has required fields
		for _, drug := range resp.HighRiskDrugs {
			if drug.RxNorm == "" {
				t.Error("drug missing rxnorm")
			}
			if drug.Name == "" {
				t.Error("drug missing name")
			}
			if drug.Category == "" {
				t.Error("drug missing category")
			}
			// Category must be one of the known categories
			if !expectedCategories[drug.Category] {
				t.Errorf("drug %s has unknown category: %s", drug.Name, drug.Category)
			}
		}

		// Validate version is present
		if resp.Version == "" {
			t.Error("expected non-empty version")
		}
	})

	t.Run("contains warfarin in drugs list", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/high-risk/categories", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var resp HighRiskCategoriesResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		found := false
		for _, drug := range resp.HighRiskDrugs {
			if drug.RxNorm == "855332" && drug.Name == "warfarin" && drug.Category == "anticoagulants" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected warfarin (rxnorm 855332) in high_risk_drugs")
		}
	})
}
