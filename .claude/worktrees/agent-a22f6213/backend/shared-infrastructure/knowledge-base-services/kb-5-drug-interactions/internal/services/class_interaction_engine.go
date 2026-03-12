package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"


	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/metrics"
)

// ClassInteractionEngine evaluates drug class-based interactions
type ClassInteractionEngine struct {
	db              *database.Database
	metrics         *metrics.Collector
	
	// Cache for drug class mappings and rules
	drugClassCache  map[string][]string // drug_code -> ATC classes
	classRuleCache  map[string][]models.DDIClassRule
	cacheTTL        time.Duration
	lastCacheUpdate time.Time
}

// NewClassInteractionEngine creates a new class interaction evaluation engine
func NewClassInteractionEngine(db *database.Database, metrics *metrics.Collector) *ClassInteractionEngine {
	return &ClassInteractionEngine{
		db:              db,
		metrics:         metrics,
		drugClassCache:  make(map[string][]string),
		classRuleCache:  make(map[string][]models.DDIClassRule),
		cacheTTL:        15 * time.Minute,
	}
}

// EvaluateClassInteractions evaluates drug class-based interactions
func (cie *ClassInteractionEngine) EvaluateClassInteractions(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) ([]models.EnhancedInteractionResult, error) {
	timer := time.Now()
	defer func() {
		cie.metrics.RecordClassInteractionCheck("class_evaluation", time.Since(timer))
	}()

	// Resolve drug codes to their therapeutic classes
	drugToClasses, err := cie.resolveDrugClasses(ctx, drugCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve drug classes: %w", err)
	}

	// Get all unique classes involved
	allClasses := cie.getAllUniqueClasses(drugToClasses)

	// Load class interaction rules
	classRules, err := cie.loadClassRules(ctx, drugCodes, allClasses, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load class rules: %w", err)
	}

	var interactions []models.EnhancedInteractionResult

	// Evaluate class-to-class interactions
	classToClassInteractions := cie.evaluateClassToClassInteractions(classRules, drugToClasses, allClasses)
	interactions = append(interactions, classToClassInteractions...)

	// Evaluate drug-to-class interactions  
	drugToClassInteractions := cie.evaluateDrugToClassInteractions(classRules, drugCodes, allClasses)
	interactions = append(interactions, drugToClassInteractions...)

	cie.metrics.RecordClassInteractionsFound(len(interactions), "total")

	return interactions, nil
}

// GetDrugTherapeuticClasses returns therapeutic classes for a drug
func (cie *ClassInteractionEngine) GetDrugTherapeuticClasses(
	ctx context.Context,
	drugCode string,
) ([]TherapeuticClass, error) {
	// TODO: Integrate with KB-7 Terminology service for comprehensive class mapping
	// For now, use embedded ATC classification logic
	
	classes := cie.mapDrugToATCClasses(drugCode)
	
	var therapeuticClasses []TherapeuticClass
	for _, classCode := range classes {
		class := TherapeuticClass{
			Code:        classCode,
			Name:        cie.getATCClassName(classCode),
			Level:       cie.getATCLevel(classCode),
			Description: cie.getATCDescription(classCode),
		}
		therapeuticClasses = append(therapeuticClasses, class)
	}
	
	return therapeuticClasses, nil
}

