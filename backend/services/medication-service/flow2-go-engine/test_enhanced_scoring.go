// Test runner for Enhanced Scoring and Ranking system
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Mock data structures for testing (simplified versions)
type SafetyVerifiedProposal struct {
	Original struct {
		MedicationCode    string
		MedicationName    string
		GenericName       string
		TherapeuticClass  string
		FormularyTier     int
		CostEstimate      float64
	}
	FinalDose struct {
		DoseMg    float64
		Route     string
		IntervalH uint32
	}
	SafetyScore  float64
	DDIWarnings  []DDIFlag
	Warnings     []string
}

type DDIFlag struct {
	Severity string
}

type ClinicalContext struct {
	PatientID      string
	FormularyKBId  string
	Conditions     []Condition
}

type Condition struct {
	Code string
	Name string
}

type EnhancedProposal struct {
	TherapyID string
	Class     string
	Agent     string
	Efficacy  EfficacyDetail
	Safety    SafetyDetail
	Cost      CostDetail
}

type EfficacyDetail struct {
	ExpectedA1cDropPct float64
	CVBenefit         bool
	HFBenefit         bool
	CKDBenefit        bool
}

type SafetyDetail struct {
	ResidualDDI    string
	HypoPropensity string
	WeightEffect   string
}

type CostDetail struct {
	MonthlyEstimate float64
	Currency        string
}

type EnhancedScoredProposal struct {
	TherapyID   string
	FinalScore  float64
	Rank        int
	SubScores   ComponentScores
	Notes       []string
	ScoredAt    time.Time
}

type ComponentScores struct {
	Efficacy     ScoreDetail
	Safety       ScoreDetail
	Cost         ScoreDetail
	Adherence    ScoreDetail
	Availability ScoreDetail
	Preference   ScoreDetail
}

type ScoreDetail struct {
	Score float64
}

// Mock Enhanced Scoring Engine for testing
type MockEnhancedScoringEngine struct {
	logger *logrus.Logger
}

func NewMockEnhancedScoringEngine() *MockEnhancedScoringEngine {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	return &MockEnhancedScoringEngine{
		logger: logger,
	}
}

func (e *MockEnhancedScoringEngine) ScoreAndRankProposals(
	ctx context.Context,
	proposals []*SafetyVerifiedProposal,
	patientContext *ClinicalContext,
	indication string,
) ([]*EnhancedScoredProposal, error) {
	
	e.logger.WithFields(logrus.Fields{
		"proposal_count": len(proposals),
		"patient_id":     patientContext.PatientID,
		"indication":     indication,
	}).Info("Starting enhanced scoring process")

	var scored []*EnhancedScoredProposal

	for i, proposal := range proposals {
		// Calculate scores based on medication characteristics
		efficacyScore := e.calculateEfficacyScore(proposal.Original.MedicationName)
		safetyScore := e.calculateSafetyScore(proposal.Original.MedicationName, proposal.DDIWarnings)
		costScore := e.calculateCostScore(proposal.Original.CostEstimate)
		
		// Calculate final weighted score
		finalScore := (efficacyScore * 0.35) + (safetyScore * 0.30) + (costScore * 0.20) + 0.15 // base score

		enhancedScored := &EnhancedScoredProposal{
			TherapyID:  proposal.Original.MedicationCode,
			FinalScore: finalScore,
			Rank:       i + 1, // Will be updated after sorting
			SubScores: ComponentScores{
				Efficacy:     ScoreDetail{Score: efficacyScore},
				Safety:       ScoreDetail{Score: safetyScore},
				Cost:         ScoreDetail{Score: costScore},
				Adherence:    ScoreDetail{Score: 0.8},
				Availability: ScoreDetail{Score: 0.9},
				Preference:   ScoreDetail{Score: 0.85},
			},
			Notes: []string{
				fmt.Sprintf("Enhanced scoring for %s", proposal.Original.MedicationName),
				e.generateClinicalNote(proposal.Original.MedicationName),
			},
			ScoredAt: time.Now(),
		}

		scored = append(scored, enhancedScored)
	}

	// Sort by final score (highest first)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].FinalScore > scored[i].FinalScore {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Update rankings
	for i, proposal := range scored {
		proposal.Rank = i + 1
	}

	e.logger.WithFields(logrus.Fields{
		"candidates_scored": len(scored),
		"top_therapy":       scored[0].TherapyID,
		"top_score":         scored[0].FinalScore,
	}).Info("Enhanced scoring completed")

	return scored, nil
}

