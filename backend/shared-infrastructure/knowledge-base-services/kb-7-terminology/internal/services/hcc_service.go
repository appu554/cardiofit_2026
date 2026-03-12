package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"

	"github.com/sirupsen/logrus"
)

// HCCService provides HCC mapping and RAF calculation services
type HCCService struct {
	graphDB       *semantic.GraphDBClient
	logger        *logrus.Logger
	hierarchies   map[string][]string // clinical_area -> ordered HCC codes
	coefficients  map[string]float64  // HCC code -> coefficient
	interactions  []models.HCCInteraction
	version       string
}

// NewHCCService creates a new HCC service instance
func NewHCCService(graphDB *semantic.GraphDBClient, logger *logrus.Logger) *HCCService {
	svc := &HCCService{
		graphDB:      graphDB,
		logger:       logger,
		hierarchies:  make(map[string][]string),
		coefficients: make(map[string]float64),
		version:      "V24",
	}

	// Initialize standard hierarchies
	svc.initializeHierarchies()
	// Initialize standard coefficients
	svc.initializeCoefficients()
	// Initialize interactions
	svc.initializeInteractions()

	return svc
}

// initializeHierarchies sets up the standard CMS HCC hierarchies
func (s *HCCService) initializeHierarchies() {
	for _, rule := range models.StandardHierarchies() {
		s.hierarchies[rule.ClinicalArea] = rule.Hierarchy
	}
}

// initializeCoefficients sets up V24 model coefficients (CMS-HCC Model)
func (s *HCCService) initializeCoefficients() {
	// V24 Community Non-Dual Aged (CNA) coefficients - 2024 values
	// These are sample values - production would load from database/GraphDB
	s.coefficients = map[string]float64{
		// Diabetes
		"HCC17": 0.302, // Diabetes with Acute Complications
		"HCC18": 0.302, // Diabetes with Chronic Complications
		"HCC19": 0.104, // Diabetes without Complication

		// Chronic Kidney Disease
		"HCC136": 0.291, // Chronic Kidney Disease, Stage 5
		"HCC137": 0.291, // Chronic Kidney Disease, Severe (Stage 4)
		"HCC138": 0.069, // Chronic Kidney Disease, Moderate (Stage 3)

		// Congestive Heart Failure
		"HCC85": 0.331, // Congestive Heart Failure
		"HCC86": 0.186, // Acute Myocardial Infarction

		// Depression
		"HCC59": 0.309, // Major Depressive, Bipolar Disorders
		"HCC60": 0.000, // Depression (no additional payment in V24)

		// COPD
		"HCC111": 0.335, // Chronic Obstructive Pulmonary Disease
		"HCC112": 0.205, // Fibrosis of Lung and Other Chronic Lung Disorders

		// Vascular Disease
		"HCC107": 0.288, // Vascular Disease with Complications
		"HCC108": 0.288, // Vascular Disease

		// Stroke
		"HCC99":  0.288, // Cerebral Hemorrhage
		"HCC100": 0.288, // Ischemic or Unspecified Stroke

		// Cancer
		"HCC8":  2.477, // Metastatic Cancer and Acute Leukemia
		"HCC9":  0.973, // Lung and Other Severe Cancers
		"HCC10": 0.679, // Lymphoma and Other Cancers
		"HCC11": 0.288, // Colorectal, Bladder, and Other Cancers
		"HCC12": 0.146, // Breast, Prostate, and Other Cancers

		// Seizure Disorders
		"HCC79": 0.220, // Seizure Disorders and Convulsions
		"HCC80": 0.220, // Coma, Brain Compression/Anoxic Damage

		// Schizophrenia
		"HCC57": 0.544, // Schizophrenia
		"HCC58": 0.309, // Reactive and Unspecified Psychosis

		// Substance Use
		"HCC54": 0.383, // Drug/Alcohol Psychosis
		"HCC55": 0.383, // Drug/Alcohol Dependence
		"HCC56": 0.000, // Drug/Alcohol Abuse

		// Additional high-impact HCCs
		"HCC2":   0.455, // Septicemia, Sepsis, Systemic Inflammatory Response Syndrome
		"HCC6":   0.288, // Opportunistic Infections
		"HCC47":  0.188, // Immunity Disorders
		"HCC48":  0.455, // Coagulation Defects
		"HCC96":  1.024, // Spinal Cord Disorders
		"HCC161": 0.455, // Chronic Ulcer of Skin
		"HCC162": 0.326, // Pressure Ulcer of Skin
		"HCC176": 0.417, // Artificial Openings for Feeding or Elimination
		"HCC177": 0.311, // Amputation Status
	}
}

