// Package ordersets provides the core order set engine for KB-12
package ordersets

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-12-ordersets-careplans/internal/cache"
	"kb-12-ordersets-careplans/internal/clients"
	"kb-12-ordersets-careplans/internal/models"
)

// OrderSetService provides order set management functionality
type OrderSetService struct {
	db        *gorm.DB
	cache     *cache.Cache
	kb1Client *clients.KB1DosingClient
	kb3Client *clients.KB3TemporalClient
	kb6Client *clients.KB6FormularyClient
	kb7Client *clients.KB7TerminologyClient
	log       *logrus.Entry

	// Template registry
	templatesMu sync.RWMutex
	templates   map[string]*models.OrderSetTemplate
}

// NewOrderSetService creates a new order set service
func NewOrderSetService(
	db *gorm.DB,
	cache *cache.Cache,
	kb1Client *clients.KB1DosingClient,
	kb3Client *clients.KB3TemporalClient,
	kb6Client *clients.KB6FormularyClient,
	kb7Client *clients.KB7TerminologyClient,
) *OrderSetService {
	svc := &OrderSetService{
		db:        db,
		cache:     cache,
		kb1Client: kb1Client,
		kb3Client: kb3Client,
		kb6Client: kb6Client,
		kb7Client: kb7Client,
		log:       logrus.WithField("service", "orderset"),
		templates: make(map[string]*models.OrderSetTemplate),
	}

	return svc
}

// Initialize loads all templates into memory and database
func (s *OrderSetService) Initialize(ctx context.Context) error {
	s.log.Info("Initializing order set service")

	// Load built-in templates
	admissionTemplates := GetAllAdmissionOrderSets()
	procedureTemplates := GetAllProcedureOrderSets()
	emergencyTemplates := GetAllEmergencyProtocols()

	allTemplates := append(admissionTemplates, procedureTemplates...)
	allTemplates = append(allTemplates, emergencyTemplates...)

	s.templatesMu.Lock()
	defer s.templatesMu.Unlock()

	for _, tmpl := range allTemplates {
		// Register in memory
		s.templates[tmpl.TemplateID] = tmpl

		// Upsert to database
		var existing models.OrderSetTemplate
		err := s.db.WithContext(ctx).Where("template_id = ?", tmpl.TemplateID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := s.db.WithContext(ctx).Create(&tmpl).Error; err != nil {
				s.log.WithError(err).WithField("template_id", tmpl.TemplateID).Error("Failed to create template")
				continue
			}
			s.log.WithField("template_id", tmpl.TemplateID).Info("Created order set template")
		} else if err == nil {
			// Update existing template
			existing.Name = tmpl.Name
			existing.Version = tmpl.Version
			existing.GuidelineSource = tmpl.GuidelineSource
			existing.Description = tmpl.Description
			existing.Orders = tmpl.Orders
			existing.TimeConstraints = tmpl.TimeConstraints
			existing.Active = tmpl.Active
			if err := s.db.WithContext(ctx).Save(&existing).Error; err != nil {
				s.log.WithError(err).WithField("template_id", tmpl.TemplateID).Error("Failed to update template")
			}
		}
	}

	s.log.WithField("count", len(allTemplates)).Info("Order set templates initialized")
	return nil
}

