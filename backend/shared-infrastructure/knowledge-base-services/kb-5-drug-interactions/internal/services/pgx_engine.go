package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/metrics"
)

// PharmacogenomicEngine evaluates patient-specific genetic interactions
type PharmacogenomicEngine struct {
	db      *database.Database
	metrics *metrics.Collector
	
	// Cache for PGx rules to avoid repeated database queries
	ruleCache map[string][]models.DDIPharmacogenomicRule
	cacheTTL  time.Duration
	lastLoad  time.Time
}

// NewPharmacogenomicEngine creates a new PGx evaluation engine
func NewPharmacogenomicEngine(db *database.Database, metrics *metrics.Collector) *PharmacogenomicEngine {
	return &PharmacogenomicEngine{
		db:        db,
		metrics:   metrics,
		ruleCache: make(map[string][]models.DDIPharmacogenomicRule),
		cacheTTL:  30 * time.Minute,
	}
}

// EvaluatePatientPGXInteractions evaluates pharmacogenomic interactions for a patient
func (pge *PharmacogenomicEngine) EvaluatePatientPGXInteractions(
	ctx context.Context,
	drugCodes []string,
	patientPGX map[string]string,
	datasetVersion string,
) ([]models.EnhancedInteractionResult, error) {
	if len(patientPGX) == 0 {
		return []models.EnhancedInteractionResult{}, nil
	}

	timer := time.Now()
	defer func() {
		pge.metrics.RecordPGXEvaluation(time.Since(timer), len(patientPGX))
	}()

	// Load relevant PGx rules for the drugs
	pgxRules, err := pge.loadPGXRules(ctx, drugCodes, datasetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to load PGx rules: %w", err)
	}

	var interactions []models.EnhancedInteractionResult

	// Evaluate each rule against patient's genetic profile
	for _, rule := range pgxRules {
		if interaction := pge.evaluatePGXRule(rule, patientPGX); interaction != nil {
			interactions = append(interactions, *interaction)
			pge.metrics.RecordPGXInteraction(rule.Gene, string(rule.Severity))
		}
	}

	return interactions, nil
}

// EvaluateDrugMetabolism provides drug metabolism assessment based on patient PGx
func (pge *PharmacogenomicEngine) EvaluateDrugMetabolism(
	ctx context.Context,
	drugCode string,
	patientPGX map[string]string,
	datasetVersion string,
) (*PharmacogenomicMetabolismAssessment, error) {
	// Load PGx rules for specific drug
	pgxRules, err := pge.loadPGXRulesForDrug(ctx, drugCode, datasetVersion)
	if err != nil {
		return nil, err
	}

	assessment := &PharmacogenomicMetabolismAssessment{
		DrugCode:           drugCode,
		PatientPGX:         patientPGX,
		MetabolismStatus:   "normal",
		RiskFactors:        []string{},
		Recommendations:    []string{},
		RequiresMonitoring: false,
		AlternativeDrugs:   []string{},
	}

	// Evaluate each relevant genetic marker
	for _, rule := range pgxRules {
		if patientPhenotype, exists := patientPGX[rule.Gene]; exists {
			metabolismImpact := pge.assessMetabolismImpact(rule, patientPhenotype)
			pge.incorporateMetabolismFindings(assessment, metabolismImpact, rule)
		}
	}

	return assessment, nil
}

// GetSupportedPGXMarkers returns list of supported pharmacogenomic markers
func (pge *PharmacogenomicEngine) GetSupportedPGXMarkers(
	ctx context.Context,
	datasetVersion string,
) (map[string][]string, error) {
	var rules []models.DDIPharmacogenomicRule
	err := pge.db.DB.WithContext(ctx).
		Select("DISTINCT gene, phenotype").
		Where("dataset_version = ? AND active = TRUE", datasetVersion).
		Find(&rules).Error
	
	if err != nil {
		return nil, err
	}

	markers := make(map[string][]string)
	for _, rule := range rules {
		if _, exists := markers[rule.Gene]; !exists {
			markers[rule.Gene] = []string{}
		}
		
		// Avoid duplicates
		phenotypeExists := false
		for _, existing := range markers[rule.Gene] {
			if existing == rule.Phenotype {
				phenotypeExists = true
				break
			}
		}
		
		if !phenotypeExists {
			markers[rule.Gene] = append(markers[rule.Gene], rule.Phenotype)
		}
	}

	return markers, nil
}

