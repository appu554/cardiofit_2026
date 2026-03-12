// Package test provides integration tests for KB-12 Order Sets & Care Plans Service
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/internal/models"
	"kb-12-ordersets-careplans/pkg/careplans"
	"kb-12-ordersets-careplans/pkg/cdshooks"
	"kb-12-ordersets-careplans/pkg/cpoe"
	"kb-12-ordersets-careplans/pkg/ordersets"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ============================================
// Order Sets Tests
// ============================================

func TestOrderSetsTemplateCount(t *testing.T) {
	counts := ordersets.GetTemplateCount()
	assert.NotNil(t, counts, "Template counts should not be nil")
	total := 0
	for category, count := range counts {
		t.Logf("Category %s: %d templates", category, count)
		total += count
	}
	t.Logf("Total templates: %d", total)
}

func TestOrderSetsGetAllAdmission(t *testing.T) {
	templates := ordersets.GetAllAdmissionOrderSets()
	assert.NotNil(t, templates, "Admission templates should not be nil")
	t.Logf("Retrieved %d admission order sets", len(templates))
}

func TestOrderSetsGetAllProcedures(t *testing.T) {
	templates := ordersets.GetAllProcedureOrderSets()
	assert.NotNil(t, templates, "Procedure templates should not be nil")
	t.Logf("Retrieved %d procedure order sets", len(templates))
}

func TestOrderSetsGetAllEmergency(t *testing.T) {
	templates := ordersets.GetAllEmergencyProtocols()
	assert.NotNil(t, templates, "Emergency protocols should not be nil")
	t.Logf("Retrieved %d emergency protocols", len(templates))
}

func TestOrderSetCategories(t *testing.T) {
	testCases := []struct {
		name    string
		getFunc func() []*models.OrderSetTemplate
	}{
		{"cardiac_admission", ordersets.GetCardiacAdmissionOrderSets},
		{"respiratory_admission", ordersets.GetRespiratoryAdmissionOrderSets},
		{"metabolic_admission", ordersets.GetMetabolicAdmissionOrderSets},
		{"gi_admission", ordersets.GetGIAdmissionOrderSets},
		{"neuro_admission", ordersets.GetNeuroAdmissionOrderSets},
		{"infectious_admission", ordersets.GetInfectiousAdmissionOrderSets},
		{"cardiac_procedure", ordersets.GetCardiacProcedureOrderSets},
		{"gi_procedure", ordersets.GetGIProcedureOrderSets},
		{"surgical_procedure", ordersets.GetSurgicalProcedureOrderSets},
		{"bedside_procedure", ordersets.GetBedsideProcedureOrderSets},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			templates := tc.getFunc()
			t.Logf("Category %s has %d templates", tc.name, len(templates))
		})
	}
}

// ============================================
// Care Plans Tests
// ============================================

func TestCarePlansGetAll(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	assert.NotNil(t, plans, "Care plans should not be nil")
	t.Logf("Found %d care plans", len(plans))
}

func TestCarePlansCount(t *testing.T) {
	counts := careplans.GetCarePlanCount()
	assert.NotNil(t, counts, "Care plan counts should not be nil")
	total := 0
	for category, count := range counts {
		t.Logf("Care plan category %s: %d", category, count)
		total += count
	}
	assert.GreaterOrEqual(t, total, 0, "Total care plan count should be non-negative")
}

func TestCarePlanCategories(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	categories := make(map[string]int)

	for _, plan := range plans {
		categories[string(plan.Category)]++
	}

	t.Logf("Care plan categories: %v", categories)
}

// ============================================
// CDS Hooks Tests
// ============================================

func TestCDSHooksDiscovery(t *testing.T) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	discovery := service.GetDiscovery()
	require.NotNil(t, discovery, "Discovery response should not be nil")
	assert.NotEmpty(t, discovery.Services, "Should have CDS services")

	// Verify expected hooks
	hookIDs := make([]string, 0)
	for _, svc := range discovery.Services {
		hookIDs = append(hookIDs, svc.ID)
		t.Logf("Hook: %s - %s", svc.ID, svc.Title)
	}

	assert.Contains(t, hookIDs, "kb12-patient-view")
	assert.Contains(t, hookIDs, "kb12-order-select")
	assert.Contains(t, hookIDs, "kb12-order-sign")
	assert.Contains(t, hookIDs, "kb12-encounter-start")
	assert.Contains(t, hookIDs, "kb12-encounter-discharge")
}

func TestCDSHooksPatientView(t *testing.T) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-instance-1",
		Context: map[string]interface{}{
			"patientId": "test-patient-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "Condition",
							"code": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://hl7.org/fhir/sid/icd-10",
										"code":    "I50.9",
										"display": "Heart failure, unspecified",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err, "ProcessHook should not error")
	require.NotNil(t, resp, "Response should not be nil")
	t.Logf("Patient view returned %d cards", len(resp.Cards))
}

func TestCDSHooksEncounterStart(t *testing.T) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-start",
		HookInstance: "test-encounter-1",
		Context: map[string]interface{}{
			"patientId":   "test-patient-456",
			"encounterId": "test-encounter-789",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"class": map[string]interface{}{
					"code": "IMP",
				},
			},
			"conditions": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "Condition",
							"code": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"code":    "A41.9",
										"display": "Sepsis, unspecified organism",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-encounter-start", req)
	require.NoError(t, err, "ProcessHook should not error")
	require.NotNil(t, resp, "Response should not be nil")
	t.Logf("Encounter start returned %d cards", len(resp.Cards))
}

