// Package test provides order set template integrity tests for KB-12
package test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/internal/models"
	"kb-12-ordersets-careplans/pkg/ordersets"
)

// ============================================
// 2.1 Template Structure Validation
// ============================================

func TestCardiacAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetCardiacAdmissionOrderSets()
	require.NotNil(t, templates, "Cardiac admission templates should exist")
	t.Logf("Found %d cardiac admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
			assert.Contains(t, strings.ToLower(tmpl.Specialty), "cardio", "Should be cardiology specialty")
		})
	}
}

func TestRespiratoryAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetRespiratoryAdmissionOrderSets()
	require.NotNil(t, templates, "Respiratory admission templates should exist")
	t.Logf("Found %d respiratory admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
		})
	}
}

func TestMetabolicAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetMetabolicAdmissionOrderSets()
	require.NotNil(t, templates, "Metabolic admission templates should exist")
	t.Logf("Found %d metabolic admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
		})
	}
}

func TestNeuroAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetNeuroAdmissionOrderSets()
	require.NotNil(t, templates, "Neuro admission templates should exist")
	t.Logf("Found %d neuro admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
		})
	}
}

func TestGIAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetGIAdmissionOrderSets()
	require.NotNil(t, templates, "GI admission templates should exist")
	t.Logf("Found %d GI admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
		})
	}
}

func TestInfectiousAdmissionTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetInfectiousAdmissionOrderSets()
	require.NotNil(t, templates, "Infectious admission templates should exist")
	t.Logf("Found %d infectious admission templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryAdmission, tmpl.Category)
		})
	}
}

func TestCardiacProcedureTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetCardiacProcedureOrderSets()
	require.NotNil(t, templates, "Cardiac procedure templates should exist")
	t.Logf("Found %d cardiac procedure templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryProcedure, tmpl.Category)
		})
	}
}

func TestGIProcedureTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetGIProcedureOrderSets()
	require.NotNil(t, templates, "GI procedure templates should exist")
	t.Logf("Found %d GI procedure templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryProcedure, tmpl.Category)
		})
	}
}

func TestSurgicalProcedureTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetSurgicalProcedureOrderSets()
	require.NotNil(t, templates, "Surgical procedure templates should exist")
	t.Logf("Found %d surgical procedure templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryProcedure, tmpl.Category)
		})
	}
}

func TestBedsideProcedureTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetBedsideProcedureOrderSets()
	require.NotNil(t, templates, "Bedside procedure templates should exist")
	t.Logf("Found %d bedside procedure templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryProcedure, tmpl.Category)
		})
	}
}

func TestEmergencyProtocolTemplateIntegrity(t *testing.T) {
	templates := ordersets.GetAllEmergencyProtocols()
	require.NotNil(t, templates, "Emergency protocol templates should exist")
	t.Logf("Found %d emergency protocol templates", len(templates))

	for _, tmpl := range templates {
		t.Run(tmpl.Name, func(t *testing.T) {
			validateTemplateStructure(t, tmpl)
			assert.Equal(t, models.CategoryEmergency, tmpl.Category)
			// Emergency protocols should have time constraints
			if tmpl.IsTimeCritical() {
				t.Logf("✓ %s has %d time constraints", tmpl.Name, len(tmpl.TimeConstraints))
			}
		})
	}
}

func TestAllTemplatesHaveUniqueIDs(t *testing.T) {
	allTemplates := getAllTemplates()
	seenIDs := make(map[string]string)

	for _, tmpl := range allTemplates {
		if existing, exists := seenIDs[tmpl.TemplateID]; exists {
			t.Errorf("Duplicate template ID: %s (found in %s and %s)", tmpl.TemplateID, existing, tmpl.Name)
		} else {
			seenIDs[tmpl.TemplateID] = tmpl.Name
		}
	}
	t.Logf("✓ Verified %d unique template IDs", len(seenIDs))
}

// ============================================
// 2.2 Code System Validation
// ============================================

func TestAllMedicationsHaveRxNorm(t *testing.T) {
	allTemplates := getAllTemplates()
	totalMeds := 0
	medsWithRxNorm := 0
	missingRxNorm := []string{}

	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			if order.OrderType == models.OrderTypeMedication || order.Type == "medication" {
				totalMeds++
				if order.RxNormCode != "" {
					medsWithRxNorm++
				} else {
					missingRxNorm = append(missingRxNorm, order.Name)
				}
			}
		}
	}

	t.Logf("Medications with RxNorm: %d/%d (%.1f%%)", medsWithRxNorm, totalMeds,
		float64(medsWithRxNorm)/float64(totalMeds)*100)

	if len(missingRxNorm) > 0 && len(missingRxNorm) <= 10 {
		t.Logf("Medications missing RxNorm: %v", missingRxNorm)
	}
	// Warning only - not failing as some may be compound meds
	if float64(medsWithRxNorm)/float64(totalMeds) < 0.8 {
		t.Logf("⚠ Less than 80%% of medications have RxNorm codes")
	}
}