// initializeInteractions sets up disease interaction factors
func (s *HCCService) initializeInteractions() {
	s.interactions = []models.HCCInteraction{
		{
			InteractionName: "HCC47_gCancer",
			RequiredHCCs:    []string{"HCC47", "HCC8"},
			Coefficient:     0.000, // Example - varies by model
			Description:     "Immunity Disorders + Cancer interaction",
		},
		{
			InteractionName: "DIABETES_CHF",
			RequiredHCCs:    []string{"HCC17", "HCC85"},
			Coefficient:     0.121,
			Description:     "Diabetes with complications + CHF interaction",
		},
		{
			InteractionName: "CHF_COPD",
			RequiredHCCs:    []string{"HCC85", "HCC111"},
			Coefficient:     0.175,
			Description:     "CHF + COPD interaction",
		},
		{
			InteractionName: "CHF_RENAL",
			RequiredHCCs:    []string{"HCC85", "HCC136"},
			Coefficient:     0.175,
			Description:     "CHF + CKD Stage 5 interaction",
		},
		{
			InteractionName: "COPD_CARD_RESP_FAIL",
			RequiredHCCs:    []string{"HCC111", "HCC82"},
			Coefficient:     0.195,
			Description:     "COPD + Cardio-Respiratory Failure interaction",
		},
	}
}

// MapICD10ToHCC maps a single ICD-10-CM code to HCC categories
func (s *HCCService) MapICD10ToHCC(ctx context.Context, icd10Code string) (*models.HCCLookupResult, error) {
	result := &models.HCCLookupResult{
		DiagnosisCode: icd10Code,
		HCCMappings:   []models.HCCMapping{},
		Valid:         false,
	}

	// Normalize the code
	normalizedCode := strings.ToUpper(strings.ReplaceAll(icd10Code, ".", ""))

	// Query GraphDB for HCC mappings
	if s.graphDB != nil {
		mappings, err := s.queryHCCMappings(ctx, normalizedCode)
		if err != nil {
			s.logger.WithError(err).WithField("icd10_code", icd10Code).Warn("GraphDB HCC lookup failed, using fallback")
		} else if len(mappings) > 0 {
			result.HCCMappings = mappings
			result.Valid = true
			return result, nil
		}
	}

	// Fallback to local lookup table
	mappings := s.localHCCLookup(normalizedCode)
	if len(mappings) > 0 {
		result.HCCMappings = mappings
		result.Valid = true
	} else {
		result.Message = fmt.Sprintf("No HCC mapping found for ICD-10 code: %s", icd10Code)
	}

	return result, nil
}