// EvaluateTripleWhammy specifically checks for ACE inhibitor + Diuretic + NSAID combination
func (cie *ClassInteractionEngine) EvaluateTripleWhammy(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) (*models.EnhancedInteractionResult, bool) {
	// Identify therapeutic classes in the drug list
	hasACEInhibitor := false
	hasDiuretic := false
	hasNSAID := false
	
	for _, drugCode := range drugCodes {
		classes := cie.mapDrugToATCClasses(drugCode)
		
		for _, class := range classes {
			if strings.HasPrefix(class, "C09A") { // ACE inhibitors
				hasACEInhibitor = true
			}
			if strings.HasPrefix(class, "C03") { // Diuretics
				hasDiuretic = true
			}
			if strings.HasPrefix(class, "M01A") { // NSAIDs
				hasNSAID = true
			}
		}
	}

	// Check for triple whammy
	if hasACEInhibitor && hasDiuretic && hasNSAID {
		interaction := &models.EnhancedInteractionResult{
			InteractionID:      "TRIPLE_WHAMMY_AKI_RISK",
			Severity:           models.SeverityModerate,
			Mechanism:          models.MechanismPD,
			ClinicalEffects:    "Triple whammy combination significantly increases acute kidney injury risk through synergistic effects on renal perfusion",
			ManagementStrategy: "Avoid triple combination when possible. If necessary, monitor renal function closely (baseline, 3-7 days, then monthly). Consider acetaminophen instead of NSAID.",
			Evidence:           models.EvidenceLevelB,
			Drug1: models.DrugInfo{
				Code: "ATC:C09A_C03_M01A",
				Name: "ACE Inhibitor + Diuretic + NSAID",
			},
			Drug2: models.DrugInfo{
				Code: "TRIPLE_WHAMMY",
				Name: "Triple Whammy Combination",
			},
			Qualifiers: map[string]string{
				"interaction_type": "class_combination",
				"risk_category":   "nephrotoxicity",
				"clinical_name":   "triple_whammy",
			},
			MonitoringParameters: map[string]interface{}{
				"renal_function": map[string]interface{}{
					"frequency":      "baseline_3_to_7_days_then_monthly",
					"parameters":     []string{"serum_creatinine", "BUN", "eGFR"},
					"critical_values": []string{"creatinine_increase_50_percent", "eGFR_decrease_25_percent"},
				},
			},
		}
		
		confidence := decimal.NewFromFloat(0.85)
		interaction.Confidence = &confidence
		
		cie.metrics.RecordTripleWhammyDetection("detected")
		
		return interaction, true
	}

	return nil, false
}

// Private helper methods

func (cie *ClassInteractionEngine) resolveDrugClasses(
	ctx context.Context,
	drugCodes []string,
) (map[string][]string, error) {
	drugToClasses := make(map[string][]string)
	
	for _, drugCode := range drugCodes {
		// Check cache first
		if classes, exists := cie.drugClassCache[drugCode]; exists && 
			time.Since(cie.lastCacheUpdate) < cie.cacheTTL {
			drugToClasses[drugCode] = classes
			continue
		}
		
		// Map drug to ATC classes
		classes := cie.mapDrugToATCClasses(drugCode)
		drugToClasses[drugCode] = classes
		
		// Update cache
		cie.drugClassCache[drugCode] = classes
	}
	
	cie.lastCacheUpdate = time.Now()
	return drugToClasses, nil
}

