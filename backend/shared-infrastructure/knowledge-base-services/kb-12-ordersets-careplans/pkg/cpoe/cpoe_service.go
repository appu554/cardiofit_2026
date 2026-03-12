// Package cpoe provides Computerized Provider Order Entry integration
// with clinical decision support, validation, and order management
package cpoe

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kb-12-ordersets-careplans/internal/clients"
)

// CPOEService handles order entry, validation, and CDS integration
type CPOEService struct {
	kb1Client     *clients.KB1Client  // Drug rules
	kb3Client     *clients.KB3Client  // Guidelines with temporal logic
	kb6Client     *clients.KB6Client  // Formulary
	kb7Client     *clients.KB7Client  // Terminology
	orderCache    sync.Map            // In-flight order sessions
	alertHandlers []AlertHandler      // Registered alert handlers
	mu            sync.RWMutex
}

// NewCPOEService creates a new CPOE service
func NewCPOEService(kb1 *clients.KB1Client, kb3 *clients.KB3Client, kb6 *clients.KB6Client, kb7 *clients.KB7Client) *CPOEService {
	return &CPOEService{
		kb1Client:     kb1,
		kb3Client:     kb3,
		kb6Client:     kb6,
		kb7Client:     kb7,
		alertHandlers: make([]AlertHandler, 0),
	}
}

// AlertHandler is a callback for processing clinical alerts
type AlertHandler func(alert *ClinicalAlert) error

// RegisterAlertHandler adds a handler for clinical alerts
func (s *CPOEService) RegisterAlertHandler(handler AlertHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertHandlers = append(s.alertHandlers, handler)
}

// OrderSession represents an active order entry session
type OrderSession struct {
	SessionID     string                 `json:"session_id"`
	PatientID     string                 `json:"patient_id"`
	EncounterID   string                 `json:"encounter_id"`
	ProviderID    string                 `json:"provider_id"`
	OrderSetID    string                 `json:"order_set_id,omitempty"`
	Orders        []*PendingOrder        `json:"orders"`
	Alerts        []*ClinicalAlert       `json:"alerts"`
	Validations   []*ValidationResult    `json:"validations"`
	PatientData   *PatientContext        `json:"patient_data"`
	Status        string                 `json:"status"` // draft, validated, signed, cancelled
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	SignedAt      *time.Time             `json:"signed_at,omitempty"`
	SignedBy      string                 `json:"signed_by,omitempty"`
}

// PatientContext contains patient-specific data for order validation
type PatientContext struct {
	PatientID       string             `json:"patient_id"`
	Age             int                `json:"age"`
	AgeUnit         string             `json:"age_unit"` // years, months, days
	Weight          float64            `json:"weight_kg"`
	Height          float64            `json:"height_cm"`
	BSA             float64            `json:"bsa_m2"`
	Sex             string             `json:"sex"`
	Pregnant        bool               `json:"pregnant"`
	Lactating       bool               `json:"lactating"`
	RenalFunction   *RenalFunction     `json:"renal_function,omitempty"`
	HepaticFunction *HepaticFunction   `json:"hepatic_function,omitempty"`
	Allergies       []PatientAllergy   `json:"allergies"`
	ActiveMeds      []ActiveMedication `json:"active_medications"`
	Diagnoses       []PatientDiagnosis `json:"diagnoses"`
	LabResults      []LabResult        `json:"lab_results"`
}

// RenalFunction represents kidney function parameters
type RenalFunction struct {
	Creatinine float64   `json:"creatinine"`
	BUN        float64   `json:"bun"`
	GFR        float64   `json:"gfr"`
	CrCl       float64   `json:"crcl"` // Creatinine clearance
	Dialysis   bool      `json:"dialysis"`
	CKDStage   string    `json:"ckd_stage"`
	MeasuredAt time.Time `json:"measured_at"`
}

// HepaticFunction represents liver function parameters
type HepaticFunction struct {
	AST        float64   `json:"ast"`
	ALT        float64   `json:"alt"`
	Bilirubin  float64   `json:"bilirubin"`
	Albumin    float64   `json:"albumin"`
	INR        float64   `json:"inr"`
	ChildPugh  string    `json:"child_pugh_class"` // A, B, C
	MELD       int       `json:"meld_score"`
	Cirrhosis  bool      `json:"cirrhosis"`
	MeasuredAt time.Time `json:"measured_at"`
}

// PatientAllergy represents a patient allergy record
type PatientAllergy struct {
	AllergenCode     string   `json:"allergen_code"`
	AllergenSystem   string   `json:"allergen_system"` // RxNorm, SNOMED
	AllergenName     string   `json:"allergen_name"`
	ReactionType     string   `json:"reaction_type"`     // allergy, intolerance, adverse
	Severity         string   `json:"severity"`          // mild, moderate, severe, life-threatening
	Manifestations   []string `json:"manifestations"`    // rash, anaphylaxis, etc.
	OnsetDate        string   `json:"onset_date"`
	Verified         bool     `json:"verified"`
	VerifiedBy       string   `json:"verified_by"`
	CrossReactivites []string `json:"cross_reactivities"` // related allergens
}

// ActiveMedication represents a current medication
type ActiveMedication struct {
	MedicationCode   string    `json:"medication_code"`
	MedicationSystem string    `json:"medication_system"`
	MedicationName   string    `json:"medication_name"`
	Dose             string    `json:"dose"`
	DoseUnit         string    `json:"dose_unit"`
	Route            string    `json:"route"`
	Frequency        string    `json:"frequency"`
	StartDate        time.Time `json:"start_date"`
	PRN              bool      `json:"prn"`
	Status           string    `json:"status"` // active, on-hold, discontinued
}

