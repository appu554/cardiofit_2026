package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"kb-formulary/internal/cache"
	"kb-formulary/internal/models"

	"github.com/google/uuid"
)

// =============================================================================
// EVENT EMITTER SERVICE
// =============================================================================

// EventHandler defines a function that handles events
type EventHandler func(event *models.FormularyEvent) error

// EventEmitter manages cross-service event emission
type EventEmitter struct {
	cache        *cache.RedisManager
	handlers     map[models.EventType][]EventHandler
	mu           sync.RWMutex
	enabled      bool
	asyncEnabled bool

	// Channels for async processing
	eventQueue   chan *models.FormularyEvent
	done         chan struct{}

	// Metrics
	emittedCount   int64
	failedCount    int64
	lastEmittedAt  *time.Time
}

// EventEmitterConfig configures the event emitter
type EventEmitterConfig struct {
	Enabled        bool
	AsyncEnabled   bool
	QueueSize      int
	RedisChannel   string
	RetryAttempts  int
	RetryDelayMs   int
}

// DefaultEventEmitterConfig returns default configuration
func DefaultEventEmitterConfig() EventEmitterConfig {
	return EventEmitterConfig{
		Enabled:       true,
		AsyncEnabled:  true,
		QueueSize:     1000,
		RedisChannel:  "kb6:events",
		RetryAttempts: 3,
		RetryDelayMs:  100,
	}
}

// NewEventEmitter creates a new event emitter service
func NewEventEmitter(cache *cache.RedisManager, config EventEmitterConfig) *EventEmitter {
	emitter := &EventEmitter{
		cache:        cache,
		handlers:     make(map[models.EventType][]EventHandler),
		enabled:      config.Enabled,
		asyncEnabled: config.AsyncEnabled,
		eventQueue:   make(chan *models.FormularyEvent, config.QueueSize),
		done:         make(chan struct{}),
	}

	// Start async processor if enabled
	if config.AsyncEnabled {
		go emitter.processQueue()
	}

	return emitter
}

// =============================================================================
// EVENT EMISSION
// =============================================================================

// Emit emits an event to all configured targets
func (e *EventEmitter) Emit(ctx context.Context, event *models.FormularyEvent) error {
	if !e.enabled {
		return nil
	}

	// Ensure event has required fields
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.SourceService == "" {
		event.SourceService = "KB6_FORMULARY"
	}
	if event.SourceVersion == "" {
		event.SourceVersion = "1.0.0"
	}
	if event.TargetServices == nil || len(event.TargetServices) == 0 {
		event.TargetServices = models.GetDefaultTargetServices(event.EventType)
	}

	// Log the event
	log.Printf("[EVENT] %s: %s (correlation: %s, severity: %s)",
		event.EventType, event.Reason, event.CorrelationID, event.Severity)

	// Publish to Redis for cross-service consumption
	if err := e.publishToRedis(ctx, event); err != nil {
		log.Printf("[EVENT] Failed to publish to Redis: %v", err)
	}

	// Handle locally registered handlers
	if e.asyncEnabled {
		select {
		case e.eventQueue <- event:
			// Queued successfully
		default:
			log.Printf("[EVENT] Queue full, dropping event: %s", event.ID)
			e.failedCount++
		}
	} else {
		e.handleEvent(event)
	}

	e.emittedCount++
	now := time.Now()
	e.lastEmittedAt = &now

	return nil
}

// EmitPA emits a Prior Authorization event
func (e *EventEmitter) EmitPA(ctx context.Context, eventType models.EventType, drug *models.DrugContext, patient *models.PatientContext, reason string, details *models.PAEventDetails, binding *models.PolicyBinding) error {
	event := models.NewFormularyEvent(eventType, uuid.New().String())
	event.DrugContext = drug
	event.PatientContext = patient
	event.Reason = reason
	event.Details = details
	event.PolicyBinding = binding

	// Add recommendations based on event type
	switch eventType {
	case models.EventPARequired:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "SUBMIT_PA",
				Priority:    "high",
				Description: "Prior Authorization is required before dispensing",
				RequiredBy:  "provider",
			},
		}
	case models.EventPADenied:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "CONSIDER_ALTERNATIVE",
				Priority:    "high",
				Description: "Consider therapeutic alternatives or appeal",
				RequiredBy:  "provider",
			},
			{
				Action:      "NOTIFY_PATIENT",
				Priority:    "normal",
				Description: "Inform patient of PA denial and next steps",
				RequiredBy:  "provider",
			},
		}
	}

	return e.Emit(ctx, &event)
}