// GetTemplate retrieves a template by ID
func (s *OrderSetService) GetTemplate(ctx context.Context, templateID string) (*models.OrderSetTemplate, error) {
	// Check memory cache first
	s.templatesMu.RLock()
	if tmpl, ok := s.templates[templateID]; ok {
		s.templatesMu.RUnlock()
		return tmpl, nil
	}
	s.templatesMu.RUnlock()

	// Check Redis cache
	if cached, err := s.cache.GetOrderSetTemplate(ctx, templateID); err == nil {
		var template models.OrderSetTemplate
		if json.Unmarshal(cached, &template) == nil {
			return &template, nil
		}
	}

	// Load from database
	var template models.OrderSetTemplate
	if err := s.db.WithContext(ctx).Where("template_id = ?", templateID).First(&template).Error; err != nil {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Cache in Redis
	if data, err := json.Marshal(template); err == nil {
		s.cache.SetOrderSetTemplate(ctx, templateID, data)
	}

	return &template, nil
}

// ListTemplates retrieves templates with filtering
func (s *OrderSetService) ListTemplates(ctx context.Context, category string, activeOnly bool, limit, offset int) ([]models.OrderSetTemplate, int64, error) {
	query := s.db.WithContext(ctx).Model(&models.OrderSetTemplate{})

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if activeOnly {
		query = query.Where("active = ?", true)
	}

	var total int64
	query.Count(&total)

	var templates []models.OrderSetTemplate
	if err := query.Limit(limit).Offset(offset).Order("name ASC").Find(&templates).Error; err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// SearchTemplates searches templates by name or description
func (s *OrderSetService) SearchTemplates(ctx context.Context, query string, limit int) ([]models.OrderSetTemplate, error) {
	var templates []models.OrderSetTemplate
	searchQuery := "%" + query + "%"

	if err := s.db.WithContext(ctx).
		Where("active = ?", true).
		Where("name ILIKE ? OR description ILIKE ?", searchQuery, searchQuery).
		Limit(limit).
		Order("name ASC").
		Find(&templates).Error; err != nil {
		return nil, err
	}

	return templates, nil
}

// ActivateOrderSet creates an instance from a template
func (s *OrderSetService) ActivateOrderSet(ctx context.Context, req *models.OrderSetActivationRequest) (*models.OrderSetActivationResponse, error) {
	// Get template
	template, err := s.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		return nil, err
	}

	if !template.Active {
		return nil, fmt.Errorf("template is not active: %s", req.TemplateID)
	}

	// Get orders from template
	orders, err := template.GetOrders()
	if err != nil {
		return nil, fmt.Errorf("failed to parse template orders: %w", err)
	}

	// Filter and customize orders
	orders = s.filterAndCustomizeOrders(ctx, orders, req)

	// Perform safety checks
	safetyAlerts := s.performSafetyChecks(ctx, orders, req.PatientContext)

	// Check for blocking alerts
	for _, alert := range safetyAlerts {
		if alert.Severity == "high" && !alert.Overridable {
			return &models.OrderSetActivationResponse{
				Success:      false,
				SafetyAlerts: safetyAlerts,
				ErrorMessage: "Order set activation blocked by safety alerts",
			}, nil
		}
	}

	// Create instance
	instance := models.OrderSetInstance{
		InstanceID:  "OSI-" + uuid.New().String()[:8],
		TemplateID:  req.TemplateID,
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		ActivatedBy: req.ActivatedBy,
		Status:      models.OrderStatusActive,
		ActivatedAt: time.Now(),
	}

	if err := instance.SetOrders(orders); err != nil {
		return nil, fmt.Errorf("failed to set orders: %w", err)
	}

	// Initialize constraint status for time-critical protocols
	constraints, _ := template.GetTimeConstraints()
	if len(constraints) > 0 {
		statuses := s.initializeConstraintStatuses(constraints, instance.ActivatedAt)
		instance.SetConstraintStatus(statuses)
	}

	// Save instance
	if err := s.db.WithContext(ctx).Create(&instance).Error; err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Invalidate patient cache
	s.cache.InvalidatePatientOrderSets(ctx, req.PatientID)

	// Build response
	response := &models.OrderSetActivationResponse{
		Success:      true,
		InstanceID:   instance.InstanceID,
		TemplateID:   template.TemplateID,
		TemplateName: template.Name,
		PatientID:    req.PatientID,
		EncounterID:  req.EncounterID,
		Orders:       orders,
		SafetyAlerts: safetyAlerts,
		ActivatedAt:  instance.ActivatedAt,
	}

	if statuses, _ := instance.GetConstraintStatus(); len(statuses) > 0 {
		response.Constraints = statuses
	}

	s.log.WithFields(logrus.Fields{
		"instance_id": instance.InstanceID,
		"template_id": req.TemplateID,
		"patient_id":  req.PatientID,
	}).Info("Order set activated")

	return response, nil
}

// filterAndCustomizeOrders filters selected orders and applies customizations
func (s *OrderSetService) filterAndCustomizeOrders(ctx context.Context, orders []models.Order, req *models.OrderSetActivationRequest) []models.Order {
	// Create selection map
	selectedMap := make(map[string]bool)
	if len(req.SelectedOrders) > 0 {
		for _, id := range req.SelectedOrders {
			selectedMap[id] = true
		}
	}

	var result []models.Order
	for _, order := range orders {
		// Skip non-selected optional orders
		if len(req.SelectedOrders) > 0 && !selectedMap[order.OrderID] && !order.Required {
			continue
		}

		order.Selected = true

		// Apply weight-based dosing if applicable
		if order.OrderType == models.OrderTypeMedication && req.PatientContext.Weight > 0 {
			order = s.applyWeightBasedDosing(ctx, order, req.PatientContext)
		}

		// Apply renal dose adjustments
		if order.OrderType == models.OrderTypeMedication && req.PatientContext.RenalFunction != "" {
			order = s.applyRenalAdjustment(ctx, order, req.PatientContext)
		}

		result = append(result, order)
	}

	return result
}

// applyWeightBasedDosing applies weight-based dosing using KB-1
func (s *OrderSetService) applyWeightBasedDosing(ctx context.Context, order models.Order, patientCtx models.PatientContext) models.Order {
	if order.DrugCode == "" || patientCtx.Weight <= 0 {
		return order
	}

	// Try to get weight-based dosing from KB-1
	if s.kb1Client.IsEnabled() {
		resp, err := s.kb1Client.GetWeightBasedDosing(ctx, order.DrugCode, "")
		if err == nil && resp.Success && resp.DosePerKg > 0 {
			calculatedDose := resp.DosePerKg * patientCtx.Weight

			// Apply min/max limits
			if resp.MinDose > 0 && calculatedDose < resp.MinDose {
				calculatedDose = resp.MinDose
			}
			if resp.MaxDose > 0 && calculatedDose > resp.MaxDose {
				calculatedDose = resp.MaxDose
			}

			order.DoseValue = calculatedDose
			order.DoseUnit = resp.DoseUnit
			order.Dose = fmt.Sprintf("%.1f %s", calculatedDose, resp.DoseUnit)
			order.Notes = fmt.Sprintf("Weight-based: %.2f %s/kg × %.1f kg", resp.DosePerKg, resp.DoseUnit, patientCtx.Weight)
		}
	}

	return order
}

// applyRenalAdjustment applies renal dose adjustments using KB-1
func (s *OrderSetService) applyRenalAdjustment(ctx context.Context, order models.Order, patientCtx models.PatientContext) models.Order {
	if order.DrugCode == "" {
		return order
	}

	// Request dose calculation with renal function (using KB-1 API format)
	if s.kb1Client.IsEnabled() {
		req := &clients.DoseCalculationRequest{
			RxNormCode: order.DrugCode,
			Patient: clients.PatientParameters{
				Age:             patientCtx.Age,
				Gender:          patientCtx.Gender,
				WeightKg:        patientCtx.Weight,
				HeightCm:        patientCtx.Height,
				SerumCreatinine: 1.0, // Default, will use CrCl from patientCtx
			},
		}

		resp, err := s.kb1Client.CalculateDose(ctx, req)
		if err == nil && resp.Success {
			// Check for dose adjustments
			for _, adj := range resp.Adjustments {
				if adj.AdjustmentType == "decrease" && adj.NewDose > 0 {
					order.DoseValue = adj.NewDose
					order.Dose = fmt.Sprintf("%.1f %s", adj.NewDose, adj.Unit)
					order.Notes = fmt.Sprintf("Renal adjustment: %s", adj.Reason)
				}
			}
		}
	}

	return order
}

// performSafetyChecks performs safety checks on orders
func (s *OrderSetService) performSafetyChecks(ctx context.Context, orders []models.Order, patientCtx models.PatientContext) []models.SafetyAlert {
	var alerts []models.SafetyAlert

	// Collect drug codes for interaction check
	var drugCodes []string
	for _, order := range orders {
		if order.OrderType == models.OrderTypeMedication && order.DrugCode != "" {
			drugCodes = append(drugCodes, order.DrugCode)
		}
	}

	// Add patient's active medications
	drugCodes = append(drugCodes, patientCtx.ActiveMeds...)

	// Check drug-drug interactions via KB-6
	if len(drugCodes) >= 2 && s.kb6Client.IsEnabled() {
		interactionReq := &clients.DrugInteractionRequest{
			DrugCodes: drugCodes,
		}

		resp, err := s.kb6Client.CheckInteractions(ctx, interactionReq)
		if err == nil && resp.Success {
			for _, interaction := range resp.Interactions {
				alert := models.SafetyAlert{
					AlertID:   "INT-" + uuid.New().String()[:8],
					AlertType: "interaction",
					Severity:  interaction.Severity,
					Message:   fmt.Sprintf("Drug interaction: %s and %s", interaction.Drug1Name, interaction.Drug2Name),
					Details:   interaction.Description,
					AffectedOrders: []string{interaction.Drug1Code, interaction.Drug2Code},
					Overridable: interaction.Severity != "severe",
					Reference:   interaction.Reference,
				}
				alerts = append(alerts, alert)
			}
		}
	}

	// Check allergies
	for _, order := range orders {
		if order.OrderType == models.OrderTypeMedication {
			for _, allergy := range patientCtx.Allergies {
				// Simple check - in production would use semantic matching
				if order.DrugName != "" && containsIgnoreCase(order.DrugName, allergy) {
					alert := models.SafetyAlert{
						AlertID:     "ALG-" + uuid.New().String()[:8],
						AlertType:   "allergy",
						Severity:    "high",
						Message:     fmt.Sprintf("Patient allergy to %s", allergy),
						SourceOrder: order.OrderID,
						Overridable: true,
					}
					alerts = append(alerts, alert)
				}
			}
		}
	}

	// Check for duplicate orders
	seenDrugs := make(map[string]string)
	for _, order := range orders {
		if order.OrderType == models.OrderTypeMedication && order.DrugCode != "" {
			if existingOrderID, exists := seenDrugs[order.DrugCode]; exists {
				alert := models.SafetyAlert{
					AlertID:        "DUP-" + uuid.New().String()[:8],
					AlertType:      "duplicate",
					Severity:       "medium",
					Message:        fmt.Sprintf("Duplicate medication: %s", order.DrugName),
					SourceOrder:    order.OrderID,
					AffectedOrders: []string{existingOrderID},
					Overridable:    true,
				}
				alerts = append(alerts, alert)
			} else {
				seenDrugs[order.DrugCode] = order.OrderID
			}
		}
	}

	return alerts
}

// initializeConstraintStatuses creates initial constraint statuses
func (s *OrderSetService) initializeConstraintStatuses(constraints []models.TimeConstraint, startTime time.Time) []models.ConstraintStatus {
	var statuses []models.ConstraintStatus

	for _, c := range constraints {
		statuses = append(statuses, models.ConstraintStatus{
			ConstraintID:    c.ConstraintID,
			Action:          c.Action,
			Status:          "pending",
			StartTime:       startTime,
			Deadline:        startTime.Add(c.Deadline),
			Severity:        c.Severity,
			PercentComplete: 0,
		})
	}

	return statuses
}

// ValidateTimeConstraints validates time constraints for an instance
func (s *OrderSetService) ValidateTimeConstraints(ctx context.Context, instanceID string) ([]models.ConstraintStatus, error) {
	// Get instance
	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	statuses, err := instance.GetConstraintStatus()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	updated := false

	for i := range statuses {
		if statuses[i].CompletedAt != nil {
			continue
		}

		statuses[i].TimeElapsed = now.Sub(statuses[i].StartTime)
		statuses[i].TimeRemaining = statuses[i].Deadline.Sub(now)

		deadline := statuses[i].Deadline.Sub(statuses[i].StartTime)
		if deadline > 0 {
			statuses[i].PercentComplete = float64(statuses[i].TimeElapsed) / float64(deadline) * 100
			if statuses[i].PercentComplete > 100 {
				statuses[i].PercentComplete = 100
			}
		}

		// Update status based on time
		if statuses[i].TimeRemaining < 0 {
			statuses[i].Status = "overdue"
			statuses[i].TimeRemaining = 0
			updated = true
		} else if statuses[i].PercentComplete >= 80 {
			statuses[i].Status = "warning"
			updated = true
		}

		// Use KB-3 for advanced validation if available
		if s.kb3Client.IsEnabled() {
			resp, err := s.kb3Client.ValidateConstraintTiming(
				ctx,
				now,
				statuses[i].StartTime,
				deadline,
				0,
			)
			if err == nil {
				statuses[i].Status = resp.Status
			}
		}
	}

	// Save updated statuses
	if updated {
		instance.SetConstraintStatus(statuses)
		s.db.WithContext(ctx).Save(&instance)
		s.cache.InvalidateConstraintStatus(ctx, instanceID)
	}

	return statuses, nil
}

// GetOverdueConstraints retrieves all overdue constraints across instances
func (s *OrderSetService) GetOverdueConstraints(ctx context.Context, patientID string) ([]ConstraintAlert, error) {
	var instances []models.OrderSetInstance

	query := s.db.WithContext(ctx).Where("status = ?", models.OrderStatusActive)
	if patientID != "" {
		query = query.Where("patient_id = ?", patientID)
	}

	if err := query.Find(&instances).Error; err != nil {
		return nil, err
	}

	var alerts []ConstraintAlert
	now := time.Now()

	for _, instance := range instances {
		statuses, err := instance.GetConstraintStatus()
		if err != nil {
			continue
		}

		for _, status := range statuses {
			if status.CompletedAt != nil {
				continue
			}

			if now.After(status.Deadline) {
				alerts = append(alerts, ConstraintAlert{
					InstanceID:    instance.InstanceID,
					PatientID:     instance.PatientID,
					EncounterID:   instance.EncounterID,
					ConstraintID:  status.ConstraintID,
					Action:        status.Action,
					Deadline:      status.Deadline,
					TimeOverdue:   now.Sub(status.Deadline),
					Severity:      status.Severity,
				})
			}
		}
	}

	return alerts, nil
}

// ConstraintAlert represents an alert for an overdue constraint
type ConstraintAlert struct {
	InstanceID   string        `json:"instance_id"`
	PatientID    string        `json:"patient_id"`
	EncounterID  string        `json:"encounter_id"`
	ConstraintID string        `json:"constraint_id"`
	Action       string        `json:"action"`
	Deadline     time.Time     `json:"deadline"`
	TimeOverdue  time.Duration `json:"time_overdue"`
	Severity     string        `json:"severity"`
}

// GenerateFHIRBundle generates a FHIR Bundle for an order set instance
func (s *OrderSetService) GenerateFHIRBundle(ctx context.Context, instanceID string) (*models.FHIRBundle, error) {
	var instance models.OrderSetInstance
	if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&instance).Error; err != nil {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}

	orders, err := instance.GetOrders()
	if err != nil {
		return nil, err
	}

	bundle := &models.FHIRBundle{
		ResourceType: "Bundle",
		ID:           uuid.New().String(),
		Type:         "collection",
		Timestamp:    time.Now(),
		Total:        len(orders),
	}

	for _, order := range orders {
		if !order.Selected {
			continue
		}

		var resource interface{}
		var resourceType string

		switch order.OrderType {
		case models.OrderTypeMedication, models.OrderTypeIVFluids:
			resource = s.generateMedicationRequest(order, instance)
			resourceType = "MedicationRequest"
		case models.OrderTypeLab, models.OrderTypeImaging, models.OrderTypeProcedure:
			resource = s.generateServiceRequest(order, instance)
			resourceType = "ServiceRequest"
		default:
			resource = s.generateTask(order, instance)
			resourceType = "Task"
		}

		entry := models.BundleEntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", uuid.New().String()),
			Resource: resource,
			Request: &models.BundleRequest{
				Method: "POST",
				URL:    resourceType,
			},
		}
		bundle.Entry = append(bundle.Entry, entry)
	}

	return bundle, nil
}

