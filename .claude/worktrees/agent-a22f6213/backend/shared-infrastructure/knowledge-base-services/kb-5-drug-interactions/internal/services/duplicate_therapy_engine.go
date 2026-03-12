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
)

// DrugTherapeuticMapping represents a drug's therapeutic classification (GORM model)
type DrugTherapeuticMapping struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion   string    `gorm:"not null;index" json:"dataset_version"`
	DrugCode         string    `gorm:"size:100;not null;index" json:"drug_code"`
	DrugName         string    `gorm:"size:200;not null" json:"drug_name"`
	ATCCode          string    `gorm:"size:20;not null;index" json:"atc_code"`         // WHO ATC classification
	ATCLevel         int       `gorm:"not null" json:"atc_level"`                      // 1-5 (1=anatomical, 5=chemical substance)
	TherapeuticClass string    `gorm:"size:200;not null" json:"therapeutic_class"`     // Human-readable class name
	PharmacologicClass string  `gorm:"size:200" json:"pharmacologic_class,omitempty"` // Pharmacologic mechanism class
	Active           bool      `gorm:"default:true" json:"active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TableName specifies the database table for GORM
func (DrugTherapeuticMapping) TableName() string {
	return "ddi_therapeutic_classes"
}

// DuplicateTherapyRule defines rules for duplicate therapy detection
type DuplicateTherapyRule struct {
	ID                    uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DatasetVersion        string              `gorm:"not null;index" json:"dataset_version"`
	ATCCodePattern        string              `gorm:"size:20;not null;index" json:"atc_code_pattern"` // e.g., "N05A" for antipsychotics
	TherapeuticClassName  string              `gorm:"size:200;not null" json:"therapeutic_class_name"`
	AllowMultiple         bool                `gorm:"default:false" json:"allow_multiple"`             // Some classes allow multiple (e.g., combination antihypertensives)
	MaxAllowedAgents      int                 `gorm:"default:1" json:"max_allowed_agents"`
	ExceptionCombinations *models.JSONB       `gorm:"type:jsonb" json:"exception_combinations,omitempty"` // Allowed combinations
	Severity              models.DDISeverity  `gorm:"not null" json:"severity"`
	ClinicalRationale     string              `gorm:"type:text;not null" json:"clinical_rationale"`
	ManagementStrategy    string              `gorm:"type:text;not null" json:"management_strategy"`
	Evidence              models.EvidenceLevel `gorm:"not null" json:"evidence"`
	Active                bool                `gorm:"default:true" json:"active"`
	CreatedAt             time.Time           `json:"created_at"`
	UpdatedAt             time.Time           `json:"updated_at"`
}

// TableName specifies the database table for GORM
func (DuplicateTherapyRule) TableName() string {
	return "ddi_duplicate_therapy_rules"
}

// DuplicateTherapyResult represents the result of a duplicate therapy check
type DuplicateTherapyResult struct {
	DuplicateType        string              `json:"duplicate_type"`           // exact, therapeutic_class, pharmacologic_class
	TherapeuticClass     string              `json:"therapeutic_class"`
	ATCCode              string              `json:"atc_code"`
	DuplicateDrugs       []DuplicateDrugInfo `json:"duplicate_drugs"`
	Severity             models.DDISeverity  `json:"severity"`
	ClinicalRationale    string              `json:"clinical_rationale"`
	ManagementStrategy   string              `json:"management_strategy"`
	AllowedCombination   bool                `json:"allowed_combination"`      // True if this is a clinically acceptable combination
	Evidence             models.EvidenceLevel `json:"evidence"`
	RequiresPharmacistReview bool            `json:"requires_pharmacist_review"`
}

// DuplicateDrugInfo represents information about a drug involved in duplication
type DuplicateDrugInfo struct {
	DrugCode         string `json:"drug_code"`
	DrugName         string `json:"drug_name"`
	ATCCode          string `json:"atc_code"`
	TherapeuticClass string `json:"therapeutic_class"`
}

// DuplicateTherapyCheckRequest represents a request to check for duplicate therapy
type DuplicateTherapyCheckRequest struct {
	DrugCodes       []string               `json:"drug_codes" binding:"required,min=2"`
	IncludeAllowed  bool                   `json:"include_allowed"`  // Include clinically allowed duplicates
	CheckLevel      string                 `json:"check_level"`      // "strict" (ATC-5), "moderate" (ATC-4), "broad" (ATC-3)
	PatientContext  *models.PatientContext `json:"patient_context,omitempty"`
}

// DuplicateTherapyEngine detects duplicate therapy situations
type DuplicateTherapyEngine struct {
	db      *database.Database
	metrics *metrics.Collector

	// Cache for therapeutic class mappings
	classCache map[string][]DrugTherapeuticMapping
	ruleCache  map[string][]DuplicateTherapyRule
	cacheTTL   time.Duration
	lastLoad   time.Time
}

// NewDuplicateTherapyEngine creates a new duplicate therapy detection engine
func NewDuplicateTherapyEngine(db *database.Database, metrics *metrics.Collector) *DuplicateTherapyEngine {
	return &DuplicateTherapyEngine{
		db:         db,
		metrics:    metrics,
		classCache: make(map[string][]DrugTherapeuticMapping),
		ruleCache:  make(map[string][]DuplicateTherapyRule),
		cacheTTL:   30 * time.Minute,
	}
}

// CheckDuplicateTherapy evaluates a drug list for duplicate therapy
func (dte *DuplicateTherapyEngine) CheckDuplicateTherapy(
	ctx context.Context,
	request DuplicateTherapyCheckRequest,
	datasetVersion string,
) ([]DuplicateTherapyResult, error) {
	if len(request.DrugCodes) < 2 {
		return []DuplicateTherapyResult{}, nil
	}

	timer := time.Now()
	defer func() {
		dte.metrics.RecordDuplicateTherapyCheck(time.Since(timer), len(request.DrugCodes))
	}()

	// Determine check level (default to moderate)
	checkLevel := request.CheckLevel
	if checkLevel == "" {
		checkLevel = "moderate"
	}
	atcLevel := dte.getATCLevelForCheckLevel(checkLevel)

	// Load therapeutic classes for all drugs
	drugClasses, err := dte.loadTherapeuticClasses(ctx, request.DrugCodes, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load therapeutic classes: %w", err)
	}

	// Group drugs by therapeutic class at specified ATC level
	classGroups := dte.groupByTherapeuticClass(drugClasses, atcLevel)

	// Load duplicate therapy rules
	rules, err := dte.loadDuplicateTherapyRules(ctx, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load duplicate therapy rules: %w", err)
	}

	var results []DuplicateTherapyResult

	// Check each class group for duplicates
	for atcCode, drugs := range classGroups {
		if len(drugs) < 2 {
			continue // No duplicate in this class
		}

		// Find applicable rule
		rule := dte.findApplicableRule(rules, atcCode)

		// Check if this is an allowed combination
		isAllowed := dte.isAllowedCombination(rule, drugs)

		// Skip allowed combinations if not requested
		if isAllowed && !request.IncludeAllowed {
			continue
		}

		result := dte.buildDuplicateResult(atcCode, drugs, rule, isAllowed)
		results = append(results, result)

		// Record metric
		dte.metrics.RecordDuplicateTherapy(atcCode, len(drugs))
	}

	// Also check for exact duplicates (same drug code)
	exactDuplicates := dte.findExactDuplicates(request.DrugCodes)
	for drugCode, count := range exactDuplicates {
		if count > 1 {
			result := DuplicateTherapyResult{
				DuplicateType:      "exact",
				TherapeuticClass:   "Same medication",
				DuplicateDrugs: []DuplicateDrugInfo{{
					DrugCode: drugCode,
					DrugName: drugCode,
				}},
				Severity:           models.SeverityMajor,
				ClinicalRationale:  "The same medication appears multiple times in the drug list. This may indicate a prescribing error.",
				ManagementStrategy: "Review medication list for duplicate entries. Reconcile orders from different prescribers or settings.",
				Evidence:           models.EvidenceLevelA,
				RequiresPharmacistReview: true,
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// GetDrugTherapeuticClasses returns all therapeutic classifications for a drug
func (dte *DuplicateTherapyEngine) GetDrugTherapeuticClasses(
	ctx context.Context,
	drugCode string,
	datasetVersion string,
) ([]DrugTherapeuticMapping, error) {
	var classes []DrugTherapeuticMapping

	err := dte.db.DB.WithContext(ctx).
		Where("drug_code = ? AND dataset_version = ? AND active = true",
			strings.ToUpper(drugCode), datasetVersion).
		Order("atc_level").
		Find(&classes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get therapeutic classes: %w", err)
	}

	return classes, nil
}

// GetClassMembers returns all drugs in a therapeutic class
func (dte *DuplicateTherapyEngine) GetClassMembers(
	ctx context.Context,
	atcCode string,
	datasetVersion string,
) ([]DrugTherapeuticMapping, error) {
	var members []DrugTherapeuticMapping

	// Match ATC code at appropriate level
	err := dte.db.DB.WithContext(ctx).
		Where("atc_code LIKE ? AND dataset_version = ? AND active = true",
			strings.ToUpper(atcCode)+"%", datasetVersion).
		Distinct("drug_code, drug_name, atc_code, therapeutic_class").
		Find(&members).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get class members: %w", err)
	}

	return members, nil
}

// GetCommonDuplicateClasses returns commonly duplicated therapeutic classes
func (dte *DuplicateTherapyEngine) GetCommonDuplicateClasses(
	ctx context.Context,
	datasetVersion string,
) ([]DuplicateTherapyRule, error) {
	var rules []DuplicateTherapyRule

	err := dte.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = true", datasetVersion).
		Order("CASE severity WHEN 'contraindicated' THEN 1 WHEN 'major' THEN 2 WHEN 'moderate' THEN 3 WHEN 'minor' THEN 4 ELSE 5 END").
		Find(&rules).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get duplicate classes: %w", err)
	}

	return rules, nil
}

// loadTherapeuticClasses loads class mappings from database or cache
func (dte *DuplicateTherapyEngine) loadTherapeuticClasses(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) ([]DrugTherapeuticMapping, error) {
	// Normalize drug codes
	normalizedCodes := make([]string, len(drugCodes))
	for i, code := range drugCodes {
		normalizedCodes[i] = strings.ToUpper(code)
	}

	cacheKey := fmt.Sprintf("%s:%s", strings.Join(normalizedCodes, ","), datasetVersion)

	// Check cache
	if time.Since(dte.lastLoad) < dte.cacheTTL {
		if cached, exists := dte.classCache[cacheKey]; exists {
			return cached, nil
		}
	}

	var classes []DrugTherapeuticMapping
	err := dte.db.DB.WithContext(ctx).
		Where("drug_code IN ? AND dataset_version = ? AND active = true",
			normalizedCodes, datasetVersion).
		Find(&classes).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	dte.classCache[cacheKey] = classes
	dte.lastLoad = time.Now()

	return classes, nil
}

// loadDuplicateTherapyRules loads rules from database or cache
func (dte *DuplicateTherapyEngine) loadDuplicateTherapyRules(
	ctx context.Context,
	datasetVersion string,
) ([]DuplicateTherapyRule, error) {
	cacheKey := datasetVersion

	// Check cache
	if time.Since(dte.lastLoad) < dte.cacheTTL {
		if cached, exists := dte.ruleCache[cacheKey]; exists {
			return cached, nil
		}
	}

	var rules []DuplicateTherapyRule
	err := dte.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = true", datasetVersion).
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	dte.ruleCache[cacheKey] = rules

	return rules, nil
}

// getATCLevelForCheckLevel converts check level to ATC hierarchy level
func (dte *DuplicateTherapyEngine) getATCLevelForCheckLevel(checkLevel string) int {
	switch checkLevel {
	case "strict":
		return 5 // ATC-5: Chemical substance level
	case "broad":
		return 3 // ATC-3: Pharmacological subgroup
	case "moderate":
		fallthrough
	default:
		return 4 // ATC-4: Chemical subgroup
	}
}

// groupByTherapeuticClass groups drugs by their therapeutic class at specified ATC level
func (dte *DuplicateTherapyEngine) groupByTherapeuticClass(
	classes []DrugTherapeuticMapping,
	atcLevel int,
) map[string][]DuplicateDrugInfo {
	groups := make(map[string][]DuplicateDrugInfo)

	for _, class := range classes {
		if class.ATCLevel > atcLevel {
			continue
		}

		// Truncate ATC code to specified level
		atcKey := dte.truncateATCCode(class.ATCCode, atcLevel)
		if atcKey == "" {
			continue
		}

		drugInfo := DuplicateDrugInfo{
			DrugCode:         class.DrugCode,
			DrugName:         class.DrugName,
			ATCCode:          class.ATCCode,
			TherapeuticClass: class.TherapeuticClass,
		}

		groups[atcKey] = append(groups[atcKey], drugInfo)
	}

	// Remove duplicate entries for same drug (can have multiple ATC codes)
	for atcKey, drugs := range groups {
		groups[atcKey] = dte.deduplicateDrugs(drugs)
	}

	return groups
}

// truncateATCCode truncates an ATC code to specified level
func (dte *DuplicateTherapyEngine) truncateATCCode(atcCode string, level int) string {
	// ATC code structure:
	// Level 1: 1 char (A = Alimentary)
	// Level 2: 3 chars (A10 = Drugs used in diabetes)
	// Level 3: 4 chars (A10B = Blood glucose lowering drugs)
	// Level 4: 5 chars (A10BA = Biguanides)
	// Level 5: 7 chars (A10BA02 = Metformin)

	lengths := map[int]int{1: 1, 2: 3, 3: 4, 4: 5, 5: 7}
	targetLen, exists := lengths[level]
	if !exists {
		return atcCode
	}

	if len(atcCode) >= targetLen {
		return atcCode[:targetLen]
	}
	return atcCode
}

// deduplicateDrugs removes duplicate drug entries
func (dte *DuplicateTherapyEngine) deduplicateDrugs(drugs []DuplicateDrugInfo) []DuplicateDrugInfo {
	seen := make(map[string]bool)
	result := []DuplicateDrugInfo{}

	for _, drug := range drugs {
		if !seen[drug.DrugCode] {
			seen[drug.DrugCode] = true
			result = append(result, drug)
		}
	}

	return result
}

// findApplicableRule finds the most specific rule for an ATC code
func (dte *DuplicateTherapyEngine) findApplicableRule(
	rules []DuplicateTherapyRule,
	atcCode string,
) *DuplicateTherapyRule {
	var bestMatch *DuplicateTherapyRule
	bestMatchLen := 0

	for i := range rules {
		rule := &rules[i]
		pattern := strings.ToUpper(rule.ATCCodePattern)
		code := strings.ToUpper(atcCode)

		// Check if rule pattern matches (prefix match)
		if strings.HasPrefix(code, pattern) || strings.HasPrefix(pattern, code) {
			matchLen := len(pattern)
			if matchLen > bestMatchLen {
				bestMatch = rule
				bestMatchLen = matchLen
			}
		}
	}

	return bestMatch
}

// isAllowedCombination checks if a drug combination is clinically allowed
func (dte *DuplicateTherapyEngine) isAllowedCombination(
	rule *DuplicateTherapyRule,
	drugs []DuplicateDrugInfo,
) bool {
	if rule == nil {
		return false
	}

	// Check if multiple agents are allowed for this class
	if rule.AllowMultiple && len(drugs) <= rule.MaxAllowedAgents {
		return true
	}

	// Check specific exception combinations
	if rule.ExceptionCombinations != nil {
		// Parse exception combinations and check
		// This would check if the specific drug combination is in the exceptions list
		// Simplified implementation - expand based on clinical needs
		return false
	}

	return false
}

// buildDuplicateResult creates a duplicate therapy result
func (dte *DuplicateTherapyEngine) buildDuplicateResult(
	atcCode string,
	drugs []DuplicateDrugInfo,
	rule *DuplicateTherapyRule,
	isAllowed bool,
) DuplicateTherapyResult {
	result := DuplicateTherapyResult{
		DuplicateType:      "therapeutic_class",
		ATCCode:            atcCode,
		DuplicateDrugs:     drugs,
		AllowedCombination: isAllowed,
	}

	// Get therapeutic class name from first drug or rule
	if len(drugs) > 0 {
		result.TherapeuticClass = drugs[0].TherapeuticClass
	}

	if rule != nil {
		result.TherapeuticClass = rule.TherapeuticClassName
		result.Severity = rule.Severity
		result.ClinicalRationale = rule.ClinicalRationale
		result.ManagementStrategy = rule.ManagementStrategy
		result.Evidence = rule.Evidence
		result.RequiresPharmacistReview = rule.Severity == models.SeverityContraindicated || rule.Severity == models.SeverityMajor
	} else {
		// Default values when no specific rule exists
		result.Severity = models.SeverityModerate
		result.ClinicalRationale = fmt.Sprintf("Multiple drugs from the same therapeutic class (%s) detected. May result in additive effects, increased toxicity, or unnecessary cost.", result.TherapeuticClass)
		result.ManagementStrategy = "Review the clinical need for multiple agents in this class. Consider therapeutic substitution or discontinuation of redundant therapy."
		result.Evidence = models.EvidenceLevelC
		result.RequiresPharmacistReview = true
	}

	return result
}

// findExactDuplicates finds drugs that appear multiple times
func (dte *DuplicateTherapyEngine) findExactDuplicates(drugCodes []string) map[string]int {
	counts := make(map[string]int)
	for _, code := range drugCodes {
		normalized := strings.ToUpper(code)
		counts[normalized]++
	}
	return counts
}

// ClearCache clears all caches
func (dte *DuplicateTherapyEngine) ClearCache() {
	dte.classCache = make(map[string][]DrugTherapeuticMapping)
	dte.ruleCache = make(map[string][]DuplicateTherapyRule)
	dte.lastLoad = time.Time{}
}

// GetATCHierarchy returns the full ATC hierarchy for a drug
func (dte *DuplicateTherapyEngine) GetATCHierarchy(
	ctx context.Context,
	drugCode string,
	datasetVersion string,
) (map[int]string, error) {
	classes, err := dte.GetDrugTherapeuticClasses(ctx, drugCode, datasetVersion)
	if err != nil {
		return nil, err
	}

	if len(classes) == 0 {
		return nil, nil
	}

	// Build hierarchy from the most specific ATC code
	var mostSpecific DrugTherapeuticMapping
	for _, class := range classes {
		if class.ATCLevel > mostSpecific.ATCLevel {
			mostSpecific = class
		}
	}

	hierarchy := make(map[int]string)
	atcCode := mostSpecific.ATCCode

	// Generate all levels from the ATC code
	for level := 1; level <= 5; level++ {
		truncated := dte.truncateATCCode(atcCode, level)
		if truncated != "" && len(truncated) >= level {
			hierarchy[level] = truncated
		}
	}

	return hierarchy, nil
}

// Common therapeutic duplicate patterns for reference (built-in knowledge)
var commonDuplicatePatterns = map[string]struct {
	ClassName    string
	Severity     models.DDISeverity
	Rationale    string
	AllowMultiple bool
}{
	"C03":   {"Diuretics", models.SeverityMajor, "Multiple diuretics increase risk of electrolyte imbalances and dehydration", false},
	"C07":   {"Beta-blockers", models.SeverityMajor, "Multiple beta-blockers increase risk of bradycardia and hypotension", false},
	"C08":   {"Calcium channel blockers", models.SeverityMajor, "Multiple CCBs increase risk of hypotension and heart block", false},
	"C09A":  {"ACE inhibitors", models.SeverityMajor, "Multiple ACE inhibitors increase risk of hyperkalemia and angioedema", false},
	"C09C":  {"ARBs", models.SeverityMajor, "Multiple ARBs increase risk of hyperkalemia and hypotension", false},
	"N05A":  {"Antipsychotics", models.SeverityMajor, "Multiple antipsychotics increase QT prolongation and metabolic risk", false},
	"N05B":  {"Anxiolytics", models.SeverityMajor, "Multiple benzodiazepines increase sedation and respiratory depression risk", false},
	"N05C":  {"Hypnotics/Sedatives", models.SeverityMajor, "Multiple sedatives increase fall risk and respiratory depression", false},
	"N06A":  {"Antidepressants", models.SeverityModerate, "Multiple antidepressants may be appropriate but require monitoring", true},
	"M01A":  {"NSAIDs", models.SeverityMajor, "Multiple NSAIDs increase GI bleeding and renal toxicity risk", false},
	"A10BA": {"Biguanides", models.SeverityMajor, "Multiple biguanides increase lactic acidosis risk", false},
	"B01A":  {"Antithrombotic agents", models.SeverityMajor, "Multiple antithrombotics significantly increase bleeding risk", false},
	"J01":   {"Antibacterials", models.SeverityModerate, "Multiple antibiotics may be appropriate for severe infections", true},
	"H02":   {"Corticosteroids", models.SeverityModerate, "Multiple systemic corticosteroids increase adverse effect risk", false},
	"R03":   {"Asthma/COPD medications", models.SeverityModerate, "Multiple bronchodilators may be appropriate in severe disease", true},
}
