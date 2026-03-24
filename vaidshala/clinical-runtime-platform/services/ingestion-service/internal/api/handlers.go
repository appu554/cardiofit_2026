package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/adapters/devices"
	"github.com/cardiofit/ingestion-service/internal/adapters/patient_reported"
	"github.com/cardiofit/ingestion-service/internal/canonical"
	fhirmapper "github.com/cardiofit/ingestion-service/internal/fhir"
	"github.com/cardiofit/ingestion-service/internal/metrics"
)

// handleFHIRObservation handles POST /fhir/Observation.
// Accepts a FHIR-like observation payload, converts to canonical, runs pipeline.
func (s *Server) handleFHIRObservation(c *gin.Context) {
	start := time.Now()
	metrics.MessagesReceived.WithLabelValues("FHIR", "direct", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Parse the incoming observation into canonical form
	var incoming struct {
		PatientID string  `json:"patient_id"`
		TenantID  string  `json:"tenant_id"`
		LOINCCode string  `json:"loinc_code"`
		Value     float64 `json:"value"`
		Unit      string  `json:"unit"`
		Timestamp string  `json:"timestamp,omitempty"`
	}
	if err := json.Unmarshal(body, &incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	patientID, err := uuid.Parse(incoming.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient_id"})
		return
	}

	tenantID := uuid.Nil
	if incoming.TenantID != "" {
		tenantID, _ = uuid.Parse(incoming.TenantID)
	}

	ts := time.Now().UTC()
	if incoming.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, incoming.Timestamp); err == nil {
			ts = parsed
		}
	}

	obs := canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       patientID,
		TenantID:        tenantID,
		SourceType:      canonical.SourceEHR,
		SourceID:        "fhir_direct",
		ObservationType: canonical.ObsGeneral,
		LOINCCode:       incoming.LOINCCode,
		Value:           incoming.Value,
		Unit:            incoming.Unit,
		Timestamp:       ts,
		RawPayload:      body,
	}

	results, err := s.orchestrator.Process(c.Request.Context(), []canonical.CanonicalObservation{obs})
	if err != nil {
		s.logger.Error("pipeline processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "observation rejected -- check DLQ for details",
			"dlq_url": "/fhir/OperationOutcome?category=dlq",
		})
		return
	}

	// Write to FHIR Store if available
	var fhirResourceID string
	if s.fhirClient != nil && len(results[0].RawPayload) > 0 {
		resp, err := s.fhirClient.Create("Observation", results[0].RawPayload)
		if err != nil {
			s.logger.Error("FHIR Store write failed", zap.Error(err))
			// Continue -- Kafka publish is more important than FHIR Store sync
		} else {
			var created map[string]interface{}
			if json.Unmarshal(resp, &created) == nil {
				if id, ok := created["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}
	}

	// Publish to Kafka
	if s.kafkaProducer != nil && s.topicRouter != nil {
		topic, key, err := s.topicRouter.Route(c.Request.Context(), &results[0])
		if err == nil {
			_ = s.kafkaProducer.Publish(
				c.Request.Context(), topic, key, &results[0],
				"Observation", fhirResourceID, s.config.Kafka.Brokers,
			)
		}
	}

	metrics.PipelineDuration.WithLabelValues(string(obs.SourceType), "total").Observe(time.Since(start).Seconds())

	c.JSON(http.StatusCreated, gin.H{
		"status":           "accepted",
		"observation_id":   results[0].ID.String(),
		"fhir_resource_id": fhirResourceID,
		"quality_score":    results[0].QualityScore,
		"flags":            results[0].Flags,
	})
}

// handleDeviceIngest handles POST /devices.
func (s *Server) handleDeviceIngest(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("DEVICE", "", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload devices.DevicePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := devices.NewDeviceAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	// Write to FHIR Store and Kafka for each result
	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
		"total":     len(observations),
		"rejected":  len(observations) - len(results),
	})
}

// handleAppCheckin handles POST /app-checkin (patient self-report from Flutter app).
func (s *Server) handleAppCheckin(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("PATIENT_REPORTED", "app_checkin", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload patient_reported.AppCheckinPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := patient_reported.NewAppCheckinAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
		"total":     len(observations),
	})
}

// handleWhatsAppIngest handles POST /whatsapp (NLU-parsed WhatsApp messages).
func (s *Server) handleWhatsAppIngest(c *gin.Context) {
	metrics.MessagesReceived.WithLabelValues("PATIENT_REPORTED", "whatsapp", "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	var payload patient_reported.WhatsAppNLUPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON: " + err.Error()})
		return
	}

	adapter := patient_reported.NewWhatsAppAdapter(s.logger)
	observations, err := adapter.Parse(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
	})
}