// Private helper methods

func (pge *PharmacogenomicEngine) loadPGXRules(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) ([]models.DDIPharmacogenomicRule, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%v", datasetVersion, drugCodes)
	if rules, exists := pge.ruleCache[cacheKey]; exists && time.Since(pge.lastLoad) < pge.cacheTTL {
		return rules, nil
	}

	// Load from database
	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	var rules []models.DDIPharmacogenomicRule
	err := pge.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = TRUE AND drug_code IN ?",
			datasetVersion, drugCodes).
		Order("severity DESC, confidence DESC").
		Find(&rules).Error

	if err != nil {
		return nil, err
	}

	// Update cache
	pge.ruleCache[cacheKey] = rules
	pge.lastLoad = time.Now()

	return rules, nil
}

func (pge *PharmacogenomicEngine) loadPGXRulesForDrug(
	ctx context.Context,
	drugCode string,
	datasetVersion string,
) ([]models.DDIPharmacogenomicRule, error) {
	var rules []models.DDIPharmacogenomicRule
	err := pge.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = TRUE AND drug_code = ?", 
			datasetVersion, drugCode).
		Order("severity DESC").
		Find(&rules).Error

	return rules, err
}

func (pge *PharmacogenomicEngine) evaluatePGXRule(
	rule models.DDIPharmacogenomicRule,
	patientPGX map[string]string,
) *models.EnhancedInteractionResult {
	// Check if patient has the relevant genetic variant
	patientPhenotype, exists := patientPGX[rule.Gene]
	if !exists || patientPhenotype != rule.Phenotype {
		return nil
	}

	// Create interaction result for PGx finding
	interaction := &models.EnhancedInteractionResult{
		InteractionID:      fmt.Sprintf("PGX_%s_%s_%s", rule.DrugCode, rule.Gene, rule.Phenotype),
		Severity:           rule.Severity,
		Mechanism:          models.MechanismPK, // PGx interactions are typically pharmacokinetic
		ClinicalEffects:    getStringValue(rule.ClinicalEffects),
		ManagementStrategy: rule.ManagementStrategy,
		Evidence:           rule.Evidence,
		PGXApplicable:      true,
		Drug1: models.DrugInfo{
			Code: rule.DrugCode,
			Name: pge.getDrugName(rule.DrugCode), // TODO: Integrate with KB-7 Terminology
		},
		Drug2: models.DrugInfo{
			Code: fmt.Sprintf("PGX_%s_%s", rule.Gene, rule.Phenotype),
			Name: fmt.Sprintf("%s %s Metabolizer", rule.Gene, rule.Phenotype),
		},
	}

	// Add qualifiers for PGx context
	interaction.Qualifiers = map[string]string{
		"gene":      rule.Gene,
		"phenotype": rule.Phenotype,
		"type":      "pharmacogenomic",
	}

	// Add monitoring parameters for high-risk PGx interactions
	if rule.Severity == models.SeverityContraindicated || rule.Severity == models.SeverityMajor {
		interaction.MonitoringParameters = pge.getPGXMonitoringParameters(rule)
	}

	return interaction
}

func (pge *PharmacogenomicEngine) assessMetabolismImpact(
	rule models.DDIPharmacogenomicRule,
	patientPhenotype string,
) PharmacogenomicMetabolismImpact {
	impact := PharmacogenomicMetabolismImpact{
		Gene:      rule.Gene,
		Phenotype: patientPhenotype,
		DrugCode:  rule.DrugCode,
		Severity:  rule.Severity,
	}

	// Assess metabolism status based on phenotype
	switch strings.ToLower(patientPhenotype) {
	case "poor", "pm":
		impact.MetabolismStatus = "poor"
		impact.ExposureChange = "increased"
		impact.RiskLevel = "high"
	case "intermediate", "im":
		impact.MetabolismStatus = "intermediate"
		impact.ExposureChange = "slightly_increased"
		impact.RiskLevel = "moderate"
	case "ultrarapid", "um":
		impact.MetabolismStatus = "ultrarapid"
		impact.ExposureChange = "decreased"
		impact.RiskLevel = "high"
	default:
		impact.MetabolismStatus = "normal"
		impact.ExposureChange = "none"
		impact.RiskLevel = "low"
	}

	// Add gene-specific assessments
	switch rule.Gene {
	case "CYP2D6":
		impact = pge.assessCYP2D6Impact(impact, rule)
	case "CYP2C19":
		impact = pge.assessCYP2C19Impact(impact, rule)
	case "SLCO1B1":
		impact = pge.assessSLCO1B1Impact(impact, rule)
	case "CYP3A5":
		impact = pge.assessCYP3A5Impact(impact, rule)
	}

	return impact
}

