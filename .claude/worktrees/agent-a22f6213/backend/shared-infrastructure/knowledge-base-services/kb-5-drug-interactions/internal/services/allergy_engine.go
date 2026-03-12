package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AllergyRule represents an allergy cross-reactivity rule
type AllergyRule struct {
	ID                   uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion       string              `gorm:"not null;index" json:"dataset_version"`
	AllergenCode         string              `gorm:"size:100;not null;index" json:"allergen_code"`        // Drug code or allergen class
	AllergenName         string              `gorm:"size:200;not null" json:"allergen_name"`
	AllergenType         string              `gorm:"size:50;not null" json:"allergen_type"`              // drug, drug_class, ingredient, excipient
	CrossReactiveDrugCode string             `gorm:"size:100;not null;index" json:"cross_reactive_drug_code"`
	CrossReactiveDrugName string             `gorm:"size:200;not null" json:"cross_reactive_drug_name"`
	CrossReactivityType  string              `gorm:"size:50;not null" json:"cross_reactivity_type"`       // structural, class, ingredient
	CrossReactivityRate  *decimal.Decimal    `gorm:"type:decimal(5,2)" json:"cross_reactivity_rate,omitempty"` // Percentage (e.g., 10.00 = 10%)
	Severity             models.DDISeverity  `gorm:"not null" json:"severity"`
	ReactionType         string              `gorm:"size:100" json:"reaction_type,omitempty"`             // anaphylaxis, rash, angioedema, etc.
	ClinicalGuidance     string              `gorm:"type:text;not null" json:"clinical_guidance"`
	AlternativeDrugs     models.StringArray  `gorm:"type:text[]" json:"alternative_drugs,omitempty"`
	Evidence             models.EvidenceLevel `gorm:"not null" json:"evidence"`
	Confidence           *decimal.Decimal    `gorm:"type:decimal(3,2);default:0.80" json:"confidence,omitempty"`
	SourceReferences     models.StringArray  `gorm:"type:text[]" json:"source_references,omitempty"`
	Active               bool                `gorm:"default:true" json:"active"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// TableName specifies the database table for GORM
func (AllergyRule) TableName() string {
	return "ddi_allergy_rules"
}

// AllergyCheckResult represents the result of an allergy cross-reactivity check
type AllergyCheckResult struct {
	AllergenCode         string              `json:"allergen_code"`
	AllergenName         string              `json:"allergen_name"`
	DrugCode             string              `json:"drug_code"`
	DrugName             string              `json:"drug_name"`
	CrossReactivityType  string              `json:"cross_reactivity_type"`
	CrossReactivityRate  float64             `json:"cross_reactivity_rate,omitempty"` // Percentage
	Severity             models.DDISeverity  `json:"severity"`
	ReactionType         string              `json:"reaction_type,omitempty"`
	ClinicalGuidance     string              `json:"clinical_guidance"`
	AlternativeDrugs     []string            `json:"alternative_drugs,omitempty"`
	Evidence             models.EvidenceLevel `json:"evidence"`
	Confidence           float64             `json:"confidence"`
	RequiresPharmacistReview bool            `json:"requires_pharmacist_review"`
	AlertLevel           string              `json:"alert_level"` // critical, high, moderate, low
}

// AllergyCheckRequest represents a request to check drug allergies
type AllergyCheckRequest struct {
	DrugCodes        []string          `json:"drug_codes" binding:"required,min=1"`
	PatientAllergies []PatientAllergy  `json:"patient_allergies" binding:"required,min=1"`
	IncludePossible  bool              `json:"include_possible"` // Include possible (low probability) cross-reactions
}

// PatientAllergy represents a patient's documented allergy
type PatientAllergy struct {
	AllergenCode  string `json:"allergen_code"`             // Drug code or allergen class code
	AllergenName  string `json:"allergen_name,omitempty"`
	AllergenType  string `json:"allergen_type,omitempty"`   // drug, drug_class, ingredient
	ReactionType  string `json:"reaction_type,omitempty"`   // anaphylaxis, rash, urticaria, etc.
	Severity      string `json:"severity,omitempty"`        // mild, moderate, severe, life-threatening
	OnsetDate     string `json:"onset_date,omitempty"`
	Verified      bool   `json:"verified"`                  // Clinically verified allergy
}

// CrossReactivityInfo provides detailed cross-reactivity information
type CrossReactivityInfo struct {
	AllergenClass      string   `json:"allergen_class"`
	RelatedDrugs       []string `json:"related_drugs"`
	CrossReactivityRate float64 `json:"cross_reactivity_rate"`
	ClinicalNote       string   `json:"clinical_note"`
}

// AllergyEngine evaluates drug allergy cross-reactivity
type AllergyEngine struct {
	db      *database.Database
	metrics *metrics.Collector

	// Cache for allergy rules
	ruleCache map[string][]AllergyRule
	cacheTTL  time.Duration
	lastLoad  time.Time
}

// NewAllergyEngine creates a new allergy cross-reactivity engine
func NewAllergyEngine(db *database.Database, metrics *metrics.Collector) *AllergyEngine {
	return &AllergyEngine{
		db:        db,
		metrics:   metrics,
		ruleCache: make(map[string][]AllergyRule),
		cacheTTL:  30 * time.Minute,
	}
}

// EvaluateAllergyRisk checks drugs against patient's documented allergies
func (ae *AllergyEngine) EvaluateAllergyRisk(
	ctx context.Context,
	request AllergyCheckRequest,
	datasetVersion string,
) ([]AllergyCheckResult, error) {
	if len(request.DrugCodes) == 0 || len(request.PatientAllergies) == 0 {
		return []AllergyCheckResult{}, nil
	}

	timer := time.Now()
	defer func() {
		ae.metrics.RecordAllergyCheck(time.Since(timer), len(request.DrugCodes)*len(request.PatientAllergies))
	}()

	var results []AllergyCheckResult

	// Load allergy rules for all patient allergies
	allergenCodes := make([]string, len(request.PatientAllergies))
	for i, allergy := range request.PatientAllergies {
		allergenCodes[i] = allergy.AllergenCode
	}

	rules, err := ae.loadAllergyRules(ctx, allergenCodes, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load allergy rules: %w", err)
	}

	// Check each drug against each allergy rule
	for _, drugCode := range request.DrugCodes {
		normalizedDrug := strings.ToUpper(drugCode)

		for _, rule := range rules {
			if ae.matchesDrug(rule, normalizedDrug) {
				// Filter low probability cross-reactions if not requested
				if !request.IncludePossible && rule.CrossReactivityRate != nil {
					rate, _ := rule.CrossReactivityRate.Float64()
					if rate < 5.0 {
						continue
					}
				}

				result := ae.buildAllergyResult(rule, ae.findPatientAllergy(request.PatientAllergies, rule.AllergenCode))
				results = append(results, result)

				// Record metric
				ae.metrics.RecordAllergyInteraction(rule.AllergenCode, normalizedDrug, string(rule.Severity))
			}
		}

		// Also check direct allergy (drug is the allergen itself)
		for _, allergy := range request.PatientAllergies {
			if strings.EqualFold(allergy.AllergenCode, normalizedDrug) {
				result := AllergyCheckResult{
					AllergenCode:         allergy.AllergenCode,
					AllergenName:         allergy.AllergenName,
					DrugCode:             normalizedDrug,
					DrugName:             allergy.AllergenName,
					CrossReactivityType:  "direct",
					CrossReactivityRate:  100.0,
					Severity:             models.SeverityContraindicated,
					ReactionType:         allergy.ReactionType,
					ClinicalGuidance:     "Patient has documented allergy to this exact drug. Do NOT administer.",
					Evidence:             models.EvidenceLevelA,
					Confidence:           1.0,
					RequiresPharmacistReview: true,
					AlertLevel:           "critical",
				}
				results = append(results, result)
			}
		}
	}

	// Sort by alert level and severity
	ae.sortByAlertLevel(results)

	return results, nil
}

// GetCrossReactivity returns cross-reactivity information for an allergen
func (ae *AllergyEngine) GetCrossReactivity(
	ctx context.Context,
	allergenCode string,
	datasetVersion string,
) ([]CrossReactivityInfo, error) {
	var rules []AllergyRule

	err := ae.db.DB.WithContext(ctx).
		Where("allergen_code = ? AND dataset_version = ? AND active = true",
			strings.ToUpper(allergenCode), datasetVersion).
		Order("cross_reactivity_rate DESC NULLS LAST").
		Find(&rules).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get cross-reactivity info: %w", err)
	}

	// Group by allergen class
	classMap := make(map[string]*CrossReactivityInfo)

	for _, rule := range rules {
		classKey := rule.AllergenType
		if _, exists := classMap[classKey]; !exists {
			rate := 0.0
			if rule.CrossReactivityRate != nil {
				rate, _ = rule.CrossReactivityRate.Float64()
			}
			classMap[classKey] = &CrossReactivityInfo{
				AllergenClass:       classKey,
				RelatedDrugs:        []string{},
				CrossReactivityRate: rate,
				ClinicalNote:        rule.ClinicalGuidance,
			}
		}
		classMap[classKey].RelatedDrugs = append(classMap[classKey].RelatedDrugs, rule.CrossReactiveDrugName)
	}

	var results []CrossReactivityInfo
	for _, info := range classMap {
		results = append(results, *info)
	}

	return results, nil
}

// GetAllergyAlternatives suggests alternative drugs for a patient with allergies
func (ae *AllergyEngine) GetAllergyAlternatives(
	ctx context.Context,
	drugCode string,
	patientAllergies []PatientAllergy,
	datasetVersion string,
) ([]string, error) {
	// First check if drug is contraindicated for the patient
	checkRequest := AllergyCheckRequest{
		DrugCodes:        []string{drugCode},
		PatientAllergies: patientAllergies,
		IncludePossible:  false,
	}

	conflicts, err := ae.EvaluateAllergyRisk(ctx, checkRequest, datasetVersion)
	if err != nil {
		return nil, err
	}

	// Collect alternatives from conflict rules
	alternativesSet := make(map[string]bool)
	for _, conflict := range conflicts {
		for _, alt := range conflict.AlternativeDrugs {
			// Verify alternative is not also allergenic for this patient
			altCheckRequest := AllergyCheckRequest{
				DrugCodes:        []string{alt},
				PatientAllergies: patientAllergies,
				IncludePossible:  true,
			}
			altConflicts, _ := ae.EvaluateAllergyRisk(ctx, altCheckRequest, datasetVersion)
			if len(altConflicts) == 0 {
				alternativesSet[alt] = true
			}
		}
	}

	alternatives := make([]string, 0, len(alternativesSet))
	for alt := range alternativesSet {
		alternatives = append(alternatives, alt)
	}

	return alternatives, nil
}

// GetCommonCrossReactivities returns well-known cross-reactivity patterns
func (ae *AllergyEngine) GetCommonCrossReactivities(
	ctx context.Context,
	datasetVersion string,
) (map[string]CrossReactivityInfo, error) {
	// Return commonly known cross-reactivity patterns
	// This serves as a quick reference for clinical staff

	commonPatterns := map[string]CrossReactivityInfo{
		"penicillin-cephalosporin": {
			AllergenClass:       "Beta-lactam antibiotics",
			RelatedDrugs:        []string{"Cephalexin", "Cefazolin", "Ceftriaxone", "Cefuroxime"},
			CrossReactivityRate: 2.0, // ~2% cross-reactivity rate
			ClinicalNote:        "Cross-reactivity between penicillins and cephalosporins is lower than historically believed (~2%). First-generation cephalosporins have slightly higher cross-reactivity with penicillins.",
		},
		"penicillin-carbapenem": {
			AllergenClass:       "Beta-lactam antibiotics",
			RelatedDrugs:        []string{"Meropenem", "Imipenem", "Ertapenem"},
			CrossReactivityRate: 1.0,
			ClinicalNote:        "Cross-reactivity between penicillins and carbapenems is approximately 1%. Carbapenems may be used with caution in penicillin-allergic patients.",
		},
		"sulfonamide-antibiotics": {
			AllergenClass:       "Sulfonamide antibiotics",
			RelatedDrugs:        []string{"Sulfamethoxazole", "Sulfasalazine", "Silver sulfadiazine"},
			CrossReactivityRate: 0.0, // Not applicable - same class
			ClinicalNote:        "Sulfonamide antibiotic allergy does NOT generally cross-react with non-antibiotic sulfonamides (thiazides, furosemide, celecoxib). Different chemical structures.",
		},
		"nsaid-aspirin": {
			AllergenClass:       "NSAIDs and Aspirin",
			RelatedDrugs:        []string{"Ibuprofen", "Naproxen", "Diclofenac", "Ketorolac"},
			CrossReactivityRate: 20.0, // Variable, up to 20%
			ClinicalNote:        "Cross-reactivity between aspirin and other NSAIDs occurs in aspirin-exacerbated respiratory disease (AERD). COX-2 selective NSAIDs (celecoxib) may be safer alternatives.",
		},
		"ace-inhibitor-angioedema": {
			AllergenClass:       "ACE Inhibitors",
			RelatedDrugs:        []string{"Lisinopril", "Enalapril", "Ramipril", "Captopril"},
			CrossReactivityRate: 100.0, // Class effect
			ClinicalNote:        "ACE inhibitor-induced angioedema is a class effect. All ACE inhibitors should be avoided. ARBs have very low cross-reactivity (~3%) but use with caution.",
		},
		"statin-myopathy": {
			AllergenClass:       "Statins (HMG-CoA reductase inhibitors)",
			RelatedDrugs:        []string{"Atorvastatin", "Simvastatin", "Rosuvastatin", "Pravastatin"},
			CrossReactivityRate: 30.0, // Variable
			ClinicalNote:        "Statin-induced myopathy may recur with different statins. Consider lower doses, different statins, or alternative lipid-lowering therapy (ezetimibe, PCSK9 inhibitors).",
		},
		"local-anesthetic-ester": {
			AllergenClass:       "Ester-type local anesthetics",
			RelatedDrugs:        []string{"Procaine", "Benzocaine", "Tetracaine", "Chloroprocaine"},
			CrossReactivityRate: 80.0, // High within class
			ClinicalNote:        "Cross-reactivity is HIGH within ester-type local anesthetics. However, amide-type local anesthetics (lidocaine, bupivacaine) have NO cross-reactivity with esters.",
		},
	}

	return commonPatterns, nil
}

// loadAllergyRules loads rules from database or cache
func (ae *AllergyEngine) loadAllergyRules(
	ctx context.Context,
	allergenCodes []string,
	datasetVersion string,
) ([]AllergyRule, error) {
	cacheKey := fmt.Sprintf("%s:%s", strings.Join(allergenCodes, ","), datasetVersion)

	// Check cache
	if time.Since(ae.lastLoad) < ae.cacheTTL {
		if cached, exists := ae.ruleCache[cacheKey]; exists {
			return cached, nil
		}
	}

	// Normalize allergen codes
	normalizedCodes := make([]string, len(allergenCodes))
	for i, code := range allergenCodes {
		normalizedCodes[i] = strings.ToUpper(code)
	}

	var rules []AllergyRule
	err := ae.db.DB.WithContext(ctx).
		Where("allergen_code IN ? AND dataset_version = ? AND active = true",
			normalizedCodes, datasetVersion).
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	ae.ruleCache[cacheKey] = rules
	ae.lastLoad = time.Now()

	return rules, nil
}

// matchesDrug checks if a cross-reactive drug matches the given drug code
func (ae *AllergyEngine) matchesDrug(rule AllergyRule, drugCode string) bool {
	normalizedRuleDrug := strings.ToUpper(rule.CrossReactiveDrugCode)
	return normalizedRuleDrug == drugCode
}

// findPatientAllergy finds the patient allergy matching the allergen code
func (ae *AllergyEngine) findPatientAllergy(allergies []PatientAllergy, allergenCode string) *PatientAllergy {
	for _, allergy := range allergies {
		if strings.EqualFold(allergy.AllergenCode, allergenCode) {
			return &allergy
		}
	}
	return nil
}

// buildAllergyResult creates a result from an allergy rule
func (ae *AllergyEngine) buildAllergyResult(rule AllergyRule, patientAllergy *PatientAllergy) AllergyCheckResult {
	confidence := 0.80
	if rule.Confidence != nil {
		confidence, _ = rule.Confidence.Float64()
	}

	crossRate := 0.0
	if rule.CrossReactivityRate != nil {
		crossRate, _ = rule.CrossReactivityRate.Float64()
	}

	result := AllergyCheckResult{
		AllergenCode:         rule.AllergenCode,
		AllergenName:         rule.AllergenName,
		DrugCode:             rule.CrossReactiveDrugCode,
		DrugName:             rule.CrossReactiveDrugName,
		CrossReactivityType:  rule.CrossReactivityType,
		CrossReactivityRate:  crossRate,
		Severity:             rule.Severity,
		ReactionType:         rule.ReactionType,
		ClinicalGuidance:     rule.ClinicalGuidance,
		AlternativeDrugs:     rule.AlternativeDrugs,
		Evidence:             rule.Evidence,
		Confidence:           confidence,
		RequiresPharmacistReview: rule.Severity == models.SeverityContraindicated || rule.Severity == models.SeverityMajor,
	}

	// Determine alert level based on severity and cross-reactivity rate
	result.AlertLevel = ae.determineAlertLevel(rule, patientAllergy)

	return result
}

// determineAlertLevel determines the alert level based on multiple factors
func (ae *AllergyEngine) determineAlertLevel(rule AllergyRule, patientAllergy *PatientAllergy) string {
	// Critical: Contraindicated or patient had severe reaction (anaphylaxis)
	if rule.Severity == models.SeverityContraindicated {
		return "critical"
	}
	if patientAllergy != nil && (patientAllergy.Severity == "life-threatening" || patientAllergy.ReactionType == "anaphylaxis") {
		return "critical"
	}

	// High: Major severity or patient had severe reaction
	if rule.Severity == models.SeverityMajor {
		return "high"
	}
	if patientAllergy != nil && patientAllergy.Severity == "severe" {
		return "high"
	}

	// Moderate: Moderate severity
	if rule.Severity == models.SeverityModerate {
		return "moderate"
	}

	// Low: Minor severity or low cross-reactivity rate
	return "low"
}

// sortByAlertLevel sorts results by clinical urgency
func (ae *AllergyEngine) sortByAlertLevel(results []AllergyCheckResult) {
	alertOrder := map[string]int{
		"critical": 1,
		"high":     2,
		"moderate": 3,
		"low":      4,
	}

	// Simple bubble sort for typical small result sets
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if alertOrder[results[j].AlertLevel] > alertOrder[results[j+1].AlertLevel] {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// ClearCache clears the allergy rules cache
func (ae *AllergyEngine) ClearCache() {
	ae.ruleCache = make(map[string][]AllergyRule)
	ae.lastLoad = time.Time{}
}