func (cie *ClassInteractionEngine) mapDrugToATCClasses(drugCode string) []string {
	// TODO: Replace with KB-7 Terminology service integration
	// Static mapping for common drugs (temporary implementation)
	
	drugClassMapping := map[string][]string{
		// Cardiovascular drugs
		"RxCUI:1998":    {"C09AA01", "C09AA"},     // Lisinopril - ACE inhibitor
		"RxCUI:29046":   {"C09AA02", "C09AA"},     // Enalapril - ACE inhibitor
		"RxCUI:214354":  {"C09CA01", "C09CA"},     // Losartan - ARB
		"RxCUI:69749":   {"C07AB02", "C07AB"},     // Metoprolol - Beta blocker
		"RxCUI:149":     {"C07AB03", "C07AB"},     // Atenolol - Beta blocker
		"RxCUI:4603":    {"C03AA03", "C03AA"},     // Hydrochlorothiazide - Diuretic
		"RxCUI:38413":   {"C03CA01", "C03CA"},     // Furosemide - Loop diuretic
		"RxCUI:11289":   {"B01AA03", "B01AA"},     // Warfarin - Anticoagulant
		"RxCUI:32968":   {"C07AG02", "C07AG"},     // Carvedilol - Alpha/beta blocker
		
		// NSAIDs
		"RxCUI:5640":    {"M01AE01", "M01AE"},     // Ibuprofen - NSAID
		"RxCUI:7258":    {"M01AE02", "M01AE"},     // Naproxen - NSAID
		"RxCUI:2670":    {"M01AE07", "M01AE"},     // Diclofenac - NSAID
		"RxCUI:140587":  {"M01AH01", "M01AH"},     // Celecoxib - COX-2 inhibitor

		// Antidepressants
		"RxCUI:32937":   {"N06AB06", "N06AB"},     // Sertraline - SSRI
		"RxCUI:36437":   {"N06AB10", "N06AB"},     // Escitalopram - SSRI
		"RxCUI:30203":   {"N06AB05", "N06AB"},     // Paroxetine - SSRI (changed from duplicate 5640)
		"RxCUI:31565":   {"N06AB04", "N06AB"},     // Citalopram - SSRI
		"RxCUI:5781":    {"N06AF03", "N06AF"},     // Phenelzine - MAOI
		
		// Anticoagulants/Antiplatelets
		"RxCUI:1154343": {"B01AC04", "B01AC"},     // Clopidogrel - Antiplatelet
		"RxCUI:1246289": {"B01AC22", "B01AC"},     // Prasugrel - Antiplatelet  
		"RxCUI:855332":  {"B01AC05", "B01AC"},     // Aspirin - Antiplatelet
		
		// Statins
		"RxCUI:36567":   {"C10AA01", "C10AA"},     // Simvastatin - HMG CoA reductase inhibitor
		"RxCUI:83367":   {"C10AA05", "C10AA"},     // Atorvastatin - HMG CoA reductase inhibitor
		"RxCUI:42463":   {"C10AA03", "C10AA"},     // Pravastatin - HMG CoA reductase inhibitor
		"RxCUI:446503":  {"C10AA07", "C10AA"},     // Rosuvastatin - HMG CoA reductase inhibitor
	}

	if classes, exists := drugClassMapping[drugCode]; exists {
		return classes
	}
	
	// If no specific mapping, try to infer from RxCUI patterns (basic fallback)
	return []string{}
}

func (cie *ClassInteractionEngine) loadClassRules(
	ctx context.Context,
	drugCodes []string,
	classCodes []string,
	datasetVersion string,
) ([]models.DDIClassRule, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%v:%v", datasetVersion, drugCodes, classCodes)
	if rules, exists := cie.classRuleCache[cacheKey]; exists && 
		time.Since(cie.lastCacheUpdate) < cie.cacheTTL {
		return rules, nil
	}

	var rules []models.DDIClassRule
	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	err := cie.db.DB.WithContext(ctx).
		Where(`dataset_version = ? AND active = TRUE AND (
			(object_type = 'drug' AND object_code IN ?) OR
			(object_type = 'class' AND object_code IN ?)
		) AND (
			(subject_type = 'drug' AND subject_code IN ?) OR
			(subject_type = 'class' AND subject_code IN ?)
		)`, datasetVersion, drugCodes, classCodes, drugCodes, classCodes).
		Order("severity DESC, confidence DESC").
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	cie.classRuleCache[cacheKey] = rules

	return rules, nil
}

func (cie *ClassInteractionEngine) evaluateClassToClassInteractions(
	rules []models.DDIClassRule,
	drugToClasses map[string][]string,
	allClasses []string,
) []models.EnhancedInteractionResult {
	var interactions []models.EnhancedInteractionResult

	// Check each rule for class-to-class matches
	for _, rule := range rules {
		if rule.ObjectType == "class" && rule.SubjectType == "class" {
			// Check if we have drugs from both classes
			hasObjectClass := cie.hasDrugsInClass(drugToClasses, rule.ObjectCode)
			hasSubjectClass := cie.hasDrugsInClass(drugToClasses, rule.SubjectCode)
			
			if hasObjectClass && hasSubjectClass {
				interaction := cie.convertClassRuleToInteraction(rule, "class_to_class")
				interactions = append(interactions, interaction)
				
				cie.metrics.RecordClassInteraction(rule.ObjectCode, rule.SubjectCode, string(rule.Severity))
			}
		}
	}

	return interactions
}