func (pge *PharmacogenomicEngine) assessCYP2D6Impact(
	impact PharmacogenomicMetabolismImpact,
	rule models.DDIPharmacogenomicRule,
) PharmacogenomicMetabolismImpact {
	// CYP2D6-specific logic for common substrates
	drugSpecificGuidance := map[string]map[string]string{
		"RxCUI:2670": { // Codeine
			"ultrarapid": "AVOID: Toxic morphine levels - use morphine/oxycodone instead",
			"poor":       "AVOID: Inadequate analgesia - use non-CYP2D6 opioid",
		},
		"RxCUI:4493": { // Metoprolol
			"ultrarapid": "May need higher doses - monitor BP/HR closely",
			"poor":       "Reduce dose 50% - monitor for bradycardia",
		},
		"RxCUI:5640": { // Paroxetine
			"ultrarapid": "Standard dosing usually adequate",
			"poor":       "Consider dose reduction or alternative SSRI",
		},
	}

	if guidance, exists := drugSpecificGuidance[rule.DrugCode]; exists {
		if specific, found := guidance[impact.Phenotype]; found {
			impact.SpecificRecommendation = specific
		}
	}

	return impact
}

func (pge *PharmacogenomicEngine) assessCYP2C19Impact(
	impact PharmacogenomicMetabolismImpact,
	rule models.DDIPharmacogenomicRule,
) PharmacogenomicMetabolismImpact {
	// CYP2C19-specific logic for antiplatelet drugs
	drugSpecificGuidance := map[string]map[string]string{
		"RxCUI:1154343": { // Clopidogrel
			"poor":       "AVOID: Inadequate antiplatelet effect - use prasugrel/ticagrelor",
			"ultrarapid": "Standard dosing adequate - enhanced antiplatelet effect",
		},
		"RxCUI:197696": { // Omeprazole
			"poor":       "Consider H2 blocker alternative - prolonged acid suppression",
			"ultrarapid": "Standard dosing adequate",
		},
	}

	if guidance, exists := drugSpecificGuidance[rule.DrugCode]; exists {
		if specific, found := guidance[impact.Phenotype]; found {
			impact.SpecificRecommendation = specific
		}
	}

	return impact
}

func (pge *PharmacogenomicEngine) assessSLCO1B1Impact(
	impact PharmacogenomicMetabolismImpact,
	rule models.DDIPharmacogenomicRule,
) PharmacogenomicMetabolismImpact {
	// SLCO1B1-specific logic for statins
	drugSpecificGuidance := map[string]map[string]string{
		"RxCUI:36567": { // Simvastatin
			"poor": "Reduce dose 50% or use pravastatin/rosuvastatin - myopathy risk",
		},
		"RxCUI:83367": { // Atorvastatin
			"poor": "Consider dose reduction - monitor for muscle symptoms",
		},
	}

	if guidance, exists := drugSpecificGuidance[rule.DrugCode]; exists {
		if specific, found := guidance[impact.Phenotype]; found {
			impact.SpecificRecommendation = specific
		}
	}

	return impact
}

func (pge *PharmacogenomicEngine) assessCYP3A5Impact(
	impact PharmacogenomicMetabolismImpact,
	rule models.DDIPharmacogenomicRule,
) PharmacogenomicMetabolismImpact {
	// CYP3A5-specific logic for immunosuppressants
	drugSpecificGuidance := map[string]map[string]string{
		"RxCUI:42316": { // Tacrolimus
			"expresser": "May need higher doses - monitor trough levels closely",
			"non_expresser": "Standard dosing - monitor trough levels",
		},
	}

	if guidance, exists := drugSpecificGuidance[rule.DrugCode]; exists {
		if specific, found := guidance[impact.Phenotype]; found {
			impact.SpecificRecommendation = specific
		}
	}

	return impact
}