// PatientDiagnosis represents a patient diagnosis
type PatientDiagnosis struct {
	Code       string    `json:"code"`
	System     string    `json:"system"`
	Display    string    `json:"display"`
	Type       string    `json:"type"` // admission, primary, secondary
	OnsetDate  time.Time `json:"onset_date"`
	Chronic    bool      `json:"chronic"`
	Status     string    `json:"status"`
}

// LabResult represents a laboratory result
type LabResult struct {
	Code          string    `json:"code"`
	System        string    `json:"system"` // LOINC
	Display       string    `json:"display"`
	Value         float64   `json:"value"`
	ValueString   string    `json:"value_string"`
	Unit          string    `json:"unit"`
	ReferenceRange string   `json:"reference_range"`
	Interpretation string   `json:"interpretation"` // normal, abnormal, critical
	CollectedAt   time.Time `json:"collected_at"`
}

// PendingOrder represents an order awaiting validation and signing
type PendingOrder struct {
	OrderID          string                 `json:"order_id"`
	OrderType        string                 `json:"order_type"` // medication, lab, imaging, procedure, nursing, diet
	Priority         string                 `json:"priority"`   // stat, urgent, routine, timed
	Status           string                 `json:"status"`     // pending, validated, signed, rejected
	SourceTemplateID string                 `json:"source_template_id,omitempty"`

	// Medication-specific fields
	Medication       *MedicationOrder       `json:"medication,omitempty"`

	// Lab-specific fields
	Lab              *LabOrder              `json:"lab,omitempty"`

	// Other order types
	Imaging          *ImagingOrder          `json:"imaging,omitempty"`
	Procedure        *ProcedureOrder        `json:"procedure,omitempty"`
	Nursing          *NursingOrder          `json:"nursing,omitempty"`
	Diet             *DietOrder             `json:"diet,omitempty"`

	// Validation results
	Alerts           []*ClinicalAlert       `json:"alerts"`
	Validations      []*ValidationResult    `json:"validations"`
	RequiresOverride bool                   `json:"requires_override"`
	OverrideReason   string                 `json:"override_reason,omitempty"`

	// Timestamps
	CreatedAt        time.Time              `json:"created_at"`
	ValidatedAt      *time.Time             `json:"validated_at,omitempty"`
}

// MedicationOrder represents a medication order
type MedicationOrder struct {
	MedicationCode   string  `json:"medication_code"`
	MedicationSystem string  `json:"medication_system"`
	MedicationName   string  `json:"medication_name"`
	Dose             float64 `json:"dose"`
	DoseUnit         string  `json:"dose_unit"`
	Route            string  `json:"route"`
	Frequency        string  `json:"frequency"`
	Duration         string  `json:"duration"`
	PRN              bool    `json:"prn"`
	PRNReason        string  `json:"prn_reason,omitempty"`
	Instructions     string  `json:"instructions"`
	MaxDailyDose     float64 `json:"max_daily_dose,omitempty"`
	Indication       string  `json:"indication"`
	SubstitutionAllowed bool `json:"substitution_allowed"`
	DispenseQuantity int     `json:"dispense_quantity,omitempty"`
	Refills          int     `json:"refills,omitempty"`
}

// LabOrder represents a laboratory order
type LabOrder struct {
	TestCode        string   `json:"test_code"`
	TestSystem      string   `json:"test_system"` // LOINC
	TestName        string   `json:"test_name"`
	Specimen        string   `json:"specimen"`
	CollectionTime  string   `json:"collection_time"`
	FastingRequired bool     `json:"fasting_required"`
	Indication      string   `json:"indication"`
	Frequency       string   `json:"frequency"` // once, daily, etc.
}

// ImagingOrder represents an imaging order
type ImagingOrder struct {
	ProcedureCode string `json:"procedure_code"`
	ProcedureName string `json:"procedure_name"`
	Modality      string `json:"modality"` // XR, CT, MRI, US, NM
	BodySite      string `json:"body_site"`
	Laterality    string `json:"laterality"`
	Contrast      bool   `json:"contrast"`
	Indication    string `json:"indication"`
	Transport     string `json:"transport"` // ambulatory, wheelchair, stretcher
}

// ProcedureOrder represents a procedure order
type ProcedureOrder struct {
	ProcedureCode string `json:"procedure_code"`
	ProcedureName string `json:"procedure_name"`
	Consent       bool   `json:"consent_obtained"`
	Location      string `json:"location"`
	Instructions  string `json:"instructions"`
}

// NursingOrder represents a nursing order
type NursingOrder struct {
	OrderText   string `json:"order_text"`
	Category    string `json:"category"` // assessment, activity, safety, hygiene
	Frequency   string `json:"frequency"`
	Instruction string `json:"instruction"`
}

// DietOrder represents a diet order
type DietOrder struct {
	DietType        string   `json:"diet_type"`
	Texture         string   `json:"texture"` // regular, mechanical soft, puree
	LiquidConsistency string `json:"liquid_consistency"` // thin, nectar, honey, pudding
	Restrictions    []string `json:"restrictions"` // sodium, potassium, fluid
	Allergies       []string `json:"allergies"`
	Supplements     []string `json:"supplements"`
	NPOAfter        string   `json:"npo_after,omitempty"`
}

// ClinicalAlert represents a CDS alert
type ClinicalAlert struct {
	AlertID       string                 `json:"alert_id"`
	AlertType     string                 `json:"alert_type"` // drug-drug, drug-allergy, dose, duplicate, contraindication, formulary
	Severity      string                 `json:"severity"`   // info, warning, critical, hard-stop
	Category      string                 `json:"category"`   // safety, efficacy, cost, regulatory
	Title         string                 `json:"title"`
	Message       string                 `json:"message"`
	Details       string                 `json:"details"`
	OrderID       string                 `json:"order_id"`
	TriggeringMed string                 `json:"triggering_medication,omitempty"`
	InteractingWith string               `json:"interacting_with,omitempty"`
	Evidence      string                 `json:"evidence,omitempty"`
	Recommendations []string             `json:"recommendations"`
	OverrideAllowed bool                 `json:"override_allowed"`
	OverrideReasons []string             `json:"override_reasons,omitempty"`
	Overridden    bool                   `json:"overridden"`
	OverriddenBy  string                 `json:"overridden_by,omitempty"`
	OverrideReason string                `json:"override_reason,omitempty"`
	Source        string                 `json:"source"` // KB-1, KB-5, KB-6, internal
	CreatedAt     time.Time              `json:"created_at"`
}

