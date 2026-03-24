package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HighRiskDrug represents a single high-risk medication entry.
type HighRiskDrug struct {
	RxNorm   string `json:"rxnorm"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

// HighRiskCategoriesResponse is the response for GET /v1/high-risk/categories.
type HighRiskCategoriesResponse struct {
	HighRiskCategories []string       `json:"high_risk_categories"`
	HighRiskDrugs      []HighRiskDrug `json:"high_risk_drugs"`
	Version            string         `json:"version"`
}

// Static high-risk medication reference data.
var highRiskCategories = []string{
	"anticoagulants",
	"insulin",
	"opioids",
	"chemotherapy",
	"antiarrhythmics",
}

var highRiskDrugs = []HighRiskDrug{
	{RxNorm: "855332", Name: "warfarin", Category: "anticoagulants"},
	{RxNorm: "311040", Name: "insulin_glargine", Category: "insulin"},
	{RxNorm: "197696", Name: "morphine", Category: "opioids"},
}

const highRiskVersion = "2026-03-23T00:00:00Z"

// handleGetHighRiskCategories returns the static list of high-risk medication
// categories and representative drugs. This is a configuration endpoint that
// does not require database access.
//
// GET /v1/high-risk/categories
func (s *Server) handleGetHighRiskCategories(c *gin.Context) {
	c.JSON(http.StatusOK, HighRiskCategoriesResponse{
		HighRiskCategories: highRiskCategories,
		HighRiskDrugs:      highRiskDrugs,
		Version:            highRiskVersion,
	})
}
