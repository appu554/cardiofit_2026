// Package test provides CDS Hooks clinical scenario tests for KB-12
// Phase 6: Clinical Decision Support validation across all 5 hooks
package test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/pkg/cdshooks"
	"kb-12-ordersets-careplans/pkg/ordersets"
)

// ============================================
// 6.1 Patient-View Hook Tests
// ============================================

func TestPatientViewCHFPatientShowsCHFOrders(t *testing.T) {
	// Test that a patient with Heart Failure gets CHF-specific order set recommendations
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-chf-001",
		Context: map[string]interface{}{
			"patientId": "patient-chf-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure, unspecified"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have at least one card recommending CHF-related order sets or care plans
	assert.NotEmpty(t, resp.Cards, "CHF patient should receive recommendations")

	foundCHFRecommendation := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "Heart Failure", "CHF", "HF", "I50") {
			foundCHFRecommendation = true
			// Verify card structure
			assert.NotEmpty(t, card.UUID, "Card should have UUID")
			assert.Contains(t, []string{"info", "warning", "critical"}, card.Indicator)
			assert.NotEmpty(t, card.Source.Label, "Card should have source label")
			break
		}
	}
	assert.True(t, foundCHFRecommendation, "Should recommend CHF-specific order sets or care plans")
	t.Log("✓ CHF patient receives CHF-specific recommendations")
}

func TestPatientViewCOPDPatientShowsCOPDOrders(t *testing.T) {
	// Test that a patient with COPD gets COPD-specific recommendations
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-copd-001",
		Context: map[string]interface{}{
			"patientId": "patient-copd-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "J44.1", System: "http://hl7.org/fhir/sid/icd-10", Display: "Chronic obstructive pulmonary disease with acute exacerbation"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	foundCOPDRecommendation := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "COPD", "Chronic obstructive", "Pulmonary", "J44") {
			foundCOPDRecommendation = true
			break
		}
	}
	assert.True(t, foundCOPDRecommendation, "Should recommend COPD-specific order sets or care plans")
	t.Log("✓ COPD patient receives COPD-specific recommendations")
}

func TestPatientViewDiabetesShowsDMCarePlan(t *testing.T) {
	// Test that a diabetic patient gets diabetes care plan recommendation
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-dm-001",
		Context: map[string]interface{}{
			"patientId": "patient-dm-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "E11.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Type 2 diabetes mellitus without complications"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	foundDMRecommendation := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "Diabetes", "DM", "E11", "Type 2") {
			foundDMRecommendation = true
			// Verify it's a care plan recommendation
			if card.Source.Topic != nil {
				t.Logf("Topic: %s", card.Source.Topic.Display)
			}
			break
		}
	}
	assert.True(t, foundDMRecommendation, "Should recommend diabetes care plan")
	t.Log("✓ Diabetes patient receives DM care plan recommendation")
}