func (cie *ClassInteractionEngine) evaluateDrugToClassInteractions(
	rules []models.DDIClassRule,
	drugCodes []string,
	classCodes []string,
) []models.EnhancedInteractionResult {
	var interactions []models.EnhancedInteractionResult

	for _, rule := range rules {
		// Drug-to-class interactions
		if rule.ObjectType == "drug" && rule.SubjectType == "class" {
			if cie.containsString(drugCodes, rule.ObjectCode) && cie.containsString(classCodes, rule.SubjectCode) {
				interaction := cie.convertClassRuleToInteraction(rule, "drug_to_class")
				interactions = append(interactions, interaction)
			}
		}
		
		// Class-to-drug interactions  
		if rule.ObjectType == "class" && rule.SubjectType == "drug" {
			if cie.containsString(classCodes, rule.ObjectCode) && cie.containsString(drugCodes, rule.SubjectCode) {
				interaction := cie.convertClassRuleToInteraction(rule, "class_to_drug")
				interactions = append(interactions, interaction)
			}
		}
	}

	return interactions
}

func (cie *ClassInteractionEngine) convertClassRuleToInteraction(
	rule models.DDIClassRule,
	interactionType string,
) models.EnhancedInteractionResult {
	interaction := models.EnhancedInteractionResult{
		InteractionID:      fmt.Sprintf("CLASS_%s_%s_%s", rule.ObjectCode, rule.SubjectCode, rule.DatasetVersion),
		Severity:           rule.Severity,
		Mechanism:          rule.Mechanism,
		ClinicalEffects:    getStringValue(rule.ClinicalEffects),
		ManagementStrategy: rule.ManagementStrategy,
		Evidence:           rule.Evidence,
		Confidence:         rule.Confidence,
		Drug1: models.DrugInfo{
			Code: rule.ObjectCode,
			Name: cie.getClassOrDrugName(rule.ObjectCode, rule.ObjectType),
		},
		Drug2: models.DrugInfo{
			Code: rule.SubjectCode,
			Name: cie.getClassOrDrugName(rule.SubjectCode, rule.SubjectType),
		},
	}

	// Add qualifiers from rule
	if rule.Qualifiers != nil {
		interaction.Qualifiers = make(map[string]string)
		for key, value := range *rule.Qualifiers {
			if valueStr, ok := value.(string); ok {
				interaction.Qualifiers[key] = valueStr
			}
		}
	}
	
	// Add interaction type qualifier
	if interaction.Qualifiers == nil {
		interaction.Qualifiers = make(map[string]string)
	}
	interaction.Qualifiers["interaction_type"] = interactionType
	interaction.Qualifiers["source"] = "class_rule"

	return interaction
}

// Clinical class interaction patterns