func (pge *PharmacogenomicEngine) incorporateMetabolismFindings(
	assessment *PharmacogenomicMetabolismAssessment,
	impact PharmacogenomicMetabolismImpact,
	rule models.DDIPharmacogenomicRule,
) {
	// Update overall metabolism status
	if impact.RiskLevel == "high" {
		assessment.MetabolismStatus = "impaired"
		assessment.RequiresMonitoring = true
	}

	// Add risk factors
	if impact.ExposureChange != "none" {
		riskFactor := fmt.Sprintf("%s %s metabolizer (%s exposure)", 
			impact.Gene, impact.Phenotype, impact.ExposureChange)
		assessment.RiskFactors = append(assessment.RiskFactors, riskFactor)
	}

	// Add specific recommendations
	if impact.SpecificRecommendation != "" {
		assessment.Recommendations = append(assessment.Recommendations, impact.SpecificRecommendation)
	}

	// Add alternative drugs if contraindicated or high risk
	if rule.Severity == models.SeverityContraindicated || impact.RiskLevel == "high" {
		alternatives := pge.getAlternativeDrugsForPGX(rule.DrugCode, rule.Gene)
		assessment.AlternativeDrugs = append(assessment.AlternativeDrugs, alternatives...)
	}
}

func (pge *PharmacogenomicEngine) getAlternativeDrugsForPGX(drugCode, gene string) []string {
	// Clinical alternatives based on PGx considerations
	alternatives := map[string]map[string][]string{
		"CYP2D6": {
			"RxCUI:2670":  {"RxCUI:7052", "RxCUI:7804"},    // Codeine -> Morphine, Oxycodone
			"RxCUI:4493":  {"RxCUI:149", "RxCUI:32968"},     // Metoprolol -> Atenolol, Carvedilol
			"RxCUI:5640":  {"RxCUI:32937", "RxCUI:36437"},   // Paroxetine -> Sertraline, Escitalopram
		},
		"CYP2C19": {
			"RxCUI:1154343": {"RxCUI:1246289", "RxCUI:1156738"}, // Clopidogrel -> Prasugrel, Ticagrelor
			"RxCUI:197696":  {"RxCUI:1927851", "RxCUI:2047"},     // Omeprazole -> Pantoprazole, Ranitidine
		},
		"SLCO1B1": {
			"RxCUI:36567": {"RxCUI:42463", "RxCUI:446503"},   // Simvastatin -> Pravastatin, Rosuvastatin
			"RxCUI:83367": {"RxCUI:42463", "RxCUI:446503"},   // Atorvastatin -> Pravastatin, Rosuvastatin
		},
	}

	if geneAlts, exists := alternatives[gene]; exists {
		if drugAlts, found := geneAlts[drugCode]; found {
			return drugAlts
		}
	}

	return []string{}
}

func (pge *PharmacogenomicEngine) getPGXMonitoringParameters(rule models.DDIPharmacogenomicRule) map[string]interface{} {
	// Gene-specific monitoring parameters
	monitoring := map[string]map[string]interface{}{
		"CYP2D6": {
			"clinical_response": map[string]interface{}{
				"frequency": "weekly",
				"duration":  "4_weeks",
				"parameters": []string{"efficacy", "adverse_effects"},
			},
			"drug_levels": map[string]interface{}{
				"frequency": "as_needed",
				"indication": "if_poor_response_or_toxicity",
			},
		},
		"CYP2C19": {
			"platelet_function": map[string]interface{}{
				"frequency": "baseline_and_2_weeks",
				"test":      "P2Y12_assay_or_VerifyNow",
			},
			"clinical_events": map[string]interface{}{
				"frequency": "ongoing",
				"parameters": []string{"thrombotic_events", "bleeding"},
			},
		},
		"SLCO1B1": {
			"muscle_toxicity": map[string]interface{}{
				"frequency": "baseline_3_months_then_annually",
				"parameters": []string{"CK", "muscle_symptoms"},
			},
			"liver_function": map[string]interface{}{
				"frequency": "baseline_3_months",
				"parameters": []string{"ALT", "AST"},
			},
		},
	}

	if geneMonitoring, exists := monitoring[rule.Gene]; exists {
		return geneMonitoring
	}

	// Default monitoring for unknown genes
	return map[string]interface{}{
		"clinical_monitoring": map[string]interface{}{
			"frequency": "monthly",
			"duration":  "3_months",
			"parameters": []string{"efficacy", "adverse_effects"},
		},
	}
}

