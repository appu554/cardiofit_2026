package cdss

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"kb-7-terminology/internal/models"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// FactBuilder Service
// ============================================================================
// FactBuilder converts FHIR resources into structured clinical facts
// that can be evaluated against the THREE-CHECK PIPELINE.

// FactBuilder defines the interface for building clinical facts from FHIR resources
type FactBuilder interface {
	// BuildFactsFromBundle extracts facts from a FHIR Bundle
	BuildFactsFromBundle(ctx context.Context, patientID string, bundle *models.FHIRBundle, options *models.FactBuilderOptions) (*models.PatientFactSet, error)

	// BuildFactsFromRequest builds facts from a FactBuilderRequest
	BuildFactsFromRequest(ctx context.Context, request *models.FactBuilderRequest) (*models.FactBuilderResponse, error)

	// Individual resource type builders
	BuildFactsFromConditions(ctx context.Context, patientID string, conditions []models.FHIRCondition, options *models.FactBuilderOptions) ([]models.ClinicalFact, error)
	BuildFactsFromObservations(ctx context.Context, patientID string, observations []models.FHIRObservation, options *models.FactBuilderOptions) ([]models.ClinicalFact, error)
	BuildFactsFromMedications(ctx context.Context, patientID string, medications []models.FHIRMedicationRequest, options *models.FactBuilderOptions) ([]models.ClinicalFact, error)
	BuildFactsFromProcedures(ctx context.Context, patientID string, procedures []models.FHIRProcedure, options *models.FactBuilderOptions) ([]models.ClinicalFact, error)
	BuildFactsFromAllergies(ctx context.Context, patientID string, allergies []models.FHIRAllergyIntolerance, options *models.FactBuilderOptions) ([]models.ClinicalFact, error)
}

// factBuilderImpl implements the FactBuilder interface
type factBuilderImpl struct {
	logger *logrus.Logger
}

// NewFactBuilder creates a new FactBuilder instance
func NewFactBuilder(logger *logrus.Logger) FactBuilder {
	return &factBuilderImpl{
		logger: logger,
	}
}

// ============================================================================
// Bundle Processing
// ============================================================================

// BuildFactsFromBundle extracts all clinical facts from a FHIR Bundle
func (fb *factBuilderImpl) BuildFactsFromBundle(ctx context.Context, patientID string, bundle *models.FHIRBundle, options *models.FactBuilderOptions) (*models.PatientFactSet, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle is nil")
	}

	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	// Parse the bundle
	parsed, err := ParseBundle(bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	fb.logger.WithFields(logrus.Fields{
		"bundle_id":        bundle.ID,
		"total_entries":    parsed.TotalEntries,
		"parsed_entries":   parsed.ParsedEntries,
		"skipped_entries":  parsed.SkippedEntries,
		"conditions_count": len(parsed.Conditions),
		"observations_count": len(parsed.Observations),
		"medications_count": len(parsed.Medications),
	}).Debug("Parsed FHIR bundle")

	// Create fact set
	factSet := &models.PatientFactSet{
		PatientID:      patientID,
		ExtractedAt:    time.Now(),
		SourceBundleID: bundle.ID,
	}

	// Build facts from each resource type
	conditions, err := fb.BuildFactsFromConditions(ctx, patientID, parsed.Conditions, options)
	if err != nil {
		factSet.ProcessingErrors = append(factSet.ProcessingErrors, fmt.Sprintf("conditions: %v", err))
	} else {
		factSet.Conditions = conditions
	}

	observations, err := fb.BuildFactsFromObservations(ctx, patientID, parsed.Observations, options)
	if err != nil {
		factSet.ProcessingErrors = append(factSet.ProcessingErrors, fmt.Sprintf("observations: %v", err))
	} else {
		factSet.Observations = observations
	}

	medications, err := fb.BuildFactsFromMedications(ctx, patientID, parsed.Medications, options)
	if err != nil {
		factSet.ProcessingErrors = append(factSet.ProcessingErrors, fmt.Sprintf("medications: %v", err))
	} else {
		factSet.Medications = medications
	}

	procedures, err := fb.BuildFactsFromProcedures(ctx, patientID, parsed.Procedures, options)
	if err != nil {
		factSet.ProcessingErrors = append(factSet.ProcessingErrors, fmt.Sprintf("procedures: %v", err))
	} else {
		factSet.Procedures = procedures
	}

	allergies, err := fb.BuildFactsFromAllergies(ctx, patientID, parsed.Allergies, options)
	if err != nil {
		factSet.ProcessingErrors = append(factSet.ProcessingErrors, fmt.Sprintf("allergies: %v", err))
	} else {
		factSet.Allergies = allergies
	}

	// Update statistics
	factSet.UpdateStatistics()

	return factSet, nil
}