func (cie *ClassInteractionEngine) CheckAnticoagulantAntiplateleCombination(
	drugCodes []string,
) *models.EnhancedInteractionResult {
	hasAnticoagulant := false
	hasAntiplatelet := false
	
	for _, drugCode := range drugCodes {
		classes := cie.mapDrugToATCClasses(drugCode)
		for _, class := range classes {
			if strings.HasPrefix(class, "B01AA") { // Anticoagulants
				hasAnticoagulant = true
			}
			if strings.HasPrefix(class, "B01AC") { // Antiplatelets
				hasAntiplatelet = true
			}
		}
	}
	
	if hasAnticoagulant && hasAntiplatelet {
		interaction := &models.EnhancedInteractionResult{
			InteractionID:      "ANTICOAGULANT_ANTIPLATELET_BLEEDING",
			Severity:           models.SeverityMajor,
			Mechanism:          models.MechanismPD,
			ClinicalEffects:    "Additive bleeding risk through different hemostatic pathways - significantly increased hemorrhage risk",
			ManagementStrategy: "Use combination only when benefits clearly outweigh risks. Consider gastroprotection with PPI. Monitor for bleeding signs closely.",
			Evidence:           models.EvidenceLevelA,
			Drug1: models.DrugInfo{Code: "ATC:B01AA", Name: "Anticoagulants"},
			Drug2: models.DrugInfo{Code: "ATC:B01AC", Name: "Antiplatelets"},
			MonitoringParameters: map[string]interface{}{
				"bleeding_assessment": map[string]interface{}{
					"frequency":   "weekly_for_first_month_then_monthly",
					"parameters":  []string{"hemoglobin", "hematocrit", "bleeding_signs"},
					"critical":    []string{"hemoglobin_drop_2g", "melena", "hematemesis"},
				},
			},
		}
		
		confidence := decimal.NewFromFloat(0.90)
		interaction.Confidence = &confidence
		
		return interaction
	}
	
	return nil
}

func (cie *ClassInteractionEngine) CheckBenzodiazepineOpioidCombination(
	drugCodes []string,
) *models.EnhancedInteractionResult {
	hasBenzodiazepine := false
	hasOpioid := false
	
	for _, drugCode := range drugCodes {
		classes := cie.mapDrugToATCClasses(drugCode)
		for _, class := range classes {
			if strings.HasPrefix(class, "N05BA") { // Benzodiazepines
				hasBenzodiazepine = true
			}
			if strings.HasPrefix(class, "N02AA") { // Opioids
				hasOpioid = true
			}
		}
	}
	
	if hasBenzodiazepine && hasOpioid {
		interaction := &models.EnhancedInteractionResult{
			InteractionID:      "BENZODIAZEPINE_OPIOID_CNS_DEPRESSION",
			Severity:           models.SeverityMajor,
			Mechanism:          models.MechanismPD,
			ClinicalEffects:    "Additive CNS depression leading to severe respiratory depression, coma, and death. FDA black box warning applies.",
			ManagementStrategy: "AVOID combination when possible. If necessary, use lowest effective doses with close monitoring. Consider naloxone availability.",
			Evidence:           models.EvidenceLevelA,
			Drug1: models.DrugInfo{Code: "ATC:N05BA", Name: "Benzodiazepines"},
			Drug2: models.DrugInfo{Code: "ATC:N02AA", Name: "Opioid Analgesics"},
			MonitoringParameters: map[string]interface{}{
				"respiratory_monitoring": map[string]interface{}{
					"frequency":   "continuous_if_inpatient_frequent_if_outpatient",
					"parameters":  []string{"respiratory_rate", "oxygen_saturation", "sedation_level"},
					"critical":    []string{"respiratory_rate_less_than_12", "oxygen_sat_less_than_90"},
				},
			},
		}
		
		confidence := decimal.NewFromFloat(0.95)
		interaction.Confidence = &confidence
		
		return interaction
	}
	
	return nil
}

// Utility methods

func (cie *ClassInteractionEngine) getAllUniqueClasses(drugToClasses map[string][]string) []string {
	classSet := make(map[string]bool)
	for _, classes := range drugToClasses {
		for _, class := range classes {
			classSet[class] = true
		}
	}
	
	var allClasses []string
	for class := range classSet {
		allClasses = append(allClasses, class)
	}
	
	return allClasses
}

func (cie *ClassInteractionEngine) hasDrugsInClass(drugToClasses map[string][]string, targetClass string) bool {
	for _, classes := range drugToClasses {
		for _, class := range classes {
			if class == targetClass || strings.HasPrefix(class, targetClass) {
				return true
			}
		}
	}
	return false
}

