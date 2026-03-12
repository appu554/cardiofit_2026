package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FHIRHandler handles FHIR R4 compliant HTTP requests
type FHIRHandler struct {
	services *services.Services
	logger   *zap.Logger
}

// NewFHIRHandler creates a new FHIR handler
func NewFHIRHandler(services *services.Services, logger *zap.Logger) *FHIRHandler {
	return &FHIRHandler{
		services: services,
		logger:   logger,
	}
}

// FHIR Resource Structures

// FHIRMedicationRequest represents a FHIR MedicationRequest resource
type FHIRMedicationRequest struct {
	ResourceType      string                     `json:"resourceType"`
	ID                string                     `json:"id,omitempty"`
	Meta              *FHIRMeta                  `json:"meta,omitempty"`
	Status            string                     `json:"status"`
	Intent            string                     `json:"intent"`
	Priority          string                     `json:"priority,omitempty"`
	Subject           FHIRReference              `json:"subject"`
	Encounter         *FHIRReference             `json:"encounter,omitempty"`
	AuthoredOn        string                     `json:"authoredOn,omitempty"`
	Requester         *FHIRReference             `json:"requester,omitempty"`
	ReasonCode        []FHIRCodeableConcept      `json:"reasonCode,omitempty"`
	ReasonReference   []FHIRReference            `json:"reasonReference,omitempty"`
	Note              []FHIRAnnotation           `json:"note,omitempty"`
	DosageInstruction []FHIRDosage               `json:"dosageInstruction,omitempty"`
	MedicationCodeableConcept *FHIRCodeableConcept `json:"medicationCodeableConcept,omitempty"`
	MedicationReference       *FHIRReference       `json:"medicationReference,omitempty"`
}

// FHIRMedication represents a FHIR Medication resource
type FHIRMedication struct {
	ResourceType string               `json:"resourceType"`
	ID           string               `json:"id,omitempty"`
	Meta         *FHIRMeta            `json:"meta,omitempty"`
	Code         *FHIRCodeableConcept `json:"code,omitempty"`
	Status       string               `json:"status,omitempty"`
	Manufacturer *FHIRReference       `json:"manufacturer,omitempty"`
	Form         *FHIRCodeableConcept `json:"form,omitempty"`
	Ingredient   []FHIRIngredient     `json:"ingredient,omitempty"`
}

// Common FHIR Data Types

type FHIRMeta struct {
	VersionId   string    `json:"versionId,omitempty"`
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
	Source      string    `json:"source,omitempty"`
	Profile     []string  `json:"profile,omitempty"`
	Security    []FHIRCoding `json:"security,omitempty"`
	Tag         []FHIRCoding `json:"tag,omitempty"`
}

type FHIRReference struct {
	Reference string `json:"reference,omitempty"`
	Type      string `json:"type,omitempty"`
	Display   string `json:"display,omitempty"`
}

type FHIRCodeableConcept struct {
	Coding []FHIRCoding `json:"coding,omitempty"`
	Text   string       `json:"text,omitempty"`
}