// BuildFactsFromRequest processes a FactBuilderRequest
func (fb *factBuilderImpl) BuildFactsFromRequest(ctx context.Context, request *models.FactBuilderRequest) (*models.FactBuilderResponse, error) {
	startTime := time.Now()
	response := &models.FactBuilderResponse{
		Success: true,
	}

	if request == nil {
		response.Success = false
		response.Errors = append(response.Errors, "request is nil")
		return response, nil
	}

	if request.PatientID == "" {
		response.Success = false
		response.Errors = append(response.Errors, "patient_id is required")
		return response, nil
	}

	if !request.HasResources() {
		response.Success = false
		response.Errors = append(response.Errors, "no FHIR resources provided")
		return response, nil
	}

	options := request.Options
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	// Create fact set
	factSet := &models.PatientFactSet{
		PatientID:   request.PatientID,
		EncounterID: request.EncounterID,
		ExtractedAt: time.Now(),
	}

	// Process bundle if provided
	if request.Bundle != nil {
		bundleFactSet, err := fb.BuildFactsFromBundle(ctx, request.PatientID, request.Bundle, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("bundle processing warning: %v", err))
		} else {
			factSet.Conditions = append(factSet.Conditions, bundleFactSet.Conditions...)
			factSet.Observations = append(factSet.Observations, bundleFactSet.Observations...)
			factSet.Medications = append(factSet.Medications, bundleFactSet.Medications...)
			factSet.Procedures = append(factSet.Procedures, bundleFactSet.Procedures...)
			factSet.Allergies = append(factSet.Allergies, bundleFactSet.Allergies...)
			factSet.SourceBundleID = bundleFactSet.SourceBundleID
		}
	}

	// Process individual resources
	if len(request.Conditions) > 0 {
		conditions, err := fb.BuildFactsFromConditions(ctx, request.PatientID, request.Conditions, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("conditions warning: %v", err))
		} else {
			factSet.Conditions = append(factSet.Conditions, conditions...)
			response.ConditionsProcessed = len(request.Conditions)
		}
	}

	if len(request.Observations) > 0 {
		observations, err := fb.BuildFactsFromObservations(ctx, request.PatientID, request.Observations, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("observations warning: %v", err))
		} else {
			factSet.Observations = append(factSet.Observations, observations...)
			response.ObservationsProcessed = len(request.Observations)
		}
	}

	if len(request.Medications) > 0 {
		medications, err := fb.BuildFactsFromMedications(ctx, request.PatientID, request.Medications, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("medications warning: %v", err))
		} else {
			factSet.Medications = append(factSet.Medications, medications...)
			response.MedicationsProcessed = len(request.Medications)
		}
	}

	if len(request.Procedures) > 0 {
		procedures, err := fb.BuildFactsFromProcedures(ctx, request.PatientID, request.Procedures, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("procedures warning: %v", err))
		} else {
			factSet.Procedures = append(factSet.Procedures, procedures...)
			response.ProceduresProcessed = len(request.Procedures)
		}
	}

	if len(request.Allergies) > 0 {
		allergies, err := fb.BuildFactsFromAllergies(ctx, request.PatientID, request.Allergies, options)
		if err != nil {
			response.Warnings = append(response.Warnings, fmt.Sprintf("allergies warning: %v", err))
		} else {
			factSet.Allergies = append(factSet.Allergies, allergies...)
			response.AllergiesProcessed = len(request.Allergies)
		}
	}

	// Update statistics
	factSet.UpdateStatistics()

	// Set response
	response.FactSet = factSet
	response.TotalResourcesProcessed = response.ConditionsProcessed + response.ObservationsProcessed +
		response.MedicationsProcessed + response.ProceduresProcessed + response.AllergiesProcessed
	response.TotalFactsExtracted = factSet.TotalFacts
	response.ProcessingTimeMs = float64(time.Since(startTime).Microseconds()) / 1000.0

	return response, nil
}

