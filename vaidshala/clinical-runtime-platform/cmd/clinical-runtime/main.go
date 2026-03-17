// Package main provides the HTTP server for the Full Clinical Runtime Platform.
//
// This server exposes the COMPLETE ENGINE FLOW with 3-Phase API:
//   - POST /v1/calculate  - Runs ALL engines, returns recommendations (NO changes)
//   - POST /v1/validate   - Clinician reviews/approves (NO changes)
//   - POST /v1/commit     - Executes approved actions (MAKES changes)
//
// ENGINES ORCHESTRATED:
//   1. CQL Engine       - Clinical truth determination ("What is true?")
//   2. Measure Engine   - Care accountability ("Are we meeting standards?")
//   3. Medication Engine - Drug recommendations ("What medications are appropriate?")
//
// FDA SaMD COMPLIANCE:
//   - Human-in-the-loop: AI recommendations require clinician approval
//   - Audit trail: Every phase logged with timestamps and user IDs
//   - Immutable snapshots: Clinical context frozen at calculate time
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vaidshala/clinical-runtime-platform/builders"
	"vaidshala/clinical-runtime-platform/clients"
	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines"
	vmcuevents "vaidshala/clinical-runtime-platform/engines/vmcu/events"
	"vaidshala/clinical-runtime-platform/factory"
)

// ============================================================================
// MAIN ENTRY POINT
// ============================================================================

