package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/models"
)

// RiskAssessmentService provides enhanced risk calculation functionality
type RiskAssessmentService struct {
	config      *config.Config
	mongoClient *mongo.Client
	redisClient *redis.Client
	
	// Risk models and calculators
	riskModels map[string]RiskModel
}

// RiskModel interface for different risk calculation models
type RiskModel interface {
	CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error)
	GetModelName() string
	GetCategories() []string
}

// NewRiskAssessmentService creates a new risk assessment service
func NewRiskAssessmentService(mongoClient *mongo.Client, redisClient *redis.Client) *RiskAssessmentService {
	service := &RiskAssessmentService{
		mongoClient: mongoClient,
		redisClient: redisClient,
		riskModels:  make(map[string]RiskModel),
	}
	
	// Initialize default risk models
	service.initializeRiskModels()
	
	return service
}

// initializeRiskModels initializes default risk assessment models
func (ras *RiskAssessmentService) initializeRiskModels() {
	// Cardiovascular risk model
	ras.riskModels["cardiovascular"] = &CardiovascularRiskModel{}
	
	// Diabetes risk model
	ras.riskModels["diabetes"] = &DiabetesRiskModel{}
	
	// Medication interaction risk model
	ras.riskModels["medication"] = &MedicationRiskModel{}
	
	// Fall risk model (for elderly patients)
	ras.riskModels["fall"] = &FallRiskModel{}
	
	// Bleeding risk model
	ras.riskModels["bleeding"] = &BleedingRiskModel{}
}

// AssessRisk performs comprehensive risk assessment for a patient
func (ras *RiskAssessmentService) AssessRisk(ctx context.Context, request *models.RiskAssessmentRequest) (*models.RiskAssessmentResult, error) {
	startTime := time.Now()
	
	result := &models.RiskAssessmentResult{
		PatientID:     request.PatientID,
		CategoryRisks: make(map[string]models.RiskScore),
		RiskFactors:   []models.RiskFactor{},
		GeneratedAt:   time.Now(),
	}
	
	// Determine which risk categories to assess
	categoriesToAssess := ras.determineRiskCategories(request)
	
	var overallRiskScore float64
	var allRiskFactors []models.RiskFactor
	
	// Assess each risk category
	for _, category := range categoriesToAssess {
		if model, exists := ras.riskModels[category]; exists {
			categoryRisk, riskFactors, err := model.CalculateRisk(request.PatientData)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate %s risk: %w", category, err)
			}
			
			// Convert to risk score
			riskScore := ras.convertToRiskScore(categoryRisk, category)
			result.CategoryRisks[category] = riskScore
			
			// Collect all risk factors
			allRiskFactors = append(allRiskFactors, riskFactors...)
			
			// Update overall risk (weighted average)
			weight := ras.getCategoryWeight(category)
			overallRiskScore += categoryRisk * weight
		}
	}
	
	// Calculate overall risk
	if len(categoriesToAssess) > 0 {
		// Normalize the overall risk score
		totalWeight := ras.getTotalWeight(categoriesToAssess)
		overallRiskScore = overallRiskScore / totalWeight
	}
	
	// Convert overall risk to risk score
	result.OverallRisk = ras.convertToRiskScore(overallRiskScore, "overall")
	
	// Sort and limit risk factors
	result.RiskFactors = ras.prioritizeRiskFactors(allRiskFactors, 10)
	
	// Generate recommendations
	result.Recommendations = ras.generateRecommendations(result, request.PatientData)
	
	result.ProcessingTime = time.Since(startTime)
	
	// Check SLA compliance
	slaThreshold := time.Duration(200) * time.Millisecond // 200ms SLA
	if result.ProcessingTime > slaThreshold {
		return result, fmt.Errorf("SLA violation: risk assessment took %v, threshold is %v", result.ProcessingTime, slaThreshold)
	}
	
	return result, nil
}