// ============================================================================
// Condition Processing
// ============================================================================

// BuildFactsFromConditions converts FHIR Conditions to ClinicalFacts
func (fb *factBuilderImpl) BuildFactsFromConditions(ctx context.Context, patientID string, conditions []models.FHIRCondition, options *models.FactBuilderOptions) ([]models.ClinicalFact, error) {
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	var facts []models.ClinicalFact
	for _, condition := range conditions {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return facts, ctx.Err()
		default:
		}

		// Check if we should include inactive conditions
		isActive := condition.IsActive()
		if !options.IncludeInactive && !isActive {
			continue
		}

		// Extract primary code
		primaryCode := ExtractPrimaryCode(condition.Code)
		if primaryCode == nil {
			continue // Skip conditions without codes
		}

		// Apply system filter if specified
		if len(options.TerminologySystems) > 0 && !containsSystem(options.TerminologySystems, primaryCode.System) {
			continue
		}

		// Generate fact ID
		factID := generateFactID(patientID, "condition", condition.ID, primaryCode.Code)

		// Create fact
		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           models.FactTypeCondition,
			Status:             ExtractClinicalStatus(condition.ClinicalStatus),
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			SourceResourceType: models.ResourceTypeCondition,
			SourceResourceID:   condition.ID,
		}

		// Add additional codes if requested
		if options.IncludeAllCodings && len(condition.Code.Coding) > 1 {
			for i := 1; i < len(condition.Code.Coding); i++ {
				coding := condition.Code.Coding[i]
				fact.AdditionalCodes = append(fact.AdditionalCodes, models.FactCoding{
					Code:    coding.Code,
					System:  NormalizeSystemURI(coding.System),
					Display: coding.Display,
					Version: coding.Version,
				})
			}
		}

		// Extract severity
		if condition.Severity != nil {
			fact.Severity = ExtractSeverity(condition.Severity)
		}

		// Extract timing
		if condition.OnsetDateTime != nil {
			fact.OnsetDateTime = condition.OnsetDateTime
		}
		if condition.RecordedDate != nil {
			fact.RecordedDateTime = condition.RecordedDate
		}

		// Extract category
		if len(condition.Category) > 0 {
			if code := ExtractPrimaryCode(&condition.Category[0]); code != nil {
				fact.Category = code.Display
				if fact.Category == "" {
					fact.Category = code.Code
				}
			}
		}

		// Derive clinical domain if requested
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		// Apply time filter
		if !isWithinTimeRange(fact.OnsetDateTime, fact.RecordedDateTime, options.EffectiveAfter, options.EffectiveBefore) {
			continue
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// ============================================================================
// Observation Processing
// ============================================================================

// BuildFactsFromObservations converts FHIR Observations to ClinicalFacts
func (fb *factBuilderImpl) BuildFactsFromObservations(ctx context.Context, patientID string, observations []models.FHIRObservation, options *models.FactBuilderOptions) ([]models.ClinicalFact, error) {
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	var facts []models.ClinicalFact
	for _, observation := range observations {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return facts, ctx.Err()
		default:
		}

		// Extract primary code
		primaryCode := ExtractPrimaryCode(observation.Code)
		if primaryCode == nil {
			continue
		}

		// Apply system filter
		if len(options.TerminologySystems) > 0 && !containsSystem(options.TerminologySystems, primaryCode.System) {
			continue
		}

		// Determine fact type from category
		category := ExtractObservationCategory(observation.Category)
		factType := CategoryToFactType(category)

		// Generate fact ID
		factID := generateFactID(patientID, "observation", observation.ID, primaryCode.Code)

		// Create fact
		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           factType,
			Status:             ExtractObservationStatus(observation.Status),
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			Category:           string(category),
			SourceResourceType: models.ResourceTypeObservation,
			SourceResourceID:   observation.ID,
		}

		// Extract numeric value
		if value, ok := observation.GetNumericValue(); ok {
			fact.NumericValue = &value
			if observation.ValueQuantity != nil {
				fact.Unit = observation.ValueQuantity.Unit
				if fact.Unit == "" {
					fact.Unit = observation.ValueQuantity.Code
				}
			}
		}

		// Extract coded value
		if observation.ValueCodeableConcept != nil {
			if code := ExtractPrimaryCode(observation.ValueCodeableConcept); code != nil {
				fact.CodedValue = code.Code
			}
		}

		// Extract boolean value
		if observation.ValueBoolean != nil {
			fact.BooleanValue = observation.ValueBoolean
		}

		// Extract reference range
		if len(observation.ReferenceRange) > 0 {
			rr := observation.ReferenceRange[0]
			if rr.Low != nil {
				fact.ReferenceRangeLow = &rr.Low.Value
			}
			if rr.High != nil {
				fact.ReferenceRangeHigh = &rr.High.Value
			}
		}

		// Extract interpretation
		interp, isAbnormal, isCritical := ExtractInterpretation(observation.Interpretation)
		fact.Interpretation = string(interp)
		fact.IsAbnormal = isAbnormal
		fact.IsCritical = isCritical

		// Check observation's built-in abnormal detection
		if observation.IsAbnormal() && !fact.IsAbnormal {
			fact.IsAbnormal = true
		}

		// Extract timing
		if observation.EffectiveDateTime != nil {
			fact.EffectiveDateTime = observation.EffectiveDateTime
		}
		if observation.Issued != nil {
			fact.RecordedDateTime = observation.Issued
		}

		// Derive clinical domain
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		// Apply time filter
		if !isWithinTimeRange(fact.EffectiveDateTime, fact.RecordedDateTime, options.EffectiveAfter, options.EffectiveBefore) {
			continue
		}

		facts = append(facts, fact)

		// Extract component observations if requested
		if options.ExtractComponents && len(observation.Component) > 0 {
			componentFacts := fb.extractComponentFacts(patientID, observation.ID, observation.Component, options)
			facts = append(facts, componentFacts...)
		}
	}

	return facts, nil
}