type FHIRCoding struct {
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

type FHIRAnnotation struct {
	AuthorReference *FHIRReference `json:"authorReference,omitempty"`
	AuthorString    string         `json:"authorString,omitempty"`
	Time            time.Time      `json:"time,omitempty"`
	Text            string         `json:"text"`
}

type FHIRDosage struct {
	Sequence              int                   `json:"sequence,omitempty"`
	Text                  string                `json:"text,omitempty"`
	AdditionalInstruction []FHIRCodeableConcept `json:"additionalInstruction,omitempty"`
	PatientInstruction    string                `json:"patientInstruction,omitempty"`
	Timing                *FHIRTiming           `json:"timing,omitempty"`
	AsNeededBoolean       *bool                 `json:"asNeededBoolean,omitempty"`
	AsNeededCodeableConcept *FHIRCodeableConcept `json:"asNeededCodeableConcept,omitempty"`
	Site                  *FHIRCodeableConcept  `json:"site,omitempty"`
	Route                 *FHIRCodeableConcept  `json:"route,omitempty"`
	Method                *FHIRCodeableConcept  `json:"method,omitempty"`
	DoseAndRate           []FHIRDoseAndRate     `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod      *FHIRRatio            `json:"maxDosePerPeriod,omitempty"`
	MaxDosePerAdministration *FHIRQuantity      `json:"maxDosePerAdministration,omitempty"`
	MaxDosePerLifetime    *FHIRQuantity         `json:"maxDosePerLifetime,omitempty"`
}

type FHIRTiming struct {
	Event  []time.Time    `json:"event,omitempty"`
	Repeat *FHIRTimingRepeat `json:"repeat,omitempty"`
	Code   *FHIRCodeableConcept `json:"code,omitempty"`
}

type FHIRTimingRepeat struct {
	BoundsQuantity *FHIRQuantity `json:"boundsQuantity,omitempty"`
	BoundsRange    *FHIRRange    `json:"boundsRange,omitempty"`
	BoundsPeriod   *FHIRPeriod   `json:"boundsPeriod,omitempty"`
	Count          int           `json:"count,omitempty"`
	CountMax       int           `json:"countMax,omitempty"`
	Duration       float64       `json:"duration,omitempty"`
	DurationMax    float64       `json:"durationMax,omitempty"`
	DurationUnit   string        `json:"durationUnit,omitempty"`
	Frequency      int           `json:"frequency,omitempty"`
	FrequencyMax   int           `json:"frequencyMax,omitempty"`
	Period         float64       `json:"period,omitempty"`
	PeriodMax      float64       `json:"periodMax,omitempty"`
	PeriodUnit     string        `json:"periodUnit,omitempty"`
	DayOfWeek      []string      `json:"dayOfWeek,omitempty"`
	TimeOfDay      []string      `json:"timeOfDay,omitempty"`
	When           []string      `json:"when,omitempty"`
	Offset         int           `json:"offset,omitempty"`
}

type FHIRDoseAndRate struct {
	Type         *FHIRCodeableConcept `json:"type,omitempty"`
	DoseRange    *FHIRRange           `json:"doseRange,omitempty"`
	DoseQuantity *FHIRQuantity        `json:"doseQuantity,omitempty"`
	RateRatio    *FHIRRatio           `json:"rateRatio,omitempty"`
	RateRange    *FHIRRange           `json:"rateRange,omitempty"`
	RateQuantity *FHIRQuantity        `json:"rateQuantity,omitempty"`
}

type FHIRQuantity struct {
	Value      float64 `json:"value,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	System     string  `json:"system,omitempty"`
	Code       string  `json:"code,omitempty"`
	Comparator string  `json:"comparator,omitempty"`
}

type FHIRRatio struct {
	Numerator   *FHIRQuantity `json:"numerator,omitempty"`
	Denominator *FHIRQuantity `json:"denominator,omitempty"`
}

type FHIRRange struct {
	Low  *FHIRQuantity `json:"low,omitempty"`
	High *FHIRQuantity `json:"high,omitempty"`
}

type FHIRPeriod struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type FHIRIngredient struct {
	ItemCodeableConcept *FHIRCodeableConcept `json:"itemCodeableConcept,omitempty"`
	ItemReference       *FHIRReference       `json:"itemReference,omitempty"`
	IsActive            bool                 `json:"isActive,omitempty"`
	Strength            *FHIRRatio           `json:"strength,omitempty"`
}

type FHIRBundle struct {
	ResourceType string       `json:"resourceType"`
	ID           string       `json:"id,omitempty"`
	Meta         *FHIRMeta    `json:"meta,omitempty"`
	Type         string       `json:"type"`
	Total        int          `json:"total,omitempty"`
	Link         []FHIRLink   `json:"link,omitempty"`
	Entry        []FHIREntry  `json:"entry,omitempty"`
}

type FHIRLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

type FHIREntry struct {
	Resource interface{}      `json:"resource,omitempty"`
	Request  *FHIRRequest     `json:"request,omitempty"`
	Response *FHIRResponse    `json:"response,omitempty"`
	Search   *FHIRSearchMode  `json:"search,omitempty"`
}

type FHIRRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type FHIRResponse struct {
	Status   string    `json:"status"`
	Location string    `json:"location,omitempty"`
	Etag     string    `json:"etag,omitempty"`
	LastModified time.Time `json:"lastModified,omitempty"`
}

type FHIRSearchMode struct {
	Mode  string  `json:"mode"`
	Score float64 `json:"score,omitempty"`
}

type FHIROperationOutcome struct {
	ResourceType string     `json:"resourceType"`
	Issue        []FHIRIssue `json:"issue"`
}

type FHIRIssue struct {
	Severity    string               `json:"severity"`
	Code        string               `json:"code"`
	Details     *FHIRCodeableConcept `json:"details,omitempty"`
	Diagnostics string               `json:"diagnostics,omitempty"`
	Location    []string             `json:"location,omitempty"`
	Expression  []string             `json:"expression,omitempty"`
}

// FHIR MedicationRequest Endpoints

// CreateMedicationRequest creates a new FHIR MedicationRequest
// @Summary Create FHIR MedicationRequest
// @Description Creates a new FHIR MedicationRequest resource
// @Tags fhir
// @Accept json
// @Produce json
// @Param medicationRequest body FHIRMedicationRequest true "FHIR MedicationRequest resource"
// @Success 201 {object} FHIRMedicationRequest
// @Failure 400 {object} FHIROperationOutcome
// @Failure 422 {object} FHIROperationOutcome
// @Security BearerAuth
// @Router /fhir/r4/MedicationRequest [post]
func (h *FHIRHandler) CreateMedicationRequest(c *gin.Context) {
	var fhirRequest FHIRMedicationRequest
	if err := c.ShouldBindJSON(&fhirRequest); err != nil {
		h.sendOperationOutcome(c, http.StatusBadRequest, "structure", "Invalid FHIR resource structure", err.Error())
		return
	}

	// Get authenticated user
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists {
		h.sendOperationOutcome(c, http.StatusUnauthorized, "security", "Authentication required", "")
		return
	}

	// Convert FHIR MedicationRequest to internal MedicationProposal
	proposal, err := h.convertFHIRMedicationRequestToProposal(&fhirRequest, authCtx.UserID)
	if err != nil {
		h.logger.Error("Failed to convert FHIR MedicationRequest", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusUnprocessableEntity, "business-rule", "Conversion failed", err.Error())
		return
	}

	// Create proposal using service
	createReq := services.CreateProposalRequest{
		PatientID:         proposal.PatientID,
		ProtocolID:        proposal.ProtocolID,
		Indication:        proposal.Indication,
		ClinicalContext:   proposal.ClinicalContext,
		MedicationDetails: proposal.MedicationDetails,
		CreatedBy:         authCtx.UserID,
	}

	createdProposal, err := h.services.MedicationService.CreateProposal(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create medication proposal", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to create resource", err.Error())
		return
	}

	// Convert back to FHIR format
	fhirResponse, err := h.convertProposalToFHIRMedicationRequest(createdProposal)
	if err != nil {
		h.logger.Error("Failed to convert proposal to FHIR", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to format response", err.Error())
		return
	}

	// Set FHIR-specific headers
	c.Header("Location", fmt.Sprintf("/fhir/r4/MedicationRequest/%s", fhirResponse.ID))
	c.Header("ETag", fmt.Sprintf("W/\"%s\"", fhirResponse.Meta.VersionId))
	c.Header("Last-Modified", fhirResponse.Meta.LastUpdated.Format(time.RFC1123))

	c.JSON(http.StatusCreated, fhirResponse)
}

// GetMedicationRequest retrieves a FHIR MedicationRequest by ID
// @Summary Get FHIR MedicationRequest
// @Description Retrieves a FHIR MedicationRequest resource by ID
// @Tags fhir
// @Produce json
// @Param id path string true "MedicationRequest ID"
// @Success 200 {object} FHIRMedicationRequest
// @Failure 404 {object} FHIROperationOutcome
// @Security BearerAuth
// @Router /fhir/r4/MedicationRequest/{id} [get]
func (h *FHIRHandler) GetMedicationRequest(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.sendOperationOutcome(c, http.StatusBadRequest, "value", "Invalid resource ID", "ID must be a valid UUID")
		return
	}

	proposal, err := h.services.MedicationService.GetProposal(c.Request.Context(), proposalID)
	if err != nil {
		if isNotFoundError(err) {
			h.sendOperationOutcome(c, http.StatusNotFound, "not-found", "Resource not found", "")
			return
		}
		h.logger.Error("Failed to get proposal", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to retrieve resource", err.Error())
		return
	}

	fhirResponse, err := h.convertProposalToFHIRMedicationRequest(proposal)
	if err != nil {
		h.logger.Error("Failed to convert proposal to FHIR", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to format response", err.Error())
		return
	}

	// Set FHIR-specific headers
	c.Header("ETag", fmt.Sprintf("W/\"%s\"", fhirResponse.Meta.VersionId))
	c.Header("Last-Modified", fhirResponse.Meta.LastUpdated.Format(time.RFC1123))

	c.JSON(http.StatusOK, fhirResponse)
}

// UpdateMedicationRequest updates a FHIR MedicationRequest
func (h *FHIRHandler) UpdateMedicationRequest(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Update not implemented", "")
}

// DeleteMedicationRequest deletes a FHIR MedicationRequest
func (h *FHIRHandler) DeleteMedicationRequest(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		h.sendOperationOutcome(c, http.StatusBadRequest, "value", "Invalid resource ID", "ID must be a valid UUID")
		return
	}

	err = h.services.MedicationService.DeleteProposal(c.Request.Context(), proposalID)
	if err != nil {
		if isNotFoundError(err) {
			h.sendOperationOutcome(c, http.StatusNotFound, "not-found", "Resource not found", "")
			return
		}
		h.logger.Error("Failed to delete proposal", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to delete resource", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// SearchMedicationRequests searches for FHIR MedicationRequest resources
// @Summary Search FHIR MedicationRequest
// @Description Searches for FHIR MedicationRequest resources with various parameters
// @Tags fhir
// @Produce json
// @Param patient query string false "Patient ID"
// @Param status query string false "Request status"
// @Param _count query int false "Number of results per page"
// @Param _offset query int false "Starting offset for pagination"
// @Success 200 {object} FHIRBundle
// @Security BearerAuth
// @Router /fhir/r4/MedicationRequest [get]
func (h *FHIRHandler) SearchMedicationRequests(c *gin.Context) {
	// Parse search parameters
	var patientID *uuid.UUID
	if patientParam := c.Query("patient"); patientParam != "" {
		// Handle patient reference format (e.g., "Patient/123")
		patientRef := strings.TrimPrefix(patientParam, "Patient/")
		if id, err := uuid.Parse(patientRef); err == nil {
			patientID = &id
		}
	}

	var status entities.ProposalStatus
	if statusParam := c.Query("status"); statusParam != "" {
		status = h.mapFHIRStatusToProposalStatus(statusParam)
	}

	// Parse pagination parameters
	count := 20
	if countParam := c.Query("_count"); countParam != "" {
		if c, err := strconv.Atoi(countParam); err == nil && c > 0 && c <= 100 {
			count = c
		}
	}

	offset := 0
	if offsetParam := c.Query("_offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	page := (offset / count) + 1

	// Build search request
	listReq := services.ListProposalsRequest{
		PatientID: patientID,
		Status:    status,
		Page:      page,
		PageSize:  count,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	result, err := h.services.MedicationService.ListProposals(c.Request.Context(), listReq)
	if err != nil {
		h.logger.Error("Failed to search proposals", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Search failed", err.Error())
		return
	}

	// Convert to FHIR Bundle
	bundle, err := h.createFHIRSearchBundle(result, c.Request.URL.String(), count, offset)
	if err != nil {
		h.logger.Error("Failed to create FHIR bundle", zap.Error(err))
		h.sendOperationOutcome(c, http.StatusInternalServerError, "exception", "Failed to format search results", err.Error())
		return
	}

	c.JSON(http.StatusOK, bundle)
}

// Additional FHIR resource handlers (simplified implementations)

func (h *FHIRHandler) GetMedication(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Medication resource not implemented", "")
}

func (h *FHIRHandler) SearchMedications(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Medication search not implemented", "")
}

func (h *FHIRHandler) GetPatient(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Patient resource not implemented", "")
}

func (h *FHIRHandler) GetObservation(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Observation resource not implemented", "")
}

func (h *FHIRHandler) SearchObservations(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Observation search not implemented", "")
}

func (h *FHIRHandler) ProcessBundle(c *gin.Context) {
	h.sendOperationOutcome(c, http.StatusNotImplemented, "not-supported", "Bundle processing not implemented", "")
}

// GetCapabilityStatement returns the FHIR Capability Statement
// @Summary Get FHIR Capability Statement
// @Description Returns the FHIR Capability Statement for this server
// @Tags fhir
// @Produce json
// @Success 200 {object} interface{} \"FHIR CapabilityStatement\"\n// @Router /fhir/r4/metadata [get]
func (h *FHIRHandler) GetCapabilityStatement(c *gin.Context) {\n\tcapabilityStatement := map[string]interface{}{\n\t\t\"resourceType\": \"CapabilityStatement\",\n\t\t\"status\":       \"active\",\n\t\t\"date\":         time.Now().Format(\"2006-01-02\"),\n\t\t\"publisher\":    \"Medication Service V2\",\n\t\t\"kind\":         \"instance\",\n\t\t\"software\": map[string]interface{}{\n\t\t\t\"name\":    \"Medication Service V2\",\n\t\t\t\"version\": \"2.0.0\",\n\t\t},\n\t\t\"fhirVersion\": \"4.0.1\",\n\t\t\"format\":      []string{\"json\"},\n\t\t\"rest\": []map[string]interface{}{\n\t\t\t{\n\t\t\t\t\"mode\": \"server\",\n\t\t\t\t\"resource\": []map[string]interface{}{\n\t\t\t\t\t{\n\t\t\t\t\t\t\"type\": \"MedicationRequest\",\n\t\t\t\t\t\t\"interaction\": []map[string]string{\n\t\t\t\t\t\t\t{\"code\": \"create\"},\n\t\t\t\t\t\t\t{\"code\": \"read\"},\n\t\t\t\t\t\t\t{\"code\": \"delete\"},\n\t\t\t\t\t\t\t{\"code\": \"search-type\"},\n\t\t\t\t\t\t},\n\t\t\t\t\t\t\"searchParam\": []map[string]interface{}{\n\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\"name\": \"patient\",\n\t\t\t\t\t\t\t\t\"type\": \"reference\",\n\t\t\t\t\t\t\t},\n\t\t\t\t\t\t\t{\n\t\t\t\t\t\t\t\t\"name\": \"status\",\n\t\t\t\t\t\t\t\t\"type\": \"token\",\n\t\t\t\t\t\t\t},\n\t\t\t\t\t\t},\n\t\t\t\t\t},\n\t\t\t\t},\n\t\t\t},\n\t\t},\n\t}\n\n\tc.JSON(http.StatusOK, capabilityStatement)\n}\n\n// Helper methods\n\n// sendOperationOutcome sends a FHIR OperationOutcome response\nfunc (h *FHIRHandler) sendOperationOutcome(c *gin.Context, statusCode int, code, diagnostics, details string) {\n\toutcome := FHIROperationOutcome{\n\t\tResourceType: \"OperationOutcome\",\n\t\tIssue: []FHIRIssue{\n\t\t\t{\n\t\t\t\tSeverity:    h.mapStatusCodeToSeverity(statusCode),\n\t\t\t\tCode:        code,\n\t\t\t\tDiagnostics: diagnostics,\n\t\t\t},\n\t\t},\n\t}\n\n\tif details != \"\" {\n\t\toutcome.Issue[0].Details = &FHIRCodeableConcept{\n\t\t\tText: details,\n\t\t}\n\t}\n\n\tc.JSON(statusCode, outcome)\n}\n\n// mapStatusCodeToSeverity maps HTTP status codes to FHIR issue severity\nfunc (h *FHIRHandler) mapStatusCodeToSeverity(statusCode int) string {\n\tswitch {\n\tcase statusCode >= 500:\n\t\treturn \"fatal\"\n\tcase statusCode >= 400:\n\t\treturn \"error\"\n\tcase statusCode >= 300:\n\t\treturn \"warning\"\n\tdefault:\n\t\treturn \"information\"\n\t}\n}\n\n// Conversion methods\n\n// convertFHIRMedicationRequestToProposal converts FHIR MedicationRequest to internal proposal\nfunc (h *FHIRHandler) convertFHIRMedicationRequestToProposal(fhirReq *FHIRMedicationRequest, createdBy string) (*entities.MedicationProposal, error) {\n\t// Parse patient ID from subject reference\n\tpatientRef := strings.TrimPrefix(fhirReq.Subject.Reference, \"Patient/\")\n\tpatientID, err := uuid.Parse(patientRef)\n\tif err != nil {\n\t\treturn nil, fmt.Errorf(\"invalid patient reference: %w\", err)\n\t}\n\n\t// Create basic proposal structure\n\tproposal := &entities.MedicationProposal{\n\t\tID:        uuid.New(),\n\t\tPatientID: patientID,\n\t\tStatus:    h.mapFHIRStatusToProposalStatus(fhirReq.Status),\n\t\tCreatedBy: createdBy,\n\t\tCreatedAt: time.Now(),\n\t\tUpdatedAt: time.Now(),\n\t}\n\n\t// Extract indication from reasonCode or reasonReference\n\tif len(fhirReq.ReasonCode) > 0 && fhirReq.ReasonCode[0].Text != \"\" {\n\t\tproposal.Indication = fhirReq.ReasonCode[0].Text\n\t}\n\n\t// Extract medication details\n\tif fhirReq.MedicationCodeableConcept != nil {\n\t\tproposal.MedicationDetails = &entities.MedicationDetails{\n\t\t\tDrugName: fhirReq.MedicationCodeableConcept.Text,\n\t\t}\n\t\tif len(fhirReq.MedicationCodeableConcept.Coding) > 0 {\n\t\t\tcoding := fhirReq.MedicationCodeableConcept.Coding[0]\n\t\t\tif coding.Display != \"\" {\n\t\t\t\tproposal.MedicationDetails.DrugName = coding.Display\n\t\t\t}\n\t\t}\n\t}\n\n\t// Convert dosage instructions to dosage recommendations\n\tif len(fhirReq.DosageInstruction) > 0 {\n\t\tproposal.DosageRecommendations = h.convertFHIRDosageToRecommendations(fhirReq.DosageInstruction)\n\t}\n\n\t// Basic clinical context (would need more sophisticated mapping)\n\tproposal.ClinicalContext = &entities.ClinicalContext{\n\t\tPatientID: patientID,\n\t}\n\n\treturn proposal, nil\n}\n\n// convertProposalToFHIRMedicationRequest converts internal proposal to FHIR MedicationRequest\nfunc (h *FHIRHandler) convertProposalToFHIRMedicationRequest(proposal *entities.MedicationProposal) (*FHIRMedicationRequest, error) {\n\tfhirReq := &FHIRMedicationRequest{\n\t\tResourceType: \"MedicationRequest\",\n\t\tID:           proposal.ID.String(),\n\t\tMeta: &FHIRMeta{\n\t\t\tVersionId:   \"1\",\n\t\t\tLastUpdated: proposal.UpdatedAt,\n\t\t\tProfile:     []string{\"http://hl7.org/fhir/StructureDefinition/MedicationRequest\"},\n\t\t},\n\t\tStatus: h.mapProposalStatusToFHIR(proposal.Status),\n\t\tIntent: \"order\",\n\t\tSubject: FHIRReference{\n\t\t\tReference: fmt.Sprintf(\"Patient/%s\", proposal.PatientID.String()),\n\t\t},\n\t\tAuthoredOn: proposal.CreatedAt.Format(time.RFC3339),\n\t}\n\n\t// Add requester if available\n\tif proposal.CreatedBy != \"\" {\n\t\tfhirReq.Requester = &FHIRReference{\n\t\t\tReference: fmt.Sprintf(\"Practitioner/%s\", proposal.CreatedBy),\n\t\t}\n\t}\n\n\t// Add reason code\n\tif proposal.Indication != \"\" {\n\t\tfhirReq.ReasonCode = []FHIRCodeableConcept{\n\t\t\t{Text: proposal.Indication},\n\t\t}\n\t}\n\n\t// Add medication\n\tif proposal.MedicationDetails != nil {\n\t\tfhirReq.MedicationCodeableConcept = &FHIRCodeableConcept{\n\t\t\tText: proposal.MedicationDetails.DrugName,\n\t\t}\n\t}\n\n\t// Convert dosage recommendations to FHIR dosage instructions\n\tif len(proposal.DosageRecommendations) > 0 {\n\t\tfhirReq.DosageInstruction = h.convertRecommendationsToFHIRDosage(proposal.DosageRecommendations)\n\t}\n\n\treturn fhirReq, nil\n}\n\n// Status mapping functions\n\nfunc (h *FHIRHandler) mapFHIRStatusToProposalStatus(fhirStatus string) entities.ProposalStatus {\n\tswitch fhirStatus {\n\tcase \"draft\":\n\t\treturn entities.ProposalStatusDraft\n\tcase \"active\":\n\t\treturn entities.ProposalStatusProposed\n\tcase \"completed\":\n\t\treturn entities.ProposalStatusCommitted\n\tcase \"cancelled\":\n\t\treturn entities.ProposalStatusRejected\n\tdefault:\n\t\treturn entities.ProposalStatusDraft\n\t}\n}\n\nfunc (h *FHIRHandler) mapProposalStatusToFHIR(status entities.ProposalStatus) string {\n\tswitch status {\n\tcase entities.ProposalStatusDraft:\n\t\treturn \"draft\"\n\tcase entities.ProposalStatusProposed:\n\t\treturn \"active\"\n\tcase entities.ProposalStatusValidated:\n\t\treturn \"active\"\n\tcase entities.ProposalStatusCommitted:\n\t\treturn \"completed\"\n\tcase entities.ProposalStatusRejected:\n\t\treturn \"cancelled\"\n\tcase entities.ProposalStatusExpired:\n\t\treturn \"entered-in-error\"\n\tdefault:\n\t\treturn \"unknown\"\n\t}\n}\n\n// Helper functions for dosage conversion (simplified implementations)\n\nfunc (h *FHIRHandler) convertFHIRDosageToRecommendations(dosageInstructions []FHIRDosage) []entities.DosageRecommendation {\n\trecommendations := make([]entities.DosageRecommendation, 0, len(dosageInstructions))\n\t\n\tfor i, dosage := range dosageInstructions {\n\t\trec := entities.DosageRecommendation{\n\t\t\tID: uuid.New(),\n\t\t\tRecommendationType: entities.RecommendationStarting,\n\t\t\tConfidenceScore: 1.0,\n\t\t}\n\t\t\n\t\t// Extract dose quantity (simplified)\n\t\tif len(dosage.DoseAndRate) > 0 && dosage.DoseAndRate[0].DoseQuantity != nil {\n\t\t\trec.DoseMg = dosage.DoseAndRate[0].DoseQuantity.Value\n\t\t\tif dosage.DoseAndRate[0].DoseQuantity.Unit != \"\" {\n\t\t\t\t// Unit conversion would go here\n\t\t\t}\n\t\t}\n\t\t\n\t\t// Extract frequency (simplified)\n\t\tif dosage.Timing != nil && dosage.Timing.Repeat != nil {\n\t\t\trec.FrequencyPerDay = dosage.Timing.Repeat.Frequency\n\t\t}\n\t\t\n\t\t// Extract route\n\t\tif dosage.Route != nil {\n\t\t\trec.Route = dosage.Route.Text\n\t\t}\n\t\t\n\t\trecommendations = append(recommendations, rec)\n\t}\n\t\n\treturn recommendations\n}\n\nfunc (h *FHIRHandler) convertRecommendationsToFHIRDosage(recommendations []entities.DosageRecommendation) []FHIRDosage {\n\tdosageInstructions := make([]FHIRDosage, 0, len(recommendations))\n\t\n\tfor i, rec := range recommendations {\n\t\tdosage := FHIRDosage{\n\t\t\tSequence: i + 1,\n\t\t\tText: rec.ClinicalNotes,\n\t\t}\n\t\t\n\t\t// Add dose quantity\n\t\tif rec.DoseMg > 0 {\n\t\t\tdosage.DoseAndRate = []FHIRDoseAndRate{\n\t\t\t\t{\n\t\t\t\t\tDoseQuantity: &FHIRQuantity{\n\t\t\t\t\t\tValue: rec.DoseMg,\n\t\t\t\t\t\tUnit:  \"mg\",\n\t\t\t\t\t\tCode:  \"mg\",\n\t\t\t\t\t\tSystem: \"http://unitsofmeasure.org\",\n\t\t\t\t\t},\n\t\t\t\t},\n\t\t\t}\n\t\t}\n\t\t\n\t\t// Add frequency\n\t\tif rec.FrequencyPerDay > 0 {\n\t\t\tdosage.Timing = &FHIRTiming{\n\t\t\t\tRepeat: &FHIRTimingRepeat{\n\t\t\t\t\tFrequency: rec.FrequencyPerDay,\n\t\t\t\t\tPeriod: 1,\n\t\t\t\t\tPeriodUnit: \"d\",\n\t\t\t\t},\n\t\t\t}\n\t\t}\n\t\t\n\t\t// Add route\n\t\tif rec.Route != \"\" {\n\t\t\tdosage.Route = &FHIRCodeableConcept{\n\t\t\t\tText: rec.Route,\n\t\t\t}\n\t\t}\n\t\t\n\t\tdosageInstructions = append(dosageInstructions, dosage)\n\t}\n\t\n\treturn dosageInstructions\n}\n\n// createFHIRSearchBundle creates a FHIR Bundle for search results\nfunc (h *FHIRHandler) createFHIRSearchBundle(result *services.ListProposalsResult, requestURL string, count, offset int) (*FHIRBundle, error) {\n\tentries := make([]FHIREntry, len(result.Proposals))\n\t\n\tfor i, proposal := range result.Proposals {\n\t\tfhirRequest, err := h.convertProposalToFHIRMedicationRequest(proposal)\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\t\n\t\tentries[i] = FHIREntry{\n\t\t\tResource: fhirRequest,\n\t\t\tSearch: &FHIRSearchMode{\n\t\t\t\tMode: \"match\",\n\t\t\t},\n\t\t}\n\t}\n\t\n\tbundle := &FHIRBundle{\n\t\tResourceType: \"Bundle\",\n\t\tID: uuid.New().String(),\n\t\tType: \"searchset\",\n\t\tTotal: result.TotalCount,\n\t\tEntry: entries,\n\t}\n\t\n\t// Add pagination links\n\tbaseURL := strings.Split(requestURL, \"?\")[0]\n\tif result.TotalCount > offset+count {\n\t\tnextOffset := offset + count\n\t\tbundle.Link = append(bundle.Link, FHIRLink{\n\t\t\tRelation: \"next\",\n\t\t\tURL: fmt.Sprintf(\"%s?_offset=%d&_count=%d\", baseURL, nextOffset, count),\n\t\t})\n\t}\n\t\n\tif offset > 0 {\n\t\tprevOffset := offset - count\n\t\tif prevOffset < 0 {\n\t\t\tprevOffset = 0\n\t\t}\n\t\tbundle.Link = append(bundle.Link, FHIRLink{\n\t\t\tRelation: \"previous\",\n\t\t\tURL: fmt.Sprintf(\"%s?_offset=%d&_count=%d\", baseURL, prevOffset, count),\n\t\t})\n\t}\n\t\n\treturn bundle, nil\n}"}]