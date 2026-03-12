// Package consumer provides Kafka event consumption for auto-enrollment
package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/clients"
	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/producer"
)

// Consumer handles Kafka message consumption for auto-enrollment
type Consumer struct {
	consumer      *kafka.Consumer
	config        *config.KafkaConfig
	logger        *logrus.Entry
	repo          *database.Repository
	engine        *criteria.Engine
	producer      *producer.EventProducer
	kb2Client     *clients.KB2Client
	running       bool
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.RWMutex
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(
	cfg *config.KafkaConfig,
	logger *logrus.Entry,
	repo *database.Repository,
	engine *criteria.Engine,
	eventProducer *producer.EventProducer,
	kb2Client *clients.KB2Client,
) (*Consumer, error) {
	logger = logger.WithField("component", "kafka-consumer")

	if !cfg.Enabled {
		logger.Warn("Kafka consumer is disabled")
		return nil, nil
	}

	kafkaConfig := &kafka.ConfigMap{
		"bootstrap.servers":       cfg.Brokers,
		"group.id":                cfg.GroupID,
		"auto.offset.reset":       "earliest",
		"enable.auto.commit":      false,
		"session.timeout.ms":      30000,
		"max.poll.interval.ms":    300000,
		"heartbeat.interval.ms":   3000,
	}

	consumer, err := kafka.NewConsumer(kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	return &Consumer{
		consumer:  consumer,
		config:    cfg,
		logger:    logger,
		repo:      repo,
		engine:    engine,
		producer:  eventProducer,
		kb2Client: kb2Client,
		stopChan:  make(chan struct{}),
	}, nil
}

// Start starts the consumer
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	c.mu.Unlock()

	// Subscribe to topics
	topics := []string{
		models.KafkaTopics.DiagnosisEvents,
		models.KafkaTopics.LabResultEvents,
		models.KafkaTopics.MedicationEvents,
		models.KafkaTopics.ProblemEvents,
	}

	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	c.logger.WithField("topics", topics).Info("Subscribed to Kafka topics")

	// Start consumer loop
	c.wg.Add(1)
	go c.consumeLoop(ctx)

	return nil
}

// Stop stops the consumer
func (c *Consumer) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopChan)
	c.wg.Wait()

	if err := c.consumer.Close(); err != nil {
		c.logger.WithError(err).Error("Failed to close Kafka consumer")
	}

	c.logger.Info("Kafka consumer stopped")
}

// consumeLoop is the main consumption loop
func (c *Consumer) consumeLoop(ctx context.Context) {
	defer c.wg.Done()

	c.logger.Info("Starting Kafka consumer loop")

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Context cancelled, stopping consumer")
			return
		case <-c.stopChan:
			c.logger.Info("Stop signal received, stopping consumer")
			return
		default:
			msg, err := c.consumer.ReadMessage(time.Second * 1)
			if err != nil {
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				c.logger.WithError(err).Warn("Error reading message")
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"topic":     *msg.TopicPartition.Topic,
					"partition": msg.TopicPartition.Partition,
					"offset":    msg.TopicPartition.Offset,
				}).Error("Failed to process message")
				continue
			}

			// Commit offset after successful processing
			if _, err := c.consumer.CommitMessage(msg); err != nil {
				c.logger.WithError(err).Warn("Failed to commit offset")
			}
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg *kafka.Message) error {
	topic := *msg.TopicPartition.Topic

	c.logger.WithFields(logrus.Fields{
		"topic":     topic,
		"partition": msg.TopicPartition.Partition,
		"offset":    msg.TopicPartition.Offset,
		"key":       string(msg.Key),
	}).Debug("Processing message")

	// Parse the clinical event
	var event models.ClinicalEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Determine event type from topic
	event.Type = c.topicToEventType(topic)

	// Process the event
	return c.processEvent(ctx, &event)
}