// generateMedicationRequest generates a FHIR MedicationRequest
func (s *OrderSetService) generateMedicationRequest(order models.Order, instance models.OrderSetInstance) *models.FHIRMedicationRequest {
	req := &models.FHIRMedicationRequest{
		ResourceType: "MedicationRequest",
		ID:           order.OrderID,
		Status:       "active",
		Intent:       "order",
		Priority:     string(order.Priority),
		Subject: models.Reference{
			Reference: "Patient/" + instance.PatientID,
		},
		Encounter: &models.Reference{
			Reference: "Encounter/" + instance.EncounterID,
		},
		AuthoredOn: instance.ActivatedAt,
		Requester: &models.Reference{
			Display: instance.ActivatedBy,
		},
	}

	// Set medication
	if order.DrugCode != "" {
		req.MedicationCodeableConcept = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
					Code:    order.DrugCode,
					Display: order.DrugName,
				},
			},
			Text: order.DrugName,
		}
	}

	// Set dosage
	if order.Dose != "" || order.Route != "" || order.Frequency != "" {
		dosage := models.DosageInstruction{
			Text: fmt.Sprintf("%s %s %s", order.Dose, order.Route, order.Frequency),
		}

		if order.Route != "" {
			dosage.Route = &models.CodeableConcept{
				Text: order.Route,
			}
		}

		if order.DoseValue > 0 {
			dosage.DoseAndRate = []models.DoseAndRate{
				{
					DoseQuantity: &models.Quantity{
						Value: order.DoseValue,
						Unit:  order.DoseUnit,
					},
				},
			}
		}

		if order.PRN {
			dosage.AsNeededBoolean = true
			if order.PRNReason != "" {
				dosage.AsNeededCodeableConcept = &models.CodeableConcept{
					Text: order.PRNReason,
				}
			}
		}

		req.DosageInstruction = []models.DosageInstruction{dosage}
	}

	return req
}