func (cie *ClassInteractionEngine) containsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func (cie *ClassInteractionEngine) getClassOrDrugName(code, codeType string) string {
	if codeType == "class" {
		return cie.getATCClassName(code)
	}
	return cie.getDrugName(code) // TODO: Integrate with KB-7 Terminology
}

func (cie *ClassInteractionEngine) getATCClassName(atcCode string) string {
	// ATC classification names (subset for common classes)
	atcNames := map[string]string{
		// Cardiovascular
		"C09AA": "ACE Inhibitors",
		"C09CA": "Angiotensin II Receptor Blockers",
		"C07AB": "Beta Blocking Agents",
		"C03AA": "Thiazide Diuretics",
		"C03CA": "Loop Diuretics",
		"B01AA": "Vitamin K Antagonists",
		"B01AC": "Platelet Aggregation Inhibitors",
		
		// Anti-inflammatory
		"M01AE": "Propionic Acid Derivatives (NSAIDs)",
		"M01AH": "COX-2 Inhibitors",
		"M01A":  "Anti-inflammatory and Antirheumatic Products",
		
		// CNS
		"N06AB": "Selective Serotonin Reuptake Inhibitors",
		"N06AF": "Monoamine Oxidase Inhibitors",
		"N05BA": "Benzodiazepine Derivatives",
		"N02AA": "Natural Opium Alkaloids",
		
		// Lipid modifying
		"C10AA": "HMG CoA Reductase Inhibitors",
	}

	if name, exists := atcNames[atcCode]; exists {
		return name
	}
	
	return atcCode // Return code if name not found
}

func (cie *ClassInteractionEngine) getATCLevel(atcCode string) int {
	// ATC hierarchy levels: 1 (anatomical) -> 5 (chemical substance)
	return len(atcCode)
}

func (cie *ClassInteractionEngine) getATCDescription(atcCode string) string {
	// TODO: Replace with comprehensive ATC description database
	descriptions := map[string]string{
		"C09AA": "ACE inhibitors, plain - reduce angiotensin II formation",
		"C09CA": "Angiotensin II receptor blockers - block AT1 receptors", 
		"C07AB": "Beta blocking agents, selective - cardioselective beta-1 blockade",
		"B01AA": "Vitamin K antagonists - inhibit vitamin K-dependent clotting factors",
		"B01AC": "Platelet aggregation inhibitors - prevent platelet activation",
		"N06AB": "SSRIs - selective serotonin reuptake inhibition",
		"N05BA": "Benzodiazepines - GABA-A receptor positive allosteric modulators",
		"M01AE": "Propionic acid NSAIDs - COX-1/COX-2 inhibition",
	}

	if desc, exists := descriptions[atcCode]; exists {
		return desc
	}
	
	return "ATC therapeutic class"
}

func (cie *ClassInteractionEngine) getDrugName(drugCode string) string {
	// TODO: Replace with KB-7 Terminology service integration
	drugNames := map[string]string{
		"RxCUI:1998":    "Lisinopril",
		"RxCUI:29046":   "Enalapril", 
		"RxCUI:214354":  "Losartan",
		"RxCUI:69749":   "Metoprolol",
		"RxCUI:149":     "Atenolol",
		"RxCUI:4603":    "Hydrochlorothiazide",
		"RxCUI:38413":   "Furosemide",
		"RxCUI:11289":   "Warfarin",
		"RxCUI:5640":    "Ibuprofen",
		"RxCUI:32937":   "Sertraline",
		"RxCUI:1154343": "Clopidogrel",
		"RxCUI:36567":   "Simvastatin",
	}

	if name, exists := drugNames[drugCode]; exists {
		return name
	}
	
	return drugCode
}

// Supporting data structures