// determineRiskCategories determines which risk categories to assess
func (ras *RiskAssessmentService) determineRiskCategories(request *models.RiskAssessmentRequest) []string {
	if len(request.RiskCategories) > 0 {
		return request.RiskCategories
	}
	
	// Auto-determine based on patient data
	categories := []string{"cardiovascular"} // Always assess cardiovascular risk
	
	// Add diabetes risk if relevant conditions
	for _, condition := range request.PatientData.Conditions {
		switch condition {
		case "diabetes", "prediabetes", "metabolic_syndrome":
			if !contains(categories, "diabetes") {
				categories = append(categories, "diabetes")
			}
		case "hypertension", "coronary_artery_disease", "heart_failure":
			// Cardiovascular already included
		}
	}
	
	// Add medication risk if patient has multiple medications
	if len(request.PatientData.Medications) >= 5 {
		categories = append(categories, "medication")
	}
	
	// Add fall risk for elderly patients
	if request.PatientData.Age >= 65 {
		categories = append(categories, "fall")
	}
	
	// Add bleeding risk for anticoagulation patients
	for _, medication := range request.PatientData.Medications {
		if ras.isAnticoagulant(medication) {
			categories = append(categories, "bleeding")
			break
		}
	}
	
	return categories
}

// convertToRiskScore converts a numerical risk to a structured risk score
func (ras *RiskAssessmentService) convertToRiskScore(risk float64, category string) models.RiskScore {
	var level string
	var percentile float64
	var description string
	
	// Categorize risk level
	if risk < 0.1 {
		level = "low"
		percentile = risk * 500 // Scale to 0-50th percentile
		description = "Low risk"
	} else if risk < 0.3 {
		level = "moderate"
		percentile = 50 + (risk-0.1)*250 // Scale to 50-100th percentile
		description = "Moderate risk"
	} else if risk < 0.7 {
		level = "high"
		percentile = 75 + (risk-0.3)*62.5 // Scale to 75-100th percentile
		description = "High risk"
	} else {
		level = "very_high"
		percentile = 90 + (risk-0.7)*33.3 // Scale to 90-100th percentile
		description = "Very high risk"
	}
	
	return models.RiskScore{
		Score:       risk,
		Level:       level,
		Percentile:  math.Min(percentile, 100),
		Confidence:  ras.calculateConfidence(risk, category),
		Description: description,
	}
}

// calculateConfidence calculates confidence in the risk score
func (ras *RiskAssessmentService) calculateConfidence(risk float64, category string) float64 {
	// Base confidence is 0.7
	confidence := 0.7
	
	// Increase confidence for moderate risks (more certain)
	if risk >= 0.1 && risk <= 0.5 {
		confidence += 0.2
	}
	
	// Decrease confidence for extreme risks (more uncertain)
	if risk < 0.05 || risk > 0.8 {
		confidence -= 0.1
	}
	
	return math.Max(0.5, math.Min(1.0, confidence))
}

// getCategoryWeight returns the weight for a risk category in overall calculation
func (ras *RiskAssessmentService) getCategoryWeight(category string) float64 {
	weights := map[string]float64{
		"cardiovascular": 0.4,
		"diabetes":       0.2,
		"medication":     0.2,
		"fall":           0.1,
		"bleeding":       0.1,
	}
	
	if weight, exists := weights[category]; exists {
		return weight
	}
	return 0.1 // Default weight
}

// getTotalWeight calculates total weight for normalization
func (ras *RiskAssessmentService) getTotalWeight(categories []string) float64 {
	total := 0.0
	for _, category := range categories {
		total += ras.getCategoryWeight(category)
	}
	return total
}

// prioritizeRiskFactors sorts and limits risk factors by impact
func (ras *RiskAssessmentService) prioritizeRiskFactors(factors []models.RiskFactor, limit int) []models.RiskFactor {
	// Sort by impact (descending)
	for i := 0; i < len(factors)-1; i++ {
		for j := i + 1; j < len(factors); j++ {
			if factors[i].Impact < factors[j].Impact {
				factors[i], factors[j] = factors[j], factors[i]
			}
		}
	}
	
	// Limit results
	if len(factors) > limit {
		return factors[:limit]
	}
	return factors
}