// generateServiceRequest generates a FHIR ServiceRequest
func (s *OrderSetService) generateServiceRequest(order models.Order, instance models.OrderSetInstance) *models.FHIRServiceRequest {
	req := &models.FHIRServiceRequest{
		ResourceType: "ServiceRequest",
		ID:           order.OrderID,
		Status:       "active",
		Intent:       "order",
		Priority:     string(order.Priority),
		Subject: models.Reference{
			Reference: "Patient/" + instance.PatientID,
		},
		Encounter: &models.Reference{
			Reference: "Encounter/" + instance.EncounterID,
		},
		AuthoredOn: instance.ActivatedAt,
		Requester: &models.Reference{
			Display: instance.ActivatedBy,
		},
	}

	// Set code based on order type
	var system, code, display string
	switch order.OrderType {
	case models.OrderTypeLab:
		system = "http://loinc.org"
		code = order.LabCode
		display = order.Name
	case models.OrderTypeImaging:
		system = "http://www.ama-assn.org/go/cpt"
		code = order.ImagingCode
		display = order.Name
	default:
		system = "http://snomed.info/sct"
		display = order.Name
	}

	if code != "" || display != "" {
		req.Code = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System:  system,
					Code:    code,
					Display: display,
				},
			},
			Text: display,
		}
	}

	// Set body site for imaging
	if order.BodySite != "" {
		req.BodySite = []models.CodeableConcept{
			{Text: order.BodySite},
		}
	}

	return req
}

