package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/google/uuid"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/infrastructure/google_fhir"
)

// FHIRMedicationService provides FHIR-compliant medication operations using Google Healthcare API
type FHIRMedicationService struct {
	logger     *zap.Logger
	fhirClient *google_fhir.GoogleFHIRClient
}

// NewFHIRMedicationService creates a new FHIR medication service with Google FHIR store
func NewFHIRMedicationService(logger *zap.Logger, fhirClient *google_fhir.GoogleFHIRClient) *FHIRMedicationService {
	return &FHIRMedicationService{
		logger:     logger,
		fhirClient: fhirClient,
	}
}

// Service request/response types

// GetMedicationsRequest represents a request to get medications
type GetMedicationsRequest struct {
	PatientID *string
	Status    *string
	Limit     *int
	Offset    *int
}

// GetMedicationRequestsRequest represents a request to get medication requests
type GetMedicationRequestsRequest struct {
	PatientID *string
	Status    *string
	Limit     *int
	Offset    *int
}

// CreateMedicationRequestRequest represents a request to create a medication request
type CreateMedicationRequestRequest struct {
	Status                    string
	Intent                    string
	SubjectID                 string
	RequesterID               *string
	EncounterID               *string
	Priority                  *string
	Note                      *string
	MedicationCodeableConcept *entities.FHIRCodeableConcept
	MedicationReference       *entities.FHIRReference
	ReasonCode                []*entities.FHIRCodeableConcept
	DosageInstructions        []*entities.FHIRDosage
}

// UpdateMedicationRequestRequest represents a request to update a medication request
type UpdateMedicationRequestRequest struct {
	Status             *string
	Priority           *string
	Note               *string
	ReasonCode         []*entities.FHIRCodeableConcept
	DosageInstructions []*entities.FHIRDosage
}

// Medication operations

// GetMedicationByID retrieves a medication by ID from Google FHIR store
func (s *FHIRMedicationService) GetMedicationByID(ctx context.Context, id string) (*entities.FHIRMedication, error) {
	s.logger.Info("Getting medication by ID from Google FHIR store", zap.String("id", id))

	// Get from Google FHIR store
	resource, err := s.fhirClient.GetResource(ctx, "Medication", id)
	if err != nil {
		s.logger.Error("Failed to get medication from FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to get medication: %w", err)
	}

	if resource == nil {
		return nil, fmt.Errorf("medication not found")
	}

	// Convert map to FHIRMedication entity
	medication, err := s.mapToMedication(resource)
	if err != nil {
		s.logger.Error("Failed to convert FHIR resource to medication entity", zap.Error(err))
		return nil, fmt.Errorf("failed to convert resource: %w", err)
	}

	return medication, nil
}

// GetMedications retrieves medications based on filters from Google FHIR store
func (s *FHIRMedicationService) GetMedications(ctx context.Context, req *GetMedicationsRequest) ([]*entities.FHIRMedication, error) {
	s.logger.Info("Getting medications from Google FHIR store",
		zap.Stringp("patientId", req.PatientID),
		zap.Stringp("status", req.Status),
		zap.Intp("limit", req.Limit),
		zap.Intp("offset", req.Offset),
	)

	// Build search parameters
	searchParams := make(map[string]string)
	if req.Status != nil {
		searchParams["status"] = *req.Status
	}
	if req.Limit != nil {
		searchParams["_count"] = fmt.Sprintf("%d", *req.Limit)
	}
	if req.Offset != nil {
		searchParams["_offset"] = fmt.Sprintf("%d", *req.Offset)
	}

	// Search in Google FHIR store
	resources, err := s.fhirClient.SearchResources(ctx, "Medication", searchParams)
	if err != nil {
		s.logger.Error("Failed to search medications in FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to search medications: %w", err)
	}

	// Convert to entities
	medications := make([]*entities.FHIRMedication, 0, len(resources))
	for _, resource := range resources {
		medication, err := s.mapToMedication(resource)
		if err != nil {
			s.logger.Warn("Failed to convert FHIR resource to medication entity", zap.Error(err))
			continue
		}
		medications = append(medications, medication)
	}

	return medications, nil
}

// MedicationRequest operations

