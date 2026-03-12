// Package cdshooks provides CDS Hooks 2.0 compliant services for clinical decision support
// Implements 5 hooks: patient-view, order-select, order-sign, encounter-start, encounter-discharge
package cdshooks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kb-12-ordersets-careplans/pkg/ordersets"
)

// CDSHooksService implements CDS Hooks 2.0 specification
type CDSHooksService struct {
	templateLoader *ordersets.TemplateLoader
	version        string
}

// NewCDSHooksService creates a new CDS Hooks service
func NewCDSHooksService(loader *ordersets.TemplateLoader) *CDSHooksService {
	return &CDSHooksService{
		templateLoader: loader,
		version:        "2.0",
	}
}

// CDS Hooks 2.0 Request/Response Types

// CDSRequest represents a CDS Hooks request
type CDSRequest struct {
	Hook         string                 `json:"hook"`
	HookInstance string                 `json:"hookInstance"`
	FHIRServer   string                 `json:"fhirServer,omitempty"`
	FHIRAuth     *FHIRAuthorization     `json:"fhirAuthorization,omitempty"`
	Context      map[string]interface{} `json:"context"`
	Prefetch     map[string]interface{} `json:"prefetch,omitempty"`
}

// FHIRAuthorization contains FHIR authorization details
type FHIRAuthorization struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Subject     string `json:"subject"`
}

// CDSResponse represents a CDS Hooks response
type CDSResponse struct {
	Cards    []*Card    `json:"cards"`
	Actions  []*Action  `json:"systemActions,omitempty"`
	Prefetch []string   `json:"prefetch,omitempty"`
}

// Card represents a CDS card to display
type Card struct {
	UUID           string          `json:"uuid,omitempty"`
	Summary        string          `json:"summary"`
	Detail         string          `json:"detail,omitempty"`
	Indicator      string          `json:"indicator"` // info, warning, critical
	Source         Source          `json:"source"`
	Suggestions    []*Suggestion   `json:"suggestions,omitempty"`
	SelectionBehavior string       `json:"selectionBehavior,omitempty"` // at-most-one, any
	OverrideReasons []*OverrideReason `json:"overrideReasons,omitempty"`
	Links          []*Link         `json:"links,omitempty"`
}

// Source describes the source of a card
type Source struct {
	Label   string `json:"label"`
	URL     string `json:"url,omitempty"`
	Icon    string `json:"icon,omitempty"`
	Topic   *Topic `json:"topic,omitempty"`
}

// Topic provides additional categorization
type Topic struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
}

// Suggestion represents an actionable suggestion
type Suggestion struct {
	Label     string    `json:"label"`
	UUID      string    `json:"uuid,omitempty"`
	IsRecommended bool  `json:"isRecommended,omitempty"`
	Actions   []*Action `json:"actions,omitempty"`
}

// Action represents a FHIR action to take
type Action struct {
	Type        string      `json:"type"` // create, update, delete
	Description string      `json:"description"`
	Resource    interface{} `json:"resource,omitempty"`
	ResourceID  string      `json:"resourceId,omitempty"`
}

// OverrideReason explains why a card might be overridden
type OverrideReason struct {
	Code    string `json:"code"`
	System  string `json:"system,omitempty"`
	Display string `json:"display"`
}

// Link provides external reference
type Link struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	Type     string `json:"type,omitempty"` // absolute, smart
	AppContext string `json:"appContext,omitempty"`
}

