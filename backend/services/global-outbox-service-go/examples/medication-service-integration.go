// Example: Medication Service Integration with Advanced Outbox Patterns
// This demonstrates complex business workflows with the outbox pattern
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	
	outboxsdk "global-outbox-service-go/pkg/outbox-sdk"
)

// MedicationOrder represents a medication order in the system
type MedicationOrder struct {
	ID                string    `json:"id" db:"id"`
	PatientID         string    `json:"patient_id" db:"patient_id"`
	ProviderID        string    `json:"provider_id" db:"provider_id"`
	MedicationName    string    `json:"medication_name" db:"medication_name"`
	Dosage            string    `json:"dosage" db:"dosage"`
	Frequency         string    `json:"frequency" db:"frequency"`
	Duration          string    `json:"duration" db:"duration"`
	Status            string    `json:"status" db:"status"`
	Priority          string    `json:"priority" db:"priority"`
	SpecialInstructions string `json:"special_instructions" db:"special_instructions"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// MedicationService demonstrates advanced outbox patterns for complex workflows
type MedicationService struct {
	outboxClient *outboxsdk.OutboxClient
	logger       *logrus.Logger
}

// NewMedicationService creates a new medication service with outbox integration
func NewMedicationService() (*MedicationService, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Configure the outbox client with medication-specific settings
	config := &outboxsdk.ClientConfig{
		ServiceName:           "medication-service",
		DatabaseURL:           "postgresql://user:pass@localhost:5432/medication_db",
		OutboxServiceGRPCURL:  "localhost:50052",
		DefaultTopic:          "clinical.medications",
		DefaultPriority:       6, // Medications are slightly higher priority
		DefaultMedicalContext: "routine",
		EnableTracing:         true,
		Timeout:               30 * time.Second,
		RetryAttempts:         3,
		CircuitBreakerEnabled: true,
	}

	outboxClient, err := outboxsdk.NewOutboxClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox client: %w", err)
	}

	return &MedicationService{
		outboxClient: outboxClient,
		logger:       logger,
	}, nil
}

// CreateMedicationOrder creates a new medication order with validation workflow
func (ms *MedicationService) CreateMedicationOrder(ctx context.Context, order *MedicationOrder) error {
	ms.logger.Infof("Creating medication order: %s for patient %s", order.MedicationName, order.PatientID)

	// Set initial values
	now := time.Now().UTC()
	order.ID = uuid.New().String()
	order.Status = "pending_validation"
	order.CreatedAt = now
	order.UpdatedAt = now

	// Determine medical context and priority based on medication and patient condition
	medicalContext, priority := ms.determineMedicalPriority(order)

	// Create the main event data
	eventData := map[string]interface{}{
		"order_id":             order.ID,
		"patient_id":           order.PatientID,
		"provider_id":          order.ProviderID,
		"medication_name":      order.MedicationName,
		"dosage":               order.Dosage,
		"frequency":            order.Frequency,
		"duration":             order.Duration,
		"priority":             order.Priority,
		"special_instructions": order.SpecialInstructions,
		"status":               order.Status,
		"created_at":           order.CreatedAt,
		"workflow_step":        "order_created",
		"requires_validation":  true,
		"action":               "created",
		"version":              "2.0",
	}

	options := &outboxsdk.EventOptions{
		Topic:          "clinical.medications.order_created",
		Priority:       priority,
		MedicalContext: medicalContext,
		CorrelationID:  fmt.Sprintf("med-order-%s", order.ID),
		Metadata: map[string]string{
			"source":        "medication-service",
			"order_id":      order.ID,
			"patient_id":    order.PatientID,
			"workflow_step": "order_created",
			"requires_validation": "true",
		},
	}

	// Publish multiple events as part of the workflow
	events := []outboxsdk.EventRequest{
		{
			EventType: "medication_order.created",
			EventData: eventData,
			Options:   options,
		},
		{
			EventType: "workflow.validation_required",
			EventData: map[string]interface{}{
				"order_id":      order.ID,
				"patient_id":    order.PatientID,
				"medication":    order.MedicationName,
				"priority":      medicalContext,
				"validation_type": "drug_interaction_check",
				"timestamp":     now,
			},
			Options: &outboxsdk.EventOptions{
				Topic:          "clinical.workflows.validation_required",
				Priority:       priority,
				MedicalContext: medicalContext,
				CorrelationID:  options.CorrelationID,
				Metadata: map[string]string{
					"source":          "medication-service",
					"validation_type": "drug_interaction_check",
					"order_id":        order.ID,
				},
			},
		},
	}

	return ms.outboxClient.SaveAndPublishBatch(
		ctx,
		events,
		func(ctx context.Context, tx pgx.Tx) error {
			// Insert the medication order
			query := `
				INSERT INTO medication_orders (
					id, patient_id, provider_id, medication_name, dosage, frequency,
					duration, status, priority, special_instructions, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			`
			
			_, err := tx.Exec(ctx, query,
				order.ID, order.PatientID, order.ProviderID, order.MedicationName,
				order.Dosage, order.Frequency, order.Duration, order.Status,
				order.Priority, order.SpecialInstructions, order.CreatedAt, order.UpdatedAt,
			)
			
			if err != nil {
				return fmt.Errorf("failed to insert medication order: %w", err)
			}

			// Create audit log entry
			auditQuery := `
				INSERT INTO medication_order_audit (
					order_id, action, status, changed_by, changed_at, notes
				) VALUES ($1, $2, $3, $4, $5, $6)
			`
			
			_, err = tx.Exec(ctx, auditQuery,
				order.ID, "created", order.Status, order.ProviderID, now,
				fmt.Sprintf("Order created for %s", order.MedicationName),
			)
			
			if err != nil {
				return fmt.Errorf("failed to create audit log: %w", err)
			}

			ms.logger.Infof("Successfully saved medication order %s", order.ID)
			return nil
		},
	)
}

// ProcessValidationResult processes the result of medication validation
func (ms *MedicationService) ProcessValidationResult(ctx context.Context, orderID string, validationResult ValidationResult) error {
	ms.logger.Infof("Processing validation result for order %s: %s", orderID, validationResult.Status)

	// Determine next status and actions
	var newStatus string
	var workflowEvents []outboxsdk.EventRequest

	medicalContext := "routine"
	priority := int32(6)

	if validationResult.HasCriticalIssues {
		newStatus = "validation_failed"
		medicalContext = "urgent"
		priority = 9
	} else if validationResult.HasWarnings {
		newStatus = "needs_approval"
		medicalContext = "urgent"
		priority = 7
	} else {
		newStatus = "validated"
		medicalContext = "routine"
		priority = 6
	}

	// Base event for validation processed
	baseEvent := map[string]interface{}{
		"order_id":           orderID,
		"validation_status":  validationResult.Status,
		"new_order_status":   newStatus,
		"has_warnings":       validationResult.HasWarnings,
		"has_critical_issues": validationResult.HasCriticalIssues,
		"validation_notes":   validationResult.Notes,
		"processed_at":       time.Now().UTC(),
		"workflow_step":      "validation_processed",
		"action":             "validation_processed",
		"version":            "2.0",
	}

	correlationID := fmt.Sprintf("med-validation-%s", orderID)

	// Add main validation processed event
	workflowEvents = append(workflowEvents, outboxsdk.EventRequest{
		EventType: "medication_order.validation_processed",
		EventData: baseEvent,
		Options: &outboxsdk.EventOptions{
			Topic:          "clinical.medications.validation_processed",
			Priority:       priority,
			MedicalContext: medicalContext,
			CorrelationID:  correlationID,
			Metadata: map[string]string{
				"source":        "medication-service",
				"order_id":      orderID,
				"workflow_step": "validation_processed",
			},
		},
	})

	// Add specific workflow events based on result
	switch newStatus {
	case "validation_failed":
		workflowEvents = append(workflowEvents, outboxsdk.EventRequest{
			EventType: "workflow.order_rejected",
			EventData: map[string]interface{}{
				"order_id":     orderID,
				"reason":       "validation_failed",
				"issues":       validationResult.Issues,
				"rejected_at":  time.Now().UTC(),
			},
			Options: &outboxsdk.EventOptions{
				Topic:          "clinical.workflows.order_rejected",
				Priority:       priority,
				MedicalContext: medicalContext,
				CorrelationID:  correlationID,
			},
		})

	case "needs_approval":
		workflowEvents = append(workflowEvents, outboxsdk.EventRequest{
			EventType: "workflow.approval_required",
			EventData: map[string]interface{}{
				"order_id":      orderID,
				"warnings":      validationResult.Warnings,
				"approval_type": "pharmacist_review",
				"priority":      medicalContext,
			},
			Options: &outboxsdk.EventOptions{
				Topic:          "clinical.workflows.approval_required",
				Priority:       priority,
				MedicalContext: medicalContext,
				CorrelationID:  correlationID,
			},
		})

	case "validated":
		workflowEvents = append(workflowEvents, outboxsdk.EventRequest{
			EventType: "workflow.order_approved",
			EventData: map[string]interface{}{
				"order_id":     orderID,
				"approved_at":  time.Now().UTC(),
				"auto_approved": true,
			},
			Options: &outboxsdk.EventOptions{
				Topic:          "clinical.workflows.order_approved",
				Priority:       priority,
				MedicalContext: medicalContext,
				CorrelationID:  correlationID,
			},
		})
	}

	return ms.outboxClient.SaveAndPublishBatch(
		ctx,
		workflowEvents,
		func(ctx context.Context, tx pgx.Tx) error {
			// Update the order status
			updateQuery := `
				UPDATE medication_orders 
				SET status = $2, updated_at = $3 
				WHERE id = $1
			`
			
			result, err := tx.Exec(ctx, updateQuery, orderID, newStatus, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("failed to update order status: %w", err)
			}

			if result.RowsAffected() == 0 {
				return fmt.Errorf("order not found: %s", orderID)
			}

			// Create audit log entry
			auditQuery := `
				INSERT INTO medication_order_audit (
					order_id, action, status, changed_by, changed_at, notes
				) VALUES ($1, $2, $3, $4, $5, $6)
			`
			
			notes := fmt.Sprintf("Validation processed: %s", validationResult.Status)
			if validationResult.HasCriticalIssues {
				notes += fmt.Sprintf(" - Critical issues: %v", validationResult.Issues)
			}
			
			_, err = tx.Exec(ctx, auditQuery,
				orderID, "validation_processed", newStatus, "system", time.Now().UTC(), notes,
			)
			
			if err != nil {
				return fmt.Errorf("failed to create audit log: %w", err)
			}

			ms.logger.Infof("Successfully processed validation for order %s, new status: %s", orderID, newStatus)
			return nil
		},
	)
}

// HandleUrgentMedicationRequest handles urgent/emergency medication requests
func (ms *MedicationService) HandleUrgentMedicationRequest(ctx context.Context, request UrgentMedicationRequest) error {
	ms.logger.Warnf("Processing URGENT medication request for patient %s", request.PatientID)

	// For urgent requests, we may need to bypass normal validation
	order := &MedicationOrder{
		ID:                  uuid.New().String(),
		PatientID:           request.PatientID,
		ProviderID:          request.ProviderID,
		MedicationName:      request.MedicationName,
		Dosage:              request.Dosage,
		Frequency:           request.Frequency,
		Duration:            "as_needed",
		Status:              "urgent_approved", // Skip normal validation
		Priority:            "urgent",
		SpecialInstructions: fmt.Sprintf("URGENT REQUEST: %s", request.Reason),
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}

	eventData := map[string]interface{}{
		"order_id":       order.ID,
		"patient_id":     order.PatientID,
		"provider_id":    order.ProviderID,
		"medication_name": order.MedicationName,
		"dosage":         order.Dosage,
		"request_type":   "urgent",
		"reason":         request.Reason,
		"emergency_override": request.EmergencyOverride,
		"bypass_validation": true,
		"created_at":     order.CreatedAt,
		"action":         "urgent_created",
		"version":        "2.0",
	}

	options := &outboxsdk.EventOptions{
		Topic:          "clinical.medications.urgent_order",
		Priority:       10, // Maximum priority
		MedicalContext: "critical",
		CorrelationID:  fmt.Sprintf("urgent-med-%s", order.ID),
		Metadata: map[string]string{
			"source":            "medication-service",
			"order_id":          order.ID,
			"patient_id":        order.PatientID,
			"request_type":      "urgent",
			"emergency_override": fmt.Sprintf("%t", request.EmergencyOverride),
		},
	}

	// Use SaveAndPublish for single urgent event
	return ms.outboxClient.SaveAndPublish(
		ctx,
		"medication_order.urgent_created",
		eventData,
		options,
		func(ctx context.Context, tx pgx.Tx) error {
			// Insert urgent order directly
			query := `
				INSERT INTO medication_orders (
					id, patient_id, provider_id, medication_name, dosage, frequency,
					duration, status, priority, special_instructions, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			`
			
			_, err := tx.Exec(ctx, query,
				order.ID, order.PatientID, order.ProviderID, order.MedicationName,
				order.Dosage, order.Frequency, order.Duration, order.Status,
				order.Priority, order.SpecialInstructions, order.CreatedAt, order.UpdatedAt,
			)
			
			if err != nil {
				return fmt.Errorf("failed to insert urgent medication order: %w", err)
			}

			// Log urgent action
			auditQuery := `
				INSERT INTO medication_order_audit (
					order_id, action, status, changed_by, changed_at, notes
				) VALUES ($1, $2, $3, $4, $5, $6)
			`
			
			notes := fmt.Sprintf("URGENT ORDER - Reason: %s, Emergency Override: %t", 
				request.Reason, request.EmergencyOverride)
			
			_, err = tx.Exec(ctx, auditQuery,
				order.ID, "urgent_created", order.Status, request.ProviderID, 
				time.Now().UTC(), notes,
			)

			return err
		},
	)
}

// Close closes the medication service
func (ms *MedicationService) Close() error {
	return ms.outboxClient.Close()
}

// Helper methods and types

// ValidationResult represents the result of medication validation
type ValidationResult struct {
	Status             string   `json:"status"`
	HasWarnings        bool     `json:"has_warnings"`
	HasCriticalIssues  bool     `json:"has_critical_issues"`
	Warnings           []string `json:"warnings"`
	Issues             []string `json:"issues"`
	Notes              string   `json:"notes"`
}

// UrgentMedicationRequest represents an urgent medication request
type UrgentMedicationRequest struct {
	PatientID         string `json:"patient_id"`
	ProviderID        string `json:"provider_id"`
	MedicationName    string `json:"medication_name"`
	Dosage            string `json:"dosage"`
	Frequency         string `json:"frequency"`
	Reason            string `json:"reason"`
	EmergencyOverride bool   `json:"emergency_override"`
}

func (ms *MedicationService) determineMedicalPriority(order *MedicationOrder) (string, int32) {
	// Determine medical context based on medication type and priority
	medication := strings.ToLower(order.MedicationName)
	priority := strings.ToLower(order.Priority)

	// Check for critical medications
	criticalMeds := []string{"epinephrine", "insulin", "nitroglycerin", "morphine", "warfarin"}
	for _, med := range criticalMeds {
		if strings.Contains(medication, med) {
			return "critical", 10
		}
	}

	// Check for urgent priority
	if priority == "urgent" || priority == "high" {
		return "urgent", 8
	}

	// Check for routine high-priority medications
	routineHighPriority := []string{"antibiotic", "blood pressure", "heart"}
	for _, med := range routineHighPriority {
		if strings.Contains(medication, med) {
			return "routine", 7
		}
	}

	// Default
	return "routine", 6
}

// Example usage
func main() {
	service, err := NewMedicationService()
	if err != nil {
		log.Fatalf("Failed to create medication service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()

	// Example 1: Create a normal medication order
	order := &MedicationOrder{
		PatientID:      "pat-001",
		ProviderID:     "doc-001",
		MedicationName: "Amoxicillin",
		Dosage:         "500mg",
		Frequency:      "twice_daily",
		Duration:       "7_days",
		Priority:       "routine",
	}

	if err := service.CreateMedicationOrder(ctx, order); err != nil {
		log.Printf("Failed to create medication order: %v", err)
	}

	// Example 2: Handle urgent medication request
	urgentRequest := UrgentMedicationRequest{
		PatientID:         "pat-002",
		ProviderID:        "doc-002",
		MedicationName:    "Epinephrine",
		Dosage:            "0.3mg",
		Frequency:         "once",
		Reason:            "Severe allergic reaction",
		EmergencyOverride: true,
	}

	if err := service.HandleUrgentMedicationRequest(ctx, urgentRequest); err != nil {
		log.Printf("Failed to handle urgent request: %v", err)
	}

	log.Println("Medication service examples completed successfully")
}