func main() {
	// Load configuration
	config := loadConfig()

	// Create orchestrator with all engines wired
	orchestrator := createOrchestrator(config)

	// Create session store for 3-phase workflow
	sessionStore := NewSessionStore()

	// Setup router
	router := setupRouter(orchestrator, sessionStore, config)

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("============================================================")
	log.Printf("  CLINICAL RUNTIME PLATFORM - FULL ORCHESTRATOR")
	log.Printf("============================================================")
	log.Printf("  Port:        %s", config.Port)
	log.Printf("  Environment: %s", config.Environment)
	log.Printf("  Region:      %s", config.Region)
	log.Printf("============================================================")
	log.Printf("  ENGINES LOADED:")
	log.Printf("    - CQL Engine (clinical truths)")
	log.Printf("    - Measure Engine (care gaps)")
	log.Printf("    - Medication Engine (drug recommendations)")
	log.Printf("============================================================")
	log.Printf("  3-PHASE API ENDPOINTS:")
	log.Printf("    POST /v1/calculate    - Run all engines")
	log.Printf("    POST /v1/validate     - Clinician approval")
	log.Printf("    POST /v1/commit       - Execute actions")
	log.Printf("    POST /v1/vmcu-events  - V-MCU event receiver (KB-19 → cache invalidation)")
	log.Printf("============================================================")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// ============================================================================
// CONFIGURATION
// ============================================================================

type Config struct {
	Port        string
	Environment string
	Region      string
	// KB Service URLs (for future KB client integration)
	KB2URL string
	KB6URL string
	KB7URL string
	KB8URL string
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8090"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Region:      getEnv("REGION", "AU"),
		KB2URL:      getEnv("KB2_URL", "http://localhost:8082"),
		KB6URL:      getEnv("KB6_URL", "http://localhost:8087"),
		KB7URL:      getEnv("KB7_URL", "http://localhost:8092"),
		KB8URL:      getEnv("KB8_URL", "http://localhost:8097"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============================================================================
// ORCHESTRATOR CREATION (Uses Factory Pattern for Full KB Wiring)
// ============================================================================

func createOrchestrator(config Config) *RuntimeOrchestrator {
	// ========================================================================
	// STEP 1: Use Factory to wire ALL KB services (KB-1, KB-4, KB-5, KB-6, KB-7, KB-8)
	// This is the CTO/CMO-approved wiring pattern that ensures full KB coverage.
	// ========================================================================
	wired, err := factory.WireOrchestratorFromEnv()
	if err != nil {
		log.Fatalf("Failed to wire orchestrator from factory: %v", err)
	}

	// Get the KB config that was used (for logging)
	kbConfig := clients.KBClientConfigFromEnv()

	log.Printf("  KB CLIENTS CONNECTED (via Factory):")
	log.Printf("    - KB-1 Drug Rules:       %s", kbConfig.KB1BaseURL)
	log.Printf("    - KB-4 Patient Safety:   %s", kbConfig.KB4BaseURL)
	log.Printf("    - KB-5 Drug Interactions:%s", kbConfig.KB5BaseURL)
	log.Printf("    - KB-6 Formulary:        %s", kbConfig.KB6BaseURL)
	log.Printf("    - KB-7 Terminology:      %s", kbConfig.KB7BaseURL)
	log.Printf("    - KB-8 Calculator:       %s", kbConfig.KB8BaseURL)

	// ========================================================================
	// STEP 2: Create CQL and Measure Engines
	// (Factory currently sets these to nil, so we create them here)
	// ========================================================================
	cqlEngine := engines.NewCQLEngine(engines.DefaultCQLEngineConfig())
	measureEngine := engines.NewMeasureEngine(engines.DefaultMeasureEngineConfig())

	// Use MedicationEngine from factory (it's already wired)
	medicationEngine := wired.MedicationEngine
	if medicationEngine == nil {
		// Fallback if factory didn't create it
		medicationEngine = engines.NewMedicationEngine(engines.DefaultMedicationEngineConfig())
	}

	return &RuntimeOrchestrator{
		cqlEngine:        cqlEngine,
		measureEngine:    measureEngine,
		medicationEngine: medicationEngine,
		snapshotBuilder:  wired.SnapshotBuilder, // Use factory-wired snapshot builder!
		kbClients:        wired.KBClients,       // Use factory-wired KB clients!
		config:           config,
	}
}

// RuntimeOrchestrator manages the complete engine flow.
type RuntimeOrchestrator struct {
	cqlEngine        *engines.CQLEngine
	measureEngine    *engines.MeasureEngine
	medicationEngine *engines.MedicationEngine
	snapshotBuilder  *builders.KnowledgeSnapshotBuilder
	kbClients        *clients.KBClients
	config           Config
}

// ============================================================================
// HTTP ROUTER SETUP
// ============================================================================

func setupRouter(orchestrator *RuntimeOrchestrator, sessionStore *SessionStore, config Config) *gin.Engine {
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Health endpoints
	router.GET("/health", handleHealth(orchestrator))
	router.GET("/ready", handleReady())

	// 3-Phase API endpoints
	v1 := router.Group("/v1")
	{
		v1.POST("/calculate", handleCalculate(orchestrator, sessionStore))
		v1.POST("/validate", handleValidate(sessionStore))
		v1.POST("/commit", handleCommit(sessionStore))
	}

	// V-MCU event receiver: KB-19 forwards MCU_GATE_CHANGED here for cache invalidation.
	// CacheInvalidator logs the invalidation; actual SafetyCache wiring happens when
	// V-MCU is instantiated by the orchestrator (future: wire safetyCache.Invalidate).
	invalidator := vmcuevents.NewCacheInvalidator(func(patientID string) {
		log.Printf("V-MCU cache invalidated for patient %s", patientID)
	})
	eventReceiver := vmcuevents.NewHTTPEventReceiver(invalidator)
	eventReceiver.RegisterRoutes(v1)

	// Session management
	router.GET("/v1/session/:id", handleGetSession(sessionStore))
	router.DELETE("/v1/session/:id", handleDeleteSession(sessionStore))

	return router
}

// ============================================================================
// HEALTH ENDPOINTS
// ============================================================================

func handleHealth(orchestrator *RuntimeOrchestrator) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "clinical-runtime-platform",
			"engines": gin.H{
				"cql_engine":        orchestrator.cqlEngine.Name(),
				"measure_engine":    orchestrator.measureEngine.Name(),
				"medication_engine": orchestrator.medicationEngine.Name(),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func handleReady() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	}
}

// ============================================================================
// PHASE 1: CALCULATE - Run All Engines (NO CHANGES)
// ============================================================================

// CalculateRequest is the input for Phase 1.
type CalculateRequest struct {
	// Patient identification
	PatientID string `json:"patient_id" binding:"required"`

	// Raw FHIR data (conditions, medications, observations, etc.)
	PatientData PatientData `json:"patient_data" binding:"required"`

	// Who is making this request
	RequestedBy string `json:"requested_by" binding:"required"`

	// Which engines to run (empty = all)
	RequestedEngines []string `json:"requested_engines,omitempty"`
}

// PatientData contains the clinical context for evaluation.
type PatientData struct {
	Demographics    Demographics    `json:"demographics"`
	Conditions      []Condition     `json:"conditions,omitempty"`
	Medications     []Medication    `json:"medications,omitempty"`
	LabResults      []LabResult     `json:"lab_results,omitempty"`
	VitalSigns      []VitalSign     `json:"vital_signs,omitempty"`
	Encounters      []Encounter     `json:"encounters,omitempty"`
}

type Demographics struct {
	BirthDate string `json:"birth_date"`
	Gender    string `json:"gender"`
	Region    string `json:"region,omitempty"`
}

type Condition struct {
	Code           string `json:"code"`
	System         string `json:"system"`
	Display        string `json:"display"`
	ClinicalStatus string `json:"clinical_status"`
}

type Medication struct {
	Code        string  `json:"code"`
	System      string  `json:"system"`
	Display     string  `json:"display"`
	Status      string  `json:"status"`
	DoseValue   float64 `json:"dose_value,omitempty"`
	DoseUnit    string  `json:"dose_unit,omitempty"`
	RouteCode   string  `json:"route_code,omitempty"`
	RouteSystem string  `json:"route_system,omitempty"`
}

type LabResult struct {
	Code      string    `json:"code"`
	System    string    `json:"system"`
	Display   string    `json:"display"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

type VitalSign struct {
	SystolicBP  float64   `json:"systolic_bp,omitempty"`
	DiastolicBP float64   `json:"diastolic_bp,omitempty"`
	HeartRate   float64   `json:"heart_rate,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	SpO2        float64   `json:"spo2,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

type Encounter struct {
	EncounterID string `json:"encounter_id"`
	Class       string `json:"class"`
	Status      string `json:"status"`
}

// CalculateResponse is the output from Phase 1.
// Designed for FDA SaMD compliance with full audit trail.
type CalculateResponse struct {
	// Request identification (echo back for correlation)
	PatientID string `json:"patient_id"`
	RequestID string `json:"request_id"`
	Phase     string `json:"phase"` // Always "calculate" for this endpoint

	// Session ID for validate/commit phases
	SessionID string `json:"session_id"`

	// Success status
	Success bool `json:"success"`

	// Engine results with timing
	EngineResults []EngineResultSummary `json:"engine_results"`

	// Combined outputs
	ClinicalFacts   []contracts.ClinicalFact   `json:"clinical_facts"`
	MeasureResults  []contracts.MeasureResult  `json:"measure_results"`
	Recommendations []contracts.Recommendation `json:"recommendations"`
	Alerts          []contracts.Alert          `json:"alerts"`

	// Care gaps - convenience array (subset of MeasureResults where careGapIdentified=true)
	CareGaps []CareGapSummary `json:"care_gaps,omitempty"`

	// Knowledge snapshot used for this execution (FDA audit requirement)
	KnowledgeSnapshot *KnowledgeSnapshotSummary `json:"knowledge_snapshot"`

	// Execution metadata
	ExecutionTimeMs int64    `json:"execution_time_ms"`
	Warnings        []string `json:"warnings,omitempty"`

	// Next step instructions
	NextStep string `json:"next_step"`
}

// CareGapSummary is a simplified care gap for the response (avoids duplication with MeasureResults)
type CareGapSummary struct {
	MeasureID   string `json:"measure_id"`
	MeasureName string `json:"measure_name"`
	Rationale   string `json:"rationale"`
	Priority    string `json:"priority"` // high, medium, low based on measure
}

// KnowledgeSnapshotSummary contains the KB state used for execution (for audit)
type KnowledgeSnapshotSummary struct {
	SnapshotTimestamp     time.Time         `json:"snapshot_timestamp"`
	KBVersions            map[string]string `json:"kb_versions"`
	ValueSetMemberships   map[string]bool   `json:"valueset_memberships"`
	TerminologySourceURL  string            `json:"terminology_source_url,omitempty"`
}

type EngineResultSummary struct {
	EngineName       string `json:"engine_name"`
	Success          bool   `json:"success"`
	FactsProduced    int    `json:"facts_produced"`
	AlertsProduced   int    `json:"alerts_produced"`
	RecsProduced     int    `json:"recommendations_produced"`
	ExecutionTimeMs  int64  `json:"execution_time_ms"`  // Milliseconds (may be 0 for sub-ms)
	ExecutionTimeUs  int64  `json:"execution_time_us"`  // Microseconds (for sub-ms granularity)
	Error            string `json:"error,omitempty"`
}

func handleCalculate(orchestrator *RuntimeOrchestrator, store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CalculateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx := c.Request.Context()
		startTime := time.Now()
		warnings := make([]string, 0)

		// Build ClinicalExecutionContext from request
		// CRITICAL: This calls KB-7 via snapshotBuilder to populate ValueSetMemberships!
		execCtx, err := orchestrator.buildExecutionContext(ctx, &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to build execution context: %v", err)})
			return
		}

		// ================================================================
		// EXECUTE ENGINE FLOW: CQL → Measure → Medication
		// ================================================================

		engineResults := make([]EngineResultSummary, 0)
		allFacts := make([]contracts.ClinicalFact, 0)
		allMeasures := make([]contracts.MeasureResult, 0)
		allRecs := make([]contracts.Recommendation, 0)
		allAlerts := make([]contracts.Alert, 0)

		// 1. CQL Engine (produces clinical truths)
		cqlStart := time.Now()
		cqlResult, err := orchestrator.cqlEngine.Evaluate(ctx, execCtx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("CQL Engine error: %v", err))
			cqlResult = &contracts.EngineResult{EngineName: "cql-engine", Success: false, Error: err.Error()}
		}
		cqlDuration := time.Since(cqlStart)
		engineResults = append(engineResults, EngineResultSummary{
			EngineName:      cqlResult.EngineName,
			Success:         cqlResult.Success,
			FactsProduced:   len(cqlResult.ClinicalFacts),
			AlertsProduced:  len(cqlResult.Alerts),
			RecsProduced:    len(cqlResult.Recommendations),
			ExecutionTimeMs: cqlDuration.Milliseconds(),
			ExecutionTimeUs: cqlDuration.Microseconds(),
			Error:           cqlResult.Error,
		})
		allFacts = append(allFacts, cqlResult.ClinicalFacts...)
		allAlerts = append(allAlerts, cqlResult.Alerts...)

		// 2. Measure Engine (consumes CQL facts, produces care gaps)
		measureStart := time.Now()
		var measureResult *contracts.EngineResult
		if len(cqlResult.ClinicalFacts) > 0 {
			measureResult, err = orchestrator.measureEngine.EvaluateWithFacts(ctx, execCtx, cqlResult.ClinicalFacts)
		} else {
			measureResult, err = orchestrator.measureEngine.Evaluate(ctx, execCtx)
			warnings = append(warnings, "Measure Engine ran without CQL facts")
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Measure Engine error: %v", err))
			measureResult = &contracts.EngineResult{EngineName: "measure-engine", Success: false, Error: err.Error()}
		}
		measureDuration := time.Since(measureStart)
		engineResults = append(engineResults, EngineResultSummary{
			EngineName:      measureResult.EngineName,
			Success:         measureResult.Success,
			FactsProduced:   len(measureResult.ClinicalFacts),
			AlertsProduced:  len(measureResult.Alerts),
			RecsProduced:    len(measureResult.Recommendations),
			ExecutionTimeMs: measureDuration.Milliseconds(),
			ExecutionTimeUs: measureDuration.Microseconds(),
			Error:           measureResult.Error,
		})
		allMeasures = append(allMeasures, measureResult.MeasureResults...)
		allRecs = append(allRecs, measureResult.Recommendations...)
		allAlerts = append(allAlerts, measureResult.Alerts...)

		// 3. Medication Engine (independent evaluation)
		medStart := time.Now()
		medResult, err := orchestrator.medicationEngine.Evaluate(ctx, execCtx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Medication Engine error: %v", err))
			medResult = &contracts.EngineResult{EngineName: "medication-engine", Success: false, Error: err.Error()}
		}
		medDuration := time.Since(medStart)
		engineResults = append(engineResults, EngineResultSummary{
			EngineName:      medResult.EngineName,
			Success:         medResult.Success,
			FactsProduced:   len(medResult.ClinicalFacts),
			AlertsProduced:  len(medResult.Alerts),
			RecsProduced:    len(medResult.Recommendations),
			ExecutionTimeMs: medDuration.Milliseconds(),
			ExecutionTimeUs: medDuration.Microseconds(),
			Error:           medResult.Error,
		})
		allRecs = append(allRecs, medResult.Recommendations...)
		allAlerts = append(allAlerts, medResult.Alerts...)

		// Extract care gaps (full for session, summary for response)
		sessionCareGaps := make([]contracts.MeasureResult, 0)
		careGapSummaries := make([]CareGapSummary, 0)
		for _, mr := range allMeasures {
			if mr.CareGapIdentified {
				sessionCareGaps = append(sessionCareGaps, mr)
				// Create summary (avoids duplication, provides priority)
				priority := "medium" // Default
				if mr.MeasureID == "CMS165" || mr.MeasureID == "CMS122" {
					priority = "high" // BP and diabetes control are high priority
				}
				careGapSummaries = append(careGapSummaries, CareGapSummary{
					MeasureID:   mr.MeasureID,
					MeasureName: mr.MeasureName,
					Rationale:   mr.Rationale,
					Priority:    priority,
				})
			}
		}

		// ================================================================
		// CREATE SESSION FOR VALIDATE/COMMIT PHASES
		// ================================================================

		requestID := execCtx.Runtime.RequestID // Use the request ID from context
		session := &Session{
			ID:              uuid.New().String(),
			PatientID:       req.PatientID,
			RequestedBy:     req.RequestedBy,
			Status:          SessionStatusPendingValidation,
			CreatedAt:       time.Now(),
			ExpiresAt:       time.Now().Add(30 * time.Minute),
			ExecutionContext: execCtx,
			ClinicalFacts:   allFacts,
			MeasureResults:  allMeasures,
			Recommendations: allRecs,
			Alerts:          allAlerts,
			CareGaps:        sessionCareGaps,
		}
		store.Save(session)

		// Build knowledge snapshot summary for audit trail
		knowledgeSnapshot := &KnowledgeSnapshotSummary{
			SnapshotTimestamp:    execCtx.Knowledge.SnapshotTimestamp,
			KBVersions:           execCtx.Knowledge.KBVersions,
			ValueSetMemberships:  execCtx.Knowledge.Terminology.ValueSetMemberships,
			TerminologySourceURL: orchestrator.config.KB7URL,
		}

		// Build response with full audit trail
		response := CalculateResponse{
			PatientID:         req.PatientID,
			RequestID:         requestID,
			Phase:             "calculate",
			SessionID:         session.ID,
			Success:           true,
			EngineResults:     engineResults,
			ClinicalFacts:     allFacts,
			MeasureResults:    allMeasures,
			Recommendations:   allRecs,
			Alerts:            allAlerts,
			CareGaps:          careGapSummaries,
			KnowledgeSnapshot: knowledgeSnapshot,
			ExecutionTimeMs:   time.Since(startTime).Milliseconds(),
			Warnings:          warnings,
			NextStep:          fmt.Sprintf("POST /v1/validate with session_id: %s", session.ID),
		}

		c.JSON(http.StatusOK, response)
	}
}

// ============================================================================
// PHASE 2: VALIDATE - Clinician Review (NO CHANGES)
// ============================================================================

// ValidateRequest is the input for Phase 2.
type ValidateRequest struct {
	SessionID string `json:"session_id" binding:"required"`

	// Clinician who is validating
	ValidatedBy string `json:"validated_by" binding:"required"`

	// Decisions on each recommendation
	Decisions []ValidationDecision `json:"decisions" binding:"required"`

	// Optional clinical notes
	ClinicalNotes string `json:"clinical_notes,omitempty"`
}

type ValidationDecision struct {
	RecommendationID string `json:"recommendation_id"`
	Approved         bool   `json:"approved"`
	Reason           string `json:"reason,omitempty"`
}

// ValidateResponse is the output from Phase 2.
type ValidateResponse struct {
	SessionID      string    `json:"session_id"`
	Status         string    `json:"status"`
	ValidatedBy    string    `json:"validated_by"`
	ValidatedAt    time.Time `json:"validated_at"`
	ApprovedCount  int       `json:"approved_count"`
	RejectedCount  int       `json:"rejected_count"`
	ApprovedActions []string `json:"approved_actions"`
	NextStep       string    `json:"next_step"`
}

func handleValidate(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ValidateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get session
		session, err := store.Get(req.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or expired"})
			return
		}

		// Check session status
		if session.Status != SessionStatusPendingValidation {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Session not in pending validation state",
				"status": session.Status,
			})
			return
		}

		// Process decisions
		approvedCount := 0
		rejectedCount := 0
		approvedActions := make([]string, 0)
		approvedRecs := make([]contracts.Recommendation, 0)

		decisionMap := make(map[string]ValidationDecision)
		for _, d := range req.Decisions {
			decisionMap[d.RecommendationID] = d
		}

		for _, rec := range session.Recommendations {
			if decision, ok := decisionMap[rec.ID]; ok {
				if decision.Approved {
					approvedCount++
					approvedActions = append(approvedActions, rec.Title)
					approvedRecs = append(approvedRecs, rec)
				} else {
					rejectedCount++
				}
			}
		}

		// Update session
		session.Status = SessionStatusValidated
		session.ValidatedBy = req.ValidatedBy
		session.ValidatedAt = time.Now()
		session.ValidationDecisions = req.Decisions
		session.ApprovedRecommendations = approvedRecs
		session.ClinicalNotes = req.ClinicalNotes
		store.Save(session)

		response := ValidateResponse{
			SessionID:       session.ID,
			Status:          string(session.Status),
			ValidatedBy:     req.ValidatedBy,
			ValidatedAt:     session.ValidatedAt,
			ApprovedCount:   approvedCount,
			RejectedCount:   rejectedCount,
			ApprovedActions: approvedActions,
			NextStep:        fmt.Sprintf("POST /v1/commit with session_id: %s", session.ID),
		}

		c.JSON(http.StatusOK, response)
	}
}

