// Package main provides the HTTP server for the Medication Advisor Engine.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cardiofit/medication-advisor-engine/advisor"
	"github.com/cardiofit/medication-advisor-engine/evidence"
	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

func main() {
	// Load configuration
	config := loadConfig()

	// Initialize stores (in-memory for standalone)
	snapshotStore := NewInMemorySnapshotStore()
	envelopeStore := NewInMemoryEnvelopeStore()

	// Create engine - will fail if KB services are not configured
	engine, err := advisor.NewMedicationAdvisorEngine(
		snapshotStore,
		envelopeStore,
		config,
	)
	if err != nil {
		log.Fatalf("❌ Failed to create Medication Advisor Engine: %v", err)
	}

	// Setup router
	router := setupRouter(engine)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", getEnv("PORT", "8101")),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("🚀 Medication Advisor Engine started on port %s", getEnv("PORT", "8101"))
	log.Printf("📋 Environment: %s", config.Environment)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func loadConfig() advisor.EngineConfig {
	return advisor.EngineConfig{
		Environment:        getEnv("ENVIRONMENT", "development"),
		SnapshotTTLMinutes: 30,
		KB1URL:             getEnv("KB1_DOSING_URL", "http://localhost:8081"),         // kb1-drug-rules
		KB2URL:             getEnv("KB2_INTERACTIONS_URL", "http://localhost:8095"),   // kb5-drug-interactions
		KB3URL:             getEnv("KB3_GUIDELINES_URL", "http://localhost:8083"),     // kb3-guidelines
		KB4URL:             getEnv("KB4_SAFETY_URL", "http://localhost:8088"),         // kb4-patient-safety
		KB5URL:             getEnv("KB5_MONITORING_URL", "http://localhost:8092"),     // kb7-terminology
		KB6URL:             getEnv("KB6_EFFICACY_URL", "http://localhost:8086"),       // kb6-formulary
		// V3 Architecture: KB-5 DDI service for drug-drug interaction checking
		KB5DDIURL:          getEnv("KB5_DDI_URL", "http://localhost:8095"),            // kb5-drug-interactions (DDI service)
		// V3 Architecture: KB-16 Lab Safety service for lab-drug contraindications (REQUIRED)
		KB16URL:            getEnv("KB16_LAB_SAFETY_URL", "http://localhost:8098"),    // kb-16-lab-interpretation
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupRouter(engine *advisor.MedicationAdvisorEngine) *gin.Engine {
	router := gin.Default()

	// Health endpoints
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, engine.Health())
	})

	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	// API v1 endpoints
	v1 := router.Group("/api/v1/advisor")
	{
		v1.POST("/calculate", handleCalculate(engine))
		v1.POST("/validate", handleValidate(engine))
		v1.POST("/commit", handleCommit(engine))
		v1.POST("/explain", handleExplain(engine))

		// V3 Risk Profile endpoint (KB-19 calls this for risk-only calculation)
		// Med-Advisor = Judge (calculates risks), KB-19 = Clerk (makes decisions)
		v1.POST("/risk-profile", handleRiskProfile(engine))
	}

	// V3 API routes (exposed at /api/v1/ for KB-19 integration)
	v1Root := router.Group("/api/v1")
	{
		// Alternative path for KB-19 client
		v1Root.POST("/risk-profile", handleRiskProfile(engine))
	}

	// CDS Hooks endpoints
	cds := router.Group("/cds-services")
	{
		cds.GET("", handleCDSDiscovery())
		cds.POST("/medication-advisor", handleCDSHook(engine))
	}

	return router
}

// API Handlers