// EmitST emits a Step Therapy event
func (e *EventEmitter) EmitST(ctx context.Context, eventType models.EventType, drug *models.DrugContext, patient *models.PatientContext, reason string, details *models.STEventDetails, binding *models.PolicyBinding) error {
	event := models.NewFormularyEvent(eventType, uuid.New().String())
	event.DrugContext = drug
	event.PatientContext = patient
	event.Reason = reason
	event.Details = details
	event.PolicyBinding = binding

	switch eventType {
	case models.EventSTNonCompliant:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "COMPLETE_STEP_THERAPY",
				Priority:    "high",
				Description: fmt.Sprintf("Complete step %d before proceeding", details.CurrentStep),
				RequiredBy:  "provider",
			},
			{
				Action:      "REQUEST_OVERRIDE",
				Priority:    "normal",
				Description: "Request step therapy override if clinically appropriate",
				RequiredBy:  "provider",
			},
		}
	}

	return e.Emit(ctx, &event)
}

// EmitQL emits a Quantity Limit event
func (e *EventEmitter) EmitQL(ctx context.Context, eventType models.EventType, drug *models.DrugContext, patient *models.PatientContext, reason string, details *models.QLEventDetails, binding *models.PolicyBinding) error {
	event := models.NewFormularyEvent(eventType, uuid.New().String())
	event.DrugContext = drug
	event.PatientContext = patient
	event.Reason = reason
	event.Details = details
	event.PolicyBinding = binding

	switch eventType {
	case models.EventQLViolation:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "ADJUST_QUANTITY",
				Priority:    "high",
				Description: fmt.Sprintf("Reduce quantity to %d units", *details.SuggestedQty),
				RequiredBy:  "provider",
			},
		}
	case models.EventQLExceeded:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "REQUEST_OVERRIDE",
				Priority:    "urgent",
				Description: "Request quantity limit override",
				RequiredBy:  "provider",
			},
		}
	}

	return e.Emit(ctx, &event)
}

// EmitOverride emits an Override event
func (e *EventEmitter) EmitOverride(ctx context.Context, eventType models.EventType, drug *models.DrugContext, patient *models.PatientContext, provider *models.ProviderContext, reason string, details *models.OverrideEventDetails, binding *models.PolicyBinding) error {
	event := models.NewFormularyEvent(eventType, uuid.New().String())
	event.DrugContext = drug
	event.PatientContext = patient
	event.ProviderContext = provider
	event.Reason = reason
	event.Details = details
	event.PolicyBinding = binding

	return e.Emit(ctx, &event)
}

// EmitGovernance emits a Governance/Policy event
func (e *EventEmitter) EmitGovernance(ctx context.Context, eventType models.EventType, drug *models.DrugContext, patient *models.PatientContext, reason string, details *models.GovernanceEventDetails, binding *models.PolicyBinding) error {
	event := models.NewFormularyEvent(eventType, uuid.New().String())
	event.DrugContext = drug
	event.PatientContext = patient
	event.Reason = reason
	event.Details = details
	event.PolicyBinding = binding
	event.Severity = models.EventSeverityHigh // Governance events are always high severity

	switch eventType {
	case models.EventGovernanceBreach:
		event.Recommendations = []models.ActionRecommendation{
			{
				Action:      "REVIEW_POLICY",
				Priority:    "urgent",
				Description: fmt.Sprintf("Policy violation detected: %s", details.ViolationType),
				RequiredBy:  "compliance_officer",
			},
		}
	}

	return e.Emit(ctx, &event)
}

// =============================================================================
// EVENT HANDLERS
// =============================================================================

// RegisterHandler registers a handler for a specific event type
func (e *EventEmitter) RegisterHandler(eventType models.EventType, handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers[eventType] = append(e.handlers[eventType], handler)
}

// handleEvent executes all handlers for an event
func (e *EventEmitter) handleEvent(event *models.FormularyEvent) {
	e.mu.RLock()
	handlers := e.handlers[event.EventType]
	e.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(event); err != nil {
			log.Printf("[EVENT] Handler error for %s: %v", event.EventType, err)
		}
	}
}

// processQueue processes events from the async queue
func (e *EventEmitter) processQueue() {
	for {
		select {
		case event := <-e.eventQueue:
			e.handleEvent(event)
		case <-e.done:
			return
		}
	}
}