// ServiceDefinition describes a CDS service
type ServiceDefinition struct {
	Hook        string              `json:"hook"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	ID          string              `json:"id"`
	Prefetch    map[string]string   `json:"prefetch,omitempty"`
	UsageRequirements string        `json:"usageRequirements,omitempty"`
}

// DiscoveryResponse contains available services
type DiscoveryResponse struct {
	Services []*ServiceDefinition `json:"services"`
}

// GetDiscovery returns the CDS Hooks discovery document
func (s *CDSHooksService) GetDiscovery() *DiscoveryResponse {
	return &DiscoveryResponse{
		Services: []*ServiceDefinition{
			{
				Hook:        "patient-view",
				Title:       "Order Set Recommendations on Patient View",
				Description: "Suggests relevant order sets and care plans based on patient conditions",
				ID:          "kb12-patient-view",
				Prefetch: map[string]string{
					"patient":     "Patient/{{context.patientId}}",
					"conditions":  "Condition?patient={{context.patientId}}&clinical-status=active",
					"medications": "MedicationRequest?patient={{context.patientId}}&status=active",
					"encounters":  "Encounter?patient={{context.patientId}}&status=in-progress",
				},
			},
			{
				Hook:        "order-select",
				Title:       "Order Set Suggestions During Order Entry",
				Description: "Recommends order sets related to the orders being entered",
				ID:          "kb12-order-select",
				Prefetch: map[string]string{
					"patient":    "Patient/{{context.patientId}}",
					"conditions": "Condition?patient={{context.patientId}}&clinical-status=active",
				},
			},
			{
				Hook:        "order-sign",
				Title:       "Final Order Set Check Before Signing",
				Description: "Validates orders against clinical protocols before signing",
				ID:          "kb12-order-sign",
				Prefetch: map[string]string{
					"patient":    "Patient/{{context.patientId}}",
					"conditions": "Condition?patient={{context.patientId}}&clinical-status=active",
				},
			},
			{
				Hook:        "encounter-start",
				Title:       "Admission Order Set Recommendations",
				Description: "Suggests appropriate admission order sets based on reason for admission",
				ID:          "kb12-encounter-start",
				Prefetch: map[string]string{
					"patient":     "Patient/{{context.patientId}}",
					"encounter":   "Encounter/{{context.encounterId}}",
					"conditions":  "Condition?patient={{context.patientId}}&clinical-status=active",
					"allergies":   "AllergyIntolerance?patient={{context.patientId}}",
				},
			},
			{
				Hook:        "encounter-discharge",
				Title:       "Discharge Planning Recommendations",
				Description: "Suggests discharge order sets and care plan activations",
				ID:          "kb12-encounter-discharge",
				Prefetch: map[string]string{
					"patient":       "Patient/{{context.patientId}}",
					"encounter":     "Encounter/{{context.encounterId}}",
					"conditions":    "Condition?patient={{context.patientId}}&clinical-status=active",
					"procedures":    "Procedure?patient={{context.patientId}}&encounter={{context.encounterId}}",
					"carePlans":     "CarePlan?patient={{context.patientId}}&status=active",
				},
			},
		},
	}
}

// ProcessHook routes to the appropriate hook handler
func (s *CDSHooksService) ProcessHook(ctx context.Context, hookID string, req *CDSRequest) (*CDSResponse, error) {
	switch hookID {
	case "kb12-patient-view":
		return s.handlePatientView(ctx, req)
	case "kb12-order-select":
		return s.handleOrderSelect(ctx, req)
	case "kb12-order-sign":
		return s.handleOrderSign(ctx, req)
	case "kb12-encounter-start":
		return s.handleEncounterStart(ctx, req)
	case "kb12-encounter-discharge":
		return s.handleEncounterDischarge(ctx, req)
	default:
		return nil, fmt.Errorf("unknown hook: %s", hookID)
	}
}

// handlePatientView processes the patient-view hook
// Suggests order sets and care plans based on active conditions
func (s *CDSHooksService) handlePatientView(ctx context.Context, req *CDSRequest) (*CDSResponse, error) {
	response := &CDSResponse{
		Cards: make([]*Card, 0),
	}

	// Extract patient conditions from prefetch
	conditions := s.extractConditions(req.Prefetch)

	// Find relevant order sets and care plans
	suggestions := s.findRelevantTemplates(conditions)

	if len(suggestions) == 0 {
		return response, nil
	}

	// Create cards for suggestions
	for _, suggestion := range suggestions {
		card := &Card{
			UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
			Summary:   fmt.Sprintf("Recommended: %s", suggestion.Name),
			Detail:    suggestion.Description,
			Indicator: "info",
			Source: Source{
				Label: "KB-12 Order Sets & Care Plans",
				URL:   "https://cardiofit.health/kb-12",
				Topic: &Topic{
					Code:    suggestion.Category,
					System:  "http://cardiofit.health/kb-12/categories",
					Display: suggestion.Category,
				},
			},
			SelectionBehavior: "any",
			Suggestions: []*Suggestion{
				{
					Label:         fmt.Sprintf("Apply %s", suggestion.Name),
					IsRecommended: true,
					Actions: []*Action{
						{
							Type:        "create",
							Description: fmt.Sprintf("Create orders from %s", suggestion.Name),
							Resource:    s.generateOrderSetFHIR(suggestion),
						},
					},
				},
			},
			Links: []*Link{
				{
					Label: "View Template Details",
					URL:   fmt.Sprintf("/kb-12/templates/%s", suggestion.TemplateID),
					Type:  "absolute",
				},
			},
		}
		response.Cards = append(response.Cards, card)
	}

	return response, nil
}

// handleOrderSelect processes the order-select hook
// Recommends order sets related to selected orders
func (s *CDSHooksService) handleOrderSelect(ctx context.Context, req *CDSRequest) (*CDSResponse, error) {
	response := &CDSResponse{
		Cards: make([]*Card, 0),
	}

	// Extract selected orders from context
	draftOrders := s.extractDraftOrders(req.Context)

	// Analyze orders for patterns
	patterns := s.analyzeOrderPatterns(draftOrders)

	// Find complementary order sets
	for _, pattern := range patterns {
		if orderSet := s.findOrderSetForPattern(pattern); orderSet != nil {
			card := &Card{
				UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
				Summary:   fmt.Sprintf("Complete protocol: %s", orderSet.Name),
				Detail:    fmt.Sprintf("The orders you're entering are part of the %s protocol. Consider applying the full order set.", orderSet.Name),
				Indicator: "info",
				Source: Source{
					Label: "KB-12 Protocol Matcher",
				},
				Suggestions: []*Suggestion{
					{
						Label:         "Apply Complete Protocol",
						IsRecommended: true,
						Actions: []*Action{
							{
								Type:        "create",
								Description: fmt.Sprintf("Apply remaining orders from %s", orderSet.Name),
								Resource:    s.generateRemainingOrders(orderSet, draftOrders),
							},
						},
					},
				},
			}
			response.Cards = append(response.Cards, card)
		}
	}

	return response, nil
}

// handleOrderSign processes the order-sign hook
// Final validation before signing orders
func (s *CDSHooksService) handleOrderSign(ctx context.Context, req *CDSRequest) (*CDSResponse, error) {
	response := &CDSResponse{
		Cards: make([]*Card, 0),
	}

	// Extract orders being signed
	draftOrders := s.extractDraftOrders(req.Context)
	conditions := s.extractConditions(req.Prefetch)

	// Check for missing protocol elements
	missing := s.checkProtocolCompleteness(draftOrders, conditions)

	for _, m := range missing {
		severity := "warning"
		if m.Critical {
			severity = "critical"
		}

		card := &Card{
			UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
			Summary:   m.Message,
			Detail:    m.Detail,
			Indicator: severity,
			Source: Source{
				Label: "KB-12 Protocol Validator",
				Topic: &Topic{
					Code:    m.Protocol,
					System:  "http://cardiofit.health/kb-12/protocols",
					Display: m.Protocol,
				},
			},
			Suggestions: []*Suggestion{
				{
					Label:         "Add Missing Items",
					IsRecommended: true,
					Actions:       m.Actions,
				},
			},
			OverrideReasons: []*OverrideReason{
				{Code: "contraindicated", Display: "Contraindicated for this patient"},
				{Code: "already-ordered", Display: "Already ordered separately"},
				{Code: "clinician-judgment", Display: "Clinical judgment - not indicated"},
			},
		}
		response.Cards = append(response.Cards, card)
	}

	return response, nil
}

// handleEncounterStart processes the encounter-start hook
// Suggests admission order sets
func (s *CDSHooksService) handleEncounterStart(ctx context.Context, req *CDSRequest) (*CDSResponse, error) {
	response := &CDSResponse{
		Cards: make([]*Card, 0),
	}

	// Extract encounter and admission reason
	encounterData := s.extractEncounterData(req.Prefetch)
	conditions := s.extractConditions(req.Prefetch)

	// Determine encounter type
	encounterType := s.determineEncounterType(encounterData)

	// Get admission order set recommendations
	recommendations := s.getAdmissionRecommendations(encounterType, conditions)

	if len(recommendations) == 0 {
		// Default general admission
		recommendations = append(recommendations, &TemplateSuggestion{
			TemplateID:  "OS-ADM-001",
			Name:        "General Medical Admission",
			Description: "Standard admission orders for medical patients",
			Category:    "admission",
			Priority:    3,
		})
	}

	for i, rec := range recommendations {
		isRecommended := i == 0 // First one is recommended

		card := &Card{
			UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
			Summary:   fmt.Sprintf("Admission Order Set: %s", rec.Name),
			Detail:    rec.Description,
			Indicator: s.priorityToIndicator(rec.Priority),
			Source: Source{
				Label: "KB-12 Admission Orders",
				Topic: &Topic{
					Code:    "admission",
					System:  "http://cardiofit.health/kb-12/categories",
					Display: "Admission Order Sets",
				},
			},
			SelectionBehavior: "at-most-one",
			Suggestions: []*Suggestion{
				{
					Label:         fmt.Sprintf("Apply %s", rec.Name),
					IsRecommended: isRecommended,
					Actions: []*Action{
						{
							Type:        "create",
							Description: fmt.Sprintf("Create admission orders from %s", rec.Name),
							Resource:    s.generateOrderSetFHIRByID(rec.TemplateID),
						},
					},
				},
			},
			Links: []*Link{
				{
					Label: "Preview Orders",
					URL:   fmt.Sprintf("/kb-12/templates/%s/preview", rec.TemplateID),
					Type:  "smart",
				},
			},
		}
		response.Cards = append(response.Cards, card)
	}

	// Add emergency protocol cards if indicated
	if emergencyProtocol := s.detectEmergencyCondition(conditions); emergencyProtocol != nil {
		card := &Card{
			UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
			Summary:   fmt.Sprintf("⚠️ EMERGENCY: %s Protocol Indicated", emergencyProtocol.Name),
			Detail:    fmt.Sprintf("Patient presentation suggests %s. Time-critical interventions required.", emergencyProtocol.Name),
			Indicator: "critical",
			Source: Source{
				Label: "KB-12 Emergency Protocols",
				Icon:  "🚨",
			},
			Suggestions: []*Suggestion{
				{
					Label:         fmt.Sprintf("ACTIVATE %s Protocol", emergencyProtocol.Name),
					IsRecommended: true,
					Actions: []*Action{
						{
							Type:        "create",
							Description: fmt.Sprintf("Activate %s emergency protocol with time tracking", emergencyProtocol.Name),
							Resource:    s.generateEmergencyProtocolFHIR(emergencyProtocol),
						},
					},
				},
			},
		}
		response.Cards = append(response.Cards, card)
	}

	return response, nil
}

// handleEncounterDischarge processes the encounter-discharge hook
// Suggests discharge orders and care plan activations
func (s *CDSHooksService) handleEncounterDischarge(ctx context.Context, req *CDSRequest) (*CDSResponse, error) {
	response := &CDSResponse{
		Cards: make([]*Card, 0),
	}

	// Extract encounter data and diagnoses
	conditions := s.extractConditions(req.Prefetch)
	procedures := s.extractProcedures(req.Prefetch)
	activeCarePlans := s.extractCarePlans(req.Prefetch)

	// Recommend discharge medication reconciliation
	card := &Card{
		UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
		Summary:   "Discharge Medication Reconciliation Required",
		Detail:    "Complete medication reconciliation before discharge to ensure safe transitions of care.",
		Indicator: "warning",
		Source: Source{
			Label: "KB-12 Discharge Planning",
		},
		Suggestions: []*Suggestion{
			{
				Label:         "Open Medication Reconciliation",
				IsRecommended: true,
				Actions: []*Action{
					{
						Type:        "create",
						Description: "Initiate discharge medication reconciliation workflow",
					},
				},
			},
		},
	}
	response.Cards = append(response.Cards, card)

	// Recommend care plans for chronic conditions
	chronicConditions := s.filterChronicConditions(conditions)
	for _, condition := range chronicConditions {
		// Check if care plan already exists
		if s.hasActiveCarePlan(condition, activeCarePlans) {
			continue
		}

		carePlan := s.findCarePlanForCondition(condition)
		if carePlan != nil {
			card := &Card{
				UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
				Summary:   fmt.Sprintf("Activate %s Care Plan", carePlan.Name),
				Detail:    fmt.Sprintf("Patient has %s. Consider activating outpatient care plan for chronic disease management.", condition.Display),
				Indicator: "info",
				Source: Source{
					Label: "KB-12 Care Plans",
					Topic: &Topic{
						Code:    carePlan.Category,
						System:  "http://cardiofit.health/kb-12/categories",
						Display: carePlan.Category,
					},
				},
				Suggestions: []*Suggestion{
					{
						Label:         fmt.Sprintf("Activate %s Care Plan", carePlan.Name),
						IsRecommended: true,
						Actions: []*Action{
							{
								Type:        "create",
								Description: fmt.Sprintf("Create care plan for %s management", condition.Display),
								Resource:    s.generateCarePlanFHIR(carePlan),
							},
						},
					},
				},
			}
			response.Cards = append(response.Cards, card)
		}
	}

	// Check for post-procedure instructions
	for _, procedure := range procedures {
		if instructions := s.getPostProcedureInstructions(procedure); instructions != nil {
			card := &Card{
				UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
				Summary:   fmt.Sprintf("Post-%s Discharge Instructions", procedure.Display),
				Detail:    fmt.Sprintf("Provide patient with post-procedural care instructions for %s.", procedure.Display),
				Indicator: "info",
				Source: Source{
					Label: "KB-12 Procedure Follow-up",
				},
				Suggestions: []*Suggestion{
					{
						Label:         "Generate Discharge Instructions",
						IsRecommended: true,
					},
				},
				Links: []*Link{
					{
						Label: "View Discharge Template",
						URL:   fmt.Sprintf("/kb-12/discharge/%s", instructions.TemplateID),
						Type:  "absolute",
					},
				},
			}
			response.Cards = append(response.Cards, card)
		}
	}

	// Follow-up appointment recommendations
	card = &Card{
		UUID:      fmt.Sprintf("card-%d", time.Now().UnixNano()),
		Summary:   "Schedule Follow-up Appointments",
		Detail:    "Based on diagnoses and procedures, the following follow-up appointments are recommended.",
		Indicator: "info",
		Source: Source{
			Label: "KB-12 Care Coordination",
		},
		Suggestions: s.generateFollowUpSuggestions(conditions, procedures),
	}
	if len(card.Suggestions) > 0 {
		response.Cards = append(response.Cards, card)
	}

	return response, nil
}

// Helper types and functions

// TemplateSuggestion represents a template recommendation
type TemplateSuggestion struct {
	TemplateID  string `json:"template_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Priority    int    `json:"priority"`
}

