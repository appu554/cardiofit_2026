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

// DrugDiseaseContraindication represents a drug-disease contraindication rule
// Maps to ddi_drug_disease_rules table
type DrugDiseaseContraindication struct {
	ID                   uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion       string               `gorm:"column:dataset_version;not null;index" json:"dataset_version"`
	DrugCode             string               `gorm:"column:drug_code;not null;index" json:"drug_code"`
	DrugName             string               `gorm:"column:drug_name;not null" json:"drug_name"`
	DiseaseCode          string               `gorm:"column:disease_code;not null;index" json:"disease_code"`
	DiseaseName          string               `gorm:"column:disease_name;not null" json:"disease_name"`
	CodeSystem           string               `gorm:"column:code_system;not null" json:"code_system"`
	ContraindicationType string               `gorm:"column:contraindication_type;not null" json:"contraindication_type"`
	Severity             models.DDISeverity   `gorm:"column:severity;not null" json:"severity"`
	ClinicalRationale    string                    `gorm:"column:clinical_rationale;not null" json:"clinical_rationale"`
	ManagementStrategy   string                    `gorm:"column:management_strategy;not null" json:"management_strategy"`
	AlternativeDrugs     models.JSONBStringArray   `gorm:"column:alternative_drugs;type:jsonb" json:"alternative_drugs,omitempty"`
	MonitoringParameters models.JSONBStringArray   `gorm:"column:monitoring_parameters;type:jsonb" json:"monitoring_parameters,omitempty"`
	Exceptions           *models.JSONB             `gorm:"column:exceptions;type:jsonb" json:"exceptions,omitempty"`
	Evidence             models.EvidenceLevel `gorm:"column:evidence;not null" json:"evidence"`
	Confidence           *decimal.Decimal     `gorm:"column:confidence;type:decimal(3,2);default:0.80" json:"confidence,omitempty"`
	SourceVendorIDs      models.StringArray   `gorm:"column:source_vendor_ids;type:text[]" json:"source_vendor_ids,omitempty"`
	Active               bool                 `gorm:"column:active;default:true" json:"active"`
	CreatedAt            time.Time            `gorm:"column:created_at" json:"created_at"`
	UpdatedAt            time.Time            `gorm:"column:updated_at" json:"updated_at"`
}

// TableName specifies the database table for GORM
func (DrugDiseaseContraindication) TableName() string {
	return "ddi_drug_disease_rules"
}

// DrugDiseaseResult represents the result of a drug-disease contraindication check
type DrugDiseaseResult struct {
	DrugCode               string              `json:"drug_code"`
	DrugName               string              `json:"drug_name"`
	DiseaseCode            string              `json:"disease_code"`
	DiseaseName            string              `json:"disease_name"`
	ContraindicationType   string              `json:"contraindication_type"`
	Severity               models.DDISeverity  `json:"severity"`
	ClinicalEffects        string              `json:"clinical_effects"`
	ManagementStrategy     string              `json:"management_strategy"`
	MonitoringRequired     bool                `json:"monitoring_required"`
	MonitoringParameters   []string            `json:"monitoring_parameters,omitempty"`
	AlternativeDrugs       []string            `json:"alternative_drugs,omitempty"`
	Evidence               models.EvidenceLevel `json:"evidence"`
	Confidence             float64             `json:"confidence"`
	RequiresPharmacistReview bool              `json:"requires_pharmacist_review"`
}

// DrugDiseaseCheckRequest represents a request to check drug-disease contraindications
type DrugDiseaseCheckRequest struct {
	DrugCodes      []string `json:"drug_codes" binding:"required,min=1"`
	DiseaseCodes   []string `json:"disease_codes" binding:"required,min=1"`   // ICD-10 or SNOMED codes
	CodeSystem     string   `json:"code_system,omitempty"`                    // "ICD-10" or "SNOMED-CT", auto-detect if empty
	PatientContext *models.PatientContext `json:"patient_context,omitempty"`
	IncludeCautions bool    `json:"include_cautions"`                         // Include "caution" level contraindications
}

// DrugDiseaseEngine evaluates drug-disease contraindications
type DrugDiseaseEngine struct {
	db      *database.Database
	metrics *metrics.Collector

	// Cache for contraindication rules
	ruleCache map[string][]DrugDiseaseContraindication
	cacheTTL  time.Duration
	lastLoad  time.Time
}