// processEvent processes a clinical event
func (c *Consumer) processEvent(ctx context.Context, event *models.ClinicalEvent) error {
	c.logger.WithFields(logrus.Fields{
		"event_type": event.Type,
		"patient_id": event.PatientID,
		"event_id":   event.ID,
	}).Debug("Processing clinical event")

	// Get patient clinical data
	patientData, err := c.getPatientClinicalData(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to get patient clinical data: %w", err)
	}

	// Evaluate against all registries
	results, err := c.engine.EvaluateAll(patientData)
	if err != nil {
		return fmt.Errorf("failed to evaluate criteria: %w", err)
	}

	// Process evaluation results
	for _, result := range results {
		if result.Eligible {
			if err := c.handleEligiblePatient(ctx, event, &result); err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"patient_id": result.PatientID,
					"registry":   result.RegistryCode,
				}).Error("Failed to handle eligible patient")
			}
		}
	}

	return nil
}

// getPatientClinicalData gets patient clinical data for evaluation
func (c *Consumer) getPatientClinicalData(ctx context.Context, event *models.ClinicalEvent) (*models.PatientClinicalData, error) {
	// Start with empty data
	patientData := &models.PatientClinicalData{
		PatientID:  event.PatientID,
		Diagnoses:  make([]models.Diagnosis, 0),
		LabResults: make([]models.LabResult, 0),
		Medications: make([]models.Medication, 0),
		Problems:   make([]models.Problem, 0),
		VitalSigns: make([]models.VitalSign, 0),
		RiskScores: make([]models.RiskScoreData, 0),
	}

	// Add data from the current event
	switch event.Type {
	case models.EventTypeDiagnosisCreated, models.EventTypeDiagnosisUpdated:
		if diag := event.GetDiagnosis(); diag != nil {
			patientData.Diagnoses = append(patientData.Diagnoses, *diag)
		}
	case models.EventTypeLabResultCreated:
		if lab := event.GetLabResult(); lab != nil {
			patientData.LabResults = append(patientData.LabResults, *lab)
		}
	case models.EventTypeMedicationStarted, models.EventTypeMedicationStopped:
		if med := event.GetMedication(); med != nil {
			patientData.Medications = append(patientData.Medications, *med)
		}
	case models.EventTypeProblemAdded:
		if code, ok := event.Data["code"].(string); ok {
			prob := models.Problem{
				Code:       code,
				Status:     "active",
				RecordedAt: event.Timestamp,
			}
			if system, ok := event.Data["code_system"].(string); ok {
				prob.CodeSystem = models.CodeSystem(system)
			}
			patientData.Problems = append(patientData.Problems, prob)
		}
	}

	// Try to enrich with data from KB-2 if available
	if c.kb2Client != nil {
		enrichedData, err := c.kb2Client.GetPatientContext(ctx, event.PatientID)
		if err != nil {
			c.logger.WithError(err).Debug("Failed to get patient context from KB-2, using event data only")
		} else if enrichedData != nil {
			// Merge enriched data
			patientData.Demographics = enrichedData.Demographics
			patientData.Diagnoses = append(patientData.Diagnoses, enrichedData.Diagnoses...)
			patientData.LabResults = append(patientData.LabResults, enrichedData.LabResults...)
			patientData.Medications = append(patientData.Medications, enrichedData.Medications...)
			patientData.Problems = append(patientData.Problems, enrichedData.Problems...)
			patientData.VitalSigns = enrichedData.VitalSigns
			patientData.RiskScores = enrichedData.RiskScores
		}
	}

	return patientData, nil
}

// handleEligiblePatient handles a patient who is eligible for a registry
func (c *Consumer) handleEligiblePatient(ctx context.Context, event *models.ClinicalEvent, result *models.CriteriaEvaluationResult) error {
	// Check if already enrolled
	existing, err := c.repo.GetEnrollmentByPatientRegistry(result.PatientID, result.RegistryCode)
	if err != nil {
		return fmt.Errorf("failed to check existing enrollment: %w", err)
	}

	if existing != nil {
		// Already enrolled - check if risk tier changed
		if existing.Status.IsActive() && existing.RiskTier != result.SuggestedRiskTier {
			return c.handleRiskTierChange(ctx, existing, result.SuggestedRiskTier)
		}
		// Re-activate if previously disenrolled
		if existing.Status == models.EnrollmentStatusDisenrolled {
			return c.reactivateEnrollment(ctx, existing, event, result)
		}
		return nil
	}

	// Create new enrollment
	return c.createEnrollment(ctx, event, result)
}