// ============================================================================
// PHASE 3: COMMIT - Execute Approved Actions (MAKES CHANGES)
// ============================================================================

// CommitRequest is the input for Phase 3.
type CommitRequest struct {
	SessionID string `json:"session_id" binding:"required"`

	// Who is committing (must match validator or be supervisor)
	CommittedBy string `json:"committed_by" binding:"required"`

	// Final confirmation
	Confirmed bool `json:"confirmed" binding:"required"`
}

// CommitResponse is the output from Phase 3.
type CommitResponse struct {
	SessionID     string             `json:"session_id"`
	Status        string             `json:"status"`
	CommittedBy   string             `json:"committed_by"`
	CommittedAt   time.Time          `json:"committed_at"`
	ActionsCommitted int             `json:"actions_committed"`
	FHIRResources []FHIRResourceRef  `json:"fhir_resources,omitempty"`
	AuditTrail    AuditTrail         `json:"audit_trail"`
}

type FHIRResourceRef struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Action       string `json:"action"` // created, updated
}

type AuditTrail struct {
	SessionID     string    `json:"session_id"`
	PatientID     string    `json:"patient_id"`
	RequestedBy   string    `json:"requested_by"`
	RequestedAt   time.Time `json:"requested_at"`
	ValidatedBy   string    `json:"validated_by"`
	ValidatedAt   time.Time `json:"validated_at"`
	CommittedBy   string    `json:"committed_by"`
	CommittedAt   time.Time `json:"committed_at"`
	EnginesRun    []string  `json:"engines_run"`
	Recommendations int     `json:"recommendations_total"`
	Approved      int       `json:"approved"`
	Rejected      int       `json:"rejected"`
	Committed     int       `json:"committed"`
}

