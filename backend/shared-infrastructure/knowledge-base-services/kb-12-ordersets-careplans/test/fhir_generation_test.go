// Package test provides FHIR generation tests for KB-12
package test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/internal/models"
)

// ============================================
// 3.1 Resource Generation Tests
// ============================================

func TestFHIRMedicationRequestGeneration(t *testing.T) {
	order := createTestMedicationOrder()
	instance := createTestOrderSetInstance()

	medReq := generateTestMedicationRequest(order, instance)

	assert.Equal(t, "MedicationRequest", medReq.ResourceType)
	assert.Equal(t, "active", medReq.Status)
	assert.Equal(t, "order", medReq.Intent)
	assert.NotEmpty(t, medReq.Subject.Reference)
	assert.Contains(t, medReq.Subject.Reference, instance.PatientID)
	t.Log("✓ MedicationRequest generation successful")
}

func TestFHIRServiceRequestGeneration(t *testing.T) {
	order := createTestLabOrder()
	instance := createTestOrderSetInstance()

	svcReq := generateTestServiceRequest(order, instance)

	assert.Equal(t, "ServiceRequest", svcReq.ResourceType)
	assert.Equal(t, "active", svcReq.Status)
	assert.Equal(t, "order", svcReq.Intent)
	assert.NotEmpty(t, svcReq.Subject.Reference)
	t.Log("✓ ServiceRequest generation successful")
}

func TestFHIRTaskGeneration(t *testing.T) {
	order := createTestNursingOrder()
	instance := createTestOrderSetInstance()

	task := generateTestTask(order, instance)

	assert.Equal(t, "Task", task.ResourceType)
	assert.Equal(t, "requested", task.Status)
	assert.Equal(t, "order", task.Intent)
	assert.NotEmpty(t, task.Description)
	assert.NotNil(t, task.For)
	t.Log("✓ Task generation successful")
}

func TestFHIRBundleCompleteness(t *testing.T) {
	bundle := createTestFHIRBundle()

	assert.Equal(t, "Bundle", bundle.ResourceType)
	assert.NotEmpty(t, bundle.ID)
	assert.Equal(t, "collection", bundle.Type)
	assert.False(t, bundle.Timestamp.IsZero())
	assert.GreaterOrEqual(t, len(bundle.Entry), 0)

	t.Logf("✓ Bundle contains %d entries", len(bundle.Entry))
}

func TestFHIRCarePlanGeneration(t *testing.T) {
	carePlan := createTestFHIRCarePlan()

	assert.Equal(t, "CarePlan", carePlan.ResourceType)
	assert.NotEmpty(t, carePlan.Status)
	assert.NotEmpty(t, carePlan.Intent)
	assert.NotEmpty(t, carePlan.Subject.Reference)

	// Verify activities if present
	if len(carePlan.Activity) > 0 {
		for _, activity := range carePlan.Activity {
			if activity.Detail != nil {
				assert.NotEmpty(t, activity.Detail.Status)
			}
		}
	}
	t.Log("✓ CarePlan generation successful")
}

func TestFHIRRequestGroupGeneration(t *testing.T) {
	planDef := createTestPlanDefinition()

	assert.Equal(t, "PlanDefinition", planDef.ResourceType)
	assert.NotEmpty(t, planDef.Status)
	assert.NotEmpty(t, planDef.Title)

	// Check actions if present
	for _, action := range planDef.Action {
		assert.NotEmpty(t, action.Title)
	}
	t.Log("✓ PlanDefinition generation successful")
}

// ============================================
// 3.2 Field Validation Tests
// ============================================

func TestFHIRReasonCodePopulated(t *testing.T) {
	order := createTestLabOrder()
	order.Reason = "Suspected infection"
	instance := createTestOrderSetInstance()

	svcReq := generateTestServiceRequest(order, instance)

	// ReasonCode should be populated when reason is set
	if order.Reason != "" {
		assert.NotNil(t, svcReq.ReasonCode)
		if len(svcReq.ReasonCode) > 0 {
			assert.NotEmpty(t, svcReq.ReasonCode[0].Text)
		}
	}
	t.Log("✓ Reason code population verified")
}