// GetMedicationRequestByID retrieves a medication request by ID from Google FHIR store
func (s *FHIRMedicationService) GetMedicationRequestByID(ctx context.Context, id string) (*entities.FHIRMedicationRequest, error) {
	s.logger.Info("Getting medication request by ID from Google FHIR store", zap.String("id", id))

	// Get from Google FHIR store
	resource, err := s.fhirClient.GetResource(ctx, "MedicationRequest", id)
	if err != nil {
		s.logger.Error("Failed to get medication request from FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to get medication request: %w", err)
	}

	if resource == nil {
		return nil, fmt.Errorf("medication request not found")
	}

	// Convert map to FHIRMedicationRequest entity
	medicationRequest, err := s.mapToMedicationRequest(resource)
	if err != nil {
		s.logger.Error("Failed to convert FHIR resource to medication request entity", zap.Error(err))
		return nil, fmt.Errorf("failed to convert resource: %w", err)
	}

	return medicationRequest, nil
}

// GetMedicationRequests retrieves medication requests based on filters from Google FHIR store
func (s *FHIRMedicationService) GetMedicationRequests(ctx context.Context, req *GetMedicationRequestsRequest) ([]*entities.FHIRMedicationRequest, error) {
	s.logger.Info("Getting medication requests from Google FHIR store",
		zap.Stringp("patientId", req.PatientID),
		zap.Stringp("status", req.Status),
		zap.Intp("limit", req.Limit),
		zap.Intp("offset", req.Offset),
	)

	// Build search parameters
	searchParams := make(map[string]string)
	if req.PatientID != nil {
		searchParams["subject"] = fmt.Sprintf("Patient/%s", *req.PatientID)
	}
	if req.Status != nil {
		searchParams["status"] = *req.Status
	}
	if req.Limit != nil {
		searchParams["_count"] = fmt.Sprintf("%d", *req.Limit)
	}
	if req.Offset != nil {
		searchParams["_offset"] = fmt.Sprintf("%d", *req.Offset)
	}

	// Search in Google FHIR store
	resources, err := s.fhirClient.SearchResources(ctx, "MedicationRequest", searchParams)
	if err != nil {
		s.logger.Error("Failed to search medication requests in FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to search medication requests: %w", err)
	}

	// Convert to entities
	medicationRequests := make([]*entities.FHIRMedicationRequest, 0, len(resources))
	for _, resource := range resources {
		medicationRequest, err := s.mapToMedicationRequest(resource)
		if err != nil {
			s.logger.Warn("Failed to convert FHIR resource to medication request entity", zap.Error(err))
			continue
		}
		medicationRequests = append(medicationRequests, medicationRequest)
	}

	return medicationRequests, nil
}

// CreateMedicationRequest creates a new medication request in Google FHIR store
func (s *FHIRMedicationService) CreateMedicationRequest(ctx context.Context, req *CreateMedicationRequestRequest) (*entities.FHIRMedicationRequest, error) {
	s.logger.Info("Creating medication request in Google FHIR store",
		zap.String("status", req.Status),
		zap.String("intent", req.Intent),
		zap.String("subjectId", req.SubjectID),
	)

	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Generate new ID
	id := uuid.New().String()
	now := time.Now()

	// Create the medication request resource
	resource := map[string]interface{}{
		"resourceType": "MedicationRequest",
		"id":           id,
		"status":       req.Status,
		"intent":       req.Intent,
		"subject": map[string]interface{}{
			"reference": fmt.Sprintf("Patient/%s", req.SubjectID),
		},
		"authoredOn": now.Format(time.RFC3339),
		"meta": map[string]interface{}{
			"lastUpdated": now.Format(time.RFC3339),
		},
	}

	// Add medication (either CodeableConcept or Reference)
	if req.MedicationCodeableConcept != nil {
		resource["medicationCodeableConcept"] = s.codeableConceptToMap(req.MedicationCodeableConcept)
	} else if req.MedicationReference != nil {
		resource["medicationReference"] = s.referenceToMap(req.MedicationReference)
	}

	// Add optional fields
	if req.RequesterID != nil {
		resource["requester"] = map[string]interface{}{
			"reference": fmt.Sprintf("Practitioner/%s", *req.RequesterID),
		}
	}

	if req.EncounterID != nil {
		resource["encounter"] = map[string]interface{}{
			"reference": fmt.Sprintf("Encounter/%s", *req.EncounterID),
		}
	}

	if req.Priority != nil {
		resource["priority"] = *req.Priority
	}

	if req.Note != nil {
		resource["note"] = []map[string]interface{}{
			{
				"text":         *req.Note,
				"authorString": "System",
				"time":         now.Format(time.RFC3339),
			},
		}
	}

	if len(req.ReasonCode) > 0 {
		reasonCodes := make([]map[string]interface{}, len(req.ReasonCode))
		for i, rc := range req.ReasonCode {
			reasonCodes[i] = s.codeableConceptToMap(rc)
		}
		resource["reasonCode"] = reasonCodes
	}

	if len(req.DosageInstructions) > 0 {
		dosages := make([]map[string]interface{}, len(req.DosageInstructions))
		for i, di := range req.DosageInstructions {
			dosages[i] = s.dosageToMap(di)
		}
		resource["dosageInstruction"] = dosages
	}

	// Create in Google FHIR store
	createdResource, err := s.fhirClient.CreateResource(ctx, "MedicationRequest", resource)
	if err != nil {
		s.logger.Error("Failed to create medication request in FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to create medication request: %w", err)
	}

	// Convert back to entity
	medicationRequest, err := s.mapToMedicationRequest(createdResource)
	if err != nil {
		s.logger.Error("Failed to convert created FHIR resource to entity", zap.Error(err))
		return nil, fmt.Errorf("failed to convert created resource: %w", err)
	}

	s.logger.Info("Created medication request in Google FHIR store", zap.String("id", id))
	return medicationRequest, nil
}