// TherapeuticClass represents a drug's therapeutic classification
type TherapeuticClass struct {
	Code        string `json:"code"`        // ATC code
	Name        string `json:"name"`        // Human-readable name
	Level       int    `json:"level"`       // ATC hierarchy level (1-5)
	Description string `json:"description"` // Detailed description
	ParentCode  string `json:"parent_code,omitempty"` // Parent ATC code
}

// ClassInteractionPattern represents a pattern for drug class interactions
type ClassInteractionPattern struct {
	PatternName       string                  `json:"pattern_name"`
	ObjectClass       string                  `json:"object_class"`
	SubjectClass      string                  `json:"subject_class"`
	Severity          models.DDISeverity      `json:"severity"`
	Mechanism         models.MechanismType    `json:"mechanism"`
	ClinicalRisk      string                  `json:"clinical_risk"`
	ManagementStrategy string                 `json:"management_strategy"`
	MonitoringRequired bool                   `json:"monitoring_required"`
	Evidence          models.EvidenceLevel    `json:"evidence"`
	Confidence        decimal.Decimal         `json:"confidence"`
}

// GetCommonClassInteractionPatterns returns clinically important class interaction patterns
func (cie *ClassInteractionEngine) GetCommonClassInteractionPatterns() []ClassInteractionPattern {
	return []ClassInteractionPattern{
		{
			PatternName:        "ACE_Inhibitor_NSAID_Nephrotoxicity",
			ObjectClass:        "C09AA", // ACE Inhibitors
			SubjectClass:       "M01A",  // NSAIDs
			Severity:           models.SeverityModerate,
			Mechanism:          models.MechanismPD,
			ClinicalRisk:       "Reduced renal perfusion, decreased antihypertensive effect, acute kidney injury risk",
			ManagementStrategy: "Avoid chronic combination. Monitor BP and renal function. Prefer acetaminophen for pain.",
			MonitoringRequired: true,
			Evidence:           models.EvidenceLevelB,
			Confidence:         decimal.NewFromFloat(0.85),
		},
		{
			PatternName:        "Anticoagulant_Antiplatelet_Bleeding",
			ObjectClass:        "B01AA", // Anticoagulants
			SubjectClass:       "B01AC", // Antiplatelets
			Severity:           models.SeverityMajor,
			Mechanism:          models.MechanismPD,
			ClinicalRisk:       "Additive bleeding risk through different hemostatic mechanisms",
			ManagementStrategy: "Use only when benefits outweigh risks. Gastroprotection recommended. Monitor bleeding closely.",
			MonitoringRequired: true,
			Evidence:           models.EvidenceLevelA,
			Confidence:         decimal.NewFromFloat(0.90),
		},
		{
			PatternName:        "CNS_Depressant_Combination",
			ObjectClass:        "N05BA", // Benzodiazepines
			SubjectClass:       "N02AA", // Opioids
			Severity:           models.SeverityMajor,
			Mechanism:          models.MechanismPD,
			ClinicalRisk:       "Severe CNS depression, respiratory depression, coma, death",
			ManagementStrategy: "AVOID when possible. If necessary, use lowest doses with continuous monitoring. FDA black box warning.",
			MonitoringRequired: true,
			Evidence:           models.EvidenceLevelA,
			Confidence:         decimal.NewFromFloat(0.95),
		},
		{
			PatternName:        "Serotonergic_Drug_Combination",
			ObjectClass:        "N06AB", // SSRIs
			SubjectClass:       "N06AF", // MAOIs
			Severity:           models.SeverityContraindicated,
			Mechanism:          models.MechanismPD,
			ClinicalRisk:       "Serotonin syndrome - potentially fatal hyperthermia, rigidity, altered mental status",
			ManagementStrategy: "CONTRAINDICATED. Minimum 14-day washout required between MAOI and SSRI.",
			MonitoringRequired: false,
			Evidence:           models.EvidenceLevelA,
			Confidence:         decimal.NewFromFloat(0.98),
		},
	}
}