// NewDrugDiseaseEngine creates a new drug-disease contraindication engine
func NewDrugDiseaseEngine(db *database.Database, metrics *metrics.Collector) *DrugDiseaseEngine {
	return &DrugDiseaseEngine{
		db:        db,
		metrics:   metrics,
		ruleCache: make(map[string][]DrugDiseaseContraindication),
		cacheTTL:  30 * time.Minute,
	}
}

// EvaluateDrugDiseaseContraindications checks drugs against patient's disease conditions
func (dde *DrugDiseaseEngine) EvaluateDrugDiseaseContraindications(
	ctx context.Context,
	request DrugDiseaseCheckRequest,
	datasetVersion string,
) ([]DrugDiseaseResult, error) {
	if len(request.DrugCodes) == 0 || len(request.DiseaseCodes) == 0 {
		return []DrugDiseaseResult{}, nil
	}

	timer := time.Now()
	defer func() {
		dde.metrics.RecordDrugDiseaseCheck(time.Since(timer), len(request.DrugCodes)*len(request.DiseaseCodes))
	}()

	// Detect code system if not specified
	codeSystem := request.CodeSystem
	if codeSystem == "" {
		codeSystem = dde.detectCodeSystem(request.DiseaseCodes)
	}

	// Load contraindication rules
	rules, err := dde.loadContraindicationRules(ctx, request.DrugCodes, codeSystem, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load drug-disease rules: %w", err)
	}

	var results []DrugDiseaseResult

	// Evaluate each drug against each disease
	for _, rule := range rules {
		for _, diseaseCode := range request.DiseaseCodes {
			if dde.matchesDisease(rule, diseaseCode, codeSystem) {
				// Skip cautions if not requested
				if rule.ContraindicationType == "caution" && !request.IncludeCautions {
					continue
				}

				// Check for exceptions based on patient context
				if dde.hasException(rule, request.PatientContext) {
					continue
				}

				result := dde.buildDrugDiseaseResult(rule, request.PatientContext)
				results = append(results, result)

				// Record metric
				dde.metrics.RecordDrugDiseaseInteraction(rule.DrugCode, rule.DiseaseCode, string(rule.Severity))
			}
		}
	}

	// Sort by severity (contraindicated > major > moderate > minor)
	dde.sortBySeverity(results)

	return results, nil
}

// CheckSingleDrugDisease checks a single drug against a single disease condition
func (dde *DrugDiseaseEngine) CheckSingleDrugDisease(
	ctx context.Context,
	drugCode string,
	diseaseCode string,
	codeSystem string,
	datasetVersion string,
) (*DrugDiseaseResult, error) {
	timer := time.Now()
	defer func() {
		dde.metrics.RecordDrugDiseaseCheck(time.Since(timer), 1)
	}()

	if codeSystem == "" {
		codeSystem = dde.detectCodeSystem([]string{diseaseCode})
	}

	var rule DrugDiseaseContraindication
	err := dde.db.DB.WithContext(ctx).
		Where("drug_code = ? AND disease_code = ? AND code_system = ? AND dataset_version = ? AND active = true",
			strings.ToUpper(drugCode), strings.ToUpper(diseaseCode), codeSystem, datasetVersion).
		First(&rule).Error

	if err != nil {
		return nil, nil // No contraindication found
	}

	result := dde.buildDrugDiseaseResult(rule, nil)
	return &result, nil
}

// GetContraindicatedDiseases returns all diseases contraindicated for a specific drug
func (dde *DrugDiseaseEngine) GetContraindicatedDiseases(
	ctx context.Context,
	drugCode string,
	datasetVersion string,
) ([]DrugDiseaseContraindication, error) {
	var contraindications []DrugDiseaseContraindication

	err := dde.db.DB.WithContext(ctx).
		Where("drug_code = ? AND dataset_version = ? AND active = true",
			strings.ToUpper(drugCode), datasetVersion).
		Order("CASE severity WHEN 'contraindicated' THEN 1 WHEN 'major' THEN 2 WHEN 'moderate' THEN 3 WHEN 'minor' THEN 4 ELSE 5 END").
		Find(&contraindications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get contraindicated diseases: %w", err)
	}

	return contraindications, nil
}

