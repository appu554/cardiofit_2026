// Example: Patient Service Integration with Transactional Outbox Pattern
// This demonstrates how a microservice implements the outbox pattern using the SDK
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	
	outboxsdk "global-outbox-service-go/pkg/outbox-sdk"
)

// Patient represents a patient entity
type Patient struct {
	ID          string    `json:"id" db:"id"`
	FirstName   string    `json:"first_name" db:"first_name"`
	LastName    string    `json:"last_name" db:"last_name"`
	Email       string    `json:"email" db:"email"`
	DateOfBirth time.Time `json:"date_of_birth" db:"date_of_birth"`
	MedicalID   string    `json:"medical_id" db:"medical_id"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PatientService demonstrates how a microservice integrates with the outbox pattern
type PatientService struct {
	outboxClient *outboxsdk.OutboxClient
	logger       *logrus.Logger
}

// NewPatientService creates a new patient service with outbox integration
func NewPatientService() (*PatientService, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Configure the outbox client
	config := &outboxsdk.ClientConfig{
		ServiceName:           "patient-service",
		DatabaseURL:           "postgresql://user:pass@localhost:5432/patient_db",
		OutboxServiceGRPCURL:  "localhost:50052",
		DefaultTopic:          "clinical.patients",
		DefaultPriority:       5,
		DefaultMedicalContext: "routine",
		EnableTracing:         true,
		Timeout:               30 * time.Second,
		RetryAttempts:         3,
		CircuitBreakerEnabled: true,
	}

	// Initialize outbox client
	outboxClient, err := outboxsdk.NewOutboxClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create outbox client: %w", err)
	}

	return &PatientService{
		outboxClient: outboxClient,
		logger:       logger,
	}, nil
}

// CreatePatient creates a new patient and publishes a patient.created event
func (ps *PatientService) CreatePatient(ctx context.Context, patient *Patient) error {
	ps.logger.Infof("Creating patient: %s %s", patient.FirstName, patient.LastName)

	// Set timestamps
	now := time.Now().UTC()
	patient.CreatedAt = now
	patient.UpdatedAt = now
	patient.Status = "active"

	// Define the event data
	eventData := map[string]interface{}{
		"patient_id":   patient.ID,
		"first_name":   patient.FirstName,
		"last_name":    patient.LastName,
		"email":        patient.Email,
		"medical_id":   patient.MedicalID,
		"created_at":   patient.CreatedAt,
		"action":       "created",
		"version":      "1.0",
	}

	// Event options
	options := &outboxsdk.EventOptions{
		Topic:          "clinical.patients.created",
		Priority:       5,
		MedicalContext: "routine",
		CorrelationID:  fmt.Sprintf("patient-create-%s", patient.ID),
		Metadata: map[string]string{
			"source":      "patient-service",
			"patient_id":  patient.ID,
			"event_version": "1.0",
		},
	}

	// Use SaveAndPublish to ensure transactional consistency
	return ps.outboxClient.SaveAndPublish(
		ctx,
		"patient.created",
		eventData,
		options,
		func(ctx context.Context, tx pgx.Tx) error {
			// This is the business logic that runs in the same transaction
			query := `
				INSERT INTO patients (
					id, first_name, last_name, email, date_of_birth, 
					medical_id, status, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`
			
			_, err := tx.Exec(ctx, query,
				patient.ID,
				patient.FirstName,
				patient.LastName,
				patient.Email,
				patient.DateOfBirth,
				patient.MedicalID,
				patient.Status,
				patient.CreatedAt,
				patient.UpdatedAt,
			)
			
			if err != nil {
				return fmt.Errorf("failed to insert patient: %w", err)
			}

			ps.logger.Infof("Successfully saved patient %s to database", patient.ID)
			return nil
		},
	)
}

// UpdatePatientStatus updates a patient's status and publishes a patient.status_changed event
func (ps *PatientService) UpdatePatientStatus(ctx context.Context, patientID, newStatus string) error {
	ps.logger.Infof("Updating patient status: %s -> %s", patientID, newStatus)

	// Determine medical context based on status change
	medicalContext := "routine"
	priority := int32(5)
	
	if newStatus == "critical" || newStatus == "emergency" {
		medicalContext = "critical"
		priority = 10
	} else if newStatus == "urgent" {
		medicalContext = "urgent"
		priority = 8
	}

	eventData := map[string]interface{}{
		"patient_id":   patientID,
		"new_status":   newStatus,
		"changed_at":   time.Now().UTC(),
		"action":       "status_changed",
		"version":      "1.0",
	}

	options := &outboxsdk.EventOptions{
		Topic:          "clinical.patients.status_changed",
		Priority:       priority,
		MedicalContext: medicalContext,
		CorrelationID:  fmt.Sprintf("patient-status-%s-%d", patientID, time.Now().Unix()),
		Metadata: map[string]string{
			"source":     "patient-service",
			"patient_id": patientID,
			"old_status": "", // Would normally fetch this
			"new_status": newStatus,
		},
	}

	return ps.outboxClient.SaveAndPublish(
		ctx,
		"patient.status_changed",
		eventData,
		options,
		func(ctx context.Context, tx pgx.Tx) error {
			// Update patient status in database
			query := `
				UPDATE patients 
				SET status = $2, updated_at = $3 
				WHERE id = $1
			`
			
			result, err := tx.Exec(ctx, query, patientID, newStatus, time.Now().UTC())
			if err != nil {
				return fmt.Errorf("failed to update patient status: %w", err)
			}

			if result.RowsAffected() == 0 {
				return fmt.Errorf("patient not found: %s", patientID)
			}

			ps.logger.Infof("Successfully updated patient %s status to %s", patientID, newStatus)
			return nil
		},
	)
}

// BatchCreatePatients demonstrates batch event publishing
func (ps *PatientService) BatchCreatePatients(ctx context.Context, patients []*Patient) error {
	ps.logger.Infof("Batch creating %d patients", len(patients))

	// Prepare batch events
	var events []outboxsdk.EventRequest
	now := time.Now().UTC()

	for _, patient := range patients {
		patient.CreatedAt = now
		patient.UpdatedAt = now
		patient.Status = "active"

		eventData := map[string]interface{}{
			"patient_id": patient.ID,
			"first_name": patient.FirstName,
			"last_name":  patient.LastName,
			"email":      patient.Email,
			"medical_id": patient.MedicalID,
			"created_at": patient.CreatedAt,
			"action":     "created",
			"version":    "1.0",
		}

		events = append(events, outboxsdk.EventRequest{
			EventType: "patient.created",
			EventData: eventData,
			Options: &outboxsdk.EventOptions{
				Topic:          "clinical.patients.created",
				Priority:       5,
				MedicalContext: "routine",
				CorrelationID:  fmt.Sprintf("batch-create-%s", patient.ID),
				Metadata: map[string]string{
					"source":       "patient-service",
					"patient_id":   patient.ID,
					"batch_size":   fmt.Sprintf("%d", len(patients)),
					"event_version": "1.0",
				},
			},
		})
	}

	// Use batch save and publish
	return ps.outboxClient.SaveAndPublishBatch(
		ctx,
		events,
		func(ctx context.Context, tx pgx.Tx) error {
			// Insert all patients in a single transaction
			query := `
				INSERT INTO patients (
					id, first_name, last_name, email, date_of_birth,
					medical_id, status, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`

			for _, patient := range patients {
				_, err := tx.Exec(ctx, query,
					patient.ID,
					patient.FirstName,
					patient.LastName,
					patient.Email,
					patient.DateOfBirth,
					patient.MedicalID,
					patient.Status,
					patient.CreatedAt,
					patient.UpdatedAt,
				)
				
				if err != nil {
					return fmt.Errorf("failed to insert patient %s: %w", patient.ID, err)
				}
			}

			ps.logger.Infof("Successfully saved %d patients to database", len(patients))
			return nil
		},
	)
}

// PublishPatientAlert demonstrates immediate event publishing (non-transactional)
func (ps *PatientService) PublishPatientAlert(ctx context.Context, patientID, alertType, message string) error {
	ps.logger.Warnf("Publishing patient alert: %s - %s", patientID, alertType)

	eventData := map[string]interface{}{
		"patient_id":   patientID,
		"alert_type":   alertType,
		"message":      message,
		"severity":     "high",
		"timestamp":    time.Now().UTC(),
		"action":       "alert",
		"version":      "1.0",
	}

	options := &outboxsdk.EventOptions{
		Topic:          "clinical.patients.alerts",
		Priority:       10, // High priority for alerts
		MedicalContext: "critical",
		CorrelationID:  fmt.Sprintf("alert-%s-%d", patientID, time.Now().Unix()),
		Metadata: map[string]string{
			"source":     "patient-service",
			"patient_id": patientID,
			"alert_type": alertType,
			"severity":   "high",
		},
	}

	// Use PublishEvent for immediate publishing (no database transaction needed)
	return ps.outboxClient.PublishEvent(ctx, "patient.alert", eventData, options)
}

// Close closes the patient service and cleans up resources
func (ps *PatientService) Close() error {
	if ps.outboxClient != nil {
		return ps.outboxClient.Close()
	}
	return nil
}

// HealthCheck checks the health of the patient service and outbox client
func (ps *PatientService) HealthCheck(ctx context.Context) error {
	return ps.outboxClient.HealthCheck(ctx)
}

// GetOutboxStats returns outbox statistics for this service
func (ps *PatientService) GetOutboxStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := ps.outboxClient.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to map for easier handling
	result := map[string]interface{}{
		"service_name":        "patient-service",
		"queue_depths":        stats.QueueDepths,
		"total_processed_24h": stats.TotalProcessed_24H,
		"dead_letter_count":   stats.DeadLetterCount,
		"success_rates":       stats.SuccessRates,
		"circuit_breaker":     stats.CircuitBreaker,
	}

	return result, nil
}

// Example usage
func main() {
	// Create patient service
	service, err := NewPatientService()
	if err != nil {
		log.Fatalf("Failed to create patient service: %v", err)
	}
	defer service.Close()

	ctx := context.Background()

	// Example 1: Create a single patient
	patient := &Patient{
		ID:          "pat-001",
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john.doe@example.com",
		DateOfBirth: time.Date(1985, 5, 15, 0, 0, 0, 0, time.UTC),
		MedicalID:   "MRN-123456",
	}

	if err := service.CreatePatient(ctx, patient); err != nil {
		log.Printf("Failed to create patient: %v", err)
	} else {
		log.Printf("Successfully created patient: %s", patient.ID)
	}

	// Example 2: Update patient status
	if err := service.UpdatePatientStatus(ctx, "pat-001", "critical"); err != nil {
		log.Printf("Failed to update patient status: %v", err)
	} else {
		log.Printf("Successfully updated patient status")
	}

	// Example 3: Publish an alert
	if err := service.PublishPatientAlert(ctx, "pat-001", "vital_signs", "Heart rate too high"); err != nil {
		log.Printf("Failed to publish alert: %v", err)
	} else {
		log.Printf("Successfully published alert")
	}

	// Example 4: Check health
	if err := service.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		log.Printf("Health check passed")
	}

	// Example 5: Get stats
	if stats, err := service.GetOutboxStats(ctx); err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		statsJSON, _ := json.MarshalIndent(stats, "", "  ")
		log.Printf("Outbox stats:\n%s", statsJSON)
	}
}