// ConditionData represents extracted condition information
type ConditionData struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
	Chronic bool   `json:"chronic"`
}

// ProcedureData represents extracted procedure information
type ProcedureData struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
}

// EncounterData represents extracted encounter information
type EncounterData struct {
	Class            string `json:"class"`
	Type             string `json:"type"`
	ReasonCode       string `json:"reason_code"`
	ReasonDisplay    string `json:"reason_display"`
	AdmitSource      string `json:"admit_source"`
	ServiceType      string `json:"service_type"`
}

// MissingProtocolItem represents a missing protocol element
type MissingProtocolItem struct {
	Message  string    `json:"message"`
	Detail   string    `json:"detail"`
	Protocol string    `json:"protocol"`
	Critical bool      `json:"critical"`
	Actions  []*Action `json:"actions"`
}

// PostProcedureInstructions contains procedure-specific discharge info
type PostProcedureInstructions struct {
	TemplateID   string   `json:"template_id"`
	ProcedureCode string  `json:"procedure_code"`
	Instructions []string `json:"instructions"`
}

// extractConditions parses conditions from prefetch data
func (s *CDSHooksService) extractConditions(prefetch map[string]interface{}) []ConditionData {
	conditions := make([]ConditionData, 0)

	if conditionsData, ok := prefetch["conditions"]; ok {
		if bundle, ok := conditionsData.(map[string]interface{}); ok {
			if entries, ok := bundle["entry"].([]interface{}); ok {
				for _, entry := range entries {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
							condition := ConditionData{}
							if code, ok := resource["code"].(map[string]interface{}); ok {
								if codings, ok := code["coding"].([]interface{}); ok && len(codings) > 0 {
									if coding, ok := codings[0].(map[string]interface{}); ok {
										condition.Code, _ = coding["code"].(string)
										condition.System, _ = coding["system"].(string)
										condition.Display, _ = coding["display"].(string)
									}
								}
							}
							// Check if chronic based on category
							if categories, ok := resource["category"].([]interface{}); ok {
								for _, cat := range categories {
									if catMap, ok := cat.(map[string]interface{}); ok {
										if codings, ok := catMap["coding"].([]interface{}); ok {
											for _, coding := range codings {
												if codingMap, ok := coding.(map[string]interface{}); ok {
													if code, _ := codingMap["code"].(string); code == "chronic" {
														condition.Chronic = true
													}
												}
											}
										}
									}
								}
							}
							conditions = append(conditions, condition)
						}
					}
				}
			}
		}
	}

	return conditions
}