func handleCommit(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CommitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if !req.Confirmed {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Commit must be confirmed"})
			return
		}

		// Get session
		session, err := store.Get(req.SessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found or expired"})
			return
		}

		// Check session status
		if session.Status != SessionStatusValidated {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Session must be validated before commit",
				"status": session.Status,
			})
			return
		}

		// Execute approved actions (create FHIR resources)
		fhirResources := make([]FHIRResourceRef, 0)
		for _, rec := range session.ApprovedRecommendations {
			// In production, this would create actual FHIR resources
			// For now, we generate references
			resource := FHIRResourceRef{
				ResourceType: mapRecommendationToFHIR(rec.Type),
				ResourceID:   uuid.New().String(),
				Action:       "created",
			}
			fhirResources = append(fhirResources, resource)
		}

		// Update session
		session.Status = SessionStatusCommitted
		session.CommittedBy = req.CommittedBy
		session.CommittedAt = time.Now()
		session.FHIRResources = fhirResources
		store.Save(session)

		// Build audit trail
		auditTrail := AuditTrail{
			SessionID:       session.ID,
			PatientID:       session.PatientID,
			RequestedBy:     session.RequestedBy,
			RequestedAt:     session.CreatedAt,
			ValidatedBy:     session.ValidatedBy,
			ValidatedAt:     session.ValidatedAt,
			CommittedBy:     req.CommittedBy,
			CommittedAt:     session.CommittedAt,
			EnginesRun:      []string{"cql-engine", "measure-engine", "medication-engine"},
			Recommendations: len(session.Recommendations),
			Approved:        len(session.ApprovedRecommendations),
			Rejected:        len(session.Recommendations) - len(session.ApprovedRecommendations),
			Committed:       len(fhirResources),
		}

		response := CommitResponse{
			SessionID:        session.ID,
			Status:           string(session.Status),
			CommittedBy:      req.CommittedBy,
			CommittedAt:      session.CommittedAt,
			ActionsCommitted: len(fhirResources),
			FHIRResources:    fhirResources,
			AuditTrail:       auditTrail,
		}

		c.JSON(http.StatusOK, response)
	}
}