func handleCalculate(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req advisor.CalculateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		resp, err := engine.Calculate(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleValidate(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req advisor.ValidateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		resp, err := engine.Validate(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleCommit(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req advisor.CommitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		resp, err := engine.Commit(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

func handleExplain(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req advisor.ExplainRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		resp, err := engine.Explain(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// handleRiskProfile is the V3 API endpoint for risk-only calculation.
// This is called by KB-19 Transaction Authority.
// Med-Advisor (8095) = Judge (calculates risks, no decisions)
// KB-19 (8119) = Clerk (makes block decisions, handles governance)
func handleRiskProfile(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req advisor.RiskProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		resp, err := engine.RiskProfile(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	}
}

// CDS Hooks Handlers

type CDSService struct {
	Hook        string `json:"hook"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ID          string `json:"id"`
}

type CDSDiscoveryResponse struct {
	Services []CDSService `json:"services"`
}

func handleCDSDiscovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		response := CDSDiscoveryResponse{
			Services: []CDSService{
				{
					Hook:        "order-select",
					Title:       "Medication Advisor",
					Description: "Provides medication recommendations based on clinical context",
					ID:          "medication-advisor",
				},
			},
		}
		c.JSON(http.StatusOK, response)
	}
}

type CDSRequest struct {
	Hook        string                 `json:"hook"`
	HookInstance string                `json:"hookInstance"`
	Context     map[string]interface{} `json:"context"`
	Prefetch    map[string]interface{} `json:"prefetch,omitempty"`
}

type CDSCard struct {
	Summary   string     `json:"summary"`
	Detail    string     `json:"detail,omitempty"`
	Indicator string     `json:"indicator"` // info, warning, critical
	Source    CDSSource  `json:"source"`
	Links     []CDSLink  `json:"links,omitempty"`
}

type CDSSource struct {
	Label string `json:"label"`
	URL   string `json:"url,omitempty"`
}

type CDSLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Type  string `json:"type"` // absolute, smart
}

type CDSResponse struct {
	Cards []CDSCard `json:"cards"`
}

func handleCDSHook(engine *advisor.MedicationAdvisorEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CDSRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Convert CDS request to Calculate request
		// This is simplified - real implementation would parse FHIR resources from prefetch
		cards := []CDSCard{
			{
				Summary:   "Medication Advisor recommendation available",
				Detail:    "Use /api/v1/advisor/calculate for detailed medication recommendations",
				Indicator: "info",
				Source: CDSSource{
					Label: "Medication Advisor Engine",
					URL:   "https://cardiofit.com/medication-advisor",
				},
			},
		}

		c.JSON(http.StatusOK, CDSResponse{Cards: cards})
	}
}

// In-memory stores for standalone operation

type InMemorySnapshotStore struct {
	snapshots map[string]*snapshot.ClinicalSnapshot
}

func NewInMemorySnapshotStore() *InMemorySnapshotStore {
	return &InMemorySnapshotStore{
		snapshots: make(map[string]*snapshot.ClinicalSnapshot),
	}
}

func (s *InMemorySnapshotStore) Save(ctx context.Context, snap *snapshot.ClinicalSnapshot) error {
	s.snapshots[snap.ID.String()] = snap
	return nil
}

func (s *InMemorySnapshotStore) Get(ctx context.Context, id string) (*snapshot.ClinicalSnapshot, error) {
	snap, ok := s.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("snapshot not found: %s", id)
	}
	return snap, nil
}

func (s *InMemorySnapshotStore) Delete(ctx context.Context, id string) error {
	delete(s.snapshots, id)
	return nil
}

func (s *InMemorySnapshotStore) List(ctx context.Context, filters snapshot.SnapshotFilters) ([]*snapshot.ClinicalSnapshot, error) {
	var result []*snapshot.ClinicalSnapshot
	for _, snap := range s.snapshots {
		if filters.PatientID != "" && snap.PatientID.String() != filters.PatientID {
			continue
		}
		result = append(result, snap)
	}
	return result, nil
}

func (s *InMemorySnapshotStore) UpdateStatus(ctx context.Context, id string, status snapshot.SnapshotStatus) error {
	snap, ok := s.snapshots[id]
	if !ok {
		return fmt.Errorf("snapshot not found: %s", id)
	}
	snap.Status = status
	return nil
}

type InMemoryEnvelopeStore struct {
	envelopes map[string]*evidence.EvidenceEnvelope
}

func NewInMemoryEnvelopeStore() *InMemoryEnvelopeStore {
	return &InMemoryEnvelopeStore{
		envelopes: make(map[string]*evidence.EvidenceEnvelope),
	}
}

func (s *InMemoryEnvelopeStore) Save(ctx context.Context, env *evidence.EvidenceEnvelope) error {
	s.envelopes[env.ID.String()] = env
	return nil
}

func (s *InMemoryEnvelopeStore) Get(ctx context.Context, id string) (*evidence.EvidenceEnvelope, error) {
	env, ok := s.envelopes[id]
	if !ok {
		return nil, fmt.Errorf("envelope not found: %s", id)
	}
	return env, nil
}

func (s *InMemoryEnvelopeStore) GetBySnapshot(ctx context.Context, snapshotID string) (*evidence.EvidenceEnvelope, error) {
	for _, env := range s.envelopes {
		if env.SnapshotID.String() == snapshotID {
			return env, nil
		}
	}
	return nil, fmt.Errorf("envelope not found for snapshot: %s", snapshotID)
}

func (s *InMemoryEnvelopeStore) List(ctx context.Context, patientID string, limit int) ([]*evidence.EvidenceEnvelope, error) {
	var result []*evidence.EvidenceEnvelope
	for _, env := range s.envelopes {
		if patientID != "" && env.PatientID.String() != patientID {
			continue
		}
		result = append(result, env)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}