// UpdateMedicationRequest updates an existing medication request in Google FHIR store
func (s *FHIRMedicationService) UpdateMedicationRequest(ctx context.Context, id string, req *UpdateMedicationRequestRequest) (*entities.FHIRMedicationRequest, error) {
	s.logger.Info("Updating medication request in Google FHIR store", zap.String("id", id))

	// Get existing medication request
	existingResource, err := s.fhirClient.GetResource(ctx, "MedicationRequest", id)
	if err != nil {
		return nil, fmt.Errorf("failed to get medication request: %w", err)
	}

	if existingResource == nil {
		return nil, fmt.Errorf("medication request not found")
	}

	// Update fields
	now := time.Now()
	if meta, ok := existingResource["meta"].(map[string]interface{}); ok {
		meta["lastUpdated"] = now.Format(time.RFC3339)
	} else {
		existingResource["meta"] = map[string]interface{}{
			"lastUpdated": now.Format(time.RFC3339),
		}
	}

	if req.Status != nil {
		existingResource["status"] = *req.Status
	}

	if req.Priority != nil {
		existingResource["priority"] = *req.Priority
	}

	if req.ReasonCode != nil {
		reasonCodes := make([]map[string]interface{}, len(req.ReasonCode))
		for i, rc := range req.ReasonCode {
			reasonCodes[i] = s.codeableConceptToMap(rc)
		}
		existingResource["reasonCode"] = reasonCodes
	}

	if req.DosageInstructions != nil {
		dosages := make([]map[string]interface{}, len(req.DosageInstructions))
		for i, di := range req.DosageInstructions {
			dosages[i] = s.dosageToMap(di)
		}
		existingResource["dosageInstruction"] = dosages
	}

	if req.Note != nil {
		noteEntry := map[string]interface{}{
			"text":         *req.Note,
			"authorString": "System",
			"time":         now.Format(time.RFC3339),
		}

		if existingNotes, ok := existingResource["note"].([]interface{}); ok {
			existingResource["note"] = append(existingNotes, noteEntry)
		} else {
			existingResource["note"] = []map[string]interface{}{noteEntry}
		}
	}

	// Update in Google FHIR store
	updatedResource, err := s.fhirClient.UpdateResource(ctx, "MedicationRequest", id, existingResource)
	if err != nil {
		s.logger.Error("Failed to update medication request in FHIR store", zap.Error(err))
		return nil, fmt.Errorf("failed to update medication request: %w", err)
	}

	// Convert back to entity
	medicationRequest, err := s.mapToMedicationRequest(updatedResource)
	if err != nil {
		s.logger.Error("Failed to convert updated FHIR resource to entity", zap.Error(err))
		return nil, fmt.Errorf("failed to convert updated resource: %w", err)
	}

	s.logger.Info("Updated medication request in Google FHIR store", zap.String("id", id))
	return medicationRequest, nil
}