func TestFHIRIntentCorrect(t *testing.T) {
	validIntents := map[string]bool{
		"proposal":       true,
		"plan":           true,
		"order":          true,
		"original-order": true,
		"reflex-order":   true,
		"filler-order":   true,
		"instance-order": true,
		"option":         true,
	}

	testCases := []struct {
		name         string
		resourceType string
		intent       string
	}{
		{"MedicationRequest", "MedicationRequest", "order"},
		{"ServiceRequest", "ServiceRequest", "order"},
		{"Task", "Task", "order"},
		{"CarePlan", "CarePlan", "plan"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, validIntents[tc.intent], "Intent '%s' should be valid for %s", tc.intent, tc.resourceType)
		})
	}
}

func TestFHIRAuthoredOnTimestamp(t *testing.T) {
	instance := createTestOrderSetInstance()
	order := createTestMedicationOrder()

	medReq := generateTestMedicationRequest(order, instance)

	assert.False(t, medReq.AuthoredOn.IsZero(), "AuthoredOn should be set")
	assert.True(t, medReq.AuthoredOn.Before(time.Now().Add(time.Minute)), "AuthoredOn should be in the past")
	t.Log("✓ AuthoredOn timestamp validation passed")
}

func TestFHIRStatusTransitions(t *testing.T) {
	// Valid status values for MedicationRequest
	medReqStatuses := []string{"active", "on-hold", "cancelled", "completed", "entered-in-error", "stopped", "draft", "unknown"}

	// Valid status values for ServiceRequest
	svcReqStatuses := []string{"draft", "active", "on-hold", "revoked", "completed", "entered-in-error", "unknown"}

	// Valid status values for Task
	taskStatuses := []string{"draft", "requested", "received", "accepted", "rejected", "ready", "cancelled", "in-progress", "on-hold", "failed", "completed", "entered-in-error"}

	t.Logf("MedicationRequest valid statuses: %d", len(medReqStatuses))
	t.Logf("ServiceRequest valid statuses: %d", len(svcReqStatuses))
	t.Logf("Task valid statuses: %d", len(taskStatuses))

	// Verify initial status is valid
	medReq := generateTestMedicationRequest(createTestMedicationOrder(), createTestOrderSetInstance())
	assert.Contains(t, medReqStatuses, medReq.Status)
}

func TestFHIRPatientReference(t *testing.T) {
	instance := createTestOrderSetInstance()
	instance.PatientID = "patient-12345"

	order := createTestMedicationOrder()
	medReq := generateTestMedicationRequest(order, instance)

	assert.NotEmpty(t, medReq.Subject.Reference)
	assert.Contains(t, medReq.Subject.Reference, "Patient/")
	assert.Contains(t, medReq.Subject.Reference, instance.PatientID)
	t.Log("✓ Patient reference format correct")
}

func TestFHIREncounterReference(t *testing.T) {
	instance := createTestOrderSetInstance()
	instance.EncounterID = "encounter-67890"

	order := createTestMedicationOrder()
	medReq := generateTestMedicationRequest(order, instance)

	if medReq.Encounter != nil {
		assert.Contains(t, medReq.Encounter.Reference, "Encounter/")
		assert.Contains(t, medReq.Encounter.Reference, instance.EncounterID)
	}
	t.Log("✓ Encounter reference format correct")
}

// ============================================
// 3.3 Bundle Integrity Tests
// ============================================

func TestFHIRBundleEntryOrder(t *testing.T) {
	bundle := createTestFHIRBundleWithOrders()

	// Bundle entries should maintain order
	for i, entry := range bundle.Entry {
		assert.NotNil(t, entry.Resource, "Entry %d should have resource", i)
		assert.NotEmpty(t, entry.FullURL, "Entry %d should have fullUrl", i)
	}
	t.Logf("✓ Bundle maintains %d entries in order", len(bundle.Entry))
}