// ValidationResult represents a validation check result
type ValidationResult struct {
	ValidationID   string `json:"validation_id"`
	ValidationType string `json:"validation_type"` // dose, route, frequency, duration, indication
	OrderID        string `json:"order_id"`
	Passed         bool   `json:"passed"`
	Message        string `json:"message"`
	Suggestion     string `json:"suggestion,omitempty"`
	Source         string `json:"source"`
}

// CreateOrderSession creates a new order entry session
func (s *CPOEService) CreateOrderSession(ctx context.Context, req *CreateSessionRequest) (*OrderSession, error) {
	sessionID := fmt.Sprintf("SESS-%d", time.Now().UnixNano())

	session := &OrderSession{
		SessionID:   sessionID,
		PatientID:   req.PatientID,
		EncounterID: req.EncounterID,
		ProviderID:  req.ProviderID,
		OrderSetID:  req.OrderSetID,
		Orders:      make([]*PendingOrder, 0),
		Alerts:      make([]*ClinicalAlert, 0),
		Validations: make([]*ValidationResult, 0),
		PatientData: req.PatientContext,
		Status:      "draft",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.orderCache.Store(sessionID, session)
	return session, nil
}

// CreateSessionRequest is the request to create an order session
type CreateSessionRequest struct {
	PatientID      string          `json:"patient_id"`
	EncounterID    string          `json:"encounter_id"`
	ProviderID     string          `json:"provider_id"`
	OrderSetID     string          `json:"order_set_id,omitempty"`
	PatientContext *PatientContext `json:"patient_context"`
}

// AddOrder adds an order to the session and performs validation
func (s *CPOEService) AddOrder(ctx context.Context, sessionID string, order *PendingOrder) (*OrderValidationResponse, error) {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)

	// Generate order ID
	order.OrderID = fmt.Sprintf("ORD-%d", time.Now().UnixNano())
	order.CreatedAt = time.Now()
	order.Status = "pending"
	order.Alerts = make([]*ClinicalAlert, 0)
	order.Validations = make([]*ValidationResult, 0)

	// Validate the order
	validationResp, err := s.ValidateOrder(ctx, session, order)
	if err != nil {
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	// Add to session
	session.Orders = append(session.Orders, order)
	session.Alerts = append(session.Alerts, order.Alerts...)
	session.UpdatedAt = time.Now()

	// Store updated session
	s.orderCache.Store(sessionID, session)

	return validationResp, nil
}

// OrderValidationResponse is the response from order validation
type OrderValidationResponse struct {
	OrderID          string              `json:"order_id"`
	Valid            bool                `json:"valid"`
	Alerts           []*ClinicalAlert    `json:"alerts"`
	Validations      []*ValidationResult `json:"validations"`
	RequiresOverride bool                `json:"requires_override"`
	BlockingAlerts   int                 `json:"blocking_alerts"`
	WarningAlerts    int                 `json:"warning_alerts"`
	InfoAlerts       int                 `json:"info_alerts"`
}

// ValidateOrder performs comprehensive order validation
func (s *CPOEService) ValidateOrder(ctx context.Context, session *OrderSession, order *PendingOrder) (*OrderValidationResponse, error) {
	response := &OrderValidationResponse{
		OrderID:     order.OrderID,
		Valid:       true,
		Alerts:      make([]*ClinicalAlert, 0),
		Validations: make([]*ValidationResult, 0),
	}

	// Medication-specific validations
	if order.Medication != nil {
		// Check for drug allergies
		allergyAlerts := s.checkDrugAllergies(session.PatientData, order.Medication)
		response.Alerts = append(response.Alerts, allergyAlerts...)

		// Check for drug-drug interactions
		interactionAlerts := s.checkDrugInteractions(ctx, session.PatientData, order.Medication)
		response.Alerts = append(response.Alerts, interactionAlerts...)

		// Validate dose for patient parameters
		doseValidations := s.validateDose(ctx, session.PatientData, order.Medication)
		response.Validations = append(response.Validations, doseValidations...)

		// Check duplicate therapy
		dupAlerts := s.checkDuplicateTherapy(session.PatientData, order.Medication)
		response.Alerts = append(response.Alerts, dupAlerts...)

		// Check formulary status
		formularyAlerts := s.checkFormularyStatus(ctx, order.Medication)
		response.Alerts = append(response.Alerts, formularyAlerts...)

		// Check contraindications
		contraAlerts := s.checkContraindications(session.PatientData, order.Medication)
		response.Alerts = append(response.Alerts, contraAlerts...)

		// Check renal dosing
		if session.PatientData.RenalFunction != nil {
			renalAlerts := s.checkRenalDosing(ctx, session.PatientData, order.Medication)
			response.Alerts = append(response.Alerts, renalAlerts...)
		}

		// Check hepatic dosing
		if session.PatientData.HepaticFunction != nil {
			hepaticAlerts := s.checkHepaticDosing(session.PatientData, order.Medication)
			response.Alerts = append(response.Alerts, hepaticAlerts...)
		}

		// Check pregnancy/lactation
		if session.PatientData.Pregnant || session.PatientData.Lactating {
			pregLactAlerts := s.checkPregnancyLactation(session.PatientData, order.Medication)
			response.Alerts = append(response.Alerts, pregLactAlerts...)
		}
	}

	// Lab-specific validations
	if order.Lab != nil {
		labValidations := s.validateLabOrder(session.PatientData, order.Lab)
		response.Validations = append(response.Validations, labValidations...)
	}

	// Imaging validations
	if order.Imaging != nil {
		imagingAlerts := s.validateImagingOrder(session.PatientData, order.Imaging)
		response.Alerts = append(response.Alerts, imagingAlerts...)
	}

	// Copy alerts to order
	order.Alerts = response.Alerts
	order.Validations = response.Validations

	// Count alerts by severity
	for _, alert := range response.Alerts {
		switch alert.Severity {
		case "hard-stop":
			response.BlockingAlerts++
			response.Valid = false
			response.RequiresOverride = true
		case "critical":
			response.BlockingAlerts++
			response.RequiresOverride = true
		case "warning":
			response.WarningAlerts++
		case "info":
			response.InfoAlerts++
		}
	}

	// Check validation failures
	for _, val := range response.Validations {
		if !val.Passed {
			response.RequiresOverride = true
		}
	}

	order.RequiresOverride = response.RequiresOverride
	if response.BlockingAlerts > 0 || !response.Valid {
		order.Status = "rejected"
	} else {
		now := time.Now()
		order.ValidatedAt = &now
		order.Status = "validated"
	}

	return response, nil
}

// checkDrugAllergies checks for drug-allergy interactions
func (s *CPOEService) checkDrugAllergies(patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	for _, allergy := range patient.Allergies {
		// Direct match
		if allergy.AllergenCode == med.MedicationCode {
			alert := &ClinicalAlert{
				AlertID:       fmt.Sprintf("ALLERGY-%d", time.Now().UnixNano()),
				AlertType:     "drug-allergy",
				Severity:      s.mapAllergySeverity(allergy.Severity),
				Category:      "safety",
				Title:         "Drug-Allergy Alert",
				Message:       fmt.Sprintf("Patient has documented allergy to %s", allergy.AllergenName),
				Details:       fmt.Sprintf("Reaction: %s, Manifestations: %v", allergy.ReactionType, allergy.Manifestations),
				OrderID:       "",
				TriggeringMed: med.MedicationName,
				Recommendations: []string{
					"Consider alternative medication",
					"If proceeding, ensure resuscitation equipment available",
				},
				OverrideAllowed: allergy.Severity != "life-threatening",
				OverrideReasons: []string{
					"Patient tolerated medication previously",
					"No suitable alternative available",
					"Benefit outweighs risk - patient informed",
				},
				Source:    "CPOE-Allergy-Check",
				CreatedAt: time.Now(),
			}
			alerts = append(alerts, alert)
		}

		// Check cross-reactivities (e.g., penicillin-cephalosporin)
		for _, crossReactive := range allergy.CrossReactivites {
			if crossReactive == med.MedicationCode || crossReactive == med.MedicationName {
				alert := &ClinicalAlert{
					AlertID:       fmt.Sprintf("CROSS-ALLERGY-%d", time.Now().UnixNano()),
					AlertType:     "drug-allergy",
					Severity:      "warning",
					Category:      "safety",
					Title:         "Cross-Reactivity Alert",
					Message:       fmt.Sprintf("Patient allergic to %s - potential cross-reactivity with %s", allergy.AllergenName, med.MedicationName),
					TriggeringMed: med.MedicationName,
					InteractingWith: allergy.AllergenName,
					Recommendations: []string{
						"Consider alternative medication",
						"Monitor closely if proceeding",
					},
					OverrideAllowed: true,
					Source:    "CPOE-CrossReactivity-Check",
					CreatedAt: time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

// mapAllergySeverity maps allergy severity to alert severity
func (s *CPOEService) mapAllergySeverity(allergySeverity string) string {
	switch allergySeverity {
	case "life-threatening":
		return "hard-stop"
	case "severe":
		return "critical"
	case "moderate":
		return "warning"
	default:
		return "info"
	}
}

// checkDrugInteractions checks for drug-drug interactions
func (s *CPOEService) checkDrugInteractions(ctx context.Context, patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	// Call KB-1 for drug interaction checking
	if s.kb1Client != nil {
		for _, activeMed := range patient.ActiveMeds {
			// Build interaction check request
			interactionReq := &clients.InteractionCheckRequest{
				DrugCodes:     []string{med.MedicationCode, activeMed.MedicationCode},
				PatientID:     patient.PatientID,
				IncludeSevere: true,
			}

			interactionResp, err := s.kb1Client.CheckInteraction(ctx, interactionReq)
			if err != nil || interactionResp == nil {
				continue // Log error but continue checking
			}

			for _, interaction := range interactionResp.Interactions {
				alert := &ClinicalAlert{
					AlertID:         fmt.Sprintf("DDI-%d", time.Now().UnixNano()),
					AlertType:       "drug-drug",
					Severity:        s.mapInteractionSeverity(interaction.Severity),
					Category:        "safety",
					Title:           "Drug-Drug Interaction",
					Message:         fmt.Sprintf("Interaction between %s and %s", med.MedicationName, activeMed.MedicationName),
					Details:         interaction.Description,
					TriggeringMed:   med.MedicationName,
					InteractingWith: activeMed.MedicationName,
					Evidence:        interaction.Mechanism,
					Recommendations: []string{interaction.Reference},
					OverrideAllowed: interaction.Severity != "contraindicated",
					Source:          "KB-1-Interactions",
					CreatedAt:       time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

// mapInteractionSeverity maps interaction severity to alert severity
func (s *CPOEService) mapInteractionSeverity(severity string) string {
	switch severity {
	case "contraindicated", "major":
		return "critical"
	case "moderate":
		return "warning"
	default:
		return "info"
	}
}

// validateDose validates medication dosing
func (s *CPOEService) validateDose(ctx context.Context, patient *PatientContext, med *MedicationOrder) []*ValidationResult {
	validations := make([]*ValidationResult, 0)

	// Call KB-1 for dose validation (using KB-1 API format)
	if s.kb1Client != nil {
		doseReq := &clients.DoseValidationRequest{
			RxNormCode:   med.MedicationCode,
			ProposedDose: med.Dose,
			Unit:         med.DoseUnit,
			Age:          patient.Age,
			Gender:       patient.Sex,
			WeightKg:     patient.Weight,
			HeightCm:     patient.Height,
		}

		doseResp, err := s.kb1Client.ValidateDose(ctx, doseReq)
		if err == nil && doseResp != nil {
			validation := &ValidationResult{
				ValidationID:   fmt.Sprintf("DOSE-%d", time.Now().UnixNano()),
				ValidationType: "dose",
				Passed:         doseResp.Valid,
				Message:        doseResp.ErrorMessage,
				Source:         "KB-1-Dosing",
			}

			if !doseResp.Valid {
				validation.Suggestion = fmt.Sprintf("Recommended range: %.2f - %.2f %s",
					doseResp.RecommendedMin, doseResp.RecommendedMax, med.DoseUnit)
			}

			validations = append(validations, validation)
		}
	}

	// Weight-based dose check for pediatrics
	if patient.Age < 18 || patient.AgeUnit == "months" || patient.AgeUnit == "days" {
		// Calculate mg/kg
		mgPerKg := med.Dose / patient.Weight
		validation := &ValidationResult{
			ValidationID:   fmt.Sprintf("PEDDOSE-%d", time.Now().UnixNano()),
			ValidationType: "pediatric_dose",
			Passed:         true, // Would need actual pediatric dosing data
			Message:        fmt.Sprintf("Pediatric dose: %.2f mg/kg", mgPerKg),
			Source:         "CPOE-Pediatric-Check",
		}
		validations = append(validations, validation)
	}

	return validations
}

// checkDuplicateTherapy checks for therapeutic duplications
func (s *CPOEService) checkDuplicateTherapy(patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	// Check for same medication
	for _, activeMed := range patient.ActiveMeds {
		if activeMed.MedicationCode == med.MedicationCode && activeMed.Status == "active" {
			alert := &ClinicalAlert{
				AlertID:       fmt.Sprintf("DUP-%d", time.Now().UnixNano()),
				AlertType:     "duplicate",
				Severity:      "warning",
				Category:      "safety",
				Title:         "Duplicate Medication",
				Message:       fmt.Sprintf("Patient already receiving %s", activeMed.MedicationName),
				Details:       fmt.Sprintf("Current: %s %s %s", activeMed.Dose, activeMed.DoseUnit, activeMed.Frequency),
				TriggeringMed: med.MedicationName,
				Recommendations: []string{
					"Review current medication",
					"Discontinue existing order if replacing",
					"Confirm dose adjustment is intentional",
				},
				OverrideAllowed: true,
				OverrideReasons: []string{
					"Dose adjustment - discontinuing previous order",
					"PRN in addition to scheduled",
					"Different indication",
				},
				Source:    "CPOE-Duplicate-Check",
				CreatedAt: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// checkFormularyStatus checks medication formulary status
func (s *CPOEService) checkFormularyStatus(ctx context.Context, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	if s.kb6Client != nil {
		formularyReq := &clients.FormularyCheckRequest{
			DrugCode: med.MedicationCode,
			DrugName: med.MedicationName,
		}

		formResp, err := s.kb6Client.CheckFormulary(ctx, formularyReq)
		if err == nil && formResp != nil {
			if !formResp.FormularyStatus.OnFormulary {
				// Build alternative names list
				altNames := make([]string, 0, len(formResp.Alternatives))
				for _, alt := range formResp.Alternatives {
					altNames = append(altNames, alt.DrugName)
				}
				alert := &ClinicalAlert{
					AlertID:     fmt.Sprintf("FORM-%d", time.Now().UnixNano()),
					AlertType:   "formulary",
					Severity:    "warning",
					Category:    "cost",
					Title:       "Non-Formulary Medication",
					Message:     fmt.Sprintf("%s is not on the hospital formulary", med.MedicationName),
					TriggeringMed: med.MedicationName,
					Recommendations: append([]string{"Consider formulary alternatives:"}, altNames...),
					OverrideAllowed: true,
					OverrideReasons: []string{
						"No suitable alternative available",
						"Prior therapy failure with alternatives",
						"Continuation of home medication",
					},
					Source:    "KB-6-Formulary",
					CreatedAt: time.Now(),
				}
				alerts = append(alerts, alert)
			}

			if formResp.PARequired {
				alert := &ClinicalAlert{
					AlertID:     fmt.Sprintf("PA-%d", time.Now().UnixNano()),
					AlertType:   "formulary",
					Severity:    "info",
					Category:    "regulatory",
					Title:       "Prior Authorization Required",
					Message:     fmt.Sprintf("%s requires prior authorization", med.MedicationName),
					Details:     fmt.Sprintf("Restrictions: %v", formResp.FormularyStatus.Restrictions),
					TriggeringMed: med.MedicationName,
					Recommendations: []string{
						"Complete prior authorization form",
						"Document medical necessity",
					},
					OverrideAllowed: true,
					Source:    "KB-6-Formulary",
					CreatedAt: time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

// checkContraindications checks for contraindications based on diagnoses
func (s *CPOEService) checkContraindications(patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	// Common contraindication patterns (would typically come from KB-1)
	contraindications := map[string][]struct {
		Diagnosis   string
		Severity    string
		Message     string
	}{
		"6809": { // Metformin
			{Diagnosis: "N18.5", Severity: "hard-stop", Message: "Metformin contraindicated in ESRD (eGFR < 30)"},
			{Diagnosis: "K70.3", Severity: "hard-stop", Message: "Metformin contraindicated in hepatic failure"},
		},
		"11289": { // Warfarin
			{Diagnosis: "K92.2", Severity: "critical", Message: "Active GI bleeding - warfarin contraindicated"},
		},
	}

	if contras, exists := contraindications[med.MedicationCode]; exists {
		for _, contra := range contras {
			for _, dx := range patient.Diagnoses {
				if dx.Code == contra.Diagnosis {
					alert := &ClinicalAlert{
						AlertID:       fmt.Sprintf("CONTRA-%d", time.Now().UnixNano()),
						AlertType:     "contraindication",
						Severity:      contra.Severity,
						Category:      "safety",
						Title:         "Contraindication Alert",
						Message:       contra.Message,
						Details:       fmt.Sprintf("Patient diagnosis: %s (%s)", dx.Display, dx.Code),
						TriggeringMed: med.MedicationName,
						OverrideAllowed: contra.Severity != "hard-stop",
						Source:    "CPOE-Contraindication-Check",
						CreatedAt: time.Now(),
					}
					alerts = append(alerts, alert)
				}
			}
		}
	}

	return alerts
}

// checkRenalDosing checks for renal dose adjustments
func (s *CPOEService) checkRenalDosing(ctx context.Context, patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	if patient.RenalFunction == nil {
		return alerts
	}

	// Call KB-1 for renal dosing (using KB-1 API format)
	if s.kb1Client != nil {
		renalReq := &clients.RenalDosingRequest{
			RxNormCode:      med.MedicationCode,
			Age:             patient.Age,
			Gender:          patient.Sex,
			WeightKg:        patient.Weight,
			HeightCm:        patient.Height,
			SerumCreatinine: patient.RenalFunction.Creatinine,
			EGFR:            patient.RenalFunction.GFR,
		}

		renalResp, err := s.kb1Client.CheckRenalDosing(ctx, renalReq)
		if err == nil && renalResp != nil {
			if renalResp.Contraindicated {
				alert := &ClinicalAlert{
					AlertID:       fmt.Sprintf("RENAL-CONTRA-%d", time.Now().UnixNano()),
					AlertType:     "contraindication",
					Severity:      "hard-stop",
					Category:      "safety",
					Title:         "Renal Contraindication",
					Message:       fmt.Sprintf("%s contraindicated at current GFR (%.0f mL/min)", med.MedicationName, patient.RenalFunction.GFR),
					Details:       renalResp.ErrorMessage,
					TriggeringMed: med.MedicationName,
					OverrideAllowed: false,
					Source:    "KB-1-Renal",
					CreatedAt: time.Now(),
				}
				alerts = append(alerts, alert)
			} else if renalResp.RequiresAdjust {
				alert := &ClinicalAlert{
					AlertID:       fmt.Sprintf("RENAL-ADJ-%d", time.Now().UnixNano()),
					AlertType:     "dose",
					Severity:      "warning",
					Category:      "safety",
					Title:         "Renal Dose Adjustment Required",
					Message:       fmt.Sprintf("Dose adjustment needed for GFR %.0f mL/min (Renal stage: %s)", patient.RenalFunction.GFR, renalResp.RenalStage),
					Details:       fmt.Sprintf("Dose reduction: %.0f%%", renalResp.ReductionPercent),
					TriggeringMed: med.MedicationName,
					Recommendations: []string{
						fmt.Sprintf("Adjusted dose: %.2f %s", renalResp.AdjustedDose, med.DoseUnit),
						renalResp.Reference,
					},
					OverrideAllowed: true,
					OverrideReasons: []string{
						"Dose verified by clinical pharmacist",
						"Recent GFR - dose appropriate",
					},
					Source:    "KB-1-Renal",
					CreatedAt: time.Now(),
				}
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

// checkHepaticDosing checks for hepatic dose adjustments
func (s *CPOEService) checkHepaticDosing(patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	if patient.HepaticFunction == nil {
		return alerts
	}

	// High-risk hepatic medications (would typically come from KB-1)
	hepaticRiskMeds := map[string]struct {
		Adjustment string
		ChildPughC string
	}{
		"161": { // Acetaminophen
			Adjustment: "Reduce dose by 50% in severe hepatic impairment",
			ChildPughC: "Avoid or use max 2g/day",
		},
		"42347": { // Bupropion
			Adjustment: "Reduce frequency in moderate impairment",
			ChildPughC: "Contraindicated",
		},
	}

	if risk, exists := hepaticRiskMeds[med.MedicationCode]; exists {
		if patient.HepaticFunction.ChildPugh == "C" {
			alert := &ClinicalAlert{
				AlertID:       fmt.Sprintf("HEPATIC-%d", time.Now().UnixNano()),
				AlertType:     "dose",
				Severity:      "critical",
				Category:      "safety",
				Title:         "Hepatic Impairment - Dose Adjustment",
				Message:       fmt.Sprintf("%s requires adjustment in Child-Pugh C", med.MedicationName),
				Details:       risk.ChildPughC,
				TriggeringMed: med.MedicationName,
				Recommendations: []string{risk.Adjustment, risk.ChildPughC},
				OverrideAllowed: true,
				Source:    "CPOE-Hepatic-Check",
				CreatedAt: time.Now(),
			}
			alerts = append(alerts, alert)
		} else if patient.HepaticFunction.ChildPugh == "B" {
			alert := &ClinicalAlert{
				AlertID:       fmt.Sprintf("HEPATIC-%d", time.Now().UnixNano()),
				AlertType:     "dose",
				Severity:      "warning",
				Category:      "safety",
				Title:         "Hepatic Impairment - Consider Adjustment",
				Message:       fmt.Sprintf("Consider dose adjustment for %s in moderate hepatic impairment", med.MedicationName),
				Details:       risk.Adjustment,
				TriggeringMed: med.MedicationName,
				OverrideAllowed: true,
				Source:    "CPOE-Hepatic-Check",
				CreatedAt: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// checkPregnancyLactation checks for pregnancy/lactation safety
func (s *CPOEService) checkPregnancyLactation(patient *PatientContext, med *MedicationOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	// FDA pregnancy categories (historical) and LactMed data would come from KB-4
	// This is a simplified version
	highRiskPregnancy := map[string]string{
		"11289":  "Warfarin is teratogenic - use heparin alternatives",
		"83367":  "Statins contraindicated in pregnancy",
		"321988": "SSRIs - balance risk/benefit, PPHN risk in 3rd trimester",
	}

	if patient.Pregnant {
		if warning, exists := highRiskPregnancy[med.MedicationCode]; exists {
			alert := &ClinicalAlert{
				AlertID:       fmt.Sprintf("PREG-%d", time.Now().UnixNano()),
				AlertType:     "contraindication",
				Severity:      "critical",
				Category:      "safety",
				Title:         "Pregnancy Warning",
				Message:       fmt.Sprintf("Pregnancy precaution for %s", med.MedicationName),
				Details:       warning,
				TriggeringMed: med.MedicationName,
				Recommendations: []string{
					"Consult OB/GYN or maternal-fetal medicine",
					"Document risk-benefit discussion with patient",
				},
				OverrideAllowed: true,
				OverrideReasons: []string{
					"Risk-benefit discussed with patient and OB",
					"No safer alternative available",
					"Life-threatening condition requires treatment",
				},
				Source:    "CPOE-Pregnancy-Check",
				CreatedAt: time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	if patient.Lactating {
		// Would check LactMed database via KB-4
		alert := &ClinicalAlert{
			AlertID:       fmt.Sprintf("LACT-%d", time.Now().UnixNano()),
			AlertType:     "safety",
			Severity:      "info",
			Category:      "safety",
			Title:         "Lactation Review Recommended",
			Message:       fmt.Sprintf("Verify lactation compatibility for %s", med.MedicationName),
			TriggeringMed: med.MedicationName,
			Recommendations: []string{
				"Review LactMed database",
				"Consider infant age and feeding frequency",
			},
			OverrideAllowed: true,
			Source:    "CPOE-Lactation-Check",
			CreatedAt: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// validateLabOrder validates lab orders
func (s *CPOEService) validateLabOrder(patient *PatientContext, lab *LabOrder) []*ValidationResult {
	validations := make([]*ValidationResult, 0)

	// Check for duplicate/recent labs
	for _, result := range patient.LabResults {
		if result.Code == lab.TestCode {
			hoursSince := time.Since(result.CollectedAt).Hours()
			if hoursSince < 24 {
				validation := &ValidationResult{
					ValidationID:   fmt.Sprintf("LABDUP-%d", time.Now().UnixNano()),
					ValidationType: "duplicate",
					Passed:         false,
					Message:        fmt.Sprintf("Recent %s result available from %.0f hours ago", lab.TestName, hoursSince),
					Suggestion:     "Consider if repeat testing is needed",
					Source:         "CPOE-Lab-Check",
				}
				validations = append(validations, validation)
			}
		}
	}

	return validations
}

// validateImagingOrder validates imaging orders
func (s *CPOEService) validateImagingOrder(patient *PatientContext, imaging *ImagingOrder) []*ClinicalAlert {
	alerts := make([]*ClinicalAlert, 0)

	// Contrast safety for renal impairment
	if imaging.Contrast && patient.RenalFunction != nil && patient.RenalFunction.GFR < 60 {
		severity := "warning"
		if patient.RenalFunction.GFR < 30 {
			severity = "critical"
		}
		alert := &ClinicalAlert{
			AlertID:     fmt.Sprintf("CONTRAST-%d", time.Now().UnixNano()),
			AlertType:   "contraindication",
			Severity:    severity,
			Category:    "safety",
			Title:       "Contrast Nephropathy Risk",
			Message:     fmt.Sprintf("IV contrast ordered with GFR %.0f mL/min", patient.RenalFunction.GFR),
			Details:     "Risk of contrast-induced nephropathy",
			Recommendations: []string{
				"Consider non-contrast alternative",
				"If proceeding: IV hydration protocol",
				"Hold metformin for 48 hours after contrast",
				"Recheck creatinine 48-72 hours post-procedure",
			},
			OverrideAllowed: true,
			OverrideReasons: []string{
				"Hydration protocol ordered",
				"Benefits outweigh risks - urgent diagnosis needed",
				"Discussed with radiology",
			},
			Source:    "CPOE-Contrast-Check",
			CreatedAt: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	// Pregnancy check for radiation
	if patient.Pregnant && (imaging.Modality == "CT" || imaging.Modality == "XR" || imaging.Modality == "NM") {
		alert := &ClinicalAlert{
			AlertID:     fmt.Sprintf("RADPREG-%d", time.Now().UnixNano()),
			AlertType:   "contraindication",
			Severity:    "critical",
			Category:    "safety",
			Title:       "Radiation Exposure in Pregnancy",
			Message:     fmt.Sprintf("%s ordered for pregnant patient", imaging.Modality),
			Recommendations: []string{
				"Consider non-ionizing alternative (US, MRI)",
				"If essential: shield fetus, minimize exposure",
				"Document risk-benefit discussion",
				"Consult radiology for lowest-dose protocol",
			},
			OverrideAllowed: true,
			OverrideReasons: []string{
				"Maternal life-threatening emergency",
				"No non-ionizing alternative adequate",
				"Discussed with radiology and OB",
			},
			Source:    "CPOE-Pregnancy-Radiation",
			CreatedAt: time.Now(),
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// SignOrders signs all validated orders in the session
func (s *CPOEService) SignOrders(ctx context.Context, sessionID string, signerID string, overrides map[string]string) (*SignOrdersResponse, error) {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)

	response := &SignOrdersResponse{
		SessionID:    sessionID,
		SignedOrders: make([]string, 0),
		FailedOrders: make([]FailedOrder, 0),
	}

	now := time.Now()

	for _, order := range session.Orders {
		// Check if order requires override
		if order.RequiresOverride {
			if overrideReason, ok := overrides[order.OrderID]; ok {
				order.OverrideReason = overrideReason
				// Mark alerts as overridden
				for _, alert := range order.Alerts {
					if alert.OverrideAllowed {
						alert.Overridden = true
						alert.OverriddenBy = signerID
						alert.OverrideReason = overrideReason
					}
				}
			} else {
				// No override provided for required alert
				hasHardStop := false
				for _, alert := range order.Alerts {
					if alert.Severity == "hard-stop" && !alert.OverrideAllowed {
						hasHardStop = true
						break
					}
				}

				if hasHardStop {
					response.FailedOrders = append(response.FailedOrders, FailedOrder{
						OrderID: order.OrderID,
						Reason:  "Hard-stop alert cannot be overridden",
					})
					continue
				}

				// Check if all alerts have overrides or are overridable
				needsOverride := false
				for _, alert := range order.Alerts {
					if (alert.Severity == "critical" || alert.Severity == "hard-stop") && !alert.Overridden {
						needsOverride = true
						break
					}
				}

				if needsOverride {
					response.FailedOrders = append(response.FailedOrders, FailedOrder{
						OrderID: order.OrderID,
						Reason:  "Override reason required for critical alerts",
					})
					continue
				}
			}
		}

		order.Status = "signed"
		response.SignedOrders = append(response.SignedOrders, order.OrderID)
	}

	// Update session status
	if len(response.FailedOrders) == 0 {
		session.Status = "signed"
		session.SignedAt = &now
		session.SignedBy = signerID
	} else if len(response.SignedOrders) > 0 {
		session.Status = "partially_signed"
	}

	session.UpdatedAt = now
	s.orderCache.Store(sessionID, session)

	response.Success = len(response.FailedOrders) == 0
	return response, nil
}

// SignOrdersResponse is the response from signing orders
type SignOrdersResponse struct {
	SessionID    string        `json:"session_id"`
	Success      bool          `json:"success"`
	SignedOrders []string      `json:"signed_orders"`
	FailedOrders []FailedOrder `json:"failed_orders"`
}

// FailedOrder represents an order that failed to sign
type FailedOrder struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// GetSession retrieves an order session
func (s *CPOEService) GetSession(sessionID string) (*OrderSession, error) {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	return sessionVal.(*OrderSession), nil
}

// CancelSession cancels an order session
func (s *CPOEService) CancelSession(sessionID string, reason string) error {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)
	session.Status = "cancelled"
	session.UpdatedAt = time.Now()
	s.orderCache.Store(sessionID, session)
	return nil
}

// RemoveOrder removes an order from a session
func (s *CPOEService) RemoveOrder(sessionID, orderID string) error {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)

	for i, order := range session.Orders {
		if order.OrderID == orderID {
			session.Orders = append(session.Orders[:i], session.Orders[i+1:]...)
			session.UpdatedAt = time.Now()
			s.orderCache.Store(sessionID, session)
			return nil
		}
	}

	return fmt.Errorf("order not found: %s", orderID)
}

// ModifyOrder modifies an existing order and revalidates
func (s *CPOEService) ModifyOrder(ctx context.Context, sessionID string, order *PendingOrder) (*OrderValidationResponse, error) {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)

	// Find and replace the order
	found := false
	for i, existingOrder := range session.Orders {
		if existingOrder.OrderID == order.OrderID {
			// Revalidate
			order.Alerts = make([]*ClinicalAlert, 0)
			order.Validations = make([]*ValidationResult, 0)

			validationResp, err := s.ValidateOrder(ctx, session, order)
			if err != nil {
				return nil, fmt.Errorf("order revalidation failed: %w", err)
			}

			session.Orders[i] = order
			session.UpdatedAt = time.Now()
			s.orderCache.Store(sessionID, session)

			found = true
			return validationResp, nil
		}
	}

	if !found {
		return nil, fmt.Errorf("order not found: %s", order.OrderID)
	}

	return nil, nil
}

// GetAlertSummary returns a summary of all alerts in a session
func (s *CPOEService) GetAlertSummary(sessionID string) (*AlertSummary, error) {
	sessionVal, ok := s.orderCache.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	session := sessionVal.(*OrderSession)

	summary := &AlertSummary{
		SessionID:      sessionID,
		TotalAlerts:    0,
		HardStops:      0,
		CriticalAlerts: 0,
		Warnings:       0,
		InfoAlerts:     0,
		AlertsByType:   make(map[string]int),
		AlertsByOrder:  make(map[string]int),
	}

	for _, order := range session.Orders {
		summary.AlertsByOrder[order.OrderID] = len(order.Alerts)
		for _, alert := range order.Alerts {
			summary.TotalAlerts++
			summary.AlertsByType[alert.AlertType]++

			switch alert.Severity {
			case "hard-stop":
				summary.HardStops++
			case "critical":
				summary.CriticalAlerts++
			case "warning":
				summary.Warnings++
			case "info":
				summary.InfoAlerts++
			}
		}
	}

	summary.CanSign = summary.HardStops == 0
	summary.RequiresOverride = summary.CriticalAlerts > 0

	return summary, nil
}

// AlertSummary provides a summary of alerts in a session
type AlertSummary struct {
	SessionID        string         `json:"session_id"`
	TotalAlerts      int            `json:"total_alerts"`
	HardStops        int            `json:"hard_stops"`
	CriticalAlerts   int            `json:"critical_alerts"`
	Warnings         int            `json:"warnings"`
	InfoAlerts       int            `json:"info_alerts"`
	AlertsByType     map[string]int `json:"alerts_by_type"`
	AlertsByOrder    map[string]int `json:"alerts_by_order"`
	CanSign          bool           `json:"can_sign"`
	RequiresOverride bool           `json:"requires_override"`
}