// extractDraftOrders parses draft orders from context
func (s *CDSHooksService) extractDraftOrders(context map[string]interface{}) []map[string]interface{} {
	orders := make([]map[string]interface{}, 0)

	if draftOrders, ok := context["draftOrders"]; ok {
		if bundle, ok := draftOrders.(map[string]interface{}); ok {
			if entries, ok := bundle["entry"].([]interface{}); ok {
				for _, entry := range entries {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
							orders = append(orders, resource)
						}
					}
				}
			}
		}
	}

	return orders
}

// extractEncounterData parses encounter from prefetch
func (s *CDSHooksService) extractEncounterData(prefetch map[string]interface{}) *EncounterData {
	if encounterData, ok := prefetch["encounter"]; ok {
		if encounter, ok := encounterData.(map[string]interface{}); ok {
			data := &EncounterData{}
			if class, ok := encounter["class"].(map[string]interface{}); ok {
				data.Class, _ = class["code"].(string)
			}
			if types, ok := encounter["type"].([]interface{}); ok && len(types) > 0 {
				if typeMap, ok := types[0].(map[string]interface{}); ok {
					if codings, ok := typeMap["coding"].([]interface{}); ok && len(codings) > 0 {
						if coding, ok := codings[0].(map[string]interface{}); ok {
							data.Type, _ = coding["code"].(string)
						}
					}
				}
			}
			if reasons, ok := encounter["reasonCode"].([]interface{}); ok && len(reasons) > 0 {
				if reasonMap, ok := reasons[0].(map[string]interface{}); ok {
					if codings, ok := reasonMap["coding"].([]interface{}); ok && len(codings) > 0 {
						if coding, ok := codings[0].(map[string]interface{}); ok {
							data.ReasonCode, _ = coding["code"].(string)
							data.ReasonDisplay, _ = coding["display"].(string)
						}
					}
				}
			}
			return data
		}
	}
	return nil
}

// extractProcedures parses procedures from prefetch
func (s *CDSHooksService) extractProcedures(prefetch map[string]interface{}) []ProcedureData {
	procedures := make([]ProcedureData, 0)

	if procData, ok := prefetch["procedures"]; ok {
		if bundle, ok := procData.(map[string]interface{}); ok {
			if entries, ok := bundle["entry"].([]interface{}); ok {
				for _, entry := range entries {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
							proc := ProcedureData{}
							if code, ok := resource["code"].(map[string]interface{}); ok {
								if codings, ok := code["coding"].([]interface{}); ok && len(codings) > 0 {
									if coding, ok := codings[0].(map[string]interface{}); ok {
										proc.Code, _ = coding["code"].(string)
										proc.System, _ = coding["system"].(string)
										proc.Display, _ = coding["display"].(string)
									}
								}
							}
							procedures = append(procedures, proc)
						}
					}
				}
			}
		}
	}

	return procedures
}