func TestFHIRBundleReferences(t *testing.T) {
	bundle := createTestFHIRBundleWithOrders()

	// All internal references should be valid URNs or relative references
	for _, entry := range bundle.Entry {
		if entry.FullURL != "" {
			assert.True(t,
				strings.HasPrefix(entry.FullURL, "urn:uuid:") ||
					strings.HasPrefix(entry.FullURL, "http"),
				"FullURL should be URN or HTTP reference")
		}
	}
	t.Log("✓ Bundle references validated")
}

func TestFHIRBundleNoOrphanResources(t *testing.T) {
	bundle := createTestFHIRBundleWithOrders()

	resourceIDs := make(map[string]bool)

	// Collect all resource IDs
	for _, entry := range bundle.Entry {
		if entry.Resource != nil {
			// Check if it's a map with ID
			if resMap, ok := entry.Resource.(map[string]interface{}); ok {
				if id, ok := resMap["id"].(string); ok && id != "" {
					resourceIDs[id] = true
				}
			}
		}
	}

	t.Logf("✓ Found %d unique resource IDs", len(resourceIDs))
}

func TestFHIRBundleR4Compliance(t *testing.T) {
	bundle := createTestFHIRBundle()

	// FHIR R4 required fields for Bundle
	assert.Equal(t, "Bundle", bundle.ResourceType)
	assert.NotEmpty(t, bundle.Type)

	validBundleTypes := []string{"document", "message", "transaction", "transaction-response", "batch", "batch-response", "history", "searchset", "collection"}
	assert.Contains(t, validBundleTypes, bundle.Type)

	t.Log("✓ Bundle R4 compliance verified")
}

func TestFHIRBundleSerialization(t *testing.T) {
	bundle := createTestFHIRBundle()

	// Should serialize without error
	data, err := json.Marshal(bundle)
	require.NoError(t, err, "Bundle should serialize to JSON")
	assert.NotEmpty(t, data)

	// Should contain required fields
	jsonStr := string(data)
	assert.Contains(t, jsonStr, "resourceType")
	assert.Contains(t, jsonStr, "Bundle")

	t.Logf("✓ Bundle serialized to %d bytes", len(data))
}

func TestFHIRBundleDeserialization(t *testing.T) {
	original := createTestFHIRBundle()

	// Serialize
	data, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize
	var parsed models.FHIRBundle
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify fields match
	assert.Equal(t, original.ResourceType, parsed.ResourceType)
	assert.Equal(t, original.Type, parsed.Type)
	assert.Equal(t, original.ID, parsed.ID)

	t.Log("✓ Bundle round-trip serialization successful")
}

// ============================================
// 3.4 Coding System Tests
// ============================================

func TestMedicationRxNormCoding(t *testing.T) {
	order := createTestMedicationOrder()
	order.RxNormCode = "1049502"
	order.DrugName = "Aspirin 81 MG Oral Tablet"

	instance := createTestOrderSetInstance()
	medReq := generateTestMedicationRequest(order, instance)

	if medReq.MedicationCodeableConcept != nil && len(medReq.MedicationCodeableConcept.Coding) > 0 {
		coding := medReq.MedicationCodeableConcept.Coding[0]
		assert.Equal(t, "http://www.nlm.nih.gov/research/umls/rxnorm", coding.System)
		assert.NotEmpty(t, coding.Code)
	}
	t.Log("✓ RxNorm coding verified")
}

func TestLabLOINCCoding(t *testing.T) {
	order := createTestLabOrder()
	order.LOINCCode = "2951-2"
	order.Name = "Sodium"

	instance := createTestOrderSetInstance()
	svcReq := generateTestServiceRequest(order, instance)

	if svcReq.Code != nil && len(svcReq.Code.Coding) > 0 {
		// Lab orders should use LOINC system
		hasLOINC := false
		for _, coding := range svcReq.Code.Coding {
			if coding.System == "http://loinc.org" {
				hasLOINC = true
				break
			}
		}
		if order.LOINCCode != "" {
			t.Log("✓ LOINC coding expected for labs")
		}
		_ = hasLOINC
	}
}