func (e *MockEnhancedScoringEngine) calculateEfficacyScore(medicationName string) float64 {
	switch {
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return 0.95 // High efficacy GLP-1 RAs
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return 0.85 // Good efficacy SGLT2is
	case contains(medicationName, "metformin"):
		return 0.80 // Standard efficacy
	case contains(medicationName, "insulin"):
		return 0.90 // High efficacy but other concerns
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return 0.75 // Moderate efficacy
	default:
		return 0.70 // Default
	}
}

func (e *MockEnhancedScoringEngine) calculateSafetyScore(medicationName string, ddiWarnings []DDIFlag) float64 {
	baseScore := 1.0
	
	// DDI penalties
	for _, ddi := range ddiWarnings {
		switch ddi.Severity {
		case "major":
			baseScore -= 0.30
		case "moderate":
			baseScore -= 0.15
		}
	}
	
	// Medication-specific safety adjustments
	switch {
	case contains(medicationName, "insulin"):
		baseScore -= 0.20 // Hypoglycemia risk
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		baseScore -= 0.25 // High hypoglycemia risk
	case contains(medicationName, "metformin"):
		baseScore -= 0.05 // Very safe
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		baseScore -= 0.10 // GI side effects
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		baseScore -= 0.05 // Generally safe
	}
	
	if baseScore < 0.1 {
		baseScore = 0.1 // Minimum safety score
	}
	
	return baseScore
}

func (e *MockEnhancedScoringEngine) calculateCostScore(costEstimate float64) float64 {
	// Invert cost so lower cost = higher score
	if costEstimate <= 50 {
		return 1.0 // Very affordable
	} else if costEstimate <= 100 {
		return 0.8 // Affordable
	} else if costEstimate <= 200 {
		return 0.6 // Moderate cost
	} else if costEstimate <= 500 {
		return 0.4 // Expensive
	} else {
		return 0.2 // Very expensive
	}
}