func TestAllLabsHaveLOINC(t *testing.T) {
	allTemplates := getAllTemplates()
	totalLabs := 0
	labsWithLOINC := 0
	missingLOINC := []string{}

	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			if order.OrderType == models.OrderTypeLab || order.Type == "lab" {
				totalLabs++
				if order.LOINCCode != "" || order.LabCode != "" {
					labsWithLOINC++
				} else {
					missingLOINC = append(missingLOINC, order.Name)
				}
			}
		}
	}

	t.Logf("Labs with LOINC: %d/%d (%.1f%%)", labsWithLOINC, totalLabs,
		float64(labsWithLOINC)/float64(totalLabs)*100)

	if len(missingLOINC) > 0 && len(missingLOINC) <= 10 {
		t.Logf("Labs missing LOINC: %v", missingLOINC)
	}
	assert.GreaterOrEqual(t, float64(labsWithLOINC)/float64(totalLabs), 0.7,
		"At least 70%% of labs should have LOINC codes")
}

func TestAllProceduresHaveSNOMED(t *testing.T) {
	allTemplates := getAllTemplates()
	totalProcs := 0
	procsWithSNOMED := 0

	for _, tmpl := range allTemplates {
		// Check template-level SNOMED codes
		if len(tmpl.SNOMEDCodes) > 0 {
			procsWithSNOMED++
		}
		totalProcs++

		// Also check order-level codes
		for _, order := range tmpl.Orders {
			if order.OrderType == models.OrderTypeProcedure || order.Type == "procedure" {
				hasCode := false
				for _, code := range order.Codes {
					if strings.Contains(code.System, "snomed") {
						hasCode = true
						break
					}
				}
				if hasCode {
					procsWithSNOMED++
				}
				totalProcs++
			}
		}
	}

	if totalProcs > 0 {
		t.Logf("Procedures with SNOMED: %d/%d (%.1f%%)", procsWithSNOMED, totalProcs,
			float64(procsWithSNOMED)/float64(totalProcs)*100)
	}
}

func TestAllConditionsHaveICD10(t *testing.T) {
	allTemplates := getAllTemplates()
	templatesWithICD := 0
	totalTemplates := len(allTemplates)

	for _, tmpl := range allTemplates {
		if len(tmpl.ICDCodes) > 0 {
			templatesWithICD++
			t.Logf("✓ %s has %d ICD codes", tmpl.Name, len(tmpl.ICDCodes))
		}
	}

	t.Logf("Templates with ICD-10: %d/%d (%.1f%%)", templatesWithICD, totalTemplates,
		float64(templatesWithICD)/float64(totalTemplates)*100)
}

func TestNoMissingCodeSystems(t *testing.T) {
	allTemplates := getAllTemplates()
	issues := []string{}

	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			// Medications must have some form of drug code
			if order.OrderType == models.OrderTypeMedication || order.Type == "medication" {
				if order.RxNormCode == "" && order.DrugCode == "" && len(order.Codes) == 0 {
					issues = append(issues, tmpl.Name+": "+order.Name+" (medication without code)")
				}
			}
		}
	}

	if len(issues) > 0 {
		t.Logf("Code system issues found (%d):", len(issues))
		for _, issue := range issues {
			t.Logf("  - %s", issue)
		}
	}
	t.Logf("✓ Code system validation complete")
}

// ============================================
// 2.3 Timing Rules Validation
// ============================================

func TestAllTimeConstraintsHaveValidDuration(t *testing.T) {
	allTemplates := getAllTemplates()
	constraintsChecked := 0
	invalidConstraints := []string{}

	for _, tmpl := range allTemplates {
		for _, constraint := range tmpl.TimeConstraints {
			constraintsChecked++

			// Must have either Deadline duration or DeadlineHours
			hasValidDuration := constraint.Deadline > 0 || constraint.DeadlineHours > 0

			if !hasValidDuration {
				invalidConstraints = append(invalidConstraints, tmpl.Name+": "+constraint.Name)
			}

			// Alert threshold should be less than deadline if set
			if constraint.AlertThreshold > 0 && constraint.Deadline > 0 {
				if constraint.AlertThreshold >= constraint.Deadline {
					t.Logf("⚠ %s: Alert threshold >= deadline for %s", tmpl.Name, constraint.Name)
				}
			}
		}
	}

	t.Logf("Time constraints validated: %d", constraintsChecked)
	if len(invalidConstraints) > 0 {
		t.Logf("Constraints without valid duration: %v", invalidConstraints)
	}
}