// extractCarePlans parses active care plans from prefetch
func (s *CDSHooksService) extractCarePlans(prefetch map[string]interface{}) []map[string]interface{} {
	carePlans := make([]map[string]interface{}, 0)

	if cpData, ok := prefetch["carePlans"]; ok {
		if bundle, ok := cpData.(map[string]interface{}); ok {
			if entries, ok := bundle["entry"].([]interface{}); ok {
				for _, entry := range entries {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
							carePlans = append(carePlans, resource)
						}
					}
				}
			}
		}
	}

	return carePlans
}

// findRelevantTemplates finds templates matching patient conditions
func (s *CDSHooksService) findRelevantTemplates(conditions []ConditionData) []*TemplateSuggestion {
	suggestions := make([]*TemplateSuggestion, 0)

	// Map of condition codes to relevant templates
	conditionTemplateMap := map[string][]string{
		"I10":    {"CP-CV-001"},                   // Hypertension -> HTN Care Plan
		"E11":    {"CP-MET-001"},                  // Type 2 Diabetes -> Diabetes Care Plan
		"E11.9":  {"CP-MET-001"},                  // T2DM unspecified
		"I50":    {"CP-CV-002", "OS-ADM-002"},     // Heart Failure -> HF Care Plan + CHF Admission
		"I50.9":  {"CP-CV-002", "OS-ADM-002"},
		"J44":    {"CP-RESP-001", "OS-ADM-003"},   // COPD -> COPD Care Plan + COPD Admission
		"J44.1":  {"CP-RESP-001", "OS-ADM-003"},
		"J45":    {"CP-RESP-002"},                 // Asthma -> Asthma Care Plan
		"I25":    {"CP-CV-003"},                   // CAD -> CAD Care Plan
		"I48":    {"CP-CV-004"},                   // AFib -> AFib Care Plan
		"N18":    {"CP-REN-001"},                  // CKD -> CKD Care Plan
		"F32":    {"CP-MH-001"},                   // Depression -> Depression Care Plan
		"M81":    {"CP-MSK-001"},                  // Osteoporosis -> Osteoporosis Care Plan
		"E03":    {"CP-MET-003"},                  // Hypothyroidism
		"E66":    {"CP-MET-002"},                  // Obesity
	}

	for _, condition := range conditions {
		// Check exact match
		if templateIDs, ok := conditionTemplateMap[condition.Code]; ok {
			for _, templateID := range templateIDs {
				if template := s.getTemplateInfo(templateID); template != nil {
					suggestions = append(suggestions, template)
				}
			}
		}

		// Check prefix match (e.g., I50.x matches I50)
		for code, templateIDs := range conditionTemplateMap {
			if len(condition.Code) > len(code) && condition.Code[:len(code)] == code {
				for _, templateID := range templateIDs {
					if template := s.getTemplateInfo(templateID); template != nil {
						// Check for duplicates
						exists := false
						for _, s := range suggestions {
							if s.TemplateID == templateID {
								exists = true
								break
							}
						}
						if !exists {
							suggestions = append(suggestions, template)
						}
					}
				}
			}
		}
	}

	return suggestions
}

// getTemplateInfo retrieves template metadata
func (s *CDSHooksService) getTemplateInfo(templateID string) *TemplateSuggestion {
	// Use the template loader to get template info
	if s.templateLoader != nil {
		ctx := context.Background()
		if template, err := s.templateLoader.GetTemplate(ctx, templateID); err == nil {
			return &TemplateSuggestion{
				TemplateID:  template.TemplateID,
				Name:        template.Name,
				Description: template.Description,
				Category:    string(template.Category),
				Priority:    2,
			}
		}
	}

	// Fallback to hardcoded metadata
	templates := map[string]*TemplateSuggestion{
		"CP-CV-001":  {TemplateID: "CP-CV-001", Name: "Hypertension Care Plan", Description: "ACC/AHA guideline-based HTN management", Category: "cardiovascular", Priority: 2},
		"CP-CV-002":  {TemplateID: "CP-CV-002", Name: "Heart Failure Care Plan", Description: "GDMT-based HFrEF management", Category: "cardiovascular", Priority: 1},
		"CP-MET-001": {TemplateID: "CP-MET-001", Name: "Diabetes Type 2 Care Plan", Description: "ADA Standards of Care", Category: "metabolic", Priority: 2},
		"CP-RESP-001": {TemplateID: "CP-RESP-001", Name: "COPD Care Plan", Description: "GOLD guideline-based COPD management", Category: "respiratory", Priority: 2},
		"OS-ADM-002": {TemplateID: "OS-ADM-002", Name: "CHF Exacerbation Admission", Description: "Heart failure admission order set", Category: "admission", Priority: 1},
		"OS-ADM-003": {TemplateID: "OS-ADM-003", Name: "COPD Exacerbation Admission", Description: "COPD admission order set", Category: "admission", Priority: 1},
	}

	return templates[templateID]
}

// determineEncounterType determines the type of encounter
func (s *CDSHooksService) determineEncounterType(encounter *EncounterData) string {
	if encounter == nil {
		return "general"
	}

	// Map encounter class to type
	switch encounter.Class {
	case "EMER":
		return "emergency"
	case "IMP":
		return "inpatient"
	case "OBSENC":
		return "observation"
	case "SS":
		return "short_stay"
	default:
		return "general"
	}
}