func mapRecommendationToFHIR(recType string) string {
	switch recType {
	case "medication":
		return "MedicationRequest"
	case "care-gap":
		return "Task"
	case "observation":
		return "Observation"
	case "procedure":
		return "ServiceRequest"
	default:
		return "Task"
	}
}

// ============================================================================
// SESSION MANAGEMENT
// ============================================================================

func handleGetSession(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		session, err := store.Get(sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"session_id":      session.ID,
			"patient_id":      session.PatientID,
			"status":          session.Status,
			"requested_by":    session.RequestedBy,
			"created_at":      session.CreatedAt,
			"expires_at":      session.ExpiresAt,
			"validated_by":    session.ValidatedBy,
			"validated_at":    session.ValidatedAt,
			"committed_by":    session.CommittedBy,
			"committed_at":    session.CommittedAt,
			"recommendations": len(session.Recommendations),
			"alerts":          len(session.Alerts),
			"care_gaps":       len(session.CareGaps),
		})
	}
}

func handleDeleteSession(store *SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("id")
		store.Delete(sessionID)
		c.JSON(http.StatusOK, gin.H{"deleted": sessionID})
	}
}

// ============================================================================
// SESSION STORE
// ============================================================================

type SessionStatus string

const (
	SessionStatusPendingValidation SessionStatus = "pending_validation"
	SessionStatusValidated         SessionStatus = "validated"
	SessionStatusCommitted         SessionStatus = "committed"
	SessionStatusExpired           SessionStatus = "expired"
)