// generateRecommendations generates risk-based recommendations
func (ras *RiskAssessmentService) generateRecommendations(result *models.RiskAssessmentResult, patient models.Patient) []models.Recommendation {
	recommendations := []models.Recommendation{}
	
	// High overall risk recommendations
	if result.OverallRisk.Level == "high" || result.OverallRisk.Level == "very_high" {
		recommendations = append(recommendations, models.Recommendation{
			Category:        "monitoring",
			Action:          "Increase monitoring frequency",
			Priority:        "high",
			Evidence:        "High overall risk profile",
			ExpectedBenefit: 0.8,
		})
	}
	
	// Category-specific recommendations
	for category, riskScore := range result.CategoryRisks {
		if riskScore.Level == "high" || riskScore.Level == "very_high" {
			rec := ras.getCategorySpecificRecommendation(category, riskScore)
			if rec != nil {
				recommendations = append(recommendations, *rec)
			}
		}
	}
	
	// Risk factor-specific recommendations
	for _, factor := range result.RiskFactors[:min(5, len(result.RiskFactors))] {
		if factor.Modifiable && factor.Impact > 0.1 {
			recommendations = append(recommendations, models.Recommendation{
				Category:        "lifestyle",
				Action:          fmt.Sprintf("Address modifiable risk factor: %s", factor.Factor),
				Priority:        ras.getPriorityForImpact(factor.Impact),
				Evidence:        factor.Evidence,
				ExpectedBenefit: factor.Impact * 0.7, // Assume 70% benefit from addressing factor
			})
		}
	}
	
	return recommendations
}

// getCategorySpecificRecommendation returns category-specific recommendations
func (ras *RiskAssessmentService) getCategorySpecificRecommendation(category string, riskScore models.RiskScore) *models.Recommendation {
	recommendations := map[string]models.Recommendation{
		"cardiovascular": {
			Category:        "prevention",
			Action:          "Consider cardioprotective therapy",
			Priority:        "high",
			Evidence:        "High cardiovascular risk",
			ExpectedBenefit: 0.6,
		},
		"diabetes": {
			Category:        "monitoring",
			Action:          "Enhance glucose monitoring",
			Priority:        "high",
			Evidence:        "High diabetes risk",
			ExpectedBenefit: 0.5,
		},
		"medication": {
			Category:        "safety",
			Action:          "Review medication interactions",
			Priority:        "high",
			Evidence:        "High medication risk",
			ExpectedBenefit: 0.7,
		},
		"fall": {
			Category:        "prevention",
			Action:          "Implement fall prevention measures",
			Priority:        "medium",
			Evidence:        "High fall risk",
			ExpectedBenefit: 0.4,
		},
		"bleeding": {
			Category:        "monitoring",
			Action:          "Monitor for bleeding signs",
			Priority:        "high",
			Evidence:        "High bleeding risk",
			ExpectedBenefit: 0.8,
		},
	}
	
	if rec, exists := recommendations[category]; exists {
		return &rec
	}
	return nil
}

// getPriorityForImpact returns priority level based on impact
func (ras *RiskAssessmentService) getPriorityForImpact(impact float64) string {
	if impact > 0.3 {
		return "high"
	} else if impact > 0.1 {
		return "medium"
	}
	return "low"
}

// isAnticoagulant checks if a medication is an anticoagulant
func (ras *RiskAssessmentService) isAnticoagulant(medication string) bool {
	anticoagulants := []string{"warfarin", "heparin", "enoxaparin", "rivaroxaban", "apixaban", "dabigatran"}
	for _, anticoag := range anticoagulants {
		if medication == anticoag {
			return true
		}
	}
	return false
}

// Utility function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Risk Model Implementations

// CardiovascularRiskModel implements cardiovascular risk assessment
type CardiovascularRiskModel struct{}