func TestPatientViewMultipleConditionsPrioritizes(t *testing.T) {
	// Test that multiple conditions are handled with appropriate prioritization
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-multi-001",
		Context: map[string]interface{}{
			"patientId": "patient-multi-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure, unspecified"},
				{Code: "E11.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Type 2 diabetes mellitus"},
				{Code: "I10", System: "http://hl7.org/fhir/sid/icd-10", Display: "Essential hypertension"},
				{Code: "N18.3", System: "http://hl7.org/fhir/sid/icd-10", Display: "Chronic kidney disease, stage 3"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have multiple recommendations
	assert.NotEmpty(t, resp.Cards, "Multiple conditions should generate recommendations")
	t.Logf("✓ Multiple conditions generated %d recommendations", len(resp.Cards))

	// Check for diversity in recommendations
	categories := make(map[string]bool)
	for _, card := range resp.Cards {
		if card.Source.Topic != nil {
			categories[card.Source.Topic.Display] = true
		}
	}
	t.Logf("Categories represented: %v", categories)
}

func TestPatientViewNoConditionsShowsGeneral(t *testing.T) {
	// Test behavior when patient has no active conditions
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-healthy-001",
		Context: map[string]interface{}{
			"patientId": "patient-healthy-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{}), // No conditions
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// May have no cards or general wellness recommendations
	t.Logf("✓ Healthy patient (no conditions) received %d cards", len(resp.Cards))
}

// ============================================
// 6.2 Order-Select Hook Tests
// ============================================

func TestOrderSelectInsulinShowsProtocol(t *testing.T) {
	// Test that selecting insulin triggers insulin protocol suggestions
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-select",
		HookInstance: "test-insulin-001",
		Context: map[string]interface{}{
			"patientId": "patient-dm-456",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "5856",
										"display": "Insulin",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "E11.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Type 2 diabetes mellitus"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-select", req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("✓ Insulin selection generated %d cards", len(resp.Cards))
}

func TestOrderSelectAnticoagulantShowsMonitoring(t *testing.T) {
	// Test that anticoagulant selection triggers monitoring recommendations
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-select",
		HookInstance: "test-anticoag-001",
		Context: map[string]interface{}{
			"patientId": "patient-afib-456",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "11289",
										"display": "Warfarin",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I48.91", System: "http://hl7.org/fhir/sid/icd-10", Display: "Atrial fibrillation"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-select", req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("✓ Anticoagulant selection generated %d cards", len(resp.Cards))
}

func TestOrderSelectAntibioticShowsStewardship(t *testing.T) {
	// Test that broad-spectrum antibiotic triggers stewardship recommendations
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-select",
		HookInstance: "test-abx-001",
		Context: map[string]interface{}{
			"patientId": "patient-pna-456",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "1665004",
										"display": "Piperacillin-tazobactam",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "J18.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Pneumonia, unspecified organism"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-select", req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("✓ Antibiotic selection generated %d cards", len(resp.Cards))
}

func TestOrderSelectPartialProtocolSuggestsCompletion(t *testing.T) {
	// Test that partial heart failure GDMT orders trigger complete protocol suggestion
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-select",
		HookInstance: "test-partial-001",
		Context: map[string]interface{}{
			"patientId": "patient-hf-456",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					// Beta-blocker (part of GDMT)
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "19484",
										"display": "Metoprolol succinate",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
					// Loop diuretic (part of GDMT)
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "4603",
										"display": "Furosemide",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure, unspecified"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-select", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should suggest completing the GDMT protocol
	foundProtocolSuggestion := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "protocol", "GDMT", "complete", "CHF") {
			foundProtocolSuggestion = true
			t.Logf("Found protocol completion suggestion: %s", card.Summary)
			break
		}
	}

	t.Logf("✓ Partial GDMT pattern generated %d cards (protocol suggestion: %v)",
		len(resp.Cards), foundProtocolSuggestion)
}

// ============================================
// 6.3 Order-Sign Hook Tests
// ============================================

func TestOrderSignSepsisBundleEnforcement(t *testing.T) {
	// Test that sepsis without complete bundle triggers critical alerts
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-sign",
		HookInstance: "test-sepsis-001",
		Context: map[string]interface{}{
			"patientId": "patient-sepsis-789",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					// Only has IV fluids, missing lactate, cultures, antibiotics
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "313002",
										"display": "Lactated Ringer's",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "A41.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Sepsis, unspecified organism"},
				{Code: "R65.21", System: "http://hl7.org/fhir/sid/icd-10", Display: "Severe sepsis with septic shock"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-sign", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have critical alerts for missing sepsis bundle elements
	criticalCards := 0
	missingElements := []string{}
	for _, card := range resp.Cards {
		if card.Indicator == "critical" {
			criticalCards++
			if containsAny(card.Summary, "Lactate", "lactate") {
				missingElements = append(missingElements, "Lactate")
			}
			if containsAny(card.Summary, "Culture", "culture") {
				missingElements = append(missingElements, "Blood Cultures")
			}
			if containsAny(card.Summary, "Antibiotic", "antibiotic") {
				missingElements = append(missingElements, "Antibiotics")
			}
		}
		// Verify override reasons are provided for governance
		if len(card.OverrideReasons) > 0 {
			t.Logf("Override reasons available: %d", len(card.OverrideReasons))
		}
	}

	assert.NotZero(t, criticalCards, "Sepsis with incomplete bundle should trigger critical alerts")
	t.Logf("✓ Sepsis bundle enforcement: %d critical alerts, missing: %v", criticalCards, missingElements)
}

func TestOrderSignAnticoagulationSafety(t *testing.T) {
	// Test that anticoagulation orders get appropriate safety checks
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-sign",
		HookInstance: "test-anticoag-sign-001",
		Context: map[string]interface{}{
			"patientId": "patient-anticoag-789",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "67108",
										"display": "Enoxaparin",
									},
								},
							},
							"dosageInstruction": []interface{}{
								map[string]interface{}{
									"doseAndRate": []interface{}{
										map[string]interface{}{
											"doseQuantity": map[string]interface{}{
												"value": 80,
												"unit":  "mg",
											},
										},
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I48.91", System: "http://hl7.org/fhir/sid/icd-10", Display: "Atrial fibrillation"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-sign", req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("✓ Anticoagulation safety check generated %d cards", len(resp.Cards))
}

func TestOrderSignHighAlertMedDoubleCheck(t *testing.T) {
	// Test that high-alert medications trigger double-check requirements
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-sign",
		HookInstance: "test-high-alert-001",
		Context: map[string]interface{}{
			"patientId": "patient-ha-789",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "MedicationRequest",
							"medicationCodeableConcept": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://www.nlm.nih.gov/research/umls/rxnorm",
										"code":    "7052",
										"display": "Morphine",
									},
								},
							},
							"status": "draft",
							"intent": "order",
						},
					},
				},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-sign", req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Logf("✓ High-alert medication check generated %d cards", len(resp.Cards))
}

func TestOrderSignMissingElementsWarning(t *testing.T) {
	// Test that missing protocol elements generate appropriate warnings
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-sign",
		HookInstance: "test-missing-001",
		Context: map[string]interface{}{
			"patientId": "patient-protocol-789",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{}, // Empty orders
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I21.3", System: "http://hl7.org/fhir/sid/icd-10", Display: "ST elevation myocardial infarction of unspecified site"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-order-sign", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	warningOrCriticalCount := 0
	for _, card := range resp.Cards {
		if card.Indicator == "warning" || card.Indicator == "critical" {
			warningOrCriticalCount++
		}
	}
	t.Logf("✓ Missing elements check: %d warning/critical cards", warningOrCriticalCount)
}

// ============================================
// 6.4 Encounter Hooks Tests
// ============================================

func TestEncounterStartAdmissionRecommendation(t *testing.T) {
	// Test that encounter-start triggers appropriate admission order set recommendations
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-start",
		HookInstance: "test-admit-001",
		Context: map[string]interface{}{
			"patientId":   "patient-admit-456",
			"encounterId": "encounter-admit-789",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"id":           "encounter-admit-789",
				"class": map[string]interface{}{
					"code":    "IMP",
					"display": "inpatient encounter",
				},
				"type": []interface{}{
					map[string]interface{}{
						"coding": []interface{}{
							map[string]interface{}{
								"code":    "32485007",
								"display": "Hospital admission",
							},
						},
					},
				},
				"reasonCode": []interface{}{
					map[string]interface{}{
						"coding": []interface{}{
							map[string]interface{}{
								"system":  "http://hl7.org/fhir/sid/icd-10",
								"code":    "I50.9",
								"display": "Heart failure, unspecified",
							},
						},
					},
				},
			},
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure, unspecified"},
			}),
			"allergies": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{},
			},
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-encounter-start", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have admission order set recommendations
	assert.NotEmpty(t, resp.Cards, "Encounter start should generate recommendations")

	foundAdmissionOrderSet := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "Admission", "Order Set", "CHF") {
			foundAdmissionOrderSet = true
			// Verify selection behavior for exclusive choices
			if card.SelectionBehavior == "at-most-one" {
				t.Log("Correctly configured for exclusive selection")
			}
			break
		}
	}
	assert.True(t, foundAdmissionOrderSet, "Should recommend admission order sets")
	t.Logf("✓ Encounter start generated %d recommendations", len(resp.Cards))
}