// generateTask generates a FHIR Task
func (s *OrderSetService) generateTask(order models.Order, instance models.OrderSetInstance) *models.FHIRTask {
	return &models.FHIRTask{
		ResourceType: "Task",
		ID:           order.OrderID,
		Status:       "requested",
		Intent:       "order",
		Priority:     string(order.Priority),
		Description:  order.Name,
		For: &models.Reference{
			Reference: "Patient/" + instance.PatientID,
		},
		Encounter: &models.Reference{
			Reference: "Encounter/" + instance.EncounterID,
		},
		AuthoredOn: instance.ActivatedAt,
		Requester: &models.Reference{
			Display: instance.ActivatedBy,
		},
		Code: &models.CodeableConcept{
			Text: string(order.OrderType),
		},
		Note: []models.Annotation{
			{Text: order.Instructions},
		},
	}
}

// GetAllTemplates returns all available order set templates from the in-memory cache
func (s *OrderSetService) GetAllTemplates() []*models.OrderSetTemplate {
	s.templatesMu.RLock()
	defer s.templatesMu.RUnlock()

	templates := make([]*models.OrderSetTemplate, 0, len(s.templates))
	for _, tmpl := range s.templates {
		templates = append(templates, tmpl)
	}
	return templates
}

// GetTemplatesByCategory returns all templates matching the given category
func (s *OrderSetService) GetTemplatesByCategory(category string) []*models.OrderSetTemplate {
	s.templatesMu.RLock()
	defer s.templatesMu.RUnlock()

	templates := make([]*models.OrderSetTemplate, 0)
	for _, tmpl := range s.templates {
		if string(tmpl.Category) == category || category == "" {
			templates = append(templates, tmpl)
		}
	}
	return templates
}

// containsIgnoreCase is defined in template_loader.go
