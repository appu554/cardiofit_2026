package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"kb-formulary/internal/services"
)

// FormularyHandler handles HTTP requests for formulary operations
type FormularyHandler struct {
	formularyService *services.FormularyService
}

// NewFormularyHandler creates a new FormularyHandler
func NewFormularyHandler(formularyService *services.FormularyService) *FormularyHandler {
	return &FormularyHandler{
		formularyService: formularyService,
	}
}

// GetCoverage handles GET /api/v1/formulary/coverage requests
func (h *FormularyHandler) GetCoverage(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	drugID := r.URL.Query().Get("drug_id")
	payerID := r.URL.Query().Get("payer_id")
	memberID := r.URL.Query().Get("member_id")
	formularyID := r.URL.Query().Get("formulary_id")

	// Validate required parameters
	if drugID == "" || payerID == "" {
		http.Error(w, "Missing required parameters: drug_id and payer_id", http.StatusBadRequest)
		return
	}

	// Create coverage request
	request := services.HTTPCoverageRequest{
		DrugID:      drugID,
		PayerID:     payerID,
		MemberID:    memberID,
		FormularyID: formularyID,
		RequestID:   generateRequestID(),
	}

	// Get coverage information
	response, err := h.formularyService.GetCoverage(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get coverage information: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=900") // 15 minutes cache

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetAlternatives handles GET /api/v1/formulary/alternatives requests
func (h *FormularyHandler) GetAlternatives(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	drugID := r.URL.Query().Get("drug_id")
	payerID := r.URL.Query().Get("payer_id")
	therapeuticClass := r.URL.Query().Get("therapeutic_class")
	maxResultsStr := r.URL.Query().Get("max_results")

	// Validate required parameters
	if drugID == "" || payerID == "" {
		http.Error(w, "Missing required parameters: drug_id and payer_id", http.StatusBadRequest)
		return
	}

	// Parse optional parameters
	maxResults := 10 // default
	if maxResultsStr != "" {
		if parsed, err := strconv.Atoi(maxResultsStr); err == nil && parsed > 0 {
			maxResults = parsed
		}
	}

	// Create alternatives request
	request := services.AlternativesRequest{
		DrugID:           drugID,
		PayerID:          payerID,
		TherapeuticClass: therapeuticClass,
		MaxResults:       maxResults,
		RequestID:        generateRequestID(),
	}

	// Get alternatives information
	response, err := h.formularyService.GetAlternatives(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get alternatives: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=900") // 15 minutes cache

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// SearchDrugs handles GET /api/v1/formulary/search requests
func (h *FormularyHandler) SearchDrugs(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	query := r.URL.Query().Get("q")
	payerID := r.URL.Query().Get("payer_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Validate required parameters
	if query == "" {
		http.Error(w, "Missing required parameter: q (search query)", http.StatusBadRequest)
		return
	}

	// Parse pagination parameters
	limit := 20 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Create search request
	request := services.HTTPSearchRequest{
		Query:     query,
		PayerID:   payerID,
		Limit:     limit,
		Offset:    offset,
		RequestID: generateRequestID(),
	}

	// Perform search
	response, err := h.formularyService.SearchDrugs(ctx, request)
	if err != nil {
		http.Error(w, "Failed to search drugs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=300") // 5 minutes cache for search results

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetFormularyInfo handles GET /api/v1/formulary/info requests
func (h *FormularyHandler) GetFormularyInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse path parameter (formulary ID)
	formularyID := r.URL.Path[len("/api/v1/formulary/info/"):]
	if formularyID == "" {
		http.Error(w, "Missing formulary ID in path", http.StatusBadRequest)
		return
	}

	// Get formulary information
	formulary, err := h.formularyService.GetFormularyInfo(ctx, formularyID)
	if err != nil {
		if err.Error() == "formulary not found" {
			http.Error(w, "Formulary not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get formulary info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=3600") // 1 hour cache for formulary info

	// Return response
	if err := json.NewEncoder(w).Encode(formulary); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HealthCheck handles GET /health requests for formulary service
func (h *FormularyHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Perform health check
	health := h.formularyService.HealthCheck(ctx)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Set status code based on health
	if health.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Return health status
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
		return
	}
}

// generateRequestID creates a unique request ID for tracing
func generateRequestID() string {
	return "kb6-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Response wrapper for consistent API responses
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	RequestID string      `json:"request_id"`
	Timestamp time.Time   `json:"timestamp"`
}

// AnalyzeCosts handles intelligent cost analysis requests
func (fh *FormularyHandler) AnalyzeCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := generateRequestID()
	
	// Parse request body
	var httpReq struct {
		TransactionID        string   `json:"transaction_id"`
		DrugRxNorms         []string `json:"drug_rxnorms"`
		PayerID             string   `json:"payer_id"`
		PlanID              string   `json:"plan_id"`
		Quantity            int      `json:"quantity"`
		IncludeAlternatives bool     `json:"include_alternatives"`
		OptimizationGoal    string   `json:"optimization_goal"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
		sendAPIResponse(w, false, nil, "Invalid request body", requestID)
		return
	}

	// Validate required fields
	if len(httpReq.DrugRxNorms) == 0 || httpReq.PayerID == "" || httpReq.PlanID == "" {
		sendAPIResponse(w, false, nil, "Missing required fields: drug_rxnorms, payer_id, plan_id", requestID)
		return
	}

	// Set defaults
	if httpReq.Quantity <= 0 {
		httpReq.Quantity = 30
	}
	if httpReq.OptimizationGoal == "" {
		httpReq.OptimizationGoal = "balanced"
	}
	if httpReq.TransactionID == "" {
		httpReq.TransactionID = requestID
	}

	// Call service
	serviceReq := &services.CostAnalysisRequest{
		TransactionID:       httpReq.TransactionID,
		DrugRxNorms:        httpReq.DrugRxNorms,
		PayerID:            httpReq.PayerID,
		PlanID:             httpReq.PlanID,
		Quantity:           httpReq.Quantity,
		IncludeAlternatives: httpReq.IncludeAlternatives,
		OptimizationGoal:   httpReq.OptimizationGoal,
	}

	analysis, err := fh.formularyService.AnalyzeCosts(r.Context(), serviceReq)
	if err != nil {
		log.Printf("Error analyzing costs for request %s: %v", requestID, err)
		sendAPIResponse(w, false, nil, "Cost analysis failed", requestID)
		return
	}

	sendAPIResponse(w, true, analysis, "", requestID)
}

// OptimizeCosts handles cost optimization requests for existing drug regimens
func (fh *FormularyHandler) OptimizeCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := generateRequestID()
	
	// Parse request body
	var httpReq struct {
		TransactionID   string `json:"transaction_id"`
		DrugRxNorms    []string `json:"drug_rxnorms"`
		PayerID        string `json:"payer_id"`
		PlanID         string `json:"plan_id"`
		OptimizationGoal string `json:"optimization_goal"`
		MaxComplexity  string `json:"max_complexity"` // simple, moderate, complex
		MinSavings     float64 `json:"min_savings"`   // Minimum savings threshold
	}
	
	if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
		sendAPIResponse(w, false, nil, "Invalid request body", requestID)
		return
	}

	// Validate required fields
	if len(httpReq.DrugRxNorms) == 0 || httpReq.PayerID == "" || httpReq.PlanID == "" {
		sendAPIResponse(w, false, nil, "Missing required fields", requestID)
		return
	}

	// Set defaults
	if httpReq.OptimizationGoal == "" {
		httpReq.OptimizationGoal = "cost"
	}
	if httpReq.MaxComplexity == "" {
		httpReq.MaxComplexity = "moderate"
	}
	if httpReq.MinSavings <= 0 {
		httpReq.MinSavings = 5.0 // $5 minimum threshold
	}

	// Call cost analysis with optimization focus
	serviceReq := &services.CostAnalysisRequest{
		TransactionID:       httpReq.TransactionID,
		DrugRxNorms:        httpReq.DrugRxNorms,
		PayerID:            httpReq.PayerID,
		PlanID:             httpReq.PlanID,
		Quantity:           30, // Standard quantity for optimization
		IncludeAlternatives: true, // Always include alternatives for optimization
		OptimizationGoal:   httpReq.OptimizationGoal,
	}

	analysis, err := fh.formularyService.AnalyzeCosts(r.Context(), serviceReq)
	if err != nil {
		log.Printf("Error optimizing costs for request %s: %v", requestID, err)
		sendAPIResponse(w, false, nil, "Cost optimization failed", requestID)
		return
	}

	// Filter recommendations based on complexity and savings thresholds
	filteredResponse := fh.filterOptimizationRecommendations(analysis, httpReq.MaxComplexity, httpReq.MinSavings)

	sendAPIResponse(w, true, filteredResponse, "", requestID)
}

// AnalyzePortfolioCosts handles portfolio-level cost analysis
func (fh *FormularyHandler) AnalyzePortfolioCosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := generateRequestID()
	
	// Parse request body for portfolio analysis
	var httpReq struct {
		TransactionID    string   `json:"transaction_id"`
		DrugPortfolio   []PortfolioDrug `json:"drug_portfolio"`
		PayerID         string   `json:"payer_id"`
		PlanID          string   `json:"plan_id"`
		AnalysisType    string   `json:"analysis_type"` // monthly, quarterly, annual
		OptimizationGoal string  `json:"optimization_goal"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&httpReq); err != nil {
		sendAPIResponse(w, false, nil, "Invalid request body", requestID)
		return
	}

	// Validate portfolio
	if len(httpReq.DrugPortfolio) == 0 {
		sendAPIResponse(w, false, nil, "Empty drug portfolio", requestID)
		return
	}

	// Extract drug RxNorms and calculate weighted quantities
	drugRxNorms := make([]string, 0, len(httpReq.DrugPortfolio))
	totalQuantity := 0
	
	for _, drug := range httpReq.DrugPortfolio {
		drugRxNorms = append(drugRxNorms, drug.RxNorm)
		totalQuantity += drug.MonthlyQuantity
	}

	// Perform portfolio analysis
	serviceReq := &services.CostAnalysisRequest{
		TransactionID:       httpReq.TransactionID,
		DrugRxNorms:        drugRxNorms,
		PayerID:            httpReq.PayerID,
		PlanID:             httpReq.PlanID,
		Quantity:           totalQuantity / len(drugRxNorms), // Average quantity
		IncludeAlternatives: true,
		OptimizationGoal:   httpReq.OptimizationGoal,
	}

	analysis, err := fh.formularyService.AnalyzeCosts(r.Context(), serviceReq)
	if err != nil {
		log.Printf("Error analyzing portfolio costs for request %s: %v", requestID, err)
		sendAPIResponse(w, false, nil, "Portfolio cost analysis failed", requestID)
		return
	}

	// Enhance with portfolio-specific insights
	portfolioInsights := fh.generatePortfolioInsights(analysis, httpReq.DrugPortfolio, httpReq.AnalysisType)

	response := map[string]interface{}{
		"cost_analysis":      analysis,
		"portfolio_insights": portfolioInsights,
		"analysis_type":     httpReq.AnalysisType,
		"total_drugs":       len(httpReq.DrugPortfolio),
	}

	sendAPIResponse(w, true, response, "", requestID)
}

// Supporting types and helper methods

// PortfolioDrug represents a drug in a patient's medication portfolio
type PortfolioDrug struct {
	RxNorm           string  `json:"rxnorm"`
	DrugName         string  `json:"drug_name"`
	MonthlyQuantity  int     `json:"monthly_quantity"`
	CurrentCost      float64 `json:"current_cost"`
	Frequency        string  `json:"frequency"` // daily, twice_daily, weekly, etc.
	TherapeuticClass string  `json:"therapeutic_class"`
}

// PortfolioInsights provides portfolio-level cost optimization insights
type PortfolioInsights struct {
	TotalMonthlySpend     float64                    `json:"total_monthly_spend"`
	OptimizedMonthlySpend float64                    `json:"optimized_monthly_spend"`
	AnnualSavingsPotential float64                   `json:"annual_savings_potential"`
	HighImpactOptimizations []HighImpactOptimization `json:"high_impact_optimizations"`
	RiskAssessment        RiskAssessment           `json:"risk_assessment"`
	ImplementationPlan    ImplementationPlan        `json:"implementation_plan"`
}

// HighImpactOptimization represents high-value optimization opportunities
type HighImpactOptimization struct {
	DrugName            string  `json:"drug_name"`
	CurrentMonthlyCost  float64 `json:"current_monthly_cost"`
	OptimizedMonthlyCost float64 `json:"optimized_monthly_cost"`
	MonthlySavings      float64 `json:"monthly_savings"`
	AnnualSavings       float64 `json:"annual_savings"`
	RecommendedAction   string  `json:"recommended_action"`
	ImplementationEase  string  `json:"implementation_ease"`
	ClinicalRisk        string  `json:"clinical_risk"`
}

// RiskAssessment provides portfolio optimization risk analysis
type RiskAssessment struct {
	OverallRiskLevel    string             `json:"overall_risk_level"`
	ComplexityFactors   []string          `json:"complexity_factors"`
	ClinicalConsiderations []string       `json:"clinical_considerations"`
	ImplementationRisks []string          `json:"implementation_risks"`
	RiskMitigationSteps []string          `json:"risk_mitigation_steps"`
}

// ImplementationPlan provides structured optimization rollout plan
type ImplementationPlan struct {
	Phase1Actions       []string `json:"phase1_actions"`        // Immediate, low-risk changes
	Phase2Actions       []string `json:"phase2_actions"`        // Moderate-risk changes
	Phase3Actions       []string `json:"phase3_actions"`        // Complex changes requiring monitoring
	EstimatedTimeline   string   `json:"estimated_timeline"`    // e.g., "6-12 weeks"
	MonitoringRequired  []string `json:"monitoring_required"`
	SuccessMetrics      []string `json:"success_metrics"`
}

// filterOptimizationRecommendations filters recommendations based on criteria
func (fh *FormularyHandler) filterOptimizationRecommendations(analysis *services.CostAnalysisResponse, maxComplexity string, minSavings float64) *services.CostAnalysisResponse {
	// Create filtered copy
	filtered := *analysis
	filtered.Recommendations = make([]services.CostOptimization, 0)

	for _, rec := range analysis.Recommendations {
		// Filter by complexity
		if !fh.isComplexityAcceptable(rec.ImplementationComplexity, maxComplexity) {
			continue
		}
		
		// Filter by minimum savings
		if rec.EstimatedSavings < minSavings {
			continue
		}
		
		filtered.Recommendations = append(filtered.Recommendations, rec)
	}

	// Recalculate totals based on filtered recommendations
	fh.recalculateFilteredTotals(&filtered)

	return &filtered
}

// isComplexityAcceptable checks if complexity meets the threshold
func (fh *FormularyHandler) isComplexityAcceptable(complexity, maxComplexity string) bool {
	complexityOrder := map[string]int{
		"simple":   1,
		"moderate": 2,
		"complex":  3,
	}
	
	currentLevel := complexityOrder[complexity]
	maxLevel := complexityOrder[maxComplexity]
	
	return currentLevel <= maxLevel
}

// recalculateFilteredTotals recalculates totals after filtering
func (fh *FormularyHandler) recalculateFilteredTotals(response *services.CostAnalysisResponse) {
	totalSavings := 0.0
	
	for _, rec := range response.Recommendations {
		totalSavings += rec.EstimatedSavings
	}
	
	response.TotalSavings = totalSavings
	response.TotalAlternativeCost = response.TotalPrimaryCost - totalSavings
	
	if response.TotalPrimaryCost > 0 {
		response.SavingsPercent = (totalSavings / response.TotalPrimaryCost) * 100.0
	}
}

// generatePortfolioInsights creates portfolio-level optimization insights
func (fh *FormularyHandler) generatePortfolioInsights(analysis *services.CostAnalysisResponse, portfolio []PortfolioDrug, analysisType string) PortfolioInsights {
	insights := PortfolioInsights{
		TotalMonthlySpend:       analysis.TotalPrimaryCost,
		OptimizedMonthlySpend:   analysis.TotalAlternativeCost,
		AnnualSavingsPotential:  analysis.TotalSavings * 12, // Annualize monthly savings
		HighImpactOptimizations: make([]HighImpactOptimization, 0),
		RiskAssessment:         fh.assessPortfolioRisk(analysis),
		ImplementationPlan:     fh.createImplementationPlan(analysis),
	}

	// Identify high-impact optimization opportunities
	for i, drugAnalysis := range analysis.DrugAnalysis {
		if drugAnalysis.BestAlternative != nil && drugAnalysis.PotentialSavings > 10.0 {
			var drugName string
			if i < len(portfolio) {
				drugName = portfolio[i].DrugName
			} else {
				drugName = drugAnalysis.DrugName
			}

			optimization := HighImpactOptimization{
				DrugName:            drugName,
				CurrentMonthlyCost:  drugAnalysis.PrimaryCost,
				OptimizedMonthlyCost: drugAnalysis.BestAlternative.EstimatedCost,
				MonthlySavings:      drugAnalysis.PotentialSavings,
				AnnualSavings:       drugAnalysis.PotentialSavings * 12,
				RecommendedAction:   fh.generateRecommendedAction(drugAnalysis.BestAlternative),
				ImplementationEase:  drugAnalysis.BestAlternative.SwitchComplexity,
				ClinicalRisk:        fh.assessClinicalRisk(drugAnalysis.BestAlternative),
			}
			
			insights.HighImpactOptimizations = append(insights.HighImpactOptimizations, optimization)
		}
	}

	return insights
}

// assessPortfolioRisk assesses overall portfolio optimization risk
func (fh *FormularyHandler) assessPortfolioRisk(analysis *services.CostAnalysisResponse) RiskAssessment {
	complexCount := 0
	moderateCount := 0
	
	for _, drugAnalysis := range analysis.DrugAnalysis {
		if drugAnalysis.BestAlternative != nil {
			switch drugAnalysis.BestAlternative.SwitchComplexity {
			case "complex":
				complexCount++
			case "moderate":
				moderateCount++
			}
		}
	}

	riskLevel := "low"
	if complexCount >= 3 || (complexCount >= 1 && moderateCount >= 3) {
		riskLevel = "high"
	} else if complexCount >= 1 || moderateCount >= 2 {
		riskLevel = "medium"
	}

	return RiskAssessment{
		OverallRiskLevel: riskLevel,
		ComplexityFactors: []string{
			fmt.Sprintf("%d complex medication switches", complexCount),
			fmt.Sprintf("%d moderate complexity switches", moderateCount),
		},
		ClinicalConsiderations: []string{
			"Patient monitoring required for therapeutic switches",
			"Efficacy assessment needed within 2-4 weeks",
			"Adverse reaction monitoring for new medications",
		},
		ImplementationRisks: []string{
			"Patient adherence challenges during transition",
			"Provider workflow disruption",
			"Potential gaps in therapy coverage",
		},
		RiskMitigationSteps: []string{
			"Staggered implementation over 2-3 months",
			"Enhanced patient education and counseling",
			"Close monitoring with follow-up appointments",
			"Provider training on new alternatives",
		},
	}
}

// createImplementationPlan creates a structured rollout plan
func (fh *FormularyHandler) createImplementationPlan(analysis *services.CostAnalysisResponse) ImplementationPlan {
	return ImplementationPlan{
		Phase1Actions: []string{
			"Implement simple generic substitutions",
			"Update preferred drug lists",
			"Notify prescribers of tier 1 alternatives",
		},
		Phase2Actions: []string{
			"Implement moderate complexity therapeutic switches",
			"Update clinical protocols",
			"Provider education on new alternatives",
		},
		Phase3Actions: []string{
			"Implement complex therapeutic switches with monitoring",
			"Specialty medication optimization",
			"Advanced prior authorization strategies",
		},
		EstimatedTimeline: "8-12 weeks for full implementation",
		MonitoringRequired: []string{
			"Patient adherence rates",
			"Clinical outcome measures",
			"Adverse event monitoring",
			"Cost savings realization",
		},
		SuccessMetrics: []string{
			"Cost reduction percentage achieved",
			"Patient satisfaction scores",
			"Clinical outcome maintenance",
			"Provider satisfaction with alternatives",
		},
	}
}

// generateRecommendedAction creates actionable recommendation text
func (fh *FormularyHandler) generateRecommendedAction(alternative *services.Alternative) string {
	switch alternative.AlternativeType {
	case "generic":
		return fmt.Sprintf("Switch to generic equivalent: %s", alternative.DrugName)
	case "therapeutic":
		return fmt.Sprintf("Consider therapeutic alternative: %s (requires clinical review)", alternative.DrugName)
	case "biosimilar":
		return fmt.Sprintf("Switch to biosimilar: %s (requires specialty consultation)", alternative.DrugName)
	case "tier_optimized":
		return fmt.Sprintf("Switch to preferred formulary option: %s", alternative.DrugName)
	default:
		return fmt.Sprintf("Consider alternative: %s", alternative.DrugName)
	}
}

// assessClinicalRisk assesses clinical risk of switching
func (fh *FormularyHandler) assessClinicalRisk(alternative *services.Alternative) string {
	if alternative.EfficacyRating >= 0.95 && alternative.SafetyProfile == "excellent" {
		return "minimal"
	} else if alternative.EfficacyRating >= 0.85 && (alternative.SafetyProfile == "good" || alternative.SafetyProfile == "excellent") {
		return "low"
	} else if alternative.EfficacyRating >= 0.75 {
		return "moderate"
	}
	return "high"
}

// sendAPIResponse sends a standardized API response
func sendAPIResponse(w http.ResponseWriter, success bool, data interface{}, errorMsg string, requestID string) {
	response := APIResponse{
		Success:   success,
		Data:      data,
		Error:     errorMsg,
		RequestID: requestID,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	
	status := http.StatusOK
	if !success {
		status = http.StatusInternalServerError
	}
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(response)
}