func (crm *CardiovascularRiskModel) CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error) {
	risk := 0.0
	factors := []models.RiskFactor{}
	
	// Age factor
	if patient.Age > 65 {
		ageRisk := math.Min(0.3, float64(patient.Age-65)*0.01)
		risk += ageRisk
		factors = append(factors, models.RiskFactor{
			Factor:     "Advanced age",
			Impact:     ageRisk,
			Evidence:   fmt.Sprintf("Age %d years", patient.Age),
			Modifiable: false,
		})
	}
	
	// Gender factor
	if patient.Gender == "male" && patient.Age > 45 {
		genderRisk := 0.1
		risk += genderRisk
		factors = append(factors, models.RiskFactor{
			Factor:     "Male gender over 45",
			Impact:     genderRisk,
			Evidence:   "Male gender with advanced age",
			Modifiable: false,
		})
	}
	
	// Condition factors
	for _, condition := range patient.Conditions {
		switch condition {
		case "hypertension":
			conditionRisk := 0.15
			risk += conditionRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Hypertension",
				Impact:     conditionRisk,
				Evidence:   "Diagnosed hypertension",
				Modifiable: true,
			})
		case "diabetes":
			conditionRisk := 0.2
			risk += conditionRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Diabetes mellitus",
				Impact:     conditionRisk,
				Evidence:   "Diagnosed diabetes",
				Modifiable: true,
			})
		case "hyperlipidemia":
			conditionRisk := 0.1
			risk += conditionRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Hyperlipidemia",
				Impact:     conditionRisk,
				Evidence:   "Diagnosed hyperlipidemia",
				Modifiable: true,
			})
		}
	}
	
	// Lab factors
	if cholesterol, exists := patient.Labs["total_cholesterol"]; exists {
		if cholesterol.Value > 240 {
			cholRisk := 0.1
			risk += cholRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Elevated cholesterol",
				Impact:     cholRisk,
				Evidence:   fmt.Sprintf("Total cholesterol: %.1f mg/dL", cholesterol.Value),
				Modifiable: true,
			})
		}
	}
	
	return math.Min(risk, 1.0), factors, nil
}

func (crm *CardiovascularRiskModel) GetModelName() string {
	return "cardiovascular"
}

func (crm *CardiovascularRiskModel) GetCategories() []string {
	return []string{"cardiovascular"}
}

// DiabetesRiskModel implements diabetes risk assessment
type DiabetesRiskModel struct{}

func (drm *DiabetesRiskModel) CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error) {
	risk := 0.0
	factors := []models.RiskFactor{}
	
	// Existing diabetes
	for _, condition := range patient.Conditions {
		if condition == "diabetes" {
			risk += 0.8 // Very high risk for complications
			factors = append(factors, models.RiskFactor{
				Factor:     "Existing diabetes",
				Impact:     0.8,
				Evidence:   "Diagnosed diabetes mellitus",
				Modifiable: true,
			})
			break
		}
	}
	
	// HbA1c factor
	if hba1c, exists := patient.Labs["hba1c"]; exists {
		if hba1c.Value > 7.0 {
			hba1cRisk := math.Min(0.4, (hba1c.Value-7.0)*0.1)
			risk += hba1cRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Elevated HbA1c",
				Impact:     hba1cRisk,
				Evidence:   fmt.Sprintf("HbA1c: %.1f%%", hba1c.Value),
				Modifiable: true,
			})
		}
	}
	
	// BMI factor (if available in metadata)
	if bmiInterface, exists := patient.Metadata["bmi"]; exists {
		if bmi, ok := bmiInterface.(float64); ok && bmi > 30 {
			bmiRisk := math.Min(0.2, (bmi-30)*0.02)
			risk += bmiRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Obesity",
				Impact:     bmiRisk,
				Evidence:   fmt.Sprintf("BMI: %.1f", bmi),
				Modifiable: true,
			})
		}
	}
	
	return math.Min(risk, 1.0), factors, nil
}

func (drm *DiabetesRiskModel) GetModelName() string {
	return "diabetes"
}

func (drm *DiabetesRiskModel) GetCategories() []string {
	return []string{"diabetes"}
}

// MedicationRiskModel implements medication interaction risk assessment
type MedicationRiskModel struct{}

func (mrm *MedicationRiskModel) CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error) {
	risk := 0.0
	factors := []models.RiskFactor{}
	
	// Polypharmacy risk
	medicationCount := len(patient.Medications)
	if medicationCount >= 5 {
		polyRisk := math.Min(0.3, float64(medicationCount-5)*0.03)
		risk += polyRisk
		factors = append(factors, models.RiskFactor{
			Factor:     "Polypharmacy",
			Impact:     polyRisk,
			Evidence:   fmt.Sprintf("%d concurrent medications", medicationCount),
			Modifiable: true,
		})
	}
	
	// High-risk medication combinations
	riskyCombos := mrm.identifyRiskyCombinations(patient.Medications)
	for _, combo := range riskyCombos {
		risk += combo.Risk
		factors = append(factors, models.RiskFactor{
			Factor:     combo.Description,
			Impact:     combo.Risk,
			Evidence:   fmt.Sprintf("Medications: %s", combo.Evidence),
			Modifiable: true,
		})
	}
	
	return math.Min(risk, 1.0), factors, nil
}

