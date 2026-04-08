package app

import (
	"encoding/json"
	"fmt"
	"math"
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

	// Reject client-submitted derived slots (e.g. BMI must be server-computed).
	if slotDef.Derived {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("%s is a derived slot and cannot be filled directly", req.SlotName),
		})
		return
	}

	// Get patientID from encounter (in production, look up from DB; here use header)
	patientIDStr := c.GetHeader("X-Patient-ID")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Patient-ID header required"})
		return
	}

	ctx := c.Request.Context()

	// Look up tenant context from enrollment for Kafka events (ISS-5).
	var tenantID, channelType string
	if h.db != nil {
		_ = h.db.QueryRow(ctx,
			`SELECT COALESCE(tenant_id::text,''), COALESCE(channel_type,'') FROM enrollments WHERE patient_id = $1`,
			patientID,
		).Scan(&tenantID, &channelType)
	}

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

	// 2b. Auto-derive BMI when both height and weight are present.
	if req.SlotName == "height" || req.SlotName == "weight" {
		if hSV, hOK := currentValues["height"]; hOK {
			if wSV, wOK := currentValues["weight"]; wOK {
				var heightCM, weightKG float64
				hErr := json.Unmarshal(hSV.Value, &heightCM)
				wErr := json.Unmarshal(wSV.Value, &weightKG)
				if hErr == nil && wErr == nil && heightCM > 0 {
					heightM := heightCM / 100.0
					bmi := weightKG / (heightM * heightM)
					bmi = math.Round(bmi*10) / 10 // round to 1 decimal
					bmiJSON, _ := json.Marshal(bmi)
					currentValues["bmi"] = slots.SlotValue{
						Value:          bmiJSON,
						ExtractionMode: "DERIVED",
						Confidence:     1.0,
					}
					// Persist derived BMI as a slot event.
					bmiEvent := slots.SlotEvent{
						PatientID:      patientID,
						SlotName:       "bmi",
						Domain:         "demographics",
						Value:          bmiJSON,
						ExtractionMode: "DERIVED",
						Confidence:     1.0,
						SourceChannel:  "SYSTEM",
						FHIRResourceID: "",
					}
					_ = h.eventStore.Append(ctx, bmiEvent)
					h.logger.Info("BMI auto-derived",
						zap.Float64("bmi", bmi),
						zap.Float64("height_cm", heightCM),
						zap.Float64("weight_kg", weightKG),
					)
				}
			}
		}
	}

	// 3. Run safety engine (<5ms, deterministic)
	snapshot := slots.BuildSnapshot(patientID, currentValues)
	safetyResult := h.safetyEngine.Evaluate(snapshot)
	safetyResultJSON, _ := json.Marshal(safetyResult)
	event.SafetyResult = safetyResultJSON

	// 4. Write to FHIR Store as Observation (demographics are set at patient creation).
	var fhirResourceID string
	if h.fhirClient != nil {
		obsJSON, err := intakefhir.ObservationFromSlot(patientID, encounterID, slotDef, req.Value)
		if err != nil {
			h.logger.Error("failed to build FHIR Observation", zap.Error(err))
		} else {
			respData, err := h.fhirClient.Create("Observation", obsJSON)
			if err != nil {
				h.logger.Error("FHIR Observation write failed, will retry", zap.Error(err))
			} else {
				var resp map[string]interface{}
				json.Unmarshal(respData, &resp)
				if id, ok := resp["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}

		// 4b. Write DetectedIssue for any safety triggers.
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

	// 6. Publish to Kafka (ISS-5: tenant context injected via _tenant_id/_channel_type)
	if h.producer != nil {
		payload := map[string]interface{}{
			"slot_name":      req.SlotName,
			"domain":         slotDef.Domain,
			"value":          req.Value,
			"safety_result":  safetyResult,
			"_tenant_id":     tenantID,
			"_channel_type":  channelType,
		}
		h.producer.Publish(ctx, kafka.TopicSlotEvents, patientID, "SLOT_FILLED", payload)

		if safetyResult.HasHardStop() {
			hsPayload := map[string]interface{}{
				"slot_name":      req.SlotName,
				"domain":         slotDef.Domain,
				"value":          req.Value,
				"safety_result":  safetyResult,
				"_tenant_id":     tenantID,
				"_channel_type":  channelType,
			}
			h.producer.Publish(ctx, kafka.TopicSafetyAlerts, patientID, "HARD_STOP", hsPayload)
		}
		if safetyResult.HasSoftFlag() {
			sfPayload := map[string]interface{}{
				"slot_name":      req.SlotName,
				"domain":         slotDef.Domain,
				"value":          req.Value,
				"safety_result":  safetyResult,
				"_tenant_id":     tenantID,
				"_channel_type":  channelType,
			}
			h.producer.Publish(ctx, kafka.TopicSafetyFlags, patientID, "SOFT_FLAG", sfPayload)
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

// ---------------------------------------------------------------------------
// PUT /fhir/Patient/:id — Edit patient demographics directly
// ---------------------------------------------------------------------------

// UpdatePatientRequest is the JSON body for PUT /fhir/Patient/:id.
type UpdatePatientRequest struct {
	Age             *int    `json:"age,omitempty"`
	Gender          *string `json:"gender,omitempty"`
	Ethnicity       *string `json:"ethnicity,omitempty"`
	PrimaryLanguage *string `json:"primary_language,omitempty"`
	GivenName       *string `json:"given_name,omitempty"`
	FamilyName      *string `json:"family_name,omitempty"`
}

// HandleUpdatePatient edits demographics on the FHIR Patient resource and PostgreSQL.
func (h *Handler) HandleUpdatePatient(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req UpdatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Look up tenant context for Kafka (ISS-5).
	var demoTenantID, demoChannelType string
	if h.db != nil {
		_ = h.db.QueryRow(ctx,
			`SELECT COALESCE(tenant_id::text,''), COALESCE(channel_type,'') FROM enrollments WHERE patient_id = $1`,
			patientID,
		).Scan(&demoTenantID, &demoChannelType)
	}

	// Build demographics patch.
	demo := intakefhir.PatientDemographics{}
	updated := []string{}
	if req.Age != nil {
		demo.Age = *req.Age
		updated = append(updated, "age")
	}
	if req.Gender != nil {
		demo.Gender = *req.Gender
		updated = append(updated, "gender")
	}
	if req.Ethnicity != nil {
		demo.Ethnicity = *req.Ethnicity
		updated = append(updated, "ethnicity")
	}
	if req.PrimaryLanguage != nil {
		demo.PrimaryLanguage = *req.PrimaryLanguage
		updated = append(updated, "primary_language")
	}

	if len(updated) == 0 && req.GivenName == nil && req.FamilyName == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	// Update FHIR Store.
	if h.fhirClient != nil {
		existingData, err := h.fhirClient.Read("Patient", patientID.String())
		if err != nil {
			h.logger.Error("failed to read Patient", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found in FHIR Store"})
			return
		}

		// Apply demographics patch.
		updatedJSON := existingData
		if len(updated) > 0 {
			updatedJSON, err = intakefhir.UpdatePatientDemographics(existingData, demo)
			if err != nil {
				h.logger.Error("failed to patch Patient", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update patient"})
				return
			}
		}

		// Apply name changes if provided.
		if req.GivenName != nil || req.FamilyName != nil {
			var patient map[string]interface{}
			json.Unmarshal(updatedJSON, &patient)
			if names, ok := patient["name"].([]interface{}); ok && len(names) > 0 {
				if name, ok := names[0].(map[string]interface{}); ok {
					if req.GivenName != nil {
						name["given"] = []string{*req.GivenName}
						updated = append(updated, "given_name")
					}
					if req.FamilyName != nil {
						name["family"] = *req.FamilyName
						updated = append(updated, "family_name")
					}
				}
			}
			updatedJSON, _ = json.Marshal(patient)
		}

		_, err = h.fhirClient.Update("Patient", patientID.String(), updatedJSON)
		if err != nil {
			h.logger.Error("FHIR Patient update failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FHIR update failed"})
			return
		}
	}

	// Also persist identity slots to PostgreSQL slot_events for consistency.
	if h.db != nil && h.eventStore != nil {
		for _, field := range updated {
			if !slots.IsPatientSlot(field) {
				continue
			}
			var val json.RawMessage
			switch field {
			case "age":
				val, _ = json.Marshal(*req.Age)
			case "gender":
				val, _ = json.Marshal(*req.Gender)
			case "ethnicity":
				val, _ = json.Marshal(*req.Ethnicity)
			case "primary_language":
				val, _ = json.Marshal(*req.PrimaryLanguage)
			}
			event := slots.SlotEvent{
				PatientID:      patientID,
				SlotName:       field,
				Domain:         "demographics",
				Value:          val,
				ExtractionMode: "EDIT",
				SourceChannel:  "APP",
				FHIRResourceID: patientID.String(),
			}
			_ = h.eventStore.Append(ctx, event)
		}
	}

	// Publish to Kafka with actual values (ISS-4).
	if h.producer != nil {
		changedValues := map[string]interface{}{}
		if req.Age != nil {
			changedValues["age"] = *req.Age
		}
		if req.Gender != nil {
			changedValues["gender"] = *req.Gender
		}
		if req.Ethnicity != nil {
			changedValues["ethnicity"] = *req.Ethnicity
		}
		if req.PrimaryLanguage != nil {
			changedValues["primary_language"] = *req.PrimaryLanguage
		}
		if req.GivenName != nil {
			changedValues["given_name"] = *req.GivenName
		}
		if req.FamilyName != nil {
			changedValues["family_name"] = *req.FamilyName
		}
		payload := map[string]interface{}{
			"patient_id":     patientID.String(),
			"updated_fields": updated,
			"values":         changedValues,
			"_tenant_id":     demoTenantID,
			"_channel_type":  demoChannelType,
		}
		h.producer.Publish(ctx, kafka.TopicPatientLifecycle, patientID, "PATIENT_DEMOGRAPHICS_UPDATED", payload)
	}

	h.logger.Info("patient demographics updated",
		zap.String("patient_id", patientID.String()),
		zap.Strings("fields", updated),
	)
	c.JSON(http.StatusOK, gin.H{
		"status":         "updated",
		"patient_id":     patientID.String(),
		"updated_fields": updated,
	})
}

// ---------------------------------------------------------------------------
// POST /fhir/Patient — Create a new patient (one-time registration)
// ---------------------------------------------------------------------------

// CreatePatientRequest is the JSON body for POST /fhir/Patient.
// Includes identity demographics so they are set on the Patient resource at registration.
type CreatePatientRequest struct {
	GivenName       string  `json:"given_name"`
	FamilyName      string  `json:"family_name"`
	Phone           string  `json:"phone"`
	ABHAID          string  `json:"abha_id,omitempty"`
	Email           string  `json:"email,omitempty"`
	Age             *int    `json:"age,omitempty"`
	Gender          *string `json:"gender,omitempty"`
	Ethnicity       *string `json:"ethnicity,omitempty"`
	PrimaryLanguage *string `json:"primary_language,omitempty"`
}

// HandleCreatePatient registers a new patient in the FHIR Store.
// If a patient with the same phone already exists, returns the existing patient.
func (h *Handler) HandleCreatePatient(c *gin.Context) {
	var req CreatePatientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}
	if req.GivenName == "" || req.FamilyName == "" || req.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "given_name, family_name, and phone are required"})
		return
	}

	// Check for existing patient by phone in FHIR Store.
	if h.fhirClient != nil {
		searchData, err := h.fhirClient.Search("Patient", map[string]string{"phone": req.Phone})
		if err == nil {
			var bundle map[string]interface{}
			if json.Unmarshal(searchData, &bundle) == nil {
				if total, ok := bundle["total"].(float64); ok && total > 0 {
					if entries, ok := bundle["entry"].([]interface{}); ok && len(entries) > 0 {
						if entry, ok := entries[0].(map[string]interface{}); ok {
							if resource, ok := entry["resource"].(map[string]interface{}); ok {
								if id, ok := resource["id"].(string); ok {
									h.logger.Info("existing patient found", zap.String("patient_id", id), zap.String("phone", req.Phone))
									c.JSON(http.StatusOK, gin.H{
										"status":     "existing",
										"patient_id": id,
										"message":    "Patient already registered with this phone number",
									})
									return
								}
							}
						}
					}
				}
			}
		}
	}

	// Create new patient in FHIR Store.
	var patientID string
	if h.fhirClient != nil {
		patientJSON, err := intakefhir.NewPatientResource(req.GivenName, req.FamilyName, req.Phone, req.ABHAID, req.Email)
		if err != nil {
			h.logger.Error("failed to build Patient resource", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create patient"})
			return
		}

		// Apply identity demographics onto the Patient resource before writing.
		demo := intakefhir.PatientDemographics{}
		hasDemographics := false
		if req.Age != nil {
			demo.Age = *req.Age
			hasDemographics = true
		}
		if req.Gender != nil {
			demo.Gender = *req.Gender
			hasDemographics = true
		}
		if req.Ethnicity != nil {
			demo.Ethnicity = *req.Ethnicity
			hasDemographics = true
		}
		if req.PrimaryLanguage != nil {
			demo.PrimaryLanguage = *req.PrimaryLanguage
			hasDemographics = true
		}
		if hasDemographics {
			patientJSON, err = intakefhir.UpdatePatientDemographics(patientJSON, demo)
			if err != nil {
				h.logger.Error("failed to apply demographics", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply demographics"})
				return
			}
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

	ctx := c.Request.Context()

	// Persist identity demographics to PostgreSQL slot_events for triple-sink.
	pid, _ := uuid.Parse(patientID)
	if h.db != nil && h.eventStore != nil {
		demoSlots := map[string]interface{}{
			"age": req.Age, "sex": req.Gender,
			"ethnicity": req.Ethnicity, "primary_language": req.PrimaryLanguage,
		}
		for slotName, val := range demoSlots {
			if val == nil {
				continue
			}
			valJSON, _ := json.Marshal(val)
			event := slots.SlotEvent{
				PatientID:      pid,
				SlotName:       slotName,
				Domain:         "demographics",
				Value:          valJSON,
				ExtractionMode: "REGISTRATION",
				SourceChannel:  "APP",
				FHIRResourceID: patientID,
			}
			_ = h.eventStore.Append(ctx, event)
		}
	}

	// Publish to Kafka.
	if h.producer != nil {
		payload := map[string]interface{}{
			"patient_id": patientID,
			"phone":      req.Phone,
		}
		if req.Age != nil {
			payload["age"] = *req.Age
		}
		if req.Gender != nil {
			payload["gender"] = *req.Gender
		}
		h.producer.Publish(ctx, kafka.TopicPatientLifecycle, pid, "PATIENT_CREATED", payload)
	}

	h.logger.Info("patient created", zap.String("patient_id", patientID))
	c.JSON(http.StatusCreated, gin.H{
		"status":     "created",
		"patient_id": patientID,
	})
}

// ---------------------------------------------------------------------------
// POST /fhir/Patient/:id/Encounter — Create an encounter for an existing patient
// ---------------------------------------------------------------------------

// CreateEncounterRequest is the JSON body for POST /fhir/Patient/:id/Encounter.
type CreateEncounterRequest struct {
	Type string `json:"type"` // intake, lab, checkup, followup
}

// HandleCreateEncounter creates a new FHIR Encounter for an existing patient.
func (h *Handler) HandleCreateEncounter(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req CreateEncounterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = "intake"
	}

	validTypes := map[string]bool{"intake": true, "lab": true, "checkup": true, "followup": true}
	if !validTypes[req.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: intake, lab, checkup, followup"})
		return
	}

	// Create Encounter in FHIR Store.
	var encounterID string
	if h.fhirClient != nil {
		encJSON, err := intakefhir.NewEncounterResource(patientID, req.Type)
		if err != nil {
			h.logger.Error("failed to build Encounter resource", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build encounter"})
			return
		}
		respData, err := h.fhirClient.Create("Encounter", encJSON)
		if err != nil {
			h.logger.Error("FHIR Encounter create failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "FHIR Encounter write failed"})
			return
		}
		var resp map[string]interface{}
		json.Unmarshal(respData, &resp)
		if id, ok := resp["id"].(string); ok {
			encounterID = id
		}
	} else {
		encounterID = uuid.New().String()
	}

	h.logger.Info("encounter created",
		zap.String("encounter_id", encounterID),
		zap.String("patient_id", patientID.String()),
		zap.String("type", req.Type),
	)
	c.JSON(http.StatusCreated, gin.H{
		"status":       "created",
		"encounter_id": encounterID,
		"patient_id":   patientID.String(),
		"type":         req.Type,
	})
}

// ---------------------------------------------------------------------------
// POST /fhir/Patient/:id/$enroll — Enroll patient into the intake program
// ---------------------------------------------------------------------------

// EnrollRequest is the JSON body for POST /fhir/Patient/:id/$enroll.
type EnrollRequest struct {
	EncounterID string `json:"encounter_id"`
	ChannelType string `json:"channel_type"` // CORPORATE, INSURANCE, GOVERNMENT
	TenantID    string `json:"tenant_id"`
}

// HandleEnroll enrolls an existing patient (with an existing encounter) into the intake program.
// Creates enrollment in PostgreSQL and publishes to Kafka.
func (h *Handler) HandleEnroll(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}
	if req.EncounterID == "" || req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "encounter_id and tenant_id are required"})
		return
	}

	encounterID, err := uuid.Parse(req.EncounterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter_id"})
		return
	}

	// Check for active enrollment — prevent double enrollment.
	if h.db != nil {
		var activeState string
		err := h.db.QueryRow(c.Request.Context(),
			`SELECT state FROM enrollments WHERE patient_id = $1`, patientID,
		).Scan(&activeState)
		if err == nil && activeState != "ENROLLED" && activeState != "DISCHARGED" {
			// Already in-progress.
			var existingEncID uuid.UUID
			_ = h.db.QueryRow(c.Request.Context(),
				`SELECT encounter_id FROM enrollments WHERE patient_id = $1`, patientID,
			).Scan(&existingEncID)
			c.JSON(http.StatusOK, gin.H{
				"status":       "already_enrolled",
				"patient_id":   patientID.String(),
				"encounter_id": existingEncID.String(),
				"state":        activeState,
			})
			return
		}
	}

	// Persist enrollment to PostgreSQL.
	if h.db != nil {
		tenantUUID, _ := uuid.Parse(req.TenantID)
		_, err := h.db.Exec(c.Request.Context(),
			`INSERT INTO enrollments (patient_id, tenant_id, channel_type, state, encounter_id)
			 VALUES ($1, $2, $3, 'CREATED', $4)
			 ON CONFLICT (patient_id) DO UPDATE SET
			   encounter_id = EXCLUDED.encounter_id,
			   channel_type = EXCLUDED.channel_type,
			   state = 'CREATED',
			   updated_at = now()`,
			patientID, tenantUUID, req.ChannelType, encounterID,
		)
		if err != nil {
			h.logger.Error("failed to persist enrollment", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create enrollment"})
			return
		}
	}

	// Publish to Kafka (ISS-5: tenant context in envelope).
	if h.producer != nil {
		payload := map[string]interface{}{
			"patient_id":    patientID.String(),
			"encounter_id":  encounterID.String(),
			"channel_type":  req.ChannelType,
			"_tenant_id":    req.TenantID,
			"_channel_type": req.ChannelType,
		}
		h.producer.Publish(c.Request.Context(), kafka.TopicPatientLifecycle, patientID, "PATIENT_ENROLLED", payload)
	}

	h.logger.Info("patient enrolled",
		zap.String("patient_id", patientID.String()),
		zap.String("encounter_id", encounterID.String()),
	)
	c.JSON(http.StatusCreated, gin.H{
		"status":       "enrolled",
		"patient_id":   patientID.String(),
		"encounter_id": encounterID.String(),
		"next_node": gin.H{
			"node_id": "demographics_basic",
			"slots":   []string{"age", "sex", "height", "weight"},
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

// HandleSearchPatient implements GET /fhir/Patient?phone=xxx&email=xxx.
// Returns the patient ID if found by phone number or email address.
func (h *Handler) HandleSearchPatient(c *gin.Context) {
	phone := c.Query("phone")
	email := c.Query("email")

	if phone == "" && email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone or email query parameter is required"})
		return
	}

	if h.fhirClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "FHIR Store not configured"})
		return
	}

	// Build search parameters — prefer phone, fall back to email.
	params := map[string]string{}
	if phone != "" {
		params["phone"] = phone
	} else {
		params["email"] = email
	}

	searchData, err := h.fhirClient.Search("Patient", params)
	if err != nil {
		h.logger.Error("FHIR Patient search failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	var bundle map[string]interface{}
	if err := json.Unmarshal(searchData, &bundle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid FHIR response"})
		return
	}

	total, _ := bundle["total"].(float64)
	if total == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "not_found",
			"message": "No patient found with the given identifier",
		})
		return
	}

	// Extract patient details from the first match.
	entries, _ := bundle["entry"].([]interface{})
	if len(entries) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"status": "not_found"})
		return
	}
	entry, _ := entries[0].(map[string]interface{})
	resource, _ := entry["resource"].(map[string]interface{})
	patientID, _ := resource["id"].(string)

	// Extract basic info for the response.
	resp := gin.H{
		"status":     "found",
		"patient_id": patientID,
	}
	if names, ok := resource["name"].([]interface{}); ok && len(names) > 0 {
		if name, ok := names[0].(map[string]interface{}); ok {
			if given, ok := name["given"].([]interface{}); ok && len(given) > 0 {
				resp["given_name"] = given[0]
			}
			if family, ok := name["family"].(string); ok {
				resp["family_name"] = family
			}
		}
	}
	if gender, ok := resource["gender"].(string); ok {
		resp["gender"] = gender
	}
	if birthDate, ok := resource["birthDate"].(string); ok {
		resp["birth_date"] = birthDate
	}

	c.JSON(http.StatusOK, resp)
}