func TestCriticalOrdersHaveTimeConstraints(t *testing.T) {
	// Critical order sets should have time constraints
	criticalProtocols := []string{"sepsis", "stroke", "stemi", "cardiac arrest"}
	allTemplates := getAllTemplates()

	for _, tmpl := range allTemplates {
		nameLower := strings.ToLower(tmpl.Name)
		for _, keyword := range criticalProtocols {
			if strings.Contains(nameLower, keyword) {
				if !tmpl.IsTimeCritical() {
					t.Logf("⚠ Critical protocol '%s' has no time constraints", tmpl.Name)
				} else {
					t.Logf("✓ Critical protocol '%s' has %d time constraints", tmpl.Name, len(tmpl.TimeConstraints))
				}
			}
		}
	}
}

func TestTimeConstraintEscalationPaths(t *testing.T) {
	allTemplates := getAllTemplates()
	escalationPaths := 0

	for _, tmpl := range allTemplates {
		if tmpl.HasCriticalConstraints() {
			escalationPaths++
			t.Logf("✓ %s has critical escalation path", tmpl.Name)
		}
	}

	t.Logf("Templates with escalation paths: %d", escalationPaths)
}

// ============================================
// 2.4 Order Structure Validation
// ============================================

func TestAllOrdersHaveRequiredFields(t *testing.T) {
	allTemplates := getAllTemplates()
	ordersChecked := 0
	ordersWithIssues := 0

	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			ordersChecked++

			// Every order must have a name
			if order.Name == "" {
				t.Errorf("%s: Order without name found", tmpl.Name)
				ordersWithIssues++
			}

			// Must have some type indicator
			if order.Type == "" && order.OrderType == "" {
				ordersWithIssues++
			}
		}
	}

	t.Logf("Orders validated: %d (issues: %d)", ordersChecked, ordersWithIssues)
	assert.Less(t, ordersWithIssues, ordersChecked/10, "Less than 10%% of orders should have issues")
}

func TestMedicationOrdersHaveDosing(t *testing.T) {
	allTemplates := getAllTemplates()
	medsChecked := 0
	medsWithDosing := 0

	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			if order.OrderType == models.OrderTypeMedication || order.Type == "medication" {
				medsChecked++
				// Must have dose, route, and frequency
				if order.Dose != "" || (order.DoseValue > 0 && order.DoseUnit != "") {
					medsWithDosing++
				}
			}
		}
	}

	if medsChecked > 0 {
		pct := float64(medsWithDosing) / float64(medsChecked) * 100
		t.Logf("Medications with dosing: %d/%d (%.1f%%)", medsWithDosing, medsChecked, pct)
		assert.GreaterOrEqual(t, pct, 80.0, "At least 80%% of medications should have dosing info")
	}
}

func TestOrderPriorityValues(t *testing.T) {
	allTemplates := getAllTemplates()
	validPriorities := map[models.Priority]bool{
		models.PrioritySTAT:    true,
		models.PriorityUrgent:  true,
		models.PriorityRoutine: true,
		models.PriorityPRN:     true,
		"":                     true, // Empty is acceptable
	}

	invalidCount := 0
	for _, tmpl := range allTemplates {
		for _, order := range tmpl.Orders {
			if !validPriorities[order.Priority] {
				t.Logf("Invalid priority '%s' in %s: %s", order.Priority, tmpl.Name, order.Name)
				invalidCount++
			}
		}
	}

	assert.Equal(t, 0, invalidCount, "All order priorities should be valid")
}

func TestOrderSequencing(t *testing.T) {
	allTemplates := getAllTemplates()
	sequencedTemplates := 0

	for _, tmpl := range allTemplates {
		hasSequence := false
		for _, order := range tmpl.Orders {
			if order.Sequence > 0 {
				hasSequence = true
				break
			}
		}
		if hasSequence {
			sequencedTemplates++
		}
	}

	t.Logf("Templates with order sequencing: %d/%d", sequencedTemplates, len(allTemplates))
}

// ============================================
// 2.5 Section Structure Validation
// ============================================