// createEnrollment creates a new enrollment
func (c *Consumer) createEnrollment(ctx context.Context, event *models.ClinicalEvent, result *models.CriteriaEvaluationResult) error {
	enrollment := &models.RegistryPatient{
		RegistryCode:     result.RegistryCode,
		PatientID:        result.PatientID,
		Status:           models.EnrollmentStatusActive,
		EnrollmentSource: c.eventTypeToEnrollmentSource(event.Type),
		SourceEventID:    event.ID,
		RiskTier:         result.SuggestedRiskTier,
		EnrolledAt:       time.Now().UTC(),
	}

	// Store matched criteria in metadata
	matchedCodes := make([]string, 0, len(result.MatchedCriteria))
	for _, mc := range result.MatchedCriteria {
		matchedCodes = append(matchedCodes, mc.CriterionID)
	}
	enrollment.Metadata = models.JSONMap{
		"matched_criteria": matchedCodes,
		"source_event":     event.Type,
	}

	if err := c.repo.CreateEnrollment(enrollment); err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"patient_id":   result.PatientID,
		"registry":     result.RegistryCode,
		"risk_tier":    result.SuggestedRiskTier,
		"source":       enrollment.EnrollmentSource,
	}).Info("Patient enrolled in registry")

	// Produce enrollment event
	if c.producer != nil {
		enrollmentEvent := models.NewEnrollmentEvent(enrollment)
		if err := c.producer.ProduceEvent(ctx, enrollmentEvent); err != nil {
			c.logger.WithError(err).Warn("Failed to produce enrollment event")
		}
	}

	return nil
}

// handleRiskTierChange handles a change in risk tier
func (c *Consumer) handleRiskTierChange(ctx context.Context, enrollment *models.RegistryPatient, newTier models.RiskTier) error {
	oldTier := enrollment.RiskTier

	if err := c.repo.UpdateEnrollmentRiskTier(enrollment.ID, oldTier, newTier, "system"); err != nil {
		return fmt.Errorf("failed to update risk tier: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"patient_id": enrollment.PatientID,
		"registry":   enrollment.RegistryCode,
		"old_tier":   oldTier,
		"new_tier":   newTier,
	}).Info("Patient risk tier changed")

	// Produce risk change event
	if c.producer != nil {
		enrollment.RiskTier = newTier
		riskEvent := models.NewRiskChangedEvent(enrollment, oldTier, newTier)
		if err := c.producer.ProduceEvent(ctx, riskEvent); err != nil {
			c.logger.WithError(err).Warn("Failed to produce risk change event")
		}
	}

	return nil
}

// reactivateEnrollment reactivates a previously disenrolled patient
func (c *Consumer) reactivateEnrollment(ctx context.Context, enrollment *models.RegistryPatient, event *models.ClinicalEvent, result *models.CriteriaEvaluationResult) error {
	if err := c.repo.UpdateEnrollmentStatus(
		enrollment.ID,
		enrollment.Status,
		models.EnrollmentStatusActive,
		"Auto-reactivated due to new clinical data",
		"system",
	); err != nil {
		return fmt.Errorf("failed to reactivate enrollment: %w", err)
	}

	// Update risk tier if different
	if enrollment.RiskTier != result.SuggestedRiskTier {
		if err := c.repo.UpdateEnrollmentRiskTier(enrollment.ID, enrollment.RiskTier, result.SuggestedRiskTier, "system"); err != nil {
			c.logger.WithError(err).Warn("Failed to update risk tier after reactivation")
		}
	}

	c.logger.WithFields(logrus.Fields{
		"patient_id": enrollment.PatientID,
		"registry":   enrollment.RegistryCode,
	}).Info("Patient enrollment reactivated")

	// Produce enrollment event
	if c.producer != nil {
		enrollment.Status = models.EnrollmentStatusActive
		enrollment.RiskTier = result.SuggestedRiskTier
		enrollmentEvent := models.NewEnrollmentEvent(enrollment)
		if err := c.producer.ProduceEvent(ctx, enrollmentEvent); err != nil {
			c.logger.WithError(err).Warn("Failed to produce enrollment event")
		}
	}

	return nil
}