// GetContraindicatedDrugs returns all drugs contraindicated for a specific disease
func (dde *DrugDiseaseEngine) GetContraindicatedDrugs(
	ctx context.Context,
	diseaseCode string,
	codeSystem string,
	datasetVersion string,
) ([]DrugDiseaseContraindication, error) {
	if codeSystem == "" {
		codeSystem = dde.detectCodeSystem([]string{diseaseCode})
	}

	var contraindications []DrugDiseaseContraindication

	err := dde.db.DB.WithContext(ctx).
		Where("disease_code = ? AND code_system = ? AND dataset_version = ? AND active = true",
			strings.ToUpper(diseaseCode), codeSystem, datasetVersion).
		Order("CASE severity WHEN 'contraindicated' THEN 1 WHEN 'major' THEN 2 WHEN 'moderate' THEN 3 WHEN 'minor' THEN 4 ELSE 5 END").
		Find(&contraindications).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get contraindicated drugs: %w", err)
	}

	return contraindications, nil
}

// GetSupportedDiseaseCodes returns all disease codes with contraindication rules
func (dde *DrugDiseaseEngine) GetSupportedDiseaseCodes(
	ctx context.Context,
	codeSystem string,
	datasetVersion string,
) ([]string, error) {
	var codes []string

	query := dde.db.DB.WithContext(ctx).
		Model(&DrugDiseaseContraindication{}).
		Where("dataset_version = ? AND active = true", datasetVersion).
		Distinct("disease_code")

	if codeSystem != "" {
		query = query.Where("code_system = ?", codeSystem)
	}

	err := query.Pluck("disease_code", &codes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get supported disease codes: %w", err)
	}

	return codes, nil
}

// loadContraindicationRules loads rules from database or cache
func (dde *DrugDiseaseEngine) loadContraindicationRules(
	ctx context.Context,
	drugCodes []string,
	codeSystem string,
	datasetVersion string,
) ([]DrugDiseaseContraindication, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s", strings.Join(drugCodes, ","), codeSystem, datasetVersion)

	// Check cache
	if time.Since(dde.lastLoad) < dde.cacheTTL {
		if cached, exists := dde.ruleCache[cacheKey]; exists {
			return cached, nil
		}
	}

	// Query database
	var rules []DrugDiseaseContraindication

	// Normalize drug codes to uppercase
	normalizedCodes := make([]string, len(drugCodes))
	for i, code := range drugCodes {
		normalizedCodes[i] = strings.ToUpper(code)
	}

	err := dde.db.DB.WithContext(ctx).
		Where("drug_code IN ? AND code_system = ? AND dataset_version = ? AND active = true",
			normalizedCodes, codeSystem, datasetVersion).
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	dde.ruleCache[cacheKey] = rules
	dde.lastLoad = time.Now()

	return rules, nil
}

// detectCodeSystem determines the code system from the format of disease codes
func (dde *DrugDiseaseEngine) detectCodeSystem(diseaseCodes []string) string {
	if len(diseaseCodes) == 0 {
		return "ICD10" // Match database format (no hyphen)
	}

	firstCode := diseaseCodes[0]

	// SNOMED codes are typically numeric and longer
	if len(firstCode) > 6 && dde.isNumeric(firstCode) {
		return "SNOMED" // Match database format (no -CT suffix)
	}

	// ICD-10 codes follow pattern like A00.0, E11.9, etc.
	if len(firstCode) >= 3 && firstCode[0] >= 'A' && firstCode[0] <= 'Z' {
		return "ICD10" // Match database format (no hyphen)
	}

	return "ICD10" // Default
}

// isNumeric checks if a string is all digits
func (dde *DrugDiseaseEngine) isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// matchesDisease checks if a rule matches a disease code
func (dde *DrugDiseaseEngine) matchesDisease(rule DrugDiseaseContraindication, diseaseCode string, codeSystem string) bool {
	normalizedCode := strings.ToUpper(diseaseCode)
	normalizedRuleCode := strings.ToUpper(rule.DiseaseCode)

	// Exact match
	if normalizedRuleCode == normalizedCode {
		return true
	}

	// ICD-10 hierarchical matching (E11 matches E11.9, E11.65, etc.)
	if codeSystem == "ICD10" || codeSystem == "ICD-10" { // Support both formats
		// Check if disease code starts with rule code (parent category)
		if strings.HasPrefix(normalizedCode, normalizedRuleCode) {
			return true
		}
		// Check if rule code starts with disease code (child matches parent)
		if strings.HasPrefix(normalizedRuleCode, normalizedCode) {
			return true
		}
	}

	return false
}

// hasException checks if patient context triggers an exception to the contraindication
func (dde *DrugDiseaseEngine) hasException(rule DrugDiseaseContraindication, patientContext *models.PatientContext) bool {
	if rule.Exceptions == nil || patientContext == nil {
		return false
	}

	// Parse exceptions JSON and check against patient context
	// Exceptions might include conditions like:
	// - age_over: 65 (contraindication only applies to elderly)
	// - renal_function_below: 30 (only applies when eGFR < 30)
	// This is a simplified implementation - expand based on clinical needs

	return false
}