type Session struct {
	ID               string
	PatientID        string
	RequestedBy      string
	Status           SessionStatus
	CreatedAt        time.Time
	ExpiresAt        time.Time
	ValidatedBy      string
	ValidatedAt      time.Time
	CommittedBy      string
	CommittedAt      time.Time
	ClinicalNotes    string

	// Frozen data
	ExecutionContext *contracts.ClinicalExecutionContext
	ClinicalFacts    []contracts.ClinicalFact
	MeasureResults   []contracts.MeasureResult
	Recommendations  []contracts.Recommendation
	Alerts           []contracts.Alert
	CareGaps         []contracts.MeasureResult

	// Validation phase
	ValidationDecisions     []ValidationDecision
	ApprovedRecommendations []contracts.Recommendation

	// Commit phase
	FHIRResources []FHIRResourceRef
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewSessionStore() *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*Session),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

func (s *SessionStore) Save(session *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.ID] = session
}

func (s *SessionStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	if time.Now().After(session.ExpiresAt) && session.Status == SessionStatusPendingValidation {
		session.Status = SessionStatusExpired
		return nil, fmt.Errorf("session expired: %s", id)
	}

	return session, nil
}

func (s *SessionStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *SessionStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			// Remove sessions older than 1 hour
			if now.Sub(session.CreatedAt) > time.Hour {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

// ============================================================================
// CONTEXT BUILDER (Uses KnowledgeSnapshotBuilder to call KB services!)
// ============================================================================

// buildExecutionContext builds ClinicalExecutionContext from request data.
// CRITICAL: Uses KnowledgeSnapshotBuilder to call KB-7 for ValueSetMemberships!
func (o *RuntimeOrchestrator) buildExecutionContext(ctx context.Context, req *CalculateRequest) (*contracts.ClinicalExecutionContext, error) {
	// Parse birth date
	var birthDate *time.Time
	if req.PatientData.Demographics.BirthDate != "" {
		if parsed, err := time.Parse("2006-01-02", req.PatientData.Demographics.BirthDate); err == nil {
			birthDate = &parsed
		}
	}

	// Build conditions with source references for audit trail
	conditions := make([]contracts.ClinicalCondition, len(req.PatientData.Conditions))
	for i, c := range req.PatientData.Conditions {
		conditions[i] = contracts.ClinicalCondition{
			Code: contracts.ClinicalCode{
				System:  c.System,
				Code:    c.Code,
				Display: c.Display,
			},
			ClinicalStatus:  c.ClinicalStatus,
			SourceReference: fmt.Sprintf("Condition/%s-%d", c.Code, i), // FHIR-style reference
		}
	}

	// Build medications with source references
	medications := make([]contracts.Medication, len(req.PatientData.Medications))
	for i, m := range req.PatientData.Medications {
		var dosage *contracts.Dosage
		if m.DoseValue > 0 {
			dosage = &contracts.Dosage{
				DoseQuantity: &contracts.Quantity{
					Value: m.DoseValue,
					Unit:  m.DoseUnit,
				},
			}
			if m.RouteCode != "" {
				dosage.Route = m.RouteCode
			}
		}
		medications[i] = contracts.Medication{
			Code: contracts.ClinicalCode{
				System:  m.System,
				Code:    m.Code,
				Display: m.Display,
			},
			Status:          m.Status,
			Dosage:          dosage,
			SourceReference: fmt.Sprintf("MedicationRequest/%s-%d", m.Code, i), // FHIR-style reference
		}
	}

	// Build lab results with source references
	labResults := make([]contracts.LabResult, len(req.PatientData.LabResults))
	for i, l := range req.PatientData.LabResults {
		ts := l.Timestamp
		labResults[i] = contracts.LabResult{
			Code: contracts.ClinicalCode{
				System:  l.System,
				Code:    l.Code,
				Display: l.Display,
			},
			Value: &contracts.Quantity{
				Value: l.Value,
				Unit:  l.Unit,
			},
			EffectiveDateTime: &ts,
			SourceReference:   fmt.Sprintf("Observation/%s-%d", l.Code, i), // FHIR-style reference
		}
	}

	// Build vital signs with source references
	vitalSigns := make([]contracts.VitalSign, 0)
	for i, v := range req.PatientData.VitalSigns {
		components := make([]contracts.ComponentValue, 0)
		if v.SystolicBP > 0 {
			components = append(components, contracts.ComponentValue{
				Code:  contracts.ClinicalCode{Code: "8480-6"}, // LOINC systolic
				Value: &contracts.Quantity{Value: v.SystolicBP},
			})
		}
		if v.DiastolicBP > 0 {
			components = append(components, contracts.ComponentValue{
				Code:  contracts.ClinicalCode{Code: "8462-4"}, // LOINC diastolic
				Value: &contracts.Quantity{Value: v.DiastolicBP},
			})
		}
		if len(components) > 0 {
			vitalSigns = append(vitalSigns, contracts.VitalSign{
				ComponentValues: components,
				SourceReference: fmt.Sprintf("Observation/vital-signs-%d", i), // FHIR-style reference
			})
		}
	}

	// Build encounters with source references
	encounters := make([]contracts.Encounter, len(req.PatientData.Encounters))
	for i, e := range req.PatientData.Encounters {
		encounters[i] = contracts.Encounter{
			EncounterID:     e.EncounterID,
			Class:           e.Class,
			Status:          e.Status,
			SourceReference: fmt.Sprintf("Encounter/%s", e.EncounterID), // FHIR-style reference
		}
	}

	// Determine region
	region := req.PatientData.Demographics.Region
	if region == "" {
		region = o.config.Region
	}

	// Build PatientContext first
	patientCtx := &contracts.PatientContext{
		Demographics: contracts.PatientDemographics{
			PatientID: req.PatientID,
			BirthDate: birthDate,
			Gender:    req.PatientData.Demographics.Gender,
			Region:    region,
		},
		ActiveConditions:  conditions,
		ActiveMedications: medications,
		RecentLabResults:  labResults,
		RecentVitalSigns:  vitalSigns,
		RecentEncounters:  encounters,
	}

	// ========================================================================
	// CRITICAL: Use KnowledgeSnapshotBuilder to call KB-7 for ValueSetMemberships!
	// This populates has_hypertension, is_diabetic, has_afib, etc.
	// ========================================================================
	var knowledge *contracts.KnowledgeSnapshot
	var err error

	if o.snapshotBuilder != nil {
		log.Printf("Building KnowledgeSnapshot from KB services...")
		knowledge, err = o.snapshotBuilder.Build(ctx, patientCtx)
		if err != nil {
			log.Printf("WARNING: KnowledgeSnapshot build failed: %v (using empty snapshot)", err)
			// Fall back to empty snapshot but continue
			knowledge = &contracts.KnowledgeSnapshot{
				SnapshotTimestamp: time.Now(),
				KBVersions:        make(map[string]string),
				Terminology: contracts.TerminologySnapshot{
					ValueSetMemberships: make(map[string]bool),
				},
			}
		} else {
			log.Printf("KnowledgeSnapshot built successfully!")
			log.Printf("  ValueSetMemberships: %v", knowledge.Terminology.ValueSetMemberships)
		}
	} else {
		log.Printf("WARNING: No snapshotBuilder available - using empty KnowledgeSnapshot")
		knowledge = &contracts.KnowledgeSnapshot{
			SnapshotTimestamp: time.Now(),
			KBVersions:        make(map[string]string),
			Terminology: contracts.TerminologySnapshot{
				ValueSetMemberships: make(map[string]bool),
			},
		}
	}

	return &contracts.ClinicalExecutionContext{
		Patient:   *patientCtx,
		Knowledge: *knowledge,
		Runtime: contracts.ExecutionMetadata{
			RequestID:        uuid.New().String(),
			RequestedBy:      req.RequestedBy,
			RequestedAt:      time.Now(),
			Region:           region,
			RequestedEngines: req.RequestedEngines,
			ExecutionMode:    "sync",
		},
	}, nil
}
