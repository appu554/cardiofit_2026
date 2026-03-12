package resolvers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/graphql-go/graphql"
	"go.uber.org/zap"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
)

// MedicationResolver handles GraphQL resolvers for medication operations
type MedicationResolver struct {
	fhirMedicationService *services.FHIRMedicationService
	logger                *zap.Logger
}

// NewMedicationResolver creates a new medication resolver
func NewMedicationResolver(
	fhirMedicationService *services.FHIRMedicationService,
	logger *zap.Logger,
) *MedicationResolver {
	return &MedicationResolver{
		fhirMedicationService: fhirMedicationService,
		logger:                logger,
	}
}

// GetMedication resolves the medication query
func (r *MedicationResolver) GetMedication(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	id, ok := params.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	r.logger.Info("Resolving medication", zap.String("id", id))

	medication, err := r.fhirMedicationService.GetMedicationByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get medication", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return r.entityToGraphQL(medication), nil
}

// GetMedications resolves the medications query
func (r *MedicationResolver) GetMedications(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context

	// Extract query parameters
	var patientID *string
	if pid, ok := params.Args["patientId"].(string); ok && pid != "" {
		patientID = &pid
	}

	var status *string
	if s, ok := params.Args["status"].(string); ok && s != "" {
		status = &s
	}

	limit := 50
	if l, ok := params.Args["limit"].(int); ok {
		limit = l
	}

	offset := 0
	if o, ok := params.Args["offset"].(int); ok {
		offset = o
	}

	r.logger.Info("Resolving medications",
		zap.Stringp("patientId", patientID),
		zap.Stringp("status", status),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	medications, err := r.fhirMedicationService.GetMedications(ctx, &services.GetMedicationsRequest{
		PatientID: patientID,
		Status:    status,
		Limit:     &limit,
		Offset:    &offset,
	})
	if err != nil {
		r.logger.Error("Failed to get medications", zap.Error(err))
		return nil, err
	}

	var result []interface{}
	for _, medication := range medications {
		result = append(result, r.entityToGraphQL(medication))
	}

	return result, nil
}

// GetMedicationRequest resolves the medicationRequest query
func (r *MedicationResolver) GetMedicationRequest(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	id, ok := params.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	r.logger.Info("Resolving medication request", zap.String("id", id))

	medicationRequest, err := r.fhirMedicationService.GetMedicationRequestByID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get medication request", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return r.medicationRequestToGraphQL(medicationRequest), nil
}

// GetMedicationRequests resolves the medicationRequests query
func (r *MedicationResolver) GetMedicationRequests(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context

	// Extract query parameters
	var patientID *string
	if pid, ok := params.Args["patientId"].(string); ok && pid != "" {
		patientID = &pid
	}

	var status *string
	if s, ok := params.Args["status"].(string); ok && s != "" {
		status = &s
	}

	limit := 50
	if l, ok := params.Args["limit"].(int); ok {
		limit = l
	}

	offset := 0
	if o, ok := params.Args["offset"].(int); ok {
		offset = o
	}

	r.logger.Info("Resolving medication requests",
		zap.Stringp("patientId", patientID),
		zap.Stringp("status", status),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	medicationRequests, err := r.fhirMedicationService.GetMedicationRequests(ctx, &services.GetMedicationRequestsRequest{
		PatientID: patientID,
		Status:    status,
		Limit:     &limit,
		Offset:    &offset,
	})
	if err != nil {
		r.logger.Error("Failed to get medication requests", zap.Error(err))
		return nil, err
	}

	var result []interface{}
	for _, medicationRequest := range medicationRequests {
		result = append(result, r.medicationRequestToGraphQL(medicationRequest))
	}

	return result, nil
}

// CreateMedicationRequest resolves the createMedicationRequest mutation
func (r *MedicationResolver) CreateMedicationRequest(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	input, ok := params.Args["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input is required")
	}

	r.logger.Info("Creating medication request", zap.Any("input", input))

	// Convert GraphQL input to service request
	createRequest, err := r.inputToCreateRequest(input)
	if err != nil {
		r.logger.Error("Failed to convert input", zap.Error(err))
		return nil, err
	}

	medicationRequest, err := r.fhirMedicationService.CreateMedicationRequest(ctx, createRequest)
	if err != nil {
		r.logger.Error("Failed to create medication request", zap.Error(err))
		return nil, err
	}

	return r.medicationRequestToGraphQL(medicationRequest), nil
}

// UpdateMedicationRequest resolves the updateMedicationRequest mutation
func (r *MedicationResolver) UpdateMedicationRequest(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	id, ok := params.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	input, ok := params.Args["input"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input is required")
	}

	r.logger.Info("Updating medication request", zap.String("id", id), zap.Any("input", input))

	// Convert GraphQL input to service request
	updateRequest, err := r.inputToUpdateRequest(input)
	if err != nil {
		r.logger.Error("Failed to convert input", zap.Error(err))
		return nil, err
	}

	medicationRequest, err := r.fhirMedicationService.UpdateMedicationRequest(ctx, id, updateRequest)
	if err != nil {
		r.logger.Error("Failed to update medication request", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	return r.medicationRequestToGraphQL(medicationRequest), nil
}

// DeleteMedicationRequest resolves the deleteMedicationRequest mutation
func (r *MedicationResolver) DeleteMedicationRequest(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	id, ok := params.Args["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id is required")
	}

	r.logger.Info("Deleting medication request", zap.String("id", id))

	err := r.fhirMedicationService.DeleteMedicationRequest(ctx, id)
	if err != nil {
		r.logger.Error("Failed to delete medication request", zap.String("id", id), zap.Error(err))
		return false, err
	}

	return true, nil
}

// Federation resolvers

// ResolveEntities resolves the _entities query for federation
func (r *MedicationResolver) ResolveEntities(params graphql.ResolveParams) (interface{}, error) {
	ctx := params.Context
	representations, ok := params.Args["representations"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("representations is required")
	}

	r.logger.Info("Resolving entities", zap.Int("count", len(representations)))

	var entities []interface{}

	for _, rep := range representations {
		representation, ok := rep.(map[string]interface{})
		if !ok {
			continue
		}

		typename, ok := representation["__typename"].(string)
		if !ok {
			continue
		}

		id, ok := representation["id"].(string)
		if !ok {
			continue
		}

		switch typename {
		case "Medication":
			medication, err := r.fhirMedicationService.GetMedicationByID(ctx, id)
			if err != nil {
				r.logger.Error("Failed to resolve medication entity", zap.String("id", id), zap.Error(err))
				continue
			}
			entities = append(entities, r.entityToGraphQL(medication))

		case "MedicationRequest":
			medicationRequest, err := r.fhirMedicationService.GetMedicationRequestByID(ctx, id)
			if err != nil {
				r.logger.Error("Failed to resolve medication request entity", zap.String("id", id), zap.Error(err))
				continue
			}
			entities = append(entities, r.medicationRequestToGraphQL(medicationRequest))

		case "Patient":
			// For Patient entities, resolve medication requests for this patient
			patientID := id
			medicationRequests, err := r.fhirMedicationService.GetMedicationRequests(ctx, &services.GetMedicationRequestsRequest{
				PatientID: &patientID,
			})
			if err != nil {
				r.logger.Error("Failed to resolve patient medication requests", zap.String("patientId", patientID), zap.Error(err))
				continue
			}

			var medicationRequestsGraphQL []interface{}
			for _, mr := range medicationRequests {
				medicationRequestsGraphQL = append(medicationRequestsGraphQL, r.medicationRequestToGraphQL(mr))
			}

			entities = append(entities, map[string]interface{}{
				"__typename":         "Patient",
				"id":                id,
				"medicationRequests": medicationRequestsGraphQL,
			})
		}
	}

	return entities, nil
}

// ResolveService resolves the _service query for federation
func (r *MedicationResolver) ResolveService(params graphql.ResolveParams) (interface{}, error) {
	r.logger.Info("Resolving service SDL")

	// Return the SDL for this service - in a real implementation this would come from the schema
	sdl := `
type Medication @key(fields: "id") {
  id: ID!
  code: CodeableConcept
  status: String!
  form: CodeableConcept
}

type MedicationRequest @key(fields: "id") {
  id: ID!
  status: String!
  intent: String!
  subject: Reference!
  medicationCodeableConcept: CodeableConcept
  medicationReference: Reference
  dosageInstruction: [Dosage]
}

type Patient @key(fields: "id") @external {
  id: ID!
  medicationRequests: [MedicationRequest]
}

type CodeableConcept @shareable {
  text: String
  coding: [Coding]
}

type Coding @shareable {
  system: String
  code: String
  display: String
}

type Reference @shareable {
  reference: String
  display: String
  type: String
}

type Dosage @shareable {
  text: String
  patientInstruction: String
  route: CodeableConcept
}

extend type Query {
  medication(id: ID!): Medication
  medications(patientId: String, status: String, limit: Int, offset: Int): [Medication]
  medicationRequest(id: ID!): MedicationRequest
  medicationRequests(patientId: String, status: String, limit: Int, offset: Int): [MedicationRequest]
}
`

	return map[string]interface{}{
		"sdl": sdl,
	}, nil
}

// Helper methods for converting between domain entities and GraphQL objects

// entityToGraphQL converts a FHIR Medication entity to GraphQL format
func (r *MedicationResolver) entityToGraphQL(medication *entities.FHIRMedication) map[string]interface{} {
	result := map[string]interface{}{
		"__typename": "Medication",
		"id":         medication.ID,
		"status":     medication.Status,
	}

	if medication.Code != nil {
		result["code"] = r.codeableConceptToGraphQL(medication.Code)
	}

	if medication.Form != nil {
		result["form"] = r.codeableConceptToGraphQL(medication.Form)
	}

	if len(medication.Identifiers) > 0 {
		var identifiers []interface{}
		for _, identifier := range medication.Identifiers {
			identifiers = append(identifiers, r.identifierToGraphQL(identifier))
		}
		result["identifier"] = identifiers
	}

	if medication.Manufacturer != nil {
		result["manufacturer"] = r.referenceToGraphQL(medication.Manufacturer)
	}

	return result
}

// medicationRequestToGraphQL converts a FHIR MedicationRequest entity to GraphQL format
func (r *MedicationResolver) medicationRequestToGraphQL(medicationRequest *entities.FHIRMedicationRequest) map[string]interface{} {
	result := map[string]interface{}{
		"__typename": "MedicationRequest",
		"id":         medicationRequest.ID,
		"status":     medicationRequest.Status,
		"intent":     medicationRequest.Intent,
	}

	if medicationRequest.Subject != nil {
		result["subject"] = r.referenceToGraphQL(medicationRequest.Subject)
	}

	if medicationRequest.MedicationCodeableConcept != nil {
		result["medicationCodeableConcept"] = r.codeableConceptToGraphQL(medicationRequest.MedicationCodeableConcept)
	}

	if medicationRequest.MedicationReference != nil {
		result["medicationReference"] = r.referenceToGraphQL(medicationRequest.MedicationReference)
	}

	if medicationRequest.Requester != nil {
		result["requester"] = r.referenceToGraphQL(medicationRequest.Requester)
	}

	if medicationRequest.Encounter != nil {
		result["encounter"] = r.referenceToGraphQL(medicationRequest.Encounter)
	}

	if medicationRequest.AuthoredOn != nil {
		result["authoredOn"] = medicationRequest.AuthoredOn.Format("2006-01-02T15:04:05Z")
	}

	if medicationRequest.Priority != "" {
		result["priority"] = medicationRequest.Priority
	}

	if len(medicationRequest.ReasonCode) > 0 {
		var reasonCodes []interface{}
		for _, reasonCode := range medicationRequest.ReasonCode {
			reasonCodes = append(reasonCodes, r.codeableConceptToGraphQL(reasonCode))
		}
		result["reasonCode"] = reasonCodes
	}

	if len(medicationRequest.DosageInstructions) > 0 {
		var dosages []interface{}
		for _, dosage := range medicationRequest.DosageInstructions {
			dosages = append(dosages, r.dosageToGraphQL(dosage))
		}
		result["dosageInstruction"] = dosages
	}

	if len(medicationRequest.Notes) > 0 {
		var notes []interface{}
		for _, note := range medicationRequest.Notes {
			notes = append(notes, r.annotationToGraphQL(note))
		}
		result["note"] = notes
	}

	return result
}

// Helper methods for converting FHIR data types

func (r *MedicationResolver) codeableConceptToGraphQL(cc *entities.FHIRCodeableConcept) map[string]interface{} {
	result := map[string]interface{}{}

	if cc.Text != "" {
		result["text"] = cc.Text
	}

	if len(cc.Coding) > 0 {
		var codings []interface{}
		for _, coding := range cc.Coding {
			codings = append(codings, r.codingToGraphQL(coding))
		}
		result["coding"] = codings
	}

	return result
}

func (r *MedicationResolver) codingToGraphQL(coding *entities.FHIRCoding) map[string]interface{} {
	result := map[string]interface{}{}

	if coding.System != "" {
		result["system"] = coding.System
	}

	if coding.Code != "" {
		result["code"] = coding.Code
	}

	if coding.Display != "" {
		result["display"] = coding.Display
	}

	if coding.Version != "" {
		result["version"] = coding.Version
	}

	if coding.UserSelected != nil {
		result["userSelected"] = *coding.UserSelected
	}

	return result
}

func (r *MedicationResolver) identifierToGraphQL(identifier *entities.FHIRIdentifier) map[string]interface{} {
	result := map[string]interface{}{}

	if identifier.Use != "" {
		result["use"] = identifier.Use
	}

	if identifier.System != "" {
		result["system"] = identifier.System
	}

	if identifier.Value != "" {
		result["value"] = identifier.Value
	}

	if identifier.Type != nil {
		result["type"] = r.codeableConceptToGraphQL(identifier.Type)
	}

	return result
}

func (r *MedicationResolver) referenceToGraphQL(reference *entities.FHIRReference) map[string]interface{} {
	result := map[string]interface{}{}

	if reference.Reference != "" {
		result["reference"] = reference.Reference
	}

	if reference.Display != "" {
		result["display"] = reference.Display
	}

	if reference.Type != "" {
		result["type"] = reference.Type
	}

	return result
}

func (r *MedicationResolver) dosageToGraphQL(dosage *entities.FHIRDosage) map[string]interface{} {
	result := map[string]interface{}{}

	if dosage.Sequence != nil {
		result["sequence"] = *dosage.Sequence
	}

	if dosage.Text != "" {
		result["text"] = dosage.Text
	}

	if dosage.PatientInstruction != "" {
		result["patientInstruction"] = dosage.PatientInstruction
	}

	if dosage.AsNeededBoolean != nil {
		result["asNeededBoolean"] = *dosage.AsNeededBoolean
	}

	if dosage.Route != nil {
		result["route"] = r.codeableConceptToGraphQL(dosage.Route)
	}

	if dosage.Method != nil {
		result["method"] = r.codeableConceptToGraphQL(dosage.Method)
	}

	return result
}

func (r *MedicationResolver) annotationToGraphQL(annotation *entities.FHIRAnnotation) map[string]interface{} {
	result := map[string]interface{}{
		"text": annotation.Text,
	}

	if annotation.AuthorString != "" {
		result["authorString"] = annotation.AuthorString
	}

	if annotation.AuthorReference != nil {
		result["authorReference"] = r.referenceToGraphQL(annotation.AuthorReference)
	}

	if annotation.Time != nil {
		result["time"] = annotation.Time.Format("2006-01-02T15:04:05Z")
	}

	return result
}

// Input conversion methods

func (r *MedicationResolver) inputToCreateRequest(input map[string]interface{}) (*services.CreateMedicationRequestRequest, error) {
	request := &services.CreateMedicationRequestRequest{}

	if status, ok := input["status"].(string); ok {
		request.Status = status
	}

	if intent, ok := input["intent"].(string); ok {
		request.Intent = intent
	}

	if subjectID, ok := input["subjectId"].(string); ok {
		request.SubjectID = subjectID
	}

	if requesterID, ok := input["requesterId"].(string); ok && requesterID != "" {
		request.RequesterID = &requesterID
	}

	if encounterID, ok := input["encounterId"].(string); ok && encounterID != "" {
		request.EncounterID = &encounterID
	}

	if priority, ok := input["priority"].(string); ok && priority != "" {
		request.Priority = &priority
	}

	if note, ok := input["note"].(string); ok && note != "" {
		request.Note = &note
	}

	// Convert medication concept or reference
	if medicationConcept, ok := input["medicationCodeableConcept"].(map[string]interface{}); ok {
		concept, err := r.inputToCodeableConcept(medicationConcept)
		if err != nil {
			return nil, fmt.Errorf("invalid medicationCodeableConcept: %w", err)
		}
		request.MedicationCodeableConcept = concept
	}

	if medicationRef, ok := input["medicationReference"].(map[string]interface{}); ok {
		reference, err := r.inputToReference(medicationRef)
		if err != nil {
			return nil, fmt.Errorf("invalid medicationReference: %w", err)
		}
		request.MedicationReference = reference
	}

	// Convert reason codes
	if reasonCodes, ok := input["reasonCode"].([]interface{}); ok {
		for _, rc := range reasonCodes {
			if reasonCodeMap, ok := rc.(map[string]interface{}); ok {
				reasonCode, err := r.inputToCodeableConcept(reasonCodeMap)
				if err != nil {
					return nil, fmt.Errorf("invalid reasonCode: %w", err)
				}
				request.ReasonCode = append(request.ReasonCode, reasonCode)
			}
		}
	}

	// Convert dosage instructions
	if dosageInstructions, ok := input["dosageInstructions"].([]interface{}); ok {
		for _, di := range dosageInstructions {
			if dosageMap, ok := di.(map[string]interface{}); ok {
				dosage, err := r.inputToDosage(dosageMap)
				if err != nil {
					return nil, fmt.Errorf("invalid dosageInstruction: %w", err)
				}
				request.DosageInstructions = append(request.DosageInstructions, dosage)
			}
		}
	}

	return request, nil
}

func (r *MedicationResolver) inputToUpdateRequest(input map[string]interface{}) (*services.UpdateMedicationRequestRequest, error) {
	request := &services.UpdateMedicationRequestRequest{}

	if status, ok := input["status"].(string); ok && status != "" {
		request.Status = &status
	}

	if priority, ok := input["priority"].(string); ok && priority != "" {
		request.Priority = &priority
	}

	if note, ok := input["note"].(string); ok && note != "" {
		request.Note = &note
	}

	// Convert reason codes
	if reasonCodes, ok := input["reasonCode"].([]interface{}); ok {
		for _, rc := range reasonCodes {
			if reasonCodeMap, ok := rc.(map[string]interface{}); ok {
				reasonCode, err := r.inputToCodeableConcept(reasonCodeMap)
				if err != nil {
					return nil, fmt.Errorf("invalid reasonCode: %w", err)
				}
				request.ReasonCode = append(request.ReasonCode, reasonCode)
			}
		}
	}

	// Convert dosage instructions
	if dosageInstructions, ok := input["dosageInstructions"].([]interface{}); ok {
		for _, di := range dosageInstructions {
			if dosageMap, ok := di.(map[string]interface{}); ok {
				dosage, err := r.inputToDosage(dosageMap)
				if err != nil {
					return nil, fmt.Errorf("invalid dosageInstruction: %w", err)
				}
				request.DosageInstructions = append(request.DosageInstructions, dosage)
			}
		}
	}

	return request, nil
}

// Helper methods for converting input types

func (r *MedicationResolver) inputToCodeableConcept(input map[string]interface{}) (*entities.FHIRCodeableConcept, error) {
	concept := &entities.FHIRCodeableConcept{}

	if text, ok := input["text"].(string); ok {
		concept.Text = text
	}

	if codings, ok := input["coding"].([]interface{}); ok {
		for _, c := range codings {
			if codingMap, ok := c.(map[string]interface{}); ok {
				coding, err := r.inputToCoding(codingMap)
				if err != nil {
					return nil, err
				}
				concept.Coding = append(concept.Coding, coding)
			}
		}
	}

	return concept, nil
}

func (r *MedicationResolver) inputToCoding(input map[string]interface{}) (*entities.FHIRCoding, error) {
	coding := &entities.FHIRCoding{}

	if system, ok := input["system"].(string); ok {
		coding.System = system
	}

	if code, ok := input["code"].(string); ok {
		coding.Code = code
	}

	if display, ok := input["display"].(string); ok {
		coding.Display = display
	}

	if version, ok := input["version"].(string); ok {
		coding.Version = version
	}

	return coding, nil
}

func (r *MedicationResolver) inputToReference(input map[string]interface{}) (*entities.FHIRReference, error) {
	reference := &entities.FHIRReference{}

	if ref, ok := input["reference"].(string); ok {
		reference.Reference = ref
	}

	if display, ok := input["display"].(string); ok {
		reference.Display = display
	}

	if refType, ok := input["type"].(string); ok {
		reference.Type = refType
	}

	return reference, nil
}

func (r *MedicationResolver) inputToDosage(input map[string]interface{}) (*entities.FHIRDosage, error) {
	dosage := &entities.FHIRDosage{}

	if sequence, ok := input["sequence"].(int); ok {
		dosage.Sequence = &sequence
	}

	if text, ok := input["text"].(string); ok {
		dosage.Text = text
	}

	if patientInstruction, ok := input["patientInstruction"].(string); ok {
		dosage.PatientInstruction = patientInstruction
	}

	if asNeeded, ok := input["asNeededBoolean"].(bool); ok {
		dosage.AsNeededBoolean = &asNeeded
	}

	if route, ok := input["route"].(map[string]interface{}); ok {
		routeConcept, err := r.inputToCodeableConcept(route)
		if err != nil {
			return nil, fmt.Errorf("invalid route: %w", err)
		}
		dosage.Route = routeConcept
	}

	if method, ok := input["method"].(map[string]interface{}); ok {
		methodConcept, err := r.inputToCodeableConcept(method)
		if err != nil {
			return nil, fmt.Errorf("invalid method: %w", err)
		}
		dosage.Method = methodConcept
	}

	return dosage, nil
}