// getAdmissionRecommendations returns admission order set recommendations
func (s *CDSHooksService) getAdmissionRecommendations(encounterType string, conditions []ConditionData) []*TemplateSuggestion {
	recommendations := make([]*TemplateSuggestion, 0)

	// Priority condition matching
	for _, condition := range conditions {
		switch {
		case s.matchesCode(condition.Code, "A41", "R65.2"): // Sepsis
			recommendations = append(recommendations, &TemplateSuggestion{
				TemplateID:  "OS-ADM-004",
				Name:        "Sepsis/Septic Shock Admission",
				Description: "SEP-1 compliant sepsis bundle",
				Category:    "admission",
				Priority:    1,
			})
		case s.matchesCode(condition.Code, "I21"): // STEMI
			recommendations = append(recommendations, &TemplateSuggestion{
				TemplateID:  "OS-ADM-005",
				Name:        "Acute MI (STEMI) Admission",
				Description: "Door-to-balloon time tracking",
				Category:    "admission",
				Priority:    1,
			})
		case s.matchesCode(condition.Code, "I50"): // Heart Failure
			recommendations = append(recommendations, &TemplateSuggestion{
				TemplateID:  "OS-ADM-002",
				Name:        "CHF Exacerbation Admission",
				Description: "Diuresis and GDMT optimization",
				Category:    "admission",
				Priority:    1,
			})
		case s.matchesCode(condition.Code, "J44"): // COPD
			recommendations = append(recommendations, &TemplateSuggestion{
				TemplateID:  "OS-ADM-003",
				Name:        "COPD Exacerbation Admission",
				Description: "Bronchodilators, steroids, antibiotics",
				Category:    "admission",
				Priority:    1,
			})
		case s.matchesCode(condition.Code, "J18", "J15", "J13"): // Pneumonia
			recommendations = append(recommendations, &TemplateSuggestion{
				TemplateID:  "OS-ADM-006",
				Name:        "Pneumonia Admission",
				Description: "CAP/HAP antimicrobial therapy",
				Category:    "admission",
				Priority:    1,
			})
		}
	}

	return recommendations
}