// extractComponentFacts extracts facts from observation components
func (fb *factBuilderImpl) extractComponentFacts(patientID, parentID string, components []models.ObservationComponent, options *models.FactBuilderOptions) []models.ClinicalFact {
	var facts []models.ClinicalFact

	for i, component := range components {
		primaryCode := ExtractPrimaryCode(component.Code)
		if primaryCode == nil {
			continue
		}

		factID := generateFactID(patientID, "observation-component", parentID, fmt.Sprintf("%s-%d", primaryCode.Code, i))

		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           models.FactTypeObservation,
			Status:             models.FactStatusCompleted,
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			SourceResourceType: models.ResourceTypeObservation,
			SourceResourceID:   fmt.Sprintf("%s#component[%d]", parentID, i),
		}

		// Extract numeric value
		if component.ValueQuantity != nil {
			fact.NumericValue = &component.ValueQuantity.Value
			fact.Unit = component.ValueQuantity.Unit
		}

		// Extract interpretation
		interp, isAbnormal, isCritical := ExtractInterpretation(component.Interpretation)
		fact.Interpretation = string(interp)
		fact.IsAbnormal = isAbnormal
		fact.IsCritical = isCritical

		// Derive clinical domain
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		facts = append(facts, fact)
	}

	return facts
}

// ============================================================================
// Medication Processing
// ============================================================================