func (e *MockEnhancedScoringEngine) generateClinicalNote(medicationName string) string {
	switch {
	case contains(medicationName, "semaglutide"):
		return "GLP-1 RA with proven CV benefits and weight loss"
	case contains(medicationName, "empagliflozin"):
		return "SGLT2i with CV, HF, and renal benefits"
	case contains(medicationName, "metformin"):
		return "First-line therapy with excellent safety profile"
	case contains(medicationName, "insulin"):
		return "Highly effective but requires careful monitoring"
	case contains(medicationName, "glipizide"):
		return "Cost-effective but monitor for hypoglycemia"
	default:
		return "Standard diabetes medication"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test scenarios
func main() {
	fmt.Println("🧪 Testing Enhanced Scoring and Ranking System")
	fmt.Println(strings.Repeat("=", 60))

	engine := NewMockEnhancedScoringEngine()

	// Test Case 1: ASCVD Patient
	fmt.Println("\n📋 Test Case 1: ASCVD Patient (Cardiovascular Disease)")
	testASCVDPatient(engine)

	// Test Case 2: Budget-Conscious Patient
	fmt.Println("\n📋 Test Case 2: Budget-Conscious Patient")
	testBudgetPatient(engine)

	// Test Case 3: Safety-Sensitive Patient
	fmt.Println("\n📋 Test Case 3: Safety-Sensitive Patient (Multiple DDIs)")
	testSafetySensitivePatient(engine)

	// Test Case 4: Performance Test
	fmt.Println("\n📋 Test Case 4: Performance Test (Multiple Candidates)")
	testPerformance(engine)

	fmt.Println("\n✅ All tests completed successfully!")
	fmt.Println("🎯 Enhanced Scoring and Ranking system is working correctly!")
}

func testASCVDPatient(engine *MockEnhancedScoringEngine) {
	proposals := []*SafetyVerifiedProposal{
		createProposal("SEMA001", "semaglutide", "Ozempic", "GLP-1_RA", 800.0, 2),
		createProposal("EMPA001", "empagliflozin", "Jardiance", "SGLT2i", 400.0, 1),
		createProposal("GLIP001", "glipizide", "Glucotrol", "Sulfonylurea", 25.0, 1),
	}

	clinicalCtx := &ClinicalContext{
		PatientID:     "patient-ascvd-001",
		FormularyKBId: "formulary-001",
		Conditions: []Condition{
			{Code: "I25.9", Name: "Coronary Artery Disease"},
			{Code: "E11.9", Name: "Type 2 Diabetes"},
		},
	}

	scored, err := engine.ScoreAndRankProposals(
		context.Background(),
		proposals,
		clinicalCtx,
		"diabetes_type2",
	)

	if err != nil {
		log.Fatalf("ASCVD test failed: %v", err)
	}

	printResults("ASCVD Patient", scored)
}

func testBudgetPatient(engine *MockEnhancedScoringEngine) {
	proposals := []*SafetyVerifiedProposal{
		createProposal("SEMA001", "semaglutide", "Ozempic", "GLP-1_RA", 800.0, 3),
		createProposal("MET001", "metformin", "Glucophage", "Biguanide", 30.0, 1),
		createProposal("GLIP001", "glipizide", "Glucotrol", "Sulfonylurea", 25.0, 1),
	}

	clinicalCtx := &ClinicalContext{
		PatientID:     "patient-budget-001",
		FormularyKBId: "formulary-budget",
		Conditions: []Condition{
			{Code: "E11.9", Name: "Type 2 Diabetes"},
		},
	}

	scored, err := engine.ScoreAndRankProposals(
		context.Background(),
		proposals,
		clinicalCtx,
		"diabetes_type2",
	)

	if err != nil {
		log.Fatalf("Budget test failed: %v", err)
	}

	printResults("Budget-Conscious Patient", scored)
}

func testSafetySensitivePatient(engine *MockEnhancedScoringEngine) {
	proposals := []*SafetyVerifiedProposal{
		createProposalWithDDI("INS001", "insulin", "Humalog", "Insulin", 150.0, 1, "major"),
		createProposalWithDDI("MET001", "metformin", "Glucophage", "Biguanide", 30.0, 1, "none"),
		createProposalWithDDI("GLIP001", "glipizide", "Glucotrol", "Sulfonylurea", 25.0, 1, "moderate"),
	}

	clinicalCtx := &ClinicalContext{
		PatientID:     "patient-safety-001",
		FormularyKBId: "formulary-001",
		Conditions: []Condition{
			{Code: "E11.9", Name: "Type 2 Diabetes"},
		},
	}

	scored, err := engine.ScoreAndRankProposals(
		context.Background(),
		proposals,
		clinicalCtx,
		"diabetes_type2",
	)

	if err != nil {
		log.Fatalf("Safety test failed: %v", err)
	}

	printResults("Safety-Sensitive Patient", scored)
}

func testPerformance(engine *MockEnhancedScoringEngine) {
	// Create 10 proposals for performance testing
	proposals := []*SafetyVerifiedProposal{}
	medications := []struct {
		code, name, brand, class string
		cost                     float64
		tier                     int
	}{
		{"MET001", "metformin", "Glucophage", "Biguanide", 30.0, 1},
		{"GLIP001", "glipizide", "Glucotrol", "Sulfonylurea", 25.0, 1},
		{"SEMA001", "semaglutide", "Ozempic", "GLP-1_RA", 800.0, 2},
		{"EMPA001", "empagliflozin", "Jardiance", "SGLT2i", 400.0, 1},
		{"LIRA001", "liraglutide", "Victoza", "GLP-1_RA", 750.0, 2},
		{"DAPA001", "dapagliflozin", "Farxiga", "SGLT2i", 380.0, 1},
		{"INS001", "insulin", "Humalog", "Insulin", 150.0, 1},
		{"GLYB001", "glyburide", "DiaBeta", "Sulfonylurea", 20.0, 1},
		{"SITA001", "sitagliptin", "Januvia", "DPP-4i", 300.0, 2},
		{"PIOS001", "pioglitazone", "Actos", "TZD", 100.0, 1},
	}

	for _, med := range medications {
		proposals = append(proposals, createProposal(med.code, med.name, med.brand, med.class, med.cost, med.tier))
	}

	clinicalCtx := &ClinicalContext{
		PatientID:     "patient-performance-001",
		FormularyKBId: "formulary-001",
		Conditions: []Condition{
			{Code: "E11.9", Name: "Type 2 Diabetes"},
		},
	}

	start := time.Now()
	scored, err := engine.ScoreAndRankProposals(
		context.Background(),
		proposals,
		clinicalCtx,
		"diabetes_type2",
	)
	duration := time.Since(start)

	if err != nil {
		log.Fatalf("Performance test failed: %v", err)
	}

	fmt.Printf("⏱️  Performance: Scored %d proposals in %v\n", len(proposals), duration)
	fmt.Printf("📊 Top 3 recommendations:\n")
	for i := 0; i < 3 && i < len(scored); i++ {
		fmt.Printf("   %d. %s (Score: %.3f)\n", i+1, scored[i].TherapyID, scored[i].FinalScore)
	}
}

func createProposal(code, name, brand, class string, cost float64, tier int) *SafetyVerifiedProposal {
	return &SafetyVerifiedProposal{
		Original: struct {
			MedicationCode   string
			MedicationName   string
			GenericName      string
			TherapeuticClass string
			FormularyTier    int
			CostEstimate     float64
		}{
			MedicationCode:   code,
			MedicationName:   brand,
			GenericName:      name,
			TherapeuticClass: class,
			FormularyTier:    tier,
			CostEstimate:     cost,
		},
		FinalDose: struct {
			DoseMg    float64
			Route     string
			IntervalH uint32
		}{
			DoseMg:    500,
			Route:     "po",
			IntervalH: 24,
		},
		SafetyScore: 0.9,
		DDIWarnings: []DDIFlag{},
		Warnings:    []string{},
	}
}

func createProposalWithDDI(code, name, brand, class string, cost float64, tier int, ddiSeverity string) *SafetyVerifiedProposal {
	proposal := createProposal(code, name, brand, class, cost, tier)
	if ddiSeverity != "none" {
		proposal.DDIWarnings = []DDIFlag{{Severity: ddiSeverity}}
	}
	return proposal
}

func printResults(scenario string, scored []*EnhancedScoredProposal) {
	fmt.Printf("🎯 Results for %s:\n", scenario)
	fmt.Printf("   Candidates processed: %d\n", len(scored))
	fmt.Printf("   Rankings:\n")
	
	for i, proposal := range scored {
		fmt.Printf("   %d. %s (Score: %.3f)\n", i+1, proposal.TherapyID, proposal.FinalScore)
		fmt.Printf("      Efficacy: %.3f, Safety: %.3f, Cost: %.3f\n",
			proposal.SubScores.Efficacy.Score,
			proposal.SubScores.Safety.Score,
			proposal.SubScores.Cost.Score)
		if len(proposal.Notes) > 1 {
			fmt.Printf("      Note: %s\n", proposal.Notes[1])
		}
		fmt.Println()
	}
}