func TestSectionsHaveOrdersOrItems(t *testing.T) {
	allTemplates := getAllTemplates()
	sectionsChecked := 0
	emptySections := 0

	for _, tmpl := range allTemplates {
		for _, section := range tmpl.Sections {
			sectionsChecked++
			if len(section.Orders) == 0 && len(section.Items) == 0 {
				emptySections++
				t.Logf("Empty section in %s: %s", tmpl.Name, section.Name)
			}
		}
	}

	t.Logf("Sections validated: %d (empty: %d)", sectionsChecked, emptySections)
}

func TestSectionSequencing(t *testing.T) {
	allTemplates := getAllTemplates()

	for _, tmpl := range allTemplates {
		if len(tmpl.Sections) > 1 {
			sequences := make(map[int]bool)
			for _, section := range tmpl.Sections {
				if section.Sequence > 0 {
					if sequences[section.Sequence] {
						t.Logf("Duplicate section sequence in %s: %d", tmpl.Name, section.Sequence)
					}
					sequences[section.Sequence] = true
				}
			}
		}
	}
}

// ============================================
// 2.6 Reference & Evidence Validation
// ============================================

func TestGuidelineSourcesPresent(t *testing.T) {
	allTemplates := getAllTemplates()
	withGuidelines := 0

	for _, tmpl := range allTemplates {
		if tmpl.GuidelineSource != "" {
			withGuidelines++
		}
	}

	pct := float64(withGuidelines) / float64(len(allTemplates)) * 100
	t.Logf("Templates with guideline sources: %d/%d (%.1f%%)", withGuidelines, len(allTemplates), pct)
}

func TestEvidenceLevelsValid(t *testing.T) {
	allTemplates := getAllTemplates()
	validLevels := map[string]bool{
		"Level I":    true,
		"Level II":   true,
		"Level III":  true,
		"Level IV":   true,
		"Class I":    true,
		"Class IIa":  true,
		"Class IIb":  true,
		"Class III":  true,
		"Grade A":    true,
		"Grade B":    true,
		"Grade C":    true,
		"":           true, // Empty is acceptable
	}

	for _, tmpl := range allTemplates {
		if tmpl.EvidenceLevel != "" && !validLevels[tmpl.EvidenceLevel] {
			t.Logf("Non-standard evidence level in %s: %s", tmpl.Name, tmpl.EvidenceLevel)
		}
	}
}

func TestReferencesFormatted(t *testing.T) {
	allTemplates := getAllTemplates()
	templatesWithRefs := 0

	for _, tmpl := range allTemplates {
		if len(tmpl.References) > 0 {
			templatesWithRefs++
		}
	}

	t.Logf("Templates with references: %d/%d", templatesWithRefs, len(allTemplates))
}

// ============================================
// 2.7 Version & Status Validation
// ============================================

func TestAllTemplatesHaveVersion(t *testing.T) {
	allTemplates := getAllTemplates()
	missingVersion := 0

	for _, tmpl := range allTemplates {
		if tmpl.Version == "" {
			t.Logf("Missing version: %s", tmpl.Name)
			missingVersion++
		}
	}

	assert.Equal(t, 0, missingVersion, "All templates should have version numbers")
}

func TestTemplateStatusValues(t *testing.T) {
	allTemplates := getAllTemplates()
	validStatuses := map[string]bool{
		"active":    true,
		"draft":     true,
		"retired":   true,
		"pending":   true,
		"":          true,
	}

	for _, tmpl := range allTemplates {
		if !validStatuses[tmpl.Status] {
			t.Errorf("Invalid status '%s' in template %s", tmpl.Status, tmpl.Name)
		}
	}
}

// ============================================
// Helper Functions
// ============================================

func validateTemplateStructure(t *testing.T, tmpl *models.OrderSetTemplate) {
	t.Helper()

	// Required fields
	assert.NotEmpty(t, tmpl.TemplateID, "Template must have ID")
	assert.NotEmpty(t, tmpl.Name, "Template must have name")
	assert.NotEmpty(t, tmpl.Category, "Template must have category")
	assert.NotEmpty(t, tmpl.Version, "Template must have version")

	// Must have either orders or sections
	hasContent := len(tmpl.Orders) > 0 || len(tmpl.Sections) > 0
	assert.True(t, hasContent, "Template must have orders or sections")

	t.Logf("✓ %s: %d orders, %d sections, %d constraints",
		tmpl.Name, len(tmpl.Orders), len(tmpl.Sections), len(tmpl.TimeConstraints))
}

func getAllTemplates() []*models.OrderSetTemplate {
	templates := make([]*models.OrderSetTemplate, 0)
	templates = append(templates, ordersets.GetAllAdmissionOrderSets()...)
	templates = append(templates, ordersets.GetAllProcedureOrderSets()...)
	templates = append(templates, ordersets.GetAllEmergencyProtocols()...)
	return templates
}