// BuildFactsFromMedications converts FHIR MedicationRequests to ClinicalFacts
func (fb *factBuilderImpl) BuildFactsFromMedications(ctx context.Context, patientID string, medications []models.FHIRMedicationRequest, options *models.FactBuilderOptions) ([]models.ClinicalFact, error) {
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	var facts []models.ClinicalFact
	for _, medication := range medications {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return facts, ctx.Err()
		default:
		}

		// Check if we should include inactive medications
		isActive := medication.IsActive()
		if !options.IncludeInactive && !isActive {
			continue
		}

		// Extract medication code
		var primaryCode *ExtractedCode
		if medication.MedicationCodeableConcept != nil {
			primaryCode = ExtractPrimaryCode(medication.MedicationCodeableConcept)
		}
		if primaryCode == nil {
			continue
		}

		// Apply system filter
		if len(options.TerminologySystems) > 0 && !containsSystem(options.TerminologySystems, primaryCode.System) {
			continue
		}

		// Generate fact ID
		factID := generateFactID(patientID, "medication", medication.ID, primaryCode.Code)

		// Create fact
		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           models.FactTypeMedication,
			Status:             ExtractMedicationStatus(medication.Status),
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			SourceResourceType: models.ResourceTypeMedicationRequest,
			SourceResourceID:   medication.ID,
		}

		// Add additional codes
		if options.IncludeAllCodings && medication.MedicationCodeableConcept != nil && len(medication.MedicationCodeableConcept.Coding) > 1 {
			for i := 1; i < len(medication.MedicationCodeableConcept.Coding); i++ {
				coding := medication.MedicationCodeableConcept.Coding[i]
				fact.AdditionalCodes = append(fact.AdditionalCodes, models.FactCoding{
					Code:    coding.Code,
					System:  NormalizeSystemURI(coding.System),
					Display: coding.Display,
					Version: coding.Version,
				})
			}
		}

		// Extract timing
		if medication.AuthoredOn != nil {
			fact.RecordedDateTime = medication.AuthoredOn
		}

		// Extract category
		if len(medication.Category) > 0 {
			if code := ExtractPrimaryCode(&medication.Category[0]); code != nil {
				fact.Category = code.Display
				if fact.Category == "" {
					fact.Category = code.Code
				}
			}
		}

		// Derive clinical domain
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		// Apply time filter
		if !isWithinTimeRange(nil, fact.RecordedDateTime, options.EffectiveAfter, options.EffectiveBefore) {
			continue
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// ============================================================================
// Procedure Processing
// ============================================================================

// BuildFactsFromProcedures converts FHIR Procedures to ClinicalFacts
func (fb *factBuilderImpl) BuildFactsFromProcedures(ctx context.Context, patientID string, procedures []models.FHIRProcedure, options *models.FactBuilderOptions) ([]models.ClinicalFact, error) {
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	var facts []models.ClinicalFact
	for _, procedure := range procedures {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return facts, ctx.Err()
		default:
		}

		// Extract primary code
		primaryCode := ExtractPrimaryCode(procedure.Code)
		if primaryCode == nil {
			continue
		}

		// Apply system filter
		if len(options.TerminologySystems) > 0 && !containsSystem(options.TerminologySystems, primaryCode.System) {
			continue
		}

		// Generate fact ID
		factID := generateFactID(patientID, "procedure", procedure.ID, primaryCode.Code)

		// Create fact
		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           models.FactTypeProcedure,
			Status:             ExtractProcedureStatus(procedure.Status),
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			SourceResourceType: models.ResourceTypeProcedure,
			SourceResourceID:   procedure.ID,
		}

		// Add additional codes
		if options.IncludeAllCodings && len(procedure.Code.Coding) > 1 {
			for i := 1; i < len(procedure.Code.Coding); i++ {
				coding := procedure.Code.Coding[i]
				fact.AdditionalCodes = append(fact.AdditionalCodes, models.FactCoding{
					Code:    coding.Code,
					System:  NormalizeSystemURI(coding.System),
					Display: coding.Display,
					Version: coding.Version,
				})
			}
		}

		// Extract timing
		if procedure.PerformedDateTime != nil {
			fact.EffectiveDateTime = procedure.PerformedDateTime
		} else if procedure.PerformedPeriod != nil && procedure.PerformedPeriod.Start != nil {
			fact.EffectiveDateTime = procedure.PerformedPeriod.Start
		}

		// Extract category
		if procedure.Category != nil {
			if code := ExtractPrimaryCode(procedure.Category); code != nil {
				fact.Category = code.Display
				if fact.Category == "" {
					fact.Category = code.Code
				}
			}
		}

		// Derive clinical domain
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		// Apply time filter
		if !isWithinTimeRange(fact.EffectiveDateTime, nil, options.EffectiveAfter, options.EffectiveBefore) {
			continue
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// ============================================================================
// Allergy Processing
// ============================================================================

// BuildFactsFromAllergies converts FHIR AllergyIntolerances to ClinicalFacts
func (fb *factBuilderImpl) BuildFactsFromAllergies(ctx context.Context, patientID string, allergies []models.FHIRAllergyIntolerance, options *models.FactBuilderOptions) ([]models.ClinicalFact, error) {
	if options == nil {
		options = models.DefaultFactBuilderOptions()
	}

	var facts []models.ClinicalFact
	for _, allergy := range allergies {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return facts, ctx.Err()
		default:
		}

		// Check if we should include inactive allergies
		isActive := allergy.IsActive()
		if !options.IncludeInactive && !isActive {
			continue
		}

		// Extract primary code
		primaryCode := ExtractPrimaryCode(allergy.Code)
		if primaryCode == nil {
			continue
		}

		// Apply system filter
		if len(options.TerminologySystems) > 0 && !containsSystem(options.TerminologySystems, primaryCode.System) {
			continue
		}

		// Generate fact ID
		factID := generateFactID(patientID, "allergy", allergy.ID, primaryCode.Code)

		// Determine status
		status := models.FactStatusActive
		if !isActive {
			status = models.FactStatusInactive
		}

		// Create fact
		fact := models.ClinicalFact{
			ID:                 factID,
			FactType:           models.FactTypeAllergy,
			Status:             status,
			Code:               primaryCode.Code,
			System:             primaryCode.System,
			Display:            primaryCode.Display,
			Criticality:        allergy.Criticality,
			SourceResourceType: models.ResourceTypeAllergyIntolerance,
			SourceResourceID:   allergy.ID,
		}

		// Add additional codes
		if options.IncludeAllCodings && allergy.Code != nil && len(allergy.Code.Coding) > 1 {
			for i := 1; i < len(allergy.Code.Coding); i++ {
				coding := allergy.Code.Coding[i]
				fact.AdditionalCodes = append(fact.AdditionalCodes, models.FactCoding{
					Code:    coding.Code,
					System:  NormalizeSystemURI(coding.System),
					Display: coding.Display,
					Version: coding.Version,
				})
			}
		}

		// Mark critical allergies
		if allergy.Criticality == "high" {
			fact.IsCritical = true
		}

		// Extract timing
		if allergy.OnsetDateTime != nil {
			fact.OnsetDateTime = allergy.OnsetDateTime
		}
		if allergy.RecordedDate != nil {
			fact.RecordedDateTime = allergy.RecordedDate
		}

		// Set category from allergy type
		if allergy.Type != "" {
			fact.Category = allergy.Type
		}
		if len(allergy.Category) > 0 {
			fact.Category = allergy.Category[0]
		}

		// Derive clinical domain
		if options.DeriveClinicalDomains {
			fact.ClinicalDomain = deriveClinicalDomainFromCode(primaryCode.Code, primaryCode.System)
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateFactID creates a deterministic ID for a fact
func generateFactID(patientID, resourceType, resourceID, code string) string {
	// Create a deterministic hash from the components
	input := fmt.Sprintf("%s|%s|%s|%s", patientID, resourceType, resourceID, code)
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars for brevity
}

// containsSystem checks if a system is in the list of allowed systems
func containsSystem(systems []string, system string) bool {
	normalizedSystem := NormalizeSystemURI(system)
	for _, s := range systems {
		if NormalizeSystemURI(s) == normalizedSystem {
			return true
		}
	}
	return false
}

// isWithinTimeRange checks if a fact is within the specified time range
func isWithinTimeRange(effectiveTime, recordedTime *time.Time, after, before *time.Time) bool {
	// If no time filters specified, accept all
	if after == nil && before == nil {
		return true
	}

	// Use effective time if available, otherwise recorded time
	var checkTime *time.Time
	if effectiveTime != nil {
		checkTime = effectiveTime
	} else if recordedTime != nil {
		checkTime = recordedTime
	}

	// If no time available, include the fact (can't filter)
	if checkTime == nil {
		return true
	}

	// Check time range
	if after != nil && checkTime.Before(*after) {
		return false
	}
	if before != nil && checkTime.After(*before) {
		return false
	}

	return true
}

// deriveClinicalDomainFromCode attempts to derive clinical domain from a code
// This is a simplified implementation - a real system would use terminology lookups
func deriveClinicalDomainFromCode(code, system string) string {
	// This is a placeholder - in production, this would query Neo4j or a terminology service
	// to determine the clinical domain based on SNOMED concept hierarchy or ICD chapter

	// For now, return empty and let the CDSS Evaluator determine domain via value set matching
	return ""
}
