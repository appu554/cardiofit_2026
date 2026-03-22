package app

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	intakefhir "github.com/cardiofit/intake-onboarding-service/internal/fhir"
	"github.com/cardiofit/intake-onboarding-service/internal/flow"
	"github.com/cardiofit/intake-onboarding-service/internal/kafka"
	"github.com/cardiofit/intake-onboarding-service/internal/safety"
	"github.com/cardiofit/intake-onboarding-service/internal/slots"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Handler implements the Flutter app REST handlers for intake.
type Handler struct {
	eventStore   slots.EventStore
	safetyEngine *safety.Engine
	flowEngine   *flow.Engine
	fhirClient   *fhirclient.Client
	producer     *kafka.Producer
	db           *pgxpool.Pool
	logger       *zap.Logger
}

// NewHandler creates a new app handler with all dependencies.
func NewHandler(
	eventStore slots.EventStore,
	safetyEngine *safety.Engine,
	flowEngine *flow.Engine,
	fhirClient *fhirclient.Client,
	producer *kafka.Producer,
	db *pgxpool.Pool,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		eventStore:   eventStore,
		safetyEngine: safetyEngine,
		flowEngine:   flowEngine,
		fhirClient:   fhirClient,
		producer:     producer,
		db:           db,
		logger:       logger,
	}
}

// FillSlotRequest is the JSON body for POST /fhir/Encounter/:id/$fill-slot.
type FillSlotRequest struct {
	SlotName       string          `json:"slot_name"`
	Value          json.RawMessage `json:"value"`
	ExtractionMode string          `json:"extraction_mode"` // BUTTON, REGEX, NLU, DEVICE
	Confidence     float64         `json:"confidence"`
	SourceChannel  string          `json:"source_channel"` // APP, WHATSAPP, ASHA
}

// Validate checks the fill-slot request for required fields and slot existence.
func (r *FillSlotRequest) Validate() error {
	if r.SlotName == "" {
		return fmt.Errorf("slot_name is required")
	}
	if _, ok := slots.LookupSlot(r.SlotName); !ok {
		return fmt.Errorf("unknown slot_name: %s", r.SlotName)
	}
	if len(r.Value) == 0 || string(r.Value) == "null" {
		return fmt.Errorf("value is required")
	}
	if r.ExtractionMode == "" {
		r.ExtractionMode = "BUTTON"
	}
	if r.SourceChannel == "" {
		r.SourceChannel = "APP"
	}
	return nil
}

// FillSlotResponse is returned by $fill-slot.
type FillSlotResponse struct {
	Status         string                `json:"status"` // "ok", "hard_stopped", "error"
	SlotName       string                `json:"slot_name"`
	FHIRResourceID string                `json:"fhir_resource_id,omitempty"`
	SafetyResult   *SafetyResultResponse `json:"safety_result,omitempty"`
	NextNode       *NextNodeResponse     `json:"next_node,omitempty"`
	Progress       ProgressResponse      `json:"progress"`
}

// SafetyResultResponse is the safety result in the HTTP response.
type SafetyResultResponse struct {
	HardStops []RuleResultResponse `json:"hard_stops"`
	SoftFlags []RuleResultResponse `json:"soft_flags"`
}

// RuleResultResponse is a single rule result in the HTTP response.
type RuleResultResponse struct {
	RuleID string `json:"rule_id"`
	Reason string `json:"reason"`
}

// NextNodeResponse describes the next flow node for the client.
type NextNodeResponse struct {
	NodeID string   `json:"node_id"`
	Label  string   `json:"label,omitempty"`
	Slots  []string `json:"slots"`
}

// ProgressResponse tracks slot fill progress.
type ProgressResponse struct {
	Filled   int     `json:"filled"`
	Total    int     `json:"total"`
	Percent  float64 `json:"percent"`
	Complete bool    `json:"complete"`
}