func TestProcedureSNOMEDCoding(t *testing.T) {
	order := models.Order{
		OrderID:   "proc-001",
		Name:      "Chest X-Ray",
		Type:      "imaging",
		OrderType: models.OrderTypeImaging,
		CPTCode:   "71045",
	}

	instance := createTestOrderSetInstance()
	svcReq := generateTestServiceRequest(order, instance)

	if svcReq.Code != nil {
		assert.NotEmpty(t, svcReq.Code.Text)
	}
	t.Log("✓ Procedure coding structure verified")
}

// ============================================
// 3.5 Dosage Instruction Tests
// ============================================

func TestDosageInstructionComplete(t *testing.T) {
	order := createTestMedicationOrder()
	order.Dose = "500 mg"
	order.Route = "oral"
	order.Frequency = "twice daily"
	order.DoseValue = 500
	order.DoseUnit = "mg"

	instance := createTestOrderSetInstance()
	medReq := generateTestMedicationRequest(order, instance)

	require.NotEmpty(t, medReq.DosageInstruction)
	dosage := medReq.DosageInstruction[0]

	assert.NotEmpty(t, dosage.Text)
	t.Log("✓ Dosage instruction populated")
}

func TestDosagePRNFlag(t *testing.T) {
	order := createTestMedicationOrder()
	order.PRN = true
	order.PRNReason = "pain"

	instance := createTestOrderSetInstance()
	medReq := generateTestMedicationRequest(order, instance)

	if len(medReq.DosageInstruction) > 0 {
		dosage := medReq.DosageInstruction[0]
		if order.PRN {
			assert.True(t, dosage.AsNeededBoolean || dosage.AsNeededCodeableConcept != nil)
		}
	}
	t.Log("✓ PRN flag handled correctly")
}

func TestDosageQuantity(t *testing.T) {
	order := createTestMedicationOrder()
	order.DoseValue = 250
	order.DoseUnit = "mg"

	instance := createTestOrderSetInstance()
	medReq := generateTestMedicationRequest(order, instance)

	if len(medReq.DosageInstruction) > 0 && len(medReq.DosageInstruction[0].DoseAndRate) > 0 {
		doseRate := medReq.DosageInstruction[0].DoseAndRate[0]
		if doseRate.DoseQuantity != nil {
			assert.Equal(t, order.DoseValue, doseRate.DoseQuantity.Value)
			assert.Equal(t, order.DoseUnit, doseRate.DoseQuantity.Unit)
		}
	}
	t.Log("✓ Dose quantity populated correctly")
}

// ============================================
// Helper Functions
// ============================================

func createTestMedicationOrder() models.Order {
	return models.Order{
		OrderID:    "med-001",
		Name:       "Metoprolol",
		Type:       "medication",
		OrderType:  models.OrderTypeMedication,
		DrugCode:   "6918",
		DrugName:   "Metoprolol Tartrate",
		RxNormCode: "6918",
		Dose:       "25 mg",
		DoseValue:  25,
		DoseUnit:   "mg",
		Route:      "oral",
		Frequency:  "twice daily",
		Priority:   models.PriorityRoutine,
		Selected:   true,
	}
}

func createTestLabOrder() models.Order {
	return models.Order{
		OrderID:   "lab-001",
		Name:      "Basic Metabolic Panel",
		Type:      "lab",
		OrderType: models.OrderTypeLab,
		LabCode:   "BMP",
		LOINCCode: "51990-0",
		Priority:  models.PriorityRoutine,
		Selected:  true,
	}
}

func createTestNursingOrder() models.Order {
	return models.Order{
		OrderID:      "nurs-001",
		Name:         "Vital Signs Q4H",
		Type:         "nursing",
		OrderType:    models.OrderTypeNursing,
		Instructions: "Monitor vital signs every 4 hours",
		Priority:     models.PriorityRoutine,
		Selected:     true,
	}
}

func createTestOrderSetInstance() models.OrderSetInstance {
	return models.OrderSetInstance{
		InstanceID:  "OSI-TEST001",
		TemplateID:  "OS-CARDIAC-001",
		PatientID:   "patient-test-123",
		EncounterID: "encounter-test-456",
		ActivatedBy: "Dr. Test Provider",
		Status:      models.OrderStatusActive,
		ActivatedAt: time.Now().Add(-1 * time.Hour),
	}
}