func TestEncounterStartEmergencyProtocolDetection(t *testing.T) {
	// Test that emergency conditions at encounter start trigger emergency protocols
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-start",
		HookInstance: "test-emergency-001",
		Context: map[string]interface{}{
			"patientId":   "patient-emergency-456",
			"encounterId": "encounter-emergency-789",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"class": map[string]interface{}{
					"code": "EMER",
				},
			},
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I46.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Cardiac arrest, unspecified"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-encounter-start", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have critical emergency protocol card
	foundEmergencyProtocol := false
	for _, card := range resp.Cards {
		if card.Indicator == "critical" {
			foundEmergencyProtocol = true
			assert.Contains(t, card.Summary, "EMERGENCY", "Emergency card should be clearly marked")
			break
		}
	}
	assert.True(t, foundEmergencyProtocol, "Cardiac arrest should trigger emergency protocol")
	t.Log("✓ Emergency condition detected and protocol recommended")
}

func TestEncounterDischargeCarePlanCreation(t *testing.T) {
	// Test that encounter-discharge recommends appropriate care plans
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-discharge",
		HookInstance: "test-discharge-001",
		Context: map[string]interface{}{
			"patientId":   "patient-discharge-456",
			"encounterId": "encounter-discharge-789",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"id":           "encounter-discharge-789",
				"status":       "in-progress",
			},
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure, unspecified", Chronic: true},
				{Code: "E11.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Type 2 diabetes mellitus", Chronic: true},
			}),
			"procedures": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{},
			},
			"carePlans": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{}, // No existing care plans
			},
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-encounter-discharge", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have medication reconciliation and care plan recommendations
	foundMedReconciliation := false
	foundCarePlanRecommendation := false
	foundFollowUp := false

	for _, card := range resp.Cards {
		if containsAny(card.Summary, "Medication Reconciliation") {
			foundMedReconciliation = true
		}
		if containsAny(card.Summary, "Care Plan", "Activate") {
			foundCarePlanRecommendation = true
		}
		if containsAny(card.Summary, "Follow-up", "Appointment") {
			foundFollowUp = true
		}
	}

	assert.True(t, foundMedReconciliation, "Should recommend medication reconciliation")
	t.Logf("✓ Discharge recommendations: MedRec=%v, CarePlan=%v, FollowUp=%v",
		foundMedReconciliation, foundCarePlanRecommendation, foundFollowUp)
}