// DeleteMedicationRequest deletes a medication request from Google FHIR store
func (s *FHIRMedicationService) DeleteMedicationRequest(ctx context.Context, id string) error {
	s.logger.Info("Deleting medication request from Google FHIR store", zap.String("id", id))

	err := s.fhirClient.DeleteResource(ctx, "MedicationRequest", id)
	if err != nil {
		s.logger.Error("Failed to delete medication request from FHIR store", zap.Error(err))
		return fmt.Errorf("failed to delete medication request: %w", err)
	}

	s.logger.Info("Deleted medication request from Google FHIR store", zap.String("id", id))
	return nil
}

// Helper methods

func (s *FHIRMedicationService) validateCreateRequest(req *CreateMedicationRequestRequest) error {
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}

	if req.Intent == "" {
		return fmt.Errorf("intent is required")
	}

	if req.SubjectID == "" {
		return fmt.Errorf("subjectId is required")
	}

	if req.MedicationCodeableConcept == nil && req.MedicationReference == nil {
		return fmt.Errorf("either medicationCodeableConcept or medicationReference is required")
	}

	return nil
}

// Conversion methods

func (s *FHIRMedicationService) mapToMedication(resource map[string]interface{}) (*entities.FHIRMedication, error) {
	medication := &entities.FHIRMedication{
		ResourceType: "Medication",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if id, ok := resource["id"].(string); ok {
		medication.ID = id
	}

	if status, ok := resource["status"].(string); ok {
		medication.Status = status
	}

	if code, ok := resource["code"].(map[string]interface{}); ok {
		medication.Code = s.mapToCodeableConcept(code)
	}

	if form, ok := resource["form"].(map[string]interface{}); ok {
		medication.Form = s.mapToCodeableConcept(form)
	}

	if manufacturer, ok := resource["manufacturer"].(map[string]interface{}); ok {
		medication.Manufacturer = s.mapToReference(manufacturer)
	}

	// Parse timestamps from meta
	if meta, ok := resource["meta"].(map[string]interface{}); ok {
		if lastUpdated, ok := meta["lastUpdated"].(string); ok {
			if t, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
				medication.UpdatedAt = t
			}
		}
	}

	return medication, nil
}

func (s *FHIRMedicationService) mapToMedicationRequest(resource map[string]interface{}) (*entities.FHIRMedicationRequest, error) {
	medicationRequest := &entities.FHIRMedicationRequest{
		ResourceType: "MedicationRequest",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if id, ok := resource["id"].(string); ok {
		medicationRequest.ID = id
	}

	if status, ok := resource["status"].(string); ok {
		medicationRequest.Status = status
	}

	if intent, ok := resource["intent"].(string); ok {
		medicationRequest.Intent = intent
	}

	if priority, ok := resource["priority"].(string); ok {
		medicationRequest.Priority = priority
	}

	if subject, ok := resource["subject"].(map[string]interface{}); ok {
		medicationRequest.Subject = s.mapToReference(subject)
	}

	if requester, ok := resource["requester"].(map[string]interface{}); ok {
		medicationRequest.Requester = s.mapToReference(requester)
	}

	if encounter, ok := resource["encounter"].(map[string]interface{}); ok {
		medicationRequest.Encounter = s.mapToReference(encounter)
	}

	if medCodeableConcept, ok := resource["medicationCodeableConcept"].(map[string]interface{}); ok {
		medicationRequest.MedicationCodeableConcept = s.mapToCodeableConcept(medCodeableConcept)
	}

	if medReference, ok := resource["medicationReference"].(map[string]interface{}); ok {
		medicationRequest.MedicationReference = s.mapToReference(medReference)
	}

	if authoredOn, ok := resource["authoredOn"].(string); ok {
		if t, err := time.Parse(time.RFC3339, authoredOn); err == nil {
			medicationRequest.AuthoredOn = &t
		}
	}

	// Parse timestamps from meta
	if meta, ok := resource["meta"].(map[string]interface{}); ok {
		if lastUpdated, ok := meta["lastUpdated"].(string); ok {
			if t, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
				medicationRequest.UpdatedAt = t
			}
		}
	}

	return medicationRequest, nil
}