func (pge *PharmacogenomicEngine) getDrugName(drugCode string) string {
	// TODO: Integrate with KB-7 Terminology service for proper drug name resolution
	// For now, return the code as placeholder
	drugNames := map[string]string{
		"RxCUI:2670":    "Codeine",
		"RxCUI:4493":    "Metoprolol", 
		"RxCUI:5640":    "Paroxetine",
		"RxCUI:1154343": "Clopidogrel",
		"RxCUI:197696":  "Omeprazole",
		"RxCUI:36567":   "Simvastatin",
		"RxCUI:83367":   "Atorvastatin",
		"RxCUI:42316":   "Tacrolimus",
	}

	if name, exists := drugNames[drugCode]; exists {
		return name
	}
	return drugCode
}

func getStringValue(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// Supporting data structures

// PharmacogenomicMetabolismAssessment provides comprehensive PGx assessment
type PharmacogenomicMetabolismAssessment struct {
	DrugCode           string   `json:"drug_code"`
	PatientPGX         map[string]string `json:"patient_pgx"`
	MetabolismStatus   string   `json:"metabolism_status"` // "normal", "impaired", "enhanced"
	RiskFactors        []string `json:"risk_factors"`
	Recommendations    []string `json:"recommendations"`
	RequiresMonitoring bool     `json:"requires_monitoring"`
	AlternativeDrugs   []string `json:"alternative_drugs"`
	ConfidenceScore    float64  `json:"confidence_score"`
}

// PharmacogenomicMetabolismImpact represents the impact of a genetic variant
type PharmacogenomicMetabolismImpact struct {
	Gene                  string                  `json:"gene"`
	Phenotype            string                  `json:"phenotype"`
	DrugCode             string                  `json:"drug_code"`
	Severity             models.DDISeverity      `json:"severity"`
	MetabolismStatus     string                  `json:"metabolism_status"`
	ExposureChange       string                  `json:"exposure_change"`
	RiskLevel           string                  `json:"risk_level"`
	SpecificRecommendation string                `json:"specific_recommendation"`
}

// ValidatePGXData validates patient pharmacogenomic data format
func (pge *PharmacogenomicEngine) ValidatePGXData(patientPGX map[string]string) error {
	supportedGenes := []string{"CYP2D6", "CYP2C19", "CYP2C9", "SLCO1B1", "CYP3A5", "VKORC1"}
	validPhenotypes := map[string][]string{
		"CYP2D6":   {"poor", "intermediate", "normal", "ultrarapid"},
		"CYP2C19":  {"poor", "intermediate", "normal", "ultrarapid"},
		"CYP2C9":   {"poor", "intermediate", "normal"},
		"SLCO1B1":  {"poor", "normal"},
		"CYP3A5":   {"expresser", "non_expresser"},
		"VKORC1":   {"low", "intermediate", "high"},
	}

	for gene, phenotype := range patientPGX {
		// Check if gene is supported
		geneSupported := false
		for _, supportedGene := range supportedGenes {
			if gene == supportedGene {
				geneSupported = true
				break
			}
		}
		
		if !geneSupported {
			return fmt.Errorf("unsupported gene: %s", gene)
		}

		// Check if phenotype is valid for this gene
		if validPhenotypes, exists := validPhenotypes[gene]; exists {
			phenotypeValid := false
			for _, validPhenotype := range validPhenotypes {
				if strings.ToLower(phenotype) == validPhenotype {
					phenotypeValid = true
					break
				}
			}
			
			if !phenotypeValid {
				return fmt.Errorf("invalid phenotype %s for gene %s", phenotype, gene)
			}
		}
	}

	return nil
}