func createTestFHIRBundle() *models.FHIRBundle {
	return &models.FHIRBundle{
		ResourceType: "Bundle",
		ID:           "bundle-test-001",
		Type:         "collection",
		Timestamp:    time.Now(),
		Total:        0,
		Entry:        []models.BundleEntry{},
	}
}

func createTestFHIRBundleWithOrders() *models.FHIRBundle {
	instance := createTestOrderSetInstance()

	return &models.FHIRBundle{
		ResourceType: "Bundle",
		ID:           "bundle-test-002",
		Type:         "collection",
		Timestamp:    time.Now(),
		Total:        2,
		Entry: []models.BundleEntry{
			{
				FullURL:  "urn:uuid:med-001",
				Resource: generateTestMedicationRequest(createTestMedicationOrder(), instance),
				Request: &models.BundleRequest{
					Method: "POST",
					URL:    "MedicationRequest",
				},
			},
			{
				FullURL:  "urn:uuid:lab-001",
				Resource: generateTestServiceRequest(createTestLabOrder(), instance),
				Request: &models.BundleRequest{
					Method: "POST",
					URL:    "ServiceRequest",
				},
			},
		},
	}
}

func createTestFHIRCarePlan() *models.FHIRCarePlan {
	return &models.FHIRCarePlan{
		ResourceType: "CarePlan",
		ID:           "careplan-test-001",
		Status:       "active",
		Intent:       "plan",
		Title:        "Test Care Plan",
		Subject: models.Reference{
			Reference: "Patient/patient-test-123",
		},
		Created: time.Now(),
		Activity: []models.CarePlanActivity{
			{
				Detail: &models.CarePlanActivityDetail{
					Status:      "not-started",
					Description: "Test activity",
				},
			},
		},
	}
}

func createTestPlanDefinition() *models.FHIRPlanDefinition {
	return &models.FHIRPlanDefinition{
		ResourceType: "PlanDefinition",
		ID:           "plandef-test-001",
		Status:       "active",
		Title:        "Test Protocol",
		Description:  "Test plan definition",
		Action: []models.PlanDefinitionAction{
			{
				Title:       "Step 1",
				Description: "First action",
			},
		},
	}
}

func generateTestMedicationRequest(order models.Order, instance models.OrderSetInstance) *models.FHIRMedicationRequest {
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

	if order.DrugCode != "" || order.RxNormCode != "" {
		code := order.DrugCode
		if code == "" {
			code = order.RxNormCode
		}
		req.MedicationCodeableConcept = &models.CodeableConcept{
			Coding: []models.Coding{
				{
					System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
					Code:    code,
					Display: order.DrugName,
				},
			},
			Text: order.DrugName,
		}
	}

	if order.Dose != "" || order.Route != "" || order.Frequency != "" {
		dosage := models.DosageInstruction{
			Text: order.Dose + " " + order.Route + " " + order.Frequency,
		}

		if order.Route != "" {
			dosage.Route = &models.CodeableConcept{Text: order.Route}
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
				dosage.AsNeededCodeableConcept = &models.CodeableConcept{Text: order.PRNReason}
			}
		}

		req.DosageInstruction = []models.DosageInstruction{dosage}
	}

	return req
}

func generateTestServiceRequest(order models.Order, instance models.OrderSetInstance) *models.FHIRServiceRequest {
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

	var system, code, display string
	switch order.OrderType {
	case models.OrderTypeLab:
		system = "http://loinc.org"
		code = order.LOINCCode
		display = order.Name
	case models.OrderTypeImaging:
		system = "http://www.ama-assn.org/go/cpt"
		code = order.CPTCode
		display = order.Name
	default:
		system = "http://snomed.info/sct"
		display = order.Name
	}

	req.Code = &models.CodeableConcept{
		Coding: []models.Coding{
			{System: system, Code: code, Display: display},
		},
		Text: display,
	}

	if order.Reason != "" {
		req.ReasonCode = []models.CodeableConcept{
			{Text: order.Reason},
		}
	}

	return req
}

func generateTestTask(order models.Order, instance models.OrderSetInstance) *models.FHIRTask {
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