// handleWearableIngest handles POST /wearables/:provider.
// Unlike the standalone wearable.Handler (which only returns HTTP), this
// method feeds wearable observations through the full pipeline:
// adapter → orchestrator (normalize + validate + map) → FHIR Store → Kafka.
func (s *Server) handleWearableIngest(c *gin.Context) {
	provider := c.Param("provider")
	metrics.MessagesReceived.WithLabelValues("WEARABLE", provider, "").Inc()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Delegate JSON → CanonicalObservation conversion to the wearable adapter
	observations, err := s.wearableHandler.ConvertPayload(provider, body)
	if err != nil {
		s.logger.Error("wearable conversion failed",
			zap.String("provider", provider),
			zap.Error(err),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	// Run through the full pipeline (normalize → validate → map → route)
	results, err := s.orchestrator.Process(c.Request.Context(), observations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pipeline processing failed"})
		return
	}

	// Publish each result to FHIR Store + Kafka
	for i := range results {
		s.publishResult(c, &results[i])
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "accepted",
		"processed": len(results),
		"total":     len(observations),
	})
}

// publishResult writes an observation to FHIR Store and publishes via outbox (or Kafka fallback).
func (s *Server) publishResult(c *gin.Context, obs *canonical.CanonicalObservation) {
	// Check for critical values
	isCritical := false
	for _, flag := range obs.Flags {
		if flag == canonical.FlagCriticalValue {
			metrics.CriticalValues.WithLabelValues(string(obs.ObservationType), obs.TenantID.String()).Inc()
			isCritical = true
			break
		}
	}

	// Map to FHIR if not already mapped
	if len(obs.RawPayload) == 0 {
		mapper := fhirmapper.NewCompositeMapper(s.logger)
		fhirJSON, err := mapper.MapToFHIR(c.Request.Context(), obs)
		if err != nil {
			s.logger.Error("FHIR mapping failed", zap.Error(err))
			return
		}
		obs.RawPayload = fhirJSON
	}

	// Write to FHIR Store
	var fhirResourceID string
	if s.fhirClient != nil {
		resourceType := "Observation"
		if obs.ObservationType == canonical.ObsMedications {
			resourceType = "MedicationStatement"
		}
		resp, err := s.fhirClient.Create(resourceType, obs.RawPayload)
		if err != nil {
			s.logger.Error("FHIR Store write failed",
				zap.String("patient_id", obs.PatientID.String()),
				zap.Error(err),
			)
		} else {
			var created map[string]interface{}
			if json.Unmarshal(resp, &created) == nil {
				if id, ok := created["id"].(string); ok {
					fhirResourceID = id
				}
			}
		}

		// For lab results, also create a DiagnosticReport
		if obs.ObservationType == canonical.ObsLabs && fhirResourceID != "" {
			drJSON, err := fhirmapper.MapDiagnosticReport(obs, fhirResourceID)
			if err == nil {
				_, _ = s.fhirClient.Create("DiagnosticReport", drJSON)
			}
		}
	}

	// Publish via outbox (preferred) or direct Kafka (fallback).
	// Outbox path uses SaveAndPublish / SaveAndPublishBatch for at-least-once
	// delivery via the Global Outbox Service's polling publisher.
	if s.outboxPublisher != nil {
		var err error
		if isCritical {
			err = s.outboxPublisher.PublishCritical(c.Request.Context(), obs, fhirResourceID)
		} else {
			err = s.outboxPublisher.Publish(c.Request.Context(), obs, fhirResourceID)
		}
		if err != nil {
			metrics.DLQMessages.WithLabelValues("OUTBOX", string(obs.SourceType)).Inc()
			s.logger.Error("outbox publish failed",
				zap.String("patient_id", obs.PatientID.String()),
				zap.Error(err),
			)
		}
		return
	}

	// Fallback: direct Kafka (used when OUTBOX_ENABLED=false or SDK init failed)
	if s.kafkaProducer != nil && s.topicRouter != nil {
		topic, key, err := s.topicRouter.Route(c.Request.Context(), obs)
		if err == nil {
			resourceType := "Observation"
			if obs.ObservationType == canonical.ObsMedications {
				resourceType = "MedicationStatement"
			}
			if pubErr := s.kafkaProducer.Publish(
				c.Request.Context(), topic, key, obs,
				resourceType, fhirResourceID, s.config.Kafka.Brokers,
			); pubErr != nil {
				metrics.DLQMessages.WithLabelValues("PUBLISH", string(obs.SourceType)).Inc()
				s.logger.Error("Kafka publish failed",
					zap.String("topic", topic),
					zap.Error(pubErr),
				)
			}

			// Dual-publish critical values to the safety-critical topic
			if isCritical {
				_ = s.kafkaProducer.Publish(
					c.Request.Context(), "ingestion.safety-critical", key, obs,
					resourceType, fhirResourceID, s.config.Kafka.Brokers,
				)
			}
		}
	}
}