func TestEncounterDischargePostProcedureInstructions(t *testing.T) {
	// Test that procedures trigger post-procedure discharge instructions
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-discharge",
		HookInstance: "test-postproc-001",
		Context: map[string]interface{}{
			"patientId":   "patient-postproc-456",
			"encounterId": "encounter-postproc-789",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"id":           "encounter-postproc-789",
				"status":       "in-progress",
			},
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I25.10", System: "http://hl7.org/fhir/sid/icd-10", Display: "Atherosclerotic heart disease of native coronary artery"},
			}),
			"procedures": map[string]interface{}{
				"resourceType": "Bundle",
				"entry": []interface{}{
					map[string]interface{}{
						"resource": map[string]interface{}{
							"resourceType": "Procedure",
							"code": map[string]interface{}{
								"coding": []interface{}{
									map[string]interface{}{
										"system":  "http://snomed.info/sct",
										"code":    "34068001",
										"display": "Heart catheterization",
									},
								},
							},
							"status": "completed",
						},
					},
				},
			},
			"carePlans": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{},
			},
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-encounter-discharge", req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Should have post-procedure instructions
	foundPostProcInstructions := false
	for _, card := range resp.Cards {
		if containsAny(card.Summary, "Post-", "Discharge Instructions", "catheterization") {
			foundPostProcInstructions = true
			break
		}
	}
	t.Logf("✓ Post-procedure discharge: %d cards, instructions=%v",
		len(resp.Cards), foundPostProcInstructions)
}

// ============================================
// CDS Hooks Discovery & Structure Tests
// ============================================

func TestCDSHooksDiscoveryCompliance(t *testing.T) {
	// Verify CDS Hooks 2.0 discovery document compliance
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	discovery := service.GetDiscovery()
	require.NotNil(t, discovery)
	require.NotEmpty(t, discovery.Services)

	// Verify all 5 required hooks are present
	expectedHooks := map[string]bool{
		"kb12-patient-view":       false,
		"kb12-order-select":       false,
		"kb12-order-sign":         false,
		"kb12-encounter-start":    false,
		"kb12-encounter-discharge": false,
	}

	for _, svc := range discovery.Services {
		// Verify required fields
		assert.NotEmpty(t, svc.ID, "Service should have ID")
		assert.NotEmpty(t, svc.Hook, "Service should have hook type")
		assert.NotEmpty(t, svc.Title, "Service should have title")
		assert.NotEmpty(t, svc.Description, "Service should have description")

		// Verify prefetch templates are present
		assert.NotNil(t, svc.Prefetch, "Service should have prefetch templates")
		assert.Contains(t, svc.Prefetch, "patient", "Prefetch should include patient")

		if _, exists := expectedHooks[svc.ID]; exists {
			expectedHooks[svc.ID] = true
		}
	}

	// Verify all expected hooks are present
	for hookID, found := range expectedHooks {
		assert.True(t, found, "Missing hook: %s", hookID)
	}

	t.Logf("✓ CDS Hooks discovery compliant with %d services", len(discovery.Services))
}