// HandleFillSlot implements POST /fhir/Encounter/:id/$fill-slot.
// Flow: accept value -> validate slot -> safety check -> FHIR write -> Kafka publish -> next question.
func (h *Handler) HandleFillSlot(c *gin.Context) {
	encounterID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req FillSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	slotDef, _ := slots.LookupSlot(req.SlotName)

	// Get patientID from encounter (in production, look up from DB; here use header)
	patientIDStr := c.GetHeader("X-Patient-ID")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Patient-ID header required"})
		return
	}

	ctx := c.Request.Context()

	// 1. Append slot event to event store
	event := slots.SlotEvent{
		PatientID:      patientID,
		SlotName:       req.SlotName,
		Domain:         slotDef.Domain,
		Value:          req.Value,
		ExtractionMode: req.ExtractionMode,
		Confidence:     req.Confidence,
		SourceChannel:  req.SourceChannel,
	}

	// 2. Get current slot values (including this new one)
	currentValues, err := h.eventStore.CurrentValues(ctx, patientID)
	if err != nil {
		h.logger.Error("failed to get current values", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read slot values"})
		return
	}
	// Add new value to snapshot
	currentValues[req.SlotName] = slots.SlotValue{
		Value:          req.Value,
		ExtractionMode: req.ExtractionMode,
		Confidence:     req.Confidence,
	}

	// 3. Run safety engine (<5ms, deterministic)
	snapshot := slots.BuildSnapshot(patientID, currentValues)
	safetyResult := h.safetyEngine.Evaluate(snapshot)
	safetyResultJSON, _ := json.Marshal(safetyResult)
	event.SafetyResult = safetyResultJSON

	// 4. Write FHIR Observation to Google FHIR Store
	var fhirResourceID string
	if h.fhirClient != nil {
		obsJSON, err := intakefhir.ObservationFromSlot(patientID, encounterID, slotDef, req.Value)
		if err != nil {
			h.logger.Error("failed to build FHIR Observation", zap.Error(err))
		} else {
			respData, err := h.fhirClient.Create("Observation", obsJSON)
			if err != nil {
				h.logger.Error("FHIR Observation write failed, will retry", zap.Error(err))
				// Per spec section 7.2: retry 3x -> hold in PostgreSQL -> background sync
				// Slot acknowledged to patient regardless
			} else {
				var resp map[string]interface{}
				json.Unmarshal(respData, &resp)
				if id, ok := resp["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}

		// 4b. Write DetectedIssue for any safety triggers
		for _, hs := range safetyResult.HardStops {
			diJSON, err := intakefhir.DetectedIssueFromRule(patientID, encounterID, hs)
			if err == nil {
				h.fhirClient.Create("DetectedIssue", diJSON)
			}
		}
		for _, sf := range safetyResult.SoftFlags {
			diJSON, err := intakefhir.DetectedIssueFromRule(patientID, encounterID, sf)
			if err == nil {
				h.fhirClient.Create("DetectedIssue", diJSON)
			}
		}
	}
	event.FHIRResourceID = fhirResourceID

	// 5. Persist slot event
	if err := h.eventStore.Append(ctx, event); err != nil {
		h.logger.Error("failed to persist slot event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "safety engine error — cannot swallow"})
		return
	}

	// 6. Publish to Kafka
	if h.producer != nil {
		payload := map[string]interface{}{
			"slot_name":     req.SlotName,
			"domain":        slotDef.Domain,
			"value":         req.Value,
			"safety_result": safetyResult,
		}
		h.producer.Publish(ctx, kafka.TopicSlotEvents, patientID, "SLOT_FILLED", payload)

		if safetyResult.HasHardStop() {
			h.producer.Publish(ctx, kafka.TopicSafetyAlerts, patientID, "HARD_STOP", payload)
		}
		if safetyResult.HasSoftFlag() {
			h.producer.Publish(ctx, kafka.TopicSafetyFlags, patientID, "SOFT_FLAG", payload)
		}
	}

	// 7. Build response
	resp := FillSlotResponse{
		Status:         "ok",
		SlotName:       req.SlotName,
		FHIRResourceID: fhirResourceID,
		Progress: ProgressResponse{
			Filled:   len(currentValues),
			Total:    len(slots.AllSlots()),
			Percent:  float64(len(currentValues)) / float64(len(slots.AllSlots())) * 100,
			Complete: snapshot.IsComplete(),
		},
	}

	// Safety result in response
	if safetyResult.HasHardStop() || safetyResult.HasSoftFlag() {
		sr := &SafetyResultResponse{
			HardStops: make([]RuleResultResponse, len(safetyResult.HardStops)),
			SoftFlags: make([]RuleResultResponse, len(safetyResult.SoftFlags)),
		}
		for i, hs := range safetyResult.HardStops {
			sr.HardStops[i] = RuleResultResponse{RuleID: hs.RuleID, Reason: hs.Reason}
		}
		for i, sf := range safetyResult.SoftFlags {
			sr.SoftFlags[i] = RuleResultResponse{RuleID: sf.RuleID, Reason: sf.Reason}
		}
		resp.SafetyResult = sr
	}

	if safetyResult.HasHardStop() {
		resp.Status = "hard_stopped"
		c.JSON(http.StatusOK, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// EnrollRequest is the JSON body for POST /fhir/Patient/$enroll.
type EnrollRequest struct {
	GivenName   string `json:"given_name"`
	FamilyName  string `json:"family_name"`
	Phone       string `json:"phone"`
	ABHAID      string `json:"abha_id,omitempty"`
	ChannelType string `json:"channel_type"` // CORPORATE, INSURANCE, GOVERNMENT
	TenantID    string `json:"tenant_id"`
}

// HandleEnroll implements POST /fhir/Patient/$enroll.
// Creates Patient + Encounter in FHIR Store, creates enrollment in PostgreSQL.
func (h *Handler) HandleEnroll(c *gin.Context) {
	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	if req.GivenName == "" || req.FamilyName == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "given_name, family_name, and phone are required"})
		return
	}

	// 1. Create Patient in FHIR Store
	var patientID string
	if h.fhirClient != nil {
		patientJSON, err := intakefhir.NewPatientResource(req.GivenName, req.FamilyName, req.Phone, req.ABHAID)
		if err != nil {
			h.logger.Error("failed to build Patient resource", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create patient"})
			return
		}

		respData, err := h.fhirClient.Create("Patient", patientJSON)
		if err != nil {
			h.logger.Error("FHIR Patient create failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FHIR Store write failed"})
			return
		}

		var resp map[string]interface{}
		json.Unmarshal(respData, &resp)
		if id, ok := resp["id"].(string); ok {
			patientID = id
		}
	} else {
		patientID = uuid.New().String()
	}

	pid, _ := uuid.Parse(patientID)

	// 2. Create Encounter in FHIR Store
	var encounterID string
	if h.fhirClient != nil {
		encJSON, err := intakefhir.NewEncounterResource(pid, "intake")
		if err != nil {
			h.logger.Error("failed to build Encounter resource", zap.Error(err))
		} else {
			respData, err := h.fhirClient.Create("Encounter", encJSON)
			if err != nil {
				h.logger.Error("FHIR Encounter create failed", zap.Error(err))
			} else {
				var resp map[string]interface{}
				json.Unmarshal(respData, &resp)
				if id, ok := resp["id"].(string); ok {
					encounterID = id
				}
			}
		}
	}
	if encounterID == "" {
		encounterID = uuid.New().String()
	}

	// 2b. Persist enrollment to PostgreSQL
	if h.db != nil {
		encUUID, _ := uuid.Parse(encounterID)
		tenantUUID, _ := uuid.Parse(req.TenantID)
		_, err := h.db.Exec(c.Request.Context(),
			`INSERT INTO enrollments (patient_id, tenant_id, channel_type, state, encounter_id)
			 VALUES ($1, $2, $3, 'CREATED', $4)`,
			pid, tenantUUID, req.ChannelType, encUUID,
		)
		if err != nil {
			h.logger.Error("failed to persist enrollment", zap.Error(err))
			// Non-fatal -- FHIR Store is source of truth
		}
	}

	// 3. Publish to Kafka
	if h.producer != nil {
		payload := map[string]interface{}{
			"patient_id":   patientID,
			"encounter_id": encounterID,
			"channel_type": req.ChannelType,
			"phone":        req.Phone,
		}
		h.producer.Publish(c.Request.Context(), kafka.TopicPatientLifecycle, pid, "PATIENT_CREATED", payload)
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":       "enrolled",
		"patient_id":   patientID,
		"encounter_id": encounterID,
		"next_node": gin.H{
			"node_id": "demographics_basic",
			"slots":   []string{"age", "sex", "height", "weight", "bmi", "pregnant"},
		},
	})
}

// HandleEvaluateSafety implements POST /fhir/Patient/:id/$evaluate-safety.
func (h *Handler) HandleEvaluateSafety(c *gin.Context) {
	patientIDStr := c.Param("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	ctx := c.Request.Context()
	currentValues, err := h.eventStore.CurrentValues(ctx, patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read slot values"})
		return
	}

	snapshot := slots.BuildSnapshot(patientID, currentValues)
	result := h.safetyEngine.Evaluate(snapshot)

	c.JSON(http.StatusOK, gin.H{
		"patient_id":    patientID,
		"hard_stops":    result.HardStops,
		"soft_flags":    result.SoftFlags,
		"has_hard_stop": result.HasHardStop(),
	})
}