// buildDrugDiseaseResult creates a result from a contraindication rule
func (dde *DrugDiseaseEngine) buildDrugDiseaseResult(rule DrugDiseaseContraindication, patientContext *models.PatientContext) DrugDiseaseResult {
	confidence := 0.80
	if rule.Confidence != nil {
		confidence, _ = rule.Confidence.Float64()
	}

	// Determine if monitoring is required based on severity and presence of monitoring parameters
	monitoringRequired := len(rule.MonitoringParameters) > 0 ||
		rule.Severity == models.SeverityMajor ||
		rule.Severity == models.SeverityContraindicated

	result := DrugDiseaseResult{
		DrugCode:               rule.DrugCode,
		DrugName:               rule.DrugName,
		DiseaseCode:            rule.DiseaseCode,
		DiseaseName:            rule.DiseaseName,
		ContraindicationType:   rule.ContraindicationType,
		Severity:               rule.Severity,
		ClinicalEffects:        rule.ClinicalRationale, // Map from ClinicalRationale column
		ManagementStrategy:     rule.ManagementStrategy,
		MonitoringRequired:     monitoringRequired,     // Computed from severity & monitoring params
		Evidence:               rule.Evidence,
		Confidence:             confidence,
		RequiresPharmacistReview: rule.Severity == models.SeverityContraindicated || rule.Severity == models.SeverityMajor,
	}

	// Parse monitoring parameters if available
	if len(rule.MonitoringParameters) > 0 {
		// Use actual monitoring parameters from database
		result.MonitoringParameters = rule.MonitoringParameters
	}

	// Suggest alternative drugs for severe contraindications
	if rule.Severity == models.SeverityContraindicated || rule.Severity == models.SeverityMajor {
		result.AlternativeDrugs = dde.suggestAlternatives(rule.DrugCode, rule.DiseaseCode)
	}

	return result
}

// suggestAlternatives provides alternative drug suggestions
func (dde *DrugDiseaseEngine) suggestAlternatives(drugCode string, diseaseCode string) []string {
	// This is a simplified implementation
	// In production, this would query a comprehensive drug alternatives database

	// Common drug-disease alternatives
	alternatives := map[string]map[string][]string{
		// NSAIDs contraindicated in CKD - suggest alternatives
		"IBUPROFEN": {
			"N18":   {"Acetaminophen", "Topical diclofenac"},     // CKD
			"K25":   {"Acetaminophen"},                            // Gastric ulcer
			"I50":   {"Acetaminophen"},                            // Heart failure
		},
		"NAPROXEN": {
			"N18":   {"Acetaminophen", "Topical diclofenac"},
			"K25":   {"Acetaminophen"},
			"I50":   {"Acetaminophen"},
		},
		// Metformin contraindicated in severe CKD
		"METFORMIN": {
			"N18.5": {"SGLT2 inhibitors (with caution)", "GLP-1 agonists", "DPP-4 inhibitors"},
		},
		// Beta blockers in asthma
		"PROPRANOLOL": {
			"J45":   {"Cardioselective beta-blockers (bisoprolol, metoprolol)"},
		},
	}

	normalizedDrug := strings.ToUpper(drugCode)
	if drugAlts, exists := alternatives[normalizedDrug]; exists {
		// Check for exact disease match first
		if alts, exists := drugAlts[strings.ToUpper(diseaseCode)]; exists {
			return alts
		}
		// Check for prefix match (ICD-10 hierarchy)
		for diseasePrefix, alts := range drugAlts {
			if strings.HasPrefix(strings.ToUpper(diseaseCode), diseasePrefix) {
				return alts
			}
		}
	}

	return nil
}

// sortBySeverity sorts results by clinical severity
func (dde *DrugDiseaseEngine) sortBySeverity(results []DrugDiseaseResult) {
	severityOrder := map[models.DDISeverity]int{
		models.SeverityContraindicated: 1,
		models.SeverityMajor:          2,
		models.SeverityModerate:       3,
		models.SeverityMinor:          4,
		models.SeverityUnknown:        5,
	}

	// Simple bubble sort for typical small result sets
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if severityOrder[results[j].Severity] > severityOrder[results[j+1].Severity] {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// ClearCache clears the contraindication rules cache
func (dde *DrugDiseaseEngine) ClearCache() {
	dde.ruleCache = make(map[string][]DrugDiseaseContraindication)
	dde.lastLoad = time.Time{}
}