func (mrm *MedicationRiskModel) identifyRiskyCombinations(medications []string) []struct {
	Description string
	Risk        float64
	Evidence    string
} {
	combinations := []struct {
		Description string
		Risk        float64
		Evidence    string
	}{}
	
	// Check for known risky combinations
	hasWarfarin := contains(medications, "warfarin")
	hasAspirin := contains(medications, "aspirin")
	
	if hasWarfarin && hasAspirin {
		combinations = append(combinations, struct {
			Description string
			Risk        float64
			Evidence    string
		}{
			Description: "Anticoagulant + antiplatelet",
			Risk:        0.25,
			Evidence:    "warfarin + aspirin",
		})
	}
	
	return combinations
}

func (mrm *MedicationRiskModel) GetModelName() string {
	return "medication"
}

func (mrm *MedicationRiskModel) GetCategories() []string {
	return []string{"medication"}
}

// FallRiskModel implements fall risk assessment
type FallRiskModel struct{}

func (frm *FallRiskModel) CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error) {
	risk := 0.0
	factors := []models.RiskFactor{}
	
	// Age factor
	if patient.Age >= 65 {
		ageRisk := math.Min(0.2, float64(patient.Age-65)*0.01)
		risk += ageRisk
		factors = append(factors, models.RiskFactor{
			Factor:     "Advanced age",
			Impact:     ageRisk,
			Evidence:   fmt.Sprintf("Age %d years", patient.Age),
			Modifiable: false,
		})
	}
	
	// Medication factors
	fallRiskMeds := []string{"sedatives", "antipsychotics", "opioids", "benzodiazepines"}
	for _, medication := range patient.Medications {
		if contains(fallRiskMeds, medication) {
			medRisk := 0.15
			risk += medRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Fall-risk medication",
				Impact:     medRisk,
				Evidence:   fmt.Sprintf("Taking %s", medication),
				Modifiable: true,
			})
		}
	}
	
	return math.Min(risk, 1.0), factors, nil
}

func (frm *FallRiskModel) GetModelName() string {
	return "fall"
}

func (frm *FallRiskModel) GetCategories() []string {
	return []string{"fall"}
}

// BleedingRiskModel implements bleeding risk assessment
type BleedingRiskModel struct{}

func (brm *BleedingRiskModel) CalculateRisk(patient models.Patient) (float64, []models.RiskFactor, error) {
	risk := 0.0
	factors := []models.RiskFactor{}
	
	// Anticoagulant factor
	anticoagulants := []string{"warfarin", "heparin", "rivaroxaban", "apixaban"}
	for _, medication := range patient.Medications {
		if contains(anticoagulants, medication) {
			anticoagRisk := 0.3
			risk += anticoagRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Anticoagulant therapy",
				Impact:     anticoagRisk,
				Evidence:   fmt.Sprintf("Taking %s", medication),
				Modifiable: true,
			})
			break
		}
	}
	
	// Age factor
	if patient.Age >= 75 {
		ageRisk := 0.15
		risk += ageRisk
		factors = append(factors, models.RiskFactor{
			Factor:     "Advanced age",
			Impact:     ageRisk,
			Evidence:   fmt.Sprintf("Age %d years", patient.Age),
			Modifiable: false,
		})
	}
	
	// History of bleeding
	for _, condition := range patient.Conditions {
		if condition == "bleeding_history" || condition == "peptic_ulcer" {
			bleedingRisk := 0.2
			risk += bleedingRisk
			factors = append(factors, models.RiskFactor{
				Factor:     "Bleeding history",
				Impact:     bleedingRisk,
				Evidence:   fmt.Sprintf("History of %s", condition),
				Modifiable: false,
			})
		}
	}
	
	return math.Min(risk, 1.0), factors, nil
}

func (brm *BleedingRiskModel) GetModelName() string {
	return "bleeding"
}

func (brm *BleedingRiskModel) GetCategories() []string {
	return []string{"bleeding"}
}