func (s *FHIRMedicationService) mapToCodeableConcept(data map[string]interface{}) *entities.FHIRCodeableConcept {
	cc := &entities.FHIRCodeableConcept{}

	if text, ok := data["text"].(string); ok {
		cc.Text = text
	}

	if codings, ok := data["coding"].([]interface{}); ok {
		for _, c := range codings {
			if coding, ok := c.(map[string]interface{}); ok {
				fc := &entities.FHIRCoding{}
				if system, ok := coding["system"].(string); ok {
					fc.System = system
				}
				if code, ok := coding["code"].(string); ok {
					fc.Code = code
				}
				if display, ok := coding["display"].(string); ok {
					fc.Display = display
				}
				cc.Coding = append(cc.Coding, fc)
			}
		}
	}

	return cc
}

func (s *FHIRMedicationService) mapToReference(data map[string]interface{}) *entities.FHIRReference {
	ref := &entities.FHIRReference{}

	if reference, ok := data["reference"].(string); ok {
		ref.Reference = reference
	}

	if display, ok := data["display"].(string); ok {
		ref.Display = display
	}

	if refType, ok := data["type"].(string); ok {
		ref.Type = refType
	}

	return ref
}

func (s *FHIRMedicationService) codeableConceptToMap(cc *entities.FHIRCodeableConcept) map[string]interface{} {
	result := make(map[string]interface{})

	if cc.Text != "" {
		result["text"] = cc.Text
	}

	if len(cc.Coding) > 0 {
		codings := make([]map[string]interface{}, len(cc.Coding))
		for i, c := range cc.Coding {
			coding := make(map[string]interface{})
			if c.System != "" {
				coding["system"] = c.System
			}
			if c.Code != "" {
				coding["code"] = c.Code
			}
			if c.Display != "" {
				coding["display"] = c.Display
			}
			codings[i] = coding
		}
		result["coding"] = codings
	}

	return result
}

func (s *FHIRMedicationService) referenceToMap(ref *entities.FHIRReference) map[string]interface{} {
	result := make(map[string]interface{})

	if ref.Reference != "" {
		result["reference"] = ref.Reference
	}

	if ref.Display != "" {
		result["display"] = ref.Display
	}

	if ref.Type != "" {
		result["type"] = ref.Type
	}

	return result
}

func (s *FHIRMedicationService) dosageToMap(dosage *entities.FHIRDosage) map[string]interface{} {
	result := make(map[string]interface{})

	if dosage.Sequence != nil {
		result["sequence"] = *dosage.Sequence
	}

	if dosage.Text != "" {
		result["text"] = dosage.Text
	}

	if dosage.PatientInstruction != "" {
		result["patientInstruction"] = dosage.PatientInstruction
	}

	if dosage.Route != nil {
		result["route"] = s.codeableConceptToMap(dosage.Route)
	}

	if dosage.Timing != nil {
		timing := make(map[string]interface{})
		if dosage.Timing.Repeat != nil {
			repeat := make(map[string]interface{})
			if dosage.Timing.Repeat.Frequency != nil {
				repeat["frequency"] = *dosage.Timing.Repeat.Frequency
			}
			if dosage.Timing.Repeat.Period != nil {
				repeat["period"] = *dosage.Timing.Repeat.Period
			}
			if dosage.Timing.Repeat.PeriodUnit != "" {
				repeat["periodUnit"] = dosage.Timing.Repeat.PeriodUnit
			}
			timing["repeat"] = repeat
		}
		result["timing"] = timing
	}

	if len(dosage.DoseAndRate) > 0 {
		doseAndRates := make([]map[string]interface{}, len(dosage.DoseAndRate))
		for i, dr := range dosage.DoseAndRate {
			doseAndRate := make(map[string]interface{})
			if dr.DoseQuantity != nil {
				quantity := make(map[string]interface{})
				if dr.DoseQuantity.Value != nil {
					quantity["value"] = *dr.DoseQuantity.Value
				}
				if dr.DoseQuantity.Unit != "" {
					quantity["unit"] = dr.DoseQuantity.Unit
				}
				if dr.DoseQuantity.System != "" {
					quantity["system"] = dr.DoseQuantity.System
				}
				if dr.DoseQuantity.Code != "" {
					quantity["code"] = dr.DoseQuantity.Code
				}
				doseAndRate["doseQuantity"] = quantity
			}
			doseAndRates[i] = doseAndRate
		}
		result["doseAndRate"] = doseAndRates
	}

	return result
}

// Helper functions for pointer types
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}