// matchesCode checks if a code matches any of the given prefixes
func (s *CDSHooksService) matchesCode(code string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(code) >= len(prefix) && code[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// detectEmergencyCondition checks for emergency conditions
func (s *CDSHooksService) detectEmergencyCondition(conditions []ConditionData) *TemplateSuggestion {
	for _, condition := range conditions {
		switch {
		case s.matchesCode(condition.Code, "I46"): // Cardiac arrest
			return &TemplateSuggestion{
				TemplateID:  "OS-EM-001",
				Name:        "Code Blue (ACLS)",
				Description: "Cardiac arrest resuscitation protocol",
				Category:    "emergency",
				Priority:    1,
			}
		case condition.Display == "Malignant hyperthermia" || condition.Code == "T88.3":
			return &TemplateSuggestion{
				TemplateID:  "OS-EM-003",
				Name:        "Malignant Hyperthermia",
				Description: "MH crisis management",
				Category:    "emergency",
				Priority:    1,
			}
		case condition.Display == "Anaphylaxis" || s.matchesCode(condition.Code, "T78.2"):
			return &TemplateSuggestion{
				TemplateID:  "OS-EM-004",
				Name:        "Anaphylaxis",
				Description: "Anaphylaxis resuscitation protocol",
				Category:    "emergency",
				Priority:    1,
			}
		}
	}
	return nil
}

// analyzeOrderPatterns analyzes draft orders for patterns
func (s *CDSHooksService) analyzeOrderPatterns(orders []map[string]interface{}) []string {
	patterns := make([]string, 0)

	// Look for medication patterns
	hasBBlocker := false
	hasACEi := false
	hasDiuretic := false
	hasSGLT2i := false

	for _, order := range orders {
		if medCode := s.extractMedicationCode(order); medCode != "" {
			switch {
			case s.isBetaBlocker(medCode):
				hasBBlocker = true
			case s.isACEiARB(medCode):
				hasACEi = true
			case s.isDiuretic(medCode):
				hasDiuretic = true
			case s.isSGLT2i(medCode):
				hasSGLT2i = true
			}
		}
	}

	// Detect HF GDMT pattern
	if (hasBBlocker || hasACEi) && hasDiuretic {
		patterns = append(patterns, "heart_failure_gdmt")
	}

	if hasACEi && hasSGLT2i {
		patterns = append(patterns, "diabetic_ckd")
	}

	return patterns
}

// Helper medication classification functions
func (s *CDSHooksService) extractMedicationCode(order map[string]interface{}) string {
	if medCodeable, ok := order["medicationCodeableConcept"].(map[string]interface{}); ok {
		if codings, ok := medCodeable["coding"].([]interface{}); ok && len(codings) > 0 {
			if coding, ok := codings[0].(map[string]interface{}); ok {
				code, _ := coding["code"].(string)
				return code
			}
		}
	}
	return ""
}

func (s *CDSHooksService) isBetaBlocker(code string) bool {
	betaBlockers := []string{"19484", "33738", "20352", "6185"} // metoprolol, carvedilol, bisoprolol, atenolol
	for _, bb := range betaBlockers {
		if code == bb {
			return true
		}
	}
	return false
}

func (s *CDSHooksService) isACEiARB(code string) bool {
	agents := []string{"29046", "52175", "35296", "83515", "321064"} // lisinopril, losartan, enalapril, valsartan, sacubitril-valsartan
	for _, a := range agents {
		if code == a {
			return true
		}
	}
	return false
}

func (s *CDSHooksService) isDiuretic(code string) bool {
	diuretics := []string{"4603", "5487", "38413"} // furosemide, bumetanide, torsemide
	for _, d := range diuretics {
		if code == d {
			return true
		}
	}
	return false
}

func (s *CDSHooksService) isSGLT2i(code string) bool {
	sglt2i := []string{"1488564", "1545653", "1373458"} // dapagliflozin, empagliflozin, canagliflozin
	for _, s := range sglt2i {
		if code == s {
			return true
		}
	}
	return false
}

// findOrderSetForPattern finds an order set matching a pattern
func (s *CDSHooksService) findOrderSetForPattern(pattern string) *TemplateSuggestion {
	patternTemplates := map[string]*TemplateSuggestion{
		"heart_failure_gdmt": {
			TemplateID:  "OS-ADM-002",
			Name:        "CHF GDMT Protocol",
			Description: "Complete GDMT optimization for heart failure",
			Category:    "admission",
			Priority:    1,
		},
		"diabetic_ckd": {
			TemplateID:  "CP-REN-001",
			Name:        "Diabetic CKD Protocol",
			Description: "KDIGO guideline-based diabetic CKD management",
			Category:    "renal",
			Priority:    1,
		},
	}
	return patternTemplates[pattern]
}

// checkProtocolCompleteness checks for missing protocol elements
func (s *CDSHooksService) checkProtocolCompleteness(orders []map[string]interface{}, conditions []ConditionData) []*MissingProtocolItem {
	missing := make([]*MissingProtocolItem, 0)

	// Check for sepsis bundle completeness
	for _, condition := range conditions {
		if s.matchesCode(condition.Code, "A41", "R65.2") {
			// Check for lactate, blood cultures, antibiotics
			hasLactate := false
			hasCultures := false
			hasAntibiotics := false

			for _, order := range orders {
				if resourceType, _ := order["resourceType"].(string); resourceType == "ServiceRequest" {
					if code := s.extractServiceRequestCode(order); code != "" {
						if code == "2524-7" || code == "32693-4" { // Lactate LOINC codes
							hasLactate = true
						}
						if code == "600-7" { // Blood culture
							hasCultures = true
						}
					}
				}
				if resourceType, _ := order["resourceType"].(string); resourceType == "MedicationRequest" {
					hasAntibiotics = true // Simplified - would check for actual antibiotics
				}
			}

			if !hasLactate {
				missing = append(missing, &MissingProtocolItem{
					Message:  "Sepsis Bundle: Lactate Missing",
					Detail:   "SEP-1 requires lactate measurement within 3 hours",
					Protocol: "SEP-1",
					Critical: true,
					Actions: []*Action{
						{
							Type:        "create",
							Description: "Order lactate level",
							Resource:    s.createLactateOrder(),
						},
					},
				})
			}

			if !hasCultures {
				missing = append(missing, &MissingProtocolItem{
					Message:  "Sepsis Bundle: Blood Cultures Missing",
					Detail:   "SEP-1 requires blood cultures before antibiotics",
					Protocol: "SEP-1",
					Critical: true,
					Actions: []*Action{
						{
							Type:        "create",
							Description: "Order blood cultures x2",
							Resource:    s.createBloodCultureOrder(),
						},
					},
				})
			}

			if !hasAntibiotics {
				missing = append(missing, &MissingProtocolItem{
					Message:  "Sepsis Bundle: Antibiotics Missing",
					Detail:   "SEP-1 requires broad-spectrum antibiotics within 3 hours",
					Protocol: "SEP-1",
					Critical: true,
					Actions: []*Action{
						{
							Type:        "create",
							Description: "Order broad-spectrum antibiotics",
						},
					},
				})
			}
		}
	}

	return missing
}

func (s *CDSHooksService) extractServiceRequestCode(order map[string]interface{}) string {
	if code, ok := order["code"].(map[string]interface{}); ok {
		if codings, ok := code["coding"].([]interface{}); ok && len(codings) > 0 {
			if coding, ok := codings[0].(map[string]interface{}); ok {
				c, _ := coding["code"].(string)
				return c
			}
		}
	}
	return ""
}

// filterChronicConditions filters for chronic conditions
func (s *CDSHooksService) filterChronicConditions(conditions []ConditionData) []ConditionData {
	chronic := make([]ConditionData, 0)
	for _, c := range conditions {
		if c.Chronic {
			chronic = append(chronic, c)
		}
		// Also check code patterns for common chronic conditions
		if s.matchesCode(c.Code, "E11", "I10", "I50", "J44", "N18", "I25", "I48") {
			if !c.Chronic {
				c.Chronic = true
			}
			chronic = append(chronic, c)
		}
	}
	return chronic
}

// hasActiveCarePlan checks if patient has an active care plan for condition
func (s *CDSHooksService) hasActiveCarePlan(condition ConditionData, carePlans []map[string]interface{}) bool {
	for _, cp := range carePlans {
		if addresses, ok := cp["addresses"].([]interface{}); ok {
			for _, addr := range addresses {
				if ref, ok := addr.(map[string]interface{}); ok {
					if display, _ := ref["display"].(string); display == condition.Display {
						return true
					}
				}
			}
		}
	}
	return false
}

// findCarePlanForCondition finds a care plan template for a condition
func (s *CDSHooksService) findCarePlanForCondition(condition ConditionData) *TemplateSuggestion {
	conditionCarePlanMap := map[string]string{
		"E11":  "CP-MET-001",  // Diabetes
		"I10":  "CP-CV-001",   // Hypertension
		"I50":  "CP-CV-002",   // Heart Failure
		"I25":  "CP-CV-003",   // CAD
		"I48":  "CP-CV-004",   // AFib
		"J44":  "CP-RESP-001", // COPD
		"J45":  "CP-RESP-002", // Asthma
		"N18":  "CP-REN-001",  // CKD
		"F32":  "CP-MH-001",   // Depression
		"M81":  "CP-MSK-001",  // Osteoporosis
		"E03":  "CP-MET-003",  // Hypothyroidism
		"E66":  "CP-MET-002",  // Obesity
	}

	for prefix, templateID := range conditionCarePlanMap {
		if s.matchesCode(condition.Code, prefix) {
			return s.getTemplateInfo(templateID)
		}
	}
	return nil
}

// getPostProcedureInstructions returns post-procedure discharge instructions
func (s *CDSHooksService) getPostProcedureInstructions(procedure ProcedureData) *PostProcedureInstructions {
	instructionsMap := map[string]*PostProcedureInstructions{
		"34068001": { // Cardiac catheterization
			TemplateID:    "DIS-PROC-001",
			ProcedureCode: "34068001",
			Instructions:  []string{"Bed rest 4-6 hours", "Watch puncture site", "No heavy lifting 48 hours"},
		},
		"40617009": { // Colonoscopy
			TemplateID:    "DIS-PROC-002",
			ProcedureCode: "40617009",
			Instructions:  []string{"Clear liquids first", "Resume normal diet", "Call if bleeding"},
		},
	}

	return instructionsMap[procedure.Code]
}

// generateFollowUpSuggestions creates follow-up appointment suggestions
func (s *CDSHooksService) generateFollowUpSuggestions(conditions []ConditionData, procedures []ProcedureData) []*Suggestion {
	suggestions := make([]*Suggestion, 0)

	// Primary care follow-up
	suggestions = append(suggestions, &Suggestion{
		Label:         "Primary Care Follow-up (7-14 days)",
		IsRecommended: true,
	})

	// Specialty follow-ups based on conditions
	for _, condition := range conditions {
		switch {
		case s.matchesCode(condition.Code, "I50"):
			suggestions = append(suggestions, &Suggestion{
				Label: "Cardiology Follow-up (1-2 weeks for HF)",
			})
		case s.matchesCode(condition.Code, "J44"):
			suggestions = append(suggestions, &Suggestion{
				Label: "Pulmonology Follow-up (2-4 weeks for COPD)",
			})
		case s.matchesCode(condition.Code, "N18"):
			suggestions = append(suggestions, &Suggestion{
				Label: "Nephrology Follow-up (2-4 weeks for CKD)",
			})
		}
	}

	return suggestions
}

// priorityToIndicator converts priority to CDS indicator
func (s *CDSHooksService) priorityToIndicator(priority int) string {
	switch priority {
	case 1:
		return "critical"
	case 2:
		return "warning"
	default:
		return "info"
	}
}

// FHIR Resource Generation Functions

func (s *CDSHooksService) generateOrderSetFHIR(suggestion *TemplateSuggestion) interface{} {
	return s.generateOrderSetFHIRByID(suggestion.TemplateID)
}

func (s *CDSHooksService) generateOrderSetFHIRByID(templateID string) interface{} {
	// Generate a RequestGroup FHIR resource for the order set
	return map[string]interface{}{
		"resourceType": "RequestGroup",
		"id":           fmt.Sprintf("orderset-%s", templateID),
		"status":       "draft",
		"intent":       "proposal",
		"instantiatesCanonical": []string{
			fmt.Sprintf("http://cardiofit.health/kb-12/PlanDefinition/%s", templateID),
		},
		"author": map[string]interface{}{
			"display": "KB-12 Order Sets & Care Plans",
		},
	}
}

func (s *CDSHooksService) generateEmergencyProtocolFHIR(protocol *TemplateSuggestion) interface{} {
	return map[string]interface{}{
		"resourceType": "RequestGroup",
		"id":           fmt.Sprintf("emergency-%s", protocol.TemplateID),
		"status":       "draft",
		"intent":       "order",
		"priority":     "stat",
		"instantiatesCanonical": []string{
			fmt.Sprintf("http://cardiofit.health/kb-12/PlanDefinition/%s", protocol.TemplateID),
		},
		"author": map[string]interface{}{
			"display": "KB-12 Emergency Protocols",
		},
	}
}

func (s *CDSHooksService) generateCarePlanFHIR(template *TemplateSuggestion) interface{} {
	return map[string]interface{}{
		"resourceType": "CarePlan",
		"id":           fmt.Sprintf("careplan-%s", template.TemplateID),
		"status":       "draft",
		"intent":       "plan",
		"title":        template.Name,
		"description":  template.Description,
		"instantiatesCanonical": []string{
			fmt.Sprintf("http://cardiofit.health/kb-12/PlanDefinition/%s", template.TemplateID),
		},
	}
}

func (s *CDSHooksService) generateRemainingOrders(orderSet *TemplateSuggestion, existingOrders []map[string]interface{}) interface{} {
	// In practice, this would compare template orders with existing orders
	// and generate only the missing ones
	return map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "collection",
		"entry":        []interface{}{}, // Would contain remaining orders
	}
}

func (s *CDSHooksService) createLactateOrder() interface{} {
	return map[string]interface{}{
		"resourceType": "ServiceRequest",
		"status":       "draft",
		"intent":       "order",
		"priority":     "stat",
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    "2524-7",
					"display": "Lactate [Moles/volume] in Serum or Plasma",
				},
			},
		},
	}
}