// =============================================================================
// REDIS PUBLISHING
// =============================================================================

// publishToRedis publishes the event to Redis for cross-service consumption
func (e *EventEmitter) publishToRedis(ctx context.Context, event *models.FormularyEvent) error {
	if e.cache == nil {
		return nil
	}

	data, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Store in list for consumers (KB-14, KB-3, etc.)
	key := fmt.Sprintf("kb6:events:%s", event.Category)
	return e.cache.LPush(ctx, key, string(data))
}

// =============================================================================
// LIFECYCLE
// =============================================================================

// Close gracefully shuts down the event emitter
func (e *EventEmitter) Close() {
	close(e.done)
}

// GetMetrics returns event emitter metrics
func (e *EventEmitter) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"emitted_count": e.emittedCount,
		"failed_count":  e.failedCount,
		"last_emitted":  e.lastEmittedAt,
		"enabled":       e.enabled,
		"async_enabled": e.asyncEnabled,
	}
}

// =============================================================================
// CONVENIENCE FUNCTIONS FOR POLICY BINDING
// =============================================================================

// CreatePolicyBindingFromPayer creates a policy binding from payer context
func CreatePolicyBindingFromPayer(payerID, planID, jurisdiction, policyName, policyVersion string) *models.PolicyBinding {
	jur := models.Jurisdiction{
		Type:      models.JurisdictionType(jurisdiction),
		Authority: "Payer",
	}

	if jurisdiction == "" {
		jur.Type = models.JurisdictionUS
	}

	binding := models.NewPolicyBinding(models.PolicyTypePriorAuth, jur, models.BindingLevelPayer)
	binding.PayerProgram = &models.PayerProgram{
		PayerID: payerID,
		PlanID:  planID,
	}
	binding.PolicyReference = models.PolicyReference{
		ID:      fmt.Sprintf("%s:%s", payerID, planID),
		Name:    policyName,
		Version: policyVersion,
	}

	return &binding
}

// CreateDrugContext creates a drug context from common parameters
func CreateDrugContext(rxnormCode, drugName, genericName, drugClass string) *models.DrugContext {
	return &models.DrugContext{
		RxNormCode:  rxnormCode,
		DrugName:    drugName,
		GenericName: genericName,
		DrugClass:   drugClass,
	}
}

// CreatePatientContext creates a patient context from common parameters
func CreatePatientContext(patientID, memberID string, age *int, diagnoses []string) *models.PatientContext {
	return &models.PatientContext{
		PatientID: patientID,
		MemberID:  memberID,
		Age:       age,
		Diagnoses: diagnoses,
	}
}

// =============================================================================
// BATCH EVENT OPERATIONS
// =============================================================================

// EventBatch allows batching multiple events
type EventBatch struct {
	events  []*models.FormularyEvent
	emitter *EventEmitter
}

// NewEventBatch creates a new event batch
func (e *EventEmitter) NewBatch() *EventBatch {
	return &EventBatch{
		events:  make([]*models.FormularyEvent, 0),
		emitter: e,
	}
}

// Add adds an event to the batch
func (b *EventBatch) Add(event *models.FormularyEvent) {
	b.events = append(b.events, event)
}

// Emit emits all events in the batch
func (b *EventBatch) Emit(ctx context.Context) error {
	var lastErr error
	for _, event := range b.events {
		if err := b.emitter.Emit(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// =============================================================================
// EVENT STREAM CONSUMER (for other services)
// =============================================================================

// EventStreamConsumer provides methods for consuming events
type EventStreamConsumer struct {
	cache    *cache.RedisManager
	category models.EventCategory
}

// NewEventStreamConsumer creates a consumer for a specific category
func NewEventStreamConsumer(cache *cache.RedisManager, category models.EventCategory) *EventStreamConsumer {
	return &EventStreamConsumer{
		cache:    cache,
		category: category,
	}
}

// Consume consumes events from the queue
func (c *EventStreamConsumer) Consume(ctx context.Context, count int) ([]*models.FormularyEvent, error) {
	key := fmt.Sprintf("kb6:events:%s", c.category)

	events := make([]*models.FormularyEvent, 0, count)
	for i := 0; i < count; i++ {
		data, err := c.cache.RPop(ctx, key)
		if err != nil || data == "" {
			break
		}

		var event models.FormularyEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			log.Printf("[CONSUMER] Failed to deserialize event: %v", err)
			continue
		}
		events = append(events, &event)
	}

	return events, nil
}