// ============================================
// CPOE Tests
// ============================================

func TestCPOEServiceCreation(t *testing.T) {
	// Create CPOE service without clients (nil-safe)
	service := cpoe.NewCPOEService(nil, nil, nil, nil)
	assert.NotNil(t, service, "CPOE service should be created")
}

func TestCPOECreateSession(t *testing.T) {
	service := cpoe.NewCPOEService(nil, nil, nil, nil)

	req := &cpoe.CreateSessionRequest{
		PatientID:   "patient-123",
		EncounterID: "encounter-456",
		ProviderID:  "provider-789",
	}

	session, err := service.CreateOrderSession(context.Background(), req)
	require.NoError(t, err, "CreateOrderSession should not error")
	require.NotNil(t, session, "Session should not be nil")
	assert.NotEmpty(t, session.SessionID, "Session should have ID")
	assert.Equal(t, "patient-123", session.PatientID)
	assert.Equal(t, "draft", session.Status)
}

func TestCPOEGetSession(t *testing.T) {
	service := cpoe.NewCPOEService(nil, nil, nil, nil)

	// Create a session first
	req := &cpoe.CreateSessionRequest{
		PatientID:   "patient-123",
		EncounterID: "encounter-456",
		ProviderID:  "provider-789",
	}

	created, err := service.CreateOrderSession(context.Background(), req)
	require.NoError(t, err)

	// Get the session
	retrieved, err := service.GetSession(created.SessionID)
	require.NoError(t, err, "GetSession should not error")
	assert.Equal(t, created.SessionID, retrieved.SessionID)
}

// ============================================
// HTTP API Tests
// ============================================

func setupTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// Health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "kb-12-ordersets-careplans",
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Order sets endpoint
	router.GET("/api/v1/ordersets", func(c *gin.Context) {
		// Combine all template types
		templates := make([]*models.OrderSetTemplate, 0)
		templates = append(templates, ordersets.GetAllAdmissionOrderSets()...)
		templates = append(templates, ordersets.GetAllProcedureOrderSets()...)
		templates = append(templates, ordersets.GetAllEmergencyProtocols()...)
		c.JSON(http.StatusOK, gin.H{
			"templates": templates,
			"count":     len(templates),
		})
	})

	// Care plans endpoint
	router.GET("/api/v1/careplans", func(c *gin.Context) {
		plans := careplans.GetAllCarePlans()
		c.JSON(http.StatusOK, gin.H{
			"care_plans": plans,
			"count":      len(plans),
		})
	})

	// CDS Hooks discovery
	loader := ordersets.NewTemplateLoader(nil, nil)
	cdsService := cdshooks.NewCDSHooksService(loader)
	router.GET("/cds-services", func(c *gin.Context) {
		c.JSON(http.StatusOK, cdsService.GetDiscovery())
	})

	return router
}

func TestHealthEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "healthy", body["status"])
	assert.Equal(t, "kb-12-ordersets-careplans", body["service"])
}

func TestOrderSetsEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/ordersets", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Contains(t, body, "templates")
	assert.Contains(t, body, "count")
}

func TestCarePlansEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/careplans", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Contains(t, body, "care_plans")
	assert.Contains(t, body, "count")
}

func TestCDSServicesEndpoint(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/cds-services", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var body cdshooks.DiscoveryResponse
	err := json.Unmarshal(resp.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.NotEmpty(t, body.Services)
}

// ============================================
// Feedback Handler Tests
// ============================================

func TestFeedbackHandler(t *testing.T) {
	handler := cdshooks.NewFeedbackHandler()

	// Record feedback
	feedback := &cdshooks.CardFeedback{
		CardID:         "card-123",
		Outcome:        "accepted",
		UserID:         "user-456",
	}

	err := handler.RecordFeedback(feedback)
	require.NoError(t, err)

	// Retrieve feedback
	retrieved, found := handler.GetFeedback("card-123")
	assert.True(t, found)
	assert.Equal(t, "accepted", retrieved.Outcome)

	// Get stats
	stats := handler.GetStats()
	assert.Equal(t, 1, stats.TotalCards)
	assert.Equal(t, 1, stats.Accepted)
}

func TestFeedbackHandlerOverride(t *testing.T) {
	handler := cdshooks.NewFeedbackHandler()

	feedback := &cdshooks.CardFeedback{
		CardID:         "card-override",
		Outcome:        "overridden",
		OverrideReason: "contraindicated",
		UserID:         "user-789",
	}

	err := handler.RecordFeedback(feedback)
	require.NoError(t, err)

	stats := handler.GetStats()
	assert.Equal(t, 1, stats.Overridden)
	assert.Equal(t, 1, stats.OverrideReasons["contraindicated"])
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkOrderSetsGetAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ordersets.GetAllAdmissionOrderSets()
		_ = ordersets.GetAllProcedureOrderSets()
		_ = ordersets.GetAllEmergencyProtocols()
	}
}

func BenchmarkCarePlansGetAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = careplans.GetAllCarePlans()
	}
}

func BenchmarkCDSHooksDiscovery(b *testing.B) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetDiscovery()
	}
}

func BenchmarkHealthEndpoint(b *testing.B) {
	router := setupTestRouter()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
	}
}

// Helper to suppress unused import warning
var _ = bytes.Buffer{}