func (s *CDSHooksService) createBloodCultureOrder() interface{} {
	return map[string]interface{}{
		"resourceType": "ServiceRequest",
		"status":       "draft",
		"intent":       "order",
		"priority":     "stat",
		"code": map[string]interface{}{
			"coding": []map[string]interface{}{
				{
					"system":  "http://loinc.org",
					"code":    "600-7",
					"display": "Bacteria identified in Blood by Culture",
				},
			},
		},
		"quantityQuantity": map[string]interface{}{
			"value": 2,
			"unit":  "sets",
		},
	}
}

// FeedbackHandler processes feedback on CDS cards
type FeedbackHandler struct {
	feedbackStore sync.Map
}

// CardFeedback represents feedback on a CDS card
type CardFeedback struct {
	CardID      string    `json:"card_id"`
	Outcome     string    `json:"outcome"` // accepted, overridden, dismissed
	OverrideReason string `json:"override_reason,omitempty"`
	Comments    string    `json:"comments,omitempty"`
	UserID      string    `json:"user_id"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewFeedbackHandler creates a new feedback handler
func NewFeedbackHandler() *FeedbackHandler {
	return &FeedbackHandler{}
}

// RecordFeedback records feedback on a CDS card
func (h *FeedbackHandler) RecordFeedback(feedback *CardFeedback) error {
	feedback.Timestamp = time.Now()
	h.feedbackStore.Store(feedback.CardID, feedback)
	return nil
}

// GetFeedback retrieves feedback for a card
func (h *FeedbackHandler) GetFeedback(cardID string) (*CardFeedback, bool) {
	if val, ok := h.feedbackStore.Load(cardID); ok {
		return val.(*CardFeedback), true
	}
	return nil, false
}

// FeedbackStats provides statistics on card feedback
type FeedbackStats struct {
	TotalCards     int            `json:"total_cards"`
	Accepted       int            `json:"accepted"`
	Overridden     int            `json:"overridden"`
	Dismissed      int            `json:"dismissed"`
	OverrideReasons map[string]int `json:"override_reasons"`
}

// GetStats returns feedback statistics
func (h *FeedbackHandler) GetStats() *FeedbackStats {
	stats := &FeedbackStats{
		OverrideReasons: make(map[string]int),
	}

	h.feedbackStore.Range(func(key, value interface{}) bool {
		feedback := value.(*CardFeedback)
		stats.TotalCards++
		switch feedback.Outcome {
		case "accepted":
			stats.Accepted++
		case "overridden":
			stats.Overridden++
			stats.OverrideReasons[feedback.OverrideReason]++
		case "dismissed":
			stats.Dismissed++
		}
		return true
	})

	return stats
}