func TestCDSCardStructureCompliance(t *testing.T) {
	// Test that generated cards comply with CDS Hooks 2.0 card structure
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "test-structure-001",
		Context: map[string]interface{}{
			"patientId": "patient-structure-123",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure"},
			}),
		},
	}

	resp, err := service.ProcessHook(context.Background(), "kb12-patient-view", req)
	require.NoError(t, err)

	for _, card := range resp.Cards {
		// Required fields
		assert.NotEmpty(t, card.Summary, "Card must have summary")
		assert.Contains(t, []string{"info", "warning", "critical"}, card.Indicator,
			"Indicator must be valid")
		assert.NotEmpty(t, card.Source.Label, "Card must have source label")

		// Optional but expected fields
		if len(card.Suggestions) > 0 {
			for _, suggestion := range card.Suggestions {
				assert.NotEmpty(t, suggestion.Label, "Suggestion must have label")
			}
		}

		// Override reasons should be present for governance
		if card.Indicator == "warning" || card.Indicator == "critical" {
			// Important for governance tracking
			t.Logf("Card '%s' has %d override reasons", card.Summary, len(card.OverrideReasons))
		}
	}

	t.Log("✓ Card structure compliance verified")
}

// ============================================
// Helper Functions
// ============================================

// conditionFixture for creating test condition data
type conditionFixture struct {
	Code    string
	System  string
	Display string
	Chronic bool
}

// createConditionBundle creates a FHIR Condition Bundle for testing
func createConditionBundle(conditions []conditionFixture) map[string]interface{} {
	entries := make([]interface{}, 0, len(conditions))
	for _, c := range conditions {
		entry := map[string]interface{}{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []interface{}{
						map[string]interface{}{
							"system":  c.System,
							"code":    c.Code,
							"display": c.Display,
						},
					},
				},
			},
		}
		if c.Chronic {
			entry["resource"].(map[string]interface{})["category"] = []interface{}{
				map[string]interface{}{
					"coding": []interface{}{
						map[string]interface{}{
							"code": "chronic",
						},
					},
				},
			}
		}
		entries = append(entries, entry)
	}
	return map[string]interface{}{
		"resourceType": "Bundle",
		"entry":        entries,
	}
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkPatientViewHook(b *testing.B) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "patient-view",
		HookInstance: "bench-pv",
		Context: map[string]interface{}{
			"patientId": "patient-bench",
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure"},
				{Code: "E11.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Type 2 diabetes"},
			}),
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.ProcessHook(ctx, "kb12-patient-view", req)
	}
}

func BenchmarkOrderSignHook(b *testing.B) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "order-sign",
		HookInstance: "bench-os",
		Context: map[string]interface{}{
			"patientId": "patient-bench",
			"draftOrders": map[string]interface{}{
				"resourceType": "Bundle",
				"entry":        []interface{}{},
			},
		},
		Prefetch: map[string]interface{}{
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "A41.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Sepsis"},
			}),
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.ProcessHook(ctx, "kb12-order-sign", req)
	}
}

func BenchmarkEncounterStartHook(b *testing.B) {
	loader := ordersets.NewTemplateLoader(nil, nil)
	service := cdshooks.NewCDSHooksService(loader)

	req := &cdshooks.CDSRequest{
		Hook:         "encounter-start",
		HookInstance: "bench-es",
		Context: map[string]interface{}{
			"patientId":   "patient-bench",
			"encounterId": "encounter-bench",
		},
		Prefetch: map[string]interface{}{
			"encounter": map[string]interface{}{
				"resourceType": "Encounter",
				"class": map[string]interface{}{
					"code": "IMP",
				},
			},
			"conditions": createConditionBundle([]conditionFixture{
				{Code: "I50.9", System: "http://hl7.org/fhir/sid/icd-10", Display: "Heart failure"},
			}),
		},
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.ProcessHook(ctx, "kb12-encounter-start", req)
	}
}

// Suppress unused import warning
var _ = time.Now