// queryHCCMappings queries GraphDB for HCC mappings
func (s *HCCService) queryHCCMappings(ctx context.Context, icd10Code string) ([]models.HCCMapping, error) {
	query := &semantic.SPARQLQuery{
		Query: fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

			SELECT ?hccCode ?description ?coefficient ?clinicalArea ?version WHERE {
				?mapping a kb7:ICD10ToHCCMapping ;
					kb7:icd10Code "%s" ;
					kb7:hccCode ?hccCode ;
					kb7:version ?version .
				OPTIONAL { ?mapping rdfs:label ?description }
				OPTIONAL { ?mapping kb7:coefficient ?coefficient }
				OPTIONAL { ?mapping kb7:clinicalArea ?clinicalArea }
			}
		`, icd10Code),
	}

	results, err := s.graphDB.ExecuteSPARQL(ctx, query)
	if err != nil {
		return nil, err
	}

	mappings := make([]models.HCCMapping, 0)
	for _, binding := range results.Results.Bindings {
		mapping := models.HCCMapping{
			HCCCode: binding["hccCode"].Value,
			Version: binding["version"].Value,
		}

		if desc, ok := binding["description"]; ok {
			mapping.Description = desc.Value
		}
		if coef, ok := binding["coefficient"]; ok {
			// Parse coefficient - simplified
			fmt.Sscanf(coef.Value, "%f", &mapping.Coefficient)
		}
		if area, ok := binding["clinicalArea"]; ok {
			mapping.ClinicalArea = area.Value
		}

		mappings = append(mappings, mapping)
	}

	return mappings, nil
}

// localHCCLookup provides local fallback HCC lookup
func (s *HCCService) localHCCLookup(icd10Code string) []models.HCCMapping {
	// Sample ICD-10 to HCC mappings - production would use database
	icd10ToHCC := map[string][]models.HCCMapping{
		// Diabetes mappings
		"E1121": {{HCCCode: "HCC18", Description: "Diabetes with Chronic Complications", ClinicalArea: "diabetes", Version: "V24"}},
		"E1122": {{HCCCode: "HCC18", Description: "Diabetes with Chronic Complications", ClinicalArea: "diabetes", Version: "V24"}},
		"E1165": {{HCCCode: "HCC17", Description: "Diabetes with Acute Complications", ClinicalArea: "diabetes", Version: "V24"}},
		"E119":  {{HCCCode: "HCC19", Description: "Diabetes without Complication", ClinicalArea: "diabetes", Version: "V24"}},

		// CKD mappings
		"N185":  {{HCCCode: "HCC136", Description: "Chronic Kidney Disease, Stage 5", ClinicalArea: "ckd", Version: "V24"}},
		"N184":  {{HCCCode: "HCC137", Description: "Chronic Kidney Disease, Stage 4", ClinicalArea: "ckd", Version: "V24"}},
		"N183":  {{HCCCode: "HCC138", Description: "Chronic Kidney Disease, Stage 3", ClinicalArea: "ckd", Version: "V24"}},

		// CHF mappings
		"I5020": {{HCCCode: "HCC85", Description: "Systolic Heart Failure, Unspecified", ClinicalArea: "chf", Version: "V24"}},
		"I5021": {{HCCCode: "HCC85", Description: "Acute Systolic Heart Failure", ClinicalArea: "chf", Version: "V24"}},
		"I5022": {{HCCCode: "HCC85", Description: "Chronic Systolic Heart Failure", ClinicalArea: "chf", Version: "V24"}},
		"I5023": {{HCCCode: "HCC85", Description: "Acute on Chronic Systolic Heart Failure", ClinicalArea: "chf", Version: "V24"}},

		// COPD mappings
		"J449":  {{HCCCode: "HCC111", Description: "COPD, Unspecified", ClinicalArea: "copd", Version: "V24"}},
		"J441":  {{HCCCode: "HCC111", Description: "COPD with Acute Exacerbation", ClinicalArea: "copd", Version: "V24"}},

		// Depression mappings
		"F322":  {{HCCCode: "HCC59", Description: "Major Depressive Disorder, Severe", ClinicalArea: "depression", Version: "V24"}},
		"F329":  {{HCCCode: "HCC60", Description: "Major Depressive Disorder, Unspecified", ClinicalArea: "depression", Version: "V24"}},

		// Cancer mappings
		"C7800": {{HCCCode: "HCC8", Description: "Metastatic Cancer to Lung", ClinicalArea: "cancer", Version: "V24"}},
		"C349":  {{HCCCode: "HCC9", Description: "Lung Cancer, Unspecified", ClinicalArea: "cancer", Version: "V24"}},
	}

	if mappings, ok := icd10ToHCC[icd10Code]; ok {
		// Add coefficients
		for i := range mappings {
			if coef, exists := s.coefficients[mappings[i].HCCCode]; exists {
				mappings[i].Coefficient = coef
			}
		}
		return mappings
	}

	return nil
}

// CalculateRAF calculates the Risk Adjustment Factor for a patient
func (s *HCCService) CalculateRAF(ctx context.Context, req *models.RAFCalculationRequest) (*models.RAFCalculationResult, error) {
	result := &models.RAFCalculationResult{
		PatientID:     req.PatientID,
		HCCCategories: []models.HCCResult{},
		ModelType:     req.ModelType,
		ModelVersion:  s.version,
		PaymentYear:   req.PaymentYear,
		CalculatedAt:  time.Now(),
	}

	if result.ModelType == "" {
		result.ModelType = "CNA" // Community Non-Dual Aged
	}

	// Step 1: Map all diagnosis codes to HCCs
	hccMap := make(map[string]*models.HCCResult) // HCC code -> result
	for _, diagCode := range req.DiagnosisCodes {
		lookup, err := s.MapICD10ToHCC(ctx, diagCode)
		if err != nil {
			s.logger.WithError(err).WithField("diagnosis_code", diagCode).Warn("Failed to map diagnosis code")
			continue
		}

		for _, mapping := range lookup.HCCMappings {
			if existing, ok := hccMap[mapping.HCCCode]; ok {
				// Add source code to existing HCC
				existing.SourceCodes = append(existing.SourceCodes, diagCode)
			} else {
				hccMap[mapping.HCCCode] = &models.HCCResult{
					HCCCode:      mapping.HCCCode,
					Description:  mapping.Description,
					Coefficient:  mapping.Coefficient,
					SourceCodes:  []string{diagCode},
					ClinicalArea: mapping.ClinicalArea,
					Applied:      true,
				}
			}
		}
	}

	// Step 2: Apply hierarchy rules (trump lower severity HCCs)
	droppedHCCs := s.applyHierarchyRules(hccMap)
	result.DroppedHCCs = droppedHCCs

	// Step 3: Calculate demographic RAF
	result.Demographics = s.calculateDemographicFactors(&req.Demographics)
	result.DemographicRAF = result.Demographics.AgeGenderCoefficient +
		result.Demographics.DisabilityCoefficient +
		result.Demographics.MedicaidCoefficient +
		result.Demographics.InstitutionalCoefficient

	// Step 4: Sum disease HCC coefficients
	diseaseRAF := 0.0
	for _, hcc := range hccMap {
		if hcc.Applied {
			diseaseRAF += hcc.Coefficient
			result.HCCCategories = append(result.HCCCategories, *hcc)
		}
	}
	result.DiseaseRAF = diseaseRAF

	// Step 5: Calculate interaction coefficients
	interactionRAF, interactions := s.calculateInteractions(hccMap)
	result.InteractionRAF = interactionRAF
	result.Interactions = interactions

	// Step 6: Calculate total RAF
	result.TotalRAF = result.DemographicRAF + result.DiseaseRAF + result.InteractionRAF

	// Sort HCC categories by coefficient (highest first)
	sort.Slice(result.HCCCategories, func(i, j int) bool {
		return result.HCCCategories[i].Coefficient > result.HCCCategories[j].Coefficient
	})

	return result, nil
}

// applyHierarchyRules applies HCC hierarchy rules and returns dropped HCCs
func (s *HCCService) applyHierarchyRules(hccMap map[string]*models.HCCResult) []models.DroppedHCC {
	dropped := []models.DroppedHCC{}

	for _, hierarchy := range s.hierarchies {
		// Find the highest severity HCC present in this hierarchy
		var highestPresent string
		for _, hcc := range hierarchy {
			if _, exists := hccMap[hcc]; exists {
				highestPresent = hcc
				break // First one found is highest
			}
		}

		if highestPresent == "" {
			continue // No HCCs from this hierarchy
		}

		// Drop all lower severity HCCs in this hierarchy
		foundHighest := false
		for _, hcc := range hierarchy {
			if hcc == highestPresent {
				foundHighest = true
				continue
			}
			if foundHighest {
				if result, exists := hccMap[hcc]; exists {
					result.Applied = false
					dropped = append(dropped, models.DroppedHCC{
						DroppedCode:   hcc,
						TrumpedByCode: highestPresent,
						Reason:        "Lower severity in hierarchy",
					})
				}
			}
		}
	}

	return dropped
}

// calculateDemographicFactors calculates demographic-based RAF components
func (s *HCCService) calculateDemographicFactors(demo *models.PatientDemographics) models.DemographicFactors {
	factors := models.DemographicFactors{
		Gender: demo.Gender,
	}

	// Age-gender coefficient (simplified - production uses full CMS tables)
	ageGroup := s.getAgeGroup(demo.Age)
	factors.AgeGroup = ageGroup

	// Sample age-gender coefficients for Community Non-Dual Aged
	ageGenderCoeffs := map[string]map[string]float64{
		"M": {
			"65-69": 0.269,
			"70-74": 0.338,
			"75-79": 0.421,
			"80-84": 0.535,
			"85-89": 0.672,
			"90-94": 0.787,
			"95+":   0.785,
		},
		"F": {
			"65-69": 0.235,
			"70-74": 0.285,
			"75-79": 0.359,
			"80-84": 0.461,
			"85-89": 0.605,
			"90-94": 0.764,
			"95+":   0.818,
		},
	}

	if genderCoeffs, ok := ageGenderCoeffs[demo.Gender]; ok {
		if coeff, ok := genderCoeffs[ageGroup]; ok {
			factors.AgeGenderCoefficient = coeff
		}
	}

	// Disability status adjustment
	if demo.OriginallyDisabled {
		factors.DisabilityCoefficient = 0.000 // Varies by model
	}

	// Medicaid status adjustment
	if demo.Medicaid {
		factors.MedicaidCoefficient = 0.048
	}

	// Institutional status
	if demo.InstitutionStatus == "institutional" {
		factors.InstitutionalCoefficient = 0.000 // Uses different model
	}

	return factors
}

// getAgeGroup returns the age group string for coefficient lookup
func (s *HCCService) getAgeGroup(age int) string {
	switch {
	case age < 65:
		return "<65"
	case age <= 69:
		return "65-69"
	case age <= 74:
		return "70-74"
	case age <= 79:
		return "75-79"
	case age <= 84:
		return "80-84"
	case age <= 89:
		return "85-89"
	case age <= 94:
		return "90-94"
	default:
		return "95+"
	}
}

// calculateInteractions calculates disease interaction coefficients
func (s *HCCService) calculateInteractions(hccMap map[string]*models.HCCResult) (float64, []models.InteractionResult) {
	totalInteractionRAF := 0.0
	interactions := []models.InteractionResult{}

	for _, interaction := range s.interactions {
		// Check if all required HCCs are present and applied
		allPresent := true
		for _, reqHCC := range interaction.RequiredHCCs {
			if hcc, exists := hccMap[reqHCC]; !exists || !hcc.Applied {
				allPresent = false
				break
			}
		}

		if allPresent {
			totalInteractionRAF += interaction.Coefficient
			interactions = append(interactions, models.InteractionResult{
				InteractionName: interaction.InteractionName,
				HCCCodes:        interaction.RequiredHCCs,
				Coefficient:     interaction.Coefficient,
				Description:     interaction.Description,
			})
		}
	}

	return totalInteractionRAF, interactions
}

// BatchCalculateRAF calculates RAF for multiple patients
func (s *HCCService) BatchCalculateRAF(ctx context.Context, req *models.HCCBatchRequest) (*models.HCCBatchResult, error) {
	result := &models.HCCBatchResult{
		Results:    make([]models.RAFCalculationResult, 0, len(req.Patients)),
		TotalCount: len(req.Patients),
		Errors:     []models.HCCError{},
	}

	for _, patientReq := range req.Patients {
		rafResult, err := s.CalculateRAF(ctx, &patientReq)
		if err != nil {
			result.ErrorCount++
			result.Errors = append(result.Errors, models.HCCError{
				PatientID: patientReq.PatientID,
				Error:     err.Error(),
			})
			continue
		}

		result.Results = append(result.Results, *rafResult)
		result.SuccessCount++
	}

	return result, nil
}

// GetHCCHierarchies returns all configured HCC hierarchies
func (s *HCCService) GetHCCHierarchies() []models.HCCHierarchyRule {
	return models.StandardHierarchies()
}

// GetHCCCoefficients returns all configured HCC coefficients
func (s *HCCService) GetHCCCoefficients() map[string]float64 {
	return s.coefficients
}