// topicToEventType converts a Kafka topic to event type
func (c *Consumer) topicToEventType(topic string) models.EventType {
	switch topic {
	case models.KafkaTopics.DiagnosisEvents:
		return models.EventTypeDiagnosisCreated
	case models.KafkaTopics.LabResultEvents:
		return models.EventTypeLabResultCreated
	case models.KafkaTopics.MedicationEvents:
		return models.EventTypeMedicationStarted
	case models.KafkaTopics.ProblemEvents:
		return models.EventTypeProblemAdded
	default:
		return ""
	}
}

// eventTypeToEnrollmentSource converts event type to enrollment source
func (c *Consumer) eventTypeToEnrollmentSource(eventType models.EventType) models.EnrollmentSource {
	switch eventType {
	case models.EventTypeDiagnosisCreated, models.EventTypeDiagnosisUpdated:
		return models.EnrollmentSourceDiagnosis
	case models.EventTypeLabResultCreated:
		return models.EnrollmentSourceLabResult
	case models.EventTypeMedicationStarted, models.EventTypeMedicationStopped:
		return models.EnrollmentSourceMedication
	case models.EventTypeProblemAdded, models.EventTypeProblemResolved:
		return models.EnrollmentSourceProblemList
	default:
		return models.EnrollmentSourceManual
	}
}

// IsRunning returns whether the consumer is running
func (c *Consumer) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

// ProcessManualEvent processes an event submitted via API
func (c *Consumer) ProcessManualEvent(ctx context.Context, req *models.ProcessEventRequest) (*models.ProcessEventResponse, error) {
	event := &models.ClinicalEvent{
		ID:          uuid.New().String(),
		Type:        req.EventType,
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		Timestamp:   time.Now().UTC(),
		Data:        req.Data,
	}

	response := &models.ProcessEventResponse{
		EventID:            event.ID,
		ProcessedAt:        time.Now().UTC(),
		EnrollmentsCreated: make([]uuid.UUID, 0),
		RiskChanges:        make([]models.RiskChangeResult, 0),
	}

	// Get patient clinical data
	patientData, err := c.getPatientClinicalData(ctx, event)
	if err != nil {
		response.Error = err.Error()
		return response, nil
	}

	// Evaluate against all registries
	results, err := c.engine.EvaluateAll(patientData)
	if err != nil {
		response.Error = err.Error()
		return response, nil
	}

	response.EvaluationResults = results

	// Process evaluation results
	for _, result := range results {
		if result.Eligible {
			existing, _ := c.repo.GetEnrollmentByPatientRegistry(result.PatientID, result.RegistryCode)

			if existing == nil {
				if err := c.createEnrollment(ctx, event, &result); err == nil {
					// Get the created enrollment ID
					newEnrollment, _ := c.repo.GetEnrollmentByPatientRegistry(result.PatientID, result.RegistryCode)
					if newEnrollment != nil {
						response.EnrollmentsCreated = append(response.EnrollmentsCreated, newEnrollment.ID)
					}
				}
			} else if existing.Status.IsActive() && existing.RiskTier != result.SuggestedRiskTier {
				if err := c.handleRiskTierChange(ctx, existing, result.SuggestedRiskTier); err == nil {
					response.RiskChanges = append(response.RiskChanges, models.RiskChangeResult{
						RegistryCode: result.RegistryCode,
						OldTier:      existing.RiskTier,
						NewTier:      result.SuggestedRiskTier,
					})
				}
			}
		}
	}

	response.Success = true
	return response, nil
}
